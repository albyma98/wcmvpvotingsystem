<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch, watchEffect } from 'vue'
import { masterApi } from '../api'
import { masterSectionPath } from '../routes'
import { useMasterWorkspace } from '../workspace'
import DataTable from '../ui/DataTable.vue'
import MasterDialog from '../ui/MasterDialog.vue'
import MasterIcon from '../ui/MasterIcon.vue'
import ClubDomainsView from './ClubDomainsView.vue'
import PortalCopy from './PortalCopy.vue'
import PortalCreateForm from './PortalCreateForm.vue'
import PortalCredentials from './PortalCredentials.vue'
import { portalErrorCode } from './portalErrors'

const props = defineProps({ route: { type: Object, required: true } })
const { follow, navigate, setContext, useDirty } = useMasterWorkspace()
const tournament = computed(() => props.route.section === 'tournaments')
const label = computed(() => tournament.value ? 'Tornei' : 'ClubHub')
const createLabel = computed(() => tournament.value ? 'Nuovo torneo' : 'Nuovo ClubHub')
const listPath = computed(() => masterSectionPath(props.route.section))
const endpoint = computed(() => `/admin/master/${props.route.section}`)

const rows = ref([])
const loading = ref(true)
const listError = ref('')
const search = ref('')
const filter = ref('')
const busy = ref(false)
const createError = ref(null)
const actionError = ref('')
const notice = ref('')
const credentials = ref(null)
const dialog = ref('')
const password = ref('')
const numberFormat = new Intl.NumberFormat('it-IT')
let loadVersion = 0
let disposed = false

const name = { key: 'name', label: 'Nome', sortable: true }
const teams = { key: 'teamsCount', label: 'Squadre', numeric: true, sortable: true }
const actions = { key: 'actions', label: 'Azioni' }
const clubColumns = [
  name,
  { key: 'active', label: 'Stato', sortable: true },
  teams,
  { key: 'postsCount', label: 'Contenuti', numeric: true, sortable: true },
  { key: 'totalVisits', label: 'Visite totali', numeric: true, sortable: true },
  { key: 'uniqueVisits', label: 'Visite uniche', numeric: true, sortable: true },
  actions,
]
const tournamentColumns = [
  name,
  { key: 'dateLabel', label: 'Date' },
  { key: 'phaseLabel', label: 'Fase', sortable: true },
  teams,
  { key: 'matchesCount', label: 'Partite', numeric: true, sortable: true },
  { key: 'adminUsername', label: 'Username', sortable: true },
  actions,
]
const columns = computed(() => tournament.value ? tournamentColumns : clubColumns)

function recordId(row) {
  return tournament.value ? row.eventId : row.id
}
function detailPath(row) {
  return `${listPath.value}/${recordId(row)}`
}
function sitePath(row) {
  return row.slug ? `${tournament.value ? '/t/' : '/club/'}${encodeURIComponent(row.slug)}` : ''
}
function adminPath(row) {
  return row.slug ? `${tournament.value ? '/ta/' : '/club-admin/'}${encodeURIComponent(row.slug)}` : ''
}
function show(value) {
  return value || '—'
}
function number(value) {
  return value == null ? '—' : numberFormat.format(value)
}
function formatDate(value) {
  if (!value) return '—'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleDateString('it-IT')
}

const selected = computed(() => rows.value.find(row => recordId(row) === props.route.id))
const domains = computed(() => !tournament.value && props.route.tab === 'domains')
const visibleCredentials = computed(() => {
  const issued = credentials.value
  if (issued && (issued.id === props.route.id || (issued.data.adminPath && props.route.mode === 'list'))) return issued.data
  return null
})
const phases = computed(() => [...new Set(rows.value.map(row => row.phaseLabel).filter(Boolean))].sort((a, b) => a.localeCompare(b, 'it')))
const filteredRows = computed(() => {
  if (!filter.value) return rows.value
  if (tournament.value) return rows.value.filter(row => row.phaseLabel === filter.value)
  return rows.value.filter(row => row.active === (filter.value === 'active'))
})
const details = computed(() => {
  if (!selected.value) return []
  const row = selected.value
  if (tournament.value) {
    return [
      ['Slug', row.slug], ['Formato', row.format], ['Date', row.dateLabel],
      ['Luogo', row.location], ['Fase', row.phaseLabel], ['Stato', row.statusLabel],
    ]
  }
  return [
    ['Slug', row.slug], ['Città', row.city], ['ID società Live', row.organizationId],
    ['Data di creazione', formatDate(row.createdAt)],
  ]
})
const metrics = computed(() => {
  const row = selected.value
  if (!row) return []
  return tournament.value
    ? [['Squadre', row.teamsCount], ['Partite', row.matchesCount]]
    : [['Squadre', row.teamsCount], ['Contenuti', row.postsCount], ['Visite totali', row.totalVisits], ['Visite uniche', row.uniqueVisits]]
})
const searchText = row => [row.name, row.slug, row.city, row.dateLabel, row.location, row.format, row.adminUsername].filter(Boolean).join(' ')
useDirty(() => busy.value || (dialog.value === 'password' && Boolean(password.value)))

