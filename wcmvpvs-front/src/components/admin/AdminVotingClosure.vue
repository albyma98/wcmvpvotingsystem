<template>
  <div v-if="summary || error" class="voting-closure">
    <p v-if="error" class="error" role="alert">{{ error }}</p>
    <button v-if="error" type="button" class="btn outline" @click="load">Aggiorna riepilogo</button>
    <template v-if="summary">
      <h3>Risultati della chiusura</h3>
      <p><strong>MVP: {{ summary.mvp_name }}</strong> · {{ summary.percentage }}% dei {{ summary.total_votes }} voti</p>
      <p class="muted">{{ summary.draw_mode === 'leaderboard' ? 'Premi ai migliori tifosi votanti in classifica monete' : 'Estrazione casuale tra i codici dei tifosi votanti' }}</p>
      <ol v-if="summary.prizes.length" class="prizes">
        <li v-for="prize in summary.prizes" :key="prize.id">
          <span>{{ prize.name }}</span>
          <strong>{{ prize.ticket_code || 'Non assegnato' }}</strong>
          <span v-if="prize.nickname" class="muted">{{ prize.nickname }}</span>
        </li>
      </ol>
      <p v-for="warning in summary.warnings" :key="warning" class="notice">{{ warning }}</p>
      <h4>Riepilogo WhatsApp agli organizzatori</h4>
      <p v-if="!notifications.length" class="muted">Nessun destinatario abilitato al momento della chiusura. Puoi configurarli nella dashboard.</p>
      <ul v-else class="notifications" aria-live="polite">
        <li v-for="notification in notifications" :key="notification.id">
          <div><strong>{{ notification.name || 'Organizzatore' }}</strong> <span class="muted">{{ notification.phone }}</span></div>
          <span :class="['notification-status', notification.status]">{{ statusLabel(notification.status) }}</span>
          <small v-if="notification.error">{{ notification.error }}</small>
        </li>
      </ul>
      <button v-if="canRetry" class="btn outline" type="button" :disabled="retrying" @click="retry">
        {{ retrying ? 'Invio in coda…' : 'Riprova notifiche non inviate' }}
      </button>
      <p class="muted closure-note">I risultati sono salvati: riattivare e richiudere l’evento non esegue una nuova estrazione.</p>
    </template>
  </div>
</template>

<script setup>
import { computed, onUnmounted, ref, watch } from 'vue';
import { apiClient } from '../../api';

const props = defineProps({ eventId: { type: Number, required: true }, authHeaders: { type: Object, required: true } });
const emit = defineEmits(['unauthorized']);
const summary = ref(null);
const notifications = ref([]);
const error = ref('');
const retrying = ref(false);
const canRetry = computed(() => notifications.value.some(n => ['failed', 'skipped'].includes(n.status)));
let timer;
let generation = 0;

function statusLabel(status) {
  return ({ pending: 'In coda', sending: 'Invio in corso', submitted: 'Accettato da Twilio', failed: 'Invio fallito', skipped: 'Configurazione mancante', unknown: 'Esito da verificare' })[status] || status;
}

async function load() {
  clearTimeout(timer);
  const current = ++generation;
  if (!props.eventId) return;
  error.value = '';
  try {
    const { data } = await apiClient.get(`/events/${props.eventId}/voting-closure`, props.authHeaders);
    if (current !== generation) return;
    summary.value = data.summary;
    notifications.value = data.notifications;
    if (notifications.value.some(n => ['pending', 'sending'].includes(n.status))) timer = setTimeout(load, 4000);
  } catch (e) {
    if (current !== generation) return;
    if (e?.response?.status === 404) { summary.value = null; notifications.value = []; return; }
    if (e?.response?.status === 401) emit('unauthorized');
    error.value = 'Impossibile aggiornare il riepilogo della chiusura.';
  }
}

async function retry() {
  if (retrying.value) return;
  const eventId = props.eventId;
  retrying.value = true;
  error.value = '';
  try {
    await apiClient.post(`/events/${eventId}/voting-closure/retry`, {}, props.authHeaders);
    if (props.eventId === eventId) await load();
  } catch (e) {
    if (e?.response?.status === 401) emit('unauthorized');
    error.value = 'Impossibile rimettere in coda le notifiche.';
  } finally { retrying.value = false; }
}

watch(() => props.eventId, () => { summary.value = null; notifications.value = []; load(); }, { immediate: true });
onUnmounted(() => { generation++; clearTimeout(timer); });
</script>

<style scoped>
.voting-closure { margin-top: 1.5rem; border-top: 1px solid #334155; padding-top: 1.25rem; }
h3, h4 { margin: .5rem 0 .8rem; }
.muted, small { color: #94a3b8; }
.prizes, .notifications { padding: 0; list-style: none; }
.prizes li { display: flex; align-items: center; flex-wrap: wrap; gap: .75rem; margin: .65rem 0; }
.prizes strong { font-family: monospace; font-size: 1.2rem; letter-spacing: .08em; }
.notifications li { display: flex; flex-wrap: wrap; gap: .5rem; justify-content: space-between; border-bottom: 1px solid #334155; padding: .8rem 0; }
.notifications small { flex-basis: 100%; }
.notification-status { font-size: .8rem; color: #c4b5fd; }
.notification-status.submitted { color: #86efac; }
.notification-status.failed, .error { color: #fca5a5; }
.notice, .notification-status.unknown, .notification-status.skipped { color: #fcd34d; }
.closure-note { font-size: .8rem; line-height: 1.5; }
</style>
