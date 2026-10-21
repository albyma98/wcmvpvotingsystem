package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

type fanMinigameAPIDB struct {
	basketballAPIDB
	round, key string
	seen       map[string]bool
}

func (db *fanMinigameAPIDB) GetEventOrganizationID(event int) (int, error) {
	if event != 42 {
		return 0, sql.ErrNoRows
	}
	return 7, nil
}
func (db *fanMinigameAPIDB) ClaimFanMinigameReward(event, org, fan int, device, key, id string) (database.FanMinigameReward, error) {
	db.calls++
	db.fanID = fan
	db.device = device
	db.key = key
	db.round = id
	amount := database.FanJourneyRewardAmount(key)
	coins := amount
	if db.seen[id] {
		coins = 0
	}
	db.seen[id] = true
	return database.FanMinigameReward{Wallet: len(db.seen) * amount, CoinsEarned: coins, RewardAmount: amount, RewardTotal: len(db.seen) * amount}, db.err
}

func TestFanMinigameAPIRepeatedRoundsAndScope(t *testing.T) {
	db := &fanMinigameAPIDB{basketballAPIDB: basketballAPIDB{fanOrg: 7}, seen: map[string]bool{}}
	router := &_router{db: db, coinsHub: newSSEHub()}
	mux := chi.NewRouter()
	mux.Post("/{eventId}", func(w http.ResponseWriter, r *http.Request) {
		router.claimFanJourneyReward(w, r, reqcontext.RequestContext{OrganizationID: 7, Logger: logrus.New()})
	})
	request := func(body, token string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, "/42", strings.NewReader(body))
		r.Header.Set("X-Device-ID", "device")
		r.Header.Set("X-Fan-Session", token)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	for _, body := range []string{`{"key":"vote","request_id":"round-123"}`, `{"key":"tap","request_id":"bad"}`, `{"key":"unknown","request_id":"round-123"}`} {
		if w := request(body, ""); w.Code != 400 || db.calls != 0 {
			t.Fatalf("invalid round: %d %s", w.Code, w.Body.String())
		}
	}
	for _, token := range []string{"invalid", "valid"} {
		db.fanOrg = 8
		want := 401
		if token == "valid" {
			want = 403
		}
		if w := request(`{"key":"tap","request_id":"round-123"}`, token); w.Code != want || db.calls != 0 {
			t.Fatalf("auth scope: %d", w.Code)
		}
	}
	db.fanOrg = 7
	for i, id := range []string{"tap-round-1", "tap-round-1", "tap-round-2"} {
		w := request(`{"key":"tap","request_id":"`+id+`","coins":999999}`, "valid")
		if w.Code != 200 || db.fanID != 9 || db.device != "device" || db.round != id {
			t.Fatalf("claim %d %s", w.Code, w.Body.String())
		}
		var result database.FanMinigameReward
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		want := 8
		if i == 1 {
			want = 0
		}
		if result.CoinsEarned != want || result.RewardAmount != 8 {
			t.Fatalf("server amount %+v", result)
		}
		if i == 2 && result.RewardTotal != 16 {
			t.Fatalf("repeat total %+v", result)
		}
	}
}
