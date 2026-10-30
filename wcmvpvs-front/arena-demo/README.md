# Joy Volley · Arena Tifosi

## Arena Tifosi 3.0 — cinque attività nello stesso palazzetto

La demo 3.0 si apre su **`/arena-demo/`**. La versione collegata ai profili reali rimane su **`/<slug>/arena`**. Apri il pulsante con la coppa **Attività Arena** in alto: le cinque attività condividono scena, avatar, wallet e progressi. La demo architettonica precedente, con il selettore di profili, rimane su `/arena-demo/?classic=1`.

### Come provare le attività

1. **Caccia al tesoro.** Apri la missione, leggi l’indizio e premi **Cerca il tesoro**. Ruota/avvicina la camera e tocca il bagliore 3D. I quattro modelli sono pallone dorato, monetina, mini coppa e sciarpa luminosa; ruotano fra dieci posizioni. Una raccolta dà inizialmente 20 monetine ABX. Riaprire o ricaricare non paga di nuovo. Nel laboratorio di sviluppo puoi mostrare la posizione, simulare il ritrovamento, ripristinare la missione o cambiare giorno/posizione.
2. **Guardaroba.** La camera raggiunge il tuo avatar e puoi ruotargli intorno trascinando la scena. Tocca un accessorio per provarlo; **Acquista** e poi **Equipaggia** lo rendono permanente. Con le 20 monetine del primo tesoro puoi comprare subito maglia, cappellino o sciarpa. Il catalogo contiene 12 accessori visibili: abiti, pantaloni, sciarpa, cappellino, occhiali, bandierina, scarpe, capelli, colore maglia ed esultanza. La rotazione settimanale è stabile per settimana ISO in Europe/Rome; la maglia alternativa è legata alla settimana di gara, due oggetti richiedono livello 2. I posseduti restano utilizzabili anche dopo la fine della disponibilità.
3. **Pronostici.** Tocca il tabellone a bordo campo oppure apri l’attività. Il primo numero indica sempre la prima squadra mostrata. Conferma uno dei sei risultati in set; puoi modificarlo prima del cutoff. Nella demo, scegli lo stesso risultato nel laboratorio e premi **Conferma risultato e verifica**: ricevi 50 punti ABX per l’esatto o 15 per il solo vincitore. Ripetere non assegna altri punti. Sono punti della classifica, non monetine spendibili né una nuova valuta. La versione reale legge gli eventi della società e aspetta la conferma amministrativa del risultato ufficiale.
4. **I miei vicini.** La camera raggiunge il tuo posto ed evidenzia gli avatar vicini. La ricerca e le schede rispettano l’opzione di pubblicità del profilo. Genera e copia il link d’invito: il destinatario accede con la propria sessione e conferma la richiesta di posto. È una richiesta esplicita di ricollocazione del solo destinatario; nessun altro tifoso viene spostato. Se i posti immediatamente vicini sono occupati viene scelto un posto libero nella stessa tribuna. Il laboratorio può aggiungere un amico demo senza cambiare i posti esistenti.
5. **Museo.** Le sei vetrine sul fondo del campo sono oggetti 3D selezionabili. Le coppe dorate sono sbloccate, quelle chiare restano da conquistare; targhe e pannello mostrano descrizione e data. La demo contiene due esempi dichiarati dimostrativi e permette di sbloccarne altri. La versione reale registra esclusivamente traguardi misurati e conserva data e valore del primo sblocco.

MVP, applauso, sciarpa, ola, ricerca tifosi, inquadrature e navigazione touch restano disponibili. Nell’arena reale MVP rimanda alla votazione esistente della società; nella demo il voto resta dimostrativo e persistente. Il laboratorio si trova in fondo al pannello attività, soltanto con `import.meta.env.DEV` e nell’adapter demo. Non esiste un endpoint pubblico per assegnare monetine, alterare il giorno o sbloccare trofei.

### Economia e persistenza server

