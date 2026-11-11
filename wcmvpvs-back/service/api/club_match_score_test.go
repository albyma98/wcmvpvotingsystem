package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

func TestClubMatchLiveCalendar(t *testing.T) {
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "club-live.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	store := NewStore(db)
	rt := &_router{store: store, router: chi.NewRouter()}
	registerClubRoutes(rt)
	siteID, err := store.CreateClubSite(ctx, ClubSite{OrganizationID: 1, Slug: "aurora", Name: "Aurora", Active: true}, []string{"Prima squadra"}, "club-aurora", "hash")
	if err != nil {
		t.Fatal(err)
	}
	admin, err := store.ClubAdminByUsername(ctx, "club-aurora")
	if err != nil {
		t.Fatal(err)
	}
	token, err := store.CreateClubSession(ctx, admin.ID, siteID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	teams, err := store.ListClubTeams(ctx, siteID, true)
	if err != nil || len(teams) != 1 {
		t.Fatalf("teams: %v (%v)", teams, err)
	}
	matchID, err := store.SaveClubMatch(ctx, siteID, ClubMatch{TeamID: teams[0].ID, Opponent: "Ospiti", ScheduledAt: "2026-11-11T20:00"})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(rt.router)
	defer server.Close()
	connect := func() *websocket.Conn {
		t.Helper()
		url := "ws" + strings.TrimPrefix(server.URL, "http") + "/v1/clubs/aurora/matches/stream"
		conn, _, err := websocket.DefaultDialer.Dial(url, http.Header{"Origin": {server.URL}})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { conn.Close() })
		return conn
	}
	read := func(conn *websocket.Conn) clubMatchMessage {
		t.Helper()
		_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		var update clubMatchMessage
		if err := conn.ReadJSON(&update); err != nil {
			t.Fatal(err)
		}
		return update
	}
	socket := connect()
	defer socket.Close()
	initial := read(socket)
	if initial.Matches == nil || len(*initial.Matches) != 1 || (*initial.Matches)[0].Status != "scheduled" {
		t.Fatalf("expected scheduled match in snapshot: %+v", initial)
	}
	score := func(action string, revision int64, authenticated bool, want int) ClubMatch {
		t.Helper()
		body := fmt.Sprintf(`{"action":%q,"revision":%d,"bestOf":3}`, action, revision)
		req := httptest.NewRequest("POST", fmt.Sprintf("/v1/club-admin/aurora/matches/%d/score", matchID), strings.NewReader(body))
		if authenticated {
			req.AddCookie(&http.Cookie{Name: clubCookieName, Value: token})
		}
		res := httptest.NewRecorder()
		rt.router.ServeHTTP(res, req)
		if res.Code != want {
			t.Fatalf("%s: got %d, want %d: %s", action, res.Code, want, res.Body)
		}
		var match ClubMatch
		if want == http.StatusOK {
			if err := json.Unmarshal(res.Body.Bytes(), &match); err != nil {
				t.Fatal(err)
			}
			update := read(socket)
			if update.Match == nil || *update.Match != match {
				t.Fatalf("published match %+v differs from saved match %+v", update.Match, match)
			}
		}
		return match
	}
	score("start", 0, false, http.StatusUnauthorized)
	match := score("start", 0, true, http.StatusOK)
	if match.Status != "live" || match.BestOf != 3 {
		t.Fatalf("start: %+v", match)
	}
	score("close_set", match.Revision, true, http.StatusUnprocessableEntity)
	match = score("point_against", match.Revision, true, http.StatusOK)
	if match.PointsAgainst != 1 {
		t.Fatalf("point: %+v", match)
	}
	score("point_against", match.Revision-1, true, http.StatusConflict)
	match = score("undo_against", match.Revision, true, http.StatusOK)
	for set := 0; set < 2; set++ {
		for point := 0; point < 25; point++ {
			match = score("point_for", match.Revision, true, http.StatusOK)
		}
		match = score("close_set", match.Revision, true, http.StatusOK)
	}
	match = score("reopen_set", match.Revision, true, http.StatusOK)
	if match.ScoreFor != 1 || match.PointsFor != 25 {
		t.Fatalf("reopened set: %+v", match)
	}
	match = score("close_set", match.Revision, true, http.StatusOK)
	match = score("finish", match.Revision, true, http.StatusOK)
	if match.Status != "completed" || match.ScoreFor != 2 || match.SetsJSON != `["25-0","25-0"]` {
		t.Fatalf("final result: %+v", match)
	}
	score("point_for", match.Revision, true, http.StatusUnprocessableEntity)
	snapshot := read(connect())
	if snapshot.Matches == nil || len(*snapshot.Matches) != 1 || (*snapshot.Matches)[0] != match {
		t.Fatalf("reconnect missed final result: %+v", snapshot)
	}
	response := httptest.NewRecorder()
	rt.router.ServeHTTP(response, httptest.NewRequest("GET", "/v1/clubs/aurora/home", nil))
	var calendar struct{ Matches []ClubMatch }
	if err := json.Unmarshal(response.Body.Bytes(), &calendar); err != nil || len(calendar.Matches) != 1 || calendar.Matches[0] != match {
		t.Fatalf("calendar missed final result: %s (%v)", response.Body, err)
	}
}

func TestClubMatchDecidingSetRequiresTwoPointLead(t *testing.T) {
	match := ClubMatch{Status: "live", BestOf: 5, ScoreFor: 2, ScoreAgainst: 2, PointsFor: 15, PointsAgainst: 14, SetsJSON: `["25-20","20-25","25-20","20-25"]`}
	if err := updateClubScore(&match, clubScoreAction{Action: "close_set"}); err != errClubMatchAction {
		t.Fatalf("closed deciding set at 15-14: %v", err)
	}
	if err := updateClubScore(&match, clubScoreAction{Action: "point_for"}); err != nil {
		t.Fatal(err)
	}
	if err := updateClubScore(&match, clubScoreAction{Action: "close_set"}); err != nil || match.ScoreFor != 3 || match.PointsFor != 0 {
		t.Fatalf("deciding set: %+v (%v)", match, err)
	}
}
