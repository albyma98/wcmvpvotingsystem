<template>
  <Teleport to="body">
    <div v-if="modelValue" class="eng-report-backdrop" @click.self="close">
      <section ref="panel" class="eng-report" role="dialog" aria-modal="true" aria-labelledby="eng-report-title" tabindex="-1" :aria-busy="loading || saving">
        <header class="eng-report__header">
          <div><p class="eng-report__eyebrow">ARENA ENGAGEMENT INTELLIGENCE</p><h2 id="eng-report-title">Verifica e genera il report</h2><p class="eng-report__event">{{ snapshot?.event?.title || eventTitle }}</p></div>
          <button class="eng-report__close" type="button" aria-label="Chiudi report" :disabled="saving" @click="close">×</button>
        </header>

        <div class="eng-report__body">
          <p v-if="loading" role="status" class="eng-report__loading">Calcolo dei 20 KPI dell’esperienza /new…</p>
          <div v-if="error" class="eng-report__error" role="alert">{{ error }}<button v-if="!snapshot && !loading" type="button" class="eng-report__link" @click="loadReport">Riprova</button></div>
          <template v-if="snapshot && !loading">
            <div class="eng-report__intro"><div><strong>Una fotografia verificabile dell’evento</strong><p>Ogni dato proviene dalla misurazione dedicata alla home /new. N/D indica una misura assente o un pubblico di riferimento non disponibile. I visitatori anonimi sono stimati per browser.</p></div><span class="eng-report__badge">20 KPI strategici</span></div>
            <p class="eng-report__meta">Calcolato il {{ dateLabel(snapshot.calculated_at) }}<template v-if="snapshot.measurement_started_at"> · Misurazione dal {{ dateLabel(snapshot.measurement_started_at) }}</template></p>
            <div v-if="snapshot.latest_report" class="eng-report__previous"><span>Ultimo report approvato: {{ dateLabel(snapshot.latest_report.approved_at) }}</span><button type="button" class="eng-report__link" :disabled="saving" @click="downloadPrevious">Scarica versione approvata</button></div>

            <section v-for="group in groups" :key="group.name" class="eng-report__group">
              <h3>{{ group.name }}</h3>
              <div class="eng-report__table-head" aria-hidden="true"><span>Indicatore e fonte</span><span>Rilevato</span><span>Valore report</span><span>Revisione</span></div>
              <article v-for="kpi in group.kpis" :key="kpi.id" class="eng-report__kpi" :class="{ 'is-corrected': kpi.corrected }">
                <div class="eng-report__kpi-name"><strong>{{ kpi.label }}</strong><details><summary>Definizione e fonte</summary><p>{{ kpi.description || formulaLabel(kpi) }}</p><p>Fonte: {{ sourceLabel(kpi.source) }}<template v-if="kpi.note"> · {{ kpi.note }}</template></p></details></div>
                <div class="eng-report__measured"><span class="eng-report__mobile-label">Rilevato</span>{{ formatReportValue(kpi.measured_value, kpi.unit) }}</div>
                <div class="eng-report__reviewed"><span class="eng-report__mobile-label">Valore report</span><strong>{{ formatReportValue(kpi.value, kpi.unit) }}</strong><small v-if="kpi.denominator !== undefined && kpi.numerator !== null && kpi.denominator !== null">{{ formatReportValue(kpi.numerator) }} / {{ formatReportValue(kpi.denominator) }}</small><small v-if="kpi.corrected">Corretto in revisione</small></div>
                <div class="eng-report__override"><button type="button" class="eng-report__link" :disabled="saving" @click="toggleOverride(kpi)">{{ overrides[kpi.id] ? 'Ripristina calcolo' : 'Modifica KPI' }}</button></div>
                <div v-if="overrides[kpi.id]" class="eng-report__override-fields"><label>Valore approvato (vuoto = N/D)<input v-model="overrides[kpi.id].value" :disabled="saving" type="number" min="0" :max="kpi.unit === 'percent' ? 100 : undefined" :step="kpi.unit === 'count' ? 1 : 'any'" :aria-label="`Valore approvato: ${kpi.label}`" /></label><label>Motivo della correzione *<input v-model="overrides[kpi.id].note" :disabled="saving" type="text" maxlength="500" :aria-label="`Motivo della correzione: ${kpi.label}`" /></label></div>
              </article>
            </section>

            <details class="eng-report__facts"><summary>Verifica o modifica i dati di base <span>Le percentuali si ricalcolano automaticamente</span></summary><div class="eng-report__facts-grid"><label v-for="key in factKeys" :key="key">{{ REPORT_FACT_LABELS[key] || key }}<input v-model="facts[key]" :disabled="saving" type="number" min="0" step="1" placeholder="N/D" /><small>Rilevato: {{ formatReportValue(snapshot.facts?.[key]) }}</small></label></div></details>
            <label class="eng-report__notes">Note per il report <span v-if="factsChanged">* Motiva le modifiche ai dati di base</span><textarea v-model="notes" :disabled="saving" rows="3" maxlength="4000" placeholder="Contesto dell’evento, verifica dei dati ed eventuali motivi di correzione…"></textarea><small>Le note complete restano archiviate nella versione approvata. Il PDF può abbreviarle per rispettare il formato scelto.</small></label>

            <fieldset class="eng-report__formats" :disabled="saving"><legend>Formato del PDF</legend><label :class="{ selected: format === 'one_page' }"><input v-model="format" type="radio" value="one_page" /><span><strong>One pager</strong><small>1 pagina · tutti i KPI e sintesi direzionale</small></span></label><label :class="{ selected: format === 'extended' }"><input v-model="format" type="radio" value="extended" /><span><strong>Report completo</strong><small>3 pagine · KPI, giochi, partner e metodologia</small></span></label></fieldset>
            <div v-if="validationErrors.length" class="eng-report__error" role="alert"><strong>Controlla i dati prima di confermare:</strong><ul><li v-for="item in validationErrors" :key="item">{{ item }}</li></ul></div>
            <p class="eng-report__signature">Il report è firmato con il solo logo dell’emittente. Dati originali, correzioni e note vengono conservati con un identificativo di approvazione.</p>
          </template>
        </div>

        <footer class="eng-report__footer"><button type="button" class="eng-report__button eng-report__button--light" :disabled="saving || loading || !snapshot" @click="resetReview">Ripristina dati rilevati</button><div><button type="button" class="eng-report__button eng-report__button--light" :disabled="saving" @click="close">Chiudi</button><button type="button" class="eng-report__button eng-report__button--primary" :disabled="loading || saving || !snapshot" @click="approveAndDownload">{{ saving ? 'Generazione in corso…' : approved ? 'Scarica PDF approvato' : 'Conferma e genera PDF' }}</button></div></footer>
      </section>
    </div>
  </Teleport>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, reactive, ref, watch } from 'vue';
