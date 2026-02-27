<template>
  <label
    class="input-card block rounded-2xl border p-3 transition-all duration-300"
    :class="[
      isValid
        ? 'border-emerald-300/70 shadow-[0_0_18px_rgba(16,185,129,0.32)]'
        : isFocused
          ? 'border-transparent shadow-[0_0_18px_rgba(56,189,248,0.35)]'
          : 'border-white/20',
    ]"
  >
    <div class="mb-2 flex items-center justify-between text-[11px] font-bold uppercase tracking-[0.14em] text-slate-300">
      <div class="flex items-center gap-2">
        <span class="text-base transition-colors duration-300" :class="isFocused ? 'text-cyan-300' : 'text-slate-300'">{{ icon }}</span>
        <span>{{ label }}</span>
      </div>
      <span v-if="optional" class="text-[10px] font-semibold tracking-normal text-slate-400">Opzionale</span>
      <span
        v-else-if="isValid"
        class="input-card-valid inline-flex h-5 w-5 items-center justify-center rounded-full bg-emerald-300/20 text-xs text-emerald-200"
      >
        ✓
      </span>
    </div>

    <slot :on-focus="onFocus" :on-blur="onBlur" />

    <p v-if="helper" class="mt-2 text-[11px] text-slate-300/85">{{ helper }}</p>
  </label>
</template>

<script setup>
import { ref } from 'vue';

defineProps({
  icon: { type: String, required: true },
  label: { type: String, required: true },
  helper: { type: String, default: '' },
  isValid: { type: Boolean, default: false },
  optional: { type: Boolean, default: false },
});

const isFocused = ref(false);

function onFocus() {
  isFocused.value = true;
}

function onBlur() {
  isFocused.value = false;
}
</script>

<style scoped>
.input-card {
  background: linear-gradient(155deg, rgba(30, 41, 59, 0.85), rgba(15, 23, 42, 0.75));
  position: relative;
}

.input-card::before {
  content: '';
  position: absolute;
  inset: -1px;
  border-radius: 1rem;
  padding: 1px;
  background: linear-gradient(120deg, rgba(56, 189, 248, 0.7), rgba(14, 165, 233, 0.1), rgba(34, 197, 94, 0.5));
  -webkit-mask: linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0);
  -webkit-mask-composite: xor;
  mask-composite: exclude;
  opacity: 0;
  transition: opacity 0.25s ease;
}

.input-card:focus-within::before {
  opacity: 1;
}

.input-card-valid {
  animation: valid-pop 0.28s ease;
}

@keyframes valid-pop {
  0% {
    transform: scale(0.72);
    opacity: 0.5;
  }
  100% {
    transform: scale(1);
    opacity: 1;
  }
}
</style>
