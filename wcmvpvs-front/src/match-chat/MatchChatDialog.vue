<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { chatEmojis, chatErrorCode, chatErrorText, chatReasons, chatRequestId, chatTime, connectMatchChat, matchChatRequest } from './matchChatApi';

const props = defineProps({ slug: String, eventId: Number, eventLabel: String, enabled: Boolean });
const emit = defineEmits(['close', 'verify', 'track']);
const dialog = ref(null);
const scroller = ref(null);
const messages = ref([]);
const before = ref(0);
const state = ref('connecting');
const canWrite = ref(false);
const draft = ref('');
const replyTo = ref(null);
const composer = ref(null);
const sending = ref(false);
const loadingOlder = ref(false);
const archived = ref(false);
const unread = ref(0);
const error = ref('');
const notice = ref('');
const reportId = ref(null);
const reason = ref('abuse');
const reporting = ref(false);
const reported = ref([]);
const reacting = ref('');
const length = computed(() => Array.from(draft.value).length);

const maxMessages = 240;
const lifetime = new AbortController();
const ownMessages = new Set();
let stopStream;
let syncing;
let olderRequest;
let retryDraft;

function request(options) {
  return matchChatRequest(props.slug, props.eventId, { signal: lifetime.signal, ...options });
}
function nearBottom() {
  const el = scroller.value;
  return !el || el.scrollHeight - el.scrollTop - el.clientHeight < 72;
}
function scrollBottom() {
  nextTick(() => {
    if (scroller.value) scroller.value.scrollTop = scroller.value.scrollHeight;
    unread.value = 0;
  });
}
function reactionsFor(message, reaction, personal) {
  const existing = message.reactions?.find(item => item.emoji === reaction.emoji);
  const count = personal ? existing?.count || 0 : reaction.count;
  const mine = personal ? Boolean(reaction.mine) : Boolean(existing?.mine);
  const other = (message.reactions || []).filter(item => item.emoji !== reaction.emoji);
  return count || mine ? [...other, { emoji: reaction.emoji, count, mine }] : other;
}
function applyUpdate(update) {
  if (update.type === 'message') {
    const incoming = update.message;
    if (messages.value.some(message => message.id === incoming.id)) return;
    if (archived.value) {
      unread.value++;
      return;
    }
    const follow = nearBottom();
    messages.value.push({ ...incoming, mine: ownMessages.has(incoming.id) });
    if (messages.value.length > maxMessages) {
      const el = scroller.value;
      const anchor = follow ? null : Array.from(el?.querySelectorAll('[data-message]') || [])
        .find(node => node.offsetTop + node.offsetHeight > el.scrollTop);
      const offset = anchor && anchor.offsetTop - el.scrollTop;
      messages.value.shift();
      before.value = messages.value[0].id;
      if (anchor) nextTick(() => { if (anchor.isConnected) el.scrollTop = anchor.offsetTop - offset; });
    }
    if (follow) scrollBottom();
    else unread.value++;
  } else if (update.type === 'removed') {
    messages.value = messages.value.filter(message => message.id !== update.message_id);
    if (reportId.value === update.message_id) reportId.value = null;
    if (replyTo.value?.id === update.message_id) replyTo.value = null;
  } else if (update.type === 'reaction') {
    const message = messages.value.find(item => item.id === update.message_id);
    if (message) message.reactions = reactionsFor(message, update.reaction, false);
  }
}

function handleError(err) {
  if (lifetime.signal.aborted) return;
  if (chatErrorCode(err) === 'chat_disabled') disable();
  if (chatErrorCode(err) === 'chat_verification_required') canWrite.value = false;
  error.value = chatErrorText(err);
}

function resetRequests() {
  syncing?.controller.abort();
  syncing = null;
  olderRequest?.controller.abort();
  olderRequest = null;
  loadingOlder.value = false;
}

function disable() {
  stopStream?.();
  stopStream = null;
  resetRequests();
  state.value = 'disabled';
  canWrite.value = false;
  reportId.value = null;
  replyTo.value = null;
  notice.value = '';
  messages.value = [];
  before.value = 0;
  unread.value = 0;
}

async function syncHistory() {
  resetRequests();
  const pending = { controller: new AbortController(), updates: [] };
  syncing = pending;
  state.value = 'syncing';
  error.value = '';
  try {
    const data = await request({ path: '/messages', params: { matchday_open: 1 }, signal: pending.controller.signal });
    if (syncing !== pending) return;
    syncing = null;
    if (pending.overflow) {
      syncHistory();
      return;
    }
    messages.value = data.messages;
    before.value = data.next_before;
    canWrite.value = data.can_write;
    archived.value = false;
    state.value = 'live';
    for (const update of pending.updates) applyUpdate(update);
    scrollBottom();
  } catch (err) {
    if (syncing !== pending) return;
    syncing = null;
    state.value = 'error';
    handleError(err);
  }
}

