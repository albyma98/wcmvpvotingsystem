package api

import (
	"database/sql"
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

func TestTapVolleyMatchmakingSeparatesGamesAndTenants(t *testing.T) {
	manager := newTapLiveManager()
	now := time.Now()
	manager.putWaiting(tapLiveWaitingFan{fanID: 1, eventID: 42, orgID: 7, gameID: "tap", createdAt: now})
	manager.putWaiting(tapLiveWaitingFan{fanID: 2, eventID: 42, orgID: 8, gameID: "tap-volley", createdAt: now})
	manager.putWaiting(tapLiveWaitingFan{fanID: 3, eventID: 43, orgID: 7, gameID: "tap-volley", createdAt: now})
	if _, found := manager.matchFor(9, 42, 7, now, "tap-volley"); found {
		t.Fatal("volley matched another game, event or tenant")
	}
	manager.putWaiting(tapLiveWaitingFan{fanID: 4, eventID: 42, orgID: 7, gameID: "tap-volley", createdAt: now})
	if _, found := manager.waitingFor(4, 42, 7, now, "tap"); found {
		t.Fatal("basket read volley queue")
	}
	opponent, found := manager.matchFor(9, 42, 7, now, "tap-volley")
	if !found || opponent.fanID != 4 {
		t.Fatalf("volley match: %+v %v", opponent, found)
	}
	opponent, found = manager.matchFor(9, 42, 7, now, "tap")
	if !found || opponent.fanID != 1 {
		t.Fatalf("basket queue changed: %+v %v", opponent, found)
	}
}

type tapVolleyAPIDB struct {
	database.AppDatabase
	match           database.TapLiveMatch
	submits, aborts int
}

func (db *tapVolleyAPIDB) GetFanBySessionToken(string, string) (database.FanProfileSummary, error) {
	return database.FanProfileSummary{Profile: database.FanProfile{ID: 1, OrganizationID: 7}}, nil
}
func (db *tapVolleyAPIDB) GetTapLiveMatchByID(id string) (database.TapLiveMatch, error) {
	if id != "match" {
		return database.TapLiveMatch{}, sql.ErrNoRows
	}
	return db.match, nil
}
func (db *tapVolleyAPIDB) SubmitTapLiveScore(string, int, int) error { db.submits++; return nil }
func (db *tapVolleyAPIDB) TryFinalizeTapLiveMatch(string) error      { return nil }
func (db *tapVolleyAPIDB) AbortTapLiveMatch(string, int) error       { db.aborts++; return nil }

func TestTapVolleyLiveEndpointsRejectTheOtherGame(t *testing.T) {
	db := &tapVolleyAPIDB{match: database.TapLiveMatch{ID: 1, MatchID: "match", GameID: "tap", EventID: 42, OrganizationID: 7, Fan1ID: 1, Fan2ID: 2, Status: "finished"}}
	router := &_router{db: db, tapLive: newTapLiveManager(), tapLiveRematch: newTapLiveRematchManager(), tapLiveHub: newSSEHub()}
	ctx := reqcontext.RequestContext{OrganizationID: 7, Logger: logrus.New()}
	mux := chi.NewRouter()
	bind := func(handler httpRouterHandler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) { withTapLiveGame("tap-volley", handler)(w, r, ctx) }
	}
	mux.Get("/{eventId}/tap-volley-live/result", bind(router.getTapLiveResult))
	mux.Post("/{eventId}/tap-volley-live/submit", bind(router.postTapLiveSubmit))
	mux.Post("/{eventId}/tap-volley-live/abort", bind(router.postTapLiveAbort))
	mux.Post("/{eventId}/tap-volley-live/rematch", bind(router.postTapLiveRematch))
	request := func(method, suffix string, want int) {
		t.Helper()
		req := httptest.NewRequest(method, "/42/tap-volley-live/"+suffix, strings.NewReader(`{"match_id":"match","score":10}`))
		req.Header.Set("X-Fan-Session", "valid")
		req.Header.Set("X-Device-ID", "device")
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		if response.Code != want {
			t.Fatalf("%s: %d want %d %s", suffix, response.Code, want, response.Body.String())
		}
	}
	request("GET", "result?match_id=match", 404)
	request("POST", "submit", 403)
	request("POST", "abort", 204)
	request("POST", "rematch", 404)
	if db.submits != 0 || db.aborts != 0 {
		t.Fatal("the volley endpoint mutated a basket match")
	}
	db.match.GameID = "tap-volley"
	request("GET", "result?match_id=match", 200)
	request("POST", "submit", 200)
	if db.submits != 1 {
		t.Fatal("volley score was not accepted")
	}
}
