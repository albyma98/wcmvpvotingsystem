<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import VolleyCourt from './VolleyCourt.vue';
import PlayerCard from './PlayerCard.vue';
import { vote } from '../api';

const props = defineProps({
  eventId: {
    type: Number,
    default: 1,
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

const sponsors = [
  {
    id: 1,
    name: 'WearingCash',
    image: 'https://wearingcash.it/cdn/shop/files/2.png?v=1741121739&width=255',
  },
  {
    id: 2,
    name: 'BluWave Partner',
    image: 'https://via.placeholder.com/320x180.png?text=BluWave+Partner',
  },
  {
    id: 3,
    name: 'Energia Italia',
    image: 'https://via.placeholder.com/320x180.png?text=Energia+Italia',
  },
  {
    id: 4,
    name: 'Tech Volley Lab',
    image: 'https://via.placeholder.com/320x180.png?text=Tech+Volley+Lab',
  },
];

const votedPlayerId = ref(null);
const isVoting = ref(false);
const cardSize = ref(88);
const errorMessage = ref('');
const pendingPlayer = ref(null);

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
});

onBeforeUnmount(() => {
  window.removeEventListener('resize', updateCardSize);
});

const disableVotes = computed(() => Boolean(votedPlayerId.value));

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

const voteForPlayer = async (player) => {
  if (isVoting.value || (votedPlayerId.value && votedPlayerId.value !== player.id)) {
    return;
  }

  if (votedPlayerId.value === player.id) {
    return;
  }

  errorMessage.value = '';
  isVoting.value = true;

  try {
    const response = await vote({ eventId: props.eventId, playerId: player.id });
    if (response?.ok) {
      votedPlayerId.value = player.id;
      pendingPlayer.value = null;
    } else {
      errorMessage.value = 'Non è stato possibile registrare il voto. Riprova.';
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
</script>

<template>
  <div class="min-h-screen bg-gradient-to-b from-slate-950 via-slate-900 to-slate-950 text-slate-100 flex flex-col">
    <main class="flex-1 overflow-y-auto">
      <div class="flex flex-col gap-10">
        <section class="px-4">
          <div class="mb-6 text-center">
            <h2 class="text-lg font-semibold uppercase tracking-[0.3em] text-slate-200">
              Team 1 - Team 2
            </h2>
            <p class="mt-2 text-sm text-slate-300">
              Tocca la card del tuo giocatore preferito per assegnarli il voto
            </p>
          </div>
          <div class="relative h-[100svh]">
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
                <div class="grid h-full grid-cols-2 grid-rows-2 gap-4">
                  <article
                    v-for="sponsor in sponsors"
                    :key="sponsor.id"
                    class="group relative flex items-center justify-center overflow-hidden rounded-3xl border border-white/10 bg-slate-900/40 shadow-[0_16px_32px_rgba(8,15,28,0.45)]"
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
                  </article>
                </div>
              </div>
            </div>
          </div>
        </section>

        <p v-if="errorMessage" class="px-4 pb-6 text-center text-sm text-rose-400">
          {{ errorMessage }}
        </p>
      </div>
    </main>

    <transition name="fade">
      <div
        v-if="isModalOpen"
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
</style>
