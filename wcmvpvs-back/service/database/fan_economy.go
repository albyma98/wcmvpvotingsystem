package database

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrFanShopFunds = errors.New("insufficient fan coins")
var ErrFanShopItem = errors.New("unknown fan shop item")
var ErrFanShopNotOwned = errors.New("fan shop item not owned")
var ErrFanLoginRequired = errors.New("existing fan phone requires verified login")
var ErrFanRewardDeviceUsed = errors.New("reward on this device belongs to another fan")

type FanWalletMovement struct {
	ID        int    `json:"id"`
	Delta     int    `json:"delta"`
	Balance   int    `json:"balance"`
	Label     string `json:"label"`
	CreatedAt string `json:"created_at"`
}

type FanShopCatalogItem struct {
	Key         string `json:"key"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Cost        int    `json:"cost"`
	Kind        string `json:"kind"`
	Owned       bool   `json:"owned"`
}

type FanCollectionItem struct {
	Key        string `json:"key"`
	Title      string `json:"title"`
	Kind       string `json:"kind"`
	EventID    int    `json:"event_id"`
	AcquiredAt string `json:"acquired_at"`
	ClubName   string `json:"club_name"`
	Opponent   string `json:"opponent"`
	EventDate  string `json:"event_date"`
}

type FanCollectionBadge struct {
	Key       string `json:"key"`
	Title     string `json:"title"`
	EventID   int    `json:"event_id"`
	ClubName  string `json:"club_name"`
	Opponent  string `json:"opponent"`
	EventDate string `json:"event_date"`
}

type FanCollection struct {
	EquippedFrame   string               `json:"equipped_frame"`
	Items           []FanCollectionItem  `json:"items"`
	Badges          []FanCollectionBadge `json:"badges"`
	MatchesFollowed int                  `json:"matches_followed"`
}

type FanWalletState struct {
	Wallet     int                  `json:"wallet"`
	Points     int                  `json:"points"`
	Movements  []FanWalletMovement  `json:"movements"`
	Catalog    []FanShopCatalogItem `json:"catalog"`
	Collection FanCollection        `json:"collection"`
}

func economyOwner(eventID, fanID int, device string) string {
	if fanID > 0 {
		return "fan:" + strconv.Itoa(fanID)
	}
	return "guest:" + strconv.Itoa(eventID) + ":" + strings.TrimSpace(device)
}
func economyHolder(fanID int, device string) string {
	if fanID > 0 {
		return "fan:" + strconv.Itoa(fanID)
	}
	return "guest:" + strings.TrimSpace(device)
}

func ensureRegisteredEconomyAccount(tx *sql.Tx, organizationID, fanID int) error {
	owner := economyOwner(0, fanID, "")
	if _, err := tx.Exec(`INSERT INTO fan_wallet_movements(organization_id,owner_key,fan_id,delta,balance,label,source) SELECT ?,?,fan_id,coins,coins,'Saldo iniziale','opening' FROM fan_wallets WHERE fan_id=? AND coins>0 AND NOT EXISTS(SELECT 1 FROM fan_economy_accounts WHERE organization_id=? AND owner_key=?)`, organizationID, owner, fanID, organizationID, owner); err != nil {
		return err
	}
	_, err := tx.Exec(`INSERT OR IGNORE INTO fan_economy_accounts(organization_id,owner_key,fan_id,points) SELECT ?,?,fan_id,MAX(coins,0) FROM fan_wallets WHERE fan_id=?`, organizationID, owner, fanID)
	return err
}

// Every persistent wallet mutation is observed by these triggers, including
// legacy callers. Opening balances are explicit; historical actions are never
// invented. The points counter only increases on real positive wallet deltas.
func ensureFanEconomyTables(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS fan_economy_accounts(organization_id INTEGER NOT NULL,owner_key TEXT NOT NULL,fan_id INTEGER NOT NULL DEFAULT 0,device_id TEXT NOT NULL DEFAULT '',event_id INTEGER NOT NULL DEFAULT 0,points INTEGER NOT NULL DEFAULT 0,PRIMARY KEY(organization_id,owner_key));`,
		`CREATE TABLE IF NOT EXISTS fan_wallet_movements(id INTEGER PRIMARY KEY AUTOINCREMENT,organization_id INTEGER NOT NULL,owner_key TEXT NOT NULL,event_id INTEGER NOT NULL DEFAULT 0,fan_id INTEGER NOT NULL DEFAULT 0,device_id TEXT NOT NULL DEFAULT '',delta INTEGER NOT NULL,balance INTEGER NOT NULL,label TEXT NOT NULL,source TEXT NOT NULL DEFAULT 'wallet',created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP);`,
		`CREATE INDEX IF NOT EXISTS idx_wallet_movements_owner ON fan_wallet_movements(organization_id,owner_key,id DESC);`,
		`CREATE TABLE IF NOT EXISTS fan_wallet_context(organization_id INTEGER NOT NULL,owner_key TEXT NOT NULL,event_id INTEGER NOT NULL DEFAULT 0,label TEXT NOT NULL DEFAULT '',source TEXT NOT NULL DEFAULT 'wallet',earn_points INTEGER NOT NULL DEFAULT 1,record_movement INTEGER NOT NULL DEFAULT 1,PRIMARY KEY(organization_id,owner_key));`,
		`CREATE TABLE IF NOT EXISTS fan_shop_profiles(organization_id INTEGER NOT NULL,holder_key TEXT NOT NULL,fan_id INTEGER NOT NULL DEFAULT 0,device_id TEXT NOT NULL DEFAULT '',equipped_frame TEXT NOT NULL DEFAULT '',PRIMARY KEY(organization_id,holder_key));`,
		`CREATE TABLE IF NOT EXISTS fan_shop_items(id INTEGER PRIMARY KEY AUTOINCREMENT,organization_id INTEGER NOT NULL,holder_key TEXT NOT NULL,fan_id INTEGER NOT NULL DEFAULT 0,device_id TEXT NOT NULL DEFAULT '',item_key TEXT NOT NULL,scope_event_id INTEGER NOT NULL DEFAULT 0,title TEXT NOT NULL,kind TEXT NOT NULL,cost_coins INTEGER NOT NULL,club_name TEXT NOT NULL,opponent TEXT NOT NULL DEFAULT '',event_date TEXT NOT NULL DEFAULT '',acquired_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,UNIQUE(organization_id,holder_key,item_key,scope_event_id));`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("fan economy schema: %w", err)
		}
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Known historical spending can seed points honestly, while the ledger
	// starts from today's spendable balance and labels it as an opening.
	if _, err := tx.Exec(`INSERT INTO fan_wallet_movements(organization_id,owner_key,fan_id,delta,balance,label,source)
	 SELECT p.organization_id,'fan:'||p.id,p.id,w.coins,w.coins,'Saldo iniziale','opening' FROM fan_profiles p JOIN fan_wallets w ON w.fan_id=p.id
	 WHERE p.organization_id>0 AND w.coins>0 AND NOT EXISTS(SELECT 1 FROM fan_economy_accounts a WHERE a.organization_id=p.organization_id AND a.owner_key='fan:'||p.id)`); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT OR IGNORE INTO fan_economy_accounts(organization_id,owner_key,fan_id,points)
	 SELECT p.organization_id,'fan:'||p.id,p.id,MAX(w.coins,0)
	 +COALESCE((SELECT SUM(r.cost_coins) FROM fan_reward_redemptions r JOIN events e ON e.id=r.event_id WHERE r.fan_id=p.id AND e.organization_id=p.organization_id AND r.cost_coins>0),0)
	 +COALESCE((SELECT SUM(d.coins_donated) FROM fan_support_donations d WHERE d.user_id=p.id AND d.organization_id=p.organization_id AND d.coins_donated>0),0)
	 FROM fan_profiles p JOIN fan_wallets w ON w.fan_id=p.id WHERE p.organization_id>0`); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO fan_wallet_movements(organization_id,owner_key,event_id,device_id,delta,balance,label,source)
	 SELECT w.organization_id,'guest:'||w.event_id||':'||w.device_id,w.event_id,w.device_id,w.coins,w.coins,'Saldo iniziale','opening' FROM guest_wallets w
	 WHERE w.organization_id>0 AND w.coins>0 AND NOT EXISTS(SELECT 1 FROM fan_economy_accounts a WHERE a.organization_id=w.organization_id AND a.owner_key='guest:'||w.event_id||':'||w.device_id)`); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT OR IGNORE INTO fan_economy_accounts(organization_id,owner_key,event_id,device_id,points) SELECT organization_id,'guest:'||event_id||':'||device_id,event_id,device_id,MAX(coins,0) FROM guest_wallets WHERE organization_id>0`); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	for _, spec := range []struct{ table, org, owner, fan, device, event string }{
		{"fan_wallets", "(SELECT organization_id FROM fan_profiles WHERE id=NEW.fan_id)", "('fan:'||NEW.fan_id)", "NEW.fan_id", "''", "0"},
		{"guest_wallets", "NEW.organization_id", "('guest:'||NEW.event_id||':'||NEW.device_id)", "0", "NEW.device_id", "NEW.event_id"},
	} {
		context := fmt.Sprintf(" FROM fan_wallet_context WHERE organization_id=%s AND owner_key=%s", spec.org, spec.owner)
		for _, operation := range []string{"INSERT", "UPDATE"} {
			delta, old, when := "NEW.coins", "0", ""
			if operation == "UPDATE" {
				delta = "(NEW.coins-OLD.coins)"
				old = "MAX(OLD.coins,0)"
				when = " AND NEW.coins<>OLD.coins"
			}
			query := fmt.Sprintf(`CREATE TRIGGER IF NOT EXISTS trg_economy_%s_%s AFTER %s ON %s WHEN %s>0%s BEGIN
			 INSERT INTO fan_wallet_movements(organization_id,owner_key,event_id,fan_id,device_id,delta,balance,label,source)
			 SELECT %s,%s,%s,%s,%s,%s,%s,'Saldo iniziale','opening' WHERE %s>0 AND NOT EXISTS(SELECT 1 FROM fan_economy_accounts WHERE organization_id=%s AND owner_key=%s);
			 INSERT INTO fan_economy_accounts(organization_id,owner_key,event_id,fan_id,device_id,points) VALUES(%s,%s,%s,%s,%s,%s) ON CONFLICT(organization_id,owner_key) DO NOTHING;
			 UPDATE fan_economy_accounts SET points=points+CASE WHEN COALESCE((SELECT earn_points%s),1)=1 THEN MAX(%s,0) ELSE 0 END WHERE organization_id=%s AND owner_key=%s;
			 INSERT INTO fan_wallet_movements(organization_id,owner_key,event_id,fan_id,device_id,delta,balance,label,source)
			 SELECT %s,%s,COALESCE((SELECT event_id%s),%s),%s,%s,%s,NEW.coins,
			 COALESCE(NULLIF((SELECT label%s),''),CASE WHEN %s>0 THEN 'Monete guadagnate' ELSE 'Monete spese' END),COALESCE((SELECT source%s),'wallet')
			 WHERE %s<>0 AND COALESCE((SELECT record_movement%s),1)=1;
			 END;`, spec.table, strings.ToLower(operation), operation, spec.table, spec.org, when,
				spec.org, spec.owner, spec.event, spec.fan, spec.device, old, old, old, spec.org, spec.owner,
				spec.org, spec.owner, spec.event, spec.fan, spec.device, old, context, delta, spec.org, spec.owner,
				spec.org, spec.owner, context, spec.event, spec.fan, spec.device, delta, context, delta, context, delta, context)
			if _, err := db.Exec(query); err != nil {
				return fmt.Errorf("fan economy trigger %s/%s: %w", spec.table, operation, err)
			}
		}
	}
	return nil
}

func economyExec(tx *sql.Tx, organizationID, eventID, fanID int, device, label, source, query string, args ...interface{}) (sql.Result, error) {
	owner := economyOwner(eventID, fanID, device)
	if _, err := tx.Exec(`INSERT INTO fan_wallet_context(organization_id,owner_key,event_id,label,source) VALUES(?,?,?,?,?) ON CONFLICT(organization_id,owner_key) DO UPDATE SET event_id=excluded.event_id,label=excluded.label,source=excluded.source,earn_points=1,record_movement=1`, organizationID, owner, eventID, label, source); err != nil {
		return nil, err
	}
	result, err := tx.Exec(query, args...)
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(`DELETE FROM fan_wallet_context WHERE organization_id=? AND owner_key=?`, organizationID, owner)
	return result, err
}

// Consolidate funds and their existing history, rather than minting another
// reward when an anonymous fan signs in. Deleting guest wallets makes retries
// safe; points include spent guest coins and are transferred only once.
func adoptGuestEconomy(tx *sql.Tx, organizationID, fanID int, device string) error {
	device = strings.TrimSpace(device)
	if organizationID <= 0 || fanID <= 0 || device == "" {
		return nil
	}
	if err := basketballFanScope(tx, organizationID, fanID); err != nil {
		return err
	}
	owner, holder := economyOwner(0, fanID, device), economyHolder(fanID, device)
	guestHolder := economyHolder(0, device)
	var balance, points int
	if err := tx.QueryRow(`SELECT COALESCE(SUM(w.coins),0),COALESCE(SUM(COALESCE(a.points,w.coins)),0) FROM guest_wallets w LEFT JOIN fan_economy_accounts a ON a.organization_id=w.organization_id AND a.owner_key='guest:'||w.event_id||':'||w.device_id WHERE w.organization_id=? AND w.device_id=?`, organizationID, device).Scan(&balance, &points); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT OR IGNORE INTO fan_wallets(fan_id,coins) VALUES(?,0)`, fanID); err != nil {
		return err
	}
	if err := ensureRegisteredEconomyAccount(tx, organizationID, fanID); err != nil {
		return err
	}
	if balance > 0 {
		if _, err := tx.Exec(`INSERT INTO fan_wallet_context(organization_id,owner_key,earn_points,record_movement) VALUES(?,?,0,0) ON CONFLICT(organization_id,owner_key) DO UPDATE SET earn_points=0,record_movement=0`, organizationID, owner); err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE fan_wallets SET coins=coins+?,updated_at=CURRENT_TIMESTAMP WHERE fan_id=?`, balance, fanID); err != nil {
			return err
		}
		if _, err := tx.Exec(`DELETE FROM fan_wallet_context WHERE organization_id=? AND owner_key=?`, organizationID, owner); err != nil {
			return err
		}
	}
	if points > 0 {
		if _, err := tx.Exec(`UPDATE fan_economy_accounts SET points=points+? WHERE organization_id=? AND owner_key=?`, points, organizationID, owner); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`UPDATE fan_wallet_movements SET owner_key=?,fan_id=? WHERE organization_id=? AND fan_id=0 AND device_id=?`, owner, fanID, organizationID, device); err != nil {
		return err
	}
	if _, err := tx.Exec(`WITH balances AS (SELECT id,SUM(delta) OVER(ORDER BY id) AS balance FROM fan_wallet_movements WHERE organization_id=? AND owner_key=?) UPDATE fan_wallet_movements SET balance=(SELECT balance FROM balances WHERE balances.id=fan_wallet_movements.id) WHERE organization_id=? AND owner_key=?`, organizationID, owner, organizationID, owner); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT OR IGNORE INTO fan_shop_items(organization_id,holder_key,fan_id,device_id,item_key,scope_event_id,title,kind,cost_coins,club_name,opponent,event_date,acquired_at) SELECT organization_id,?,?,device_id,item_key,scope_event_id,title,kind,cost_coins,club_name,opponent,event_date,acquired_at FROM fan_shop_items WHERE organization_id=? AND holder_key=?`, holder, fanID, organizationID, guestHolder); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT OR IGNORE INTO fan_shop_profiles(organization_id,holder_key,fan_id,device_id) VALUES(?,?,?,?)`, organizationID, holder, fanID, device); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE fan_shop_profiles SET equipped_frame=COALESCE((SELECT equipped_frame FROM fan_shop_profiles WHERE organization_id=? AND holder_key=?),'') WHERE organization_id=? AND holder_key=? AND equipped_frame=''`, organizationID, guestHolder, organizationID, holder); err != nil {
		return err
	}
	for _, table := range []string{"fan_shop_items", "fan_shop_profiles"} {
		if _, err := tx.Exec(`DELETE FROM `+table+` WHERE organization_id=? AND holder_key=?`, organizationID, guestHolder); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`DELETE FROM fan_economy_accounts WHERE organization_id=? AND fan_id=0 AND device_id=?`, organizationID, device); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM guest_wallets WHERE organization_id=? AND device_id=?`, organizationID, device); err != nil {
		return err
	}
	for _, table := range []string{"fan_journey_rewards", "basketball_shots", "playmaker_sessions", "playmaker_passes", "crossover_sessions", "crossover_duels"} {
		if _, err := tx.Exec(`UPDATE OR IGNORE `+table+` SET fan_id=? WHERE organization_id=? AND device_id=? AND fan_id=0`, fanID, organizationID, device); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`UPDATE votes SET user_id=? WHERE device_id=? AND (user_id IS NULL OR user_id=0) AND EXISTS(SELECT 1 FROM events e WHERE e.id=votes.event_id AND e.organization_id=?)`, fanID, device, organizationID); err != nil {
		return err
	}
	var brandedExists int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='branded_game_participations'`).Scan(&brandedExists); err != nil {
		return err
	}
	if brandedExists > 0 {
		if _, err := tx.Exec(`UPDATE branded_game_participations SET user_id=? WHERE device_id=? AND (user_id IS NULL OR user_id=0) AND EXISTS(SELECT 1 FROM events e WHERE e.id=branded_game_participations.event_id AND e.organization_id=?)`, fanID, device, organizationID); err != nil {
			return err
		}
	}
	return nil
}
