package api

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestEngagementReportExtendedAnalyticsContractAndPrivacy(t *testing.T) {
	f := newEngagementReportFixture(t)
	metadata := map[string]interface{}{
		"section": "mvp", "from_section": "home", "device_type": "mobile", "browser_family": "Safari",
		"placement": "home_grid", "trigger": "mvp", "mode": "live", "outcome": "success", "reason": "network", "error_code": "timeout", "phase": "confirmed", "cta": "mvp",
		"game_id": "tap-volley", "round_id": "round-test-123", "item_key": "match_card", "experience": "good", "perks_interest": "yes", "sport": "Pallavolo",
		"is_change": true, "registered": true, "has_voted": true, "has_ticket": true, "success": true,
		"player_id": float64(7), "previous_player_id": float64(8), "duration_ms": float64(234), "latency_ms": float64(12), "active_seconds": float64(15), "visible_seconds": float64(20), "schema_version": float64(1), "visit_count": float64(2), "status_code": float64(200), "app_version": "2.0.1",
		"started_at": "2026-10-25T10:00:00.000Z", "ended_at": "2026-10-25T10:00:15.000Z",
		"name": "Private name", "phone": "+391234567890", "otp": "123456", "nickname": "Private fan", "text": "Private answer", "url": "https://private.test/?token=secret", "fan_id": float64(99), "fan_session_token": "private-token", "ticket_code": "1234", "qr": "private-qr",
	}
	// Every advertised event is accepted through the same real ingestion path.
	i := 0
	for name := range reportSignalNames {
		if name == "registration_completed" || name == "sponsor_view" || name == "sponsor_click" || name == "sponsored_game_view" {
			continue
		} // Existing verified-profile/sponsor rules remain mandatory.
		i++
		payload := map[string]interface{}{"events": []interface{}{engagementReportSignalPayload(fmt.Sprintf("extended-%d", i), name, metadata)}}
		engagementReportRequireStatus(t, engagementReportRequest(t, f.router.recordReportSignals, http.MethodPost, f.event, payload, f.ctx, map[string]string{"X-Device-ID": "private-device"}), http.StatusNoContent)
	}
	signals, err := f.router.loadReportSignals(f.event)
	if err != nil || len(signals) != i {
		t.Fatalf("stored %d/%d signals: %v", len(signals), i, err)
	}
	for _, signal := range signals {
		for _, key := range []string{"name", "phone", "otp", "nickname", "text", "url", "fan_id", "fan_session_token", "ticket_code", "qr"} {
			if _, found := signal.Metadata[key]; found {
				t.Errorf("private/untrusted property %s stored in %s", key, signal.Name)
			}
		}
		if signal.Metadata["section"] != "mvp" || signal.Metadata["game_id"] != "tap-volley" || signal.Metadata["player_id"] != float64(7) || signal.Metadata["schema_version"] != float64(1) {
			t.Fatalf("valid metadata dropped: %+v", signal)
		}
	}
}

func TestEngagementReportMetadataRejectsInvalidTypedValues(t *testing.T) {
	input := map[string]interface{}{
		"section": "private free text", "game_id": "private-game", "reason": "private stack trace", "placement": "https://private.test", "source": "private source", "registered": "yes",
		"player_id": float64(0), "previous_player_id": 7.5, "sponsor_id": math.Inf(1), "seconds": float64(16), "duration_ms": float64(86400001), "active_seconds": math.NaN(), "coins": float64(-1), "status_code": 200.5,
		"schema_version": float64(2), "round_id": "private / payload", "app_version": "private text", "started_at": "private time", "ended_at": "2026-13-99T10:00:00Z",
	}
	if got := cleanReportMetadata(input); len(got) != 0 {
		t.Fatalf("invalid values survived: %+v", got)
	}
}

func TestEngagementReportDiagnosticsDoNotAlterExistingKPIs(t *testing.T) {
	base := []reportSignal{
		reportTestSignal("view", "page_view", "guest-a", 0, "10", map[string]interface{}{"vote_available": true, "games_available": true, "source": "arena"}),
		reportTestSignal("vote", "vote_completed", "guest-a", 0, "12", nil),
	}
	prior := []reportSignal{reportTestSignal("old-view", "page_view", "guest-b", 0, "05", nil)}
	diagnostics := []reportSignal{
		reportTestSignal("load", "home_loading_started", "unready-browser", 0, "01", nil),
		reportTestSignal("failure", "home_load_failed", "unready-browser", 0, "30", nil),
		// Must not change guest identity resolution or inferred duration either.
		reportTestSignal("profile", "profile_loaded", "guest-a", 99, "50", map[string]interface{}{"registered": true}),
		reportTestSignal("return", "visit_returned", "guest-b", 0, "45", map[string]interface{}{"visit_count": float64(2)}),
	}
	for _, operational := range []reportOperationalData{{}, {Signals: []reportSignal{reportTestSignal("raw-vote", "vote_completed", "operative", 9, "14", map[string]interface{}{"_operational": true})}}} {
		facts, details, started := calculateCombinedReportMetrics(base, prior, operational, reportOperationalData{})
		kpis := buildCombinedReportKPIs(facts, details, base, prior, operational, reportOperationalData{})
		withDiagnostics := append(append([]reportSignal{}, base...), diagnostics...)
		priorDiagnostics := append(append([]reportSignal{}, prior...), diagnostics...)
		gotFacts, gotDetails, gotStarted := calculateCombinedReportMetrics(withDiagnostics, priorDiagnostics, operational, reportOperationalData{})
		gotKPIs := buildCombinedReportKPIs(gotFacts, gotDetails, withDiagnostics, priorDiagnostics, operational, reportOperationalData{})
		if !reflect.DeepEqual(facts, gotFacts) || !reflect.DeepEqual(details, gotDetails) || started != gotStarted || !reflect.DeepEqual(kpis, gotKPIs) {
			want, _ := json.Marshal(struct {
				Facts   map[string]*float64
				Details reportDetails
				KPIs    []reportKPI
				Started string
			}{facts, details, kpis, started})
			got, _ := json.Marshal(struct {
				Facts   map[string]*float64
				Details reportDetails
				KPIs    []reportKPI
				Started string
			}{gotFacts, gotDetails, gotKPIs, gotStarted})
			t.Fatalf("diagnostics changed report\nwant %s\ngot %s", want, got)
		}
	}
	facts, _, started := calculateReportMetrics(diagnostics, diagnostics)
	for key, value := range facts {
		if value != nil {
			t.Errorf("diagnostics fabricated %s=%v", key, *value)
		}
	}
	if strings.TrimSpace(started) != "" {
		t.Fatalf("diagnostic-only report started at %s", started)
	}
}
