package api

import (
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

type reportKPI struct {
	ID             string   `json:"id"`
	Label          string   `json:"label"`
	Group          string   `json:"group"`
	Unit           string   `json:"unit"`
	Value          *float64 `json:"value"`
	Source         string   `json:"source"`
	Note           string   `json:"note"`
	Description    string   `json:"description"`
	FactKey        string   `json:"fact_key,omitempty"`
	NumeratorKey   string   `json:"numerator_key,omitempty"`
	DenominatorKey string   `json:"denominator_key,omitempty"`
	Numerator      *float64 `json:"numerator,omitempty"`
	Denominator    *float64 `json:"denominator,omitempty"`
	Corrected      bool     `json:"corrected,omitempty"`
}

type reportGame struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Players         int    `json:"players"`
	RoundsStarted   int    `json:"rounds_started"`
	RoundsCompleted int    `json:"rounds_completed"`
}

type reportSponsor struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Reach       int      `json:"reach"`
	Impressions int      `json:"impressions"`
	Clickers    int      `json:"clickers"`
	CTR         *float64 `json:"ctr"`
}

type reportDetails struct {
	Games    []reportGame    `json:"games"`
	Sponsors []reportSponsor `json:"sponsors"`
}

var engagementReportFactKeys = []string{
	"visitors", "onsite_visitors", "attendees", "active_visitors", "active_seconds",
	"voters", "vote_eligible_visitors", "players", "game_eligible_visitors",
	"rounds_completed", "repeat_players", "multi_experience_visitors", "new_profiles",
	"registration_starters", "returning_profiles", "previous_event_profiles",
	"redeemers", "shop_eligible_profiles", "sponsor_reach", "sponsor_impressions",
	"sponsor_clickers", "sponsored_players", "sponsored_exposed_visitors",
	"feedback_respondents", "feedback_eligible_visitors", "experience_positive",
	"experience_responses", "perks_positive", "perks_responses",
}

type reportVisitorSet map[string]bool

func reportMetricFloat(value float64) *float64 { return &value }

func reportMetricString(meta map[string]interface{}, key string) string {
	value, _ := meta[key].(string)
	return strings.TrimSpace(value)
}

func reportMetricNumber(meta map[string]interface{}, key string) float64 {
	switch value := meta[key].(type) {
	case float64:
		return value
	case int:
		return float64(value)
	case int64:
		return float64(value)
	case json.Number:
		number, _ := value.Float64()
		return number
	}
	return 0
}

func reportMetricTime(value string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, strings.TrimSpace(value)); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func reportMetricRatio(numerator, denominator *float64, percent bool) *float64 {
	if numerator == nil || denominator == nil || *denominator <= 0 {
		return nil
	}
	value := *numerator / *denominator
	if percent {
		value *= 100
	}
	return reportMetricFloat(value)
}

func reportMetricIntersection(a, b reportVisitorSet) reportVisitorSet {
	result := reportVisitorSet{}
	for id := range a {
		if b[id] {
			result[id] = true
		}
	}
	return result
}

func reportMetricActive(name string) bool {
	switch name {
	case "vote_completed", "game_started", "game_completed", "redemption_completed", "sponsor_click", "feedback_completed":
		return true
	}
	return false
}

type reportMetricInterval struct{ start, end time.Time }
type reportMetricGameState struct {
	name      string
	players   reportVisitorSet
	started   reportVisitorSet
	completed reportVisitorSet
}
type reportMetricSponsorState struct {
	name        string
	reach       reportVisitorSet
	clickers    reportVisitorSet
	impressions reportVisitorSet
	sponsored   reportVisitorSet
	players     reportVisitorSet
}

