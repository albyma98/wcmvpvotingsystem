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
.base-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.4rem;
  border-radius: 8px;
  border: 0;
  padding: 0.55rem 1.2rem;
  font-family: 'IBM Plex Sans', system-ui, sans-serif;
  font-size: 0.875rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.18s ease;
  letter-spacing: 0.02em;
}
.base-btn:disabled { opacity: 0.5; cursor: not-allowed; pointer-events: none; }
.base-btn--sm { padding: 0.35rem 0.85rem; font-size: 0.8rem; }

.base-btn--primary {
  background: linear-gradient(135deg, #0284c7, #4f46e5);
  color: #fff;
  box-shadow: 0 4px 20px rgba(2, 132, 199, 0.35);
}
.base-btn--primary:hover { box-shadow: 0 6px 28px rgba(2,132,199,0.5); transform: translateY(-1px); }

.base-btn--secondary {
  background: rgba(56,189,248,0.1);
  color: #38bdf8;
  border: 1px solid rgba(56,189,248,0.2);
}
.base-btn--secondary:hover { background: rgba(56,189,248,0.18); }

.base-btn--danger {
  background: rgba(239,68,68,0.15);
  color: #f87171;
  border: 1px solid rgba(239,68,68,0.2);
}
.base-btn--danger:hover { background: rgba(239,68,68,0.25); }

.base-btn--success {
  background: linear-gradient(135deg, #059669, #0d9488);
  color: #fff;
  box-shadow: 0 4px 20px rgba(5,150,105,0.3);
}
.base-btn--success:hover { box-shadow: 0 6px 28px rgba(5,150,105,0.45); transform: translateY(-1px); }

.base-btn--warning {
  background: linear-gradient(135deg, #d97706, #b45309);
  color: #fff;
}
.base-btn--outline {
  background: transparent;
  color: #94a3b8;
  border: 1px solid rgba(148,163,184,0.25);
}
.base-btn--outline:hover { border-color: rgba(56,189,248,0.4); color: #38bdf8; }

.base-btn--link {
  background: transparent;
  color: #38bdf8;
  padding: 0.2rem 0.4rem;
}
.base-btn--link:hover { color: #7dd3fc; }
</style>
