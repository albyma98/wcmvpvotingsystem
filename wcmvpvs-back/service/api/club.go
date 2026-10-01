package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
)

const clubCookieName = "club_admin_session"
const clubSessionTTL = 14 * 24 * time.Hour

func registerClubRoutes(rt *_router) {
	if err := rt.store.EnsureClubTables(); err != nil {
		panic("club tables: " + err.Error())
	}

	// Provisioning centrale.
	rt.router.Get("/admin/master/clubs", rt.wrapAdmin(rt.listMasterClubs))
	rt.router.Post("/admin/master/clubs", rt.wrapAdmin(rt.createMasterClub))
	rt.router.Put("/admin/master/clubs/{id}/password", rt.wrapAdmin(rt.resetMasterClubPassword))
	rt.router.Delete("/admin/master/clubs/{id}", rt.wrapAdmin(rt.deleteMasterClub))

	// Sito pubblico annuale.
	rt.router.Get("/v1/clubs/{slug}/home", rt.clubPublicHome)
	rt.router.Get("/v1/clubs/{slug}/teams/{teamSlug}", rt.clubPublicTeam)

	// CMS Club, completamente separato dall'admin Live.
	rt.router.Post("/v1/club-admin/{slug}/login", rt.clubAdminLogin)
	rt.router.Post("/v1/club-admin/{slug}/logout", rt.clubAdminLogout)
	rt.router.Get("/v1/club-admin/{slug}/overview", rt.wrapClubAdmin(rt.clubAdminOverview))
	rt.router.Put("/v1/club-admin/{slug}/settings", rt.wrapClubAdmin(rt.clubAdminUpdateSettings))
	rt.router.Get("/v1/club-admin/{slug}/teams", rt.wrapClubAdmin(rt.clubAdminTeams))
	rt.router.Post("/v1/club-admin/{slug}/teams", rt.wrapClubAdmin(rt.clubAdminSaveTeam))
	rt.router.Put("/v1/club-admin/{slug}/teams/{id}", rt.wrapClubAdmin(rt.clubAdminSaveTeam))
	rt.router.Delete("/v1/club-admin/{slug}/teams/{id}", rt.wrapClubAdmin(rt.clubAdminDeleteTeam))
	rt.router.Put("/v1/club-admin/{slug}/teams/{id}/players", rt.wrapClubAdmin(rt.clubAdminSavePlayers))
	rt.router.Get("/v1/club-admin/{slug}/posts", rt.wrapClubAdmin(rt.clubAdminPosts))
	rt.router.Post("/v1/club-admin/{slug}/posts", rt.wrapClubAdmin(rt.clubAdminSavePost))
	rt.router.Put("/v1/club-admin/{slug}/posts/{id}", rt.wrapClubAdmin(rt.clubAdminSavePost))
	rt.router.Delete("/v1/club-admin/{slug}/posts/{id}", rt.wrapClubAdmin(rt.clubAdminDeletePost))
	rt.router.Get("/v1/club-admin/{slug}/gallery", rt.wrapClubAdmin(rt.clubAdminGallery))
	rt.router.Post("/v1/club-admin/{slug}/gallery", rt.wrapClubAdmin(rt.clubAdminSaveGallery))
	rt.router.Put("/v1/club-admin/{slug}/gallery/{id}", rt.wrapClubAdmin(rt.clubAdminSaveGallery))
	rt.router.Delete("/v1/club-admin/{slug}/gallery/{id}", rt.wrapClubAdmin(rt.clubAdminDeleteGallery))
	rt.router.Get("/v1/club-admin/{slug}/matches", rt.wrapClubAdmin(rt.clubAdminMatches))
	rt.router.Post("/v1/club-admin/{slug}/matches", rt.wrapClubAdmin(rt.clubAdminSaveMatch))
	rt.router.Put("/v1/club-admin/{slug}/matches/{id}", rt.wrapClubAdmin(rt.clubAdminSaveMatch))
	rt.router.Delete("/v1/club-admin/{slug}/matches/{id}", rt.wrapClubAdmin(rt.clubAdminDeleteMatch))
}

func (rt *_router) listMasterClubs(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !rt.ensureSuperAdmin(w, ctx) {
		return
	}
	items, err := rt.store.ListClubSites(r.Context())
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	writeJSON(w, 200, map[string]interface{}{"clubs": items})
}

