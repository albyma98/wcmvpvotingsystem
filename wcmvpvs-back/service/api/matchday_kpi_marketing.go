package api

import (
	"context"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	matchdayChannelSources = "tracking_events.session_id, device_id, fan_id, occurred_at, metadata_json (session.started)"
	matchdayReportChannels = "engagement_report_signals.session_id, device_key, fan_id, occurred_at, metadata.source (page_view)"
)

func loadMatchdayMarketing(ctx context.Context, q matchdayQueryer, orgID int, data map[int]*matchdayFacts, voteSecret string) error {
	if len(data) == 0 {
		return nil
	}
	if err := loadMatchdayFeedback(ctx, q, orgID, data); err != nil {
		return err
	}
	return loadMatchdaySponsors(ctx, q, orgID, data, voteSecret)
}

type matchdayEntry struct {
	channel, person string
	at              time.Time
	qr              bool
}

type matchdayChannel struct {
	access, acquired, activated int
	people                      map[string]bool
}

type matchdayAcquisition struct {
	channels                                 map[string]*matchdayChannel
	people                                   map[string]bool
	access, acquired, comparable, active, qr int
}

func matchdayEntryChannel(rawURL, referrer string) (string, bool) {
	entry, err := url.Parse(rawURL)
	if err != nil || rawURL == "" {
		return "Sorgente sconosciuta", false
	}
	params := entry.Query()
	source := matchdayChannelKey(params.Get("utm_source"))
	if source == "" {
		source = matchdayChannelKey(params.Get("source"))
	}
	medium := matchdayChannelKey(params.Get("utm_medium"))
	qr := medium == "qr" || medium == "qr-code" || medium == "qrcode" ||
		source == "qr" || source == "qr-code" || source == "qrcode" ||
		strings.HasPrefix(source, "qr-") || strings.HasSuffix(source, "-qr")
	switch strings.TrimSuffix(strings.TrimPrefix(source, "qr-"), "-qr") {
	case "led", "led-arena", "arena-led", "ledwall", "led-wall", "maxischermo":
		return "LED", qr
	case "giornalino", "match-programme", "programma-partita", "programma", "magazine":
		return "Giornalino", qr
	case "arena":
		return "Arena", qr
	case "remote":
		return "Remoto", qr
	case "direct":
		return "Diretto", qr
	}
	if qr {
		return "QR fisico", true
	}
	if medium == "social" || medium == "organic-social" || medium == "paid-social" || matchdaySocialSource(source) {
		return "Social", false
	}
	switch {
	case medium == "email" || source == "newsletter" || source == "email":
		return "Email / newsletter", false
	case medium == "cpc" || medium == "paid" || medium == "paid-search":
		return "Campagne a pagamento", false
	case medium == "organic" || source == "google" || source == "bing" || source == "duckduckgo":
		return "Ricerca", false
	case (source != "" && source != "unknown") || medium != "" || params.Get("utm_campaign") != "":
		return "Altre campagne", false
	case referrer == "":
		return "Diretto / senza referrer", false
	}
	ref, err := url.Parse(referrer)
	if err != nil || ref.Hostname() == "" {
		return "Sorgente sconosciuta", false
	}
	host := strings.ToLower(ref.Hostname())
	if strings.EqualFold(host, entry.Hostname()) {
		return "Navigazione interna", false
	}
	for _, domain := range []string{"facebook.com", "instagram.com", "tiktok.com", "x.com", "twitter.com", "linkedin.com", "youtube.com", "youtu.be", "wa.me", "whatsapp.com", "t.co", "fb.com", "t.me", "telegram.org"} {
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return "Social", false
		}
	}
	if strings.HasPrefix(host, "www.google.") || host == "google.com" || host == "bing.com" || strings.HasSuffix(host, ".bing.com") || host == "duckduckgo.com" {
		return "Ricerca", false
	}
	return "Referral", false
}

func matchdaySocialSource(source string) bool {
	switch source {
	case "social", "facebook", "instagram", "ig", "fb", "tiktok", "x", "twitter", "linkedin", "youtube", "whatsapp", "telegram":
		return true
	}
	return false
}

