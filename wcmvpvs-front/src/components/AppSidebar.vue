<template>
  <div class="sidebar-shell">
    <div class="sidebar-header">
      <span class="sidebar-logo">MVP</span>
      <div class="sidebar-title">Admin</div>
    </div>
    <PanelMenu
      class="sidebar-menu"
      :model="menuModel"
      :pt="{
        headerAction: ({ context }) => ({
          class: ['sidebar-link', { active: activeKey === context.item.id }],
        }),
      }"
    >
      <template #itemicon="slotProps">
        <i :class="slotProps.item.icon"></i>
      </template>
      <template #itemlabel="slotProps">
        <div class="sidebar-link__text">
          <span class="sidebar-link__label">{{ slotProps.item.label }}</span>
          <small v-if="slotProps.item.description">{{ slotProps.item.description }}</small>
        </div>
      </template>
    </PanelMenu>
  </div>
</template>

<script setup>
import { computed, defineEmits, defineProps } from 'vue';
import PanelMenu from 'primevue/panelmenu';

const props = defineProps({
  items: {
    type: Array,
    default: () => [],
  },
  activeKey: {
    type: String,
    default: '',
  },
});

const emit = defineEmits(['select']);

const menuModel = computed(() =>
  props.items.map((item) => ({
    id: item.id,
    label: item.label,
    icon: item.icon ? `pi ${item.icon}` : undefined,
    description: item.description,
    command: () => emit('select', item.id),
  })),
);
</script>

<style scoped>
.sidebar-shell {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 1rem;
  padding: 1.5rem 1.25rem;
}

.sidebar-header {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.5rem 0.75rem;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 12px;
}

.sidebar-logo {
  width: 42px;
  height: 42px;
  display: grid;
  place-items: center;
  border-radius: 12px;
  background: linear-gradient(135deg, #6366f1, #06b6d4);
  font-weight: 800;
  letter-spacing: 0.5px;
  color: #0b1224;
}

.sidebar-title {
  font-size: 1rem;
  font-weight: 700;
  letter-spacing: 0.02em;
}

.sidebar-menu {
  margin-top: 0.5rem;
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.02);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.05);
}

.sidebar-link {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 0.85rem;
  border: 1px solid transparent;
  background: transparent;
  color: inherit;
  padding: 0.95rem 0.85rem;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.sidebar-link i {
  font-size: 1.1rem;
}

.sidebar-link__text {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
}

.sidebar-link__label {
  font-weight: 600;
}

.sidebar-link small {
  color: #cbd5e1;
  font-size: 0.75rem;
}

.sidebar-link:hover {
  background: rgba(255, 255, 255, 0.05);
  border-color: rgba(255, 255, 255, 0.1);
}

.sidebar-link.active {
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.18), rgba(6, 182, 212, 0.18));
  border-color: rgba(99, 102, 241, 0.35);
  color: #ffffff;
}

:deep(.p-panelmenu-panel) {
  border: none;
  background: transparent;
  color: inherit;
}

:deep(.p-panelmenu-panel + .p-panelmenu-panel) {
  border-top: 1px solid rgba(255, 255, 255, 0.04);
}

:deep(.p-panelmenu-content) {
  border: none;
  background: transparent;
  padding: 0;
}

:deep(.p-panelmenu-icon) {
  display: none;
}

</style>
