package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
)

type sponsorCatalogEntry struct {
	database.Sponsor
	CreativeID     string `json:"creative_id,omitempty"`
	TextCreativeID string `json:"analytics_text_creative_id,omitempty"`
	AnalyticsGrant string `json:"analytics_grant,omitempty"`
}

type sponsorGrant struct {
	Club         int      `json:"c"`
	Match        int      `json:"m"`
	Sponsor      int      `json:"s"`
	Creative     string   `json:"v"`
	TextCreative string   `json:"t"`
	Name         string   `json:"n"`
	Placements   []string `json:"p"`
	Link         bool     `json:"l"`
	Expires      int64    `json:"e"`
}

func (rt *_router) sponsorAnalyticsConfigured() bool {
	switch strings.TrimSpace(rt.VoteSecret) {
	case "", "secret", "modifica-questa-stringa":
		return false
	default:
		return true
	}
}

func sponsorMAC(secret, purpose, value string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = io.WriteString(mac, "sponsor-analytics/"+purpose+"/v1\x00"+value)
	return mac.Sum(nil)
}

func signSponsorGrant(secret string, grant sponsorGrant) string {
	raw, _ := json.Marshal(grant)
	encoded := base64.RawURLEncoding.EncodeToString(raw)
	return encoded + "." + base64.RawURLEncoding.EncodeToString(sponsorMAC(secret, "grant", encoded))
}

