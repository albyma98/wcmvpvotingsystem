<script setup>
import { computed, onMounted, onScopeDispose, ref, useId } from 'vue';
import { readArenaCatalog } from '../arenas/arenaCatalog';

const props = defineProps({
  assignedArena: { type: Object, default: null },
  disabled: Boolean,
  label: { type: String, default: 'Arena assegnata' },
});
const model = defineModel({ type: Number });
const fieldId = useId();
const catalog = ref([]);
const loading = ref(false);
const failed = ref(false);
const options = computed(() => {
  if (props.assignedArena && !catalog.value.some(({ id }) => id === props.assignedArena.id)) {
    return [...catalog.value, props.assignedArena];
  }
  return catalog.value;
});
const selected = computed(() => options.value.find(({ id }) => id === model.value));
const abort = new AbortController();

async function load() {
  loading.value = true;
  failed.value = false;
  try {
    catalog.value = await readArenaCatalog({ signal: abort.signal });
  } catch {
    if (!abort.signal.aborted) failed.value = true;
  } finally {
    loading.value = false;
  }
}

onMounted(load);
onScopeDispose(() => abort.abort());
</script>

<template>
  <div class="m-stack arena-select">
    <label class="m-field" :for="fieldId">
      <span>{{ label }}</span>
      <select :id="fieldId" v-model="model" class="m-input" :disabled="disabled || loading || failed">
        <option :value="null">Nessuna arena</option>
        <option v-for="arena in options" :key="arena.id" :value="arena.id">
          {{ arena.name }}{{ arena.city ? ` · ${arena.city}` : '' }}{{ arena.is_active ? '' : ' (inattiva)' }}
        </option>
      </select>
    </label>
    <div v-if="selected" class="arena-preview" aria-live="polite">
      <strong>{{ selected.name }}</strong>
      <small class="m-muted-text">{{ [selected.city, selected.slug].filter(Boolean).join(' · ') }}</small>
      <span v-if="!selected.is_active" class="m-badge m-warn">Inattiva</span>
    </div>
    <p v-if="loading" class="m-muted-text" role="status">Caricamento arene…</p>
    <div v-if="failed" class="m-stack">
      <p class="m-error" role="alert">Impossibile caricare le arene disponibili.</p>
      <div><button class="m-btn" type="button" :disabled="disabled" @click="load">Riprova</button></div>
    </div>
  </div>
</template>

<style scoped>
.arena-select { min-width: 0; }
.arena-preview { display: flex; flex-wrap: wrap; align-items: baseline; gap: .35rem .8rem; overflow-wrap: anywhere; }
</style>
