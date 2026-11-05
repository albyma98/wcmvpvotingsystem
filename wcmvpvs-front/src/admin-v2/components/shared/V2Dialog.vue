<template><Teleport to="body"><div v-if="open" class="admin-v2 v2-dialog-backdrop" @click.self="close"><section ref="panel" class="v2-dialog" role="dialog" aria-modal="true" :aria-labelledby="titleId" tabindex="-1" @keydown="keydown"><header class="v2-panel-heading"><h2 :id="titleId">{{ title }}</h2><button class="v2-button v2-icon-button" type="button" aria-label="Chiudi finestra" :disabled="busy" @click="close"><V2Icon name="close" /></button></header><div class="v2-dialog-body"><slot /></div><footer v-if="$slots.footer" class="v2-dialog-footer"><slot name="footer" /></footer></section></div></Teleport></template>
<script setup>
import { nextTick, onBeforeUnmount, ref, useId, watch } from 'vue';
import V2Icon from './V2Icon.vue';
const props = defineProps({ open: Boolean, title: String, busy: Boolean });
const emit = defineEmits(['close']);
const titleId = useId(); const panel = ref(null); let previous; let overflow; let siblings = [];
function close() { if (!props.busy) emit('close'); }
function keydown(e) {
 if (e.key === 'Escape') { e.preventDefault(); close(); }
 if (e.key !== 'Tab') return;
 const items = [...panel.value.querySelectorAll('button:not(:disabled),a[href],input:not(:disabled),select:not(:disabled),textarea:not(:disabled),[tabindex="0"]')].filter(el => el.getClientRects().length);
 const first = items[0], last = items.at(-1);
 if (!first) { e.preventDefault(); panel.value.focus(); }
 else if (e.shiftKey && (document.activeElement === first || document.activeElement === panel.value)) { e.preventDefault(); last.focus(); }
 else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first.focus(); }
}
function release() {
 if (overflow !== undefined) { document.body.style.overflow = overflow; overflow = undefined; }
 siblings.forEach(([el, inert]) => { el.inert = inert; }); siblings = [];
 if (previous?.isConnected) previous.focus(); previous = null;
}
watch(() => props.open, async open => {
 if (!open) { release(); return; }
 previous = document.activeElement; overflow = document.body.style.overflow; document.body.style.overflow = 'hidden';
 await nextTick(); if (!props.open) return;
 const backdrop = panel.value?.parentElement;
 siblings = [...document.body.children].filter(el => el !== backdrop && !['SCRIPT','STYLE'].includes(el.tagName)).map(el => [el, el.inert]);
 siblings.forEach(([el]) => { el.inert = true; }); panel.value?.focus();
}, { immediate: true });
onBeforeUnmount(release);
</script>
