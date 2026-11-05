<template>
  <div v-if="!isStaff" class="lottery-shell admin-portal">
    <header class="lottery-header">
      <div>
        <p class="lottery-kicker">ArenaBoostX <span aria-hidden="true">/</span> Coinvolgimento fan</p>
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
      <p v-if="loginError" class="error" role="alert">{{ loginError }}</p>
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

      <p v-if="isLoadingEvents" class="loading-message" role="status">Caricamento degli eventi…</p>
      <p v-if="isLoadingTickets" class="loading-message" role="status">Aggiornamento dei ticket…</p>
      <p v-if="globalError" class="error" role="alert">{{ globalError }}</p>
      <p v-if="drawError" class="error" role="alert">{{ drawError }}</p>

      <div v-if="selectedEventId" class="lottery-status">
        <p class="muted">
          Codici ammessi: <strong>{{ allowedCodesCount }}</strong><span v-if="totalCodesCount > 0"> / Totali: <strong>{{ totalCodesCount }}</strong></span>
        </p>
        <p v-if="lastWinner" class="winner-banner" role="status">
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
              <span v-if="prize.winner.status"> · SMS: {{ smsStatusLabel(prize.winner.status) }}</span>
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

      <div v-if="selectedPrizes.length" class="draw-card" :class="{ active: isDrawing }" :aria-busy="isDrawing">
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

const currentPath = ref(typeof window !== 'undefined' ? window.location.pathname : '/');
const pathSegments = computed(() =>
  currentPath.value
    .split('/')
    .map((part) => part.trim())
    .filter(Boolean),
);

const organizationSlug = computed(() => {
  if (!pathSegments.value.length) {
    return '';
  }

  if (pathSegments.value[0] === 'admin') {
    return '';
  }

  return pathSegments.value[0];
});

const basePath = computed(() => {
  if (organizationSlug.value) {
    return `/${organizationSlug.value}`;
  }
  return (import.meta.env.BASE_URL ?? '/').replace(/\/+$/, '');
});

const teams = ref([]);
const events = ref([]);
const tickets = ref([]);
const totalCodesCount = ref(0);
const allowedCodesCount = computed(() => tickets.value.length);
const selectedEventId = ref(0);
const eventInput = ref('');

const activeUsername = ref(localStorage.getItem('adminUsername') || '');
const activeRole = ref((localStorage.getItem('adminRole') || '').trim().toLowerCase());
const isStaff = computed(() => activeRole.value === 'staff');
const isAuthenticated = computed(() => Boolean(activeUsername.value));

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

const authHeaders = computed(() => ({ headers: {} }));

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

function maskPhone(phone) {
  const value = String(phone || "").trim();
  if (!value) return "";
  if (value.length <= 4) return value;
  return `${value.slice(0, 3)}***${value.slice(-2)}`;
}

function formatTicketDisplay(ticket) {
  const code = ticket.ticketCode || "";
  const nickname = String(ticket.nickname || "").trim();
  const maskedPhone = maskPhone(ticket.phone);
  if (nickname) return `${code} · ${nickname}`;
  if (maskedPhone) return `${code} · ${maskedPhone}`;
  return code;
}

