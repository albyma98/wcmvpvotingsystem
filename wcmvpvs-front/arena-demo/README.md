# Joy Volley · Arena Tifosi

MVP di tribuna digitale popolata, costruito sul palazzetto esistente di Gioia del Colle. Usa Vue 3, Vite e Three.js. È una **pagina autonoma** in `/arena-demo/`, con entry point, stili e bundle propri: non richiede backend, login, API o dati degli eventi. L’app principale continua a essere servita da `/`.

## Come usare Arena Tifosi

1. All’apertura sono presenti **60 profili tifoso demo e 280 comparse**, distribuiti sui due lati lunghi. Il tuo profilo iniziale è **GioiaFan**, posto **B-08-24**. I dati sono dimostrativi.
2. Tocca il **profilo in basso a sinistra** per cambiare tifoso dal selettore oppure cercare un nickname (per esempio `MarcoGioia`). I risultati della ricerca sono selezionabili anche da tastiera. Toccare un avatar di un profilo demo consente a sua volta di impersonarlo; le comparse non sono selezionabili. La scelta chiude il pannello e avvicina la camera al nuovo posto.
3. **Trova il mio posto** avvicina dolcemente la camera all’avatar. Nome, settore, fila, posto, livello, partite e interazioni sono mostrati nel pannello; indicatore e badge segnalano il tuo posto nella scena.
4. **Vota MVP** apre tre giocatori inventati. Scegli un giocatore e conferma: il profilo vota una sola volta, il suo avatar si alza e applaude insieme ai vicini. Il voto resta salvato dopo una ricarica e il pulsante consente di consultarlo. È un voto locale di prova, senza effetto sulle partite reali.
5. La **barra di gioco sovrapposta al 3D** raccoglie una sola volta le quattro azioni: MVP, applauso, sciarpa e ola. Le ultime tre attivano reazioni di settore o una propagazione sulle due tribune. Cooldown di 4.5, 7 e 12 secondi per profilo limitano le ripetizioni e sono visibili sui pulsanti.
6. L’**icona impostazioni in alto a destra** apre un pannello a comparsa. **Scena** contiene i filtri per comparse e soli profili attivi e i dettagli del palazzetto. **Reset demo**, nel fondo del menu, cancella con conferma esclusivamente voti/interazioni della demo e mantiene gli stessi posti.
7. Nello stesso menu, **Viste** contiene i cinque preset architettonici e il tour cinematico di 18 secondi. Sono disponibili anche formato 9:16, schermo intero del browser, guida e fotografia originale.

Il 3D occupa automaticamente tutta la finestra, senza sidebar o scorrimento della pagina. Profilo, impostazioni e voto si aprono sopra la scena; su mobile i pannelli di profilo e impostazioni sono ancorati al bordo inferiore e scorrono internamente. La barra delle azioni lascia il posto ai comandi del tour durante la visita automatica.

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
- All’avvio il campo è libero per Arena Tifosi. La scheda **Scena** permette di riattivare le 36 sedie della fotografia, la copertura e le barriere perimetrali.
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
    props.js                sedie opzionali, tavoli e barriere
