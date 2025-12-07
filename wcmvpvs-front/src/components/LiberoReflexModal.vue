<template>
  <div class="libero-reflex">
    <p class="libero-reflex__eyebrow">Mini missione</p>
    <h4 class="libero-reflex__title">Libero Reflex</h4>
    <p class="libero-reflex__subtitle">Tocca al momento giusto!</p>

    <div class="libero-reflex__status">
      <span class="libero-reflex__attempts">Tentativi: {{ attempts }} / {{ maxAttempts }}</span>
      <span class="libero-reflex__success" :class="{ 'is-positive': successCount >= 2 }">
        Blocchi riusciti: {{ successCount }}
      </span>
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
      <div class="libero-reflex__zone" aria-hidden="true">
        <span class="libero-reflex__zone-label">Zona verde</span>
      </div>

      <div class="libero-reflex__ball" :class="ballClasses" :style="ballStyle">
        <img :src="volleyballBall" alt="Palla da volley" />
      </div>

      <Transition name="libero-reflex-feedback">
        <div
          v-if="feedbackMessage"
          class="libero-reflex__feedback"
          :class="{ 'is-success': feedbackState === 'success', 'is-fail': feedbackState === 'fail' }"
        >
          {{ feedbackMessage }}
        </div>
      </Transition>
    </div>

    <div class="libero-reflex__cta">
      <button
        type="button"
        class="libero-reflex__primary"
        :disabled="!enabled || isPlaying"
        @click="startGame"
      >
        {{ gameCtaLabel }}
      </button>
      <button type="button" class="libero-reflex__secondary" @click="emit('close')">Chiudi</button>
    </div>

    <div v-if="gameOver" class="libero-reflex__summary" aria-live="polite">
      <p class="libero-reflex__summary-title">{{ summaryTitle }}</p>
      <p class="libero-reflex__summary-subtitle">{{ summarySubtitle }}</p>
      <div class="libero-reflex__summary-actions">
        <button type="button" class="libero-reflex__primary" @click="startGame">Riprova</button>
        <button type="button" class="libero-reflex__secondary" @click="emit('close')">Chiudi</button>
      </div>
    </div>

    <p v-if="completed" class="libero-reflex__completion-hint">Missione già completata: puoi rigiocare, ma la ricompensa è già stata assegnata.</p>
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

const maxAttempts = 3;
const targetStart = 42;
const targetEnd = 58;
const minX = 10;
const maxX = 90;
const baseSpeed = 0.085;

const attempts = ref(0);
const successCount = ref(0);
const isPlaying = ref(false);
const gameOver = ref(false);
const feedbackState = ref<'success' | 'fail' | ''>('');
const feedbackMessage = ref('');
const ballX = ref(12);
const direction = ref(1);
let animationFrame = 0;
let lastTimestamp = 0;
let feedbackTimer: ReturnType<typeof setTimeout> | null = null;

const gameCtaLabel = computed(() => {
  if (isPlaying.value) {
    return 'In gioco...';
  }
  if (gameOver.value) {
    return 'Gioca di nuovo';
  }
  return 'Gioca';
});

const arenaClasses = computed(() => ({
  'libero-reflex__arena--success': feedbackState.value === 'success',
  'libero-reflex__arena--fail': feedbackState.value === 'fail',
  'libero-reflex__arena--disabled': !props.enabled,
  'libero-reflex__arena--done': gameOver.value,
}));

const ballClasses = computed(() => ({
  'libero-reflex__ball--highlight': feedbackState.value === 'success',
}));

const ballStyle = computed(() => ({
  left: `${ballX.value}%`,
  transform: `translate(-50%, -50%) scale(${feedbackState.value === 'success' ? 1.08 : 1})`,
}));

const summaryTitle = computed(() => {
  if (successCount.value >= 2) {
    return 'Ottimo! Hai bloccato la palla!';
  }
  return 'Quasi! Riprova per ottenere la chance bonus.';
});

const summarySubtitle = computed(() => {
  if (successCount.value >= 2) {
    return 'Hai sincronizzato i riflessi: +1 chance assegnata.';
  }
  if (attempts.value === maxAttempts) {
    return '3 tentativi consumati, concentra i riflessi e riparti!';
  }
  return 'Premi Gioca per una nuova sfida lampo.';
});

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

const resetGame = () => {
  attempts.value = 0;
  successCount.value = 0;
  gameOver.value = false;
  feedbackState.value = '';
  feedbackMessage.value = '';
  ballX.value = 12;
  direction.value = 1;
  stopAnimation();
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
  const nextPosition = ballX.value + direction.value * delta * baseSpeed;
  if (nextPosition > maxX || nextPosition < minX) {
    direction.value *= -1;
  }
  const clamped = Math.min(maxX, Math.max(minX, nextPosition));
  ballX.value = clamped;
  animationFrame = window.requestAnimationFrame(animate);
};

const finishGame = () => {
  gameOver.value = true;
  isPlaying.value = false;
  stopAnimation();
  emit('game-finished', { attempts: attempts.value, successCount: successCount.value });
};

const startGame = () => {
  if (!props.enabled) {
    return;
  }
  clearFeedback();
  resetGame();
  isPlaying.value = true;
  gameOver.value = false;
  animationFrame = window.requestAnimationFrame(animate);
};

