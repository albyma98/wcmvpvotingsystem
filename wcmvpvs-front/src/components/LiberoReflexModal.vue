<template>
  <div class="libero-reflex">
    <p class="libero-reflex__eyebrow">Mini missione</p>
    <h4 class="libero-reflex__title">Libero Reflex</h4>
    <p class="libero-reflex__subtitle">Tocca al momento giusto!</p>
    <div class="libero-reflex__instruction" aria-live="polite">
      <span class="libero-reflex__instruction-icon" aria-hidden="true">✋</span>
      <span>Blocca la palla <strong>QUANDO</strong> è nella zona verde!</span>
    </div>

    <div class="libero-reflex__status">
      <div class="libero-reflex__level" :style="levelStyle">
        <span class="libero-reflex__level-chip">Livello</span>
        <strong>{{ currentLevel }} / {{ maxLevel }}</strong>
      </div>
      <div class="libero-reflex__lives" aria-label="Vite rimaste">
        <span class="libero-reflex__lives-label">Vite</span>
        <span
          v-for="heart in hearts"
          :key="heart.id"
          :class="['libero-reflex__heart', { 'is-empty': heart.empty }]"
        >
          ♥
        </span>
      </div>
    </div>

    <div
      class="libero-reflex__arena"
      :class="arenaClasses"
      role="button"
      tabindex="0"
      aria-label="Campo di Libero Reflex"
      @pointerdown.prevent="handleTap"
      @keydown.space.prevent="handleTap"
      @keydown.enter.prevent="handleTap"
    >
      <div class="libero-reflex__zone" :style="zoneStyle" aria-hidden="true">
        <div class="libero-reflex__zone-arrows" aria-hidden="true"></div>
        <span class="libero-reflex__zone-label">Zona verde</span>
      </div>

      <div class="libero-reflex__ball" :class="ballClasses" :style="ballStyle">
        <img :src="volleyballBall" alt="Palla da volley" />
      </div>

      <Transition name="libero-reflex-perfect">
        <div v-if="feedbackState === 'success'" class="libero-reflex__perfect">Perfetto!</div>
      </Transition>

      <Transition name="libero-reflex-level">
        <div key="currentLevel" class="libero-reflex__level-toast">LIVELLO {{ currentLevel }}</div>
      </Transition>

      <Transition name="libero-reflex-feedback">
        <div
          v-if="feedbackMessage"
          class="libero-reflex__feedback"
          :style="feedbackStyle"
          :class="{ 'is-success': feedbackState === 'success', 'is-fail': feedbackState === 'fail' }"
        >
          {{ feedbackMessage }}
        </div>
      </Transition>
    </div>

    <div v-if="statusMessage" class="libero-reflex__banner" :class="`libero-reflex__banner--${statusType}`" aria-live="polite">
      {{ statusMessage }}
    </div>

    <div class="libero-reflex__cta">
      <button
        type="button"
        class="libero-reflex__primary"
        :disabled="!enabled"
        @click="startGame"
      >
        {{ gameCtaLabel }}
      </button>
      <button type="button" class="libero-reflex__secondary" @click="emit('close')">Chiudi</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue';
import volleyballBall from '../assets/volleyball.svg';

type GameResult = {
  attempts: number;
  successCount: number;
};

