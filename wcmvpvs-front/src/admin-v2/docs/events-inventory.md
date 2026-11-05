# Eventi — inventario e migrazione

Fonte canonica: `src/components/AdminPortal.vue`, UI inline Eventi e relativo controller. `AdminEventsSection.vue` non viene montato dal portale ed è incompleto; non è stato riutilizzato.

La V2 monta soltanto componenti nuovi. `EventsView` usa il controller esistente attraverso `inject('admin-v2')`, mantenendo serializzazione, normalizzazione, validazioni, protezione HTTP, bozze e permessi. Non aggiunge endpoint.

| Funzione esistente | Nuova posizione |
| --- | --- |
| Ricerca, filtro stato, lista non conclusi, stato attivo/chiuso | `/events` |
| Creazione con squadre diverse, data/ora, luogo, pre/post voto, branded, premi/SMS | `/events/create` |
| Attiva, pagina voto, copia link, concludi, elimina | `/events/:id` |
| Disattiva tutti | `/events` |
| Esperienze pre/post voto e gioco sponsorizzato | `/events/:id/settings` |
| Premi multipli, SMS vincita, vincitori, rimozione protetta | `/events/:id/prizes` |
| Quattro domande e codici risposta fissi, testo suggerimenti | `/events/:id/feedback` |
| Stato quiz, reload/save config, CRUD domande, 2–4 risposte, indice corretto, ordine | `/events/:id/quiz` |
| CRUD stories, thumbnail, video URL/upload, nome/titolo, pubblicazione, riordino | `/events/:id/stories` |
| Storico/report e lotteria | Collegamenti alle funzioni esistenti |

Branded Game conserva sponsor ID/nome/logo, due colori, cinque giochi (`tap_challenge`, `memory_flash`, `sponsor_rush`, `free_throw_challenge`, `stack_it`), CTA, reward coins/none, coin e limite partite. Stack It usa i campi già presenti nella configurazione: texture, colori, sfondo, due suoni, coin per blocco, bonus perfect, velocità iniziale 400–3000 ms, curva `gentle/standard/aggressive`, CTA label/URL. L'anteprima usa i valori della bozza senza generare risultati simulati.

API immutate: `/events`, `/events/:id`, `/events/:id/activate`, `/events/deactivate`, `/events/:id/conclude`, `/admin/events/:id/quiz`, `/admin/events/:id/quiz/questions`, `/admin/events/:id/quiz/questions/:questionId`, `/admin/events/:id/stories`, `/admin/events/:id/stories/:storyId`, `/admin/events/:id/stories/upload-video`. L'header società e i cookie sono gestiti dal controller/servizio condiviso.

Il controller conserva validazioni squadre/data, branded ID/nome/tipo/reward/limite e stories thumbnail/video. I salvataggi esperienza/premi/sondaggio restano separati; il reload conserva le bozze degli altri tab. I premi assegnati restano non rimovibili. La conclusione passa allo storico; chiusura votazioni e relativi effetti sono nella pagina Votazioni.

I tab hanno URL diretti e mantengono lo slug. I controlli usano label associate, native select/date picker, focus e design system locali. Eliminazione/conclusione/disattivazione e reload di bozze richiedono conferma. Le operazioni usano feedback centralizzato e prevenzione doppi invii. Le form partecipano alla protezione modifiche non salvate dell'app.

Verifica automatica: `node --test src/admin-v2/events.test.js` rende le viste effettive con normalizzatori reali e fixture API, coprendo elenco, creazione, tutti i tab, configurazioni branded, vincitori protetti, upload e pagina evento inesistente. Questi test non attestano operazioni su un backend autenticato reale.
