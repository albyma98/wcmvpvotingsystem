package api

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
	"time"
)

func TestEngagementReportSelectionIsFrozenCanonicalAndDoesNotDiscardOtherFacts(t *testing.T) {
	f := newEngagementReportFixture(t)
	page := engagementReportSignalPayload("estimate-page", "page_view", map[string]interface{}{"vote_available": true})
	vote := engagementReportSignalPayload("estimate-vote", "vote_completed", nil)
	page["occurred_at"] = time.Now().UTC().Add(-2 * time.Minute).Format(time.RFC3339)
	vote["occurred_at"] = time.Now().UTC().Add(-time.Minute).Format(time.RFC3339)
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.recordReportSignals, http.MethodPost, f.event, map[string]interface{}{"events": []interface{}{page, vote}}, f.ctx, map[string]string{"X-Device-ID": "time-estimation-guest"}), http.StatusNoContent)
	draft := engagementReportDecodeSnapshot(t, engagementReportRequest(t, f.router.prepareEngagementReport, http.MethodGet, f.event, nil, f.ctx, nil))
	if len(draft.KPIs) != 30 || len(draft.SelectedKPIIDs) != 20 {
		t.Fatalf("wrong selectable defaults %d %+v", len(draft.KPIs), draft.SelectedKPIIDs)
	}
	input := map[string]interface{}{"calculation_id": draft.CalculationID, "format": "one_page", "selected_kpi_ids": []string{"feedback_collected", "estimated_total_time", "visitors"}, "descriptions": map[string]string{"sponsor_reach": "Testo conservato anche se non selezionato."}}
	approved := engagementReportDecodeSnapshot(t, engagementReportRequest(t, f.router.approveEngagementReport, http.MethodPost, f.event, input, f.ctx, nil))
	want := []string{"visitors", "estimated_total_time", "feedback_collected"}
	if !reflect.DeepEqual(approved.SelectedKPIIDs, want) || len(approved.KPIs) != 30 {
		t.Fatalf("selection discarded measurements %+v", approved.SelectedKPIIDs)
	}
	if reportDescriptionKPI(t, approved, "sponsor_reach").DisplayDescription != "Testo conservato anche se non selezionato." {
		t.Fatal("unselected caption was discarded")
	}
	estimate := reportDescriptionKPI(t, approved, "estimated_total_time")
	if estimate.Value == nil || *estimate.Value != 60 || estimate.Source != "estimated" || estimate.Corrected {
		t.Fatalf("missing distinct estimate %+v", estimate)
	}
	input["selected_kpi_ids"] = want
	retry := engagementReportDecodeSnapshot(t, engagementReportRequest(t, f.router.approveEngagementReport, http.MethodPost, f.event, input, f.ctx, nil))
	if retry.ID != approved.ID {
		t.Fatal("selection order created duplicate approval")
	}
	input["selected_kpi_ids"] = []string{"visitors"}
	changed := engagementReportDecodeSnapshot(t, engagementReportRequest(t, f.router.approveEngagementReport, http.MethodPost, f.event, input, f.ctx, nil))
	if changed.ID == approved.ID || len(changed.SelectedKPIIDs) != 1 {
		t.Fatal("changed selection did not create a new version")
	}
	latest := engagementReportDecodeSnapshot(t, engagementReportRequest(t, f.router.prepareEngagementReport, http.MethodGet, f.event, nil, f.ctx, nil))
	if latest.LatestReport == nil || !reflect.DeepEqual(latest.LatestReport.SelectedKPIIDs, []string{"visitors"}) {
		t.Fatal("download lost approved selection")
	}
}

func TestEngagementReportSelectionRejectsEmptyUnknownRepeatedAndOverfullOnePage(t *testing.T) {
	f := newEngagementReportFixture(t)
	draft := engagementReportDecodeSnapshot(t, engagementReportRequest(t, f.router.prepareEngagementReport, http.MethodGet, f.event, nil, f.ctx, nil))
	all := []string{}
	for _, kpi := range draft.KPIs {
		all = append(all, kpi.ID)
	}
	for _, selected := range [][]string{{}, {"unknown"}, {"visitors", "visitors"}, all} {
		input := map[string]interface{}{"calculation_id": draft.CalculationID, "format": "one_page", "selected_kpi_ids": selected}
		engagementReportRequireStatus(t, engagementReportRequest(t, f.router.approveEngagementReport, http.MethodPost, f.event, input, f.ctx, nil), http.StatusBadRequest)
	}
	approved := engagementReportDecodeSnapshot(t, engagementReportRequest(t, f.router.approveEngagementReport, http.MethodPost, f.event, map[string]interface{}{"calculation_id": draft.CalculationID, "format": "extended", "selected_kpi_ids": all}, f.ctx, nil))
	if len(approved.SelectedKPIIDs) != 30 {
		t.Fatal("three-page report cannot select all 30 KPIs")
	}
}

func TestEngagementReportSelectionSupportsPreviousTwentyKPIDrafts(t *testing.T) {
	f := newEngagementReportFixture(t)
	draft := engagementReportDecodeSnapshot(t, engagementReportRequest(t, f.router.prepareEngagementReport, http.MethodGet, f.event, nil, f.ctx, nil))
	draft.KPIs = draft.KPIs[:20]
	draft.SelectedKPIIDs = nil
	encoded, err := json.Marshal(draft)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.conn.Exec(`UPDATE engagement_report_calculations SET payload=? WHERE id=?`, string(encoded), draft.CalculationID); err != nil {
		t.Fatal(err)
	}
	input := map[string]interface{}{"calculation_id": draft.CalculationID, "format": "one_page"}
	approved := engagementReportDecodeSnapshot(t, engagementReportRequest(t, f.router.approveEngagementReport, http.MethodPost, f.event, input, f.ctx, nil))
	if len(approved.KPIs) != 20 || len(approved.SelectedKPIIDs) != 20 {
		t.Fatal("old calculation gained synthetic KPI definitions")
	}
	input["selected_kpi_ids"] = []string{"estimated_total_time"}
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.approveEngagementReport, http.MethodPost, f.event, input, f.ctx, nil), http.StatusBadRequest)
}
