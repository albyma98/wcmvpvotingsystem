<script setup>
import { computed, nextTick, onScopeDispose, reactive, ref } from 'vue';
import { DEFAULT_ORGANIZATION_COLORS, isHexColor } from '../../organizationTheme.js';
import { masterApi } from '../api';
import { useMasterWorkspace } from '../workspace';
import OrganizationArenaSelect from './OrganizationArenaSelect.vue';
import { COLOR_FIELDS, ORGANIZATIONS_ENDPOINT, ORGANIZATION_SPORTS, organizationDraft, organizationPayload } from './organizationFields';

const props = defineProps({
  organization: { type: Object, default: null },
  cancelHref: { type: String, required: true },
});
const emit = defineEmits(['saved']);
const { follow, useDirty } = useMasterWorkspace();
const draft = reactive(organizationDraft(props.organization));
const baseline = ref({ ...draft });
const form = ref(null);
const saving = ref(false);
const readingLogo = ref(false);
const errors = ref({});
const saveError = ref('');
const logoError = ref('');
let reader = null;
let disposed = false;

useDirty(computed(() => saving.value || readingLogo.value || Object.keys(baseline.value).some((key) => draft[key] !== baseline.value[key])));

onScopeDispose(() => {
  disposed = true;
  reader?.abort();
});

function selectLogo(event) {
  const input = event.target;
  const [file] = input.files;
  if (!file) return;
  input.value = '';
  logoError.value = '';
  if (file.size > 2 * 1024 * 1024) {
    logoError.value = 'Il file del logo supera il limite di 2 MiB.';
    return;
  }
  if (file.type && !file.type.startsWith('image/')) {
    logoError.value = 'Seleziona un file immagine valido.';
    return;
  }
  reader = new FileReader();
  readingLogo.value = true;
  reader.onload = () => {
    draft.logo_url = String(reader.result ?? '');
    readingLogo.value = false;
    reader = null;
  };
  reader.onerror = () => {
    logoError.value = 'Impossibile leggere il file selezionato.';
    readingLogo.value = false;
    reader = null;
  };
  reader.readAsDataURL(file);
}

function removeLogo() {
  draft.logo_url = '';
  logoError.value = '';
}

function selectColor(key, event) {
  draft[key] = event.target.value;
  delete errors.value[key];
}

function validate() {
  const result = {};
  if (!draft.name) result.name = 'Inserisci il nome della società.';
  if (!draft.slug) result.slug = 'Inserisci uno slug o l’URL pubblico della società.';
  if (!ORGANIZATION_SPORTS.includes(draft.sport)) result.sport = 'Seleziona lo sport praticato dalla società.';
  for (const { key } of COLOR_FIELDS) {
    if (!isHexColor(draft[key])) result[key] = 'Inserisci un colore nel formato #RRGGBB.';
  }
  if (!form.value.elements.namedItem('sms_cost').validity.valid) result.sms_cost = 'Inserisci un importo pari o superiore a zero, con al massimo due decimali.';
  if (!form.value.elements.namedItem('free_sms').validity.valid) result.free_sms = 'Inserisci un numero intero pari o superiore a zero.';
  return result;
}

