# Inventario funzionale amministrazione società: operazioni e analisi

Fonte canonica: `src/components/AdminPortal.vue` e componenti montati da esso. Il file legacy rimane invariato. La V2 usa i controller legacy come moduli headless, mantenendo payload, normalizzazioni, validazioni e API.

| Area | Funzioni effettive | API esistenti |
| --- | --- | --- |
| Squadre | Elenco, aggiunta nome, eliminazione | GET/POST `/teams`, DELETE `/teams/:id` |
| Giocatori | Slot completi nome/cognome/ruolo/maglia/squadra/convocato; schema 12/13/14; URL o upload ottimizzato 600×600 qualità .75; anteprima, rimozione foto; ripristino dati; salvataggio roster con rimozione slot svuotati e overflow | GET/POST `/players`, PUT `/players/settings`, PUT/DELETE `/players/:id` |
| Sponsor | Massimo quattro; nome pubblico e report, sito, logo upload; posizione; slider numero sponsor visibili che aggiorna primi N in ordine; modifica/elimina | GET/POST `/admin/sponsors`, PUT/DELETE `/admin/sponsors/:id` |
| Coupon | Titolo e partner obbligatori; descrizione; stato draft/active/paused/archived; limite zero illimitato; date; immagine URL/upload/rimozione; associazione multipla eventi con ricerca; conteggi viste/richieste/utilizzi; CRUD | GET/POST `/admin/coupons`, PUT/DELETE `/admin/coupons/:id` |
| Partner | Credenziali esercenti; username/password; nome fallback; data creazione; cambio password; eliminazione | GET/POST `/admin/partners`, PUT/DELETE `/admin/partners/:id` |
| Amministratori | Account con username/password/ruolo libero, elenco, eliminazione | GET/POST `/admins`, DELETE `/admins/:id` |
| Chiusura votazioni | Partita attiva; chiudi votazioni con assegnazione premi e WhatsApp; riapri tramite activate; disattiva; riepilogo MVP, percentuale, vincitori e warning; notifiche e retry failed/skipped; polling quattro secondi per pending/sending | POST `/events/:id/close-votes`, POST `/events/:id/activate`, POST `/events/deactivate`, GET `/events/:id/voting-closure`, POST `/events/:id/voting-closure/retry` |
| Risultati | Selezione evento, classifica MVP ordinata e barre, voti totali, aggiornamento manuale e polling cinque secondi; KPI sponsor e andamento visualizzazioni/click | GET `/events/:id/results`, GET `/admin/events/:id/sponsors/analytics` |
| Selfie | Selezione evento, galleria, immagine originale, didascalia, data/device/peso/consenso/stato; eliminazione | GET `/admin/events/:id/selfies`, DELETE `/admin/selfies/:id` |
| Storico | Elenco eventi conclusi; report AI e rigenerazione; report PDF con generatore originale e per destinatario; MVP/premi/codici; sponsor e timeline; survey e risposte aperte; permanenza/aperture/abbandoni; tracking e metadati; funnel, segmenti ospiti/registrati, attività più frequenti; andamento voti per quindici minuti; purge password superadmin | GET `/admin/events/history`, POST `/admin/events/:id/ai-report`, POST `/admin/events/:id/purge` e API report già presenti |
| Report | KPI reali e fonti; selezione KPI, correzioni/fatti, descrizioni/note, formato one pager/completo, PDF precedente; profili executive/sponsor/marketing, sponsor aggregato/singolo; confronto, appendice, analisi, fonti manuali con revisione e storico, snapshot/versioni | `/admin/events/:id/engagement-report`, sottorisorse `/sources`, `/analysis`, `/snapshots` esistenti |

## Permessi e sessione

- Superadmin: tutte le quattordici sezioni.
- Bar: solo BAR, quando abilitato.
- Ogni altro ruolo, incluso staff: solo chiusura votazioni.
- Il ruolo viene riallineato da `/admin/me`; username e ruolo in localStorage sono indizi UI. L'autenticazione resta quella cookie del client `withCredentials` e del backend.
- `authHeaders` include `X-Organization-Slug`; l'interceptor del client aggiunge lo slug da URL se manca. I nuovi link restano nello slug della società.
- Risposte 401 richiedono un nuovo login. Le API applicano i permessi reali e l'isolamento dati.

## BAR e marketing: limiti da preservare

BAR comprende dashboard, ordini live, storico con ricerca/data/stato, prodotti, menù composti, categorie con immagini, suggerimenti prodotto/categoria, disponibilità locale, statistiche, clienti/storico cliente, cassa e preferenze locali. API: `/admin/bar/overview`, `/orders`, `/orders/:id/status`, `/products`, `/categories`, `/menus`, `/suggestions/products`, `/suggestions/categories` e sottorisorse. Polling ogni quindici secondi. Le preferenze e la disponibilità rapida del legacy non hanno endpoint di persistenza: non vanno presentate come salvate sul server.

Marketing comprende audience con ricerca e dettaglio, consenso, SMS singolo, campagna con filtri genere/nuovi utenti/saldo/top saldo cumulabili, conteggio destinatari e costo, creazione/invio/test campagna, template per categoria, log e fatturazione. Le chiamate SMS sono azioni reali e i test funzionali devono usare risposte controllate, senza inviare comunicazioni ai tifosi.

## Controlli di parità e verifica delle viste

- Coupon: `highlight` non è esposto dal legacy e il serializer invia sempre `false`; la V2 conserva questo comportamento.
- Sponsor: la posizione è assegnata dal controller; il vecchio editor la mostra, mentre la visibilità è gestita dal cursore dei primi N sponsor. La V2 conserva questi controlli, senza cambiare la semantica del payload.
- I funnel dello storico sono sequenze `steps` con `key`, `label`, `countLabel`, non oggetti di conversione con `from/to/rate`. La vista mostra ogni passaggio normalizzato.
- `rendering.test.js` carica realmente i componenti Vue via Vite SSR e usa il controller originale generato per normalizzare payload server popolati di storico, coupon, roster e selfie. Verifica i dati resi e l’assenza di chiamate API durante il rendering.
