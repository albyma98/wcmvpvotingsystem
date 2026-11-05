package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
)

func recipientReviewGET(t *testing.T, f engagementReportFixture, s engagementReportSnapshot, profile string, compare bool, ctx reqcontext.RequestContext) *httptest.ResponseRecorder {
	t.Helper()
	h := func(w http.ResponseWriter, r *http.Request, c reqcontext.RequestContext) {
		r.URL.RawQuery = url.Values{"calculation_id": {s.CalculationID}, "recipient": {profile}, "compare_previous": {fmt.Sprint(compare)}}.Encode()
		f.router.getRecipientAnalysisReview(w, r, c)
	}
	return engagementReportRequest(t, h, "GET", f.event, nil, ctx, nil)
}

func recipientDecodeReview(t *testing.T, response *httptest.ResponseRecorder) recipientAnalysisReviewResponse {
	t.Helper()
	engagementReportRequireStatus(t, response, 200)
	var data recipientAnalysisReviewResponse
	if err := json.Unmarshal(response.Body.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	return data
}

func recipientReviewPayload(s engagementReportSnapshot, profile string, compare bool, draft recipientAnalysisReviewResponse, action string) map[string]interface{} {
	return map[string]interface{}{"calculation_id": s.CalculationID, "recipient": profile, "compare_previous": compare, "generation_version": draft.Review.GenerationVersion, "revision": draft.Review.Revision, "action": action, "analysis": draft.Analysis}
}

func recipientApproveAnalysis(t *testing.T, f engagementReportFixture, s engagementReportSnapshot, profile string, compare bool) int {
	t.Helper()
	draft := recipientDecodeReview(t, recipientReviewGET(t, f, s, profile, compare, f.ctx))
	r := engagementReportRequest(t, f.router.saveRecipientAnalysisReview, "PUT", f.event, recipientReviewPayload(s, profile, compare, draft, "approve"), f.ctx, nil)
	return recipientDecodeReview(t, r).Review.Revision
}

func TestRecipientNumericProseReviewApprovalDiscardAndRegeneration(t *testing.T) {
	f := newEngagementReportFixture(t)
	recipientPageSignal(t, f)
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_MODEL", "test-model")
	old := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = old })
	calls := 0
	provided := recipientAIProviderExample()
	provided.Headline = "Due opportunità e partecipazione al 42%"
	http.DefaultTransport = recipientTestTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		text, _ := json.Marshal(provided)
		raw, _ := json.Marshal(map[string]interface{}{"status": "completed", "usage": map[string]int{"input_tokens": 1200, "output_tokens": 600, "total_tokens": 1800}, "output": []interface{}{map[string]interface{}{"type": "message", "content": []interface{}{map[string]interface{}{"type": "output_text", "text": string(text)}}}}})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(raw))), Header: make(http.Header)}, nil
	})
	s := recipientDraft(t, f, "executive", 0)
	generate := map[string]interface{}{"calculation_id": s.CalculationID, "recipient": "executive", "compare_previous": true}
	r := engagementReportRequest(t, f.router.generateRecipientReportAnalysis, "POST", f.event, generate, f.ctx, nil)
	draft := recipientDecodeReview(t, r)
	if calls != 1 || draft.Status != "ready" || draft.Review.State != "draft" || draft.Analysis.Headline != provided.Headline || len(draft.Warnings) != 1 || draft.Usage.Requests != 1 {
		t.Fatal("numeric prose was blocked, retried or auto-approved", r.Body.String(), calls)
	}
	export := map[string]interface{}{"calculation_id": s.CalculationID, "recipient": "executive", "compare_previous": true, "include_ai": true, "analysis_review_revision": 0}
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.approveEngagementReport, "POST", f.event, export, f.ctx, nil), 409)
	draft.Analysis.Headline = "Titolo rivisto dall’amministratore"
	draft.Analysis.Summary = strings.Repeat("Interpretazione verificata dei dati. ", 30) + "FINE SINTESI 42%"
	draft.Analysis.PositiveInsights[0].Title = "Evidenza rivista"
	draft.Analysis.PositiveInsights[0].Description = "Testo rivisto, con due opportunità."
	draft.Analysis.Recommendations[0].Action = "Azione rivista"
	draft.Analysis.Recommendations[0].Reason = "Motivazione rivista"
	draft.Analysis.Limitations = []string{"Nota rivista dal responsabile."}
	r = engagementReportRequest(t, f.router.saveRecipientAnalysisReview, "PUT", f.event, recipientReviewPayload(s, "executive", true, draft, "draft"), f.ctx, nil)
	draft = recipientDecodeReview(t, r)
	if draft.Review.State != "draft" || draft.Review.Revision != 1 || !draft.Review.Edited || calls != 1 {
		t.Fatal("draft save did not retain edited text without AI", r.Body.String())
	}
	// Reopening a new calculation with identical inputs restores the saved draft.
	reopened := recipientDraft(t, f, "executive", 0)
	loaded := recipientDecodeReview(t, recipientReviewGET(t, f, reopened, "executive", true, f.ctx))
	if loaded.Analysis.Summary != draft.Analysis.Summary || loaded.Review.Revision != 1 {
		t.Fatal("draft lost on reopen")
	}
	r = engagementReportRequest(t, f.router.saveRecipientAnalysisReview, "PUT", f.event, recipientReviewPayload(s, "executive", true, draft, "approve"), f.ctx, nil)
	approved := recipientDecodeReview(t, r)
	if approved.Review.State != "approved" || approved.Review.Revision != 2 || approved.Review.ReviewedBy != f.ctx.AdminUsername || approved.Review.ReviewedAt == "" {
		t.Fatal("approval audit missing", r.Body.String())
	}
	delete(export, "analysis_review_revision")
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.approveEngagementReport, "POST", f.event, export, f.ctx, nil), 409)
	export["analysis_review_revision"] = 2
	r = engagementReportRequest(t, f.router.approveEngagementReport, "POST", f.event, export, f.ctx, nil)
	engagementReportRequireStatus(t, r, 200)
	frozen := engagementReportDecodeSnapshot(t, r)
	if frozen.Analysis == nil || reportDigest(*frozen.Analysis) != reportDigest(approved.Analysis) || frozen.AnalysisReview.State != "approved" || frozen.AnalysisReview.Revision != 2 || calls != 1 {
		t.Fatal("PDF approval did not freeze exact approved text", r.Body.String())
	}
	// Editing an approved analysis immediately revokes its saved approval.
	approved.Analysis.Summary = "Seconda versione da approvare."
	r = engagementReportRequest(t, f.router.saveRecipientAnalysisReview, "PUT", f.event, recipientReviewPayload(s, "executive", true, approved, "draft"), f.ctx, nil)
	draft = recipientDecodeReview(t, r)
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.approveEngagementReport, "POST", f.event, export, f.ctx, nil), 409)
	r = engagementReportRequest(t, f.router.saveRecipientAnalysisReview, "PUT", f.event, recipientReviewPayload(s, "executive", true, draft, "discard"), f.ctx, nil)
	discarded := recipientDecodeReview(t, r)
	if discarded.Review.State != "discarded" || calls != 1 {
		t.Fatal("discard invoked AI or failed")
	}
	export["include_ai"] = false
	r = engagementReportRequest(t, f.router.approveEngagementReport, "POST", f.event, export, f.ctx, nil)
	engagementReportRequireStatus(t, r, 200)
	without := engagementReportDecodeSnapshot(t, r)
	if without.Analysis != nil || without.AnalysisReview != nil {
		t.Fatal("discarded analysis included in PDF")
	}
	var archived string
	if err := f.conn.QueryRow(`SELECT payload FROM engagement_report_snapshots WHERE id=?`, frozen.ID).Scan(&archived); err != nil || !strings.Contains(archived, "FINE SINTESI 42%") {
		t.Fatal("later review changed frozen PDF", err)
	}
	r = engagementReportRequest(t, f.router.saveRecipientAnalysisReview, "PUT", f.event, recipientReviewPayload(s, "executive", true, discarded, "approve"), f.ctx, nil)
	approved = recipientDecodeReview(t, r)
	generate["regenerate"] = true
	r = engagementReportRequest(t, f.router.generateRecipientReportAnalysis, "POST", f.event, generate, f.ctx, nil)
	regenerated := recipientDecodeReview(t, r)
	if calls != 2 || regenerated.Review.State != "draft" || regenerated.Review.GenerationVersion == approved.Review.GenerationVersion || regenerated.Analysis.Headline != provided.Headline {
		t.Fatal("regeneration inherited approval or edits", r.Body.String())
	}
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.saveRecipientAnalysisReview, "PUT", f.event, recipientReviewPayload(s, "executive", true, approved, "approve"), f.ctx, nil), 409)
	var cached string
	if err := f.conn.QueryRow(`SELECT response_json FROM engagement_report_analysis_cache WHERE cache_key=?`, recipientAnalysisKeyMust(t, s, "executive", true)).Scan(&cached); err != nil || strings.Contains(cached, "Titolo rivisto") {
		t.Fatal("editorial edits overwrote original provider output", err)
	}
	if _, err := f.conn.Exec(`UPDATE engagement_report_analysis_cache SET status='pending' WHERE cache_key=?`, recipientAnalysisKeyMust(t, s, "executive", true)); err != nil {
		t.Fatal(err)
	}
	engagementReportRequireStatus(t, recipientReviewGET(t, f, s, "executive", true, f.ctx), 409)
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.saveRecipientAnalysisReview, "PUT", f.event, recipientReviewPayload(s, "executive", true, regenerated, "approve"), f.ctx, nil), 409)
}

