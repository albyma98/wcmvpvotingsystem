<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue';
import { masterApi } from '../api';
import { useMasterWorkspace } from '../workspace';
import DataTable from '../ui/DataTable.vue';
import MasterDialog from '../ui/MasterDialog.vue';
import MasterIcon from '../ui/MasterIcon.vue';
import { dateLabel, number } from './analyticsFormat';

const ENDPOINT = '/admin/master/qr-redirects';
const redirects = ref([]);
const search = ref('');
const filter = ref('');
const loading = ref(false);
const loadError = ref('');
const actionError = ref('');
const notice = ref('');
const editor = ref(null);
const toDelete = ref(null);
const operation = ref('');
const pendingRename = ref(null);
const formError = ref('');
const errors = reactive({ source_path: '', target_path: '' });
const form = reactive({ source_path: '', target_path: '' });
const controller = new AbortController();

const editing = computed(() => Boolean(editor.value?.id));
const busy = computed(() => Boolean(operation.value));
const dirty = computed(() => Boolean(editor.value) && (
  form.source_path !== editor.value.source_path || form.target_path !== editor.value.target_path || Boolean(pendingRename.value)
));
const { useDirty } = useMasterWorkspace();
useDirty(computed(() => dirty.value || busy.value));

const filtered = computed(() => redirects.value.filter(row => !filter.value || (row.active ? 'active' : 'inactive') === filter.value));
const columns = [
  { key: 'source_path', label: 'Percorso QR', sortable: true },
  { key: 'target_path', label: 'Destinazione', sortable: true },
  { key: 'active', label: 'Stato', sortable: true, value: row => row.active ? 'Attivo' : 'Disattivato' },
  { key: 'created_at', label: 'Collegato il', sortable: true, value: row => Date.parse(row.created_at) || null },
  { key: 'hits', label: 'Arrivi da QR', numeric: true, sortable: true },
  { key: 'actions', label: 'Azioni' },
];
const searchText = row => `${row.source_path || ''} ${row.target_path || ''}`;

async function load() {
  if (loading.value || controller.signal.aborted) return;
  loading.value = true;
  loadError.value = '';
  try {
    const { data } = await masterApi.get(ENDPOINT, { signal: controller.signal });
    redirects.value = Array.isArray(data) ? data.filter(Boolean) : [];
  } catch {
    if (!controller.signal.aborted) loadError.value = 'Impossibile caricare i collegamenti QR. Riprova.';
  } finally {
    loading.value = false;
  }
}

function openEditor(row) {
  editor.value = { id: row?.id, source_path: row?.source_path || '', target_path: row?.target_path || '' };
  form.source_path = editor.value.source_path;
  form.target_path = editor.value.target_path;
  formError.value = '';
  errors.source_path = '';
  errors.target_path = '';
  notice.value = '';
}

function closeEditor() {
  if (busy.value) return;
  if (dirty.value && !window.confirm(pendingRename.value
    ? 'Il nuovo percorso è già stato salvato e quello precedente è ancora presente. Vuoi chiudere comunque?'
    : 'Scartare le modifiche non salvate?')) return;
  editor.value = null;
  pendingRename.value = null;
}

function validate() {
  const input = form.source_path.trim();
  const source = input.startsWith('/') ? input : `/${input}`;
  const target = form.target_path.trim();
  errors.source_path = !input ? 'Inserisci il percorso di origine.' : '';
  if (source.startsWith('/admin')) errors.source_path = 'I percorsi che iniziano con /admin sono riservati.';
  errors.target_path = !target ? 'Inserisci la destinazione.' : '';
  if (target && !/^(\/|https?:\/\/)/i.test(target)) errors.target_path = 'Usa un percorso con / oppure un URL HTTP o HTTPS.';
  return { valid: !errors.source_path && !errors.target_path, source, target };
}

function applyRow(row) {
  if (row?.id) redirects.value = [row, ...redirects.value.filter(item => item.id !== row.id)];
}

async function remove(id) {
  try {
    await masterApi.delete(`${ENDPOINT}/${id}`);
  } catch (error) {
    if (error?.response?.status !== 404) throw error;
  }
  redirects.value = redirects.value.filter(row => row.id !== id);
}

