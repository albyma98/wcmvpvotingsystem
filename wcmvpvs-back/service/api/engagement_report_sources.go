package api

// This adapter reads event operations used by the fan home directly. It does not
// read event-history analytics or any tournament tables. Operational evidence is
// not a page impression: callers must keep its coverage distinct from telemetry.
import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type reportSourceRows struct {
	Table string `json:"table"`
	Rows  int    `json:"rows"`
}

type reportOperationalSummary struct {
	Votes                        int `json:"votes"`
	IdentifiedParticipants       int `json:"identified_participants"`
	GameRewardReceipts           int `json:"game_reward_receipts"`
	ShopRedemptions              int `json:"shop_redemptions"`
	CoinsEarned                  int `json:"coins_earned"`
	CoinsSpent                   int `json:"coins_spent"`
	FeedbackResponses            int `json:"feedback_responses"`
	SponsorClicks                int `json:"sponsor_clicks"`
	SponsorClicksWithoutIdentity int `json:"sponsor_clicks_without_identity"`
}

type reportOperationalFeedback struct {
	PerksResponses int `json:"perks_responses"`
	PerksPositive  int `json:"perks_positive"`
}

type reportOperationalData struct {
	Signals  []reportSignal            `json:"-"`
	Sources  []reportSourceRows        `json:"sources"`
	Summary  reportOperationalSummary  `json:"summary"`
	Feedback reportOperationalFeedback `json:"feedback"`
}

func reportOperationalDeviceKey(orgID int, deviceID string) string {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(strconv.Itoa(orgID) + ":" + deviceID))
	return hex.EncodeToString(hash[:])
}

func reportOperationalEpoch(milliseconds int64) string {
	if milliseconds <= 0 {
		return ""
	}
	return time.UnixMilli(milliseconds).UTC().Format(time.RFC3339Nano)
}

func reportOperationalGame(key string) string {
	for _, game := range []string{"reaction", "quiz", "tap", "sponsor-rush", "memory-flash", "basketball", "playmaker", "crossover"} {
		if key == game {
			return game
		}
	}
	return ""
}

