package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
)

func recipientDraft(t *testing.T, f engagementReportFixture, profile string, sponsor int) engagementReportSnapshot {
	t.Helper()
	h := func(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
		r.URL.RawQuery = fmt.Sprintf("recipient=%s&sponsor_id=%d", profile, sponsor)
		f.router.prepareEngagementReport(w, r, ctx)
	}
	r := engagementReportRequest(t, h, "GET", f.event, nil, f.ctx, nil)
	engagementReportRequireStatus(t, r, 200)
	var s engagementReportSnapshot
	if err := json.Unmarshal(r.Body.Bytes(), &s); err != nil {
		t.Fatal(err)
	}
	return s
}
func recipientTestKPI(t *testing.T, s engagementReportSnapshot, id string) reportKPI {
	t.Helper()
	for _, k := range s.KPIs {
		if k.ID == id {
			return k
		}
	}
	t.Fatalf("missing %s", id)
	return reportKPI{}
}
func recipientPageSignal(t *testing.T, f engagementReportFixture) {
	t.Helper()
	m := map[string]interface{}{"source": "arena", "vote_available": true, "games_available": true, "feedback_available": true}
	raw, _ := json.Marshal(m)
	_, err := f.conn.Exec(`INSERT INTO engagement_report_signals(event_id,signal_id,name,session_id,device_key,fan_id,occurred_at,metadata) VALUES(?,?,?,?,?,?,?,?)`, f.event, "visit-reports", "page_view", "session-reports", "device-reports", 0, "2026-10-21T19:00:00Z", string(raw))
	if err != nil {
		t.Fatal(err)
	}
}
func recipientSourcePayload(value float64, revision int, verified bool) interface{} {
	status := "unverified"
	if verified {
		status = "verified"
	}
	return map[string]interface{}{"sources": []reportManualSource{{FieldKey: "attendees", Value: value, Unit: "people", SourceType: "official_attendance", SourceDescription: "Distinta ufficiale biglietteria", VerificationStatus: status, Revision: revision}}}
}

func TestRecipientProfilesUseOriginalCatalog(t *testing.T) {
	counts := map[string]int{"executive": 5, "marketing": 10, "sponsor": 5, "analytics": 20}
	known := map[string]bool{}
	for _, k := range buildReportKPIs(nil) {
		known[k.ID] = true
	}
	for _, p := range reportRecipientProfiles() {
		if len(p.KPIIDs) != counts[p.ID] {
			t.Fatal(p.ID, len(p.KPIIDs))
		}
		seen := map[string]bool{}
		for _, id := range p.KPIIDs {
			if !known[id] || seen[id] {
				t.Fatal(id)
			}
			seen[id] = true
		}
		if p.Objective == "" || p.AIInstructions == "" || p.Layout == "" {
			t.Fatal("incomplete profile")
		}
	}
}

