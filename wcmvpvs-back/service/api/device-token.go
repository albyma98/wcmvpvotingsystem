package api

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
)

const (
	deviceTokenCookieName = "wcmvp_device_token"
	deviceTokenLength     = 32
)

func generateDeviceToken() (string, error) {
	buf := make([]byte, deviceTokenLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func readDeviceTokenFromCookie(r *http.Request) string {
	cookie, err := r.Cookie(deviceTokenCookieName)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cookie.Value)
}

func isRequestSecure(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	proto := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")))
	return proto == "https"
}

func (rt *_router) getDeviceToken(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	existing := readDeviceTokenFromCookie(r)
	var token string
	var err error
	fresh := false
	if existing != "" {
		token = existing
	} else {
		token, err = generateDeviceToken()
		if err != nil {
			ctx.Logger.WithError(err).Error("cannot generate device token")
			_ = writeJSONMessage(w, http.StatusInternalServerError, "Servizio non disponibile. Riprova tra pochi istanti.")
			return
		}
		fresh = true
	}

	expiry := time.Now().Add(180 * 24 * time.Hour)
	http.SetCookie(w, &http.Cookie{
		Name:     deviceTokenCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   isRequestSecure(r),
		SameSite: http.SameSiteLaxMode,
		Expires:  expiry,
	})

	resp := map[string]interface{}{
		"token":      token,
		"expires_at": expiry.UTC().Format(time.RFC3339),
	}
	if fresh {
		resp["fresh"] = true
	}

	if err := writeJSON(w, http.StatusOK, resp); err != nil {
		ctx.Logger.WithError(err).Error("cannot write device token response")
	}
}
