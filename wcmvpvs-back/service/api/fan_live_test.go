package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

type fanLiveDB struct {
	database.AppDatabase
	event       database.Event
	quiz        database.EventQuizConfig
	resultsRead int
}

func (db *fanLiveDB) GetFanLiveEvent(id, org int) (database.Event, error) {
	if id != db.event.ID || org != db.event.OrganizationID {
		return database.Event{}, sql.ErrNoRows
	}
	return db.event, nil
}
func (db *fanLiveDB) GetEventQuizConfig(int) (database.EventQuizConfig, error) { return db.quiz, nil }
func (db *fanLiveDB) GetOrganizationMinigames(int) ([]string, error) {
	return []string{"quiz", "reaction"}, nil
}
func (db *fanLiveDB) ListEventQuizQuestions(int) ([]database.EventQuizQuestion, error) {
	return []database.EventQuizQuestion{{ID: 1}}, nil
}
func (db *fanLiveDB) GetEventVoteLeaderboard(int, int) ([]database.EventVoteLeaderboardEntry, error) {
	db.resultsRead++
	return []database.EventVoteLeaderboardEntry{{PlayerID: 1, Votes: 10}, {PlayerID: 2, Votes: 10}, {PlayerID: 3, Votes: 5}}, nil
}

func TestFanLiveStatus(t *testing.T) {
	db := &fanLiveDB{event: database.Event{ID: 42, OrganizationID: 7, IsActive: true}, quiz: database.EventQuizConfig{Enabled: true}}
	router := &_router{db: db}
	ctx := reqcontext.RequestContext{OrganizationID: 7, Logger: logrus.New()}
	mux := chi.NewRouter()
	mux.Get("/{eventId}", func(w http.ResponseWriter, r *http.Request) { router.getFanLiveStatus(w, r, ctx) })
	read := func() map[string]interface{} {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/42", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("status %d", response.Code)
		}
		var data map[string]interface{}
		if err := json.Unmarshal(response.Body.Bytes(), &data); err != nil {
			t.Fatal(err)
		}
		return data
	}
	data := read()
	if data["quiz"].(map[string]interface{})["available"] != true || db.resultsRead != 0 {
		t.Fatalf("active quiz/results: %+v", data)
	}
	db.quiz.ActiveFrom = time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	data = read()
	if data["quiz"].(map[string]interface{})["available"] != false || data["quiz"].(map[string]interface{})["starts_at"] == "" {
		t.Fatalf("future quiz: %+v", data)
	}
	db.event.IsConcluded = true
	data = read()
	if len(data["mvp"].([]interface{})) != 2 || data["quiz"].(map[string]interface{})["available"] != false {
		t.Fatalf("concluded/tied MVP: %+v", data)
	}
	ctx.OrganizationID = 8
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/42", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("cross-club status %d", response.Code)
	}
}

func TestFanRankGapUsesActualTieBreaking(t *testing.T) {
	older := database.FanLeaderboardEntry{FanID: 1, Coins: 42}
	newer := database.FanLeaderboardEntry{FanID: 2, Coins: 50}
	if gap := fanRankGap(older, newer); gap != 8 {
		t.Fatalf("older fan can overtake on equal coins: %d", gap)
	}
	older.Coins, newer.Coins = 50, 42
	if gap := fanRankGap(newer, older); gap != 9 {
		t.Fatalf("newer fan needs one more coin: %d", gap)
	}
}
