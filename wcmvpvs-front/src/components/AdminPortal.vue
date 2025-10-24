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

      <section v-else-if="section === 'results'" class="card results-card">
        <header class="section-header">
          <h2>Risultati votazioni</h2>
          <p>Seleziona un evento per vedere la classifica MVP aggiornata in tempo reale.</p>
        </header>

        <div class="results-controls">
          <label>
            Evento
            <select v-model.number="selectedResultsEventId" :disabled="!events.length">
              <option disabled value="0">Seleziona un evento</option>
              <option v-for="event in events" :key="event.id" :value="event.id">
                {{ eventLabel(event) }}
              </option>
            </select>
          </label>
          <button
            class="btn secondary"
            type="button"
            @click="fetchEventResults({ showLoader: true })"
            :disabled="isLoadingResults || !selectedResultsEventId"
          >
            {{ isLoadingResults ? 'Aggiornamento…' : 'Aggiorna ora' }}
          </button>
        </div>

        <div v-if="selectedResultsEvent" class="results-summary">
          <h3>{{ selectedResultsEventLabel }}</h3>
          <p class="muted">{{ selectedResultsEventDate || 'Data da definire' }}</p>
        </div>

        <p v-if="resultsError" class="error">{{ resultsError }}</p>
        <div v-else-if="!events.length" class="info-banner">
          Crea un evento per visualizzare i risultati delle votazioni MVP.
        </div>
        <div v-else class="results-leaderboard">
          <div class="results-meta">
            <span><strong>Voti totali:</strong> {{ totalVotes }}</span>
            <span v-if="lastResultsUpdateLabel"><strong>Ultimo aggiornamento:</strong> {{ lastResultsUpdateLabel }}</span>
            <span class="auto-refresh">Aggiornamento automatico ogni 5 secondi</span>
          </div>
          <p v-if="isLoadingResults" class="muted">Caricamento risultati…</p>
          <p v-else-if="!hasResultsVotes" class="muted">Non ci sono ancora voti per questo evento.</p>
          <ul class="leaderboard-list" aria-live="polite">
            <li v-for="(entry, index) in resultsLeaderboard" :key="entry.id" class="leaderboard-item">
              <div class="rank">#{{ index + 1 }}</div>
              <div class="player-name">
                <span class="lastname">{{ entry.lastNameUpper }}</span>
                <span class="firstname">{{ entry.firstName }}</span>
              </div>
              <div class="votes">
                <strong>{{ entry.votes }}</strong>
                <span class="muted">{{ entry.votes === 1 ? 'voto' : 'voti' }}</span>
              </div>
              <div class="progress" role="presentation">
                <div class="progress-bar" :style="{ width: `${entry.percentage}%` }"></div>
              </div>
            </li>
          </ul>
        </div>
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

      <section v-else-if="section === 'sponsors'" class="card">
        <header class="section-header">
          <h2>Sponsor</h2>
          <p>Gestisci fino a {{ maxSponsors }} sponsor da mostrare nella schermata pubblica.</p>
        </header>

        <div class="sponsor-controls" role="group" aria-label="Visibilità sponsor">
          <label class="sponsor-range">
            <span>Numero di sponsor visibili: {{ desiredActiveSponsorCount }} / {{ maxSponsors }}</span>
            <input
              type="range"
              min="0"
              :max="sponsorSliderMax"
              v-model.number="desiredActiveSponsorCount"
              @change="applyActiveSponsorCount"
              :disabled="!sponsors.length || isApplyingSponsorCount"
            />
          </label>
          <p class="muted small">Gli sponsor attivi vengono mostrati nell'ordine indicato qui sotto.</p>
        </div>

        <form @submit.prevent="createSponsor" class="form-grid sponsor-form">
          <label>
            Nome sponsor
            <input v-model.trim="newSponsor.name" type="text" placeholder="Es. Partner ufficiale" required />
          </label>
          <label>
            Link (opzionale)
            <input v-model.trim="newSponsor.linkUrl" type="url" placeholder="https://example.com" />
          </label>
          <label class="file-input">
            Logo sponsor
            <input type="file" accept="image/*" @change="handleNewSponsorLogoChange" />
          </label>
          <div v-if="newSponsor.logoData" class="sponsor-preview new" aria-label="Anteprima logo nuovo sponsor">
            <img :src="newSponsor.logoData" alt="Anteprima logo sponsor" />
          </div>
          <button class="btn primary" type="submit" :disabled="isCreatingSponsor">
            {{ isCreatingSponsor ? 'Salvataggio…' : 'Aggiungi sponsor' }}
          </button>
        </form>

        <ul v-if="sponsors.length" class="item-list sponsors-list">
          <li v-for="sponsor in sponsors" :key="sponsor.id" class="item sponsor-item">
            <div class="item-body sponsor-body">
              <div class="sponsor-preview" :aria-label="`Logo sponsor ${sponsor.name || sponsor.position}`">
                <img
                  v-if="sponsor.logoData"
                  :src="sponsor.logoData"
                  :alt="`Logo ${sponsor.name || 'sponsor'}`"
                />
                <span v-else class="empty-logo">Logo non disponibile</span>
              </div>
              <div class="sponsor-fields">
                <div class="form-grid compact">
                  <label>
                    Nome sponsor
                    <input v-model.trim="sponsor.name" type="text" required />
                  </label>
                  <label>
                    Link (opzionale)
                    <input v-model.trim="sponsor.linkUrl" type="url" placeholder="https://example.com" />
                  </label>
                  <label class="file-input">
                    Aggiorna logo
                    <input type="file" accept="image/*" @change="(event) => handleSponsorLogoChange(event, sponsor)" />
                  </label>
                </div>
                <p class="muted sponsor-meta">
                  Posizione {{ sponsor.position }} • {{ sponsor.isActive ? 'Visibile' : 'Nascosto' }}
                </p>
              </div>
            </div>
            <div class="item-actions vertical">
              <button
                class="btn secondary"
                type="button"
                @click="updateSponsorEntry(sponsor)"
                :disabled="sponsorBeingUpdated === sponsor.id"
              >
                <span v-if="sponsorBeingUpdated === sponsor.id">Salvataggio…</span>
                <span v-else>Salva</span>
              </button>
              <button
                class="btn danger"
                type="button"
                @click="deleteSponsorEntry(sponsor.id)"
                :disabled="sponsorBeingDeleted === sponsor.id"
              >
                <span v-if="sponsorBeingDeleted === sponsor.id">Eliminazione…</span>
                <span v-else>Elimina</span>
              </button>
            </div>
          </li>
        </ul>
        <p v-else class="muted text-center">Nessuno sponsor configurato al momento.</p>
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
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue';
import { apiClient } from '../api';
import { roster } from '../roster';

