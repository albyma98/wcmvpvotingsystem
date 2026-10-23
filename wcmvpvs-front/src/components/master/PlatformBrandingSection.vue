<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue';
import { apiClient } from '../../api';

const ENDPOINT = '/admin/master/report-branding';
const MAX_SOURCE_BYTES = 4 * 1024 * 1024;
const MAX_ENCODED_LENGTH = 512 * 1024;
const MAX_DIMENSION = 2048;
const form = reactive({ company_name: 'ArenaBoostX', logo_data: '' });
const saved = reactive({ company_name: 'ArenaBoostX', logo_data: '' });
const fileInput = ref(null);
const loaded = ref(false);
const loading = ref(false);
const saving = ref(false);
const processingImage = ref(false);
const error = ref('');
const success = ref('');
const imageInfo = ref('');
const dirty = computed(() => form.company_name !== saved.company_name || form.logo_data !== saved.logo_data);
const busy = computed(() => loading.value || saving.value || processingImage.value);
let revision = 0;
let loadController;

function applyBranding(data) {
  const next = {
    company_name: typeof data?.company_name === 'string' && data.company_name.trim() ? data.company_name : 'ArenaBoostX',
    logo_data: typeof data?.logo_data === 'string' ? data.logo_data : '',
  };
  Object.assign(saved, next);
  Object.assign(form, next);
}

async function load() {
  const currentRevision = ++revision;
  loadController?.abort();
  loadController = new AbortController();
  loading.value = true;
  error.value = '';
  success.value = '';
  try {
    const { data } = await apiClient.get(ENDPOINT, { signal: loadController.signal });
    if (currentRevision !== revision) return;
    applyBranding(data);
    loaded.value = true;
  } catch (cause) {
    if (currentRevision === revision && cause?.code !== 'ERR_CANCELED') {
      error.value = 'Impossibile caricare l’identità della piattaforma. Riprova.';
    }
  } finally {
    if (currentRevision === revision) loading.value = false;
  }
}

function reset() {
  ++revision;
  processingImage.value = false;
  Object.assign(form, saved);
  error.value = '';
  success.value = '';
  imageInfo.value = '';
  if (fileInput.value) fileInput.value.value = '';
}

function removeLogo() {
  ++revision;
  processingImage.value = false;
  form.logo_data = '';
  error.value = '';
  success.value = '';
  imageInfo.value = '';
  if (fileInput.value) fileInput.value.value = '';
}

async function readLogo(event) {
  const file = event.target.files?.[0];
  if (!file) return;
  const currentRevision = ++revision;
  processingImage.value = true;
  error.value = '';
  success.value = '';
  imageInfo.value = '';
  let objectUrl;
  try {
    if (!['image/png', 'image/jpeg', 'image/webp'].includes(file.type)) {
      throw new Error('Seleziona un logo PNG, JPEG o WebP.');
    }
    if (file.size > MAX_SOURCE_BYTES) throw new Error('Il file originale deve pesare al massimo 4 MB.');
    objectUrl = URL.createObjectURL(file);
    const img = new Image();
    await new Promise((resolve, reject) => {
      img.onload = resolve;
      img.onerror = () => reject(new Error('Il file non contiene un’immagine valida.'));
      img.src = objectUrl;
    });
    if (currentRevision !== revision) return;
    if (!img.naturalWidth || !img.naturalHeight || Math.max(img.naturalWidth, img.naturalHeight) > MAX_DIMENSION) {
      throw new Error('Il logo deve avere dimensioni comprese tra 1 e 2048 pixel per lato.');
    }
    const canvas = document.createElement('canvas');
    const context = canvas.getContext('2d');
    if (!context) throw new Error('Impossibile elaborare l’immagine in questo browser.');
    let width = img.naturalWidth;
    let height = img.naturalHeight;
    let encoded = '';
    // Canvas removes ancillary metadata and keeps PNG/WebP transparency.
    const outputType = file.type === 'image/jpeg' ? 'image/jpeg' : 'image/png';
    for (let attempt = 0; attempt < 20; attempt += 1) {
      canvas.width = width;
      canvas.height = height;
      context.clearRect(0, 0, width, height);
      context.drawImage(img, 0, 0, width, height);
      encoded = canvas.toDataURL(outputType, 0.9);
      if (encoded.length <= MAX_ENCODED_LENGTH) break;
      width = Math.max(1, Math.round(width * 0.8));
      height = Math.max(1, Math.round(height * 0.8));
    }
    if (encoded.length > MAX_ENCODED_LENGTH || !/^data:image\/(png|jpeg);base64,/.test(encoded)) {
      throw new Error('Il logo supera il limite di 512 KB. Usa un’immagine più semplice.');
    }
    if (currentRevision !== revision) return;
    form.logo_data = encoded;
    imageInfo.value = `${width} × ${height} px · ${Math.ceil(encoded.length / 1024)} KB · pronto per il report`;
  } catch (cause) {
    if (currentRevision === revision) error.value = cause.message || 'Impossibile elaborare il logo.';
  } finally {
    if (objectUrl) URL.revokeObjectURL(objectUrl);
    if (currentRevision === revision) {
      processingImage.value = false;
      event.target.value = '';
    }
  }
}

