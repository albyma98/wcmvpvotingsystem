<template>
  <button :class="['base-button', `base-button--${variant}`, `base-button--${size}`]" v-bind="$attrs">
    <slot />
  </button>
</template>

<script setup>
defineProps({
  variant: { type: String, default: 'primary' },
  size: { type: String, default: 'md' },
});
</script>

<style scoped>
.base-button { border-radius: 999px; font-weight: 600; border: 1px solid transparent; cursor: pointer; }
.base-button--md { min-height: var(--control-height); padding: 0.625rem 1.2rem; }
.base-button--sm { min-height: 36px; padding: 0.45rem 0.85rem; font-size: 0.88rem; }
.base-button--primary { background: linear-gradient(135deg,#2563eb,#7c3aed); color: #fff; }
.base-button--secondary { background: #e2e8f0; color: #0f172a; }
.base-button--danger { background: #fff; color: #dc2626; border-color: #fecaca; }
.base-button--ghost { background: transparent; color: #334155; border-color: var(--ui-border); }
.base-button:disabled { opacity: .6; cursor: not-allowed; }
</style>
