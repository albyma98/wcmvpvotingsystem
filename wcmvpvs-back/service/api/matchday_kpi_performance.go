package api

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"
)

func matchdayPartial(m matchdayMetric, reason string) matchdayMetric {
	if m.Status == "available" {
		m.Status, m.Reason = "partial", reason
	}
	return m
}

func calculateMatchdayGames(f *matchdayFacts) {
	type round struct{ started, completed bool }
	type entry struct {
		players map[string]bool
		rounds  map[string]round
	}
	strong := map[string]bool{}
	modernReaction := map[string]bool{}
	for _, g := range f.games {
		actor := f.Resolve(g.identity)
		if g.round != "" && g.source != "reaction_tests" {
			strong[actor+"\x00"+g.game] = true
			if g.game == "reaction" {
				modernReaction[actor] = true
			}
		}
	}
	players, sources := map[string]bool{}, map[string]bool{}
	for _, signal := range f.Signals {
		if gamesAvailable, _ := signal.Metadata["games_available"].(bool); signal.Name == "page_view" && gamesAvailable {
			sources["engagement_report_signals"] = true
			break
		}
	}
	playerStarts := map[string]int{}
	games := map[string]*entry{}
	for _, g := range f.games {
		actor := f.Resolve(g.identity)
		if g.game == "" || (g.source == "reaction_tests" && modernReaction[actor]) {
			continue
		}
		if actor != "" {
			players[actor] = true
		}
		if games[g.game] == nil {
			games[g.game] = &entry{map[string]bool{}, map[string]round{}}
		}
		e := games[g.game]
		if actor != "" {
			e.players[actor] = true
		}
		sources[g.source] = true
		if g.round == "" {
			if actor != "" && !strong[actor+"\x00"+g.game] && strings.HasPrefix(g.source, "fan_journey_") {
				g.round = "legacy:" + actor
			} else {
				continue
			}
		}
		key := g.round
		// A live head-to-head round is a single match with two participants.
		if g.source != "tap_live_matches" && !strings.HasPrefix(g.game, "tap") {
			key = actor + "\x00" + g.round
		}
		state := e.rounds[key]
		if g.start && (!state.started || g.source == "tap_live_matches") && actor != "" {
			if g.source == "tap_live_matches" {
				marker := "live:" + actor + "\x00" + g.round
				if strong[marker] {
					continue
				}
				strong[marker] = true
			}
			playerStarts[actor]++
			f.Actions = append(f.Actions, matchdayAction{Identity: g.identity, Feature: "game", Count: 1})
		} else if g.start && !state.started {
			f.Actions = append(f.Actions, matchdayAction{Identity: g.identity, Feature: "game", Count: 1})
		}
		state.started = state.started || g.start
		state.completed = state.completed || g.completed
		e.rounds[key] = state
	}
	gameSources := make([]string, 0, len(sources))
	for s := range sources {
		gameSources = append(gameSources, s)
	}
	sort.Strings(gameSources)
	starts, completed, matched, replays, participatingStarts := 0, 0, 0, 0, 0
	rows := make([]matchdayRow, 0, len(games))
	for game, e := range games {
		var gs, gc int
		for _, r := range e.rounds {
			if r.started {
				gs++
			}
			if r.completed {
				gc++
				if r.started {
					matched++
				}
			}
		}
		starts, completed = starts+gs, completed+gc
		rows = append(rows, matchdayRow{Label: game, Values: map[string]*float64{"players": reportMetricFloat(float64(len(e.players))), "starts": reportMetricFloat(float64(gs)), "completed": reportMetricFloat(float64(gc))}})
	}
	for _, count := range playerStarts {
		participatingStarts += count
		if count >= 2 {
			replays++
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if *a.Values["players"] != *b.Values["players"] {
			return *a.Values["players"] > *b.Values["players"]
		}
		return a.Label < b.Label
	})
	gameMetrics := []matchdayMetric{
		matchdayCount("games_players", "Giocatori unici", "Identità con evidenza di gioco.", "Identità distinte nei round osservati", float64(len(players)), gameSources...),
		matchdayCount("games_starts", "Partite avviate", "Round con avvio osservato o confermato dal gioco.", "Round distinti con avvio", float64(starts), gameSources...),
		matchdayCount("games_completions", "Partite completate", "Round con conclusione osservata o confermata dal gioco.", "Round distinti con completamento", float64(completed), gameSources...),
		matchdayRatio("games_completion_rate", "Completion rate giochi", "Completamenti abbinati agli avvii dello stesso round.", "Round avviati e completati / avvii × 100", "percent", float64(matched), float64(starts), gameSources...),
		matchdayRatio("games_replay_rate", "Replay rate", "Giocatori con almeno due avvii osservati.", "Giocatori con ≥ 2 avvii / giocatori unici × 100", "percent", float64(replays), float64(len(players)), gameSources...),
		matchdayRatio("games_per_player", "Sessioni di gioco per giocatore", "Conta la partecipazione individuale anche nei giochi a due.", "Avvii attribuibili a un giocatore / giocatori unici", "ratio", float64(participatingStarts), float64(len(players)), gameSources...),
		{Code: "games_popular", Name: "Giochi più utilizzati", Description: "Partecipazione e round per titolo.", Formula: "Giocatori, avvii e completamenti osservati per gioco", Unit: "count", Source: "db", Status: "available", Sources: gameSources, Rows: rows, Columns: []matchdayColumn{{"players", "Giocatori", "count"}, {"starts", "Avvii", "count"}, {"completed", "Completamenti", "count"}}},
	}
	for _, m := range gameMetrics {
		if len(gameSources) == 0 {
			m.Value, m.Numerator, m.Denominator, m.Status, m.Reason = nil, nil, nil, "missing", "Nessuna raccolta di partite osservata per questo evento."
		} else {
			m = matchdayPartial(m, "Copertura dei giochi e identificativi di round variabili. Il tasso usa solo completamenti abbinati a un avvio.")
		}
		matchdayPut(f, m)
	}
	matchdayPut(f, matchdayCount("games_rewards", "Partecipanti all’estrazione", "Profili con un’iscrizione salvata all’estrazione dell’evento.", "Profili distinti con iscrizione", float64(f.lottery), "fan_lottery_entries"),
		matchdayCount("games_prizes", "Premi assegnati", "Premi con vincitore registrato.", "Premi con vincitore", float64(f.rewards), "event_prizes"))
}

