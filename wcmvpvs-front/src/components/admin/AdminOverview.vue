<template>
  <div class="overview">
    <header class="overview__heading">
      <div>
        <p class="overview__eyebrow">Il tuo centro di controllo</p>
        <h2>Ogni grande esperienza<br class="overview__desktop-break" /> parte da qui.</h2>
        <p>Eventi, squadra e fan. Tutto quello che serve per la tua società, in un unico posto.</p>
      </div>
      <button v-if="canOpen('events')" type="button" class="btn primary" @click="$emit('select', 'events')">
        <AdminIcon name="plus" :size="18" /> Gestisci eventi
      </button>
    </header>

    <div class="overview__kpis" aria-label="Riepilogo società">
      <button v-for="metric in metrics" :key="metric.id" type="button" class="overview__kpi" @click="$emit('select', metric.id)">
        <span class="overview__kpi-top"><span>{{ metric.label }}</span><AdminIcon :name="metric.icon" :size="19" /></span>
        <strong>{{ metric.value.toLocaleString('it-IT') }}</strong>
        <span class="overview__kpi-bottom"><span>{{ metric.detail }}</span><AdminIcon name="arrow-right" :size="16" /></span>
      </button>
    </div>

    <div class="overview__workspace">
      <section class="overview__panel overview__events">
        <header class="overview__panel-heading">
          <div><h3>Il calendario della tua arena</h3><p>Le partite attive e i prossimi appuntamenti.</p></div>
          <button v-if="canOpen('events')" class="overview__text-action" type="button" @click="$emit('select', 'events')">Tutti gli eventi <AdminIcon name="arrow-right" :size="16" /></button>
        </header>
        <div v-if="upcomingEvents.length" class="overview__event-list">
          <button v-for="event in upcomingEvents" :key="event.id" type="button" class="overview__event" @click="$emit('select', 'events')">
            <span class="overview__event-date"><strong>{{ datePart(event, 'day') }}</strong><span>{{ datePart(event, 'month') }}</span></span>
            <span class="overview__event-info"><strong>{{ getEventLabel(event) }}</strong><span>{{ getEventDate(event.start_datetime) }}<template v-if="event.location"> · {{ event.location }}</template></span></span>
            <span class="overview__status" :class="{ 'overview__status--live': event.is_active && !event.votes_closed }"><span v-if="event.is_active && !event.votes_closed" class="overview__live-dot" />{{ event.is_active ? (event.votes_closed ? 'Voto chiuso' : 'Live') : 'In programma' }}</span>
            <AdminIcon name="chevron-right" :size="17" />
          </button>
        </div>
        <div v-else class="overview__empty">
          <span class="overview__empty-icon"><AdminIcon name="calendar" :size="28" /></span>
          <h4>La prossima partita inizia qui</h4>
          <p>Crea un evento per preparare votazioni, premi ed esperienze per i tuoi fan.</p>
          <button v-if="canOpen('events')" type="button" class="btn outline" @click="$emit('select', 'events')">Vai agli eventi <AdminIcon name="arrow-right" :size="16" /></button>
        </div>
        <footer class="overview__event-footer"><AdminIcon name="calendar" :size="16" /><span>{{ events.length }} eventi totali · {{ concludedCount }} conclusi</span></footer>
      </section>

      <aside class="overview__spotlight">
        <span class="overview__spotlight-symbol" aria-hidden="true"><AdminIcon name="trophy" :size="38" /></span>
        <span class="overview__spotlight-label">MATCH DAY, SOTTO CONTROLLO</span>
        <template v-if="activeEvent">
          <span class="overview__spotlight-badge"><span class="overview__live-dot" />{{ activeEvent.votes_closed ? 'Votazioni chiuse' : 'Evento attivo' }}</span>
          <h3>{{ getEventLabel(activeEvent) }}</h3>
          <p>{{ activeEvent.votes_closed ? 'Consulta i risultati e gestisci la chiusura delle votazioni.' : 'Segui i risultati e accompagna i fan fino all’ultimo voto.' }}</p>
          <button v-if="canOpen('results')" type="button" @click="$emit('select', 'results')">Apri i risultati <AdminIcon name="arrow-right" :size="18" /></button>
        </template>
        <template v-else>
          <h3>Porta il tuo pubblico<br />al centro del gioco.</h3>
          <p>Prepara il prossimo evento e trasforma ogni partita in un’esperienza da ricordare.</p>
          <button v-if="canOpen('events')" type="button" @click="$emit('select', 'events')">Prepara la tua arena <AdminIcon name="arrow-right" :size="18" /></button>
        </template>
        <span class="overview__spotlight-bottom">ArenaBoostX · Fan engagement platform</span>
      </aside>
    </div>

    <section aria-labelledby="overview-quick-title">
      <header class="overview__quick-heading"><div><h3 id="overview-quick-title">Dai spazio alle tue idee</h3><p>Le attività di ogni giorno, a portata di clic.</p></div><span>Accesso rapido</span></header>
      <div class="overview__quick-grid">
        <button v-for="action in actions" :key="action.id" type="button" class="overview__quick" @click="$emit('select', action.id)">
          <span class="overview__quick-icon"><AdminIcon :name="action.id" :size="23" /></span>
          <strong>{{ action.label }}</strong><span>{{ action.description }}</span>
          <span class="overview__quick-link">Apri sezione <AdminIcon name="arrow-right" :size="16" /></span>
        </button>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed } from 'vue';
