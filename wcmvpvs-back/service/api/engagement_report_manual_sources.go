package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
)

type reportManualSource struct {
	EventID            int     `json:"event_id"`
	FieldKey           string  `json:"field_key"`
	SponsorID          int     `json:"sponsor_id,omitempty"`
	Value              float64 `json:"value"`
	Unit               string  `json:"unit"`
	SourceType         string  `json:"source_type"`
	SourceDescription  string  `json:"source_description"`
	EnteredBy          int     `json:"entered_by"`
	EnteredAt          string  `json:"entered_at"`
	UpdatedAt          string  `json:"updated_at"`
	VerificationStatus string  `json:"verification_status"`
	Note               string  `json:"note"`
	Revision           int     `json:"revision"`
}

func (rt *_router) loadReportManualSources(eventID int) ([]reportManualSource, error) {
	rows, err := rt.db.SQLConn().Query(`SELECT event_id,field_key,sponsor_id,value,unit,source_type,source_description,entered_by,entered_at,updated_at,verification_status,note,revision FROM engagement_report_manual_sources WHERE event_id=? ORDER BY sponsor_id,field_key`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []reportManualSource{}
	for rows.Next() {
		var m reportManualSource
		if err = rows.Scan(&m.EventID, &m.FieldKey, &m.SponsorID, &m.Value, &m.Unit, &m.SourceType, &m.SourceDescription, &m.EnteredBy, &m.EnteredAt, &m.UpdatedAt, &m.VerificationStatus, &m.Note, &m.Revision); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func manualKPIDefinition(fieldKey string) (reportKPI, bool) {
	if !strings.HasPrefix(fieldKey, "kpi:") {
		return reportKPI{}, false
	}
	for _, k := range buildReportKPIs(nil) {
		if fieldKey == "kpi:"+k.ID {
			return k, true
		}
	}
	return reportKPI{}, false
}

func manualKPIField(k reportKPI) reportSourceField {
	kind, max := "number", float64(1e12)
	if k.Unit == "count" {
		kind = "integer"
	}
	if k.Unit == "percent" {
		max = 100
	}
	return reportSourceField{Key: "kpi:" + k.ID, Label: k.Label, Type: kind, Unit: k.Unit, Min: 0, Max: max, Validation: "Valore KPI manuale con fonte obbligatoria, stato di verifica e storico; conteggi interi e percentuali tra zero e cento."}
}

func manualKPIScope(k reportKPI, sponsorID int) int {
	if k.Group == "Sponsor" {
		return sponsorID
	}
	return 0
}

func filterReportManualSources(sources []reportManualSource, sponsorID int) []reportManualSource {
	out := []reportManualSource{}
	for _, m := range sources {
		if m.FieldKey == "attendees" && m.SponsorID == 0 {
			out = append(out, m)
			continue
		}
		if k, ok := manualKPIDefinition(m.FieldKey); ok && m.SponsorID == manualKPIScope(k, sponsorID) {
			out = append(out, m)
		}
	}
	return out
}

func isManualKPI(k reportKPI) bool {
	for _, m := range k.Provenance {
		if m.SourceType == "manual_kpi" {
			return true
		}
	}
	return false
}

func recipientSourceAdmin(w http.ResponseWriter, ctx reqcontext.RequestContext) bool {
	if ctx.AdminID <= 0 || (ctx.AdminRole != "admin" && ctx.AdminRole != "superadmin") {
		reportError(w, 403, "Le sorgenti richiedono un amministratore della società.")
		return false
	}
	return true
}

func validateManualReportSource(m reportManualSource) error {
	if m.SponsorID < 0 {
		return fmt.Errorf("Ambito sponsor non valido.")
	}
	field := reportSourceField{Type: "integer", Unit: "people", Min: 0, Max: 10000000}
	if m.FieldKey == "attendees" {
		if m.SourceType != "official_attendance" || m.SponsorID != 0 {
			return fmt.Errorf("Fonte presenze o ambito non valido.")
		}
	} else {
		k, ok := manualKPIDefinition(m.FieldKey)
		if !ok || m.SourceType != "manual_kpi" || m.SponsorID != manualKPIScope(k, m.SponsorID) {
			return fmt.Errorf("KPI manuale o ambito non valido.")
		}
		field = manualKPIField(k)
	}
	if m.Unit != field.Unit || math.IsNaN(m.Value) || math.IsInf(m.Value, 0) || m.Value < field.Min || m.Value > field.Max || field.Type == "integer" && m.Value != math.Trunc(m.Value) {
		return fmt.Errorf("Valore o unità non validi: rispetta i limiti del KPI e usa conteggi interi.")
	}
	if utf8.RuneCountInString(strings.TrimSpace(m.SourceDescription)) < 3 || utf8.RuneCountInString(m.SourceDescription) > 500 || strings.ContainsRune(m.SourceDescription, '\x00') || utf8.RuneCountInString(m.Note) > 1000 {
		return fmt.Errorf("Inserisci una fonte da tre a cinquecento caratteri e una nota entro mille caratteri.")
	}
	if m.VerificationStatus != "verified" && m.VerificationStatus != "unverified" {
		return fmt.Errorf("Stato di verifica non valido.")
	}
	if m.Revision < 0 {
		return fmt.Errorf("Revisione non valida.")
	}
	return nil
}

func (rt *_router) getRecipientReportSources(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !recipientSourceAdmin(w, ctx) {
		return
	}
	id, _, ok := rt.reportEventScope(w, r, ctx)
	if !ok {
		return
	}
	sources, err := rt.loadReportManualSources(id)
	if err != nil {
		reportError(w, 500, "Impossibile caricare le sorgenti.")
		return
	}
	rows, err := rt.db.SQLConn().Query(`SELECT previous_json,current_json,changed_by,changed_at,sponsor_id,field_key FROM engagement_report_source_history WHERE event_id=? ORDER BY id DESC`, id)
	if err != nil {
		reportError(w, 500, "Impossibile caricare lo storico.")
		return
	}
	defer rows.Close()
	type change struct {
		Previous  json.RawMessage `json:"previous"`
		Current   json.RawMessage `json:"current"`
		By        int             `json:"changed_by"`
		At        string          `json:"changed_at"`
		SponsorID int             `json:"sponsor_id"`
		FieldKey  string          `json:"field_key"`
	}
	history := []change{}
	for rows.Next() {
		var c change
		var prev sql.NullString
		var current string
		if err = rows.Scan(&prev, &current, &c.By, &c.At, &c.SponsorID, &c.FieldKey); err != nil {
			reportError(w, 500, "Storico non disponibile.")
			return
		}
		c.Current = json.RawMessage(current)
		if prev.Valid {
			c.Previous = json.RawMessage(prev.String)
		}
		history = append(history, c)
	}
	if rows.Err() != nil {
		reportError(w, 500, "Storico non disponibile.")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	_ = writeJSON(w, 200, map[string]interface{}{"sources": sources, "history": history, "precedence": []string{"automatic", "manual_verified", "manual_unverified", "unavailable"}})
}

func (rt *_router) saveRecipientReportSources(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !recipientSourceAdmin(w, ctx) {
		return
	}
	id, org, ok := rt.reportEventScope(w, r, ctx)
	if !ok {
		return
	}
	var input struct {
		Sources []reportManualSource `json:"sources"`
	}
	if !decodeReportBody(w, r, 128*1024, &input) {
		return
	}
	if len(input.Sources) == 0 || len(input.Sources) > 31 {
		reportError(w, 400, "Invia da uno a trentuno completamenti.")
		return
	}
	seen := map[string]bool{}
	for _, m := range input.Sources {
		if err := validateManualReportSource(m); err != nil {
			reportError(w, 400, err.Error())
			return
		}
		key := fmt.Sprintf("%s/%d", m.FieldKey, m.SponsorID)
		if seen[key] {
			reportError(w, 400, "Completamento duplicato.")
			return
		}
		seen[key] = true
		if m.SponsorID > 0 {
			var exists int
			if rt.db.SQLConn().QueryRow(`SELECT id FROM sponsors WHERE id=? AND organization_id=?`, m.SponsorID, org).Scan(&exists) != nil {
				reportError(w, 404, "Sponsor non disponibile per questa società.")
				return
			}
		}
	}
	// Check the current automatic evidence before committing any manual value.
	live := map[int]engagementReportSnapshot{}
	for _, m := range input.Sources {
		s, loaded := live[m.SponsorID]
		if !loaded {
			var err error
			s, err = rt.buildRecipientReport(id, org, m.SponsorID)
			if err != nil {
				reportError(w, 500, "Impossibile verificare i dati automatici.")
				return
			}
			live[m.SponsorID] = s
		}
		if m.FieldKey == "attendees" {
			if s.OriginalFacts["attendees"] != nil {
				reportError(w, 409, "Esistono presenze automatiche: nessuna sovrascrittura manuale consentita.")
				return
			}
			continue
		}
		for _, k := range s.KPIs {
			if m.FieldKey == "kpi:"+k.ID && k.Value != nil && !isManualKPI(k) {
				reportError(w, 409, "Il KPI dispone di dati calcolabili: aggiorna il report prima di completarlo manualmente.")
				return
			}
		}
	}
	tx, err := rt.db.SQLConn().Begin()
	if err != nil {
		reportError(w, 500, "Impossibile salvare la sorgente.")
		return
	}
	defer tx.Rollback()
	for _, m := range input.Sources {
		var old reportManualSource
		err = tx.QueryRow(`SELECT event_id,field_key,sponsor_id,value,unit,source_type,source_description,entered_by,entered_at,updated_at,verification_status,note,revision FROM engagement_report_manual_sources WHERE event_id=? AND field_key=? AND sponsor_id=?`, id, m.FieldKey, m.SponsorID).Scan(&old.EventID, &old.FieldKey, &old.SponsorID, &old.Value, &old.Unit, &old.SourceType, &old.SourceDescription, &old.EnteredBy, &old.EnteredAt, &old.UpdatedAt, &old.VerificationStatus, &old.Note, &old.Revision)
		if err != nil && err != sql.ErrNoRows {
			reportError(w, 500, "Impossibile leggere la sorgente.")
			return
		}
		if old.Revision != m.Revision {
			reportError(w, 409, "La sorgente è stata modificata da un altro amministratore. Ricarica i dati prima di salvare.")
			return
		}
		now := time.Now().UTC().Format(time.RFC3339Nano)
		m.EventID, m.EnteredBy, m.EnteredAt, m.UpdatedAt, m.Revision = id, ctx.AdminID, now, now, old.Revision+1
		if old.Revision > 0 {
			m.EnteredAt = old.EnteredAt
			m.EnteredBy = old.EnteredBy
		}
		m.SourceDescription = strings.TrimSpace(m.SourceDescription)
		m.Note = strings.TrimSpace(m.Note)
		_, err = tx.Exec(`INSERT INTO engagement_report_manual_sources(event_id,field_key,sponsor_id,value,unit,source_type,source_description,entered_by,entered_at,updated_at,verification_status,note,revision) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(event_id,field_key,sponsor_id) DO UPDATE SET value=excluded.value,unit=excluded.unit,source_type=excluded.source_type,source_description=excluded.source_description,updated_at=excluded.updated_at,verification_status=excluded.verification_status,note=excluded.note,revision=excluded.revision`, id, m.FieldKey, m.SponsorID, m.Value, m.Unit, m.SourceType, m.SourceDescription, m.EnteredBy, m.EnteredAt, m.UpdatedAt, m.VerificationStatus, m.Note, m.Revision)
		if err != nil {
			reportError(w, 500, "Impossibile salvare la sorgente.")
			return
		}
		var before interface{}
		if old.Revision > 0 {
			raw, _ := json.Marshal(old)
			before = string(raw)
		}
		after, _ := json.Marshal(m)
		_, err = tx.Exec(`INSERT INTO engagement_report_source_history(event_id,field_key,sponsor_id,previous_json,current_json,changed_by,changed_at) VALUES(?,?,?,?,?,?,?)`, id, m.FieldKey, m.SponsorID, before, string(after), ctx.AdminID, now)
		if err != nil {
			reportError(w, 500, "Impossibile salvare lo storico.")
			return
		}
	}
	if err = tx.Commit(); err != nil {
		reportError(w, 500, "Impossibile confermare la sorgente.")
		return
	}
	_ = writeJSON(w, 200, map[string]interface{}{"saved": true, "message": "Completamenti salvati per l'evento. Ricalcola il report per utilizzare i nuovi dati."})
}
