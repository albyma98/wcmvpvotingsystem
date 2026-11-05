<template><section class="v2-panel"><header v-if="title || $slots.actions" class="v2-panel-heading"><div><h2 v-if="title">{{ title }}</h2><p v-if="description" class="v2-muted">{{ description }}</p></div><div v-if="$slots.actions" class="v2-toolbar"><slot name="actions" /></div></header><slot /></section></template>
<script setup>defineProps({ title: String, description: String });</script>
