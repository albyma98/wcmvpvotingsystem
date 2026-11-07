package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
)

func merchStripeTestConfig() Config {
	return Config{
		StripeSecretKey: "sk_test_fixture", MerchStripeWebhookSecret: "whsec_fixture",
		MerchStripeConnectAccounts: `{"club-uno":"acct_club1","club-due":"acct_club2","club-tre":"acct_platform","club-quattro":"acct_express"}`,
		MerchPublicBaseURL:         "https://portal.example.com",
	}
}

func TestMerchStripeAccountConfiguration(t *testing.T) {
	for _, raw := range []string{
		`[]`, `null`, `{"club-uno":"acct_same","club-due":"acct_same"}`,
		`{"Club Uno":"acct_club1"}`, `{"club":"pk_test_not_a_connected_account"}`,
	} {
		if _, err := parseMerchStripeAccounts(raw); err == nil {
			t.Errorf("accepted invalid account mapping %s", raw)
		}
	}
	for _, tc := range []struct {
		name, key, base, webhook string
		configured               bool
	}{
		{"sandbox localhost", "sk_test_fixture", "http://localhost:5173", "whsec_fixture", true},
		{"sandbox loopback", "rk_test_fixture", "http://127.0.0.1:5173", "whsec_fixture", true},
		{"live https", "sk_live_fixture", "https://portal.example.com", "whsec_fixture", true},
		{"live insecure", "sk_live_fixture", "http://localhost:5173", "whsec_fixture", false},
		{"public HTTP", "sk_test_fixture", "http://portal.example.com", "whsec_fixture", false},
		{"invalid base", "sk_test_fixture", "https://portal.example.com?x=1", "whsec_fixture", false},
		{"public key", "pk_test_fixture", "https://portal.example.com", "whsec_fixture", false},
		{"missing webhook", "sk_test_fixture", "https://portal.example.com", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := merchStripeTestConfig()
			cfg.StripeSecretKey, cfg.MerchPublicBaseURL, cfg.MerchStripeWebhookSecret = tc.key, tc.base, tc.webhook
			rt := &_router{merchStripe: newMerchStripeClient(cfg)}
			if rt.merchStripeConfigured() != tc.configured {
				t.Fatalf("configured = %v, want %v", !tc.configured, tc.configured)
			}
			if account, status := rt.merchStripeStatus(context.Background(), "absent-club"); account != "" || status != "NOT_CONFIGURED" {
				t.Fatalf("unconfigured club: %q %q", account, status)
			}
		})
	}
}

