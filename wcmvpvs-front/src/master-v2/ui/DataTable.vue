<script setup>
import { computed, ref, useId, watch } from 'vue';
import MasterIcon from './MasterIcon.vue';

const props = defineProps({
  rows: { type: Array, default: () => [] },
  columns: { type: Array, required: true },
  rowKey: { type: [String, Function], default: 'id' },
  search: { type: String, default: undefined },
  searchText: Function,
  searchPlaceholder: { type: String, default: 'Cerca nell’elenco…' },
  label: { type: String, default: 'Elenco dei risultati' },
  empty: { type: String, default: 'Non ci sono ancora elementi.' },
  loading: Boolean,
  error: String,
});
const emit = defineEmits(['update:search', 'retry']);
const localSearch = ref('');
const searchInput = computed({
  get: () => props.search ?? localSearch.value,
  set: (value) => { localSearch.value = value; emit('update:search', value); },
});
const sortKey = ref('');
const direction = ref(1);
const page = ref(1);
const pageSize = ref(10);
const pageSizeId = useId();
const collator = new Intl.Collator('it', { numeric: true, sensitivity: 'base' });
const normalize = (text) => String(text ?? '').normalize('NFD').replace(/\p{M}/gu, '').toLocaleLowerCase('it');
const valueFor = (column, row) => column.value ? column.value(row) : row[column.key];
const keyFor = (row) => typeof props.rowKey === 'function' ? props.rowKey(row) : row[props.rowKey];

const filtered = computed(() => {
  const query = normalize(searchInput.value.trim());
  const rows = query && props.searchText ? props.rows.filter((row) => normalize(props.searchText(row)).includes(query)) : props.rows;
  const column = props.columns.find(({ key }) => key === sortKey.value);
  if (!column) return rows;
  return [...rows].sort((left, right) => {
    const a = valueFor(column, left);
    const b = valueFor(column, right);
    if (a == null || b == null) return Number(a == null) - Number(b == null);
    return direction.value * (column.numeric ? Number(a) - Number(b) : collator.compare(a, b));
  });
});
const pageCount = computed(() => Math.max(1, Math.ceil(filtered.value.length / pageSize.value)));
const first = computed(() => (page.value - 1) * pageSize.value);
const visibleRows = computed(() => filtered.value.slice(first.value, first.value + pageSize.value));
watch([searchInput, pageSize, sortKey, direction], () => { page.value = 1; });
watch(pageCount, (count) => { page.value = Math.min(page.value, count); });

function sort(column) {
  direction.value = sortKey.value === column.key ? -direction.value : 1;
  sortKey.value = column.key;
}

function sortState(column) {
  if (sortKey.value === column.key) return direction.value === 1 ? 'ascending' : 'descending';
  return undefined;
}
</script>

<template>
  <div class="m-data" :aria-busy="loading">
    <div v-if="searchText || $slots.filters" class="m-table-tools">
      <label v-if="searchText" class="m-search">
        <MasterIcon name="search" :size="17" />
        <span class="m-sr">Cerca in {{ label.toLowerCase() }}</span>
        <input v-model="searchInput" class="m-input" type="search" :placeholder="searchPlaceholder" />
      </label>
      <slot name="filters" />
    </div>
    <div v-if="error" class="m-table-state" role="alert">
      <MasterIcon name="alert" />
      <p>{{ error }}</p>
      <button type="button" class="m-btn" :disabled="loading" @click="emit('retry')">Riprova</button>
    </div>
    <div v-else-if="loading && !rows.length" class="m-table-state" role="status">
      <span class="m-spinner" aria-hidden="true"></span><p>Caricamento in corso…</p>
    </div>
    <div v-else-if="!filtered.length" class="m-table-state" role="status">
      <MasterIcon name="inbox" :size="22" />
      <p>{{ searchInput.trim() ? 'Nessun risultato per questa ricerca.' : empty }}</p>
      <button v-if="searchInput.trim()" type="button" class="m-btn" @click="searchInput = ''">Azzera ricerca</button>
    </div>
    <template v-else>
      <div class="m-table-scroll" tabindex="0" role="region" :aria-label="label">
        <table class="m-table">
          <caption class="m-sr">{{ label }}</caption>
          <thead>
            <tr>
              <th v-for="column in columns" :key="column.key" scope="col" :aria-sort="sortState(column)" :class="{ 'm-numeric': column.numeric }">
                <button v-if="column.sortable" type="button" class="m-sort" @click="sort(column)">
                  {{ column.label }}<MasterIcon :name="sortState(column) === 'ascending' ? 'arrow-up' : sortState(column) ? 'arrow-down' : 'chevrons-up-down'" :size="13" />
                </button>
                <template v-else>{{ column.label }}</template>
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in visibleRows" :key="keyFor(row)">
              <td v-for="column in columns" :key="column.key" :class="{ 'm-numeric': column.numeric }">
                <slot :name="`cell-${column.key}`" :row="row" :value="valueFor(column, row)">{{ valueFor(column, row) ?? '—' }}</slot>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="m-pagination">
        <span aria-live="polite">{{ first + 1 }}–{{ first + visibleRows.length }} di {{ filtered.length }}</span>
        <div class="m-actions">
          <label :for="pageSizeId">Righe per pagina</label>
          <select :id="pageSizeId" v-model.number="pageSize" class="m-input">
            <option v-for="size in [10, 25, 50]" :key="size" :value="size">{{ size }}</option>
          </select>
          <button type="button" class="m-btn m-icon-btn" aria-label="Pagina precedente" :disabled="page <= 1" @click="page -= 1"><MasterIcon name="chevron-left" :size="15" /></button>
          <button type="button" class="m-btn m-icon-btn" aria-label="Pagina successiva" :disabled="page >= pageCount" @click="page += 1"><MasterIcon name="chevron-right" :size="15" /></button>
        </div>
      </div>
    </template>
  </div>
</template>