func matchdayChannelKey(value string) string {
	return strings.Join(strings.Fields(strings.NewReplacer("_", " ", "-", " ").Replace(strings.ToLower(value))), "-")
}

func matchdayAcquisitionFor(f *matchdayFacts) matchdayAcquisition {
	out := matchdayAcquisition{channels: map[string]*matchdayChannel{}, people: map[string]bool{}}
	entries := map[string]matchdayEntry{}
	putEntry := func(key, channel string, qr bool, identity matchdayIdentity, at string) {
		if identity.Session == "" {
			return
		}
		key += identity.Session
		entry := matchdayEntry{channel: channel, qr: qr, person: f.Resolve(identity), at: reportMetricTime(at)}
		if current, exists := entries[key]; !exists || matchdayEntryBefore(entry, current) {
			entries[key] = entry
		}
	}
	for _, row := range f.Tracking {
		if row.Name == "session.started" {
			channel, qr := matchdayEntryChannel(reportMetricString(row.Metadata, "url"), reportMetricString(row.Metadata, "referrer"))
			putEntry("tracking:", channel, qr, matchdayTrackingIdentity(row), row.At)
		}
	}
	for _, signal := range f.Signals {
		if signal.Name == "page_view" {
			channel := "Sorgente sconosciuta"
			switch reportMetricString(signal.Metadata, "source") {
			case "arena":
				channel = "Arena"
			case "remote":
				channel = "Remoto"
			}
			putEntry("report:", channel, false, matchdayReportIdentity(signal), signal.OccurredAt)
		}
	}

	first := map[string]matchdayEntry{}
	active := map[string]bool{}
	for _, action := range f.Actions {
		if id := f.Resolve(action.Identity); action.Count > 0 && id != "" {
			active[id] = true
		}
	}
	for _, entry := range entries {
		channel := out.channels[entry.channel]
		if channel == nil {
			channel = &matchdayChannel{people: map[string]bool{}}
			out.channels[entry.channel] = channel
		}
		channel.access++
		out.access++
		if entry.qr {
			out.qr++
		}
		if entry.person == "" {
			continue
		}
		channel.people[entry.person] = true
		out.people[entry.person] = true
		if current, exists := first[entry.person]; !exists || matchdayEntryBefore(entry, current) {
			first[entry.person] = entry
		}
	}
	for person, entry := range first {
		if event := f.FirstEvent[person]; event > 0 && matchdayComparableIdentity(person) {
			out.comparable++
			if event == f.Event.ID {
				out.channels[entry.channel].acquired++
				out.acquired++
			}
		}
		if active[person] {
			out.active++
		}
	}
	for _, channel := range out.channels {
		for person := range channel.people {
			if active[person] {
				channel.activated++
			}
		}
	}
	return out
}

func matchdayEntryBefore(a, b matchdayEntry) bool {
	if a.at.IsZero() != b.at.IsZero() {
		return !a.at.IsZero()
	}
	if a.at.Equal(b.at) {
		return a.channel < b.channel
	}
	return a.at.Before(b.at)
}

func matchdayTrackingIdentity(row matchdayTracking) matchdayIdentity {
	return row.identity()
}

func matchdayReportIdentity(row reportSignal) matchdayIdentity {
	return matchdayIdentity{DeviceKey: row.DeviceID, Session: row.SessionID, FanID: row.FanID}
}

func matchdayDomainPartial(metric matchdayMetric, reason string) matchdayMetric {
	if metric.Value != nil {
		metric.Status = "partial"
		metric.Reason = reason
	}
	return metric
}

func matchdayDomainValue(value float64) *float64 { return &value }

func matchdayDomainRatio(numerator, denominator int, scale float64) *float64 {
	if denominator <= 0 {
		return nil
	}
	return matchdayDomainValue(float64(numerator) * scale / float64(denominator))
}

