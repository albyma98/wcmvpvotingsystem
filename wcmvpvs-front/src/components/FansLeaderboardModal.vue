<template>
  <Teleport to="body">
    <Transition name="leaderboard-fade">
      <div
        v-if="modelValue"
        class="leaderboard-modal fixed inset-0 z-[90] flex items-end bg-slate-950/80 p-2 sm:items-center sm:justify-center"
        role="dialog"
        aria-label="Classifica tifosi"
        aria-modal="true"
      >
        <div class="leaderboard-modal__card w-full max-w-md overflow-hidden rounded-2xl border border-white/25 bg-slate-950 text-white shadow-2xl">
          <header class="flex items-center justify-between border-b border-white/15 px-4 py-3">
            <h2 class="text-base font-black uppercase tracking-wide">Classifica tifosi</h2>
            <button
              type="button"
              class="rounded-md border border-white/25 px-2 py-1 text-xs font-bold"
              @click="emit('update:modelValue', false)"
            >
              CHIUDI
            </button>
          </header>

          <div class="max-h-[75vh] overflow-y-auto p-4">
            <div class="mb-3 flex gap-2" role="group" aria-label="Periodo classifica">
              <button v-for="option in periods" :key="option.value" type="button" class="flex-1 rounded-lg border px-3 py-2 text-sm font-bold" :class="period === option.value ? 'border-amber-300 bg-amber-300 text-slate-950' : 'border-white/25 bg-white/5'" :aria-pressed="period === option.value" @click="selectPeriod(option.value)">{{ option.label }}</button>
            </div>
            <p class="mb-3 text-xs text-white/70">{{ period === 'event' ? 'Punti guadagnati in questa partita' : 'Punti totali guadagnati nella società' }}</p>
            <p v-if="loading && !rows.length" role="status">Caricamento classifica…</p>
            <p v-else-if="error" role="status">{{ error }}</p>
            <p v-else-if="!rows.length">{{ eventId ? 'Gioca e vota per popolare la classifica.' : 'La classifica sarà disponibile con un evento attivo.' }}</p>
            <ol v-if="rows.length && !error" class="space-y-2" :aria-label="period === 'event' ? 'Classifica tifosi di questa giornata' : 'Classifica generale tifosi della società'" :aria-busy="loading">
              <li
                v-for="entry in rows"
                :key="entry.rank"
                class="flex items-center justify-between gap-3 rounded-lg border px-3 py-2"
                :class="entry.is_me ? 'border-amber-300/50 bg-amber-400/10' : 'border-white/15 bg-white/5'"
                :aria-current="entry.is_me ? 'true' : undefined"
              >
                <p class="min-w-0 break-words font-bold">{{ medals[entry.rank - 1] || `#${entry.rank}` }} {{ entry.name }}{{ entry.is_me ? ' · Tu' : '' }}</p>
                <p class="shrink-0 font-black">{{ entry.points }} pt</p>
              </li>
            </ol>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { onBeforeUnmount, ref, watch } from 'vue';
import { apiClient, getDeviceHeaders, getFanSessionHeaders, getOrganizationSlug } from '../api';

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false,
  },
  topList: {
    type: Array,
    default: () => [],
  },
  eventId: {
    type: Number,
    default: 0,
  },
});

const emit = defineEmits(['update:modelValue']);

const medals = ['🥇', '🥈', '🥉'];
const periods = [{ value: 'general', label: 'Generale' }, { value: 'event', label: 'Questa giornata' }];
const period = ref('event');
const rows = ref([]);
const loading = ref(false);
const error = ref('');
let revision = 0;

async function loadLeaderboard() {
  const request = ++revision;
  const eventId = props.eventId;
  const organization = getOrganizationSlug();
  if (!props.modelValue || !eventId) {
    rows.value = [];
    loading.value = false;
    error.value = '';
    return;
  }
  loading.value = true;
  error.value = '';
  const isCurrent = () => request === revision && props.modelValue && eventId === props.eventId && organization === getOrganizationSlug();
  try {
    const { data } = await apiClient.get(`/events/${eventId}/coins-leaderboard`, {
      headers: { ...getDeviceHeaders(), ...getFanSessionHeaders() },
      params: { view: 'list', period: period.value },
    });
    if (!isCurrent()) return;
    rows.value = (Array.isArray(data?.ranking) ? data.ranking : []).map(entry => ({
      ...entry,
      points: Number(entry.points ?? entry.coins ?? 0),
    }));
  } catch {
    if (isCurrent()) error.value = 'Impossibile caricare la classifica. Riapri il pannello per riprovare.';
  } finally {
    if (request === revision) loading.value = false;
  }
}

function selectPeriod(value) {
  if (!periods.some(option => option.value === value) || period.value === value) return;
  period.value = value;
  rows.value = [];
  return loadLeaderboard();
}

watch([() => props.modelValue, () => props.eventId, () => props.topList], ([open, event], previous = []) => {
  // Parent preview changes also signal live updates while this list is open.
  if (open && (!previous[0] || event !== previous[1])) {
    period.value = 'event';
    rows.value = [];
  }
  loadLeaderboard();
}, { immediate: true });
onBeforeUnmount(() => {
  revision += 1;
});
</script>

<style scoped>
.leaderboard-fade-enter-active,
.leaderboard-fade-leave-active {
  transition: opacity 0.2s ease;
}

.leaderboard-fade-enter-from,
.leaderboard-fade-leave-to {
  opacity: 0;
}

.leaderboard-modal__card {
  background:
    radial-gradient(circle at 20% 0%, rgba(251, 191, 36, 0.2), transparent 55%),
    radial-gradient(circle at 90% 10%, rgba(59, 130, 246, 0.2), transparent 45%),
    #020617;
}
</style>