const basePath = import.meta.env.BASE_URL ?? '/';
const baseVoteUrl = new URL(basePath, window.location.origin);
const RESULTS_POLL_INTERVAL = 5000;

let resultsPollHandle = 0;

const section = ref('events');
const tabs = [
  { id: 'events', label: 'Eventi' },
  { id: 'results', label: 'Risultati' },
  { id: 'teams', label: 'Squadre' },
  { id: 'players', label: 'Giocatori' },
  { id: 'sponsors', label: 'Sponsor' },
  { id: 'admins', label: 'Admin' },
];

const teams = ref([]);
const players = ref([]);
const events = ref([]);
const admins = ref([]);
const sponsors = ref([]);
const updatingEventId = ref(0);
const isDisablingEvents = ref(false);
const selectedResultsEventId = ref(0);
const eventResults = ref([]);
const isLoadingResults = ref(false);
const resultsError = ref('');
const lastResultsUpdate = ref(null);

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
const maxSponsors = 4;
const newSponsor = reactive({
  name: '',
  linkUrl: '',
  logoData: '',
  isActive: true,
});
const desiredActiveSponsorCount = ref(0);
const isCreatingSponsor = ref(false);
const sponsorBeingUpdated = ref(0);
const sponsorBeingDeleted = ref(0);
const isApplyingSponsorCount = ref(false);
const lastCreatedEventLink = ref('');

