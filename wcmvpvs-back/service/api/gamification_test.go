package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

func TestGamificationMasterAndPublicRestrictions(t *testing.T) {
	conn, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "gamification.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	db, err := database.New(conn)
	if err != nil {
		t.Fatal(err)
	}
	club, err := db.CreateOrganization(database.Organization{Name: "Volley", Slug: "volley", Sport: "Pallavolo", IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	other, err := db.CreateOrganization(database.Organization{Name: "Basket", Slug: "basket", Sport: "Pallacanestro", IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	event, err := db.CreateEvent(database.Event{OrganizationID: club.ID, Team1ID: club.TeamID, Team2ID: other.TeamID, StartDateTime: "2026-10-25T18:00:00Z", IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SetActiveEvent(event, club.ID); err != nil {
		t.Fatal(err)
	}
	appRouter, err := New(Config{Database: db, Logger: logrus.New()})
	if err != nil {
		t.Fatal(err)
	}
	defer appRouter.Close()
	router := appRouter.(*_router)
	ctx := reqcontext.RequestContext{AdminRole: "superadmin", Logger: logrus.New()}
	master := chi.NewRouter()
	master.Get("/", func(w http.ResponseWriter, r *http.Request) { router.listMasterMinigames(w, r, ctx) })
	master.Put("/{gameId}", func(w http.ResponseWriter, r *http.Request) { router.updateMasterMinigame(w, r, ctx) })
	request := func(handler http.Handler, method, path, body, slug string, status int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("X-Organization-Slug", slug)
		r.Header.Set("X-Device-ID", "test-device")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != status {
			t.Fatalf("%s %s: %d want %d: %s", method, path, w.Code, status, w.Body.String())
		}
		return w
	}
	ctx.AdminRole = "admin"
	request(master, "GET", "/", "", "", 403)
	request(master, "PUT", "/basketball", `{"sports":["Tutti"]}`, "", 403)
	ctx.AdminRole = "superadmin"
	request(master, "PUT", "/basketball", `{}`, "", 400)
	request(master, "PUT", "/basketball", `{"sports":null}`, "", 400)
	request(master, "PUT", "/basketball", `{"sports":["Tutti","Calcio"]}`, "", 400)
	request(master, "PUT", "/basketball", `{"sports":["Tennis"]}`, "", 400)
	request(master, "PUT", "/unknown", `{"sports":["Tutti"]}`, "", 404)
	for _, game := range []string{"basketball", "playmaker", "crossover", "reaction", "quiz", "tap", "tap-volley", "sponsor-rush", "memory-flash", "free_throw_challenge", "stack_it", "libero-reflex", "volley-rush", "setter-pro", "monster-block"} {
		request(master, "PUT", "/"+game, `{"sports":["Pallacanestro"]}`, "", 200)
	}
	list := request(master, "GET", "/", "", "", 200)
	var games []database.Minigame
	if err := json.Unmarshal(list.Body.Bytes(), &games); err != nil || len(games) != 15 {
		t.Fatalf("master catalog: %+v %v", games, err)
	}
	public := router.Handler()
	path := "/events/" + strconv.Itoa(event)
	request(public, "GET", path+"/minigames", "", "basket", 404)
	available := request(public, "GET", path+"/minigames", "", "volley", 200)
	if strings.TrimSpace(available.Body.String()) != "[]" {
		t.Fatalf("all games should be restricted: %s", available.Body.String())
	}
	for _, endpoint := range []struct{ method, suffix string }{
		{"GET", "/reaction-test"}, {"POST", "/reaction-test"},
		{"POST", "/basketball/shots"}, {"POST", "/basketball/shots/1"},
		{"POST", "/playmaker/sessions"}, {"POST", "/playmaker/sessions/1/passes"},
		{"POST", "/crossover/sessions"}, {"POST", "/crossover/sessions/1/duels"},
		{"POST", "/tap-live/queue"}, {"POST", "/tap-live/submit"}, {"POST", "/tap-live/rematch"}, {"GET", "/tap-live/stream"},
		{"POST", "/tap-volley-live/queue"}, {"POST", "/tap-volley-live/submit"}, {"POST", "/tap-volley-live/rematch"}, {"GET", "/tap-volley-live/stream"},
	} {
		request(public, endpoint.method, path+endpoint.suffix, `{}`, "volley", 403)
	}
	request(public, "GET", "/public"+path+"/quiz", "", "volley", 403)
	request(public, "POST", "/public"+path+"/quiz/answer", `{}`, "volley", 403)
	for _, game := range []string{"volley-rush", "setter-pro", "monster-block"} {
		request(public, "POST", path+"/volleyball/"+game+"/rounds", `{}`, "volley", 403)
		request(public, "POST", path+"/volleyball/"+game+"/rounds/test-round", `{}`, "volley", 403)
		request(master, "PUT", "/"+game, `{"sports":["Pallavolo"]}`, "", 200)
		request(public, "POST", path+"/volleyball/"+game+"/rounds", `{}`, "volley", 200)
		request(master, "PUT", "/"+game, `{"sports":[]}`, "", 200)
	}
	request(public, "POST", path+"/volleyball/unknown/rounds", `{}`, "volley", 404)
	request(public, "POST", path+"/journey-rewards", `{"key":"reaction","request_id":"round-123"}`, "volley", 403)
	request(public, "POST", path+"/journey-rewards", `{"key":"tap-volley","request_id":"volley-round-123"}`, "volley", 403)
	for _, gameType := range []string{"tap_challenge", "memory_flash", "sponsor_rush", "free_throw_challenge", "stack_it"} {
		if _, err := conn.Exec(`UPDATE events SET show_branded_game=1, branded_game_config=? WHERE id=?`, `{"game_type":"`+gameType+`"}`, event); err != nil {
			t.Fatal(err)
		}
		request(public, "GET", path+"/branded-game", "", "volley", 403)
		request(public, "POST", path+"/branded-game/result", `{}`, "volley", 403)
	}
	live := request(public, "GET", path+"/fan-live", "", "volley", 200)
	if !strings.Contains(live.Body.String(), `"available_minigames":[]`) {
		t.Fatalf("live catalog: %s", live.Body.String())
	}
	request(master, "PUT", "/basketball", `{"sports":["Tutti"]}`, "", 200)
	available = request(public, "GET", path+"/minigames", "", "volley", 200)
	if !strings.Contains(available.Body.String(), `"basketball"`) {
		t.Fatalf("Tutti did not re-enable the game: %s", available.Body.String())
	}
	request(master, "PUT", "/basketball", `{"sports":[]}`, "", 200)
	request(public, "POST", path+"/basketball/shots", `{}`, "volley", 403)
	request(master, "PUT", "/tap-volley", `{"sports":["Pallavolo"]}`, "", 200)
	credit := request(public, "POST", path+"/journey-rewards", `{"key":"tap-volley","request_id":"volley-round-123"}`, "volley", 200)
	if !strings.Contains(credit.Body.String(), `"coins_earned":8`) {
		t.Fatalf("volley reward: %s", credit.Body.String())
	}
	retry := request(public, "POST", path+"/journey-rewards", `{"key":"tap-volley","request_id":"volley-round-123"}`, "volley", 200)
	if !strings.Contains(retry.Body.String(), `"coins_earned":0`) {
		t.Fatalf("duplicate volley reward: %s", retry.Body.String())
	}
	request(public, "POST", path+"/tap-volley-live/queue", `{}`, "volley", 401)
	request(public, "POST", path+"/tap-live/queue", `{}`, "volley", 403)
}