Le migrazioni additive vengono eseguite all’avvio del backend. `arena_members` rende persistenti posto, opt-in pubblico ed equipaggiamento, con unicità `(org,fan)` e `(org,slot)`. Gli avatar dei profili riservati restano visibili, ma nickname e scheda pubblica sono nascosti; il proprietario vede sempre la propria identità. Le vecchie assegnazioni calcolate solo nel browser vengono inizializzate una volta e da quel momento non si compattano dopo cancellazioni.

`arena_claims` ha unicità club/tifoso/giorno; il server sceglie tesoro e posizione da club e giorno **Europe/Rome**, usando il proprio orologio e dati timezone inclusi nel binario. `arena_predictions` ha unicità club/tifoso/partita e una ricompensa liquidata una sola volta, anche se il risultato viene confermato ripetutamente. `arena_invites` usa codici casuali opachi, validi 30 giorni e utilizzabili da un solo destinatario. Un link non contiene credenziali e non seleziona l’identità dell’invitante.

Acquisti e premi usano **`fan_wallets`, `fan_economy_accounts`, `fan_wallet_movements`, `fan_shop_items`** e i trigger economici già esistenti. Gli articoli avatar sono oggetti non consumabili `arena:<key>` nella collezione ABX, con scope club. Nessun saldo parallelo: la transazione prende il lock di scrittura della società prima di leggere e aggiorna insieme proprietà, saldo e registro. I punti da pronostico aumentano esclusivamente il contatore punti ABX. I client non forniscono prezzi, premi, identità del proprietario o orari attendibili.

### API e configurazione amministrativa

- `GET /api/fan/arena/activities`: stato, missione giornaliera, catalogo, saldo del proprietario, eventi, membri e trofei.
- `POST /api/fan/arena/activities`: azioni `claim`, `buy`, `equip`, `remove`, `predict`, `privacy`, `invite`, `accept`, `wave`, `clap`. Ogni mutazione richiede una sessione tifoso valida della medesima organizzazione.
- `GET /api/fan/arena`: roster pubblico, posto persistente, equipaggiamento e `is_me` verificato dal server.
- `POST /api/admin/arena/activities`: soltanto amministratori autorizzati della società; azioni `config` e `result`, protette dal middleware admin esistente. I tifosi non possono chiamarle tramite il percorso pubblico.

Tutte le richieste reali usano `X-Organization-Slug`, `X-Device-ID` e, quando autenticati, `X-Fan-Session`. Gli aggiornamenti restituiscono lo stato effettivo e sono `no-store`. Il completamento dell’ola viene confermato con `wave-complete` dopo almeno 8 secondi misurati dal server; le conferme anticipate non contano. Esempi di body amministrativi (usare la sessione admin esistente):

```json
{"action":"config","config":{"treasure":20,"exact":50,"winner":15,"growth":100,"growth_big":500,"challenge":100,"collective":3,"record":10,"cutoff_minutes":0}}
```

```json
{"action":"result","event_id":123,"score":"3-1","confirm":true}
```

Il cutoff è l’inizio ufficiale meno `cutoff_minutes` (0 per la chiusura all’inizio). La conferma del risultato è accettata solo dopo l’inizio; dopo una prima conferma non è possibile cambiare il risultato e riassegnare i premi. Non sono inventati risultati sportivi reali. Per una correzione ufficiale successiva serve una procedura amministrativa di storno, fuori dal flusso corrente.

I traguardi reali sono: soglie di profili completati (100/500 di default); prima ola con almeno 3 identità distinte entro 10 secondi; 100 partecipanti unici alla caccia; tribuna vincitrice della settimana precedente per cacciatori unici, con partecipazione di entrambi i settori e almeno `collective` partecipanti totali; record giornaliero di partecipazione sopra la soglia configurata. Non esiste una competizione precedente da riutilizzare nel modulo Arena: queste misure sono definite sulle attività realmente registrate. I trofei non possono essere sbloccati arbitrariamente attraverso le API pubbliche.

