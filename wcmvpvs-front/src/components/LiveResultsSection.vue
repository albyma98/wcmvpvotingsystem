<template>
  <section class="px-4">
    <div class="live-results-card" role="region" aria-live="polite">
      <div class="live-results-card__decor"></div>
      <div class="live-results-card__content">
        <header class="live-results-card__header">
          <p class="live-results-card__eyebrow">Aggiornamento voti</p>
          <h2 class="live-results-card__title">Classifica in tempo reale</h2>
          <p class="live-results-card__subtitle">
            Scopri chi sta guidando il voto MVP e quando arrivano più preferenze.
          </p>
        </header>

        <div v-if="isLoading" class="live-results-card__state" role="status">
          <span class="live-results-card__spinner" aria-hidden="true"></span>
          <p>Caricamento dei risultati…</p>
        </div>

        <div v-else-if="errorMessage" class="live-results-card__state live-results-card__state--error">
          <p>{{ errorMessage }}</p>
        </div>

        <div v-else>
          <p v-if="!hasVotes" class="live-results-card__empty">I voti stanno arrivando…</p>

          <template v-else>
            <div class="live-results-summary">
              <p class="live-results-summary__label">Totale voti registrati</p>
              <p class="live-results-summary__value">{{ totalVotesLabel }}</p>
            </div>

            <ol class="live-leaderboard">
              <li
                v-for="(entry, index) in enhancedLeaderboard"
                :key="`${entry.id}-${index}`"
                class="live-leaderboard__item"
              >
                <div class="live-leaderboard__rank">{{ index + 1 }}</div>
                <div class="live-leaderboard__avatar" aria-hidden="true">
                  <img v-if="entry.imageUrl" :src="entry.imageUrl" alt="" />
                  <div v-else class="live-leaderboard__avatar-fallback">
                    <span>{{ entry.initials }}</span>
                  </div>
                </div>
                <div class="live-leaderboard__info">
                  <p class="live-leaderboard__name">{{ entry.name }}</p>
                  <p class="live-leaderboard__meta">
                    <span class="live-leaderboard__percentage">{{ entry.percentageLabel }}</span>
                    <span class="live-leaderboard__votes">{{ entry.votesLabel }} voti</span>
                  </p>
                </div>
              </li>
            </ol>

            <div class="live-chart" role="img" aria-label="Andamento dei voti minuto per minuto">
              <svg :viewBox="chartViewBox" xmlns="http://www.w3.org/2000/svg">
                <defs>
                  <linearGradient id="liveChartFill" x1="0" x2="0" y1="0" y2="1">
                    <stop offset="0%" stop-color="rgba(14, 165, 233, 0.55)" />
                    <stop offset="100%" stop-color="rgba(14, 165, 233, 0.05)" />
                  </linearGradient>
                </defs>
                <path v-if="chartAreaPath" :d="chartAreaPath" fill="url(#liveChartFill)" />
                <path v-if="chartLinePath" :d="chartLinePath" fill="none" stroke="rgba(125, 211, 252, 0.9)" stroke-width="3" stroke-linecap="round" stroke-linejoin="round" />
                <g v-if="chartDots.length">
                  <circle
                    v-for="(dot, dotIndex) in chartDots"
                    :key="`dot-${dotIndex}`"
                    :cx="dot.x"
                    :cy="dot.y"
                    r="4.2"
                    fill="#0ea5e9"
                  >
                    <title>{{ dot.tooltip }}</title>
                  </circle>
                </g>
              </svg>

              <div class="live-chart__labels" aria-hidden="true">
                <span>{{ timelineWindow.start }}</span>
                <span>{{ timelineWindow.end }}</span>
              </div>
            </div>
          </template>

          <p v-if="updatedLabel" class="live-results-card__updated">Aggiornato alle {{ updatedLabel }}</p>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup>
import { computed, onBeforeUnmount, ref, watch } from 'vue';
import { fetchLiveVoteSummary } from '../api';

const props = defineProps({
  eventId: {
    type: Number,
    required: true,
  },
  enabled: {
    type: Boolean,
    default: true,
  },
  pollInterval: {
    type: Number,
    default: 5000,
  },
});

const state = ref(null);
const isLoading = ref(false);
const errorMessage = ref('');
const isFetching = ref(false);
let pollTimer = null;

const chartWidth = 320;
const chartHeight = 140;
const chartPaddingX = 18;
const chartPaddingY = 18;