const handleTap = () => {
  if (!isPlaying.value || gameOver.value) {
    return;
  }

  const nextAttempts = attempts.value + 1;
  attempts.value = nextAttempts;
  const inTarget = ballX.value >= targetStart && ballX.value <= targetEnd;

  if (feedbackTimer) {
    clearTimeout(feedbackTimer);
  }

  if (inTarget) {
    successCount.value += 1;
    feedbackState.value = 'success';
    feedbackMessage.value = 'BLOCK!';
  } else {
    feedbackState.value = 'fail';
    feedbackMessage.value = 'Troppo presto o troppo tardi!';
    if (typeof navigator !== 'undefined' && 'vibrate' in navigator) {
      try {
        navigator.vibrate(80);
      } catch (error) {
        // ignore vibration errors
      }
    }
  }

  feedbackTimer = setTimeout(clearFeedback, 800);

  if (nextAttempts >= maxAttempts) {
    finishGame();
  }
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
  font-size: 1.6rem;
  margin: 0;
  color: #0b1021;
}

.libero-reflex__subtitle {
  margin: 0;
  color: #334155;
}

.libero-reflex__status {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  padding: 0.75rem 1rem;
  font-weight: 600;
  color: #0b132a;
}

.libero-reflex__success {
  color: #ef4444;
  font-weight: 700;
}

.libero-reflex__success.is-positive {
  color: #16a34a;
}

.libero-reflex__arena {
  position: relative;
  min-height: 220px;
  border-radius: 18px;
  background: linear-gradient(180deg, #0f172a, #111827);
  overflow: hidden;
  border: 1px solid #1f2937;
  box-shadow: 0 12px 38px rgba(15, 23, 42, 0.28);
  transition: box-shadow 0.25s ease, transform 0.25s ease;
}

.libero-reflex__arena:focus {
  outline: none;
  box-shadow: 0 0 0 3px rgba(14, 165, 233, 0.4);
}

.libero-reflex__arena--success {
  box-shadow: 0 20px 44px rgba(34, 197, 94, 0.32), 0 0 0 2px rgba(34, 197, 94, 0.25);
}

.libero-reflex__arena--fail {
  box-shadow: 0 18px 30px rgba(239, 68, 68, 0.28), 0 0 0 2px rgba(239, 68, 68, 0.25);
}

.libero-reflex__arena--disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.libero-reflex__arena--done {
  border-color: rgba(14, 165, 233, 0.4);
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
  width: 32%;
  height: 100%;
  background: linear-gradient(180deg, rgba(34, 197, 94, 0.18), rgba(34, 197, 94, 0.32));
  border-left: 2px dashed rgba(22, 163, 74, 0.8);
  border-right: 2px dashed rgba(22, 163, 74, 0.8);
  border-radius: 12px;
}

.libero-reflex__zone-label {
  position: absolute;
  padding: 0.35rem 0.65rem;
  background: #ecfdf3;
  color: #166534;
  border-radius: 999px;
  font-size: 0.8rem;
  font-weight: 700;
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
  transition: transform 0.12s ease, box-shadow 0.12s ease;
  filter: drop-shadow(0 8px 12px rgba(0, 0, 0, 0.35));
}

.libero-reflex__ball img {
  width: 56px;
  height: 56px;
}

.libero-reflex__ball--highlight {
  box-shadow: 0 14px 28px rgba(34, 197, 94, 0.4);
}

.libero-reflex__feedback {
  position: absolute;
  bottom: 12px;
  left: 50%;
  transform: translateX(-50%);
  padding: 0.55rem 0.9rem;
  border-radius: 12px;
  font-weight: 800;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  background: rgba(15, 23, 42, 0.85);
  color: #fff;
  border: 1px solid rgba(255, 255, 255, 0.12);
  box-shadow: 0 8px 20px rgba(0, 0, 0, 0.35);
}

.libero-reflex__feedback.is-success {
  color: #22c55e;
}

.libero-reflex__feedback.is-fail {
  color: #ef4444;
}

.libero-reflex-feedback-enter-active,
.libero-reflex-feedback-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.libero-reflex-feedback-enter-from,
.libero-reflex-feedback-leave-to {
  opacity: 0;
  transform: translate(-50%, 8px);
}

.libero-reflex__cta {
  display: flex;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.libero-reflex__primary,
.libero-reflex__secondary {
  border: none;
  border-radius: 12px;
  padding: 0.85rem 1.1rem;
  font-size: 1rem;
  font-weight: 700;
  cursor: pointer;
  transition: transform 0.12s ease, box-shadow 0.12s ease, opacity 0.2s ease;
}

.libero-reflex__primary {
  background: linear-gradient(135deg, #22c55e, #16a34a);
  color: #f8fafc;
  box-shadow: 0 12px 20px rgba(34, 197, 94, 0.3);
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
  background: #0f172a;
  color: #e2e8f0;
  box-shadow: 0 8px 16px rgba(15, 23, 42, 0.25);
}

.libero-reflex__secondary:active {
  transform: translateY(1px);
}

.libero-reflex__summary {
  background: #ecfeff;
  border: 1px solid #bae6fd;
  border-radius: 14px;
  padding: 1rem;
  color: #0c4a6e;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.45);
}

.libero-reflex__summary-title {
  margin: 0 0 0.25rem;
  font-size: 1.2rem;
  font-weight: 800;
}

.libero-reflex__summary-subtitle {
  margin: 0 0 0.75rem;
}

.libero-reflex__summary-actions {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.libero-reflex__completion-hint {
  margin: 0;
  padding: 0.5rem 0.75rem;
  background: #fef9c3;
  border: 1px solid #fcd34d;
  border-radius: 10px;
  color: #854d0e;
  font-weight: 600;
}

@media (max-width: 640px) {
  .libero-reflex__arena {
    min-height: 190px;
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
