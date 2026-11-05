package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
)

const recipientPromptVersion = "recipient-report-v4"
const recipientOpenAIURL = "https://api.openai.com/v1/responses"

type recipientInsight struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	MetricIDs   []string `json:"metric_ids"`
}
type recipientRecommendation struct {
	Action    string   `json:"action"`
	Reason    string   `json:"reason"`
	MetricIDs []string `json:"metric_ids"`
	Priority  string   `json:"priority"`
}
type recipientAnalysis struct {
	Headline         string                    `json:"headline"`
	Summary          string                    `json:"summary"`
	PositiveInsights []recipientInsight        `json:"positive_insights"`
	CriticalInsights []recipientInsight        `json:"critical_insights"`
	Recommendations  []recipientRecommendation `json:"recommendations"`
	Limitations      []string                  `json:"limitations"`
}

// The provider uses one list so maxItems limits positive and critical insights
// together. Public responses, cached analyses and PDFs retain their two lists.
type recipientProviderInsight struct {
	recipientInsight
	Kind string `json:"kind"`
}

type recipientProviderAnalysis struct {
	Headline        string                     `json:"headline"`
	Summary         string                     `json:"summary"`
	Insights        []recipientProviderInsight `json:"insights"`
	Recommendations []recipientRecommendation  `json:"recommendations"`
	Limitations     []string                   `json:"limitations"`
}

// Diagnostics contain fixed messages and provider identifiers only. Never
// expose the provider's raw error message, request body or Authorization value.
type recipientAIDiagnostic struct {
	Reason          string `json:"reason"`
	Message         string `json:"message"`
	HTTPStatus      int    `json:"http_status,omitempty"`
	ProviderCode    string `json:"provider_code,omitempty"`
	RequestID       string `json:"request_id,omitempty"`
	ValidationError string `json:"validation_error,omitempty"`
}

func (d *recipientAIDiagnostic) Error() string { return d.Reason }

type recipientCachedAnalysis struct {
	recipientAnalysis
	Diagnostic *recipientAIDiagnostic `json:"diagnostic,omitempty"`
	Usage      *recipientAIUsage      `json:"usage,omitempty"`
}

// Provider counters include every attempt, even locally rejected output.
// A cached response returns the original counters without a new API request.
type recipientAIUsage struct {
	Requests          int `json:"requests"`
	ReportedRequests  int `json:"reported_requests"`
	InputTokens       int `json:"input_tokens"`
	CachedInputTokens int `json:"cached_input_tokens"`
	OutputTokens      int `json:"output_tokens"`
	TotalTokens       int `json:"total_tokens"`
	RequestBytes      int `json:"max_request_bytes"`
}

type recipientAIAttemptOptions struct {
	RetryHint string
	Usage     *recipientAIUsage
}

func recipientDiagnostic(reason string) *recipientAIDiagnostic {
	messages := map[string]string{
		"missing_api_key":        "OPENAI_API_KEY non è presente nel processo backend.",
		"missing_model":          "OPENAI_MODEL non è configurato nel processo backend.",
		"no_available_kpis":      "Nessun KPI del destinatario contiene dati disponibili da analizzare.",
		"analysis_not_generated": "Nessuna analisi generata per questi dati e opzioni. Premi Genera analisi.",
		"analysis_pending":       "L'analisi è ancora in elaborazione. Attendi e riprova l'anteprima.",
		"cached_fallback":        "Sintesi deterministica salvata in precedenza. Premi Rigenera analisi per verificare la causa attuale.",
		"cache_unavailable":      "La cache dell'analisi non è disponibile.",
		"request_timeout":        "Tempo massimo dell'analisi OpenAI superato.",
		"network_error":          "Connessione a OpenAI non riuscita. Verifica rete, DNS e accesso HTTPS del backend.",
		"provider_http_error":    "OpenAI ha rifiutato la richiesta. Controlla http_status e provider_code.",
		"invalid_response":       "La risposta OpenAI non è leggibile nel formato previsto.",
		"response_too_large":     "La risposta OpenAI supera la dimensione consentita.",
		"response_incomplete":    "OpenAI ha restituito una risposta incompleta, per esempio per limite di output.",
		"provider_refusal":       "OpenAI non ha prodotto l'analisi richiesta.",
		"invalid_output":         "L'analisi OpenAI non ha superato i controlli sulla struttura o sui riferimenti ai KPI disponibili.",
		"invalid_input":          "I dati del report non possono essere inviati al modello. Verifica i KPI del calcolo.",
		"input_too_large":        "La richiesta supera il budget di input del report. Nessuna chiamata OpenAI eseguita.",
	}
	return &recipientAIDiagnostic{Reason: reason, Message: messages[reason]}
}

