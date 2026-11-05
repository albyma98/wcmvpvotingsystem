<template>
  <svg :width="size" :height="size" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false">
    <path v-for="(path, index) in paths" :key="index" :d="path" />
  </svg>
</template>

<script setup>
import { computed } from 'vue';
const props = defineProps({ name: { type: String, default: 'dashboard' }, size: { type: [Number, String], default: 20 } });
const icons = {
  dashboard: ['M3 3h7v7H3zM14 3h7v7h-7zM3 14h7v7H3zM14 14h7v7h-7z'],
  events: ['M5 5h14a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2ZM16 3v4M8 3v4M3 11h18M8 15h2M14 15h2'],
  teams: ['M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2M16 4a4 4 0 0 1 0 8M22 21v-2a4 4 0 0 0-3-3.87M13 7a4 4 0 1 1-8 0 4 4 0 0 1 8 0Z'],
  players: ['M20 21v-2a6 6 0 0 0-6-6h-4a6 6 0 0 0-6 6v2M16 6a4 4 0 1 1-8 0 4 4 0 0 1 8 0Z'],
  sponsors: ['m12 3 2.8 5.7 6.3.9-4.5 4.4 1.1 6.3-5.7-3-5.7 3 1.1-6.3L3.2 9.6l6-.9L12 3Z'],
  partners: ['m8 13 3 3a2 2 0 0 0 3-3l-2-2M6 5l3-2 4 2 4-1 5 7-4 3-3 5a2 2 0 0 1-3 0l-1 1a2 2 0 0 1-3-1l-1-1a2 2 0 0 1-2-2l-3-4 4-7ZM9 5l-3 6 2 2 4-4 3 1'],
  coupons: ['M3 7a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2v3a2 2 0 0 0 0 4v3a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-3a2 2 0 0 0 0-4V7ZM15 5v3M15 11v2M15 16v3'],
  results: ['M4 20h16M7 16V9M12 16V4M17 16v-5'],
  analytics: ['M3 3v18h18M7 14l4-4 4 2 5-7'],
  marketing: ['m3 10 16-6v16L3 14v-4ZM6 15l2 6h4l-2-5M22 9v6'],
  admins: ['M12 3 3 7v5c0 5 9 9 9 9s9-4 9-9V7l-9-4ZM9 12l2 2 4-4'],
  settings: ['M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8ZM10 2h4l1 3 3 1 3-1 2 4-2 3 1 3-2 4-3-1-3 1-1 3h-4l-1-3-3-1-3 1-2-4 2-3-1-3 2-4 3 1 3-1 1-3Z'],
  bar: ['M5 3h14l-1 8a6 6 0 0 1-12 0L5 3ZM12 17v4M8 21h8M6 7h12'],
  lottery: ['M6 8h12v5a6 6 0 0 1-12 0V8ZM6 10H3v2a4 4 0 0 0 4 4M18 10h3v2a4 4 0 0 1-4 4M12 19v3M8 22h8M8 4l1-2M16 4l-1-2M12 4V1'],
  logout: ['M9 4H5a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h4M14 8l4 4-4 4M8 12h13'],
  menu: ['M4 6h16M4 12h16M4 18h16'],
  close: ['m6 6 12 12M6 18 18 6'],
  'chevron-left': ['m15 18-6-6 6-6'],
  'chevron-right': ['m9 18 6-6-6-6'],
  chevrons: ['m8 8 4-4 4 4M8 16l4 4 4-4'],
  'arrow-right': ['M4 12h16m-6-6 6 6-6 6'],
  'arrow-up-right': ['M6 18 18 6M6 6h12v12'],
  plus: ['M12 5v14M5 12h14'],
  trend: ['m3 17 6-6 4 4 8-10M15 5h6v6'],
  search: ['M20 20l-5-5M17 10a7 7 0 1 1-14 0 7 7 0 0 1 14 0Z'],
  check: ['m5 12 4 4L19 6'],
  globe: ['M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0ZM3 12h18M12 3c5 5 5 13 0 18-5-5-5-13 0-18Z'],
  home: ['m3 10 9-7 9 7M5 9v12h14V9M9 21v-7h6v7'],
  image: ['M5 3h14a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2ZM21 15l-6-6L3 21M8 7h.01'],
  file: ['M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8l-6-6ZM14 2v6h6M8 13h8M8 17h6'],
  boost: ['m13 2-9 12h7l-1 8 10-13h-8l1-7Z'],
  history: ['M3 11a9 9 0 1 1 2 7M3 4v7h7M12 7v5l3 2'],
  closing: ['M6 10V7a6 6 0 0 1 12 0v3M5 10h14v12H5V10ZM12 14v4'],
};
icons.calendar = icons.events;
icons.users = icons.teams;
icons.chart = icons.results;
icons.trophy = icons.lottery;
icons.shield = icons.admins;
icons.selfies = icons.image;
const paths = computed(() => icons[props.name] || icons.dashboard);
</script>
