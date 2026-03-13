<template>
  <div class="section-header">
    <div class="section-header__text">
      <h2 class="section-header__title">{{ title }}</h2>
      <p v-if="description" class="section-header__desc">{{ description }}</p>
    </div>
    <div v-if="$slots.actions" class="section-header__actions">
      <slot name="actions" />
    </div>
  </div>
</template>
<script setup>
defineProps({ title: String, description: String });
</script>
<style scoped>
.section-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1.25rem;
  margin-bottom: 1.5rem;
  flex-wrap: wrap;
}
.section-header__title {
  margin: 0 0 0.25rem;
  font-family: 'Barlow Condensed', 'Impact', sans-serif;
  font-size: 1.4rem;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: #0f172a;
}
.section-header__desc {
  margin: 0;
  font-size: 0.875rem;
  color: rgba(148, 163, 184, 0.7);
  line-height: 1.5;
}
.section-header__actions {
  display: flex;
  gap: 0.75rem;
  flex-wrap: wrap;
  flex-shrink: 0;
}
</style>
