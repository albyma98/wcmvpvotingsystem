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
    position: { x: 20, y: 58 },
  },
  {
    id: 2,
    name: 'Marco Rossi',
    role: 'Palleggiatore',
    number: 5,
    tier: 'gold',
    zone: 'court',
    position: { x: 50, y: 52 },
  },
  {
    id: 3,
    name: 'Giovanni Esposito',
    role: 'Centrale',
    number: 8,
    tier: 'silver',
    zone: 'court',
    position: { x: 80, y: 58 },
  },
  {
    id: 4,
    name: 'Davide Ricci',
    role: 'Schiacciatore',
    number: 17,
    tier: 'gold',
    zone: 'court',
    position: { x: 28, y: 72 },
  },
  {
    id: 5,
    name: 'Matteo Sala',
    role: 'Centrale',
    number: 12,
    tier: 'silver',
    zone: 'court',
    position: { x: 50, y: 76 },
  },
  {
    id: 6,
    name: 'Stefano Neri',
    role: 'Schiacciatore',
    number: 14,
    tier: 'gold',
    zone: 'court',
    position: { x: 72, y: 72 },
  },
  {
    id: 7,
    name: 'Alessio Galli',
    role: 'Libero',
    number: 1,
    tier: 'bronze',
    zone: 'court',
    position: { x: 50, y: 88 },
    libero: true,
  },
  {
    id: 8,
    name: 'Riccardo Leone',
    role: 'Opposto',
    number: 18,
    tier: 'silver',
    zone: 'bench',
    position: { x: 16, y: 30 },
  },
  {
    id: 9,
    name: 'Paolo Greco',
    role: 'Centrale',
    number: 6,
    tier: 'bronze',
    zone: 'bench',
    position: { x: 50, y: 30 },
  },
  {
    id: 10,
    name: 'Andrea Vitale',
    role: 'Palleggiatore',
    number: 3,
    tier: 'silver',
    zone: 'bench',
    position: { x: 84, y: 30 },
  },
  {
    id: 11,
    name: 'Fabio Conti',
    role: 'Schiacciatore',
    number: 19,
    tier: 'bronze',
    zone: 'bench',
    position: { x: 16, y: 78 },
  },
  {
    id: 12,
    name: 'Nicola Ferretti',
    role: 'Libero',
    number: 2,
    tier: 'silver',
    zone: 'bench',
    position: { x: 50, y: 78 },
  },
  {
    id: 13,
    name: 'Gabriele Costa',
    role: 'Opposto',
    number: 21,
    tier: 'bronze',
    zone: 'bench',
    position: { x: 84, y: 78 },
  },
];

const fieldPlayers = computed(() => roster.filter((player) => player.zone === 'court'));
const benchPlayers = computed(() => roster.filter((player) => player.zone === 'bench'));

const votedPlayerId = ref(null);
const isVoting = ref(false);
const cardSize = ref(88);
const errorMessage = ref('');

const clamp = (value, min, max) => Math.min(Math.max(value, min), max);

const updateCardSize = () => {
  const width = window.innerWidth;
  const height = window.innerHeight;
  const sizeFromWidth = width / 4.6;
  const sizeFromHeight = height / 8.6;
  cardSize.value = clamp(Math.min(sizeFromWidth, sizeFromHeight), 64, 120);
};

onMounted(() => {
  updateCardSize();
  window.addEventListener('resize', updateCardSize, { passive: true });
});

onBeforeUnmount(() => {
  window.removeEventListener('resize', updateCardSize);
});

const disableVotes = computed(() => Boolean(votedPlayerId.value));

const benchPositionStyle = (player) => ({
  left: `${player.position.x}%`,
  top: `${player.position.y}%`,
  transform: 'translate(-50%, -50%)',
});

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
        @vote="voteForPlayer"
      />

      <section
        class="relative mx-auto w-full max-w-lg rounded-3xl border border-white/10 bg-slate-900/80 shadow-xl shadow-slate-900/60 px-4 pt-5 pb-6"
      >
        <h2 class="text-center text-xs font-semibold uppercase tracking-[0.35em] text-slate-400">
          Riserve
        </h2>
        <div class="relative mt-4 w-full" :style="{ aspectRatio: '6 / 3' }">
          <div
            v-for="player in benchPlayers"
            :key="player.id"
            class="absolute"
            :style="benchPositionStyle(player)"
          >
            <PlayerCard
              :player="player"
              :card-size="cardSize"
              :is-selected="votedPlayerId === player.id"
              :disabled="disableVotes && votedPlayerId !== player.id"
              :is-voting="isVoting"
              compact
              @vote="() => voteForPlayer(player)"
            />
          </div>
        </div>
      </section>

      <p v-if="errorMessage" class="text-center text-sm text-rose-400">
        {{ errorMessage }}
      </p>
    </main>
  </div>
</template>
