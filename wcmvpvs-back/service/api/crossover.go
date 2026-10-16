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

func crossoverWriteError(w http.ResponseWriter, ctx reqcontext.RequestContext, err error) {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		_ = writeJSONMessage(w, http.StatusNotFound, "Sessione non trovata.")
	case errors.Is(err, database.ErrBasketballClosed):
		_ = writeJSONMessage(w, http.StatusConflict, "La partita non è attiva.")
	case errors.Is(err, database.ErrCrossoverExpired):
		_ = writeJSONMessage(w, http.StatusGone, "La sfida è terminata. Inizia una nuova partita.")
	case errors.Is(err, database.ErrCrossoverTooEarly):
		_ = writeJSONMessage(w, http.StatusConflict, "Attendi che il duello sia completato.")
	case errors.Is(err, database.ErrCrossoverOrder):
		_ = writeJSONMessage(w, http.StatusConflict, "Aggiorna la sessione per continuare dal prossimo duello.")
	case errors.Is(err, database.ErrCrossoverGesture):
		_ = writeJSONMessage(w, http.StatusBadRequest, "Movimento non valido.")
	default:
		ctx.Logger.WithError(err).Error("crossover duel")
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (rt *_router) createCrossoverSession(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, fanID, deviceID, ok := rt.basketballIdentity(w, r, ctx)
	if !ok {
		return
	}
	session, err := rt.db.CreateCrossoverSession(eventID, ctx.OrganizationID, fanID, deviceID, time.Now())
	if err != nil {
		crossoverWriteError(w, ctx, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	_ = writeJSON(w, http.StatusOK, session)
}

func (rt *_router) submitCrossoverDuel(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, fanID, deviceID, ok := rt.basketballIdentity(w, r, ctx)
	if !ok {
		return
	}
	sessionID := strings.TrimSpace(chi.URLParam(r, "sessionId"))
	if len(sessionID) != 36 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Sessione non valida.")
		return
	}
	var payload struct {
		Duel        *int `json:"duel"`
		DuelStartMS *int `json:"duel_start_ms"`
		Samples     []struct {
			TMS *int     `json:"t_ms"`
			X   *float64 `json:"x"`
		} `json:"samples"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil || payload.Duel == nil || payload.DuelStartMS == nil || len(payload.Samples) < 2 || len(payload.Samples) > 128 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Movimento non valido.")
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Movimento non valido.")
		return
	}
	samples := make([]database.CrossoverSample, 0, len(payload.Samples))
	for _, sample := range payload.Samples {
		if sample.TMS == nil || sample.X == nil {
			_ = writeJSONMessage(w, http.StatusBadRequest, "Movimento non valido.")
			return
		}
		samples = append(samples, database.CrossoverSample{TMS: *sample.TMS, X: *sample.X})
	}
	result, err := rt.db.ResolveCrossoverDuel(eventID, ctx.OrganizationID, fanID, deviceID, sessionID, *payload.Duel, *payload.DuelStartMS, samples, time.Now())
	if err != nil {
		crossoverWriteError(w, ctx, err)
		return
	}
	if result.CoinsEarned > 0 && rt.coinsHub != nil {
		rt.coinsHub.BroadcastThrottled(eventID, 2*time.Second)
	}
	w.Header().Set("Cache-Control", "no-store")
	_ = writeJSON(w, http.StatusOK, result)
}
