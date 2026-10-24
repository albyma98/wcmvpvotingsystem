package api

import (
	"encoding/json"
	"testing"
	"time"
)

func reportTimeTestSignal(id, name, device string, fanID, offset int, meta map[string]interface{}) reportSignal {
	at := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC).Add(time.Duration(offset) * time.Second)
	return reportSignal{ID: id, Name: name, DeviceID: device, FanID: fanID, SessionID: "session-" + device, OccurredAt: at.Format(time.RFC3339), Metadata: meta}
}

func TestEngagementReportTimeEstimateCapsAndExcludesLongGaps(t *testing.T) {
	signals := []reportSignal{
		reportTimeTestSignal("page", "page_view", "a", 0, 0, nil),
		reportTimeTestSignal("game", "game_started", "a", 0, 120, map[string]interface{}{"game_id": "quiz", "round_id": "r1"}),
		reportTimeTestSignal("vote", "vote_completed", "a", 0, 430, nil),
		reportTimeTestSignal("shop", "shop_view", "a", 0, 440, nil),
		reportTimeTestSignal("singleton", "page_view", "b", 0, 0, nil),
	}
	facts, details, _ := calculateReportMetrics(signals, nil)
	reportTestFact(t, facts, "estimated_seconds", 70)
	reportTestFact(t, facts, "estimated_timed_visitors", 1)
	reportTestKPI(t, facts, "estimated_avg_time", reportMetricFloat(70))
	got := details.TimeEstimate
	if got == nil || got.CoveredVisitors != 1 || got.TotalVisitors != 2 || got.ExcludedVisitors != 1 || got.GapLimitSeconds != 300 || got.IntervalCapSeconds != 60 || got.MeasuredSeconds != 0 || got.InferredSeconds != 70 {
		t.Fatalf("bad coverage: %+v", got)
	}
	// A five-minute gap is accepted, but contributes only the first minute.
	_, estimate := calculateReportTimeEstimate([]reportSignal{signals[0], reportTimeTestSignal("at-limit", "shop_view", "a", 0, 300, nil)})
	if estimate.InferredSeconds != 60 {
		t.Fatalf("limit gap: %+v", estimate)
	}
	facts, _, _ = calculateReportMetrics([]reportSignal{signals[0], reportTimeTestSignal("over-limit", "shop_view", "a", 0, 301, nil)}, nil)
	if facts["estimated_seconds"] != nil || facts["estimated_timed_visitors"] != nil {
		t.Fatal("long gap invented a duration")
	}
}

func TestEngagementReportTimeEstimateMergesMeasuredAndInferredAcrossDevices(t *testing.T) {
	span := func(id, device string, start, end int) reportSignal {
		s := reportTimeTestSignal(id, "active_time", device, 7, end, nil)
		s.Metadata = map[string]interface{}{"seconds": end - start, "started_at": reportTimeTestSignal("", "", "", 0, start, nil).OccurredAt, "ended_at": s.OccurredAt}
		return s
	}
	signals := []reportSignal{
		reportTimeTestSignal("page", "page_view", "phone", 0, 0, nil),
		reportTimeTestSignal("vote", "vote_completed", "phone", 7, 20, nil),
		span("phone-time", "phone", 10, 25),
		span("laptop-time", "laptop", 20, 35),
	}
	facts, details, _ := calculateReportMetrics(signals, nil)
	reportTestFact(t, facts, "visitors", 1)
	reportTestFact(t, facts, "identified_visitors", 1)
	reportTestFact(t, facts, "guest_visitors", 0)
	reportTestFact(t, facts, "estimated_seconds", 35)
	if details.TimeEstimate.MeasuredSeconds != 25 || details.TimeEstimate.InferredSeconds != 10 || details.TimeEstimate.CoveredVisitors != 1 {
		t.Fatalf("overlap not merged: %+v", details.TimeEstimate)
	}
}

