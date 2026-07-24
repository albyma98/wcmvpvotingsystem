package api

// ============================================================================
// TOURNAMENT ADMIN — layer HTTP.
// Due mondi in questo file, entrambi paralleli alle società:
//
// 1) MASTER (/admin/master/tournaments): il superadmin crea/elenca i tornei.
//    Riusa wrapAdmin + ensureSuperAdmin esistenti — zero logica nuova di auth.
//
// 2) TOURNAMENT ADMIN (/v1/ta/{slug}/...): il pannello dell'organizzatore.
//    Credenziali dedicate (tournament_admins), cookie separato (ta_session),
//    sessione scopata sull'evento: l'admin del torneo X non tocca il torneo Y
//    né, mai, nulla del mondo società.
// ============================================================================

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
)

const taCookieName = "ta_session"
const taSessionTTL = 96 * time.Hour // copre un torneo weekend con margine

// registerTournamentAdminRoutes: chiamata da registerTournamentRoutes.
func registerTournamentAdminRoutes(rt *_router) {
	// --- Master (superadmin) ---
	rt.router.Get("/admin/master/tournaments", rt.wrapAdmin(rt.listMasterTournaments))
	rt.router.Post("/admin/master/tournaments", rt.wrapAdmin(rt.createMasterTournament))
	rt.router.Put("/admin/master/tournaments/{id}/password", rt.wrapAdmin(rt.resetMasterTournamentPassword))
	rt.router.Delete("/admin/master/tournaments/{id}", rt.wrapAdmin(rt.deleteMasterTournament))

	// --- Auth admin torneo ---
	rt.router.Post("/v1/ta/{slug}/login", rt.taLogin)
	rt.router.Post("/v1/ta/{slug}/logout", rt.taLogout)

	// --- Pannello (tutte protette da wrapTA) ---
	rt.router.Get("/v1/ta/{slug}/overview", rt.wrapTA(rt.taOverview))
	rt.router.Get("/v1/ta/{slug}/teams", rt.wrapTA(rt.taListTeams))
	rt.router.Post("/v1/ta/{slug}/teams", rt.wrapTA(rt.taCreateTeams))
	rt.router.Put("/v1/ta/{slug}/teams/{id}", rt.wrapTA(rt.taUpdateTeam))
	rt.router.Put("/v1/ta/{slug}/teams/{id}/players", rt.wrapTA(rt.taSetTeamPlayers))
	rt.router.Delete("/v1/ta/{slug}/teams/{id}", rt.wrapTA(rt.taDeleteTeam))
	rt.router.Get("/v1/ta/{slug}/matches", rt.wrapTA(rt.taListMatches))
	rt.router.Post("/v1/ta/{slug}/matches", rt.wrapTA(rt.taCreateMatch))
	rt.router.Put("/v1/ta/{slug}/matches/{id}", rt.wrapTA(rt.taUpdateMatch))
	rt.router.Put("/v1/ta/{slug}/matches/{id}/anchor", rt.wrapTA(rt.taSetMatchAnchor))
	rt.router.Delete("/v1/ta/{slug}/matches/{id}", rt.wrapTA(rt.taDeleteMatch))
	rt.router.Post("/v1/ta/{slug}/matches/{id}/score", rt.wrapTA(rt.taScoreAction))
	rt.router.Get("/v1/ta/{slug}/sponsors", rt.wrapTA(rt.taListSponsors))
	rt.router.Post("/v1/ta/{slug}/sponsors", rt.wrapTA(rt.taCreateSponsor))
	rt.router.Delete("/v1/ta/{slug}/sponsors/{id}", rt.wrapTA(rt.taDeleteSponsor))
	rt.router.Get("/v1/ta/{slug}/shop", rt.wrapTA(rt.taListShopProducts))
	rt.router.Post("/v1/ta/{slug}/shop", rt.wrapTA(rt.taCreateShopProduct))
	rt.router.Delete("/v1/ta/{slug}/shop/{id}", rt.wrapTA(rt.taDeleteShopProduct))
	rt.router.Get("/v1/ta/{slug}/shop/reservations", rt.wrapTA(rt.taListShopReservations))
	rt.router.Get("/v1/ta/{slug}/settings", rt.wrapTA(rt.taGetSettings))
	rt.router.Put("/v1/ta/{slug}/settings", rt.wrapTA(rt.taUpdateSettings))
	rt.router.Post("/v1/ta/{slug}/bracket/generate", rt.wrapTA(rt.taGenerateBracket))
}

