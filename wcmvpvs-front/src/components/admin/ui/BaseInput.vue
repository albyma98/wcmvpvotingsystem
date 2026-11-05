<template>
  <div v-if="$slots.default" class="ui-field" v-bind="$attrs"><slot /></div>
  <input v-else class="base-input" v-bind="$attrs" />
</template>
<script setup>
defineOptions({ inheritAttrs: false });
</script>
<style scoped>
.ui-field { display: flex; flex-direction: column; gap: 7px; min-width: 0; }
.base-input,.ui-field :deep(input:not([type="checkbox"]):not([type="radio"])) { box-sizing: border-box; width: 100%; min-width: 0; min-height: var(--control-height,42px); padding: 10px 12px; background: #fff; border: 1px solid #cbd3e1; border-radius: var(--radius-sm,8px); color: var(--text-primary,#182238); font: 400 14px/1.45 var(--font-body,'Segoe UI',sans-serif); transition: border-color .15s,box-shadow .15s; color-scheme: light; }
.base-input::placeholder,.ui-field :deep(input::placeholder) { color: #66738a; opacity: 1; }
.base-input:focus,.ui-field :deep(input:focus) { outline: 2px solid var(--accent,#4f46e5); outline-offset: 2px; border-color: var(--accent,#4f46e5); box-shadow: 0 0 0 3px rgba(79,70,229,.12); }
.base-input:disabled,.ui-field :deep(input:disabled) { background: #f1f3f7; color: #606c80; cursor: not-allowed; }
@media(max-width:640px) { .base-input,.ui-field :deep(input) { font-size: 16px; min-height: 44px; } }
</style>
