package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

type matchChatFixture struct {
	rt              *_router
	handler         http.Handler
	server          *httptest.Server
	a, b, other     int
	slug, otherSlug string
	admin           string
}

func newMatchChatFixture(t *testing.T) matchChatFixture {
	t.Helper()
	rt, ctx, a, otherOrg := votingAPIFixture(t)
	db := rt.db.SQLConn()
	other, err := rt.db.GetOrganization(otherOrg)
	if err != nil {
		t.Fatal(err)
	}
	event, err := rt.db.GetEventByID(a)
	if err != nil {
		t.Fatal(err)
	}
	insertEvent := func(org int) int {
		result, err := db.Exec(`INSERT INTO events(organization_id,team1_id,team2_id,start_datetime)
			VALUES(?,?,?,'2026-10-25T18:00:00Z')`, org, event.Team1ID, event.Team2ID)
		if err != nil {
			t.Fatal(err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			t.Fatal(err)
		}
		return int(id)
	}
	adminID, err := rt.db.CreateAdmin(database.Admin{Username: "moderator", PasswordHash: "test", Role: "admin", OrganizationID: ctx.OrganizationID})
	if err != nil {
		t.Fatal(err)
	}
	rt.sessionTimeout = time.Hour
	rt.adminSessions = map[string]adminSession{}
	admin, err := rt.createAdminSession(adminID, "moderator", "admin", ctx.OrganizationID, ctx.OrganizationTeamID, ctx.OrganizationSlug)
	if err != nil {
		t.Fatal(err)
	}
	rt.router = chi.NewRouter()
	rt.router.Get("/events", rt.wrapAdmin(rt.listEvents))
	registerEngagementReportRoutes(rt)
	registerMatchChatRoutes(rt)
	server := httptest.NewServer(rt.router)
	t.Cleanup(server.Close)
	return matchChatFixture{rt: rt, handler: rt.router, server: server, a: a, b: insertEvent(ctx.OrganizationID), other: insertEvent(otherOrg),
		slug: ctx.OrganizationSlug, otherSlug: other.Slug, admin: admin}
}

func chatTestPath(event int, path string) string {
	return fmt.Sprintf("/events/%d/match-chat%s", event, path)
}

func (f matchChatFixture) request(t *testing.T, method, path, slug, token, body string, want int) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("X-Organization-Slug", slug)
	r.Header.Set("X-Device-ID", "device1")
	if token == f.admin {
		r.AddCookie(&http.Cookie{Name: adminCookieName(), Value: token})
	} else if token != "" {
		r.Header.Set("X-Fan-Session", token)
	}
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, r)
	if w.Code != want {
		t.Errorf("%s %s: status %d, want %d (%s)", method, path, w.Code, want, w.Body.String())
	}
	return w
}

func decodeChatTest(t *testing.T, w *httptest.ResponseRecorder, out interface{}) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), out); err != nil {
		t.Fatal(err, w.Body.String())
	}
}