async function save() {
  if (!loaded.value || busy.value) return;
  const companyName = form.company_name.trim();
  if (!companyName || companyName.length > 120) {
    error.value = 'Inserisci il nome dell’emittente, fino a 120 caratteri.';
    return;
  }
  const currentRevision = ++revision;
  saving.value = true;
  error.value = '';
  success.value = '';
  try {
    const payload = { company_name: companyName, logo_data: form.logo_data };
    const { data } = await apiClient.put(ENDPOINT, payload);
    if (currentRevision !== revision) return;
    applyBranding(data);
    success.value = 'Identità salvata. I prossimi report utilizzeranno questo nome e questo logo.';
  } catch (cause) {
    if (currentRevision === revision) {
      error.value = cause?.response?.status === 403
        ? 'Solo il super amministratore può modificare l’identità della piattaforma.'
        : 'Salvataggio non riuscito. Le modifiche restano disponibili: riprova.';
    }
  } finally {
    if (currentRevision === revision) saving.value = false;
  }
}

onMounted(load);
onBeforeUnmount(() => {
  ++revision;
  loadController?.abort();
});
</script>

<template>
  <section class="branding-section" aria-labelledby="platform-branding-heading" :aria-busy="busy">
    <header class="branding-heading">
      <div>
        <p class="eyebrow">Identità globale</p>
        <h2 id="platform-branding-heading">Emittente dei report</h2>
        <p class="hint">Il nome e il logo della piattaforma firmano i PDF scaricati da tutte le società, sia nel formato one pager sia nel report completo.</p>
      </div>
      <span class="scope-badge">Tutte le società</span>
    </header>

    <p v-if="loading" class="hint" role="status">Caricamento dell’identità della piattaforma…</p>
    <form v-if="loaded" @submit.prevent="save">
      <div class="branding-grid">
        <div class="branding-fields">
          <label for="report-company-name">Nome societario dell’emittente</label>
          <input id="report-company-name" v-model="form.company_name" name="company_name" type="text" maxlength="120" required :disabled="busy" placeholder="ArenaBoostX" @input="success = ''" />
          <p class="field-hint">Comparirà nell’intestazione e nella firma del report.</p>

          <label for="report-company-logo">Logo dell’emittente</label>
          <input id="report-company-logo" ref="fileInput" class="logo-file" type="file" accept="image/png,image/jpeg,image/webp" :disabled="loading || saving" aria-describedby="report-logo-help" @change="readLogo" />
          <p id="report-logo-help" class="field-hint">PNG, JPEG o WebP; massimo 4 MB e 2048 × 2048 px. Il file viene ottimizzato fino a 512 KB, mantenendo le proporzioni e la trasparenza.</p>
          <p v-if="processingImage" class="field-hint" role="status">Preparazione del logo…</p>
          <p v-else-if="imageInfo" class="field-hint">{{ imageInfo }}</p>
          <button v-if="form.logo_data || processingImage" class="button secondary remove-button" type="button" :disabled="loading || saving" @click="removeLogo">Rimuovi logo</button>
        </div>

        <aside class="signature-preview" aria-label="Anteprima firma del report">
          <p class="preview-label">Anteprima firma</p>
          <div class="signature-paper">
            <div class="preview-logo">
              <img v-if="form.logo_data" :src="form.logo_data" :alt="`Logo ${form.company_name || 'emittente'}`" />
              <span v-else class="logo-placeholder">Logo emittente</span>
            </div>
            <strong>{{ form.company_name.trim() || 'ArenaBoostX' }}</strong>
            <span class="preview-description">Fan engagement · Report evento</span>
          </div>
          <p class="field-hint">L’identità della società sportiva resta presente nel report insieme alla firma dell’emittente.</p>
        </aside>
      </div>
      <div class="branding-actions">
        <button class="button primary" type="submit" :disabled="busy || !dirty">{{ saving ? 'Salvataggio…' : 'Salva identità' }}</button>
        <button class="button secondary" type="button" :disabled="loading || saving || (!dirty && !processingImage)" @click="reset">Annulla modifiche</button>
        <span v-if="dirty && !saving" class="unsaved">Modifiche da salvare</span>
      </div>
    </form>
    <p v-if="error" class="feedback error" role="alert">{{ error }}</p>
    <p v-if="success" class="feedback success" role="status">{{ success }}</p>
    <button v-if="!loaded && !loading" class="button secondary" type="button" @click="load">Riprova caricamento</button>
  </section>
