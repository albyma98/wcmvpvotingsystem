<template><div class="ui-field base-table" role="region" aria-label="Tabella dati" tabindex="0"><slot /></div></template>
<script setup></script>
<style scoped>
.base-table { display: block; width: 100%; min-width: 0; overflow-x: auto; border: 1px solid var(--border,#e3e7ef); border-radius: var(--radius-md,12px); background: #fff; -webkit-overflow-scrolling: touch; }
.base-table:focus-visible { outline: 3px solid var(--accent,#4f46e5); outline-offset: 3px; }
.base-table :deep(table) { width: 100%; border-collapse: collapse; color: var(--text-primary,#182238); font-size: 13px; }
.base-table :deep(th),.base-table :deep(td) { padding: 13px 16px; border-bottom: 1px solid var(--border,#e3e7ef); text-align: left; }
.base-table :deep(th) { background: #f8f9fc; color: #526078; font-size: 12px; font-weight: 600; white-space: nowrap; }
.base-table :deep(tbody tr:hover) { background: #fafaff; }
.base-table :deep(tbody tr:last-child td) { border-bottom: 0; }
</style>