function smsStatusLabel(status) {
  switch (String(status || "").trim()) {
    case "notified": return "inviato";
    case "notified_failed": return "errore";
    case "assigned": return "non inviato";
    default: return "non inviato";
  }
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
        nickname: typeof (winner.nickname ?? winner.Nickname) === 'string'
          ? (winner.nickname ?? winner.Nickname)
          : '',
        phone: typeof (winner.phone ?? winner.Phone) === 'string'
          ? (winner.phone ?? winner.Phone)
          : '',
        status: typeof (winner.status ?? winner.Status) === 'string'
          ? (winner.status ?? winner.Status)
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
    isConcluded: Boolean(event.is_concluded ?? event.isConcluded),
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

function parseEventDate(event) {
  const date = event?.start_datetime ? new Date(event.start_datetime) : null;
  return date && !Number.isNaN(date.getTime()) ? date : null;
}

function findLatestCompletedEvent(eventList) {
  if (!Array.isArray(eventList) || eventList.length === 0) {
    return null;
  }

  const now = new Date();
  let latestPastEvent = null;
  let latestPastDate = null;
  let latestOverallEvent = null;
  let latestOverallDate = null;

  eventList.forEach((event) => {
    const eventDate = parseEventDate(event);
    if (!eventDate) {
      return;
    }

    if (!latestOverallDate || eventDate > latestOverallDate) {
      latestOverallEvent = event;
      latestOverallDate = eventDate;
    }

    if (eventDate <= now && (!latestPastDate || eventDate > latestPastDate)) {
      latestPastEvent = event;
      latestPastDate = eventDate;
    }
  });

  return latestPastEvent || latestOverallEvent;
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
  const existingSelection = events.value.find((event) => event.id === selectedEventId.value);
  const preferredEvent = existingSelection || findLatestCompletedEvent(events.value) || events.value[0];

  selectedEventId.value = preferredEvent.id;
  eventInput.value = eventLabel(preferredEvent);
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
  totalCodesCount.value = 0;
  selectedEventId.value = 0;
  currentPrizeId.value = 0;
  lastWinner.value = null;
  displayCode.value = 'Seleziona un premio per iniziare';
  drawError.value = '';
  isResettingPrize.value = 0;
  clearTimers();
}

async function logout() {
  try { await apiClient.post('/admin/logout'); } catch (_) { /* ignora */ }
  activeUsername.value = '';
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
    activeRole.value = (data.role || '').trim().toLowerCase();
    activeUsername.value = data.username;
    localStorage.setItem('adminUsername', activeUsername.value);
    localStorage.setItem('adminRole', activeRole.value);
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
    const normalizedEvents = Array.isArray(data)
      ? data.map(normalizeEvent).filter((event) => !event.isConcluded)
      : [];
    events.value = normalizedEvents;
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
  if (isStaff.value) { goToPortal(); return; }
  try {
    const { data } = await secureRequest(() => apiClient.get('/admin/me', authHeaders.value));
    activeRole.value = (data.role || '').trim().toLowerCase();
    localStorage.setItem('adminRole', activeRole.value);
    if (isStaff.value) { goToPortal(); return; }
  } catch (_) { return; }
  await Promise.all([loadTeams(), loadEvents()]);
}

async function loadTickets(eventId) {
  if (!eventId) {
    tickets.value = [];
    totalCodesCount.value = 0;
    return;
  }
  isLoadingTickets.value = true;
  globalError.value = '';
  drawError.value = '';
  try {
    const response = await secureRequest(() => apiClient.get(`/events/${eventId}/tickets`, authHeaders.value));
    const { data, headers } = response;
    const total = Number(headers?.['x-lottery-total-codes'] || 0);
    totalCodesCount.value = Number.isFinite(total) ? total : 0;
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
      drawError.value = error?.response?.data?.message || 'Il premio risulta già assegnato. Aggiorna la pagina e riprova.';
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
    } else if (error?.response?.status === 409) {
      drawError.value = error?.response?.data?.message || 'I premi della chiusura automatica sono definitivi.';
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
  const target = new URL(basePath.value || '/', window.location.origin);
  if (!target.pathname.endsWith('/')) {
    target.pathname = `${target.pathname}/`;
  }
  target.pathname = `${target.pathname.replace(/\/+$/, '')}/admin`;
  if (isStaff.value) window.location.replace(target.toString());
  else window.location.href = target.toString();
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
.lottery-shell.admin-portal {
  min-height: 100vh;
  min-height: 100dvh;
  width: 100%;
  padding: 0 32px 48px;
  display: flex;
  flex-direction: column;
  gap: 28px;
  background: var(--bg-base, #f6f7fb);
  color: var(--text-primary, #182238);
  font-family: var(--font-body, 'Segoe UI', sans-serif);
}
.lottery-header { width: min(1120px,100%); margin: 0 auto; padding: 32px 0 28px; display: flex; align-items: flex-start; justify-content: space-between; flex-wrap: wrap; gap: 24px; border-bottom: 1px solid var(--border, #e3e7ef); }
.lottery-kicker { display: flex; align-items: center; flex-wrap: wrap; gap: 10px; color: var(--text-secondary, #526078); font-size: 12px; font-weight: 600; margin: 0 0 15px; }
.lottery-kicker span { color: #9aa4b6; font-weight: 400; }
.lottery-header h1 { font-size: clamp(26px,3vw,32px); letter-spacing: -.04em; font-weight: 700; line-height: 1.2; margin: 0 0 9px; }
.subtitle { color: var(--text-secondary, #526078); margin: 0; font-size: 14px; line-height: 1.6; }
.header-actions { display: flex; align-items: center; justify-content: flex-end; flex-wrap: wrap; gap: 10px; }
.header-actions > .muted { width: 100%; text-align: right; font-size: 12px; margin-bottom: 4px; }
.header-actions .btn.ghost { border: 1px solid #cbd3e1; background: #fff; color: #3c4960; }
.header-actions .btn.ghost:hover { background: #f3f2ff; border-color: #a5a0ed; color: #4338ca; }
.lottery-card { display: flex; flex-direction: column; gap: 24px; width: min(1120px,100%); margin: 0 auto; }
.card { padding: 28px; background: #fff; border: 1px solid var(--border, #e3e7ef); border-radius: var(--radius-lg,14px); box-shadow: var(--shadow-card,0 2px 5px rgba(24,34,56,.025)); }
.lottery-shell .login-card { margin: 20px auto 0; width: min(460px,100%); padding: 32px; }
.login-card h2 { margin: 0 0 20px; font-size: 20px; }
.form-grid { display: grid; gap: 20px; }
label { display: flex; flex-direction: column; gap: 8px; font-size: 13px; font-weight: 600; color: #3c4960; }
input, select { box-sizing: border-box; width: 100%; min-width: 0; min-height: var(--control-height,42px); border: 1px solid #cbd3e1; border-radius: 8px; padding: 10px 12px; background: #fff; color: var(--text-primary,#182238); font: 400 14px/1.45 var(--font-body,'Segoe UI',sans-serif); color-scheme: light; }
input::placeholder { color: #66738a; opacity: 1; }
input:disabled, select:disabled { background: #f1f3f7; color: #606c80; opacity: 1; cursor: not-allowed; }
input:focus-visible, select:focus-visible, button:focus-visible { outline: 3px solid var(--accent,#4f46e5); outline-offset: 3px; }
select option { background: #fff; color: #182238; }
.btn { display: inline-flex; align-items: center; justify-content: center; gap: 8px; min-height: 42px; padding: 10px 15px; border: 1px solid transparent; border-radius: 8px; font: 600 13px/1.4 var(--font-body,'Segoe UI',sans-serif); text-align: center; cursor: pointer; transition: background .15s,border-color .15s; }
.btn.primary { background: var(--accent,#4f46e5); border-color: var(--accent,#4f46e5); color: #fff; }
.btn.primary:not(:disabled):hover { background: var(--abx-primary-hover,#4338ca); border-color: var(--abx-primary-hover,#4338ca); }
.btn.secondary, .btn.outline { background: #fff; color: #3c4960; border-color: #cbd3e1; }
.btn:is(.secondary,.outline):not(:disabled):hover { background: #f5f4ff; color: #4338ca; border-color: #a5a0ed; }
.btn:disabled { cursor: not-allowed; opacity: .58; }
.filters { display: flex; align-items: flex-end; flex-wrap: wrap; gap: 16px; padding: 20px; border: 1px solid var(--border,#e3e7ef); border-radius: 10px; background: #f8f9fc; }
.filters label { flex: 1 1 260px; min-width: 0; }
.lottery-status { display: flex; flex-direction: column; gap: 12px; padding-bottom: 20px; border-bottom: 1px solid var(--border,#e3e7ef); }
.lottery-status > .muted { font-size: 13px; }
.lottery-status strong { color: var(--text-primary,#182238); }
.prizes-wrapper { display: grid; grid-template-columns: repeat(auto-fit,minmax(min(100%,280px),1fr)); gap: 16px; }
.prize-card { display: flex; flex-direction: column; justify-content: space-between; gap: 24px; min-width: 0; padding: 22px; border: 1px solid var(--border,#e3e7ef); border-radius: 12px; background: #fff; transition: border-color .15s,box-shadow .15s; }
.prize-card.active { border-color: #a5a0ed; background: #fcfbff; box-shadow: 0 0 0 2px #eeedff; }
.prize-card__info { display: flex; flex-direction: column; gap: 7px; min-width: 0; }
.prize-card__info h3 { margin: 0; font-size: 17px; font-weight: 650; line-height: 1.4; overflow-wrap: anywhere; }
.prize-card__info > .muted { font-size: 12px; }
.prize-card__actions { display: flex; align-items: stretch; flex-wrap: wrap; gap: 10px; }
.prize-card__actions > .btn { flex: 1 1 150px; }
.no-prizes-message { padding: 20px; background: #f8f9fc; border: 1px dashed #cbd3e1; border-radius: 10px; }
.draw-card { display: flex; flex-direction: column; gap: 18px; align-items: center; text-align: center; padding: 28px; border: 1px solid #dedbff; border-radius: 12px; background: #f8f7ff; }
.draw-card__subtitle { margin: 0; color: #4338ca; font-size: 13px; }
.draw-card .display { width: min(100%,680px); display: flex; align-items: center; justify-content: center; min-height: 140px; padding: 30px 24px; border: 1px solid #dedbff; border-radius: 12px; background: #fff; color: #312e81; font-size: clamp(22px,3.5vw,36px); font-weight: 700; letter-spacing: -.02em; line-height: 1.4; overflow-wrap: anywhere; }
.draw-card.active .display { animation: draw-pulse .8s ease-in-out infinite alternate; }
.draw-card > .muted { font-size: 13px; }
.winner-banner { padding: 12px 14px; margin: 0; border: 1px solid #bcebd6; border-radius: 8px; background: #ecfdf5; color: #087a55; font-size: 13px; font-weight: 500; line-height: 1.6; overflow-wrap: anywhere; }
.winner-banner strong { color: #066244; }
.muted { margin: 0; color: var(--text-secondary,#526078); }
.error { margin: 0; padding: 12px 16px; border: 1px solid #fecdd3; border-radius: 8px; background: #fff1f2; color: var(--abx-danger,#b42335); font-size: 13px; }
.loading-message { margin: 0; padding: 12px 16px; border-radius: 8px; background: #f3f2ff; color: #4338ca; font-size: 13px; }
@keyframes draw-pulse { from { border-color: #a5a0ed; box-shadow: 0 0 0 0 rgba(79,70,229,.06); } to { border-color: #4f46e5; box-shadow: 0 0 0 4px rgba(79,70,229,.08); } }
@media(max-width:640px) {
  .lottery-shell.admin-portal { padding: 0 16px 32px; gap: 20px; }
  .lottery-header { padding: 24px 0 20px; gap: 20px; }
  .header-actions { justify-content: flex-start; width: 100%; }
  .header-actions > .muted { text-align: left; }
  .lottery-card { padding: 18px; gap: 20px; }
  .filters { padding: 16px; gap: 14px; }
  .filters label, .filters > .btn { flex: 1 1 100%; }
  .btn { min-height: 44px; }
  input, select { min-height: 44px; font-size: 16px; }
  .draw-card { padding: 18px; }
  .draw-card .display { min-height: 120px; padding: 24px 16px; }
  .prize-card { padding: 18px; }
  .lottery-shell .login-card { padding: 24px; }
}
@media(prefers-reduced-motion:reduce) { .draw-card.active .display { animation: none; } }
</style>
