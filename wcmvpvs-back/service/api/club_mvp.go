package api

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

const clubMVPCookie = "club_mvp_voter"

func registerClubMVPRoutes(rt *_router) {
	rt.clubMVPUpdates = newSSEHub()
	rt.router.Get("/v1/clubs/{slug}/mvp", rt.clubPublicMVP)
	rt.router.Get("/v1/clubs/{slug}/matches/{id}/mvp", rt.clubPublicMVP)
	rt.router.Post("/v1/clubs/{slug}/matches/{id}/mvp/vote", rt.clubPublicMVPVote)
	rt.router.Get("/v1/club-admin/{slug}/matches/{id}/mvp", rt.wrapClubAdmin(rt.clubAdminMVP))
	rt.router.Put("/v1/club-admin/{slug}/matches/{id}/mvp", rt.wrapClubAdmin(rt.clubAdminMVP))
	rt.router.Get("/v1/club-admin/{slug}/mvp/stream", rt.wrapClubAdmin(rt.clubAdminMVPStream))
}

func (rt *_router) clubMVPPublicSite(r *http.Request) (clubMVPBrand, error) {
	slug := chi.URLParam(r, "slug")
	if host, err := requestClubHostname(r); err == nil && rt.clubDomains != nil && !rt.clubDomains.isReserved(host) {
		resolved, _, err := rt.store.ResolveClubDomain(r.Context(), host)
		if !clubDomainNotFound(err) && err != nil {
			return clubMVPBrand{}, err
		}
		if err == nil && resolved != slug {
			return clubMVPBrand{}, errClubNotFound
		}
	}
	var site clubMVPBrand
	err := rt.store.db.QueryRowContext(r.Context(), `SELECT id,name,logo_url,primary_color,secondary_color,title_font,body_font
		FROM club_sites WHERE slug=? AND active=1`, slug).
		Scan(&site.ID, &site.Name, &site.LogoURL, &site.PrimaryColor, &site.SecondaryColor, &site.TitleFont, &site.BodyFont)
	if errors.Is(err, sql.ErrNoRows) {
		err = errClubNotFound
	}
	return site, err
}

// Match Day's caller-supplied device ID is unsuitable for an anonymous ballot.
// A persistent server signature prevents submitting a fabricated cookie value.
func (rt *_router) clubMVPIdentity(r *http.Request) (key, hash []byte, err error) {
	if err := rt.store.db.QueryRowContext(r.Context(), `SELECT secret FROM club_mvp_cookie_key WHERE id=1`).Scan(&key); err != nil {
		return nil, nil, err
	}
	cookie, err := r.Cookie(clubMVPCookie)
	if err != nil {
		return key, nil, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil || len(raw) != 2*sha256.Size {
		return key, nil, nil
	}
	if !hmac.Equal(raw[sha256.Size:], signClubMVPIdentity(key, raw[:sha256.Size])) {
		return key, nil, nil
	}
	digest := sha256.Sum256(raw[:sha256.Size])
	return key, digest[:], nil
}

func signClubMVPIdentity(key, id []byte) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(id)
	return mac.Sum(nil)
}

func issueClubMVPIdentity(w http.ResponseWriter, r *http.Request, key []byte) error {
	var id [sha256.Size]byte
	if _, err := rand.Read(id[:]); err != nil {
		return err
	}
	token := append(id[:], signClubMVPIdentity(key, id[:])...)
	http.SetCookie(w, &http.Cookie{
		Name: clubMVPCookie, Value: base64.RawURLEncoding.EncodeToString(token),
		Path: "/api/v1/clubs/", MaxAge: int((365 * 24 * time.Hour).Seconds()),
		HttpOnly: true, SameSite: http.SameSiteStrictMode,
		Secure: r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
	})
	return nil
}

