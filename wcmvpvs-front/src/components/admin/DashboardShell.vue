<template>
  <div class="dashboard-shell" :class="variantClass">
    <button class="dashboard-shell__mobile-toggle" type="button" @click="isSidebarOpen = true">
      <span class="sr-only">Apri menu</span>
      ☰
    </button>
    <aside
      class="dashboard-shell__sidebar"
      :class="{ 'dashboard-shell__sidebar--open': isSidebarOpen }"
      aria-label="Menu amministrazione"
    >
      <div class="dashboard-shell__brand">
        <slot name="brand">
          <p class="dashboard-shell__eyebrow">Area amministrativa</p>
          <h2>{{ title }}</h2>
        </slot>
      </div>
      <nav class="dashboard-shell__nav">
        <button
          v-for="item in menuItems"
          :key="item.id"
          type="button"
          class="dashboard-shell__nav-item"
          :class="{ active: item.id === activeSection }"
          @click="handleSelect(item.id)"
        >
          <span class="dashboard-shell__icon" aria-hidden="true">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6">
              <path
                v-for="(path, index) in resolveIcon(item.icon)"
                :key="`${item.id}-${index}`"
                :d="path"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </svg>
          </span>
          <span>{{ item.label }}</span>
        </button>
      </nav>
      <div class="dashboard-shell__sidebar-footer">
        <slot name="sidebar-footer" />
      </div>
      <button class="dashboard-shell__close" type="button" @click="isSidebarOpen = false">
        Chiudi
      </button>
    </aside>
    <div class="dashboard-shell__main">
      <header class="dashboard-shell__topbar">
        <div class="dashboard-shell__breadcrumbs" v-if="breadcrumbs.length">
          <span v-for="(crumb, index) in breadcrumbs" :key="crumb.label">
            <span>{{ crumb.label }}</span>
            <span v-if="index < breadcrumbs.length - 1" aria-hidden="true">/</span>
          </span>
        </div>
        <div class="dashboard-shell__header">
          <div>
            <p class="dashboard-shell__status" v-if="activeItem">{{ activeItem.label }}</p>
            <h1>{{ title }}</h1>
            <p class="dashboard-shell__subtitle">{{ subtitle }}</p>
          </div>
          <div class="dashboard-shell__actions">
            <div class="dashboard-shell__user" v-if="userName">
              <div class="dashboard-shell__avatar" aria-hidden="true">{{ userInitials }}</div>
              <div>
                <p class="dashboard-shell__user-label">Amministratore</p>
                <p class="dashboard-shell__user-name">{{ userName }}</p>
              </div>
            </div>
            <div class="dashboard-shell__toolbar">
              <slot name="top-actions" />
            </div>
          </div>
        </div>
      </header>
      <main class="dashboard-shell__content">
        <slot />
      </main>
    </div>
    <div v-if="isSidebarOpen" class="dashboard-shell__overlay" @click="isSidebarOpen = false"></div>
  </div>
</template>

<script setup>
import { computed, ref } from "vue";

const props = defineProps({
  title: { type: String, default: "" },
  subtitle: { type: String, default: "" },
  menuItems: { type: Array, default: () => [] },
  activeSection: { type: String, default: "" },
  userName: { type: String, default: "" },
  breadcrumbs: { type: Array, default: () => [] },
  variant: { type: String, default: "default" },
});

const emit = defineEmits(["select"]);
const isSidebarOpen = ref(false);

const variantClass = computed(() => `dashboard-shell--${props.variant}`);
const activeItem = computed(() =>
  props.menuItems.find((item) => item.id === props.activeSection) || null,
);

const userInitials = computed(() => {
  if (!props.userName) {
    return "";
  }
  return props.userName
    .split(" ")
    .map((word) => word[0])
    .join("")
    .slice(0, 2)
    .toUpperCase();
});

const ICONS = {
  calendar: ["M7 3v3", "M17 3v3", "M4 9h16v10H4z"],
  lock: ["M7 10V7a5 5 0 0 1 10 0v3", "M6 10h12v9H6z", "M12 14v2"],
  chart: ["M5 5v14", "M5 19h14", "M9 16V9", "M13 16v-6", "M17 16V7"],
  camera: ["M5 7h14l-2 10H7z", "M15 5l-2-2h-2l-2 2", "M12 11a3 3 0 1 1 0 6 3 3 0 0 1 0-6z"],
  history: ["M5 13a7 7 0 1 0 2-5.24", "M5 5v4H9", "M12 10v4l3 2"],
  users: ["M7 10a3 3 0 1 0 0-6 3 3 0 0 0 0 6z", "M17 10a3 3 0 1 0 0-6 3 3 0 0 0 0 6z", "M3 20v-1a4 4 0 0 1 4-4h1", "M20 20v-1a4 4 0 0 0-4-4h-1"],
  user: ["M12 12a4 4 0 1 0-0.01-8A4 4 0 0 0 12 12z", "M5 20a7 7 0 0 1 14 0"],
  sparkles: ["M12 3l1.2 3.4L16 7.5l-2.5 2 0.9 3.5-2.4-1.8-2.4 1.8 0.9-3.5L8 7.5l2.8-1.1L12 3z", "M5 14l0.6 1.7L7 16.4l-1.4 1.1L6.4 19 5 18l-1.4 1 0.4-1.5L2.6 16.4 4 15.7z", "M19 14l0.6 1.7L21 16.4l-1.4 1.1L20.4 19 19 18l-1.4 1 0.4-1.5-1.2-0.9 1.4-0.7z"],
  shield: ["M12 3l7 3v6c0 4.5-3 8.5-7 10-4-1.5-7-5.5-7-10V6z"],
  dashboard: ["M4 13h7v8H4z", "M13 3h7v8h-7z", "M13 13h7v8h-7z", "M4 3h7v8H4z"],
  ticket: ["M5 7h14v3a2 2 0 0 1 0 4v3H5v-3a2 2 0 0 0 0-4z", "M9 7v10"],
};

