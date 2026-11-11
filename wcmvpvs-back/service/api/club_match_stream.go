package api

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

type clubMatchMessage struct {
	Matches *[]ClubMatch `json:"matches,omitempty"`
	Match   *ClubMatch   `json:"match,omitempty"`
}

type clubMatchHub struct {
	mu      sync.Mutex
	clients map[int64]map[chan []byte]struct{}
}

func (h *clubMatchHub) subscribe(siteID int64) (<-chan []byte, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients == nil {
		h.clients = make(map[int64]map[chan []byte]struct{})
	}
	if h.clients[siteID] == nil {
		h.clients[siteID] = make(map[chan []byte]struct{})
	}
	ch := make(chan []byte, 16)
	h.clients[siteID][ch] = struct{}{}
	return ch, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		delete(h.clients[siteID], ch)
		if len(h.clients[siteID]) == 0 {
			delete(h.clients, siteID)
		}
	}
}

func (h *clubMatchHub) publish(siteID int64, message clubMatchMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.clients[siteID]) == 0 {
		return
	}
	payload, err := json.Marshal(message)
	if err != nil {
		return
	}
	for ch := range h.clients[siteID] {
		select {
		case ch <- payload:
		default:
			// A lagging client reconnects and receives a fresh snapshot.
			close(ch)
			delete(h.clients[siteID], ch)
		}
	}
}

// A calendar edit can change the set of visible matches. Reconnecting replaces
// the calendar snapshot, including deletions and matches moved between teams.
func (h *clubMatchHub) reset(siteID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.clients[siteID] {
		close(ch)
	}
	delete(h.clients, siteID)
}

var clubMatchUpgrader = websocket.Upgrader{HandshakeTimeout: 5 * time.Second}

func (rt *_router) clubPublicMatchStream(w http.ResponseWriter, r *http.Request) {
	site, err := rt.store.ClubSiteBySlug(r.Context(), chi.URLParam(r, "slug"))
	if err != nil || !site.Active {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "club_not_found"})
		return
	}
	updates, unsubscribe := rt.clubMatches.subscribe(site.ID)
	defer unsubscribe()

	matches, err := rt.store.ListClubMatches(r.Context(), site.ID, 0)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "matches_unavailable"})
		return
	}
	conn, err := clubMatchUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	conn.SetReadLimit(512)
	const pongWait = 60 * time.Second
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	disconnected := make(chan struct{})
	go func() {
		// NextReader handles control frames. Any application frame ends this
		// read-only connection.
		_, _, _ = conn.NextReader()
		close(disconnected)
	}()

	const writeWait = 5 * time.Second
	_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := conn.WriteJSON(clubMatchMessage{Matches: &matches}); err != nil {
		return
	}
	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()
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
		case <-ping.C:
			if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait)); err != nil {
				return
			}
		}
	}
}
