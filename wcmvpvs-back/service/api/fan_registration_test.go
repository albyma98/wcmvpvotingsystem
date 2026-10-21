package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
)

func withFanRegistrationAcceptance(body string) string {
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		panic(err)
	}
	payload["accepted_terms"] = true
	payload["age_eligibility_declared"] = true
	payload["minimum_age_declared"] = database.FanMinimumAge
	payload["legal_version"] = database.FanLegalVersion
	data, _ := json.Marshal(payload)
	return string(data)
}

func TestFanRegistrationRejectsMissingDeclarationsBeforeSMSOrDatabase(t *testing.T) {
	router := &_router{}
	for _, tc := range []struct {
		field string
		value interface{}
		code  string
	}{
		{"age_eligibility_declared", false, "AGE_ELIGIBILITY_REQUIRED"},
		{"minimum_age_declared", 13, "AGE_ELIGIBILITY_REQUIRED"},
		{"accepted_terms", false, "TERMS_REQUIRED"},
		{"legal_version", "old", "LEGAL_VERSION_CHANGED"},
	} {
		t.Run(tc.field, func(t *testing.T) {
			var payload map[string]interface{}
			_ = json.Unmarshal([]byte(withFanRegistrationAcceptance(`{"phone":"+393331234567","nickname":"Fan","mode":"register"}`)), &payload)
			payload[tc.field] = tc.value
			body, _ := json.Marshal(payload)
			for _, endpoint := range []string{"start", "register"} {
				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodPost, "/"+endpoint, strings.NewReader(string(body)))
				ctx := reqcontext.RequestContext{OrganizationID: 7}
				if endpoint == "start" {
					router.postAuthStart(w, r, ctx)
				} else {
					router.postRegisterFan(w, r, ctx)
				}
				if w.Code < 400 || !strings.Contains(w.Body.String(), tc.code) {
					t.Fatalf("%s: %d %s", endpoint, w.Code, w.Body.String())
				}
			}
		})
	}
	w := httptest.NewRecorder()
	router.postRegisterFan(w, httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(withFanRegistrationAcceptance(`{"phone":"+393331234567","nickname":"Fan"}`))), reqcontext.RequestContext{OrganizationID: 7})
	if w.Code != 401 || !strings.Contains(w.Body.String(), "PHONE_VERIFICATION_REQUIRED") {
		t.Fatalf("missing OTP session: %d %s", w.Code, w.Body.String())
	}
}

func TestFanRegistrationAudienceIsNotAvailableToClubAdministrators(t *testing.T) {
	router := &_router{}
	for _, handler := range []httpRouterHandler{router.listMarketingAudience, router.getMarketingAudienceFan} {
		w := httptest.NewRecorder()
		handler(w, httptest.NewRequest(http.MethodGet, "/admin/marketing/audience", nil), reqcontext.RequestContext{OrganizationID: 7, AdminRole: "admin"})
		if w.Code != http.StatusForbidden {
			t.Fatalf("contacts exposed: %d", w.Code)
		}
	}
}
