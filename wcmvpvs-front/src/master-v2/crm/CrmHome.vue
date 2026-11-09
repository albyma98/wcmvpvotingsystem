
<script setup>
import { computed, reactive, ref } from 'vue';
import {
  activeCheckIn, addDays, asc, companyName, crmPath, dateText, eur, nextChecklist, openOpportunity,
  opportunityTypes, phases, priorities, relationships, romeDay, taskTypes,
} from './model';

const props = defineProps({ data: { type: Object, required: true }, tab: String, busy: Boolean, follow: Function });
const emit = defineEmits(['edit', 'task', 'import']);
const filters = reactive({ search: '', sport: '', category: '', priority: '', owner: 0, archived: false, type: '', phase: '', noStep: false });
const pipelineView = ref('kanban');
const year = props.data.today.slice(0, 4);
const period = reactive({ from: year + '-01-01', to: year + '-12-31' });
const byCompany = computed(() => {
  const map = new Map();
  for (const record of props.data.records) {
    if (!map.has(record.companyId)) map.set(record.companyId, []);
    map.get(record.companyId).push(record);
  }
  return map;
});
const companyMap = computed(() => new Map(props.data.companies.map((company) => [company.id, company])));
const ownerName = (id) => props.data.operators.find((operator) => operator.id === id)?.name || 'Da assegnare';
const facets = (name) => [...new Set(props.data.companies.map((company) => company[name]).filter(Boolean))].sort(asc);
const matches = computed(() => {
  const search = filters.search.trim().toLocaleLowerCase('it');
  return props.data.companies.filter((company) => {
    const items = byCompany.value.get(company.id) || [];
    const names = [company.name, company.sponsoredName, ...items.filter((item) => item.kind === 'contact').flatMap((item) => [item.title, item.email])];
    return (filters.archived || !company.archived) &&
      (!search || names.some((name) => name?.toLocaleLowerCase('it').includes(search))) &&
      (!filters.sport || company.sport === filters.sport) &&
      (!filters.category || company.category === filters.category) &&
      (!filters.priority || company.priority === filters.priority) &&
      (!filters.owner || company.ownerId === filters.owner || items.some((item) => item.ownerId === filters.owner));
  });
});
const scope = computed(() => {
  const companies = new Set(matches.value.filter((item) => !item.archived).map((item) => item.id));
  return props.data.records.filter((item) => companies.has(item.companyId) && !item.archived);
});
const opportunities = computed(() => scope.value.filter((item) => item.kind === 'opportunity'));
const tasks = computed(() => scope.value.filter((item) => item.kind === 'task' && item.status === 'open').sort((a, b) => asc(a.due, b.due) || b.id - a.id));
const queue = computed(() => {
  const today = props.data.today;
  return [
    { label: 'Scaduti', rows: tasks.value.filter((item) => item.due < today) },
    { label: 'Oggi', rows: tasks.value.filter((item) => item.due === today) },
    { label: 'Prossimi 7 giorni', rows: tasks.value.filter((item) => item.due > today && item.due <= addDays(today, 7)) },
  ];
});
const pipeline = computed(() => opportunities.value.filter((item) =>
  (!filters.type || item.type === filters.type) && (!filters.phase || item.status === filters.phase) &&
  (!filters.noStep || (openOpportunity(item) && !item.nextStep?.trim()))));
const commercial = computed(() => {
  const open = opportunities.value.filter(openOpportunity);
  const today = props.data.today;
  const signed = scope.value.filter((item) => item.kind === 'contract' && item.status !== 'draft' && item.start <= period.to && item.end >= period.from);
  const dated = open.filter((item) => item.due && item.due >= period.from && item.due <= period.to);
  const sum = (items, key) => items.some((item) => item[key] != null) ? items.reduce((total, item) => total + (item[key] ?? 0), 0) : null;
  return {
    open: open.length, pilots: scope.value.filter((item) => item.kind === 'pilot' && item.status === 'active' && item.start <= today && item.end >= today).length,
    clients: matches.value.filter((item) => !item.archived && item.status === 'active').length,
    estimated: sum(dated, 'estimated'), signed: sum(signed, 'amount'),
    undated: open.filter((item) => !item.due).length,
  };
});

