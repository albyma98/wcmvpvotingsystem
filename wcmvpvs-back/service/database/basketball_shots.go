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
	ErrBasketballClosed   = errors.New("basketball event is closed")
	ErrBasketballExpired  = errors.New("basketball shot expired")
	ErrBasketballTooEarly = errors.New("basketball shot is still in flight")
	ErrBasketballGesture  = errors.New("invalid basketball gesture")
)

type BasketballHoop struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type BasketballShot struct {
	ShotID    string         `json:"shot_id"`
	Hoop      BasketballHoop `json:"hoop"`
	Attempt   int            `json:"attempt"`
	ExpiresAt string         `json:"expires_at"`
}

type BasketballShotResult struct {
	ShotID           string `json:"shot_id"`
	Made             bool   `json:"made"`
	Clean            bool   `json:"clean"`
	Wallet           int    `json:"wallet"`
	CoinsEarned      int    `json:"coins_earned"`
	BasketballEarned int    `json:"basketball_earned"`
}

func basketballHoopForAttempt(attempt int) BasketballHoop {
	hoops := [...]BasketballHoop{{180, 200}, {120, 205}, {240, 195}, {150, 215}, {225, 205}, {105, 195}, {255, 215}, {180, 185}}
	return hoops[attempt%len(hoops)]
}

// Coordinates match the 360x560 court used by the game. A drag aims from the ball
// (180,466) toward the issued hoop; the hoop's elliptical catch area is 30x40.
func basketballEvaluateShot(hoop BasketballHoop, dx, dy float64) (bool, bool, error) {
	if math.IsNaN(dx) || math.IsNaN(dy) || math.IsInf(dx, 0) || math.IsInf(dy, 0) {
		return false, false, ErrBasketballGesture
	}
	if math.Hypot(dx, dy) < 42 || dy > -30 {
		return false, false, ErrBasketballGesture
	}
	x, y := (180+dx-hoop.X)/30, (466+dy-hoop.Y)/40
	err := x*x + y*y
	return err <= 1, err <= 0.16, nil
}

func basketballEventActive(tx *sql.Tx, eventID, organizationID int) error {
	var active, concluded bool
	if err := tx.QueryRow(`SELECT is_active,is_concluded FROM events WHERE id=? AND organization_id=?`, eventID, organizationID).Scan(&active, &concluded); err != nil {
		return err
	}
	if !active || concluded {
		return ErrBasketballClosed
	}
	return nil
}

func basketballFanScope(tx *sql.Tx, organizationID, fanID int) error {
	if fanID == 0 {
		return nil
	}
	var id int
	return tx.QueryRow(`SELECT id FROM fan_profiles WHERE id=? AND organization_id=?`, fanID, organizationID).Scan(&id)
}

