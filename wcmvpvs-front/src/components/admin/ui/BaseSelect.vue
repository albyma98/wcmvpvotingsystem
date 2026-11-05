<template><div class="ui-field"><slot /></div></template>
<script setup></script>
<style scoped>
.ui-field { display: flex; flex-direction: column; gap: 7px; min-width: 0; }
.ui-field :deep(select) { width: 100%; min-width: 0; min-height: var(--control-height,42px); padding: 10px 30px 10px 12px; border: 1px solid #cbd3e1; border-radius: var(--radius-sm,8px); background: #fff; color: var(--text-primary,#182238); font: 400 14px/1.45 var(--font-body,'Segoe UI',sans-serif); appearance: auto; cursor: pointer; color-scheme: light; }
.ui-field :deep(option) { background: #fff; color: #182238; }
.ui-field :deep(select:focus-visible) { outline: 2px solid var(--accent,#4f46e5); outline-offset: 2px; border-color: var(--accent,#4f46e5); }
.ui-field :deep(select:disabled) { background: #f1f3f7; color: #606c80; cursor: not-allowed; }
@media(max-width:640px) { .ui-field :deep(select) { font-size: 16px; min-height: 44px; } }
</style>
