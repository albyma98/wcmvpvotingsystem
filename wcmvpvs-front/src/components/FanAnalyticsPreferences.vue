<template>
  <div class="analytics-preferences ph-no-capture">
    <button ref="triggerEl" class="analytics-preferences-link" type="button" @click="openPreferences">Preferenze analisi</button>
    <Teleport to="body">
      <section v-if="isOpen" ref="noticeEl" class="analytics-notice ph-no-capture" :style="themeStyle" role="dialog" aria-labelledby="fan-analytics-title" tabindex="-1" @keydown.esc.prevent="closePreferences">
        <header><h2 id="fan-analytics-title">Ci aiuti a migliorare l’esperienza?</h2><button class="analytics-close" type="button" aria-label="Chiudi preferenze analisi" @click="closePreferences">✕</button></header>
        <p>Con il tuo consenso usiamo PostHog per misurare azioni, tempi e ritorni, con un identificativo tecnico del browser. Telefono, OTP, nickname, risposte libere e codice del biglietto sono esclusi. Le sessioni non vengono registrate.</p>
        <p>La scelta è facoltativa: voto, giochi e profilo funzionano anche senza queste analisi. Puoi cambiarla da “Preferenze analisi”.</p>
        <details class="analytics-details"><summary>Informativa privacy</summary><div class="analytics-policy"><template v-for="(block, index) in legalBlocks" :key="index"><h3 v-if="block.type === 'heading' || block.type === 'title'">{{ block.text }}</h3><p v-else>{{ block.text }}</p></template></div></details>
        <p v-if="browserPreference" class="analytics-browser-note">Il browser richiede di non essere tracciato: l’invio a PostHog resta disattivato.</p>
        <div class="analytics-actions"><button type="button" @click="choose(false)">Continua senza analisi</button><button type="button" :disabled="browserPreference" @click="choose(true)">Accetta analisi</button></div>
      </section>
    </Teleport>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue';
import { getFanAnalyticsConsent, setFanAnalyticsConsent } from '../lib/posthog';
import { legalTextBlocks } from '../fanRegistration';
import privacyText from '../legal/fan-privacy.md?raw';

const props = defineProps({ themeStyle: { type: Object, default: () => ({}) }, clubName: { type: String, default: '' } });
const emit = defineEmits(['changed']);
const consent = ref(getFanAnalyticsConsent());
const isOpen = ref(consent.value === null);
const triggerEl = ref(null);
const noticeEl = ref(null);
const legalBlocks = computed(() => legalTextBlocks(privacyText, props.clubName));
const browserPreference = computed(() => typeof navigator !== 'undefined' && (navigator.globalPrivacyControl === true || navigator.doNotTrack === '1' || navigator.doNotTrack === 'yes'));
async function openPreferences() {
  consent.value = getFanAnalyticsConsent();
  isOpen.value = true;
  await nextTick();
  noticeEl.value?.focus({ preventScroll: true });
}
function closePreferences() { isOpen.value = false; triggerEl.value?.focus({ preventScroll: true }); }
function choose(value) {
  if (value && browserPreference.value) return;
  const previous = getFanAnalyticsConsent();
  setFanAnalyticsConsent(value);
  consent.value = getFanAnalyticsConsent();
  if (consent.value !== previous) emit('changed', consent.value);
  closePreferences();
}
function refreshPreference() { consent.value = getFanAnalyticsConsent(); }
function synchronizePreference(event) {
  if (event?.key && event.key !== 'arenaboostx:fan-analytics-consent:v1') return;
  const previous = consent.value;
  refreshPreference();
  if (consent.value !== previous) emit('changed', consent.value);
}
onMounted(() => {
  window.addEventListener('arenaboostx:fan-analytics-consent', refreshPreference);
  window.addEventListener('storage', synchronizePreference);
});
onBeforeUnmount(() => {
  window.removeEventListener('arenaboostx:fan-analytics-consent', refreshPreference);
  window.removeEventListener('storage', synchronizePreference);
});
</script>

<style scoped>
.analytics-preferences { margin-top: 4px; text-align: center; }
.analytics-preferences-link { min-height: 36px; border: 0; background: transparent; color: var(--muted); font: inherit; font-size: 11px; text-decoration: underline; cursor: pointer; }
.analytics-notice { position: fixed; z-index: 5000; left: 50%; bottom: max(12px, env(safe-area-inset-bottom)); transform: translateX(-50%); width: min(440px, calc(100vw - 24px)); max-height: calc(100dvh - 32px); overflow-y: auto; box-sizing: border-box; padding: 17px; border: 1px solid var(--line, #d4dae2); border-radius: 18px; background: var(--card, #fff); color: var(--ink, #172334); box-shadow: 0 10px 48px #11182740; font-family: Inter, system-ui, sans-serif; }
.analytics-notice header { display: flex; align-items: flex-start; gap: 12px; justify-content: space-between; }
.analytics-notice h2 { margin: 0; font-size: 16px; line-height: 1.35; }
.analytics-notice p { margin: 9px 0; font-size: 12px; line-height: 1.55; }
.analytics-close { flex: none; width: 32px; height: 32px; border: 1px solid var(--line, #d4dae2); border-radius: 9px; background: transparent; color: inherit; cursor: pointer; }
.analytics-details { margin-top: 10px; font-size: 12px; }
.analytics-details summary { padding: 5px 0; cursor: pointer; text-decoration: underline; }
.analytics-policy { max-height: 180px; overflow-y: auto; padding: 6px 10px; border: 1px solid var(--line, #d4dae2); border-radius: 10px; }
.analytics-policy h3 { margin: 10px 0 5px; font-size: 12px; }
.analytics-browser-note { font-weight: 700; }
.analytics-actions { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; margin-top: 12px; }
.analytics-actions button { min-height: 44px; padding: 10px 8px; border: 1px solid var(--line, #d4dae2); border-radius: 11px; background: var(--bg, #f4f6f8); color: inherit; font: inherit; font-size: 12px; font-weight: 750; cursor: pointer; }
.analytics-actions button:disabled { opacity: .5; cursor: default; }
.analytics-notice button:focus-visible, .analytics-preferences-link:focus-visible, .analytics-details summary:focus-visible { outline: 3px solid var(--red-text, #1458a6); outline-offset: 3px; }
</style>
