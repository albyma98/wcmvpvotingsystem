<template>
  <div class="mvp-vote-flight" :class="{ 'is-landing': isLanding, 'has-sponsors': sponsors.length }" :style="{ '--reveal-duration': `${revealDuration}ms` }" aria-hidden="true">
    <div class="mvp-vote-flight__backdrop"></div>
    <div class="mvp-vote-flight__stage">
      <div ref="card" class="mvp-vote-flight__card">
        <FanPlayerCard class="mvp-vote-flight__face" :player="player" :club-logo-url="clubLogoUrl" selected loading="eager" />
        <span class="mvp-vote-flight__back">★ MVP ★</span>
      </div>
      <div v-if="sponsors.length" class="mvp-vote-flight__sponsors">
        <p class="mvp-vote-flight__sponsor-label">Partner del Match Day</p>
        <div class="mvp-vote-flight__sponsor-row">
          <div
            v-for="{ placement, sponsor } in sponsors"
            :key="placement"
            class="mvp-vote-flight__sponsor"
            :data-placement="placement"
            :data-sponsor-id="sponsor.sponsorId"
          >
            <img
              v-if="failedLogos[placement] !== sponsor.imageUrl"
              :src="sponsor.imageUrl"
              :alt="sponsor.name || 'Sponsor'"
              @error="failedLogos[placement] = sponsor.imageUrl"
            />
            <span v-else>{{ sponsor.name || 'Sponsor' }}</span>
          </div>
        </div>
      </div>
      <p class="mvp-vote-flight__message">
        <span>Hai votato</span>
        <strong>{{ player.fullName }}</strong>
        <span class="mvp-vote-flight__progress"></span>
      </p>
    </div>
  </div>
</template>

<script setup>
import { onBeforeUnmount, onMounted, ref } from 'vue';
import FanPlayerCard from './FanPlayerCard.vue';

const props = defineProps({
  player: { type: Object, required: true },
  clubLogoUrl: { type: String, default: '' },
  sponsors: { type: Array, default: () => [] },
  source: { type: Object, default: null },
  destination: { type: Object, required: true },
});
const emit = defineEmits(['finished']);
const revealDuration = 4360;
const card = ref(null);
const isLanding = ref(false);
const failedLogos = ref({});
let animation;
let disposed = false;

function cancelAnimation() {
  animation?.cancel();
}

function position(x = 0, y = 0, scaleX = 1, scaleY = 1, turn = 0, tilt = 0) {
  return `translate(${x}px, ${y}px) scale(${scaleX}, ${scaleY}) rotateY(${turn}deg) rotateZ(${tilt}deg)`;
}

function relativeTo(center, rect) {
  return {
    x: rect.left + rect.width / 2 - center.left - center.width / 2,
    y: rect.top + rect.height / 2 - center.top - center.height / 2,
    scaleX: rect.width / center.width,
    scaleY: rect.height / center.height,
  };
}

onMounted(async () => {
  const element = card.value;
  try {
    if (!element?.animate) return;
    const center = element.getBoundingClientRect();
    if (!center.width || !center.height) return;
    window.addEventListener('resize', cancelAnimation);
    const source = props.source && relativeTo(center, props.source);
    animation = element.animate([
      {
        transform: source ? position(source.x, source.y, source.scaleX, source.scaleY) : position(0, 44, .48, .48),
        opacity: source ? 1 : 0,
        easing: 'cubic-bezier(.26,.63,.35,1)',
      },
      { transform: position(0, 0, 1.045, 1.045, 360), opacity: 1, offset: 720 / revealDuration, easing: 'ease-out' },
      { transform: position(0, 0, 1, 1, 360), opacity: 1, offset: 900 / revealDuration },
      { transform: position(0, 0, 1, 1, 360), opacity: 1 },
    ], { duration: revealDuration, fill: 'both' });
    await animation.finished;
    if (disposed || !props.destination.isConnected) return;
    animation.cancel();

    // The receipt is laid out behind the overlay, including any page scroll after the roster collapses.
    props.destination.scrollIntoView({ block: 'nearest', inline: 'nearest', behavior: 'instant' });
    const destination = props.destination.getBoundingClientRect();
    if (!destination.width || !destination.height) return;
    const { x, y, scaleX, scaleY } = relativeTo(element.getBoundingClientRect(), destination);
    isLanding.value = true;
    animation = element.animate([
      { transform: position() },
      { transform: position(x * .29 - 20, y * .25 - 18, .81, .81, 0, -9), offset: .36 },
      { transform: position(x, y, scaleX, scaleY) },
    ], { duration: 640, easing: 'cubic-bezier(.45,.14,.28,1)', fill: 'both' });
    await animation.finished;
  } catch {
    // A cancelled or unavailable animation leaves the confirmed receipt accessible.
  } finally {
    if (!disposed) emit('finished');
  }
});

onBeforeUnmount(() => {
  disposed = true;
  window.removeEventListener('resize', cancelAnimation);
  cancelAnimation();
});
</script>

