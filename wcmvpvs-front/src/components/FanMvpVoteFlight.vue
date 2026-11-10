<template>
  <div class="mvp-vote-flight" :class="{ 'is-landing': isLanding }" aria-hidden="true">
    <div class="mvp-vote-flight__backdrop"></div>
    <div class="mvp-vote-flight__stage">
      <div ref="card" class="mvp-vote-flight__card">
        <FanPlayerCard class="mvp-vote-flight__face" :player="player" :club-logo-url="clubLogoUrl" selected loading="eager" />
        <span class="mvp-vote-flight__back">★ MVP ★</span>
      </div>
      <p class="mvp-vote-flight__message">
        <span>Hai votato</span>
        <strong>{{ player.fullName }}</strong>
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
  source: { type: Object, default: null },
  destination: { type: Object, required: true },
});
const emit = defineEmits(['finished']);
const card = ref(null);
const isLanding = ref(false);
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
      { transform: position(0, 0, 1.045, 1.045, 360), opacity: 1, offset: .54, easing: 'ease-out' },
      { transform: position(0, 0, 1, 1, 360), opacity: 1, offset: .67 },
      { transform: position(0, 0, 1, 1, 360), opacity: 1 },
    ], { duration: 4900, fill: 'both' });
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
    ], { duration: 2100, easing: 'cubic-bezier(.45,.14,.28,1)', fill: 'both' });
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
  transition: opacity 1.2s ease;
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
.mvp-vote-flight__card {
  position: relative;
  width: min(clamp(126px, 47vw, 220px), 35dvh);
  transform-style: preserve-3d;
  will-change: transform;
}
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
  transition: opacity .8s ease;
}
.mvp-vote-flight__message > * { display: block; animation: mvp-vote-message .9s ease-out 1.5s both; }
.mvp-vote-flight__message span {
  color: #ffdb89;
  font-size: .75rem;
  font-weight: 800;
  letter-spacing: .18em;
  text-transform: uppercase;
}
.mvp-vote-flight__message strong { font-size: clamp(1.25rem, 6vw, 2.25rem); line-height: 1.08; }
.is-landing .mvp-vote-flight__backdrop,
.is-landing .mvp-vote-flight__message { opacity: 0; }
@keyframes mvp-vote-message { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: none; } }
</style>
