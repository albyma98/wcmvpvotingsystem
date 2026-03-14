package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/go-chi/chi/v5"
)

// clearSSEDeadline removes the write deadline so the long-lived SSE connection
// is not killed by the server's WriteTimeout (default 5s).
func clearSSEDeadline(w http.ResponseWriter) {
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{})
	_ = rc.SetReadDeadline(time.Time{})
}

// getVotesStream opens an SSE stream that emits a notification whenever a new vote
// is cast for the given event. Clients should re-fetch /events/{id}/votes/live on receipt.
func (rt *_router) getVotesStream(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	idParam := chi.URLParam(r, "id")
	eventID, err := strconv.Atoi(idParam)
	if err != nil || eventID <= 0 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Identificativo evento non valido.")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Streaming non supportato.")
		return
	}

	clearSSEDeadline(w)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, ": connected\n\n")
	flusher.Flush()

	ch, unsub := rt.votesHub.Subscribe(eventID)
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
