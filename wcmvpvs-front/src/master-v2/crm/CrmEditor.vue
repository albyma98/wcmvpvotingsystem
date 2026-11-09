
<script setup>
import { computed, reactive, ref } from 'vue';
import MasterDialog from '../ui/MasterDialog.vue';
import {
  activityTypes, contactTypes, contractStates, healthLabels, issueStates, localTime, metricLabels,
  opportunityTypes, phases, pilotStates, priorities, relationships, taskStates, taskTypes,
} from './model';

const props = defineProps({
  spec: { type: Object, required: true }, company: Object, records: Array,
  operators: Array, organizations: Array, events: Array, today: String,
  busy: Boolean, error: String,
});
const emit = defineEmits(['save', 'close']);
const { kind, mode, initial } = props.spec;
const labels = {
  company: 'società', contact: 'referente', opportunity: 'opportunità', activity: 'contatto o nota', task: 'attività',
  pilot: 'pilota', issue: 'criticità', goal: 'obiettivo', metric: 'misurazione manuale', contract: 'contratto', payment: 'incasso',
};
const draft = reactive({ ...initial });
const wantFollowUp = ref(false);
const followUp = reactive({ title: '', due: props.today, ownerId: props.company?.ownerId || 0, priority: 'normal' });
for (const key of ['estimated', 'amount']) {
  if (draft[key] != null) draft[key] /= 100;
}
if (kind === 'activity') draft.date = draft.date ? localTime(draft.date) : localTime();
if (kind === 'pilot') draft.eventIds = [...(draft.eventIds || [])];

