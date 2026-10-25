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
	ErrVolleyballClosed   = errors.New("volleyball event is closed")
	ErrVolleyballExpired  = errors.New("volleyball round expired")
	ErrVolleyballTooEarly = errors.New("volleyball round is still in flight")
	ErrVolleyballGesture  = errors.New("invalid volleyball gesture")
	ErrVolleyballGame     = errors.New("invalid volleyball game")
)

type VolleyballPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type VolleyballRound struct {
	RoundID      string          `json:"round_id"`
	GameID       string          `json:"game_id"`
	Attempt      int             `json:"attempt"`
	Target       VolleyballPoint `json:"target"`
	Blocker      VolleyballPoint `json:"blocker"`
	TimingTarget float64         `json:"timing_target"`
	ExpiresAt    string          `json:"expires_at"`
}

type VolleyballResult struct {
	RoundID     string `json:"round_id"`
	GameID      string `json:"game_id"`
	Success     bool   `json:"success"`
	Perfect     bool   `json:"perfect"`
	Wallet      int    `json:"wallet"`
	CoinsEarned int    `json:"coins_earned"`
	GameEarned  int    `json:"game_earned"`
}

func ValidVolleyballGame(gameID string) bool {
	return gameID == "volley-rush" || gameID == "setter-pro" || gameID == "monster-block"
}

func ensureVolleyballTables(conn *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS volleyball_rounds (
			round_id TEXT PRIMARY KEY, event_id INTEGER NOT NULL, organization_id INTEGER NOT NULL,
			device_id TEXT NOT NULL, fan_id INTEGER NOT NULL DEFAULT 0, game_id TEXT NOT NULL,
			attempt INTEGER NOT NULL, target_x REAL NOT NULL, target_y REAL NOT NULL,
			blocker_x REAL NOT NULL, blocker_y REAL NOT NULL, timing_target REAL NOT NULL,
			issued_at INTEGER NOT NULL, expires_at INTEGER NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending', perfect INTEGER NOT NULL DEFAULT 0,
			UNIQUE(event_id,organization_id,device_id,game_id,attempt))`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_volleyball_pending ON volleyball_rounds(event_id,organization_id,device_id,game_id) WHERE status='pending'`,
		`CREATE INDEX IF NOT EXISTS idx_volleyball_rewards ON volleyball_rounds(event_id,organization_id,game_id,fan_id,device_id,status)`,
	}
	for _, statement := range statements {
		if _, err := conn.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

// All coordinates use the same 360x560 court as the client. Targets cycle by
// attempt, so reopening the game never changes the target of a pending round.
func volleyballTargetForAttempt(gameID string, attempt int) (VolleyballPoint, VolleyballPoint) {
	xs := [...]float64{82, 278, 116, 244, 156, 204, 92, 268}
	i := attempt % len(xs)
	switch gameID {
	case "volley-rush":
		return VolleyballPoint{xs[i], 136 + float64(i%3)*28}, VolleyballPoint{360 - xs[i], 285}
	case "setter-pro":
		return VolleyballPoint{xs[(i+2)%len(xs)], 104 + float64(i%4)*18}, VolleyballPoint{180, 285}
	default:
		return VolleyballPoint{xs[(i+1)%len(xs)], 266 + float64(i%3)*9}, VolleyballPoint{180, 180}
	}
}

// A successful action requires both accurate aim and a timed contact. A perfect
// action is cosmetic: every success earns exactly one coin.
func volleyballEvaluateRound(gameID string, target, blocker VolleyballPoint, timingTarget, aimX, aimY, timing float64) (bool, bool, error) {
	if !ValidVolleyballGame(gameID) {
		return false, false, ErrVolleyballGame
	}
	for _, value := range []float64{aimX, aimY, timing} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return false, false, ErrVolleyballGesture
		}
	}
	if aimX < 0 || aimX > 360 || aimY < 0 || aimY > 560 || timing < 0 || timing > 1 {
		return false, false, ErrVolleyballGesture
	}
	rx, ry, window := 48.0, 40.0, 0.24
	switch gameID {
	case "setter-pro":
		rx, ry, window = 34, 30, 0.20
	case "monster-block":
		rx, ry, window = 44, 34, 0.16
	}
	dx, dy := (aimX-target.X)/rx, (aimY-target.Y)/ry
	distance := dx*dx + dy*dy
	timingError := math.Abs(timing - timingTarget)
	success := distance <= 1 && timingError <= window+1e-9
	if gameID == "volley-rush" {
		// Trace the spike from the contact point through the net. Even a ball
		// aimed into the target cannot score if its path crosses the blocker.
		if aimY >= blocker.Y {
			success = false
		} else {
			netX := 180 + (aimX-180)*(466-blocker.Y)/(466-aimY)
			if math.Abs(netX-blocker.X) <= 30 {
				success = false
			}
		}
	}
	return success, success && distance <= 0.16 && timingError <= 0.055+1e-9, nil
}

