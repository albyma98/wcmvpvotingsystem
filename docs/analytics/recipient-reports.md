# Report per destinatario — ArenaBoostX

Il generatore esistente viene esteso, conservando il percorso originale di revisione ed esportazione. Nessuna modifica alla raccolta dei segnali dei tifosi e nessuna scrittura sui dati di produzione durante i test.

## Ricognizione del repository

| Area | Implementazione verificata |
| --- | --- |
| Ingresso amministrativo | `wcmvpvs-front/src/components/EngagementReportModal.vue`, aperto dalla selezione evento già presente nell'amministrazione |
| Revisione e catalogo frontend originali | `wcmvpvs-front/src/engagementReport.js` |
| PDF originale | `wcmvpvs-front/src/engagementReportPdf.js`, jsPDF, formato A4, identità visiva e firma logo riutilizzati |
| API report | `wcmvpvs-back/service/api/engagement_reports.go`, registrazione da `api-handler.go` |
| Calcolo e definizioni | `engagement_report_metrics.go`, `engagement_report_combined.go`, `engagement_report_sources.go`, `engagement_report_time.go` |
| Persistenza | SQLite, `database.go`; tabelle report inizializzate da `ensureEngagementReportTables` |
| Autenticazione | `admin-auth.go`: cookie HttpOnly o Bearer, esclusione partner/bar/staff dalle rotte report; `reportEventScope` limita l'evento alla società corrente |
| AI esistente | `ai_service.go`, `ai_event_reports.go`, `event_ai_reports`: flusso precedente lasciato funzionante; la nuova analisi usa Responses e lo stesso sottosistema engagement report |

Il catalogo Go è già composto da **30 KPI**: i primi venti sono quelli elencati nella richiesta, seguiti da dieci indicatori ulteriori. `buildReportKPIs` resta l'unica fonte delle definizioni e delle formule. Il generatore originale mantiene tutti i trenta; Analytics e appendice usano i primi venti, senza eliminare gli altri.

## Componenti e file introdotti

| File | Responsabilità |
| --- | --- |
| `service/api/engagement_report_recipients.go` | Configurazioni dei quattro profili, orchestrazione del calcolo esistente, confronto Go, indisponibilità, snapshot e archivio |
| `service/api/engagement_report_manual_sources.go` | Validazione presenze e completamenti di tutti i KPI N/D, ambito sponsor, autorizzazione, transazione e storico |
| `service/api/engagement_report_recipient_ai.go` | Responses API, schema rigoroso, validazione, cache e sintesi deterministica |
| `service/api/engagement_report_analysis_review.go` | Bozze editoriali, validazione dei testi, approvazione/scarto, revisioni concorrenti e autore della decisione |
| `service/api/report_migrations/001_recipient_reports.sql` | Migrazione idempotente incorporata nel binario con `go:embed` |
| `service/api/report_migrations/002_manual_kpis.sql` | Estensione transazionale delle sorgenti manuali a tutti i KPI, conservazione delle presenze e introduzione dello scope sponsor |
| `service/api/report_migrations/003_analysis_reviews.sql` | Revisioni editoriali persistenti, separate dall’output originale del provider |
| `service/api/engagement_report_recipients_test.go` | Test calcolo, SQLite in memoria, sicurezza e AI simulata |
| `src/components/RecipientReportBuilder.vue` | Stato reattivo, destinatario, completamento, analisi esplicita, anteprima PDF e archivio |
| `src/components/ReportKpiCard.vue` | Valore, N/D, azione contestuale, confronto e provenienza |
| `src/components/ReportAnalysisView.vue` | Editor di titoli, sintesi, evidenze, indicazioni e note; salvataggio, approvazione e scarto |
| `src/recipientReport.js` | Selezione del catalogo server, presentazione confronti, validazione sorgente |
| `src/recipientReport.test.js`, `src/components/RecipientReportBuilder.test.js` | Test dinamica Vue e PDF dei quattro profili |
| `test/recipientReportFixture.js`, `test/generateRecipientReportExamples.mjs` | Esempi sintetici; etichette e definizioni lette dal catalogo Go verificato |

Percorsi backend relativi a `wcmvpvs-back`, frontend relativi a `wcmvpvs-front`. Modificati anche `EngagementReportModal.vue`, il generatore PDF esistente, gli struct report/KPI, la pulizia delle tabelle in `database.go`, `.env.example` e gli script di `package.json`.

## Configurazioni e flusso

`reportRecipientProfiles()` contiene ID, nome, descrizione, ordine KPI, gruppi, obiettivo, tono, istruzioni AI, layout, limite insight e abilitazione appendice. I profili referenziano gli ID del catalogo Go: non contengono formule duplicate. Il frontend riceve le configurazioni dal backend.

