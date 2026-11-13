package api

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
)

type feedbackAnswerDB struct {
	fanLiveDB
	saved []string
}

func (db *feedbackAnswerDB) RecordEventFeedbackAnswer(_ int, responseID, question, value string) error {
	db.saved = []string{responseID, question, value}
	return nil
}

func TestFeedbackAnswerValidationAndEventScope(t *testing.T) {
	db := &feedbackAnswerDB{fanLiveDB: fanLiveDB{event: database.Event{ID: 42, OrganizationID: 7, ShowFeedbackSurvey: true}}}
	rt := &_router{db: db}
	ctx := reqcontext.RequestContext{OrganizationID: 7}
	id := strings.Repeat("a", 32)
	payload := map[string]string{"response_id": id, "question": "experience", "value": "very_easy"}
	post := func(event, wantStatus int) {
		t.Helper()
		got := engagementReportRequest(t, rt.submitEventFeedbackAnswer, http.MethodPost, event, payload, ctx, nil)
		if got.Code != wantStatus {
			t.Fatalf("status %d, want %d", got.Code, wantStatus)
		}
	}
	post(43, http.StatusNotFound)
	payload["response_id"] = "invalid"
	post(42, http.StatusBadRequest)
	payload["response_id"] = id
	payload["value"] = "unknown"
	post(42, http.StatusBadRequest)
	payload["value"] = "very_easy"
	db.event.ShowFeedbackSurvey = false
	post(42, http.StatusForbidden)
	if db.saved != nil {
		t.Fatalf("unexpected writes: %v", db.saved)
	}
	db.event.ShowFeedbackSurvey = true
	post(42, http.StatusNoContent)
	if strings.Join(db.saved, ",") != id+",experience,very_easy" {
		t.Fatalf("unexpected answer: %v", db.saved)
	}
}

func TestFeedbackAnswersUpdateOneAnonymousResponse(t *testing.T) {
	f := newEngagementReportFixture(t)
	if _, err := f.conn.Exec(`UPDATE events SET show_feedback_survey=1 WHERE id=?`, f.event); err != nil {
		t.Fatal(err)
	}
	if err := f.router.db.RecordEventFeedback(database.EventFeedback{
		EventID: f.event, Experience: "very_easy", TeamSpirit: "medium", PerksInterest: "maybe", MiniGamesInterest: "no",
	}); err != nil {
		t.Fatal(err)
	}
	post := func(question, value string) {
		t.Helper()
		payload := map[string]string{"response_id": strings.Repeat("b", 32), "question": question, "value": value}
		res := engagementReportRequest(t, f.router.submitEventFeedbackAnswer, http.MethodPost, f.event, payload, f.ctx, nil)
		if res.Code != http.StatusNoContent {
			t.Fatalf("status %d saving %s", res.Code, question)
		}
	}
	post("suggestion", "")
	if summary, err := f.router.db.GetEventFeedbackSummary(f.event); err != nil || summary.TotalResponses != 1 {
		t.Fatalf("skipping an optional answer created feedback: %+v, %v", summary, err)
	}
	post("experience", "easy")
	summary, err := f.router.db.GetEventFeedbackSummary(f.event)
	if err != nil || summary.TotalResponses != 2 || summary.ExperienceCounts["easy"] != 1 || summary.PerksInterestCounts["yes"] != 0 {
		t.Fatalf("partial feedback was not counted correctly: %+v, %v", summary, err)
	}
	post("perks_interest", "yes")
	post("experience", "hard")
	post("experience", "hard")
	post("report_experience", "good")
	post("suggestion", "Piu giochi")
	var experience, perks, overall, suggestion string
	err = f.conn.QueryRow(`SELECT experience, perks_interest, report_experience, suggestion
		FROM event_feedback WHERE event_id=? AND response_id IS NOT NULL`, f.event).Scan(&experience, &perks, &overall, &suggestion)
	if err != nil || experience != "hard" || perks != "yes" || overall != "good" || suggestion != "Piu giochi" {
		t.Fatalf("saved feedback: %q %q %q %q, %v", experience, perks, overall, suggestion, err)
	}
	summary, err = f.router.db.GetEventFeedbackSummary(f.event)
	if err != nil || summary.TotalResponses != 2 || summary.ExperienceCounts["hard"] != 1 {
		t.Fatalf("retries or edits duplicated the response: %+v, %v", summary, err)
	}
}

