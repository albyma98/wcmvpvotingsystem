package api

import (
	"net/http"
)

// liveness is an HTTP handler that checks the API server status. If the server cannot serve requests (e.g., some
// resources are not ready), this should reply with HTTP Status 500. Otherwise, with HTTP Status 200
func (rt *_router) liveness(w http.ResponseWriter, r *http.Request) {
	/* Example of liveness check:
	   if err := rt.DB.Ping(); err != nil {
	           w.WriteHeader(http.StatusInternalServerError)
	           return
	   }*/
	rt.baseLogger.Debug("liveness probe responded with 200")
}
