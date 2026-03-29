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
	if ctx.OrganizationID == 0 {
		ctx.Logger.Warn("missing organization while casting vote")
		_ = writeJSONMessage(w, http.StatusBadRequest, "Organizzazione non valida.")
		return
	}

	ctx.Logger.Infof("vote received for player %d event %d", req.PlayerID, req.EventID)

	activeEvent, err := rt.db.GetActiveEvent(ctx.OrganizationID)
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
	if !rt.ensureEventInOrganization(w, ctx, req.EventID) {
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
		fanID     *int
	)

	if token := rt.fanSessionTokenFromRequest(r); token != "" {
		if me, fanErr := rt.db.GetFanBySessionToken(token, rt.deviceIDFromRequest(r)); fanErr == nil {
			fanID = &me.Profile.ID
		} else if !errors.Is(fanErr, sql.ErrNoRows) {
			ctx.Logger.WithError(fanErr).Warn("cannot resolve fan session while casting vote")
		}
	}

	existingVote, existingVoteErr := rt.db.GetDeviceVote(req.EventID, req.DeviceID)
	if existingVoteErr == nil {
		code = strings.TrimSpace(existingVote.TicketCode)
		signature = strings.TrimSpace(existingVote.TicketSignature)
	}
	if existingVoteErr != nil && !errors.Is(existingVoteErr, sql.ErrNoRows) {
		ctx.Logger.WithError(existingVoteErr).Error("cannot load existing vote before update")
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Servizio non disponibile. Riprova tra pochi istanti.")
		return
	}

	if code == "" || signature == "" {
		for attempt := 0; attempt < maxCodeGenerationAttempts; attempt++ {
			var err error
			code, err = generateNumericCode()
			if err != nil {
				ctx.Logger.WithError(err).Error("cannot generate vote code")
				_ = writeJSONMessage(w, http.StatusInternalServerError, "Servizio non disponibile. Riprova tra pochi istanti.")
				return
			}
			signature = signCode(rt.VoteSecret, code)
			if err := rt.db.AddVote(req.EventID, req.PlayerID, code, signature, req.DeviceID, fanID); err != nil {
				switch {
				case isVoteCodeCollision(err):
					ctx.Logger.WithError(err).Warn("duplicate vote code detected, retrying")
					continue
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
	} else {
		if err := rt.db.AddVote(req.EventID, req.PlayerID, code, signature, req.DeviceID, fanID); err != nil {
			ctx.Logger.WithError(err).Error("cannot update vote while reusing existing ticket code")
			_ = writeJSONMessage(w, http.StatusInternalServerError, "Servizio non disponibile. Riprova tra pochi istanti.")
			return
		}
		ctx.Logger.WithFields(map[string]interface{}{
			"event_id": req.EventID,
			"code":     code,
		}).Info("reused existing vote code for device")
	}

	if code == "" {
		ctx.Logger.Error("unable to generate unique vote code after multiple attempts")
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Servizio non disponibile. Riprova tra pochi istanti.")
		return
	}

	rt.votesHub.Broadcast(req.EventID)
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
