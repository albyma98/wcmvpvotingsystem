package api

import (
	"net/http"
	"strconv"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/go-chi/chi/v5"
)

type voteCountResponse struct {
	Total int `json:"total"`
}

func (rt *_router) getEventVoteCount(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	idParam := chi.URLParam(r, "id")
	eventID, err := strconv.Atoi(idParam)
	if err != nil || eventID <= 0 {
		ctx.Logger.WithField("event_id", idParam).Warn("invalid event id for vote count")
		_ = writeJSONMessage(w, http.StatusBadRequest, "Identificativo evento non valido.")
		return
	}

	count, err := rt.db.GetEventVoteCount(eventID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot fetch vote count")
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Impossibile recuperare il totale voti in questo momento.")
		return
	}

	if err := writeJSON(w, http.StatusOK, voteCountResponse{Total: count}); err != nil {
		ctx.Logger.WithError(err).Error("cannot encode vote count response")
	}
}
