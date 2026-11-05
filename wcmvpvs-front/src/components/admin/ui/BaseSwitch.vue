<template><div class="ui-field"><slot /></div></template>
<script setup></script>
<style scoped>
.ui-field { display: flex; flex-direction: column; gap: 7px; min-width: 0; }
.ui-field :deep(label) { display: flex; align-items: center; gap: 9px; min-height: var(--control-height,42px); color: var(--text-primary,#182238); font-size: 13px; cursor: pointer; }
.ui-field :deep(input) { width: 18px; height: 18px; min-height: 18px; margin: 0; padding: 0; flex: 0 0 18px; accent-color: var(--accent,#4f46e5); cursor: pointer; }
.ui-field :deep(input:focus-visible) { outline: 3px solid var(--accent,#4f46e5); outline-offset: 3px; }
.ui-field :deep(label:has(input:disabled)) { color: #606c80; cursor: not-allowed; }
</style>
