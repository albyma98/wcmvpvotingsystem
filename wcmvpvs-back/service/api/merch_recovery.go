package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
)

var (
	merchVerifyServiceID  = regexp.MustCompile(`^VA[0-9a-fA-F]{32}$`)
	merchVerifyChallenge  = regexp.MustCompile(`^VE[0-9a-fA-F]{32}$`)
	merchVerificationCode = regexp.MustCompile(`^[0-9]{4,10}$`)
)

func newMerchVerifyClient(cfg Config) *twilioVerifyClient {
	serviceID := strings.TrimSpace(cfg.MerchTwilioVerifySID)
	if !merchVerifyServiceID.MatchString(serviceID) || strings.EqualFold(serviceID, strings.TrimSpace(cfg.TwilioVerifySID)) {
		return nil
	}
	return newTwilioVerifyClient(twilioVerifyConfig{
		AccountSID: cfg.TwilioAccountSID, AuthToken: cfg.TwilioAuthToken, ServiceSID: serviceID,
	})
}

func registerMerchRecoveryRoutes(rt *_router, prefix string) {
	rt.router.Post(prefix+"/orders/{id}/recovery/start", rt.wrapMerchPublic(rt.startMerchRecovery))
	rt.router.Post(prefix+"/orders/{id}/recovery/verify", rt.wrapMerchPublic(rt.verifyMerchRecovery))
}

func (rt *_router) merchRecoveryAvailable(ctx context.Context, order merchOrder) bool {
	if order.FanID != 0 || order.Mode != "offline" || order.Phone == "" ||
		rt.merchVerify == nil || !rt.merchVerify.enabled() ||
		rt.twilioMessaging == nil || !rt.twilioMessaging.enabled() ||
		rt.merchStripe == nil || rt.merchStripe.base == nil || rt.merchStripe.base.Scheme != "https" {
		return false
	}
	var enabled bool
	err := rt.db.SQLConn().QueryRowContext(ctx, `SELECT COALESCE(json_extract(config,'$.sms_notifications')=1,0)
		AND NOT EXISTS (SELECT 1 FROM merch_recovery_verifications WHERE organization_id=? AND order_id=? AND verified_at>0)
		FROM merch_settings WHERE organization_id=?`, order.OrganizationID, order.ID, order.OrganizationID).Scan(&enabled)
	return err == nil && enabled
}

func (rt *_router) merchRecoveryOrder(r *http.Request, org int) (merchOrder, error) {
	o, err := scanMerchOrder(rt.db.SQLConn().QueryRowContext(r.Context(), "SELECT "+merchOrderColumns+
		" FROM merch_orders WHERE organization_id=? AND id=?", org, chi.URLParam(r, "id")))
	if err != nil {
		return o, err
	}
	if !o.acceptsGuestToken(r.Header.Get("X-Merch-Access")) || o.FanID != 0 || o.Mode != "offline" {
		return o, sql.ErrNoRows
	}
	return o, nil
}

func (rt *_router) merchRecoveryRate(r *http.Request, order merchOrder, action string, limit int) error {
	var allowed bool
	err := rt.merchWrite(r.Context(), func(tx *sql.Tx) error {
		ipAllowed, err := merchRate(r.Context(), tx, order.OrganizationID, action+"-ip", rt.getClientIP(r), time.Hour, 30)
		if err != nil || !ipAllowed {
			return err
		}
		allowed, err = merchRate(r.Context(), tx, order.OrganizationID, action+"-phone", order.Phone, time.Hour, limit)
		return err
	})
	if err == nil && !allowed {
		return merchError{http.StatusTooManyRequests, "Troppe richieste di recupero. Riprova più tardi."}
	}
	return err
}

