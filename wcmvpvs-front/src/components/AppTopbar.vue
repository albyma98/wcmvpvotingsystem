<template>
  <div class="topbar-shell">
    <div class="topbar-titles">
      <div class="breadcrumb" v-if="breadcrumb && breadcrumb.length">
        <span v-for="(crumb, index) in breadcrumb" :key="crumb">{{ crumb }}</span>
      </div>
      <h2>{{ title }}</h2>
      <p v-if="subtitle">{{ subtitle }}</p>
    </div>
    <div class="topbar-actions">
      <slot name="actions" />
      <div class="avatar-chip">
        <Avatar shape="circle" icon="pi pi-user" size="large" />
        <div class="avatar-meta">
          <strong>{{ username }}</strong>
          <small>Amministratore</small>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { defineProps } from 'vue';

const props = defineProps({
  title: {
    type: String,
    default: '',
  },
  subtitle: {
    type: String,
    default: '',
  },
  breadcrumb: {
    type: Array,
    default: () => [],
  },
  username: {
    type: String,
    default: '',
  },
});
</script>

<style scoped>
.topbar-shell {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.topbar-titles h2 {
  margin: 0;
  font-size: 1.5rem;
  color: #0f172a;
}

.topbar-titles p {
  margin: 0.2rem 0 0;
  color: #475569;
}

.breadcrumb {
  display: flex;
  gap: 0.35rem;
  align-items: center;
  font-size: 0.85rem;
  color: #94a3b8;
  margin-bottom: 0.25rem;
}

.breadcrumb span + span::before {
  content: '/';
  margin-right: 0.35rem;
  color: #cbd5e1;
}

.topbar-actions {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.avatar-chip {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.35rem 0.75rem;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
}

.avatar-meta {
  display: flex;
  flex-direction: column;
  line-height: 1.2;
  color: #0f172a;
}

.avatar-meta small {
  color: #64748b;
}
</style>
