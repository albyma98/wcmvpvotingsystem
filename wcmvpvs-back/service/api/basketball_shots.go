package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
)

func (rt *_router) basketballIdentity(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) (int, int, string, bool) {
	eventID, err := parseNumericID(chi.URLParam(r, "eventId"))
	deviceID := rt.deviceIDFromRequest(r)
	if err != nil || ctx.OrganizationID <= 0 || deviceID == "" || len(deviceID) > 200 {
		w.WriteHeader(http.StatusBadRequest)
		return 0, 0, "", false
	}
	fanID := 0
	if token := rt.fanSessionTokenFromRequest(r); token != "" {
		fan, err := rt.db.GetFanBySessionToken(token, deviceID)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return 0, 0, "", false
		}
		if fan.Profile.OrganizationID != ctx.OrganizationID {
			w.WriteHeader(http.StatusForbidden)
			return 0, 0, "", false
		}
		fanID = fan.Profile.ID
	}
	return eventID, fanID, deviceID, true
}

func basketballWriteError(w http.ResponseWriter, ctx reqcontext.RequestContext, err error) {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		w.WriteHeader(http.StatusNotFound)
	case errors.Is(err, database.ErrBasketballClosed):
		_ = writeJSONMessage(w, http.StatusConflict, "La partita non è attiva.")
	case errors.Is(err, database.ErrBasketballExpired):
		_ = writeJSONMessage(w, http.StatusGone, "Questo tiro è scaduto. Prepara una nuova palla.")
	case errors.Is(err, database.ErrBasketballTooEarly):
		_ = writeJSONMessage(w, http.StatusConflict, "Attendi che il tiro sia completato.")
	case errors.Is(err, database.ErrBasketballGesture):
		w.WriteHeader(http.StatusBadRequest)
	default:
		ctx.Logger.WithError(err).Error("basketball shot")
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (rt *_router) createBasketballShot(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, fanID, deviceID, ok := rt.basketballIdentity(w, r, ctx)
	if !ok {
		return
	}
	shot, err := rt.db.CreateBasketballShot(eventID, ctx.OrganizationID, fanID, deviceID, time.Now())
	if err != nil {
		basketballWriteError(w, ctx, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	_ = writeJSON(w, http.StatusOK, shot)
}

func (rt *_router) submitBasketballShot(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, fanID, deviceID, ok := rt.basketballIdentity(w, r, ctx)
	if !ok {
		return
	}
	shotID := strings.TrimSpace(chi.URLParam(r, "shotId"))
	if len(shotID) != 36 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var payload struct {
		DX *float64 `json:"dx"`
		DY *float64 `json:"dy"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil || payload.DX == nil || payload.DY == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	result, err := rt.db.ResolveBasketballShot(eventID, ctx.OrganizationID, fanID, deviceID, shotID, *payload.DX, *payload.DY, time.Now())
	if err != nil {
		basketballWriteError(w, ctx, err)
		return
	}
	if result.CoinsEarned > 0 && rt.coinsHub != nil {
		rt.coinsHub.BroadcastThrottled(eventID, 2*time.Second)
	}
	w.Header().Set("Cache-Control", "no-store")
	_ = writeJSON(w, http.StatusOK, result)
}
