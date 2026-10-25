<template>
  <article class="voting-settings dashboard-card">
    <header>
      <h3>Organizzatori e chiusura votazioni</h3>
      <p>Alla chiusura vengono assegnati i premi e inviato il riepilogo WhatsApp agli organizzatori abilitati.</p>
    </header>
    <p v-if="loading" role="status">Caricamento configurazione…</p>
    <p v-if="error" class="error" role="alert">{{ error }}</p>
    <button v-if="!loaded && !loading" class="btn outline" type="button" @click="load">Riprova caricamento</button>
    <form v-if="loaded" @submit.prevent="save">
      <fieldset :disabled="saving || loading || testing">
        <label class="field">
          Template SID Twilio della società
          <input v-model.trim="settings.template_sid" type="text" placeholder="HX…" maxlength="34"
            pattern="HX[0-9a-fA-F]{32}" :required="hasEnabledRecipients" autocomplete="off" spellcheck="false" />
          <small>Il template WhatsApp approvato deve prevedere le 8 variabili per MVP, percentuale e tre premi con i codici vincitori.</small>
        </label>
        <div class="template-test">
          <div class="template-test-controls">
            <label class="field">Numero WhatsApp di prova
              <input v-model.trim="testPhone" type="tel" placeholder="+39 333 1234567" maxlength="40" autocomplete="off" @keydown.enter.prevent="testTemplate" />
            </label>
            <button class="btn outline" type="button" :disabled="!canTest" @click="testTemplate">{{ testing ? 'Invio…' : 'Testa' }}</button>
          </div>
          <small>Invia un WhatsApp reale con il Template SID inserito e risultati di esempio. Il numero di prova non viene salvato tra gli organizzatori.</small>
          <p v-if="testMessage" class="success-message" role="status">{{ testMessage }}</p>
          <p v-if="testError" class="error" role="alert">{{ testError }}</p>
        </div>
        <fieldset class="draw-modes">
          <legend>Assegnazione dei premi</legend>
          <label class="radio-option">
            <input v-model="settings.draw_mode" type="radio" name="voting-draw-mode" value="random" />
            <span><strong>Estrazione casuale</strong><small>Come nella lotteria: tra i codici di chi ha votato alla partita con un profilo tifoso e telefono verificato.</small></span>
          </label>
          <label class="radio-option">
            <input v-model="settings.draw_mode" type="radio" name="voting-draw-mode" value="leaderboard" />
            <span><strong>Estrai 3 migliori in classifica tifoso</strong><small>Primi tre per punti nella classifica monete, tra i tifosi che hanno votato alla partita. Un premio per tifoso; a parità vale l’ordine della classifica.</small></span>
          </label>
        </fieldset>
        <fieldset class="ticket-visibility">
          <legend>Biglietto lotteria nel profilo tifoso</legend>
          <div class="recipient-status">
            <label class="enabled"><input v-model="settings.show_lottery_ticket" type="radio" name="voting-lottery-ticket" :value="true" /> Visibile dopo il voto</label>
            <label class="enabled"><input v-model="settings.show_lottery_ticket" type="radio" name="voting-lottery-ticket" :value="false" /> Nascosto</label>
          </div>
          <small>Nel profilo di /{{ organizationSlug }}/new, mostra il codice e il QR del biglietto dopo il voto. Il biglietto rimane aperto anche dopo la chiusura delle votazioni.</small>
        </fieldset>
        <div class="recipients-heading"><h4>Destinatari WhatsApp</h4><span>{{ settings.recipients.length }}/20</span></div>
        <p v-if="!settings.recipients.length" class="muted">Aggiungi i numeri degli organizzatori autorizzati a ricevere i risultati della società.</p>
        <div v-for="(recipient, index) in settings.recipients" :key="index" class="recipient-row">
          <label class="field">Nome organizzatore
            <input v-model.trim="recipient.name" type="text" maxlength="100" placeholder="Es. Responsabile evento" />
          </label>
          <label class="field">Numero WhatsApp
            <input v-model.trim="recipient.phone" type="tel" placeholder="+39 333 1234567" required :aria-label="`Numero WhatsApp organizzatore ${index + 1}`" />
          </label>
          <div class="recipient-controls">
            <fieldset class="recipient-status" :aria-label="`Ricezione WhatsApp organizzatore ${index + 1}`">
              <label class="enabled"><input v-model="recipient.enabled" type="radio" :name="`voting-recipient-enabled-${index}`" :value="true" /> Abilitato</label>
              <label class="enabled"><input v-model="recipient.enabled" type="radio" :name="`voting-recipient-enabled-${index}`" :value="false" /> Disabilitato</label>
            </fieldset>
            <button class="btn outline" type="button" :aria-label="`Rimuovi organizzatore ${index + 1}`" @click="settings.recipients.splice(index, 1)">Rimuovi</button>
          </div>
        </div>
        <div class="settings-actions">
          <button class="btn outline" type="button" :disabled="settings.recipients.length >= 20" @click="addRecipient">Aggiungi organizzatore</button>
          <button class="btn primary" type="submit">{{ saving ? 'Salvataggio…' : 'Salva configurazione' }}</button>
        </div>
      </fieldset>
      <p v-if="saved" class="success-message" role="status">Configurazione della società salvata.</p>
    </form>
  </article>
