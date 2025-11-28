package api

import (
	"encoding/csv"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
)

type marketingOverviewResponse struct {
	ProfiledFans         int                          `json:"profiled_fans"`
	TotalVoters          int                          `json:"total_voters"`
	AgeDistribution      map[string]int               `json:"age_distribution"`
	LocationDistribution map[string]int               `json:"location_distribution"`
	AttendanceBreakdown  map[string]int               `json:"attendance_breakdown"`
	SponsorPreferences   map[string]int               `json:"sponsor_preferences"`
	Gamification         database.GamificationSummary `json:"gamification"`
	Consent              database.ConsentBreakdown    `json:"consent"`
	Badges               []database.MarketingBadge    `json:"badges"`
	LatestPolicies       []database.PrivacyPolicy     `json:"privacy_policies"`
	SegmentParticipation map[string]int               `json:"segment_participation"`
}

type marketingSegmentFilter struct {
	AgeRange          string
	Location          string
	SponsorPreference string
	ContactChannel    string
	MinPoints         int
}

func (rt *_router) getMarketingOverview(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	orgID := ctx.OrganizationID
	profiles, err := rt.db.ListFanProfilesWithStats(orgID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list fan profiles")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	votes, err := rt.db.ListVotesByOrganization(orgID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list votes for marketing overview")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	gamificationEvents, err := rt.db.ListGamificationEvents(orgID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list gamification events")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	consentBreakdown, err := rt.db.GetConsentBreakdown(orgID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot read consents")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	badges, err := rt.db.ListMarketingBadges(orgID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list marketing badges")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	policies, _ := rt.db.ListPrivacyPolicies(orgID)

	overview := marketingOverviewResponse{
		ProfiledFans:         len(profiles),
		TotalVoters:          len(votes),
		AgeDistribution:      map[string]int{},
		LocationDistribution: map[string]int{},
		AttendanceBreakdown:  map[string]int{},
		SponsorPreferences:   map[string]int{},
		Gamification:         database.BuildGamificationSummary(profiles, gamificationEvents),
		Consent:              consentBreakdown,
		Badges:               badges,
		LatestPolicies:       policies,
		SegmentParticipation: map[string]int{},
	}

	for _, profile := range profiles {
		if profile.AgeRange != "" {
			overview.AgeDistribution[profile.AgeRange]++
		}
		if profile.Location != "" {
			overview.LocationDistribution[profile.Location]++
		}
		if profile.AttendanceFrequency != "" {
			overview.AttendanceBreakdown[profile.AttendanceFrequency]++
		}
		if profile.SponsorPreference != "" {
			overview.SponsorPreferences[profile.SponsorPreference]++
		}
		if profile.Points > 0 {
			overview.SegmentParticipation["gamified"]++
		}
		if profile.PromoOptIn {
			overview.SegmentParticipation["opt_in"]++
		}
	}

	rt.respondWithJSON(w, http.StatusOK, overview)
}

func (rt *_router) listMarketingProfiles(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	orgID := ctx.OrganizationID
	profiles, err := rt.db.ListFanProfilesWithStats(orgID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list fan profiles")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	filter := marketingSegmentFilter{
		AgeRange:          strings.TrimSpace(r.URL.Query().Get("age_range")),
		Location:          strings.TrimSpace(r.URL.Query().Get("location")),
		SponsorPreference: strings.TrimSpace(r.URL.Query().Get("sponsor_preference")),
		ContactChannel:    strings.TrimSpace(r.URL.Query().Get("contact_channel")),
	}
	if minPoints, err := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("min_points"))); err == nil && minPoints > 0 {
		filter.MinPoints = minPoints
	}

	filtered := make([]database.FanProfile, 0, len(profiles))
	for _, profile := range profiles {
		if filter.AgeRange != "" && profile.AgeRange != filter.AgeRange {
			continue
		}
		if filter.Location != "" && !strings.EqualFold(profile.Location, filter.Location) {
			continue
		}
		if filter.SponsorPreference != "" && !strings.EqualFold(profile.SponsorPreference, filter.SponsorPreference) {
			continue
		}
		if filter.ContactChannel != "" && !strings.EqualFold(profile.ContactChannel, filter.ContactChannel) {
			continue
		}
		if filter.MinPoints > 0 && profile.Points < filter.MinPoints {
			continue
		}
		filtered = append(filtered, profile)
	}

	if strings.EqualFold(r.URL.Query().Get("format"), "csv") {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=segment.csv")
		encoder := csv.NewWriter(w)
		_ = encoder.Write([]string{"Nome", "Cognome", "Email", "Età", "Località", "Frequenza", "Canale scoperta", "Sponsor preferito", "Canale contatto", "Consenso Club", "Consenso Sponsor", "Consenso Analytics", "Punti"})
		for _, p := range filtered {
			_ = encoder.Write([]string{
				p.FirstName,
				p.LastName,
				p.Email,
				p.AgeRange,
				p.Location,
				p.AttendanceFrequency,
				p.DiscoveryChannel,
				p.SponsorPreference,
				p.ContactChannel,
				strconv.FormatBool(p.ConsentClub),
				strconv.FormatBool(p.ConsentSponsors),
				strconv.FormatBool(p.ConsentAnalytics),
				strconv.Itoa(p.Points),
			})
		}
		encoder.Flush()
		return
	}

	rt.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"profiles": filtered,
		"total":    len(filtered),
	})
}

