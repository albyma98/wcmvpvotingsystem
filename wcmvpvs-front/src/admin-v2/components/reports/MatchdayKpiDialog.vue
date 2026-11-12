<script setup>
import { computed, nextTick, onBeforeUnmount, ref, useId, watch } from 'vue';
import { apiClient } from '../../../api';
import V2Dialog from '../shared/V2Dialog.vue';
import MatchdayMetricCard from './MatchdayMetricCard.vue';
import { formatKpi, formatKpiDate, manualChangeError, manualChanges, manualDrafts } from './matchdayKpis.js';
import '../../styles/admin-v2.css';

const props = defineProps({ event: { type: Object, default: null }, authHeaders: { type: Object, default: () => ({}) } });
const emit = defineEmits(['close', 'unauthorized', 'draft-cleared']);

const snapshot = ref(null);
const loading = ref(false);
const saving = ref(false);
const error = ref('');
const errorAction = ref('');
const success = ref('');
const scope = ref('');
const limit = ref(8);
const activeTab = ref('executive');
const manualOpen = ref(false);
const drafts = ref({});
const validation = ref({});
const confirmDiscard = ref(false);
const tabList = ref(null);
const manualHeading = ref(null);
const manualForm = ref(null);
const saveStatus = ref(null);
const id = useId();
let request;

const eventMeta = computed(() => snapshot.value?.event || {});
const eventDate = computed(() => formatKpiDate(eventMeta.value.start_datetime || props.event?.startDatetime));
const matchup = computed(() => [eventMeta.value.home_team, eventMeta.value.away_team].filter(Boolean).join(' – '));
const sections = computed(() => snapshot.value?.sections || []);
const tabs = computed(() => [{ id: 'executive', title: 'Executive' }, ...sections.value]);
const currentMetrics = computed(() => {
  if (activeTab.value !== 'executive') return sections.value.find(section => section.id === activeTab.value)?.metrics || [];
  const byCode = new Map(sections.value.flatMap(section => section.metrics.map(metric => [metric.code, metric])));
  return (snapshot.value?.executive || []).map(code => byCode.get(code)).filter(Boolean);
});
const coverage = computed(() => snapshot.value?.availability || {});
const manualFields = computed(() => snapshot.value?.manual_fields || []);
const changes = computed(() => manualChanges(manualFields.value, drafts.value));
const previous = computed(() => snapshot.value?.comparison?.previous_event);
const busy = computed(() => loading.value || saving.value);
const unitLabel = { count: 'numero', eur: '€', percent: '%', seconds: 'secondi', milliseconds: 'millisecondi' };

function reportError(cause, fallback) {
  const status = cause?.response?.status;
  if (status === 401) {
    emit('unauthorized');
    return;
  }
  if (status === 403) error.value = 'Il tuo account non può accedere ai KPI di questo evento.';
  else if (status === 404) error.value = 'L’evento non è disponibile.';
  else if (status === 400 && typeof cause.response.data?.message === 'string') error.value = cause.response.data.message;
  else error.value = fallback;
}

function updateReport(data, retained = {}) {
  if (data?.event?.id !== props.event?.id || !Array.isArray(data.sections) || !Array.isArray(data.executive)) {
    throw new Error('Invalid Matchday KPI response');
  }
  snapshot.value = data;
  scope.value = data.scope;
  limit.value = data.limit;
  drafts.value = manualDrafts(manualFields.value, retained);
  validation.value = {};
}

async function loadReport() {
  if (!props.event?.id) return;
  const retained = Object.fromEntries(changes.value.map(({ code }) => [code, drafts.value[code]]));
  request?.abort();
  const controller = new AbortController();
  request = controller;
  loading.value = true;
  error.value = '';
  errorAction.value = '';
  try {
    const { data } = await apiClient.get(`/admin/events/${props.event.id}/matchday-kpis`, {
      ...props.authHeaders,
      params: { scope: scope.value || undefined, limit: limit.value },
      signal: controller.signal,
    });
    if (!controller.signal.aborted) {
      updateReport(data, retained);
      await nextTick();
      if (!changes.value.length) clearDraftTracking();
    }
  } catch (cause) {
    if (controller.signal.aborted) return;
    if (snapshot.value) {
      scope.value = snapshot.value.scope;
      limit.value = snapshot.value.limit;
    }
    reportError(cause, 'Impossibile caricare i KPI Matchday.');
    errorAction.value = 'load';
  } finally {
    if (request === controller) loading.value = false;
  }
}