func (rt *_router) startMerchRecovery(w http.ResponseWriter, r *http.Request, org database.Organization) {
	w.Header().Set("Cache-Control", "no-store")
	order, err := rt.merchRecoveryOrder(r, org.ID)
	if err != nil {
		rt.writeMerchError(w, err)
		return
	}
	if !rt.merchRecoveryAvailable(r.Context(), order) {
		rt.writeMerchError(w, merchConflict("Recupero via SMS non disponibile per questa prenotazione"))
		return
	}
	if err := rt.merchRecoveryRate(r, order, "recovery-start", 3); err != nil {
		rt.writeMerchError(w, err)
		return
	}
	result, err := rt.merchVerify.postForm("Verifications", url.Values{"To": {order.Phone}, "Channel": {"sms"}})
	var challenge struct{ SID string }
	if err != nil || json.Unmarshal(result, &challenge) != nil || !merchVerifyChallenge.MatchString(challenge.SID) {
		_ = writeJSONMessage(w, http.StatusBadGateway, "Invio del codice SMS non riuscito")
		return
	}
	_, err = rt.db.SQLConn().ExecContext(r.Context(), `INSERT INTO merch_recovery_verifications
		(organization_id,order_id,challenge_sid,expires_at) VALUES (?,?,?,?)
		ON CONFLICT(organization_id,order_id) DO UPDATE SET challenge_sid=excluded.challenge_sid,expires_at=excluded.expires_at
		WHERE verified_at=0`, org.ID, order.ID, challenge.SID, time.Now().Add(10*time.Minute).Unix())
	if err != nil {
		rt.writeMerchError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) verifyMerchRecovery(w http.ResponseWriter, r *http.Request, org database.Organization) {
	w.Header().Set("Cache-Control", "no-store")
	order, err := rt.merchRecoveryOrder(r, org.ID)
	if err != nil {
		rt.writeMerchError(w, err)
		return
	}
	var input struct{ Code string }
	r.Body = http.MaxBytesReader(w, r.Body, 256)
	if json.NewDecoder(r.Body).Decode(&input) != nil || !merchVerificationCode.MatchString(input.Code) {
		rt.writeMerchError(w, merchInvalid("Inserisci il codice ricevuto via SMS"))
		return
	}
	var challenge string
	var expiresAt, verifiedAt int64
	err = rt.db.SQLConn().QueryRowContext(r.Context(), `SELECT challenge_sid,expires_at,verified_at
		FROM merch_recovery_verifications WHERE organization_id=? AND order_id=?`, org.ID, order.ID).
		Scan(&challenge, &expiresAt, &verifiedAt)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		rt.writeMerchError(w, err)
		return
	}
	if verifiedAt != 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if challenge == "" || time.Now().Unix() >= expiresAt || !rt.merchRecoveryAvailable(r.Context(), order) {
		rt.writeMerchError(w, merchConflict("Richiedi un nuovo codice di verifica"))
		return
	}
	if err := rt.merchRecoveryRate(r, order, "recovery-verify", 8); err != nil {
		rt.writeMerchError(w, err)
		return
	}
	result, err := rt.merchVerify.postForm("VerificationCheck", url.Values{"VerificationSid": {challenge}, "Code": {input.Code}})
	var checked struct{ Status string }
	if err != nil || json.Unmarshal(result, &checked) != nil || checked.Status != "approved" {
		rt.writeMerchError(w, merchInvalid("Codice non valido o scaduto"))
		return
	}
	// Keep the approved verification if the browser disconnects before receiving its response.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = rt.merchWrite(ctx, func(tx *sql.Tx) error {
		now := time.Now().Unix()
		result, err := tx.ExecContext(ctx, `UPDATE merch_recovery_verifications SET verified_at=?
			WHERE organization_id=? AND order_id=? AND challenge_sid=? AND verified_at=0`, now, org.ID, order.ID, challenge)
		if err != nil {
			return err
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rows == 0 {
			var verifiedAt int64
			if err := tx.QueryRowContext(ctx, `SELECT verified_at FROM merch_recovery_verifications
				WHERE organization_id=? AND order_id=?`, org.ID, order.ID).Scan(&verifiedAt); err != nil {
				return err
			}
			if verifiedAt == 0 {
				return merchConflict("Usa il codice di verifica più recente")
			}
			return nil
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO merch_notifications(organization_id,order_id,event_type,available_at,updated_at)
			VALUES (?,?,'RECOVERY_LINK',?,?) ON CONFLICT(organization_id,order_id,event_type) DO NOTHING`, org.ID, order.ID, now, now)
		return err
	})
	if err != nil {
		rt.writeMerchError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
