package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
)

type couponPayload struct {
	Title        string `json:"title"`
	ShortDesc    string `json:"short_desc"`
	SponsorID    int    `json:"sponsor_id"`
	MatchIDs     []int  `json:"match_ids"`
	StartDate    string `json:"start_date"`
	EndDate      string `json:"end_date"`
	MaxUses      int    `json:"max_uses"`
	Status       string `json:"status"`
	ImageURL     string `json:"image_url"`
	Highlight    bool   `json:"highlight"`
	Segmentation string `json:"segmentation"`
}

type couponClaimRequest struct {
	MatchID int `json:"match_id"`
	UserID  int `json:"user_id"`
}

type couponValidationRequest struct {
	Code      string `json:"code"`
	SponsorID int    `json:"sponsor_id"`
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

	coupon := database.Coupon{
		Title:          payload.Title,
		ShortDesc:      payload.ShortDesc,
		SponsorID:      payload.SponsorID,
		MatchIDs:       payload.MatchIDs,
		StartDate:      payload.StartDate,
		EndDate:        payload.EndDate,
		MaxUses:        payload.MaxUses,
		Status:         defaultCouponStatus(payload.Status),
		ImageURL:       payload.ImageURL,
		Highlight:      payload.Highlight,
		Segmentation:   payload.Segmentation,
		OrganizationID: ctx.OrganizationID,
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

	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	coupon := database.Coupon{
		ID:             id,
		Title:          payload.Title,
		ShortDesc:      payload.ShortDesc,
		SponsorID:      payload.SponsorID,
		MatchIDs:       payload.MatchIDs,
		StartDate:      payload.StartDate,
		EndDate:        payload.EndDate,
		MaxUses:        payload.MaxUses,
		Status:         defaultCouponStatus(payload.Status),
		ImageURL:       payload.ImageURL,
		Highlight:      payload.Highlight,
		Segmentation:   payload.Segmentation,
		OrganizationID: ctx.OrganizationID,
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
	segment := strings.TrimSpace(r.URL.Query().Get("segment"))
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
		if segment != "" && !strings.EqualFold(c.Segmentation, "all") && !strings.EqualFold(c.Segmentation, segment) {
			continue
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
	writeJSON(w, http.StatusOK, claim)
}

func (rt *_router) validatePartnerCoupon(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var payload couponValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || strings.TrimSpace(payload.Code) == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if payload.SponsorID == 0 {
		payload.SponsorID = ctx.OrganizationID
	}
	claim, err := rt.db.RedeemCoupon(payload.Code, payload.SponsorID)
	if err != nil {
		if err == database.ErrCouponUnavailable {
			writeJSON(w, http.StatusOK, map[string]string{"status": "scaduto"})
			return
		}
		if err == database.ErrCouponMaxReached {
			writeJSON(w, http.StatusOK, map[string]string{"status": "usato"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "valido", "coupon": claim})
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
