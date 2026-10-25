package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

type votingTransport func(*http.Request) (*http.Response, error)

func (fn votingTransport) RoundTrip(r *http.Request) (*http.Response, error) { return fn(r) }

func votingAPIFixture(t *testing.T) (*_router, reqcontext.RequestContext, int, int) {
	t.Helper()
	conn, err := sql.Open("sqlite3", "file:"+filepath.Join(t.TempDir(), "voting.db")+"?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { conn.Close() })
	db, err := database.New(conn)
	if err != nil {
		t.Fatal(err)
	}
	club, err := db.CreateOrganization(database.Organization{Name: "Club", Slug: "closure-club", IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	other, err := db.CreateOrganization(database.Organization{Name: "Other", Slug: "closure-other", IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	result, err := conn.Exec(`INSERT INTO events(organization_id,team1_id,team2_id,start_datetime,is_active) VALUES(?,?,?,'2026-10-25T18:00:00Z',1)`, club.ID, club.TeamID, other.TeamID)
	if err != nil {
		t.Fatal(err)
	}
	event, _ := result.LastInsertId()
	result, err = conn.Exec(`INSERT INTO players(first_name,last_name,role,team_id,organization_id) VALUES('Mario','Rossi','player',?,?)`, club.TeamID, club.ID)
	if err != nil {
		t.Fatal(err)
	}
	player, _ := result.LastInsertId()
	for i := 1; i <= 3; i++ {
		result, err = conn.Exec(`INSERT INTO fan_profiles(organization_id,nickname,phone,phone_e164,phone_verified_at,accepted_terms) VALUES(?,?,?,?,CURRENT_TIMESTAMP,1)`, club.ID, fmt.Sprintf("Fan %d", i), fmt.Sprintf("+39333999900%d", i), fmt.Sprintf("+39333999900%d", i))
		if err != nil {
			t.Fatal(err)
		}
		fan, _ := result.LastInsertId()
		if _, err := conn.Exec(`INSERT INTO fan_sessions(token,fan_id,device_id) VALUES(?,?,?)`, fmt.Sprintf("session%d", i), fan, fmt.Sprintf("device%d", i)); err != nil {
			t.Fatal(err)
		}
		if _, err := conn.Exec(`INSERT INTO fan_wallets(fan_id,coins) VALUES(?,?)`, fan, 400-i*100); err != nil {
			t.Fatal(err)
		}
		if _, err := conn.Exec(`INSERT INTO votes(event_id,player_id,ticket_code,ticket_signature,device_id) VALUES(?,?,?,'sig',?)`, event, player, fmt.Sprintf("%04d", i), fmt.Sprintf("device%d", i)); err != nil {
			t.Fatal(err)
		}
		if _, err := conn.Exec(`INSERT INTO event_prizes(event_id,name,position) VALUES(?,?,?)`, event, fmt.Sprintf("Premio %d", i), i); err != nil {
			t.Fatal(err)
		}
	}
	_, err = db.SaveVotingSettings(club.ID, database.VotingSettings{DrawMode: database.VotingDrawLeaderboard, TemplateSID: "HX" + strings.Repeat("a", 32), Recipients: []database.VotingRecipient{
		{Name: "Primo", Phone: "+393331234567", Enabled: true}, {Name: "Secondo", Phone: "+393331234568", Enabled: true}}})
	if err != nil {
		t.Fatal(err)
	}
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	rt := &_router{db: db, baseLogger: logger, twilioMessaging: newTwilioMessagingClient(twilioMessagingConfig{AccountSID: "ACtest", AuthToken: "test", WhatsAppMessagingServiceSID: "MGtest"})}
	return rt, reqcontext.RequestContext{OrganizationID: club.ID, OrganizationSlug: club.Slug, AdminRole: "superadmin", Logger: logger}, int(event), other.ID
}

func votingAPIRequest(rt *_router, ctx reqcontext.RequestContext, event int, method, path string, handler httpRouterHandler) *httptest.ResponseRecorder {
	mux := chi.NewRouter()
	serve := func(w http.ResponseWriter, r *http.Request) { handler(w, r, ctx) }
	if method == http.MethodGet {
		mux.Get("/{id}", serve)
	} else {
		mux.Post("/{id}", serve)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(method, "/"+strconv.Itoa(event)+path, nil))
	return w
}

func drainVotingQueue(t *testing.T, rt *_router) {
	t.Helper()
	for i := 0; i < 20; i++ {
		processed, err := rt.processVotingNotification()
		if err != nil {
			t.Fatal(err)
		}
		if !processed {
			return
		}
	}
	t.Fatal("notification queue did not drain")
}

func TestVotingClosureAPIStaffClosesAndTwilioReceivesCompanyTemplate(t *testing.T) {
	rt, ctx, event, other := votingAPIFixture(t)
	calls := 0
	rt.twilioMessaging.httpClient.Transport = votingTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Host != "api.twilio.com" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %v", r)
		}
		account, token, ok := r.BasicAuth()
		if !ok || account != "ACtest" || token != "test" {
			t.Fatal("missing server credentials")
		}
		body, _ := io.ReadAll(r.Body)
		form, err := url.ParseQuery(string(body))
		if err != nil {
			t.Fatal(err)
		}
		if form.Get("Body") != "" || form.Get("ContentSid") != "HX"+strings.Repeat("a", 32) || form.Get("MessagingServiceSid") != "MGtest" {
			t.Fatalf("template parameters: %v", form)
		}
		var variables map[string]string
		if err := json.Unmarshal([]byte(form.Get("ContentVariables")), &variables); err != nil {
			t.Fatal(err)
		}
		if len(variables) != 8 || variables["1"] != "Mario Rossi" || variables["2"] != "100,00" || variables["4"] != "0001" || variables["8"] != "0003" {
			t.Fatalf("template variables: %v", variables)
		}
		if form.Get("To") != "whatsapp:+393331234567" && form.Get("To") != "whatsapp:+393331234568" {
			t.Fatalf("recipient: %s", form.Get("To"))
		}
		return &http.Response{StatusCode: 201, Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`{"sid":"SM%d","status":"queued"}`, calls))), Header: make(http.Header)}, nil
	})
	ctx.AdminRole = "staff"
	w := votingAPIRequest(rt, ctx, event, http.MethodPost, "", rt.closeEventVoting)
	if w.Code != 200 || calls != 0 {
		t.Fatalf("close blocked on external send: %d calls=%d %s", w.Code, calls, w.Body.String())
	}
	drainVotingQueue(t, rt)
	if calls != 2 {
		t.Fatalf("organizer sends: %d", calls)
	}
	for i := 0; i < 2; i++ {
		w = votingAPIRequest(rt, ctx, event, http.MethodPost, "", rt.closeEventVoting)
		if w.Code != 200 {
			t.Fatalf("repeat: %d", w.Code)
		}
		drainVotingQueue(t, rt)
	}
	if calls != 2 {
		t.Fatal("repeated close resent messages")
	}
	w = votingAPIRequest(rt, ctx, event, http.MethodGet, "", rt.getVotingClosure)
	if w.Code != 200 || strings.Contains(w.Body.String(), "+393331234567") || !strings.Contains(w.Body.String(), `"status":"submitted"`) {
		t.Fatalf("status/phone privacy: %d %s", w.Code, w.Body.String())
	}
	ctx.OrganizationID = other
	w = votingAPIRequest(rt, ctx, event, http.MethodGet, "", rt.getVotingClosure)
	if w.Code < 400 {
		t.Fatalf("another society read results: %d", w.Code)
	}
}