const unscheduled = computed(() => {
  const rows = opportunities.value.filter((item) => openOpportunity(item) && !item.nextStep?.trim() && !item.resumeOn)
    .map((item) => ({ id: 'opp-' + item.id, companyId: item.companyId, title: item.title, text: 'Opportunità senza prossimo passo', tab: 'opportunita', opportunity: item.id }));
  for (const company of matches.value.filter((item) => !item.archived)) {
    const items = byCompany.value.get(company.id) || [];
    if (company.status === 'onboarding' && nextChecklist(items)) {
      const last = items.filter((item) => item.kind === 'checklist' && item.status === 'done').map((item) => romeDay(item.completedAt)).sort().at(-1);
      if (!last || last <= addDays(props.data.today, -7)) rows.push({ id: 'onboarding-' + company.id, companyId: company.id, title: 'Onboarding da riprendere', text: 'Prossimo passo della checklist', tab: 'piloti' });
    }
    if (company.status === 'active' && !activeCheckIn(company, items)) rows.push({ id: 'checkin-' + company.id, companyId: company.id, title: 'Pianifica un check-in', text: 'Cliente senza prossimo confronto', tab: 'attivita' });
  }
  return rows;
});
const watchRows = computed(() => {
  const today = props.data.today;
  const horizon = addDays(today, 90);
  const rows = [];
  for (const item of scope.value) {
    const common = { id: item.id, title: item.title, companyId: item.companyId };
    if (item.kind === 'pilot' && item.status === 'active' && item.end <= addDays(today, 14)) {
      rows.push({ ...common, tab: 'piloti', text: 'Fine pilota', date: item.end });
    } else if (item.kind === 'issue' && item.status === 'open') {
      rows.push({ ...common, tab: 'andamento', text: 'Criticità · ' + priorities[item.priority], date: item.due });
    } else if (item.kind === 'contract' && item.status === 'signed' && (item.end <= horizon || item.due <= today)) {
      const renewal = opportunities.value.find((opportunity) => opportunity.contractId === item.id);
      if (renewal?.status === 'won') continue;
      const window = item.end < today ? 'Contratto scaduto' : item.end <= addDays(today, 30) ? 'Rinnovo entro 30 giorni'
        : item.end <= addDays(today, 60) ? 'Rinnovo entro 60 giorni' : item.end <= horizon ? 'Rinnovo entro 90 giorni' : 'Avvio rinnovo in scadenza';
      rows.push({ ...common, tab: 'contratti', text: window + ' · avvio ' + dateText(item.due), date: item.end });
    } else if (item.kind === 'opportunity' && openOpportunity(item) && item.resumeOn && item.resumeOn <= addDays(today, 7)) {
      rows.push({ ...common, tab: 'opportunita', opportunity: item.id, text: 'Ricontatto dopo la pausa', date: item.resumeOn });
    }
  }
  for (const company of matches.value.filter((item) => !item.archived && item.status === 'active')) {
    if (company.checkIn && company.checkIn <= addDays(today, 7)) rows.push({ id: 'check-' + company.id, title: 'Check-in programmato', companyId: company.id, tab: 'attivita', text: 'Confronto con il club', date: company.checkIn });
  }
  return rows.sort((a, b) => asc(a.date, b.date));
});

const companyLabel = (id) => companyName(companyMap.value.get(id));
</script>

