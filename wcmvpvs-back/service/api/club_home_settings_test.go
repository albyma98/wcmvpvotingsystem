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

const clubTestIcon = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jY9kAAAAASUVORK5CYII="

func TestClubHomeSettingsCMSPublicRoundTrip(t *testing.T) {
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "club.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := NewStore(db)
	rt := &_router{store: store, router: chi.NewRouter()}
	registerClubRoutes(rt)
	ctx := context.Background()
	siteID, err := store.CreateClubSite(ctx, ClubSite{OrganizationID: 1, Slug: "aurora", Name: "Aurora", Active: true}, []string{"Under 16"}, "club-aurora", "hash")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateClubSite(ctx, ClubSite{OrganizationID: 2, Slug: "other", Name: "Other", Active: true}, nil, "club-other", "hash"); err != nil {
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
	call := func(method, path, body, session string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if session != "" {
			r.AddCookie(&http.Cookie{Name: clubCookieName, Value: session})
		}
		w := httptest.NewRecorder()
		rt.router.ServeHTTP(w, r)
		return w
	}
	teams, err := store.ListClubTeams(ctx, siteID, true)
	if err != nil || len(teams) != 1 {
		t.Fatalf("teams=%v err=%v", teams, err)
	}
	teamKey := fmt.Sprintf("team-%d", teams[0].ID)
	contacts := []ClubContact{
		{Name: " Segreteria ", Role: "Iscrizioni", Phone: "+39 012 3456789", Email: "info@aurora.it", Address: "Via Roma 1", Website: "https://aurora.it"},
		{Name: "Mario Rossi", Role: "Direttore sportivo"},
	}
	encoded, _ := json.Marshal(map[string]interface{}{"contacts": contacts, "tileIcons": map[string]string{"contacts": clubTestIcon, teamKey: clubTestIcon}})
	if w := call("PUT", "/v1/club-admin/aurora/settings", string(encoded), ""); w.Code != 401 {
		t.Fatalf("anonymous write: %d", w.Code)
	}
	if w := call("PUT", "/v1/club-admin/other/settings", string(encoded), token); w.Code != 403 {
		t.Fatalf("cross-club write: %d", w.Code)
	}
	if w := call("PUT", "/v1/club-admin/aurora/settings", string(encoded), token); w.Code != 200 {
		t.Fatalf("save: %d %s", w.Code, w.Body.String())
	}
	for _, path := range []string{"/v1/clubs/aurora/home", "/v1/club-admin/aurora/overview"} {
		w := call("GET", path, "", token)
		var payload struct {
			Club ClubSite `json:"club"`
		}
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &payload) != nil {
			t.Fatalf("load %s: %d %s", path, w.Code, w.Body.String())
		}
		if len(payload.Club.Contacts) != 2 || payload.Club.Contacts[0].Name != "Segreteria" || payload.Club.Contacts[0].Email != "info@aurora.it" || payload.Club.TileIcons[teamKey] != clubTestIcon || payload.Club.TileIcons["contacts"] != clubTestIcon {
			t.Fatalf("lost saved settings: %#v", payload.Club)
		}
	}
	other, _ := store.ClubSiteBySlug(ctx, "other")
	if len(other.Contacts) != 0 || len(other.TileIcons) != 0 {
		t.Fatal("settings leaked to another club")
	}
	// Invalid edits must leave the previously published contacts and icons intact.
	for _, body := range []string{
		`{"contacts":[{"name":""}]}`,
		`{"contacts":[{"name":"Test","email":"wrong"}]}`,
		`{"contacts":[{"name":"Test","website":"javascript:alert(1)"}]}`,
		`{"tileIcons":{"contacts":"data:image/png;base64,bm90IGFuIGltYWdl"}}`,
		`{"tileIcons":{"contacts":"data:image/svg+xml;base64,PHN2Zz4="}}`,
		`{"name":"` + strings.Repeat("x", maxClubSettingsBytes) + `"}`,
	} {
		if w := call("PUT", "/v1/club-admin/aurora/settings", body, token); w.Code != 400 {
			t.Fatalf("invalid edit accepted: %d", w.Code)
		}
		saved, _ := store.ClubSiteByID(ctx, siteID)
		if saved.Name != "Aurora" || len(saved.Contacts) != 2 || saved.TileIcons["contacts"] != clubTestIcon {
			t.Fatal("invalid edit changed published settings")
		}
	}
	// A subsequent identity edit retains omitted homepage fields.
	if w := call("PUT", "/v1/club-admin/aurora/settings", `{"tagline":"Forza Aurora"}`, token); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	saved, _ := store.ClubSiteByID(ctx, siteID)
	if len(saved.Contacts) != 2 || saved.TileIcons[teamKey] != clubTestIcon {
		t.Fatal("identity edit erased contacts or icons")
	}
	// Reordering, editing and deleting contacts also survive a database reload.
	contacts[0].Name = "Segreteria aggiornata"
	contacts[0], contacts[1] = contacts[1], contacts[0]
	encoded, _ = json.Marshal(map[string]interface{}{"contacts": contacts})
	if w := call("PUT", "/v1/club-admin/aurora/settings", string(encoded), token); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	saved, _ = store.ClubSiteByID(ctx, siteID)
	if saved.Contacts[0].Name != "Mario Rossi" || saved.Contacts[1].Name != "Segreteria aggiornata" {
		t.Fatal("edited contact order was not persisted")
	}
	if w := call("DELETE", fmt.Sprintf("/v1/club-admin/aurora/teams/%d", teams[0].ID), "", token); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if w := call("PUT", "/v1/club-admin/aurora/settings", `{"tagline":"Nuova stagione"}`, token); w.Code != 200 {
		t.Fatalf("deleted team icon prevented saving: %d %s", w.Code, w.Body.String())
	}
	saved, _ = store.ClubSiteByID(ctx, siteID)
	if _, exists := saved.TileIcons[teamKey]; exists || saved.TileIcons["contacts"] != clubTestIcon {
		t.Fatal("deleted team icon was not pruned independently")
	}
	if w := call("PUT", "/v1/club-admin/aurora/settings", `{"contacts":[],"tileIcons":{}}`, token); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	saved, _ = store.ClubSiteByID(ctx, siteID)
	if len(saved.Contacts) != 0 || len(saved.TileIcons) != 0 {
		t.Fatal("removing contacts and restoring default icons was not persisted")
	}
}

func TestClubHomeSettingsLegacyMigration(t *testing.T) {
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "legacy.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE club_sites (
		id INTEGER PRIMARY KEY, organization_id INTEGER NOT NULL UNIQUE, slug TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL, tagline TEXT NOT NULL DEFAULT '', city TEXT NOT NULL DEFAULT '',
		logo_url TEXT NOT NULL DEFAULT '', hero_image_url TEXT NOT NULL DEFAULT '',
		primary_color TEXT NOT NULL DEFAULT '#172554', secondary_color TEXT NOT NULL DEFAULT '#f59e0b',
		season_label TEXT NOT NULL DEFAULT '', active INTEGER NOT NULL DEFAULT 1,
		created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
	); INSERT INTO club_sites (id, organization_id, slug, name) VALUES (1, 1, 'existing', 'Existing Club')`)
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore(db)
	for i := 0; i < 2; i++ {
		if err := store.EnsureClubTables(); err != nil {
			t.Fatal(err)
		}
	}
	site, err := store.ClubSiteBySlug(context.Background(), "existing")
	if err != nil || site.Name != "Existing Club" || site.Contacts == nil || site.TileIcons == nil {
		t.Fatalf("migration lost existing site or defaults: %#v err=%v", site, err)
	}
}
