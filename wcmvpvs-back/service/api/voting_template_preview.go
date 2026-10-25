package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
)

var votingTestTemplateSIDPattern = regexp.MustCompile(`^HX[0-9a-fA-F]{32}$`)
var votingTestPhonePattern = regexp.MustCompile(`^\+[1-9][0-9]{6,14}$`)

func (rt *_router) allowVotingTemplateTest(organizationID int) bool {
	rt.votingTemplateTestsMu.Lock()
	defer rt.votingTemplateTestsMu.Unlock()
	if rt.votingTemplateTests == nil {
		rt.votingTemplateTests = map[string][]time.Time{}
	}
	return allowRate(rt.votingTemplateTests, strconv.Itoa(organizationID), 5, time.Minute, time.Now())
}

func (rt *_router) sendVotingTemplateTest(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !strings.EqualFold(ctx.AdminRole, "superadmin") {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	if ctx.OrganizationID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	reader := http.MaxBytesReader(w, r.Body, 2<<10)
	defer reader.Close()
	var payload struct {
		TemplateSID string `json:"template_sid"`
		Phone       string `json:"phone"`
	}
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Richiesta di test non valida.")
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Richiesta di test non valida.")
		return
	}
	payload.TemplateSID = strings.TrimSpace(payload.TemplateSID)
	payload.Phone = strings.Map(func(c rune) rune {
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '-' || c == '(' || c == ')' {
			return -1
		}
		return c
	}, strings.TrimSpace(payload.Phone))
	if !votingTestTemplateSIDPattern.MatchString(payload.TemplateSID) {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Inserisci un Template SID valido: HX seguito da 32 caratteri esadecimali.")
		return
	}
	if !votingTestPhonePattern.MatchString(payload.Phone) {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Inserisci il numero di prova con prefisso internazionale.")
		return
	}
	if !rt.twilioMessaging.enabledWhatsApp() {
		_ = writeJSONMessage(w, http.StatusServiceUnavailable, "Servizio WhatsApp Twilio non configurato sul server.")
		return
	}
	if !rt.allowVotingTemplateTest(ctx.OrganizationID) {
		w.Header().Set("Retry-After", "60")
		_ = writeJSONMessage(w, http.StatusTooManyRequests, "Troppi test inviati. Attendi un minuto prima di riprovare.")
		return
	}
	// Reuse the production variable builder without reading or changing events,
	// organization settings, recipients, winners, or notification queues.
	example := database.VotingClosure{MVPName: "Mario Rossi (TEST)", Percentage: "42,75", Prizes: []database.VotingClosurePrize{
		{Name: "Maglia autografata", TicketCode: "0123"},
		{Name: "Pallone firmato", TicketCode: "0456"},
		{Name: "Buono merchandising", TicketCode: "0789"},
	}}
	result, err := rt.twilioMessaging.SendWhatsAppResultsTemplate(payload.Phone, payload.TemplateSID, example.TemplateVariables())
	if err != nil {
		message := "Esito del test incerto. Verifica WhatsApp o il registro Twilio prima di riprovare."
		var providerError twilioError
		if errors.As(err, &providerError) {
			message = "Twilio ha rifiutato il test. Controlla il template e il mittente WhatsApp nel registro Twilio."
			var details struct {
				Code int `json:"code"`
			}
			if json.Unmarshal([]byte(providerError.Message), &details) == nil && details.Code > 0 {
				message = "Twilio ha rifiutato il test (codice " + strconv.Itoa(details.Code) + "). Controlla il template e il mittente WhatsApp nel registro Twilio."
			}
		}
		ctx.Logger.WithFields(map[string]interface{}{"organization_id": ctx.OrganizationID, "phone": maskPhone(payload.Phone)}).Warn("whatsapp organizer template test not accepted")
		_ = writeJSONMessage(w, twilioHTTPStatus(err), message)
		return
	}
	if result.Status == "failed" || result.Status == "undelivered" {
		_ = writeJSONMessage(w, http.StatusBadGateway, "Twilio ha segnalato un problema nell'invio del test.")
		return
	}
	ctx.Logger.WithFields(map[string]interface{}{"organization_id": ctx.OrganizationID, "phone": maskPhone(payload.Phone), "twilio_sid": result.SID}).Info("whatsapp organizer template test accepted")
	_ = writeJSON(w, http.StatusAccepted, map[string]string{"message": "Test accettato da Twilio. Controlla WhatsApp al numero indicato.", "message_sid": result.SID, "status": result.Status})
}