func TestMatchdayFeedbackUsesSavedAnswersForOutcomesAndBrowserIdentity(t *testing.T) {
	f := newEngagementReportFixture(t)
	if _, err := f.conn.Exec(`UPDATE events SET show_feedback_survey=1 WHERE id=?`, f.event); err != nil {
		t.Fatal(err)
	}
	responseID := strings.Repeat("a", 32)
	failedID := strings.Repeat("b", 32)
	post := func(device, response, question, value string, status int) {
		t.Helper()
		body := map[string]string{"response_id": response, "question": question, "value": value}
		res := engagementReportRequest(t, f.router.submitEventFeedbackAnswer, http.MethodPost, f.event, body, f.ctx, map[string]string{"X-Device-ID": device})
		if res.Code != status {
			t.Fatalf("status %d, want %d saving %s", res.Code, status, question)
		}
	}
	post("answering-browser", responseID, "experience", "easy", http.StatusNoContent)
	post("other-browser", responseID, "experience", "hard", http.StatusConflict)
	post("answering-browser", responseID, "team_spirit", "high", http.StatusNoContent)
	post("answering-browser", responseID, "perks_interest", "yes", http.StatusNoContent)
	post("answering-browser", responseID, "mini_games_interest", "maybe", http.StatusNoContent)
	post("answering-browser", responseID, "report_experience", "good", http.StatusNoContent)
	post("answering-browser", responseID, "report_experience", "good", http.StatusNoContent)

	facts := &matchdayFacts{
		Event: matchdayEvent{ID: f.event}, Metrics: map[string]matchdayMetric{},
		Resolve: func(id matchdayIdentity) string { return id.DeviceKey },
		Signals: []reportSignal{
			{Name: "feedback_opened", DeviceID: "failed-browser", Metadata: map[string]interface{}{"feedback_response_id": failedID}},
			{Name: "feedback_completed", DeviceID: "failed-browser", Metadata: map[string]interface{}{"feedback_response_id": failedID}},
			{Name: "cta_viewed", DeviceID: "view-only", Metadata: map[string]interface{}{"cta": "feedback", "placement": "feedback_home", "feedback_measured": true}},
		},
	}
	if err := loadMatchdayFeedback(context.Background(), f.conn, f.org.ID, map[int]*matchdayFacts{f.event: facts}); err != nil {
		t.Fatal(err)
	}
	for code, want := range map[string]float64{
		"feedback_responses": 1, "feedback_started": 2, "feedback_completed": 1,
		"feedback_respondents": 1, "feedback_response_rate": 100.0 / 3, "feedback_completion_rate": 50, "feedback_satisfaction": 100,
	} {
		checkMatchdayDomainMetric(t, facts, code, want)
		if facts.Metrics[code].Status != "available" {
			t.Fatalf("%s has status %s", code, facts.Metrics[code].Status)
		}
	}
	if len(facts.Actions) != 1 || facts.Actions[0].Identity.DeviceKey != reportOperationalDeviceKey(f.org.ID, "answering-browser") {
		t.Fatalf("unexpected feedback actions: %+v", facts.Actions)
	}
}

func TestMatchdayLiveTapRoundMergesBrowserAndAuthoritativeOutcome(t *testing.T) {
	first := matchdayIdentity{DeviceKey: "first"}
	second := matchdayIdentity{DeviceKey: "second"}
	const round = "head-to-head"
	facts := &matchdayFacts{
		Metrics: map[string]matchdayMetric{}, Resolve: func(id matchdayIdentity) string { return id.DeviceKey },
		games: []matchdayGameObservation{
			{identity: first, game: "tap", round: round, source: "engagement_report_signals", live: true, start: true},
			{identity: first, game: "tap", round: round, source: "engagement_report_signals", live: true, completed: true},
			{identity: first, game: "tap", round: round, source: "tap_live_matches", live: true, start: true, abandoned: true},
			{identity: second, game: "tap", round: round, source: "tap_live_matches", live: true, start: true, abandoned: true},
		},
	}
	calculateMatchdayGames(facts)
	for code, want := range map[string]float64{
		"games_players": 2, "games_starts": 1, "games_completions": 0, "games_completion_rate": 0, "games_per_player": 1, "games_replay_rate": 0,
	} {
		checkMatchdayDomainMetric(t, facts, code, want)
		if facts.Metrics[code].Status != "available" {
			t.Fatalf("%s has status %s", code, facts.Metrics[code].Status)
		}
	}
	if len(facts.Actions) != 2 {
		t.Fatalf("unexpected duplicated game actions: %+v", facts.Actions)
	}
}