const hasEnoughTeams = computed(() => teams.value.length >= 2);
const activeEventId = computed(() => {
  const activeEvent = events.value.find((event) => event.is_active);
  return activeEvent ? activeEvent.id : 0;
});
const activeSponsorCount = computed(() => sponsors.value.filter((item) => item.isActive).length);
const sponsorSliderMax = computed(() =>
  sponsors.value.length ? Math.min(maxSponsors, sponsors.value.length) : maxSponsors,
);
const selectedResultsEvent = computed(() =>
  events.value.find((event) => event.id === selectedResultsEventId.value) || null,
);
const selectedResultsEventLabel = computed(() =>
  selectedResultsEvent.value ? eventLabel(selectedResultsEvent.value) : '',
);
const selectedResultsEventDate = computed(() =>
  selectedResultsEvent.value ? formatEventDate(selectedResultsEvent.value.start_datetime) : '',
);
const resultsLeaderboard = computed(() => {
  const aggregated = new Map(
    eventResults.value.map((item) => [
      Number(item.player_id) || 0,
      {
        votes: Number(item.votes) || 0,
        lastVoteAt: typeof item.last_vote_at === 'string' ? item.last_vote_at : '',
      },
    ]),
  );

  const entries = roster.map((player) => {
    const stats = aggregated.get(player.id) || { votes: 0, lastVoteAt: '' };
    const firstName = player.firstName || player.name.split(/\s+/)[0] || player.name || '';
    const remainingParts = player.lastName
      ? player.lastName
      : player.name.split(/\s+/).slice(1).join(' ');
    const lastName = remainingParts;
    const baseName = `${firstName}${lastName ? ` ${lastName}` : ''}`.trim();
    const fallback = baseName || player.name;
    const lastNameUpper = (lastName || firstName || player.name || '').toUpperCase();
    return {
      id: player.id,
      firstName: firstName || fallback,
      lastName,
      lastNameUpper,
      fullName: fallback,
      votes: stats.votes,
      lastVoteAt: stats.lastVoteAt,
    };
  });

  entries.sort((a, b) => {
    if (b.votes !== a.votes) {
      return b.votes - a.votes;
    }
    if (a.lastVoteAt && b.lastVoteAt && a.lastVoteAt !== b.lastVoteAt) {
      return a.lastVoteAt.localeCompare(b.lastVoteAt);
    }
    if (a.lastVoteAt && !b.lastVoteAt) {
      return -1;
    }
    if (!a.lastVoteAt && b.lastVoteAt) {
      return 1;
    }
    const lastNameComparison = a.lastName.localeCompare(b.lastName);
    if (lastNameComparison !== 0) {
      return lastNameComparison;
    }
    const firstNameComparison = a.firstName.localeCompare(b.firstName);
    if (firstNameComparison !== 0) {
      return firstNameComparison;
    }
    return a.id - b.id;
  });

  let highestVotes = 0;
  entries.forEach((entry) => {
    if (entry.votes > highestVotes) {
      highestVotes = entry.votes;
    }
  });

  return entries.map((entry) => ({
    ...entry,
    percentage: highestVotes > 0 ? Math.round((entry.votes / highestVotes) * 100) : 0,
  }));
});
const totalVotes = computed(() =>
  eventResults.value.reduce((sum, item) => sum + (Number(item.votes) || 0), 0),
);
const hasResultsVotes = computed(() => totalVotes.value > 0);
const lastResultsUpdateLabel = computed(() =>
  lastResultsUpdate.value ? lastResultsUpdate.value.toLocaleString('it-IT') : '',
);

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
  resetNewSponsorForm();
  desiredActiveSponsorCount.value = Math.min(sponsorSliderMax.value, activeSponsorCount.value);
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

watch(events, () => {
  ensureResultsSelection();
  if (section.value === 'results' && selectedResultsEventId.value) {
    fetchEventResults();
  }
});

function clearCollections() {
  teams.value = [];
  players.value = [];
  events.value = [];
  admins.value = [];
  sponsors.value = [];
  lastCreatedEventLink.value = '';
  resetResultsState();
}

function stopResultsPolling() {
  if (resultsPollHandle) {
    window.clearInterval(resultsPollHandle);
    resultsPollHandle = 0;
  }
}

function startResultsPolling() {
  stopResultsPolling();
  if (!selectedResultsEventId.value) {
    return;
  }
  resultsPollHandle = window.setInterval(() => {
    fetchEventResults().catch(() => {
      /* silent */
    });
  }, RESULTS_POLL_INTERVAL);
}

function resetResultsState() {
  stopResultsPolling();
  selectedResultsEventId.value = 0;
  eventResults.value = [];
  resultsError.value = '';
  lastResultsUpdate.value = null;
  isLoadingResults.value = false;
}

function ensureResultsSelection() {
  if (!events.value.length) {
    selectedResultsEventId.value = 0;
    return;
  }
  const exists = events.value.some((event) => event.id === selectedResultsEventId.value);
  if (!exists) {
    const active = events.value.find((event) => event.is_active);
    selectedResultsEventId.value = active ? active.id : events.value[0].id;
  }
}

