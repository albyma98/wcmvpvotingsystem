<template>
  <header class="admin-header">
    <div class="admin-header__left">
      <button class="admin-header__burger" type="button" @click="$emit('toggle-menu')" aria-label="Menu">
        <span /><span /><span />
      </button>
      <div class="admin-header__breadcrumb">
        <span class="admin-header__group">{{ sectionGroup }}</span>
        <span class="admin-header__sep">/</span>
        <h1 class="admin-header__title">{{ title }}</h1>
      </div>
    </div>
    <div class="admin-header__right">
      <slot name="actions" />
    </div>
  </header>
</template>

<script setup>
defineProps({ title: String, sectionGroup: String });
defineEmits(['toggle-menu']);
</script>

<style scoped>
.admin-header {
  position: sticky;
  top: 0;
  z-index: 30;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.85rem 2rem;
  background: rgba(255, 255, 255, 0.92);
  backdrop-filter: blur(16px);
  border-bottom: 1px solid #f1f5f9;
  box-shadow: 0 1px 4px rgba(15, 23, 42, 0.06);
}

.admin-header__left { display: flex; align-items: center; gap: 1rem; }

.admin-header__burger {
  display: none;
  flex-direction: column;
  justify-content: center;
  gap: 4px;
  width: 36px; height: 36px;
  padding: 0;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  background: #f8fafc;
  cursor: pointer;
  align-items: center;
}

.admin-header__burger span {
  display: block;
  width: 16px; height: 2px;
  background: #64748b;
  border-radius: 2px;
  transition: background 0.2s;
}

.admin-header__burger:hover span { background: #0284c7; }

.admin-header__breadcrumb { display: flex; align-items: center; gap: 0.5rem; }

.admin-header__group {
  font-size: 0.78rem;
  font-weight: 600;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: #94a3b8;
}

.admin-header__sep { color: #cbd5e1; font-weight: 300; }

.admin-header__title {
  margin: 0;
  font-family: 'Barlow Condensed', 'Impact', sans-serif;
  font-size: 1.3rem;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: #0f172a;
}

.admin-header__right { display: flex; align-items: center; gap: 0.75rem; }

@media (max-width: 1023px) {
  .admin-header__burger { display: flex; }
  .admin-header { padding: 0.85rem 1rem; }
}
</style>
