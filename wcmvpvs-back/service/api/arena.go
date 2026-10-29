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
	meNickname := ""
	if token := rt.fanSessionTokenFromRequest(r); token != "" {
		if me, err := rt.db.GetFanBySessionToken(token, rt.deviceIDFromRequest(r)); err == nil && me.Profile.OrganizationID == ctx.OrganizationID && !me.Profile.RegistrationPending {
			meID = me.Profile.ID
			meNickname = me.Profile.Nickname
		}
	}
	for i := range fans {
		fans[i].IsMe = meID > 0 && fans[i].ID == meID
		if fans[i].IsMe {
			fans[i].Nickname = meNickname
		} else if !fans[i].Public {
			fans[i].Nickname = ""
		}
	}
	_ = writeJSON(w, http.StatusOK, map[string]interface{}{"organization_id": ctx.OrganizationID, "slug": ctx.OrganizationSlug, "fans": fans})
}
