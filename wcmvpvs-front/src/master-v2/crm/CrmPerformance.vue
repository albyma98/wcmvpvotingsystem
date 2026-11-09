
<script setup>
import { computed, reactive, ref, watch } from 'vue';
import { resolveApiUrl } from '../../api';
import {
  comparableMetric, contactTimeline, dateText, healthAssessment, healthLabels, issueStates,
  metricLabels, metricText, num, precedingMatch, priorities,
} from './model';

const props = defineProps({
  company: Object, records: Array, today: String, trend: Object, trendError: String, trendLoading: Boolean,
  busy: Boolean, ownerName: Function,
});
const emit = defineEmits(['edit', 'trend']);
const period = reactive({ from: '', to: '' });
const showArchived = ref(false);
watch(() => props.trend, (value) => {
  if (value) Object.assign(period, { from: value.from, to: value.to });
}, { immediate: true });
const lastCheckIn = computed(() => contactTimeline(props.records).find((item) => item.type === 'checkin'));
const health = computed(() => healthAssessment(props.company, props.records, props.trend, props.today));
const current = computed(() => props.trend?.matches[0]);
const previous = computed(() => precedingMatch(props.trend?.matches || []));
const read = (match, id) => match?.kpis.find((item) => item.id === id);
const sourceLabels = {
  operational: 'Dati operativi ABX', mixed: 'Operazioni e telemetria ABX', new_tracking: 'Telemetria ABX',
  manual: 'Dato manuale del report', manual_confirmed: 'Dato manuale del report · confermato',
  manual_unverified: 'Dato manuale del report · da verificare', estimated: 'Stima ABX', unavailable: 'ND',
};
const sourceText = (value) => sourceLabels[value] || (value?.startsWith('manual') ? 'Dato manuale del report' : value);
const listed = (kind) => props.records.filter((item) => item.kind === kind && (showArchived.value || !item.archived));
const periodValid = computed(() => period.from && period.to && period.from <= period.to);
const matchDelta = (a, b) => {
  if (!comparableMetric(a, b)) return 'ND';
  const value = a.value - b.value;
  return (value > 0 ? '+' : '') + num(value) + (a.unit === 'percent' ? ' p.p.' : '');
};
const lastGoalMatch = (goal) => (props.trend?.matches || []).find((match) =>
  match.concluded && (!goal.start || match.day >= goal.start) && (!goal.end || match.day <= goal.end)
    && (!goal.season || goal.season === props.company.season || (goal.start && goal.end)) && read(match, goal.metric)?.value != null);
const eventName = (id) => props.trend?.events.find((event) => event.id === id)?.title || 'Gara #' + id;
</script>

