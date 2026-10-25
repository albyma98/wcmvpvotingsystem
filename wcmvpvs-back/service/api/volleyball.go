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

func volleyballWriteError(w http.ResponseWriter, ctx reqcontext.RequestContext, err error) {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		w.WriteHeader(http.StatusNotFound)
	case errors.Is(err, database.ErrVolleyballClosed):
		_ = writeJSONMessage(w, http.StatusConflict, "La partita non è attiva.")
	case errors.Is(err, database.ErrVolleyballExpired):
		_ = writeJSONMessage(w, http.StatusGone, "Questa palla è scaduta. Prepara una nuova azione.")
	case errors.Is(err, database.ErrVolleyballTooEarly):
		_ = writeJSONMessage(w, http.StatusConflict, "Completa l'azione prima di inviare il risultato.")
	case errors.Is(err, database.ErrVolleyballGesture), errors.Is(err, database.ErrVolleyballGame):
		w.WriteHeader(http.StatusBadRequest)
	default:
		ctx.Logger.WithError(err).Error("volleyball round")
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (rt *_router) createVolleyballRound(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	gameID := chi.URLParam(r, "gameId")
	if !database.ValidVolleyballGame(gameID) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	eventID, fanID, deviceID, ok := rt.basketballIdentity(w, r, ctx)
	if !ok {
		return
	}
	round, err := rt.db.CreateVolleyballRound(eventID, ctx.OrganizationID, fanID, deviceID, gameID, time.Now())
	if err != nil {
		volleyballWriteError(w, ctx, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	_ = writeJSON(w, http.StatusOK, round)
}

func (rt *_router) submitVolleyballRound(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	gameID := chi.URLParam(r, "gameId")
	if !database.ValidVolleyballGame(gameID) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	eventID, fanID, deviceID, ok := rt.basketballIdentity(w, r, ctx)
	if !ok {
		return
	}
	roundID := strings.TrimSpace(chi.URLParam(r, "roundId"))
	if len(roundID) != 36 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var payload struct {
		AimX   *float64 `json:"aim_x"`
		AimY   *float64 `json:"aim_y"`
		Timing *float64 `json:"timing"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil || payload.AimX == nil || payload.AimY == nil || payload.Timing == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	result, err := rt.db.ResolveVolleyballRound(eventID, ctx.OrganizationID, fanID, deviceID, gameID, roundID, *payload.AimX, *payload.AimY, *payload.Timing, time.Now())
	if err != nil {
		volleyballWriteError(w, ctx, err)
		return
	}
	if result.CoinsEarned > 0 && rt.coinsHub != nil {
		rt.coinsHub.BroadcastThrottled(eventID, 2*time.Second)
	}
	w.Header().Set("Cache-Control", "no-store")
	_ = writeJSON(w, http.StatusOK, result)
}
