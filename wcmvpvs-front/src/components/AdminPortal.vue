<template>
  <div class="admin-portal">
    <header class="admin-header">
      <h1>Area amministratore</h1>
      <p class="subtitle">Gestisci eventi, squadre e votazioni MVP</p>
    </header>

    <section v-if="!isAuthenticated" class="card login-card">
      <h2>Accedi</h2>
      <form @submit.prevent="login" class="form-grid">
        <label>
          Username
          <input v-model.trim="loginForm.username" type="text" autocomplete="username" required />
        </label>
        <label>
          Password
          <input v-model="loginForm.password" type="password" autocomplete="current-password" required />
        </label>
        <button class="btn primary" type="submit" :disabled="isLoggingIn">
          {{ isLoggingIn ? 'Accesso in corso…' : 'Entra' }}
        </button>
      </form>
      <p v-if="loginError" class="error">{{ loginError }}</p>
    </section>

    <section v-else class="portal">
      <div class="toolbar">
        <div class="user-info">
          <span>Connesso come <strong>{{ activeUsername }}</strong></span>
          <button class="btn outline" type="button" @click="goToLottery">Lotteria</button>
          <button class="btn secondary" type="button" @click="logout">Esci</button>
        </div>
        <nav class="tab-bar" aria-label="Sezioni amministrative">
          <button
            v-for="tab in tabs"
            :key="tab.id"
            :class="['tab', { active: section === tab.id }]"
            type="button"
            @click="section = tab.id"
          >
            {{ tab.label }}
          </button>
        </nav>
      </div>

      <p v-if="globalError" class="error">{{ globalError }}</p>

      <section v-if="section === 'events'" class="card">
        <header class="section-header">
          <h2>Eventi</h2>
          <p>Crea una nuova partita per abilitare il voto pubblico.</p>
        </header>
        <div class="actions-row">
          <button
            class="btn outline"
            type="button"
            @click="deactivateEvents"
            :disabled="!activeEventId || isDisablingEvents"
          >
            {{ isDisablingEvents ? 'Disattivazione…' : 'Disattiva eventi' }}
          </button>
        </div>
        <p v-if="!hasEnoughTeams" class="info-banner">
          Aggiungi almeno due squadre dalla sezione "Squadre" per abilitare la creazione di un evento.
        </p>
        <form @submit.prevent="createEvent" class="form-grid">
          <label>
            Squadra di casa
            <input
              v-model="teamInputs.home"
              type="text"
              list="admin-team-options"
              :disabled="!hasEnoughTeams"
              placeholder="Digita il nome della squadra"
              required
              @change="handleTeamInput('home')"
              @blur="handleTeamInput('home')"
            />
            <small class="field-hint" v-if="hasEnoughTeams">
              Scegli dalla lista oppure digita per filtrare le squadre disponibili.
            </small>
          </label>
          <label>
            Squadra ospite
            <input
              v-model="teamInputs.away"
              type="text"
              list="admin-team-options"
              :disabled="!hasEnoughTeams"
              placeholder="Digita il nome della squadra"
              required
              @change="handleTeamInput('away')"
              @blur="handleTeamInput('away')"
            />
            <small class="field-hint" v-if="hasEnoughTeams">
              Seleziona una squadra diversa da quella di casa.
            </small>
          </label>
          <datalist id="admin-team-options">
            <option v-for="team in teams" :key="team.id" :value="teamOptionValue(team)"></option>
          </datalist>
          <label>
            Data e ora
            <input
              v-model="newEvent.start_datetime"
              type="datetime-local"
              :disabled="!hasEnoughTeams"
              required
            />
          </label>
          <label>
            Location
            <input
              v-model.trim="newEvent.location"
              type="text"
              placeholder="Es. Palazzetto dello Sport"
              :disabled="!hasEnoughTeams"
            />
          </label>
          <button class="btn primary" type="submit" :disabled="!hasEnoughTeams">Crea evento</button>
        </form>

        <div v-if="lastCreatedEventLink" class="hint">
          Nuovo evento creato! Link pubblico:
          <a :href="lastCreatedEventLink" target="_blank" rel="noopener">{{ lastCreatedEventLink }}</a>
          <button class="btn link" type="button" @click="copyLink(lastCreatedEventLink)">Copia</button>
        </div>

        <ul class="item-list">
          <li v-for="event in events" :key="event.id" :class="['item', { active: event.is_active }]">
            <div class="item-body">
              <h3>
                {{ eventLabel(event) }}
                <span v-if="event.is_active" class="badge">Attivo</span>
              </h3>
              <p class="muted">{{ formatEventDate(event.start_datetime) }} • {{ event.location || 'Location da definire' }}</p>
              <p class="muted">
                Link voto:
                <a :href="buildEventLink(event.id)" target="_blank" rel="noopener">{{ buildEventLink(event.id) }}</a>
              </p>
            </div>
            <div class="item-actions">
              <button
                class="btn success"
                type="button"
                @click="activateEvent(event.id)"
                :disabled="event.is_active || updatingEventId === event.id"
              >
                <span v-if="event.is_active">Evento attivo</span>
                <span v-else-if="updatingEventId === event.id">Attivazione…</span>
                <span v-else>Attiva</span>
              </button>
              <button class="btn secondary" type="button" @click="openVote(event.id)">Apri pagina voto</button>
              <button class="btn danger" type="button" @click="deleteEvent(event.id)">Elimina</button>
            </div>
          </li>
        </ul>
      </section>

      <section v-else-if="section === 'teams'" class="card">
        <header class="section-header">
          <h2>Squadre</h2>
        </header>
        <form @submit.prevent="createTeam" class="form-inline">
          <input v-model.trim="newTeamName" type="text" placeholder="Nome squadra" required />
          <button class="btn primary" type="submit">Aggiungi</button>
        </form>
        <ul class="item-list compact">
          <li v-for="team in teams" :key="team.id" class="item">
            <span>{{ team.name }}</span>
            <button class="btn danger" type="button" @click="deleteTeam(team.id)">Elimina</button>
          </li>
        </ul>
      </section>

      <section v-else-if="section === 'players'" class="card">
        <header class="section-header">
          <h2>Giocatori</h2>
        </header>
        <form @submit.prevent="createPlayer" class="form-grid">
          <input v-model.trim="newPlayer.first_name" type="text" placeholder="Nome" required />
          <input v-model.trim="newPlayer.last_name" type="text" placeholder="Cognome" required />
          <input v-model.trim="newPlayer.role" type="text" placeholder="Ruolo" required />
          <input v-model.number="newPlayer.jersey_number" type="number" min="0" placeholder="Numero maglia" />
          <input v-model.trim="newPlayer.image_url" type="url" placeholder="URL immagine" />
          <label>
            Squadra
            <select v-model.number="newPlayer.team_id" required>
              <option disabled value="0">Seleziona squadra</option>
              <option v-for="team in teams" :key="team.id" :value="team.id">
                {{ team.name }}
              </option>
            </select>
          </label>
          <button class="btn primary" type="submit">Aggiungi</button>
        </form>
        <ul class="item-list">
          <li v-for="player in players" :key="player.id" class="item">
            <div class="item-body">
              <h3>{{ player.first_name }} {{ player.last_name }}</h3>
              <p class="muted">{{ player.role }} • #{{ player.jersey_number }} • {{ teamName(player.team_id) }}</p>
            </div>
            <div class="item-actions">
              <button class="btn danger" type="button" @click="deletePlayer(player.id)">Elimina</button>
            </div>
          </li>
        </ul>
      </section>

      <section v-else-if="section === 'admins'" class="card">
        <header class="section-header">
          <h2>Utenti amministratori</h2>
        </header>
        <form @submit.prevent="createAdmin" class="form-grid">
          <input v-model.trim="newAdmin.username" type="text" placeholder="Username" required />
          <input v-model="newAdmin.password" type="password" placeholder="Password" required />
          <input v-model.trim="newAdmin.role" type="text" placeholder="Ruolo (es. staff)" />
          <button class="btn primary" type="submit">Aggiungi</button>
        </form>
        <ul class="item-list compact">
          <li v-for="admin in admins" :key="admin.id" class="item">
            <div>
              <strong>{{ admin.username }}</strong>
              <span class="muted"> • {{ admin.role || 'staff' }}</span>
            </div>
            <button class="btn danger" type="button" @click="deleteAdmin(admin.id)">Elimina</button>
          </li>
        </ul>
      </section>
    </section>
  </div>
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue';
import { apiClient } from '../api';