async function save() {
  if (busy.value || !changes.value.length || errorAction.value) return;
  const fields = new Map(manualFields.value.map(field => [field.code, field]));
  validation.value = Object.fromEntries(changes.value.flatMap(change => {
    const issue = manualChangeError(change, fields.get(change.code));
    return issue ? [[change.code, issue]] : [];
  }));
  if (Object.keys(validation.value).length) {
    await nextTick();
    manualHeading.value?.parentElement?.querySelector('[aria-invalid="true"]')?.focus();
    return;
  }

  const controller = new AbortController();
  request = controller;
  saving.value = true;
  error.value = '';
  success.value = '';
  try {
    const { data } = await apiClient.put(`/admin/events/${props.event.id}/matchday-kpis`, { values: changes.value }, {
      ...props.authHeaders,
      params: { scope: scope.value, limit: limit.value },
      signal: controller.signal,
    });
    if (!controller.signal.aborted) {
      updateReport(data);
      manualOpen.value = false;
      success.value = 'Dati salvati e KPI ricalcolati.';
      await nextTick();
      clearDraftTracking();
      saveStatus.value?.focus();
    }
  } catch (cause) {
    if (controller.signal.aborted) return;
    if (cause?.response?.status === 409) {
      error.value = 'Un altro aggiornamento ha cambiato i dati. Ricarica i valori salvati e verifica le tue modifiche prima di riprovare.';
      errorAction.value = 'conflict';
    } else {
      reportError(cause, 'Impossibile salvare i dati. Controlla i valori e riprova.');
    }
  } finally {
    if (request === controller) saving.value = false;
  }
}

function clearValidation(code) {
  delete validation.value[code];
  success.value = '';
}

function requestClose() {
  if (changes.value.length) confirmDiscard.value = true;
  else close();
}

function clearDraftTracking() {
  if (manualForm.value) emit('draft-cleared', manualForm.value);
}

function close() {
  clearDraftTracking();
  emit('close');
}

async function toggleEditor() {
  manualOpen.value = !manualOpen.value;
  confirmDiscard.value = false;
  await nextTick();
  if (manualOpen.value) manualHeading.value?.focus();
  else tabList.value?.querySelector('[aria-selected="true"]')?.focus();
}

async function handleTabKeys(event) {
  const index = tabs.value.findIndex(tab => tab.id === activeTab.value);
  const last = tabs.value.length - 1;
  const next = { ArrowLeft: index <= 0 ? last : index - 1, ArrowRight: index >= last ? 0 : index + 1, Home: 0, End: last }[event.key];
  if (next === undefined) return;
  event.preventDefault();
  activeTab.value = tabs.value[next].id;
  await nextTick();
  tabList.value?.querySelector('[aria-selected="true"]')?.focus();
}

watch(() => props.event?.id, eventId => {
  request?.abort();
  request = null;
  snapshot.value = null;
  loading.value = false;
  saving.value = false;
  error.value = '';
  errorAction.value = '';
  success.value = '';
  scope.value = '';
  limit.value = 8;
  activeTab.value = 'executive';
  manualOpen.value = false;
  confirmDiscard.value = false;
  drafts.value = {};
  validation.value = {};
  if (eventId && typeof window !== 'undefined') loadReport();
}, { immediate: true });
onBeforeUnmount(() => {
  request?.abort();
  clearDraftTracking();
});
</script>

