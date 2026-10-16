package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
)

func fanShopError(w http.ResponseWriter, ctx reqcontext.RequestContext, err error) {
	switch {
	case errors.Is(err, database.ErrFanShopFunds):
		_ = writeJSONMessage(w, http.StatusConflict, "Monete insufficienti.")
	case errors.Is(err, database.ErrFanShopNotOwned):
		_ = writeJSONMessage(w, http.StatusConflict, "Acquista la cornice prima di attivarla.")
	case errors.Is(err, database.ErrFanShopItem):
		_ = writeJSONMessage(w, http.StatusBadRequest, "Articolo non disponibile.")
	case errors.Is(err, sql.ErrNoRows):
		_ = writeJSONMessage(w, http.StatusNotFound, "Evento o profilo non disponibile.")
	default:
		ctx.Logger.WithError(err).Error("fan wallet/store")
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Impossibile aggiornare il wallet.")
	}
}

func (rt *_router) getFanWallet(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, fanID, device, ok := rt.basketballIdentity(w, r, ctx)
	if !ok {
		return
	}
	state, err := rt.db.GetFanWalletState(eventID, ctx.OrganizationID, fanID, device)
	if err != nil {
		fanShopError(w, ctx, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	_ = writeJSON(w, http.StatusOK, state)
}

func (rt *_router) mutateFanShop(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext, equip bool) {
	eventID, fanID, device, ok := rt.basketballIdentity(w, r, ctx)
	if !ok {
		return
	}
	var payload struct {
		ItemKey *string `json:"item_key"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil || payload.ItemKey == nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Richiesta non valida.")
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Richiesta non valida.")
		return
	}
	var err error
	if equip {
		err = rt.db.EquipFanShopItem(eventID, ctx.OrganizationID, fanID, device, *payload.ItemKey)
	} else {
		err = rt.db.PurchaseFanShopItem(eventID, ctx.OrganizationID, fanID, device, *payload.ItemKey)
	}
	if err != nil {
		fanShopError(w, ctx, err)
		return
	}
	state, err := rt.db.GetFanWalletState(eventID, ctx.OrganizationID, fanID, device)
	if err != nil {
		fanShopError(w, ctx, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	_ = writeJSON(w, http.StatusOK, state)
}

func (rt *_router) purchaseFanShopItem(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	rt.mutateFanShop(w, r, ctx, false)
}
func (rt *_router) equipFanShopItem(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	rt.mutateFanShop(w, r, ctx, true)
}
