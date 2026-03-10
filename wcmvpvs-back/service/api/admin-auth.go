package api

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
)

func (rt *_router) wrapAdmin(fn httpRouterHandler) http.HandlerFunc {
	return rt.wrap(func(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
		token := parseBearerToken(r.Header.Get("Authorization"))
		session, ok := rt.getAdminSession(token)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if strings.EqualFold(session.Role, "partner") {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if strings.EqualFold(session.Role, "bar") && !strings.HasPrefix(r.URL.Path, "/admin/bar") {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		ctx.AdminID = session.AdminID
		ctx.AdminRole = session.Role
		ctx.AdminUsername = session.Username

		if !strings.HasPrefix(r.URL.Path, "/admin/master") {
			if ctx.OrganizationID == 0 || ctx.OrganizationSlug == "" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if session.OrganizationID != 0 && session.OrganizationID != ctx.OrganizationID {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
		}

		fn(w, r, ctx)
	})
}

func parseBearerToken(header string) string {
	if header == "" {
		return ""
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}

func (rt *_router) getAdminSession(token string) (adminSession, bool) {
	if token == "" {
		return adminSession{}, false
	}

	rt.adminSessionsMu.RLock()
	session, ok := rt.adminSessions[token]
	rt.adminSessionsMu.RUnlock()
	if !ok {
		return adminSession{}, false
	}

	if time.Now().After(session.ExpiresAt) {
		rt.adminSessionsMu.Lock()
		delete(rt.adminSessions, token)
		rt.adminSessionsMu.Unlock()
		return adminSession{}, false
	}

	// extend the session deadline on each successful validation
	rt.adminSessionsMu.Lock()
	session.ExpiresAt = time.Now().Add(rt.sessionTimeout)
	rt.adminSessions[token] = session
	rt.adminSessionsMu.Unlock()

	return session, true
}

func (rt *_router) createAdminSession(adminID int, username, role string, orgID, orgTeamID int, orgSlug string) (string, error) {
	token, err := generateSessionToken()
	if err != nil {
		return "", err
	}

	rt.adminSessionsMu.Lock()
	rt.adminSessions[token] = adminSession{
		AdminID:            adminID,
		Username:           username,
		Role:               role,
		OrganizationID:     orgID,
		OrganizationSlug:   orgSlug,
		OrganizationTeamID: orgTeamID,
		ExpiresAt:          time.Now().Add(rt.sessionTimeout),
	}
	rt.adminSessionsMu.Unlock()

	return token, nil
}

func (rt *_router) getPartnerSession(token string) (adminSession, bool) {
	if token == "" {
		return adminSession{}, false
	}

	rt.partnerSessionsMu.RLock()
	session, ok := rt.partnerSessions[token]
	rt.partnerSessionsMu.RUnlock()
	if !ok {
		return adminSession{}, false
	}

	if time.Now().After(session.ExpiresAt) {
		rt.partnerSessionsMu.Lock()
		delete(rt.partnerSessions, token)
		rt.partnerSessionsMu.Unlock()
		return adminSession{}, false
	}

	rt.partnerSessionsMu.Lock()
	session.ExpiresAt = time.Now().Add(rt.sessionTimeout)
	rt.partnerSessions[token] = session
	rt.partnerSessionsMu.Unlock()

	return session, true
}

func (rt *_router) createPartnerSession(adminID int, username string, orgID int, orgSlug string) (string, error) {
	token, err := generateSessionToken()
	if err != nil {
		return "", err
	}

	rt.partnerSessionsMu.Lock()
	rt.partnerSessions[token] = adminSession{
		AdminID:          adminID,
		Username:         username,
		Role:             "partner",
		OrganizationID:   orgID,
		OrganizationSlug: orgSlug,
		ExpiresAt:        time.Now().Add(rt.sessionTimeout),
	}
	rt.partnerSessionsMu.Unlock()

	return token, nil
}

func (rt *_router) wrapPartner(fn httpRouterHandler) http.HandlerFunc {
	return rt.wrap(func(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
		token := parseBearerToken(r.Header.Get("Authorization"))
		session, ok := rt.getPartnerSession(token)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if ctx.OrganizationID == 0 || ctx.OrganizationSlug == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if session.OrganizationID != ctx.OrganizationID {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		ctx.MerchantID = session.AdminID
		ctx.MerchantUsername = session.Username
		ctx.OrganizationID = session.OrganizationID
		ctx.OrganizationSlug = session.OrganizationSlug

		fn(w, r, ctx)
	})
}

func generateSessionToken() (string, error) {
	const size = 32
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(buf)
	if token == "" {
		return "", errors.New("empty session token")
	}
	return token, nil
}
