<template>
  <div class="app-layout">
    <!-- Topbar principale fissato in alto -->
    <header class="app-topbar">
      <div class="topbar-inner">
        <slot name="topbar" />
      </div>
    </header>

    <div class="app-shell">
      <!-- Sidebar di navigazione verticale -->
      <aside class="app-sidebar">
        <AppSidebar :items="menuItems" :active-key="activeKey" @select="onSelect" />
      </aside>

      <!-- Area contenuti responsiva -->
      <main class="app-content">
        <slot name="content" />
      </main>
    </div>
  </div>
</template>

<script setup>
import { defineEmits, defineProps } from 'vue';
import AppSidebar from './AppSidebar.vue';

const emit = defineEmits(['select']);
const props = defineProps({
  menuItems: {
    type: Array,
    default: () => [],
  },
  activeKey: {
    type: String,
    default: '',
  },
});

function onSelect(key) {
  emit('select', key);
}
</script>

<style scoped>
.app-layout {
  min-height: 100vh;
  background: linear-gradient(135deg, #0b1224 0%, #0f172a 35%, #0b1120 100%);
  color: #0f172a;
}

.app-topbar {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 10;
  backdrop-filter: blur(10px);
  background: rgba(15, 23, 42, 0.75);
  border-bottom: 1px solid rgba(148, 163, 184, 0.25);
  box-shadow: 0 15px 50px rgba(0, 0, 0, 0.3);
}

.topbar-inner {
  max-width: 1440px;
  margin: 0 auto;
  padding: 1rem 1.75rem;
}

.app-shell {
  display: grid;
  grid-template-columns: 280px 1fr;
  gap: 0;
  max-width: 1440px;
  margin: 0 auto;
  padding: 5.5rem 1.25rem 2rem;
}

.app-sidebar {
  position: sticky;
  top: 4.5rem;
  align-self: start;
  background: linear-gradient(180deg, rgba(15, 23, 42, 0.96), rgba(15, 23, 42, 0.9));
  color: #e2e8f0;
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 18px;
  box-shadow: 0 24px 60px rgba(0, 0, 0, 0.25);
  overflow: hidden;
  height: calc(100vh - 6rem);
}

.app-content {
  padding: 0 0 1.5rem 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  min-height: calc(100vh - 5.5rem);
}

@media (max-width: 1024px) {
  .app-shell {
    grid-template-columns: 1fr;
    padding: 5.5rem 1rem 2rem;
  }

  .app-sidebar {
    position: relative;
    top: 0;
    height: auto;
  }
}

@media (max-width: 768px) {
  .app-content {
    padding-left: 0;
  }
}
</style>
