<template>
  <section class="px-4">
    <div
      class="reaction-card"
      :class="{ 'reaction-card--disabled': !props.enabled }"
    >
      <div class="reaction-card__decor"></div>
      <div class="reaction-card__content">
        <header class="reaction-card__header">
          <p class="reaction-card__eyebrow">Mini gioco</p>
          <h2 class="reaction-card__title">Reaction Test</h2>
          <p class="reaction-card__subtitle">
            Tocca la palla appena appare sullo schermo e scopri se sei più veloce degli altri tifosi!
          </p>
        </header>

        <div class="reaction-stage" @pointerdown.prevent="handleStageTap">
          <div v-if="loadingStatus" class="reaction-stage__loading" role="status" aria-live="polite">
            <span class="reaction-spinner" aria-hidden="true"></span>
            <p>Caricamento…</p>
          </div>

          <template v-else>
            <div v-if="gameState === 'countdown'" class="reaction-stage__countdown">
              <span class="reaction-stage__countdown-number">{{ countdownValue }}</span>
              <p class="reaction-stage__hint">Preparati…</p>
            </div>

            <div v-else-if="gameState === 'waiting'" class="reaction-stage__message">
              <p class="reaction-stage__hint">Mantieni la concentrazione… la palla sta per arrivare!</p>
            </div>

            <div v-else-if="gameState === 'ready'" class="reaction-stage__ball-wrapper">
              <div class="reaction-ball" role="button" aria-label="Tocca la palla"></div>
              <p class="reaction-stage__hint">Vai! Tocca la palla!</p>
            </div>

            <div v-else-if="gameState === 'result'" class="reaction-stage__result">
              <p class="reaction-stage__result-text" aria-live="polite">{{ resultMessage }}</p>
              <p v-if="comparisonMessage" class="reaction-stage__comparison">{{ comparisonMessage }}</p>
            </div>

            <div v-else-if="gameState === 'cooldown'" class="reaction-stage__message">
              <p class="reaction-stage__hint">{{ cooldownMessage }}</p>
            </div>

            <div v-else class="reaction-stage__instructions">
              <p>
                Premi <strong>Inizia</strong>, attendi il conto alla rovescia e tocca la palla appena appare. Se tocchi troppo
                presto dovrai aspettare un minuto prima di riprovare.
              </p>
            </div>
          </template>
        </div>

        <div class="reaction-actions">
          <button
            type="button"
            class="reaction-button"
            :class="{ 'reaction-button--disabled': startDisabled }"
            :disabled="startDisabled"
            @click="startGame"
          >
            {{ isInProgress ? 'In corso…' : 'Inizia' }}
          </button>
          <p v-if="cooldownLabel" class="reaction-cooldown" aria-live="polite">{{ cooldownLabel }}</p>
        </div>

        <p v-if="infoMessage" class="reaction-message">{{ infoMessage }}</p>
        <p v-if="errorMessage" class="reaction-message reaction-message--error">{{ errorMessage }}</p>

        <div class="reaction-stats">
          <div class="reaction-stat">
            <p class="reaction-stat__label">Media dei tifosi</p>
            <p class="reaction-stat__value">{{ averageDisplay }}</p>
            <p class="reaction-stat__hint">
              {{ attemptsCount > 0 ? `Calcolata su ${attemptsCount} tentativi` : 'Ancora nessun tentativo registrato' }}
            </p>
          </div>
          <div class="reaction-stat">
            <p class="reaction-stat__label">Il tuo ultimo risultato</p>
            <p class="reaction-stat__value">{{ lastResultDisplay }}</p>
            <p class="reaction-stat__hint">
              {{ lastResultHint }}
            </p>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { fetchReactionTestStatus, submitReactionTestResult } from '../api';

const props = defineProps({
  eventId: {
    type: Number,
    required: true,
  },
  enabled: {
    type: Boolean,
    default: false,
  },
});

const loadingStatus = ref(false);
const isSubmitting = ref(false);
const gameState = ref('idle');
const countdownValue = ref(3);
const infoMessage = ref('');
const errorMessage = ref('');
const resultMessage = ref('');
const comparisonMessage = ref('');
const attemptsCount = ref(0);
const averageMs = ref(0);
const lastResultMs = ref(null);
const cooldownUntil = ref(null);
const nowTs = ref(Date.now());
const reactionStartAt = ref(0);

let countdownTimer = null;
let delayTimer = null;
let nowTimer = null;

const cooldownStorageKey = computed(() => (props.eventId ? `reaction_test_cooldown_${props.eventId}` : ''));

const isInProgress = computed(() => ['countdown', 'waiting', 'ready'].includes(gameState.value));
const showCooldown = computed(() => cooldownUntil.value instanceof Date);

