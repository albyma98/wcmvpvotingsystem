package database

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	_ "time/tzdata"
)

var ErrArena = errors.New("attività non disponibile o richiesta non valida")
var rome, _ = time.LoadLocation("Europe/Rome")

type ArenaItem struct {
	Key          string `json:"key"`
	Title        string `json:"title"`
	Slot         string `json:"slot"`
	Cost         int    `json:"cost"`
	Level        int    `json:"level"`
	Availability string `json:"availability"`
}

func ArenaCatalog() []ArenaItem {
	return []ArenaItem{
		{"club-shirt", "Maglia Joy Volley", "shirt", 20, 1, "always"}, {"away-shirt", "Maglia alternativa", "shirt", 25, 1, "event"}, {"hoodie", "Felpa", "shirt", 35, 1, "weekly"}, {"sport-pants", "Pantaloni sportivi", "pants", 15, 1, "always"},
		{"scarf", "Sciarpa", "scarf", 20, 1, "always"}, {"cap", "Cappellino", "hat", 20, 1, "always"}, {"glasses", "Occhiali", "glasses", 30, 1, "always"}, {"flag", "Bandierina", "flag", 35, 1, "weekly"},
		{"shoes", "Scarpe speciali", "shoes", 25, 1, "always"}, {"hair", "Acconciatura alternativa", "hair", 20, 1, "always"}, {"shirt-color", "Maglia verde lime", "shirt", 30, 2, "level"}, {"celebration", "Esultanza speciale", "celebration", 40, 2, "level"},
	}
}

type ArenaConfig struct {
	Treasure      int `json:"treasure"`
	Exact         int `json:"exact"`
	Winner        int `json:"winner"`
	Growth        int `json:"growth"`
	GrowthBig     int `json:"growth_big"`
	Challenge     int `json:"challenge"`
	Collective    int `json:"collective"`
	Record        int `json:"record"`
	CutoffMinutes int `json:"cutoff_minutes"`
}

func defaultArenaConfig() ArenaConfig { return ArenaConfig{20, 50, 15, 100, 500, 100, 3, 10, 0} }

