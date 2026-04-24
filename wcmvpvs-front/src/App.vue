<template>
  <div class="app-shell">
    <AdminLottery v-if="appView === 'lottery'" />
    <MasterPortal v-else-if="appView === 'master'" />
    <AdminPortal v-else-if="appView === 'portal'" :organization-slug="organizationSlug" />
    <TicketValidationView v-else-if="appView === 'ticket-validation'" />
    <CashLanding v-else-if="appView === 'landing'" />
    <PartnerPortal
      v-else-if="appView === 'partner'"
      :current-path="currentPath"
      :current-search="currentSearch"
    />
    <template v-else-if="appView === 'newui'">
      <LiveExperienceHome
        :event-id="resolvedEventId"
        :team-name="newUiTeamName"
        :team-logo-url="newUiTeamLogoUrl"
        :match-label="newUiMatchLabel"
        :voted-player-image-url="newUiSelectedPlayerImageUrl"
        :voted-player-name="newUiSelectedPlayerName"
        :voted-player-last-name="newUiSelectedPlayerLastName"
        :voted-player-number="newUiSelectedPlayerNumber"
        :registration-prompt-signal="newUiRegistrationPromptSignal"
        :active-event="activeEvent"
        @feature-select="handleNewUiFeatureSelect"
      />
      <NewUiVoteModal
        v-if="showNewUiVoteModal"
        :event-id="resolvedEventId"
        @close="showNewUiVoteModal = false"
        @voted="handleNewUiPlayerVoted"
      />
    </template>
    <DevStackItDemo v-else-if="appView === 'dev-stack-it'" />
    <VoteScreen
      v-else
      :event-id="resolvedEventId"
      :active-event="activeEvent"
      :active-event-checked="hasCheckedActiveEvent"
      :loading-active-event="isFetchingActiveEvent"
    />

  </div>
</template>

