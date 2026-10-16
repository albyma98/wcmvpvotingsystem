package database

import (
	"database/sql"
	"strings"
)

func (db *appdbimpl) RecordFanJourneyQuizAnswer(eventID, organizationID int, deviceID string, questionID int) error {
	_, err := db.c.Exec(`INSERT OR IGNORE INTO fan_journey_quiz_answers(event_id,organization_id,device_id,question_id) VALUES(?,?,?,?)`, eventID, organizationID, strings.TrimSpace(deviceID), questionID)
	return err
}

func (db *appdbimpl) ListFanJourneyRewards(eventID, organizationID, fanID int, deviceID string) (map[string]int, error) {
	rows, err := db.c.Query(`SELECT reward_key,MAX(coins) FROM fan_journey_rewards WHERE event_id=? AND organization_id=? AND ((device_id=? AND fan_id=0) OR (?>0 AND fan_id=?)) GROUP BY reward_key`, eventID, organizationID, deviceID, fanID, fanID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	rewards := map[string]int{}
	for rows.Next() {
		var key string
		var coins int
		if err := rows.Scan(&key, &coins); err != nil {
			return nil, err
		}
		rewards[key] = coins
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	var basketball int
	if err := db.c.QueryRow(`SELECT COUNT(*) FROM basketball_shots WHERE event_id=? AND organization_id=? AND status='made' AND ((device_id=? AND fan_id=0) OR (?>0 AND fan_id=?))`, eventID, organizationID, deviceID, fanID, fanID).Scan(&basketball); err != nil {
		return nil, err
	}
	if basketball > 0 {
		rewards["basketball"] = basketball
	}
	var playmaker int
	if err := db.c.QueryRow(`SELECT COUNT(*) FROM playmaker_passes WHERE event_id=? AND organization_id=? AND assisted=1 AND ((device_id=? AND fan_id=0) OR (?>0 AND fan_id=?))`, eventID, organizationID, deviceID, fanID, fanID).Scan(&playmaker); err != nil {
		return nil, err
	}
	if playmaker > 0 {
		rewards["playmaker"] = playmaker
	}
	var crossover int
	if err := db.c.QueryRow(`SELECT COUNT(*) FROM crossover_duels WHERE event_id=? AND organization_id=? AND won=1 AND ((device_id=? AND fan_id=0) OR (?>0 AND fan_id=?))`, eventID, organizationID, deviceID, fanID, fanID).Scan(&crossover); err != nil {
		return nil, err
	}
	if crossover > 0 {
		rewards["crossover"] = crossover
	}
	if fanID > 0 {
		var liveTap int
		if err := db.c.QueryRow(`SELECT COALESCE(SUM(CASE WHEN fan1_id=? THEN fan1_coins ELSE 0 END+CASE WHEN fan2_id=? THEN fan2_coins ELSE 0 END),0) FROM tap_live_matches WHERE event_id=? AND organization_id=? AND status='finished' AND (fan1_id=? OR fan2_id=?)`, fanID, fanID, eventID, organizationID, fanID, fanID).Scan(&liveTap); err != nil {
			return nil, err
		}
		if liveTap > 0 {
			rewards["tap"] += liveTap
		}
	}
	return rewards, nil
}

// Ledger insertion and wallet increment commit together. A retry returns the current balance.
func FanJourneyRewardAmount(key string) int {
	switch key {
	case "vote", "reaction":
		return 10
	case "quiz":
		return 15
	case "tap", "memory-flash":
		return 8
	case "sponsor-rush":
		return 12
	}
	return 0
}

func (db *appdbimpl) ClaimFanJourneyReward(eventID, organizationID, fanID int, deviceID, rewardKey string) (int, int, error) {
	amount := FanJourneyRewardAmount(rewardKey)
	if amount == 0 {
		return 0, 0, sql.ErrNoRows
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return 0, 0, sql.ErrNoRows
	}
	tx, err := db.c.Begin()
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()
	var exists int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM fan_journey_rewards WHERE event_id=? AND organization_id=? AND reward_key=? AND (device_id=? OR (?>0 AND fan_id=?))`, eventID, organizationID, rewardKey, deviceID, fanID, fanID).Scan(&exists); err != nil {
		return 0, 0, err
	}
	credited := 0
	if exists > 0 {
		var owned int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM fan_journey_rewards WHERE event_id=? AND organization_id=? AND reward_key=? AND ((device_id=? AND fan_id=0) OR (?>0 AND fan_id=?))`, eventID, organizationID, rewardKey, deviceID, fanID, fanID).Scan(&owned); err != nil {
			return 0, 0, err
		}
		if owned == 0 {
			return 0, 0, ErrFanRewardDeviceUsed
		}
	}
	if exists == 0 {
		var count int
		if rewardKey == "vote" {
			if err := tx.QueryRow(`SELECT COUNT(*) FROM votes v JOIN events e ON e.id=v.event_id WHERE v.event_id=? AND e.organization_id=? AND v.device_id=?`, eventID, organizationID, deviceID).Scan(&count); err != nil {
				return 0, 0, err
			}
			if count == 0 {
				return 0, 0, sql.ErrNoRows
			}
		} else if rewardKey == "quiz" {
			var limit int
			if err := tx.QueryRow(`SELECT questions_per_session FROM event_quiz WHERE event_id=?`, eventID).Scan(&limit); err != nil {
				return 0, 0, err
			}
			var total, answered int
			if err := tx.QueryRow(`SELECT COUNT(*),COALESCE(SUM(CASE WHEN EXISTS(SELECT 1 FROM fan_journey_quiz_answers a WHERE a.event_id=? AND a.organization_id=? AND a.device_id=? AND a.question_id=q.id) THEN 1 ELSE 0 END),0)
			FROM (SELECT id FROM event_quiz_questions WHERE quiz_id=? ORDER BY order_index,id LIMIT ?) q`, eventID, organizationID, deviceID, eventID, limit).Scan(&total, &answered); err != nil {
				return 0, 0, err
			}
			if total == 0 || total != answered {
				return 0, 0, sql.ErrNoRows
			}
		} else if rewardKey == "reaction" {
			if err := tx.QueryRow(`SELECT COUNT(*) FROM events WHERE id=? AND organization_id=? AND is_active=1 AND is_concluded=0`, eventID, organizationID).Scan(&count); err != nil {
				return 0, 0, err
			}
			if count == 0 {
				return 0, 0, sql.ErrNoRows
			}
			var hasTable int
			if err := tx.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='reaction_tests'`).Scan(&hasTable); err != nil {
				return 0, 0, err
			}
			if hasTable > 0 {
				var attempts, valid int
				if err := tx.QueryRow(`SELECT COUNT(*),COALESCE(MAX(is_valid),0) FROM reaction_tests WHERE event_id=? AND device_id=?`, eventID, deviceID).Scan(&attempts, &valid); err != nil {
					return 0, 0, err
				}
				if attempts > 0 && valid == 0 {
					return 0, 0, sql.ErrNoRows
				}
			}
		} else {
			// Legacy local games declare completion, but the server controls the
			// fixed reward and its once-per-event receipt; no balance is accepted.
			if err := tx.QueryRow(`SELECT COUNT(*) FROM events WHERE id=? AND organization_id=? AND is_active=1 AND is_concluded=0`, eventID, organizationID).Scan(&count); err != nil {
				return 0, 0, err
			}
			if count == 0 {
				return 0, 0, sql.ErrNoRows
			}
		}
		if _, err := tx.Exec(`INSERT INTO fan_journey_rewards(event_id,organization_id,device_id,fan_id,reward_key,coins) VALUES(?,?,?,?,?,?)`, eventID, organizationID, deviceID, fanID, rewardKey, amount); err != nil {
			return 0, 0, err
		}
		if fanID > 0 {
			_, err = economyExec(tx, organizationID, eventID, fanID, deviceID, "Bonus "+rewardKey, "journey:"+rewardKey, `INSERT INTO fan_wallets(fan_id,coins) VALUES(?,?) ON CONFLICT(fan_id) DO UPDATE SET coins=fan_wallets.coins+excluded.coins,updated_at=CURRENT_TIMESTAMP`, fanID, amount)
		} else {
			_, err = economyExec(tx, organizationID, eventID, 0, deviceID, "Bonus "+rewardKey, "journey:"+rewardKey, `INSERT INTO guest_wallets(event_id,organization_id,device_id,coins) VALUES(?,?,?,?) ON CONFLICT(event_id,organization_id,device_id) DO UPDATE SET coins=guest_wallets.coins+excluded.coins,updated_at=CURRENT_TIMESTAMP`, eventID, organizationID, deviceID, amount)
		}
		if err != nil {
			return 0, 0, err
		}
		credited = amount
	}
	var wallet int
	if fanID > 0 {
		err = tx.QueryRow(`SELECT coins FROM fan_wallets WHERE fan_id=?`, fanID).Scan(&wallet)
	} else {
		err = tx.QueryRow(`SELECT coins FROM guest_wallets WHERE event_id=? AND organization_id=? AND device_id=?`, eventID, organizationID, deviceID).Scan(&wallet)
	}
	if err != nil {
		return 0, 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}
	return wallet, credited, nil
}
