<template>
  <div class="memory-flash relative flex h-full min-h-0 w-full flex-col overflow-hidden rounded-2xl border border-white/10 bg-slate-950/70 p-3 text-white md:p-4">
    <header class="rounded-2xl border border-white/15 bg-slate-900/70 p-3 backdrop-blur">
      <div class="flex items-center justify-between gap-3">
        <div class="rounded-full border border-amber-300/45 bg-amber-300/15 px-3 py-1 text-sm font-black text-amber-200">
          🪙 {{ displayedCoins }}
        </div>
        <div class="flex-1">
          <div class="flex items-center justify-between text-xs font-semibold uppercase tracking-[0.16em] text-slate-300">
            <span>{{ statusLabel }}</span>
            <span>{{ timeLabel }}</span>
          </div>
          <div class="mt-1 h-2 overflow-hidden rounded-full bg-white/10">
            <div class="h-full rounded-full bg-gradient-to-r from-emerald-300 via-amber-300 to-rose-400 transition-[width] duration-100" :style="{ width: `${timerProgress}%` }" />
          </div>
        </div>
        <button type="button" class="close-btn" aria-label="Chiudi gioco" @click="emit('exit')">×</button>
      </div>
      <p class="mt-2 text-center text-xs text-slate-300">8 monete al primo completamento della partita</p>
    </header>

    <main class="mt-3 grid flex-1 grid-cols-2 gap-2 overflow-hidden">
      <button
        v-for="card in cards"
        :key="card.id"
        type="button"
        class="memory-card"
        :class="{
          'is-flipped': card.isFlipped || card.isMatched,
          'is-matched': card.isMatched,
          'is-disabled': boardLocked || state !== 'playing' || card.isMatched,
          'is-shaking': shakeCardIds.includes(card.id),
        }"
        :disabled="boardLocked || state !== 'playing' || card.isFlipped || card.isMatched"
        @click="onCardTap(card.id)"
      >
        <span class="sr-only">Carta memory</span>
        <span class="memory-card__inner">
          <span class="memory-card__face memory-card__face--back">?</span>
          <span class="memory-card__face memory-card__face--front">{{ card.emoji }}</span>
        </span>
      </button>
    </main>

    <Transition name="overlay-fade">
      <div v-if="state === 'win' || state === 'lose'" class="absolute inset-0 z-20 flex items-center justify-center bg-slate-950/90 p-4">
        <div class="w-full max-w-sm rounded-2xl border border-white/15 bg-slate-900/90 p-5 text-center shadow-2xl">
          <p class="text-xs font-bold uppercase tracking-[0.2em] text-slate-300">{{ state === 'win' ? 'Vittoria' : 'Tempo scaduto' }}</p>
          <h3 class="mt-2 text-2xl font-black text-white">
            {{ state === 'win' ? 'Memory completato!' : 'Ci sei quasi!' }}
          </h3>
          <p v-if="state === 'win'" class="mt-2 text-sm font-semibold text-emerald-300">{{ alreadyRewarded ? 'Bonus della partita già ottenuto. Puoi continuare a giocare.' : '8 monete al primo completamento della partita.' }}</p>
          <p v-else class="mt-2 text-sm text-slate-200">Mostriamo le coppie mancanti per aiutarti al prossimo tentativo.</p>

          <div v-if="state === 'win'" class="mt-4 grid grid-cols-1 gap-2">
            <button type="button" class="cta-primary" @click="startRound">Gioca ancora gratis</button>
          </div>

          <div v-else class="mt-4 grid grid-cols-1 gap-2 sm:grid-cols-2">
            <button type="button" class="cta-primary" @click="retryNow">Riprova gratis</button>
            <button type="button" class="cta-secondary" @click="emit('exit')">Esci</button>
          </div>
        </div>
      </div>
    </Transition>

  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';

const emit = defineEmits(['claim', 'exit']);
const props = defineProps({
  walletCoins: {
    type: Number,
    default: 0,
  },
  alreadyRewarded: { type: Boolean, default: false },
});

