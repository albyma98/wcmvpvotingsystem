package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

func TestArenaCatalogAndOrganizationAssignment(t *testing.T) {
	conn, err := sql.Open("sqlite3", "file:"+filepath.ToSlash(filepath.Join(t.TempDir(), "arenas.db"))+"?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	db, err := database.New(conn)
	if err != nil {
		t.Fatal(err)
	}

	logger := logrus.New()
	rt := &_router{db: db, baseLogger: logger}
	ctx := reqcontext.RequestContext{AdminRole: "superadmin", Logger: logger}
	master := func(handler httpRouterHandler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) { handler(w, r, ctx) }
	}
	mux := chi.NewRouter()
	mux.Get("/arenas", master(rt.listMasterArenas))
	mux.Post("/arenas", master(rt.saveMasterArena))
	mux.Get("/arenas/{id}", master(rt.getMasterArena))
	mux.Put("/arenas/{id}", master(rt.saveMasterArena))
	mux.Post("/organizations", master(rt.createMasterOrganization))
	mux.Get("/organizations/{id}", master(rt.getMasterOrganizationDetail))
	mux.Put("/organizations/{id}", master(rt.updateMasterOrganization))
	mux.Patch("/organizations/{id}/arena", master(rt.setMasterOrganizationArena))
	mux.Get("/fan/arena/scene", rt.wrap(rt.getOrganizationArenaScene))
	request := func(method, path, body string, status int, response interface{}) {
		t.Helper()
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
		if w.Code != status {
			t.Fatalf("%s %s: got %d, want %d, body %s", method, path, w.Code, status, w.Body)
		}
		if response != nil {
			if err := json.Unmarshal(w.Body.Bytes(), response); err != nil {
				t.Fatalf("%s %s: %v", method, path, err)
			}
		}
	}

	var catalog struct {
		Arenas []database.ArenaSummary `json:"arenas"`
	}
	request(http.MethodGet, "/arenas", "", http.StatusOK, &catalog)
	if len(catalog.Arenas) != 2 || catalog.Arenas[0].Slug != "baltur-arena" || catalog.Arenas[1].Slug != "joy-volley-arena" {
		t.Fatalf("unexpected arena seed: %+v", catalog.Arenas)
	}
	baltur := catalog.Arenas[0]
	if baltur.Capacity == nil || *baltur.Capacity != 2000 || baltur.City != "Cento" || !baltur.IsActive {
		t.Fatalf("unexpected Baltur metadata: %+v", baltur)
	}
	balturPath := fmt.Sprintf("/arenas/%d", baltur.ID)
	var seededScene database.Arena
	request(http.MethodGet, balturPath, "", http.StatusOK, &seededScene)
	var layout struct {
		Sectors []json.RawMessage `json:"sectors"`
	}
	if err := json.Unmarshal(seededScene.SceneConfig, &layout); err != nil || len(layout.Sectors) < 7 {
		t.Fatalf("expected configured Baltur sectors: %v", err)
	}

	const custom = `{"name":"Arena prova","slug":"arena-prova","city":"Roma","is_active":true,"scene_config":{}}`
	request(http.MethodPost, "/arenas", custom, http.StatusCreated, nil)
	request(http.MethodPost, "/arenas", custom, http.StatusConflict, nil)
	request(http.MethodPost, "/arenas", strings.Replace(custom, `"scene_config":{}`, `"scene_config":[]`, 1), http.StatusBadRequest, nil)
	request(http.MethodPost, "/organizations", `{"name":"Invalid","arena_id":99999}`, http.StatusBadRequest, nil)
	var club database.Organization
	request(http.MethodPost, "/organizations", fmt.Sprintf(`{"name":"Club Cento","is_active":true,"arena_id":%d}`, baltur.ID), http.StatusCreated, &club)
	var other database.Organization
	request(http.MethodPost, "/organizations", `{"name":"Club Roma","is_active":true}`, http.StatusCreated, &other)
	if club.ArenaID == nil || *club.ArenaID != baltur.ID || club.Arena == nil || club.Arena.Slug != baltur.Slug || other.Arena != nil || other.ArenaID != nil {
		t.Fatalf("unexpected assignments: %+v, %+v", club, other)
	}
	if _, err := conn.Exec(`UPDATE organizations SET arena_id = 99999 WHERE id = ?`, other.ID); err == nil {
		t.Fatal("database accepted an unknown arena ID")
	}

	clubPath := fmt.Sprintf("/organizations/%d", club.ID)
	clubScene := "/fan/arena/scene?organization_slug=" + club.Slug
	request(http.MethodPut, clubPath, `{"name":"Club Cento","is_active":true}`, http.StatusOK, nil)
	var detail struct {
		Organization database.Organization `json:"organization"`
	}
	request(http.MethodGet, clubPath, "", http.StatusOK, &detail)
	if detail.Organization.Arena == nil || detail.Organization.Arena.ID != baltur.ID {
		t.Fatal("existing organization update dropped its arena")
	}
	var scene struct {
		Arena *database.Arena `json:"arena"`
	}
	request(http.MethodGet, clubScene, "", http.StatusOK, &scene)
	if scene.Arena == nil || scene.Arena.Slug != baltur.Slug || len(scene.Arena.SceneConfig) == 0 {
		t.Fatal("assigned scene missing")
	}
	request(http.MethodGet, "/fan/arena/scene?organization_slug="+other.Slug, "", http.StatusOK, &scene)
	if scene.Arena != nil {
		t.Fatal("scene leaked to an unassigned organization")
	}

	request(http.MethodPut, balturPath, `{"name":"Baltur Arena","slug":"baltur-cento","city":"Cento","is_active":false,"scene_config":{}}`, http.StatusOK, nil)
	if _, err := database.New(conn); err != nil {
		t.Fatal(err)
	}
	request(http.MethodGet, "/arenas", "", http.StatusOK, &catalog)
	if len(catalog.Arenas) != 3 || catalog.Arenas[1].Slug != "baltur-cento" || catalog.Arenas[1].IsActive {
		t.Fatalf("startup lost the catalog edit or repeated the seed: %+v", catalog.Arenas)
	}
	request(http.MethodGet, clubScene, "", http.StatusOK, &scene)
	if scene.Arena != nil {
		t.Fatal("inactive arena scene was exposed")
	}
	request(http.MethodPatch, clubPath+"/arena", `{"arena_id":99999}`, http.StatusBadRequest, nil)
	request(http.MethodPatch, clubPath+"/arena", `{"arena_id":null}`, http.StatusOK, &club)
	if club.ArenaID != nil || club.Arena != nil || club.Name != "Club Cento" {
		t.Fatalf("unexpected clear result: %+v", club)
	}
	request(http.MethodPatch, clubPath+"/arena", fmt.Sprintf(`{"arena_id":%d}`, baltur.ID), http.StatusOK, &club)
	request(http.MethodPut, clubPath, `{"name":"Club Cento","is_active":true,"arena_id":null}`, http.StatusOK, &club)
	if club.ArenaID != nil || club.Arena != nil {
		t.Fatalf("explicit null update did not clear assignment: %+v", club)
	}
	ctx.AdminRole = "staff"
	request(http.MethodGet, "/arenas", "", http.StatusForbidden, nil)
	request(http.MethodPatch, clubPath+"/arena", `{"arena_id":null}`, http.StatusForbidden, nil)
}
