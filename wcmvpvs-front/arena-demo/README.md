# Joy Volley · Arena Lab

MVP 3D navigabile ispirato alla foto fornita del palazzetto di Gioia del Colle. Usa Vue 3, Vite e Three.js, riutilizzando il frontend esistente. È una **pagina autonoma** in `/arena-demo/`, con entry point, stili e bundle propri: non richiede backend, login, API o dati degli eventi. L’app principale continua a essere servita da `/`.

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
- **Come nella foto**: sedie sparse visibili all’avvio. **Giorno evento**: campo libero.
- Scheda **Scena**: layer delle sedie, copertura in legno e barriere perimetrali.
- **Avvia tour cinematico**: percorso automatico di 18 secondi, panoramica, tribuna principale, campo, sedute gialle e vista finale. I pulsanti Interrompi e i preset interrompono il tour. Il cambio scheda del browser lo ferma.
- Icona telefono: viewport realmente in rapporto 9:16. Reset e schermo intero nella stessa toolbar; dove la Fullscreen API non è disponibile, il pulsante propone il formato verticale.
- Foto di partenza ingrandibile in una finestra, chiudibile anche con Esc.
- Preset, tab, toggle e pulsanti sono utilizzabili da tastiera. Le transizioni tra preset rispettano `prefers-reduced-motion`; il tour parte solo su richiesta.

## Fedeltà e limiti

I riferimenti principali sono la tribuna alta chiara con due scale arancioni, la tribuna bassa gialla, le ringhiere blu, le pareti a pannelli beige, le porte blu, il tetto scuro in legno con travi, la fascia di finestre alte, le luci e le bandiere. Il parquet include linee sportive e un logo centrale stilizzato. La vista iniziale cerca la composizione della foto, con campo e sedie in primo piano, gradinata grigia a destra e sedute gialle in basso.

È una ricostruzione interpretativa da **una sola fotografia**, senza misure certificate. Dimensioni, lati non visibili, sequenza delle porte, bandiere e numero di sedie sono plausibili e semplificati. Il marchio sul parquet non è un logo ufficiale. Le pareti e le falde sono visibili dall’interno e aperte dall’esterno per consentire viste a spaccato; non si tratta di un simulatore con collisioni architettoniche. Le luci sul parquet sono sfumature precalcolate, non riflessi fisici.

Il formato verticale prepara l’inquadratura: **non esporta un video**. Per un reel si può registrare lo schermo durante il tour. La compatibilità touch è implementata tramite OrbitControls; le prestazioni su iPhone e Android reali richiedono verifica sui dispositivi di destinazione.

## Organizzazione

```text
src/arena/
  main.js                   entry indipendente
  ArenaApp.vue              shell e stato dei layer
  ArenaIcon.vue             icone SVG
  arena.css                 stili responsive isolati dall’app principale
  camera/
    cameraSystem.js         controlli, limiti e preset
    cinematic.js            traiettorie e clock del tour
    cinematic.test.js       regressione del clock e validità del percorso
  scene/
    createArena.js          lifecycle, renderer, resize e cleanup GPU
    primitives.js           batching instanziato e materiali
    environment.js          pareti, porte, tetto, bandiere e luci
    stands.js               due tribune e corrimano
    field.js                parquet e segnaletica su texture canvas
    props.js                sedie opzionali, tavoli e barriere
```

Le geometrie ripetute sono `InstancedMesh`, aggregate per materiale: sedie, sedute gialle, gradoni, ringhiere e travi non generano una draw call per pezzo. DPR limitato a 1.75 desktop / 1.25 smartphone, tre luci, niente ombre dinamiche o postprocessing. Texture principali generate localmente; la foto è servita da `public/arena/reference.jpg`. I font Google sono opzionali: senza rete vengono usati i fallback di sistema. Il rendering si sospende quando la scheda non è visibile; lo smontaggio rilascia controlli, observer, materiali, texture e geometrie.

## Verifica e prossimi sviluppi ABX

```powershell
node --test src/arena/camera/cinematic.test.js
```

Il test copre il primo frame che precede il click, la validità delle pose durante tutti i 18 secondi e un frame ritardato oltre il termine.

Verifiche eseguite: build di produzione dell’intero frontend, 3 test cinematici superati, risposta HTTP 200, cinque preset, rotazione e zoom, cambio di allestimento, layer opzionali, tour completo fino al termine e assenza di errori nella console durante il collaudo finale. Il layout è stato controllato anche con viewport 390 × 844; il rapporto della viewport verticale misurato nel DOM è 0.5625 (9:16). Non è stato effettuato un collaudo su telefoni fisici.

Screenshot della demo: [desktop](screenshots/desktop.jpg) e [viewport mobile](screenshots/mobile.jpg).

`createArena.js` espone gruppi nominati `crowd`, `sponsors`, `interaction-zones`. Per aggiungere avatar o sponsor, popolare questi gruppi senza cambiare il campo o la camera. Per aree cliccabili, collegare un `Raycaster` agli oggetti di `interaction-zones`. Coreografie ed effetti MVP possono essere layer aggiuntivi, controllati dal futuro stato ABX.

Per sostituire elementi low-poly con GLB: caricare il modello con `GLTFLoader`, usare metri e asse Y verticale, mantenerlo nel relativo gruppo e applicare le stesse trasformazioni. Per molte sedute/avatar, mantenere geometria e materiali condivisi con instancing; per animazioni individuali valutare livelli di dettaglio e un numero limitato di scheletri. Branding e palette sono centralizzati in `scene/primitives.js`; le texture del campo in `scene/field.js`.
