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
  canCollect,
  canBoost,
} = useFanEnergy();

const claimResult = ref("");
const isBoosting = ref(false);
const isClaiming = ref(false);

const energyLabel = computed(() => energy.value.toLocaleString("it-IT"));
const boostGlow = computed(() => isBoostReady.value && !isBoostActive.value);

async function handleBoost() {
  if (isBoosting.value || !canBoost.value) {
    claimResult.value = "Boost non ancora pronto";
    return;
  }
  isBoosting.value = true;
  const ok = await triggerBoost();
  isBoosting.value = false;
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

async function handleCollect() {
  if (isClaiming.value || !canCollect.value) {
    claimResult.value = "Servono 100 energia";
    return;
  }
  isClaiming.value = true;
  const ok = await claim();
  isClaiming.value = false;
  if (!ok) {
    claimResult.value = "Errore nella raccolta";
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
      <span class="fan-energy-cap">Cap 999</span>
    </div>

    <div class="fan-energy-counter" aria-live="polite">
      <p class="fan-energy-counter__value">{{ energyLabel }}</p>
      <p class="fan-energy-counter__subtitle">+5 energia ogni 5s</p>
    </div>

    <div class="fan-energy-boost">
      <div class="fan-energy-boost__row">
        <p class="fan-energy-boost__hint">Boost: +20 energia ogni 5s (30s)</p>
        <p
          class="fan-energy-boost__status"
          :class="{
            'fan-energy-boost__label--active': isBoostActive,
            'fan-energy-boost__label--ready': boostGlow,
          }"
        >
          {{ boostLabel }}
        </p>
      </div>
      <div class="fan-energy-progress" role="presentation">
        <div class="fan-energy-progress__track">
          <div
            class="fan-energy-progress__fill"
            :class="{
              'fan-energy-progress__fill--ready': boostGlow,
              'fan-energy-progress__fill--active': isBoostActive,
            }"
            :style="{ width: `${boostProgressPct}%` }"
          ></div>
        </div>
        <span
          class="fan-energy-progress__countdown"
          :class="{
            'fan-energy-boost__label--active': isBoostActive,
            'fan-energy-boost__label--ready': boostGlow,
          }"
        >
          {{ boostLabel }}
        </span>
      </div>
    </div>

    <div class="fan-energy-actions">
      <button
        type="button"
        class="fan-energy-btn fan-energy-btn--ghost"
        :class="{ 'fan-energy-btn--glow': boostGlow }"
        :disabled="!canBoost || isBoosting"
        @click="handleBoost"
      >
        Boost
      </button>
      <button
        type="button"
        class="fan-energy-btn fan-energy-btn--primary"
        :disabled="!canCollect || isClaiming"
        :class="{ 'fan-energy-btn--pulse': claimPulse, 'fan-energy-btn--glow': canCollect }"
        @click="handleCollect"
      >
        Raccogli
      </button>
    </div>

    <p v-if="claimResult" class="fan-energy-feedback">{{ claimResult }}</p>
  </section>
</template>

<style scoped>
.fan-energy-card {
  @apply bg-slate-900/80 border border-slate-800 rounded-2xl p-4 shadow-lg flex flex-col gap-3;
}

.fan-energy-header {
  @apply flex items-center justify-between gap-2;
}

.fan-energy-title {
  @apply text-lg font-semibold text-white leading-tight;
}

.fan-energy-cap {
  @apply px-2 py-1 text-xs font-semibold rounded-full bg-slate-800 text-slate-200 border border-slate-700;
}

.fan-energy-counter {
  @apply text-center flex flex-col gap-1;
}

.fan-energy-counter__value {
  @apply text-4xl font-black text-white tracking-tight;
}

.fan-energy-counter__subtitle {
  @apply text-sm text-slate-400;
}

.fan-energy-boost {
  @apply flex flex-col gap-2;
}

.fan-energy-boost__row {
  @apply flex items-center justify-between gap-3;
}

.fan-energy-boost__status {
  @apply text-sm font-semibold text-slate-300;
}

.fan-energy-boost__label--active {
  @apply text-emerald-300;
}

.fan-energy-boost__label--ready {
  @apply text-emerald-200;
}

.fan-energy-boost__hint {
  @apply text-xs text-slate-500;
}

.fan-energy-progress {
  @apply w-full flex items-center gap-3;
}

.fan-energy-progress__track {
  @apply flex-1 h-1.5 bg-slate-800 rounded-full overflow-hidden;
}

.fan-energy-progress__fill {
  @apply h-full bg-slate-600 transition-all duration-500 ease-out;
}

.fan-energy-progress__fill--active {
  @apply bg-emerald-400;
}

.fan-energy-progress__fill--ready {
  @apply bg-emerald-500 shadow-[0_0_12px_rgba(16,185,129,0.65)];
}

.fan-energy-progress__countdown {
  @apply text-xs font-semibold text-slate-400 min-w-[86px] text-right;
}

.fan-energy-actions {
  @apply grid grid-cols-2 gap-2;
}

.fan-energy-btn {
  @apply w-full inline-flex items-center justify-center rounded-xl px-4 py-2.5 text-sm font-semibold transition-all duration-150;
}

.fan-energy-btn--ghost {
  @apply bg-slate-800 text-white hover:bg-slate-700 disabled:opacity-60 disabled:cursor-not-allowed;
}

.fan-energy-btn--primary {
  @apply bg-emerald-500 text-slate-900 hover:bg-emerald-400 disabled:bg-slate-800 disabled:text-slate-400 disabled:cursor-not-allowed;
}

.fan-energy-btn--pulse {
  animation: fan-pulse 0.4s ease-in-out;
}

.fan-energy-btn--glow {
  box-shadow: 0 0 14px rgba(52, 211, 153, 0.4);
}

.fan-energy-feedback {
  @apply text-sm text-emerald-200 text-center;
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
