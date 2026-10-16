package database

import (
	"database/sql"
	"errors"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

var (
	ErrCrossoverExpired  = errors.New("crossover session expired")
	ErrCrossoverTooEarly = errors.New("crossover duel is still in progress")
	ErrCrossoverOrder    = errors.New("crossover duel out of order")
	ErrCrossoverGesture  = errors.New("invalid crossover trace")
)

type CrossoverSample struct {
	TMS int     `json:"t_ms"`
	X   float64 `json:"x"`
}

type CrossoverStats struct {
	Wins         int `json:"wins"`
	Streak       int `json:"streak"`
	BestStreak   int `json:"best_streak"`
	QualityTotal int `json:"quality_total"`
}

type CrossoverSession struct {
	SessionID   string `json:"session_id"`
	StartedAt   string `json:"started_at"`
	EndsAt      string `json:"ends_at"`
	ServerNow   string `json:"server_now"`
	RemainingMS int64  `json:"remaining_ms"`
	NextDuel    int    `json:"next_duel"`
	CrossoverStats
}

type CrossoverDuelResult struct {
	SessionID       string `json:"session_id"`
	Duel            int    `json:"duel"`
	Won             bool   `json:"won"`
	Clean           bool   `json:"clean"`
	Quality         int    `json:"quality"`
	Reason          string `json:"reason"`
	Wallet          int    `json:"wallet"`
	CoinsEarned     int    `json:"coins_earned"`
	CrossoverEarned int    `json:"crossover_earned"`
	NextDuel        int    `json:"next_duel"`
	RemainingMS     int64  `json:"remaining_ms"`
	CrossoverStats
}

type crossoverEvaluation struct {
	won, clean bool
	quality    int
	reason     string
}
type crossoverBait struct{ side, start, end float64 }
type crossoverCandidate struct {
	at     float64
	result crossoverEvaluation
}

const crossoverEpsilon = 1e-7

func validateCrossoverSamples(samples []CrossoverSample) error {
	if len(samples) < 2 || len(samples) > 128 || samples[0].TMS != 0 || samples[0].X != 0 {
		return ErrCrossoverGesture
	}
	for index, sample := range samples {
		if sample.TMS < 0 || sample.TMS > 6000 || math.IsNaN(sample.X) || math.IsInf(sample.X, 0) || sample.X < -1 || sample.X > 1 || (index > 0 && sample.TMS <= samples[index-1].TMS) {
			return ErrCrossoverGesture
		}
	}
	return nil
}

func crossoverTraceX(samples []CrossoverSample, at float64) float64 {
	if at <= 0 {
		return samples[0].X
	}
	for index := 1; index < len(samples); index++ {
		b, a := samples[index], samples[index-1]
		if at <= float64(b.TMS) {
			return a.X + (b.X-a.X)*(at-float64(a.TMS))/float64(b.TMS-a.TMS)
		}
	}
	return samples[len(samples)-1].X
}

// Qualify continuous holds by interpolating threshold entry/exit times rather
// than counting samples, so the rule is independent of touch sampling rate.
func crossoverBaits(samples []CrossoverSample) []crossoverBait {
	baits := []crossoverBait{}
	for _, side := range []float64{-1, 1} {
		intervals := []crossoverBait{}
		for index := 1; index < len(samples); index++ {
			a, b := samples[index-1], samples[index]
			ax, bx := side*a.X, side*b.X
			if ax < .45 && bx < .45 {
				continue
			}
			crossing := float64(a.TMS)
			if ax != bx {
				crossing += (.45 - ax) / (bx - ax) * float64(b.TMS-a.TMS)
			}
			start, end := crossing, crossing
			if ax >= .45 {
				start = float64(a.TMS)
			}
			if bx >= .45 {
				end = float64(b.TMS)
			}
			if len(intervals) > 0 && start <= intervals[len(intervals)-1].end+crossoverEpsilon {
				intervals[len(intervals)-1].end = end
			} else {
				intervals = append(intervals, crossoverBait{side, start, end})
			}
		}
		for _, interval := range intervals {
			if interval.end-interval.start+crossoverEpsilon >= 180 {
				baits = append(baits, interval)
			}
		}
	}
	return baits
}

func evaluateCrossoverDuel(duel int, samples []CrossoverSample) (crossoverEvaluation, error) {
	failure := crossoverEvaluation{reason: "no_feint"}
	if duel < 0 || duel > 2 {
		return failure, ErrCrossoverGesture
	}
	if err := validateCrossoverSamples(samples); err != nil {
		return failure, err
	}
	windows := [...]float64{1000, 850, 700}
	delays := [...]float64{300, 250, 200}
	candidates := []crossoverCandidate{}
	for _, bait := range crossoverBaits(samples) {
		cutAt := -1.0
		threshold := -bait.side * .65
		for index := 1; index < len(samples); index++ {
			a, b := samples[index-1], samples[index]
			if float64(b.TMS) < bait.end || bait.side*b.X > -.65 {
				continue
			}
			crossing := float64(a.TMS)
			if bait.side*a.X > -.65 {
				crossing += (-.65 - bait.side*a.X) / ((b.X - a.X) * bait.side) * float64(b.TMS-a.TMS)
			}
			if crossing+crossoverEpsilon >= bait.end {
				cutAt = math.Max(crossing, bait.end)
				break
			}
		}
		candidate := crossoverCandidate{at: bait.end, result: crossoverEvaluation{reason: "no_crossover"}}
		if cutAt >= 0 {
			candidate.at = cutAt
			reversal := cutAt - bait.end
			defenderX := .9 * crossoverTraceX(samples, math.Max(0, cutAt-delays[duel]))
			gap := math.Abs(threshold - defenderX)
			switch {
			case reversal > windows[duel]+crossoverEpsilon:
				candidate.result.reason = "too_slow"
			case gap+crossoverEpsilon < .75:
				candidate.result.reason = "blocked"
			default:
				quality := 60 + (gap-.75)*25 + (1-reversal/windows[duel])*20
				quality = math.Max(0, math.Min(100, quality))
				candidate.result = crossoverEvaluation{won: true, quality: int(math.Floor(quality + .5)), reason: "beat"}
				candidate.result.clean = candidate.result.quality >= 90
			}
		}
		candidates = append(candidates, candidate)
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].at < candidates[j].at })
	priorities := map[string]int{"no_feint": 0, "no_crossover": 1, "too_slow": 2, "blocked": 3}
	for _, candidate := range candidates {
		if candidate.result.won {
			return candidate.result, nil
		}
		if priorities[candidate.result.reason] > priorities[failure.reason] {
			failure = candidate.result
		}
	}
	return failure, nil
}

