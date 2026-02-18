<template>
  <div class="live-experience relative h-[100dvh] overflow-hidden text-white">
    <div class="arena-bg absolute inset-0" aria-hidden="true" />
    <div class="vignette absolute inset-0" aria-hidden="true" />

    <main class="relative z-10 flex h-full flex-col px-3 pb-3 pt-3 sm:px-4">
      <LiveHeader
        :team-name="teamName"
        :team-logo-url="teamLogoUrl"
        :is-live="isLive"
        :sponsor-line="sponsorLine"
      />

      <section class="hero animate-on-enter mt-[3.2vh] text-center">
        <h1 class="font-black uppercase leading-[0.92] tracking-tight drop-shadow-[0_4px_14px_rgba(0,0,0,0.85)]">
          <span class="block text-[clamp(2rem,10vw,4rem)]">ENTRA NELLA</span>
          <span class="block text-[clamp(2.7rem,12vw,4.8rem)] text-amber-400">PARTITA</span>
        </h1>
        <p class="mx-auto mt-2 max-w-[92%] border-t border-amber-300/50 pt-2 text-[clamp(0.9rem,3.8vw,1.4rem)] font-extrabold tracking-tight text-slate-100/95 drop-shadow-md">
          Vota • Gioca • Vinci • Partecipa
        </p>
      </section>

      <section class="animate-on-enter mt-[3.2vh] grid grid-cols-3 gap-2.5">
        <FeatureCard
          v-for="feature in features"
          :key="feature.id"
          v-bind="feature"
          @select="onFeatureSelect"
        />
      </section>

      <LiveResultsBar class="animate-on-enter mt-auto" :results="results" />
    </main>
  </div>
</template>

<script setup>
import FeatureCard from '../components/FeatureCard.vue';
import LiveHeader from '../components/LiveHeader.vue';
import LiveResultsBar from '../components/LiveResultsBar.vue';

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
  features: {
    type: Array,
    default: () => [
      {
        id: 'vote-mvp',
        title: 'VOTA L\'MVP',
        subtitle: 'del pubblico',
        description: 'Votazioni aperte',
        actionLabel: 'CLICCA ORA',
        icon: '◔',
        theme: 'orange',
      },
      {
        id: 'game-live',
        title: 'GIOCO LIVE',
        subtitle: 'Reaction Challenge',
        description: 'Batti il record del pubblico',
        actionLabel: 'GIOCA',
        icon: '⚡',
        theme: 'blue',
      },
      {
        id: 'lottery-live',
        title: 'ESTRAZIONE PREMI',
        subtitle: 'in diretta',
        description: 'Sei già dentro se partecipi',
        actionLabel: 'SCOPRI',
        icon: '▣',
        theme: 'green',
      },
    ],
  },
  results: {
    type: Array,
    default: () => [
      { name: 'ROSSI', value: 42 },
      { name: 'BIANCHI', value: 37 },
      { name: 'VERDI', value: 21 },
    ],
  },
});

const emit = defineEmits(['feature-select']);

function onFeatureSelect(featureId) {
  emit('feature-select', featureId);
}
</script>

<style scoped>
.live-experience {
  background:
    radial-gradient(circle at 50% -15%, rgba(59, 130, 246, 0.35), transparent 55%),
    radial-gradient(circle at 85% 26%, rgba(251, 191, 36, 0.32), transparent 38%),
    radial-gradient(circle at 14% 28%, rgba(255, 255, 255, 0.21), transparent 28%),
    linear-gradient(180deg, #030712 0%, #0f172a 45%, #030712 100%);
}

.arena-bg {
  background:
    radial-gradient(circle at 20% 24%, rgba(255, 255, 255, 0.46), transparent 13%),
    radial-gradient(circle at 83% 27%, rgba(255, 180, 92, 0.5), transparent 16%),
    radial-gradient(circle at 50% 68%, rgba(255, 255, 255, 0.16), transparent 35%),
    linear-gradient(180deg, rgba(15, 23, 42, 0.72) 0%, rgba(2, 6, 23, 0.85) 100%);
  filter: blur(1.8px);
}

.vignette {
  background: radial-gradient(circle at center, rgba(2, 6, 23, 0) 44%, rgba(2, 6, 23, 0.8) 100%);
}

.animate-on-enter {
  animation: fade-slide-up 0.6s ease both;
}

@keyframes fade-slide-up {
  from {
    opacity: 0;
    transform: translateY(12px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (max-height: 760px) {
  .hero {
    margin-top: 2vh;
  }

  :deep(article) {
    min-height: 196px;
    padding-top: 0.5rem;
    padding-bottom: 0.5rem;
  }
}

@media (prefers-reduced-motion: reduce) {
  .animate-on-enter {
    animation: none;
  }
}
</style>
