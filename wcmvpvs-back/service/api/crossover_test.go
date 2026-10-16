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

type crossoverAPIDB struct {
	basketballAPIDB
	duel, start int
	samples     []database.CrossoverSample
}

func (db *crossoverAPIDB) CreateCrossoverSession(event, org, fan int, device string, now time.Time) (database.CrossoverSession, error) {
	db.calls++
	db.fanID = fan
	db.device = device
	if event != 42 || org != 7 {
		return database.CrossoverSession{}, sql.ErrNoRows
	}
	return database.CrossoverSession{SessionID: "12345678-1234-1234-1234-123456789012", StartedAt: now.UTC().Format(time.RFC3339Nano), EndsAt: now.Add(25 * time.Second).UTC().Format(time.RFC3339Nano), ServerNow: now.UTC().Format(time.RFC3339Nano), RemainingMS: 25000, NextDuel: 1, CrossoverStats: database.CrossoverStats{Wins: 1, Streak: 1, BestStreak: 1, QualityTotal: 94}}, db.err
}

func (db *crossoverAPIDB) ResolveCrossoverDuel(event, org, fan int, device, session string, duel, start int, samples []database.CrossoverSample, now time.Time) (database.CrossoverDuelResult, error) {
	db.calls++
	db.fanID = fan
	db.device = device
	db.duel = duel
	db.start = start
	db.samples = samples
	return database.CrossoverDuelResult{SessionID: session, Duel: duel, Won: true, Clean: true, Quality: 94, Reason: "beat", Wallet: 12, CoinsEarned: 1, CrossoverEarned: 4, NextDuel: duel + 1, RemainingMS: 22000, CrossoverStats: database.CrossoverStats{Wins: 2, Streak: 2, BestStreak: 2, QualityTotal: 188}}, db.err
}

func TestCrossoverAPIContractIdentityAndBodyValidation(t *testing.T) {
	db := &crossoverAPIDB{basketballAPIDB: basketballAPIDB{fanOrg: 7}}
	router := &_router{db: db, coinsHub: newSSEHub()}
	ctx := reqcontext.RequestContext{OrganizationID: 7, Logger: logrus.New()}
	mux := chi.NewRouter()
	mux.Post("/{eventId}/sessions", func(w http.ResponseWriter, r *http.Request) { router.createCrossoverSession(w, r, ctx) })
	mux.Post("/{eventId}/sessions/{sessionId}/duels", func(w http.ResponseWriter, r *http.Request) { router.submitCrossoverDuel(w, r, ctx) })
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
	var session database.CrossoverSession
	if err := json.Unmarshal(response.Body.Bytes(), &session); err != nil || session.RemainingMS != 25000 || session.NextDuel != 1 || session.Wins != 1 || session.Streak != 1 || session.BestStreak != 1 || session.QualityTotal != 94 || session.ServerNow == "" {
		t.Fatalf("session body %+v: %v", session, err)
	}
	path := "/42/sessions/12345678-1234-1234-1234-123456789012/duels"
	valid := `{"duel":1,"duel_start_ms":2000,"samples":[{"t_ms":0,"x":0},{"t_ms":100,"x":0.8},{"t_ms":400,"x":0.8},{"t_ms":500,"x":-0.8}]}`
	for _, body := range []string{
		`{"duel":0}`,
		`{"duel":0,"duel_start_ms":0,"samples":[{"t_ms":0},{"t_ms":100,"x":0.8}]}`,
		`{"duel":0,"duel_start_ms":0,"samples":[{"x":0},{"t_ms":100,"x":0.8}]}`,
		`{"duel":0,"duel_start_ms":0,"samples":[{"t_ms":0,"x":null},{"t_ms":100,"x":0.8}]}`,
		`{"duel":0.5,"duel_start_ms":0,"samples":[{"t_ms":0,"x":0},{"t_ms":100,"x":0.8}]}`,
		`{"duel":0,"duel_start_ms":0,"samples":[{"t_ms":0,"x":0},{"t_ms":100.5,"x":0.8}]}`,
		`{"duel":0,"duel_start_ms":0,"samples":[{"t_ms":0,"x":0,"won":true},{"t_ms":100,"x":0.8}]}`,
		strings.Replace(valid, `"samples":`, `"wallet":999,"samples":`, 1),
		strings.Replace(valid, `"x":0.8`, `"x":1e999`, 1),
		valid + ` {}`,
		valid + strings.Repeat(" ", 16384),
	} {
		before := db.calls
		if response := request(path, body, "device", ""); response.Code != 400 || db.calls != before {
			t.Fatalf("invalid body %.180s: %d", body, response.Code)
		}
	}
	tooMany := make([]map[string]interface{}, 129)
	for index := range tooMany {
		tooMany[index] = map[string]interface{}{"t_ms": index, "x": 0}
	}
	body, err := json.Marshal(map[string]interface{}{"duel": 0, "duel_start_ms": 0, "samples": tooMany})
	if err != nil {
		t.Fatal(err)
	}
	if response := request(path, string(body), "device", ""); response.Code != 400 {
		t.Fatalf("too many samples %d", response.Code)
	}
	response = request(path, valid, "device", "")
	if response.Code != 200 || db.fanID != 0 || db.duel != 1 || db.start != 2000 || len(db.samples) != 4 || db.samples[3].TMS != 500 || db.samples[3].X != -.8 {
		t.Fatalf("duel %d %s", response.Code, response.Body.String())
	}
	var result database.CrossoverDuelResult
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil || result.CrossoverEarned != 4 || result.CoinsEarned != 1 || result.Reason != "beat" || result.NextDuel != 2 || result.Wins != 2 || result.QualityTotal != 188 {
		t.Fatalf("result %+v: %v", result, err)
	}
	for _, tc := range []struct {
		err    error
		status int
	}{{database.ErrBasketballClosed, 409}, {database.ErrCrossoverTooEarly, 409}, {database.ErrCrossoverOrder, 409}, {database.ErrCrossoverExpired, 410}, {database.ErrCrossoverGesture, 400}, {sql.ErrNoRows, 404}} {
		db.err = tc.err
		response = request(path, valid, "device", "")
		if response.Code != tc.status || !strings.Contains(response.Body.String(), "message") {
			t.Fatalf("error %v: %d %s", tc.err, response.Code, response.Body.String())
		}
	}
}
