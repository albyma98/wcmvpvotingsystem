<template>
  <header class="section-header-modern">
    <div>
      <h2>{{ title }}</h2>
      <p v-if="description">{{ description }}</p>
    </div>
    <div class="section-header-modern__actions">
      <slot name="actions" />
    </div>
  </header>
</template>

<script setup>
defineProps({
  title: { type: String, required: true },
  description: { type: String, default: '' },
});
</script>

<style scoped>
.section-header-modern { display:flex; justify-content:space-between; gap:1rem; align-items:flex-start; margin-bottom: 1rem; }
.section-header-modern h2 { margin:0; color:#0f172a; }
.section-header-modern p { margin:.4rem 0 0; color:#64748b; }
.section-header-modern__actions { display:flex; gap:.6rem; flex-wrap:wrap; }
@media (max-width: 768px) { .section-header-modern { flex-direction:column; } .section-header-modern__actions{width:100%;} }
</style>
