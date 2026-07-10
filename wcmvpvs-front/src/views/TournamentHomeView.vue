<script setup>
// Adattato per l'app ArenaBoostX: il progetto non usa vue-router, la
// navigazione è path-based e gestita da App.vue. Riceviamo lo slug come prop
// ed emettiamo 'navigate' verso l'host, che chiama il suo navigateTo().
import { computed } from 'vue'
import { useTournamentHome } from '@/composables/useTournamentHome'
import TournamentHero from '@/components/tournament/TournamentHero.vue'
import LiveMatchCarousel from '@/components/tournament/LiveMatchCarousel.vue'
import NextMatchCard from '@/components/tournament/NextMatchCard.vue'
import TournamentTileGrid from '@/components/tournament/TournamentTileGrid.vue'
import SponsorStrip from '@/components/tournament/SponsorStrip.vue'
import SunsetPadel from '@/views/SunsetPadel.vue'
import '@/assets/tournament-tokens.css'

const props = defineProps({
  slug: { type: String, required: true },
})
const emit = defineEmits(['navigate'])

// Dati reali dal backend: snapshot su /home + polling di /live ogni 10s
// (partite in corso e prossima partita si aggiornano da sole).
const { tournament, liveMatches, nextMatch, tiles, sponsors, loading } =
  useTournamentHome(props.slug)

function onTile (tile) {
  emit('navigate', `/t/${props.slug}${tile.route}`)
}

// --- Layout selezionabile: 'classic' (default) | 'sunset' -------------------
const useSunset = computed(() => tournament.value?.layout === 'sunset')
// Mapping dei dati reali sulle props del layout Sunset.
const brandTop = computed(() => (tournament.value?.name || '').split(' ')[0] || '')
const brandBottom = computed(() => (tournament.value?.name || '').split(' ').slice(1).join(' '))
const sunsetNext = computed(() => ({
  time: nextMatch.value?.time || '—',
  home: nextMatch.value?.teamA?.name || '—',
  away: nextMatch.value?.teamB?.name || '—'
}))
// Sunset usa le stesse route delle tile; live/signup sono scorciatoie.
function onSunsetNav (route) { emit('navigate', `/t/${props.slug}${route}`) }
</script>

<template>
  <!-- Layout SUNSET: grafica alternativa, stessi dati e navigazione. -->
  <SunsetPadel
    v-if="useSunset && !loading"
    :status="tournament.statusLabel"
    :brand-top="brandTop"
    :brand-bottom="brandBottom"
    :logo="tournament.logo"
    :subtitle="tournament.format"
    :date="tournament.dateLabel"
    :place="tournament.location"
    :next-match="sunsetNext"
    :live-matches="liveMatches"
    :tiles="tiles"
    :sponsors="sponsors"
    @navigate="onSunsetNav"
    @live="onSunsetNav('/calendar')"
    @signup="onSunsetNav('/event')"
  />

  <!-- Layout CLASSIC (default): above-the-fold, hero fisso + tile + sponsor. -->
  <div class="tm-page" v-else>
    <TournamentHero v-if="tournament" :tournament="tournament" />

    <div class="tm-wrap" v-if="!loading">
      <!-- Gruppo superiore: carosello + prossima assorbono lo spazio in alto,
           le tile restano ancorate appena sopra gli sponsor. -->
      <div class="tm-main">
        <div class="tm-top">
          <LiveMatchCarousel :matches="liveMatches" />
          <NextMatchCard v-if="nextMatch" :match="nextMatch" />
        </div>
        <TournamentTileGrid class="tm-tiles" :tiles="tiles" @select="onTile" />
      </div>
      <!-- Sponsor: ancorata alla fine dell'above-the-fold. -->
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
  padding: 0 14px calc(8px + env(safe-area-inset-bottom));
  gap: clamp(10px, 1.7dvh, 18px);
}
/* Gruppo superiore: assorbe tutto lo spazio residuo e distribuisce le sezioni,
   così la sponsor box (flex:none) resta ancorata al bordo inferiore del fold.
   overflow:hidden → su schermi bassi taglia pulito qui senza invadere la box. */
.tm-main {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: clamp(8px, 1.4dvh, 16px);
  overflow: hidden;
}
/* Carosello + prossima partita: assorbono lo spazio in alto e si distribuiscono. */
.tm-top {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  justify-content: space-evenly;
  gap: clamp(6px, 1dvh, 12px);
}
/* Anti-overlap: le sezioni non si restringono sotto la loro altezza naturale. */
.tm-top > * { flex-shrink: 0; }
/* Le 8 tile: ancorate appena sopra gli sponsor, mai compresse. */
.tm-tiles { flex-shrink: 0; }
.tm-loading {
  flex: 1;
  display: grid;
  place-items: center;
  color: var(--tm-text-dim);
  font-size: 13px;
  letter-spacing: 1px;
}
</style>
