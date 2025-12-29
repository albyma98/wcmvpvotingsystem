<template>
  <AppShell>
    <section class="page">
      <div class="page__header">
        <div>
          <p class="eyebrow">Analitiche</p>
          <h1>Storico eventi</h1>
          <p class="muted">KPI cards, tabella eventi e dettaglio selezionato in una UI SaaS.</p>
        </div>
      </div>

      <div class="kpi-grid">
        <div class="card kpi" v-for="card in kpiCards" :key="card.label">
          <p class="eyebrow">{{ card.label }}</p>
          <h2>{{ card.value }}</h2>
          <p class="delta" :class="{ positive: card.delta > 0 }">
            {{ card.delta > 0 ? '+' : '' }}{{ card.delta }}% rispetto al periodo precedente
          </p>
        </div>
      </div>

      <div class="grid">
        <div class="col col-8">
          <div class="card">
            <div class="table">
              <div class="table__head">
                <span>Evento</span>
                <span>Data</span>
                <span>Voti</span>
                <span>Click sponsor</span>
                <span>Stato</span>
              </div>
              <div
                v-for="event in events"
                :key="event.id"
                class="table__row selectable"
                :class="{ selected: event.id === selectedEvent?.id }"
                @click="selectedEvent = event"
              >
                <div>
                  <p class="strong">{{ event.name }}</p>
                  <p class="muted">{{ event.teams }}</p>
                </div>
                <div class="muted">{{ event.date }}</div>
                <div class="strong">{{ event.votes.toLocaleString() }}</div>
                <div class="strong">{{ event.sponsorClicks.toLocaleString() }}</div>
                <div><span class="pill" :class="event.status">{{ event.status }}</span></div>
              </div>
            </div>
          </div>
        </div>
        <div class="col col-4">
          <div class="card">
            <p class="eyebrow">Dettaglio evento</p>
            <h3>{{ selectedEvent?.name || 'Seleziona un evento' }}</h3>
            <p class="muted">
              {{ selectedEvent ? 'Analisi rapida di voti, utenti e sponsor performance.' : 'Clicca una riga per vedere il dettaglio.' }}
            </p>
            <div v-if="selectedEvent" class="detail">
              <div class="detail__row">
                <span>Voti totali</span>
                <strong>{{ selectedEvent.votes.toLocaleString() }}</strong>
              </div>
              <div class="detail__row">
                <span>Utenti</span>
                <strong>{{ selectedEvent.users.toLocaleString() }}</strong>
              </div>
              <div class="detail__row">
                <span>Click sponsor</span>
                <strong>{{ selectedEvent.sponsorClicks.toLocaleString() }}</strong>
              </div>
              <div class="detail__row">
                <span>Esito</span>
                <strong>{{ selectedEvent.status }}</strong>
              </div>
              <div class="actions-row">
                <button class="primary" type="button">Scarica report</button>
                <button class="ghost" type="button">Apri dettagli</button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  </AppShell>
</template>

<script setup>
import { ref } from 'vue';
import AppShell from '../components/layout/AppShell.vue';

const kpiCards = [
  { label: 'Voti totali', value: '48.2K', delta: 7.4 },
  { label: 'Utenti unici', value: '12.9K', delta: 4.1 },
  { label: 'CTR sponsor', value: '18%', delta: 2.3 },
];

const events = [
  { id: 1, name: 'Playoff - Gara 3', teams: 'Power Volley vs Sir Safety', date: '12 Mar 2025', votes: 18240, users: 5230, sponsorClicks: 1240, status: 'live' },
  { id: 2, name: 'Regular Season', teams: 'Milano vs Padova', date: '05 Mar 2025', votes: 11200, users: 4200, sponsorClicks: 820, status: 'closed' },
  { id: 3, name: 'Coppa Italia', teams: 'Milano vs Modena', date: '22 Feb 2025', votes: 9800, users: 3600, sponsorClicks: 620, status: 'closed' },
];

const selectedEvent = ref(events[0]);
</script>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.page__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.eyebrow {
  text-transform: uppercase;
  letter-spacing: 0.12em;
  font-weight: 800;
  color: var(--text-muted);
  margin: 0;
}

h1 {
  margin: 0.1rem 0;
}

.muted {
  color: var(--text-muted);
  margin: 0;
}

.kpi-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 0.75rem;
}

.card {
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid var(--border-strong);
  border-radius: 1rem;
  padding: 1rem;
}

.kpi {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.delta {
  color: var(--text-muted);
  margin: 0;
}

.delta.positive {
  color: #22c55e;
}

.grid {
  display: grid;
  grid-template-columns: repeat(12, 1fr);
  gap: 1rem;
}

.col {
  display: flex;
  flex-direction: column;
}

.col-8 {
  grid-column: span 8;
}

.col-4 {
  grid-column: span 4;
}

.table {
  display: grid;
  gap: 0.5rem;
}

.table__head,
.table__row {
  display: grid;
  grid-template-columns: 2fr 1fr 1fr 1fr 1fr;
  align-items: center;
  gap: 0.75rem;
}

.table__head {
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-size: 0.9rem;
}

.table__row {
  padding: 0.75rem;
  border-radius: 0.9rem;
  border: 1px solid var(--border-strong);
  background: rgba(255, 255, 255, 0.02);
  cursor: pointer;
  transition: border-color 0.15s ease, transform 0.15s ease;
}

.table__row:hover {
  border-color: rgba(96, 165, 250, 0.4);
  transform: translateY(-1px);
}

.table__row.selected {
  border-color: rgba(96, 165, 250, 0.6);
  box-shadow: 0 10px 24px rgba(37, 99, 235, 0.2);
}

.strong {
  margin: 0;
  font-weight: 800;
}

.pill {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.35rem 0.7rem;
  border-radius: 999px;
  border: 1px solid var(--border-strong);
  background: rgba(255, 255, 255, 0.05);
  font-weight: 700;
}

.pill.live {
  border-color: rgba(34, 197, 94, 0.35);
  background: rgba(34, 197, 94, 0.1);
}

.detail {
  display: grid;
  gap: 0.6rem;
  margin-top: 1rem;
}

.detail__row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.6rem 0.75rem;
  border-radius: 0.8rem;
  border: 1px solid var(--border-strong);
  background: rgba(255, 255, 255, 0.02);
}

.actions-row {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 0.5rem;
}

button {
  border: none;
  border-radius: 0.8rem;
  padding: 0.75rem 1rem;
  font-weight: 800;
  cursor: pointer;
}

.primary {
  background: linear-gradient(135deg, #60a5fa, #2563eb);
  color: #0b1220;
}

.ghost {
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid var(--border-strong);
  color: var(--text-primary);
}

@media (max-width: 1080px) {
  .grid {
    grid-template-columns: 1fr;
  }

  .col-8,
  .col-4 {
    grid-column: span 1;
  }

  .table__head,
  .table__row {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
