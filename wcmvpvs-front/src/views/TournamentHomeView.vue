<script setup>
// Adattato per l'app ArenaBoostX: il progetto non usa vue-router, la
// navigazione è path-based e gestita da App.vue. Riceviamo lo slug come prop
// ed emettiamo 'navigate' verso l'host, che chiama il suo navigateTo().
import { useTournamentHome } from '@/composables/useTournamentHome'
import TournamentHero from '@/components/tournament/TournamentHero.vue'
import LiveMatchCarousel from '@/components/tournament/LiveMatchCarousel.vue'
import NextMatchCard from '@/components/tournament/NextMatchCard.vue'
import TournamentTileGrid from '@/components/tournament/TournamentTileGrid.vue'
import SponsorStrip from '@/components/tournament/SponsorStrip.vue'
import '@/assets/tournament-tokens.css'

const props = defineProps({
  slug: { type: String, required: true },
})
const emit = defineEmits(['navigate'])

// mock: true → dati demo (Sunset Beach Cup) senza toccare la rete.
// Togliere { mock: true } quando il backend è collegato.
const { tournament, liveMatches, nextMatch, tiles, sponsors, loading } =
  useTournamentHome(props.slug, { mock: true })

function onTile (tile) {
  emit('navigate', `/t/${props.slug}${tile.route}`)
}
</script>

<template>
  <!-- Layout above-the-fold: hero a dimensione fissa, body che riempie il
       resto senza mai scrollare. Zero bottom nav: tutta la navigazione passa
       dalle tile, che sono il pattern giusto per l'onboarding-da-QR (il tifoso
       scansiona ed è già dentro, senza imparare una barra). -->
  <div class="tm-page">
    <TournamentHero v-if="tournament" :tournament="tournament" />

    <div class="tm-wrap" v-if="!loading">
      <LiveMatchCarousel :matches="liveMatches" />
      <NextMatchCard v-if="nextMatch" :match="nextMatch" />
      <TournamentTileGrid :tiles="tiles" @select="onTile" />
      <SponsorStrip :sponsors="sponsors" />
    </div>

    <div v-else class="tm-loading" aria-live="polite">Caricamento torneo…</div>
  </div>
</template>

<style scoped>
.tm-page {
  width: 100%;
  max-width: 430px;
  margin: 0 auto;
  height: 100vh;    /* fallback per browser senza dvh */
  height: 100dvh;
  background: #0A0A0E;                 /* fallback hard: mai trasparente sopra l'app club */
  background: var(--tm-bg, #0A0A0E);
  color: var(--tm-text);
  display: flex;
  flex-direction: column;
  overflow: hidden; /* garanzia hard: nessuno scroll verticale, mai */
}
/* Body: riempie lo spazio residuo e distribuisce le sezioni.
   Gli spazi si comprimono in modo fluido su schermi bassi (dvh), si aprono su schermi alti. */
.tm-wrap {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  justify-content: space-evenly;
  padding: 0 14px calc(8px + env(safe-area-inset-bottom));
  gap: clamp(6px, 1dvh, 12px);
}
/* Anti-overlap: i figli non devono MAI restringersi sotto la loro altezza
   naturale. Se il totale supera lo spazio, meglio un taglio pulito in fondo
   che contenuti sovrapposti (il bug visto su Safari iOS). */
.tm-wrap > * { flex-shrink: 0; }
.tm-loading {
  flex: 1;
  display: grid;
  place-items: center;
  color: var(--tm-text-dim);
  font-size: 13px;
  letter-spacing: 1px;
}
</style>
