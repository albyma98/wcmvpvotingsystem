<script setup>
import { computed } from 'vue';
import PlayerCard from './PlayerCard.vue';
import teamLogo from '../assets/team-logo.svg';

const props = defineProps({
  players: {
    type: Array,
    default: () => [],
  },
  cardSize: {
    type: Number,
    default: 90,
  },
  selectedPlayerId: {
    type: Number,
    default: null,
  },
  disableVotes: {
    type: Boolean,
    default: false,
  },
  isVoting: {
    type: Boolean,
    default: false,
  },
});

const emits = defineEmits(['select']);

const positionStyle = computed(() => (player) => ({
  left: `${player.position.x}%`,
  top: `${player.position.y}%`,
  transform: 'translate(-50%, -50%)',
}));
</script>

<template>
  <section class="relative mx-auto h-full w-full overflow-hidden">
    <div
      class="absolute inset-0 overflow-hidden rounded-[2.75rem] border-4 border-[rgba(64,34,10,0.35)] bg-gradient-to-b from-court-light via-court-base to-court-dark shadow-court"
    >
      <div class="absolute inset-0 opacity-50 mix-blend-soft-light bg-court-wood-planks"></div>
      <div class="absolute inset-0 opacity-70 mix-blend-overlay bg-court-wood-grain"></div>
      <div class="absolute inset-0 flex items-center justify-center pointer-events-none">
        <img
          :src="teamLogo"
          alt="Team crest"
          class="h-[42%] opacity-[0.14] drop-shadow-[0_20px_35px_rgba(65,34,9,0.35)]"
        />
      </div>
      <div class="absolute inset-0 pointer-events-none opacity-[0.65] bg-[radial-gradient(circle_at_center,_rgba(0,0,0,0)_38%,_rgba(31,20,9,0.28)_100%)]"></div>
      <div class="absolute inset-x-[12%] top-1/2 border-t border-b border-white/25"></div>
      <div class="absolute inset-x-0 top-1/2 h-[18px] -translate-y-1/2">
        <div class="absolute inset-x-[8%] top-0 h-[4px] rounded-full bg-white/80 shadow-lg shadow-white/30"></div>
        <div class="absolute inset-x-[6%] top-1/2 h-[2px] -translate-y-1/2 bg-white/90"></div>
        <div class="absolute inset-x-[8%] bottom-0 h-[1px] bg-white/70"></div>
      </div>
      <div class="absolute inset-x-[12%] top-[25%] h-[2px] bg-white/30"></div>
      <div class="absolute inset-x-[12%] top-[75%] h-[2px] bg-white/30"></div>
    </div>

    <div class="absolute inset-0">
      <div
        v-for="player in players"
        :key="player.id"
        class="absolute"
        :style="positionStyle(player)"
      >
        <PlayerCard
          :player="player"
          :card-size="cardSize"
          :is-selected="selectedPlayerId === player.id"
          :disabled="disableVotes && selectedPlayerId !== player.id"
          :is-voting="isVoting"
          @select="() => emits('select', player)"
        />
      </div>
    </div>
  </section>
</template>
