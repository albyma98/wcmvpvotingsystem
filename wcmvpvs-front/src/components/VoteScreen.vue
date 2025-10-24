<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import VolleyCourt from './VolleyCourt.vue';
import PlayerCard from './PlayerCard.vue';
import { apiClient, vote } from '../api';

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

const roster = [
  {
    id: 1,
    name: 'Matteo Paris',
    role: 'Opposto',
    number: 10,
    tier: 'gold',
    position: { x: 20, y: 14 },
    avatar: 'https://www.legavolley.it/Foto.aspx?Key=UTR-VIN-96&sqid=6733&heightImg=600',
  },
  {
    id: 2,
    name: 'Giuseppe Longo',
    role: 'Palleggiatore',
    number: 8,
    tier: 'gold',
    position: { x: 50, y: 14 },
    avatar: 'https://www.legavolley.it/Foto.aspx?Key=UTR-VIN-96&sqid=6733&heightImg=600',
  },
  {
    id: 3,
    name: 'Felice Sette',
    role: 'Centrale',
    number: 7,
    tier: 'silver',
    position: { x: 80, y: 14 },
    avatar: 'https://www.legavolley.it/Foto.aspx?Key=UTR-VIN-96&sqid=6733&heightImg=600',
  },
  {
    id: 4,
    name: 'Sebastiano Milan',
    role: 'Schiacciatore',
    number: 9,
    tier: 'gold',
    position: { x: 20, y: 32 },
    avatar: 'https://www.legavolley.it/Foto.aspx?Key=UTR-VIN-96&sqid=6733&heightImg=600',
  },
  {
    id: 5,
    name: 'Leonardo Carta',
    role: 'Centrale',
    number: 15,
    tier: 'silver',
    position: { x: 50, y: 32 },
    avatar: 'https://www.legavolley.it/Foto.aspx?Key=UTR-VIN-96&sqid=6733&heightImg=600',
  },
  {
    id: 6,
    name: 'Romolo Mariano',
    role: 'Schiacciatore',
    number: 3,
    tier: 'gold',
    position: { x: 80, y: 32 },
    avatar: 'https://www.legavolley.it/Foto.aspx?Key=UTR-VIN-96&sqid=6733&heightImg=600',
  },
  {
    id: 7,
    name: 'Alessio Santangelo',
    role: 'Libero',
    number: 1,
    tier: 'bronze',
    position: { x: 50, y: 50 },
    avatar: 'https://www.legavolley.it/Foto.aspx?Key=UTR-VIN-96&sqid=6733&heightImg=600',
  },
  {
    id: 8,
    name: 'Alberto Marra',
    role: 'Opposto',
    number: 5,
    tier: 'bronze',
    position: { x: 20, y: 68 },
    avatar: 'https://www.legavolley.it/Foto.aspx?Key=UTR-VIN-96&sqid=6733&heightImg=600',
  },
  {
    id: 9,
    name: 'Cristian Frumuselu',
    role: 'Centrale',
    number: 6,
    tier: 'bronze',
    position: { x: 50, y: 68 },
    avatar: 'https://www.legavolley.it/Foto.aspx?Key=UTR-VIN-96&sqid=6733&heightImg=600',
  },
  {
    id: 10,
    name: 'Vincenzo Utro',
    role: 'Palleggiatore',
    number: 33,
    tier: 'silver',
    position: { x: 80, y: 68 },
    avatar: 'https://www.legavolley.it/Foto.aspx?Key=UTR-VIN-96&sqid=6733&heightImg=600',
  },
  {
    id: 11,
    name: 'Francesco Pierri',
    role: 'Schiacciatore',
    number: 14,
    tier: 'bronze',
    position: { x: 20, y: 86 },
    avatar: 'https://www.legavolley.it/Foto.aspx?Key=UTR-VIN-96&sqid=6733&heightImg=600',
  },
  {
    id: 12,
    name: 'Alessandro Chinello',
    role: 'Libero',
    number: 13,
    tier: 'silver',
    position: { x: 50, y: 86 },
    avatar: 'https://www.legavolley.it/Foto.aspx?Key=UTR-VIN-96&sqid=6733&heightImg=600',
  },
  {
    id: 13,
    name: 'Sandi Persoglia',
    role: 'Opposto',
    number: 11,
    tier: 'bronze',
    position: { x: 80, y: 86 },
    avatar: 'https://www.legavolley.it/Foto.aspx?Key=UTR-VIN-96&sqid=6733&heightImg=600',
  },
];

