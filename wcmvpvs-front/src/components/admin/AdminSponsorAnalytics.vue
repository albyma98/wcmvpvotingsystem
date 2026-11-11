<template>
  <section class="v2-panel v2-stack">
    <h2>Sponsor Analytics</h2>
    <form class="v2-stack" @submit.prevent="load">
      <div class="v2-grid-3">
        <label class="v2-field">Sponsor
          <select v-model="selection.sponsor">
            <option value="">Tutti gli sponsor</option>
            <option v-for="sponsor in sponsorOptions" :key="sponsor.id" :value="String(sponsor.id)">{{ sponsor.name }}</option>
          </select>
        </label>
        <label class="v2-field">Partita
          <select v-model="selection.match">
            <option value="">Tutte le partite e le visite senza partita</option>
            <option v-for="event in events" :key="event.id" :value="String(event.id)">{{ eventLabel(event) }}</option>
          </select>
        </label>
        <label class="v2-field">Stagione
          <select v-model.number="season" @change="setSeason">
            <option :value="0">Intervallo personalizzato</option>
            <option v-for="year in seasons" :key="year" :value="year">{{ year }}/{{ year + 1 }}</option>
          </select>
        </label>
      </div>
      <div class="v2-grid-3">
        <label class="v2-field">Dal (UTC)<input v-model="selection.from" type="date" required @input="season = 0" /></label>
        <label class="v2-field">Al (UTC)<input v-model="selection.to" type="date" :min="selection.from" required @input="season = 0" /></label>
        <div class="v2-toolbar"><button type="submit" class="v2-button v2-primary" :disabled="loading">Aggiorna</button></div>
      </div>
    </form>
    <p v-if="loading" class="v2-muted" role="status">Caricamento delle statistiche…</p>
    <p v-else-if="error" class="v2-error" role="alert">{{ error }}</p>
    <template v-else-if="report">
      <p v-if="!report.total.impressions && !report.total.clicks" class="v2-empty">Nessuna misurazione sponsor nel periodo.</p>
      <template v-else>
        <div class="v2-grid-4">
          <article v-for="metric in overview" :key="metric.label" class="v2-metric">
            <span>{{ metric.label }}</span><strong>{{ metric.value }}</strong>
            <small v-if="metric.note">{{ metric.note }}</small>
          </article>
        </div>
        <p class="v2-muted">La reach stima i browser distinti. Le impression geometriche dei giochi entrano nel totale misurato. La viewability riguarda l’inventario HTML e il CTR usa le impression HTML qualificate. Le durate usano le esposizioni con chiusura ricevuta.</p>

        <h3>Posizionamenti</h3>
        <div class="v2-table-wrap">
          <table class="v2-table">
            <thead><tr><th scope="col">Posizionamento</th><th scope="col">Impression</th><th scope="col">Viewable</th><th scope="col">Reach</th><th scope="col">Tempo totale</th><th scope="col">Clic</th><th scope="col">CTR</th></tr></thead>
            <tbody>
              <tr v-for="row in report.placements" :key="row.placement_id + row.game_id">
                <th scope="row">{{ placementLabel(row) }}</th>
                <td>{{ format(row.impressions) }}</td><td>{{ format(row.viewable_impressions) }}</td>
                <td>{{ format(row.reach) }}</td><td>{{ seconds(row.total_visible_ms) }}</td>
                <td>{{ format(row.clicks) }}</td><td>{{ format(row.ctr, '%') }}</td>
              </tr>
            </tbody>
          </table>
        </div>

        <details>
          <summary>Sponsor e creatività</summary>
          <div class="v2-table-wrap">
            <table class="v2-table">
              <thead><tr><th scope="col">Sponsor</th><th scope="col">Impression</th><th scope="col">Reach</th><th scope="col">Clic</th></tr></thead>
              <tbody><tr v-for="row in report.sponsors" :key="row.sponsor_id">
                <th scope="row">{{ row.sponsor_name }}</th><td>{{ format(row.impressions) }}</td>
                <td>{{ format(row.reach) }}</td><td>{{ format(row.clicks) }}</td>
              </tr></tbody>
            </table>
          </div>
          <div class="v2-table-wrap">
            <table class="v2-table">
              <thead><tr><th scope="col">Creatività</th><th scope="col">Posizionamento</th><th scope="col">Impression</th><th scope="col">Viewable</th><th scope="col">Reach</th><th scope="col">Clic</th></tr></thead>
              <tbody><tr v-for="row in report.creatives" :key="[row.sponsor_id,row.creative_id,row.placement_id,row.game_id].join(':')">
                <th scope="row"><span :title="row.creative_id">{{ row.sponsor_name }} · {{ row.creative_id.slice(0, 8) }}</span></th>
                <td>{{ placementLabel(row) }}</td><td>{{ format(row.impressions) }}</td>
                <td>{{ format(row.viewable_impressions) }}</td><td>{{ format(row.reach) }}</td><td>{{ format(row.clicks) }}</td>
              </tr></tbody>
            </table>
          </div>
        </details>

        <details>
          <summary>Andamento giornaliero</summary>
          <div class="v2-table-wrap">
            <table class="v2-table">
              <thead><tr><th scope="col">Giorno (UTC)</th><th scope="col">Impression</th><th scope="col">Reach giornaliera</th></tr></thead>
              <tbody><tr v-for="row in report.trend" :key="row.day">
                <th scope="row"><time :datetime="row.day">{{ row.day }}</time></th>
                <td><meter :value="row.impressions" :max="peak" aria-label="Impression">{{ row.impressions }}</meter> {{ format(row.impressions) }}</td>
                <td><meter v-if="row.reach !== null" :value="row.reach" :max="peak" aria-label="Reach">{{ row.reach }}</meter> {{ format(row.reach) }}</td>
              </tr></tbody>
            </table>
          </div>
        </details>
      </template>
    </template>
  </section>
