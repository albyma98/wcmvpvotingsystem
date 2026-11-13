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
	m.flush()

	now := time.Now().Truncate(30 * time.Second).Unix()
	if _, err := f.conn.Exec(`UPDATE events SET is_active=0,is_concluded=1 WHERE id=?`, f.event); err != nil {
		t.Fatal(err)
	}
	if _, err := f.conn.Exec(`UPDATE matchday_technology_windows SET opened_at=?,closed_at=? WHERE event_id=?`,
		(now-90)*1000, (now+1)*1000, f.event); err != nil {
		t.Fatal(err)
	}
	if _, err := f.conn.Exec(`INSERT INTO matchday_health_probes(slot_start,healthy) VALUES(?,1),(?,0),(?,1)`, now-60, now-30, now); err != nil {
		t.Fatal(err)
	}
	facts := &matchdayFacts{Metrics: map[string]matchdayMetric{}}
	if err := loadMatchdayTechnology(context.Background(), f.conn, f.org.ID, map[int]*matchdayFacts{f.event: facts}); err != nil {
		t.Fatal(err)
	}
	got := facts.Technology
	if got.requests != 4 || got.errors != 2 || got.serverErrors != 1 || got.votes != 2 || got.successfulVotes != 1 || got.limited != 1 {
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
	click := database.SponsorAnalyticsEvent{ID: "click-1", ExposureID: "exposure-2", SessionID: "session-1", ClubID: org, MatchID: f.event, SponsorID: 1, SponsorName: "Partner", CreativeID: "logo", Placement: "home_footer", Method: database.SponsorHTML, Type: database.SponsorClick, Timestamp: 3000, Clickable: true, ReportDeviceKey: key}
	if count, err := db.RecordSponsorAnalytics(context.Background(), "audience", []database.SponsorAnalyticsEvent{start, qualified, checkpoint, click}); err != nil || count != 4 {
		t.Fatalf("stored %d observations, error %v", count, err)
	}
	facts := &matchdayFacts{Metrics: map[string]matchdayMetric{}}
	if err := loadMatchdaySponsors(context.Background(), f.conn, org, map[int]*matchdayFacts{f.event: facts}, ""); err != nil {
		t.Fatal(err)
	}
	if metric := facts.Metrics["sponsor_viewability"]; metric.Status != "available" || metric.Value == nil || *metric.Value != 100 {
		t.Fatalf("unexpected sponsor geometry: %+v", metric)
	}
	if metric := facts.Metrics["sponsor_visible_time"]; metric.Status != "partial" || metric.Value == nil || *metric.Value != 5000 {
		t.Fatalf("unexpected sponsor checkpoint: %+v", metric)
	}
	if len(facts.Actions) != 1 || facts.Actions[0].Identity.DeviceKey != key {
		t.Fatalf("sponsor click missing identity: %+v", facts.Actions)
	}
}

func TestMatchdayTechnologyUsesMatchMerchPaymentOutcomes(t *testing.T) {
	facts := &matchdayFacts{Metrics: map[string]matchdayMetric{}}
	facts.Operations.barPaid, facts.Operations.barClosed = 2, 2
	facts.Operations.merch = matchdayMerchFacts{observedPayment: true, paymentPaid: 2, paymentResolved: 3}
	calculateMatchdayTechnology(facts)
	metric := facts.Metrics["tech_payment_success"]
	matchdayOperationsAssertValue(t, metric, 200.0/3)
	if metric.Status != "available" || metric.coverage != "merch-online" || *metric.Denominator != 3 {
		t.Fatalf("unexpected merchandising payment KPI: %+v", metric)
	}
	facts.Operations.merch = matchdayMerchFacts{observedPayment: true}
	calculateMatchdayTechnology(facts)
	if metric := facts.Metrics["tech_payment_success"]; metric.Value != nil || metric.Status != "not_measurable" {
		t.Fatalf("unresolved online payments were counted as bar: %+v", metric)
	}
}