### Demo, test e limiti

L’adapter demo usa una sola chiave locale, **`abx.arena.activities.demo.v3`**, con saldo dimostrativo e progressi separati dai profili reali. Giorno simulato, premi manuali e risultati demo non hanno valore nel backend ABX. LocalStorage non è una barriera antifrode: tutte le garanzie economiche reali appartengono alle transazioni server. **Ripristina dati demo** cancella soltanto questa simulazione. La vecchia demo conserva la sua chiave separata.

```powershell
cd wcmvpvs-front
npm run dev
# aprire /arena-demo/ sulla porta stampata da Vite
npm run test:arena
npm run build
```

Backend: `CGO_ENABLED=1 go test ./service/database ./service/api` con GCC/SQLite disponibili. Sul PC di sviluppo è disponibile anche Ubuntu tramite WSL; i test sono stati avviati lì per usare SQLite reale. I test specifici `-run Arena` coprono giorno/DST, transazioni, richieste ripetute e concorrenti, fondi insufficienti, inventario, cutoff, liquidazione unica, inviti, privacy e confini admin. Il frontend verifica anche trasformazioni 3D degli accessori, posti e regressioni delle animazioni.

Il museo e gli hotspot condividono il renderer dell’arena, con geometrie semplici, texture generate localmente e materiali condivisi. Il guardaroba riusa gli avatar instanziati. Il pannello e gli oggetti delle attività sono moduli caricati su richiesta; il DPR resta limitato. Nessuna nuova dipendenza, servizio esterno o acquisto con denaro reale.

Verifica 3.0 completata: build di produzione e **23 test frontend superati**; suite database completa, suite API completa e test Arena aggiornati superati con SQLite reale su Ubuntu/WSL. La fixture del test CRUD torneo è stata allineata alle migrazioni P1 già usate dal router. Controllato anche che il modulo del laboratorio non sia presente nei bundle di produzione. Nel browser sono stati provati ritrovamento del tesoro tramite tap 3D, acquisto/equipaggiamento del cappellino, persistenza dopo refresh, pronostico premiato e chiuso, invito e amico demo, museo e MVP. Layout controllato a 390 × 844 e 1280 × 900; console pulita nella sessione finale. Screenshot: [attività desktop](screenshots/activities-desktop.jpg), [guardaroba mobile](screenshots/activities-wardrobe-mobile.jpg), [museo mobile](screenshots/activities-museum-mobile.jpg). I dati usati per la prova demo sono stati ripristinati dal solo laboratorio locale.

Le reazioni visive restano locali; l’ola viene registrata sul server per misurare il traguardo ma non è ancora trasmessa agli altri browser. Il risultato ufficiale va confermato dall’API amministrativa: un eventuale feed sportivo futuro dovrà alimentare quel passaggio. La capienza è 1330 posti; oltre la capienza serve un’estensione della mappa. Non è stato eseguito un collaudo su iPhone/Android fisici. La build richiede il rilascio coordinato di frontend e backend, con fallback SPA per `/<slug>/arena`.

## Arena delle società

L’app principale espone ora **`/<slug>/arena`**, caricata su richiesta. Usa `GET /api/fan/arena` con lo slug della società e la sessione tifoso già esistente. Il server risolve l’`organization_id`, estrae esclusivamente i profili con registrazione completata e termini accettati, e restituisce ID, nickname autorizzato, posto, equipaggiamento e indicazione del proprio profilo. Telefono, sessioni e altri dati personali non sono inclusi.

La tribuna contiene soltanto avatar associati a profili reali della società, con posti persistenti sul server. La mappa ha 1330 posti; i profili si aggiornano all’apertura e dopo le azioni. Non viene simulato uno stato online. La ricerca pubblica richiede il consenso del tifoso.