function queue(pending, update) {
  if (pending.updates.length < 256) pending.updates.push(update);
  else pending.overflow = true;
}

function startStream() {
  stopStream?.();
  state.value = 'connecting';
  stopStream = connectMatchChat(props.slug, props.eventId, {
    onUpdate(update) {
      if (update.type === 'status') {
        if (update.enabled) syncHistory();
        else disable();
        return;
      }
      if (syncing) queue(syncing, update);
      if (olderRequest) queue(olderRequest, update);
      if (state.value === 'live') applyUpdate(update);
    },
    onDisconnect() {
      resetRequests();
      state.value = 'reconnecting';
    },
  });
}

async function loadOlder() {
  if (!before.value || olderRequest || state.value !== 'live') return;
  const pending = { controller: new AbortController(), updates: [] };
  olderRequest = pending;
  loadingOlder.value = true;
  try {
    const data = await request({ path: '/messages', params: { before: before.value }, signal: pending.controller.signal });
    if (olderRequest !== pending) return;
    if (pending.overflow) return;
    const previous = messages.value;
    const removed = new Set(pending.updates.filter(update => update.type === 'removed').map(update => update.message_id));
    const older = data.messages.filter(message => !removed.has(message.id));
    const byId = new Map(older.map(message => [message.id, message]));
    for (const update of pending.updates) {
      const message = byId.get(update.message_id);
      if (message && update.type === 'reaction') message.reactions = reactionsFor(message, update.reaction, false);
    }
    const el = scroller.value;
    const anchor = el?.querySelector('[data-message]');
    const offset = anchor && anchor.offsetTop - el.scrollTop;
    const combined = [...older, ...previous];
    if (combined.length > maxMessages) archived.value = true;
    messages.value = combined.slice(0, maxMessages);
    before.value = data.next_before;
    await nextTick();
    if (anchor?.isConnected) el.scrollTop = anchor.offsetTop - offset;
  } catch (err) {
    if (olderRequest === pending) handleError(err);
  } finally {
    if (olderRequest === pending) {
      olderRequest = null;
      loadingOlder.value = false;
    }
  }
}

async function send() {
  const text = draft.value.trim();
  if (!text || length.value > 180 || sending.value || state.value !== 'live') return;
  if (!canWrite.value) { emit('verify'); return; }
  const replyId = replyTo.value?.id;
  const id = retryDraft?.text === text && retryDraft.replyId === replyId ? retryDraft.id : chatRequestId();
  retryDraft = { text, id, replyId };
  sending.value = true;
  error.value = '';
  notice.value = '';
  try {
    const response = await request({ method: 'post', path: '/messages', data: { text, request_id: id, ...(replyId ? { reply_to_id: replyId } : {}) } });
    if (lifetime.signal.aborted) return;
    const messageId = response.message.id;
    if (ownMessages.size >= maxMessages) ownMessages.delete(ownMessages.values().next().value);
    ownMessages.add(messageId);
    const message = messages.value.find(item => item.id === messageId);
    if (message) message.mine = true;
    if (draft.value.trim() === text) draft.value = '';
    if (replyTo.value?.id === replyId) replyTo.value = null;
    retryDraft = null;
    if (response.changed) emit('track', 'match_chat_message_sent');
  } catch (err) {
    if (chatErrorCode(err) === 'chat_reply_unavailable' && replyTo.value?.id === replyId) replyTo.value = null;
    handleError(err);
  } finally {
    sending.value = false;
  }
}

function beginReply(message) {
  if (!canWrite.value) { emit('verify'); return; }
  replyTo.value = { id: message.id, nickname: message.nickname };
  nextTick(() => composer.value?.focus());
}

function replyAuthor(id) {
  return messages.value.find(message => message.id === id)?.nickname || 'un messaggio precedente';
}

async function react(message, emoji) {
  if (state.value !== 'live' || reacting.value) return;
  if (!canWrite.value) { emit('verify'); return; }
  const active = !message.reactions?.find(item => item.emoji === emoji)?.mine;
  reacting.value = `${message.id}:${emoji}`;
  error.value = '';
  try {
    const response = await request({ method: 'put', path: `/messages/${message.id}/reaction`, data: { emoji, active } });
    const current = messages.value.find(item => item.id === message.id);
    if (current) current.reactions = reactionsFor(current, { ...response.reaction, mine: active }, true);
    if (response.changed && active) emit('track', 'match_chat_reaction_added');
  } catch (err) {
    handleError(err);
  } finally {
    reacting.value = '';
  }
}