func TestReportsUseMeasuredSponsorDataAndMatchClicksToTheSameSponsor(t *testing.T) {
	f := newEngagementReportFixture(t)
	sponsor := func(name string, position int) int {
		id, err := f.router.db.CreateSponsor(database.Sponsor{OrganizationID: f.org.ID, Name: name, Position: position, LogoData: "logo", IsActive: true})
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	a, b := sponsor("Partner A", 1), sponsor("Partner B", 2)
	insert := func(id string, event, org, sponsorID int, audience string, viewable, clicks int) {
		t.Helper()
		_, err := f.conn.Exec(`INSERT INTO sponsor_measured_exposures
			(id,club_id,match_id,sponsor_id,sponsor_name,creative_id,placement_id,game_id,method,session_id,audience_key,
			 start_ms,last_ms,max_percentage,impression,viewable,full_duration,clickable,clicks)
			VALUES (?,?,?,?,'Partner','creative','post_vote','','html_display','session',?,1,1001,100,?,?,0,1,?)`,
			id, org, event, sponsorID, audience, viewable, viewable, clicks)
		if err != nil {
			t.Fatal(err)
		}
	}
	insert("a1", f.event, f.org.ID, a, "one", 1, 2)
	insert("a2", f.event, f.org.ID, a, "one", 1, 0)
	insert("a3", f.event, f.org.ID, a, "two", 1, 0)
	insert("b1", f.event, f.org.ID, b, "one", 1, 1)
	insert("b2", f.event, f.org.ID, b, "two", 0, 1)
	insert("foreign", f.event, f.other.ID, a, "foreign", 1, 1)
	legacy := engagementReportDecodeSnapshot(t, engagementReportRequest(t, f.router.prepareEngagementReport, http.MethodGet, f.event, nil, f.ctx, nil))
	for _, snapshot := range []engagementReportSnapshot{legacy, recipientDraft(t, f, "sponsor", 0)} {
		reportTestFact(t, snapshot.Facts, "sponsor_impressions", 4)
		reportTestFact(t, snapshot.Facts, "sponsor_reach", 2)
		reportTestFact(t, snapshot.Facts, "sponsor_clickers", 1)
		reportTestFact(t, snapshot.Facts, "total_sponsor_clicks", 4)
		k := recipientTestKPI(t, snapshot, "sponsor_ctr")
		if k.Value == nil || *k.Value != 50 || k.Source != "operational" {
			t.Fatalf("wrong CTR for exposed audience: %+v", k)
		}
		if len(snapshot.Details.Sponsors) != 2 || snapshot.Details.Sponsors[0].Impressions != 3 {
			t.Fatalf("missing measured partner breakdown: %+v", snapshot.Details.Sponsors)
		}
	}
	dedicated := recipientDraft(t, f, "sponsor", b)
	reportTestFact(t, dedicated.Facts, "sponsor_reach", 1)
	reportTestFact(t, dedicated.Facts, "sponsor_clickers", 1)
	reportTestFact(t, dedicated.Facts, "total_sponsor_clicks", 2)
	insert("anonymous", f.event, f.org.ID, b, "", 1, 0)
	dedicated = recipientDraft(t, f, "sponsor", b)
	reportTestFact(t, dedicated.Facts, "sponsor_impressions", 2)
	if dedicated.Facts["sponsor_reach"] != nil || recipientTestKPI(t, dedicated, "sponsor_ctr").Value != nil {
		t.Fatal("anonymous exposure cannot establish reach or CTR")
	}
}

func TestRecipientManualAttendancePersistsRecalculatesAndAudits(t *testing.T) {
	f := newEngagementReportFixture(t)
	recipientPageSignal(t, f)
	before := recipientDraft(t, f, "executive", 0)
	k := recipientTestKPI(t, before, "venue_adoption")
	if k.Value != nil || !k.Availability.ManualAllowed || k.Availability.Reason != "manual_source_missing" {
		t.Fatalf("%+v", k)
	}
	saved := engagementReportRequest(t, f.router.saveRecipientReportSources, "PUT", f.event, recipientSourcePayload(100, 0, true), f.ctx, nil)
	engagementReportRequireStatus(t, saved, 200)
	after := recipientDraft(t, f, "marketing", 0)
	k = recipientTestKPI(t, after, "venue_adoption")
	if k.Value == nil || *k.Value != 1 || k.Source != "manual_verified" || len(k.Provenance) != 1 || k.Provenance[0].EnteredBy != f.ctx.AdminID {
		t.Fatalf("bad adoption %+v", k)
	}
	if before.DataHash == after.DataHash {
		t.Fatal("manual edit did not invalidate hash")
	}
	executive := recipientDraft(t, f, "executive", 0)
	if *recipientTestKPI(t, executive, "venue_adoption").Value != 1 {
		t.Fatal("source lost between profiles")
	}
	custom := engagementReportDecodeSnapshot(t, engagementReportRequest(t, f.router.prepareEngagementReport, http.MethodGet, f.event, nil, f.ctx, nil))
	if k := recipientTestKPI(t, custom, "venue_adoption"); k.Value == nil || *k.Value != 1 || k.Source != "manual_verified" {
		t.Fatalf("custom report did not use the official attendance source: %+v", k)
	}
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.saveRecipientReportSources, "PUT", f.event, recipientSourcePayload(200, 0, true), f.ctx, nil), 409)
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.saveRecipientReportSources, "PUT", f.event, recipientSourcePayload(200, 1, false), f.ctx, nil), 200)
	history := engagementReportRequest(t, f.router.getRecipientReportSources, "GET", f.event, nil, f.ctx, nil)
	engagementReportRequireStatus(t, history, 200)
	var h struct {
		History []json.RawMessage `json:"history"`
	}
	_ = json.Unmarshal(history.Body.Bytes(), &h)
	if len(h.History) != 2 {
		t.Fatal(history.Body.String())
	}
	updated := recipientDraft(t, f, "executive", 0)
	if recipientTestKPI(t, updated, "venue_adoption").Source != "manual_unverified" {
		t.Fatal("verification lost")
	}
}

func TestRecipientSourcesRejectDerivedDataInvalidInputAndUnauthorized(t *testing.T) {
	f := newEngagementReportFixture(t)
	for _, role := range []string{"staff", "partner", "bar", ""} {
		ctx := f.ctx
		ctx.AdminRole = role
		engagementReportRequireStatus(t, engagementReportRequest(t, f.router.saveRecipientReportSources, "PUT", f.event, recipientSourcePayload(100, 0, true), ctx, nil), 403)
	}
	ctx := f.ctx
	ctx.OrganizationID = f.other.ID
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.saveRecipientReportSources, "PUT", f.event, recipientSourcePayload(100, 0, true), ctx, nil), 404)
	for _, m := range []reportManualSource{
		{FieldKey: "sponsored_participation", Value: 12, Unit: "percent", SourceType: "official_attendance", SourceDescription: "invented", VerificationStatus: "verified"},
		{FieldKey: "attendees", Value: -1, Unit: "people", SourceType: "official_attendance", SourceDescription: "official", VerificationStatus: "verified"},
		{FieldKey: "attendees", Value: 1.5, Unit: "people", SourceType: "official_attendance", SourceDescription: "official", VerificationStatus: "verified"},
		{FieldKey: "attendees", Value: 100, Unit: "people", SourceType: "official_attendance", SourceDescription: "", VerificationStatus: "verified"},
	} {
		r := engagementReportRequest(t, f.router.saveRecipientReportSources, "PUT", f.event, map[string]interface{}{"sources": []reportManualSource{m}}, f.ctx, nil)
		engagementReportRequireStatus(t, r, 400)
	}
	s := recipientDraft(t, f, "executive", 0)
	if recipientTestKPI(t, s, "sponsored_participation").Availability.ManualAllowed {
		t.Fatal("missing KPI must remain unavailable without database evidence")
	}
}

