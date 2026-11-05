<script setup>
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue';
import { apiClient } from '../api';
import { reportDownloadFilename } from '../engagementReport.js';
import { manualSourceErrors, recipientKpis } from '../recipientReport.js';
import ReportKpiCard from './ReportKpiCard.vue';
import ReportAnalysisView from './ReportAnalysisView.vue';

const props = defineProps({ eventId: { type: Number, required: true }, authHeaders: { type: Object, default: () => ({}) } });
const emit = defineEmits(['generated', 'unauthorized', 'busy']);
const snapshot = ref(null), recipient = ref('executive'), compare = ref(true), appendix = ref(false), includeAI = ref(true), notes = ref('');
const loading = ref(false), working = ref(false), aiLoading = ref(false), sourceSaving = ref(false), error = ref(''), approved = ref(null), previewUrl = ref('');
const sponsors = ref([]), sponsorID = ref(0), sponsorMode = ref('aggregate'), sourceOpen = ref(false), sourceErrors = ref([]), sourceMessage = ref(''), history = ref(null);
const sourceForm = reactive({ value: '', source_description: '', note: '', verified: false, confirmed: false, revision: 0 });
const analyses = reactive(new Map());
const versions = ref(null);
let controller, aiController;
const profile = computed(() => snapshot.value?.profiles?.find(p => p.id === recipient.value));
const kpis = computed(() => recipientKpis(snapshot.value, recipient.value));
const missing = computed(() => kpis.value.filter(k => k.value == null));
const recoverable = computed(() => missing.value.filter(k => k.availability?.manual_allowed));
const groups = computed(() => (profile.value?.groups || []).map(name => ({ name, kpis: kpis.value.filter(k => k.group === name) })).filter(g => g.kpis.length));
const analysisKey = computed(() => `${snapshot.value?.data_hash}|${recipient.value}|${compare.value}`);
const analysis = computed(() => analyses.get(analysisKey.value));
const busy = computed(() => loading.value || working.value || aiLoading.value || sourceSaving.value);
const sourceValidation = computed(() => manualSourceErrors(sourceForm));
const endpoint = computed(() => `/admin/events/${props.eventId}/engagement-report`);

function invalidatePreview() { approved.value = null; if (previewUrl.value) URL.revokeObjectURL(previewUrl.value); previewUrl.value = ''; }
watch([recipient, compare, appendix, includeAI, notes, analysis], invalidatePreview, { flush: 'sync' });
watch(busy, value => emit('busy', value), { immediate: true });
watch(recipient, () => { sourceOpen.value = false; if (snapshot.value?.sponsor_id && recipient.value !== 'sponsor') { sponsorMode.value = 'aggregate'; sponsorID.value = 0; load(); } });
function fail(cause, message) { if (cause?.response?.status === 401) emit('unauthorized'); error.value = cause?.response?.data?.message || cause?.response?.data?.error || message; }

async function load() {
  controller?.abort(); aiController?.abort(); controller = new AbortController(); const request = controller;
  loading.value = true; error.value = ''; invalidatePreview();
  const scope = recipient.value === 'sponsor' && sponsorMode.value === 'dedicated' ? Number(sponsorID.value) : 0;
  if (sponsorMode.value === 'dedicated' && recipient.value === 'sponsor' && !scope) { snapshot.value = null; loading.value = false; return; }
  // Remove stale sponsor-scoped data while fetching a different population.
  snapshot.value = null;
  try {
    const { data } = await apiClient.get(`${endpoint.value}?recipient=${recipient.value}${scope ? `&sponsor_id=${scope}` : ''}`, { ...props.authHeaders, signal: request.signal });
    if (request.signal.aborted) return;
    if (!data?.calculation_id || !Array.isArray(data.profiles)) throw new Error('Calcolo non valido');
    snapshot.value = data; sourceMessage.value = '';
  } catch (cause) { if (!request.signal.aborted) fail(cause, 'Impossibile caricare il report.'); }
  finally { if (controller === request) loading.value = false; }
}

