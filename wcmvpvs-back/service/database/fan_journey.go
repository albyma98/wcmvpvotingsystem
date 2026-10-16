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
	rows, err := db.c.Query(`SELECT reward_key,MAX(coins) FROM fan_journey_rewards WHERE event_id=? AND organization_id=? AND (device_id=? OR (?>0 AND fan_id=?)) GROUP BY reward_key`, eventID, organizationID, deviceID, fanID, fanID)
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
	if err := db.c.QueryRow(`SELECT COUNT(*) FROM basketball_shots WHERE event_id=? AND organization_id=? AND status='made' AND (device_id=? OR (?>0 AND fan_id=?))`, eventID, organizationID, deviceID, fanID, fanID).Scan(&basketball); err != nil {
		return nil, err
	}
	if basketball > 0 {
		rewards["basketball"] = basketball
	}
	var playmaker int
	if err := db.c.QueryRow(`SELECT COUNT(*) FROM playmaker_passes WHERE event_id=? AND organization_id=? AND assisted=1 AND (device_id=? OR (?>0 AND fan_id=?))`, eventID, organizationID, deviceID, fanID, fanID).Scan(&playmaker); err != nil {
		return nil, err
	}
	if playmaker > 0 {
		rewards["playmaker"] = playmaker
	}
	return rewards, nil
}

// Ledger insertion and wallet increment commit together. A retry returns the current balance.
func (db *appdbimpl) ClaimFanJourneyReward(eventID, organizationID, fanID int, deviceID, rewardKey string) (int, int, error) {
	amount := 0
	switch rewardKey {
	case "vote":
		amount = 10
	case "quiz":
		amount = 15
	default:
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
	if exists == 0 {
		var count int
		if rewardKey == "vote" {
			if err := tx.QueryRow(`SELECT COUNT(*) FROM votes v JOIN events e ON e.id=v.event_id WHERE v.event_id=? AND e.organization_id=? AND v.device_id=?`, eventID, organizationID, deviceID).Scan(&count); err != nil {
				return 0, 0, err
			}
			if count == 0 {
				return 0, 0, sql.ErrNoRows
			}
		} else {
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
		}
		if _, err := tx.Exec(`INSERT INTO fan_journey_rewards(event_id,organization_id,device_id,fan_id,reward_key,coins) VALUES(?,?,?,?,?,?)`, eventID, organizationID, deviceID, fanID, rewardKey, amount); err != nil {
			return 0, 0, err
		}
		if fanID > 0 {
			_, err = tx.Exec(`INSERT INTO fan_wallets(fan_id,coins) VALUES(?,?) ON CONFLICT(fan_id) DO UPDATE SET coins=fan_wallets.coins+excluded.coins,updated_at=CURRENT_TIMESTAMP`, fanID, amount)
		} else {
			_, err = tx.Exec(`INSERT INTO guest_wallets(event_id,organization_id,device_id,coins) VALUES(?,?,?,?) ON CONFLICT(event_id,organization_id,device_id) DO UPDATE SET coins=guest_wallets.coins+excluded.coins,updated_at=CURRENT_TIMESTAMP`, eventID, organizationID, deviceID, amount)
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
