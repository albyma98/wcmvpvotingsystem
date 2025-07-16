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

// postTicket generates a new voting ticket and stores it
func (rt *_router) postTicket(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	id, err := uuid.NewV4()
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot generate code")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	code := id.String()

	h := hmac.New(sha256.New, []byte(rt.VoteSecret))
	_, _ = h.Write([]byte(code))
	signature := hex.EncodeToString(h.Sum(nil))

	if err := rt.db.AddTicket(code, signature); err != nil {
		ctx.Logger.WithError(err).Error("cannot store ticket")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

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