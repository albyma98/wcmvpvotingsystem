package api

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
)

func manualKPISource(k reportKPI, value float64, sponsorID int) reportManualSource {
	return reportManualSource{FieldKey: "kpi:" + k.ID, SponsorID: sponsorID, Value: value, Unit: k.Unit, SourceType: "manual_kpi", SourceDescription: "Registro esterno della società", VerificationStatus: "unverified"}
}

func TestManualKPICompletesEveryMissingIndicatorWithAuditAndFrozenValues(t *testing.T) {
	f := newEngagementReportFixture(t)
	before := recipientDraft(t, f, "analytics", 0)
	sources := []reportManualSource{}
	for _, k := range before.KPIs {
		if k.Value != nil || !k.Availability.ManualAllowed {
			t.Fatalf("%s not editable when missing", k.ID)
		}
		value := float64(5)
		if k.Unit != "count" {
			value = 12.5
		}
		sources = append(sources, manualKPISource(k, value, 0))
	}
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.saveRecipientReportSources, "PUT", f.event, map[string]interface{}{"sources": sources}, f.ctx, nil), 200)
	after := recipientDraft(t, f, "marketing", 0)
	if before.DataHash == after.DataHash {
		t.Fatal("manual values did not invalidate analysis hash")
	}
	for _, k := range after.KPIs {
		if k.Value == nil || k.Source != "manual_unverified" || !isManualKPI(k) || k.Numerator != nil || k.Denominator != nil {
			t.Fatalf("manual provenance or value lost: %+v", k)
		}
		if k.Provenance[0].EnteredBy != f.ctx.AdminID || k.Provenance[0].Revision != 1 {
			t.Fatal("missing audit metadata")
		}
	}
	if after.Facts["visitors"] != nil {
		t.Fatal("direct KPI value manufactured underlying telemetry facts")
	}
	approval := engagementReportRequest(t, f.router.approveEngagementReport, "POST", f.event, map[string]interface{}{"calculation_id": after.CalculationID, "recipient": "analytics", "include_ai": true, "include_appendix": true}, f.ctx, nil)
	engagementReportRequireStatus(t, approval, 200)
	var frozen engagementReportSnapshot
	_ = json.Unmarshal(approval.Body.Bytes(), &frozen)
	// A concurrent/stale revision in a batch rolls back the entire batch.
	update := append([]reportManualSource{}, sources[:2]...)
	update[0].Value = 6
	update[0].Revision = 1
	update[1].Revision = 999
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.saveRecipientReportSources, "PUT", f.event, map[string]interface{}{"sources": update}, f.ctx, nil), 409)
	still := recipientDraft(t, f, "executive", 0)
	if *recipientTestKPI(t, still, "visitors").Value != 5 {
		t.Fatal("partially committed a conflicted batch")
	}
	var history int
	if err := f.conn.QueryRow(`SELECT COUNT(*) FROM engagement_report_source_history WHERE event_id=?`, f.event).Scan(&history); err != nil || history != len(sources) {
		t.Fatal(history, err)
	}
	update = append([]reportManualSource{}, sources[:1]...)
	update[0].Revision = 1
	update[0].Value = 7
	update[0].VerificationStatus = "verified"
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.saveRecipientReportSources, "PUT", f.event, map[string]interface{}{"sources": update}, f.ctx, nil), 200)
	if *recipientTestKPI(t, frozen, "visitors").Value != 5 {
		t.Fatal("old approved report changed")
	}
	changed := recipientDraft(t, f, "executive", 0)
	if k := recipientTestKPI(t, changed, "visitors"); *k.Value != 7 || k.Source != "manual_verified" {
		t.Fatal(k)
	}
	// New automatic observations take precedence and the conflict is explicit.
	recipientPageSignal(t, f)
	live := recipientDraft(t, f, "executive", 0)
	if k := recipientTestKPI(t, live, "visitors"); *k.Value != 1 || isManualKPI(k) || len(live.Conflicts) == 0 {
		t.Fatalf("automatic evidence silently overwritten %+v", k)
	}
	update[0].Revision = 2
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.saveRecipientReportSources, "PUT", f.event, map[string]interface{}{"sources": update}, f.ctx, nil), 409)
}

func TestManualKPIValidationUnitsLimitsAndSponsorIsolation(t *testing.T) {
	f := newEngagementReportFixture(t)
	reach := buildReportKPIs(nil)[13]
	create := func(org, position int, name string) int {
		id, err := f.router.db.CreateSponsor(database.Sponsor{OrganizationID: org, Position: position, Name: name, LogoData: "logo", IsActive: true})
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	first := create(f.org.ID, 1, "A")
	second := create(f.org.ID, 2, "B")
	foreign := create(f.other.ID, 1, "Other")
	aggregate := manualKPISource(reach, 99, 0)
	dedicated := manualKPISource(reach, 10, first)
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.saveRecipientReportSources, "PUT", f.event, map[string]interface{}{"sources": []reportManualSource{aggregate, dedicated}}, f.ctx, nil), 200)
	for _, tc := range []struct {
		sponsor int
		want    float64
		missing bool
	}{{0, 99, false}, {first, 10, false}, {second, 0, true}} {
		s := recipientDraft(t, f, "sponsor", tc.sponsor)
		k := recipientTestKPI(t, s, "sponsor_reach")
		if tc.missing {
			if k.Value != nil {
				t.Fatal("attributed another sponsor's manual value")
			}
		} else if k.Value == nil || *k.Value != tc.want {
			t.Fatal(k)
		}
	}
	dedicated.SponsorID = foreign
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.saveRecipientReportSources, "PUT", f.event, map[string]interface{}{"sources": []reportManualSource{dedicated}}, f.ctx, nil), 404)
	for _, m := range []reportManualSource{
		manualKPISource(reportKPI{ID: "unknown", Unit: "count"}, 1, 0),
		manualKPISource(reportKPI{ID: "visitors", Unit: "count"}, 1.5, 0),
		manualKPISource(reportKPI{ID: "active_time", Unit: "seconds"}, -1, 0),
		manualKPISource(reportKPI{ID: "sponsor_ctr", Unit: "percent"}, 100.1, 0),
		manualKPISource(reportKPI{ID: "visitors", Unit: "count"}, 1, first),
		manualKPISource(reportKPI{ID: "visitors", Unit: "percent"}, 1, 0),
	} {
		engagementReportRequireStatus(t, engagementReportRequest(t, f.router.saveRecipientReportSources, "PUT", f.event, map[string]interface{}{"sources": []reportManualSource{m}}, f.ctx, nil), 400)
	}
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
	sources, err := f.router.loadReportManualSources(f.event)
	if err != nil || len(sources) != 1 || sources[0].Value != 123 || sources[0].Revision != 3 || sources[0].Note != "Original note" || sources[0].SponsorID != 0 {
		t.Fatal(sources, err)
	}
	k := manualKPISource(reportKPI{ID: "active_time", Unit: "seconds"}, 12.5, 0)
	r := engagementReportRequest(t, f.router.saveRecipientReportSources, "PUT", f.event, map[string]interface{}{"sources": []reportManualSource{k}}, f.ctx, nil)
	engagementReportRequireStatus(t, r, 200)
	if strings.Contains(r.Body.String(), "error") {
		t.Fatal(r.Body.String())
	}
}
