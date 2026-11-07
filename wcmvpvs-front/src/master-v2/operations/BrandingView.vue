<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue';
import { masterApi } from '../api';
import { useMasterWorkspace } from '../workspace';
import MasterIcon from '../ui/MasterIcon.vue';
import { prepareReportLogo } from './reportLogo';

const ENDPOINT = '/admin/master/report-branding';
const form = reactive({ company_name: 'ArenaBoostX', logo_data: '' });
const saved = ref(null);
const loaded = computed(() => Boolean(saved.value));
const loading = ref(false);
const saving = ref(false);
const processing = ref(false);
const error = ref('');
const nameError = ref('');
const imageError = ref('');
const imageInfo = ref('');
const success = ref('');
const fileInput = ref(null);
const controller = new AbortController();
let imageController;

const dirty = computed(() => loaded.value && (form.company_name !== saved.value.company_name || form.logo_data !== saved.value.logo_data));
const busy = computed(() => loading.value || saving.value || processing.value);
const { useDirty } = useMasterWorkspace();
useDirty(computed(() => dirty.value || processing.value || saving.value));

function applyBranding(data) {
  saved.value = {
    company_name: typeof data?.company_name === 'string' && data.company_name.trim() ? data.company_name : 'ArenaBoostX',
    logo_data: typeof data?.logo_data === 'string' ? data.logo_data : '',
  };
  Object.assign(form, saved.value);
}

async function load() {
  if (busy.value) return;
  loading.value = true;
  error.value = '';
  try {
    const { data } = await masterApi.get(ENDPOINT, { signal: controller.signal });
    if (!controller.signal.aborted) applyBranding(data);
  } catch {
    if (!controller.signal.aborted) error.value = 'Impossibile caricare l’identità dell’emittente. Riprova.';
  } finally {
    loading.value = false;
  }
}

function clearImageFeedback() {
  imageController?.abort();
  processing.value = false;
  imageInfo.value = '';
  imageError.value = '';
  success.value = '';
  if (fileInput.value) fileInput.value.value = '';
}

function removeLogo() {
  clearImageFeedback();
  form.logo_data = '';
}

function reset() {
  clearImageFeedback();
  Object.assign(form, saved.value);
  nameError.value = '';
  error.value = '';
}

async function readLogo(event) {
  const file = event.target.files?.[0];
  if (!file) return;
  clearImageFeedback();
  imageController = new AbortController();
  const signal = imageController.signal;
  processing.value = true;
  try {
    const logo = await prepareReportLogo(file, signal);
    if (signal.aborted) return;
    form.logo_data = logo.data;
    imageInfo.value = `${logo.width} × ${logo.height} px · ${logo.kilobytes} KB`;
  } catch (cause) {
    if (!signal.aborted) imageError.value = cause.message || 'Impossibile elaborare il logo.';
  } finally {
    if (!signal.aborted) processing.value = false;
  }
}

async function save() {
  if (!dirty.value || busy.value) return;
  const companyName = form.company_name.trim();
  nameError.value = '';
  error.value = '';
  success.value = '';
  if (!companyName || [...companyName].length > 120 || /[\r\n\x00]/.test(companyName)) {
    nameError.value = 'Inserisci un nome da 1 a 120 caratteri su una sola riga.';
    return;
  }
  saving.value = true;
  try {
    const { data } = await masterApi.put(ENDPOINT, { company_name: companyName, logo_data: form.logo_data });
    applyBranding(data);
    success.value = 'Identità salvata. I prossimi report useranno la firma aggiornata.';
  } catch (cause) {
    error.value = cause?.response?.data?.message || 'Salvataggio non riuscito. Le modifiche restano disponibili.';
  } finally {
    saving.value = false;
  }
}

onMounted(load);
onBeforeUnmount(() => {
  controller.abort();
  imageController?.abort();
});
</script>

