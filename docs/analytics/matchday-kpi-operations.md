# Matchday KPI Analytics — Audit shop, chat e tecnologia

Disponibilità verificata sullo schema e sui punti di raccolta del progetto. **DB** identifica dati attribuiti direttamente alla partita. **Parziale** indica una popolazione osservata o un perimetro più ristretto. **Mancante** indica un aggregato integrabile. **Non misurabile** indica che mancano l’attribuzione o le osservazioni necessarie per la formula.

## 8. Merchandising e shop

| Metrica | Tabelle e colonne | Formula | Disponibilità e limiti |
| --- | --- | --- | --- |
| Visitatori unici shop virtuale | `engagement_report_signals.event_id`, `name`, `fan_id`, `device_key`, `session_id` | Identità canoniche distinte con `name='shop_view'` | DB con aperture osservate. Copertura della sezione riscatti della home tifosi. |
| Aperture shop virtuale | `engagement_report_signals.event_id`, `signal_id`, `name` | Conteggio dei segnali distinti `shop_view` | DB con aperture osservate. Le vecchie interfacce possono avere una copertura diversa. |
| Aperture e visitatori unici shop merchandising | `merch_analytics_events.organization_id`, `event_id`, `type`, `product_id` esaminate | Aperture e identità uniche con apertura attribuita alla partita | Non misurabili. `merch_analytics_events.event_id` è una chiave di idempotenza per il singolo segnale. Le righe sono attribuite alla società e non contengono un’identità visitatore. |
| Conversione acquisti merchandising | `merch_orders.organization_id`, `fan_id`, `payment_status` e `merch_analytics_events` esaminate | Acquirenti unici / visitatori unici dello stesso shop × 100 | Non misurabile. Servono l’attribuzione matchday e un’identità confrontabile sui due lati del funnel. |
| Acquisti confermati | `merch_orders.id`, `organization_id`, `request_id`, `payment_status`, `fulfillment_status`, `stripe_session`, `stripe_payment_intent` esaminate | Conteggio di ordini pagati distinti ed eleggibili | Mancante. Integrabile con `shop_paid_orders` da un registro riferito alla partita. Il perimetro esclude pagamenti falliti, annullati, rimborsati e duplicati. |
| Valore lordo degli ordini confermati | `merch_orders.total_cents`, `currency`, `payment_status`, `fulfillment_status` esaminate | Somma degli importi lordi degli stessi ordini eleggibili | Mancante. Integrabile in EUR con `shop_gmv`. Le prenotazioni non pagate sono separate. |
| Valore medio ordine | Valori integrativi `shop_gmv` e `shop_paid_orders` | GMV / acquisti confermati | Manuale derivato quando sono disponibili entrambi i dati. Denominatore zero senza valore calcolato. |
| Prenotazioni confermate | `merch_orders.organization_id`, `mode`, `payment_status`, `fulfillment_status`, `created_at` esaminate | Conteggio di prenotazioni distinte, attribuite e confermate | Mancante. Integrabile con `shop_reservations`. I totali sono presentati separatamente dagli acquisti anche nella dashboard Executive. |
| Rimborsi e prenotazioni non pagate | `merch_payment_records.organization_id`, `order_id`, `status`, `amount_cents` e `merch_orders.mode`, `payment_status`, `fulfillment_status` esaminate | Somma dei rimborsi confermati e conteggio delle prenotazioni confermate non pagate | Non misurabili per partita. La contabilità Stripe e offline conserva il dettaglio a livello di società. |
| Prodotti più visualizzati o acquistati | `engagement_report_signals.name`, `metadata.item_key`, `metadata.placement` e `fan_wallet_movements.event_id`, `organization_id`, `source`, `delta` | Visualizzazioni `shop_item_viewed` nella sezione shop e riscatti da movimenti negativi `shop:club_frame` o `shop:match_card`, per oggetto | Parziale. Copre i prodotti virtuali. Le visite dalla vetrina del profilo sono escluse. La classifica commerciale rimane a livello di società. |
| Ordini pagati al bar e valore lordo | `tracking_events.event_id`, `organization_id`, `event_name`, `metadata_json.session_id` e `bar_orders.organization_id`, `stripe_reference`, `payment_status`, `order_status`, `total_cents` | Conteggio e somma degli ordini con `payment_status='paid'`, stato ordine non annullato e riferimento Stripe osservato in una sola partita della stessa società | Parziale. Il collegamento deriva da `bar.checkout_redirected` o `bar.order_completed`. Riferimenti associati a più partite sono esclusi. L’importo lordo è espresso in EUR. `bar.menu_opened` misura le aperture osservate del menu. Il flusso bar non conserva un registro di rimborsi per partita. |

Gli ordini bar pagati e verificati contribuiscono al totale delle azioni significative. Per la deduplicazione dei tifosi si usa il dispositivo osservato al redirect quando è univoco per quell’ordine. Gli ordini e i riscatti virtuali misurano operazioni distinte. Un valore manuale aggregato per le vendite o le prenotazioni non identifica i singoli tifosi e quindi non entra nel FAR.

## 9. Community e chat live

`event_chat_settings.event_id` e `enabled` determinano l’abilitazione corrente. Quando la chat è disabilitata e non esistono messaggi o aperture osservate, i KPI mostrano «Funzione non attiva». Le impostazioni mantengono l’ultimo stato. L’attività salvata resta consultabile anche dopo una disattivazione.