func volleyballEventActive(tx *sql.Tx, eventID, organizationID int) error {
	var active, concluded bool
	if err := tx.QueryRow(`SELECT is_active,is_concluded FROM events WHERE id=? AND organization_id=?`, eventID, organizationID).Scan(&active, &concluded); err != nil {
		return err
	}
	if !active || concluded {
		return ErrVolleyballClosed
	}
	return nil
}

func (db *appdbimpl) CreateVolleyballRound(eventID, organizationID, fanID int, deviceID, gameID string, now time.Time) (VolleyballRound, error) {
	if !ValidVolleyballGame(gameID) {
		return VolleyballRound{}, ErrVolleyballGame
	}
	deviceID = strings.TrimSpace(deviceID)
	if eventID <= 0 || organizationID <= 0 || fanID < 0 || deviceID == "" || len(deviceID) > 200 {
		return VolleyballRound{}, sql.ErrNoRows
	}
	tx, err := db.c.Begin()
	if err != nil {
		return VolleyballRound{}, err
	}
	defer tx.Rollback()
	if err := volleyballEventActive(tx, eventID, organizationID); err != nil {
		return VolleyballRound{}, err
	}
	if err := basketballFanScope(tx, organizationID, fanID); err != nil {
		return VolleyballRound{}, err
	}
	// Leave enough time to complete the aim and timed contact on a fresh ball.
	if _, err := tx.Exec(`UPDATE volleyball_rounds SET status='expired' WHERE event_id=? AND organization_id=? AND device_id=? AND game_id=? AND status='pending' AND expires_at<=?`, eventID, organizationID, deviceID, gameID, now.Add(10*time.Second).UnixMilli()); err != nil {
		return VolleyballRound{}, err
	}
	round := VolleyballRound{GameID: gameID, TimingTarget: 0.7}
	var expires int64
	var owner int
	err = tx.QueryRow(`SELECT round_id,attempt,target_x,target_y,blocker_x,blocker_y,timing_target,expires_at,fan_id FROM volleyball_rounds WHERE event_id=? AND organization_id=? AND device_id=? AND game_id=? AND status='pending'`, eventID, organizationID, deviceID, gameID).Scan(&round.RoundID, &round.Attempt, &round.Target.X, &round.Target.Y, &round.Blocker.X, &round.Blocker.Y, &round.TimingTarget, &expires, &owner)
	if err == nil && owner > 0 && owner != fanID {
		return VolleyballRound{}, sql.ErrNoRows
	}
	if err == sql.ErrNoRows {
		if err := tx.QueryRow(`SELECT COALESCE(MAX(attempt)+1,0) FROM volleyball_rounds WHERE event_id=? AND organization_id=? AND device_id=? AND game_id=?`, eventID, organizationID, deviceID, gameID).Scan(&round.Attempt); err != nil {
			return VolleyballRound{}, err
		}
		round.RoundID = uuid.Must(uuid.NewV4()).String()
		round.Target, round.Blocker = volleyballTargetForAttempt(gameID, round.Attempt)
		expires = now.Add(2 * time.Minute).UnixMilli()
		_, err = tx.Exec(`INSERT INTO volleyball_rounds(round_id,event_id,organization_id,device_id,fan_id,game_id,attempt,target_x,target_y,blocker_x,blocker_y,timing_target,issued_at,expires_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, round.RoundID, eventID, organizationID, deviceID, fanID, gameID, round.Attempt, round.Target.X, round.Target.Y, round.Blocker.X, round.Blocker.Y, round.TimingTarget, now.UnixMilli(), expires)
	}
	if err != nil {
		return VolleyballRound{}, err
	}
	round.ExpiresAt = time.UnixMilli(expires).UTC().Format(time.RFC3339Nano)
	if err := tx.Commit(); err != nil {
		return VolleyballRound{}, err
	}
	return round, nil
}

func volleyballRewardLabel(gameID string) string {
	switch gameID {
	case "volley-rush":
		return "Punto Volley Rush"
	case "setter-pro":
		return "Alzata Setter Pro"
	default:
		return "Muro Monster Block"
	}
}

// The receipt, wallet credit and economy entry commit in one transaction.
// Retrying returns the saved outcome with no second reward, including when the
// registered fan resumes on another device.
func (db *appdbimpl) ResolveVolleyballRound(eventID, organizationID, fanID int, deviceID, gameID, roundID string, aimX, aimY, timing float64, now time.Time) (VolleyballResult, error) {
	result := VolleyballResult{RoundID: roundID, GameID: gameID}
	if !ValidVolleyballGame(gameID) {
		return result, ErrVolleyballGame
	}
	deviceID = strings.TrimSpace(deviceID)
	if eventID <= 0 || organizationID <= 0 || fanID < 0 || deviceID == "" || len(deviceID) > 200 || len(roundID) != 36 {
		return result, sql.ErrNoRows
	}
	tx, err := db.c.Begin()
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	if err := basketballFanScope(tx, organizationID, fanID); err != nil {
		return result, err
	}
	var target, blocker VolleyballPoint
	var timingTarget float64
	var issued, expires int64
	var status string
	var roundDevice string
	err = tx.QueryRow(`SELECT target_x,target_y,blocker_x,blocker_y,timing_target,issued_at,expires_at,status,perfect,device_id FROM volleyball_rounds WHERE round_id=? AND event_id=? AND organization_id=? AND game_id=? AND ((device_id=? AND fan_id=0) OR (?>0 AND fan_id=?))`, roundID, eventID, organizationID, gameID, deviceID, fanID, fanID).Scan(&target.X, &target.Y, &blocker.X, &blocker.Y, &timingTarget, &issued, &expires, &status, &result.Perfect, &roundDevice)
	if err != nil {
		return result, err
	}
	if status == "pending" {
		if err := volleyballEventActive(tx, eventID, organizationID); err != nil {
			return result, err
		}
		if now.UnixMilli() >= expires {
			return result, ErrVolleyballExpired
		}
		if now.UnixMilli()-issued < 600 {
			return result, ErrVolleyballTooEarly
		}
		success, perfect, err := volleyballEvaluateRound(gameID, target, blocker, timingTarget, aimX, aimY, timing)
		if err != nil {
			return result, err
		}
		status = "miss"
		if success {
			status = "success"
		}
		changed, err := tx.Exec(`UPDATE volleyball_rounds SET status=?,perfect=?,fan_id=? WHERE round_id=? AND status='pending'`, status, perfect, fanID, roundID)
		if err != nil {
			return result, err
		}
		n, err := changed.RowsAffected()
		if err != nil {
			return result, err
		}
		if n != 1 {
			return result, sql.ErrNoRows
		}
		result.Perfect = perfect
		if success {
			if fanID > 0 {
				_, err = economyExec(tx, organizationID, eventID, fanID, roundDevice, volleyballRewardLabel(gameID), gameID, `INSERT INTO fan_wallets(fan_id,coins) VALUES(?,1) ON CONFLICT(fan_id) DO UPDATE SET coins=fan_wallets.coins+1,updated_at=CURRENT_TIMESTAMP`, fanID)
			} else {
				_, err = economyExec(tx, organizationID, eventID, 0, deviceID, volleyballRewardLabel(gameID), gameID, `INSERT INTO guest_wallets(event_id,organization_id,device_id,coins) VALUES(?,?,?,1) ON CONFLICT(event_id,organization_id,device_id) DO UPDATE SET coins=guest_wallets.coins+1,updated_at=CURRENT_TIMESTAMP`, eventID, organizationID, deviceID)
			}
			if err != nil {
				return result, err
			}
			result.CoinsEarned = 1
		}
	} else if status == "expired" {
		return result, ErrVolleyballExpired
	}
	result.Success = status == "success"
	if fanID > 0 {
		err = tx.QueryRow(`SELECT coins FROM fan_wallets WHERE fan_id=?`, fanID).Scan(&result.Wallet)
	} else {
		err = tx.QueryRow(`SELECT coins FROM guest_wallets WHERE event_id=? AND organization_id=? AND device_id=?`, eventID, organizationID, deviceID).Scan(&result.Wallet)
	}
	if err != nil && err != sql.ErrNoRows {
		return result, err
	}
	if err := tx.QueryRow(`SELECT COUNT(*) FROM volleyball_rounds WHERE event_id=? AND organization_id=? AND game_id=? AND status='success' AND ((device_id=? AND fan_id=0) OR (?>0 AND fan_id=?))`, eventID, organizationID, gameID, deviceID, fanID, fanID).Scan(&result.GameEarned); err != nil {
		return result, err
	}
	if err := tx.Commit(); err != nil {
		return result, err
	}
	return result, nil
}