Gli ospiti possono esplorare e cercare nickname. La sessione valida della stessa società abilita il proprio avatar e le reazioni locali; selezionare un altro tifoso ne mostra il posto senza impersonarlo. Il pulsante MVP porta al percorso tifoso esistente `/<slug>/new`, dove resta gestita la votazione ufficiale. Le reazioni non sono ancora sincronizzate fra utenti. Gli stili della pagina sono isolati e la scena viene rilasciata all’uscita. Il percorso `/arena-demo/` conserva la demo indipendente.

L’integrazione richiede il rilascio sia del frontend sia del backend aggiornati e il normale fallback del server SPA a `index.html` per `/<slug>/arena`.

MVP di tribuna digitale popolata, costruito sul palazzetto esistente di Gioia del Colle. Usa Vue 3, Vite e Three.js. È una **pagina autonoma** in `/arena-demo/`, con entry point, stili e bundle propri: non richiede backend, login, API o dati degli eventi. L’app principale continua a essere servita da `/`.

## Come usare la demo classica (`?classic=1`)

1. All’apertura sono presenti **60 profili tifoso demo e 280 comparse**, distribuiti sui due lati lunghi. Il tuo profilo iniziale è **GioiaFan**, posto **B-08-24**. I dati sono dimostrativi.
2. Tocca il **profilo in basso a sinistra** per cambiare tifoso dal selettore oppure cercare un nickname (per esempio `MarcoGioia`). I risultati della ricerca sono selezionabili anche da tastiera. Toccare un avatar di un profilo demo consente a sua volta di impersonarlo; le comparse non sono selezionabili. La scelta chiude il pannello e avvicina la camera al nuovo posto.
3. **Trova il mio posto** avvicina dolcemente la camera all’avatar. Nome, settore, fila, posto, livello, partite e interazioni sono mostrati nel pannello; indicatore e badge segnalano il tuo posto nella scena.
4. **Vota MVP** apre tre giocatori inventati. Scegli un giocatore e conferma: il profilo vota una sola volta, il suo avatar si alza e applaude insieme ai vicini. Il voto resta salvato dopo una ricarica e il pulsante consente di consultarlo. È un voto locale di prova, senza effetto sulle partite reali.
5. La **barra di gioco sovrapposta al 3D** raccoglie una sola volta le quattro azioni: MVP, applauso, sciarpa e ola. Le ultime tre attivano reazioni di settore o una propagazione sulle due tribune. Cooldown di 4.5, 7 e 12 secondi per profilo limitano le ripetizioni e sono visibili sui pulsanti.
6. L’**icona impostazioni in alto a destra** apre un pannello a comparsa. **Scena** contiene i filtri per comparse e soli profili attivi e i dettagli del palazzetto. **Reset demo**, nel fondo del menu, cancella con conferma esclusivamente voti/interazioni della demo e mantiene gli stessi posti.
7. Nello stesso menu, **Viste** contiene i cinque preset architettonici e il tour cinematico di 18 secondi. Sono disponibili anche formato 9:16, schermo intero del browser, guida e fotografia originale.

Il 3D occupa automaticamente tutta la finestra, senza sidebar o scorrimento della pagina. Profilo, impostazioni e voto si aprono sopra la scena; su mobile i pannelli di profilo e impostazioni sono ancorati al bordo inferiore e scorrono internamente. La barra delle azioni lascia il posto ai comandi del tour durante la visita automatica.

Sul campo si svolge uno **scambio automatico di pallavolo 6 contro 6**: squadra blu e squadra gialla, ricezioni, alzate, schiacciate e muri sincronizzati con il pallone. La rete è sulla linea centrale, con nastro a 2.43 m, pali imbottiti blu e antenne. Lo scambio coreografato si ripete senza aggiungere comandi alla barra. **Viste → Campo centrale** offre un’inquadratura ravvicinata.

Profili, assegnazioni, voto, cooldown e azione recente usano soltanto la chiave `abx.arena-tifosi.demo.v1` in localStorage. Il salvataggio vale per origine e browser: `localhost` e `127.0.0.1` hanno archivi distinti. Se lo storage è bloccato, la demo continua in memoria e lo segnala. Le presenze attive sono simulate; impersonare un profilo lo rende attivo nella demo.

