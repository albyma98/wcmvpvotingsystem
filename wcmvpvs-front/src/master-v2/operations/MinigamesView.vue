<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { masterApi } from '../api';
import { useMasterWorkspace } from '../workspace';
import DataTable from '../ui/DataTable.vue';
import MasterIcon from '../ui/MasterIcon.vue';

const ENDPOINT = '/admin/master/minigames';
const options = [
  { key: 'football', label: 'Calcio' },
  { key: 'volleyball', label: 'Pallavolo' },
  { key: 'basketball', label: 'Pallacanestro' },
  { key: 'all', label: 'Tutti' },
];
const columns = [{ key: 'name', label: 'Minigioco', sortable: true }, ...options, { key: 'actions', label: 'Azioni' }];
const games = ref([]);
const loading = ref(false);
const error = ref('');
const search = ref('');
const availability = ref('');
const controller = new AbortController();

function isDirty(game) {
  return game.selected.length !== game.sports.length || game.selected.some(sport => !game.sports.includes(sport));
}

const hasChanges = computed(() => games.value.some(game => isDirty(game) || game.saving));
const filtered = computed(() => games.value.filter(game => !availability.value || (game.sports.length ? 'enabled' : 'disabled') === availability.value));
const { useDirty } = useMasterWorkspace();
useDirty(hasChanges);

async function load() {
  if (loading.value || hasChanges.value) return;
  loading.value = true;
  error.value = '';
  try {
    const { data } = await masterApi.get(ENDPOINT, { signal: controller.signal });
    games.value = (Array.isArray(data) ? data.filter(Boolean) : []).map(game => {
      const sports = Array.isArray(game.sports) ? game.sports : [];
      return { ...game, sports, selected: [...sports], saving: false, error: '', saved: false };
    });
  } catch {
    if (!controller.signal.aborted) error.value = 'Impossibile caricare i minigiochi. Riprova.';
  } finally {
    loading.value = false;
  }
}

function toggle(game, sport, checked) {
  game.error = '';
  game.saved = false;
  if (sport === 'Tutti') {
    game.selected = checked ? ['Tutti'] : [];
    return;
  }
  game.selected = game.selected.filter(value => value !== 'Tutti' && value !== sport);
  if (checked) game.selected.push(sport);
}

function reset(game) {
  game.selected = [...game.sports];
  game.error = '';
  game.saved = false;
}

async function save(game) {
  if (game.saving || !isDirty(game)) return;
  game.saving = true;
  game.error = '';
  game.saved = false;
  try {
    const { data } = await masterApi.put(`${ENDPOINT}/${encodeURIComponent(game.id)}`, { sports: [...game.selected] });
    game.sports = [...data.sports];
    game.selected = [...data.sports];
    game.saved = true;
  } catch (cause) {
    game.error = cause?.response?.data?.message || 'Salvataggio non riuscito. Riprova per applicare le modifiche.';
  } finally {
    game.saving = false;
  }
}

onMounted(load);
onBeforeUnmount(() => controller.abort());
</script>

<template>
  <section class="m-panel minigames-directory" aria-labelledby="minigames-heading">
    <header class="m-panel-head">
      <div><h2 id="minigames-heading">Disponibilità per sport</h2><p>Imposta quali minigiochi sono accessibili alle società di ciascuno sport.</p></div>
      <button class="m-btn" type="button" :disabled="loading || hasChanges" @click="load"><MasterIcon name="refresh" :size="16" />Aggiorna</button>
    </header>
    <p class="minigame-guide">«Tutti» abilita il minigioco per ogni sport. Con tutte le caselle vuote il minigioco è disabilitato. Le eventuali attivazioni sui singoli eventi si gestiscono nelle rispettive configurazioni.</p>
    <p v-if="hasChanges" class="minigame-changes" role="status">Ci sono modifiche da salvare. Salva o annulla dalla riga del minigioco.</p>
    <DataTable v-model:search="search" :rows="filtered" :columns="columns" label="Disponibilità dei minigiochi" :search-text="game => `${game.name || ''} ${game.description || ''}`" search-placeholder="Nome o descrizione" :loading="loading" :error="error" empty="Nessun minigioco per i filtri selezionati." @retry="load">
      <template #filters>
        <label class="m-field">Disponibilità salvata<select v-model="availability" class="m-input"><option value="">Tutti i minigiochi</option><option value="enabled">Attivi</option><option value="disabled">Disabilitati</option></select></label>
      </template>
      <template #cell-name="{ row }">
        <div class="minigame-name"><strong>{{ row.name || row.id }}</strong><small v-if="row.description" class="m-muted-text">{{ row.description }}</small><span :class="['m-badge', row.sports.length ? 'm-good' : 'm-muted']">{{ row.sports.length ? 'Disponibile' : 'Disabilitato' }}</span></div>
      </template>
      <template v-for="sport in options" :key="sport.key" #[`cell-${sport.key}`]="{ row }">
        <label class="minigame-toggle">
          <input type="checkbox" :checked="row.selected.includes(sport.label)" :disabled="row.saving || loading" :aria-label="`${row.name || row.id} · ${sport.label}`" @change="toggle(row, sport.label, $event.target.checked)" />
        </label>
      </template>
      <template #cell-actions="{ row }">
        <div class="minigame-action">
          <div class="m-actions">
            <button class="m-btn m-primary" type="button" :disabled="row.saving || !isDirty(row)" :aria-label="`Salva sport per ${row.name || row.id}`" @click="save(row)">{{ row.saving ? 'Salvataggio…' : 'Salva' }}</button>
            <button v-if="isDirty(row)" class="m-btn m-subtle" type="button" :disabled="row.saving" :aria-label="`Annulla modifiche per ${row.name || row.id}`" @click="reset(row)">Annulla</button>
          </div>
          <small v-if="row.error" class="m-error" role="alert">{{ row.error }}</small>
          <small v-else-if="row.saved" class="m-success" role="status">Salvato</small>
        </div>
      </template>
    </DataTable>
  </section>
</template>

<style scoped>
.minigames-directory { min-width: 0; }
.minigame-guide, .minigame-changes { padding: 11px 14px; border: 1px solid var(--m-line); border-radius: 8px; font-size: .87rem; line-height: 1.6; margin: 0 0 14px; }
.minigame-guide { background: var(--m-bg); color: var(--m-muted); }
.minigame-changes { background: var(--m-soft); color: var(--m-ink); }
.minigame-name { display: grid; gap: 5px; min-width: 11rem; max-width: 26rem; }
.minigame-name .m-badge { width: fit-content; }
.minigame-toggle { display: grid; place-items: center; width: 36px; min-height: 36px; cursor: pointer; }
.minigame-toggle input { width: 17px; height: 17px; cursor: pointer; accent-color: var(--m-accent); }
.minigame-toggle:has(:disabled), .minigame-toggle input:disabled { cursor: wait; }
.minigame-action { display: grid; gap: 5px; min-width: 7rem; }
.minigame-action .m-actions { justify-content: flex-start; }
</style>