const basePath = import.meta.env.BASE_URL ?? '/';
const baseVoteUrl = new URL(basePath, window.location.origin);

const section = ref('events');
const tabs = [
  { id: 'events', label: 'Eventi' },
  { id: 'teams', label: 'Squadre' },
  { id: 'players', label: 'Giocatori' },
  { id: 'admins', label: 'Admin' },
];

const teams = ref([]);
const players = ref([]);
const events = ref([]);
const admins = ref([]);
const updatingEventId = ref(0);
const isDisablingEvents = ref(false);

const newTeamName = ref('');
const newPlayer = reactive({
  first_name: '',
  last_name: '',
  role: '',
  jersey_number: 0,
  image_url: '',
  team_id: 0,
});
const newEvent = reactive({
  team1_id: 0,
  team2_id: 0,
  start_datetime: '',
  location: '',
});
const teamInputs = reactive({
  home: '',
  away: '',
});
const newAdmin = reactive({
  username: '',
  password: '',
  role: '',
});
const lastCreatedEventLink = ref('');

const hasEnoughTeams = computed(() => teams.value.length >= 2);
const activeEventId = computed(() => {
  const activeEvent = events.value.find((event) => event.is_active);
  return activeEvent ? activeEvent.id : 0;
});

const token = ref(localStorage.getItem('adminToken') || '');
const activeUsername = ref(localStorage.getItem('adminUsername') || '');
const isAuthenticated = computed(() => Boolean(token.value));

