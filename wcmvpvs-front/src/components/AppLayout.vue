<template>
  <div class="app-layout">
    <aside class="app-sidebar desktop-only">
      <AppSidebar :items="menuItems" :active-key="activeKey" @select="onSelect" />
    </aside>
    <div class="app-main">
      <header class="app-topbar">
        <div class="topbar-row">
          <Button
            class="menu-toggle"
            icon="pi pi-bars"
            rounded
            outlined
            @click="mobileMenuVisible = true"
            aria-label="Apri navigazione"
          />
          <slot name="topbar" />
        </div>
      </header>
      <main class="app-content">
        <slot name="content" />
      </main>
    </div>

    <Sidebar
      v-model:visible="mobileMenuVisible"
      position="left"
      class="app-mobile-sidebar"
      :dismissable="true"
      :modal="true"
      header="Navigazione"
    >
      <AppSidebar :items="menuItems" :active-key="activeKey" @select="onMobileSelect" />
    </Sidebar>
  </div>
</template>

<script setup>
import { defineEmits, defineProps, ref } from 'vue';
import Button from 'primevue/button';
import Sidebar from 'primevue/sidebar';
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

const mobileMenuVisible = ref(false);

function onSelect(key) {
  emit('select', key);
}

function onMobileSelect(key) {
  mobileMenuVisible.value = false;
  onSelect(key);
}
</script>

<style scoped>
.app-layout {
  display: grid;
  grid-template-columns: 280px 1fr;
  min-height: 80vh;
  background: radial-gradient(circle at 20% 20%, rgba(99, 102, 241, 0.08), transparent 35%),
    radial-gradient(circle at 80% 0%, rgba(6, 182, 212, 0.08), transparent 32%),
    linear-gradient(135deg, #f8fafc 0%, #eef2ff 100%);
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
  padding: 1.25rem 1.75rem;
  box-shadow: 0 10px 30px rgba(15, 23, 42, 0.06);
}

.topbar-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.menu-toggle {
  display: none;
}

.app-content {
  padding: 1.5rem 1.75rem 2rem;
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.app-mobile-sidebar :deep(.sidebar-shell) {
  padding-top: 0.25rem;
}

.desktop-only {
  display: block;
}

@media (max-width: 1200px) {
  .app-layout {
    grid-template-columns: 260px 1fr;
  }
}

@media (max-width: 1024px) {
  .app-layout {
    grid-template-columns: 1fr;
    border-radius: 12px;
  }

  .app-sidebar.desktop-only {
    display: none;
  }

  .menu-toggle {
    display: inline-flex;
  }
}

@media (max-width: 640px) {
  .app-topbar {
    padding: 1rem 1.25rem;
  }

  .app-content {
    padding: 1rem 1.25rem 1.5rem;
  }
}
</style>
