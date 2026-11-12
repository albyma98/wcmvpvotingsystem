package api

import (
	"database/sql"
	"encoding/json"
	"errors"
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

func (rt *_router) loadReportManualSources(eventID int, attendanceOnly bool) ([]reportManualSource, error) {
	rows, err := rt.db.SQLConn().Query(`SELECT event_id,field_key,sponsor_id,value,unit,source_type,source_description,entered_by,entered_at,updated_at,verification_status,note,revision
		FROM engagement_report_manual_sources WHERE event_id=? AND (?=0 OR (field_key='attendees' AND sponsor_id=0 AND source_type='official_attendance'))
		ORDER BY sponsor_id,field_key`, eventID, attendanceOnly)
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

func recipientSourceAdmin(w http.ResponseWriter, ctx reqcontext.RequestContext) bool {
	if ctx.AdminID <= 0 || (ctx.AdminRole != "admin" && ctx.AdminRole != "superadmin") {
		reportError(w, 403, "Le sorgenti richiedono un amministratore della società.")
		return false
	}
	return true
}

func validateManualReportSource(m reportManualSource) error {
	if m.FieldKey != "attendees" || m.SourceType != "official_attendance" || m.SponsorID != 0 {
		return fmt.Errorf("Questa operazione accetta solo il dato di base delle presenze ufficiali. I KPI vengono calcolati dal database.")
	}
	if m.Unit != "people" || math.IsNaN(m.Value) || m.Value < 0 || m.Value > 10000000 || m.Value != math.Trunc(m.Value) {
		return fmt.Errorf("Indica un numero intero di persone tra 0 e 10.000.000.")
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
	sources, err := rt.loadReportManualSources(id, false)
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
	id, _, ok := rt.reportEventScope(w, r, ctx)
	if !ok {
		return
	}
	var input struct {
		Sources []reportManualSource `json:"sources"`
	}
	if !decodeReportBody(w, r, 128*1024, &input) {
		return
	}
	if len(input.Sources) != 1 {
		reportError(w, 400, "Invia una sorgente per le presenze ufficiali.")
		return
	}
	m := input.Sources[0]
	if err := validateManualReportSource(m); err != nil {
		reportError(w, 400, err.Error())
		return
	}
	tx, err := rt.db.SQLConn().Begin()
	if err != nil {
		reportError(w, 500, "Impossibile salvare la sorgente.")
		return
	}
	defer tx.Rollback()
	err = storeOfficialAttendance(tx, id, ctx.AdminID, m, time.Now().UTC().Format(time.RFC3339Nano))
	if errors.Is(err, errMatchdayRevision) {
		reportError(w, 409, "La sorgente è stata modificata da un altro amministratore. Ricarica i dati prima di salvare.")
		return
	}
	if err != nil {
		reportError(w, 500, "Impossibile salvare la sorgente.")
		return
	}
	if err = tx.Commit(); err != nil {
		reportError(w, 500, "Impossibile confermare la sorgente.")
		return
	}
	_ = writeJSON(w, 200, map[string]interface{}{"saved": true, "message": "Presenze ufficiali salvate per l'evento. Ricalcola il report per utilizzare i nuovi dati."})
}

func storeOfficialAttendance(tx *sql.Tx, eventID, adminID int, m reportManualSource, now string) error {
	var old reportManualSource
	err := tx.QueryRow(`SELECT event_id,field_key,sponsor_id,value,unit,source_type,source_description,entered_by,entered_at,updated_at,verification_status,note,revision FROM engagement_report_manual_sources WHERE event_id=? AND field_key=? AND sponsor_id=?`, eventID, m.FieldKey, m.SponsorID).Scan(&old.EventID, &old.FieldKey, &old.SponsorID, &old.Value, &old.Unit, &old.SourceType, &old.SourceDescription, &old.EnteredBy, &old.EnteredAt, &old.UpdatedAt, &old.VerificationStatus, &old.Note, &old.Revision)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if old.Revision != m.Revision {
		return errMatchdayRevision
	}
	m.EventID, m.EnteredBy, m.EnteredAt, m.UpdatedAt, m.Revision = eventID, adminID, now, now, old.Revision+1
	if old.Revision > 0 {
		m.EnteredAt = old.EnteredAt
		m.EnteredBy = old.EnteredBy
	}
	m.SourceDescription = strings.TrimSpace(m.SourceDescription)
	m.Note = strings.TrimSpace(m.Note)
	_, err = tx.Exec(`INSERT INTO engagement_report_manual_sources(event_id,field_key,sponsor_id,value,unit,source_type,source_description,entered_by,entered_at,updated_at,verification_status,note,revision) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(event_id,field_key,sponsor_id) DO UPDATE SET value=excluded.value,unit=excluded.unit,source_type=excluded.source_type,source_description=excluded.source_description,updated_at=excluded.updated_at,verification_status=excluded.verification_status,note=excluded.note,revision=excluded.revision`, eventID, m.FieldKey, m.SponsorID, m.Value, m.Unit, m.SourceType, m.SourceDescription, m.EnteredBy, m.EnteredAt, m.UpdatedAt, m.VerificationStatus, m.Note, m.Revision)
	if err != nil {
		return err
	}
	var before interface{}
	if old.Revision > 0 {
		raw, _ := json.Marshal(old)
		before = string(raw)
	}
	after, _ := json.Marshal(m)
	_, err = tx.Exec(`INSERT INTO engagement_report_source_history(event_id,field_key,sponsor_id,previous_json,current_json,changed_by,changed_at) VALUES(?,?,?,?,?,?,?)`, eventID, m.FieldKey, m.SponsorID, before, string(after), adminID, now)
	return err
}
