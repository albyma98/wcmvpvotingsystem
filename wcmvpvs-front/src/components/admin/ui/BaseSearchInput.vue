<template>
  <label class="base-search">
    <span v-if="label">{{ label }}</span>
    <input v-bind="$attrs" :value="modelValue" type="search" @input="$emit('update:modelValue', $event.target.value)" />
  </label>
</template>
<script setup>
defineProps({ modelValue: { type: String, default: '' }, label: { type: String, default: '' } });
defineEmits(['update:modelValue']);
</script>
<style scoped>
.base-search { position: relative; display: flex; flex-direction: column; gap: 7px; min-width: 0; color: var(--text-primary,#182238); font-size: 13px; font-weight: 500; }
.base-search input { width: 100%; min-width: 0; min-height: var(--control-height,42px); border: 1px solid #cbd3e1; border-radius: 8px; padding: 10px 12px; background: #fff; color: var(--text-primary,#182238); font: 400 14px/1.45 var(--font-body,'Segoe UI',sans-serif); }
.base-search input::placeholder { color: #66738a; opacity: 1; }
.base-search input:focus-visible { outline: 2px solid var(--accent,#4f46e5); outline-offset: 2px; border-color: var(--accent,#4f46e5); }
@media(max-width:640px) { .base-search input { min-height: 44px; font-size: 16px; } }
</style>