import AdminIcon from './ui/AdminIcon.vue';

const props = defineProps({
  actions: { type: Array, default: () => [] },
  allowedSections: { type: Array, default: () => [] },
  events: { type: Array, default: () => [] },
  teams: { type: Array, default: () => [] },
  players: { type: Array, default: () => [] },
  sponsors: { type: Array, default: () => [] },
  getEventLabel: { type: Function, required: true },
  getEventDate: { type: Function, required: true },
});
defineEmits(['select']);
const canOpen = id => props.allowedSections.some(section => section.id === id);
const activeEvent = computed(() => props.events.find(event => event.is_active && !event.is_concluded));
const concludedCount = computed(() => props.events.filter(event => event.is_concluded).length);
const upcomingEvents = computed(() => props.events.filter(event => !event.is_concluded &&
  (event.is_active || Date.parse(event.start_datetime) >= Date.now()),
).slice().sort((a, b) =>
  Number(Boolean(b.is_active)) - Number(Boolean(a.is_active)) ||
  (Date.parse(a.start_datetime) || 0) - (Date.parse(b.start_datetime) || 0),
).slice(0, 4));
const metrics = computed(() => [
  { id: 'events', label: 'Eventi', icon: 'calendar', value: props.events.length, detail: `${props.events.filter(event => event.is_active && !event.is_concluded).length} attivi in arena` },
  { id: 'teams', label: 'Squadre', icon: 'teams', value: props.teams.length, detail: 'Il tuo mondo sportivo' },
  { id: 'players', label: 'Giocatori', icon: 'users', value: props.players.length, detail: 'Nel roster della società' },
  { id: 'sponsors', label: 'Sponsor', icon: 'partners', value: props.sponsors.length, detail: 'Al fianco della tua società' },
].filter(metric => canOpen(metric.id)));
function datePart(event, part) {
  const date = new Date(event.start_datetime);
  return Number.isNaN(date.getTime()) ? '—' : date.toLocaleDateString('it-IT', part === 'day' ? { day: '2-digit' } : { month: 'short' });
}
</script>