const EMOJIS = ['🏐', '⚡', '🔥', '💎', '👑'];
const TOTAL_TIME_MS = 20_000;
const PREVIEW_MS = 2_000;
const MISMATCH_FLIP_BACK_MS = 600;
const WIN_REWARD = 8;

const state = ref('preview');
const cards = ref([]);
const selectedIds = ref([]);
const boardLocked = ref(true);
const shakeCardIds = ref([]);
const timeLeftMs = ref(TOTAL_TIME_MS);
const displayedCoins = computed(() => Number.isFinite(props.walletCoins) ? Math.max(0, Math.floor(props.walletCoins)) : 0);

let countdownId;
let previewTimeoutId;
let mismatchTimeoutId;
let hapticCooldown = false;
let rewardRequested = false;

const timeLabel = computed(() => `${Math.ceil(Math.max(0, timeLeftMs.value) / 1000)}s`);
const timerProgress = computed(() => (Math.max(0, timeLeftMs.value) / TOTAL_TIME_MS) * 100);
const statusLabel = computed(() => {
  if (state.value === 'preview') return 'Preview';
  if (state.value === 'playing') return 'Playing';
  if (state.value === 'win') return 'Win';
  return 'Lose';
});

onMounted(() => {
  startRound();
});

onBeforeUnmount(() => {
  clearAllTimers();
});

function clearAllTimers() {
  if (typeof window === 'undefined') {
    return;
  }
  window.clearTimeout(previewTimeoutId);
  window.clearTimeout(mismatchTimeoutId);
  window.clearInterval(countdownId);
}

function buildDeck() {
  const duplicated = [...EMOJIS, ...EMOJIS];
  return duplicated
    .sort(() => Math.random() - 0.5)
    .map((emoji, index) => ({
      id: index + 1,
      emoji,
      isFlipped: true,
      isMatched: false,
    }));
}

function startRound() {
  clearAllTimers();
  state.value = 'preview';
  boardLocked.value = true;
  selectedIds.value = [];
  shakeCardIds.value = [];
  cards.value = buildDeck();
  timeLeftMs.value = TOTAL_TIME_MS;

  if (typeof window === 'undefined') {
    return;
  }

  previewTimeoutId = window.setTimeout(() => {
    cards.value = cards.value.map((card) => ({ ...card, isFlipped: false }));
    state.value = 'playing';
    boardLocked.value = false;
    startCountdown();
  }, PREVIEW_MS);
}

function startCountdown() {
  if (typeof window === 'undefined') {
    return;
  }
  const startsAt = Date.now();
  countdownId = window.setInterval(() => {
    const elapsed = Date.now() - startsAt;
    timeLeftMs.value = Math.max(0, TOTAL_TIME_MS - elapsed);
    if (timeLeftMs.value <= 0) {
      onLose();
    }
  }, 80);
}

function onCardTap(cardId) {
  if (boardLocked.value || state.value !== 'playing') {
    return;
  }

  const card = cards.value.find((entry) => entry.id === cardId);
  if (!card || card.isFlipped || card.isMatched || selectedIds.value.length >= 2) {
    return;
  }

  vibrate(12);
  card.isFlipped = true;
  selectedIds.value.push(cardId);

  if (selectedIds.value.length < 2) {
    return;
  }

  boardLocked.value = true;
  const [firstId, secondId] = selectedIds.value;
  const first = cards.value.find((entry) => entry.id === firstId);
  const second = cards.value.find((entry) => entry.id === secondId);

  if (!first || !second) {
    resetTurn();
    return;
  }

  if (first.emoji === second.emoji) {
    first.isMatched = true;
    second.isMatched = true;
    selectedIds.value = [];
    boardLocked.value = false;
    vibrate(24);

    if (cards.value.every((entry) => entry.isMatched)) {
      onWin();
    }
    return;
  }

  shakeCardIds.value = [first.id, second.id];
  mismatchTimeoutId = window.setTimeout(() => {
    first.isFlipped = false;
    second.isFlipped = false;
    resetTurn();
  }, MISMATCH_FLIP_BACK_MS);
}

