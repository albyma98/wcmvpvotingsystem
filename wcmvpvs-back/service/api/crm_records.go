package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
)

var crmLabels = map[string]string{
	"lead": "Prospect", "onboarding": "Onboarding", "active": "Cliente attivo", "former": "Ex cliente",
	"to_contact": "Da contattare", "contacted": "Contattato", "conversation": "In conversazione", "demo": "Call / demo",
	"pilot": "Pilota", "proposal": "Proposta", "negotiation": "Negoziazione", "won": "Vinta", "lost": "Persa",
	"regular": "Regolare", "watch": "Da seguire", "risk": "A rischio", "insufficient": "Dati insufficienti",
	"open": "Aperto", "done": "Completato", "resolved": "Risolta", "signed": "Firmato", "ended": "Terminato",
	"planned": "Pianificato", "completed": "Concluso", "draft": "Bozza",
	"contact": "Referente", "opportunity": "Opportunità", "task": "Attività da fare", "checklist": "Onboarding",
	"issue": "Criticità", "goal": "Obiettivo", "metric": "Misurazione manuale", "contract": "Contratto", "payment": "Incasso",
}

func crmStatusLabel(key string) string {
	if key == "" {
		return "—"
	}
	if label := crmLabels[key]; label != "" {
		return label
	}
	return key
}

func crmValidateURL(value string) error {
	if value == "" {
		return nil
	}
	parsed, err := url.Parse(value)
	if err != nil || !crmIs(parsed.Scheme, "https", "http") || parsed.Hostname() == "" || parsed.User != nil {
		return crmProblem("Inserisci un indirizzo web completo (https:// o http://).")
	}
	return nil
}

var crmChecklist = map[string]string{
	"setup": "Configurazione società", "training": "Formazione referente",
	"materials": "Verifica materiali di lancio", "first_match": "Prima gara attiva",
}

func crmRecordKey(value crmRecord) interface{} {
	switch {
	case value.Kind == "checklist":
		return value.Type
	case value.Kind == "opportunity" && value.Type == "renewal" && value.ContractID > 0:
		return fmt.Sprintf("renewal:%d", value.ContractID)
	}
	return nil
}

func crmCheckRecordLink(tx *sql.Tx, companyID, id int64, kind string) error {
	if id == 0 {
		return nil
	}
	var found string
	err := tx.QueryRow("SELECT kind FROM crm_records WHERE company_id=? AND id=?", companyID, id).Scan(&found)
	if err == sql.ErrNoRows || (err == nil && found != kind) {
		return crmProblem("I riferimenti devono appartenere alla stessa società.")
	}
	return err
}

func crmValidateEvent(tx *sql.Tx, company crmCompany, eventID int64) error {
	var found int
	err := tx.QueryRow("SELECT EXISTS(SELECT 1 FROM events WHERE id=? AND organization_id=? AND organization_id>0)", eventID, company.ABXOrgID).Scan(&found)
	if err == nil && found == 0 {
		return crmProblem("Seleziona una gara della società ABX collegata.")
	}
	return err
}

