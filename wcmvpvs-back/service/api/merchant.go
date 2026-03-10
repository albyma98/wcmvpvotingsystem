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

func (rt *_router) merchantLogin(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var payload struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if payload.Username == "" || payload.Password == "" || ctx.OrganizationID == 0 || ctx.OrganizationSlug == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	admin, err := rt.db.GetAdminByUsername(payload.Username, ctx.OrganizationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	isBar := strings.HasPrefix(strings.ToUpper(strings.TrimSpace(admin.DisplayName)), "BAR")
	if !strings.EqualFold(admin.Role, "partner") || !isBar || !adminPasswordMatches(admin.PasswordHash, payload.Password) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	token, err := rt.createPartnerSession(admin.ID, admin.Username, admin.DisplayName, ctx.OrganizationID, ctx.OrganizationSlug, true)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(struct {
		Token       string `json:"token"`
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
	}{Token: token, Username: admin.Username, DisplayName: admin.DisplayName})
}

func (rt *_router) merchantDashboardSummary(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	orders, err := rt.db.ListMerchantBarOrders(ctx.MerchantID, nil, "", "")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	s := map[string]int{"new": 0, "in_preparazione": 0, "pronto": 0, "completato": 0, "annullato": 0}
	total := 0
	for _, o := range orders {
		total += o.TotalCents
		status := normalizeMerchantStatus(o.OrderStatus)
		s[status]++
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"orders_received":     len(orders),
		"orders_pending":      s["new"],
		"orders_preparing":    s["in_preparazione"],
		"orders_ready":        s["pronto"],
		"orders_completed":    s["completato"],
		"total_revenue_cents": total,
	})
}

func normalizeMerchantStatus(raw string) string {
	s := strings.TrimSpace(strings.ToLower(raw))
	switch s {
	case "new", "nuovo":
		return "new"
	case "in_preparazione", "in preparazione", "preparing", "pending", "paid":
		return "in_preparazione"
	case "ready", "pronto":
		return "pronto"
	case "completed", "completato":
		return "completato"
	case "cancelled", "annullato":
		return "annullato"
	default:
		return "new"
	}
}

func (rt *_router) listMerchantOrders(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	statusParam := strings.TrimSpace(r.URL.Query().Get("status"))
	statuses := []string{}
	if statusParam != "" && statusParam != "all" {
		statuses = append(statuses, statusParam)
	}
	orders, err := rt.db.ListMerchantBarOrders(ctx.MerchantID, statuses, r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(orders)
}

func (rt *_router) updateMerchantOrderStatus(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	var payload struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	status := normalizeMerchantStatus(payload.Status)
	if err := rt.db.UpdateMerchantBarOrderStatus(id, ctx.MerchantID, status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) listMerchantProducts(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	items, err := rt.db.ListMerchantProducts(ctx.MerchantID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(items)
}

func (rt *_router) upsertMerchantProduct(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var payload struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		PriceCents  int    `json:"price_cents"`
		IsActive    bool   `json:"is_active"`
		IsAvailable bool   `json:"is_available"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	item, err := rt.db.UpsertMerchantProduct(database.MerchantProduct{ID: payload.ID, MerchantID: ctx.MerchantID, OrganizationID: ctx.OrganizationID, Name: payload.Name, PriceCents: payload.PriceCents, IsActive: payload.IsActive, IsAvailable: payload.IsAvailable})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	_ = json.NewEncoder(w).Encode(item)
}

func (rt *_router) updateMerchantProductFlags(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	var payload struct {
		IsActive    bool `json:"is_active"`
		IsAvailable bool `json:"is_available"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := rt.db.UpdateMerchantProductFlags(id, ctx.MerchantID, payload.IsActive, payload.IsAvailable); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