func matchdayAudienceCoverage(f *matchdayFacts) string {
	coverageSources := []string{}
	for _, s := range f.Signals {
		if s.Name == "page_view" {
			coverageSources = append(coverageSources, "engagement_report_signals")
			break
		}
	}
	for _, t := range f.Tracking {
		if t.Name == "page.view" || t.Name == "session.started" {
			coverageSources = append(coverageSources, "tracking_events")
			break
		}
	}
	if f.legacyDurations > 0 {
		coverageSources = append(coverageSources, "page_engagements")
	}
	if len(coverageSources) == 0 {
		return "operational"
	}
	return strings.Join(coverageSources, "+")
}

func calculateMatchdayPerformance(f, previous *matchdayFacts) {
	reach := float64(len(f.Reach))
	active := map[string]map[string]bool{}
	actions, attributedActions := 0, 0
	for _, a := range f.Actions {
		if a.Count <= 0 {
			continue
		}
		actions += a.Count
		if actor := f.Resolve(a.Identity); actor != "" {
			if active[actor] == nil {
				active[actor] = map[string]bool{}
			}
			feature := a.Feature
			if feature == "purchase" || feature == "reservation" {
				feature = "shop"
			}
			active[actor][feature] = true
			attributedActions += a.Count
		}
	}
	multifunction := 0
	for _, features := range active {
		if len(features) >= 2 {
			multifunction++
		}
	}
	audienceSources := []string{"engagement_report_signals", "tracking_events", "page_engagements", "votes", "game rounds", "event_chat_messages", "fan_profiles"}
	actionSources := []string{"votes", "game rounds", "event_feedback", "event_chat_messages", "sponsor_clicks", "sponsor_measured_exposures", "bar_orders", "engagement_report_signals", "tracking_events"}
	const audienceReason = "Copertura delle identità osservate. Browser non associati a un profilo possono rappresentare la stessa persona."
	const actionReason = "Copertura delle azioni tracciate. Gli aggregati esterni e le azioni prive di identità non identificano nuovi tifosi attivi."
	const actionDefinition = "Voti validi, avvii di gioco, feedback, messaggi umani, click sponsor e acquisti o prenotazioni attribuibili."
	const farFormula = "Identità con almeno un’azione significativa / identità raggiunte × 100"
	const reachFormula = "Profili, dispositivi e sessioni anonime distinti, con deduplicazione delle associazioni affidabili"
	const penetrationFormula = "Identità raggiunte / spettatori presenti × 100"
	const interactionsFormula = "Azioni significative attribuibili / tifosi attivi"
	metrics := []matchdayMetric{
		matchdayPartial(matchdayCount("direction_reach", "Utenti unici", "Identità digitali osservate nella partita.", reachFormula, reach, audienceSources...), audienceReason),
		matchdayPartial(matchdayRatio("direction_activation", "Tasso di attivazione", actionDefinition, farFormula, "percent", float64(len(active)), reach, actionSources...), actionReason),
		matchdayPartial(matchdayRatio("direction_far", "Fan Activation Rate (FAR)", actionDefinition, farFormula, "percent", float64(len(active)), reach, actionSources...), actionReason),
		matchdayPartial(matchdayRatio("direction_engagement", "Engagement per tifoso", "Azioni attribuibili ai tifosi attivi.", interactionsFormula, "ratio", float64(attributedActions), float64(len(active)), actionSources...), actionReason),
		matchdayPartial(matchdayCount("mvp_interactions", "Interazioni significative", actionDefinition, "Somma delle azioni significative osservate", float64(actions), actionSources...), actionReason),
		matchdayPartial(matchdayRatio("mvp_interactions_average", "Interazioni per tifoso attivo", "Azioni attribuibili ai tifosi attivi.", interactionsFormula, "ratio", float64(attributedActions), float64(len(active)), actionSources...), actionReason),
		matchdayPartial(matchdayRatio("mvp_multifunction", "Engagement multifunzione", "Identità che hanno usato almeno due funzioni distinte.", "Identità con ≥ 2 funzioni / identità raggiunte × 100", "percent", float64(multifunction), reach, actionSources...), actionReason),
	}
	for _, m := range metrics {
		m.coverage = f.coverage
		if reach == 0 && (m.Code != "mvp_interactions" || actions == 0) {
			m.Value, m.Numerator, m.Denominator, m.Status, m.Reason = nil, nil, nil, "missing", "Nessuna identità osservata per l’evento."
		}
		matchdayPut(f, m)
	}
	penetration := matchdayUnavailable("direction_penetration", "Penetrazione sul pubblico", "Rapporto tra pubblico digitale e spettatori in arena.", penetrationFormula, "percent", "missing", "Servono identità digitali osservate e presenze fisiche ufficiali.", append(append([]string(nil), audienceSources...), "engagement_report_manual_sources")...)
	attendance := f.Manual["attendees"].Value
	if attendance == nil {
		penetration.Missing = []string{"attendees"}
	}
	if attendance != nil && reach > 0 {
		penetration = matchdayPartial(matchdayRatio(penetration.Code, penetration.Name, penetration.Description, penetration.Formula, "percent", reach, *attendance, penetration.Sources...), "Il pubblico digitale può includere utenti remoti. Le presenze provengono dalla fonte indicata dall’amministratore.")
		penetration.Source = "mixed"
		penetration.coverage = f.coverage
	}
	matchdayPut(f, penetration)
	calculateMatchdayRetention(f, previous, audienceSources)
	calculateMatchdayMVP(f)
	calculateMatchdayDuration(f)
	calculateMatchdayRevenue(f)
}

