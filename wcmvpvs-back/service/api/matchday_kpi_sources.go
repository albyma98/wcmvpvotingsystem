package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

func matchdaySignalIdentity(s reportSignal) matchdayIdentity {
	return matchdayIdentity{DeviceKey: s.DeviceID, Session: s.SessionID, FanID: s.FanID}
}

func (t matchdayTracking) identity() matchdayIdentity {
	return matchdayIdentity{Device: t.Device, Session: t.Session}
}

func loadMatchdayTelemetry(ctx context.Context, q matchdayQueryer, orgID int, data map[int]*matchdayFacts) error {
	in, args := matchdayIDs(data)
	args = append([]interface{}{orgID}, args...)
	if err := matchdayRead(ctx, q, `SELECT s.event_id,s.signal_id,s.name,s.session_id,s.device_key,s.fan_id,s.occurred_at,s.metadata
		FROM engagement_report_signals s JOIN events e ON e.id=s.event_id WHERE e.organization_id=? AND s.event_id IN (`+in+`)
		ORDER BY s.occurred_at,s.signal_id`, args, func(rows *sql.Rows) error {
		var id int
		var s reportSignal
		var metadata string
		if err := rows.Scan(&id, &s.ID, &s.Name, &s.SessionID, &s.DeviceID, &s.FanID, &s.OccurredAt, &metadata); err != nil {
			return err
		}
		_ = json.Unmarshal([]byte(metadata), &s.Metadata)
		data[id].Signals = append(data[id].Signals, s)
		return nil
	}); err != nil {
		return err
	}
	return matchdayRead(ctx, q, `SELECT t.event_id,t.id,t.event_name,t.page,t.source,t.device_id,t.session_id,t.occurred_at,t.metadata_json
		FROM tracking_events t JOIN events e ON e.id=t.event_id WHERE e.organization_id=? AND t.event_id IN (`+in+`)
		AND t.organization_id IN (0,e.organization_id) AND (t.event_name IN ('session.started','page.view','game.started','game.completed')
		OR t.event_name LIKE 'bar.%' OR t.event_name LIKE 'feedback.%' OR t.event_name LIKE 'vote.%' OR t.event_name LIKE 'registration.%')
		ORDER BY t.occurred_at,t.id`, args, func(rows *sql.Rows) error {
		var id int
		var t matchdayTracking
		var metadata string
		if err := rows.Scan(&id, &t.ID, &t.Name, &t.Page, &t.Source, &t.Device, &t.Session, &t.At, &metadata); err != nil {
			return err
		}
		_ = json.Unmarshal([]byte(metadata), &t.Metadata)
		data[id].Tracking = append(data[id].Tracking, t)
		return nil
	})
}

func loadMatchdayPageEngagement(ctx context.Context, q matchdayQueryer, orgID int, data map[int]*matchdayFacts) error {
	in, args := matchdayIDs(data)
	return matchdayRead(ctx, q, `SELECT p.event_id,p.device_id,COUNT(*),SUM(p.duration_seconds) FROM page_engagements p
		JOIN events e ON e.id=p.event_id WHERE e.organization_id=? AND p.event_id IN (`+in+`)
		AND p.duration_seconds>0 GROUP BY p.event_id,p.device_id`, append([]interface{}{orgID}, args...), func(rows *sql.Rows) error {
		var eventID, observations, seconds int
		var device string
		if err := rows.Scan(&eventID, &device, &observations, &seconds); err != nil {
			return err
		}
		f := data[eventID]
		f.legacyDevices = append(f.legacyDevices, device)
		f.legacyDurations += observations
		f.legacySeconds += seconds
		return nil
	})
}

type matchdayVote struct {
	identity   matchdayIdentity
	playerID   int
	playerName string
}

type matchdayGameObservation struct {
	identity            matchdayIdentity
	game, round, source string
	start, completed    bool
	abandoned           bool
	live                bool
}

