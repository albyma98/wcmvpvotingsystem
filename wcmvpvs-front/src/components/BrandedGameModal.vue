<template>
  <Teleport to="body">
    <Transition name="bg-modal-fade">
      <div
        v-if="modelValue"
        class="fixed inset-0 z-[125] flex flex-col overflow-y-auto"
        role="dialog"
        aria-modal="true"
        :aria-label="`Mini-game con ${config.sponsor_name}`"
        @keydown.esc="handleDismiss"
      >
        <!-- Overlay -->
        <div class="absolute inset-0 bg-slate-950/92 backdrop-blur-sm" aria-hidden="true" />

        <!-- Panel -->
        <div class="relative flex min-h-screen w-full flex-col">

          <!-- Header brandizzato -->
          <header
            class="sticky top-0 z-10 px-4 pb-3 pt-4"
            :style="{ background: config.primary_color }"
          >
            <div class="flex items-start justify-between gap-3">
              <div class="flex items-center gap-2.5 min-w-0">
                <img
                  v-if="config.sponsor_logo_url"
                  :src="config.sponsor_logo_url"
                  :alt="`Logo ${config.sponsor_name}`"
                  class="h-9 w-9 shrink-0 rounded-lg object-contain"
                />
                <span v-else class="shrink-0 text-2xl" aria-hidden="true">🏆</span>
                <div class="min-w-0">
                  <p
                    class="truncate text-xs font-semibold uppercase tracking-widest opacity-75"
                    :style="{ color: config.secondary_color }"
                  >
                    Sponsored by
                  </p>
                  <h2
                    class="truncate text-base font-black leading-tight"
                    :style="{ color: config.secondary_color }"
                  >
                    {{ config.sponsor_name }}
                  </h2>
                  <p
                    class="mt-0.5 text-[0.7rem] font-semibold opacity-70"
                    :style="{ color: config.secondary_color }"
                  >
                    {{ gameTypeLabel }}
                    <span v-if="config.reward_type === 'coins' && config.reward_coins > 0">
                      · {{ config.reward_coins }} 🪙
                    </span>
                  </p>
                </div>
              </div>
              <button
                type="button"
                class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-xl font-bold opacity-80 transition hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-white/50"
                :style="{ color: config.secondary_color }"
                aria-label="Chiudi"
                @click="handleDismiss"
              >
                ✕
              </button>
            </div>
          </header>

          <!-- Body -->
          <div class="flex flex-1 flex-col bg-slate-950 px-4 pb-10 pt-5">

            <!-- ── Fase: exhausted ── -->
            <div v-if="phase === 'exhausted'" class="flex flex-1 flex-col items-center justify-center gap-4 text-center">
              <span class="text-5xl" aria-hidden="true">✅</span>
              <h3 class="text-xl font-black text-white">Hai già giocato!</h3>
              <p class="text-sm text-slate-400">
                Hai usato tutte le {{ config.max_plays_per_user }} partite disponibili per questo evento.
              </p>
              <BrandedGameCta v-if="config.cta_url" :label="config.cta_label" :url="config.cta_url" @click="handleCtaClick" />
              <button
                type="button"
                class="mt-2 rounded-full border border-white/20 px-6 py-2.5 text-sm font-bold text-white transition hover:bg-white/10"
                @click="handleDismiss"
              >
                Chiudi
              </button>
            </div>

            <!-- ── Fase: intro ── -->
            <div v-else-if="phase === 'intro'" class="flex flex-1 flex-col items-center justify-center gap-5 text-center">
              <div
                class="flex h-20 w-20 items-center justify-center rounded-2xl shadow-xl"
                :style="{ background: config.primary_color }"
              >
                <img
                  v-if="config.sponsor_logo_url"
                  :src="config.sponsor_logo_url"
                  alt=""
                  class="h-12 w-12 object-contain"
                />
                <span v-else class="text-4xl" aria-hidden="true">🏆</span>
              </div>
              <div>
                <h3 class="text-2xl font-black text-white">{{ gameTypeLabel }}</h3>
                <p class="mt-1 text-sm text-slate-400">Presentato da {{ config.sponsor_name }}</p>
              </div>
              <div
                v-if="config.reward_type === 'coins' && config.reward_coins > 0"
                class="rounded-full border border-amber-300/40 bg-amber-300/10 px-5 py-2 text-base font-black text-amber-200"
              >
                Vinci {{ config.reward_coins }} 🪙 completando il gioco
              </div>
              <p class="text-xs text-slate-500">
                {{ playsRemaining }} partita/e rimanente/i
              </p>
              <button
                type="button"
                class="mt-2 inline-flex items-center gap-2 rounded-2xl px-8 py-3.5 text-base font-black shadow-lg transition active:scale-95"
                :style="{ background: config.primary_color, color: config.secondary_color }"
                @click="startGame"
              >
                Inizia ▶
              </button>
              <BrandedGameCta v-if="config.cta_url" :label="config.cta_label" :url="config.cta_url" @click="handleCtaClick" />
            </div>

            <!-- ── Fase: playing ── -->
            <div v-else-if="phase === 'playing'" class="flex flex-1 flex-col">
              <div class="relative min-h-[min(62vh,560px)]">
                <component
                  :is="gameComponent"
                  :event-id="eventId"
                  :wallet-coins="walletCoins"
                  class="h-full w-full"
                  @claim="handleClaim"
                  @exit="handleDismiss"
                />
              </div>
            </div>

            <!-- ── Fase: submitting ── -->
            <div v-else-if="phase === 'submitting'" class="flex flex-1 flex-col items-center justify-center gap-4">
              <div class="h-8 w-8 animate-spin rounded-full border-4 border-white/20 border-t-white" aria-hidden="true" />
              <p class="text-sm text-slate-400">Registrazione risultato…</p>
            </div>

            <!-- ── Fase: result ── -->
            <div v-else-if="phase === 'result'" class="flex flex-1 flex-col items-center justify-center gap-5 text-center">
              <span class="text-6xl" aria-hidden="true">{{ lastResult.rewarded_coins > 0 ? '🏆' : '💪' }}</span>
              <div>
                <h3 class="text-2xl font-black text-white">
                  {{ lastResult.rewarded_coins > 0 ? `Hai vinto ${lastResult.rewarded_coins} 🪙!` : 'Partita completata!' }}
                </h3>
                <p class="mt-1 text-sm text-slate-400">
                  {{ lastResult.remaining_plays > 0
                    ? `${lastResult.remaining_plays} partita/e rimanente/i`
                    : 'Hai usato tutte le partite per questo evento.' }}
                </p>
              </div>
              <BrandedGameCta v-if="config.cta_url" :label="config.cta_label" :url="config.cta_url" @click="handleCtaClick" />
              <div class="flex gap-3">
                <button
                  v-if="lastResult.remaining_plays > 0"
                  type="button"
                  class="rounded-full px-6 py-2.5 text-sm font-black shadow-lg transition active:scale-95"
                  :style="{ background: config.primary_color, color: config.secondary_color }"
                  @click="restartGame"
                >
                  Gioca ancora
                </button>
                <button
                  type="button"
                  class="rounded-full border border-white/20 px-6 py-2.5 text-sm font-bold text-white transition hover:bg-white/10"
                  @click="handleDismiss"
                >
                  Chiudi
                </button>
              </div>
            </div>

            <!-- ── Fase: error ── -->
            <div v-else-if="phase === 'error'" class="flex flex-1 flex-col items-center justify-center gap-4 text-center">
              <span class="text-4xl" aria-hidden="true">⚠️</span>
              <p class="text-sm text-slate-400">{{ errorMessage }}</p>
              <button
                type="button"
                class="rounded-full border border-white/20 px-6 py-2.5 text-sm font-bold text-white"
                @click="handleDismiss"
              >
                Chiudi
              </button>
            </div>

          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { computed, defineAsyncComponent, onErrorCaptured, onUnmounted, ref, watch } from 'vue';
