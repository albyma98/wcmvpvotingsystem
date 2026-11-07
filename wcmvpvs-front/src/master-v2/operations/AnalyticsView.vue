<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { masterApi } from '../api';
import DataTable from '../ui/DataTable.vue';
import MasterIcon from '../ui/MasterIcon.vue';
import AnalyticsTrend from './AnalyticsTrend.vue';
import { dateLabel, delta, deltaTone, duration, monthLabel, number, percent } from './analyticsFormat';

const analytics = ref(null);
const loading = ref(false);
const error = ref('');
const clubTrendId = ref('');
const leaderboardSearch = ref('');
const sponsorSearch = ref('');
const engagementSearch = ref('');
const controller = new AbortController();

const list = value => Array.isArray(value) ? value.filter(Boolean) : [];
const leaderboard = computed(() => list(analytics.value?.organization_leaderboard));
const clubTrends = computed(() => list(analytics.value?.vote_trends?.per_organization));
const clubTrend = computed(() => clubTrends.value.find(club => String(club.organization_id) === clubTrendId.value));
const sponsorStats = computed(() => analytics.value?.sponsor_stats);
const engagement = computed(() => analytics.value?.engagement);
const monthly = computed(() => analytics.value?.monthly_summary);
const metrics = computed(() => [
  { label: 'Impression sponsor', value: number(sponsorStats.value?.total_impressions), hint: 'Esposizioni registrate' },
  { label: 'CTR medio sponsor', value: percent(sponsorStats.value?.average_ctr), hint: 'Tasso di clic complessivo' },
  { label: 'Tempo totale utenti', value: duration(engagement.value?.total_duration_seconds), hint: 'Sulle pagine evento' },
  { label: 'Tempo medio per utente', value: duration(engagement.value?.average_duration_per_user), hint: 'Sessione media per tifoso' },
  { label: 'Tempo medio per partita', value: duration(engagement.value?.average_duration_per_match), hint: 'Tempo complessivo per match' },
  { label: 'Voti mese corrente', value: number(monthly.value?.current?.votes), hint: monthLabel(monthly.value?.current?.month), change: monthly.value?.votes_change },
]);
const rankings = computed(() => [
  { name: 'Di sempre', events: list(analytics.value?.top_events?.all_time) },
  { name: 'Ultimi 7 giorni', events: list(analytics.value?.top_events?.last_7_days) },
]);
const leaderboardColumns = [
  { key: 'name', label: 'Società', sortable: true },
  { key: 'total_votes', label: 'Voti totali', numeric: true, sortable: true },
  { key: 'votes_last_7_days', label: 'Ultimi 7 giorni', numeric: true, sortable: true },
  { key: 'total_events', label: 'Eventi', numeric: true, sortable: true },
  { key: 'growth_percentage', label: 'Crescita', numeric: true, sortable: true },
];
const sponsorColumns = [
  { key: 'name', label: 'Società', sortable: true },
  { key: 'impressions', label: 'Impression', numeric: true, sortable: true },
  { key: 'clicks', label: 'Click', numeric: true, sortable: true },
  { key: 'ctr', label: 'CTR', numeric: true, sortable: true },
];
const engagementColumns = [
  { key: 'name', label: 'Società', sortable: true },
  { key: 'total_duration_seconds', label: 'Tempo totale', numeric: true, sortable: true },
  { key: 'average_duration_per_match', label: 'Media per partita', numeric: true, sortable: true },
  { key: 'average_duration_per_user', label: 'Media per utente', numeric: true, sortable: true },
];
const monthlyMetrics = [
  { key: 'votes', label: 'Voti' },
  { key: 'events', label: 'Partite' },
  { key: 'unique_users', label: 'Utenti unici' },
];

const clubName = club => club?.name || club?.slug || (club?.organization_id ? `Società #${club.organization_id}` : 'Società');
const clubSearch = club => [club.name, club.city, club.slug].filter(Boolean).join(' ');

async function load() {
  if (loading.value) return;
  loading.value = true;
  error.value = '';
  try {
    const { data } = await masterApi.get('/admin/master/analytics', { signal: controller.signal });
    if (!data || typeof data !== 'object') throw new Error('Risposta analytics non valida');
    analytics.value = data;
    if (!clubTrend.value) clubTrendId.value = clubTrends.value.length ? String(clubTrends.value[0].organization_id) : '';
  } catch {
    if (!controller.signal.aborted) error.value = 'Impossibile caricare le analytics. Riprova.';
  } finally {
    loading.value = false;
  }
}

onMounted(load);
onBeforeUnmount(() => controller.abort());
</script>

