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
)

func TestClubCMSVisibilityAndStructuredEntries(t *testing.T) {
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "cms.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	store := NewStore(db)
	if err := store.EnsureClubTables(); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	siteID, err := store.CreateClubSite(ctx, ClubSite{OrganizationID: 1, Slug: "cms", Name: "CMS Club", Active: true}, []string{"Under 18"}, "club-cms", "hash")
	if err != nil {
		t.Fatal(err)
	}
	admin, err := store.ClubAdminByUsername(ctx, "club-cms")
	if err != nil {
		t.Fatal(err)
	}
	token, err := store.CreateClubSession(ctx, admin.ID, siteID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	rt := &_router{store: store, router: chi.NewRouter()}
	registerClubRoutes(rt)
	call := func(method, endpoint, body string) []byte {
		t.Helper()
		r := httptest.NewRequest(method, "/v1/club-admin/cms"+endpoint, strings.NewReader(body))
		r.AddCookie(&http.Cookie{Name: clubCookieName, Value: token})
		w := httptest.NewRecorder()
		rt.router.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("%s %s: %d %s", method, endpoint, w.Code, w.Body.String())
		}
		return w.Body.Bytes()
	}

	for _, resource := range []struct {
		path, fields string
	}{
		{"teams", `"name":"Seconda squadra"`},
		{"posts", `"title":"Comunicazione"`},
		{"gallery", `"imageUrl":"/foto.jpg"`},
	} {
		t.Run(resource.path, func(t *testing.T) {
			for index, active := range []string{`,"active":false`, `,"active":true`, ""} {
				fields := resource.fields
				if resource.path == "teams" {
					fields = fmt.Sprintf(`"name":"Seconda squadra %d"`, index)
				}
				var result struct{ ID int64 }
				if err := json.Unmarshal(call("POST", "/"+resource.path, "{"+fields+active+"}"), &result); err != nil {
					t.Fatal(err)
				}
				var overview map[string]json.RawMessage
				if err := json.Unmarshal(call("GET", "/overview", ""), &overview); err != nil {
					t.Fatal(err)
				}
				var records []struct {
					ID     int64
					Active bool
				}
				if err := json.Unmarshal(overview[resource.path], &records); err != nil {
					t.Fatal(err)
				}
				found := false
				for _, record := range records {
					if record.ID == result.ID {
						found = true
						if record.Active != (active != `,"active":false`) {
							t.Fatalf("visibility was not preserved: %#v", record)
						}
					}
				}
				if !found {
					t.Fatal("created record missing from CMS")
				}
			}
		})
	}

	// Hidden records stay editable in the CMS and are absent from the public hub.
	w := httptest.NewRecorder()
	rt.router.ServeHTTP(w, httptest.NewRequest("GET", "/v1/clubs/cms/home", nil))
	var public struct {
		Teams   []ClubTeam
		Posts   []ClubPost
		Gallery []ClubGalleryItem
	}
	if err := json.Unmarshal(w.Body.Bytes(), &public); err != nil || w.Code != 200 {
		t.Fatalf("public hub: %d %s", w.Code, w.Body.String())
	}
	if len(public.Teams) != 3 || len(public.Posts) != 2 || len(public.Gallery) != 2 {
		t.Fatalf("hidden records leaked or default visibility changed: %#v", public)
	}

	teamID := public.Teams[0].ID
	players := `{"players":[{"firstName":"Maria Luisa","lastName":"De Luca","jerseyNumber":7,"role":"Centrale"},{"firstName":"Anna","lastName":"Rossi","jerseyNumber":0}]}`
	call("PUT", fmt.Sprintf("/teams/%d/players", teamID), players)
	roster, err := store.ListClubPlayers(ctx, siteID, teamID)
	if err != nil || len(roster) != 2 || roster[0].FirstName != "Maria Luisa" || roster[0].LastName != "De Luca" || roster[1].JerseyNumber != 0 {
		t.Fatalf("structured roster: %#v %v", roster, err)
	}

	var created struct{ ID int64 }
	body := fmt.Sprintf(`{"teamId":%d,"opponent":"Volley","scheduledAt":"2026-10-25T18:30","home":false,"status":"completed","scoreFor":1,"scoreAgainst":0,"setsJson":"[\"25-18\"]"}`, teamID)
	if err := json.Unmarshal(call("POST", "/matches", body), &created); err != nil {
		t.Fatal(err)
	}
	call("PUT", fmt.Sprintf("/matches/%d", created.ID), strings.Replace(body, "25-18", "25-20", 1))
	matches, err := store.ListClubMatches(ctx, siteID, teamID)
	if err != nil || len(matches) != 1 || matches[0].SetsJSON != `["25-20"]` || matches[0].Home {
		t.Fatalf("updated match partials: %#v %v", matches, err)
	}
}
