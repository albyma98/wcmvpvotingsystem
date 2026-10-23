package api

import (
	"net/http"
	"testing"
)

func reportCombinedEvidence(id, name, device string, fan int, metadata map[string]interface{}) reportSignal {
	s := reportTestSignal(id, name, device, fan, "10", metadata)
	if s.Metadata == nil {
		s.Metadata = map[string]interface{}{}
	}
	s.Metadata["_operational"] = true
	return s
}

func TestEngagementReportCombinedOperationalOnlyRecoversFactsWithoutFalseAudiences(t *testing.T) {
	op := reportOperationalData{Signals: []reportSignal{
		reportCombinedEvidence("vote", "vote_completed", "a", 7, nil),
		reportCombinedEvidence("r1", "game_completed", "a", 7, map[string]interface{}{"game_id": "quiz", "round_id": "r1"}),
		reportCombinedEvidence("r2", "game_completed", "a", 7, map[string]interface{}{"game_id": "quiz", "round_id": "r2"}),
		reportCombinedEvidence("legacy", "game_completed", "a", 7, map[string]interface{}{"game_id": "quiz", "round_id": "legacy:quiz", "_legacy_round": true}),
		reportCombinedEvidence("r3", "game_completed", "b", 0, map[string]interface{}{"game_id": "reaction", "round_id": "r3"}),
		reportCombinedEvidence("shop", "redemption_completed", "a", 7, nil),
	}, Summary: reportOperationalSummary{Votes: 1, GameRewardReceipts: 4, ShopRedemptions: 1, FeedbackResponses: 2}, Feedback: reportOperationalFeedback{PerksResponses: 2, PerksPositive: 1}}
	prior := reportOperationalData{Signals: []reportSignal{reportCombinedEvidence("old", "vote_completed", "old-device", 7, nil)}}
	facts, details, _ := calculateCombinedReportMetrics(nil, nil, op, prior)
	for key, want := range map[string]float64{"visitors": 2, "active_visitors": 2, "voters": 1, "players": 2, "rounds_completed": 3, "repeat_players": 1, "multi_experience_visitors": 1, "redeemers": 1, "returning_profiles": 1, "previous_event_profiles": 1} {
		reportTestFact(t, facts, key, want)
	}
	for _, key := range []string{"engagement_eligible_visitors", "active_seconds", "timed_visitors", "vote_eligible_visitors", "game_eligible_visitors", "new_profiles", "registration_starters", "shop_eligible_profiles", "sponsor_reach", "sponsor_impressions", "feedback_respondents", "experience_responses"} {
		if facts[key] != nil {
			t.Errorf("invented %s=%v", key, *facts[key])
		}
	}
	reportTestKPI(t, facts, "active_participation", nil)
	reportTestKPI(t, facts, "multi_experience", reportMetricFloat(50))
	reportTestKPI(t, facts, "perks_interest", reportMetricFloat(50))
	reportTestKPI(t, facts, "event_retention", reportMetricFloat(100))
	for _, k := range buildCombinedReportKPIs(facts, details, nil, nil, op, prior) {
		if k.ID == "visitors" && (k.Source != "operational" || k.Label != "Partecipanti digitali osservabili") {
			t.Fatalf("wrong provenance: %+v", k)
		}
	}
}

func TestEngagementReportCombinedLegacyCompletionSurvivesModernStartWithoutInventingReplay(t *testing.T) {
	op := reportOperationalData{Signals: []reportSignal{
		reportCombinedEvidence("answer", "game_started", "a", 7, map[string]interface{}{"game_id": "quiz", "round_id": "modern"}),
		reportCombinedEvidence("legacy", "game_completed", "a", 7, map[string]interface{}{"game_id": "quiz", "round_id": "legacy:quiz", "_legacy_round": true}),
	}}
	facts, details, _ := calculateCombinedReportMetrics(nil, nil, op, reportOperationalData{})
	reportTestFact(t, facts, "rounds_completed", 1)
	reportTestFact(t, facts, "repeat_players", 0)
	if len(details.Games) != 1 || details.Games[0].RoundsStarted != 1 || details.Games[0].RoundsCompleted != 1 {
		t.Fatalf("legacy proof was lost or doubled %+v", details.Games)
	}
}

