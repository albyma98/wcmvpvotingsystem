package api

import (
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"net/http"
)

func (rt *_router) getArenaFans(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if ctx.OrganizationID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	fans, err := rt.db.GetArenaFans(ctx.OrganizationID)
	if err != nil {
		ctx.Logger.WithError(err).Error("arena fans")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	meID := 0
	if token := rt.fanSessionTokenFromRequest(r); token != "" {
		if me, err := rt.db.GetFanBySessionToken(token, rt.deviceIDFromRequest(r)); err == nil && me.Profile.OrganizationID == ctx.OrganizationID && !me.Profile.RegistrationPending {
			meID = me.Profile.ID
		}
	}
	for i := range fans {
		fans[i].IsMe = meID > 0 && fans[i].ID == meID
	}
	_ = writeJSON(w, http.StatusOK, map[string]interface{}{"organization_id": ctx.OrganizationID, "slug": ctx.OrganizationSlug, "fans": fans})
}
