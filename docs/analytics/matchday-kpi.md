# Audit Matchday KPI Analytics

Il pannello si apre da **Storico eventi → KPI Matchday** nelle due interfacce di amministrazione della società. Il calcolo legge le fonti operative e analitiche SQLite esistenti. Le query aggregano gli eventi confrontati per sorgente e le risposte contengono solo indicatori, etichette e dati integrativi amministrativi.

**Disponibile** indica una misura prodotta dalla fonte indicata, anche quando vale zero. **Parziale** indica che la misura riguarda una coorte, un canale o un sottoinsieme osservato. **Dato mancante** indica un input o uno storico assente. **Non misurabile** indica che manca il tracciamento richiesto. **Non applicabile** include rapporti con denominatore zero e funzionalità disabilitate. La provenienza è indicata separatamente come DB, Manuale o Misto.

Il confronto seleziona gli eventi conclusi della stessa società (`events.organization_id,is_concluded,start_datetime`), eventualmente filtrati per campionato (`organizations.team_id`, `teams.championship`, `events.team1_id,team2_id`). Il campionato è quello configurato oggi per la squadra, perché l'archivio Live non conserva il campionato storico né un identificativo stagione. La stagione è convenzionalmente da luglio a giugno. Il valore precedente usa la partita immediatamente precedente nel perimetro. La media usa le partite selezionate della stessa stagione con fonti e copertura compatibili, con la partita corrente esclusa. I delta assoluti tra percentuali si leggono in punti percentuali. La variazione relativa divide il delta per il riferimento e resta vuota se il riferimento vale zero.

## Identità e copertura

- L'identità esplicita di un profilo è accettata solo se il profilo appartiene alla società ed è attivo per la registrazione (`fan_profiles.id,organization_id,registration_pending`). `engagement_report_signals.fan_id` viene associato dal backend a una sessione autenticata. Il campo `fan_id` inviato dal browser nella telemetria legacy non certifica un profilo.
- L'hash per società degli identificativi dei dispositivi operativi coincide con `engagement_report_signals.device_key`. Un dispositivo viene associato a un profilo quando c'è un solo collegamento affidabile. Un'associazione locale ha precedenza sulla storia globale. Un dispositivo condiviso senza una risoluzione univoca resta distinto. Una sessione priva di profilo e dispositivo vale solo per il proprio evento. Nessun numero di telefono, token di sessione o identificativo di tifoso viene incluso nella risposta.
- Per i nuovi utenti si usa la prima partita osservata anche al di fuori della finestra dei grafici. La scansione storica usa segnali di ingresso e registrazione, sessioni e permanenze legacy e voti. Un'identità che ha partecipato in passato soltanto tramite una fonte storica senza ingressi osservati può apparire come nuova.

## 1. Direzione e performance generale

| KPI | Tabelle e colonne | Formula, stato e integrazione |
| --- | --- | --- |
| Utenti unici raggiunti | `engagement_report_signals.event_id,device_key,session_id,fan_id`, `tracking_events.event_id,device_id,session_id`, `page_engagements.event_id,device_id,duration_seconds`, `votes.event_id,user_id,device_id` e identità dalle operazioni descritte nei reparti | Cardinalità dell'unione deduplicata. Parziale per copertura. Mancante quando l'evento è privo di identità osservate. |
| Penetrazione sul pubblico | Fonti della reach, `engagement_report_manual_sources.field_key,value,source_description` | Utenti raggiunti / spettatori presenti × 100. Misto e parziale. Mancante senza presenze ufficiali integrabili come `attendees`. Il numeratore può includere pubblico remoto. |
| Tasso di attivazione e FAR | Reach, `votes`, round con avvio, `event_feedback` e segnali feedback, `event_chat_messages.message_type,author_user_id`, click sponsor, `tracking_events.metadata_json` e `bar_orders.stripe_reference,payment_status` | Identità con almeno un'azione significativa / identità raggiunte × 100. Parziale. Le azioni senza identità e gli aggregati esterni sono esclusi dal numeratore deduplicato. |
| Engagement per tifoso | Stesse fonti del FAR | Azioni con identità / tifosi con almeno un'azione significativa. Parziale. |
| Retention interpartita | Identità persistenti delle due partite, `events.start_datetime` | Intersezione / coorte della precedente × 100. Parziale con copertura comparabile. Storico insufficiente senza una coorte utilizzabile. |
| Ricavi generati | `bar_orders`, `tracking_events`, `merch_match_orders`, `merch_orders`, `merch_payment_records`, `matchday_kpi_manual_revisions` | Incassi bar attribuiti, ricavi merchandising attribuiti al netto dei rimborsi e incassi offline dichiarati. Il dato shop può essere integrato per gli eventi precedenti privi di raccolta automatica. |

