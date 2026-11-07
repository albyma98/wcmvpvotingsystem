package api

import (
	"context"
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

type merchRecoveryTransport func(*http.Request) (*http.Response, error)

func (f merchRecoveryTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestMerchRecoveryRequiresOrderAccessAndVerifiedPhone(t *testing.T) {
	ctx := context.Background()
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetMaxOpenConns(1)
	db, err := database.New(conn)
	if err != nil {
		t.Fatal(err)
	}
	org, err := db.CreateOrganization(database.Organization{Name: "Fixture Shop", Slug: "fixture-shop", IsActive: true})
	if err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		TwilioAccountSID: "fixture", TwilioAuthToken: "fixture",
		TwilioVerifySID: "VA" + strings.Repeat("a", 32), MerchTwilioVerifySID: "VA" + strings.Repeat("b", 32),
		MerchPublicBaseURL: "https://shop.example",
	}
	rt := &_router{
		router: chi.NewRouter(), db: db, baseLogger: logrus.New(),
		merchStripe: newMerchStripeClient(cfg), merchVerify: newMerchVerifyClient(cfg),
		twilioMessaging: newTwilioMessagingClient(twilioMessagingConfig{
			AccountSID: "fixture", AuthToken: "fixture", MessagingServiceSID: "fixture",
		}),
	}
	if err := rt.ensureMerchMigration(); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(`INSERT INTO merch_settings(organization_id,config) VALUES (?,?)`,
		org.ID, `{"enabled":true,"sms_notifications":true}`); err != nil {
		t.Fatal(err)
	}
	p := merchProduct{Name: "Fixture scarf", PriceCents: 1800, Stock: 2, Published: true, MaxQuantity: 2, SaleMode: "inherit"}
	if err := rt.saveMerchProduct(ctx, org.ID, org.Slug, "", &p); err != nil {
		t.Fatal(err)
	}
	const phone = "+393331234567"
	expected := int64(1800)
	order, err := rt.createMerchOrder(ctx, org.ID, "", merchCheckoutInput{
		Mode: "offline", Fulfillment: "pickup", Items: []merchLine{{ProductID: p.ID, Quantity: 1}},
		Name: "Guest Example", Phone: phone, RequestID: "fixture-recovery-checkout-123", ExpectedTotalCents: &expected,
	}, database.FanProfile{}, "192.0.2.1")
	if err != nil {
		t.Fatal(err)
	}
	if !rt.merchRecoveryAvailable(ctx, order) {
		t.Fatal("guest should be offered recovery")
	}

	starts, checks, sends := 0, 0, 0
	challengeID := "VE" + strings.Repeat("1", 32)
	rt.merchVerify.httpClient = &http.Client{Transport: merchRecoveryTransport(func(r *http.Request) (*http.Response, error) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		switch {
		case strings.HasSuffix(r.URL.Path, "/Verifications"):
			starts++
			if r.Form.Get("To") != phone {
				t.Fatalf("verification phone: %s", r.Form.Get("To"))
			}
			return merchRecoveryResponse(`{"sid":"` + challengeID + `"}`), nil
		case strings.HasSuffix(r.URL.Path, "/VerificationCheck"):
			checks++
			if r.Form.Get("VerificationSid") != challengeID || r.Form.Get("To") != "" {
				t.Fatal("verification was not bound to the stored challenge")
			}
			if r.Form.Get("Code") == "123456" {
				return merchRecoveryResponse(`{"status":"approved"}`), nil
			}
			return merchRecoveryResponse(`{"status":"pending"}`), nil
		default:
			t.Fatalf("unexpected verify route: %s", r.URL.Path)
			return nil, nil
		}
	})}
	access := merchGuestToken(order.ID, order.RequestID)
	rt.twilioMessaging.httpClient = &http.Client{Transport: merchRecoveryTransport(func(r *http.Request) (*http.Response, error) {
		sends++
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("To") != phone {
			t.Fatal("message sent to a different phone")
		}
		_, link, ok := strings.Cut(r.Form.Get("Body"), "https://")
		if !ok {
			t.Fatal("message is missing the private link")
		}
		u, err := url.Parse("https://" + link)
		if err != nil {
			t.Fatal(err)
		}
		capability, err := url.ParseQuery(u.Fragment)
		if err != nil || u.RawQuery != "" || u.Path != "/fixture-shop/new/orders" ||
			capability.Get("order") != order.ID || capability.Get("access") != access {
			t.Fatal("recovery capability was not scoped to the order in the URL fragment")
		}
		return merchRecoveryResponse(`{"sid":"fixture-message"}`), nil
	})}

	registerMerchRecoveryRoutes(rt, "/v1/clubs/{slug}/merch")
	request := func(step, token, body string, status int) {
		t.Helper()
		path := "/v1/clubs/fixture-shop/merch/orders/" + order.ID + "/recovery/" + step
		r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		r.Header.Set("X-Merch-Access", token)
		w := httptest.NewRecorder()
		rt.router.ServeHTTP(w, r)
		if w.Code != status {
			t.Fatalf("%s: HTTP %d, want %d. %s", step, w.Code, status, w.Body.String())
		}
	}

	request("start", "invalid", "", http.StatusNotFound)
	request("start", access, "", http.StatusNoContent)
	request("verify", access, `{"code":"999999"}`, http.StatusBadRequest)
	if err := rt.processMerchNotifications(ctx); err != nil || sends != 0 {
		t.Fatalf("unverified SMS: count %d, err %v", sends, err)
	}
	request("verify", access, `{"code":"123456"}`, http.StatusNoContent)
	request("verify", access, `{"code":"123456"}`, http.StatusNoContent)
	for i := 0; i < 2; i++ {
		if err := rt.processMerchNotifications(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if starts != 1 || checks != 2 || sends != 1 || rt.merchRecoveryAvailable(ctx, order) {
		t.Fatalf("start=%d check=%d send=%d", starts, checks, sends)
	}
}

func merchRecoveryResponse(body string) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}
