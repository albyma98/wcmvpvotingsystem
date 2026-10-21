package database

import (
	"database/sql"
	"errors"
	"regexp"
	"strings"
)

var ErrFanMinigameRound = errors.New("invalid minigame round")
var minigameRoundPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,128}$`)

func ValidFanMinigameRoundID(id string) bool { return minigameRoundPattern.MatchString(id) }

type FanMinigameReward struct {
	Wallet       int `json:"wallet"`
	CoinsEarned  int `json:"coins_earned"`
	RewardAmount int `json:"reward_amount"`
	RewardTotal  int `json:"reward_total"`
}

func (db *appdbimpl) RecordFanMinigameQuizAnswer(eventID, organizationID int, deviceID, requestID string, questionID int) error {
	if !ValidFanMinigameRoundID(requestID) || strings.TrimSpace(deviceID) == "" {
		return ErrFanMinigameRound
	}
	_, err := db.c.Exec(`INSERT OR IGNORE INTO fan_minigame_quiz_answers(event_id,organization_id,device_id,request_id,question_id) VALUES(?,?,?,?,?)`, eventID, organizationID, deviceID, requestID, questionID)
	return err
}

// A new completed round earns its fixed reward. Retries of that round share a
// receipt, even after login or on another device; receipt and wallet commit together.
func (db *appdbimpl) ClaimFanMinigameReward(eventID, organizationID, fanID int, deviceID, rewardKey, requestID string) (FanMinigameReward, error) {
	result := FanMinigameReward{RewardAmount: FanJourneyRewardAmount(rewardKey)}
	deviceID = strings.TrimSpace(deviceID)
	if result.RewardAmount == 0 || rewardKey == "vote" || !ValidFanMinigameRoundID(requestID) || deviceID == "" || len(deviceID) > 200 || fanID < 0 {
		return result, ErrFanMinigameRound
	}
	tx, err := db.c.Begin()
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	if err := basketballFanScope(tx, organizationID, fanID); err != nil {
		return result, err
	}
	var owner int
	var device, key string
	err = tx.QueryRow(`SELECT fan_id,device_id,reward_key FROM fan_minigame_rewards WHERE event_id=? AND organization_id=? AND request_id=?`, eventID, organizationID, requestID).Scan(&owner, &device, &key)
	if err == nil {
		if key != rewardKey {
			return result, ErrFanMinigameRound
		}
		if !(fanID > 0 && owner == fanID) && !(fanID == 0 && owner == 0 && device == deviceID) {
			return result, ErrFanRewardDeviceUsed
		}
	} else if errors.Is(err, sql.ErrNoRows) {
		if err := basketballEventActive(tx, eventID, organizationID); err != nil {
			return result, err
		}
		if rewardKey == "quiz" {
			var limit int
			var enabled bool
			if err := tx.QueryRow(`SELECT questions_per_session,enabled FROM event_quiz WHERE event_id=?`, eventID).Scan(&limit, &enabled); err != nil {
				return result, err
			}
			if !enabled {
				return result, sql.ErrNoRows
			}
			var total, answered int
			if err := tx.QueryRow(`SELECT COUNT(*),COALESCE(SUM(CASE WHEN EXISTS(SELECT 1 FROM fan_minigame_quiz_answers a WHERE a.event_id=? AND a.organization_id=? AND a.device_id=? AND a.request_id=? AND a.question_id=q.id) THEN 1 ELSE 0 END),0) FROM (SELECT id FROM event_quiz_questions WHERE quiz_id=? ORDER BY order_index,id LIMIT ?) q`, eventID, organizationID, deviceID, requestID, eventID, limit).Scan(&total, &answered); err != nil {
				return result, err
			}
			if total == 0 || total != answered {
				return result, sql.ErrNoRows
			}
		}
		// These local games declare completion. Amounts are always chosen by the
		// server; only a distinct round can produce another wallet increment.
		if _, err := tx.Exec(`INSERT INTO fan_minigame_rewards(event_id,organization_id,request_id,device_id,fan_id,reward_key,coins) VALUES(?,?,?,?,?,?,?)`, eventID, organizationID, requestID, deviceID, fanID, rewardKey, result.RewardAmount); err != nil {
			return result, err
		}
		if fanID > 0 {
			_, err = economyExec(tx, organizationID, eventID, fanID, deviceID, "Partita "+rewardKey, "minigame:"+rewardKey, `INSERT INTO fan_wallets(fan_id,coins) VALUES(?,?) ON CONFLICT(fan_id) DO UPDATE SET coins=fan_wallets.coins+excluded.coins,updated_at=CURRENT_TIMESTAMP`, fanID, result.RewardAmount)
		} else {
			_, err = economyExec(tx, organizationID, eventID, 0, deviceID, "Partita "+rewardKey, "minigame:"+rewardKey, `INSERT INTO guest_wallets(event_id,organization_id,device_id,coins) VALUES(?,?,?,?) ON CONFLICT(event_id,organization_id,device_id) DO UPDATE SET coins=guest_wallets.coins+excluded.coins,updated_at=CURRENT_TIMESTAMP`, eventID, organizationID, deviceID, result.RewardAmount)
		}
		if err != nil {
			return result, err
		}
		result.CoinsEarned = result.RewardAmount
	} else {
		return result, err
	}
	if fanID > 0 {
		err = tx.QueryRow(`SELECT coins FROM fan_wallets WHERE fan_id=?`, fanID).Scan(&result.Wallet)
	} else {
		err = tx.QueryRow(`SELECT coins FROM guest_wallets WHERE event_id=? AND organization_id=? AND device_id=?`, eventID, organizationID, deviceID).Scan(&result.Wallet)
	}
	if err != nil {
		return result, err
	}
	if err := tx.QueryRow(`SELECT COALESCE((SELECT MAX(coins) FROM fan_journey_rewards WHERE event_id=? AND organization_id=? AND reward_key=? AND ((fan_id=0 AND device_id=?) OR (?>0 AND fan_id=?))),0)+COALESCE((SELECT SUM(coins) FROM fan_minigame_rewards WHERE event_id=? AND organization_id=? AND reward_key=? AND ((fan_id=0 AND device_id=?) OR (?>0 AND fan_id=?))),0)`, eventID, organizationID, rewardKey, deviceID, fanID, fanID, eventID, organizationID, rewardKey, deviceID, fanID, fanID).Scan(&result.RewardTotal); err != nil {
		return result, err
	}
	if rewardKey == "tap" && fanID > 0 {
		var live int
		if err := tx.QueryRow(`SELECT COALESCE(SUM(CASE WHEN fan1_id=? THEN fan1_coins ELSE 0 END+CASE WHEN fan2_id=? THEN fan2_coins ELSE 0 END),0) FROM tap_live_matches WHERE event_id=? AND organization_id=? AND status='finished' AND (fan1_id=? OR fan2_id=?)`, fanID, fanID, eventID, organizationID, fanID, fanID).Scan(&live); err != nil {
			return result, err
		}
		result.RewardTotal += live
	}
	if err := tx.Commit(); err != nil {
		return result, err
	}
	return result, nil
}
