package api

import (
	"context"
	"net/http"
	"time"
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

func (rt *_router) readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	var exists bool
	if err := rt.db.SQLConn().QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE name='events' AND type='table')`).Scan(&exists); err != nil || !exists {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
