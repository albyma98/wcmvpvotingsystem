<template>
  <div class="reaction-game h-full min-h-0 w-full overflow-hidden">
    <section class="reaction-shell flex h-full min-h-0 flex-col gap-4">
      <header class="hud rounded-2xl border border-white/15 bg-slate-900/60 p-3 backdrop-blur">
        <div class="flex flex-wrap items-center justify-between gap-2 text-sm text-slate-200">
          <p class="font-semibold">Round {{ currentRound }}/{{ TOTAL_ROUNDS }}</p>
          <p class="font-semibold">Best: {{ bestDisplay }}</p>
          <p class="font-semibold text-amber-200">Reward: +{{ previewCoins }} 🪙</p>
        </div>
      </header>

      <main class="relative flex min-h-0 flex-1">
        <button
          type="button"
          class="play-area group"
          :class="playAreaClass"
          :aria-label="playAreaAriaLabel"
          @click="onPlayTap"
        >
          <div class="pointer-events-none px-4 text-center">
            <p v-if="phase === 'countdown'" class="countdown-text text-7xl font-black text-white">{{ countdownValue }}</p>
            <template v-else>
              <p class="text-4xl font-black text-white md:text-6xl">{{ mainMessage }}</p>
              <p class="mt-2 text-sm font-semibold uppercase tracking-[0.24em] text-white/75 md:text-base">{{ subMessage }}</p>
            </template>
            <p v-if="phase === 'result'" class="mt-5 text-xl font-bold text-emerald-200">{{ resultMessage }}</p>
          </div>

          <span v-if="showCoinBurst" class="coin-burst">+{{ roundCoins }} 🪙</span>

          <div v-if="phase === 'wait'" class="pulse-dot" aria-hidden="true" />
          <div v-if="phase === 'summary'" class="sparkles" aria-hidden="true" />
        </button>
      </main>

      <footer class="controls grid grid-cols-1 gap-2 sm:grid-cols-2">
        <button
          type="button"
          class="cta-primary"
          :aria-label="primaryLabel"
          :disabled="isPrimaryDisabled"
          @click="onPrimaryAction"
        >
          {{ primaryLabel }}
        </button>
        <button
          type="button"
          class="cta-secondary"
          aria-label="Esci dal gioco"
          @click="emit('exit')"
        >
          Esci
        </button>
      </footer>
    </section>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, ref } from 'vue';

const emit = defineEmits(['claim', 'exit']);

const TOTAL_ROUNDS = 3;
const FALSTART_PENALTY_MS = 300;

const phase = ref('intro');
const currentRound = ref(1);
const countdownValue = ref(3);
const falseStarts = ref(0);
const roundResults = ref([]);
const goStartTime = ref(0);
const roundCoins = ref(0);
const showCoinBurst = ref(false);
const tapLocked = ref(false);
const claimRequested = ref(false);

let countdownInterval;
let waitTimeout;
let coinBurstTimeout;

const bestMs = computed(() => {
  if (!roundResults.value.length) {
    return null;
  }

  return Math.min(...roundResults.value.map((result) => result.totalMs));
});

const avgMs = computed(() => {
  if (!roundResults.value.length) {
    return null;
  }

  const total = roundResults.value.reduce((accumulator, result) => accumulator + result.totalMs, 0);
  return Math.round(total / roundResults.value.length);
});

const coins = computed(() => {
  if (roundResults.value.length < TOTAL_ROUNDS || avgMs.value == null) {
    return 0;
  }

  let amount = 5;

  if (avgMs.value <= 230) {
    amount = 15;
  } else if (avgMs.value <= 300) {
    amount = 10;
  } else if (avgMs.value <= 380) {
    amount = 7;
  }

  if (falseStarts.value >= 2) {
    return Math.min(amount, 5);
  }

  return amount;
});

const previewCoins = computed(() => {
  if (phase.value === 'summary') {
    return coins.value;
  }

  return Math.max(coins.value, 5);
});

const bestDisplay = computed(() => (bestMs.value == null ? '—' : `${bestMs.value} ms`));

const mainMessage = computed(() => {
  switch (phase.value) {
    case 'intro':
      return 'Reaction Test';
    case 'wait':
      return 'Aspetta…';
    case 'go':
      return 'TAP!';
    case 'result':
      return 'Round completato';
    case 'summary':
      return 'Risultato finale';
    default:
      return 'Ready';
  }
});

