package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
)

type aiBarUpsellAPIRequest struct {
	Trigger               string           `json:"trigger"`
	EventID               int              `json:"event_id,omitempty"`
	EventPhase            string           `json:"event_phase,omitempty"`
	Cart                  []aiCartItem     `json:"cart"`
	PurchaseHistory       []string         `json:"purchase_history,omitempty"`
	AdminPriorityProducts []string         `json:"admin_priority_products,omitempty"`
	AvailableProducts     []aiProductInput `json:"available_products,omitempty"`
}

type aiPopupAPIRequest struct {
	TriggerType         string                   `json:"trigger_type"`
	EventID             int                      `json:"event_id,omitempty"`
	EventPhase          string                   `json:"event_phase,omitempty"`
	Objective           string                   `json:"objective"`
	SessionsCount       int                      `json:"sessions_count,omitempty"`
	GamesPlayed         int                      `json:"games_played,omitempty"`
	Coins               int                      `json:"coins,omitempty"`
	LastGame            string                   `json:"last_game,omitempty"`
	LastPurchase        string                   `json:"last_purchase,omitempty"`
	InactiveSeconds     int                      `json:"inactive_seconds,omitempty"`
	CartItemsCount      int                      `json:"cart_items_count,omitempty"`
	CartTotalCents      int                      `json:"cart_total_cents,omitempty"`
	SessionID           string                   `json:"session_id,omitempty"`
	PopupHistorySession []map[string]interface{} `json:"popup_history_session,omitempty"`
	Extra               map[string]interface{}   `json:"extra,omitempty"`
}

func (rt *_router) generateAIUpsellSuggestions(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var payload aiBarUpsellAPIRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "payload non valido")
		return
	}
	if len(payload.Cart) == 0 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "carrello richiesto")
		return
	}
	available := payload.AvailableProducts
	if len(available) == 0 {
		products, err := rt.db.ListShopProducts()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		prioritySet := map[string]struct{}{}
		for _, name := range payload.AdminPriorityProducts {
			prioritySet[strings.ToLower(strings.TrimSpace(name))] = struct{}{}
		}
		for _, product := range products {
			_, priority := prioritySet[strings.ToLower(strings.TrimSpace(product.Name))]
			available = append(available, aiProductInput{ProductID: "product:" + strconv.Itoa(product.ID), Name: product.Name, Category: product.Category, Available: true, Visible: true, Priority: priority, PriceCents: product.PriceCents})
		}
	}
	userID, sessionID := getAIIdentity(r, ctx.OrganizationID, rt.db)
	trackingSignals, _ := rt.db.ListRecentTrackingSignals(payload.EventID, sessionID, 12)
	serviceReq := aiUpsellRequest{UserID: userID, SessionID: sessionID, Trigger: strings.TrimSpace(payload.Trigger), EventID: payload.EventID, EventPhase: payload.EventPhase, Cart: payload.Cart, PurchaseHistory: payload.PurchaseHistory, AdminPriorityProducts: payload.AdminPriorityProducts, AvailableProducts: available, TrackingSignals: trackingSignals}
	response := rt.aiService.GenerateUpsellSuggestions(r.Context(), serviceReq)
	inputJSON, _ := json.Marshal(serviceReq)
	outputJSON, _ := json.Marshal(response)
	logItem, err := rt.db.CreateAIInteractionLog(database.AIInteractionLog{FeatureType: "upsell", Trigger: payload.Trigger, UserID: userID, SessionID: sessionID, OrganizationID: ctx.OrganizationID, EventID: payload.EventID, InputJSON: string(inputJSON), OutputJSON: string(outputJSON), Status: "generated"})
	if err == nil {
		response.Interaction = logItem.ID
	}
	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func (rt *_router) generateAIPopupMessage(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var payload aiPopupAPIRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "payload non valido")
		return
	}
	payload.SessionID = strings.TrimSpace(payload.SessionID)
	if payload.SessionID == "" {
		_, payload.SessionID = getAIIdentity(r, ctx.OrganizationID, rt.db)
	}
	if payload.TriggerType == "" || payload.Objective == "" {
		_ = writeJSONMessage(w, http.StatusBadRequest, "trigger e objective sono obbligatori")
		return
	}
	if payload.SessionID != "" {
		state, err := getPopupSessionState(rt.db, payload.SessionID, payload.TriggerType, rt.aiService.cfg.MaxPopupsSession, rt.aiService.cfg.PopupCooldown)
		if err == nil && state.WithinCooldown {
			w.Header().Set("content-type", "application/json")
			_ = json.NewEncoder(w).Encode(aiPopupResponse{Source: "suppressed", CooldownSecs: int(rt.aiService.cfg.PopupCooldown.Seconds())})
			return
		}
	}
	userID, _ := getAIIdentity(r, ctx.OrganizationID, rt.db)
	trackingSignals, _ := rt.db.ListRecentTrackingSignals(payload.EventID, payload.SessionID, 12)
	serviceReq := aiPopupRequest{UserID: userID, SessionID: payload.SessionID, TriggerType: payload.TriggerType, EventID: payload.EventID, EventPhase: payload.EventPhase, Objective: payload.Objective, SessionsCount: payload.SessionsCount, GamesPlayed: payload.GamesPlayed, Coins: payload.Coins, LastGame: payload.LastGame, LastPurchase: payload.LastPurchase, InactiveSeconds: payload.InactiveSeconds, CartItemsCount: payload.CartItemsCount, CartTotalCents: payload.CartTotalCents, PopupHistorySession: payload.PopupHistorySession, TrackingSignals: trackingSignals, Extra: payload.Extra}
	response := rt.aiService.GeneratePopupMessage(r.Context(), serviceReq)
	inputJSON, _ := json.Marshal(serviceReq)
	outputJSON, _ := json.Marshal(response)
	logItem, err := rt.db.CreateAIInteractionLog(database.AIInteractionLog{FeatureType: "popup", Trigger: payload.TriggerType, UserID: userID, SessionID: payload.SessionID, OrganizationID: ctx.OrganizationID, EventID: payload.EventID, InputJSON: string(inputJSON), OutputJSON: string(outputJSON), Status: "generated"})
	if err == nil {
		response.InteractionID = logItem.ID
	}
	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func (rt *_router) trackAIInteractionOutcome(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	_ = ctx
	interactionID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || interactionID <= 0 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "interaction id non valido")
		return
	}
	var payload aiInteractionTrackRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "payload non valido")
		return
	}
	occurredAt := time.Now().UTC()
	if strings.TrimSpace(payload.OccurredAt) != "" {
		if parsed, err := time.Parse(time.RFC3339, payload.OccurredAt); err == nil {
			occurredAt = parsed.UTC()
		}
	}
	if err := rt.db.UpdateAIInteractionOutcome(interactionID, payload.Outcome, occurredAt); err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