func TestRecipientAvailabilityAndAutomaticConflict(t *testing.T) {
	facts := map[string]*float64{"attendees": reportMetricFloat(200)}
	conflicts := resolveReportManualSources(facts, []reportManualSource{{FieldKey: "attendees", Value: 100, VerificationStatus: "verified"}})
	if len(conflicts) != 1 || *facts["attendees"] != 200 {
		t.Fatal("automatic evidence overwritten")
	}
	facts = map[string]*float64{"onsite_visitors": reportMetricFloat(0), "attendees": reportMetricFloat(0)}
	k := buildReportKPIs(facts)[1]
	if a := recipientKPIAvailability(k, facts, nil, 0, reportEvent{}); a.Reason != "zero_denominator" || !a.ManualAllowed {
		t.Fatalf("%+v", a)
	}
	k.Numerator = reportMetricFloat(200)
	k.Denominator = reportMetricFloat(100)
	if a := recipientKPIAvailability(k, facts, nil, 0, reportEvent{}); a.Reason != "incompatible_population" {
		t.Fatal(a)
	}
	for _, tc := range []struct {
		k       reportKPI
		signals []reportSignal
		event   reportEvent
		reason  string
	}{
		{reportKPI{ID: "event_retention"}, nil, reportEvent{}, "not_applicable"},
		{reportKPI{ID: "mvp_participation"}, []reportSignal{{Metadata: map[string]interface{}{"vote_available": false}}}, reportEvent{}, "feature_unavailable"},
		{reportKPI{ID: "visitors"}, nil, reportEvent{StartDateTime: time.Now().Add(time.Hour).Format(time.RFC3339)}, "not_synchronized"},
		{reportKPI{ID: "sponsored_participation"}, nil, reportEvent{}, "tracking_missing"},
	} {
		if got := recipientKPIAvailability(tc.k, nil, tc.signals, 0, tc.event); got.Reason != tc.reason {
			t.Fatalf("%s: %+v", tc.reason, got)
		}
	}
}

func TestRecipientComparisonUsesConcludedSameOrganization(t *testing.T) {
	f := newEngagementReportFixture(t)
	recipientPageSignal(t, f)
	create := func(org int, date string, concluded bool) int {
		id, err := f.router.db.CreateEvent(database.Event{OrganizationID: org, Team1ID: f.org.TeamID, Team2ID: f.other.TeamID, StartDateTime: date})
		if err != nil {
			t.Fatal(err)
		}
		if concluded {
			_, err = f.conn.Exec(`UPDATE events SET is_concluded=1 WHERE id=?`, id)
			if err != nil {
				t.Fatal(err)
			}
		}
		return id
	}
	prev := create(f.org.ID, "2026-10-10T10:00:00Z", true)
	create(f.org.ID, "2026-10-20T10:00:00Z", false)
	create(f.other.ID, "2026-10-20T10:00:00Z", true)
	s := recipientDraft(t, f, "marketing", 0)
	if s.PreviousEvent == nil || s.PreviousEvent.ID != prev {
		t.Fatal("wrong preceding event", s.PreviousEvent)
	}
	current := reportKPI{Unit: "count", Value: reportMetricFloat(15)}
	previous := reportKPI{Unit: "count", Value: reportMetricFloat(10)}
	c := compareRecipientKPI(current, previous, true)
	if *c.Absolute != 5 || *c.Percent != 50 || c.Direction != "up" {
		t.Fatal(c)
	}
	previous.Value = reportMetricFloat(0)
	c = compareRecipientKPI(current, previous, true)
	if c.Percent != nil || c.Absolute == nil {
		t.Fatal("zero baseline", c)
	}
	current.Unit = "percent"
	previous.Unit = "percent"
	previous.Value = reportMetricFloat(10)
	c = compareRecipientKPI(current, previous, true)
	if *c.PercentagePoints != 5 || c.Percent != nil {
		t.Fatal(c)
	}
	previous.Description = "different population"
	if compareRecipientKPI(current, previous, true).Absolute != nil {
		t.Fatal("inhomogeneous comparison")
	}
}

