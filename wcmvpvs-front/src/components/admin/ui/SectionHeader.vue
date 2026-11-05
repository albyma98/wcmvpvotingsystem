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
.section-header { display: flex; align-items: flex-start; justify-content: space-between; gap: 20px; margin-bottom: 24px; flex-wrap: wrap; }
.section-header__text { min-width: 0; flex: 1 1 240px; }
.section-header__title { margin: 0 0 6px; color: var(--text-primary,#182238); font: 650 20px/1.3 var(--font-body,'Segoe UI',sans-serif); letter-spacing: -.025em; }
.section-header__desc { margin: 0; font-size: 14px; color: var(--text-secondary,#526078); line-height: 1.6; }
.section-header__actions { display: flex; gap: 8px; flex-wrap: wrap; }
</style>
