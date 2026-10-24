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

func TestMasterOrganizationSport(t *testing.T) {
	conn, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "sport.db"))
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
	mux := chi.NewRouter()
	mux.Post("/", func(w http.ResponseWriter, r *http.Request) { router.createMasterOrganization(w, r, ctx) })
	mux.Put("/{id}", func(w http.ResponseWriter, r *http.Request) { router.updateMasterOrganization(w, r, ctx) })
	mux.Get("/", func(w http.ResponseWriter, r *http.Request) { router.listMasterOrganizations(w, r, ctx) })
	mux.Get("/{id}", func(w http.ResponseWriter, r *http.Request) { router.getMasterOrganizationDetail(w, r, ctx) })
	request := func(method, path, body string, status int) *httptest.ResponseRecorder {
		t.Helper()
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(method, path, strings.NewReader(body)))
		if response.Code != status {
			t.Fatalf("%s %s: got %d, want %d; %s", method, path, response.Code, status, response.Body.String())
		}
		return response
	}
	sports := []string{"Calcio", "Pallacanestro", "Pallavolo"}
	for i, sport := range sports {
		create := request(http.MethodPost, "/", `{"name":"`+sport+`","sport":"`+sport+`"}`, http.StatusCreated)
		var org database.Organization
		if err := json.Unmarshal(create.Body.Bytes(), &org); err != nil {
			t.Fatal(err)
		}
		if org.Sport != sport {
			t.Fatalf("create sport: %+v", org)
		}
		path := "/" + strconv.Itoa(org.ID)
		updatedSport := sports[(i+1)%len(sports)]
		request(http.MethodPut, path, `{"name":"`+sport+`","sport":" `+updatedSport+` "}`, http.StatusOK)
		// Older clients omitting the new field must preserve its stored value.
		request(http.MethodPut, path, `{"name":"`+sport+`"}`, http.StatusOK)
		request(http.MethodPut, path, `{"name":"`+sport+`","sport":"Tennis"}`, http.StatusBadRequest)
		bySlug, err := db.GetOrganizationBySlug(org.Slug)
		if err != nil || bySlug.Sport != updatedSport {
			t.Fatalf("stored sport: %+v, %v", bySlug, err)
		}
		detail := request(http.MethodGet, path, "", http.StatusOK)
		var data struct {
			Organization database.Organization `json:"organization"`
		}
		if err := json.Unmarshal(detail.Body.Bytes(), &data); err != nil {
			t.Fatal(err)
		}
		if data.Organization.Sport != updatedSport {
			t.Fatalf("detail sport: %+v", data)
		}
	}
	list := request(http.MethodGet, "/", "", http.StatusOK)
	var orgs []database.Organization
	if err := json.Unmarshal(list.Body.Bytes(), &orgs); err != nil {
		t.Fatal(err)
	}
	if len(orgs) != 3 {
		t.Fatalf("list count: %d", len(orgs))
	}
	for _, org := range orgs {
		if org.Sport == "" {
			t.Fatalf("sport missing from list: %+v", org)
		}
	}
	request(http.MethodPost, "/", `{"name":"Invalid","sport":"Tennis"}`, http.StatusBadRequest)
	ctx.AdminRole = "staff"
	request(http.MethodPost, "/", `{"name":"Staff","sport":"Calcio"}`, http.StatusForbidden)
	request(http.MethodPut, "/"+strconv.Itoa(orgs[0].ID), `{"name":"Staff","sport":"Calcio"}`, http.StatusForbidden)
}