func calculateMatchdayRetention(f, previous *matchdayFacts, sources []string) {
	const formula = "Identità persistenti della partita precedente tornate / identità persistenti della precedente × 100"
	m := matchdayUnavailable("direction_retention", "Retention interpartita", "Cohorte della partita precedente nel perimetro selezionato.", formula, "percent", "missing", "Storico insufficiente o pubblico non confrontabile.", sources...)
	r := matchdayUnavailable("crm_returning", "Tifosi ritornati", "Identità persistenti presenti in entrambe le partite.", "Intersezione delle identità persistenti osservate", "count", "missing", m.Reason, sources...)
	if previous != nil && len(f.Reach) > 0 && len(previous.Reach) > 0 && f.coverage == previous.coverage {
		var total, returned, current int
		for actor := range previous.Reach {
			if matchdayComparableIdentity(actor) {
				total++
				if f.Reach[actor] {
					returned++
				}
			}
		}
		for actor := range f.Reach {
			if matchdayComparableIdentity(actor) {
				current++
			}
		}
		if total > 0 && current > 0 {
			m = matchdayPartial(matchdayRatio(m.Code, m.Name, m.Description, m.Formula, "percent", float64(returned), float64(total), sources...), "Coorte osservata con profili e dispositivi persistenti. Le sessioni anonime sono escluse.")
			r = matchdayPartial(matchdayCount(r.Code, r.Name, r.Description, r.Formula, float64(returned), sources...), m.Reason)
		}
	}
	m.coverage, r.coverage = f.coverage, f.coverage
	crm := m
	crm.Code, crm.Name = "crm_retention", "Retention dei tifosi"
	matchdayPut(f, m, crm, r)
}