async function save() {
  if (busy.value || !editor.value) return;
  formError.value = '';
  const { valid, source, target } = validate();
  if (!valid) return;
  if (!pendingRename.value && source !== editor.value.source_path && redirects.value.some(row => row.source_path === source)) {
    if (!window.confirm(`Il percorso ${source} è già collegato. Sostituire la sua destinazione?`)) return;
  }

  operation.value = 'save';
  actionError.value = '';
  try {
    if (!pendingRename.value) {
      const { data } = await masterApi.post(ENDPOINT, { source_path: source, target_path: target });
      if (!data?.id || data.source_path !== source) throw new Error('Risposta del redirect non valida');
      applyRow(data);
      // Compare the normalized source. The API prepends the slash to bare paths.
      if (editing.value && source !== editor.value.source_path && data.id !== editor.value.id) {
        pendingRename.value = { id: editor.value.id, source: editor.value.source_path };
      }
    }
    if (pendingRename.value) await remove(pendingRename.value.id);
    notice.value = editing.value ? 'Collegamento aggiornato.' : 'Collegamento salvato.';
    editor.value = null;
    pendingRename.value = null;
    await load();
  } catch (error) {
    if (pendingRename.value) {
      formError.value = `Il nuovo percorso è stato salvato. Non è stato possibile rimuovere ${pendingRename.value.source}. Riprova la rimozione.`;
    } else if (error?.response?.status === 400) {
      formError.value = 'Il server ha rifiutato i percorsi. Controlla origine e destinazione.';
    } else {
      formError.value = 'Impossibile salvare il collegamento. Riprova.';
    }
  } finally {
    operation.value = '';
  }
}

async function toggle(row) {
  if (busy.value || loading.value) return;
  operation.value = `toggle-${row.id}`;
  actionError.value = '';
  notice.value = '';
  try {
    await masterApi.put(`${ENDPOINT}/${row.id}/active`, { active: !row.active });
    applyRow({ ...row, active: !row.active });
    notice.value = `${row.source_path} ${row.active ? 'disattivato' : 'attivato'}.`;
    await load();
  } catch {
    actionError.value = `Impossibile cambiare lo stato di ${row.source_path}. Riprova.`;
  } finally {
    operation.value = '';
  }
}

function openDelete(row) {
  formError.value = '';
  toDelete.value = row;
}

async function confirmDelete() {
  if (busy.value || !toDelete.value) return;
  operation.value = 'delete';
  formError.value = '';
  actionError.value = '';
  notice.value = '';
  try {
    const { id, source_path } = toDelete.value;
    await remove(id);
    toDelete.value = null;
    notice.value = `Collegamento ${source_path} eliminato.`;
    await load();
  } catch {
    formError.value = 'Impossibile eliminare il collegamento. Riprova.';
  } finally {
    operation.value = '';
  }
}

onMounted(load);
onBeforeUnmount(() => controller.abort());
</script>

