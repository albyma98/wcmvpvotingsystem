package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
)

const merchStripeAPIVersion = "2026-10-28.endive"
const merchStripeOrderMetadata = "arenaboostx_merch_order"

var (
	merchStripeAccountPattern = regexp.MustCompile(`^acct_[a-zA-Z0-9]+$`)
	merchStripeSlugPattern    = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	errMerchStripeUnavailable = errors.New("Stripe Connect unavailable")
)

type merchStripeClient struct {
	key           string
	webhookSecret string
	live          bool
	base          *url.URL
	accounts      map[string]string
	client        *http.Client
	apiBase       string
	accountChecks sync.Map
	platformID    atomic.Value
}

type merchStripeAccountCheck struct {
	expires time.Time
	ready   bool
}

type merchStripeEvent struct {
	EventID, OrderID, AccountID, SessionID, PaymentIntent, Status, Currency string
	RefundID, CustomerName, CustomerEmail, CustomerPhone                    string
	TotalCents, RefundedCents                                               int64
	OccurredAt                                                              int64
}

type merchStripeSession struct {
	ID                string            `json:"id"`
	URL               string            `json:"url"`
	Mode              string            `json:"mode"`
	Status            string            `json:"status"`
	PaymentStatus     string            `json:"payment_status"`
	PaymentIntent     string            `json:"payment_intent"`
	AmountTotal       *int64            `json:"amount_total"`
	Currency          string            `json:"currency"`
	ClientReferenceID string            `json:"client_reference_id"`
	Live              bool              `json:"livemode"`
	Metadata          map[string]string `json:"metadata"`
	CustomerDetails   struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Phone string `json:"phone"`
	} `json:"customer_details"`
}

type merchStripeRefund struct {
	ID            string            `json:"id"`
	Amount        int64             `json:"amount"`
	Currency      string            `json:"currency"`
	PaymentIntent string            `json:"payment_intent"`
	Status        string            `json:"status"`
	Metadata      map[string]string `json:"metadata"`
}

type merchStripeAPIError struct {
	status int
}

func (e merchStripeAPIError) Error() string {
	return fmt.Sprintf("Stripe API returned HTTP %d", e.status)
}

