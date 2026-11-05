<template>
  <header class="admin-header">
    <div class="admin-header__left">
      <button
        class="admin-header__burger"
        type="button"
        aria-label="Apri menu di navigazione"
        aria-controls="admin-navigation"
        :aria-expanded="menuOpen"
        @click="$emit('toggle-menu')"
      >
        <AdminIcon name="menu" />
      </button>
      <div class="admin-header__breadcrumb" aria-label="Sezione corrente">
        <span class="admin-header__workspace">Workspace</span>
        <AdminIcon class="admin-header__sep" name="chevron-right" :size="14" />
        <span v-if="sectionGroup" class="admin-header__group">{{ sectionGroup }}</span>
        <AdminIcon v-if="sectionGroup" class="admin-header__sep" name="chevron-right" :size="14" />
        <h1 class="admin-header__title">{{ title }}</h1>
      </div>
    </div>
    <div class="admin-header__right">
      <slot name="actions" />
    </div>
  </header>
</template>

<script setup>
import AdminIcon from './ui/AdminIcon.vue';
defineProps({ title: String, sectionGroup: String, menuOpen: { type: Boolean, default: false } });
defineEmits(['toggle-menu']);
</script>

<style scoped>
.admin-header { position: sticky; top: 0; z-index: 30; min-height: 72px; display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 12px 36px; background: #ffffffed; backdrop-filter: blur(16px); border-bottom: 1px solid #e7eaf1; font-family: var(--font-body, 'Inter', system-ui, sans-serif); }
.admin-header__left { display: flex; align-items: center; gap: 12px; min-width: 0; }
.admin-header__burger { display: none; align-items: center; justify-content: center; flex: 0 0 40px; width: 40px; height: 40px; padding: 0; border: 1px solid #dfe3ec; border-radius: 9px; background: white; color: #475569; cursor: pointer; }
.admin-header__burger:hover { background: #f5f5ff; color: #4f46e5; border-color: #c7d2fe; }
.admin-header__burger:focus-visible { outline: 3px solid #a5b4fc; outline-offset: 2px; }
.admin-header__breadcrumb { display: flex; align-items: center; gap: 10px; min-width: 0; }
.admin-header__workspace, .admin-header__group { font-size: 12px; font-weight: 500; color: #647087; white-space: nowrap; }
.admin-header__sep { color: #a7afbe; flex-shrink: 0; }
.admin-header__title { margin: 0; color: #1e293b; font-family: inherit; font-size: 13px; line-height: 1.4; font-weight: 650; letter-spacing: -0.015em; text-transform: none; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.admin-header__right { display: flex; align-items: center; justify-content: flex-end; gap: 12px; min-width: 0; flex-shrink: 0; }
@media (max-width: 1199px) { .admin-header { padding-inline: 24px; } }
@media (max-width: 1023px) { .admin-header__burger { display: flex; } .admin-header__workspace, .admin-header__workspace + .admin-header__sep { display: none; } }
@media (max-width: 599px) { .admin-header { min-height: 68px; padding: 10px 16px; gap: 8px; } .admin-header__group, .admin-header__group + .admin-header__sep { display: none; } .admin-header__right { gap: 8px; max-width: 45%; } .admin-header__breadcrumb { gap: 6px; } }
</style>
