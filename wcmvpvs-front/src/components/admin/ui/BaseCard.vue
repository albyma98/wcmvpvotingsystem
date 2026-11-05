<template>
  <div class="base-card" :class="{ 'base-card--dark': dark }">
    <slot />
  </div>
</template>
<script setup>
defineProps({ dark: { type: Boolean, default: false } });
</script>
<style scoped>
.base-card { min-width: 0; padding: 24px; background: var(--bg-card,#fff); border: 1px solid var(--border,#e3e7ef); border-radius: var(--radius-lg,14px); box-shadow: var(--shadow-card,0 2px 5px rgba(24,34,56,.025)); }
.base-card--dark { background: var(--bg-card,#fff); }
@media(max-width:640px) { .base-card { padding: 18px; } }
</style>
