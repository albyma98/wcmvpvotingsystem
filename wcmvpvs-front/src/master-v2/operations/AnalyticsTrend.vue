<script setup>
import { computed } from 'vue';
import { number, trendPoints } from './analyticsFormat';

const props = defineProps({ title: String, series: Array });

const points = computed(() => trendPoints(props.series));
const axis = computed(() => {
  const maximum = Math.max(1, ...points.value.map(point => point.votes));
  const step = Math.ceil(maximum / 4);
  return [4, 3, 2, 1, 0].map(tick => ({ value: tick * step, y: 16 + (4 - tick) * 44 }));
});
const marks = computed(() => {
  if (!points.value.length) return [];
  const start = points.value[0].time;
  const span = points.value.at(-1).time - start;
  return points.value.map(point => ({
    ...point,
    x: span ? 54 + 642 * (point.time - start) / span : 375,
    y: 192 - 176 * point.votes / axis.value[0].value,
  }));
});
const line = computed(() => marks.value.map(mark => `${mark.x},${mark.y}`).join(' '));
const area = computed(() => {
  if (!marks.value.length) return '';
  return `${marks.value[0].x},192 ${line.value} ${marks.value.at(-1).x},192`;
});
const summary = computed(() => {
  if (!points.value.length) return '';
  const total = points.value.reduce((sum, point) => sum + point.votes, 0);
  const peak = Math.max(...points.value.map(point => point.votes));
  return `${number(total)} voti nel periodo · Picco di ${number(peak)} voti al giorno`;
});
</script>

<template>
  <figure class="analytics-trend">
    <figcaption>
      <strong>{{ title }}</strong>
      <span v-if="summary" class="m-muted-text">{{ summary }}</span>
    </figcaption>
    <template v-if="points.length">
      <svg viewBox="0 0 712 212" aria-hidden="true">
        <g v-for="tick in axis" :key="tick.y" class="chart-tick">
          <line x1="54" x2="696" :y1="tick.y" :y2="tick.y" />
          <text x="47" :y="tick.y + 4" text-anchor="end">{{ number(tick.value) }}</text>
        </g>
        <polygon :points="area" class="chart-area" />
        <polyline :points="line" class="chart-line" />
        <circle v-for="point in marks" :key="point.time" :cx="point.x" :cy="point.y" r="3" class="chart-mark">
          <title>{{ point.label }} · {{ number(point.votes) }} voti</title>
        </circle>
      </svg>
      <div class="chart-dates" aria-hidden="true"><span>{{ points[0].label }}</span><span>{{ points.at(-1).label }}</span></div>
      <details class="chart-data">
        <summary>Vedi i dati giornalieri</summary>
        <div class="m-table-scroll">
          <table class="m-table" :aria-label="`Voti per giorno · ${title}`">
            <thead><tr><th scope="col">Data</th><th scope="col">Voti</th></tr></thead>
            <tbody><tr v-for="point in points" :key="point.time"><td>{{ point.label }}</td><td>{{ number(point.votes) }}</td></tr></tbody>
          </table>
        </div>
      </details>
    </template>
    <p v-else class="m-muted-text">Nessun dato per il periodo.</p>
  </figure>
</template>

<style scoped>
.analytics-trend { min-width: 0; margin: 0; padding: 16px 0 0; }
figcaption { display: grid; gap: 4px; margin-bottom: 8px; }
figcaption span, .chart-dates, summary { font-size: 0.875rem; }
svg { display: block; width: 100%; height: auto; }
.chart-tick { font-size: 12px; fill: currentColor; }
.chart-tick line { stroke: currentColor; opacity: .15; }
.chart-area { fill: var(--m-soft); }
.chart-line { fill: none; stroke: var(--m-accent); stroke-width: 2.5; stroke-linecap: round; stroke-linejoin: round; }
.chart-mark { fill: var(--m-accent); stroke: var(--m-surface); stroke-width: 1; }
.chart-dates { display: flex; justify-content: space-between; margin: -9px 2% 0 7.6%; color: var(--m-muted, #58657a); }
.chart-data { margin-top: 16px; }
summary { cursor: pointer; width: fit-content; }
.chart-data .m-table-scroll { max-height: 22rem; margin-top: 8px; }
</style>
