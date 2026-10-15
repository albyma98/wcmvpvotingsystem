package database

import "database/sql"

func (db *appdbimpl) GetFanLiveEvent(eventID, organizationID int) (Event, error) {
	var event Event
	var surveyJSON sql.NullString
	err := db.c.QueryRow(`SELECT e.id, e.organization_id, e.is_active, e.votes_closed, e.is_concluded,
		IFNULL(t2.name, ''), e.show_feedback_survey, e.feedback_survey_config FROM events e LEFT JOIN teams t2 ON t2.id=e.team2_id
		WHERE e.id=? AND e.organization_id=?`, eventID, organizationID).Scan(
		&event.ID, &event.OrganizationID, &event.IsActive, &event.VotesClosed, &event.IsConcluded, &event.Team2Name, &event.ShowFeedbackSurvey, &surveyJSON)
	if err == nil {
		config := decodeEventFeedbackSurveyConfig(surveyJSON)
		event.FeedbackSurvey = &config
	}
	return event, err
}

// Use the same deterministic ordering as the existing club wallet leaderboard.
func (db *appdbimpl) GetFanNearbyRanks(organizationID, fanID int) ([]FanLeaderboardEntry, error) {
	rows, err := db.c.Query(`WITH ranked AS (
		SELECT p.id, p.nickname, w.coins, ROW_NUMBER() OVER (ORDER BY w.coins DESC, p.id ASC) AS position
		FROM fan_profiles p JOIN fan_wallets w ON w.fan_id=p.id WHERE p.organization_id=?
	) SELECT id, nickname, coins, position FROM ranked
	WHERE position BETWEEN (SELECT position-1 FROM ranked WHERE id=?) AND (SELECT position+1 FROM ranked WHERE id=?)
	ORDER BY position`, organizationID, fanID, fanID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []FanLeaderboardEntry{}
	for rows.Next() {
		var entry FanLeaderboardEntry
		if err := rows.Scan(&entry.FanID, &entry.Nickname, &entry.Coins, &entry.Rank); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}
