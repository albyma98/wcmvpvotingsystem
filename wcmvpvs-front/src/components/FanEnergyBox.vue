<script setup>
import { computed, ref } from "vue";
import useFanEnergy from "../composables/useFanEnergy";

const {
  energy,
  isBoostActive,
  isBoostReady,
  boostLabel,
  boostProgressPct,
  triggerBoost,
  claim,
  claimPulse,
} = useFanEnergy();

const claimResult = ref("");

const energyLabel = computed(() => `Energia: ${energy.value}`);
const canCollect = computed(() => energy.value >= 100);
const boostButtonDisabled = computed(() => !isBoostReady.value);

function handleBoost() {
  const ok = triggerBoost();
  if (!ok) {
    claimResult.value = "Boost non ancora pronto";
    return;
  }
  claimResult.value = "Boost attivato!";
  setTimeout(() => {
    if (claimResult.value === "Boost attivato!") {
      claimResult.value = "";
    }
  }, 1500);
}

function handleCollect() {
  const ok = claim();
  if (!ok) {
    claimResult.value = "Servono 100 energia";
    return;
  }
  claimResult.value = "+1 ticket ottenuto";
  setTimeout(() => {
    if (claimResult.value === "+1 ticket ottenuto") {
      claimResult.value = "";
    }
  }, 2000);
}
</script>

<template>
  <section class="fan-energy-card" aria-label="Energia Tifoso">
    <div class="fan-energy-header">
      <h3 class="fan-energy-title">⚡ Energia Tifoso</h3>
      <p class="fan-energy-subtitle">+1 ogni 8s • Boost ogni 5 min (30s)</p>
    </div>

    <div class="fan-energy-counter" aria-live="polite">
      <p class="fan-energy-counter__label">Energia</p>
      <p class="fan-energy-counter__value">{{ energyLabel }}</p>
      <p class="fan-energy-counter__cap">Cap 999</p>
    </div>

    <div class="fan-energy-boost">
      <div class="fan-energy-boost__meta">
        <p class="fan-energy-boost__label">Boost</p>
        <p class="fan-energy-boost__status" :class="{ 'fan-energy-boost__status--active': isBoostActive }">
          {{ boostLabel }}
        </p>
      </div>
      <div class="fan-energy-progress" role="presentation">
        <div class="fan-energy-progress__fill" :style="{ width: `${boostProgressPct}%` }"></div>
      </div>
      <button
        type="button"
        class="fan-energy-btn fan-energy-btn--ghost"
        :disabled="boostButtonDisabled"
        @click="handleBoost"
      >
        Boost
      </button>
    </div>

    <button
      type="button"
      class="fan-energy-btn fan-energy-btn--primary"
      :disabled="!canCollect"
      :class="{ 'fan-energy-btn--pulse': claimPulse }"
      @click="handleCollect"
    >
      Raccogli
    </button>

    <p v-if="claimResult" class="fan-energy-feedback">{{ claimResult }}</p>
  </section>
</template>

<style scoped>
.fan-energy-card {
  @apply bg-slate-900/70 border border-slate-800 rounded-2xl p-4 shadow-lg flex flex-col gap-4;
}

.fan-energy-header {
  @apply flex flex-col gap-1;
}

.fan-energy-title {
  @apply text-lg font-semibold text-white leading-tight;
}

.fan-energy-subtitle {
  @apply text-sm text-slate-300 leading-tight;
}

.fan-energy-counter {
  @apply flex flex-col gap-1;
}

.fan-energy-counter__label {
  @apply text-xs uppercase tracking-wide text-slate-400;
}

.fan-energy-counter__value {
  @apply text-3xl font-black text-white;
}

.fan-energy-counter__cap {
  @apply text-xs text-slate-500;
}

.fan-energy-boost {
  @apply flex flex-col gap-2;
}

.fan-energy-boost__meta {
  @apply flex items-center justify-between;
}

.fan-energy-boost__label {
  @apply text-sm font-medium text-slate-200;
}

.fan-energy-boost__status {
  @apply text-sm text-slate-400;
}

.fan-energy-boost__status--active {
  @apply text-emerald-300 font-semibold;
}

.fan-energy-progress {
  @apply w-full h-2 bg-slate-800 rounded-full overflow-hidden;
}

.fan-energy-progress__fill {
  @apply h-full bg-emerald-400 transition-all duration-500 ease-out;
}

.fan-energy-btn {
  @apply w-full inline-flex items-center justify-center rounded-xl px-4 py-2 text-sm font-semibold transition-colors duration-150;
}

.fan-energy-btn--ghost {
  @apply bg-slate-800 text-white hover:bg-slate-700 disabled:opacity-50 disabled:cursor-not-allowed;
}

.fan-energy-btn--primary {
  @apply bg-emerald-500 text-slate-900 hover:bg-emerald-400 disabled:bg-slate-800 disabled:text-slate-400 disabled:cursor-not-allowed;
}

.fan-energy-btn--pulse {
  animation: fan-pulse 0.4s ease-in-out;
}

.fan-energy-feedback {
  @apply text-sm text-emerald-200;
}

@keyframes fan-pulse {
  0% {
    transform: scale(1);
  }
  50% {
    transform: scale(1.03);
  }
  100% {
    transform: scale(1);
  }
}
</style>