type masterClubInput struct {
	OrganizationID int64    `json:"organizationId"`
	Name           string   `json:"name"`
	Slug           string   `json:"slug"`
	City           string   `json:"city"`
	Tagline        string   `json:"tagline"`
	LogoURL        string   `json:"logoUrl"`
	HeroImageURL   string   `json:"heroImageUrl"`
	PrimaryColor   string   `json:"primaryColor"`
	SecondaryColor string   `json:"secondaryColor"`
	SeasonLabel    string   `json:"seasonLabel"`
	Categories     []string `json:"categories"`
}

func (rt *_router) createMasterClub(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !rt.ensureSuperAdmin(w, ctx) {
		return
	}
	var in masterClubInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		http.Error(w, `{"error":"bad_json"}`, 400)
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Slug = slugifyTournament(firstNonEmpty(in.Slug, in.Name))
	if in.Name == "" || in.Slug == "" {
		http.Error(w, `{"error":"name_required"}`, 400)
		return
	}
	if _, err := rt.store.ClubSiteBySlug(r.Context(), in.Slug); err == nil {
		http.Error(w, `{"error":"slug_taken"}`, 409)
		return
	}

	orgID := in.OrganizationID
	if orgID <= 0 {
		org, err := rt.db.CreateOrganization(database.Organization{Name: in.Name, Slug: in.Slug, City: strings.TrimSpace(in.City), LogoURL: strings.TrimSpace(in.LogoURL), IsActive: true, RosterSchema: 13, BarEnabled: false})
		if err != nil {
			ctx.Logger.WithError(err).Error("cannot create club organization")
			http.Error(w, `{"error":"organization_create_failed"}`, 409)
			return
		}
		orgID = int64(org.ID)
	}
	password, err := randomPassword(9)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	if len(in.Categories) == 0 {
		in.Categories = []string{"Prima Divisione Maschile", "Prima Divisione Femminile", "Giovanile 1", "Giovanile 2"}
	}
	site := ClubSite{OrganizationID: orgID, Slug: in.Slug, Name: in.Name, City: strings.TrimSpace(in.City), Tagline: strings.TrimSpace(in.Tagline), LogoURL: strings.TrimSpace(in.LogoURL), HeroImageURL: strings.TrimSpace(in.HeroImageURL), PrimaryColor: cleanColor(in.PrimaryColor, "#172554"), SecondaryColor: cleanColor(in.SecondaryColor, "#f59e0b"), SeasonLabel: strings.TrimSpace(in.SeasonLabel), Active: true}
	username := "club-" + in.Slug
	id, err := rt.store.CreateClubSite(r.Context(), site, in.Categories, username, string(hash))
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot create club site")
		http.Error(w, `{"error":"create_failed"}`, 409)
		return
	}
	writeJSON(w, 201, map[string]interface{}{"id": id, "slug": in.Slug, "publicPath": "/club/" + in.Slug, "adminPath": "/club-admin/" + in.Slug, "adminUsername": username, "adminPassword": password})
}

func cleanColor(value, fallback string) string {
	value = strings.TrimSpace(value)
	if len(value) == 7 && strings.HasPrefix(value, "#") {
		return value
	}
	return fallback
}

