package api

import (
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

type fanEconomyAPIDB struct {
	basketballAPIDB
	purchased, equipped string
	mutations           int
	registerErr         error
}

func (db *fanEconomyAPIDB) GetFanWalletState(event, org, fan int, device string) (database.FanWalletState, error) {
	db.calls++
	db.fanID = fan
	db.device = device
	return database.FanWalletState{Wallet: 50, Points: 100, Movements: []database.FanWalletMovement{{ID: 1, Delta: 10, Balance: 50, Label: "Canestro", CreatedAt: "2026-10-16T18:00:00Z"}}, Catalog: []database.FanShopCatalogItem{{Key: "club_frame", Title: "Cornice", Cost: 20, Kind: "frame", Owned: true}}, Collection: database.FanCollection{EquippedFrame: "club_frame", Items: []database.FanCollectionItem{}, Badges: []database.FanCollectionBadge{}, MatchesFollowed: 1}}, db.err
}
func (db *fanEconomyAPIDB) PurchaseFanShopItem(event, org, fan int, device, key string) error {
	db.purchased = key
	db.mutations++
	return db.err
}
func (db *fanEconomyAPIDB) EquipFanShopItem(event, org, fan int, device, key string) error {
	db.equipped = key
	db.mutations++
	return db.err
}
func (db *fanEconomyAPIDB) RegisterFan(database.FanRegisterInput) (database.FanProfileSummary, error) {
	return database.FanProfileSummary{}, db.registerErr
}
func (db *fanEconomyAPIDB) GetFanLeaderboard(int, int, int) ([]database.FanLeaderboardEntry, error) {
	return []database.FanLeaderboardEntry{{FanID: 1, Nickname: "Leader", Coins: 100}}, nil
}
func (db *fanEconomyAPIDB) GetFanRank(int, int, int) (database.FanLeaderboardEntry, error) {
	return database.FanLeaderboardEntry{FanID: 9, Rank: 2, Coins: 40}, nil
}
func (db *fanEconomyAPIDB) GetFanNearbyRanks(int, int) ([]database.FanLeaderboardEntry, error) {
	return []database.FanLeaderboardEntry{{FanID: 1, Nickname: "Leader", Rank: 1, Coins: 100}, {FanID: 9, Nickname: "Me", Rank: 2, Coins: 40}}, nil
}

func TestFanEconomyAPIContractAndServerOwnedPurchase(t *testing.T) {
	db := &fanEconomyAPIDB{basketballAPIDB: basketballAPIDB{fanOrg: 7}}
	router := &_router{db: db}
	ctx := reqcontext.RequestContext{OrganizationID: 7, Logger: logrus.New()}
	mux := chi.NewRouter()
	mux.Get("/{eventId}/wallet", func(w http.ResponseWriter, r *http.Request) { router.getFanWallet(w, r, ctx) })
	mux.Post("/{eventId}/purchase", func(w http.ResponseWriter, r *http.Request) { router.purchaseFanShopItem(w, r, ctx) })
	mux.Post("/{eventId}/equip", func(w http.ResponseWriter, r *http.Request) { router.equipFanShopItem(w, r, ctx) })
	mux.Post("/{eventId}/guest-coins", func(w http.ResponseWriter, r *http.Request) { router.postGuestCoins(w, r, ctx) })
	mux.Post("/{eventId}/redeem", func(w http.ResponseWriter, r *http.Request) { router.postFanRewardRedeem(w, r, ctx) })
	request := func(method, path, body, token string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("X-Device-ID", "device")
		r.Header.Set("X-Fan-Session", token)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	response := request(http.MethodGet, "/42/wallet", "", "")
	if response.Code != 200 || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("wallet %d %s", response.Code, response.Body.String())
	}
	var data map[string]interface{}
	if err := json.Unmarshal(response.Body.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	collection, ok := data["collection"].(map[string]interface{})
	if !ok || data["wallet"] != float64(50) || data["points"] != float64(100) || collection["equipped_frame"] != "club_frame" || collection["matches_followed"] != float64(1) {
		t.Fatalf("contract %+v", data)
	}
	for _, body := range []string{`{}`, `{"item_key":null}`, `{"item_key":"club_frame","cost":0}`, `{"item_key":"club_frame","wallet":999}`, `{"item_key":"club_frame"} {}`} {
		before := db.mutations
		if response := request(http.MethodPost, "/42/purchase", body, ""); response.Code != 400 || db.mutations != before {
			t.Fatalf("client controlled price/balance %s: %d", body, response.Code)
		}
	}
	response = request(http.MethodPost, "/42/purchase", `{"item_key":"club_frame"}`, "valid")
	if response.Code != 200 || db.purchased != "club_frame" || db.fanID != 9 {
		t.Fatalf("purchase %d %s", response.Code, response.Body.String())
	}
	response = request(http.MethodPost, "/42/equip", `{"item_key":""}`, "valid")
	if response.Code != 200 || db.equipped != "" {
		t.Fatalf("unequip %d", response.Code)
	}
	before := db.mutations
	response = request(http.MethodPost, "/42/guest-coins", `{"coins":999999}`, "")
	if response.Code != 200 || db.mutations != before || !strings.Contains(response.Body.String(), `"wallet":50`) {
		t.Fatalf("absolute client wallet must not write: %d %s", response.Code, response.Body.String())
	}
	response = request(http.MethodPost, "/42/redeem", `{"reward_key":"club_frame","cost_coins":0}`, "valid")
	if response.Code != 200 || db.purchased != "club_frame" {
		t.Fatalf("legacy redeem must use server catalog %d", response.Code)
	}
	db.fanOrg = 8
	if response := request(http.MethodGet, "/42/wallet", "", "valid"); response.Code != 403 {
		t.Fatalf("cross-org session %d", response.Code)
	}
	db.fanOrg = 7
	for _, tc := range []struct {
		err    error
		status int
	}{{database.ErrFanShopFunds, 409}, {database.ErrFanShopNotOwned, 409}, {database.ErrFanShopItem, 400}} {
		db.err = tc.err
		if response := request(http.MethodPost, "/42/purchase", `{"item_key":"club_frame"}`, ""); response.Code != tc.status {
			t.Fatalf("error %v: %d", tc.err, response.Code)
		}
	}
}

func TestFanEconomyExistingPhoneRequiresOTPLogin(t *testing.T) {
	db := &fanEconomyAPIDB{registerErr: database.ErrFanLoginRequired}
	router := &_router{db: db}
	r := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"phone":"+390004","nickname":"Other"}`))
	w := httptest.NewRecorder()
	router.postRegisterFan(w, r, reqcontext.RequestContext{OrganizationID: 7, Logger: logrus.New()})
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "codice") {
		t.Fatalf("existing phone auth %d %s", w.Code, w.Body.String())
	}
}

func TestFanEconomyLeaderboardExposesPointsWithCompatibleCoins(t *testing.T) {
	db := &fanEconomyAPIDB{basketballAPIDB: basketballAPIDB{fanOrg: 7}}
	router := &_router{db: db}
	mux := chi.NewRouter()
	mux.Get("/{eventId}", func(w http.ResponseWriter, r *http.Request) {
		router.getCoinsLeaderboard(w, r, reqcontext.RequestContext{OrganizationID: 7, Logger: logrus.New()})
	})
	r := httptest.NewRequest(http.MethodGet, "/42", nil)
	r.Header.Set("X-Device-ID", "device")
	r.Header.Set("X-Fan-Session", "valid")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	var data map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	top := data["top3"].([]interface{})[0].(map[string]interface{})
	user := data["userRank"].(map[string]interface{})
	nearby := data["nearby"].([]interface{})[1].(map[string]interface{})
	if top["points"] != top["coins"] || top["points"] != float64(100) || user["points"] != float64(40) || nearby["points"] != float64(40) {
		t.Fatalf("score aliases %+v", data)
	}
}
