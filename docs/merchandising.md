# Merchandising ArenaBoostX

## Percorsi

| Superficie | Percorso | Operazioni |
| --- | --- | --- |
| Amministrazione V2 | `/:slug/admin/new/merch` | Dashboard, catalogo, ordini, impostazioni e analytics |
| Master Admin V2 | Scheda società → Merchandising | Accesso al modulo con la sessione Master esistente |
| Portale tifoso | `/:slug/new/shop` | Catalogo, ricerca, categorie e carrello |
| Scheda condivisibile | `/:slug/new/shop/:product` | Scheda rapida, variante e checkout |
| Ricevute | `/:slug/new/orders` | Ordini del profilo e ricevute guest salvate |
| Arena Tifosi 3D | Shop e guardaroba | Collegamento alla stessa scheda tramite la chiave dell'articolo digitale |

Attiva lo Shop nelle impostazioni della società, scegli una modalità di vendita e almeno una modalità di consegna, poi pubblica i prodotti. Le modalità del prodotto restringono quelle abilitate a livello società. La visibilità nella navigazione è configurabile separatamente. I prezzi sono in EUR, salvati in centesimi. L'upload accetta JPEG, PNG e WebP statici fino a 5 MiB.

## Pagamento online

1. Collega alla piattaforma l'account Stripe della società con accesso alla Dashboard completa, gestione delle perdite a carico di Stripe, pagamenti con carta e accrediti attivi.
2. Imposta `STRIPE_SECRET_KEY`, `MERCH_STRIPE_CONNECT_ACCOUNTS` come oggetto JSON con le coppie slug/account, `MERCH_PUBLIC_BASE_URL` con l'origine HTTPS del portale e `MERCH_STRIPE_WEBHOOK_SECRET` con il segreto dell'endpoint Connect.
3. In Stripe crea un endpoint eventi **snapshot** per gli account connessi su `POST /api/merch/stripe/webhook`. Sottoscrivi `checkout.session.completed`, `checkout.session.async_payment_succeeded`, `checkout.session.async_payment_failed`, `checkout.session.expired`, `refund.created`, `refund.updated` e `refund.failed`.
4. Verifica lo stato **Account verificato** nelle impostazioni della società e abilita il pagamento online. Con una configurazione mancante o indisponibile, il portale offre la prenotazione se attiva.

ArenaBoostX apre Stripe Checkout ospitato dopo aver ricalcolato disponibilità, prezzi e spedizione. Il charge è diretto sull'account connesso. Stripe mostra i metodi compatibili con account, cliente e dispositivo, inclusi carta e, dove disponibili, Apple Pay, Google Pay e Link. La piattaforma non applica una commissione nel codice. Incassi e rimborsi sono registrati dallo stato verificato lato server. Il redirect dal checkout apre la ricevuta in attesa di aggiornamento.

