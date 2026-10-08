<template>
  <nav class="sidebar">
    <div class="sidebar__brand">
      <div class="sidebar__brand-text">
        <span class="sidebar__brand-name">ABX ADMIN</span>
        <span v-if="organizationName || organizationSlug" class="sidebar__org" :title="organizationName || organizationSlug">{{ organizationName || organizationSlug }}</span>
      </div>
    </div>

    <div class="sidebar__divider" />

    <div class="sidebar__nav">
      <div v-for="group in groups" :key="group.label" class="sidebar__group">
        <p class="sidebar__group-label">{{ group.label }}</p>
        <div class="sidebar__group-items">
          <button
            v-for="item in group.items"
            :key="item.id"
            type="button"
            class="sidebar__item"
            :class="{ active: activeSection === item.id }"
            :aria-pressed="activeSection === item.id"
            @click="$emit('select', item.id)"
          >
            <span class="sidebar__item-label">{{ item.label }}</span>
          </button>
        </div>
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
defineProps({ groups: Array, activeSection: String, organizationSlug: String, organizationName: String });
defineEmits(['select', 'lottery', 'logout']);
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
  min-height: 74px;
  padding: 0.8rem 1rem;
}

.sidebar__brand-text { display: flex; flex-direction: column; min-width: 0; gap: 0.15rem; }

.sidebar__brand-name {
  font-family: 'Barlow Condensed', 'Impact', sans-serif;
  font-size: 1.25rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: #0f172a;
}

.sidebar__org {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  font-size: 0.82rem;
  color: #0284c7;
  font-weight: 600;
  line-height: 1.2;
  overflow: hidden;
  overflow-wrap: anywhere;
}

.sidebar__divider {
  height: 1px;
  margin: 0 1.25rem;
  background: #f1f5f9;
}

.sidebar__nav {
  flex: 1;
  min-height: 0;
  padding: 0.55rem 0.7rem;
  display: flex;
  flex-direction: column;
  gap: 0.55rem;
  overflow: hidden;
}

.sidebar__group { min-width: 0; }

.sidebar__group-items {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.25rem;
}

.sidebar__group-items > :only-child { grid-column: 1 / -1; }

.sidebar__group-label {
  margin: 0 0 0.25rem 0.3rem;
  font-size: 0.62rem;
  font-weight: 700;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: #94a3b8;
}

.sidebar__item {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 38px;
  width: 100%;
  padding: 0.35rem 0.4rem;
  border-radius: 8px;
  border: 1px solid transparent;
  background: transparent;
  color: #64748b;
  font-size: 0.75rem;
  line-height: 1.15;
  font-weight: 500;
  cursor: pointer;
  text-align: center;
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
  border-color: #bae6fd;
  font-weight: 600;
}

.sidebar__item-label { font-family: 'IBM Plex Sans', system-ui, sans-serif; }

.sidebar__footer {
  padding: 0.55rem 0.7rem;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.4rem;
}

.sidebar__action {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.35rem;
  min-height: 36px;
  width: 100%;
  padding: 0.35rem 0.4rem;
  border-radius: 8px;
  font-size: 0.78rem;
  font-weight: 600;
  cursor: pointer;
  text-align: center;
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

@media (max-height: 720px) {
  .sidebar__nav { overflow-y: auto; }
}
</style>
