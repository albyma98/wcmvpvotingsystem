<template>
  <div class="app-shell">
    <header class="app-shell__topbar">
      <div class="app-shell__title">
        <p v-if="eyebrow" class="app-shell__eyebrow">{{ eyebrow }}</p>
        <h1>{{ title }}</h1>
        <p v-if="subtitle" class="app-shell__subtitle">{{ subtitle }}</p>
      </div>
      <div class="app-shell__top-actions">
        <slot name="top-actions"></slot>
      </div>
    </header>

    <div class="app-shell__body">
      <aside class="app-shell__sidenav">
        <div class="app-shell__nav-header">
          <slot name="nav-header"></slot>
        </div>
        <nav class="app-shell__nav" aria-label="Navigazione principale">
          <button
            v-for="item in navItems"
            :key="item.key"
            class="app-shell__nav-btn"
            :class="{ 'app-shell__nav-btn--active': item.key === activeKey }"
            type="button"
            @click="$emit('navigate', item.key)"
            :aria-current="item.key === activeKey ? 'page' : undefined"
          >
            <span class="app-shell__nav-label">{{ item.label }}</span>
            <Tag v-if="item.badge" :value="item.badge" severity="info" />
          </button>
        </nav>
      </aside>
      <main class="app-shell__content">
        <slot></slot>
      </main>
    </div>
  </div>
</template>

<script setup>
import { defineProps } from 'vue';
import Tag from 'primevue/tag';

defineProps({
  eyebrow: {
    type: String,
    default: '',
  },
  title: {
    type: String,
    required: true,
  },
  subtitle: {
    type: String,
    default: '',
  },
  navItems: {
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
.app-shell {
  min-height: 100vh;
  display: grid;
  grid-template-rows: 56px 1fr;
  grid-template-columns: 260px 1fr;
  background: radial-gradient(circle at 20% 20%, rgba(14, 165, 233, 0.08), transparent 25%),
    #0b1021;
  color: #e2e8f0;
}

.app-shell__topbar {
  grid-column: 1 / -1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0 1.5rem;
  backdrop-filter: blur(12px);
  background: rgba(15, 23, 42, 0.75);
  border-bottom: 1px solid rgba(226, 232, 240, 0.08);
  height: 56px;
  min-height: 56px;
}

.app-shell__title h1 {
  margin: 0;
  font-size: 1.2rem;
  letter-spacing: -0.02em;
}

.app-shell__title {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 0.1rem;
}

.app-shell__subtitle {
  margin: 0.05rem 0 0;
  color: rgba(226, 232, 240, 0.75);
  font-size: 0.9rem;
}

.app-shell__eyebrow {
  margin: 0;
  text-transform: uppercase;
  letter-spacing: 0.18em;
  font-size: 0.75rem;
  color: rgba(94, 234, 212, 0.9);
}

.app-shell__top-actions {
  display: flex;
  gap: 0.5rem;
  align-items: center;
}

.app-shell__body {
  display: grid;
  grid-template-columns: 260px 1fr;
  grid-template-rows: 1fr;
}

.app-shell__sidenav {
  background: rgba(15, 23, 42, 0.92);
  border-right: 1px solid rgba(226, 232, 240, 0.08);
  padding: 1.25rem 1rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.app-shell__nav-header {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  padding: 0.5rem 0.25rem;
}

.app-shell__nav {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.app-shell__nav-btn {
  width: 100%;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 0.75rem;
  background: rgba(148, 163, 184, 0.08);
  border: 1px solid rgba(148, 163, 184, 0.15);
  color: #e2e8f0;
  padding: 0.75rem 0.9rem;
  border-radius: 0.75rem;
  cursor: pointer;
  transition: all 0.2s ease;
}

.app-shell__nav-btn:hover {
  background: rgba(14, 165, 233, 0.12);
  border-color: rgba(14, 165, 233, 0.45);
}

.app-shell__nav-btn--active {
  background: linear-gradient(135deg, rgba(14, 165, 233, 0.2), rgba(14, 165, 233, 0.35));
  border-color: rgba(14, 165, 233, 0.7);
  box-shadow: 0 8px 24px rgba(14, 165, 233, 0.18);
}

.app-shell__nav-label {
  font-weight: 600;
  letter-spacing: -0.01em;
}

.app-shell__content {
  padding: 1.5rem;
  overflow: auto;
  background: radial-gradient(circle at 80% 0%, rgba(79, 70, 229, 0.08), transparent 30%),
    linear-gradient(180deg, rgba(15, 23, 42, 0.7), rgba(15, 23, 42, 0.95));
}

@media (max-width: 1024px) {
  .app-shell {
    grid-template-columns: 220px 1fr;
  }

  .app-shell__body {
    grid-template-columns: 220px 1fr;
  }
}

@media (max-width: 768px) {
  .app-shell {
    grid-template-columns: 1fr;
    grid-template-rows: auto auto 1fr;
  }

  .app-shell__body {
    grid-template-columns: 1fr;
  }

  .app-shell__sidenav {
    flex-direction: row;
    overflow-x: auto;
    gap: 0.5rem;
  }

  .app-shell__nav {
    flex-direction: row;
  }

  .app-shell__nav-btn {
    width: auto;
    white-space: nowrap;
  }
}
</style>
