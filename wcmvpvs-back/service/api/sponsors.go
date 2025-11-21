package api

import (
	"encoding/json"
	"net/http"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
)

func (rt *_router) listPublicSponsors(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if ctx.OrganizationID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sponsors, err := rt.db.ListActiveSponsors(ctx.OrganizationID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list active sponsors")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(sponsors)
	ctx.Logger.WithField("sponsors", len(sponsors)).Info("listed active sponsors")
}