</template>

<script setup>
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue';
import { apiClient } from '../../api';

const props = defineProps({
  events: { type: Array, default: () => [] },
  sponsors: { type: Array, default: () => [] },
  authHeaders: { type: Object, required: true },
  eventLabel: { type: Function, default: (event) => `Partita #${event.id}` },
});

const today = new Date();
const currentSeason = today.getUTCFullYear() - Number(today.getUTCMonth() < 6);
const seasons = Array.from({ length: 5 }, (_, index) => currentSeason - index);
const season = ref(currentSeason);
const selection = reactive({ sponsor: '', match: '', from: '', to: '' });
const report = ref(null);
const knownSponsors = ref([]);
const loading = ref(false);
const error = ref('');
let request;

function setSeason() {
  if (!season.value) return;
  selection.from = `${season.value}-07-01`;
  selection.to = season.value === currentSeason ? today.toISOString().slice(0, 10) : `${season.value + 1}-06-30`;
}
setSeason();

const sponsorOptions = computed(() => {
  const options = new Map(knownSponsors.value.map((item) => [item.sponsor_id, { id: item.sponsor_id, name: item.sponsor_name }]));
  for (const sponsor of props.sponsors) options.set(sponsor.id, sponsor);
  return [...options.values()].sort((a, b) => a.name.localeCompare(b.name, 'it'));
});
const formatter = new Intl.NumberFormat('it-IT', { maximumFractionDigits: 1 });
const format = (number, suffix = '') => number == null ? 'N/D' : formatter.format(number) + suffix;
const seconds = (ms) => ms == null ? 'N/D' : format(ms / 1000, ' s');
const peak = computed(() => Math.max(1, ...(report.value?.trend || []).map((row) => row.impressions)));
const overview = computed(() => {
  const m = report.value.total;
  return [
    { label: 'Impression misurabili', value: format(m.impressions), note: `${format(m.geometric_impressions)} geometriche` },
    { label: 'Impression viewable', value: format(m.viewable_impressions), note: `${format(m.viewability_rate, '%')} di viewability HTML` },
    { label: 'Reach stimata', value: format(m.reach) },
    { label: 'Frequenza media', value: format(m.frequency) },
    { label: 'Tempo totale', value: seconds(m.total_visible_ms), note: `${format(m.duration_samples)} esposizioni con durata` },
    { label: 'Tempo medio', value: seconds(m.average_visible_ms) },
    { label: 'Clic', value: format(m.clicks), note: `${format(m.unique_clickers)} browser distinti` },
    { label: 'CTR', value: format(m.ctr, '%') },
    { label: 'Full-view rate', value: format(m.full_view_rate, '%'), note: 'Conferma MVP' },
    { label: 'Conversioni attribuite', value: format(m.conversions) },
  ];
});

const games = { basketball: 'Basket Rush', playmaker: 'Playmaker', crossover: 'Crossover Rush' };
const placements = {
  home_footer: 'Partner in home', post_vote_left: 'Conferma MVP · sinistra', post_vote_right: 'Conferma MVP · destra',
  sponsor_rush_entry: 'Sponsor Rush · scheda missione', sponsor_rush_logo: 'Sponsor Rush · logo in gioco',
  game_wall_left: 'Cartellone sinistro', game_wall_right: 'Cartellone destro',
};

function placementLabel(row) {
  const label = placements[row.placement_id] || row.placement_id;
  return games[row.game_id] ? `${games[row.game_id]} · ${label}` : label;
}

async function load() {
  request?.abort();
  const controller = new AbortController();
  request = controller;
  loading.value = true;
  error.value = '';
  try {
    const { data } = await apiClient.get('/admin/sponsors/analytics', {
      ...props.authHeaders, signal: controller.signal,
      params: { from: selection.from, to: selection.to, sponsor_id: selection.sponsor, match_id: selection.match },
    });
    if (controller.signal.aborted) return;
    report.value = data;
    if (!selection.sponsor) knownSponsors.value = data.sponsors;
  } catch (err) {
    if (controller.signal.aborted) return;
    error.value = err.response?.status === 400 ? 'Scegli un intervallo valido di massimo 367 giorni.' : 'Statistiche sponsor non disponibili. Riprova.';
  } finally {
    if (!controller.signal.aborted) loading.value = false;
  }
}
onMounted(load);
onUnmounted(() => request?.abort());
</script>

<style scoped>
details { min-width: 0; }
summary { cursor: pointer; padding: .5rem 0; }
meter { vertical-align: middle; width: min(8rem, 35vw); }
</style>
