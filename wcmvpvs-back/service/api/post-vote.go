package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

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
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.DeviceID) == "" {
		ctx.Logger.Warn("vote request missing device id")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx.Logger.Infof("vote received for player %d event %d", req.PlayerID, req.EventID)

	activeEvent, err := rt.db.GetActiveEvent()
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			ctx.Logger.Warn("vote attempted with no active event")
			w.WriteHeader(http.StatusConflict)
			return
		}
		ctx.Logger.WithError(err).Error("cannot retrieve active event")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if activeEvent.ID != req.EventID || !activeEvent.IsActive {
		ctx.Logger.Warn("vote attempted for inactive event")
		w.WriteHeader(http.StatusConflict)
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
			w.WriteHeader(http.StatusInternalServerError)
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
				w.WriteHeader(http.StatusConflict)
				return
			case isUniqueConstraintError(err):
				ctx.Logger.WithError(err).Error("vote unique constraint violation")
				w.WriteHeader(http.StatusInternalServerError)
				return
			default:
				ctx.Logger.WithError(err).Error("cannot store vote")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

		ctx.Logger.Infof("generated vote code %s", code)
		ctx.Logger.Info("vote stored in database")
		break
	}

	if code == "" {
		ctx.Logger.Error("unable to generate unique vote code after multiple attempts")
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
	ctx.Logger.WithFields(map[string]interface{}{
		"event_id":  req.EventID,
		"player_id": req.PlayerID,
		"code":      code,
	}).Info("vote response sent")
}
