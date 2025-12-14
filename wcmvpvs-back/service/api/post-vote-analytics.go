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

type postVoteActionPayload struct {
	Action   string `json:"action"`
	DeviceID string `json:"device_id,omitempty"`
}

func (rt *_router) recordPostVoteAction(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(chi.URLParam(r, "eventId"))
	if err != nil || eventID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var payload postVoteActionPayload
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048)).Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
		ctx.Logger.WithError(err).Warn("invalid post vote action payload")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	action := strings.TrimSpace(payload.Action)
	deviceID := strings.TrimSpace(payload.DeviceID)
	if deviceID == "" {
		deviceID = rt.deviceIDFromRequest(r)
	}

	if err := rt.db.RecordPostVoteAction(eventID, deviceID, action); err != nil {
		switch {
		case errors.Is(err, database.ErrInvalidSponsorData):
			w.WriteHeader(http.StatusBadRequest)
		case errors.Is(err, sql.ErrNoRows):
			w.WriteHeader(http.StatusNotFound)
		default:
			ctx.Logger.WithError(err).WithField("event_id", eventID).Warn("cannot record post vote action")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
