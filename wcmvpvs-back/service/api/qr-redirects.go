package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/go-chi/chi/v5"
)

func (rt *_router) listMasterQRRedirects(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !rt.ensureSuperAdmin(w, ctx) {
		return
	}

	redirects, err := rt.db.ListQRRedirects()
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list qr redirects")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(redirects)
}

func (rt *_router) upsertMasterQRRedirect(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !rt.ensureSuperAdmin(w, ctx) {
		return
	}

	var payload struct {
		SourcePath string `json:"source_path"`
		TargetPath string `json:"target_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		ctx.Logger.WithError(err).Warn("invalid qr redirect payload")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sourcePath, targetPath, ok := normalizeQRRedirectPayload(payload.SourcePath, payload.TargetPath)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	entry, err := rt.db.UpsertQRRedirect(sourcePath, targetPath)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot save qr redirect")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(entry)
}

func (rt *_router) deleteMasterQRRedirect(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !rt.ensureSuperAdmin(w, ctx) {
		return
	}

	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	if id <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := rt.db.DeleteQRRedirect(id); err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		ctx.Logger.WithError(err).Error("cannot delete qr redirect")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) handleQRRedirectNotFound(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	path := strings.TrimSpace(r.URL.Path)
	if path == "" || path == "/" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	redirect, err := rt.db.GetQRRedirectBySource(path)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		ctx.Logger.WithError(err).Error("cannot resolve qr redirect")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if r.Method == http.MethodGet {
		if err := rt.db.IncrementQRRedirectHit(redirect.ID); err != nil {
			ctx.Logger.WithError(err).Warn("cannot increment qr redirect hit")
		}
	}

	http.Redirect(w, r, redirect.TargetPath, http.StatusFound)
}

func normalizeQRRedirectPayload(sourcePath, targetPath string) (string, string, bool) {
	cleanSource := strings.TrimSpace(sourcePath)
	cleanTarget := strings.TrimSpace(targetPath)
	if cleanSource == "" || cleanTarget == "" {
		return "", "", false
	}
	if !strings.HasPrefix(cleanSource, "/") {
		cleanSource = "/" + cleanSource
	}
	if !strings.HasPrefix(cleanTarget, "/") && !strings.HasPrefix(strings.ToLower(cleanTarget), "http://") && !strings.HasPrefix(strings.ToLower(cleanTarget), "https://") {
		return "", "", false
	}
	if strings.HasPrefix(cleanSource, "/admin") {
		return "", "", false
	}
	return cleanSource, cleanTarget, true
}
