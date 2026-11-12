package api

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mattn/go-sqlite3"
)

//go:embed report_migrations/004_matchday_kpi_manual.sql
var matchdayManualMigration string

var errMatchdayRevision = errors.New("stale matchday revision")

type matchdayManualRevision struct {
	Value     *float64 `json:"value,omitempty"`
	Note      string   `json:"note,omitempty"`
	UpdatedAt string   `json:"updated_at,omitempty"`
	UpdatedBy int      `json:"updated_by,omitempty"`
	Revision  int      `json:"revision"`
}

type matchdayManualField struct {
	Code        string `json:"code"`
	Label       string `json:"label"`
	Unit        string `json:"unit"`
	Description string `json:"description"`
	matchdayManualRevision
	History []matchdayManualRevision `json:"history"`
}

type matchdayManualInput struct {
	Code     string   `json:"code"`
	Value    *float64 `json:"value"`
	Note     string   `json:"note"`
	Revision int      `json:"revision"`
}

var matchdayManualDefinitions = []matchdayManualField{
	{Code: "attendees", Label: "Spettatori presenti", Unit: "count", Description: "Presenze fisiche ufficiali della partita. Dato condiviso con il report di engagement."},
	{Code: "marketing_spend", Label: "Spesa promozionale", Unit: "eur", Description: "Spesa di acquisizione attribuita alla partita, in euro."},
	{Code: "qr_scans", Label: "Scansioni QR tracciate", Unit: "count", Description: "Scansioni effettive attribuite all’evento da un contatore esterno, per la stessa campagna dei QR tracciati."},
	{Code: "offline_revenue", Label: "Ricavi offline confermati", Unit: "eur", Description: "Incassi attribuiti all’evento da una fonte esterna, al netto dei rimborsi e distinti dagli incassi digitali."},
	{Code: "shop_paid_orders", Label: "Acquisti shop confermati", Unit: "count", Description: "Ordini pagati e attribuiti alla partita da una fonte esterna, esclusi rimborsi, annullamenti e duplicati."},
	{Code: "shop_gmv", Label: "Valore lordo shop confermato", Unit: "eur", Description: "Valore in euro degli stessi ordini shop confermati. Esclude i rimborsi e le prenotazioni non pagate."},
	{Code: "shop_reservations", Label: "Prenotazioni shop confermate", Unit: "count", Description: "Prenotazioni confermate e attribuite alla partita, conteggiate separatamente dagli acquisti."},
}

func matchdayManualDefinition(code string) (matchdayManualField, bool) {
	for _, field := range matchdayManualDefinitions {
		if field.Code == code {
			return field, true
		}
	}
	return matchdayManualField{}, false
}

func registerMatchdayKPIRoutes(rt *_router) {
	if _, err := rt.db.SQLConn().Exec(matchdayManualMigration); err != nil {
		rt.baseLogger.WithError(err).Error("cannot initialize Matchday KPI manual sources")
	}
	rt.router.Get("/admin/events/{eventId}/matchday-kpis", rt.wrapAdmin(rt.handleMatchdayKPI))
	rt.router.Put("/admin/events/{eventId}/matchday-kpis", rt.wrapAdmin(rt.handleMatchdayKPI))
}

func isMatchdaySQLiteBusy(err error) bool {
	var e sqlite3.Error
	return errors.As(err, &e) && (e.Code == sqlite3.ErrBusy || e.Code == sqlite3.ErrLocked)
}

func validateMatchdayManual(input []matchdayManualInput) error {
	if len(input) == 0 || len(input) > len(matchdayManualDefinitions) {
		return errors.New("Invia almeno un dato integrativo valido.")
	}
	seen := make(map[string]bool, len(input))
	for _, row := range input {
		definition, exists := matchdayManualDefinition(row.Code)
		if !exists || seen[row.Code] || row.Revision < 0 || row.Value == nil {
			return errors.New("Dato integrativo o revisione non validi.")
		}
		seen[row.Code] = true
		value := *row.Value
		max := 1e9
		if row.Code == "attendees" {
			max = 1e7
		}
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > max ||
			(definition.Unit == "count" && value != math.Trunc(value)) ||
			(definition.Unit == "eur" && math.Abs(value*100-math.Round(value*100)) > 0.0001) {
			return fmt.Errorf("Valore non valido per %s.", definition.Label)
		}
		length := utf8.RuneCountInString(strings.TrimSpace(row.Note))
		if length < 3 || length > 500 || strings.ContainsRune(row.Note, '\x00') {
			return fmt.Errorf("Indica la fonte di %s (da 3 a 500 caratteri).", definition.Label)
		}
	}
	return nil
}

