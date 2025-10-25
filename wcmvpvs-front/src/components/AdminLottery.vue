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
        <p v-if="lastWinner" class="winner-banner">
          Ultima estrazione: <strong>{{ lastWinnerDisplay }}</strong>
        </p>
      </div>

      <p
        v-if="selectedEventId && !selectedPrizes.length"
        class="muted no-prizes-message"
      >
        Nessun premio configurato per questo evento. Aggiorna i premi dal pannello amministratore.
      </p>

      <div v-if="selectedPrizes.length" class="prizes-wrapper">
        <div
          v-for="prize in selectedPrizes"
          :key="prize.id"
          :class="['prize-card', { active: currentPrizeId === prize.id }]"
        >
          <div class="prize-card__info">
            <h3>{{ prize.name || `Premio ${prize.position}` }}</h3>
            <p class="muted">Estrazione n° {{ prize.position }}</p>
            <p v-if="prize.winner" class="winner-banner">
              Vincitore: <strong>{{ prize.winnerDisplay }}</strong>
            </p>
          </div>
          <div class="prize-card__actions">
            <button
              class="btn primary"
              type="button"
              @click="startDraw(prize)"
              :disabled="!canDrawPrize(prize)"
            >
              <span v-if="prize.winner">Premio assegnato</span>
              <span v-else-if="isDrawing && currentPrizeId === prize.id">Estrazione in corso…</span>
              <span v-else>Estrai vincitore</span>
            </button>
            <button
              v-if="prize.winner"
              class="btn outline"
              type="button"
              @click="resetPrize(prize)"
              :disabled="isDrawing || isResettingPrize === prize.id"
            >
              {{ isResettingPrize === prize.id ? 'Ripristino…' : 'Annulla vincitore' }}
            </button>
          </div>
        </div>
      </div>

      <div v-if="selectedPrizes.length" class="draw-card" :class="{ active: isDrawing }">
        <p v-if="currentPrizeLabel" class="draw-card__subtitle">
          Estrazione premio: <strong>{{ currentPrizeLabel }}</strong>
        </p>
        <div class="display">
          <span>{{ currentDisplay }}</span>
        </div>
        <p v-if="!currentPrizeId" class="muted">Seleziona un premio per iniziare.</p>
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
const currentPrizeId = ref(0);
const lastWinner = ref(null);
const displayCode = ref('Seleziona un premio per iniziare');
const animationInterval = ref(null);
const animationTimeout = ref(null);
const isResettingPrize = ref(0);

const authHeaders = computed(() => ({
  headers: {
    Authorization: token.value ? `Bearer ${token.value}` : '',
  },
}));

const selectedEvent = computed(
  () => events.value.find((event) => event.id === selectedEventId.value) || null,
);

const selectedPrizes = computed(() => {
  const event = selectedEvent.value;
  if (!event || !Array.isArray(event.prizes)) {
    return [];
  }
  return event.prizes.map((prize) => ({
    ...prize,
    winnerDisplay: prize.winner
      ? formatTicketDisplay({
          ticketCode: prize.winner.ticketCode,
          playerFirstName: prize.winner.playerFirstName,
          playerLastName: prize.winner.playerLastName,
        })
      : '',
  }));
});

const currentPrize = computed(
  () => selectedPrizes.value.find((prize) => prize.id === currentPrizeId.value) || null,
);

const currentPrizeLabel = computed(() => {
  if (!currentPrize.value) {
    return '';
  }
  return currentPrize.value.name || `Premio ${currentPrize.value.position}`;
});

const lastWinnerDisplay = computed(() => {
  if (!lastWinner.value) {
    return '';
  }
  return formatTicketDisplay(lastWinner.value);
});

const currentDisplay = computed(() => {
  if (isDrawing.value && displayCode.value) {
    return displayCode.value;
  }
  if (currentPrize.value && currentPrize.value.winner && currentPrizeId.value === currentPrize.value.id) {
    return currentPrize.value.winnerDisplay;
  }
  if (lastWinner.value) {
    return lastWinnerDisplay.value;
  }
  return displayCode.value || 'Nessun ticket selezionato';
});

function eventLabel(event) {
  return `${teamName(event.team1_id)} - ${teamName(event.team2_id)}`;
}

function formatTicketDisplay(ticket) {
  const parts = [ticket.ticketCode];
  const fullName = [ticket.playerFirstName, ticket.playerLastName].filter(Boolean).join(' ');
  if (fullName) {
    parts.push(fullName);
  }
  return parts.join(' • ');
}

