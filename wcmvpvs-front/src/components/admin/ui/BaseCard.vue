<template>
  <div class="base-card" :class="{ 'base-card--dark': dark }">
    <slot />
  </div>
</template>
<script setup>
defineProps({ dark: { type: Boolean, default: false } });
</script>
<style scoped>
.base-card {
  background: #111827;
  border: 1px solid rgba(56, 189, 248, 0.08);
  border-radius: 16px;
  padding: 1.5rem;
  box-shadow: 0 8px 32px rgba(0,0,0,0.4);
}
.base-card--dark {
  background: #0d1424;
}
</style>
