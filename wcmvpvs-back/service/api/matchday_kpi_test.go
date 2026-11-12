package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
)

func TestMatchdayArchiveRoundTrip(t *testing.T) {
	f := newEngagementReportFixture(t)
	rt := f.router
	rt.router = chi.NewRouter()
	rt.sessionTimeout = time.Hour
	rt.adminSessions = map[string]adminSession{
		"matchday-admin": {AdminID: f.ctx.AdminID, Role: "admin", OrganizationID: f.org.ID, ExpiresAt: time.Now().Add(time.Hour)},
		"matchday-staff": {AdminID: 8, Role: "staff", OrganizationID: f.org.ID, ExpiresAt: time.Now().Add(time.Hour)},
	}
	registerMatchdayKPIRoutes(rt)
	exec := func(query string, args ...interface{}) {
		t.Helper()
		if _, err := f.conn.Exec(query, args...); err != nil {
			t.Fatal(err)
		}
	}
	newEvent := func(org, home, away int, date string) int {
		t.Helper()
		id, err := rt.db.CreateEvent(database.Event{OrganizationID: org, Team1ID: home, Team2ID: away, StartDateTime: date})
		if err != nil {
			t.Fatal(err)
		}
		exec(`UPDATE events SET is_concluded=1 WHERE id=?`, id)
		return id
	}
	previous := newEvent(f.org.ID, f.org.TeamID, f.other.TeamID, "2026-10-14T19:30:00Z")
	older := newEvent(f.org.ID, f.org.TeamID, f.other.TeamID, "2026-09-30T19:30:00Z")
	foreign := newEvent(f.other.ID, f.other.TeamID, f.org.TeamID, "2026-10-20T19:30:00Z")
	otherDivision := newEvent(f.org.ID, f.other.TeamID, f.other.TeamID, "2026-10-20T18:00:00Z")
	exec(`UPDATE teams SET championship='Serie A' WHERE id=?`, f.org.TeamID)
	exec(`UPDATE events SET is_concluded=1 WHERE id=?`, f.event)
	exec(`INSERT INTO fan_profiles(id,organization_id,nickname,phone,created_at) VALUES(41,?,'test','fake','2026-10-21 15:00:00')`, f.org.ID)
	player, err := rt.db.CreatePlayer(database.Player{FirstName: "Test", LastName: "Giocatore", TeamID: f.org.TeamID, OrganizationID: f.org.ID})
	if err != nil {
		t.Fatal(err)
	}
	for i, device := range []string{"device-a", "device-b"} {
		exec(`INSERT INTO votes(event_id,player_id,ticket_code,ticket_signature,device_id,user_id) VALUES(?,?,?,?,?,41)`, f.event, player, fmt.Sprint(i), "signature", device)
	}
	signal := func(event int, id, name, device, at, metadata string, fan int) {
		t.Helper()
		exec(`INSERT INTO engagement_report_signals(event_id,signal_id,name,session_id,device_key,fan_id,occurred_at,metadata) VALUES(?,?,?,?,?,?,?,?)`,
			event, id, name, "session-"+device, reportOperationalDeviceKey(f.org.ID, device), fan, at, metadata)
	}
	for _, event := range []int{f.event, previous, older, otherDivision, foreign} {
		signal(event, "viewer-a", "page_view", "device-a", "2026-10-21T19:31:00Z", `{}`, 0)
	}
	signal(previous, "viewer-old", "page_view", "old-only", "2026-10-14T19:31:00Z", `{}`, 0)
	signal(f.event, "viewer-b", "page_view", "device-b", "2026-10-21T19:31:00Z", `{}`, 41)
	for i := 1; i <= 149; i++ {
		device := fmt.Sprintf("guest-%d", i)
		signal(f.event, fmt.Sprintf("guest-view-%d", i), "page_view", device, "2026-10-21T19:31:00Z", `{}`, 0)
		if i <= 94 {
			meta := fmt.Sprintf(`{"game_id":"tap","round_id":"game-round-%d"}`, i)
			signal(f.event, fmt.Sprintf("start-%d", i), "game_started", device, "2026-10-21T19:32:00Z", meta, 0)
			if i <= 47 {
				signal(f.event, fmt.Sprintf("complete-%d", i), "game_completed", device, "2026-10-21T19:32:05Z", meta, 0)
			}
		}
	}
	request := func(method, token, slug string, event int, suffix, payload string, status int) matchdayResponse {
		t.Helper()
		r := httptest.NewRequest(method, fmt.Sprintf("/admin/events/%d/matchday-kpis%s", event, suffix), bytes.NewBufferString(payload))
		r.Header.Set("X-Organization-Slug", slug)
		if token != "" {
			r.AddCookie(&http.Cookie{Name: adminCookieName(), Value: token})
		}
		w := httptest.NewRecorder()
		rt.router.ServeHTTP(w, r)
		engagementReportRequireStatus(t, w, status)
		var response matchdayResponse
		if status == http.StatusOK {
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(w.Body.String(), "device-a") || strings.Contains(w.Body.String(), "fake") {
				t.Fatal("aggregate response contains a fan identifier")
			}
		}
		return response
	}
	get := func(event int, suffix string) matchdayResponse {
		return request(http.MethodGet, "matchday-admin", f.org.Slug, event, suffix, "", http.StatusOK)
	}
	put := func(body string, status int) matchdayResponse {
		return request(http.MethodPut, "matchday-admin", f.org.Slug, f.event, "", body, status)
	}
	request(http.MethodGet, "", f.org.Slug, f.event, "", "", http.StatusUnauthorized)
	request(http.MethodGet, "matchday-staff", f.org.Slug, f.event, "", "", http.StatusForbidden)
	request(http.MethodGet, "matchday-admin", f.other.Slug, foreign, "", "", http.StatusUnauthorized)
	request(http.MethodGet, "matchday-admin", f.org.Slug, foreign, "", "", http.StatusNotFound)
	request(http.MethodPut, "matchday-admin", f.org.Slug, foreign, "", `{"values":[{"code":"attendees","value":2,"note":"Test fonte","revision":0}]}`, http.StatusNotFound)
	request(http.MethodGet, "matchday-admin", f.org.Slug, f.event, "?limit=0", "", http.StatusBadRequest)
	r := get(f.event, "")
	if len(r.Sections) != 10 || len(r.Executive) != 13 || r.Scope != "division" || r.Comparison.PreviousEvent == nil || r.Comparison.PreviousEvent.ID != previous || r.Comparison.Season != "2026/2027" {
		t.Fatalf("unexpected dashboard structure or comparison: %+v", r.Comparison)
	}
	for _, code := range r.Executive {
		matchdayTestMetric(t, r, code)
	}
	if club := get(f.event, "?scope=club&limit=1"); club.Comparison.PreviousEvent == nil || club.Comparison.PreviousEvent.ID != otherDivision || club.Comparison.EventsConsidered != 1 {
		t.Fatalf("incorrect club history: %+v", club.Comparison)
	}
	matchdayRequireValue(t, matchdayTestMetric(t, r, "direction_reach").Value, 150)
	matchdayRequireValue(t, matchdayTestMetric(t, r, "direction_far").Value, 95.0/150*100)
	matchdayRequireValue(t, matchdayTestMetric(t, r, "direction_retention").Value, 50)
	matchdayRequireValue(t, matchdayTestMetric(t, r, "mvp_voters").Value, 1)
	matchdayRequireValue(t, matchdayTestMetric(t, r, "mvp_conversion").Value, 100.0/150)
	matchdayRequireValue(t, matchdayTestMetric(t, r, "games_completion_rate").Value, 50)
	matchdayRequireValue(t, matchdayTestMetric(t, r, "crm_new_profiles").Value, 1)
	cmp := matchdayTestMetric(t, r, "direction_reach").Comparison
	matchdayRequireValue(t, cmp.Previous, 2)
	matchdayRequireValue(t, cmp.Average, 1.5)
	matchdayRequireValue(t, cmp.AbsoluteDelta, 148)
	if metric := matchdayTestMetric(t, r, "direction_penetration"); metric.Value != nil || metric.Status != "missing" {
		t.Fatalf("unexpected penetration before input: %+v", metric)
	}
	r = put(`{"values":[{"code":"attendees","value":1200,"note":"Contapersone arena","revision":0},{"code":"offline_revenue","value":30.50,"note":"Chiusura cassa","revision":0}]}`, http.StatusOK)
	matchdayRequireValue(t, matchdayTestMetric(t, r, "direction_penetration").Value, 12.5)
	matchdayRequireValue(t, matchdayTestMetric(t, r, "direction_far").Value, 95.0/150*100)
	matchdayRequireValue(t, matchdayTestMetric(t, r, "direction_revenue").Value, 30.5)
	put(`{"values":[{"code":"offline_revenue","value":99,"note":"Chiusura cassa","revision":1},{"code":"attendees","value":1000,"note":"Contapersone arena","revision":0}]}`, http.StatusConflict)
	matchdayRequireValue(t, matchdayTestMetric(t, get(f.event, ""), "direction_revenue").Value, 30.5)
	put(`{"values":[{"code":"direction_reach","value":1,"note":"Dato illegittimo","revision":0}]}`, http.StatusBadRequest)
	r = put(`{"values":[{"code":"attendees","value":0,"note":"Correzione ufficiale","revision":1},{"code":"offline_revenue","value":25,"note":"Rettifica cassa","revision":1}]}`, http.StatusOK)
	if metric := matchdayTestMetric(t, r, "direction_penetration"); metric.Value != nil || metric.Status != "not_applicable" || metric.Denominator == nil || *metric.Denominator != 0 {
		t.Fatalf("zero denominator lost: %+v", metric)
	}
	for _, field := range r.ManualFields {
		if field.Code == "attendees" || field.Code == "offline_revenue" {
			if field.Revision != 2 || len(field.History) != 1 || field.History[0].UpdatedBy != f.ctx.AdminID || field.UpdatedBy != f.ctx.AdminID {
				t.Fatalf("missing revision attribution for %s: %+v", field.Code, field)
			}
		}
	}
	shared, err := rt.loadReportManualSources(f.event, true)
	if err != nil || len(shared) != 1 || shared[0].Value != 0 || shared[0].Revision != 2 {
		t.Fatalf("engagement attendance source not shared: %v, %v", shared, err)
	}
}

func matchdayTestMetric(t *testing.T, response matchdayResponse, code string) matchdayMetric {
	t.Helper()
	for _, section := range response.Sections {
		for _, m := range section.Metrics {
			if m.Code == code {
				return m
			}
		}
	}
	t.Fatalf("metric %q not returned", code)
	return matchdayMetric{}
}

func matchdayRequireValue(t *testing.T, value *float64, want float64) {
	t.Helper()
	if value == nil || *value < want-1e-9 || *value > want+1e-9 {
		if value == nil {
			t.Fatalf("value missing; want %v", want)
		}
		t.Fatalf("value=%v; want %v", *value, want)
	}
}