func recipientErrorDiagnostic(err error) *recipientAIDiagnostic {
	var diagnostic *recipientAIDiagnostic
	if errors.As(err, &diagnostic) {
		return diagnostic
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return recipientDiagnostic("request_timeout")
	}
	return recipientDiagnostic("network_error")
}

func recipientReportModel() string { return strings.TrimSpace(os.Getenv("OPENAI_MODEL")) }
func recipientReportTimeout() time.Duration {
	d, err := time.ParseDuration(os.Getenv("OPENAI_REPORT_TIMEOUT"))
	if err != nil || d < time.Second || d > 2*time.Minute {
		return 45 * time.Second
	}
	return d
}

// RecipientReportRequestTimeout is the processing budget the HTTP server must
// allow before sending its response, including the single validation retry.
func RecipientReportRequestTimeout() time.Duration { return recipientReportTimeout() }

func recipientSelectedKPIs(s engagementReportSnapshot) []reportKPI {
	out := []reportKPI{}
	if s.Recipient == nil {
		return out
	}
	for _, id := range s.Recipient.KPIIDs {
		for _, k := range s.KPIs {
			if k.ID == id {
				out = append(out, k)
				break
			}
		}
	}
	return out
}

func recipientAnalysisKey(s engagementReportSnapshot, model string) string {
	previousID := 0
	if s.PreviousEvent != nil {
		previousID = s.PreviousEvent.ID
	}
	return reportDigest(struct {
		Event               int
		Profile             *reportRecipientProfile
		Previous            int
		Hash, Prompt, Model string
		Compare             bool
	}{s.Event.ID, s.Recipient, previousID, s.DataHash, recipientPromptVersion, model, s.ComparePrevious})
}

func recipientFallback(s engagementReportSnapshot) recipientAnalysis {
	a := recipientAnalysis{Headline: "Lettura dei dati disponibili", Summary: "Il report presenta soltanto indicatori sostenuti dalle fonti dell'evento. Le variazioni descrivono differenze osservate e non dimostrano cause o benefici economici.", PositiveInsights: []recipientInsight{}, CriticalInsights: []recipientInsight{}, Recommendations: []recipientRecommendation{}, Limitations: []string{"Analisi deterministica: OpenAI non richiesta o non disponibile. N/D non equivale a zero.", "Le identità osservabili non certificano persone fisiche. I campioni rispondenti possono essere piccoli e non rappresentativi."}}
	if s.Recipient != nil {
		switch s.Recipient.ID {
		case "executive":
			a.Headline = "Partecipazione e valore per la società"
			a.Summary = "Leggere acquisizione e ritorno insieme alla copertura commerciale. Consolidare la misurazione tra eventi prima di definire obiettivi di crescita."
		case "marketing":
			a.Headline = "Acquisizione e comportamento dei tifosi"
			a.Summary = "Esaminare il percorso dalla partecipazione alla registrazione e il ritorno dei profili. Verificare la copertura dei denominatori prima di pianificare interventi."
		case "sponsor":
			a.Headline = "Visibilità e interazioni misurabili"
			a.Summary = "Copertura e impression documentano visibilità; clic e giochi descrivono interazioni osservate. Non costituiscono vendite o ritorno economico. Il pubblico complessivo resta contesto dell'evento."
		case "analytics":
			a.Headline = "Qualità e limiti della misurazione"
			a.Summary = "Interpretare i valori insieme alle definizioni, ai denominatori e alle fonti. Differenze tra eventi possono riflettere coperture diverse; il ritorno misura il solo evento precedente."
		}
	}
	for _, k := range recipientSelectedKPIs(s) {
		if k.Value == nil {
			a.Limitations = append(a.Limitations, k.Label+": "+k.Availability.Message)
		}
	}
	return a
}

