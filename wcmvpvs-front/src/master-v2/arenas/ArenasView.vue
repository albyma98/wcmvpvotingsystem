<script setup>
import { computed, ref, watch, watchEffect } from 'vue';
import { masterApi } from '../api';
import { masterSectionPath } from '../routes';
import { useMasterWorkspace } from '../workspace';
import DataTable from '../ui/DataTable.vue';
import MasterIcon from '../ui/MasterIcon.vue';
import ArenaForm from './ArenaForm.vue';
import { ARENAS_ENDPOINT, readArenaCatalog } from './arenaCatalog';

const props = defineProps({ route: { type: Object, required: true } });
const { follow, navigate, setContext } = useMasterWorkspace();
const list = ref([]);
const arena = ref(null);
const loading = ref(false);
const error = ref('');
const notice = ref(null);
const refresh = ref(0);
const search = ref('');
const status = ref('');
const path = masterSectionPath('arenas');
const arenaHref = (id) => `${path}/${id}`;
const editHref = (id) => `${arenaHref(id)}/edit`;
const createHref = `${path}/create`;
const number = new Intl.NumberFormat('it-IT');
const filtered = computed(() => list.value.filter((item) => !status.value || Boolean(item.is_active) === (status.value === 'active')));
const visibleNotice = computed(() => {
  if (!notice.value) return null;
  if (props.route.mode === 'list' && notice.value.id == null) return notice.value;
  if (props.route.mode === 'detail' && props.route.id === notice.value.id) return notice.value;
  return null;
});
const searchText = (row) => [row.name, row.slug, row.city, row.team_name, `ID ${row.id}`].filter(Boolean).join(' ');
const columns = [
  { key: 'name', label: 'Arena', sortable: true },
  { key: 'slug', label: 'Slug', sortable: true },
  { key: 'city', label: 'Città', sortable: true },
  { key: 'team_name', label: 'Squadra', sortable: true },
  { key: 'capacity', label: 'Capienza', sortable: true, numeric: true },
  { key: 'is_active', label: 'Stato', sortable: true, value: (row) => row.is_active ? 'Attiva' : 'Inattiva' },
  { key: 'actions', label: 'Azioni' },
];

watchEffect(() => {
  const { mode, id } = props.route;
  const name = arena.value?.id === id ? arena.value.name : '';
  const context = {
    list: { action: { label: 'Nuova arena', href: createHref, icon: 'plus' } },
    create: { title: 'Nuova arena' },
    edit: { title: name ? `Modifica ${name}` : 'Modifica arena' },
    detail: { title: name || 'Scheda arena', action: name ? { label: 'Modifica arena', href: editHref(id), icon: 'edit' } : undefined },
  };
  setContext(context[mode]);
});

watch([() => props.route.mode, () => props.route.id], () => {
  if (!visibleNotice.value) notice.value = null;
});

watch([() => props.route.mode, () => props.route.id, refresh], async ([mode, id], _, onCleanup) => {
  error.value = '';
  arena.value = null;
  if (mode === 'create') {
    loading.value = false;
    return;
  }

  const request = new AbortController();
  onCleanup(() => request.abort());
  const config = { signal: request.signal };
  loading.value = true;
  try {
    if (mode === 'list') {
      const catalog = await readArenaCatalog(config);
      if (!request.signal.aborted) list.value = catalog;
    } else {
      const { data } = await masterApi.get(`${ARENAS_ENDPOINT}/${id}`, config);
      if (request.signal.aborted) return;
      if (data?.id) arena.value = data;
      else error.value = 'Dettagli dell’arena non disponibili.';
    }
  } catch (failure) {
    if (request.signal.aborted) return;
    error.value = failure?.response?.status === 404 ? 'Arena non trovata.' : mode === 'list'
      ? 'Impossibile caricare il catalogo delle arene.'
      : 'Impossibile caricare i dettagli dell’arena.';
  } finally {
    if (!request.signal.aborted) loading.value = false;
  }
}, { immediate: true });

function reload() {
  refresh.value += 1;
}

function afterSave(data) {
  const { id, mode } = props.route;
  const savedId = data?.id || id;
  notice.value = { id: savedId, message: mode === 'create' ? 'Arena creata.' : 'Modifiche all’arena salvate.' };
  navigate(savedId ? arenaHref(savedId) : path, { discard: true });
}
</script>

