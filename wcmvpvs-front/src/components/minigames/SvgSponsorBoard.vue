<template>
  <g
    v-if="enabled"
    aria-hidden="true"
    pointer-events="none"
    :data-sponsor-id="sponsorId"
    :data-creative-id="creativeId"
    :data-organization-id="organizationId"
    :data-game-id="gameId"
    :data-placement="`game_wall_${position}`"
    :transform="`translate(${x} ${y})`"
  >
    <defs v-if="variant === 'led'">
      <linearGradient :id="panelId" x2="0%" y2="100%">
        <stop stop-color="#273945" />
        <stop offset="1" stop-color="#101a24" />
      </linearGradient>
    </defs>
    <rect y="2" :width="width" :height="height" rx="3" fill="#040910" opacity=".6" />
    <rect
      v-if="variant === 'led'"
      :width="width"
      :height="height"
      rx="3"
      fill="none"
      stroke="#94d7e9"
      stroke-width="5"
      stroke-opacity=".06"
    />
    <rect
      :width="width"
      :height="height"
      rx="3"
      :fill="variant === 'led' ? `url(#${panelId})` : '#16232d'"
      stroke="#9dd8e6"
      stroke-width=".8"
      stroke-opacity=".42"
    />
    <image
      v-if="imageBounds"
      :key="imageUrl"
      :href="imageUrl"
      v-bind="imageBounds"
      preserveAspectRatio="xMidYMid meet"
      @error="imageRatio = 0"
    />
    <rect
      v-else
      x="3"
      y="3"
      :width="width - 6"
      :height="height - 6"
      rx="1.5"
      fill="#07121b"
      fill-opacity=".32"
    />
  </g>
</template>

<script setup>
import { computed, ref, useId, watch } from 'vue';

const props = defineProps({
  sponsorId: { type: [Number, String], default: null },
  creativeId: { type: [Number, String], default: null },
  organizationId: { type: [Number, String], default: null },
  gameId: { type: String, required: true },
  imageUrl: { type: String, default: '' },
  position: { type: String, required: true, validator: (value) => ['left', 'right'].includes(value) },
  x: { type: Number, required: true },
  y: { type: Number, required: true },
  width: { type: Number, required: true },
  height: { type: Number, required: true },
  variant: { type: String, default: 'led', validator: (value) => ['led', 'static'].includes(value) },
  enabled: { type: Boolean, default: true },
});

const panelId = `sponsor-board-${useId()}`;
const imageRatio = ref(0);
const imageBounds = computed(() => {
  if (!imageRatio.value) return null;
  const width = Math.min(props.width * .88, props.height * .8 * imageRatio.value);
  const height = width / imageRatio.value;
  return { x: (props.width - width) / 2, y: (props.height - height) / 2, width, height };
});

// Un SVG senza viewBox può adattarsi all'intero rettangolo di <image>.
watch([() => props.enabled, () => props.imageUrl], ([enabled, url], _, onCleanup) => {
  imageRatio.value = 0;
  if (!enabled || !url || typeof Image === 'undefined') return;
  const logo = new Image();
  logo.onload = () => {
    if (logo.naturalHeight) imageRatio.value = logo.naturalWidth / logo.naturalHeight;
  };
  onCleanup(() => { logo.onload = null; });
  logo.src = url;
}, { immediate: true });
</script>