// ============================== MASTER ======================================

var slugCleaner = regexp.MustCompile(`[^a-z0-9]+`)

func slugifyTournament(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = slugCleaner.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func randomPassword(nBytes int) (string, error) {
	buf := make([]byte, nBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func (rt *_router) listMasterTournaments(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !rt.ensureSuperAdmin(w, ctx) {
		return
	}
	list, err := rt.store.ListTournaments(r.Context())
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list tournaments")
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"tournaments": list})
}

func (rt *_router) createMasterTournament(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !rt.ensureSuperAdmin(w, ctx) {
		return
	}
	var in TournamentCreateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, `{"error":"bad_json"}`, http.StatusBadRequest)
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		http.Error(w, `{"error":"name_required"}`, http.StatusBadRequest)
		return
	}
	if in.Slug = slugifyTournament(firstNonEmpty(in.Slug, in.Name)); in.Slug == "" {
		http.Error(w, `{"error":"invalid_slug"}`, http.StatusBadRequest)
		return
	}
	if exists, err := rt.store.SlugExists(r.Context(), in.Slug); err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	} else if exists {
		http.Error(w, `{"error":"slug_taken"}`, http.StatusConflict)
		return
	}

	// Credenziali admin torneo: generate qui, mostrate UNA volta al master.
	adminUsername := "ta-" + in.Slug
	password, err := randomPassword(9) // 12 caratteri url-safe
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}

	eventID, err := rt.store.CreateTournament(r.Context(), in, adminUsername, string(hash))
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot create tournament")
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"eventId":       eventID,
		"slug":          in.Slug,
		"publicPath":    "/t/" + in.Slug,
		"adminPath":     "/ta/" + in.Slug,
		"adminUsername": adminUsername,
		"adminPassword": password, // mostrata solo ora: in DB c'è solo l'hash
	})
}

// resetMasterTournamentPassword reimposta la password admin di un torneo.
// La password non è recuperabile (in DB c'è solo l'hash bcrypt): il master può
// solo assegnarne una nuova. Body {password} opzionale — se vuoto ne generiamo
// una casuale. In entrambi i casi la ritorniamo in chiaro UNA sola volta.
func (rt *_router) resetMasterTournamentPassword(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !rt.ensureSuperAdmin(w, ctx) {
		return
	}
	eventID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || eventID <= 0 {
		http.Error(w, `{"error":"bad_id"}`, http.StatusBadRequest)
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body) // body opzionale
	password := strings.TrimSpace(body.Password)
	if password == "" {
		if password, err = randomPassword(9); err != nil {
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}
	} else if len(password) < 6 {
		http.Error(w, `{"error":"password_too_short"}`, http.StatusBadRequest)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	username, err := rt.store.SetTAAdminPassword(r.Context(), eventID, string(hash))
	if errors.Is(err, errTANotFound) {
		http.Error(w, `{"error":"tournament_not_found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot reset tournament password")
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"adminUsername": username,
		"adminPassword": password,
	})
}

// deleteMasterTournament elimina un torneo e tutti i suoi dati collegati.
func (rt *_router) deleteMasterTournament(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !rt.ensureSuperAdmin(w, ctx) {
		return
	}
	eventID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || eventID <= 0 {
		http.Error(w, `{"error":"bad_id"}`, http.StatusBadRequest)
		return
	}
	// Invalidiamo le cache pubbliche PRIMA del delete (dopo, lo slug non esiste più).
	rt.invalidateTournamentCaches(r, eventID)
	if err := rt.store.DeleteTournament(r.Context(), eventID); err != nil {
		if errors.Is(err, errTANotFound) {
			http.Error(w, `{"error":"tournament_not_found"}`, http.StatusNotFound)
			return
		}
		ctx.Logger.WithError(err).Error("cannot delete tournament")
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

// ========================== AUTH ADMIN TORNEO ================================

func (rt *_router) setTACookie(w http.ResponseWriter, r *http.Request, token string, maxAge int) {
	secure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
	http.SetCookie(w, &http.Cookie{
		Name: taCookieName, Value: token, Path: "/",
		MaxAge: maxAge, HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode,
	})
}

func (rt *_router) taLogin(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"bad_json"}`, http.StatusBadRequest)
		return
	}
	eventID, err := rt.store.EventIDBySlug(r.Context(), slug)
	if err != nil {
		http.Error(w, `{"error":"tournament_not_found"}`, http.StatusNotFound)
		return
	}
	admin, err := rt.store.GetTAAdminByUsername(r.Context(), strings.TrimSpace(body.Username))
	// Confronto sempre eseguito (anche su admin inesistente) per non rivelare
	// via timing quali username esistono.
	dummy := "$2a$10$7EqJtq98hPqEX7fNZaFWoOhi5B0G1Zt0P0mBz0m0m0m0m0m0m0m0m"
	hash := dummy
	if err == nil {
		hash = admin.PasswordHash
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(body.Password)) != nil || err != nil || admin.EventID != eventID {
		http.Error(w, `{"error":"invalid_credentials"}`, http.StatusUnauthorized)
		return
	}
	token, err := rt.store.CreateTASession(r.Context(), admin.ID, eventID, taSessionTTL)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	rt.setTACookie(w, r, token, int(taSessionTTL.Seconds()))
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "slug": slug})
}