## 2. Marketing e acquisizione

Le sessioni di ingresso provengono da `tracking_events.event_name='session.started'` e da `engagement_report_signals.name='page_view'`. Per attribuirle si usa il primo ingresso della sessione per flusso. Su `/new` il browser converte UTM, `source` e referrer in una categoria ammessa (`metadata.entry_channel`). La nuova raccolta conserva la categoria, senza salvare URL, referrer o parametri grezzi. Il campo `metadata.source` resta disponibile per la compatibilità con il report precedente.

| KPI | Tabelle e colonne | Formula e disponibilità |
| --- | --- | --- |
| Accessi e utenti per canale | `tracking_events.session_id,device_id,occurred_at,metadata_json`, `engagement_report_signals.session_id,device_key,fan_id,occurred_at,metadata.entry_channel` | Sessioni d'ingresso distinte e identità distinte per canale. Disponibile per gli ingressi con la nuova raccolta, parziale se include ingressi legacy. |
| Nuovi utenti per canale | Colonne degli ingressi sopra, identità persistenti e cronologia del club. | Prima partita osservata per identità, attribuita al primo ingresso tracciato. La nuova raccolta usa le identità confrontabili con lo storico disponibile. |
| Attivazione per canale | Colonne degli ingressi sopra e fonti delle azioni significative (FAR). | Utenti del canale con almeno un'azione significativa / utenti del canale × 100. La copertura dipende dalla raccolta degli ingressi e delle azioni. |
| Conversione QR | `matchday_qr_arrivals.event_id,organization_id,redirect_id,channel,opened_at,landed_at`. Integrazione `qr_scans` per campagne esterne. | Arrivi confermati / aperture misurate dei link QR gestiti × 100. Un token casuale lega ogni apertura a un solo arrivo. Se la campagna usa un contatore esterno resta disponibile il dato manuale. |
| Costo per acquisizione | Fonti dei nuovi utenti e integrazione `marketing_spend`. | Spesa promozionale / nuove identità osservate. Mancante senza spesa esterna. Parziale con entrambi i dati. |
| Distribuzione del traffico | Colonne degli ingressi sopra. | Accessi del canale / accessi osservati × 100. Il valore in evidenza è la quota maggiore. Parziale. |
| Variazione per canale | Colonne degli ingressi sopra per le due partite comparabili. | Accessi correnti − precedenti e differenza / accessi precedenti × 100. Il dettaglio include anche la variazione degli utenti unici. Parziale con due partite misurate, storico insufficiente negli altri casi. |

Per misurare una campagna QR, il redirect gestito deve puntare a `/<slug>/new` sullo stesso dominio durante una partita attiva. Per distinguere LED e giornalino si possono configurare `utm_source=led` e `utm_source=giornalino`. La misura riguarda l'apertura del link. Una lettura del codice senza apertura e le scansioni di QR diretti senza redirect richiedono un contatore esterno.

## 3. Fan engagement e MVP

| KPI | Tabelle e colonne | Formula e disponibilità |
| --- | --- | --- |
| Votanti unici MVP | `votes.event_id,user_id,device_id`, `fan_profiles.id,organization_id` | Identità distinte con voti accettati. Disponibile, oppure mancante quando ci sono voti senza identità deduplicabile. |
| Conversione MVP | Fonti dei votanti e della reach | Votanti unici / identità raggiunte × 100. Parziale per copertura della reach. |
| Engagement multifunzione | Fonti del FAR | Identità con almeno due tipi di azione / reach × 100. Parziale. Acquisti e prenotazioni appartengono alla stessa funzione shop. |
| Interazioni significative | Fonti del FAR | Azioni osservate, anche quando manca l'identità dell'autore. Parziale. |
| Interazioni medie per tifoso attivo | Fonti del FAR | Azioni attribuibili / tifosi attivi. Parziale. |
| Durata media delle sessioni attive | `engagement_report_signals.name,session_id,device_key,occurred_at,metadata` (`active_time`, `seconds`, `started_at`, `ended_at`) | Unione degli intervalli attivi per dispositivo/sessione, con ciascun intervallo limitato a 15 secondi, poi media per sessione misurata. Parziale. Non misurabile senza intervalli. |
| Permanenza media legacy | `page_engagements.event_id,duration_seconds` | Media delle durate registrate. Parziale perché la fonte conserva osservazioni senza intervalli attivi e identificativi di sessione. |
| Distribuzione dei voti | `votes.event_id,player_id`, `players.id,first_name,last_name,team_id` | Conteggio per giocatore e quota sul totale dei voti accettati. Disponibile. |
| MVP Light | `club_mvp_votes.match_id,voter_hash,player_id`, `club_matches.id,team_id,site_id` | Non applicabile in questo archivio. Il calendario ClubHub usa un identificativo partita distinto e manca una relazione certificata verso `events.id`. |

