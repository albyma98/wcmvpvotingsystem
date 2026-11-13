package main

import (
	"net/http"

	"github.com/gorilla/handlers"
)

func applyCORSHandler(h http.Handler) http.Handler {
	return handlers.CORS(
		handlers.AllowedHeaders([]string{
			"Content-Type",
			"Authorization",
			"X-Organization-Slug",
			"X-Device-Id",
			"X-Fan-Session",
			"X-Merch-Access",
			"X-Matchday-Grant",
		}),
		handlers.AllowedMethods([]string{"GET", "POST", "OPTIONS", "DELETE", "PUT", "PATCH"}),
		handlers.AllowedOrigins(allowedOrigins()),
		handlers.MaxAge(300),
	)(h)
}
