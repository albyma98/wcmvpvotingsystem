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

type createAdminBarCategoryPayload struct {
	Name     string `json:"name"`
	ImageURL string `json:"image_url"`
}

type createAdminBarMenuPayload struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	PriceCents  int    `json:"price_cents"`
	Items       []struct {
		ProductID int `json:"product_id"`
		Quantity  int `json:"quantity"`
	} `json:"items"`
}

func (rt *_router) listAdminBarProducts(w http.ResponseWriter, _ *http.Request, _ reqcontext.RequestContext) {
	products, err := rt.db.ListShopProducts()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(products)
}

func (rt *_router) createAdminBarProduct(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var payload createShopProductPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "payload non valido")
		return
	}
	payload.Name = strings.TrimSpace(payload.Name)
	payload.Description = strings.TrimSpace(payload.Description)
	payload.ImageURL = strings.TrimSpace(payload.ImageURL)
	if payload.Name == "" || payload.PriceCents <= 0 || payload.CategoryID <= 0 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "nome, prezzo e categoria validi obbligatori")
		return
	}
	if _, err := rt.db.GetBarCategory(payload.CategoryID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_ = writeJSONMessage(w, http.StatusBadRequest, "categoria selezionata non valida")
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	created, err := rt.db.CreateShopProduct(database.ShopProduct{Name: payload.Name, Description: payload.Description, PriceCents: payload.PriceCents, ImageURL: payload.ImageURL, CategoryID: payload.CategoryID})
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot create admin bar product")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func (rt *_router) deleteAdminBarProduct(w http.ResponseWriter, r *http.Request, _ reqcontext.RequestContext) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "identificativo prodotto non valido")
		return
	}
	if err := rt.db.SoftDeleteShopProduct(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) listAdminBarCategories(w http.ResponseWriter, _ *http.Request, _ reqcontext.RequestContext) {
	items, err := rt.db.ListBarCategories(false)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(items)
}

func (rt *_router) createAdminBarCategory(w http.ResponseWriter, r *http.Request, _ reqcontext.RequestContext) {
	var payload createAdminBarCategoryPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "payload non valido")
		return
	}
	payload.Name = strings.TrimSpace(payload.Name)
	payload.ImageURL = strings.TrimSpace(payload.ImageURL)
	if payload.Name == "" || payload.ImageURL == "" {
		_ = writeJSONMessage(w, http.StatusBadRequest, "nome e immagine categoria obbligatori")
		return
	}
	created, err := rt.db.CreateBarCategory(database.BarCategory{Name: payload.Name, ImageURL: payload.ImageURL})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func (rt *_router) updateAdminBarCategory(w http.ResponseWriter, r *http.Request, _ reqcontext.RequestContext) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "identificativo categoria non valido")
		return
	}
	var payload createAdminBarCategoryPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "payload non valido")
		return
	}
	payload.Name = strings.TrimSpace(payload.Name)
	payload.ImageURL = strings.TrimSpace(payload.ImageURL)
	if payload.Name == "" || payload.ImageURL == "" {
		_ = writeJSONMessage(w, http.StatusBadRequest, "nome e immagine categoria obbligatori")
		return
	}
	updated, err := rt.db.UpdateBarCategory(database.BarCategory{ID: id, Name: payload.Name, ImageURL: payload.ImageURL})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(updated)
}

func (rt *_router) deleteAdminBarCategory(w http.ResponseWriter, r *http.Request, _ reqcontext.RequestContext) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "identificativo categoria non valido")
		return
	}
	if err := rt.db.SoftDeleteBarCategory(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) listAdminBarMenus(w http.ResponseWriter, _ *http.Request, _ reqcontext.RequestContext) {
	menus, err := rt.db.ListBarMenus(false)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(menus)
}

func (rt *_router) createAdminBarMenu(w http.ResponseWriter, r *http.Request, _ reqcontext.RequestContext) {
	var payload createAdminBarMenuPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "payload non valido")
		return
	}
	payload.Name = strings.TrimSpace(payload.Name)
	payload.Description = strings.TrimSpace(payload.Description)
	if payload.Name == "" || payload.PriceCents <= 0 || len(payload.Items) == 0 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "nome, prezzo e prodotti del menu sono obbligatori")
		return
	}
	items := make([]database.BarMenuItem, 0, len(payload.Items))
	for _, item := range payload.Items {
		items = append(items, database.BarMenuItem{ProductID: item.ProductID, Quantity: item.Quantity})
	}
	created, err := rt.db.CreateBarMenu(database.BarMenu{Name: payload.Name, Description: payload.Description, PriceCents: payload.PriceCents, Items: items})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_ = writeJSONMessage(w, http.StatusBadRequest, "uno o più prodotti selezionati non esistono")
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func (rt *_router) deleteAdminBarMenu(w http.ResponseWriter, r *http.Request, _ reqcontext.RequestContext) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "identificativo menu non valido")
		return
	}
	if err := rt.db.SoftDeleteBarMenu(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
