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

    <button
      type="button"
      class="group relative flex h-11 w-11 shrink-0 items-center justify-center overflow-hidden rounded-full border border-white/35 bg-slate-900/80 text-white shadow-[0_8px_20px_rgba(2,6,23,0.5)] transition hover:-translate-y-0.5 hover:border-amber-300/70"
      aria-label="Apri il profilo utente"
      @click="emit('profile-click')"
    >
      <img
        v-if="profileAvatarUrl"
        :src="profileAvatarUrl"
        alt="Avatar utente"
        class="h-full w-full object-cover"
      >
      <span v-else class="text-lg" aria-hidden="true">👤</span>
      <span
        v-if="isLive"
        class="pointer-events-none absolute -bottom-0.5 right-0.5 h-2.5 w-2.5 rounded-full bg-emerald-400 ring-2 ring-slate-950"
        aria-hidden="true"
      />
    </button>
  </header>
</template>

<script setup>
const emit = defineEmits(['profile-click']);

defineProps({
  teamName: {
    type: String,
    default: 'TEAM',
  },
  teamLogoUrl: {
    type: String,
    default: '',
  },
  profileAvatarUrl: {
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