func (rt *_router) taLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(taCookieName); err == nil {
		rt.store.DeleteTASession(r.Context(), c.Value)
	}
	rt.setTACookie(w, r, "", -1)
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

// wrapTA: autentica la sessione torneo e verifica che sia scopata sullo slug.
type taHandler func(w http.ResponseWriter, r *http.Request, eventID int64)

func (rt *_router) wrapTA(fn taHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := ""
		if c, err := r.Cookie(taCookieName); err == nil {
			token = c.Value
		}
		if token == "" {
			token = parseBearerToken(r.Header.Get("Authorization"))
		}
		if token == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, eventID, ok := rt.store.GetTASession(r.Context(), token)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		slugEventID, err := rt.store.EventIDBySlug(r.Context(), chi.URLParam(r, "slug"))
		if err != nil || slugEventID != eventID {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		fn(w, r, eventID)
	}
}

// ============================ HANDLER PANNELLO ================================

func (rt *_router) taOverview(w http.ResponseWriter, r *http.Request, eventID int64) {
	settings, slug, err := rt.store.GetTASettings(r.Context(), eventID)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	teams, _ := rt.store.ListTATeams(r.Context(), eventID)
	matches, _ := rt.store.ListTAMatches(r.Context(), eventID)
	live := 0
	for _, m := range matches {
		if m.Status == "live" {
			live++
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"settings": settings, "slug": slug,
		"teamsCount": len(teams), "matchesCount": len(matches), "liveCount": live,
	})
}

func (rt *_router) taListTeams(w http.ResponseWriter, r *http.Request, eventID int64) {
	teams, err := rt.store.ListTATeams(r.Context(), eventID)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"teams": teams})
}

// taCreateTeams accetta {teams:[{name,shortName,city,groupName}]} oppure
// {csv:"Nome;Sigla;Città;Girone\n..."} — l'import CSV è il flusso principale.
func (rt *_router) taCreateTeams(w http.ResponseWriter, r *http.Request, eventID int64) {
	var body struct {
		Teams []TATeam `json:"teams"`
		CSV   string   `json:"csv"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"bad_json"}`, http.StatusBadRequest)
		return
	}
	teams := body.Teams
	for _, line := range strings.Split(body.CSV, "\n") {
		parts := strings.Split(line, ";")
		t := TATeam{Name: strings.TrimSpace(parts[0])}
		if len(parts) > 1 {
			t.ShortName = strings.TrimSpace(parts[1])
		}
		if len(parts) > 2 {
			t.City = strings.TrimSpace(parts[2])
		}
		if len(parts) > 3 {
			t.GroupName = strings.TrimSpace(parts[3])
		}
		if t.Name != "" {
			teams = append(teams, t)
		}
	}
	n, err := rt.store.InsertTATeams(r.Context(), eventID, teams)
	if err != nil {
		rt.baseLogger.WithError(err).WithField("eventID", eventID).Error("cannot insert tournament teams")
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"inserted": n})
}

// taUpdateTeam modifica una squadra già creata (nome, sigla, città, girone).
// La rosa si modifica a parte via taSetTeamPlayers.
func (rt *_router) taUpdateTeam(w http.ResponseWriter, r *http.Request, eventID int64) {
	teamID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var body struct {
		Name      string `json:"name"`
		ShortName string `json:"shortName"`
		City      string `json:"city"`
		GroupName string `json:"groupName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"bad_json"}`, http.StatusBadRequest)
		return
	}
	err := rt.store.UpdateTATeam(r.Context(), eventID, teamID, body.Name, body.ShortName, body.City, body.GroupName)
	if err != nil {
		switch {
		case err == errTANotFound:
			http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		case err.Error() == "name_required":
			http.Error(w, `{"error":"name_required"}`, http.StatusBadRequest)
		default:
			rt.baseLogger.WithError(err).WithField("eventID", eventID).Error("cannot update tournament team")
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		}
		return
	}
	rt.invalidateTournamentCaches(r, eventID)
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

