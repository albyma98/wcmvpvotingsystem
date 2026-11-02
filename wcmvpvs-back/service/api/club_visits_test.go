package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClubVisitsCountsIsolationAndDeletion(t *testing.T) {
	rt, siteID, otherID, _ := clubDomainFixture(t)
	ctx := context.Background()
	// The schema change is safe to run again on existing installations.
	if err := rt.store.EnsureClubTables(); err != nil {
		t.Fatal(err)
	}
	check := func(slug string, total, unique int64) {
		t.Helper()
		w := domainRequest(rt, "GET", "/admin/master/clubs", "", "master", "mvp.wearingcash.it")
		var result struct {
			Clubs []struct {
				Slug         string `json:"slug"`
				TotalVisits  int64  `json:"totalVisits"`
				UniqueVisits int64  `json:"uniqueVisits"`
			}
		}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || w.Code != 200 {
			t.Fatalf("master list: %d %s", w.Code, w.Body.String())
		}
		for _, club := range result.Clubs {
			if club.Slug == slug {
				if club.TotalVisits != total || club.UniqueVisits != unique {
					t.Fatalf("%s counters: %+v, want %d/%d", slug, club, total, unique)
				}
				return
			}
		}
		t.Fatalf("club %s missing", slug)
	}
	check("aurora", 0, 0)
	visit := func(slug, visitor, id string) {
		t.Helper()
		body := fmt.Sprintf(`{"visitorId":%q,"visitId":%q}`, visitor, id)
		w := domainRequest(rt, "POST", "/v1/clubs/"+slug+"/visits", body, "", "mvp.wearingcash.it")
		if w.Code != http.StatusNoContent || w.Body.Len() != 0 {
			t.Fatalf("collect: %d %s", w.Code, w.Body.String())
		}
	}
	const visitor = "123e4567-e89b-42d3-a456-426614174000"
	const visitID = "123e4567-e89b-42d3-a456-426614174001"
	visit("aurora", visitor, visitID)
	visit("aurora", strings.ToUpper(visitor), strings.ToUpper(visitID)) // Duplicate delivery.
	visit("aurora", visitor, "123e4567-e89b-42d3-a456-426614174002")
	visit("aurora", "123e4567-e89b-42d3-a456-426614174003", "123e4567-e89b-42d3-a456-426614174004")
	check("aurora", 3, 2)
	check("other", 0, 0)
	visit("other", visitor, visitID) // IDs are scoped to each club.
	check("other", 1, 1)
	var firstKey, secondKey string
	if err := rt.store.db.QueryRow(`SELECT visitor_key FROM club_visits WHERE site_id=? AND visit_id=?`, siteID, visitID).Scan(&firstKey); err != nil {
		t.Fatal(err)
	}
	if err := rt.store.db.QueryRow(`SELECT visitor_key FROM club_visits WHERE site_id=? AND visit_id=?`, otherID, visitID).Scan(&secondKey); err != nil {
		t.Fatal(err)
	}
	if len(firstKey) != 64 || firstKey == secondKey || firstKey == visitor {
		t.Fatal("visitor keys must be hashed and scoped to the club")
	}
	if err := rt.store.DeleteClubSite(ctx, siteID); err != nil {
		t.Fatal(err)
	}
	var remaining int
	if err := rt.store.db.QueryRow(`SELECT COUNT(*) FROM club_visits WHERE site_id=?`, siteID).Scan(&remaining); err != nil || remaining != 0 {
		t.Fatalf("visit cleanup: %d %v", remaining, err)
	}
	check("other", 1, 1)
}

func TestClubVisitPrivacyAndAuthorization(t *testing.T) {
	rt, siteID, _, _ := clubDomainFixture(t)
	for token, want := range map[string]int{"": 401, "staff": 403} {
		w := domainRequest(rt, "GET", "/admin/master/clubs", "", token, "mvp.wearingcash.it")
		if w.Code != want || strings.Contains(w.Body.String(), "totalVisits") {
			t.Fatalf("unauthorized stats: %s %d %s", token, w.Code, w.Body.String())
		}
	}
	ctx := context.Background()
	admin, err := rt.store.ClubAdminByUsername(ctx, "aurora")
	if err != nil {
		t.Fatal(err)
	}
	token, err := rt.store.CreateClubSession(ctx, admin.ID, siteID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/v1/clubs/aurora/home", "/v1/club-admin/aurora/overview", "/admin/master/clubs"} {
		r := httptest.NewRequest("GET", path, nil)
		r.AddCookie(&http.Cookie{Name: clubCookieName, Value: token})
		w := httptest.NewRecorder()
		rt.router.ServeHTTP(w, r)
		if strings.Contains(w.Body.String(), "totalVisits") || strings.Contains(w.Body.String(), "uniqueVisits") {
			t.Fatalf("counters leaked to %s: %s", path, w.Body.String())
		}
		if path == "/admin/master/clubs" && w.Code != 401 || path != "/admin/master/clubs" && w.Code != 200 {
			t.Fatalf("unexpected status %s: %d", path, w.Code)
		}
	}
}

func TestClubVisitsRejectInvalidCollection(t *testing.T) {
	rt, siteID, _, _ := clubDomainFixture(t)
	const valid = `{"visitorId":"123e4567-e89b-42d3-a456-426614174000","visitId":"123e4567-e89b-42d3-a456-426614174001"}`
	for _, body := range []string{"", `{}`, `{"visitorId":"arbitrary","visitId":"bad"}`, valid + `{}`, strings.Repeat(" ", 1024) + valid, strings.Replace(valid, "123e4567-e89b-42d3-a456-426614174000", "00000000-0000-0000-0000-000000000000", 1)} {
		w := domainRequest(rt, "POST", "/v1/clubs/aurora/visits", body, "", "mvp.wearingcash.it")
		if w.Code != 400 {
			t.Fatalf("accepted invalid visit: %d %s", w.Code, w.Body.String())
		}
	}
	w := domainRequest(rt, "POST", "/v1/clubs/missing/visits", valid, "", "mvp.wearingcash.it")
	if w.Code != 404 {
		t.Fatalf("missing club: %d", w.Code)
	}
	if _, err := rt.store.db.Exec(`UPDATE club_sites SET active=0 WHERE id=?`, siteID); err != nil {
		t.Fatal(err)
	}
	w = domainRequest(rt, "POST", "/v1/clubs/aurora/visits", valid, "", "mvp.wearingcash.it")
	if w.Code != 404 {
		t.Fatalf("inactive club: %d", w.Code)
	}
	w = domainRequest(rt, "GET", "/v1/clubs/aurora/visits", "", "", "mvp.wearingcash.it")
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("public read allowed: %d", w.Code)
	}
}
