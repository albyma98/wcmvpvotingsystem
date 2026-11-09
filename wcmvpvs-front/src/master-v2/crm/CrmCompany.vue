
<script setup>
import { computed, ref } from 'vue';
import { resolveApiUrl } from '../../api';
import CrmPerformance from './CrmPerformance.vue';
import {
  activeCheckIn, activityTypes, checklist, companyName, contactTimeline, contactTypes, contractStates, crmPath, dateText, dateTimeText,
  eur, issueStates, nextChecklist, num, openOpportunity, opportunityTypes, phases, pilotStates, priorities, relationships, taskTypes,
} from './model';

const props = defineProps({
  company: { type: Object, required: true }, records: { type: Array, required: true },
  route: Object, operators: Array, organizations: Array, today: String,
  trend: Object, trendError: String, trendLoading: Boolean, busy: Boolean, follow: Function,
});
const emit = defineEmits(['edit', 'task', 'checklist', 'renewal', 'trend']);
const tabs = {
  panoramica: 'Panoramica', referenti: 'Referenti', opportunita: 'Opportunità',
  attivita: 'Attività', piloti: 'Piloti e onboarding', andamento: 'Andamento', contratti: 'Contratti',
};
const canWrite = computed(() => !props.company.archived && !props.busy);
const showHistory = ref(false);
const grouped = computed(() => {
  const groups = {};
  for (const item of props.records) (groups[item.kind] ||= []).push(item);
  return groups;
});
const recordsOf = (kind) => grouped.value[kind] || [];
const visible = (kind) => recordsOf(kind).filter((item) => showHistory.value || !item.archived);
const contacts = computed(() => recordsOf('contact').filter((item) => !item.archived));
const byId = computed(() => new Map(props.records.map((item) => [item.id, item])));
const label = (id) => byId.value.get(id)?.title || '—';
const ownerName = (id) => props.operators.find((operator) => operator.id === id)?.name || 'Da assegnare';
const nextCheck = computed(() => activeCheckIn(props.company, props.records));
const orgName = computed(() => props.organizations.find((org) => org.id === props.company.abxOrgId)?.name || 'Da collegare');
const lastContact = computed(() => contactTimeline(props.records)[0]);
const activeTasks = computed(() => recordsOf('task').filter((item) => !item.archived && item.status === 'open').sort((a, b) => a.due.localeCompare(b.due)));
const nextOpportunity = computed(() => recordsOf('opportunity').filter((item) => openOpportunity(item) && item.nextStep && !item.resumeOn)
  .sort((a, b) => (a.due || '9999').localeCompare(b.due || '9999'))[0]);
const issues = computed(() => recordsOf('issue').filter((item) => !item.archived && item.status === 'open'));
const urgency = { urgent: 0, high: 1, normal: 2, low: 3 };
const topIssue = computed(() => [...issues.value].sort((a, b) => urgency[a.priority] - urgency[b.priority] || a.due.localeCompare(b.due))[0]);
const due = computed(() => [
  ...activeTasks.value.map((item) => item.due), ...issues.value.map((item) => item.due), nextCheck.value,
  ...recordsOf('opportunity').filter(openOpportunity).map((item) => item.resumeOn || item.due),
  ...recordsOf('pilot').filter((item) => !item.archived && ['planned', 'active'].includes(item.status)).flatMap((item) => [item.end, item.due]),
  ...recordsOf('contract').filter((item) => !item.archived && item.status === 'signed').flatMap((item) => [item.due, item.end]),
].filter(Boolean).sort()[0]);
const timeline = computed(() => [...visible('activity'), ...recordsOf('history')].sort((a, b) => Date.parse(b.date) - Date.parse(a.date) || b.id - a.id));
const selectedOpp = computed(() => recordsOf('opportunity').find((item) => item.id === props.route.opportunityId));
const selectedPipeline = computed(() => props.route.opportunityId ? (selectedOpp.value ? [selectedOpp.value] : []) : visible('opportunity'));
const board = computed(() => Object.entries(checklist).map(([type, title]) =>
  recordsOf('checklist').find((item) => item.type === type) || { type, title, status: 'open' }));
