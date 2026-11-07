<script setup>
import { computed, nextTick, onScopeDispose, ref, watch, watchEffect } from 'vue';
import { masterApi } from '../api';
import { useMasterWorkspace } from '../workspace';
import DataTable from '../ui/DataTable.vue';
import MasterDialog from '../ui/MasterDialog.vue';
import MasterIcon from '../ui/MasterIcon.vue';
import OrganizationDetail from './OrganizationDetail.vue';
import OrganizationForm from './OrganizationForm.vue';
import { adminOrganizationUrl, organizationDate, ORGANIZATIONS_ENDPOINT, ORGANIZATIONS_PATH, ORGANIZATION_SPORTS, publicOrganizationUrl, smsPrice } from './organizationFields';

const props = defineProps({ route: { type: Object, required: true } });
const { follow, navigate, setContext, useDirty } = useMasterWorkspace();
const organizations = ref([]);
const detail = ref(null);
const loading = ref(false);
const loadError = ref('');
const notice = ref(null);
const refresh = ref(0);
const search = ref('');
const sport = ref('');
const status = ref('');
const deleteTarget = ref(null);
const deletePassword = ref('');
const deleteError = ref('');
const deleting = ref(false);
const listTitle = ref(null);
const passwordInput = ref(null);
let disposed = false;

useDirty(deleting);
onScopeDispose(() => { disposed = true; });

const organizationHref = (id) => `${ORGANIZATIONS_PATH}/${id}`;
const editHref = (id) => `${organizationHref(id)}/edit`;
const createHref = `${ORGANIZATIONS_PATH}/create`;
const filtered = computed(() => organizations.value.filter((organization) => (
  (!sport.value || (sport.value === 'unset' ? !organization.sport : organization.sport === sport.value))
  && (!status.value || Boolean(organization.is_active) === (status.value === 'active'))
)));
const filtersApplied = computed(() => Boolean(sport.value || status.value));
const visibleNotice = computed(() => {
  if (!notice.value) return null;
  if (props.route.mode === 'list' && notice.value.id == null) return notice.value;
  if (props.route.mode === 'detail' && props.route.id === notice.value.id) return notice.value;
  return null;
});
const searchText = (row) => [row.name, row.slug, row.sport, row.city, `ID ${row.id}`].filter(Boolean).join(' ');
const dateSortValue = (row) => {
  const timestamp = Date.parse(row.created_at);
  return Number.isFinite(timestamp) ? timestamp : null;
};
const columns = [
  { key: 'name', label: 'Società', sortable: true },
  { key: 'sport', label: 'Sport e città', sortable: true },
  { key: 'slug', label: 'Slug o URL', sortable: true },
  { key: 'is_active', label: 'Stato', sortable: true, value: (row) => row.is_active ? 'Attiva' : 'Disattiva' },
  { key: 'bar_enabled', label: 'BAR', sortable: true, value: (row) => row.bar_enabled ? 'Attivo' : 'Disattivo' },
  { key: 'sms_cost', label: 'Costo SMS', sortable: true, numeric: true },
  { key: 'free_sms', label: 'SMS gratuiti', sortable: true, numeric: true },
  { key: 'created_at', label: 'Creata il', sortable: true, value: dateSortValue, numeric: true },
  { key: 'actions', label: 'Azioni' },
];

watchEffect(() => {
  const { mode, id } = props.route;
  const name = detail.value?.organization?.id === id ? detail.value.organization.name : '';
  const context = {
    list: { action: { label: 'Nuova società', href: createHref, icon: 'plus' } },
    create: { title: 'Nuova società' },
    edit: { title: name ? `Modifica ${name}` : 'Modifica società' },
    detail: { title: name || 'Scheda società', action: name ? { label: 'Modifica società', href: editHref(id), icon: 'edit' } : undefined },
  };
  setContext(context[mode]);
});

watch([() => props.route.mode, () => props.route.id], () => {
  if (!visibleNotice.value) notice.value = null;
  closeDelete();
});

watch([() => props.route.mode, () => props.route.id, refresh], async ([mode, id], _, onCleanup) => {
  loadError.value = '';
  detail.value = null;
  if (mode === 'create') {
    loading.value = false;
    return;
  }

  const request = new AbortController();
  onCleanup(() => request.abort());
  loading.value = true;
  try {
    const { data } = await masterApi.get(mode === 'list' ? ORGANIZATIONS_ENDPOINT : `${ORGANIZATIONS_ENDPOINT}/${id}`, { signal: request.signal });
    if (request.signal.aborted) return;
    if (mode === 'list') organizations.value = Array.isArray(data) ? data : [];
    else if (data?.organization) detail.value = data;
    else loadError.value = 'Dettagli della società non disponibili.';
  } catch (error) {
    if (request.signal.aborted) return;
    loadError.value = error?.response?.status === 404 ? 'Società non trovata.' : mode === 'list'
      ? 'Impossibile caricare l’elenco delle società.'
      : 'Impossibile caricare i dettagli della società.';
  } finally {
    if (!request.signal.aborted) loading.value = false;
  }
}, { immediate: true });