const options = (map) => Object.entries(map).map(([id, name]) => ({ id, name }));
const field = (key, label, type = 'text', required = false, extra = {}) => ({ key, label, type, required, ...extra });
const select = (key, label, values, required = false) => field(key, label, 'select', required, { values: Array.isArray(values) ? values : options(values) });
const text = (key, label, required = false) => field(key, label, 'text', required);
const textarea = (key, label, required = false) => field(key, label, 'textarea', required);
const date = (key, label, required = false) => field(key, label, 'date', required);
const money = (key, label, required = false) => field(key, label + ' (€)', 'number', required, { step: '0.01' });
const number = (key, label, step = '1') => field(key, label, 'number', false, { step });
const owner = (label = 'Responsabile interno') => select('ownerId', label, props.operators);
const contact = (label = 'Referente') => select('contactId', label, props.records.filter((item) => item.kind === 'contact' && (!item.archived || item.id === draft.contactId)).map((item) => ({ id: item.id, name: item.title + (item.archived ? ' · storico' : '') })));
const opportunity = () => select('opportunityId', 'Opportunità collegata', props.records.filter((item) => item.kind === 'opportunity' && (!item.archived || item.id === draft.opportunityId)).map((item) => ({ id: item.id, name: item.title })));
const archive = (label) => field('archived', label, 'checkbox');
const status = (map, label = 'Stato') => select('status', label, map, true);
const priority = (label = 'Priorità') => select('priority', label, priorities);
const eventList = computed(() => {
  const found = props.events.map((event) => ({ id: event.id, name: event.title + ' · ' + (event.day || 'data ND') }));
  const selected = [...(draft.eventIds || []), draft.eventId].filter(Boolean);
  for (const id of selected) if (!found.some((event) => event.id === id)) found.push({ id, name: 'Gara storica #' + id });
  return found;
});
const fields = computed(() => {
  if (mode === 'reschedule') return [date('due', 'Nuova scadenza', true)];
  if (mode === 'assign') return [owner('Assegnatario')];
  if (mode === 'phase') return [
    status(phases, 'Fase commerciale'),
    ...(draft.status === 'won' ? [money('amount', 'Valore concordato', true)] : []),
    ...(draft.status === 'lost' ? [textarea('reason', 'Motivo della perdita', true)] : []),
  ];
  if (mode === 'pause') return [date('resumeOn', 'Data di ricontatto'), textarea('reason', 'Motivo della pausa', Boolean(draft.resumeOn))];
  if (mode === 'link') return [select('abxOrgId', 'Società ABX collegata', props.organizations, true)];
  if (mode === 'health') return [select('health', 'Valutazione manuale', healthLabels), textarea('healthReason', 'Motivazione', Boolean(draft.health))];
  switch (kind) {
    case 'company':
      return [text('name', 'Nome società', true), text('sponsoredName', 'Nome sponsorizzato'), text('sport', 'Sport'),
        text('category', 'Campionato / categoria'), text('season', 'Stagione'), text('city', 'Città'),
        field('website', 'Sito web', 'url'), text('source', 'Fonte del contatto'), priority(), owner(),
        status(relationships, 'Stato del rapporto'), select('abxOrgId', 'Società ABX collegata', props.organizations),
        date('checkIn', 'Prossimo check-in'), number('cadenceDays', 'Cadenza attesa delle gare (giorni)'),
        select('health', 'Valutazione manuale', healthLabels), textarea('healthReason', 'Motivazione della valutazione manuale', Boolean(draft.health)),
        ...(draft.id ? [archive('Scheda archiviata')] : [])];
    case 'contact':
      return [text('title', 'Nome e cognome', true), text('role', 'Ruolo'),
        field('email', 'Email professionale', 'email'), field('phone', 'Telefono professionale', 'tel'),
        select('type', 'Influenza sulla decisione', contactTypes),
        select('channel', 'Canale preferito', { email: 'Email', call: 'Telefonata', whatsapp: 'WhatsApp', meeting: 'Incontro' }),
        text('source', 'Fonte'), date('date', 'Verificato il'), ...(draft.id ? [archive('Referente precedente')] : [])];
    case 'opportunity':
      return [text('title', 'Titolo', true), ...(draft.contractId ? [] : [select('type', 'Tipo', opportunityTypes, true)]),
        status(phases, 'Fase commerciale'), contact('Referente principale'), owner(), priority(), money('estimated', 'Valore stimato'),
        text('season', 'Stagione'), date('due', 'Decisione prevista'), textarea('goal', 'Obiettivo del club'),
        textarea('obstacle', 'Ostacolo principale'), textarea('nextStep', 'Prossimo passo'),
        ...(draft.status === 'won' ? [money('amount', 'Valore concordato', true)] : []),
        ...(draft.status === 'lost' ? [textarea('reason', 'Motivo della perdita', true)] : []),
        ...(draft.id ? [archive('Opportunità archiviata')] : [])];
    case 'activity':
      return [select('type', 'Tipo', activityTypes, true), text('title', 'Titolo', true),
        field('date', 'Data e ora (Europe/Rome)', 'datetime-local', true), contact(), opportunity(),
        textarea('notes', 'Dettagli'), textarea('result', 'Esito'), ...(draft.id ? [archive('Contatto o nota archiviati')] : [])];
    case 'task':
      return [text('title', 'Titolo', true), select('type', 'Tipo', taskTypes, true), date('due', 'Scadenza', true),
        status(taskStates), priority(), owner('Assegnatario'), contact(), opportunity(), textarea('notes', 'Dettagli'), ...(draft.id ? [archive('Attività archiviata')] : [])];
    case 'pilot':
      return [text('title', 'Titolo', true), status(pilotStates), contact('Referente operativo'), opportunity(), date('start', 'Inizio pilota', true),
        date('end', 'Fine pilota', true), number('planned', 'Gare previste'), number('played', 'Gare svolte'),
        date('due', 'Riunione di valutazione'), field('eventIds', 'Gare ABX collegate', 'events', false, { values: eventList.value }),
        textarea('goal', 'Obiettivi e KPI target'), textarea('notes', 'Risultati e feedback del club'),
        textarea('result', 'Esito del pilota', draft.status === 'completed'), textarea('nextStep', 'Passo commerciale successivo', draft.status === 'completed'),
        field('reportUrl', 'Link al report del pilota', 'url'), ...(draft.id ? [archive('Pilota archiviato')] : [])];
    case 'issue':
      return [text('title', 'Titolo', true), status(issueStates), select('priority', 'Gravità', priorities, true), owner(),
        date('due', 'Scadenza', true), textarea('notes', 'Descrizione', true), textarea('nextStep', 'Azione', true), textarea('result', 'Esito della risoluzione'),
        ...(draft.id ? [archive('Criticità archiviata')] : [])];
    case 'goal':
      return [text('title', 'Obiettivo concordato', true), text('season', 'Stagione'), date('start', 'Dal'), date('end', 'Al'),
        select('metric', 'KPI associato', metricLabels), { ...number('value', 'Soglia minima (0–100 per i KPI percentuali)', 'any'), required: Boolean(draft.metric) },
        textarea('result', 'Risultati raggiunti'), textarea('notes', 'Contesto e accordi con il club'),
        field('reportUrl', 'Link al report di gara o stagione', 'url'), ...(draft.id ? [archive('Obiettivo archiviato')] : [])];
    case 'metric':
      return [text('title', 'Nome della misurazione', true), select('metric', 'KPI ABX equivalente (se applicabile)', metricLabels),
        { ...number('value', 'Valore (0–100 per i KPI percentuali)', 'any'), required: true }, number('denominator', 'Denominatore, se disponibile', 'any'),
        text('role', 'Unità per una misurazione personalizzata'), select('eventId', 'Gara ABX', eventList.value),
        date('date', 'Data della misurazione', true), text('source', 'Fonte', true), textarea('notes', 'Note sulla misurazione'),
        field('reportUrl', 'Link alla fonte', 'url'), ...(draft.id ? [archive('Misurazione archiviata')] : [])];
    case 'contract':
      return [text('title', 'Titolo', true), text('season', 'Stagione', true), status(contractStates),
        contact('Referente per il rinnovo'), opportunity(), date('start', 'Inizio contratto', true),
        date('end', 'Fine contratto', true), date('due', 'Avvia il rinnovo entro il', draft.status !== 'draft'),
        money('estimated', 'Valore stimato'), ...(draft.status !== 'draft' ? [money('amount', 'Importo firmato', true)] : []),
        textarea('inKind', 'Componenti in natura'), textarea('paymentTerms', 'Termini di pagamento'),
        field('reportUrl', 'Link al report della stagione', 'url'), ...(draft.id ? [archive('Contratto archiviato')] : [])];
    case 'payment':
      return [text('title', 'Descrizione del versamento', true), date('date', 'Data di incasso', true),
        money('amount', 'Importo incassato', true), textarea('notes', 'Riferimento o note'), ...(draft.id ? [archive('Incasso archiviato')] : [])];
  }
  return [];
});

