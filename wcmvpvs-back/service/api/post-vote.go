package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
)

// postVote handles a vote submission
const fingerprintRetentionWindow = 48 * time.Hour

func (rt *_router) postVote(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var req struct {
		PlayerID    int                      `json:"player_id"`
		EventID     int                      `json:"event_id"`
		DeviceToken string                   `json:"device_token"`
		Fingerprint deviceFingerprintPayload `json:"fingerprint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctx.Logger.WithError(err).Error("cannot decode vote request")
		_ = writeJSONMessage(w, http.StatusBadRequest, "Richiesta di voto non valida.")
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

	now := time.Now()

	if err := req.Fingerprint.validate(); err != nil {
		ctx.Logger.WithError(err).Warn("invalid fingerprint payload")
		_ = writeJSONMessage(w, http.StatusBadRequest, "Impossibile verificare il dispositivo. Ricarica la pagina e riprova.")
		return
	}

	fingerprintHash := generateDailyFingerprintHash(req.EventID, req.Fingerprint, now)

	clientIP := rt.getClientIP(r)

	req.DeviceToken = strings.TrimSpace(req.DeviceToken)
	deviceKey := req.DeviceToken
	if deviceKey == "" {
		deviceKey = fingerprintHash
	}

	if limited, message := rt.shouldThrottleVoteAttempt(deviceKey, fingerprintHash, clientIP, now); limited {
		ctx.Logger.WithFields(map[string]interface{}{
			"device_token": deviceKey,
			"fingerprint":  fingerprintHash,
			"client_ip":    clientIP,
		}).Warn("vote attempt throttled")
		_ = writeJSONMessage(w, http.StatusTooManyRequests, message)
		return
	}

	if err := rt.db.PruneExpiredFingerprintLocks(now); err != nil {
		ctx.Logger.WithError(err).Warn("cannot prune fingerprint locks")
	}

	if err := rt.db.ClearExpiredVoteFingerprints(now.Add(-fingerprintRetentionWindow)); err != nil {
		ctx.Logger.WithError(err).Warn("cannot clear expired vote fingerprints")
	}

	if err := rt.db.LockFingerprint(req.EventID, fingerprintHash, now.Add(fingerprintRetentionWindow)); err != nil {
		if errors.Is(err, database.ErrFingerprintAlreadyUsed) {
			ctx.Logger.WithFields(map[string]interface{}{
				"event_id":     req.EventID,
				"fingerprint":  fingerprintHash,
				"device_token": deviceKey,
			}).Warn("duplicate fingerprint vote detected")
			_ = writeJSONMessage(w, http.StatusConflict, "Hai già espresso un voto per questa partita oggi.")
			return
		}
		ctx.Logger.WithError(err).Error("cannot lock fingerprint for vote")
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Servizio non disponibile. Riprova tra pochi istanti.")
		return
	}

	unlockOnFailure := true
	defer func() {
		if unlockOnFailure {
			if err := rt.db.UnlockFingerprint(req.EventID, fingerprintHash); err != nil {
				ctx.Logger.WithError(err).Warn("cannot release fingerprint lock")
			}
		}
	}()

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

		if err := rt.db.AddVote(req.EventID, req.PlayerID, code, signature, fingerprintHash); err != nil {
			switch {
			case isVoteCodeCollision(err):
				ctx.Logger.WithError(err).Warn("duplicate vote code detected, retrying")
				continue
			case isVoteFingerprintCollision(err):
				ctx.Logger.WithError(err).Warn("duplicate vote attempt for fingerprint")
				unlockOnFailure = false
				_ = writeJSONMessage(w, http.StatusConflict, "Hai già espresso un voto per questa partita oggi.")
				return
			case isUniqueConstraintError(err):
				ctx.Logger.WithError(err).Error("vote unique constraint violation")
				unlockOnFailure = false
				if err := rt.db.UnlockFingerprint(req.EventID, fingerprintHash); err != nil {
					ctx.Logger.WithError(err).Warn("cannot unlock fingerprint after constraint error")
				}
				_ = writeJSONMessage(w, http.StatusInternalServerError, "Servizio non disponibile. Riprova tra pochi istanti.")
				return
			default:
				ctx.Logger.WithError(err).Error("cannot store vote")
				unlockOnFailure = false
				if err := rt.db.UnlockFingerprint(req.EventID, fingerprintHash); err != nil {
					ctx.Logger.WithError(err).Warn("cannot unlock fingerprint after store failure")
				}
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
		unlockOnFailure = false
		if err := rt.db.UnlockFingerprint(req.EventID, fingerprintHash); err != nil {
			ctx.Logger.WithError(err).Warn("cannot unlock fingerprint after code exhaustion")
		}
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Servizio non disponibile. Riprova tra pochi istanti.")
		return
	}
	validationURL, err := rt.buildTicketValidationURL(req.EventID, code, signature)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot build ticket validation URL")
	}

	unlockOnFailure = false

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
