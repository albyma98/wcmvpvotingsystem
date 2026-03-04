<template>
  <nav class="sidebar">
    <div class="sidebar__brand">
      <h2>Admin SaaS</h2>
      <p v-if="organizationSlug">{{ organizationSlug }}</p>
    </div>
    <div v-for="group in groups" :key="group.label" class="sidebar__group">
      <h3>{{ group.label }}</h3>
      <button
        v-for="item in group.items"
        :key="item.id"
        type="button"
        class="sidebar__item"
        :class="{ active: activeSection === item.id }"
        @click="$emit('select', item.id)"
      >
        {{ item.label }}
      </button>
    </div>
    <div class="sidebar__footer">
      <button class="sidebar__action" type="button" @click="$emit('lottery')">Lotteria</button>
      <button class="sidebar__action secondary" type="button" @click="$emit('logout')">Esci</button>
    </div>
  </nav>
</template>

<script setup>
defineProps({ groups: Array, activeSection: String, organizationSlug: String });
defineEmits(["select", "lottery", "logout"]);
</script>

<style scoped>
.sidebar { padding: 1.25rem 1rem; display:flex; flex-direction:column; min-height:100%; gap:1rem; }
.sidebar__brand h2 { margin:0; color:#fff; }
.sidebar__brand p { margin:.2rem 0 0; color:#94a3b8; font-size:.9rem; }
.sidebar__group h3 { margin:0 0 .5rem; font-size:.78rem; text-transform:uppercase; letter-spacing:.08em; color:#94a3b8; }
.sidebar__item,.sidebar__action { width:100%; text-align:left; border:0; background:transparent; color:#cbd5e1; padding:.6rem .75rem; border-radius:.65rem; cursor:pointer; }
.sidebar__item.active,.sidebar__item:hover,.sidebar__action:hover { background:rgba(148,163,184,.2); color:#fff; }
.sidebar__footer { margin-top:auto; display:grid; gap:.5rem; }
.sidebar__action.secondary { background: rgba(239,68,68,.15); color:#fecaca; }
</style>
