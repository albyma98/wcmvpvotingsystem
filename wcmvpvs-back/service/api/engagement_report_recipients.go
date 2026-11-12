package api

import (
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/go-chi/chi/v5"
)

// Profiles reference the existing Go catalog; definitions and denominators have
// a single owner in buildReportKPIs. Slice order is the PDF and UI order.
type reportRecipientProfile struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	KPIIDs         []string `json:"kpi_ids"`
	Groups         []string `json:"groups"`
	Objective      string   `json:"objective"`
	Tone           string   `json:"tone"`
	AIInstructions string   `json:"ai_instructions"`
	Layout         string   `json:"layout"`
	MaxInsights    int      `json:"max_insights"`
	AllowAppendix  bool     `json:"allow_appendix"`
}

func reportOriginalKPIIDs() []string {
	all := buildReportKPIs(nil)
	ids := make([]string, 0, 20)
	for _, k := range all[:20] {
		ids = append(ids, k.ID)
	}
	return ids
}

func reportRecipientProfiles() []reportRecipientProfile {
	return []reportRecipientProfile{
		{"executive", "Dirigenza", "Il valore generato per la società: crescita, acquisizione, ritorno e valore commerciale.", strings.Fields("visitors venue_adoption new_profiles event_retention sponsor_reach"), []string{"Pubblico", "Acquisizione", "Fidelizzazione", "Sponsor"}, "Mostrare il valore per la società sportiva", "Semplice, chiaro, direzionale", "Sintesi breve per un presidente. Massimo tre informazioni importanti e una conclusione strategica. Evita gergo.", "executive", 3, true},
		{"marketing", "Marketing", "Acquisizione, conversione, coinvolgimento e ritorno dei tifosi.", strings.Fields("visitors active_participation new_profiles registration_conversion event_retention active_time mvp_participation multi_experience feedback_response sponsor_reach"), []string{"Pubblico", "Acquisizione", "Engagement", "Fidelizzazione", "Feedback", "Sponsor"}, "Comprendere il comportamento e individuare opportunità", "Analitico e comprensibile", "Analizza acquisizione, conversione, engagement, ritorno e uso funzionalità. Massimo tre raccomandazioni operative. Non attribuire cause senza prove.", "marketing", 3, true},
		{"sponsor", "Sponsor", "Visibilità e interazioni misurabili, aggregate o attribuite a uno sponsor.", strings.Fields("sponsor_reach sponsor_impressions sponsor_ctr sponsored_participation visitors"), []string{"Sponsor", "Pubblico"}, "Dimostrare il valore misurabile delle attivazioni", "Concreto, commerciale e prudente", "Concentrati su copertura, impression, frequenza, clic e giochi sponsorizzati. Distingui visibilità da conversioni commerciali. Non inventare ROI o ricavi. Il pubblico complessivo è contesto evento, non copertura del partner.", "sponsor", 3, true},
		{"analytics", "Analitico / completo", "Tutti i venti indicatori originali con denominatori, metodo e limiti.", reportOriginalKPIIDs(), []string{"Pubblico", "Engagement", "Gaming", "Acquisizione", "Fidelizzazione", "Sponsor", "Feedback"}, "Consentire una lettura tecnica verificabile", "Tecnico e sintetico", "Evidenzia anomalie e limiti senza ripetere tutti i KPI. Non confondere stime e misure.", "analytics", 3, false},
	}
}

func findReportProfile(id string) (reportRecipientProfile, bool) {
	for _, p := range reportRecipientProfiles() {
		if p.ID == id {
			return p, true
		}
	}
	return reportRecipientProfile{}, false
}

type reportAvailability struct {
	Reason        string              `json:"reason"`
	Message       string              `json:"message"`
	ManualAllowed bool                `json:"manual_allowed"`
	Fields        []reportSourceField `json:"fields"`
	Dependencies  []string            `json:"dependencies"`
}
type reportSourceField struct {
	Key        string  `json:"key"`
	Label      string  `json:"label"`
	Type       string  `json:"type"`
	Unit       string  `json:"unit"`
	Min        float64 `json:"min"`
	Max        float64 `json:"max"`
	Validation string  `json:"validation"`
}
type reportComparison struct {
	Previous           *float64             `json:"previous"`
	Absolute           *float64             `json:"absolute"`
	Percent            *float64             `json:"percent"`
	PercentagePoints   *float64             `json:"percentage_points"`
	Direction          string               `json:"direction"`
	Reason             string               `json:"reason,omitempty"`
	PreviousSource     string               `json:"previous_source,omitempty"`
	PreviousProvenance []reportManualSource `json:"previous_provenance,omitempty"`
}

