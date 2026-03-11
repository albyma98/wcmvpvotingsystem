package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
)

type barProduct struct {
	ID               string        `json:"id"`
	Name             string        `json:"name"`
	PriceCents       int64         `json:"price_cents"`
	ImageEmoji       string        `json:"image_emoji"`
	ImageURL         string        `json:"image_url,omitempty"`
	Description      string        `json:"description"`
	Category         string        `json:"category,omitempty"`
	CategoryImageURL string        `json:"category_image_url,omitempty"`
	IsMenu           bool          `json:"is_menu,omitempty"`
	Items            []interface{} `json:"items,omitempty"`
}

type barCheckoutItemPayload struct {
	ProductID string `json:"product_id"`
	Quantity  int64  `json:"quantity"`
}

type barCheckoutRequestPayload struct {
	Items  []barCheckoutItemPayload `json:"items"`
	Sector string                   `json:"sector"`
	Row    string                   `json:"row"`
	Seat   string                   `json:"seat"`
	Notes  string                   `json:"notes"`
}

type barCheckoutResponse struct {
	CheckoutURL string `json:"checkout_url"`
	SessionID   string `json:"session_id"`
	OrderID     int    `json:"order_id"`
}

type barConfirmCheckoutRequest struct {
	SessionID string `json:"session_id"`
}

type barConfirmCheckoutResponse struct {
	Confirmed bool `json:"confirmed"`
}

type stripeCheckoutSessionResponse struct {
	ID            string `json:"id"`
	URL           string `json:"url"`
	PaymentStatus string `json:"payment_status"`
}

func (rt *_router) listBarProducts(w http.ResponseWriter, _ *http.Request, _ reqcontext.RequestContext) {
	products, err := rt.db.ListShopProducts()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	menus, err := rt.db.ListBarMenus(false)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	result := make([]barProduct, 0, len(products)+len(menus))
	for _, p := range products {
		result = append(result, barProduct{ID: "product:" + strconv.Itoa(p.ID), Name: p.Name, PriceCents: int64(p.PriceCents), Description: p.Description, ImageURL: p.ImageURL, Category: p.Category, CategoryImageURL: p.CategoryImageURL})
	}
	for _, m := range menus {
		result = append(result, barProduct{ID: "menu:" + strconv.Itoa(m.ID), Name: m.Name, PriceCents: int64(m.PriceCents), Description: m.Description, IsMenu: true})
	}
	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (rt *_router) createBarCheckoutSession(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if strings.TrimSpace(rt.stripeSecretKey) == "" {
		_ = writeJSONMessage(w, http.StatusFailedDependency, "configurazione Stripe mancante")
		return
	}

	var payload barCheckoutRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "payload non valido")
		return
	}

	payload.Sector = strings.TrimSpace(payload.Sector)
	payload.Row = strings.TrimSpace(payload.Row)
	payload.Seat = strings.TrimSpace(payload.Seat)
	payload.Notes = strings.TrimSpace(payload.Notes)

	if payload.Sector == "" || payload.Row == "" || payload.Seat == "" {
		_ = writeJSONMessage(w, http.StatusBadRequest, "inserisci settore, fila e posto")
		return
	}
	if len(payload.Items) == 0 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "aggiungi almeno un prodotto")
		return
	}

	qtyMap := map[string]int64{}
	for _, item := range payload.Items {
		id := strings.TrimSpace(item.ProductID)
		if id == "" || item.Quantity <= 0 {
			_ = writeJSONMessage(w, http.StatusBadRequest, "elementi ordine non validi")
			return
		}
		qtyMap[id] += item.Quantity
	}

	form := url.Values{}
	form.Set("mode", "payment")
	base := resolveCheckoutRedirectBase(rt.stripeSuccessURL, r)
	form.Set("success_url", base+"?barOrderSuccess=1&session_id={CHECKOUT_SESSION_ID}")
	form.Set("cancel_url", base+"?barOrderCancelled=1")
	form.Set("payment_method_types[0]", "card")

	totalCents := int64(0)
	productsSnapshot := make([]map[string]interface{}, 0, len(qtyMap))
	quantitiesSnapshot := make([]map[string]interface{}, 0, len(qtyMap))
	lineIndex := 0
	for id, qty := range qtyMap {
		if qty <= 0 {
			continue
		}
		var name string
		var priceCents int64
		if strings.HasPrefix(id, "product:") {
			pid, err := strconv.Atoi(strings.TrimPrefix(id, "product:"))
			if err != nil || pid <= 0 {
				_ = writeJSONMessage(w, http.StatusBadRequest, "elementi ordine non validi")
				return
			}
			product, err := rt.db.GetShopProduct(pid)
			if err != nil {
				_ = writeJSONMessage(w, http.StatusBadRequest, "uno dei prodotti selezionati non è disponibile")
				return
			}
			name = product.Name
			priceCents = int64(product.PriceCents)
		} else if strings.HasPrefix(id, "menu:") {
			mid, err := strconv.Atoi(strings.TrimPrefix(id, "menu:"))
			if err != nil || mid <= 0 {
				_ = writeJSONMessage(w, http.StatusBadRequest, "elementi ordine non validi")
				return
			}
			menus, err := rt.db.ListBarMenus(false)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			found := false
			for _, m := range menus {
				if m.ID == mid {
					name = m.Name
					priceCents = int64(m.PriceCents)
					found = true
					break
				}
			}
			if !found {
				_ = writeJSONMessage(w, http.StatusBadRequest, "uno dei menu selezionati non è disponibile")
				return
			}
		} else {
			_ = writeJSONMessage(w, http.StatusBadRequest, "elementi ordine non validi")
			return
		}
		prefix := "line_items[" + strconv.Itoa(lineIndex) + "]"
		form.Set(prefix+"[quantity]", strconv.FormatInt(qty, 10))
		form.Set(prefix+"[price_data][currency]", "eur")
		form.Set(prefix+"[price_data][unit_amount]", strconv.FormatInt(priceCents, 10))
		form.Set(prefix+"[price_data][product_data][name]", name)
		totalCents += priceCents * qty
		productsSnapshot = append(productsSnapshot, map[string]interface{}{"id": id, "name": name, "price_cents": priceCents})
		quantitiesSnapshot = append(quantitiesSnapshot, map[string]interface{}{"id": id, "quantity": qty})
		lineIndex++
	}

	if totalCents <= 0 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "ordine non valido")
		return
	}

	createdSession, err := rt.createStripeCheckoutSession(form)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot create stripe checkout session")
		_ = writeJSONMessage(w, http.StatusBadGateway, "errore creazione checkout")
		return
	}

	productsJSON, _ := json.Marshal(productsSnapshot)
	quantitiesJSON, _ := json.Marshal(quantitiesSnapshot)

	createdOrder, err := rt.db.CreateBarOrder(database.BarOrder{
		OrganizationID:  ctx.OrganizationID,
		PartnerID:       0,
		ProductsJSON:    string(productsJSON),
		QuantitiesJSON:  string(quantitiesJSON),
		TotalCents:      int(totalCents),
		Sector:          payload.Sector,
		Row:             payload.Row,
		Seat:            payload.Seat,
		Notes:           payload.Notes,
		OrderStatus:     "pending",
		PaymentStatus:   "pending",
		StripeReference: createdSession.ID,
	})
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot persist bar order")
		_ = writeJSONMessage(w, http.StatusInternalServerError, "impossibile salvare ordine")
		return
	}

	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(barCheckoutResponse{CheckoutURL: createdSession.URL, SessionID: createdSession.ID, OrderID: createdOrder.ID})
}

