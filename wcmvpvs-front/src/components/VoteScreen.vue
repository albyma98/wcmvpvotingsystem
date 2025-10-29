<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import VolleyCourt from './VolleyCourt.vue';
import PlayerCard from './PlayerCard.vue';
import SelfieMvpSection from './SelfieMvpSection.vue';
import ReactionTestSection from './ReactionTestSection.vue';
import { apiClient, vote, fetchVoteStatus, resolveApiUrl } from '../api';
import { mapPlayersToLayout } from '../roster';

const props = defineProps({
  eventId: {
    type: Number,
    default: undefined,
  },
  activeEvent: {
    type: Object,
    default: null,
  },
  activeEventChecked: {
    type: Boolean,
    default: false,
  },
  loadingActiveEvent: {
    type: Boolean,
    default: false,
  },
});

const rawPlayers = ref([]);
const isLoadingPlayers = ref(false);
const playersError = ref('');

const fieldPlayers = computed(() => mapPlayersToLayout(rawPlayers.value));

const sponsors = ref([]);
const hasVoted = ref(false);
const isCheckingVoteStatus = ref(false);

const totalVotes = ref(0);
const isVoteTotalLoading = ref(false);
const voteTotalError = ref('');
const isRefreshingVoteTotal = ref(false);
let voteTotalTimer = null;
let countdownTimer = null;
const nowTimestamp = ref(Date.now());

const updateNowTimestamp = () => {
  nowTimestamp.value = Date.now();
};

const stopCountdownTimer = () => {
  if (typeof window !== 'undefined' && countdownTimer) {
    window.clearInterval(countdownTimer);
    countdownTimer = null;
  }
};

const startCountdownTimer = () => {
  if (typeof window === 'undefined') {
    return;
  }
  stopCountdownTimer();
  updateNowTimestamp();
  countdownTimer = window.setInterval(updateNowTimestamp, 1000);
};

const formattedVoteTotal = computed(() =>
  Number.isFinite(totalVotes.value)
    ? totalVotes.value.toLocaleString('it-IT')
    : '0',
);

const stopVoteTotalPolling = () => {
  if (voteTotalTimer) {
    window.clearInterval(voteTotalTimer);
    voteTotalTimer = null;
  }
};

const startVoteTotalPolling = () => {
  stopVoteTotalPolling();
  voteTotalTimer = window.setInterval(() => {
    refreshVoteTotal({ silent: true });
  }, 4000);
};

const refreshVoteTotal = async ({ silent = false } = {}) => {
  const eventId = currentEventId.value;
  if (!eventId) {
    totalVotes.value = 0;
    voteTotalError.value = '';
    if (!silent) {
      isVoteTotalLoading.value = false;
    }
    return;
  }

  if (isRefreshingVoteTotal.value) {
    return;
  }

  isRefreshingVoteTotal.value = true;
  if (!silent) {
    isVoteTotalLoading.value = true;
  }

  try {
    const { data } = await apiClient.get(`/events/${eventId}/votes/count`);
    const rawTotal = Number(
      typeof data?.total === 'number' ? data.total : data?.count ?? 0,
    );
    totalVotes.value = Number.isFinite(rawTotal) ? rawTotal : 0;
    voteTotalError.value = '';
  } catch (error) {
    console.error('Impossibile aggiornare il totale voti', error);
    voteTotalError.value = 'Totale voti non disponibile in questo momento.';
  } finally {
    if (!silent) {
      isVoteTotalLoading.value = false;
    }
    isRefreshingVoteTotal.value = false;
  }
};

async function loadSponsors() {
  try {
    const { data } = await apiClient.get('/sponsors');
    if (Array.isArray(data)) {
      sponsors.value = data
        .map((item, index) => {
          const image = typeof item?.logo_data === 'string' ? item.logo_data : '';
          if (!image) {
            return null;
          }
          const resolvedName =
            typeof item?.name === 'string' && item.name.trim() ? item.name.trim() : '';
          const resolvedLink =
            typeof item?.link_url === 'string' && item.link_url.trim()
              ? item.link_url.trim()
              : '';
          return {
            id: Number(item?.id) || index + 1,
            name: resolvedName,
            image,
            link: resolvedLink,
          };
        })
        .filter(Boolean);
    } else {
      sponsors.value = [];
    }
  } catch (error) {
    console.error('Impossibile caricare gli sponsor', error);
    sponsors.value = [];
  }
}

function recordSponsorClick(sponsor) {
  if (!sponsor || !sponsor.id) {
    return;
  }
  const eventId = currentEventId.value;
  if (!eventId) {
    return;
  }
  const endpoint = `/events/${eventId}/sponsors/${sponsor.id}/click`;
  const url = resolveApiUrl(endpoint);
  const payload = JSON.stringify({ at: new Date().toISOString() });

  if (typeof navigator !== 'undefined' && typeof navigator.sendBeacon === 'function') {
    try {
      const blob = new Blob([payload], { type: 'application/json' });
      navigator.sendBeacon(url, blob);
      return;
    } catch (error) {
      // ignore and try fetch fallback
    }
  }

  if (typeof fetch === 'function') {
    fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: payload,
      keepalive: true,
    }).catch(() => {});
    return;
  }

  apiClient.post(endpoint).catch(() => {});
}

