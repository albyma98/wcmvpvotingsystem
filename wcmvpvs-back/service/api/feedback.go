package api

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
)

var (
	feedbackExperienceOptions = map[string]struct{}{
		"very_easy": {},
		"easy":      {},
		"complex":   {},
		"hard":      {},
	}
	feedbackTeamSpiritOptions = map[string]struct{}{
		"high":   {},
		"medium": {},
		"low":    {},
	}
	feedbackPerksOptions = map[string]struct{}{
		"yes":   {},
		"maybe": {},
		"no":    {},
	}
	feedbackMiniGamesOptions = map[string]struct{}{
		"super_excited": {},
		"maybe":         {},
		"no":            {},
	}
)

func (rt *_router) submitEventFeedback(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(chi.URLParam(r, "eventId"))
	if err != nil || eventID <= 0 {
		ctx.Logger.WithField("event_id", chi.URLParam(r, "eventId")).Warn("invalid event id while recording feedback")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	reader := http.MaxBytesReader(w, r.Body, 4096)
	defer reader.Close()

	body, err := io.ReadAll(reader)
	if err != nil {
		ctx.Logger.WithError(err).Warn("invalid feedback payload")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var payload struct {
		Experience        string `json:"experience"`
		TeamSpirit        string `json:"team_spirit"`
		PerksInterest     string `json:"perks_interest"`
		MiniGamesInterest string `json:"mini_games_interest"`
		Suggestion        string `json:"suggestion"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		ctx.Logger.WithError(err).Warn("invalid feedback payload")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	event, err := rt.db.GetEventByID(eventID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && ctx.OrganizationID > 0 && event.OrganizationID != ctx.OrganizationID) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		ctx.Logger.WithError(err).Warn("cannot load event for feedback")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	questions := []string{"experience", "team_spirit", "perks_interest", "mini_games_interest", "suggestion"}
	if event.FeedbackSurvey != nil && event.FeedbackSurvey.SelectedQuestions != nil {
		questions = event.FeedbackSurvey.SelectedQuestions
	}
	values := map[string]string{
		"experience":          strings.TrimSpace(strings.ToLower(payload.Experience)),
		"team_spirit":         strings.TrimSpace(strings.ToLower(payload.TeamSpirit)),
		"perks_interest":      strings.TrimSpace(strings.ToLower(payload.PerksInterest)),
		"mini_games_interest": strings.TrimSpace(strings.ToLower(payload.MiniGamesInterest)),
		"suggestion":          strings.TrimSpace(payload.Suggestion),
	}
	answers := make(map[string]string, len(questions))
	for _, question := range questions {
		value := values[question]
		if !validFeedbackAnswer(question, value) || (len(questions) == 1 && value == "") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		answers[question] = value
	}
	feedback := database.EventFeedback{
		EventID:           eventID,
		Experience:        answers["experience"],
		TeamSpirit:        answers["team_spirit"],
		PerksInterest:     answers["perks_interest"],
		MiniGamesInterest: answers["mini_games_interest"],
		Suggestion:        answers["suggestion"],
	}

	if err := rt.db.RecordEventFeedback(feedback); err != nil {
		ctx.Logger.WithError(err).Warn("cannot record feedback")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (rt *_router) submitEventFeedbackAnswer(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := parseNumericID(chi.URLParam(r, "eventId"))
	if err != nil || ctx.OrganizationID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var payload struct {
		ResponseID string `json:"response_id"`
		Question   string `json:"question"`
		Value      string `json:"value"`
	}
	reader := http.MaxBytesReader(w, r.Body, 4096)
	defer reader.Close()
	body, err := io.ReadAll(reader)
	if err != nil || json.Unmarshal(body, &payload) != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	responseID, err := hex.DecodeString(payload.ResponseID)
	if err != nil || len(responseID) != 16 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	value := strings.TrimSpace(payload.Value)
	if !validFeedbackAnswer(payload.Question, value) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	event, err := rt.db.GetFanLiveEvent(eventID, ctx.OrganizationID)
	if errors.Is(err, sql.ErrNoRows) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		ctx.Logger.WithError(err).Warn("cannot load event for feedback")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !event.ShowFeedbackSurvey {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	if cfg := event.FeedbackSurvey; cfg != nil && cfg.SelectedQuestions != nil {
		selected := false
		for _, question := range cfg.SelectedQuestions {
			if question == payload.Question {
				selected = true
				break
			}
		}
		if !selected || (len(cfg.SelectedQuestions) == 1 && value == "") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
	key := hex.EncodeToString(responseID)
	deviceID := strings.TrimSpace(r.Header.Get("X-Device-ID"))
	if deviceID == "" || len(deviceID) > 160 {
		err = rt.db.RecordEventFeedbackAnswer(eventID, key, payload.Question, value)
	} else {
		fanID := 0
		if token := rt.fanSessionTokenFromRequest(r); token != "" {
			if lookupErr := rt.db.SQLConn().QueryRowContext(r.Context(), `SELECT p.id FROM fan_sessions s JOIN fan_profiles p ON p.id=s.fan_id
				WHERE s.token=? AND p.organization_id=? AND p.registration_pending=0`, token, ctx.OrganizationID).Scan(&fanID); lookupErr != nil && !errors.Is(lookupErr, sql.ErrNoRows) {
				ctx.Logger.WithError(lookupErr).Warn("cannot resolve feedback profile")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
		err = rt.db.RecordEventFeedbackTrackedAnswer(eventID, key, payload.Question, value, reportOperationalDeviceKey(ctx.OrganizationID, deviceID), fanID)
	}
	if errors.Is(err, database.ErrFeedbackResponseOwner) {
		w.WriteHeader(http.StatusConflict)
		return
	}
	if err != nil {
		ctx.Logger.WithError(err).Warn("cannot record feedback answer")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func validFeedbackAnswer(question, value string) bool {
	var options map[string]struct{}
	switch question {
	case "experience":
		options = feedbackExperienceOptions
	case "team_spirit":
		options = feedbackTeamSpiritOptions
	case "perks_interest":
		options = feedbackPerksOptions
	case "mini_games_interest":
		options = feedbackMiniGamesOptions
	case "report_experience":
		return value == "awesome" || value == "good" || value == "ok" || value == "bad"
	case "suggestion":
		return len([]rune(value)) <= 80
	}
	_, ok := options[value]
	return ok
}
