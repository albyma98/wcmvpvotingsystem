package api

import (
	"net/http"
	"testing"
)

func TestReportKeepsHistoricalManualValuesOutOfNewCalculations(t *testing.T) {
	f := newEngagementReportFixture(t)
	if _, err := f.conn.Exec(`INSERT INTO engagement_report_manual_sources
		(event_id,field_key,value,unit,source_type,source_description,entered_by,entered_at,updated_at,verification_status,revision)
		VALUES (?,'kpi:visitors',500,'count','manual_kpi','Registro storico',77,'2026-10-20','2026-10-20','verified',1)`, f.event); err != nil {
		t.Fatal(err)
	}
	snapshot := recipientDraft(t, f, "executive", 0)
	if k := recipientTestKPI(t, snapshot, "visitors"); k.Value != nil || k.Availability.ManualAllowed || len(snapshot.ManualSources) != 0 {
		t.Fatalf("historical override was used: %+v", k)
	}
	sources, err := f.router.loadReportManualSources(f.event, false)
	if err != nil || len(sources) != 1 || sources[0].Value != 500 {
		t.Fatal(sources, err)
	}
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.saveRecipientReportSources, http.MethodPut, f.event,
		map[string]interface{}{"sources": sources}, f.ctx, nil), http.StatusBadRequest)
}

func TestManualKPIMigrationPreservesAttendanceAndIsIdempotent(t *testing.T) {
	f := newEngagementReportFixture(t)
	// Reconstruct only the old reporting source tables in this isolated fixture.
	if _, err := f.conn.Exec(`DROP TABLE engagement_report_manual_sources; DROP TABLE engagement_report_source_history;` + recipientReportMigration); err != nil {
		t.Fatal(err)
	}
	if _, err := f.conn.Exec(`INSERT INTO engagement_report_manual_sources(event_id,field_key,value,unit,source_type,source_description,entered_by,entered_at,updated_at,verification_status,note,revision) VALUES(?,'attendees',123,'people','official_attendance','Legacy source',77,'2026-10-20','2026-10-21','verified','Original note',3)`, f.event); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := f.router.ensureRecipientReportTables(); err != nil {
			t.Fatal(err)
		}
	}
	sources, err := f.router.loadReportManualSources(f.event, true)
	if err != nil || len(sources) != 1 || sources[0].Value != 123 || sources[0].Revision != 3 || sources[0].Note != "Original note" || sources[0].SponsorID != 0 {
		t.Fatal(sources, err)
	}
}
