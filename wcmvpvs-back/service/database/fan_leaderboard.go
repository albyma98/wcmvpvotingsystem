package database

// Event points come from earned movements, including guest earnings adopted by
// a registered profile. Spending and unattributed opening balances never count.
func (db *appdbimpl) GetFanEventLeaderboard(eventID, organizationID, limit int) ([]FanLeaderboardEntry, error) {
	if limit == 0 {
		limit = 3
	}
	// Tap live payouts predate event-aware wallet movements, so their finalized
	// receipts supply the event attribution instead of their unscoped movements.
	rows, err := db.c.Query(`WITH earned AS (
		SELECT fan_id,event_id,organization_id,delta FROM fan_wallet_movements
		WHERE event_id=? AND organization_id=? AND delta>0 AND source<>'opening'
		UNION ALL
		SELECT fan1_id,event_id,organization_id,fan1_coins FROM tap_live_matches
		WHERE event_id=? AND organization_id=? AND status='finished' AND fan1_coins>0
		UNION ALL
		SELECT fan2_id,event_id,organization_id,fan2_coins FROM tap_live_matches
		WHERE event_id=? AND organization_id=? AND status='finished' AND fan2_coins>0
		)
		SELECT p.id,p.nickname,SUM(m.delta),p.created_at,COALESCE(s.equipped_frame,'')
		FROM earned m
		JOIN events e ON e.id=m.event_id AND e.organization_id=m.organization_id
		JOIN fan_profiles p ON p.id=m.fan_id AND p.organization_id=m.organization_id
		LEFT JOIN fan_shop_profiles s ON s.organization_id=p.organization_id AND s.holder_key='fan:'||p.id
		GROUP BY p.id,p.nickname,p.created_at,s.equipped_frame
		ORDER BY SUM(m.delta) DESC,p.id ASC LIMIT ?`, eventID, organizationID, eventID, organizationID, eventID, organizationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []FanLeaderboardEntry{}
	for rows.Next() {
		var entry FanLeaderboardEntry
		if err := rows.Scan(&entry.FanID, &entry.Nickname, &entry.Coins, &entry.CreatedAt, &entry.EquippedFrame); err != nil {
			return nil, err
		}
		entry.Points = entry.Coins
		entry.Rank = len(entries) + 1
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}