func TestEngagementReportTimeEstimateUnknownDurationIsNotZero(t *testing.T) {
	missing := reportTimeTestSignal("missing-time", "game_started", "missing", 0, 0, map[string]interface{}{"game_id": "quiz"})
	missing.OccurredAt = ""
	signals := []reportSignal{reportTimeTestSignal("one", "vote_completed", "singleton", 0, 0, nil), missing}
	facts, details, _ := calculateReportMetrics(signals, nil)
	if facts["estimated_seconds"] != nil || facts["estimated_timed_visitors"] != nil || details.TimeEstimate.CoveredVisitors != 0 || details.TimeEstimate.ExcludedVisitors != 2 {
		t.Fatalf("unknown duration was treated as zero: %+v %+v", facts, details.TimeEstimate)
	}
	reportTestKPI(t, facts, "estimated_total_time", nil)
	reportTestKPI(t, facts, "estimated_avg_time", nil)
	// A valid heartbeat alone is sufficient evidence for its measured interval.
	heartbeat := reportTimeTestSignal("heartbeat", "active_time", "a", 0, 15, map[string]interface{}{"seconds": 15})
	facts, _, _ = calculateReportMetrics([]reportSignal{heartbeat}, nil)
	reportTestFact(t, facts, "estimated_seconds", 15)
}

func TestEngagementReportTimeEstimateDeduplicatesReceiptsAndRetainsResolvedSteps(t *testing.T) {
	raw := func(id, table, name string, offset int, meta map[string]interface{}) reportSignal {
		meta["_operational"], meta["_table"] = true, table
		return reportTimeTestSignal(id, name, "a", 0, offset, meta)
	}
	signals := []reportSignal{
		reportTimeTestSignal("page", "page_view", "a", 0, 0, nil),
		reportTimeTestSignal("start", "game_started", "a", 0, 10, map[string]interface{}{"game_id": "quiz", "round_id": "r1"}),
		reportTimeTestSignal("complete", "game_completed", "a", 0, 30, map[string]interface{}{"game_id": "quiz", "round_id": "r1"}),
		raw("receipt", "fan_minigame_rewards", "game_completed", 95, map[string]interface{}{"game_id": "quiz", "round_id": "r1"}),
		raw("credit", "fan_wallet_movements", "game_started", 100, map[string]interface{}{"game_id": "quiz"}),
	}
	seconds, estimate := calculateReportTimeEstimate(signals)
	if seconds != 30 || estimate.InferredSeconds != 30 {
		t.Fatalf("duplicate reward inflated duration: %v %+v", seconds, estimate)
	}
	steps := []reportSignal{
		raw("session", "playmaker_sessions", "game_started", 0, map[string]interface{}{"game_id": "playmaker", "round_id": "session"}),
		raw("pass:1", "playmaker_passes", "game_started", 10, map[string]interface{}{"game_id": "playmaker", "round_id": "session"}),
		raw("pass:2", "playmaker_passes", "game_started", 40, map[string]interface{}{"game_id": "playmaker", "round_id": "session"}),
	}
	seconds, _ = calculateReportTimeEstimate(steps)
	if seconds != 40 {
		t.Fatalf("resolved passes lost: %v", seconds)
	}
	// Browser/native copies of a vote cannot form an interval on their own.
	votes := []reportSignal{
		reportTimeTestSignal("vote", "vote_completed", "a", 0, 10, nil),
		raw("stored-vote", "votes", "vote_completed", 11, map[string]interface{}{}),
		raw("vote-credit", "fan_wallet_movements", "vote_completed", 20, map[string]interface{}{}),
	}
	seconds, estimate = calculateReportTimeEstimate(votes)
	if seconds != 0 || estimate.CoveredVisitors != 0 {
		t.Fatalf("one vote became multiple actions: %v %+v", seconds, estimate)
	}
	// Permutation and repeated source rows cannot change the result.
	baseline, _ := json.Marshal(estimate)
	seconds, reversed := calculateReportTimeEstimate([]reportSignal{votes[2], votes[1], votes[0], votes[1]})
	actual, _ := json.Marshal(reversed)
	if seconds != 0 || string(baseline) != string(actual) {
		t.Fatalf("unstable estimate: %s != %s", baseline, actual)
	}
}

