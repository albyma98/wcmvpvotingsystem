<template>
  <input class="base-input" v-bind="$attrs" />
</template>
<script setup>
defineOptions({ inheritAttrs: false });
</script>
<style scoped>
.base-input {
  width: 100%;
  padding: 0.6rem 0.9rem;
  background: rgba(15,23,42,0.6);
  border: 1px solid rgba(56,189,248,0.15);
  border-radius: 8px;
  color: #e2eaf6;
  font-family: 'IBM Plex Sans', system-ui, sans-serif;
  font-size: 0.875rem;
  transition: border-color 0.18s, box-shadow 0.18s;
  min-height: 40px;
}
.base-input::placeholder { color: rgba(148,163,184,0.4); }
.base-input:focus {
  outline: none;
  border-color: #38bdf8;
  box-shadow: 0 0 0 3px rgba(56,189,248,0.15);
}
</style>
