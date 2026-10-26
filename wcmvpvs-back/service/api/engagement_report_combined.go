package api

import "fmt"

// Operational records supplement browser observations; they are never written
// back into engagement_report_signals. Both sources use the same event-local
// identity and round keys, so one action is not counted twice.
func calculateCombinedReportMetrics(signals, previous []reportSignal, operational, previousOperational reportOperationalData) (map[string]*float64, reportDetails, string) {
	signals, previous = reportMetricSignals(signals), reportMetricSignals(previous)
	all := append(append([]reportSignal{}, signals...), operational.Signals...)
	prior := append(append([]reportSignal{}, previous...), previousOperational.Signals...)
	facts, details, started := calculateReportMetrics(all, prior)
	details.Sources = append([]reportSourceRows{{Table: "engagement_report_signals", Rows: len(signals)}}, operational.Sources...)
	details.Operational = &operational.Summary
	// Durable totals own their metric. Browser observations are a fallback,
	// never an additional copy of a saved click, purchase or feedback response.
	if operational.Summary.SponsorClicks > 0 {
		facts["total_sponsor_clicks"] = reportMetricFloat(float64(operational.Summary.SponsorClicks))
	}
	if operational.Summary.ShopRedemptions > 0 {
		facts["shop_redemptions"] = reportMetricFloat(float64(operational.Summary.ShopRedemptions))
	}
	if operational.Summary.FeedbackResponses > 0 {
		facts["feedback_collected"] = reportMetricFloat(float64(operational.Summary.FeedbackResponses))
	}
	if details.TimeEstimate != nil {
		t := details.TimeEstimate
		details.CoverageNotes = append(details.CoverageNotes, fmt.Sprintf("Tempo stimato: durata ricostruibile per %d identità su %d osservate; %d escluse dalla media. I tempi misurati sono uniti ai primi 60 secondi tra azioni distanti al massimo 5 minuti, senza sommare sovrapposizioni. Non è una misura certificata di permanenza.", t.CoveredVisitors, t.TotalVisitors, t.ExcludedVisitors))
	}
	hasOperational := len(operational.Signals) > 0 || operational.Summary.FeedbackResponses > 0 || operational.Summary.SponsorClicks > 0 || operational.Summary.ShopRedemptions > 0 || operational.Summary.Votes > 0 || operational.Summary.GameRewardReceipts > 0
	if !hasOperational {
		return facts, details, started
	}
	details.CoverageNotes = append(details.CoverageNotes,
		"Le attività operative e i segnali /new sono uniti per profilo o dispositivo nell'evento. Le tabelle condivise non conservano sempre la pagina di origine: le attività storiche sono attribuite all'evento, non esclusivamente a /new.",
		"I partecipanti osservabili sono un minimo verificabile: le attività salvate non ricostruiscono le visite passive. Le percentuali descrivono solo il pubblico misurato; senza il relativo denominatore restano N/D.",
		"Le giocate confermate uniscono round tracciati e ricevute di premio con lo stesso ID; nel basket ogni tiro risolto è un round. I premi legacy senza ID valgono al massimo una prova per partecipante e gioco. Tiri non risolti e sessioni aperte non provano un completamento.",
	)
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
		for _, key := range []string{"sponsor_reach", "sponsor_impressions", "sponsor_clickers", "sponsored_players", "sponsored_exposed_visitors", "sponsor_engaged_exposed"} {
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
	signals, previous = reportMetricSignals(signals), reportMetricSignals(previous)
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
		case "visitors", "active_participation", "multi_experience", "event_retention", "identified_visitors", "guest_visitors", "audience_registration_conversion":
			names = []string{"*"}
		case "mvp_participation":
			names = []string{"vote_completed"}
		case "game_participation", "completed_rounds", "replay_rate":
			names = []string{"game_started", "game_completed"}
		case "redemption_conversion", "shop_redemptions":
			names = []string{"redemption_completed"}
		case "sponsor_ctr", "sponsor_clicks":
			names = []string{"sponsor_click"}
		case "sponsor_interactors", "sponsor_engagement":
			names = []string{"sponsor_click", "game_started", "game_completed"}
		}
		tracked, raw := signals, operational.Signals
		if k.ID == "event_retention" {
			tracked = append(append([]reportSignal{}, signals...), previous...)
			raw = append(append([]reportSignal{}, raw...), previousOperational.Signals...)
		}
		hasOperational := contains(raw, names...)
		hasTracked := contains(tracked, names...)
		if k.ID == "sponsor_interactors" || k.ID == "sponsor_engagement" {
			attributed := func(items []reportSignal) bool {
				for _, item := range items {
					if reportMetricNumber(item.Metadata, "sponsor_id") > 0 && (item.Name == "sponsor_click" || item.Name == "game_started" || item.Name == "game_completed") {
						return true
					}
				}
				return false
			}
			hasOperational, hasTracked = attributed(raw), attributed(tracked)
		}
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
		if k.ID == "estimated_total_time" || k.ID == "estimated_avg_time" {
			if details.TimeEstimate != nil && k.Value != nil {
				t := details.TimeEstimate
				k.Source = "estimated"
				k.Note = fmt.Sprintf("Stima su %d identità di %d osservate; %d escluse dalla media. Massimo 60 secondi tra azioni consecutive, esclusi gap oltre 5 minuti; sovrapposizioni con heartbeat eliminate. Non è tempo di permanenza misurato.", t.CoveredVisitors, t.TotalVisitors, t.ExcludedVisitors)
			} else {
				k.Note = "Nessun intervallo valido ricostruibile: una sola azione o azioni distanti oltre 5 minuti non giustificano una durata. I partecipanti senza durata non sono considerati pari a zero."
			}
		}
		total := 0
		switch k.ID {
		case "sponsor_clicks":
			total = operational.Summary.SponsorClicks
		case "shop_redemptions":
			total = operational.Summary.ShopRedemptions
		case "feedback_collected":
			total = operational.Summary.FeedbackResponses
		}
		if total > 0 {
			k.Source = "operational"
			k.Note = "Totale delle azioni salvate e attribuite all'evento. Le copie dei segnali non sono sommate. I feedback e i clic possono non identificare una persona."
		} else if k.Value != nil && (k.ID == "sponsor_clicks" || k.ID == "shop_redemptions" || k.ID == "feedback_collected") {
			k.Source = "new_tracking"
			k.Note = "Nessun totale operativo disponibile: minimo osservato nei segnali, deduplicato. Non equivale alla copertura completa di tutte le azioni dell'evento."
		}
		if k.ID == "sponsor_interactors" && k.Value != nil {
			k.Note = "Identità con clic o giochi attribuiti a uno sponsor, deduplicate anche tra sponsor. Clic anonimi e giochi senza attribuzione sponsor sono esclusi."
		}
		if k.ID == "sponsor_engagement" && k.Value != nil {
			k.Note = "Intersezione tra esposizione qualificata e clic/gioco dello stesso sponsor; ogni identità conta una sola volta nel totale."
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