//go:embed report_migrations/001_recipient_reports.sql
var recipientReportMigration string

//go:embed report_migrations/002_manual_kpis.sql
var recipientManualKPIMigration string

//go:embed report_migrations/003_analysis_reviews.sql
var recipientAnalysisReviewMigration string

func (rt *_router) ensureRecipientReportTables() error {
	tx, err := rt.db.SQLConn().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(recipientReportMigration); err != nil {
		return err
	}
	rows, err := tx.Query(`PRAGMA table_info(engagement_report_manual_sources)`)
	if err != nil {
		return err
	}
	migrated := false
	for rows.Next() {
		var cid, notNull, pk int
		var name, kind string
		var defaultValue interface{}
		if err = rows.Scan(&cid, &name, &kind, &notNull, &defaultValue, &pk); err != nil {
			rows.Close()
			return err
		}
		migrated = migrated || name == "sponsor_id"
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if !migrated {
		if _, err = tx.Exec(recipientManualKPIMigration); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(recipientAnalysisReviewMigration); err != nil {
		return err
	}
	return tx.Commit()
}

func reportDigest(value interface{}) string {
	raw, _ := json.Marshal(value)
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:])
}

// The database has is_concluded but no cancellation status. Only explicitly
// concluded events qualify. Dates alone never turn a scheduled event into one.
func (rt *_router) precedingConcludedEvent(event reportEvent, orgID int) (int, error) {
	var id int
	err := rt.db.SQLConn().QueryRow(`SELECT id FROM events WHERE organization_id=? AND is_concluded=1 AND id<>? AND julianday(start_datetime)<julianday(?) ORDER BY julianday(start_datetime) DESC,id DESC LIMIT 1`, orgID, event.ID, event.StartDateTime).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return id, err
}

func (rt *_router) recipientReportEvent(eventID, orgID int) (reportEvent, error) {
	var e reportEvent
	var a, b string
	err := rt.db.SQLConn().QueryRow(`SELECT e.id,e.start_datetime,COALESCE(e.location,''),COALESCE(t1.name,''),COALESCE(t2.name,'') FROM events e LEFT JOIN teams t1 ON t1.id=e.team1_id LEFT JOIN teams t2 ON t2.id=e.team2_id WHERE e.id=? AND e.organization_id=?`, eventID, orgID).Scan(&e.ID, &e.StartDateTime, &e.Location, &a, &b)
	if err != nil {
		return e, err
	}
	org, err := rt.db.GetOrganization(orgID)
	if err != nil {
		return e, err
	}
	e.Title = strings.TrimSpace(a + " - " + b)
	if e.Title == "-" || e.Title == "" {
		e.Title = fmt.Sprintf("Evento #%d", eventID)
	}
	e.OrganizationName, e.OrganizationLogoURL = org.Name, org.LogoURL
	return e, nil
}

func reportSponsorSignals(signals []reportSignal, sponsorID int) []reportSignal {
	out := make([]reportSignal, 0, len(signals))
	for _, s := range signals {
		if int(reportMetricNumber(s.Metadata, "sponsor_id")) == sponsorID {
			out = append(out, s)
		}
	}
	return out
}

const recipientReportSchemaVersion = 3

