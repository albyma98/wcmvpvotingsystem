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
    name: 'Luca Bianchi',
    role: 'Opposto',
    number: 10,
    tier: 'gold',
    zone: 'court',
    position: { x: 18, y: 24 },
    avatar: 'https://api.dicebear.com/7.x/adventurer/svg?seed=Luca%20Bianchi',
  },
  {
    id: 2,
    name: 'Marco Rossi',
    role: 'Palleggiatore',
    number: 5,
    tier: 'gold',
    zone: 'court',
    position: { x: 50, y: 20 },
    avatar: 'https://api.dicebear.com/7.x/adventurer/svg?seed=Marco%20Rossi',
  },
  {
    id: 3,
    name: 'Giovanni Esposito',
    role: 'Centrale',
    number: 8,
    tier: 'silver',
    zone: 'court',
    position: { x: 82, y: 24 },
    avatar: 'https://api.dicebear.com/7.x/adventurer/svg?seed=Giovanni%20Esposito',
  },
  {
    id: 4,
    name: 'Davide Ricci',
    role: 'Schiacciatore',
    number: 17,
    tier: 'gold',
    zone: 'court',
    position: { x: 26, y: 74 },
    avatar: 'https://api.dicebear.com/7.x/adventurer/svg?seed=Davide%20Ricci',
  },
  {
    id: 5,
    name: 'Matteo Sala',
    role: 'Centrale',
    number: 12,
    tier: 'silver',
    zone: 'court',
    position: { x: 50, y: 80 },
    avatar: 'https://api.dicebear.com/7.x/adventurer/svg?seed=Matteo%20Sala',
  },
  {
    id: 6,
    name: 'Stefano Neri',
    role: 'Schiacciatore',
    number: 14,
    tier: 'gold',
    zone: 'court',
    position: { x: 74, y: 74 },
    avatar: 'https://api.dicebear.com/7.x/adventurer/svg?seed=Stefano%20Neri',
  },
  {
    id: 7,
    name: 'Alessio Galli',
    role: 'Libero',
    number: 1,
    tier: 'bronze',
    zone: 'libero',
    position: { x: 50, y: 50 },
    libero: true,
    avatar: 'https://api.dicebear.com/7.x/adventurer/svg?seed=Alessio%20Galli',
  },
  {
    id: 8,
    name: 'Riccardo Leone',
    role: 'Opposto',
    number: 18,
    tier: 'silver',
    zone: 'bench',
    position: { x: 16, y: 30 },
    avatar: 'https://api.dicebear.com/7.x/adventurer/svg?seed=Riccardo%20Leone',
  },
  {
    id: 9,
    name: 'Paolo Greco',
    role: 'Centrale',
    number: 6,
    tier: 'bronze',
    zone: 'bench',
    position: { x: 50, y: 30 },
    avatar: 'https://api.dicebear.com/7.x/adventurer/svg?seed=Paolo%20Greco',
  },
  {
    id: 10,
    name: 'Andrea Vitale',
    role: 'Palleggiatore',
    number: 3,
    tier: 'silver',
    zone: 'bench',
    position: { x: 84, y: 30 },
    avatar: 'https://api.dicebear.com/7.x/adventurer/svg?seed=Andrea%20Vitale',
  },
  {
    id: 11,
    name: 'Fabio Conti',
    role: 'Schiacciatore',
    number: 19,
    tier: 'bronze',
    zone: 'bench',
    position: { x: 16, y: 70 },
    avatar: 'https://api.dicebear.com/7.x/adventurer/svg?seed=Fabio%20Conti',
  },
  {
    id: 12,
    name: 'Nicola Ferretti',
    role: 'Libero',
    number: 2,
    tier: 'silver',
    zone: 'bench',
    position: { x: 50, y: 70 },
    avatar: 'https://api.dicebear.com/7.x/adventurer/svg?seed=Nicola%20Ferretti',
  },
  {
    id: 13,
    name: 'Gabriele Costa',
    role: 'Opposto',
    number: 21,
    tier: 'bronze',
    zone: 'bench',
    position: { x: 84, y: 70 },
    avatar: 'https://api.dicebear.com/7.x/adventurer/svg?seed=Gabriele%20Costa',
  },
];

const fieldPlayers = computed(() => roster.filter((player) => player.zone === 'court'));
const liberoPlayer = computed(() => roster.find((player) => player.zone === 'libero'));
const benchPlayers = computed(() => roster.filter((player) => player.zone === 'bench'));

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

const playerPositionStyle = (player) => ({
  left: `${player.position.x}%`,
  top: `${player.position.y}%`,
  transform: 'translate(-50%, -50%)',
});

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
    <header class="px-6 pt-6 pb-3 text-center">
      <p class="uppercase tracking-[0.35em] text-xs text-slate-400">MVP Voting System</p>
      <h1 class="mt-2 text-2xl font-semibold tracking-wide">Volley MVP - Match Day</h1>
      <p class="mt-1 text-sm text-slate-300">Tocca la card del tuo giocatore preferito per assegnargli il voto.</p>
    </header>

    <main class="flex-1 flex flex-col gap-5 px-4 pb-6">
      <VolleyCourt
        :players="fieldPlayers"
        :card-size="cardSize"
        :selected-player-id="votedPlayerId"
        :disable-votes="disableVotes"
        :is-voting="isVoting"
        @select="openPlayerModal"
      />

      <section
        v-if="liberoPlayer"
        class="relative mx-auto w-full max-w-lg rounded-3xl border border-white/10 bg-slate-900/80 px-4 pt-5 pb-6 shadow-xl shadow-slate-900/60"
      >
        <h2 class="text-center text-xs font-semibold uppercase tracking-[0.35em] text-slate-400">Libero</h2>
        <div class="relative mt-4 w-full" :style="{ aspectRatio: '6 / 2' }">
          <div class="absolute inset-0">
            <div class="absolute" :style="playerPositionStyle(liberoPlayer)">
              <PlayerCard
                :player="liberoPlayer"
                :card-size="cardSize"
                :is-selected="votedPlayerId === liberoPlayer.id"
                :disabled="disableVotes && votedPlayerId !== liberoPlayer.id"
                :is-voting="isVoting"
                @select="openPlayerModal(liberoPlayer)"
              />
            </div>
          </div>
        </div>
      </section>

      <section
        class="relative mx-auto w-full max-w-lg rounded-3xl border border-white/10 bg-slate-900/80 px-4 pt-5 pb-6 shadow-xl shadow-slate-900/60"
      >
        <h2 class="text-center text-xs font-semibold uppercase tracking-[0.35em] text-slate-400">
          Riserve
        </h2>
        <div class="relative mt-4 w-full" :style="{ aspectRatio: '6 / 3' }">
          <div class="absolute inset-0">
            <div
              v-for="player in benchPlayers"
              :key="player.id"
              class="absolute"
              :style="playerPositionStyle(player)"
            >
              <PlayerCard
                :player="player"
                :card-size="cardSize"
                :is-selected="votedPlayerId === player.id"
                :disabled="disableVotes && votedPlayerId !== player.id"
                :is-voting="isVoting"
                @select="openPlayerModal(player)"
              />
            </div>
          </div>
        </div>
      </section>

      <p v-if="errorMessage" class="text-center text-sm text-rose-400">
        {{ errorMessage }}
      </p>
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
          <PlayerCard
            v-if="pendingPlayer"
            :player="pendingPlayer"
            :card-size="cardSize * 1.3"
            :is-selected="votedPlayerId === pendingPlayer.id"
            :disabled="true"
          />

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