<template>
  <div class="m-stack arenas-view">
    <p v-if="visibleNotice" class="m-success" role="status">{{ visibleNotice.message }}</p>

    <section v-if="route.mode === 'list'" class="m-panel m-stack">
      <header class="m-panel-head">
        <div><h2>Catalogo arene</h2><p>Arene disponibili per l’associazione alle società e per Arena Tifosi.</p></div>
        <button class="m-btn" type="button" :disabled="loading" @click="reload"><MasterIcon name="refresh" :size="15" /> Aggiorna elenco</button>
      </header>
      <DataTable
        v-model:search="search"
        :rows="filtered"
        :columns="columns"
        :search-text="searchText"
        search-placeholder="Cerca arena, città o squadra…"
        label="Catalogo arene"
        :empty="status ? 'Nessuna arena corrisponde al filtro.' : 'Nessuna arena nel catalogo.'"
        :loading="loading"
        :error="error"
        @retry="reload"
      >
        <template #filters>
          <label>
            <span class="m-sr">Filtra per stato</span>
            <select v-model="status" class="m-input">
              <option value="">Tutti gli stati</option>
              <option value="active">Attive</option>
              <option value="inactive">Inattive</option>
            </select>
          </label>
          <button v-if="status" class="m-btn m-subtle" type="button" @click="status = ''">Azzera filtro</button>
        </template>
        <template #cell-name="{ row }">
          <div class="arena-cell">
            <span class="arena-mark"><MasterIcon name="box" :size="17" /></span>
            <div class="arena-name">
              <a class="m-link" :href="arenaHref(row.id)" @click="follow">{{ row.name || 'Arena senza nome' }}</a>
              <small class="m-muted-text">ID {{ row.id }}</small>
            </div>
          </div>
        </template>
        <template #cell-slug="{ value }"><code v-if="value">{{ value }}</code><span v-else>—</span></template>
        <template #cell-city="{ value }">{{ value || '—' }}</template>
        <template #cell-team_name="{ value }">{{ value || '—' }}</template>
        <template #cell-capacity="{ value }">{{ value == null ? '—' : number.format(value) }}</template>
        <template #cell-is_active="{ row, value }"><span :class="['m-badge', row.is_active ? 'm-good' : 'm-muted']">{{ value }}</span></template>
        <template #cell-actions="{ row }">
          <a :href="editHref(row.id)" class="m-btn m-subtle m-icon-btn" :aria-label="`Modifica ${row.name}`" title="Modifica" @click="follow"><MasterIcon name="edit" :size="16" /></a>
        </template>
      </DataTable>
    </section>

    <template v-else>
      <div class="m-actions">
        <a class="m-btn m-subtle" :href="path" @click="follow"><MasterIcon name="arrow-left" :size="16" /> Catalogo arene</a>
      </div>
      <p v-if="loading" class="m-panel" role="status">Caricamento dei dettagli dell’arena…</p>
      <div v-else-if="error" class="m-panel m-stack">
        <p class="m-error" role="alert">{{ error }}</p>
        <div><button type="button" class="m-btn" @click="reload">Riprova</button></div>
      </div>
      <ArenaForm v-else-if="route.mode === 'create'" :cancel-href="path" @saved="afterSave" />
      <ArenaForm v-else-if="route.mode === 'edit' && arena" :key="route.id" :arena="arena" :cancel-href="arenaHref(route.id)" @saved="afterSave" />
      <template v-else-if="arena">
        <section class="m-panel m-stack">
          <header class="m-panel-head">
            <div><h2>Informazioni generali</h2><p>Riferimenti dell’arena nel catalogo.</p></div>
            <span :class="['m-badge', arena.is_active ? 'm-good' : 'm-muted']">{{ arena.is_active ? 'Attiva' : 'Inattiva' }}</span>
          </header>
          <dl class="arena-details">
            <div><dt>ID arena</dt><dd>{{ arena.id }}</dd></div>
            <div><dt>Slug</dt><dd><code>{{ arena.slug || '—' }}</code></dd></div>
            <div><dt>Città</dt><dd>{{ arena.city || '—' }}</dd></div>
            <div><dt>Squadra</dt><dd>{{ arena.team_name || '—' }}</dd></div>
            <div><dt>Capienza</dt><dd>{{ arena.capacity == null ? '—' : number.format(arena.capacity) }}</dd></div>
          </dl>
        </section>
        <section class="m-panel m-stack">
          <header class="m-panel-head"><div><h2>Configurazione 3D</h2><p>Configurazione salvata della scena di Arena Tifosi.</p></div></header>
          <pre v-if="arena.scene_config" class="scene-preview" tabindex="0" role="region" aria-label="Configurazione JSON della scena">{{ JSON.stringify(arena.scene_config, null, 2) }}</pre>
          <p v-else class="m-muted-text">Configurazione non disponibile.</p>
        </section>
      </template>
    </template>
  </div>
</template>

<style scoped>
.arenas-view :deep(.m-table) { min-width: 46rem; }
.arena-cell { display: flex; align-items: center; gap: .65rem; min-width: 10rem; }
.arena-mark { display: grid; width: 2rem; height: 2rem; flex: none; place-items: center; border: 1px solid var(--m-line); border-radius: .45rem; background: var(--m-soft); color: var(--m-accent); }
.arena-name { display: grid; gap: .2rem; min-width: 0; overflow-wrap: anywhere; }
.arena-name a { font-weight: 650; }
.arena-details { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(100%, 9rem), 1fr)); gap: 1rem; margin: 0; }
.arena-details dt { color: var(--m-muted); font-size: .85rem; }
.arena-details dd { margin: .25rem 0 0; overflow-wrap: anywhere; }
.scene-preview { margin: 0; padding: 1rem; max-height: 22rem; overflow: auto; overflow-wrap: anywhere; white-space: pre-wrap; border-radius: .5rem; border: 1px solid var(--m-line); background: var(--m-bg); font: .875rem/1.55 ui-monospace, Consolas, monospace; }
</style>
