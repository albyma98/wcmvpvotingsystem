package api

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
)

func TestEngagementReportOperationalSourcesReadEventFactsWithoutInventingTelemetry(t *testing.T) {
	f := newEngagementReportFixture(t)
	exec := func(query string, args ...interface{}) int64 {
		t.Helper()
		result, err := f.conn.Exec(query, args...)
		if err != nil {
			t.Fatal(err)
		}
		id, _ := result.LastInsertId()
		return id
	}
	fan := exec(`INSERT INTO fan_profiles(organization_id,nickname,phone,registration_pending) VALUES(?,'Report fan','+39111',0)`, f.org.ID)
	player := exec(`INSERT INTO players(organization_id,team_id,first_name,last_name,role) VALUES(?,?,'First','Last','player')`, f.org.ID, f.org.TeamID)
	sponsor := exec(`INSERT INTO sponsors(organization_id,position,name,report_name,logo_data) VALUES(?,1,'Partner','Report partner','logo')`, f.org.ID)
	foreignSponsor := exec(`INSERT INTO sponsors(organization_id,position,name,logo_data) VALUES(?,1,'Other partner','logo')`, f.other.ID)
	exec(`INSERT INTO votes(event_id,player_id,device_id,user_id,ticket_code,ticket_signature) VALUES(?,?,?,?,'report-ticket','signature')`, f.event, player, "browser", fan)
	exec(`INSERT INTO fan_minigame_rewards(event_id,organization_id,request_id,device_id,fan_id,reward_key,coins) VALUES(?,?,'reaction-round','browser',?,'reaction',5)`, f.event, f.org.ID, fan)
	exec(`INSERT INTO fan_minigame_rewards(event_id,organization_id,request_id,device_id,reward_key,coins) VALUES(?,?,'zero-round','zero-browser','tap',0)`, f.event, f.org.ID)
	exec(`INSERT INTO fan_minigame_rewards(event_id,organization_id,request_id,device_id,reward_key,coins) VALUES(?,?,'other-product','other-browser','unrelated',50)`, f.event, f.org.ID)
	exec(`INSERT INTO fan_journey_rewards(event_id,organization_id,device_id,fan_id,reward_key,coins) VALUES(?,?,'browser',?,'quiz',10)`, f.event, f.org.ID, fan)
	for _, question := range []int{1, 2} {
		exec(`INSERT INTO fan_minigame_quiz_answers(event_id,organization_id,device_id,request_id,question_id) VALUES(?,?,'browser','quiz-round',?)`, f.event, f.org.ID, question)
	}
	exec(`INSERT INTO fan_journey_quiz_answers(event_id,organization_id,device_id,question_id) VALUES(?,?,'quiz-guest',3)`, f.event, f.org.ID)
	now := time.Now().UnixMilli()
	for attempt, status := range []string{"pending", "made", "miss"} {
		exec(`INSERT INTO basketball_shots(shot_id,event_id,organization_id,device_id,fan_id,attempt,hoop_x,hoop_y,issued_at,expires_at,status) VALUES(?,?,?,'browser',?,?,0,0,?,?,?)`, fmt.Sprintf("shot-%d", attempt), f.event, f.org.ID, fan, attempt, now, now+1000, status)
	}
	exec(`INSERT INTO playmaker_sessions(session_id,event_id,organization_id,device_id,fan_id,started_at,ends_at,status) VALUES('playmaker-round',?,?,'browser',?,?,?,'closed')`, f.event, f.org.ID, fan, now, now+30000)
	exec(`INSERT INTO playmaker_passes(session_id,attempt,event_id,organization_id,device_id,fan_id,elapsed_ms,reason,resolved_at) VALUES('playmaker-round',0,?,?,'browser',?,100,'resolved',?)`, f.event, f.org.ID, fan, now+100)
	exec(`INSERT INTO crossover_sessions(session_id,event_id,organization_id,device_id,fan_id,started_at,ends_at) VALUES('crossover-round',?,?,'browser',?,?,?)`, f.event, f.org.ID, fan, now, now+25000)
	exec(`INSERT INTO crossover_duels(session_id,duel,event_id,organization_id,device_id,fan_id,duel_start_ms,reason,resolved_at) VALUES('crossover-round',0,?,?,'browser',?,100,'resolved',?)`, f.event, f.org.ID, fan, now+100)
	for _, movement := range []struct {
		source string
		delta  int
	}{{"minigame:reaction", 5}, {"shop:match_card", -20}, {"opening", 500}, {"transfer", 100}, {"journey:tap-live", 100}, {"minigame:branded", 100}, {"shop:unrelated", -100}} {
		exec(`INSERT INTO fan_wallet_movements(organization_id,owner_key,event_id,fan_id,device_id,delta,balance,label,source) VALUES(?,'fan:test',?,?,'browser',?,0,'Report movement',?)`, f.org.ID, f.event, fan, movement.delta, movement.source)
	}
	for _, perks := range []string{"yes", "maybe", "invalid"} {
		exec(`INSERT INTO event_feedback(event_id,experience,team_spirit,perks_interest,mini_games_interest) VALUES(?,'very_easy','yes',?,'yes')`, f.event, perks)
	}
	for _, device := range []string{"browser", ""} {
		exec(`INSERT INTO sponsor_clicks(event_id,sponsor_id,device_id) VALUES(?,?,?)`, f.event, sponsor, device)
	}
	exec(`INSERT INTO sponsor_clicks(event_id,sponsor_id,device_id) VALUES(?,?,'browser')`, f.event, foreignSponsor)

	data, err := f.router.loadReportOperationalData(f.event, f.org.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := reportOperationalSummary{Votes: 1, IdentifiedParticipants: 1, GameRewardReceipts: 2, ShopRedemptions: 1, CoinsEarned: 5, CoinsSpent: 20, FeedbackResponses: 3, SponsorClicks: 2, SponsorClicksWithoutIdentity: 1}
	if data.Summary != want {
		t.Fatalf("summary=%+v want=%+v", data.Summary, want)
	}
	if data.Feedback != (reportOperationalFeedback{PerksResponses: 2, PerksPositive: 1}) {
		t.Fatalf("feedback=%+v", data.Feedback)
	}
	counts := map[string]int{}
	for _, source := range data.Sources {
		counts[source.Table] = source.Rows
	}
	if len(counts) != 14 || counts["fan_profiles"] != 1 || counts["fan_minigame_rewards"] != 3 || counts["basketball_shots"] != 3 || counts["fan_wallet_movements"] != 2 || counts["sponsor_clicks"] != 2 {
		t.Fatalf("source counts=%+v", counts)
	}
	completed, resolvedShots := 0, 0
	for _, signal := range data.Signals {
		if signal.Metadata["_operational"] != true {
			t.Fatalf("missing source marker: %+v", signal)
		}
		if signal.Name == "page_view" || signal.Name == "active_time" || signal.Name == "feedback_completed" || signal.Name == "registration_completed" || signal.Name == "sponsor_view" {
			t.Fatalf("invented observation: %+v", signal)
		}
		if signal.DeviceID != "" && len(signal.DeviceID) != 64 {
			t.Fatalf("unhashed device: %+v", signal)
		}
		if strings.Contains(signal.ID, "browser") {
			t.Fatalf("device leaked through signal ID: %+v", signal)
		}
		if signal.Name == "game_completed" {
			completed++
		}
		if signal.Metadata["game_id"] == "basketball" {
			resolvedShots++
			if signal.Name != "game_completed" || (signal.Metadata["round_id"] != "shot-1" && signal.Metadata["round_id"] != "shot-2") {
				t.Fatalf("resolved shot lost its completed round ID: %+v", signal)
			}
		}
		if signal.Metadata["_table"] == "fan_minigame_quiz_answers" && (signal.OccurredAt != "" || signal.Metadata["round_id"] != "quiz-round") {
			t.Fatalf("quiz source corrupted: %+v", signal)
		}
		if signal.Metadata["_table"] == "sponsor_clicks" && signal.Metadata["name"] != "Report partner" {
			t.Fatalf("wrong report sponsor name: %+v", signal)
		}
	}
	if completed != 4 || resolvedShots != 2 {
		t.Fatalf("completed=%d resolved shots=%d", completed, resolvedShots)
	}
	for _, signal := range data.Signals {
		if signal.Metadata["_table"] == "votes" && signal.DeviceID != reportOperationalDeviceKey(f.org.ID, "browser") {
			t.Fatal("device hashing differs from report ingest")
		}
	}
}

func TestEngagementReportOperationalSourcesRespectTenantAndPendingProfiles(t *testing.T) {
	f := newEngagementReportFixture(t)
	otherEvent, err := f.router.db.CreateEvent(database.Event{OrganizationID: f.other.ID, Team1ID: f.other.TeamID, Team2ID: f.org.TeamID, StartDateTime: "2026-10-22T19:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	for _, spec := range []struct {
		org           int
		pending       int
		phone, device string
	}{{f.org.ID, 1, "+39222", "pending"}, {f.other.ID, 0, "+39333", "foreign"}} {
		result, err := f.conn.Exec(`INSERT INTO fan_profiles(organization_id,nickname,phone,registration_pending) VALUES(?,'Profile',?,?)`, spec.org, spec.phone, spec.pending)
		if err != nil {
			t.Fatal(err)
		}
		fan, _ := result.LastInsertId()
		if _, err = f.conn.Exec(`INSERT INTO fan_minigame_rewards(event_id,organization_id,request_id,device_id,fan_id,reward_key,coins) VALUES(?,?,?,?,?,'tap',5)`, f.event, f.org.ID, spec.device, spec.device, fan); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = f.conn.Exec(`INSERT INTO fan_minigame_rewards(event_id,organization_id,request_id,device_id,reward_key,coins) VALUES(?,?,'foreign-event','outside','tap',5)`, otherEvent, f.other.ID); err != nil {
		t.Fatal(err)
	}
	data, err := f.router.loadReportOperationalData(f.event, f.org.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Signals) != 2 || data.Summary.IdentifiedParticipants != 0 || data.Summary.GameRewardReceipts != 2 {
		t.Fatalf("unexpected data=%+v", data)
	}
	for _, signal := range data.Signals {
		if signal.FanID != 0 || signal.DeviceID == "" {
			t.Fatalf("unverified/cross-tenant profile adopted: %+v", signal)
		}
	}
	data, err = f.router.loadReportOperationalData(otherEvent, f.org.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Signals) != 0 || data.Summary != (reportOperationalSummary{}) {
		t.Fatalf("cross-tenant event leaked: %+v", data)
	}
}

func TestEngagementReportOperationalRoundIDsDeduplicateReceiptsQuizAndTelemetry(t *testing.T) {
	f := newEngagementReportFixture(t)
	if _, err := f.conn.Exec(`INSERT INTO fan_minigame_rewards(event_id,organization_id,request_id,device_id,reward_key,coins) VALUES(?,?,'same-round','browser','quiz',5)`, f.event, f.org.ID); err != nil {
		t.Fatal(err)
	}
	for _, question := range []int{1, 2} {
		if _, err := f.conn.Exec(`INSERT INTO fan_minigame_quiz_answers(event_id,organization_id,device_id,request_id,question_id) VALUES(?,?,'browser','same-round',?)`, f.event, f.org.ID, question); err != nil {
			t.Fatal(err)
		}
	}
	data, err := f.router.loadReportOperationalData(f.event, f.org.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, signal := range data.Signals {
		if signal.Metadata["round_id"] != "same-round" {
			t.Fatalf("lost shared stable round ID: %+v", signal)
		}
	}
	telemetry := reportSignal{ID: "client-completion", Name: "game_completed", DeviceID: reportOperationalDeviceKey(f.org.ID, "browser"), SessionID: "browser-session", OccurredAt: time.Now().UTC().Format(time.RFC3339Nano), Metadata: map[string]interface{}{"game_id": "quiz", "round_id": "same-round"}}
	facts, details, _ := calculateReportMetrics(append(data.Signals, telemetry), nil)
	if facts["rounds_completed"] == nil || *facts["rounds_completed"] != 1 || facts["repeat_players"] == nil || *facts["repeat_players"] != 0 {
		t.Fatalf("double-counted round facts=%+v", facts)
	}
	if len(details.Games) != 1 || details.Games[0].RoundsStarted != 1 || details.Games[0].RoundsCompleted != 1 {
		t.Fatalf("double-counted game=%+v", details.Games)
	}
}

func TestEngagementReportOperationalBasketballShotMatchesTelemetryRound(t *testing.T) {
	f := newEngagementReportFixture(t)
	now := time.Now().UnixMilli()
	if _, err := f.conn.Exec(`INSERT INTO basketball_shots(shot_id,event_id,organization_id,device_id,attempt,hoop_x,hoop_y,issued_at,expires_at,status) VALUES('same-shot',?,?,'browser',0,0,0,?,?,'miss')`, f.event, f.org.ID, now, now+1000); err != nil {
		t.Fatal(err)
	}
	data, err := f.router.loadReportOperationalData(f.event, f.org.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Signals) != 1 || data.Signals[0].Name != "game_completed" || data.Signals[0].Metadata["round_id"] != "same-shot" {
		t.Fatalf("invalid shot evidence=%+v", data.Signals)
	}
	all := append([]reportSignal{}, data.Signals...)
	for i, name := range []string{"game_started", "game_completed"} {
		all = append(all, reportSignal{ID: fmt.Sprintf("basketball-client-%d", i), Name: name, DeviceID: reportOperationalDeviceKey(f.org.ID, "browser"), SessionID: "browser-session", OccurredAt: time.Now().UTC().Format(time.RFC3339Nano), Metadata: map[string]interface{}{"game_id": "basketball", "round_id": "same-shot"}})
	}
	facts, details, _ := calculateReportMetrics(all, nil)
	if facts["rounds_completed"] == nil || *facts["rounds_completed"] != 1 || facts["players"] == nil || *facts["players"] != 1 || facts["repeat_players"] == nil || *facts["repeat_players"] != 0 {
		t.Fatalf("basketball shot counted twice: %+v", facts)
	}
	if len(details.Games) != 1 || details.Games[0].RoundsStarted != 1 || details.Games[0].RoundsCompleted != 1 {
		t.Fatalf("duplicate basketball round details: %+v", details.Games)
	}
}
