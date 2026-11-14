<script setup>
import { computed } from 'vue';
import { formatDelta, formatKpi, formatKpiDate, sourceLabels, statusLabels } from './matchdayKpis.js';

const props = defineProps({ metric: { type: Object, required: true }, event: { type: Object, required: true } });
const hasValue = computed(() => Number.isFinite(props.metric.value));
const hasRows = computed(() => !!props.metric.rows?.length);
const inlineRows = computed(() => props.metric.code === 'shop_conversions' && hasRows.value);
const shopColumn = computed(() => props.metric.columns?.[0]);
const comparison = computed(() => props.metric.comparison);
const hasComparison = computed(() => Number.isFinite(comparison.value?.previous) || Number.isFinite(comparison.value?.average));
const series = computed(() => {
  const history = comparison.value?.series || [];
  return [...history, {
    event_id: props.event.id,
    label: props.event.title,
    start_datetime: props.event.start_datetime,
    value: props.metric.value,
    current: true,
  }];
});
const numericSeries = computed(() => series.value.map(point => point.value).filter(Number.isFinite));
const showChart = computed(() => numericSeries.value.length > 1);
const chartRange = computed(() => {
  const lower = Math.min(0, ...numericSeries.value);
  const upper = Math.max(0, ...numericSeries.value);
  return { lower, span: upper - lower || 1 };
});

function chartStyle(value) {
  const { lower, span } = chartRange.value;
  const baseline = -lower / span * 100;
  return {
    '--zero': `${baseline}%`,
    '--start': `${baseline + Math.min(0, value || 0) / span * 100}%`,
    '--length': `${Math.abs(value || 0) / span * 100}%`,
  };
}
</script>

<template>
  <article class="matchday-metric" :class="{ 'matchday-metric--wide': hasRows && !inlineRows }">
    <header class="matchday-metric__header">
      <h3>{{ metric.name }}</h3>
      <div class="matchday-metric__badges">
        <span v-if="sourceLabels[metric.source]" class="v2-badge" :class="`matchday-metric__source--${metric.source}`">{{ sourceLabels[metric.source] }}</span>
        <span v-if="metric.status === 'partial' || metric.status === 'missing'" class="v2-badge warning">
          {{ metric.status === 'missing' ? 'Mancante' : statusLabels.partial }}
        </span>
      </div>
    </header>

    <strong v-if="hasValue" class="matchday-metric__value">{{ formatKpi(metric.value, metric.unit) }}</strong>
    <dl v-else-if="inlineRows && shopColumn" class="matchday-metric__shop">
      <div v-for="row in metric.rows" :key="row.label">
        <dt>{{ row.label }}</dt><dd>{{ formatKpi(row.values[shopColumn.id], shopColumn.unit) }}</dd>
      </div>
    </dl>
    <p v-else class="matchday-metric__status">
      {{ metric.status === 'not_applicable' && metric.reason === 'Funzione non attiva' ? metric.reason :
        metric.status === 'available' ? 'Nessuna voce registrata' : statusLabels[metric.status] || 'Dato mancante' }}
    </p>

    <dl v-if="hasComparison" class="matchday-metric__compare">
      <div v-if="Number.isFinite(comparison.previous)">
        <dt>Partita precedente</dt>
        <dd>
          {{ formatKpi(comparison.previous, metric.unit) }}
          <small v-if="Number.isFinite(comparison.absolute_delta)">
            {{ formatDelta(comparison.absolute_delta, metric.unit) }}<template v-if="Number.isFinite(comparison.percent_delta)"> ({{ formatDelta(comparison.percent_delta, 'relative') }}%)</template>
          </small>
        </dd>
      </div>
      <div v-if="Number.isFinite(comparison.average)">
        <dt>Media precedente</dt>
        <dd>
          {{ formatKpi(comparison.average, metric.unit) }}
          <small v-if="Number.isFinite(comparison.average_absolute_delta)">
            {{ formatDelta(comparison.average_absolute_delta, metric.unit) }}<template v-if="Number.isFinite(comparison.average_percent_delta)"> ({{ formatDelta(comparison.average_percent_delta, 'relative') }}%)</template>
          </small>
        </dd>
      </div>
    </dl>
    <p v-else-if="hasValue" class="matchday-metric__history-empty v2-muted">Storico insufficiente</p>

    <details class="matchday-metric__details">
      <summary :title="metric.formula || metric.description">{{ hasRows && !inlineRows ? 'Dati, formula e fonti' : 'Formula e fonti' }}</summary>
      <div class="matchday-metric__details-body">
        <p v-if="metric.description">{{ metric.description }}</p>
        <p v-if="metric.reason && metric.reason !== 'Funzione non attiva'" class="v2-muted">{{ metric.reason }}</p>
        <p v-if="metric.formula && metric.formula !== metric.description"><strong>Formula</strong> · {{ metric.formula }}</p>
        <p v-if="metric.numerator != null || metric.denominator != null">
          <strong>Numeratore</strong> {{ formatKpi(metric.numerator) }} · <strong>Denominatore</strong> {{ formatKpi(metric.denominator) }}
        </p>
        <p v-if="metric.sources?.length"><strong>Fonti</strong> · {{ metric.sources.join(' · ') }}</p>
        <p v-if="metric.missing?.length" class="v2-muted"><strong>Dati necessari</strong> · {{ metric.missing.join(' · ') }}</p>

        <div v-if="hasRows && !inlineRows && metric.columns?.length" class="v2-table-wrap" tabindex="0" :aria-label="metric.name">
          <table class="v2-table">
            <thead><tr><th scope="col">Voce</th><th v-for="column in metric.columns" :key="column.id" scope="col">{{ column.label }}</th></tr></thead>
            <tbody>
              <tr v-for="(row, index) in metric.rows" :key="index">
                <th scope="row">{{ row.label }}<small v-if="row.group">{{ row.group }}</small></th>
                <td v-for="column in metric.columns" :key="column.id">{{ formatKpi(row.values[column.id], column.unit) }}</td>
              </tr>
            </tbody>
          </table>
        </div>

        <figure v-if="showChart" class="matchday-metric__chart">
          <figcaption>Andamento per partita</figcaption>
          <ol>
            <li v-for="point in series" :key="point.event_id" :class="{ current: point.current }" :title="point.label">
              <span class="matchday-metric__bar" :style="chartStyle(point.value)" :class="{ 'has-value': Number.isFinite(point.value) }" aria-hidden="true"></span>
              <span class="matchday-metric__chart-value">{{ formatKpi(point.value, metric.unit) }}</span>
              <small>{{ point.current ? 'Attuale' : formatKpiDate(point.start_datetime, true) || point.label }}</small>
            </li>
          </ol>
        </figure>
      </div>
    </details>
  </article>