<template>
  <div class="m-stack" :aria-busy="loading">
    <section class="m-panel analytics-section" aria-labelledby="analytics-kpis">
      <header class="m-panel-head">
        <div><h2 id="analytics-kpis">Indicatori di piattaforma</h2><p>Coinvolgimento dei tifosi, voti e visibilità degli sponsor.</p></div>
        <button class="m-btn" type="button" :disabled="loading" @click="load"><MasterIcon name="refresh" :size="16" />{{ loading ? 'Aggiornamento…' : 'Aggiorna dati' }}</button>
      </header>
      <p v-if="error" class="m-error" role="alert">{{ error }}</p>
      <p v-if="loading && !analytics" class="m-muted-text" role="status">Caricamento delle analytics…</p>
      <dl v-if="analytics" class="analytics-metrics">
        <div v-for="item in metrics" :key="item.label" class="analytics-metric">
          <dt>{{ item.label }}</dt>
          <dd class="analytics-metric-value">{{ item.value }}</dd>
          <dd class="m-muted-text">{{ item.hint }}</dd>
          <dd v-if="item.change" :class="['m-badge', deltaTone(item.change.absolute)]" :aria-label="`Variazione rispetto al mese precedente ${delta(item.change)}`">{{ delta(item.change) }}</dd>
        </div>
      </dl>
    </section>

    <template v-if="analytics">
      <section class="m-panel analytics-section" aria-labelledby="analytics-vote-trends">
        <header class="m-panel-head"><div><h2 id="analytics-vote-trends">Andamento dei voti</h2><p>Votazioni giornaliere negli ultimi 30 giorni.</p></div></header>
        <div class="analytics-split">
          <AnalyticsTrend title="Totale piattaforma" :series="analytics.vote_trends?.global" />
          <div class="analytics-club-trend">
            <label v-if="clubTrends.length" class="m-field">Società per il confronto
              <select v-model="clubTrendId" class="m-input">
                <option v-for="club in clubTrends" :key="club.organization_id" :value="String(club.organization_id)">{{ clubName(club) }}</option>
              </select>
            </label>
            <AnalyticsTrend v-if="clubTrend" :title="clubName(clubTrend)" :series="clubTrend.data" />
            <p v-else class="m-muted-text">Nessun dato per società.</p>
          </div>
        </div>
      </section>

      <section class="m-panel analytics-section" aria-labelledby="analytics-leaderboard">
        <header class="m-panel-head"><div><h2 id="analytics-leaderboard">Classifica società</h2><p>Voti storici, eventi e crescita negli ultimi 7 giorni.</p></div></header>
        <DataTable v-model:search="leaderboardSearch" :rows="leaderboard" :columns="leaderboardColumns" row-key="organization_id" label="Classifica società" :search-text="clubSearch" search-placeholder="Nome, città o slug" empty="Nessuna società disponibile.">
          <template #cell-name="{ row }">
            <div class="analytics-club">
              <img v-if="row.logo_url" :src="row.logo_url" loading="lazy" alt="" />
              <div><strong>{{ clubName(row) }}</strong><small class="m-muted-text">{{ row.city || '—' }}</small></div>
            </div>
          </template>
          <template #cell-total_votes="{ value }">{{ number(value) }}</template>
          <template #cell-votes_last_7_days="{ value }">{{ number(value) }}</template>
          <template #cell-total_events="{ value }">{{ number(value) }}</template>
          <template #cell-growth_percentage="{ value }"><span :class="['m-badge', deltaTone(value)]">{{ percent(value) }}</span></template>
        </DataTable>
      </section>

      <section class="m-panel analytics-section" aria-labelledby="analytics-top-events">
        <header class="m-panel-head"><div><h2 id="analytics-top-events">Eventi più votati</h2><p>Le partite con più voti nello storico e nell’ultima settimana.</p></div></header>
        <div class="analytics-split">
          <article v-for="ranking in rankings" :key="ranking.name" class="analytics-ranking">
            <h3>{{ ranking.name }}</h3>
            <ol v-if="ranking.events.length">
              <li v-for="(event, index) in ranking.events" :key="event.event_id">
                <span class="rank-number" aria-hidden="true">{{ index + 1 }}</span>
                <div><strong>{{ event.label || 'Evento' }}</strong><small class="m-muted-text">{{ event.organization_name || '—' }} · {{ dateLabel(event.start_date) }}</small></div>
                <span class="m-badge m-muted">{{ number(event.total_votes) }} voti</span>
              </li>
            </ol>
            <p v-else class="m-muted-text">Nessun evento disponibile.</p>
          </article>
        </div>
      </section>

      <section class="m-panel analytics-section" aria-labelledby="analytics-monthly">
        <header class="m-panel-head"><div><h2 id="analytics-monthly">Riepilogo mensile</h2><p>Confronto tra il mese corrente e quello precedente.</p></div></header>
        <div class="m-table-scroll">
          <table class="m-table" aria-labelledby="analytics-monthly">
            <thead><tr><th scope="col">Indicatore</th><th scope="col">{{ monthLabel(monthly?.current?.month) }}</th><th scope="col">{{ monthLabel(monthly?.previous?.month) }}</th><th scope="col">Variazione</th></tr></thead>
            <tbody>
              <tr v-for="metric in monthlyMetrics" :key="metric.key">
                <th scope="row">{{ metric.label }}</th>
                <td>{{ number(monthly?.current?.[metric.key]) }}</td>
                <td>{{ number(monthly?.previous?.[metric.key]) }}</td>
                <td><span :class="['m-badge', deltaTone(monthly?.[`${metric.key}_change`]?.absolute)]">{{ delta(monthly?.[`${metric.key}_change`]) }}</span></td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section class="m-panel analytics-section" aria-labelledby="analytics-sponsors">
        <header class="m-panel-head"><div><h2 id="analytics-sponsors">Statistiche sponsor</h2><p>Impression e clic aggregati e per società.</p></div></header>
        <dl class="analytics-metrics analytics-metrics-compact">
          <div class="analytics-metric"><dt>Totale impression</dt><dd class="analytics-metric-value">{{ number(sponsorStats?.total_impressions) }}</dd></div>
          <div class="analytics-metric"><dt>Totale click</dt><dd class="analytics-metric-value">{{ number(sponsorStats?.total_clicks) }}</dd></div>
          <div class="analytics-metric"><dt>CTR medio</dt><dd class="analytics-metric-value">{{ percent(sponsorStats?.average_ctr) }}</dd></div>
        </dl>
        <DataTable v-model:search="sponsorSearch" :rows="list(sponsorStats?.organizations)" :columns="sponsorColumns" row-key="organization_id" label="Statistiche sponsor" :search-text="clubSearch" search-placeholder="Nome o slug della società" empty="Nessuna statistica sponsor disponibile.">
          <template #cell-name="{ row }">{{ clubName(row) }}</template>
          <template #cell-impressions="{ value }">{{ number(value) }}</template>
          <template #cell-clicks="{ value }">{{ number(value) }}</template>
          <template #cell-ctr="{ value }">{{ percent(value) }}</template>
        </DataTable>
      </section>

      <section class="m-panel analytics-section" aria-labelledby="analytics-engagement">
        <header class="m-panel-head"><div><h2 id="analytics-engagement">Tempo medio per società</h2><p>Tempo sulle pagine evento, totale e in media per partita e per tifoso.</p></div></header>
        <DataTable v-model:search="engagementSearch" :rows="list(engagement?.organizations)" :columns="engagementColumns" row-key="organization_id" label="Tempo medio per società" :search-text="clubSearch" search-placeholder="Nome o slug della società" empty="Nessun dato di coinvolgimento disponibile.">
          <template #cell-name="{ row }"><strong>{{ clubName(row) }}</strong><small class="analytics-club-slug m-muted-text">{{ row.slug || `ID ${row.organization_id}` }}</small></template>
          <template #cell-total_duration_seconds="{ value }">{{ duration(value) }}</template>
          <template #cell-average_duration_per_match="{ value }">{{ duration(value) }}</template>
          <template #cell-average_duration_per_user="{ value }">{{ duration(value) }}</template>
        </DataTable>
      </section>
    </template>
  </div>