watchEffect(() => {
  if (props.route.mode === 'list') {
    setContext({ action: { label: createLabel.value, href: `${listPath.value}/create`, icon: 'plus' } })
  } else if (props.route.mode === 'create') {
    setContext({ title: createLabel.value })
  } else {
    setContext({ title: domains.value ? 'Domini personalizzati' : selected.value?.name || label.value })
  }
})

watch(() => props.route, ({ mode, id }, { mode: previousMode, id: previousId }) => {
  if (mode === 'create' && previousMode !== 'create') createError.value = null
  if (previousMode === 'detail' && id !== previousId && credentials.value?.id === previousId) credentials.value = null
  dialog.value = ''
  password.value = ''
})

async function load() {
  const version = ++loadVersion
  loading.value = true
  listError.value = ''
  try {
    const { data } = await masterApi.get(endpoint.value)
    if (!disposed && version === loadVersion) rows.value = data[props.route.section] ?? []
  } catch {
    if (!disposed && version === loadVersion) {
      listError.value = tournament.value ? 'Impossibile caricare i tornei.' : 'Impossibile caricare i ClubHub.'
    }
  } finally {
    if (!disposed && version === loadVersion) loading.value = false
  }
}

async function create(payload) {
  if (busy.value) return
  const createRoute = props.route
  busy.value = true
  createError.value = null
  notice.value = ''
  try {
    const { data } = await masterApi.post(endpoint.value, payload)
    if (disposed) return
    credentials.value = { id: recordId(data), data }
    if (props.route === createRoute) navigate(detailPath(data), { discard: true })
    await load()
  } catch (error) {
    if (disposed) return
    const code = portalErrorCode(error)
    if (code === 'name_required') createError.value = payload.slug?.trim()
      ? { field: 'slug', message: 'Lo slug deve contenere almeno una lettera o una cifra.' }
      : { field: 'name', message: 'Inserisci un nome che consenta di generare uno slug valido.' }
    else if (code === 'invalid_slug') createError.value = { field: 'name', message: 'Il nome deve contenere almeno una lettera o una cifra utilizzabile nello slug.' }
    else if (error?.response?.status === 409) {
      createError.value = tournament.value
        ? { field: 'name', message: 'Lo slug generato è già in uso. Scegli un nome diverso.' }
        : { field: payload.slug?.trim() ? 'slug' : 'name', message: 'Slug già utilizzato o società già presente.' }
    } else {
      createError.value = { message: tournament.value ? 'Impossibile creare il torneo. Riprova.' : 'Impossibile creare il ClubHub. Riprova.' }
    }
  } finally {
    busy.value = false
  }
}

function openDialog(action) {
  dialog.value = action
  actionError.value = ''
  password.value = ''
}
function closeDialog() {
  if (dialog.value === 'password' && password.value && !window.confirm('Scartare la password inserita?')) return
  dialog.value = ''
  actionError.value = ''
  password.value = ''
}

async function resetPassword() {
  if (busy.value || !selected.value) return
  actionError.value = ''
  const value = password.value.trim()
  const bytes = new TextEncoder().encode(value).length
  if (bytes > 0 && bytes < 6) {
    actionError.value = 'La password deve avere almeno 6 caratteri.'
    return
  }
  if (bytes > 72) {
    actionError.value = 'La password può contenere al massimo 72 byte.'
    return
  }
  const row = selected.value
  busy.value = true
  try {
    const { data } = await masterApi.put(`${endpoint.value}/${recordId(row)}/password`, { password: value })
    if (disposed) return
    credentials.value = { id: recordId(row), data }
    password.value = ''
    dialog.value = ''
    await load()
  } catch (error) {
    if (disposed) return
    actionError.value = portalErrorCode(error) === 'password_too_short'
      ? 'La password deve avere almeno 6 caratteri.'
      : 'Impossibile reimpostare la password. Riprova.'
  } finally {
    busy.value = false
  }
}

