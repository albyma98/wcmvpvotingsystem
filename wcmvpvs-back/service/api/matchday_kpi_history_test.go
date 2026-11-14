package api

import (
	"context"
	"testing"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
)

func TestMatchdayHistoryUsesLocalSeasonForDivision(t *testing.T) {
	f := newEngagementReportFixture(t)
	exec := func(query string, args ...interface{}) {
		t.Helper()
		if _, err := f.conn.Exec(query, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`UPDATE teams SET championship='Serie A' WHERE id=?`, f.org.TeamID)
	newEvent := func(date string) int {
		t.Helper()
		event, err := f.router.db.CreateEvent(database.Event{OrganizationID: f.org.ID, Team1ID: f.org.TeamID, Team2ID: f.other.TeamID, StartDateTime: date})
		if err != nil {
			t.Fatal(err)
		}
		exec(`UPDATE events SET is_concluded=1 WHERE id=?`, event)
		return event
	}
	old := newEvent("2026-06-30T21:59:00Z")
	current := newEvent("2026-06-30T22:01:00Z")
	ctx := context.Background()
	event, prior, scope, err := loadMatchdayEvents(ctx, f.conn, f.org.ID, current, 4, "")
	if err != nil || scope != "division" || matchdaySeason(event.start) != 2026 || len(prior) != 0 {
		t.Fatalf("division season: event=%+v prior=%+v scope=%s err=%v", event, prior, scope, err)
	}
	_, prior, scope, err = loadMatchdayEvents(ctx, f.conn, f.org.ID, current, 4, "club")
	if err != nil || scope != "club" || len(prior) != 1 || prior[0].ID != old {
		t.Fatalf("club history: prior=%+v scope=%s err=%v", prior, scope, err)
	}
	facts := func(e matchdayEvent) *matchdayFacts {
		reach := map[string]bool{"device:a": true}
		return &matchdayFacts{Event: e, Metrics: map[string]matchdayMetric{}, Reach: reach, persistentReach: reach, coverage: "new_home_entries"}
	}
	now := facts(event)
	calculateMatchdayRetention(now, facts(prior[0]), nil)
	if got := now.Metrics["direction_retention"]; got.Value != nil || got.Status != "missing" {
		t.Fatalf("retention crossed the local season: %+v", got)
	}
}

func TestMatchdayComparisonSignedBaseline(t *testing.T) {
	current := matchdayCount("test", "Test", "", "", 5, "fixture")
	previous := matchdayCount("test", "Test", "", "", -10, "fixture")
	old := &matchdayFacts{Event: matchdayEvent{start: time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)}, Metrics: map[string]matchdayMetric{"test": previous}}
	comparison := matchdayCompare(current, []*matchdayFacts{old}, 2026)
	matchdayRequireValue(t, comparison.AbsoluteDelta, 15)
	matchdayRequireValue(t, comparison.PercentDelta, -150)
	old.Metrics["test"] = matchdayCount("test", "Test", "", "", 0, "fixture")
	comparison = matchdayCompare(current, []*matchdayFacts{old}, 2026)
	matchdayRequireValue(t, comparison.AbsoluteDelta, 5)
	if comparison.PercentDelta != nil || comparison.AveragePercentDelta != nil {
		t.Fatalf("zero baseline has a relative delta: %+v", comparison)
	}
}

