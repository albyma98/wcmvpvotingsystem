package api

// Operational records supplement browser observations; they are never written
// back into engagement_report_signals. Both sources use the same event-local
// identity and round keys, so one action is not counted twice.
func calculateCombinedReportMetrics(signals, previous []reportSignal, operational, previousOperational reportOperationalData) (map[string]*float64, reportDetails, string) {
	all := append(append([]reportSignal{}, signals...), operational.Signals...)
	prior := append(append([]reportSignal{}, previous...), previousOperational.Signals...)
	facts, details, started := calculateReportMetrics(all, prior)
	details.Sources = append([]reportSourceRows{{Table: "engagement_report_signals", Rows: len(signals)}}, operational.Sources...)
	details.Operational = &operational.Summary
	hasOperational := len(operational.Signals) > 0 || operational.Summary.FeedbackResponses > 0 || operational.Summary.SponsorClicks > 0
	if !hasOperational {
		return facts, details, started
	}
	details.CoverageNotes = []string{
		"Le attività operative e i segnali /new sono uniti per profilo o dispositivo nell'evento. Le tabelle condivise non conservano sempre la pagina di origine: le attività storiche sono attribuite all'evento, non esclusivamente a /new.",
		"I partecipanti osservabili sono un minimo verificabile: le attività salvate non ricostruiscono le visite passive. Le percentuali descrivono solo il pubblico misurato; senza il relativo denominatore restano N/D.",
		"Le giocate confermate uniscono round tracciati e ricevute di premio con lo stesso ID; nel basket ogni tiro risolto è un round. I premi legacy senza ID valgono al massimo una prova per partecipante e gioco. Tiri non risolti e sessioni aperte non provano un completamento.",
	}
	seen := map[string]bool{}
	shopAudience := false
	for _, signal := range signals {
		seen[signal.Name] = true
		if signal.FanID > 0 && (signal.Name == "page_view" || signal.Name == "shop_view") {
			shopAudience = true
		}
	}
	if !seen["active_time"] {
		facts["active_seconds"] = nil
	}
	if !seen["registration_completed"] {
		facts["new_profiles"] = nil
	}
	if !seen["registration_started"] && !seen["registration_completed"] {
		facts["registration_starters"] = nil
	}
	if !shopAudience {
		facts["shop_eligible_profiles"] = nil
	}
	if !seen["sponsor_view"] && !seen["sponsored_game_view"] {
		for _, key := range []string{"sponsor_reach", "sponsor_impressions", "sponsor_clickers", "sponsored_players", "sponsored_exposed_visitors"} {
			facts[key] = nil
		}
	}
	if !seen["feedback_completed"] {
		for _, key := range []string{"feedback_respondents", "experience_responses", "experience_positive", "perks_positive", "perks_responses"} {
			facts[key] = nil
		}
	}
	if operational.Summary.FeedbackResponses > 0 {
		// event_feedback has no device/fan identifier. Do not invent unique
		// respondents or add anonymous answers to their browser copies.
		details.CoverageNotes = append(details.CoverageNotes, "I feedback salvati sono risposte anonime, non utenti unici. Il tasso di risposta usa solo il campione identificabile nei segnali, senza aggiungere le risposte anonime. L'interesse premi usa le risposte valide salvate, senza sommare le copie telemetriche. La facilità di voto storica non è soddisfazione complessiva.")
	}
	if operational.Feedback.PerksResponses > 0 {
		facts["perks_responses"] = reportMetricFloat(float64(operational.Feedback.PerksResponses))
		facts["perks_positive"] = reportMetricFloat(float64(operational.Feedback.PerksPositive))
	}
	return facts, details, started
}

func buildCombinedReportKPIs(facts map[string]*float64, details reportDetails, signals, previous []reportSignal, operational, previousOperational reportOperationalData) []reportKPI {
	kpis := buildReportKPIs(facts)
	contains := func(items []reportSignal, names ...string) bool {
		for _, item := range items {
			for _, name := range names {
				if name == "*" || item.Name == name {
					return true
				}
			}
		}
		return false
	}
	for i := range kpis {
		k := &kpis[i]
		names := []string{}
		switch k.ID {
		case "visitors", "active_participation", "multi_experience", "event_retention":
			names = []string{"*"}
		case "mvp_participation":
			names = []string{"vote_completed"}
		case "game_participation", "completed_rounds", "replay_rate":
			names = []string{"game_started", "game_completed"}
		case "redemption_conversion":
			names = []string{"redemption_completed"}
		case "sponsor_ctr":
			names = []string{"sponsor_click"}
		}
		tracked, raw := signals, operational.Signals
		if k.ID == "event_retention" {
			tracked = append(append([]reportSignal{}, signals...), previous...)
			raw = append(append([]reportSignal{}, raw...), previousOperational.Signals...)
		}
		hasOperational := contains(raw, names...)
		hasTracked := contains(tracked, names...)
		if hasOperational && k.ID == "visitors" {
			k.Label = "Partecipanti digitali osservabili"
			k.Description = "Unione dei profili e dispositivi osservati nei segnali e nelle attività salvate dell'evento. È un minimo verificabile e non include tutte le visite passive storiche."
		}
		if hasOperational && k.ID == "completed_rounds" {
			k.Label = "Giocate completate confermate"
		}
		if hasOperational && k.Value != nil {
			k.Source = "operational"
			if hasTracked || k.NumeratorKey != "" && k.ID != "event_retention" && len(signals) > 0 {
				k.Source = "mixed"
			}
			k.Note = "Dati operativi dell'evento e osservazioni disponibili, deduplicati per identità e round. Copertura minima verificabile; consultare le note sulle fonti."
		}
		if k.ID == "perks_interest" && operational.Feedback.PerksResponses > 0 {
			k.Source = "operational"
			k.Note = "Risposte valide di event_feedback; le copie dei segnali non vengono sommate. Il campione è composto da risposte anonime, non da utenti unici."
		}
		if k.ID == "feedback_response" && k.Value != nil && operational.Summary.FeedbackResponses > 0 {
			k.Note = "Rispondenti unici e pubblico eleggibile del solo campione tracciato. Le risposte anonime salvate sono riportate separatamente e non vengono aggiunte al numeratore."
		}
		if k.Value == nil && details.Operational != nil && (len(operational.Signals) > 0 || operational.Summary.FeedbackResponses > 0 || operational.Summary.SponsorClicks > 0) {
			switch k.ID {
			case "active_participation":
				k.Note = "Le azioni salvate provano partecipazione, ma senza osservazioni della home non misurano l'intero pubblico digitale."
			case "new_profiles", "registration_conversion":
				k.Note = "Le anagrafiche del club non conservano l'evento di registrazione. Senza segnali del relativo percorso l'attribuzione a questo evento non è verificabile."
			case "feedback_response":
				k.Note = "event_feedback non identifica il rispondente: è disponibile il numero di risposte, non il tasso su utenti unici."
			case "satisfaction":
				k.Note = "La domanda storica sulla facilità di voto non misura soddisfazione. Occorrono risposte alla domanda dedicata sull'esperienza."
			}
		}
	}
	return kpis
}
