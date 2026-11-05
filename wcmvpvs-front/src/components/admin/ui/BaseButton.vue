<template>
  <button
    :type="type"
    :disabled="disabled"
    class="base-btn"
    :class="[`base-btn--${variant}`, { 'base-btn--sm': size === 'sm' }]"
  >
    <slot />
  </button>
</template>
<script setup>
defineProps({
  variant: { type: String, default: 'primary' },
  type: { type: String, default: 'button' },
  disabled: { type: Boolean, default: false },
  size: { type: String, default: 'md' },
});
</script>
<style scoped>
.base-btn { display: inline-flex; align-items: center; justify-content: center; gap: 8px; min-height: var(--control-height,42px); max-width: 100%; padding: 10px 16px; border: 1px solid transparent; border-radius: var(--radius-sm,8px); font: 600 13px/1.4 var(--font-body,'Segoe UI',sans-serif); text-align: center; cursor: pointer; transition: background .15s,border-color .15s,box-shadow .15s; }
.base-btn:disabled { opacity: .58; cursor: not-allowed; box-shadow: none; }
.base-btn:focus-visible { outline: 3px solid var(--accent,#4f46e5); outline-offset: 3px; }
.base-btn--sm { min-height: 34px; padding: 7px 12px; font-size: 12px; }
.base-btn--primary { background: var(--accent,#4f46e5); color: #fff; border-color: var(--accent,#4f46e5); }
.base-btn--primary:not(:disabled):hover { background: var(--abx-primary-hover,#4338ca); border-color: var(--abx-primary-hover,#4338ca); }
.base-btn--secondary,.base-btn--outline { background: #fff; color: #3c4960; border-color: #cbd3e1; }
.base-btn:is(.base-btn--secondary,.base-btn--outline):not(:disabled):hover { background: #f5f4ff; border-color: #a5a0ed; color: #4338ca; }
.base-btn--danger { background: #fff1f2; color: #b42335; border-color: #fecdd3; }
.base-btn--danger:not(:disabled):hover { background: #ffe4e6; border-color: #fda4af; }
.base-btn--success { background: #087a55; color: #fff; border-color: #087a55; }
.base-btn--success:not(:disabled):hover { background: #066244; }
.base-btn--warning { background: #fffbeb; color: #92400e; border-color: #fde68a; }
.base-btn--warning:not(:disabled):hover { background: #fef3c7; }
.base-btn--link { background: transparent; color: var(--accent,#4f46e5); padding: 6px 8px; }
.base-btn--link:not(:disabled):hover { background: #eeedff; color: #4338ca; }
@media(max-width:640px) { .base-btn,.base-btn--sm { min-height: 44px; } }
</style>
