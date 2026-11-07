<script setup>
import { computed, onMounted, ref, useId } from 'vue';
import { apiClient } from '../../api';
import BaseButton from '../admin/ui/BaseButton.vue';
import BaseSelect from '../admin/ui/BaseSelect.vue';

const props = defineProps({
  assignedArena: { type: Object, default: null },
  disabled: Boolean,
  label: { type: String, default: 'Arena assegnata' },
});
const arenaId = defineModel({ type: Number });
const fieldId = useId();
const arenas = ref([]);
const loading = ref(true);
const loadError = ref('');
const options = computed(() => {
  const current = props.assignedArena;
  return current && !arenas.value.some(arena => arena.id === current.id)
    ? [...arenas.value, current]
    : arenas.value;
});
const selected = computed(() => options.value.find(arena => arena.id === arenaId.value));

async function load() {
  loading.value = true;
  loadError.value = '';
  try {
    const { data } = await apiClient.get('/admin/master/arenas');
    arenas.value = data.arenas ?? [];
  } catch {
    loadError.value = 'Impossibile caricare le arene disponibili.';
  } finally {
    loading.value = false;
  }
}

onMounted(load);
</script>

<template>
  <div class="arena-field">
    <BaseSelect>
      <label :for="fieldId">{{ label }}</label>
      <select :id="fieldId" v-model="arenaId" :disabled="disabled || loading || Boolean(loadError)">
        <option :value="null">Nessuna arena</option>
        <option v-for="arena in options" :key="arena.id" :value="arena.id">
          {{ arena.name }} · {{ arena.city }}{{ arena.is_active ? '' : ' (inattiva)' }}
        </option>
      </select>
    </BaseSelect>
    <div aria-live="polite">
      <div v-if="selected" class="arena-preview">
        <strong>{{ selected.name }}</strong>
        <span>{{ selected.city }} · <code>{{ selected.slug }}</code></span>
        <small v-if="!selected.is_active">Arena inattiva</small>
      </div>
    </div>
    <p v-if="loading" class="arena-hint" role="status">Caricamento arene…</p>
    <div v-if="loadError" class="arena-load-error">
      <p role="alert">{{ loadError }}</p>
      <BaseButton variant="outline" size="sm" :disabled="disabled" @click="load">Riprova</BaseButton>
    </div>
  </div>
</template>

<style scoped>
.arena-field { display: grid; gap: 0.55rem; min-width: 0; }
.arena-field label { font-weight: 600; }
.arena-preview { display: grid; gap: 0.2rem; border: 1px solid #e2e8f0; border-radius: 0.65rem; padding: 0.7rem 0.8rem; background: #f8fafc; overflow-wrap: anywhere; }
.arena-preview span, .arena-hint { color: #475569; font-size: 0.85rem; }
.arena-preview small { color: #9a3412; }
.arena-load-error { display: flex; align-items: center; flex-wrap: wrap; gap: 0.5rem; color: #b91c1c; font-size: 0.85rem; }
.arena-field p { margin: 0; }
</style>
