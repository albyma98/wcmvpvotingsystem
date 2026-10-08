package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

type organizationDeleteAuthDB struct {
	database.AppDatabase
	admin       database.Admin
	deleteCalls int
}

func (db *organizationDeleteAuthDB) GetAdminByID(int) (database.Admin, error) {
	return db.admin, nil
}

func (db *organizationDeleteAuthDB) DeleteOrganizationData(int) ([]int, error) {
	db.deleteCalls++
	return nil, nil
}

func TestDeleteMasterOrganizationRequiresCurrentSuperadminPassword(t *testing.T) {
	db := &organizationDeleteAuthDB{admin: database.Admin{ID: 1, Role: "superadmin", PasswordHash: hashAdminPassword("correct-password")}}
	router := &_router{
		db:              db,
		loginRateByIP:   map[string][]time.Time{},
		loginRateByUser: map[string][]time.Time{},
		adminSessions:   map[string]adminSession{},
		partnerSessions: map[string]adminSession{},
	}
	requestContext := reqcontext.RequestContext{AdminID: 1, AdminRole: "superadmin", AdminUsername: "master", Logger: logrus.New()}
	mux := chi.NewRouter()
	mux.Delete("/admin/master/organizations/{id}", func(w http.ResponseWriter, r *http.Request) {
		router.deleteMasterOrganization(w, r, requestContext)
	})
	request := func(password string) *http.Request {
		return httptest.NewRequest(http.MethodDelete, "/admin/master/organizations/7", strings.NewReader(`{"password":"`+password+`"}`))
	}

	wrong := httptest.NewRecorder()
	mux.ServeHTTP(wrong, request("wrong-password"))
	if wrong.Code != http.StatusForbidden || db.deleteCalls != 0 {
		t.Fatalf("wrong password: status=%d deletions=%d", wrong.Code, db.deleteCalls)
	}

	db.admin.Role = "staff"
	notSuperadmin := httptest.NewRecorder()
	mux.ServeHTTP(notSuperadmin, request("correct-password"))
	if notSuperadmin.Code != http.StatusForbidden || db.deleteCalls != 0 {
		t.Fatalf("revoked role: status=%d deletions=%d", notSuperadmin.Code, db.deleteCalls)
	}

	db.admin.Role = "superadmin"
	valid := httptest.NewRecorder()
	mux.ServeHTTP(valid, request("correct-password"))
	if valid.Code != http.StatusOK || db.deleteCalls != 1 {
		t.Fatalf("correct password: status=%d deletions=%d", valid.Code, db.deleteCalls)
	}
}