func calculateMatchdayMVP(f *matchdayFacts) {
	voters, distribution := map[string]bool{}, map[int]int{}
	names := map[int]string{}
	for _, v := range f.votes {
		if actor := f.Resolve(v.identity); actor != "" {
			voters[actor] = true
		}
		distribution[v.playerID]++
		names[v.playerID] = v.playerName
	}
	rows := make([]matchdayRow, 0, len(distribution))
	for id, n := range distribution {
		name := names[id]
		if name == "" {
			name = fmt.Sprintf("Giocatore #%d", id)
		}
		rows = append(rows, matchdayRow{Label: name, Group: "MVP evento", Values: map[string]*float64{"votes": reportMetricFloat(float64(n)), "share": reportMetricRatio(reportMetricFloat(float64(n)), reportMetricFloat(float64(len(f.votes))), true)}})
	}
	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if *a.Values["votes"] != *b.Values["votes"] {
			return *a.Values["votes"] > *b.Values["votes"]
		}
		return a.Label < b.Label
	})
	conversion := matchdayPartial(matchdayRatio("mvp_conversion", "Conversione MVP", "Votanti distinti sul pubblico digitale osservato.", "Votanti MVP / utenti unici × 100", "percent", float64(len(voters)), float64(len(f.Reach)), "votes", "engagement_report_signals", "tracking_events"), "Copertura del pubblico digitale osservato.")
	conversion.coverage = f.coverage
	count := matchdayCount("mvp_voters", "Votanti unici MVP", "Identità distinte con un voto accettato dal sistema MVP evento.", "Identità distinte in votes", float64(len(voters)), "votes", "fan_profiles")
	if len(voters) == 0 && len(f.votes) > 0 {
		count.Value, count.Status, count.Reason = nil, "missing", "Voti presenti senza identità deduplicabile."
	}
	matchdayPut(f, count, conversion,
		matchdayMetric{Code: "mvp_distribution", Name: "Distribuzione dei voti MVP", Description: "Voti accettati per giocatore nell’evento archiviato.", Formula: "Voti per giocatore e quota sul totale", Unit: "count", Source: "db", Status: "available", Value: reportMetricFloat(float64(len(f.votes))), Sources: []string{"votes", "players"}, Rows: rows, Columns: []matchdayColumn{{"votes", "Voti", "count"}, {"share", "Quota", "percent"}}},
		matchdayUnavailable("mvp_light", "Votazione MVP Light", "MVP semplificato del calendario ClubHub.", "Votanti per partita ClubHub", "count", "not_applicable", "Le partite ClubHub e gli eventi di questo archivio hanno identificativi distinti e nessun collegamento certificato.", "club_mvp_votes", "club_matches"))
}