## 4. Gamification e premi

Fonti dei round, tutte filtrate per evento e società. I round con chiave comune vengono uniti per gioco e identità. I match Tap a due giocatori condividono `match_id` tra browser e server e contano una partecipazione per giocatore. Il risultato salvato distingue le partite completate dagli abbandoni. Le ricevute legacy prive di chiave round valgono come riscontro in assenza di un round più preciso per la stessa identità e gioco.

| Fonte | Colonne e prova registrata |
| --- | --- |
| `engagement_report_signals`, `tracking_events` | Nome del segnale, `event_id`, chiavi identità e `metadata.game_id,round_id,mode` o gli equivalenti in `metadata_json`. Avvio, completamento e abbandono distinti. |
| `basketball_shots`, `volleyball_rounds` | `event_id,organization_id,device_id,fan_id,shot_id` o `round_id,game_id`, `status`. Soltanto tiri risolti e round con esito `success` o `miss`. |
| `playmaker_sessions`, `crossover_sessions` | `event_id,organization_id,device_id,fan_id,session_id`; `crossover_sessions.next_duel` certifica il completamento quando i duelli richiesti sono risolti. |
| `fan_minigame_rewards`, `fan_journey_rewards` | `event_id,organization_id,device_id,fan_id,reward_key,coins` e `request_id` nelle ricevute moderne. Accrediti positivi dei giochi supportati. |
| `fan_minigame_quiz_answers`, `fan_journey_quiz_answers` | `event_id,organization_id,device_id` e `request_id` quando disponibile. Una risposta dimostra che il round è iniziato. |
| `reaction_tests`, `branded_game_participations`, `tap_live_matches` | `reaction_tests.id,event_id,device_id,is_valid`; `branded_game_participations.id,event_id,device_id,user_id,completed`; `tap_live_matches.event_id,organization_id,fan1_id,fan2_id,match_id,game_id,started_at,status,abandoned_by_fan_id`. |

| KPI | Formula e disponibilità |
| --- | --- |
| Giocatori unici | Identità distinte con evidenza di gioco. Disponibile con identità e round correlabili. |
| Partite avviate | Round distinti con avvio osservato o certificato da un'operazione di gioco. Disponibile con identità e round correlabili. |
| Partite completate | Round distinti con prova di completamento. Parziale quando manca un avvio correlabile o un esito finale. |
| Completion rate | Completamenti abbinati a un avvio dello stesso round / avvii × 100. Parziale quando rimangono round senza esito o senza avvio correlabile. |
| Replay rate | Giocatori con almeno due avvii / giocatori unici × 100. Disponibile con identità e round correlabili. |
| Sessioni medie | Partecipazioni individuali agli avvii / giocatori unici. Disponibile con identità e round correlabili. |
| Giochi più utilizzati | Classifica per giocatori unici, con avvii e completamenti per gioco. Parziale quando le osservazioni sono incomplete. |
| Partecipanti ai premi | Profili distinti iscritti in `fan_lottery_entries.event_id,fan_id`, con profili della stessa società. Disponibile. |
| Premi assegnati | `event_prizes.event_id,winner_user_id,winner_vote_id,winner_lottery_code`. Premi con vincitore salvato. Disponibile. |

Quando una partita è priva di evidenze di raccolta, i KPI dei round mostrano un dato mancante. Le pagine con `metadata.games_available=true` possono produrre uno zero di round osservati con copertura parziale. I giochi lato browser richiedono la consegna dei segnali di avvio e conclusione. Un dispositivo spento o una connessione persa possono lasciare un round senza esito.

## 5. Customer experience e feedback

Il flusso one-tap salva un campo alla volta e aggiorna la stessa scheda tramite `event_id,response_id`. Le nuove risposte includono la chiave pseudonima del browser e, con sessione autenticata della stessa società, il riferimento al profilo. Le schede storiche restano anonime. Il nuovo segnale di apertura porta lo stesso identificativo di scheda. Gli esiti delle nuove compilazioni vengono dalle risposte salvate.