func recipientAnalysisSchema(s engagementReportSnapshot) map[string]interface{} {
	text := func(max int) map[string]interface{} {
		return map[string]interface{}{"type": "string", "minLength": 1, "maxLength": max}
	}
	object := func(properties map[string]interface{}) map[string]interface{} {
		required := []string{}
		for key := range properties {
			required = append(required, key)
		}
		sort.Strings(required)
		return map[string]interface{}{"type": "object", "properties": properties, "required": required, "additionalProperties": false}
	}
	ids := []string{}
	for _, k := range recipientSelectedKPIs(s) {
		if k.Value != nil {
			ids = append(ids, k.ID)
		}
	}
	metricIDs := map[string]interface{}{"type": "array", "minItems": 1, "maxItems": len(ids), "items": map[string]interface{}{"type": "string", "enum": ids}}
	insight := object(map[string]interface{}{"title": text(70), "description": text(200), "metric_ids": metricIDs})
	recommendation := object(map[string]interface{}{"action": text(140), "reason": text(180), "metric_ids": metricIDs, "priority": map[string]interface{}{"type": "string", "enum": []string{"high", "medium", "low"}}})
	array := func(item interface{}, max int) map[string]interface{} {
		return map[string]interface{}{"type": "array", "items": item, "maxItems": max}
	}
	return object(map[string]interface{}{"headline": text(90), "summary": text(360), "positive_insights": array(insight, s.Recipient.MaxInsights), "critical_insights": array(insight, s.Recipient.MaxInsights), "recommendations": array(recommendation, 3), "limitations": array(text(160), 6)})
}

func recipientProviderSchema(s engagementReportSnapshot) map[string]interface{} {
	// Each call creates a fresh schema, keeping the public contract independent.
	schema := recipientAnalysisSchema(s)
	properties := schema["properties"].(map[string]interface{})
	insights := properties["positive_insights"].(map[string]interface{})
	item := insights["items"].(map[string]interface{})
	item["properties"].(map[string]interface{})["kind"] = map[string]interface{}{"type": "string", "enum": []string{"positive", "critical"}}
	itemRequired := append(item["required"].([]string), "kind")
	sort.Strings(itemRequired)
	item["required"] = itemRequired
	delete(properties, "positive_insights")
	delete(properties, "critical_insights")
	properties["insights"] = insights
	required := []string{}
	for key := range properties {
		required = append(required, key)
	}
	sort.Strings(required)
	schema["required"] = required
	return schema
}

func validateRecipientProviderAnalysis(raw []byte, s engagementReportSnapshot) (recipientAnalysis, error) {
	var tree interface{}
	if err := json.Unmarshal(raw, &tree); err != nil {
		return recipientAnalysis{}, err
	}
	if err := validateRecipientJSON(tree, recipientProviderSchema(s)); err != nil {
		return recipientAnalysis{}, err
	}
	var provided recipientProviderAnalysis
	if err := json.Unmarshal(raw, &provided); err != nil {
		return recipientAnalysis{}, err
	}
	analysis := recipientAnalysis{Headline: provided.Headline, Summary: provided.Summary, PositiveInsights: []recipientInsight{}, CriticalInsights: []recipientInsight{}, Recommendations: provided.Recommendations, Limitations: provided.Limitations}
	for _, insight := range provided.Insights {
		if insight.Kind == "positive" {
			analysis.PositiveInsights = append(analysis.PositiveInsights, insight.recipientInsight)
		} else {
			analysis.CriticalInsights = append(analysis.CriticalInsights, insight.recipientInsight)
		}
	}
	// Retain public validation of budget, references and lengths. Numeric prose
	// is retained as an editable draft requiring administrator approval.
	normalized, err := json.Marshal(analysis)
	if err != nil {
		return recipientAnalysis{}, err
	}
	return validateRecipientAnalysis(normalized, s)
}