function resetTurn() {
  selectedIds.value = [];
  shakeCardIds.value = [];
  boardLocked.value = false;
}

function onWin() {
  if (state.value !== 'playing') return;
  state.value = 'win';
  boardLocked.value = true;
  window.clearInterval(countdownId);
  if (!rewardRequested && !props.alreadyRewarded) {
    rewardRequested = true;
    emit('claim', { coins: WIN_REWARD, gameId: 'memory-flash' });
  }
}

function onLose() {
  state.value = 'lose';
  boardLocked.value = true;
  window.clearInterval(countdownId);
  cards.value = cards.value.map((card) => ({
    ...card,
    isFlipped: true,
  }));
}

function retryNow() {
  startRound();
}

function vibrate(duration) {
  if (hapticCooldown || typeof navigator === 'undefined' || typeof navigator.vibrate !== 'function') {
    return;
  }
  hapticCooldown = true;
  navigator.vibrate(duration);
  window.setTimeout(() => {
    hapticCooldown = false;
  }, 70);
}
</script>

<style scoped>
.memory-card {
  perspective: 900px;
}

.memory-card__inner {
  position: relative;
  display: block;
  height: 100%;
  width: 100%;
  min-height: 0;
  transform-style: preserve-3d;
  transition: transform 0.35s ease;
}

.memory-card.is-flipped .memory-card__inner {
  transform: rotateY(180deg);
}

.memory-card__face {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 0.95rem;
  backface-visibility: hidden;
  font-size: clamp(1.5rem, 4.8vw, 2.2rem);
  font-weight: 900;
}

.memory-card__face--back {
  border: 1px solid rgba(255, 255, 255, 0.26);
  background: linear-gradient(145deg, rgba(59, 130, 246, 0.58), rgba(168, 85, 247, 0.54));
}

.memory-card__face--front {
  transform: rotateY(180deg);
  border: 1px solid rgba(167, 243, 208, 0.34);
  background: rgba(15, 23, 42, 0.9);
}

.memory-card.is-matched .memory-card__face--front {
  animation: matchGlow 0.9s ease;
  box-shadow: 0 0 18px rgba(52, 211, 153, 0.65);
}

.memory-card.is-shaking {
  animation: cardShake 0.36s ease;
}

.memory-card,
.memory-card__inner {
  height: 100%;
}

.close-btn {
  height: 2.2rem;
  width: 2.2rem;
  border-radius: 999px;
  border: 1px solid rgba(255, 255, 255, 0.2);
  background: rgba(255, 255, 255, 0.07);
  font-size: 1.4rem;
  line-height: 1;
}

.cta-primary,
.cta-secondary {
  border-radius: 999px;
  padding: 0.65rem 1rem;
  font-size: 0.9rem;
  font-weight: 800;
}

.cta-primary {
  background: linear-gradient(135deg, #34d399, #22d3ee);
  color: #082f49;
}

.cta-primary:disabled {
  opacity: 0.45;
}

.cta-secondary {
  border: 1px solid rgba(255, 255, 255, 0.3);
  background: rgba(15, 23, 42, 0.65);
  color: white;
}

.overlay-fade-enter-active,
.overlay-fade-leave-active {
  transition: opacity 0.2s ease;
}

.overlay-fade-enter-from,
.overlay-fade-leave-to {
  opacity: 0;
}

@keyframes cardShake {
  0%,
  100% { transform: translateX(0); }
  35% { transform: translateX(-4px); }
  70% { transform: translateX(4px); }
}

@keyframes matchGlow {
  0% { box-shadow: 0 0 0 rgba(52, 211, 153, 0); }
  40% { box-shadow: 0 0 20px rgba(52, 211, 153, 0.85); }
  100% { box-shadow: 0 0 6px rgba(52, 211, 153, 0.35); }
}

</style>
