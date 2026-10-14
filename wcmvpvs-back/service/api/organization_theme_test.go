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
	router.createMasterOrganization(create, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"Theme club","slug":"theme-club","logo_url":"https://example.com/logo.png","primary_color":"#123456","secondary_color":"#abcdef","tertiary_color":"#fedcba"}`)), ctx)
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
	if len(theme) != 5 || theme["name"] != "Theme club" || theme["logo_url"] != "data:image/png;base64,aGVsbG8=" || theme["primary_color"] != "#654321" || theme["secondary_color"] != "#aaaaaa" || theme["tertiary_color"] != "#ffffff" {
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