</template>

<style scoped>
.analytics-section { min-width: 0; }
.analytics-metrics { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; margin: 0; }
.analytics-metric { min-width: 0; display: grid; align-content: start; gap: 5px; padding: 14px 16px; background: var(--m-bg); border: 1px solid var(--m-line); border-radius: var(--m-radius); }
.analytics-metric dt { font-weight: 600; font-size: .85rem; }
.analytics-metric dd { margin: 0; }
.analytics-metric-value { font-size: clamp(1.2rem, 2.5vw, 1.7rem); font-weight: 700; font-variant-numeric: tabular-nums; overflow-wrap: anywhere; }
.analytics-metric .m-badge { width: fit-content; }
.analytics-metric .m-muted-text { font-size: .8rem; }
.analytics-metrics-compact { margin-bottom: 20px; }
.analytics-split { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); align-items: start; gap: 22px; }
.analytics-club-trend { min-width: 0; }
.analytics-club-trend select { max-width: 100%; }
.analytics-club { display: flex; align-items: center; gap: 10px; }
.analytics-club img { width: 30px; height: 30px; flex: none; object-fit: contain; }
.analytics-club small, .analytics-club-slug, .analytics-ranking small { display: block; }
.analytics-ranking { padding: 14px; background: var(--m-bg); border: 1px solid var(--m-line); border-radius: var(--m-radius); }
.analytics-ranking h3 { margin: 0; font-size: .95rem; }
.analytics-ranking ol { list-style: none; padding: 0; margin: 8px 0 0; }
.analytics-ranking li { display: grid; grid-template-columns: auto minmax(0, 1fr) auto; gap: 10px; align-items: center; padding: 10px 0; }
.analytics-ranking li + li { border-top: 1px solid var(--m-line); }
.analytics-ranking strong { font-size: .86rem; overflow-wrap: anywhere; }
.rank-number { align-self: start; width: 1.5rem; height: 1.5rem; display: grid; place-items: center; border-radius: 6px; background: var(--m-soft); font-weight: 600; font-size: .8rem; }
.analytics-section th, .analytics-section td { font-variant-numeric: tabular-nums; }
@media (max-width: 1050px) { .analytics-split { grid-template-columns: 1fr; } }
@media (max-width: 620px) { .analytics-metrics { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@media (max-width: 360px) { .analytics-metrics { grid-template-columns: 1fr; } }
</style>
