package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
)

type engagementPayload struct {
	DurationSeconds int    `json:"duration_seconds"`
	DeviceID        string `json:"device_id,omitempty"`
}

func (rt *_router) recordPageEngagement(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(chi.URLParam(r, "eventId"))
	if err != nil || eventID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var payload engagementPayload
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048)).Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
		ctx.Logger.WithError(err).Warn("invalid engagement payload")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if payload.DurationSeconds <= 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	deviceID := strings.TrimSpace(payload.DeviceID)
	if deviceID == "" {
		deviceID = rt.deviceIDFromRequest(r)
	}

	if err := rt.db.RecordEngagementSession(eventID, deviceID, payload.DurationSeconds); err != nil {
		switch {
		case errors.Is(err, database.ErrInvalidSponsorData):
			w.WriteHeader(http.StatusBadRequest)
		case errors.Is(err, sql.ErrNoRows):
			w.WriteHeader(http.StatusNotFound)
		default:
			ctx.Logger.WithError(err).WithField("event_id", eventID).Warn("cannot record page engagement")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) getEventEngagementStats(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(chi.URLParam(r, "eventId"))
	if err != nil || eventID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	stats, err := rt.db.GetEventEngagement(eventID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		ctx.Logger.WithError(err).WithField("event_id", eventID).Error("cannot load engagement stats")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(stats)
}
