
import { crmBase } from '../routes.js';

export const crmPath = (id, tab = '', opportunity = 0) => [crmBase, id, tab, opportunity].filter(Boolean).join('/');
export const companyAPI = (id) => crmBase + '/companies' + (id ? '/' + id : '');
export const phases = {
  to_contact: 'Da contattare', contacted: 'Contattato', conversation: 'In conversazione',
  demo: 'Call / demo', pilot: 'Pilota', proposal: 'Proposta', negotiation: 'Negoziazione', won: 'Vinta', lost: 'Persa',
};
export const priorities = { low: 'Bassa', normal: 'Normale', high: 'Alta', urgent: 'Urgente' };
export const relationships = { lead: 'Prospect', pilot: 'Pilota', onboarding: 'Onboarding', active: 'Cliente attivo', former: 'Ex cliente' };
export const opportunityTypes = { new: 'Nuovo cliente', renewal: 'Rinnovo', expansion: 'Ampliamento' };
export const activityTypes = { email: 'Email', call: 'Telefonata', whatsapp: 'WhatsApp', meeting: 'Incontro', demo: 'Demo', note: 'Nota', pilot: 'Aggiornamento pilota', checkin: 'Check-in con il club' };
export const taskTypes = { followup: 'Follow-up', call: 'Call', demo: 'Demo', pilot: 'Avvio pilota', postmatch: 'Verifica post-partita', report: 'Consegna report', checkin: 'Check-in con il club', other: 'Altro' };
export const contactTypes = { decision: 'Decisore', operations: 'Referente operativo', introducer: 'Può introdurci', influencer: 'Influenza la decisione' };
export const healthLabels = { regular: 'Regolare', watch: 'Da seguire', risk: 'A rischio', insufficient: 'Dati insufficienti' };
export const checklist = { setup: 'Configurazione società', training: 'Formazione referente', materials: 'Verifica materiali di lancio', first_match: 'Prima gara attiva' };
export const pilotStates = { planned: 'Pianificato', active: 'Attivo', completed: 'Concluso' };
export const contractStates = { draft: 'Bozza', signed: 'Firmato', ended: 'Terminato' };
export const issueStates = { open: 'Aperta', resolved: 'Risolta' };
export const taskStates = { open: 'Da fare', done: 'Completato' };
export const metricLabels = {
  visitors: 'Visitatori digitali unici', new_profiles: 'Nuovi profili verificati', event_retention: 'Ritorno alla gara successiva',
  active_participation: 'Partecipazione attiva', game_participation: 'Partecipazione ai giochi',
  sponsor_reach: 'Copertura unica sponsor', sponsor_interactors: 'Partecipanti attivi con gli sponsor',
  sponsored_participation: 'Partecipazione ai giochi sponsorizzati',
};

const rome = { timeZone: 'Europe/Rome' };
const shortDate = new Intl.DateTimeFormat('it-IT', { ...rome, dateStyle: 'short' });
const dateTime = new Intl.DateTimeFormat('it-IT', { ...rome, dateStyle: 'short', timeStyle: 'short' });
const localClock = new Intl.DateTimeFormat('sv-SE', { ...rome, dateStyle: 'short', timeStyle: 'short' });
const currency = new Intl.NumberFormat('it-IT', { style: 'currency', currency: 'EUR' });
const decimal = new Intl.NumberFormat('it-IT', { maximumFractionDigits: 2 });
export const dateText = (value) => value ? shortDate.format(new Date(value.length === 10 ? value + 'T12:00:00Z' : value)) : '—';
export const dateTimeText = (value) => value ? dateTime.format(new Date(value)) : '—';
export const localTime = (value = new Date()) => localClock.format(new Date(value)).replace(' ', 'T');
export const romeDay = (value) => value?.length === 10 ? value : value ? localTime(value).slice(0, 10) : '';
export const eur = (value) => value == null ? 'ND' : currency.format(value / 100);
export const num = (value) => value == null ? 'ND' : decimal.format(value);
export const metricText = (value, unit) => value == null ? 'ND' : num(value) + (unit === 'percent' ? '%' : '');
export const openOpportunity = (record) => !record.archived && !['won', 'lost'].includes(record.status);
export const companyName = (company) => company?.sponsoredName || company?.name || 'Società';
export const errorText = (error) => error.response?.data?.message || (error.response?.status === 404 ? 'La scheda non è disponibile.' : 'Operazione non riuscita. Riprova.');
export const asc = (a, b) => a.localeCompare(b, 'it', { sensitivity: 'base' });