func (rt *_router) buildRecipientEvent(eventID, orgID, sponsorID, previousID int) (engagementReportSnapshot, []reportSignal, error) {
	s := engagementReportSnapshot{SchemaVersion: recipientReportSchemaVersion, CalculatedAt: time.Now().UTC().Format(time.RFC3339Nano), SponsorID: sponsorID, Profiles: reportRecipientProfiles()}
	var err error
	s.Event, err = rt.recipientReportEvent(eventID, orgID)
	if err != nil {
		return s, nil, err
	}
	s.Branding, err = rt.loadReportBranding()
	if err != nil {
		return s, nil, err
	}
	signals, err := rt.loadReportSignals(eventID)
	if err != nil {
		return s, nil, err
	}
	op, err := rt.loadReportOperationalData(eventID, orgID)
	if err != nil {
		return s, nil, err
	}
	var previous []reportSignal
	var prevOp reportOperationalData
	if previousID > 0 {
		previous, err = rt.loadReportSignals(previousID)
		if err != nil {
			return s, nil, err
		}
		prevOp, err = rt.loadReportOperationalData(previousID, orgID)
		if err != nil {
			return s, nil, err
		}
	}
	s.Facts, s.Details, s.MeasurementStartedAt = calculateCombinedReportMetrics(signals, previous, op, prevOp)
	sponsorSignals, sponsorOp := signals, op
	if sponsorID > 0 {
		sponsorSignals = reportSponsorSignals(signals, sponsorID)
		sponsorOp = reportOperationalData{Signals: reportSponsorSignals(op.Signals, sponsorID)}
		sponsorOp.Summary.SponsorClicks = op.SponsorClicksByID[sponsorID]
		sponsorOp.Summary.SponsorClicksWithoutIdentity = sponsorOp.Summary.SponsorClicks
		for _, signal := range sponsorOp.Signals {
			if signal.Name == "sponsor_click" {
				sponsorOp.Summary.SponsorClicksWithoutIdentity--
			}
		}
		filtered, details, _ := calculateCombinedReportMetrics(sponsorSignals, nil, sponsorOp, reportOperationalData{})
		for _, k := range buildReportKPIs(nil) {
			if k.Group != "Sponsor" {
				continue
			}
			for _, key := range []string{k.FactKey, k.NumeratorKey, k.DenominatorKey} {
				if key != "" {
					s.Facts[key] = filtered[key]
				}
			}
		}
		s.Details.Sponsors = details.Sponsors
		s.Details.Operational.SponsorClicks = sponsorOp.Summary.SponsorClicks
		s.Details.Operational.SponsorClicksWithoutIdentity = sponsorOp.Summary.SponsorClicksWithoutIdentity
		s.Details.CoverageNotes = append(s.Details.CoverageNotes, "Le metriche sponsor si riferiscono al partner selezionato. I partecipanti digitali rappresentano il contesto complessivo dell'evento.")
	}
	// A count of zero needs evidence that exposure measurement existed. A
	// click-only legacy record cannot establish that nobody saw the partner.
	hasExposure, hasSponsoredExposure := false, false
	for _, signal := range sponsorSignals {
		hasExposure = hasExposure || signal.Name == "sponsor_view" || signal.Name == "sponsored_game_view"
		hasSponsoredExposure = hasSponsoredExposure || signal.Name == "sponsored_game_view"
	}
	if !hasExposure {
		for _, key := range []string{"sponsor_reach", "sponsor_impressions", "sponsor_clickers", "sponsor_engaged_exposed", "sponsor_engagement_reach"} {
			s.Facts[key] = nil
		}
	}
	if !hasSponsoredExposure {
		s.Facts["sponsored_players"], s.Facts["sponsored_exposed_visitors"] = nil, nil
	}
	measured, measuredSponsors, err := rt.loadReportMeasuredSponsors(eventID, orgID, sponsorID)
	if err != nil {
		return s, nil, err
	}
	measuredNotes := measured.apply(s.Facts, &s.Details, measuredSponsors)
	s.ManualSources, err = rt.loadReportManualSources(eventID, true)
	if err != nil {
		return s, nil, err
	}
	s.Details.Sources = append(s.Details.Sources, reportSourceRows{Table: "engagement_report_manual_sources", Rows: len(s.ManualSources)})
	automatic := s.Facts["attendees"]
	s.OriginalFacts = map[string]*float64{}
	for key, value := range s.Facts {
		s.OriginalFacts[key] = value
	}
	s.Conflicts = resolveReportManualSources(s.Facts, s.ManualSources)
	s.KPIs = buildCombinedReportKPIs(s.Facts, s.Details, signals, previous, op, prevOp)
	if sponsorID > 0 {
		scopedKPIs := buildCombinedReportKPIs(s.Facts, s.Details, sponsorSignals, nil, sponsorOp, reportOperationalData{})
		for i := range scopedKPIs {
			if scopedKPIs[i].Group == "Sponsor" {
				s.KPIs[i] = scopedKPIs[i]
			}
		}
	}
	for i := range s.KPIs {
		k := &s.KPIs[i]
		if note, measured := measuredNotes[k.ID]; measured {
			// Comparison must match the measurement definition as well as the units.
			k.Description = note
			k.Note = note
			if k.Value != nil {
				k.Source = "operational"
			}
			if k.ID == "sponsor_impressions" {
				k.DisplayDescription = "Esposizioni HTML qualificate registrate per la partita."
			}
		}
		if k.ID == "venue_adoption" && automatic == nil {
			for _, m := range s.ManualSources {
				if m.FieldKey == "attendees" {
					k.Provenance = append(k.Provenance, m)
				}
			}
			if k.Value != nil && len(k.Provenance) > 0 {
				k.Source = "manual_" + k.Provenance[0].VerificationStatus
				k.Note = "Presenze da fonte manuale; numeratore limitato alla provenienza arena esplicita. Tale provenienza non certifica la presenza fisica."
			}
		}
		k.Availability = recipientKPIAvailability(*k, s.Facts, signals, previousID, s.Event)
		if k.Availability.Reason == "incompatible_population" {
			k.Value = nil
			k.Source = "unavailable"
		}
	}
	return s, signals, nil
}

