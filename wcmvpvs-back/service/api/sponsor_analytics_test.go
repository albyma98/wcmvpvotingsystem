package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/gofrs/uuid"
)

func TestMeasuredSponsorAnalytics(t *testing.T) {
	rt, scope, match, other := votingAPIFixture(t)
	rt.VoteSecret = "sponsor-test-secret"
	ctx := context.Background()
	id, err := rt.db.CreateSponsor(database.Sponsor{OrganizationID: scope.OrganizationID, Position: 1, Name: "Partner",
		LogoData: "image-a", LinkURL: "https://example.com/", IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := rt.db.SaveSponsorPlacements(ctx, scope.OrganizationID, database.PostVotePlacements, map[string]int{"post_vote_left": id}); err != nil {
		t.Fatal(err)
	}
	if err := rt.db.SaveSponsorPlacements(ctx, scope.OrganizationID, database.GameWallPlacements, map[string]int{"basketball_right": id}); err != nil {
		t.Fatal(err)
	}
	catalog := httptest.NewRecorder()
	rt.listPublicSponsors(catalog, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/sponsors?include_placements=1&analytics_match_id=%d", match), nil), scope)
	var entries []sponsorCatalogEntry
	if err := json.Unmarshal(catalog.Body.Bytes(), &entries); err != nil || len(entries) != 1 || entries[0].AnalyticsGrant == "" {
		t.Fatalf("catalog status %d, body %s, err %v", catalog.Code, catalog.Body, err)
	}
	entry := entries[0]

	now := time.Now()
	at := now.Add(-15 * time.Second).UnixMilli()
	newID := func() string { return uuid.Must(uuid.NewV4()).String() }
	newExposure := func(placement, game, method string, session string) database.SponsorAnalyticsEvent {
		return database.SponsorAnalyticsEvent{ID: newID(), Type: database.SponsorImpression, ClubID: scope.OrganizationID, MatchID: match,
			SponsorID: id, CreativeID: entry.CreativeID, Grant: entry.AnalyticsGrant, ExposureID: newID(), SessionID: session,
			Placement: placement, Game: game, Method: method, Timestamp: at, Percentage: 100}
	}
	step := func(e database.SponsorAnalyticsEvent, name string, elapsed, duration, continuous int) database.SponsorAnalyticsEvent {
		e.ID, e.Type, e.Duration, e.Continuous = newID(), name, duration, continuous
		e.Timestamp += int64(elapsed)
		return e
	}
	post := func(events []database.SponsorAnalyticsEvent, device string, org int, wantStatus, wantAccepted int) {
		t.Helper()
		raw, err := json.Marshal(map[string]interface{}{"device_id": device, "match_id": match, "events": events})
		if err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		reqScope := scope
		reqScope.OrganizationID = org
		rt.ingestMeasuredSponsorEvents(w, httptest.NewRequest(http.MethodPost, "/sponsors/analytics/events", bytes.NewReader(raw)), reqScope)
		if w.Code != wantStatus {
			t.Fatalf("status %d want %d: %s", w.Code, wantStatus, w.Body)
		}
		if w.Code == http.StatusOK {
			var result map[string]int
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || result["accepted"] != wantAccepted {
				t.Fatalf("accepted %s want %d: %v", w.Body, wantAccepted, err)
			}
		}
	}

	browser := "existing-device-001"
	session := newID()
	home := newExposure("home_footer", "", database.SponsorHTML, session)
	mvp := newExposure("post_vote_left", "", database.SponsorHTML, session)
	svg := newExposure("game_wall_right", "basketball", database.SponsorSVG, session)
	missingStart := newExposure("home_footer", "", database.SponsorHTML, session)
	invalidPlacement := newExposure("game_wall_left", "basketball", database.SponsorSVG, session)
	forgedCreative := newExposure("home_footer", "", database.SponsorHTML, session)
	forgedCreative.CreativeID = "another"
	batch := []database.SponsorAnalyticsEvent{
		home, step(home, database.SponsorViewable, 10, 1000, 1000), step(home, database.SponsorViewable, 1000, 1000, 1000),
		step(home, database.SponsorClick, 1250, 1250, 1250), step(home, database.SponsorExposureEnd, 2000, 1500, 1500),
		step(home, database.SponsorClick, 2500, 1500, 1500),
		mvp, step(mvp, database.SponsorViewable, 1000, 1000, 1000),
		step(mvp, database.SponsorFullView, 2000, 2000, 2000), step(mvp, database.SponsorFullView, 4000, 4000, 4000),
		step(mvp, database.SponsorExposureEnd, 4200, 4200, 4200),
		svg, step(svg, database.SponsorViewable, 1000, 1000, 1000), step(svg, database.SponsorExposureEnd, 1300, 1300, 1300),
		invalidPlacement, forgedCreative, step(missingStart, database.SponsorExposureEnd, 1500, 1000, 1000),
	}
	post(batch, browser, scope.OrganizationID, http.StatusOK, 10)
	post(batch, browser, scope.OrganizationID, http.StatusOK, 0)
	post(batch, browser, other, http.StatusNotFound, 0)
	returnVisit := newExposure("home_footer", "", database.SponsorHTML, newID())
	post([]database.SponsorAnalyticsEvent{returnVisit, step(returnVisit, database.SponsorViewable, 1000, 1000, 1000)},
		browser, scope.OrganizationID, http.StatusOK, 2)

	filter := database.SponsorAnalyticsFilter{ClubID: scope.OrganizationID, MatchID: match, SponsorID: id,
		From: now.Add(-time.Hour), Until: now.Add(time.Hour)}
	report, err := rt.db.GetMeasuredSponsorAnalytics(ctx, filter)
	if err != nil {
		t.Fatal(err)
	}
	m := report.Total
	if m.Impressions != 4 || m.GeometricImpressions != 1 || m.Reach == nil || *m.Reach != 1 ||
		m.Viewable == nil || *m.Viewable != 3 || m.FullViews == nil || *m.FullViews != 1 ||
		m.Clicks == nil || *m.Clicks != 1 || m.UniqueClicks == nil || *m.UniqueClicks != 1 ||
		m.TotalDuration == nil || *m.TotalDuration != 7000 || m.DurationSamples != 3 || m.Conversions != nil ||
		len(report.Placements) != 3 || len(report.Trend) == 0 || len(report.Creatives) != 3 {
		t.Fatalf("report mismatch: %+v", report)
	}
	for _, row := range report.Placements {
		if row.Method == database.SponsorSVG && (row.Viewability != nil || row.CTR != nil || row.Clicks != nil) {
			t.Fatalf("invented SVG KPI: %+v", row)
		}
	}
	anonymous := newExposure("home_footer", "", database.SponsorHTML, newID())
	post([]database.SponsorAnalyticsEvent{anonymous}, "", scope.OrganizationID, http.StatusOK, 1)
	report, err = rt.db.GetMeasuredSponsorAnalytics(ctx, filter)
	if err != nil || report.Total.Impressions != 5 || report.Total.Reach != nil {
		t.Fatalf("anonymous reach: %+v, %v", report.Total, err)
	}
	filter.ClubID = other
	report, err = rt.db.GetMeasuredSponsorAnalytics(ctx, filter)
	if err != nil || report.Total.Impressions != 0 || len(report.Placements) != 0 {
		t.Fatalf("cross-club report: %+v, %v", report, err)
	}

	if err := rt.db.DeleteSponsor(id, scope.OrganizationID); err != nil {
		t.Fatal(err)
	}
	filter.ClubID = scope.OrganizationID
	report, err = rt.db.GetMeasuredSponsorAnalytics(ctx, filter)
	if err != nil || report.Total.Impressions != 5 || len(report.Sponsors) != 1 || report.Sponsors[0].SponsorName != "Partner" {
		t.Fatalf("lost sponsor history: %+v, %v", report, err)
	}
}

func TestSponsorGrantAndBatchRateLimit(t *testing.T) {
	for _, placeholder := range []string{"", "secret", "modifica-questa-stringa"} {
		if (&_router{VoteSecret: placeholder}).sponsorAnalyticsConfigured() {
			t.Fatal("enabled analytics signing with a public configuration placeholder")
		}
	}
	now := time.Unix(1800000000, 0)
	grant := sponsorGrant{Club: 1, Match: 2, Sponsor: 3, Creative: "creative", Placements: []string{"home_footer"}, Expires: now.Unix() + 60}
	token := signSponsorGrant("secret", grant)
	if decoded, ok := readSponsorGrant("secret", token, now); !ok || decoded.Creative != grant.Creative {
		t.Fatalf("signed catalog failed: %+v", decoded)
	}
	if _, ok := readSponsorGrant("different", token, now); ok {
		t.Fatal("accepted forged grant")
	}
	if _, ok := readSponsorGrant("secret", token, now.Add(time.Hour)); ok {
		t.Fatal("accepted expired grant")
	}
	var limiter sponsorAnalyticsLimiter
	for i := 0; i < 20; i++ {
		if !limiter.allow(now, "wifi", "device-a", 1, 24) {
			t.Fatal("throttled ordinary device batches")
		}
	}
	if limiter.allow(now, "wifi", "device-a", 1, 1) || !limiter.allow(now, "wifi", "device-b", 1, 24) ||
		!limiter.allow(now.Add(time.Minute), "wifi", "device-a", 1, 24) {
		t.Fatal("rate limit did not isolate devices and reset after a minute")
	}
}