const cooldownRemainingMs = computed(() => {
  if (!(cooldownUntil.value instanceof Date)) {
    return 0;
  }
  return Math.max(0, cooldownUntil.value.getTime() - nowTs.value);
});

const isCooldownActive = computed(() => cooldownRemainingMs.value > 0);

const cooldownLabel = computed(() => {
  if (!isCooldownActive.value) {
    return '';
  }
  const seconds = Math.max(1, Math.ceil(cooldownRemainingMs.value / 1000));
  if (seconds >= 60) {
    const minutes = Math.ceil(seconds / 60);
    return `Puoi riprovare tra ${minutes} minuto${minutes > 1 ? 'i' : ''}.`;
  }
  return `Puoi riprovare tra ${seconds} secondo${seconds !== 1 ? 'i' : ''}.`;
});

const cooldownMessage = computed(() =>
  cooldownLabel.value || 'Attendi un momento prima di riprovare!',
);

const averageDisplay = computed(() => {
  if (averageMs.value <= 0 || Number.isNaN(averageMs.value)) {
    return '—';
  }
  return `${Math.round(averageMs.value)} ms`;
});

const lastResultDisplay = computed(() => {
  if (typeof lastResultMs.value === 'number') {
    return `${lastResultMs.value} ms`;
  }
  return 'Ancora nessun tentativo';
});

const lastResultHint = computed(() => {
  if (typeof lastResultMs.value === 'number' && averageMs.value > 0) {
    if (lastResultMs.value < Math.round(averageMs.value)) {
      return 'Più veloce della media! 🔥';
    }
    if (lastResultMs.value === Math.round(averageMs.value)) {
      return 'In perfetta media! 💪';
    }
    return 'Un pizzico più lento della media, ritenta!';
  }
  return 'Gioca per registrare il tuo tempo.';
});

const startDisabled = computed(
  () =>
    !props.enabled ||
    isInProgress.value ||
    isSubmitting.value ||
    loadingStatus.value ||
    isCooldownActive.value,
);

function clearTimers() {
  if (countdownTimer) {
    window.clearInterval(countdownTimer);
    countdownTimer = null;
  }
  if (delayTimer) {
    window.clearTimeout(delayTimer);
    delayTimer = null;
  }
}

function resetStage() {
  clearTimers();
  gameState.value = 'idle';
  countdownValue.value = 3;
  infoMessage.value = '';
  errorMessage.value = '';
  resultMessage.value = '';
  comparisonMessage.value = '';
}

function updateNow() {
  nowTs.value = Date.now();
}

function setCooldownUntil(date) {
  const key = cooldownStorageKey.value;
  if (date instanceof Date && !Number.isNaN(date.getTime())) {
    cooldownUntil.value = date;
    if (key && typeof window !== 'undefined') {
      window.localStorage.setItem(key, date.toISOString());
    }
  } else {
    cooldownUntil.value = null;
    if (key && typeof window !== 'undefined') {
      window.localStorage.removeItem(key);
    }
  }
}

function restoreCooldown() {
  const key = cooldownStorageKey.value;
  if (!key || typeof window === 'undefined') {
    return;
  }
  const stored = window.localStorage.getItem(key);
  if (!stored) {
    setCooldownUntil(null);
    return;
  }
  const parsed = new Date(stored);
  if (!Number.isNaN(parsed.getTime())) {
    setCooldownUntil(parsed);
  } else {
    setCooldownUntil(null);
  }
}

function ensureCooldown(date) {
  if (!(date instanceof Date) || Number.isNaN(date.getTime())) {
    return;
  }
  if (!(cooldownUntil.value instanceof Date) || date.getTime() > cooldownUntil.value.getTime()) {
    setCooldownUntil(date);
  }
}

async function loadStatus() {
  if (!props.eventId || !props.enabled) {
    return;
  }
  loadingStatus.value = true;
  errorMessage.value = '';
  try {
    const { ok, data } = await fetchReactionTestStatus(props.eventId);
    if (!ok) {
      throw new Error('status_error');
    }
    attemptsCount.value = Number.isFinite(data?.attempts) ? data.attempts : 0;
    averageMs.value = Number.isFinite(data?.average_ms) ? data.average_ms : 0;
    lastResultMs.value = typeof data?.last_result_ms === 'number' ? data.last_result_ms : null;
    if (typeof data?.next_allowed_at === 'string') {
      const next = new Date(data.next_allowed_at);
      if (!Number.isNaN(next.getTime())) {
        ensureCooldown(next);
      }
    }
  } catch (error) {
    console.error('reaction test status error', error);
    errorMessage.value = 'Impossibile recuperare le statistiche del Reaction Test.';
  } finally {
    loadingStatus.value = false;
  }
}