func newMerchStripeClient(cfg Config) *merchStripeClient {
	key := strings.TrimSpace(cfg.StripeSecretKey)
	webhook := strings.TrimSpace(cfg.MerchStripeWebhookSecret)
	live := strings.HasPrefix(key, "sk_live_") || strings.HasPrefix(key, "rk_live_")
	base, err := url.Parse(strings.TrimSpace(cfg.MerchPublicBaseURL))
	if err != nil || !merchStripeReturnScheme(base, live) || base.Hostname() == "" || base.User != nil ||
		base.RawQuery != "" || base.ForceQuery || base.Fragment != "" {
		base = nil
	}
	accounts, err := parseMerchStripeAccounts(cfg.MerchStripeConnectAccounts)
	if err != nil && cfg.Logger != nil {
		cfg.Logger.Warn("invalid MERCH_STRIPE_CONNECT_ACCOUNTS; online merchandising unavailable")
	}
	return &merchStripeClient{
		key: key, webhookSecret: webhook, base: base, accounts: accounts,
		live:    live,
		apiBase: "https://api.stripe.com/v1",
		client: &http.Client{
			Timeout: 8 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func merchStripeReturnScheme(base *url.URL, live bool) bool {
	if base.Scheme == "https" {
		return true
	}
	if live || base.Scheme != "http" {
		return false
	}
	host := base.Hostname()
	return host == "localhost" || net.ParseIP(host).IsLoopback()
}

func parseMerchStripeAccounts(raw string) (map[string]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var accounts map[string]string
	if err := json.Unmarshal([]byte(raw), &accounts); err != nil || accounts == nil {
		return nil, errors.New("expected a JSON object")
	}
	seen := make(map[string]bool, len(accounts))
	for slug, account := range accounts {
		if !merchStripeSlugPattern.MatchString(slug) || !merchStripeAccountPattern.MatchString(account) || seen[account] {
			return nil, errors.New("invalid or duplicate Stripe account assignment")
		}
		seen[account] = true
	}
	return accounts, nil
}

func (c *merchStripeClient) keyConfigured() bool {
	return c != nil && (c.live || strings.HasPrefix(c.key, "sk_test_") || strings.HasPrefix(c.key, "rk_test_"))
}

func (rt *_router) merchStripeConfigured() bool {
	c := rt.merchStripe
	return c.keyConfigured() && strings.HasPrefix(c.webhookSecret, "whsec_") && c.base != nil
}

func (rt *_router) merchStripeAccount(slug string) string {
	if rt.merchStripe == nil {
		return ""
	}
	return rt.merchStripe.accounts[slug]
}

func (rt *_router) merchStripeStatus(ctx context.Context, slug string) (account, status string) {
	account = rt.merchStripeAccount(slug)
	if account == "" || !rt.merchStripeConfigured() {
		return account, "NOT_CONFIGURED"
	}
	if err := rt.validateMerchStripeAccount(ctx, account); err != nil {
		return account, "UNAVAILABLE"
	}
	return account, "READY"
}

func (rt *_router) validateMerchStripeAccount(ctx context.Context, account string) error {
	c := rt.merchStripe
	if !rt.merchStripeConfigured() || !merchStripeAccountPattern.MatchString(account) {
		return errMerchStripeUnavailable
	}
	if cached, ok := c.accountChecks.Load(account); ok {
		check := cached.(merchStripeAccountCheck)
		if time.Now().Before(check.expires) {
			if check.ready {
				return nil
			}
			return errMerchStripeUnavailable
		}
	}
	var current struct {
		ID             string            `json:"id"`
		ChargesEnabled bool              `json:"charges_enabled"`
		PayoutsEnabled bool              `json:"payouts_enabled"`
		Capabilities   map[string]string `json:"capabilities"`
		Controller     struct {
			Dashboard struct {
				Type string `json:"type"`
			} `json:"stripe_dashboard"`
			Losses struct {
				Payments string `json:"payments"`
			} `json:"losses"`
			Fees struct {
				Payer string `json:"payer"`
			} `json:"fees"`
		} `json:"controller"`
	}
	err := c.request(ctx, http.MethodGet, "/accounts/"+account, "", "", nil, &current)
	ready := err == nil && current.ID == account && current.ChargesEnabled && current.PayoutsEnabled &&
		current.Capabilities["card_payments"] == "active" && current.Controller.Dashboard.Type == "full" &&
		current.Controller.Losses.Payments == "stripe" && current.Controller.Fees.Payer == "account"
	if ready {
		platformID, err := c.platformAccount(ctx)
		ready = err == nil && platformID != account
	}
	ttl := time.Minute
	if !ready {
		ttl = 10 * time.Second
	}
	if ctx.Err() == nil {
		c.accountChecks.Store(account, merchStripeAccountCheck{expires: time.Now().Add(ttl), ready: ready})
	}
	if !ready {
		return errMerchStripeUnavailable
	}
	return nil
}

func (c *merchStripeClient) platformAccount(ctx context.Context) (string, error) {
	if cached := c.platformID.Load(); cached != nil {
		return cached.(string), nil
	}
	var account struct {
		ID string `json:"id"`
	}
	if err := c.request(ctx, http.MethodGet, "/account", "", "", nil, &account); err != nil {
		return "", err
	}
	if !merchStripeAccountPattern.MatchString(account.ID) {
		return "", errMerchStripeUnavailable
	}
	c.platformID.Store(account.ID)
	return account.ID, nil
}

func (c *merchStripeClient) request(ctx context.Context, method, path, account, idempotency string, values url.Values, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, method, c.apiBase+path, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.key, "")
	req.Header.Set("Stripe-Version", merchStripeAPIVersion)
	if values != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if account != "" {
		req.Header.Set("Stripe-Account", account)
	}
	if idempotency != "" {
		req.Header.Set("Idempotency-Key", idempotency)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return merchStripeAPIError{status: resp.StatusCode}
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(out)
}

func (c *merchStripeClient) returnURL(slug, orderID, result string) string {
	target := *c.base
	target.Path = strings.TrimRight(target.Path, "/") + "/" + slug + "/new/orders"
	target.RawPath = ""
	query := url.Values{"payment": {result}, "order": {orderID}}
	target.RawQuery = query.Encode()
	return target.String()
}

func merchStripeLine(form url.Values, index int, name, currency string, price int64, quantity int) {
	prefix := fmt.Sprintf("line_items[%d]", index)
	form.Set(prefix+"[price_data][product_data][name]", name)
	form.Set(prefix+"[price_data][currency]", currency)
	form.Set(prefix+"[price_data][unit_amount]", strconv.FormatInt(price, 10))
	form.Set(prefix+"[quantity]", strconv.Itoa(quantity))
}

func (c *merchStripeClient) checkoutForm(slug string, order merchOrder) (url.Values, error) {
	currency := strings.ToLower(order.Currency)
	if order.ID == "" || len(order.Items) == 0 || len(order.Items) > 99 || len(currency) != 3 ||
		order.ShippingCents < 0 || order.ShippingCents > order.TotalCents || order.TotalCents <= 0 {
		return nil, errors.New("invalid merchandise checkout snapshot")
	}
	form := url.Values{
		"mode":                                       {"payment"},
		"customer_creation":                          {"if_required"},
		"adaptive_pricing[enabled]":                  {"false"},
		"allowed_payment_method_types[0]":            {"card"},
		"allowed_payment_method_types[1]":            {"link"},
		"client_reference_id":                        {order.ID},
		"metadata[" + merchStripeOrderMetadata + "]": {order.ID},
		"payment_intent_data[metadata][" + merchStripeOrderMetadata + "]": {order.ID},
		"expires_at":  {strconv.FormatInt(order.ExpiresAt, 10)},
		"success_url": {c.returnURL(slug, order.ID, "return")},
		"cancel_url":  {c.returnURL(slug, order.ID, "cancel")},
	}
	if order.Email != "" {
		form.Set("customer_email", order.Email)
	}
	computed := order.ShippingCents
	for i, item := range order.Items {
		if item.Quantity < 1 || item.UnitCents < 0 || item.UnitCents > (order.TotalCents-computed)/int64(item.Quantity) {
			return nil, errors.New("invalid merchandise checkout item")
		}
		computed += item.UnitCents * int64(item.Quantity)
		name := item.Name
		if item.VariantName != "" {
			name += " · " + item.VariantName
		}
		if runes := []rune(name); len(runes) > 250 {
			name = string(runes[:250])
		}
		merchStripeLine(form, i, name, currency, item.UnitCents, item.Quantity)
	}
	if computed != order.TotalCents {
		return nil, errors.New("merchandise checkout total mismatch")
	}
	if order.ShippingCents > 0 {
		merchStripeLine(form, len(order.Items), "Spedizione", currency, order.ShippingCents, 1)
	}
	return form, nil
}

func (rt *_router) createMerchStripeCheckout(ctx context.Context, slug string, order merchOrder, _ string) (sessionID, checkoutURL string, err error) {
	c := rt.merchStripe
	if order.StripeAccount == "" || order.StripeAccount != rt.merchStripeAccount(slug) ||
		order.ExpiresAt <= time.Now().Unix() {
		return "", "", errMerchStripeUnavailable
	}
	if err := rt.validateMerchStripeAccount(ctx, order.StripeAccount); err != nil {
		return "", "", err
	}
	form, err := c.checkoutForm(slug, order)
	if err != nil {
		return "", "", err
	}
	var session merchStripeSession
	if err := c.request(ctx, http.MethodPost, "/checkout/sessions", order.StripeAccount, "merch-checkout-"+order.ID, form, &session); err != nil {
		return "", "", err
	}
	redirect, err := url.Parse(session.URL)
	if err != nil || redirect.Scheme != "https" || redirect.Hostname() == "" || redirect.User != nil ||
		session.Live != c.live || session.Mode != "payment" || session.Status != "open" ||
		session.ID == "" || session.ClientReferenceID != order.ID || session.Metadata[merchStripeOrderMetadata] != order.ID ||
		session.AmountTotal == nil || *session.AmountTotal != order.TotalCents || !strings.EqualFold(session.Currency, order.Currency) {
		return "", "", errors.New("invalid Stripe checkout session response")
	}
	return session.ID, session.URL, nil
}

func merchStripeSessionEvent(account, eventID, kind string, session merchStripeSession) (merchStripeEvent, bool) {
	orderID := session.Metadata[merchStripeOrderMetadata]
	if orderID == "" || session.ClientReferenceID != orderID || session.Mode != "payment" ||
		session.ID == "" || session.AmountTotal == nil || session.Currency == "" {
		return merchStripeEvent{}, false
	}
	var status string
	switch kind {
	case "checkout.session.completed":
		if session.Status == "complete" && session.PaymentStatus == "unpaid" {
			status = "PENDING_ONLINE"
		}
		if session.Status == "complete" && session.PaymentStatus == "paid" && session.PaymentIntent != "" {
			status = "PAID"
		}
	case "checkout.session.async_payment_succeeded":
		if session.Status == "complete" && session.PaymentStatus == "paid" && session.PaymentIntent != "" {
			status = "PAID"
		}
	case "checkout.session.async_payment_failed":
		if session.Status == "complete" && session.PaymentStatus == "unpaid" {
			status = "FAILED"
		}
	case "checkout.session.expired":
		if session.Status == "expired" && session.PaymentStatus == "unpaid" {
			status = "EXPIRED"
		}
	}
	if status == "" {
		return merchStripeEvent{}, false
	}
	return merchStripeEvent{
		EventID: eventID, OrderID: orderID, AccountID: account, SessionID: session.ID,
		PaymentIntent: session.PaymentIntent, Status: status, Currency: strings.ToUpper(session.Currency),
		TotalCents: *session.AmountTotal, CustomerName: session.CustomerDetails.Name,
		CustomerEmail: session.CustomerDetails.Email, CustomerPhone: session.CustomerDetails.Phone,
	}, true
}

func merchStripeRefundEvent(account, eventID string, refund merchStripeRefund) (merchStripeEvent, bool) {
	if refund.Status != "succeeded" || refund.ID == "" || refund.PaymentIntent == "" || refund.Amount <= 0 || refund.Currency == "" {
		return merchStripeEvent{}, false
	}
	return merchStripeEvent{
		EventID: eventID, OrderID: refund.Metadata[merchStripeOrderMetadata], AccountID: account,
		PaymentIntent: refund.PaymentIntent, Status: "REFUNDED", Currency: strings.ToUpper(refund.Currency),
		RefundID: refund.ID, RefundedCents: refund.Amount,
	}, true
}

func registerMerchStripeRoutes(rt *_router) {
	rt.router.Post("/merch/stripe/webhook", rt.wrap(rt.handleMerchStripeWebhook))
}

func (rt *_router) handleMerchStripeWebhook(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	c := rt.merchStripe
	if !c.keyConfigured() || !strings.HasPrefix(c.webhookSecret, "whsec_") {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	payload, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil || !verifyStripeWebhookSignature(payload, r.Header.Get("Stripe-Signature"), c.webhookSecret) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var envelope struct {
		ID      string `json:"id"`
		Type    string `json:"type"`
		Account string `json:"account"`
		Live    *bool  `json:"livemode"`
		Created int64  `json:"created"`
		Data    struct {
			Object json.RawMessage `json:"object"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil || envelope.ID == "" || envelope.Live == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if *envelope.Live != c.live || !merchStripeAccountPattern.MatchString(envelope.Account) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	var event merchStripeEvent
	var ok bool
	switch envelope.Type {
	case "checkout.session.completed", "checkout.session.async_payment_succeeded",
		"checkout.session.async_payment_failed", "checkout.session.expired":
		var session merchStripeSession
		if err := json.Unmarshal(envelope.Data.Object, &session); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if session.Live != *envelope.Live {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		event, ok = merchStripeSessionEvent(envelope.Account, envelope.ID, envelope.Type, session)
	case "refund.created", "refund.updated", "refund.failed":
		var refund merchStripeRefund
		if err := json.Unmarshal(envelope.Data.Object, &refund); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		event, ok = merchStripeRefundEvent(envelope.Account, envelope.ID, refund)
	}
	if ok {
		event.OccurredAt = envelope.Created
		err := rt.applyMerchStripeEvent(r.Context(), event)
		if errors.Is(err, sql.ErrNoRows) && event.Status == "REFUNDED" && event.OrderID == "" {
			err = rt.applyExternalMerchStripeRefund(r.Context(), event)
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			ctx.Logger.WithError(err).WithField("stripe_event", envelope.ID).Error("cannot apply merchandise Stripe event")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) applyExternalMerchStripeRefund(ctx context.Context, event merchStripeEvent) error {
	// A Dashboard refund can arrive before Checkout attached its payment intent.
	// Only an intent created by this integration carries its local order ID.
	var intent struct {
		ID       string            `json:"id"`
		Live     bool              `json:"livemode"`
		Metadata map[string]string `json:"metadata"`
	}
	c := rt.merchStripe
	path := "/payment_intents/" + url.PathEscape(event.PaymentIntent)
	if err := c.request(ctx, http.MethodGet, path, event.AccountID, "", nil, &intent); err != nil {
		return err
	}
	event.OrderID = intent.Metadata[merchStripeOrderMetadata]
	if intent.ID != event.PaymentIntent || intent.Live != c.live || event.OrderID == "" {
		return sql.ErrNoRows
	}
	return rt.applyMerchStripeEvent(ctx, event)
}

func (rt *_router) reconcileMerchStripeOrder(ctx context.Context, order merchOrder) error {
	c := rt.merchStripe
	if order.StripeSession == "" || !c.keyConfigured() || !merchStripeAccountPattern.MatchString(order.StripeAccount) {
		return nil
	}
	var session merchStripeSession
	path := "/checkout/sessions/" + url.PathEscape(order.StripeSession)
	if err := c.request(ctx, http.MethodGet, path, order.StripeAccount, "", nil, &session); err != nil {
		return err
	}
	if session.ID != order.StripeSession || session.Live != c.live ||
		session.Metadata[merchStripeOrderMetadata] != order.ID || session.ClientReferenceID != order.ID {
		return errors.New("unexpected Stripe session during merchandise reconciliation")
	}
	if session.Status == "open" && order.ExpiresAt <= time.Now().Unix() {
		err := c.request(ctx, http.MethodPost, path+"/expire", order.StripeAccount, "merch-expire-"+order.ID, url.Values{}, &session)
		if err != nil {
			// Checkout may have completed since the GET. Resolve its current state on the next pass.
			return err
		}
	}
	var kind string
	switch session.Status {
	case "complete":
		kind = "checkout.session.completed"
	case "expired":
		kind = "checkout.session.expired"
	}
	event, ok := merchStripeSessionEvent(order.StripeAccount, "reconcile-"+session.ID+"-"+session.Status+"-"+session.PaymentStatus, kind, session)
	if !ok {
		return nil
	}
	return rt.applyMerchStripeEvent(ctx, event)
}

func (rt *_router) refundMerchStripe(ctx context.Context, order merchOrder) error {
	c := rt.merchStripe
	if !c.keyConfigured() || !merchStripeAccountPattern.MatchString(order.StripeAccount) || order.StripePaymentIntent == "" {
		return errMerchStripeUnavailable
	}
	form := url.Values{
		"payment_intent": {order.StripePaymentIntent},
		"metadata[" + merchStripeOrderMetadata + "]": {order.ID},
	}
	var refund merchStripeRefund
	if err := c.request(ctx, http.MethodPost, "/refunds", order.StripeAccount, "merch-refund-"+order.ID, form, &refund); err != nil {
		return err
	}
	if refund.ID == "" || refund.PaymentIntent != order.StripePaymentIntent || !strings.EqualFold(refund.Currency, order.Currency) ||
		refund.Amount < 1 || refund.Amount > order.TotalCents {
		return errors.New("invalid Stripe refund response")
	}
	switch refund.Status {
	case "succeeded":
		event, _ := merchStripeRefundEvent(order.StripeAccount, "refund-response-"+refund.ID, refund)
		event.OrderID = order.ID
		return rt.applyMerchStripeEvent(ctx, event)
	case "pending", "requires_action":
		return nil
	default:
		return errors.New("Stripe did not accept the merchandise refund")
	}
}
