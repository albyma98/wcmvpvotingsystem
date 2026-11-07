<script setup>
import { computed, nextTick, onScopeDispose, reactive, ref } from 'vue';
import { readArenaScene } from '../../arena/scene/sceneConfig.js';
import { masterApi } from '../api';
import { useMasterWorkspace } from '../workspace';
import { ARENAS_ENDPOINT } from './arenaCatalog';

const props = defineProps({
  arena: { type: Object, default: null },
  cancelHref: { type: String, required: true },
});
const emit = defineEmits(['saved']);
const { follow, useDirty } = useMasterWorkspace();
const draft = reactive({
  name: props.arena?.name ?? '',
  slug: props.arena?.slug ?? '',
  city: props.arena?.city ?? '',
  team_name: props.arena?.team_name ?? '',
  capacity: props.arena?.capacity ?? '',
  is_active: props.arena?.is_active ?? true,
  scene: JSON.stringify(props.arena?.scene_config ?? {}, null, 2),
});
const baseline = ref({ ...draft });
const form = ref(null);
const saving = ref(false);
const errors = ref({});
const saveError = ref('');
let disposed = false;

useDirty(computed(() => saving.value || Object.keys(baseline.value).some((key) => draft[key] !== baseline.value[key])));
onScopeDispose(() => { disposed = true; });

function validateScene(scene) {
  let config;
  try {
    config = JSON.parse(scene);
  } catch {
    return { error: 'La configurazione della scena contiene JSON non valido.' };
  }
  if (!config || typeof config !== 'object' || Array.isArray(config)) {
    return { error: 'La configurazione della scena deve essere un oggetto JSON.' };
  }
  if (new TextEncoder().encode(JSON.stringify(config)).length > 256 * 1024) {
    return { error: 'La configurazione della scena supera il limite di 256 KiB.' };
  }
  if (config.schema_version === 1 && config.court?.sport === 'basket' && !readArenaScene(config)) {
    return { error: 'Configurazione basket non valida. Verifica le dimensioni del campo e dell’arena, i lati, le righe e le sedute dei settori.' };
  }
  return { config };
}

function validate() {
  const result = {};
  if (!draft.name) result.name = 'Inserisci il nome dell’arena.';
  if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(draft.slug)) result.slug = 'Usa lettere minuscole, numeri e trattini.';
  if (!draft.city) result.city = 'Inserisci la città dell’arena.';
  if (form.value.elements.namedItem('capacity').validity.badInput || (draft.capacity !== '' && (!Number.isSafeInteger(draft.capacity) || draft.capacity < 1))) {
    result.capacity = 'Inserisci un numero intero positivo.';
  }
  const scene = validateScene(draft.scene);
  if (scene.error) result.scene = scene.error;
  return { result, sceneConfig: scene.config };
}

async function focusInvalid() {
  await nextTick();
  form.value?.querySelector('[aria-invalid="true"]')?.focus();
}