func crossoverSessionStats(tx *sql.Tx, sessionID string) (CrossoverStats, error) {
	stats := CrossoverStats{}
	rows, err := tx.Query(`SELECT won,quality FROM crossover_duels WHERE session_id=? ORDER BY duel`, sessionID)
	if err != nil {
		return stats, err
	}
	defer rows.Close()
	for rows.Next() {
		var won bool
		var quality int
		if err := rows.Scan(&won, &quality); err != nil {
			return stats, err
		}
		stats.QualityTotal += quality
		if won {
			stats.Wins++
			stats.Streak++
			if stats.Streak > stats.BestStreak {
				stats.BestStreak = stats.Streak
			}
		} else {
			stats.Streak = 0
		}
	}
	return stats, rows.Err()
}

func (db *appdbimpl) CreateCrossoverSession(eventID, organizationID, fanID int, deviceID string, now time.Time) (CrossoverSession, error) {
	deviceID = strings.TrimSpace(deviceID)
	if eventID <= 0 || organizationID <= 0 || fanID < 0 || deviceID == "" || len(deviceID) > 200 {
		return CrossoverSession{}, sql.ErrNoRows
	}
	tx, err := db.c.Begin()
	if err != nil {
		return CrossoverSession{}, err
	}
	defer tx.Rollback()
	if err := basketballEventActive(tx, eventID, organizationID); err != nil {
		return CrossoverSession{}, err
	}
	if err := basketballFanScope(tx, organizationID, fanID); err != nil {
		return CrossoverSession{}, err
	}
	if _, err := tx.Exec(`UPDATE crossover_sessions SET status='closed' WHERE event_id=? AND organization_id=? AND device_id=? AND status='active' AND (ends_at<=? OR next_duel>=3)`, eventID, organizationID, deviceID, now.UnixMilli()); err != nil {
		return CrossoverSession{}, err
	}
	var session CrossoverSession
	var started, ends int64
	var owner int
	err = tx.QueryRow(`SELECT session_id,started_at,ends_at,next_duel,fan_id FROM crossover_sessions WHERE event_id=? AND organization_id=? AND device_id=? AND status='active'`, eventID, organizationID, deviceID).Scan(&session.SessionID, &started, &ends, &session.NextDuel, &owner)
	if err == nil && owner > 0 && owner != fanID {
		return CrossoverSession{}, sql.ErrNoRows
	}
	if err == sql.ErrNoRows {
		session.SessionID = uuid.Must(uuid.NewV4()).String()
		started, ends = now.UnixMilli(), now.Add(25*time.Second).UnixMilli()
		_, err = tx.Exec(`INSERT INTO crossover_sessions(session_id,event_id,organization_id,device_id,fan_id,started_at,ends_at) VALUES(?,?,?,?,?,?,?)`, session.SessionID, eventID, organizationID, deviceID, fanID, started, ends)
	}
	if err != nil {
		return CrossoverSession{}, err
	}
	if session.CrossoverStats, err = crossoverSessionStats(tx, session.SessionID); err != nil {
		return CrossoverSession{}, err
	}
	session.StartedAt = time.UnixMilli(started).UTC().Format(time.RFC3339Nano)
	session.EndsAt = time.UnixMilli(ends).UTC().Format(time.RFC3339Nano)
	session.ServerNow = now.UTC().Format(time.RFC3339Nano)
	session.RemainingMS = playmakerRemaining(ends, now.UnixMilli())
	if err := tx.Commit(); err != nil {
		return CrossoverSession{}, err
	}
	return session, nil
}