async function remove() {
  if (busy.value || !selected.value) return
  busy.value = true
  actionError.value = ''
  const id = recordId(selected.value)
  try {
    await masterApi.delete(`${endpoint.value}/${id}`)
    if (disposed) return
    rows.value = rows.value.filter(row => recordId(row) !== id)
    dialog.value = ''
    credentials.value = null
    notice.value = tournament.value ? 'Torneo eliminato.' : 'ClubHub eliminato.'
    if (props.route.id === id) navigate(listPath.value, { discard: true })
    await load()
  } catch {
    if (!disposed) actionError.value = 'Eliminazione non riuscita. Riprova.'
  } finally {
    busy.value = false
  }
}

onMounted(load)
onBeforeUnmount(() => { disposed = true })
</script>

<template>
  <div class="m-stack portal-view">
    <a v-if="route.mode !== 'list'" class="m-link portal-back" :href="listPath" @click="follow">
      <MasterIcon name="arrow-left" :size="16" /> Torna a {{ label }}
    </a>

    <PortalCreateForm v-if="route.mode === 'create'" :tournament="tournament" :list-path="listPath" :busy="busy" :error="createError" @submit="create" @clear-error="createError = null" />
    <template v-else>
      <PortalCredentials v-if="visibleCredentials && !domains" :credentials="visibleCredentials" :tournament="tournament" @dismiss="credentials = null" />
      <p v-if="notice && route.mode === 'list'" class="m-success" role="status">{{ notice }}</p>

      <section v-if="route.mode === 'list'" class="m-panel">
        <header class="m-panel-head">
          <div>
            <h2>{{ tournament ? 'Catalogo tornei' : 'Portali societari' }}</h2>
            <p v-if="tournament">Consulta lo stato dei tornei e gestisci gli accessi degli organizzatori.</p>
            <p v-else>Gestisci i siti societari, i domini personalizzati e l’accesso al CMS.</p>
          </div>
          <button class="m-btn" type="button" :disabled="loading" @click="load"><MasterIcon name="refresh" :size="16" /> Aggiorna</button>
        </header>
        <DataTable v-model:search="search" :rows="filteredRows" :columns="columns" :row-key="recordId" :search-text="searchText" :label="tournament ? 'Elenco tornei' : 'Elenco ClubHub'"
          :empty="tournament ? 'Nessun torneo trovato.' : 'Nessun ClubHub trovato.'" :loading="loading" :error="listError" @retry="load">
          <template #filters>
            <label class="m-field portal-filter">
              <span>{{ tournament ? 'Fase' : 'Stato' }}</span>
              <select v-model="filter" class="m-input">
                <option value="">{{ tournament ? 'Tutte le fasi' : 'Tutti gli stati' }}</option>
                <template v-if="tournament"><option v-for="phase in phases" :key="phase" :value="phase">{{ phase }}</option></template>
                <template v-else><option value="active">Attivi</option><option value="inactive">Inattivi</option></template>
              </select>
            </label>
          </template>
          <template #cell-name="{ row }">
            <a class="m-link portal-row-name" :href="detailPath(row)" @click="follow">{{ show(row.name) }}</a>
            <small v-if="row.slug || row.city" class="portal-row-meta m-muted-text">{{ [row.slug, row.city].filter(Boolean).join(' · ') }}</small>
          </template>
          <template #cell-active="{ value }"><span class="m-badge" :class="value ? 'm-good' : 'm-muted'">{{ value == null ? '—' : value ? 'Attivo' : 'Inattivo' }}</span></template>
          <template #cell-phaseLabel="{ value }"><span class="m-badge m-muted">{{ show(value) }}</span></template>
          <template #cell-adminUsername="{ value }">
            <span class="portal-username"><code>{{ show(value) }}</code><PortalCopy v-if="value" :value="value" label="Copia username" /></span>
          </template>
          <template #cell-teamsCount="{ value }">{{ number(value) }}</template>
          <template #cell-postsCount="{ value }">{{ number(value) }}</template>
          <template #cell-totalVisits="{ value }">{{ number(value) }}</template>
          <template #cell-uniqueVisits="{ value }">{{ number(value) }}</template>
          <template #cell-matchesCount="{ value }">{{ number(value) }}</template>
          <template #cell-actions="{ row }"><a class="m-btn m-subtle" :href="detailPath(row)" :aria-label="`Apri scheda di ${row.name}`" @click="follow">Apri scheda</a></template>
        </DataTable>
        <p v-if="!tournament" class="m-muted-text portal-visit-note">Le visite totali comprendono ogni apertura della home o di una pagina squadra dall’attivazione del tracciamento. Le visite uniche contano i browser distinti per ciascun ClubHub. Dati riservati al Master Admin.</p>
      </section>

      <template v-else>
        <div v-if="listError" class="m-panel">
          <p class="m-error" role="alert">{{ listError }}</p>
          <button class="m-btn" type="button" @click="load">Riprova</button>
        </div>
        <p v-if="loading && !selected" class="m-panel m-muted-text" role="status">Caricamento scheda…</p>
        <p v-else-if="!selected && !listError" class="m-panel m-muted-text">{{ tournament ? 'Torneo non trovato.' : 'ClubHub non trovato.' }}</p>

        <template v-if="selected">
          <nav v-if="!tournament" class="portal-tabs" aria-label="Sezioni della scheda ClubHub">
            <a :href="detailPath(selected)" :aria-current="domains ? undefined : 'page'" @click="follow">Panoramica</a>
            <a :href="`${detailPath(selected)}/domains`" :aria-current="domains ? 'page' : undefined" @click="follow">Domini personalizzati</a>
          </nav>
          <ClubDomainsView v-if="domains" :key="selected.id" :club="selected" />
          <template v-else>
            <section class="m-panel">
              <header class="m-panel-head">
                <div><h2>Dati generali</h2><p>{{ selected.name }}</p></div>
                <span v-if="!tournament && selected.active != null" class="m-badge" :class="selected.active ? 'm-good' : 'm-muted'">{{ selected.active ? 'Attivo' : 'Inattivo' }}</span>
              </header>
              <dl class="portal-detail-grid">
                <div v-for="[key, value] in details" :key="key">
                  <dt>{{ key }}</dt>
                  <dd>
                    <a v-if="key === 'ID società Live' && value > 0" class="m-link" :href="`${masterSectionPath('organizations')}/${value}`" @click="follow">#{{ value }}</a>
                    <template v-else>{{ show(value) }}</template>
                  </dd>
                </div>
              </dl>
            </section>
            <section class="m-panel">
              <header class="m-panel-head"><div><h2>{{ tournament ? 'Organizzazione' : 'Contenuti e visite' }}</h2><p v-if="!tournament">Visite registrate sulle pagine pubbliche dal momento dell’attivazione del tracciamento.</p></div></header>
              <dl class="portal-detail-grid portal-metrics">
                <div v-for="[key, value] in metrics" :key="key"><dt>{{ key }}</dt><dd>{{ number(value) }}</dd></div>
              </dl>
            </section>
            <section class="m-panel">
              <header class="m-panel-head">
                <div><h2>Link e accesso</h2><p>{{ tournament ? 'Accedi al sito dei tifosi o al pannello dell’organizzatore.' : 'Accedi al sito pubblico o al CMS del ClubHub.' }}</p></div>
              </header>
              <div class="m-actions">
                <a v-if="sitePath(selected)" class="m-btn" :href="sitePath(selected)" target="_blank" rel="noopener noreferrer">{{ tournament ? 'Portale tifosi' : 'Sito pubblico' }} <MasterIcon name="arrow-up-right" :size="16" /></a>
                <a v-if="adminPath(selected)" class="m-btn" :href="adminPath(selected)" target="_blank" rel="noopener noreferrer">{{ tournament ? 'Pannello admin' : 'CMS ClubHub' }} <MasterIcon name="arrow-up-right" :size="16" /></a>
              </div>
              <div class="portal-access">
                <div class="portal-username"><span>Username</span><code>{{ show(selected.adminUsername) }}</code><PortalCopy v-if="selected.adminUsername" :value="selected.adminUsername" label="Copia username" /></div>
                <button class="m-btn" type="button" :disabled="busy" @click="openDialog('password')"><MasterIcon name="key" :size="16" /> Reimposta password</button>
              </div>
            </section>
            <section class="m-panel">
              <header class="m-panel-head">
                <div><h2>Eliminazione</h2><p>{{ tournament ? 'Elimina il torneo e i dati associati.' : 'Elimina il portale e i suoi contenuti. La società Live collegata resta disponibile.' }}</p></div>
                <button class="m-btn m-danger" type="button" :disabled="busy" @click="openDialog('delete')"><MasterIcon name="trash" :size="16" /> {{ tournament ? 'Elimina torneo' : 'Elimina ClubHub' }}</button>
              </header>
            </section>
          </template>
        </template>
      </template>
    </template>

    <MasterDialog v-if="dialog === 'password'" title="Reimposta password" :busy="busy" @close="closeDialog">
      <form id="portal-password-form" class="m-stack" @submit.prevent="resetPassword">
        <p>Inserisci una nuova password oppure lascia vuoto per generarne una casuale.</p>
        <p v-if="tournament">L’operazione disconnette le sessioni attive del pannello torneo.</p>
        <label class="m-field">
          <span>Nuova password</span>
          <input v-model="password" class="m-input" type="password" autocomplete="new-password" placeholder="Vuoto per generarla" :disabled="busy"
            :aria-invalid="actionError ? 'true' : undefined" :aria-describedby="actionError ? 'portal-password-error' : undefined" @input="actionError = ''">
          <small v-if="actionError" id="portal-password-error" class="m-error" role="alert">{{ actionError }}</small>
        </label>
      </form>
      <template #actions>
        <button class="m-btn" type="button" :disabled="busy" @click="closeDialog">Annulla</button>
        <button class="m-btn m-primary" type="submit" form="portal-password-form" :disabled="busy">{{ busy ? 'Salvataggio…' : 'Salva password' }}</button>
      </template>
    </MasterDialog>

    <MasterDialog v-if="dialog === 'delete'" :title="tournament ? 'Elimina torneo' : 'Elimina ClubHub'" :busy="busy" @close="closeDialog">
      <p v-if="tournament">Eliminerai definitivamente <strong>{{ selected?.name }}</strong>, incluse squadre, partite, sponsor e accessi di amministrazione.</p>
      <p v-else>Eliminerai definitivamente <strong>{{ selected?.name }}</strong> e tutti i contenuti del portale. La società Live collegata resta disponibile.</p>
      <p v-if="actionError" class="m-error" role="alert">{{ actionError }}</p>
      <template #actions>
        <button class="m-btn" type="button" :disabled="busy" @click="closeDialog">Annulla</button>
        <button class="m-btn m-danger" type="button" :disabled="busy" @click="remove">{{ busy ? 'Eliminazione…' : 'Conferma eliminazione' }}</button>
      </template>
    </MasterDialog>
  </div>
