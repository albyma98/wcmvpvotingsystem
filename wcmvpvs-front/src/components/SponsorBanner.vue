<template>
  <component :is="tag" v-sponsor-placement="{ sponsor, placement, enabled, clickable, fullDuration, creativeVariant }">
    <slot />
  </component>
</template>

<script setup>
import { useSponsorPlacement } from '../useSponsorTracking.js';

defineProps({
  tag: { type: String, default: 'div' },
  sponsor: { type: Object, required: true },
  placement: { type: String, required: true },
  enabled: { type: Boolean, default: true },
  clickable: { type: Boolean, default: false },
  fullDuration: { type: Number, default: 0 },
  creativeVariant: { type: String, default: '' },
});
const vSponsorPlacement = useSponsorPlacement();
</script>