<template>
  <section class="m-panel qr-directory" aria-labelledby="qr-directory-heading">
    <header class="m-panel-head">
      <div><h2 id="qr-directory-heading">Collegamenti QR</h2><p>Gestisci la destinazione dei percorsi e consulta gli accessi registrati.</p></div>
      <div class="m-actions">
        <button class="m-btn" type="button" :disabled="loading || busy" @click="load"><MasterIcon name="refresh" :size="16" />Aggiorna</button>
        <button class="m-btn m-primary" type="button" :disabled="loading || busy" @click="openEditor()"><MasterIcon name="plus" :size="16" />Nuovo collegamento</button>
      </div>
    </header>
    <p v-if="actionError" class="m-error" role="alert">{{ actionError }}</p>
    <p v-if="notice" class="m-success" role="status">{{ notice }}</p>
    <DataTable v-model:search="search" :rows="filtered" :columns="columns" label="Collegamenti QR" :search-text="searchText" search-placeholder="Percorso QR o destinazione" :loading="loading" :error="loadError" empty="Nessun collegamento per i filtri selezionati." @retry="load">
      <template #filters>
        <label class="m-field">Stato<select v-model="filter" class="m-input"><option value="">Tutti gli stati</option><option value="active">Attivi</option><option value="inactive">Disattivati</option></select></label>
      </template>
      <template #cell-source_path="{ value }"><code class="qr-path">{{ value || '—' }}</code></template>
      <template #cell-target_path="{ value }"><code class="qr-path">{{ value || '—' }}</code></template>
      <template #cell-active="{ row, value }"><span :class="['m-badge', row.active ? 'm-good' : 'm-muted']">{{ value }}</span></template>
      <template #cell-created_at="{ row }">{{ dateLabel(row.created_at) }}</template>
      <template #cell-hits="{ value }">{{ number(value) }}</template>
      <template #cell-actions="{ row }">
        <div class="m-actions qr-row-actions">
          <button class="m-btn m-subtle" type="button" :disabled="busy || loading" :aria-label="`Modifica ${row.source_path}`" @click="openEditor(row)">Modifica</button>
          <button class="m-btn m-subtle" type="button" :disabled="busy || loading" :aria-label="`${row.active ? 'Disattiva' : 'Attiva'} ${row.source_path}`" @click="toggle(row)">{{ operation === `toggle-${row.id}` ? 'Aggiornamento…' : row.active ? 'Disattiva' : 'Attiva' }}</button>
          <button class="m-btn m-danger" type="button" :disabled="busy || loading" :aria-label="`Elimina ${row.source_path}`" @click="openDelete(row)">Elimina</button>
        </div>
      </template>
    </DataTable>

    <MasterDialog v-if="editor" :title="editing ? 'Modifica collegamento' : 'Nuovo collegamento'" description="Imposta il percorso da cui parte il QR e la sua destinazione." :busy="busy" @close="closeEditor">
      <form id="master-qr-editor" class="m-stack" novalidate @submit.prevent="save">
        <label class="m-field">Percorso QR di origine
          <input v-model="form.source_path" class="m-input" type="text" placeholder="/qrred" required :disabled="busy || Boolean(pendingRename)" :aria-invalid="Boolean(errors.source_path)" aria-describedby="master-qr-source-help" @input="errors.source_path = ''" />
          <small id="master-qr-source-help" :class="errors.source_path ? 'm-error' : 'm-muted-text'">{{ errors.source_path || 'Il percorso deve essere interno al sito.' }}</small>
        </label>
        <label class="m-field">Destinazione del reindirizzamento
          <input v-model="form.target_path" class="m-input" type="text" placeholder="/joy-volley" required :disabled="busy || Boolean(pendingRename)" :aria-invalid="Boolean(errors.target_path)" aria-describedby="master-qr-target-help" @input="errors.target_path = ''" />
          <small id="master-qr-target-help" :class="errors.target_path ? 'm-error' : 'm-muted-text'">{{ errors.target_path || 'Usa un percorso interno o un URL completo.' }}</small>
        </label>
        <p v-if="editing && !pendingRename" class="m-muted-text qr-rename-note">Se cambi il percorso di origine, quello precedente verrà rimosso. Per un nuovo percorso il contatore degli accessi parte da zero.</p>
        <p v-if="formError" class="m-error" role="alert">{{ formError }}</p>
      </form>
      <template #actions>
        <button class="m-btn" type="button" :disabled="busy" @click="closeEditor">{{ pendingRename ? 'Chiudi' : 'Annulla' }}</button>
        <button class="m-btn m-primary" type="submit" form="master-qr-editor" :disabled="busy">{{ busy ? 'Salvataggio…' : pendingRename ? 'Riprova rimozione' : 'Salva collegamento' }}</button>
      </template>
    </MasterDialog>

    <MasterDialog v-if="toDelete" title="Elimina collegamento QR" :description="`Il percorso ${toDelete.source_path} smetterà di reindirizzare e il conteggio degli accessi verrà eliminato.`" :busy="busy" @close="toDelete = null">
      <p v-if="formError" class="m-error" role="alert">{{ formError }}</p>
      <template #actions>
        <button class="m-btn" type="button" :disabled="busy" @click="toDelete = null">Annulla</button>
        <button class="m-btn m-danger" type="button" :disabled="busy" @click="confirmDelete">{{ busy ? 'Eliminazione…' : 'Elimina collegamento' }}</button>
      </template>
    </MasterDialog>
  </section>
</template>

<style scoped>
.qr-directory { min-width: 0; }
.qr-path { display: inline-block; max-width: 22ch; overflow-wrap: anywhere; color: var(--m-ink); }
.qr-row-actions { justify-content: flex-start; }
.qr-rename-note { margin: 0; font-size: .87rem; }
</style>