| Profilo | KPI principali | Pagine principali |
| --- | --- | --- |
| Dirigenza | `visitors`, `venue_adoption`, `new_profiles`, `event_retention`, `sponsor_reach` | 1 |
| Marketing | `visitors`, `active_participation`, `new_profiles`, `registration_conversion`, `event_retention`, `active_time`, `mvp_participation`, `multi_experience`, `feedback_response`, `sponsor_reach` | 1 pagina KPI + analisi/metodo |
| Sponsor | `sponsor_reach`, `sponsor_impressions`, `sponsor_ctr`, `sponsored_participation`, `visitors` | 1 |
| Analytics | Primi 20 KPI originali, nello stesso ordine | Paginazione secondo definizioni, denominatori e analisi |

Il cambio destinatario seleziona immediatamente i KPI già calcolati, senza reload e senza chiamare OpenAI. Il passaggio da un report dedicato a un partner a un altro ambito ricarica il calcolo, perché cambia la popolazione misurata. I dati manuali restano associati all'evento.

L'anteprima è il **PDF effettivo**, generato dallo snapshot approvato e visualizzato tramite URL Blob. Anche l'anteprima salva una versione immutabile. L'esportazione successiva usa la stessa versione. L'archivio permette di scaricare nuovamente gli snapshot salvati senza ricalcolare le fonti. Ogni modifica alle opzioni o ai dati invalida l'anteprima locale; le versioni già approvate rimangono in archivio.

Il renderer usa header, footer, palette, font e firma esistenti. Il testo tecnico usa interruzioni pagina automatiche. Executive e Sponsor sintetizzano testi lunghi mantenendo una pagina principale; il JSON integrale resta nello snapshot. Marketing parte da due pagine e, quando analisi e note sono lunghe, aggiunge approfondimenti anziché sovrapporre o perdere testo. L'appendice include i venti KPI con definizioni e valori; Analytics non la duplica.

## Endpoint

Tutti i percorsi sotto `/admin` richiedono la sessione amministrativa e lo scope della società.

| Metodo | Percorso | Uso |
| --- | --- | --- |
| GET | `/admin/events/{eventId}/engagement-report` | Flusso originale; espone anche le configurazioni profili |
| GET | `/admin/events/{eventId}/engagement-report?recipient=executive` | Calcolo nuovo, tutti i KPI con confronti e motivi N/D; profili ammessi: `executive`, `marketing`, `sponsor`, `analytics` |
| GET | stesso percorso con `recipient=sponsor&sponsor_id={id}` | Calcolo del solo partner; nessun ID significa ambito aggregato |
| PUT | `/admin/events/{eventId}/engagement-report/sources` | Salvataggio transazionale delle sorgenti consentite |
| GET | `/admin/events/{eventId}/engagement-report/sources` | Sorgenti e storico delle modifiche |
| POST | `/admin/events/{eventId}/engagement-report/analysis` | Generazione esplicita o cache; corpo: `calculation_id`, `recipient`, `compare_previous`, `regenerate` |
| POST | `/admin/events/{eventId}/engagement-report` | Approvazione originale oppure nuova con `recipient`, `calculation_id`, `compare_previous`, `include_appendix`, `include_ai`, `notes` |
| GET | `/admin/events/{eventId}/engagement-report/snapshots` | Archivio a pagine di 50; `before` è il timestamp della versione più vecchia della pagina |
| GET | `/admin/events/{eventId}/engagement-report/snapshots/{reportId}` | Versione immutabile da rigenerare in PDF |

Un calcolo dello schema nuovo non può passare nel ramo di approvazione originale per aggirare i vincoli. Il ramo destinatario rifiuta `facts`, `overrides`, `descriptions` e selezioni KPI arbitrarie. Le formule e la composizione del profilo restano di proprietà del server. Il percorso originale conserva le revisioni già supportate per compatibilità con i client e i report precedenti.

Esempio di sorgente:

```json
{
  "sources": [{
    "field_key": "attendees",
    "value": 1800,
    "unit": "people",
    "source_type": "official_attendance",
    "source_description": "Distinta ufficiale biglietteria, riferimento evento",
    "verification_status": "verified",
    "note": "Conteggio comunicato dalla società",
    "revision": 0
  }]
}
```

`revision: 0` crea la sorgente. Gli aggiornamenti devono inviare la revisione letta: un dato modificato da un altro amministratore restituisce 409. ID amministratore e timestamp sono attribuiti dal backend. L'interfaccia richiede la conferma di partita, popolazione e salvataggio. Senza verifica esplicita la sorgente viene salvata come `unverified`.

## Sorgenti manuali, migrazione e N/D

