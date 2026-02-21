<template>
  <main class="scrape-test-page">
    <h1>Test scrape slug società</h1>
    <p>Inserisci lo slug/URL della società e controlla in tempo reale il JSON estratto.</p>

    <form class="controls" @submit.prevent="runScrapeTest">
      <input
        v-model.trim="slugInput"
        type="text"
        placeholder="es. volley-milano oppure https://mia-societa.it"
      />
      <button type="submit" :disabled="isLoading">{{ isLoading ? 'Caricamento…' : 'Esegui test' }}</button>
    </form>

    <p v-if="errorMessage" class="error">{{ errorMessage }}</p>

    <pre v-if="resultJson" class="json-output">{{ resultJson }}</pre>
  </main>
</template>

<script setup>
import { computed, ref } from 'vue';
import { apiClient } from '../api';

const urlParams = new URLSearchParams(typeof window !== 'undefined' ? window.location.search : '');
const slugInput = ref(
  urlParams.get('slug') || urlParams.get('organization_slug') || urlParams.get('org') || '',
);
const isLoading = ref(false);
const errorMessage = ref('');
const result = ref(null);

const resultJson = computed(() => (result.value ? JSON.stringify(result.value, null, 2) : ''));

async function runScrapeTest() {
  isLoading.value = true;
  errorMessage.value = '';

  const selectedSlug = slugInput.value.trim();
  const config = selectedSlug
    ? {
        headers: {
          'X-Organization-Slug': selectedSlug,
        },
      }
    : {};

  try {
    const [contextResponse, activeEventResponse, playersResponse] = await Promise.allSettled([
      apiClient.get('/context', config),
      apiClient.get('/active-event', config),
      apiClient.get('/public/players', config),
    ]);

    result.value = {
      executed_at: new Date().toISOString(),
      organization_slug: selectedSlug,
      context: settleToPayload(contextResponse),
      active_event: settleToPayload(activeEventResponse),
      public_players: settleToPayload(playersResponse),
    };
  } catch (error) {
    errorMessage.value = "Errore durante l'esecuzione del test scrape.";
  } finally {
    isLoading.value = false;
  }
}

function settleToPayload(settledResult) {
  if (settledResult.status === 'fulfilled') {
    return {
      ok: true,
      status: settledResult.value?.status,
      data: settledResult.value?.data ?? null,
    };
  }

  return {
    ok: false,
    status: settledResult.reason?.response?.status ?? null,
    error: settledResult.reason?.response?.data || settledResult.reason?.message || 'unknown_error',
  };
}

if (slugInput.value) {
  runScrapeTest();
}
</script>

<style scoped>
.scrape-test-page {
  min-height: 100vh;
  padding: 24px;
  color: #f8fafc;
}

.controls {
  display: flex;
  gap: 12px;
  margin: 16px 0;
}

input {
  flex: 1;
  min-width: 240px;
  padding: 10px 12px;
  border-radius: 8px;
  border: 1px solid #334155;
  background: #0f172a;
  color: #f8fafc;
}

button {
  padding: 10px 14px;
  border-radius: 8px;
  border: 1px solid #38bdf8;
  background: #0ea5e9;
  color: #082f49;
  font-weight: 600;
  cursor: pointer;
}

button:disabled {
  opacity: 0.6;
  cursor: wait;
}

.error {
  color: #fca5a5;
}

.json-output {
  margin-top: 18px;
  background: #020617;
  border: 1px solid #1e293b;
  border-radius: 8px;
  padding: 14px;
  white-space: pre-wrap;
  word-break: break-word;
}
</style>
