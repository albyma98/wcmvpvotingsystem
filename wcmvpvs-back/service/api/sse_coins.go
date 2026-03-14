package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/go-chi/chi/v5"
)

// getCoinsStream opens an SSE stream that emits a notification whenever the coins
// leaderboard for the given event changes. Clients should re-fetch
// /events/{eventId}/coins-leaderboard on receipt.
func (rt *_router) getCoinsStream(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	idParam := chi.URLParam(r, "eventId")
	eventID, err := parseNumericID(idParam)
	if err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Identificativo evento non valido.")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Streaming non supportato.")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, ": connected\n\n")
	flusher.Flush()

	ch, unsub := rt.coinsHub.Subscribe(eventID)
	defer unsub()

	keepalive := time.NewTicker(25 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ch:
			fmt.Fprintf(w, "data: {}\n\n")
			flusher.Flush()
		case <-keepalive.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}
