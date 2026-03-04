<template>
  <section class="base-card" :class="[`base-card--${variant}`]">
    <slot />
  </section>
</template>

<script setup>
defineProps({
  variant: {
    type: String,
    default: 'default',
  },
});
</script>

<style scoped>
.base-card {
  background: #fff;
  border: 1px solid var(--ui-border);
  border-radius: var(--radius);
  padding: 1.25rem;
  box-shadow: 0 8px 20px rgba(15, 23, 42, 0.06);
}
.base-card--muted {
  background: #f8fafc;
}
</style>