## Avvio dal repository

Richiede Node.js 22.12+ oppure una versione LTS compatibile con Vite 7.

```powershell
cd wcmvpvs-front
npm install
npm run dev
```

Apri **http://localhost:5173/arena-demo/**. Se la porta è occupata, usa quella stampata da Vite.

Per smartphone: PC e telefono sulla stessa rete Wi-Fi, poi apri `http://IP-DEL-PC:5173/arena-demo/`. Vite ascolta già su `0.0.0.0`: usa l’indirizzo **Network** stampato nel terminale. L’indirizzo può cambiare a ogni connessione. L’accessibilità da altri dispositivi dipende dalla rete locale e dalle autorizzazioni già configurate per Node sul PC.

## Build

```powershell
npm run build
npm run preview -- --host 0.0.0.0
```

Apri `/arena-demo/` sull’indirizzo stampato dall’anteprima (normalmente `http://localhost:4173/arena-demo/`). La build produce sia `dist/index.html`, per l’app esistente, sia `dist/arena-demo/index.html`. Su un server statico, conserva questa struttura e il contenuto di `dist/assets` e `dist/arena`.

## Controlli

- Mouse sinistro / un dito: rotazione con inerzia.
- Rotella / pinch: zoom limitato; tasto destro / due dita: pan limitato.
- Cinque preset animati: vista generale, tribuna destra, tribuna sinistra, campo, angolo alto.
- All’avvio sono visibili rete, pallone e dodici giocatori. La scheda **Scena** permette di riattivare le 36 sedie della fotografia, la copertura e le barriere perimetrali. L’allestimento con le sedie sostituisce temporaneamente la partita; disattivandolo riappaiono rete e squadre.
- **Avvia tour cinematico**: percorso automatico di 18 secondi, panoramica, tribuna principale, campo, sedute gialle e vista finale. I pulsanti Interrompi e i preset interrompono il tour. Il cambio scheda del browser lo ferma.
- Scorciatoie desktop: **1** MVP, **2** applauso, **3** sciarpa, **4** ola, **F** trova il mio posto. Sono sospese mentre è aperta una finestra o si scrive nei campi.
- **Formato 9:16** nel menu: viewport in rapporto verticale. La scena usa già tutta la finestra per impostazione predefinita; **Schermo intero** nasconde anche l’interfaccia del browser quando la Fullscreen API è disponibile. Tutti i pannelli rimangono accessibili anche in questa modalità.
- Foto di partenza ingrandibile in una finestra, chiudibile anche con Esc.
- Preset, tab, toggle e pulsanti sono utilizzabili da tastiera. Le transizioni tra preset rispettano `prefers-reduced-motion`; il tour parte solo su richiesta.

## Fedeltà e limiti

I riferimenti principali restano la gradinata chiara con due scale arancioni, le ringhiere blu, le pareti a pannelli beige, il parquet, le porte blu e il tetto scuro in legno con travi. La disposizione è stata aggiornata secondo le indicazioni successive alla prima demo:

- Due tribune speculari sui **lati lunghi** del campo da pallavolo, con la stessa struttura di 19 gradoni della tribuna grigia originale.
- Una tribuna senza seggiolini; quella opposta con molti seggiolini gialli e blu, mantenendo libere le scale arancioni.
- Nessuna tribuna sui **lati corti**, con **due porte blu su ciascuna parete di fondo**, quattro in totale.
- Bandiere e barra delle luci sopra la tribuna senza seggiolini.
- Preset e tour riposizionati sulle due nuove tribune. Pareti vicine e tetto si aprono nelle viste esterne per consentire lo spaccato.

È una ricostruzione interpretativa da **una sola fotografia**, corretta con le indicazioni del committente, senza misure certificate. Dimensioni e dettagli non specificati restano plausibili e semplificati. Il marchio sul parquet non è un logo ufficiale. Non si tratta di un simulatore con collisioni architettoniche. Le luci sul parquet sono sfumature precalcolate, non riflessi fisici.