// Validate the very same strict schema used by Responses. Reject omitted/null
// fields, wrong types, unknown keys, invalid references, lengths and priorities.
func validateRecipientJSON(value interface{}, schema map[string]interface{}) error {
	switch schema["type"] {
	case "object":
		v, ok := value.(map[string]interface{})
		if !ok {
			return fmt.Errorf("object required")
		}
		props := schema["properties"].(map[string]interface{})
		if len(v) != len(props) {
			return fmt.Errorf("required or unknown properties")
		}
		for key, s := range props {
			child, ok := v[key]
			if !ok {
				return fmt.Errorf("missing %s", key)
			}
			if err := validateRecipientJSON(child, s.(map[string]interface{})); err != nil {
				return err
			}
		}
	case "array":
		v, ok := value.([]interface{})
		if !ok {
			return fmt.Errorf("array required")
		}
		if max, ok := schema["maxItems"].(int); ok && len(v) > max {
			return fmt.Errorf("too many items")
		}
		if min, ok := schema["minItems"].(int); ok && len(v) < min {
			return fmt.Errorf("too few items")
		}
		for _, child := range v {
			if err := validateRecipientJSON(child, schema["items"].(map[string]interface{})); err != nil {
				return err
			}
		}
	case "string":
		v, ok := value.(string)
		if !ok || strings.ContainsRune(v, '\x00') {
			return fmt.Errorf("valid string required")
		}
		if min, ok := schema["minLength"].(int); ok && utf8.RuneCountInString(strings.TrimSpace(v)) < min {
			return fmt.Errorf("empty text")
		}
		if max, ok := schema["maxLength"].(int); ok && utf8.RuneCountInString(v) > max {
			return fmt.Errorf("text too long")
		}
		if values, ok := schema["enum"].([]string); ok {
			for _, allowed := range values {
				if v == allowed {
					return nil
				}
			}
			return fmt.Errorf("invalid enum or metric")
		}
	default:
		return fmt.Errorf("unsupported schema")
	}
	return nil
}

var recipientNumberWords = regexp.MustCompile(`(?i)\b(zero|uno|due|tre|quattro|cinque|sei|sette|otto|nove|dieci|cento|mille|milione|milioni|percento|percentuale|one|two|three|hundred|thousand|million)\b`)

func validateRecipientAnalysis(raw []byte, s engagementReportSnapshot) (recipientAnalysis, error) {
	var tree interface{}
	if err := json.Unmarshal(raw, &tree); err != nil {
		return recipientAnalysis{}, err
	}
	if err := validateRecipientJSON(tree, recipientAnalysisSchema(s)); err != nil {
		return recipientAnalysis{}, err
	}
	var a recipientAnalysis
	if err := json.Unmarshal(raw, &a); err != nil {
		return a, err
	}
	if len(a.PositiveInsights)+len(a.CriticalInsights) > s.Recipient.MaxInsights {
		return a, fmt.Errorf("insight limit exceeded")
	}
	return a, nil
}

// Numeric prose is retained for human review, rather than rejected and retried.
func recipientAnalysisWarnings(a recipientAnalysis) []string {
	texts := []string{a.Headline, a.Summary}
	texts = append(texts, a.Limitations...)
	for _, i := range append(append([]recipientInsight{}, a.PositiveInsights...), a.CriticalInsights...) {
		texts = append(texts, i.Title, i.Description)
	}
	for _, rec := range a.Recommendations {
		texts = append(texts, rec.Action, rec.Reason)
	}
	for _, t := range texts {
		if strings.IndexFunc(t, unicode.IsNumber) >= 0 || recipientNumberWords.MatchString(t) || strings.ContainsAny(t, "%€$") {
			return []string{"Il testo contiene riferimenti numerici: confrontali con i KPI prima di approvare l’analisi."}
		}
	}
	return []string{}
}