const fieldPlayers = computed(() => roster);

const sponsors = ref([]);
const hasSponsors = computed(() => sponsors.value.length > 0);
const sponsorGridClass = computed(() => {
  const count = sponsors.value.length;
  if (count <= 1) {
    return 'grid-cols-1';
  }
  if (count === 2) {
    return 'grid-cols-1 sm:grid-cols-2';
  }
  if (count === 3) {
    return 'grid-cols-1 sm:grid-cols-2 xl:grid-cols-3';
  }
  return 'grid-cols-1 sm:grid-cols-2 xl:grid-cols-4';
});

const normalizeLink = (value) => {
  if (typeof value !== 'string') {
    return '';
  }
  const trimmed = value.trim();
  if (!trimmed) {
    return '';
  }
  if (/^https?:\/\//i.test(trimmed)) {
    return trimmed;
  }
  if (/^[a-z][a-z0-9+.-]*:/i.test(trimmed)) {
    return '';
  }
  return `https://${trimmed}`;
};

const guessSponsorName = (record, slot) => {
  const directName = typeof record?.name === 'string' ? record.name.trim() : '';
  if (directName) {
    return directName;
  }
  const title = typeof record?.title === 'string' ? record.title.trim() : '';
  if (title) {
    return title;
  }
  const link = normalizeLink(record?.link_url || record?.link);
  if (link && typeof window !== 'undefined') {
    try {
      const parsed = new URL(link, link.startsWith('http') ? undefined : window.location.origin);
      const host = parsed.hostname.replace(/^www\./i, '');
      if (host) {
        return host;
      }
    } catch (error) {
      // ignore malformed URLs
    }
  }
  return `Sponsor ${slot}`;
};

const normalizeSponsorRecord = (record, index) => {
  if (!record || typeof record !== 'object') {
    return null;
  }
  const rawImage = typeof record.image_data === 'string' ? record.image_data.trim() : '';
  if (!rawImage) {
    return null;
  }
  const rawSlot = Number.parseInt(record.slot ?? '', 10);
  const slot = Number.isFinite(rawSlot) && rawSlot > 0 ? rawSlot : index + 1;
  const link = normalizeLink(record.link_url);
  return {
    id: record.id ?? slot ?? index + 1,
    slot,
    name: guessSponsorName(record, slot),
    image: rawImage,
    link,
  };
};

const loadSponsors = async () => {
  try {
    const { data } = await apiClient.get('/sponsors');
    const normalized = Array.isArray(data) ? data : [];
    sponsors.value = normalized
      .map((record, index) => normalizeSponsorRecord(record, index))
      .filter(Boolean);
  } catch (error) {
    console.error('load sponsors error', error);
    sponsors.value = [];
  }
};

const votedPlayerId = ref(null);
const isVoting = ref(false);
const cardSize = ref(88);
const errorMessage = ref('');
const pendingPlayer = ref(null);
const showTicketModal = ref(false);
const ticketCode = ref('');
const ticketQrUrl = ref('');

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

watch(currentEventId, () => {
  votedPlayerId.value = null;
  pendingPlayer.value = null;
  errorMessage.value = '';
  showTicketModal.value = false;
  ticketCode.value = '';
  ticketQrUrl.value = '';
});

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
});

onBeforeUnmount(() => {
  window.removeEventListener('resize', updateCardSize);
});

