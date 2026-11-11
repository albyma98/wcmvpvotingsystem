<script setup>
import { computed, inject, ref, watch } from 'vue';
import { chatDateTime, chatErrorText, chatReasons, chatRequestId, connectMatchChat, matchChatRequest } from '../../../match-chat/matchChatApi';

const props = defineProps({ event: { type: Object, required: true } });
const c = inject('admin-v2');
const enabled = ref(false);
const loaded = ref(false);
const busy = ref(false);
const error = ref('');
const stats = ref({});
const messages = ref([]);
const reports = ref([]);
const mutes = ref([]);
const messagesBefore = ref(0);
const reportsBefore = ref(0);
const mutesAfter = ref(0);
const official = ref('');
const muteMinutes = ref(15);
const changed = ref(false);
const conversion = computed(() => stats.value.opens ? Math.round(stats.value.openers_who_wrote / stats.value.opens * 100) : 0);
let scope;
let officialRetry;

function request(options = {}) {
  return matchChatRequest(scope.slug, scope.eventId, { admin: true, signal: scope.controller.signal, ...options });
}
async function loadMessages(append = false) {
  const data = await request({ path: '/messages', params: append ? { before: messagesBefore.value } : {} });
  messages.value = data.messages.slice().reverse();
  messagesBefore.value = data.next_before;
}
async function loadReports(append = false) {
  const data = await request({ path: '/reports', params: append ? { before: reportsBefore.value } : {} });
  reports.value = data.reports;
  reportsBefore.value = data.next_before;
}
async function loadMutes(append = false) {
  const data = await request({ path: '/mutes', params: append ? { after: mutesAfter.value } : {} });
  mutes.value = data.mutes;
  mutesAfter.value = data.next_after;
}
async function reload() {
  const data = await request();
  enabled.value = data.enabled;
  stats.value = data.stats;
  await Promise.all([loadMessages(), loadReports(), loadMutes()]);
  loaded.value = true;
  changed.value = false;
}
async function run(action) {
  if (busy.value) return;
  busy.value = true;
  error.value = '';
  const current = scope;
  try {
    await action();
  } catch (err) {
    if (scope === current && !current.controller.signal.aborted) error.value = chatErrorText(err);
  } finally {
    if (scope === current) busy.value = false;
  }
}
async function toggle() {
  const response = await request({ method: 'put', path: '/settings', data: { enabled: !enabled.value } });
  enabled.value = response.enabled;
}
async function publish() {
  const text = official.value.trim();
  if (!text || Array.from(text).length > 180) return;
  const requestId = officialRetry?.text === text ? officialRetry.id : chatRequestId();
  officialRetry = { text, id: requestId };
  await request({ method: 'post', path: '/messages', data: { text, request_id: requestId } });
  official.value = '';
  officialRetry = null;
  await reload();
}
function remove(message) {
  c.confirmAction('Rimuovere questo messaggio?', 'Il contenuto verrà nascosto ai tifosi e resterà disponibile nella moderazione di questo evento.',
    () => run(async () => {
      await request({ method: 'delete', path: `/messages/${message.id}` });
      await reload();
    }));
}
async function mute(fanId) {
  await request({ method: 'put', path: `/mutes/${fanId}`, data: { minutes: Number(muteMinutes.value) } });
  await loadMutes();
}
async function unmute(fanId) {
  await request({ method: 'delete', path: `/mutes/${fanId}` });
  await loadMutes();
}
async function review(id) {
  await request({ method: 'put', path: `/reports/${id}` });
  await loadReports();
}

