<template>
  <div class="app-layout">
    <aside class="app-sidebar">
      <AppSidebar :items="menuItems" :active-key="activeKey" @select="onSelect" />
    </aside>
    <div class="app-main">
      <header class="app-topbar">
        <slot name="topbar" />
      </header>
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
  display: grid;
  grid-template-columns: 280px 1fr;
  min-height: 80vh;
  background: linear-gradient(135deg, #f8fafc 0%, #eef2ff 100%);
  border-radius: 18px;
  box-shadow: 0 25px 80px rgba(15, 23, 42, 0.15);
  overflow: hidden;
}

.app-sidebar {
  background: #0f172a;
  color: #e2e8f0;
  border-right: 1px solid rgba(255, 255, 255, 0.05);
}

.app-main {
  display: flex;
  flex-direction: column;
  min-height: 100%;
}

.app-topbar {
  position: sticky;
  top: 0;
  z-index: 2;
  background: #ffffff;
  border-bottom: 1px solid #e2e8f0;
  padding: 1.5rem 1.75rem;
  box-shadow: 0 10px 30px rgba(15, 23, 42, 0.06);
}

.app-content {
  padding: 1.5rem 1.75rem 2rem;
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

@media (max-width: 1024px) {
  .app-layout {
    grid-template-columns: 1fr;
  }

  .app-sidebar {
    display: none;
  }
}
</style>