const loginForm = reactive({
  username: '',
  password: '',
});
const isLoggingIn = ref(false);
const loginError = ref('');
const globalError = ref('');

const authHeaders = computed(() => ({
  headers: {
    Authorization: token.value ? `Bearer ${token.value}` : '',
  },
}));

function resetForms() {
  newTeamName.value = '';
  Object.assign(newPlayer, {
    first_name: '',
    last_name: '',
    role: '',
    jersey_number: 0,
    image_url: '',
    team_id: 0,
  });
  Object.assign(newEvent, {
    team1_id: 0,
    team2_id: 0,
    start_datetime: '',
    location: '',
  });
  teamInputs.home = '';
  teamInputs.away = '';
  Object.assign(newAdmin, { username: '', password: '', role: '' });
}

function ensureValidTeamSelection() {
  if (!hasEnoughTeams.value) {
    newEvent.team1_id = 0;
    newEvent.team2_id = 0;
    teamInputs.home = '';
    teamInputs.away = '';
    return;
  }

  const availableIds = new Set(teams.value.map((team) => team.id));

  if (!availableIds.has(newEvent.team1_id)) {
    newEvent.team1_id = 0;
    teamInputs.home = '';
  }

  if (
    !availableIds.has(newEvent.team2_id) ||
    (newEvent.team1_id !== 0 && newEvent.team1_id === newEvent.team2_id)
  ) {
    newEvent.team2_id = 0;
    teamInputs.away = '';
  }

  syncTeamInputsFromIds();
}

watch(teams, ensureValidTeamSelection);
watch(hasEnoughTeams, (enough) => {
  if (!enough) {
    newEvent.team1_id = 0;
    newEvent.team2_id = 0;
    teamInputs.home = '';
    teamInputs.away = '';
  }
});

function clearCollections() {
  teams.value = [];
  players.value = [];
  events.value = [];
  admins.value = [];
  lastCreatedEventLink.value = '';
}

function buildEventLink(eventId) {
  const url = new URL(baseVoteUrl.toString());
  if (eventId) {
    url.searchParams.set('eventId', String(eventId));
  } else {
    url.searchParams.delete('eventId');
  }
  return url.toString();
}

function goToLottery() {
  const target = new URL(basePath || '/', window.location.origin);
  if (!target.pathname.endsWith('/')) {
    target.pathname = `${target.pathname}/`;
  }
  target.pathname = `${target.pathname.replace(/\/+$/, '')}/admin/lottery`;
  window.location.href = target.toString();
}

function teamOptionValue(team) {
  return `${team.name} (#${team.id})`;
}