// calculateReportMetrics reads only signals emitted by /new. Each successful
// action also proves a visit; it never manufactures an eligible audience when
// the corresponding availability flag was not observed.
func calculateReportMetrics(signals []reportSignal, previous []reportSignal) (map[string]*float64, reportDetails, string) {
	facts := map[string]*float64{}
	for _, key := range engagementReportFactKeys {
		facts[key] = nil
	}
	details := reportDetails{Games: []reportGame{}, Sponsors: []reportSponsor{}}
	if len(signals) == 0 {
		return facts, details, ""
	}
	// A shared device is linked only when exactly one identified profile was
	// seen on it during this event. No cross-event anonymous matching occurs.
	devices := map[string]map[int]bool{}
	for _, signal := range signals {
		if signal.FanID > 0 && signal.DeviceID != "" {
			if devices[signal.DeviceID] == nil {
				devices[signal.DeviceID] = map[int]bool{}
			}
			devices[signal.DeviceID][signal.FanID] = true
		}
	}
	identity := func(signal reportSignal) string {
		if signal.FanID > 0 {
			return "fan:" + strconv.Itoa(signal.FanID)
		}
		if len(devices[signal.DeviceID]) == 1 {
			for fanID := range devices[signal.DeviceID] {
				return "fan:" + strconv.Itoa(fanID)
			}
		}
		if signal.DeviceID != "" {
			return "device:" + signal.DeviceID
		}
		if signal.SessionID != "" {
			return "session:" + signal.SessionID
		}
		return ""
	}
	ordered := append([]reportSignal{}, signals...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return reportMetricTime(ordered[i].OccurredAt).Before(reportMetricTime(ordered[j].OccurredAt))
	})
	visitors, onsite, active := reportVisitorSet{}, reportVisitorSet{}, reportVisitorSet{}
	voters, voteEligible := reportVisitorSet{}, reportVisitorSet{}
	players, gameEligible := reportVisitorSet{}, reportVisitorSet{}
	profiles, registrationStarters := reportVisitorSet{}, reportVisitorSet{}
	redeemers, shopEligible := reportVisitorSet{}, reportVisitorSet{}
	respondents, feedbackEligible := reportVisitorSet{}, reportVisitorSet{}
	experienceResponses, experiencePositive := reportVisitorSet{}, reportVisitorSet{}
	perksResponses, perksPositive := reportVisitorSet{}, reportVisitorSet{}
	features := map[string]reportVisitorSet{}
	startedRounds, completedRounds := reportVisitorSet{}, reportVisitorSet{}
	roundsByPlayer := map[string]reportVisitorSet{}
	games := map[string]*reportMetricGameState{}
	sponsors := map[int]*reportMetricSponsorState{}
	intervals := map[string][]reportMetricInterval{}
	voteFlags, gameFlags, feedbackFlags, arenaSource := false, false, false, false
	seenIDs := reportVisitorSet{}
	var earliest time.Time
	for _, signal := range ordered {
		if signal.ID != "" {
			if seenIDs[signal.ID] {
				continue
			}
			seenIDs[signal.ID] = true
		}
		visitor := identity(signal)
		if visitor == "" {
			continue
		}
		occurred := reportMetricTime(signal.OccurredAt)
		if !occurred.IsZero() && (earliest.IsZero() || occurred.Before(earliest)) {
			earliest = occurred
		}
		visitors[visitor] = true
		if reportMetricActive(signal.Name) {
			active[visitor] = true
			if features[visitor] == nil {
				features[visitor] = reportVisitorSet{}
			}
			feature := strings.TrimSuffix(signal.Name, "_completed")
			if signal.Name == "game_started" {
				feature = "game"
			}
			features[visitor][feature] = true
		}
		switch signal.Name {
		case "page_view":
			if reportMetricString(signal.Metadata, "source") == "arena" {
				arenaSource = true
				onsite[visitor] = true
			}
			if available, exists := signal.Metadata["vote_available"].(bool); exists {
				voteFlags = true
				if available {
					voteEligible[visitor] = true
				}
			}
			if available, exists := signal.Metadata["games_available"].(bool); exists {
				gameFlags = true
				if available {
					gameEligible[visitor] = true
				}
			}
			if available, exists := signal.Metadata["feedback_available"].(bool); exists {
				feedbackFlags = true
				if available {
					feedbackEligible[visitor] = true
				}
			}
			if signal.FanID > 0 {
				shopEligible[visitor] = true
			}
		case "active_time":
			seconds := reportMetricNumber(signal.Metadata, "seconds")
			if math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds <= 0 {
				continue
			}
			seconds = math.Min(seconds, 15)
			end := reportMetricTime(reportMetricString(signal.Metadata, "ended_at"))
			if end.IsZero() {
				end = occurred
			}
			if end.IsZero() {
				continue
			}
			start := reportMetricTime(reportMetricString(signal.Metadata, "started_at"))
			limit := end.Add(-time.Duration(seconds * float64(time.Second)))
			if start.IsZero() || start.Before(limit) {
				start = limit
			}
			if start.Before(end) {
				intervals[visitor] = append(intervals[visitor], reportMetricInterval{start, end})
			}
		case "vote_completed":
			voters[visitor] = true
		case "game_started", "game_completed":
			gameID := reportMetricString(signal.Metadata, "game_id")
			if gameID == "" {
				continue
			}
			players[visitor] = true
			if games[gameID] == nil {
				games[gameID] = &reportMetricGameState{name: gameID, players: reportVisitorSet{}, started: reportVisitorSet{}, completed: reportVisitorSet{}}
			}
			game := games[gameID]
			if name := reportMetricString(signal.Metadata, "name"); name != "" {
				game.name = name
			}
			game.players[visitor] = true
			if roundID := reportMetricString(signal.Metadata, "round_id"); roundID != "" {
				round := visitor + "\x00" + gameID + "\x00" + roundID
				startedRounds[round], game.started[round] = true, true
				if roundsByPlayer[visitor] == nil {
					roundsByPlayer[visitor] = reportVisitorSet{}
				}
				roundsByPlayer[visitor][round] = true
				if signal.Name == "game_completed" {
					completedRounds[round], game.completed[round] = true, true
				}
			}
		case "registration_started":
			registrationStarters[visitor] = true
		case "registration_completed":
			if signal.FanID > 0 {
				profiles[visitor], registrationStarters[visitor] = true, true
			}
		case "shop_view":
			if signal.FanID > 0 {
				shopEligible[visitor] = true
			}
		case "redemption_completed":
			if signal.FanID > 0 {
				redeemers[visitor], shopEligible[visitor] = true, true
			}
		case "feedback_completed":
			if respondents[visitor] {
				continue
			}
			respondents[visitor] = true
			switch reportMetricString(signal.Metadata, "experience") {
			case "awesome", "good":
				experienceResponses[visitor], experiencePositive[visitor] = true, true
			case "ok", "bad":
				experienceResponses[visitor] = true
			}
			switch reportMetricString(signal.Metadata, "perks_interest") {
			case "yes":
				perksResponses[visitor], perksPositive[visitor] = true, true
			case "maybe", "little", "no":
				perksResponses[visitor] = true
			}
		}
		sponsorID := int(reportMetricNumber(signal.Metadata, "sponsor_id"))
		if sponsorID > 0 && (signal.Name == "sponsor_view" || signal.Name == "sponsored_game_view" || signal.Name == "sponsor_click" || signal.Name == "game_started" || signal.Name == "game_completed") {
			if sponsors[sponsorID] == nil {
				sponsors[sponsorID] = &reportMetricSponsorState{name: "Sponsor " + strconv.Itoa(sponsorID), reach: reportVisitorSet{}, clickers: reportVisitorSet{}, impressions: reportVisitorSet{}, sponsored: reportVisitorSet{}, players: reportVisitorSet{}}
			}
			sponsor := sponsors[sponsorID]
			if name := reportMetricString(signal.Metadata, "name"); name != "" && signal.Name != "game_started" && signal.Name != "game_completed" {
				sponsor.name = name
			}
			switch signal.Name {
			case "sponsor_view", "sponsored_game_view":
				sponsor.reach[visitor] = true
				impression := visitor + "\x00" + signal.SessionID + "\x00" + signal.Name + "\x00" + reportMetricString(signal.Metadata, "placement")
				sponsor.impressions[impression] = true
				if signal.Name == "sponsored_game_view" {
					sponsor.sponsored[visitor] = true
				}
			case "sponsor_click":
				sponsor.clickers[visitor] = true
			case "game_started", "game_completed":
				sponsor.players[visitor] = true
			}
		}
	}
	if len(visitors) == 0 {
		return facts, details, ""
	}
	for _, key := range engagementReportFactKeys {
		facts[key] = reportMetricFloat(0)
	}
	facts["attendees"], facts["onsite_visitors"] = nil, nil
	count := func(key string, set reportVisitorSet) { facts[key] = reportMetricFloat(float64(len(set))) }
	count("visitors", visitors)
	count("active_visitors", active)
	count("voters", voters)
	count("players", players)
	count("rounds_completed", completedRounds)
	count("new_profiles", profiles)
	count("registration_starters", registrationStarters)
	count("redeemers", redeemers)
	count("shop_eligible_profiles", shopEligible)
	count("feedback_respondents", respondents)
	count("experience_responses", experienceResponses)
	count("experience_positive", experiencePositive)
	count("perks_responses", perksResponses)
	count("perks_positive", perksPositive)
	if arenaSource {
		count("onsite_visitors", onsite)
	}
	for _, spec := range []struct {
		key      string
		observed bool
		eligible reportVisitorSet
		actions  reportVisitorSet
	}{{"vote_eligible_visitors", voteFlags, voteEligible, voters}, {"game_eligible_visitors", gameFlags, gameEligible, players}, {"feedback_eligible_visitors", feedbackFlags, feedbackEligible, respondents}} {
		facts[spec.key] = nil
		if spec.observed {
			for visitor := range spec.actions {
				spec.eligible[visitor] = true
			}
			count(spec.key, spec.eligible)
		}
	}
	for visitor, rounds := range roundsByPlayer {
		if len(rounds) >= 2 && players[visitor] {
			*facts["repeat_players"]++
		}
	}
	for _, used := range features {
		if len(used) >= 2 {
			*facts["multi_experience_visitors"]++
		}
	}
	var activeSeconds float64
	for _, spans := range intervals {
		sort.Slice(spans, func(i, j int) bool { return spans[i].start.Before(spans[j].start) })
		merged := spans[0]
		for _, span := range spans[1:] {
			if !span.start.After(merged.end) {
				if span.end.After(merged.end) {
					merged.end = span.end
				}
			} else {
				activeSeconds += merged.end.Sub(merged.start).Seconds()
				merged = span
			}
		}
		activeSeconds += merged.end.Sub(merged.start).Seconds()
	}
	facts["active_seconds"] = reportMetricFloat(math.Round(activeSeconds))
	previousProfiles, returning := reportVisitorSet{}, reportVisitorSet{}
	previousDevices := map[string]map[int]bool{}
	for _, signal := range previous {
		if signal.FanID > 0 && signal.DeviceID != "" {
			if previousDevices[signal.DeviceID] == nil {
				previousDevices[signal.DeviceID] = map[int]bool{}
			}
			previousDevices[signal.DeviceID][signal.FanID] = true
		}
	}
	for _, signal := range previous {
		fanID := signal.FanID
		if fanID == 0 && signal.DeviceID != "" && len(previousDevices[signal.DeviceID]) == 1 {
			for resolved := range previousDevices[signal.DeviceID] {
				fanID = resolved
			}
		}
		if fanID > 0 && reportMetricActive(signal.Name) {
			previousProfiles["fan:"+strconv.Itoa(fanID)] = true
		}
	}
	facts["previous_event_profiles"], facts["returning_profiles"] = nil, nil
	if len(previous) > 0 {
		count("previous_event_profiles", previousProfiles)
		for profile := range previousProfiles {
			if active[profile] {
				returning[profile] = true
			}
		}
		count("returning_profiles", returning)
	}
	for id, game := range games {
		details.Games = append(details.Games, reportGame{ID: id, Name: game.name, Players: len(game.players), RoundsStarted: len(game.started), RoundsCompleted: len(game.completed)})
	}
	sort.Slice(details.Games, func(i, j int) bool { return details.Games[i].ID < details.Games[j].ID })
	allReach, allClickers, allSponsored, allSponsoredPlayers := reportVisitorSet{}, reportVisitorSet{}, reportVisitorSet{}, reportVisitorSet{}
	impressions := 0
	for id, sponsor := range sponsors {
		clickers := reportMetricIntersection(sponsor.clickers, sponsor.reach)
		sponsoredPlayers := reportMetricIntersection(sponsor.players, sponsor.sponsored)
		for visitor := range sponsor.reach {
			allReach[visitor] = true
		}
		for visitor := range clickers {
			allClickers[visitor] = true
		}
		for visitor := range sponsor.sponsored {
			allSponsored[visitor] = true
		}
		for visitor := range sponsoredPlayers {
			allSponsoredPlayers[visitor] = true
		}
		impressions += len(sponsor.impressions)
		details.Sponsors = append(details.Sponsors, reportSponsor{ID: id, Name: sponsor.name, Reach: len(sponsor.reach), Impressions: len(sponsor.impressions), Clickers: len(clickers), CTR: reportMetricRatio(reportMetricFloat(float64(len(clickers))), reportMetricFloat(float64(len(sponsor.reach))), true)})
	}
	sort.Slice(details.Sponsors, func(i, j int) bool { return details.Sponsors[i].ID < details.Sponsors[j].ID })
	count("sponsor_reach", allReach)
	count("sponsor_clickers", allClickers)
	count("sponsored_exposed_visitors", allSponsored)
	count("sponsored_players", allSponsoredPlayers)
	facts["sponsor_impressions"] = reportMetricFloat(float64(impressions))
	startedAt := ""
	if !earliest.IsZero() {
		startedAt = earliest.UTC().Format(time.RFC3339)
	}
	return facts, details, startedAt
}