func resolveReportManualSources(facts map[string]*float64, sources []reportManualSource) []string {
	conflicts := []string{}
	for _, m := range sources {
		if m.FieldKey != "attendees" {
			continue
		}
		if automatic := facts[m.FieldKey]; automatic != nil {
			if *automatic != m.Value {
				conflicts = append(conflicts, "Presenze: il dato automatico differisce dalla sorgente manuale. Il dato automatico resta prioritario; verificare la fonte.")
			}
		} else {
			facts[m.FieldKey] = reportMetricFloat(m.Value)
		}
	}
	return conflicts
}

func recipientKPIAvailability(k reportKPI, facts map[string]*float64, signals []reportSignal, previousID int, event reportEvent) *reportAvailability {
	a := &reportAvailability{Reason: "available", Message: "Dato disponibile.", Fields: []reportSourceField{}, Dependencies: []string{}}
	for _, key := range []string{k.FactKey, k.NumeratorKey, k.DenominatorKey} {
		if key != "" {
			a.Dependencies = append(a.Dependencies, key)
		}
	}
	if k.ID == "venue_adoption" && k.Numerator != nil && k.Denominator != nil && *k.Denominator > 0 && *k.Numerator > *k.Denominator {
		a.ManualAllowed = true
		a.Reason, a.Message = "incompatible_population", "Il pubblico con provenienza arena supera le presenze: popolazioni incompatibili, verificare la fonte."
		return a
	}
	if k.Value != nil {
		return a
	}
	a.Reason, a.Message = "tracking_missing", "Dati della partita insufficienti per questo indicatore."
	a.ManualAllowed = k.ID == "venue_adoption"
	if k.ID == "event_retention" && previousID == 0 {
		a.Reason, a.Message = "not_applicable", "Nessuna partita precedente conclusa della stessa società."
		return a
	}
	if k.ID == "venue_adoption" && facts["attendees"] == nil {
		a.Reason, a.Message, a.ManualAllowed = "manual_source_missing", "Per calcolare questo indicatore mancano le presenze ufficiali. Il numeratore richiede provenienza arena esplicita.", true
		a.Fields = append(a.Fields, reportSourceField{"attendees", "Numero spettatori ufficiali", "integer", "people", 0, 10000000, "Intero, fonte obbligatoria, stessa partita e popolazione; revisione contro modifiche concorrenti."})
		if facts["onsite_visitors"] == nil {
			a.Message += " Manca anche il tracciamento arena: le presenze da sole non rendono calcolabile il KPI."
		}
		return a
	}
	flag := ""
	switch k.ID {
	case "mvp_participation":
		flag = "vote_available"
	case "game_participation":
		flag = "games_available"
	case "feedback_response":
		flag = "feedback_available"
	}
	observed, enabled := false, false
	for _, s := range signals {
		if v, ok := s.Metadata[flag].(bool); ok {
			observed = true
			enabled = enabled || v
		}
	}
	if flag != "" && observed && !enabled {
		a.Reason, a.Message = "feature_unavailable", "La funzionalità non risultava disponibile nelle osservazioni dell'evento."
		return a
	}
	if k.Denominator != nil && *k.Denominator == 0 {
		a.Reason, a.Message = "zero_denominator", "Non calcolabile: il denominatore è zero."
		return a
	}
	// No speculative promise of delayed data: pending sync requires evidence.
	// A scheduled start in the future is evidence that collection is incomplete.
	if event.StartDateTime != "" && time.Now().Before(reportMetricTime(event.StartDateTime)) {
		a.Reason, a.Message = "not_synchronized", "Evento non ancora iniziato: raccolta e sincronizzazione non completate."
	}
	return a
}