func (db *appdbimpl) CreateBasketballShot(eventID, organizationID, fanID int, deviceID string, now time.Time) (BasketballShot, error) {
	deviceID = strings.TrimSpace(deviceID)
	if eventID <= 0 || organizationID <= 0 || deviceID == "" || len(deviceID) > 200 || fanID < 0 {
		return BasketballShot{}, sql.ErrNoRows
	}
	tx, err := db.c.Begin()
	if err != nil {
		return BasketballShot{}, err
	}
	defer tx.Rollback()
	if err := basketballEventActive(tx, eventID, organizationID); err != nil {
		return BasketballShot{}, err
	}
	if err := basketballFanScope(tx, organizationID, fanID); err != nil {
		return BasketballShot{}, err
	}
	// The UI renews before a ball expires, leaving enough time to aim and finish
	// the flight without cycling through the same nearly-expired shot.
	if _, err := tx.Exec(`UPDATE basketball_shots SET status='expired' WHERE event_id=? AND organization_id=? AND device_id=? AND status='pending' AND expires_at<=?`, eventID, organizationID, deviceID, now.Add(10*time.Second).UnixMilli()); err != nil {
		return BasketballShot{}, err
	}
	var shot BasketballShot
	var expires int64
	var owner int
	err = tx.QueryRow(`SELECT shot_id,attempt,hoop_x,hoop_y,expires_at,fan_id FROM basketball_shots WHERE event_id=? AND organization_id=? AND device_id=? AND status='pending'`, eventID, organizationID, deviceID).Scan(&shot.ShotID, &shot.Attempt, &shot.Hoop.X, &shot.Hoop.Y, &expires, &owner)
	if err == nil && owner > 0 && owner != fanID {
		return BasketballShot{}, sql.ErrNoRows
	}
	if err == sql.ErrNoRows {
		if err := tx.QueryRow(`SELECT COALESCE(MAX(attempt)+1,0) FROM basketball_shots WHERE event_id=? AND organization_id=? AND device_id=?`, eventID, organizationID, deviceID).Scan(&shot.Attempt); err != nil {
			return BasketballShot{}, err
		}
		shot.ShotID = uuid.Must(uuid.NewV4()).String()
		shot.Hoop = basketballHoopForAttempt(shot.Attempt)
		expires = now.Add(2 * time.Minute).UnixMilli()
		_, err = tx.Exec(`INSERT INTO basketball_shots(shot_id,event_id,organization_id,device_id,fan_id,attempt,hoop_x,hoop_y,issued_at,expires_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, shot.ShotID, eventID, organizationID, deviceID, fanID, shot.Attempt, shot.Hoop.X, shot.Hoop.Y, now.UnixMilli(), expires)
	}
	if err != nil {
		return BasketballShot{}, err
	}
	shot.ExpiresAt = time.UnixMilli(expires).UTC().Format(time.RFC3339Nano)
	if err := tx.Commit(); err != nil {
		return BasketballShot{}, err
	}
	return shot, nil
}

// A shot receipt and its +1 wallet increment commit together. Replaying a shot
// returns its original result and the current balance without crediting again.
func (db *appdbimpl) ResolveBasketballShot(eventID, organizationID, fanID int, deviceID, shotID string, dx, dy float64, now time.Time) (BasketballShotResult, error) {
	deviceID = strings.TrimSpace(deviceID)
	result := BasketballShotResult{ShotID: shotID}
	if eventID <= 0 || organizationID <= 0 || deviceID == "" || fanID < 0 {
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
	var hoop BasketballHoop
	var issued, expires int64
	var status string
	var owner int
	err = tx.QueryRow(`SELECT hoop_x,hoop_y,issued_at,expires_at,status,clean,fan_id FROM basketball_shots WHERE shot_id=? AND event_id=? AND organization_id=? AND device_id=?`, shotID, eventID, organizationID, deviceID).Scan(&hoop.X, &hoop.Y, &issued, &expires, &status, &result.Clean, &owner)
	if err != nil {
		return result, err
	}
	if owner > 0 && owner != fanID {
		return result, sql.ErrNoRows
	}
	if status == "pending" {
		if err := basketballEventActive(tx, eventID, organizationID); err != nil {
			return result, err
		}
		if now.UnixMilli() >= expires {
			return result, ErrBasketballExpired
		}
		if now.UnixMilli()-issued < 600 {
			return result, ErrBasketballTooEarly
		}
		made, clean, err := basketballEvaluateShot(hoop, dx, dy)
		if err != nil {
			return result, err
		}
		status = "miss"
		if made {
			status = "made"
		}
		changed, err := tx.Exec(`UPDATE basketball_shots SET status=?,clean=?,fan_id=? WHERE shot_id=? AND status='pending'`, status, clean, fanID, shotID)
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
		result.Clean = clean
		if made {
			if fanID > 0 {
				_, err = economyExec(tx, organizationID, eventID, fanID, deviceID, "Canestro Basket Rush", "basketball", `INSERT INTO fan_wallets(fan_id,coins) VALUES(?,1) ON CONFLICT(fan_id) DO UPDATE SET coins=fan_wallets.coins+1,updated_at=CURRENT_TIMESTAMP`, fanID)
			} else {
				_, err = economyExec(tx, organizationID, eventID, 0, deviceID, "Canestro Basket Rush", "basketball", `INSERT INTO guest_wallets(event_id,organization_id,device_id,coins) VALUES(?,?,?,1) ON CONFLICT(event_id,organization_id,device_id) DO UPDATE SET coins=guest_wallets.coins+1,updated_at=CURRENT_TIMESTAMP`, eventID, organizationID, deviceID)
			}
			if err != nil {
				return result, err
			}
			result.CoinsEarned = 1
		}
	} else if status == "expired" {
		return result, ErrBasketballExpired
	}
	result.Made = status == "made"
	if fanID > 0 {
		err = tx.QueryRow(`SELECT coins FROM fan_wallets WHERE fan_id=?`, fanID).Scan(&result.Wallet)
	} else {
		err = tx.QueryRow(`SELECT coins FROM guest_wallets WHERE event_id=? AND organization_id=? AND device_id=?`, eventID, organizationID, deviceID).Scan(&result.Wallet)
	}
	if err != nil && err != sql.ErrNoRows {
		return result, err
	}
	if err := tx.QueryRow(`SELECT COUNT(*) FROM basketball_shots WHERE event_id=? AND organization_id=? AND status='made' AND ((device_id=? AND fan_id=0) OR (?>0 AND fan_id=?))`, eventID, organizationID, deviceID, fanID, fanID).Scan(&result.BasketballEarned); err != nil {
		return result, err
	}
	if err := tx.Commit(); err != nil {
		return result, err
	}
	return result, nil
}
