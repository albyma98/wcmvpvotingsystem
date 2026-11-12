package api

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
)

const (
	matchdayFeedbackDB     = "event_feedback.response_id, experience, team_spirit, perks_interest, mini_games_interest, report_experience, suggestion"
	matchdayFeedbackReport = "engagement_report_signals.name, fan_id, device_key, session_id, metadata"
	matchdayFeedbackLegacy = "tracking_events.event_name, fan_id, device_id, session_id"
)

type matchdayFeedbackData struct {
	enabled                                 bool
	config                                  database.EventFeedbackSurveyConfig
	records, answers, complete, suggestions int
	experiencePositive, experienceAnswers   int
	distribution                            map[string]map[string]int
	themes                                  map[string]int
}

func loadMatchdayFeedback(ctx context.Context, q matchdayQueryer, orgID int, data map[int]*matchdayFacts) error {
	in, ids := matchdayIDs(data)
	rows, err := q.QueryContext(ctx, `
		SELECT e.id, e.show_feedback_survey, COALESCE(e.feedback_survey_config,''),
			COALESCE(f.id,0), COALESCE(f.experience,''), COALESCE(f.team_spirit,''),
			COALESCE(f.perks_interest,''), COALESCE(f.mini_games_interest,''),
			COALESCE(f.report_experience,''), COALESCE(f.suggestion,'')
		FROM events e LEFT JOIN event_feedback f ON f.event_id=e.id
		WHERE e.organization_id=? AND e.id IN (`+in+`)`, append([]interface{}{orgID}, ids...)...)
	if err != nil {
		return fmt.Errorf("matchday feedback: %w", err)
	}
	defer rows.Close()
	all := map[int]*matchdayFeedbackData{}
	for rows.Next() {
		var eventID, rowID int
		var enabled bool
		var rawConfig, experience, spirit, perks, games, rating, suggestion string
		if err := rows.Scan(&eventID, &enabled, &rawConfig, &rowID, &experience, &spirit, &perks, &games, &rating, &suggestion); err != nil {
			return err
		}
		item := all[eventID]
		if item == nil {
			var config *database.EventFeedbackSurveyConfig
			if rawConfig != "" {
				_ = json.Unmarshal([]byte(rawConfig), &config)
			}
			item = &matchdayFeedbackData{enabled: enabled, config: database.NormalizeEventFeedbackSurveyConfig(config), distribution: map[string]map[string]int{}, themes: map[string]int{}}
			all[eventID] = item
		}
		if rowID > 0 {
			item.add(experience, spirit, perks, games, rating, suggestion)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for id, item := range all {
		item.apply(data[id])
	}
	return nil
}

func (d *matchdayFeedbackData) add(experience, spirit, perks, games, rating, suggestion string) {
	answered := 0
	for _, item := range []struct{ question, value string }{
		{"experience", experience}, {"team_spirit", spirit},
		{"perks_interest", perks}, {"mini_games_interest", games},
	} {
		if validFeedbackAnswer(item.question, item.value) {
			d.answer(item.question, item.value)
			answered++
		}
	}
	if answered == 4 {
		d.complete++
	}
	if validFeedbackAnswer("report_experience", rating) {
		d.answer("report_experience", rating)
		d.experienceAnswers++
		if rating == "awesome" || rating == "good" {
			d.experiencePositive++
		}
		answered++
	}
	if strings.TrimSpace(suggestion) != "" {
		d.suggestions++
		answered++
		matchdayFeedbackThemes(suggestion, d.themes)
	}
	if answered > 0 {
		d.records++
		d.answers += answered
	}
}

func (d *matchdayFeedbackData) answer(question, value string) {
	if d.distribution[question] == nil {
		d.distribution[question] = map[string]int{}
	}
	d.distribution[question][value]++
}

func matchdayFeedbackThemes(text string, out map[string]int) {
	words := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) })
	for _, theme := range []struct {
		label    string
		prefixes []string
	}{
		{"Voto MVP", []string{"vot", "mvp"}},
		{"Giochi e quiz", []string{"gioc", "quiz", "sfid"}},
		{"Premi e vantaggi", []string{"premi", "vantagg", "coupon", "scont"}},
		{"Accesso e connessione", []string{"access", "conness", "wifi", "caricam", "errore", "bloccat"}},
		{"Chat e community", []string{"chat", "messagg", "community"}},
		{"Shop e servizi", []string{"shop", "merch", "bar", "ordin", "prenot"}},
	} {
		if matchdayThemeMentioned(words, theme.prefixes) {
			out[theme.label]++
		}
	}
}

func matchdayThemeMentioned(words, prefixes []string) bool {
	for _, word := range words {
		for _, prefix := range prefixes {
			if strings.HasPrefix(word, prefix) {
				return true
			}
		}
	}
	return false
}