</template>

<script setup>
import { computed, onUnmounted, reactive, ref, watch } from 'vue';
import { apiClient } from '../../api';

const props = defineProps({
  authHeaders: { type: Object, required: true },
  organizationSlug: { type: String, default: '' },
});
const emit = defineEmits(['unauthorized']);
const settings = reactive({ template_sid: '', draw_mode: 'random', show_lottery_ticket: false, recipients: [] });
const loading = ref(false);
const loaded = ref(false);
const saving = ref(false);
const saved = ref(false);
const error = ref('');
const testPhone = ref('');
const testing = ref(false);
const testMessage = ref('');
const testError = ref('');
const hasEnabledRecipients = computed(() => settings.recipients.some(recipient => recipient.enabled));
const canTest = computed(() => loaded.value && !loading.value && !saving.value && !testing.value && Boolean(settings.template_sid.trim()) && Boolean(testPhone.value.trim()));
let generation = 0;

async function load() {
  const current = ++generation;
  loading.value = true;
  loaded.value = false;
  error.value = '';
  saved.value = false;
  testPhone.value = '';
  testMessage.value = '';
  testError.value = '';
  testing.value = false;
  try {
    const { data } = await apiClient.get('/admin/voting-settings', props.authHeaders);
    if (current !== generation) return;
    Object.assign(settings, data, { show_lottery_ticket: data?.show_lottery_ticket === true });
    loaded.value = true;
  } catch (e) {
    if (current !== generation) return;
    if (e?.response?.status === 401) emit('unauthorized');
    error.value = 'Impossibile caricare la configurazione delle notifiche.';
  } finally {
    if (current === generation) loading.value = false;
  }
}

function addRecipient() {
  settings.recipients.push({ name: '', phone: '', enabled: true });
  saved.value = false;
}

async function testTemplate() {
  if (!canTest.value) return;
  testMessage.value = '';
  testError.value = '';
  const templateSID = settings.template_sid.trim();
  const phone = testPhone.value.replace(/[\s()-]/g, '');
  if (!/^HX[0-9a-fA-F]{32}$/.test(templateSID)) {
    testError.value = 'Inserisci un Template SID valido: HX seguito da 32 caratteri esadecimali.';
    return;
  }
  if (!/^\+[1-9][0-9]{6,14}$/.test(phone)) {
    testError.value = 'Inserisci il numero di prova con prefisso internazionale, per esempio +39 333 1234567.';
    return;
  }
  const current = generation;
  testing.value = true;
  try {
    const { data } = await apiClient.post('/admin/voting-settings/test', { template_sid: templateSID, phone }, props.authHeaders);
    if (current !== generation) return;
    testMessage.value = data?.message || 'Test accettato da Twilio. Controlla WhatsApp al numero indicato.';
  } catch (e) {
    if (current !== generation) return;
    if (e?.response?.status === 401) emit('unauthorized');
    testError.value = e?.response?.data?.message || 'Esito del test incerto. Verifica WhatsApp prima di riprovare.';
  } finally {
    if (current === generation) testing.value = false;
  }
}

