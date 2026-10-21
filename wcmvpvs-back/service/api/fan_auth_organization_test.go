package api

import (
	"database/sql"
	"encoding/json"
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

type organizationOTPTransport struct{ sends, checks int }

func (transport *organizationOTPTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if strings.HasSuffix(r.URL.Path, "/VerificationCheck") {
		transport.checks++
	} else {
		transport.sends++
	}
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"status":"approved"}`))}, nil
}

func TestFanAuthSamePhoneInDifferentOrganizations(t *testing.T) {
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { conn.Close() })
	db, err := database.New(conn)
	if err != nil {
		t.Fatal(err)
	}
	firstClub, err := db.CreateOrganization(database.Organization{Name: "First Club", Slug: "phone-first", IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	secondClub, err := db.CreateOrganization(database.Organization{Name: "Second Club", Slug: "phone-second", IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	const phone = "+393331234567"
	first, err := db.RegisterFan(database.FanRegisterInput{OrganizationID: firstClub.ID, Nickname: "FirstFan", Phone: phone, AcceptedTerms: true, SessionToken: "first-owner", DeviceID: "original-device"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AddFanWalletCoins(first.Profile.ID, 50); err != nil {
		t.Fatal(err)
	}
	transport := &organizationOTPTransport{}
	verify := newTwilioVerifyClient(twilioVerifyConfig{AccountSID: "test", AuthToken: "test", ServiceSID: "test"})
	verify.httpClient = &http.Client{Transport: transport}
	request := func(org int, handler string, body, token string) *httptest.ResponseRecorder {
		t.Helper()
		router := &_router{db: db, twilioVerify: verify, authRateByPhone: map[string][]time.Time{}, authRateByIP: map[string][]time.Time{}}
		r := httptest.NewRequest(http.MethodPost, "/"+handler, strings.NewReader(body))
		r.Header.Set("X-Device-ID", "new-device")
		if token != "" {
			r.Header.Set("X-Fan-Session", token)
		}
		w := httptest.NewRecorder()
		ctx := reqcontext.RequestContext{OrganizationID: org, Logger: logrus.New()}
		switch handler {
		case "start":
			router.postAuthStart(w, r, ctx)
		case "verify":
			router.postAuthVerify(w, r, ctx)
		case "resend":
			router.postAuthResend(w, r, ctx)
		case "register":
			router.postRegisterFan(w, r, ctx)
		case "me":
			router.getFanMe(w, r, ctx)
		}
		return w
	}
	decode := func(w *httptest.ResponseRecorder, status int) map[string]interface{} {
		t.Helper()
		if w.Code != status {
			t.Fatalf("status %d, expected %d: %s", w.Code, status, w.Body.String())
		}
		var payload map[string]interface{}
		if w.Body.Len() > 0 {
			if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
		}
		return payload
	}
	loginBody := `{"phone":"` + phone + `","mode":"login"}`
	registerBody := withFanRegistrationAcceptance(`{"phone":"` + phone + `","mode":"register"}`)
	verifyBody := `{"phone":"` + phone + `","code":"123456"}`
	phoneBody := `{"phone":"` + phone + `"}`
	for _, tc := range []struct{ handler, body string }{{"start", loginBody}, {"verify", verifyBody}, {"resend", phoneBody}} {
		decode(request(secondClub.ID, tc.handler, tc.body, ""), 404)
	}
	if transport.sends != 0 || transport.checks != 0 {
		t.Fatal("another organization's profile reached OTP provider")
	}
	decode(request(secondClub.ID, "start", registerBody, ""), 204)
	pending, err := db.GetFanByPhoneE164(secondClub.ID, phone)
	if err != nil || pending.ID == first.Profile.ID || !pending.RegistrationPending {
		t.Fatalf("separate pending profile: %+v %v", pending, err)
	}
	decode(request(secondClub.ID, "resend", phoneBody, ""), 204)
	data := decode(request(secondClub.ID, "verify", verifyBody, ""), 200)
	secondToken := data["token"].(string)
	data = decode(request(secondClub.ID, "me", "", secondToken), 200)
	if data["registered"] != false || data["registration_pending"] != true {
		t.Fatalf("pending profile treated as complete: %+v", data)
	}
	data = decode(request(secondClub.ID, "register", withFanRegistrationAcceptance(`{"phone":"`+phone+`","nickname":"SecondFan","accepted_terms":true}`), secondToken), 200)
	if int(data["user"].(map[string]interface{})["id"].(float64)) != pending.ID || data["wallet"] != float64(0) {
		t.Fatalf("registration merged profiles or wallets: %+v", data)
	}
	decode(request(secondClub.ID, "start", registerBody, ""), 409)
	for _, tc := range []struct{ org, id, wallet int }{{firstClub.ID, first.Profile.ID, 50}, {secondClub.ID, pending.ID, 0}} {
		decode(request(tc.org, "start", loginBody, ""), 204)
		token := decode(request(tc.org, "verify", verifyBody, ""), 200)["token"].(string)
		me := decode(request(tc.org, "me", "", token), 200)
		if me["registered"] != true || me["wallet"] != float64(tc.wallet) || int(me["user"].(map[string]interface{})["id"].(float64)) != tc.id {
			t.Fatalf("login resolved the wrong club profile: %+v", me)
		}
	}
	decode(request(secondClub.ID, "me", "", "first-owner"), 403)
	var device string
	if err := conn.QueryRow(`SELECT device_id FROM fan_sessions WHERE token='first-owner'`).Scan(&device); err != nil || device != "original-device" {
		t.Fatalf("wrong-club request modified another session: %q %v", device, err)
	}
	for _, handler := range []string{"start", "verify", "resend"} {
		decode(request(0, handler, registerBody, ""), 400)
	}
}
