<template>
  <span class="base-badge" :class="`base-badge--${variant}`"><slot /></span>
</template>
<script setup>
defineProps({ variant: { type: String, default: 'default' } });
</script>
<style scoped>
.base-badge { display: inline-flex; align-items: center; gap: 5px; padding: 4px 8px; border: 1px solid #e0e4ee; border-radius: 6px; font-size: 11px; line-height: 1.4; font-weight: 600; }
.base-badge--default { background: #f4f5f9; color: #526078; }
.base-badge--success { background: #ecfdf5; color: #087a55; border-color: #bcebd6; }
.base-badge--danger { background: #fff1f2; color: #b42335; border-color: #fecdd3; }
.base-badge--warning { background: #fffbeb; color: #92400e; border-color: #f5df9d; }
.base-badge--info { background: #eeedff; color: #4338ca; border-color: #dcd8ff; }
</style>
