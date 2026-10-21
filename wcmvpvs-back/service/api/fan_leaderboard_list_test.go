package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

type fanLeaderboardListAPIDB struct {
	fanEconomyAPIDB
	rows              []database.FanLeaderboardEntry
	event, org, limit int
}

func (db *fanLeaderboardListAPIDB) GetFanLeaderboard(event, org, limit int) ([]database.FanLeaderboardEntry, error) {
	db.event, db.org, db.limit = event, org, limit
	return db.rows, db.err
}

func TestFanLeaderboardListIncludesEveryFanAndMarksOnlyTheAuthenticatedProfile(t *testing.T) {
	for _, tc := range []struct {
		name    string
		token   string
		fanOrg  int
		pending bool
		self    int
	}{
		{"registered", "valid", 7, false, 1},
		{"guest", "", 7, false, 0},
		{"invalid session", "invalid", 7, false, 0},
		{"other club", "valid", 8, false, 0},
		{"pending registration", "valid", 7, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := &fanLeaderboardListAPIDB{
				fanEconomyAPIDB: fanEconomyAPIDB{basketballAPIDB: basketballAPIDB{fanOrg: tc.fanOrg}, registrationPending: tc.pending},
				rows: []database.FanLeaderboardEntry{
					{FanID: 1, Rank: 1, Nickname: "Omonimo", Coins: 100},
					{FanID: 2, Rank: 2, Nickname: "Secondo", Coins: 90},
					{FanID: 3, Rank: 3, Nickname: "Terzo", Coins: 80},
					{FanID: 9, Rank: 4, Nickname: "Omonimo", Coins: 70, EquippedFrame: "club_frame"},
					{FanID: 5, Rank: 5, Nickname: "Quinto", Coins: 60},
				},
			}
			router := &_router{db: db}
			mux := chi.NewRouter()
			mux.Get("/{eventId}", func(w http.ResponseWriter, r *http.Request) {
				router.getCoinsLeaderboard(w, r, reqcontext.RequestContext{OrganizationID: 7, Logger: logrus.New()})
			})
			r := httptest.NewRequest(http.MethodGet, "/42?view=list", nil)
			r.Header.Set("X-Device-ID", "device")
			r.Header.Set("X-Fan-Session", tc.token)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
			var data struct {
				Ranking []struct {
					Rank          int    `json:"rank"`
					Points        int    `json:"points"`
					IsMe          bool   `json:"is_me"`
					EquippedFrame string `json:"equipped_frame"`
				} `json:"ranking"`
				Top3     []interface{} `json:"top3"`
				UserRank *struct {
					Rank int `json:"rank"`
				} `json:"userRank"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
				t.Fatal(err)
			}
			if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" || db.event != 42 || db.org != 7 || db.limit != -1 || len(data.Ranking) != 5 || len(data.Top3) != 3 {
				t.Fatalf("full club ranking: status=%d query=%d/%d/%d body=%s", w.Code, db.event, db.org, db.limit, w.Body.String())
			}
			self := 0
			for index, row := range data.Ranking {
				if row.Rank != index+1 {
					t.Fatalf("ranking order: %+v", data.Ranking)
				}
				if row.IsMe {
					self++
					if row.Rank != 4 || row.Points != 70 || row.EquippedFrame != "club_frame" {
						t.Fatalf("wrong current fan: %+v", row)
					}
				}
			}
			if self != tc.self || (tc.self == 1 && (data.UserRank == nil || data.UserRank.Rank != 4)) || (tc.self == 0 && data.UserRank != nil) {
				t.Fatalf("current fan repeated or incorrectly identified: %s", w.Body.String())
			}
		})
	}
}