func (rt *_router) clubPublicMVP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	var matchID int64
	if chi.URLParam(r, "id") != "" {
		var err error
		if matchID, err = parseClubID(r, "id"); err != nil || matchID <= 0 {
			clubMVPError(w, errClubMVPClosed)
			return
		}
	}
	site, err := rt.clubMVPPublicSite(r)
	if err != nil {
		clubMVPError(w, err)
		return
	}
	key, hash, err := rt.clubMVPIdentity(r)
	if err != nil {
		clubMVPError(w, err)
		return
	}
	data, err := rt.store.clubMVPPublic(r.Context(), site, matchID, hash)
	if err == nil && len(hash) == 0 {
		err = issueClubMVPIdentity(w, r, key)
	}
	if err != nil {
		clubMVPError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (rt *_router) clubPublicMVPVote(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	matchID, err := parseClubID(r, "id")
	var vote struct {
		PlayerID int64 `json:"playerId"`
	}
	mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || matchID <= 0 || mediaType != "application/json" ||
		json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&vote) != nil || vote.PlayerID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_vote"})
		return
	}
	site, err := rt.clubMVPPublicSite(r)
	if err != nil {
		clubMVPError(w, err)
		return
	}
	_, hash, err := rt.clubMVPIdentity(r)
	if err != nil {
		clubMVPError(w, err)
		return
	}
	if len(hash) == 0 {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "voter_cookie_required"})
		return
	}
	scope := "club-mvp:" + strconv.FormatInt(matchID, 10) + ":"
	if limited, _ := rt.shouldThrottleVoteAttempt(scope+base64.RawURLEncoding.EncodeToString(hash), scope+clientIPFromRequest(r), time.Now()); limited {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate_limited"})
		return
	}
	player, duplicate, err := rt.store.castClubMVPVote(r.Context(), site.ID, matchID, vote.PlayerID, hash)
	if err != nil {
		clubMVPError(w, err)
		return
	}
	if duplicate {
		writeJSON(w, http.StatusConflict, map[string]interface{}{"error": "already_voted", "voted": player})
		return
	}
	rt.clubMVPUpdates.BroadcastThrottled(int(site.ID), time.Second)
	writeJSON(w, http.StatusCreated, map[string]interface{}{"voted": player})
}

func (rt *_router) clubAdminMVP(w http.ResponseWriter, r *http.Request, siteID int64) {
	w.Header().Set("Cache-Control", "no-store")
	matchID, err := parseClubID(r, "id")
	if err != nil || matchID <= 0 {
		clubMVPError(w, errClubNotFound)
		return
	}
	if r.Method == http.MethodPut {
		var change clubMVPChange
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384)).Decode(&change) != nil ||
			(change.PlayerIDs == nil && change.Enabled == nil && change.QRTarget == nil) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_configuration"})
			return
		}
		if err := rt.store.updateClubMVP(r.Context(), siteID, matchID, change); err != nil {
			clubMVPError(w, err)
			return
		}
		rt.resetClubMatches(siteID)
	}
	data, err := rt.store.clubMVPAdmin(r.Context(), siteID, matchID)
	if err != nil {
		clubMVPError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (rt *_router) resetClubMatches(siteID int64) {
	rt.clubMatches.reset(siteID)
	if rt.clubMVPUpdates != nil {
		rt.clubMVPUpdates.Broadcast(int(siteID))
	}
}

func (rt *_router) clubAdminMVPStream(w http.ResponseWriter, r *http.Request, siteID int64) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	clearSSEDeadline(w)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Accel-Buffering", "no")
	ch, unsubscribe := rt.clubMVPUpdates.Subscribe(int(siteID))
	defer unsubscribe()
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()
	const update = "data: {}\n\n"
	if _, err := fmt.Fprint(w, update); err != nil {
		return
	}
	flusher.Flush()
	for {
		message := update
		select {
		case <-r.Context().Done():
			return
		case <-ch:
		case <-ticker.C:
			message = ": keepalive\n\n"
		}
		if _, err := fmt.Fprint(w, message); err != nil {
			return
		}
		flusher.Flush()
	}
}

func clubMVPError(w http.ResponseWriter, err error) {
	status, code := http.StatusInternalServerError, "mvp_unavailable"
	switch {
	case errors.Is(err, errClubNotFound):
		status, code = http.StatusNotFound, "match_not_found"
	case errors.Is(err, errClubMVPClosed):
		status, code = http.StatusConflict, errClubMVPClosed.Error()
	case errors.Is(err, errClubMVPPlayer):
		status, code = http.StatusUnprocessableEntity, errClubMVPPlayer.Error()
	case errors.Is(err, errClubMVPLocked):
		status, code = http.StatusConflict, errClubMVPLocked.Error()
	case errors.Is(err, errClubMVPEmpty):
		status, code = http.StatusUnprocessableEntity, errClubMVPEmpty.Error()
	}
	writeJSON(w, status, map[string]string{"error": code})
}