La migrazione crea `engagement_report_manual_sources`, `engagement_report_source_history` e `engagement_report_analysis_cache`, con indici, vincoli e riferimenti all'evento. Viene applicata transazionalmente all'avvio nel punto già usato dai report; non richiede file SQL sul server di runtime. Le tabelle operative preesistenti e i segnali non vengono alterati. La cancellazione esplicita di un evento include le nuove tabelle anche quando le cascade SQLite sono disabilitate.

Tutti i KPI **N/D** sono completabili manualmente, come richiesto nell'estensione successiva. Il pulsante **Inserisci valore** apre il pannello sul KPI selezionato; un menu consente di scegliere un altro indicatore mancante. Sono conservati fonte obbligatoria, nota facoltativa, stato di verifica, autore, timestamp e revisioni. Dopo il salvataggio è disponibile **Modifica valore manuale**. Il completamento è individuale nell'interfaccia; l'endpoint accetta anche batch transazionali fino a trentuno elementi.

Per un completamento diretto usare `field_key: "kpi:<id_catalogo>"`, `source_type: "manual_kpi"` e l'unità effettiva del catalogo (`count`, `percent`, `seconds`). I conteggi devono essere interi non negativi; percentuali tra zero e cento; tempi non negativi, anche decimali. Conteggi e tempi hanno limite massimo di mille miliardi. Il backend valida ID, unità, ambito, fonte e revisione. Un KPI già disponibile da sorgenti calcolabili non può essere sostituito tramite il completamento N/D.

La sorgente **`attendees`** rimane disponibile separatamente, con interi da zero a dieci milioni e unità `people`. Il suo inserimento ricalcola l'adozione secondo la formula originale; un valore KPI diretto è invece presentato come **KPI manuale verificato / NON verificato**, senza inventare tracciamento o numeratori e denominatori. Gli altri KPI non vengono ricalcolati a partire da un valore derivato inserito direttamente. Ogni KPI N/D espone il motivo originario e il campo manuale con tipo, unità e limiti.

L'adozione conserva la formula effettiva:

```text
onsite_visitors / attendees × 100
```

Non usa il totale dei partecipanti digitali come sostituto degli osservati con provenienza arena. Se il tracciamento arena manca, l'inserimento delle presenze non rende il KPI calcolabile. Se il numeratore supera le presenze, il valore resta N/D e viene segnalata l'incompatibilità. La provenienza arena, da sola, non certifica la presenza fisica.

Precedenza: **valore calcolabile > completamento manuale verificato > completamento manuale non verificato > N/D**. I record attivi hanno una revisione e uno stato. Se arrivano dati calcolabili in conflitto con un completamento diretto, il valore calcolabile viene mantenuto e il conflitto è esposto all'amministratore e nel report. I valori non verificati sono marcati **NON verificati** in UI e PDF. Lo storico conserva i JSON prima e dopo ogni modifica, autore, timestamp, campo e ambito sponsor nella stessa transazione. Un batch con una revisione obsoleta viene annullato integralmente.

La migrazione `002_manual_kpis.sql` ricrea la tabella delle sorgenti con una chiave `(event_id, field_key, sponsor_id)`, copiando prima tutte le presenze precedenti con scope zero. Aggiunge lo scope anche allo storico, conservando le vecchie righe. L'avvio controlla la presenza della nuova colonna per applicare la migrazione una sola volta. Non è necessario eseguire SQL manualmente sul server.

Per i KPI del gruppo Sponsor, `sponsor_id: 0` identifica il totale evento; un ID positivo identifica esclusivamente quel partner, verificato nella stessa società. I dati manuali aggregati non vengono usati nei report dedicati e viceversa. Gli altri KPI, inclusi i partecipanti digitali di contesto, restano associati all'evento con scope zero. I valori salvati entrano nell'hash dati, nei confronti e negli snapshot: modificandoli cambia la cache AI utilizzabile. L'AI riceve il tipo di provenienza e il limite di misurazione originario, senza ricevere il testo libero della fonte o delle note.

Verifica dell'estensione: **74 test frontend superati**, suite API/database/server e build di produzione superate. Tre nuovi test backend coprono tutti i trenta KPI mancanti, revisioni e rollback dei batch, precedenza dei dati calcolabili, separazione degli sponsor, validazione e migrazione delle presenze precedenti. Verificato anche il pannello Vue reale in Edge con API simulate: selezione del KPI, inserimento di una percentuale decimale, conferma, salvataggio e riapertura del valore.

Motivi N/D: `manual_source_missing`, `tracking_missing`, `zero_denominator`, `feature_unavailable`, `not_applicable`, `not_synchronized`; aggiunto `incompatible_population`. L'indisponibilità di una funzionalità richiede evidenza delle osservazioni. Non si promette che dati storici arriveranno in futuro: `not_synchronized` è usato quando l'evento non è ancora iniziato, non per attribuire arbitrariamente un ritardo a un dato assente.

