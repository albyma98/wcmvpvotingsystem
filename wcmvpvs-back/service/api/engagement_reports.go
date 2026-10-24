package api

// The report combines /new observations with raw, event-scoped operational
// evidence. Event-history aggregates never substitute for missing observations.
import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/go-chi/chi/v5"
)

type reportSignal struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	SessionID  string                 `json:"session_id"`
	OccurredAt string                 `json:"occurred_at"`
	DeviceID   string                 `json:"-"`
	FanID      int                    `json:"-"`
	Metadata   map[string]interface{} `json:"metadata"`
}

type reportBranding struct {
	CompanyName string `json:"company_name"`
	LogoData    string `json:"logo_data"`
}

type reportEvent struct {
	ID                  int    `json:"id"`
	Title               string `json:"title"`
	StartDateTime       string `json:"start_datetime"`
	Location            string `json:"location"`
	OrganizationName    string `json:"organization_name"`
	OrganizationLogoURL string `json:"organization_logo_url"`
}

type reportOverride struct {
	Value *float64 `json:"value"`
	Note  string   `json:"note"`
}

type engagementReportSnapshot struct {
	ID                   string                    `json:"id,omitempty"`
	CalculationID        string                    `json:"calculation_id"`
	Event                reportEvent               `json:"event"`
	Branding             reportBranding            `json:"branding"`
	CalculatedAt         string                    `json:"calculated_at"`
	MeasurementStartedAt string                    `json:"measurement_started_at,omitempty"`
	ApprovedAt           string                    `json:"approved_at,omitempty"`
	ApprovedBy           string                    `json:"approved_by,omitempty"`
	Format               string                    `json:"format,omitempty"`
	Facts                map[string]*float64       `json:"facts"`
	KPIs                 []reportKPI               `json:"kpis"`
	SelectedKPIIDs       []string                  `json:"selected_kpi_ids,omitempty"`
	Details              reportDetails             `json:"details"`
	OriginalFacts        map[string]*float64       `json:"original_facts,omitempty"`
	Overrides            map[string]reportOverride `json:"overrides,omitempty"`
	Notes                string                    `json:"notes,omitempty"`
	LatestReport         *engagementReportSnapshot `json:"latest_report,omitempty"`
	SchemaVersion        int                       `json:"schema_version"`
}

func (rt *_router) ensureEngagementReportTables() error {
	if rt.db.SQLConn() == nil {
		return fmt.Errorf("report storage is unavailable")
	}
	for _, query := range []string{
		`CREATE TABLE IF NOT EXISTS engagement_report_signals (event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE, signal_id TEXT NOT NULL, name TEXT NOT NULL, session_id TEXT NOT NULL, device_key TEXT NOT NULL, fan_id INTEGER NOT NULL DEFAULT 0, occurred_at TEXT NOT NULL, metadata TEXT NOT NULL, received_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP, PRIMARY KEY(event_id, signal_id))`,
		`CREATE INDEX IF NOT EXISTS engagement_report_signals_event ON engagement_report_signals(event_id, occurred_at)`,
		`CREATE TABLE IF NOT EXISTS engagement_report_branding (id INTEGER PRIMARY KEY CHECK(id=1), company_name TEXT NOT NULL, logo_data TEXT NOT NULL DEFAULT '')`,
		`CREATE TABLE IF NOT EXISTS engagement_report_calculations (id TEXT PRIMARY KEY, event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE, admin_id INTEGER NOT NULL, payload TEXT NOT NULL, created_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS engagement_report_snapshots (id TEXT PRIMARY KEY, calculation_id TEXT NOT NULL, approval_key TEXT NOT NULL UNIQUE, event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE, admin_id INTEGER NOT NULL, payload TEXT NOT NULL, approved_at TEXT NOT NULL)`,
	} {
		if _, err := rt.db.SQLConn().Exec(query); err != nil {
			return err
		}
	}
	return nil
}