func callRecipientOpenAI(ctx context.Context, client *http.Client, url, key, model string, s engagementReportSnapshot, options ...recipientAIAttemptOptions) (recipientAnalysis, error) {
	var attempt recipientAIAttemptOptions
	if len(options) > 0 {
		attempt = options[0]
	}
	payload, err := recipientAIRequestPayload(s, model, attempt.RetryHint)
	if err != nil {
		return recipientAnalysis{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return recipientAnalysis{}, err
	}
	request.Header.Set("Authorization", "Bearer "+key)
	request.Header.Set("Content-Type", "application/json")
	if attempt.Usage != nil {
		attempt.Usage.Requests++
		if len(payload) > attempt.Usage.RequestBytes {
			attempt.Usage.RequestBytes = len(payload)
		}
	}
	response, err := client.Do(request)
	if err != nil {
		return recipientAnalysis{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		diagnostic := recipientDiagnostic("provider_http_error")
		diagnostic.HTTPStatus = response.StatusCode
		// Codes are useful for invalid_api_key, insufficient_quota, model_not_found
		// and invalid_json_schema. Raw messages may contain secrets; omit them.
		var body struct {
			Error struct {
				Code string `json:"code"`
				Type string `json:"type"`
			} `json:"error"`
		}
		_ = json.NewDecoder(io.LimitReader(response.Body, 64*1024)).Decode(&body)
		code := body.Error.Code
		if code == "" {
			code = body.Error.Type
		}
		if recipientProviderIdentifier.MatchString(code) {
			diagnostic.ProviderCode = code
		}
		if id := response.Header.Get("x-request-id"); recipientProviderIdentifier.MatchString(id) {
			diagnostic.RequestID = id
		}
		return recipientAnalysis{}, diagnostic
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, 256*1024+1))
	if err != nil {
		return recipientAnalysis{}, recipientErrorDiagnostic(err)
	}
	if len(raw) > 256*1024 {
		return recipientAnalysis{}, recipientDiagnostic("response_too_large")
	}
	var envelope struct {
		Status string `json:"status"`
		Usage  *struct {
			InputTokens        int `json:"input_tokens"`
			OutputTokens       int `json:"output_tokens"`
			TotalTokens        int `json:"total_tokens"`
			InputTokensDetails struct {
				CachedTokens int `json:"cached_tokens"`
			} `json:"input_tokens_details"`
		} `json:"usage"`
		Output []struct {
			Type    string `json:"type"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if json.Unmarshal(raw, &envelope) != nil {
		return recipientAnalysis{}, recipientDiagnostic("invalid_response")
	}
	if attempt.Usage != nil && envelope.Usage != nil {
		usage := envelope.Usage
		attempt.Usage.ReportedRequests++
		attempt.Usage.InputTokens += usage.InputTokens
		attempt.Usage.CachedInputTokens += usage.InputTokensDetails.CachedTokens
		attempt.Usage.OutputTokens += usage.OutputTokens
		attempt.Usage.TotalTokens += usage.TotalTokens
	}
	if envelope.Status != "completed" {
		return recipientAnalysis{}, recipientDiagnostic("response_incomplete")
	}
	text := ""
	for _, output := range envelope.Output {
		for _, part := range output.Content {
			if part.Type == "refusal" {
				return recipientAnalysis{}, recipientDiagnostic("provider_refusal")
			}
			if part.Type == "output_text" {
				text += part.Text
			}
		}
	}
	analysis, err := validateRecipientProviderAnalysis([]byte(text), s)
	if err != nil {
		diagnostic := recipientDiagnostic("invalid_output")
		diagnostic.ValidationError = err.Error()
		return recipientAnalysis{}, diagnostic
	}
	return analysis, nil
}

var recipientProviderIdentifier = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,160}$`)

func (rt *_router) attachCachedRecipientAnalysis(s *engagementReportSnapshot) {
	model := recipientReportModel()
	key := recipientAnalysisKey(*s, model)
	var raw, status string
	err := rt.db.SQLConn().QueryRow(`SELECT response_json,status FROM engagement_report_analysis_cache WHERE cache_key=?`, key).Scan(&raw, &status)
	if err == nil && (status == "ready" || status == "fallback") {
		var a recipientCachedAnalysis
		if json.Unmarshal([]byte(raw), &a) == nil {
			s.Analysis = &a.recipientAnalysis
			s.AnalysisStatus = status
			s.AnalysisDiagnostic = a.Diagnostic
			if status == "fallback" && s.AnalysisDiagnostic == nil {
				s.AnalysisDiagnostic = recipientDiagnostic("cached_fallback")
			}
		}
	}
	if s.Analysis == nil {
		a := recipientFallback(*s)
		s.Analysis = &a
		s.AnalysisStatus = "fallback"
		s.AnalysisDiagnostic = recipientDiagnostic("analysis_not_generated")
		if status == "pending" {
			s.AnalysisDiagnostic = recipientDiagnostic("analysis_pending")
		} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
			s.AnalysisDiagnostic = recipientDiagnostic("cache_unavailable")
		}
	}
	s.AnalysisModel = model
	s.PromptVersion = recipientPromptVersion
}

// A SQLite reservation coalesces identical concurrent requests across server
// instances. Leases expire after a crash; regeneration never steals a live one.
func (rt *_router) generateRecipientReportAnalysis(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, _, ok := rt.reportEventScope(w, r, ctx)
	if !ok {
		return
	}
	var input struct {
		CalculationID   string `json:"calculation_id"`
		Recipient       string `json:"recipient"`
		ComparePrevious bool   `json:"compare_previous"`
		Regenerate      bool   `json:"regenerate"`
	}
	if !decodeReportBody(w, r, 4096, &input) {
		return
	}
	s, err := rt.loadRecipientCalculation(input.CalculationID, id, ctx.AdminID)
	if err != nil {
		reportError(w, 404, "Calcolo non disponibile.")
		return
	}
	s, err = recipientSnapshot(s, input.Recipient, input.ComparePrevious, false)
	if err != nil {
		reportError(w, 400, err.Error())
		return
	}
	reportCtx, cancel := context.WithTimeout(r.Context(), recipientReportTimeout())
	defer cancel()
	model, key := recipientReportModel(), ""
	key = recipientAnalysisKey(s, model)
	previousID := 0
	if s.PreviousEvent != nil {
		previousID = s.PreviousEvent.ID
	}
	for {
		var status, raw, updated string
		err = rt.db.SQLConn().QueryRowContext(reportCtx, `SELECT status,response_json,updated_at FROM engagement_report_analysis_cache WHERE cache_key=?`, key).Scan(&status, &raw, &updated)
		if err != nil && err != sql.ErrNoRows {
			reportError(w, 503, "Cache analisi temporaneamente non disponibile.")
			return
		}
		if err == nil && !input.Regenerate && (status == "ready" || status == "fallback" && time.Since(reportMetricTime(updated)) < 5*time.Minute) {
			rt.writeRecipientAnalysisGeneration(w, s, ctx.AdminID, true)
			return
		}
		now := time.Now().UTC().Format(time.RFC3339Nano)
		res, e := rt.db.SQLConn().ExecContext(reportCtx, `INSERT INTO engagement_report_analysis_cache(cache_key,event_id,recipient_profile,previous_event_id,data_hash,prompt_version,model,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?,'pending',?,?) ON CONFLICT(cache_key) DO UPDATE SET status='pending',updated_at=excluded.updated_at,version=version+1 WHERE updated_at=? AND (status<>'pending' OR updated_at<?)`, key, id, input.Recipient, previousID, s.DataHash, recipientPromptVersion, model, now, now, updated, time.Now().UTC().Add(-3*time.Minute).Format(time.RFC3339Nano))
		if e != nil {
			reportError(w, 503, "Impossibile prenotare l'analisi.")
			return
		}
		count, _ := res.RowsAffected()
		if count > 0 {
			break
		}
		input.Regenerate = false
		select {
		case <-reportCtx.Done():
			a := recipientFallback(s)
			_ = writeJSON(w, 200, map[string]interface{}{"analysis": a, "status": "fallback", "cached": false, "diagnostic": recipientDiagnostic("request_timeout")})
			return
		case <-time.After(150 * time.Millisecond):
		}
	}
	a, status := recipientFallback(s), "fallback"
	var diagnostic *recipientAIDiagnostic
	var usage *recipientAIUsage
	available := 0
	for _, k := range recipientSelectedKPIs(s) {
		if k.Value != nil {
			available++
		}
	}
	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	switch {
	case apiKey == "":
		diagnostic = recipientDiagnostic("missing_api_key")
	case model == "":
		diagnostic = recipientDiagnostic("missing_model")
	case available == 0:
		diagnostic = recipientDiagnostic("no_available_kpis")
	}
	if diagnostic == nil {
		client := &http.Client{Timeout: recipientReportTimeout()}
		usage = &recipientAIUsage{}
		retryHint := ""
		for attempt := 0; attempt < 2; attempt++ {
			generated, e := callRecipientOpenAI(reportCtx, client, recipientOpenAIURL, apiKey, model, s, recipientAIAttemptOptions{RetryHint: retryHint, Usage: usage})
			if e == nil {
				a, status = generated, "ready"
				diagnostic = nil
				break
			}
			diagnostic = recipientErrorDiagnostic(e)
			if diagnostic.Reason == "invalid_output" {
				retryHint = diagnostic.ValidationError
			}
			if ctx.Logger != nil {
				ctx.Logger.WithField("event_id", id).WithField("recipient", input.Recipient).WithField("model", model).WithField("reason", diagnostic.Reason).WithField("http_status", diagnostic.HTTPStatus).WithField("provider_code", diagnostic.ProviderCode).WithField("provider_request_id", diagnostic.RequestID).WithField("validation_error", diagnostic.ValidationError).WithField("attempt", attempt+1).Warn("recipient report OpenAI attempt failed")
			}
			if reportCtx.Err() != nil || diagnostic.Reason != "invalid_output" {
				break
			}
		}
		if ctx.Logger != nil {
			ctx.Logger.WithField("event_id", id).WithField("recipient", input.Recipient).WithField("model", model).WithField("requests", usage.Requests).WithField("reported_requests", usage.ReportedRequests).WithField("input_tokens", usage.InputTokens).WithField("cached_input_tokens", usage.CachedInputTokens).WithField("output_tokens", usage.OutputTokens).WithField("max_request_bytes", usage.RequestBytes).Info("recipient report OpenAI usage")
		}
	}
	if status == "fallback" && ctx.Logger != nil {
		ctx.Logger.WithField("event_id", id).WithField("recipient", input.Recipient).WithField("reason", diagnostic.Reason).Warn("recipient report using deterministic fallback")
	}
	raw, _ := json.Marshal(recipientCachedAnalysis{recipientAnalysis: a, Diagnostic: diagnostic, Usage: usage})
	_, err = rt.db.SQLConn().Exec(`UPDATE engagement_report_analysis_cache SET response_json=?,status=?,updated_at=? WHERE cache_key=?`, string(raw), status, time.Now().UTC().Format(time.RFC3339Nano), key)
	if err != nil {
		reportError(w, 503, "Analisi prodotta ma non salvata. Riprova.")
		return
	}
	rt.writeRecipientAnalysisGeneration(w, s, ctx.AdminID, false)
}

func (rt *_router) writeRecipientAnalysisGeneration(w http.ResponseWriter, s engagementReportSnapshot, adminID int, cached bool) {
	response, err := loadRecipientAnalysisReview(rt.db.SQLConn(), s, adminID)
	if err != nil {
		recipientReviewError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	_ = writeJSON(w, 200, struct {
		recipientAnalysisReviewResponse
		Cached bool `json:"cached"`
	}{response, cached})
}
