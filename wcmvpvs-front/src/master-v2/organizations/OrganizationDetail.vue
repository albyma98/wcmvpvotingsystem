<script setup>
import { computed, ref, watch } from 'vue';
import { normalizeOrganizationColors } from '../../organizationTheme.js';
import { masterApi } from '../api';
import { useMasterWorkspace } from '../workspace';
import MasterIcon from '../ui/MasterIcon.vue';
import OrganizationArenaSelect from './OrganizationArenaSelect.vue';
import { adminOrganizationUrl, COLOR_FIELDS, organizationDate, ORGANIZATIONS_ENDPOINT, ORGANIZATIONS_PATH, publicOrganizationUrl, smsPrice } from './organizationFields';

const props = defineProps({
  organization: { type: Object, required: true },
  stats: { type: Object, default: () => ({}) },
  tab: { type: String, default: '' },
});
const emit = defineEmits(['updated', 'remove']);
const { follow, useDirty } = useMasterWorkspace();
const href = computed(() => `${ORGANIZATIONS_PATH}/${props.organization.id}`);
const currentArena = computed(() => props.organization.arena_id ?? null);
const colors = computed(() => normalizeOrganizationColors(props.organization));
const arenaId = ref(currentArena.value);
const saving = ref(false);
const saved = ref(false);
const error = ref('');
const changed = computed(() => arenaId.value !== currentArena.value);
const number = new Intl.NumberFormat('it-IT');
const count = (value) => value == null ? '—' : number.format(value);

useDirty(computed(() => saving.value || (props.tab === 'arena' && changed.value)));

watch([() => props.tab, currentArena], () => {
  arenaId.value = currentArena.value;
  error.value = '';
});
watch(() => props.tab, () => { saved.value = false; });

function clearFeedback() {
  error.value = '';
  saved.value = false;
}

function resetArena() {
  arenaId.value = currentArena.value;
  clearFeedback();
}