func saveMatchdayManual(ctx context.Context, tx *sql.Tx, orgID, eventID, adminID int, input []matchdayManualInput) error {
	var archived int
	if err := tx.QueryRowContext(ctx, `SELECT id FROM events WHERE organization_id=? AND id=? AND is_concluded=1`, orgID, eventID).Scan(&archived); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, item := range input {
		if item.Code == "attendees" {
			m := reportManualSource{FieldKey: "attendees", Value: *item.Value, Unit: "people", SourceType: "official_attendance",
				SourceDescription: strings.TrimSpace(item.Note), VerificationStatus: "unverified", Revision: item.Revision}
			if err := storeOfficialAttendance(tx, eventID, adminID, m, now); err != nil {
				return err
			}
			continue
		}
		definition, _ := matchdayManualDefinition(item.Code)
		result, err := tx.ExecContext(ctx, `INSERT INTO matchday_kpi_manual_revisions
			(organization_id,event_id,metric_code,revision,value,unit,source_note,updated_at,updated_by)
			SELECT ?,?,?,?,?,?,?,?,? WHERE ?=(SELECT COALESCE(MAX(revision),0) FROM matchday_kpi_manual_revisions
			WHERE organization_id=? AND event_id=? AND metric_code=?)`, orgID, eventID, item.Code, item.Revision+1,
			*item.Value, definition.Unit, strings.TrimSpace(item.Note), now, adminID, item.Revision, orgID, eventID, item.Code)
		if err != nil {
			return err
		}
		if n, err := result.RowsAffected(); err != nil {
			return err
		} else if n != 1 {
			return errMatchdayRevision
		}
	}
	return nil
}

func loadMatchdayManual(ctx context.Context, q matchdayQueryer, orgID int, data map[int]*matchdayFacts) error {
	in, args := matchdayIDs(data)
	args = append([]interface{}{orgID}, args...)
	if err := matchdayRead(ctx, q, `SELECT m.event_id,m.metric_code,m.value,m.source_note,m.updated_at,m.updated_by,m.revision
		FROM matchday_kpi_manual_revisions m WHERE m.organization_id=? AND m.event_id IN (`+in+`)
		AND m.revision=(SELECT MAX(p.revision) FROM matchday_kpi_manual_revisions p
		WHERE p.organization_id=m.organization_id AND p.event_id=m.event_id AND p.metric_code=m.metric_code)`, args, func(rows *sql.Rows) error {
		var eventID int
		var code string
		var r matchdayManualRevision
		if err := rows.Scan(&eventID, &code, &r.Value, &r.Note, &r.UpdatedAt, &r.UpdatedBy, &r.Revision); err != nil {
			return err
		}
		if f, ok := matchdayManualDefinition(code); ok {
			f.matchdayManualRevision = r
			data[eventID].Manual[code] = f
		}
		return nil
	}); err != nil {
		return err
	}
	return matchdayRead(ctx, q, `SELECT m.event_id,m.value,m.source_description,m.updated_at,
		COALESCE((SELECT h.changed_by FROM engagement_report_source_history h WHERE h.event_id=m.event_id
		AND h.field_key='attendees' AND h.sponsor_id=0 ORDER BY h.id DESC LIMIT 1),m.entered_by),m.revision
		FROM engagement_report_manual_sources m JOIN events e ON e.id=m.event_id WHERE e.organization_id=?
		AND m.event_id IN (`+in+`) AND m.field_key='attendees' AND m.sponsor_id=0 AND m.source_type='official_attendance'`, args, func(rows *sql.Rows) error {
		var eventID int
		f, _ := matchdayManualDefinition("attendees")
		if err := rows.Scan(&eventID, &f.Value, &f.Note, &f.UpdatedAt, &f.UpdatedBy, &f.Revision); err != nil {
			return err
		}
		data[eventID].Manual[f.Code] = f
		return nil
	})
}

func loadMatchdayManualHistory(ctx context.Context, q matchdayQueryer, orgID int, f *matchdayFacts) error {
	if err := matchdayRead(ctx, q, `SELECT metric_code,value,source_note,updated_at,updated_by,revision
		FROM matchday_kpi_manual_revisions WHERE organization_id=? AND event_id=? ORDER BY metric_code,revision DESC`, []interface{}{orgID, f.Event.ID}, func(rows *sql.Rows) error {
		var code string
		var revision matchdayManualRevision
		if err := rows.Scan(&code, &revision.Value, &revision.Note, &revision.UpdatedAt, &revision.UpdatedBy, &revision.Revision); err != nil {
			return err
		}
		if field, ok := f.Manual[code]; ok && revision.Revision < field.Revision {
			field.History = append(field.History, revision)
			f.Manual[code] = field
		}
		return nil
	}); err != nil {
		return err
	}
	field, exists := f.Manual["attendees"]
	if !exists {
		return nil
	}
	err := matchdayRead(ctx, q, `SELECT h.current_json,h.changed_at,h.changed_by FROM engagement_report_source_history h
		JOIN events e ON e.id=h.event_id WHERE e.organization_id=? AND h.event_id=? AND h.field_key='attendees' AND h.sponsor_id=0
		ORDER BY h.id DESC`, []interface{}{orgID, f.Event.ID}, func(rows *sql.Rows) error {
		var raw string
		var r matchdayManualRevision
		if err := rows.Scan(&raw, &r.UpdatedAt, &r.UpdatedBy); err != nil {
			return err
		}
		var source reportManualSource
		if err := json.Unmarshal([]byte(raw), &source); err != nil {
			return err
		}
		if source.Revision < field.Revision && source.SourceType == "official_attendance" {
			r.Value, r.Note, r.Revision = reportMetricFloat(source.Value), source.SourceDescription, source.Revision
			field.History = append(field.History, r)
		}
		return nil
	})
	f.Manual[field.Code] = field
	return err
}