func registerEngagementReportRoutes(rt *_router) {
	if err := rt.ensureEngagementReportTables(); err != nil {
		rt.baseLogger.WithError(err).Error("cannot initialize engagement reports")
	}
	rt.router.Post("/events/{eventId}/report-signals", rt.wrap(rt.recordReportSignals))
	rt.router.Get("/admin/events/{eventId}/engagement-report", rt.wrapAdmin(rt.prepareEngagementReport))
	rt.router.Post("/admin/events/{eventId}/engagement-report", rt.wrapAdmin(rt.approveEngagementReport))
	rt.router.Get("/admin/master/report-branding", rt.wrapAdmin(rt.getReportBranding))
	rt.router.Put("/admin/master/report-branding", rt.wrapAdmin(rt.putReportBranding))
}

func reportError(w http.ResponseWriter, status int, message string) {
	_ = writeJSONMessage(w, status, message)
}

func decodeReportBody(w http.ResponseWriter, r *http.Request, limit int64, target interface{}) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, limit))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		reportError(w, http.StatusBadRequest, "Richiesta non valida o troppo grande.")
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		reportError(w, http.StatusBadRequest, "Invia un solo oggetto JSON.")
		return false
	}
	return true
}

func (rt *_router) reportEventScope(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) (int, int, bool) {
	id, err := strconv.Atoi(chi.URLParam(r, "eventId"))
	if err != nil || id <= 0 {
		reportError(w, http.StatusBadRequest, "Evento non valido.")
		return 0, 0, false
	}
	org, err := rt.db.GetEventOrganizationID(id)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && (ctx.OrganizationID <= 0 || org != ctx.OrganizationID)) {
		reportError(w, http.StatusNotFound, "Evento non disponibile per questa società.")
		return 0, 0, false
	}
	if err != nil {
		reportError(w, http.StatusInternalServerError, "Impossibile caricare l'evento.")
		return 0, 0, false
	}
	return id, org, true
}

var reportSignalNames = map[string]bool{"page_view": true, "active_time": true, "vote_completed": true, "game_started": true, "game_completed": true, "registration_started": true, "registration_completed": true, "redemption_completed": true, "sponsor_view": true, "sponsor_click": true, "sponsored_game_view": true, "shop_view": true, "feedback_completed": true}

func cleanReportMetadata(input map[string]interface{}) map[string]interface{} {
	out := map[string]interface{}{}
	for _, key := range []string{"game_id", "round_id", "name", "placement", "item_key", "experience", "perks_interest", "source", "started_at", "ended_at"} {
		if value, ok := input[key].(string); ok && utf8.ValidString(value) {
			out[key] = string([]rune(value)[:minReportInt(len([]rune(value)), 160)])
		}
	}
	for _, key := range []string{"vote_available", "games_available", "feedback_available"} {
		if value, ok := input[key].(bool); ok {
			out[key] = value
		}
	}
	if value, ok := input["sponsor_id"].(float64); ok && value > 0 && value <= 1e12 && value == math.Trunc(value) {
		out["sponsor_id"] = value
	}
	if value, ok := input["seconds"].(float64); ok && value >= 0 && value <= 15 {
		out["seconds"] = value
	}
	return out
}

func minReportInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (rt *_router) recordReportSignals(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, orgID, ok := rt.reportEventScope(w, r, ctx)
	if !ok {
		return
	}
	deviceID := rt.deviceIDFromRequest(r)
	if deviceID == "" || len(deviceID) > 160 {
		reportError(w, http.StatusBadRequest, "Identità del browser assente o non valida.")
		return
	}
	var input struct {
		Events          []reportSignal `json:"events"`
		FanSessionToken string         `json:"fan_session_token,omitempty"`
	}
	if !decodeReportBody(w, r, 64*1024, &input) {
		return
	}
	if len(input.Events) == 0 || len(input.Events) > 20 {
		reportError(w, http.StatusBadRequest, "Invia da 1 a 20 misurazioni.")
		return
	}
	fanID := 0
	token := rt.fanSessionTokenFromRequest(r)
	if token == "" {
		token = input.FanSessionToken
	}
	if token != "" {
		// Measurement must not adopt wallets or alter a session in another club.
		_ = rt.db.SQLConn().QueryRow(`SELECT p.id FROM fan_sessions s JOIN fan_profiles p ON p.id=s.fan_id WHERE s.token=? AND p.organization_id=? AND p.registration_pending=0`, token, orgID).Scan(&fanID)
	}
	deviceHash := sha256.Sum256([]byte(strconv.Itoa(orgID) + ":" + deviceID))
	deviceKey := hex.EncodeToString(deviceHash[:])
	now := time.Now().UTC()
	for i := range input.Events {
		s := &input.Events[i]
		at, err := time.Parse(time.RFC3339Nano, s.OccurredAt)
		if !reportSignalNames[s.Name] || len(s.ID) == 0 || len(s.ID) > 100 || len(s.SessionID) == 0 || len(s.SessionID) > 100 || err != nil || at.After(now.Add(5*time.Minute)) || at.Before(now.Add(-30*24*time.Hour)) {
			reportError(w, http.StatusBadRequest, "Misurazione non valida.")
			return
		}
		s.OccurredAt = at.UTC().Format(time.RFC3339Nano)
		s.Metadata = cleanReportMetadata(s.Metadata)
		if s.Name == "game_started" || s.Name == "game_completed" {
			game, _ := s.Metadata["game_id"].(string)
			round, _ := s.Metadata["round_id"].(string)
			validGames := map[string]bool{"reaction": true, "quiz": true, "basketball": true, "playmaker": true, "crossover": true, "tap": true, "sponsor-rush": true, "memory-flash": true}
			if !validGames[game] || strings.TrimSpace(round) == "" {
				reportError(w, 400, "Round di gioco non valido.")
				return
			}
		}
		if s.Name == "sponsor_view" || s.Name == "sponsor_click" || s.Name == "sponsored_game_view" {
			if _, exists := s.Metadata["sponsor_id"]; !exists {
				reportError(w, 400, "Sponsor non valido.")
				return
			}
		}
		if s.Name == "registration_completed" && fanID == 0 {
			reportError(w, http.StatusBadRequest, "Registrazione non verificata.")
			return
		}
		if sponsor, ok := s.Metadata["sponsor_id"].(float64); ok {
			var sponsorName string
			var sponsorURL string
			if err := rt.db.SQLConn().QueryRow(`SELECT COALESCE(NULLIF(report_name,''),name),COALESCE(link_url,'') FROM sponsors WHERE id=? AND organization_id=?`, int64(sponsor), orgID).Scan(&sponsorName, &sponsorURL); err != nil {
				reportError(w, http.StatusBadRequest, "Sponsor non disponibile.")
				return
			}
			if s.Name == "sponsor_click" && !strings.HasPrefix(strings.ToLower(sponsorURL), "http://") && !strings.HasPrefix(strings.ToLower(sponsorURL), "https://") {
				reportError(w, 400, "Collegamento sponsor non valido.")
				return
			}
			s.Metadata["name"] = sponsorName
		}
	}
	tx, err := rt.db.SQLConn().Begin()
	if err != nil {
		reportError(w, 500, "Impossibile salvare le misurazioni.")
		return
	}
	defer tx.Rollback()
	for _, s := range input.Events {
		metadata, _ := json.Marshal(s.Metadata)
		_, err = tx.Exec(`INSERT INTO engagement_report_signals(event_id,signal_id,name,session_id,device_key,fan_id,occurred_at,metadata) VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(event_id,signal_id) DO NOTHING`, eventID, s.ID, s.Name, s.SessionID, deviceKey, fanID, s.OccurredAt, string(metadata))
		if err != nil {
			reportError(w, 500, "Impossibile salvare le misurazioni.")
			return
		}
	}
	if err = tx.Commit(); err != nil {
		reportError(w, 500, "Impossibile salvare le misurazioni.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) loadReportBranding() (reportBranding, error) {
	out := reportBranding{CompanyName: "ArenaBoostX"}
	err := rt.db.SQLConn().QueryRow(`SELECT company_name,logo_data FROM engagement_report_branding WHERE id=1`).Scan(&out.CompanyName, &out.LogoData)
	if err == sql.ErrNoRows {
		return out, nil
	}
	return out, err
}

func (rt *_router) getReportBranding(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !rt.ensureSuperAdmin(w, ctx) {
		return
	}
	value, err := rt.loadReportBranding()
	if err != nil {
		reportError(w, 500, "Impossibile caricare il marchio.")
		return
	}
	_ = writeJSON(w, 200, value)
}

func validateReportBranding(value reportBranding) error {
	if utf8.RuneCountInString(value.CompanyName) < 1 || utf8.RuneCountInString(value.CompanyName) > 120 || strings.ContainsAny(value.CompanyName, "\r\n\x00") {
		return fmt.Errorf("Il nome deve contenere da 1 a 120 caratteri su una riga.")
	}
	if value.LogoData == "" {
		return nil
	}
	if len(value.LogoData) > 512*1024 {
		return fmt.Errorf("Il logo deve occupare al massimo 512 KB.")
	}
	prefix := "data:image/png;base64,"
	format := "png"
	if strings.HasPrefix(value.LogoData, "data:image/jpeg;base64,") {
		prefix = "data:image/jpeg;base64,"
		format = "jpeg"
	}
	if !strings.HasPrefix(value.LogoData, prefix) {
		return fmt.Errorf("Carica un logo PNG o JPEG.")
	}
	data, err := base64.StdEncoding.Strict().DecodeString(strings.TrimPrefix(value.LogoData, prefix))
	if err != nil {
		return fmt.Errorf("Logo non valido.")
	}
	config, kind, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || kind != format || config.Width < 1 || config.Height < 1 || config.Width > 2048 || config.Height > 2048 {
		return fmt.Errorf("Il logo deve essere un'immagine valida di massimo 2048 pixel per lato.")
	}
	if _, _, err = image.Decode(bytes.NewReader(data)); err != nil {
		return fmt.Errorf("Il file immagine è danneggiato.")
	}
	return nil
}

func (rt *_router) putReportBranding(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !rt.ensureSuperAdmin(w, ctx) {
		return
	}
	var input reportBranding
	if !decodeReportBody(w, r, 540*1024, &input) {
		return
	}
	input.CompanyName = strings.TrimSpace(input.CompanyName)
	if err := validateReportBranding(input); err != nil {
		reportError(w, 400, err.Error())
		return
	}
	_, err := rt.db.SQLConn().Exec(`INSERT INTO engagement_report_branding(id,company_name,logo_data) VALUES(1,?,?) ON CONFLICT(id) DO UPDATE SET company_name=excluded.company_name,logo_data=excluded.logo_data`, input.CompanyName, input.LogoData)
	if err != nil {
		reportError(w, 500, "Impossibile salvare il marchio.")
		return
	}
	_ = writeJSON(w, 200, input)
}

func (rt *_router) loadReportSignals(eventID int) ([]reportSignal, error) {
	rows, err := rt.db.SQLConn().Query(`SELECT signal_id,name,session_id,device_key,fan_id,occurred_at,metadata FROM engagement_report_signals WHERE event_id=? ORDER BY occurred_at,signal_id`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []reportSignal{}
	for rows.Next() {
		var s reportSignal
		var raw string
		if err = rows.Scan(&s.ID, &s.Name, &s.SessionID, &s.DeviceID, &s.FanID, &s.OccurredAt, &raw); err != nil {
			return nil, err
		}
		if err = json.Unmarshal([]byte(raw), &s.Metadata); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func newReportID() string {
	var random [18]byte
	if _, err := rand.Read(random[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(random[:])
}

func (rt *_router) prepareEngagementReport(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, orgID, ok := rt.reportEventScope(w, r, ctx)
	if !ok {
		return
	}
	var event reportEvent
	err := rt.db.SQLConn().QueryRow(`SELECT e.id,e.start_datetime,COALESCE(e.location,''),COALESCE(t1.name,''),COALESCE(t2.name,'') FROM events e LEFT JOIN teams t1 ON t1.id=e.team1_id LEFT JOIN teams t2 ON t2.id=e.team2_id WHERE e.id=? AND e.organization_id=?`, eventID, orgID).Scan(&event.ID, &event.StartDateTime, &event.Location, &event.Title, &event.OrganizationName)
	if err != nil {
		reportError(w, 500, "Impossibile caricare l'evento.")
		return
	}
	brand, err := rt.loadReportBranding()
	if err != nil {
		reportError(w, 500, "Impossibile caricare il marchio.")
		return
	}
	signals, err := rt.loadReportSignals(eventID)
	if err != nil {
		reportError(w, 500, "Impossibile calcolare le misurazioni.")
		return
	}
	operational, err := rt.loadReportOperationalData(eventID, orgID)
	if err != nil {
		reportError(w, 500, "Impossibile caricare i dati operativi dell'evento.")
		return
	}
	previous := []reportSignal{}
	previousOperational := reportOperationalData{}
	var previousID int
	err = rt.db.SQLConn().QueryRow(`SELECT id FROM events WHERE organization_id=? AND id<>? AND (julianday(start_datetime) < julianday(?) OR (julianday(start_datetime)=julianday(?) AND id<?)) ORDER BY julianday(start_datetime) DESC,id DESC LIMIT 1`, orgID, eventID, event.StartDateTime, event.StartDateTime, eventID).Scan(&previousID)
	if err != nil && err != sql.ErrNoRows {
		reportError(w, 500, "Impossibile confrontare l'evento precedente.")
		return
	}
	if previousID > 0 {
		previous, err = rt.loadReportSignals(previousID)
		if err != nil {
			reportError(w, 500, "Impossibile confrontare l'evento precedente.")
			return
		}
		previousOperational, err = rt.loadReportOperationalData(previousID, orgID)
		if err != nil {
			reportError(w, 500, "Impossibile confrontare le attività dell'evento precedente.")
			return
		}
	}
	facts, details, started := calculateCombinedReportMetrics(signals, previous, operational, previousOperational)
	title := strings.TrimSpace(event.Title + " - " + event.OrganizationName)
	if title == "-" || title == "" {
		title = fmt.Sprintf("Evento #%d", event.ID)
	}
	org, err := rt.db.GetOrganization(orgID)
	if err != nil {
		reportError(w, 500, "Impossibile caricare la società.")
		return
	}
	snapshot := engagementReportSnapshot{CalculationID: newReportID(), Event: reportEvent{event.ID, title, event.StartDateTime, event.Location, org.Name, org.LogoURL}, Branding: brand, CalculatedAt: time.Now().UTC().Format(time.RFC3339Nano), MeasurementStartedAt: started, Facts: facts, KPIs: buildCombinedReportKPIs(facts, details, signals, previous, operational, previousOperational), Details: details, SchemaVersion: 1}
	// Preserve the existing one-page default while exposing every available
	// indicator in the review form. Additional KPIs can be selected explicitly.
	for i, kpi := range snapshot.KPIs {
		if i >= 20 {
			break
		}
		snapshot.SelectedKPIIDs = append(snapshot.SelectedKPIIDs, kpi.ID)
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		reportError(w, 500, "Impossibile preparare il report.")
		return
	}
	_, err = rt.db.SQLConn().Exec(`INSERT INTO engagement_report_calculations(id,event_id,admin_id,payload,created_at) VALUES(?,?,?,?,?)`, snapshot.CalculationID, eventID, ctx.AdminID, string(raw), snapshot.CalculatedAt)
	if err != nil {
		reportError(w, 500, "Impossibile preparare il report.")
		return
	}
	// Expire only unapproved calculation drafts; approved reports are durable.
	_, _ = rt.db.SQLConn().Exec(`DELETE FROM engagement_report_calculations WHERE created_at < ? AND id NOT IN (SELECT calculation_id FROM engagement_report_snapshots)`, time.Now().UTC().Add(-7*24*time.Hour).Format(time.RFC3339Nano))
	var latest string
	err = rt.db.SQLConn().QueryRow(`SELECT payload FROM engagement_report_snapshots WHERE event_id=? ORDER BY approved_at DESC LIMIT 1`, eventID).Scan(&latest)
	if err == nil {
		var approved engagementReportSnapshot
		if json.Unmarshal([]byte(latest), &approved) == nil {
			snapshot.LatestReport = &approved
		}
	} else if err != sql.ErrNoRows {
		reportError(w, 500, "Impossibile caricare l'ultima versione approvata.")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	_ = writeJSON(w, 200, snapshot)
}

func sameReportNumber(a, b *float64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func (rt *_router) approveEngagementReport(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, _, ok := rt.reportEventScope(w, r, ctx)
	if !ok {
		return
	}
	var input struct {
		CalculationID  string                    `json:"calculation_id"`
		Format         string                    `json:"format"`
		Facts          map[string]*float64       `json:"facts"`
		Overrides      map[string]reportOverride `json:"overrides"`
		Descriptions   map[string]string         `json:"descriptions"`
		SelectedKPIIDs *[]string                 `json:"selected_kpi_ids"`
		Notes          string                    `json:"notes"`
	}
	if !decodeReportBody(w, r, 128*1024, &input) {
		return
	}
	if (input.Format != "one_page" && input.Format != "extended") || len(input.CalculationID) != 36 || utf8.RuneCountInString(input.Notes) > 4000 {
		reportError(w, 400, "Formato, note o calcolo non validi.")
		return
	}
	var raw string
	err := rt.db.SQLConn().QueryRow(`SELECT payload FROM engagement_report_calculations WHERE id=? AND event_id=? AND admin_id=?`, input.CalculationID, eventID, ctx.AdminID).Scan(&raw)
	if err == sql.ErrNoRows {
		reportError(w, 404, "Calcolo non disponibile. Riapri il report.")
		return
	}
	if err != nil {
		reportError(w, 500, "Impossibile caricare il calcolo.")
		return
	}
	var snapshot engagementReportSnapshot
	if err = json.Unmarshal([]byte(raw), &snapshot); err != nil {
		reportError(w, 500, "Calcolo non valido.")
		return
	}
	original := snapshot.Facts
	reviewed := make(map[string]*float64, len(original))
	for k, v := range original {
		reviewed[k] = v
	}
	for k, v := range input.Facts {
		if _, known := original[k]; !known {
			reportError(w, 400, "Dato di base sconosciuto.")
			return
		}
		if v != nil && (math.IsNaN(*v) || math.IsInf(*v, 0) || *v < 0 || *v > 1e12 || *v != math.Trunc(*v)) {
			reportError(w, 400, "I dati di base devono essere interi non negativi.")
			return
		}
		reviewed[k] = v
	}
	input.Notes = strings.TrimSpace(input.Notes)
	if active, visitors := reviewed["active_visitors"], reviewed["visitors"]; active != nil && visitors != nil && *active > *visitors {
		reportError(w, 400, "I partecipanti attivi non possono superare i partecipanti digitali osservabili.")
		return
	}
	// Formulas, captions and the available indicator set belong to the frozen
	// calculation. An older 20-KPI draft must not gain newly introduced KPIs.
	kpis := append([]reportKPI{}, snapshot.KPIs...)
	selected, err := validateReportKPISelection(kpis, input.SelectedKPIIDs, snapshot.SelectedKPIIDs, input.Format)
	if err != nil {
		reportError(w, 400, err.Error())
		return
	}
	knownKPIs := map[string]bool{}
	reviewedDescriptions := map[string]string{}
	for i := range kpis {
		k := &kpis[i]
		if k.FactKey != "" {
			k.Value = reviewed[k.FactKey]
		} else {
			k.Numerator, k.Denominator = reviewed[k.NumeratorKey], reviewed[k.DenominatorKey]
			k.Value = reportMetricRatio(k.Numerator, k.Denominator, k.Unit == "percent")
		}
		knownKPIs[k.ID] = true
		if description, exists := input.Descriptions[k.ID]; exists {
			description = strings.Join(strings.Fields(description), " ")
			if utf8.RuneCountInString(description) > 120 || strings.ContainsRune(description, '\x00') {
				reportError(w, 400, "La breve descrizione deve contenere al massimo 120 caratteri validi.")
				return
			}
			k.DisplayDescription = description
		}
		reviewedDescriptions[k.ID] = k.DisplayDescription
		if k.Unit == "percent" && k.Numerator != nil && k.Denominator != nil && *k.Numerator > *k.Denominator {
			reportError(w, 400, "Il numeratore supera il pubblico di riferimento: "+k.Label)
			return
		}
		k.Corrected = !sameReportNumber(k.Value, snapshot.KPIs[i].Value) || (k.FactKey != "" && !sameReportNumber(reviewed[k.FactKey], original[k.FactKey])) || (k.NumeratorKey != "" && !sameReportNumber(reviewed[k.NumeratorKey], original[k.NumeratorKey])) || (k.DenominatorKey != "" && !sameReportNumber(reviewed[k.DenominatorKey], original[k.DenominatorKey]))
		if k.Corrected {
			k.Source = "manual"
			k.Note = input.Notes
		}
		if override, exists := input.Overrides[k.ID]; exists {
			override.Note = strings.TrimSpace(override.Note)
			if utf8.RuneCountInString(override.Note) > 500 {
				reportError(w, 400, "La nota facoltativa non può superare 500 caratteri.")
				return
			}
			v := override.Value
			if v != nil && (math.IsNaN(*v) || math.IsInf(*v, 0) || *v < 0 || *v > 1e12 || (k.Unit == "percent" && *v > 100) || (k.Unit == "count" && *v != math.Trunc(*v))) {
				reportError(w, 400, "Valore KPI non valido.")
				return
			}
			k.Value = v
			// A directly reviewed rate need not equal the measured fraction.
			// Keep that fraction in original_facts, not beside the corrected value.
			k.Numerator, k.Denominator = nil, nil
			k.Source = "manual"
			k.Corrected = true
			k.Note = override.Note
			input.Overrides[k.ID] = override
		}
	}
	for id := range input.Overrides {
		if !knownKPIs[id] {
			reportError(w, 400, "Indicatore sconosciuto.")
			return
		}
	}
	for id := range input.Descriptions {
		if !knownKPIs[id] {
			reportError(w, 400, "Indicatore sconosciuto nella descrizione.")
			return
		}
	}
	snapshot.ID = newReportID()
	snapshot.ApprovedAt = time.Now().UTC().Format(time.RFC3339Nano)
	snapshot.ApprovedBy = ctx.AdminUsername
	snapshot.Format = input.Format
	snapshot.OriginalFacts = original
	snapshot.Facts = reviewed
	snapshot.KPIs = kpis
	snapshot.SelectedKPIIDs = selected
	snapshot.Overrides = input.Overrides
	snapshot.Notes = input.Notes
	rawBytes, err := json.Marshal(snapshot)
	if err != nil {
		reportError(w, 500, "Impossibile generare il report.")
		return
	}
	approvalInput, _ := json.Marshal(struct {
		CalculationID  string
		Format         string
		Facts          map[string]*float64
		Overrides      map[string]reportOverride
		Descriptions   map[string]string
		SelectedKPIIDs []string
		Notes          string
	}{input.CalculationID, input.Format, reviewed, input.Overrides, reviewedDescriptions, selected, input.Notes})
	approvalHash := sha256.Sum256(approvalInput)
	approvalKey := hex.EncodeToString(approvalHash[:])
	_, err = rt.db.SQLConn().Exec(`INSERT INTO engagement_report_snapshots(id,calculation_id,approval_key,event_id,admin_id,payload,approved_at) VALUES(?,?,?,?,?,?,?) ON CONFLICT(approval_key) DO NOTHING`, snapshot.ID, snapshot.CalculationID, approvalKey, eventID, ctx.AdminID, string(rawBytes), snapshot.ApprovedAt)
	if err != nil {
		reportError(w, 500, "Impossibile salvare l'approvazione.")
		return
	}
	// A transport retry returns the same immutable approved version.
	if err = rt.db.SQLConn().QueryRow(`SELECT payload FROM engagement_report_snapshots WHERE approval_key=? AND event_id=?`, approvalKey, eventID).Scan(&raw); err != nil {
		reportError(w, 500, "Impossibile caricare il report approvato.")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, raw)
}

// Sort helper is shared by report verification tests without relying on map order.
func reportFactKeys(facts map[string]*float64) []string {
	keys := make([]string, 0, len(facts))
	for k := range facts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