async function fetchEventResults({ showLoader = false } = {}) {
  if (!selectedResultsEventId.value) {
    eventResults.value = [];
    resultsError.value = '';
    lastResultsUpdate.value = null;
    return;
  }
  if (showLoader) {
    isLoadingResults.value = true;
  }
  resultsError.value = '';
  try {
    const { data } = await secureRequest(() =>
      apiClient.get(`/events/${selectedResultsEventId.value}/results`, authHeaders.value),
    );
    if (Array.isArray(data)) {
      eventResults.value = data.map((item) => ({
        player_id: Number(item.player_id) || 0,
        votes: Number(item.votes) || 0,
        last_vote_at: typeof item.last_vote_at === 'string' ? item.last_vote_at : '',
      }));
    } else {
      eventResults.value = [];
    }
    lastResultsUpdate.value = new Date();
  } catch (error) {
    if (error?.response?.status === 404) {
      resultsError.value = 'Evento non trovato.';
    } else if (error?.response?.status === 400) {
      resultsError.value = 'Richiesta non valida per i risultati.';
    } else if (error?.response?.status !== 401) {
      resultsError.value = 'Impossibile caricare i risultati. Riprova più tardi.';
    }
  } finally {
    if (showLoader) {
      isLoadingResults.value = false;
    }
  }
}

function normalizeSponsorResponse(item) {
  if (!item || typeof item !== 'object') {
    return null;
  }
  const normalizedName = typeof item.name === 'string' ? item.name.trim() : '';
  const normalizedLink = typeof item.link_url === 'string' ? item.link_url.trim() : '';
  return {
    id: Number(item.id) || 0,
    name: normalizedName,
    linkUrl: normalizedLink,
    position: Number(item.position) || 0,
    logoData: typeof item.logo_data === 'string' ? item.logo_data : '',
    isActive: Boolean(item.is_active),
  };
}

function serializeSponsorPayload(sponsor) {
  return {
    name: sponsor.name.trim(),
    link_url: sponsor.linkUrl.trim(),
    position: sponsor.position,
    logo_data: sponsor.logoData,
    is_active: sponsor.isActive,
  };
}

function nextSponsorPosition() {
  const used = new Set(sponsors.value.map((item) => item.position));
  for (let index = 1; index <= maxSponsors; index += 1) {
    if (!used.has(index)) {
      return index;
    }
  }
  return Math.min(maxSponsors, sponsors.value.length + 1);
}

function sortedSponsors() {
  return [...sponsors.value].sort((a, b) => a.position - b.position);
}

function recomputeActiveSponsorSlider() {
  desiredActiveSponsorCount.value = Math.min(sponsorSliderMax.value, activeSponsorCount.value);
}

function resetNewSponsorForm() {
  Object.assign(newSponsor, { name: '', linkUrl: '', logoData: '', isActive: true });
}

async function readFileAsDataUrl(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      resolve(typeof reader.result === 'string' ? reader.result : '');
    };
    reader.onerror = () => {
      reject(reader.error || new Error('Impossibile leggere il file'));
    };
    reader.readAsDataURL(file);
  });
}

async function handleSponsorLogoChange(event, targetSponsor) {
  const [file] = event?.target?.files || [];
  if (!file) {
    return;
  }
  globalError.value = '';
  try {
    const dataUrl = await readFileAsDataUrl(file);
    if (dataUrl) {
      targetSponsor.logoData = dataUrl;
    }
  } catch (error) {
    console.error('Errore caricamento logo sponsor', error);
    globalError.value = 'Impossibile caricare il logo selezionato.';
  } finally {
    if (event?.target) {
      event.target.value = '';
    }
  }
}

