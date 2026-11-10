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

	experience := strings.TrimSpace(strings.ToLower(payload.Experience))
	teamSpirit := strings.TrimSpace(strings.ToLower(payload.TeamSpirit))
	perksInterest := strings.TrimSpace(strings.ToLower(payload.PerksInterest))
	miniGamesInterest := strings.TrimSpace(strings.ToLower(payload.MiniGamesInterest))
	suggestion := strings.TrimSpace(payload.Suggestion)

	if _, ok := feedbackExperienceOptions[experience]; !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if _, ok := feedbackTeamSpiritOptions[teamSpirit]; !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if _, ok := feedbackPerksOptions[perksInterest]; !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if _, ok := feedbackMiniGamesOptions[miniGamesInterest]; !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	feedback := database.EventFeedback{
		EventID:           eventID,
		Experience:        experience,
		TeamSpirit:        teamSpirit,
		PerksInterest:     perksInterest,
		MiniGamesInterest: miniGamesInterest,
		Suggestion:        suggestion,
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
	if err := rt.db.RecordEventFeedbackAnswer(eventID, hex.EncodeToString(responseID), payload.Question, value); err != nil {
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
