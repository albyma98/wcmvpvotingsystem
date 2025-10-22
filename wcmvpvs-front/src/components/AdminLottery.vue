<template>
  <div class="lottery-shell">
    <header class="lottery-header">
      <div>
        <h1>Lotteria evento</h1>
        <p class="subtitle">Estrai casualmente un vincitore tra i ticket validi</p>
      </div>
      <div class="header-actions" v-if="isAuthenticated">
        <span class="muted">Connesso come <strong>{{ activeUsername }}</strong></span>
        <button class="btn ghost" type="button" @click="goToPortal">Torna al pannello</button>
        <button class="btn secondary" type="button" @click="logout">Esci</button>
      </div>
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

    <section v-else class="card lottery-card">
      <div class="filters">
        <label>
          Seleziona evento
          <input
            v-model="eventInput"
            type="text"
            list="lottery-event-options"
            :disabled="isLoadingEvents || !events.length"
            placeholder="Digita per filtrare gli eventi"
            @change="handleEventInput"
            @blur="handleEventInput"
          />
          <datalist id="lottery-event-options">
            <option v-for="event in events" :key="event.id" :value="eventLabel(event)"></option>
          </datalist>
        </label>
        <button class="btn secondary" type="button" @click="refreshTickets" :disabled="!selectedEventId || isLoadingTickets">
          Aggiorna ticket
        </button>
      </div>

      <p v-if="globalError" class="error">{{ globalError }}</p>
      <p v-if="drawError" class="error">{{ drawError }}</p>

      <div v-if="selectedEventId" class="lottery-status">
        <p class="muted">
          Ticket disponibili: <strong>{{ tickets.length }}</strong>
        </p>
        <p v-if="winner" class="winner-banner">
          Vincitore: <strong>{{ winnerDisplay }}</strong>
        </p>
      </div>

      <div class="draw-card" :class="{ active: isDrawing }">
        <div class="display">
          <span>{{ currentDisplay }}</span>
        </div>
        <button
          class="btn primary"
          type="button"
          @click="startDraw"
          :disabled="isDrawing || !canDraw"
        >
          {{ isDrawing ? 'Estrazione in corso…' : 'Estrai vincitore' }}
        </button>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';
import { apiClient } from '../api';

const basePath = import.meta.env.BASE_URL ?? '/';

const teams = ref([]);
const events = ref([]);
const tickets = ref([]);
const selectedEventId = ref(0);
const eventInput = ref('');

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
const drawError = ref('');
const isLoadingEvents = ref(false);
const isLoadingTickets = ref(false);

const isDrawing = ref(false);
const winner = ref(null);
const displayCode = ref('Premi “Estrai vincitore”');
const animationInterval = ref(null);
const animationTimeout = ref(null);

const authHeaders = computed(() => ({
  headers: {
    Authorization: token.value ? `Bearer ${token.value}` : '',
  },
}));

const winnerDisplay = computed(() => {
  if (!winner.value) {
    return '';
  }
  return formatTicketDisplay(winner.value);
});

const currentDisplay = computed(() => {
  if (isDrawing.value && displayCode.value) {
    return displayCode.value;
  }
  if (winner.value) {
    return winnerDisplay.value;
  }
  return displayCode.value || 'Nessun ticket selezionato';
});

