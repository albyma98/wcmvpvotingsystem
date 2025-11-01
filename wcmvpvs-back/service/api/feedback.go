package api

import (
	"encoding/json"
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