</template>

<style scoped>
.matchday-metric { display: flex; flex-direction: column; gap: 9px; min-width: 0; padding: 14px; background: var(--v2-surface); border: 1px solid var(--v2-line); border-radius: 12px; overflow-wrap: anywhere; }
.matchday-metric--wide { grid-column: 1 / -1; }
.matchday-metric__header, .matchday-metric__badges { display: flex; align-items: start; flex-wrap: wrap; gap: 5px 8px; }
.matchday-metric__header { justify-content: space-between; }
.matchday-metric__header h3 { font-size: 14px; line-height: 1.35; }
.matchday-metric__source--db { background: var(--v2-primary-soft); color: var(--v2-primary); }
.matchday-metric__source--manual { background: #eaf1fb; color: #254f7c; }
.matchday-metric__source--mixed { background: #eeeaf7; color: #574185; }
.matchday-metric__value { font-size: 28px; letter-spacing: -0.04em; font-variant-numeric: tabular-nums; line-height: 1.15; }
.matchday-metric__status { color: var(--v2-muted); font-weight: 600; }
.matchday-metric__compare, .matchday-metric__shop { display: grid; gap: 5px; margin: 0; }
.matchday-metric__compare div, .matchday-metric__shop div { display: flex; flex-wrap: wrap; justify-content: space-between; gap: 2px 8px; }
.matchday-metric__compare dt, .matchday-metric__history-empty { font-size: 12px; color: var(--v2-muted); }
.matchday-metric__compare dd, .matchday-metric__shop dd { margin: 0; font-weight: 650; }
.matchday-metric__compare dd { font-size: 12px; }
.matchday-metric__compare dd small { font-weight: 400; }
.matchday-metric__shop { font-size: 13px; }
.matchday-metric .matchday-metric__details { border: 0; border-top: 1px solid var(--v2-line); border-radius: 0; padding: 7px 0 0; margin-top: auto; background: transparent; }
.matchday-metric .matchday-metric__details summary { min-height: 22px; font-size: 12px; color: var(--v2-primary); }
.matchday-metric .matchday-metric__details[open] summary { margin-bottom: 9px; }
.matchday-metric__details-body { display: grid; gap: 8px; font-size: 12px; }
.matchday-metric__details-body tbody th { white-space: normal; }
.matchday-metric__details-body tbody th small { display: block; }
.matchday-metric__chart { min-width: 0; margin: 4px 0 0; }
.matchday-metric__chart figcaption { font-weight: 650; }
.matchday-metric__chart ol { display: flex; gap: 7px; overflow-x: auto; list-style: none; padding: 4px 0; margin: 0; }
.matchday-metric__chart li { min-width: 47px; flex: 1 0 min-content; display: grid; gap: 2px; text-align: center; }
.matchday-metric__chart li.current { font-weight: 700; }
.matchday-metric__bar { height: 58px; position: relative; background: var(--v2-canvas); border-radius: 4px; }
.matchday-metric__bar::before { content: ''; position: absolute; inset-inline: 0; bottom: var(--zero); height: 1px; background: var(--v2-line); }
.matchday-metric__bar.has-value::after { content: ''; position: absolute; inset-inline: 23%; bottom: var(--start); height: max(2px, var(--length)); background: #87aaa6; border-radius: 3px; }
.matchday-metric__chart .current .matchday-metric__bar::after { background: var(--v2-primary); }
.matchday-metric__chart-value { white-space: nowrap; font-variant-numeric: tabular-nums; }
</style>
