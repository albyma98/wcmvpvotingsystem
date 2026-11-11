package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
)

// Public branding is available even when the organization has no active event.
func (rt *_router) getOrganizationTheme(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if ctx.OrganizationID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	org, err := rt.db.GetOrganization(ctx.OrganizationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			ctx.Logger.WithError(err).Error("cannot load organization theme")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	walls, err := rt.db.GetSponsorPlacements(r.Context(), ctx.OrganizationID, database.GameWallPlacements, true)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot load public game wall sponsors")
	}
	postVote, err := rt.db.GetSponsorPlacements(r.Context(), ctx.OrganizationID, database.PostVotePlacements, true)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot load public post-vote sponsors")
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(struct {
		Name             string         `json:"name"`
		LogoURL          string         `json:"logo_url"`
		Sport            string         `json:"sport"`
		PrimaryColor     string         `json:"primary_color"`
		SecondaryColor   string         `json:"secondary_color"`
		TertiaryColor    string         `json:"tertiary_color"`
		GameWallSponsors map[string]int `json:"game_wall_sponsors,omitempty"`
		PostVoteSponsors map[string]int `json:"post_vote_sponsors,omitempty"`
	}{
		Name: org.Name, LogoURL: org.LogoURL, Sport: org.Sport,
		PrimaryColor: org.PrimaryColor, SecondaryColor: org.SecondaryColor, TertiaryColor: org.TertiaryColor,
		GameWallSponsors: walls, PostVoteSponsors: postVote,
	})
}
