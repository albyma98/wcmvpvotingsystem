<template>
  <aside
    class="sidenav"
    :class="{
      'is-collapsed': collapsed,
      'is-overlay': overlay,
      'is-visible': visible,
    }"
  >
    <div class="sidenav__header">
      <p class="sidenav__title">Navigazione</p>
      <button class="ghost-button icon-only" type="button" aria-label="Chiudi menu" @click="$emit('close')">
        ✕
      </button>
    </div>
    <nav class="sidenav__groups">
      <div v-for="section in sections" :key="section.label" class="sidenav__group">
        <p class="group-label">{{ section.label }}</p>
        <ul>
          <li v-for="item in section.items" :key="item.to">
            <RouterLink
              :to="item.to"
              :custom-class="navLinkClass(item.to)"
              exact
              @click="$emit('navigate', item.to)"
            >
              <span class="icon">{{ item.icon }}</span>
              <span class="label" v-if="!collapsed">{{ item.label }}</span>
            </RouterLink>
          </li>
        </ul>
      </div>
    </nav>
  </aside>
</template>

<script setup>
import { useRoute } from '../../router';

defineProps({
  sections: {
    type: Array,
    default: () => [],
  },
  collapsed: {
    type: Boolean,
    default: false,
  },
  overlay: {
    type: Boolean,
    default: false,
  },
  visible: {
    type: Boolean,
    default: true,
  },
});

const route = useRoute();

const isActive = (path) => {
  return route.value?.fullPath?.startsWith(path);
};

const navLinkClass = (path) => {
  return `nav-link ${isActive(path) ? 'active' : ''}`.trim();
};
</script>

<style scoped>
.sidenav {
  position: sticky;
  top: 0;
  align-self: flex-start;
  width: 260px;
  min-height: calc(100vh - 64px);
  padding: 1rem;
  border-right: 1px solid var(--border-strong);
  background: rgba(11, 16, 28, 0.9);
  backdrop-filter: blur(14px);
  transition: width 0.2s ease, transform 0.2s ease, opacity 0.2s ease;
}

.sidenav.is-collapsed {
  width: 86px;
}

.sidenav.is-overlay {
  position: fixed;
  inset: 0 auto 0 0;
  transform: translateX(-100%);
  z-index: 40;
}

.sidenav.is-overlay.is-visible {
  transform: translateX(0);
  box-shadow: 0 20px 48px rgba(0, 0, 0, 0.45);
}

.sidenav__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0.5rem;
}

.sidenav__title {
  margin: 0;
  color: var(--text-muted);
  font-weight: 700;
  letter-spacing: 0.02em;
}

.sidenav__groups {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.sidenav__group ul {
  list-style: none;
  padding: 0;
  margin: 0.35rem 0 0 0;
  display: grid;
  gap: 0.25rem;
}

.group-label {
  margin: 0;
  color: var(--text-muted);
  font-size: 0.75rem;
  font-weight: 800;
  letter-spacing: 0.08em;
}

.nav-link {
  display: inline-flex;
  align-items: center;
  gap: 0.65rem;
  width: 100%;
  padding: 0.6rem 0.75rem;
  border-radius: 0.75rem;
  color: var(--text-primary);
  text-decoration: none;
  border: 1px solid transparent;
  transition: all 0.16s ease;
}

.nav-link:hover {
  background: rgba(255, 255, 255, 0.05);
  border-color: var(--border-strong);
}

.nav-link.active {
  background: rgba(96, 165, 250, 0.12);
  border-color: rgba(96, 165, 250, 0.25);
  box-shadow: 0 10px 26px rgba(37, 99, 235, 0.15);
}

.icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 1.75rem;
  height: 1.75rem;
  border-radius: 0.6rem;
  background: rgba(255, 255, 255, 0.05);
  font-weight: 700;
  color: #8ab4ff;
}

.label {
  font-weight: 700;
}

.ghost-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.35rem 0.4rem;
  border-radius: 0.6rem;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid var(--border-strong);
  color: var(--text-primary);
}

.ghost-button:hover {
  background: rgba(255, 255, 255, 0.08);
}

.icon-only {
  width: 2rem;
  height: 2rem;
}

@media (max-width: 1024px) {
  .sidenav {
    min-height: 100vh;
    border-right: none;
  }
}
</style>
