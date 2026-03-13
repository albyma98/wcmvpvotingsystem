<template>
  <nav class="sidebar">
    <div class="sidebar__brand">
      <div class="sidebar__logo">
        <svg width="28" height="28" viewBox="0 0 28 28" fill="none">
          <circle cx="14" cy="14" r="13" stroke="url(#sg)" stroke-width="2"/>
          <path d="M9 14l3.5 3.5L19 10" stroke="#0284c7" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"/>
          <defs>
            <linearGradient id="sg" x1="0" y1="0" x2="28" y2="28" gradientUnits="userSpaceOnUse">
              <stop stop-color="#0284c7"/>
              <stop offset="1" stop-color="#6366f1"/>
            </linearGradient>
          </defs>
        </svg>
      </div>
      <div class="sidebar__brand-text">
        <span class="sidebar__brand-name">MVP Admin</span>
        <span v-if="organizationSlug" class="sidebar__org">{{ organizationSlug }}</span>
      </div>
    </div>

    <div class="sidebar__divider" />

    <div class="sidebar__nav">
      <div v-for="group in groups" :key="group.label" class="sidebar__group">
        <p class="sidebar__group-label">{{ group.label }}</p>
        <button
          v-for="item in group.items"
          :key="item.id"
          type="button"
          class="sidebar__item"
          :class="{ active: activeSection === item.id }"
          @click="$emit('select', item.id)"
        >
          <span class="sidebar__item-icon">{{ sectionIcons[item.id] || '◆' }}</span>
          <span class="sidebar__item-label">{{ item.label }}</span>
          <span v-if="activeSection === item.id" class="sidebar__item-indicator" />
        </button>
      </div>
    </div>

    <div class="sidebar__divider" />

    <div class="sidebar__footer">
      <button class="sidebar__action sidebar__action--lottery" type="button" @click="$emit('lottery')">
        <span>🎰</span> Lotteria
      </button>
      <button class="sidebar__action sidebar__action--logout" type="button" @click="$emit('logout')">
        <span>↩</span> Esci
      </button>
    </div>
  </nav>
</template>

<script setup>
defineProps({ groups: Array, activeSection: String, organizationSlug: String });
defineEmits(['select', 'lottery', 'logout']);

const sectionIcons = {
  dashboard: '⊞', events: '📅', sponsors: '🏷', coupons: '🎟',
  selfies: '📸', teams: '🏐', players: '👤', closing: '🔒',
  results: '📊', history: '🗂', bar: '🍺', partners: '🤝',
  admins: '🛡', marketing: '📣',
};
</script>

<style scoped>
.sidebar {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #ffffff;
}

.sidebar__brand {
  display: flex;
  align-items: center;
  gap: 0.85rem;
  padding: 1.5rem 1.25rem 1.25rem;
}

.sidebar__logo {
  flex-shrink: 0;
  width: 42px; height: 42px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #e0f2fe;
  border: 1px solid #bae6fd;
  border-radius: 12px;
}

.sidebar__brand-text { display: flex; flex-direction: column; gap: 0.1rem; }

.sidebar__brand-name {
  font-family: 'Barlow Condensed', 'Impact', sans-serif;
  font-size: 1.15rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: #0f172a;
}

.sidebar__org {
  font-size: 0.72rem;
  color: #0284c7;
  font-weight: 500;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.sidebar__divider {
  height: 1px;
  margin: 0 1.25rem;
  background: #f1f5f9;
}

.sidebar__nav {
  flex: 1;
  padding: 0.75rem;
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  overflow-y: auto;
  scrollbar-width: none;
}
.sidebar__nav::-webkit-scrollbar { display: none; }

.sidebar__group { display: flex; flex-direction: column; gap: 0.15rem; }

.sidebar__group-label {
  margin: 0 0 0.35rem 0.5rem;
  font-size: 0.64rem;
  font-weight: 700;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: #94a3b8;
}

.sidebar__item {
  position: relative;
  display: flex;
  align-items: center;
  gap: 0.65rem;
  width: 100%;
  padding: 0.58rem 0.75rem;
  border-radius: 10px;
  border: 0;
  background: transparent;
  color: #64748b;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  text-align: left;
  transition: all 0.18s ease;
  font-family: inherit;
  overflow: hidden;
}

.sidebar__item:hover {
  background: #f1f5f9;
  color: #0f172a;
}

.sidebar__item.active {
  background: #e0f2fe;
  color: #0284c7;
  border: 1px solid #bae6fd;
  font-weight: 600;
}

.sidebar__item-icon { font-size: 1rem; width: 1.25rem; text-align: center; flex-shrink: 0; }
.sidebar__item-label { flex: 1; font-family: 'IBM Plex Sans', system-ui, sans-serif; }

.sidebar__item-indicator {
  position: absolute;
  right: 0; top: 50%;
  transform: translateY(-50%);
  width: 3px; height: 60%;
  background: linear-gradient(180deg, #0284c7, #6366f1);
  border-radius: 3px 0 0 3px;
}

.sidebar__footer {
  padding: 0.75rem;
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.sidebar__action {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  width: 100%;
  padding: 0.6rem 0.75rem;
  border-radius: 10px;
  border: 0;
  font-size: 0.875rem;
  font-weight: 600;
  cursor: pointer;
  text-align: left;
  font-family: inherit;
  transition: all 0.18s ease;
}

.sidebar__action--lottery {
  background: #fef3c7;
  color: #92400e;
  border: 1px solid #fcd34d;
}
.sidebar__action--lottery:hover { background: #fde68a; }

.sidebar__action--logout {
  background: #fee2e2;
  color: #dc2626;
  border: 1px solid #fecaca;
}
.sidebar__action--logout:hover { background: #fecaca; }
</style>