func calculateMatchdayMarketing(f *matchdayFacts, previous *matchdayFacts) {
	c := matchdayAcquisitionFor(f)
	sources := []string{matchdayChannelSources, matchdayReportChannels}
	const limits = "Copertura delle sole sessioni osservate. La home tifosi registra soltanto Arena, Remoto o sorgente sconosciuta."
	const acquisitionLimits = "Prima presenza osservata nello storico disponibile. L’identità può essere un profilo o un dispositivo persistente."
	const noChannels = "Nessuna sessione di ingresso osservata per la partita."
	const accessFormula = "Sessioni di ingresso distinte e identità uniche per canale."
	const newFormula = "Identità persistenti alla prima partita osservata, attribuite al primo ingresso tracciato."
	const activeFormula = "Utenti con almeno un’azione significativa / utenti del canale × 100."
	const shareFormula = "Accessi del canale / accessi totali osservati × 100. In evidenza la quota maggiore."

	access := matchdayUnavailable("marketing_access", "Accessi e utenti per canale", "Ingressi osservati nelle interfacce tifosi.", accessFormula, "count", "not_measurable", noChannels, sources...)
	acquired := matchdayUnavailable("marketing_new_users", "Nuovi utenti per canale", "Nuove identità osservate per canale.", newFormula, "count", "not_measurable", noChannels, sources...)
	activation := matchdayUnavailable("marketing_activation", "Attivazione per canale", "Quota di visitatori tracciati che ha interagito.", activeFormula, "percent", "not_measurable", noChannels, sources...)
	share := matchdayUnavailable("marketing_traffic_share", "Distribuzione del traffico", "Quota di ingressi per sorgente.", shareFormula, "percent", "not_measurable", noChannels, sources...)

	if c.access > 0 {
		access = matchdayDomainPartial(matchdayCount(access.Code, access.Name, access.Description, accessFormula, float64(c.access), sources...), limits)
		if c.comparable > 0 {
			acquired = matchdayDomainPartial(matchdayCount(acquired.Code, acquired.Name, acquired.Description, newFormula, float64(c.acquired), sources...), acquisitionLimits)
		} else {
			acquired.Reason = "Nessuna identità persistente confrontabile tra gli ingressi osservati."
		}
		activation = matchdayDomainPartial(matchdayRatio(activation.Code, activation.Name, activation.Description, activeFormula, "percent", float64(c.active), float64(len(c.people)), sources...), limits)
		largest := 0
		for _, channel := range matchdaySortedChannels(c.channels) {
			a := c.channels[channel]
			if a.access > largest {
				largest = a.access
			}
			row := matchdayRow{Label: channel, Values: map[string]*float64{
				"access": matchdayDomainValue(float64(a.access)), "unique": matchdayDomainValue(float64(len(a.people))),
				"new": matchdayDomainValue(float64(a.acquired)), "active": matchdayDomainRatio(a.activated, len(a.people), 100),
				"share": matchdayDomainRatio(a.access, c.access, 100),
			}}
			access.Rows = append(access.Rows, row)
			if c.comparable > 0 {
				acquired.Rows = append(acquired.Rows, row)
			}
			activation.Rows = append(activation.Rows, row)
		}
		share = matchdayDomainPartial(matchdayRatio(share.Code, share.Name, share.Description, shareFormula, "percent", float64(largest), float64(c.access), sources...), limits)
		share.Rows = access.Rows
		access.Columns = []matchdayColumn{{ID: "access", Label: "Accessi", Unit: "count"}, {ID: "unique", Label: "Utenti unici", Unit: "count"}}
		acquired.Columns = []matchdayColumn{{ID: "new", Label: "Nuovi utenti", Unit: "count"}}
		activation.Columns = []matchdayColumn{{ID: "active", Label: "Attivazione", Unit: "percent"}}
		share.Columns = []matchdayColumn{{ID: "share", Label: "Traffico", Unit: "percent"}}
	}
	qr := matchdayUnavailable("marketing_qr_conversion", "Conversione QR", "Sessioni d’ingresso attribuite esplicitamente a un QR.", "Sessioni QR / scansioni QR effettive × 100.", "percent", "missing", "Servono le scansioni effettive rilevate da una fonte esterna.", append(sources, "Integrazione manuale: qr_scans")...)
	qr.Missing = []string{"qr_scans"}
	if scans := f.Manual["qr_scans"].Value; scans != nil {
		qr.Missing = nil
		qr.Denominator = scans
		qr.Source = "mixed"
		if c.qr > 0 {
			qr = matchdayRatio(qr.Code, qr.Name, qr.Description, qr.Formula, "percent", float64(c.qr), *scans, qr.Sources...)
			qr.Source = "mixed"
			qr = matchdayDomainPartial(qr, "Verificare che scansioni e sessioni si riferiscano agli stessi QR.")
		} else {
			qr.Status = "not_measurable"
			qr.Reason = "Nessuna sessione con attribuzione QR esplicita osservata."
		}
	}
	cpa := matchdayUnavailable("marketing_cpa", "Costo per acquisizione", "Spesa promozionale per nuova identità osservata.", "Spesa promozionale / nuove identità acquisite osservate.", "eur", "missing", "Inserisci la spesa promozionale della partita.", append(sources, "Integrazione manuale: marketing_spend")...)
	cpa.Missing = []string{"marketing_spend"}
	if spend := f.Manual["marketing_spend"].Value; spend != nil {
		cpa.Missing = nil
		cpa.Numerator = spend
		cpa.Source = "mixed"
		if acquired.Value == nil {
			cpa.Status = "not_measurable"
			cpa.Reason = acquired.Reason
		} else {
			cpa = matchdayRatio(cpa.Code, cpa.Name, cpa.Description, cpa.Formula, "eur", *spend, *acquired.Value, cpa.Sources...)
			cpa.Source = "mixed"
			cpa = matchdayDomainPartial(cpa, acquisitionLimits)
		}
	}

	change := matchdayUnavailable("marketing_channel_change", "Variazione per canale", "Confronto con la partita precedente selezionata.", "Accessi correnti − accessi precedenti. Variazione % = differenza / accessi precedenti × 100.", "count", "not_measurable", "Storico insufficiente.", sources...)
	if previous != nil && c.access > 0 {
		p := matchdayAcquisitionFor(previous)
		if p.access > 0 {
			change = matchdayDomainPartial(matchdayCount(change.Code, change.Name, change.Description, change.Formula, float64(c.access-p.access), sources...), limits)
			change.Columns = []matchdayColumn{{ID: "current", Label: "Accessi", Unit: "count"}, {ID: "previous", Label: "Accessi precedenti", Unit: "count"}, {ID: "delta", Label: "Differenza", Unit: "count"}, {ID: "change", Label: "Variazione", Unit: "percent"}, {ID: "unique_delta", Label: "Differenza utenti", Unit: "count"}}
			channels := map[string]*matchdayChannel{}
			for k, v := range c.channels {
				channels[k] = v
			}
			for k, v := range p.channels {
				channels[k] = v
			}
			for _, channel := range matchdaySortedChannels(channels) {
				a, b := c.channels[channel], p.channels[channel]
				if a == nil {
					a = &matchdayChannel{}
				}
				if b == nil {
					b = &matchdayChannel{}
				}
				delta := a.access - b.access
				change.Rows = append(change.Rows, matchdayRow{Label: channel, Values: map[string]*float64{
					"current": matchdayDomainValue(float64(a.access)), "previous": matchdayDomainValue(float64(b.access)),
					"delta": matchdayDomainValue(float64(delta)), "change": matchdayDomainRatio(delta, b.access, 100),
					"unique_delta": matchdayDomainValue(float64(len(a.people) - len(b.people))),
				}})
			}
		}
	}
	matchdayPut(f, access, acquired, activation, share, qr, cpa, change)
}

func matchdaySortedChannels(channels map[string]*matchdayChannel) []string {
	names := make([]string, 0, len(channels))
	for name := range channels {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
