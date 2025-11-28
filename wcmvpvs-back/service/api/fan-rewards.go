package api

import (
	"database/sql"
	"errors"
	"net/http"
	"sort"
	"strings"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
)

type fanProfilePayload struct {
	FanID               string   `json:"fan_id"`
	DeviceID            string   `json:"device_id"`
	Email               string   `json:"email"`
	Name                string   `json:"name"`
	AgeRange            string   `json:"age_range"`
	Location            string   `json:"location"`
	AttendanceFrequency string   `json:"attendance_frequency"`
	FavoritePlayer      string   `json:"favorite_player"`
	ContentPreferences  []string `json:"content_preferences"`
	Interests           []string `json:"interests"`
	EventID             int      `json:"event_id"`
	ConsentClub         bool     `json:"consent_club"`
	ConsentSponsors     bool     `json:"consent_sponsors"`
	ConsentAnalytics    bool     `json:"consent_analytics"`
	PolicyVersion       string   `json:"policy_version"`
	Actions             struct {
		Voted           bool `json:"voted"`
		ReactionGame    bool `json:"reaction_game"`
		SurveyCompleted bool `json:"survey_completed"`
		SponsorScan     bool `json:"sponsor_scan"`
	} `json:"actions"`
}

type fanRewardsResponse struct {
	Profile      database.FanProfile       `json:"profile"`
	Badges       []database.MarketingBadge `json:"badges"`
	NextBadge    *database.MarketingBadge  `json:"next_badge,omitempty"`
	Points       int                       `json:"points"`
	PointsToNext int                       `json:"points_to_next,omitempty"`
}