func buildReportKPIs(facts map[string]*float64) []reportKPI {
	definitions := []reportKPI{
		{ID: "visitors", Label: "Visitatori digitali unici", Group: "Pubblico", Unit: "count", FactKey: "visitors", Description: "Identità osservate in /new: profilo verificato quando disponibile, altrimenti dispositivo. Un dispositivo condiviso resta distinto se il collegamento al profilo è ambiguo."},
		{ID: "venue_adoption", Label: "Adozione del pubblico presente", Group: "Pubblico", Unit: "percent", NumeratorKey: "onsite_visitors", DenominatorKey: "attendees", Description: "Visitatori con provenienza arena esplicita divisi per le presenze effettive inserite dalla società. La provenienza non dimostra da sola la presenza fisica."},
		{ID: "active_participation", Label: "Partecipazione attiva", Group: "Engagement", Unit: "percent", NumeratorKey: "active_visitors", DenominatorKey: "visitors", Description: "Visitatori con voto, avvio o completamento gioco, riscatto, clic sponsor o feedback, rispetto ai visitatori osservati."},
		{ID: "active_time", Label: "Tempo attivo medio", Group: "Engagement", Unit: "seconds", NumeratorKey: "active_seconds", DenominatorKey: "visitors", Description: "Secondi di attività dichiarata in primo piano per visitatore. Intervalli sovrapposti dello stesso utente vengono uniti; ogni heartbeat vale al massimo 15 secondi."},
		{ID: "mvp_participation", Label: "Partecipazione al voto MVP", Group: "Engagement", Unit: "percent", NumeratorKey: "voters", DenominatorKey: "vote_eligible_visitors", Description: "Votanti unici rispetto ai visitatori per cui la votazione risultava disponibile. I cambi di voto non aggiungono un votante."},
		{ID: "game_participation", Label: "Partecipazione ai giochi", Group: "Gaming", Unit: "percent", NumeratorKey: "players", DenominatorKey: "game_eligible_visitors", Description: "Giocatori unici con almeno un avvio o completamento rispetto ai visitatori con giochi disponibili."},
		{ID: "completed_rounds", Label: "Round di gioco completati", Group: "Gaming", Unit: "count", FactKey: "rounds_completed", Description: "Round distinti per utente, gioco e identificativo del round. La dichiarazione di completamento non certifica ogni gesto di gioco."},
		{ID: "replay_rate", Label: "Tasso di rigioco", Group: "Gaming", Unit: "percent", NumeratorKey: "repeat_players", DenominatorKey: "players", Description: "Giocatori con almeno due round distinti avviati, anche in giochi diversi, rispetto ai giocatori. Un completamento recupera un avvio non ricevuto."},
		{ID: "multi_experience", Label: "Uso di più esperienze", Group: "Engagement", Unit: "percent", NumeratorKey: "multi_experience_visitors", DenominatorKey: "active_visitors", Description: "Partecipanti attivi che usano almeno due funzionalità diverse tra voto, gaming, shop, sponsor e feedback."},
		{ID: "new_profiles", Label: "Nuovi profili verificati", Group: "Acquisizione", Unit: "count", FactKey: "new_profiles", Description: "Profili identificati con completamento della prima registrazione attribuito all'evento; accessi di profili esistenti esclusi."},
		{ID: "registration_conversion", Label: "Conversione alla registrazione", Group: "Acquisizione", Unit: "percent", NumeratorKey: "new_profiles", DenominatorKey: "registration_starters", Description: "Nuovi profili verificati rispetto agli utenti che iniziano la registrazione. Un completamento recupera il relativo avvio eventualmente mancante."},
		{ID: "event_retention", Label: "Ritorno dall'evento precedente", Group: "Fidelizzazione", Unit: "percent", NumeratorKey: "returning_profiles", DenominatorKey: "previous_event_profiles", Description: "Profili attivi nell'evento precedente che partecipano attivamente anche a questo evento. Confronto limitato a profili identificati; nessun collegamento tra ospiti di eventi diversi."},
		{ID: "redemption_conversion", Label: "Conversione al riscatto", Group: "Fidelizzazione", Unit: "percent", NumeratorKey: "redeemers", DenominatorKey: "shop_eligible_profiles", Description: "Profili con almeno un riscatto completato rispetto ai profili registrati che accedono alla home o allo shop. Gli oggetti sono riscattati in monete virtuali, non in euro."},
		{ID: "sponsor_reach", Label: "Copertura unica sponsor", Group: "Sponsor", Unit: "count", FactKey: "sponsor_reach", Description: "Visitatori unici con una visualizzazione sponsor qualificata; utenti esposti a più sponsor conteggiati una sola volta nel totale."},
		{ID: "sponsor_impressions", Label: "Impression sponsor visibili", Group: "Sponsor", Unit: "count", FactKey: "sponsor_impressions", Description: "Visualizzazioni qualificate distinte per visitatore, sessione, sponsor e posizionamento. Ripetizioni dello stesso segnale non aumentano il conteggio."},
		{ID: "sponsor_ctr", Label: "CTR sponsor su utenti esposti", Group: "Sponsor", Unit: "percent", NumeratorKey: "sponsor_clickers", DenominatorKey: "sponsor_reach", Description: "Utenti con clic verso uno sponsor effettivamente visto rispetto alla copertura sponsor unica. I clic senza una corrispondente esposizione non entrano nel numeratore."},
		{ID: "sponsored_participation", Label: "Partecipazione ai giochi sponsorizzati", Group: "Sponsor", Unit: "percent", NumeratorKey: "sponsored_players", DenominatorKey: "sponsored_exposed_visitors", Description: "Visitatori che avviano o completano il gioco dello stesso sponsor a cui sono stati esposti, rispetto agli esposti a giochi sponsorizzati."},
		{ID: "feedback_response", Label: "Risposta al feedback", Group: "Feedback", Unit: "percent", NumeratorKey: "feedback_respondents", DenominatorKey: "feedback_eligible_visitors", Description: "Rispondenti unici rispetto ai visitatori con sondaggio disponibile. Per ogni identità viene considerata la prima risposta dell'evento."},
		{ID: "satisfaction", Label: "Esperienza positiva", Group: "Feedback", Unit: "percent", NumeratorKey: "experience_positive", DenominatorKey: "experience_responses", Description: "Risposte Fantastica o Bella rispetto alle risposte valide alla domanda sull'esperienza. Il risultato descrive soltanto il campione rispondente."},
		{ID: "perks_interest", Label: "Interesse per premi e vantaggi", Group: "Feedback", Unit: "percent", NumeratorKey: "perks_positive", DenominatorKey: "perks_responses", Description: "Risposte Sì molto rispetto alle risposte valide alla domanda su premi e vantaggi; Forse non è conteggiato come interesse positivo."},
	}
	for index := range definitions {
		kpi := &definitions[index]
		kpi.Source = "new_tracking"
		kpi.Note = "Segnali attribuiti a /new; gli ospiti sono identificati per dispositivo."
		if kpi.FactKey != "" {
			kpi.Value = facts[kpi.FactKey]
		} else {
			kpi.Numerator, kpi.Denominator = facts[kpi.NumeratorKey], facts[kpi.DenominatorKey]
			kpi.Value = reportMetricRatio(kpi.Numerator, kpi.Denominator, kpi.Unit == "percent")
		}
		if kpi.Value == nil {
			kpi.Source = "unavailable"
			kpi.Note = "Dati non disponibili o denominatore assente/nullo. Nessuna stima applicata."
		}
		if kpi.ID == "venue_adoption" && kpi.Value != nil {
			kpi.Source = "manual"
			kpi.Note = "Presenze confermate manualmente dalla società; provenienza arena rilevata dalla piattaforma."
		}
	}
	return definitions
}