func (rt *_router) confirmBarCheckoutSession(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if strings.TrimSpace(rt.stripeSecretKey) == "" {
		_ = writeJSONMessage(w, http.StatusFailedDependency, "configurazione Stripe mancante")
		return
	}

	var payload barConfirmCheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "payload non valido")
		return
	}
	payload.SessionID = strings.TrimSpace(payload.SessionID)
	if payload.SessionID == "" {
		_ = writeJSONMessage(w, http.StatusBadRequest, "session_id obbligatorio")
		return
	}

	s, err := rt.getStripeCheckoutSession(payload.SessionID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot fetch stripe session")
		_ = writeJSONMessage(w, http.StatusBadGateway, "errore verifica pagamento")
		return
	}

	if s.PaymentStatus == "paid" {
		if err := rt.db.UpdateBarOrderPaymentByStripeReference(payload.SessionID, "paid", "paid"); err != nil && !errors.Is(err, sql.ErrNoRows) {
			ctx.Logger.WithError(err).Error("cannot update bar order payment")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(barConfirmCheckoutResponse{Confirmed: s.PaymentStatus == "paid"})
}

func (rt *_router) handleBarStripeWebhook(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	secret := strings.TrimSpace(rt.stripeWebhookSecret)
	if secret == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "body non leggibile")
		return
	}

	if !verifyStripeWebhookSignature(payload, r.Header.Get("Stripe-Signature"), secret) {
		_ = writeJSONMessage(w, http.StatusBadRequest, "firma webhook non valida")
		return
	}

	var event struct {
		Type string `json:"type"`
		Data struct {
			Object struct {
				ID string `json:"id"`
			} `json:"object"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &event); err != nil {
		ctx.Logger.WithError(err).Warn("invalid stripe webhook payload")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sessionID := strings.TrimSpace(event.Data.Object.ID)
	if sessionID != "" {
		switch event.Type {
		case "checkout.session.completed":
			if err := rt.db.UpdateBarOrderPaymentByStripeReference(sessionID, "paid", "paid"); err != nil && !errors.Is(err, sql.ErrNoRows) {
				ctx.Logger.WithError(err).Error("cannot mark bar order paid from webhook")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		case "checkout.session.expired":
			_ = rt.db.UpdateBarOrderPaymentByStripeReference(sessionID, "expired", "cancelled")
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) createStripeCheckoutSession(values url.Values) (stripeCheckoutSessionResponse, error) {
	req, err := http.NewRequest(http.MethodPost, "https://api.stripe.com/v1/checkout/sessions", strings.NewReader(values.Encode()))
	if err != nil {
		return stripeCheckoutSessionResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+rt.stripeSecretKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return stripeCheckoutSessionResponse{}, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 400 {
		return stripeCheckoutSessionResponse{}, errors.New(string(body))
	}
	var out stripeCheckoutSessionResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return stripeCheckoutSessionResponse{}, err
	}
	return out, nil
}

func (rt *_router) getStripeCheckoutSession(sessionID string) (stripeCheckoutSessionResponse, error) {
	req, err := http.NewRequest(http.MethodGet, "https://api.stripe.com/v1/checkout/sessions/"+url.PathEscape(sessionID), nil)
	if err != nil {
		return stripeCheckoutSessionResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+rt.stripeSecretKey)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return stripeCheckoutSessionResponse{}, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 400 {
		return stripeCheckoutSessionResponse{}, errors.New(string(body))
	}
	var out stripeCheckoutSessionResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return stripeCheckoutSessionResponse{}, err
	}
	return out, nil
}

func verifyStripeWebhookSignature(payload []byte, signatureHeader string, secret string) bool {
	parts := strings.Split(signatureHeader, ",")
	var timestamp, sig string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "t=") {
			timestamp = strings.TrimPrefix(part, "t=")
		}
		if strings.HasPrefix(part, "v1=") {
			sig = strings.TrimPrefix(part, "v1=")
		}
	}
	if timestamp == "" || sig == "" {
		return false
	}
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}
	if time.Since(time.Unix(ts, 0)) > 5*time.Minute {
		return false
	}
	signedPayload := timestamp + "." + string(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(signedPayload))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(sig))
}

func resolveCheckoutRedirectBase(configured string, r *http.Request) string {
	if normalized, ok := normalizeAbsoluteURL(configured); ok {
		return normalized
	}

	if r != nil {
		if origin := strings.TrimSpace(r.Header.Get("Origin")); origin != "" {
			if normalized, ok := normalizeAbsoluteURL(origin); ok {
				return normalized
			}
		}

		if referer := strings.TrimSpace(r.Header.Get("Referer")); referer != "" {
			if parsed, err := url.Parse(referer); err == nil {
				if normalized, ok := normalizeAbsoluteURL(parsed.Scheme + "://" + parsed.Host); ok {
					return normalized
				}
			}
		}

		host := strings.TrimSpace(r.Header.Get("X-Forwarded-Host"))
		if host == "" {
			host = strings.TrimSpace(r.Host)
		}
		host = sanitizeHost(host)
		if host != "" {
			scheme := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))
			if scheme != "http" && scheme != "https" {
				scheme = "https"
			}
			if normalized, ok := normalizeAbsoluteURL(scheme + "://" + host); ok {
				return normalized
			}
		}
	}

	return "https://mvp.wearingcash.it/newui"
}

func normalizeAbsoluteURL(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", false
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	if parsed.Path == "" {
		parsed.Path = "/newui"
	}
	return parsed.String(), true
}

func sanitizeHost(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.Contains(raw, ",") {
		raw = strings.TrimSpace(strings.Split(raw, ",")[0])
	}
	if strings.Contains(raw, "://") {
		if parsed, err := url.Parse(raw); err == nil {
			raw = parsed.Host
		}
	}
	if host, _, err := net.SplitHostPort(raw); err == nil {
		raw = host
	}
	return strings.TrimSpace(raw)
}

type barOverviewResponse struct {
	OrdersReceived int                      `json:"orders_received"`
	OrdersPending  int                      `json:"orders_pending"`
	OrdersPrep     int                      `json:"orders_in_preparation"`
	OrdersReady    int                      `json:"orders_ready"`
	OrdersDone     int                      `json:"orders_completed"`
	RevenueCents   int                      `json:"revenue_cents"`
	AvgTicketCents int                      `json:"avg_ticket_cents"`
	LatestOrders   []database.BarOrder      `json:"latest_orders"`
	TopProducts    []map[string]interface{} `json:"top_products"`
}

func isBarStatusAllowed(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "nuovo", "new", "pending", "in preparazione", "in_preparazione", "preparing", "ready", "pronto", "completato", "completed", "annullato", "cancelled":
		return true
	default:
		return false
	}
}

func normalizeBarStatus(status string) string {
	s := strings.ToLower(strings.TrimSpace(status))
	switch s {
	case "nuovo", "new", "pending":
		return "new"
	case "in preparazione", "in_preparazione", "preparing":
		return "in_preparazione"
	case "pronto", "ready":
		return "pronto"
	case "completato", "completed":
		return "completato"
	case "annullato", "cancelled":
		return "annullato"
	default:
		return s
	}
}

func (rt *_router) resolveBarPartnerScope(ctx reqcontext.RequestContext, r *http.Request) int {
	if strings.EqualFold(ctx.AdminRole, "superadmin") {
		pid, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("partner_id")))
		if pid > 0 {
			return pid
		}
		return 0
	}
	if !strings.EqualFold(ctx.AdminRole, "bar") {
		return -1
	}
	partners, err := rt.db.ListPartners(ctx.OrganizationID)
	if err != nil {
		return -1
	}
	for _, p := range partners {
		if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(p.Username)), "BAR") && strings.EqualFold(strings.TrimSpace(p.Username), strings.TrimSpace(ctx.AdminUsername)) {
			return p.ID
		}
	}
	for _, p := range partners {
		if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(p.Username)), "BAR") {
			return p.ID
		}
	}
	return -1
}

func (rt *_router) listAdminBarOrders(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	partnerID := rt.resolveBarPartnerScope(ctx, r)
	if partnerID < 0 {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	status := normalizeBarStatus(r.URL.Query().Get("status"))
	orders, err := rt.db.ListBarOrders(ctx.OrganizationID, partnerID, status)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(orders)
}

func (rt *_router) updateAdminBarOrderStatus(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	partnerID := rt.resolveBarPartnerScope(ctx, r)
	if partnerID < 0 {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	order, err := rt.db.GetBarOrder(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if ctx.OrganizationID > 0 && order.OrganizationID != 0 && order.OrganizationID != ctx.OrganizationID {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	if partnerID > 0 && order.PartnerID > 0 && order.PartnerID != partnerID {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	var payload struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || !isBarStatusAllowed(payload.Status) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := rt.db.UpdateBarOrderStatus(id, normalizeBarStatus(payload.Status)); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) getAdminBarOverview(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	partnerID := rt.resolveBarPartnerScope(ctx, r)
	if partnerID < 0 {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	orders, err := rt.db.ListBarOrders(ctx.OrganizationID, partnerID, "")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	resp := barOverviewResponse{}
	productQty := map[string]int{}
	for _, o := range orders {
		resp.OrdersReceived++
		resp.RevenueCents += o.TotalCents
		s := normalizeBarStatus(o.OrderStatus)
		switch s {
		case "new", "pending":
			resp.OrdersPending++
		case "in_preparazione":
			resp.OrdersPrep++
		case "pronto":
			resp.OrdersReady++
		case "completato":
			resp.OrdersDone++
		}
		var qtyRows []map[string]interface{}
		_ = json.Unmarshal([]byte(o.QuantitiesJSON), &qtyRows)
		for _, q := range qtyRows {
			id, _ := q["id"].(string)
			v, _ := q["quantity"].(float64)
			productQty[id] += int(v)
		}
	}
	if resp.OrdersReceived > 0 {
		resp.AvgTicketCents = resp.RevenueCents / resp.OrdersReceived
	}
	if len(orders) > 8 {
		resp.LatestOrders = orders[:8]
	} else {
		resp.LatestOrders = orders
	}
	productNames := map[string]string{}
	for _, o := range orders {
		var productRows []map[string]interface{}
		_ = json.Unmarshal([]byte(o.ProductsJSON), &productRows)
		for _, row := range productRows {
			id, _ := row["id"].(string)
			name, _ := row["name"].(string)
			if strings.TrimSpace(id) != "" && strings.TrimSpace(name) != "" {
				productNames[id] = name
			}
		}
	}
	for id, qty := range productQty {
		if qty > 0 {
			resp.TopProducts = append(resp.TopProducts, map[string]interface{}{"id": id, "name": productNames[id], "qty": qty})
		}
	}
	_ = json.NewEncoder(w).Encode(resp)
}