func (rt *_router) getFanRewardsProfile(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	fanID := strings.TrimSpace(r.URL.Query().Get("fan_id"))
	deviceID := rt.deviceIDFromRequest(r)
	email := strings.TrimSpace(r.URL.Query().Get("email"))

	if fanID == "" && deviceID == "" && email == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	profile, err := rt.db.GetFanProfileWithStats(ctx.OrganizationID, fanID, deviceID, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		ctx.Logger.WithError(err).Error("cannot fetch fan profile")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	badges, _ := rt.db.ListMarketingBadges(ctx.OrganizationID)
	response := buildRewardResponse(profile, badges)
	rt.respondWithJSON(w, http.StatusOK, response)
}

func (rt *_router) upsertFanRewardsProfile(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var payload fanProfilePayload
	if err := rt.decodeJSONBody(w, r, &payload); err != nil {
		ctx.Logger.WithError(err).Warn("invalid fan profile payload")
		return
	}

	payload.DeviceID = strings.TrimSpace(payload.DeviceID)
	if payload.DeviceID == "" {
		payload.DeviceID = rt.deviceIDFromRequest(r)
	}

	if payload.FanID == "" && payload.Email == "" && payload.DeviceID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var interests []string
	interests = append(interests, payload.Interests...)
	if payload.FavoritePlayer != "" {
		interests = append(interests, "player:"+payload.FavoritePlayer)
	}
	if len(payload.ContentPreferences) > 0 {
		interests = append(interests, "content:"+strings.Join(payload.ContentPreferences, ";"))
	}

	profile := database.FanProfile{
		OrganizationID:      ctx.OrganizationID,
		EventID:             payload.EventID,
		FanID:               payload.FanID,
		DeviceID:            payload.DeviceID,
		FirstName:           payload.Name,
		Email:               payload.Email,
		AgeRange:            payload.AgeRange,
		Location:            payload.Location,
		AttendanceFrequency: payload.AttendanceFrequency,
		SponsorPreference:   payload.FavoritePlayer,
		Interests:           strings.Join(interests, "|"),
		ContactChannel:      "app",
		PromoOptIn:          payload.ConsentClub || payload.ConsentSponsors,
	}

	// Capture existing profile to avoid double counting
	var existing database.FanProfile
	existing, _ = rt.db.GetFanProfileWithStats(ctx.OrganizationID, payload.FanID, payload.DeviceID, payload.Email)

	saved, err := rt.db.UpsertFanProfile(profile)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot upsert fan profile")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	saved.OrganizationID = ctx.OrganizationID

	// Assign gamification events for new information
	rt.assignProfileGamification(saved, existing, payload)

	// Store consent snapshot if provided
	if payload.ConsentClub || payload.ConsentSponsors || payload.ConsentAnalytics {
		consent := database.FanConsent{
			OrganizationID:   ctx.OrganizationID,
			FanProfileID:     saved.ID,
			PolicyVersion:    strings.TrimSpace(payload.PolicyVersion),
			ConsentClub:      payload.ConsentClub,
			ConsentSponsors:  payload.ConsentSponsors,
			ConsentAnalytics: payload.ConsentAnalytics,
			IP:               rt.getClientIP(r),
		}
		if _, err := rt.db.LogFanConsent(consent); err != nil {
			ctx.Logger.WithError(err).Error("cannot log fan consent")
		}
	}

	updated, err := rt.db.GetFanProfileWithStats(ctx.OrganizationID, saved.FanID, saved.DeviceID, saved.Email)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot refresh profile after upsert")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	badges, _ := rt.db.ListMarketingBadges(ctx.OrganizationID)
	response := buildRewardResponse(updated, badges)
	rt.respondWithJSON(w, http.StatusCreated, response)
}

func (rt *_router) assignProfileGamification(saved database.FanProfile, previous database.FanProfile, payload fanProfilePayload) {
	const baseFieldPoints = 5
	const consentPoints = 8
	const actionVotePoints = 10
	const actionReactionPoints = 8
	const actionSurveyPoints = 6
	const actionSponsorScan = 5

	awardIfNew := func(source string, newValue string, oldValue string) {
		if strings.TrimSpace(newValue) == "" {
			return
		}
		if strings.TrimSpace(oldValue) == strings.TrimSpace(newValue) {
			return
		}
		_ = rt.recordGamificationIfNeeded(saved.OrganizationID, saved.ID, payload.EventID, source, baseFieldPoints)
	}

	awardIfNew("profile:name", payload.Name, previous.FirstName)
	awardIfNew("profile:age_range", payload.AgeRange, previous.AgeRange)
	awardIfNew("profile:location", payload.Location, previous.Location)
	awardIfNew("profile:frequency", payload.AttendanceFrequency, previous.AttendanceFrequency)
	awardIfNew("profile:favorite_player", payload.FavoritePlayer, previous.SponsorPreference)

	if len(payload.ContentPreferences) > 0 && payload.Interests != nil {
		_ = rt.recordGamificationIfNeeded(saved.OrganizationID, saved.ID, payload.EventID, "profile:interests", baseFieldPoints)
	}

	if payload.ConsentClub {
		_ = rt.recordGamificationIfNeeded(saved.OrganizationID, saved.ID, payload.EventID, "consent:club", consentPoints)
	}
	if payload.ConsentSponsors {
		_ = rt.recordGamificationIfNeeded(saved.OrganizationID, saved.ID, payload.EventID, "consent:sponsors", consentPoints)
	}
	if payload.ConsentAnalytics {
		_ = rt.recordGamificationIfNeeded(saved.OrganizationID, saved.ID, payload.EventID, "consent:analytics", consentPoints)
	}

	if payload.Actions.Voted {
		_ = rt.recordGamificationIfNeeded(saved.OrganizationID, saved.ID, payload.EventID, "action:mvp_vote", actionVotePoints)
	}
	if payload.Actions.ReactionGame {
		_ = rt.recordGamificationIfNeeded(saved.OrganizationID, saved.ID, payload.EventID, "action:reaction_game", actionReactionPoints)
	}
	if payload.Actions.SurveyCompleted {
		_ = rt.recordGamificationIfNeeded(saved.OrganizationID, saved.ID, payload.EventID, "action:survey", actionSurveyPoints)
	}
	if payload.Actions.SponsorScan {
		_ = rt.recordGamificationIfNeeded(saved.OrganizationID, saved.ID, payload.EventID, "action:sponsor_scan", actionSponsorScan)
	}
}

func (rt *_router) recordGamificationIfNeeded(organizationID int, profileID int, eventID int, source string, points int) error {
	if profileID == 0 || strings.TrimSpace(source) == "" || points == 0 {
		return nil
	}
	already, err := rt.db.HasGamificationEvent(profileID, source, eventID)
	if err != nil {
		return err
	}
	if already {
		return nil
	}
	return rt.db.RecordGamificationEvent(database.GamificationEvent{
		OrganizationID: organizationID,
		FanProfileID:   profileID,
		EventID:        eventID,
		Source:         source,
		Points:         points,
	})
}

func buildRewardResponse(profile database.FanProfile, badges []database.MarketingBadge) fanRewardsResponse {
	response := fanRewardsResponse{Profile: profile, Badges: badges, Points: profile.Points}
	sort.SliceStable(badges, func(i, j int) bool { return badges[i].Threshold < badges[j].Threshold })
	for _, b := range badges {
		if profile.Points >= b.Threshold {
			response.Profile.Badges = append(response.Profile.Badges, b.Name)
			continue
		}
		if response.NextBadge == nil {
			badge := b
			response.NextBadge = &badge
			response.PointsToNext = b.Threshold - profile.Points
		}
	}
	return response
}