function beginReport(message) {
  if (!canWrite.value) { emit('verify'); return; }
  reportId.value = reportId.value === message.id ? null : message.id;
  reason.value = 'abuse';
}
async function report() {
  const id = reportId.value;
  if (!id || reporting.value || state.value !== 'live') return;
  reporting.value = true;
  error.value = '';
  try {
    const response = await request({ method: 'post', path: `/messages/${id}/report`, data: { reason: reason.value } });
    reported.value = [...reported.value.slice(-99), id];
    reportId.value = null;
    notice.value = 'Segnalazione ricevuta.';
    if (response.changed) emit('track', 'match_chat_message_reported');
  } catch (err) {
    handleError(err);
  } finally {
    reporting.value = false;
  }
}

function updateViewport() {
  const viewport = window.visualViewport;
  if (!viewport || !dialog.value) return;
  const follow = nearBottom() && !archived.value;
  dialog.value.style.setProperty('--chat-viewport', `${viewport.height}px`);
  dialog.value.style.setProperty('--chat-bottom', `${Math.max(0, window.innerHeight - viewport.height - viewport.offsetTop)}px`);
  if (follow) scrollBottom();
}

watch(() => props.enabled, enabled => { if (enabled) startStream(); else disable(); });
onMounted(() => {
  dialog.value.showModal();
  updateViewport();
  window.visualViewport?.addEventListener('resize', updateViewport);
  window.visualViewport?.addEventListener('scroll', updateViewport);
  if (props.enabled) startStream();
  else disable();
});
onBeforeUnmount(() => {
  stopStream?.();
  resetRequests();
  lifetime.abort();
  window.visualViewport?.removeEventListener('resize', updateViewport);
  window.visualViewport?.removeEventListener('scroll', updateViewport);
  dialog.value?.close();
});
</script>