## Confronti e attribuzione sponsor

La partita precedente è quella della stessa società con data anteriore e `is_concluded=1`. La semplice data trascorsa non equivale a conclusione. Per calcolare il ritorno della partita precedente si carica anche la sua precedente conclusa. I valori manuali di entrambe le partite sono inclusi nel confronto e nello snapshot.

Go calcola differenza assoluta, variazione relativa dei conteggi se la base è positiva e punti percentuali per i tassi. Una base zero lascia la variazione percentuale N/D; unità, definizioni e chiavi dei denominatori diverse impediscono il confronto. Le direzioni sono `up`, `down`, `unchanged`, `unavailable`, senza associare automaticamente verde/rosso o giudizi positivi/negativi. Gli aggiornamenti successivi cambiano il nuovo calcolo e la sua cache, non i PDF approvati.

Per uno sponsor si filtrano le prove tramite `sponsor_id` e si riusa il calcolatore esistente. Non si usano somme o denominatori aggregati per attribuire risultati al partner. Il pubblico digitale complessivo è marcato come contesto evento. Senza esposizioni disaggregate, copertura, impression e CTR sono N/D; senza esposizione al gioco sponsorizzato anche la relativa partecipazione è N/D. I clic da soli non dimostrano un pubblico esposto pari a zero.

## OpenAI e cache

Nel `.env` del backend, già caricato da Docker Compose:

```dotenv
OPENAI_API_KEY=...
OPENAI_MODEL=gpt-4.1-mini
OPENAI_REPORT_TIMEOUT=45s
```

Il modello è configurabile e deve supportare Responses e Structured Outputs. Se chiave o modello mancano, si usa la sintesi deterministica. Non è necessaria una chiave per eseguire i test o generare gli esempi. Il timeout accetta durate Go da un secondo a due minuti; default 45 secondi. La chiave non viene esportata al frontend. L'integrazione usa HTTPS verso `/v1/responses`, `store: false`, `text.format.type: json_schema`, `strict: true`, proprietà obbligatorie e `additionalProperties: false` su tutti gli oggetti. Riferimento: [documentazione ufficiale Structured Outputs](https://developers.openai.com/api/docs/guides/structured-outputs?api-mode=responses).

L'input include società, titolo e data degli eventi, valori selezionati, denominatori, definizioni, confronti già calcolati, origine, stato di verifica e limiti. Il profilo fornisce obiettivo, tono e istruzioni una sola volta nel prompt. Esclude segnali grezzi, ID tifoso/dispositivo/sessione, note manuali, descrizioni libere delle fonti, autore della sorgente e branding. `organization_logo_url` non viene inviato: può contenere un intero logo Base64 che sarebbe elaborato come testo a pagamento. I loghi rimangono nello snapshot per il PDF. La regressione verifica la richiesta HTTP simulata nei quattro profili con e senza confronto, mantenendo identico il payload anche con grandi loghi Base64. Le etichette sono trattate dal prompt come dati non affidabili, non come istruzioni.

Go valida lo stesso schema inviato al modello, campi obbligatori e sconosciuti, tipi, dimensioni, priorità `high/medium/low`, riferimenti a KPI selezionati **disponibili** e massimo tre insight complessivi e tre raccomandazioni. Da `recipient-report-v4` i numeri nel testo non causano un rifiuto né un retry: producono un avviso da verificare prima dell’approvazione amministrativa. Il testo originale viene importato nella bozza modificabile. Non vengono accettati rifiuti del provider, risposte incomplete o payload eccessivi. È previsto al massimo un retry per `invalid_output` strutturale nell'unico timeout totale, poi sintesi deterministica specifica del destinatario. Errori HTTP, rete, input e risposte incomplete non provocano un secondo invio identico.

La cache SQLite identifica evento, profilo e sua configurazione, evento precedente, confronto attivo, hash dati, versione prompt e modello. L'hash include sorgenti manuali con revisione, denominatori, valori precedenti, limiti e ambito sponsor. Non include ID e timestamp volatili del calcolo. Modificare i dati, il profilo, il prompt o il modello produce una chiave differente. Le vecchie analisi possono rimanere archiviate senza essere riutilizzate con dati nuovi.

La prenotazione atomica `pending` accorpa richieste identiche concorrenti, anche tra processi che condividono SQLite, con confronto sul timestamp per evitare una corsa tra lettura e prenotazione. Le lease scadono dopo tre minuti in caso di crash. I fallback vengono riutilizzati per cinque minuti; `regenerate` rinnova esplicitamente il risultato, senza sottrarre una prenotazione in corso. Approvare, salvare o scartare un’analisi, aprire l’anteprima ed esportare un PDF non effettua chiamate OpenAI. Il PDF include soltanto la revisione editoriale approvata; la sintesi locale richiede la stessa approvazione se viene inclusa.