const title = mode === 'reschedule' ? 'Riprogramma attività'
  : mode === 'assign' ? 'Assegna attività'
  : mode === 'phase' ? 'Cambia fase commerciale'
  : mode === 'pause' ? 'Pausa e ricontatto'
  : mode === 'link' ? 'Collega società ABX'
  : mode === 'health' ? 'Valutazione manuale'
  : (draft.id ? 'Modifica' : 'Aggiungi') + ' ' + labels[kind];

function update(item, event) {
  if (item.type === 'number') {
    draft[item.key] = event.target.value === '' ? (item.key === 'cadenceDays' ? 0 : null) : event.target.valueAsNumber;
  } else if (item.type === 'checkbox') {
    draft[item.key] = event.target.checked;
  } else if (item.type === 'select') {
    const value = event.target.value;
    draft[item.key] = item.values.find((option) => String(option.id) === value)?.id ?? (item.key.endsWith('Id') ? 0 : '');
    if (item.key === 'status' && kind === 'opportunity' && value === 'lost' && initial.status !== 'lost') draft.reason = '';
  } else {
    draft[item.key] = event.target.value;
  }
}

function submit() {
  const payload = { ...draft };
  for (const key of ['estimated', 'amount']) {
    if (payload[key] != null) payload[key] = Math.round(payload[key] * 100);
  }
  if (kind === 'activity' && wantFollowUp.value) payload.followUp = { ...followUp };
  emit('save', payload);
}
</script>

<template>
  <MasterDialog :title="title" :description="mode === 'pause' ? 'Imposta una data di ricontatto e il motivo. Svuota la data per riprendere la trattativa.' : ''" :busy="busy" @close="emit('close')">
    <form class="crm-stack" @submit.prevent="submit">
      <div v-if="error" class="m-error" role="alert">{{ error }}</div>
      <fieldset class="crm-fieldset" :disabled="busy">
        <div class="m-fields">
          <template v-for="item in fields" :key="item.key">
            <fieldset v-if="item.type === 'events'" class="crm-event-options m-field-wide">
              <legend>{{ item.label }}</legend>
              <p v-if="!item.values.length" class="m-muted-text">Collega una società ABX con gare per selezionarle qui.</p>
              <label v-for="event in item.values" :key="event.id" class="crm-inline">
                <input v-model="draft.eventIds" type="checkbox" :value="event.id" /> {{ event.name }}
              </label>
            </fieldset>
            <label v-else :class="['m-field', { 'm-field-wide': ['textarea', 'checkbox'].includes(item.type) }]">
              <span v-if="item.type === 'checkbox'" class="crm-inline"><input type="checkbox" :checked="draft[item.key]" @change="update(item, $event)" />{{ item.label }}</span>
              <template v-else>
                {{ item.label }}
                <select v-if="item.type === 'select'" :value="draft[item.key] || ''" :required="item.required" @change="update(item, $event)">
                  <option value="">{{ item.required ? 'Seleziona…' : '—' }}</option>
                  <option v-for="option in item.values" :key="option.id" :value="option.id">{{ option.name }}</option>
                </select>
                <textarea v-else-if="item.type === 'textarea'" :value="draft[item.key] || ''" :required="item.required" rows="2" maxlength="4000" @input="update(item, $event)" />
                <input v-else :type="item.type" :value="draft[item.key] ?? ''" :required="item.required"
                  :min="item.type === 'number' ? 0 : undefined" :step="item.step" maxlength="300" @input="update(item, $event)" />
              </template>
            </label>
          </template>
        </div>
        <template v-if="kind === 'activity' && !draft.id">
          <label class="crm-inline crm-follow-up"><input v-model="wantFollowUp" type="checkbox" /> Pianifica un follow-up</label>
          <div v-if="wantFollowUp" class="m-fields">
            <label class="m-field">Titolo follow-up<input v-model.trim="followUp.title" required maxlength="300" /></label>
            <label class="m-field">Scadenza<input v-model="followUp.due" type="date" required /></label>
            <label class="m-field">Assegnatario<select v-model="followUp.ownerId"><option :value="0">Da assegnare</option><option v-for="operator in operators" :key="operator.id" :value="operator.id">{{ operator.name }}</option></select></label>
            <label class="m-field">Priorità<select v-model="followUp.priority"><option v-for="(label, key) in priorities" :key="key" :value="key">{{ label }}</option></select></label>
          </div>
        </template>
      </fieldset>
      <div class="crm-actions">
        <button class="m-btn m-primary" type="submit" :disabled="busy">{{ busy ? 'Salvataggio…' : 'Salva' }}</button>
        <button class="m-btn" type="button" :disabled="busy" @click="emit('close')">Annulla</button>
      </div>
    </form>
  </MasterDialog>
</template>
