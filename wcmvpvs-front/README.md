# Vue 3 + Vite

## Joy Volley · Arena Tifosi

L’Arena Tifosi 3D è disponibile in `/arena-demo/`, come pagina autonoma senza backend: 60 profili demo, 280 comparse, ricerca avatar, voto MVP e azioni animate.
Avvio: `npm install`, `npm run dev`, poi aprire `http://localhost:5173/arena-demo/`.
Vedere [istruzioni, controlli e architettura](arena-demo/README.md).

## Sponsor nelle arene di basket

In `src/games/gameWallSponsors.js`, associa gli ID degli sponsor attivi di Fan Live allo slug della società, al gioco (`basketball`, `playmaker`, `crossover`) e al lato (`left`, `right`).

```js
'slug-societa': {
  basketball: { left: 12, right: 34 },
  playmaker: { left: 34, right: 12 },
  crossover: { left: 12 },
},
```

I pannelli usano i loghi del catalogo Fan Live e mostrano una superficie neutra quando un lato è senza assegnazione o il logo non è disponibile. Le modifiche alla mappa richiedono una nuova build del frontend.

This template should help get you started developing with Vue 3 in Vite. The template uses Vue 3 `<script setup>` SFCs, check out the [script setup docs](https://v3.vuejs.org/api/sfc-script-setup.html#sfc-script-setup) to learn more.

Learn more about IDE Support for Vue in the [Vue Docs Scaling up Guide](https://vuejs.org/guide/scaling-up/tooling.html#ide-support).
