<script setup>
import V2Icon from '../../admin-v2/components/shared/V2Icon.vue';

defineProps({ name: String, size: { type: [Number, String], default: 18 } });

const paths = {
  building: 'M6 22V4a2 2 0 0 1 2-2h8a2 2 0 0 1 2 2v18ZM2 22h20M10 6h4M10 10h4M10 14h4M10 22v-4h4v4',
  box: 'm12 3 9 5v8l-9 5-9-5V8Zm-9 5 9 5 9-5m-9 5v8m-4.5-15.5 9 5',
  file: 'M6 2h8l4 4v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2Zm8 0v5h5M8 12h6m-6 4h6',
  gamepad: 'M7 6h10c3 0 4 4 5 8s-1 6-4 4l-3-3H9l-3 3c-3 2-5 0-4-4s2-8 5-8Zm-1 5h4m-2-2v4m7-2h.01m3 1h.01',
  qr: 'M3 3h6v6H3zm12 0h6v6h-6zM3 15h6v6H3zm12 0h2v2h-2zm4 4h2v2h-2zm0-6h2m-8 8h2m-2-8h2',
  panel: 'M3 5h18v14H3zm6 0v14m7-10-3 3 3 3',
  'arrow-left': 'M20 12H4m6-6-6 6 6 6',
  'chevron-down': 'm6 9 6 6 6-6',
  'chevrons-up-down': 'm8 9 4-4 4 4m-8 6 4 4 4-4',
  'arrow-up': 'M12 19V5m-6 6 6-6 6 6',
  'arrow-down': 'M12 5v14m-6-6 6 6 6-6',
  copy: 'M8 8h12v12H8zM16 8V4H4v12h4',
  key: 'M16 15a5 5 0 1 0-5-5c0 .5.07 1 .2 1.4L3 19.6V22h4v-3h3v-3l2-2a5 5 0 0 0 4 1Zm1-6h.01',
  trash: 'M4 7h16M9 7V4h6v3M6 7l1 14h10l1-14m-8 4v6m4-6v6',
  edit: 'm13 6 5 5M4 20l5-1L21 7l-5-5L4 14z',
  chart: 'M3 3v18h18M7 15l4-5 4 3 5-7',
  activity: 'M2 12h5l3-7 4 14 3-7h5',
  calendar: 'M4 5h16v16H4zm0 5h16M8 3v4m8-4v4',
  inbox: 'M4 4h16l2 10v6H2v-6Zm-2 10h6l2 3h4l2-3h6',
  alert: 'M12 3 2 21h20L12 3Zm0 6v5m0 3h.01',
};
</script>

<template>
  <svg v-if="paths[name]" :width="size" :height="size" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
    <path :d="paths[name]" />
  </svg>
  <V2Icon v-else :name="name" :size="size" />
</template>
