package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/gorilla/websocket"
)

type chatUpdate struct {
	Type      string                 `json:"type"`
	Enabled   *bool                  `json:"enabled,omitempty"`
	Message   *database.ChatMessage  `json:"message,omitempty"`
	MessageID int64                  `json:"message_id,omitempty"`
	Reaction  *database.ChatReaction `json:"reaction,omitempty"`
}

type chatHub struct {
	mu          sync.Mutex
	rooms       map[int]map[chan []byte]bool
	connections int
}

func (h *chatHub) subscribe(eventID int, preview bool) (<-chan []byte, func(), bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.connections >= 5000 || len(h.rooms[eventID]) >= 2000 {
		return nil, nil, false
	}
	if h.rooms == nil {
		h.rooms = make(map[int]map[chan []byte]bool)
	}
	if h.rooms[eventID] == nil {
		h.rooms[eventID] = make(map[chan []byte]bool)
	}
	ch := make(chan []byte, 32)
	h.rooms[eventID][ch] = preview
	h.connections++
	return ch, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		h.connections--
		delete(h.rooms[eventID], ch)
		if len(h.rooms[eventID]) == 0 {
			delete(h.rooms, eventID)
		}
	}, true
}

func (h *chatHub) publish(eventID int, update chatUpdate, closeRoom bool) {
	payload, err := json.Marshal(update)
	if err != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch, preview := range h.rooms[eventID] {
		if update.Type == "preview" && !preview {
			continue
		}
		sent := false
		select {
		case ch <- payload:
			sent = true
		default:
		}
		if !sent || closeRoom {
			close(ch)
			delete(h.rooms[eventID], ch)
		}
	}
	if len(h.rooms[eventID]) == 0 {
		delete(h.rooms, eventID)
	}
}

func (c *matchChat) publishStatus(eventID int, enabled, deleted bool) {
	update := chatUpdate{Type: "status", Enabled: &enabled}
	c.status.publish(eventID, update, deleted)
	c.messages.publish(eventID, update, !enabled)
}

func (c *matchChat) publishMessageUpdate(eventID int, update chatUpdate) {
	c.messages.publish(eventID, update, false)
	if update.Type == "message" {
		c.publishPreview(eventID, update.Message)
	}
}

func (c *matchChat) publishPreview(eventID int, message *database.ChatMessage) {
	c.status.publish(eventID, chatUpdate{Type: "preview", Message: message}, false)
}

func (c *matchChat) deleteEvent(eventID int) {
	lock := &c.writes[eventID%len(c.writes)]
	lock.Lock()
	defer lock.Unlock()
	c.publishStatus(eventID, false, true)
}

func (rt *_router) matchChatStatusStream(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	rt.serveMatchChatStream(w, r, ctx, true)
}

func (rt *_router) matchChatStream(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	rt.serveMatchChatStream(w, r, ctx, false)
}

func (rt *_router) serveMatchChatStream(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext, statusOnly bool) {
	origin, err := url.Parse(r.Header.Get("Origin"))
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	if err != nil || origin.Scheme != scheme || !strings.EqualFold(origin.Host, r.Host) {
		matchChatError(w, http.StatusForbidden, "invalid_origin")
		return
	}
	store, ok := rt.matchChatScope(w, r, ctx)
	if !ok {
		return
	}
	hub := &rt.chat.messages
	preview := statusOnly && r.URL.Query().Get("preview") == "1"
	if statusOnly {
		hub = &rt.chat.status
	}
	lock := &rt.chat.writes[store.EventID%len(rt.chat.writes)]
	lock.Lock()
	enabled, err := store.Enabled(r.Context())
	initial := chatUpdate{Type: "status", Enabled: &enabled}
	if err == nil && enabled && preview {
		initial.Message, err = store.Preview(r.Context())
	}
	if err == nil && !statusOnly && !enabled {
		err = database.ErrChatDisabled
	}
	if err != nil {
		lock.Unlock()
		matchChatStoreError(w, ctx, err)
		return
	}
	updates, unsubscribe, ok := hub.subscribe(store.EventID, preview)
	lock.Unlock()
	if !ok {
		matchChatError(w, http.StatusServiceUnavailable, "chat_capacity")
		return
	}
	defer unsubscribe()
	conn, err := clubMatchUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	const writeWait = 5 * time.Second
	const pongWait = 60 * time.Second
	conn.SetReadLimit(512)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	disconnected := make(chan struct{})
	go func() {
		// Clients use the existing authenticated HTTP session for all writes.
		_, _, _ = conn.NextReader()
		close(disconnected)
	}()
	_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := conn.WriteJSON(initial); err != nil {
		return
	}
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-disconnected:
			return
		case payload, ok := <-updates:
			if !ok {
				_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, ""), time.Now().Add(writeWait))
				return
			}
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				return
			}
		case <-ticker.C:
			if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait)); err != nil {
				return
			}
		}
	}
}
