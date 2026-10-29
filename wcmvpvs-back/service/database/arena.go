package database

// ArenaFan intentionally contains only the public identity of a completed profile.
type ArenaFan struct {
	ID       int    `json:"id"`
	Nickname string `json:"nickname"`
	IsMe     bool   `json:"is_me"`
}

func (db *appdbimpl) GetArenaFans(organizationID int) ([]ArenaFan, error) {
	rows, err := db.c.Query(`SELECT id, nickname FROM fan_profiles WHERE organization_id=? AND registration_pending=0 AND accepted_terms=1 ORDER BY id`, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	fans := []ArenaFan{}
	for rows.Next() {
		var fan ArenaFan
		if err := rows.Scan(&fan.ID, &fan.Nickname); err != nil {
			return nil, err
		}
		fans = append(fans, fan)
	}
	return fans, rows.Err()
}
