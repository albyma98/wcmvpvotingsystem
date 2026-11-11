<script setup>
import { defineAsyncComponent, nextTick, ref, watch } from 'vue';
import { connectMatchChat } from './matchChatApi';

const MatchChatDialog = defineAsyncComponent(() => import('./MatchChatDialog.vue'));
const props = defineProps({ slug: String, eventId: Number, eventLabel: String });
const emit = defineEmits(['verify', 'track']);
const enabled = ref(false);
const opened = ref(false);
const trigger = ref(null);

watch(() => [props.slug, props.eventId], ([slug, eventId], _, cleanup) => {
  enabled.value = false;
  opened.value = false;
  if (!slug || !eventId) return;
  cleanup(connectMatchChat(slug, eventId, {
    statusOnly: true,
    onUpdate: update => { enabled.value = update.enabled === true; },
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
  <button v-if="enabled" ref="trigger" type="button" class="match-chat-trigger" aria-haspopup="dialog" @click="open">
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
      <path d="M5 5h14a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2h-7l-5 4v-4H5a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2Z"/><path d="M8 11h8"/>
    </svg>
    <span>Chat Live</span>
  </button>
  <MatchChatDialog v-if="opened" :slug="slug" :event-id="eventId" :event-label="eventLabel" :enabled="enabled"
    @close="close" @verify="verify" @track="emit('track', $event)" />
</template>

<style scoped>
.match-chat-trigger {
  min-height: 44px;
  min-width: 44px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  padding: 0 8px;
  border: 1px solid var(--line);
  border-radius: 14px;
  background: var(--red-soft);
  color: var(--red-text);
  font: inherit;
  font-weight: 800;
  font-size: .75rem;
  cursor: pointer;
}
.match-chat-trigger svg { width: 18px; height: 18px; }
.match-chat-trigger:focus-visible { outline: 2px solid var(--red-text); outline-offset: 2px; }
@media (max-width: 460px) {
  .match-chat-trigger { flex-direction: column; gap: 1px; padding-inline: 4px; font-size: .625rem; }
}
</style>