### Diagnosi di un fallback in produzione

La presenza delle variabili nel file `.env` va verificata nel processo del container, senza stampare la chiave:

```sh
docker compose exec -T backend sh -c 'test -n "$OPENAI_API_KEY" && echo OPENAI_API_KEY=presente || echo OPENAI_API_KEY=assente; printf "OPENAI_MODEL=%s\nOPENAI_REPORT_TIMEOUT=%s\n" "$OPENAI_MODEL" "$OPENAI_REPORT_TIMEOUT"'
```

Per tentare una chiamata effettiva premere **Genera analisi** o **Rigenera analisi**. Il solo controllo **Includi analisi AI** richiede l'inclusione del risultato disponibile nell'esportazione; non avvia una chiamata a pagamento.

La risposta di `POST .../engagement-report/analysis` espone `diagnostic` quando viene usata una sintesi deterministica. Lo snapshot approvato conserva lo stesso oggetto in `analysis_diagnostic`. `diagnostic.reason` distingue:

| Motivo | Significato |
| --- | --- |
| `missing_api_key`, `missing_model` | Configurazione assente nel processo backend |
| `no_available_kpis` | Il destinatario non ha valori disponibili da analizzare |
| `analysis_not_generated`, `analysis_pending` | Nessuna analisi richiesta o richiesta ancora in corso |
| `provider_http_error` | OpenAI ha risposto con errore: consultare `http_status`, `provider_code` e `request_id` |
| `request_timeout`, `network_error` | Timeout oppure problema di connessione dal backend |
| `invalid_output` | Risposta rifiutata dai controlli locali; `validation_error` contiene il motivo senza includere il testo prodotto |
| `input_too_large`, `invalid_input` | Richiesta oltre il budget di dimensione o dati non serializzabili: nessuna chiamata OpenAI eseguita |
| `response_incomplete`, `invalid_response`, `provider_refusal` | Risposta incompleta, non leggibile o rifiuto del provider |
| `cached_fallback` | Fallback prodotto prima dell'introduzione della diagnostica: rigenerare per conoscere la causa |

La diagnostica viene mantenuta anche nella cache, senza modifiche allo schema SQLite. Il JSON in cache conserva i campi dell'analisi e un campo separato `diagnostic`; i payload precedenti restano leggibili. La diagnostica non cambia la struttura pubblica dell'analisi. Il retry di una risposta non valida comunica il codice della violazione al modello, mantenendo gli stessi controlli e il limite di un retry.

I log del backend riportano `recipient report OpenAI attempt failed` con codici, stato HTTP, tentativo e ID richiesta. Le chiavi, i corpi delle richieste e i messaggi grezzi del provider non sono registrati né restituiti, perché possono contenere informazioni sensibili.

Il timeout HTTP del server copre il maggiore tra il budget AI precedente e `OPENAI_REPORT_TIMEOUT`, con cinque secondi aggiuntivi per consegnare la risposta. Con `OPENAI_REPORT_TIMEOUT=45s` il server concede almeno cinquanta secondi. In precedenza il budget HTTP dipendeva soltanto dal vecchio servizio AI e poteva interrompere la risposta prima del completamento del report.

Verifica della correzione: tutti i quattordici test `TestRecipient*` superati, inclusi errori HTTP del provider, assenza di configurazione, cache, rigenerazione e protezione dei messaggi sensibili. Superata anche la suite API completa e il test del server che verifica i budget HTTP di cinquanta e novantacinque secondi. Le chiamate OpenAI dei test sono simulate; la causa specifica del fallback in produzione va letta dalla nuova diagnostica dopo il deployment e una rigenerazione esplicita.

### Limite complessivo degli insight — prompt v2

La diagnostica `invalid_output / insight limit exceeded` identifica un difetto del primo schema: `positive_insights` e `critical_insights` potevano contenere ciascuno tre elementi, mentre il vincolo applicativo era tre elementi complessivi. Il solo prompt non rendeva quel vincolo strutturale.

Da `recipient-report-v2`, lo schema interno inviato a Responses usa un unico array `insights` con `maxItems` uguale a `MaxInsights` del profilo. Ogni elemento contiene titolo, descrizione, KPI e `kind` con enum `positive/critical`. Dopo la validazione di questo schema il backend separa le categorie nei campi pubblici `positive_insights` e `critical_insights`, conservando integralmente gli elementi e applicando tutti i controlli già esistenti. Interfaccia, cache, snapshot e renderer PDF continuano a usare il formato pubblico originario. Non viene aumentato il limite né vengono troncati gli insight per aggirare una violazione.

