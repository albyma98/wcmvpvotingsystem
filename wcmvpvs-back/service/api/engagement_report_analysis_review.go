package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
)

type recipientAnalysisReview struct {
	State             string `json:"state"`
	GenerationVersion int    `json:"generation_version"`
	Revision          int    `json:"revision"`
	UpdatedAt         string `json:"updated_at,omitempty"`
	ReviewedAt        string `json:"reviewed_at,omitempty"`
	ReviewedBy        string `json:"reviewed_by,omitempty"`
	Edited            bool   `json:"edited"`
}

type recipientAnalysisReviewResponse struct {
	Analysis      recipientAnalysis       `json:"analysis"`
	Review        recipientAnalysisReview `json:"review"`
	Status        string                  `json:"status"`
	Model         string                  `json:"model"`
	PromptVersion string                  `json:"prompt_version"`
	Warnings      []string                `json:"warnings"`
	Usage         *recipientAIUsage       `json:"usage,omitempty"`
	Diagnostic    *recipientAIDiagnostic  `json:"diagnostic"`
}

type recipientReviewQuerier interface {
	QueryRow(string, ...interface{}) *sql.Row
}

var errRecipientReviewPending = errors.New("analysis generation pending")

// AI output stays in its shared cache; editorial decisions belong to the
// reviewing administrator and the exact data/profile/model/generation.
func loadRecipientAnalysisReview(q recipientReviewQuerier, s engagementReportSnapshot, adminID int) (recipientAnalysisReviewResponse, error) {
	model := recipientReportModel()
	key := recipientAnalysisKey(s, model)
	response := recipientAnalysisReviewResponse{Analysis: recipientFallback(s), Review: recipientAnalysisReview{State: "draft"}, Status: "fallback", Model: model, PromptVersion: recipientPromptVersion, Diagnostic: recipientDiagnostic("analysis_not_generated")}
	var raw, status string
	err := q.QueryRow(`SELECT response_json,status,version FROM engagement_report_analysis_cache WHERE cache_key=?`, key).Scan(&raw, &status, &response.Review.GenerationVersion)
	if err != nil && err != sql.ErrNoRows {
		return response, err
	}
	if err == nil {
		if status == "pending" {
			return response, errRecipientReviewPending
		}
		var cached recipientCachedAnalysis
		if err := json.Unmarshal([]byte(raw), &cached); err != nil {
			return response, err
		}
		response.Analysis, response.Status, response.Usage, response.Diagnostic = cached.recipientAnalysis, status, cached.Usage, cached.Diagnostic
	}
	original := reportDigest(response.Analysis)
	var savedGeneration int
	var saved recipientAnalysisReview
	err = q.QueryRow(`SELECT generation_version,state,analysis_json,revision,updated_at,reviewed_at,reviewed_by FROM engagement_report_analysis_reviews WHERE cache_key=? AND admin_id=?`, key, adminID).Scan(&savedGeneration, &saved.State, &raw, &saved.Revision, &saved.UpdatedAt, &saved.ReviewedAt, &saved.ReviewedBy)
	if err != nil && err != sql.ErrNoRows {
		return response, err
	}
	if err == nil {
		// Keep the row revision for optimistic locking even after regeneration.
		response.Review.Revision = saved.Revision
		if savedGeneration == response.Review.GenerationVersion {
			if err := json.Unmarshal([]byte(raw), &response.Analysis); err != nil {
				return response, err
			}
			saved.GenerationVersion = savedGeneration
			response.Review = saved
		}
	}
	response.Review.Edited = reportDigest(response.Analysis) != original
	response.Warnings = recipientAnalysisWarnings(response.Analysis)
	return response, nil
}

func recipientEditorialSchema(s engagementReportSnapshot, approving bool) map[string]interface{} {
	schema := recipientAnalysisSchema(s)
	props := schema["properties"].(map[string]interface{})
	setText := func(field map[string]interface{}, max int) {
		field["minLength"] = 0
		if approving {
			field["minLength"] = 1
		}
		field["maxLength"] = max
	}
	setText(props["headline"].(map[string]interface{}), 180)
	setText(props["summary"].(map[string]interface{}), 6000)
	for _, name := range []string{"positive_insights", "critical_insights"} {
		fields := props[name].(map[string]interface{})["items"].(map[string]interface{})["properties"].(map[string]interface{})
		setText(fields["title"].(map[string]interface{}), 180)
		setText(fields["description"].(map[string]interface{}), 3000)
	}
	fields := props["recommendations"].(map[string]interface{})["items"].(map[string]interface{})["properties"].(map[string]interface{})
	setText(fields["action"].(map[string]interface{}), 500)
	setText(fields["reason"].(map[string]interface{}), 3000)
	limits := props["limitations"].(map[string]interface{})
	limits["maxItems"] = 32
	setText(limits["items"].(map[string]interface{}), 1000)
	return schema
}

func validateRecipientEditorialAnalysis(raw []byte, s engagementReportSnapshot, approving bool) (recipientAnalysis, error) {
	var tree interface{}
	if err := json.Unmarshal(raw, &tree); err != nil {
		return recipientAnalysis{}, err
	}
	if err := validateRecipientJSON(tree, recipientEditorialSchema(s, approving)); err != nil {
		return recipientAnalysis{}, err
	}
	var analysis recipientAnalysis
	if err := json.Unmarshal(raw, &analysis); err != nil {
		return analysis, err
	}
	if len(analysis.PositiveInsights)+len(analysis.CriticalInsights) > s.Recipient.MaxInsights {
		return analysis, fmt.Errorf("insight limit exceeded")
	}
	return analysis, nil
}