function normalizePrize(prize, index = 0) {
  if (!prize || typeof prize !== 'object') {
    return null;
  }
  const winner = prize.winner && typeof prize.winner === 'object' ? prize.winner : null;
  const normalizedWinner = winner
    ? {
        voteId: Number(winner.vote_id ?? winner.voteId) || 0,
        ticketCode: typeof (winner.ticket_code ?? winner.ticketCode) === 'string'
          ? (winner.ticket_code ?? winner.ticketCode)
          : '',
        playerId: Number(winner.player_id ?? winner.playerId) || 0,
        playerFirstName:
          typeof (winner.player_first_name ?? winner.playerFirstName) === 'string'
            ? (winner.player_first_name ?? winner.playerFirstName)
            : '',
        playerLastName:
          typeof (winner.player_last_name ?? winner.playerLastName) === 'string'
            ? (winner.player_last_name ?? winner.playerLastName)
            : '',
        assignedAt:
          typeof (winner.assigned_at ?? winner.assignedAt) === 'string'
            ? (winner.assigned_at ?? winner.assignedAt)
            : '',
      }
    : null;
  const position = Number(prize.position) || index + 1;
  return {
    id: Number(prize.id) || 0,
    eventId: Number(prize.event_id ?? prize.eventId) || 0,
    name: typeof prize.name === 'string' ? prize.name : '',
    position,
    winner: normalizedWinner,
  };
}

function normalizeEvent(event) {
  const normalized = {
    id: event.id,
    team1_id: event.team1_id,
    team2_id: event.team2_id,
    start_datetime: event.start_datetime,
    location: event.location,
  };
  const prizes = Array.isArray(event.prizes)
    ? event.prizes
        .map((prize, index) => normalizePrize(prize, index))
        .filter(Boolean)
        .sort((a, b) => {
          if (a.position === b.position) {
            return a.id - b.id;
          }
          return a.position - b.position;
        })
    : [];
  normalized.prizes = prizes;
  return normalized;
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
    currentPrizeId.value = 0;
    return;
  }
  const selectedEvent = events.value.find((event) => event.id === selectedEventId.value) || events.value[0];
  selectedEventId.value = selectedEvent.id;
  eventInput.value = eventLabel(selectedEvent);
  ensureCurrentPrize();
}

function ensureCurrentPrize() {
  const event = selectedEvent.value;
  if (!event || !Array.isArray(event.prizes) || event.prizes.length === 0) {
    currentPrizeId.value = 0;
    return;
  }
  const availablePrize = event.prizes.find((prize) => !prize.winner) || event.prizes[0];
  if (!availablePrize) {
    currentPrizeId.value = 0;
    return;
  }
  const currentExists = event.prizes.some((prize) => prize.id === currentPrizeId.value);
  if (!currentExists) {
    currentPrizeId.value = availablePrize.id;
    return;
  }
  const currentPrizeEntry = event.prizes.find((prize) => prize.id === currentPrizeId.value);
  if (currentPrizeEntry?.winner) {
    const nextAvailable = event.prizes.find((prize) => !prize.winner);
    if (nextAvailable) {
      currentPrizeId.value = nextAvailable.id;
    }
  }
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
  currentPrizeId.value = 0;
  lastWinner.value = null;
  displayCode.value = 'Seleziona un premio per iniziare';
  drawError.value = '';
  isResettingPrize.value = 0;
  clearTimers();
}

function logout() {
  token.value = '';
  activeUsername.value = '';
  localStorage.removeItem('adminToken');
  localStorage.removeItem('adminUsername');
  localStorage.removeItem('adminRole');
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
    localStorage.setItem('adminRole', data.role || '');
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
    ensureCurrentPrize();
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
    const { data } = await secureRequest(() => apiClient.get(`/events/${eventId}/tickets`, authHeaders.value));
    tickets.value = Array.isArray(data) ? data.map(normalizeTicket) : [];
    if (!tickets.value.length) {
      displayCode.value = 'Nessun ticket disponibile';
    } else if (!isDrawing.value) {
      displayCode.value = 'Seleziona un premio per iniziare';
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

function canDrawPrize(prize) {
  if (!prize || !prize.id) {
    return false;
  }
  if (prize.eventId && prize.eventId !== selectedEventId.value) {
    return false;
  }
  if (prize.winner) {
    return false;
  }
  if (isDrawing.value || isLoadingTickets.value) {
    return false;
  }
  return selectedEventId.value > 0 && tickets.value.length > 0;
}

function startDraw(prize) {
  if (!canDrawPrize(prize)) {
    if (!tickets.value.length) {
      drawError.value = 'Nessun ticket disponibile per questo evento.';
    }
    return;
  }
  drawError.value = '';
  lastWinner.value = null;
  currentPrizeId.value = prize.id;
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
    finalizePrizeDraw(prize, finalTicket);
  }, 3000);
}