async function save() {
  if (saving.value || testing.value) return;
  const current = generation;
  saving.value = true;
  saved.value = false;
  error.value = '';
  try {
    const { data } = await apiClient.put('/admin/voting-settings', {
      template_sid: settings.template_sid,
      draw_mode: settings.draw_mode,
      show_lottery_ticket: settings.show_lottery_ticket === true,
      recipients: settings.recipients.map(recipient => ({ ...recipient })),
    }, props.authHeaders);
    if (current !== generation) return;
    Object.assign(settings, data, { show_lottery_ticket: data?.show_lottery_ticket === true });
    saved.value = true;
  } catch (e) {
    if (current !== generation) return;
    if (e?.response?.status === 401) emit('unauthorized');
    error.value = e?.response?.data?.message || 'Impossibile salvare la configurazione. Riprova.';
  } finally {
    saving.value = false;
  }
}

watch(() => props.organizationSlug, load, { immediate: true });
watch(settings, () => { saved.value = false; }, { deep: true, flush: 'sync' });
watch([testPhone, () => settings.template_sid], () => { testMessage.value = ''; testError.value = ''; }, { flush: 'sync' });
onUnmounted(() => { generation++; });
</script>

<style scoped>
.voting-settings { grid-column: 1 / -1; }
.voting-settings h3, .voting-settings h4 { margin: 0; }
.voting-settings header p, .muted, small { color: var(--admin-text-secondary, #94a3b8); }
.voting-settings fieldset { border: 0; padding: 0; margin: 0; min-width: 0; }
.field { display: grid; gap: .5rem; font-size: .9rem; }
.field input { width: 100%; min-width: 0; box-sizing: border-box; padding: .75rem; border-radius: .6rem; border: 1px solid #334155; background: #0f172a; color: #f1f5f9; }
small { display: block; font-size: .8rem; line-height: 1.5; font-weight: 400; }
.template-test { margin-top: 1rem; padding: 1rem; border: 1px solid #cbd5e1; border-radius: .65rem; }
.template-test-controls { display: grid; grid-template-columns: minmax(0, 1fr) auto; align-items: end; gap: .75rem; margin-bottom: .65rem; }
.voting-settings .draw-modes, .voting-settings .ticket-visibility { margin: 1.5rem 0; }
.draw-modes legend, .ticket-visibility legend { margin-bottom: .75rem; font-weight: 600; }
.ticket-visibility small { margin-top: .65rem; }
.radio-option { display: flex; align-items: flex-start; gap: .65rem; padding: .85rem; margin-top: .5rem; background: rgba(148,163,184,.06); border: 1px solid #334155; border-radius: .65rem; cursor: pointer; }
.radio-option input { width: auto; accent-color: #8b5cf6; }
.recipients-heading, .settings-actions { display: flex; align-items: center; justify-content: space-between; gap: .75rem; flex-wrap: wrap; }
.recipients-heading span { font-size: .8rem; color: #94a3b8; }
.recipient-row { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); align-items: end; gap: .75rem; margin-top: 1rem; }
.recipient-controls { grid-column: 1 / -1; display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 1rem; padding-bottom: .5rem; }
.recipient-status { display: flex; flex-wrap: wrap; gap: .75rem; }
.enabled { display: flex; align-items: center; gap: .6rem; min-height: 44px; box-sizing: border-box; padding: .5rem .75rem; border: 1px solid #cbd5e1; border-radius: .6rem; background: #fff; color: #334155; font-size: .85rem; white-space: nowrap; cursor: pointer; }
.enabled:has(input:checked) { border-color: #0284c7; background: #f0f9ff; color: #075985; }
.recipient-status input[type="radio"] { appearance: none; display: inline-block; flex: 0 0 20px; width: 20px; height: 20px; min-width: 20px; min-height: 20px; padding: 0; margin: 0; box-sizing: border-box; border: 2px solid #64748b; border-radius: 50%; background: #fff; cursor: pointer; }
.recipient-status input[type="radio"]:checked { border-color: #0284c7; background: radial-gradient(circle, #0284c7 0 42%, #fff 45%); }
.recipient-status input[type="radio"]:focus-visible { outline: 2px solid #0284c7; outline-offset: 3px; }
.enabled:has(input:disabled) { opacity: .55; cursor: not-allowed; }
.recipient-status input[type="radio"]:disabled { cursor: not-allowed; }
.settings-actions { margin-top: 1.25rem; }
.error { color: #fca5a5; }
.success-message { color: #86efac; }
@media (max-width: 440px) { .recipient-row, .template-test-controls { grid-template-columns: 1fr; } }
</style>