</template>

<style scoped>
.branding-section{padding:24px;border:1px solid rgba(148,163,184,.2);border-radius:18px;background:rgba(15,23,42,.8);color:#e2e8f0}
.branding-heading{display:flex;align-items:flex-start;justify-content:space-between;gap:24px;margin-bottom:24px}.branding-heading h2{margin:5px 0 8px;font-size:24px;color:#f8fafc}.eyebrow{margin:0;color:#fbbf24;font-size:11px;font-weight:800;letter-spacing:.13em;text-transform:uppercase}.hint{margin:0;max-width:740px;color:#aab8ca;font-size:14px;line-height:1.6}.scope-badge{flex-shrink:0;padding:7px 10px;border:1px solid #334155;border-radius:999px;color:#cbd5e1;font-size:12px}.branding-grid{display:grid;grid-template-columns:minmax(0,1.2fr) minmax(250px,.8fr);gap:30px}.branding-fields label{display:block;margin-bottom:8px;color:#e2e8f0;font-size:14px;font-weight:650}.branding-fields input[type=text]{box-sizing:border-box;width:100%;border:1px solid #475569;border-radius:10px;padding:12px;background:#0b1424;color:#f8fafc;font:inherit}.branding-fields input:focus-visible,.button:focus-visible{outline:2px solid #fbbf24;outline-offset:3px}.field-hint{margin:8px 0 22px;color:#aab8ca;font-size:12px;line-height:1.6}.logo-file{box-sizing:border-box;width:100%;font:inherit;font-size:13px;color:#cbd5e1}.logo-file::file-selector-button{margin-right:12px;padding:10px 12px;border:1px solid #475569;border-radius:8px;background:#1e293b;color:#e2e8f0;cursor:pointer}.signature-preview{padding:18px;border:1px solid rgba(148,163,184,.2);border-radius:12px;background:#0b1424}.preview-label{margin:0 0 12px;color:#94a3b8;font-size:11px;font-weight:700;text-transform:uppercase;letter-spacing:.1em}.signature-paper{display:flex;flex-direction:column;align-items:center;justify-content:center;gap:8px;min-height:200px;padding:20px;border-radius:8px;background:#fff;color:#172554;text-align:center;overflow-wrap:anywhere}.signature-paper strong{font-size:19px}.preview-logo{display:flex;align-items:center;justify-content:center;min-height:70px;width:100%;margin-bottom:5px}.preview-logo img{max-width:200px;max-height:80px;object-fit:contain}.logo-placeholder{padding:20px;border:1px dashed #cbd5e1;border-radius:6px;color:#64748b;font-size:12px}.preview-description{font-size:11px;color:#64748b}.signature-preview>.field-hint{margin-bottom:0}.branding-actions{display:flex;align-items:center;flex-wrap:wrap;gap:12px;margin-top:24px;padding-top:22px;border-top:1px solid rgba(148,163,184,.2)}.button{border-radius:9px;padding:11px 15px;font:inherit;font-size:13px;font-weight:700;cursor:pointer}.primary{border:1px solid #fbbf24;background:#fbbf24;color:#0f172a}.secondary{border:1px solid #475569;background:transparent;color:#e2e8f0}.button:disabled{opacity:.5;cursor:default}.remove-button{margin-top:-8px}.unsaved{color:#fbbf24;font-size:12px}.feedback{margin:18px 0 0;padding:12px;border-radius:9px;font-size:13px;line-height:1.5}.error{color:#fecaca;background:rgba(127,29,29,.25);border:1px solid rgba(248,113,113,.3)}.success{color:#bbf7d0;background:rgba(20,83,45,.25);border:1px solid rgba(74,222,128,.3)}
@media(max-width:800px){.branding-section{padding:18px}.branding-grid{grid-template-columns:1fr;gap:20px}.branding-heading{flex-direction:column;gap:12px}.scope-badge{align-self:flex-start}}
</style>
