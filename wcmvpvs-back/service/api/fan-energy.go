package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
)

const (
	fanEnergyCap         = 999
	fanEnergyTapCooldown = 2 * time.Second
)

type fanEnergyStatusResponse struct {
	Energy           int   `json:"energy"`
	Cap              int   `json:"cap"`
	BoostReadyAt     int64 `json:"boostReadyAt"`
	BoostActiveUntil int64 `json:"boostActiveUntil"`
	TapReadyAt       int64 `json:"tapReadyAt"`
	Tickets          int   `json:"tickets"`
	Now              int64 `json:"now"`
}

type fanEnergyClaimPayload struct {
	Amount int `json:"amount"`
}

type fanEnergyClaimResponse struct {
	Status  string `json:"status"`
	Awarded struct {
		Type string `json:"type"`
		Qty  int    `json:"qty"`
	} `json:"awarded"`
	fanEnergyStatusResponse
}

func (rt *_router) fanEnergyStatus(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	deviceID := rt.deviceIDFromRequest(r)
	if deviceID == "" {
		_ = writeJSONMessage(w, http.StatusBadRequest, "deviceId mancante")
		return
	}

	state, err := rt.db.GetFanEnergyState(deviceID)
	if err != nil {
		rt.writeFanEnergyError(w, err)
		return
	}

	_ = writeJSON(w, http.StatusOK, toFanEnergyResponse(state))
}

func (rt *_router) fanEnergyBoost(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	deviceID := rt.deviceIDFromRequest(r)
	if deviceID == "" {
		_ = writeJSONMessage(w, http.StatusBadRequest, "deviceId mancante")
		return
	}

	state, err := rt.db.ActivateFanEnergyBoost(deviceID)
	if err != nil {
		rt.writeFanEnergyError(w, err)
		return
	}

	_ = writeJSON(w, http.StatusOK, toFanEnergyResponse(state))
}

func (rt *_router) fanEnergyClaim(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	deviceID := rt.deviceIDFromRequest(r)
	if deviceID == "" {
		_ = writeJSONMessage(w, http.StatusBadRequest, "deviceId mancante")
		return
	}

	var payload fanEnergyClaimPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
		_ = writeJSONMessage(w, http.StatusBadRequest, "payload non valido")
		return
	}
	if payload.Amount <= 0 {
		payload.Amount = 100
	}

	state, err := rt.db.ClaimFanEnergy(deviceID, payload.Amount)
	if err != nil {
		rt.writeFanEnergyError(w, err)
		return
	}

	resp := fanEnergyClaimResponse{
		Status:                  "ok",
		fanEnergyStatusResponse: toFanEnergyResponse(state),
	}
	resp.Awarded.Type = "ticket"
	resp.Awarded.Qty = payload.Amount / 100

	_ = writeJSON(w, http.StatusOK, resp)
}

func (rt *_router) fanEnergyTap(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	deviceID := rt.deviceIDFromRequest(r)
	if deviceID == "" {
		_ = writeJSONMessage(w, http.StatusBadRequest, "deviceId mancante")
		return
	}

	state, err := rt.db.TapFanEnergy(deviceID)
	if err != nil {
		rt.writeFanEnergyError(w, err)
		return
	}

	_ = writeJSON(w, http.StatusOK, toFanEnergyResponse(state))
}

func (rt *_router) writeFanEnergyError(w http.ResponseWriter, err error) {
	switch {
	case err == nil:
		return
	case errors.Is(err, database.ErrFanEnergyBoostNotReady):
		_ = writeJSONMessage(w, http.StatusConflict, "boost non ancora pronto")
	case errors.Is(err, database.ErrFanEnergyInsufficientEnergy):
		_ = writeJSONMessage(w, http.StatusBadRequest, "energia insufficiente")
	case errors.Is(err, database.ErrFanEnergyDeviceRequired):
		_ = writeJSONMessage(w, http.StatusBadRequest, "deviceId mancante")
	case errors.Is(err, database.ErrFanEnergyTapCooldown):
		_ = writeJSONMessage(w, http.StatusTooManyRequests, "tap troppo rapido")
	default:
		_ = writeJSONMessage(w, http.StatusInternalServerError, "errore imprevisto")
	}
}

func toFanEnergyResponse(state database.FanEnergyState) fanEnergyStatusResponse {
	return fanEnergyStatusResponse{
		Energy:           state.Energy,
		Cap:              fanEnergyCap,
		BoostReadyAt:     state.BoostReadyAt.UTC().UnixMilli(),
		BoostActiveUntil: timeToMillis(state.BoostActiveUntil),
		TapReadyAt:       timeToMillis(state.LastTapAt.Add(fanEnergyTapCooldown)),
		Tickets:          state.Tickets,
		Now:              state.LastTs.UTC().UnixMilli(),
	}
}

func timeToMillis(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UTC().UnixMilli()
}