const resolvedSummary = computed(() => {
  if (!state.value || typeof state.value !== 'object') {
    return null;
  }

  const total = Number.isFinite(Number(state.value.total)) ? Number(state.value.total) : 0;
  const updatedAt = typeof state.value.updated_at === 'string' ? state.value.updated_at : '';

  const leaderboard = Array.isArray(state.value.leaderboard)
    ? state.value.leaderboard.map((entry) => ({
        playerId: Number(entry?.player_id) || 0,
        firstName: typeof entry?.first_name === 'string' ? entry.first_name : '',
        lastName: typeof entry?.last_name === 'string' ? entry.last_name : '',
        imageUrl: typeof entry?.image_url === 'string' ? entry.image_url : '',
        votes: Number(entry?.votes) || 0,
        percentage: Number.isFinite(entry?.percentage) ? Number(entry.percentage) : 0,
        lastVoteAt: typeof entry?.last_vote_at === 'string' ? entry.last_vote_at : '',
        displayName: typeof entry?.display_name === 'string' ? entry.display_name : '',
      }))
    : [];

  const timeline = Array.isArray(state.value.timeline)
    ? state.value.timeline
        .map((item) => ({
          timestamp: typeof item?.timestamp === 'string' ? item.timestamp : '',
          votes: Number(item?.votes) || 0,
        }))
        .filter((item) => item.timestamp)
    : [];

  return { total, updatedAt, leaderboard, timeline };
});

const totalVotes = computed(() => resolvedSummary.value?.total ?? 0);
const totalVotesLabel = computed(() => totalVotes.value.toLocaleString('it-IT'));

const leaderboardEntries = computed(() => resolvedSummary.value?.leaderboard ?? []);

const hasVotes = computed(
  () => totalVotes.value > 0 && leaderboardEntries.value.some((entry) => entry.votes > 0),
);

const enhancedLeaderboard = computed(() => {
  const total = totalVotes.value;

  return leaderboardEntries.value.map((entry, index) => {
    const firstName = entry.firstName?.trim() || '';
    const lastName = entry.lastName?.trim() || '';
    const preferredName = entry.displayName?.trim() || `${firstName} ${lastName}`.trim();
    const fallbackName = preferredName || `Giocatore #${entry.playerId || index + 1}`;
    const percentageValue = Number.isFinite(entry.percentage)
      ? entry.percentage
      : total > 0
      ? (entry.votes / total) * 100
      : 0;
    const hasDecimals = Math.abs(Math.round(percentageValue) - percentageValue) > 0.05;
    const percentageLabel = `${percentageValue.toLocaleString('it-IT', {
      minimumFractionDigits: hasDecimals ? 1 : 0,
      maximumFractionDigits: 1,
    })}%`;

    const votesLabel = entry.votes.toLocaleString('it-IT');
    const initialsSource = preferredName || `${firstName} ${lastName}`.trim();
    const initials = initialsSource
      .split(' ')
      .filter(Boolean)
      .slice(0, 2)
      .map((part) => part[0]?.toUpperCase())
      .join('');

    return {
      id: entry.playerId || index + 1,
      name: fallbackName,
      imageUrl: entry.imageUrl,
      votes: entry.votes,
      votesLabel,
      percentageValue,
      percentageLabel,
      initials: initials || 'MVP',
    };
  });
});

const parsedTimeline = computed(() => {
  const raw = resolvedSummary.value?.timeline ?? [];
  const points = raw
    .map((item) => {
      const date = new Date(item.timestamp);
      if (Number.isNaN(date.getTime())) {
        return null;
      }
      return {
        date,
        votes: item.votes,
      };
    })
    .filter(Boolean)
    .sort((a, b) => a.date.getTime() - b.date.getTime());

  if (!points.length) {
    return [];
  }

  return points;
});

const chartCoordinates = computed(() => {
  const points = parsedTimeline.value;
  if (!points.length) {
    return [];
  }

  const minTime = points[0].date.getTime();
  const maxTime = points[points.length - 1].date.getTime();
  const duration = Math.max(1, maxTime - minTime);
  const maxVotes = points.reduce((acc, point) => Math.max(acc, point.votes), 0);
  const safeMaxVotes = Math.max(1, maxVotes);

  const effectiveWidth = chartWidth - chartPaddingX * 2;
  const effectiveHeight = chartHeight - chartPaddingY * 2;

  return points.map((point, index) => {
    const rawRatio = duration === 0 ? index / Math.max(1, points.length - 1) : (point.date.getTime() - minTime) / duration;
    const xRatio = Number.isFinite(rawRatio) ? rawRatio : 0;
    const yRatio = point.votes <= 0 ? 0 : point.votes / safeMaxVotes;

    return {
      x: chartPaddingX + xRatio * effectiveWidth,
      y: chartHeight - chartPaddingY - yRatio * effectiveHeight,
      votes: point.votes,
      date: point.date,
    };
  });
});

