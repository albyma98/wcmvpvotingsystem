package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
)

func registerMatchChatRoutes(rt *_router) {
	fan := "/events/{eventId}/match-chat"
	rt.router.Get(fan+"/status/stream", rt.wrap(rt.matchChatStatusStream))
	rt.router.Get(fan+"/stream", rt.wrap(rt.matchChatStream))
	rt.router.Get(fan+"/messages", rt.wrap(rt.matchChatHistory))
	rt.router.Post(fan+"/messages", rt.wrap(rt.matchChatFanWrite))
	rt.router.Put(fan+"/messages/{messageId}/reaction", rt.wrap(rt.matchChatFanWrite))
	rt.router.Post(fan+"/messages/{messageId}/report", rt.wrap(rt.matchChatFanWrite))

	admin := "/admin" + fan
	rt.router.Get(admin, rt.wrapAdmin(rt.matchChatAdminRead))
	rt.router.Get(admin+"/messages", rt.wrapAdmin(rt.matchChatAdminRead))
	rt.router.Get(admin+"/reports", rt.wrapAdmin(rt.matchChatAdminRead))
	rt.router.Get(admin+"/mutes", rt.wrapAdmin(rt.matchChatAdminRead))
	rt.router.Put(admin+"/settings", rt.wrapAdmin(rt.matchChatAdminWrite))
	rt.router.Post(admin+"/messages", rt.wrapAdmin(rt.matchChatAdminWrite))
	rt.router.Delete(admin+"/messages/{messageId}", rt.wrapAdmin(rt.matchChatAdminWrite))
	rt.router.Put(admin+"/reports/{reportId}", rt.wrapAdmin(rt.matchChatAdminWrite))
	rt.router.Put(admin+"/mutes/{fanId}", rt.wrapAdmin(rt.matchChatAdminWrite))
	rt.router.Delete(admin+"/mutes/{fanId}", rt.wrapAdmin(rt.matchChatAdminWrite))
}

type matchChat struct {
	// The same event lock covers commit, broadcast and subscription handshake.
	writes      [64]sync.Mutex
	status      chatHub
	messages    chatHub
	activityMu  sync.Mutex
	activity    map[int][]time.Time
	nextCleanup time.Time
}

func (c *matchChat) allowActivity(fanID int) bool {
	c.activityMu.Lock()
	defer c.activityMu.Unlock()
	now := time.Now()
	if c.activity == nil {
		c.activity = make(map[int][]time.Time)
	}
	cutoff := now.Add(-time.Minute)
	if now.After(c.nextCleanup) {
		for id, times := range c.activity {
			if len(times) == 0 || !times[len(times)-1].After(cutoff) {
				delete(c.activity, id)
			}
		}
		c.nextCleanup = now.Add(time.Minute)
	}
	times := c.activity[fanID]
	if times == nil && len(c.activity) >= 20000 {
		return false
	}
	for len(times) > 0 && !times[0].After(cutoff) {
		times = times[1:]
	}
	if len(times) >= 24 {
		return false
	}
	c.activity[fanID] = append(times, now)
	return true
}

func matchChatInt(value string, fallback, min, max int) int {
	n, err := strconv.Atoi(value)
	if err != nil || n < min || n > max {
		return fallback
	}
	return n
}

var (
	chatInterval  = time.Duration(matchChatInt(os.Getenv("MATCH_CHAT_MIN_INTERVAL_SECONDS"), 5, 1, 60)) * time.Second
	chatPerMinute = matchChatInt(os.Getenv("MATCH_CHAT_MESSAGES_PER_MINUTE"), 8, 1, 60)
	chatRequestID = regexp.MustCompile("^[A-Za-z0-9_-]{16,80}$")
)

func (rt *_router) matchChatScope(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) (database.MatchChatStore, bool) {
	eventID, err := strconv.Atoi(chi.URLParam(r, "eventId"))
	if err != nil || eventID <= 0 {
		matchChatError(w, 400, "invalid_event")
		return database.MatchChatStore{}, false
	}
	if !rt.ensureEventInOrganization(w, ctx, eventID) {
		return database.MatchChatStore{}, false
	}
	return database.MatchChatStore{DB: rt.db.SQLConn(), EventID: eventID, OrganizationID: ctx.OrganizationID}, true
}

