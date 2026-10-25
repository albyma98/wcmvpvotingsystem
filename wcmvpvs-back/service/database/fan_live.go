package database

import "database/sql"

func (db *appdbimpl) GetFanLiveEvent(eventID, organizationID int) (Event, error) {
	var event Event
	var surveyJSON sql.NullString
	err := db.c.QueryRow(`SELECT e.id, e.organization_id, e.is_active, e.votes_closed, e.is_concluded,
		IFNULL(t2.name, ''), e.show_feedback_survey, e.feedback_survey_config, COALESCE(vs.show_lottery_ticket,0)
		FROM events e LEFT JOIN teams t2 ON t2.id=e.team2_id
		LEFT JOIN organization_voting_settings vs ON vs.organization_id=e.organization_id
		WHERE e.id=? AND e.organization_id=?`, eventID, organizationID).Scan(
		&event.ID, &event.OrganizationID, &event.IsActive, &event.VotesClosed, &event.IsConcluded, &event.Team2Name, &event.ShowFeedbackSurvey, &surveyJSON, &event.ShowLotteryTicket)
	if err == nil {
		config := decodeEventFeedbackSurveyConfig(surveyJSON)
		event.FeedbackSurvey = &config
	}
	return event, err
}

// Use the same deterministic ordering as the existing club wallet leaderboard.
func (db *appdbimpl) GetFanNearbyRanks(organizationID, fanID int) ([]FanLeaderboardEntry, error) {
	rows, err := db.c.Query(`WITH ranked AS (
		SELECT p.id, p.nickname, COALESCE(a.points,w.coins) AS coins, COALESCE(s.equipped_frame,'') AS equipped_frame, ROW_NUMBER() OVER (ORDER BY COALESCE(a.points,w.coins) DESC, p.id ASC) AS position
		FROM fan_profiles p JOIN fan_wallets w ON w.fan_id=p.id
		LEFT JOIN fan_economy_accounts a ON a.organization_id=p.organization_id AND a.owner_key='fan:'||p.id
		LEFT JOIN fan_shop_profiles s ON s.organization_id=p.organization_id AND s.holder_key='fan:'||p.id WHERE p.organization_id=?
	) SELECT id, nickname, coins, position, equipped_frame FROM ranked
	WHERE position BETWEEN (SELECT position-1 FROM ranked WHERE id=?) AND (SELECT position+1 FROM ranked WHERE id=?)
	ORDER BY position`, organizationID, fanID, fanID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []FanLeaderboardEntry{}
	for rows.Next() {
		var entry FanLeaderboardEntry
		if err := rows.Scan(&entry.FanID, &entry.Nickname, &entry.Coins, &entry.Rank, &entry.EquippedFrame); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
		entries[len(entries)-1].Points = entry.Coins
	}
	return entries, rows.Err()
}
