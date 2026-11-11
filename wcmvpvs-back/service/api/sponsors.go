package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
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

func (rt *_router) getGameWallSponsors(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	assignments, err := rt.db.GetGameWallSponsors(r.Context(), ctx.OrganizationID, false)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot load game wall sponsors")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, assignments)
}

func (rt *_router) saveGameWallSponsors(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var assignments map[string]int
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048))
	if err := decoder.Decode(&assignments); err != nil || assignments == nil || decoder.Decode(&struct{}{}) != io.EOF {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Assegnazioni non valide."})
		return
	}
	if err := rt.db.SaveGameWallSponsors(r.Context(), ctx.OrganizationID, assignments); err != nil {
		if errors.Is(err, database.ErrInvalidGameWallSponsor) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Una parete o uno sponsor non è disponibile. Aggiorna i dati e riprova."})
		} else {
			ctx.Logger.WithError(err).Error("cannot save game wall sponsors")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