<template>
  <div v-if="!loaded" class="m-panel m-stack" :aria-busy="loading">
    <p v-if="loading" class="m-muted-text" role="status">Caricamento dell’identità dell’emittente…</p>
    <template v-else-if="error"><p class="m-error" role="alert">{{ error }}</p><button class="m-btn" type="button" @click="load">Riprova</button></template>
  </div>

  <form v-else class="branding-layout" novalidate :aria-busy="busy" @submit.prevent="save">
    <section class="m-panel m-stack branding-editor" aria-labelledby="master-branding-issuer">
      <header class="m-panel-head"><div><h2 id="master-branding-issuer">Emittente dei report</h2><p>Imposta l’identità che firma i PDF one pager e completi.</p></div><span class="m-badge m-muted">Tutte le società</span></header>
      <label class="m-field">Nome societario dell’emittente
        <input v-model="form.company_name" class="m-input" name="company_name" type="text" placeholder="ArenaBoostX" required :disabled="saving" :aria-invalid="Boolean(nameError)" aria-describedby="master-branding-name-help" @input="nameError = ''; success = ''" />
        <small id="master-branding-name-help" :class="nameError ? 'm-error' : 'm-muted-text'">{{ nameError || 'Identificativo interno. La firma del PDF mostra solo il logo.' }}</small>
      </label>
      <div class="branding-upload">
        <label class="m-field">Logo dell’emittente
          <input ref="fileInput" class="m-input branding-file" type="file" accept="image/png,image/jpeg,image/webp" :disabled="saving" :aria-invalid="Boolean(imageError)" :aria-describedby="imageError ? 'master-branding-file-help master-branding-file-error' : 'master-branding-file-help'" @change="readLogo" />
          <small id="master-branding-file-help" class="m-muted-text">PNG, JPEG o WebP. Fino a 4 MB e 2048 × 2048 px. Ottimizzazione fino a 512 KB con proporzioni e trasparenza originali.</small>
        </label>
        <p id="master-branding-file-error" v-if="imageError" class="m-error" role="alert">{{ imageError }}</p>
        <p v-if="processing" class="m-muted-text" role="status">Preparazione del logo…</p>
        <p v-else-if="imageInfo" class="m-muted-text">{{ imageInfo }}</p>
        <button v-if="form.logo_data || processing" class="m-btn" type="button" :disabled="saving" @click="removeLogo">Rimuovi logo</button>
      </div>
      <p v-if="error" class="m-error" role="alert">{{ error }}</p>
      <p v-if="success" class="m-success" role="status">{{ success }}</p>
      <footer class="m-actions branding-actions">
        <span v-if="dirty && !saving" class="m-badge m-warn" role="status">Modifiche da salvare</span>
        <button class="m-btn" type="button" :disabled="saving || (!dirty && !processing)" @click="reset">Annulla modifiche</button>
        <button class="m-btn m-primary" type="submit" :disabled="busy || !dirty">{{ saving ? 'Salvataggio…' : 'Salva identità' }}</button>
      </footer>
    </section>

    <aside class="m-panel m-stack branding-preview" aria-labelledby="master-branding-preview">
      <header class="m-panel-head"><div><h2 id="master-branding-preview">Anteprima firma</h2><p>Posizionamento del logo nel PDF.</p></div></header>
      <div class="branding-paper">
        <span class="branding-paper-label">REPORT ARENABOOSTX</span>
        <span class="branding-paper-line" aria-hidden="true"></span>
        <span class="branding-paper-line branding-paper-line-short" aria-hidden="true"></span>
        <div class="branding-signature">
          <img v-if="form.logo_data" :src="form.logo_data" alt="Logo dell’emittente" />
          <span v-else class="branding-placeholder m-muted-text"><MasterIcon name="file" :size="20" />Nessun logo caricato</span>
        </div>
      </div>
      <p class="m-muted-text branding-note">L’identità della società sportiva resta presente nei suoi report.</p>
    </aside>
  </form>
</template>

<style scoped>
.branding-layout { display: grid; grid-template-columns: minmax(0, 1.3fr) minmax(0, 1fr); align-items: start; gap: 20px; }
.branding-editor, .branding-preview { min-width: 0; }
.branding-upload { display: grid; gap: 8px; justify-items: start; }
.branding-upload label, .branding-file { width: 100%; min-width: 0; }
.branding-upload p { margin: 0; }
.branding-file::file-selector-button { font: inherit; font-weight: 600; color: var(--m-ink); background: var(--m-soft); border: 1px solid var(--m-line); border-radius: 6px; padding: 5px 9px; margin-right: 8px; cursor: pointer; }
.branding-actions { border-top: 1px solid var(--m-line); padding-top: 14px; }
.branding-paper { display: grid; gap: 9px; min-height: 190px; padding: 16px; background: var(--m-bg); border: 1px solid var(--m-line); border-radius: var(--m-radius); }
.branding-paper-label { font-size: .7rem; font-weight: 700; letter-spacing: .06em; color: var(--m-muted); }
.branding-paper-line { height: 5px; background: var(--m-line); border-radius: 5px; }
.branding-paper-line-short { width: 62%; }
.branding-signature { display: grid; place-items: center; margin-top: 20px; padding: 12px; min-height: 82px; border: 1px dashed var(--m-line); border-radius: 8px; background: var(--m-surface); }
.branding-signature img { display: block; max-width: min(100%, 190px); max-height: 72px; object-fit: contain; }
.branding-placeholder { display: flex; gap: 8px; align-items: center; font-size: .82rem; }
.branding-note { margin: 0; font-size: .86rem; }
@media (max-width: 1000px) { .branding-layout { grid-template-columns: 1fr; } }
</style>
