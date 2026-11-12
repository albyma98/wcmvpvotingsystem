package api

import (
	"context"
	"encoding/base64"
	"math"
	"strconv"
	"testing"
)

func TestMatchdayMarketingAttributedEntries(t *testing.T) {
	resolve := func(id matchdayIdentity) string {
		if id.Device != "" {
			return "device:" + id.Device
		}
		if id.DeviceKey != "" {
			return "device:" + id.DeviceKey
		}
		return "session:" + id.Session
	}
	old := &matchdayFacts{
		Event: matchdayEvent{ID: 1}, Resolve: resolve,
		Tracking: []matchdayTracking{{Name: "session.started", Session: "old", Device: "b", Metadata: map[string]interface{}{"url": "https://club.test/?utm_source=led&utm_medium=qr"}}},
	}
	current := &matchdayFacts{
		Event: matchdayEvent{ID: 2}, Resolve: resolve, Metrics: map[string]matchdayMetric{},
		FirstEvent: map[string]int{"device:a": 2, "device:b": 1},
		Tracking: []matchdayTracking{
			{Name: "session.started", Session: "same", Device: "a", At: "2026-10-22T10:00:00Z", Metadata: map[string]interface{}{"url": "https://club.test/?utm_source=led&utm_medium=qr"}},
			{Name: "session.started", Session: "same", Device: "a", At: "2026-10-22T10:01:00Z", Metadata: map[string]interface{}{"url": "https://club.test/?utm_source=giornalino"}},
			{Name: "session.started", Session: "other", Device: "b", At: "2026-10-22T10:04:00Z", Metadata: map[string]interface{}{"url": "https://club.test/", "referrer": "https://m.facebook.com/example"}},
		},
		Signals: []reportSignal{{Name: "page_view", SessionID: "new-fan-home", DeviceID: "a", OccurredAt: "2026-10-22T10:02:00Z", Metadata: map[string]interface{}{"source": "arena"}}},
		Actions: []matchdayAction{{Identity: matchdayIdentity{Device: "a"}, Feature: "feedback", Count: 1}},
		Manual:  map[string]matchdayManualField{},
	}
	calculateMatchdayMarketing(current, old)
	checkMatchdayDomainMetric(t, current, "marketing_access", 3)
	checkMatchdayDomainMetric(t, current, "marketing_new_users", 1)
	checkMatchdayDomainMetric(t, current, "marketing_activation", 50)
	checkMatchdayDomainMetric(t, current, "marketing_channel_change", 2)
	if got := current.Metrics["marketing_qr_conversion"]; got.Value != nil || got.Status != "missing" {
		t.Fatalf("QR without scans = %+v", got)
	}
	current.Manual["qr_scans"] = matchdayManualField{matchdayManualRevision: matchdayManualRevision{Value: matchdayDomainValue(20)}}
	current.Manual["marketing_spend"] = matchdayManualField{matchdayManualRevision: matchdayManualRevision{Value: matchdayDomainValue(40)}}
	calculateMatchdayMarketing(current, old)
	checkMatchdayDomainMetric(t, current, "marketing_qr_conversion", 5)
	checkMatchdayDomainMetric(t, current, "marketing_cpa", 40)
	if got := current.Metrics["marketing_qr_conversion"]; got.Source != "mixed" {
		t.Fatalf("QR provenance = %+v", got)
	}
}

