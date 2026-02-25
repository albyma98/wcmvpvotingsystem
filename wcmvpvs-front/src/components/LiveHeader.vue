<template>
  <header class="flex h-20 items-center gap-3 rounded-xl border border-white/20 bg-slate-950/55 px-3 backdrop-blur-md">
    <div
      class="relative flex h-14 w-14 shrink-0 items-center justify-center overflow-hidden border-2 border-amber-300/80 bg-slate-950 text-sm font-extrabold uppercase tracking-wide text-white shadow-[0_0_14px_rgba(250,180,60,0.55)]"
      :class="teamLogoUrl ? 'rounded-xl' : 'rounded-full'"
    >
      <img
        v-if="teamLogoUrl"
        :src="teamLogoUrl"
        :alt="`Logo ${teamName}`"
        class="h-full w-full object-contain bg-white"
      >
      <span v-else>{{ teamName }}</span>
    </div>

    <div class="min-w-0 flex-1 leading-tight">
      <slot>
        <p class="truncate text-center text-[clamp(0.86rem,2.8vw,1.16rem)] font-extrabold tracking-tight text-white">
          LIVE EXPERIENCE UFFICIALE
        </p>
        <p class="mt-1 truncate text-center text-[clamp(0.62rem,2.1vw,0.84rem)] text-slate-200/90">
          {{ sponsorLine }}
        </p>
      </slot>
    </div>

    <div
      class="flex shrink-0 items-center gap-2"
      :aria-label="isLive ? 'Stato live attivo' : 'Stato offline'"
      role="status"
    >
      <span
        class="text-xs font-extrabold uppercase tracking-wide"
        :class="isLive ? 'text-red-300' : 'text-slate-400'"
      >
        {{ isLive ? 'LIVE' : 'OFFLINE' }}
      </span>
      <span
        class="h-2.5 w-2.5 rounded-full"
        :class="isLive ? 'bg-red-500 live-dot' : 'bg-slate-500'"
        aria-hidden="true"
      />
    </div>
  </header>
</template>

<script setup>
defineProps({
  teamName: {
    type: String,
    default: 'TEAM',
  },
  teamLogoUrl: {
    type: String,
    default: '',
  },
  isLive: {
    type: Boolean,
    default: true,
  },
  sponsorLine: {
    type: String,
    default: 'Powered by MVP System',
  },
});
</script>

<style scoped>
.live-dot {
  animation: pulse-dot 1.2s ease-in-out infinite;
  box-shadow: 0 0 0 rgba(239, 68, 68, 0.55);
}

@keyframes pulse-dot {
  0% {
    transform: scale(0.95);
    box-shadow: 0 0 0 0 rgba(239, 68, 68, 0.5);
  }
  70% {
    transform: scale(1.08);
    box-shadow: 0 0 0 9px rgba(239, 68, 68, 0);
  }
  100% {
    transform: scale(0.95);
    box-shadow: 0 0 0 0 rgba(239, 68, 68, 0);
  }
}

@media (prefers-reduced-motion: reduce) {
  .live-dot {
    animation: none;
  }
}
</style>
