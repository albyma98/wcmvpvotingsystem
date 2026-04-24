package main

import (
	"net/http"
	"os"
	"strings"
)

func applySecurityHeaders(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent MIME-type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// Block iframe embedding (clickjacking)
		w.Header().Set("X-Frame-Options", "DENY")
		// Enforce HTTPS for 1 year, include subdomains
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		// Limit referrer info sent to external sites
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// Disable browser features not needed by the app
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		h.ServeHTTP(w, r)
	})
}

// allowedOrigin returns the CORS origin from env ALLOWED_ORIGIN,
// falling back to the production domain. Supports multiple origins
// separated by comma (e.g. "https://mvp.wearingcash.it,http://localhost:5173").
func allowedOrigins() []string {
	raw := strings.TrimSpace(os.Getenv("ALLOWED_ORIGIN"))
	if raw == "" {
		return []string{"https://mvp.wearingcash.it"}
	}
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, p := range parts {
		if o := strings.TrimSpace(p); o != "" {
			origins = append(origins, o)
		}
	}
	return origins
}
