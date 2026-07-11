<script setup>
import { computed, ref, watch, onBeforeUnmount } from 'vue'
import { track as posthogTrack, EVENTS as PH_EVENTS } from '@/lib/track'

const props = defineProps({
  sponsors: { type: Array, default: () => [] },
  // { id, name, logo?, url?, tier: 'main'|'partner', brandColor? }
  tournamentSlug: { type: String, default: '' }
})

// sponsor_strip_shown: una volta, quando la strip riceve degli sponsor (impression aggregata).
let shownTracked = false
watch(
  () => props.sponsors.length,
  (n) => {
    if (!n || shownTracked) return
    shownTracked = true
    posthogTrack(PH_EVENTS.TOURNAMENT_SPONSOR_SHOWN, {
      tournament_slug: props.tournamentSlug,
      surface: 'tournament',
      sponsor_count: n,
      main_count: props.sponsors.filter(s => s.tier === 'main').length,
      partner_count: props.sponsors.filter(s => s.tier !== 'main').length,
    })
  },
  { immediate: true },
)

function onSponsorClick (s) {
  posthogTrack(PH_EVENTS.TOURNAMENT_SPONSOR_CLICKED, {
    tournament_slug: props.tournamentSlug,
    surface: 'tournament',
    sponsor_id: s.id,
    sponsor_name: s.name,
    tier: s.tier || 'partner',
    has_url: !!s.url,
  })
}

const mainSponsors = computed(() => props.sponsors.filter(s => s.tier === 'main'))
const partnerSponsors = computed(() =>
  props.sponsors.filter(s => s.tier !== 'main'))

// Track duplicato per loop marquee senza salti
const marqueeTrack = computed(() => [...partnerSponsors.value, ...partnerSponsors.value])

// Main sponsor: fino a 3 stanno insieme nel box. Oltre 3 diventa uno slideshow
// a loop, pagine da 3, così ogni logo resta grande e leggibile a rotazione.
const MAIN_PER_SLIDE = 3
const SLIDE_MS = 4500
const mainPages = computed(() => {
  const pages = []
  for (let i = 0; i < mainSponsors.value.length; i += MAIN_PER_SLIDE) {
    pages.push(mainSponsors.value.slice(i, i + MAIN_PER_SLIDE))
  }
  return pages
})
const pageIndex = ref(0)
const currentPage = computed(
  () => mainPages.value[pageIndex.value] ?? mainPages.value[0] ?? []
)

let timer = null
function stopSlides () { if (timer) { clearInterval(timer); timer = null } }
function startSlides () {
  stopSlides()
  if (mainPages.value.length > 1) {
    timer = setInterval(() => {
      pageIndex.value = (pageIndex.value + 1) % mainPages.value.length
    }, SLIDE_MS)
  }
}
// (Ri)avvia quando il numero di pagine cambia; tieni l'indice in range.
watch(() => mainPages.value.length, (n) => {
  if (pageIndex.value >= n) pageIndex.value = 0
  startSlides()
}, { immediate: true })
onBeforeUnmount(stopSlides)

/**
 * Inventory a due livelli:
 * - MAIN: riga fissa, loghi a dimensione hero, sempre visibili → tier premium
 * - PARTNER: marquee continuo stile LED bordo campo → tier base
 * Tracking esposizioni/tap da agganciare qui (PostHog) per il report sponsor.
 */
</script>

<template>
  <section class="sponsor-block" :class="{ 'has-main': mainSponsors.length }" v-if="sponsors.length">
    <div class="label">Main Sponsor</div>

    <div class="sponsor-main" v-if="mainSponsors.length">
      <Transition name="sp-fade" mode="out-in">
        <div class="sp-page" :key="pageIndex">
          <component
            v-for="s in currentPage" :key="s.id"
            :is="s.url ? 'a' : 'span'" :href="s.url" target="_blank" rel="noopener"
            class="sp-main-item"
            @click="onSponsorClick(s)"
          >
            <img v-if="s.logo" :src="s.logo" :alt="s.name" loading="lazy" />
            <span v-else class="sp-text">{{ s.name }}</span>
          </component>
        </div>
      </Transition>
    </div>

    <!-- Partner nascosti quando ci sono main sponsor: lo spazio va ai loghi main. -->
    <div class="sponsor-marquee" v-if="!mainSponsors.length && partnerSponsors.length">
      <div class="marquee-track">
        <component
          v-for="(s, i) in marqueeTrack" :key="`${s.id}-${i}`"
          :is="s.url ? 'a' : 'span'" :href="s.url" target="_blank" rel="noopener"
          class="sp-pill"
          @click="onSponsorClick(s)"
        >
          <img v-if="s.logo" :src="s.logo" :alt="s.name" loading="lazy" />
          <span v-else-if="s.brandColor" class="dot" :style="{ background: s.brandColor }"></span>
          {{ s.name }}
        </component>
      </div>
    </div>
  </section>