| KPI | Tabelle e colonne | Formula e disponibilità |
| --- | --- | --- |
| Rispondenti unici | `event_feedback.report_device_key,report_fan_id`; segnali storici in `engagement_report_signals` e `tracking_events`. | Identità deduplicate con risposta salvata. Disponibile per schede attribuibili, parziale con schede anonime o segnali storici. |
| Feedback raccolti | `event_feedback.response_id,experience,team_spirit,perks_interest,mini_games_interest,report_experience,suggestion`. | Schede con almeno una risposta valida, anche parziali. Disponibile. |
| Risposte totali | Stesse colonne di `event_feedback`. | Somma dei campi validi salvati, incluso un eventuale suggerimento testuale. Disponibile. |
| Tasso di risposta | `cta_viewed` per `feedback_home`, `feedback_opened` con identificativo di scheda e prime risposte salvate. | Rispondenti associabili a un'esposizione / utenti esposti osservati × 100. Disponibile per la coorte misurata. Senza esposizioni è non misurabile. Il flag `feedback_available` indica la disponibilità della funzione. |
| Distribuzione per domanda | Campi di risposta in `event_feedback`, etichette da `events.feedback_survey_config`. | Conteggio per opzione e percentuale sul totale delle risposte della domanda. Disponibile. |
| Soddisfazione | `event_feedback.report_experience`. | Risposte `awesome` o `good` / risposte valide `awesome,good,ok,bad` × 100. Disponibile con una scala esplicita, altrimenti non misurabile. Il campo `experience` misura la facilità di voto e non entra nella formula. |
| Sondaggi iniziati | `engagement_report_signals.metadata.feedback_response_id`, `event_feedback.response_id`; segnali storici per sessione. | Schede con primo tentativo o prima risposta salvata. Disponibile per il nuovo flusso, parziale per le sessioni storiche. |
| Sondaggi completati | `event_feedback.experience,team_spirit,perks_interest,mini_games_interest,report_experience`. | Schede con risposte valide alle quattro domande configurate e, per le schede attribuite, alla valutazione complessiva. Disponibile. |
| Completamento sondaggi | Schede iniziate e risposte salvate con lo stesso `response_id`; fallback storico per sessione. | Schede con risposte obbligatorie salvate / schede iniziate × 100. Disponibile per il nuovo flusso, parziale con osservazioni storiche. |
| Temi nei suggerimenti | `event_feedback.suggestion`. | Conteggio schede con testo e categorie da parole chiave menzionate da almeno due schede. Disponibile. I suggerimenti originali non vengono restituiti. |

Con `events.show_feedback_survey=0` e nessuna attività osservata i KPI sono non applicabili. La visibilità misura l'intero questionario, con almeno metà dell'area visibile per un secondo. Una risposta prova l'esposizione anche quando precede la rilevazione della visibilità. Con sole interazioni come prova di esposizione, il tasso resta parziale.

## 6. Sponsorship e commerciale

Per il match si riusano il tracciamento commerciale e le aggregazioni di `sponsor_measured_exposures`, filtrate con `club_id,match_id`. L'impression viene registrata alla prima osservazione con area positiva. La soglia di viewability è almeno il 50% visibile per 1.000 ms continui. La nuova raccolta applica la stessa soglia ai placement geometrici e distingue il metodo HTML o geometrico. Durante le esposizioni lunghe registra campioni periodici di durata. Per le esposizioni interrotte la durata si ferma all'ultimo campione ricevuto.

