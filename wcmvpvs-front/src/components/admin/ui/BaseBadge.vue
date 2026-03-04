<template><span :class="['base-badge', `base-badge--${status}`]"><slot>{{ label }}</slot></span></template>
<script setup>
defineProps({ label: { type: String, default: '' }, status: { type: String, default: 'draft' } });
</script>
<style scoped>
.base-badge{display:inline-flex;padding:.2rem .6rem;border-radius:999px;font-size:.72rem;font-weight:700;text-transform:uppercase;letter-spacing:.06em;background:#e2e8f0;color:#334155}
.base-badge--active{background:#dcfce7;color:#166534}.base-badge--paused{background:#fef3c7;color:#92400e}.base-badge--archived{background:#e2e8f0;color:#475569}.base-badge--draft{background:#dbeafe;color:#1d4ed8}
</style>
