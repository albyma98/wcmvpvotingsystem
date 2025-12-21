package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
)

type couponPayload struct {
	Title          string `json:"title"`
	ShortDesc      string `json:"short_desc"`
	SponsorID      int    `json:"sponsor_id"`
	MerchantID     int    `json:"merchant_id"`
	MatchIDs       []int  `json:"match_ids"`
	StartDate      string `json:"start_date"`
	EndDate        string `json:"end_date"`
	MaxUses        int    `json:"max_uses"`
	Status         string `json:"status"`
	ImageURL       string `json:"image_url"`
	Highlight      bool   `json:"highlight"`
	RedemptionType string `json:"redemption_type"`
	ManualCode     string `json:"manual_code"`
	RedeemURL      string `json:"redeem_url"`
}

type couponClaimRequest struct {
	MatchID int `json:"match_id"`
	UserID  int `json:"user_id"`
}

type couponValidationRequest struct {
	Code       string `json:"code"`
	MerchantID int    `json:"merchant_id"`
	Signature  string `json:"signature"`
}

func (rt *_router) listAdminCoupons(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	coupons, err := rt.db.ListCoupons(ctx.OrganizationID)
	if err != nil {
		ctx.Logger.WithError(err).Error("list coupons")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, coupons)
}

func (rt *_router) createAdminCoupon(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var payload couponPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if payload.MerchantID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	redemptionType := normalizeCouponRedemptionType(payload.RedemptionType)
	if redemptionType == "code" {
		if strings.TrimSpace(payload.ManualCode) == "" || strings.TrimSpace(payload.RedeemURL) == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	coupon := database.Coupon{
		Title:          payload.Title,
		ShortDesc:      payload.ShortDesc,
		SponsorID:      payload.SponsorID,
		MerchantID:     payload.MerchantID,
		MatchIDs:       payload.MatchIDs,
		StartDate:      payload.StartDate,
		EndDate:        payload.EndDate,
		MaxUses:        payload.MaxUses,
		Status:         defaultCouponStatus(payload.Status),
		ImageURL:       payload.ImageURL,
		Highlight:      payload.Highlight,
		Segmentation:   "all",
		OrganizationID: ctx.OrganizationID,
		RedemptionType: redemptionType,
		ManualCode:     payload.ManualCode,
		RedeemURL:      payload.RedeemURL,
	}

	stored, err := rt.db.CreateCoupon(coupon)
	if err != nil {
		ctx.Logger.WithError(err).Error("create coupon")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, stored)
}

func (rt *_router) updateAdminCoupon(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var payload couponPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if payload.MerchantID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	redemptionType := normalizeCouponRedemptionType(payload.RedemptionType)
	if redemptionType == "code" {
		if strings.TrimSpace(payload.ManualCode) == "" || strings.TrimSpace(payload.RedeemURL) == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	coupon := database.Coupon{
		ID:             id,
		Title:          payload.Title,
		ShortDesc:      payload.ShortDesc,
		SponsorID:      payload.SponsorID,
		MerchantID:     payload.MerchantID,
		MatchIDs:       payload.MatchIDs,
		StartDate:      payload.StartDate,
		EndDate:        payload.EndDate,
		MaxUses:        payload.MaxUses,
		Status:         defaultCouponStatus(payload.Status),
		ImageURL:       payload.ImageURL,
		Highlight:      payload.Highlight,
		Segmentation:   "all",
		OrganizationID: ctx.OrganizationID,
		RedemptionType: redemptionType,
		ManualCode:     payload.ManualCode,
		RedeemURL:      payload.RedeemURL,
	}

	updated, err := rt.db.UpdateCoupon(coupon)
	if err != nil {
		ctx.Logger.WithError(err).Error("update coupon")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (rt *_router) deleteAdminCoupon(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	if err := rt.db.DeleteCoupon(id, ctx.OrganizationID); err != nil {
		ctx.Logger.WithError(err).Warn("delete coupon")
		if err == database.ErrCouponUnavailable || err == database.ErrCouponMaxReached || err == database.ErrInvalidSponsorData {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) listEventCoupons(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, _ := strconv.Atoi(chi.URLParam(r, "eventId"))
	coupons, err := rt.db.ListCoupons(ctx.OrganizationID)
	if err != nil {
		ctx.Logger.WithError(err).Error("list coupons")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	filtered := make([]database.Coupon, 0)
	for _, c := range coupons {
		if !isCouponActive(c) {
			continue
		}
		if len(c.MatchIDs) > 0 {
			matchAllowed := false
			for _, mid := range c.MatchIDs {
				if mid == eventID {
					matchAllowed = true
					break
				}
			}
			if !matchAllowed {
				continue
			}
		}
		if err := rt.db.RecordCouponView(c.ID); err != nil {
			ctx.Logger.WithError(err).Warn("coupon view tracking")
		}
		filtered = append(filtered, c)
	}

	writeJSON(w, http.StatusOK, filtered)
}

func (rt *_router) recordCouponView(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, _ := strconv.Atoi(chi.URLParam(r, "couponId"))
	if err := rt.db.RecordCouponView(id); err != nil {
		ctx.Logger.WithError(err).Warn("view coupon")
	}
	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) claimCoupon(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, _ := strconv.Atoi(chi.URLParam(r, "couponId"))
	var payload couponClaimRequest
	_ = json.NewDecoder(r.Body).Decode(&payload)
	var userPtr *int
	if payload.UserID > 0 {
		userPtr = &payload.UserID
	}

	coupon, err := rt.db.GetCouponByID(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	redemptionType := normalizeCouponRedemptionType(coupon.RedemptionType)
	if redemptionType == "code" {
		if !isCouponActive(coupon) {
			w.WriteHeader(http.StatusConflict)
			return
		}
		if coupon.MaxUses > 0 && coupon.TotalClaims >= coupon.MaxUses {
			w.WriteHeader(http.StatusConflict)
			return
		}
		code := strings.TrimSpace(coupon.ManualCode)
		if code == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := rt.db.RecordCouponClaim(coupon.ID); err != nil {
			ctx.Logger.WithError(err).Warn("claim manual coupon")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"code":              code,
			"redeem_url":        strings.TrimSpace(coupon.RedeemURL),
			"redemption_type":   redemptionType,
			"coupon_id":         coupon.ID,
			"sponsor_id":        coupon.SponsorID,
			"merchant_id":       coupon.MerchantID,
			"manual_redemption": true,
		})
		return
	}

	claim, err := rt.db.ClaimCoupon(id, userPtr, payload.MatchID)
	if err != nil {
		ctx.Logger.WithError(err).Warn("claim coupon")
		if err == database.ErrCouponUnavailable || err == database.ErrCouponMaxReached {
			w.WriteHeader(http.StatusConflict)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	merchantID := 0
	orgSlug := ""
	if claim.Coupon != nil {
		merchantID = claim.Coupon.MerchantID
		if claim.Coupon.OrganizationID > 0 {
			if org, err := rt.db.GetOrganization(claim.Coupon.OrganizationID); err == nil {
				orgSlug = strings.TrimSpace(org.Slug)
			} else {
				ctx.Logger.WithError(err).Warn("cannot resolve organization for coupon")
			}
		}
	}
	signature := signCouponPayload(rt.VoteSecret, claim.Code, merchantID)
	validationURL := rt.buildCouponValidationURL(claim.Code, merchantID, signature, orgSlug)

	response := struct {
		database.UserCoupon `json:",inline"`
		Signature           string `json:"signature"`
		QRData              string `json:"qr_data,omitempty"`
		RedemptionType      string `json:"redemption_type,omitempty"`
		RedeemURL           string `json:"redeem_url,omitempty"`
	}{UserCoupon: claim, Signature: signature, QRData: validationURL, RedemptionType: redemptionType, RedeemURL: strings.TrimSpace(coupon.RedeemURL)}

	writeJSON(w, http.StatusOK, response)
}

func (rt *_router) validatePartnerCoupon(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var payload couponValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	payload.Code = strings.TrimSpace(payload.Code)
	if payload.Code == "" || strings.TrimSpace(payload.Signature) == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if ctx.MerchantID == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	expectedSignature := signCouponPayload(rt.VoteSecret, payload.Code, ctx.MerchantID)
	if payload.Signature != expectedSignature {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "firma non valida"})
		return
	}

	claim, err := rt.db.RedeemCoupon(payload.Code, ctx.MerchantID)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "codice inesistente"})
		case errors.Is(err, database.ErrCouponWrongSponsor):
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "sponsor errato"})
		case errors.Is(err, database.ErrCouponExpired):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "coupon scaduto"})
		case errors.Is(err, database.ErrCouponAlreadyUsed):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "coupon già usato"})
		case errors.Is(err, database.ErrCouponMaxReached):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "limite max raggiunto"})
		case errors.Is(err, database.ErrCouponUnavailable):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "coupon non attivo"})
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "valido", "coupon": claim})
}

