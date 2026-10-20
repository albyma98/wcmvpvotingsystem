<template>
  <div class="mvp-celebration">
    <div class="mvp-celebration-stage">
      <div class="mvp-effects" aria-hidden="true">
        <span class="mvp-spotlight"></span>
        <span class="mvp-rays"></span>
        <span v-for="ring in 3" :key="`ring-${ring}`" class="mvp-shockwave" :style="{ '--ring-index': ring - 1 }"></span>
        <span v-for="piece in confetti" :key="piece.id" class="mvp-confetti" :class="{ 'is-round': piece.id % 4 === 0 }" :style="piece.style"></span>
      </div>
      <span class="mvp-card-shadow" aria-hidden="true"></span>
      <div class="mvp-champion-card">
        <FanPlayerCard :player="player" :club-logo-url="clubLogoUrl" selected loading="eager" />
        <span class="mvp-confirm-seal" aria-hidden="true">✓</span>
      </div>
    </div>
    <div class="mvp-champion-banner" aria-hidden="true"><span>★</span> IL TUO MVP <span>★</span></div>
  </div>
</template>

<script setup>
import FanPlayerCard from './FanPlayerCard.vue';

defineProps({
  player: { type: Object, required: true },
  clubLogoUrl: { type: String, default: '' },
});

// Fixed trajectories keep the burst consistent and within the mobile sheet.
const confetti = Array.from({ length: 28 }, (_, id) => {
  const angle = (id / 28) * Math.PI * 2;
  const distance = 72 + (id % 5) * 15;
  return {
    id,
    style: {
      '--burst-x': `${Math.round(Math.cos(angle) * distance)}px`,
      '--burst-y': `${Math.round(Math.sin(angle) * distance - 32)}px`,
      '--fall-y': `${75 + (id % 4) * 19}px`,
      '--piece-turn': `${(id % 2 ? 1 : -1) * (180 + id * 23)}deg`,
      '--piece-color': ['var(--gold, #f5b321)', 'var(--red, #e4212e)', '#edca75', 'var(--ink, #23161a)'][id % 4],
      animationDelay: `${180 + (id % 7) * 24}ms`,
    },
  };
});
</script>