function beginCountdown() {
  gameState.value = 'countdown';
  countdownValue.value = 3;
  infoMessage.value = 'Il conto alla rovescia è partito!';
  countdownTimer = window.setInterval(() => {
    countdownValue.value -= 1;
    if (countdownValue.value <= 0) {
      clearTimers();
      beginWaitingPhase();
    }
  }, 1000);
}

function beginWaitingPhase() {
  gameState.value = 'waiting';
  infoMessage.value = 'Appena vedi la palla, toccala al volo!';
  const delay = 1200 + Math.random() * 2000;
  delayTimer = window.setTimeout(() => {
    gameState.value = 'ready';
    infoMessage.value = 'Vai! Tocca la palla!';
    reactionStartAt.value = performance.now();
  }, delay);
}

function handleFalseStart() {
  clearTimers();
  gameState.value = 'cooldown';
  errorMessage.value = 'Troppo presto, riprova!';
  infoMessage.value = '';
  const next = new Date(Date.now() + 60_000);
  ensureCooldown(next);
}

async function finalizeReaction() {
  clearTimers();
  gameState.value = 'result';
  infoMessage.value = 'Misurazione completata!';
  errorMessage.value = '';

  const measured = Math.max(0, Math.round(performance.now() - reactionStartAt.value));
  resultMessage.value = `Hai reagito in ${measured} millisecondi.`;
  ensureCooldown(new Date(Date.now() + 60_000));
  isSubmitting.value = true;
  try {
    const { ok, data, status } = await submitReactionTestResult(props.eventId, measured);
    if (!ok) {
      if (status === 429 && typeof data?.next_allowed_at === 'string') {
        const next = new Date(data.next_allowed_at);
        ensureCooldown(next);
        errorMessage.value = data?.message || 'Aspetta qualche secondo prima di riprovare.';
        gameState.value = 'cooldown';
        return;
      }
      throw new Error('submit_error');
    }

    attemptsCount.value = Number.isFinite(data?.attempts) ? data.attempts : attemptsCount.value;
    averageMs.value = Number.isFinite(data?.average_ms) ? data.average_ms : averageMs.value;
    lastResultMs.value = typeof data?.reaction_time_ms === 'number' ? data.reaction_time_ms : measured;
    const roundedAverage = averageMs.value > 0 ? Math.round(averageMs.value) : null;
    if (data?.faster_than_average) {
      comparisonMessage.value = 'Più veloce della media! ⚡';
    } else if (
      typeof lastResultMs.value === 'number' &&
      typeof roundedAverage === 'number' &&
      lastResultMs.value === roundedAverage
    ) {
      comparisonMessage.value = 'In perfetta media! 💪';
    } else {
      comparisonMessage.value = 'Un filo più lento della media, ma puoi migliorare!';
    }

    if (typeof data?.next_allowed_at === 'string') {
      const next = new Date(data.next_allowed_at);
      ensureCooldown(next);
    }
  } catch (error) {
    console.error('reaction test submit error', error);
    errorMessage.value = 'Non siamo riusciti a salvare il risultato. Riprova tra un attimo!';
  } finally {
    isSubmitting.value = false;
  }
}

function startGame() {
  if (startDisabled.value) {
    return;
  }
  resetStage();
  beginCountdown();
}

function handleStageTap() {
  if (!props.enabled) {
    return;
  }
  if (gameState.value === 'ready') {
    finalizeReaction();
    return;
  }
  if (gameState.value === 'countdown' || gameState.value === 'waiting') {
    handleFalseStart();
  }
}

watch(
  () => [props.eventId, props.enabled],
  async ([eventId, enabled]) => {
    resetStage();
    attemptsCount.value = 0;
    averageMs.value = 0;
    lastResultMs.value = null;
    if (!eventId || !enabled) {
      return;
    }
    restoreCooldown();
    await loadStatus();
  },
  { immediate: true },
);

watch(cooldownRemainingMs, (remaining) => {
  if (remaining <= 0 && showCooldown.value) {
    setCooldownUntil(null);
  }
});

onMounted(() => {
  updateNow();
  nowTimer = window.setInterval(updateNow, 500);
});

onBeforeUnmount(() => {
  clearTimers();
  if (nowTimer) {
    window.clearInterval(nowTimer);
    nowTimer = null;
  }
});
</script>

<style scoped>
.reaction-card {
  position: relative;
  overflow: hidden;
  border-radius: 2.25rem;
  border: 1px solid rgba(148, 163, 184, 0.2);
  background: radial-gradient(circle at top left, rgba(30, 64, 175, 0.35), transparent 55%),
    linear-gradient(180deg, rgba(15, 23, 42, 0.92) 0%, rgba(30, 41, 59, 0.95) 50%, rgba(15, 23, 42, 0.92) 100%);
  color: #e2e8f0;
  box-shadow: 0 26px 52px rgba(8, 15, 28, 0.55);
  padding: 2.5rem 2rem;
}

