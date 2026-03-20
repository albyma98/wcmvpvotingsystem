package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
)

func (rt *_router) getAdminEventAIReport(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(chi.URLParam(r, "eventId"))
	if err != nil || eventID <= 0 || !rt.ensureEventInOrganization(w, ctx, eventID) {
		return
	}
	report, err := rt.db.GetEventAIReport(eventID)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		ctx.Logger.WithError(err).Error("cannot load ai event report")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (rt *_router) generateAdminEventAIReport(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(chi.URLParam(r, "eventId"))
	if err != nil || eventID <= 0 || !rt.ensureEventInOrganization(w, ctx, eventID) {
		return
	}
	metrics, err := rt.db.GetEventAIReportMetrics(eventID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot build ai report metrics")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	reqPayload := aiEventReportRequest{EventID: eventID, Metrics: metrics}
	promptJSON, _ := json.Marshal(reqPayload)
	requestTimeout := 30 * time.Second
	if rt.aiService != nil && rt.aiService.cfg.RequestTimeout > 0 {
		requestTimeout = rt.aiService.cfg.RequestTimeout
	}
	reportCtx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	aiResp := rt.aiService.GenerateEventReport(reportCtx, reqPayload)
	responseJSON, _ := json.Marshal(aiResp)

	report, err := rt.db.UpsertEventAIReport(database.EventAIReport{
		EventID:          eventID,
		OrganizationID:   ctx.OrganizationID,
		Status:           "generated",
		Source:           strings.TrimSpace(aiResp.Source),
		ExecutiveSummary: aiResp.ExecutiveSummary,
		FullReport:       aiResp.FullReport,
		Insights:         aiResp.Insights,
		Suggestions:      aiResp.Suggestions,
		Strengths:        aiResp.Strengths,
		Criticalities:    aiResp.Criticalities,
		Metrics:          metrics,
		PromptJSON:       string(promptJSON),
		ResponseJSON:     string(responseJSON),
		GeneratedAt:      time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot store ai event report")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, report)
}
