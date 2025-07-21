package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/gofrs/uuid"
	"github.com/julienschmidt/httprouter"
)

// postVote handles a vote submission
func (rt *_router) postVote(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	var req struct {
		PlayerID int    `json:"player_id"`
		EventID  int    `json:"event_id"`
		DeviceID string `json:"device_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctx.Logger.WithError(err).Error("cannot decode vote request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx.Logger.Infof("vote received for player %d event %d", req.PlayerID, req.EventID)
	id, err := uuid.NewV4()
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot generate code")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	code := id.String()
	ctx.Logger.Infof("generated vote code %s", code)
	h := hmac.New(sha256.New, []byte(rt.VoteSecret))
	h.Write([]byte(code))
	signature := hex.EncodeToString(h.Sum(nil))

	if err := rt.db.AddVote(req.EventID, req.PlayerID, code, signature, req.DeviceID); err != nil {
		ctx.Logger.WithError(err).Error("cannot store vote")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	ctx.Logger.Info("vote stored in database")
	qrDataBytes, _ := json.Marshal(struct {
		Code      string `json:"code"`
		Signature string `json:"signature"`
	}{Code: code, Signature: signature})

	resp := struct {
		Code      string `json:"code"`
		Signature string `json:"signature"`
		QRData    string `json:"qr_data"`
	}{Code: code, Signature: signature, QRData: string(qrDataBytes)}

	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
