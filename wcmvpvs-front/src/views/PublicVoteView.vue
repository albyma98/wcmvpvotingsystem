<template>
  <div class="public-wrapper">
    <VoteScreen
      :event-id="resolvedEventId"
      :active-event="activeEvent"
      :active-event-checked="hasCheckedActiveEvent"
      :loading-active-event="isFetchingActiveEvent"
    />
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import VoteScreen from '../components/VoteScreen.vue';
import { apiClient } from '../api';

function readEventId(search) {
  const params = new URLSearchParams(search || '');
  const raw = params.get('eventId') ?? params.get('eventID');
  const parsed = Number.parseInt(raw ?? '', 10);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : undefined;
}

const currentEventId = ref(typeof window !== 'undefined' ? readEventId(window.location.search) : undefined);
const activeEvent = ref(null);
const isFetchingActiveEvent = ref(false);
const hasCheckedActiveEvent = ref(false);

const resolvedEventId = computed(() => currentEventId.value ?? activeEvent.value?.id);

function handlePopState() {
  if (typeof window === 'undefined') return;
  currentEventId.value = readEventId(window.location.search);
  if (!currentEventId.value) {
    fetchActiveEvent();
  }
}

async function fetchActiveEvent() {
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
  if (!resolvedEventId.value) {
    fetchActiveEvent();
  }
  if (typeof window !== 'undefined') {
    window.addEventListener('popstate', handlePopState, { passive: true });
  }
});

onBeforeUnmount(() => {
  if (typeof window !== 'undefined') {
    window.removeEventListener('popstate', handlePopState);
  }
});
</script>

<style scoped>
.public-wrapper {
  min-height: 100vh;
  background: linear-gradient(180deg, #0f172a 0%, #1e293b 50%, #0f172a 100%);
}
</style>
