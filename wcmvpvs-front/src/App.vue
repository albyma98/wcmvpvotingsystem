<template>
  <div class="app-shell">
    <AdminPortal v-if="isAdminRoute" />
    <VoteScreen v-else :event-id="currentEventId" />
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import AdminPortal from './components/AdminPortal.vue';
import VoteScreen from './components/VoteScreen.vue';

function readEventId(search) {
  const params = new URLSearchParams(search || '');
  const raw = params.get('eventId') ?? params.get('eventID');
  const parsed = Number.parseInt(raw ?? '', 10);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : undefined;
}

const currentPath = ref(typeof window !== 'undefined' ? window.location.pathname : '/');
const currentEventId = ref(typeof window !== 'undefined' ? readEventId(window.location.search) : undefined);

const isAdminRoute = computed(() => currentPath.value.startsWith('/admin'));

function handlePopState() {
  currentPath.value = window.location.pathname;
  currentEventId.value = readEventId(window.location.search);
}

onMounted(() => {
  window.addEventListener('popstate', handlePopState, { passive: true });
});

onBeforeUnmount(() => {
  window.removeEventListener('popstate', handlePopState);
});
</script>

<style scoped>
.app-shell {
  min-height: 100vh;
  background: linear-gradient(180deg, #0f172a 0%, #1e293b 50%, #0f172a 100%);
}
</style>
