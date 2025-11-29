<template>
  <Card class="app-table">
    <template v-if="title" #title>{{ title }}</template>
    <template #content>
      <DataTable
        :value="value"
        :paginator="paginator"
        :rows="rows"
        :rows-per-page-options="[5, 10, 20, 50]"
        :loading="loading"
        scrollable
        scroll-height="flex"
        :show-gridlines="true"
      >
        <slot name="columns" />
      </DataTable>
    </template>
  </Card>
</template>

<script setup>
import { defineProps } from 'vue';

const props = defineProps({
  title: { type: String, default: '' },
  value: { type: Array, default: () => [] },
  paginator: { type: Boolean, default: true },
  rows: { type: Number, default: 10 },
  loading: { type: Boolean, default: false },
});
</script>

<style scoped>
.app-table :deep(.p-datatable) {
  width: 100%;
}
</style>