function syncTeamInputsFromIds() {
  const homeTeam = teams.value.find((team) => team.id === newEvent.team1_id);
  const awayTeam = teams.value.find((team) => team.id === newEvent.team2_id);
  teamInputs.home = homeTeam ? teamOptionValue(homeTeam) : '';
  teamInputs.away = awayTeam ? teamOptionValue(awayTeam) : '';
}

function findTeamFromInput(value) {
  const normalized = value.trim().toLowerCase();
  if (!normalized) {
    return undefined;
  }
  return (
    teams.value.find((team) => teamOptionValue(team).toLowerCase() === normalized) ||
    teams.value.find((team) => team.name.trim().toLowerCase() === normalized)
  );
}

function handleTeamInput(position) {
  const key = position === 'home' ? 'team1_id' : 'team2_id';
  const otherKey = position === 'home' ? 'team2_id' : 'team1_id';
  const otherInputKey = position === 'home' ? 'away' : 'home';
  const rawValue = teamInputs[position] || '';
  const matchedTeam = findTeamFromInput(rawValue);

  if (matchedTeam) {
    if (newEvent[otherKey] === matchedTeam.id) {
      newEvent[otherKey] = 0;
      teamInputs[otherInputKey] = '';
    }
    newEvent[key] = matchedTeam.id;
    teamInputs[position] = teamOptionValue(matchedTeam);
  } else {
    newEvent[key] = 0;
    teamInputs[position] = '';
  }
}

function eventLabel(event) {
  return `${teamName(event.team1_id)} vs ${teamName(event.team2_id)}`;
}

function teamName(id) {
  const team = teams.value.find((teamItem) => teamItem.id === id);
  return team ? team.name : '—';
}

function formatEventDate(value) {
  if (!value) {
    return 'Data da definire';
  }
  const date = new Date(value);
  if (!Number.isNaN(date.valueOf())) {
    return date.toLocaleString('it-IT');
  }
  return value.replace('T', ' ');
}

async function login() {
  if (isLoggingIn.value) {
    return;
  }
  loginError.value = '';
  globalError.value = '';
  isLoggingIn.value = true;
  try {
    const { data } = await apiClient.post('/admin/login', {
      username: loginForm.username,
      password: loginForm.password,
    });
    token.value = data.token;
    activeUsername.value = data.username;
    localStorage.setItem('adminToken', token.value);
    localStorage.setItem('adminUsername', activeUsername.value);
    loginForm.username = '';
    loginForm.password = '';
    await loadAll();
  } catch (error) {
    if (error?.response?.status === 401) {
      loginError.value = 'Credenziali non valide.';
    } else {
      loginError.value = 'Impossibile completare l\'accesso. Riprova.';
    }
  } finally {
    isLoggingIn.value = false;
  }
}

function logout() {
  token.value = '';
  activeUsername.value = '';
  localStorage.removeItem('adminToken');
  localStorage.removeItem('adminUsername');
  clearCollections();
}

function handleUnauthorized() {
  logout();
  loginError.value = 'Sessione scaduta. Effettua di nuovo il login.';
}

async function secureRequest(executor) {
  try {
    return await executor();
  } catch (error) {
    if (error?.response?.status === 401) {
      handleUnauthorized();
    } else {
      globalError.value = 'Si è verificato un errore imprevisto. Riprova più tardi.';
    }
    throw error;
  }
}

async function loadTeams() {
  const { data } = await secureRequest(() => apiClient.get('/teams', authHeaders.value));
  teams.value = data;
  ensureValidTeamSelection();
}

async function loadPlayers() {
  const { data } = await secureRequest(() => apiClient.get('/players', authHeaders.value));
  players.value = data;
}

async function loadEvents() {
  const { data } = await secureRequest(() => apiClient.get('/events', authHeaders.value));
  events.value = data;
}

async function loadAdmins() {
  const { data } = await secureRequest(() => apiClient.get('/admins', authHeaders.value));
  admins.value = data;
}

async function loadAll() {
  if (!isAuthenticated.value) {
    return;
  }
  await Promise.all([loadTeams(), loadPlayers(), loadEvents(), loadAdmins()]);
  resetForms();
}

