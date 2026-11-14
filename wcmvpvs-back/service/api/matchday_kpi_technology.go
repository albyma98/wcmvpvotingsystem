package api

import (
	"context"
	"database/sql"
	"sort"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
)

type matchdayTechnologyFacts struct {
	latencies                                       map[int64]int
	requests, errors, clientErrors, limitedRequests int
	serverErrors, votes, successfulVotes, limited   int
	probes, healthy, expectedProbes                 int
}

func loadMatchdayTechnology(ctx context.Context, q matchdayQueryer, orgID int, data map[int]*matchdayFacts) error {
	in, ids := matchdayIDs(data)
	if len(ids) == 0 {
		return nil
	}
	args := append([]interface{}{orgID}, ids...)
	if err := matchdayRead(ctx, q, `SELECT event_id,route,http_status,latency_ms,samples FROM matchday_api_requests
		WHERE organization_id=? AND event_id IN (`+in+`)`, args, func(rows *sql.Rows) error {
		var event, status, count int
		var route string
		var latency int64
		if err := rows.Scan(&event, &route, &status, &latency, &count); err != nil {
			return err
		}
		f := &data[event].Technology
		if f.latencies == nil {
			f.latencies = make(map[int64]int)
		}
		f.latencies[latency] += count
		f.requests += count
		switch {
		case status == 429:
			f.errors += count
			f.limitedRequests += count
		case status >= 400 && status < 500:
			f.errors += count
			f.clientErrors += count
		case status >= 500 && status < 600:
			f.errors += count
			f.serverErrors += count
		}
		if route == "vote" {
			f.votes += count
			if status >= 200 && status < 300 {
				f.successfulVotes += count
			}
			if status == 429 {
				f.limited += count
			}
		}
		return nil
	}); err != nil {
		return err
	}
	// Count expected probe slots per active period independently of observed
	// samples. A stopped probe cannot tell whether the API was reachable.
	probeInterval := database.MatchdayProbeIntervalSeconds * 1000
	probeArgs := append([]interface{}{probeInterval - 1, probeInterval, probeInterval - 1, probeInterval, database.MatchdayProbeIntervalSeconds}, args...)
	return matchdayRead(ctx, q, `SELECT w.event_id,
		(w.closed_at+?)/?-(w.opened_at+?)/?,
		COUNT(p.slot_start), COALESCE(SUM(p.healthy),0)
		FROM matchday_technology_windows w
		LEFT JOIN matchday_health_probes p ON p.slot_start>=(w.opened_at+999)/1000 AND p.slot_start<(w.closed_at+999)/1000
			AND p.slot_start % ? = 0
		WHERE w.organization_id=? AND w.event_id IN (`+in+`) AND w.closed_at IS NOT NULL
		GROUP BY w.id`, probeArgs, func(rows *sql.Rows) error {
		var event, expected, probes, healthy int
		if err := rows.Scan(&event, &expected, &probes, &healthy); err != nil {
			return err
		}
		f := &data[event].Technology
		f.expectedProbes += expected
		f.probes += probes
		f.healthy += healthy
		return nil
	})
}

func (f matchdayTechnologyFacts) latencyP95() int64 {
	latencies := make([]int64, 0, len(f.latencies))
	for latency := range f.latencies {
		latencies = append(latencies, latency)
	}
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	target := f.requests - f.requests/20 // ceil(95% of the number of samples)
	for _, latency := range latencies {
		target -= f.latencies[latency]
		if target <= 0 {
			return latency
		}
	}
	return 0
}