func TestMatchdaySponsorMeasurementAndFeedbackCoverage(t *testing.T) {
	f := newEngagementReportFixture(t)
	exec := func(query string, args ...interface{}) {
		t.Helper()
		if _, err := f.conn.Exec(query, args...); err != nil {
			t.Fatal(err)
		}
	}
	const secret = "matchday-sponsor-test-secret"
	const browser = "observed-device-abc"
	audience := base64.RawURLEncoding.EncodeToString(sponsorMAC(secret, "audience", strconv.Itoa(f.org.ID)+"\x00"+browser))
	insert := func(id string, org, sponsor int, placement, game, method string, viewable, duration, click int) {
		exec(`INSERT INTO sponsor_measured_exposures
			(id,club_id,match_id,sponsor_id,sponsor_name,creative_id,placement_id,game_id,method,session_id,
			 audience_key,start_ms,last_ms,visible_ms,continuous_ms,max_percentage,impression,viewable,ended,full_duration,clickable,clicks)
			VALUES(?,?,?,?,'Sponsor','creative',?,?,?,'session',?,1000,3000,?,0,90,1,?,1,0,?,?)`,
			id, org, f.event, sponsor, placement, game, method, audience, duration, viewable, click, click)
	}
	insert("home", f.org.ID, 1, "home_footer", "", "html_display", 1, 1500, 1)
	insert("vote", f.org.ID, 2, "post_vote_left", "", "html_display", 0, 300, 0)
	insert("svg", f.org.ID, 2, "game_wall_left", "basketball", "svg_geometry", 0, 0, 0)
	insert("foreign", f.other.ID, 3, "home_footer", "", "html_display", 1, 1500, 1)
	exec(`UPDATE events SET show_feedback_survey=1 WHERE id=?`, f.event)
	exec(`INSERT INTO event_feedback(event_id,response_id,experience,team_spirit,perks_interest,mini_games_interest,report_experience,suggestion)
		VALUES (?,'first','easy','high','yes','maybe','good','Suggerisco più giochi'),(?,'second','easy','','','','','Un gioco in più')`, f.event, f.event)
	facts := &matchdayFacts{
		Event: matchdayEvent{ID: f.event}, Metrics: map[string]matchdayMetric{},
		Resolve: func(id matchdayIdentity) string {
			if id.Device != "" {
				return "device:" + id.Device
			}
			if id.DeviceKey != "" {
				return "device:" + id.DeviceKey
			}
			return ""
		},
		Tracking: []matchdayTracking{{Device: browser}},
		Signals: []reportSignal{
			{Name: "cta_viewed", DeviceID: "observed", SessionID: "feedback", Metadata: map[string]interface{}{"cta": "feedback", "placement": "feedback_home"}},
			{Name: "feedback_opened", DeviceID: "observed", SessionID: "feedback"},
			{Name: "feedback_completed", DeviceID: "observed", SessionID: "feedback"},
			{Name: "page_view", DeviceID: "merely-eligible", Metadata: map[string]interface{}{"feedback_available": true}},
		},
	}
	if err := loadMatchdayMarketing(context.Background(), f.conn, f.org.ID, map[int]*matchdayFacts{f.event: facts}, secret); err != nil {
		t.Fatal(err)
	}
	for code, want := range map[string]float64{
		"sponsor_impressions": 3, "sponsor_reach": 1, "sponsor_viewable": 1, "sponsor_viewability": 50,
		"sponsor_frequency": 3, "sponsor_ctr": 100.0 / 3, "sponsor_avg_visible_time": 900, "sponsor_placements": 3,
		"feedback_responses": 2, "feedback_answers": 8, "feedback_completed": 1, "feedback_respondents": 1,
		"feedback_response_rate": 100, "feedback_completion_rate": 100, "feedback_satisfaction": 100,
	} {
		checkMatchdayDomainMetric(t, facts, code, want)
	}
	if rows := facts.Metrics["sponsor_impressions"].Rows; len(rows) != 2 {
		t.Fatalf("per-sponsor rows = %+v", rows)
	}
	linked := false
	for _, action := range facts.Actions {
		if action.Feature == "sponsor_click" && action.Identity.Device == browser && action.Count == 1 {
			linked = true
		}
	}
	if !linked {
		t.Fatalf("measured sponsor click should link to known device, actions=%+v", facts.Actions)
	}
}

func checkMatchdayDomainMetric(t *testing.T, f *matchdayFacts, code string, want float64) {
	t.Helper()
	metric := f.Metrics[code]
	if metric.Value == nil || math.Abs(*metric.Value-want) > .011 {
		t.Fatalf("%s = %+v, want %.3f", code, metric, want)
	}
}
