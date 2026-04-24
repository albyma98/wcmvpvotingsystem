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
		}),
		handlers.AllowedMethods([]string{"GET", "POST", "OPTIONS", "DELETE", "PUT"}),
		handlers.AllowedOrigins(allowedOrigins()),
		handlers.MaxAge(300),
	)(h)
}