<template>
  <dialog ref="dialog" class="chat-dialog ph-no-capture" aria-labelledby="match-chat-title"
    @cancel.prevent="emit('close')" @click.self="emit('close')">
    <section class="chat-sheet">
      <header class="chat-header">
        <div>
          <h2 id="match-chat-title">Match Chat</h2>
          <p>{{ eventLabel }}</p>
        </div>
        <span v-if="state === 'live'" class="chat-live"><i></i> LIVE</span>
        <button type="button" class="chat-close" aria-label="Chiudi Match Chat" autofocus @click="emit('close')">×</button>
      </header>

      <div ref="scroller" class="chat-messages" tabindex="0" aria-label="Messaggi"
        @scroll="nearBottom() && !archived && (unread = 0)">
        <button v-if="before" type="button" class="chat-older" :disabled="loadingOlder || state !== 'live'" @click="loadOlder">
          {{ loadingOlder ? 'Caricamento…' : 'Messaggi precedenti' }}
        </button>
        <p v-if="state === 'disabled'" class="chat-empty">La società ha disattivato la chat per questa partita.</p>
        <p v-else-if="!messages.length && state === 'live'" class="chat-empty">La tribuna è ancora in silenzio. Rompi il ghiaccio!</p>
        <p v-else-if="!messages.length && state !== 'error'" class="chat-empty">Connessione alla chat…</p>

        <article v-for="message in messages" :key="message.id" data-message class="chat-message"
          :class="{ 'chat-mine': message.mine, 'chat-official': message.type === 'official' }">
          <span class="chat-avatar" aria-hidden="true">{{ Array.from(message.nickname)[0]?.toLocaleUpperCase() }}</span>
          <div class="chat-bubble">
            <div class="chat-byline">
              <strong>{{ message.nickname }}</strong>
              <span v-if="message.type === 'official'" class="chat-badge">Ufficiale</span>
              <span v-else-if="message.type === 'ai'" class="chat-badge">AI</span>
              <time :datetime="new Date(message.created_at).toISOString()">{{ chatTime(message.created_at) }}</time>
            </div>
            <p v-if="message.reply_to_id" class="chat-reply-reference">In risposta a {{ replyAuthor(message.reply_to_id) }}</p>
            <p>{{ message.text }}</p>
            <div class="chat-reactions">
              <button v-for="emoji in chatEmojis" :key="emoji" type="button"
                :aria-label="`Reazione ${emoji}, ${message.reactions?.find(item => item.emoji === emoji)?.count || 0}`"
                :aria-pressed="Boolean(message.reactions?.find(item => item.emoji === emoji)?.mine)"
                :disabled="state !== 'live' || Boolean(reacting)" @click="react(message, emoji)">
                {{ emoji }}<span v-if="message.reactions?.find(item => item.emoji === emoji)?.count">{{ message.reactions.find(item => item.emoji === emoji).count }}</span>
              </button>
              <button v-if="message.type === 'fan' && !message.mine" type="button" class="chat-flag"
                :disabled="state !== 'live' || sending" @click="beginReply(message)">Rispondi</button>
              <button v-if="!message.mine && !reported.includes(message.id)" type="button" class="chat-flag"
                :aria-expanded="reportId === message.id" :disabled="state !== 'live'" @click="beginReport(message)">Segnala</button>
            </div>
            <form v-if="reportId === message.id" class="chat-report" @submit.prevent="report">
              <label>Motivo <select v-model="reason"><option v-for="(label, value) in chatReasons" :key="value" :value="value">{{ label }}</option></select></label>
              <button type="submit" :disabled="reporting || state !== 'live'">Invia segnalazione</button>
              <button type="button" @click="reportId = null">Annulla</button>
            </form>
          </div>
        </article>
      </div>

      <footer v-if="state !== 'disabled'" class="chat-footer">
        <button v-if="unread || archived" type="button" class="chat-new" :disabled="state !== 'live'" @click="archived ? syncHistory() : scrollBottom()">
          {{ archived ? 'Torna ai messaggi recenti' : `Nuovi messaggi (${unread}) ↓` }}
        </button>
        <p v-if="state === 'reconnecting'" role="status">Connessione interrotta. Riconnessione in corso…</p>
        <p v-if="error" role="alert">{{ error }}</p>
        <p v-if="notice" role="status">{{ notice }}</p>
        <button v-if="state === 'error'" type="button" @click="syncHistory">Ricarica i messaggi</button>
        <button v-else-if="state === 'live' && !canWrite" type="button" class="chat-verify" @click="emit('verify')">Accedi o crea un profilo verificato per partecipare</button>
        <div v-if="canWrite && replyTo" class="chat-reply-draft">
          <span>Risposta a {{ replyTo.nickname }}</span>
          <button type="button" :disabled="sending" @click="replyTo = null">Annulla risposta</button>
        </div>
        <form v-if="canWrite && state !== 'error'" class="chat-composer" @submit.prevent="send">
          <input ref="composer" v-model="draft" type="text" aria-label="Scrivi un messaggio" aria-describedby="match-chat-count"
            placeholder="Scrivi alla tribuna…" maxlength="360" enterkeyhint="send" autocomplete="off" />
          <button type="submit" :disabled="sending || state !== 'live' || !draft.trim() || length > 180">{{ sending ? 'Invio…' : 'Invia' }}</button>
          <small id="match-chat-count" :class="{ 'chat-over-limit': length > 180 }">{{ length }}/180</small>
        </form>
      </footer>
    </section>
  </dialog>
</template>