func compareRecipientKPI(current, previous reportKPI, exists bool) *reportComparison {
	c := &reportComparison{Direction: "unavailable", Previous: previous.Value, PreviousSource: previous.Source, PreviousProvenance: previous.Provenance}
	if !exists {
		c.Reason = "Nessuna partita precedente conclusa."
		return c
	}
	if current.Value == nil || previous.Value == nil {
		c.Reason = "Valore attuale o precedente N/D."
		return c
	}
	if current.Unit != previous.Unit || current.NumeratorKey != previous.NumeratorKey || current.DenominatorKey != previous.DenominatorKey || current.Description != previous.Description {
		c.Reason = "Definizioni o popolazioni non omogenee."
		return c
	}
	d := *current.Value - *previous.Value
	c.Absolute, c.Direction = reportMetricFloat(d), "unchanged"
	if d > 0 {
		c.Direction = "up"
	} else if d < 0 {
		c.Direction = "down"
	}
	if current.Unit == "percent" {
		c.PercentagePoints = reportMetricFloat(d)
	} else if current.Unit == "count" && *previous.Value > 0 {
		c.Percent = reportMetricFloat(d / *previous.Value * 100)
	}
	if current.Unit == "count" && *previous.Value == 0 {
		c.Reason = "Variazione percentuale non calcolabile: base precedente zero."
	}
	return c
}

func (rt *_router) buildRecipientReport(eventID, orgID, sponsorID int) (engagementReportSnapshot, error) {
	e, err := rt.recipientReportEvent(eventID, orgID)
	if err != nil {
		return engagementReportSnapshot{}, err
	}
	prevID, err := rt.precedingConcludedEvent(e, orgID)
	if err != nil {
		return engagementReportSnapshot{}, err
	}
	s, _, err := rt.buildRecipientEvent(eventID, orgID, sponsorID, prevID)
	if err != nil {
		return s, err
	}
	var p engagementReportSnapshot
	if prevID > 0 {
		pe, e := rt.recipientReportEvent(prevID, orgID)
		if e != nil {
			return s, e
		}
		priorID, e := rt.precedingConcludedEvent(pe, orgID)
		if e != nil {
			return s, e
		}
		p, _, err = rt.buildRecipientEvent(prevID, orgID, sponsorID, priorID)
		if err != nil {
			return s, err
		}
		s.PreviousEvent = &p.Event
	}
	previous := map[string]reportKPI{}
	for _, k := range p.KPIs {
		previous[k.ID] = k
	}
	for i := range s.KPIs {
		s.KPIs[i].Comparison = compareRecipientKPI(s.KPIs[i], previous[s.KPIs[i].ID], prevID > 0)
	}
	if sponsorID > 0 {
		err = rt.db.SQLConn().QueryRow(`SELECT COALESCE(NULLIF(report_name,''),name) FROM sponsors WHERE id=? AND organization_id=?`, sponsorID, orgID).Scan(&s.SponsorName)
		if err != nil {
			return s, err
		}
	}
	// Hash excludes volatile calculation IDs/time. It includes previous values,
	// provenance revisions, denominators, limitations, definitions and branding.
	s.DataHash = reportDigest(struct {
		Event       reportEvent
		Previous    *reportEvent
		Facts       map[string]*float64
		KPIs        []reportKPI
		Details     reportDetails
		Sources     []reportManualSource
		Conflicts   []string
		SponsorID   int
		SponsorName string
		Branding    reportBranding
	}{s.Event, s.PreviousEvent, s.Facts, s.KPIs, s.Details, s.ManualSources, s.Conflicts, s.SponsorID, s.SponsorName, s.Branding})
	return s, nil
}

