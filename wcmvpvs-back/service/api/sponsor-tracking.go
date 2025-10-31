package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
)

type sponsorSessionPayload struct {
	DeviceID string `json:"device_id"`
}

type sponsorExposurePayload struct {
	DeviceID   string    `json:"device_id"`
	SponsorIDs []int     `json:"sponsor_ids"`
	Sponsors   []int     `json:"sponsors"`
	Type       string    `json:"type"`
	DurationMs int       `json:"duration_ms"`
	Timestamp  time.Time `json:"-"`
}

type sponsorAnalyticsResponse struct {
	TotalUsers         int                             `json:"total_users"`
	SeenUsers          int                             `json:"seen_users"`
	SeenRate           float64                         `json:"seen_rate"`
	WatchedUsers       int                             `json:"watched_users"`
	AverageWatchTimeMs float64                         `json:"average_watch_time_ms"`
	TotalWatchTimeMs   int64                           `json:"total_watch_time_ms"`
	TotalClicks        int                             `json:"total_clicks"`
	ClickRate          float64                         `json:"click_rate"`
	UniqueClickers     int                             `json:"unique_clickers"`
	TopSponsor         *sponsorAnalyticsTopSponsor     `json:"top_sponsor,omitempty"`
	Timeline           []sponsorAnalyticsTimelinePoint `json:"timeline"`
}

type sponsorAnalyticsTopSponsor struct {
	SponsorID int    `json:"sponsor_id"`
	Name      string `json:"name"`
	Views     int    `json:"views"`
}

type sponsorAnalyticsTimelinePoint struct {
        Timestamp string `json:"timestamp"`
        Seen      int    `json:"seen"`
        Watched   int    `json:"watched"`
        Clicks    int    `json:"clicks"`
}

func buildSponsorAnalyticsResponse(summary database.SponsorAnalytics) sponsorAnalyticsResponse {
        response := sponsorAnalyticsResponse{
                TotalUsers:         summary.TotalSessions,
                SeenUsers:          summary.SeenSessions,
                WatchedUsers:       summary.WatchedSessions,
                AverageWatchTimeMs: summary.AverageWatchTime,
                TotalWatchTimeMs:   summary.TotalWatchTimeMs,
                TotalClicks:        summary.TotalClicks,
                UniqueClickers:     summary.UniqueClickers,
        }

        if summary.TotalSessions > 0 {
                response.SeenRate = float64(summary.SeenSessions) / float64(summary.TotalSessions) * 100
                response.ClickRate = float64(summary.UniqueClickers) / float64(summary.TotalSessions) * 100
        }

        if summary.TopSponsor != nil {
                response.TopSponsor = &sponsorAnalyticsTopSponsor{
                        SponsorID: summary.TopSponsor.SponsorID,
                        Name:      summary.TopSponsor.Name,
                        Views:     summary.TopSponsor.Views,
                }
        }

        if len(summary.Timeline) > 0 {
                response.Timeline = make([]sponsorAnalyticsTimelinePoint, 0, len(summary.Timeline))
                for _, item := range summary.Timeline {
                        response.Timeline = append(response.Timeline, sponsorAnalyticsTimelinePoint{
                                Timestamp: item.Timestamp,
                                Seen:      item.Seen,
                                Watched:   item.Watched,
                                Clicks:    item.Clicks,
                        })
                }
        }

        return response
}

func (payload *sponsorExposurePayload) normalizedSponsorIDs() []int {
        ids := make([]int, 0, len(payload.SponsorIDs)+len(payload.Sponsors))
        ids = append(ids, payload.SponsorIDs...)
        ids = append(ids, payload.Sponsors...)
        seen := make(map[int]struct{}, len(ids))
	normalized := make([]int, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}
	return normalized
}

func (rt *_router) recordSponsorSessionEvent(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(chi.URLParam(r, "eventId"))
	if err != nil || eventID <= 0 {
		ctx.Logger.WithField("event_id", chi.URLParam(r, "eventId")).Warn("invalid event id for sponsor session")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var payload sponsorSessionPayload
	if r.Body != nil {
		defer r.Body.Close()
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
		if err := decoder.Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
			ctx.Logger.WithError(err).Warn("invalid sponsor session payload")
		}
	}

	deviceID := strings.TrimSpace(payload.DeviceID)
	if deviceID == "" {
		deviceID = rt.deviceIDFromRequest(r)
	}

	if err := rt.db.RecordSponsorSession(eventID, deviceID); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			ctx.Logger.WithError(err).Warn("cannot record sponsor session")
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) recordSponsorExposureEvent(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(chi.URLParam(r, "eventId"))
	if err != nil || eventID <= 0 {
		ctx.Logger.WithField("event_id", chi.URLParam(r, "eventId")).Warn("invalid event id for sponsor exposure")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var payload sponsorExposurePayload
	if r.Body != nil {
		defer r.Body.Close()
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192))
		if err := decoder.Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
			ctx.Logger.WithError(err).Warn("invalid sponsor exposure payload")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	sponsorIDs := payload.normalizedSponsorIDs()
	if len(sponsorIDs) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	deviceID := strings.TrimSpace(payload.DeviceID)
	if deviceID == "" {
		deviceID = rt.deviceIDFromRequest(r)
	}

	if err := rt.db.RecordSponsorExposure(eventID, sponsorIDs, deviceID, payload.Type, payload.DurationMs); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if errors.Is(err, database.ErrInvalidSponsorData) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		ctx.Logger.WithError(err).Warn("cannot record sponsor exposure")
	}

	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) getSponsorAnalytics(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(chi.URLParam(r, "eventId"))
	if err != nil || eventID <= 0 {
		ctx.Logger.WithField("event_id", chi.URLParam(r, "eventId")).Warn("invalid event id for sponsor analytics")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	summary, err := rt.db.GetSponsorAnalytics(eventID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		ctx.Logger.WithError(err).Error("cannot load sponsor analytics")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

        response := buildSponsorAnalyticsResponse(summary)

        w.Header().Set("content-type", "application/json")
        if err := json.NewEncoder(w).Encode(response); err != nil {
                ctx.Logger.WithError(err).Error("cannot encode sponsor analytics response")
        }
}