func calculateMatchdayDuration(f *matchdayFacts) {
	const formula = "Secondi attivi osservati / sessioni attive misurate"
	spans := map[string][]reportMetricInterval{}
	for _, s := range f.Signals {
		if s.Name != "active_time" || s.SessionID == "" {
			continue
		}
		seconds := math.Min(15, reportMetricNumber(s.Metadata, "seconds"))
		if seconds <= 0 || math.IsNaN(seconds) {
			continue
		}
		end := reportMetricTime(reportMetricString(s.Metadata, "ended_at"))
		if end.IsZero() {
			end = reportMetricTime(s.OccurredAt)
		}
		if end.IsZero() {
			continue
		}
		start := reportMetricTime(reportMetricString(s.Metadata, "started_at"))
		limit := end.Add(-time.Duration(seconds * float64(time.Second)))
		if start.IsZero() || start.Before(limit) {
			start = limit
		}
		if start.Before(end) {
			key := s.DeviceID + "\x00" + s.SessionID
			spans[key] = append(spans[key], reportMetricInterval{start, end})
		}
	}
	var seconds float64
	for _, intervals := range spans {
		seconds += reportTimeUnionSeconds(intervals)
	}
	m := matchdayUnavailable("mvp_session_duration", "Durata media delle sessioni attive", "Tempo attivo misurato dalla home tifosi.", formula, "seconds", "not_measurable", "Nessun intervallo di attività misurato.", "engagement_report_signals")
	if len(spans) > 0 {
		m = matchdayPartial(matchdayRatio(m.Code, m.Name, m.Description, m.Formula, m.Unit, seconds, float64(len(spans)), m.Sources...), "Media delle sole sessioni con intervalli attivi misurati.")
	}
	matchdayPut(f, m)
	const legacyFormula = "Secondi registrati / osservazioni di permanenza"
	legacy := matchdayUnavailable("mvp_legacy_duration", "Permanenza media legacy", "Durate inviate dall’interfaccia precedente.", legacyFormula, "seconds", "not_measurable", "Nessuna durata legacy registrata per la partita.", "page_engagements.duration_seconds")
	if f.legacyDurations > 0 {
		legacy = matchdayPartial(matchdayRatio(legacy.Code, legacy.Name, legacy.Description, legacy.Formula, legacy.Unit, float64(f.legacySeconds), float64(f.legacyDurations), legacy.Sources...), "Media delle durate registrate. La fonte legacy è priva di un identificativo di sessione e di intervalli attivi.")
	}
	matchdayPut(f, legacy)
}

