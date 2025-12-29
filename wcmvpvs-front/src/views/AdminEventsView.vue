<template>
  <AppShell>
    <section class="page">
      <div class="page__header">
        <div>
          <p class="eyebrow">Setup evento</p>
          <h1>Eventi</h1>
          <p class="muted">Crea, avvia e monitora le partite con un layout full width pensato per l'uso pro.</p>
        </div>
        <div class="pill-group">
          <span class="pill success">Attivo</span>
          <span class="pill ghost">Live sync</span>
          <button class="primary ghost" type="button" @click="openPreview">Anteprima fan</button>
        </div>
      </div>

      <div class="grid">
        <div class="col col-8">
          <div class="card">
            <div class="card__header">
              <div>
                <p class="eyebrow">Nuovo evento</p>
                <h3>Impostazioni di gara</h3>
              </div>
              <div class="kpi-badge">
                <span class="dot online"></span>
                Modalità attiva
              </div>
            </div>
            <form class="form-grid" @submit.prevent="saveEvent">
              <label>
                Squadra di casa
                <input v-model="eventForm.homeTeam" type="text" placeholder="Es. Power Volley Milano" required />
              </label>
              <label>
                Squadra ospite
                <input v-model="eventForm.awayTeam" type="text" placeholder="Es. Sir Safety Perugia" required />
              </label>
              <label>
                Data e ora
                <input v-model="eventForm.date" type="datetime-local" required />
              </label>
              <label>
                Location
                <input v-model="eventForm.location" type="text" placeholder="Arena o palazzetto" />
              </label>
              <label class="full-row">
                Sponsor pre-voto
                <textarea
                  v-model="eventForm.preVote"
                  rows="2"
                  placeholder="Definisci sponsor e contenuti mostrati prima della votazione"
                ></textarea>
              </label>
              <label class="full-row">
                Esperienza post-voto
                <textarea
                  v-model="eventForm.postVote"
                  rows="2"
                  placeholder="Trend voti, selfie MVP, reaction test..."
                ></textarea>
              </label>
              <div class="form-actions">
                <label class="toggle">
                  <input v-model="eventForm.showCounter" type="checkbox" />
                  <span>Mostra contatore voti live</span>
                </label>
                <label class="toggle">
                  <input v-model="eventForm.allowSelfie" type="checkbox" />
                  <span>Abilita selfie MVP</span>
                </label>
                <label class="toggle">
                  <input v-model="eventForm.allowSurvey" type="checkbox" />
                  <span>Abilita survey feedback</span>
                </label>
              </div>
              <div class="actions-row">
                <button class="ghost" type="button" @click="resetForm">Reset</button>
                <button class="primary" type="submit">Salva impostazioni</button>
              </div>
            </form>
          </div>
        </div>
        <div class="col col-4 kpi-column">
          <div class="card stack">
            <div class="stack__item">
              <p class="muted">Voti raccolti</p>
              <h2>{{ kpis.votes.toLocaleString() }}</h2>
              <p class="delta positive">+8% vs ultima gara</p>
            </div>
            <div class="stack__item">
              <p class="muted">Utenti unici</p>
              <h2>{{ kpis.users.toLocaleString() }}</h2>
              <p class="delta">+3% sessioni</p>
            </div>
            <div class="stack__item">
              <p class="muted">Click sponsor</p>
              <h2>{{ kpis.sponsorClicks.toLocaleString() }}</h2>
              <p class="delta positive">+12% engagement</p>
            </div>
          </div>
          <div class="card actions">
            <p class="eyebrow">Azioni rapide</p>
            <div class="action-buttons">
              <button class="primary" type="button" @click="setLive">Avvia evento</button>
              <button class="ghost" type="button" @click="closeEvent">Chiudi evento</button>
              <button class="ghost" type="button" @click="openPreview">Anteprima</button>
            </div>
            <div class="status">
              <span class="pill success">Live</span>
              <p class="muted">Evento sincronizzato con scoreboard e flussi fan.</p>
            </div>
          </div>
        </div>
      </div>

      <div class="card">
        <div class="card__header">
          <div>
            <p class="eyebrow">Roadmap eventi</p>
            <h3>Calendario e stato</h3>
          </div>
          <div class="filters">
            <button class="ghost" type="button" @click="filterStatus = 'all'">Tutti</button>
            <button class="ghost" type="button" @click="filterStatus = 'live'">Live</button>
            <button class="ghost" type="button" @click="filterStatus = 'closed'">Chiusi</button>
          </div>
        </div>
        <div class="table">
          <div class="table__head">
            <span>Evento</span>
            <span>Data</span>
            <span>Location</span>
            <span>Stato</span>
            <span>Azioni</span>
          </div>
          <div v-for="event in filteredEvents" :key="event.id" class="table__row">
            <div>
              <p class="strong">{{ event.name }}</p>
              <p class="muted">{{ event.teams }}</p>
            </div>
            <div>
              <p class="muted">{{ event.date }}</p>
              <p class="strong">{{ event.time }}</p>
            </div>
            <div class="muted">{{ event.location }}</div>
            <div>
              <span class="pill" :class="event.status">{{ statusLabel(event.status) }}</span>
            </div>
            <div class="row-actions">
              <button class="ghost" type="button">Dettagli</button>
              <button class="primary" type="button">Apri live</button>
            </div>
          </div>
        </div>
      </div>
    </section>
  </AppShell>
