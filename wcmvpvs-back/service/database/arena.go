package database

import (
	"encoding/json"
	"time"
)

// ArenaFan intentionally contains only the public identity of a completed profile.
type ArenaFan struct {
	ID        int               `json:"id"`
	Nickname  string            `json:"nickname"`
	IsMe      bool              `json:"is_me"`
	Public    bool              `json:"public"`
	Slot      int               `json:"slot"`
	Equipment map[string]string `json:"equipment"`
}

func (db *appdbimpl) GetArenaFans(organizationID int) ([]ArenaFan, error) {
	if organizationID <= 0 {
		return []ArenaFan{}, nil
	}
	if _, err := db.ArenaActivities(organizationID, 0, ArenaCommand{Action: "state"}, time.Now()); err != nil {
		return nil, err
	}
	rows, err := db.c.Query(`SELECT p.id,CASE WHEN m.public=1 THEN p.nickname ELSE '' END,m.public,m.slot,m.equipment FROM fan_profiles p JOIN arena_members m ON m.org=p.organization_id AND m.fan=p.id WHERE p.organization_id=? AND p.registration_pending=0 AND p.accepted_terms=1 ORDER BY p.id`, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	fans := []ArenaFan{}
	for rows.Next() {
		var fan ArenaFan
		var equipment string
		if err := rows.Scan(&fan.ID, &fan.Nickname, &fan.Public, &fan.Slot, &equipment); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(equipment), &fan.Equipment); err != nil {
			return nil, err
		}
		fans = append(fans, fan)
	}
	return fans, rows.Err()
}
