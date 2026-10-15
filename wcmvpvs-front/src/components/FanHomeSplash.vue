<template>
  <div class="fan-home-splash" :style="themeStyle" role="status" aria-live="polite" aria-busy="true">
    <div class="splash-content">
      <img v-if="logoUrl" class="splash-logo" :src="logoUrl" :alt="name || 'Logo della società'" />
      <span v-else class="splash-initials" aria-hidden="true">{{ initials }}</span>
      <p>Caricando i giocatori...</p>
      <span class="splash-spinner" aria-hidden="true"></span>
    </div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted } from 'vue';
import { organizationThemeStyle } from '../organizationTheme';

const props = defineProps({
  colors: { type: Object, default: () => ({}) },
  logoUrl: { type: String, default: '' },
  name: { type: String, default: '' },
});
const emit = defineEmits(['minimum-duration-elapsed']);
let minimumDurationTimer;
onMounted(() => {
  minimumDurationTimer = window.setTimeout(() => emit('minimum-duration-elapsed'), 2500);
});
onBeforeUnmount(() => window.clearTimeout(minimumDurationTimer));
const themeStyle = computed(() => organizationThemeStyle(props.colors));
const initials = computed(() => props.name.trim().split(/\s+/).filter(Boolean).slice(0, 3).map((part) => part[0]).join(''));
</script>

<style scoped>
.fan-home-splash {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: grid;
  place-items: center;
  min-height: 100dvh;
  padding: 32px;
  box-sizing: border-box;
  background: var(--red);
  color: var(--on-primary);
  font-family: 'Inter', system-ui, sans-serif;
}
.splash-content { display: flex; flex-direction: column; align-items: center; text-align: center; }
.splash-logo, .splash-initials { width: clamp(180px, 48vw, 260px); height: clamp(180px, 48vw, 260px); }
.splash-logo { display: block; object-fit: contain; filter: drop-shadow(0 10px 24px rgb(0 0 0 / 14%)); }
.splash-initials { display: grid; place-items: center; font-size: 64px; font-weight: 800; }
.splash-content p { margin: 28px 0 22px; font-size: clamp(17px, 4vw, 21px); font-weight: 600; line-height: 1.5; }
.splash-spinner { width: 24px; height: 24px; border: 3px solid currentColor; border-right-color: transparent; border-radius: 50%; animation: splash-spin 0.85s linear infinite; }
@keyframes splash-spin { to { transform: rotate(360deg); } }
@media (prefers-reduced-motion: reduce) { .splash-spinner { animation: none; } }
</style>