async function createTeam() {
  if (!newTeamName.value) {
    return;
  }
  globalError.value = '';
  await secureRequest(() => apiClient.post('/teams', { name: newTeamName.value }, authHeaders.value));
  newTeamName.value = '';
  await loadTeams();
}

async function deleteTeam(id) {
  globalError.value = '';
  await secureRequest(() => apiClient.delete(`/teams/${id}`, authHeaders.value));
  await loadTeams();
}

async function createPlayer() {
  globalError.value = '';
  await secureRequest(() => apiClient.post('/players', newPlayer, authHeaders.value));
  Object.assign(newPlayer, {
    first_name: '',
    last_name: '',
    role: '',
    jersey_number: 0,
    image_url: '',
    team_id: 0,
  });
  await loadPlayers();
}

async function deletePlayer(id) {
  globalError.value = '';
  await secureRequest(() => apiClient.delete(`/players/${id}`, authHeaders.value));
  await loadPlayers();
}

async function createEvent() {
  globalError.value = '';
  if (!hasEnoughTeams.value) {
    globalError.value = 'Aggiungi almeno due squadre per creare un evento.';
    return;
  }
  if (!newEvent.team1_id || !newEvent.team2_id) {
    globalError.value = 'Seleziona entrambe le squadre.';
    return;
  }
  if (newEvent.team1_id === newEvent.team2_id) {
    globalError.value = 'Le due squadre devono essere diverse.';
    return;
  }
  if (!newEvent.start_datetime) {
    globalError.value = 'Imposta data e ora della partita.';
    return;
  }

  const { data } = await secureRequest(() => apiClient.post('/events', newEvent, authHeaders.value));
  await loadEvents();
  if (data?.id) {
    lastCreatedEventLink.value = buildEventLink(data.id);
  }
  Object.assign(newEvent, {
    team1_id: 0,
    team2_id: 0,
    start_datetime: '',
    location: '',
  });
}

async function deleteEvent(id) {
  globalError.value = '';
  await secureRequest(() => apiClient.delete(`/events/${id}`, authHeaders.value));
  await loadEvents();
}

async function activateEvent(id) {
  if (updatingEventId.value === id) {
    return;
  }
  globalError.value = '';
  updatingEventId.value = id;
  try {
    await secureRequest(() => apiClient.post(`/events/${id}/activate`, {}, authHeaders.value));
    await loadEvents();
  } finally {
    updatingEventId.value = 0;
  }
}

async function deactivateEvents() {
  if (isDisablingEvents.value) {
    return;
  }
  globalError.value = '';
  isDisablingEvents.value = true;
  try {
    await secureRequest(() => apiClient.post('/events/deactivate', {}, authHeaders.value));
    await loadEvents();
  } finally {
    isDisablingEvents.value = false;
  }
}

async function createAdmin() {
  globalError.value = '';
  await secureRequest(() => apiClient.post('/admins', newAdmin, authHeaders.value));
  Object.assign(newAdmin, { username: '', password: '', role: '' });
  await loadAdmins();
}

async function deleteAdmin(id) {
  globalError.value = '';
  await secureRequest(() => apiClient.delete(`/admins/${id}`, authHeaders.value));
  await loadAdmins();
}

function openVote(eventId) {
  const url = buildEventLink(eventId);
  window.open(url, '_blank', 'noopener');
}

async function copyLink(link) {
  try {
    await navigator.clipboard.writeText(link);
    globalError.value = '';
  } catch (error) {
    globalError.value = 'Impossibile copiare il link automaticamente.';
  }
}

if (isAuthenticated.value) {
  loadAll();
}
</script>

<style scoped>
.admin-portal {
  margin: 0 auto;
  max-width: 960px;
  padding: 2rem 1.5rem 3rem;
  color: #0f172a;
}

.admin-header {
  text-align: center;
  margin-bottom: 2rem;
}

.admin-header h1 {
  font-size: 2rem;
  margin: 0;
}

.subtitle {
  margin: 0.5rem 0 0;
  color: #475569;
}

