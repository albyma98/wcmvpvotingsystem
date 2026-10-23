package api

import (
	"encoding/json"
	"math"
	"testing"
)

func reportTestSignal(id, name, device string, fan int, second string, metadata map[string]interface{}) reportSignal {
	return reportSignal{ID: id, Name: name, DeviceID: device, FanID: fan, SessionID: "session-" + device, OccurredAt: "2026-10-23T12:00:" + second + "Z", Metadata: metadata}
}

func reportTestFact(t *testing.T, facts map[string]*float64, key string, want float64) {
	t.Helper()
	if facts[key] == nil || math.Abs(*facts[key]-want) > 0.000001 {
		t.Fatalf("%s = %v, want %v", key, facts[key], want)
	}
}

func reportTestKPI(t *testing.T, facts map[string]*float64, id string, want *float64) {
	t.Helper()
	for _, kpi := range buildReportKPIs(facts) {
		if kpi.ID != id {
			continue
		}
		if want == nil {
			if kpi.Value != nil || kpi.Source != "unavailable" {
				t.Fatalf("%s should be unavailable: %+v", id, kpi)
			}
		} else if kpi.Value == nil || math.Abs(*kpi.Value-*want) > 0.000001 {
			t.Fatalf("%s = %+v, want %v", id, kpi, *want)
		}
		return
	}
	t.Fatalf("KPI %s missing", id)
}

func TestEngagementReportMetricsNoSignalsIsUnavailable(t *testing.T) {
	facts, details, startedAt := calculateReportMetrics(nil, nil)
	for key, value := range facts {
		if value != nil {
			t.Errorf("unobserved fact %s = %v", key, *value)
		}
	}
	if startedAt != "" || details.Games == nil || details.Sponsors == nil || len(details.Games)+len(details.Sponsors) != 0 {
		t.Fatalf("unexpected empty report: %+v %q", details, startedAt)
	}
	kpis := buildReportKPIs(facts)
	if len(kpis) != 20 {
		t.Fatalf("got %d KPIs", len(kpis))
	}
	seen := map[string]bool{}
	for _, kpi := range kpis {
		if seen[kpi.ID] || kpi.Value != nil || kpi.Source != "unavailable" || kpi.Label == "" || kpi.Description == "" {
			t.Errorf("invalid definition: %+v", kpi)
		}
		seen[kpi.ID] = true
	}
	if _, err := json.Marshal(kpis); err != nil {
		t.Fatal(err)
	}
}

func TestEngagementReportMetricsIdentityAndEligibility(t *testing.T) {
	page := reportTestSignal("page", "page_view", "phone", 0, "01", map[string]interface{}{"vote_available": false, "source": "arena"})
	vote := reportTestSignal("vote", "vote_completed", "phone", 7, "02", nil)
	facts, _, startedAt := calculateReportMetrics([]reportSignal{vote, page, vote}, nil)
	reportTestFact(t, facts, "visitors", 1)
	reportTestFact(t, facts, "active_visitors", 1)
	reportTestFact(t, facts, "voters", 1)
	reportTestFact(t, facts, "vote_eligible_visitors", 1)
	reportTestFact(t, facts, "onsite_visitors", 1)
	if facts["game_eligible_visitors"] != nil || facts["attendees"] != nil || startedAt != "2026-10-23T12:00:01Z" {
		t.Fatalf("invented denominator or bad timestamp: %+v %q", facts, startedAt)
	}
	reportTestKPI(t, facts, "mvp_participation", reportMetricFloat(100))
	reportTestKPI(t, facts, "venue_adoption", nil)
	// A shared device cannot merge anonymous actions into either of two people.
	other := reportTestSignal("other", "page_view", "phone", 8, "03", nil)
	facts, _, _ = calculateReportMetrics([]reportSignal{page, vote, other}, nil)
	reportTestFact(t, facts, "visitors", 3)
	// A successful vote proves a visit, but no eligibility flag means unknown.
	facts, _, _ = calculateReportMetrics([]reportSignal{vote}, nil)
	reportTestFact(t, facts, "visitors", 1)
	if facts["vote_eligible_visitors"] != nil {
		t.Fatal("vote-only telemetry invented an eligible audience")
	}
	reportTestKPI(t, facts, "mvp_participation", nil)
}

