<template>
  <SvgSponsorBoard
    v-for="side in sides"
    :key="side"
    :position="side"
    :game-id="gameId"
    :sponsor-id="sponsors[side]?.sponsorId"
    :creative-id="sponsors[side]?.creativeId"
    :organization-id="sponsors[side]?.organizationId"
    :image-url="sponsors[side]?.imageUrl"
    :x="side === 'left' ? layout.inset : 360 - layout.inset - layout.width"
    :y="layout.y"
    :width="layout.width"
    :height="layout.height"
  />
</template>

<script setup>
import { computed } from 'vue';
import SvgSponsorBoard from './SvgSponsorBoard.vue';

const props = defineProps({
  gameId: { type: String, required: true },
  sponsors: { type: Object, default: () => ({}) },
});

// Coordinate nel viewBox 360 x 560, specchiate rispetto al centro del campo.
const layouts = {
  basketball: { inset: 12, y: 86, width: 80, height: 34 },
  playmaker: { inset: 14, y: 116, width: 102, height: 34 },
  crossover: { inset: 14, y: 103, width: 102, height: 34 },
};
const sides = ['left', 'right'];
const layout = computed(() => layouts[props.gameId]);
</script>