func TestMatchdayCRMAndRetentionRequireAttributableIdentity(t *testing.T) {
	f := newEngagementReportFixture(t)
	exec := func(query string, args ...interface{}) {
		t.Helper()
		if _, err := f.conn.Exec(query, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO fan_profiles (id,organization_id,nickname,phone,created_at,phone_verified_at,marketing_consent)
		VALUES (51,?,'A','test-a','2026-10-15 12:00:00','2026-10-15 12:00:00',1),
		       (52,?,'B','test-b','2026-10-21 12:00:00','2026-10-21 12:00:00',1)`, f.org.ID, f.org.ID)
	newEvent := func(date string) int {
		t.Helper()
		event, err := f.router.db.CreateEvent(database.Event{OrganizationID: f.org.ID, Team1ID: f.org.TeamID, Team2ID: f.other.TeamID, StartDateTime: date})
		if err != nil {
			t.Fatal(err)
		}
		return event
	}
	older := newEvent("2026-10-01T18:00:00Z")
	old := newEvent("2026-10-08T18:00:00Z")
	previous := newEvent("2026-10-15T18:00:00Z")
	exec(`INSERT INTO engagement_report_signals (event_id,signal_id,name,session_id,device_key,fan_id,occurred_at,metadata)
		VALUES (?,'shared-a','page_view','a','shared',51,'2026-10-01T18:00:00Z','{}'),
		       (?,'unique-a','page_view','a','unique-a',51,'2026-10-01T18:00:00Z','{}'),
		       (?,'shared-b','game_started','b','shared',52,'2026-10-08T18:00:00Z','{}')`, older, older, old)
	makeFacts := func(id int, day int, signals []reportSignal) *matchdayFacts {
		return &matchdayFacts{Event: matchdayEvent{ID: id, start: time.Date(2026, time.October, day, 18, 0, 0, 0, time.UTC)}, Signals: signals, Metrics: map[string]matchdayMetric{}}
	}
	prev := makeFacts(previous, 15, []reportSignal{
		{Name: "page_view", DeviceID: "shared"},
		{Name: "page_view", DeviceID: "unique-a"},
		{Name: "page_view", DeviceID: "old-only"},
	})
	current := makeFacts(f.event, 21, []reportSignal{
		{Name: "page_view", DeviceID: "shared"},
		{Name: "page_view", DeviceID: "unique-a"},
		{Name: "page_view", DeviceID: "unique-b", FanID: 52},
	})
	if err := calculateMatchdayAudience(context.Background(), f.conn, f.org.ID, []*matchdayFacts{current, prev}, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	calculateMatchdayRetention(current, prev, nil)
	checkMatchdayDomainMetric(t, current, "crm_identified", 1)
	checkMatchdayDomainMetric(t, current, "crm_new_profiles", 1)
	checkMatchdayDomainMetric(t, current, "crm_identification_rate", 100.0/3)
	checkMatchdayDomainMetric(t, prev, "crm_identified", 0)
	checkMatchdayDomainMetric(t, current, "direction_retention", 50)
	if !current.Reach["device:shared"] || current.persistentReach["device:shared"] || current.FirstEvent["device:shared"] != 0 {
		t.Fatalf("ambiguous browser entered the persistent audience: %+v", current.persistentReach)
	}
	if got := current.Metrics["crm_opt_in"]; got.Value != nil || got.Status != "not_measurable" {
		t.Fatalf("legacy consent was attributed to the match: %+v", got)
	}
	current.Signals[0].Metadata = map[string]interface{}{"entry_channel": "direct"}
	calculateMatchdayCRM(current, nil, nil)
	for _, code := range []string{"crm_opt_in", "crm_new_consents"} {
		if got := current.Metrics[code]; got.Value != nil || got.Status != "not_applicable" {
			t.Fatalf("modern marketing consent %s: %+v", code, got)
		}
	}
}

func TestMatchdayAcquisitionIsIndependentOfChartHistory(t *testing.T) {
	f := newEngagementReportFixture(t)
	if _, err := f.conn.Exec(`INSERT INTO fan_profiles(id,organization_id,nickname,phone) VALUES(51,?,'A','test-a')`, f.org.ID); err != nil {
		t.Fatal(err)
	}
	previous, err := f.router.db.CreateEvent(database.Event{OrganizationID: f.org.ID, Team1ID: f.org.TeamID, Team2ID: f.other.TeamID, StartDateTime: "2026-10-15T18:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	old := &matchdayFacts{
		Event:   matchdayEvent{ID: previous, start: time.Date(2026, time.October, 15, 18, 0, 0, 0, time.UTC)},
		Metrics: map[string]matchdayMetric{},
		Actions: []matchdayAction{{Identity: matchdayIdentity{DeviceKey: "browser", FanID: 51}, Feature: "game", Count: 1}},
	}
	for _, includePrevious := range []bool{false, true} {
		current := &matchdayFacts{
			Event:   matchdayEvent{ID: f.event, start: time.Date(2026, time.October, 21, 19, 30, 0, 0, time.UTC)},
			Metrics: map[string]matchdayMetric{},
			Signals: []reportSignal{{Name: "page_view", SessionID: "current", DeviceID: "browser"}},
		}
		events := []*matchdayFacts{current}
		if includePrevious {
			events = append(events, old)
		}
		if err := calculateMatchdayAudience(context.Background(), f.conn, f.org.ID, events, func() error { return nil }); err != nil {
			t.Fatal(err)
		}
		if !current.Reach["device:browser"] || current.FirstEvent["device:browser"] != f.event {
			t.Fatalf("chart history %t changed the current acquisition: reach=%+v first=%+v", includePrevious, current.Reach, current.FirstEvent)
		}
	}
}
