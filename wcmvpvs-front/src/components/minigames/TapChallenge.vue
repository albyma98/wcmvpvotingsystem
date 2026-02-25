<template>
  <div class="tap-challenge flex h-full min-h-0 w-full flex-col">
    <header class="rounded-2xl border border-white/15 bg-slate-900/60 p-3 text-slate-100 backdrop-blur">
      <div class="flex items-center justify-between gap-2 text-sm font-semibold">
        <p>Tempo: {{ timeLabel }}</p>
        <p>Tap: {{ tapCount }}</p>
      </div>
      <p class="mt-1 text-xs uppercase tracking-[0.2em] text-slate-300">{{ statusLabel }}</p>
    </header>

    <main
      ref="gameAreaRef"
      class="game-area mt-4 flex-1 rounded-2xl border border-white/20 bg-slate-900/70"
      @touchmove.prevent
    >
      <button
        v-if="status === 'playing'"
        ref="ballRef"
        type="button"
        class="ball"
        :style="ballStyle"
        aria-label="Tappa la palla"
        @click="onTap"
      >
        <span aria-hidden="true">🏐</span>
      </button>

      <div v-else class="overlay">
        <p class="text-xs uppercase tracking-[0.2em] text-slate-300">{{ statusLabel }}</p>
        <h3 class="mt-2 text-2xl font-black text-white">
          <template v-if="status === 'ready'">Tap Challenge</template>
          <template v-else>Hai guadagnato {{ earnedCoins }} monete</template>
        </h3>
        <p v-if="status === 'ready'" class="mt-2 text-sm text-slate-200">Tappa la palla più volte possibile in 10 secondi.</p>
        <p v-else class="mt-2 text-sm text-slate-200">Totale tap validi: {{ tapCount }}</p>
        <div v-if="status === 'finished'" class="sparkles" aria-hidden="true" />
      </div>
    </main>

    <p v-if="errorMessage" class="mt-3 rounded-xl border border-red-300/40 bg-red-500/10 px-3 py-2 text-sm text-red-200">
      {{ errorMessage }}
    </p>

    <footer class="mt-4 grid grid-cols-1 gap-2 sm:grid-cols-2">
      <button type="button" class="cta-primary" :disabled="isPrimaryDisabled" @click="onPrimaryAction">{{ primaryLabel }}</button>
      <button type="button" class="cta-secondary" @click="emit('exit')">Esci</button>
    </footer>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, ref } from 'vue';
import { awardTapChallengeCoins } from '../../services/coins';

const props = defineProps({
  eventId: {
    type: Number,
    default: 0,
  },
  cooldownSeconds: {
    type: Number,
    default: 60,
  },
});

const emit = defineEmits(['claim', 'exit']);

const ROUND_DURATION_MS = 10_000;
const TICK_MS = 100;
const MIN_DISTANCE_PX = 30;
const BASE_MOVE_INTERVAL_MS = 850;
const MIN_MOVE_INTERVAL_MS = 220;
const SPEED_GAIN_PER_TAP = 0.92;

const status = ref('ready');
const timeLeftMs = ref(ROUND_DURATION_MS);
const tapCount = ref(0);
const cooldownUntil = ref(0);
const errorMessage = ref('');
const isSubmitting = ref(false);
const claimRequested = ref(false);
const nowTs = ref(Date.now());

const gameAreaRef = ref(null);
const ballRef = ref(null);
const ballX = ref(0);
const ballY = ref(0);
const lastBallX = ref(0);
const lastBallY = ref(0);
const moveIntervalMs = ref(BASE_MOVE_INTERVAL_MS);

let timerId;
let moveTimerId;
let gameEndsAt = 0;
let cooldownTickId;

const earnedCoins = computed(() => tapCount.value);
const isCooldownActive = computed(() => cooldownUntil.value > nowTs.value);
const cooldownRemainingMs = computed(() => Math.max(0, cooldownUntil.value - nowTs.value));

const statusLabel = computed(() => {
  if (status.value === 'playing') {
    return 'PLAYING';
  }
  if (status.value === 'finished') {
    return 'FINISHED';
  }
  return 'READY';
});

const timeLabel = computed(() => `${(Math.max(0, timeLeftMs.value) / 1000).toFixed(1)}s`);