func signCouponPayload(secret, code string, merchantID int) string {
	payload := code
	if merchantID > 0 {
		payload = fmt.Sprintf("%s|%d", code, merchantID)
	}
	return signCode(secret, payload)
}

func (rt *_router) buildCouponValidationURL(code string, merchantID int, signature string, orgSlug string) string {
	baseURL := strings.TrimSpace(rt.ticketValidationBaseURL)
	if baseURL == "" || code == "" || signature == "" {
		return ""
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}

	basePath := strings.TrimSuffix(parsed.Path, "/")
	if orgSlug != "" {
		parsed.Path = fmt.Sprintf("%s/%s/partner/validate", basePath, url.PathEscape(orgSlug))
	} else {
		parsed.Path = basePath + "/partner/validate"
	}
	q := parsed.Query()
	q.Set("c", code)
	q.Set("s", signature)
	if merchantID > 0 {
		q.Set("m", strconv.Itoa(merchantID))
	}
	if orgSlug != "" {
		q.Set("organization_slug", orgSlug)
	}
	parsed.RawQuery = q.Encode()
	return parsed.String()
}

func (rt *_router) listUserCoupons(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	userIDParam := strings.TrimSpace(r.URL.Query().Get("user_id"))
	var userPtr *int
	if userIDParam != "" {
		if val, err := strconv.Atoi(userIDParam); err == nil && val > 0 {
			userPtr = &val
		}
	}
	coupons, err := rt.db.ListUserCoupons(userPtr, 0)
	if err != nil {
		ctx.Logger.WithError(err).Error("list user coupons")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, coupons)
}

func isCouponActive(c database.Coupon) bool {
	if !strings.EqualFold(c.Status, "active") {
		return false
	}
	if c.StartDate != "" {
		if t, err := time.Parse(time.RFC3339, c.StartDate); err == nil && time.Now().Before(t) {
			return false
		}
	}
	if c.EndDate != "" {
		if t, err := time.Parse(time.RFC3339, c.EndDate); err == nil && time.Now().After(t) {
			return false
		}
	}
	return true
}

func defaultCouponStatus(status string) string {
	trimmed := strings.TrimSpace(status)
	if trimmed == "" {
		return "draft"
	}
	return trimmed
}

func normalizeCouponRedemptionType(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "code" {
		return "code"
	}
	return "qr"
}
