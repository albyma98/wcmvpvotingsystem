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

type volleyballAPIDB struct {
	database.AppDatabase
	err                     error
	calls, fanID, fanOrg    int
	device, gameID, roundID string
	aimX, aimY, timing      float64
}

func (db *volleyballAPIDB) GetFanBySessionToken(token, device string) (database.FanProfileSummary, error) {
	if token != "valid" {
		return database.FanProfileSummary{}, sql.ErrNoRows
	}
	return database.FanProfileSummary{Profile: database.FanProfile{ID: 9, OrganizationID: db.fanOrg}}, nil
}

func (db *volleyballAPIDB) CreateVolleyballRound(event, org, fan int, device, game string, now time.Time) (database.VolleyballRound, error) {
	db.calls++
	db.fanID, db.device, db.gameID = fan, device, game
	if event != 42 || org != 7 {
		return database.VolleyballRound{}, sql.ErrNoRows
	}
	return database.VolleyballRound{RoundID: "12345678-1234-1234-1234-123456789012", GameID: game, Target: database.VolleyballPoint{X: 82, Y: 136}, TimingTarget: .7}, db.err
}

func (db *volleyballAPIDB) ResolveVolleyballRound(event, org, fan int, device, game, round string, aimX, aimY, timing float64, now time.Time) (database.VolleyballResult, error) {
	db.calls++
	db.fanID, db.device, db.gameID, db.roundID = fan, device, game, round
	db.aimX, db.aimY, db.timing = aimX, aimY, timing
	return database.VolleyballResult{RoundID: round, GameID: game, Success: true, Perfect: true, Wallet: 12, CoinsEarned: 1, GameEarned: 4}, db.err
}

func TestVolleyballAPIContractAndScope(t *testing.T) {
	db := &volleyballAPIDB{fanOrg: 7}
	router := &_router{db: db, coinsHub: newSSEHub()}
	ctx := reqcontext.RequestContext{OrganizationID: 7, Logger: logrus.New()}
	mux := chi.NewRouter()
	mux.Post("/{eventId}/volleyball/{gameId}/rounds", func(w http.ResponseWriter, r *http.Request) { router.createVolleyballRound(w, r, ctx) })
	mux.Post("/{eventId}/volleyball/{gameId}/rounds/{roundId}", func(w http.ResponseWriter, r *http.Request) { router.submitVolleyballRound(w, r, ctx) })
	request := func(path, body, device, token string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		r.Header.Set("X-Device-ID", device)
		r.Header.Set("X-Fan-Session", token)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	base := "/42/volleyball/volley-rush/rounds"
	if response := request(base, "", "", ""); response.Code != 400 || db.calls != 0 {
		t.Fatalf("missing device %d", response.Code)
	}
	if response := request(base, "", "device", "invalid"); response.Code != 401 || db.calls != 0 {
		t.Fatalf("invalid session %d", response.Code)
	}
	db.fanOrg = 8
	if response := request(base, "", "device", "valid"); response.Code != 403 || db.calls != 0 {
		t.Fatalf("foreign session %d", response.Code)
	}
	db.fanOrg = 7
	response := request(base, "", "device", "valid")
	if response.Code != 200 || db.fanID != 9 || db.device != "device" || db.gameID != "volley-rush" || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("issued %d %s", response.Code, response.Body.String())
	}
	var round database.VolleyballRound
	if err := json.Unmarshal(response.Body.Bytes(), &round); err != nil || round.Target.X != 82 || round.TimingTarget != .7 {
		t.Fatalf("round %+v: %v", round, err)
	}
	path := base + "/12345678-1234-1234-1234-123456789012"
	for _, body := range []string{
		`{"aim_x":82,"aim_y":136}`,
		`{"aim_x":null,"aim_y":136,"timing":0.7}`,
		`{"aim_x":82,"aim_y":136,"timing":null}`,
		`{"aim_x":82,"aim_y":136,"timing":0.7,"success":true}`,
		`{"aim_x":82,"aim_y":136,"timing":0.7,"coins":99}`,
		`{"aim_x":82,"aim_y":136,"timing":0.7} {}`,
		`{"aim_x":NaN,"aim_y":136,"timing":0.7}`,
	} {
		calls := db.calls
		if response := request(path, body, "device", ""); response.Code != 400 || db.calls != calls {
			t.Fatalf("bad payload %s: %d", body, response.Code)
		}
	}
	response = request(path, `{"aim_x":82,"aim_y":136,"timing":0.7}`, "device", "")
	if response.Code != 200 || db.fanID != 0 || db.aimX != 82 || db.aimY != 136 || db.timing != .7 || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("submitted %d %s", response.Code, response.Body.String())
	}
	var result database.VolleyballResult
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil || !result.Success || !result.Perfect || result.GameEarned != 4 || result.CoinsEarned != 1 {
		t.Fatalf("result %+v: %v", result, err)
	}
	for _, game := range []string{"volley-rush", "setter-pro", "monster-block"} {
		response := request("/42/volleyball/"+game+"/rounds", "", "device", "")
		if response.Code != 200 || db.gameID != game {
			t.Fatalf("game %s %d", game, response.Code)
		}
	}
	for _, invalid := range []string{"/42/volleyball/unknown/rounds", base + "/bad-id"} {
		calls := db.calls
		if response := request(invalid, "{}", "device", ""); response.Code != 400 || db.calls != calls {
			t.Fatalf("bad path %s %d", invalid, response.Code)
		}
	}
	for _, tc := range []struct {
		err    error
		status int
	}{{database.ErrVolleyballClosed, 409}, {database.ErrVolleyballTooEarly, 409}, {database.ErrVolleyballExpired, 410}, {database.ErrVolleyballGesture, 400}, {database.ErrVolleyballGame, 400}, {sql.ErrNoRows, 404}} {
		db.err = tc.err
		if response := request(path, `{"aim_x":82,"aim_y":136,"timing":0.7}`, "device", ""); response.Code != tc.status {
			t.Fatalf("error %v: %d", tc.err, response.Code)
		}
	}
}