const primaryLabel = computed(() => {
  if (status.value === 'playing') {
    return 'In corso…';
  }
  if (isSubmitting.value || claimRequested.value) {
    return 'Accredito…';
  }
  if (status.value === 'finished' && errorMessage.value) {
    return 'Riprova accredito';
  }
  if (status.value === 'finished') {
    return 'Riscatta monete';
  }
  if (isCooldownActive.value) {
    return `In cooldown ${formatCooldown(cooldownRemainingMs.value)}`;
  }
  return 'Inizia';
});

const isPrimaryDisabled = computed(() => {
  if (status.value === 'playing' || isSubmitting.value || claimRequested.value) {
    return true;
  }

  if (status.value === 'finished') {
    return false;
  }

  return isCooldownActive.value;
});

const ballStyle = computed(() => ({
  transform: `translate3d(${ballX.value}px, ${ballY.value}px, 0)`,
}));

onBeforeUnmount(() => {
  stopTimer();
  stopBallMovement();
  if (cooldownTickId && typeof window !== 'undefined') {
    window.clearInterval(cooldownTickId);
    cooldownTickId = undefined;
  }
});

if (typeof window !== 'undefined') {
  cooldownTickId = window.setInterval(() => {
    nowTs.value = Date.now();
  }, 250);
}

function stopTimer() {
  if (!timerId || typeof window === 'undefined') {
    return;
  }
  window.clearInterval(timerId);
  timerId = undefined;
}

function stopBallMovement() {
  if (!moveTimerId || typeof window === 'undefined') {
    return;
  }
  window.clearTimeout(moveTimerId);
  moveTimerId = undefined;
}

function scheduleBallMovement() {
  stopBallMovement();

  if (typeof window === 'undefined' || status.value !== 'playing') {
    return;
  }

  moveTimerId = window.setTimeout(() => {
    if (status.value !== 'playing') {
      return;
    }
    repositionBall();
    scheduleBallMovement();
  }, moveIntervalMs.value);
}

function onPrimaryAction() {
  if (status.value === 'playing' || isSubmitting.value || claimRequested.value) {
    return;
  }

  if (status.value === 'finished') {
    claimReward();
    return;
  }

  if (isCooldownActive.value) {
    return;
  }
  startGame();
}

async function startGame() {
  stopTimer();
  stopBallMovement();
  errorMessage.value = '';
  claimRequested.value = false;
  status.value = 'playing';
  tapCount.value = 0;
  timeLeftMs.value = ROUND_DURATION_MS;
  moveIntervalMs.value = BASE_MOVE_INTERVAL_MS;
  gameEndsAt = Date.now() + ROUND_DURATION_MS;

  await nextTick();
  repositionBall(true);
  scheduleBallMovement();

  if (typeof window === 'undefined') {
    return;
  }

  timerId = window.setInterval(() => {
    const remaining = gameEndsAt - Date.now();
    timeLeftMs.value = Math.max(0, remaining);
    if (remaining <= 0) {
      finishGame();
    }
  }, TICK_MS);
}

function repositionBall(force = false) {
  const area = gameAreaRef.value;
  const ball = ballRef.value;
  if (!area || !ball) {
    return;
  }

  const areaRect = area.getBoundingClientRect();
  const ballRect = ball.getBoundingClientRect();
  const maxX = Math.max(0, areaRect.width - ballRect.width);
  const maxY = Math.max(0, areaRect.height - ballRect.height);

  let nextX = 0;
  let nextY = 0;
  let attempts = 0;

  do {
    nextX = clamp(Math.random() * maxX, 0, maxX);
    nextY = clamp(Math.random() * maxY, 0, maxY);
    attempts += 1;
  } while (!force && attempts < 8 && distance(nextX, nextY, lastBallX.value, lastBallY.value) < MIN_DISTANCE_PX);

  ballX.value = nextX;
  ballY.value = nextY;
  lastBallX.value = nextX;
  lastBallY.value = nextY;
}

function onTap() {
  if (status.value !== 'playing') {
    return;
  }

  tapCount.value += 1;
  moveIntervalMs.value = Math.max(MIN_MOVE_INTERVAL_MS, moveIntervalMs.value * SPEED_GAIN_PER_TAP);
  if (typeof navigator !== 'undefined') {
    navigator.vibrate?.(10);
  }
  repositionBall();
  scheduleBallMovement();
}

