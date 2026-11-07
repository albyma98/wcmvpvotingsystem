<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { masterApi } from '../api'
import { useMasterWorkspace } from '../workspace'
import MasterDialog from '../ui/MasterDialog.vue'
import MasterIcon from '../ui/MasterIcon.vue'
import PortalCopy from './PortalCopy.vue'
import { portalErrorCode } from './portalErrors'

const props = defineProps({ club: { type: Object, required: true } })
const { useDirty } = useMasterWorkspace()
const domainData = ref(null)
const hostname = ref('')
const includeWWW = ref(true)
const loading = ref(true)
const operation = ref('')
const loadError = ref('')
const changeError = ref('')
const hostnameError = ref('')
const notice = ref('')
const removing = ref(null)
const removeError = ref('')
const domains = computed(() => domainData.value?.domains ?? [])
const records = computed(() => [...new Map(domains.value
  .filter(domain => domain.verificationName)
  .map(domain => [domain.verificationName, domain])).values()])
const targetIPs = computed(() => domainData.value?.targetIPs?.length
  ? domainData.value.targetIPs.join(', ')
  : `gli IP pubblici di ${domainData.value?.target || 'ArenaBoostX'}`)
const endpoint = `/admin/master/clubs/${props.club.id}/domains`
const fieldErrors = {
  invalid_domain: 'Inserisci un dominio pubblico valido, senza percorsi o porte. I domini della piattaforma sono riservati.',
  domain_taken: 'Questo dominio o il suo alias è già associato a un ClubHub.',
  domain_limit: 'Puoi associare fino a 10 domini a questo ClubHub.',
}

let timer
let pendingRead = false
let disposed = false
let version = 0
let savedWWW = includeWWW.value
useDirty(() => Boolean(operation.value || hostname.value || includeWWW.value !== savedWWW))

async function load(background = false) {
  if (pendingRead || operation.value || disposed) return
  const readVersion = version
  pendingRead = true
  if (!background) {
    loading.value = true
    loadError.value = ''
  }
  try {
    const { data } = await masterApi.get(endpoint)
    if (disposed || readVersion !== version) return
    domainData.value = data
    loadError.value = ''
  } catch {
    if (!disposed && readVersion === version) loadError.value = 'Impossibile aggiornare lo stato dei domini.'
  } finally {
    pendingRead = false
    loading.value = false
  }
}

async function change(action, send, successMessage) {
  if (operation.value || disposed) return false
  operation.value = action
  version += 1
  notice.value = ''
  changeError.value = ''
  removeError.value = ''
  hostnameError.value = ''
  try {
    const { data } = await send()
    if (disposed) return false
    domainData.value = data
    loadError.value = ''
    notice.value = successMessage
    return true
  } catch (error) {
    if (disposed) return false
    const message = fieldErrors[portalErrorCode(error)]
    if (action === 'add' && message) hostnameError.value = message
    else if (action === 'remove') removeError.value = 'Impossibile scollegare il dominio. Riprova.'
    else changeError.value = 'Operazione sui domini non riuscita. Riprova.'
    return false
  } finally {
    operation.value = ''
  }
}

async function add() {
  const name = hostname.value.trim()
  if (!name) {
    hostnameError.value = 'Inserisci il nome del dominio.'
    return
  }
  if (await change('add', () => masterApi.post(endpoint, { hostname: name, includeWWW: includeWWW.value }),
    'Dominio associato. Configura i record DNS per avviare la verifica.')) {
    savedWWW = includeWWW.value
    hostname.value = ''
  }
}

function verify() {
  return change('verify', () => masterApi.post(`${endpoint}/verify`), 'Verifica richiesta. Lo stato si aggiorna automaticamente.')
}

function makePrimary(domain) {
  return change('primary', () => masterApi.put(`${endpoint}/${domain.id}/primary`), 'Dominio principale aggiornato.')
}

async function remove() {
  if (await change('remove', () => masterApi.delete(`${endpoint}/${removing.value.id}`), 'Dominio scollegato.')) {
    removing.value = null
  }
}

function closeRemove() {
  removing.value = null
  removeError.value = ''
}

onMounted(() => {
  load()
  timer = window.setInterval(() => {
    if (document.visibilityState !== 'hidden') load(true)
  }, 15000)
})
onBeforeUnmount(() => {
  disposed = true
  window.clearInterval(timer)
})
</script>

