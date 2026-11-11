package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

func TestMasterOrganizationColorsAndPublicTheme(t *testing.T) {
	conn, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "theme.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	db, err := database.New(conn)
	if err != nil {
		t.Fatal(err)
	}
	router := &_router{db: db, baseLogger: logrus.New()}
	ctx := reqcontext.RequestContext{AdminRole: "superadmin", Logger: logrus.New()}
	create := httptest.NewRecorder()
	router.createMasterOrganization(create, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"Theme club","slug":"theme-club","sport":"Pallavolo","logo_url":"https://example.com/logo.png","primary_color":"#123456","secondary_color":"#abcdef","tertiary_color":"#fedcba"}`)), ctx)
	if create.Code != http.StatusCreated {
		t.Fatalf("create status: %d", create.Code)
	}
	var org database.Organization
	if err := json.Unmarshal(create.Body.Bytes(), &org); err != nil {
		t.Fatal(err)
	}
	if org.LogoURL != "https://example.com/logo.png" || org.PrimaryColor != "#123456" || org.SecondaryColor != "#abcdef" || org.TertiaryColor != "#fedcba" {
		t.Fatalf("create API colors: %+v", org)
	}
	mux := chi.NewRouter()
	mux.Put("/{id}", func(w http.ResponseWriter, r *http.Request) { router.updateMasterOrganization(w, r, ctx) })
	update := httptest.NewRecorder()
	mux.ServeHTTP(update, httptest.NewRequest(http.MethodPut, "/"+strconv.Itoa(org.ID), strings.NewReader(`{"name":"Theme club","slug":"theme-club","logo_url":"data:image/png;base64,aGVsbG8=","primary_color":"#654321","secondary_color":"#aaaaaa","tertiary_color":"#ffffff"}`)))
	if update.Code != http.StatusOK {
		t.Fatalf("update status: %d", update.Code)
	}
	public := router.wrap(router.getOrganizationTheme)
	response := httptest.NewRecorder()
	public(response, httptest.NewRequest(http.MethodGet, "/organization-theme?organization_slug=theme-club", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("theme without active event: %d", response.Code)
	}
	var theme map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &theme); err != nil {
		t.Fatal(err)
	}
	if len(theme) != 6 || theme["sport"] != "Pallavolo" || theme["name"] != "Theme club" || theme["logo_url"] != "data:image/png;base64,aGVsbG8=" || theme["primary_color"] != "#654321" || theme["secondary_color"] != "#aaaaaa" || theme["tertiary_color"] != "#ffffff" {
		t.Fatalf("public theme: %+v", theme)
	}
	for _, query := range []string{"", "?organization_slug=unknown"} {
		missing := httptest.NewRecorder()
		public(missing, httptest.NewRequest(http.MethodGet, "/organization-theme"+query, nil))
		if missing.Code != http.StatusBadRequest && missing.Code != http.StatusNotFound {
			t.Fatalf("missing scope: %d", missing.Code)
		}
	}
	ctx.AdminRole = "staff"
	denied := httptest.NewRecorder()
	router.createMasterOrganization(denied, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`)), ctx)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("staff create status: %d", denied.Code)
	}
}

func TestPublicSponsorPlacementsIncludeAssignedHiddenSponsors(t *testing.T) {
	rt, ctx, _, otherOrg := votingAPIFixture(t)
	create := func(org, position int) int {
		t.Helper()
		id, err := rt.db.CreateSponsor(database.Sponsor{
			OrganizationID: org, Position: position, Name: "Sponsor", LogoData: "logo", IsActive: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	active := create(ctx.OrganizationID, 1)
	hiddenWall := create(ctx.OrganizationID, 2)
	hiddenMVP := create(ctx.OrganizationID, 3)
	create(ctx.OrganizationID, 4)
	create(otherOrg, 1)
	if _, err := rt.db.SQLConn().Exec(`UPDATE sponsors SET is_active = 0 WHERE organization_id = ? AND id <> ?`, ctx.OrganizationID, active); err != nil {
		t.Fatal(err)
	}
	if err := rt.db.SaveSponsorPlacements(context.Background(), ctx.OrganizationID, database.GameWallPlacements, map[string]int{"basketball_left": hiddenWall}); err != nil {
		t.Fatal(err)
	}
	if err := rt.db.SaveSponsorPlacements(context.Background(), ctx.OrganizationID, database.PostVotePlacements, map[string]int{"post_vote_right": hiddenMVP}); err != nil {
		t.Fatal(err)
	}

	checkCatalog := func(query string, want []int) {
		t.Helper()
		response := httptest.NewRecorder()
		rt.listPublicSponsors(response, httptest.NewRequest(http.MethodGet, "/sponsors"+query, nil), ctx)
		var sponsors []database.Sponsor
		if err := json.Unmarshal(response.Body.Bytes(), &sponsors); response.Code != http.StatusOK || err != nil {
			t.Fatalf("catalog status %d, body %s, error %v", response.Code, response.Body.String(), err)
		}
		ids := make([]int, 0, len(sponsors))
		for _, sponsor := range sponsors {
			ids = append(ids, sponsor.ID)
			if sponsor.IsActive != (sponsor.ID == active) {
				t.Fatalf("unexpected active flag for sponsor %d", sponsor.ID)
			}
		}
		if !reflect.DeepEqual(ids, want) {
			t.Fatalf("catalog %q: got %v, want %v", query, ids, want)
		}
	}
	checkCatalog("", []int{active})
	checkCatalog("?include_placements=1", []int{active, hiddenWall, hiddenMVP})

	theme := httptest.NewRecorder()
	rt.getOrganizationTheme(theme, httptest.NewRequest(http.MethodGet, "/organization-theme", nil), ctx)
	var assignments struct {
		Walls map[string]int `json:"game_wall_sponsors"`
		MVP   map[string]int `json:"post_vote_sponsors"`
	}
	if err := json.Unmarshal(theme.Body.Bytes(), &assignments); theme.Code != http.StatusOK || err != nil {
		t.Fatalf("theme status %d, body %s, error %v", theme.Code, theme.Body.String(), err)
	}
	if assignments.Walls["basketball_left"] != hiddenWall || assignments.MVP["post_vote_right"] != hiddenMVP {
		t.Fatalf("missing hidden placements: %+v", assignments)
	}
	if err := rt.db.SaveSponsorPlacements(context.Background(), ctx.OrganizationID, database.PostVotePlacements, nil); err != nil {
		t.Fatal(err)
	}
	checkCatalog("?include_placements=1", []int{active, hiddenWall})
}
