package api

// ============================================================================
// TOURNAMENT STREAM — SSE pubblico per la modalità torneo.
// Sostituisce il polling: i client (tifosi, console operatore, pannello admin)
// aprono un EventSource su /v1/tournaments/{slug}/stream e, a ogni notifica,
// rifanno la loro fetch (home/live/matches/standings). Il segnale è un semplice
// "tick" senza dati sensibili: un unico stream pubblico per-slug serve tutti.
// L'hub è keyed per event ID; qui risolviamo slug -> eventID una volta sola.
// ============================================================================

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func (rt *_router) getTournamentStream(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	eventID, err := rt.store.EventIDBySlug(r.Context(), slug)
	if err != nil {
		http.Error(w, `{"error":"tournament_not_found"}`, http.StatusNotFound)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, `{"error":"stream_unsupported"}`, http.StatusInternalServerError)
		return
	}

	// Rimuove le deadline: la connessione SSE è long-lived e verrebbe uccisa
	// dal WriteTimeout del server (vedi clearSSEDeadline in sse_votes.go).
	clearSSEDeadline(w)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, ": connected\n\n")
	flusher.Flush()

	ch, unsub := rt.tournamentHub.Subscribe(int(eventID))
	defer unsub()

	keepalive := time.NewTicker(25 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ch:
			fmt.Fprintf(w, "event: update\ndata: {}\n\n")
			flusher.Flush()
		case <-keepalive.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}