type matchdayFeedbackCohort struct {
	respondents map[string]matchdayIdentity
	exposed     map[string]bool
	opened      map[string]bool
	completed   map[string]bool
}

func matchdayFeedbackTelemetry(f *matchdayFacts) matchdayFeedbackCohort {
	c := matchdayFeedbackCohort{
		respondents: map[string]matchdayIdentity{}, exposed: map[string]bool{},
		opened: map[string]bool{}, completed: map[string]bool{},
	}
	add := func(flow, name string, identity matchdayIdentity) {
		person := f.Resolve(identity)
		if person == "" {
			return
		}
		session := identity.Session
		if session == "" {
			session = person
		}
		session = flow + ":" + session
		switch name {
		case "exposed":
			c.exposed[person] = true
		case "opened":
			c.opened[session] = true
		case "completed":
			c.completed[session] = true
			c.respondents[person] = identity
		case "answered":
			c.respondents[person] = identity
		}
	}
	for _, signal := range f.Signals {
		identity := matchdayReportIdentity(signal)
		switch signal.Name {
		case "cta_viewed":
			if reportMetricString(signal.Metadata, "cta") == "feedback" && reportMetricString(signal.Metadata, "placement") == "feedback_home" {
				add("report", "exposed", identity)
			}
		case "feedback_opened":
			add("report", "opened", identity)
		case "feedback_completed":
			add("report", "completed", identity)
		case "feedback_step":
			if reportMetricString(signal.Metadata, "trigger") == "feedback" && reportMetricString(signal.Metadata, "outcome") == "selected" {
				add("report", "answered", identity)
			}
		}
	}
	for _, row := range f.Tracking {
		identity := matchdayTrackingIdentity(row)
		switch row.Name {
		case "feedback.modal_opened":
			add("tracking", "exposed", identity)
			add("tracking", "opened", identity)
		case "feedback.submitted":
			add("tracking", "completed", identity)
		}
	}
	return c
}

