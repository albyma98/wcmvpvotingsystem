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
		if !rt.validateAdminToken(token) {
			w.WriteHeader(http.StatusUnauthorized)
			return
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

func (rt *_router) validateAdminToken(token string) bool {
	if token == "" {
		return false
	}

	rt.adminSessionsMu.RLock()
	session, ok := rt.adminSessions[token]
	rt.adminSessionsMu.RUnlock()
	if !ok {
		return false
	}

	if time.Now().After(session.ExpiresAt) {
		rt.adminSessionsMu.Lock()
		delete(rt.adminSessions, token)
		rt.adminSessionsMu.Unlock()
		return false
	}

	// extend the session deadline on each successful validation
	rt.adminSessionsMu.Lock()
	session.ExpiresAt = time.Now().Add(rt.sessionTimeout)
	rt.adminSessions[token] = session
	rt.adminSessionsMu.Unlock()

	return true
}

func (rt *_router) createAdminSession(adminID int) (string, error) {
	token, err := generateSessionToken()
	if err != nil {
		return "", err
	}

	rt.adminSessionsMu.Lock()
	rt.adminSessions[token] = adminSession{
		AdminID:   adminID,
		ExpiresAt: time.Now().Add(rt.sessionTimeout),
	}
	rt.adminSessionsMu.Unlock()

	return token, nil
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
