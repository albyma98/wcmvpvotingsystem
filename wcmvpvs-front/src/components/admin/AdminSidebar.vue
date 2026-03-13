<template>
  <nav class="sidebar">
    <!-- Brand -->
    <div class="sidebar__brand">
      <div class="sidebar__logo">
        <svg width="28" height="28" viewBox="0 0 28 28" fill="none">
          <circle cx="14" cy="14" r="13" stroke="url(#sg)" stroke-width="2"/>
          <path d="M9 14l3.5 3.5L19 10" stroke="#38bdf8" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"/>
          <defs>
            <linearGradient id="sg" x1="0" y1="0" x2="28" y2="28" gradientUnits="userSpaceOnUse">
              <stop stop-color="#38bdf8"/>
              <stop offset="1" stop-color="#818cf8"/>
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

    <!-- Navigation groups -->
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

    <!-- Footer actions -->
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
  dashboard: '⊞',
  events: '📅',
  sponsors: '🏷',
  coupons: '🎟',
  selfies: '📸',
  teams: '🏐',
  players: '👤',
  closing: '🔒',
  results: '📊',
  history: '🗂',
  bar: '🍺',
  partners: '🤝',
  admins: '🛡',
  marketing: '📣',
};
</script>

<style scoped>
.sidebar {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 0;
  background: #0d1424;
}

.sidebar__brand {
  display: flex;
  align-items: center;
  gap: 0.85rem;
  padding: 1.5rem 1.25rem 1.25rem;
}

.sidebar__logo {
  flex-shrink: 0;
  width: 42px;
  height: 42px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(56, 189, 248, 0.1);
  border: 1px solid rgba(56, 189, 248, 0.2);
  border-radius: 12px;
}

.sidebar__brand-text {
  display: flex;
  flex-direction: column;
  gap: 0.1rem;
}

.sidebar__brand-name {
  font-family: 'Barlow Condensed', 'Impact', sans-serif;
  font-size: 1.15rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: #f0f6ff;
}

.sidebar__org {
  font-size: 0.72rem;
  color: #38bdf8;
  font-weight: 500;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  opacity: 0.85;
}

.sidebar__divider {
  height: 1px;
  margin: 0 1.25rem;
  background: linear-gradient(90deg, transparent, rgba(56, 189, 248, 0.15), transparent);
}

.sidebar__nav {
  flex: 1;
  padding: 0.75rem 0.75rem;
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  overflow-y: auto;
  scrollbar-width: none;
}

.sidebar__nav::-webkit-scrollbar { display: none; }

.sidebar__group {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}

.sidebar__group-label {
  margin: 0 0 0.4rem 0.5rem;
  font-size: 0.65rem;
  font-weight: 700;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: rgba(148, 163, 184, 0.5);
}

.sidebar__item {
  position: relative;
  display: flex;
  align-items: center;
  gap: 0.65rem;
  width: 100%;
  padding: 0.6rem 0.75rem;
  border-radius: 10px;
  border: 0;
  background: transparent;
  color: rgba(148, 163, 184, 0.75);
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  text-align: left;
  transition: all 0.18s ease;
  font-family: inherit;
  overflow: hidden;
}

.sidebar__item:hover {
  background: rgba(56, 189, 248, 0.08);
  color: #e2eaf6;
}

.sidebar__item.active {
  background: linear-gradient(135deg, rgba(56, 189, 248, 0.18), rgba(129, 140, 248, 0.12));
  color: #f0f6ff;
  border: 1px solid rgba(56, 189, 248, 0.2);
}

.sidebar__item-icon {
  font-size: 1rem;
  width: 1.25rem;
  text-align: center;
  flex-shrink: 0;
}

.sidebar__item-label {
  flex: 1;
  font-family: 'IBM Plex Sans', system-ui, sans-serif;
}

.sidebar__item-indicator {
  position: absolute;
  right: 0;
  top: 50%;
  transform: translateY(-50%);
  width: 3px;
  height: 60%;
  background: linear-gradient(180deg, #38bdf8, #818cf8);
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
  background: rgba(250, 204, 21, 0.1);
  color: #fbbf24;
  border: 1px solid rgba(251, 191, 36, 0.15);
}

.sidebar__action--lottery:hover {
  background: rgba(250, 204, 21, 0.18);
}

.sidebar__action--logout {
  background: rgba(248, 113, 113, 0.1);
  color: #f87171;
  border: 1px solid rgba(248, 113, 113, 0.15);
}

.sidebar__action--logout:hover {
  background: rgba(248, 113, 113, 0.18);
}
</style>
