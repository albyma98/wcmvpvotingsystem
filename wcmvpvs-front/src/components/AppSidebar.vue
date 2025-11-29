<template>
  <div class="sidebar-shell">
    <div class="sidebar-header">
      <span class="sidebar-logo">MVP</span>
      <div class="sidebar-title">Admin</div>
    </div>
    <nav class="sidebar-menu" aria-label="Navigazione amministratore">
      <button
        v-for="item in items"
        :key="item.id"
        type="button"
        class="sidebar-link"
        :class="{ active: activeKey === item.id }"
        @click="$emit('select', item.id)"
      >
        <i :class="['pi', item.icon]"></i>
        <div class="sidebar-link__text">
          <span class="sidebar-link__label">{{ item.label }}</span>
          <small v-if="item.description">{{ item.description }}</small>
        </div>
      </button>
    </nav>
  </div>
</template>

<script setup>
import { defineProps } from 'vue';

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
</script>

<style scoped>
.sidebar-shell {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 1rem;
  padding: 1.25rem;
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
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.sidebar-link {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 0.85rem;
  border: 1px solid transparent;
  background: transparent;
  color: inherit;
  padding: 0.9rem 0.9rem;
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
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.15), rgba(6, 182, 212, 0.15));
  border-color: rgba(99, 102, 241, 0.3);
  color: #ffffff;
}
</style>