</template>

<script setup>
import { computed, reactive, ref } from 'vue';
import AppShell from '../components/layout/AppShell.vue';

const eventForm = reactive({
  homeTeam: 'Power Volley',
  awayTeam: 'Sir Safety',
  date: '',
  location: 'Allianz Cloud, Milano',
  preVote: 'Sponsor ledwall, countdown e CTA di engagement.',
  postVote: 'Trend voti, selfie MVP e reaction test.',
  showCounter: true,
  allowSelfie: true,
  allowSurvey: true,
});

const kpis = reactive({
  votes: 18240,
  users: 5230,
  sponsorClicks: 1240,
});

const events = ref([
  { id: 1, name: 'Playoff - Gara 3', teams: 'Power Volley vs Sir Safety', date: '12 Mar 2025', time: '20:30', location: 'Allianz Cloud', status: 'live' },
  { id: 2, name: 'Regular Season', teams: 'Milano vs Padova', date: '05 Mar 2025', time: '18:00', location: 'Allianz Cloud', status: 'scheduled' },
  { id: 3, name: 'Coppa Italia', teams: 'Milano vs Modena', date: '22 Feb 2025', time: '20:30', location: 'PalaPanini', status: 'closed' },
]);

const filterStatus = ref('all');

const filteredEvents = computed(() => {
  if (filterStatus.value === 'all') return events.value;
  return events.value.filter((event) => event.status === filterStatus.value);
});

function statusLabel(status) {
  if (status === 'live') return 'Live';
  if (status === 'closed') return 'Chiuso';
  return 'Programmato';
}

function resetForm() {
  eventForm.homeTeam = '';
  eventForm.awayTeam = '';
  eventForm.date = '';
  eventForm.location = '';
  eventForm.preVote = '';
  eventForm.postVote = '';
  eventForm.showCounter = true;
  eventForm.allowSelfie = true;
  eventForm.allowSurvey = true;
}

function saveEvent() {
  const nextId = events.value.length + 1;
  events.value.unshift({
    id: nextId,
    name: `${eventForm.homeTeam} vs ${eventForm.awayTeam}`,
    teams: `${eventForm.homeTeam} vs ${eventForm.awayTeam}`,
    date: new Date(eventForm.date).toLocaleDateString('it-IT'),
    time: new Date(eventForm.date).toLocaleTimeString('it-IT', { hour: '2-digit', minute: '2-digit' }),
    location: eventForm.location || 'Da definire',
    status: 'scheduled',
  });
  resetForm();
}

function openPreview() {
  console.info('Apro anteprima fan');
}

function setLive() {
  filterStatus.value = 'live';
}

function closeEvent() {
  filterStatus.value = 'closed';
}
</script>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.page__header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem;
}

.eyebrow {
  text-transform: uppercase;
  letter-spacing: 0.12em;
  font-weight: 800;
  color: var(--text-muted);
  margin: 0;
}

h1 {
  margin: 0.2rem 0;
  font-size: 1.75rem;
}

.muted {
  color: var(--text-muted);
  margin: 0;
}

.pill-group {
  display: flex;
  gap: 0.5rem;
  align-items: center;
}

.grid {
  display: grid;
  grid-template-columns: repeat(12, 1fr);
  gap: 1rem;
}

