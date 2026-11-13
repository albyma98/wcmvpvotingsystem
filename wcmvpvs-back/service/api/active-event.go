package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
)

func (rt *_router) getActiveEvent(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if ctx.OrganizationID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	event, err := rt.db.GetActiveEvent(ctx.OrganizationID)
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
	validateMatchdayRequestEvent(r, event.ID)

	event.AvailableMinigames, err = rt.db.GetOrganizationMinigames(ctx.OrganizationID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot fetch active event minigames")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("content-type", "application/json")
	if rt.matchdayRequests != nil && r.URL.Query().Get("matchday") == "1" && event.IsActive && !event.IsConcluded {
		_ = json.NewEncoder(w).Encode(struct {
			database.Event
			MatchdayGrant string `json:"matchday_grant"`
		}{Event: event, MatchdayGrant: rt.matchdayRequests.grant(ctx.OrganizationID, event.ID, time.Now())})
	} else {
		_ = json.NewEncoder(w).Encode(event)
	}
	ctx.Logger.WithField("event_id", event.ID).Info("returned active event")
}