const disableVotes = computed(
  () => Boolean(votedPlayerId.value) || showInactiveNotice.value || isCheckingActiveEvent.value
);

const openPlayerModal = (player) => {
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
  ticketCode.value = '';
  ticketQrUrl.value = '';
};

const voteForPlayer = async (player) => {
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

      const codeSource = voteResult.code || '';
      const qrSource = voteResult.qr_data || '';

      if (codeSource) {
        ticketCode.value = codeSource;
        ticketQrUrl.value = qrSource
          ? `https://api.qrserver.com/v1/create-qr-code/?size=180x180&data=${encodeURIComponent(qrSource)}`
          : '';
        showTicketModal.value = true;
      } else {
        console.warn('voteForPlayer: missing ticket data', response);
        errorMessage.value = 'Non siamo riusciti a generare il QR del ticket. Riprova.';
      }
    } else {
      errorMessage.value = "Non e stato possibile registrare il voto. Riprova.";
    }
  } catch (error) {
    console.error('vote error', error);
    errorMessage.value = 'Si e verificato un errore. Riprova tra qualche istante.';
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
</script>

<template>
  <div class="min-h-screen bg-gradient-to-b from-slate-950 via-slate-900 to-slate-950 text-slate-100 flex flex-col">
    <main
      v-if="!isCheckingActiveEvent && !showInactiveNotice"
      class="flex-1 overflow-y-auto"
    >
      <div class="flex flex-col gap-10">
        <section class="px-4">
          <div class="mb-6 text-center">
            <h2 class="text-lg font-semibold uppercase tracking-[0.1em] text-slate-200">
              {{ eventTitle }}
            </h2>
            <p class="mt-2 text-sm text-slate-300">
              Tocca la card del tuo giocatore preferito per assegnarli il voto
            </p>
          </div>
          <div class="relative h-[95svh]">
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
        </section>

        <section class="px-4">
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
                <div
                  v-if="hasSponsors"
                  class="grid h-full gap-4"
                  :class="sponsorGridClass"
                >
                  <article
                    v-for="sponsor in sponsors"
                    :key="sponsor.id"
                    class="group relative flex items-center justify-center overflow-hidden rounded-3xl border border-white/10 bg-slate-900/40 shadow-[0_16px_32px_rgba(8,15,28,0.45)]"
                  >
                    <div class="pointer-events-none absolute inset-0 bg-gradient-to-br from-white/5 via-transparent to-white/10 opacity-0 transition-opacity duration-300 group-hover:opacity-100"></div>
                    <a
                      v-if="sponsor.link"
                      class="absolute inset-0 z-10"
                      :href="sponsor.link"
                      target="_blank"
                      rel="noopener"
                      :aria-label="`Visita ${sponsor.name}`"
                    ></a>
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
                  </article>
                </div>
                <p v-else class="py-12 text-center text-sm text-slate-400">
                  Spazio sponsor disponibile.
                </p>
              </div>
            </div>
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
            Mostra questo codice allo staff per completare la registrazione.
          </p>
          <div class="mt-5 flex flex-col items-center gap-2 text-xs text-slate-200">
            <p class="font-semibold tracking-[0.2em]">Codice: {{ ticketCode }}</p>
          </div>
          <img
            v-if="ticketQrUrl"
            :src="ticketQrUrl"
            alt="QR code"
            class="mx-auto mt-6 h-40 w-40 rounded-3xl border border-white/10 bg-white p-3"
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

.inactive-panel {
  width: 100%;
  max-width: 480px;
  padding: 2.5rem 2rem;
  border-radius: 2rem;
  border: 1px solid rgba(148, 163, 184, 0.25);
  background: rgba(15, 23, 42, 0.65);
  box-shadow: 0 30px 60px rgba(15, 23, 42, 0.55);
}

.inactive-panel h2 {
  margin: 0;
}

.inactive-panel p {
  margin: 0;
  line-height: 1.6;
}
</style>
