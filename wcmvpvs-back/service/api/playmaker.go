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

func playmakerWriteError(w http.ResponseWriter, ctx reqcontext.RequestContext, err error) {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		_ = writeJSONMessage(w, http.StatusNotFound, "Sessione non trovata.")
	case errors.Is(err, database.ErrBasketballClosed):
		_ = writeJSONMessage(w, http.StatusConflict, "La partita non è attiva.")
	case errors.Is(err, database.ErrPlaymakerExpired):
		_ = writeJSONMessage(w, http.StatusGone, "La sfida è terminata. Inizia una nuova partita.")
	case errors.Is(err, database.ErrPlaymakerTooEarly):
		_ = writeJSONMessage(w, http.StatusConflict, "Attendi che il passaggio sia completato.")
	case errors.Is(err, database.ErrPlaymakerAttempt):
		_ = writeJSONMessage(w, http.StatusConflict, "Aggiorna la sessione per continuare dal prossimo passaggio.")
	case errors.Is(err, database.ErrPlaymakerGesture):
		_ = writeJSONMessage(w, http.StatusBadRequest, "Passaggio non valido.")
	default:
		ctx.Logger.WithError(err).Error("playmaker pass")
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (rt *_router) createPlaymakerSession(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, fanID, deviceID, ok := rt.basketballIdentity(w, r, ctx)
	if !ok {
		return
	}
	session, err := rt.db.CreatePlaymakerSession(eventID, ctx.OrganizationID, fanID, deviceID, time.Now())
	if err != nil {
		playmakerWriteError(w, ctx, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	_ = writeJSON(w, http.StatusOK, session)
}

func (rt *_router) submitPlaymakerPass(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, fanID, deviceID, ok := rt.basketballIdentity(w, r, ctx)
	if !ok {
		return
	}
	sessionID := strings.TrimSpace(chi.URLParam(r, "sessionId"))
	if len(sessionID) != 36 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var payload struct {
		Attempt   *int     `json:"attempt"`
		ElapsedMS *int     `json:"elapsed_ms"`
		DX        *float64 `json:"dx"`
		DY        *float64 `json:"dy"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil || payload.Attempt == nil || payload.ElapsedMS == nil || payload.DX == nil || payload.DY == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	result, err := rt.db.ResolvePlaymakerPass(eventID, ctx.OrganizationID, fanID, deviceID, sessionID, *payload.Attempt, *payload.ElapsedMS, *payload.DX, *payload.DY, time.Now())
	if err != nil {
		playmakerWriteError(w, ctx, err)
		return
	}
	if result.CoinsEarned > 0 && rt.coinsHub != nil {
		rt.coinsHub.BroadcastThrottled(eventID, 2*time.Second)
	}
	w.Header().Set("Cache-Control", "no-store")
	_ = writeJSON(w, http.StatusOK, result)
}
