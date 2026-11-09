package api

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
)

var crmMetricIDs = []string{"visitors", "new_profiles", "event_retention", "active_participation", "game_participation", "sponsor_reach", "sponsor_interactors", "sponsored_participation"}

func crmMetrics(facts map[string]*float64) []reportKPI {
	var result []reportKPI
	for _, id := range crmMetricIDs {
		for _, kpi := range buildReportKPIs(facts) {
			if kpi.ID == id {
				result = append(result, kpi)
				break
			}
		}
	}
	return result
}

func crmHasMetric(value string) bool { return crmIs(value, crmMetricIDs...) }

func crmPercentMetric(value string) bool {
	return crmIs(value, "event_retention", "active_participation", "game_participation", "sponsored_participation")
}

type crmEvent struct {
	ID         int    `json:"id"`
	Title      string `json:"title"`
	Start      string `json:"start"`
	Day        string `json:"day"`
	Activated  bool   `json:"activated"`
	Concluded  bool   `json:"concluded"`
	ReportPath string `json:"reportPath,omitempty"`
}

type crmMatch struct {
	crmEvent
	KPIs []reportKPI `json:"kpis"`
}

type crmTrend struct {
	From         string      `json:"from"`
	To           string      `json:"to"`
	LastActivity *crmEvent   `json:"lastActivity"`
	Activated    int         `json:"activated"`
	Events       []crmEvent  `json:"events"`
	Matches      []crmMatch  `json:"matches"`
	Definitions  []reportKPI `json:"definitions"`
}

func crmEventDay(value string) string {
	// ArenaBoostX saves event times as RFC3339 or as local fixture times.
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05", crmLocalTime, "2006-01-02 15:04:05", crmDate} {
		if parsed, err := time.ParseInLocation(layout, value, crmRome); err == nil {
			return parsed.In(crmRome).Format(crmDate)
		}
	}
	return ""
}

func crmEvents(q crmQuery, orgID int64, slug string) ([]crmEvent, error) {
	rows, err := q.Query(`SELECT e.id,e.start_datetime,COALESCE(t1.name,'')||' – '||COALESCE(t2.name,''),
		e.is_concluded,
		e.is_active OR e.is_concluded
		OR EXISTS(SELECT 1 FROM engagement_report_signals s WHERE s.event_id=e.id)
		OR EXISTS(SELECT 1 FROM votes v WHERE v.event_id=e.id)
		OR EXISTS(SELECT 1 FROM event_feedback f WHERE f.event_id=e.id)
		OR EXISTS(SELECT 1 FROM fan_wallet_movements w WHERE w.event_id=e.id AND w.organization_id=e.organization_id)
		FROM events e LEFT JOIN teams t1 ON t1.id=e.team1_id LEFT JOIN teams t2 ON t2.id=e.team2_id
		WHERE e.organization_id=? ORDER BY julianday(e.start_datetime) DESC,e.id DESC`, orgID)
	events := []crmEvent{}
	err = crmReadRows(rows, err, func(row crmScanner) error {
		var event crmEvent
		if err := row.Scan(&event.ID, &event.Start, &event.Title, &event.Concluded, &event.Activated); err != nil {
			return err
		}
		event.Day = crmEventDay(event.Start)
		if event.Concluded {
			event.ReportPath = fmt.Sprintf("/admin/events/history/%d/report?organization_slug=%s", event.ID, url.QueryEscape(slug))
		}
		events = append(events, event)
		return nil
	})
	return events, err
}

func (rt *_router) crmPerformance(r *http.Request, _ reqcontext.RequestContext) (interface{}, error) {
	id, err := crmID(r, "companyID")
	if err != nil {
		return nil, err
	}
	today := time.Now().In(crmRome)
	from, to := r.URL.Query().Get("from"), r.URL.Query().Get("to")
	if from == "" {
		from = today.AddDate(0, 0, -89).Format(crmDate)
	}
	if to == "" {
		to = today.Format(crmDate)
	}
	if err := crmValidateDates(from, to); err != nil {
		return nil, err
	}
	if from > to {
		return nil, crmProblem("La fine del periodo deve seguire l’inizio.")
	}
	db := rt.db.SQLConn()
	company, err := crmReadCompany(db, id)
	if err != nil {
		return nil, err
	}
	trend := crmTrend{From: from, To: to, Events: []crmEvent{}, Matches: []crmMatch{}, Definitions: crmMetrics(nil)}
	if company.ABXOrgID == 0 {
		return trend, nil
	}
	var slug string
	if err := db.QueryRow("SELECT slug FROM organizations WHERE id=?", company.ABXOrgID).Scan(&slug); err != nil {
		return nil, err
	}
	trend.Events, err = crmEvents(db, company.ABXOrgID, slug)
	if err != nil {
		return nil, err
	}
	for _, event := range trend.Events {
		if !event.Activated || event.Day == "" || event.Day > today.Format(crmDate) {
			continue
		}
		if trend.LastActivity == nil {
			copy := event
			trend.LastActivity = &copy
		}
		if event.Day < from || event.Day > to {
			continue
		}
		trend.Activated++
		if len(trend.Matches) >= 8 {
			continue
		}
		previous, err := rt.precedingConcludedEvent(reportEvent{ID: event.ID, StartDateTime: event.Start}, int(company.ABXOrgID))
		if err != nil {
			return nil, err
		}
		snapshot, _, err := rt.buildRecipientEvent(event.ID, int(company.ABXOrgID), 0, previous)
		if err != nil {
			return nil, err
		}
		match := crmMatch{crmEvent: event, KPIs: []reportKPI{}}
		for _, metricID := range crmMetricIDs {
			for _, kpi := range snapshot.KPIs {
				if kpi.ID == metricID {
					match.KPIs = append(match.KPIs, kpi)
					break
				}
			}
		}
		trend.Matches = append(trend.Matches, match)
	}
	return trend, nil
}
