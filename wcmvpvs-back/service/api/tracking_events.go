package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
)

type trackingEventsRequest struct {
	SessionID       string                 `json:"session_id"`
	Page            string                 `json:"page"`
	Source          string                 `json:"source"`
	FanSessionToken string                 `json:"fan_session_token"`
	Events          []trackingEventPayload `json:"events"`
}

type trackingEventPayload struct {
	Name             string                 `json:"name"`
	Domain           string                 `json:"domain"`
	OccurredAt       string                 `json:"occurred_at"`
	Page             string                 `json:"page"`
	Section          string                 `json:"section"`
	Source           string                 `json:"source"`
	SessionID        string                 `json:"session_id"`
	DeviceID         string                 `json:"device_id"`
	FanID            int                    `json:"fan_id"`
	OrganizationID   int                    `json:"organization_id"`
	OrganizationSlug string                 `json:"organization_slug"`
	LoginState       string                 `json:"login_state"`
	ProfileState     string                 `json:"profile_state"`
	Metadata         map[string]interface{} `json:"metadata"`
}

func (rt *_router) recordTrackingEvents(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(chi.URLParam(r, "eventId"))
	if err != nil || eventID <= 0 {
		ctx.Logger.WithField("event_id", chi.URLParam(r, "eventId")).Warn("invalid event id for tracking events")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	resolvedOrganizationID := ctx.OrganizationID
	resolvedOrganizationSlug := strings.TrimSpace(ctx.OrganizationSlug)
	if resolvedOrganizationID > 0 {
		if !rt.ensureEventInOrganization(w, ctx, eventID) {
			return
		}
	} else {
		eventOrganizationID, orgErr := rt.db.GetEventOrganizationID(eventID)
		if orgErr != nil {
			if errors.Is(orgErr, sql.ErrNoRows) {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			ctx.Logger.WithError(orgErr).Error("cannot resolve event organization for tracking")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		resolvedOrganizationID = eventOrganizationID
		if resolvedOrganizationID > 0 {
			if org, orgErr := rt.db.GetOrganization(resolvedOrganizationID); orgErr == nil {
				resolvedOrganizationSlug = strings.TrimSpace(org.Slug)
			}
		}
	}

	var payload trackingEventsRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 512*1024)).Decode(&payload); err != nil {
		ctx.Logger.WithError(err).Warn("invalid tracking events payload")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(payload.Events) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	deviceID := rt.deviceIDFromRequest(r)
	fanSessionToken := strings.TrimSpace(payload.FanSessionToken)
	if fanSessionToken == "" {
		fanSessionToken = rt.fanSessionTokenFromRequest(r)
	}

	resolvedFanID := 0
	if fanSessionToken != "" {
		if summary, fanErr := rt.db.GetFanBySessionToken(fanSessionToken, deviceID); fanErr == nil {
			resolvedFanID = summary.Profile.ID
		}
	}

	items := make([]database.TrackingEvent, 0, len(payload.Events))
	for _, item := range payload.Events {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		metadataJSON, _ := json.Marshal(item.Metadata)
		fanID := item.FanID
		if fanID <= 0 {
			fanID = resolvedFanID
		}
		items = append(items, database.TrackingEvent{
			Name:             name,
			Domain:           strings.TrimSpace(item.Domain),
			SessionID:        firstNonEmpty(strings.TrimSpace(item.SessionID), strings.TrimSpace(payload.SessionID)),
			DeviceID:         firstNonEmpty(strings.TrimSpace(item.DeviceID), strings.TrimSpace(deviceID)),
			FanID:            fanID,
			OrganizationID:   firstPositive(item.OrganizationID, resolvedOrganizationID),
			OrganizationSlug: firstNonEmpty(strings.TrimSpace(item.OrganizationSlug), resolvedOrganizationSlug),
			EventID:          eventID,
			Page:             firstNonEmpty(strings.TrimSpace(item.Page), strings.TrimSpace(payload.Page)),
			Section:          strings.TrimSpace(item.Section),
			Source:           firstNonEmpty(strings.TrimSpace(item.Source), strings.TrimSpace(payload.Source)),
			LoginState:       strings.TrimSpace(item.LoginState),
			ProfileState:     strings.TrimSpace(item.ProfileState),
			OccurredAt:       strings.TrimSpace(item.OccurredAt),
			MetadataJSON:     string(metadataJSON),
		})
	}

	if len(items) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := rt.db.RecordTrackingEvents(eventID, items); err != nil {
		if !errors.Is(err, http.ErrBodyReadAfterClose) {
			ctx.Logger.WithError(err).Error("cannot record tracking events")
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstPositive(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}