function resolveIcon(icon) {
  if (icon && ICONS[icon]) {
    return ICONS[icon];
  }
  return ["M4 6h16v12H4z", "M8 3v3", "M16 3v3"];
}

function handleSelect(id) {
  emit("select", id);
  isSidebarOpen.value = false;
}
</script>

<style scoped>
.dashboard-shell {
  position: relative;
  display: flex;
  gap: 1.5rem;
  min-height: calc(100vh - 3rem);
  padding: 1.5rem;
  border-radius: 1.5rem;
  background: linear-gradient(135deg, #0f172a, #111827 30%, #0b1120 90%);
  color: #0f172a;
}

.dashboard-shell__mobile-toggle {
  position: fixed;
  top: 1rem;
  left: 1rem;
  z-index: 30;
  border: none;
  border-radius: 999px;
  background: rgba(15, 23, 42, 0.85);
  color: #f8fafc;
  padding: 0.5rem 0.9rem;
  font-size: 1.1rem;
  display: none;
}

.dashboard-shell__sidebar {
  width: 260px;
  flex-shrink: 0;
  background: rgba(15, 23, 42, 0.85);
  border-radius: 1.25rem;
  padding: 1.5rem 1.25rem;
  color: #cbd5f5;
  display: flex;
  flex-direction: column;
  gap: 1rem;
  box-shadow: 0 25px 80px rgba(2, 6, 23, 0.65);
}

.dashboard-shell__brand h2 {
  margin: 0.2rem 0 0;
  font-size: 1.2rem;
  color: #f8fafc;
}

.dashboard-shell__eyebrow {
  margin: 0;
  font-size: 0.85rem;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: rgba(248, 250, 252, 0.7);
}

.dashboard-shell__nav {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.dashboard-shell__nav-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  width: 100%;
  border: none;
  border-radius: 0.75rem;
  padding: 0.65rem 0.75rem;
  background: transparent;
  color: inherit;
  font-size: 0.95rem;
  font-weight: 500;
  text-align: left;
  transition: background 0.2s ease, color 0.2s ease;
}

.dashboard-shell__nav-item.active,
.dashboard-shell__nav-item:hover {
  background: rgba(255, 255, 255, 0.12);
  color: #f8fafc;
}

.dashboard-shell__icon svg {
  width: 1.2rem;
  height: 1.2rem;
}

.dashboard-shell__sidebar-footer {
  margin-top: auto;
  font-size: 0.85rem;
  color: rgba(248, 250, 252, 0.75);
}

.dashboard-shell__close {
  display: none;
  margin-top: 1rem;
  border: 1px solid rgba(248, 250, 252, 0.4);
  padding: 0.45rem 0.9rem;
  border-radius: 999px;
  background: transparent;
  color: inherit;
}

.dashboard-shell__main {
  flex: 1;
  min-width: 0;
  background: rgba(248, 250, 252, 0.98);
  border-radius: 1.5rem;
  padding: 1.5rem;
  box-shadow: 0 35px 80px rgba(15, 23, 42, 0.2);
}

.dashboard-shell__topbar {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.dashboard-shell__breadcrumbs {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
  font-size: 0.85rem;
  color: #64748b;
}

.dashboard-shell__header {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.dashboard-shell__status {
  margin: 0;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: #94a3b8;
  font-size: 0.8rem;
}

.dashboard-shell__header h1 {
  margin: 0;
  font-size: 1.8rem;
  color: #0f172a;
}

.dashboard-shell__subtitle {
  margin: 0.35rem 0 0;
  color: #475569;
}

.dashboard-shell__actions {
  display: flex;
  gap: 1rem;
  align-items: center;
  flex-wrap: wrap;
}

.dashboard-shell__user {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.5rem 0.85rem;
  border-radius: 999px;
  background: rgba(15, 23, 42, 0.04);
}

.dashboard-shell__avatar {
  width: 2.4rem;
  height: 2.4rem;
  border-radius: 999px;
  background: #0f172a;
  color: #f8fafc;
  display: grid;
  place-items: center;
  font-weight: 600;
}

.dashboard-shell__user-label {
  margin: 0;
  font-size: 0.75rem;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: #94a3b8;
}

.dashboard-shell__user-name {
  margin: 0;
  font-size: 0.95rem;
  font-weight: 600;
  color: #0f172a;
}

.dashboard-shell__toolbar {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.dashboard-shell__content {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.dashboard-shell__overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.35);
  z-index: 20;
}

.dashboard-shell--master {
  background: linear-gradient(135deg, #0f172a, #0e7490);
}

.dashboard-shell--lottery {
  background: linear-gradient(135deg, #0f172a, #9333ea);
}

@media (max-width: 1024px) {
  .dashboard-shell {
    flex-direction: column;
    padding: 1rem;
  }

  .dashboard-shell__mobile-toggle {
    display: inline-flex;
  }

  .dashboard-shell__sidebar {
    position: fixed;
    top: 0;
    left: 0;
    bottom: 0;
    transform: translateX(-100%);
    width: 80%;
    max-width: 320px;
    z-index: 30;
    border-radius: 0;
  }

  .dashboard-shell__sidebar--open {
    transform: translateX(0);
  }

  .dashboard-shell__close {
    display: inline-flex;
  }
}
</style>