async function finalizePrizeDraw(prize, ticket) {
  try {
    const updatedPrize = await assignPrizeWinner(prize, ticket);
    displayCode.value = formatTicketDisplay(ticket);
    lastWinner.value = ticket;
    tickets.value = tickets.value.filter((item) => item.voteId !== ticket.voteId);
    updateEventPrize(updatedPrize);
    ensureCurrentPrize();
  } catch (error) {
    if (error?.response?.status === 409) {
      drawError.value = 'Il premio risulta già assegnato. Aggiorna la pagina e riprova.';
    } else if (error?.response?.status === 400) {
      drawError.value = 'Ticket non valido per questo evento.';
    } else if (error?.response?.status !== 401) {
      drawError.value = "Impossibile completare l'estrazione. Riprova.";
    }
    await loadEvents();
    if (selectedEventId.value) {
      await loadTickets(selectedEventId.value);
    }
  } finally {
    isDrawing.value = false;
  }
}

async function assignPrizeWinner(prize, ticket) {
  const { data } = await secureRequest(() =>
    apiClient.post(
      `/events/${prize.eventId}/prizes/${prize.id}/assign`,
      { vote_id: ticket.voteId },
      authHeaders.value,
    ),
  );
  return normalizePrize(data, prize.position - 1 || 0);
}

function updateEventPrize(updatedPrize) {
  events.value = events.value.map((event) => {
    if (event.id !== updatedPrize.eventId) {
      return event;
    }
    const prizes = event.prizes
      .map((prize) => (prize.id === updatedPrize.id ? updatedPrize : prize))
      .sort((a, b) => {
        if (a.position === b.position) {
          return a.id - b.id;
        }
        return a.position - b.position;
      });
    return { ...event, prizes };
  });
}

async function resetPrize(prize) {
  if (!prize || !prize.id || isResettingPrize.value === prize.id) {
    return;
  }
  drawError.value = '';
  isResettingPrize.value = prize.id;
  try {
    await secureRequest(() =>
      apiClient.delete(`/events/${prize.eventId}/prizes/${prize.id}/winner`, authHeaders.value),
    );
    await loadEvents();
    if (selectedEventId.value) {
      await loadTickets(selectedEventId.value);
    }
    if (lastWinner.value && prize.winner && lastWinner.value.voteId === prize.winner.voteId) {
      lastWinner.value = null;
    }
    displayCode.value = 'Seleziona un premio per iniziare';
    currentPrizeId.value = prize.id;
    ensureCurrentPrize();
  } catch (error) {
    if (error?.response?.status === 404) {
      drawError.value = 'Premio non trovato o già ripristinato.';
    } else if (error?.response?.status !== 401) {
      drawError.value = 'Impossibile annullare il vincitore. Riprova.';
    }
  } finally {
    isResettingPrize.value = 0;
  }
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
    drawError.value = '';
    lastWinner.value = null;
    loadTickets(id);
    ensureCurrentPrize();
  } else {
    if (!eventInput.value) {
      eventInput.value = '';
    }
    tickets.value = [];
    currentPrizeId.value = 0;
    lastWinner.value = null;
    displayCode.value = 'Seleziona un premio per iniziare';
  }
});

watch(events, ensureValidSelection);

watch(selectedEvent, ensureCurrentPrize);

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

.draw-card__subtitle {
  margin: 0;
  font-weight: 600;
  color: #2563eb;
}

.prizes-wrapper {
  display: grid;
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.prize-card {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 1rem;
  border-radius: 1rem;
  background: rgba(241, 245, 249, 0.85);
  border: 1px solid rgba(148, 163, 184, 0.35);
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}

.prize-card.active {
  border-color: #6366f1;
  box-shadow: 0 10px 25px rgba(99, 102, 241, 0.15);
}

.prize-card__info {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.prize-card__info h3 {
  margin: 0;
}

.prize-card__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.no-prizes-message {
  margin: 0 0 1.5rem;
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
  .prizes-wrapper {
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  }

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