import { apiClient } from '../api';
import { REPORT_KPIS, REPORT_FACT_LABELS, calculateReportKpis, formatReportValue, numericReportValue, reportDownloadFilename, validateReportReview } from '../engagementReport.js';

const props = defineProps({ modelValue: Boolean, eventId: { type: Number, default: 0 }, eventTitle: { type: String, default: '' }, authHeaders: { type: Object, default: () => ({}) } });
const emit = defineEmits(['update:modelValue', 'generated', 'unauthorized']);
const panel = ref(null), snapshot = ref(null), loading = ref(false), saving = ref(false), error = ref(''), approved = ref(null);
const facts = reactive({}), overrides = reactive({}), notes = ref(''), format = ref('one_page'), validationErrors = ref([]);
let requestController, previousFocus, previousOverflow, dialogOpen = false;
const factKeys = computed(() => [...new Set(REPORT_KPIS.flatMap(kpi => [kpi.numeratorKey, kpi.denominatorKey]).filter(Boolean))]);
const kpis = computed(() => calculateReportKpis(facts, snapshot.value?.kpis || [], overrides, snapshot.value?.facts || {}));
const groups = computed(() => [...new Set(REPORT_KPIS.map(kpi => kpi.group))].map(name => ({ name, kpis: kpis.value.filter(kpi => kpi.group === name) })));
const factsChanged = computed(() => factKeys.value.some(key => numericReportValue(facts[key]) !== numericReportValue(snapshot.value?.facts?.[key])));
watch([facts, overrides, notes, format], () => { approved.value = null; validationErrors.value = []; }, { deep: true, flush: 'sync' });