import { apiClient } from '../api';
import { trackAppEvent } from '../eventTracking';

// ── Sub-component: CTA sponsor ──────────────────────────────────────────────
// Inline per evitare un file separato per un componente di 3 righe
const BrandedGameCta = {
  props: {
    label: { type: String, default: '' },
    url: { type: String, default: '' },
  },
  emits: ['click'],
  template: `
    <a
      :href="url"
      target="_blank"
      rel="noopener noreferrer"
      class="inline-flex items-center gap-1.5 rounded-full border border-white/20 px-5 py-2 text-sm font-bold text-slate-300 transition hover:bg-white/10 hover:text-white"
      @click.stop="$emit('click')"
    >
      🔗 {{ label || 'Scopri di più' }}
    </a>
  `,
};

// ── Game components (lazy per tipo) ────────────────────────────────────────
const gameComponentMap = {
  tap_challenge: defineAsyncComponent(() =>
    import('./minigames/TapChallenge.vue'),
  ),
  memory_flash: defineAsyncComponent(() =>
    import('./minigames/MemoryFlashGame.vue'),
  ),
  sponsor_rush: defineAsyncComponent(() =>
    import('./minigames/SponsorRushGame.vue'),
  ),
};

// ── Props / emits ──────────────────────────────────────────────────────────
const props = defineProps({
  modelValue: { type: Boolean, default: false },
  eventId: { type: Number, default: 0 },
  config: { type: Object, required: true },
  canPlay: { type: Boolean, default: true },
  playsUsed: { type: Number, default: 0 },
  walletCoins: { type: Number, default: 0 },
});

const emit = defineEmits(['update:modelValue', 'coins-earned']);

// ── State ──────────────────────────────────────────────────────────────────
const phase = ref('intro');
const lastResult = ref({ rewarded_coins: 0, remaining_plays: 0 });
const errorMessage = ref('');