const subMessage = computed(() => {
  switch (phase.value) {
    case 'intro':
      return '3 round rapidi. Tappa appena diventa verde.';
    case 'wait':
      return 'Non toccare ora o prendi penalità';
    case 'go':
      return 'Tap anywhere!';
    case 'result':
      return 'Feedback istantaneo';
    case 'summary':
      return `Avg ${avgMs.value ?? '—'} ms • Best ${bestMs.value ?? '—'} ms`;
    default:
      return '';
  }
});

const resultMessage = computed(() => {
  const latest = roundResults.value[roundResults.value.length - 1];
  if (!latest || phase.value !== 'result') {
    return '';
  }

  if (latest.falstart) {
    return `Falstart! Penalità +${FALSTART_PENALTY_MS} ms`;
  }

  return `${latest.reactionMs} ms`;
});

const playAreaClass = computed(() => ({
  wait: phase.value === 'wait',
  go: phase.value === 'go',
  result: phase.value === 'result',
  summary: phase.value === 'summary',
  shake: phase.value === 'result' && roundResults.value[roundResults.value.length - 1]?.falstart,
  flash: phase.value === 'go',
}));

const playAreaAriaLabel = computed(() => {
  if (phase.value === 'go') {
    return 'Tocca ora per registrare il tempo di reazione';
  }

  return 'Area di gioco reaction test';
});

const primaryLabel = computed(() => {
  if (phase.value === 'intro') {
    return 'Inizia';
  }

  if (phase.value === 'result' && currentRound.value < TOTAL_ROUNDS) {
    return 'Prossimo round';
  }

  if (phase.value === 'result' && currentRound.value >= TOTAL_ROUNDS) {
    return 'Vai al riepilogo';
  }

  if (phase.value === 'summary') {
    return claimRequested.value ? 'Riscatto in corso…' : 'Riscatta monete';
  }

  return 'Attendi…';
});

const isPrimaryDisabled = computed(() => phase.value === 'summary' && claimRequested.value);

function clearTimers() {
  if (typeof window === 'undefined') {
    return;
  }

  if (countdownInterval) {
    window.clearInterval(countdownInterval);
    countdownInterval = undefined;
  }

  if (waitTimeout) {
    window.clearTimeout(waitTimeout);
    waitTimeout = undefined;
  }

  if (coinBurstTimeout) {
    window.clearTimeout(coinBurstTimeout);
    coinBurstTimeout = undefined;
  }
}

function resetGame() {
  clearTimers();
  phase.value = 'intro';
  currentRound.value = 1;
  countdownValue.value = 3;
  falseStarts.value = 0;
  roundResults.value = [];
  tapLocked.value = false;
  showCoinBurst.value = false;
  claimRequested.value = false;
}

function startCountdown() {
  clearTimers();
  phase.value = 'countdown';
  countdownValue.value = 3;
  tapLocked.value = false;

  countdownInterval = window.setInterval(() => {
    countdownValue.value -= 1;

    if (countdownValue.value <= 0) {
      window.clearInterval(countdownInterval);
      countdownInterval = undefined;
      startWait();
    }
  }, 450);
}

function startWait() {
  phase.value = 'wait';
  const delay = Math.floor(Math.random() * 1601) + 600;

  waitTimeout = window.setTimeout(() => {
    phase.value = 'go';
    goStartTime.value = performance.now();
    tapLocked.value = false;
  }, delay);
}

function finalizeRound(result) {
  roundResults.value = [...roundResults.value, result];
  phase.value = 'result';
  tapLocked.value = true;
}

function onPlayTap() {
  if (phase.value === 'go') {
    if (tapLocked.value) {
      return;
    }

    tapLocked.value = true;
    const reactionMs = Math.max(1, Math.round(performance.now() - goStartTime.value));
    clearTimers();

    finalizeRound({
      round: currentRound.value,
      reactionMs,
      totalMs: reactionMs,
      falstart: false,
    });

    showRoundCoins();
    return;
  }

  if (phase.value === 'wait' || phase.value === 'countdown') {
    clearTimers();
    falseStarts.value += 1;

    finalizeRound({
      round: currentRound.value,
      reactionMs: null,
      totalMs: FALSTART_PENALTY_MS,
      falstart: true,
    });
  }
}

function showRoundCoins() {
  roundCoins.value = Math.max(1, Math.round(coins.value / TOTAL_ROUNDS) || 2);
  showCoinBurst.value = true;

  coinBurstTimeout = window.setTimeout(() => {
    showCoinBurst.value = false;
  }, 900);
}