func calculateMatchdayServerTechnology(f *matchdayFacts) {
	const requests = "matchday_api_requests.event_id, route, http_status, latency_ms, samples"
	const probes = "matchday_health_probes.slot_start, healthy; matchday_technology_windows.opened_at, closed_at"
	observed := f.Technology
	uptime := matchdayUnavailable("tech_uptime", "Disponibilità del servizio",
		"Raggiungibilità dell’API dal punto di accesso configurato per la sonda durante l’attivazione della partita.",
		"Sonde riuscite / sonde eseguite × 100", "percent", "not_measurable",
		"Nessuno slot di sonda atteso negli intervalli di attivazione chiusi della partita.", probes)
	if observed.expectedProbes > 0 {
		if observed.probes > 0 {
			uptime = matchdayRatio(uptime.Code, uptime.Name, uptime.Description, uptime.Formula, uptime.Unit, float64(observed.healthy), float64(observed.probes), probes)
			if observed.probes < observed.expectedProbes {
				uptime.Status, uptime.Reason = "partial", "Alcuni intervalli non sono stati osservati dalla sonda."
				uptime.coverage = "incomplete-probe-coverage"
			}
		} else {
			uptime.Reason = "Nessuna misura ricevuta negli slot attesi."
		}
		uptime.Columns = []matchdayColumn{{ID: "count", Label: "Sonde", Unit: "count"}}
		uptime.Rows = []matchdayRow{
			{Label: "Attese", Values: map[string]*float64{"count": matchdayOperationsValue(observed.expectedProbes)}},
			{Label: "Riuscite", Values: map[string]*float64{"count": matchdayOperationsValue(observed.healthy)}},
			{Label: "Fallite", Values: map[string]*float64{"count": matchdayOperationsValue(observed.probes - observed.healthy)}},
			{Label: "Attese senza misura", Values: map[string]*float64{"count": matchdayOperationsValue(observed.expectedProbes - observed.probes)}},
		}
	}
	matchdayPut(f, uptime)
	if observed.requests == 0 {
		return
	}
	latency := matchdayCount("tech_api_p95", "Latenza API p95", "Tempo lato server delle richieste HTTP misurate dalla home tifosi, esclusi i flussi in streaming.", "95º percentile delle durate HTTP, in millisecondi", float64(observed.latencyP95()), requests)
	latency.Unit = "milliseconds"
	errorRate := matchdayRatio("tech_request_error_rate", "Tasso di errore delle richieste", "Risposte HTTP 4xx o 5xx sul traffico API misurato della partita.", "Richieste HTTP 4xx o 5xx / richieste misurate × 100", "percent", float64(observed.errors), float64(observed.requests), requests)
	errorRate.Columns = []matchdayColumn{{ID: "count", Label: "Risposte HTTP", Unit: "count"}}
	errorRate.Rows = []matchdayRow{
		{Label: "4xx (esclusi i 429)", Values: map[string]*float64{"count": matchdayOperationsValue(observed.clientErrors)}},
		{Label: "429 · Limite di frequenza", Values: map[string]*float64{"count": matchdayOperationsValue(observed.limitedRequests)}},
		{Label: "5xx · Errore lato server", Values: map[string]*float64{"count": matchdayOperationsValue(observed.serverErrors)}},
	}
	anomalies := matchdayCount("tech_anomalies", "Tentativi di voto limitati", "Tentativi respinti dal limite di frequenza dell’API di voto.", "Richieste di voto con risposta HTTP 429", float64(observed.limited), requests)
	issues := matchdayCount("tech_application_issues", "Errori applicativi lato server", "Risposte HTTP di errore del servizio misurate durante la partita.", "Richieste HTTP con risposta 5xx", float64(observed.serverErrors), requests)
	matchdayPut(f, latency, errorRate, anomalies, issues)
	if observed.votes > 0 {
		vote := matchdayRatio("tech_vote_success", "Votazioni completate correttamente", "Esiti HTTP dei tentativi di voto ricevuti dal server per la partita.", "Richieste di voto 2xx / richieste di voto misurate × 100", "percent", float64(observed.successfulVotes), float64(observed.votes), requests)
		vote.Columns = []matchdayColumn{{ID: "count", Label: "Richieste HTTP", Unit: "count"}}
		vote.Rows = []matchdayRow{
			{Label: "Invii", Values: map[string]*float64{"count": matchdayOperationsValue(observed.votes)}},
			{Label: "Completati", Values: map[string]*float64{"count": matchdayOperationsValue(observed.successfulVotes)}},
			{Label: "Respinti o falliti", Values: map[string]*float64{"count": matchdayOperationsValue(observed.votes - observed.successfulVotes)}},
			{Label: "Respinti per frequenza", Values: map[string]*float64{"count": matchdayOperationsValue(observed.limited)}},
		}
		matchdayPut(f, vote)
	}
}
