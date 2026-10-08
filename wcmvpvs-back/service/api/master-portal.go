package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
)

func (rt *_router) ensureSuperAdmin(w http.ResponseWriter, ctx reqcontext.RequestContext) bool {
	if strings.EqualFold(ctx.AdminRole, "superadmin") {
		return true
	}
	w.WriteHeader(http.StatusForbidden)
	return false
}

func (rt *_router) getMasterSummary(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !rt.ensureSuperAdmin(w, ctx) {
		return
	}

	summary, err := rt.db.GetMasterDashboardSummary()
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot load master summary")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(summary)
}

func (rt *_router) getMasterAnalytics(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !rt.ensureSuperAdmin(w, ctx) {
		return
	}

	analytics, err := rt.db.GetMasterAnalytics()
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot load master analytics")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(analytics)
}

func (rt *_router) listMasterOrganizations(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !rt.ensureSuperAdmin(w, ctx) {
		return
	}

	orgs, err := rt.db.ListOrganizations()
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list organizations")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(orgs)
}

func (rt *_router) createMasterOrganization(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !rt.ensureSuperAdmin(w, ctx) {
		return
	}

	var payload struct {
		Name       string  `json:"name"`
		Slug       string  `json:"slug"`
		City       string  `json:"city"`
		LogoURL    string  `json:"logo_url"`
		IsActive   bool    `json:"is_active"`
		SMSCost    float64 `json:"sms_cost"`
		FreeSMS    int     `json:"free_sms"`
		BarEnabled bool    `json:"bar_enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		ctx.Logger.WithError(err).Warn("invalid payload while creating organization")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	org, err := rt.db.CreateOrganization(database.Organization{
		Name:       payload.Name,
		Slug:       payload.Slug,
		City:       payload.City,
		LogoURL:    payload.LogoURL,
		IsActive:   payload.IsActive,
		SMSCost:    payload.SMSCost,
		FreeSMS:    payload.FreeSMS,
		BarEnabled: payload.BarEnabled,
	})
	if err != nil {
		if errors.Is(err, database.ErrInvalidOrganizationData) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		ctx.Logger.WithError(err).Error("cannot create organization")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(org)
}

func (rt *_router) updateMasterOrganization(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !rt.ensureSuperAdmin(w, ctx) {
		return
	}

	orgID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || orgID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var payload struct {
		Name       string  `json:"name"`
		Slug       string  `json:"slug"`
		City       string  `json:"city"`
		LogoURL    string  `json:"logo_url"`
		IsActive   bool    `json:"is_active"`
		SMSCost    float64 `json:"sms_cost"`
		FreeSMS    int     `json:"free_sms"`
		BarEnabled bool    `json:"bar_enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		ctx.Logger.WithError(err).Warn("invalid payload while updating organization")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	org, err := rt.db.UpdateOrganization(database.Organization{
		ID:         orgID,
		Name:       payload.Name,
		Slug:       payload.Slug,
		City:       payload.City,
		LogoURL:    payload.LogoURL,
		IsActive:   payload.IsActive,
		SMSCost:    payload.SMSCost,
		FreeSMS:    payload.FreeSMS,
		BarEnabled: payload.BarEnabled,
	})
	if err != nil {
		switch {
		case errors.Is(err, database.ErrInvalidOrganizationData):
			w.WriteHeader(http.StatusBadRequest)
		case errors.Is(err, sql.ErrNoRows):
			w.WriteHeader(http.StatusNotFound)
		default:
			ctx.Logger.WithError(err).Error("cannot update organization")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	_ = json.NewEncoder(w).Encode(org)
}

func (rt *_router) getMasterOrganizationDetail(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !rt.ensureSuperAdmin(w, ctx) {
		return
	}

	orgID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || orgID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	org, err := rt.db.GetOrganization(orgID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		ctx.Logger.WithError(err).Error("cannot load organization detail")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	stats, err := rt.db.GetOrganizationStats(orgID)
	if err != nil {
		ctx.Logger.WithError(err).WithField("organization_id", orgID).Error("cannot load organization stats")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(struct {
		Organization database.Organization      `json:"organization"`
		Stats        database.OrganizationStats `json:"stats"`
	}{Organization: org, Stats: stats})
}

func (rt *_router) deleteMasterOrganization(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !rt.ensureSuperAdmin(w, ctx) {
		return
	}
	orgID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || orgID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var payload struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || strings.TrimSpace(payload.Password) == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if !rt.allowLoginRequest(ctx.AdminUsername, clientIPFromRequest(r)) {
		w.WriteHeader(http.StatusTooManyRequests)
		return
	}
	admin, err := rt.db.GetAdminByID(ctx.AdminID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot verify superadmin before deleting organization")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !strings.EqualFold(admin.Role, "superadmin") || !adminPasswordMatches(admin.PasswordHash, payload.Password) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	eventIDs, err := rt.db.DeleteOrganizationData(orgID)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			w.WriteHeader(http.StatusNotFound)
		case errors.Is(err, database.ErrProtectedOrganization), errors.Is(err, database.ErrOrganizationSharedData):
			w.WriteHeader(http.StatusConflict)
		default:
			ctx.Logger.WithError(err).WithField("organization_id", orgID).Error("cannot delete organization")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	rt.adminSessionsMu.Lock()
	for token, session := range rt.adminSessions {
		if session.OrganizationID == orgID {
			delete(rt.adminSessions, token)
		}
	}
	rt.adminSessionsMu.Unlock()
	rt.partnerSessionsMu.Lock()
	for token, session := range rt.partnerSessions {
		if session.OrganizationID == orgID {
			delete(rt.partnerSessions, token)
		}
	}
	rt.partnerSessionsMu.Unlock()

	mediaCleanupFailed := false
	for _, eventID := range eventIDs {
		for _, mediaDir := range []string{"selfies", "story-videos"} {
			path := filepath.Join("tmp", mediaDir, fmt.Sprintf("event_%d", eventID))
			if err := os.RemoveAll(path); err != nil {
				mediaCleanupFailed = true
				ctx.Logger.WithError(err).WithField("path", path).Warn("cannot remove organization event media")
			}
		}
	}
	ctx.Logger.WithField("organization_id", orgID).Info("organization permanently deleted")
	_ = json.NewEncoder(w).Encode(struct {
		MediaCleanupFailed bool `json:"media_cleanup_failed"`
	}{MediaCleanupFailed: mediaCleanupFailed})
}
