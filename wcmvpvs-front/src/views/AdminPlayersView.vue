<template>
  <AppShell>
    <section class="page">
      <div class="page__header">
        <div>
          <p class="eyebrow">Roster editor</p>
          <h1>Giocatori</h1>
          <p class="muted">Gestisci convocazioni, ruoli e foto con una tabella card-based.</p>
        </div>
        <button class="primary" type="button" @click="addPlayer">Nuovo giocatore</button>
      </div>

      <div class="grid">
        <div class="col col-8">
          <div class="card">
            <div class="table">
              <div class="table__head">
                <span>Giocatore</span>
                <span>Numero</span>
                <span>Ruolo</span>
                <span>Convocato</span>
                <span>Azioni</span>
              </div>
              <div v-for="player in players" :key="player.id" class="table__row">
                <div class="player">
                  <div class="avatar">{{ player.initials }}</div>
                  <div>
                    <p class="strong">{{ player.name }}</p>
                    <p class="muted">{{ player.team }}</p>
                  </div>
                </div>
                <div class="muted">#{{ player.number }}</div>
                <div><span class="pill ghost">{{ player.role }}</span></div>
                <div>
                  <label class="toggle">
                    <input v-model="player.active" type="checkbox" />
                    <span>{{ player.active ? 'Convocato' : 'In attesa' }}</span>
                  </label>
                </div>
                <div class="row-actions">
                  <button class="ghost" type="button" @click="selectPlayer(player)">Modifica</button>
                  <button class="ghost danger" type="button" @click="removePlayer(player.id)">Rimuovi</button>
                </div>
              </div>
            </div>
          </div>
        </div>
        <div class="col col-4">
          <div class="card">
            <p class="eyebrow">Dettagli</p>
            <h3>{{ selectedPlayer?.name || 'Seleziona un giocatore' }}</h3>
            <p class="muted">Aggiorna ruolo, numero e stato convocazione.</p>
            <form v-if="selectedPlayer" class="form" @submit.prevent="persistPlayer">
              <label>
                Nome
                <input v-model="selectedPlayer.name" type="text" required />
              </label>
              <label>
                Numero
                <input v-model.number="selectedPlayer.number" type="number" min="1" required />
              </label>
              <label>
                Ruolo
                <input v-model="selectedPlayer.role" type="text" required />
              </label>
              <label class="toggle">
                <input v-model="selectedPlayer.active" type="checkbox" />
                <span>Convocato</span>
              </label>
              <div class="actions-row">
                <button class="ghost" type="button" @click="selectedPlayer = null">Annulla</button>
                <button class="primary" type="submit">Salva</button>
              </div>
            </form>
            <div v-else class="empty">
              <p class="muted">Seleziona un giocatore per modificare i dettagli.</p>
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

const players = ref([
  { id: 1, name: 'Alejandro Hernandez', team: 'Power Volley', number: 9, role: 'Opposto', active: true, initials: 'AH' },
  { id: 2, name: 'Marco Bianchi', team: 'Power Volley', number: 13, role: 'Centrale', active: true, initials: 'MB' },
  { id: 3, name: 'Davide Colombo', team: 'Power Volley', number: 7, role: 'Schiacciatore', active: false, initials: 'DC' },
]);

const selectedPlayer = ref(players.value[0]);

function selectPlayer(player) {
  selectedPlayer.value = { ...player };
}

function persistPlayer() {
  players.value = players.value.map((player) => (player.id === selectedPlayer.value.id ? { ...selectedPlayer.value } : player));
}

function removePlayer(id) {
  players.value = players.value.filter((player) => player.id !== id);
  if (selectedPlayer.value?.id === id) {
    selectedPlayer.value = null;
  }
}

function addPlayer() {
  const nextId = players.value.length + 1;
  const newPlayer = {
    id: nextId,
    name: `Nuovo giocatore ${nextId}`,
    team: 'Power Volley',
    number: 99,
    role: 'Schiacciatore',
    active: false,
    initials: 'NJ',
  };
  players.value = [newPlayer, ...players.value];
  selectedPlayer.value = newPlayer;
}
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

.card {
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid var(--border-strong);
  border-radius: 1rem;
  padding: 1rem;
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
}

.player {
  display: inline-flex;
  align-items: center;
  gap: 0.65rem;
}

.avatar {
  width: 2.5rem;
  height: 2.5rem;
  border-radius: 0.8rem;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #f472b6, #a855f7);
  font-weight: 800;
  color: #0b1220;
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

.pill.ghost {
  background: rgba(255, 255, 255, 0.04);
}

.toggle {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  color: var(--text-primary);
  font-weight: 700;
}

.row-actions {
  display: inline-flex;
  gap: 0.5rem;
  justify-content: flex-end;
}

.form {
  display: grid;
  gap: 0.75rem;
  margin-top: 1rem;
}

label {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  color: var(--text-primary);
  font-weight: 700;
}

input {
  width: 100%;
  padding: 0.85rem 0.9rem;
  border-radius: 0.75rem;
  border: 1px solid var(--border-strong);
  background: rgba(255, 255, 255, 0.03);
  color: var(--text-primary);
}

input:focus {
  outline: 2px solid rgba(96, 165, 250, 0.4);
}

.actions-row {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
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

.ghost.danger {
  border-color: rgba(248, 113, 113, 0.35);
  color: #fca5a5;
}

.strong {
  margin: 0;
  font-weight: 800;
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
