# Vue 3 + Vite

## Joy Volley · Arena Tifosi

L’Arena Tifosi 3D è disponibile in `/arena-demo/`, come pagina autonoma senza backend: 60 profili demo, 280 comparse, ricerca avatar, voto MVP e azioni animate.
Avvio: `npm install`, `npm run dev`, poi aprire `http://localhost:5173/arena-demo/`.
Vedere [istruzioni, controlli e architettura](arena-demo/README.md).

## Sponsor nelle arene di basket

Nell’admin della società, apri **Sponsor → Banner nei minigiochi**. Scegli lo sponsor per ogni parete di Basket Rush, Playmaker e Crossover Rush, poi premi **Salva banner**. Le assegnazioni si applicano alla successiva apertura di Fan Live.

I pannelli usano i loghi degli sponsor visibili di Fan Live. Scegli **Pannello neutro** per svuotare una parete. Anche uno sponsor nascosto, eliminato o con un logo irraggiungibile lascia il pannello neutro.

## Sponsor nella conferma MVP

In `src/sponsorPlacements.js`, associa lo slug della società agli ID degli sponsor del catalogo Fan Live.

```js
'slug-societa': { post_vote_left: 12, post_vote_right: 34 },
```

La conferma usa gli sponsor attivi assegnati. Con un solo ID attivo il banner è centrato. Le modifiche alle assegnazioni richiedono una nuova build del frontend.

This template should help get you started developing with Vue 3 in Vite. The template uses Vue 3 `<script setup>` SFCs, check out the [script setup docs](https://v3.vuejs.org/api/sfc-script-setup.html#sfc-script-setup) to learn more.

Learn more about IDE Support for Vue in the [Vue Docs Scaling up Guide](https://vuejs.org/guide/scaling-up/tooling.html#ide-support).
