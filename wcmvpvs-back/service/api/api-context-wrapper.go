package api

import (
	"net/http"
	"strings"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// httpRouterHandler is the signature for functions that accepts a reqcontext.RequestContext in addition to those
// required by the standard http handlers.
type httpRouterHandler func(http.ResponseWriter, *http.Request, reqcontext.RequestContext)

// wrap parses the request and adds a reqcontext.RequestContext instance related to the request.
func (rt *_router) wrap(fn httpRouterHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqUUID, err := uuid.NewV4()
		if err != nil {
			rt.baseLogger.WithError(err).Error("can't generate a request UUID")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var ctx = reqcontext.RequestContext{
			ReqUUID: reqUUID,
		}

		// Create a request-specific logger
		ctx.Logger = rt.baseLogger.WithFields(logrus.Fields{
			"reqid":     ctx.ReqUUID.String(),
			"remote-ip": r.RemoteAddr,
		})

		if !rt.populateOrganizationFromRequest(w, r, &ctx) {
			return
		}
		if !rt.enforceBarFeatureAccess(w, r, ctx) {
			return
		}
		ctx.Logger.Infof("handling %s %s", r.Method, r.URL.Path)
		// Call the next handler in chain (usually, the handler function for the path)
		fn(w, r, ctx)
	}
}

func (rt *_router) isBarFeaturePath(path string) bool {
	return strings.HasPrefix(path, "/bar") || strings.HasPrefix(path, "/admin/bar") || strings.HasPrefix(path, "/ai/bar")
}

func (rt *_router) enforceBarFeatureAccess(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) bool {
	if !rt.isBarFeaturePath(r.URL.Path) {
		return true
	}
	if ctx.OrganizationID == 0 {
		return true
	}
	org, err := rt.db.GetOrganization(ctx.OrganizationID)
	if err != nil {
		ctx.Logger.WithError(err).Warn("cannot resolve organization while enforcing bar feature flag")
		w.WriteHeader(http.StatusNotFound)
		return false
	}
	if org.BarEnabled {
		return true
	}
	w.WriteHeader(http.StatusNotFound)
	return false
}

func (rt *_router) populateOrganizationFromRequest(w http.ResponseWriter, r *http.Request, ctx *reqcontext.RequestContext) bool {
	slug := strings.TrimSpace(r.Header.Get("X-Organization-Slug"))
	if slug == "" {
		slug = strings.TrimSpace(r.URL.Query().Get("organization_slug"))
	}
	if slug == "" {
		return true
	}

	org, err := rt.db.GetOrganizationBySlug(slug)
	if err != nil {
		ctx.Logger.WithError(err).Warn("organization not found for slug")
		w.WriteHeader(http.StatusNotFound)
		return false
	}

	ctx.OrganizationSlug = org.Slug
	ctx.OrganizationID = org.ID
	ctx.OrganizationTeamID = org.TeamID
	return true
}
