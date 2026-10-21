package database

import (
	"database/sql"
	"sort"
	"strings"
)

func fanShopCatalog() []FanShopCatalogItem {
	return []FanShopCatalogItem{
		{Key: "club_frame", Title: "Cornice della società", Description: "Personalizza il tuo profilo con i colori della società.", Cost: 20, Kind: "frame"},
		{Key: "match_card", Title: "Card ricordo della partita", Description: "Conserva nella collezione il ricordo di questa partita.", Cost: 15, Kind: "match_card"},
	}
}

func fanShopEvent(tx *sql.Tx, eventID, organizationID int) (string, string, string, error) {
	var club, opponent, date string
	err := tx.QueryRow(`SELECT o.name,COALESCE(t.name,''),e.start_datetime FROM events e JOIN organizations o ON o.id=e.organization_id LEFT JOIN teams t ON t.id=e.team2_id WHERE e.id=? AND e.organization_id=?`, eventID, organizationID).Scan(&club, &opponent, &date)
	return club, opponent, date, err
}

func ensureEconomyWallet(tx *sql.Tx, eventID, organizationID, fanID int, device string) error {
	if eventID <= 0 || organizationID <= 0 || fanID < 0 || strings.TrimSpace(device) == "" {
		return sql.ErrNoRows
	}
	if fanID > 0 {
		if err := basketballFanScope(tx, organizationID, fanID); err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT OR IGNORE INTO fan_wallets(fan_id,coins) VALUES(?,0)`, fanID); err != nil {
			return err
		}
		return ensureRegisteredEconomyAccount(tx, organizationID, fanID)
	}
	_, err := tx.Exec(`INSERT OR IGNORE INTO guest_wallets(event_id,organization_id,device_id,coins) VALUES(?,?,?,0)`, eventID, organizationID, device)
	return err
}

func (db *appdbimpl) PurchaseFanShopItem(eventID, organizationID, fanID int, device, itemKey string) error {
	device = strings.TrimSpace(device)
	var item FanShopCatalogItem
	for _, candidate := range fanShopCatalog() {
		if candidate.Key == itemKey {
			item = candidate
		}
	}
	if item.Key == "" {
		return ErrFanShopItem
	}
	tx, err := db.c.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	club, opponent, date, err := fanShopEvent(tx, eventID, organizationID)
	if err != nil {
		return err
	}
	if err := ensureEconomyWallet(tx, eventID, organizationID, fanID, device); err != nil {
		return err
	}
	holder := economyHolder(fanID, device)
	scope := eventID
	if item.Kind == "frame" {
		scope = 0
		opponent = ""
		date = ""
	}
	var existing int
	err = tx.QueryRow(`SELECT id FROM fan_shop_items WHERE organization_id=? AND holder_key=? AND item_key=? AND scope_event_id=?`, organizationID, holder, item.Key, scope).Scan(&existing)
	if err == nil {
		return tx.Commit()
	}
	if err != sql.ErrNoRows {
		return err
	}
	var changed sql.Result
	if fanID > 0 {
		changed, err = economyExec(tx, organizationID, eventID, fanID, device, "Acquisto: "+item.Title, "shop:"+item.Key, `UPDATE fan_wallets SET coins=coins-?,updated_at=CURRENT_TIMESTAMP WHERE fan_id=? AND coins>=?`, item.Cost, fanID, item.Cost)
	} else {
		changed, err = economyExec(tx, organizationID, eventID, 0, device, "Acquisto: "+item.Title, "shop:"+item.Key, `UPDATE guest_wallets SET coins=coins-?,updated_at=CURRENT_TIMESTAMP WHERE event_id=? AND organization_id=? AND device_id=? AND coins>=?`, item.Cost, eventID, organizationID, device, item.Cost)
	}
	if err != nil {
		return err
	}
	if count, err := changed.RowsAffected(); err != nil {
		return err
	} else if count != 1 {
		return ErrFanShopFunds
	}
	if _, err := tx.Exec(`INSERT INTO fan_shop_items(organization_id,holder_key,fan_id,device_id,item_key,scope_event_id,title,kind,cost_coins,club_name,opponent,event_date) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, organizationID, holder, fanID, device, item.Key, scope, item.Title, item.Kind, item.Cost, club, opponent, date); err != nil {
		return err
	}
	if item.Kind == "frame" {
		if _, err := tx.Exec(`INSERT INTO fan_shop_profiles(organization_id,holder_key,fan_id,device_id,equipped_frame) VALUES(?,?,?,?,'club_frame') ON CONFLICT(organization_id,holder_key) DO UPDATE SET equipped_frame='club_frame'`, organizationID, holder, fanID, device); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (db *appdbimpl) EquipFanShopItem(eventID, organizationID, fanID int, device, itemKey string) error {
	device = strings.TrimSpace(device)
	if itemKey != "" && itemKey != "club_frame" {
		return ErrFanShopItem
	}
	tx, err := db.c.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, _, _, err := fanShopEvent(tx, eventID, organizationID); err != nil {
		return err
	}
	if err := ensureEconomyWallet(tx, eventID, organizationID, fanID, device); err != nil {
		return err
	}
	holder := economyHolder(fanID, device)
	if itemKey != "" {
		var count int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM fan_shop_items WHERE organization_id=? AND holder_key=? AND item_key='club_frame' AND scope_event_id=0`, organizationID, holder).Scan(&count); err != nil {
			return err
		}
		if count == 0 {
			return ErrFanShopNotOwned
		}
	}
	if _, err := tx.Exec(`INSERT INTO fan_shop_profiles(organization_id,holder_key,fan_id,device_id,equipped_frame) VALUES(?,?,?,?,?) ON CONFLICT(organization_id,holder_key) DO UPDATE SET equipped_frame=excluded.equipped_frame`, organizationID, holder, fanID, device, itemKey); err != nil {
		return err
	}
	return tx.Commit()
}

func collectionParticipation(tx *sql.Tx, organizationID, fanID int, device string) (map[int]int, map[int]bool, error) {
	args := []interface{}{}
	parts := []string{}
	for _, spec := range []struct{ table, amount, condition, group string }{
		{"fan_journey_rewards", "MAX(coins)", "1=1", "event_id,reward_key"},
		{"fan_minigame_rewards", "SUM(coins)", "1=1", "event_id,reward_key"},
		{"basketball_shots", "COUNT(*)", "status='made'", "event_id"},
		{"playmaker_passes", "COUNT(*)", "assisted=1", "event_id"},
		{"crossover_duels", "COUNT(*)", "won=1", "event_id"},
	} {
		parts = append(parts, `SELECT event_id,`+spec.amount+` AS earned FROM `+spec.table+` WHERE organization_id=? AND `+spec.condition+` AND ((?>0 AND fan_id=?) OR (fan_id=0 AND device_id=?)) GROUP BY `+spec.group)
		args = append(args, organizationID, fanID, fanID, device)
	}
	if fanID > 0 {
		parts = append(parts, `SELECT event_id,fan1_coins AS earned FROM tap_live_matches WHERE organization_id=? AND fan1_id=? AND status='finished'`, `SELECT event_id,fan2_coins AS earned FROM tap_live_matches WHERE organization_id=? AND fan2_id=? AND status='finished'`)
		args = append(args, organizationID, fanID, organizationID, fanID)
	}
	var hasBranded int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='branded_game_participations'`).Scan(&hasBranded); err != nil {
		return nil, nil, err
	}
	if hasBranded > 0 {
		parts = append(parts, `SELECT b.event_id,b.rewarded_coins AS earned FROM branded_game_participations b JOIN events e ON e.id=b.event_id WHERE e.organization_id=? AND b.completed=1 AND ((?>0 AND b.user_id=?) OR ((b.user_id IS NULL OR b.user_id=0) AND b.device_id=?))`)
		args = append(args, organizationID, fanID, fanID, device)
	}
	rows, err := tx.Query(`SELECT event_id,SUM(earned) FROM (`+strings.Join(parts, " UNION ALL ")+`) GROUP BY event_id`, args...)
	if err != nil {
		return nil, nil, err
	}
	earned, followed := map[int]int{}, map[int]bool{}
	for rows.Next() {
		var event, coins int
		if err := rows.Scan(&event, &coins); err != nil {
			rows.Close()
			return nil, nil, err
		}
		earned[event] = coins
		if coins > 0 {
			followed[event] = true
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, nil, err
	}
	rows.Close()
	rows, err = tx.Query(`SELECT DISTINCT v.event_id FROM votes v JOIN events e ON e.id=v.event_id WHERE e.organization_id=? AND ((?>0 AND v.user_id=?) OR ((v.user_id IS NULL OR v.user_id=0) AND v.device_id=?))`, organizationID, fanID, fanID, device)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var event int
		if err := rows.Scan(&event); err != nil {
			return nil, nil, err
		}
		followed[event] = true
	}
	return earned, followed, rows.Err()
}

func (db *appdbimpl) GetFanWalletState(eventID, organizationID, fanID int, device string) (FanWalletState, error) {
	device = strings.TrimSpace(device)
	state := FanWalletState{Movements: []FanWalletMovement{}, Catalog: fanShopCatalog(), Collection: FanCollection{Items: []FanCollectionItem{}, Badges: []FanCollectionBadge{}}}
	tx, err := db.c.Begin()
	if err != nil {
		return state, err
	}
	defer tx.Rollback()
	if _, _, _, err := fanShopEvent(tx, eventID, organizationID); err != nil {
		return state, err
	}
	if err := ensureEconomyWallet(tx, eventID, organizationID, fanID, device); err != nil {
		return state, err
	}
	owner, holder := economyOwner(eventID, fanID, device), economyHolder(fanID, device)
	if fanID > 0 {
		err = tx.QueryRow(`SELECT coins FROM fan_wallets WHERE fan_id=?`, fanID).Scan(&state.Wallet)
	} else {
		err = tx.QueryRow(`SELECT coins FROM guest_wallets WHERE event_id=? AND organization_id=? AND device_id=?`, eventID, organizationID, device).Scan(&state.Wallet)
	}
	if err != nil {
		return state, err
	}
	if err := tx.QueryRow(`SELECT points FROM fan_economy_accounts WHERE organization_id=? AND owner_key=?`, organizationID, owner).Scan(&state.Points); err != nil {
		return state, err
	}
	rows, err := tx.Query(`SELECT id,delta,balance,label,created_at FROM fan_wallet_movements WHERE organization_id=? AND owner_key=? ORDER BY id DESC LIMIT 30`, organizationID, owner)
	if err != nil {
		return state, err
	}
	for rows.Next() {
		var movement FanWalletMovement
		if err := rows.Scan(&movement.ID, &movement.Delta, &movement.Balance, &movement.Label, &movement.CreatedAt); err != nil {
			rows.Close()
			return state, err
		}
		state.Movements = append(state.Movements, movement)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return state, err
	}
	rows.Close()
	rows, err = tx.Query(`SELECT item_key,title,kind,scope_event_id,acquired_at,club_name,opponent,event_date FROM fan_shop_items WHERE organization_id=? AND holder_key=? ORDER BY id DESC`, organizationID, holder)
	if err != nil {
		return state, err
	}
	for rows.Next() {
		var item FanCollectionItem
		if err := rows.Scan(&item.Key, &item.Title, &item.Kind, &item.EventID, &item.AcquiredAt, &item.ClubName, &item.Opponent, &item.EventDate); err != nil {
			rows.Close()
			return state, err
		}
		state.Collection.Items = append(state.Collection.Items, item)
		for index := range state.Catalog {
			if state.Catalog[index].Key == item.Key && (item.Kind == "frame" || item.EventID == eventID) {
				state.Catalog[index].Owned = true
			}
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return state, err
	}
	rows.Close()
	if err := tx.QueryRow(`SELECT equipped_frame FROM fan_shop_profiles WHERE organization_id=? AND holder_key=?`, organizationID, holder).Scan(&state.Collection.EquippedFrame); err != nil && err != sql.ErrNoRows {
		return state, err
	}
	earned, followed, err := collectionParticipation(tx, organizationID, fanID, device)
	if err != nil {
		return state, err
	}
	state.Collection.MatchesFollowed = len(followed)
	for event, coins := range earned {
		if coins < 25 {
			continue
		}
		club, opponent, date, err := fanShopEvent(tx, event, organizationID)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return state, err
		}
		state.Collection.Badges = append(state.Collection.Badges, FanCollectionBadge{Key: "tifoso_attivo", Title: "Tifoso Attivo", EventID: event, ClubName: club, Opponent: opponent, EventDate: date})
	}
	sort.Slice(state.Collection.Badges, func(i, j int) bool { return state.Collection.Badges[i].EventID > state.Collection.Badges[j].EventID })
	if err := tx.Commit(); err != nil {
		return state, err
	}
	return state, nil
}
