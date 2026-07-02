<script setup>
import { computed } from 'vue'

const props = defineProps({
  sponsors: { type: Array, default: () => [] }
  // { id, name, logo?, url?, tier: 'main'|'partner', brandColor? }
})

const mainSponsors = computed(() => props.sponsors.filter(s => s.tier === 'main'))
const partnerSponsors = computed(() =>
  props.sponsors.filter(s => s.tier !== 'main'))

// Track duplicato per loop marquee senza salti
const marqueeTrack = computed(() => [...partnerSponsors.value, ...partnerSponsors.value])

/**
 * Inventory a due livelli:
 * - MAIN: riga fissa, loghi a dimensione hero, sempre visibili → tier premium
 * - PARTNER: marquee continuo stile LED bordo campo → tier base
 * Tracking esposizioni/tap da agganciare qui (PostHog) per il report sponsor.
 */
</script>

<template>
  <section class="sponsor-block" v-if="sponsors.length">
    <div class="label">Main Sponsor</div>

    <div class="sponsor-main" v-if="mainSponsors.length">
      <component
        v-for="s in mainSponsors" :key="s.id"
        :is="s.url ? 'a' : 'span'" :href="s.url" target="_blank" rel="noopener"
        class="sp-main-item"
      >
        <img v-if="s.logo" :src="s.logo" :alt="s.name" loading="lazy" />
        <span v-else class="sp-text">{{ s.name }}</span>
      </component>
    </div>

    <div class="sponsor-marquee" v-if="partnerSponsors.length">
      <div class="marquee-track">
        <component
          v-for="(s, i) in marqueeTrack" :key="`${s.id}-${i}`"
          :is="s.url ? 'a' : 'span'" :href="s.url" target="_blank" rel="noopener"
          class="sp-pill"
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
  padding: clamp(10px,1.7dvh,16px) 0 clamp(11px,1.8dvh,16px); overflow: hidden;
}
.label {
  font-size: clamp(8px,1.2dvh,10.5px); font-weight: 800; letter-spacing: 1.8px;
  color: var(--tm-text-faint); text-transform: uppercase; padding: 0 14px;
}
/* MAIN: dimensione da hero — l'inventory premium si vede da lontano */
.sponsor-main {
  display: flex; align-items: center; justify-content: center;
  gap: clamp(22px,5vw,38px); margin-top: clamp(8px,1.4dvh,14px); padding: 0 14px;
}
.sp-main-item { display: inline-flex; align-items: center; text-decoration: none; color: inherit; }
.sponsor-main img { height: clamp(32px,5dvh,46px); object-fit: contain; }
.sp-text { font-weight: 900; font-size: clamp(19px,3dvh,26px); letter-spacing: .4px; white-space: nowrap; }
/* PARTNER: marquee continuo stile LED bordo campo */
.sponsor-marquee {
  margin-top: clamp(9px,1.6dvh,15px); overflow: hidden; position: relative;
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
  padding: clamp(7px,1.2dvh,11px) 15px; font-weight: 800;
  font-size: clamp(11.5px,1.8dvh,14.5px); letter-spacing: .3px; white-space: nowrap;
  text-decoration: none;
}
.sp-pill img { height: clamp(17px,2.7dvh,24px); object-fit: contain; }
.sp-pill .dot { width: 9px; height: 9px; border-radius: 50%; flex: none; }
</style>
