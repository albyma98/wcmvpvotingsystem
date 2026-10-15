package api

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
)

func (rt *_router) getFanLiveStatus(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := parseNumericID(chi.URLParam(r, "eventId"))
	if err != nil || ctx.OrganizationID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	event, err := rt.db.GetFanLiveEvent(eventID, ctx.OrganizationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			ctx.Logger.WithError(err).Error("fan live status")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	quiz := map[string]interface{}{"available": false}
	if cfg, err := rt.db.GetEventQuizConfig(eventID); err == nil && cfg.Enabled && event.IsActive && !event.IsConcluded {
		quiz["starts_at"], quiz["ends_at"] = cfg.ActiveFrom, cfg.ActiveTo
		if quizInWindow(cfg) {
			if questions, err := rt.db.ListEventQuizQuestions(eventID); err == nil && len(questions) > 0 {
				quiz["available"] = true
			}
		}
	}
	winners := []database.EventVoteLeaderboardEntry{}
	if event.IsConcluded {
		entries, err := rt.db.GetEventVoteLeaderboard(eventID, 1000)
		if err != nil {
			ctx.Logger.WithError(err).Error("fan live final results")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		for _, entry := range entries {
			if len(entries) > 0 && entry.Votes == entries[0].Votes && entry.Votes > 0 {
				winners = append(winners, entry)
			}
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	_ = writeJSON(w, http.StatusOK, map[string]interface{}{
		"event_id": event.ID, "is_active": event.IsActive, "votes_closed": event.VotesClosed,
		"is_concluded": event.IsConcluded, "team2_name": event.Team2Name, "quiz": quiz, "mvp": winners,
		"show_feedback_survey": event.ShowFeedbackSurvey, "feedback_survey": event.FeedbackSurvey,
	})
}