Per le prove end-to-end usa un account collegato e chiavi **sandbox** insieme al webhook della stessa modalità. La suite automatica usa SQLite temporaneo, fixture HTTP per Stripe e webhook firmati localmente. Apple Pay e Google Pay richiedono un dispositivo e un contesto supportati per una verifica reale. [Stripe Direct Charges](https://docs.stripe.com/connect/direct-charges?platform=web&ui=stripe-hosted) · [Webhook Connect](https://docs.stripe.com/connect/webhooks)

## Prenotazioni e notifiche

Il tifoso può confermare da ospite con nome e un recapito secondo la regola della società. La ricevuta mostra codice, importo da pagare, consegna e istruzioni della società. L'amministratore può confermare, preparare, registrare l'incasso integrale dichiarato e segnare il prodotto come consegnato. Il pagamento offline ha un proprio record con importo, metodo, data, operatore e riferimento.

Le ricevute sono disponibili nel profilo o con un collegamento privato per l'ospite, valido 180 giorni e salvabile dal browser. Per le notifiche di servizio, configura `ACCOUNT_SID`, `AUTH_TOKEN` e `MESSAGING_SERVICE_SID` e abilita gli SMS nelle impostazioni della società. Gli SMS automatici sono destinati ai profili il cui recapito verificato coincide con quello dell'ordine. Un ospite che ha fornito un telefono può chiedere il link via SMS dalla ricevuta, dopo aver confermato un codice di verifica. Questa funzione richiede un'origine HTTPS e `MERCH_TWILIO_VERIFY_SERVICE_SID` con un servizio Verify separato da quello usato per il login. [Twilio Verify](https://www.twilio.com/docs/verify/api/verification)

La coda SQLite accoda un solo SMS per ordine e tipo di evento. Un timeout con esito ignoto viene conservato come tale per evitare invii duplicati. Le conferme restano disponibili nel portale. Il recapito email può essere raccolto per la gestione dell'ordine da parte della società. Gli invii automatici del modulo usano il canale SMS.

## Stati

| Asse | Stati |
| --- | --- |
| Pagamento | `PENDING_ONLINE`, `UNPAID_OFFLINE`, `PAID`, `PAID_OFFLINE`, `FAILED`, `EXPIRED`, `REFUNDED` |
| Evasione | `PENDING`, `CONFIRMED`, `PREPARING`, `READY_FOR_PICKUP`, `COMPLETED`, `CANCELLED`, `EXPIRED`, `REVIEW_REQUIRED` |

Il job riconcilia le sessioni Stripe e libera le riserve scadute. Un pagamento tardivo con stock già rilasciato resta in **Da verificare**. L'operatore può ispezionare lo storico e richiedere il rimborso Stripe. Lo Shop ammette prodotti su ordinazione e prenotazioni con o senza riserva fisica immediata.

## API e persistenza

In produzione, aggiungi il prefisso `/api` ai percorsi della tabella.

| Base | Endpoint |
| --- | --- |
| `/v1/clubs/{slug}/merch` | `GET /`, `POST /quote`, `GET/POST /orders`, `GET /orders/{id}`, `POST /orders/{id}/recovery/{start,verify}`, `GET /media/{id}`, `POST /events` |
| `/v1/club-admin/{slug}/merch` | `GET/PUT /settings`, `GET/POST /products`, `PUT/DELETE /products/{id}`, `POST /media`, `GET /orders`, `GET /orders/{id}`, `POST /orders/{id}/actions`, `GET /analytics` |
| `/merch/stripe` | `POST /webhook` |

Il backend applica la migrazione SQLite `service/api/merch_migrations/001_merch.sql` e lo schema della coda SMS all'avvio. Catalogo, immagini, ordini, riserve, movimenti, transazioni, storico ed eventi analytics sono isolati per società. Le righe di ordine conservano una copia dei dati commerciali. Le immagini sono salvate in SQLite e seguono il volume dati esistente. La vendita e le riserve usano transazioni serializzate, con idempotenza per richieste di ordine e webhook. Gli analytics separano il valore delle prenotazioni da pagare dagli incassi Stripe, al netto dei rimborsi registrati.

## File e componenti

| Area | File aggiunti | Integrazioni esistenti aggiornate |
| --- | --- | --- |
| Admin Vue | `admin-v2/views/MerchView.vue`, `MerchProductEditor.vue`, `MerchSettings.vue`, `MerchOrderDetail.vue`, `admin-v2/merchModel.js` | `AdminV2App.vue`, `V2Workspace.vue`, `V2Dialog.vue`, `V2Icon.vue`, `config/navigation.js`, `OrganizationDetail.vue` |
| Portale Vue | `src/merch/FanMerchShop.vue`, `FanMerchLink.vue`, `MerchCheckout.vue`, `MerchOrders.vue`, `MerchRecovery.vue`, `merchApi.js`, `shopModel.js`, `shop.css` | `App.vue`, `FanLiveCento.vue`, `OrganizationArena.vue`, `ActivitiesPanel.vue`, `ArenaIcon.vue` |
| Backend Go | `merch_http.go`, `merch_catalog.go`, `merch_orders.go`, `merch_lifecycle.go`, `merch_media.go`, `merch_stripe.go`, `merch_notifications.go`, `merch_recovery.go`, `merch_migrations/001_merch.sql` | `api.go`, `api-handler.go`, `bar.go`, `shutdown.go`, `cmd/webapi/main.go`, `cmd/webapi/cors.go` |
| Test | `merch_core_test.go`, `merch_stripe_test.go`, `merch_recovery_test.go`, `merchModel.test.js`, `shop.test.js`, `test/merch-browser.mjs` | `workspaceRendering.test.js` |
| Configurazione e guida | `docs/merchandising.md` | `.env.example`, `src/admin-v2/README.md` |

## Verifica locale

```sh
cd wcmvpvs-back
CGO_ENABLED=1 go test ./...
cd ../wcmvpvs-front
npm run build
npm run test:admin-v2
node --test src/merch/*.test.js
```

Per la prova della vetrina a 390 px e del dettaglio desktop, imposta `MERCH_TEST_BROWSER` sul percorso dell'eseguibile Chromium o Edge ed esegui `node test/merch-browser.mjs`. La prova usa un server locale con dati in memoria.
