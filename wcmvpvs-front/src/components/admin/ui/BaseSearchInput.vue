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
.base-search{display:flex;flex-direction:column;gap:.45rem}.base-search input{min-height:var(--control-height)}
</style>
