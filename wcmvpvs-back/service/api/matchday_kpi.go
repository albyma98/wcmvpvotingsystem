package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
)

type matchdayQueryer interface {
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type matchdayEvent struct {
	ID            int    `json:"id"`
	Title         string `json:"title"`
	StartDateTime string `json:"start_datetime"`
	HomeTeam      string `json:"home_team"`
	AwayTeam      string `json:"away_team"`
	Organization  string `json:"organization"`
	Division      string `json:"division"`
	start         time.Time
}

type matchdayTracking struct {
	ID                                      int
	Name, Page, Source, Device, Session, At string
	Metadata                                map[string]interface{}
}

type matchdayIdentity struct {
	Device, DeviceKey, Session string
	FanID                      int
}

type matchdayAction struct {
	Identity matchdayIdentity
	Feature  string
	Count    int
}

type matchdayFacts struct {
	Event                          matchdayEvent
	Signals                        []reportSignal
	Tracking                       []matchdayTracking
	Metrics                        map[string]matchdayMetric
	Manual                         map[string]matchdayManualField
	Actions                        []matchdayAction
	Resolve                        func(matchdayIdentity) string
	Reach                          map[string]bool
	persistentReach                map[string]bool
	FirstEvent                     map[string]int
	BarRevenue                     *float64
	MerchRevenue                   *float64
	Technology                     matchdayTechnologyFacts
	qrOpens, qrLandings            int
	registrations                  []matchdayIdentity
	Operations                     matchdayOperationsFacts
	votes                          []matchdayVote
	games                          []matchdayGameObservation
	feedback                       *matchdayFeedbackData
	rewards                        int
	lottery                        int
	legacyDevices                  []string
	legacyDurations, legacySeconds int
	coverage                       string
}

type matchdayColumn struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Unit  string `json:"unit"`
}

type matchdayRow struct {
	Label  string              `json:"label"`
	Group  string              `json:"group,omitempty"`
	Values map[string]*float64 `json:"values"`
}