func readSponsorGrant(secret, token string, now time.Time) (sponsorGrant, bool) {
	var grant sponsorGrant
	encoded, signature, ok := strings.Cut(token, ".")
	if !ok || len(token) > 2048 {
		return grant, false
	}
	sig, err := base64.RawURLEncoding.DecodeString(signature)
	if err != nil || !hmac.Equal(sig, sponsorMAC(secret, "grant", encoded)) {
		return grant, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || json.Unmarshal(raw, &grant) != nil || grant.Expires < now.Unix() {
		return sponsorGrant{}, false
	}
	return grant, true
}

func hasSponsorLink(link string) bool {
	link = strings.ToLower(strings.TrimSpace(link))
	return strings.HasPrefix(link, "https://") || strings.HasPrefix(link, "http://")
}

// Extend the existing placement-aware catalog. A signed snapshot lets an
// exposure finish after a club updates its logo or changes an assignment.
func (rt *_router) writeSponsorAnalyticsCatalog(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext, sponsors []database.Sponsor, matchID int) {
	if matchID > 0 {
		club, err := rt.db.GetEventOrganizationID(matchID)
		if err != nil || club != ctx.OrganizationID {
			writeJSON(w, http.StatusOK, sponsors)
			return
		}
	}
	postVote, err := rt.db.GetSponsorPlacements(r.Context(), ctx.OrganizationID, database.PostVotePlacements, false)
	if err != nil {
		writeJSON(w, http.StatusOK, sponsors)
		return
	}
	walls, err := rt.db.GetSponsorPlacements(r.Context(), ctx.OrganizationID, database.GameWallPlacements, false)
	if err != nil {
		writeJSON(w, http.StatusOK, sponsors)
		return
	}
	entries := make([]sponsorCatalogEntry, 0, len(sponsors))
	active := 0
	expires := time.Now().Add(24 * time.Hour).Unix()
	for _, sponsor := range sponsors {
		entry := sponsorCatalogEntry{Sponsor: sponsor}
		if strings.TrimSpace(sponsor.LogoData) == "" {
			entries = append(entries, entry)
			continue
		}
		var placements []string
		if sponsor.IsActive {
			if active < 6 {
				placements = append(placements, "home_footer")
			}
			if active == 0 {
				placements = append(placements, "sponsor_rush_entry", "sponsor_rush_logo")
			}
			active++
		}
		for placement, id := range postVote {
			if id == sponsor.ID {
				placements = append(placements, placement)
			}
		}
		for placement, id := range walls {
			if id == sponsor.ID {
				placements = append(placements, placement)
			}
		}
		if len(placements) > 0 {
			hash := sha256.Sum256([]byte(sponsor.LogoData))
			entry.CreativeID = fmt.Sprintf("%x", hash[:16])
			textHash := sha256.Sum256([]byte("text\x00" + strings.TrimSpace(sponsor.Name)))
			entry.TextCreativeID = fmt.Sprintf("%x", textHash[:16])
			name := []rune(sponsor.Name)
			if len(name) > 100 {
				name = name[:100]
			}
			entry.AnalyticsGrant = signSponsorGrant(rt.VoteSecret, sponsorGrant{
				Club: ctx.OrganizationID, Match: matchID, Sponsor: sponsor.ID, Creative: entry.CreativeID,
				TextCreative: entry.TextCreativeID, Name: string(name), Placements: placements,
				Link: hasSponsorLink(sponsor.LinkURL), Expires: expires,
			})
		}
		entries = append(entries, entry)
	}
	writeJSON(w, http.StatusOK, entries)
}

func validateSponsorPlacement(e *database.SponsorAnalyticsEvent, grant sponsorGrant) bool {
	if grant.Club != e.ClubID || grant.Match != e.MatchID || grant.Sponsor != e.SponsorID || e.CreativeID == "" {
		return false
	}
	key := e.Placement
	method := database.SponsorHTML
	creativeValid := grant.Creative == e.CreativeID
	switch e.Placement {
	case "home_footer":
		e.Clickable = grant.Link
	case "post_vote_left", "post_vote_right":
		e.FullDuration = database.SponsorPostVoteDuration
		creativeValid = creativeValid || grant.TextCreative == e.CreativeID
	case "sponsor_rush_entry":
		creativeValid = grant.TextCreative == e.CreativeID
		if e.Game != "sponsor-rush" {
			return false
		}
	case "sponsor_rush_logo":
		if e.Game != "sponsor-rush" {
			return false
		}
		method = database.SponsorInGame
	case "game_wall_left", "game_wall_right":
		switch e.Game {
		case "basketball", "playmaker", "crossover":
			key = e.Game + "_" + strings.TrimPrefix(e.Placement, "game_wall_")
			method = database.SponsorSVG
		default:
			return false
		}
	default:
		return false
	}
	if (key == "home_footer" || e.FullDuration > 0) && e.Game != "" {
		return false
	}
	if !creativeValid || method != e.Method || (e.Type == database.SponsorClick && !e.Clickable) {
		return false
	}
	for _, allowed := range grant.Placements {
		if allowed == key {
			e.SponsorName = grant.Name
			return true
		}
	}
	return false
}

var sponsorUUID = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$`)
var sponsorDeviceID = regexp.MustCompile(`^[a-zA-Z0-9_-]{16,80}$`)

type sponsorRateWindow struct {
	minute int64
	counts map[string]int
}

type sponsorAnalyticsLimiter struct {
	mu     sync.Mutex
	window sponsorRateWindow
}

func (l *sponsorAnalyticsLimiter) allow(now time.Time, ip, audience string, club, cost int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	minute := now.Unix() / 60
	if l.window.counts == nil || l.window.minute != minute {
		l.window = sponsorRateWindow{minute: minute, counts: make(map[string]int)}
	}
	counts := l.window.counts
	clubKey := strconv.Itoa(club)
	ipKey := "ip:" + ip
	clubKey = "club:" + clubKey
	deviceKey := "device:" + audience
	if counts["global"]+cost > 36000 || counts[clubKey]+cost > 12000 || counts[ipKey]+cost > 10000 ||
		(audience != "" && counts[deviceKey]+cost > 480) {
		return false
	}
	counts["global"] += cost
	counts[clubKey] += cost
	counts[ipKey] += cost
	if audience != "" {
		counts[deviceKey] += cost
	}
	return true
}

func (rt *_router) ingestMeasuredSponsorEvents(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if ctx.OrganizationID <= 0 || !rt.sponsorAnalyticsConfigured() {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var batch struct {
		Device string                           `json:"device_id"`
		Match  int                              `json:"match_id"`
		Events []database.SponsorAnalyticsEvent `json:"events"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 60*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&batch); err != nil || decoder.Decode(&struct{}{}) != io.EOF ||
		len(batch.Events) == 0 || len(batch.Events) > 24 || batch.Match < 0 ||
		(batch.Device != "" && !sponsorDeviceID.MatchString(batch.Device)) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	now := time.Now()
	audience := ""
	if batch.Device != "" {
		input := strconv.Itoa(ctx.OrganizationID) + "\x00" + batch.Device
		audience = base64.RawURLEncoding.EncodeToString(sponsorMAC(rt.VoteSecret, "audience", input))
	}
	if !rt.sponsorLimiter.allow(now, rt.getClientIP(r), audience, ctx.OrganizationID, len(batch.Events)) {
		w.Header().Set("Retry-After", strconv.FormatInt(60-now.Unix()%60, 10))
		w.WriteHeader(http.StatusTooManyRequests)
		return
	}
	if batch.Match > 0 && !rt.ensureEventInOrganization(w, ctx, batch.Match) {
		return
	}
	grants := make(map[string]sponsorGrant)
	valid := make([]database.SponsorAnalyticsEvent, 0, len(batch.Events))
	earliest, latest := now.Add(-24*time.Hour).UnixMilli(), now.Add(2*time.Minute).UnixMilli()
	for _, e := range batch.Events {
		if e.ClubID != ctx.OrganizationID || e.MatchID != batch.Match || !sponsorUUID.MatchString(e.ID) ||
			!sponsorUUID.MatchString(e.ExposureID) || !sponsorUUID.MatchString(e.SessionID) ||
			e.Timestamp < earliest || e.Timestamp > latest || e.Duration < 0 || e.Duration > 7200000 ||
			e.Continuous < 0 || e.Continuous > e.Duration || e.Percentage < 0 || e.Percentage > 100 {
			continue
		}
		switch e.Type {
		case database.SponsorImpression, database.SponsorViewable, database.SponsorGeometryViewable, database.SponsorExposureSample,
			database.SponsorExposureEnd, database.SponsorFullView, database.SponsorClick:
		default:
			continue
		}
		grant, exists := grants[e.Grant]
		if !exists {
			grant, _ = readSponsorGrant(rt.VoteSecret, e.Grant, now)
			grants[e.Grant] = grant
		}
		if grant.Expires >= now.Unix() && validateSponsorPlacement(&e, grant) {
			if batch.Match > 0 && batch.Device != "" {
				e.ReportDeviceKey = reportOperationalDeviceKey(ctx.OrganizationID, batch.Device)
			}
			valid = append(valid, e)
		}
	}
	accepted := 0
	if len(valid) > 0 {
		var err error
		accepted, err = rt.db.RecordSponsorAnalytics(r.Context(), audience, valid)
		if err != nil {
			ctx.Logger.WithError(err).Warn("sponsor analytics batch could not be saved")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]int{"received": len(batch.Events), "accepted": accepted})
}

