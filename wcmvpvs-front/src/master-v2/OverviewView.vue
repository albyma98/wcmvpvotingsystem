<script setup>
import { computed, onMounted, ref } from 'vue';
import { masterApi } from './api';
import { masterSectionPath } from './routes';
import { useMasterWorkspace } from './workspace';
import MasterIcon from './ui/MasterIcon.vue';

const props = defineProps({ initialSummary: Object });
const emit = defineEmits(['summary-consumed']);
const { follow } = useMasterWorkspace();
const summary = ref(props.initialSummary ?? null);
const clubs = ref(null);
const tournaments = ref(null);
const loading = ref(true);
const failed = ref([]);
const number = new Intl.NumberFormat('it-IT');
const metric = (value) => typeof value === 'number' ? number.format(value) : '—';

const metrics = [
  { key: 'total_organizations', name: 'Società registrate', context: 'Anagrafiche di piattaforma', icon: 'building', tone: 'purple' },
  { key: 'total_votes', name: 'Voti complessivi', context: 'Storico di piattaforma', icon: 'results', tone: 'blue' },
  { key: 'votes_last_7_days', name: 'Voti negli ultimi 7 giorni', context: 'Attività recente', icon: 'activity', tone: 'green' },
  { key: 'total_events', name: 'Partite registrate', context: 'Eventi presenti nel sistema', icon: 'calendar', tone: 'amber' },
];
const directories = computed(() => [
  { section: 'clubs', name: 'ClubHub attivi', value: clubs.value?.filter((club) => club.active).length, icon: 'globe' },
  { section: 'tournaments', name: 'Tornei registrati', value: tournaments.value?.length, icon: 'lottery' },
]);
const actions = [
  { section: 'organizations', label: 'Registra una società', context: 'Anagrafica, identità visiva e servizi', icon: 'building', create: true },
  { section: 'clubs', label: 'Crea un ClubHub', context: 'Portale pubblico e accesso al CMS', icon: 'globe', create: true },
  { section: 'tournaments', label: 'Crea un torneo', context: 'Formato, sede e accesso operativo', icon: 'lottery', create: true },
  { section: 'arenas', label: 'Apri il catalogo arene', context: 'Scene 3D e configurazione', icon: 'box' },
];

function catalog(data, key) {
  const items = Array.isArray(data) ? data : data?.[key];
  return Array.isArray(items) ? items : [];
}

async function refresh(useInitial = false) {
  loading.value = true;
  failed.value = [];
  const calls = [
    { name: 'metriche', call: useInitial && summary.value ? Promise.resolve() : masterApi.get('/admin/master/summary').then(({ data }) => { summary.value = data; }) },
    { name: 'ClubHub', call: masterApi.get('/admin/master/clubs').then(({ data }) => { clubs.value = catalog(data, 'clubs'); }) },
    { name: 'tornei', call: masterApi.get('/admin/master/tournaments').then(({ data }) => { tournaments.value = catalog(data, 'tournaments'); }) },
  ];
  const results = await Promise.allSettled(calls.map(({ call }) => call));
  failed.value = calls.filter((_, index) => results[index].status === 'rejected').map(({ name }) => name);
  loading.value = false;
}

onMounted(() => {
  if (summary.value) emit('summary-consumed');
  refresh(true);
});
</script>

<template>
  <div class="m-overview m-stack" :aria-busy="loading">
    <section class="m-overview-banner" aria-labelledby="m-overview-label">
      <div>
        <span class="m-overview-kicker"><span></span> Workspace di piattaforma</span>
        <h2 id="m-overview-label">Controllo centrale</h2>
        <p>Una vista sulle attività ArenaBoostX, dalle votazioni ai portali dei club.</p>
      </div>
      <a class="m-btn" :href="masterSectionPath('analytics')" @click="follow">Esplora Analytics<MasterIcon name="arrow-right" :size="16" /></a>
    </section>

    <section class="m-overview-metrics" aria-labelledby="m-metrics-label">
      <div class="m-section-head">
        <div><h2 id="m-metrics-label">Attività di piattaforma</h2><p>Indicatori aggregati per tutte le società.</p></div>
        <button type="button" class="m-btn" :disabled="loading" @click="refresh()"><MasterIcon name="refresh" :size="16" />{{ loading ? 'Aggiornamento…' : 'Aggiorna dati' }}</button>
      </div>
      <p v-if="failed.length" class="m-error" role="alert">Aggiornamento non riuscito per {{ failed.join(', ') }}. Se presenti, sono visibili i valori precedenti.</p>
      <p v-if="loading" class="m-sr" role="status">Caricamento degli indicatori</p>
      <div class="m-metric-grid">
        <article v-for="item in metrics" :key="item.key" class="m-panel m-metric">
          <span class="m-metric-icon" :class="`m-tone-${item.tone}`"><MasterIcon :name="item.icon" :size="19" /></span>
          <h3>{{ item.name }}</h3>
          <strong>{{ metric(summary?.[item.key]) }}</strong>
          <p>{{ item.context }}</p>
        </article>
      </div>
    </section>

    <div class="m-overview-bottom">
      <section class="m-panel m-overview-list" aria-labelledby="m-actions-label">
        <div class="m-panel-head"><h2 id="m-actions-label">Azioni rapide</h2><p>Accedi direttamente alle operazioni più frequenti.</p></div>
        <a v-for="item in actions" :key="item.section" class="m-overview-row" :href="`${masterSectionPath(item.section)}${item.create ? '/create' : ''}`" @click="follow">
          <span class="m-row-icon"><MasterIcon :name="item.icon" /></span>
          <span><strong>{{ item.label }}</strong><small>{{ item.context }}</small></span>
          <MasterIcon class="m-row-arrow" name="chevron-right" :size="17" />
        </a>
      </section>

      <section class="m-panel m-overview-list" aria-labelledby="m-directories-label">
        <div class="m-panel-head"><h2 id="m-directories-label">Portali e tornei</h2><p>Accesso agli elenchi operativi.</p></div>
        <a v-for="item in directories" :key="item.section" class="m-overview-row" :href="masterSectionPath(item.section)" @click="follow">
          <span class="m-row-icon"><MasterIcon :name="item.icon" /></span>
          <strong>{{ item.name }}</strong>
          <strong class="m-row-count">{{ metric(item.value) }}</strong>
          <MasterIcon class="m-row-arrow" name="chevron-right" :size="17" />
        </a>
      </section>
    </div>
  </div>
</template>