async function save() {
  if (saving.value) return;
  saveError.value = '';
  const { result, sceneConfig } = validate();
  errors.value = result;
  if (Object.keys(result).length) {
    await focusInvalid();
    return;
  }
  const { name, slug, city, team_name, capacity, is_active } = draft;
  const payload = { name, slug, city, team_name, capacity: capacity === '' ? null : capacity, is_active, scene_config: sceneConfig };
  saving.value = true;
  try {
    const { data } = props.arena
      ? await masterApi.put(`${ARENAS_ENDPOINT}/${props.arena.id}`, payload)
      : await masterApi.post(ARENAS_ENDPOINT, payload);
    if (!disposed) {
      baseline.value = { ...draft };
      emit('saved', data);
    }
  } catch (error) {
    if (error?.response?.status === 409) {
      saving.value = false;
      errors.value.slug = error.response.data?.error || 'Lo slug è già utilizzato da un’altra arena.';
      await focusInvalid();
    } else {
      saveError.value = error?.response?.data?.error || 'Impossibile salvare l’arena. Riprova.';
    }
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <form ref="form" class="m-stack arena-editor" novalidate @submit.prevent="save">
    <fieldset class="editor-fields m-stack" :disabled="saving">
      <section class="m-panel m-stack">
        <header class="m-panel-head"><div><h2>Informazioni generali</h2><p>Identità, posizione e stato dell’arena nel catalogo.</p></div></header>
        <div class="m-fields">
          <label class="m-field">
            <span>Nome arena</span>
            <input v-model.trim="draft.name" class="m-input" name="name" placeholder="Es. Baltur Arena" required :aria-invalid="Boolean(errors.name)" :aria-describedby="errors.name ? 'm-arena-name-error' : undefined" @input="delete errors.name" />
            <small v-if="errors.name" id="m-arena-name-error" class="m-error">{{ errors.name }}</small>
          </label>
          <label class="m-field">
            <span>Slug</span>
            <input v-model.trim="draft.slug" class="m-input" name="slug" placeholder="es. baltur-arena" required :aria-invalid="Boolean(errors.slug)" :aria-describedby="errors.slug ? 'm-arena-slug-error' : undefined" @input="delete errors.slug" />
            <small v-if="errors.slug" id="m-arena-slug-error" class="m-error">{{ errors.slug }}</small>
          </label>
          <label class="m-field">
            <span>Città</span>
            <input v-model.trim="draft.city" class="m-input" name="city" placeholder="Es. Cento" required :aria-invalid="Boolean(errors.city)" :aria-describedby="errors.city ? 'm-arena-city-error' : undefined" @input="delete errors.city" />
            <small v-if="errors.city" id="m-arena-city-error" class="m-error">{{ errors.city }}</small>
          </label>
          <label class="m-field">
            <span>Squadra <small class="m-muted-text">(facoltativa)</small></span>
            <input v-model.trim="draft.team_name" class="m-input" name="team_name" placeholder="Es. Benedetto XIV Cento" />
          </label>
          <label class="m-field">
            <span>Capienza <small class="m-muted-text">(facoltativa)</small></span>
            <input v-model.number="draft.capacity" class="m-input" name="capacity" type="number" min="1" step="1" placeholder="Es. 2000" :aria-invalid="Boolean(errors.capacity)" :aria-describedby="errors.capacity ? 'm-arena-capacity-error' : undefined" @input="delete errors.capacity" />
            <small v-if="errors.capacity" id="m-arena-capacity-error" class="m-error">{{ errors.capacity }}</small>
          </label>
        </div>
        <label class="arena-status"><input v-model="draft.is_active" type="checkbox" /> Arena attiva</label>
      </section>

      <section class="m-panel m-stack">
        <header class="m-panel-head"><div><h2>Configurazione 3D</h2><p>Impostazioni della scena per Arena Tifosi.</p></div></header>
        <label class="m-field">
          <span>Configurazione della scena (JSON)</span>
          <textarea v-model="draft.scene" class="m-input scene-editor" name="scene" rows="12" placeholder="{}" spellcheck="false" required :aria-invalid="Boolean(errors.scene)" :aria-describedby="errors.scene ? 'm-arena-scene-error' : 'm-arena-scene-hint'" @input="delete errors.scene" />
          <small v-if="errors.scene" id="m-arena-scene-error" class="m-error">{{ errors.scene }}</small>
          <small v-else id="m-arena-scene-hint" class="m-muted-text">Oggetto JSON con campo, settori, materiali e impostazioni. Massimo 256 KiB.</small>
        </label>
      </section>
    </fieldset>

    <div class="m-panel editor-footer">
      <p v-if="saveError" class="m-error" role="alert">{{ saveError }}</p>
      <div class="m-actions">
        <a class="m-btn" :href="cancelHref" @click="follow">Annulla</a>
        <button class="m-btn m-primary" type="submit" :disabled="saving">{{ saving ? 'Salvataggio…' : 'Salva arena' }}</button>
      </div>
    </div>
  </form>
</template>

<style scoped>
.arena-editor { max-width: 68rem; }
.editor-fields { min-width: 0; border: 0; padding: 0; margin: 0; }
.arena-status { display: flex; width: fit-content; align-items: center; gap: .65rem; }
.scene-editor { font-family: ui-monospace, Consolas, monospace; font-size: .875rem; resize: vertical; tab-size: 2; }
.editor-footer .m-actions { justify-content: flex-end; }
</style>
