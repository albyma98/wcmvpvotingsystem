<template>
  <Teleport to="body">
    <Transition name="earn-modal-fade">
      <div
        v-if="modelValue"
        class="fixed inset-0 z-[120] flex"
        role="dialog"
        aria-modal="true"
        aria-label="Guadagna Monete"
        @click.self="closeModal"
      >
        <div class="absolute inset-0 bg-slate-950/90 backdrop-blur-sm" aria-hidden="true" />

        <Transition name="earn-modal-slide">
          <div class="relative flex h-full w-full flex-col overflow-hidden">
            <header class="sticky top-0 z-10 border-b border-white/10 bg-slate-950/85 px-4 py-4 backdrop-blur md:px-6">
              <div class="flex items-start justify-between gap-4">
                <div>
                  <h2 class="text-2xl font-black text-white md:text-3xl">{{ activeView === 'list' ? 'Guadagna Monete' : activeGame?.title }}</h2>
                  <p class="mt-1 text-sm text-slate-300 md:text-base">
                    {{ activeView === 'list' ? 'Scegli un’attività e accumula monete' : 'Completa il gioco per ottenere ricompense.' }}
                  </p>
                </div>
                <div class="flex items-center gap-2">
                  <button
                    v-if="activeView === 'game'"
                    type="button"
                    class="inline-flex h-10 items-center justify-center rounded-full border border-white/20 bg-white/5 px-4 text-sm font-semibold text-white transition hover:bg-white/15 disabled:opacity-60"
                    aria-label="Torna alla lista dei giochi"
                    :disabled="isClaiming"
                    @click="goBack"
                  >
                    ← Back
                  </button>
                  <button
                    type="button"
                    class="inline-flex h-10 w-10 items-center justify-center rounded-full border border-white/20 bg-white/5 text-2xl leading-none text-white transition hover:bg-white/15 disabled:opacity-60"
                    aria-label="Chiudi modale Guadagna Monete"
                    :disabled="isClaiming"
                    @click="closeModal"
                  >
                    ×
                  </button>
                </div>
              </div>
            </header>

            <div class="flex-1 px-4 pb-8 pt-5 md:px-6" :class="activeView === 'game' ? 'overflow-hidden' : 'overflow-y-auto'">
              <Transition name="slide" mode="out-in">
                <div v-if="activeView === 'list'" key="list" class="mx-auto grid max-w-6xl grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
                  <button
                    v-for="option in earnOptions"
                    :key="option.id"
                    type="button"
                    class="group rounded-2xl border border-white/15 bg-white/10 p-4 text-left shadow-[0_10px_28px_rgba(15,23,42,0.45)] backdrop-blur transition hover:-translate-y-0.5 hover:bg-white/15 disabled:cursor-not-allowed disabled:opacity-60"
                    :disabled="!option.isAvailable || cooldowns[option.id] > 0"
                    @click="handleOptionClick(option)"
                  >
                    <div class="flex items-start justify-between gap-2">
                      <span class="text-2xl" aria-hidden="true">{{ option.icon }}</span>
                    </div>
                    <h3 class="mt-3 text-lg font-extrabold text-white">{{ option.title }}</h3>
                    <p class="mt-1 text-sm text-slate-300">{{ option.description }}</p>
                    <p class="mt-4 text-xs font-semibold uppercase tracking-wide" :class="!option.isAvailable ? 'text-slate-400' : cooldowns[option.id] > 0 ? 'text-orange-300' : 'text-emerald-300'">
                      {{ !option.isAvailable ? 'Temporaneamente non disponibile' : cooldowns[option.id] > 0 ? `${option.id === 'tap' ? 'IN COOLDOWN' : 'In cooldown'} ${formatCooldown(cooldowns[option.id])}` : 'Disponibile' }}
                    </p>
                  </button>
                </div>

                <div v-else key="game" class="mx-auto flex h-full min-h-0 w-full max-w-6xl flex-col">
                  <div ref="gameStageRef" class="flex h-full flex-1 items-stretch rounded-2xl border border-white/10 bg-white/5 p-4 text-white md:p-6">
                    <ReactionTestGame
                      v-if="activeGame?.id === 'reaction'"
                      class="h-full w-full"
                      @claim="handleClaim"
                      @exit="goBack"
                    />
                    <QuickQuizGame
                      v-else-if="activeGame?.id === 'quiz'"
                      class="h-full w-full"
                      :event-id="eventId"
                      @claim="handleClaim"
                      @exit="goBack"
                    />
                    <TapChallenge
                      v-else-if="activeGame?.id === 'tap'"
                      class="h-full w-full"
                      :event-id="eventId"
                      :cooldown-seconds="activeGame?.cooldownSeconds || 60"
                      @claim="handleClaim"
                      @exit="goBack"
                    />
                    <MemoryFlashGame
                      v-else-if="activeGame?.id === 'memory-flash'"
                      class="h-full w-full"
                      :wallet-coins="walletCoins"
                      :free-retry="freeRetry"
                      @claim="handleClaim"
                      @spend="handleSpend"
                      @consume-free-retry="consumeFreeRetry"
                      @exit="goBack"
                    />
                    <div v-else class="flex w-full items-center justify-center rounded-xl border border-dashed border-white/20 bg-slate-900/40 p-6 text-center text-slate-200">
                      Game coming soon
                    </div>
                  </div>
                </div>
              </Transition>
            </div>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>

  <CoinCollectAnimation ref="coinAnimationRef" />
