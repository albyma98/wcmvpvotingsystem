package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
)

type merchPublicHandler func(http.ResponseWriter, *http.Request, database.Organization)

func (rt *_router) wrapMerchPublic(fn merchPublicHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		org, err := rt.db.GetOrganizationBySlug(chi.URLParam(r, "slug"))
		if err != nil {
			rt.writeMerchError(w, err)
			return
		}
		if !org.IsActive {
			rt.writeMerchError(w, sql.ErrNoRows)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		fn(w, r, org)
	}
}

func (rt *_router) wrapMerchAdmin(fn httpRouterHandler) http.HandlerFunc {
	wrapped := rt.wrapAdmin(func(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
		if !strings.EqualFold(ctx.OrganizationSlug, chi.URLParam(r, "slug")) ||
			(!strings.EqualFold(ctx.AdminRole, "admin") && !strings.EqualFold(ctx.AdminRole, "superadmin")) {
			rt.writeMerchError(w, merchError{http.StatusForbidden, "Società non autorizzata"})
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		fn(w, r, ctx)
	})
	return func(w http.ResponseWriter, r *http.Request) {
		// The tenant header doubles as a non-simple browser request header for
		// cookie-authenticated mutations, including multipart uploads.
		if r.Method != http.MethodGet && (!strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Organization-Slug")), chi.URLParam(r, "slug")) || r.Header.Get("Sec-Fetch-Site") == "cross-site") {
			rt.writeMerchError(w, merchError{http.StatusForbidden, "Richiesta non autorizzata"})
			return
		}
		if r.Header.Get("X-Organization-Slug") == "" {
			r = r.Clone(r.Context())
			r.Header.Set("X-Organization-Slug", chi.URLParam(r, "slug"))
		}
		wrapped(w, r)
	}
}

func registerMerchRoutes(rt *_router) {
	const public = "/v1/clubs/{slug}/merch"
	const admin = "/v1/club-admin/{slug}/merch"
	rt.router.Get(public, rt.wrapMerchPublic(rt.getMerchCatalog))
	rt.router.Post(public+"/quote", rt.wrapMerchPublic(rt.postMerchQuote))
	rt.router.Post(public+"/orders", rt.wrapMerchPublic(rt.postMerchOrder))
	rt.router.Get(public+"/orders", rt.wrapMerchPublic(rt.getMerchOrders))
	rt.router.Get(public+"/orders/{id}", rt.wrapMerchPublic(rt.getMerchOrder))
	rt.router.Get(public+"/media/{id}", rt.wrapMerchPublic(rt.getMerchMedia))
	rt.router.Post(public+"/events", rt.wrapMerchPublic(rt.postMerchAnalytics))
	registerMerchRecoveryRoutes(rt, public)
	rt.router.Get(admin+"/settings", rt.wrapMerchAdmin(rt.getAdminMerchSettings))
	rt.router.Put(admin+"/settings", rt.wrapMerchAdmin(rt.putAdminMerchSettings))
	rt.router.Get(admin+"/products", rt.wrapMerchAdmin(rt.getAdminMerchProducts))
	rt.router.Post(admin+"/products", rt.wrapMerchAdmin(rt.saveAdminMerchProduct))
	rt.router.Put(admin+"/products/{id}", rt.wrapMerchAdmin(rt.saveAdminMerchProduct))
	rt.router.Delete(admin+"/products/{id}", rt.wrapMerchAdmin(rt.deleteAdminMerchProduct))
	rt.router.Post(admin+"/media", rt.wrapMerchAdmin(rt.postMerchMedia))
	rt.router.Get(admin+"/orders", rt.wrapMerchAdmin(rt.getAdminMerchOrders))
	rt.router.Get(admin+"/orders/{id}", rt.wrapMerchAdmin(rt.getAdminMerchOrder))
	rt.router.Post(admin+"/orders/{id}/actions", rt.wrapMerchAdmin(rt.postAdminMerchOrderAction))
	rt.router.Get(admin+"/analytics", rt.wrapMerchAdmin(rt.getAdminMerchAnalytics))
	registerMerchStripeRoutes(rt)
}

func readMerchJSON(w http.ResponseWriter, r *http.Request, value interface{}) error {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 128<<10))
	if err := d.Decode(value); err != nil {
		return merchInvalid("Richiesta non valida")
	}
	var trailing json.RawMessage
	if err := d.Decode(&trailing); err != io.EOF {
		return merchInvalid("Richiesta non valida")
	}
	return nil
}