.reaction-card--disabled {
  opacity: 0.6;
  filter: grayscale(0.1);
}

.reaction-card__decor {
  position: absolute;
  inset: 0;
  background: linear-gradient(140deg, rgba(14, 165, 233, 0.08), transparent 60%);
  pointer-events: none;
}

.reaction-card__content {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.reaction-card__header {
  text-align: left;
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.reaction-card__eyebrow {
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.32em;
  color: rgba(125, 211, 252, 0.9);
  font-weight: 600;
}

.reaction-card__title {
  font-size: 1.75rem;
  font-weight: 700;
  color: #f8fafc;
}

.reaction-card__subtitle {
  font-size: 0.95rem;
  color: rgba(226, 232, 240, 0.8);
}

.reaction-stage {
  position: relative;
  border-radius: 1.75rem;
  border: 1px dashed rgba(96, 165, 250, 0.4);
  background: rgba(15, 23, 42, 0.6);
  padding: 2rem 1.5rem;
  text-align: center;
  min-height: 12rem;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  gap: 1rem;
  transition: border-color 0.3s ease, transform 0.3s ease;
}

.reaction-stage:active {
  transform: scale(0.99);
}

.reaction-stage__loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.75rem;
  font-size: 0.95rem;
}

.reaction-spinner {
  width: 2.5rem;
  height: 2.5rem;
  border-radius: 50%;
  border: 3px solid rgba(125, 211, 252, 0.35);
  border-top-color: rgba(56, 189, 248, 0.85);
  animation: reaction-spin 1s linear infinite;
}

.reaction-stage__countdown-number {
  font-size: 3.75rem;
  font-weight: 700;
  line-height: 1;
  color: #38bdf8;
}

.reaction-stage__hint {
  font-size: 1rem;
  color: rgba(226, 232, 240, 0.85);
}

.reaction-stage__ball-wrapper {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1.25rem;
}

.reaction-ball {
  width: 5rem;
  height: 5rem;
  border-radius: 50%;
  background: radial-gradient(circle at 30% 30%, #fef08a, #f97316 70%, #ea580c 100%);
  box-shadow: 0 12px 28px rgba(249, 115, 22, 0.45);
  animation: reaction-bounce 0.6s ease-in-out infinite alternate;
}

.reaction-stage__result-text {
  font-size: 1.2rem;
  font-weight: 600;
  color: #fcd34d;
}

.reaction-stage__comparison {
  font-size: 0.95rem;
  color: rgba(226, 232, 240, 0.9);
}

.reaction-stage__instructions {
  font-size: 0.95rem;
  color: rgba(226, 232, 240, 0.85);
  line-height: 1.5;
}

.reaction-actions {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 0.75rem;
}

.reaction-button {
  background: linear-gradient(120deg, #38bdf8, #2563eb);
  color: #0f172a;
  font-weight: 700;
  border: none;
  border-radius: 9999px;
  padding: 0.75rem 1.75rem;
  font-size: 1rem;
  cursor: pointer;
  box-shadow: 0 18px 32px rgba(56, 189, 248, 0.35);
  transition: transform 0.2s ease, box-shadow 0.2s ease, opacity 0.2s ease;
}

.reaction-button:hover:not(.reaction-button--disabled) {
  transform: translateY(-1px);
  box-shadow: 0 22px 38px rgba(56, 189, 248, 0.45);
}

.reaction-button--disabled {
  cursor: not-allowed;
  opacity: 0.6;
  box-shadow: none;
}

.reaction-cooldown {
  font-size: 0.9rem;
  color: rgba(148, 163, 184, 0.9);
}

.reaction-message {
  font-size: 0.95rem;
  color: rgba(226, 232, 240, 0.92);
}

.reaction-message--error {
  color: #fca5a5;
}

.reaction-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 1rem;
}

.reaction-stat {
  border-radius: 1.5rem;
  background: rgba(15, 23, 42, 0.65);
  padding: 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  border: 1px solid rgba(148, 163, 184, 0.18);
}

.reaction-stat__label {
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.2em;
  color: rgba(148, 163, 184, 0.85);
}

.reaction-stat__value {
  font-size: 1.4rem;
  font-weight: 700;
  color: #f8fafc;
}

.reaction-stat__hint {
  font-size: 0.9rem;
  color: rgba(148, 163, 184, 0.85);
}

@keyframes reaction-spin {
  to {
    transform: rotate(360deg);
  }
}

@keyframes reaction-bounce {
  from {
    transform: translateY(0);
  }
  to {
    transform: translateY(-12px);
  }
}

@media (min-width: 768px) {
  .reaction-card {
    padding: 3rem 3rem 3.5rem;
  }

  .reaction-stage {
    min-height: 14rem;
  }
}
</style>
