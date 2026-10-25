<template>
  <TapChallenge v-bind="$props" variant="volleyball" @claim="emit('claim', $event)" @exit="emit('exit')" @round-started="emit('round-started', $event)" @round-completed="emit('round-completed', $event)" />
</template>

<script setup>
import TapChallenge from './TapChallenge.vue';
defineProps({
  eventId: { type: Number, default: 0 }, cooldownSeconds: { type: Number, default: 0 },
  completionReward: { type: Number, default: 8 }, claiming: { type: Boolean, default: false },
});
const emit = defineEmits(['claim', 'exit', 'round-started', 'round-completed']);
</script>
