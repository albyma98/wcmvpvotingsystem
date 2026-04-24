<template>
  <article
    ref="entryRef"
    role="button"
    :tabindex="canPlay ? 0 : -1"
    :aria-label="canPlay ? `Gioca con ${config.sponsor_name}` : `Hai già giocato con ${config.sponsor_name}`"
    :aria-disabled="!canPlay"
    class="animate-on-enter mt-[2.4vh] w-full select-none"
    :class="canPlay ? 'cursor-pointer' : 'cursor-default'"
    @click="handleClick"
    @keydown.enter.prevent="handleClick"
    @keydown.space.prevent="handleClick"
  >
    <div
      class="flex items-center gap-3 rounded-2xl px-4 py-3 shadow-lg transition-opacity"
      :class="canPlay ? '' : 'opacity-55'"
      :style="{ background: config.primary_color, color: config.secondary_color }"
    >
      <!-- Logo sponsor -->
      <img
        v-if="config.sponsor_logo_url"
        :src="config.sponsor_logo_url"
        :alt="`Logo ${config.sponsor_name}`"
        class="h-9 w-9 shrink-0 rounded-lg object-contain"
        loading="lazy"
      />
      <span v-else class="shrink-0 text-2xl" aria-hidden="true">🏆</span>

      <!-- Testo -->
      <div class="flex min-w-0 flex-1 flex-col">
        <span class="truncate text-sm font-black leading-tight">
          {{ canPlay ? `Gioca con ${config.sponsor_name}` : `Hai già giocato con ${config.sponsor_name}` }}
        </span>
        <span class="mt-0.5 text-[0.7rem] font-semibold opacity-75">
          {{ gameTypeLabel }}
        </span>
      </div>

      <!-- Badge reward -->
      <span
        v-if="canPlay && config.reward_type === 'coins' && config.reward_coins > 0"
        class="shrink-0 rounded-full bg-amber-400 px-2.5 py-0.5 text-xs font-black text-slate-900"
      >
        {{ config.reward_coins }} 🪙
      </span>

      <!-- Indicatore stato -->
      <span v-if="canPlay" class="shrink-0 text-lg font-black" aria-hidden="true">▶</span>
      <span v-else class="shrink-0 text-sm font-bold opacity-80" aria-hidden="true">✓</span>
    </div>
  </article>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { trackAppEvent } from '../eventTracking';

const props = defineProps({
  config: { type: Object, required: true },
  canPlay: { type: Boolean, default: true },
  playsUsed: { type: Number, default: 0 },
  eventId: { type: Number, default: 0 },
});

const emit = defineEmits(['open']);

const entryRef = ref(null);

const gameTypeLabel = computed(() => {
  const map = {
    tap_challenge: 'Tap Battle ⚡',
    memory_flash: 'Memory Flash 🃏',
    sponsor_rush: 'Sponsor Rush 🏃',
  };
  return map[props.config?.game_type] ?? props.config?.game_type ?? '';
});

// IntersectionObserver — traccia impression una sola volta per sessione
const impressionKey = computed(
  () => `branded_game_impression_${props.eventId}_${props.config?.sponsor_id || ''}`,
);

let observer = null;

function trackImpressionOnce() {
  if (typeof sessionStorage === 'undefined') return;
  if (sessionStorage.getItem(impressionKey.value)) return;
  sessionStorage.setItem(impressionKey.value, '1');
  trackAppEvent(
    'branded_game.impression',
    {
      sponsor_id: props.config?.sponsor_id,
      game_type: props.config?.game_type,
      event_id: props.eventId,
      can_play: props.canPlay,
    },
    'branded_game',
  );
}

onMounted(() => {
  if (typeof IntersectionObserver === 'undefined') {
    // Browser vecchio: traccia impression al primo click (vedi handleClick)
    return;
  }
  if (!entryRef.value) return;
  observer = new IntersectionObserver(
    (entries) => {
      if (entries[0]?.isIntersecting) {
        trackImpressionOnce();
        observer?.disconnect();
        observer = null;
      }
    },
    { threshold: 0.5 },
  );
  observer.observe(entryRef.value);
});

onBeforeUnmount(() => {
  observer?.disconnect();
  observer = null;
});

function handleClick() {
  if (!props.canPlay) return;
  // Fallback impression per browser senza IntersectionObserver
  if (typeof IntersectionObserver === 'undefined') {
    trackImpressionOnce();
  }
  trackAppEvent(
    'branded_game.entry_clicked',
    {
      sponsor_id: props.config?.sponsor_id,
      game_type: props.config?.game_type,
      event_id: props.eventId,
    },
    'branded_game',
  );
  emit('open');
}
</script>