func TestRecipientSponsorNeverUsesOtherPartnerExposure(t *testing.T) {
	f := newEngagementReportFixture(t)
	recipientPageSignal(t, f)
	a, err := f.router.db.CreateSponsor(database.Sponsor{OrganizationID: f.org.ID, Position: 1, Name: "Partner A", LogoData: "logo", LinkURL: "https://example.com", IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	b, err := f.router.db.CreateSponsor(database.Sponsor{OrganizationID: f.org.ID, Position: 2, Name: "Partner B", LogoData: "logo", LinkURL: "https://example.com", IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	for i, name := range []string{"sponsor_view", "sponsor_click"} {
		raw := fmt.Sprintf(`{"sponsor_id":%d,"name":"Partner A","placement":"home_partner"}`, a)
		_, err = f.conn.Exec(`INSERT INTO engagement_report_signals(event_id,signal_id,name,session_id,device_key,occurred_at,metadata) VALUES(?,?,?,?,?,?,?)`, f.event, fmt.Sprint("sponsor", i), name, "session-reports", "device-reports", "2026-10-21T19:01:00Z", raw)
		if err != nil {
			t.Fatal(err)
		}
	}
	s := recipientDraft(t, f, "sponsor", a)
	if k := recipientTestKPI(t, s, "sponsor_reach"); k.Value == nil || *k.Value != 1 {
		t.Fatalf("%+v", k)
	}
	s = recipientDraft(t, f, "sponsor", b)
	for _, id := range []string{"sponsor_reach", "sponsor_impressions", "sponsor_ctr", "sponsored_participation"} {
		if recipientTestKPI(t, s, id).Value != nil {
			t.Fatal("attributed other sponsor", id)
		}
	}
	if *recipientTestKPI(t, s, "visitors").Value != 1 {
		t.Fatal("event audience context lost")
	}
}

func recipientAIExample() recipientAnalysis {
	return recipientAnalysis{Headline: "Partecipazione osservata", Summary: "Le fonti descrivono il pubblico digitale misurabile.", PositiveInsights: []recipientInsight{{Title: "Pubblico misurabile", Description: "La partecipazione offre una base osservabile per la società.", MetricIDs: []string{"visitors"}}}, CriticalInsights: []recipientInsight{}, Recommendations: []recipientRecommendation{{Action: "Consolidare la raccolta", Reason: "La continuità aiuta a leggere gli eventi.", MetricIDs: []string{"visitors"}, Priority: "medium"}}, Limitations: []string{"La copertura osservata può essere incompleta."}}
}

func recipientAIProviderExample() recipientProviderAnalysis {
	a := recipientAIExample()
	return recipientProviderAnalysis{Headline: a.Headline, Summary: a.Summary, Insights: []recipientProviderInsight{{recipientInsight: a.PositiveInsights[0], Kind: "positive"}}, Recommendations: a.Recommendations, Limitations: a.Limitations}
}

func TestRecipientSharedInsightBudgetAndPublicContract(t *testing.T) {
	for _, profile := range reportRecipientProfiles() {
		for positive := 0; positive <= profile.MaxInsights+1; positive++ {
			for critical := 0; critical <= profile.MaxInsights+1; critical++ {
				t.Run(fmt.Sprintf("%s/positive-%d/critical-%d", profile.ID, positive, critical), func(t *testing.T) {
					s := engagementReportSnapshot{Recipient: &profile, KPIs: []reportKPI{{ID: "visitors", Value: reportMetricFloat(5)}}}
					provided := recipientAIProviderExample()
					provided.Insights = []recipientProviderInsight{}
					insight := recipientAIExample().PositiveInsights[0]
					for i := 0; i < positive; i++ {
						provided.Insights = append(provided.Insights, recipientProviderInsight{recipientInsight: insight, Kind: "positive"})
					}
					for i := 0; i < critical; i++ {
						provided.Insights = append(provided.Insights, recipientProviderInsight{recipientInsight: insight, Kind: "critical"})
					}
					raw, _ := json.Marshal(provided)
					a, err := validateRecipientProviderAnalysis(raw, s)
					if positive+critical > profile.MaxInsights {
						if err == nil {
							t.Fatal("combined budget exceeded without rejection")
						}
						return
					}
					if err != nil {
						t.Fatal(err)
					}
					if len(a.PositiveInsights) != positive || len(a.CriticalInsights) != critical {
						t.Fatal("insights lost or misclassified")
					}
					public, _ := json.Marshal(a)
					var fields map[string]interface{}
					_ = json.Unmarshal(public, &fields)
					if _, wire := fields["insights"]; wire {
						t.Fatal("provider contract leaked into public API")
					}
					if _, err := validateRecipientAnalysis(public, s); err != nil {
						t.Fatal("public validation no longer holds", err)
					}
					publicProperties := recipientAnalysisSchema(s)["properties"].(map[string]interface{})
					item := publicProperties["positive_insights"].(map[string]interface{})["items"].(map[string]interface{})["properties"].(map[string]interface{})
					if _, leaked := item["kind"]; leaked {
						t.Fatal("provider schema mutated public schema")
					}
				})
			}
		}
	}
}

func TestRecipientProviderInsightValidationAndConfiguredLimit(t *testing.T) {
	p, _ := findReportProfile("executive")
	p.MaxInsights = 1
	s := engagementReportSnapshot{Recipient: &p, KPIs: []reportKPI{{ID: "visitors", Value: reportMetricFloat(5)}}}
	valid, _ := json.Marshal(recipientAIProviderExample())
	if _, err := validateRecipientProviderAnalysis(valid, s); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{
		strings.Replace(string(valid), `"kind":"positive"`, `"kind":"unknown"`, 1),
		strings.Replace(string(valid), `"kind":"positive"`, `"unexpected":true,"kind":"positive"`, 1),
		strings.Replace(string(valid), `"metric_ids":["visitors"]`, `"metric_ids":["sponsor_reach"]`, 1),
	} {
		if _, err := validateRecipientProviderAnalysis([]byte(raw), s); err == nil {
			t.Fatal("invalid insight accepted", raw)
		}
	}
	provided := recipientAIProviderExample()
	provided.Insights = append(provided.Insights, provided.Insights[0])
	over, _ := json.Marshal(provided)
	if _, err := validateRecipientProviderAnalysis(over, s); err == nil {
		t.Fatal("profile-specific budget ignored")
	}
	// Previously accepted by each category's schema, but rejected by Go only:
	// the public validator must still reject that original three-plus-three case.
	p.MaxInsights = 3
	a := recipientAIExample()
	a.PositiveInsights = []recipientInsight{a.PositiveInsights[0], a.PositiveInsights[0], a.PositiveInsights[0]}
	a.CriticalInsights = append([]recipientInsight{}, a.PositiveInsights...)
	raw, _ := json.Marshal(a)
	if _, err := validateRecipientAnalysis(raw, s); err == nil || err.Error() != "insight limit exceeded" {
		t.Fatal("public combined-budget guard lost", err)
	}
}

func TestRecipientAIValidatesStrictSchemaAndAllowsNumericProseForReview(t *testing.T) {
	p, _ := findReportProfile("executive")
	s := engagementReportSnapshot{Recipient: &p, KPIs: []reportKPI{{ID: "visitors", Value: reportMetricFloat(5)}}}
	valid, _ := json.Marshal(recipientAIExample())
	if _, err := validateRecipientAnalysis(valid, s); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{strings.Replace(string(valid), `"medium"`, `"urgent"`, 1), strings.Replace(string(valid), `"metric_ids":["visitors"]`, `"metric_ids":["sponsor_reach"]`, 1), strings.Replace(string(valid), `"positive_insights":[`, `"unexpected":true,"positive_insights":[`, 1), strings.Replace(string(valid), `"critical_insights":[]`, `"critical_insights":null`, 1), `{"headline":"missing fields"}`} {
		if _, err := validateRecipientAnalysis([]byte(bad), s); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
	for _, headline := range []string{"Crescita 99%", "Crescita cento", "Due opportunità", "Valore € 10"} {
		raw := strings.Replace(string(valid), `"headline":"Partecipazione osservata"`, `"headline":"`+headline+`"`, 1)
		analysis, err := validateRecipientAnalysis([]byte(raw), s)
		if err != nil || len(recipientAnalysisWarnings(analysis)) != 1 {
			t.Fatal("numeric prose must reach human review", headline, err)
		}
	}
	client := &http.Client{Transport: recipientTestTransport(func(r *http.Request) (*http.Response, error) {
		var request map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&request)
		format := request["text"].(map[string]interface{})["format"].(map[string]interface{})
		if format["strict"] != true || format["type"] != "json_schema" || request["store"] != false {
			t.Error("Responses schema missing", request)
		}
		properties := format["schema"].(map[string]interface{})["properties"].(map[string]interface{})
		if _, old := properties["positive_insights"]; old {
			t.Error("provider still has independent insight budgets")
		}
		if properties["insights"].(map[string]interface{})["maxItems"] != float64(p.MaxInsights) {
			t.Error("shared insight budget missing")
		}
		provided, _ := json.Marshal(recipientAIProviderExample())
		raw, _ := json.Marshal(map[string]interface{}{"status": "completed", "output": []interface{}{map[string]interface{}{"type": "message", "content": []interface{}{map[string]interface{}{"type": "output_text", "text": string(provided)}}}}})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(raw))), Header: make(http.Header)}, nil
	})}
	if _, err := callRecipientOpenAI(context.Background(), client, recipientOpenAIURL, "test-key", "test-model", s); err != nil {
		t.Fatal(err)
	}
}

type recipientTestTransport func(*http.Request) (*http.Response, error)

func (f recipientTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRecipientAICacheRetryFallbackInvalidationAndSnapshots(t *testing.T) {
	f := newEngagementReportFixture(t)
	recipientPageSignal(t, f)
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_MODEL", "test-model")
	old := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = old })
	var calls int32
	valid, _ := json.Marshal(recipientAIProviderExample())
	http.DefaultTransport = recipientTestTransport(func(r *http.Request) (*http.Response, error) {
		count := atomic.AddInt32(&calls, 1)
		text := string(valid)
		if count == 1 {
			text = `{"invalid":true}`
		}
		raw, _ := json.Marshal(map[string]interface{}{"status": "completed", "usage": map[string]interface{}{"input_tokens": 1000 + count, "input_tokens_details": map[string]int{"cached_tokens": 250}, "output_tokens": 100 + count, "total_tokens": 1100 + 2*count}, "output": []interface{}{map[string]interface{}{"type": "message", "content": []interface{}{map[string]interface{}{"type": "output_text", "text": text}}}}})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(raw))), Header: make(http.Header)}, nil
	})
	s := recipientDraft(t, f, "executive", 0)
	analysisInput := map[string]interface{}{"calculation_id": s.CalculationID, "recipient": "executive", "compare_previous": true}
	r := engagementReportRequest(t, f.router.generateRecipientReportAnalysis, "POST", f.event, analysisInput, f.ctx, nil)
	engagementReportRequireStatus(t, r, 200)
	if atomic.LoadInt32(&calls) != 2 || !strings.Contains(r.Body.String(), `"status":"ready"`) {
		t.Fatal(calls, r.Body.String())
	}
	var first struct {
		Usage *recipientAIUsage `json:"usage"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &first); err != nil || first.Usage == nil || first.Usage.Requests != 2 || first.Usage.ReportedRequests != 2 || first.Usage.InputTokens != 2003 || first.Usage.OutputTokens != 203 || first.Usage.CachedInputTokens != 500 || first.Usage.TotalTokens != 2206 || first.Usage.RequestBytes > recipientAIRequestMaxBytes {
		t.Fatal("usage must include both attempts, including rejected output", r.Body.String(), err)
	}
	r = engagementReportRequest(t, f.router.generateRecipientReportAnalysis, "POST", f.event, analysisInput, f.ctx, nil)
	engagementReportRequireStatus(t, r, 200)
	if atomic.LoadInt32(&calls) != 2 || !strings.Contains(r.Body.String(), `"cached":true`) {
		t.Fatal("cache missed", calls, r.Body.String())
	}
	var cached struct {
		Usage *recipientAIUsage `json:"usage"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &cached); err != nil || cached.Usage == nil || *cached.Usage != *first.Usage {
		t.Fatal("cached usage changed or triggered a new request", r.Body.String(), err)
	}
	reviewRevision := recipientApproveAnalysis(t, f, s, "executive", true)
	approve := map[string]interface{}{"calculation_id": s.CalculationID, "recipient": "executive", "compare_previous": true, "include_appendix": true, "include_ai": true, "analysis_review_revision": reviewRevision}
	r = engagementReportRequest(t, f.router.approveEngagementReport, "POST", f.event, approve, f.ctx, nil)
	engagementReportRequireStatus(t, r, 200)
	var frozen engagementReportSnapshot
	_ = json.Unmarshal(r.Body.Bytes(), &frozen)
	if frozen.ID == "" || frozen.AnalysisStatus != "ready" || !frozen.IncludeAppendix || len(frozen.SelectedKPIIDs) != 5 {
		t.Fatal(r.Body.String())
	}
	retry := engagementReportRequest(t, f.router.approveEngagementReport, "POST", f.event, approve, f.ctx, nil)
	if retry.Body.String() != r.Body.String() {
		t.Fatal("snapshot not idempotent")
	}
	archive := engagementReportRequest(t, f.router.listRecipientReportSnapshots, "GET", f.event, nil, f.ctx, nil)
	engagementReportRequireStatus(t, archive, 200)
	if !strings.Contains(archive.Body.String(), frozen.ID) {
		t.Fatal("approved version not archived")
	}
	getVersion := func(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
		mux := chi.NewRouter()
		mux.Get("/{eventId}/{reportId}", func(w http.ResponseWriter, r *http.Request) { f.router.getRecipientReportSnapshot(w, r, ctx) })
		r.URL.Path = fmt.Sprintf("/%d/%s", f.event, frozen.ID)
		mux.ServeHTTP(w, r)
	}
	version := engagementReportRequest(t, getVersion, "GET", f.event, nil, f.ctx, nil)
	engagementReportRequireStatus(t, version, 200)
	if version.Body.String() != retry.Body.String() {
		t.Fatal("archive changed frozen data")
	}
	foreign := f.ctx
	foreign.OrganizationID = f.other.ID
	engagementReportRequireStatus(t, engagementReportRequest(t, getVersion, "GET", f.event, nil, foreign, nil), 404)
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.saveRecipientReportSources, "PUT", f.event, recipientSourcePayload(100, 0, true), f.ctx, nil), 200)
	updated := recipientDraft(t, f, "executive", 0)
	if updated.DataHash == s.DataHash {
		t.Fatal("data hash unchanged")
	}
	if recipientTestKPI(t, frozen, "venue_adoption").Value != nil {
		t.Fatal("frozen snapshot modified")
	}
	http.DefaultTransport = recipientTestTransport(func(r *http.Request) (*http.Response, error) {
		atomic.AddInt32(&calls, 1)
		return nil, fmt.Errorf("API unavailable")
	})
	analysisInput["calculation_id"] = updated.CalculationID
	r = engagementReportRequest(t, f.router.generateRecipientReportAnalysis, "POST", f.event, analysisInput, f.ctx, nil)
	engagementReportRequireStatus(t, r, 200)
	if atomic.LoadInt32(&calls) != 3 || !strings.Contains(r.Body.String(), `"status":"fallback"`) {
		t.Fatal("fallback or retry count", calls, r.Body.String())
	}
	for _, profile := range reportRecipientProfiles() {
		approve["recipient"] = profile.ID
		approve["calculation_id"] = updated.CalculationID
		approve["analysis_review_revision"] = recipientApproveAnalysis(t, f, updated, profile.ID, true)
		r = engagementReportRequest(t, f.router.approveEngagementReport, "POST", f.event, approve, f.ctx, nil)
		engagementReportRequireStatus(t, r, 200)
		var v engagementReportSnapshot
		_ = json.Unmarshal(r.Body.Bytes(), &v)
		if v.Analysis == nil || v.Recipient.ID != profile.ID || profile.ID == "analytics" && v.IncludeAppendix {
			t.Fatal("invalid recipient snapshot", r.Body.String())
		}
	}
}

