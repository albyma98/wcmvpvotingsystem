package api

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

type matchdayChatFacts struct {
	enabled                                              bool
	measured, complete, everEnabled, openWithoutIdentity bool
	fans, lostAuthors, official, bots                    int
	reported, removed, moderated, fanReactions           int
	replies, blocked                                     int
	openIdentities                                       []matchdayIdentity
	authors                                              map[int]bool
}

type matchdayOperationsFacts struct {
	chat         matchdayChatFacts
	merch        matchdayMerchFacts
	virtualItems map[string]int
	barOrders    int
	barPaid      int
	barClosed    int
}

func (f matchdayOperationsFacts) eachIdentity(use func(matchdayIdentity)) {
	for _, group := range [][]matchdayIdentity{f.merch.visitIdentities, f.merch.buyerIdentities, f.chat.openIdentities} {
		for _, identity := range group {
			use(identity)
		}
	}
}

func loadMatchdayOperations(ctx context.Context, q matchdayQueryer, orgID int, data map[int]*matchdayFacts) error {
	if len(data) == 0 {
		return nil
	}
	if err := loadMatchdayChat(ctx, q, orgID, data); err != nil {
		return fmt.Errorf("matchday chat: %w", err)
	}
	if err := loadMatchdayChatMeasurement(ctx, q, orgID, data); err != nil {
		return fmt.Errorf("matchday chat measurement: %w", err)
	}
	ids, args := matchdayIDs(data)
	rows, err := q.QueryContext(ctx, `SELECT event_id, source, COUNT(*) FROM fan_wallet_movements
		WHERE event_id IN (`+ids+`) AND organization_id=? AND delta<0 AND source IN ('shop:club_frame','shop:match_card')
		GROUP BY event_id,source`, append(args, orgID)...)
	if err != nil {
		return fmt.Errorf("matchday virtual shop: %w", err)
	}
	for rows.Next() {
		var eventID, count int
		var source string
		if err := rows.Scan(&eventID, &source, &count); err != nil {
			rows.Close()
			return err
		}
		f := data[eventID]
		if f.Operations.virtualItems == nil {
			f.Operations.virtualItems = map[string]int{}
		}
		f.Operations.virtualItems[strings.TrimPrefix(source, "shop:")] = count
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if err := loadMatchdayBar(ctx, q, orgID, data); err != nil {
		return fmt.Errorf("matchday bar: %w", err)
	}
	if err := loadMatchdayMerch(ctx, q, orgID, data); err != nil {
		return fmt.Errorf("matchday merchandising: %w", err)
	}
	return nil
}

func loadMatchdayChat(ctx context.Context, q matchdayQueryer, orgID int, data map[int]*matchdayFacts) error {
	ids, args := matchdayIDs(data)
	rows, err := q.QueryContext(ctx, `
		WITH selected AS (
			SELECT id,organization_id FROM events WHERE id IN (`+ids+`) AND organization_id=?
		), reports AS (
			SELECT event_id,message_id,MAX(review_status='reviewed') AS reviewed
			FROM event_chat_reports WHERE event_id IN (SELECT id FROM selected) GROUP BY event_id,message_id
		), messages AS (
			SELECT m.event_id,m.message_type,COALESCE(p.id,0) AS author_id,COUNT(*) AS messages,
				SUM(r.message_id IS NOT NULL) AS reported,SUM(m.moderation_status='removed') AS removed,
				SUM(m.moderation_status='removed' OR COALESCE(r.reviewed,0)=1) AS moderated
			FROM event_chat_messages m JOIN selected e ON e.id=m.event_id
			LEFT JOIN fan_profiles p ON p.id=m.author_user_id AND p.organization_id=e.organization_id
			LEFT JOIN reports r ON r.event_id=m.event_id AND r.message_id=m.id
			GROUP BY m.event_id,m.message_type,COALESCE(p.id,0)
		), reactions AS (
			SELECT r.event_id,COUNT(*) AS reactions
			FROM selected e JOIN event_chat_reactions r ON r.event_id=e.id
			JOIN event_chat_messages m ON m.event_id=r.event_id AND m.id=r.message_id
			JOIN fan_profiles p ON p.id=r.fan_id AND p.organization_id=e.organization_id
			WHERE m.message_type='fan' AND m.author_user_id IS NOT r.fan_id
			GROUP BY r.event_id
		)
		SELECT e.id,COALESCE(s.enabled,0),COALESCE(m.message_type,''),COALESCE(m.author_id,0),
			COALESCE(m.messages,0),COALESCE(m.reported,0),COALESCE(m.removed,0),COALESCE(m.moderated,0),COALESCE(r.reactions,0)
		FROM selected e LEFT JOIN event_chat_settings s ON s.event_id=e.id
		LEFT JOIN messages m ON m.event_id=e.id LEFT JOIN reactions r ON r.event_id=e.id`, append(args, orgID)...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var eventID, author, count, reported, removed, moderated, reactions int
		var enabled bool
		var kind string
		if err := rows.Scan(&eventID, &enabled, &kind, &author, &count, &reported, &removed, &moderated, &reactions); err != nil {
			return err
		}
		f := data[eventID]
		chat := &f.Operations.chat
		chat.enabled, chat.fanReactions = enabled, reactions
		switch kind {
		case "fan":
			chat.fans += count
			chat.reported += reported
			chat.removed += removed
			chat.moderated += moderated
			if author > 0 {
				if chat.authors == nil {
					chat.authors = map[int]bool{}
				}
				chat.authors[author] = true
			} else {
				chat.lostAuthors += count
			}
			f.Actions = append(f.Actions, matchdayAction{Identity: matchdayIdentity{FanID: author}, Feature: "chat", Count: count})
		case "official":
			chat.official += count
		case "ai":
			chat.bots += count
		}
	}
	return rows.Err()
}

func loadMatchdayBar(ctx context.Context, q matchdayQueryer, orgID int, data map[int]*matchdayFacts) error {
	ids, args := matchdayIDs(data)
	// A Stripe reference can be attached to one match only. The reference is
	// checked against the order, and conflicting observations across events
	// are dropped even when the other event is outside the comparison window.
	rows, err := q.QueryContext(ctx, `
		WITH observations AS (
			SELECT o.id,o.stripe_reference,o.payment_status,o.order_status,o.total_cents,MIN(t.event_id) AS event_id
			FROM events e JOIN tracking_events t ON t.event_id=e.id
			JOIN bar_orders o ON o.organization_id=e.organization_id
				AND o.stripe_reference=CASE WHEN json_valid(t.metadata_json) THEN json_extract(t.metadata_json,'$.session_id') END
			WHERE e.organization_id=? AND t.organization_id IN (0,e.organization_id) AND o.stripe_reference<>''
				AND t.event_name IN ('bar.checkout_redirected','bar.order_completed')
			GROUP BY o.id HAVING COUNT(DISTINCT t.event_id)=1
		)
		SELECT event_id,stripe_reference,payment_status,order_status,total_cents FROM observations
		WHERE event_id IN (`+ids+`)`, append([]interface{}{orgID}, args...)...)
	if err != nil {
		return err
	}
	defer rows.Close()
	// Build the browser identity index once per match. The payment outcome
	// remains countable even when identity evidence is absent or conflicting.
	devices := map[int]map[string]string{}
	for eventID, f := range data {
		for _, item := range f.Tracking {
			if item.Name != "bar.checkout_redirected" || item.Device == "" {
				continue
			}
			ref, _ := item.Metadata["session_id"].(string)
			if ref == "" {
				continue
			}
			if devices[eventID] == nil {
				devices[eventID] = map[string]string{}
			}
			if previous, exists := devices[eventID][ref]; exists && previous != item.Device {
				devices[eventID][ref] = ""
			} else {
				devices[eventID][ref] = item.Device
			}
		}
	}
	for rows.Next() {
		var eventID int
		var cents int64
		var reference, payment, order string
		if err := rows.Scan(&eventID, &reference, &payment, &order, &cents); err != nil {
			return err
		}
		f := data[eventID]
		f.Operations.barOrders++
		paid := payment == "paid" && normalizeBarStatus(order) != "annullato"
		if paid || payment == "expired" {
			f.Operations.barClosed++
		}
		if paid {
			f.Operations.barPaid++
			if f.BarRevenue == nil {
				f.BarRevenue = new(float64)
			}
			*f.BarRevenue += float64(cents) / 100
			f.Actions = append(f.Actions, matchdayAction{Identity: matchdayIdentity{Device: devices[eventID][reference]}, Feature: "purchase", Count: 1})
		}
	}
	return rows.Err()
}

func calculateMatchdayOperations(f *matchdayFacts) {
	calculateMatchdayShop(f)
	calculateMatchdayChat(f)
	calculateMatchdayTechnology(f)
}

func calculateMatchdayShop(f *matchdayFacts) {
	const shopSignals = "engagement_report_signals.name, device_key, fan_id, session_id"
	const merchOrders = "merch_orders.organization_id, payment_status, fulfillment_status, total_cents"
	const merchViews = "merch_analytics_events.organization_id, event_id, type, product_id"
	const virtualLedger = "fan_wallet_movements.event_id, organization_id, source, delta"
	virtualOpens, barOpens := 0, 0
	visitors := map[string]bool{}
	views := map[string]int{}
	for _, signal := range f.Signals {
		switch signal.Name {
		case "shop_view":
			virtualOpens++
			identity := f.Resolve(matchdaySignalIdentity(signal))
			if identity != "" {
				visitors[identity] = true
			}
		case "shop_item_viewed":
			key, _ := signal.Metadata["item_key"].(string)
			if signal.Metadata["placement"] != "profile" && (key == "club_frame" || key == "match_card") {
				views[key]++
			}
		}
	}
	for _, item := range f.Tracking {
		if item.Name == "bar.menu_opened" {
			barOpens++
		}
	}
	const virtualUnavailable = "Non risultano aperture dello shop virtuale nella telemetria dell'evento."
	opens := matchdayUnavailable("shop_opens", "Aperture shop virtuale", "Aperture della sezione riscatti in monete virtuali.", "Conteggio shop_view", "count", "not_measurable", virtualUnavailable, shopSignals)
	unique := matchdayUnavailable("shop_visitors", "Visitatori unici shop virtuale", "Identità uniche osservate nella sezione riscatti in monete virtuali.", "Identità distinte con shop_view", "count", "not_measurable", virtualUnavailable, shopSignals)
	if virtualOpens > 0 {
		opens = matchdayCount(opens.Code, opens.Name, opens.Description, opens.Formula, float64(virtualOpens), shopSignals)
		if len(visitors) > 0 {
			unique = matchdayCount(unique.Code, unique.Name, unique.Description, unique.Formula, float64(len(visitors)), shopSignals)
		} else {
			unique.Reason = "Sono state osservate aperture senza identità associata."
		}
	}
	const noMatch = "Lo shop merchandising registra dati per società, senza associazione alla partita."
	const noMatchVisitor = "Le analytics merchandising non identificano i visitatori e non associano le aperture alla partita."
	matchdayPut(f, opens, unique,
		matchdayUnavailable("shop_merch_opens", "Aperture shop merchandising", "Aperture dello shop commerciale attribuibili alla partita.", "Conteggio aperture commerciali", "count", "not_measurable", noMatchVisitor, merchViews),
		matchdayUnavailable("shop_merch_visitors", "Visitatori unici shop merchandising", "Identità uniche che hanno aperto lo shop commerciale per la partita.", "Identità distinte con apertura dello shop", "count", "not_measurable", noMatchVisitor, merchViews),
		matchdayUnavailable("shop_conversion_rate", "Conversione acquisti merchandising", "Acquirenti unici sullo stesso pubblico che ha visitato lo shop commerciale.", "Acquirenti unici / visitatori unici dello shop × 100", "percent", "not_measurable", noMatchVisitor, merchViews, merchOrders),
		matchdayUnavailable("shop_refunds", "Rimborsi merchandising", "Importi rimborsati attribuibili alla partita, distinti dal valore degli ordini.", "Somma dei rimborsi confermati", "eur", "not_measurable", noMatch, "merch_payment_records.organization_id, order_id, status, amount_cents"),
		matchdayUnavailable("shop_unpaid_reservations", "Prenotazioni non pagate", "Prenotazioni confermate attribuibili alla partita che risultano ancora da pagare.", "Conteggio prenotazioni confermate e non pagate", "count", "not_measurable", noMatch, merchOrders))
	paid := matchdayShopManual(f, "shop_paid_orders", "Acquisti confermati", "Ordini merchandising pagati e attribuiti alla partita, esclusi falliti, annullati e rimborsati.", "Conteggio ordini confermati distinti", "count", merchOrders)
	gmv := matchdayShopManual(f, "shop_gmv", "Valore lordo degli ordini confermati (GMV)", "Valore degli acquisti merchandising confermati attribuibili alla partita.", "Somma degli importi lordi degli ordini confermati", "eur", merchOrders)
	reservations := matchdayShopManual(f, "shop_reservations", "Prenotazioni confermate", "Prenotazioni merchandising attribuibili alla partita, incluse quelle non pagate già confermate.", "Conteggio prenotazioni confermate distinte", "count", merchOrders)
	aov := matchdayUnavailable("shop_aov", "Valore medio ordine", "Valore medio per acquisto merchandising confermato.", "GMV / acquisti confermati", "eur", "missing", "Servono GMV e acquisti confermati dello stesso perimetro.", merchOrders)
	if gmv.Value == nil {
		aov.Missing = append(aov.Missing, gmv.Code)
	}
	if paid.Value == nil {
		aov.Missing = append(aov.Missing, paid.Code)
	}
	if paid.Value != nil && gmv.Value != nil {
		aov = matchdayRatio(aov.Code, aov.Name, aov.Description, aov.Formula, aov.Unit, *gmv.Value, *paid.Value, "Integrazioni amministratore: GMV e acquisti confermati")
		aov.Source = "manual"
	}
	executive := matchdayUnavailable("shop_conversions", "Conversioni shop", "Acquisti e prenotazioni attribuibili alla partita, mostrati separatamente.", "Conteggi distinti per tipo", "count", "missing", noMatch, merchOrders)
	executive.Columns = []matchdayColumn{{ID: "count", Label: "Totale", Unit: "count"}}
	executive.Rows = []matchdayRow{{Label: paid.Name, Values: map[string]*float64{"count": paid.Value}}, {Label: reservations.Name, Values: map[string]*float64{"count": reservations.Value}}}
	if paid.Value != nil || reservations.Value != nil {
		executive.Source = "manual"
		executive.Sources = []string{"Integrazioni amministratore: acquisti e prenotazioni"}
		executive.Status = "partial"
		if paid.Value != nil && reservations.Value != nil {
			executive.Status, executive.Reason = "available", ""
		}
	}
	if paid.Value == nil {
		executive.Missing = append(executive.Missing, paid.Code)
	}
	if reservations.Value == nil {
		executive.Missing = append(executive.Missing, reservations.Code)
	}
	products := matchdayUnavailable("shop_top_products", "Prodotti più visualizzati o acquistati", "La classifica disponibile riguarda gli oggetti dello shop virtuale.", "Visualizzazioni dello shop e riscatti per oggetto", "count", "not_measurable", "Non risultano dati attribuibili alla partita per i prodotti dello shop merchandising.", merchViews, shopSignals, virtualLedger)
	if len(views) > 0 || len(f.Operations.virtualItems) > 0 {
		products.Status, products.Source = "partial", "db"
		products.Columns = []matchdayColumn{{ID: "views", Label: "Visualizzazioni", Unit: "count"}, {ID: "redemptions", Label: "Riscatti", Unit: "count"}}
		for _, item := range []struct{ key, label string }{{"club_frame", "Cornice società"}, {"match_card", "Carta partita"}} {
			viewCount, redemptions := views[item.key], f.Operations.virtualItems[item.key]
			if viewCount == 0 && redemptions == 0 {
				continue
			}
			values := map[string]*float64{"redemptions": matchdayOperationsValue(redemptions)}
			if len(views) > 0 {
				values["views"] = matchdayOperationsValue(viewCount)
			}
			products.Rows = append(products.Rows, matchdayRow{Label: item.label, Group: "Shop virtuale", Values: values})
		}
		sort.SliceStable(products.Rows, func(i, j int) bool {
			if len(views) > 0 && *products.Rows[i].Values["views"] != *products.Rows[j].Values["views"] {
				return *products.Rows[i].Values["views"] > *products.Rows[j].Values["views"]
			}
			return *products.Rows[i].Values["redemptions"] > *products.Rows[j].Values["redemptions"]
		})
	}
	matchdayPut(f, paid, gmv, reservations, aov, executive, products)
	bar := matchdayUnavailable("shop_bar_paid_orders", "Ordini pagati al bar", "Ordini del bar digitale collegati alla partita tramite osservazioni del checkout.", "Ordini pagati distinti per riferimento Stripe", "count", "not_measurable", "Non è stato possibile collegare gli ordini del bar alla partita.", "bar_orders.stripe_reference, payment_status, order_status, total_cents", "tracking_events.event_name, metadata_json.session_id")
	if f.Operations.barOrders > 0 {
		bar = matchdayCount(bar.Code, bar.Name, bar.Description, bar.Formula, float64(f.Operations.barPaid), bar.Sources...)
		bar.Status, bar.Reason = "partial", "La copertura dipende dai riferimenti del checkout registrati dal browser."
	}
	bar.Columns = []matchdayColumn{{ID: "count", Label: "Totale", Unit: "count"}, {ID: "amount", Label: "Valore", Unit: "eur"}}
	if barOpens > 0 {
		bar.Rows = append(bar.Rows, matchdayRow{Label: "Aperture menu osservate", Values: map[string]*float64{"count": matchdayOperationsValue(barOpens)}})
	}
	if f.BarRevenue != nil {
		bar.Rows = append(bar.Rows, matchdayRow{Label: "Valore lordo ordini pagati", Values: map[string]*float64{"amount": f.BarRevenue}})
	}
	matchdayPut(f, bar)
	applyMatchdayMerch(f)
}

func matchdayShopManual(f *matchdayFacts, code, name, description, formula, unit, table string) matchdayMetric {
	field := f.Manual[code]
	if field.Value == nil {
		metric := matchdayUnavailable(code, name, description, formula, unit, "missing", "Serve un dato esterno riferito alla partita.", table)
		metric.Missing = []string{code}
		return metric
	}
	return matchdayMetric{Code: code, Name: name, Description: description, Formula: formula, Unit: unit,
		Source: "manual", Status: "available", Value: field.Value, Sources: []string{"Integrazione amministratore"}}
}

func matchdayOperationsValue(count int) *float64 {
	value := float64(count)
	return &value
}

func calculateMatchdayChat(f *matchdayFacts) {
	const settings = "event_chat_settings.event_id, enabled"
	const messages = "event_chat_messages.event_id, message_type, author_user_id, moderation_status"
	const reports = "event_chat_reports.event_id, message_id, review_status"
	const reactions = "event_chat_reactions.event_id, message_id, fan_id"
	const openSignals = "engagement_report_signals.name, fan_id, device_key, session_id"
	const openRecords = "event_chat_opens.event_id, device_key, fan_id"
	const repliesSource = "event_chat_replies.event_id, message_id, reply_to_id"
	const blockedSource = "event_chat_blocked_attempts.event_id, attempts"
	const chatMissing = "Le aperture non sono state osservate nella telemetria dell'evento."
	const deletedAuthors = "Alcuni messaggi hanno perso il riferimento al profilo dell'autore."
	chat := f.Operations.chat
	openers := map[string]bool{}
	for _, opener := range chat.openIdentities {
		if identity := f.Resolve(opener); identity != "" {
			openers[identity] = true
		}
	}
	for _, signal := range f.Signals {
		if signal.Name == "match_chat_opened" {
			identity := f.Resolve(matchdaySignalIdentity(signal))
			if identity != "" {
				openers[identity] = true
			}
		}
	}
	active := chat.enabled || chat.everEnabled || chat.fans+chat.official+chat.bots > 0 || len(openers) > 0 || chat.openWithoutIdentity
	put := func(metrics ...matchdayMetric) {
		if !active {
			for i := range metrics {
				metrics[i].Value, metrics[i].Numerator, metrics[i].Denominator = nil, nil, nil
				metrics[i].Status, metrics[i].Reason = "not_applicable", "Funzione non attiva"
				metrics[i].Sources = append(metrics[i].Sources, settings)
			}
		}
		matchdayPut(f, metrics...)
	}
	opens := matchdayUnavailable("chat_openers", "Utenti che hanno aperto la chat", "Identità distinte osservate all'apertura della chat.", "Identità distinte con match_chat_opened", "count", "not_measurable", chatMissing, openSignals)
	if chat.measured || len(openers) > 0 {
		opens = matchdayCount(opens.Code, opens.Name, opens.Description, "Identità distinte che hanno aperto la chat", float64(len(openers)), openRecords, openSignals)
		if chat.openWithoutIdentity {
			opens.Status, opens.Reason = "partial", "Alcune aperture sono prive di un identificativo pseudonimo."
		}
	}
	replies := matchdayUnavailable("chat_replies", "Risposte tra tifosi", "Messaggi che rispondono a un altro messaggio.", "Conteggio risposte con messaggio padre", "count", "not_measurable", "La registrazione delle risposte non era attiva per questa partita.", repliesSource)
	blocked := matchdayUnavailable("chat_blocked", "Messaggi bloccati", "Tentativi di invio respinti dalla moderazione o dalle regole di chat.", "Conteggio dei tentativi bloccati", "count", "not_measurable", "La registrazione dei tentativi respinti non era attiva per questa partita.", blockedSource)
	if chat.measured || chat.replies > 0 {
		replies = matchdayCount(replies.Code, replies.Name, replies.Description, replies.Formula, float64(chat.replies), repliesSource)
	}
	if chat.measured || chat.blocked > 0 {
		blocked = matchdayCount(blocked.Code, blocked.Name, blocked.Description, blocked.Formula, float64(chat.blocked), blockedSource)
	}
	if !chat.complete {
		const reason = "La raccolta non è documentata dalla prima configurazione della chat per la partita."
		for _, metric := range []*matchdayMetric{&opens, &replies, &blocked} {
			if metric.Value != nil && metric.Status == "available" {
				metric.Status, metric.Reason = "partial", reason
			}
		}
	}
	authors := matchdayCount("chat_participants", "Partecipanti unici in chat", "Tifosi con almeno un messaggio salvato nella chat, inclusi quelli poi rimossi.", "COUNT(DISTINCT autore dei messaggi fan)", float64(len(chat.authors)), messages)
	reachedAuthors := 0
	for author := range chat.authors {
		if f.Reach[f.Resolve(matchdayIdentity{FanID: author})] {
			reachedAuthors++
		}
	}
	adoption := matchdayRatio("chat_adoption", "Adozione chat", "Tifosi raggiunti con almeno un messaggio nella chat.", "Autori tra i raggiunti / utenti unici ABX × 100", "percent", float64(reachedAuthors), float64(len(f.Reach)), messages)
	average := matchdayRatio("chat_average_messages", "Messaggi medi per partecipante", "Media di messaggi dei tifosi con autore identificabile.", "Messaggi con autore / autori unici", "ratio", float64(chat.fans-chat.lostAuthors), float64(len(chat.authors)), messages)
	if chat.lostAuthors > 0 {
		for _, metric := range []*matchdayMetric{&authors, &adoption, &average} {
			if metric.Value != nil {
				metric.Status, metric.Reason = "partial", deletedAuthors
			}
		}
	}
	put(opens, authors, adoption, average,
		matchdayCount("chat_total_messages", "Messaggi umani complessivi", "Messaggi dei tifosi e messaggi ufficiali, inclusi quelli rimossi.", "Messaggi fan + messaggi ufficiali", float64(chat.fans+chat.official), messages),
		matchdayCount("chat_bot_messages", "Messaggi automatici", "Messaggi automatici distinti dai messaggi umani.", "Conteggio messaggi ai", float64(chat.bots), messages),
		matchdayCount("chat_fan_reactions", "Reazioni tra tifosi", "Reazioni tuttora registrate da un tifoso al messaggio di un altro.", "Reazioni a messaggi fan, escluse le autoreazioni", float64(chat.fanReactions), messages, reactions),
		replies,
		matchdayCount("chat_reported", "Messaggi segnalati", "Messaggi dei tifosi con almeno una segnalazione.", "Messaggi fan distinti con almeno un report", float64(chat.reported), messages, reports),
		matchdayCount("chat_removed", "Messaggi rimossi", "Messaggi dei tifosi rimossi da un moderatore.", "Conteggio messaggi fan con stato removed", float64(chat.removed), messages),
		blocked,
		matchdayRatio("chat_moderation_rate", "Tasso di moderazione", "Messaggi dei tifosi rimossi o con segnalazione esaminata.", "Messaggi fan rimossi o esaminati / messaggi fan × 100", "percent", float64(chat.moderated), float64(chat.fans), messages, reports))
}

func calculateMatchdayTechnology(f *matchdayFacts) {
	const signals = "engagement_report_signals.event_id, signal_id, name, metadata.reason"
	const bar = "bar_orders.organization_id, stripe_reference, payment_status, order_status"
	const attribution = "tracking_events.event_id, event_name, metadata_json.session_id"
	const requests = "Non esiste una serie storica delle richieste API associata alla partita."
	matchdayPut(f,
		matchdayUnavailable("tech_uptime", "Disponibilità del servizio", "Percentuale del tempo dell'evento in cui il servizio risultava disponibile.", "Tempo disponibile / durata osservata × 100", "percent", "not_measurable", "Le sonde esistenti non conservano osservazioni di disponibilità per partita."),
		matchdayUnavailable("tech_api_p95", "Latenza API p95", "95º percentile della durata delle richieste API della partita.", "95º percentile delle latenze API", "milliseconds", "not_measurable", requests),
		matchdayUnavailable("tech_request_error_rate", "Tasso di errore delle richieste", "Richieste API della partita con esito di errore.", "Richieste in errore / richieste totali × 100", "percent", "not_measurable", requests),
		matchdayUnavailable("tech_anomalies", "Anomalie e tentativi sospetti", "Tentativi rilevati da controlli di sicurezza o antifrode.", "Conteggio degli eventi di sicurezza persistiti", "count", "not_measurable", "I limiti di frequenza non producono un registro di anomalie associato alla partita."))
	var submitted, completed, failed, limited int
	failures := map[string]map[string]int{}
	for _, signal := range f.Signals {
		switch signal.Name {
		case "vote_submitted":
			submitted++
		case "vote_completed":
			completed++
		case "vote_failed":
			failed++
			if signal.Metadata["reason"] == "rate_limited" {
				limited++
			}
		}
		if label := matchdayFailureLabel(signal.Name); label != "" {
			reason, _ := signal.Metadata["reason"].(string)
			if failures[label] == nil {
				failures[label] = map[string]int{}
			}
			failures[label][matchdayFailureReason(reason)]++
		}
	}
	vote := matchdayUnavailable("tech_vote_success", "Votazioni completate correttamente", "Esiti positivi fra i tentativi con esito osservato nella home tifosi.", "vote_completed / (vote_completed + vote_failed) × 100", "percent", "not_measurable", "La telemetria non contiene esiti di voto della home tifosi per la partita.", signals)
	if completed+failed > 0 {
		vote = matchdayRatio(vote.Code, vote.Name, vote.Description, vote.Formula, vote.Unit, float64(completed), float64(completed+failed), signals)
		vote.Status, vote.Reason = "partial", "Sono inclusi gli esiti ricevuti dalla home tifosi. Il totale degli invii osservati ha copertura indipendente."
	}
	if submitted+completed+failed > 0 {
		vote.Columns = []matchdayColumn{{ID: "count", Label: "Eventi osservati", Unit: "count"}}
		vote.Rows = []matchdayRow{
			{Label: "Invii", Values: map[string]*float64{"count": matchdayOperationsValue(submitted)}},
			{Label: "Completati", Values: map[string]*float64{"count": matchdayOperationsValue(completed)}},
			{Label: "Falliti", Values: map[string]*float64{"count": matchdayOperationsValue(failed)}},
			{Label: "Voti respinti per limite di frequenza", Values: map[string]*float64{"count": matchdayOperationsValue(limited)}},
		}
	}
	payment := matchdayUnavailable("tech_payment_success", "Pagamenti riusciti", "Esiti dei pagamenti digitali attribuiti alla partita.", "Pagamenti riusciti / pagamenti con esito definitivo × 100", "percent", "not_measurable", "Nessun esito definitivo di pagamento è attribuibile alla partita.")
	if merch := f.Operations.merch; merch.paymentResolved > 0 {
		const orders = "merch_match_orders.event_id, merch_orders.mode, payment_status"
		payment = matchdayRatio(payment.Code, payment.Name, "Esiti definitivi degli ordini online dello shop attribuiti alla partita.",
			"Ordini pagati (anche se poi rimborsati) / ordini con esito di pagamento definitivo × 100", payment.Unit,
			float64(merch.paymentPaid), float64(merch.paymentResolved), orders)
		payment.coverage = "merch-online"
	} else if f.Operations.barClosed > 0 && !f.Operations.merch.observedPayment {
		payment = matchdayRatio(payment.Code, payment.Name, "Esiti del checkout bar digitale attribuiti alla partita.", "Ordini bar pagati / checkout bar con esito definitivo × 100", payment.Unit, float64(f.Operations.barPaid), float64(f.Operations.barClosed), bar, attribution)
		payment.Status, payment.Reason = "partial", "Il campione comprende solo checkout bar collegabili alla partita. I checkout ancora in corso sono esclusi."
	}
	issues := matchdayUnavailable("tech_application_issues", "Errori e criticità osservati", "Segnali di errore provenienti dalla home tifosi.", "Conteggio diagnostiche client selezionate, per categoria e motivo", "count", "not_measurable", "La telemetria non contiene diagnostiche di errore per la partita.", signals)
	if len(failures) > 0 {
		var total int
		for feature, reasons := range failures {
			for reason, count := range reasons {
				total += count
				issues.Rows = append(issues.Rows, matchdayRow{Label: reason, Group: feature, Values: map[string]*float64{"count": matchdayOperationsValue(count)}})
			}
		}
		sort.Slice(issues.Rows, func(i, j int) bool {
			if issues.Rows[i].Group != issues.Rows[j].Group {
				return issues.Rows[i].Group < issues.Rows[j].Group
			}
			return issues.Rows[i].Label < issues.Rows[j].Label
		})
		issues.Status, issues.Source, issues.Value = "partial", "db", matchdayOperationsValue(total)
		issues.Reason = "Diagnostiche ricevute dal browser. La copertura del servizio richiederebbe misure lato server."
		issues.Columns = []matchdayColumn{{ID: "count", Label: "Eventi osservati", Unit: "count"}}
	}
	matchdayPut(f, vote, payment, issues)
	calculateMatchdayServerTechnology(f)
}

func matchdayFailureLabel(name string) string {
	return map[string]string{
		"home_load_failed": "Home tifosi", "vote_failed": "Voto MVP", "game_reward_failed": "Premio gioco",
		"registration_failed": "Registrazione", "login_failed": "Accesso", "redemption_failed": "Riscatto virtuale",
		"frame_equip_failed": "Cornice", "feedback_failed": "Feedback", "leaderboard_failed": "Classifica",
		"profile_failed": "Profilo", "profile_update_failed": "Aggiornamento profilo", "wallet_failed": "Portafoglio",
		"vote_reward_failed": "Premio voto",
	}[name]
}

func matchdayFailureReason(reason string) string {
	if label := (map[string]string{
		"network": "Rete", "timeout": "Timeout", "server_error": "Errore server", "validation": "Validazione",
		"rate_limited": "Limite di frequenza", "load_failed": "Caricamento", "save_failed": "Salvataggio",
		"request_failed": "Richiesta fallita", "session_expired": "Sessione scaduta", "not_allowed": "Accesso negato",
		"conflict": "Conflitto", "not_found": "Risorsa assente", "unavailable": "Funzione indisponibile",
	})[reason]; label != "" {
		return label
	}
	return "Altro o non specificato"
}
