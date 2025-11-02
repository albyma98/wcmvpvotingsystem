<template>
  <div class="app-shell">
    <AdminLottery v-if="adminView === 'lottery'" />
    <AdminPortal v-else-if="adminView === 'portal'" />
    <TicketValidationView v-else-if="adminView === 'ticket-validation'" />
    <CashLanding v-else-if="adminView === 'landing'" />
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
import AdminLottery from './components/AdminLottery.vue';
import TicketValidationView from './components/TicketValidationView.vue';
import CashLanding from './components/CashLanding.vue';
import VoteScreen from './components/VoteScreen.vue';
import { apiClient } from './api';

function readEventId(search) {
  const params = new URLSearchParams(search || '');
  const raw = params.get('eventId') ?? params.get('eventID');
  const parsed = Number.parseInt(raw ?? '', 10);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : undefined;
}

const currentPath = ref(typeof window !== 'undefined' ? window.location.pathname : '/');
const currentEventId = ref(typeof window !== 'undefined' ? readEventId(window.location.search) : undefined);
const activeEvent = ref(null);
const isFetchingActiveEvent = ref(false);
const hasCheckedActiveEvent = ref(false);

const adminView = computed(() => {
  if (currentPath.value.startsWith('/admin/lottery')) {
    return 'lottery';
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
  return 'public';
});

const resolvedEventId = computed(() => currentEventId.value ?? activeEvent.value?.id);

function handlePopState() {
  currentPath.value = window.location.pathname;
  currentEventId.value = readEventId(window.location.search);

  if (adminView.value === 'public') {
    fetchActiveEvent();
  }
}

async function fetchActiveEvent() {
  if (adminView.value !== 'public') {
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

onMounted(() => {
  window.addEventListener('popstate', handlePopState, { passive: true });
  if (adminView.value === 'public') {
    fetchActiveEvent();
  }
});

onBeforeUnmount(() => {
  window.removeEventListener('popstate', handlePopState);
});

watch(adminView, (view) => {
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