func (d *matchdayFeedbackData) apply(f *matchdayFacts) {
	c := matchdayFeedbackTelemetry(f)
	const responseFormula = "Rispondenti osservati e associabili a un’esposizione / utenti con esposizione misurata × 100."
	const flowFormula = "Sessioni con invio e avvio osservati / sessioni con avvio osservato × 100."
	const responseScope = "Il tasso copre le esposizioni osservate nella home e le aperture del questionario precedente. Le risposte anonime senza collegamento restano fuori dal tasso."
	const flowScope = "Il tasso copre gli avvii osservati e i relativi completamenti nella stessa sessione."
	const peopleScope = "Rispondenti associabili a un profilo, dispositivo o sessione. Le risposte salvate senza identità possono riguardare altri tifosi."
	telemetry := []string{matchdayFeedbackReport, matchdayFeedbackLegacy}
	response := matchdayUnavailable("feedback_response_rate", "Tasso di risposta", "Percentuale del pubblico esposto che ha risposto.", responseFormula, "percent", "not_measurable", "Nessuna esposizione al questionario misurata.", telemetry...)
	respondents := matchdayUnavailable("feedback_respondents", "Rispondenti unici", "Identità distinte che hanno risposto.", "Deduplicazione dei rispondenti osservabili.", "count", "not_measurable", "Le risposte salvate non includono un identificativo del tifoso.", telemetry...)
	starts := matchdayUnavailable("feedback_started", "Sondaggi iniziati", "Aperture o primi tentativi di risposta osservati.", "Sessioni distinte con apertura o primo tentativo.", "count", "not_measurable", "Nessun avvio di questionario osservato.", telemetry...)
	completionRate := matchdayUnavailable("feedback_completion_rate", "Completamento sondaggi", "Percentuale delle sessioni avviate che hanno completato il questionario.", flowFormula, "percent", "not_measurable", "Nessun avvio di questionario osservato.", telemetry...)
	satisfaction := matchdayUnavailable("feedback_satisfaction", "Tasso di soddisfazione", "Risposte positive alla domanda dedicata all’esperienza.", "Risposte awesome o good / risposte valide alla domanda dedicata × 100.", "percent", "not_measurable", "Nessuna risposta su una scala di soddisfazione riconosciuta.", matchdayFeedbackDB)
	if len(c.exposed) > 0 {
		matched := 0
		for person := range c.respondents {
			if c.exposed[person] {
				matched++
			}
		}
		response = matchdayDomainPartial(matchdayRatio(response.Code, response.Name, response.Description, responseFormula, "percent", float64(matched), float64(len(c.exposed)), telemetry...), responseScope)
	}
	if len(c.respondents) > 0 || (len(c.exposed) > 0 && d.records == 0) {
		respondents = matchdayDomainPartial(matchdayCount(respondents.Code, respondents.Name, respondents.Description, respondents.Formula, float64(len(c.respondents)), telemetry...), peopleScope)
	}
	if len(c.opened) > 0 {
		matched := 0
		for session := range c.completed {
			if c.opened[session] {
				matched++
			}
		}
		starts = matchdayDomainPartial(matchdayCount(starts.Code, starts.Name, starts.Description, starts.Formula, float64(len(c.opened)), telemetry...), flowScope)
		completionRate = matchdayDomainPartial(matchdayRatio(completionRate.Code, completionRate.Name, completionRate.Description, flowFormula, "percent", float64(matched), float64(len(c.opened)), telemetry...), flowScope)
	}
	if d.experienceAnswers > 0 {
		satisfaction = matchdayRatio(satisfaction.Code, satisfaction.Name, satisfaction.Description, satisfaction.Formula, "percent", float64(d.experiencePositive), float64(d.experienceAnswers), matchdayFeedbackDB)
	}
	records := matchdayCount("feedback_responses", "Feedback raccolti", "Questionari con almeno una risposta salvata, inclusi quelli parziali.", "Numero di schede di risposta salvate con almeno un campo valido.", float64(d.records), matchdayFeedbackDB)
	answers := matchdayCount("feedback_answers", "Risposte totali raccolte", "Tutte le risposte valide salvate nelle schede.", "Somma delle risposte valide per domanda, incluso il suggerimento facoltativo.", float64(d.answers), matchdayFeedbackDB)
	distribution := matchdayCount("feedback_distribution", "Distribuzione per domanda", "Risposte per opzione e per domanda del questionario.", "Conteggio per opzione della domanda.", float64(d.answers-d.suggestions), matchdayFeedbackDB, "events.feedback_survey_config")
	distribution.Columns = []matchdayColumn{{ID: "count", Label: "Risposte", Unit: "count"}, {ID: "share", Label: "Quota per domanda", Unit: "percent"}}
	questions := append(d.config.Questions, database.EventFeedbackQuestionConfig{
		ID: "report_experience", Title: "Esperienza complessiva",
		Answers: []database.EventFeedbackAnswerConfig{
			{Value: "awesome", Label: "Fantastica"}, {Value: "good", Label: "Bella"},
			{Value: "ok", Label: "Discreta"}, {Value: "bad", Label: "Negativa"},
		},
	})
	for _, question := range questions {
		counts := d.distribution[question.ID]
		total := 0
		for _, count := range counts {
			total += count
		}
		if total == 0 {
			continue
		}
		for _, answer := range question.Answers {
			distribution.Rows = append(distribution.Rows, matchdayRow{Label: answer.Label, Group: question.Title, Values: map[string]*float64{
				"count": matchdayDomainValue(float64(counts[answer.Value])), "share": matchdayDomainRatio(counts[answer.Value], total, 100),
			}})
		}
	}
	completed := matchdayCount("feedback_completed", "Sondaggi completati", "Schede con tutte le risposte obbligatorie salvate.", "Schede con risposta valida a tutte e quattro le domande obbligatorie.", float64(d.complete), matchdayFeedbackDB)
	themes := matchdayCount("feedback_themes", "Temi nei suggerimenti", "Suggerimenti testuali salvati. Il dettaglio mostra le categorie menzionate da almeno due schede.", "Conteggio schede con suggerimenti e aggregazione per parole chiave.", float64(d.suggestions), matchdayFeedbackDB)
	themes.Columns = []matchdayColumn{{ID: "count", Label: "Menzioni", Unit: "count"}}
	for label, count := range d.themes {
		if count >= 2 {
			themes.Rows = append(themes.Rows, matchdayRow{Label: label, Values: map[string]*float64{"count": matchdayDomainValue(float64(count))}})
		}
	}
	sort.Slice(themes.Rows, func(i, j int) bool { return themes.Rows[i].Label < themes.Rows[j].Label })
	metrics := []matchdayMetric{records, answers, respondents, response, distribution, satisfaction, starts, completed, completionRate, themes}
	if !d.enabled && d.records == 0 && len(c.exposed) == 0 && len(c.respondents) == 0 && len(c.opened) == 0 {
		for i, metric := range metrics {
			metrics[i] = matchdayUnavailable(metric.Code, metric.Name, metric.Description, metric.Formula, metric.Unit, "not_applicable", "Sondaggio disattivato per l’evento.", "events.show_feedback_survey")
		}
	}
	matchdayPut(f, metrics...)

	for _, identity := range c.respondents {
		f.Actions = append(f.Actions, matchdayAction{Identity: identity, Feature: "feedback", Count: 1})
	}
	if anonymous := d.records - len(c.respondents); anonymous > 0 {
		f.Actions = append(f.Actions, matchdayAction{Feature: "feedback", Count: anonymous})
	}
}