<template>
  <div class="m-stack">
    <section class="m-panel">
      <header class="m-panel-head">
        <div><h2>Associa un dominio</h2><p>Collega il sito ufficiale a {{ club.name }}. L’autorizzazione avviene dopo la verifica dei record DNS.</p></div>
        <button class="m-btn" type="button" :disabled="Boolean(operation) || loading" @click="verify">
          <MasterIcon name="refresh" :size="16" /> {{ operation === 'verify' ? 'Verifica in corso…' : 'Verifica ora' }}
        </button>
      </header>
      <form class="domain-form" @submit.prevent="add">
        <label class="m-field">
          <span>Nuovo dominio</span>
          <input v-model="hostname" class="m-input" type="text" inputmode="url" autocomplete="off" spellcheck="false" placeholder="nomesocieta.it"
            required :disabled="Boolean(operation)" :aria-invalid="hostnameError ? 'true' : undefined"
            :aria-describedby="hostnameError ? 'club-domain-error' : undefined" @input="hostnameError = ''">
          <small v-if="hostnameError" id="club-domain-error" class="m-error" role="alert">{{ hostnameError }}</small>
        </label>
        <label class="domain-alias"><input v-model="includeWWW" type="checkbox" :disabled="Boolean(operation)"> Collega anche www</label>
        <button class="m-btn m-primary" type="submit" :disabled="Boolean(operation) || loading">
          {{ operation === 'add' ? 'Associazione…' : 'Associa dominio' }}
        </button>
      </form>
      <p v-if="changeError" class="m-error" role="alert">{{ changeError }}</p>
      <p v-if="notice" class="m-success" role="status">{{ notice }}</p>
    </section>

    <section class="m-panel">
      <header class="m-panel-head">
        <div><h2>Indirizzi associati</h2><p>Controlla l’autorizzazione e scegli l’indirizzo principale del sito.</p></div>
        <button v-if="loadError" class="m-btn" type="button" :disabled="Boolean(operation) || loading" @click="load()">Riprova</button>
      </header>
      <p v-if="loadError" class="m-error" role="alert">{{ loadError }}</p>
      <p v-if="loading" class="m-muted-text" role="status">Caricamento domini…</p>
      <p v-else-if="domainData && !domains.length" class="m-muted-text">Nessun dominio associato. Il sito è disponibile all’indirizzo ClubHub della piattaforma.</p>
      <div v-if="domains.length" class="m-table-scroll" tabindex="0" role="region" aria-label="Elenco domini associati">
        <table class="m-table">
          <thead><tr><th scope="col">Dominio</th><th scope="col">Stato DNS</th><th scope="col">Ultimo controllo</th><th scope="col">Azioni</th></tr></thead>
          <tbody>
            <tr v-for="domain in domains" :key="domain.id">
              <td><strong class="domain-host">{{ domain.hostname }}</strong><span v-if="domain.primary" class="m-badge m-muted">Principale</span></td>
              <td>
                <span class="m-badge" :class="domain.status === 'active' ? 'm-good' : 'm-warn'">{{ domain.status === 'active' ? 'Autorizzato' : 'In attesa dei DNS' }}</span>
                <small v-if="domain.message" class="domain-status-detail m-muted-text">{{ domain.message }}</small>
              </td>
              <td>{{ domain.checkedAt ? new Date(domain.checkedAt).toLocaleString('it-IT') : '—' }}</td>
              <td>
                <div class="m-actions domain-actions">
                  <a v-if="domain.status === 'active'" class="m-link" :href="`https://${domain.hostname}`" target="_blank" rel="noopener noreferrer">Apri sito <MasterIcon name="arrow-up-right" :size="14" /></a>
                  <button v-if="!domain.primary" class="m-btn m-subtle" type="button" :disabled="Boolean(operation)" @click="makePrimary(domain)">Rendi principale</button>
                  <button class="m-btn m-danger" type="button" :disabled="Boolean(operation)" @click="removing = domain">Scollega</button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <section v-if="domains.length" class="m-panel">
      <header class="m-panel-head"><div><h2>Configurazione DNS</h2><p>Inserisci questi record presso il gestore del dominio e mantienili attivi.</p></div></header>
      <p>Configura i record A/AAAA verso <strong>{{ targetIPs }}</strong>. Per gli indirizzi www puoi usare un CNAME verso <strong>{{ domainData?.target || '—' }}</strong>.</p>
      <div v-if="records.length" class="m-table-scroll" tabindex="0" role="region" aria-label="Record TXT di verifica">
        <table class="m-table">
          <thead><tr><th scope="col">Tipo</th><th scope="col">Nome / Host</th><th scope="col">Valore</th><th scope="col">Azioni</th></tr></thead>
          <tbody>
            <tr v-for="record in records" :key="record.verificationName">
              <td>TXT</td>
              <td><code>{{ record.verificationName }}</code></td>
              <td><code>{{ record.verificationValue }}</code></td>
              <td><PortalCopy v-if="record.verificationValue" :value="record.verificationValue" label="Copia valore" /></td>
            </tr>
          </tbody>
        </table>
      </div>
      <p class="m-muted-text">Il controllo viene ripetuto ogni minuto. Il certificato HTTPS viene generato al primo accesso a un dominio autorizzato.</p>
    </section>

    <MasterDialog v-if="removing" title="Scollega dominio" :busy="operation === 'remove'" @close="closeRemove">
      <p>Il portale {{ club.name }} non sarà più accessibile da <strong>{{ removing.hostname }}</strong>.</p>
      <p v-if="removeError" class="m-error" role="alert">{{ removeError }}</p>
      <template #actions>
        <button class="m-btn" type="button" :disabled="Boolean(operation)" @click="closeRemove">Annulla</button>
        <button class="m-btn m-danger" type="button" :disabled="Boolean(operation)" @click="remove">{{ operation === 'remove' ? 'Rimozione…' : 'Scollega dominio' }}</button>
      </template>
    </MasterDialog>
  </div>
</template>

<style scoped>
.domain-form { display: grid; grid-template-columns: minmax(170px, 1fr) auto auto; align-items: start; gap: 12px 16px; }
.domain-form > button, .domain-alias { margin-top: 24px; }
.domain-alias { display: flex; align-items: center; gap: 8px; min-height: 38px; }
.domain-alias input { width: 16px; height: 16px; accent-color: var(--m-accent); }
.domain-host, .domain-status-detail { display: block; margin-bottom: 4px; overflow-wrap: anywhere; }
.domain-actions { justify-content: flex-start; }
.m-table code { overflow-wrap: anywhere; }
@media (max-width: 780px) {
  .domain-form { grid-template-columns: minmax(0, 1fr); }
  .domain-form > button, .domain-alias { margin-top: 0; }
  .domain-form > button { justify-self: start; }
}
</style>
