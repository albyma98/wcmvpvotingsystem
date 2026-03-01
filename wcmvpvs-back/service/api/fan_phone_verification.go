package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/gofrs/uuid"
)

type startPhoneVerificationPayload struct {
	Phone string `json:"phone"`
}

type checkPhoneVerificationPayload struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}

type twilioVerificationResponse struct {
	Status string `json:"status"`
}

func normalizeFanPhone(value string) string {
	cleaned := strings.Map(func(r rune) rune {
		if r == '+' || (r >= '0' && r <= '9') {
			return r
		}
		return -1
	}, strings.TrimSpace(value))
	if cleaned == "" {
		return ""
	}
	if strings.HasPrefix(cleaned, "+") {
		return cleaned
	}
	if strings.HasPrefix(cleaned, "00") {
		return "+" + strings.TrimPrefix(cleaned, "00")
	}
	if strings.HasPrefix(cleaned, "39") && len(cleaned) >= 10 {
		return "+" + cleaned
	}
	return "+39" + cleaned
}

func (rt *_router) twilioVerifyConfigured() bool {
	return strings.TrimSpace(rt.twilioAccountSID) != "" && strings.TrimSpace(rt.twilioAuthToken) != "" && strings.TrimSpace(rt.twilioVerifySID) != ""
}

func (rt *_router) callTwilioVerify(path string, values url.Values) (twilioVerificationResponse, error) {
	if !rt.twilioVerifyConfigured() {
		return twilioVerificationResponse{}, fmt.Errorf("twilio verify not configured")
	}
	endpoint := fmt.Sprintf("https://verify.twilio.com/v2/Services/%s/%s", rt.twilioVerifySID, path)
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBufferString(values.Encode()))
	if err != nil {
		return twilioVerificationResponse{}, err
	}
	req.SetBasicAuth(rt.twilioAccountSID, rt.twilioAuthToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return twilioVerificationResponse{}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return twilioVerificationResponse{}, fmt.Errorf("twilio verify error status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed twilioVerificationResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return twilioVerificationResponse{}, err
	}
	return parsed, nil
}

func (rt *_router) postStartFanPhoneVerification(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !rt.twilioVerifyConfigured() {
		_ = writeJSONMessage(w, http.StatusServiceUnavailable, "Verifica telefono non disponibile")
		return
	}
	var payload startPhoneVerificationPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Richiesta non valida")
		return
	}
	phone := normalizeFanPhone(payload.Phone)
	if !phoneRegex.MatchString(phone) {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Numero di telefono non valido")
		return
	}
	channel := strings.TrimSpace(rt.twilioVerifyChannel)
	if channel == "" {
		channel = "sms"
	}
	_, err := rt.callTwilioVerify("Verifications", url.Values{"To": {phone}, "Channel": {channel}})
	if err != nil {
		ctx.Logger.WithError(err).Error("twilio start verification failed")
		_ = writeJSONMessage(w, http.StatusBadGateway, "Invio OTP non riuscito")
		return
	}
	_ = writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

func (rt *_router) postCheckFanPhoneVerification(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !rt.twilioVerifyConfigured() {
		_ = writeJSONMessage(w, http.StatusServiceUnavailable, "Verifica telefono non disponibile")
		return
	}
	var payload checkPhoneVerificationPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Richiesta non valida")
		return
	}
	phone := normalizeFanPhone(payload.Phone)
	code := strings.TrimSpace(payload.Code)
	if !phoneRegex.MatchString(phone) || code == "" {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Numero o codice OTP non validi")
		return
	}
	resp, err := rt.callTwilioVerify("VerificationCheck", url.Values{"To": {phone}, "Code": {code}})
	if err != nil {
		ctx.Logger.WithError(err).Error("twilio check verification failed")
		_ = writeJSONMessage(w, http.StatusBadGateway, "Verifica OTP non riuscita")
		return
	}
	if strings.ToLower(strings.TrimSpace(resp.Status)) != "approved" {
		_ = writeJSONMessage(w, http.StatusUnauthorized, "Codice OTP non valido")
		return
	}

	summary, err := rt.db.GetFanByPhone(ctx.OrganizationID, phone)
	if err == nil {
		sessionToken := uuid.Must(uuid.NewV4()).String()
		if err := rt.db.UpsertFanSession(sessionToken, summary.Profile.ID, rt.deviceIDFromRequest(r)); err != nil {
			ctx.Logger.WithError(err).Error("cannot upsert fan session after otp login")
			_ = writeJSONMessage(w, http.StatusInternalServerError, "Impossibile aprire sessione")
			return
		}
		_ = writeJSON(w, http.StatusOK, map[string]interface{}{
			"verified":      true,
			"registered":    true,
			"session_token": sessionToken,
			"user":          summary.Profile,
			"wallet":        summary.Wallet,
		})
		return
	}
	if err != nil && err != sql.ErrNoRows {
		ctx.Logger.WithError(err).Error("cannot lookup fan by phone")
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Errore verifica telefono")
		return
	}

	verificationToken := uuid.Must(uuid.NewV4()).String()
	if err := rt.db.SavePhoneVerificationToken(ctx.OrganizationID, phone, verificationToken); err != nil {
		ctx.Logger.WithError(err).Error("cannot persist phone verification token")
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Errore verifica telefono")
		return
	}

	_ = writeJSON(w, http.StatusOK, map[string]interface{}{
		"verified":           true,
		"registered":         false,
		"verification_token": verificationToken,
	})
}