func TestEngagementReportMetricsActiveTimeMergesTabsAndCapsHeartbeat(t *testing.T) {
	span := func(id, device string, fan int, start, end string, seconds float64) reportSignal {
		return reportTestSignal(id, "active_time", device, fan, "59", map[string]interface{}{
			"started_at": "2026-10-23T12:00:" + start + "Z", "ended_at": "2026-10-23T12:00:" + end + "Z", "seconds": seconds,
		})
	}
	signals := []reportSignal{
		span("a", "a", 3, "00", "15", 15),
		span("b", "b", 3, "10", "25", 15),
		span("c", "c", 4, "00", "59", 999),
		span("negative", "c", 4, "25", "30", -5),
		span("inverted", "c", 4, "58", "57", 5),
	}
	facts, _, _ := calculateReportMetrics(signals, nil)
	reportTestFact(t, facts, "visitors", 2)
	reportTestFact(t, facts, "active_seconds", 40)
	reportTestKPI(t, facts, "active_time", reportMetricFloat(20))
}

func TestEngagementReportMetricsRoundsDeduplicateAndRecoverMissingStarts(t *testing.T) {
	game := func(id, name, device, round string) reportSignal {
		return reportTestSignal(id, name, device, 0, "10", map[string]interface{}{"game_id": "quiz", "round_id": round, "name": "Quiz Lampo"})
	}
	signals := []reportSignal{
		game("start", "game_started", "a", "r1"),
		game("finish", "game_completed", "a", "r1"),
		game("finish-retry", "game_completed", "a", "r1"),
		game("recovered", "game_completed", "a", "r2"),
		game("second-player", "game_started", "b", "r3"),
		reportTestSignal("vote", "vote_completed", "a", 0, "12", nil),
		reportTestSignal("page", "page_view", "a", 0, "01", map[string]interface{}{"games_available": true}),
	}
	facts, details, _ := calculateReportMetrics(signals, nil)
	reportTestFact(t, facts, "players", 2)
	reportTestFact(t, facts, "rounds_completed", 2)
	reportTestFact(t, facts, "repeat_players", 1)
	reportTestFact(t, facts, "multi_experience_visitors", 1)
	reportTestKPI(t, facts, "replay_rate", reportMetricFloat(50))
	reportTestKPI(t, facts, "game_participation", reportMetricFloat(100))
	if len(details.Games) != 1 || details.Games[0].Players != 2 || details.Games[0].RoundsStarted != 3 || details.Games[0].RoundsCompleted != 2 {
		t.Fatalf("round detail: %+v", details.Games)
	}
}

func TestEngagementReportMetricsSponsorClicksRequireSameSponsorExposure(t *testing.T) {
	sponsorSignal := func(id, name, device string, sponsor int, placement string) reportSignal {
		return reportTestSignal(id, name, device, 0, "10", map[string]interface{}{"sponsor_id": sponsor, "name": "Sponsor", "placement": placement, "game_id": "quiz", "round_id": id})
	}
	signals := []reportSignal{
		sponsorSignal("a-view", "sponsor_view", "a", 1, "footer"),
		sponsorSignal("a-view-retry", "sponsor_view", "a", 1, "footer"),
		sponsorSignal("a-click-unexposed", "sponsor_click", "a", 2, "footer"),
		sponsorSignal("b-view", "sponsor_view", "b", 2, "footer"),
		sponsorSignal("b-click", "sponsor_click", "b", 2, "footer"),
		sponsorSignal("a-game-view", "sponsored_game_view", "a", 1, "game"),
		sponsorSignal("a-wrong-game", "game_started", "a", 2, "game"),
		sponsorSignal("a-right-game", "game_started", "a", 1, "game"),
	}
	facts, details, _ := calculateReportMetrics(signals, nil)
	reportTestFact(t, facts, "sponsor_reach", 2)
	reportTestFact(t, facts, "sponsor_clickers", 1)
	reportTestFact(t, facts, "sponsor_impressions", 3)
	reportTestFact(t, facts, "sponsored_players", 1)
	reportTestFact(t, facts, "sponsored_exposed_visitors", 1)
	reportTestKPI(t, facts, "sponsor_ctr", reportMetricFloat(50))
	reportTestKPI(t, facts, "sponsored_participation", reportMetricFloat(100))
	if len(details.Sponsors) != 2 || details.Sponsors[0].Clickers != 0 || details.Sponsors[1].Clickers != 1 {
		t.Fatalf("sponsor detail: %+v", details.Sponsors)
	}
}

