<template>
  <div class="fixed inset-0 z-[200] bg-slate-950/95 text-white">
    <div class="flex h-full flex-col p-4 sm:p-6">
      <header class="mb-4 flex items-start justify-between gap-3">
        <div>
          <p class="text-xs font-semibold uppercase tracking-[0.25em] text-amber-300/90">MVP del pubblico</p>
          <h2 class="text-2xl font-black uppercase tracking-tight sm:text-3xl">Scegli il tuo MVP</h2>
          <p class="text-sm text-slate-200/90">Tocca un giocatore per votare. La scelta comparirà nella home newui.</p>
        </div>
        <button
          type="button"
          class="rounded-md border border-white/30 bg-white/10 px-3 py-1.5 text-sm font-bold uppercase"
          @click="$emit('close')"
        >
          Chiudi
        </button>
      </header>

      <div class="relative min-h-0 flex-1">
        <VolleyCourtModal
          v-if="players.length"
          :players="players"
          :card-size="76"
          :selected-player-id="selectedPlayerId"
          :is-voting="isVoting"
          :disable-votes="isVoting"
          :voting-open="true"
          class="h-full"
          @select="handlePlayerSelect"
        />
        <div v-else class="flex h-full items-center justify-center rounded-3xl border border-white/25 bg-white/5 p-6 text-center">
          <p v-if="isLoadingPlayers">Caricamento giocatori…</p>
          <p v-else-if="playersError" class="text-red-300">{{ playersError }}</p>
          <p v-else>Nessun giocatore disponibile.</p>
        </div>
      </div>

      <p v-if="feedbackMessage" class="mt-4 text-sm font-semibold" :class="feedbackClass">{{ feedbackMessage }}</p>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue';
import VolleyCourtModal from './VolleyCourtModal.vue';
import { apiClient, fetchVoteStatus, vote } from '../api';
import { trackAppEvent, updateTrackingContext } from '../eventTracking';
import { DEFAULT_ROSTER_SCHEMA, mapPlayersToLayout } from '../roster';

const props = defineProps({
  eventId: {
    type: Number,
    default: undefined,
  },
});

const emit = defineEmits(['close', 'voted']);

const rawPlayers = ref([]);
const rosterSchema = ref(DEFAULT_ROSTER_SCHEMA);
const isLoadingPlayers = ref(false);
const playersError = ref('');
const selectedPlayerId = ref(null);
const isVoting = ref(false);
const feedbackMessage = ref('');
const isErrorFeedback = ref(false);

const players = computed(() => {
  const calledUp = Array.isArray(rawPlayers.value)
    ? rawPlayers.value.filter((player) => player?.is_called_up === true)
    : [];

  const schema = calledUp.length;
  const effectiveSchema = schema === 12 || schema === 13 || schema === 14 ? schema : rosterSchema.value;
  return mapPlayersToLayout(calledUp, { layoutSchema: effectiveSchema });
});

const feedbackClass = computed(() => (isErrorFeedback.value ? 'text-red-300' : 'text-emerald-300'));

async function loadPlayers() {
  trackAppEvent('vote.screen_opened', { event_id: props.eventId }, 'vote');
  isLoadingPlayers.value = true;
  playersError.value = '';
  try {
    const { data } = await apiClient.get('/public/players');
    const schemaCandidate = Number(data?.roster_schema);
    rosterSchema.value =
      schemaCandidate === 12 || schemaCandidate === 13 || schemaCandidate === 14
        ? schemaCandidate
        : DEFAULT_ROSTER_SCHEMA;

    const payload = Array.isArray(data?.players) ? data.players : data;
    rawPlayers.value = Array.isArray(payload) ? payload : [];
    trackAppEvent('vote.roster_loaded', { players_count: rawPlayers.value.length }, 'vote');
  } catch (error) {
    console.error('Impossibile caricare i giocatori per newui', error);
    playersError.value = 'Non è stato possibile caricare i giocatori. Riprova più tardi.';
    trackAppEvent('vote.roster_load_failed', { message: playersError.value }, 'vote');
    rawPlayers.value = [];
  } finally {
    isLoadingPlayers.value = false;
  }
}

async function loadCurrentVote() {
  if (!props.eventId) {
    return;
  }

  const response = await fetchVoteStatus(props.eventId);
  if (response?.ok && response.playerId) {
    selectedPlayerId.value = response.playerId;
    trackAppEvent('vote.current_selection_loaded', { player_id: response.playerId }, 'vote');
  }
}

async function handlePlayerSelect(player) {
  if (!props.eventId || !player?.id || isVoting.value) {
    return;
  }

  isVoting.value = true;
  feedbackMessage.value = '';
  trackAppEvent('vote.player_selected', { player_id: player.id, player_name: `${player.first_name || ''} ${player.last_name || ''}`.trim() }, 'vote');
  try {
    trackAppEvent('vote.submit_attempted', { event_id: props.eventId, player_id: player.id }, 'vote');
    const response = await vote({ eventId: props.eventId, playerId: player.id });
    if (!response?.ok) {
      isErrorFeedback.value = true;
      feedbackMessage.value = response?.message || 'Voto non registrato. Riprova.';
      trackAppEvent('vote.submit_failed', { player_id: player.id, message: feedbackMessage.value, status: response?.status || 0 }, 'vote');
      return;
    }

    selectedPlayerId.value = player.id;
    isErrorFeedback.value = false;
    feedbackMessage.value = 'Voto registrato con successo!';
    trackAppEvent('vote.submitted', { player_id: player.id, player_name: `${player.first_name || ''} ${player.last_name || ''}`.trim() }, 'vote');
    emit('voted', player);
  } catch (error) {
    console.error('Errore voto newui', error);
    isErrorFeedback.value = true;
    feedbackMessage.value = 'Si è verificato un errore. Riprova.';
    trackAppEvent('vote.submit_failed', { player_id: player.id, message: feedbackMessage.value }, 'vote');
  } finally {
    isVoting.value = false;
  }
}

onMounted(async () => {
  await Promise.all([loadPlayers(), loadCurrentVote()]);
});
</script>