func (rt *_router) prepareRecipientReport(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, org, ok := rt.reportEventScope(w, r, ctx)
	if !ok {
		return
	}
	profile, valid := findReportProfile(r.URL.Query().Get("recipient"))
	sponsorID := 0
	if raw := r.URL.Query().Get("sponsor_id"); raw != "" {
		var err error
		sponsorID, err = strconv.Atoi(raw)
		if err != nil || sponsorID < 0 || sponsorID > 0 && profile.ID != "sponsor" {
			reportError(w, 400, "Sponsor non valido.")
			return
		}
	}
	if !valid {
		reportError(w, 400, "Destinatario non valido.")
		return
	}
	if sponsorID > 0 {
		var found int
		if rt.db.SQLConn().QueryRow(`SELECT id FROM sponsors WHERE id=? AND organization_id=?`, sponsorID, org).Scan(&found) != nil {
			reportError(w, 404, "Sponsor non disponibile per questa società.")
			return
		}
	}
	s, err := rt.buildRecipientReport(id, org, sponsorID)
	if err != nil {
		reportError(w, 500, "Impossibile calcolare il report destinatario.")
		return
	}
	s.CalculationID = newReportID()
	s.SelectedKPIIDs = profile.KPIIDs
	if profile.ID == "sponsor" {
		s.CommercialSponsors = rt.measuredSponsorReportForEvent(r, ctx, org, id, sponsorID)
	}
	raw, err := json.Marshal(s)
	if err == nil {
		_, err = rt.db.SQLConn().Exec(`INSERT INTO engagement_report_calculations(id,event_id,admin_id,payload,created_at) VALUES(?,?,?,?,?)`, s.CalculationID, id, ctx.AdminID, string(raw), s.CalculatedAt)
	}
	if err != nil {
		reportError(w, 500, "Impossibile salvare il calcolo.")
		return
	}
	_, _ = rt.db.SQLConn().Exec(`DELETE FROM engagement_report_calculations WHERE created_at < ? AND id NOT IN (SELECT calculation_id FROM engagement_report_snapshots)`, time.Now().UTC().Add(-7*24*time.Hour).Format(time.RFC3339Nano))
	w.Header().Set("Cache-Control", "no-store")
	_ = writeJSON(w, 200, s)
}

func (rt *_router) loadRecipientCalculation(id string, eventID, adminID int) (engagementReportSnapshot, error) {
	var raw string
	var s engagementReportSnapshot
	err := rt.db.SQLConn().QueryRow(`SELECT payload FROM engagement_report_calculations WHERE id=? AND event_id=? AND admin_id=?`, id, eventID, adminID).Scan(&raw)
	if err == nil {
		err = json.Unmarshal([]byte(raw), &s)
	}
	if err == nil && s.SchemaVersion != recipientReportSchemaVersion {
		err = fmt.Errorf("calcolo destinatario richiesto")
	}
	return s, err
}

func recipientSnapshot(s engagementReportSnapshot, profileID string, compare, appendix bool) (engagementReportSnapshot, error) {
	p, ok := findReportProfile(profileID)
	if !ok || s.SponsorID > 0 && profileID != "sponsor" {
		return s, fmt.Errorf("Destinatario non valido per questo calcolo.")
	}
	s.Recipient = &p
	s.SelectedKPIIDs = p.KPIIDs
	s.ComparePrevious = compare
	s.IncludeAppendix = appendix && p.AllowAppendix
	s.Format = "one_page"
	if p.ID == "analytics" || p.ID == "marketing" {
		s.Format = "extended"
	}
	if !compare {
		for i := range s.KPIs {
			s.KPIs[i].Comparison = nil
		}
		s.PreviousEvent = nil
	}
	return s, nil
}

