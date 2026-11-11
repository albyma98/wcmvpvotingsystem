<script setup>
import { defineAsyncComponent, nextTick, ref, watch } from 'vue';
import { connectMatchChat } from './matchChatApi';

const MatchChatDialog = defineAsyncComponent(() => import('./MatchChatDialog.vue'));
const props = defineProps({ slug: String, eventId: Number, eventLabel: String });
const emit = defineEmits(['verify', 'track']);
const enabled = ref(false);
const latestMessage = ref(null);
const opened = ref(false);
const trigger = ref(null);

watch(() => [props.slug, props.eventId], ([slug, eventId], _, cleanup) => {
  enabled.value = false;
  latestMessage.value = null;
  opened.value = false;
  if (!slug || !eventId) return;
  cleanup(connectMatchChat(slug, eventId, {
    statusOnly: true,
    preview: true,
    onUpdate(update) {
      if (update.type === 'status') {
        enabled.value = update.enabled === true;
        latestMessage.value = enabled.value ? update.message ?? null : null;
      } else if (update.type === 'preview' && enabled.value) {
        latestMessage.value = update.message ?? null;
      }
    },
    onDisconnect() { latestMessage.value = null; },
  }));
}, { immediate: true });

function open() {
  opened.value = true;
  emit('track', 'match_chat_opened');
}

function close(restoreFocus = true) {
  opened.value = false;
  emit('track', 'match_chat_closed');
  if (restoreFocus) nextTick(() => trigger.value?.focus());
}

function verify() {
  close(false);
  emit('verify');
}
</script>

<template>
  <button v-if="enabled" ref="trigger" type="button" class="match-chat-trigger" aria-haspopup="dialog" :aria-expanded="opened" @click="open">
    <svg class="match-chat-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
      <path d="M5 5h14a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2h-7l-5 4v-4H5a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2Z"/><path d="M8 11h8"/>
    </svg>
    <span class="match-chat-copy">
      <b>Chat Live</b>
      <span class="match-chat-preview">
        <template v-if="latestMessage"><strong>{{ latestMessage.nickname }}:</strong> {{ latestMessage.text }}</template>
        <template v-else>Apri la conversazione</template>
      </span>
    </span>
    <svg class="match-chat-chevron" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m9 6 6 6-6 6"/></svg>
  </button>
  <MatchChatDialog v-if="opened" :slug="slug" :event-id="eventId" :event-label="eventLabel" :enabled="enabled"
    @close="close" @verify="verify" @track="emit('track', $event)" />
</template>

<style scoped>
.match-chat-trigger {
  min-height: 56px;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 9px;
  padding: 8px 12px;
  border: 1px solid var(--line);
  border-radius: var(--home-radius);
  background: var(--card);
  box-shadow: var(--home-shadow);
  color: var(--ink);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.match-chat-trigger:hover { border-color: var(--red-text); }
.match-chat-icon, .match-chat-chevron { width: 19px; height: 19px; flex: none; color: var(--red-text); }
.match-chat-copy { display: grid; flex: 1; min-width: 0; gap: 2px; font-size: .875rem; }
.match-chat-copy b { font-weight: 800; }
.match-chat-preview { overflow: hidden; color: var(--ink); font-size: .75rem; text-overflow: ellipsis; white-space: nowrap; }
.match-chat-trigger:focus-visible { outline: 2px solid var(--red-text); outline-offset: 2px; }
</style>