</template>

<script setup>
import { computed, onBeforeUnmount, ref, watch } from 'vue';
import CoinCollectAnimation from './CoinCollectAnimation.vue';
import ReactionTestGame from './ReactionTestGame.vue';
import QuickQuizGame from './QuickQuizGame.vue';
import TapChallenge from './minigames/TapChallenge.vue';
import MemoryFlashGame from './minigames/MemoryFlashGame.vue';
import { getEarnCooldownRemainingSeconds, startEarnCooldown } from '../utils/earnCooldown';

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false,
  },
  walletTargetEl: {
    type: Object,
    default: null,
  },
  eventId: {
    type: Number,
    default: 0,
  },
  walletCoins: {
    type: Number,
    default: 0,
  },
  freeRetry: {
    type: Number,
    default: 0,
  },
});

const emit = defineEmits(['update:modelValue', 'earned', 'coins-earned', 'consume-free-retry']);

const activeView = ref('list');
const activeGame = ref(null);
const isClaiming = ref(false);
const gameStageRef = ref(null);
const coinAnimationRef = ref(null);

const earnOptions = [
  { id: 'reaction', title: 'Reaction Test', description: 'Testa i riflessi e scala la classifica.', reward: 10, icon: '⚡', type: 'game', cooldownSeconds: 90, isAvailable: true },
  { id: 'quiz', title: 'Quiz Lampo', description: 'Rispondi veloce a domande a tema match.', reward: 15, icon: '🧠', type: 'game', cooldownSeconds: 120, isAvailable: false },
  { id: 'tap', title: 'Tap Challenge', description: 'Tappa più forte che puoi in 10 secondi.', reward: 8, icon: '👆', type: 'game', cooldownSeconds: 60, isAvailable: true },
  { id: 'memory-flash', title: 'Memory Flash', description: 'Memorizza le coppie e chiudi il board prima del tempo.', reward: 8, icon: '🧩', type: 'game', cooldownSeconds: 60, isAvailable: true },
];

const nowTick = ref(Date.now());
const cooldowns = computed(() =>
  earnOptions.reduce((accumulator, option) => {
    const defaultCooldown = getEarnCooldownRemainingSeconds(option.id, nowTick.value);
    if (option.id === 'tap') {
      accumulator[option.id] = Math.max(defaultCooldown, getTapChallengeCooldownSeconds(nowTick.value));
      return accumulator;
    }
    accumulator[option.id] = defaultCooldown;
    return accumulator;
  }, {}),
);

let intervalId;

