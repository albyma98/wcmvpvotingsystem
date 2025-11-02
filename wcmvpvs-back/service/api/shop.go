package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
)

type checkoutItemPayload struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}

type checkoutRequestPayload struct {
	CustomerName  string                `json:"customer_name"`
	CustomerEmail string                `json:"customer_email"`
	CustomerNotes string                `json:"customer_notes"`
	Items         []checkoutItemPayload `json:"items"`
}

type checkoutResponsePayload struct {
	Order database.ShopOrder `json:"order"`
}

func (rt *_router) listShopProducts(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	products, err := rt.db.ListShopProducts()
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list shop products")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(products)
	ctx.Logger.WithField("products", len(products)).Info("listed shop products")
}

func (rt *_router) getShopProduct(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "identificativo prodotto non valido")
		return
	}

	product, err := rt.db.GetShopProduct(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		ctx.Logger.WithError(err).Error("cannot load shop product")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(product)
	ctx.Logger.WithField("product_id", product.ID).Info("loaded shop product")
}

func (rt *_router) checkoutShopOrder(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var payload checkoutRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		ctx.Logger.WithError(err).Warn("invalid checkout payload")
		_ = writeJSONMessage(w, http.StatusBadRequest, "payload non valido")
		return
	}

	payload.CustomerName = strings.TrimSpace(payload.CustomerName)
	payload.CustomerEmail = strings.TrimSpace(payload.CustomerEmail)
	payload.CustomerNotes = strings.TrimSpace(payload.CustomerNotes)

	if payload.CustomerName == "" || payload.CustomerEmail == "" {
		_ = writeJSONMessage(w, http.StatusBadRequest, "inserisci nome ed email per completare l'ordine")
		return
	}

	if len(payload.Items) == 0 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "aggiungi almeno un prodotto al carrello")
		return
	}

	type aggregatedItem struct {
		quantity int
	}

	aggregated := make(map[int]*aggregatedItem)
	orderedIDs := make([]int, 0, len(payload.Items))
	for _, item := range payload.Items {
		if item.ProductID <= 0 || item.Quantity <= 0 {
			_ = writeJSONMessage(w, http.StatusBadRequest, "articolo non valido nel carrello")
			return
		}

		if existing, ok := aggregated[item.ProductID]; ok {
			existing.quantity += item.Quantity
			continue
		}

		aggregated[item.ProductID] = &aggregatedItem{quantity: item.Quantity}
		orderedIDs = append(orderedIDs, item.ProductID)
	}

	if len(aggregated) == 0 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "carrello vuoto")
		return
	}

	orderItems := make([]database.ShopOrderItem, 0, len(aggregated))
	totalCents := 0

	for _, productID := range orderedIDs {
		info := aggregated[productID]
		product, err := rt.db.GetShopProduct(productID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				_ = writeJSONMessage(w, http.StatusBadRequest, "uno dei prodotti selezionati non è più disponibile")
				return
			}
			ctx.Logger.WithError(err).Error("cannot retrieve product for checkout")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		orderItems = append(orderItems, database.ShopOrderItem{
			ProductID:       product.ID,
			ProductName:     product.Name,
			ProductImageURL: product.ImageURL,
			Quantity:        info.quantity,
			UnitPriceCents:  product.PriceCents,
		})
		totalCents += product.PriceCents * info.quantity
	}

	if totalCents <= 0 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "impossibile calcolare il totale dell'ordine")
		return
	}

	order, err := rt.db.CreateShopOrder(database.ShopOrder{
		CustomerName:  payload.CustomerName,
		CustomerEmail: payload.CustomerEmail,
		CustomerNotes: payload.CustomerNotes,
		TotalCents:    totalCents,
	}, orderItems)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot create shop order")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(checkoutResponsePayload{Order: order})
	ctx.Logger.WithFields(map[string]interface{}{
		"order_id":    order.ID,
		"total_cents": order.TotalCents,
		"items":       len(order.Items),
	}).Info("shop order created")
}