func TestMerchStripeCheckoutDirectCharge(t *testing.T) {
	rt := &_router{merchStripe: newMerchStripeClient(merchStripeTestConfig())}
	expiresAt := time.Now().Add(45 * time.Minute).Unix()
	order := merchOrder{
		ID: "fixture-order", StripeAccount: "acct_club1", Email: "fan@example.com", ExpiresAt: expiresAt,
		merchQuote: merchQuote{
			Currency: "EUR", TotalCents: 3900, ShippingCents: 500,
			Items: []merchOrderItem{{Name: "Maglia", VariantName: "M / Blu", Quantity: 2, UnitCents: 1700}},
		},
	}
	type wire struct{ Account, Idempotency, Body string }
	requests := make(chan wire, 2)
	stripe := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, _, ok := r.BasicAuth()
		if !ok || key != "sk_test_fixture" || r.Header.Get("Stripe-Version") != merchStripeAPIVersion {
			t.Error("invalid Stripe authentication or API version")
		}
		switch r.URL.Path {
		case "/account":
			fmt.Fprint(w, `{"id":"acct_platform"}`)
		case "/accounts/acct_club1", "/accounts/acct_platform":
			if r.Method != http.MethodGet || r.Header.Get("Stripe-Account") != "" {
				t.Error("account lookup must be scoped to the platform's connected accounts")
			}
			fmt.Fprintf(w, `{"id":%q,"charges_enabled":true,"payouts_enabled":true,"capabilities":{"card_payments":"active"},"controller":{"stripe_dashboard":{"type":"full"},"losses":{"payments":"stripe"},"fees":{"payer":"account"}}}`, strings.TrimPrefix(r.URL.Path, "/accounts/"))
		case "/accounts/acct_club2":
			fmt.Fprint(w, `{"id":"acct_club2","charges_enabled":true,"payouts_enabled":false,"capabilities":{"card_payments":"active"},"controller":{"stripe_dashboard":{"type":"full"},"losses":{"payments":"stripe"},"fees":{"payer":"account"}}}`)
		case "/accounts/acct_express":
			fmt.Fprint(w, `{"id":"acct_express","charges_enabled":true,"payouts_enabled":true,"capabilities":{"card_payments":"active"},"controller":{"stripe_dashboard":{"type":"express"},"losses":{"payments":"application"},"fees":{"payer":"application"}}}`)
		case "/checkout/sessions":
			if r.Method != http.MethodPost || r.ParseForm() != nil {
				t.Error("checkout session request is not a valid form POST")
			}
			select {
			case requests <- wire{r.Header.Get("Stripe-Account"), r.Header.Get("Idempotency-Key"), r.PostForm.Encode()}:
			default:
				t.Error("unexpected extra Stripe checkout request")
			}
			for k, want := range map[string]string{
				"mode": "payment", "client_reference_id": order.ID, "expires_at": strconv.FormatInt(expiresAt, 10),
				"customer_email": "fan@example.com", "customer_creation": "if_required", "adaptive_pricing[enabled]": "false",
				"allowed_payment_method_types[0]": "card", "allowed_payment_method_types[1]": "link",
				"line_items[0][price_data][currency]": "eur", "line_items[0][price_data][unit_amount]": "1700",
				"line_items[0][price_data][product_data][name]": "Maglia · M / Blu", "line_items[0][quantity]": "2",
				"line_items[1][price_data][unit_amount]": "500",
				"success_url":                            "https://portal.example.com/club-uno/new/orders?order=fixture-order&payment=return",
				"cancel_url":                             "https://portal.example.com/club-uno/new/orders?order=fixture-order&payment=cancel",
			} {
				if got := r.PostForm.Get(k); got != want {
					t.Errorf("%s = %q, want %q", k, got, want)
				}
			}
			if r.PostForm.Has("payment_intent_data[transfer_data][destination]") ||
				r.PostForm.Has("shipping_address_collection[allowed_countries][0]") {
				t.Error("unexpected transfer destination or duplicate shipping address collection")
			}
			fmt.Fprint(w, `{"id":"cs_test_fixture","url":"https://checkout.stripe.com/c/pay/cs_test_fixture","mode":"payment","status":"open","payment_status":"unpaid","amount_total":3900,"currency":"eur","client_reference_id":"fixture-order","metadata":{"arenaboostx_merch_order":"fixture-order"},"livemode":false}`)
		default:
			t.Errorf("unexpected Stripe endpoint: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer stripe.Close()
	rt.merchStripe.apiBase = stripe.URL

	for attempt := 0; attempt < 2; attempt++ {
		id, redirect, err := rt.createMerchStripeCheckout(context.Background(), "club-uno", order, "private-guest-token")
		if err != nil || id != "cs_test_fixture" || redirect != "https://checkout.stripe.com/c/pay/cs_test_fixture" {
			t.Fatalf("create Checkout = %q %q %v", id, redirect, err)
		}
	}
	first, retry := <-requests, <-requests
	if first.Account != "acct_club1" || first.Idempotency != "merch-checkout-fixture-order" || first != retry ||
		strings.Contains(first.Body, "private-guest-token") {
		t.Fatalf("unsafe checkout scope or retry: %+v %+v", first, retry)
	}
	if _, _, err := rt.createMerchStripeCheckout(context.Background(), "club-due", order, ""); !errors.Is(err, errMerchStripeUnavailable) {
		t.Fatal("mismatched merchant account accepted")
	}
	if _, status := rt.merchStripeStatus(context.Background(), "club-due"); status != "UNAVAILABLE" {
		t.Fatalf("account with disabled payouts is marked %q", status)
	}
	if _, status := rt.merchStripeStatus(context.Background(), "club-tre"); status != "UNAVAILABLE" {
		t.Fatalf("platform account is marked %q", status)
	}
	if _, status := rt.merchStripeStatus(context.Background(), "club-quattro"); status != "UNAVAILABLE" {
		t.Fatalf("Express account is marked %q", status)
	}
	order.TotalCents++
	if _, _, err := rt.createMerchStripeCheckout(context.Background(), "club-uno", order, ""); err == nil {
		t.Fatal("inconsistent order total accepted")
	}
}

func TestMerchStripeSessionEventsVerifyPaymentState(t *testing.T) {
	total := int64(3900)
	for _, tc := range []struct {
		kind, stripeStatus, paymentStatus, intent, want string
	}{
		{"checkout.session.completed", "complete", "paid", "pi_fixture", "PAID"},
		{"checkout.session.completed", "complete", "unpaid", "pi_fixture", "PENDING_ONLINE"},
		{"checkout.session.completed", "complete", "paid", "", ""},
		{"checkout.session.completed", "open", "unpaid", "", ""},
		{"checkout.session.async_payment_succeeded", "complete", "paid", "pi_fixture", "PAID"},
		{"checkout.session.async_payment_succeeded", "complete", "unpaid", "pi_fixture", ""},
		{"checkout.session.async_payment_failed", "complete", "unpaid", "pi_fixture", "FAILED"},
		{"checkout.session.expired", "expired", "unpaid", "", "EXPIRED"},
	} {
		t.Run(tc.kind+"/"+tc.paymentStatus+"/"+tc.want, func(t *testing.T) {
			session := merchStripeSession{
				ID: "cs_test_fixture", Mode: "payment", Status: tc.stripeStatus, PaymentStatus: tc.paymentStatus,
				PaymentIntent: tc.intent, AmountTotal: &total, Currency: "eur", ClientReferenceID: "order",
				Metadata: map[string]string{merchStripeOrderMetadata: "order"},
			}
			event, ok := merchStripeSessionEvent("acct_club1", "evt_test", tc.kind, session)
			if ok != (tc.want != "") || event.Status != tc.want {
				t.Fatalf("unexpected event: %+v, valid=%v", event, ok)
			}
			if ok && (event.TotalCents != total || event.Currency != "EUR" || event.OrderID != "order" || event.AccountID != "acct_club1") {
				t.Fatalf("wrong event scope: %+v", event)
			}
			session.ClientReferenceID = "another-order"
			if _, ok := merchStripeSessionEvent("acct_club1", "evt_test", tc.kind, session); ok {
				t.Fatal("checkout session with mismatched order metadata accepted")
			}
		})
	}
}

func merchStripeTestSignature(payload, secret string, at time.Time) string {
	timestamp := strconv.FormatInt(at.Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = fmt.Fprintf(mac, "%s.%s", timestamp, payload)
	return "t=" + timestamp + ",v1=" + hex.EncodeToString(mac.Sum(nil))
}

func TestMerchStripeWebhookSignatureAndScope(t *testing.T) {
	rt := &_router{merchStripe: newMerchStripeClient(merchStripeTestConfig())}
	now := time.Now()
	for _, tc := range []struct {
		name, account, key string
		live               bool
		age                time.Duration
		want               int
	}{
		{"signed, rotated keys", "acct_fixture", "whsec_fixture", false, 0, http.StatusNoContent},
		{"wrong secret", "acct_fixture", "whsec_bar", false, 0, http.StatusBadRequest},
		{"stale", "acct_fixture", "whsec_fixture", false, -6 * time.Minute, http.StatusBadRequest},
		{"future", "acct_fixture", "whsec_fixture", false, 6 * time.Minute, http.StatusBadRequest},
		{"different mode", "acct_fixture", "whsec_fixture", true, 0, http.StatusNoContent},
		{"platform event", "", "whsec_fixture", false, 0, http.StatusNoContent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload := fmt.Sprintf(`{"id":"evt_test","account":%q,"livemode":%t,"type":"customer.created","data":{"object":{}}}`, tc.account, tc.live)
			req := httptest.NewRequest(http.MethodPost, "/merch/stripe/webhook", strings.NewReader(payload))
			req.Header.Set("Stripe-Signature", merchStripeTestSignature(payload, tc.key, now.Add(tc.age))+",v1=previous_signature")
			response := httptest.NewRecorder()
			rt.handleMerchStripeWebhook(response, req, reqcontext.RequestContext{})
			if response.Code != tc.want {
				t.Fatalf("HTTP %d, want %d", response.Code, tc.want)
			}
		})
	}
}

func TestMerchStripeRefundUsesImmutableConnectedAccount(t *testing.T) {
	rt := &_router{merchStripe: newMerchStripeClient(merchStripeTestConfig())}
	stripe := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/refunds" || r.Header.Get("Stripe-Account") != "acct_original" ||
			r.Header.Get("Idempotency-Key") != "merch-refund-fixture" || r.ParseForm() != nil ||
			r.PostForm.Get("payment_intent") != "pi_fixture" || r.PostForm.Get("metadata["+merchStripeOrderMetadata+"]") != "fixture" {
			t.Errorf("refund not scoped to original direct charge: %+v", r)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "re_fixture", "payment_intent": "pi_fixture", "currency": "eur", "amount": 3900, "status": "pending",
		})
	}))
	defer stripe.Close()
	rt.merchStripe.apiBase = stripe.URL
	order := merchOrder{
		ID: "fixture", StripeAccount: "acct_original", StripePaymentIntent: "pi_fixture",
		merchQuote: merchQuote{Currency: "EUR", TotalCents: 3900},
	}
	if err := rt.refundMerchStripe(context.Background(), order); err != nil {
		t.Fatalf("create refund: %v", err)
	}
	refund := merchStripeRefund{ID: "re_fixture", PaymentIntent: "pi_fixture", Currency: "eur", Amount: 3900, Status: "pending"}
	if _, ok := merchStripeRefundEvent(order.StripeAccount, "evt_refund", refund); ok {
		t.Fatal("pending refund accepted as completed")
	}
	refund.Status = "succeeded"
	if event, ok := merchStripeRefundEvent(order.StripeAccount, "evt_refund", refund); !ok ||
		event.Status != "REFUNDED" || event.RefundedCents != order.TotalCents || event.RefundID != refund.ID {
		t.Fatalf("unexpected refund event: %+v, valid=%v", event, ok)
	}
}

