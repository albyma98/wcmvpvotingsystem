<template>
  <nav class="sidebar" :class="{ 'sidebar--collapsed': collapsed }" aria-label="Navigazione amministrazione società">
    <div class="sidebar__brand">
      <span class="sidebar__brand-mark" aria-hidden="true"><AdminIcon name="boost" :size="21" /></span>
      <div class="sidebar__brand-text">
        <span class="sidebar__brand-name">ArenaBoost<span>X</span></span>
        <span class="sidebar__brand-caption">CLUB WORKSPACE</span>
      </div>
    </div>

    <div class="sidebar__workspace" :title="organizationName || organizationSlug || 'La tua società'">
      <span class="sidebar__org-avatar" aria-hidden="true">{{ organizationInitials }}</span>
      <div class="sidebar__workspace-text">
        <span class="sidebar__org">{{ organizationName || organizationSlug || 'La tua società' }}</span>
        <span class="sidebar__org-caption">Pannello di gestione</span>
      </div>
      <AdminIcon class="sidebar__workspace-chevron" name="chevrons" :size="15" />
    </div>

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
            :aria-current="activeSection === item.id ? 'page' : undefined"
            :aria-label="item.label"
            :title="item.label"
            @click="$emit('select', item.id)"
          >
            <AdminIcon class="sidebar__item-icon" :name="item.id" :size="18" />
            <span class="sidebar__item-label">{{ item.label }}</span>
            <span v-if="activeSection === item.id" class="sidebar__active-dot" aria-hidden="true" />
          </button>
        </div>
      </div>
    </div>

    <div class="sidebar__footer">
      <button v-if="showLottery" class="sidebar__action sidebar__action--lottery" type="button" aria-label="Lotteria" title="Lotteria" @click="$emit('lottery')">
        <AdminIcon name="lottery" :size="18" /><span class="sidebar__action-label">Lotteria</span><AdminIcon class="sidebar__external" name="arrow-up-right" :size="15" />
      </button>
      <button class="sidebar__action sidebar__action--logout" type="button" aria-label="Esci" title="Esci" @click="$emit('logout')">
        <AdminIcon name="logout" :size="18" /><span class="sidebar__action-label">Esci dal workspace</span>
      </button>
      <button
        class="sidebar__collapse"
        type="button"
        :aria-label="collapsed ? 'Espandi menu' : 'Comprimi menu'"
        :aria-expanded="!collapsed"
        :title="collapsed ? 'Espandi menu' : 'Comprimi menu'"
        @click="$emit('toggle-collapse')"
      >
        <AdminIcon :name="collapsed ? 'chevron-right' : 'chevron-left'" :size="17" /><span class="sidebar__action-label">Comprimi menu</span>
      </button>
    </div>
  </nav>
</template>

<script setup>
import { computed } from 'vue';
import AdminIcon from './ui/AdminIcon.vue';
const props = defineProps({
  groups: { type: Array, default: () => [] },
  activeSection: String,
  organizationSlug: String,
  organizationName: String,
  showLottery: { type: Boolean, default: true },
  collapsed: { type: Boolean, default: false },
});
defineEmits(['select', 'lottery', 'logout', 'toggle-collapse']);
const organizationInitials = computed(() => (props.organizationName || props.organizationSlug || 'AB')
  .split(/[\s-]+/).filter(Boolean).slice(0, 2).map(word => word[0]).join('').toUpperCase());
</script>