La versione prompt produce una chiave cache differente dai precedenti fallback. Gli snapshot già approvati restano invariati. La versione corrente è `prompt_version: "recipient-report-v4"`, con input compatto e revisione editoriale, mantenendo lo stesso vincolo complessivo sugli insight. L'uso di `maxItems` per l'unico array segue la [documentazione ufficiale OpenAI Structured Outputs](https://developers.openai.com/api/docs/guides/structured-outputs).

I test di regressione coprono tutte le combinazioni tra zero e quattro insight per categoria nei quattro profili, i limiti personalizzati, i KPI inesistenti, i numeri non consentiti, le categorie sconosciute e la compatibilità del JSON pubblico. Il caso originario di tre insight positivi più tre critici rimane rifiutato dalla validazione pubblica.

### Ottimizzazione input e misure di consumo — prompt v3

`engagement_report_recipient_input.go` definisce DTO dedicati al provider: nessuna serializzazione delle strutture UI complete. Vengono inviati soltanto i KPI del destinatario. I KPI disponibili conservano integralmente etichetta, definizione, valore, unità, numeratore, denominatore e fonte; quelli N/D diventano una lista compatta con ID, etichetta e motivo. Zero resta un valore disponibile. Le note sostanziali vengono raggruppate per testo con i relativi `metric_ids`; le note generiche duplicate e i limiti su funzionalità estranee al profilo vengono omessi. Il confronto contiene solo valori, variazioni già calcolate, motivo e provenienza sintetica precedente; è assente quando disabilitato.

Il tipo di fonte manuale e lo stato di verifica restano distinti sia per il dato corrente sia per quello precedente, senza inviare testo libero o identificativi dell'autore. L'ambito `event_total`/`selected_partner` impedisce di confondere i totali sponsor dell'evento con quelli del partner. Titoli e nomi decorativi hanno limiti espliciti, mantenendo intatto lo snapshot originale. Le definizioni e i numeri dei KPI non vengono troncati per rientrare nel budget.

L'intera richiesta HTTP, incluse istruzioni e schema di risposta, ha un limite di **24 KiB**. Se viene superato, il provider non viene chiamato: si restituiscono sintesi deterministica e `diagnostic.reason: "input_too_large"`. Questo è un limite di byte, non un limite esatto di token. Valori non serializzabili producono `invalid_input` prima dell'HTTP.

La verifica locale copre **24 casi sintetici**: quattro profili, dati automatici/parziali/manuali, con e senza confronto. Con il tokenizer `o200k_base` di `gpt-4.1-mini`, la somma locale di dati, istruzioni e JSON Schema va da **999 a 3.431 token**, con richieste complete fino a **15.924 byte**. Rispetto allo stesso input già privo di loghi, la riduzione stimata va dal **23,2% al 59,5%**. Esempi con tutti i KPI disponibili e confronto attivo:

| Profilo | Prima, senza loghi | Dopo | Riduzione |
| --- | ---: | ---: | ---: |
| Dirigenza | 1.720 | 1.289 | 25,1% |
| Marketing | 2.537 | 1.774 | 30,1% |
| Sponsor | 1.786 | 1.301 | 27,2% |
| Analitico | 4.122 | 2.810 | 31,8% |

Sono stime locali su fixture, non consumi di produzione né conteggi comprensivi della formattazione interna del provider. Il conteggio effettivo è quello restituito da OpenAI. Le misure non richiedono una chiave API; tiktoken è stato installato solo in una cartella temporanea e non è una nuova dipendenza del backend. Per esportare localmente le sole fixture sintetiche impostare `REPORT_AI_TEST_EXPORT_DIR` ed eseguire `go test ./service/api -run '^TestRecipientAIRequestBudgets$' -count=1`.

La risposta `POST .../engagement-report/analysis` espone ora `usage` con `input_tokens`, `cached_input_tokens`, `output_tokens`, `total_tokens`, `requests`, `reported_requests` e `max_request_bytes`. I token sommano tutti i tentativi, anche una risposta rifiutata dalla validazione locale. `cached_input_tokens` è un sottoinsieme di `input_tokens`. Se `reported_requests < requests`, alcuni tentativi non hanno restituito contatori e il consumo noto è parziale; zero non prova che quei tentativi non abbiano avuto costi. Con `cached: true`, `usage` descrive la generazione originale archiviata e **non** nuove chiamate o nuovi addebiti. Le vecchie cache senza contatori rimangono leggibili. I log `recipient report OpenAI usage` contengono contatori e dimensioni, senza prompt o segreti.

Superata la suite API completa con CGO e SQLite in memoria. Le regressioni verificano integrità dei denominatori e dei valori decimali, N/D e zero, esclusione di loghi e campi UI, provenienza manuale corrente/precedente, deduplicazione delle note, ambito sponsor, blocco degli input eccessivi, cache dei contatori, contabilizzazione dei retry e assenza di retry inutili. Nessuna chiamata OpenAI a pagamento è stata effettuata.

## Revisione editoriale e approvazione — prompt v4

Il blocco `free numeric claims forbidden` è stato sostituito da un avviso: cifre, numeri in lettere, percentuali e simboli monetari restano nei testi della bozza. `status: "ready"` indica che il provider ha restituito una struttura valida, non che la bozza sia stata approvata o che i numeri siano automaticamente certificati. L’approvazione è indicata separatamente da `review.state: "approved"`. Le istruzioni continuano a richiedere dati coerenti con i KPI. Structured Outputs può contenere errori di contenuto anche quando la struttura è corretta, come precisato nella [documentazione ufficiale](https://developers.openai.com/api/docs/guides/structured-outputs#handling-mistakes).

Sono modificabili titolo principale, sintesi, titolo e descrizione di ogni evidenza positiva/critica, azione e motivazione di ogni raccomandazione, priorità e tutte le note/limitazioni. Gli elementi non necessari possono essere rimossi. Sono disponibili **Salva bozza**, **Approva analisi**, **Scarta analisi** e **Ricarica bozza salvata**. Una modifica locale invalida subito l’anteprima e l’approvazione corrente nell’interfaccia; salvare la modifica riporta anche la revisione persistente in stato `draft`. Scartare esclude l’analisi dal PDF e consente di esportare senza la sezione; la decisione di scarto viene ripristinata anche riaprendo la pagina. Una nuova generazione richiede una nuova approvazione.

Le bozze sono salvate nella tabella `engagement_report_analysis_reviews`, senza alterare l’output AI originale della cache. La chiave comprende l’identità della cache (dati, modello, destinatario, confronto, ambito sponsor, versione prompt) e l’amministratore che effettua la revisione. Ogni record conserva numero di revisione, versione della generazione, stato, data di aggiornamento e autore/data di approvazione o scarto. Riaprire un nuovo calcolo con gli stessi dati ripristina la revisione dell’amministratore; altri amministratori, profili, confronti o dati diversi non ereditano l’approvazione. La migrazione `003_analysis_reviews.sql` è idempotente ed eseguita automaticamente all’avvio. La pulizia dell’evento elimina anche le relative revisioni.

| Endpoint | Operazione | Chiama OpenAI |
| --- | --- | --- |
| `POST .../engagement-report/analysis` | Genera o rigenera la bozza, con eventuale riuso della cache | Solo se necessario, su click esplicito |
| `GET .../engagement-report/analysis/review` | Legge la revisione salvata oppure prepara una sintesi locale da rivedere | No |
| `PUT .../engagement-report/analysis/review` | Salva, approva o scarta i testi con `action: draft/approve/discard` | No |
| `POST .../engagement-report` | Congela lo snapshot per anteprima/esportazione | No |

La GET usa `calculation_id`, `recipient` e `compare_previous`. La PUT richiede gli stessi campi, `generation_version`, `revision`, `action` e `analysis` per salvataggio/approvazione. La struttura dell’analisi mantiene i campi pubblici originali. Una bozza incompleta può essere salvata o scartata, ma per approvarla devono essere compilati i testi degli elementi rimasti. I limiti editoriali sono: titolo 180 caratteri, sintesi 6.000, titoli evidenze 180, descrizioni 3.000, azioni 500, motivazioni 3.000 e fino a 32 note di 1.000 caratteri. Restano i limiti complessivi sugli insight e i riferimenti a KPI selezionati disponibili. Non si accettano proprietà sconosciute o cambiamenti al catalogo attraverso l’editor.

Il salvataggio controlla revisione e versione della generazione in transazione; richieste obsolete restituiscono `409` e il frontend conserva il testo non salvato. Una generazione in corso impedisce approvazioni ed esportazioni della versione sostituita. Il POST di esportazione con `include_ai: true` richiede `analysis_review_revision` corrispondente a una revisione approvata; altrimenti restituisce `409`. Con `include_ai: false` il PDF viene creato senza analisi. La lettura della revisione approvata e la creazione dello snapshot avvengono nella stessa transazione.

Il PDF conserva i testi approvati e `analysis_review`, includendo autore, data e revisione. Il renderer identifica i testi come approvati, mostra tutte le raccomandazioni e note e impagina i testi lunghi su pagine successive senza troncarli. Il simbolo euro viene scritto come `EUR` nel font PDF. I vecchi snapshot privi di metadati editoriali rimangono esportabili. L’ottimizzazione dell’input e il limite di 24 KiB restano attivi; nessun testo modificato viene reinviato al modello durante la revisione.

Verifica: **80/80 test frontend**, suite API/database/server complete superate, provider simulato e SQLite in memoria. Il controllo nel browser ha verificato modifica di tutti i campi, salvataggio e riapertura, blocco dell’export prima dell’approvazione, invalidazione dopo modifica, scarto persistente e tre esportazioni con una sola generazione AI simulata. Verificata la vista mobile senza scorrimento orizzontale. I PDF sintetici breve/lungo risultano rispettivamente di una e due pagine, con testi completi e impaginazione verificata mediante rendering a risoluzione fissa. Nessuna chiamata OpenAI a pagamento.

## Verifica ed esempi

Da `wcmvpvs-front`:

```sh
npm run test:reports
npm run build
npm run reports:examples
```

Da `wcmvpvs-back`, con compilatore C e CGO per il driver SQLite già usato dal progetto:

```sh
CGO_ENABLED=1 go test ./service/api ./service/database
```

I test nuovi usano SQLite `:memory:` e trasporti HTTP simulati. Coprono profili e ordine KPI, cambio reattivo, N/D, input non ammessi, autorizzazione e isolamento società, revisioni concorrenti, storico, provenienza, ricalcolo, popolazioni incompatibili, confronti omogenei e base zero, attribuzione sponsor, JSON AI, retry, fallback, cache, invalidazione e snapshot. I test originali del renderer sono mantenuti. La rigenerazione dello stesso snapshot produce PDF identici a livello di byte, con ID e data di creazione deterministici.

Gli esempi in `output/pdf/arenaboostx-{executive,marketing,sponsor,analytics}-esempio.pdf` sono chiaramente dimostrativi e contengono dati sintetici. L'esportatore non apre database né file `.env`. Le definizioni derivano dal catalogo Go; per provare l'appendice usare la fixture con `include_appendix` attivo.

### Risultati della verifica locale

| Controllo | Risultato |
| --- | --- |
| `npm run test:reports` | **71/71 superati**, inclusi i test originali e la telemetria |
| Nuovi test Go `TestRecipient*` | **11/11 superati**, SQLite in memoria e provider simulato |
| `go test ./service/api ./service/database -count=1 -timeout 240s` con CGO | **Entrambe le suite superate**; API 84,206 s, database 143,075 s |
| `npm run build` | **Riuscita**; rimangono gli avvisi del progetto su dati Browserslist e dimensioni dei bundle |
| Esempi PDF | Dirigenza **1** pagina, Marketing **2**, Sponsor **1**, Analytics **4** |
| Verifica visiva | Tutte le **8 pagine** renderizzate con Poppler e ispezionate: nessuna sovrapposizione o uscita dai margini |
| Riproducibilità PDF | Test di uguaglianza dei byte per lo stesso snapshot **superato** |
| `git diff --check` | **Superato** |

Sul computer di verifica è stato usato un compilatore C portabile temporaneo per abilitare CGO, senza cambiare PATH o impostazioni di sistema. Le immagini temporanee di QA sono state rimosse; restano i quattro PDF dimostrativi. Nessun database di produzione è stato aperto o modificato dai test.

## Assunzioni e limiti operativi

- Lo schema eventi non contiene uno stato di annullamento distinto. Il nuovo confronto esclude gli eventi non conclusi; distinguere un evento annullato erroneamente marcato concluso richiederebbe un nuovo stato nel dominio eventi, che non viene inventato in questa modifica.
- Nessuna certificazione automatica della presenza fisica: è conservata la semantica della provenienza arena già adottata.
- Tutti i KPI N/D sono completabili direttamente dall'amministratore, con provenienza esplicita. Un valore manuale non certifica né ricostruisce il tracciamento mancante; le definizioni e i motivi originari di indisponibilità rimangono consultabili.
- La validazione strutturale e dei riferimenti ai KPI è automatica. I numeri nel testo generano un avviso e richiedono revisione amministrativa; un JSON Schema non dimostra la veridicità semantica di un’interpretazione o di una formulazione causale.
- Executive e Sponsor mantengono un formato compatto con KPI principali; i testi approvati lunghi possono aggiungere pagine per conservare integralmente il contenuto. Marketing privilegia la completezza con eventuali pagine di approfondimento oltre alle due principali.
- Le verifiche OpenAI usano risposte simulate: disponibilità, crediti e accesso al modello dell'account vanno verificati con le credenziali del deployment. Non sono state eseguite chiamate a pagamento.
- I report congelano dati, configurazione e analisi. Il renderer non è distribuito come binario versionato per ogni snapshot: future modifiche al renderer possono cambiare il layout; mantenere la versione dell'applicazione per una riproduzione storica identica.
