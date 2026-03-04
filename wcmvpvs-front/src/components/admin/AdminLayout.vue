<template>
  <div class="admin-layout" :class="{ 'is-mobile-open': mobileOpen }">
    <aside class="admin-layout__sidebar">
      <slot name="sidebar" />
    </aside>
    <div class="admin-layout__main">
      <slot name="header" />
      <main class="admin-layout__content">
        <slot />
      </main>
    </div>
    <button
      v-if="mobileOpen"
      class="admin-layout__overlay"
      type="button"
      aria-label="Chiudi menu"
      @click="$emit('close-mobile')"
    />
  </div>
</template>

<script setup>
defineProps({
  mobileOpen: {
    type: Boolean,
    default: false,
  },
});
defineEmits(["close-mobile"]);
</script>

<style scoped>
.admin-layout { min-height: 100vh; display: grid; grid-template-columns: 280px 1fr; background: #f1f5f9; }
.admin-layout__sidebar { background: #0f172a; color: #e2e8f0; position: sticky; top: 0; height: 100vh; overflow-y: auto; }
.admin-layout__main { min-width: 0; display: flex; flex-direction: column; min-height: 100vh; }
.admin-layout__content { flex: 1; padding: 1.5rem; overflow: auto; }
.admin-layout__overlay { display: none; }
@media (max-width: 1023px) {
  .admin-layout { display: block; }
  .admin-layout__sidebar { position: fixed; z-index: 40; inset: 0 auto 0 0; width: min(82vw, 300px); transform: translateX(-101%); transition: transform 0.25s ease; }
  .admin-layout.is-mobile-open .admin-layout__sidebar { transform: translateX(0); }
  .admin-layout__content { padding: 1rem; }
  .admin-layout__overlay { display: block; position: fixed; inset: 0; z-index: 30; border: 0; background: rgba(15, 23, 42, 0.45); }
}
</style>