function refreshList() {
  refresh.value += 1;
}

function resetFilters() {
  sport.value = '';
  status.value = '';
}

function updateOrganization(organization) {
  if (detail.value?.organization?.id === organization.id) detail.value = { ...detail.value, organization };
  organizations.value = organizations.value.map((item) => item.id === organization.id ? organization : item);
}

function afterSave(organization) {
  const { id, mode } = props.route;
  const savedId = organization?.id || id;
  notice.value = { id: savedId, message: mode === 'create' ? 'Società creata.' : 'Modifiche alla società salvate.' };
  navigate(savedId ? organizationHref(savedId) : ORGANIZATIONS_PATH, { discard: true });
}

function startDelete(organization) {
  deleteTarget.value = organization;
  deletePassword.value = '';
  deleteError.value = '';
  notice.value = null;
}

function closeDelete() {
  if (deleting.value) return;
  deleteTarget.value = null;
  deletePassword.value = '';
  deleteError.value = '';
}

async function removeOrganization() {
  const organization = deleteTarget.value;
  if (!organization || !deletePassword.value || deleting.value) return;
  deleting.value = true;
  deleteError.value = '';
  try {
    const { data } = await masterApi.delete(`${ORGANIZATIONS_ENDPOINT}/${organization.id}`, { data: { password: deletePassword.value } });
    if (disposed) return;
    deleteTarget.value = null;
    organizations.value = organizations.value.filter(({ id }) => id !== organization.id);
    notice.value = data?.media_cleanup_failed
      ? { warning: true, message: `${organization.name} eliminata. Alcuni file multimediali non sono stati rimossi. Consulta i log del server.` }
      : { message: `${organization.name} e i dati collegati sono stati eliminati.` };
    if (props.route.id === organization.id) navigate(ORGANIZATIONS_PATH, { discard: true });
    else {
      await nextTick();
      listTitle.value?.focus({ preventScroll: true });
      refreshList();
    }
  } catch (error) {
    const messages = {
      403: 'Password del superadmin non corretta.',
      404: 'Società non trovata.',
      409: 'La società contiene dati condivisi o protetti e non può essere eliminata automaticamente.',
      429: 'Troppi tentativi. Attendi qualche minuto e riprova.',
    };
    deleteError.value = messages[error?.response?.status] || 'Impossibile completare l’eliminazione. Aggiorna l’elenco prima di riprovare.';
  } finally {
    deletePassword.value = '';
    deleting.value = false;
    if (deleteTarget.value && !disposed) {
      await nextTick();
      passwordInput.value?.focus();
    }
  }
}
</script>

