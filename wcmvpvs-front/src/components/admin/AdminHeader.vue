<template>
  <header class="admin-header-shell">
    <div>
      <p class="crumb">Admin / {{ sectionGroup }}</p>
      <h1>{{ title }}</h1>
    </div>
    <div class="admin-header-shell__actions">
      <button class="burger" type="button" @click="$emit('toggle-menu')">☰</button>
      <slot name="actions" />
    </div>
  </header>
</template>

<script setup>
defineProps({ title: String, sectionGroup: String });
defineEmits(["toggle-menu"]);
</script>

<style scoped>
.admin-header-shell { position: sticky; top:0; z-index:20; background:#fff; border-bottom:1px solid #e2e8f0; padding:1rem 1.5rem; display:flex; justify-content:space-between; gap:1rem; align-items:center; }
h1 { margin:.1rem 0 0; font-size:1.45rem; }
.crumb { margin:0; color:#64748b; font-size:.85rem; }
.burger { display:none; border:1px solid #cbd5e1; border-radius:.5rem; background:#fff; padding:.4rem .65rem; }
@media (max-width:1023px){ .admin-header-shell{ padding: .9rem 1rem;} .burger{display:inline-flex;} }
</style>