<style scoped>
.overview { display: grid; gap: 28px; }
.overview__heading { display: flex; justify-content: space-between; align-items: center; gap: 24px; padding: 6px 0 2px; }
.overview__eyebrow { margin: 0 0 10px !important; color: var(--accent) !important; font-size: 12px !important; font-weight: 650; letter-spacing: .05em; }
.overview__heading h2 { margin: 0; color: var(--text-primary); font-size: clamp(26px, 2.7vw, 36px); line-height: 1.2; letter-spacing: -.035em; font-weight: 700; }
.overview__heading p { margin: 12px 0 0; color: var(--text-secondary); font-size: 13px; line-height: 1.7; max-width: 530px; }
.overview__heading > .btn { flex-shrink: 0; }
.overview__kpis { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 16px; }
.overview__kpi { display: flex; flex-direction: column; align-items: stretch; gap: 15px; padding: 21px; background: var(--bg-card); border: 1px solid var(--border); border-radius: 12px; text-align: left; cursor: pointer; box-shadow: 0 2px 3px #18223803; transition: border-color .18s, box-shadow .18s; min-width: 0; }
.overview__kpi:hover { border-color: #a5a0f4; box-shadow: 0 4px 16px #4f46e50a; }
.overview__kpi-top, .overview__kpi-bottom { display: flex; justify-content: space-between; align-items: center; gap: 8px; color: var(--text-secondary); font-size: 12px; }
.overview__kpi-top > svg { color: var(--accent); }
.overview__kpi > strong { font-size: 32px; font-weight: 650; color: var(--text-primary); line-height: 1; letter-spacing: -.04em; font-variant-numeric: tabular-nums; }
.overview__kpi-bottom { color: var(--text-muted); font-size: 11px; }
.overview__workspace { display: grid; grid-template-columns: minmax(0, 1.8fr) minmax(260px, 1fr); gap: 20px; }
.overview__panel { background: var(--bg-card); border: 1px solid var(--border); border-radius: 12px; overflow: hidden; min-width: 0; }
.overview__panel-heading { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 12px; padding: 22px; }
.overview h3 { font-weight: 650; font-size: 15px; letter-spacing: -.015em; color: var(--text-primary); margin: 0; }
.overview__panel-heading p, .overview__quick-heading p { margin: 6px 0 0; color: var(--text-muted); font-size: 12px; }
.overview__text-action { display: inline-flex; align-items: center; gap: 6px; color: var(--accent); font-size: 11px; font-weight: 600; min-height: 32px; }
.overview__event-list { padding: 0 22px; }
.overview__event { display: flex; align-items: center; gap: 14px; width: 100%; text-align: left; border-top: 1px solid var(--border-subtle); padding: 16px 0; cursor: pointer; color: var(--text-muted); }
.overview__event:hover .overview__event-info > strong { color: var(--accent); }
.overview__event-date { display: flex; flex-direction: column; align-items: center; gap: 2px; width: 44px; flex-shrink: 0; padding: 8px 4px; border: 1px solid var(--border); border-radius: 8px; background: var(--bg-card-hover); }
.overview__event-date > strong { color: var(--text-primary); font-size: 17px; line-height: 1; font-variant-numeric: tabular-nums; }
.overview__event-date > span { font-size: 10px; text-transform: uppercase; }
.overview__event-info { flex: 1; min-width: 0; display: grid; gap: 6px; }
.overview__event-info > strong { color: var(--text-primary); font-size: 12px; font-weight: 600; overflow-wrap: anywhere; }
.overview__event-info > span { font-size: 10px; line-height: 1.5; }
.overview__status { display: inline-flex; align-items: center; gap: 5px; background: #f0f2f7; color: #526078; padding: 4px 7px; border-radius: 5px; font-size: 10px; white-space: nowrap; }
.overview__status--live { background: #e9f8ef; color: #187544; }
.overview__live-dot { width: 6px; height: 6px; display: inline-block; border-radius: 50%; background: currentColor; flex-shrink: 0; }
.overview__event-footer { display: flex; gap: 8px; align-items: center; padding: 13px 22px; background: var(--bg-card-hover); border-top: 1px solid var(--border-subtle); font-size: 11px; color: var(--text-muted); }
.overview__empty { display: flex; flex-direction: column; align-items: center; justify-content: center; text-align: center; padding: 25px 24px 32px; }
.overview__empty-icon { display: grid; place-items: center; width: 54px; height: 54px; color: var(--accent); background: #f0efff; border-radius: 14px; margin-bottom: 15px; }
.overview__empty h4 { margin: 0; font-size: 14px; font-weight: 600; color: var(--text-primary); }
.overview__empty p { color: var(--text-muted); font-size: 12px; line-height: 1.7; max-width: 320px; margin: 9px 0 16px; }
.overview__spotlight { position: relative; display: flex; flex-direction: column; align-items: flex-start; overflow: hidden; border-radius: 12px; padding: 27px; background: radial-gradient(ellipse at 95% 0%, #34316a, transparent 65%), #171c34; color: #fff; min-width: 0; }
.overview__spotlight::after { content: ''; position: absolute; width: 220px; height: 220px; border: 1px solid #ffffff0d; border-radius: 50%; right: -90px; bottom: -140px; box-shadow: 0 0 0 36px #ffffff03, 0 0 0 72px #ffffff02; pointer-events: none; }
.overview__spotlight-symbol { display: grid; place-items: center; width: 65px; height: 65px; border: 1px solid #ffffff20; background: #ffffff09; border-radius: 18px; margin-bottom: 22px; color: #c7c4ff; }
.overview__spotlight-label { font-size: 9px; letter-spacing: .13em; color: #c3c7e1; font-weight: 600; }
.overview__spotlight h3 { margin: 14px 0 0; font-size: 23px; line-height: 1.35; color: #fff; letter-spacing: -.035em; }
.overview__spotlight p { margin: 13px 0 23px; color: #c3c7dc; font-size: 12px; line-height: 1.8; }
.overview__spotlight button { display: flex; justify-content: space-between; align-items: center; gap: 18px; min-height: 42px; padding: 0 14px; width: 100%; background: #fff; color: #272343; border-radius: 7px; font-weight: 600; font-size: 12px; margin-top: auto; }
.overview__spotlight button:hover { background: #eeecff; }
.overview__spotlight-badge { display: inline-flex; gap: 6px; align-items: center; color: #a3efc7; font-size: 11px; margin-top: 16px; }
.overview__spotlight-bottom { color: #bdc3d8; font-size: 9px; margin-top: 20px; }
.overview__quick-heading { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-bottom: 17px; }
.overview__quick-heading > span { color: var(--text-muted); font-size: 11px; }
.overview__quick-grid { display: grid; grid-template-columns: repeat(5, minmax(0, 1fr)); gap: 12px; }
.overview__quick { display: flex; flex-direction: column; align-items: flex-start; padding: 19px; text-align: left; background: var(--bg-card); border: 1px solid var(--border); border-radius: 11px; min-width: 0; transition: border-color .18s, box-shadow .18s; cursor: pointer; }
.overview__quick:hover { border-color: #a5a0f4; box-shadow: 0 4px 15px #4f46e509; }
.overview__quick-icon { display: grid; place-items: center; width: 40px; height: 40px; color: #5b50c9; background: #f1efff; border-radius: 10px; margin-bottom: 18px; }
.overview__quick:nth-child(2) .overview__quick-icon { background: #fff4e7; color: #9b601b; }
.overview__quick:nth-child(3) .overview__quick-icon { background: #e9f6f0; color: #207552; }
.overview__quick:nth-child(4) .overview__quick-icon { background: #eef2ff; color: #415db0; }
.overview__quick:nth-child(5) .overview__quick-icon { background: #fdf0f5; color: #a43e65; }
.overview__quick > strong { color: var(--text-primary); font-size: 13px; font-weight: 600; }
.overview__quick > span:not(.overview__quick-icon):not(.overview__quick-link) { margin-top: 7px; color: var(--text-muted); font-size: 11px; line-height: 1.7; }
.overview__quick-link { display: flex; gap: 6px; align-items: center; color: var(--accent); font-weight: 600; font-size: 10px; padding-top: 18px; margin-top: auto; }
@media (min-width: 1450px) { .overview__desktop-break { display: none; } }
@media (max-width: 1200px) { .overview__quick-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); } .overview__workspace { grid-template-columns: minmax(0, 1.5fr) minmax(240px, 1fr); } }
@media (max-width: 760px) { .overview { gap: 23px; } .overview__heading { align-items: flex-start; flex-direction: column; gap: 18px; } .overview__desktop-break { display: none; } .overview__kpis { grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; } .overview__workspace { grid-template-columns: minmax(0, 1fr); } .overview__quick-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } .overview__spotlight { padding: 24px; } .overview__spotlight-symbol { width: 48px; height: 48px; margin-bottom: 17px; } .overview__spotlight h3 br { display: none; } }
@media (max-width: 420px) { .overview__kpi { padding: 16px; } .overview__kpi-bottom { font-size: 10px; } .overview__panel-heading, .overview__event-footer { padding: 18px; } .overview__event-list { padding: 0 16px; } .overview__event { gap: 10px; flex-wrap: wrap; } .overview__event-info { flex-basis: calc(100% - 64px); } .overview__status { margin-left: 54px; } .overview__event > svg { margin-left: auto; } .overview__quick { padding: 16px; } .overview__quick-heading > span { display: none; } }
</style>
