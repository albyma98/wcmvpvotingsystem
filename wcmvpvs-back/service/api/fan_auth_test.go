package api

import (
	"database/sql"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/sirupsen/logrus"
)

type pendingFanAuthDB struct {
	database.AppDatabase
	profile database.FanProfile
	created int
}

func (db *pendingFanAuthDB) GetFanByPhoneE164(string) (database.FanProfile, error) {
	if db.profile.ID == 0 {
		return database.FanProfile{}, sql.ErrNoRows
	}
	return db.profile, nil
}
func (db *pendingFanAuthDB) CreateFanWithPhoneE164(string) (database.FanProfile, error) {
	db.created++
	db.profile = database.FanProfile{ID: 1}
	return db.profile, nil
}

type pendingFanAuthTransport struct{ sends int }

func (transport *pendingFanAuthTransport) RoundTrip(*http.Request) (*http.Response, error) {
	transport.sends++
	if transport.sends == 1 {
		return nil, errors.New("temporary SMS failure")
	}
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{}`))}, nil
}

func TestFanAuthRegistrationCanRetryPendingPhone(t *testing.T) {
	db := &pendingFanAuthDB{}
	transport := &pendingFanAuthTransport{}
	verify := newTwilioVerifyClient(twilioVerifyConfig{AccountSID: "test", AuthToken: "test", ServiceSID: "test"})
	verify.httpClient = &http.Client{Transport: transport}
	router := &_router{db: db, twilioVerify: verify, authRateByPhone: map[string][]time.Time{}, authRateByIP: map[string][]time.Time{}}
	start := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/auth/start", strings.NewReader(`{"phone":"+393331234567","mode":"register"}`))
		router.postAuthStart(w, r, reqcontext.RequestContext{OrganizationID: 7, Logger: logrus.New()})
		return w
	}
	if w := start(); w.Code != http.StatusBadGateway {
		t.Fatalf("SMS failure: %d %s", w.Code, w.Body.String())
	}
	if w := start(); w.Code != http.StatusNoContent {
		t.Fatalf("pending phone retry: %d %s", w.Code, w.Body.String())
	}
	if db.created != 1 || transport.sends != 2 {
		t.Fatalf("retry recreated profile or skipped sending: %d/%d", db.created, transport.sends)
	}
	db.profile.OrganizationID = 7
	if w := start(); w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "PHONE_ALREADY_EXISTS") {
		t.Fatalf("completed profile requires login: %d %s", w.Code, w.Body.String())
	}
}
