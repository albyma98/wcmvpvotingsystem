<template>
  <section class="sponsor-placements" :class="placementGroup">
    <header>
      <h2>{{ config.title }}</h2>
      <p>{{ config.description }}</p>
    </header>
    <p v-if="loading" role="status">Caricamento delle assegnazioni…</p>
    <button v-else-if="!saved && error" type="button" :class="run ? 'v2-button' : 'btn secondary'" @click="load">
      Riprova
    </button>
    <form v-else-if="saved" class="placement-form" @submit.prevent="submit">
      <div class="placement-groups">
        <fieldset v-for="group in config.groups" :key="group.label" :disabled="saving">
          <legend>{{ group.label }}</legend>
          <label v-for="slot in group.slots" :key="slot.id" class="v2-field">
            {{ slot.label }}
            <select v-model="draft[slot.id]">
              <option :value="0">{{ config.emptyLabel }}</option>
              <option v-for="sponsor in sponsors" :key="sponsor.id" :value="sponsor.id">
                {{ sponsor.name || `Sponsor ${sponsor.position}` }}{{ sponsor.isActive ? '' : ' (nascosto)' }}
              </option>
            </select>
          </label>
        </fieldset>
      </div>
      <div>
        <button type="submit" :class="run ? 'v2-button v2-primary' : 'btn primary'" :disabled="saving || !dirty">
          {{ saving ? 'Salvataggio…' : config.submitLabel }}
        </button>
      </div>
    </form>
    <p v-if="error" class="placement-error" role="alert">{{ error }}</p>
    <p v-if="success && !dirty" class="placement-success" role="status">Assegnazioni salvate.</p>
  </section>
</template>

<script setup>
import { computed, ref, watch } from 'vue';
import { apiClient } from '../../api';
import { gameWallGames } from '../../games/gameWallSponsors';
import { postVoteSponsorSlots } from '../../sponsorPlacements';

const props = defineProps({
  authHeaders: { type: Object, required: true },
  sponsors: { type: Array, required: true },
  placementGroup: { type: String, required: true, validator: (value) => ['game-walls', 'post-vote'].includes(value) },
  run: { type: Function, default: null },
});
const endpoint = `/admin/sponsors/${props.placementGroup}`;
const config = props.placementGroup === 'post-vote' ? {
  title: 'Banner nella conferma MVP',
  description: 'Scegli gli sponsor da mostrare dopo il voto. Se scegli un solo sponsor, il banner sarà centrato.',
  emptyLabel: 'Nessuno sponsor',
  submitLabel: 'Salva banner MVP',
  groups: [{ label: 'Partner del Match Day', slots: postVoteSponsorSlots }],
} : {
  title: 'Banner nei minigiochi',
  description: 'Scegli gli sponsor da mostrare nei pannelli. Seleziona «Pannello neutro» per lasciare libera una parete.',
  emptyLabel: 'Pannello neutro',
  submitLabel: 'Salva banner',
  groups: gameWallGames.map((game) => ({
    label: game.label,
    slots: [
      { id: `${game.id}_left`, label: 'Parete sinistra' },
      { id: `${game.id}_right`, label: 'Parete destra' },
    ],
  })),
};
const slots = config.groups.flatMap((group) => group.slots.map((slot) => slot.id));
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
  return props.run ? props.run(`sponsor-${props.placementGroup}`, save) : save();
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
.sponsor-placements, .placement-form { display: grid; gap: 16px; }
.sponsor-placements header { display: grid; gap: 6px; }
.sponsor-placements header p { color: var(--v2-muted, #526078); font-size: .875rem; }
.placement-groups { display: grid; gap: 12px; grid-template-columns: repeat(auto-fit, minmax(min(100%, 210px), 1fr)); }
.placement-groups fieldset { display: grid; gap: 12px; min-width: 0; padding: 12px; border: 1px solid var(--v2-line, #dce2e9); border-radius: 8px; }
.post-vote fieldset { grid-template-columns: repeat(auto-fit, minmax(min(100%, 210px), 1fr)); }
.placement-groups legend { padding: 0 6px; font-weight: 600; }
.placement-groups label { display: grid; gap: 4px; }
.placement-groups select { width: 100%; min-height: 42px; }
.placement-error { color: #a32135; }
.placement-success { color: #116443; }
</style>
