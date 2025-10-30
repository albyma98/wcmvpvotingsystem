package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
)

type historyTimelineBucket struct {
	Start string `json:"start"`
	End   string `json:"end"`
	Label string `json:"label"`
	Votes int    `json:"votes"`
}

type eventHistoryPrize struct {
	ID               int    `json:"id"`
	Name             string `json:"name"`
	Position         int    `json:"position"`
	WinnerTicketCode string `json:"winner_ticket_code,omitempty"`
}

type eventHistoryEntry struct {
	ID                 int                         `json:"id"`
	Title              string                      `json:"title"`
	StartDateTime      string                      `json:"start_datetime"`
	Location           string                      `json:"location"`
	TotalVotes         int                         `json:"total_votes"`
	SponsorClicksTotal int                         `json:"sponsor_clicks_total"`
	MVP                *database.EventMVP          `json:"mvp,omitempty"`
	SponsorClicks      []database.SponsorClickStat `json:"sponsor_clicks"`
	Timeline           []historyTimelineBucket     `json:"timeline"`
	HomeTeam           string                      `json:"home_team"`
	AwayTeam           string                      `json:"away_team"`
	Prizes             []eventHistoryPrize         `json:"prizes"`
	HasPrizeDraw       bool                        `json:"has_prize_draw"`
}

type historyEntryWrapper struct {
	entry     eventHistoryEntry
	startTime time.Time
}

func (rt *_router) recordSponsorClick(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(chi.URLParam(r, "eventId"))
	if err != nil || eventID <= 0 {
		ctx.Logger.WithField("event_id", chi.URLParam(r, "eventId")).Warn("invalid event id while tracking sponsor click")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sponsorID, err := strconv.Atoi(chi.URLParam(r, "sponsorId"))
	if err != nil || sponsorID <= 0 {
		ctx.Logger.WithField("sponsor_id", chi.URLParam(r, "sponsorId")).Warn("invalid sponsor id while tracking sponsor click")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var payload struct {
		DeviceID string `json:"device_id"`
	}
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
			ctx.Logger.WithError(err).Warn("invalid sponsor click payload")
		}
	}

	deviceID := strings.TrimSpace(payload.DeviceID)
	if deviceID == "" {
		deviceID = rt.deviceIDFromRequest(r)
	}

	if err := rt.db.RecordSponsorClick(eventID, sponsorID, deviceID); err != nil {
		ctx.Logger.WithError(err).Warn("cannot record sponsor click")
	}

	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) getEventHistory(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	events, err := rt.db.ListEvents()
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list events for history")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	wrappers := make([]historyEntryWrapper, 0, len(events))

	for _, event := range events {
		if !event.IsConcluded {
			continue
		}

		startTime, normalizedStart := parseEventStart(event.StartDateTime)

		totalVotes, err := rt.db.GetEventVoteCount(event.ID)
		if err != nil {
			ctx.Logger.WithError(err).WithField("event_id", event.ID).Warn("cannot count votes for history")
			continue
		}

		mvp, err := rt.db.GetEventMVP(event.ID)
		var mvpPtr *database.EventMVP
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				ctx.Logger.WithError(err).WithField("event_id", event.ID).Warn("cannot compute MVP for history")
			}
		} else {
			mvpPtr = &mvp
		}

		sponsorStats, err := rt.db.GetSponsorClickStats(event.ID)
		sponsorTotal := 0
		if err != nil {
			ctx.Logger.WithError(err).WithField("event_id", event.ID).Warn("cannot load sponsor stats for history")
			sponsorStats = []database.SponsorClickStat{}
		}
		for _, stat := range sponsorStats {
			sponsorTotal += stat.Clicks
		}

		voteTimestamps, err := rt.db.ListEventVoteTimestamps(event.ID)
		if err != nil {
			ctx.Logger.WithError(err).WithField("event_id", event.ID).Warn("cannot load vote timestamps for history")
			voteTimestamps = nil
		}

		prizeList, err := rt.db.ListEventPrizes(event.ID)
		if err != nil {
			ctx.Logger.WithError(err).WithField("event_id", event.ID).Warn("cannot load prizes for history")
			prizeList = nil
		}

		prizes := make([]eventHistoryPrize, 0, len(prizeList))
		hasPrizeDraw := false
		for _, prize := range prizeList {
			winnerCode := ""
			if prize.Winner != nil {
				winnerCode = strings.TrimSpace(prize.Winner.TicketCode)
				if winnerCode != "" {
					hasPrizeDraw = true
				}
			}

			prizes = append(prizes, eventHistoryPrize{
				ID:               prize.ID,
				Name:             strings.TrimSpace(prize.Name),
				Position:         prize.Position,
				WinnerTicketCode: winnerCode,
			})
		}

		sort.SliceStable(prizes, func(i, j int) bool {
			if prizes[i].Position == prizes[j].Position {
				return prizes[i].ID < prizes[j].ID
			}
			return prizes[i].Position < prizes[j].Position
		})

		if startTime.IsZero() && len(voteTimestamps) > 0 {
			startTime = voteTimestamps[0]
			normalizedStart = startTime.UTC().Format(time.RFC3339)
		}

		timeline := buildVoteTimeline(startTime, voteTimestamps)

		entry := eventHistoryEntry{
			ID:                 event.ID,
			Title:              buildEventTitle(event),
			StartDateTime:      normalizedStart,
			Location:           strings.TrimSpace(event.Location),
			TotalVotes:         totalVotes,
			SponsorClicksTotal: sponsorTotal,
			MVP:                mvpPtr,
			SponsorClicks:      sponsorStats,
			Timeline:           timeline,
			HomeTeam:           strings.TrimSpace(event.Team1Name),
			AwayTeam:           strings.TrimSpace(event.Team2Name),
			Prizes:             prizes,
			HasPrizeDraw:       hasPrizeDraw,
		}

		wrappers = append(wrappers, historyEntryWrapper{entry: entry, startTime: startTime})
	}

	sort.SliceStable(wrappers, func(i, j int) bool {
		return wrappers[i].startTime.After(wrappers[j].startTime)
	})

	history := make([]eventHistoryEntry, 0, len(wrappers))
	for _, wrapper := range wrappers {
		history = append(history, wrapper.entry)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(history); err != nil {
		ctx.Logger.WithError(err).Error("cannot encode event history response")
	}
}

