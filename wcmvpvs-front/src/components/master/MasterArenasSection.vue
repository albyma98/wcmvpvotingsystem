<script setup>
import { computed, onMounted, ref } from 'vue';
import { apiClient } from '../../api';
import { readArenaScene } from '../../arena/scene/sceneConfig.js';
import BaseBadge from '../admin/ui/BaseBadge.vue';
import BaseButton from '../admin/ui/BaseButton.vue';
import BaseCard from '../admin/ui/BaseCard.vue';
import BaseInput from '../admin/ui/BaseInput.vue';
import BaseTable from '../admin/ui/BaseTable.vue';
import BaseTextarea from '../admin/ui/BaseTextarea.vue';
import SectionHeader from '../admin/ui/SectionHeader.vue';

const endpoint = '/admin/master/arenas';
const arenas = ref([]);
const form = ref(null);
const loading = ref(true);
const openingId = ref(null);
const saving = ref(false);
const loadError = ref('');
const formError = ref('');
const notice = ref('');
const busy = computed(() => openingId.value !== null || saving.value);
const capacityFormat = new Intl.NumberFormat('it-IT');

async function load() {
  loading.value = true;
  loadError.value = '';
  try {
    const { data } = await apiClient.get(endpoint);
    arenas.value = data.arenas ?? [];
  } catch {
    loadError.value = 'Impossibile caricare il catalogo delle arene.';
  } finally {
    loading.value = false;
  }
}

function openCreate() {
  formError.value = '';
  notice.value = '';
  form.value = { id: null, name: '', slug: '', city: '', team_name: '', capacity: '', is_active: true, scene: '{}' };
}

async function openEdit(id) {
  openingId.value = id;
  loadError.value = '';
  formError.value = '';
  notice.value = '';
  try {
    const { data } = await apiClient.get(`${endpoint}/${id}`);
    form.value = {
      id: data.id,
      name: data.name,
      slug: data.slug,
      city: data.city,
      team_name: data.team_name || '',
      capacity: data.capacity ?? '',
      is_active: data.is_active,
      scene: JSON.stringify(data.scene_config, null, 2),
    };
  } catch {
    loadError.value = 'Impossibile aprire l’arena. Aggiorna l’elenco e riprova.';
  } finally {
    openingId.value = null;
  }
}

function closeEditor() {
  form.value = null;
  formError.value = '';
}

async function save() {
  if (!form.value || busy.value) return;
  const { id, name, slug, city, team_name, capacity, is_active, scene } = form.value;
  formError.value = '';

  if (!name.trim() || !city.trim() || !/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(slug)) {
    formError.value = 'Inserisci nome, città e uno slug con lettere minuscole, numeri e trattini.';
    return;
  }
  if (capacity !== '' && (!Number.isSafeInteger(capacity) || capacity < 1)) {
    formError.value = 'La capienza deve essere un numero intero positivo.';
    return;
  }

  let sceneConfig;
  try {
    sceneConfig = JSON.parse(scene);
  } catch {
    formError.value = 'La configurazione della scena contiene JSON non valido.';
    return;
  }
  if (!sceneConfig || typeof sceneConfig !== 'object' || Array.isArray(sceneConfig)) {
    formError.value = 'La configurazione della scena deve essere un oggetto JSON.';
    return;
  }
  if (new TextEncoder().encode(JSON.stringify(sceneConfig)).length > 256 * 1024) {
    formError.value = 'La configurazione della scena supera il limite di 256 KiB.';
    return;
  }
  if (sceneConfig.schema_version === 1 && sceneConfig.court?.sport === 'basket' && !readArenaScene(sceneConfig)) {
    formError.value = 'Configurazione basket non valida. Verifica le dimensioni del campo e dell’arena, i lati, le righe e le sedute dei settori.';
    return;
  }

  saving.value = true;
  const payload = { name, slug, city, team_name, capacity: capacity === '' ? null : capacity, is_active, scene_config: sceneConfig };
  try {
    const { data } = id === null
      ? await apiClient.post(endpoint, payload)
      : await apiClient.put(`${endpoint}/${id}`, payload);
    arenas.value = [...arenas.value.filter(arena => arena.id !== data.id), data].sort((a, b) => a.name.localeCompare(b.name, 'it'));
    notice.value = id === null ? 'Arena creata.' : 'Arena aggiornata.';
    closeEditor();
  } catch (error) {
    formError.value = error?.response?.data?.error || 'Impossibile salvare l’arena. Riprova.';
  } finally {
    saving.value = false;
  }
}

onMounted(load);
</script>

