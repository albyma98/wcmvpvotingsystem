package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

type playmakerAPIDB struct {
	basketballAPIDB
	attempt, elapsed int
}

func (db *playmakerAPIDB) CreatePlaymakerSession(event, org, fan int, device string, now time.Time) (database.PlaymakerSession, error) {
	db.calls++
	db.fanID = fan
	db.device = device
	if event != 42 || org != 7 {
		return database.PlaymakerSession{}, sql.ErrNoRows
	}
	return database.PlaymakerSession{SessionID: "12345678-1234-1234-1234-123456789012", StartedAt: now.UTC().Format(time.RFC3339Nano), EndsAt: now.Add(30 * time.Second).UTC().Format(time.RFC3339Nano), ServerNow: now.UTC().Format(time.RFC3339Nano), RemainingMS: 30000, Attempt: 0}, db.err
}

func (db *playmakerAPIDB) ResolvePlaymakerPass(event, org, fan int, device, session string, attempt, elapsed int, dx, dy float64, now time.Time) (database.PlaymakerPassResult, error) {
	db.calls++
	db.fanID = fan
	db.device = device
	db.attempt = attempt
	db.elapsed = elapsed
	db.dx = dx
	db.dy = dy
	return database.PlaymakerPassResult{SessionID: session, Attempt: attempt, Assisted: true, Clean: true, Reason: "assist", Wallet: 12, CoinsEarned: 1, PlaymakerEarned: 4, NextAttempt: attempt + 1, RemainingMS: 28000}, db.err
}

func TestPlaymakerAPIContractAndIdentity(t *testing.T) {
	db := &playmakerAPIDB{basketballAPIDB: basketballAPIDB{fanOrg: 7}}
	router := &_router{db: db, coinsHub: newSSEHub()}
	ctx := reqcontext.RequestContext{OrganizationID: 7, Logger: logrus.New()}
	mux := chi.NewRouter()
	mux.Post("/{eventId}/sessions", func(w http.ResponseWriter, r *http.Request) { router.createPlaymakerSession(w, r, ctx) })
	mux.Post("/{eventId}/sessions/{sessionId}/passes", func(w http.ResponseWriter, r *http.Request) { router.submitPlaymakerPass(w, r, ctx) })
	request := func(path, body, device, token string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		r.Header.Set("X-Device-ID", device)
		r.Header.Set("X-Fan-Session", token)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	for _, tc := range []struct {
		device, token string
		status        int
	}{{"", "", 400}, {"device", "invalid", 401}} {
		if response := request("/42/sessions", "{}", tc.device, tc.token); response.Code != tc.status || db.calls != 0 {
			t.Fatalf("identity %+v: %d", tc, response.Code)
		}
	}
	db.fanOrg = 8
	if response := request("/42/sessions", "{}", "device", "valid"); response.Code != 403 || db.calls != 0 {
		t.Fatalf("cross-org session %d", response.Code)
	}
	db.fanOrg = 7
	response := request("/42/sessions", "{}", "device", "valid")
	if response.Code != 200 || db.fanID != 9 || db.device != "device" || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("session %d %s", response.Code, response.Body.String())
	}
	var session database.PlaymakerSession
	if err := json.Unmarshal(response.Body.Bytes(), &session); err != nil || session.RemainingMS != 30000 || session.Attempt != 0 || session.ServerNow == "" {
		t.Fatalf("session body %+v: %v", session, err)
	}
	path := "/42/sessions/12345678-1234-1234-1234-123456789012/passes"
	for _, body := range []string{`{"dx":0}`, `{"attempt":0,"elapsed_ms":0,"dx":null,"dy":-218}`, `{"attempt":0.5,"elapsed_ms":0,"dx":-98,"dy":-218}`, `{"attempt":0,"elapsed_ms":0,"dx":-98,"dy":-218,"wallet":999}`, `{"attempt":0,"elapsed_ms":0,"dx":-98,"dy":-218} {}`} {
		if response := request(path, body, "device", ""); response.Code != 400 {
			t.Fatalf("invalid body %s: %d", body, response.Code)
		}
	}
	response = request(path, `{"attempt":0,"elapsed_ms":1200,"dx":-98,"dy":-218}`, "device", "")
	if response.Code != 200 || db.fanID != 0 || db.attempt != 0 || db.elapsed != 1200 || db.dx != -98 || db.dy != -218 {
		t.Fatalf("pass %d %s", response.Code, response.Body.String())
	}
	var result database.PlaymakerPassResult
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil || result.PlaymakerEarned != 4 || result.CoinsEarned != 1 || result.Reason != "assist" || result.NextAttempt != 1 {
		t.Fatalf("result %+v: %v", result, err)
	}
	for _, tc := range []struct {
		err    error
		status int
	}{{database.ErrBasketballClosed, 409}, {database.ErrPlaymakerTooEarly, 409}, {database.ErrPlaymakerAttempt, 409}, {database.ErrPlaymakerExpired, 410}, {database.ErrPlaymakerGesture, 400}, {sql.ErrNoRows, 404}} {
		db.err = tc.err
		response = request(path, `{"attempt":0,"elapsed_ms":0,"dx":-98,"dy":-218}`, "device", "")
		if response.Code != tc.status || !strings.Contains(response.Body.String(), "message") {
			t.Fatalf("error %v: %d %s", tc.err, response.Code, response.Body.String())
		}
	}
}