func (rt *_router) taDeleteTeam(w http.ResponseWriter, r *http.Request, eventID int64) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := rt.store.DeleteTATeam(r.Context(), eventID, id); err != nil {
		if err.Error() == "team_in_use" {
			http.Error(w, `{"error":"team_in_use"}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

// taSetTeamPlayers sostituisce l'intera rosa di una squadra: {players:[{firstName,lastName}]}.
// Le coppie vuote vengono scartate lato store; la lista può quindi arrivare piena
// di slot vuoti dal form (8 coppie fisse) senza problemi.
func (rt *_router) taSetTeamPlayers(w http.ResponseWriter, r *http.Request, eventID int64) {
	teamID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var body struct {
		Players []TAPlayer `json:"players"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"bad_json"}`, http.StatusBadRequest)
		return
	}
	if err := rt.store.ReplaceTAPlayers(r.Context(), eventID, teamID, body.Players); err != nil {
		if err == errTANotFound {
			http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
			return
		}
		rt.baseLogger.WithError(err).WithField("eventID", eventID).Error("cannot replace tournament players")
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	rt.invalidateTournamentCaches(r, eventID)
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

func (rt *_router) taListMatches(w http.ResponseWriter, r *http.Request, eventID int64) {
	matches, err := rt.store.ListTAMatches(r.Context(), eventID)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"matches": matches})
}

func (rt *_router) taCreateMatch(w http.ResponseWriter, r *http.Request, eventID int64) {
	var body struct {
		Court       string `json:"court"`
		Time        string `json:"time"`        // "18:30" — mostrato al tifoso
		ScheduledAt string `json:"scheduledAt"` // ISO per l'ordinamento
		Stage       string `json:"stage"`       // '' = girone; altrimenti fase finale
		TeamAID     int64  `json:"teamAId"`
		TeamBID     int64  `json:"teamBId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.TeamAID == 0 || body.TeamBID == 0 || body.TeamAID == body.TeamBID {
		http.Error(w, `{"error":"bad_input"}`, http.StatusBadRequest)
		return
	}
	id, err := rt.store.CreateTAMatch(r.Context(), eventID, body.Court, body.Time, body.ScheduledAt, body.Stage, body.TeamAID, body.TeamBID)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	rt.invalidateTournamentCaches(r, eventID)
	writeJSON(w, http.StatusCreated, map[string]interface{}{"id": id})
}

func (rt *_router) taUpdateMatch(w http.ResponseWriter, r *http.Request, eventID int64) {
	var body struct {
		Court       string `json:"court"`
		Time        string `json:"time"`
		ScheduledAt string `json:"scheduledAt"`
		Stage       string `json:"stage"`
		TeamAID     int64  `json:"teamAId"`
		TeamBID     int64  `json:"teamBId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.TeamAID == 0 || body.TeamBID == 0 || body.TeamAID == body.TeamBID {
		http.Error(w, `{"error":"bad_input"}`, http.StatusBadRequest)
		return
	}
	err := rt.store.UpdateTAMatch(r.Context(), eventID, chi.URLParam(r, "id"),
		body.Court, body.Time, body.ScheduledAt, body.Stage, body.TeamAID, body.TeamBID)
	if errors.Is(err, errTANotFound) {
		http.Error(w, `{"error":"match_not_found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	rt.invalidateTournamentCaches(r, eventID)
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

// taSetMatchAnchor marca/smarca una partita come "inizio torneo" (àncora del
// calendario). Al più una per evento: lo store azzera le altre.
func (rt *_router) taSetMatchAnchor(w http.ResponseWriter, r *http.Request, eventID int64) {
	var body struct {
		Anchor bool `json:"anchor"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"bad_json"}`, http.StatusBadRequest)
		return
	}
	err := rt.store.SetTAMatchAnchor(r.Context(), eventID, chi.URLParam(r, "id"), body.Anchor)
	if errors.Is(err, errTANotFound) {
		http.Error(w, `{"error":"match_not_found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	rt.invalidateTournamentCaches(r, eventID)
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

func (rt *_router) taDeleteMatch(w http.ResponseWriter, r *http.Request, eventID int64) {
	if err := rt.store.DeleteTAMatch(r.Context(), eventID, chi.URLParam(r, "id")); err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	rt.invalidateTournamentCaches(r, eventID)
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

func (rt *_router) taScoreAction(w http.ResponseWriter, r *http.Request, eventID int64) {
	var body struct {
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"bad_json"}`, http.StatusBadRequest)
		return
	}
	err := rt.store.ApplyScoreAction(r.Context(), eventID, chi.URLParam(r, "id"), body.Action)
	switch {
	case errors.Is(err, errTANotFound):
		http.Error(w, `{"error":"match_not_found"}`, http.StatusNotFound)
		return
	case err != nil && (err.Error() == "set_tied" || err.Error() == "unknown_action" || err.Error() == "teams_not_ready"):
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadRequest)
		return
	case err != nil:
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	// Il polling dei tifosi deve vedere il punto SUBITO, non a fine TTL.
	rt.invalidateTournamentScoreCaches(r, eventID, body.Action == "finish" || body.Action == "reopen")
	matches, _ := rt.store.ListTAMatches(r.Context(), eventID)
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "matches": matches})
}

