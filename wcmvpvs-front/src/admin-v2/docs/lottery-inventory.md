# Lotteria — inventario e migrazione

Fonte: `src/components/AdminLottery.vue`. La V2 usa soltanto il controller headless del file, con una UI nuova in `LotteryView.vue`. La sessione è quella del workspace e i controlli originali `/admin/me` restano nel controller. Il ruolo `staff` è escluso; la condizione legacy ammette gli altri ruoli, incluso `bar`.

La rotta V2 è `/:slug/admin/new/lottery`, con selezione diretta opzionale in `/lottery/:eventId`. Il selettore include gli eventi non conclusi, come l’admin precedente, e conserva la scelta automatica dell’evento più recente. I nomi delle squadre, date, luogo, premi e ticket provengono dalle API esistenti.

Funzioni migrate: selezione evento, refresh eventi/ticket, numero ticket ammessi e totali, elenco premi ordinato, estrazione animata originale, assegnazione persistente del vincitore, esclusione del ticket appena assegnato, display del codice, riepilogo dell’ultimo vincitore, stato SMS, annullamento del vincitore e gestione degli errori 400/401/404/409. Il sistema conserva il divieto di estrazione senza ticket, con premio già assegnato o durante estrazione/caricamento. Le assegnazioni definitive automatiche sono protette dal backend.

API immutate:

- `GET /admin/me`, `GET /teams`, `GET /events`.
- `GET /events/:eventId/tickets`; totali nell’header `x-lottery-total-codes`.
- `POST /events/:eventId/prizes/:prizeId/assign`, payload `{ vote_id }`.
- `DELETE /events/:eventId/prizes/:prizeId/winner`.

L’header `X-Organization-Slug` continua a essere aggiunto dall’interceptor esistente di `apiClient`, e le chiamate usano i cookie della sessione. Nessuna autenticazione parallela e nessun endpoint nuovo.

UI: native select con label, azioni con dimensione minima condivisa, conferma di estrazione e annullamento, protezione dai doppi invii, display live silenzioso durante la rotazione, annuncio del risultato finale, card responsive. I timer vengono rimossi al cambio pagina con il cleanup originale e quello della view.

I test di contratto usano il controller reale compilato ed eseguono assegnazione/reset contro API fixture, verificano payload, condizioni di abilitazione, stato SMS, protezioni 409 e chiusura dei timer. Non attestano l’invio di SMS o l’assegnazione su un backend di produzione autenticato.
