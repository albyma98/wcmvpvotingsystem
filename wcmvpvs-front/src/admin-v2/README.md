# ArenaBoostX · amministrazione società V2

Frontend indipendente su `/:slug/admin/new`, aggiunto alla selezione delle viste in `App.vue`. Il router del progetto è basato su History API: non è stata introdotta una seconda libreria di routing. Il vecchio `/:slug/admin`, tutti i suoi componenti e il backend sono invariati.

## Pagine e copertura

| Percorso relativo a `/:slug/admin/new` | Funzioni |
| --- | --- |
| `/` | Panoramica, eventi operativi, roster, squadre, sponsor e accessi rapidi, tutti derivati dalle risposte esistenti |
| `/events` | Ricerca e filtro eventi, attivazione/disattivazione, creazione, conclusione, eliminazione, link pubblico |
| `/events/create` | Squadre, data/ora, luogo, esperienze, gioco sponsorizzato, premi e SMS vittoria |
| `/events/:id/:tab` | Esperienza, premi, sondaggio, quiz, stories; tutti i tipi di gioco e le impostazioni Stack It |
| `/closing` | Chiusura MVP, riattivazione, disattivazione, esiti premi e notifiche WhatsApp, retry consentiti |
| `/lottery` | Selezione evento, ticket validi/totali, estrazione, vincitore e revoca assegnazione |
| `/results` | Classifica MVP aggiornata e analytics sponsor |
| `/history` | Archivio, MVP/premi, survey, permanenza, sponsor, tracking/funnel/segmenti, report AI, PDF e report per destinatario, purge protetto da password |
| `/selfies` | Selezione evento, anteprima, consenso/metadati e rimozione selfie |
| `/teams` | Creazione ed eliminazione delle squadre |
| `/players` | Schema e 14 slot roster, convocazioni, upload/URL/ottimizzazione immagini, ripristino e salvataggio |
| `/sponsors` | Creazione/modifica/eliminazione, logo, nome report, link e visibilità dei primi N sponsor |
| `/coupons` | CRUD, partner, stato, date, limiti, immagini, ricerca/associazione eventi e contatori |
| `/partners` | Account partner e aggiornamento password |
| `/admins` | Account amministrativi e ruoli previsti dal sistema |
| `/bar` | Tutte le 11 aree operative: dashboard, ordini live, storico, prodotti/menù, categorie, suggerimenti, disponibilità locale, statistiche, clienti, cassa, preferenze locali |
| `/merch` | Catalogo, ordini e prenotazioni, impostazioni e analytics del [modulo merchandising](../../../docs/merchandising.md) |
| `/marketing` | Audience, dettaglio fan/SMS singolo, filtri cumulativi, campagne, stima costi, invio/test, template e registro |
| `/settings` | Template WhatsApp e test, destinatari, estrazione casuale/classifica e visibilità biglietto |

Inventari di dettaglio: [Eventi](docs/events-inventory.md), [operazioni e analisi](docs/operations-inventory.md). Le funzioni del bar che erano soltanto locali sono ancora locali e vengono presentate esplicitamente come tali.

## Architettura

- `AdminV2App.vue`: composizione pagine, sessione verificata, permessi esistenti, feedback, annullamento richieste in uscita e protezione bozze.
- `layouts/V2Workspace.vue`: navigazione raggruppata, sidebar comprimibile, drawer mobile e menu account.
- `config/navigation.js`, `router/routes.js`: metadati e percorsi con slug.
- `components/shared`: pannelli, icone e dialog con focus trap/inert/ripristino focus.
- `components/events`, `components/forms`, `components/dashboard`: presentazione specifica delle operazioni.
- `styles/admin-v2.css` definisce i token e gli stili dei controlli locali sotto `.admin-v2`.
- `composables`: sessione e notifiche. Il compilatore `build/adminV2Controllers.js` estrae a build time il solo script dei controller originali, senza importarne template o CSS e senza duplicare payload o regole. Dettagli e guardie in [CONTROLLERS.md](CONTROLLERS.md).

La modale report e il generatore per destinatario hanno presentazioni V2 dedicate in `components/reports`, comprese schede KPI e analisi, con gli stessi controller e lo stesso generatore PDF esistenti. I grafici specializzati sono riutilizzati. Non vengono introdotti endpoint, statistiche simulate, dipendenze runtime o sistemi di autenticazione paralleli.

## Sviluppo e verifica

```sh
npm run dev
npm run build
npm run test:admin-v2
```

La suite usa Node test runner, compilazione dei componenti Vue, rendering SSR e controller reali con risposte di prova. Copre routing, ruoli, 401/403, verifica server prima delle risorse, logout concorrenti, stati di caricamento, cleanup, payload bar, bozze e componenti critici.

Per ripetere la verifica manuale isolata:

```sh
node test/admin-v2-preview.mjs
```

Aprire `http://localhost:5187/societa-test/admin/new`. Account **esclusivamente di prova**: `qa/qa`, `staff/qa`, `bar/qa`. È possibile cambiare porta tramite `ADMIN_V2_QA_PORT`. Il server di prova mantiene i dati solo in memoria, non contatta il backend Go e non invia messaggi. Gli endpoint sconosciuti rispondono 501. Lo script è fuori dal bundle e non viene avviato dall’applicazione.

La verifica su fixture non certifica credenziali, autorizzazioni del server distribuito, consegna SMS/WhatsApp, storage degli upload o generazione AI reali. Questi flussi devono essere verificati con una sessione autorizzata nell’ambiente di integrazione. Il controllo WCAG svolto copre focus, label, tastiera e colori del design system; non costituisce una certificazione completa con tecnologie assistive.
