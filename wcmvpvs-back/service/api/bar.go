package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
)

type barProduct struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	PriceCents  int64  `json:"price_cents"`
	ImageEmoji  string `json:"image_emoji"`
	Description string `json:"description"`
}

var barProductsCatalog = []barProduct{
	{ID: "beer_40", Name: "Birra 40cl", PriceCents: 500, ImageEmoji: "🍺", Description: "Birra fresca 40cl"},
	{ID: "coca_cola", Name: "Coca-Cola", PriceCents: 300, ImageEmoji: "🥤", Description: "Lattina Coca-Cola"},
	{ID: "chips", Name: "Patatine", PriceCents: 250, ImageEmoji: "🍟", Description: "Porzione patatine"},
	{ID: "sandwich", Name: "Panino", PriceCents: 650, ImageEmoji: "🥪", Description: "Panino farcito"},
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

func mapBarProducts() map[string]barProduct {
	mapped := make(map[string]barProduct, len(barProductsCatalog))
	for _, p := range barProductsCatalog {
		mapped[p.ID] = p
	}
	return mapped
}

func (rt *_router) listBarProducts(w http.ResponseWriter, _ *http.Request, _ reqcontext.RequestContext) {
	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(barProductsCatalog)
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

	catalog := mapBarProducts()
	qtyMap := map[string]int64{}
	for _, item := range payload.Items {
		id := strings.TrimSpace(item.ProductID)
		if _, ok := catalog[id]; !ok || item.Quantity <= 0 {
			_ = writeJSONMessage(w, http.StatusBadRequest, "elementi ordine non validi")
			return
		}
		qtyMap[id] += item.Quantity
	}

	form := url.Values{}
	form.Set("mode", "payment")
	base := strings.TrimRight(strings.TrimSpace(rt.stripeSuccessURL), "/")
	if base == "" {
		base = "http://localhost:5173/newui"
	}
	form.Set("success_url", base+"?barOrderSuccess=1&session_id={CHECKOUT_SESSION_ID}")
	form.Set("cancel_url", base+"?barOrderCancelled=1")
	form.Set("payment_method_types[0]", "card")

	totalCents := int64(0)
	productsSnapshot := make([]map[string]interface{}, 0, len(qtyMap))
	quantitiesSnapshot := make([]map[string]interface{}, 0, len(qtyMap))
	lineIndex := 0
	for _, product := range barProductsCatalog {
		qty := qtyMap[product.ID]
		if qty <= 0 {
			continue
		}
		prefix := "line_items[" + strconv.Itoa(lineIndex) + "]"
		form.Set(prefix+"[quantity]", strconv.FormatInt(qty, 10))
		form.Set(prefix+"[price_data][currency]", "eur")
		form.Set(prefix+"[price_data][unit_amount]", strconv.FormatInt(product.PriceCents, 10))
		form.Set(prefix+"[price_data][product_data][name]", product.Name)
		totalCents += product.PriceCents * qty
		productsSnapshot = append(productsSnapshot, map[string]interface{}{"id": product.ID, "name": product.Name, "price_cents": product.PriceCents})
		quantitiesSnapshot = append(quantitiesSnapshot, map[string]interface{}{"id": product.ID, "quantity": qty})
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