func (rt *_router) createMarketingBadge(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var payload database.MarketingBadge
	if err := rt.decodeJSONBody(w, r, &payload); err != nil {
		ctx.Logger.WithError(err).Warn("invalid marketing badge payload")
		return
	}
	payload.OrganizationID = ctx.OrganizationID
	if payload.Name == "" || payload.Threshold <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	badge, err := rt.db.SaveMarketingBadge(payload)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot save marketing badge")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	rt.respondWithJSON(w, http.StatusCreated, badge)
}

func (rt *_router) listMarketingBadges(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	badges, err := rt.db.ListMarketingBadges(ctx.OrganizationID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list marketing badges")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	rt.respondWithJSON(w, http.StatusOK, map[string]interface{}{"badges": badges})
}

func (rt *_router) getMarketingConsents(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	breakdown, err := rt.db.GetConsentBreakdown(ctx.OrganizationID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot get consent breakdown")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	policies, _ := rt.db.ListPrivacyPolicies(ctx.OrganizationID)
	rt.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"breakdown": breakdown,
		"policies":  policies,
	})
}

func (rt *_router) upsertPrivacyPolicy(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var payload database.PrivacyPolicy
	if err := rt.decodeJSONBody(w, r, &payload); err != nil {
		ctx.Logger.WithError(err).Warn("invalid privacy policy payload")
		return
	}
	payload.OrganizationID = ctx.OrganizationID
	if strings.TrimSpace(payload.Version) == "" || strings.TrimSpace(payload.Title) == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	policy, err := rt.db.SavePrivacyPolicy(payload)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot save privacy policy")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	rt.respondWithJSON(w, http.StatusCreated, policy)
}

func (rt *_router) recordFanProfile(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var payload database.FanProfile
	if err := rt.decodeJSONBody(w, r, &payload); err != nil {
		ctx.Logger.WithError(err).Warn("invalid fan profile payload")
		return
	}
	payload.OrganizationID = ctx.OrganizationID
	payload.Email = strings.TrimSpace(payload.Email)
	if payload.Email == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	profile, err := rt.db.SaveFanProfile(payload)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot save fan profile")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	rt.respondWithJSON(w, http.StatusCreated, profile)
}

func (rt *_router) logFanConsent(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var payload database.FanConsent
	if err := rt.decodeJSONBody(w, r, &payload); err != nil {
		ctx.Logger.WithError(err).Warn("invalid consent payload")
		return
	}
	payload.OrganizationID = ctx.OrganizationID
	consent, err := rt.db.LogFanConsent(payload)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot save consent")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	rt.respondWithJSON(w, http.StatusCreated, consent)
}

func (rt *_router) recordGamificationEvent(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var payload database.GamificationEvent
	if err := rt.decodeJSONBody(w, r, &payload); err != nil {
		ctx.Logger.WithError(err).Warn("invalid gamification payload")
		return
	}
	payload.OrganizationID = ctx.OrganizationID
	if payload.FanProfileID == 0 || payload.Points == 0 || strings.TrimSpace(payload.Source) == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if payload.SeasonLabel == "" {
		payload.SeasonLabel = time.Now().Format("2006")
	}
	if err := rt.db.RecordGamificationEvent(payload); err != nil {
		ctx.Logger.WithError(err).Error("cannot record gamification event")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func filterAndSortProfiles(profiles []database.FanProfile) []database.FanProfile {
	sort.SliceStable(profiles, func(i, j int) bool { return profiles[i].Points > profiles[j].Points })
	return profiles
}