func matchChatError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]string{"error": code})
}

func matchChatStoreError(w http.ResponseWriter, ctx reqcontext.RequestContext, err error) {
	switch {
	case errors.Is(err, database.ErrChatDisabled):
		matchChatError(w, 409, "chat_disabled")
	case errors.Is(err, database.ErrChatMuted):
		matchChatError(w, 403, "chat_muted")
	case errors.Is(err, database.ErrChatRate):
		w.Header().Set("Retry-After", "5")
		matchChatError(w, 429, "chat_rate_limited")
	case errors.Is(err, database.ErrChatDuplicate), errors.Is(err, database.ErrChatRequestConflict):
		matchChatError(w, 409, err.Error())
	case errors.Is(err, sql.ErrNoRows):
		matchChatError(w, 404, "chat_item_not_found")
	default:
		ctx.Logger.WithError(err).Error("match chat operation failed")
		matchChatError(w, 500, "chat_unavailable")
	}
}

func decodeMatchChat(w http.ResponseWriter, r *http.Request, value interface{}) bool {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	d.DisallowUnknownFields()
	if d.Decode(value) != nil || d.Decode(&struct{}{}) != io.EOF {
		matchChatError(w, 400, "invalid_request")
		return false
	}
	return true
}

type matchChatInput struct {
	Text      string `json:"text"`
	RequestID string `json:"request_id"`
}

func readMatchChatMessage(w http.ResponseWriter, r *http.Request) (matchChatInput, bool) {
	var in matchChatInput
	if !decodeMatchChat(w, r, &in) {
		return in, false
	}
	in.Text = strings.TrimSpace(in.Text)
	if !chatRequestID.MatchString(in.RequestID) || !moderateMatchChatText(in.Text) {
		matchChatError(w, 400, "invalid_message")
		return in, false
	}
	return in, true
}

// This is the pre-publication moderation check shared by both send routes.
// Messages are rendered as plain text, including markup and URLs.
func moderateMatchChatText(text string) bool {
	return text != "" && utf8.RuneCountInString(text) <= 180 &&
		strings.IndexFunc(text, func(r rune) bool { return unicode.IsControl(r) && r != '\n' }) < 0
}

func matchChatCursor(w http.ResponseWriter, r *http.Request, key string) (int64, bool) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return 0, true
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		matchChatError(w, 400, "invalid_cursor")
		return 0, false
	}
	return id, true
}

func matchChatID(w http.ResponseWriter, r *http.Request, key string) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, key), 10, 64)
	if err != nil || id <= 0 {
		matchChatError(w, 400, "invalid_id")
		return 0, false
	}
	return id, true
}

func (rt *_router) matchChatHistory(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	w.Header().Set("Cache-Control", "private, no-store")
	store, ok := rt.matchChatScope(w, r, ctx)
	if !ok {
		return
	}
	before, ok := matchChatCursor(w, r, "before")
	if !ok {
		return
	}
	var fanID int
	if token := strings.TrimSpace(r.Header.Get("X-Fan-Session")); token != "" {
		var err error
		fanID, _, err = store.VerifiedFan(r.Context(), token)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			matchChatStoreError(w, ctx, err)
			return
		}
	}
	history, err := store.History(r.Context(), before, fanID, false)
	if err != nil {
		matchChatStoreError(w, ctx, err)
		return
	}
	writeJSON(w, 200, history)
}