func (rt *_router) getMerchCatalog(w http.ResponseWriter, r *http.Request, org database.Organization) {
	settings, err := loadMerchSettings(r.Context(), rt.db.SQLConn(), org.ID)
	if err != nil {
		rt.writeMerchError(w, err)
		return
	}
	settings.SMSNotifications = false
	if r.URL.Query().Get("summary") == "1" {
		_ = writeJSON(w, http.StatusOK, map[string]interface{}{"settings": map[string]bool{"enabled": settings.Enabled, "visible": settings.Visible}})
		return
	}
	products := []merchProduct{}
	if settings.Enabled {
		if settings.OnlineEnabled {
			_, status := rt.merchStripeStatus(r.Context(), org.Slug)
			settings.OnlineEnabled = status == "READY"
			if !settings.OnlineEnabled && settings.OfflineEnabled {
				settings.DefaultMode = "offline"
			}
		}
		products, err = queryMerchProducts(r.Context(), rt.db.SQLConn(), org.ID, org.Slug, settings, true)
		if err != nil {
			rt.writeMerchError(w, err)
			return
		}
	}
	_ = writeJSON(w, http.StatusOK, map[string]interface{}{"settings": settings, "products": products})
}

func (rt *_router) postMerchQuote(w http.ResponseWriter, r *http.Request, org database.Organization) {
	var in merchCheckoutInput
	if err := readMerchJSON(w, r, &in); err != nil {
		rt.writeMerchError(w, err)
		return
	}
	conn := rt.db.SQLConn()
	settings, err := loadMerchSettings(r.Context(), conn, org.ID)
	if err != nil {
		rt.writeMerchError(w, err)
		return
	}
	if in.Mode == "online" && settings.OnlineEnabled {
		_, status := rt.merchStripeStatus(r.Context(), org.Slug)
		settings.OnlineEnabled = status == "READY"
	}
	// Use a single read snapshot for all product and inventory queries.
	tx, err := conn.BeginTx(r.Context(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		rt.writeMerchError(w, err)
		return
	}
	defer tx.Rollback()
	quote, err := quoteMerchOrder(r.Context(), tx, org.ID, settings, in)
	if err != nil {
		rt.writeMerchError(w, err)
		return
	}
	_ = writeJSON(w, http.StatusOK, quote)
}

type merchCheckoutResponse struct {
	Order       merchOrder `json:"order"`
	CheckoutURL string     `json:"checkout_url,omitempty"`
	AccessToken string     `json:"access_token,omitempty"`
}

func (rt *_router) resumeMerchCheckout(order merchOrder, slug string) string {
	if order.Mode == "online" && order.PaymentStatus == "PENDING_ONLINE" &&
		!order.StripeAwaiting && order.ExpiresAt > time.Now().Unix() &&
		order.StripeAccount != "" && order.StripeAccount == rt.merchStripeAccount(slug) &&
		rt.merchStripeConfigured() {
		return order.StripeURL
	}
	return ""
}

func (rt *_router) postMerchOrder(w http.ResponseWriter, r *http.Request, org database.Organization) {
	var in merchCheckoutInput
	if err := readMerchJSON(w, r, &in); err != nil {
		rt.writeMerchError(w, err)
		return
	}
	if !merchRequestID.MatchString(in.RequestID) || in.ExpectedTotalCents == nil {
		rt.writeMerchError(w, merchInvalid("Riepilogo o identificativo richiesta mancante"))
		return
	}
	fan, err := rt.merchFan(r, org.ID)
	if err != nil {
		rt.writeMerchError(w, err)
		return
	}
	account := ""
	if in.Mode == "online" {
		id, status := rt.merchStripeStatus(r.Context(), org.Slug)
		if status == "READY" {
			account = id
		}
	}
	order, err := rt.createMerchOrder(r.Context(), org.ID, account, in, fan, rt.getClientIP(r))
	if err != nil {
		rt.writeMerchError(w, err)
		return
	}
	order.SMSRecoveryAvailable = rt.merchRecoveryAvailable(r.Context(), order)
	resp := merchCheckoutResponse{Order: order}
	if fan.ID == 0 {
		resp.AccessToken = merchGuestToken(order.ID, order.RequestID)
	}
	if order.Mode == "online" && order.PaymentStatus == "PENDING_ONLINE" && !order.StripeAwaiting && order.ExpiresAt > time.Now().Unix() {
		if order.StripeURL != "" && rt.resumeMerchCheckout(order, org.Slug) != "" {
			resp.CheckoutURL = order.StripeURL
		} else if order.StripeSession == "" && order.StripeAccount == account && order.ExpiresAt-time.Now().Unix() >= 30*60+5 {
			sessionID, url, err := rt.createMerchStripeCheckout(r.Context(), org.Slug, order, resp.AccessToken)
			if err != nil {
				rt.baseLogger.WithError(err).WithField("order_id", order.ID).Warn("Stripe merchandise checkout creation failed")
				// Return the capability with a recoverable error. Retrying with
				// the same request ID uses the same order and Stripe key.
				_ = writeJSON(w, http.StatusBadGateway, map[string]interface{}{"order": order, "access_token": resp.AccessToken, "message": "Checkout non disponibile. Riprova per riprendere lo stesso ordine."})
				return
			}
			if err := rt.attachMerchStripeSession(r.Context(), order.ID, order.StripeAccount, sessionID, url); err != nil {
				rt.writeMerchError(w, err)
				return
			}
			resp.Order, err = loadMerchOrder(r.Context(), rt.db.SQLConn(), org.ID, order.ID)
			if err != nil {
				rt.writeMerchError(w, err)
				return
			}
			if resp.Order.PaymentStatus == "PENDING_ONLINE" && !resp.Order.StripeAwaiting && resp.Order.ExpiresAt > time.Now().Unix() {
				resp.CheckoutURL = url
			}
		}
	}
	if resp.Order.Mode == "online" && resp.Order.PaymentStatus == "PENDING_ONLINE" && !resp.Order.StripeAwaiting && resp.Order.ExpiresAt > time.Now().Unix() && resp.CheckoutURL == "" {
		_ = writeJSON(w, http.StatusBadGateway, map[string]interface{}{"order": resp.Order, "access_token": resp.AccessToken, "message": "Checkout non disponibile. Riprova per riprendere lo stesso ordine."})
		return
	}
	resp.Order.CheckoutAvailable = resp.CheckoutURL != ""
	_ = writeJSON(w, http.StatusCreated, resp)
}

func (rt *_router) getMerchOrders(w http.ResponseWriter, r *http.Request, org database.Organization) {
	fan, err := rt.merchFan(r, org.ID)
	if err != nil {
		rt.writeMerchError(w, err)
		return
	}
	offset, err := merchPageOffset(r)
	if err != nil {
		rt.writeMerchError(w, err)
		return
	}
	items := []merchOrder{}
	if fan.ID > 0 {
		items, err = queryMerchOrders(r.Context(), rt.db.SQLConn(), org.ID, fan.ID, "", "", offset)
		if err != nil {
			rt.writeMerchError(w, err)
			return
		}
	}
	for i := range items {
		items[i].CheckoutAvailable = rt.resumeMerchCheckout(items[i], org.Slug) != ""
	}
	_ = writeJSON(w, http.StatusOK, merchOrderPage(items))
}

func (rt *_router) getMerchOrder(w http.ResponseWriter, r *http.Request, org database.Organization) {
	fan, err := rt.merchFan(r, org.ID)
	if err != nil {
		rt.writeMerchError(w, err)
		return
	}
	order, err := loadMerchOrder(r.Context(), rt.db.SQLConn(), org.ID, chi.URLParam(r, "id"))
	if err != nil {
		rt.writeMerchError(w, err)
		return
	}
	if (fan.ID == 0 || fan.ID != order.FanID) && !order.acceptsGuestToken(r.Header.Get("X-Merch-Access")) {
		rt.writeMerchError(w, sql.ErrNoRows)
		return
	}
	order.SMSRecoveryAvailable = rt.merchRecoveryAvailable(r.Context(), order)
	checkoutURL := rt.resumeMerchCheckout(order, org.Slug)
	order.CheckoutAvailable = checkoutURL != ""
	_ = writeJSON(w, http.StatusOK, merchCheckoutResponse{Order: order, CheckoutURL: checkoutURL})
}

func (rt *_router) getAdminMerchSettings(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	settings, err := loadMerchSettings(r.Context(), rt.db.SQLConn(), ctx.OrganizationID)
	if err != nil {
		rt.writeMerchError(w, err)
		return
	}
	settings.StripeAccountID, settings.StripeStatus = rt.merchStripeStatus(r.Context(), ctx.OrganizationSlug)
	_ = writeJSON(w, http.StatusOK, map[string]interface{}{"settings": settings})
}

func (rt *_router) putAdminMerchSettings(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	settings, err := loadMerchSettings(r.Context(), rt.db.SQLConn(), ctx.OrganizationID)
	if err != nil {
		rt.writeMerchError(w, err)
		return
	}
	if err := readMerchJSON(w, r, &settings); err != nil {
		rt.writeMerchError(w, err)
		return
	}
	if err := validateMerchSettings(&settings); err != nil {
		rt.writeMerchError(w, err)
		return
	}
	raw, err := json.Marshal(settings)
	if err == nil {
		err = rt.merchWrite(r.Context(), func(tx *sql.Tx) error {
			_, err := tx.ExecContext(r.Context(), `INSERT INTO merch_settings (organization_id,config) VALUES (?,?)
				ON CONFLICT(organization_id) DO UPDATE SET config=excluded.config`, ctx.OrganizationID, string(raw))
			return err
		})
	}
	if err != nil {
		rt.writeMerchError(w, err)
		return
	}
	settings.StripeAccountID, settings.StripeStatus = rt.merchStripeStatus(r.Context(), ctx.OrganizationSlug)
	_ = writeJSON(w, http.StatusOK, map[string]interface{}{"settings": settings})
}

func (rt *_router) getAdminMerchProducts(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	conn := rt.db.SQLConn()
	settings, err := loadMerchSettings(r.Context(), conn, ctx.OrganizationID)
	if err != nil {
		rt.writeMerchError(w, err)
		return
	}
	products, err := queryMerchProducts(r.Context(), conn, ctx.OrganizationID, ctx.OrganizationSlug, settings, false)
	if err != nil {
		rt.writeMerchError(w, err)
		return
	}
	_ = writeJSON(w, http.StatusOK, map[string]interface{}{"products": products})
}

func (rt *_router) saveAdminMerchProduct(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	p := merchProduct{MaxQuantity: 5, SaleMode: "inherit"}
	if err := readMerchJSON(w, r, &p); err != nil {
		rt.writeMerchError(w, err)
		return
	}
	if err := rt.saveMerchProduct(r.Context(), ctx.OrganizationID, ctx.OrganizationSlug, chi.URLParam(r, "id"), &p); err != nil {
		rt.writeMerchError(w, err)
		return
	}
	_ = writeJSON(w, http.StatusOK, map[string]interface{}{"product": p})
}

func (rt *_router) deleteAdminMerchProduct(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if err := rt.archiveMerchProduct(r.Context(), ctx.OrganizationID, chi.URLParam(r, "id")); err != nil {
		rt.writeMerchError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func queryMerchOrders(ctx context.Context, q merchQuerier, org, fan int, filter, search string, offset int) ([]merchOrder, error) {
	args := []interface{}{org}
	query := "SELECT " + merchOrderColumns + " FROM merch_orders WHERE organization_id=?"
	if fan != 0 {
		query += " AND fan_id=?"
		args = append(args, fan)
	}
	if filter != "" {
		switch filter {
		case "TO_PREPARE":
			query += " AND fulfillment_status IN ('CONFIRMED','PREPARING')"
		case "PAID", "PAID_OFFLINE", "PENDING_ONLINE", "FAILED", "REFUNDED":
			query += " AND payment_status=?"
		case "UNPAID_OFFLINE":
			query += " AND payment_status=? AND fulfillment_status NOT IN ('CANCELLED','EXPIRED')"
		case "PENDING", "CONFIRMED", "PREPARING", "READY_FOR_PICKUP", "COMPLETED", "CANCELLED", "EXPIRED", "REVIEW_REQUIRED":
			query += " AND fulfillment_status=?"
		default:
			return nil, merchInvalid("Filtro non valido")
		}
		if filter != "TO_PREPARE" {
			args = append(args, filter)
		}
	}
	if search != "" {
		query += " AND (instr(lower(code),lower(?))>0 OR instr(lower(name),lower(?))>0 OR instr(lower(email),lower(?))>0 OR instr(phone,?)>0)"
		args = append(args, search, search, search, search)
	}
	args = append(args, offset)
	rows, err := q.QueryContext(ctx, query+" ORDER BY created_at DESC,id DESC LIMIT 101 OFFSET ?", args...)
	if err != nil {
		return nil, err
	}
	orders := []merchOrder{}
	for rows.Next() {
		order, err := scanMerchOrder(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		orders = append(orders, order)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for i := range orders {
		if err := loadMerchOrderItems(ctx, q, &orders[i]); err != nil {
			return nil, err
		}
	}
	return orders, nil
}

func merchPageOffset(r *http.Request) (int, error) {
	value := r.URL.Query().Get("offset")
	if value == "" {
		return 0, nil
	}
	offset, err := strconv.Atoi(value)
	if err != nil || offset < 0 || offset > 1000000 {
		return 0, merchInvalid("Paginazione non valida")
	}
	return offset, nil
}

func merchOrderPage(items []merchOrder) map[string]interface{} {
	hasMore := len(items) > 100
	if hasMore {
		items = items[:100]
	}
	return map[string]interface{}{"orders": items, "has_more": hasMore}
}

func (rt *_router) getAdminMerchOrders(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	offset, err := merchPageOffset(r)
	if err != nil {
		rt.writeMerchError(w, err)
		return
	}
	search := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(search) > 160 {
		rt.writeMerchError(w, merchInvalid("Ricerca troppo lunga"))
		return
	}
	items, err := queryMerchOrders(r.Context(), rt.db.SQLConn(), ctx.OrganizationID, 0, r.URL.Query().Get("status"), search, offset)
	if err != nil {
		rt.writeMerchError(w, err)
		return
	}
	_ = writeJSON(w, http.StatusOK, merchOrderPage(items))
}

func (rt *_router) getAdminMerchOrder(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	conn := rt.db.SQLConn()
	order, err := loadMerchOrder(r.Context(), conn, ctx.OrganizationID, chi.URLParam(r, "id"))
	if err == nil {
		err = loadMerchOrderAudit(r.Context(), conn, &order)
	}
	if err != nil {
		rt.writeMerchError(w, err)
		return
	}
	_ = writeJSON(w, http.StatusOK, map[string]interface{}{"order": order})
}

func (rt *_router) postAdminMerchOrderAction(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var a merchAdminAction
	if err := readMerchJSON(w, r, &a); err != nil {
		rt.writeMerchError(w, err)
		return
	}
	order, err := rt.applyMerchAdminAction(r.Context(), ctx.OrganizationID, chi.URLParam(r, "id"), fmt.Sprintf("%s (#%d)", ctx.AdminUsername, ctx.AdminID), a)
	if err != nil {
		rt.writeMerchError(w, err)
		return
	}
	_ = writeJSON(w, http.StatusOK, map[string]interface{}{"order": order})
}

type merchAnalyticsInput struct {
	EventID   string `json:"event_id"`
	Type      string `json:"type"`
	ProductID string `json:"product_id"`
	VariantID string `json:"variant_id"`
}

func (rt *_router) postMerchAnalytics(w http.ResponseWriter, r *http.Request, org database.Organization) {
	var in merchAnalyticsInput
	if err := readMerchJSON(w, r, &in); err != nil {
		rt.writeMerchError(w, err)
		return
	}
	if !merchRequestID.MatchString(in.EventID) {
		rt.writeMerchError(w, merchInvalid("Identificativo evento non valido"))
		return
	}
	switch in.Type {
	case "merch_impression", "merch_product_view", "merch_variant_selected", "merch_outbound_click", "merch_avatar_try_on":
	default:
		rt.writeMerchError(w, merchInvalid("Evento non valido"))
		return
	}
	if (in.ProductID == "" && in.Type != "merch_impression" && in.Type != "merch_outbound_click") || (in.VariantID == "" && in.Type == "merch_variant_selected") {
		rt.writeMerchError(w, merchInvalid("Prodotto o variante mancanti"))
		return
	}
	ctx := r.Context()
	var rejection error
	err := rt.merchWrite(ctx, func(tx *sql.Tx) error {
		settings, err := loadMerchSettings(ctx, tx, org.ID)
		if err != nil {
			return err
		}
		if !settings.Enabled {
			return sql.ErrNoRows
		}
		ok, err := merchRate(ctx, tx, org.ID, "analytics", rt.getClientIP(r), time.Minute, 600)
		if err != nil {
			return err
		}
		if !ok {
			rejection = merchError{http.StatusTooManyRequests, "Troppe richieste"}
			return nil
		}
		if in.ProductID != "" {
			var existing string
			err := tx.QueryRowContext(ctx, "SELECT id FROM merch_products WHERE organization_id=? AND id=? AND published=1 AND archived=0", org.ID, in.ProductID).Scan(&existing)
			if errors.Is(err, sql.ErrNoRows) {
				rejection = err
				return nil
			}
			if err != nil {
				return err
			}
		}
		if in.VariantID != "" {
			var existing string
			err := tx.QueryRowContext(ctx, "SELECT id FROM merch_product_variants WHERE organization_id=? AND product_id=? AND id=? AND active=1", org.ID, in.ProductID, in.VariantID).Scan(&existing)
			if errors.Is(err, sql.ErrNoRows) {
				rejection = err
				return nil
			}
			if err != nil {
				return err
			}
		}
		return addMerchAnalytics(ctx, tx, org.ID, in.EventID, in.Type, in.ProductID, in.VariantID)
	})
	if err == nil {
		err = rejection
	}
	if err != nil {
		rt.writeMerchError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type merchAnalytics struct {
	PublishedProducts            int                 `json:"published_products"`
	OrdersReceived               int                 `json:"orders_received"`
	PendingReservations          int                 `json:"pending_reservations"`
	OnlineRevenueCents           int64               `json:"online_revenue_cents"`
	UnpaidReservations           int                 `json:"unpaid_reservations"`
	OfflineReservationValueCents int64               `json:"offline_reservation_value_cents"`
	PaymentCompleted             int                 `json:"payment_completed"`
	ConfirmedReservations        int                 `json:"confirmed_reservations"`
	CheckoutStarted              int                 `json:"checkout_started"`
	CheckoutAbandoned            int                 `json:"checkout_abandoned"`
	ConversionRate               float64             `json:"conversion_rate"`
	Events                       map[string]int      `json:"events"`
	TopProducts                  []merchProductViews `json:"top_products"`
	LowStock                     []merchProductStock `json:"low_stock"`
}

type merchProductViews struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Views int    `json:"views"`
}

type merchProductStock struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Stock int    `json:"stock"`
}

func (rt *_router) getAdminMerchAnalytics(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	data, err := loadMerchAnalytics(r.Context(), rt.db.SQLConn(), ctx.OrganizationID, ctx.OrganizationSlug)
	if err != nil {
		rt.writeMerchError(w, err)
		return
	}
	_ = writeJSON(w, http.StatusOK, data)
}

func loadMerchAnalytics(ctx context.Context, q merchQuerier, org int, slug string) (merchAnalytics, error) {
	a := merchAnalytics{Events: map[string]int{}, TopProducts: []merchProductViews{}, LowStock: []merchProductStock{}}
	err := q.QueryRowContext(ctx, `SELECT
		COALESCE(SUM(mode='offline' OR payment_status IN ('PAID','REFUNDED')),0),
		COALESCE(SUM(mode='offline' AND payment_status='UNPAID_OFFLINE' AND fulfillment_status='PENDING' AND expires_at>?),0),
		COALESCE(SUM(mode='offline' AND payment_status='UNPAID_OFFLINE' AND fulfillment_status NOT IN ('CANCELLED','EXPIRED') AND expires_at>?),0),
		COALESCE(SUM(CASE WHEN mode='offline' AND payment_status='UNPAID_OFFLINE' AND fulfillment_status NOT IN ('CANCELLED','EXPIRED') AND expires_at>? THEN total_cents ELSE 0 END),0),
		COALESCE(SUM(mode='offline' AND fulfillment_status IN ('CONFIRMED','PREPARING','READY_FOR_PICKUP','COMPLETED')),0),
		COALESCE(SUM(mode='online' AND payment_status IN ('FAILED','EXPIRED')),0)
		FROM merch_orders WHERE organization_id=?`, time.Now().Unix(), time.Now().Unix(), time.Now().Unix(), org).Scan(&a.OrdersReceived, &a.PendingReservations, &a.UnpaidReservations, &a.OfflineReservationValueCents, &a.ConfirmedReservations, &a.CheckoutAbandoned)
	if err != nil {
		return a, err
	}
	err = q.QueryRowContext(ctx, `SELECT COALESCE(SUM(CASE WHEN status='PAID' THEN amount_cents WHEN status='REFUNDED' THEN -amount_cents ELSE 0 END),0)
		FROM merch_payment_records WHERE organization_id=? AND source='stripe'`, org).Scan(&a.OnlineRevenueCents)
	if err != nil {
		return a, err
	}
	rows, err := q.QueryContext(ctx, "SELECT type,COUNT(*) FROM merch_analytics_events WHERE organization_id=? GROUP BY type", org)
	if err != nil {
		return a, err
	}
	for rows.Next() {
		var key string
		var count int
		if err := rows.Scan(&key, &count); err != nil {
			rows.Close()
			return a, err
		}
		a.Events[key] = count
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return a, err
	}
	a.CheckoutStarted = a.Events["merch_checkout_started"]
	a.PaymentCompleted = a.Events["merch_payment_completed"]
	if a.CheckoutStarted != 0 {
		a.ConversionRate = float64(a.PaymentCompleted+a.Events["merch_reservation_created"]) / float64(a.CheckoutStarted) * 100
	}
	rows, err = q.QueryContext(ctx, `SELECT p.id,p.name,COUNT(*) AS views FROM merch_analytics_events e
		JOIN merch_products p ON p.organization_id=e.organization_id AND p.id=e.product_id
		WHERE e.organization_id=? AND e.type='merch_product_view' GROUP BY p.id,p.name ORDER BY views DESC,p.name LIMIT 10`, org)
	if err != nil {
		return a, err
	}
	for rows.Next() {
		var p merchProductViews
		if err := rows.Scan(&p.ID, &p.Name, &p.Views); err != nil {
			rows.Close()
			return a, err
		}
		a.TopProducts = append(a.TopProducts, p)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return a, err
	}
	products, err := queryMerchProducts(ctx, q, org, slug, merchSettings{}, false)
	if err != nil {
		return a, err
	}
	for _, p := range products {
		if !p.Published {
			continue
		}
		stock, tracked := p.Available, !p.OnDemand
		if len(p.Variants) > 0 {
			stock, tracked = 0, false
			for _, v := range p.Variants {
				if v.OnDemand {
					continue
				}
				tracked = true
				stock += v.Available
			}
		}
		if tracked && stock <= 5 {
			a.LowStock = append(a.LowStock, merchProductStock{p.ID, p.Name, stock})
		}
	}
	return a, nil
}
