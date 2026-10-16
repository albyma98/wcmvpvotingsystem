package database

import (
	"database/sql"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

var (
	ErrPlaymakerExpired  = errors.New("playmaker session expired")
	ErrPlaymakerTooEarly = errors.New("playmaker pass still in flight")
	ErrPlaymakerAttempt  = errors.New("playmaker attempt out of order")
	ErrPlaymakerGesture  = errors.New("invalid playmaker gesture")
)

type PlaymakerSession struct {
	SessionID   string `json:"session_id"`
	StartedAt   string `json:"started_at"`
	EndsAt      string `json:"ends_at"`
	ServerNow   string `json:"server_now"`
	RemainingMS int64  `json:"remaining_ms"`
	Attempt     int    `json:"attempt"`
	Assists     int    `json:"assists"`
	Streak      int    `json:"streak"`
	BestStreak  int    `json:"best_streak"`
}

type PlaymakerPassResult struct {
	SessionID       string `json:"session_id"`
	Attempt         int    `json:"attempt"`
	Assisted        bool   `json:"assisted"`
	Clean           bool   `json:"clean"`
	Reason          string `json:"reason"`
	Wallet          int    `json:"wallet"`
	CoinsEarned     int    `json:"coins_earned"`
	PlaymakerEarned int    `json:"playmaker_earned"`
	NextAttempt     int    `json:"next_attempt"`
	RemainingMS     int64  `json:"remaining_ms"`
}

type playmakerPoint struct{ x, y float64 }

func playmakerSegmentDistanceSquared(point, start, end playmakerPoint) float64 {
	dx, dy := end.x-start.x, end.y-start.y
	t := ((point.x-start.x)*dx + (point.y-start.y)*dy) / (dx*dx + dy*dy)
	t = math.Max(0, math.Min(1, t))
	x, y := point.x-(start.x+t*dx), point.y-(start.y+t*dy)
	return x*x + y*y
}

// This is the same deterministic court snapshot as the client's Playmaker
// physics. The client supplies a gesture, never a score or wallet balance.
func evaluatePlaymakerPass(attempt, elapsedMS int, dx, dy float64) (bool, bool, string, error) {
	if attempt < 0 || elapsedMS < 0 || elapsedMS > 30000 || math.IsNaN(dx) || math.IsNaN(dy) || math.IsInf(dx, 0) || math.IsInf(dy, 0) || math.Hypot(dx, dy) < 42 || dy > -30 {
		return false, false, "", ErrPlaymakerGesture
	}
	t := float64(elapsedMS) / 1000
	level := attempt / 4
	if level > 5 {
		level = 5
	}
	speed := 1.15 + float64(level)*.16
	phase := float64(attempt%8) * .65
	w := math.Sin(t*speed*.72 + phase)
	q := math.Sin(t*speed + phase + .5)
	mates := [...]playmakerPoint{{82 + 18*w, 248}, {278 - 18*w, 248}}
	aim := playmakerPoint{180 + dx, 466 + dy}
	chosen := mates[0]
	best := math.Inf(1)
	for _, mate := range mates {
		x, y := (aim.x-mate.x)/35, (aim.y-mate.y)/42
		err := x*x + y*y
		if err < best {
			best = err
			chosen = mate
		}
	}
	if best > 1 {
		return false, false, "off_target", nil
	}
	start := playmakerPoint{180, 466}
	if playmakerSegmentDistanceSquared(playmakerPoint{180 + 60*q, 350}, start, chosen) <= 22*22 {
		return false, false, "intercepted", nil
	}
	if attempt >= 8 && playmakerSegmentDistanceSquared(playmakerPoint{180 + 92*q, 318}, start, chosen) <= 20*20 {
		return false, false, "intercepted", nil
	}
	return true, best <= .16, "assist", nil
}

func playmakerRemaining(ends, now int64) int64 {
	remaining := ends - now
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (db *appdbimpl) CreatePlaymakerSession(eventID, organizationID, fanID int, deviceID string, now time.Time) (PlaymakerSession, error) {
	deviceID = strings.TrimSpace(deviceID)
	if eventID <= 0 || organizationID <= 0 || fanID < 0 || deviceID == "" || len(deviceID) > 200 {
		return PlaymakerSession{}, sql.ErrNoRows
	}
	tx, err := db.c.Begin()
	if err != nil {
		return PlaymakerSession{}, err
	}
	defer tx.Rollback()
	if err := basketballEventActive(tx, eventID, organizationID); err != nil {
		return PlaymakerSession{}, err
	}
	if err := basketballFanScope(tx, organizationID, fanID); err != nil {
		return PlaymakerSession{}, err
	}
	if _, err := tx.Exec(`UPDATE playmaker_sessions SET status='closed' WHERE event_id=? AND organization_id=? AND device_id=? AND status='active' AND ends_at<=?`, eventID, organizationID, deviceID, now.UnixMilli()); err != nil {
		return PlaymakerSession{}, err
	}
	var session PlaymakerSession
	var started, ends int64
	var owner int
	err = tx.QueryRow(`SELECT session_id,started_at,ends_at,next_attempt,fan_id FROM playmaker_sessions WHERE event_id=? AND organization_id=? AND device_id=? AND status='active'`, eventID, organizationID, deviceID).Scan(&session.SessionID, &started, &ends, &session.Attempt, &owner)
	if err == nil && owner > 0 && owner != fanID {
		return PlaymakerSession{}, sql.ErrNoRows
	}
	if err == sql.ErrNoRows {
		session.SessionID = uuid.Must(uuid.NewV4()).String()
		started, ends = now.UnixMilli(), now.Add(30*time.Second).UnixMilli()
		_, err = tx.Exec(`INSERT INTO playmaker_sessions(session_id,event_id,organization_id,device_id,fan_id,started_at,ends_at) VALUES(?,?,?,?,?,?,?)`, session.SessionID, eventID, organizationID, deviceID, fanID, started, ends)
	}
	if err != nil {
		return PlaymakerSession{}, err
	}
	rows, err := tx.Query(`SELECT assisted FROM playmaker_passes WHERE session_id=? ORDER BY attempt`, session.SessionID)
	if err != nil {
		return PlaymakerSession{}, err
	}
	for rows.Next() {
		var assisted bool
		if err := rows.Scan(&assisted); err != nil {
			rows.Close()
			return PlaymakerSession{}, err
		}
		if assisted {
			session.Assists++
			session.Streak++
			if session.Streak > session.BestStreak {
				session.BestStreak = session.Streak
			}
		} else {
			session.Streak = 0
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return PlaymakerSession{}, err
	}
	if err := rows.Close(); err != nil {
		return PlaymakerSession{}, err
	}
	session.StartedAt = time.UnixMilli(started).UTC().Format(time.RFC3339Nano)
	session.EndsAt = time.UnixMilli(ends).UTC().Format(time.RFC3339Nano)
	session.ServerNow = now.UTC().Format(time.RFC3339Nano)
	session.RemainingMS = playmakerRemaining(ends, now.UnixMilli())
	if err := tx.Commit(); err != nil {
		return PlaymakerSession{}, err
	}
	return session, nil
}

// Receipts, the sequential session cursor and the +1 wallet increment are one
// transaction. Saved results remain retryable after the session/event closes.
func (db *appdbimpl) ResolvePlaymakerPass(eventID, organizationID, fanID int, deviceID, sessionID string, attempt, elapsedMS int, dx, dy float64, now time.Time) (PlaymakerPassResult, error) {
	deviceID = strings.TrimSpace(deviceID)
	result := PlaymakerPassResult{SessionID: sessionID, Attempt: attempt}
	if eventID <= 0 || organizationID <= 0 || fanID < 0 || deviceID == "" || len(deviceID) > 200 {
		return result, sql.ErrNoRows
	}
	if attempt < 0 {
		return result, ErrPlaymakerGesture
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
	err = tx.QueryRow(`SELECT started_at,ends_at,last_resolved_at,next_attempt,fan_id FROM playmaker_sessions WHERE session_id=? AND event_id=? AND organization_id=? AND device_id=?`, sessionID, eventID, organizationID, deviceID).Scan(&started, &ends, &lastResolved, &result.NextAttempt, &owner)
	if err != nil {
		return result, err
	}
	if owner > 0 && owner != fanID {
		return result, sql.ErrNoRows
	}
	err = tx.QueryRow(`SELECT assisted,clean,reason FROM playmaker_passes WHERE session_id=? AND attempt=?`, sessionID, attempt).Scan(&result.Assisted, &result.Clean, &result.Reason)
	if err == sql.ErrNoRows {
		if err := basketballEventActive(tx, eventID, organizationID); err != nil {
			return result, err
		}
		if now.UnixMilli() > ends+15000 {
			return result, ErrPlaymakerExpired
		}
		// The grace window belongs to the single pass already in flight when
		// the countdown ends, rather than extending the playable round.
		if now.UnixMilli() > ends {
			var lateReceipts int
			if err := tx.QueryRow(`SELECT COUNT(*) FROM playmaker_passes WHERE session_id=? AND resolved_at>?`, sessionID, ends).Scan(&lateReceipts); err != nil {
				return result, err
			}
			if lateReceipts > 0 {
				return result, ErrPlaymakerExpired
			}
		}
		if attempt != result.NextAttempt {
			return result, ErrPlaymakerAttempt
		}
		if elapsedMS < 0 || elapsedMS > 30000 || int64(elapsedMS) > now.UnixMilli()-started+1500 {
			return result, ErrPlaymakerGesture
		}
		if lastResolved == 0 {
			lastResolved = started
		}
		if now.UnixMilli()-lastResolved < 1400 {
			return result, ErrPlaymakerTooEarly
		}
		assisted, clean, reason, err := evaluatePlaymakerPass(attempt, elapsedMS, dx, dy)
		if err != nil {
			return result, err
		}
		updated, err := tx.Exec(`UPDATE playmaker_sessions SET next_attempt=next_attempt+1,last_resolved_at=?,fan_id=? WHERE session_id=? AND next_attempt=?`, now.UnixMilli(), fanID, sessionID, attempt)
		if err != nil {
			return result, err
		}
		if n, err := updated.RowsAffected(); err != nil {
			return result, err
		} else if n != 1 {
			return result, ErrPlaymakerAttempt
		}
		if _, err := tx.Exec(`INSERT INTO playmaker_passes(session_id,attempt,event_id,organization_id,device_id,fan_id,elapsed_ms,assisted,clean,reason,resolved_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, sessionID, attempt, eventID, organizationID, deviceID, fanID, elapsedMS, assisted, clean, reason, now.UnixMilli()); err != nil {
			return result, err
		}
		result.Assisted, result.Clean, result.Reason = assisted, clean, reason
		result.NextAttempt = attempt + 1
		if assisted {
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
	if err := tx.QueryRow(`SELECT COUNT(*) FROM playmaker_passes WHERE event_id=? AND organization_id=? AND assisted=1 AND (device_id=? OR (?>0 AND fan_id=?))`, eventID, organizationID, deviceID, fanID, fanID).Scan(&result.PlaymakerEarned); err != nil {
		return result, err
	}
	result.RemainingMS = playmakerRemaining(ends, now.UnixMilli())
	if err := tx.Commit(); err != nil {
		return result, err
	}
	return result, nil
}
