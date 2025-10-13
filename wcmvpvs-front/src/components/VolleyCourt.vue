<script setup>
import { computed } from 'vue';
import PlayerCard from './PlayerCard.vue';

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
  <section class="relative mx-auto w-full max-w-[min(100vw,480px)]" :style="{ aspectRatio: '3 / 4.2' }">
    <div
      class="absolute inset-0 overflow-hidden rounded-[2.75rem] border-4 border-white/10 bg-gradient-to-b from-pitch-light via-pitch-base to-pitch-dark shadow-pitch"
    >
      <div class="absolute inset-0 opacity-35 bg-court-grid"></div>
      <div class="absolute inset-x-[12%] top-1/2 border-t border-b border-white/25"></div>
      <div class="absolute inset-x-0 top-1/2 h-[18px] -translate-y-1/2">
        <div class="absolute inset-x-[8%] top-0 h-[4px] rounded-full bg-white/70 shadow-lg shadow-white/40"></div>
        <div class="absolute inset-x-[6%] top-1/2 h-[2px] -translate-y-1/2 bg-white/90"></div>
        <div class="absolute inset-x-[8%] bottom-0 h-[1px] bg-white/60"></div>
      </div>
      <div class="absolute inset-x-[12%] top-[25%] h-[2px] bg-white/20"></div>
      <div class="absolute inset-x-[12%] top-[75%] h-[2px] bg-white/20"></div>
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
