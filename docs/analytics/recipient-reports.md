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
| `service/api/engagement_report_manual_sources.go` | Validazione presenze, autorizzazione amministrativa, transazione e storico |
| `service/api/engagement_report_recipient_ai.go` | Responses API, schema rigoroso, validazione, cache e sintesi deterministica |
| `service/api/report_migrations/001_recipient_reports.sql` | Migrazione idempotente incorporata nel binario con `go:embed` |
| `service/api/engagement_report_recipients_test.go` | Test calcolo, SQLite in memoria, sicurezza e AI simulata |
| `src/components/RecipientReportBuilder.vue` | Stato reattivo, destinatario, completamento, analisi esplicita, anteprima PDF e archivio |
| `src/components/ReportKpiCard.vue` | Valore, N/D, azione contestuale, confronto e provenienza |
| `src/components/ReportAnalysisView.vue` | Analisi e riferimenti numerici inseriti dai valori verificati |
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

La sola integrazione manuale ammessa è **`attendees`**, intero da zero a dieci milioni, unità `people`, fonte obbligatoria. La raccolta attuale non offre altre sorgenti mancanti ricostruibili con sufficiente garanzia. Non sono ammessi tassi inseriti direttamente né conteggi inventati per sostituire esposizioni o utenti unici. Ogni KPI espone dipendenze e una configurazione `availability` con motivo, spiegazione, possibilità di input e campi con tipo, unità, limiti e validazioni.

L'adozione conserva la formula effettiva:

```text
onsite_visitors / attendees × 100
```

Non usa il totale dei partecipanti digitali come sostituto degli osservati con provenienza arena. Se il tracciamento arena manca, l'inserimento delle presenze non rende il KPI calcolabile. Se il numeratore supera le presenze, il valore resta N/D e viene segnalata l'incompatibilità. La provenienza arena, da sola, non certifica la presenza fisica.

Precedenza: **automatico > manuale verificato > manuale non verificato > N/D**. I record attivi hanno una revisione e uno stato, quindi non competono tra loro due record manuali per lo stesso campo. In presenza di una futura sorgente automatica in conflitto, il valore automatico viene mantenuto e il conflitto è esposto all'amministratore e nel report; il salvataggio non sovrascrive la sorgente automatica. I valori manuali non verificati possono completare il denominatore, ma sono marcati **NON verificati** in UI e PDF. Lo storico conserva i JSON prima e dopo ogni modifica, autore e timestamp nella stessa transazione.

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

L'input include profilo, società, eventi, valori selezionati, denominatori, definizioni, confronti già calcolati, origine, stato di verifica e limiti. Esclude segnali grezzi, ID tifoso/dispositivo/sessione, note manuali, descrizioni libere delle fonti e autore della sorgente. Le etichette sono trattate dal prompt come dati non affidabili, non come istruzioni.

Go valida lo stesso schema inviato al modello, campi obbligatori e sconosciuti, tipi, dimensioni, priorità `high/medium/low`, riferimenti a KPI selezionati **disponibili** e massimo tre insight complessivi e tre raccomandazioni. Il testo AI non può contenere numeri Unicode, simboli percentuali/monetari o i termini numerici esplicitamente vietati: i riferimenti numerici degli insight vengono inseriti dal renderer dai valori verificati. Non vengono accettati rifiuti, risposte incomplete o payload eccessivi. È previsto al massimo un retry nell'unico timeout totale, poi sintesi deterministica specifica del destinatario.

La cache SQLite identifica evento, profilo e sua configurazione, evento precedente, confronto attivo, hash dati, versione prompt e modello. L'hash include sorgenti manuali con revisione, denominatori, valori precedenti, limiti e ambito sponsor. Non include ID e timestamp volatili del calcolo. Modificare i dati, il profilo, il prompt o il modello produce una chiave differente. Le vecchie analisi possono rimanere archiviate senza essere riutilizzate con dati nuovi.

La prenotazione atomica `pending` accorpa richieste identiche concorrenti, anche tra processi che condividono SQLite, con confronto sul timestamp per evitare una corsa tra lettura e prenotazione. Le lease scadono dopo tre minuti in caso di crash. I fallback vengono riutilizzati per cinque minuti; `regenerate` rinnova esplicitamente il risultato, senza sottrarre una prenotazione in corso. Approvare o esportare un PDF non effettua chiamate OpenAI: usa il risultato valido in cache oppure la sintesi deterministica.

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
| `response_incomplete`, `invalid_response`, `provider_refusal` | Risposta incompleta, non leggibile o rifiuto del provider |
| `cached_fallback` | Fallback prodotto prima dell'introduzione della diagnostica: rigenerare per conoscere la causa |