<script setup>
import { computed, defineAsyncComponent, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import VoteScreen from './components/VoteScreen.vue';
import LiveExperienceHome from './views/LiveExperienceHome.vue';
import NewUiVoteModal from './components/NewUiVoteModal.vue';

// Dev playground components
const DevStackItDemo = defineAsyncComponent(() => import('./views/DevStackItDemo.vue'));

// Admin components loaded only when the URL matches an admin route
const AdminPortal = defineAsyncComponent(() => import('./components/AdminPortal.vue'));
const MasterPortal = defineAsyncComponent(() => import('./components/MasterPortal.vue'));
const AdminLottery = defineAsyncComponent(() => import('./components/AdminLottery.vue'));
const PartnerPortal = defineAsyncComponent(() => import('./components/PartnerPortal.vue'));
const TicketValidationView = defineAsyncComponent(() => import('./components/TicketValidationView.vue'));
const CashLanding = defineAsyncComponent(() => import('./components/CashLanding.vue'));
import { apiClient, fetchVoteStatus } from './api';

function readEventId(search) {
  const params = new URLSearchParams(search || '');
  const raw = params.get('eventId') ?? params.get('eventID');
  const parsed = Number.parseInt(raw ?? '', 10);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : undefined;
}

const currentPath = ref(typeof window !== 'undefined' ? window.location.pathname : '/');
const currentSearch = ref(typeof window !== 'undefined' ? window.location.search : '');
const currentEventId = ref(typeof window !== 'undefined' ? readEventId(window.location.search) : undefined);
const activeEvent = ref(null);
const isFetchingActiveEvent = ref(false);
const hasCheckedActiveEvent = ref(false);

const pathSegments = computed(() =>
  currentPath.value
    .split('/')
    .map((part) => part.trim())
    .filter(Boolean),
);

const isNewUiPath = computed(() => {
  const segments = pathSegments.value;
  if (!segments.length) {
    return false;
  }
  return segments[0] === 'newui' || segments[segments.length - 1] === 'newui';
});

const organizationSlug = computed(() => {
  if (pathSegments.value.length) {
    if (isNewUiPath.value) {
      if (pathSegments.value[0] === 'newui') {
        return pathSegments.value[1] ?? '';
      }
      if (pathSegments.value[pathSegments.value.length - 1] === 'newui') {
        return pathSegments.value[0] ?? '';
      }
    }

    if (
      pathSegments.value[0] !== 'admin' &&
      pathSegments.value[0] !== 'partner'
    ) {
      return pathSegments.value[0];
    }
  }

  const params = new URLSearchParams(currentSearch.value || '');
  const fromQuery =
    params.get('organization_slug') || params.get('org') || params.get('organization') || '';
  return fromQuery.trim();
});

const appView = computed(() => {
  if (currentPath.value === '/dev/stack-it-demo') {
    return 'dev-stack-it';
  }
  if (currentPath.value.startsWith('/admin/master')) {
    return 'master';
  }
  const hasOrganizationLotteryRoute =
    pathSegments.value.length >= 3 &&
    pathSegments.value[1] === 'admin' &&
    pathSegments.value[2] === 'lottery';
  if (currentPath.value.startsWith('/admin/lottery') || hasOrganizationLotteryRoute) {
    return 'lottery';
  }
  if (pathSegments.value.length >= 2 && pathSegments.value[1] === 'admin') {
    return 'portal';
  }
  if (currentPath.value.startsWith('/admin')) {
    return 'portal';
  }
  if (currentPath.value.startsWith('/lottery/validate')) {
    return 'ticket-validation';
  }
  if (currentPath.value.startsWith('/welcome')) {
    return 'landing';
  }
  if (currentPath.value.includes('/partner')) {
    return 'partner';
  }
  if (isNewUiPath.value) {
    return 'newui';
  }
  return 'public';
});

const resolvedEventId = computed(() => currentEventId.value ?? activeEvent.value?.id);
const newUiTeamName = computed(() => {
  const organizationName = String(activeEvent.value?.organization_name || '').trim();
  if (organizationName) {
    return organizationName;
  }

  const fallbackName = String(activeEvent.value?.team1_name || '').trim();
  return fallbackName || 'TEAM';
});
const newUiTeamLogoUrl = computed(() => {
  const directLogo = String(
    activeEvent.value?.organization_logo_url ||
      activeEvent.value?.logo_url ||
      activeEvent.value?.organization?.logo_url ||
      '',
  ).trim();

  return directLogo;
});
const newUiMatchLabel = computed(() => {
  const homeTeamName = String(activeEvent.value?.team1_name || '').trim();
  const awayTeamName = String(activeEvent.value?.team2_name || '').trim();

  if (homeTeamName && awayTeamName) {
    return `${homeTeamName} - ${awayTeamName}`;
  }

  return 'Vota • Gioca • Vinci • Partecipa';
});

const showNewUiVoteModal = ref(false);
const newUiSelectedPlayer = ref(null);
const newUiRegistrationPromptSignal = ref(0);

const NEW_UI_VOTE_STORAGE_KEY = 'newui:voted-player';

function sanitizeStoredPlayer(candidate) {
  if (!candidate || typeof candidate !== 'object') {
    return null;
  }

  const id = Number(candidate.id);
  if (!Number.isFinite(id) || id <= 0) {
    return null;
  }

  return {
    id,
    name: typeof candidate.name === 'string' ? candidate.name : '',
    lastName: typeof candidate.lastName === 'string' ? candidate.lastName : '',
    number: candidate.number == null ? '' : String(candidate.number),
    avatar: typeof candidate.avatar === 'string' ? candidate.avatar : '',
  };
}

function readStoredVotedPlayer(eventId) {
  if (typeof window === 'undefined' || !eventId) {
    return null;
  }

  try {
    const raw = window.localStorage.getItem(`${NEW_UI_VOTE_STORAGE_KEY}:${eventId}`);
    if (!raw) {
      return null;
    }
    return sanitizeStoredPlayer(JSON.parse(raw));
  } catch (error) {
    return null;
  }
}

function writeStoredVotedPlayer(eventId, player) {
  if (typeof window === 'undefined' || !eventId) {
    return;
  }

  if (!player) {
    window.localStorage.removeItem(`${NEW_UI_VOTE_STORAGE_KEY}:${eventId}`);
    return;
  }

  try {
    window.localStorage.setItem(`${NEW_UI_VOTE_STORAGE_KEY}:${eventId}`, JSON.stringify(player));
  } catch (error) {
    // no-op: localStorage may be unavailable on some browsers
  }
}

const newUiSelectedPlayerImageUrl = computed(() =>
  typeof newUiSelectedPlayer.value?.avatar === 'string' ? newUiSelectedPlayer.value.avatar : '',
);
const newUiSelectedPlayerName = computed(() =>
  typeof newUiSelectedPlayer.value?.name === 'string' ? newUiSelectedPlayer.value.name : '',
);
const newUiSelectedPlayerLastName = computed(() =>
  typeof newUiSelectedPlayer.value?.lastName === 'string' ? newUiSelectedPlayer.value.lastName : '',
);
const newUiSelectedPlayerNumber = computed(() =>
  newUiSelectedPlayer.value?.number == null ? '' : String(newUiSelectedPlayer.value.number),
);

function handlePopState() {
  currentPath.value = window.location.pathname;
  currentSearch.value = window.location.search;
  currentEventId.value = readEventId(window.location.search);

  if (appView.value === 'public') {
    fetchActiveEvent();
  }
}

function handleNewUiFeatureSelect(featureId) {
  if (featureId === 'vote-mvp') {
    showNewUiVoteModal.value = true;
  }
}

function handleNewUiPlayerVoted(player) {
  if (!player) {
    return;
  }
  newUiSelectedPlayer.value = sanitizeStoredPlayer(player) || player;
  writeStoredVotedPlayer(resolvedEventId.value, newUiSelectedPlayer.value);
  showNewUiVoteModal.value = false;
  newUiRegistrationPromptSignal.value += 1;
}

async function fetchNewUiPlayerById(playerId) {
  if (!playerId) {
    return null;
  }

  try {
    const { data } = await apiClient.get('/public/players');
    const players = Array.isArray(data?.players) ? data.players : Array.isArray(data) ? data : [];
    const selected = players.find((player) => Number(player?.id) === Number(playerId));
    if (!selected) {
      return null;
    }

    const fullName = [String(selected?.first_name || '').trim(), String(selected?.last_name || '').trim()]
      .filter(Boolean)
      .join(' ')
      .trim();
    return sanitizeStoredPlayer({
      id: Number(selected.id),
      name: fullName,
      lastName: String(selected?.last_name || '').trim(),
      number: selected?.jersey_number == null ? '' : String(selected.jersey_number),
      avatar: String(selected?.image_url || '').trim(),
    });
  } catch (error) {
    console.error('Impossibile caricare il giocatore votato per newui', error);
    return null;
  }
}

async function hydrateNewUiVote() {
  if (appView.value !== 'newui' || !resolvedEventId.value) {
    newUiSelectedPlayer.value = null;
    return;
  }

  const eventId = resolvedEventId.value;
  const storedPlayer = readStoredVotedPlayer(eventId);
  if (storedPlayer) {
    newUiSelectedPlayer.value = storedPlayer;
  }

  const voteStatus = await fetchVoteStatus(eventId);
  if (!voteStatus?.ok || !voteStatus.hasVoted || !voteStatus.playerId) {
    if (!voteStatus?.hasVoted) {
      newUiSelectedPlayer.value = null;
      writeStoredVotedPlayer(eventId, null);
    }
    return;
  }

  if (storedPlayer?.id === voteStatus.playerId) {
    return;
  }

  const playerFromApi = await fetchNewUiPlayerById(voteStatus.playerId);
  if (playerFromApi) {
    newUiSelectedPlayer.value = playerFromApi;
    writeStoredVotedPlayer(eventId, playerFromApi);
  }
}

async function fetchActiveEvent() {
  if (appView.value !== 'public' && appView.value !== 'newui') {
    return;
  }

  if (isFetchingActiveEvent.value) {
    return;
  }

  isFetchingActiveEvent.value = true;
  hasCheckedActiveEvent.value = false;
  try {
    const { data } = await apiClient.get('/active-event');
    activeEvent.value = data ?? null;
  } catch (error) {
    if (error?.response?.status === 204 || error?.response?.status === 404) {
      activeEvent.value = null;
    } else {
      console.error('Impossibile recuperare l\'evento attivo', error);
      activeEvent.value = null;
    }
  } finally {
    isFetchingActiveEvent.value = false;
    hasCheckedActiveEvent.value = true;
  }
}

function navigateTo(path, replace = false) {
  if (typeof window === 'undefined') {
    currentPath.value = path || '/';
    currentSearch.value = '';
    currentEventId.value = undefined;
    return;
  }

  try {
    const target = new URL(path, window.location.origin);
    if (replace) {
      window.history.replaceState({}, '', target);
    } else {
      window.history.pushState({}, '', target);
    }
    currentPath.value = target.pathname;
    currentSearch.value = target.search;
    currentEventId.value = readEventId(target.search);
  } catch (error) {
    console.error('Navigazione non riuscita', error);
  }

  if (typeof window !== 'undefined') {
    window.dispatchEvent(new Event('popstate'));
  }
}

onMounted(() => {
  window.addEventListener('popstate', handlePopState, { passive: true });
  if (appView.value === 'public' || appView.value === 'newui') {
    fetchActiveEvent();
  }
});

onBeforeUnmount(() => {
  window.removeEventListener('popstate', handlePopState);
});

watch(appView, (view) => {
  if (view === 'public' || view === 'newui') {
    fetchActiveEvent();
  } else {
    activeEvent.value = null;
    hasCheckedActiveEvent.value = false;
  }
});

watch(
  [appView, resolvedEventId],
  async () => {
    await hydrateNewUiVote();
  },
  { immediate: true },
);
</script>

<style scoped>
.app-shell {
  min-height: 100vh;
  background: linear-gradient(180deg, #0f172a 0%, #1e293b 50%, #0f172a 100%);
}
</style>