func TestVotingClosureSettingsRequireSuperadminAndValidatePayload(t *testing.T) {
	rt, ctx, _, _ := votingAPIFixture(t)
	for _, role := range []string{"staff", "bar", "partner", ""} {
		ctx.AdminRole = role
		for _, handler := range []httpRouterHandler{rt.getVotingSettings, rt.putVotingSettings} {
			w := httptest.NewRecorder()
			handler(w, httptest.NewRequest(http.MethodPut, "/", strings.NewReader(`{}`)), ctx)
			if w.Code != 403 {
				t.Fatalf("%s accessed configuration: %d", role, w.Code)
			}
		}
	}
	ctx.AdminRole = "superadmin"
	for _, body := range []string{`{"template_sid":"wrong","draw_mode":"random"}`, `{"draw_mode":"random","unexpected":true}`, `{"draw_mode":"random"} {}`} {
		w := httptest.NewRecorder()
		rt.putVotingSettings(w, httptest.NewRequest(http.MethodPut, "/", strings.NewReader(body)), ctx)
		if w.Code != 400 {
			t.Fatalf("accepted %s: %d", body, w.Code)
		}
	}
	w := httptest.NewRecorder()
	rt.getVotingSettings(w, httptest.NewRequest(http.MethodGet, "/", nil), ctx)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "HX"+strings.Repeat("a", 32)) {
		t.Fatalf("settings read: %d %s", w.Code, w.Body.String())
	}
}

