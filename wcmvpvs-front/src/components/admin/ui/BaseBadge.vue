<template>
  <span class="base-badge" :class="`base-badge--${variant}`"><slot /></span>
</template>
<script setup>
defineProps({ variant: { type: String, default: 'default' } });
</script>
<style scoped>
.base-badge {
  display: inline-flex;
  align-items: center;
  padding: 0.2rem 0.65rem;
  border-radius: 999px;
  font-size: 0.68rem;
  font-weight: 700;
  letter-spacing: 0.1em;
  text-transform: uppercase;
}
.base-badge--default { background: rgba(148,163,184,0.15); color: #94a3b8; }
.base-badge--success { background: rgba(5,150,105,0.18); color: #34d399; border: 1px solid rgba(52,211,153,0.2); }
.base-badge--danger  { background: rgba(239,68,68,0.15); color: #f87171; border: 1px solid rgba(248,113,113,0.2); }
.base-badge--warning { background: rgba(245,158,11,0.15); color: #d97706; border: 1px solid rgba(251,191,36,0.2); }
.base-badge--info    { background: rgba(56,189,248,0.15); color: #0284c7; border: 1px solid rgba(2,132,199,0.2); }
</style>
