<script setup>
import { ref } from 'vue'
import TeamBadge from './TeamBadge.vue'

defineProps({
  matches: { type: Array, default: () => [] }
})

const activeSlide = ref(0)
const carousel = ref(null)

function onScroll () {
  const el = carousel.value
  if (!el) return
  activeSlide.value = Math.round(el.scrollLeft / el.clientWidth)
}
</script>

<template>
  <div v-if="matches.length">
    <div class="live-carousel" ref="carousel" @scroll.passive="onScroll">
      <article class="match-card" v-for="m in matches" :key="m.id">
        <div class="match-head">
          <span class="label"><span class="live-dot" aria-hidden="true"></span>PARTITA IN CORSO</span>
          <span class="court">{{ m.court }}</span>
        </div>
        <div class="match-body">
          <TeamBadge :team="m.teamA" size="lg" />
          <div class="score-block">
            <div class="score">{{ m.score.a }} : {{ m.score.b }}</div>
            <div class="set-label">{{ m.setLabel }}</div>
          </div>
          <TeamBadge :team="m.teamB" size="lg" />
        </div>
        <div class="set-scores">
          <template v-for="(s, i) in m.sets" :key="i">
            <span v-if="i" class="sep">|</span><span>{{ s }}</span>
          </template>
        </div>
      </article>
    </div>
    <div class="carousel-dots" v-if="matches.length > 1" aria-hidden="true">
      <span v-for="(_, i) in matches" :key="i" :class="{ active: i === activeSlide }"></span>
    </div>
  </div>
</template>

<style scoped>
.live-carousel {
  display: flex; overflow-x: auto; scroll-snap-type: x mandatory;
  gap: 10px; margin-top: 14px; scrollbar-width: none;
}
.live-carousel::-webkit-scrollbar { display: none; }
.match-card {
  scroll-snap-align: center; flex: 0 0 100%; min-width: 0;
  border-radius: var(--tm-radius); overflow: hidden;
  border: 1px solid var(--tm-border);
  background: linear-gradient(180deg, rgba(139,32,38,.55) 0%, rgba(80,22,28,.28) 46%, var(--tm-surface) 100%);
}
.match-head { display: flex; align-items: center; justify-content: space-between; padding: clamp(7px,1.2dvh,11px) 12px 2px; }
.match-head .label { display: inline-flex; align-items: center; gap: 6px; font-size: clamp(9px,1.35dvh,11px); font-weight: 800; letter-spacing: 1px; }
.match-head .court { font-size: clamp(9px,1.35dvh,11px); font-weight: 700; letter-spacing: 1px; color: var(--tm-text-dim); }
.match-body { display: grid; grid-template-columns: 1fr auto 1fr; align-items: center; padding: clamp(3px,.6dvh,6px) 12px; }
.score-block { text-align: center; padding: 0 8px; }
.score { font-size: clamp(24px,4.4dvh,38px); font-weight: 900; letter-spacing: 2px; line-height: 1; font-variant-numeric: tabular-nums; }
.set-label { margin-top: 2px; font-size: clamp(8.5px,1.3dvh,10.5px); font-weight: 700; letter-spacing: 1px; color: var(--tm-text-dim); }
.set-scores {
  display: flex; justify-content: center; gap: 10px; padding: 1px 0 clamp(6px,1.1dvh,11px);
  font-size: clamp(10px,1.5dvh,12.5px); font-weight: 700; color: var(--tm-text-dim); font-variant-numeric: tabular-nums;
}
.set-scores .sep { opacity: .4; }
.live-dot { width: 8px; height: 8px; border-radius: 50%; background: var(--tm-live); animation: tm-pulse 1.6s ease-in-out infinite; }
@keyframes tm-pulse { 0%,100% { opacity: 1; } 50% { opacity: .45; } }
@media (prefers-reduced-motion: reduce) { .live-dot { animation: none; } }
.carousel-dots { display: flex; justify-content: center; gap: 6px; margin-top: -2px; }
.carousel-dots span { width: 5px; height: 5px; border-radius: 50%; background: rgba(255,255,255,.22); transition: background .2s; }
.carousel-dots span.active { background: #fff; }
</style>