func (rt *_router) getRecipientAnalysisReview(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, _, ok := rt.reportEventScope(w, r, ctx)
	if !ok {
		return
	}
	compare, err := strconv.ParseBool(r.URL.Query().Get("compare_previous"))
	if err != nil {
		reportError(w, 400, "Opzione di confronto non valida.")
		return
	}
	s, err := rt.loadRecipientCalculation(r.URL.Query().Get("calculation_id"), id, ctx.AdminID)
	if err != nil {
		reportError(w, 404, "Calcolo non disponibile.")
		return
	}
	s, err = recipientSnapshot(s, r.URL.Query().Get("recipient"), compare, false)
	if err != nil {
		reportError(w, 400, "Destinatario non valido.")
		return
	}
	response, err := loadRecipientAnalysisReview(rt.db.SQLConn(), s, ctx.AdminID)
	if err != nil {
		recipientReviewError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	_ = writeJSON(w, 200, response)
}

func recipientReviewError(w http.ResponseWriter, err error) {
	if errors.Is(err, errRecipientReviewPending) {
		reportError(w, 409, "L’analisi è in generazione. Attendi prima di rivederla.")
		return
	}
	reportError(w, 503, "Revisione dell’analisi temporaneamente non disponibile.")
}

func (rt *_router) saveRecipientAnalysisReview(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, _, ok := rt.reportEventScope(w, r, ctx)
	if !ok {
		return
	}
	var input struct {
		CalculationID     string          `json:"calculation_id"`
		Recipient         string          `json:"recipient"`
		ComparePrevious   bool            `json:"compare_previous"`
		GenerationVersion int             `json:"generation_version"`
		Revision          int             `json:"revision"`
		Action            string          `json:"action"`
		Analysis          json.RawMessage `json:"analysis"`
	}
	if !decodeReportBody(w, r, 128*1024, &input) {
		return
	}
	if input.Revision < 0 || input.GenerationVersion < 0 || (input.Action != "draft" && input.Action != "approve" && input.Action != "discard") {
		reportError(w, 400, "Azione o revisione non valida.")
		return
	}
	s, err := rt.loadRecipientCalculation(input.CalculationID, id, ctx.AdminID)
	if err != nil {
		reportError(w, 404, "Calcolo non disponibile.")
		return
	}
	s, err = recipientSnapshot(s, input.Recipient, input.ComparePrevious, false)
	if err != nil {
		reportError(w, 400, "Destinatario non valido.")
		return
	}
	tx, err := rt.db.SQLConn().Begin()
	if err != nil {
		recipientReviewError(w, err)
		return
	}
	defer tx.Rollback()
	response, err := loadRecipientAnalysisReview(tx, s, ctx.AdminID)
	if err != nil {
		recipientReviewError(w, err)
		return
	}
	if response.Review.Revision != input.Revision || response.Review.GenerationVersion != input.GenerationVersion {
		reportError(w, 409, "La bozza è cambiata o è stata rigenerata. Ricaricala prima di continuare.")
		return
	}
	state := "discarded"
	if input.Action != "discard" {
		response.Analysis, err = validateRecipientEditorialAnalysis(input.Analysis, s, input.Action == "approve")
		if err != nil {
			reportError(w, 400, "Verifica i testi, i limiti di lunghezza e i riferimenti ai KPI disponibili. Rimuovi gli elementi vuoti prima di approvare.")
			return
		}
		state = "draft"
		if input.Action == "approve" {
			state = "approved"
		}
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	reviewedAt, reviewedBy := "", ""
	if state != "draft" {
		reviewedAt, reviewedBy = now, ctx.AdminUsername
	}
	raw, _ := json.Marshal(response.Analysis)
	key := recipientAnalysisKey(s, recipientReportModel())
	result, err := tx.Exec(`INSERT INTO engagement_report_analysis_reviews(cache_key,admin_id,event_id,generation_version,state,analysis_json,revision,updated_at,reviewed_at,reviewed_by) VALUES(?,?,?,?,?,?,?,?,?,?) ON CONFLICT(cache_key,admin_id) DO UPDATE SET generation_version=excluded.generation_version,state=excluded.state,analysis_json=excluded.analysis_json,revision=excluded.revision,updated_at=excluded.updated_at,reviewed_at=excluded.reviewed_at,reviewed_by=excluded.reviewed_by WHERE engagement_report_analysis_reviews.revision=?`, key, ctx.AdminID, id, input.GenerationVersion, state, string(raw), input.Revision+1, now, reviewedAt, reviewedBy, input.Revision)
	if err != nil {
		recipientReviewError(w, err)
		return
	}
	count, _ := result.RowsAffected()
	if count != 1 {
		reportError(w, 409, "La bozza è stata modificata in un’altra finestra. Ricaricala.")
		return
	}
	response, err = loadRecipientAnalysisReview(tx, s, ctx.AdminID)
	if err != nil {
		recipientReviewError(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		recipientReviewError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	_ = writeJSON(w, 200, response)
}