func loadMatchdayVotesAndGames(ctx context.Context, q matchdayQueryer, orgID int, data map[int]*matchdayFacts) error {
	in, args := matchdayIDs(data)
	args = append([]interface{}{orgID}, args...)
	if err := matchdayRead(ctx, q, `SELECT v.event_id,COALESCE(v.user_id,0),COALESCE(v.device_id,''),v.player_id,
		TRIM(COALESCE(p.first_name,'')||' '||COALESCE(p.last_name,'')) FROM votes v
		JOIN events e ON e.id=v.event_id LEFT JOIN players p ON p.id=v.player_id AND p.team_id IN (e.team1_id,e.team2_id)
		WHERE e.organization_id=? AND v.event_id IN (`+in+`)`, args, func(rows *sql.Rows) error {
		var id int
		var v matchdayVote
		if err := rows.Scan(&id, &v.identity.FanID, &v.identity.Device, &v.playerID, &v.playerName); err != nil {
			return err
		}
		data[id].votes = append(data[id].votes, v)
		data[id].Actions = append(data[id].Actions, matchdayAction{Identity: v.identity, Feature: "mvp", Count: 1})
		return nil
	}); err != nil {
		return err
	}
	for _, f := range data {
		for _, s := range f.Signals {
			if s.Name == "game_started" || s.Name == "game_completed" || s.Name == "game_abandoned" {
				if game := reportMetricString(s.Metadata, "game_id"); game != "" {
					f.games = append(f.games, matchdayGameObservation{matchdaySignalIdentity(s), game,
						reportMetricString(s.Metadata, "round_id"), "engagement_report_signals", s.Name == "game_started", s.Name == "game_completed", s.Name == "game_abandoned",
						reportMetricString(s.Metadata, "mode") == "live"})
				}
			}
		}
		for _, t := range f.Tracking {
			if t.Name != "game.started" && t.Name != "game.completed" {
				continue
			}
			if game := reportMetricString(t.Metadata, "game_id"); game != "" {
				f.games = append(f.games, matchdayGameObservation{t.identity(), game,
					reportMetricString(t.Metadata, "round_id"), "tracking_events", t.Name == "game.started", t.Name == "game.completed", false,
					reportMetricString(t.Metadata, "mode") == "live"})
			}
		}
	}
	// All structured queries yield event, raw device, fan, game, round, start and outcome (1 completed, 2 abandoned).
	// Issuing a hoop target or preparing a volleyball round does not certify a start.
	for _, spec := range []struct{ source, query string }{
		{"basketball_shots", `SELECT event_id,device_id,fan_id,'basketball',shot_id,1,1 FROM basketball_shots WHERE organization_id=? AND status IN ('made','miss') AND event_id IN (`},
		{"volleyball_rounds", `SELECT event_id,device_id,fan_id,game_id,round_id,1,1 FROM volleyball_rounds WHERE organization_id=? AND status IN ('success','miss') AND event_id IN (`},
		{"playmaker_sessions", `SELECT event_id,device_id,fan_id,'playmaker',session_id,1,0 FROM playmaker_sessions WHERE organization_id=? AND event_id IN (`},
		{"crossover_sessions", `SELECT event_id,device_id,fan_id,'crossover',session_id,1,next_duel>=3 FROM crossover_sessions WHERE organization_id=? AND event_id IN (`},
		{"fan_minigame_rewards", `SELECT event_id,device_id,fan_id,reward_key,request_id,1,1 FROM fan_minigame_rewards WHERE organization_id=? AND reward_key IN ('reaction','quiz','tap','tap-volley','sponsor-rush','memory-flash') AND coins>0 AND event_id IN (`},
		{"fan_journey_rewards", `SELECT event_id,device_id,fan_id,reward_key,'',0,1 FROM fan_journey_rewards WHERE organization_id=? AND reward_key IN ('reaction','quiz','tap','tap-volley','sponsor-rush','memory-flash') AND coins>0 AND event_id IN (`},
		{"fan_minigame_quiz_answers", `SELECT DISTINCT event_id,device_id,0,'quiz',request_id,1,0 FROM fan_minigame_quiz_answers WHERE organization_id=? AND event_id IN (`},
		{"fan_journey_quiz_answers", `SELECT DISTINCT event_id,device_id,0,'quiz','',1,0 FROM fan_journey_quiz_answers WHERE organization_id=? AND event_id IN (`},
		{"reaction_tests", `SELECT r.event_id,r.device_id,0,'reaction','reaction:'||r.id,1,1 FROM reaction_tests r JOIN events e ON e.id=r.event_id WHERE e.organization_id=? AND r.is_valid=1 AND r.event_id IN (`},
		{"branded_game_participations", `SELECT g.event_id,g.device_id,COALESCE(g.user_id,0),'branded_game',CAST(g.id AS TEXT),1,g.completed FROM branded_game_participations g JOIN events e ON e.id=g.event_id WHERE e.organization_id=? AND g.event_id IN (`},
		{"tap_live_matches", `SELECT g.event_id,'',CASE p.n WHEN 1 THEN g.fan1_id ELSE g.fan2_id END,g.game_id,g.match_id,1,
			CASE WHEN g.status='finished' AND COALESCE(g.abandoned_by_fan_id,0)>0 THEN 2 WHEN g.status='finished' THEN 1 ELSE 0 END
			FROM tap_live_matches g CROSS JOIN (SELECT 1 AS n UNION ALL SELECT 2) p WHERE g.organization_id=? AND g.started_at IS NOT NULL AND g.event_id IN (`},
	} {
		if err := matchdayRead(ctx, q, spec.query+in+`)`, args, func(rows *sql.Rows) error {
			var id, outcome int
			g := matchdayGameObservation{source: spec.source, live: spec.source == "tap_live_matches"}
			if err := rows.Scan(&id, &g.identity.Device, &g.identity.FanID, &g.game, &g.round, &g.start, &outcome); err != nil {
				return err
			}
			g.completed, g.abandoned = outcome == 1, outcome == 2
			data[id].games = append(data[id].games, g)
			return nil
		}); err != nil {
			return err
		}
	}
	if err := matchdayRead(ctx, q, `SELECT p.event_id, COUNT(*) FROM event_prizes p JOIN events e ON e.id=p.event_id
		WHERE e.organization_id=? AND p.event_id IN (`+in+`) AND (p.winner_user_id IS NOT NULL OR p.winner_vote_id IS NOT NULL OR p.winner_lottery_code<>'') GROUP BY p.event_id`, args, func(rows *sql.Rows) error {
		var eventID, prizes int
		if err := rows.Scan(&eventID, &prizes); err != nil {
			return err
		}
		data[eventID].rewards = prizes
		return nil
	}); err != nil {
		return err
	}
	return matchdayRead(ctx, q, `SELECT l.event_id,COUNT(DISTINCT l.fan_id) FROM fan_lottery_entries l JOIN events e ON e.id=l.event_id
		JOIN fan_profiles f ON f.id=l.fan_id AND f.organization_id=e.organization_id
		WHERE e.organization_id=? AND l.event_id IN (`+in+`) GROUP BY l.event_id`, args, func(rows *sql.Rows) error {
		var eventID, count int
		if err := rows.Scan(&eventID, &count); err != nil {
			return err
		}
		data[eventID].lottery = count
		return nil
	})
}

