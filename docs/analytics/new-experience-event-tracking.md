# Tracciamento esperienza tifoso `/slug/new`

Gli eventi espliciti dell'esperienza tifoso alimentano la raccolta interna
`/events/{eventId}/report-signals`. La stessa osservazione viene inoltrata una sola
volta a PostHog come `fan_new.<nome_evento>`, se l'utente ha accettato le analisi
facoltative. Un ritentativo della raccolta interna mantiene l'ID originale e non
genera un nuovo evento PostHog. Nessun replay delle azioni precedenti al consenso.

## Percorsi coperti

| Area | Osservazioni |
| --- | --- |
| Home e navigazione | Inizio/caricamento riuscito o fallito, visita, ritorno nella stessa partita, pagina visibile/nascosta, uscita, apertura/chiusura sezioni, CTA cliccate e CTA realmente visibili. |
| MVP | Apertura, scelta del giocatore, richiesta di voto, voto confermato, cambio MVP confermato, errore e latenza, accredito del bonus confermato o fallito. |
| Giochi | Apertura del gioco, round iniziato/completato, modalità live/singola, risultato/punteggio, abbandono di un round iniziato, chiusura, premio accreditato o fallito. L'apertura della schermata iniziale non vale come round giocato. |
| Sponsor | Esposizione qualificata, clic sullo sponsor e visibilità del gioco sponsorizzato; ID pubblico e posizione, senza destinazione URL. |
| Registrazione/accesso | Avvio, passaggi, OTP richiesto/verificato o fallito, completamento e abbandono, senza contenuti dei campi. |
| Shop/profilo | Apertura shop e articolo, avvio/esito riscatto, equipaggiamento cornice, caricamento profilo/wallet/classifica, apertura/esito modifica profilo. |
| Feedback | Apertura, avanzamento, abbandono, invio confermato, errore; solo categorie delle risposte già previste, nessun testo libero. |
| Lotteria | Riga del codice visibile, clic per aprire il profilo, QR realmente visibile; nessun codice biglietto, contenuto QR o firma. |

Il contratto completo di nomi e proprietà è in
`wcmvpvs-front/src/fanAnalytics.js`; il backend applica la stessa lista chiusa e
gli stessi limiti tipizzati prima di memorizzare i dati. Gli eventi di successo
vengono emessi dopo la conferma dell'operazione, evitando di confondere un clic con
un risultato. Richieste asincrone di una società/partita precedente non vengono
attribuite al nuovo contesto.

## Tempo e ritorni

- `active_time`: intervalli fino a 15 secondi, solo con pagina visibile e attività
  recente; dopo 60 secondi di inattività il conteggio si interrompe.
- `visible_seconds`: tempo con pagina visibile, anche senza interazioni.
- `duration_ms`: durata trascorsa nella visita, distinta dal tempo attivo.
- `visit_returned`: nuova apertura/ripristino della stessa partita nel browser.
  Senza consenso il contatore resta nella sessione della scheda; con consenso può
  persistere nel browser. Non identifica con certezza una persona né un ritorno
  da un dispositivo diverso. Il conteggio non equivale al numero di presenti.

L'uscita usa invii keepalive, ma chiusura forzata del browser, assenza di rete e
blocco della telemetria possono causare osservazioni mancanti. Il tempo inviato a
PostHog inizia dalla scelta positiva: accettare a metà visita non invia il tempo
precedente. Queste durate restano misure del browser, non presenze certificate.

## Privacy e preferenze

PostHog richiede una scelta positiva separata e revocabile tramite “Preferenze
analisi”. Rifiutare non impedisce voto, giochi o profilo. DNT/GPC sono rispettati.
L'istanza dedicata usa un identificatore casuale del browser separato per società,
senza collegarlo al profilo tifoso o al device ID dell'applicazione.

Non vengono inviati nome/cognome, telefono, email, OTP, nickname, testo digitato,
token/sessioni di autenticazione, URL/query/referrer, dati del biglietto o del QR.
Autocapture, registrazioni sessione, raccolta automatica degli errori/rete e
profilazione delle persone sono disabilitati nell'istanza dedicata. Il filtro
finale PostHog rimuove anche le proprietà automatiche dell'SDK. I numeri del roster
e dello sponsor sono ID pubblici; browser e dispositivo sono categorie generiche.
Il contratto non include coordinate, valori dei campi o messaggi d'errore grezzi.

La raccolta interna mantiene l'identificazione tecnica già esistente, con device
hash per società e profilo verificato ricavato dal server; queste credenziali non
sono inoltrate a PostHog. Le informative e la versione di accettazione sono state
aggiornate insieme all'interfaccia delle preferenze.

## Report e analisi

Gli eventi diagnostici nuovi vengono conservati, ma il report esistente continua
a usare solo i 13 tipi di segnale originali per audience, tempo stimato,
copertura, retention e attribuzione delle fonti dei 30 KPI. Un errore di caricamento
o un passaggio del modulo non amplia retroattivamente il denominatore del report.

In PostHog gli ID di società, partita, segnale e sessione permettono funnel e
confronti evento per evento: apertura MVP → invio → conferma; apertura gioco →
inizio → completamento → premio; visibilità sponsor → clic; avvio registrazione →
passaggi → completamento; esposizione feedback → invio; visibilità QR → apertura.
Filtrare sempre società, evento e consenso/campione osservato; i browser che
rifiutano o bloccano PostHog non sono rappresentati nelle sue conversioni.

L'invio esterno richiede `VITE_POSTHOG_KEY` nell'ambiente di build e usa l'endpoint
UE configurato nell'app. I test verificano contratti, filtri, duplicati e finestre
di consenso senza inviare dati reali. Nessuna dashboard remota viene creata
automaticamente da questa modifica.
