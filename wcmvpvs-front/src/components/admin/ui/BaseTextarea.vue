<template><div class="ui-field"><slot /></div></template>
<script setup></script>
<style scoped>
.ui-field { display: flex; flex-direction: column; gap: 7px; min-width: 0; }
.ui-field :deep(textarea) { box-sizing: border-box; width: 100%; min-width: 0; min-height: 104px; padding: 10px 12px; border: 1px solid #cbd3e1; border-radius: var(--radius-sm,8px); background: #fff; color: var(--text-primary,#182238); font: 400 14px/1.55 var(--font-body,'Segoe UI',sans-serif); resize: vertical; }
.ui-field :deep(textarea::placeholder) { color: #66738a; opacity: 1; }
.ui-field :deep(textarea:focus-visible) { outline: 2px solid var(--accent,#4f46e5); outline-offset: 2px; border-color: var(--accent,#4f46e5); }
.ui-field :deep(textarea:disabled) { background: #f1f3f7; color: #606c80; cursor: not-allowed; }
@media(max-width:640px) { .ui-field :deep(textarea) { font-size: 16px; } }
</style>
