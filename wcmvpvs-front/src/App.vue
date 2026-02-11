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
    <LiveExperienceHome v-else-if="appView === 'newui'" />
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

const pathSegments = computed(() =>
  currentPath.value
    .split('/')
    .map((part) => part.trim())
    .filter(Boolean),
);

const organizationSlug = computed(() => {
  if (pathSegments.value.length) {
    if (pathSegments.value[0] !== 'admin' && pathSegments.value[0] !== 'shop' && pathSegments.value[0] !== 'partner') {
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
  if (currentPath.value.startsWith('/newui')) {
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

async function fetchActiveEvent() {
  if (appView.value !== 'public') {
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
  if (appView.value === 'public') {
    fetchActiveEvent();
  }
});

onBeforeUnmount(() => {
  window.removeEventListener('popstate', handlePopState);
});

watch(appView, (view) => {
  if (view === 'public') {
    fetchActiveEvent();
  } else {
    activeEvent.value = null;
    hasCheckedActiveEvent.value = false;
  }
});
</script>

<style scoped>
.app-shell {
  min-height: 100vh;
  background: linear-gradient(180deg, #0f172a 0%, #1e293b 50%, #0f172a 100%);
}
</style>