async function save() {
  if (saving.value || readingLogo.value) return;
  saveError.value = '';
  errors.value = validate();
  if (Object.keys(errors.value).length) {
    await nextTick();
    form.value?.querySelector('[aria-invalid="true"]')?.focus();
    return;
  }
  saving.value = true;
  const payload = organizationPayload(draft, props.organization);
  try {
    const { data } = props.organization
      ? await masterApi.put(`${ORGANIZATIONS_ENDPOINT}/${props.organization.id}`, payload)
      : await masterApi.post(ORGANIZATIONS_ENDPOINT, payload);
    if (!disposed) {
      baseline.value = { ...draft };
      emit('saved', data);
    }
  } catch (error) {
    const status = error?.response?.status;
    saveError.value = error?.response?.data?.error || (status === 404
      ? 'Società non trovata.'
      : status === 400 ? 'Dati non validi. Controlla i campi e riprova.' : 'Impossibile salvare la società. Riprova.');
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <form ref="form" class="m-stack organization-editor" novalidate @submit.prevent="save">
    <fieldset class="editor-fields m-stack" :disabled="saving">
      <section class="m-panel m-stack">
        <header class="m-panel-head">
          <div><h2>Anagrafica</h2><p>Identità della società e indirizzo della pagina tifosi.</p></div>
        </header>
        <div class="m-fields">
          <label class="m-field">
            <span>Nome società</span>
            <input v-model.trim="draft.name" class="m-input" name="name" placeholder="Es. Volley Milano" required :aria-invalid="Boolean(errors.name)" :aria-describedby="errors.name ? 'm-org-name-error' : undefined" @input="delete errors.name" />
            <small v-if="errors.name" id="m-org-name-error" class="m-error">{{ errors.name }}</small>
          </label>
          <label class="m-field">
            <span>Slug o URL pubblico</span>
            <input v-model.trim="draft.slug" class="m-input" name="slug" placeholder="volley-milano o https://mia-societa.it" required :aria-invalid="Boolean(errors.slug)" :aria-describedby="errors.slug ? 'm-org-slug-error' : 'm-org-slug-hint'" @input="delete errors.slug" />
            <small v-if="errors.slug" id="m-org-slug-error" class="m-error">{{ errors.slug }}</small>
            <small v-else id="m-org-slug-hint" class="m-muted-text">Puoi inserire uno slug, che verrà normalizzato, oppure un URL completo.</small>
          </label>
          <label class="m-field">
            <span>Città o descrizione</span>
            <input v-model.trim="draft.city" class="m-input" name="city" placeholder="Es. Milano" />
          </label>
          <label class="m-field">
            <span>Sport praticato</span>
            <select v-model="draft.sport" class="m-input" name="sport" required :aria-invalid="Boolean(errors.sport)" :aria-describedby="errors.sport ? 'm-org-sport-error' : undefined" @change="delete errors.sport">
              <option value="" disabled>Seleziona uno sport</option>
              <option v-for="sport in ORGANIZATION_SPORTS" :key="sport" :value="sport">{{ sport }}</option>
            </select>
            <small v-if="errors.sport" id="m-org-sport-error" class="m-error">{{ errors.sport }}</small>
          </label>
        </div>
        <label class="m-choice"><input v-model="draft.is_active" type="checkbox" /> Società attiva</label>
      </section>

      <section class="m-panel m-stack">
        <header class="m-panel-head">
          <div><h2>Identità visiva</h2><p>Logo e colori della home tifosi.</p></div>
        </header>
        <div class="m-fields">
          <label class="m-field">
            <span>URL del logo</span>
            <input v-model.trim="draft.logo_url" class="m-input" :disabled="readingLogo" placeholder="https://… o data:image/…" />
          </label>
          <label class="m-field">
            <span>Carica logo</span>
            <input class="m-input" type="file" accept="image/png,image/jpeg,image/webp,image/svg+xml" :disabled="readingLogo" :aria-invalid="Boolean(logoError)" :aria-describedby="logoError ? 'm-org-logo-error' : 'm-org-logo-hint'" @change="selectLogo" />
            <small v-if="logoError" id="m-org-logo-error" class="m-error" role="alert">{{ logoError }}</small>
            <small v-else id="m-org-logo-hint" class="m-muted-text">PNG, JPG, WebP o SVG. Massimo 2 MiB.</small>
          </label>
        </div>
        <p v-if="readingLogo" role="status" class="m-muted-text">Lettura immagine…</p>
        <div v-if="draft.logo_url" class="logo-preview">
          <img :src="draft.logo_url" alt="Anteprima del logo" />
          <button type="button" class="m-btn m-subtle" :disabled="readingLogo" @click="removeLogo">Rimuovi logo</button>
        </div>
        <div class="m-fields organization-colors">
          <div v-for="{ key, label, hint } in COLOR_FIELDS" :key="key" class="m-field">
            <label :for="`m-org-${key}`">{{ label }}</label>
            <div class="color-control">
              <input class="m-input" type="color" :aria-label="`Seleziona ${label.toLowerCase()}`" :value="isHexColor(draft[key]) ? draft[key] : DEFAULT_ORGANIZATION_COLORS[key]" @input="selectColor(key, $event)" />
              <input :id="`m-org-${key}`" v-model.trim="draft[key]" class="m-input" required maxlength="7" placeholder="#000000" :aria-invalid="Boolean(errors[key])" :aria-describedby="`m-org-${key}-hint`" @input="delete errors[key]" />
            </div>
            <small :id="`m-org-${key}-hint`" :class="errors[key] ? 'm-error' : 'm-muted-text'">{{ errors[key] || hint }}</small>
          </div>
        </div>
      </section>

      <section class="m-panel m-stack">
        <header class="m-panel-head">
          <div><h2>Servizi e arena</h2><p>Configura i servizi disponibili per la società.</p></div>
        </header>
        <div class="m-fields">
          <label class="m-field">
            <span>Costo SMS (€)</span>
            <input v-model.number="draft.sms_cost" class="m-input" name="sms_cost" type="number" min="0" step="0.01" placeholder="0,08" :aria-invalid="Boolean(errors.sms_cost)" :aria-describedby="errors.sms_cost ? 'm-org-sms-error' : undefined" @input="delete errors.sms_cost" />
            <small v-if="errors.sms_cost" id="m-org-sms-error" class="m-error">{{ errors.sms_cost }}</small>
          </label>
          <label class="m-field">
            <span>SMS gratuiti</span>
            <input v-model.number="draft.free_sms" class="m-input" name="free_sms" type="number" min="0" step="1" placeholder="0" :aria-invalid="Boolean(errors.free_sms)" :aria-describedby="errors.free_sms ? 'm-org-free-sms-error' : undefined" @input="delete errors.free_sms" />
            <small v-if="errors.free_sms" id="m-org-free-sms-error" class="m-error">{{ errors.free_sms }}</small>
          </label>
          <OrganizationArenaSelect v-model="draft.arena_id" :assigned-arena="organization?.arena" :disabled="saving" />
        </div>
        <label class="m-choice"><input v-model="draft.bar_enabled" type="checkbox" /> Funzionalità BAR attiva</label>
      </section>
    </fieldset>

    <div class="m-panel editor-footer">
      <p v-if="saveError" class="m-error" role="alert">{{ saveError }}</p>
      <div class="m-actions">
        <a class="m-btn" :href="cancelHref" @click="follow">Annulla</a>
        <button class="m-btn m-primary" type="submit" :disabled="saving || readingLogo">{{ saving ? 'Salvataggio…' : 'Salva società' }}</button>
      </div>
    </div>
  </form>
</template>

<style scoped>
.organization-editor { max-width: 68rem; }
.editor-fields { min-width: 0; border: 0; padding: 0; margin: 0; }
.m-choice, .logo-preview, .color-control { display: flex; align-items: center; gap: .65rem; }
.m-choice { align-self: start; width: fit-content; }
.logo-preview { flex-wrap: wrap; }
.logo-preview img { display: block; width: 3.5rem; height: 3.5rem; object-fit: contain; border: 1px solid var(--m-line); border-radius: .5rem; }
.organization-colors { grid-template-columns: repeat(auto-fit, minmax(min(100%, 12rem), 1fr)); }
.color-control input[type="color"] { width: 2.75rem; padding: .25rem; flex: none; }
.color-control input:not([type="color"]) { min-width: 0; }
.editor-footer .m-actions { justify-content: flex-end; }
</style>