La diagnostica viene mantenuta anche nella cache, senza modifiche allo schema SQLite. Il JSON in cache conserva i campi dell'analisi e un campo separato `diagnostic`; i payload precedenti restano leggibili. La diagnostica non cambia la struttura pubblica dell'analisi. Il retry di una risposta non valida comunica il codice della violazione al modello, mantenendo gli stessi controlli e il limite di un retry.

I log del backend riportano `recipient report OpenAI attempt failed` con codici, stato HTTP, tentativo e ID richiesta. Le chiavi, i corpi delle richieste e i messaggi grezzi del provider non sono registrati né restituiti, perché possono contenere informazioni sensibili.

Il timeout HTTP del server copre il maggiore tra il budget AI precedente e `OPENAI_REPORT_TIMEOUT`, con cinque secondi aggiuntivi per consegnare la risposta. Con `OPENAI_REPORT_TIMEOUT=45s` il server concede almeno cinquanta secondi. In precedenza il budget HTTP dipendeva soltanto dal vecchio servizio AI e poteva interrompere la risposta prima del completamento del report.

Verifica della correzione: tutti i quattordici test `TestRecipient*` superati, inclusi errori HTTP del provider, assenza di configurazione, cache, rigenerazione e protezione dei messaggi sensibili. Superata anche la suite API completa e il test del server che verifica i budget HTTP di cinquanta e novantacinque secondi. Le chiamate OpenAI dei test sono simulate; la causa specifica del fallback in produzione va letta dalla nuova diagnostica dopo il deployment e una rigenerazione esplicita.

### Limite complessivo degli insight — prompt v2

La diagnostica `invalid_output / insight limit exceeded` identifica un difetto del primo schema: `positive_insights` e `critical_insights` potevano contenere ciascuno tre elementi, mentre il vincolo applicativo era tre elementi complessivi. Il solo prompt non rendeva quel vincolo strutturale.

Da `recipient-report-v2`, lo schema interno inviato a Responses usa un unico array `insights` con `maxItems` uguale a `MaxInsights` del profilo. Ogni elemento contiene titolo, descrizione, KPI e `kind` con enum `positive/critical`. Dopo la validazione di questo schema il backend separa le categorie nei campi pubblici `positive_insights` e `critical_insights`, conservando integralmente gli elementi e applicando tutti i controlli già esistenti. Interfaccia, cache, snapshot e renderer PDF continuano a usare il formato pubblico originario. Non viene aumentato il limite né vengono troncati gli insight per aggirare una violazione.

La nuova versione prompt produce una chiave cache differente dai precedenti fallback. Gli snapshot già approvati restano invariati. Per attivare la correzione aggiornare il codice, ricostruire il backend e premere **Rigenera analisi**: nella risposta compare `prompt_version: "recipient-report-v2"`. L'uso di `maxItems` per l'unico array segue la [documentazione ufficiale OpenAI Structured Outputs](https://developers.openai.com/api/docs/guides/structured-outputs).

I test di regressione coprono tutte le combinazioni tra zero e quattro insight per categoria nei quattro profili, i limiti personalizzati, i KPI inesistenti, i numeri non consentiti, le categorie sconosciute e la compatibilità del JSON pubblico. Il caso originario di tre insight positivi più tre critici rimane rifiutato dalla validazione pubblica.

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
- Solo le presenze dispongono di una sorgente manuale compatibile; esposizioni, votanti unici e tracciamento mancante non vengono ricostruiti arbitrariamente.
- La validazione strutturale e dei riferimenti numerici è automatica. La veridicità semantica di ogni interpretazione libera, incluse formulazioni causali, non è dimostrabile con un JSON Schema: rimangono le istruzioni conservative e la revisione amministrativa dell'anteprima.
- Executive e Sponsor mantengono una pagina principale e abbreviano testi lunghi; analisi integrale e provenienza completa sono nello snapshot. Marketing privilegia la completezza con eventuali pagine di approfondimento oltre alle due principali.
- Le verifiche OpenAI usano risposte simulate: disponibilità, crediti e accesso al modello dell'account vanno verificati con le credenziali del deployment. Non sono state eseguite chiamate a pagamento.
- I report congelano dati, configurazione e analisi. Il renderer non è distribuito come binario versionato per ogni snapshot: future modifiche al renderer possono cambiare il layout; mantenere la versione dell'applicazione per una riproduzione storica identica.
