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

func TestClubSponsorsLifecycleAndIsolation(t *testing.T) {
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "sponsors.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	store := NewStore(db)
	if err := store.EnsureClubTables(); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	ids := map[string]int64{}
	tokens := map[string]string{}
	for index, slug := range []string{"alpha", "beta"} {
		id, err := store.CreateClubSite(ctx, ClubSite{OrganizationID: int64(index + 1), Slug: slug, Name: slug, Active: true}, nil, "club-"+slug, "hash")
		if err != nil {
			t.Fatal(err)
		}
		ids[slug] = id
		admin, err := store.ClubAdminByUsername(ctx, "club-"+slug)
		if err != nil {
			t.Fatal(err)
		}
		tokens[slug], err = store.CreateClubSession(ctx, admin.ID, id, time.Hour)
		if err != nil {
			t.Fatal(err)
		}
	}
	rt := &_router{store: store, router: chi.NewRouter()}
	registerClubRoutes(rt)
	call := func(method, path, token, body string, status int) []byte {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if token != "" {
			r.AddCookie(&http.Cookie{Name: clubCookieName, Value: token})
		}
		w := httptest.NewRecorder()
		rt.router.ServeHTTP(w, r)
		if w.Code != status {
			t.Fatalf("%s %s: status %d want %d; %s", method, path, w.Code, status, w.Body.String())
		}
		return w.Body.Bytes()
	}
	base := "/v1/club-admin/alpha"
	validBody := `{"name":" Sponsor ","logoUrl":" /logo.png ","website":" https://sponsor.example ","description":" Descrizione ","position":2}`
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete} {
		path := base + "/sponsors"
		if method == http.MethodPut || method == http.MethodDelete {
			path += "/1"
		}
		call(method, path, "", validBody, 401)
		call(method, path, tokens["beta"], validBody, 403)
	}
	for _, body := range []string{
		`{`, validBody + validBody,
		`{"name":"Sponsor","logoUrl":"/logo.png","website":"javascript:alert(1)"}`,
		`{"name":"Sponsor","logoUrl":"data:image/png;base64,bm90IGFuIGltYWdl","website":"https://sponsor.example"}`,
		`{"name":"Sponsor","logoUrl":"/logo.png","website":"https://sponsor.example","position":1.5}`,
		strings.Repeat(" ", maxClubSponsorBodyBytes+1) + validBody,
	} {
		call("POST", base+"/sponsors", tokens["alpha"], body, 400)
	}
	create := func(slug, body string) int64 {
		t.Helper()
		var response struct{ ID int64 }
		if err := json.Unmarshal(call("POST", "/v1/club-admin/"+slug+"/sponsors", tokens[slug], body, 200), &response); err != nil || response.ID <= 0 {
			t.Fatalf("create sponsor: %#v, %v", response, err)
		}
		return response.ID
	}
	first := create("alpha", validBody)
	hidden := create("alpha", `{"name":"Hidden","logoUrl":"/hidden.png","website":"https://hidden.example","position":0,"active":false}`)
	second := create("alpha", fmt.Sprintf(`{"name":"Second","logoUrl":%q,"website":"https://second.example","position":1}`, clubTestIcon))
	foreign := create("beta", `{"name":"Foreign","logoUrl":"/foreign.png","website":"https://foreign.example"}`)
	// Payload IDs are ignored on POST and cannot overwrite an existing sponsor.
	clone := create("alpha", fmt.Sprintf(`{"id":%d,"name":"Clone","logoUrl":"/clone.png","website":"https://clone.example","position":2}`, first))
	if clone == first {
		t.Fatal("POST body ID updated an existing sponsor")
	}
	for _, recordID := range []int64{foreign, 999999} {
		path := fmt.Sprintf("%s/sponsors/%d", base, recordID)
		call("PUT", path, tokens["alpha"], validBody, 404)
		call("DELETE", path, tokens["alpha"], "", 404)
	}
	var overview struct {
		Sponsors []ClubSponsor
		Counts   map[string]int
	}
	if err := json.Unmarshal(call("GET", base+"/overview", tokens["alpha"], "", 200), &overview); err != nil {
		t.Fatal(err)
	}
	if len(overview.Sponsors) != 4 || overview.Counts["sponsors"] != 4 || overview.Sponsors[0].ID != hidden || overview.Sponsors[0].Active {
		t.Fatalf("overview omitted hidden sponsor/count: %#v", overview)
	}
	var listing struct{ Sponsors []ClubSponsor }
	if err := json.Unmarshal(call("GET", base+"/sponsors", tokens["alpha"], "", 200), &listing); err != nil {
		t.Fatal(err)
	}
	if len(listing.Sponsors) != 4 || listing.Sponsors[2].Name != "Sponsor" || listing.Sponsors[2].LogoURL != "/logo.png" || listing.Sponsors[2].Website != "https://sponsor.example" || listing.Sponsors[2].Description != "Descrizione" || !listing.Sponsors[2].Active {
		t.Fatalf("sponsor fields/defaults/trim: %#v", listing)
	}
	var public struct{ Sponsors []ClubSponsor }
	if err := json.Unmarshal(call("GET", "/v1/clubs/alpha/home", "", "", 200), &public); err != nil {
		t.Fatal(err)
	}
	if len(public.Sponsors) != 3 || public.Sponsors[0].ID != second || public.Sponsors[1].ID != first || public.Sponsors[2].ID != clone {
		t.Fatalf("public visibility or position/ID order: %#v", public.Sponsors)
	}
	if public.Sponsors[0].LogoURL != clubTestIcon || public.Sponsors[0].Description != "" {
		t.Fatalf("uploaded logo or optional description changed: %#v", public.Sponsors[0])
	}
	// The URL ID wins over a conflicting payload ID; visibility can be changed explicitly.
	call("PUT", fmt.Sprintf("%s/sponsors/%d", base, first), tokens["alpha"], fmt.Sprintf(`{"id":%d,"name":"Updated","logoUrl":"/updated.png","website":"https://updated.example","active":false}`, foreign), 200)
	items, err := store.ListClubSponsors(ctx, ids["alpha"], true)
	if err != nil || len(items) != 2 {
		t.Fatalf("updated visibility: %#v %v", items, err)
	}
	foreignItems, err := store.ListClubSponsors(ctx, ids["beta"], false)
	if err != nil || len(foreignItems) != 1 || foreignItems[0].Name != "Foreign" {
		t.Fatalf("another club was modified: %#v %v", foreignItems, err)
	}
	call("DELETE", fmt.Sprintf("%s/sponsors/%d", base, second), tokens["alpha"], "", 200)
	call("DELETE", fmt.Sprintf("%s/sponsors/%d", base, second), tokens["alpha"], "", 404)
	// Repeated table setup preserves records, while deleting a club removes its sponsors only.
	if err := store.EnsureClubTables(); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteClubSite(ctx, ids["alpha"]); err != nil {
		t.Fatal(err)
	}
	items, err = store.ListClubSponsors(ctx, ids["alpha"], false)
	if err != nil || len(items) != 0 {
		t.Fatalf("deleted club left sponsors: %#v %v", items, err)
	}
	foreignItems, err = store.ListClubSponsors(ctx, ids["beta"], false)
	if err != nil || len(foreignItems) != 1 {
		t.Fatalf("another club lost sponsors: %#v %v", foreignItems, err)
	}
}

