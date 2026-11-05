<template>
  <Teleport to="body">
    <div v-if="open" class="admin-portal admin-dialog-scope" @click.self="requestClose">
      <section
        ref="panel"
        class="admin-dialog__surface"
        role="dialog"
        aria-modal="true"
        :aria-label="label"
        :aria-busy="busy"
        tabindex="-1"
      >
        <slot />
      </section>
    </div>
  </Teleport>
</template>

<script setup>
import { nextTick, onBeforeUnmount, ref, watch } from 'vue';

const props = defineProps({
  open: { type: Boolean, default: false },
  label: { type: String, default: 'Finestra di dialogo' },
  busy: { type: Boolean, default: false },
});
const emit = defineEmits(['close']);
const panel = ref(null);
let previousFocus;
let previousOverflow;
let disposed = false;
let dialogActive = false;

function focusableElements() {
  return [...(panel.value?.querySelectorAll('button:not(:disabled), a[href], input:not(:disabled), select:not(:disabled), textarea:not(:disabled), summary, [contenteditable="true"], [tabindex]:not([tabindex="-1"])') || [])]
    .filter(element => element.getClientRects().length && !element.closest('[inert]'));
}
function requestClose() { if (!props.busy) emit('close'); }
function handleKeydown(event) {
  if (!dialogActive || !props.open || !panel.value) return;
  if (event.key === 'Escape') { event.preventDefault(); requestClose(); return; }
  if (event.key !== 'Tab') return;
  const elements = focusableElements();
  if (!elements.length) { event.preventDefault(); panel.value?.focus(); return; }
  const first = elements[0];
  const last = elements.at(-1);
  const focusOutside = !panel.value.contains(document.activeElement);
  if (event.shiftKey && (document.activeElement === first || document.activeElement === panel.value || focusOutside)) {
    event.preventDefault(); last.focus();
  } else if (!event.shiftKey && (document.activeElement === last || focusOutside)) {
    event.preventDefault(); first.focus();
  }
}
function containFocus(event) {
  if (!dialogActive || !props.open || !panel.value || panel.value.contains(event.target)) return;
  (focusableElements()[0] || panel.value).focus();
}
function releaseDialog() {
  dialogActive = false;
  document.removeEventListener('keydown', handleKeydown, true);
  document.removeEventListener('focusin', containFocus, true);
  if (previousOverflow !== undefined) {
    document.body.style.overflow = previousOverflow;
    previousOverflow = undefined;
  }
  const target = previousFocus;
  previousFocus = undefined;
  nextTick(() => { if ((disposed || !props.open) && target?.isConnected) target.focus(); });
}
watch(() => props.open, async open => {
  if (typeof document === 'undefined') return;
  if (!open) { releaseDialog(); return; }
  previousFocus = document.activeElement;
  previousOverflow = document.body.style.overflow;
  document.body.style.overflow = 'hidden';
  dialogActive = true;
  document.addEventListener('keydown', handleKeydown, true);
  document.addEventListener('focusin', containFocus, true);
  await nextTick();
  if (props.open) (focusableElements()[0] || panel.value)?.focus();
}, { immediate: true });
onBeforeUnmount(() => { disposed = true; if (typeof document !== 'undefined') releaseDialog(); });
</script>

<style scoped>
.admin-dialog-scope { position: fixed; inset: 0; z-index: 1000; display: flex; align-items: center; justify-content: center; min-height: 0; padding: 24px; overflow-y: auto; overscroll-behavior: contain; background: #10172b80; backdrop-filter: blur(4px); color: var(--text-primary, #172033); font-family: var(--font-body, system-ui, sans-serif); }
.admin-dialog__surface { width: min(100%, 640px); max-height: calc(100vh - 48px); max-height: calc(100dvh - 48px); min-width: 0; overflow-y: auto; overscroll-behavior: contain; border: 1px solid #e2e6ef; border-radius: 16px; background: white; box-shadow: 0 24px 80px #0f172a40; }
.admin-dialog__surface:focus { outline: none; }
@media (max-width: 599px) { .admin-dialog-scope { padding: 16px; } .admin-dialog__surface { max-height: calc(100dvh - 32px); border-radius: 12px; } }
</style>