func TestEngagementReportCombinedMixedDeduplicatesAndUsesMeasuredTimeSample(t *testing.T) {
	signals := []reportSignal{
		reportTestSignal("page", "page_view", "a", 0, "01", map[string]interface{}{"games_available": true, "feedback_available": true}),
		reportTestSignal("round", "game_completed", "a", 0, "10", map[string]interface{}{"game_id": "quiz", "round_id": "same-round"}),
		reportTestSignal("time", "active_time", "a", 0, "15", map[string]interface{}{"seconds": 15}),
		reportTestSignal("feedback", "feedback_completed", "a", 0, "20", map[string]interface{}{"experience": "good", "perks_interest": "yes"}),
		reportTestSignal("view", "sponsor_view", "a", 0, "05", map[string]interface{}{"sponsor_id": 1}),
		reportTestSignal("click", "sponsor_click", "a", 0, "06", map[string]interface{}{"sponsor_id": 1}),
	}
	op := reportOperationalData{Signals: []reportSignal{
		reportCombinedEvidence("round", "game_completed", "a", 7, map[string]interface{}{"game_id": "quiz", "round_id": "same-round"}),
		reportCombinedEvidence("legacy", "game_completed", "a", 7, map[string]interface{}{"game_id": "quiz", "round_id": "legacy:quiz", "_legacy_round": true}),
		reportCombinedEvidence("click", "sponsor_click", "a", 7, map[string]interface{}{"sponsor_id": 1}),
		// Deliberately collides with a browser signal ID: source namespaces
		// must keep the independently stored vote, not discard it.
		reportCombinedEvidence("page", "vote_completed", "b", 0, nil),
	}, Summary: reportOperationalSummary{FeedbackResponses: 2}, Feedback: reportOperationalFeedback{PerksResponses: 2, PerksPositive: 1}}
	facts, details, _ := calculateCombinedReportMetrics(signals, nil, op, reportOperationalData{})
	for key, want := range map[string]float64{"visitors": 2, "players": 1, "rounds_completed": 1, "repeat_players": 0, "voters": 1, "sponsor_reach": 1, "sponsor_clickers": 1, "active_seconds": 15, "timed_visitors": 1} {
		reportTestFact(t, facts, key, want)
	}
	reportTestKPI(t, facts, "active_time", reportMetricFloat(15))
	reportTestKPI(t, facts, "sponsor_ctr", reportMetricFloat(100))
	reportTestKPI(t, facts, "perks_interest", reportMetricFloat(50))
	reportTestKPI(t, facts, "feedback_response", reportMetricFloat(100))
	reportTestKPI(t, facts, "satisfaction", reportMetricFloat(100))
	for _, k := range buildCombinedReportKPIs(facts, details, signals, nil, op, reportOperationalData{}) {
		if (k.ID == "visitors" || k.ID == "completed_rounds" || k.ID == "sponsor_ctr") && k.Source != "mixed" {
			t.Errorf("lost mixed provenance %+v", k)
		}
	}
}

func TestEngagementReportPrepareAndApprovalIncludeOperationalDataWithoutSignals(t *testing.T) {
	f := newEngagementReportFixture(t)
	if _, err := f.conn.Exec(`INSERT INTO fan_minigame_rewards(event_id,organization_id,request_id,device_id,reward_key,coins) VALUES(?,?,'old-round','old-browser','quiz',5)`, f.event, f.org.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.conn.Exec(`INSERT INTO event_feedback(event_id,experience,team_spirit,perks_interest,mini_games_interest) VALUES(?,'very_easy','high','yes','maybe')`, f.event); err != nil {
		t.Fatal(err)
	}
	draft := engagementReportDecodeSnapshot(t, engagementReportRequest(t, f.router.prepareEngagementReport, http.MethodGet, f.event, nil, f.ctx, nil))
	reportTestFact(t, draft.Facts, "visitors", 1)
	reportTestFact(t, draft.Facts, "rounds_completed", 1)
	if draft.Details.Operational == nil || draft.Details.Operational.GameRewardReceipts != 1 || draft.Details.Operational.FeedbackResponses != 1 || len(draft.Details.Sources) < 14 || len(draft.Details.CoverageNotes) == 0 {
		t.Fatalf("missing operational diagnostics %+v", draft.Details)
	}
	approved := engagementReportDecodeSnapshot(t, engagementReportRequest(t, f.router.approveEngagementReport, http.MethodPost, f.event, map[string]interface{}{"calculation_id": draft.CalculationID, "format": "one_page"}, f.ctx, nil))
	for _, k := range approved.KPIs {
		if (k.ID == "visitors" || k.ID == "completed_rounds" || k.ID == "perks_interest") && (k.Value == nil || k.Source != "operational" || k.Corrected) {
			t.Errorf("approval changed operational provenance %+v", k)
		}
	}
	if approved.Details.Operational == nil || approved.Details.Operational.GameRewardReceipts != 1 {
		t.Fatal("approved snapshot lost operational details")
	}
}