.portal {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.toolbar {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

@media (min-width: 768px) {
  .toolbar {
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
  }
}

.user-info {
  display: flex;
  align-items: center;
  gap: 1rem;
  flex-wrap: wrap;
}

.tab-bar {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.tab {
  border: 1px solid #cbd5f5;
  background: #f8fafc;
  border-radius: 999px;
  padding: 0.5rem 1.25rem;
  cursor: pointer;
  color: #334155;
  transition: background-color 0.2s ease, color 0.2s ease;
}

.tab.active {
  background: linear-gradient(135deg, #2563eb, #7c3aed);
  border-color: transparent;
  color: #fff;
}

.card {
  background: #ffffff;
  border-radius: 1rem;
  padding: 1.5rem;
  box-shadow: 0 15px 35px rgba(15, 23, 42, 0.1);
  border: 1px solid rgba(148, 163, 184, 0.18);
}

.login-card {
  max-width: 480px;
  margin: 0 auto;
}

.section-header h2 {
  margin: 0 0 0.5rem;
}

.section-header p {
  margin: 0;
  color: #64748b;
}

.info-banner {
  margin: 0 0 1rem;
  padding: 0.85rem 1rem;
  border-radius: 0.75rem;
  background: rgba(59, 130, 246, 0.12);
  color: #1d4ed8;
  font-weight: 500;
}

.actions-row {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-bottom: 1rem;
}

.actions-row .btn {
  padding-left: 1.25rem;
  padding-right: 1.25rem;
}

.form-grid {
  display: grid;
  gap: 1rem;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  margin-bottom: 1.5rem;
}

.form-inline {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  margin-bottom: 1.5rem;
}

.form-grid label {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  font-weight: 600;
  color: #1e293b;
}

input,
select {
  border-radius: 0.75rem;
  border: 1px solid #cbd5f5;
  padding: 0.65rem 0.85rem;
  font-size: 0.95rem;
  background: #f8fafc;
  color: #0f172a;
}

.field-hint {
  font-size: 0.75rem;
  color: #64748b;
}

input:focus,
select:focus {
  outline: none;
  border-color: #6366f1;
  box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.2);
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.35rem;
  border-radius: 999px;
  border: none;
  padding: 0.6rem 1.4rem;
  font-weight: 600;
  cursor: pointer;
  transition: transform 0.15s ease, box-shadow 0.15s ease;
}

.btn.primary {
  background: linear-gradient(135deg, #2563eb, #7c3aed);
  color: #fff;
  box-shadow: 0 12px 25px rgba(59, 130, 246, 0.35);
}

.btn.secondary {
  background: #e2e8f0;
  color: #0f172a;
}

.btn.success {
  background: #22c55e;
  color: #fff;
}

.btn.success:disabled {
  opacity: 0.8;
  cursor: default;
}

.btn.outline {
  background: transparent;
  color: #2563eb;
  border: 1px solid rgba(37, 99, 235, 0.4);
}

.btn.danger {
  background: #f87171;
  color: #fff;
}

.btn.link {
  background: transparent;
  color: #2563eb;
  padding: 0.2rem 0.4rem;
}

.btn:disabled {
  cursor: not-allowed;
  opacity: 0.7;
  box-shadow: none;
}

.btn:not(:disabled):hover {
  transform: translateY(-1px);
  box-shadow: 0 10px 20px rgba(15, 23, 42, 0.15);
}

.item-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.item-list.compact {
  gap: 0.5rem;
}

.item {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 1rem;
  border-radius: 0.85rem;
  border: 1px solid rgba(148, 163, 184, 0.3);
  background: rgba(248, 250, 252, 0.8);
}

.item.active {
  border-color: rgba(99, 102, 241, 0.55);
  box-shadow: 0 10px 20px rgba(99, 102, 241, 0.2);
}

@media (min-width: 768px) {
  .item {
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
  }
}

.item-body h3 {
  margin: 0 0 0.35rem;
}

.badge {
  display: inline-flex;
  align-items: center;
  padding: 0.15rem 0.55rem;
  margin-left: 0.5rem;
  border-radius: 999px;
  background: rgba(79, 70, 229, 0.18);
  color: #4338ca;
  font-size: 0.7rem;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.item-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.muted {
  color: #64748b;
  margin: 0;
}

.error {
  color: #dc2626;
  margin-top: 0.75rem;
}

.hint {
  margin: 1rem 0 0;
  padding: 1rem;
  border-radius: 0.75rem;
  background: rgba(37, 99, 235, 0.08);
  color: #1d4ed8;
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  align-items: center;
}
</style>