const handleSponsorClick = (sponsor) => {
  recordSponsorClick(sponsor);
};

async function loadPlayers() {
  isLoadingPlayers.value = true;
  playersError.value = '';
  try {
    const { data } = await apiClient.get('/public/players');
    if (Array.isArray(data)) {
      rawPlayers.value = data.map((item) => ({
        id: Number(item?.id) || 0,
        first_name: typeof item?.first_name === 'string' ? item.first_name : '',
        last_name: typeof item?.last_name === 'string' ? item.last_name : '',
        role: typeof item?.role === 'string' ? item.role : '',
        jersey_number:
          typeof item?.jersey_number === 'number'
            ? item.jersey_number
            : Number.isFinite(Number(item?.jersey_number))
            ? Number(item?.jersey_number)
            : null,
        image_url: typeof item?.image_url === 'string' ? item.image_url : '',
      }));
    } else {
      rawPlayers.value = [];
    }
  } catch (error) {
    console.error('Impossibile caricare i giocatori', error);
    playersError.value = 'Non è stato possibile caricare i giocatori. Riprova più tardi.';
    rawPlayers.value = [];
  } finally {
    isLoadingPlayers.value = false;
  }
}

const votedPlayerId = ref(null);
const isVoting = ref(false);
const cardSize = ref(88);
const errorMessage = ref('');
const pendingPlayer = ref(null);
const showTicketModal = ref(false);
const showAlreadyVotedModal = ref(false);
const ticketCode = ref('');
const ticketQrUrl = ref('');
const ticketLoadError = ref('');
const isTicketLoading = ref(false);
const showVoteSummary = computed(
  () => hasVoted.value && Boolean(ticketCode.value || ticketQrUrl.value),
);

const handleSelfieSubmitted = () => {
  hasVoted.value = true;
};

async function refreshVoteStatus(eventId) {
  if (!eventId) {
    hasVoted.value = false;
    return;
  }
  isCheckingVoteStatus.value = true;
  try {
    const { ok, hasVoted: status } = await fetchVoteStatus(eventId);
    if (ok) {
      hasVoted.value = Boolean(status);
    }
  } catch (error) {
    console.warn('Impossibile verificare lo stato del voto', error);
  } finally {
    isCheckingVoteStatus.value = false;
  }
}

const sanitizeName = (value) => {
  if (typeof value !== 'string') {
    return '';
  }
  return value.trim();
};

const resolveTeamName = (event, keys) => {
  if (!event) {
    return '';
  }

  for (const key of keys) {
    if (key in event) {
      const resolved = sanitizeName(event[key]);
      if (resolved) {
        return resolved;
      }
    }
  }

  return '';
};

const homeTeamName = computed(() =>
  resolveTeamName(props.activeEvent, ['team1_name', 'team1', 'home_team', 'homeTeam', 'team1Name'])
);

const awayTeamName = computed(() =>
  resolveTeamName(props.activeEvent, ['team2_name', 'team2', 'away_team', 'awayTeam', 'team2Name'])
);

const eventTitle = computed(() => {
  const home = homeTeamName.value;
  const away = awayTeamName.value;

  if (home || away) {
    const fallbackHome = home || 'Squadra di casa';
    const fallbackAway = away || 'Squadra ospite';
    return `${fallbackHome} - ${fallbackAway}`;
  }

  return 'Vota il tuo MVP';
});

const currentEventId = computed(() => props.eventId ?? props.activeEvent?.id);
const showInactiveNotice = computed(() => props.activeEventChecked && !props.activeEvent);
const isCheckingActiveEvent = computed(() => props.loadingActiveEvent && !props.activeEventChecked);
const isVotingClosed = computed(() => {
  if (!props.activeEvent) {
    return false;
  }
  const raw =
    props.activeEvent.votes_closed ?? props.activeEvent.votesClosed ?? props.activeEvent.is_voting_closed;
  return Boolean(raw);
});

const resolveEventStartValue = (event) => {
  if (!event || typeof event !== 'object') {
    return null;
  }

  const candidateKeys = [
    'start_datetime',
    'startDatetime',
    'startDateTime',
    'start_time',
    'startTime',
    'start_at',
    'startAt',
    'start',
  ];

  for (const key of candidateKeys) {
    if (key in event) {
      const value = event[key];
      if (value instanceof Date) {
        return value;
      }
      if (typeof value === 'string') {
        const trimmed = value.trim();
        if (trimmed) {
          return trimmed;
        }
      }
      if (typeof value === 'number' && Number.isFinite(value)) {
        return value;
      }
    }
  }

  return null;
};

const eventStartTimestamp = computed(() => {
  const raw = resolveEventStartValue(props.activeEvent);
  if (!raw) {
    return null;
  }

  if (raw instanceof Date) {
    const timestamp = raw.getTime();
    return Number.isFinite(timestamp) ? timestamp : null;
  }

  if (typeof raw === 'number') {
    return raw > 0 ? raw : null;
  }

  if (typeof raw === 'string') {
    const normalized = raw.includes('T') ? raw : raw.replace(' ', 'T');
    const parsed = new Date(normalized);
    const timestamp = parsed.getTime();
    return Number.isNaN(timestamp) ? null : timestamp;
  }

  return null;
});

