package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Representative, synthetic inputs for cost measurements; no database or API key.
func recipientCostSnapshot(profile reportRecipientProfile, scenario string, compare bool) engagementReportSnapshot {
	facts := map[string]*float64{}
	for _, key := range engagementReportFactKeys {
		facts[key] = reportMetricFloat(100)
	}
	facts["visitors"], facts["attendees"] = reportMetricFloat(1200), reportMetricFloat(4000)
	s := engagementReportSnapshot{
		Recipient: &profile, ComparePrevious: compare,
		Event: reportEvent{ID: 2, Title: "Squadra A - Squadra B", StartDateTime: "2026-11-05T18:00:00Z", Location: "Arena", OrganizationName: "Società dimostrativa"},
		KPIs:  buildReportKPIs(facts),
	}
	if profile.ID == "sponsor" {
		s.SponsorID, s.SponsorName = 7, "Partner dimostrativo"
	}
	for i := range s.KPIs {
		k := &s.KPIs[i]
		k.Availability = recipientKPIAvailability(*k, facts, nil, 1, s.Event)
		if scenario == "partial" && i%2 == 0 {
			k.Value, k.Numerator, k.Denominator = nil, nil, nil
			k.Source = "unavailable"
			k.Availability = recipientKPIAvailability(*k, nil, nil, 1, s.Event)
		}
		if scenario == "manual" {
			k.Source = "manual_unverified"
			k.Provenance = []reportManualSource{{SourceType: "manual_kpi", VerificationStatus: "unverified", Note: "PRIVATE-NOTE", SourceDescription: "PRIVATE-SOURCE", EnteredBy: 987654}}
			k.Note = "Valore KPI completato manualmente, non ricostruito dal tracciamento. Motivo originario: Dati automatici insufficienti."
			k.Numerator, k.Denominator = nil, nil
		}
		if compare {
			previous := *k
			previous.Value = reportMetricFloat(75)
			k.Comparison = compareRecipientKPI(*k, previous, true)
		}
	}
	if compare {
		s.PreviousEvent = &reportEvent{ID: 1, Title: "Squadra A - Squadra C", StartDateTime: "2026-10-29T18:00:00Z", Location: "Arena", OrganizationName: s.Event.OrganizationName}
	}
	s.Details.CoverageNotes = []string{
		"Le attività operative e i segnali /new sono uniti per profilo o dispositivo nell'evento. Le tabelle condivise non conservano sempre la pagina di origine: le attività storiche sono attribuite all'evento, non esclusivamente a /new.",
		"I partecipanti osservabili sono un minimo verificabile: le attività salvate non ricostruiscono le visite passive. Le percentuali descrivono solo il pubblico misurato; senza il relativo denominatore restano N/D.",
	}
	return s
}

func TestRecipientAIRequestBudgets(t *testing.T) {
	for _, profile := range reportRecipientProfiles() {
		for _, scenario := range []string{"measured", "partial", "manual"} {
			for _, compare := range []bool{false, true} {
				name := fmt.Sprintf("%s-%s-compare-%t", profile.ID, scenario, compare)
				t.Run(name, func(t *testing.T) {
					s := recipientCostSnapshot(profile, scenario, compare)
					client := &http.Client{Transport: recipientTestTransport(func(r *http.Request) (*http.Response, error) {
						raw, err := io.ReadAll(r.Body)
						if err != nil {
							t.Fatal(err)
						}
						if len(raw) > recipientAIRequestMaxBytes {
							t.Fatalf("request exceeds cost budget: %d bytes", len(raw))
						}
						if strings.Contains(string(raw), "PRIVATE") {
							t.Fatal("private provenance in provider request")
						}
						// Optional local export contains only the synthetic fixture.
						if dir := os.Getenv("REPORT_AI_TEST_EXPORT_DIR"); dir != "" {
							if err := os.MkdirAll(dir, 0700); err != nil {
								t.Fatal(err)
							}
							if err := os.WriteFile(filepath.Join(dir, name+".json"), raw, 0600); err != nil {
								t.Fatal(err)
							}
						}
						t.Logf("complete request: %d bytes", len(raw))
						provided, _ := json.Marshal(recipientAIProviderExample())
						if scenario == "partial" {
							for _, k := range recipientSelectedKPIs(s) {
								if k.Value != nil {
									provided = []byte(strings.ReplaceAll(string(provided), "visitors", k.ID))
									break
								}
							}
						}
						envelope, _ := json.Marshal(map[string]interface{}{"status": "completed", "output": []interface{}{map[string]interface{}{"type": "message", "content": []interface{}{map[string]interface{}{"type": "output_text", "text": string(provided)}}}}})
						return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(envelope))), Header: make(http.Header)}, nil
					})}
					if _, err := callRecipientOpenAI(context.Background(), client, recipientOpenAIURL, "test-key", "gpt-4.1-mini", s); err != nil {
						t.Fatal(err)
					}
				})
			}
		}
	}
}