func TestClubSponsorValidation(t *testing.T) {
	valid := ClubSponsor{Name: "Sponsor", LogoURL: "/logo.png", Website: "https://sponsor.example", Active: true}
	for _, tc := range []struct {
		name   string
		change func(*ClubSponsor)
	}{
		{"missing name", func(v *ClubSponsor) { v.Name = " " }},
		{"long name", func(v *ClubSponsor) { v.Name = strings.Repeat("a", 201) }},
		{"long description", func(v *ClubSponsor) { v.Description = strings.Repeat("a", 3001) }},
		{"negative position", func(v *ClubSponsor) { v.Position = -1 }},
		{"large position", func(v *ClubSponsor) { v.Position = 1000001 }},
		{"missing website", func(v *ClubSponsor) { v.Website = "" }},
		{"website javascript", func(v *ClubSponsor) { v.Website = "javascript:alert(1)" }},
		{"website relative", func(v *ClubSponsor) { v.Website = "/sponsor" }},
		{"website host missing", func(v *ClubSponsor) { v.Website = "https:///sponsor" }},
		{"website whitespace", func(v *ClubSponsor) { v.Website = "https://sponsor.example/a b" }},
		{"website credentials", func(v *ClubSponsor) { v.Website = "https://user:pass@sponsor.example" }},
		{"missing logo", func(v *ClubSponsor) { v.LogoURL = "" }},
		{"logo javascript", func(v *ClubSponsor) { v.LogoURL = "javascript:alert(1)" }},
		{"logo protocol relative", func(v *ClubSponsor) { v.LogoURL = "//external.example/logo.png" }},
		{"logo backslashes", func(v *ClubSponsor) { v.LogoURL = `/\external.example/logo.png` }},
		{"logo malformed url", func(v *ClubSponsor) { v.LogoURL = "https://%invalid/logo.png" }},
		{"logo svg data", func(v *ClubSponsor) { v.LogoURL = "data:image/svg+xml;base64,PHN2Zz4=" }},
		{"logo fake image", func(v *ClubSponsor) { v.LogoURL = "data:image/png;base64,bm90IGFuIGltYWdl" }},
		{"logo bad base64", func(v *ClubSponsor) { v.LogoURL = "data:image/png;base64,!!!!" }},
		{"logo MIME mismatch", func(v *ClubSponsor) { v.LogoURL = strings.Replace(clubTestIcon, "image/png", "image/jpeg", 1) }},
		{"logo oversize", func(v *ClubSponsor) {
			v.LogoURL = "data:image/png;base64," + strings.Repeat("A", maxClubSponsorLogoBytes)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := valid
			tc.change(&v)
			if err := validateClubSponsor(&v); err == nil {
				t.Fatal("invalid sponsor accepted")
			}
		})
	}
	for _, logo := range []string{clubTestIcon, "/images/logo.png?size=200", "https://sponsor.example/logo.png", "http://sponsor.example/logo.png"} {
		v := valid
		v.LogoURL = logo
		if err := validateClubSponsor(&v); err != nil {
			t.Fatalf("valid logo rejected: %q %v", logo, err)
		}
	}
}
