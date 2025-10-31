<template>
  <div class="vote-trend-chart" role="img" :aria-label="accessibleLabel">
    <svg :viewBox="viewBox" xmlns="http://www.w3.org/2000/svg">
      <defs>
        <linearGradient :id="gradientId" x1="0" x2="0" y1="0" y2="1">
          <stop offset="0%" stop-color="rgba(14, 165, 233, 0.55)" />
          <stop offset="100%" stop-color="rgba(14, 165, 233, 0.05)" />
        </linearGradient>
      </defs>
      <path v-if="areaPath" :d="areaPath" :fill="gradientFill" />
      <path
        v-if="linePath"
        :d="linePath"
        fill="none"
        stroke="rgba(125, 211, 252, 0.9)"
        stroke-width="3"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
      <g v-if="chartDots.length">
        <template v-for="(dot, index) in chartDots" :key="`dot-${index}`">
          <circle :cx="dot.x" :cy="dot.y" r="4.2" fill="#0ea5e9">
            <title>{{ dot.tooltip }}</title>
          </circle>
          <text
            :x="dot.x"
            :y="dot.y - 10"
            text-anchor="middle"
            class="vote-trend-chart__dot-label"
          >
            {{ dot.label }}
          </text>
        </template>
      </g>
    </svg>

    <div v-if="resolvedWindow.start || resolvedWindow.end" class="vote-trend-chart__labels" aria-hidden="true">
      <span>{{ resolvedWindow.start }}</span>
      <span>{{ resolvedWindow.end }}</span>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue';

const props = defineProps({
  points: {
    type: Array,
    default: () => [],
  },
  width: {
    type: Number,
    default: 320,
  },
  height: {
    type: Number,
    default: 140,
  },
  paddingX: {
    type: Number,
    default: 18,
  },
  paddingY: {
    type: Number,
    default: 18,
  },
  accessibleLabel: {
    type: String,
    default: 'Andamento dei voti',
  },
  startLabel: {
    type: String,
    default: '',
  },
  endLabel: {
    type: String,
    default: '',
  },
});

const gradientId = `voteTrendFill-${Math.random().toString(36).slice(2, 10)}`;
const gradientFill = computed(() => `url(#${gradientId})`);

const timeFormatter = new Intl.DateTimeFormat('it-IT', {
  hour: '2-digit',
  minute: '2-digit',
});

function parsePoint(point) {
  if (!point) {
    return null;
  }

  const rawDate = point.date ?? point.timestamp ?? point.time ?? null;
  const date = rawDate instanceof Date ? rawDate : rawDate ? new Date(rawDate) : null;
  if (!date || Number.isNaN(date.valueOf())) {
    return null;
  }

  const rawValue = Number(point.value ?? point.votes ?? 0);
  if (!Number.isFinite(rawValue)) {
    return null;
  }

  const label = typeof point.label === 'string' ? point.label : '';
  const tooltip = typeof point.tooltip === 'string' ? point.tooltip : '';

  return {
    date,
    value: rawValue,
    label,
    tooltip,
  };
}

const normalizedPoints = computed(() => {
  return props.points
    .map((point) => parsePoint(point))
    .filter(Boolean)
    .sort((a, b) => a.date.getTime() - b.date.getTime());
});

const chartViewWidth = computed(() => Math.max(1, props.width));
const chartViewHeight = computed(() => Math.max(1, props.height));
const effectiveWidth = computed(() => Math.max(1, chartViewWidth.value - props.paddingX * 2));
const effectiveHeight = computed(() => Math.max(1, chartViewHeight.value - props.paddingY * 2));

const chartCoordinates = computed(() => {
  const points = normalizedPoints.value;
  if (!points.length) {
    return [];
  }

  const minTime = points[0].date.getTime();
  const maxTime = points[points.length - 1].date.getTime();
  const duration = Math.max(1, maxTime - minTime);
  const maxValue = points.reduce((acc, item) => Math.max(acc, item.value), 0);
  const safeMaxValue = Math.max(1, maxValue);

  return points.map((point, index) => {
    const timeRatio = duration === 0 ? index / Math.max(1, points.length - 1) : (point.date.getTime() - minTime) / duration;
    const xRatio = Number.isFinite(timeRatio) ? timeRatio : 0;
    const yRatio = point.value <= 0 ? 0 : point.value / safeMaxValue;

    return {
      x: props.paddingX + xRatio * effectiveWidth.value,
      y: chartViewHeight.value - props.paddingY - yRatio * effectiveHeight.value,
      value: point.value,
      date: point.date,
      label: point.label,
      tooltip: point.tooltip,
    };
  });
});

const linePath = computed(() => {
  const coords = chartCoordinates.value;
  if (!coords.length) {
    return '';
  }
  return coords
    .map((point, index) => `${index === 0 ? 'M' : 'L'}${point.x.toFixed(2)},${point.y.toFixed(2)}`)
    .join(' ');
});

const areaPath = computed(() => {
  const coords = chartCoordinates.value;
  if (!coords.length) {
    return '';
  }
  const baselineY = chartViewHeight.value - props.paddingY;
  const start = `M${coords[0].x.toFixed(2)},${baselineY.toFixed(2)}`;
  const lines = coords
    .map((point) => `L${point.x.toFixed(2)},${point.y.toFixed(2)}`)
    .join(' ');
  const end = `L${coords[coords.length - 1].x.toFixed(2)},${baselineY.toFixed(2)} Z`;
  return `${start} ${lines} ${end}`;
});

function formatTimeLabel(date, fallback = '') {
  if (!(date instanceof Date) || Number.isNaN(date.valueOf())) {
    return fallback;
  }
  try {
    return timeFormatter.format(date);
  } catch (error) {
    return fallback;
  }
}

const chartDots = computed(() => {
  const coords = chartCoordinates.value;
  if (!coords.length) {
    return [];
  }

  return coords.map((point) => {
    const tooltip = point.tooltip || `${Number(point.value || 0).toLocaleString('it-IT')} voti · ${point.label || formatTimeLabel(point.date)}`;
    return {
      x: Number(point.x.toFixed(2)),
      y: Number(point.y.toFixed(2)),
      tooltip,
      label: Number(point.value || 0).toLocaleString('it-IT'),
    };
  });
});

const viewBox = computed(() => `0 0 ${chartViewWidth.value} ${chartViewHeight.value}`);

const resolvedWindow = computed(() => {
  const points = normalizedPoints.value;
  if (!points.length) {
    return { start: props.startLabel || '', end: props.endLabel || '' };
  }

  const first = points[0];
  const last = points[points.length - 1];
  const start = props.startLabel || first.label || formatTimeLabel(first.date, '');
  const end = props.endLabel || last.label || formatTimeLabel(last.date, '');

  return { start, end };
});
</script>

<style scoped>
.vote-trend-chart {
  margin-top: 1rem;
  border-radius: 1.5rem;
  border: 1px solid rgba(148, 163, 184, 0.12);
  background: rgba(15, 23, 42, 0.6);
  padding: 1rem 1.1rem 1rem 1rem;
}

.vote-trend-chart svg {
  width: 100%;
  height: auto;
  display: block;
}

.vote-trend-chart__dot-label {
  font-size: 0.65rem;
  fill: #e2e8f0;
  font-weight: 600;
  pointer-events: none;
}

.vote-trend-chart__labels {
  display: flex;
  justify-content: space-between;
  margin-top: 0.6rem;
  font-size: 0.75rem;
  color: rgba(148, 163, 184, 0.7);
}
</style>
