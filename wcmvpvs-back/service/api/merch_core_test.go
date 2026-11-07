package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
)

type merchTestFixture struct {
	rt         *_router
	conn       *sql.DB
	org, other database.Organization
}

func newMerchTestFixture(t *testing.T) *merchTestFixture {
	t.Helper()
	conn, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "merch.db")+"?_busy_timeout=5000")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	db, err := database.New(conn)
	if err != nil {
		t.Fatal(err)
	}
	createOrg := func(name string) database.Organization {
		t.Helper()
		org, err := db.CreateOrganization(database.Organization{Name: name, Slug: strings.ToLower(name), Sport: "Pallavolo", IsActive: true})
		if err != nil {
			t.Fatal(err)
		}
		return org
	}
	org, other := createOrg("Volley"), createOrg("Rivale")
	rt := &_router{db: db, router: chi.NewRouter(), baseLogger: logrus.New(), adminSessions: map[string]adminSession{}, sessionTimeout: time.Hour}
	if err := rt.ensureMerchMigration(); err != nil {
		t.Fatal(err)
	}
	if err := rt.ensureMerchMigration(); err != nil {
		t.Fatal("second migration:", err)
	}
	registerMerchRoutes(rt)
	f := &merchTestFixture{rt: rt, conn: conn, org: org, other: other}
	s := defaultMerchSettings()
	s.Enabled = true
	s.OnlineEnabled = true
	s.PickupPoint = "Segreteria"
	s.PaymentInstructions = "Paga in sede"
	f.settings(t, org.ID, s)
	f.settings(t, other.ID, s)
	return f
}

func (f *merchTestFixture) settings(t *testing.T, org int, s merchSettings) {
	t.Helper()
	raw, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.conn.Exec("INSERT INTO merch_settings(organization_id,config) VALUES(?,?) ON CONFLICT(organization_id) DO UPDATE SET config=excluded.config", org, string(raw))
	if err != nil {
		t.Fatal(err)
	}
}

func (f *merchTestFixture) product(t *testing.T, stock int) merchProduct {
	t.Helper()
	p := merchProduct{Name: "Maglia", Category: "Abbigliamento", PriceCents: 3200, Stock: stock, MaxQuantity: 3, Published: true}
	if err := f.rt.saveMerchProduct(context.Background(), f.org.ID, f.org.Slug, "", &p); err != nil {
		t.Fatal(err)
	}
	return p
}

func (f *merchTestFixture) order(t *testing.T, p merchProduct, mode string) merchCheckoutInput {
	t.Helper()
	id, err := merchID()
	if err != nil {
		t.Fatal(err)
	}
	cents := p.PriceCents
	return merchCheckoutInput{Mode: mode, Fulfillment: "pickup", Items: []merchLine{{ProductID: p.ID, Quantity: 1}}, Name: "Mario Rossi", Email: "mario@example.com", RequestID: id, ExpectedTotalCents: &cents}
}

func merchRequest(t *testing.T, h http.Handler, method, path string, body interface{}, headers map[string]string, status int) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != nil {
		if value, ok := body.(io.Reader); ok {
			reader = value
		} else {
			raw, err := json.Marshal(body)
			if err != nil {
				t.Fatal(err)
			}
			reader = bytes.NewReader(raw)
		}
	}
	r := httptest.NewRequest(method, path, reader)
	for key, value := range headers {
		r.Header.Set(key, value)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != status {
		t.Fatalf("%s %s = %d (want %d) %s", method, path, w.Code, status, w.Body.String())
	}
	return w
}

func merchDecode(t *testing.T, w *httptest.ResponseRecorder, destination interface{}) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), destination); err != nil {
		t.Fatal(err, w.Body.String())
	}
}

