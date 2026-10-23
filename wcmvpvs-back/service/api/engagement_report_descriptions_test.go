package api

import (
	"net/http"
	"strings"
	"testing"
	"unicode/utf8"
)

func reportDescriptionKPI(t *testing.T, snapshot engagementReportSnapshot, id string) reportKPI {
	t.Helper()
	for _, kpi := range snapshot.KPIs {
		if kpi.ID == id {
			return kpi
		}
	}
	t.Fatalf("missing KPI %s", id)
	return reportKPI{}
}

func TestEngagementReportValuesCanBeCorrectedWithoutReasonsAndDescriptionsAreOptional(t *testing.T) {
	f := newEngagementReportFixture(t)
	draft := engagementReportDecodeSnapshot(t, engagementReportRequest(t, f.router.prepareEngagementReport, http.MethodGet, f.event, nil, f.ctx, nil))
	for _, k := range draft.KPIs {
		if strings.TrimSpace(k.DisplayDescription) == "" || utf8.RuneCountInString(k.DisplayDescription) > 120 {
			t.Fatalf("missing/long default caption %+v", k)
		}
	}
	input := map[string]interface{}{
		"calculation_id": draft.CalculationID, "format": "one_page",
		"facts":        map[string]*float64{"visitors": reportMetricFloat(5)},
		"overrides":    map[string]reportOverride{"active_participation": {Value: reportMetricFloat(70)}},
		"descriptions": map[string]string{"visitors": " Pubblico   verificato dalla società. ", "satisfaction": "", "new_profiles": strings.Repeat("è", 120)},
	}
	approved := engagementReportDecodeSnapshot(t, engagementReportRequest(t, f.router.approveEngagementReport, http.MethodPost, f.event, input, f.ctx, nil))
	visitors := reportDescriptionKPI(t, approved, "visitors")
	active := reportDescriptionKPI(t, approved, "active_participation")
	if visitors.Value == nil || *visitors.Value != 5 || !visitors.Corrected || visitors.Source != "manual" || visitors.Note != "" || visitors.DisplayDescription != "Pubblico verificato dalla società." {
		t.Fatalf("correction without reason failed %+v", visitors)
	}
	if active.Value == nil || *active.Value != 70 || active.Numerator != nil || active.Denominator != nil || !active.Corrected || active.Note != "" {
		t.Fatalf("override without reason failed %+v", active)
	}
	if reportDescriptionKPI(t, approved, "satisfaction").DisplayDescription != "" {
		t.Fatal("explicit empty description was replaced")
	}
	if utf8.RuneCountInString(reportDescriptionKPI(t, approved, "new_profiles").DisplayDescription) != 120 {
		t.Fatal("Unicode character limit counted bytes")
	}
	if reportDescriptionKPI(t, approved, "visitors").Description != reportDescriptionKPI(t, draft, "visitors").Description {
		t.Fatal("caption replaced calculation definition")
	}
	if reportDescriptionKPI(t, approved, "sponsor_ctr").DisplayDescription != reportDescriptionKPI(t, draft, "sponsor_ctr").DisplayDescription {
		t.Fatal("omitted caption was lost")
	}
	if approved.Notes != "" {
		t.Fatal("approval invented a correction reason")
	}
}

func TestEngagementReportDescriptionOnlyVersionsAreFrozenAndIdempotent(t *testing.T) {
	f := newEngagementReportFixture(t)
	draft := engagementReportDecodeSnapshot(t, engagementReportRequest(t, f.router.prepareEngagementReport, http.MethodGet, f.event, nil, f.ctx, nil))
	input := map[string]interface{}{"calculation_id": draft.CalculationID, "format": "extended", "descriptions": map[string]string{"visitors": "Pubblico dell'evento."}}
	first := engagementReportDecodeSnapshot(t, engagementReportRequest(t, f.router.approveEngagementReport, http.MethodPost, f.event, input, f.ctx, nil))
	k := reportDescriptionKPI(t, first, "visitors")
	if k.Corrected || k.Source != reportDescriptionKPI(t, draft, "visitors").Source {
		t.Fatalf("caption edit changed measured provenance %+v", k)
	}
	// Explicitly passing unchanged defaults has the same effective content as
	// omitting them, so the approved version must remain the same on retry.
	all := map[string]string{}
	for _, item := range first.KPIs {
		all[item.ID] = item.DisplayDescription
	}
	input["descriptions"] = all
	retry := engagementReportDecodeSnapshot(t, engagementReportRequest(t, f.router.approveEngagementReport, http.MethodPost, f.event, input, f.ctx, nil))
	if retry.ID != first.ID {
		t.Fatal("equivalent captions created a duplicate approval")
	}
	all["visitors"] = ""
	second := engagementReportDecodeSnapshot(t, engagementReportRequest(t, f.router.approveEngagementReport, http.MethodPost, f.event, input, f.ctx, nil))
	if second.ID == first.ID || reportDescriptionKPI(t, second, "visitors").DisplayDescription != "" {
		t.Fatal("caption removal did not create its own version")
	}
	latest := engagementReportDecodeSnapshot(t, engagementReportRequest(t, f.router.prepareEngagementReport, http.MethodGet, f.event, nil, f.ctx, nil))
	if latest.LatestReport == nil || latest.LatestReport.ID != second.ID || reportDescriptionKPI(t, *latest.LatestReport, "visitors").DisplayDescription != "" {
		t.Fatal("downloadable approved version lost optional captions")
	}
	if reportDescriptionKPI(t, first, "visitors").DisplayDescription != "Pubblico dell'evento." || reportDescriptionKPI(t, latest, "visitors").DisplayDescription == "" {
		t.Fatal("caption review changed original/default calculation")
	}
}

func TestEngagementReportRejectsInvalidDescriptionKeysAndLengths(t *testing.T) {
	f := newEngagementReportFixture(t)
	draft := engagementReportDecodeSnapshot(t, engagementReportRequest(t, f.router.prepareEngagementReport, http.MethodGet, f.event, nil, f.ctx, nil))
	for _, descriptions := range []map[string]string{
		{"unknown": "Text"}, {"visitors": strings.Repeat("è", 121)}, {"visitors": "Text\x00"},
	} {
		input := map[string]interface{}{"calculation_id": draft.CalculationID, "format": "one_page", "descriptions": descriptions}
		engagementReportRequireStatus(t, engagementReportRequest(t, f.router.approveEngagementReport, http.MethodPost, f.event, input, f.ctx, nil), http.StatusBadRequest)
	}
	var stored int
	if err := f.conn.QueryRow(`SELECT COUNT(*) FROM engagement_report_snapshots`).Scan(&stored); err != nil || stored != 0 {
		t.Fatalf("invalid caption persisted: %d %v", stored, err)
	}
}
