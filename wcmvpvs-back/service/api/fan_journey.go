package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
)

func (rt *_router) claimFanJourneyReward(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := parseNumericID(chi.URLParam(r, "eventId"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if !rt.ensureEventInOrganization(w, ctx, eventID) {
		return
	}
	deviceID := rt.deviceIDFromRequest(r)
	if deviceID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var payload struct {
		Key       string `json:"key"`
		RequestID string `json:"request_id"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	amount := database.FanJourneyRewardAmount(payload.Key)
	if amount == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if payload.RequestID != "" && (!database.ValidFanMinigameRoundID(payload.RequestID) || payload.Key == "vote") {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	fanID := 0
	if token := rt.fanSessionTokenFromRequest(r); token != "" {
		fan, err := rt.db.GetFanBySessionToken(token, deviceID)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if fan.Profile.OrganizationID != ctx.OrganizationID {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		fanID = fan.Profile.ID
	}
	var response interface{}
	if payload.RequestID != "" {
		response, err = rt.db.ClaimFanMinigameReward(eventID, ctx.OrganizationID, fanID, deviceID, payload.Key, payload.RequestID)
	} else {
		var wallet, credited int
		wallet, credited, err = rt.db.ClaimFanJourneyReward(eventID, ctx.OrganizationID, fanID, deviceID, payload.Key)
		response = map[string]interface{}{"wallet": wallet, "coins_earned": credited, "reward_amount": amount}
	}
	if err != nil {
		if errors.Is(err, database.ErrFanRewardDeviceUsed) {
			_ = writeJSONMessage(w, http.StatusConflict, "Questo risultato è già stato assegnato ad un altro profilo.")
			return
		}
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, database.ErrBasketballClosed) || errors.Is(err, database.ErrFanMinigameRound) {
			_ = writeJSONMessage(w, http.StatusConflict, "Completa l’attività prima di ritirare il bonus.")
		} else {
			ctx.Logger.WithError(err).Error("claim fan journey reward")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	rt.coinsHub.Broadcast(eventID)
	_ = writeJSON(w, http.StatusOK, response)
}
