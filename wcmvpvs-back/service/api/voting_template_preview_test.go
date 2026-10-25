package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func votingTemplatePreviewRouter(transport votingTransport) *_router {
	rt := staffAccessRouter("superadmin")
	rt.twilioMessaging = newTwilioMessagingClient(twilioMessagingConfig{AccountSID: "ACtest", AuthToken: "test-token", WhatsAppMessagingServiceSID: "MGtest"})
	rt.twilioMessaging.httpClient.Transport = transport
	return rt
}

func votingTemplatePreviewRequest(rt *_router, body, slug string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/admin/voting-settings/test", strings.NewReader(body))
	r.Header.Set("X-Organization-Slug", slug)
	r.Header.Set("Content-Type", "application/json")
	r.AddCookie(&http.Cookie{Name: adminCookieName(), Value: "session"})
	w := httptest.NewRecorder()
	rt.wrapAdmin(rt.sendVotingTemplateTest)(w, r)
	return w
}

func votingTemplatePreviewPayload(sid, phone string) string {
	data, _ := json.Marshal(map[string]string{"template_sid": sid, "phone": phone})
	return string(data)
}

func TestVotingTemplatePreviewUsesEnteredSIDAndProductionVariablesWithoutSaving(t *testing.T) {
	calls := 0
	sid := "HX" + strings.Repeat("b", 32)
	rt := votingTemplatePreviewRouter(func(r *http.Request) (*http.Response, error) {
		calls++
		account, token, ok := r.BasicAuth()
		if !ok || account != "ACtest" || token != "test-token" {
			t.Fatal("server credentials missing")
		}
		if r.URL.Host != "api.twilio.com" || r.Method != http.MethodPost {
			t.Fatalf("unexpected provider request: %v", r)
		}
		body, _ := io.ReadAll(r.Body)
		form, err := url.ParseQuery(string(body))
		if err != nil {
			t.Fatal(err)
		}
		if form.Get("ContentSid") != sid || form.Get("To") != "whatsapp:+393334443322" || form.Get("MessagingServiceSid") != "MGtest" {
			t.Fatalf("template request: %v", form)
		}
		if form.Has("Body") || form.Has("MediaUrl") {
			t.Fatal("template sent as a freeform message")
		}
		var variables map[string]string
		if err := json.Unmarshal([]byte(form.Get("ContentVariables")), &variables); err != nil {
			t.Fatal(err)
		}
		want := map[string]string{"1": "Mario Rossi (TEST)", "2": "42,75", "3": "Maglia autografata", "4": "0123", "5": "Pallone firmato", "6": "0456", "7": "Buono merchandising", "8": "0789"}
		if !reflect.DeepEqual(variables, want) {
			t.Fatalf("incorrect test values: %+v", variables)
		}
		return &http.Response{StatusCode: http.StatusCreated, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"sid":"SMpreview","status":"queued"}`))}, nil
	})
	// The mock database only implements organization lookup for authentication.
	// Accessing/saving settings, events, winners or recipients would panic.
	w := votingTemplatePreviewRequest(rt, votingTemplatePreviewPayload(" "+sid+" ", "+39 (333) 444-3322"), "club")
	if w.Code != http.StatusAccepted || calls != 1 {
		t.Fatalf("preview: status=%d calls=%d body=%s", w.Code, calls, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "SMpreview") || !strings.Contains(w.Body.String(), "accettato da Twilio") || strings.Contains(w.Body.String(), "+393334443322") {
		t.Fatalf("incorrect acceptance response: %s", w.Body.String())
	}
}

func TestVotingTemplatePreviewRejectsUnauthorizedAndInvalidRequestsBeforeSend(t *testing.T) {
	calls := 0
	transport := votingTransport(func(*http.Request) (*http.Response, error) {
		calls++
		t.Fatal("invalid request sent a message")
		return nil, errors.New("unexpected send")
	})
	valid := votingTemplatePreviewPayload("HX"+strings.Repeat("a", 32), "+393334443322")
	for _, role := range []string{"staff", "bar", "partner", "master", ""} {
		rt := votingTemplatePreviewRouter(transport)
		session := rt.adminSessions["session"]
		session.Role = role
		rt.adminSessions["session"] = session
		if w := votingTemplatePreviewRequest(rt, valid, "club"); w.Code != http.StatusForbidden {
			t.Fatalf("role %q: %d %s", role, w.Code, w.Body.String())
		}
	}
	rt := votingTemplatePreviewRouter(transport)
	for _, body := range []string{
		`{}`, `[]`, `{"template_sid":"wrong","phone":"+393334443322"}`,
		votingTemplatePreviewPayload("HX"+strings.Repeat("g", 32), "+393334443322"),
		votingTemplatePreviewPayload("AC"+strings.Repeat("a", 32), "+393334443322"),
		votingTemplatePreviewPayload("HX"+strings.Repeat("a", 32), "3334443322"),
		votingTemplatePreviewPayload("HX"+strings.Repeat("a", 32), "+000000000"),
		votingTemplatePreviewPayload("HX"+strings.Repeat("a", 32), "+39１２３４５６７８"),
		valid + ` {}`, strings.TrimSuffix(valid, "}") + `,"organization_id":99}`,
		votingTemplatePreviewPayload("HX"+strings.Repeat("a", 32), strings.Repeat("1", 3000)),
	} {
		if w := votingTemplatePreviewRequest(rt, body, "club"); w.Code != http.StatusBadRequest {
			t.Fatalf("accepted invalid payload: %d %s", w.Code, w.Body.String())
		}
	}
	if w := votingTemplatePreviewRequest(rt, valid, "other"); w.Code != http.StatusUnauthorized {
		t.Fatalf("cross-club test allowed: %d", w.Code)
	}
	if w := votingTemplatePreviewRequest(rt, valid, ""); w.Code != http.StatusBadRequest {
		t.Fatalf("unscoped test allowed: %d", w.Code)
	}
	if calls != 0 {
		t.Fatal("validation did not prevent sending")
	}
}

func TestVotingTemplatePreviewReportsProviderFailureAndUncertainSends(t *testing.T) {
	for _, tc := range []struct {
		name, response string
		status         int
		err            error
		message        string
		wantStatus     int
	}{
		{"provider rejection", `{"code":21656,"message":"test-token private provider detail"}`, 400, nil, "codice 21656", http.StatusBadGateway},
		{"provider rate limit", `{"code":20429}`, 429, nil, "codice 20429", http.StatusTooManyRequests},
		{"network ambiguity", "", 0, errors.New("connection interrupted after write"), "incerto", http.StatusBadGateway},
		{"missing receipt", `{"status":"queued"}`, 201, nil, "incerto", http.StatusBadGateway},
		{"failed status", `{"sid":"SMfailed","status":"failed"}`, 201, nil, "problema", http.StatusBadGateway},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			rt := votingTemplatePreviewRouter(func(*http.Request) (*http.Response, error) {
				calls++
				if tc.err != nil {
					return nil, tc.err
				}
				return &http.Response{StatusCode: tc.status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(tc.response))}, nil
			})
			w := votingTemplatePreviewRequest(rt, votingTemplatePreviewPayload("HX"+strings.Repeat("a", 32), "+393334443322"), "club")
			if w.Code != tc.wantStatus || calls != 1 || !strings.Contains(w.Body.String(), tc.message) || strings.Contains(w.Body.String(), "test-token") {
				t.Fatalf("status=%d calls=%d body=%s", w.Code, calls, w.Body.String())
			}
		})
	}
}

func TestVotingTemplatePreviewRequiresConfigurationAndLimitsRepeatedSends(t *testing.T) {
	calls := 0
	rt := votingTemplatePreviewRouter(func(*http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 201, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"sid":"SMpreview","status":"queued"}`))}, nil
	})
	valid := votingTemplatePreviewPayload("HX"+strings.Repeat("a", 32), "+393334443322")
	client := rt.twilioMessaging
	rt.twilioMessaging = nil
	if w := votingTemplatePreviewRequest(rt, valid, "club"); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("missing sender: %d", w.Code)
	}
	rt.twilioMessaging = client
	for i := 0; i < 5; i++ {
		if w := votingTemplatePreviewRequest(rt, valid, "club"); w.Code != http.StatusAccepted {
			t.Fatalf("test %d: %d", i, w.Code)
		}
	}
	w := votingTemplatePreviewRequest(rt, valid, "club")
	if w.Code != http.StatusTooManyRequests || w.Header().Get("Retry-After") != "60" || calls != 5 {
		t.Fatalf("rate limit: status=%d calls=%d", w.Code, calls)
	}
}