<template>
  <div class="crm-stack">
    <div class="m-panel crm-stack">
      <div class="crm-actions">
        <button type="button" class="m-btn m-primary" :disabled="busy" @click="emit('edit', 'company', {})">Nuova società</button>
        <button type="button" class="m-btn" :disabled="busy" @click="emit('import')">Importa società e referenti</button>
      </div>
      <div class="crm-filters">
        <label class="m-field">Cerca società o referente<input v-model="filters.search" type="search" placeholder="Nome o email" /></label>
        <label class="m-field">Sport<select v-model="filters.sport"><option value="">Tutti</option><option v-for="item in facets('sport')" :key="item">{{ item }}</option></select></label>
        <label class="m-field">Campionato / categoria<select v-model="filters.category"><option value="">Tutti</option><option v-for="item in facets('category')" :key="item">{{ item }}</option></select></label>
        <label class="m-field">Priorità società<select v-model="filters.priority"><option value="">Tutte</option><option v-for="(label, key) in priorities" :key="key" :value="key">{{ label }}</option></select></label>
        <label class="m-field">Responsabile<select v-model="filters.owner"><option :value="0">Tutti</option><option v-for="operator in data.operators" :key="operator.id" :value="operator.id">{{ operator.name }}</option></select></label>
      </div>
    </div>

    <template v-if="tab === 'oggi'">
      <section class="m-panel crm-stack">
        <header class="crm-heading">
          <div><h2>Attività da fare</h2><p>Oggi {{ dateText(data.today) }} · Europe/Rome</p></div>
        </header>
        <div class="crm-queue">
          <div v-for="group in queue" :key="group.label" class="crm-stack crm-queue-column">
            <h3>{{ group.label }} <span class="m-badge m-muted">{{ group.rows.length }}</span></h3>
            <p v-if="!group.rows.length" class="m-muted-text">Nessuna attività.</p>
            <article v-for="item in group.rows" :key="item.id" class="crm-item">
              <a class="m-link" :href="item.opportunityId ? crmPath(item.companyId, 'opportunita', item.opportunityId) : crmPath(item.companyId, 'attivita')" @click="follow">{{ item.title }}</a>
              <p>{{ companyLabel(item.companyId) }} · {{ taskTypes[item.type] }} · {{ dateText(item.due) }}</p>
              <p>{{ ownerName(item.ownerId) }}<template v-if="item.priority"> · {{ priorities[item.priority] }}</template></p>
              <div class="crm-actions">
                <button class="m-btn crm-small" type="button" :disabled="busy" @click="emit('task', item)">Completa</button>
                <button class="m-btn crm-small" type="button" :disabled="busy" @click="emit('edit', 'task', item, 'reschedule')">Riprogramma</button>
                <button class="m-btn crm-small" type="button" :disabled="busy" @click="emit('edit', 'task', item, 'assign')">Assegna</button>
              </div>
            </article>
          </div>
        </div>
      </section>
      <div class="m-grid">
        <section v-for="group in [{ label: 'Da pianificare', rows: unscheduled }, { label: 'Da seguire', rows: watchRows }]" :key="group.label" class="m-panel crm-stack">
          <h2>{{ group.label }} <span class="m-badge m-muted">{{ group.rows.length }}</span></h2>
          <p v-if="!group.rows.length" class="m-muted-text">Nessuna voce nel filtro attuale.</p>
          <div v-for="row in group.rows" :key="row.id" class="crm-item">
            <a class="m-link" :href="crmPath(row.companyId, row.tab, row.opportunity)" @click="follow">{{ row.title }}</a>
            <p>{{ companyLabel(row.companyId) }} · {{ row.text }}<template v-if="row.date"> · {{ dateText(row.date) }}</template></p>
          </div>
        </section>
      </div>
      <section class="m-panel crm-stack">
        <header class="crm-heading"><div><h2>Riepilogo commerciale</h2><p>Conteggi al {{ dateText(data.today) }}. Importi nel periodo selezionato.</p></div></header>
        <div class="crm-actions">
          <label class="m-field">Dal<input v-model="period.from" type="date" /></label>
          <label class="m-field">Al<input v-model="period.to" type="date" /></label>
        </div>
        <p v-if="!period.from || !period.to || period.from > period.to" class="m-error">Seleziona un periodo valido.</p>
        <dl v-else class="crm-stats">
          <div><dt>Opportunità aperte</dt><dd>{{ commercial.open }}</dd></div>
          <div><dt>Piloti attivi</dt><dd>{{ commercial.pilots }}</dd></div>
          <div><dt>Clienti attivi</dt><dd>{{ commercial.clients }}</dd></div>
          <div><dt>Valore aperto · stima</dt><dd>{{ eur(commercial.estimated) }}</dd></div>
          <div><dt>Valore contrattualizzato · firmato</dt><dd>{{ eur(commercial.signed) }}</dd></div>
        </dl>
        <p class="m-muted-text">Stime per data di decisione prevista. Contratti per periodo di validità. Opportunità senza data di decisione: {{ commercial.undated }}.</p>
      </section>
    </template>

    <section v-else-if="tab === 'societa'" class="m-panel crm-stack">
      <div class="crm-heading"><h2>Società <span class="m-badge m-muted">{{ matches.length }}</span></h2><label class="crm-inline"><input v-model="filters.archived" type="checkbox" /> Includi schede archiviate</label></div>
      <p v-if="!matches.length" class="m-muted-text">Nessuna società nel filtro attuale. Crea una scheda o importa un CSV.</p>
      <div v-else class="m-table-scroll">
        <table class="m-table">
          <thead><tr><th>Società</th><th>Sport e categoria</th><th>Rapporto</th><th>Priorità</th><th>Responsabile</th><th>ABX</th></tr></thead>
          <tbody>
            <tr v-for="company in matches" :key="company.id">
              <td><a class="m-link" :href="crmPath(company.id)" @click="follow">{{ companyName(company) }}</a><div v-if="company.sponsoredName">{{ company.name }}</div></td>
              <td>{{ [company.sport, company.category].filter(Boolean).join(' · ') || '—' }}</td>
              <td>{{ relationships[company.status] }}<span v-if="company.archived" class="m-badge m-muted">Archiviata</span></td>
              <td>{{ priorities[company.priority] }}</td><td>{{ ownerName(company.ownerId) }}</td><td>{{ company.abxOrgId ? 'Collegata' : 'Da collegare' }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <section v-else class="m-panel crm-stack">
      <div class="crm-heading">
        <div><h2>Pipeline commerciale</h2><p>Apri una società per aggiungere una trattativa.</p></div>
        <div class="crm-actions">
          <button type="button" class="m-btn" :aria-pressed="pipelineView === 'kanban'" @click="pipelineView = 'kanban'">Kanban</button>
          <button type="button" class="m-btn" :aria-pressed="pipelineView === 'table'" @click="pipelineView = 'table'">Tabella</button>
        </div>
      </div>
      <div class="crm-filters">
        <label class="m-field">Tipo<select v-model="filters.type"><option value="">Tutti</option><option v-for="(label, key) in opportunityTypes" :key="key" :value="key">{{ label }}</option></select></label>
        <label class="m-field">Fase<select v-model="filters.phase"><option value="">Tutte</option><option v-for="(label, key) in phases" :key="key" :value="key">{{ label }}</option></select></label>
        <label class="crm-inline"><input v-model="filters.noStep" type="checkbox" /> Aperte senza prossimo passo</label>
      </div>
      <p v-if="!pipeline.length" class="m-muted-text">Nessuna opportunità nel filtro attuale.</p>
      <div v-else-if="pipelineView === 'kanban'" class="crm-kanban" tabindex="0" aria-label="Pipeline per fase commerciale">
        <div v-for="(label, phase) in phases" v-show="!filters.phase || filters.phase === phase" :key="phase" class="crm-queue-column crm-stack">
          <h3>{{ label }} <span class="m-badge m-muted">{{ pipeline.filter((item) => item.status === phase).length }}</span></h3>
          <article v-for="item in pipeline.filter((item) => item.status === phase)" :key="item.id" class="crm-item">
            <a class="m-link" :href="crmPath(item.companyId, 'opportunita', item.id)" @click="follow">{{ item.title }}</a>
            <p>{{ companyLabel(item.companyId) }} · {{ opportunityTypes[item.type] }}</p>
            <p v-if="item.resumeOn">Pausa fino al {{ dateText(item.resumeOn) }}</p>
            <p v-if="openOpportunity(item) && !item.nextStep?.trim()" class="m-badge m-warn">Senza prossimo passo</p>
            <button class="m-btn crm-small" type="button" :disabled="busy" @click="emit('edit', 'opportunity', item, 'phase')">Cambia fase</button>
          </article>
        </div>
      </div>
      <div v-else class="m-table-scroll">
        <table class="m-table">
          <thead><tr><th>Opportunità</th><th>Società</th><th>Tipo e fase</th><th>Stima</th><th>Decisione</th><th>Prossimo passo</th><th>Azioni</th></tr></thead>
          <tbody>
            <tr v-for="item in pipeline" :key="item.id">
              <td><a class="m-link" :href="crmPath(item.companyId, 'opportunita', item.id)" @click="follow">{{ item.title }}</a></td>
              <td>{{ companyLabel(item.companyId) }}</td>
              <td>{{ opportunityTypes[item.type] }} · {{ phases[item.status] }}<br v-if="item.resumeOn" /><small v-if="item.resumeOn">Pausa fino al {{ dateText(item.resumeOn) }}</small></td>
              <td>{{ eur(item.estimated) }}</td><td>{{ dateText(item.due) }}</td><td>{{ item.nextStep || '—' }}</td>
              <td><button class="m-btn crm-small" type="button" :disabled="busy" @click="emit('edit', 'opportunity', item, 'phase')">Cambia fase</button></td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </div>
</template>