export function addDays(value, amount) {
  const date = new Date(value + 'T12:00:00Z');
  date.setUTCDate(date.getUTCDate() + amount);
  return date.toISOString().slice(0, 10);
}

export function nextChecklist(records) {
  const items = records.filter((record) => record.kind === 'checklist');
  return Object.keys(checklist).find((type) => !items.some((record) => record.type === type && record.status === 'done'));
}

export function contactTimeline(records) {
  return records.filter((record) => record.kind === 'activity' && !record.archived && record.type !== 'note' && record.type !== 'pilot')
    .sort((a, b) => Date.parse(b.date) - Date.parse(a.date) || b.id - a.id);
}

export function activeCheckIn(company, records) {
  const latest = romeDay(contactTimeline(records).find((record) => record.type === 'checkin')?.date);
  const planned = company.checkIn && company.checkIn > latest ? company.checkIn : '';
  return [planned, ...records.filter((record) => !record.archived && record.kind === 'task' && record.type === 'checkin' && record.status === 'open').map((record) => record.due)]
    .filter(Boolean).sort()[0];
}

export function comparableMetric(a, b) {
  return a?.value != null && b?.value != null && a.unit === b.unit && a.source === b.source &&
    a.description === b.description && a.numerator_key === b.numerator_key && a.denominator_key === b.denominator_key &&
    !a.source.startsWith('manual');
}

export function precedingMatch(matches) {
  const [current, ...previous] = matches;
  if (!current?.concluded) return null;
  return previous.find((match) => match.concluded && current.kpis.some((metric) => comparableMetric(metric, match.kpis.find((item) => item.id === metric.id)))) || null;
}

export function healthAssessment(company, records, trend, today) {
  const risks = [];
  const watch = [];
  const operating = ['active', 'pilot'].includes(company.status);
  const checkin = activeCheckIn(company, records);
  for (const issue of records.filter((record) => record.kind === 'issue' && !record.archived && record.status === 'open')) {
    (['high', 'urgent'].includes(issue.priority) ? risks : watch).push('Criticità aperta: ' + issue.title);
  }
  if (operating && checkin && checkin < today) watch.push('Check-in scaduto il ' + dateText(checkin));
  const latest = trend?.lastActivity;
  if (operating && company.cadenceDays && latest) {
    const nextFixtureDue = addDays(latest.day, company.cadenceDays);
    const missed = trend.events.find((event) => !event.activated && event.day >= nextFixtureDue && event.day < today);
    if (missed) watch.push('Nessuna attivazione rilevata per la gara del ' + dateText(missed.day) + ', oltre la cadenza attesa.');
  }
  if (operating && trend?.matches.length) {
    for (const goal of records.filter((item) => item.kind === 'goal' && item.metric && item.value != null && !item.archived && (!item.season || item.season === company.season))) {
      const matches = trend.matches.filter((match) => match.concluded && (!goal.start || match.day >= goal.start) && (!goal.end || match.day <= goal.end));
      const [current] = matches;
      const value = current?.kpis.find((item) => item.id === goal.metric);
      const earlier = matches.slice(1).find((match) => comparableMetric(value, match.kpis.find((item) => item.id === goal.metric)));
      if (earlier && value.value < goal.value && earlier.kpis.find((item) => item.id === goal.metric).value < goal.value) {
        watch.push(goal.title + ' sotto soglia nelle gare del ' + dateText(current.day) + ' e del ' + dateText(earlier.day));
      }
    }
  }
  if (risks.length) return { status: 'risk', reasons: [...risks, ...watch] };
  if (watch.length) return { status: 'watch', reasons: watch };
  if (!operating || !trend?.matches.some((match) => match.kpis.some((metric) => metric.value != null))) {
    return { status: 'insufficient', reasons: ['Fase del rapporto o misurazioni disponibili insufficienti per valutare l’andamento.'] };
  }
  return { status: 'regular', reasons: ['Le gare misurate e i check-in disponibili non mostrano i segnali di attenzione configurati.'] };
}

export const csvExample = [
  'codice_societa;societa;sport;campionato;stagione;citta;codice_referente;referente;ruolo;email;telefono;influenza;canale',
  'club-demo;Club Demo;Basket;Serie B;2026/27;Roma;demo-1;Giulia Rossi;Direzione;giulia@example.com;;decision;email',
  'club-demo;Club Demo;Basket;Serie B;2026/27;Roma;demo-2;Paolo Bianchi;Marketing;paolo@example.com;;operations;call',
].join('\n');
