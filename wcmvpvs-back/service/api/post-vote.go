package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
)

// postVote handles a vote submission
func (rt *_router) postVote(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var req struct {
		PlayerID int    `json:"player_id"`
		EventID  int    `json:"event_id"`
		DeviceID string `json:"device_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctx.Logger.WithError(err).Error("cannot decode vote request")
		_ = writeJSONMessage(w, http.StatusBadRequest, "Richiesta di voto non valida.")
		return
	}

	req.DeviceID = strings.TrimSpace(req.DeviceID)
	if req.DeviceID == "" {
		ctx.Logger.Warn("vote request missing device id")
		_ = writeJSONMessage(w, http.StatusBadRequest, "Impossibile registrare il voto senza un identificativo dispositivo valido.")
		return
	}

	ctx.Logger.Infof("vote received for player %d event %d", req.PlayerID, req.EventID)

	activeEvent, err := rt.db.GetActiveEvent()
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			ctx.Logger.Warn("vote attempted with no active event")
			_ = writeJSONMessage(w, http.StatusConflict, "Nessuna votazione attiva al momento.")
			return
		}
		ctx.Logger.WithError(err).Error("cannot retrieve active event")
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Servizio non disponibile. Riprova tra pochi istanti.")
		return
	}

	if activeEvent.ID != req.EventID || !activeEvent.IsActive {
		ctx.Logger.Warn("vote attempted for inactive event")
		_ = writeJSONMessage(w, http.StatusConflict, "Le votazioni per questa partita non sono disponibili.")
		return
	}

	if activeEvent.VotesClosed {
		ctx.Logger.Warn("vote attempted but voting is closed")
		_ = writeJSONMessage(w, http.StatusConflict, "Le votazioni per questa partita sono chiuse.")
		return
	}

	clientIP := rt.getClientIP(r)
	if limited, message := rt.shouldThrottleVoteAttempt(req.DeviceID, clientIP, time.Now()); limited {
		ctx.Logger.WithFields(map[string]interface{}{
			"device_id": req.DeviceID,
			"client_ip": clientIP,
		}).Warn("vote attempt throttled")
		_ = writeJSONMessage(w, http.StatusTooManyRequests, message)
		return
	}

	var (
		code      string
		signature string
	)

	for attempt := 0; attempt < maxCodeGenerationAttempts; attempt++ {
		var err error
		code, err = generateNumericCode()
		if err != nil {
			ctx.Logger.WithError(err).Error("cannot generate vote code")
			_ = writeJSONMessage(w, http.StatusInternalServerError, "Servizio non disponibile. Riprova tra pochi istanti.")
			return
		}
		signature = signCode(rt.VoteSecret, code)

		if err := rt.db.AddVote(req.EventID, req.PlayerID, code, signature, req.DeviceID); err != nil {
			switch {
			case isVoteCodeCollision(err):
				ctx.Logger.WithError(err).Warn("duplicate vote code detected, retrying")
				continue
			case isVoteDeviceCollision(err):
				ctx.Logger.WithError(err).Warn("duplicate vote attempt for device")
				_ = writeJSONMessage(w, http.StatusConflict, "Hai già votato per questa partita.")
				return
			case isUniqueConstraintError(err):
				ctx.Logger.WithError(err).Error("vote unique constraint violation")
				_ = writeJSONMessage(w, http.StatusInternalServerError, "Servizio non disponibile. Riprova tra pochi istanti.")
				return
			default:
				ctx.Logger.WithError(err).Error("cannot store vote")
				_ = writeJSONMessage(w, http.StatusInternalServerError, "Servizio non disponibile. Riprova tra pochi istanti.")
				return
			}
		}

		ctx.Logger.Infof("generated vote code %s", code)
		ctx.Logger.Info("vote stored in database")
		break
	}

	if code == "" {
		ctx.Logger.Error("unable to generate unique vote code after multiple attempts")
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Servizio non disponibile. Riprova tra pochi istanti.")
		return
	}
	validationURL, err := rt.buildTicketValidationURL(req.EventID, code, signature)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot build ticket validation URL")
	}

	resp := struct {
		Code      string `json:"code"`
		Signature string `json:"signature"`
		QRData    string `json:"qr_data"`
		Message   string `json:"message"`
	}{Code: code, Signature: signature, QRData: validationURL, Message: "Voto registrato con successo."}

	if err := writeJSON(w, http.StatusOK, resp); err != nil {
		ctx.Logger.WithError(err).Error("cannot encode vote response")
	}
	ctx.Logger.WithFields(map[string]interface{}{
		"event_id":  req.EventID,
		"player_id": req.PlayerID,
		"code":      code,
	}).Info("vote response sent")
}
