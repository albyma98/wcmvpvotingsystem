<template>
  <div class="app-shell">
    <AdminLottery v-if="appView === 'lottery'" />
    <MasterPortal v-else-if="appView === 'master'" />
    <AdminPortal v-else-if="appView === 'portal'" :organization-slug="organizationSlug" />
    <TicketValidationView v-else-if="appView === 'ticket-validation'" />
    <CashLanding v-else-if="appView === 'landing'" />
    <ShopAdminPortal
      v-else-if="appView === 'shop-admin'"
      :current-path="currentPath"
      :current-search="currentSearch"
      :on-navigate="navigateTo"
    />
    <ShopShell
      v-else-if="appView === 'shop'"
      :current-path="currentPath"
      :current-search="currentSearch"
      :on-navigate="navigateTo"
    />
    <PartnerPortal
      v-else-if="appView === 'partner'"
      :current-path="currentPath"
      :current-search="currentSearch"
    />
    <LiveExperienceHome v-else-if="appView === 'newui'" @feature-select="handleNewUiFeatureSelect" />
    <VoteScreen
      v-else
      :event-id="resolvedEventId"
      :active-event="activeEvent"
      :active-event-checked="hasCheckedActiveEvent"
      :loading-active-event="isFetchingActiveEvent"
    />

    <div
      v-if="showMvpVoteModal"
      class="newui-modal-overlay"
      role="dialog"
      aria-modal="true"
      aria-label="Vota l'MVP del pubblico"
      @click.self="closeMvpVoteModal"
    >
      <div class="newui-modal-panel">
        <button
          type="button"
          class="newui-modal-close"
          aria-label="Chiudi modale voto MVP"
          @click="closeMvpVoteModal"
        >
          ✕
        </button>
        <VoteMVP :event-id="resolvedEventId" />
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import AdminPortal from './components/AdminPortal.vue';
import MasterPortal from './components/MasterPortal.vue';
import AdminLottery from './components/AdminLottery.vue';
import TicketValidationView from './components/TicketValidationView.vue';
import CashLanding from './components/CashLanding.vue';
import VoteScreen from './components/VoteScreen.vue';
import ShopShell from './components/shop/ShopShell.vue';
import ShopAdminPortal from './components/shop/ShopAdminPortal.vue';
import PartnerPortal from './components/PartnerPortal.vue';
import LiveExperienceHome from './views/LiveExperienceHome.vue';
import VoteMVP from './components/VoteMVP.vue';
import { apiClient } from './api';

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
const showMvpVoteModal = ref(false);

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
      pathSegments.value[0] !== 'shop' &&
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
  if (currentPath.value.startsWith('/shop/admin')) {
    return 'shop-admin';
  }
  if (currentPath.value.startsWith('/shop')) {
    return 'shop';
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
    showMvpVoteModal.value = true;
  }
}

function closeMvpVoteModal() {
  showMvpVoteModal.value = false;
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
    console.error('Navigazione shop non riuscita', error);
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
    showMvpVoteModal.value = false;
  }
});
</script>

<style scoped>
.app-shell {
  min-height: 100vh;
  background: linear-gradient(180deg, #0f172a 0%, #1e293b 50%, #0f172a 100%);
}

.newui-modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 1000;
  background: rgba(2, 6, 23, 0.74);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
}

.newui-modal-panel {
  position: relative;
  width: min(100%, 920px);
  max-height: calc(100dvh - 2rem);
  overflow: auto;
  border-radius: 1rem;
  background: #f8fafc;
  box-shadow: 0 32px 70px rgba(2, 6, 23, 0.55);
}

.newui-modal-close {
  position: sticky;
  top: 0.75rem;
  float: right;
  margin: 0.75rem 0.75rem 0 0;
  border: 0;
  border-radius: 999px;
  width: 2.2rem;
  height: 2.2rem;
  font-size: 1.05rem;
  font-weight: 700;
  color: #0f172a;
  background: rgba(148, 163, 184, 0.25);
  cursor: pointer;
}
</style>
