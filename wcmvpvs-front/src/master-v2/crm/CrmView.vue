
<script setup>
import { computed, onBeforeUnmount, ref, watch } from 'vue';
import { stopPostHogOnUntrackedPage } from '../../lib/posthog';
import { masterApi } from '../api';
import { useMasterWorkspace } from '../workspace';
import MasterDialog from '../ui/MasterDialog.vue';
import CrmHome from './CrmHome.vue';
import CrmCompany from './CrmCompany.vue';
import CrmEditor from './CrmEditor.vue';
import { companyAPI, companyName, crmPath, csvExample, errorText } from './model';
import './crm.css';

const props = defineProps({ route: { type: Object, required: true } });
stopPostHogOnUntrackedPage();
const { follow, navigate, setContext, useDirty } = useMasterWorkspace();
const home = ref(null);
const detail = ref(null);
const trend = ref(null);
const trendError = ref('');
const trendLoading = ref(false);
const loading = ref(true);
const loadError = ref('');
const notice = ref('');
const actionError = ref('');
const editor = ref(null);
const editorError = ref('');
const importing = ref(false);
const csv = ref('');
const busy = ref(false);
const company = computed(() => detail.value?.company);
const records = computed(() => detail.value?.records || []);
const companyId = computed(() => props.route.id);
const nav = { oggi: 'Oggi', societa: 'Società', pipeline: 'Pipeline' };
let controller;
let trendController;
let mounted = true;
let trendPeriod = {};

useDirty(() => busy.value);

async function loadPage() {
  controller?.abort();
  const request = new AbortController();
  controller = request;
  const id = companyId.value;
  loading.value = true;
  loadError.value = '';
  try {
    const [index, selected] = await Promise.all([
      masterApi.get(crmPath(), { signal: request.signal }),
      id ? masterApi.get(companyAPI(id), { signal: request.signal }) : null,
    ]);
    if (request.signal.aborted) return;
    home.value = index.data;
    detail.value = selected?.data || null;
  } catch (failure) {
    if (!request.signal.aborted) loadError.value = errorText(failure);
  } finally {
    if (!request.signal.aborted) loading.value = false;
  }
}

async function loadTrend(params = trendPeriod) {
  const id = companyId.value;
  if (!id || company.value?.id !== id) return;
  trendPeriod = params;
  trendController?.abort();
  const request = new AbortController();
  trendController = request;
  trendError.value = '';
  trendLoading.value = true;
  try {
    const { data } = await masterApi.get(companyAPI(id) + '/performance', { params, signal: request.signal });
    if (!request.signal.aborted) trend.value = data;
  } catch (failure) {
    if (!request.signal.aborted) trendError.value = errorText(failure);
  } finally {
    if (!request.signal.aborted) trendLoading.value = false;
  }
}

watch(companyId, () => {
  detail.value = null;
  trend.value = null;
  trendPeriod = {};
  trendError.value = '';
  trendLoading.value = false;
  trendController?.abort();
  loadPage();
}, { immediate: true });

watch([() => props.route.tab, () => company.value?.id, () => company.value?.abxOrgId], ([tab, id, abx], [, previousId, previousABX] = []) => {
  if (!id) return;
  if (id !== previousId || abx !== previousABX) {
    trendController?.abort();
    trend.value = null;
    trendError.value = '';
    trendLoading.value = false;
  }
  if (['andamento', 'piloti', 'contratti'].includes(tab) && !trend.value && !trendLoading.value) loadTrend();
});

watch([company, () => props.route], ([value]) => {
  setContext({ title: value ? companyName(value) : 'CRM' });
}, { immediate: true, flush: 'post' });

onBeforeUnmount(() => {
  mounted = false;
  controller?.abort();
  trendController?.abort();
});

function openEditor(kind, values = {}, mode) {
  if (busy.value) return;
  actionError.value = '';
  const today = home.value.today;
  const common = { companyId: companyId.value, ownerId: company.value?.ownerId || 0, season: company.value?.season || '' };
  const defaults = {
    company: { status: 'lead', priority: 'normal' }, contact: { date: '' },
    opportunity: { ...common, type: 'new', status: 'to_contact', priority: 'normal' },
    activity: { type: 'note' }, task: { ...common, type: 'followup', status: 'open', due: today, priority: 'normal' },
    pilot: { status: 'planned', start: today }, issue: { ...common, status: 'open', priority: 'normal', due: today },
    contract: { ...common, status: 'draft', start: today }, payment: { date: today },
    goal: { season: company.value?.season || '' }, metric: { date: today },
  };
  editorError.value = '';
  editor.value = { kind, initial: { ...defaults[kind], ...(kind === 'company' ? {} : { kind }), ...values }, mode };
  if (['pilot', 'metric'].includes(kind) && !trend.value && !trendLoading.value) loadTrend();
}

async function saveRecord(value) {
  const base = companyAPI(value.companyId || companyId.value) + '/records';
  return value.id ? masterApi.put(base + '/' + value.id, value) : masterApi.post(base, value);
}

async function commit(call, message, finish, errors = actionError) {
  if (busy.value) return;
  busy.value = true;
  notice.value = '';
  errors.value = '';
  try {
    const { data } = await call();
    if (!mounted) return;
    notice.value = typeof message === 'function' ? message(data) : message;
    busy.value = false;
    if (!finish?.(data)) await loadPage();
  } catch (failure) {
    if (mounted) errors.value = errorText(failure);
  } finally {
    if (mounted) busy.value = false;
  }
}