<style scoped>
.mvp-vote-flight {
  position: fixed;
  inset: 0;
  z-index: 160;
  display: grid;
  place-items: center;
  overflow: hidden;
  touch-action: none;
}
.mvp-vote-flight__backdrop {
  position: absolute;
  inset: 0;
  background: radial-gradient(ellipse at center, var(--red-deep) -44%, #241526 52%, #100a15 100%);
  transition: opacity .36s ease;
}
.mvp-vote-flight__backdrop::after {
  position: absolute;
  inset: 0;
  background:
    radial-gradient(ellipse at center, #ffdc8930, transparent 52%),
    repeating-conic-gradient(from -15deg at center, #f5d6890c 0 12deg, transparent 12deg 27deg);
  mask-image: radial-gradient(ellipse, #000 8%, transparent 72%);
  content: '';
}
.mvp-vote-flight__stage { position: relative; perspective: 1000px; }
.mvp-vote-flight.has-sponsors {
  padding: env(safe-area-inset-top) env(safe-area-inset-right) env(safe-area-inset-bottom) env(safe-area-inset-left);
}
.has-sponsors .mvp-vote-flight__stage {
  display: grid;
  justify-items: center;
  row-gap: clamp(6px, 2dvh, 14px);
  width: min(86vw, 520px, 100%);
}
.mvp-vote-flight__card {
  position: relative;
  width: min(clamp(126px, 47vw, 220px), 35dvh);
  transform-style: preserve-3d;
  will-change: transform;
}
.has-sponsors .mvp-vote-flight__card { z-index: 1; }
.mvp-vote-flight__face,
.mvp-vote-flight__back { backface-visibility: hidden; }
.mvp-vote-flight__back {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  transform: rotateY(180deg);
  clip-path: polygon(0 9%, 17% 4%, 50% 0, 83% 4%, 100% 9%, 100% 85%, 89% 93%, 50% 100%, 11% 93%, 0 85%);
  background: radial-gradient(ellipse at 50% 40%, #fff3be, #e2b85f 60%, #ad7d35);
  color: #573912;
  font-size: clamp(.9rem, 5vw, 1.5rem);
  font-weight: 900;
  letter-spacing: .12em;
}
.mvp-vote-flight__sponsors {
  width: min(100%, 308px);
  text-align: center;
  transition: opacity .24s ease;
}
.mvp-vote-flight__sponsor-label {
  margin: 0 0 clamp(4px, 1dvh, 7px);
  color: #e9cf96;
  font-size: .625rem;
  font-weight: 700;
  line-height: 1.2;
  letter-spacing: .14em;
  text-transform: uppercase;
}
.mvp-vote-flight__sponsor-row {
  display: flex;
  justify-content: center;
  gap: clamp(6px, 2.5vw, 10px);
}
.mvp-vote-flight__sponsor {
  box-sizing: border-box;
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 0 1 149px;
  min-width: 0;
  height: clamp(30px, 9dvh, 54px);
  padding: clamp(3px, 1.2dvh, 8px) 10px;
  overflow: hidden;
  border: 1px solid #d8b777;
  border-radius: 9px;
  background: linear-gradient(160deg, #fffcf6, #f1e7d6);
  box-shadow: 0 0 12px #e6b85b26;
  color: #513b24;
  font-size: .75rem;
  font-weight: 750;
  line-height: 1.1;
}
.mvp-vote-flight__sponsor img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}
.mvp-vote-flight__sponsor span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.mvp-vote-flight__message {
  position: absolute;
  top: calc(100% + 14px);
  left: 50%;
  width: min(86vw, 520px);
  margin: 0;
  transform: translateX(-50%);
  color: #fff8ef;
  text-align: center;
  overflow-wrap: anywhere;
  transition: opacity .24s ease;
}
.has-sponsors .mvp-vote-flight__message { position: static; width: 100%; transform: none; }
.mvp-vote-flight__message > * { display: block; animation: mvp-vote-message .26s ease-out .16s both; }
.mvp-vote-flight__message span:first-child {
  color: #ffdb89;
  font-size: .75rem;
  font-weight: 800;
  letter-spacing: .18em;
  text-transform: uppercase;
}
.mvp-vote-flight__message strong { font-size: clamp(1.25rem, 6vw, 2.25rem); line-height: 1.08; }
.mvp-vote-flight__progress {
  width: 64px;
  height: 2px;
  margin: 10px auto 0;
  overflow: hidden;
  border-radius: 2px;
  background: #ffdb8940;
}
.mvp-vote-flight__progress::after {
  display: block;
  height: 100%;
  background: #ffdb89;
  transform-origin: left;
  animation: mvp-vote-progress var(--reveal-duration) linear both;
  content: '';
}
.is-landing .mvp-vote-flight__backdrop,
.is-landing .mvp-vote-flight__sponsors,
.is-landing .mvp-vote-flight__message { opacity: 0; }
@media (max-height: 420px) {
  .has-sponsors .mvp-vote-flight__card { width: min(47vw, 31dvh); }
  .has-sponsors .mvp-vote-flight__message strong { font-size: clamp(1.125rem, 6dvh, 1.5rem); }
}
@keyframes mvp-vote-message { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: none; } }
@keyframes mvp-vote-progress { from { transform: scaleX(0); } to { transform: scaleX(1); } }
</style>