func (db *appdbimpl) ResolveCrossoverDuel(eventID, organizationID, fanID int, deviceID, sessionID string, duel, duelStartMS int, samples []CrossoverSample, now time.Time) (CrossoverDuelResult, error) {
	deviceID = strings.TrimSpace(deviceID)
	result := CrossoverDuelResult{SessionID: sessionID, Duel: duel}
	if eventID <= 0 || organizationID <= 0 || fanID < 0 || deviceID == "" || len(deviceID) > 200 {
		return result, sql.ErrNoRows
	}
	if duel < 0 || duel > 2 {
		return result, ErrCrossoverGesture
	}
	tx, err := db.c.Begin()
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	if err := basketballFanScope(tx, organizationID, fanID); err != nil {
		return result, err
	}
	var started, ends, lastResolved int64
	var owner int
	err = tx.QueryRow(`SELECT started_at,ends_at,last_resolved_at,next_duel,fan_id FROM crossover_sessions WHERE session_id=? AND event_id=? AND organization_id=? AND device_id=?`, sessionID, eventID, organizationID, deviceID).Scan(&started, &ends, &lastResolved, &result.NextDuel, &owner)
	if err != nil {
		return result, err
	}
	if owner > 0 && owner != fanID {
		return result, sql.ErrNoRows
	}
	err = tx.QueryRow(`SELECT won,clean,quality,reason FROM crossover_duels WHERE session_id=? AND duel=?`, sessionID, duel).Scan(&result.Won, &result.Clean, &result.Quality, &result.Reason)
	if err == sql.ErrNoRows {
		if err := basketballEventActive(tx, eventID, organizationID); err != nil {
			return result, err
		}
		if now.UnixMilli() > ends+15000 {
			return result, ErrCrossoverExpired
		}
		if duel != result.NextDuel || result.NextDuel >= 3 {
			return result, ErrCrossoverOrder
		}
		if err := validateCrossoverSamples(samples); err != nil {
			return result, err
		}
		if duelStartMS < 0 || duelStartMS > 25000 || duelStartMS+samples[len(samples)-1].TMS > 25000 || int64(duelStartMS+samples[len(samples)-1].TMS) > now.UnixMilli()-started+1500 {
			return result, ErrCrossoverGesture
		}
		if now.UnixMilli() > ends {
			var late int
			if err := tx.QueryRow(`SELECT COUNT(*) FROM crossover_duels WHERE session_id=? AND resolved_at>?`, sessionID, ends).Scan(&late); err != nil {
				return result, err
			}
			if late > 0 {
				return result, ErrCrossoverExpired
			}
		}
		if lastResolved == 0 {
			lastResolved = started
		}
		if now.UnixMilli()-lastResolved < 1600 {
			return result, ErrCrossoverTooEarly
		}
		evaluation, err := evaluateCrossoverDuel(duel, samples)
		if err != nil {
			return result, err
		}
		updated, err := tx.Exec(`UPDATE crossover_sessions SET next_duel=next_duel+1,last_resolved_at=?,fan_id=?,status=CASE WHEN next_duel>=2 THEN 'closed' ELSE status END WHERE session_id=? AND next_duel=?`, now.UnixMilli(), fanID, sessionID, duel)
		if err != nil {
			return result, err
		}
		if n, err := updated.RowsAffected(); err != nil {
			return result, err
		} else if n != 1 {
			return result, ErrCrossoverOrder
		}
		if _, err := tx.Exec(`INSERT INTO crossover_duels(session_id,duel,event_id,organization_id,device_id,fan_id,duel_start_ms,won,clean,quality,reason,resolved_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, sessionID, duel, eventID, organizationID, deviceID, fanID, duelStartMS, evaluation.won, evaluation.clean, evaluation.quality, evaluation.reason, now.UnixMilli()); err != nil {
			return result, err
		}
		result.Won, result.Clean, result.Quality, result.Reason = evaluation.won, evaluation.clean, evaluation.quality, evaluation.reason
		result.NextDuel = duel + 1
		if evaluation.won {
			if fanID > 0 {
				_, err = tx.Exec(`INSERT INTO fan_wallets(fan_id,coins) VALUES(?,1) ON CONFLICT(fan_id) DO UPDATE SET coins=fan_wallets.coins+1,updated_at=CURRENT_TIMESTAMP`, fanID)
			} else {
				_, err = tx.Exec(`INSERT INTO guest_wallets(event_id,organization_id,device_id,coins) VALUES(?,?,?,1) ON CONFLICT(event_id,organization_id,device_id) DO UPDATE SET coins=guest_wallets.coins+1,updated_at=CURRENT_TIMESTAMP`, eventID, organizationID, deviceID)
			}
			if err != nil {
				return result, err
			}
			result.CoinsEarned = 1
		}
	} else if err != nil {
		return result, err
	}
	if fanID > 0 {
		err = tx.QueryRow(`SELECT coins FROM fan_wallets WHERE fan_id=?`, fanID).Scan(&result.Wallet)
	} else {
		err = tx.QueryRow(`SELECT coins FROM guest_wallets WHERE event_id=? AND organization_id=? AND device_id=?`, eventID, organizationID, deviceID).Scan(&result.Wallet)
	}
	if err != nil && err != sql.ErrNoRows {
		return result, err
	}
	if err := tx.QueryRow(`SELECT COUNT(*) FROM crossover_duels WHERE event_id=? AND organization_id=? AND won=1 AND (device_id=? OR (?>0 AND fan_id=?))`, eventID, organizationID, deviceID, fanID, fanID).Scan(&result.CrossoverEarned); err != nil {
		return result, err
	}
	if result.CrossoverStats, err = crossoverSessionStats(tx, sessionID); err != nil {
		return result, err
	}
	result.RemainingMS = playmakerRemaining(ends, now.UnixMilli())
	if err := tx.Commit(); err != nil {
		return result, err
	}
	return result, nil
}