<style scoped>
.sidebar { display: flex; flex-direction: column; height: 100%; min-width: 0; background: #111827; color: #e5e7eb; font-family: var(--font-body, 'Inter', system-ui, sans-serif); }
.sidebar__brand { display: flex; align-items: center; gap: 10px; flex-shrink: 0; min-height: 76px; padding: 16px 23px; }
.sidebar__brand-mark { display: grid; place-items: center; flex: 0 0 32px; width: 32px; height: 34px; color: white; background: #6366f1; border-radius: 10px; box-shadow: 0 3px 12px #6366f130; }
.sidebar__brand-text { display: flex; flex-direction: column; min-width: 0; gap: 4px; }
.sidebar__brand-name { color: #f8fafc; font-size: 16px; font-weight: 750; letter-spacing: -0.65px; white-space: nowrap; line-height: 1; }
.sidebar__brand-name > span { color: #a5b4fc; }
.sidebar__brand-caption { color: #9ca7bc; font-size: 9px; font-weight: 650; letter-spacing: 1.6px; line-height: 1.2; }
.sidebar__workspace { display: flex; align-items: center; gap: 9px; min-width: 0; margin: 3px 15px 19px; padding: 11px 10px; border: 1px solid #ffffff12; border-radius: 9px; background: #ffffff05; }
.sidebar__org-avatar { display: grid; place-items: center; flex: 0 0 31px; height: 31px; border-radius: 7px; color: #dbe3ff; background: #303b58; font-size: 11px; font-weight: 650; }
.sidebar__workspace-text { min-width: 0; display: flex; flex: 1; flex-direction: column; gap: 4px; }
.sidebar__org { font-size: 12px; font-weight: 650; line-height: 1.3; color: #eef2ff; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.sidebar__org-caption { color: #a0abc0; font-size: 10px; white-space: nowrap; }
.sidebar__workspace-chevron { color: #8c98ae; flex-shrink: 0; }
.sidebar__nav { display: flex; flex: 1; min-height: 0; flex-direction: column; gap: 19px; padding: 0 12px 20px; overflow-y: auto; overflow-x: hidden; scrollbar-width: thin; scrollbar-color: #394156 transparent; }
.sidebar__group { min-width: 0; }
.sidebar__group-label { margin: 0 0 7px 13px; color: #95a1b9; font-size: 10px; font-weight: 650; letter-spacing: 1.3px; text-transform: uppercase; }
.sidebar__group-items { display: grid; gap: 3px; }
.sidebar__item { position: relative; display: flex; align-items: center; gap: 11px; width: 100%; min-height: 39px; padding: 9px 12px; border: 1px solid transparent; border-radius: 7px; background: transparent; color: #b6c0d3; font-family: inherit; font-size: 12px; line-height: 1.3; font-weight: 500; cursor: pointer; text-align: left; transition: background 150ms, color 150ms; }
.sidebar__item-icon { flex-shrink: 0; color: #8e9bb4; }
.sidebar__item:hover { background: #ffffff08; color: #fff; }
.sidebar__item:hover .sidebar__item-icon { color: #dbe2f4; }
.sidebar__item.active { background: #4f46e5; color: white; font-weight: 600; box-shadow: 0 3px 10px #0000000c; }
.sidebar__item.active .sidebar__item-icon { color: #e0e7ff; }
.sidebar__item-label { flex: 1; min-width: 0; overflow-wrap: anywhere; }
.sidebar__active-dot { width: 5px; height: 5px; border-radius: 50%; background: #c7d2fe; flex-shrink: 0; }
.sidebar__footer { display: grid; gap: 3px; margin: 0 12px; padding: 12px 0 14px; border-top: 1px solid #ffffff0e; }
.sidebar__action, .sidebar__collapse { display: flex; align-items: center; gap: 11px; min-height: 38px; width: 100%; padding: 9px 12px; color: #b6c0d3; border: 0; border-radius: 7px; background: transparent; cursor: pointer; font-family: inherit; font-size: 12px; font-weight: 500; text-align: left; }
.sidebar__action > svg, .sidebar__collapse > svg { flex-shrink: 0; }
.sidebar__action:hover, .sidebar__collapse:hover { color: white; background: #ffffff08; }
.sidebar__action--lottery { color: #c7d2fe; }
.sidebar__action--logout:hover { color: #fda4af; background: #fb71850a; }
.sidebar__external { margin-left: auto; opacity: .8; }
.sidebar__collapse { color: #98a5bd; margin-top: 6px; padding-top: 10px; border-top: 1px solid #ffffff0e; border-radius: 0 0 7px 7px; }
.sidebar :is(button):focus-visible { outline: 2px solid #a5b4fc; outline-offset: 2px; }
@media (min-width: 1024px) {
  .sidebar--collapsed .sidebar__brand { justify-content: center; padding-inline: 12px; }
  .sidebar--collapsed .sidebar__brand-text, .sidebar--collapsed .sidebar__workspace-text, .sidebar--collapsed .sidebar__workspace-chevron, .sidebar--collapsed .sidebar__item-label, .sidebar--collapsed .sidebar__action-label, .sidebar--collapsed .sidebar__external, .sidebar--collapsed .sidebar__active-dot { display: none; }
  .sidebar--collapsed .sidebar__workspace { justify-content: center; margin: 3px 12px 20px; padding: 10px 0; }
  .sidebar--collapsed .sidebar__group-label { height: 1px; margin: 4px 10px 10px; overflow: hidden; background: #ffffff12; color: transparent; }
  .sidebar--collapsed .sidebar__nav { gap: 12px; }
  .sidebar--collapsed .sidebar__item, .sidebar--collapsed .sidebar__action, .sidebar--collapsed .sidebar__collapse { justify-content: center; padding-inline: 0; }
}
@media (max-width: 1023px) { .sidebar__collapse { display: none; } .sidebar__brand { padding-right: 50px; } .sidebar__item, .sidebar__action { min-height: 44px; font-size: 13px; } .sidebar__org { font-size: 12px; } .sidebar__group-label { font-size: 10px; } }
@media (prefers-reduced-motion: reduce) { .sidebar__item { transition: none; } }
</style>