func (rt *_router) loadReportOperationalData(eventID, orgID int) (reportOperationalData, error) {
	out := reportOperationalData{Signals: []reportSignal{}, Sources: []reportSourceRows{}}
	conn := rt.db.SQLConn()
	verified := map[int]bool{}
	// Validate stored profile IDs against this tenant. Never export profile PII.
	rows, err := conn.Query(`SELECT p.id FROM fan_profiles p JOIN events e ON e.organization_id=p.organization_id WHERE e.id=? AND e.organization_id=? AND p.registration_pending=0`, eventID, orgID)
	if err != nil {
		return out, fmt.Errorf("report operational profiles: %w", err)
	}
	for rows.Next() {
		var fanID int
		if err = rows.Scan(&fanID); err != nil {
			_ = rows.Close()
			return out, err
		}
		verified[fanID] = true
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return out, err
	}
	out.Sources = append(out.Sources, reportSourceRows{Table: "fan_profiles", Rows: len(verified)})
	participants := map[int]bool{}
	add := func(table, id, name, device string, fanID int, at string, meta map[string]interface{}) {
		if !verified[fanID] {
			fanID = 0
		}
		deviceKey := reportOperationalDeviceKey(orgID, device)
		if deviceKey == "" && fanID == 0 {
			return
		}
		if fanID > 0 {
			participants[fanID] = true
		}
		if meta == nil {
			meta = map[string]interface{}{}
		}
		meta["_operational"] = true
		meta["_table"] = table
		// Composite legacy primary keys contain raw device IDs. Keep even the
		// internal synthetic identifier opaque so it cannot leak via diagnostics.
		idHash := sha256.Sum256([]byte(id))
		out.Signals = append(out.Signals, reportSignal{ID: "operational:" + table + ":" + hex.EncodeToString(idHash[:]), Name: name, SessionID: "", DeviceID: deviceKey, FanID: fanID, OccurredAt: at, Metadata: meta})
	}
	read := func(table, query string, scan func(*sql.Rows) error) error {
		rows, err := conn.Query(query, eventID, orgID)
		if err != nil {
			return fmt.Errorf("report operational %s: %w", table, err)
		}
		defer rows.Close()
		source := reportSourceRows{Table: table}
		for rows.Next() {
			if err = scan(rows); err != nil {
				return fmt.Errorf("report operational %s: %w", table, err)
			}
			source.Rows++
		}
		if err = rows.Err(); err != nil {
			return fmt.Errorf("report operational %s: %w", table, err)
		}
		out.Sources = append(out.Sources, source)
		return nil
	}
	if err = read("votes", `SELECT v.id,COALESCE(v.device_id,''),COALESCE(v.user_id,0),COALESCE(v.created_at,'') FROM votes v JOIN events e ON e.id=v.event_id WHERE v.event_id=? AND e.organization_id=?`, func(rows *sql.Rows) error {
		var id, fanID int
		var device, at string
		if err := rows.Scan(&id, &device, &fanID, &at); err != nil {
			return err
		}
		out.Summary.Votes++
		add("votes", strconv.Itoa(id), "vote_completed", device, fanID, at, nil)
		return nil
	}); err != nil {
		return out, err
	}
	for _, spec := range []struct {
		table, query string
		legacy       bool
	}{
		{"fan_minigame_rewards", `SELECT request_id,device_id,fan_id,reward_key,coins,created_at FROM fan_minigame_rewards WHERE event_id=? AND organization_id=?`, false},
		{"fan_journey_rewards", `SELECT device_id||':'||reward_key,device_id,fan_id,reward_key,coins,created_at FROM fan_journey_rewards WHERE event_id=? AND organization_id=?`, true},
	} {
		if err = read(spec.table, spec.query, func(rows *sql.Rows) error {
			var id, device, key, at string
			var fanID, coins int
			if err := rows.Scan(&id, &device, &fanID, &key, &coins, &at); err != nil {
				return err
			}
			game := reportOperationalGame(key)
			if coins <= 0 || game == "" || game == "basketball" || game == "playmaker" || game == "crossover" {
				return nil
			}
			out.Summary.GameRewardReceipts++
			round := id
			meta := map[string]interface{}{"game_id": game}
			if spec.legacy {
				round = "legacy:" + key
				meta["_legacy_round"] = true
			}
			meta["round_id"] = round
			add(spec.table, id, "game_completed", device, fanID, at, meta)
			return nil
		}); err != nil {
			return out, err
		}
	}
	for _, spec := range []struct {
		table, query string
		legacy       bool
	}{
		{"fan_minigame_quiz_answers", `SELECT device_id||':'||request_id||':'||question_id,device_id,request_id FROM fan_minigame_quiz_answers WHERE event_id=? AND organization_id=?`, false},
		{"fan_journey_quiz_answers", `SELECT device_id||':'||question_id,device_id,'' FROM fan_journey_quiz_answers WHERE event_id=? AND organization_id=?`, true},
	} {
		if err = read(spec.table, spec.query, func(rows *sql.Rows) error {
			var id, device, round string
			if err := rows.Scan(&id, &device, &round); err != nil {
				return err
			}
			meta := map[string]interface{}{"game_id": "quiz", "round_id": round}
			if spec.legacy {
				meta["round_id"] = "legacy:quiz"
				meta["_legacy_round"] = true
			}
			// Answer tables do not store a creation time or a fan ID.
			add(spec.table, id, "game_started", device, 0, "", meta)
			return nil
		}); err != nil {
			return out, err
		}
	}
	if err = read("basketball_shots", `SELECT shot_id,device_id,fan_id,status,issued_at FROM basketball_shots WHERE event_id=? AND organization_id=?`, func(rows *sql.Rows) error {
		var id, device, status string
		var fanID int
		var issued int64
		if err := rows.Scan(&id, &device, &fanID, &status, &issued); err != nil {
			return err
		}
		if status == "made" || status == "miss" {
			// Basketball's fan component defines each resolved shot as a round
			// and emits the same shot_id on its browser completion signal.
			add("basketball_shots", id, "game_completed", device, fanID, reportOperationalEpoch(issued), map[string]interface{}{"game_id": "basketball", "round_id": id})
		}
		return nil
	}); err != nil {
		return out, err
	}
	for _, spec := range []struct{ table, game, query string }{
		{"playmaker_sessions", "playmaker", `SELECT session_id,device_id,fan_id,started_at FROM playmaker_sessions WHERE event_id=? AND organization_id=?`},
		{"playmaker_passes", "playmaker", `SELECT session_id||':'||attempt,device_id,fan_id,resolved_at,session_id FROM playmaker_passes WHERE event_id=? AND organization_id=?`},
		{"crossover_sessions", "crossover", `SELECT session_id,device_id,fan_id,started_at FROM crossover_sessions WHERE event_id=? AND organization_id=?`},
		{"crossover_duels", "crossover", `SELECT session_id||':'||duel,device_id,fan_id,resolved_at,session_id FROM crossover_duels WHERE event_id=? AND organization_id=?`},
	} {
		if err = read(spec.table, spec.query, func(rows *sql.Rows) error {
			var id, device, round string
			var fanID int
			var at int64
			if strings.HasSuffix(spec.table, "_sessions") {
				if err := rows.Scan(&id, &device, &fanID, &at); err != nil {
					return err
				}
				round = id
			} else if err := rows.Scan(&id, &device, &fanID, &at, &round); err != nil {
				return err
			}
			// Session closure/expiration does not certify a completed round.
			add(spec.table, id, "game_started", device, fanID, reportOperationalEpoch(at), map[string]interface{}{"game_id": spec.game, "round_id": round})
			return nil
		}); err != nil {
			return out, err
		}
	}
	if err = read("fan_wallet_movements", `SELECT id,device_id,fan_id,source,delta,created_at FROM fan_wallet_movements WHERE event_id=? AND organization_id=? AND source IN ('basketball','playmaker','crossover','journey:vote','journey:reaction','journey:quiz','journey:tap','journey:sponsor-rush','journey:memory-flash','journey:basketball','journey:playmaker','journey:crossover','minigame:reaction','minigame:quiz','minigame:tap','minigame:sponsor-rush','minigame:memory-flash','minigame:basketball','minigame:playmaker','minigame:crossover','shop:club_frame','shop:match_card')`, func(rows *sql.Rows) error {
		var id, fanID, delta int
		var device, source, at string
		if err := rows.Scan(&id, &device, &fanID, &source, &delta, &at); err != nil {
			return err
		}
		if delta > 0 {
			out.Summary.CoinsEarned += delta
		}
		if delta < 0 {
			out.Summary.CoinsSpent -= delta
		}
		if delta < 0 && strings.HasPrefix(source, "shop:") {
			out.Summary.ShopRedemptions++
			add("fan_wallet_movements", strconv.Itoa(id), "redemption_completed", device, fanID, at, map[string]interface{}{"item_key": strings.TrimPrefix(source, "shop:")})
		} else if delta > 0 {
			key := strings.TrimPrefix(strings.TrimPrefix(source, "minigame:"), "journey:")
			if key == "vote" {
				add("fan_wallet_movements", strconv.Itoa(id), "vote_completed", device, fanID, at, nil)
			} else if game := reportOperationalGame(key); game != "" {
				// The ledger corroborates participation; receipts own round IDs.
				add("fan_wallet_movements", strconv.Itoa(id), "game_started", device, fanID, at, map[string]interface{}{"game_id": game})
			}
		}
		return nil
	}); err != nil {
		return out, err
	}
	if err = read("event_feedback", `SELECT f.perks_interest FROM event_feedback f JOIN events e ON e.id=f.event_id WHERE f.event_id=? AND e.organization_id=?`, func(rows *sql.Rows) error {
		var perks string
		if err := rows.Scan(&perks); err != nil {
			return err
		}
		out.Summary.FeedbackResponses++
		switch perks {
		case "yes":
			out.Feedback.PerksPositive++
			out.Feedback.PerksResponses++
		case "maybe", "little", "no":
			out.Feedback.PerksResponses++
		}
		// Feedback has no participant ID and its experience field describes
		// voting ease, not the new overall-experience satisfaction question.
		return nil
	}); err != nil {
		return out, err
	}
	if err = read("sponsor_clicks", `SELECT c.id,COALESCE(c.device_id,''),c.sponsor_id,COALESCE(NULLIF(s.report_name,''),s.name),c.clicked_at FROM sponsor_clicks c JOIN events e ON e.id=c.event_id JOIN sponsors s ON s.id=c.sponsor_id AND s.organization_id=e.organization_id WHERE c.event_id=? AND e.organization_id=?`, func(rows *sql.Rows) error {
		var id, sponsor int
		var device, name, at string
		if err := rows.Scan(&id, &device, &sponsor, &name, &at); err != nil {
			return err
		}
		out.Summary.SponsorClicks++
		if strings.TrimSpace(device) == "" {
			out.Summary.SponsorClicksWithoutIdentity++
			return nil
		}
		add("sponsor_clicks", strconv.Itoa(id), "sponsor_click", device, 0, at, map[string]interface{}{"sponsor_id": sponsor, "name": name})
		return nil
	}); err != nil {
		return out, err
	}
	out.Summary.IdentifiedParticipants = len(participants)
	return out, nil
}