Il formato verticale prepara l’inquadratura: **non esporta un video**. Per un reel si può registrare lo schermo durante il tour. La compatibilità touch è implementata tramite OrbitControls; le prestazioni su iPhone e Android reali richiedono verifica sui dispositivi di destinazione.

## Organizzazione

```text
src/arena/
  main.js                   entry indipendente
  ArenaApp.vue              shell e stato dei layer
  ArenaIcon.vue             icone SVG
  arena.css                 stili responsive isolati dall’app principale
  fans.css                  estensione UI Arena Tifosi
  immersive.css             scena a schermo pieno, HUD e pannelli a comparsa
  seating/seatMap.js        posti, scale escluse, seed e cluster pesati
  crowd/
    demoProfiles.js         60 identità demo e metadati
    createCrowd.js          geometrie instanziate, pose, evidenza e picking
    createCrowd.test.js     validità delle pose e filtri
  actions/
    demoSession.js          persistenza locale, voto unico, cooldown e reset
    crowdDirector.js        reazioni di vicinato e propagazione dell’ola
    demoSession.test.js     seating, persistenza, voto e azioni
  mvp/players.js            tre giocatori fittizi
  ui/
    FanPanel.vue            ricerca, identità e posto
    MvpDialog.vue           selezione e conferma del voto
  camera/
    cameraSystem.js         controlli, limiti e preset
    cinematic.js            traiettorie e clock del tour
    cinematic.test.js       regressione del clock e validità del percorso
  scene/
    createArena.js          lifecycle, renderer, resize e cleanup GPU
    layout.js               dimensioni, lati lunghi/corti, gradoni e porte
    primitives.js           batching instanziato e materiali
    environment.js          pareti, porte, tetto, bandiere e luci
    stands.js               due tribune e corrimano
    field.js                parquet e segnaletica su texture canvas
    volleyballMatch.js      rete, atleti instanziati, pallone e ombre leggere
    volleyballRally.js      traiettoria continua e pose dei dodici giocatori
    volleyballRally.test.js continuità, passaggio sopra la rete e limiti del campo
    props.js                sedie opzionali, tavoli e barriere
```

Le geometrie ripetute sono `InstancedMesh`, aggregate per materiale: sedie, sedute gialle, gradoni, ringhiere e travi non generano una draw call per pezzo. DPR limitato a 1.5 desktop / 1.25 smartphone, con un budget massimo di circa 2.2 milioni di pixel per il canvas a schermo pieno. Tre luci, niente ombre dinamiche o postprocessing. Texture principali generate localmente; la foto è servita da `public/arena/reference.jpg`. I font Google sono opzionali: senza rete vengono usati i fallback di sistema. Il rendering si sospende quando la scheda non è visibile; lo smontaggio rilascia controlli, observer, materiali, texture e geometrie.

La mappa astratta contiene 1330 posti: 665 per tribuna, senza occupare le scale. Profili e seggiolini leggono la stessa mappa. Un seed stabile e un campionamento pesato favoriscono zone centrali e cluster; le comparse privilegiano i dintorni dei profili e lasciano vuoti irregolari. I 340 avatar condividono batch instanziati per parti e acconciature, con due livelli di dettaglio descritti sotto. Nessun GLB o texture esterna da scaricare per i tifosi. Le pose dettagliate si aggiornano al massimo a 30 Hz, quelle lontane a 15 Hz; gli idle coinvolgono solo un sottoinsieme della folla. Il rendering della camera mantiene il proprio ciclo fluido. `prefers-reduced-motion` disattiva i piccoli movimenti automatici; le azioni esplicite restano disponibili.