func TestRecipientAIInputPreservesEvidenceAndExcludesUI(t *testing.T) {
	p, _ := findReportProfile("analytics")
	p.AIInstructions = "PROFILE-INSTRUCTION"
	p.Name, p.Description, p.Layout = "UI-ONLY", "UI-ONLY", "UI-ONLY"
	s := recipientCostSnapshot(p, "measured", true)
	s.KPIs[0].Value = reportMetricFloat(0)
	s.KPIs[1].Value = reportMetricFloat(42.125)
	s.KPIs[1].Numerator, s.KPIs[1].Denominator = nil, nil
	s.KPIs[1].Source = "manual_unverified"
	s.KPIs[1].Provenance = []reportManualSource{{SourceType: "manual_kpi", VerificationStatus: "unverified", SourceDescription: "PRIVATE-SOURCE", Note: "PRIVATE-NOTE"}}
	s.KPIs[1].Note = "Misura manuale senza ricostruzione del tracciamento."
	s.KPIs[2].Value = nil
	s.KPIs[2].Availability = &reportAvailability{Reason: "tracking_missing", Message: "UI-ONLY", Fields: []reportSourceField{{Label: "UI-ONLY", Validation: "UI-ONLY"}}}
	s.KPIs[2].Note = "Il pubblico eleggibile non è ricostruibile dalle sole azioni salvate."
	s.KPIs[3].Note = "Nota condivisa."
	s.KPIs[4].Note = "Nota condivisa."
	s.KPIs[3].Comparison.PreviousProvenance = []reportManualSource{{SourceType: "manual_kpi", VerificationStatus: "verified", SourceDescription: "PRIVATE-PREVIOUS"}}
	s.Details.CoverageNotes = append(s.Details.CoverageNotes, s.Details.CoverageNotes[0], "Tempo stimato: NOT-SELECTED", "I feedback salvati sono anonimi.")
	input := recipientAIInput(s)
	metrics := map[string]recipientAIMetric{}
	for _, metric := range input.KPIs {
		metrics[metric.ID] = metric
	}
	if m := metrics["visitors"]; m.Value != 0 || m.Definition != s.KPIs[0].Description {
		t.Fatal("zero or exact definition lost", m)
	}
	if m := metrics["venue_adoption"]; m.Value != 42.125 || m.Numerator != nil || m.Denominator != nil || len(m.ManualSources) != 1 || m.ManualSources[0].Verification != "unverified" || m.ManualSources[0].Type != "manual_kpi" {
		t.Fatal("direct manual measurement changed", m)
	}
	if _, exists := metrics["active_participation"]; exists || len(input.Unavailable) != 1 || input.Unavailable[0].ID != "active_participation" || input.Unavailable[0].Reason != "tracking_missing" {
		t.Fatal("N/D treated as an available metric", input.Unavailable)
	}
	if m := metrics["active_time"]; m.Denominator == nil || *m.Denominator != *s.KPIs[3].Denominator || m.Numerator == nil || *m.Numerator != *s.KPIs[3].Numerator || m.Comparison == nil || m.Comparison.Previous == nil || *m.Comparison.Previous != 75 || m.Comparison.ManualSources[0].Verification != "verified" {
		t.Fatal("denominators, comparison or prior provenance lost", m)
	}
	shared, global := 0, 0
	for _, limit := range input.Limits {
		if limit.Text == "Nota condivisa." && len(limit.MetricIDs) == 2 {
			shared++
		}
		if limit.Text == s.Details.CoverageNotes[0] {
			global++
		}
	}
	if shared != 1 || global != 1 {
		t.Fatal("repeated limitations were not deduplicated")
	}
	raw, err := recipientAIRequestPayload(s, "gpt-4.1-mini", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"UI-ONLY", "PRIVATE", "NOT-SELECTED", "shop_redemptions", "manual_allowed", "allow_appendix"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatal("unnecessary context leaked", forbidden)
		}
	}
	if strings.Count(string(raw), "PROFILE-INSTRUCTION") != 1 {
		t.Fatal("profile instructions duplicated")
	}
	s.ComparePrevious = false
	input = recipientAIInput(s)
	if input.PreviousEvent != nil {
		t.Fatal("disabled previous event sent")
	}
	for _, metric := range input.KPIs {
		if metric.Comparison != nil {
			t.Fatal("disabled comparison sent")
		}
	}
}