async function loadSponsors() {
  try { const { data } = await apiClient.get('/admin/sponsors', props.authHeaders); sponsors.value = Array.isArray(data) ? data : data?.sponsors || []; }
  catch (cause) { fail(cause, 'Impossibile caricare gli sponsor.'); }
}
function changeSponsor() { load(); }

function openSources() {
  const source = snapshot.value?.manual_sources?.find(m => m.field_key === 'attendees');
  Object.assign(sourceForm, { value: source?.value ?? '', source_description: source?.source_description || '', note: source?.note || '', verified: source?.verification_status === 'verified', confirmed: false, revision: source?.revision || 0 });
  sourceErrors.value = []; sourceOpen.value = true; history.value = null;
}
async function saveSources() {
  sourceErrors.value = sourceValidation.value; if (sourceErrors.value.length) return;
  sourceSaving.value = true; error.value = '';
  try {
    await apiClient.put(`${endpoint.value}/sources`, { sources: [{ field_key: 'attendees', value: Number(sourceForm.value), unit: 'people', source_type: 'official_attendance', source_description: sourceForm.source_description.trim(), note: sourceForm.note.trim(), verification_status: sourceForm.verified ? 'verified' : 'unverified', revision: sourceForm.revision }] }, props.authHeaders);
    sourceOpen.value = false; await load();
    sourceMessage.value = 'Presenze salvate per l’evento. KPI e confronti ricalcolati; i dati rimangono disponibili tra destinatari.';
  } catch (cause) { fail(cause, 'Impossibile salvare le presenze.'); }
  finally { sourceSaving.value = false; }
}
async function showHistory() {
  try { const { data } = await apiClient.get(`${endpoint.value}/sources`, props.authHeaders); history.value = data.history || []; }
  catch (cause) { fail(cause, 'Impossibile caricare lo storico.'); }
}

async function generateAnalysis(regenerate = false) {
  if (!snapshot.value || busy.value) return;
  aiLoading.value = true; error.value = ''; aiController?.abort(); aiController = new AbortController(); const request = aiController;
  const key = analysisKey.value;
  try {
    const { data } = await apiClient.post(`${endpoint.value}/analysis`, { calculation_id: snapshot.value.calculation_id, recipient: recipient.value, compare_previous: compare.value, regenerate }, { ...props.authHeaders, signal: request.signal });
    if (!request.signal.aborted && data?.analysis) analyses.set(key, data);
  } catch (cause) { if (!request.signal.aborted) fail(cause, 'Analisi non disponibile. Puoi generare il PDF con la sintesi deterministica.'); }
  finally { if (request === aiController) aiLoading.value = false; }
}

async function renderReport(download = false) {
  if (!snapshot.value || busy.value) return;
  working.value = true; error.value = '';
  try {
    if (!approved.value) {
      const { data } = await apiClient.post(endpoint.value, { calculation_id: snapshot.value.calculation_id, recipient: recipient.value, compare_previous: compare.value, include_appendix: appendix.value, include_ai: includeAI.value, format: profile.value?.layout === 'analytics' || recipient.value === 'marketing' ? 'extended' : 'one_page', notes: notes.value.trim() }, props.authHeaders);
      if (!data?.id || !data.approved_at) throw new Error('Snapshot non valido');
      approved.value = data;
    }
    const { createEngagementReportPdf } = await import('../engagementReportPdf.js');
    const pdf = createEngagementReportPdf(approved.value);
    if (download) { pdf.save(reportDownloadFilename(approved.value)); emit('generated', { id: props.eventId, title: snapshot.value.event.title }); }
    else { if (previewUrl.value) URL.revokeObjectURL(previewUrl.value); previewUrl.value = URL.createObjectURL(pdf.output('blob')); }
  } catch (cause) { fail(cause, 'Impossibile generare il PDF. Lo snapshot approvato, se presente, resta salvato.'); }
  finally { working.value = false; }
}

