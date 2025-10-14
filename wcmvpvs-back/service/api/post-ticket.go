package api

import (
	"encoding/json"
	"net/http"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

// postTicket generates a new voting ticket and stores it
func (rt *_router) postTicket(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	var (
		code      string
		signature string
	)

	for attempt := 0; attempt < maxCodeGenerationAttempts; attempt++ {
		var err error
		code, err = generateNumericCode()
		if err != nil {
			ctx.Logger.WithError(err).Error("cannot generate ticket code")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		signature = signCode(rt.VoteSecret, code)

		if err := rt.db.AddTicket(code, signature); err != nil {
			if isTicketCodeCollision(err) || isUniqueConstraintError(err) {
				ctx.Logger.WithError(err).Warn("duplicate ticket code detected, retrying")
				continue
			}
			ctx.Logger.WithError(err).Error("cannot store ticket")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		ctx.Logger.Infof("generated ticket %s", code)
		ctx.Logger.Info("ticket stored in database")
		break
	}

	if code == "" {
		ctx.Logger.Error("unable to generate unique ticket code after multiple attempts")
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