<template>
  <V2Dialog :open="!!event" :busy="saving" title="Matchday KPI Analytics" wide @close="requestClose">
    <div class="matchday">
      <div class="matchday-event">
        <div>
          <h3>{{ eventMeta.title || event?.title }}</h3>
          <p v-if="matchup && matchup !== eventMeta.title">{{ matchup }}</p>
          <p class="v2-muted">{{ [eventDate, eventMeta.organization, eventMeta.division].filter(Boolean).join(' · ') }}</p>
        </div>
        <p v-if="snapshot" class="matchday-coverage" role="status">
          <strong>{{ coverage.available }} / {{ coverage.total }}</strong> KPI disponibili
          <small v-if="coverage.partial">{{ coverage.partial }} parziali</small>
        </p>
      </div>

      <p v-if="loading" role="status" class="v2-muted">Calcolo dei KPI Matchday…</p>
      <div v-if="error" class="matchday-feedback v2-error" role="alert">
        <p>{{ error }}</p>
        <button v-if="errorAction" class="v2-button" type="button" :disabled="busy" @click="loadReport">
          {{ errorAction === 'conflict' ? 'Ricarica valori salvati' : 'Riprova' }}
        </button>
      </div>
      <p v-if="success" ref="saveStatus" class="v2-success" role="status" tabindex="-1">{{ success }}</p>

      <template v-if="snapshot">
        <section v-show="manualOpen" class="matchday-manual">
          <h3 ref="manualHeading" tabindex="-1">Dati integrativi</h3>
          <p v-if="!manualFields.length" class="v2-empty">Non ci sono dati integrabili manualmente per questo evento.</p>
          <form v-else :id="`${id}-manual-form`" ref="manualForm" novalidate @submit.prevent="save">
            <fieldset v-for="field in manualFields" :key="field.code" :disabled="busy">
              <legend>{{ field.label }}</legend>
              <p v-if="field.description" class="v2-muted">{{ field.description }}</p>
              <div class="matchday-manual__fields">
                <label class="v2-field">
                  Valore ({{ unitLabel[field.unit] || field.unit }})
                  <input v-model="drafts[field.code].value" type="number" min="0" :step="field.unit === 'count' ? '1' : 'any'"
                    :aria-invalid="validation[field.code]?.input === 'value'"
                    :aria-describedby="validation[field.code]?.input === 'value' ? `${id}-${field.code}-error` : undefined" @input="clearValidation(field.code)" />
                </label>
                <label class="v2-field">
                  Fonte o nota
                  <input v-model="drafts[field.code].note" type="text" maxlength="500"
                    :aria-invalid="validation[field.code]?.input === 'note'"
                    :aria-describedby="validation[field.code]?.input === 'note' ? `${id}-${field.code}-error` : undefined" @input="clearValidation(field.code)" />
                </label>
              </div>
              <p v-if="validation[field.code]" :id="`${id}-${field.code}-error`" class="matchday-manual__error" role="alert">{{ validation[field.code].message }}</p>
              <p v-if="field.updated_at" class="v2-muted matchday-manual__saved">
                Valore salvato {{ formatKpi(field.value, field.unit) }}<template v-if="field.note"> · {{ field.note }}</template> · {{ formatKpiDate(field.updated_at) }}
                <template v-if="field.updated_by"> · {{ typeof field.updated_by === 'number' ? `Admin #${field.updated_by}` : field.updated_by }}</template>
              </p>
              <details v-if="field.history?.length" class="matchday-manual__history">
                <summary>Storico modifiche</summary>
                <ol>
                  <li v-for="entry in field.history" :key="entry.revision">
                    <strong>{{ formatKpi(entry.value, field.unit) }}</strong> · {{ entry.note }} · {{ formatKpiDate(entry.updated_at) }}
                    <template v-if="entry.updated_by"> · {{ typeof entry.updated_by === 'number' ? `Admin #${entry.updated_by}` : entry.updated_by }}</template>
                  </li>
                </ol>
              </details>
            </fieldset>
          </form>
        </section>

        <template v-if="!manualOpen">
          <div class="matchday-comparison">
            <div class="matchday-filters">
              <label class="v2-field">Confronta con
                <select v-model="scope" :disabled="busy" @change="loadReport">
                  <option value="club">Stessa società</option>
                  <option v-if="eventMeta.division || scope === 'division'" value="division">Stessa divisione</option>
                </select>
              </label>
              <label class="v2-field">Partite precedenti
                <select v-model.number="limit" :disabled="busy" @change="loadReport">
                  <option v-for="size in [1, 4, 8, 12, 20]" :key="size" :value="size">Fino a {{ size }}</option>
                </select>
              </label>
            </div>
            <div class="matchday-comparison__reference v2-muted" aria-live="polite">
              <p v-if="previous"><strong>Precedente</strong> · {{ [previous.title, formatKpiDate(previous.start_datetime)].filter(Boolean).join(' · ') }}</p>
              <p v-else>Storico insufficiente</p>
              <p v-if="snapshot.comparison?.events_considered"><strong>Media</strong> · {{ snapshot.comparison.events_considered }} {{ snapshot.comparison.events_considered === 1 ? 'partita precedente' : 'partite precedenti' }}<template v-if="snapshot.comparison.season"> · {{ snapshot.comparison.season }}</template></p>
            </div>
          </div>

          <div ref="tabList" class="v2-tabs matchday-tabs" role="tablist" aria-label="Reparti analytics" @keydown="handleTabKeys">
            <button v-for="tab in tabs" :id="`${id}-tab-${tab.id}`" :key="tab.id" type="button" role="tab"
              :aria-controls="`${id}-tabpanel`" :aria-selected="tab.id === activeTab" :tabindex="tab.id === activeTab ? 0 : -1"
              @click="activeTab = tab.id">{{ tab.title }}</button>
          </div>
          <section :id="`${id}-tabpanel`" :aria-labelledby="`${id}-tab-${activeTab}`" class="matchday-cards" role="tabpanel" tabindex="0">
            <p v-if="!currentMetrics.length" class="v2-empty matchday-cards__empty">Nessuna metrica disponibile in questa sezione.</p>
            <MatchdayMetricCard v-for="metric in currentMetrics" :key="metric.code" :metric="metric" :event="eventMeta" />
          </section>
        </template>
      </template>
    </div>

    <template #footer>
      <template v-if="confirmDiscard">
        <p class="matchday-discard" role="status">Le modifiche ai dati integrativi non sono ancora salvate.</p>
        <button class="v2-button" type="button" @click="confirmDiscard = false">Continua</button>
        <button class="v2-button v2-danger" type="button" @click="close">Chiudi senza salvare</button>
      </template>
      <template v-else>
        <button v-if="snapshot" class="v2-button" type="button" :disabled="busy" :aria-expanded="manualOpen" @click="toggleEditor">
          {{ manualOpen ? 'Torna ai KPI' : 'Integra dati mancanti' }}
        </button>
        <button v-if="manualOpen && (changes.length || saving)" class="v2-button v2-primary" type="submit" :form="`${id}-manual-form`" :disabled="busy || !!errorAction">
          {{ saving ? 'Salvataggio…' : 'Salva e ricalcola' }}
        </button>
      </template>
    </template>
  </V2Dialog>
