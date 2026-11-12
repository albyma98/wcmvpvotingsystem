package api

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

func TestClubMVPLightLifecycle(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite3", "file:"+filepath.ToSlash(filepath.Join(t.TempDir(), "club-mvp.db"))+"?_foreign_keys=on&_busy_timeout=5000&_journal_mode=WAL")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	store := NewStore(db)
	rt := &_router{
		store: store, router: chi.NewRouter(),
		voteRateByDevice: map[string][]time.Time{}, voteRateByIP: map[string][]time.Time{},
	}
	registerClubRoutes(rt)
	if err := store.EnsureClubTables(); err != nil {
		t.Fatal(err)
	}
	createClub := func(orgID int64, slug string) (int64, []ClubTeam, *http.Cookie) {
		t.Helper()
		id, err := store.CreateClubSite(ctx, ClubSite{OrganizationID: orgID, Slug: slug, Name: slug, Active: true},
			[]string{"Under 19", "Under 17"}, "admin-"+slug, "hash")
		if err != nil {
			t.Fatal(err)
		}
		teams, err := store.ListClubTeams(ctx, id, true)
		if err != nil || len(teams) != 2 {
			t.Fatalf("divisions: %v, %v", teams, err)
		}
		admin, err := store.ClubAdminByUsername(ctx, "admin-"+slug)
		if err != nil {
			t.Fatal(err)
		}
		token, err := store.CreateClubSession(ctx, admin.ID, id, time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		return id, teams, &http.Cookie{Name: clubCookieName, Value: token}
	}
	siteID, teams, admin := createClub(1, "aurora")
	_, _, otherAdmin := createClub(2, "rivale")
	teamID := teams[0].ID
	createRoster := func(divisionID int64, count int) []ClubPlayer {
		t.Helper()
		items := make([]ClubPlayer, count)
		for i := range items {
			items[i] = ClubPlayer{FirstName: fmt.Sprintf("Atleta %02d", i+1), LastName: "Rossi", JerseyNumber: i + 1, Role: "Centrale"}
		}
		if err := store.ReplaceClubPlayers(ctx, siteID, divisionID, items); err != nil {
			t.Fatal(err)
		}
		roster, err := store.ListClubPlayers(ctx, siteID, divisionID)
		if err != nil || len(roster) != count {
			t.Fatalf("roster: %v, %v", roster, err)
		}
		return roster
	}
	roster := createRoster(teamID, 18)
	otherDivision := createRoster(teams[1].ID, 1)
	createMatch := func(opponent string) int64 {
		t.Helper()
		id, err := store.SaveClubMatch(ctx, siteID, ClubMatch{
			TeamID: teamID, Opponent: opponent, ScheduledAt: "2026-11-14T18:30", Home: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	a, b := createMatch("Ospiti A"), createMatch("Ospiti B")
	adminPath := func(matchID int64) string { return fmt.Sprintf("/v1/club-admin/aurora/matches/%d", matchID) }
	publicPath := func(matchID int64) string { return fmt.Sprintf("/v1/clubs/aurora/matches/%d/mvp", matchID) }
	const qr = "/v1/clubs/aurora/mvp"

	call := func(method, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "https://platform.invalid"+path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if cookie != nil {
			req.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		rt.router.ServeHTTP(w, req)
		return w
	}
	expect := func(status int, method, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
		t.Helper()
		w := call(method, path, body, cookie)
		if w.Code != status {
			t.Fatalf("%s %s: got %d, want %d: %s", method, path, w.Code, status, w.Body)
		}
		return w
	}
	decode := func(w *httptest.ResponseRecorder, out interface{}) {
		t.Helper()
		if err := json.Unmarshal(w.Body.Bytes(), out); err != nil {
			t.Fatal(err)
		}
	}
	readBallot := func(path string, cookie *http.Cookie) (clubMVPPublic, *httptest.ResponseRecorder) {
		t.Helper()
		w := expect(200, "GET", path, "", cookie)
		var ballot clubMVPPublic
		decode(w, &ballot)
		if strings.Contains(w.Body.String(), "totalVotes") || strings.Contains(w.Body.String(), "percentage") {
			t.Fatal("admin results leaked in public response")
		}
		return ballot, w
	}
	readAdmin := func(matchID int64) clubMVPAdmin {
		t.Helper()
		var data clubMVPAdmin
		decode(expect(200, "GET", adminPath(matchID)+"/mvp", "", admin), &data)
		return data
	}
	voteBody := func(id int64) string { return fmt.Sprintf(`{"playerId":%d}`, id) }
	prepare := func(matchID int64, chosen []ClubPlayer) {
		t.Helper()
		ids := make([]int64, 0, len(chosen))
		for _, p := range chosen {
			ids = append(ids, p.ID)
		}
		body, err := json.Marshal(map[string][]int64{"playerIds": ids})
		if err != nil {
			t.Fatal(err)
		}
		expect(200, "PUT", adminPath(matchID)+"/mvp", string(body), admin)
	}

	expect(409, "GET", qr, "", nil)
	expect(401, "GET", adminPath(a)+"/mvp", "", nil)
	expect(403, "GET", adminPath(a)+"/mvp", "", otherAdmin)
	expect(403, "GET", "/v1/club-admin/aurora/mvp/stream", "", otherAdmin)
	expect(403, "PUT", adminPath(a)+"/mvp", `{"enabled":true}`, otherAdmin)
	expect(422, "PUT", adminPath(a)+"/mvp", fmt.Sprintf(`{"playerIds":[%d]}`, otherDivision[0].ID), admin)
	prepare(a, roster[:12])
	prepare(b, roster[12:])
	if data := readAdmin(a); len(data.Roster) != 18 || len(data.Callups) != 12 || data.Open || data.CanOpen {
		t.Fatalf("scheduled ballot: %+v", data)
	}
	expect(409, "PUT", adminPath(a)+"/mvp", `{"enabled":true}`, admin)
	for _, id := range []int64{a, b} {
		expect(200, "POST", adminPath(id)+"/score", `{"action":"start","revision":0,"bestOf":3}`, admin)
	}
	expect(409, "GET", qr, "", nil)
	prepare(b, nil)
	expect(422, "PUT", adminPath(b)+"/mvp", `{"enabled":true}`, admin)
	prepare(b, roster[12:])
	expect(200, "PUT", adminPath(a)+"/mvp", `{"enabled":true}`, admin)
	ballot, w := readBallot(qr, nil)
	if ballot.Match.ID != a || ballot.Team.Name != teams[0].Name || len(ballot.Players) != 12 {
		t.Fatalf("QR ballot: %+v", ballot)
	}
	for i, player := range ballot.Players {
		if player.ID != roster[i].ID {
			t.Fatalf("unexpected eligible player %d", player.ID)
		}
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || !cookies[0].Secure || cookies[0].SameSite != http.SameSiteStrictMode {
		t.Fatalf("voter cookie: %+v", cookies)
	}
	device := cookies[0]
	if err := store.EnsureClubTables(); err != nil {
		t.Fatal(err)
	}
	forged := &http.Cookie{Name: clubMVPCookie, Value: base64.RawURLEncoding.EncodeToString(make([]byte, 64))}
	expect(403, "POST", publicPath(a)+"/vote", voteBody(roster[0].ID), nil)
	expect(403, "POST", publicPath(a)+"/vote", voteBody(roster[0].ID), forged)
	expect(422, "POST", publicPath(a)+"/vote", voteBody(roster[12].ID), device)

	// Before the first vote, even an open match's call-ups can be corrected.
	prepare(a, roster[1:13])
	ballot, _ = readBallot(publicPath(a), device)
	if len(ballot.Players) != 12 || ballot.Players[0].ID != roster[1].ID {
		t.Fatalf("updated ballot: %+v", ballot.Players)
	}
	expect(200, "PUT", adminPath(b)+"/mvp", `{"enabled":true}`, admin)
	expect(409, "GET", qr, "", device)
	expect(200, "PUT", adminPath(b)+"/mvp", `{"qrTarget":true}`, admin)
	ballot, _ = readBallot(qr, device)
	if ballot.Match.ID != b || len(ballot.Players) != 6 || ballot.Players[0].ID != roster[12].ID {
		t.Fatalf("second match call-ups: %+v", ballot)
	}
	expect(200, "PUT", adminPath(a)+"/mvp", `{"qrTarget":true}`, admin)
	if readAdmin(b).QRTarget {
		t.Fatal("previous QR priority was not removed")
	}

	updates, unsubscribe := rt.clubMVPUpdates.Subscribe(int(siteID))
	defer unsubscribe()
	start := make(chan struct{})
	statuses := make(chan int, 2)
	var voters sync.WaitGroup
	for i := 0; i < 2; i++ {
		voters.Add(1)
		go func() {
			defer voters.Done()
			<-start
			statuses <- call("POST", publicPath(a)+"/vote", voteBody(roster[1].ID), device).Code
		}()
	}
	close(start)
	voters.Wait()
	close(statuses)
	counts := map[int]int{}
	for status := range statuses {
		counts[status]++
	}
	if counts[201] != 1 || counts[409] != 1 {
		t.Fatalf("simultaneous vote responses: %v", counts)
	}
	select {
	case <-updates:
	case <-time.After(time.Second):
		t.Fatal("successful vote did not notify the admin stream")
	}
	expect(409, "POST", publicPath(a)+"/vote", voteBody(roster[2].ID), device)
	ballot, _ = readBallot(publicPath(a), device)
	if ballot.Voted == nil || ballot.Voted.ID != roster[1].ID {
		t.Fatalf("existing vote: %+v", ballot.Voted)
	}
	expect(409, "PUT", adminPath(a)+"/mvp", `{"playerIds":[]}`, admin)
	expect(201, "POST", publicPath(b)+"/vote", voteBody(roster[12].ID), device)

	_, w = readBallot(publicPath(a), nil)
	secondDevice := w.Result().Cookies()[0]
	expect(201, "POST", publicPath(a)+"/vote", voteBody(roster[2].ID), secondDevice)
	data := readAdmin(a)
	if data.TotalVotes != 2 || data.PlayersVoted != 2 || !data.Callups[0].Leader || !data.Callups[1].Leader || data.Callups[0].Percentage != 50 {
		t.Fatalf("ranking: %+v", data)
	}
	expect(409, "DELETE", fmt.Sprintf("/v1/club-admin/aurora/teams/%d", teamID), "", admin)

	// Editing or removing a roster entry keeps both votes and their snapshots.
	updated := append([]ClubPlayer{}, roster[:1]...)
	updated = append(updated, roster[2:]...)
	updated[1].FirstName = "Nome aggiornato"
	if err := store.ReplaceClubPlayers(ctx, siteID, teamID, updated); err != nil {
		t.Fatal(err)
	}
	ballot, _ = readBallot(publicPath(a), nil)
	if len(ballot.Players) != 11 || ballot.Players[0].ID != roster[2].ID {
		t.Fatalf("roster eligibility after edit: %+v", ballot.Players)
	}
	expect(200, "PUT", adminPath(b)+"/mvp", `{"enabled":false}`, admin)
	if _, err := db.Exec(`UPDATE club_matches SET score_for=2, sets_json='["25-20","25-18"]' WHERE id=?`, a); err != nil {
		t.Fatal(err)
	}
	expect(200, "POST", adminPath(a)+"/score", `{"action":"finish","revision":1}`, admin)
	expect(409, "POST", publicPath(a)+"/vote", voteBody(roster[3].ID), secondDevice)
	expect(409, "GET", publicPath(a), "", device)
	expect(409, "GET", qr, "", device)
	data = readAdmin(a)
	if !data.Final || data.Enabled || data.Open || data.QRTarget || data.FinalizedAt == "" || data.ClosedAt == "" || data.TotalVotes != 2 ||
		data.Callups[0].FirstName != roster[1].FirstName || data.Callups[0].Eligible || data.Callups[1].FirstName != roster[2].FirstName {
		t.Fatalf("historical ranking: %+v", data)
	}

	expect(401, "DELETE", adminPath(a), "", nil)
	expect(403, "DELETE", adminPath(a), "", otherAdmin)
	expect(200, "DELETE", fmt.Sprintf("/v1/club-admin/rivale/matches/%d", a), "", otherAdmin)
	if data := readAdmin(a); data.TotalVotes != 2 {
		t.Fatalf("deletion from another club affected ranking: %+v", data)
	}
	expect(200, "DELETE", adminPath(a), "", admin)
	expect(404, "GET", adminPath(a)+"/mvp", "", admin)
	var calendar struct{ Matches []ClubMatch }
	decode(expect(200, "GET", "/v1/club-admin/aurora/matches", "", admin), &calendar)
	if len(calendar.Matches) != 1 || calendar.Matches[0].ID != b {
		t.Fatalf("calendar after deletion: %+v", calendar.Matches)
	}
	for _, table := range []string{"club_mvp_votes", "club_mvp_callups", "club_mvp_matches"} {
		var remaining int
		if err := db.QueryRow(`SELECT COUNT(*) FROM `+table+` WHERE match_id=?`, a).Scan(&remaining); err != nil || remaining != 0 {
			t.Fatalf("match deletion left %d rows in %s: %v", remaining, table, err)
		}
	}
	if data := readAdmin(b); data.TotalVotes != 1 {
		t.Fatalf("match deletion affected the other ballot: %+v", data)
	}

	if err := store.DeleteClubSite(ctx, siteID); err != nil {
		t.Fatal(err)
	}
	var remaining int
	if err := db.QueryRow(`SELECT COUNT(*) FROM club_mvp_votes`).Scan(&remaining); err != nil || remaining != 0 {
		t.Fatalf("full site deletion left %d votes: %v", remaining, err)
	}
}