<template>
  <section class="arena-catalog" aria-label="Catalogo arene">
    <SectionHeader title="Arene" description="Gestisci le arene che puoi assegnare alle società e la configurazione di Arena Tifosi.">
      <template #actions>
        <BaseButton variant="outline" :disabled="loading || busy" @click="load">Aggiorna elenco</BaseButton>
        <BaseButton :disabled="loading || busy" @click="openCreate">Nuova arena</BaseButton>
      </template>
    </SectionHeader>

    <p v-if="loadError" class="error" role="alert">{{ loadError }}</p>
    <p v-if="notice" class="notice" role="status">{{ notice }}</p>

    <BaseCard v-if="form" class="arena-editor">
      <h3>{{ form.id === null ? 'Crea arena' : 'Modifica arena' }}</h3>
      <form class="arena-form" @submit.prevent="save">
        <BaseInput>
          <label for="master-arena-name">Nome</label>
          <input id="master-arena-name" v-model.trim="form.name" required :disabled="busy" />
        </BaseInput>
        <BaseInput>
          <label for="master-arena-slug">Slug</label>
          <input id="master-arena-slug" v-model.trim="form.slug" required :disabled="busy" pattern="[a-z0-9]+(-[a-z0-9]+)*" placeholder="es. baltur-arena" />
        </BaseInput>
        <BaseInput>
          <label for="master-arena-city">Città</label>
          <input id="master-arena-city" v-model.trim="form.city" required :disabled="busy" />
        </BaseInput>
        <BaseInput>
          <label for="master-arena-team">Squadra <small>(facoltativa)</small></label>
          <input id="master-arena-team" v-model.trim="form.team_name" :disabled="busy" />
        </BaseInput>
        <BaseInput>
          <label for="master-arena-capacity">Capienza <small>(facoltativa)</small></label>
          <input id="master-arena-capacity" v-model.number="form.capacity" type="number" min="1" step="1" :disabled="busy" />
        </BaseInput>
        <label class="active-field">
          <input v-model="form.is_active" type="checkbox" :disabled="busy" />
          Arena attiva
        </label>
        <BaseTextarea class="arena-scene">
          <label for="master-arena-scene">Configurazione della scena</label>
          <textarea id="master-arena-scene" v-model="form.scene" rows="14" spellcheck="false" aria-describedby="master-arena-scene-hint" required :disabled="busy" />
          <small id="master-arena-scene-hint">Inserisci un oggetto JSON con campo, settori, materiali e impostazioni della scena.</small>
        </BaseTextarea>
        <p v-if="formError" class="error form-feedback" role="alert">{{ formError }}</p>
        <div class="form-actions">
          <BaseButton variant="outline" :disabled="busy" @click="closeEditor">Annulla</BaseButton>
          <BaseButton type="submit" :disabled="busy">{{ saving ? 'Salvataggio…' : 'Salva arena' }}</BaseButton>
        </div>
      </form>
    </BaseCard>

    <p v-if="loading" role="status">Caricamento arene…</p>
    <BaseTable v-else-if="arenas.length" aria-label="Elenco arene">
      <table>
        <thead>
          <tr><th scope="col">Nome</th><th scope="col">Slug</th><th scope="col">Città</th><th scope="col">Capienza</th><th scope="col">Stato</th><th scope="col">Azioni</th></tr>
        </thead>
        <tbody>
          <tr v-for="arena in arenas" :key="arena.id">
            <td><strong>{{ arena.name }}</strong></td>
            <td><code>{{ arena.slug }}</code></td>
            <td>{{ arena.city }}</td>
            <td>{{ arena.capacity == null ? '—' : capacityFormat.format(arena.capacity) }}</td>
            <td><BaseBadge :variant="arena.is_active ? 'success' : 'default'">{{ arena.is_active ? 'Attiva' : 'Inattiva' }}</BaseBadge></td>
            <td><BaseButton variant="outline" size="sm" :aria-label="`Modifica ${arena.name}`" :disabled="busy" @click="openEdit(arena.id)">{{ openingId === arena.id ? 'Apertura…' : 'Modifica' }}</BaseButton></td>
          </tr>
        </tbody>
      </table>
    </BaseTable>
    <p v-else-if="!loadError">Nessuna arena nel catalogo.</p>
  </section>
</template>

<style scoped>
.arena-catalog { min-width: 0; }
.arena-editor { margin-bottom: 1.25rem; }
.arena-editor h3 { margin: 0 0 1rem; }
.arena-form { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(100%, 200px), 1fr)); gap: 1rem; }
.arena-form label { font-weight: 600; }
.arena-form small { font-weight: 400; color: #475569; }
.active-field { display: flex; align-items: center; align-self: end; gap: 0.5rem; min-height: 42px; }
.arena-scene, .form-feedback, .form-actions { grid-column: 1 / -1; }
.arena-scene :deep(textarea) { font-family: ui-monospace, Consolas, monospace; }
.form-actions { display: flex; justify-content: flex-end; flex-wrap: wrap; gap: 0.5rem; }
.error { color: #b91c1c; }
.notice { color: #166534; }
</style>