func (rt *_router) getMeasuredSponsorAnalytics(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	const day = "2006-01-02"
	query := r.URL.Query()
	match, errMatch := optionalSponsorAnalyticsID(query.Get("match_id"))
	sponsor, errSponsor := optionalSponsorAnalyticsID(query.Get("sponsor_id"))
	if errMatch != nil || errSponsor != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	filter := sponsorAnalyticsMatchFilter(ctx.OrganizationID, match, sponsor)
	if match == 0 || query.Has("from") || query.Has("to") {
		from, errFrom := time.Parse(day, query.Get("from"))
		to, errTo := time.Parse(day, query.Get("to"))
		if errFrom != nil || errTo != nil || to.Before(from) || to.Sub(from) > 366*24*time.Hour {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		filter.From, filter.Until = from, to.AddDate(0, 0, 1)
	}
	if match > 0 && !rt.ensureEventInOrganization(w, ctx, match) {
		return
	}
	report, err := rt.db.GetMeasuredSponsorAnalytics(r.Context(), filter)
	if err != nil {
		ctx.Logger.WithError(err).Warn("cannot load measured sponsor analytics")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, report)
}

func sponsorAnalyticsMatchFilter(club, match, sponsor int) database.SponsorAnalyticsFilter {
	return database.SponsorAnalyticsFilter{
		ClubID: club, MatchID: match, SponsorID: sponsor, From: time.Unix(0, 0).UTC(),
		Until: time.Now().UTC().Truncate(24 * time.Hour).Add(24 * time.Hour),
	}
}

func optionalSponsorAnalyticsID(value string) (int, error) {
	if value == "" || value == "0" {
		return 0, nil
	}
	id, err := strconv.Atoi(value)
	if err != nil || id < 1 {
		return 0, fmt.Errorf("invalid id")
	}
	return id, nil
}

// Commercial report snapshots use the same aggregates as the admin dashboard.
// Their existing engagement and historical sponsor fields remain independent.
func (rt *_router) measuredSponsorReportForEvent(r *http.Request, ctx reqcontext.RequestContext, club, match, sponsor int) *database.SponsorAnalyticsReport {
	report, err := rt.db.GetMeasuredSponsorAnalytics(r.Context(), sponsorAnalyticsMatchFilter(club, match, sponsor))
	if err != nil {
		ctx.Logger.WithError(err).Warn("cannot attach sponsor analytics to report")
		return nil
	}
	if report.Total.Impressions == 0 && report.Total.Clicks == nil {
		return nil
	}
	return &report
}