func TestRecipientConcurrentAIRequestsShareReservation(t *testing.T) {
	f := newEngagementReportFixture(t)
	recipientPageSignal(t, f)
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_MODEL", "test-model")
	old := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = old })
	entered, release := make(chan struct{}, 2), make(chan struct{})
	var calls int32
	valid, _ := json.Marshal(recipientAIProviderExample())
	http.DefaultTransport = recipientTestTransport(func(r *http.Request) (*http.Response, error) {
		atomic.AddInt32(&calls, 1)
		entered <- struct{}{}
		select {
		case <-release:
		case <-r.Context().Done():
			return nil, r.Context().Err()
		}
		raw, _ := json.Marshal(map[string]interface{}{"status": "completed", "output": []interface{}{map[string]interface{}{"type": "message", "content": []interface{}{map[string]interface{}{"type": "output_text", "text": string(valid)}}}}})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(raw))), Header: make(http.Header)}, nil
	})
	s := recipientDraft(t, f, "executive", 0)
	input := map[string]interface{}{"calculation_id": s.CalculationID, "recipient": "executive", "compare_previous": true}
	responses := make(chan int, 2)
	request := func() {
		r := engagementReportRequest(t, f.router.generateRecipientReportAnalysis, "POST", f.event, input, f.ctx, nil)
		responses <- r.Code
	}
	go request()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("provider not invoked")
	}
	go request()
	// Let the second reader observe pending before completing the first call.
	time.Sleep(30 * time.Millisecond)
	close(release)
	for i := 0; i < 2; i++ {
		select {
		case code := <-responses:
			if code != 200 {
				t.Fatal(code)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("duplicate request hung")
		}
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Fatal("duplicate provider calls", calls)
	}
}

