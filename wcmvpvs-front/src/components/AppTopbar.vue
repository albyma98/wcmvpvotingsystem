<template>
  <div class="topbar-shell">
    <div class="topbar-branding">
      <div class="brand-mark">
        <i class="pi pi-shield"></i>
      </div>
      <div class="topbar-titles">
        <div class="breadcrumb" v-if="breadcrumb && breadcrumb.length">
          <span v-for="(crumb, index) in breadcrumb" :key="crumb">{{ crumb }}</span>
        </div>
        <h2>{{ title }}</h2>
        <p v-if="subtitle">{{ subtitle }}</p>
      </div>
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
  gap: 1.5rem;
  color: #e2e8f0;
}

.topbar-branding {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.brand-mark {
  width: 48px;
  height: 48px;
  display: grid;
  place-items: center;
  border-radius: 14px;
  background: linear-gradient(135deg, #6366f1, #06b6d4);
  color: #0b1224;
  box-shadow: 0 10px 30px rgba(6, 182, 212, 0.35);
}

.topbar-titles h2 {
  margin: 0;
  font-size: 1.5rem;
  color: #f8fafc;
}

.topbar-titles p {
  margin: 0.15rem 0 0;
  color: #cbd5e1;
}

.breadcrumb {
  display: flex;
  gap: 0.35rem;
  align-items: center;
  font-size: 0.85rem;
  color: #a5b4fc;
  margin-bottom: 0.25rem;
  text-transform: uppercase;
  letter-spacing: 0.08em;
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
  padding: 0.4rem 0.75rem;
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: 14px;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.08);
}

.avatar-meta {
  display: flex;
  flex-direction: column;
  line-height: 1.2;
  color: #f8fafc;
}

.avatar-meta small {
  color: #cbd5e1;
}
</style>