watch(() => props.event.id, (id, _, cleanup) => {
  scope = { eventId: id, slug: c.organizationSlug, controller: new AbortController() };
  const current = scope;
  enabled.value = false;
  loaded.value = false;
  busy.value = false;
  error.value = '';
  messages.value = [];
  reports.value = [];
  mutes.value = [];
  official.value = '';
  officialRetry = null;
  run(reload);
  let stopMessages;
  const stopStatus = connectMatchChat(current.slug, id, {
    statusOnly: true,
    onUpdate(update) {
      if (scope !== current) return;
      enabled.value = update.enabled === true;
      stopMessages?.();
      if (enabled.value) {
        stopMessages = connectMatchChat(current.slug, id, {
          onUpdate(message) { if (message.type !== 'status') changed.value = true; },
          onDisconnect() { changed.value = true; },
        });
      }
    },
  });
  cleanup(() => { current.controller.abort(); stopStatus(); stopMessages?.(); });
}, { immediate: true });

</script>

<template>
  <section class="v2-stack chat-admin ph-no-capture" aria-labelledby="event-chat-title">
    <div class="v2-toolbar">
      <h2 id="event-chat-title">Match Chat</h2>
      <button class="v2-button" type="button" :disabled="busy" @click="run(reload)">Aggiorna conversazione e statistiche</button>
    </div>
    <p v-if="error" class="v2-error" role="alert">{{ error }}</p>
    <p v-if="!loaded" role="status">{{ busy ? 'Caricamento Match Chat…' : 'Dati non disponibili. Riprova ad aggiornare.' }}</p>
    <template v-if="loaded">
      <label class="v2-check chat-admin-toggle">
        <input type="checkbox" role="switch" :checked="enabled" :disabled="busy" @click.prevent="run(toggle)" />
        Abilita Match Chat per questo evento
      </label>
      <p class="v2-muted">{{ enabled ? 'La chat è disponibile nella homepage dei tifosi di questa partita.' : 'La chat è disattivata. La cronologia è conservata.' }}</p>
      <dl class="chat-admin-stats">
        <div><dt>Messaggi</dt><dd>{{ stats.messages }}</dd></div>
        <div><dt>Tifosi che scrivono</dt><dd>{{ stats.authors }}</dd></div>
        <div><dt>Reazioni</dt><dd>{{ stats.reactions }}</dd></div>
        <div><dt>Segnalazioni</dt><dd>{{ stats.reports }}</dd></div>
        <div><dt>Aperture uniche</dt><dd>{{ stats.opens }}</dd></div>
        <div><dt>Partecipazione dopo apertura</dt><dd>{{ conversion }}%</dd></div>
      </dl>
      <p v-if="changed" class="v2-notice" role="status">La conversazione è cambiata. Aggiorna per visualizzare i nuovi contenuti.</p>

      <form class="v2-stack" @submit.prevent="run(publish)">
        <label class="v2-field">Messaggio ufficiale della società
          <textarea v-model="official" rows="2" maxlength="360" :disabled="busy || !enabled" placeholder="Scrivi ai tuoi tifosi…"></textarea>
          <small>{{ Array.from(official).length }}/180 caratteri</small>
        </label>
        <div><button class="v2-button v2-primary" type="submit" :disabled="busy || !enabled || !official.trim() || Array.from(official).length > 180">Pubblica come società</button></div>
      </form>

      <h3>Messaggi dell’evento</h3>
      <label class="v2-field">Durata della sospensione
        <select v-model.number="muteMinutes" :disabled="busy"><option :value="5">5 minuti</option><option :value="15">15 minuti</option><option :value="60">1 ora</option><option :value="1440">24 ore</option></select>
      </label>
      <div class="v2-toolbar">
        <button type="button" class="v2-button" :disabled="busy" @click="run(() => loadMessages())">Messaggi più recenti</button>
        <button v-if="messagesBefore" type="button" class="v2-button" :disabled="busy" @click="run(() => loadMessages(true))">Messaggi precedenti</button>
      </div>
      <p v-if="!messages.length" class="v2-empty">Nessun messaggio per questo evento.</p>
      <article v-for="message in messages" :key="message.id" class="chat-admin-message">
        <div class="v2-toolbar"><strong>{{ message.nickname }}</strong><time>{{ chatDateTime(message.created_at) }}</time>
          <span class="v2-badge">{{ message.type === 'official' ? 'Società' : message.type === 'ai' ? 'AI' : 'Tifoso' }}</span>
          <span class="v2-badge">{{ message.status === 'removed' ? 'Rimosso' : 'Visibile' }}</span>
        </div>
        <p>{{ message.text }}</p>
        <div class="v2-toolbar">
          <span v-for="reaction in message.reactions" :key="reaction.emoji">{{ reaction.emoji }} {{ reaction.count }}</span>
          <button v-if="message.status !== 'removed'" type="button" class="v2-button v2-danger" :disabled="busy" @click="remove(message)">Rimuovi</button>
          <button v-if="message.author_id" type="button" class="v2-button" :disabled="busy" @click="run(() => mute(message.author_id))">Sospendi {{ message.nickname }}</button>
        </div>
      </article>

      <h3>Segnalazioni</h3>
      <p v-if="!reports.length" class="v2-empty">Nessuna segnalazione per questo evento.</p>
      <article v-for="report in reports" :key="report.id" class="chat-admin-message">
        <div class="v2-toolbar"><strong>{{ report.nickname }}</strong><time>{{ chatDateTime(report.created_at) }}</time>
          <span class="v2-badge">{{ report.status === 'pending' ? 'Da esaminare' : 'Esaminata' }}</span>
        </div>
        <p>{{ report.text }}</p><p class="v2-muted">{{ chatReasons[report.reason] }}</p>
        <div class="v2-toolbar">
          <button v-if="report.message_status !== 'removed'" type="button" class="v2-button v2-danger" :disabled="busy" @click="remove({ id: report.message_id })">Rimuovi messaggio</button>
          <button v-if="report.status === 'pending'" type="button" class="v2-button" :disabled="busy" @click="run(() => review(report.id))">Segna come esaminata</button>
        </div>
      </article>
      <div v-if="reports.length" class="v2-toolbar">
        <button type="button" class="v2-button" :disabled="busy" @click="run(() => loadReports())">Segnalazioni più recenti</button>
        <button v-if="reportsBefore" type="button" class="v2-button" :disabled="busy" @click="run(() => loadReports(true))">Altre segnalazioni</button>
      </div>

      <h3>Sospensioni attive</h3>
      <p v-if="!mutes.length" class="v2-muted">Nessun tifoso sospeso per questo evento.</p>
      <article v-for="mute in mutes" :key="mute.fan_id" class="chat-admin-message">
        <div class="v2-toolbar"><strong>{{ mute.nickname }}</strong><span>Fino al {{ chatDateTime(mute.until) }}</span>
          <button type="button" class="v2-button" :disabled="busy" @click="run(() => unmute(mute.fan_id))">Annulla sospensione</button>
        </div>
      </article>
      <div v-if="mutes.length" class="v2-toolbar">
        <button type="button" class="v2-button" :disabled="busy" @click="run(() => loadMutes())">Inizio elenco sospensioni</button>
        <button v-if="mutesAfter" type="button" class="v2-button" :disabled="busy" @click="run(() => loadMutes(true))">Altre sospensioni</button>
      </div>
    </template>
  </section>
</template>

<style scoped>
.chat-admin-toggle { min-height: 44px; font-weight: 700; }
.chat-admin-toggle input { width: 20px; height: 20px; }
.chat-admin-stats { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; margin: 0; }
.chat-admin-stats > div { padding: 15px; background: var(--v2-soft); border-radius: 10px; }
.chat-admin-stats dt { color: var(--v2-muted); font-size: .8rem; }
.chat-admin-stats dd { margin: 5px 0 0; font-size: 1.4rem; font-weight: 750; }
.chat-admin-message { padding: 16px; border: 1px solid var(--v2-line); border-radius: 10px; }
.chat-admin-message p { white-space: pre-wrap; overflow-wrap: anywhere; margin: 10px 0 !important; }
.chat-admin-message time { font-size: .75rem; color: var(--v2-muted); }
.chat-admin .v2-field select { max-width: 220px; }
@media(max-width:640px) { .chat-admin-stats { grid-template-columns: repeat(2, minmax(0,1fr)); } }
</style>