</template>

<style scoped>
.portal-view { min-width: 0; }
.portal-back { align-self: start; display: inline-flex; align-items: center; gap: 6px; }
.portal-row-name { font-weight: 650; overflow-wrap: anywhere; }
.portal-row-meta { display: block; margin-top: 4px; overflow-wrap: anywhere; }
.portal-filter { min-width: 130px; }
.portal-view :deep(.m-table-tools) { align-items: flex-end; }
.portal-username { display: inline-flex; flex-wrap: wrap; align-items: center; gap: 6px 10px; }
.portal-username code { overflow-wrap: anywhere; }
.portal-visit-note { margin-bottom: 0; }
.portal-tabs { display: flex; flex-wrap: wrap; gap: 4px 18px; border-bottom: 1px solid var(--m-line); }
.portal-tabs a { padding: 8px 2px; border-bottom: 2px solid transparent; color: var(--m-muted); font-weight: 600; }
.portal-tabs a[aria-current="page"] { border-color: var(--m-accent); color: var(--m-ink); }
.portal-detail-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(180px, 100%), 1fr)); gap: 14px; margin: 0; }
.portal-detail-grid > div { min-width: 0; padding: 12px 14px; background: var(--m-soft); border-radius: var(--m-radius); }
.portal-detail-grid dt { color: var(--m-muted); font-size: .83em; }
.portal-detail-grid dd { margin: 6px 0 0; font-weight: 650; overflow-wrap: anywhere; }
.portal-metrics dd { font-size: 1.5rem; font-variant-numeric: tabular-nums; }
.portal-access { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 10px; padding-top: 16px; margin-top: 16px; border-top: 1px solid var(--m-line); }
</style>
