package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/go-chi/chi/v5"
)

const reactionTestCooldown = time.Minute

type reactionTestStatusResponse struct {
	Attempts      int     `json:"attempts"`
	AverageMs     float64 `json:"average_ms"`
	LastResultMs  *int    `json:"last_result_ms,omitempty"`
	LastAttemptAt *string `json:"last_attempt_at,omitempty"`
	NextAllowedAt *string `json:"next_allowed_at,omitempty"`
}

type reactionTestResultRequest struct {
	ReactionTimeMs int `json:"reaction_time_ms"`
}

type reactionTestCooldownResponse struct {
	Message       string `json:"message"`
	NextAllowedAt string `json:"next_allowed_at"`
}

type reactionTestResultResponse struct {
	ReactionTimeMs    int     `json:"reaction_time_ms"`
	AverageMs         float64 `json:"average_ms"`
	Attempts          int     `json:"attempts"`
	FasterThanAverage bool    `json:"faster_than_average"`
	NextAllowedAt     string  `json:"next_allowed_at"`
}

func (rt *_router) getReactionTestStatus(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(chi.URLParam(r, "eventId"))
	if err != nil || eventID <= 0 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Evento non valido.")
		return
	}

	deviceID := rt.deviceIDFromRequest(r)
	if deviceID == "" {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Identificativo dispositivo mancante.")
		return
	}

	stats, err := rt.db.GetReactionTestStats(eventID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot fetch reaction test stats")
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Impossibile recuperare le statistiche del Reaction Test.")
		return
	}

	response := reactionTestStatusResponse{
		Attempts:  stats.Attempts,
		AverageMs: stats.Average,
	}

	attempt, err := rt.db.GetLatestReactionTestAttempt(eventID, deviceID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			ctx.Logger.WithError(err).Error("cannot fetch last reaction test attempt")
			_ = writeJSONMessage(w, http.StatusInternalServerError, "Impossibile recuperare lo stato del Reaction Test.")
			return
		}
	} else {
		if attempt.IsValid {
			response.LastResultMs = &attempt.ReactionTimeMs
		}
		if !attempt.CreatedAt.IsZero() {
			ts := attempt.CreatedAt.UTC().Format(time.RFC3339)
			response.LastAttemptAt = &ts
			allowAt := attempt.CreatedAt.Add(reactionTestCooldown)
			if time.Until(allowAt) > 0 {
				next := allowAt.UTC().Format(time.RFC3339)
				response.NextAllowedAt = &next
			}
		}
	}

	if err := writeJSON(w, http.StatusOK, response); err != nil {
		ctx.Logger.WithError(err).Error("cannot encode reaction test status")
	}
}

func (rt *_router) postReactionTestResult(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(chi.URLParam(r, "eventId"))
	if err != nil || eventID <= 0 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Evento non valido.")
		return
	}

	deviceID := rt.deviceIDFromRequest(r)
	if deviceID == "" {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Identificativo dispositivo mancante.")
		return
	}

	var payload reactionTestResultRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Formato richiesta non valido.")
		return
	}

	reactionMs := payload.ReactionTimeMs
	if reactionMs < 50 || reactionMs > 10000 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Tempo di reazione non valido.")
		return
	}

	if attempt, err := rt.db.GetLatestReactionTestAttempt(eventID, deviceID); err == nil {
		if !attempt.CreatedAt.IsZero() {
			allowAt := attempt.CreatedAt.Add(reactionTestCooldown)
			if time.Until(allowAt) > 0 {
				next := allowAt.UTC().Format(time.RFC3339)
				_ = writeJSON(w, http.StatusTooManyRequests, reactionTestCooldownResponse{
					Message:       "Puoi riprovare tra pochissimo!",
					NextAllowedAt: next,
				})
				return
			}
		}
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		ctx.Logger.WithError(err).Error("cannot check reaction test cooldown")
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Impossibile registrare il tentativo ora.")
		return
	}

	attempt, err := rt.db.RecordReactionTestAttempt(eventID, deviceID, reactionMs)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot record reaction test attempt")
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Impossibile salvare il risultato del Reaction Test.")
		return
	}

	stats, err := rt.db.GetReactionTestStats(eventID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot refresh reaction test stats")
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Impossibile aggiornare le statistiche del Reaction Test.")
		return
	}

	allowAt := attempt.CreatedAt
	if allowAt.IsZero() {
		allowAt = time.Now().UTC()
	}
	allowAt = allowAt.Add(reactionTestCooldown)

	faster := stats.Average > 0 && float64(reactionMs) < stats.Average

	response := reactionTestResultResponse{
		ReactionTimeMs:    reactionMs,
		AverageMs:         stats.Average,
		Attempts:          stats.Attempts,
		FasterThanAverage: faster,
		NextAllowedAt:     allowAt.UTC().Format(time.RFC3339),
	}

	if err := writeJSON(w, http.StatusCreated, response); err != nil {
		ctx.Logger.WithError(err).Error("cannot encode reaction test result")
	}
}
