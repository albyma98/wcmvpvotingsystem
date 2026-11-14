package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
)

func TestMatchdayTechnologyFromSignedRequestsAndIndependentProbeSlots(t *testing.T) {
	f := newEngagementReportFixture(t)
	rt := f.router
	m := &matchdayRequestRecorder{db: f.conn, logger: rt.baseLogger, secret: "test-signer", pending: make(map[matchdayRequestKey]int)}
	rt.matchdayRequests = m
	if _, err := f.conn.Exec(`UPDATE events SET is_active=1 WHERE id=?`, f.event); err != nil {
		t.Fatal(err)
	}
	initial := httptest.NewRequest(http.MethodGet, "/active-event?matchday=1", nil)
	initial.Header.Set("X-Organization-Slug", f.org.Slug)
	response := httptest.NewRecorder()
	rt.wrap(rt.getActiveEvent).ServeHTTP(response, initial)
	var body struct {
		ID    int    `json:"id"`
		Grant string `json:"matchday_grant"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body.ID != f.event || body.Grant == "" {
		t.Fatalf("initial response %d: %s (%v)", response.Code, response.Body.String(), err)
	}

	serve := func(path string, organization, status, payloadEvent int, grant string) {
		r := httptest.NewRequest(http.MethodPost, path, nil)
		r.Header.Set(matchdayGrantHeader, grant)
		w := httptest.NewRecorder()
		rt.observeMatchdayRequest(w, r, organization, time.Now(), func(w http.ResponseWriter, r *http.Request) {
			validateMatchdayRequestEvent(r, payloadEvent)
			w.WriteHeader(status)
		})
	}
	serve("/vote", f.org.ID, 200, f.event, body.Grant)
	serve("/vote", f.org.ID, 429, f.event, body.Grant)
	serve("/vote", f.org.ID, 409, f.event+1, body.Grant)
	serve("/vote", f.other.ID, 500, f.event, body.Grant)
	serve("/vote", f.org.ID, 500, f.event, body.Grant+"x")
	serve(fmt.Sprintf("/events/%d/fan-live/stream", f.event), f.org.ID, 500, f.event, body.Grant)
	serve(fmt.Sprintf("/events/%d/fan-live", f.event), f.org.ID, 500, f.event, body.Grant)
	serve(fmt.Sprintf("/events/%d/fan-live", f.event), f.org.ID, 404, f.event, body.Grant)
	serve(fmt.Sprintf("/events/%d/fan-live", f.event), f.org.ID, 429, f.event, body.Grant)
	serve(fmt.Sprintf("/events/%d/fan-live", f.event), f.org.ID, 302, f.event, body.Grant)
	serve(fmt.Sprintf("/events/%d/fan-live", f.event), f.org.ID, 600, f.event, body.Grant)
	m.flush()

	now := time.Now().Truncate(30 * time.Second).Unix()
	if _, err := f.conn.Exec(`UPDATE events SET is_active=0,is_concluded=1 WHERE id=?`, f.event); err != nil {
		t.Fatal(err)
	}
	if _, err := f.conn.Exec(`UPDATE matchday_technology_windows SET opened_at=?,closed_at=? WHERE event_id=?`,
		(now-90)*1000, (now+1)*1000, f.event); err != nil {
		t.Fatal(err)
	}
	if _, err := f.conn.Exec(`INSERT INTO matchday_health_probes(slot_start,healthy) VALUES(?,1),(?,0),(?,1),(?,1)`, now-60, now-30, now, now-1); err != nil {
		t.Fatal(err)
	}
	facts := &matchdayFacts{Metrics: map[string]matchdayMetric{}}
	if err := loadMatchdayTechnology(context.Background(), f.conn, f.org.ID, map[int]*matchdayFacts{f.event: facts}); err != nil {
		t.Fatal(err)
	}
	got := facts.Technology
	if got.requests != 8 || got.errors != 4 || got.clientErrors != 1 || got.limitedRequests != 2 || got.serverErrors != 1 || got.votes != 2 || got.successfulVotes != 1 || got.limited != 1 {
		t.Fatalf("unexpected HTTP observations: %+v", got)
	}
	if got.expectedProbes != 4 || got.probes != 3 || got.healthy != 2 {
		t.Fatalf("unexpected health observations: %+v", got)
	}
	calculateMatchdayServerTechnology(facts)
	if metric := facts.Metrics["tech_uptime"]; metric.Status != "partial" || metric.Value == nil || *metric.Value < 66 || *metric.Value > 67 {
		t.Fatalf("unexpected uptime: %+v", metric)
	}
	if metric := facts.Metrics["tech_vote_success"]; metric.Status != "available" || metric.Value == nil || *metric.Value != 50 {
		t.Fatalf("unexpected voting outcome: %+v", metric)
	}
	errorRate := facts.Metrics["tech_request_error_rate"]
	if errorRate.Value == nil || *errorRate.Value != 50 || len(errorRate.Rows) != 3 {
		t.Fatalf("unexpected HTTP error rate: %+v", errorRate)
	}
	for i, want := range []float64{1, 2, 1} {
		if count := errorRate.Rows[i].Values["count"]; count == nil || *count != want {
			t.Fatalf("unexpected HTTP error count for %s: %v", errorRate.Rows[i].Label, count)
		}
	}
}

func TestMatchdayTechnologyWithNoProbeMeasurements(t *testing.T) {
	facts := &matchdayFacts{
		Metrics:    map[string]matchdayMetric{},
		Technology: matchdayTechnologyFacts{expectedProbes: 4},
	}
	calculateMatchdayServerTechnology(facts)
	metric := facts.Metrics["tech_uptime"]
	if metric.Status != "not_measurable" || metric.Value != nil || len(metric.Rows) != 4 {
		t.Fatalf("unexpected uptime with no probes: %+v", metric)
	}
	for i, want := range []float64{4, 0, 0, 4} {
		if count := metric.Rows[i].Values["count"]; count == nil || *count != want {
			t.Fatalf("unexpected probe count for %s: %v", metric.Rows[i].Label, count)
		}
	}
	facts.Technology.probes = facts.Technology.expectedProbes
	calculateMatchdayServerTechnology(facts)
	if metric := facts.Metrics["tech_uptime"]; metric.Status != "available" || metric.Value == nil || *metric.Value != 0 {
		t.Fatalf("failed probes are an observed 0%% uptime: %+v", metric)
	}
}

func TestMatchdayLatencyP95UsesWeightedNearestRank(t *testing.T) {
	for _, tc := range []struct {
		facts matchdayTechnologyFacts
		want  int64
	}{
		{matchdayTechnologyFacts{requests: 20, latencies: map[int64]int{0: 18, 12: 1, 2000: 1}}, 12},
		{matchdayTechnologyFacts{requests: 19, latencies: map[int64]int{0: 18, 2000: 1}}, 2000},
		{matchdayTechnologyFacts{requests: 1, latencies: map[int64]int{0: 1}}, 0},
	} {
		if got := tc.facts.latencyP95(); got != tc.want {
			t.Errorf("p95 of %v = %d, want %d", tc.facts.latencies, got, tc.want)
		}
	}
}

func TestMatchdayMeasuredSponsorGeometryAndIncompleteExposure(t *testing.T) {
	f := newEngagementReportFixture(t)
	db := f.router.db
	org := f.org.ID
	key := reportOperationalDeviceKey(org, "test-device-0123456")
	start := database.SponsorAnalyticsEvent{ID: "impression-1", ExposureID: "exposure-1", SessionID: "session-1", ClubID: org, MatchID: f.event, SponsorID: 1, SponsorName: "Partner", CreativeID: "logo", Placement: "game_wall_left", Game: "basketball", Method: database.SponsorSVG, Type: database.SponsorImpression, Timestamp: 1000, Percentage: 100, GeometrySampled: true, ReportDeviceKey: key}
	qualified := start
	qualified.ID, qualified.Type, qualified.Timestamp, qualified.Duration, qualified.Continuous = "qualified-1", database.SponsorGeometryViewable, 2000, 1000, 1000
	checkpoint := qualified
	checkpoint.ID, checkpoint.Type, checkpoint.Timestamp, checkpoint.Duration, checkpoint.Continuous = "checkpoint-1", database.SponsorExposureSample, 6000, 5000, 5000
	legacyGeometry := start
	legacyGeometry.ID, legacyGeometry.ExposureID, legacyGeometry.GeometrySampled = "impression-old", "exposure-old", false
	click := database.SponsorAnalyticsEvent{ID: "click-1", ExposureID: "exposure-2", SessionID: "session-1", ClubID: org, MatchID: f.event, SponsorID: 1, SponsorName: "Partner", CreativeID: "logo", Placement: "home_footer", Method: database.SponsorHTML, Type: database.SponsorClick, Timestamp: 3000, Clickable: true, ReportDeviceKey: key}
	if count, err := db.RecordSponsorAnalytics(context.Background(), "audience", []database.SponsorAnalyticsEvent{start, qualified, checkpoint, legacyGeometry, click}); err != nil || count != 5 {
		t.Fatalf("stored %d observations, error %v", count, err)
	}
	facts := &matchdayFacts{Metrics: map[string]matchdayMetric{}}
	if err := loadMatchdaySponsors(context.Background(), f.conn, org, map[int]*matchdayFacts{f.event: facts}, ""); err != nil {
		t.Fatal(err)
	}
	if metric := facts.Metrics["sponsor_viewability"]; metric.Status != "partial" || metric.Value == nil || *metric.Value != 100 || *metric.Denominator != 1 {
		t.Fatalf("unexpected sponsor geometry: %+v", metric)
	}
	if metric := facts.Metrics["sponsor_visible_time"]; metric.Status != "partial" || metric.Value == nil || *metric.Value != 5000 {
		t.Fatalf("unexpected sponsor checkpoint: %+v", metric)
	}
	if len(facts.Actions) != 1 || facts.Actions[0].Identity.DeviceKey != key {
		t.Fatalf("sponsor click missing identity: %+v", facts.Actions)
	}
	if metric := facts.Metrics["sponsor_ctr"]; metric.Status != "not_measurable" || metric.Value != nil {
		t.Fatalf("a click without a corresponding impression cannot establish CTR: %+v", metric)
	}

	html := click
	html.ID, html.ExposureID, html.Type, html.Percentage = "impression-3", "exposure-3", database.SponsorImpression, 100
	htmlClick := html
	htmlClick.ID, htmlClick.Type = "click-3", database.SponsorClick
	noClick := html
	noClick.ID, noClick.ExposureID = "impression-4", "exposure-4"
	noClick.SponsorID, noClick.SponsorName = 2, "Partner 2"
	if count, err := db.RecordSponsorAnalytics(context.Background(), "audience", []database.SponsorAnalyticsEvent{html, htmlClick, noClick}); err != nil || count != 3 {
		t.Fatalf("stored %d observations, error %v", count, err)
	}
	facts = &matchdayFacts{Metrics: map[string]matchdayMetric{}}
	if err := loadMatchdaySponsors(context.Background(), f.conn, org, map[int]*matchdayFacts{f.event: facts}, ""); err != nil {
		t.Fatal(err)
	}
	if metric := facts.Metrics["sponsor_ctr"]; metric.Value == nil || *metric.Value != 50 || *metric.Numerator != 1 || *metric.Denominator != 2 {
		t.Fatalf("CTR must exclude non-clickable impressions and unmatched clicks: %+v", metric)
	}
	if metric := facts.Metrics["sponsor_clicks"]; metric.Value == nil || *metric.Value != 2 {
		t.Fatalf("click count must include unmatched clicks: %+v", metric)
	}
	if metric := facts.Metrics["sponsor_reach"]; metric.Value == nil || *metric.Value != 1 {
		t.Fatalf("reach must deduplicate across placements and sponsors: %+v", metric)
	}
	for _, code := range []string{"sponsor_impressions", "sponsor_placements"} {
		for _, row := range facts.Metrics[code].Rows {
			ctr := row.Values["ctr"]
			if row.Label == "Tabellone di gioco, sinistra" {
				if ctr != nil {
					t.Fatalf("%s has a CTR for a non-clickable placement: %+v", code, row)
				}
				continue
			}
			wantClicks, wantImpressions := 1.0, 2.0
			switch row.Label {
			case "Partner":
				wantImpressions = 1
			case "Partner 2":
				wantClicks, wantImpressions = 0, 1
			}
			numerator, denominator := row.Values["clicks_on_impressions"], row.Values["clickable_impressions"]
			if ctr == nil || *ctr != 100*wantClicks/wantImpressions || numerator == nil || *numerator != wantClicks || denominator == nil || *denominator != wantImpressions {
				t.Fatalf("%s has incorrect CTR details: %+v", code, row)
			}
		}
	}
}

func TestMatchdayTechnologyUsesMatchMerchPaymentOutcomes(t *testing.T) {
	facts := &matchdayFacts{Metrics: map[string]matchdayMetric{}}
	facts.Operations.barPayments, facts.Operations.barOrders = 2, 2
	facts.Operations.merch = matchdayMerchFacts{paymentPaid: 2, paymentFailed: 1, paymentExpired: 1, paymentPending: 5}
	calculateMatchdayTechnology(facts)
	metric := facts.Metrics["tech_payment_success"]
	matchdayOperationsAssertValue(t, metric, 50)
	if metric.Status != "available" || metric.coverage != "merch-online" || *metric.Denominator != 4 || len(metric.Rows) != 5 {
		t.Fatalf("unexpected merchandising payment KPI: %+v", metric)
	}
	for i, expected := range []float64{9, 2, 1, 1, 5} {
		matchdayRequireValue(t, metric.Rows[i].Values["count"], expected)
	}
	facts.Operations.merch = matchdayMerchFacts{paymentPending: 5}
	calculateMatchdayTechnology(facts)
	if metric := facts.Metrics["tech_payment_success"]; metric.Value != nil || metric.Status != "not_measurable" || len(metric.Rows) != 5 {
		t.Fatalf("unresolved online payments were counted as bar: %+v", metric)
	}
}