<style scoped>
.mvp-celebration { position: relative; isolation: isolate; display: flex; flex: none; flex-direction: column; align-items: center; width: 100%; padding: 0 0 7px; }
.mvp-celebration-stage { position: relative; display: grid; place-items: center; width: 100%; height: clamp(214px, 31dvh, 252px); perspective: 900px; }
.mvp-effects { position: absolute; inset: 0 -8px; z-index: -1; overflow: hidden; pointer-events: none; }
.mvp-spotlight { position: absolute; inset: -4% 0; background: radial-gradient(ellipse, #f5ce7499, var(--red-soft, #ffe8e6) 44%, transparent 70%); animation: mvp-light-up 800ms ease-out both; }
.mvp-rays { position: absolute; inset: -30%; background: repeating-conic-gradient(from 0deg at 50% 50%, transparent 0deg 17deg, #f3cc793d 17deg 26deg, transparent 26deg 36deg); mask-image: radial-gradient(ellipse, #000, transparent 65%); animation: mvp-rays-open 1200ms cubic-bezier(.16, 1, .3, 1) both; }
.mvp-shockwave { position: absolute; left: 50%; top: 48%; width: 120px; height: 120px; border: 2px solid var(--gold, #f5b321); border-radius: 50%; opacity: 0; animation: mvp-impact-ring 850ms ease-out calc(140ms + var(--ring-index) * 115ms) both; }
.mvp-shockwave:nth-of-type(4) { border-color: var(--red-line, #ffd2ce); }
.mvp-confetti { position: absolute; left: 50%; top: 42%; width: 6px; height: 11px; border-radius: 2px; background: var(--piece-color); opacity: 0; animation: mvp-confetti-burst 1100ms cubic-bezier(.15, .6, .3, 1) both; }
.mvp-confetti.is-round { width: 5px; height: 5px; border-radius: 50%; }
.mvp-card-shadow { position: absolute; bottom: 7px; width: 152px; max-width: 60%; height: 22px; border-radius: 50%; background: radial-gradient(ellipse, #996d3a42, transparent 70%); animation: mvp-shadow-land 720ms ease-out both; }
.mvp-champion-card { position: relative; width: clamp(118px, 36vw, 150px); max-width: 43%; transform-origin: 50% 85%; animation: mvp-champion-reveal 720ms cubic-bezier(.16, .85, .25, 1) both; }
.mvp-champion-card :deep(.fan-player-card) { filter: drop-shadow(0 9px 15px #a16e3040); }
.mvp-champion-card :deep(.mvp-player-card)::after { content: ''; position: absolute; z-index: 3; inset: -30% -65%; background: linear-gradient(110deg, transparent 38%, #ffffffcc 49%, #ffffff24 57%, transparent 63%); pointer-events: none; animation: mvp-champion-shine 700ms ease 270ms both; }
.mvp-champion-card :deep(.mvp-player-caption strong) { font-size: clamp(12px, 3.4vw, 15px); }
.mvp-confirm-seal { position: absolute; z-index: 4; top: 22%; right: -8px; display: grid; place-items: center; width: 34px; height: 34px; border: 3px solid var(--card, #fff); border-radius: 50%; background: var(--red, #e4212e); color: var(--on-primary, #fff); font-size: 20px; font-weight: 900; box-shadow: 0 4px 13px var(--red-line, #ffd2ce); animation: mvp-seal-stamp 430ms cubic-bezier(.16, 1, .3, 1) 380ms both; }
.mvp-champion-banner { position: relative; display: flex; align-items: center; justify-content: center; gap: 12px; margin-top: -3px; border: 1px solid #e6bd62; border-radius: 7px; padding: 7px 19px; background: linear-gradient(115deg, #f5d58c, #fff0bf 48%, #edc66f); color: #644313; font-size: 11px; font-weight: 900; letter-spacing: .14em; box-shadow: 0 3px 0 #cfa95738; transform: rotate(-3deg); animation: mvp-banner-land 500ms cubic-bezier(.16, 1, .3, 1) 330ms both; }
.mvp-champion-banner > span { font-size: 12px; }
@keyframes mvp-champion-reveal {
  0% { opacity: 0; transform: translateY(32px) rotateY(-70deg) rotateZ(-12deg) scale(.58); }
  44% { opacity: 1; transform: translateY(-11px) rotateY(9deg) rotateZ(4deg) scale(1.09); }
  72% { transform: translateY(3px) rotateY(-3deg) rotateZ(-1deg) scale(.985); }
  100% { opacity: 1; transform: none; }
}
@keyframes mvp-champion-shine { from { transform: translateX(-65%); } to { transform: translateX(65%); } }
@keyframes mvp-light-up { 0% { opacity: 0; transform: scale(.45); } 45% { opacity: 1; transform: scale(1.08); } 100% { opacity: 1; transform: scale(1); } }
@keyframes mvp-rays-open { from { opacity: 0; transform: rotate(-24deg) scale(.5); } to { opacity: 1; transform: rotate(12deg) scale(1); } }
@keyframes mvp-impact-ring { 0% { opacity: 0; transform: translate(-50%, -50%) scale(.35); } 15% { opacity: .65; } 100% { opacity: 0; transform: translate(-50%, -50%) scale(2.5); } }
@keyframes mvp-confetti-burst {
  0% { opacity: 0; transform: translate(0, 0) rotate(0) scale(.4); }
  16% { opacity: 1; }
  48% { opacity: 1; transform: translate(var(--burst-x), var(--burst-y)) rotate(calc(var(--piece-turn) * .5)) scale(1); }
  100% { opacity: 0; transform: translate(calc(var(--burst-x) * 1.2), calc(var(--burst-y) + var(--fall-y))) rotate(var(--piece-turn)) scale(.65); }
}
@keyframes mvp-shadow-land { from { opacity: 0; transform: scale(.4); } to { opacity: 1; transform: scale(1); } }
@keyframes mvp-seal-stamp { 0% { opacity: 0; transform: scale(2.1) rotate(25deg); } 65% { opacity: 1; transform: scale(.88) rotate(-8deg); } 100% { opacity: 1; transform: none; } }
@keyframes mvp-banner-land { 0% { opacity: 0; transform: translateY(12px) rotate(-7deg) scale(.7); } 65% { opacity: 1; transform: translateY(-2px) rotate(-2deg) scale(1.04); } 100% { opacity: 1; transform: rotate(-3deg); } }
@media (max-height: 700px) { .mvp-celebration-stage { height: 202px; } .mvp-champion-card { width: 124px; } }
@media (prefers-reduced-motion: reduce) {
  .mvp-effects, .mvp-champion-card :deep(.mvp-player-card)::after { display: none; }
  .mvp-champion-card, .mvp-confirm-seal, .mvp-card-shadow, .mvp-champion-banner { animation: none; }
}
</style>
