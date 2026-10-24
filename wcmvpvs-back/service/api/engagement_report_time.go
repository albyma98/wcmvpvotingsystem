package api

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// This is a reproducible activity-based estimate, not a dwell-time measurement.
// Each adjacent action pair contributes its first 60 seconds only when its gap
// is at most five minutes. No time is assigned before/after the known actions.
type reportTimeEstimate struct {
	Method             string  `json:"method"`
	CoveredVisitors    int     `json:"covered_visitors"`
	TotalVisitors      int     `json:"total_visitors"`
	ExcludedVisitors   int     `json:"excluded_visitors"`
	GapLimitSeconds    int     `json:"gap_limit_seconds"`
	IntervalCapSeconds int     `json:"interval_cap_seconds"`
	MeasuredSeconds    float64 `json:"measured_seconds"`
	InferredSeconds    float64 `json:"inferred_seconds"`
}

type reportTimePoint struct {
	at       time.Time
	priority int
}

func reportTimeUnionSeconds(spans []reportMetricInterval) float64 {
	if len(spans) == 0 {
		return 0
	}
	spans = append([]reportMetricInterval{}, spans...)
	sort.Slice(spans, func(i, j int) bool {
		if spans[i].start.Equal(spans[j].start) {
			return spans[i].end.Before(spans[j].end)
		}
		return spans[i].start.Before(spans[j].start)
	})
	merged, seconds := spans[0], float64(0)
	for _, span := range spans[1:] {
		if !span.start.After(merged.end) {
			if span.end.After(merged.end) {
				merged.end = span.end
			}
		} else {
			seconds += merged.end.Sub(merged.start).Seconds()
			merged = span
		}
	}
	return seconds + merged.end.Sub(merged.start).Seconds()
}