func TestMerchAdminCatalogUploadAndTenantIsolation(t *testing.T) {
	f := newMerchTestFixture(t)
	f.rt.adminSessions["operator"] = adminSession{AdminID: 7, Username: "staff-volley", Role: "admin", OrganizationID: f.org.ID, ExpiresAt: time.Now().Add(time.Hour)}
	f.rt.adminSessions["master"] = adminSession{AdminID: 8, Username: "master", Role: "superadmin", ExpiresAt: time.Now().Add(time.Hour)}
	admin := "/v1/club-admin/" + f.org.Slug + "/merch"
	public := "/v1/clubs/" + f.org.Slug + "/merch"
	headers := map[string]string{"Authorization": "Bearer operator", "X-Organization-Slug": f.org.Slug}
	h := f.rt.router
	merchRequest(t, h, "PUT", admin+"/settings", map[string]bool{"visible": false}, map[string]string{"Authorization": "Bearer operator"}, 403)
	merchRequest(t, h, "GET", "/v1/club-admin/"+f.other.Slug+"/merch/products", nil, headers, 403)
	merchRequest(t, h, "GET", "/v1/club-admin/"+f.other.Slug+"/merch/products", nil, map[string]string{"Authorization": "Bearer master"}, 200)
	badBody := &bytes.Buffer{}
	bad := multipart.NewWriter(badBody)
	badPart, _ := bad.CreateFormFile("file", "payload.png")
	_, _ = badPart.Write([]byte("<svg><script/></svg>"))
	_ = bad.Close()
	headers["Content-Type"] = bad.FormDataContentType()
	merchRequest(t, h, "POST", admin+"/media", badBody, headers, 400)
	body := &bytes.Buffer{}
	form := multipart.NewWriter(body)
	part, err := form.CreateFormFile("file", "foto.png")
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(part, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	if err := form.Close(); err != nil {
		t.Fatal(err)
	}
	headers["Content-Type"] = form.FormDataContentType()
	var media merchMedia
	merchDecode(t, merchRequest(t, h, "POST", admin+"/media", body, headers, 201), &media)
	delete(headers, "Content-Type")
	mediaPath := strings.TrimPrefix(media.URL, "/api")
	merchRequest(t, h, "GET", mediaPath, nil, nil, 404)
	merchRequest(t, h, "GET", mediaPath, nil, headers, 200)
	merchRequest(t, h, "GET", "/v1/clubs/"+f.other.Slug+"/merch/media/"+media.ID, nil, nil, 404)
	var saved struct {
		Product merchProduct `json:"product"`
	}
	p := merchProduct{Name: "Felpa", PriceCents: 4000, MaxQuantity: 2, Media: []merchMedia{media}, Variants: []merchVariant{{Size: "M", Color: "Blu", Stock: 2}}}
	merchRequest(t, h, "POST", "/v1/club-admin/"+f.other.Slug+"/merch/products", p, map[string]string{"Authorization": "Bearer master", "X-Organization-Slug": f.other.Slug}, 400)
	merchDecode(t, merchRequest(t, h, "POST", admin+"/products", p, headers, 200), &saved)
	var catalog struct {
		Products []merchProduct `json:"products"`
	}
	merchDecode(t, merchRequest(t, h, "GET", public, nil, nil, 200), &catalog)
	if len(catalog.Products) != 0 {
		t.Fatal("draft leaked")
	}
	p = saved.Product
	p.Published = true
	merchDecode(t, merchRequest(t, h, "PUT", admin+"/products/"+p.ID, p, headers, 200), &saved)
	merchRequest(t, h, "PUT", admin+"/products/"+p.ID, p, headers, 409)
	merchDecode(t, merchRequest(t, h, "GET", public, nil, nil, 200), &catalog)
	if len(catalog.Products) != 1 || catalog.Products[0].Variants[0].Stock != 2 || catalog.Products[0].Media[0].ID != media.ID {
		t.Fatalf("catalog %+v", catalog.Products)
	}
	merchRequest(t, h, "GET", mediaPath, nil, nil, 200)
	merchRequest(t, h, "DELETE", admin+"/products/"+p.ID, nil, headers, 204)
	merchRequest(t, h, "GET", mediaPath, nil, nil, 404)
}

func TestMerchGuestQuoteOrderRecoveryOfflinePaymentAndAnalytics(t *testing.T) {
	f := newMerchTestFixture(t)
	p := f.product(t, 3)
	h := f.rt.router
	public := "/v1/clubs/" + f.org.Slug + "/merch"
	other := "/v1/clubs/" + f.other.Slug + "/merch"
	in := f.order(t, p, "offline")
	w := merchRequest(t, h, "POST", public+"/quote", in, nil, 200)
	var quote merchQuote
	merchDecode(t, w, &quote)
	if quote.TotalCents != 3200 || len(quote.Items) != 1 {
		t.Fatal(quote)
	}
	stale := int64(3000)
	in.ExpectedTotalCents = &stale
	merchRequest(t, h, "POST", public+"/orders", in, nil, 409)
	in.ExpectedTotalCents = &quote.TotalCents
	var created, repeated merchCheckoutResponse
	merchDecode(t, merchRequest(t, h, "POST", public+"/orders", in, nil, 201), &created)
	merchDecode(t, merchRequest(t, h, "POST", public+"/orders", in, nil, 201), &repeated)
	if created.Order.ID != repeated.Order.ID || created.AccessToken != repeated.AccessToken || created.Order.PaymentStatus != "UNPAID_OFFLINE" || created.CheckoutURL != "" || created.Order.PickupPoint != "Segreteria" {
		t.Fatalf("unexpected checkout %+v %+v", created, repeated)
	}
	orderPath := "/orders/" + created.Order.ID
	merchRequest(t, h, "GET", public+orderPath, nil, nil, 404)
	access := map[string]string{"X-Merch-Access": created.AccessToken}
	merchRequest(t, h, "GET", other+orderPath, nil, access, 404)
	merchRequest(t, h, "GET", public+orderPath, nil, access, 200)
	evt := merchAnalyticsInput{EventID: in.RequestID, Type: "merch_product_view", ProductID: p.ID}
	merchRequest(t, h, "POST", public+"/events", evt, nil, 204)
	merchRequest(t, h, "POST", public+"/events", evt, nil, 204)
	merchRequest(t, h, "POST", other+"/events", evt, nil, 404)
	evt.Type = "merch_payment_completed"
	merchRequest(t, h, "POST", public+"/events", evt, nil, 400)
	ctx := context.Background()
	before, err := loadMerchAnalytics(ctx, f.conn, f.org.ID, f.org.Slug)
	if err != nil || before.OfflineReservationValueCents != 3200 || before.OnlineRevenueCents != 0 || before.Events["merch_product_view"] != 1 || before.CheckoutStarted != 1 {
		t.Fatalf("analytics %+v %v", before, err)
	}
	order, err := f.rt.applyMerchAdminAction(ctx, f.org.ID, created.Order.ID, "operatore", merchAdminAction{Action: "record_payment", Confirmed: true, AmountCents: 3200, Method: "Contanti", Reference: "cassa"})
	if err != nil || order.PaymentStatus != "PAID_OFFLINE" || order.FulfillmentStatus != "CONFIRMED" || len(order.Payments) != 1 || order.Payments[0].Actor != "operatore" {
		t.Fatalf("order %+v %v", order, err)
	}
	for _, action := range []string{"prepare", "ready", "complete"} {
		order, err = f.rt.applyMerchAdminAction(ctx, f.org.ID, created.Order.ID, "operatore", merchAdminAction{Action: action, Confirmed: true})
		if err != nil {
			t.Fatal(action, err)
		}
	}
	if order.FulfillmentStatus != "COMPLETED" || order.PaymentStatus != "PAID_OFFLINE" {
		t.Fatal(order)
	}
	after, err := loadMerchAnalytics(ctx, f.conn, f.org.ID, f.org.Slug)
	if err != nil || after.OfflineReservationValueCents != 0 || after.OnlineRevenueCents != 0 {
		t.Fatalf("analytics %+v %v", after, err)
	}
}

func TestMerchConcurrentLastUnitExpiryAndLatePayment(t *testing.T) {
	f := newMerchTestFixture(t)
	ctx := context.Background()
	p := f.product(t, 1)
	first, second := f.order(t, p, "online"), f.order(t, p, "online")
	type attempt struct {
		order merchOrder
		err   error
	}
	out := make(chan attempt, 2)
	var wait sync.WaitGroup
	for _, input := range []merchCheckoutInput{first, second} {
		wait.Add(1)
		go func(in merchCheckoutInput) {
			defer wait.Done()
			order, err := f.rt.createMerchOrder(ctx, f.org.ID, "acct_test", in, database.FanProfile{}, "test")
			out <- attempt{order, err}
		}(input)
	}
	wait.Wait()
	close(out)
	var held merchOrder
	conflicts := 0
	for result := range out {
		var problem merchError
		switch {
		case result.err == nil:
			held = result.order
		case errors.As(result.err, &problem) && problem.status == 409:
			conflicts++
		default:
			t.Fatal(result.err)
		}
	}
	if conflicts != 1 || held.ID == "" {
		t.Fatalf("got %d conflicts, held %+v", conflicts, held)
	}
	if _, err := f.conn.Exec("UPDATE merch_orders SET expires_at=? WHERE id=?", time.Now().Add(-time.Minute).Unix(), held.ID); err != nil {
		t.Fatal(err)
	}
	if err := f.rt.reconcileMerchOrders(ctx); err != nil {
		t.Fatal(err)
	}
	next, err := f.rt.createMerchOrder(ctx, f.org.ID, "", f.order(t, p, "offline"), database.FanProfile{}, "test")
	if err != nil {
		t.Fatal("expired hold not released:", err)
	}
	event := merchStripeEvent{EventID: "evt_late", OrderID: held.ID, AccountID: "acct_test", SessionID: "cs_test_late", PaymentIntent: "pi_late", Status: "PAID", Currency: "EUR", TotalCents: held.TotalCents}
	for n := 0; n < 2; n++ {
		if err := f.rt.applyMerchStripeEvent(ctx, event); err != nil {
			t.Fatal(err)
		}
	}
	paid, err := loadMerchOrder(ctx, f.conn, f.org.ID, held.ID)
	if err != nil || paid.PaymentStatus != "PAID" || paid.FulfillmentStatus != "REVIEW_REQUIRED" || paid.StockStatus != "RELEASED" {
		t.Fatalf("paid %+v %v", paid, err)
	}
	reservation, err := loadMerchOrder(ctx, f.conn, f.org.ID, next.ID)
	if err != nil || reservation.StockStatus != "HELD" {
		t.Fatalf("reservation %+v %v", reservation, err)
	}
	a, err := loadMerchAnalytics(ctx, f.conn, f.org.ID, f.org.Slug)
	if err != nil || a.OnlineRevenueCents != held.TotalCents || a.PaymentCompleted != 1 {
		t.Fatalf("analytics %+v %v", a, err)
	}
}

func TestMerchExistingProfileOwnOrdersAndPrefilledPhone(t *testing.T) {
	f := newMerchTestFixture(t)
	phone := "+393331234567"
	fan, err := f.rt.db.CreateFanWithPhoneE164(f.org.ID, phone)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.rt.db.MarkFanPhoneVerified(f.org.ID, phone, time.Now()); err != nil {
		t.Fatal(err)
	}
	// Complete an existing fan registration with its normal persisted state.
	if _, err := f.conn.Exec("UPDATE fan_profiles SET nickname='Fan',accepted_terms=1,registration_pending=0 WHERE id=?", fan.ID); err != nil {
		t.Fatal(err)
	}
	token := "existing-fan-session"
	if err := f.rt.db.UpsertFanSession(token, fan.ID, "device"); err != nil {
		t.Fatal(err)
	}
	p := f.product(t, 2)
	in := f.order(t, p, "offline")
	in.Email = ""
	h := f.rt.router
	public := "/v1/clubs/" + f.org.Slug + "/merch"
	headers := map[string]string{"X-Fan-Session": token}
	var created merchCheckoutResponse
	merchDecode(t, merchRequest(t, h, "POST", public+"/orders", in, headers, 201), &created)
	if created.Order.Phone != phone || created.AccessToken != "" {
		t.Fatalf("checkout %+v", created)
	}
	merchRequest(t, h, "GET", public+"/orders/"+created.Order.ID, nil, headers, 200)
	merchRequest(t, h, "GET", public+"/orders/"+created.Order.ID, nil, nil, 404)
	var list struct {
		Orders []merchOrder `json:"orders"`
	}
	merchDecode(t, merchRequest(t, h, "GET", public+"/orders", nil, headers, 200), &list)
	if len(list.Orders) != 1 || list.Orders[0].ID != created.Order.ID {
		t.Fatal(list)
	}
	merchRequest(t, h, "GET", "/v1/clubs/"+f.other.Slug+"/merch/orders", nil, headers, 403)
}