async function loadVersions(older = false) {
  try {
    const cursor = older && versions.value?.length ? `?before=${encodeURIComponent(versions.value.at(-1).approved_at)}` : '';
    const { data } = await apiClient.get(`${endpoint.value}/snapshots${cursor}`, props.authHeaders);
    versions.value = older ? [...versions.value, ...data] : data;
  } catch (cause) { fail(cause, 'Impossibile caricare le versioni approvate.'); }
}
async function downloadVersion(id) {
  if (busy.value) return;
  working.value = true;
  try {
    const { data } = await apiClient.get(`${endpoint.value}/snapshots/${id}`, props.authHeaders);
    const { createEngagementReportPdf } = await import('../engagementReportPdf.js');
    createEngagementReportPdf(data).save(reportDownloadFilename(data));
  } catch (cause) { fail(cause, 'Impossibile scaricare la versione approvata.'); }
  finally { working.value = false; }
}

watch(() => props.eventId, () => { analyses.clear(); versions.value = null; sourceOpen.value = false; load(); loadSponsors(); }, { immediate: true });
onBeforeUnmount(() => { controller?.abort(); aiController?.abort(); if (previewUrl.value) URL.revokeObjectURL(previewUrl.value); emit('busy', false); });
</script>

<template>
  <div class="recipient-builder" :aria-busy="busy">
    <p class="recipient-builder__eyebrow">Report per destinatario</p>
    <nav class="recipient-builder__tabs" aria-label="Destinatario del report"><button v-for="p in snapshot?.profiles || [{ id: 'executive', name: 'Dirigenza' }, { id: 'marketing', name: 'Marketing' }, { id: 'sponsor', name: 'Sponsor' }, { id: 'analytics', name: 'Analitico' }]" :key="p.id" type="button" :aria-pressed="recipient === p.id" :disabled="busy" @click="recipient = p.id">{{ p.name }}</button></nav>
    <div v-if="recipient === 'sponsor'" class="recipient-builder__options"><label>Ambito sponsor<select v-model="sponsorMode" :disabled="busy" @change="changeSponsor"><option value="aggregate">Sponsor aggregati dell’evento</option><option value="dedicated">Singolo sponsor</option></select></label><label v-if="sponsorMode === 'dedicated'">Partner<select v-model="sponsorID" :disabled="busy" @change="changeSponsor"><option :value="0">Seleziona un partner</option><option v-for="s in sponsors" :key="s.id" :value="s.id">{{ s.report_name || s.name }}</option></select></label></div>
    <p v-if="error" role="alert" class="recipient-builder__error">{{ error }} <button type="button" :disabled="busy" @click="load">Ricarica</button></p>
    <p v-if="loading" role="status">Calcolo dei dati e confronto con la precedente partita conclusa…</p>
    <template v-if="snapshot && !loading">
      <h3>{{ profile?.name }}</h3><p>{{ profile?.description }}</p>
      <div class="recipient-builder__options"><label><input v-model="compare" type="checkbox" :disabled="busy" /> Confronta con partita precedente</label><label><input v-model="includeAI" type="checkbox" :disabled="busy" /> Includi analisi AI / sintesi</label><label><input v-model="appendix" type="checkbox" :disabled="busy || !profile?.allow_appendix" /> Includi appendice completa (20 KPI)</label></div>
      <p class="recipient-builder__caption">Sezioni PDF: {{ profile?.id === 'analytics' ? '20 indicatori, valori, denominatori e metodologia' : `${kpis.length} indicatori, confronto e sintesi` }}{{ appendix && profile?.allow_appendix ? ' + appendice dei 20 KPI' : '' }}. {{ profile?.id === 'analytics' ? 'Il corpo comprende già l’appendice.' : '' }}</p>
      <p v-if="compare" class="recipient-builder__caption">{{ snapshot.previous_event ? `Partita precedente: ${snapshot.previous_event.title} · ${snapshot.previous_event.start_datetime}` : 'Nessuna partita precedente conclusa disponibile.' }} La direzione della variazione non indica automaticamente un miglioramento.</p>
      <p v-for="conflict in snapshot.conflicts || []" :key="conflict" role="alert" class="recipient-builder__error">{{ conflict }}</p>
      <div v-if="missing.length" class="recipient-builder__notice"><strong>{{ missing.length }} indicatori N/D; {{ recoverable.length }} richiedono dati aggiuntivi recuperabili.</strong><p>Il completamento è facoltativo. {{ missing.length - recoverable.length }} indicatori restano N/D per dati non recuperabili manualmente.</p><button v-if="recoverable.length" type="button" :disabled="busy" @click="openSources">Completa dati mancanti</button></div>
      <p v-if="sourceMessage" role="status">{{ sourceMessage }}</p>
      <button v-if="snapshot.manual_sources?.length && !sourceOpen" type="button" :disabled="busy" @click="openSources">Verifica sorgente presenze salvata</button>
      <section v-if="sourceOpen" class="recipient-builder__source" aria-label="Completa sorgenti mancanti">
        <h3>Presenze ufficiali · Adozione del pubblico presente</h3><p>Si integra il denominatore. Il numeratore rimane il pubblico con provenienza arena rilevato automaticamente, che non certifica la presenza fisica. Senza tracciamento compatibile il KPI resta N/D.</p>
        <form @submit.prevent="saveSources"><div class="recipient-builder__source-fields"><label>Numero spettatori ufficiali<input v-model="sourceForm.value" type="number" min="0" max="10000000" step="1" required :disabled="busy" /></label><label>Fonte del dato<input v-model="sourceForm.source_description" type="text" minlength="3" maxlength="500" required placeholder="Es. distinta ufficiale / sistema biglietteria, riferimento" :disabled="busy" /></label><label>Nota facoltativa<textarea v-model="sourceForm.note" maxlength="1000" :disabled="busy"></textarea></label></div><label><input v-model="sourceForm.verified" type="checkbox" :disabled="busy" /> Ho verificato la fonte (altrimenti il report indica NON verificata)</label><label><input v-model="sourceForm.confirmed" type="checkbox" :disabled="busy" /> Confermo partita e popolazione di riferimento e il salvataggio delle presenze</label><ul v-if="sourceErrors.length || sourceForm.confirmed && sourceValidation.length" role="alert"><li v-for="message in sourceValidation" :key="message">{{ message }}</li></ul><div class="recipient-builder__actions"><button type="submit" :disabled="busy">Salva e ricalcola</button><button type="button" :disabled="busy" @click="sourceOpen = false">Salta completamento</button><button type="button" :disabled="busy" @click="showHistory">Storico modifiche</button></div></form>
        <ol v-if="history"><li v-for="(change, i) in history" :key="i">{{ change.changed_at }} · amministratore {{ change.changed_by }} · {{ change.previous?.value ?? 'N/D' }} → {{ change.current.value }} · {{ change.current.source_description }} · {{ change.current.verification_status }}</li></ol>
      </section>
      <section v-for="group in groups" :key="group.name"><h3>{{ group.name }}</h3><div class="recipient-builder__grid"><ReportKpiCard v-for="kpi in group.kpis" :key="kpi.id" :kpi="kpi" :compare="compare" :busy="busy" :event-context="!!snapshot.sponsor_id && kpi.id === 'visitors'" @complete="openSources" /></div></section>
      <section class="recipient-builder__ai"><div class="recipient-builder__actions"><button type="button" :disabled="busy || !includeAI" @click="generateAnalysis(false)">{{ aiLoading ? 'Analisi in corso…' : 'Genera analisi' }}</button><button type="button" :disabled="busy || !includeAI" @click="generateAnalysis(true)">Rigenera analisi</button></div><p class="recipient-builder__caption">L’AI si attiva solo su richiesta. I risultati identici usano la cache; in assenza dell’API il PDF include una sintesi deterministica.</p><ReportAnalysisView v-if="includeAI && analysis" :analysis="analysis.analysis" :status="analysis.status" :kpis="kpis" /></section>
      <details><summary>Limiti e metodologia</summary><ul><li v-for="(note, i) in snapshot.details?.coverage_notes || []" :key="i">{{ note }}</li></ul><p>Precedenza: automatico, manuale verificato, manuale non verificato, N/D. I conflitti sono segnalati. Le revisioni approvate conservano i dati originali.</p></details>
      <label class="recipient-builder__notes">Note del report<textarea v-model="notes" maxlength="4000" rows="2" :disabled="busy"></textarea></label>
      <div class="recipient-builder__actions"><button type="button" :disabled="busy" @click="renderReport(false)">Anteprima PDF</button><button type="button" class="recipient-builder__primary" :disabled="busy" @click="renderReport(true)">{{ working ? 'Generazione…' : 'Esporta PDF personalizzato' }}</button><button type="button" :disabled="busy" @click="load">Aggiorna sorgenti e confronto</button></div>
      <p class="recipient-builder__caption">Anteprima ed esportazione salvano uno snapshot versionato. La versione approvata resta riproducibile anche se le sorgenti cambiano successivamente.</p>
      <details @toggle="event => { if (event.target.open && versions === null) loadVersions(); }"><summary>Archivio versioni approvate</summary><p v-if="versions?.length === 0">Nessuna versione approvata per questa partita.</p><ul v-if="versions"><li v-for="version in versions" :key="version.id"><button type="button" :disabled="busy" @click="downloadVersion(version.id)">{{ version.recipient }}{{ version.sponsor ? ` · ${version.sponsor}` : '' }} · {{ version.approved_at }}</button></li></ul><button v-if="versions?.length && versions.length % 50 === 0" type="button" @click="loadVersions(true)">Carica versioni precedenti</button></details>
      <iframe v-if="previewUrl" :src="previewUrl" title="Anteprima del PDF approvato" class="recipient-builder__preview"></iframe>
    </template>
  </div>
