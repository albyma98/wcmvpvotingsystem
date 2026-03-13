<template>
  <div class="admin-layout" :class="{ 'sidebar-open': mobileOpen }">
    <aside class="admin-layout__sidebar">
      <slot name="sidebar" />
    </aside>
    <div class="admin-layout__body">
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
defineProps({ mobileOpen: { type: Boolean, default: false } });
defineEmits(['close-mobile']);
</script>

<style scoped>
.admin-layout {
  min-height: 100vh;
  display: grid;
  grid-template-columns: 260px 1fr;
  background: var(--bg-base, #f0f4f8);
}

.admin-layout__sidebar {
  position: sticky;
  top: 0;
  height: 100vh;
  overflow-y: auto;
  overflow-x: hidden;
  background: var(--sidebar-bg, #ffffff);
  border-right: 1px solid rgba(15, 23, 42, 0.08);
  scrollbar-width: thin;
  scrollbar-color: #cbd5e1 transparent;
  z-index: 50;
}

.admin-layout__body {
  min-width: 0;
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}

.admin-layout__content {
  flex: 1;
  padding: 1.75rem 2rem;
  overflow: auto;
}

.admin-layout__overlay { display: none; }

@media (max-width: 1023px) {
  .admin-layout { display: block; }
  .admin-layout__sidebar {
    position: fixed;
    inset: 0 auto 0 0;
    width: min(82vw, 280px);
    transform: translateX(-101%);
    transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    z-index: 60;
    box-shadow: 8px 0 48px rgba(15, 23, 42, 0.15);
  }
  .admin-layout.sidebar-open .admin-layout__sidebar { transform: translateX(0); }
  .admin-layout__content { padding: 1rem; }
  .admin-layout__overlay {
    display: block;
    position: fixed;
    inset: 0;
    z-index: 55;
    border: 0;
    background: rgba(15, 23, 42, 0.35);
    backdrop-filter: blur(4px);
    cursor: pointer;
  }
}
</style>