func TestRecipientAIRejectsExcessiveOrInvalidInputBeforeHTTP(t *testing.T) {
	p, _ := findReportProfile("analytics")
	for _, invalid := range []string{"oversized", "invalid-number"} {
		t.Run(invalid, func(t *testing.T) {
			s := recipientCostSnapshot(p, "measured", true)
			reason := "input_too_large"
			if invalid == "oversized" {
				s.KPIs[0].Description = strings.Repeat("evidenza essenziale ", recipientAIRequestMaxBytes)
			} else {
				s.KPIs[0].Value = reportMetricFloat(math.NaN())
				reason = "invalid_input"
			}
			calls := 0
			client := &http.Client{Transport: recipientTestTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				return nil, fmt.Errorf("HTTP must not be called")
			})}
			usage := &recipientAIUsage{}
			_, err := callRecipientOpenAI(context.Background(), client, recipientOpenAIURL, "test-key", "gpt-4.1-mini", s, recipientAIAttemptOptions{Usage: usage})
			if err == nil || recipientErrorDiagnostic(err).Reason != reason || calls != 0 || usage.Requests != 0 {
				t.Fatal("invalid input reached provider", calls, usage, err)
			}
		})
	}
}

func TestRecipientAIContextLengthsAndSponsorScope(t *testing.T) {
	p, _ := findReportProfile("sponsor")
	s := recipientCostSnapshot(p, "manual", true)
	longName := strings.Repeat("À", 100000)
	s.Event.Title, s.Event.OrganizationName, s.SponsorName = longName, longName, longName
	s.PreviousEvent.Title = longName
	input := recipientAIInput(s)
	if len([]rune(input.Organization)) != 120 || len([]rune(input.Event.Title)) != 160 || len([]rune(input.SponsorName)) != 120 || len([]rune(input.PreviousEvent.Title)) != 160 || input.SponsorScope != "selected_partner" {
		t.Fatal("unbounded decorative context or missing sponsor scope")
	}
	if _, err := recipientAIRequestPayload(s, "gpt-4.1-mini", ""); err != nil {
		t.Fatal("long names should not consume KPI budget", err)
	}
	if s.Event.Title != longName {
		t.Fatal("PDF snapshot modified")
	}
	s.SponsorID = 0
	input = recipientAIInput(s)
	if input.SponsorScope != "event_total" || input.SponsorName != "" {
		t.Fatal("event totals attributed to a dedicated sponsor")
	}
}

func TestRecipientAIPermanentFailuresDoNotRetry(t *testing.T) {
	for _, failure := range []string{"unauthorized", "oversized"} {
		t.Run(failure, func(t *testing.T) {
			f := newEngagementReportFixture(t)
			recipientPageSignal(t, f)
			t.Setenv("OPENAI_API_KEY", "test-key")
			t.Setenv("OPENAI_MODEL", "test-model")
			s := recipientDraft(t, f, "executive", 0)
			if failure == "oversized" {
				s.KPIs[0].Description = strings.Repeat("evidenza ", recipientAIRequestMaxBytes)
				raw, _ := json.Marshal(s)
				if _, err := f.conn.Exec(`UPDATE engagement_report_calculations SET payload=? WHERE id=?`, string(raw), s.CalculationID); err != nil {
					t.Fatal(err)
				}
			}
			calls := 0
			old := http.DefaultTransport
			t.Cleanup(func() { http.DefaultTransport = old })
			http.DefaultTransport = recipientTestTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: 401, Body: io.NopCloser(strings.NewReader(`{"error":{"code":"invalid_api_key"}}`)), Header: make(http.Header)}, nil
			})
			for i := 0; i < 2; i++ {
				r := engagementReportRequest(t, f.router.generateRecipientReportAnalysis, "POST", f.event, map[string]interface{}{"calculation_id": s.CalculationID, "recipient": "executive"}, f.ctx, nil)
				engagementReportRequireStatus(t, r, 200)
				var response struct {
					Status     string                 `json:"status"`
					Cached     bool                   `json:"cached"`
					Usage      *recipientAIUsage      `json:"usage"`
					Diagnostic *recipientAIDiagnostic `json:"diagnostic"`
				}
				if err := json.Unmarshal(r.Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				wantCalls, wantReason := 1, "provider_http_error"
				if failure == "oversized" {
					wantCalls, wantReason = 0, "input_too_large"
				}
				if calls != wantCalls || response.Status != "fallback" || response.Cached != (i == 1) || response.Usage == nil || response.Usage.Requests != wantCalls || response.Usage.ReportedRequests != 0 || response.Diagnostic == nil || response.Diagnostic.Reason != wantReason {
					t.Fatal("unnecessary retry or incorrect usage", calls, r.Body.String())
				}
			}
		})
	}
}