func (rt *_router) matchChatFanWrite(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	store, ok := rt.matchChatScope(w, r, ctx)
	if !ok {
		return
	}
	token := strings.TrimSpace(r.Header.Get("X-Fan-Session"))
	if token == "" {
		matchChatError(w, 401, "chat_verification_required")
		return
	}
	fanID, nickname, err := store.VerifiedFan(r.Context(), token)
	if errors.Is(err, sql.ErrNoRows) {
		matchChatError(w, 401, "chat_verification_required")
		return
	}
	if err != nil {
		matchChatStoreError(w, ctx, err)
		return
	}
	now := time.Now()
	var update chatUpdate
	var result interface{}
	var action func(*sql.Tx) error
	c := r.Context()

	switch {
	case strings.HasSuffix(r.URL.Path, "/messages"):
		in, ok := readMatchChatMessage(w, r)
		if !ok {
			return
		}
		action = func(tx *sql.Tx) error {
			message, created, err := store.Send(c, tx, database.ChatSend{FanID: fanID, Nickname: nickname, Text: in.Text,
				RequestID: in.RequestID, Now: time.Now().UnixMilli(), MinInterval: chatInterval, PerMinute: chatPerMinute, DuplicateWindow: 30 * time.Second})
			if created {
				public := message
				public.Mine = false
				update = chatUpdate{Type: "message", Message: &public}
			}
			result = map[string]interface{}{"message": message, "changed": created}
			return err
		}
	case strings.HasSuffix(r.URL.Path, "/reaction"):
		id, ok := matchChatID(w, r, "messageId")
		if !ok {
			return
		}
		var in struct {
			Emoji  string `json:"emoji"`
			Active *bool  `json:"active"`
		}
		if !decodeMatchChat(w, r, &in) {
			return
		}
		switch in.Emoji {
		case "❤️", "🔥", "😂", "👏", "💪":
		default:
			matchChatError(w, 400, "invalid_reaction")
			return
		}
		if in.Active == nil {
			matchChatError(w, 400, "invalid_reaction")
			return
		}
		action = func(tx *sql.Tx) error {
			if err := store.CheckMute(c, tx, fanID, now.UnixMilli()); err != nil {
				return err
			}
			reaction, changed, err := store.React(c, tx, id, fanID, in.Emoji, *in.Active)
			if changed {
				public := reaction
				public.Mine = false
				update = chatUpdate{Type: "reaction", MessageID: id, Reaction: &public}
			}
			result = map[string]interface{}{"reaction": reaction, "changed": changed}
			return err
		}
	default:
		id, ok := matchChatID(w, r, "messageId")
		if !ok {
			return
		}
		var in struct {
			Reason string `json:"reason"`
		}
		if !decodeMatchChat(w, r, &in) {
			return
		}
		switch in.Reason {
		case "abuse", "spam", "other":
		default:
			matchChatError(w, 400, "invalid_report")
			return
		}
		action = func(tx *sql.Tx) error {
			changed, err := store.Report(c, tx, id, fanID, in.Reason, now.UnixMilli())
			result = map[string]bool{"changed": changed}
			return err
		}
	}
	if !rt.chat.allowActivity(fanID) {
		matchChatStoreError(w, ctx, database.ErrChatRate)
		return
	}
	lock := &rt.chat.writes[store.EventID%len(rt.chat.writes)]
	lock.Lock()
	err = store.Write(c, true, func(tx *sql.Tx, _ bool) error { return action(tx) })
	if err == nil && update.Type != "" {
		rt.chat.messages.publish(store.EventID, update, false)
	}
	lock.Unlock()
	if err != nil {
		matchChatStoreError(w, ctx, err)
		return
	}
	writeJSON(w, 200, result)
}

func (rt *_router) matchChatAdminRead(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	w.Header().Set("Cache-Control", "private, no-store")
	store, ok := rt.matchChatScope(w, r, ctx)
	if !ok {
		return
	}
	c := r.Context()
	var result interface{}
	var err error
	switch {
	case strings.HasSuffix(r.URL.Path, "/messages"):
		before, ok := matchChatCursor(w, r, "before")
		if !ok {
			return
		}
		result, err = store.History(c, before, 0, true)
	case strings.HasSuffix(r.URL.Path, "/reports"):
		before, ok := matchChatCursor(w, r, "before")
		if !ok {
			return
		}
		reports, next, e := store.Reports(c, before)
		result, err = map[string]interface{}{"reports": reports, "next_before": next}, e
	case strings.HasSuffix(r.URL.Path, "/mutes"):
		after, ok := matchChatCursor(w, r, "after")
		if !ok {
			return
		}
		mutes, next, e := store.Mutes(c, time.Now().UnixMilli(), int(after))
		result, err = map[string]interface{}{"mutes": mutes, "next_after": next}, e
	default:
		enabled, e := store.Enabled(c)
		err = e
		if err == nil {
			stats, e := store.Stats(c)
			result, err = map[string]interface{}{"enabled": enabled, "stats": stats}, e
		}
	}
	if err != nil {
		matchChatStoreError(w, ctx, err)
		return
	}
	writeJSON(w, 200, result)
}