func TestEngagementReportCombinedCommercialTotalsPreferOperationalRecords(t *testing.T) {
	signals := []reportSignal{
		reportTimeTestSignal("click", "sponsor_click", "a", 0, 0, map[string]interface{}{"sponsor_id": 1}),
		reportTimeTestSignal("shop", "redemption_completed", "a", 4, 10, map[string]interface{}{"item_key": "match_card"}),
		reportTimeTestSignal("feedback", "feedback_completed", "a", 4, 20, map[string]interface{}{"experience": "good"}),
		reportTimeTestSignal("registered", "registration_completed", "a", 4, 5, nil),
		reportTimeTestSignal("guest", "page_view", "b", 0, 0, nil),
	}
	operational := reportOperationalData{Summary: reportOperationalSummary{SponsorClicks: 9, SponsorClicksWithoutIdentity: 8, ShopRedemptions: 3, FeedbackResponses: 6}}
	facts, details, _ := calculateCombinedReportMetrics(signals, nil, operational, reportOperationalData{})
	reportTestFact(t, facts, "total_sponsor_clicks", 9)
	reportTestFact(t, facts, "shop_redemptions", 3)
	reportTestFact(t, facts, "feedback_collected", 6)
	reportTestFact(t, facts, "identified_visitors", 1)
	reportTestFact(t, facts, "guest_visitors", 1)
	reportTestFact(t, facts, "sponsor_interactors", 1)
	reportTestKPI(t, facts, "audience_registration_conversion", reportMetricFloat(50))
	if facts["sponsor_engaged_exposed"] != nil {
		t.Fatal("anonymous operational clicks invented sponsor exposures")
	}
	for _, kpi := range buildCombinedReportKPIs(facts, details, signals, nil, operational, reportOperationalData{}) {
		if (kpi.ID == "sponsor_clicks" || kpi.ID == "shop_redemptions" || kpi.ID == "feedback_collected") && kpi.Source != "operational" {
			t.Fatalf("wrong total provenance: %+v", kpi)
		}
		if (kpi.ID == "estimated_total_time" || kpi.ID == "estimated_avg_time") && kpi.Source != "estimated" {
			t.Fatalf("estimate labelled as measured: %+v", kpi)
		}
	}
	// Without a saved total, browser observations remain a documented minimum.
	facts, _, _ = calculateCombinedReportMetrics(signals, nil, reportOperationalData{}, reportOperationalData{})
	reportTestFact(t, facts, "total_sponsor_clicks", 1)
	reportTestFact(t, facts, "shop_redemptions", 1)
	reportTestFact(t, facts, "feedback_collected", 1)
}

func TestEngagementReportTimeEstimateUndatedProofDoesNotHideDatedEvidence(t *testing.T) {
	page := reportTimeTestSignal("page", "page_view", "a", 0, 0, nil)
	answer := reportTimeTestSignal("answer", "game_started", "a", 0, 0, map[string]interface{}{"_operational": true, "_table": "fan_minigame_quiz_answers", "game_id": "quiz", "round_id": "r1"})
	answer.OccurredAt = ""
	legacy := reportTimeTestSignal("legacy", "game_completed", "a", 0, 30, map[string]interface{}{"_operational": true, "_table": "fan_journey_rewards", "_legacy_round": true, "game_id": "quiz", "round_id": "legacy:quiz"})
	credit := reportTimeTestSignal("credit", "game_started", "a", 0, 30, map[string]interface{}{"_operational": true, "_table": "fan_wallet_movements", "game_id": "quiz"})
	for _, dated := range []reportSignal{legacy, credit} {
		seconds, estimate := calculateReportTimeEstimate([]reportSignal{page, answer, dated})
		if seconds != 30 || estimate.CoveredVisitors != 1 {
			t.Fatalf("undated answer hid %s timestamp: %v %+v", dated.ID, seconds, estimate)
		}
	}
	start := reportTimeTestSignal("start", "game_started", "a", 0, 10, map[string]interface{}{"game_id": "quiz", "round_id": "r1"})
	seconds, _ := calculateReportTimeEstimate([]reportSignal{start, legacy})
	if seconds != 20 {
		t.Fatalf("modern start hid legacy completion: %v", seconds)
	}
	// A saved current vote can be updated later than its browser action. Prefer
	// the native action timestamp, just as for completion receipts, for a
	// conservative interval instead of inflating it with the delayed copy.
	vote := reportTimeTestSignal("vote", "vote_completed", "a", 0, 10, nil)
	savedVote := reportTimeTestSignal("stored-vote", "vote_completed", "a", 0, 100, map[string]interface{}{"_operational": true, "_table": "votes"})
	seconds, _ = calculateReportTimeEstimate([]reportSignal{page, vote, savedVote})
	if seconds != 10 {
		t.Fatalf("delayed vote copy inflated duration: %v", seconds)
	}
}