func TestEngagementReportMetricsFeedbackAndFunnelRecovery(t *testing.T) {
	signals := []reportSignal{
		reportTestSignal("late-feedback", "feedback_completed", "a", 1, "20", map[string]interface{}{"experience": "bad", "perks_interest": "no"}),
		reportTestSignal("first-feedback", "feedback_completed", "a", 1, "10", map[string]interface{}{"experience": "good", "perks_interest": "yes"}),
		reportTestSignal("other-feedback", "feedback_completed", "b", 2, "10", map[string]interface{}{"experience": "awesome", "perks_interest": "maybe"}),
		reportTestSignal("start", "registration_started", "a", 0, "01", nil),
		reportTestSignal("complete", "registration_completed", "a", 1, "05", nil),
		reportTestSignal("recover-complete", "registration_completed", "b", 2, "05", nil),
		reportTestSignal("anonymous-complete", "registration_completed", "c", 0, "05", nil),
		reportTestSignal("page", "page_view", "a", 1, "02", map[string]interface{}{"feedback_available": true}),
		reportTestSignal("redeem", "redemption_completed", "b", 2, "10", nil),
		reportTestSignal("guest-shop", "shop_view", "c", 0, "10", nil),
	}
	facts, _, _ := calculateReportMetrics(signals, nil)
	reportTestFact(t, facts, "new_profiles", 2)
	reportTestFact(t, facts, "registration_starters", 2)
	reportTestFact(t, facts, "shop_eligible_profiles", 2)
	reportTestFact(t, facts, "feedback_respondents", 2)
	reportTestKPI(t, facts, "registration_conversion", reportMetricFloat(100))
	reportTestKPI(t, facts, "redemption_conversion", reportMetricFloat(50))
	reportTestKPI(t, facts, "feedback_response", reportMetricFloat(100))
	reportTestKPI(t, facts, "satisfaction", reportMetricFloat(100))
	reportTestKPI(t, facts, "perks_interest", reportMetricFloat(50))
}

func TestEngagementReportMetricsRetentionExcludesAnonymousCrossEventMatching(t *testing.T) {
	previous := []reportSignal{
		reportTestSignal("old-vote", "vote_completed", "a", 1, "01", nil),
		reportTestSignal("old-game", "game_started", "b", 2, "01", map[string]interface{}{"game_id": "quiz", "round_id": "old"}),
		reportTestSignal("old-guest", "vote_completed", "shared", 0, "01", nil),
	}
	current := []reportSignal{
		reportTestSignal("current-vote", "vote_completed", "new-device", 1, "01", nil),
		reportTestSignal("current-guest", "vote_completed", "shared", 0, "01", nil),
		reportTestSignal("current-page", "page_view", "b", 2, "01", nil),
	}
	facts, _, _ := calculateReportMetrics(current, previous)
	reportTestFact(t, facts, "previous_event_profiles", 2)
	reportTestFact(t, facts, "returning_profiles", 1)
	reportTestKPI(t, facts, "event_retention", reportMetricFloat(50))
	facts, _, _ = calculateReportMetrics(current, nil)
	reportTestKPI(t, facts, "event_retention", nil)
	// A tracked audience with no meaningful actions has an observed zero.
	facts, _, _ = calculateReportMetrics(current[2:], nil)
	reportTestKPI(t, facts, "active_participation", reportMetricFloat(0))
}

func TestEngagementReportMetricsRetentionLinksPriorGuestOnlyInsidePriorEvent(t *testing.T) {
	previous := []reportSignal{
		reportTestSignal("old-anonymous-vote", "vote_completed", "old-browser", 0, "01", nil),
		reportTestSignal("old-verified-profile", "registration_completed", "old-browser", 7, "02", nil),
	}
	current := []reportSignal{reportTestSignal("new-identified-vote", "vote_completed", "different-browser", 7, "01", nil)}
	facts, _, _ := calculateReportMetrics(current, previous)
	reportTestFact(t, facts, "previous_event_profiles", 1)
	reportTestFact(t, facts, "returning_profiles", 1)
	previous = append(previous, reportTestSignal("old-shared-browser", "page_view", "old-browser", 8, "03", nil))
	facts, _, _ = calculateReportMetrics(current, previous)
	reportTestFact(t, facts, "previous_event_profiles", 0)
	reportTestFact(t, facts, "returning_profiles", 0)
}
