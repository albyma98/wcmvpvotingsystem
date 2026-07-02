package api

// Tournament Mode — contratto API per la home torneo.
// Due endpoint: /home (snapshot completo, chiamato una volta) e /live
// (payload minimo, pollato ogni 10s da centinaia di device in tribuna).
// Tenere /live leggero è ciò che rende sostenibile il polling su Lightsail.
//
// Adattato per ArenaBoostX: gli handler sono metodi di *_router (il server
// reale del progetto). store e liveCache sono campi di _router, iniettati in
// New(). Lo stub `type Server struct` originale è stato fuso qui e rimosso;
// il writeJSON del modulo è stato eliminato in favore di quello esistente
// in response.go.

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type Team struct {
	Name string `json:"name"`
	Logo string `json:"logo,omitempty"`
	Sub  string `json:"sub,omitempty"` // es. città: "PESARO"
}

type Score struct {
	A int `json:"a"`
	B int `json:"b"`
}

type LiveMatch struct {
	ID       string   `json:"id"`
	Court    string   `json:"court"` // "CAMPO 2"
	TeamA    Team     `json:"teamA"`
	TeamB    Team     `json:"teamB"`
	Score    Score    `json:"score"`    // set vinti
	SetLabel string   `json:"setLabel"` // "1° SET"
	Sets     []string `json:"sets"`     // ["21-18","18-21"]
}

type NextMatch struct {
	Court string `json:"court"`
	Time  string `json:"time"` // "18:30" — già formattato lato server, il client non fa timezone math
	TeamA Team   `json:"teamA"`
	TeamB Team   `json:"teamB"`
}

type Tile struct {
	ID    string `json:"id"`
	Icon  string `json:"icon"` // chiave icona nel client: calendar|chart|bracket|star|trophy|gallery|doc|info
	Label string `json:"label"`
	Sub   string `json:"sub"`
	Color string `json:"color"` // hex — configurabile per torneo dal pannello admin
	Route string `json:"route"`
}

type Sponsor struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Logo       string `json:"logo,omitempty"`
	URL        string `json:"url,omitempty"`
	Tier       string `json:"tier"`                 // "main" | "partner" — inventory differenziato
	BrandColor string `json:"brandColor,omitempty"` // usato dal marquee quando manca il logo
}

type TournamentInfo struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Format      string `json:"format"`      // "BEACH VOLLEY 4X4"
	DateLabel   string `json:"dateLabel"`   // "8 - 11 GIUGNO 2024"
	Location    string `json:"location"`    // "LIDO DI CLASSE, RA"
	StatusLabel string `json:"statusLabel"` // "TORNEO IN CORSO"
	PhaseLabel  string `json:"phaseLabel"`  // "FASE A GIRONI"
	Logo        string `json:"logo,omitempty"`
	HeroImage   string `json:"heroImage,omitempty"`
}

type HomeResponse struct {
	Tournament  TournamentInfo `json:"tournament"`
	LiveMatches []LiveMatch    `json:"liveMatches"`
	NextMatch   *NextMatch     `json:"nextMatch,omitempty"`
	Tiles       []Tile         `json:"tiles"`
	Sponsors    []Sponsor      `json:"sponsors"`
}

type LiveResponse struct {
	Tournament  *TournamentPhase `json:"tournament,omitempty"`
	LiveMatches []LiveMatch      `json:"liveMatches"`
	NextMatch   *NextMatch       `json:"nextMatch"`
}

type TournamentPhase struct {
	StatusLabel string `json:"statusLabel"`
	PhaseLabel  string `json:"phaseLabel"`
}

// HandleTournamentLive risponde al polling dei device in tribuna.
// Cache in-memory con TTL 3s: 500 device che pollano ogni 10s = ~50 req/s,
// ma SQLite viene toccato al massimo ~1 volta ogni 3 secondi.
func (rt *_router) HandleTournamentLive(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	if cached, ok := rt.liveCache.Get(slug); ok {
		w.Header().Set("Cache-Control", "public, max-age=3")
		_ = writeJSON(w, http.StatusOK, cached)
		return
	}

	resp, err := rt.store.GetTournamentLive(r.Context(), slug)
	if err != nil {
		http.Error(w, `{"error":"tournament_not_found"}`, http.StatusNotFound)
		return
	}

	rt.liveCache.Set(slug, resp, 3*time.Second)

	// Il client può cachare a sua volta: riduce richieste duplicate su reti d'arena instabili
	w.Header().Set("Cache-Control", "public, max-age=3")
	_ = writeJSON(w, http.StatusOK, resp)
}

// HandleTournamentHome: snapshot completo, cacheable più a lungo
// (tiles e sponsor cambiano solo da pannello admin).
func (rt *_router) HandleTournamentHome(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	resp, err := rt.store.GetTournamentHome(r.Context(), slug)
	if err != nil {
		http.Error(w, `{"error":"tournament_not_found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=30")
	_ = writeJSON(w, http.StatusOK, resp)
}

// decodeSets converte la colonna sets_json (es. `["21-18","18-21"]`) in slice.
func decodeSets(raw string) []string {
	if raw == "" || raw == "null" {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return []string{}
	}
	return out
}
