package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
)

func (rt *_router) getActiveEvent(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	event, err := rt.db.GetActiveEvent()
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			ctx.Logger.Info("no active event configured")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		ctx.Logger.WithError(err).Error("cannot fetch active event")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(event)
	ctx.Logger.WithField("event_id", event.ID).Info("returned active event")
}