function finishGame() {
  if (status.value !== 'playing') {
    return;
  }

  stopTimer();
  stopBallMovement();
  status.value = 'finished';
  timeLeftMs.value = 0;

  const nextCooldown = Date.now() + Math.max(0, props.cooldownSeconds) * 1000;
  cooldownUntil.value = nextCooldown;
  if (typeof window !== 'undefined') {
    window.localStorage.setItem('tap_challenge_cooldown_until', String(nextCooldown));
  }

}

async function claimReward() {
  if (isSubmitting.value || claimRequested.value || status.value !== 'finished') {
    return;
  }

  claimRequested.value = true;

  if (earnedCoins.value <= 0) {
    claimRequested.value = false;
    emit('claim', { coins: 0 });
    return;
  }

  isSubmitting.value = true;
  errorMessage.value = '';

  const requestId = typeof crypto !== 'undefined' && crypto.randomUUID ? crypto.randomUUID() : `tap_${Date.now()}`;
  const result = await awardTapChallengeCoins({
    amount: earnedCoins.value,
    requestId,
    eventContextId: props.eventId,
    meta: {
      taps: tapCount.value,
      durationMs: ROUND_DURATION_MS,
    },
  });

  isSubmitting.value = false;

  if (!result.ok) {
    errorMessage.value = 'Errore accredito, riprova.';
    claimRequested.value = false;
    return;
  }

  emit('claim', { coins: earnedCoins.value, source: 'tap_challenge', meta: { taps: tapCount.value } });
}

function clamp(value, min, max) {
  return Math.min(max, Math.max(min, value));
}

function distance(ax, ay, bx, by) {
  return Math.hypot(ax - bx, ay - by);
}

function formatCooldown(ms) {
  const totalSeconds = Math.max(0, Math.ceil(ms / 1000));
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`;
}

if (typeof window !== 'undefined') {
  const stored = Number.parseInt(window.localStorage.getItem('tap_challenge_cooldown_until') || '', 10);
  if (Number.isFinite(stored) && stored > Date.now()) {
    cooldownUntil.value = stored;
  }
}
</script>

<style scoped>
.tap-challenge {
  user-select: none;
}

.game-area {
  min-height: 280px;
  position: relative;
  overflow: hidden;
  touch-action: manipulation;
}

.ball {
  position: absolute;
  left: 0;
  top: 0;
  width: 68px;
  height: 68px;
  border-radius: 9999px;
  border: 2px solid rgba(255, 255, 255, 0.6);
  background: radial-gradient(circle at 30% 25%, #fef3c7, #f59e0b 62%, #b45309);
  color: #111827;
  font-size: 33px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 12px 26px rgba(2, 6, 23, 0.4);
  touch-action: manipulation;
  will-change: transform;
}

.overlay {
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 1rem;
  position: relative;
}


.sparkles,
.sparkles::before,
.sparkles::after {
  content: '';
  position: absolute;
  width: 14px;
  height: 14px;
  border-radius: 9999px;
  background: rgba(250, 204, 21, 0.8);
  filter: blur(0.2px);
}

.sparkles {
  top: 18%;
  left: 26%;
  animation: twinkle 1.2s infinite;
}

.sparkles::before {
  top: 170%;
  left: 240%;
  animation: twinkle 1s infinite 0.2s;
}

.sparkles::after {
  top: 110%;
  left: 480%;
  animation: twinkle 1.4s infinite 0.1s;
}

@keyframes twinkle {
  0%,
  100% {
    opacity: 0.4;
    transform: scale(0.9);
  }
  50% {
    opacity: 1;
    transform: scale(1.2);
  }
}

.cta-primary,
.cta-secondary {
  border-radius: 9999px;
  padding: 0.75rem 1rem;
  font-weight: 800;
}

.cta-primary {
  background: #fbbf24;
  color: #0f172a;
}

.cta-primary:disabled {
  opacity: 0.65;
}

.cta-secondary {
  border: 1px solid rgba(255, 255, 255, 0.2);
  color: #fff;
  background: rgba(255, 255, 255, 0.08);
}
</style>