<template>
  <div class="m-stack organizations-view">
    <p v-if="visibleNotice" :class="visibleNotice.warning ? 'm-error' : 'm-success'" role="status">{{ visibleNotice.message }}</p>

    <section v-if="route.mode === 'list'" class="m-panel m-stack">
      <header class="m-panel-head">
        <div><h2 ref="listTitle" tabindex="-1">Elenco società</h2><p>Consulta anagrafiche, servizi e collegamenti alle pagine delle società.</p></div>
        <button class="m-btn" type="button" :disabled="loading" @click="refreshList"><MasterIcon name="refresh" :size="15" /> Aggiorna elenco</button>
      </header>
      <DataTable
        v-model:search="search"
        :rows="filtered"
        :columns="columns"
        :search-text="searchText"
        search-placeholder="Cerca società, città o slug…"
        label="Elenco società sportive"
        :empty="filtersApplied ? 'Nessuna società corrisponde ai filtri.' : 'Nessuna società registrata.'"
        :loading="loading"
        :error="loadError"
        @retry="refreshList"
      >
        <template #filters>
          <label>
            <span class="m-sr">Filtra per sport</span>
            <select v-model="sport" class="m-input">
              <option value="">Tutti gli sport</option>
              <option v-for="item in ORGANIZATION_SPORTS" :key="item" :value="item">{{ item }}</option>
              <option value="unset">Sport da impostare</option>
            </select>
          </label>
          <label>
            <span class="m-sr">Filtra per stato</span>
            <select v-model="status" class="m-input">
              <option value="">Tutti gli stati</option>
              <option value="active">Attive</option>
              <option value="inactive">Disattive</option>
            </select>
          </label>
          <button v-if="filtersApplied" class="m-btn m-subtle" type="button" @click="resetFilters">Azzera filtri</button>
        </template>
        <template #cell-name="{ row }">
          <div class="organization-cell">
            <div class="organization-logo">
              <img v-if="row.logo_url" :src="row.logo_url" alt="" loading="lazy" />
              <MasterIcon v-else name="building" :size="17" />
            </div>
            <div class="cell-text">
              <a class="m-link organization-name" :href="organizationHref(row.id)" @click="follow">{{ row.name || 'Società senza nome' }}</a>
              <small class="m-muted-text">ID {{ row.id }}</small>
            </div>
          </div>
        </template>
        <template #cell-sport="{ row }">
          <div class="cell-text sport-cell"><span>{{ row.sport || 'Da impostare' }}</span><small class="m-muted-text">{{ row.city || '—' }}</small></div>
        </template>
        <template #cell-slug="{ row }">
          <div v-if="row.slug" class="cell-text slug-cell">
            <a :href="publicOrganizationUrl(row.slug)" class="m-link" target="_blank" rel="noopener noreferrer">{{ row.slug }}</a>
            <small><a :href="adminOrganizationUrl(row)" class="m-link" :aria-label="`Pannello amministrativo di ${row.name}`" target="_blank" rel="noopener noreferrer">Admin</a></small>
          </div>
          <span v-else>—</span>
        </template>
        <template #cell-is_active="{ row, value }"><span :class="['m-badge', row.is_active ? 'm-good' : 'm-muted']">{{ value }}</span></template>
        <template #cell-bar_enabled="{ row, value }"><span :class="['m-badge', row.bar_enabled ? 'm-good' : 'm-muted']">{{ value }}</span></template>
        <template #cell-sms_cost="{ value }">{{ value == null ? '—' : smsPrice.format(value) }}</template>
        <template #cell-created_at="{ row }">{{ organizationDate(row.created_at) }}</template>
        <template #cell-actions="{ row }">
          <div class="m-actions row-actions">
            <a :href="editHref(row.id)" class="m-btn m-subtle m-icon-btn" :aria-label="`Modifica ${row.name}`" title="Modifica" @click="follow"><MasterIcon name="edit" :size="16" /></a>
            <button type="button" class="m-btn m-subtle m-icon-btn remove-button" :aria-label="`Elimina ${row.name}`" title="Elimina" @click="startDelete(row)"><MasterIcon name="trash" :size="16" /></button>
          </div>
        </template>
      </DataTable>
    </section>

    <template v-else>
      <div class="m-actions">
        <a class="m-btn m-subtle" :href="ORGANIZATIONS_PATH" @click="follow"><MasterIcon name="arrow-left" :size="16" /> Elenco società</a>
      </div>
      <p v-if="loading" class="m-panel" role="status">Caricamento dei dettagli della società…</p>
      <div v-else-if="loadError" class="m-panel m-stack">
        <p class="m-error" role="alert">{{ loadError }}</p>
        <div><button type="button" class="m-btn" @click="refreshList">Riprova</button></div>
      </div>
      <OrganizationForm v-else-if="route.mode === 'create'" :cancel-href="ORGANIZATIONS_PATH" @saved="afterSave" />
      <OrganizationForm v-else-if="route.mode === 'edit' && detail" :key="route.id" :organization="detail.organization" :cancel-href="organizationHref(route.id)" @saved="afterSave" />
      <OrganizationDetail v-else-if="detail" :key="route.id" :organization="detail.organization" :stats="detail.stats || {}" :tab="route.tab" @updated="updateOrganization" @remove="startDelete" />
    </template>

    <MasterDialog
      v-if="deleteTarget"
      title="Elimina definitivamente la società"
      :description="`Stai per eliminare ${deleteTarget.name} (ID ${deleteTarget.id}) con eventi, voti, fan, sponsor, contenuti e account collegati. L’operazione è irreversibile.`"
      :busy="deleting"
      @close="closeDelete"
    >
      <form id="m-org-delete" class="m-stack" @submit.prevent="removeOrganization">
        <label class="m-field">
          <span>Password del superadmin</span>
          <input ref="passwordInput" v-model="deletePassword" class="m-input" type="password" autocomplete="current-password" required autofocus :disabled="deleting" :aria-invalid="Boolean(deleteError)" :aria-describedby="deleteError ? 'm-org-delete-error' : undefined" @input="deleteError = ''" />
        </label>
        <p v-if="deleteError" id="m-org-delete-error" class="m-error" role="alert">{{ deleteError }}</p>
      </form>
      <template #actions>
        <button type="button" class="m-btn" :disabled="deleting" @click="closeDelete">Annulla</button>
        <button type="submit" form="m-org-delete" class="m-btn m-danger" :disabled="deleting || !deletePassword">{{ deleting ? 'Eliminazione…' : 'Elimina definitivamente' }}</button>
      </template>
    </MasterDialog>
  </div>
</template>

<style scoped>
.organizations-view :deep(.m-table) { min-width: 60rem; }
.organization-cell { display: flex; align-items: center; gap: .65rem; min-width: 9.5rem; }
.organization-logo { display: grid; place-items: center; width: 2rem; height: 2rem; flex: none; overflow: hidden; border-radius: .45rem; border: 1px solid var(--m-line); background: var(--m-soft); color: var(--m-accent); }
.organization-logo img { width: 100%; height: 100%; object-fit: contain; }
.cell-text { display: grid; gap: .2rem; min-width: 0; overflow-wrap: anywhere; }
.sport-cell { min-width: 6rem; }
.slug-cell { min-width: 7rem; max-width: 15rem; }
.organization-name { font-weight: 650; }
.row-actions { flex-wrap: nowrap; gap: .15rem; }
.remove-button:hover { color: #a42438; }
</style>