func recipientAnalysisKeyMust(t *testing.T, s engagementReportSnapshot, profile string, compare bool) string {
	t.Helper()
	prepared, err := recipientSnapshot(s, profile, compare, false)
	if err != nil {
		t.Fatal(err)
	}
	return recipientAnalysisKey(prepared, recipientReportModel())
}

func TestRecipientReviewAuthorizationIsolationValidationAndPurge(t *testing.T) {
	f := newEngagementReportFixture(t)
	recipientPageSignal(t, f)
	s := recipientDraft(t, f, "executive", 0)
	draft := recipientDecodeReview(t, recipientReviewGET(t, f, s, "executive", true, f.ctx))
	if draft.Review.GenerationVersion != 0 || draft.Status != "fallback" {
		t.Fatal("local draft should not call AI")
	}
	for _, kind := range []string{"blank", "too-long", "unknown-kpi", "unknown-field"} {
		payload := recipientReviewPayload(s, "executive", true, draft, "approve")
		var analysis map[string]interface{}
		raw, _ := json.Marshal(draft.Analysis)
		_ = json.Unmarshal(raw, &analysis)
		switch kind {
		case "blank":
			analysis["summary"] = ""
		case "too-long":
			analysis["summary"] = strings.Repeat("x", 6001)
		case "unknown-kpi":
			analysis["positive_insights"] = []interface{}{map[string]interface{}{"title": "Titolo", "description": "Testo", "metric_ids": []string{"unknown"}}}
		case "unknown-field":
			analysis["unexpected"] = true
		}
		payload["analysis"] = analysis
		engagementReportRequireStatus(t, engagementReportRequest(t, f.router.saveRecipientAnalysisReview, "PUT", f.event, payload, f.ctx, nil), 400)
	}
	draft.Analysis.Summary = ""
	r := engagementReportRequest(t, f.router.saveRecipientAnalysisReview, "PUT", f.event, recipientReviewPayload(s, "executive", true, draft, "draft"), f.ctx, nil)
	draft = recipientDecodeReview(t, r)
	// Even an incomplete draft can be discarded without approving its contents.
	r = engagementReportRequest(t, f.router.saveRecipientAnalysisReview, "PUT", f.event, recipientReviewPayload(s, "executive", true, draft, "discard"), f.ctx, nil)
	draft = recipientDecodeReview(t, r)
	stale := recipientReviewPayload(s, "executive", true, draft, "discard")
	stale["revision"] = 0
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.saveRecipientAnalysisReview, "PUT", f.event, stale, f.ctx, nil), 409)
	foreign := f.ctx
	foreign.OrganizationID = f.other.ID
	engagementReportRequireStatus(t, recipientReviewGET(t, f, s, "executive", true, foreign), 404)
	otherAdmin := f.ctx
	otherAdmin.AdminID++
	engagementReportRequireStatus(t, recipientReviewGET(t, f, s, "executive", true, otherAdmin), 404)
	otherFixture := f
	otherFixture.ctx = otherAdmin
	otherCalculation := recipientDraft(t, otherFixture, "executive", 0)
	otherDraft := recipientDecodeReview(t, recipientReviewGET(t, otherFixture, otherCalculation, "executive", true, otherAdmin))
	if otherDraft.Review.Revision != 0 || otherDraft.Review.State != "draft" {
		t.Fatal("another admin inherited review")
	}
	for _, scope := range []struct {
		profile string
		compare bool
	}{{"marketing", true}, {"executive", false}} {
		isolated := recipientDecodeReview(t, recipientReviewGET(t, f, s, scope.profile, scope.compare, f.ctx))
		if isolated.Review.Revision != 0 || isolated.Review.State != "draft" {
			t.Fatal("review crossed profile or comparison scope")
		}
	}
	recipientApproveAnalysis(t, f, s, "marketing", true)
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.saveRecipientReportSources, "PUT", f.event, recipientSourcePayload(100, 0, true), f.ctx, nil), 200)
	updated := recipientDraft(t, f, "marketing", 0)
	changed := recipientDecodeReview(t, recipientReviewGET(t, f, updated, "marketing", true, f.ctx))
	if updated.DataHash == s.DataHash || changed.Review.Revision != 0 || changed.Review.State != "draft" {
		t.Fatal("changed KPI data inherited editorial approval")
	}
	if err := f.router.ensureRecipientReportTables(); err != nil {
		t.Fatal("review migration is not idempotent", err)
	}
	if err := f.router.db.PurgeEventData(f.event); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := f.conn.QueryRow(`SELECT COUNT(*) FROM engagement_report_analysis_reviews WHERE event_id=?`, f.event).Scan(&count); err != nil || count != 0 {
		t.Fatal("purge retained editorial reviews", count, err)
	}
}