const props = defineProps({
  enabled: {
    type: Boolean,
    default: true,
  },
  completed: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits<{
  (e: 'close'): void;
  (e: 'game-finished', payload: GameResult): void;
}>();

const minX = 10;
const maxX = 90;
const baseSpeed = 0.085;
const maxLevel = 5;

const isPlaying = ref(false);
const feedbackState = ref<'success' | 'fail' | ''>('');
const feedbackMessage = ref('');
const ballX = ref(12);
const direction = ref(1);
const speedMultiplier = ref(1);
const targetWidth = ref(0.32);
const targetStart = ref(42);
const targetEnd = ref(58);
const currentLevel = ref(1);
const lives = ref(3);
const statusMessage = ref('');
const statusType = ref<'info' | 'success' | 'warning' | 'error'>('info');
const levelConfigs = [
  { speed: 1.0, targetWidth: 0.4 },
  { speed: 1.3, targetWidth: 0.32 },
  { speed: 1.6, targetWidth: 0.26 },
  { speed: 2.0, targetWidth: 0.2 },
  { speed: 2.4, targetWidth: 0.16 },
];
let animationFrame = 0;
let lastTimestamp = 0;
let feedbackTimer: ReturnType<typeof setTimeout> | null = null;

const gameCtaLabel = computed(() => {
  return 'Gioca';
});

const arenaClasses = computed(() => ({
  'libero-reflex__arena--success': feedbackState.value === 'success',
  'libero-reflex__arena--fail': feedbackState.value === 'fail',
  'libero-reflex__arena--disabled': !props.enabled,
}));

const ballClasses = computed(() => ({
  'libero-reflex__ball--highlight': feedbackState.value === 'success',
  'libero-reflex__ball--fast': speedMultiplier.value > 1.5,
  'libero-reflex__ball--pulse': isPlaying.value,
}));

const ballStyle = computed(() => ({
  left: `${ballX.value}%`,
  transform: `translate(-50%, -50%) scale(${feedbackState.value === 'success' ? 1.08 : 1})`,
}));

const zoneStyle = computed(() => ({
  '--zone-width': `${(targetWidth.value * 100).toFixed(0)}%`,
}));

const feedbackStyle = computed(() => ({
  left: `${ballX.value}%`,
  transform: 'translate(-50%, -110%)',
}));

const levelStyle = computed(() => {
  const palette = ['#38bdf8', '#22c55e', '#fbbf24', '#fb923c', '#ef4444'];
  const index = Math.min(currentLevel.value - 1, palette.length - 1);
  return {
    '--level-color': palette[index],
  };
});

const hearts = computed(() =>
  Array.from({ length: 3 }, (_, index) => ({
    id: index,
    empty: lives.value <= index,
  })),
);

const stopAnimation = () => {
  if (animationFrame) {
    cancelAnimationFrame(animationFrame);
    animationFrame = 0;
  }
  lastTimestamp = 0;
};

const clearFeedback = () => {
  if (feedbackTimer) {
    clearTimeout(feedbackTimer);
    feedbackTimer = null;
  }
  feedbackState.value = '';
  feedbackMessage.value = '';
};

const applyLevelConfig = (level: number) => {
  const config = levelConfigs[level - 1];
  if (!config) return;
  speedMultiplier.value = config.speed;
  targetWidth.value = config.targetWidth;
  const halfWidth = (targetWidth.value * 100) / 2;
  targetStart.value = 50 - halfWidth;
  targetEnd.value = 50 + halfWidth;
};

const resetBall = () => {
  ballX.value = 12;
  direction.value = 1;
};

const animate = (timestamp: number) => {
  if (!isPlaying.value) {
    return;
  }
  if (!lastTimestamp) {
    lastTimestamp = timestamp;
  }
  const delta = timestamp - lastTimestamp;
  lastTimestamp = timestamp;
  const nextPosition = ballX.value + direction.value * delta * baseSpeed * speedMultiplier.value;
  if (nextPosition > maxX || nextPosition < minX) {
    direction.value *= -1;
  }
  const clamped = Math.min(maxX, Math.max(minX, nextPosition));
  ballX.value = clamped;
  animationFrame = window.requestAnimationFrame(animate);
};

const startGame = () => {
  if (!props.enabled) {
    return;
  }
  clearFeedback();
  statusMessage.value = '';
  statusType.value = 'info';
  currentLevel.value = 1;
  lives.value = 3;
  applyLevelConfig(1);
  resetBall();
  isPlaying.value = true;
  animationFrame = window.requestAnimationFrame(animate);
};

const handleTap = () => {
  if (!isPlaying.value) {
    return;
  }

  const inTarget = ballX.value >= targetStart.value && ballX.value <= targetEnd.value;

  if (feedbackTimer) {
    clearTimeout(feedbackTimer);
  }

  if (inTarget) {
    feedbackState.value = 'success';
    feedbackMessage.value = 'Preso!';
    statusMessage.value = '';
    const nextLevel = currentLevel.value + 1;
    currentLevel.value = nextLevel;

    if (nextLevel > maxLevel) {
      statusMessage.value = 'Hai completato tutti i livelli! Il gioco ricomincia dal livello 1.';
      statusType.value = 'success';
      currentLevel.value = 1;
      lives.value = 3;
    }

    applyLevelConfig(currentLevel.value);
  } else {
    feedbackState.value = 'fail';
    feedbackMessage.value = 'Troppo presto / troppo tardi!';
    statusType.value = 'warning';
    if (typeof navigator !== 'undefined' && 'vibrate' in navigator) {
      try {
        navigator.vibrate(80);
      } catch (error) {
        // ignore vibration errors
      }
    }
    lives.value -= 1;

    if (lives.value <= 0) {
      statusMessage.value = 'Hai finito le vite, il gioco è ricominciato dal livello 1.';
      statusType.value = 'error';
      currentLevel.value = 1;
      lives.value = 3;
      applyLevelConfig(1);
    } else {
      statusMessage.value = `Attento! Ti rimangono ${lives.value} vite.`;
    }
  }

  feedbackTimer = setTimeout(clearFeedback, 700);
};

watch(
  () => props.enabled,
  (enabled) => {
    if (!enabled) {
      isPlaying.value = false;
      stopAnimation();
    }
  },
);

onBeforeUnmount(() => {
  stopAnimation();
  clearFeedback();
});
</script>

<style scoped>
.libero-reflex {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  color: #0f172a;
}

.libero-reflex__eyebrow {
  font-size: 0.75rem;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: #0ea5e9;
  margin: 0;
}

.libero-reflex__title {
  font-size: 1.7rem;
  margin: 0;
  color: #0b1021;
}

.libero-reflex__subtitle {
  margin: 0;
  color: #334155;
}

.libero-reflex__instruction {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  border-radius: 999px;
  background: rgba(14, 165, 233, 0.08);
  border: 1px solid rgba(14, 165, 233, 0.18);
  color: #0b172a;
  font-weight: 700;
  box-shadow: 0 8px 20px rgba(14, 165, 233, 0.12);
}

.libero-reflex__instruction-icon {
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background: linear-gradient(135deg, #22c55e, #16a34a);
  color: #f8fafc;
  box-shadow: 0 8px 16px rgba(34, 197, 94, 0.35);
}

.libero-reflex__status {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 0.75rem;
  background: rgba(15, 23, 42, 0.72);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 14px;
  padding: 0.8rem 1rem;
  font-weight: 600;
  color: #e2e8f0;
  backdrop-filter: blur(8px);
  box-shadow: 0 12px 28px rgba(15, 23, 42, 0.25);
}

.libero-reflex__level {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  color: var(--level-color, #38bdf8);
}

.libero-reflex__level strong {
  font-size: 1.1rem;
}

.libero-reflex__level-chip {
  padding: 0.25rem 0.6rem;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.12);
  text-transform: uppercase;
  font-size: 0.75rem;
  letter-spacing: 0.05em;
}

.libero-reflex__lives {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
}

.libero-reflex__lives-label {
  font-size: 0.85rem;
  color: #cbd5e1;
}

.libero-reflex__heart {
  color: #ef4444;
  font-size: 1.3rem;
  text-shadow: 0 4px 12px rgba(239, 68, 68, 0.5);
  animation: heart-glow 1.6s ease-in-out infinite;
}

.libero-reflex__heart.is-empty {
  color: #cbd5e1;
  text-shadow: none;
  opacity: 0.4;
  animation: heart-pop 0.38s ease forwards;
}

.libero-reflex__arena {
  position: relative;
  min-height: 230px;
  border-radius: 18px;
  background: radial-gradient(circle at 50% 20%, rgba(59, 130, 246, 0.18), transparent 35%),
    linear-gradient(180deg, #0b1224, #0f172a 55%, #0b1224);
  overflow: hidden;
  border: 1px solid rgba(255, 255, 255, 0.08);
  box-shadow: 0 12px 40px rgba(15, 23, 42, 0.32), 0 0 0 1px rgba(255, 255, 255, 0.04);
  transition: box-shadow 0.25s ease, transform 0.25s ease;
}

.libero-reflex__arena:focus {
  outline: none;
  box-shadow: 0 0 0 3px rgba(14, 165, 233, 0.35), 0 12px 40px rgba(14, 165, 233, 0.18);
}

.libero-reflex__arena::after {
  content: '';
  position: absolute;
  inset: 0;
  background: radial-gradient(circle at 50% 50%, rgba(0, 0, 0, 0.25), transparent 50%);
  pointer-events: none;
}

.libero-reflex__arena--success {
  box-shadow: 0 20px 44px rgba(34, 197, 94, 0.32), 0 0 0 2px rgba(34, 197, 94, 0.25);
}

.libero-reflex__arena--fail {
  box-shadow: 0 18px 30px rgba(239, 68, 68, 0.28), 0 0 0 2px rgba(239, 68, 68, 0.25);
  animation: arena-flash 0.25s ease;
}

.libero-reflex__arena--disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.libero-reflex__zone {
  position: absolute;
  inset: 0;
  display: flex;
  justify-content: center;
  align-items: center;
  pointer-events: none;
}

.libero-reflex__zone::before {
  content: '';
  width: var(--zone-width, 32%);
  height: 100%;
  background: linear-gradient(180deg, rgba(34, 197, 94, 0.18), rgba(34, 197, 94, 0.42));
  border-left: 2px solid rgba(22, 163, 74, 0.6);
  border-right: 2px solid rgba(22, 163, 74, 0.6);
  border-radius: 14px;
  box-shadow: 0 0 18px rgba(34, 197, 94, 0.45), 0 0 0 2px rgba(74, 222, 128, 0.18);
  animation: zone-breathe 2.4s ease-in-out infinite;
  transition: width 0.5s ease, box-shadow 0.4s ease;
}

.libero-reflex__zone-arrows {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  pointer-events: none;
  width: calc(var(--zone-width, 32%) + 52px);
  left: 50%;
  transform: translateX(-50%);
}

.libero-reflex__zone-arrows::before,
.libero-reflex__zone-arrows::after {
  content: '';
  width: 12px;
  height: 18px;
  border-radius: 4px;
  background: linear-gradient(135deg, rgba(74, 222, 128, 0.9), rgba(16, 185, 129, 0.4));
  clip-path: polygon(100% 50%, 0 0, 0 100%);
  opacity: 0.55;
  animation: arrow-pulse 1.2s ease-in-out infinite;
}

.libero-reflex__zone-arrows::after {
  transform: rotate(180deg);
}

.libero-reflex__zone-label {
  position: absolute;
  padding: 0.25rem 0.55rem;
  background: rgba(236, 253, 245, 0.9);
  color: #166534;
  border-radius: 999px;
  font-size: 0.7rem;
  font-weight: 800;
  border: 1px solid rgba(22, 101, 52, 0.2);
  box-shadow: 0 8px 20px rgba(16, 185, 129, 0.25);
}

.libero-reflex__ball {
  position: absolute;
  top: 50%;
  width: 64px;
  height: 64px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  transition: transform 0.12s ease, box-shadow 0.12s ease, filter 0.12s ease;
  filter: drop-shadow(0 8px 12px rgba(0, 0, 0, 0.35));
  z-index: 2;
}

.libero-reflex__ball img {
  width: 56px;
  height: 56px;
}

.libero-reflex__ball--pulse {
  animation: volley-bounce 1s ease-in-out infinite;
}

.libero-reflex__ball--fast {
  filter: drop-shadow(0 10px 14px rgba(0, 0, 0, 0.35)) blur(1px);
}

.libero-reflex__ball--highlight {
  box-shadow: 0 16px 30px rgba(74, 222, 128, 0.45);
  animation: ball-burst 0.45s ease;
}

.libero-reflex__feedback {
  position: absolute;
  top: 50%;
  padding: 0.45rem 0.8rem;
  border-radius: 999px;
  font-weight: 800;
  letter-spacing: 0.02em;
  background: rgba(15, 23, 42, 0.9);
  color: #fff;
  border: 1px solid rgba(255, 255, 255, 0.1);
  box-shadow: 0 8px 20px rgba(0, 0, 0, 0.35);
  z-index: 3;
}

.libero-reflex__feedback.is-success {
  color: #22c55e;
}

.libero-reflex__feedback.is-fail {
  color: #ef4444;
}

.libero-reflex__perfect {
  position: absolute;
  top: 48%;
  left: 50%;
  transform: translate(-50%, -50%);
  color: #f8fafc;
  font-weight: 900;
  letter-spacing: 0.08em;
  text-shadow: 0 8px 18px rgba(34, 197, 94, 0.35);
  z-index: 3;
}

.libero-reflex__level-toast {
  position: absolute;
  top: 14px;
  left: 50%;
  transform: translateX(-50%);
  padding: 0.35rem 0.75rem;
  background: rgba(15, 23, 42, 0.85);
  color: #e2e8f0;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 12px;
  letter-spacing: 0.08em;
  font-weight: 800;
  z-index: 3;
}

.libero-reflex-feedback-enter-active,
.libero-reflex-feedback-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.libero-reflex-feedback-enter-from,
.libero-reflex-feedback-leave-to {
  opacity: 0;
  transform: translate(-50%, -90%);
}

.libero-reflex-perfect-enter-active,
.libero-reflex-perfect-leave-active {
  transition: opacity 0.6s ease, transform 0.6s ease;
}

.libero-reflex-perfect-enter-from,
.libero-reflex-perfect-leave-to {
  opacity: 0;
  transform: translate(-50%, -40%);
}

.libero-reflex-level-enter-active,
.libero-reflex-level-leave-active {
  transition: opacity 0.8s ease, transform 0.8s ease;
}

.libero-reflex-level-enter-from,
.libero-reflex-level-leave-to {
  opacity: 0;
  transform: translate(-50%, 20px);
}

.libero-reflex__cta {
  display: flex;
  gap: 0.9rem;
  flex-wrap: wrap;
  margin-top: 1rem;
}

.libero-reflex__primary,
.libero-reflex__secondary {
  border: none;
  border-radius: 14px;
  padding: 1rem 1.2rem;
  font-size: 1rem;
  font-weight: 800;
  cursor: pointer;
  transition: transform 0.12s ease, box-shadow 0.12s ease, opacity 0.2s ease, background 0.2s ease;
}

.libero-reflex__primary {
  background: linear-gradient(135deg, #22c55e, #16a34a);
  color: #f8fafc;
  box-shadow: 0 14px 26px rgba(34, 197, 94, 0.35);
  flex: 1 1 180px;
}

.libero-reflex__primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  box-shadow: none;
}

.libero-reflex__primary:not(:disabled):active {
  transform: translateY(1px);
}

.libero-reflex__secondary {
  background: transparent;
  color: #0f172a;
  text-decoration: underline;
  font-weight: 700;
  padding-inline: 0;
}

.libero-reflex__secondary:active {
  transform: translateY(1px);
}

.libero-reflex__banner {
  padding: 0.7rem 0.9rem;
  border-radius: 12px;
  font-weight: 700;
  border: 1px solid transparent;
  background: rgba(15, 23, 42, 0.06);
  color: #0f172a;
}

.libero-reflex__banner--info {
  border-color: #e2e8f0;
}

.libero-reflex__banner--success {
  background: #ecfdf3;
  border-color: #bbf7d0;
  color: #166534;
}

.libero-reflex__banner--warning {
  background: #fff7ed;
  border-color: #fed7aa;
  color: #9a3412;
}

.libero-reflex__banner--error {
  background: #fef2f2;
  border-color: #fecdd3;
  color: #b91c1c;
}

@keyframes zone-breathe {
  0% {
    box-shadow: 0 0 14px rgba(34, 197, 94, 0.35), 0 0 0 2px rgba(74, 222, 128, 0.16);
  }
  50% {
    box-shadow: 0 0 22px rgba(34, 197, 94, 0.55), 0 0 0 4px rgba(74, 222, 128, 0.2);
  }
  100% {
    box-shadow: 0 0 14px rgba(34, 197, 94, 0.35), 0 0 0 2px rgba(74, 222, 128, 0.16);
  }
}

@keyframes arrow-pulse {
  0%,
  100% {
    opacity: 0.3;
    transform: translateX(0);
  }
  50% {
    opacity: 0.85;
    transform: translateX(3px);
  }
}

@keyframes heart-glow {
  0%,
  100% {
    filter: drop-shadow(0 0 0 rgba(239, 68, 68, 0.2));
  }
  50% {
    filter: drop-shadow(0 0 12px rgba(239, 68, 68, 0.45));
  }
}

@keyframes heart-pop {
  0% {
    transform: scale(1);
    opacity: 0.9;
  }
  40% {
    transform: scale(1.25);
    opacity: 0.8;
  }
  100% {
    transform: scale(0.6) rotate(-8deg);
    opacity: 0.3;
  }
}

@keyframes volley-bounce {
  0%,
  100% {
    transform: translate(-50%, -50%) translateY(0);
  }
  50% {
    transform: translate(-50%, -50%) translateY(-6px);
  }
}

@keyframes ball-burst {
  0% {
    transform: translate(-50%, -50%) scale(1);
    box-shadow: 0 12px 28px rgba(74, 222, 128, 0.15);
  }
  50% {
    transform: translate(-50%, -50%) scale(1.15);
    box-shadow: 0 0 26px rgba(74, 222, 128, 0.45);
  }
  100% {
    transform: translate(-50%, -50%) scale(1.05);
    box-shadow: 0 14px 24px rgba(74, 222, 128, 0.3);
  }
}

@keyframes arena-flash {
  0% {
    box-shadow: 0 18px 30px rgba(239, 68, 68, 0.4), 0 0 0 3px rgba(239, 68, 68, 0.28);
  }
  100% {
    box-shadow: 0 18px 30px rgba(239, 68, 68, 0.18), 0 0 0 2px rgba(239, 68, 68, 0.15);
  }
}

@media (max-width: 640px) {
  .libero-reflex__arena {
    min-height: 200px;
  }

  .libero-reflex__ball {
    width: 56px;
    height: 56px;
  }

  .libero-reflex__ball img {
    width: 48px;
    height: 48px;
  }
}
</style>
