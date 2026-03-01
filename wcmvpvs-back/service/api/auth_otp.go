package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/gofrs/uuid"
)

const (
	authPhoneLimit = 3
	authIPLimit    = 20
	authWindow     = 10 * time.Minute
)

type authStartPayload struct {
	Phone string `json:"phone"`
	Mode  string `json:"mode"`
}

type authVerifyPayload struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}

type authErrorResponse struct {
	Error string `json:"error"`
}

func (rt *_router) postAuthStart(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var payload authStartPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		_ = writeJSON(w, http.StatusBadRequest, authErrorResponse{Error: "INVALID_REQUEST"})
		return
	}

	phone, ok := normalizeE164(payload.Phone)
	if !ok {
		_ = writeJSON(w, http.StatusBadRequest, authErrorResponse{Error: "INVALID_PHONE"})
		return
	}

	if !rt.allowAuthRequest(phone, clientIPFromRequest(r)) {
		ctx.Logger.WithField("phone", maskPhone(phone)).Warn("OTP start rate limited")
		_ = writeJSON(w, http.StatusTooManyRequests, authErrorResponse{Error: "RATE_LIMITED"})
		return
	}

	mode := strings.ToLower(strings.TrimSpace(payload.Mode))
	found, err := rt.db.GetFanByPhoneE164(phone)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		ctx.Logger.WithError(err).Error("cannot lookup user by phone")
		_ = writeJSON(w, http.StatusInternalServerError, authErrorResponse{Error: "INTERNAL_ERROR"})
		return
	}
	exists := err == nil

	switch mode {
	case "register":
		if exists {
			_ = writeJSON(w, http.StatusConflict, authErrorResponse{Error: "PHONE_ALREADY_EXISTS"})
			return
		}
		if _, err := rt.db.CreateFanWithPhoneE164(phone); err != nil {
			ctx.Logger.WithError(err).WithField("phone", maskPhone(phone)).Error("cannot create pending user")
			_ = writeJSON(w, http.StatusInternalServerError, authErrorResponse{Error: "INTERNAL_ERROR"})
			return
		}
	case "login":
		if !exists {
			_ = writeJSON(w, http.StatusNotFound, authErrorResponse{Error: "USER_NOT_FOUND"})
			return
		}
		_ = found
	default:
		_ = writeJSON(w, http.StatusBadRequest, authErrorResponse{Error: "INVALID_MODE"})
		return
	}

	if err := rt.twilioVerify.StartSMSVerification(phone); err != nil {
		ctx.Logger.WithError(err).WithField("phone", maskPhone(phone)).Warn("cannot start OTP verification")
		_ = writeJSON(w, twilioHTTPStatus(err), authErrorResponse{Error: "OTP_SEND_FAILED"})
		return
	}

	ctx.Logger.WithFields(map[string]interface{}{"phone": maskPhone(phone), "mode": mode}).Info("OTP verification started")
	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) postAuthVerify(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var payload authVerifyPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		_ = writeJSON(w, http.StatusBadRequest, authErrorResponse{Error: "INVALID_REQUEST"})
		return
	}
	phone, ok := normalizeE164(payload.Phone)
	if !ok {
		_ = writeJSON(w, http.StatusBadRequest, authErrorResponse{Error: "INVALID_PHONE"})
		return
	}
	code := strings.TrimSpace(payload.Code)
	if code == "" {
		_ = writeJSON(w, http.StatusBadRequest, authErrorResponse{Error: "INVALID_CODE"})
		return
	}

	fan, err := rt.db.GetFanByPhoneE164(phone)
	if errors.Is(err, sql.ErrNoRows) {
		_ = writeJSON(w, http.StatusNotFound, authErrorResponse{Error: "USER_NOT_FOUND"})
		return
	}
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot lookup user by phone")
		_ = writeJSON(w, http.StatusInternalServerError, authErrorResponse{Error: "INTERNAL_ERROR"})
		return
	}

	approved, err := rt.twilioVerify.CheckSMSVerification(phone, code)
	if err != nil {
		ctx.Logger.WithError(err).WithField("phone", maskPhone(phone)).Warn("OTP verification check failed")
		_ = writeJSON(w, twilioHTTPStatus(err), authErrorResponse{Error: "OTP_CHECK_FAILED"})
		return
	}
	if !approved {
		_ = writeJSON(w, http.StatusUnauthorized, authErrorResponse{Error: "INVALID_OTP"})
		return
	}

	if err := rt.db.MarkFanPhoneVerified(phone, time.Now().UTC()); err != nil {
		ctx.Logger.WithError(err).Error("cannot mark phone verified")
		_ = writeJSON(w, http.StatusInternalServerError, authErrorResponse{Error: "INTERNAL_ERROR"})
		return
	}

	token := uuid.Must(uuid.NewV4()).String()
	if err := rt.db.UpsertFanSession(token, fan.ID, rt.deviceIDFromRequest(r)); err != nil {
		ctx.Logger.WithError(err).Error("cannot create fan session")
		_ = writeJSON(w, http.StatusInternalServerError, authErrorResponse{Error: "INTERNAL_ERROR"})
		return
	}

	ctx.Logger.WithField("phone", maskPhone(phone)).Info("OTP verification completed")
	_ = writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

func (rt *_router) postAuthResend(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var payload struct {
		Phone string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		_ = writeJSON(w, http.StatusBadRequest, authErrorResponse{Error: "INVALID_REQUEST"})
		return
	}
	phone, ok := normalizeE164(payload.Phone)
	if !ok {
		_ = writeJSON(w, http.StatusBadRequest, authErrorResponse{Error: "INVALID_PHONE"})
		return
	}
	if !rt.allowAuthRequest(phone, clientIPFromRequest(r)) {
		_ = writeJSON(w, http.StatusTooManyRequests, authErrorResponse{Error: "RATE_LIMITED"})
		return
	}
	if _, err := rt.db.GetFanByPhoneE164(phone); errors.Is(err, sql.ErrNoRows) {
		_ = writeJSON(w, http.StatusNotFound, authErrorResponse{Error: "USER_NOT_FOUND"})
		return
	} else if err != nil {
		ctx.Logger.WithError(err).Error("cannot lookup user by phone")
		_ = writeJSON(w, http.StatusInternalServerError, authErrorResponse{Error: "INTERNAL_ERROR"})
		return
	}
	if err := rt.twilioVerify.StartSMSVerification(phone); err != nil {
		ctx.Logger.WithError(err).WithField("phone", maskPhone(phone)).Warn("cannot resend OTP")
		_ = writeJSON(w, twilioHTTPStatus(err), authErrorResponse{Error: "OTP_SEND_FAILED"})
		return
	}
	ctx.Logger.WithField("phone", maskPhone(phone)).Info("OTP verification resent")
	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) allowAuthRequest(phone string, ip string) bool {
	now := time.Now()
	rt.authRateMu.Lock()
	defer rt.authRateMu.Unlock()
	if !allowRate(rt.authRateByPhone, phone, authPhoneLimit, authWindow, now) {
		return false
	}
	if ip != "" && !allowRate(rt.authRateByIP, ip, authIPLimit, authWindow, now) {
		return false
	}
	return true
}

func clientIPFromRequest(r *http.Request) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func maskPhone(phone string) string {
	if len(phone) <= 6 {
		return "******"
	}
	return phone[:3] + strings.Repeat("*", len(phone)-7) + phone[len(phone)-4:]
}