func crmValidateRecord(tx *sql.Tx, company crmCompany, value *crmRecord, old crmRecord, now string) error {
	if value.Kind == "checklist" {
		value.Title = crmChecklist[value.Type]
	}
	value.Title = strings.TrimSpace(value.Title)
	if value.Title == "" || len(value.Title) > 300 {
		return crmProblem("Inserisci un titolo (massimo 300 caratteri).")
	}
	if !crmIs(value.Priority, "", "low", "normal", "high", "urgent") {
		return crmProblem("Priorità non valida.")
	}
	if err := crmValidateDates(value.Start, value.End, value.Due, value.ResumeOn); err != nil {
		return err
	}
	if value.Start != "" && value.End != "" && value.Start > value.End {
		return crmProblem("La fine del periodo deve seguire l’inizio.")
	}
	if value.Kind != "activity" {
		if err := crmValidateDates(value.Date); err != nil {
			return err
		}
	}
	for _, amount := range []*int64{value.Estimated, value.Amount} {
		if amount != nil && (*amount < 0 || *amount > 100000000000) {
			return crmProblem("Inserisci un importo valido, in euro.")
		}
	}
	if err := crmValidateURL(value.ReportURL); err != nil {
		return err
	}
	if err := crmValidateOwner(tx, value.OwnerID); err != nil {
		return err
	}
	for _, link := range []struct {
		id   int64
		kind string
	}{{value.ContactID, "contact"}, {value.OpportunityID, "opportunity"}, {value.ContractID, "contract"}} {
		if err := crmCheckRecordLink(tx, company.ID, link.id, link.kind); err != nil {
			return err
		}
	}
	if value.ID > 0 && (value.OpportunityID == value.ID || value.ContactID == value.ID || value.ContractID == value.ID) {
		return crmProblem("Una voce non può essere collegata a se stessa.")
	}
	switch value.Kind {
	case "contact":
		value.Email = strings.TrimSpace(value.Email)
		if value.Email != "" {
			address, err := mail.ParseAddress(value.Email)
			if err != nil || address.Address != value.Email {
				return crmProblem("Indirizzo email professionale non valido.")
			}
		}
		if !crmIs(value.Type, "", "decision", "operations", "introducer", "influencer") ||
			!crmIs(value.Channel, "", "email", "call", "whatsapp", "meeting") {
			return crmProblem("Influenza o canale del referente non validi.")
		}
	case "opportunity":
		if !crmIs(value.Type, "new", "renewal", "expansion") ||
			!crmIs(value.Status, "to_contact", "contacted", "conversation", "demo", "pilot", "proposal", "negotiation", "won", "lost") {
			return crmProblem("Seleziona tipo e fase dell’opportunità.")
		}
		if value.ContractID != 0 && value.Type != "renewal" {
			return crmProblem("Il contratto di origine è previsto per i rinnovi.")
		}
		if value.ResumeOn != "" && strings.TrimSpace(value.Reason) == "" {
			return crmProblem("Indica il motivo della pausa.")
		}
		switch value.Status {
		case "won":
			if value.Amount == nil {
				return crmProblem("Indica il valore concordato, anche se pari a zero.")
			}
			value.ResumeOn, value.Reason = "", ""
		case "lost":
			if strings.TrimSpace(value.Reason) == "" {
				return crmProblem("Indica il motivo della perdita.")
			}
			value.ResumeOn, value.Amount = "", nil
		default:
			value.Amount = nil
			if value.ResumeOn == "" {
				value.Reason = ""
			}
		}
	case "activity":
		if !crmIs(value.Type, "email", "call", "whatsapp", "meeting", "demo", "note", "pilot", "checkin") {
			return crmProblem("Seleziona un tipo di contatto o nota.")
		}
		parsed, err := time.ParseInLocation(crmLocalTime, value.Date, crmRome)
		if err != nil || parsed.Format(crmLocalTime) != value.Date {
			return crmProblem("Inserisci data e ora valide (Europe/Rome).")
		}
		value.Date = parsed.UTC().Format(time.RFC3339)
	case "task", "checklist":
		if !crmIs(value.Status, "open", "done") {
			return crmProblem("Stato dell’attività non valido.")
		}
		if value.Kind == "task" && (value.Due == "" || !crmIs(value.Type, "followup", "call", "demo", "pilot", "postmatch", "report", "checkin", "other")) {
			return crmProblem("Seleziona un tipo e una scadenza per l’attività.")
		}
		if value.Kind == "checklist" && old.ID != 0 && old.Type != value.Type {
			return crmProblem("La voce di onboarding è già definita.")
		}
		value.CompletedAt = ""
		if value.Status == "done" {
			value.CompletedAt = old.CompletedAt
			if value.CompletedAt == "" {
				value.CompletedAt = now
			}
		}
	case "pilot":
		if value.Start == "" || value.End == "" || !crmIs(value.Status, "planned", "active", "completed") {
			return crmProblem("Indica il periodo e lo stato del pilota.")
		}
		if (value.Planned != nil && *value.Planned < 0) || (value.Played != nil && *value.Played < 0) {
			return crmProblem("Il numero di gare deve essere positivo o pari a zero.")
		}
		if value.Status == "completed" && (strings.TrimSpace(value.Result) == "" || strings.TrimSpace(value.NextStep) == "") {
			return crmProblem("Registra l’esito del pilota e il prossimo passo commerciale.")
		}
		seen := map[int64]bool{}
		for _, id := range value.EventIDs {
			if id <= 0 || seen[id] {
				return crmProblem("Ogni gara può essere collegata una sola volta.")
			}
			seen[id] = true
			if !crmHasEvent(old.EventIDs, id) {
				if err := crmValidateEvent(tx, company, id); err != nil {
					return err
				}
			}
		}
	case "issue":
		if !crmIs(value.Status, "open", "resolved") || !crmIs(value.Priority, "low", "normal", "high", "urgent") ||
			strings.TrimSpace(value.Notes) == "" || strings.TrimSpace(value.NextStep) == "" || value.Due == "" {
			return crmProblem("Indica descrizione, gravità, azione e scadenza della criticità.")
		}
		value.CompletedAt = ""
		if value.Status == "resolved" {
			value.CompletedAt = old.CompletedAt
			if value.CompletedAt == "" {
				value.CompletedAt = now
			}
		}
	case "goal", "metric":
		if value.Kind == "metric" && (strings.TrimSpace(value.Source) == "" || value.Date == "" || value.Value == nil) {
			return crmProblem("Indica valore misurato, fonte e data del dato manuale.")
		}
		if value.Metric != "" && !crmHasMetric(value.Metric) {
			return crmProblem("Seleziona un KPI valido.")
		}
		if value.Kind == "goal" && value.Metric != "" && value.Value == nil {
			return crmProblem("Indica la soglia concordata per il KPI.")
		}
		if (value.Value != nil && (*value.Value < 0 || (crmPercentMetric(value.Metric) && *value.Value > 100))) ||
			(value.Denominator != nil && *value.Denominator < 0) {
			return crmProblem("Valore del KPI o denominatore fuori intervallo.")
		}
		if value.EventID != 0 && value.EventID != old.EventID {
			return crmValidateEvent(tx, company, value.EventID)
		}
	case "contract":
		if strings.TrimSpace(value.Season) == "" || value.Start == "" || value.End == "" || !crmIs(value.Status, "draft", "signed", "ended") {
			return crmProblem("Indica stagione, periodo e stato del contratto.")
		}
		if value.Status != "draft" && (value.Amount == nil || value.Due == "" || value.Due > value.End) {
			return crmProblem("Indica importo firmato e avvio rinnovo entro la fine del contratto.")
		}
		paid, err := crmPaid(tx, value.ID, 0)
		if err != nil {
			return err
		}
		if paid > 0 && (value.Status == "draft" || value.Amount == nil || *value.Amount < paid) {
			return crmProblem("L’importo firmato deve coprire gli incassi già registrati.")
		}
		if value.Status == "draft" {
			value.Amount = nil
		}
	case "payment":
		contract, err := crmReadRecord(tx, company.ID, value.ContractID)
		if err != nil {
			return err
		}
		if contract.Kind != "contract" || contract.Amount == nil || contract.Status == "draft" || value.Date == "" || value.Amount == nil || *value.Amount <= 0 {
			return crmProblem("Indica data e importo positivo per un contratto firmato.")
		}
		paid, err := crmPaid(tx, value.ContractID, value.ID)
		if err != nil {
			return err
		}
		if !value.Archived && paid+*value.Amount > *contract.Amount {
			return crmProblem("L’incasso supera l’importo firmato residuo.")
		}
	default:
		return crmProblem("Tipo di voce non valido.")
	}
	return nil
}