</template>

<style scoped>
.matchday { display: grid; gap: 14px; min-width: 0; }
.matchday-event { display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 10px 20px; padding: 16px 18px; border-radius: 12px; background: var(--v2-primary-soft); }
.matchday-event h3 { font-size: clamp(18px, 2vw, 24px); letter-spacing: -.035em; line-height: 1.25; }
.matchday-event p { margin-top: 3px; }
.matchday-coverage { border-left: 1px solid #bbd2c7; padding-left: 14px; font-size: 13px; }
.matchday-coverage strong { display: block; font-size: 20px; line-height: 1.15; }
.matchday-coverage small { display: block; }
.matchday-comparison { display: flex; align-items: end; flex-wrap: wrap; gap: 10px 18px; }
.matchday-filters { display: flex; flex-wrap: wrap; gap: 10px; }
.matchday-filters label { min-width: 142px; }
.matchday-comparison__reference { font-size: 12px; padding-block: 3px; }
.matchday .matchday-tabs { margin: 0; padding: 4px 4px 6px; }
.matchday-tabs button { flex: 0 0 auto; white-space: nowrap; }
.matchday-cards { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 10px; min-width: 0; }
.matchday-cards__empty { grid-column: 1 / -1; }
.matchday-feedback { display: flex; align-items: center; justify-content: space-between; gap: 10px; flex-wrap: wrap; }
.matchday-manual, .matchday-manual form { display: grid; gap: 12px; }
.matchday-manual__fields { display: grid; grid-template-columns: minmax(110px, 1fr) 2fr; gap: 12px; margin-top: 8px; }
.matchday-manual .matchday-manual__error { color: var(--v2-danger); margin-top: 8px; }
.matchday-manual .matchday-manual__saved { font-size: 12px; margin-top: 8px; overflow-wrap: anywhere; }
.matchday-manual .matchday-manual__history { margin-top: 10px; padding: 7px 0 0; border: 0; border-top: 1px solid var(--v2-line); border-radius: 0; font-size: 12px; }
.matchday-manual__history ol { margin: 0; padding-left: 20px; overflow-wrap: anywhere; }
.matchday-discard { align-self: center; margin-inline-end: auto; }
@media (max-width: 1050px) { .matchday-cards { grid-template-columns: repeat(3, minmax(0, 1fr)); } }
@media (max-width: 760px) {
  .matchday .matchday-tabs { flex-wrap: nowrap; overflow-x: auto; }
  .matchday-cards { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}
@media (max-width: 440px) {
  .matchday-cards, .matchday-manual__fields { grid-template-columns: minmax(0, 1fr); }
  .matchday-filters { width: 100%; }
  .matchday-filters label { min-width: 0; flex: 1 1 120px; }
  .matchday-coverage { padding-left: 0; border-left: 0; }
  .matchday-event { padding: 12px; }
}
</style>