function dateLabel(value) { const date = new Date(value); return value && Number.isFinite(date.getTime()) ? new Intl.DateTimeFormat('it-IT', { dateStyle: 'short', timeStyle: 'short' }).format(date) : 'N/D'; }
function sourceLabel(value) { return ({ new_tracking: 'Misurazione home /new', manual: 'Inserimento manuale verificato', unavailable: 'Misura non disponibile' })[value] || value || 'Misura non disponibile'; }
function formulaLabel(kpi) { return kpi.denominatorKey ? `${REPORT_FACT_LABELS[kpi.numeratorKey]} / ${REPORT_FACT_LABELS[kpi.denominatorKey]}${kpi.unit === 'percent' ? ' × 100' : ''}.` : REPORT_FACT_LABELS[kpi.numeratorKey]; }
function close() { if (!saving.value) emit('update:modelValue', false); }
function resetReview() {
  Object.keys(facts).forEach(key => delete facts[key]);
  factKeys.value.forEach(key => { facts[key] = snapshot.value?.facts?.[key] ?? null; });
  Object.keys(overrides).forEach(key => delete overrides[key]);
  notes.value = ''; error.value = ''; validationErrors.value = []; approved.value = null;
}
function toggleOverride(kpi) { if (overrides[kpi.id]) delete overrides[kpi.id]; else overrides[kpi.id] = { value: kpi.value, note: '' }; }
function handleFailure(cause, fallback) {
  if (cause?.response?.status === 401) emit('unauthorized');
  error.value = typeof cause?.response?.data?.message === 'string' ? cause.response.data.message : typeof cause?.response?.data?.error === 'string' ? cause.response.data.error : fallback;
}
async function loadReport() {
  if (!props.eventId) return;
  requestController?.abort(); requestController = new AbortController();
  const controller = requestController;
  loading.value = true; error.value = ''; snapshot.value = null;
  try {
    const { data } = await apiClient.get(`/admin/events/${props.eventId}/engagement-report`, { ...props.authHeaders, signal: controller.signal });
    if (controller.signal.aborted) return;
    if (!data?.calculation_id || !Array.isArray(data.kpis)) throw new Error('Invalid report');
    snapshot.value = data; resetReview();
  } catch (cause) { if (!controller.signal.aborted) handleFailure(cause, 'Impossibile calcolare i KPI. Riprova.'); }
  finally { if (controller === requestController) loading.value = false; }
}
async function downloadSnapshot(report) {
  const { createEngagementReportPdf } = await import('../engagementReportPdf.js');
  const pdf = createEngagementReportPdf(report);
  pdf.save(reportDownloadFilename(report));
  emit('generated', { id: props.eventId, title: snapshot.value?.event?.title || props.eventTitle });
}
async function downloadPrevious() {
  if (!snapshot.value?.latest_report) return;
  saving.value = true; error.value = '';
  try { await downloadSnapshot(snapshot.value.latest_report); }
  catch (cause) { handleFailure(cause, 'Impossibile scaricare il PDF approvato. Riprova.'); }
  finally { saving.value = false; }
}
async function approveAndDownload() {
  if (!snapshot.value || saving.value) return;
  validationErrors.value = validateReportReview(facts, overrides, notes.value);
  if (factsChanged.value && !notes.value.trim()) validationErrors.value.push('Indica nelle note il motivo delle modifiche ai dati di base.');
  if (validationErrors.value.length) return;
  saving.value = true; error.value = '';
  try {
    if (!approved.value) {
      const cleanFacts = Object.fromEntries(Object.entries(facts).map(([key, value]) => [key, numericReportValue(value)]));
      const cleanOverrides = Object.fromEntries(Object.entries(overrides).map(([key, value]) => [key, { value: numericReportValue(value.value), note: value.note.trim() }]));
      const { data } = await apiClient.post(`/admin/events/${props.eventId}/engagement-report`, { calculation_id: snapshot.value.calculation_id, format: format.value, facts: cleanFacts, overrides: cleanOverrides, notes: notes.value.trim() }, props.authHeaders);
      if (!data?.id || !Array.isArray(data.kpis) || !data.approved_at) throw new Error('Invalid approval');
      approved.value = data;
      snapshot.value.latest_report = data;
    }
    await downloadSnapshot(approved.value);
  } catch (cause) { handleFailure(cause, approved.value ? 'Il report è approvato e salvato. Riprova a scaricare il PDF.' : 'Impossibile approvare il report. Verifica i dati e riprova.'); }
  finally { saving.value = false; }
}
function keyHandler(event) {
  if (event.key === 'Escape') { event.preventDefault(); close(); return; }
  if (event.key !== 'Tab' || !panel.value) return;
  const focusables = [...panel.value.querySelectorAll('button:not(:disabled), input:not(:disabled), textarea:not(:disabled), summary, [tabindex="0"]')].filter(node => node.getClientRects().length);
  if (!focusables.length) { event.preventDefault(); panel.value.focus(); return; }
  const first = focusables[0], last = focusables[focusables.length - 1];
  if (event.shiftKey && (document.activeElement === first || !panel.value.contains(document.activeElement) || document.activeElement === panel.value)) { event.preventDefault(); last.focus(); }
  else if (!event.shiftKey && (document.activeElement === last || !panel.value.contains(document.activeElement))) { event.preventDefault(); first.focus(); }
}
function releaseDialog() {
  requestController?.abort();
  if (!dialogOpen) return;
  document.removeEventListener('keydown', keyHandler, true); document.body.style.overflow = previousOverflow; previousFocus?.focus?.(); dialogOpen = false;
}
watch(() => [props.modelValue, props.eventId], async ([open]) => {
  if (!open) { releaseDialog(); return; }
  if (!dialogOpen) { previousFocus = document.activeElement; previousOverflow = document.body.style.overflow; document.body.style.overflow = 'hidden'; document.addEventListener('keydown', keyHandler, true); dialogOpen = true; }
  await nextTick(); panel.value?.focus(); loadReport();
}, { immediate: true });
onBeforeUnmount(releaseDialog);
</script>