func TestRecipientAIInputExcludesLogosFromProviderRequests(t *testing.T) {
	logo := "data:image/png;base64," + strings.Repeat("BASE64-LOGO", 64*1024)
	for _, profile := range reportRecipientProfiles() {
		for _, compare := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/compare=%t", profile.ID, compare), func(t *testing.T) {
				s := engagementReportSnapshot{
					Recipient: &profile, ComparePrevious: compare,
					Event: reportEvent{ID: 2, Title: "Partita attuale", StartDateTime: "2026-11-05T18:00:00Z", Location: "Arena", OrganizationName: "Club"},
					KPIs:  []reportKPI{{ID: "visitors", Label: "Partecipanti", Value: reportMetricFloat(5)}},
				}
				if compare {
					s.PreviousEvent = &reportEvent{ID: 1, Title: "Partita precedente", StartDateTime: "2026-10-29T18:00:00Z", Location: "Arena", OrganizationName: "Club"}
				}
				baseline, err := json.Marshal(recipientAIInput(s))
				if err != nil {
					t.Fatal(err)
				}
				s.Event.OrganizationLogoURL = logo
				if compare {
					s.PreviousEvent.OrganizationLogoURL = logo
				}
				s.Branding = reportBranding{CompanyName: "PDF-ONLY-BRAND", LogoData: logo}
				client := &http.Client{Transport: recipientTestTransport(func(r *http.Request) (*http.Response, error) {
					var request struct {
						Input string `json:"input"`
					}
					if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
						t.Fatal(err)
					}
					if request.Input != string(baseline) {
						t.Fatalf("branding changed provider input: baseline=%d bytes, actual=%d bytes", len(baseline), len(request.Input))
					}
					for _, forbidden := range []string{"organization_logo_url", "data:image", "BASE64-LOGO", "PDF-ONLY-BRAND"} {
						if strings.Contains(request.Input, forbidden) {
							t.Fatalf("provider input contains branding: %s", forbidden)
						}
					}
					var sent struct {
						Event         recipientAIEvent  `json:"event"`
						PreviousEvent *recipientAIEvent `json:"previous_event"`
					}
					if err := json.Unmarshal([]byte(request.Input), &sent); err != nil {
						t.Fatal(err)
					}
					expected := recipientAIEvent{Title: "Partita attuale", Date: "2026-11-05"}
					if sent.Event != expected {
						t.Fatal("current event context changed")
					}
					if compare {
						expectedPrevious := recipientAIEvent{Title: "Partita precedente", Date: "2026-10-29"}
						if sent.PreviousEvent == nil || *sent.PreviousEvent != expectedPrevious {
							t.Fatal("previous event context changed")
						}
					} else if sent.PreviousEvent != nil {
						t.Fatal("previous event context manufactured")
					}
					t.Logf("AI input excludes logos: %d bytes", len(request.Input))
					provided, _ := json.Marshal(recipientAIProviderExample())
					raw, _ := json.Marshal(map[string]interface{}{"status": "completed", "output": []interface{}{map[string]interface{}{"type": "message", "content": []interface{}{map[string]interface{}{"type": "output_text", "text": string(provided)}}}}})
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(raw))), Header: make(http.Header)}, nil
				})}
				if _, err := callRecipientOpenAI(context.Background(), client, recipientOpenAIURL, "test-key", "test-model", s); err != nil {
					t.Fatal(err)
				}
				if s.Event.OrganizationLogoURL != logo || compare && s.PreviousEvent.OrganizationLogoURL != logo || s.Branding.LogoData != logo {
					t.Fatal("provider sanitization changed PDF branding in the snapshot")
				}
			})
		}
	}
}