type matchdayMetric struct {
	Code        string              `json:"code"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Unit        string              `json:"unit"`
	Formula     string              `json:"formula"`
	Source      string              `json:"source,omitempty"`
	Status      string              `json:"status"`
	Reason      string              `json:"reason,omitempty"`
	Value       *float64            `json:"value"`
	Numerator   *float64            `json:"numerator,omitempty"`
	Denominator *float64            `json:"denominator,omitempty"`
	Sources     []string            `json:"sources"`
	Missing     []string            `json:"missing,omitempty"`
	Rows        []matchdayRow       `json:"rows,omitempty"`
	Columns     []matchdayColumn    `json:"columns,omitempty"`
	Comparison  *matchdayComparison `json:"comparison,omitempty"`
	coverage    string
}

type matchdayPoint struct {
	EventID       int     `json:"event_id"`
	Label         string  `json:"label"`
	StartDateTime string  `json:"start_datetime"`
	Value         float64 `json:"value"`
}

type matchdayComparison struct {
	Previous             *float64        `json:"previous,omitempty"`
	Average              *float64        `json:"average,omitempty"`
	AbsoluteDelta        *float64        `json:"absolute_delta,omitempty"`
	PercentDelta         *float64        `json:"percent_delta,omitempty"`
	AverageAbsoluteDelta *float64        `json:"average_absolute_delta,omitempty"`
	AveragePercentDelta  *float64        `json:"average_percent_delta,omitempty"`
	Series               []matchdayPoint `json:"series"`
}

type matchdaySection struct {
	ID      string           `json:"id"`
	Title   string           `json:"title"`
	Metrics []matchdayMetric `json:"metrics"`
}

var matchdayDepartments = []matchdaySection{
	{ID: "direction", Title: "Direzione e performance generale"},
	{ID: "marketing", Title: "Marketing e acquisizione"},
	{ID: "mvp", Title: "Fan engagement e MVP"},
	{ID: "games", Title: "Gamification e premi"},
	{ID: "feedback", Title: "Customer experience e feedback"},
	{ID: "sponsor", Title: "Sponsorship e commerciale"},
	{ID: "crm", Title: "CRM e fidelizzazione"},
	{ID: "shop", Title: "Merchandising e shop"},
	{ID: "chat", Title: "Community e chat live"},
	{ID: "tech", Title: "Tecnologia e affidabilità"},
}

var matchdayExecutive = []string{
	"direction_reach", "direction_penetration", "mvp_multifunction", "direction_retention",
	"mvp_voters", "games_players", "feedback_responses", "crm_new_profiles", "crm_new_consents",
	"sponsor_reach", "sponsor_viewable", "shop_conversions", "direction_far",
}

type matchdayResponse struct {
	Event      matchdayEvent `json:"event"`
	Scope      string        `json:"scope"`
	Limit      int           `json:"limit"`
	Comparison struct {
		PreviousEvent    *matchdayEvent `json:"previous_event,omitempty"`
		Season           string         `json:"season"`
		EventsConsidered int            `json:"events_considered"`
	} `json:"comparison"`
	Executive    []string              `json:"executive"`
	Sections     []matchdaySection     `json:"sections"`
	ManualFields []matchdayManualField `json:"manual_fields"`
	Availability struct {
		Available int `json:"available"`
		Partial   int `json:"partial"`
		Total     int `json:"total"`
	} `json:"availability"`
}

func matchdayCount(code, name, description, formula string, value float64, sources ...string) matchdayMetric {
	return matchdayMetric{Code: code, Name: name, Description: description, Formula: formula, Unit: "count", Value: reportMetricFloat(value), Source: "db", Status: "available", Sources: sources}
}

func matchdayRatio(code, name, description, formula, unit string, numerator, denominator float64, sources ...string) matchdayMetric {
	m := matchdayCount(code, name, description, formula, numerator, sources...)
	m.Unit, m.Numerator, m.Denominator = unit, reportMetricFloat(numerator), reportMetricFloat(denominator)
	m.Value = reportMetricRatio(m.Numerator, m.Denominator, unit == "percent")
	if m.Value == nil {
		m.Status, m.Reason = "not_applicable", "Denominatore pari a zero."
	}
	return m
}

func matchdayUnavailable(code, name, description, formula, unit, status, reason string, sources ...string) matchdayMetric {
	return matchdayMetric{Code: code, Name: name, Description: description, Formula: formula, Unit: unit, Status: status, Reason: reason, Sources: sources}
}

func matchdayPut(f *matchdayFacts, metrics ...matchdayMetric) {
	for _, metric := range metrics {
		if metric.Sources == nil {
			metric.Sources = []string{}
		}
		f.Metrics[metric.Code] = metric
	}
}

func matchdayIDs(data map[int]*matchdayFacts) (string, []interface{}) {
	ids := make([]int, 0, len(data))
	for id := range data {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	return strings.TrimSuffix(strings.Repeat("?,", len(args)), ","), args
}

func matchdayRead(ctx context.Context, q matchdayQueryer, query string, args []interface{}, scan func(*sql.Rows) error) error {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := scan(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}

func matchdaySeason(at time.Time) int {
	if at.IsZero() {
		return 0
	}
	at = at.In(matchdayRome)
	year := at.Year()
	if at.Month() < time.July {
		year--
	}
	return year
}

func loadMatchdayEvents(ctx context.Context, q matchdayQueryer, orgID, eventID, limit int, scope string) (matchdayEvent, []matchdayEvent, string, error) {
	var current matchdayEvent
	var events []matchdayEvent
	err := matchdayRead(ctx, q, `SELECT e.id,e.start_datetime,COALESCE(h.name,''),COALESCE(a.name,''),o.name,
		COALESCE(CASE WHEN e.team1_id=o.team_id THEN h.championship WHEN e.team2_id=o.team_id THEN a.championship END,'')
		FROM events e JOIN organizations o ON o.id=e.organization_id
		LEFT JOIN teams h ON h.id=e.team1_id LEFT JOIN teams a ON a.id=e.team2_id
		WHERE e.organization_id=? AND e.is_concluded=1`, []interface{}{orgID}, func(rows *sql.Rows) error {
		var e matchdayEvent
		if err := rows.Scan(&e.ID, &e.StartDateTime, &e.HomeTeam, &e.AwayTeam, &e.Organization, &e.Division); err != nil {
			return err
		}
		e.start, e.StartDateTime = parseEventStart(e.StartDateTime)
		e.Division = strings.TrimSpace(e.Division)
		e.Title = buildEventTitle(database.Event{ID: e.ID, Team1Name: e.HomeTeam, Team2Name: e.AwayTeam})
		if e.ID == eventID {
			current = e
		} else {
			events = append(events, e)
		}
		return nil
	})
	if err != nil {
		return current, nil, scope, err
	}
	if current.ID == 0 {
		return current, nil, scope, sql.ErrNoRows
	}
	if current.Division == "" {
		scope = "club"
	} else if scope == "" {
		scope = "division"
	}
	prior := events[:0]
	season := matchdaySeason(current.start)
	for _, e := range events {
		if e.start.IsZero() || current.start.IsZero() || !e.start.Before(current.start) {
			continue
		}
		// The team only stores its current championship. Older seasons cannot
		// be assumed to have used the same division.
		if scope == "division" && (matchdaySeason(e.start) != season || !strings.EqualFold(e.Division, current.Division)) {
			continue
		}
		prior = append(prior, e)
	}
	sort.Slice(prior, func(i, j int) bool {
		a, b := prior[i], prior[j]
		if !a.start.Equal(b.start) {
			return a.start.After(b.start)
		}
		return a.ID > b.ID
	})
	if len(prior) > limit+1 {
		prior = prior[:limit+1]
	}
	return current, prior, scope, nil
}

func matchdayCompare(metric matchdayMetric, prior []*matchdayFacts, season int) *matchdayComparison {
	if metric.Value == nil {
		return nil
	}
	out := &matchdayComparison{Series: []matchdayPoint{}}
	var sum float64
	for i, f := range prior {
		old := f.Metrics[metric.Code]
		if old.Value == nil || old.Unit != metric.Unit || old.Source != metric.Source || old.coverage != metric.coverage || !reflect.DeepEqual(old.Sources, metric.Sources) {
			continue
		}
		if i == 0 {
			out.Previous = old.Value
		}
		if matchdaySeason(f.Event.start) == season {
			sum += *old.Value
			out.Series = append(out.Series, matchdayPoint{f.Event.ID, f.Event.Title, f.Event.StartDateTime, *old.Value})
		}
	}
	if len(out.Series) > 0 {
		out.Average = reportMetricFloat(sum / float64(len(out.Series)))
		for i, j := 0, len(out.Series)-1; i < j; i, j = i+1, j-1 {
			out.Series[i], out.Series[j] = out.Series[j], out.Series[i]
		}
	}
	compare := func(base *float64) (*float64, *float64) {
		if base == nil {
			return nil, nil
		}
		delta := reportMetricFloat(*metric.Value - *base)
		if delta == nil || *base == 0 {
			return delta, nil
		}
		return delta, reportMetricFloat(*delta / *base * 100)
	}
	out.AbsoluteDelta, out.PercentDelta = compare(out.Previous)
	out.AverageAbsoluteDelta, out.AveragePercentDelta = compare(out.Average)
	return out
}

func (rt *_router) matchdayKPIReport(ctx context.Context, q matchdayQueryer, orgID, eventID, limit int, scope string, manualHistory bool) (matchdayResponse, error) {
	current, events, scope, err := loadMatchdayEvents(ctx, q, orgID, eventID, limit, scope)
	if err != nil {
		return matchdayResponse{}, err
	}
	data := make(map[int]*matchdayFacts, len(events)+1)
	ordered := make([]*matchdayFacts, 0, len(events)+1)
	for _, event := range append([]matchdayEvent{current}, events...) {
		f := &matchdayFacts{Event: event, Metrics: make(map[string]matchdayMetric), Manual: make(map[string]matchdayManualField)}
		data[event.ID], ordered = f, append(ordered, f)
	}
	loaders := []struct {
		name string
		load func(context.Context, matchdayQueryer, int, map[int]*matchdayFacts) error
	}{
		{"manual", loadMatchdayManual}, {"telemetry", loadMatchdayTelemetry}, {"page engagement", loadMatchdayPageEngagement}, {"votes and games", loadMatchdayVotesAndGames},
		{"feedback", preloadMatchdayFeedback}, {"operations", loadMatchdayOperations},
		{"QR arrivals", loadMatchdayQRArrivals}, {"registrations", loadMatchdayRegistrations}, {"technology", loadMatchdayTechnology},
	}
	for _, loader := range loaders {
		if err := loader.load(ctx, q, orgID, data); err != nil {
			return matchdayResponse{}, fmt.Errorf("matchday %s: %w", loader.name, err)
		}
	}
	if err := calculateMatchdayAudience(ctx, q, orgID, ordered, func() error {
		return loadMatchdayMarketing(ctx, q, orgID, data, rt.VoteSecret)
	}); err != nil {
		return matchdayResponse{}, fmt.Errorf("matchday audience: %w", err)
	}
	for i, f := range ordered {
		var previous *matchdayFacts
		if i+1 < len(ordered) {
			previous = ordered[i+1]
		}
		calculateMatchdayOperations(f)
		calculateMatchdayPerformance(f, previous)
		calculateMatchdayMarketing(f, previous)
	}
	f := ordered[0]
	result := matchdayResponse{Event: current, Scope: scope, Limit: limit, Executive: matchdayExecutive, Sections: append([]matchdaySection(nil), matchdayDepartments...)}
	prior := ordered[1:]
	if len(prior) > limit {
		prior = prior[:limit]
	}
	if len(prior) > 0 {
		result.Comparison.PreviousEvent = &prior[0].Event
	}
	season := matchdaySeason(current.start)
	if !current.start.IsZero() {
		result.Comparison.Season = fmt.Sprintf("%d/%d", season, season+1)
	}
	for _, item := range prior {
		if matchdaySeason(item.Event.start) == season {
			result.Comparison.EventsConsidered++
		}
	}
	for i, section := range result.Sections {
		metrics := []matchdayMetric{}
		for code, m := range f.Metrics {
			if !strings.HasPrefix(code, section.ID+"_") {
				continue
			}
			m.Comparison = matchdayCompare(m, prior, season)
			metrics = append(metrics, m)
			if code == "direction_activation" {
				continue // Legacy API alias for direction_far, omitted from the displayed KPI count.
			}
			result.Availability.Total++
			switch m.Status {
			case "available":
				result.Availability.Available++
			case "partial":
				result.Availability.Partial++
			}
		}
		sort.Slice(metrics, func(i, j int) bool { return metrics[i].Name < metrics[j].Name })
		result.Sections[i].Metrics = metrics
	}
	if manualHistory {
		if err := loadMatchdayManualHistory(ctx, q, orgID, f); err != nil {
			return matchdayResponse{}, err
		}
	}
	for _, definition := range matchdayManualDefinitions {
		if !f.allowsMatchdayManual(definition.Code) {
			continue
		}
		field, ok := f.Manual[definition.Code]
		if !ok {
			field = definition
		}
		if field.History == nil {
			field.History = []matchdayManualRevision{}
		}
		result.ManualFields = append(result.ManualFields, field)
	}
	return result, nil
}

func (f *matchdayFacts) allowsMatchdayManual(code string) bool {
	switch code {
	case "qr_scans":
		return f.qrOpens == 0
	case "shop_paid_orders", "shop_gmv", "shop_reservations":
		return !f.Operations.merch.observed
	default:
		return true
	}
}

func (rt *_router) handleMatchdayKPI(w http.ResponseWriter, r *http.Request, auth reqcontext.RequestContext) {
	if !recipientSourceAdmin(w, auth) {
		return
	}
	eventID, orgID, ok := rt.reportEventScope(w, r, auth)
	if !ok {
		return
	}
	query := r.URL.Query()
	scope, limit := query.Get("scope"), 8
	if raw := query.Get("limit"); raw != "" {
		limit, _ = strconv.Atoi(raw)
	}
	if (scope != "" && scope != "club" && scope != "division") || limit < 1 || limit > 20 {
		reportError(w, http.StatusBadRequest, "Confronto non valido.")
		return
	}
	var input struct {
		Values []matchdayManualInput `json:"values"`
	}
	if r.Method == http.MethodPut {
		if !decodeReportBody(w, r, 16*1024, &input) {
			return
		}
		if err := validateMatchdayManual(input.Values); err != nil {
			reportError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	ctx := r.Context()
	tx, err := rt.db.SQLConn().BeginTx(ctx, nil)
	if err == nil {
		defer tx.Rollback()
		if r.Method == http.MethodPut {
			err = saveMatchdayManual(ctx, tx, orgID, eventID, auth.AdminID, input.Values)
		}
		if err == nil {
			var result matchdayResponse
			result, err = rt.matchdayKPIReport(ctx, tx, orgID, eventID, limit, scope, true)
			if err == nil {
				for _, input := range input.Values {
					allowed := false
					for _, field := range result.ManualFields {
						allowed = allowed || input.Code == field.Code
					}
					if !allowed {
						reportError(w, http.StatusConflict, "Il dato è rilevato automaticamente. Aggiorna la scheda per vedere il valore misurato.")
						return
					}
				}
				err = tx.Commit()
				if err == nil {
					_ = writeJSON(w, http.StatusOK, result)
					return
				}
			}
		}
	}
	switch {
	case errors.Is(err, sql.ErrNoRows):
		reportError(w, http.StatusNotFound, "Evento archiviato non disponibile per questa società.")
	case errors.Is(err, errMatchdayRevision), isMatchdaySQLiteBusy(err):
		reportError(w, http.StatusConflict, "Dati aggiornati da un’altra richiesta. Ricarica il pannello e riprova.")
	default:
		auth.Logger.WithError(err).WithField("event_id", eventID).Error("matchday KPI report")
		reportError(w, http.StatusInternalServerError, "Impossibile calcolare i KPI Matchday.")
	}
}