func (rt *_router) purgeEvent(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !strings.EqualFold(ctx.AdminRole, "superadmin") {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	eventID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || eventID <= 0 {
		ctx.Logger.WithField("event_id", chi.URLParam(r, "id")).Warn("invalid event id while purging")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var payload struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		ctx.Logger.WithError(err).Warn("invalid payload while purging event")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(payload.Password) == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	admin, err := rt.db.GetAdminByID(ctx.AdminID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot retrieve admin while purging event")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !adminPasswordMatches(admin.PasswordHash, payload.Password) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if err := rt.db.PurgeEventData(eventID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		ctx.Logger.WithError(err).Error("cannot purge event")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parseEventStart(value string) (time.Time, string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, ""
	}

	candidates := []string{trimmed}
	if !strings.Contains(trimmed, "T") {
		candidates = append(candidates, strings.Replace(trimmed, " ", "T", 1))
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
	}

	for _, candidate := range candidates {
		for _, layout := range layouts {
			if ts, err := time.ParseInLocation(layout, candidate, time.UTC); err == nil {
				return ts, ts.UTC().Format(time.RFC3339)
			}
		}
	}

	return time.Time{}, trimmed
}

func buildEventTitle(event database.Event) string {
	home := strings.TrimSpace(event.Team1Name)
	away := strings.TrimSpace(event.Team2Name)
	switch {
	case home != "" && away != "":
		return fmt.Sprintf("%s - %s", home, away)
	case home != "":
		return home
	case away != "":
		return away
	default:
		return fmt.Sprintf("Evento #%d", event.ID)
	}
}

func buildVoteTimeline(start time.Time, votes []time.Time) []historyTimelineBucket {
	if start.IsZero() {
		return []historyTimelineBucket{}
	}

	bucketDuration := 15 * time.Minute
	totalWindow := 3 * time.Hour
	bucketCount := int(totalWindow / bucketDuration)
	if bucketCount <= 0 {
		bucketCount = 1
	}

	windowStart := start.Add(-1 * time.Hour)

	counts := make([]int, bucketCount)
	for _, ts := range votes {
		if ts.IsZero() {
			continue
		}

		diff := ts.Sub(windowStart)
		index := 0
		if diff > 0 {
			index = int(diff / bucketDuration)
			if index >= bucketCount {
				index = bucketCount - 1
			}
		}
		counts[index]++
	}

	timeline := make([]historyTimelineBucket, 0, bucketCount)
	for i := 0; i < bucketCount; i++ {
		bucketStart := windowStart.Add(time.Duration(i) * bucketDuration)
		bucketEnd := bucketStart.Add(bucketDuration)
		timeline = append(timeline, historyTimelineBucket{
			Start: bucketStart.UTC().Format(time.RFC3339),
			End:   bucketEnd.UTC().Format(time.RFC3339),
			Label: fmt.Sprintf("%s-%s", bucketStart.Format("15:04"), bucketEnd.Format("15:04")),
			Votes: counts[i],
		})
	}

	return timeline
}