async function saveArena() {
  if (!changed.value || saving.value) return;
  clearFeedback();
  saving.value = true;
  try {
    const { data } = await masterApi.patch(`${ORGANIZATIONS_ENDPOINT}/${props.organization.id}/arena`, { arena_id: arenaId.value });
    emit('updated', data);
    arenaId.value = data.arena_id ?? null;
    saved.value = true;
  } catch (failure) {
    error.value = failure?.response?.data?.error || 'Impossibile salvare l’assegnazione. Riprova.';
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <div class="m-stack organization-detail">
    <section class="m-panel organization-header">
      <div class="profile-summary">
        <div class="profile-logo">
          <img v-if="organization.logo_url" :src="organization.logo_url" alt="" />
          <MasterIcon v-else name="building" :size="25" />
        </div>
        <div class="profile-title">
          <p class="m-muted-text">Società #{{ organization.id }}</p>
          <h2>{{ organization.name }}</h2>
          <div class="profile-meta">
            <span :class="['m-badge', organization.is_active ? 'm-good' : 'm-muted']">{{ organization.is_active ? 'Attiva' : 'Disattiva' }}</span>
            <span v-if="organization.sport">{{ organization.sport }}</span>
            <span v-if="organization.city">{{ organization.city }}</span>
          </div>
        </div>
      </div>
      <div class="m-actions">
        <a v-if="organization.slug" :href="publicOrganizationUrl(organization.slug)" class="m-btn" target="_blank" rel="noopener noreferrer">Pagina pubblica</a>
        <a :href="adminOrganizationUrl(organization)" class="m-btn" target="_blank" rel="noopener noreferrer">Pannello società</a>
      </div>
    </section>

    <nav aria-label="Scheda della società" class="organization-tabs">
      <a :href="href" :aria-current="tab !== 'arena' ? 'page' : undefined" @click="follow">Panoramica</a>
      <a :href="`${href}/arena`" :aria-current="tab === 'arena' ? 'page' : undefined" @click="follow">Arena assegnata</a>
    </nav>

    <section v-if="tab === 'arena'" class="m-panel m-stack organization-arena">
      <header class="m-panel-head">
        <div><h2>Arena assegnata</h2><p>Associa un’arena del catalogo alla società.</p></div>
      </header>
      <form class="m-stack" @submit.prevent="saveArena">
        <OrganizationArenaSelect v-model="arenaId" :assigned-arena="organization.arena" :disabled="saving" label="Seleziona arena" @update:model-value="clearFeedback" />
        <p v-if="error" class="m-error" role="alert">{{ error }}</p>
        <p v-if="saved" class="m-success" role="status">Assegnazione salvata.</p>
        <div class="m-actions">
          <button class="m-btn" type="button" :disabled="!changed || saving" @click="resetArena">Annulla modifiche</button>
          <button class="m-btn m-primary" type="submit" :disabled="!changed || saving">{{ saving ? 'Salvataggio…' : 'Salva assegnazione' }}</button>
        </div>
      </form>
    </section>

    <template v-else>
      <div class="detail-columns">
        <section class="m-panel m-stack">
          <header class="m-panel-head"><div><h2>Anagrafica e servizi</h2><p>Configurazione attuale della società.</p></div></header>
          <dl class="detail-list">
            <div><dt>Slug o URL pubblico</dt><dd><a v-if="organization.slug" class="m-link" :href="publicOrganizationUrl(organization.slug)" target="_blank" rel="noopener noreferrer">{{ organization.slug }}</a><span v-else>—</span></dd></div>
            <div><dt>Sport praticato</dt><dd>{{ organization.sport || 'Da impostare' }}</dd></div>
            <div><dt>Città o descrizione</dt><dd>{{ organization.city || '—' }}</dd></div>
            <div><dt>Creato il</dt><dd>{{ organizationDate(organization.created_at) }}</dd></div>
            <div><dt>Funzionalità BAR</dt><dd><span :class="['m-badge', organization.bar_enabled ? 'm-good' : 'm-muted']">{{ organization.bar_enabled ? 'Attiva' : 'Disattiva' }}</span></dd></div>
            <div><dt>Costo SMS</dt><dd>{{ organization.sms_cost == null ? '—' : smsPrice.format(organization.sms_cost) }}</dd></div>
            <div><dt>SMS gratuiti</dt><dd>{{ count(organization.free_sms) }}</dd></div>
            <div><dt>Arena</dt><dd><a class="m-link" :href="`${href}/arena`" @click="follow">{{ organization.arena?.name || 'Nessuna arena' }}</a></dd></div>
          </dl>
        </section>

        <section class="m-panel m-stack">
          <header class="m-panel-head"><div><h2>Riepilogo voti</h2><p>Attività registrata per questa società.</p></div></header>
          <dl class="detail-list metrics-list">
            <div><dt>Totale voti</dt><dd>{{ count(stats.total_votes) }}</dd></div>
            <div><dt>Partite totali</dt><dd>{{ count(stats.total_matches) }}</dd></div>
            <div><dt>Voti ultima partita</dt><dd>{{ count(stats.last_match_votes) }}</dd></div>
            <div><dt>Data ultima partita</dt><dd>{{ organizationDate(stats.last_match_date) }}</dd></div>
          </dl>
        </section>
      </div>

      <section class="m-panel m-stack">
        <header class="m-panel-head"><div><h2>Colori della società</h2><p>Palette usata nella home tifosi.</p></div></header>
        <dl class="brand-palette">
          <div v-for="{ key, label } in COLOR_FIELDS" :key="key">
            <dt>{{ label }}</dt>
            <dd><span class="color-swatch" :style="{ backgroundColor: colors[key] }" aria-hidden="true"></span><code>{{ colors[key].toUpperCase() }}</code></dd>
          </div>
        </dl>
      </section>

      <section class="m-panel m-panel-head">
        <div><h2>Eliminazione della società</h2><p>L’eliminazione rimuove definitivamente la società e i dati collegati.</p></div>
        <button type="button" class="m-btn m-danger" @click="emit('remove', organization)">Elimina società</button>
      </section>
    </template>
  </div>
</template>

<style scoped>
.organization-header { display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1rem; }
.profile-summary { display: flex; min-width: 0; align-items: center; gap: 1rem; }
.profile-logo { display: grid; place-items: center; width: 3.5rem; height: 3.5rem; flex: none; overflow: hidden; border: 1px solid var(--m-line); border-radius: .75rem; color: var(--m-accent); background: var(--m-soft); }
.profile-logo img { width: 100%; height: 100%; object-fit: contain; }
.profile-title { min-width: 0; overflow-wrap: anywhere; }
.profile-title h2 { font-size: 1.2rem; margin-block: .15rem .35rem; }
.profile-meta { display: flex; flex-wrap: wrap; align-items: center; gap: .3rem .65rem; color: var(--m-muted); font-size: .875rem; }
.organization-tabs { display: flex; flex-wrap: wrap; gap: .25rem 1rem; border-bottom: 1px solid var(--m-line); }
.organization-tabs a { display: block; color: var(--m-muted); padding: .65rem .1rem; font-weight: 650; text-decoration: none; border-bottom: 2px solid transparent; }
.organization-tabs a:hover, .organization-tabs a[aria-current] { color: var(--m-accent); }
.organization-tabs a[aria-current] { border-bottom-color: var(--m-accent); }
.detail-columns { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(100%, 23rem), 1fr)); gap: 1rem; align-items: start; }
.detail-list, .brand-palette { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(100%, 10rem), 1fr)); gap: 1rem; margin: 0; }
.detail-list dt, .brand-palette dt { color: var(--m-muted); font-size: .85rem; }
.detail-list dd, .brand-palette dd { margin: .25rem 0 0; overflow-wrap: anywhere; }
.metrics-list dd { font-weight: 650; font-variant-numeric: tabular-nums; }
.brand-palette dd { display: flex; align-items: center; gap: .5rem; }
.color-swatch { display: inline-block; width: 1.5rem; height: 1.5rem; border-radius: .3rem; border: 1px solid var(--m-line); }
.organization-arena { max-width: 42rem; }
</style>
