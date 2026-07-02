<script setup>
defineProps({
  tiles: { type: Array, default: () => [] }
  // tile: { id, icon, label, sub, color, route }
})
defineEmits(['select'])

// Icone inline: zero dipendenze, zero richieste extra su rete d'arena congestionata
const ICONS = {
  calendar: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="4" width="18" height="18" rx="2"/><path d="M16 2v4M8 2v4M3 10h18"/></svg>',
  chart: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M6 20V13M12 20V4M18 20v-9"/></svg>',
  bracket: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="5" cy="5" r="2.2"/><circle cx="5" cy="19" r="2.2"/><circle cx="19" cy="12" r="2.2"/><path d="M7.2 5H12v14H7.2M12 12h4.8"/></svg>',
  star: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linejoin="round"><path d="m12 3 2.7 5.7 6.3.8-4.6 4.3 1.2 6.2L12 17l-5.6 3 1.2-6.2L3 9.5l6.3-.8Z"/></svg>',
  trophy: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M8 21h8M12 17v4M7 4h10v6a5 5 0 0 1-10 0Z"/><path d="M7 6H4a3 3 0 0 0 3 5M17 6h3a3 3 0 0 1-3 5"/></svg>',
  gallery: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="9" cy="9" r="2"/><path d="m21 15-4.5-4.5L6 21"/></svg>',
  doc: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8Z"/><path d="M14 2v6h6M9 13h6M9 17h6"/></svg>',
  info: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="9"/><path d="M12 16v-5M12 8h.01"/></svg>'
}
</script>

<template>
  <!-- div, NON <nav>: Materialize (caricato globalmente da index.html) colora
       ogni <nav> con background #ee6e73 → si vedeva una banda rossa nei gap. -->
  <div class="tile-grid" role="navigation" aria-label="Sezioni torneo">
    <button
      class="tile"
      v-for="tile in tiles"
      :key="tile.id"
      :style="{ background: tile.color }"
      @click="$emit('select', tile)"
    >
      <span class="tile-icon" v-html="ICONS[tile.icon] || ICONS.info"></span>
      <span class="tile-label">{{ tile.label }}</span>
      <span class="tile-sub">{{ tile.sub }}</span>
    </button>
  </div>
</template>

<style scoped>
.tile-grid { display: grid; grid-template-columns: repeat(4, var(--tm-tile)); grid-auto-rows: var(--tm-tile); gap: 9px; justify-content: space-between; }
.tile-grid { background: transparent; } /* difesa extra contro stili globali su container */
.tile {
  width: var(--tm-tile); height: var(--tm-tile);
  border: none; border-radius: 12px; color: #fff; cursor: pointer;
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  gap: clamp(2px,.5dvh,6px); padding: 4px 3px; text-align: center;
  transition: transform .12s ease, filter .12s ease;
}
.tile:active { transform: scale(.95); filter: brightness(1.15); }
.tile:focus-visible { outline: 2px solid var(--tm-gold); outline-offset: 2px; }
.tile-icon :deep(svg) { width: clamp(16px,2.8dvh,24px); height: clamp(16px,2.8dvh,24px); display: block; }
.tile-label { font-size: clamp(8px,1.25dvh,10.5px); font-weight: 800; letter-spacing: .4px; line-height: 1.05; text-transform: uppercase; }
.tile-sub { font-size: clamp(6.5px,1dvh,8.5px); color: rgba(255,255,255,.65); line-height: 1.05; margin-top: -1px; }
</style>
