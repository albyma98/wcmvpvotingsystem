package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/sirupsen/logrus"
)

type staffAccessDB struct{ database.AppDatabase }

func (db *staffAccessDB) GetOrganizationBySlug(slug string) (database.Organization, error) {
	id := 7
	if slug == "other" {
		id = 8
	}
	return database.Organization{ID: id, Slug: slug, BarEnabled: true}, nil
}

func (db *staffAccessDB) GetOrganization(id int) (database.Organization, error) {
	return database.Organization{ID: id, Slug: "club", BarEnabled: true}, nil
}

func staffAccessRouter(role string) *_router {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return &_router{db: &staffAccessDB{}, baseLogger: logger, sessionTimeout: time.Hour, adminSessions: map[string]adminSession{
		"session": {AdminID: 1, Username: "Operator", Role: role, OrganizationID: 7, OrganizationSlug: "club", ExpiresAt: time.Now().Add(time.Hour)},
	}}
}

func TestStaffAccessOnlyVotingClosureRoutes(t *testing.T) {
	for _, tc := range []struct {
		method, path string
		allowed      bool
	}{
		{http.MethodGet, "/admin/me", true},
		{http.MethodGet, "/events", true},
		{http.MethodPost, "/events/42/close-votes", true},
		{http.MethodGet, "/events/42/voting-closure", true},
		{http.MethodPost, "/events/42/voting-closure/retry", true},
		{http.MethodPost, "/events/42/activate", true},
		{http.MethodPost, "/events/deactivate", true},
		{http.MethodGet, "/teams", false},
		{http.MethodGet, "/players", false},
		{http.MethodGet, "/admins", false},
		{http.MethodGet, "/admin/events/history", false},
		{http.MethodGet, "/admin/events/42/stories", false},
		{http.MethodGet, "/admin/events/42/quiz", false},
		{http.MethodGet, "/admin/voting-settings", false},
		{http.MethodPut, "/admin/voting-settings", false},
		{http.MethodPost, "/admin/voting-settings/test", false},
		{http.MethodGet, "/admin/marketing/audience", false},
		{http.MethodGet, "/admin/master/organizations", false},
		{http.MethodGet, "/admin/bar/orders", false},
		{http.MethodGet, "/events/42/tickets", false},
		{http.MethodPost, "/events/42/prizes/1/assign", false},
		{http.MethodDelete, "/events/42/prizes/1/winner", false},
		{http.MethodPost, "/events", false},
		{http.MethodPut, "/events/42", false},
		{http.MethodDelete, "/events/42", false},
		{http.MethodPost, "/events/42/conclude", false},
		{http.MethodPost, "/notifications/test-sms", false},
		{http.MethodGet, "/events/42/voting-closure/retry", false},
		{http.MethodPost, "/events/not-an-id/close-votes", false},
		{http.MethodPost, "/events/0/close-votes", false},
		{http.MethodPost, "/events/-1/close-votes", false},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			router := staffAccessRouter("staff")
			called := false
			handler := router.wrapAdmin(func(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
				called = true
				if ctx.AdminRole != "staff" || ctx.OrganizationID != 7 {
					t.Fatalf("incorrect session scope: %+v", ctx)
				}
				w.WriteHeader(http.StatusNoContent)
			})
			r := httptest.NewRequest(tc.method, tc.path, nil)
			r.Header.Set("X-Organization-Slug", "club")
			r.AddCookie(&http.Cookie{Name: adminCookieName(), Value: "session"})
			w := httptest.NewRecorder()
			handler(w, r)
			want := http.StatusForbidden
			if tc.allowed {
				want = http.StatusNoContent
			}
			if w.Code != want || called != tc.allowed {
				t.Fatalf("status=%d handler=%v; wanted %d", w.Code, called, want)
			}
		})
	}
}

func TestStaffAccessKeepsOrganizationAndSessionBoundaries(t *testing.T) {
	for _, tc := range []struct {
		token, slug string
		status      int
	}{
		{"session", "other", http.StatusUnauthorized},
		{"session", "", http.StatusBadRequest},
		{"invalid", "club", http.StatusUnauthorized},
	} {
		router := staffAccessRouter("staff")
		r := httptest.NewRequest(http.MethodGet, "/events", nil)
		r.Header.Set("Authorization", "Bearer "+tc.token)
		r.Header.Set("X-Organization-Slug", tc.slug)
		w := httptest.NewRecorder()
		router.wrapAdmin(func(http.ResponseWriter, *http.Request, reqcontext.RequestContext) {
			t.Fatal("invalid staff session reached handler")
		})(w, r)
		if w.Code != tc.status {
			t.Fatalf("%+v: %d", tc, w.Code)
		}
	}
	for _, role := range []string{"superadmin", "master"} {
		router := staffAccessRouter(role)
		r := httptest.NewRequest(http.MethodGet, "/teams", nil)
		r.Header.Set("X-Organization-Slug", "club")
		r.Header.Set("Authorization", "Bearer session")
		w := httptest.NewRecorder()
		router.wrapAdmin(func(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
			w.WriteHeader(http.StatusNoContent)
		})(w, r)
		if w.Code != http.StatusNoContent {
			t.Fatalf("%s access changed: %d", role, w.Code)
		}
	}
}