function eventLabel(event) {
  return `${teamName(event.team1_id)} - ${teamName(event.team2_id)}`;
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

function formatTicketDisplay(ticket) {
  const parts = [ticket.ticketCode];
  const fullName = [ticket.playerFirstName, ticket.playerLastName].filter(Boolean).join(' ');
  if (fullName) {
    parts.push(fullName);
  }
  return parts.join(' • ');
}

function normalizeEvent(event) {
  return {
    id: event.id,
    team1_id: event.team1_id,
    team2_id: event.team2_id,
    start_datetime: event.start_datetime,
    location: event.location,
  };
}

function normalizeTeam(team) {
  return {
    id: team.id,
    name: team.name,
  };
}

function normalizeTicket(ticket) {
  return {
    voteId: ticket.vote_id,
    ticketCode: ticket.ticket_code,
    playerId: ticket.player_id,
    playerFirstName: ticket.player_first_name,
    playerLastName: ticket.player_last_name,
    createdAt: ticket.created_at,
  };
}

function ensureValidSelection() {
  if (events.value.length === 0) {
    selectedEventId.value = 0;
    eventInput.value = '';
    tickets.value = [];
    return;
  }
  const selectedEvent = events.value.find((event) => event.id === selectedEventId.value) || events.value[0];
  selectedEventId.value = selectedEvent.id;
  eventInput.value = eventLabel(selectedEvent);
}

function handleEventInput() {
  const normalized = eventInput.value.trim().toLowerCase();
  if (!normalized) {
    selectedEventId.value = 0;
    return;
  }
  const eventMatch =
    events.value.find((event) => eventLabel(event).toLowerCase() === normalized) ||
    events.value.find((event) => eventLabel(event).toLowerCase().includes(normalized));
  selectedEventId.value = eventMatch ? eventMatch.id : 0;
}

function teamName(id) {
  const team = teams.value.find((teamItem) => teamItem.id === id);
  return team ? team.name : `Squadra ${id}`;
}

function clearTimers() {
  if (animationInterval.value) {
    clearInterval(animationInterval.value);
    animationInterval.value = null;
  }
  if (animationTimeout.value) {
    clearTimeout(animationTimeout.value);
    animationTimeout.value = null;
  }
}

function resetState() {
  events.value = [];
  tickets.value = [];
  selectedEventId.value = 0;
  winner.value = null;
  displayCode.value = 'Premi “Estrai vincitore”';
  clearTimers();
}

function logout() {
  token.value = '';
  activeUsername.value = '';
  localStorage.removeItem('adminToken');
  localStorage.removeItem('adminUsername');
  resetState();
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
    await loadInitialData();
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

async function loadEvents() {
  if (!isAuthenticated.value) {
    return;
  }
  isLoadingEvents.value = true;
  globalError.value = '';
  try {
    const { data } = await secureRequest(() => apiClient.get('/events', authHeaders.value));
    events.value = Array.isArray(data) ? data.map(normalizeEvent) : [];
    ensureValidSelection();
  } finally {
    isLoadingEvents.value = false;
  }
}

async function loadTeams() {
  if (!isAuthenticated.value) {
    return;
  }
  try {
    const { data } = await secureRequest(() => apiClient.get('/teams', authHeaders.value));
    teams.value = Array.isArray(data) ? data.map(normalizeTeam) : [];
  } catch (error) {
    if (error?.response?.status === 401) {
      return;
    }
  }
}

async function loadInitialData() {
  await Promise.all([loadTeams(), loadEvents()]);
}

async function loadTickets(eventId) {
  if (!eventId) {
    tickets.value = [];
    return;
  }
  isLoadingTickets.value = true;
  globalError.value = '';
  drawError.value = '';
  try {
    const { data } = await secureRequest(() => apiClient.get(`/events/tickets/${eventId}`, authHeaders.value));
    tickets.value = Array.isArray(data) ? data.map(normalizeTicket) : [];
    if (!tickets.value.length) {
      displayCode.value = 'Nessun ticket disponibile';
      winner.value = null;
    } else {
      displayCode.value = 'Premi “Estrai vincitore”';
      winner.value = null;
    }
  } finally {
    isLoadingTickets.value = false;
  }
}

function refreshTickets() {
  if (selectedEventId.value) {
    loadTickets(selectedEventId.value);
  }
}

const canDraw = computed(() => selectedEventId.value > 0 && tickets.value.length > 0 && !isLoadingTickets.value);

function startDraw() {
  if (!canDraw.value || isDrawing.value) {
    if (!tickets.value.length) {
      drawError.value = 'Nessun ticket disponibile per questo evento.';
    }
    return;
  }
  drawError.value = '';
  winner.value = null;
  isDrawing.value = true;
  clearTimers();

  const updateDisplay = () => {
    if (!tickets.value.length) {
      displayCode.value = 'Nessun ticket disponibile';
      return;
    }
    const randomTicket = tickets.value[Math.floor(Math.random() * tickets.value.length)];
    displayCode.value = formatTicketDisplay(randomTicket);
  };

  updateDisplay();
  animationInterval.value = setInterval(updateDisplay, 120);
  animationTimeout.value = setTimeout(() => {
    clearTimers();
    if (!tickets.value.length) {
      drawError.value = 'Nessun ticket disponibile per questo evento.';
      displayCode.value = 'Nessun ticket disponibile';
      isDrawing.value = false;
      return;
    }
    const finalTicket = tickets.value[Math.floor(Math.random() * tickets.value.length)];
    winner.value = finalTicket;
    displayCode.value = formatTicketDisplay(finalTicket);
    isDrawing.value = false;
  }, 3000);
}

function goToPortal() {
  if (typeof window === 'undefined') {
    return;
  }
  const target = new URL(basePath || '/', window.location.origin);
  if (!target.pathname.endsWith('/')) {
    target.pathname = `${target.pathname}/`;
  }
  target.pathname = `${target.pathname.replace(/\/+$/, '')}/admin`;
  window.location.href = target.toString();
}

watch(selectedEventId, (id) => {
  if (id) {
    const currentEvent = events.value.find((event) => event.id === id);
    if (currentEvent) {
      eventInput.value = eventLabel(currentEvent);
    }
    loadTickets(id);
  } else {
    if (!eventInput.value) {
      eventInput.value = '';
    }
    tickets.value = [];
  }
});

watch(events, ensureValidSelection);

watch(isAuthenticated, (value) => {
  if (value) {
    loadInitialData();
  }
});

onMounted(() => {
  if (isAuthenticated.value) {
    loadInitialData();
  }
});

onBeforeUnmount(() => {
  clearTimers();
});
</script>

<style scoped>
.lottery-shell {
  max-width: 960px;
  margin: 0 auto;
  padding: 2.5rem 1.5rem 3rem;
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  color: #0f172a;
}

.lottery-header {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  color: #f8fafc;
}

.lottery-header h1 {
  font-size: 2.2rem;
  margin: 0;
}

.subtitle {
  margin: 0;
  color: rgba(248, 250, 252, 0.75);
}

.header-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  align-items: center;
}

.card {
  background: rgba(248, 250, 252, 0.92);
  border-radius: 1rem;
  box-shadow: 0 20px 40px rgba(15, 23, 42, 0.25);
  padding: 1.75rem;
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.login-card h2 {
  margin: 0;
}

.form-grid {
  display: grid;
  gap: 1rem;
}

label {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
  font-weight: 600;
}

input,
select {
  width: 100%;
  border-radius: 0.75rem;
  border: 1px solid #cbd5f5;
  padding: 0.6rem 0.85rem;
  font-size: 1rem;
  background: #f1f5f9;
  color: #0f172a;
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

.btn.ghost {
  background: transparent;
  color: #f8fafc;
  border: 1px solid rgba(248, 250, 252, 0.6);
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

.filters {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  align-items: flex-end;
}

.filters label {
  flex: 1 1 260px;
}

.lottery-status {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.draw-card {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  align-items: center;
  text-align: center;
}

.draw-card .display {
  width: min(100%, 480px);
  border-radius: 1.2rem;
  padding: 2.5rem 1rem;
  background: radial-gradient(circle at 30% 30%, rgba(59, 130, 246, 0.18), rgba(37, 99, 235, 0.35));
  color: #0f172a;
  font-size: clamp(1.5rem, 4vw, 2.5rem);
  font-weight: 700;
  letter-spacing: 0.08em;
  min-height: 4.2rem;
  display: flex;
  align-items: center;
  justify-content: center;
}

.draw-card.active .display {
  animation: pulse 0.6s ease-in-out infinite alternate;
}

.winner-banner {
  margin: 0;
  padding: 0.75rem 1rem;
  border-radius: 0.85rem;
  background: rgba(22, 163, 74, 0.15);
  color: #166534;
  font-weight: 600;
}

.muted {
  color: #64748b;
  margin: 0;
}

.error {
  color: #dc2626;
  margin: 0;
}

@keyframes pulse {
  from {
    transform: scale(0.995);
    box-shadow: 0 0 0 rgba(59, 130, 246, 0.25);
  }
  to {
    transform: scale(1.01);
    box-shadow: 0 15px 25px rgba(59, 130, 246, 0.3);
  }
}

@media (min-width: 768px) {
  .lottery-header {
    flex-direction: row;
    justify-content: space-between;
    align-items: center;
  }

  .draw-card {
    padding: 1rem 2rem 2rem;
  }
}
</style>
