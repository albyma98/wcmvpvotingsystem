package api

import (
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/go-chi/chi/v5"
)

type liveLeaderboardEntry struct {
	PlayerID    int     `json:"player_id"`
	FirstName   string  `json:"first_name"`
	LastName    string  `json:"last_name"`
	ImageURL    string  `json:"image_url"`
	Votes       int     `json:"votes"`
	Percentage  float64 `json:"percentage"`
	LastVoteAt  string  `json:"last_vote_at"`
	DisplayName string  `json:"display_name"`
}

type liveTimelinePoint struct {
	Timestamp string `json:"timestamp"`
	Votes     int    `json:"votes"`
}

type liveVotesResponse struct {
	Total       int                    `json:"total"`
	Leaderboard []liveLeaderboardEntry `json:"leaderboard"`
	Timeline    []liveTimelinePoint    `json:"timeline"`
	UpdatedAt   string                 `json:"updated_at"`
}

func (rt *_router) getLiveEventVotes(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	idParam := chi.URLParam(r, "id")
	eventID, err := strconv.Atoi(idParam)
	if err != nil || eventID <= 0 {
		ctx.Logger.WithField("event_id", idParam).Warn("invalid event id for live votes")
		_ = writeJSONMessage(w, http.StatusBadRequest, "Identificativo evento non valido.")
		return
	}

	leaderboardRows, err := rt.db.GetEventVoteLeaderboard(eventID, 5)
	if err != nil {
		ctx.Logger.WithError(err).WithField("event_id", eventID).Error("cannot fetch vote leaderboard")
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Impossibile aggiornare la classifica dei voti in questo momento.")
		return
	}

	totalVotes, err := rt.db.GetEventVoteCount(eventID)
	if err != nil {
		ctx.Logger.WithError(err).WithField("event_id", eventID).Error("cannot fetch total vote count")
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Impossibile aggiornare il totale voti in questo momento.")
		return
	}

	voteTimestamps, err := rt.db.ListEventVoteTimestamps(eventID)
	if err != nil {
		ctx.Logger.WithError(err).WithField("event_id", eventID).Warn("cannot load vote timeline")
		voteTimestamps = nil
	}

	timeline := buildLiveTimeline(voteTimestamps)

	leaderboard := make([]liveLeaderboardEntry, 0, len(leaderboardRows))
	for _, row := range leaderboardRows {
		percentage := 0.0
		if totalVotes > 0 && row.Votes > 0 {
			percentage = math.Round((float64(row.Votes)/float64(totalVotes))*1000) / 10
		}

		firstName := strings.TrimSpace(row.FirstName)
		lastName := strings.TrimSpace(row.LastName)
		displayName := strings.TrimSpace(strings.Join([]string{firstName, lastName}, " "))
		displayName = strings.TrimSpace(displayName)

		leaderboard = append(leaderboard, liveLeaderboardEntry{
			PlayerID:    row.PlayerID,
			FirstName:   firstName,
			LastName:    lastName,
			ImageURL:    strings.TrimSpace(row.ImageURL),
			Votes:       row.Votes,
			Percentage:  percentage,
			LastVoteAt:  row.LastVoteAt,
			DisplayName: displayName,
		})
	}

	response := liveVotesResponse{
		Total:       totalVotes,
		Leaderboard: leaderboard,
		Timeline:    timeline,
		UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
	}

	if err := writeJSON(w, http.StatusOK, response); err != nil {
		ctx.Logger.WithError(err).Error("cannot encode live votes response")
	}
}

func buildLiveTimeline(votes []time.Time) []liveTimelinePoint {
	if len(votes) == 0 {
		return []liveTimelinePoint{}
	}

	counts := make(map[time.Time]int)
	for _, ts := range votes {
		if ts.IsZero() {
			continue
		}
		minute := ts.Truncate(time.Minute)
		counts[minute]++
	}

	if len(counts) == 0 {
		return []liveTimelinePoint{}
	}

	minutes := make([]time.Time, 0, len(counts))
	for key := range counts {
		minutes = append(minutes, key)
	}

	sort.Slice(minutes, func(i, j int) bool {
		return minutes[i].Before(minutes[j])
	})

	timeline := make([]liveTimelinePoint, 0, len(minutes))
	cumulative := 0
	for _, minute := range minutes {
		cumulative += counts[minute]
		timeline = append(timeline, liveTimelinePoint{
			Timestamp: minute.UTC().Format(time.RFC3339),
			Votes:     cumulative,
		})
	}

	return timeline
}
