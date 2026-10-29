package api

import (
	"encoding/json"
	"errors"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"io"
	"net/http"
	"time"
)

func (rt *_router) arenaActivities(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if ctx.OrganizationID <= 0 {
		w.WriteHeader(400)
		return
	}
	fan := 0
	if token := rt.fanSessionTokenFromRequest(r); token != "" {
		me, err := rt.db.GetFanBySessionToken(token, rt.deviceIDFromRequest(r))
		if err != nil || me.Profile.OrganizationID != ctx.OrganizationID || me.Profile.RegistrationPending {
			w.WriteHeader(403)
			return
		}
		fan = me.Profile.ID
	}
	cmd := database.ArenaCommand{Action: "state"}
	if r.Method == http.MethodPost {
		d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
		d.DisallowUnknownFields()
		if err := d.Decode(&cmd); err != nil {
			w.WriteHeader(400)
			return
		}
		if d.Decode(&struct{}{}) != io.EOF {
			w.WriteHeader(400)
			return
		}
		admin := cmd.Action == "config" || cmd.Action == "result"
		if admin && ctx.AdminID <= 0 {
			w.WriteHeader(403)
			return
		}
		if !admin && fan <= 0 {
			w.WriteHeader(401)
			return
		}
	}
	state, err := rt.db.ArenaActivities(ctx.OrganizationID, fan, cmd, time.Now())
	if err != nil {
		status := 500
		message := "Impossibile aggiornare l’attività. Riprova."
		if errors.Is(err, database.ErrArena) {
			status = 409
			message = "Attività chiusa, invito non disponibile o richiesta non valida."
		}
		if errors.Is(err, database.ErrFanShopFunds) {
			status = 409
			message = "Monetine insufficienti."
		}
		if errors.Is(err, database.ErrFanShopNotOwned) {
			status = 409
			message = "Acquista l’accessorio prima di equipaggiarlo."
		}
		ctx.Logger.WithError(err).Warn("arena activity")
		_ = writeJSONMessage(w, status, message)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	_ = writeJSON(w, 200, state)
}