| Metrica | Tabelle e colonne | Formula | Disponibilità e limiti |
| --- | --- | --- | --- |
| Utenti che hanno aperto la chat | `engagement_report_signals.event_id`, `name`, `fan_id`, `device_key`, `session_id` | Identità canoniche distinte con `match_chat_opened` | DB con aperture osservate. L’assenza di segnali può dipendere dalla copertura del client. |
| Partecipanti unici | `event_chat_messages.event_id`, `message_type`, `author_user_id` e `fan_profiles.id`, `organization_id` | Profili distinti con almeno un messaggio `fan` salvato | DB, parziale se qualche autore è stato cancellato. Sono inclusi i messaggi rimossi dopo il salvataggio. |
| Adozione chat | Stessa sorgente partecipanti più identità del pubblico Matchday | Autori fan tra gli utenti raggiunti / utenti raggiunti × 100 | DB con denominatore disponibile. |
| Messaggi complessivi e automatici | `event_chat_messages.event_id`, `message_type` | Somma dei tipi `fan` e `official`. Conteggio separato di `ai`. | DB. I messaggi ufficiali e automatici sono esclusi dai KPI di partecipazione fan. |
| Messaggi medi per partecipante | `event_chat_messages.event_id`, `message_type`, `author_user_id` | Messaggi fan con autore identificabile / autori fan distinti | DB con denominatore positivo, parziale se qualche autore è stato cancellato. |
| Reazioni tra tifosi | `event_chat_reactions.event_id`, `message_id`, `fan_id` e `event_chat_messages.message_type`, `author_user_id` | Reazioni attualmente salvate a messaggi fan, escluse le autoreazioni | DB. Una reazione ritirata viene cancellata. |
| Risposte tra tifosi | `event_chat_messages` esaminata | Conteggio di messaggi con messaggio padre | Non misurabile. La chat non conserva riferimenti di risposta. |
| Messaggi segnalati | `event_chat_reports.event_id`, `message_id` e `event_chat_messages.message_type` | Messaggi fan distinti con almeno una segnalazione | DB. Più segnalazioni sullo stesso messaggio sono contate una volta. |
| Messaggi rimossi | `event_chat_messages.event_id`, `message_type`, `moderation_status` | Conteggio di messaggi fan con `moderation_status='removed'` | DB. |
| Messaggi bloccati | `event_chat_mutes.event_id`, `fan_id`, `muted_until` esaminate | Conteggio dei tentativi di invio bloccati | Non misurabile. I provvedimenti sui profili sono registrati. I tentativi di invio respinti non vengono conservati. |
| Tasso di moderazione | `event_chat_messages.event_id`, `message_type`, `moderation_status` e `event_chat_reports.event_id`, `message_id`, `review_status` | Messaggi fan distinti rimossi o con almeno una segnalazione esaminata / messaggi fan salvati × 100 | DB con denominatore positivo. I messaggi contemporaneamente esaminati e rimossi entrano una volta. |

## 10. Tecnologia e affidabilità

| Metrica | Tabelle e colonne | Formula | Disponibilità e limiti |
| --- | --- | --- | --- |
| Disponibilità del servizio | Sonda `cmd/healthcheck/healthcheck.go` e logging di `cmd/webapi/main.go` esaminati | Tempo disponibile / durata osservata × 100 | Non misurabile. Nessuna serie storica di disponibilità è interrogabile per partita. |
| Latenza API p95 | Logging server e metadati client esaminati | 95º percentile della latenza delle richieste API attribuite alla partita | Non misurabile. Le durate di azioni client non rappresentano le latenze del servizio. |
| Error rate delle richieste | Logging server e metadati client esaminati | Richieste API in errore / richieste API attribuite alla partita × 100 | Non misurabile. Manca il denominatore completo. |
| Votazioni completate correttamente | `engagement_report_signals.event_id`, `signal_id`, `name`, `metadata.reason` | `vote_completed` / (`vote_completed` + `vote_failed`) × 100 | Parziale. Misura gli esiti ricevuti dalla home tifosi. `vote_submitted` e `vote_failed` con `reason='rate_limited'` sono esposti come conteggi osservati. |
| Pagamenti riusciti | `bar_orders.organization_id`, `stripe_reference`, `payment_status`, `order_status` e `tracking_events.event_id`, `event_name`, `metadata_json.session_id` | Ordini bar pagati e non annullati / checkout bar attribuiti conclusi come pagati o scaduti × 100 | Parziale. Campione di checkout attribuibili. Un checkout scaduto può non aver raggiunto l’invio del pagamento. Gli ordini merchandising non contengono un riferimento alla partita. |
| Anomalie e tentativi sospetti | Contatori in memoria di `service/api/vote-rate-limit.go` e diagnostiche client esaminati | Conteggio di eventi di sicurezza verificati e persistiti | Non misurabile. Il rifiuto per frequenza è una diagnostica osservabile senza classificazione antifrode. |
| Errori e criticità applicative | `engagement_report_signals.event_id`, `signal_id`, `name`, `metadata.reason` | Conteggio delle diagnostiche di errore ricevute e classificate per area e causa | Parziale. Copertura client e lista di motivi validata. Non è una misura dell’error rate API. |

## Instrumentazione necessaria per estendere la copertura

- Propagazione dell’identificativo partita dall’ingresso nello shop commerciale alle aperture, alle visualizzazioni, ai checkout, agli ordini e alle prenotazioni. Un identificativo visitatore pseudonimo e coerente consente il funnel, mentre il legame server con l’ordine consente di includere incassi, storni e rimborsi.
- Associazione server tra partita e ordini del bar, con esiti del pagamento e del rimborso distinguibili dagli esiti del checkout.
- Telemetria chat con aperture misurate, relazioni di risposta, eventi di invio bloccato e storico di attivazione, quando servono misure complete di quelle funzioni.
- Serie temporale di sonde di disponibilità e metriche di richieste server con esiti, latenze e contesto partita. Tracciamento coerente degli esiti voto e pagamento lato server e registro di anomalie verificabili per partita.