<style scoped>
.chat-dialog {
  position: fixed;
  inset: 0;
  margin: auto;
  width: min(36rem, calc(100% - 2rem));
  height: min(43rem, 88dvh);
  max-height: var(--chat-viewport, 100dvh);
  padding: 0;
  color: var(--ink);
  background: var(--bg);
  border: 1px solid var(--line);
  border-radius: 24px;
  overflow: hidden;
}
.chat-dialog::backdrop { background: #09101cb3; }
.chat-dialog[open] { animation: chat-in .18s ease-out; }
@keyframes chat-in { from { opacity: 0; transform: translateY(14px); } }
.chat-sheet { display: flex; flex-direction: column; height: 100%; min-height: 0; }
.chat-header { display: flex; align-items: center; gap: .75rem; padding: 12px 16px; border-bottom: 1px solid var(--line); background: var(--card); }
.chat-header > div { flex: 1; min-width: 0; }
.chat-header h2 { margin: 0; font-size: 1.2rem; }
.chat-header p { margin: 3px 0 0; color: var(--muted); font-size: .8rem; overflow-wrap: anywhere; }
.chat-live, .chat-badge { font-size: .65rem; font-weight: 850; color: var(--red-text); background: var(--red-soft); border-radius: 20px; padding: 4px 8px; white-space: nowrap; }
.chat-live i { display: inline-block; width: 7px; height: 7px; border-radius: 50%; background: currentColor; margin-right: 3px; }
.chat-dialog button { cursor: pointer; font-family: inherit; }
.chat-dialog button:disabled { cursor: default; opacity: .55; }
.chat-dialog button:focus-visible, .chat-dialog input:focus-visible, .chat-dialog select:focus-visible { outline: 2px solid var(--red-text); outline-offset: 2px; }
.chat-close { width: 44px; height: 44px; border-radius: 50%; border: 1px solid var(--line); background: var(--bg); color: var(--ink); font-size: 1.5rem; flex: none; }
.chat-messages { position: relative; flex: 1; overflow-y: auto; overscroll-behavior: contain; overflow-anchor: none; padding: 12px; }
.chat-empty { padding: 2rem 1rem; text-align: center; color: var(--muted); }
.chat-message { display: flex; gap: 7px; align-items: flex-start; margin: 0 0 12px; }
.chat-mine { flex-direction: row-reverse; }
.chat-avatar { display: grid; place-items: center; border-radius: 50%; width: 28px; height: 28px; flex: none; background: var(--red-soft); color: var(--red-text); font-weight: 850; font-size: .75rem; }
.chat-bubble { min-width: 0; max-width: min(calc(100% - 35px), 29rem); border: 1px solid var(--line); border-radius: 15px; background: var(--card); padding: 8px 9px 5px; }
.chat-mine .chat-bubble { background: var(--red-soft); }
.chat-official .chat-bubble { border-color: var(--red-text); }
.chat-byline { display: flex; align-items: center; gap: 7px; flex-wrap: wrap; font-size: .75rem; overflow-wrap: anywhere; }
.chat-byline time { color: var(--muted); }
.chat-bubble p { margin: 6px 0 3px; font-size: .9rem; line-height: 1.4; overflow-wrap: anywhere; white-space: pre-wrap; }
.chat-bubble .chat-reply-reference { color: var(--muted); font-size: .75rem; }
.chat-reactions { display: flex; flex-wrap: wrap; align-items: center; gap: 2px; }
.chat-reactions button { min-height: 36px; min-width: 34px; color: var(--muted); border-radius: 10px; border: 1px solid transparent; padding: 0 3px; background: transparent; }
.chat-reactions button[aria-pressed=true] { border-color: var(--red-text); background: var(--red-soft); color: var(--red-text); }
.chat-reactions button span { font-size: .7rem; margin-left: 2px; }
.chat-reactions .chat-flag { font-size: .72rem; text-decoration: underline; padding-inline: 4px; }
.chat-report { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; padding: 8px 0; font-size: .75rem; }
.chat-report select, .chat-report button, .chat-footer button, .chat-older { min-height: 42px; border: 1px solid var(--line); border-radius: 10px; padding: 5px 9px; color: var(--ink); background: var(--card); font-weight: 700; }
.chat-older { display: block; margin: 0 auto 12px; }
.chat-footer { position: relative; padding: 8px 12px calc(8px + env(safe-area-inset-bottom)); border-top: 1px solid var(--line); background: var(--bg); }
.chat-footer p { font-size: .8rem; margin: 3px 0 7px; }
.chat-footer .chat-new { position: absolute; right: 12px; bottom: calc(100% + 7px); background: var(--red-soft); color: var(--red-text); }
.chat-footer .chat-verify { width: 100%; }
.chat-reply-draft { display: flex; align-items: center; justify-content: space-between; gap: 8px; margin-bottom: 6px; color: var(--muted); font-size: .8rem; }
.chat-reply-draft span { min-width: 0; overflow-wrap: anywhere; }
.chat-composer { display: grid; grid-template-columns: minmax(0, 1fr) auto; gap: 4px 8px; }
.chat-composer input { width: 100%; min-height: 44px; border: 1px solid var(--line); border-radius: 12px; padding: 0 10px; font-size: 16px; color: var(--ink); background: var(--card); }
.chat-composer button { background: var(--red); color: var(--on-primary); min-width: 60px; }
.chat-composer small { color: var(--muted); text-align: right; }
.chat-composer .chat-over-limit { color: var(--red-text); font-weight: 800; }
@media (max-width: 640px) {
  .chat-dialog { inset: auto 0 var(--chat-bottom, 0); margin: 0; width: 100%; max-width: 100%; height: min(88dvh, var(--chat-viewport, 100dvh)); border-radius: 20px 20px 0 0; }
}
@media (prefers-reduced-motion: reduce) { .chat-dialog[open] { animation: none; } }
</style>
