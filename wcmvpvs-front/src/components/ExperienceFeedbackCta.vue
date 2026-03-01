<template>
  <button
    type="button"
    class="experience-feedback-cta"
    :style="ctaStyle"
    :disabled="disabled"
    :aria-disabled="disabled"
    @click="emit('select')"
  >
    <span class="experience-feedback-cta__title">Migliora la tua esperienza</span>
    <span class="experience-feedback-cta__subtitle">(sono solo 15 secondi)</span>
  </button>
</template>

<script setup>
import { computed } from 'vue';

const props = defineProps({
  heightPx: {
    type: Number,
    default: 0,
  },
  disabled: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits(['select']);

const ctaStyle = computed(() => ({
  minHeight: 'clamp(8.5rem, 24vh, 18rem)',
  height: props.heightPx > 0 ? `${props.heightPx}px` : 'auto',
}));
</script>

<style scoped>
.experience-feedback-cta {
  position: relative;
  display: flex;
  width: 100%;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.3rem;
  border-radius: 0.75rem;
  border: 1px solid rgba(255, 255, 255, 0.42);
  background:
    radial-gradient(circle at 20% 18%, rgba(251, 191, 36, 0.36), transparent 42%),
    radial-gradient(circle at 80% 80%, rgba(96, 165, 250, 0.32), transparent 36%),
    linear-gradient(160deg, rgba(30, 41, 59, 0.9), rgba(15, 23, 42, 0.98));
  box-shadow: 0 0 28px rgba(251, 191, 36, 0.25), 0 14px 32px rgba(15, 23, 42, 0.45);
  text-align: center;
  color: #f8fafc;
  transition: transform 0.18s ease, box-shadow 0.18s ease, border-color 0.18s ease;
}

.experience-feedback-cta:hover {
  transform: translateY(-1px) scale(1.01);
  border-color: rgba(251, 191, 36, 0.65);
  box-shadow: 0 0 34px rgba(251, 191, 36, 0.36), 0 16px 36px rgba(15, 23, 42, 0.55);
}

.experience-feedback-cta:active {
  transform: scale(0.985);
}

.experience-feedback-cta:disabled {
  cursor: default;
  opacity: 0.72;
  filter: grayscale(0.2);
}

.experience-feedback-cta__title {
  font-size: clamp(1.1rem, 4.8vw, 1.8rem);
  font-weight: 900;
  letter-spacing: -0.01em;
  text-shadow: 0 6px 20px rgba(0, 0, 0, 0.45);
}

.experience-feedback-cta__subtitle {
  font-size: clamp(0.78rem, 3.2vw, 1rem);
  font-weight: 700;
  color: rgba(226, 232, 240, 0.95);
}
</style>