Nella build verificata, il bundle JavaScript dell’arena è circa **165 kB gzip** e il CSS circa **12 kB gzip**, esclusi runtime Vue condiviso, foto, font esterni e overhead HTTP. L’arena non è importata dalla pagina principale, che mantiene il suo bundle separato. Queste misure riguardano gli asset compilati, non un tempo di caricamento o una garanzia di FPS su telefono.

## Verifica e prossimi sviluppi ABX

```powershell
npm run test:arena
```

I 27 test coprono posti validi/univoci e riparazione dei salvataggi, persistenza, voto unico, cooldown, storage bloccato, reset, reazioni di vicinato, propagazione dell’ola, matrici delle pose, traiettoria cinematica, attività, aspetto e LOD. I tre test della partita controllano sei atleti per metà campo, pose finite, continuità di tutti i contatti e del ciclo, passaggio del pallone sopra il nastro e pose ferme con movimento ridotto.

Gli atleti condividono una mesh instanziata per corpi e arti e una per le teste. Ombre morbide precalcolate, pallone e rete usano geometrie/materiali condivisi, senza modelli o texture scaricati. Le pose e il pallone si aggiornano al massimo a 30 Hz; il ciclo si sospende con la scena nascosta. Con `prefers-reduced-motion` la partita resta in posa statica. Verificati panoramica, vista Campo centrale e viewport mobile nel browser. Screenshot: [campo con rete e squadre](screenshots/volleyball-desktop.jpg), [partita su mobile](screenshots/volleyball-mobile.jpg).

Verifiche eseguite per Arena Tifosi: build di produzione dell’intero frontend e **13 test superati**; ricerca di `MarcoGioia`, richiamo camera e posto A-12-23; voto per Luca Rinaldi e conferma della persistenza dopo ricarica, con secondo voto bloccato; applauso, sciarpa e ola; filtri con contatori 340 → 60 → 46; rotazione, zoom e tour completo; nessun errore o warning nella console nel controllo finale. Layout e finestra MVP verificati a 390 × 844 senza overflow orizzontale. Il formato 9:16 mantiene l’implementazione già collaudata nella versione architettonica. Non è stato effettuato un collaudo su telefoni fisici.

La disposizione aggiornata è stata verificata anche sulle geometrie generate: due tribune con 19 gradoni e dimensioni identiche, entrambe parallele al lato lungo, nessuna tribuna sui lati corti, due porte per ciascuna parete di fondo e bandiere sopra la tribuna senza seggiolini. Tour e build sono stati ricontrollati dopo lo spostamento.

Interfaccia immersiva verificata nel browser a **1280 × 900** e **390 × 844**: canvas a tutta finestra, pagina senza scorrimento, profilo e ricerca a comparsa, accesso al voto dalla barra e voto precedente conservato. Screenshot aggiornati: [desktop a tutto schermo](screenshots/immersive-desktop.jpg), [mobile a tutto schermo](screenshots/immersive-mobile.jpg) e [MVP su mobile](screenshots/immersive-vote-mobile.jpg). Restano conservati anche gli screenshot delle precedenti versioni.

`createArena.js` mantiene i gruppi `crowd` (ora popolato), `sponsors` e `interaction-zones`. Il picking della folla avviene soltanto su click/tap breve, non su ogni frame; drag e gesture a più dita non selezionano avatar.

Per l’integrazione ABX reale, sostituire `demoSession.js` con un adapter autenticato che carichi l’identità tifoso e il roster della partita, mantenga il medesimo `seatId` lato server e invii il voto alle API ufficiali con validazione e idempotenza server. La presenza e le azioni collettive possono essere sincronizzate tramite SSE/WebSocket, mantenendo le animazioni separate in `crowdDirector.js`. Filler e profili reali restano due classi distinte: i primi non incrementano le presenze reali del prodotto.

La demo non è multiplayer, non certifica presenze e non sincronizza voti fra dispositivi o tab concorrenti. Non include audio o registrazione video. Il collaudo responsive nel browser non sostituisce la misura di FPS, tempi di avvio e consumo su iPhone/Android reali.