| KPI | Tabelle e colonne | Formula e disponibilità |
| --- | --- | --- |
| Impression per sponsor | `sponsor_measured_exposures.sponsor_id,sponsor_name,impression,method`. | Somma di `impression` per sponsor, sia HTML sia geometrica. Disponibile con misurazione osservata. |
| Viewable impressions | `sponsor_measured_exposures.sponsor_id,method,viewable,geometry_sampled,geometry_viewable`. | Somma di `viewable` sulle esposizioni HTML e di `geometry_viewable` su quelle geometriche misurate. Parziale se alcune impression geometriche precedono l'introduzione della misura. |
| Viewability rate | `sponsor_measured_exposures.impression,method,viewable,geometry_sampled,geometry_viewable`. | Impression visibili / impression HTML o geometriche con misura della soglia × 100. |
| Reach | `sponsor_measured_exposures.audience_key,impression`, fallback `sponsor_exposures.device_id,exposure_type`. | Dispositivi distinti con impression, deduplicati anche tra sponsor nel totale. Non misurabile se mancano chiavi necessarie. La copertura legacy `seen` o `watched` è parziale e distinta nello storico. |
| Tempo visibile totale e medio | `sponsor_measured_exposures.impression,ended,visible_ms`. | Somma dei millisecondi sopra soglia / esposizioni con durata osservata per la media. Comprende gli zeri confermati dalla chiusura. Parziale se mancano chiusure. |
| Click ricevuti | `sponsor_measured_exposures.clickable,clicks`, fallback `sponsor_clicks.sponsor_id,device_id`. | Somma dei click misurati quando il placement supporta il tracking. I click legacy sono parziali e distinti nello storico. |
| CTR | `sponsor_measured_exposures.clicks,clickable,impression`. | Click / tutte le impression misurate × 100. Non misurabile quando manca il denominatore comune. |
| Conversioni attribuibili | `user_coupons.match_id,coupon_id,used_at`, `coupons.sponsor_id,organization_id`, `sponsors.id,organization_id`. | Coupon riscattati associati a match e sponsor, senza inferire acquisti. Parziale quando esistono coupon osservati. |
| Frequency | `sponsor_measured_exposures.impression,audience_key`. | Impression / dispositivi distinti raggiunti. Non misurabile senza reach. |
| Performance per posizione | `sponsor_measured_exposures.placement_id,game_id,method` con le colonne dei KPI sopra. | Stesse aggregazioni per posizione, gioco e metodo. Comprende home tifosi, conferma MVP, tabelloni SVG e Sponsor Rush quando misurati. |

I nuovi click misurati conservano `report_device_key`, la chiave pseudonima del browser nello stesso perimetro del report. Questo permette di contare anche chi ha interagito solo con uno sponsor nel FAR. Per i record precedenti resta la riconciliazione con i dispositivi osservati. Le conversioni su siti esterni richiedono una conferma della piattaforma o del partner che gestisce la destinazione.

## 7. CRM e fidelizzazione

| KPI | Tabelle e colonne | Formula e disponibilità |
| --- | --- | --- |
| Nuovi profili registrati | `fan_matchday_registrations.fan_id,event_id,organization_id,registered_at,device_key`; fallback da `fan_profiles` e `fan_registration_acceptances` | Prime registrazioni verificate salvate dal server con l'evento. Per gli eventi precedenti resta la stima parziale basata sulle attività del profilo e sul giorno locale. |
| Profili identificati | `fan_profiles.id,organization_id,registration_pending`, identità osservate | Cardinalità delle identità riconciliate con un profilo. Parziale. |
| Identification rate | Fonti dei profili e della reach | Profili identificati / reach × 100. Parziale. |
| Nuovi consensi marketing validi | Politica del profilo `/new`. | Non applicabile. Il profilo non raccoglie consensi per SMS promozionali. |
| Opt-in rate | Politica del profilo `/new`; `fan_profiles.marketing_consent,phone_verified_at` per gli eventi legacy. | Non applicabile su `/new`. Per gli eventi legacy il valore parziale fotografa lo stato attuale dei nuovi profili osservati. |
| Tifosi ritornati | Fonti della reach e partite precedenti | Intersezione delle identità persistenti per due partite comparabili. Parziale. |
| Retention delle coorti | Stesse fonti della retention Direzione | Intersezione / coorte della partita precedente × 100, ricalcolata per ogni punto dello storico. Parziale. |
| Evoluzione fanbase identificata | Fonti dei profili identificati | Serie di profili distinti osservati per evento. Parziale. |

Presenze ufficiali, spesa promozionale e incassi offline rimangono fonti manuali.

## 8–10. Merchandising, chat e tecnologia

Le tabelle, formule e coperture per ciascun indicatore operativo sono nel [dettaglio di merchandising, chat e tecnologia](matchday-kpi-operations.md).

## Integrazioni e revisioni

Le presenze ufficiali usano il dato già condiviso con il report di engagement (`engagement_report_manual_sources`, `engagement_report_source_history`). Spesa, scansioni, ricavi offline e aggregati shop sono versionati in `matchday_kpi_manual_revisions` con società, evento, codice, valore, unità, fonte, data e amministratore. Il salvataggio e il calcolo condividono una transazione. Una revisione obsoleta produce un conflitto e preserva l'intero gruppo di dati precedente. Le correzioni manuali sono limitate alle dipendenze elencate nell'audit.