func (rt *_router) taListSponsors(w http.ResponseWriter, r *http.Request, eventID int64) {
	sponsors, err := rt.store.ListTASponsors(r.Context(), eventID)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"sponsors": sponsors})
}

// maxSponsorLogoBytes limita la data-URL del logo sponsor (~500KB di immagine
// dopo l'inflazione base64). Il pannello ridimensiona già a 400px lato client:
// il cap è una difesa server-side contro payload fuori scala.
const maxSponsorLogoBytes = 700 * 1024

var allowedLogoDataPrefixes = []string{
	"data:image/png", "data:image/jpeg", "data:image/jpg", "data:image/webp", "data:image/gif",
}

// sanitizeSponsorLogo valida il logo: accetta un URL http(s) o path assoluto
// (link a immagine esterna) oppure una data-URL immagine (upload salvato inline,
// nessuna dipendenza da storage su disco). Rifiuta data-URL non-immagine o
// troppo pesanti e schemi non previsti (es. javascript:, data:text/html).
func sanitizeSponsorLogo(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", nil
	}
	if strings.HasPrefix(s, "data:") {
		if len(s) > maxSponsorLogoBytes {
			return "", fmt.Errorf("logo_too_large")
		}
		lower := strings.ToLower(s)
		for _, p := range allowedLogoDataPrefixes {
			if strings.HasPrefix(lower, p) {
				return s, nil
			}
		}
		return "", fmt.Errorf("logo_bad_type")
	}
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "/") {
		return s, nil
	}
	return "", fmt.Errorf("logo_bad_url")
}

func (rt *_router) taCreateSponsor(w http.ResponseWriter, r *http.Request, eventID int64) {
	r.Body = http.MaxBytesReader(w, r.Body, maxSponsorLogoBytes+4096)
	var sp TASponsor
	if err := json.NewDecoder(r.Body).Decode(&sp); err != nil || strings.TrimSpace(sp.Name) == "" {
		http.Error(w, `{"error":"bad_input"}`, http.StatusBadRequest)
		return
	}
	logo, err := sanitizeSponsorLogo(sp.LogoURL)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadRequest)
		return
	}
	sp.LogoURL = logo
	id, err := rt.store.CreateTASponsor(r.Context(), eventID, sp)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	rt.invalidateTournamentCaches(r, eventID)
	writeJSON(w, http.StatusCreated, map[string]interface{}{"id": id})
}

func (rt *_router) taDeleteSponsor(w http.ResponseWriter, r *http.Request, eventID int64) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := rt.store.DeleteTASponsor(r.Context(), eventID, id); err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	rt.invalidateTournamentCaches(r, eventID)
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

func (rt *_router) taListShopProducts(w http.ResponseWriter, r *http.Request, eventID int64) {
	products, err := rt.store.ListTAShopProducts(r.Context(), eventID)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"products": products})
}

func sanitizeShopText(raw string, max int) string {
	value := strings.TrimSpace(raw)
	if len([]rune(value)) > max {
		value = string([]rune(value)[:max])
	}
	return value
}

func sanitizeShopExtras(extras []TournamentShopExtra) ([]TournamentShopExtra, error) {
	if len(extras) > 12 {
		extras = extras[:12]
	}
	out := make([]TournamentShopExtra, 0, len(extras))
	for _, extra := range extras {
		title := sanitizeShopText(extra.Title, 80)
		if title == "" && extra.PriceCents == 0 {
			continue
		}
		if title == "" || extra.PriceCents <= 0 || extra.PriceCents > 100000000 {
			return nil, fmt.Errorf("bad_extra")
		}
		out = append(out, TournamentShopExtra{Title: title, PriceCents: extra.PriceCents})
	}
	return out, nil
}

