package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
)

func adminEventRequest(t *testing.T, rt *_router, ctx reqcontext.RequestContext, method, path string, payload interface{}) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	mux := chi.NewRouter()
	mux.Post("/events", func(w http.ResponseWriter, r *http.Request) { rt.createEvent(w, r, ctx) })
	mux.Put("/events/{id}", func(w http.ResponseWriter, r *http.Request) { rt.updateEvent(w, r, ctx) })
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(string(body))))
	return w
}

func TestAdminEventCreationAndIndependentSaves(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("survey_enabled_%t", enabled), func(t *testing.T) {
			rt, ctx, existingID, _ := votingAPIFixture(t)
			existing, err := rt.db.GetEventByID(existingID)
			if err != nil {
				t.Fatal(err)
			}
			survey := database.DefaultEventFeedbackSurveyConfig()
			survey.Questions[0].Title = "Domanda della società"
			survey.Questions[0].Answers[0].Label = "Risposta personalizzata"
			created := adminEventRequest(t, rt, ctx, http.MethodPost, "/events", map[string]interface{}{
				"team1_id": existing.Team1ID, "team2_id": existing.Team2ID,
				"start_datetime": "2026-11-01T19:00", "location": "Arena",
				"show_feedback_survey": enabled, "show_selfie": false, "show_vote_counter": false,
				"feedback_survey": survey,
				"prizes":          []database.EventPrize{{Name: "Maglia", Position: 1, WinSMSText: "Hai vinto la maglia!"}},
			})
			if created.Code != http.StatusOK {
				t.Fatalf("create status %d: %s", created.Code, created.Body.String())
			}
			var result struct{ ID int }
			if err := json.Unmarshal(created.Body.Bytes(), &result); err != nil || result.ID == 0 {
				t.Fatalf("create response: %s, %v", created.Body.String(), err)
			}
			path := fmt.Sprintf("/events/%d", result.ID)
			load := func() database.Event {
				t.Helper()
				event, err := rt.db.GetEventByID(result.ID)
				if err != nil {
					t.Fatal(err)
				}
				if event.Team1ID != existing.Team1ID || event.Team2ID != existing.Team2ID || event.StartDateTime != "2026-11-01T19:00" || event.Location != "Arena" || event.ShowFeedbackSurvey != enabled || event.ShowSelfie || event.ShowVoteCounter {
					t.Fatalf("unrelated fields changed: %+v", event)
				}
				return event
			}
			update := func(payload interface{}, wantStatus int) {
				t.Helper()
				w := adminEventRequest(t, rt, ctx, http.MethodPut, path, payload)
				if w.Code != wantStatus {
					t.Fatalf("update status %d want %d: %s", w.Code, wantStatus, w.Body.String())
				}
			}
			initial := load()
			if !reflect.DeepEqual(*initial.FeedbackSurvey, survey) || len(initial.Prizes) != 1 || initial.Prizes[0].WinSMSText != "Hai vinto la maglia!" {
				t.Fatalf("creation lost survey/prizes: %+v", initial)
			}

			survey.Questions[0].Title = "Sondaggio aggiornato"
			update(map[string]interface{}{"feedback_survey": survey}, http.StatusNoContent)
			afterSurvey := load()
			if !reflect.DeepEqual(afterSurvey.Prizes, initial.Prizes) || !reflect.DeepEqual(*afterSurvey.FeedbackSurvey, survey) {
				t.Fatal("saving the survey changed prizes or lost the new survey")
			}

			prize := initial.Prizes[0]
			prize.Name, prize.WinSMSText = "Pallone", "Hai vinto il pallone!"
			update(map[string]interface{}{"prizes": []database.EventPrize{prize}}, http.StatusNoContent)
			afterPrizes := load()
			if !reflect.DeepEqual(*afterPrizes.FeedbackSurvey, survey) || len(afterPrizes.Prizes) != 1 || afterPrizes.Prizes[0].ID != prize.ID || afterPrizes.Prizes[0].Name != prize.Name || afterPrizes.Prizes[0].WinSMSText != prize.WinSMSText {
				t.Fatal("saving prizes changed the survey or lost prize data")
			}

			update(map[string]interface{}{"show_reaction_test": false}, http.StatusNoContent)
			afterSettings := load()
			if afterSettings.ShowReactionTest || !reflect.DeepEqual(afterSettings.Prizes, afterPrizes.Prizes) || !reflect.DeepEqual(*afterSettings.FeedbackSurvey, survey) {
				t.Fatal("saving settings changed prizes or the survey")
			}
			update(map[string]interface{}{"prizes": nil, "feedback_survey": nil}, http.StatusNoContent)
			if unchanged := load(); !reflect.DeepEqual(unchanged.Prizes, afterSettings.Prizes) || !reflect.DeepEqual(*unchanged.FeedbackSurvey, survey) {
				t.Fatal("null collections must preserve existing data")
			}
			update(map[string]interface{}{"prizes": []database.EventPrize{}}, http.StatusNoContent)
			if empty := load(); len(empty.Prizes) != 0 || !reflect.DeepEqual(*empty.FeedbackSurvey, survey) {
				t.Fatal("explicitly removing all prizes must preserve the survey")
			}
		})
	}
}

func TestAdminEventSurveySavePreservesAssignedPrizes(t *testing.T) {
	rt, ctx, eventID, _ := votingAPIFixture(t)
	if _, err := rt.db.CloseEventVotingAndDraw(eventID, ctx.OrganizationID); err != nil {
		t.Fatal(err)
	}
	before, err := rt.db.GetEventByID(eventID)
	if err != nil || len(before.Prizes) != 3 || before.Prizes[0].Winner == nil {
		t.Fatalf("assigned prizes fixture: %+v, %v", before.Prizes, err)
	}
	survey := database.DefaultEventFeedbackSurveyConfig()
	survey.Questions[0].Title = "Domanda dopo l'estrazione"
	path := fmt.Sprintf("/events/%d", eventID)
	w := adminEventRequest(t, rt, ctx, http.MethodPut, path, map[string]interface{}{"feedback_survey": survey})
	if w.Code != http.StatusNoContent {
		t.Fatalf("survey save with winners: %d", w.Code)
	}
	after, err := rt.db.GetEventByID(eventID)
	if err != nil || !reflect.DeepEqual(after.Prizes, before.Prizes) || !reflect.DeepEqual(*after.FeedbackSurvey, survey) {
		t.Fatalf("assigned prizes/survey changed: %+v, %v", after, err)
	}
	w = adminEventRequest(t, rt, ctx, http.MethodPut, path, map[string]interface{}{"prizes": []database.EventPrize{}, "location": "Must roll back"})
	if w.Code != http.StatusConflict {
		t.Fatalf("explicit winner removal status: %d", w.Code)
	}
	after, err = rt.db.GetEventByID(eventID)
	if err != nil || after.Location != before.Location || !reflect.DeepEqual(after.Prizes, before.Prizes) {
		t.Fatalf("failed removal must roll back all changes: %+v, %v", after, err)
	}
}