func (f matchChatFixture) socket(t *testing.T, event int, path, slug, origin string, want int) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(f.server.URL, "http") + chatTestPath(event, path) + "?organization_slug=" + slug
	header := http.Header{}
	if origin != "" {
		header.Set("Origin", origin)
	}
	dialer := websocket.Dialer{HandshakeTimeout: 3 * time.Second}
	conn, response, err := dialer.Dial(url, header)
	if want != http.StatusSwitchingProtocols {
		if response == nil || response.StatusCode != want {
			t.Fatalf("socket status %v, want %d (%v)", response, want, err)
		}
		response.Body.Close()
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func readChatTestUpdate(t *testing.T, conn *websocket.Conn, kind string) chatUpdate {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var update chatUpdate
	if err := conn.ReadJSON(&update); err != nil || update.Type != kind {
		t.Fatalf("expected %s, received %+v (%v)", kind, update, err)
	}
	return update
}

func (f matchChatFixture) enable(t *testing.T, event int, enabled bool) {
	t.Helper()
	f.request(t, "PUT", "/admin"+chatTestPath(event, "/settings"), f.slug, f.admin, fmt.Sprintf(`{"enabled":%t}`, enabled), 200)
}

func TestMatchChatRealtimeIsolationAndToggle(t *testing.T) {
	f := newMatchChatFixture(t)
	var events []database.Event
	decodeChatTest(t, f.request(t, "GET", "/events", f.slug, f.admin, "", 200), &events)
	if len(events) != 2 {
		t.Fatalf("admin event list with optional locations: %+v", events)
	}
	f.socket(t, f.a, "/stream", f.slug, f.server.URL, 409)
	f.socket(t, f.a, "/status/stream", f.slug, "https://different.invalid", 403)
	f.socket(t, f.a, "/status/stream", f.slug, "", 403)
	f.socket(t, f.other, "/status/stream", f.slug, f.server.URL, 404)
	status := f.socket(t, f.a, "/status/stream", f.slug, f.server.URL, 101)
	if *readChatTestUpdate(t, status, "status").Enabled {
		t.Fatal("new event enabled")
	}
	f.enable(t, f.a, true)
	if !*readChatTestUpdate(t, status, "status").Enabled {
		t.Fatal("enable not broadcast")
	}
	f.socket(t, f.b, "/stream", f.slug, f.server.URL, 409)
	f.enable(t, f.b, true)

	a1 := f.socket(t, f.a, "/stream", f.slug, f.server.URL, 101)
	a2 := f.socket(t, f.a, "/stream", f.slug, f.server.URL, 101)
	b := f.socket(t, f.b, "/stream", f.slug, f.server.URL, 101)
	for _, conn := range []*websocket.Conn{a1, a2, b} {
		readChatTestUpdate(t, conn, "status")
	}

	const text = "<script>alert(1)</script> https://example.invalid"
	send := `{"text":"` + text + `","request_id":"send-first-match-chat"}`
	path := chatTestPath(f.a, "/messages")
	var first struct {
		Message database.ChatMessage
		Changed bool
	}
	w := f.request(t, "POST", path, f.slug, "session1", send, 200)
	decodeChatTest(t, w, &first)
	if !first.Changed || !first.Message.Mine || first.Message.Text != text {
		t.Fatalf("invalid send: %+v", first)
	}
	for _, conn := range []*websocket.Conn{a1, a2} {
		msg := readChatTestUpdate(t, conn, "message").Message
		if msg.ID != first.Message.ID || msg.Mine || msg.AuthorID != 0 || msg.Nickname != "Fan 1" || msg.Text != text {
			t.Fatalf("public payload: %+v", msg)
		}
	}
	w = f.request(t, "POST", path, f.slug, "session1", send, 200)
	decodeChatTest(t, w, &first)
	if first.Changed {
		t.Fatal("retry inserted a duplicate")
	}
	reaction := fmt.Sprintf("/messages/%d/reaction", first.Message.ID)
	f.request(t, "PUT", chatTestPath(f.b, reaction), f.slug, "session2", `{"emoji":"🔥","active":true}`, 404)
	f.request(t, "PUT", chatTestPath(f.a, reaction), f.slug, "session2", `{"emoji":"🔥","active":true}`, 200)
	f.request(t, "PUT", chatTestPath(f.a, reaction), f.slug, "session2", `{"emoji":"🔥","active":true}`, 200)
	for _, conn := range []*websocket.Conn{a1, a2} {
		if got := readChatTestUpdate(t, conn, "reaction").Reaction; got.Count != 1 || got.Mine {
			t.Fatalf("reaction: %+v", got)
		}
	}
	f.request(t, "POST", "/admin"+chatTestPath(f.b, "/messages"), f.slug, f.admin, `{"text":"Evento B","request_id":"send-official-other-event"}`, 200)
	if m := readChatTestUpdate(t, b, "message").Message; m.Type != "official" || m.Text != "Evento B" {
		t.Fatalf("wrong room: %+v", m)
	}

	f.enable(t, f.a, false)
	for _, conn := range []*websocket.Conn{status, a1, a2} {
		if *readChatTestUpdate(t, conn, "status").Enabled {
			t.Fatal("disable not broadcast")
		}
	}
	if _, _, err := a2.ReadMessage(); err == nil {
		t.Fatal("disabled room still open")
	}
	f.socket(t, f.a, "/stream", f.slug, f.server.URL, 409)
	f.request(t, "POST", path, f.slug, "session1", send, 409)
	f.request(t, "GET", path, f.slug, "session1", "", 409)
	f.request(t, "PUT", chatTestPath(f.a, reaction), f.slug, "session2", `{"emoji":"🔥","active":false}`, 409)
	report := fmt.Sprintf("/messages/%d/report", first.Message.ID)
	f.request(t, "POST", chatTestPath(f.a, report), f.slug, "session2", `{"reason":"abuse"}`, 409)
	f.enable(t, f.a, true)
	reconnected := f.socket(t, f.a, "/stream", f.slug, f.server.URL, 101)
	if !*readChatTestUpdate(t, reconnected, "status").Enabled {
		t.Fatal("room not enabled on reconnect")
	}

	var history database.ChatHistory
	decodeChatTest(t, f.request(t, "GET", path, f.slug, "session2", "", 200), &history)
	if len(history.Messages) != 1 || history.Messages[0].ID != first.Message.ID || history.Messages[0].AuthorID != 0 ||
		len(history.Messages[0].Reactions) != 1 || !history.Messages[0].Reactions[0].Mine {
		t.Fatalf("history after reactivation: %+v", history)
	}
	history = database.ChatHistory{}
	decodeChatTest(t, f.request(t, "GET", chatTestPath(f.b, "/messages"), f.slug, "", "", 200), &history)
	if history.CanWrite || len(history.Messages) != 1 || history.Messages[0].Text != "Evento B" || len(history.Messages[0].Reactions) != 0 {
		t.Fatalf("other event history: %+v", history)
	}
	f.request(t, "PUT", chatTestPath(f.a, reaction), f.slug, "session2", `{"emoji":"🔥","active":false}`, 200)
	if got := readChatTestUpdate(t, reconnected, "reaction").Reaction; got.Count != 0 {
		t.Fatalf("reaction: %+v", got)
	}
}

func TestMatchChatAuthorizationModerationAndMetrics(t *testing.T) {
	f := newMatchChatFixture(t)
	f.enable(t, f.a, true)
	path := chatTestPath(f.a, "/messages")
	const body = `{"text":"Forza ragazzi","request_id":"moderation-test-001"}`
	f.request(t, "POST", path, f.slug, "", body, 401)
	f.request(t, "POST", path, f.slug, "not-a-session", body, 401)
	f.request(t, "POST", path, f.otherSlug, "session1", body, 404)
	f.request(t, "PUT", "/admin"+chatTestPath(f.other, "/settings"), f.otherSlug, f.admin, `{"enabled":true}`, 401)
	f.request(t, "PUT", "/admin"+chatTestPath(f.other, "/settings"), f.slug, f.admin, `{"enabled":true}`, 404)
	f.request(t, "POST", path, f.slug, "session2", `{"text":"x","request_id":"spoofed-author-test","author_user_id":5}`, 400)
	f.request(t, "POST", path, f.slug, "session2", fmt.Sprintf(`{"text":"%s","request_id":"long-message-test"}`, strings.Repeat("💪", 181)), 400)
	if _, err := f.rt.db.SQLConn().Exec(`UPDATE fan_profiles SET phone_verified_at=NULL WHERE id=(SELECT fan_id FROM fan_sessions WHERE token='session3')`); err != nil {
		t.Fatal(err)
	}
	f.request(t, "POST", path, f.slug, "session3", body, 401)

	fan := f.socket(t, f.a, "/stream", f.slug, f.server.URL, 101)
	readChatTestUpdate(t, fan, "status")
	f.request(t, "POST", path, f.slug, "session1", body, 200)
	messageID := readChatTestUpdate(t, fan, "message").Message.ID
	f.request(t, "POST", path, f.slug, "session1", `{"text":"Altro","request_id":"moderation-test-002"}`, 429)
	// Move only the stored timestamp past the short interval, leaving the
	// duplicate window intact without sleeping.
	if _, err := f.rt.db.SQLConn().Exec(`UPDATE event_chat_messages SET created_at=? WHERE id=?`, time.Now().Add(-6*time.Second).UnixMilli(), messageID); err != nil {
		t.Fatal(err)
	}
	w := f.request(t, "POST", path, f.slug, "session1", `{"text":"FORZA  ragazzi","request_id":"moderation-test-003"}`, 409)
	if !strings.Contains(w.Body.String(), "chat_duplicate") {
		t.Fatal(w.Body.String())
	}

	var history database.ChatHistory
	decodeChatTest(t, f.request(t, "GET", "/admin"+path, f.slug, f.admin, "", 200), &history)
	if len(history.Messages) != 1 || history.Messages[0].AuthorID == 0 {
		t.Fatalf("admin history: %+v", history)
	}
	mutePath := "/admin" + chatTestPath(f.a, fmt.Sprintf("/mutes/%d", history.Messages[0].AuthorID))
	f.request(t, "PUT", mutePath, f.slug, f.admin, `{"minutes":5}`, 200)
	f.request(t, "POST", path, f.slug, "session1", `{"text":"Altro","request_id":"moderation-test-004"}`, 403)
	f.request(t, "DELETE", mutePath, f.slug, f.admin, "", 200)

	msgPath := fmt.Sprintf("/messages/%d", messageID)
	f.request(t, "POST", chatTestPath(f.a, msgPath+"/report"), f.slug, "session2", `{"reason":"spam"}`, 200)
	f.request(t, "POST", chatTestPath(f.a, msgPath+"/report"), f.slug, "session2", `{"reason":"spam"}`, 200)
	var reports struct{ Reports []database.ChatReport }
	decodeChatTest(t, f.request(t, "GET", "/admin"+chatTestPath(f.a, "/reports"), f.slug, f.admin, "", 200), &reports)
	if len(reports.Reports) != 1 || reports.Reports[0].Status != "pending" {
		t.Fatalf("reports: %+v", reports)
	}
	f.request(t, "PUT", "/admin"+chatTestPath(f.a, fmt.Sprintf("/reports/%d", reports.Reports[0].ID)), f.slug, f.admin, "", 200)
	f.request(t, "DELETE", "/admin"+chatTestPath(f.a, msgPath), f.slug, f.admin, "", 200)
	if update := readChatTestUpdate(t, fan, "removed"); update.MessageID != messageID {
		t.Fatal(update)
	}
	decodeChatTest(t, f.request(t, "GET", path, f.slug, "", "", 200), &history)
	if len(history.Messages) != 0 {
		t.Fatalf("removed message exposed: %+v", history)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	for i, token := range []string{"", "session1", "session1"} {
		f.request(t, "POST", fmt.Sprintf("/events/%d/report-signals", f.a), f.slug, token, fmt.Sprintf(
			`{"events":[{"name":"match_chat_opened","id":"chat-open-%d","session_id":"browser-session","occurred_at":%q}]}`, i, now), 204)
	}
	var overview struct {
		Enabled bool
		Stats   database.ChatStats
	}
	decodeChatTest(t, f.request(t, "GET", "/admin"+chatTestPath(f.a, ""), f.slug, f.admin, "", 200), &overview)
	if !overview.Enabled || overview.Stats.Opens != 1 || overview.Stats.OpenersWhoWrote != 1 || overview.Stats.Reports != 1 || overview.Stats.Messages != 1 {
		t.Fatalf("incorrect metrics: %+v", overview)
	}
	var settingsChanges int
	if err := f.rt.db.SQLConn().QueryRow(`SELECT COUNT(*) FROM engagement_report_signals WHERE event_id=? AND name='match_chat_enabled'`, f.a).Scan(&settingsChanges); err != nil || settingsChanges != 1 {
		t.Fatalf("setting change analytics: %d (%v)", settingsChanges, err)
	}
}

func TestMatchChatConcurrentRetryAndCursor(t *testing.T) {
	f := newMatchChatFixture(t)
	f.rt.db.SQLConn().SetMaxOpenConns(4)
	f.enable(t, f.a, true)
	path := chatTestPath(f.a, "/messages")
	const body = `{"text":"Forza","request_id":"concurrent-chat-test"}`
	var requests sync.WaitGroup
	for i := 0; i < 8; i++ {
		requests.Add(1)
		go func() {
			defer requests.Done()
			f.request(t, "POST", path, f.slug, "session1", body, 200)
		}()
	}
	requests.Wait()
	for i := 0; i < database.ChatPageSize+3; i++ {
		f.request(t, "POST", "/admin"+path, f.slug, f.admin, fmt.Sprintf(`{"text":"Aggiornamento %d","request_id":"official-cursor-test-%d"}`, i, i), 200)
	}
	var history database.ChatHistory
	decodeChatTest(t, f.request(t, "GET", path, f.slug, "", "", 200), &history)
	if len(history.Messages) != database.ChatPageSize || history.NextBefore != history.Messages[0].ID {
		t.Fatalf("first page: %+v", history)
	}
	cursor := history.NextBefore
	decodeChatTest(t, f.request(t, "GET", fmt.Sprintf("%s?before=%d", path, cursor), f.slug, "", "", 200), &history)
	if len(history.Messages) != 4 || history.NextBefore != 0 || history.Messages[0].Text != "Forza" || history.Messages[3].ID >= cursor {
		t.Fatalf("second page: %+v", history)
	}
	if err := f.rt.db.PurgeEventData(f.a); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := f.rt.db.SQLConn().QueryRow(`SELECT COUNT(*) FROM event_chat_messages WHERE event_id=?`, f.a).Scan(&count); err != nil || count != 0 {
		t.Fatalf("purge left chat data: %d (%v)", count, err)
	}
}