</template>

<style scoped>
.recipient-builder{color:#243e50;font-size:13px;line-height:1.6}.recipient-builder__eyebrow{color:#087562;font-size:10px;font-weight:800;letter-spacing:.13em;text-transform:uppercase}.recipient-builder__tabs{display:grid;grid-template-columns:repeat(4,1fr);gap:8px;margin:12px 0 24px}.recipient-builder button{font:inherit;border:1px solid #c6d5df;border-radius:8px;padding:9px 13px;background:white;color:#274154;cursor:pointer}.recipient-builder button:disabled{opacity:.5;cursor:wait}.recipient-builder__tabs button[aria-pressed=true],.recipient-builder .recipient-builder__primary{background:#0e6659;color:white;border-color:#0e6659}.recipient-builder__grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px}.recipient-builder h3{font-size:15px;margin:22px 0 10px}.recipient-builder__options,.recipient-builder__actions{display:flex;flex-wrap:wrap;gap:12px 18px;margin:16px 0}.recipient-builder__caption{font-size:11px;color:#657389}.recipient-builder__notice{border:1px solid #dfc88a;background:#fffcf2;padding:16px;border-radius:12px;margin:18px 0;color:#805918}.recipient-builder__source{padding:20px;background:#f3f7fa;border:1px solid #c6d5df;border-radius:12px}.recipient-builder__source h3{margin-top:0}.recipient-builder__source-fields{display:grid;grid-template-columns:1fr 2fr;gap:14px}.recipient-builder__source label{display:block;margin:10px 0}.recipient-builder input[type=number],.recipient-builder input[type=text],.recipient-builder textarea,.recipient-builder select{display:block;box-sizing:border-box;width:100%;font:inherit;padding:10px;border:1px solid #c6d5df;border-radius:8px;background:white;margin-top:6px}.recipient-builder input[type=checkbox]{accent-color:#087562}.recipient-builder__source-fields>label:last-child{grid-column:1/-1}.recipient-builder__ai{margin:25px 0}.recipient-builder__error{background:#fff0ef;color:#9a2929;padding:12px;border-radius:8px}.recipient-builder__notes{display:block;margin-top:20px}.recipient-builder__preview{width:100%;height:720px;border:1px solid #c6d5df;border-radius:8px;margin-top:16px}.recipient-builder :focus-visible{outline:3px solid #23a98e;outline-offset:3px}.recipient-builder details{font-size:12px;margin:14px 0}.recipient-builder summary{cursor:pointer}@media(max-width:700px){.recipient-builder__tabs{grid-template-columns:1fr 1fr}.recipient-builder__grid,.recipient-builder__source-fields{grid-template-columns:1fr}.recipient-builder__preview{height:500px}}
</style>