type ArenaCommand struct {
	Action   string       `json:"action"`
	Key      string       `json:"key"`
	EventID  int          `json:"event_id"`
	Score    string       `json:"score"`
	Day      string       `json:"day"`
	Location int          `json:"location"`
	Code     string       `json:"code"`
	Public   bool         `json:"public"`
	Confirm  bool         `json:"confirm"`
	Config   *ArenaConfig `json:"config,omitempty"`
}
type ArenaTreasure struct {
	Day      string `json:"day"`
	Kind     int    `json:"kind"`
	Location int    `json:"location"`
	Reward   int    `json:"reward"`
	Claimed  bool   `json:"claimed"`
}
type ArenaMember struct {
	FanID     int               `json:"fan_id"`
	Slot      int               `json:"slot"`
	Public    bool              `json:"public"`
	Equipment map[string]string `json:"equipment"`
}
type ArenaMatch struct {
	ID      int    `json:"id"`
	Home    string `json:"home"`
	Away    string `json:"away"`
	Starts  string `json:"starts"`
	Cutoff  string `json:"cutoff"`
	Score   string `json:"score"`
	Choice  string `json:"choice"`
	Reward  int    `json:"reward"`
	Settled bool   `json:"settled"`
	Open    bool   `json:"open"`
}
type ArenaTrophy struct {
	Key         string `json:"key"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Unlocked    string `json:"unlocked"`
	Value       int    `json:"value"`
	Target      int    `json:"target"`
}
type ArenaActivityState struct {
	Now        string            `json:"now"`
	Treasure   ArenaTreasure     `json:"treasure"`
	Catalog    []ArenaItem       `json:"catalog"`
	Featured   []string          `json:"featured"`
	Owned      []string          `json:"owned"`
	Equipment  map[string]string `json:"equipment"`
	Wallet     int               `json:"wallet"`
	Points     int               `json:"points"`
	Level      int               `json:"level"`
	Member     *ArenaMember      `json:"member"`
	Members    []ArenaMember     `json:"members"`
	Matches    []ArenaMatch      `json:"matches"`
	Trophies   []ArenaTrophy     `json:"trophies"`
	Invitation string            `json:"invitation"`
	Config     ArenaConfig       `json:"config"`
}

func ensureArenaTables(db *sql.DB) error {
	for _, q := range []string{
		`CREATE TABLE IF NOT EXISTS arena_settings(org INTEGER PRIMARY KEY,config TEXT NOT NULL DEFAULT '{}')`,
		`CREATE TABLE IF NOT EXISTS arena_members(org INTEGER NOT NULL,fan INTEGER NOT NULL,slot INTEGER NOT NULL CHECK(slot>=0 AND slot<1330),public INTEGER NOT NULL DEFAULT 0,equipment TEXT NOT NULL DEFAULT '{}',PRIMARY KEY(org,fan),UNIQUE(org,slot))`,
		`CREATE TABLE IF NOT EXISTS arena_claims(org INTEGER NOT NULL,fan INTEGER NOT NULL,day TEXT NOT NULL,coins INTEGER NOT NULL,PRIMARY KEY(org,fan,day))`,
		`CREATE TABLE IF NOT EXISTS arena_predictions(org INTEGER NOT NULL,fan INTEGER NOT NULL,event INTEGER NOT NULL,choice TEXT NOT NULL,reward INTEGER,PRIMARY KEY(org,fan,event))`,
		`CREATE TABLE IF NOT EXISTS arena_results(org INTEGER NOT NULL,event INTEGER NOT NULL,score TEXT NOT NULL,confirmed_at TEXT NOT NULL,PRIMARY KEY(org,event))`,
		`CREATE TABLE IF NOT EXISTS arena_invites(code TEXT PRIMARY KEY,org INTEGER NOT NULL,inviter INTEGER NOT NULL,created_at TEXT NOT NULL,accepted_by INTEGER,accepted_at TEXT)`,
		`CREATE TABLE IF NOT EXISTS arena_actions(org INTEGER NOT NULL,fan INTEGER NOT NULL,kind TEXT NOT NULL,day TEXT NOT NULL,at INTEGER NOT NULL,completed INTEGER NOT NULL DEFAULT 0,PRIMARY KEY(org,fan,kind,day))`,
		`CREATE TABLE IF NOT EXISTS arena_trophies(org INTEGER NOT NULL,key TEXT NOT NULL,title TEXT NOT NULL,description TEXT NOT NULL,unlocked TEXT NOT NULL,value INTEGER NOT NULL,PRIMARY KEY(org,key))`,
	} {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	if _, err := db.Exec(`ALTER TABLE arena_actions ADD COLUMN completed INTEGER NOT NULL DEFAULT 0`); err != nil && !strings.Contains(err.Error(), "duplicate column") {
		return err
	}
	return nil
}
func arenaDay(now time.Time) string { return now.In(rome).Format("2006-01-02") }
func arenaTreasure(org int, now time.Time, reward int) ArenaTreasure {
	day := arenaDay(now)
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d:%s", org, day)))
	return ArenaTreasure{Day: day, Kind: int(sum[0]) % 4, Location: int(sum[1]) % 10, Reward: reward}
}
func arenaFeatured(now time.Time) []string {
	year, week := now.In(rome).ISOWeek()
	if (year+week)%2 == 0 {
		return []string{"hoodie", "scarf", "shoes"}
	}
	return []string{"flag", "cap", "away-shirt"}
}
func arenaScore(score string) bool {
	for _, s := range []string{"3-0", "3-1", "3-2", "2-3", "1-3", "0-3"} {
		if s == score {
			return true
		}
	}
	return false
}
func arenaDate(raw string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05"} {
		if t, e := time.ParseInLocation(layout, raw, rome); e == nil {
			return t, nil
		}
	}
	return time.Time{}, ErrArena
}
func arenaConfig(tx *sql.Tx, org int) (ArenaConfig, error) {
	c := defaultArenaConfig()
	var raw string
	err := tx.QueryRow(`SELECT config FROM arena_settings WHERE org=?`, org).Scan(&raw)
	if err != nil {
		return c, err
	}
	err = json.Unmarshal([]byte(raw), &c)
	return c, err
}
func arenaMember(tx *sql.Tx, org, fan int) (ArenaMember, error) {
	m := ArenaMember{FanID: fan, Equipment: map[string]string{}}
	var raw string
	err := tx.QueryRow(`SELECT slot,public,equipment FROM arena_members WHERE org=? AND fan=?`, org, fan).Scan(&m.Slot, &m.Public, &raw)
	if err == nil {
		err = json.Unmarshal([]byte(raw), &m.Equipment)
	}
	return m, err
}
func arenaAssign(tx *sql.Tx, org, fan int) error {
	var count int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM arena_members WHERE org=? AND fan=?`, org, fan).Scan(&count); err != nil || count > 0 {
		return err
	}
	_, err := tx.Exec(`WITH RECURSIVE slots(n) AS (SELECT 0 UNION ALL SELECT n+1 FROM slots WHERE n<1329) INSERT INTO arena_members(org,fan,slot) SELECT ?,?,n FROM slots WHERE NOT EXISTS(SELECT 1 FROM arena_members WHERE org=? AND slot=n) ORDER BY n LIMIT 1`, org, fan, org)
	return err
}