<style scoped>
.eng-report-backdrop{position:fixed;inset:0;background:rgba(7,18,38,.68);backdrop-filter:blur(5px);z-index:10030;display:flex;align-items:center;justify-content:center;padding:24px;color:#17263b}
.eng-report{background:#fff;width:min(1120px,100%);max-height:calc(100dvh - 48px);display:grid;grid-template-rows:auto minmax(0,1fr) auto;border-radius:20px;box-shadow:0 32px 100px #07122655;overflow:hidden;outline:none;font-family:inherit}
.eng-report__header{padding:24px 28px 20px;background:#0e2139;color:#fff;display:flex;align-items:flex-start;justify-content:space-between;gap:20px}.eng-report__eyebrow{color:#6de0c2;font-size:10px;letter-spacing:.15em;font-weight:800;margin:0 0 9px}.eng-report__header h2{font-size:24px;line-height:1.2;margin:0}.eng-report__event{font-size:14px;color:#c3d4e9;margin:8px 0 0;overflow-wrap:anywhere}.eng-report__close{color:#fff;background:#ffffff15;border:1px solid #ffffff30;border-radius:10px;font-size:26px;line-height:1;width:38px;height:38px;cursor:pointer;flex-shrink:0}
.eng-report__body{padding:24px 28px;overflow:auto;overscroll-behavior:contain}.eng-report__intro{display:flex;justify-content:space-between;align-items:flex-start;gap:20px;background:#eff8f7;border:1px solid #d4eae5;border-radius:12px;padding:16px}.eng-report__intro p{font-size:12px;line-height:1.6;margin:7px 0 0;color:#516378}.eng-report__badge{font-size:11px;white-space:nowrap;background:#d2f1e9;color:#146451;border-radius:20px;padding:6px 10px;font-weight:700}.eng-report__meta{font-size:11px;color:#657389;margin:13px 0 22px}.eng-report__group{margin:22px 0}.eng-report__group h3{font-size:16px;margin:0 0 10px}.eng-report__table-head,.eng-report__kpi{display:grid;grid-template-columns:minmax(220px,1fr) 110px 130px 130px;gap:14px;align-items:center}.eng-report__table-head{background:#f4f6fa;font-size:10px;text-transform:uppercase;letter-spacing:.05em;color:#657389;padding:10px 12px;border-radius:8px}.eng-report__kpi{padding:14px 12px;border-bottom:1px solid #e7ecf2}.eng-report__kpi.is-corrected{background:#fffbef}.eng-report__kpi-name strong{font-size:13px}.eng-report__kpi-name details{font-size:11px;color:#657389;margin-top:5px}.eng-report__kpi-name summary{cursor:pointer}.eng-report__kpi-name details p{line-height:1.5;margin:6px 0}.eng-report__measured{font-size:13px;color:#657389}.eng-report__reviewed strong{font-size:17px;color:#0f6659;display:block}.eng-report__reviewed small{font-size:10px;display:block;margin-top:3px;color:#657389}.eng-report__mobile-label{display:none}.eng-report__link{border:0;padding:3px 0;background:transparent;color:#076d61;text-align:left;text-decoration:underline;text-underline-offset:3px;font-size:12px;cursor:pointer}.eng-report__override-fields{grid-column:1 / -1;display:grid;grid-template-columns:1fr 2fr;gap:15px;padding-top:5px}.eng-report label{font-size:12px;font-weight:600}.eng-report input[type=number],.eng-report input[type=text],.eng-report textarea{display:block;width:100%;box-sizing:border-box;font:inherit;font-weight:400;color:#17263b;background:#fff;border:1px solid #c8d3df;border-radius:8px;padding:10px;margin-top:6px}.eng-report textarea{resize:vertical;min-height:78px;max-height:220px}.eng-report :focus-visible{outline:3px solid #23a98e;outline-offset:3px}.eng-report__facts{background:#f6f8fb;padding:16px;border:1px solid #e2e8f0;border-radius:12px;margin:20px 0}.eng-report__facts summary{font-weight:700;font-size:13px;cursor:pointer}.eng-report__facts summary span{display:block;font-weight:400;color:#657389;font-size:11px;margin-top:5px}.eng-report__facts-grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:17px;margin-top:20px}.eng-report small{font-weight:400}.eng-report__facts-grid small,.eng-report__notes small{color:#657389;font-size:10px;display:block;margin-top:5px}.eng-report__notes{display:block}.eng-report__notes>span{color:#9b6210;font-size:11px;margin-left:8px}.eng-report__formats{padding:0;border:0;display:flex;gap:12px;margin:24px 0 15px}.eng-report__formats legend{font-size:13px;font-weight:700;margin-bottom:10px}.eng-report__formats label{flex:1;display:flex;gap:10px;align-items:center;border:1px solid #dbe3ec;padding:14px;border-radius:10px;cursor:pointer}.eng-report__formats label.selected{border-color:#148b76;background:#effaf7}.eng-report__formats input{accent-color:#11826e}.eng-report__formats strong,.eng-report__formats small{display:block}.eng-report__formats small{font-size:11px;color:#657389;margin-top:5px}.eng-report__signature{color:#657389;font-size:11px;line-height:1.6}.eng-report__footer{background:#f8fafc;border-top:1px solid #e1e7ef;padding:16px 28px;display:flex;gap:12px;justify-content:space-between}.eng-report__footer>div{display:flex;gap:10px}.eng-report__button{padding:11px 15px;border-radius:9px;border:1px solid #ccd7e3;font:inherit;font-size:12px;font-weight:700;cursor:pointer}.eng-report__button--light{background:#fff;color:#3e526b}.eng-report__button--primary{background:#0b7665;border-color:#0b7665;color:#fff}.eng-report button:disabled{opacity:.5;cursor:wait}.eng-report__error{font-size:12px;line-height:1.6;color:#9a2929;background:#fff0ef;border:1px solid #f0cdca;padding:12px 15px;border-radius:10px;margin-bottom:15px}.eng-report__error .eng-report__link{display:block;margin-top:8px}.eng-report__error ul{padding-left:18px;margin:5px 0}.eng-report__previous{font-size:11px;background:#f5f7fb;display:flex;gap:12px;justify-content:space-between;align-items:center;padding:10px 12px;border-radius:8px}.eng-report__loading{padding:45px 10px;text-align:center;color:#657389}
@media(max-width:760px){.eng-report-backdrop{padding:8px}.eng-report{max-height:calc(100dvh - 16px);border-radius:12px}.eng-report__header{padding:20px}.eng-report__header h2{font-size:20px}.eng-report__body{padding:18px}.eng-report__badge{display:none}.eng-report__table-head{display:none}.eng-report__kpi{grid-template-columns:1fr 1fr;gap:10px;padding:14px 4px}.eng-report__kpi-name{grid-column:1 / -1}.eng-report__override{grid-column:1 / -1}.eng-report__mobile-label{display:block;text-transform:uppercase;font-size:9px;color:#657389;margin-bottom:5px}.eng-report__facts-grid{grid-template-columns:1fr 1fr}.eng-report__override-fields{grid-template-columns:1fr}.eng-report__formats{flex-direction:column}.eng-report__footer{padding:12px;flex-wrap:wrap}.eng-report__footer>div{margin-left:auto;flex-wrap:wrap}.eng-report__button{padding:10px;font-size:11px}.eng-report__previous{align-items:flex-start;flex-direction:column}}
@media(prefers-reduced-motion:reduce){*{scroll-behavior:auto!important}}
</style>