async function handleNewSponsorLogoChange(event) {
  await handleSponsorLogoChange(event, newSponsor);
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

async function loadSponsors() {
  const { data } = await secureRequest(() => apiClient.get('/admin/sponsors', authHeaders.value));
  const normalized = Array.isArray(data)
    ? data
        .map((item) => normalizeSponsorResponse(item))
        .filter((item) => item && item.id)
        .sort((a, b) => a.position - b.position)
    : [];
  sponsors.value = normalized;
  recomputeActiveSponsorSlider();
}

async function loadAll() {
  if (!isAuthenticated.value) {
    return;
  }
  await Promise.all([loadTeams(), loadPlayers(), loadEvents(), loadAdmins(), loadSponsors()]);
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

async function createSponsor() {
  if (isCreatingSponsor.value) {
    return;
  }
  globalError.value = '';
  if (sponsors.value.length >= maxSponsors) {
    globalError.value = `Puoi configurare al massimo ${maxSponsors} sponsor.`;
    return;
  }
  const trimmedName = newSponsor.name.trim();
  if (!trimmedName) {
    globalError.value = 'Inserisci il nome dello sponsor.';
    return;
  }
  if (!newSponsor.logoData) {
    globalError.value = 'Carica un logo per lo sponsor.';
    return;
  }
  const payload = serializeSponsorPayload({
    name: trimmedName,
    linkUrl: newSponsor.linkUrl,
    logoData: newSponsor.logoData,
    position: nextSponsorPosition(),
    isActive: false,
  });
  isCreatingSponsor.value = true;
  try {
    await secureRequest(() => apiClient.post('/admin/sponsors', payload, authHeaders.value));
    resetNewSponsorForm();
    await loadSponsors();
  } catch (error) {
    if (error?.response?.status === 400) {
      globalError.value = 'Controlla i dati inseriti: sono disponibili massimo 4 sponsor.';
    }
  } finally {
    isCreatingSponsor.value = false;
  }
}

async function updateSponsorEntry(sponsor) {
  if (sponsorBeingUpdated.value === sponsor.id) {
    return;
  }
  globalError.value = '';
  const trimmedName = sponsor.name.trim();
  if (!trimmedName) {
    globalError.value = 'Inserisci il nome dello sponsor.';
    return;
  }
  if (!sponsor.logoData) {
    globalError.value = 'Carica un logo per lo sponsor.';
    return;
  }
  sponsorBeingUpdated.value = sponsor.id;
  try {
    const payload = serializeSponsorPayload({
      name: trimmedName,
      linkUrl: sponsor.linkUrl,
      logoData: sponsor.logoData,
      position: sponsor.position,
      isActive: sponsor.isActive,
    });
    await secureRequest(() => apiClient.put(`/admin/sponsors/${sponsor.id}`, payload, authHeaders.value));
    await loadSponsors();
  } catch (error) {
    if (error?.response?.status === 400) {
      globalError.value = 'Controlla i dati dello sponsor e riprova.';
    } else if (error?.response?.status === 404) {
      globalError.value = 'Sponsor non trovato. Aggiorna la pagina.';
    }
  } finally {
    sponsorBeingUpdated.value = 0;
  }
}

async function deleteSponsorEntry(id) {
  if (sponsorBeingDeleted.value === id) {
    return;
  }
  globalError.value = '';
  sponsorBeingDeleted.value = id;
  try {
    await secureRequest(() => apiClient.delete(`/admin/sponsors/${id}`, authHeaders.value));
    await loadSponsors();
  } catch (error) {
    if (error?.response?.status === 404) {
      globalError.value = 'Sponsor già rimosso.';
    }
  } finally {
    sponsorBeingDeleted.value = 0;
  }
}

async function applyActiveSponsorCount() {
  if (isApplyingSponsorCount.value) {
    return;
  }
  if (!sponsors.value.length) {
    desiredActiveSponsorCount.value = 0;
    return;
  }
  globalError.value = '';
  const target = Math.max(0, Math.min(maxSponsors, desiredActiveSponsorCount.value));
  isApplyingSponsorCount.value = true;
  try {
    const updates = [];
    sortedSponsors().forEach((sponsor, index) => {
      const shouldBeActive = index < target;
      if (sponsor.isActive !== shouldBeActive) {
        const payload = serializeSponsorPayload({
          name: sponsor.name.trim(),
          linkUrl: sponsor.linkUrl,
          logoData: sponsor.logoData,
          position: sponsor.position,
          isActive: shouldBeActive,
        });
        updates.push(
          secureRequest(() =>
            apiClient.put(`/admin/sponsors/${sponsor.id}`, payload, authHeaders.value),
          ),
        );
      }
    });
    if (updates.length) {
      await Promise.all(updates);
    }
    await loadSponsors();
  } catch (error) {
    if (error?.response?.status === 400) {
      globalError.value = 'Impossibile aggiornare il numero di sponsor visibili. Verifica i dati e riprova.';
    }
  } finally {
    isApplyingSponsorCount.value = false;
  }
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

watch(section, (value, oldValue) => {
  if (value === 'results') {
    ensureResultsSelection();
    fetchEventResults({ showLoader: true });
    startResultsPolling();
  } else if (oldValue === 'results') {
    stopResultsPolling();
  }
});

watch(selectedResultsEventId, (eventId) => {
  if (section.value === 'results' && eventId) {
    fetchEventResults({ showLoader: true });
    startResultsPolling();
  } else if (!eventId) {
    stopResultsPolling();
  }
});

if (isAuthenticated.value) {
  loadAll();
}

onBeforeUnmount(() => {
  stopResultsPolling();
});
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

.muted.small {
  font-size: 0.8rem;
}

.text-center {
  text-align: center;
}

.sponsor-controls {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  margin-bottom: 1.5rem;
}

.sponsor-range {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  font-weight: 600;
  color: #1e293b;
}

.sponsor-range input[type='range'] {
  accent-color: #2563eb;
}

.sponsor-form {
  align-items: flex-end;
}

.sponsor-preview {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 0.85rem;
  border: 1px dashed rgba(148, 163, 184, 0.6);
  background: rgba(241, 245, 249, 0.6);
  overflow: hidden;
  min-height: 120px;
}

.sponsor-preview.new {
  min-height: 100px;
}

.sponsor-preview img {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
}

.empty-logo {
  font-size: 0.85rem;
  color: #94a3b8;
}

.sponsors-list {
  margin-top: 1.5rem;
}

.sponsor-item {
  gap: 1.25rem;
}

.sponsor-body {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

@media (min-width: 768px) {
  .sponsor-body {
    flex-direction: row;
    align-items: center;
  }

  .sponsor-preview {
    flex: 0 0 220px;
    min-height: 140px;
  }
}

.sponsor-fields {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.form-grid.compact {
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  margin-bottom: 0.75rem;
}

.sponsor-meta {
  font-size: 0.85rem;
}

.item-actions.vertical {
  flex-direction: column;
  align-items: stretch;
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

.results-card {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.results-controls {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  align-items: flex-end;
}

.results-controls label {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  font-weight: 600;
  color: #1e293b;
}

.results-summary h3 {
  margin: 0;
  font-size: 1.25rem;
}

.results-summary .muted {
  margin: 0.25rem 0 0;
}

.results-leaderboard {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.results-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  font-size: 0.95rem;
  color: #475569;
}

.results-meta .auto-refresh {
  font-size: 0.85rem;
  color: #64748b;
}

.leaderboard-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.leaderboard-item {
  display: grid;
  grid-template-columns: 70px minmax(0, 1fr) 120px;
  gap: 1rem;
  align-items: center;
  padding: 0.85rem 1rem;
  border-radius: 1rem;
  background: linear-gradient(135deg, rgba(15, 23, 42, 0.85), rgba(30, 64, 175, 0.9));
  color: #f8fafc;
  box-shadow: 0 12px 24px rgba(15, 23, 42, 0.3);
}

.leaderboard-item .rank {
  font-size: 1.5rem;
  font-weight: 700;
  text-align: center;
}

.leaderboard-item .player-name {
  display: flex;
  flex-direction: column;
  line-height: 1.2;
}

.leaderboard-item .player-name .lastname {
  font-size: 1.2rem;
  letter-spacing: 0.08em;
}

.leaderboard-item .player-name .firstname {
  font-size: 0.95rem;
  text-transform: capitalize;
  opacity: 0.9;
}

.leaderboard-item .votes {
  display: flex;
  align-items: baseline;
  gap: 0.35rem;
  font-size: 1rem;
  justify-content: flex-end;
}

.leaderboard-item .votes strong {
  font-size: 1.4rem;
}

.leaderboard-item .progress {
  grid-column: 1 / -1;
  height: 6px;
  border-radius: 999px;
  background: rgba(148, 163, 184, 0.35);
  overflow: hidden;
}

.leaderboard-item .progress-bar {
  height: 100%;
  background: linear-gradient(135deg, #facc15, #f97316);
  border-radius: inherit;
  transition: width 0.4s ease;
}

@media (max-width: 640px) {
  .leaderboard-item {
    grid-template-columns: 56px minmax(0, 1fr);
  }

  .leaderboard-item .votes {
    grid-column: 1 / -1;
    justify-content: flex-start;
  }
}
</style>