<template>
  <div class="crm-stack">
    <section class="m-panel crm-stack">
      <div class="crm-heading">
        <div><h2>Stato del cliente</h2><p>Valutazione calcolata sulle gare disponibili nel periodo e sulle attività aperte.</p></div>
        <button type="button" class="m-btn" :disabled="busy" @click="emit('edit', 'company', company, 'health')">Valutazione manuale</button>
      </div>
      <div class="m-grid">
        <div class="crm-item">
          <h3>Calcolata · <span :class="['m-badge', health.status === 'regular' ? 'm-good' : health.status === 'insufficient' ? 'm-muted' : 'm-warn']">{{ healthLabels[health.status] }}</span></h3>
          <ul><li v-for="reason in health.reasons" :key="reason">{{ reason }}</li></ul>
        </div>
        <div class="crm-item">
          <h3>Manuale · <span class="m-badge m-muted">{{ healthLabels[company.health] || 'Non impostata' }}</span></h3>
          <p v-if="company.health" class="crm-text">{{ company.healthReason }}</p>
        </div>
      </div>
      <dl class="crm-meta">
        <div><dt>Ultimo confronto con il club</dt><dd>{{ lastCheckIn ? dateText(lastCheckIn.date) : 'ND' }}</dd></div>
        <div><dt>Ultima gara con attività ABX</dt><dd>{{ trend?.lastActivity ? dateText(trend.lastActivity.day) + ' · ' + trend.lastActivity.title : 'ND' }}</dd></div>
        <div><dt>Gare attivate nel periodo</dt><dd>{{ trend && company.abxOrgId ? trend.activated : 'ND' }}</dd></div>
      </dl>
    </section>

    <section class="m-panel crm-stack">
      <header class="crm-heading"><div><h2>Andamento gare</h2><p>Fino alle 8 gare attivate più recenti nel periodo. Date in Europe/Rome.</p></div></header>
      <p v-if="!company.abxOrgId" class="m-muted-text">Collega una società ABX dalla panoramica per consultare le gare. Puoi già registrare obiettivi e misurazioni manuali.</p>
      <template v-else>
        <form class="crm-actions" @submit.prevent="emit('trend', { ...period })">
          <label class="m-field">Dal<input v-model="period.from" type="date" required /></label>
          <label class="m-field">Al<input v-model="period.to" type="date" required /></label>
          <button class="m-btn" type="submit" :disabled="trendLoading || !periodValid">Applica periodo</button>
        </form>
        <p v-if="trendLoading" role="status">Caricamento KPI delle gare…</p>
        <p v-else-if="trendError" class="m-error" role="alert">{{ trendError }} <button type="button" class="m-btn crm-small" @click="emit('trend')">Riprova</button></p>
        <p v-else-if="!current" class="m-muted-text">Nessuna gara con attivazione o attività rilevata nel periodo.</p>
        <template v-else>
          <p class="m-muted-text">Confronto tra {{ current.title }} ({{ dateText(current.day) }}) e {{ previous ? previous.title + ' (' + dateText(previous.day) + ')' : 'una gara precedente comparabile non disponibile nel periodo' }}.</p>
          <div class="m-table-scroll crm-table-tight" tabindex="0" aria-label="Confronto tra gare">
            <table class="m-table crm-kpi">
              <thead><tr><th>KPI e definizione</th><th>{{ dateText(current.day) }}</th><th>{{ previous ? dateText(previous.day) : 'Precedente' }}</th><th>Variazione</th></tr></thead>
              <tbody>
                <tr v-for="metric in current.kpis" :key="metric.id">
                  <td><strong>{{ metric.label }}</strong><small>{{ metric.display_description || metric.description }}</small></td>
                  <td>{{ metricText(metric.value, metric.unit) }}
                    <small v-if="metric.denominator != null">{{ num(metric.numerator) }} su {{ num(metric.denominator) }}</small>
                    <small v-else-if="metric.unit === 'percent'">Denominatore ND</small>
                    <small>{{ metric.value == null ? (metric.availability?.message || metric.note) : sourceText(metric.source) }}</small>
                    <small v-for="source in metric.provenance || []" :key="source.id">{{ source.source || source.source_name || source.note }} {{ source.observed_at || source.source_date || '' }}</small>
                  </td>
                  <td>{{ metricText(read(previous, metric.id)?.value, metric.unit) }}
                    <small v-if="read(previous, metric.id)?.denominator != null">{{ num(read(previous, metric.id).numerator) }} su {{ num(read(previous, metric.id).denominator) }}</small>
                    <small v-else-if="metric.unit === 'percent'">Denominatore ND</small>
                    <small v-if="read(previous, metric.id)?.value != null">{{ sourceText(read(previous, metric.id).source) }}</small>
                  </td>
                  <td>{{ matchDelta(metric, read(previous, metric.id)) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <p class="m-muted-text">La variazione richiede gare concluse con definizione, unità e fonte omogenee. Per i tassi la variazione è in punti percentuali.</p>
          <div class="m-table-scroll crm-table-tight" tabindex="0" aria-label="KPI delle gare recenti">
            <table class="m-table crm-kpi">
              <thead><tr><th>Gara</th><th v-for="metric in trend.definitions" :key="metric.id">{{ metric.label }}</th><th>Report</th></tr></thead>
              <tbody>
                <tr v-for="match in trend.matches" :key="match.id">
                  <td>{{ dateText(match.day) }}<small>{{ match.title }}</small><small v-if="!match.concluded">Gara non conclusa</small></td>
                  <td v-for="metric in match.kpis" :key="metric.id">{{ metricText(metric.value, metric.unit) }}
                    <small v-if="metric.denominator != null">{{ num(metric.numerator) }} su {{ num(metric.denominator) }}</small>
                    <small v-else-if="metric.unit === 'percent'">Denominatore ND</small>
                    <small v-if="metric.value != null">{{ sourceText(metric.source) }}</small>
                  </td>
                  <td><a v-if="match.reportPath" class="m-link" :href="resolveApiUrl(match.reportPath)" target="_blank" rel="noopener noreferrer">PDF gara</a><span v-else>—</span></td>
                </tr>
              </tbody>
            </table>
          </div>
        </template>
      </template>
    </section>

    <section class="m-panel crm-stack">
      <div class="crm-heading"><h2>Obiettivi e misurazioni manuali</h2>
        <div class="crm-actions">
          <button type="button" class="m-btn" :disabled="busy" @click="emit('edit', 'goal', {})">Aggiungi obiettivo</button>
          <button type="button" class="m-btn" :disabled="busy" @click="emit('edit', 'metric', {})">Registra dato manuale</button>
        </div>
      </div>
      <label class="crm-inline"><input v-model="showArchived" type="checkbox" /> Includi voci archiviate</label>
      <p v-if="!listed('goal').length && !listed('metric').length" class="m-muted-text">Nessun obiettivo o dato manuale registrato.</p>
      <article v-for="item in listed('goal')" :key="item.id" class="crm-item">
        <div class="crm-actions"><h3>{{ item.title }}</h3><span v-if="item.archived" class="m-badge m-muted">Archiviato</span></div>
        <p v-if="item.season || item.start || item.end">Periodo {{ item.season || '—' }} · {{ dateText(item.start) }} – {{ dateText(item.end) }}</p>
        <p v-if="item.metric">Soglia minima · {{ metricLabels[item.metric] }} · {{ metricText(item.value, trend?.definitions.find((metric) => metric.id === item.metric)?.unit) }}</p>
        <p v-if="item.metric">
          Ultimo valore da report · {{ metricText(read(lastGoalMatch(item), item.metric)?.value, read(lastGoalMatch(item), item.metric)?.unit) }}
          <template v-if="lastGoalMatch(item)"> · {{ dateText(lastGoalMatch(item).day) }} · {{ sourceText(read(lastGoalMatch(item), item.metric)?.source) }}</template>
        </p>
        <p class="crm-text">Risultati concordati · {{ item.result || 'ND' }}</p>
        <p v-if="item.notes" class="crm-text">{{ item.notes }}</p>
        <div class="crm-actions">
          <button class="m-btn crm-small" type="button" :disabled="busy" @click="emit('edit', 'goal', item)">Modifica obiettivo</button>
          <a v-if="item.reportUrl" class="m-link" :href="item.reportUrl" target="_blank" rel="noopener noreferrer">Report gara o stagione</a>
        </div>
      </article>
      <article v-for="item in listed('metric')" :key="item.id" class="crm-item">
        <div class="crm-actions"><h3>{{ item.title }}</h3><span class="m-badge m-muted">{{ item.archived ? 'Archiviata · ' : '' }}Manuale CRM</span></div>
        <p>{{ metricText(item.value, trend?.definitions.find((metric) => metric.id === item.metric)?.unit) }} {{ item.role }}<template v-if="item.denominator != null"> · denominatore {{ num(item.denominator) }}</template></p>
        <p>Fonte {{ item.source }} · {{ dateText(item.date) }}<template v-if="item.eventId"> · {{ eventName(item.eventId) }}</template></p>
        <p v-if="item.notes" class="crm-text">{{ item.notes }}</p>
        <div class="crm-actions">
          <button class="m-btn crm-small" type="button" :disabled="busy" @click="emit('edit', 'metric', item)">Modifica dato</button>
          <a v-if="item.reportUrl" class="m-link" :href="item.reportUrl" target="_blank" rel="noopener noreferrer">Apri fonte</a>
        </div>
      </article>
    </section>

    <section class="m-panel crm-stack">
      <div class="crm-heading"><h2>Criticità</h2><button type="button" class="m-btn m-primary" :disabled="busy" @click="emit('edit', 'issue', {})">Aggiungi criticità</button></div>
      <p v-if="!listed('issue').length" class="m-muted-text">Nessuna criticità registrata.</p>
      <article v-for="item in listed('issue')" :key="item.id" class="crm-item">
        <div class="crm-actions"><h3>{{ item.title }}</h3><span :class="['m-badge', item.status === 'resolved' ? 'm-good' : 'm-warn']">{{ issueStates[item.status] }}</span><span v-if="item.archived" class="m-badge m-muted">Archiviata</span></div>
        <p class="crm-text">{{ item.notes }}</p>
        <p>{{ priorities[item.priority] }} · {{ ownerName(item.ownerId) }} · Scadenza {{ dateText(item.due) }}</p>
        <p class="crm-text">Azione · {{ item.nextStep }}</p>
        <p v-if="item.completedAt">Risolta {{ dateText(item.completedAt) }}<template v-if="item.result"> · {{ item.result }}</template></p>
        <button class="m-btn crm-small" type="button" :disabled="busy" @click="emit('edit', 'issue', item)">Aggiorna criticità</button>
      </article>
    </section>
  </div>
</template>