const onboardingNext = computed(() => checklist[nextChecklist(props.records)]);

function mailto(item) {
  const address = encodeURIComponent(item.email).replace('%40', '@');
  return 'mailto:' + address + '?subject=' + encodeURIComponent('ArenaBoostX · ' + companyName(props.company));
}
const findEvent = (id) => props.trend?.events.find((event) => event.id === id);
const payments = (id) => recordsOf('payment').filter((item) => item.contractId === id && (showHistory.value || !item.archived));
const paid = (id) => payments(id).filter((item) => !item.archived).reduce((total, item) => total + item.amount, 0);
const renewal = (id) => recordsOf('opportunity').find((item) => item.contractId === id && item.type === 'renewal');
const seasonEvents = (contract) => props.trend?.events.filter((event) => event.activated && event.day >= contract.start && event.day <= contract.end && event.day <= props.today) || [];
const seasonGoals = (contract) => recordsOf('goal').filter((item) => !item.archived && (!item.season || item.season === contract.season)
  && (!item.start || item.start <= contract.end) && (!item.end || item.end >= contract.start));
function quickActivity(type, title, contactId = 0) {
  emit('edit', 'activity', { type, title, contactId, opportunityId: selectedOpp.value?.id || 0 });
}
</script>

<template>
  <div class="crm-stack">
    <section class="m-panel crm-stack">
      <header class="crm-heading">
        <div>
          <div class="crm-actions">
            <span class="m-badge">{{ relationships[company.status] }}</span><span v-if="company.archived" class="m-badge m-muted">Archiviata</span>
            <span class="m-badge m-muted">{{ priorities[company.priority] }}</span>
          </div>
          <p>{{ [company.sponsoredName && company.name, company.sport, company.category, company.season, company.city].filter(Boolean).join(' · ') }}</p>
        </div>
        <button class="m-btn" type="button" :disabled="busy" @click="emit('edit', 'company', company)">Modifica scheda</button>
      </header>
      <dl class="crm-meta">
        <div><dt>Ultimo contatto</dt><dd>{{ lastContact ? dateTimeText(lastContact.date) + ' · ' + activityTypes[lastContact.type] : 'Nessuno registrato' }}</dd></div>
        <div><dt>Prossimo passo</dt><dd>{{ activeTasks[0]?.title || nextOpportunity?.nextStep || 'Da pianificare' }}</dd></div>
        <div><dt>Criticità</dt><dd>{{ topIssue?.title || 'Nessuna aperta' }}</dd></div>
        <div><dt>Scadenza più vicina</dt><dd>{{ dateText(due) }}</dd></div>
      </dl>
      <div class="crm-actions">
        <button class="m-btn crm-small" type="button" :disabled="!canWrite" @click="quickActivity('call', 'Telefonata')">Registra chiamata</button>
        <button class="m-btn crm-small" type="button" :disabled="!canWrite" @click="quickActivity('note', 'Nota')">Aggiungi nota</button>
        <button class="m-btn crm-small" type="button" :disabled="!canWrite" @click="emit('edit', 'task', { opportunityId: selectedOpp?.id || 0 })">Pianifica attività</button>
        <a v-if="contacts.find((item) => item.email)" class="m-btn crm-small" :href="mailto(contacts.find((item) => item.email))">Apri email</a>
      </div>
    </section>
    <nav class="crm-tabs" aria-label="Scheda società">
      <a v-for="(label, tab) in tabs" :key="tab" :href="crmPath(company.id, tab === 'panoramica' ? '' : tab)" :aria-current="route.tab === tab ? 'page' : undefined" class="m-btn crm-small" @click="follow">{{ label }}</a>
    </nav>

    <section v-if="route.tab === 'panoramica'" class="m-panel crm-stack">
      <h2>Anagrafica e rapporto</h2>
      <dl class="crm-meta">
        <div><dt>Società ABX collegata</dt><dd>{{ orgName }}</dd></div>
        <div><dt>Responsabile interno</dt><dd>{{ ownerName(company.ownerId) }}</dd></div>
        <div><dt>Fonte del contatto</dt><dd>{{ company.source || '—' }}</dd></div>
        <div><dt>Sito</dt><dd><a v-if="company.website" class="m-link" :href="company.website" target="_blank" rel="noopener noreferrer">{{ company.website }}</a><span v-else>—</span></dd></div>
        <div><dt>Prossimo check-in</dt><dd>{{ dateText(nextCheck) }}</dd></div>
        <div><dt>Cadenza delle gare</dt><dd>{{ company.cadenceDays ? company.cadenceDays + ' giorni' : '—' }}</dd></div>
        <div v-if="company.status === 'onboarding'"><dt>Prossimo passaggio onboarding</dt><dd>{{ onboardingNext || 'Checklist completata' }}</dd></div>
      </dl>
      <div class="crm-actions">
        <button v-if="!company.abxOrgId" class="m-btn m-primary" type="button" :disabled="!canWrite" @click="emit('edit', 'company', company, 'link')">Collega società ABX</button>
        <a class="m-btn" :href="crmPath(company.id, 'andamento')" @click="follow">Vedi andamento e criticità</a>
      </div>
    </section>

    <section v-else-if="route.tab === 'referenti'" class="m-panel crm-stack">
      <div class="crm-heading"><h2>Referenti</h2><button class="m-btn m-primary" type="button" :disabled="!canWrite" @click="emit('edit', 'contact', {})">Aggiungi referente</button></div>
      <p v-if="!recordsOf('contact').length" class="m-muted-text">Nessun referente registrato.</p>
      <div class="m-grid">
        <article v-for="item in recordsOf('contact')" :key="item.id" class="crm-item">
          <div class="crm-actions"><h3>{{ item.title }}</h3><span v-if="item.archived" class="m-badge m-muted">Referente precedente</span><span v-if="item.type" class="m-badge">{{ contactTypes[item.type] }}</span></div>
          <p>{{ item.role || 'Ruolo da verificare' }}</p>
          <p v-if="item.email || item.phone">{{ [item.email, item.phone].filter(Boolean).join(' · ') }}</p>
          <p>{{ item.channel ? activityTypes[item.channel] : 'Canale da definire' }} · Verificato {{ dateText(item.date) }}</p>
          <p v-if="item.source">Fonte {{ item.source }}</p>
          <div class="crm-actions">
            <button class="m-btn crm-small" type="button" :disabled="!canWrite" @click="emit('edit', 'contact', item)">Modifica</button>
            <button v-if="!item.archived" class="m-btn crm-small" type="button" :disabled="!canWrite" @click="quickActivity('call', 'Telefonata', item.id)">Registra chiamata</button>
            <a v-if="item.email && !item.archived" class="m-btn crm-small" :href="mailto(item)">Apri email</a>
          </div>
        </article>
      </div>
    </section>

    <section v-else-if="route.tab === 'opportunita'" class="m-panel crm-stack">
      <div class="crm-heading">
        <div><h2>{{ selectedOpp?.title || 'Opportunità' }}</h2><a v-if="route.opportunityId" class="m-link" :href="crmPath(company.id, 'opportunita')" @click="follow">Tutte le opportunità</a></div>
        <button class="m-btn m-primary" type="button" :disabled="!canWrite" @click="emit('edit', 'opportunity', {})">Nuova opportunità</button>
      </div>
      <label v-if="!route.opportunityId" class="crm-inline"><input v-model="showHistory" type="checkbox" /> Includi archiviate</label>
      <p v-if="!selectedPipeline.length" class="m-muted-text">{{ route.opportunityId ? 'Opportunità non trovata in questa scheda.' : 'Nessuna opportunità da mostrare.' }}</p>
      <article v-for="item in selectedPipeline" :key="item.id" class="crm-item">
        <h3 v-if="!route.opportunityId"><a class="m-link" :href="crmPath(company.id, 'opportunita', item.id)" @click="follow">{{ item.title }}</a></h3>
        <div class="crm-actions">
          <span class="m-badge">{{ phases[item.status] }}</span><span class="m-badge m-muted">{{ opportunityTypes[item.type] }}</span>
          <span v-if="item.archived" class="m-badge m-muted">Archiviata</span><span v-if="openOpportunity(item) && !item.nextStep?.trim()" class="m-badge m-warn">Senza prossimo passo</span>
        </div>
        <dl class="crm-meta">
          <div><dt>Referente principale</dt><dd>{{ label(item.contactId) }}</dd></div>
          <div><dt>Responsabile</dt><dd>{{ ownerName(item.ownerId) }}</dd></div>
          <div><dt>Stima</dt><dd>{{ eur(item.estimated) }}</dd></div>
          <div><dt>Stagione / decisione</dt><dd>{{ item.season || '—' }} · {{ dateText(item.due) }}</dd></div>
          <div><dt>Prossimo passo</dt><dd>{{ item.nextStep || '—' }}</dd></div>
          <div v-if="item.goal"><dt>Obiettivo del club</dt><dd>{{ item.goal }}</dd></div>
          <div v-if="item.obstacle"><dt>Ostacolo</dt><dd>{{ item.obstacle }}</dd></div>
          <div v-if="item.resumeOn"><dt>Pausa fino al {{ dateText(item.resumeOn) }}</dt><dd>{{ item.reason }}</dd></div>
          <div v-if="item.status === 'lost'"><dt>Motivo della perdita</dt><dd>{{ item.reason }}</dd></div>
          <div v-if="item.status === 'won'"><dt>Valore concordato</dt><dd>{{ eur(item.amount) }}</dd></div>
          <div v-if="item.contractId"><dt>Contratto di origine</dt><dd><a class="m-link" :href="crmPath(company.id, 'contratti')" @click="follow">{{ label(item.contractId) }}</a></dd></div>
        </dl>
        <div class="crm-actions">
          <button class="m-btn crm-small" type="button" :disabled="!canWrite" @click="emit('edit', 'opportunity', item)">Modifica</button>
          <button class="m-btn crm-small" type="button" :disabled="!canWrite || item.archived" @click="emit('edit', 'opportunity', item, 'phase')">Cambia fase</button>
          <button v-if="openOpportunity(item)" class="m-btn crm-small" type="button" :disabled="!canWrite" @click="emit('edit', 'opportunity', item, 'pause')">{{ item.resumeOn ? 'Gestisci pausa' : 'Metti in pausa' }}</button>
          <button v-if="!item.archived" class="m-btn crm-small" type="button" :disabled="!canWrite" @click="emit('edit', 'pilot', { title: 'Pilota · ' + item.title, opportunityId: item.id })">Apri pilota</button>
          <button v-if="item.status === 'won' && !item.archived" class="m-btn crm-small" type="button" :disabled="!canWrite" @click="emit('edit', 'contract', { title: item.title, opportunityId: item.id, contactId: item.contactId, season: item.season || company.season, amount: item.amount, estimated: item.estimated, status: 'signed' })">Registra contratto</button>
        </div>
        <details v-if="route.opportunityId" class="crm-details">
          <summary>Storico della trattativa</summary>
          <ol class="crm-timeline">
            <li v-for="entry in timeline.filter((event) => event.opportunityId === item.id)" :key="entry.id">
              <strong>{{ entry.title }}</strong><p>{{ dateTimeText(entry.date) }} · {{ entry.authorName }}</p><p class="crm-text">{{ entry.notes }}</p>
            </li>
          </ol>
        </details>
      </article>
    </section>

    <div v-else-if="route.tab === 'attivita'" class="crm-stack">
      <section class="m-panel crm-stack">
        <div class="crm-heading"><h2>Attività pianificate</h2><button class="m-btn m-primary" type="button" :disabled="!canWrite" @click="emit('edit', 'task', {})">Nuova attività</button></div>
        <label class="crm-inline"><input v-model="showHistory" type="checkbox" /> Includi voci archiviate</label>
        <p v-if="!visible('task').length" class="m-muted-text">Nessuna attività pianificata.</p>
        <div class="m-grid">
          <article v-for="item in visible('task')" :key="item.id" class="crm-item">
            <h3>{{ item.title }}</h3>
            <p>{{ taskTypes[item.type] }} · {{ dateText(item.due) }} · {{ ownerName(item.ownerId) }}<template v-if="item.priority"> · {{ priorities[item.priority] }}</template></p>
            <p v-if="item.notes" class="crm-text">{{ item.notes }}</p>
            <a v-if="item.opportunityId" class="m-link" :href="crmPath(company.id, 'opportunita', item.opportunityId)" @click="follow">{{ label(item.opportunityId) }}</a>
            <p v-if="item.archived" class="m-badge m-muted">Archiviata</p>
            <p v-else-if="item.completedAt" class="m-badge m-good">Completata {{ dateTimeText(item.completedAt) }}</p>
            <div class="crm-actions">
              <button class="m-btn crm-small" type="button" :disabled="!canWrite" @click="emit('edit', 'task', item)">Modifica</button>
              <button v-if="!item.archived" class="m-btn crm-small" type="button" :disabled="!canWrite" @click="emit('task', item)">{{ item.status === 'done' ? 'Riapri' : 'Completa' }}</button>
            </div>
          </article>
        </div>
      </section>
      <section class="m-panel crm-stack">
        <div class="crm-heading">
          <h2>Timeline</h2>
          <div class="crm-actions">
            <button class="m-btn m-primary" type="button" :disabled="!canWrite" @click="quickActivity('email', '')">Registra contatto</button>
            <button class="m-btn" type="button" :disabled="!canWrite" @click="quickActivity('checkin', 'Check-in con il club')">Registra check-in</button>
          </div>
        </div>
        <p v-if="!timeline.length" class="m-muted-text">Nessun contatto o aggiornamento registrato.</p>
        <ol v-else class="crm-timeline">
          <li v-for="item in timeline" :key="item.id">
            <h3>{{ item.title }}</h3>
            <p class="m-muted-text">{{ dateTimeText(item.date) }} · {{ item.authorName || 'Operatore master' }}<template v-if="item.kind === 'activity'"> · {{ activityTypes[item.type] }}</template></p>
            <p v-if="item.contactId">Referente {{ label(item.contactId) }}</p>
            <a v-if="item.opportunityId" class="m-link" :href="crmPath(company.id, 'opportunita', item.opportunityId)" @click="follow">{{ label(item.opportunityId) }}</a>
            <p v-if="item.notes" class="crm-text">{{ item.notes }}</p>
            <p v-if="item.result" class="crm-text">{{ item.result }}</p>
            <p v-if="item.archived" class="m-badge m-muted">Archiviata</p>
            <button v-if="item.kind === 'activity'" class="m-btn crm-small" type="button" :disabled="!canWrite" @click="emit('edit', 'activity', item)">Modifica</button>
          </li>
        </ol>
      </section>
    </div>

    <div v-else-if="route.tab === 'piloti'" class="crm-stack">
      <section class="m-panel crm-stack">
        <div class="crm-heading"><h2>Piloti</h2><button class="m-btn m-primary" type="button" :disabled="!canWrite" @click="emit('edit', 'pilot', {})">Nuovo pilota</button></div>
        <label class="crm-inline"><input v-model="showHistory" type="checkbox" /> Includi piloti archiviati</label>
        <p v-if="!visible('pilot').length" class="m-muted-text">Nessun pilota registrato.</p>
        <article v-for="item in visible('pilot')" :key="item.id" class="crm-item">
          <div class="crm-actions"><h3>{{ item.title }}</h3><span class="m-badge">{{ pilotStates[item.status] }}</span><span v-if="item.archived" class="m-badge m-muted">Archiviato</span></div>
          <dl class="crm-meta">
            <div><dt>Periodo</dt><dd>{{ dateText(item.start) }} – {{ dateText(item.end) }}</dd></div>
            <div><dt>Gare svolte / previste</dt><dd>{{ num(item.played) }} / {{ num(item.planned) }}</dd></div>
            <div><dt>Referente operativo</dt><dd>{{ label(item.contactId) }}</dd></div>
            <div><dt>Riunione di valutazione</dt><dd>{{ dateText(item.due) }}</dd></div>
            <div v-if="item.goal"><dt>Obiettivi e KPI target</dt><dd>{{ item.goal }}</dd></div>
            <div v-if="item.notes"><dt>Risultati e feedback</dt><dd>{{ item.notes }}</dd></div>
            <div v-if="item.result"><dt>Esito del pilota</dt><dd>{{ item.result }}</dd></div>
            <div v-if="item.nextStep"><dt>Passo commerciale successivo</dt><dd>{{ item.nextStep }}</dd></div>
          </dl>
          <p v-for="id in item.eventIds || []" :key="id">
            {{ findEvent(id)?.title || 'Gara #' + id }}<template v-if="findEvent(id)?.day"> · {{ dateText(findEvent(id).day) }}</template>
            <a v-if="findEvent(id)?.reportPath" class="m-link" :href="resolveApiUrl(findEvent(id).reportPath)" target="_blank" rel="noopener noreferrer"> · Report gara</a>
          </p>
          <div class="crm-actions">
            <a v-if="item.opportunityId" class="m-link" :href="crmPath(company.id, 'opportunita', item.opportunityId)" @click="follow">{{ label(item.opportunityId) }}</a>
            <a v-if="item.reportUrl" class="m-link" :href="item.reportUrl" target="_blank" rel="noopener noreferrer">Report pilota</a>
            <button class="m-btn crm-small" type="button" :disabled="!canWrite" @click="emit('edit', 'pilot', item)">Modifica pilota</button>
            <button class="m-btn crm-small" type="button" :disabled="!canWrite" @click="emit('edit', 'activity', { type: 'pilot', title: 'Aggiornamento · ' + item.title, opportunityId: item.opportunityId, contactId: item.contactId })">Registra aggiornamento</button>
          </div>
        </article>
        <p v-if="trendError" class="m-error" role="alert">{{ trendError }} <button class="m-btn crm-small" type="button" @click="emit('trend')">Ricarica gare ABX</button></p>
      </section>
      <section class="m-panel crm-stack">
        <div class="crm-heading"><div><h2>Checklist onboarding</h2><p>{{ onboardingNext ? 'Prossimo passo · ' + onboardingNext : 'Tutti i passaggi completati.' }}</p></div></div>
        <div v-for="item in board" :key="item.type" class="crm-item">
          <label class="crm-inline">
            <input type="checkbox" :checked="item.status === 'done'" :disabled="!canWrite" @change="$event.target.checked = item.status === 'done'; emit('checklist', item)" />
            {{ item.title }} <span v-if="item.completedAt" class="m-badge m-good">{{ dateTimeText(item.completedAt) }}</span>
          </label>
        </div>
      </section>
    </div>

    <CrmPerformance v-else-if="route.tab === 'andamento'" :company="company" :records="records" :today="today" :trend="trend" :trend-error="trendError"
      :trend-loading="trendLoading" :busy="!canWrite" :owner-name="ownerName" @edit="(...args) => emit('edit', ...args)" @trend="(params) => emit('trend', params)" />

    <section v-else class="m-panel crm-stack">
      <div class="crm-heading"><h2>Contratti e rinnovi</h2><button class="m-btn m-primary" type="button" :disabled="!canWrite" @click="emit('edit', 'contract', {})">Nuovo contratto</button></div>
      <label class="crm-inline"><input v-model="showHistory" type="checkbox" /> Includi voci archiviate</label>
      <p v-if="!visible('contract').length" class="m-muted-text">Nessun contratto registrato.</p>
      <article v-for="item in visible('contract')" :key="item.id" class="crm-item">
        <div class="crm-actions"><h3>{{ item.title }} · {{ item.season }}</h3><span class="m-badge">{{ contractStates[item.status] }}</span><span v-if="item.archived" class="m-badge m-muted">Archiviato</span></div>
        <dl class="crm-meta">
          <div><dt>Periodo</dt><dd>{{ dateText(item.start) }} – {{ dateText(item.end) }}</dd></div>
          <div><dt>Avvio rinnovo</dt><dd>{{ dateText(item.due) }}</dd></div>
          <div><dt>Referente</dt><dd>{{ label(item.contactId) }}</dd></div>
          <div><dt>Stima</dt><dd>{{ eur(item.estimated) }}</dd></div>
          <div><dt>Importo firmato</dt><dd>{{ eur(item.amount) }}</dd></div>
          <div><dt>Incassato</dt><dd>{{ item.status === 'draft' ? '—' : eur(paid(item.id)) }}<template v-if="item.amount != null"> · {{ paid(item.id) >= item.amount ? 'Saldato' : paid(item.id) ? 'Parziale' : 'Da incassare' }}</template></dd></div>
          <div v-if="item.inKind"><dt>Componenti in natura</dt><dd>{{ item.inKind }}</dd></div>
          <div v-if="item.paymentTerms"><dt>Termini di pagamento</dt><dd>{{ item.paymentTerms }}</dd></div>
        </dl>
        <div class="crm-actions">
          <button class="m-btn crm-small" type="button" :disabled="!canWrite" @click="emit('edit', 'contract', item)">Modifica contratto</button>
          <button v-if="item.amount > paid(item.id) && !item.archived" class="m-btn crm-small" type="button" :disabled="!canWrite" @click="emit('edit', 'payment', { contractId: item.id, title: 'Versamento' })">Registra incasso</button>
          <a v-if="renewal(item.id)" class="m-btn crm-small" :href="crmPath(company.id, 'opportunita', renewal(item.id).id)" @click="follow">Apri rinnovo</a>
          <button v-else-if="item.status !== 'draft' && !item.archived" class="m-btn m-primary crm-small" type="button" :disabled="!canWrite" @click="emit('renewal', item)">Crea opportunità di rinnovo</button>
        </div>
        <details v-if="payments(item.id).length" class="crm-details">
          <summary>Movimenti di incasso</summary>
          <div class="crm-stack">
            <div v-for="entry in payments(item.id)" :key="entry.id" class="crm-actions">
              {{ dateText(entry.date) }} · {{ entry.title }} · {{ eur(entry.amount) }}<span v-if="entry.archived" class="m-badge m-muted">Archiviato</span>
              <button class="m-btn crm-small" type="button" :disabled="!canWrite" @click="emit('edit', 'payment', entry)">Modifica incasso</button>
            </div>
          </div>
        </details>
        <details class="crm-details">
          <summary>Riepilogo per il rinnovo</summary>
          <div class="crm-stack">
            <p v-if="trendLoading" role="status">Caricamento gare…</p>
            <p v-else-if="trendError" class="m-error">{{ trendError }} <button type="button" class="m-btn crm-small" @click="emit('trend')">Riprova</button></p>
            <p v-else-if="trend && company.abxOrgId">{{ seasonEvents(item).length }} gare attivate nel periodo del contratto.</p>
            <p v-else>Gare ABX ND per questa scheda.</p>
            <div class="crm-actions">
              <a v-if="item.reportUrl" class="m-link" :href="item.reportUrl" target="_blank" rel="noopener noreferrer">Report stagione</a>
              <a class="m-link" :href="crmPath(company.id, 'andamento')" @click="follow">KPI e andamento</a>
            </div>
            <ul v-if="seasonEvents(item).length">
              <li v-for="event in seasonEvents(item)" :key="event.id">
                {{ dateText(event.day) }} · {{ event.title }}
                <a v-if="event.reportPath" class="m-link" :href="resolveApiUrl(event.reportPath)" target="_blank" rel="noopener noreferrer">Report gara</a>
              </li>
            </ul>
            <h3>Obiettivi e risultati</h3>
            <p v-if="!seasonGoals(item).length" class="m-muted-text">Nessun obiettivo della stagione registrato.</p>
            <p v-for="goal in seasonGoals(item)" :key="goal.id">{{ goal.title }} · {{ goal.result || 'Risultato ND' }}
              <a v-if="goal.reportUrl" class="m-link" :href="goal.reportUrl" target="_blank" rel="noopener noreferrer">Report obiettivo</a>
            </p>
            <h3>Criticità da discutere</h3>
            <p v-if="!issues.length" class="m-muted-text">Nessuna criticità aperta.</p>
            <p v-for="issue in issues" :key="issue.id">{{ issue.title }} · {{ issueStates[issue.status] }} · {{ issue.nextStep }} entro {{ dateText(issue.due) }}</p>
          </div>
        </details>
      </article>
    </section>
  </div>
</template>