func crmHasEvent(ids []int64, id int64) bool {
	for _, linked := range ids {
		if linked == id {
			return true
		}
	}
	return false
}

func crmPaid(tx *sql.Tx, contractID, excludeID int64) (paid int64, err error) {
	if contractID != 0 {
		err = tx.QueryRow("SELECT COALESCE(SUM(json_extract(details,'$.amount')),0) FROM crm_records WHERE kind='payment' AND contract_id=? AND id<>? AND archived=0",
			contractID, excludeID).Scan(&paid)
	}
	return
}

func crmInsertRecord(tx *sql.Tx, value crmRecord, key interface{}, now string, actor reqcontext.RequestContext) (int64, error) {
	raw, err := json.Marshal(value.crmRecordDetails)
	if err != nil {
		return 0, err
	}
	result, err := tx.Exec("INSERT INTO crm_records(company_id,kind,stable_key,contact_id,opportunity_id,contract_id,owner_id,details,archived,author_id,author_name,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,(SELECT id FROM admins WHERE id=?),?,?,?)",
		value.CompanyID, value.Kind, key, crmNull(value.ContactID), crmNull(value.OpportunityID), crmNull(value.ContractID), crmNull(value.OwnerID),
		string(raw), value.Archived, actor.AdminID, actor.AdminUsername, now, now)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func crmHistory(tx *sql.Tx, companyID, opportunityID int64, title, notes, now string, actor reqcontext.RequestContext) error {
	_, err := crmInsertRecord(tx, crmRecord{CompanyID: companyID, OpportunityID: opportunityID, Kind: "history",
		crmRecordDetails: crmRecordDetails{Title: title, Notes: notes, Date: now}}, nil, now, actor)
	return err
}

func crmLogRecord(tx *sql.Tx, value, old crmRecord, now string, actor reqcontext.RequestContext) error {
	if value.Kind == "activity" && old.ID == 0 {
		return nil
	}
	title := crmStatusLabel(value.Kind) + " · " + value.Title
	note := "Registrazione creata"
	opportunityID := value.OpportunityID
	if value.Kind == "opportunity" {
		opportunityID = value.ID
	}
	switch {
	case old.ID > 0 && old.Archived != value.Archived:
		if value.Archived {
			note = "Voce archiviata"
		} else {
			note = "Voce riattivata"
		}
	case old.Status != value.Status:
		note = crmStatusLabel(old.Status) + " → " + crmStatusLabel(value.Status)
	case old.ResumeOn != value.ResumeOn || (value.ResumeOn != "" && old.Reason != value.Reason):
		note = "Pausa aggiornata"
		if value.ResumeOn != "" {
			note = "In pausa fino al " + value.ResumeOn + " · " + value.Reason
		}
	case old.ID > 0:
		note = "Registrazione aggiornata"
	}
	if value.Kind == "opportunity" && value.Status == "lost" {
		note += "\nMotivo: " + value.Reason
	}
	if value.Kind == "opportunity" && value.Status == "won" && value.Amount != nil {
		note += fmt.Sprintf("\nValore concordato: € %d,%02d", *value.Amount/100, *value.Amount%100)
	}
	if value.Kind == "pilot" && value.Result != "" {
		note += "\nEsito: " + value.Result + "\nProssimo passo: " + value.NextStep
	}
	return crmHistory(tx, value.CompanyID, opportunityID, title, note, now, actor)
}

type crmRecordRequest struct {
	crmRecord
	FollowUp *crmRecord `json:"followUp,omitempty"`
}

func (rt *_router) crmSaveRecord(r *http.Request, actor reqcontext.RequestContext) (interface{}, error) {
	companyID, err := crmID(r, "companyID")
	if err != nil {
		return nil, err
	}
	var input crmRecordRequest
	if err := crmDecode(r, &input); err != nil {
		return nil, err
	}
	value := input.crmRecord
	value.CompanyID, value.ID = companyID, 0
	if r.Method == http.MethodPut {
		if value.ID, err = crmID(r, "recordID"); err != nil {
			return nil, err
		}
	}
	return rt.crmWrite(r.Context(), func(tx *sql.Tx) (interface{}, error) {
		company, err := crmReadCompany(tx, companyID)
		if err != nil {
			return nil, err
		}
		if company.Archived {
			return nil, crmProblem("Riattiva la scheda società prima di modificarne le voci.")
		}
		var old crmRecord
		if value.ID != 0 {
			old, err = crmReadRecord(tx, companyID, value.ID)
			if err != nil {
				return nil, err
			}
			if old.Kind == "history" || value.Kind != old.Kind || old.ContractID != value.ContractID {
				return nil, crmProblem("Questa voce o il suo collegamento non possono essere modificati.")
			}
			if old.Revision != value.Revision {
				return nil, errCRMRevision
			}
		}
		now := crmNow()
		if err := crmValidateRecord(tx, company, &value, old, now); err != nil {
			return nil, err
		}
		if value.ID == 0 {
			value.ID, err = crmInsertRecord(tx, value, crmRecordKey(value), now, actor)
		} else {
			raw, marshalErr := json.Marshal(value.crmRecordDetails)
			if marshalErr != nil {
				return nil, marshalErr
			}
			_, err = tx.Exec("UPDATE crm_records SET contact_id=?,opportunity_id=?,owner_id=?,details=?,archived=?,revision=revision+1,updated_at=? WHERE id=?",
				crmNull(value.ContactID), crmNull(value.OpportunityID), crmNull(value.OwnerID), string(raw), value.Archived, now, value.ID)
		}
		if err != nil {
			return nil, err
		}
		if input.FollowUp != nil {
			if value.Kind != "activity" || old.ID != 0 {
				return nil, crmProblem("Aggiungi il follow-up insieme a un nuovo contatto.")
			}
			task := crmRecord{CompanyID: companyID, Kind: "task", ContactID: value.ContactID, OpportunityID: value.OpportunityID,
				OwnerID: input.FollowUp.OwnerID, crmRecordDetails: crmRecordDetails{Title: input.FollowUp.Title, Type: "followup", Status: "open", Due: input.FollowUp.Due, Priority: input.FollowUp.Priority}}
			if err := crmValidateRecord(tx, company, &task, crmRecord{}, now); err != nil {
				return nil, err
			}
			task.ID, err = crmInsertRecord(tx, task, nil, now, actor)
			if err != nil {
				return nil, err
			}
			if err := crmLogRecord(tx, task, crmRecord{}, now, actor); err != nil {
				return nil, err
			}
		}
		if err := crmLogRecord(tx, value, old, now, actor); err != nil {
			return nil, err
		}
		return crmReadRecord(tx, companyID, value.ID)
	})
}

func (rt *_router) crmRenewal(r *http.Request, actor reqcontext.RequestContext) (interface{}, error) {
	companyID, err := crmID(r, "companyID")
	if err != nil {
		return nil, err
	}
	contractID, err := crmID(r, "recordID")
	if err != nil {
		return nil, err
	}
	return rt.crmWrite(r.Context(), func(tx *sql.Tx) (interface{}, error) {
		company, err := crmReadCompany(tx, companyID)
		if err != nil {
			return nil, err
		}
		contract, err := crmReadRecord(tx, companyID, contractID)
		if err != nil {
			return nil, err
		}
		if contract.Kind != "contract" || contract.Archived || company.Archived || contract.Status == "draft" {
			return nil, crmProblem("Per creare il rinnovo seleziona un contratto firmato o terminato su una scheda attiva.")
		}
		value := crmRecord{CompanyID: companyID, ContractID: contractID, ContactID: contract.ContactID, OwnerID: company.OwnerID,
			Kind: "opportunity", crmRecordDetails: crmRecordDetails{Title: "Rinnovo dopo " + contract.Season, Type: "renewal",
				Status: "to_contact", Due: contract.End, Estimated: contract.Amount, NextStep: "Pianificare il confronto di rinnovo"}}
		key := crmRecordKey(value)
		var existing int64
		err = tx.QueryRow("SELECT id FROM crm_records WHERE company_id=? AND kind='opportunity' AND stable_key=?", companyID, key).Scan(&existing)
		if err == nil {
			return crmReadRecord(tx, companyID, existing)
		}
		if err != sql.ErrNoRows {
			return nil, err
		}
		now := crmNow()
		if err := crmValidateRecord(tx, company, &value, crmRecord{}, now); err != nil {
			return nil, err
		}
		value.ID, err = crmInsertRecord(tx, value, key, now, actor)
		if err != nil {
			return nil, err
		}
		task := crmRecord{CompanyID: companyID, OpportunityID: value.ID, ContactID: value.ContactID, OwnerID: company.OwnerID,
			Kind: "task", crmRecordDetails: crmRecordDetails{Title: value.NextStep, Type: "checkin", Status: "open", Due: contract.Due, Priority: "normal"}}
		if _, err := crmInsertRecord(tx, task, nil, now, actor); err != nil {
			return nil, err
		}
		if err := crmLogRecord(tx, value, crmRecord{}, now, actor); err != nil {
			return nil, err
		}
		return crmReadRecord(tx, companyID, value.ID)
	})
}