func calculateReportTimeEstimate(signals []reportSignal) (float64, reportTimeEstimate) {
	result := reportTimeEstimate{Method: "Unione dei tempi misurati e dei primi 60 secondi tra azioni consecutive distanti al massimo 5 minuti; nessuna durata prima della prima o dopo l'ultima azione.", GapLimitSeconds: 300, IntervalCapSeconds: 60}
	identity := reportMetricIdentityResolver(signals)
	visitors, modernGames, modernCompletions, strongActions := reportVisitorSet{}, reportVisitorSet{}, reportVisitorSet{}, reportVisitorSet{}
	for _, signal := range signals {
		actor := identity(signal)
		if actor == "" {
			continue
		}
		visitors[actor] = true
		// An undated answer proves participation, but cannot replace a dated
		// completion/credit when reconstructing the chronology of this event.
		if reportMetricTime(signal.OccurredAt).IsZero() {
			continue
		}
		table := reportMetricString(signal.Metadata, "_table")
		if signal.Name == "vote_completed" && table != "fan_wallet_movements" {
			strongActions[actor+"\x00vote"] = true
		}
		if signal.Name == "game_started" || signal.Name == "game_completed" {
			game := reportMetricString(signal.Metadata, "game_id")
			legacy, _ := signal.Metadata["_legacy_round"].(bool)
			if !legacy && reportMetricString(signal.Metadata, "round_id") != "" {
				modernGames[actor+"\x00"+game] = true
				if signal.Name == "game_completed" {
					modernCompletions[actor+"\x00"+game] = true
				}
			}
			if table != "fan_wallet_movements" {
				strongActions[actor+"\x00game:"+game] = true
			}
		}
	}
	points := map[string]map[string]reportTimePoint{}
	measured := map[string][]reportMetricInterval{}
	seen := reportVisitorSet{}
	for _, signal := range signals {
		actor := identity(signal)
		if actor == "" {
			continue
		}
		operational, _ := signal.Metadata["_operational"].(bool)
		origin := "browser:"
		if operational {
			origin = "operational:"
		}
		if signal.ID != "" {
			// A repeated ingestion ID is one observation. Sorting and the semantic
			// keys below also make ordinary duplicate source rows harmless.
			key := origin + signal.ID
			if seen[key] {
				continue
			}
			seen[key] = true
		}
		at := reportMetricTime(signal.OccurredAt)
		if signal.Name == "active_time" {
			seconds := reportMetricNumber(signal.Metadata, "seconds")
			if math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds <= 0 {
				continue
			}
			end := reportMetricTime(reportMetricString(signal.Metadata, "ended_at"))
			if end.IsZero() {
				end = at
			}
			if end.IsZero() {
				continue
			}
			limit := end.Add(-time.Duration(math.Min(seconds, 15) * float64(time.Second)))
			start := reportMetricTime(reportMetricString(signal.Metadata, "started_at"))
			if start.IsZero() || start.Before(limit) {
				start = limit
			}
			if start.Before(end) {
				measured[actor] = append(measured[actor], reportMetricInterval{start, end})
			}
			continue
		}
		if at.IsZero() {
			continue
		}
		table := reportMetricString(signal.Metadata, "_table")
		priority := 20 // Prefer the browser's action timestamp to a late reward receipt.
		if operational {
			priority = 10
		}
		key := ""
		switch signal.Name {
		case "page_view", "shop_view", "sponsor_view", "sponsored_game_view":
			key = signal.Name + "\x00" + at.UTC().Format(time.RFC3339Nano)
		case "vote_completed":
			key = "vote"
			if table == "fan_wallet_movements" && strongActions[actor+"\x00vote"] {
				continue
			}
		case "game_started", "game_completed":
			game, round := reportMetricString(signal.Metadata, "game_id"), reportMetricString(signal.Metadata, "round_id")
			if game == "" {
				continue
			}
			legacy, _ := signal.Metadata["_legacy_round"].(bool)
			if legacy {
				// A modern start alone does not replace a dated legacy completion.
				// Retain that end point until a modern completion can replace it.
				if signal.Name == "game_started" && modernGames[actor+"\x00"+game] || signal.Name == "game_completed" && modernCompletions[actor+"\x00"+game] {
					continue
				}
			}
			if table == "fan_wallet_movements" && strongActions[actor+"\x00game:"+game] {
				continue
			}
			key = signal.Name + "\x00" + game + "\x00" + round
			if table == "playmaker_passes" || table == "crossover_duels" {
				// These rows are real server-resolved actions within one session,
				// not repeated evidence that the session was merely opened.
				key += "\x00step:" + signal.ID
			}
		case "registration_started", "registration_completed", "feedback_completed":
			key = signal.Name
		case "redemption_completed":
			key = signal.Name + "\x00" + reportMetricString(signal.Metadata, "item_key")
		case "sponsor_click":
			// Matching click copies may have different ingestion IDs. Close clicks
			// on the same partner are collapsed below before estimating gaps.
			key = "click:" + strconv.Itoa(int(reportMetricNumber(signal.Metadata, "sponsor_id"))) + ":" + at.UTC().Format(time.RFC3339Nano)
		default:
			continue
		}
		if points[actor] == nil {
			points[actor] = map[string]reportTimePoint{}
		}
		current, exists := points[actor][key]
		if !exists || priority > current.priority || priority == current.priority && at.Before(current.at) {
			points[actor][key] = reportTimePoint{at: at, priority: priority}
		}
	}
	var total, measuredTotal float64
	for actor := range visitors {
		actorPoints := []reportTimePoint{}
		clicks := map[string][]reportTimePoint{}
		for key, point := range points[actor] {
			if strings.HasPrefix(key, "click:") {
				partner := strings.SplitN(strings.TrimPrefix(key, "click:"), ":", 2)[0]
				clicks[partner] = append(clicks[partner], point)
			} else {
				actorPoints = append(actorPoints, point)
			}
		}
		for _, group := range clicks {
			sort.Slice(group, func(i, j int) bool { return group[i].at.Before(group[j].at) })
			var previous time.Time
			for _, point := range group {
				if previous.IsZero() || point.at.Sub(previous) > 2*time.Second {
					actorPoints = append(actorPoints, point)
					previous = point.at
				}
			}
		}
		sort.Slice(actorPoints, func(i, j int) bool { return actorPoints[i].at.Before(actorPoints[j].at) })
		spans := append([]reportMetricInterval{}, measured[actor]...)
		for index := 1; index < len(actorPoints); index++ {
			start, end := actorPoints[index-1].at, actorPoints[index].at
			gap := end.Sub(start)
			if gap <= 0 || gap > time.Duration(result.GapLimitSeconds)*time.Second {
				continue
			}
			if gap > time.Duration(result.IntervalCapSeconds)*time.Second {
				end = start.Add(time.Duration(result.IntervalCapSeconds) * time.Second)
			}
			spans = append(spans, reportMetricInterval{start, end})
		}
		actorMeasured, actorTotal := reportTimeUnionSeconds(measured[actor]), reportTimeUnionSeconds(spans)
		if actorTotal > 0 {
			result.CoveredVisitors++
		}
		measuredTotal += actorMeasured
		total += actorTotal
	}
	result.TotalVisitors = len(visitors)
	result.ExcludedVisitors = result.TotalVisitors - result.CoveredVisitors
	result.MeasuredSeconds = math.Round(measuredTotal)
	result.InferredSeconds = math.Max(0, math.Round(total)-result.MeasuredSeconds)
	return math.Round(total), result
}