type matchdayProfile struct {
	created, accepted time.Time
}

type matchdayIdentityLinks struct {
	devices, sessions map[string]string
}

func matchdayLink(links map[string]string, key, value string) {
	if key == "" || value == "" {
		return
	}
	if prior, found := links[key]; found && prior != value {
		links[key] = ""
	} else if !found {
		links[key] = value
	}
}

func loadMatchdayRegistrations(ctx context.Context, q matchdayQueryer, orgID int, data map[int]*matchdayFacts) error {
	in, args := matchdayIDs(data)
	return matchdayRead(ctx, q, `SELECT r.event_id,r.fan_id,r.device_key FROM fan_matchday_registrations r
		JOIN fan_profiles p ON p.id=r.fan_id AND p.organization_id=r.organization_id AND p.registration_pending=0
		WHERE r.organization_id=? AND r.event_id IN (`+in+`)`, append([]interface{}{orgID}, args...), func(rows *sql.Rows) error {
		var id int
		var identity matchdayIdentity
		if err := rows.Scan(&id, &identity.FanID, &identity.DeviceKey); err != nil {
			return err
		}
		data[id].registrations = append(data[id].registrations, identity)
		return nil
	})
}

func calculateMatchdayAudience(ctx context.Context, q matchdayQueryer, orgID int, ordered []*matchdayFacts, loadAttributedActions func() error) error {
	profiles := map[int]matchdayProfile{}
	if err := matchdayRead(ctx, q, `SELECT p.id,p.created_at,COALESCE(MIN(a.accepted_at),'')
		FROM fan_profiles p LEFT JOIN fan_registration_acceptances a ON a.fan_id=p.id AND a.organization_id=p.organization_id
		WHERE p.organization_id=? AND p.registration_pending=0 GROUP BY p.id`, []interface{}{orgID}, func(rows *sql.Rows) error {
		var id int
		var created, accepted string
		var p matchdayProfile
		if err := rows.Scan(&id, &created, &accepted); err != nil {
			return err
		}
		p.created, _ = parseEventStart(created)
		p.accepted, _ = parseEventStart(accepted)
		profiles[id] = p
		return nil
	}); err != nil {
		return err
	}
	device := func(id matchdayIdentity) string {
		if id.DeviceKey != "" {
			return id.DeviceKey
		}
		return reportOperationalDeviceKey(orgID, id.Device)
	}
	profile := func(id matchdayIdentity) string {
		if _, ok := profiles[id.FanID]; ok && id.FanID > 0 {
			return "fan:" + strconv.Itoa(id.FanID)
		}
		return ""
	}
	allDevices := map[string]string{}
	local := map[int]matchdayIdentityLinks{}
	for _, f := range ordered {
		links := matchdayIdentityLinks{devices: map[string]string{}, sessions: map[string]string{}}
		local[f.Event.ID] = links
		f.eachIdentity(func(id matchdayIdentity) {
			matchdayLink(links.devices, device(id), profile(id))
		})
	}
	for key, owner := range local[ordered[0].Event.ID].devices {
		allDevices[key] = owner
	}
	// Keep one observation per identity tuple and historical event for first observed
	// acquisition. This pass is independent of the chart limit and division filter.
	type sighting struct {
		event    int
		at       time.Time
		identity matchdayIdentity
	}
	var historical []sighting
	historicalLinks := map[int]matchdayIdentityLinks{}
	if current := ordered[0].Event.start; !current.IsZero() {
		err := matchdayRead(ctx, q, `WITH past AS (SELECT id,start_datetime,organization_id FROM events WHERE organization_id=? AND julianday(start_datetime)<julianday(?))
		SELECT e.id,e.start_datetime,s.device_key,COALESCE(s.fan_id,0),1 FROM past e JOIN engagement_report_signals s ON s.event_id=e.id
		WHERE s.name IN ('page_view','registration_completed') OR s.fan_id>0 GROUP BY e.id,s.device_key,s.fan_id
		UNION ALL SELECT e.id,e.start_datetime,t.device_id,0,0 FROM past e JOIN tracking_events t ON t.event_id=e.id
		WHERE t.organization_id IN (0,e.organization_id) AND t.event_name IN ('session.started','page.view') GROUP BY e.id,t.device_id
		UNION ALL SELECT e.id,e.start_datetime,COALESCE(v.device_id,''),COALESCE(v.user_id,0),0 FROM past e JOIN votes v ON v.event_id=e.id GROUP BY e.id,v.device_id,v.user_id
		UNION ALL SELECT e.id,e.start_datetime,p.device_id,0,0 FROM past e JOIN page_engagements p ON p.event_id=e.id WHERE p.duration_seconds>0 GROUP BY e.id,p.device_id
		UNION ALL SELECT e.id,e.start_datetime,r.device_key,r.fan_id,1 FROM past e JOIN fan_matchday_registrations r ON r.event_id=e.id AND r.organization_id=e.organization_id`, []interface{}{orgID, current.Format(time.RFC3339)}, func(rows *sql.Rows) error {
			var s sighting
			var at, key string
			var hashed bool
			if err := rows.Scan(&s.event, &at, &key, &s.identity.FanID, &hashed); err != nil {
				return err
			}
			s.at, _ = parseEventStart(at)
			s.identity.Device = key
			if hashed {
				s.identity.Device, s.identity.DeviceKey = "", key
			}
			key, owner := device(s.identity), profile(s.identity)
			if key != "" && owner != "" {
				links := historicalLinks[s.event]
				if links.devices == nil {
					links.devices = map[string]string{}
					historicalLinks[s.event] = links
				}
				matchdayLink(links.devices, key, owner)
				matchdayLink(allDevices, key, owner)
			}
			historical = append(historical, s)
			return nil
		})
		if err != nil {
			return err
		}
	}
	resolveDevice := func(id matchdayIdentity, links matchdayIdentityLinks) string {
		if owner := profile(id); owner != "" {
			return owner
		}
		key := device(id)
		if key == "" {
			return ""
		}
		if owner, seen := links.devices[key]; seen {
			if owner != "" {
				return owner
			}
		} else if owner := allDevices[key]; owner != "" {
			return owner
		}
		return "device:" + key
	}
	firstSeen, firstEvent := map[string]time.Time{}, map[string]int{}
	persistent := func(actor string) bool {
		if strings.HasPrefix(actor, "device:") {
			owner, linked := allDevices[strings.TrimPrefix(actor, "device:")]
			return !linked || owner != ""
		}
		return strings.HasPrefix(actor, "fan:")
	}
	observe := func(actor string, event int, at time.Time) {
		if !persistent(actor) || at.IsZero() {
			return
		}
		prior := firstSeen[actor]
		if prior.IsZero() || at.Before(prior) || (at.Equal(prior) && event < firstEvent[actor]) {
			firstSeen[actor], firstEvent[actor] = at, event
		}
	}
	for _, s := range historical {
		observe(resolveDevice(s.identity, historicalLinks[s.event]), s.event, s.at)
	}
	for _, f := range ordered {
		eventID := f.Event.ID
		links := local[eventID]
		f.eachIdentity(func(id matchdayIdentity) { matchdayLink(links.sessions, id.Session, resolveDevice(id, links)) })
		f.Resolve = func(id matchdayIdentity) string {
			if actor := resolveDevice(id, links); actor != "" {
				return actor
			}
			if id.Session == "" {
				return ""
			}
			if actor := links.sessions[id.Session]; actor != "" {
				return actor
			}
			return "session:" + strconv.Itoa(eventID) + ":" + id.Session
		}
	}
	if err := loadAttributedActions(); err != nil {
		return err
	}
	for _, f := range ordered {
		f.coverage = matchdayAudienceCoverage(f)
		calculateMatchdayGames(f)
		f.Reach, f.FirstEvent = map[string]bool{}, firstEvent
		f.persistentReach = map[string]bool{}
		identified := map[int]bool{}
		f.eachIdentity(func(id matchdayIdentity) {
			if actor := f.Resolve(id); actor != "" {
				f.Reach[actor] = true
				if persistent(actor) {
					f.persistentReach[actor] = true
				}
				if f == ordered[0] {
					observe(actor, f.Event.ID, f.Event.start)
				}
			}
			if profile(id) != "" {
				identified[id.FanID] = true
			}
		})
		calculateMatchdayCRM(f, profiles, identified)
	}
	return nil
}

func matchdayComparableIdentity(id string) bool {
	return strings.HasPrefix(id, "fan:") || strings.HasPrefix(id, "device:")
}

func (f *matchdayFacts) eachIdentity(use func(matchdayIdentity)) {
	f.Operations.eachIdentity(use)
	for _, identity := range f.registrations {
		use(identity)
	}
	if f.feedback != nil {
		for _, response := range f.feedback.tracked {
			use(response.identity)
		}
	}
	for _, device := range f.legacyDevices {
		use(matchdayIdentity{Device: device})
	}
	for _, s := range f.Signals {
		use(matchdaySignalIdentity(s))
	}
	for _, t := range f.Tracking {
		use(t.identity())
	}
	for _, g := range f.games {
		use(g.identity)
	}
	for _, a := range f.Actions {
		use(a.Identity)
	}
}
