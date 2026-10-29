<template>
  <svg
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="1.5"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
  >
    <path v-if="name === 'arena'" d="m3 18 9-13 9 13M7 13h10M12 5v13M3 20h18" />
    <template v-else-if="name === 'cube'">
      <path d="m12 3 9 5v8l-9 5-9-5V8zM3 8l9 5 9-5M12 13v8" />
    </template>
    <template v-else-if="name === 'view'">
      <path d="m3 8 9-5 9 5-9 5zM3 12l9 5 9-5M3 16l9 5 9-5" />
    </template>
    <template v-else-if="name === 'play'">
      <path d="m9 5 11 7-11 7z" fill="currentColor" stroke="none" />
    </template>
    <path
      v-else-if="name === 'stop'"
      d="M7 7h10v10H7z"
      fill="currentColor"
      stroke="none"
    />
    <path v-else-if="name === 'reset'" d="M3 10a9 9 0 1 1 2 8M3 4v6h6" />
    <template v-else-if="name === 'expand'">
      <path d="M8 3H3v5m13-5h5v5M3 16v5h5m13-5v5h-5" />
    </template>
    <template v-else-if="name === 'portrait'">
      <rect x="6" y="2" width="12" height="20" rx="2" />
      <path d="M10 18h4" />
    </template>
    <template v-else-if="name === 'mouse'">
      <rect x="6" y="2" width="12" height="20" rx="6" />
      <path d="M12 2v7" />
    </template>
    <template v-else-if="name === 'zoom'">
      <circle cx="10" cy="10" r="6" />
      <path d="m15 15 6 6M7 10h6m-3-3v6" />
    </template>
    <path v-else-if="name === 'arrow'" d="M5 12h14m-6-6 6 6-6 6" />
    <template v-else-if="name === 'photo'">
      <rect x="3" y="4" width="18" height="16" rx="2" />
      <circle cx="8" cy="9" r="1" />
      <path d="m3 17 6-5 4 3 4-5 4 5" />
    </template>
    <template v-else-if="name === 'close'">
      <path d="m6 6 12 12M6 18 18 6" />
    </template>
    <template v-else-if="name === 'info'">
      <circle cx="12" cy="12" r="9" />
      <path d="M12 11v6m0-10v.1" />
    </template>
    <path v-else d="m9 6 6 6-6 6" />
  </svg>
</template>
<script setup>
defineProps({ name: String });
</script>