```

Le geometrie ripetute sono `InstancedMesh`, aggregate per materiale: sedie, sedute gialle, gradoni, ringhiere e travi non generano una draw call per pezzo. DPR limitato a 1.5 desktop / 1.25 smartphone, con un budget massimo di circa 2.2 milioni di pixel per il canvas a schermo pieno. Tre luci, niente ombre dinamiche o postprocessing. Texture principali generate localmente; la foto è servita da `public/arena/reference.jpg`. I font Google sono opzionali: senza rete vengono usati i fallback di sistema. Il rendering si sospende quando la scheda non è visibile; lo smontaggio rilascia controlli, observer, materiali, texture e geometrie.

La mappa astratta contiene 1330 posti: 665 per tribuna, senza occupare le scale. Profili e seggiolini leggono la stessa mappa. Un seed stabile e un campionamento pesato favoriscono zone centrali e cluster; le comparse privilegiano i dintorni dei profili e lasciano vuoti irregolari. I 340 avatar condividono cinque mesh instanziate per corpi, teste, capelli, arti/dettagli e sciarpe, più indicatori semplici. Nessun rig, GLB o texture esterna per i tifosi. Le pose vengono aggiornate al massimo a 30 Hz, soltanto per avatar idle selezionati e soggetti coinvolti nelle azioni; le comparse lontane ricevono aggiornamenti meno frequenti. Il rendering della camera mantiene il proprio ciclo fluido. `prefers-reduced-motion` disattiva i piccoli movimenti automatici; le azioni esplicite restano disponibili.

Nella build verificata, il bundle JavaScript dell’arena è circa **162 kB gzip** e il CSS circa **12 kB gzip**, esclusi runtime Vue condiviso, foto, font esterni e overhead HTTP. L’arena non è importata dalla pagina principale, che mantiene il suo bundle separato. Queste misure riguardano gli asset compilati, non un tempo di caricamento o una garanzia di FPS su telefono.

## Verifica e prossimi sviluppi ABX

```powershell
npm run test:arena
```

I 13 test coprono posti validi/univoci e riparazione dei salvataggi, persistenza, voto unico, cooldown, storage bloccato, reset, reazioni di vicinato, propagazione dell’ola, matrici delle pose e traiettoria cinematica.

Verifiche eseguite per Arena Tifosi: build di produzione dell’intero frontend e **13 test superati**; ricerca di `MarcoGioia`, richiamo camera e posto A-12-23; voto per Luca Rinaldi e conferma della persistenza dopo ricarica, con secondo voto bloccato; applauso, sciarpa e ola; filtri con contatori 340 → 60 → 46; rotazione, zoom e tour completo; nessun errore o warning nella console nel controllo finale. Layout e finestra MVP verificati a 390 × 844 senza overflow orizzontale. Il formato 9:16 mantiene l’implementazione già collaudata nella versione architettonica. Non è stato effettuato un collaudo su telefoni fisici.

La disposizione aggiornata è stata verificata anche sulle geometrie generate: due tribune con 19 gradoni e dimensioni identiche, entrambe parallele al lato lungo, nessuna tribuna sui lati corti, due porte per ciascuna parete di fondo e bandiere sopra la tribuna senza seggiolini. Tour e build sono stati ricontrollati dopo lo spostamento.

Interfaccia immersiva verificata nel browser a **1280 × 900** e **390 × 844**: canvas a tutta finestra, pagina senza scorrimento, profilo e ricerca a comparsa, accesso al voto dalla barra e voto precedente conservato. Screenshot aggiornati: [desktop a tutto schermo](screenshots/immersive-desktop.jpg), [mobile a tutto schermo](screenshots/immersive-mobile.jpg) e [MVP su mobile](screenshots/immersive-vote-mobile.jpg). Restano conservati anche gli screenshot delle precedenti versioni.

`createArena.js` mantiene i gruppi `crowd` (ora popolato), `sponsors` e `interaction-zones`. Il picking della folla avviene soltanto su click/tap breve, non su ogni frame; drag e gesture a più dita non selezionano avatar.

Per l’integrazione ABX reale, sostituire `demoSession.js` con un adapter autenticato che carichi l’identità tifoso e il roster della partita, mantenga il medesimo `seatId` lato server e invii il voto alle API ufficiali con validazione e idempotenza server. La presenza e le azioni collettive possono essere sincronizzate tramite SSE/WebSocket, mantenendo le animazioni separate in `crowdDirector.js`. Filler e profili reali restano due classi distinte: i primi non incrementano le presenze reali del prodotto.

La demo non è multiplayer, non certifica presenze e non sincronizza voti fra dispositivi o tab concorrenti. Non include audio o registrazione video. Il collaudo responsive nel browser non sostituisce la misura di FPS, tempi di avvio e consumo su iPhone/Android reali.

Per sostituire elementi low-poly con GLB: caricare il modello con `GLTFLoader`, usare metri e asse Y verticale, mantenerlo nel relativo gruppo e applicare le stesse trasformazioni. Per molte sedute/avatar, mantenere geometria e materiali condivisi con instancing; per animazioni individuali valutare livelli di dettaglio e un numero limitato di scheletri. Branding e palette sono centralizzati in `scene/primitives.js`; le texture del campo in `scene/field.js`.