function saveEditor(value) {
  const isCompany = editor.value.kind === 'company';
  commit(
    () => isCompany
      ? (value.id ? masterApi.put(companyAPI(value.id), value) : masterApi.post(companyAPI(), value))
      : saveRecord(value),
    'Modifica salvata.',
    (data) => {
      editor.value = null;
      if (isCompany && !value.id) return navigate(crmPath(data.id));
      return false;
    },
    editorError,
  );
}

function changeTask(item) {
  commit(() => saveRecord({ ...item, status: item.status === 'done' ? 'open' : 'done' }), item.status === 'done' ? 'Attività riaperta.' : 'Attività completata.');
}

function toggleChecklist(item) {
  commit(() => saveRecord({ ...item, kind: 'checklist', status: item.status === 'done' ? 'open' : 'done' }), 'Checklist aggiornata.');
}

function createRenewal(contract) {
  commit(
    () => masterApi.post(companyAPI(companyId.value) + '/contracts/' + contract.id + '/renewal'),
    'Opportunità di rinnovo disponibile. Il primo confronto è nella coda attività.',
    (item) => { navigate(crmPath(companyId.value, 'opportunita', item.id)); return false; },
  );
}

function importCSV() {
  commit(
    () => masterApi.post(crmPath(null, 'import'), { csv: csv.value }),
    (data) => 'Import completato. Nuove società: ' + data.companiesAdded + ' · nuovi referenti: ' + data.contactsAdded + '.',
    (data) => {
      importing.value = false;
      csv.value = '';
      if (data.companyIds.length === 1) return navigate(crmPath(data.companyIds[0], 'referenti'));
      if (props.route.tab !== 'societa') navigate(crmPath(null, 'societa'));
      return false;
    },
    editorError,
  );
}

async function readCSV(event) {
  const file = event.target.files?.[0];
  if (!file) return;
  if (file.size > 380000) {
    editorError.value = 'Scegli un CSV inferiore a 380 KB.';
    return;
  }
  try {
    csv.value = await file.text();
    editorError.value = '';
  } catch {
    editorError.value = 'Impossibile leggere il file CSV.';
  }
}

function openImport() {
  editorError.value = '';
  csv.value = '';
  importing.value = true;
}
</script>

<template>
  <div class="crm-stack">
    <nav class="crm-tabs" aria-label="Sezioni CRM">
      <a v-for="(label, tab) in nav" :key="tab" class="m-btn" :aria-current="!route.id && route.tab === tab ? 'page' : undefined"
        :href="crmPath(null, tab === 'oggi' ? '' : tab)" @click="follow">{{ label }}</a>
    </nav>
    <p v-if="notice" class="crm-notice" role="status">{{ notice }}</p>
    <div v-if="actionError" class="m-error crm-actions" role="alert">{{ actionError }} <button class="m-btn" type="button" @click="loadPage">Ricarica</button></div>
    <div v-if="loadError" class="m-error crm-actions" role="alert">{{ loadError }} <button class="m-btn" type="button" @click="loadPage">Riprova</button></div>
    <div v-else-if="loading && (!home || (route.id && company?.id !== route.id))" class="m-panel" role="status">Caricamento CRM…</div>
    <CrmHome v-else-if="!route.id && home" :data="home" :tab="route.tab" :busy="busy || loading" :follow="follow" @edit="openEditor" @task="changeTask" @import="openImport" />
    <CrmCompany v-else-if="company?.id === route.id && home" :key="company.id" :company="company" :records="records" :route="route"
      :operators="home.operators" :organizations="home.organizations" :today="home.today" :trend="trend" :trend-error="trendError" :trend-loading="trendLoading"
      :busy="busy || loading" :follow="follow" @edit="openEditor" @task="changeTask" @checklist="toggleChecklist" @renewal="createRenewal" @trend="loadTrend" />

    <CrmEditor v-if="editor" :spec="editor" :company="company" :records="records" :operators="home.operators" :organizations="home.organizations"
      :events="trend?.events || []" :today="home.today" :busy="busy" :error="editorError" @save="saveEditor" @close="editor = null" />
    <MasterDialog v-if="importing" title="Importa società e referenti" :busy="busy" @close="importing = false">
      <form class="crm-stack" @submit.prevent="importCSV">
        <p>Carica o incolla un CSV UTF-8. Usa una riga per referente e ripeti il codice della società. Per aggiungere un nuovo referente usa un codice stabile oppure l’email. Le righe già importate vengono riconosciute.</p>
        <a class="m-link" :href="'data:text/csv;charset=utf-8,' + encodeURIComponent(csvExample)" download="esempio-crm.csv">Scarica esempio con due referenti</a>
        <label class="m-field">File CSV<input type="file" accept=".csv,text/csv" :disabled="busy" @change="readCSV" /></label>
        <label class="m-field">Contenuto CSV<textarea v-model="csv" :disabled="busy" required rows="8" maxlength="380000" /></label>
        <div v-if="editorError" class="m-error" role="alert">{{ editorError }}</div>
        <button type="submit" class="m-btn m-primary" :disabled="busy || !csv.trim()">{{ busy ? 'Importazione…' : 'Importa' }}</button>
      </form>
    </MasterDialog>
  </div>
</template>