func calculateMatchdayCRM(f *matchdayFacts, profiles map[int]matchdayProfile) {
	identified, newProfiles, valid := 0, 0, 0
	local := matchdayRome
	const dateOnly = "2006-01-02"
	day := f.Event.start.In(local).Format(dateOnly)
	for actor := range f.Reach {
		if !strings.HasPrefix(actor, "fan:") {
			continue
		}
		identified++
		id, _ := strconv.Atoi(strings.TrimPrefix(actor, "fan:"))
		p := profiles[id]
		registered := p.accepted
		if registered.IsZero() {
			registered = p.created
		}
		if f.Event.start.IsZero() || registered.IsZero() || registered.In(local).Format(dateOnly) != day {
			continue
		}
		newProfiles++
		if p.verified && p.consent {
			valid++
		}
	}
	sources := []string{"fan_profiles", "fan_registration_acceptances", "engagement_report_signals", "tracking_events", "votes"}
	const attribution = "Profili associati a un’attività della partita e registrati nel giorno locale dell’evento."
	current := matchdayPartial(matchdayCount("crm_new_profiles", "Nuovi profili", attribution, "Profili osservati con prima registrazione in data evento", float64(newProfiles), sources...), attribution)
	fanbase := matchdayPartial(matchdayCount("crm_fanbase", "Fanbase identificata", "Serie dei profili osservati per partita.", "Profili distinti osservati", float64(identified), sources...), "Copertura dei profili identificabili nelle attività dell’evento.")
	profilesMetric := fanbase
	profilesMetric.Code, profilesMetric.Name = "crm_identified", "Profili identificati"
	identification := matchdayPartial(matchdayRatio("crm_identification_rate", "Identification rate", "Quota dei visitatori associata a un profilo.", "Profili identificati / utenti unici × 100", "percent", float64(identified), float64(len(f.Reach)), sources...), fanbase.Reason)
	optIn := matchdayPartial(matchdayRatio("crm_opt_in", "Opt-in rate", "Stato attuale del consenso per i nuovi profili verificati.", "Nuovi profili con consenso e verifica / nuovi profili osservati × 100", "percent", float64(valid), float64(newProfiles), sources...), "Lo stato del profilo non conserva la data del consenso o le sue revisioni.")
	for _, m := range []matchdayMetric{current, fanbase, profilesMetric, identification, optIn} {
		m.coverage = f.coverage
		if len(f.Reach) == 0 {
			m.Value, m.Numerator, m.Denominator, m.Status, m.Reason = nil, nil, nil, "missing", "Nessuna identità osservata."
		} else if f.Event.start.IsZero() && (m.Code == current.Code || m.Code == optIn.Code) {
			m.Value, m.Numerator, m.Denominator, m.Status, m.Reason = nil, nil, nil, "missing", "Data dell’evento non disponibile."
		}
		matchdayPut(f, m)
	}
	matchdayPut(f, matchdayUnavailable("crm_new_consents", "Nuovi consensi marketing", "Nuovi consensi con data e prova attribuibile a un profilo della partita.", "Consensi validi acquisiti durante l’evento", "count", "not_measurable", "Il profilo conserva lo stato corrente del consenso. I consensi dei contatti legacy sono separati dai profili tifosi.", "fan_profiles.marketing_consent", "contacts"))
}

var matchdayRome, _ = time.LoadLocation("Europe/Rome")

func calculateMatchdayRevenue(f *matchdayFacts) {
	m := matchdayUnavailable("direction_revenue", "Ricavi generati", "Ricavi confermati attribuibili alla partita.", "Ricavi confermati e attribuiti da fonti distinte", "eur", "missing", "Non sono disponibili incassi attribuiti all’evento.", "bar_orders", "tracking_events", "matchday_kpi_manual_revisions")
	var total float64
	for _, component := range []struct {
		value  *float64
		source string
	}{
		{f.BarRevenue, "db"}, {f.Manual["shop_gmv"].Value, "manual"}, {f.Manual["offline_revenue"].Value, "manual"},
	} {
		if component.value == nil {
			continue
		}
		total += *component.value
		if m.Source == "" {
			m.Source = component.source
		} else if m.Source != component.source {
			m.Source = "mixed"
		}
	}
	if m.Source != "" {
		m.Value, m.Status, m.Reason = reportMetricFloat(math.Round(total*100)/100), "partial", "Somma degli incassi attribuiti disponibili. La copertura dipende dalla presenza di fonti bar, shop e offline."
		m.coverage = fmt.Sprintf("%t:%t:%t", f.BarRevenue != nil, f.Manual["shop_gmv"].Value != nil, f.Manual["offline_revenue"].Value != nil)
	} else {
		m.Missing = []string{"offline_revenue", "shop_gmv"}
	}
	matchdayPut(f, m)
}