func TestVotingClosureTwilioFailuresRetryOnlyFailedOrganizers(t *testing.T) {
	rt, ctx, event, _ := votingAPIFixture(t)
	calls := 0
	reject := true
	rt.twilioMessaging.httpClient.Transport = votingTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		body, _ := io.ReadAll(r.Body)
		form, _ := url.ParseQuery(string(body))
		if reject && strings.HasSuffix(form.Get("To"), "4568") {
			return &http.Response{StatusCode: 400, Body: io.NopCloser(strings.NewReader(`{"code":21656}`)), Header: make(http.Header)}, nil
		}
		return &http.Response{StatusCode: 201, Body: io.NopCloser(strings.NewReader(`{"sid":"SMaccepted","status":"queued"}`)), Header: make(http.Header)}, nil
	})
	w := votingAPIRequest(rt, ctx, event, http.MethodPost, "", rt.closeEventVoting)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	drainVotingQueue(t, rt)
	_, notifications, err := rt.db.GetVotingClosure(event, ctx.OrganizationID)
	if err != nil || notifications[0].Status != "submitted" || notifications[1].Status != "failed" {
		t.Fatalf("failure states: %+v %v", notifications, err)
	}
	reject = false
	w = votingAPIRequest(rt, ctx, event, http.MethodPost, "", rt.retryVotingNotifications)
	if w.Code != 202 {
		t.Fatal(w.Code)
	}
	drainVotingQueue(t, rt)
	if calls != 3 {
		t.Fatalf("successful recipient resent: %d", calls)
	}
	prizes, _ := rt.db.ListEventPrizes(event)
	if prizes[0].Winner == nil || prizes[0].Winner.TicketCode != "0001" {
		t.Fatal("provider failure changed results")
	}
}

func TestVotingClosureAmbiguousTwilioSendIsNotBlindlyRetried(t *testing.T) {
	rt, ctx, event, _ := votingAPIFixture(t)
	calls := 0
	rt.twilioMessaging.httpClient.Transport = votingTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		return nil, errors.New("connection lost after request write")
	})
	if w := votingAPIRequest(rt, ctx, event, http.MethodPost, "", rt.closeEventVoting); w.Code != 200 {
		t.Fatal(w.Code)
	}
	drainVotingQueue(t, rt)
	_, notifications, _ := rt.db.GetVotingClosure(event, ctx.OrganizationID)
	if notifications[0].Status != "unknown" || notifications[1].Status != "unknown" {
		t.Fatalf("ambiguous state: %+v", notifications)
	}
	_ = rt.db.RetryVotingNotifications(event, ctx.OrganizationID)
	drainVotingQueue(t, rt)
	if calls != 2 {
		t.Fatalf("ambiguous send repeated: %d", calls)
	}
}

func TestVotingClosureKeepsWinnerWhatsAppNotifications(t *testing.T) {
	rt, ctx, event, _ := votingAPIFixture(t)
	rt.twilioMessaging.cfg.WhatsAppWinnerContentSID = "HX" + strings.Repeat("c", 32)
	winnerCalls := 0
	rt.twilioMessaging.httpClient.Transport = votingTransport(func(r *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(r.Body)
		form, _ := url.ParseQuery(string(body))
		if form.Get("ContentSid") == rt.twilioMessaging.cfg.WhatsAppWinnerContentSID {
			winnerCalls++
		}
		return &http.Response{StatusCode: 201, Body: io.NopCloser(strings.NewReader(`{"sid":"SMaccepted","status":"queued"}`)), Header: make(http.Header)}, nil
	})
	if w := votingAPIRequest(rt, ctx, event, http.MethodPost, "", rt.closeEventVoting); w.Code != 200 {
		t.Fatal(w.Code)
	}
	drainVotingQueue(t, rt)
	prizes, _ := rt.db.ListEventPrizes(event)
	if winnerCalls != 3 || prizes[0].Winner.NotifiedAt == "" {
		t.Fatalf("winner notifications lost: calls=%d winner=%+v", winnerCalls, prizes[0].Winner)
	}
}