const chartLinePath = computed(() => {
  const coords = chartCoordinates.value;
  if (!coords.length) {
    return '';
  }

  return coords
    .map((point, index) => `${index === 0 ? 'M' : 'L'}${point.x.toFixed(2)},${point.y.toFixed(2)}`)
    .join(' ');
});

const chartAreaPath = computed(() => {
  const coords = chartCoordinates.value;
  if (!coords.length) {
    return '';
  }

  const baselineY = chartHeight - chartPaddingY;
  const start = `M${coords[0].x.toFixed(2)},${baselineY.toFixed(2)}`;
  const lines = coords
    .map((point) => `L${point.x.toFixed(2)},${point.y.toFixed(2)}`)
    .join(' ');
  const end = `L${coords[coords.length - 1].x.toFixed(2)},${baselineY.toFixed(2)} Z`;
  return `${start} ${lines} ${end}`;
});

const chartDots = computed(() => {
  const coords = chartCoordinates.value;
  if (!coords.length) {
    return [];
  }

  return coords.map((point) => ({
    x: Number(point.x.toFixed(2)),
    y: Number(point.y.toFixed(2)),
    tooltip: `${formatVotesLabel(point.votes)} voti · ${formatTimeLabel(point.date)}`,
  }));
});

const chartViewBox = computed(() => `0 0 ${chartWidth} ${chartHeight}`);

const timelineWindow = computed(() => {
  const points = parsedTimeline.value;
  if (!points.length) {
    return { start: '', end: '' };
  }

  return {
    start: formatTimeLabel(points[0].date),
    end: formatTimeLabel(points[points.length - 1].date),
  };
});

const updatedLabel = computed(() => {
  const updatedAt = resolvedSummary.value?.updatedAt;
  if (!updatedAt) {
    return '';
  }
  const date = new Date(updatedAt);
  if (Number.isNaN(date.getTime())) {
    return '';
  }
  return formatUpdatedLabel(date);
});

const shouldPoll = computed(() => props.enabled && Number.isFinite(props.eventId) && props.eventId > 0);

function formatTimeLabel(date) {
  try {
    return new Intl.DateTimeFormat('it-IT', {
      hour: '2-digit',
      minute: '2-digit',
    }).format(date);
  } catch (error) {
    return '';
  }
}

function formatUpdatedLabel(date) {
  try {
    return new Intl.DateTimeFormat('it-IT', {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    }).format(date);
  } catch (error) {
    return '';
  }
}

function formatVotesLabel(value) {
  return Number(value || 0).toLocaleString('it-IT');
}

function clearPollTimer() {
  if (pollTimer && typeof window !== 'undefined') {
    window.clearInterval(pollTimer);
  }
  pollTimer = null;
}

function startPolling() {
  if (typeof window === 'undefined') {
    return;
  }
  clearPollTimer();
  const interval = Math.max(3000, Number.isFinite(props.pollInterval) ? Number(props.pollInterval) : 5000);
  pollTimer = window.setInterval(() => {
    loadData({ silent: true });
  }, interval);
}

async function loadData({ silent = false } = {}) {
  if (!shouldPoll.value || !props.eventId) {
    state.value = null;
    return;
  }

  if (isFetching.value) {
    return;
  }

  isFetching.value = true;
  if (!silent) {
    isLoading.value = true;
    errorMessage.value = '';
  }

  try {
    const { ok, data } = await fetchLiveVoteSummary(props.eventId);
    if (ok) {
      state.value = data ?? null;
      if (!silent) {
        errorMessage.value = '';
      }
    } else {
      if (!silent) {
        errorMessage.value = 'Impossibile aggiornare i risultati in questo momento.';
      }
    }
  } catch (error) {
    if (!silent) {
      errorMessage.value = 'Impossibile aggiornare i risultati in questo momento.';
    }
  } finally {
    if (!silent) {
      isLoading.value = false;
    }
    isFetching.value = false;
  }
}

watch(
  shouldPoll,
  (active) => {
    if (active) {
      loadData();
      startPolling();
    } else {
      clearPollTimer();
    }
  },
  { immediate: true },
);

watch(
  () => props.eventId,
  () => {
    state.value = null;
    if (shouldPoll.value) {
      loadData();
      startPolling();
    }
  },
);

watch(
  () => props.pollInterval,
  () => {
    if (shouldPoll.value) {
      startPolling();
    }
  },
);

onBeforeUnmount(() => {
  clearPollTimer();
});
</script>

<style scoped>
.live-results-card {
  position: relative;
  overflow: hidden;
  border-radius: 2.25rem;
  border: 1px solid rgba(100, 116, 139, 0.3);
  background: linear-gradient(145deg, rgba(15, 23, 42, 0.92), rgba(30, 41, 59, 0.88));
  box-shadow: 0 26px 52px rgba(8, 15, 28, 0.55);
}

