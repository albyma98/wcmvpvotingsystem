<template>
  <section class="game-wall-sponsors">
    <header>
      <h2>Banner nei minigiochi</h2>
      <p>I pannelli mostrano i loghi degli sponsor visibili. Seleziona «Pannello neutro» per lasciare libera una parete.</p>
    </header>
    <p v-if="loading" role="status">Caricamento delle assegnazioni…</p>
    <button v-else-if="!saved && error" type="button" :class="run ? 'v2-button' : 'btn secondary'" @click="load">
      Riprova
    </button>
    <form v-else-if="saved" class="wall-form" @submit.prevent="submit">
      <div class="wall-games">
        <fieldset v-for="game in gameWallGames" :key="game.id" :disabled="saving">
          <legend>{{ game.label }}</legend>
          <label v-for="side in sides" :key="side.id" class="v2-field">
            {{ side.label }}
            <select v-model="draft[`${game.id}_${side.id}`]">
              <option :value="0">Pannello neutro</option>
              <option v-for="sponsor in sponsors" :key="sponsor.id" :value="sponsor.id" :disabled="!sponsor.isActive">
                {{ sponsor.name || `Sponsor ${sponsor.position}` }}{{ sponsor.isActive ? '' : ' (nascosto)' }}
              </option>
            </select>
          </label>
        </fieldset>
      </div>
      <div>
        <button type="submit" :class="run ? 'v2-button v2-primary' : 'btn primary'" :disabled="saving || !dirty">
          {{ saving ? 'Salvataggio…' : 'Salva banner' }}
        </button>
      </div>
    </form>
    <p v-if="error" class="wall-error" role="alert">{{ error }}</p>
    <p v-if="success && !dirty" class="wall-success" role="status">Assegnazioni salvate.</p>
  </section>
</template>

<script setup>
import { computed, ref, watch } from 'vue';
import { apiClient } from '../../api';
import { gameWallGames } from '../../games/gameWallSponsors';

const props = defineProps({
  authHeaders: { type: Object, required: true },
  sponsors: { type: Array, required: true },
  run: { type: Function, default: null },
});
const endpoint = '/admin/sponsors/game-walls';
const sides = [
  { id: 'left', label: 'Parete sinistra' },
  { id: 'right', label: 'Parete destra' },
];
const slots = gameWallGames.flatMap((game) => sides.map((side) => `${game.id}_${side.id}`));
const saved = ref(null);
const draft = ref({});
const loading = ref(false);
const saving = ref(false);
const error = ref('');
const success = ref(false);
const dirty = computed(() => Boolean(saved.value) && slots.some((slot) => draft.value[slot] !== saved.value[slot]));
let requestScope;

watch(() => props.authHeaders.headers?.['X-Organization-Slug'], (slug, _, onCleanup) => {
  const scope = new AbortController();
  requestScope = scope;
  onCleanup(() => scope.abort());
  saved.value = null;
  draft.value = {};
  saving.value = false;
  loading.value = false;
  error.value = '';
  success.value = false;
  if (slug) void load();
}, { immediate: true });

watch(() => props.sponsors, (catalog, previous) => {
  if (!saved.value) return;
  const removed = new Set(previous.filter((item) => !catalog.some((sponsor) => sponsor.id === item.id)).map((item) => item.id));
  for (const assignments of [saved.value, draft.value]) {
    for (const slot of slots) {
      if (removed.has(assignments[slot])) assignments[slot] = 0;
    }
  }
});

async function load() {
  if (loading.value) return;
  const { signal } = requestScope;
  loading.value = true;
  error.value = '';
  try {
    const { data } = await apiClient.get(endpoint, { ...props.authHeaders, signal });
    if (signal.aborted) return;
    if (!data || typeof data !== 'object' || Array.isArray(data)) throw new Error('Invalid assignments');
    saved.value = Object.fromEntries(slots.map((slot) => [slot, data[slot] || 0]));
    draft.value = { ...saved.value };
  } catch {
    if (!signal.aborted) error.value = 'Impossibile caricare le assegnazioni dei banner.';
  } finally {
    if (!signal.aborted) loading.value = false;
  }
}

function submit() {
  return props.run ? props.run('sponsor-game-walls', save) : save();
}

async function save() {
  if (saving.value || !dirty.value) return;
  const { signal } = requestScope;
  const next = { ...draft.value };
  saving.value = true;
  error.value = '';
  success.value = false;
  try {
    await apiClient.put(endpoint, next, { ...props.authHeaders, signal });
    if (!signal.aborted) {
      saved.value = { ...draft.value };
      success.value = true;
    }
  } catch (err) {
    if (!signal.aborted) error.value = err?.response?.data?.message || 'Impossibile salvare le assegnazioni dei banner.';
  } finally {
    if (!signal.aborted) saving.value = false;
  }
}
</script>

<style scoped>
.game-wall-sponsors, .wall-form { display: grid; gap: 16px; }
.game-wall-sponsors header { display: grid; gap: 6px; }
.game-wall-sponsors header p { color: var(--v2-muted, #526078); font-size: .875rem; }
.wall-games { display: grid; gap: 12px; grid-template-columns: repeat(auto-fit, minmax(min(100%, 210px), 1fr)); }
.wall-games fieldset { display: grid; gap: 12px; min-width: 0; padding: 12px; border: 1px solid var(--v2-line, #dce2e9); border-radius: 8px; }
.wall-games legend { padding: 0 6px; font-weight: 600; }
.wall-games label { display: grid; gap: 4px; }
.wall-games select { width: 100%; min-height: 42px; }
.wall-error { color: #a32135; }
.wall-success { color: #116443; }
</style>