Per sostituire elementi low-poly con GLB: caricare il modello con `GLTFLoader`, usare metri e asse Y verticale, mantenerlo nel relativo gruppo e applicare le stesse trasformazioni. Per molte sedute/avatar, mantenere geometria e materiali condivisi con instancing; per animazioni individuali valutare livelli di dettaglio e un numero limitato di scheletri. Branding e palette sono centralizzati in `scene/primitives.js`; le texture del campo in `scene/field.js`.

## Avatar cartoon heroic · ottobre 2026

Il nuovo modello condiviso da demo e `/slug/arena` ha spalle più ampie, vita rastremata, testa moderatamente ingrandita, occhi con iride e riflessi, sopracciglia, naso, sorriso/espressioni, orecchie, quattro acconciature di base, tre corporature, mani con pollice/dita e sneakers con suole, linguetta e lacci. Superfici e materiali restano stilizzati; non sono richiesti modelli scaricati. Il guardaroba mantiene i suoi capi, accessori e azioni.

- `crowd/avatarModel.js`: geometrie condivise e materiali, batch dei due LOD, UV del torso Lathe e hook `setJerseyTexture(texture)` per una futura maglia ufficiale. La texture coinvolge soltanto il materiale della maglia; nessun marchio ufficiale inventato.
- `crowd/avatarPose.js`: stessa posa e stessi punti di articolazione nei due LOD. In piedi l’avatar avanza davanti alla seduta. Posti e identità restano invariati.
- `crowd/avatarLook.js`: varianti deterministiche dal seed del profilo. Le scelte libere vengono salvate nel JSON equipment esistente (`look-hair`, `look-body`, `look-skin`, `look-hairColor`, `look-shirt`, `look-expression`). Il comando `appearance` convalida chiavi e intervalli per il tifoso autenticato della società. Nessuna migrazione o modifica a saldo, inventario e posto; i capi equipaggiati hanno precedenza sul colore base.
- `crowd/createCrowd.js`: avatar selezionato e vicini in LOD 0, massimo 8 su desktop e 4 sotto 700 px. Ingresso entro 6 m, uscita oltre 8 m, rivalutazione ogni 140 ms per limitare i cambi. Il resto della folla usa il LOD 1 instanziato. Durante l’ola si aggiornano i partecipanti raggiunti dall’onda; il dettaglio alto non viene duplicato per tutta la folla.

In sviluppo apri **`/avatar-studio`**: rotazione e zoom, aspetto e guardaroba, posa seduta/in piedi, applauso/sciarpa/ola/esultanza, confronto forzato dei LOD e scena da 340 avatar. Le scelte dello studio restano nel solo laboratorio. La pagina è esclusa dalla build di produzione; i controlli di aspetto reali si trovano in **Attività → Il mio guardaroba → Aspetto del mio avatar**.

Verifica browser a 1280 × 900: scena da 340 avatar e 8 dettagliati, circa **519 mila triangoli** e **39 draw call** nella vista di prova, 60 FPS osservati dopo l’assestamento; aggiornamento pose intorno a 3 ms nel campione idle. Nell’ola sono stati osservati circa 4–6 ms e 60 FPS dopo l’avvio. Queste misure dipendono dal PC/browser, non sono una garanzia su telefoni. Verificato anche il layout 390 × 844 senza overflow orizzontale. I test Node impongono il limite dei LOD, matrici finite, budget sotto 550 mila triangoli e persistenza senza alterare l’economia; i test Go Arena verificano anche il salvataggio reale dell’aspetto e il rifiuto di valori non validi.

Screenshot: [avatar nello studio](screenshots/avatar-studio-hero.jpg), [studio mobile](screenshots/avatar-studio-mobile.jpg), [340 avatar](screenshots/avatar-crowd-340.jpg), [guardaroba nell’arena](screenshots/avatar-arena-wardrobe.jpg). La build frontend e i test Go Arena passano; restano da misurare FPS, consumo e avvio su iPhone/Android fisici.