.live-results-card__decor {
  pointer-events: none;
  position: absolute;
  inset: 0;
  background: radial-gradient(circle at top left, rgba(148, 163, 184, 0.18), transparent 58%);
}

.live-results-card__content {
  position: relative;
  padding: 1.5rem 1.75rem 1.75rem;
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.live-results-card__header {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.live-results-card__eyebrow {
  font-size: 0.65rem;
  letter-spacing: 0.45em;
  text-transform: uppercase;
  font-weight: 600;
  color: rgba(148, 163, 184, 0.9);
}

.live-results-card__title {
  font-size: clamp(1.1rem, 3.5vw, 1.4rem);
  font-weight: 700;
  color: #f8fafc;
}

.live-results-card__subtitle {
  font-size: 0.9rem;
  color: rgba(203, 213, 225, 0.85);
}

.live-results-card__state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
  padding: 1.75rem 1rem;
  font-size: 0.95rem;
  color: rgba(226, 232, 240, 0.9);
}

.live-results-card__state--error {
  color: rgba(248, 113, 113, 0.95);
}

.live-results-card__spinner {
  width: 1.75rem;
  height: 1.75rem;
  border-radius: 999px;
  border: 3px solid rgba(148, 163, 184, 0.2);
  border-top-color: rgba(14, 165, 233, 0.9);
  animation: live-spinner 1s linear infinite;
}

@keyframes live-spinner {
  0% {
    transform: rotate(0deg);
  }
  100% {
    transform: rotate(360deg);
  }
}

.live-results-card__empty {
  padding: 1.5rem 1rem;
  text-align: center;
  font-size: 1rem;
  color: rgba(203, 213, 225, 0.85);
  font-weight: 500;
}

.live-results-summary {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.live-results-summary__label {
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.3em;
  color: rgba(148, 163, 184, 0.85);
}

.live-results-summary__value {
  font-size: 1.75rem;
  font-weight: 700;
  color: #38bdf8;
}

.live-leaderboard {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.9rem;
}

.live-leaderboard__item {
  display: grid;
  grid-template-columns: auto auto 1fr;
  align-items: center;
  gap: 0.9rem;
  padding: 0.65rem 0.75rem;
  border-radius: 1.25rem;
  background: rgba(15, 23, 42, 0.65);
  border: 1px solid rgba(148, 163, 184, 0.12);
}

.live-leaderboard__rank {
  width: 2rem;
  height: 2rem;
  border-radius: 999px;
  background: rgba(14, 165, 233, 0.18);
  color: #38bdf8;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
}

.live-leaderboard__avatar {
  width: 2.4rem;
  height: 2.4rem;
  border-radius: 0.95rem;
  overflow: hidden;
  background: rgba(15, 23, 42, 0.8);
  display: flex;
  align-items: center;
  justify-content: center;
}

.live-leaderboard__avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.live-leaderboard__avatar-fallback {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  color: rgba(226, 232, 240, 0.95);
  background: rgba(148, 163, 184, 0.18);
}

.live-leaderboard__info {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}

.live-leaderboard__name {
  font-size: 1rem;
  font-weight: 600;
  color: #e2e8f0;
}

.live-leaderboard__meta {
  font-size: 0.85rem;
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  color: rgba(148, 163, 184, 0.85);
}

.live-leaderboard__percentage {
  color: #38bdf8;
  font-weight: 600;
}

.live-chart {
  margin-top: 1rem;
  border-radius: 1.5rem;
  border: 1px solid rgba(148, 163, 184, 0.12);
  background: rgba(15, 23, 42, 0.6);
  padding: 1rem 1.1rem 1rem 1rem;
}

.live-chart svg {
  width: 100%;
  height: auto;
  display: block;
}

.live-chart__labels {
  display: flex;
  justify-content: space-between;
  margin-top: 0.6rem;
  font-size: 0.75rem;
  color: rgba(148, 163, 184, 0.7);
}

.live-results-card__updated {
  margin-top: 0.25rem;
  font-size: 0.75rem;
  color: rgba(148, 163, 184, 0.75);
}

@media (min-width: 768px) {
  .live-results-card__content {
    padding: 2rem 2.5rem 2.25rem;
  }

  .live-results-summary__value {
    font-size: 2rem;
  }

  .live-leaderboard__item {
    gap: 1.1rem;
    padding: 0.85rem 1.05rem;
  }

  .live-leaderboard__rank {
    width: 2.4rem;
    height: 2.4rem;
  }

  .live-leaderboard__avatar {
    width: 2.8rem;
    height: 2.8rem;
    border-radius: 1.1rem;
  }
}
</style>