func (rt *_router) matchChatAdminWrite(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	store, ok := rt.matchChatScope(w, r, ctx)
	if !ok {
		return
	}
	c := r.Context()
	now := time.Now()
	var update chatUpdate
	var result interface{} = map[string]bool{"ok": true}
	var action func(*sql.Tx, bool) error
	requireEnabled := false

	switch {
	case strings.HasSuffix(r.URL.Path, "/settings"):
		var in struct {
			Enabled *bool `json:"enabled"`
		}
		if !decodeMatchChat(w, r, &in) {
			return
		}
		if in.Enabled == nil {
			matchChatError(w, 400, "invalid_settings")
			return
		}
		action = func(tx *sql.Tx, enabled bool) error {
			changed := enabled != *in.Enabled
			if changed {
				update = chatUpdate{Type: "status", Enabled: in.Enabled}
			}
			result = map[string]bool{"enabled": *in.Enabled, "changed": changed}
			return store.SetEnabled(c, tx, *in.Enabled, ctx.AdminID, now.UnixMilli())
		}
	case strings.HasSuffix(r.URL.Path, "/messages"):
		in, ok := readMatchChatMessage(w, r)
		if !ok {
			return
		}
		requireEnabled = true
		action = func(tx *sql.Tx, _ bool) error {
			message, changed, err := store.Send(c, tx, database.ChatSend{AdminID: ctx.AdminID, Text: in.Text, RequestID: in.RequestID, Now: now.UnixMilli()})
			if changed {
				update = chatUpdate{Type: "message", Message: &message}
			}
			result = map[string]interface{}{"message": message, "changed": changed}
			return err
		}
	case chi.URLParam(r, "messageId") != "":
		id, ok := matchChatID(w, r, "messageId")
		if !ok {
			return
		}
		action = func(tx *sql.Tx, _ bool) error {
			changed, err := store.Remove(c, tx, id, now.UnixMilli())
			if changed {
				update = chatUpdate{Type: "removed", MessageID: id}
			}
			return err
		}
	case chi.URLParam(r, "reportId") != "":
		id, ok := matchChatID(w, r, "reportId")
		if !ok {
			return
		}
		action = func(tx *sql.Tx, _ bool) error { return store.Review(c, tx, id) }
	default:
		id, ok := matchChatID(w, r, "fanId")
		if !ok {
			return
		}
		var until int64
		if r.Method == http.MethodPut {
			var in struct {
				Minutes int `json:"minutes"`
			}
			if !decodeMatchChat(w, r, &in) {
				return
			}
			if in.Minutes < 1 || in.Minutes > 1440 {
				matchChatError(w, 400, "invalid_mute_duration")
				return
			}
			until = now.Add(time.Duration(in.Minutes) * time.Minute).UnixMilli()
		}
		action = func(tx *sql.Tx, _ bool) error { return store.SetMute(c, tx, int(id), ctx.AdminID, until) }
	}

	lock := &rt.chat.writes[store.EventID%len(rt.chat.writes)]
	lock.Lock()
	err := store.Write(c, requireEnabled, action)
	if err == nil && update.Type != "" {
		if update.Type == "status" {
			rt.chat.publishStatus(store.EventID, *update.Enabled, false)
		} else {
			rt.chat.messages.publish(store.EventID, update, false)
		}
	}
	lock.Unlock()
	if err != nil {
		matchChatStoreError(w, ctx, err)
		return
	}
	if update.Type == "status" {
		name := "match_chat_disabled"
		if *update.Enabled {
			name = "match_chat_enabled"
		}
		_, err := store.DB.ExecContext(c, `INSERT INTO engagement_report_signals
			(event_id,signal_id,name,session_id,device_key,occurred_at,metadata)
			VALUES(?,lower(hex(randomblob(16))),?,'match-chat-admin','',?,'{}')`,
			store.EventID, name, now.UTC().Format(time.RFC3339Nano))
		if err != nil {
			ctx.Logger.WithError(err).Warn("cannot record match chat setting change")
		}
	}
	writeJSON(w, 200, result)
}