func TestRecipientAIInputOmitsPrivateProvenanceAndCacheKeysVary(t *testing.T) {
	p, _ := findReportProfile("executive")
	s := engagementReportSnapshot{Event: reportEvent{ID: 1}, Recipient: &p, DataHash: "data-a", ComparePrevious: true, KPIs: []reportKPI{{ID: "visitors", Value: reportMetricFloat(1), Provenance: []reportManualSource{{SourceDescription: "PRIVATE-SOURCE", Note: "PRIVATE-NOTE", EnteredBy: 12345}}, Comparison: &reportComparison{PreviousProvenance: []reportManualSource{{Note: "PRIVATE-PREVIOUS"}}}}}}
	raw, _ := json.Marshal(recipientAIInput(s))
	if strings.Contains(string(raw), "PRIVATE") || strings.Contains(string(raw), "12345") {
		t.Fatal("private provenance sent to provider", string(raw))
	}
	key := recipientAnalysisKey(s, "model-a")
	if key == recipientAnalysisKey(s, "model-b") {
		t.Fatal("model ignored")
	}
	s.DataHash = "data-b"
	if key == recipientAnalysisKey(s, "model-a") {
		t.Fatal("source hash ignored")
	}
	s.DataHash = "data-a"
	s.ComparePrevious = false
	if key == recipientAnalysisKey(s, "model-a") {
		t.Fatal("comparison ignored")
	}
}

func TestRecipientPurgeCleansNewTablesWithoutCascades(t *testing.T) {
	f := newEngagementReportFixture(t)
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.saveRecipientReportSources, "PUT", f.event, recipientSourcePayload(100, 0, true), f.ctx, nil), 200)
	if err := f.router.db.PurgeEventData(f.event); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"engagement_report_manual_sources", "engagement_report_source_history", "engagement_report_analysis_cache"} {
		var count int
		if err := f.conn.QueryRow(`SELECT COUNT(*) FROM `+table+` WHERE event_id=?`, f.event).Scan(&count); err != nil || count != 0 {
			t.Fatal(table, count, err)
		}
	}
}