func (rt *_router) approveRecipientReport(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext, eventID int, calculationID, profileID string, compare, appendix, includeAI bool, notes string, reviewRevision *int) {
	if utf8.RuneCountInString(notes) > 4000 {
		reportError(w, 400, "Note troppo lunghe.")
		return
	}
	s, err := rt.loadRecipientCalculation(calculationID, eventID, ctx.AdminID)
	if err != nil {
		reportError(w, 404, "Calcolo non disponibile. Riapri il report.")
		return
	}
	s, err = recipientSnapshot(s, profileID, compare, appendix)
	if err != nil {
		reportError(w, 400, err.Error())
		return
	}
	// Frozen draft approval is intentional: edits after calculation create a new
	// version rather than rewriting the data used by an existing PDF.
	tx, err := rt.db.SQLConn().Begin()
	if err != nil {
		reportError(w, 503, "Impossibile preparare la versione del report.")
		return
	}
	defer tx.Rollback()
	s.Notes = strings.TrimSpace(notes)
	if includeAI {
		review, err := loadRecipientAnalysisReview(tx, s, ctx.AdminID)
		if err != nil {
			recipientReviewError(w, err)
			return
		}
		if review.Review.State != "approved" || reviewRevision == nil || *reviewRevision != review.Review.Revision {
			reportError(w, 409, "Approva la versione corrente dell’analisi prima di includerla nel PDF, oppure deseleziona Includi analisi / sintesi.")
			return
		}
		s.Analysis, s.AnalysisReview = &review.Analysis, &review.Review
		s.AnalysisStatus, s.AnalysisModel, s.PromptVersion = review.Status, review.Model, review.PromptVersion
		s.AnalysisDiagnostic = review.Diagnostic
	}
	key := reportDigest(struct {
		Calculation string
		Profile     string
		Compare     bool
		Appendix    bool
		AI          *recipientAnalysis
		Notes       string
		AIStatus    string
		Model       string
		Prompt      string
		Diagnostic  *recipientAIDiagnostic
		Review      *recipientAnalysisReview
	}{calculationID, profileID, compare, s.IncludeAppendix, s.Analysis, s.Notes, s.AnalysisStatus, s.AnalysisModel, s.PromptVersion, s.AnalysisDiagnostic, s.AnalysisReview})
	s.ID = newReportID()
	s.ApprovedAt = time.Now().UTC().Format(time.RFC3339Nano)
	s.ApprovedBy = ctx.AdminUsername
	raw, _ := json.Marshal(s)
	_, err = tx.Exec(`INSERT INTO engagement_report_snapshots(id,calculation_id,approval_key,event_id,admin_id,payload,approved_at) VALUES(?,?,?,?,?,?,?) ON CONFLICT(approval_key) DO NOTHING`, s.ID, s.CalculationID, key, eventID, ctx.AdminID, string(raw), s.ApprovedAt)
	if err != nil {
		reportError(w, 500, "Impossibile salvare lo snapshot.")
		return
	}
	var saved string
	if tx.QueryRow(`SELECT payload FROM engagement_report_snapshots WHERE approval_key=? AND event_id=?`, key, eventID).Scan(&saved) != nil {
		reportError(w, 500, "Impossibile caricare lo snapshot.")
		return
	}
	if err := tx.Commit(); err != nil {
		reportError(w, 503, "Impossibile salvare la versione approvata del report.")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(saved))
}

func (rt *_router) listRecipientReportSnapshots(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, _, ok := rt.reportEventScope(w, r, ctx)
	if !ok {
		return
	}
	before := r.URL.Query().Get("before")
	if before == "" {
		before = "9999"
	}
	rows, err := rt.db.SQLConn().Query(`SELECT payload FROM engagement_report_snapshots WHERE event_id=? AND approved_at<? ORDER BY approved_at DESC LIMIT 50`, id, before)
	if err != nil {
		reportError(w, 500, "Archivio non disponibile.")
		return
	}
	defer rows.Close()
	type version struct {
		ID         string `json:"id"`
		ApprovedAt string `json:"approved_at"`
		Recipient  string `json:"recipient"`
		Sponsor    string `json:"sponsor"`
	}
	versions := []version{}
	for rows.Next() {
		var raw string
		var s engagementReportSnapshot
		if rows.Scan(&raw) != nil || json.Unmarshal([]byte(raw), &s) != nil {
			reportError(w, 500, "Versione archiviata non leggibile.")
			return
		}
		name := "Generatore originale"
		if s.Recipient != nil {
			name = s.Recipient.Name
		}
		versions = append(versions, version{s.ID, s.ApprovedAt, name, s.SponsorName})
	}
	if rows.Err() != nil {
		reportError(w, 500, "Archivio non disponibile.")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	_ = writeJSON(w, 200, versions)
}

func (rt *_router) getRecipientReportSnapshot(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, _, ok := rt.reportEventScope(w, r, ctx)
	if !ok {
		return
	}
	var raw string
	err := rt.db.SQLConn().QueryRow(`SELECT payload FROM engagement_report_snapshots WHERE id=? AND event_id=?`, chi.URLParam(r, "reportId"), id).Scan(&raw)
	if err == sql.ErrNoRows {
		reportError(w, 404, "Versione non disponibile per questo evento.")
		return
	}
	if err != nil {
		reportError(w, 500, "Versione non disponibile.")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(raw))
}
