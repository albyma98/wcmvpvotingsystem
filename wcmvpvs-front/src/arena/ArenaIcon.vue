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
    <template v-if="name === 'fans'">
      <circle cx="9" cy="7" r="3" />
      <path
        d="M3 21v-4a6 6 0 0 1 12 0v4M16 4a3 3 0 0 1 0 6m2 3a5 5 0 0 1 3 5v3"
      />
    </template>
    <template v-else-if="name === 'trophy'">
      <path
        d="M8 3h8v6a4 4 0 0 1-8 0zM8 5H4v3a4 4 0 0 0 4 4m8-7h4v3a4 4 0 0 1-4 4M12 13v5m-4 3h8m-9-3h10"
      />
    </template>
    <template v-else-if="name === 'clap'">
      <path
        d="m8 13-2-4a2 2 0 0 0-3 2l3 7c2 4 6 4 8 1l4-6a2 2 0 0 0-3-2l-3 3-2-6a2 2 0 0 0-3 1l1 4M16 3l1 3m4 0-2 2M12 2v3"
      />
    </template>
    <template v-else-if="name === 'scarf'">
      <path
        d="M3 7h18v5H3zM4 12v7h4v-7m8 0v7h4v-7M6 7v5m12-5v5M4 21v-2m4 2v-2m8 2v-2m4 2v-2"
      />
    </template>
    <template v-else-if="name === 'wave'">
      <path
        d="M2 15c3 0 3-8 6-8s3 10 6 10 3-8 8-8M2 21c3 0 3-8 6-8s3 10 6 10 3-8 8-8"
      />
    </template>
    <template v-else-if="name === 'search'">
      <circle cx="10" cy="10" r="6" />
      <path d="m15 15 6 6" />
    </template>
    <template v-else-if="name === 'pin'">
      <path d="M19 10c0 5-7 11-7 11S5 15 5 10a7 7 0 1 1 14 0z" />
      <circle cx="12" cy="10" r="2" />
    </template>
    <path v-else-if="name === 'check'" d="m5 12 4 4L19 6" />
    <path
      v-else-if="name === 'arena'"
      d="m3 18 9-13 9 13M7 13h10M12 5v13M3 20h18"
    />
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