const timeUntilEventStartMs = computed(() => {
  const start = eventStartTimestamp.value;
  if (!start) {
    return 0;
  }
  const diff = start - nowTimestamp.value;
  return diff > 0 ? diff : 0;
});

const countdownSeconds = computed(() => Math.ceil(timeUntilEventStartMs.value / 1000));

const countdownParts = computed(() => {
  const total = countdownSeconds.value;
  const days = Math.floor(total / 86400);
  const hours = Math.floor((total % 86400) / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const seconds = total % 60;
  const totalHours = Math.floor(total / 3600);
  return { days, hours, minutes, seconds, totalHours };
});

const countdownLabel = computed(() => {
  const { totalHours, minutes, seconds } = countdownParts.value;
  return [totalHours, minutes, seconds].map((value) => String(value).padStart(2, '0')).join(':');
});

const countdownDaysLabel = computed(() => {
  const { days, hours } = countdownParts.value;
  if (days <= 0) {
    return '';
  }
  const dayLabel = days === 1 ? 'giorno' : 'giorni';
  const hourLabel = hours === 1 ? 'ora' : 'ore';
  return `${days} ${dayLabel} e ${hours} ${hourLabel} rimanenti`;
});

const countdownStartTimeLabel = computed(() => {
  const start = eventStartTimestamp.value;
  if (!start) {
    return '';
  }
  try {
    return new Intl.DateTimeFormat('it-IT', {
      dateStyle: 'full',
      timeStyle: 'short',
    }).format(new Date(start));
  } catch (error) {
    const date = new Date(start);
    if (typeof date.toLocaleString === 'function') {
      return date.toLocaleString('it-IT');
    }
    return date.toString();
  }
});

const isEventUpcoming = computed(() => timeUntilEventStartMs.value > 0);

watch(currentEventId, (eventId) => {
  votedPlayerId.value = null;
  pendingPlayer.value = null;
  errorMessage.value = '';
  showTicketModal.value = false;
  ticketCode.value = '';
  ticketQrUrl.value = '';
  ticketLoadError.value = '';
  isTicketLoading.value = false;
  showAlreadyVotedModal.value = false;
  totalVotes.value = 0;
  voteTotalError.value = '';
  stopVoteTotalPolling();
  if (eventId) {
    refreshVoteTotal();
    startVoteTotalPolling();
    hasVoted.value = false;
    refreshVoteStatus(eventId);
  } else {
    hasVoted.value = false;
  }
});

watch(fieldPlayers, (players) => {
  if (!pendingPlayer.value) {
    return;
  }
  const replacement = players.find((player) => player.id === pendingPlayer.value.id);
  if (replacement) {
    pendingPlayer.value = replacement;
  } else {
    pendingPlayer.value = null;
  }
});

watch(isVotingClosed, (closed) => {
  if (closed) {
    pendingPlayer.value = null;
    showTicketModal.value = false;
    showAlreadyVotedModal.value = false;
    ticketLoadError.value = '';
    isTicketLoading.value = false;
  }
});

watch(
  isEventUpcoming,
  (upcoming) => {
    if (upcoming) {
      startCountdownTimer();
      pendingPlayer.value = null;
      showTicketModal.value = false;
      showAlreadyVotedModal.value = false;
      ticketLoadError.value = '';
      isTicketLoading.value = false;
    } else {
      stopCountdownTimer();
    }
  },
  { immediate: true },
);

const clamp = (value, min, max) => Math.min(Math.max(value, min), max);

const updateCardSize = () => {
  const width = window.innerWidth;
  const height = window.innerHeight;
  const sizeFromWidth = width / 5.8;
  const sizeFromHeight = height / 9.8;
  cardSize.value = clamp(Math.min(sizeFromWidth, sizeFromHeight), 58, 112);
};

onMounted(() => {
  updateCardSize();
  window.addEventListener('resize', updateCardSize, { passive: true });
  loadSponsors();
  loadPlayers();
  if (currentEventId.value) {
    refreshVoteTotal();
    startVoteTotalPolling();
    refreshVoteStatus(currentEventId.value);
  }
});

onBeforeUnmount(() => {
  window.removeEventListener('resize', updateCardSize);
  stopVoteTotalPolling();
  stopCountdownTimer();
});

const disableVotes = computed(
  () =>
    Boolean(votedPlayerId.value) ||
    showInactiveNotice.value ||
    isCheckingActiveEvent.value ||
    isVotingClosed.value ||
    isEventUpcoming.value,
);

const openPlayerModal = (player) => {
  if (isVotingClosed.value || isEventUpcoming.value) {
    return;
  }

  if ((disableVotes.value && votedPlayerId.value !== player.id) || isVoting.value) {
    return;
  }
  pendingPlayer.value = player;
  errorMessage.value = '';
};

const closeModal = () => {
  if (isVoting.value) {
    return;
  }
  pendingPlayer.value = null;
};

const closeTicketModal = () => {
  showTicketModal.value = false;
  isTicketLoading.value = false;
};

const closeAlreadyVotedModal = () => {
  showAlreadyVotedModal.value = false;
};

const voteForPlayer = async (player) => {
  if (isVotingClosed.value || isEventUpcoming.value) {
    return;
  }

  if (isVoting.value || (votedPlayerId.value && votedPlayerId.value !== player.id)) {
    return;
  }

  if (votedPlayerId.value === player.id) {
    return;
  }

  errorMessage.value = '';
  isVoting.value = true;

  const eventId = currentEventId.value;
  if (!eventId) {
    errorMessage.value = 'Nessun evento attivo al momento.';
    isVoting.value = false;
    return;
  }

  try {
    const response = await vote({ eventId, playerId: player.id });
    if (response?.ok) {
      const voteResult = response.vote || {};
      votedPlayerId.value = player.id;
      pendingPlayer.value = null;
      hasVoted.value = true;

      const codeSource = voteResult.code || '';
      const qrSource = voteResult.qr_data || '';

      if (codeSource) {
        ticketCode.value = codeSource;
        ticketLoadError.value = '';
        isTicketLoading.value = Boolean(qrSource);
        ticketQrUrl.value = qrSource
          ? `https://api.qrserver.com/v1/create-qr-code/?size=180x180&data=${encodeURIComponent(qrSource)}`
          : '';
        if (!qrSource) {
          isTicketLoading.value = false;
        }
        showTicketModal.value = true;
        refreshVoteTotal({ silent: true });
      } else {
        console.warn('voteForPlayer: missing ticket data', response);
        errorMessage.value = 'Non siamo riusciti a generare il QR del ticket. Riprova.';
      }
    } else {
      if (response?.status === 409) {
        pendingPlayer.value = null;
        showAlreadyVotedModal.value = true;
        errorMessage.value = '';
        if (!votedPlayerId.value) {
          votedPlayerId.value = -1;
        }
        hasVoted.value = true;
      } else if (response?.status === 429) {
        errorMessage.value =
          response?.message ||
          'Stai votando troppo rapidamente. Attendi qualche istante e riprova.';
      } else {
        errorMessage.value =
          response?.message || 'Non è stato possibile registrare il voto. Riprova.';
      }
    }
  } catch (error) {
    console.error('vote error', error);
    errorMessage.value = 'Si è verificato un errore. Riprova tra qualche istante.';
  } finally {
    isVoting.value = false;
  }
};

const isModalOpen = computed(() => Boolean(pendingPlayer.value));

const modalActionLabel = computed(() => {
  if (!pendingPlayer.value) {
    return 'Vota MVP';
  }
  if (votedPlayerId.value === pendingPlayer.value.id) {
    return 'Voto registrato';
  }
  if (isVoting.value) {
    return 'Invio...';
  }
  return 'Vota MVP';
});

const confirmVote = () => {
  if (!pendingPlayer.value || votedPlayerId.value === pendingPlayer.value.id) {
    return;
  }
  voteForPlayer(pendingPlayer.value);
};

const handleQrLoaded = () => {
  isTicketLoading.value = false;
};

const handleQrError = () => {
  isTicketLoading.value = false;
  ticketQrUrl.value = '';
  ticketLoadError.value = 'Non siamo riusciti a caricare il QR del ticket. Riprova tra qualche istante.';
};
</script>

<template>
  <div class="min-h-screen bg-gradient-to-b from-slate-950 via-slate-900 to-slate-950 text-slate-100 flex flex-col">
    <main
      v-if="!isCheckingActiveEvent && !showInactiveNotice"
      class="flex-1 overflow-y-auto"
    >
      <div
        class="flex flex-col"
        :class="hasVoted ? 'gap-6' : 'gap-10'"
      >
        <section v-if="isVotingClosed" class="px-4">
          <div class="closed-banner" role="status" aria-live="polite">
            <h3>Votazioni chiuse</h3>
            <p>Grazie per aver partecipato! Ti aspettiamo alla prossima partita al palazzetto.</p>
          </div>
        </section>
        <section v-if="showVoteSummary" class="px-4">
          <div class="vote-summary" role="status" aria-live="polite">
            <div class="vote-summary__content">
              <p class="vote-summary__eyebrow">Hai votato!</p>
              <h3 class="vote-summary__title">Conserva il tuo codice per l'estrazione</h3>
              <p class="vote-summary__code" aria-label="Codice di voto">
                Codice: <span>{{ ticketCode }}</span>
              </p>
              <p class="vote-summary__hint">
                Mostra questo codice e il QR allo staff in caso di estrazione del premio.
              </p>
              <p v-if="ticketLoadError" class="vote-summary__error">{{ ticketLoadError }}</p>
            </div>
            <div class="vote-summary__qr" aria-hidden="true">
              <div v-if="isTicketLoading" class="vote-summary__qr-loader">
                <span class="qr-loader"></span>
              </div>
              <img
                v-else-if="ticketQrUrl"
                :src="ticketQrUrl"
                alt="QR code"
              />
              <div v-else class="vote-summary__qr-placeholder">QR non disponibile</div>
            </div>
          </div>
        </section>
        <section v-if="!hasVoted" class="px-4">
          <div class="mb-6 text-center">
            <h2 class="text-lg font-semibold uppercase tracking-[0.1em] text-slate-200">
              {{ eventTitle }}
            </h2>
            <p v-if="!isEventUpcoming" class="mt-2 text-sm text-slate-300">
              Tocca la card del tuo giocatore preferito per assegnarli il voto
            </p>
            <p v-else class="mt-2 text-sm text-slate-300">
              La votazione sarà disponibile all'inizio della partita.
            </p>
          </div>
          <div v-if="fieldPlayers.length" class="relative h-[95svh]">
            <VolleyCourt
              class="block h-full w-full"
              :players="fieldPlayers"
              :card-size="cardSize"
              :selected-player-id="votedPlayerId"
              :disable-votes="disableVotes"
              :is-voting="isVoting"
              @select="openPlayerModal"
            />
          </div>
          <p
            v-else-if="isLoadingPlayers"
            class="players-message"
          >
            Caricamento dei giocatori in corso…
          </p>
          <p
            v-else-if="playersError"
            class="players-message error"
          >
            {{ playersError }}
          </p>
          <p v-else class="players-message">
            I giocatori non sono ancora stati configurati. Torna più tardi!
          </p>
        </section>
        <section v-else class="px-4">
          <div class="after-vote-panel">
            <h3>{{ eventTitle }}</h3>
            <p>
              Hai già espresso il tuo voto per questa partita. Conserva il codice mostrato in alto e attendi l'estrazione dei premi.
            </p>
          </div>
        </section>

        <SelfieMvpSection
          v-if="currentEventId && (hasVoted || isCheckingVoteStatus)"
          :class="['px-4', hasVoted ? 'pt-0' : '']"
          :event-id="currentEventId"
          :enabled="hasVoted"
          :loading-status="isCheckingVoteStatus"
          :compact="hasVoted"
          @selfie-submitted="handleSelfieSubmitted"
        />

        <ReactionTestSection
          v-if="currentEventId && hasVoted"
          class="mt-8"
          :event-id="currentEventId"
          :enabled="hasVoted"
        />

        <section v-if="sponsors.length" class="px-4">
          <div
            class="relative overflow-hidden rounded-[2.25rem] border border-slate-700/40 bg-gradient-to-br from-slate-900 via-slate-800 to-slate-950 shadow-[0_26px_52px_rgba(8,15,28,0.55)]"
            aria-labelledby="sponsor-title"
          >
            <div class="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top_left,_rgba(148,163,184,0.18),_transparent_55%)]"></div>
            <div class="pointer-events-none absolute inset-0 bg-[linear-gradient(145deg,_rgba(30,41,59,0.45),_transparent_60%)] mix-blend-screen"></div>

            <div class="relative flex h-full flex-col">
              <header class="px-6 pt-6 pb-4">
                <p
                  id="sponsor-title"
                  class="text-xs font-semibold uppercase tracking-[0.45em] text-slate-300"
                >
                  Sponsor
                </p>
              </header>

              <div class="flex-1 px-6 pb-6">
                <div class="grid h-full grid-cols-2 grid-rows-2 gap-4">
                  <template v-for="sponsor in sponsors" :key="sponsor.id">
                    <a
                      v-if="sponsor.link"
                      class="group relative flex items-center justify-center overflow-hidden rounded-3xl border border-white/10 bg-slate-900/40 shadow-[0_16px_32px_rgba(8,15,28,0.45)]"
                      :href="sponsor.link"
                      target="_blank"
                      rel="noopener noreferrer"
                      :aria-label="sponsor.name"
                      @click="handleSponsorClick(sponsor)"
                    >
                      <div class="absolute inset-0 bg-gradient-to-br from-white/5 via-transparent to-white/10 opacity-0 transition-opacity duration-300 group-hover:opacity-100"></div>
                      <img
                        :src="sponsor.image"
                        :alt="sponsor.name"
                        class="relative h-full w-full object-cover"
                      />
                      <div class="pointer-events-none absolute inset-x-0 bottom-0 bg-gradient-to-t from-slate-950/85 via-slate-950/25 to-transparent px-4 pb-4 pt-8">
                        <p class="text-xs font-medium uppercase tracking-[0.25em] text-slate-200 text-center">
                          {{ sponsor.name }}
                        </p>
                      </div>
                    </a>
                    <div
                      v-else
                      class="group relative flex items-center justify-center overflow-hidden rounded-3xl border border-white/10 bg-slate-900/40 shadow-[0_16px_32px_rgba(8,15,28,0.45)]"
                      :aria-label="sponsor.name"
                    >
                      <div class="absolute inset-0 bg-gradient-to-br from-white/5 via-transparent to-white/10 opacity-0 transition-opacity duration-300 group-hover:opacity-100"></div>
                      <img
                        :src="sponsor.image"
                        :alt="sponsor.name"
                        class="relative h-full w-full object-cover"
                      />
                      <div class="pointer-events-none absolute inset-x-0 bottom-0 bg-gradient-to-t from-slate-950/85 via-slate-950/25 to-transparent px-4 pb-4 pt-8">
                        <p class="text-xs font-medium uppercase tracking-[0.25em] text-slate-200 text-center">
                          {{ sponsor.name }}
                        </p>
                      </div>
                    </div>
                  </template>
                </div>
              </div>
            </div>
          </div>
        </section>

        <section v-if="currentEventId" class="px-4">
          <div class="vote-counter" role="status" aria-live="polite">
            <div class="vote-counter__header">
              <p class="vote-counter__title">Totale voti registrati</p>
              <span
                v-if="isVoteTotalLoading"
                class="vote-counter__spinner"
                aria-hidden="true"
              ></span>
            </div>
            <p class="vote-counter__value">{{ formattedVoteTotal }}</p>
            <p v-if="voteTotalError" class="vote-counter__message error">
              {{ voteTotalError }}
            </p>
            <p v-else class="vote-counter__message">
              Aggiornamento automatico ogni pochi secondi
            </p>
          </div>
        </section>

        <p v-if="errorMessage" class="px-4 pb-6 text-center text-sm text-rose-400">
          {{ errorMessage }}
        </p>
      </div>
    </main>

    <div
      v-else
      class="flex flex-1 items-center justify-center px-6 py-12 text-center"
    >
      <div class="inactive-panel">
        <template v-if="isCheckingActiveEvent">
          <h2 class="text-2xl font-semibold uppercase tracking-[0.2em] text-slate-100">
            Verifica evento in corso…
          </h2>
          <p class="mt-4 text-base text-slate-300">
            Stiamo controllando se è disponibile una partita su cui votare.
          </p>
        </template>
        <template v-else>
          <h2 class="text-2xl font-semibold uppercase tracking-[0.2em] text-slate-100">
            Nessuna partita in corso
          </h2>
          <p class="mt-4 text-base text-slate-300">
            Attendi la prossima partita per votare il tuo MVP. Ti aspettiamo al palazzetto!
          </p>
        </template>
      </div>
    </div>

    <transition name="fade">
      <div
        v-if="!showInactiveNotice && isEventUpcoming"
        class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/85 px-6 py-10"
        role="dialog"
        aria-modal="true"
        aria-labelledby="countdown-title"
      >
        <div class="countdown-dialog">
          <p id="countdown-title" class="countdown-dialog__title">La votazione inizierà a breve</p>
          <p class="countdown-dialog__subtitle">Il voto sarà disponibile tra</p>
          <p class="countdown-timer">{{ countdownLabel }}</p>
          <p v-if="countdownDaysLabel" class="countdown-dialog__details">{{ countdownDaysLabel }}</p>
          <p v-if="countdownStartTimeLabel" class="countdown-dialog__details">
            Inizio previsto: {{ countdownStartTimeLabel }}
          </p>
        </div>
      </div>
    </transition>

    <transition name="fade">
      <div
        v-if="!showInactiveNotice && isModalOpen"
        class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 px-6 py-10"
      >
        <button class="absolute inset-0" type="button" @click="closeModal" aria-label="Chiudi"></button>
        <div
          class="relative z-10 w-full max-w-xs rounded-[2.25rem] border border-white/10 bg-slate-900/95 px-6 py-7 text-center shadow-[0_30px_60px_rgba(15,23,42,0.6)]"
        >
        <div class="flex justify-center">
          <PlayerCard
            v-if="pendingPlayer"
            :player="pendingPlayer"
            :card-size="cardSize * 1.3"
            :is-selected="votedPlayerId === pendingPlayer.id"
            :disabled="true"
          />
        </div>
          <div class="mt-6 flex flex-col gap-3">
            <button
              class="w-full rounded-full bg-yellow-400 px-4 py-3 text-sm font-semibold uppercase tracking-[0.35em] text-slate-900 transition-colors duration-200 hover:bg-yellow-300 disabled:cursor-not-allowed disabled:opacity-70"
              type="button"
              :disabled="isVoting || !pendingPlayer || votedPlayerId === pendingPlayer.id"
              @click="confirmVote"
            >
              {{ modalActionLabel }}
            </button>
            <button
              class="w-full rounded-full border border-white/15 px-4 py-3 text-sm font-semibold uppercase tracking-[0.3em] text-slate-200 transition-colors duration-200 hover:bg-white/10"
              type="button"
              @click="closeModal"
            >
              Annulla
            </button>
          </div>
        </div>
      </div>
    </transition>

    <transition name="fade">
      <div
        v-if="!showInactiveNotice && showTicketModal"
        class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 px-6 py-10"
      >
        <button class="absolute inset-0" type="button" @click="closeTicketModal" aria-label="Chiudi"></button>
        <div
          class="relative z-10 w-full max-w-sm rounded-[2.25rem] border border-white/10 bg-slate-900/95 px-6 py-7 text-center shadow-[0_30px_60px_rgba(15,23,42,0.6)]"
        >
          <h3 class="text-lg font-semibold uppercase tracking-[0.35em] text-slate-100">Voto registrato</h3>
          <p class="mt-3 text-sm text-slate-300">
            Fai subito uno screenshot di questa pagina e conservalo.
              Attendi la fine della partita per l'estrazione dei premi e mostra lo screenshot allo staff nel caso in cui venga estratto il tuo codice.
          </p>
          <div class="important-notice" role="alert">
            <p class="font-semibold uppercase tracking-[0.25em] text-yellow-300">Importante</p>
            <p class="mt-2 text-sm leading-relaxed text-slate-100">
              SENZA LO SCREENSHOT IL PREMIO NON POTRA' ESSERE ASSEGNATO.
            </p>
          </div>
          <div class="mt-5 flex flex-col items-center gap-2 text-lg text-slate-200">
            <p class="font-bold tracking-[0.2em]">Codice: {{ ticketCode }}</p>
          </div>
          <div
            v-if="isTicketLoading"
            class="mt-6 flex flex-col items-center gap-3 text-slate-200"
            role="status"
            aria-live="polite"
          >
            <span class="qr-loader"></span>
            <p class="text-sm font-semibold uppercase tracking-[0.3em] text-slate-300">Attendi…</p>
          </div>
          <p v-if="ticketLoadError" class="mt-4 text-sm text-rose-300">
            {{ ticketLoadError }}
          </p>
          <img
            v-if="ticketQrUrl"
            :src="ticketQrUrl"
            alt="QR code"
            class="mx-auto mt-6 h-40 w-40 rounded-3xl border border-white/10 bg-white p-3"
            :class="{ hidden: isTicketLoading }"
            @load="handleQrLoaded"
            @error="handleQrError"
          />
          <button
            class="mt-7 w-full rounded-full bg-yellow-400 px-4 py-3 text-sm font-semibold uppercase tracking-[0.35em] text-slate-900 transition-colors duration-200 hover:bg-yellow-300"
            type="button"
            @click="closeTicketModal"
          >
            Chiudi
          </button>
        </div>
      </div>
    </transition>

    <transition name="fade">
      <div
        v-if="!showInactiveNotice && showAlreadyVotedModal"
        class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 px-6 py-10"
      >
        <button class="absolute inset-0" type="button" @click="closeAlreadyVotedModal" aria-label="Chiudi"></button>
        <div
          class="relative z-10 w-full max-w-sm rounded-[2.25rem] border border-white/10 bg-slate-900/95 px-6 py-7 text-center shadow-[0_30px_60px_rgba(15,23,42,0.6)]"
        >
          <h3 class="text-lg font-semibold uppercase tracking-[0.35em] text-slate-100">Hai già votato</h3>
          <p class="mt-3 text-sm text-slate-300">
            Puoi esprimere il tuo voto una sola volta per partita. Attendi la fine della gara per scoprire l'estrazione dei premi.
          </p>
          <button
            class="mt-7 w-full rounded-full bg-yellow-400 px-4 py-3 text-sm font-semibold uppercase tracking-[0.35em] text-slate-900 transition-colors duration-200 hover:bg-yellow-300"
            type="button"
            @click="closeAlreadyVotedModal"
          >
            Ho capito
          </button>
        </div>
      </div>
    </transition>
  </div>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

.closed-banner {
  border-radius: 2rem;
  border: 1px solid rgba(148, 163, 184, 0.25);
  background: rgba(15, 23, 42, 0.75);
  padding: 1.75rem 1.5rem;
  text-align: center;
  box-shadow: 0 24px 48px rgba(15, 23, 42, 0.45);
}

.closed-banner h3 {
  margin: 0 0 0.75rem;
  font-size: 1.1rem;
  letter-spacing: 0.2em;
  text-transform: uppercase;
  color: #fbbf24;
}

.closed-banner p {
  margin: 0;
  font-size: 0.95rem;
  color: #e2e8f0;
}

.vote-summary {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  padding: 1.75rem 1.5rem;
  border-radius: 2rem;
  border: 1px solid rgba(148, 163, 184, 0.35);
  background: linear-gradient(145deg, rgba(15, 23, 42, 0.9), rgba(30, 41, 59, 0.75));
  box-shadow: 0 28px 52px rgba(15, 23, 42, 0.55);
}

.vote-summary__content {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.vote-summary__eyebrow {
  margin: 0;
  font-size: 0.75rem;
  letter-spacing: 0.45em;
  text-transform: uppercase;
  color: #facc15;
}

.vote-summary__title {
  margin: 0;
  font-size: 1.05rem;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: #f8fafc;
}

.vote-summary__code {
  margin: 0.25rem 0 0;
  font-size: 1.2rem;
  font-weight: 700;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: #f8fafc;
}

.vote-summary__code span {
  color: #38bdf8;
}

.vote-summary__hint {
  margin: 0.5rem 0 0;
  font-size: 0.9rem;
  color: rgba(226, 232, 240, 0.85);
  line-height: 1.5;
}

.vote-summary__error {
  margin: 0.5rem 0 0;
  font-size: 0.85rem;
  color: #fecaca;
}

.vote-summary__qr {
  display: flex;
  align-items: center;
  justify-content: center;
}

.vote-summary__qr img {
  width: 112px;
  height: 112px;
  border-radius: 1.5rem;
  border: 1px solid rgba(148, 163, 184, 0.25);
  background: #fff;
  padding: 0.75rem;
}

.vote-summary__qr-loader {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 112px;
  height: 112px;
  border-radius: 1.5rem;
  border: 1px dashed rgba(148, 163, 184, 0.35);
}

.vote-summary__qr-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 112px;
  height: 112px;
  border-radius: 1.5rem;
  border: 1px solid rgba(148, 163, 184, 0.2);
  color: rgba(148, 163, 184, 0.75);
  font-size: 0.75rem;
  letter-spacing: 0.2em;
  text-transform: uppercase;
  text-align: center;
  padding: 0.75rem;
}

.vote-summary__qr-loader .qr-loader {
  width: 2.5rem;
  height: 2.5rem;
}

.after-vote-panel {
  padding: 1.5rem 1.5rem;
  border-radius: 1.75rem;
  border: 1px solid rgba(148, 163, 184, 0.3);
  background: rgba(15, 23, 42, 0.6);
  box-shadow: 0 24px 48px rgba(15, 23, 42, 0.45);
  text-align: center;
}

.after-vote-panel h3 {
  margin: 0;
  font-size: 1rem;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: #f8fafc;
}

.after-vote-panel p {
  margin: 0.75rem 0 0;
  font-size: 0.9rem;
  color: rgba(226, 232, 240, 0.85);
}

@media (min-width: 640px) {
  .vote-summary {
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
    gap: 2rem;
  }

  .vote-summary__content {
    flex: 1;
  }

  .vote-summary__qr {
    flex-shrink: 0;
  }
}

.inactive-panel {
  width: 100%;
  max-width: 480px;
  padding: 2.5rem 2rem;
  border-radius: 2rem;
  border: 1px solid rgba(148, 163, 184, 0.25);
  background: rgba(15, 23, 42, 0.65);
  box-shadow: 0 30px 60px rgba(15, 23, 42, 0.55);
}

.players-message {
  margin: 2rem auto;
  max-width: 420px;
  padding: 1.5rem;
  border-radius: 1.5rem;
  border: 1px solid rgba(148, 163, 184, 0.25);
  background: rgba(15, 23, 42, 0.55);
  text-align: center;
  font-size: 0.95rem;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: #e2e8f0;
}

.players-message.error {
  border-color: rgba(248, 113, 113, 0.35);
  background: rgba(127, 29, 29, 0.45);
  color: #fee2e2;
}

.important-notice {
  margin-top: 1.75rem;
  padding: 1.5rem 1.25rem;
  border-radius: 1.75rem;
  border: 1px solid rgba(250, 204, 21, 0.5);
  background: rgba(30, 64, 175, 0.35);
  box-shadow: 0 20px 40px rgba(15, 23, 42, 0.45);
  text-align: center;
}

.inactive-panel h2 {
  margin: 0;
}

.inactive-panel p {
  margin: 0;
  line-height: 1.6;
}

.qr-loader {
  width: 3rem;
  height: 3rem;
  border-radius: 9999px;
  border: 4px solid rgba(148, 163, 184, 0.25);
  border-top-color: #fbbf24;
  animation: qr-spin 0.9s linear infinite;
}

.countdown-dialog {
  width: 100%;
  max-width: 480px;
  padding: 2.75rem 2.25rem;
  border-radius: 2.5rem;
  border: 1px solid rgba(148, 163, 184, 0.25);
  background: rgba(15, 23, 42, 0.9);
  box-shadow: 0 35px 60px rgba(15, 23, 42, 0.6);
  text-align: center;
}

.countdown-dialog__title {
  margin: 0;
  font-size: 1.1rem;
  letter-spacing: 0.3em;
  text-transform: uppercase;
  color: #fbbf24;
}

.countdown-dialog__subtitle {
  margin: 1rem 0 0;
  font-size: 0.9rem;
  letter-spacing: 0.2em;
  text-transform: uppercase;
  color: #cbd5f5;
}

.countdown-timer {
  margin: 1.75rem 0 1rem;
  font-size: clamp(2.75rem, 8vw, 3.75rem);
  font-weight: 700;
  letter-spacing: 0.14em;
  color: #38bdf8;
  text-shadow: 0 18px 36px rgba(56, 189, 248, 0.45);
}

.countdown-dialog__details {
  margin: 0.5rem 0 0;
  font-size: 0.95rem;
  letter-spacing: 0.08em;
  color: #e2e8f0;
}

.vote-counter {
  margin-top: -0.5rem;
  margin-bottom: 1rem;
  padding: 1.75rem 1.5rem;
  border-radius: 2rem;
  border: 1px solid rgba(148, 163, 184, 0.3);
  background: linear-gradient(145deg, rgba(15, 23, 42, 0.9), rgba(30, 41, 59, 0.65));
  box-shadow: 0 28px 48px rgba(15, 23, 42, 0.5);
}

.vote-counter__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.vote-counter__title {
  margin: 0;
  font-size: 0.75rem;
  letter-spacing: 0.35em;
  text-transform: uppercase;
  color: #e2e8f0;
}

.vote-counter__spinner {
  width: 1.5rem;
  height: 1.5rem;
  border-radius: 9999px;
  border: 3px solid rgba(148, 163, 184, 0.25);
  border-top-color: #38bdf8;
  animation: counter-spin 0.8s linear infinite;
}

.vote-counter__value {
  margin: 1rem 0 0;
  font-size: clamp(2.5rem, 6vw, 3.25rem);
  font-weight: 700;
  letter-spacing: 0.08em;
  color: #fbbf24;
  text-shadow: 0 12px 24px rgba(251, 191, 36, 0.35);
}

.vote-counter__message {
  margin: 0.75rem 0 0;
  font-size: 0.85rem;
  color: #cbd5f5;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.vote-counter__message.error {
  color: #fecaca;
}

@keyframes qr-spin {
  to {
    transform: rotate(360deg);
  }
}

@keyframes counter-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
