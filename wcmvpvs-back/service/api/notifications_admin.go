package api

import (
	"encoding/json"
	"net/http"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
)

func (rt *_router) sendTestWinnerSMS(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var payload struct {
		Phone string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Payload non valido")
		return
	}
	phone, ok := normalizeE164(payload.Phone)
	if !ok {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Numero non valido")
		return
	}
	if _, err := rt.twilioMessaging.SendWhatsApp(phone, winnerExtractedSMSMessage); err != nil {
		ctx.Logger.WithError(err).WithField("phone", maskPhone(phone)).Warn("test whatsapp failed")
		_ = writeJSONMessage(w, twilioHTTPStatus(err), "Invio WhatsApp fallito")
		return
	}
	ctx.Logger.WithField("phone", maskPhone(phone)).Info("test whatsapp sent")
	_ = writeJSON(w, http.StatusOK, map[string]string{"message": "WhatsApp inviato"})
}
