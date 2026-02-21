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
                  <h2 class="text-2xl font-black text-white md:text-3xl">Guadagna Monete</h2>
                  <p class="mt-1 text-sm text-slate-300 md:text-base">Scegli un’attività e accumula monete</p>
                </div>
                <button
                  type="button"
                  class="inline-flex h-10 w-10 items-center justify-center rounded-full border border-white/20 bg-white/5 text-2xl leading-none text-white transition hover:bg-white/15"
                  aria-label="Chiudi modale Guadagna Monete"
                  @click="closeModal"
                >
                  ×
                </button>
              </div>
            </header>

            <div class="flex-1 overflow-y-auto px-4 pb-8 pt-5 md:px-6">
              <div class="mx-auto grid max-w-6xl grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
                <button
                  v-for="option in earnOptions"
                  :key="option.id"
                  type="button"
                  class="group rounded-2xl border border-white/15 bg-white/10 p-4 text-left shadow-[0_10px_28px_rgba(15,23,42,0.45)] backdrop-blur transition hover:-translate-y-0.5 hover:bg-white/15 disabled:cursor-not-allowed disabled:opacity-60"
                  :disabled="cooldowns[option.id] > 0"
                  @click="handleOptionClick(option)"
                >
                  <div class="flex items-start justify-between gap-2">
                    <span class="text-2xl" aria-hidden="true">{{ option.icon }}</span>
                    <span class="rounded-full border border-amber-300/40 bg-amber-300/15 px-2 py-0.5 text-xs font-bold text-amber-200">
                      +{{ option.reward }}
                    </span>
                  </div>
                  <h3 class="mt-3 text-lg font-extrabold text-white">{{ option.title }}</h3>
                  <p class="mt-1 text-sm text-slate-300">{{ option.description }}</p>
                  <p class="mt-4 text-xs font-semibold uppercase tracking-wide" :class="cooldowns[option.id] > 0 ? 'text-orange-300' : 'text-emerald-300'">
                    {{ cooldowns[option.id] > 0 ? `In cooldown ${formatCooldown(cooldowns[option.id])}` : 'Disponibile' }}
                  </p>
                </button>
              </div>
            </div>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { computed, onBeforeUnmount, ref, watch } from 'vue';
import { getEarnCooldownRemainingSeconds, startEarnCooldown } from '../utils/earnCooldown';

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits(['update:modelValue']);

const earnOptions = [
  { id: 'reaction-test', title: 'Reaction Test', description: 'Testa i riflessi e scala la classifica.', reward: 10, icon: '⚡', type: 'game', cooldownSeconds: 90 },
  { id: 'quiz-lampo', title: 'Quiz Lampo', description: 'Rispondi veloce a domande a tema match.', reward: 15, icon: '🧠', type: 'game', cooldownSeconds: 120 },
  { id: 'tap-challenge', title: 'Tap Challenge', description: 'Tappa più forte che puoi in 10 secondi.', reward: 8, icon: '👆', type: 'game', cooldownSeconds: 60 },
  { id: 'pronostico-set', title: 'Pronostico Set', description: 'Indovina il risultato del prossimo set.', reward: 12, icon: '🎯', type: 'game', cooldownSeconds: 180 },
  { id: 'codice-sponsor', title: 'Codice Sponsor', description: 'Inserisci il codice mostrato sul maxischermo.', reward: 20, icon: '🏷️', type: 'action', cooldownSeconds: 300 },
  { id: 'condividi-torna', title: 'Condividi & Torna', description: 'Condividi l’evento e torna per riscattare.', reward: 5, icon: '📣', type: 'action', cooldownSeconds: 150 },
];

const nowTick = ref(Date.now());
const cooldowns = computed(() =>
  earnOptions.reduce((accumulator, option) => {
    accumulator[option.id] = getEarnCooldownRemainingSeconds(option.id, nowTick.value);
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

function forceTick() {
  nowTick.value = Date.now();
}

function closeModal() {
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

function getRouter() {
  if (typeof window === 'undefined') {
    return { push: () => undefined };
  }

  const fromWindow = window.__VUE_ROUTER__;
  if (fromWindow && typeof fromWindow.push === 'function') {
    return fromWindow;
  }

  return {
    push(path) {
      window.location.assign(path);
    },
  };
}

function handleOptionClick(option) {
  if (cooldowns.value[option.id] > 0) {
    return;
  }

  startEarnCooldown(option.id, option.cooldownSeconds);
  forceTick();

  if (option.type === 'game') {
    const router = getRouter();
    router.push(`/earn/${option.id}`);
    closeModal();
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
</style>
