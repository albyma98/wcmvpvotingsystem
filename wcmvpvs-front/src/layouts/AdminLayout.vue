<template>
  <div class="admin-layout">
    <Sidebar
      v-model:visible="sidebarVisible"
      position="left"
      class="admin-layout__sidebar-overlay"
      modal
      block-scroll
    >
      <div class="admin-layout__sidebar-content">
        <div class="admin-layout__brand" v-if="$slots.brand">
          <slot name="brand" />
        </div>
        <Menu :model="menuItems" class="admin-layout__menu" />
        <div class="admin-layout__sidebar-footer" v-if="$slots['sidebar-footer']">
          <slot name="sidebar-footer" />
        </div>
      </div>
    </Sidebar>

    <div class="admin-layout__shell">
      <aside class="admin-layout__sidebar">
        <div class="admin-layout__brand" v-if="$slots.brand">
          <slot name="brand" />
        </div>
        <Menu :model="menuItems" class="admin-layout__menu" />
        <div class="admin-layout__sidebar-footer" v-if="$slots['sidebar-footer']">
          <slot name="sidebar-footer" />
        </div>
      </aside>

      <div class="admin-layout__main">
        <Toolbar class="admin-layout__toolbar">
          <template #start>
            <div class="admin-layout__toolbar-start">
              <Button
                class="admin-layout__menu-toggle"
                icon="pi pi-bars"
                severity="secondary"
                text
                rounded
                @click="sidebarVisible = true"
              />
              <div class="admin-layout__brand" v-if="$slots.brand">
                <slot name="brand" />
              </div>
            </div>
          </template>
          <template #end>
            <div class="admin-layout__toolbar-end">
              <div class="admin-layout__toolbar-actions" v-if="$slots['toolbar-actions']">
                <slot name="toolbar-actions" />
              </div>
              <div class="admin-layout__user" v-if="username">
                <Avatar :label="usernameInitials" shape="circle" size="large" class="admin-layout__avatar" />
                <div class="admin-layout__user-info">
                  <span class="admin-layout__user-label">Amministratore</span>
                  <strong class="admin-layout__user-name">{{ username }}</strong>
                </div>
              </div>
              <div class="admin-layout__toolbar-buttons">
                <Button
                  label="Lotteria"
                  icon="pi pi-ticket"
                  severity="secondary"
                  outlined
                  class="admin-layout__toolbar-button"
                  @click="handleNavigateLottery"
                />
                <Button
                  label="Esci"
                  icon="pi pi-sign-out"
                  severity="danger"
                  class="admin-layout__toolbar-button"
                  @click="handleLogout"
                />
              </div>
            </div>
          </template>
        </Toolbar>

        <div class="admin-layout__alerts" v-if="$slots.alerts">
          <slot name="alerts" />
        </div>

        <div class="admin-layout__content">
          <slot />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue';
import Sidebar from 'primevue/sidebar';
import Toolbar from 'primevue/toolbar';
import Button from 'primevue/button';
import Menu from 'primevue/menu';
import Avatar from 'primevue/avatar';

const props = defineProps({
  tabs: {
    type: Array,
    default: () => [],
  },
  activeTab: {
    type: String,
    default: '',
  },
  username: {
    type: String,
    default: '',
  },
});

const emit = defineEmits(['tab-change', 'logout', 'navigate-lottery']);

const sidebarVisible = ref(false);

const menuItems = computed(() =>
  (props.tabs || []).map((tab) => ({
    label: tab.label,
    key: tab.id,
    icon: tab.icon,
    command: () => handleTabChange(tab.id),
    class: tab.id === props.activeTab ? 'admin-layout__menu-item--active' : undefined,
  })),
);

const usernameInitials = computed(() => {
  if (!props.username) {
    return '';
  }
  const parts = props.username.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) {
    return props.username.slice(0, 2).toUpperCase();
  }
  const [first, second] = parts;
  if (second) {
    return `${first[0]}${second[0]}`.toUpperCase();
  }
  return first.slice(0, 2).toUpperCase();
});

function handleTabChange(tabId) {
  emit('tab-change', tabId);
  sidebarVisible.value = false;
}

function handleLogout() {
  emit('logout');
}

function handleNavigateLottery() {
  emit('navigate-lottery');
}
</script>

<style scoped>
.admin-layout {
  display: flex;
  min-height: 100vh;
  background: var(--p-surface-50);
}

.admin-layout__shell {
  display: grid;
  grid-template-columns: 280px 1fr;
  width: 100%;
}

.admin-layout__sidebar {
  display: none;
  flex-direction: column;
  gap: 1rem;
  background: var(--p-surface);
  border-right: 1px solid var(--p-surface-200);
  padding: 1.5rem 1rem;
}

.admin-layout__sidebar-content {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  height: 100%;
}

.admin-layout__sidebar-overlay {
  --p-sidebar-width: 16rem;
}

.admin-layout__menu {
  border: none;
}

.admin-layout__menu :deep(.p-menuitem-link) {
  border-radius: 0.75rem;
  transition: all 0.2s ease;
}

.admin-layout__menu :deep(.p-menuitem-link:hover) {
  background: var(--p-surface-200);
}

.admin-layout__menu :deep(.p-menuitem-link .p-menuitem-text) {
  font-weight: 600;
}

.admin-layout__menu-item--active :deep(.p-menuitem-link) {
  background: var(--p-primary-color);
  color: var(--p-primary-contrast-color);
  box-shadow: 0 12px 32px -20px var(--p-primary-500);
}

.admin-layout__brand {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--p-primary-700);
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.admin-layout__main {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  background: linear-gradient(180deg, var(--p-surface-50), var(--p-surface-100));
}

.admin-layout__toolbar {
  padding: 1rem 2rem;
  border-bottom: 1px solid var(--p-surface-200);
  background: rgba(255, 255, 255, 0.85);
  backdrop-filter: blur(12px);
  position: sticky;
  top: 0;
  z-index: 10;
}

.admin-layout__toolbar-start {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.admin-layout__menu-toggle {
  display: inline-flex;
}

.admin-layout__toolbar-end {
  display: flex;
  align-items: center;
  gap: 1.5rem;
}

.admin-layout__toolbar-actions {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.admin-layout__toolbar-buttons {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.admin-layout__user {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding-right: 1rem;
  border-right: 1px solid var(--p-surface-200);
}

.admin-layout__user-label {
  display: block;
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--p-text-muted-color);
}

.admin-layout__user-name {
  display: block;
  font-size: 0.95rem;
}

.admin-layout__alerts {
  padding: 1rem 2rem 0;
}

.admin-layout__content {
  padding: 2rem;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2rem;
}

.admin-layout__sidebar-footer {
  margin-top: auto;
}

@media (min-width: 992px) {
  .admin-layout__sidebar {
    display: flex;
  }
  .admin-layout__menu-toggle {
    display: none;
  }
}

@media (max-width: 991px) {
  .admin-layout__shell {
    grid-template-columns: 1fr;
  }
  .admin-layout__sidebar {
    display: none;
  }
  .admin-layout__toolbar {
    padding: 1rem;
  }
  .admin-layout__content {
    padding: 1.5rem 1rem 2rem;
  }
  .admin-layout__toolbar-end {
    gap: 1rem;
  }
  .admin-layout__user {
    display: none;
  }
}
</style>