func (rt *_router) resetMasterClubPassword(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !rt.ensureSuperAdmin(w, ctx) {
		return
	}
	id, err := parseClubID(r, "id")
	if err != nil {
		http.Error(w, `{"error":"bad_id"}`, 400)
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	password := strings.TrimSpace(body.Password)
	if password == "" {
		password, err = randomPassword(9)
	} else if len(password) < 6 {
		http.Error(w, `{"error":"password_too_short"}`, 400)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	username, err := rt.store.SetClubAdminPassword(r.Context(), id, string(hash))
	if errors.Is(err, errClubNotFound) {
		http.Error(w, `{"error":"club_not_found"}`, 404)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	writeJSON(w, 200, map[string]interface{}{"adminUsername": username, "adminPassword": password})
}
func (rt *_router) deleteMasterClub(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !rt.ensureSuperAdmin(w, ctx) {
		return
	}
	id, err := parseClubID(r, "id")
	if err != nil {
		http.Error(w, `{"error":"bad_id"}`, 400)
		return
	}
	if err = rt.store.DeleteClubSite(r.Context(), id); errors.Is(err, errClubNotFound) {
		http.Error(w, `{"error":"club_not_found"}`, 404)
		return
	} else if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (rt *_router) clubPublicHome(w http.ResponseWriter, r *http.Request) {
	site, err := rt.store.ClubSiteBySlug(r.Context(), chi.URLParam(r, "slug"))
	if err != nil || !site.Active {
		http.Error(w, `{"error":"club_not_found"}`, 404)
		return
	}
	teams, _ := rt.store.ListClubTeams(r.Context(), site.ID, true)
	posts, _ := rt.store.ListClubPosts(r.Context(), site.ID, 0, true)
	gallery, _ := rt.store.ListClubGallery(r.Context(), site.ID, 0, true)
	matches, _ := rt.store.ListClubMatches(r.Context(), site.ID, 0)
	if len(posts) > 12 {
		posts = posts[:12]
	}
	if len(gallery) > 16 {
		gallery = gallery[:16]
	}
	writeJSON(w, 200, map[string]interface{}{"club": site, "teams": teams, "posts": posts, "gallery": gallery, "matches": matches})
}
func (rt *_router) clubPublicTeam(w http.ResponseWriter, r *http.Request) {
	site, err := rt.store.ClubSiteBySlug(r.Context(), chi.URLParam(r, "slug"))
	if err != nil || !site.Active {
		http.Error(w, `{"error":"club_not_found"}`, 404)
		return
	}
	team, err := rt.store.ClubTeamBySlug(r.Context(), site.ID, chi.URLParam(r, "teamSlug"))
	if err != nil || !team.Active {
		http.Error(w, `{"error":"team_not_found"}`, 404)
		return
	}
	players, _ := rt.store.ListClubPlayers(r.Context(), site.ID, team.ID)
	posts, _ := rt.store.ListClubPosts(r.Context(), site.ID, team.ID, true)
	gallery, _ := rt.store.ListClubGallery(r.Context(), site.ID, team.ID, true)
	matches, _ := rt.store.ListClubMatches(r.Context(), site.ID, team.ID)
	writeJSON(w, 200, map[string]interface{}{"club": site, "team": team, "players": players, "posts": posts, "gallery": gallery, "matches": matches})
}

func setClubCookie(w http.ResponseWriter, r *http.Request, token string, maxAge int) {
	secure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
	http.SetCookie(w, &http.Cookie{Name: clubCookieName, Value: token, Path: "/", MaxAge: maxAge, HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode})
}
func (rt *_router) clubAdminLogin(w http.ResponseWriter, r *http.Request) {
	site, err := rt.store.ClubSiteBySlug(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		http.Error(w, `{"error":"club_not_found"}`, 404)
		return
	}
	var b struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if json.NewDecoder(r.Body).Decode(&b) != nil {
		http.Error(w, `{"error":"bad_json"}`, 400)
		return
	}
	if !rt.allowLoginRequest(b.Username, clientIPFromRequest(r)) {
		http.Error(w, `{"error":"rate_limited"}`, http.StatusTooManyRequests)
		return
	}
	a, findErr := rt.store.ClubAdminByUsername(r.Context(), b.Username)
	dummy := "$2a$10$7EqJtq98hPqEX7fNZaFWoOhi5B0G1Zt0P0mBz0m0m0m0m0m0m0m"
	hash := dummy
	if findErr == nil {
		hash = a.PasswordHash
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(b.Password)) != nil || findErr != nil || a.SiteID != site.ID {
		http.Error(w, `{"error":"invalid_credentials"}`, 401)
		return
	}
	token, err := rt.store.CreateClubSession(r.Context(), a.ID, site.ID, clubSessionTTL)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	setClubCookie(w, r, token, int(clubSessionTTL.Seconds()))
	writeJSON(w, 200, map[string]interface{}{"ok": true, "slug": site.Slug})
}
func (rt *_router) clubAdminLogout(w http.ResponseWriter, r *http.Request) {
	if c, e := r.Cookie(clubCookieName); e == nil {
		rt.store.DeleteClubSession(r.Context(), c.Value)
	}
	setClubCookie(w, r, "", -1)
	writeJSON(w, 200, map[string]bool{"ok": true})
}

type clubAdminHandler func(http.ResponseWriter, *http.Request, int64)

func (rt *_router) wrapClubAdmin(fn clubAdminHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := ""
		if c, e := r.Cookie(clubCookieName); e == nil {
			token = c.Value
		}
		if token == "" {
			token = parseBearerToken(r.Header.Get("Authorization"))
		}
		siteID, ok := rt.store.ClubSession(r.Context(), token)
		if !ok {
			w.WriteHeader(401)
			return
		}
		site, e := rt.store.ClubSiteBySlug(r.Context(), chi.URLParam(r, "slug"))
		if e != nil || site.ID != siteID {
			w.WriteHeader(403)
			return
		}
		fn(w, r, siteID)
	}
}

func (rt *_router) clubAdminOverview(w http.ResponseWriter, r *http.Request, siteID int64) {
	site, _ := rt.store.ClubSiteByID(r.Context(), siteID)
	teams, _ := rt.store.ListClubTeams(r.Context(), siteID, false)
	posts, _ := rt.store.ListClubPosts(r.Context(), siteID, 0, false)
	gallery, _ := rt.store.ListClubGallery(r.Context(), siteID, 0, false)
	matches, _ := rt.store.ListClubMatches(r.Context(), siteID, 0)
	playerCount := 0
	for i := range teams {
		players, _ := rt.store.ListClubPlayers(r.Context(), siteID, teams[i].ID)
		playerCount += len(players)
	}
	writeJSON(w, 200, map[string]interface{}{"club": site, "teams": teams, "posts": posts, "gallery": gallery, "matches": matches, "counts": map[string]int{"teams": len(teams), "players": playerCount, "posts": len(posts), "photos": len(gallery), "matches": len(matches)}})
}
func (rt *_router) clubAdminUpdateSettings(w http.ResponseWriter, r *http.Request, siteID int64) {
	site, err := rt.store.ClubSiteByID(r.Context(), siteID)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, 404)
		return
	}
	if json.NewDecoder(r.Body).Decode(&site) != nil {
		http.Error(w, `{"error":"bad_json"}`, 400)
		return
	}
	site.ID = siteID
	site.PrimaryColor = cleanColor(site.PrimaryColor, "#172554")
	site.SecondaryColor = cleanColor(site.SecondaryColor, "#f59e0b")
	if strings.TrimSpace(site.Name) == "" {
		http.Error(w, `{"error":"name_required"}`, 400)
		return
	}
	if err = rt.store.UpdateClubSite(r.Context(), site); err != nil {
		http.Error(w, `{"error":"save_failed"}`, 500)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func parseClubID(r *http.Request, key string) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, key), 10, 64)
}
func (rt *_router) clubAdminTeams(w http.ResponseWriter, r *http.Request, siteID int64) {
	items, _ := rt.store.ListClubTeams(r.Context(), siteID, false)
	writeJSON(w, 200, map[string]interface{}{"teams": items, "players": clubPlayersMap(rt, r, siteID, items)})
}
func clubPlayersMap(rt *_router, r *http.Request, siteID int64, teams []ClubTeam) map[string][]ClubPlayer {
	out := map[string][]ClubPlayer{}
	for _, t := range teams {
		p, _ := rt.store.ListClubPlayers(r.Context(), siteID, t.ID)
		out[strconv.FormatInt(t.ID, 10)] = p
	}
	return out
}
func (rt *_router) clubAdminSaveTeam(w http.ResponseWriter, r *http.Request, siteID int64) {
	var v ClubTeam
	if json.NewDecoder(r.Body).Decode(&v) != nil {
		http.Error(w, `{"error":"bad_json"}`, 400)
		return
	}
	if raw := chi.URLParam(r, "id"); raw != "" {
		v.ID, _ = strconv.ParseInt(raw, 10, 64)
	}
	if strings.TrimSpace(v.Name) == "" {
		http.Error(w, `{"error":"name_required"}`, 400)
		return
	}
	if v.ID == 0 && !v.Active {
		v.Active = true
	}
	id, err := rt.store.SaveClubTeam(r.Context(), siteID, v)
	if err != nil {
		http.Error(w, `{"error":"save_failed"}`, 409)
		return
	}
	writeJSON(w, 200, map[string]interface{}{"id": id, "ok": true})
}
func (rt *_router) clubAdminDeleteTeam(w http.ResponseWriter, r *http.Request, siteID int64) {
	id, e := parseClubID(r, "id")
	if e != nil {
		http.Error(w, `{"error":"bad_id"}`, 400)
		return
	}
	e = rt.store.DeleteClubTeam(r.Context(), siteID, id)
	if e != nil {
		http.Error(w, `{"error":"delete_failed"}`, 500)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}
func (rt *_router) clubAdminSavePlayers(w http.ResponseWriter, r *http.Request, siteID int64) {
	id, e := parseClubID(r, "id")
	if e != nil {
		http.Error(w, `{"error":"bad_id"}`, 400)
		return
	}
	var b struct {
		Players []ClubPlayer `json:"players"`
	}
	if json.NewDecoder(r.Body).Decode(&b) != nil {
		http.Error(w, `{"error":"bad_json"}`, 400)
		return
	}
	if e = rt.store.ReplaceClubPlayers(r.Context(), siteID, id, b.Players); e != nil {
		http.Error(w, `{"error":"save_failed"}`, 500)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (rt *_router) clubAdminPosts(w http.ResponseWriter, r *http.Request, siteID int64) {
	v, _ := rt.store.ListClubPosts(r.Context(), siteID, 0, false)
	writeJSON(w, 200, map[string]interface{}{"posts": v})
}
func (rt *_router) clubAdminSavePost(w http.ResponseWriter, r *http.Request, siteID int64) {
	var v ClubPost
	if json.NewDecoder(r.Body).Decode(&v) != nil {
		http.Error(w, `{"error":"bad_json"}`, 400)
		return
	}
	if raw := chi.URLParam(r, "id"); raw != "" {
		v.ID, _ = strconv.ParseInt(raw, 10, 64)
	}
	if strings.TrimSpace(v.Title) == "" {
		http.Error(w, `{"error":"title_required"}`, 400)
		return
	}
	if v.Kind == "" {
		v.Kind = "announcement"
	}
	if v.ID == 0 && !v.Active {
		v.Active = true
	}
	id, e := rt.store.SaveClubPost(r.Context(), siteID, v)
	if e != nil {
		http.Error(w, `{"error":"save_failed"}`, 500)
		return
	}
	writeJSON(w, 200, map[string]interface{}{"id": id, "ok": true})
}
func (rt *_router) clubAdminDeletePost(w http.ResponseWriter, r *http.Request, siteID int64) {
	rt.clubAdminDeleteRecord(w, r, siteID, "club_posts")
}
func (rt *_router) clubAdminGallery(w http.ResponseWriter, r *http.Request, siteID int64) {
	v, _ := rt.store.ListClubGallery(r.Context(), siteID, 0, false)
	writeJSON(w, 200, map[string]interface{}{"gallery": v})
}
func (rt *_router) clubAdminSaveGallery(w http.ResponseWriter, r *http.Request, siteID int64) {
	var v ClubGalleryItem
	if json.NewDecoder(r.Body).Decode(&v) != nil {
		http.Error(w, `{"error":"bad_json"}`, 400)
		return
	}
	if raw := chi.URLParam(r, "id"); raw != "" {
		v.ID, _ = strconv.ParseInt(raw, 10, 64)
	}
	if strings.TrimSpace(v.ImageURL) == "" {
		http.Error(w, `{"error":"image_required"}`, 400)
		return
	}
	if v.ID == 0 && !v.Active {
		v.Active = true
	}
	id, e := rt.store.SaveClubGallery(r.Context(), siteID, v)
	if e != nil {
		http.Error(w, `{"error":"save_failed"}`, 500)
		return
	}
	writeJSON(w, 200, map[string]interface{}{"id": id, "ok": true})
}
func (rt *_router) clubAdminDeleteGallery(w http.ResponseWriter, r *http.Request, siteID int64) {
	rt.clubAdminDeleteRecord(w, r, siteID, "club_gallery")
}
func (rt *_router) clubAdminMatches(w http.ResponseWriter, r *http.Request, siteID int64) {
	v, _ := rt.store.ListClubMatches(r.Context(), siteID, 0)
	writeJSON(w, 200, map[string]interface{}{"matches": v})
}
func (rt *_router) clubAdminSaveMatch(w http.ResponseWriter, r *http.Request, siteID int64) {
	var v ClubMatch
	if json.NewDecoder(r.Body).Decode(&v) != nil {
		http.Error(w, `{"error":"bad_json"}`, 400)
		return
	}
	if raw := chi.URLParam(r, "id"); raw != "" {
		v.ID, _ = strconv.ParseInt(raw, 10, 64)
	}
	if v.TeamID <= 0 || strings.TrimSpace(v.Opponent) == "" || strings.TrimSpace(v.ScheduledAt) == "" {
		http.Error(w, `{"error":"required_fields"}`, 400)
		return
	}
	id, e := rt.store.SaveClubMatch(r.Context(), siteID, v)
	if e != nil {
		http.Error(w, `{"error":"save_failed"}`, 500)
		return
	}
	writeJSON(w, 200, map[string]interface{}{"id": id, "ok": true})
}
func (rt *_router) clubAdminDeleteMatch(w http.ResponseWriter, r *http.Request, siteID int64) {
	rt.clubAdminDeleteRecord(w, r, siteID, "club_matches")
}
func (rt *_router) clubAdminDeleteRecord(w http.ResponseWriter, r *http.Request, siteID int64, table string) {
	id, e := parseClubID(r, "id")
	if e != nil {
		http.Error(w, `{"error":"bad_id"}`, 400)
		return
	}
	if e = rt.store.DeleteClubRecord(r.Context(), siteID, table, id); e != nil {
		http.Error(w, `{"error":"delete_failed"}`, 500)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}