func TestMerchStripeSignedWebhooksSettleExactlyOnce(t *testing.T) {
	f := newMerchTestFixture(t)
	rt := f.rt
	rt.merchStripe = newMerchStripeClient(merchStripeTestConfig())
	ctx := context.Background()
	product := f.product(t, 2)
	input := f.order(t, product, "online")
	order, err := rt.createMerchOrder(ctx, f.org.ID, "acct_club1", input, database.FanProfile{}, "192.0.2.10")
	if err != nil {
		t.Fatal(err)
	}
	session := map[string]interface{}{
		"id": "cs_test_fixture", "mode": "payment", "status": "complete", "payment_status": "unpaid",
		"amount_total": order.TotalCents, "currency": "eur", "client_reference_id": order.ID, "livemode": false,
		"payment_intent": "pi_fixture", "metadata": map[string]string{merchStripeOrderMetadata: order.ID},
		"customer_details": map[string]string{"name": "Fan Fixture", "email": "fan@example.com"},
	}
	post := func(eventID, account, kind string, object map[string]interface{}, expectedHTTP int) {
		t.Helper()
		body, err := json.Marshal(map[string]interface{}{
			"id": eventID, "account": account, "livemode": false, "created": time.Now().Unix(), "type": kind,
			"data": map[string]interface{}{"object": object},
		})
		if err != nil {
			t.Fatal(err)
		}
		merchRequest(t, rt.router, http.MethodPost, "/merch/stripe/webhook", strings.NewReader(string(body)),
			map[string]string{"Stripe-Signature": merchStripeTestSignature(string(body), "whsec_fixture", time.Now())}, expectedHTTP)
	}
	getOrder := func() merchOrder {
		t.Helper()
		current, err := loadMerchOrder(ctx, f.conn, f.org.ID, order.ID)
		if err != nil {
			t.Fatal(err)
		}
		if err := loadMerchOrderAudit(ctx, f.conn, &current); err != nil {
			t.Fatal(err)
		}
		return current
	}

	post("evt_checkout_processing", "acct_club1", "checkout.session.completed", session, http.StatusNoContent)
	if got := getOrder(); got.PaymentStatus != "PENDING_ONLINE" || !got.StripeAwaiting || len(got.Payments) != 0 {
		t.Fatalf("unsettled Checkout was treated as paid: %+v", got)
	}
	session["payment_status"] = "paid"
	session["amount_total"] = order.TotalCents + 1
	post("evt_wrong_total", "acct_club1", "checkout.session.async_payment_succeeded", session, http.StatusInternalServerError)
	session["amount_total"] = order.TotalCents
	post("evt_wrong_account", "acct_club2", "checkout.session.async_payment_succeeded", session, http.StatusNoContent)
	if got := getOrder(); got.PaymentStatus != "PENDING_ONLINE" {
		t.Fatal("another connected account changed the order")
	}
	post("evt_checkout_paid", "acct_club1", "checkout.session.async_payment_succeeded", session, http.StatusNoContent)
	post("evt_checkout_paid", "acct_club1", "checkout.session.async_payment_succeeded", session, http.StatusNoContent)
	if got := getOrder(); got.PaymentStatus != "PAID" || got.FulfillmentStatus != "CONFIRMED" || len(got.Payments) != 1 {
		t.Fatalf("checkout did not settle exactly once: %+v", got)
	}
	session["payment_status"], session["status"] = "unpaid", "expired"
	post("evt_stale_expiry", "acct_club1", "checkout.session.expired", session, http.StatusNoContent)
	if got := getOrder(); got.PaymentStatus != "PAID" || len(got.Payments) != 1 {
		t.Fatalf("stale expiry reverted a settled payment: %+v", got)
	}
	var stock int
	if err := f.conn.QueryRowContext(ctx, "SELECT stock FROM merch_products WHERE id=?", product.ID).Scan(&stock); err != nil || stock != 2-input.Items[0].Quantity {
		t.Fatalf("inventory was not consumed exactly once: %d %v", stock, err)
	}

	refund := map[string]interface{}{
		"id": "re_fixture", "amount": order.TotalCents, "currency": "eur", "payment_intent": "pi_fixture", "status": "pending",
	}
	post("evt_refund_pending", "acct_club1", "refund.created", refund, http.StatusNoContent)
	if got := getOrder(); got.PaymentStatus != "PAID" || len(got.Payments) != 1 {
		t.Fatalf("pending refund was treated as settled: %+v", got)
	}
	refund["status"] = "succeeded"
	post("evt_refund_succeeded", "acct_club1", "refund.updated", refund, http.StatusNoContent)
	post("evt_refund_succeeded_again", "acct_club1", "refund.updated", refund, http.StatusNoContent)
	if got := getOrder(); got.PaymentStatus != "REFUNDED" || got.FulfillmentStatus != "CANCELLED" || len(got.Payments) != 2 {
		t.Fatalf("refund did not settle exactly once: %+v", got)
	}
}