func TestRecipientProviderDiagnosticsDoNotLeakRawErrors(t *testing.T) {
	p, _ := findReportProfile("executive")
	s := engagementReportSnapshot{Recipient: &p, KPIs: []reportKPI{{ID: "visitors", Value: reportMetricFloat(1)}}}
	for _, tc := range []struct {
		name               string
		status             int
		body, reason, code string
	}{
		{"key", 401, `{"error":{"code":"invalid_api_key","message":"SECRET-KEY private source"}}`, "provider_http_error", "invalid_api_key"},
		{"quota", 429, `{"error":{"code":"insufficient_quota","message":"SECRET-KEY"}}`, "provider_http_error", "insufficient_quota"},
		{"schema", 400, `{"error":{"code":"invalid_json_schema","message":"SECRET-KEY"}}`, "provider_http_error", "invalid_json_schema"},
		{"incomplete", 200, `{"status":"incomplete","output":[]}`, "response_incomplete", ""},
		{"invalid_output", 200, `{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"{}"}]}]}`, "invalid_output", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &http.Client{Transport: recipientTestTransport(func(r *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body)), Header: http.Header{"X-Request-Id": []string{"req_test-report"}}}, nil
			})}
			_, err := callRecipientOpenAI(context.Background(), client, recipientOpenAIURL, "SECRET-KEY", "test-model", s)
			if err == nil {
				t.Fatal("expected failure")
			}
			d := recipientErrorDiagnostic(err)
			if d.Reason != tc.reason || d.ProviderCode != tc.code {
				t.Fatalf("%+v", d)
			}
			if tc.status != 200 && (d.HTTPStatus != tc.status || d.RequestID != "req_test-report") {
				t.Fatalf("missing provider identifiers %+v", d)
			}
			raw, _ := json.Marshal(d)
			if strings.Contains(string(raw), "SECRET") || strings.Contains(string(raw), "private source") {
				t.Fatal("raw provider error leaked", string(raw))
			}
		})
	}
	if d := recipientErrorDiagnostic(context.DeadlineExceeded); d.Reason != "request_timeout" {
		t.Fatal(d)
	}
	if d := recipientErrorDiagnostic(fmt.Errorf("SECRET-KEY")); d.Reason != "network_error" || strings.Contains(d.Message, "SECRET") {
		t.Fatal(d)
	}
}

func TestRecipientMissingConfigurationDiagnosticsPersistAndRegenerate(t *testing.T) {
	f := newEngagementReportFixture(t)
	recipientPageSignal(t, f)
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_MODEL", "test-model")
	s := recipientDraft(t, f, "executive", 0)
	input := map[string]interface{}{"calculation_id": s.CalculationID, "recipient": "executive", "compare_previous": true}
	for i := 0; i < 2; i++ {
		r := engagementReportRequest(t, f.router.generateRecipientReportAnalysis, "POST", f.event, input, f.ctx, nil)
		engagementReportRequireStatus(t, r, 200)
		var response struct {
			Status     string                 `json:"status"`
			Cached     bool                   `json:"cached"`
			Diagnostic *recipientAIDiagnostic `json:"diagnostic"`
		}
		_ = json.Unmarshal(r.Body.Bytes(), &response)
		if response.Status != "fallback" || response.Cached != (i == 1) || response.Diagnostic == nil || response.Diagnostic.Reason != "missing_api_key" {
			t.Fatal(r.Body.String())
		}
	}
	prepared, _ := recipientSnapshot(s, "executive", true, false)
	f.router.attachCachedRecipientAnalysis(&prepared)
	if prepared.AnalysisDiagnostic == nil || prepared.AnalysisDiagnostic.Reason != "missing_api_key" {
		t.Fatal("snapshot diagnostic lost")
	}
	t.Setenv("OPENAI_API_KEY", "test-key")
	old := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = old })
	valid, _ := json.Marshal(recipientAIProviderExample())
	http.DefaultTransport = recipientTestTransport(func(r *http.Request) (*http.Response, error) {
		raw, _ := json.Marshal(map[string]interface{}{"status": "completed", "output": []interface{}{map[string]interface{}{"type": "message", "content": []interface{}{map[string]interface{}{"type": "output_text", "text": string(valid)}}}}})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(raw))), Header: make(http.Header)}, nil
	})
	input["regenerate"] = true
	r := engagementReportRequest(t, f.router.generateRecipientReportAnalysis, "POST", f.event, input, f.ctx, nil)
	engagementReportRequireStatus(t, r, 200)
	if !strings.Contains(r.Body.String(), `"status":"ready"`) || !strings.Contains(r.Body.String(), `"diagnostic":null`) {
		t.Fatal(r.Body.String())
	}
	prepared, _ = recipientSnapshot(s, "executive", true, false)
	f.router.attachCachedRecipientAnalysis(&prepared)
	if prepared.AnalysisStatus != "ready" || prepared.AnalysisDiagnostic != nil {
		t.Fatal("old failure survived success")
	}
}

func TestRecipientConfigurationAndUnrequestedAnalysisReasons(t *testing.T) {
	for _, tc := range []struct {
		name, key, model, reason string
		page                     bool
	}{
		{"model", "test-key", "", "missing_model", true},
		{"data", "test-key", "test-model", "no_available_kpis", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newEngagementReportFixture(t)
			if tc.page {
				recipientPageSignal(t, f)
			}
			t.Setenv("OPENAI_API_KEY", tc.key)
			t.Setenv("OPENAI_MODEL", tc.model)
			s := recipientDraft(t, f, "executive", 0)
			prepared, _ := recipientSnapshot(s, "executive", true, false)
			f.router.attachCachedRecipientAnalysis(&prepared)
			if prepared.AnalysisDiagnostic.Reason != "analysis_not_generated" {
				t.Fatal(prepared.AnalysisDiagnostic)
			}
			r := engagementReportRequest(t, f.router.generateRecipientReportAnalysis, "POST", f.event, map[string]interface{}{"calculation_id": s.CalculationID, "recipient": "executive", "compare_previous": true}, f.ctx, nil)
			engagementReportRequireStatus(t, r, 200)
			if !strings.Contains(r.Body.String(), `"reason":"`+tc.reason+`"`) {
				t.Fatal(r.Body.String())
			}
		})
	}
}
