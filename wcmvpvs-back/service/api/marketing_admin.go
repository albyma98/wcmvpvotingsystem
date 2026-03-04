package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
)

func (rt *_router) allowMarketingSend(orgID int) bool {
	rt.marketingRateMu.Lock()
	defer rt.marketingRateMu.Unlock()
	now := time.Now()
	window := now.Add(-1 * time.Minute)
	list := rt.marketingRateByOrg[orgID][:0]
	for _, ts := range rt.marketingRateByOrg[orgID] {
		if ts.After(window) {
			list = append(list, ts)
		}
	}
	if len(list) >= 120 {
		rt.marketingRateByOrg[orgID] = list
		return false
	}
	list = append(list, now)
	rt.marketingRateByOrg[orgID] = list
	return true
}

func (rt *_router) listMarketingAudience(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	items, err := rt.db.ListMarketingAudience(ctx.OrganizationID, r.URL.Query().Get("q"), false)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_ = writeJSON(w, http.StatusOK, items)
}
func (rt *_router) getMarketingAudienceFan(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, _ := strconv.Atoi(chi.URLParam(r, "fanId"))
	item, err := rt.db.GetMarketingAudienceFan(ctx.OrganizationID, id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	_ = writeJSON(w, http.StatusOK, item)
}
func (rt *_router) sendSingleMarketingSMS(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !rt.allowMarketingSend(ctx.OrganizationID) {
		_ = writeJSONMessage(w, http.StatusTooManyRequests, "rate limit")
		return
	}
	var p struct {
		FanID   int    `json:"fan_id"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	fan, err := rt.db.GetMarketingAudienceFan(ctx.OrganizationID, p.FanID)
	if err != nil || !fan.AcceptedTerms {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	msg, _ := rt.db.CreateSMSMessage(database.SMSMessage{OrganizationID: ctx.OrganizationID, FanID: fan.FanID, Phone: fan.Phone, Body: strings.TrimSpace(p.Message), Status: "queued"})
	res, sendErr := rt.twilioMessaging.SendSMS(fan.Phone, p.Message)
	status := "sent"
	errText := ""
	if sendErr != nil {
		status = "failed"
		errText = sendErr.Error()
	}
	_ = rt.db.UpdateSMSMessageDelivery(msg.ID, res.SID, status, errText)
	_ = writeJSON(w, http.StatusOK, map[string]interface{}{"id": msg.ID, "status": status})
}
func (rt *_router) createSMSCampaign(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var p struct {
		Name        string `json:"name"`
		Message     string `json:"message"`
		Query       string `json:"query"`
		FanIDs      []int  `json:"fan_ids"`
		ScheduledAt string `json:"scheduled_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	aud, err := rt.db.ListMarketingAudience(ctx.OrganizationID, p.Query, true)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	selected := map[int]bool{}
	for _, id := range p.FanIDs {
		selected[id] = true
	}
	targets := []database.MarketingAudienceEntry{}
	for _, f := range aud {
		if (len(selected) == 0 || selected[f.FanID]) && f.AcceptedTerms {
			targets = append(targets, f)
		}
	}
	filters, _ := json.Marshal(map[string]interface{}{"query": p.Query, "fan_ids": p.FanIDs})
	camp, err := rt.db.CreateSMSCampaign(database.SMSCampaign{OrganizationID: ctx.OrganizationID, Name: strings.TrimSpace(p.Name), Message: strings.TrimSpace(p.Message), FiltersJSON: string(filters), RecipientCount: len(targets), Status: "draft", ScheduledAt: strings.TrimSpace(p.ScheduledAt), CreatedByAdmin: ctx.AdminID})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	for _, t := range targets {
		_, _ = rt.db.CreateSMSMessage(database.SMSMessage{OrganizationID: ctx.OrganizationID, CampaignID: camp.ID, FanID: t.FanID, Phone: t.Phone, Body: camp.Message, Status: "queued"})
	}
	_ = writeJSON(w, http.StatusOK, map[string]interface{}{"campaign": camp, "recipients": len(targets), "estimated_cost": float64(len(targets)) * 0.08})
}
func (rt *_router) listSMSCampaigns(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	items, err := rt.db.ListSMSCampaigns(ctx.OrganizationID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_ = writeJSON(w, http.StatusOK, items)
}
func (rt *_router) sendSMSCampaignTest(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !rt.allowMarketingSend(ctx.OrganizationID) {
		_ = writeJSONMessage(w, http.StatusTooManyRequests, "rate limit")
		return
	}
	var p struct {
		Phone string `json:"phone"`
	}
	_ = json.NewDecoder(r.Body).Decode(&p)
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	cs, _ := rt.db.ListSMSCampaigns(ctx.OrganizationID)
	body := ""
	for _, c := range cs {
		if c.ID == id {
			body = c.Message
			break
		}
	}
	if body == "" || strings.TrimSpace(p.Phone) == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	msg, _ := rt.db.CreateSMSMessage(database.SMSMessage{OrganizationID: ctx.OrganizationID, CampaignID: id, Phone: p.Phone, Body: body, Status: "queued"})
	res, err := rt.twilioMessaging.SendSMS(p.Phone, body)
	status := "sent"
	errText := ""
	if err != nil {
		status = "failed"
		errText = err.Error()
	}
	_ = rt.db.UpdateSMSMessageDelivery(msg.ID, res.SID, status, errText)
	_ = writeJSON(w, http.StatusOK, map[string]interface{}{"status": status})
}
func (rt *_router) sendSMSCampaignNow(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !rt.allowMarketingSend(ctx.OrganizationID) {
		_ = writeJSONMessage(w, http.StatusTooManyRequests, "rate limit")
		return
	}
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	logs, err := rt.db.ListSMSMessages(ctx.OrganizationID, id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	sent := 0
	for _, item := range logs {
		if item.Status != "queued" {
			continue
		}
		res, sErr := rt.twilioMessaging.SendSMS(item.Phone, item.Body)
		status := "sent"
		errText := ""
		if sErr != nil {
			status = "failed"
			errText = sErr.Error()
		}
		_ = rt.db.UpdateSMSMessageDelivery(item.ID, res.SID, status, errText)
		sent++
		time.Sleep(120 * time.Millisecond)
	}
	_ = writeJSON(w, http.StatusOK, map[string]interface{}{"processed": sent})
}
func (rt *_router) listSMSTemplates(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	items, err := rt.db.ListSMSTemplates(ctx.OrganizationID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_ = writeJSON(w, http.StatusOK, items)
}
func (rt *_router) createSMSTemplate(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var p database.SMSTemplate
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	p.OrganizationID = ctx.OrganizationID
	item, err := rt.db.CreateSMSTemplate(p)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_ = writeJSON(w, http.StatusOK, item)
}
func (rt *_router) updateSMSTemplate(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var p database.SMSTemplate
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	p.ID, _ = strconv.Atoi(chi.URLParam(r, "id"))
	p.OrganizationID = ctx.OrganizationID
	item, err := rt.db.UpdateSMSTemplate(p)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_ = writeJSON(w, http.StatusOK, item)
}
func (rt *_router) deleteSMSTemplate(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	if err := rt.db.DeleteSMSTemplate(ctx.OrganizationID, id); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (rt *_router) listSMSLogs(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	campaignID, _ := strconv.Atoi(r.URL.Query().Get("campaign_id"))
	items, err := rt.db.ListSMSMessages(ctx.OrganizationID, campaignID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_ = writeJSON(w, http.StatusOK, items)
}