</template>

<style scoped>
.sponsor-block {
  flex: none; background: var(--tm-surface);
  border: 1px solid var(--tm-border); border-radius: var(--tm-radius);
  padding: clamp(12px,2dvh,19px) 0 clamp(13px,2.2dvh,19px); overflow: hidden;
}
.label {
  font-size: clamp(8px,1.2dvh,10.5px); font-weight: 800; letter-spacing: 1.8px;
  color: var(--tm-text-faint); text-transform: uppercase; padding: 0 14px;
}
/* MAIN: dimensione da hero — l'inventory premium si vede da lontano */
.sponsor-main {
  display: flex; align-items: center; justify-content: center;
  gap: clamp(22px,5vw,38px); margin-top: clamp(10px,1.7dvh,17px); padding: 0 14px;
}
.sp-main-item { display: inline-flex; align-items: center; text-decoration: none; color: inherit; }
.sponsor-main img { height: clamp(38px,6dvh,55px); object-fit: contain; }

/* --- Con main sponsor: il blocco cresce e i loghi riempiono tutto lo spazio,
   stretchati per occupare la cella. I partner sono omessi (v-if sopra). --- */
.sponsor-block.has-main {
  display: flex; flex-direction: column;
  flex: none;                          /* niente crescita: altezza fissa, non si prende il fold */
  height: clamp(68px, 15dvh, 136px);   /* altezza max relativa al display, con cap */
  padding: clamp(5px,0.9dvh,8px);      /* box aderente ai loghi: niente padding in eccesso */
}
.has-main .label { display: none; }    /* via la scritta: tutto lo spazio ai loghi */
.has-main .sponsor-main {
  flex: 1; min-height: 0; align-items: stretch;
  margin-top: 0; padding: 0;           /* i loghi arrivano ai bordi del box */
}
.has-main .sp-page {
  flex: 1; min-width: 0; display: flex; align-items: stretch;
  justify-content: center; gap: clamp(6px,2vw,14px);
}
.has-main .sp-main-item { flex: 1 1 0; min-width: 0; align-items: stretch; }
/* Slideshow main sponsor: dissolvenza tra le pagine */
.sp-fade-enter-active, .sp-fade-leave-active { transition: opacity .45s ease; }
.sp-fade-enter-from, .sp-fade-leave-to { opacity: 0; }
@media (prefers-reduced-motion: reduce) {
  .sp-fade-enter-active, .sp-fade-leave-active { transition: none; }
}
.has-main .sponsor-main img {
  width: 100%; height: 100%; object-fit: fill; border-radius: 10px;
}
.has-main .sp-text {
  width: 100%; display: flex; align-items: center; justify-content: center;
  text-align: center; overflow: hidden;
}
.sp-text { font-weight: 900; font-size: clamp(19px,3dvh,26px); letter-spacing: .4px; white-space: nowrap; }
/* PARTNER: marquee continuo stile LED bordo campo */
.sponsor-marquee {
  margin-top: clamp(11px,1.9dvh,18px); overflow: hidden; position: relative;
  -webkit-mask-image: linear-gradient(90deg,transparent,#000 8%,#000 92%,transparent);
  mask-image: linear-gradient(90deg,transparent,#000 8%,#000 92%,transparent);
}
.marquee-track {
  display: flex; align-items: center; gap: 12px; width: max-content;
  animation: tm-marquee 32s linear infinite;
}
.sponsor-marquee:active .marquee-track,
.sponsor-marquee:hover .marquee-track { animation-play-state: paused; }
@keyframes tm-marquee { from { transform: translateX(0); } to { transform: translateX(-50%); } }
@media (prefers-reduced-motion: reduce) { .marquee-track { animation: none; } }
/* Pastiglia chiara: lo sponsor vive coi suoi colori su fondo leggibile */
.sp-pill {
  flex: none; display: inline-flex; align-items: center; gap: 8px;
  background: rgba(255,255,255,.92); color: #111; border-radius: 11px;
  padding: clamp(8px,1.4dvh,13px) 15px; font-weight: 800;
  font-size: clamp(11.5px,1.8dvh,14.5px); letter-spacing: .3px; white-space: nowrap;
  text-decoration: none;
}
.sp-pill img { height: clamp(20px,3.2dvh,29px); object-fit: contain; }
.sp-pill .dot { width: 9px; height: 9px; border-radius: 50%; flex: none; }
</style>