function onPrimaryAction() {
  if (phase.value === 'intro') {
    resetGame();
    startCountdown();
    return;
  }

  if (phase.value === 'result') {
    if (currentRound.value < TOTAL_ROUNDS) {
      currentRound.value += 1;
      startCountdown();
      return;
    }

    phase.value = 'summary';
    return;
  }

  if (phase.value === 'summary') {
    if (claimRequested.value) {
      return;
    }

    claimRequested.value = true;
    emit('claim', {
      gameId: 'reaction',
      coins: coins.value,
      avgMs: avgMs.value,
      bestMs: bestMs.value,
    });
  }
}

onBeforeUnmount(() => {
  clearTimers();
});
</script>

<style scoped>
.reaction-game {
  color: #fff;
}

.play-area {
  position: relative;
  display: flex;
  width: 100%;
  min-height: 0;
  flex: 1;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border: 1px solid rgba(255, 255, 255, 0.18);
  border-radius: 1.25rem;
  background: radial-gradient(circle at 20% 15%, rgba(147, 51, 234, 0.25), rgba(15, 23, 42, 0.95));
  transition: transform 0.15s ease, background 0.2s ease;
}

.play-area.wait {
  background: radial-gradient(circle at 30% 10%, rgba(245, 158, 11, 0.25), rgba(51, 24, 15, 0.95));
}

.play-area.go {
  background: radial-gradient(circle at 35% 0%, rgba(74, 222, 128, 0.5), rgba(6, 78, 59, 0.95));
}

.play-area.result {
  background: radial-gradient(circle at 35% 0%, rgba(56, 189, 248, 0.25), rgba(15, 23, 42, 0.95));
}

.play-area.summary {
  background: radial-gradient(circle at 35% 0%, rgba(45, 212, 191, 0.35), rgba(15, 23, 42, 0.95));
}

.countdown-text {
  text-shadow: 0 0 24px rgba(148, 163, 184, 0.35);
}

.pulse-dot {
  position: absolute;
  bottom: 12%;
  width: 18px;
  height: 18px;
  border-radius: 9999px;
  background: #fde68a;
  box-shadow: 0 0 0 rgba(253, 230, 138, 0.7);
  animation: pulse 0.9s infinite;
}

.coin-burst {
  position: absolute;
  top: 20%;
  right: 12%;
  font-weight: 800;
  color: #fef08a;
  animation: float-up 0.9s ease forwards;
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
  top: 15%;
  left: 24%;
  animation: twinkle 1.2s infinite;
}

.sparkles::before {
  top: 180%;
  left: 260%;
  animation: twinkle 1s infinite 0.2s;
}

.sparkles::after {
  top: 110%;
  left: 500%;
  animation: twinkle 1.4s infinite 0.1s;
}

.cta-primary,
.cta-secondary {
  height: 52px;
  border-radius: 0.95rem;
  font-weight: 700;
}

.cta-primary {
  border: 1px solid rgba(74, 222, 128, 0.35);
  background: linear-gradient(120deg, rgba(16, 185, 129, 0.45), rgba(5, 150, 105, 0.9));
  color: #ecfdf5;
}

.cta-primary:disabled {
  opacity: 0.7;
}

.cta-secondary {
  border: 1px solid rgba(255, 255, 255, 0.22);
  background: rgba(15, 23, 42, 0.65);
  color: #e2e8f0;
}

.flash {
  animation: flash 0.22s ease;
}

.shake {
  animation: shake 0.22s linear;
}

@keyframes pulse {
  0% {
    transform: scale(0.8);
    box-shadow: 0 0 0 0 rgba(253, 230, 138, 0.65);
  }
  70% {
    transform: scale(1.2);
    box-shadow: 0 0 0 14px rgba(253, 230, 138, 0);
  }
  100% {
    transform: scale(0.8);
    box-shadow: 0 0 0 0 rgba(253, 230, 138, 0);
  }
}

@keyframes flash {
  0% { transform: scale(1); }
  50% { transform: scale(1.01); }
  100% { transform: scale(1); }
}

@keyframes shake {
  0% { transform: translateX(0); }
  25% { transform: translateX(-6px); }
  50% { transform: translateX(6px); }
  75% { transform: translateX(-4px); }
  100% { transform: translateX(0); }
}

@keyframes float-up {
  0% { opacity: 0; transform: translateY(8px); }
  20% { opacity: 1; }
  100% { opacity: 0; transform: translateY(-28px); }
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
</style>