const playsRemaining = computed(() =>
  Math.max(0, props.config.max_plays_per_user - props.playsUsed),
);

const gameTypeLabel = computed(() => {
  const map = {
    tap_challenge: 'Tap Battle ⚡',
    memory_flash: 'Memory Flash 🃏',
    sponsor_rush: 'Sponsor Rush 🏃',
  };
  return map[props.config?.game_type] ?? props.config?.game_type ?? '';
});

const gameComponent = computed(
  () => gameComponentMap[props.config?.game_type] ?? null,
);

// ── Lifecycle di apertura ──────────────────────────────────────────────────
watch(
  () => props.modelValue,
  (open) => {
    if (!open) return;
    // Reset ogni volta che si apre
    lastResult.value = { rewarded_coins: 0, remaining_plays: 0 };
    errorMessage.value = '';
    phase.value = props.canPlay ? 'intro' : 'exhausted';

    trackAppEvent(
      'branded_game.opened',
      {
        sponsor_id: props.config?.sponsor_id,
        game_type: props.config?.game_type,
        event_id: props.eventId,
        can_play: props.canPlay,
      },
      'branded_game',
    );
  },
);

// Error boundary — cattura crash del mini-game e chiude pulito
onErrorCaptured((err) => {
  console.error('[BrandedGameModal] game error captured:', err);
  errorMessage.value = 'Il gioco ha riscontrato un errore. Riprova più tardi.';
  phase.value = 'error';
  return false; // non propagare
});

// ESC key dal documento (focus trap semplice: il div cattura già ESC via @keydown)
function handleEscKey(e) {
  if (e.key === 'Escape' && props.modelValue) handleDismiss();
}
document.addEventListener('keydown', handleEscKey);
onUnmounted(() => document.removeEventListener('keydown', handleEscKey));

// ── Azioni ─────────────────────────────────────────────────────────────────
function startGame() {
  if (!gameComponent.value) {
    phase.value = 'error';
    errorMessage.value = 'Tipo di gioco non supportato.';
    return;
  }
  phase.value = 'playing';
  trackAppEvent(
    'branded_game.started',
    {
      sponsor_id: props.config?.sponsor_id,
      game_type: props.config?.game_type,
      event_id: props.eventId,
    },
    'branded_game',
  );
}

function restartGame() {
  phase.value = 'playing';
  trackAppEvent(
    'branded_game.started',
    {
      sponsor_id: props.config?.sponsor_id,
      game_type: props.config?.game_type,
      event_id: props.eventId,
      replay: true,
    },
    'branded_game',
  );
}

async function handleClaim(payload) {
  const score = typeof payload?.coins === 'number' ? payload.coins : 0;
  phase.value = 'submitting';

  trackAppEvent(
    'branded_game.completed',
    {
      sponsor_id: props.config?.sponsor_id,
      game_type: props.config?.game_type,
      event_id: props.eventId,
      score,
    },
    'branded_game',
  );

  try {
    const { data } = await apiClient.post(
      `/events/${props.eventId}/branded-game/result`,
      {
        score,
        duration_ms: 0,
        completed: true,
        payload: { game_score: score },
        session_id: '',
      },
    );

    lastResult.value = {
      rewarded_coins: data?.rewarded_coins ?? 0,
      remaining_plays: data?.remaining_plays ?? 0,
    };

    if (lastResult.value.rewarded_coins > 0) {
      emit('coins-earned', lastResult.value.rewarded_coins);
      trackAppEvent(
        'branded_game.reward_claimed',
        {
          sponsor_id: props.config?.sponsor_id,
          reward_type: props.config?.reward_type,
          rewarded_coins: lastResult.value.rewarded_coins,
          event_id: props.eventId,
        },
        'branded_game',
      );
    }

    phase.value = 'result';
  } catch (err) {
    if (err?.response?.status === 409) {
      // plays esauriti lato server
      lastResult.value = { rewarded_coins: 0, remaining_plays: 0 };
      phase.value = 'exhausted';
    } else {
      errorMessage.value = 'Impossibile registrare il risultato. Riprova più tardi.';
      phase.value = 'error';
    }
  }
}

function handleCtaClick() {
  trackAppEvent(
    'branded_game.cta_clicked',
    {
      sponsor_id: props.config?.sponsor_id,
      cta_url: props.config?.cta_url,
      event_id: props.eventId,
    },
    'branded_game',
  );
}

function handleDismiss() {
  trackAppEvent(
    'branded_game.dismissed',
    {
      sponsor_id: props.config?.sponsor_id,
      phase: phase.value,
      event_id: props.eventId,
    },
    'branded_game',
  );
  emit('update:modelValue', false);
}
</script>

<style scoped>
.bg-modal-fade-enter-active,
.bg-modal-fade-leave-active {
  transition: opacity 0.2s ease;
}
.bg-modal-fade-enter-from,
.bg-modal-fade-leave-to {
  opacity: 0;
}
</style>