func (rt *_router) taCreateShopProduct(w http.ResponseWriter, r *http.Request, eventID int64) {
	r.Body = http.MaxBytesReader(w, r.Body, maxSponsorLogoBytes+16384)
	var product TournamentShopProduct
	if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
		http.Error(w, `{"error":"bad_json"}`, http.StatusBadRequest)
		return
	}
	image, err := sanitizeSponsorLogo(product.ImageURL)
	if err != nil || image == "" {
		code := "image_required"
		if err != nil {
			code = err.Error()
		}
		http.Error(w, fmt.Sprintf(`{"error":%q}`, code), http.StatusBadRequest)
		return
	}
	if product.PriceCents <= 0 || product.PriceCents > 100000000 {
		http.Error(w, `{"error":"bad_price"}`, http.StatusBadRequest)
		return
	}
	extras, err := sanitizeShopExtras(product.Extras)
	if err != nil {
		http.Error(w, `{"error":"bad_extra"}`, http.StatusBadRequest)
		return
	}
	product.ImageURL = image
	product.Title = sanitizeShopText(product.Title, 120)
	product.Description = sanitizeShopText(product.Description, 600)
	product.Extras = extras
	id, err := rt.store.CreateTAShopProduct(r.Context(), eventID, product)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	rt.invalidateTournamentCaches(r, eventID)
	writeJSON(w, http.StatusCreated, map[string]interface{}{"id": id})
}

func (rt *_router) taDeleteShopProduct(w http.ResponseWriter, r *http.Request, eventID int64) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := rt.store.DeleteTAShopProduct(r.Context(), eventID, id); err != nil {
		if errors.Is(err, errTANotFound) {
			http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	rt.invalidateTournamentCaches(r, eventID)
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

func (rt *_router) taListShopReservations(w http.ResponseWriter, r *http.Request, eventID int64) {
	reservations, err := rt.store.ListTAShopReservations(r.Context(), eventID)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"reservations": reservations})
}

func (rt *_router) taGetSettings(w http.ResponseWriter, r *http.Request, eventID int64) {
	settings, slug, err := rt.store.GetTASettings(r.Context(), eventID)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"settings": settings, "slug": slug})
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (rt *_router) taUpdateSettings(w http.ResponseWriter, r *http.Request, eventID int64) {
	// L'intestazione (logo) può essere una data-URL inline: alza il cap del body.
	r.Body = http.MaxBytesReader(w, r.Body, maxSponsorLogoBytes*2+16384)
	var st TASettings
	if err := json.NewDecoder(r.Body).Decode(&st); err != nil {
		http.Error(w, `{"error":"bad_json"}`, http.StatusBadRequest)
		return
	}
	// Intestazione home tifosi: valida come i loghi (data-URL immagine o URL).
	logo, err := sanitizeSponsorLogo(st.Logo)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}
	st.Logo = logo
	organizerLogo, err := sanitizeSponsorLogo(st.OrganizerLogo)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}
	st.OrganizerLogo = organizerLogo
	// Premi: testo libero per categoria, con limite per campo (anti-abuso).
	st.Prizes = sanitizeTournamentPrizes(st.Prizes)
	// Punti classifica: interi non negativi, con un tetto ragionevole.
	st.PointsPerWin = clampInt(st.PointsPerWin, 0, 100)
	st.PointsPerDraw = clampInt(st.PointsPerDraw, 0, 100)
	st.PointsPerLoss = clampInt(st.PointsPerLoss, 0, 100)
	st.PointsPerTieWin = clampInt(st.PointsPerTieWin, 0, 100)
	st.PointsPerTieLoss = clampInt(st.PointsPerTieLoss, 0, 100)
	// Formula set: solo al meglio dei 3 o dei 5.
	if st.SetsBestOf != 5 {
		st.SetsBestOf = 3
	}
	st.BracketQualifiers = clampInt(st.BracketQualifiers, 1, 8)
	st.StandingsLegendText = sanitizeShopText(st.StandingsLegendText, 300)
	if st.FanLayout != "sunset" {
		st.FanLayout = "classic"
	}
	if err := rt.store.UpdateTASettings(r.Context(), eventID, st); err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	rt.invalidateTournamentCaches(r, eventID)
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}
