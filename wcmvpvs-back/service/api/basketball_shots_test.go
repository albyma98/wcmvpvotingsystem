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

type basketballAPIDB struct {
	database.AppDatabase
	err    error
	calls  int
	fanOrg int
	fanID  int
	device string
	dx, dy float64
}

func (db *basketballAPIDB) GetFanBySessionToken(token, device string) (database.FanProfileSummary, error) {
	if token != "valid" {
		return database.FanProfileSummary{}, sql.ErrNoRows
	}
	return database.FanProfileSummary{Profile: database.FanProfile{ID: 9, OrganizationID: db.fanOrg}}, nil
}
func (db *basketballAPIDB) CreateBasketballShot(event, org, fan int, device string, now time.Time) (database.BasketballShot, error) {
	db.calls++
	db.fanID = fan
	db.device = device
	if event != 42 || org != 7 {
		return database.BasketballShot{}, sql.ErrNoRows
	}
	return database.BasketballShot{ShotID: "12345678-1234-1234-1234-123456789012", Hoop: database.BasketballHoop{X: 180, Y: 200}}, db.err
}
func (db *basketballAPIDB) ResolveBasketballShot(event, org, fan int, device, shot string, dx, dy float64, now time.Time) (database.BasketballShotResult, error) {
	db.calls++
	db.fanID = fan
	db.device = device
	db.dx = dx
	db.dy = dy
	return database.BasketballShotResult{ShotID: shot, Made: true, Wallet: 12, CoinsEarned: 1, BasketballEarned: 4}, db.err
}

func TestBasketballAPIContractAndScope(t *testing.T) {
	db := &basketballAPIDB{fanOrg: 7}
	router := &_router{db: db, coinsHub: newSSEHub()}
	ctx := reqcontext.RequestContext{OrganizationID: 7, Logger: logrus.New()}
	mux := chi.NewRouter()
	mux.Post("/{eventId}/shots", func(w http.ResponseWriter, r *http.Request) { router.createBasketballShot(w, r, ctx) })
	mux.Post("/{eventId}/shots/{shotId}", func(w http.ResponseWriter, r *http.Request) { router.submitBasketballShot(w, r, ctx) })
	request := func(path, body, device, token string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		r.Header.Set("X-Device-ID", device)
		r.Header.Set("X-Fan-Session", token)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	if response := request("/42/shots", "", "", ""); response.Code != 400 || db.calls != 0 {
		t.Fatalf("missing device %d", response.Code)
	}
	if response := request("/42/shots", "", "device", "invalid"); response.Code != 401 || db.calls != 0 {
		t.Fatalf("invalid session %d", response.Code)
	}
	db.fanOrg = 8
	if response := request("/42/shots", "", "device", "valid"); response.Code != 403 || db.calls != 0 {
		t.Fatalf("cross-org session %d", response.Code)
	}
	db.fanOrg = 7
	response := request("/42/shots", "", "device", "valid")
	if response.Code != 200 || db.fanID != 9 || db.device != "device" || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("issued %d %s", response.Code, response.Body.String())
	}
	path := "/42/shots/12345678-1234-1234-1234-123456789012"
	for _, body := range []string{`{"dx":0}`, `{"dx":null,"dy":-266}`, `{"dx":0,"dy":-266,"wallet":999}`, `{"dx":0,"dy":-266} {}`} {
		if response := request(path, body, "device", ""); response.Code != 400 {
			t.Fatalf("bad payload %s: %d", body, response.Code)
		}
	}
	response = request(path, `{"dx":0,"dy":-266}`, "device", "")
	if response.Code != 200 || db.fanID != 0 || db.dx != 0 || db.dy != -266 {
		t.Fatalf("submitted %d %s", response.Code, response.Body.String())
	}
	var result database.BasketballShotResult
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil || result.BasketballEarned != 4 || result.CoinsEarned != 1 {
		t.Fatalf("result %+v: %v", result, err)
	}
	for _, tc := range []struct {
		err    error
		status int
	}{{database.ErrBasketballClosed, 409}, {database.ErrBasketballTooEarly, 409}, {database.ErrBasketballExpired, 410}, {database.ErrBasketballGesture, 400}, {sql.ErrNoRows, 404}} {
		db.err = tc.err
		if response := request(path, `{"dx":0,"dy":-266}`, "device", ""); response.Code != tc.status {
			t.Fatalf("error %v: %d", tc.err, response.Code)
		}
	}
}

func TestBasketballLeaderboardUpdatesCoalesceAndDeliverTrailing(t *testing.T) {
	hub := newSSEHub()
	updates, unsubscribe := hub.Subscribe(42)
	defer unsubscribe()
	other, unsubscribeOther := hub.Subscribe(43)
	defer unsubscribeOther()
	hub.BroadcastThrottled(42, 80*time.Millisecond)
	select {
	case <-updates:
	default:
		t.Fatal("first update must be immediate")
	}
	for n := 0; n < 20; n++ {
		hub.BroadcastThrottled(42, 80*time.Millisecond)
	}
	select {
	case <-updates:
		t.Fatal("burst must be coalesced")
	default:
	}
	hub.BroadcastThrottled(43, 80*time.Millisecond)
	select {
	case <-other:
	default:
		t.Fatal("events must have separate windows")
	}
	select {
	case <-updates:
	case <-time.After(time.Second):
		t.Fatal("final update must be delivered without another basket")
	}
	select {
	case <-updates:
		t.Fatal("one trailing update expected")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestBasketballImmediateLeaderboardUpdateCancelsOlderTimer(t *testing.T) {
	hub := newSSEHub()
	updates, unsubscribe := hub.Subscribe(42)
	defer unsubscribe()
	hub.BroadcastThrottled(42, 80*time.Millisecond)
	<-updates
	hub.BroadcastThrottled(42, 80*time.Millisecond)
	// Simulate an older trailing callback waiting for the hub lock after its
	// deadline. A newer immediate update must invalidate that callback.
	hub.mu.Lock()
	state := hub.throttled[42]
	previousGeneration := state.generation
	state.last = time.Now().Add(-time.Second)
	hub.mu.Unlock()
	hub.BroadcastThrottled(42, 80*time.Millisecond)
	select {
	case <-updates:
	default:
		t.Fatal("new immediate update expected")
	}
	hub.mu.Lock()
	invalidated := state.timer == nil && state.generation != previousGeneration
	hub.mu.Unlock()
	if !invalidated {
		t.Fatal("older callback must be invalidated even if timer.Stop loses a race")
	}
	select {
	case <-updates:
		t.Fatal("cancelled timer must not send a duplicate")
	case <-time.After(150 * time.Millisecond):
	}
}