watch(
  () => props.modelValue,
  (isOpen) => {
    if (typeof window === 'undefined' || typeof document === 'undefined') {
      return;
    }

    if (isOpen) {
      document.body.style.overflow = 'hidden';
      if (!intervalId) {
        intervalId = window.setInterval(() => {
          forceTick();
        }, 1000);
      }
      window.addEventListener('keydown', onKeydown);
      forceTick();
      return;
    }

    document.body.style.overflow = '';
    goBack();
    window.removeEventListener('keydown', onKeydown);
    if (intervalId) {
      window.clearInterval(intervalId);
      intervalId = undefined;
    }
  },
  { immediate: true },
);

onBeforeUnmount(() => {
  if (typeof document !== 'undefined') {
    document.body.style.overflow = '';
  }

  if (typeof window !== 'undefined') {
    window.removeEventListener('keydown', onKeydown);
    if (intervalId) {
      window.clearInterval(intervalId);
      intervalId = undefined;
    }
  }
});

function resolveWalletTarget() {
  if (props.walletTargetEl && props.walletTargetEl instanceof HTMLElement) {
    return props.walletTargetEl;
  }

  return document.getElementById('wallet-coin-target');
}

function forceTick() {
  nowTick.value = Date.now();
}

function closeModal() {
  if (isClaiming.value) {
    return;
  }

  goBack();
  emit('update:modelValue', false);
}

function onKeydown(event) {
  if (event.key === 'Escape') {
    closeModal();
  }
}

function formatCooldown(totalSeconds) {
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`;
}

function getTapChallengeCooldownSeconds(now = Date.now()) {
  if (typeof window === 'undefined') {
    return 0;
  }

  const cooldownUntil = Number.parseInt(window.localStorage.getItem('tap_challenge_cooldown_until') || '', 10);
  if (!Number.isFinite(cooldownUntil)) {
    return 0;
  }

  return Math.max(0, Math.ceil((cooldownUntil - now) / 1000));
}

function openGame(option) {
  activeGame.value = option;
  activeView.value = 'game';
}

function goBack() {
  if (isClaiming.value) {
    return;
  }

  activeView.value = 'list';
  activeGame.value = null;
}

async function handleClaim(payload) {
  if (isClaiming.value) {
    return;
  }

  isClaiming.value = true;
  const coins = Math.max(0, Number(payload?.coins) || 0);

  try {
    const toEl = resolveWalletTarget();
    const fromEl = gameStageRef.value;

    if (coinAnimationRef.value?.play && fromEl && toEl) {
      await coinAnimationRef.value.play({
        fromEl,
        toEl,
        count: 18,
        amount: coins,
      });
    }

    emit('earned', payload);
    emit('coins-earned', coins);
    activeView.value = 'list';
    activeGame.value = null;
  } finally {
    isClaiming.value = false;
  }
}

function handleSpend(payload) {
  const coins = Math.max(0, Number(payload?.coins) || 0);
  if (!coins) {
    return;
  }

  emit('coins-earned', -coins);
}

function consumeFreeRetry() {
  emit('consume-free-retry');
}

function handleOptionClick(option) {
  if (!option.isAvailable) {
    return;
  }

  if (cooldowns.value[option.id] > 0) {
    return;
  }

  startEarnCooldown(option.id, option.cooldownSeconds);
  forceTick();

  if (option.type === 'game') {
    openGame(option);
    return;
  }

  if (typeof window !== 'undefined') {
    window.alert(`${option.title}: azione disponibile a breve.`);
  }
}
</script>

<style scoped>
.earn-modal-fade-enter-active,
.earn-modal-fade-leave-active {
  transition: opacity 0.22s ease;
}

.earn-modal-fade-enter-from,
.earn-modal-fade-leave-to {
  opacity: 0;
}

.earn-modal-slide-enter-active,
.earn-modal-slide-leave-active {
  transition: transform 0.24s ease, opacity 0.24s ease;
}

.earn-modal-slide-enter-from,
.earn-modal-slide-leave-to {
  opacity: 0;
  transform: translateY(20px);
}

.slide-enter-active,
.slide-leave-active {
  transition: transform 0.25s ease, opacity 0.25s ease;
}

.slide-enter-from {
  opacity: 0;
  transform: translateX(24px);
}

.slide-leave-to {
  opacity: 0;
  transform: translateX(-24px);
}
</style>