// Every transaction takes the club write lock before reading mutable state.
// Rewards and purchases commit with the existing wallet and its audit triggers.
func (db *appdbimpl) ArenaActivities(org, fan int, cmd ArenaCommand, now time.Time) (ArenaActivityState, error) {
	state := ArenaActivityState{Now: now.Format(time.RFC3339), Catalog: ArenaCatalog(), Featured: arenaFeatured(now), Owned: []string{}, Equipment: map[string]string{}, Members: []ArenaMember{}, Matches: []ArenaMatch{}, Trophies: []ArenaTrophy{}}
	if org <= 0 {
		return state, ErrArena
	}
	tx, err := db.c.Begin()
	if err != nil {
		return state, err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`INSERT OR IGNORE INTO arena_settings(org) VALUES(?)`, org); err != nil {
		return state, err
	}
	if _, err = tx.Exec(`UPDATE arena_settings SET config=config WHERE org=?`, org); err != nil {
		return state, err
	}
	config, err := arenaConfig(tx, org)
	if err != nil {
		return state, err
	}
	state.Config = config
	state.Treasure = arenaTreasure(org, now, config.Treasure)
	// Persist assignments once; deleted profiles never compact another fan's seat.
	rows, err := tx.Query(`SELECT id FROM fan_profiles p WHERE organization_id=? AND registration_pending=0 AND accepted_terms=1 AND NOT EXISTS(SELECT 1 FROM arena_members m WHERE m.org=p.organization_id AND m.fan=p.id) ORDER BY id`, org)
	if err != nil {
		return state, err
	}
	ids := []int{}
	for rows.Next() {
		var id int
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return state, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return state, err
	}
	for _, id := range ids {
		if err = arenaAssign(tx, org, id); err != nil {
			return state, err
		}
	}
	if fan > 0 {
		var valid int
		if err = tx.QueryRow(`SELECT id FROM fan_profiles WHERE id=? AND organization_id=? AND registration_pending=0 AND accepted_terms=1`, fan, org).Scan(&valid); err != nil {
			return state, err
		}
		if _, err = tx.Exec(`INSERT OR IGNORE INTO fan_wallets(fan_id,coins) VALUES(?,0)`, fan); err != nil {
			return state, err
		}
		if err = ensureRegisteredEconomyAccount(tx, org, fan); err != nil {
			return state, err
		}
	}
	if cmd.Action != "" && cmd.Action != "state" && cmd.Action != "config" && cmd.Action != "result" && fan <= 0 {
		return state, ErrArena
	}
	switch cmd.Action {
	case "", "state":
	case "claim":
		if cmd.Day != state.Treasure.Day || cmd.Location != state.Treasure.Location {
			return state, ErrArena
		}
		var changed sql.Result
		changed, err = tx.Exec(`INSERT OR IGNORE INTO arena_claims(org,fan,day,coins) VALUES(?,?,?,?)`, org, fan, state.Treasure.Day, config.Treasure)
		if err == nil {
			var n int64
			n, err = changed.RowsAffected()
			if err == nil && n > 0 {
				_, err = economyExec(tx, org, 0, fan, "arena", "Tesoro giornaliero", "arena:treasure", `UPDATE fan_wallets SET coins=coins+?,updated_at=CURRENT_TIMESTAMP WHERE fan_id=?`, config.Treasure, fan)
			}
		}
	case "buy", "equip", "remove":
		var item ArenaItem
		for _, i := range ArenaCatalog() {
			if i.Key == cmd.Key {
				item = i
			}
		}
		if item.Key == "" {
			return state, ErrArena
		}
		var owned int
		err = tx.QueryRow(`SELECT COUNT(*) FROM fan_shop_items WHERE organization_id=? AND fan_id=? AND item_key=? AND scope_event_id=0`, org, fan, "arena:"+item.Key).Scan(&owned)
		if err != nil {
			return state, err
		}
		if cmd.Action == "buy" && owned == 0 {
			var points int
			err = tx.QueryRow(`SELECT points FROM fan_economy_accounts WHERE organization_id=? AND owner_key=?`, org, economyOwner(0, fan, "")).Scan(&points)
			if err != nil {
				return state, err
			}
			available := item.Availability != "weekly"
			if item.Availability == "event" {
				var count int
				if err = tx.QueryRow(`SELECT COUNT(*) FROM events WHERE organization_id=? AND is_concluded=0 AND (is_active=1 OR datetime(start_datetime) BETWEEN datetime(?) AND datetime(?))`, org, now.Format(time.RFC3339), now.AddDate(0, 0, 7).Format(time.RFC3339)).Scan(&count); err != nil {
					return state, err
				}
				available = count > 0
			}
			for _, key := range state.Featured {
				if key == item.Key {
					available = true
				}
			}
			if !available || 1+points/100 < item.Level {
				return state, ErrArena
			}
			var changed sql.Result
			changed, err = economyExec(tx, org, 0, fan, "arena", "Guardaroba: "+item.Title, "arena:wardrobe", `UPDATE fan_wallets SET coins=coins-?,updated_at=CURRENT_TIMESTAMP WHERE fan_id=? AND coins>=?`, item.Cost, fan, item.Cost)
			if err != nil {
				return state, err
			}
			var n int64
			n, err = changed.RowsAffected()
			if err != nil {
				return state, err
			}
			if n != 1 {
				return state, ErrFanShopFunds
			}
			_, err = tx.Exec(`INSERT INTO fan_shop_items(organization_id,holder_key,fan_id,device_id,item_key,scope_event_id,title,kind,cost_coins,club_name,opponent,event_date) VALUES(?,?,?,'arena',?,0,?,'avatar',?,'','','')`, org, economyHolder(fan, ""), fan, "arena:"+item.Key, item.Title, item.Cost)
		} else if cmd.Action != "buy" {
			if owned == 0 {
				return state, ErrFanShopNotOwned
			}
			var m ArenaMember
			m, err = arenaMember(tx, org, fan)
			if err != nil {
				return state, err
			}
			if cmd.Action == "remove" {
				delete(m.Equipment, item.Slot)
			} else {
				m.Equipment[item.Slot] = item.Key
			}
			var raw []byte
			raw, err = json.Marshal(m.Equipment)
			if err == nil {
				_, err = tx.Exec(`UPDATE arena_members SET equipment=? WHERE org=? AND fan=?`, string(raw), org, fan)
			}
		}
	case "predict":
		if !arenaScore(cmd.Score) {
			return state, ErrArena
		}
		var date string
		err = tx.QueryRow(`SELECT start_datetime FROM events WHERE id=? AND organization_id=? AND is_concluded=0`, cmd.EventID, org).Scan(&date)
		if err != nil {
			return state, err
		}
		start, e := arenaDate(date)
		if e != nil || !now.Before(start.Add(-time.Duration(config.CutoffMinutes)*time.Minute)) {
			return state, ErrArena
		}
		var confirmed int
		if err = tx.QueryRow(`SELECT COUNT(*) FROM arena_results WHERE org=? AND event=?`, org, cmd.EventID).Scan(&confirmed); err != nil {
			return state, err
		}
		if confirmed > 0 {
			return state, ErrArena
		}
		_, err = tx.Exec(`INSERT INTO arena_predictions(org,fan,event,choice) VALUES(?,?,?,?) ON CONFLICT(org,fan,event) DO UPDATE SET choice=excluded.choice WHERE arena_predictions.reward IS NULL`, org, fan, cmd.EventID, cmd.Score)
	case "privacy":
		_, err = tx.Exec(`UPDATE arena_members SET public=? WHERE org=? AND fan=?`, cmd.Public, org, fan)
	case "invite":
		// Reuse a pending invitation; codes contain no identity or credentials.
		e := tx.QueryRow(`SELECT code FROM arena_invites WHERE org=? AND inviter=? AND accepted_by IS NULL ORDER BY created_at DESC LIMIT 1`, org, fan).Scan(&state.Invitation)
		if e == sql.ErrNoRows {
			b := make([]byte, 18)
			if _, err = rand.Read(b); err == nil {
				state.Invitation = hex.EncodeToString(b)
				_, err = tx.Exec(`INSERT INTO arena_invites(code,org,inviter,created_at) VALUES(?,?,?,?)`, state.Invitation, org, fan, now.Format(time.RFC3339))
			}
		} else {
			err = e
		}
	case "accept":
		var inviter int
		var accepted sql.NullInt64
		var created string
		err = tx.QueryRow(`SELECT inviter,accepted_by,created_at FROM arena_invites WHERE code=? AND org=?`, cmd.Code, org).Scan(&inviter, &accepted, &created)
		if err != nil {
			return state, ErrArena
		}
		if accepted.Valid {
			if int(accepted.Int64) != fan {
				return state, ErrArena
			}
			break
		}
		date, e := time.Parse(time.RFC3339, created)
		if e != nil || now.Sub(date) > 30*24*time.Hour || inviter == fan {
			return state, ErrArena
		}
		host, e := arenaMember(tx, org, inviter)
		if e != nil {
			return state, e
		}
		me, e := arenaMember(tx, org, fan)
		if e != nil {
			return state, e
		}
		best := me.Slot
		distance := 1 << 30
		// 665 seats per stand, 35 seats per row, matching the shared 3D map.
		used := map[int]bool{}
		rs, e := tx.Query(`SELECT slot FROM arena_members WHERE org=? AND fan<>?`, org, fan)
		if e != nil {
			return state, e
		}
		for rs.Next() {
			var s int
			if e = rs.Scan(&s); e != nil {
				rs.Close()
				return state, e
			}
			used[s] = true
		}
		rs.Close()
		for s := host.Slot / 665 * 665; s < (host.Slot/665+1)*665; s++ {
			if used[s] {
				continue
			}
			dr, dc := s%665/35-host.Slot%665/35, s%35-host.Slot%35
			d := dr*dr*80 + dc*dc*100 // approximately .49 m row rise/depth versus .55 m column spacing
			if d < distance {
				distance = d
				best = s
			}
		}
		if distance == 1<<30 {
			return state, ErrArena
		}
		_, err = tx.Exec(`UPDATE arena_members SET slot=? WHERE org=? AND fan=?`, best, org, fan)
		if err == nil {
			_, err = tx.Exec(`UPDATE arena_invites SET accepted_by=?,accepted_at=? WHERE code=? AND accepted_by IS NULL`, fan, now.Format(time.RFC3339), cmd.Code)
		}
	case "wave", "clap":
		_, err = tx.Exec(`INSERT INTO arena_actions(org,fan,kind,day,at) VALUES(?,?,?,?,?) ON CONFLICT(org,fan,kind,day) DO UPDATE SET at=excluded.at,completed=0 WHERE arena_actions.at<=excluded.at-12`, org, fan, cmd.Action, arenaDay(now), now.Unix())
	case "wave-complete":
		_, err = tx.Exec(`UPDATE arena_actions SET completed=1 WHERE org=? AND fan=? AND kind='wave' AND at<=? AND at>=?`, org, fan, now.Unix()-8, now.Unix()-20)
	case "config":
		if cmd.Config == nil {
			return state, ErrArena
		}
		c := *cmd.Config
		if c.CutoffMinutes < 0 || c.CutoffMinutes > 1440 || c.Treasure < 0 || c.Treasure > 1000 || c.Exact < 0 || c.Exact > 1000 || c.Winner < 0 || c.Winner > c.Exact || c.Growth < 1 || c.GrowthBig < c.Growth || c.Challenge < 1 || c.Collective < 2 || c.Record < 1 {
			return state, ErrArena
		}
		raw, _ := json.Marshal(c)
		config = c
		_, err = tx.Exec(`UPDATE arena_settings SET config=? WHERE org=?`, string(raw), org)
	case "result":
		if !cmd.Confirm || !arenaScore(cmd.Score) {
			return state, ErrArena
		}
		var raw string
		err = tx.QueryRow(`SELECT start_datetime FROM events WHERE id=? AND organization_id=?`, cmd.EventID, org).Scan(&raw)
		if err != nil {
			return state, err
		}
		start, e := arenaDate(raw)
		if e != nil || now.Before(start) {
			return state, ErrArena
		}
		var previous string
		e = tx.QueryRow(`SELECT score FROM arena_results WHERE org=? AND event=?`, org, cmd.EventID).Scan(&previous)
		if e == nil && previous != cmd.Score {
			return state, ErrArena
		}
		if e != nil && e != sql.ErrNoRows {
			return state, e
		}
		_, err = tx.Exec(`INSERT OR IGNORE INTO arena_results(org,event,score,confirmed_at) VALUES(?,?,?,?)`, org, cmd.EventID, cmd.Score, now.Format(time.RFC3339))
	default:
		return state, ErrArena
	}
	if err != nil {
		return state, err
	}
	// Settle only confirmed official results, once per fan/event. Points share the
	state.Config = config
	state.Treasure.Reward = config.Treasure
	// existing leaderboard account; predictions never create a second wallet.
	rs, err := tx.Query(`SELECT p.fan,p.event,p.choice,r.score FROM arena_predictions p JOIN arena_results r ON r.org=p.org AND r.event=p.event WHERE p.org=? AND p.reward IS NULL`, org)
	if err != nil {
		return state, err
	}
	type settle struct{ fan, event, points int }
	rewards := []settle{}
	for rs.Next() {
		var id, event int
		var choice, score string
		if err = rs.Scan(&id, &event, &choice, &score); err != nil {
			rs.Close()
			return state, err
		}
		points := 0
		if choice == score {
			points = config.Exact
		} else if (choice[0] == '3') == (score[0] == '3') {
			points = config.Winner
		}
		rewards = append(rewards, settle{id, event, points})
	}
	rs.Close()
	for _, r := range rewards {
		if err = ensureRegisteredEconomyAccount(tx, org, r.fan); err != nil {
			return state, err
		}
		if _, err = tx.Exec(`UPDATE fan_economy_accounts SET points=points+? WHERE organization_id=? AND owner_key=?`, r.points, org, economyOwner(0, r.fan, "")); err != nil {
			return state, err
		}
		if _, err = tx.Exec(`UPDATE arena_predictions SET reward=? WHERE org=? AND fan=? AND event=? AND reward IS NULL`, r.points, org, r.fan, r.event); err != nil {
			return state, err
		}
	}
	if err = arenaTrophies(tx, org, config, now); err != nil {
		return state, err
	}
	if fan > 0 {
		var claimed int
		if err = tx.QueryRow(`SELECT COUNT(*) FROM arena_claims WHERE org=? AND fan=? AND day=?`, org, fan, state.Treasure.Day).Scan(&claimed); err != nil {
			return state, err
		}
		state.Treasure.Claimed = claimed > 0
		err = tx.QueryRow(`SELECT w.coins,a.points FROM fan_wallets w JOIN fan_economy_accounts a ON a.fan_id=w.fan_id AND a.organization_id=? WHERE w.fan_id=?`, org, fan).Scan(&state.Wallet, &state.Points)
		if err != nil {
			return state, err
		}
		state.Level = 1 + state.Points/100
		m, e := arenaMember(tx, org, fan)
		if e == nil {
			state.Member = &m
			state.Equipment = m.Equipment
		} else if e != sql.ErrNoRows {
			return state, e
		}
		rows, err = tx.Query(`SELECT substr(item_key,7) FROM fan_shop_items WHERE organization_id=? AND fan_id=? AND kind='avatar'`, org, fan)
		if err != nil {
			return state, err
		}
		for rows.Next() {
			var key string
			if err = rows.Scan(&key); err != nil {
				rows.Close()
				return state, err
			}
			state.Owned = append(state.Owned, key)
		}
		rows.Close()
	}
	rows, err = tx.Query(`SELECT fan,slot,public,equipment FROM arena_members WHERE org=?`, org)
	if err != nil {
		return state, err
	}
	for rows.Next() {
		var m ArenaMember
		var raw string
		if err = rows.Scan(&m.FanID, &m.Slot, &m.Public, &raw); err != nil {
			rows.Close()
			return state, err
		}
		if err = json.Unmarshal([]byte(raw), &m.Equipment); err != nil {
			rows.Close()
			return state, err
		}
		state.Members = append(state.Members, m)
	}
	rows.Close()
	rows, err = tx.Query(`SELECT e.id,COALESCE(t1.name,''),COALESCE(t2.name,''),e.start_datetime,COALESCE(r.score,''),COALESCE(p.choice,''),p.reward FROM events e LEFT JOIN teams t1 ON t1.id=e.team1_id LEFT JOIN teams t2 ON t2.id=e.team2_id LEFT JOIN arena_results r ON r.org=e.organization_id AND r.event=e.id LEFT JOIN arena_predictions p ON p.org=e.organization_id AND p.event=e.id AND p.fan=? WHERE e.organization_id=? ORDER BY e.start_datetime DESC LIMIT 100`, fan, org)
	if err != nil {
		return state, err
	}
	for rows.Next() {
		var m ArenaMatch
		var reward sql.NullInt64
		if err = rows.Scan(&m.ID, &m.Home, &m.Away, &m.Starts, &m.Score, &m.Choice, &reward); err != nil {
			rows.Close()
			return state, err
		}
		start, e := arenaDate(m.Starts)
		m.Cutoff = start.Add(-time.Duration(config.CutoffMinutes) * time.Minute).Format(time.RFC3339)
		m.Open = e == nil && now.Before(start.Add(-time.Duration(config.CutoffMinutes)*time.Minute)) && m.Score == ""
		m.Settled = reward.Valid
		m.Reward = int(reward.Int64)
		state.Matches = append(state.Matches, m)
	}
	rows.Close()
	rows, err = tx.Query(`SELECT key,title,description,unlocked,value FROM arena_trophies WHERE org=?`, org)
	if err != nil {
		return state, err
	}
	unlocked := map[string]ArenaTrophy{}
	for rows.Next() {
		var t ArenaTrophy
		if err = rows.Scan(&t.Key, &t.Title, &t.Description, &t.Unlocked, &t.Value); err != nil {
			rows.Close()
			return state, err
		}
		unlocked[t.Key] = t
	}
	rows.Close()
	for _, t := range arenaMilestones(config) {
		if u, ok := unlocked[t.Key]; ok {
			u.Target = t.Target
			state.Trophies = append(state.Trophies, u)
		} else {
			state.Trophies = append(state.Trophies, t)
		}
	}
	return state, tx.Commit()
}
func absArena(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
func arenaMilestones(c ArenaConfig) []ArenaTrophy {
	return []ArenaTrophy{{Key: "fans100", Title: "Primi tifosi", Description: "Profili idonei della società", Target: c.Growth}, {Key: "fans500", Title: "La community cresce", Description: "Profili idonei della società", Target: c.GrowthBig}, {Key: "ola", Title: "Prima coreografia collettiva", Description: "Tifosi unici che avviano l’ola entro 10 secondi", Target: c.Collective}, {Key: "challenge", Title: "Cacciatori di tesori", Description: "Partecipanti unici alla caccia giornaliera", Target: c.Challenge}, {Key: "champion", Title: "Tribuna campione", Description: "Settore con più cacciatori unici nella settimana precedente", Target: 1}, {Key: "record", Title: "Record di partecipazione", Description: "Massimo di partecipanti unici in una giornata di caccia", Target: c.Record}}
}
func arenaTrophies(tx *sql.Tx, org int, c ArenaConfig, now time.Time) error {
	var fans, challenge, record, ola int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM fan_profiles WHERE organization_id=? AND registration_pending=0 AND accepted_terms=1`, org).Scan(&fans); err != nil {
		return err
	}
	if err := tx.QueryRow(`SELECT COUNT(DISTINCT fan) FROM arena_claims WHERE org=?`, org).Scan(&challenge); err != nil {
		return err
	}
	if err := tx.QueryRow(`SELECT COALESCE(MAX(n),0) FROM (SELECT COUNT(*) n FROM arena_claims WHERE org=? GROUP BY day)`, org).Scan(&record); err != nil {
		return err
	}
	if err := tx.QueryRow(`SELECT COUNT(DISTINCT fan) FROM arena_actions WHERE org=? AND kind='wave' AND completed=1 AND at>=?`, org, now.Unix()-20).Scan(&ola); err != nil {
		return err
	}
	monday := now.In(rome).AddDate(0, 0, -(int(now.In(rome).Weekday())+6)%7)
	end := monday.Format("2006-01-02")
	start := monday.AddDate(0, 0, -7).Format("2006-01-02")
	var a, b int
	if err := tx.QueryRow(`SELECT COUNT(DISTINCT CASE WHEN m.slot<665 THEN c.fan END),COUNT(DISTINCT CASE WHEN m.slot>=665 THEN c.fan END) FROM arena_claims c JOIN arena_members m ON m.org=c.org AND m.fan=c.fan WHERE c.org=? AND c.day>=? AND c.day<?`, org, start, end).Scan(&a, &b); err != nil {
		return err
	}
	champion := 0
	sector := "A"
	if a != b && a > 0 && b > 0 && a+b >= c.Collective {
		champion = 1
		if b > a {
			sector = "B"
		}
	}
	values := []int{fans, fans, ola, challenge, champion, record}
	for i, t := range arenaMilestones(c) {
		if values[i] < t.Target {
			continue
		}
		if t.Key == "champion" {
			t.Description += " · settore " + sector + " · " + start
		}
		if _, err := tx.Exec(`INSERT OR IGNORE INTO arena_trophies(org,key,title,description,unlocked,value) VALUES(?,?,?,?,?,?)`, org, t.Key, t.Title, t.Description, now.Format(time.RFC3339), values[i]); err != nil {
			return err
		}
	}
	return nil
}