.col {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.col-8 {
  grid-column: span 8;
}

.col-4 {
  grid-column: span 4;
}

.card {
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid var(--border-strong);
  border-radius: 1rem;
  padding: 1rem;
  box-shadow: 0 12px 30px rgba(0, 0, 0, 0.25);
}

.card__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 0.5rem;
}

.kpi-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.45rem 0.7rem;
  border-radius: 999px;
  background: rgba(34, 197, 94, 0.12);
  color: #c0f8d6;
  font-weight: 700;
}

.dot {
  width: 9px;
  height: 9px;
  border-radius: 50%;
  display: inline-flex;
}

.dot.online {
  background: #22c55e;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.85rem 1rem;
  margin-top: 1rem;
}

label {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  color: var(--text-primary);
  font-weight: 700;
}

input,
textarea {
  width: 100%;
  padding: 0.85rem 0.9rem;
  border-radius: 0.75rem;
  border: 1px solid var(--border-strong);
  background: rgba(255, 255, 255, 0.03);
  color: var(--text-primary);
}

input:focus,
textarea:focus {
  outline: 2px solid rgba(96, 165, 250, 0.4);
}

.full-row {
  grid-column: span 2;
}

.form-actions {
  grid-column: span 2;
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 0.75rem;
  align-items: center;
}

.toggle {
  display: inline-flex;
  align-items: center;
  gap: 0.6rem;
  background: rgba(255, 255, 255, 0.02);
  padding: 0.65rem 0.75rem;
  border-radius: 0.8rem;
  border: 1px solid var(--border-strong);
  font-weight: 600;
}

.actions-row {
  grid-column: span 2;
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 0.5rem;
}

button {
  border: none;
  cursor: pointer;
  border-radius: 0.8rem;
  font-weight: 800;
  padding: 0.75rem 1rem;
  color: var(--text-primary);
}

.primary {
  background: linear-gradient(135deg, #60a5fa, #2563eb);
  color: #0b1220;
  box-shadow: 0 12px 28px rgba(37, 99, 235, 0.25);
}

.primary.ghost {
  background: rgba(96, 165, 250, 0.12);
  color: var(--text-primary);
  border: 1px solid rgba(96, 165, 250, 0.35);
}

.ghost {
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid var(--border-strong);
}

.pill {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.45rem 0.75rem;
  border-radius: 999px;
  font-weight: 800;
  border: 1px solid var(--border-strong);
  color: var(--text-primary);
  background: rgba(255, 255, 255, 0.05);
}

.pill.success {
  border-color: rgba(34, 197, 94, 0.4);
  background: rgba(34, 197, 94, 0.1);
}

.pill.ghost {
  border-color: var(--border-strong);
  background: transparent;
}

.card.stack {
  gap: 0.75rem;
}

.stack__item {
  padding: 0.65rem 0.85rem;
  border-radius: 0.85rem;
  background: rgba(255, 255, 255, 0.02);
  border: 1px solid var(--border-strong);
}

.stack__item h2 {
  margin: 0.1rem 0;
}

.delta {
  color: var(--text-muted);
  margin: 0;
}

.delta.positive {
  color: #22c55e;
}

.card.actions .action-buttons {
  display: grid;
  gap: 0.5rem;
}

.card.actions .status {
  margin-top: 1rem;
}

.table {
  margin-top: 0.75rem;
  display: grid;
  gap: 0.35rem;
}

.table__head,
.table__row {
  display: grid;
  grid-template-columns: 2fr 1fr 1fr 1fr 1fr;
  gap: 0.75rem;
  align-items: center;
}

.table__head {
  padding: 0.65rem 0.75rem;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-size: 0.85rem;
}

.table__row {
  padding: 0.75rem 0.85rem;
  border-radius: 0.75rem;
  background: rgba(255, 255, 255, 0.02);
  border: 1px solid var(--border-strong);
}

.strong {
  font-weight: 800;
  margin: 0;
}

.row-actions {
  display: inline-flex;
  gap: 0.5rem;
  justify-content: flex-end;
}

@media (max-width: 1080px) {
  .grid {
    grid-template-columns: repeat(1, minmax(0, 1fr));
  }
  .col-8,
  .col-4,
  .full-row,
  .actions-row,
  .form-actions {
    grid-column: span 1 !important;
  }
  .form-grid {
    grid-template-columns: 1fr;
  }
  .table__head,
  .table__row {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
