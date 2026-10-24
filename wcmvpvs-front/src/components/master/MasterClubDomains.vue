<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'

const props = defineProps({ club: { type: Object, required: true } })
const domains = ref([])
const target = ref('')
const targetIPs = ref([])
const hostname = ref('')
const includeWWW = ref(true)
const loading = ref(true)
const busy = ref(false)
const error = ref('')
const notice = ref('')
const verificationRecords = computed(() => [...new Map(domains.value.map(domain => [domain.verificationName, domain])).values()])
const endpoint = () => `/api/admin/master/clubs/${props.club.id}/domains`
let timer, disposed = false
let requestSequence = 0

const errors = {
  invalid_domain: 'Inserisci un dominio pubblico valido, senza percorsi o porte. I domini della piattaforma sono riservati.',
  domain_taken: 'Questo dominio o il suo alias è già associato a un ClubHub.',
  domain_limit: 'Puoi associare fino a 10 domini a questo club.',
}
function apply(data) {
  if (disposed) return
  domains.value = data.domains || []
  target.value = data.target || ''
  targetIPs.value = data.targetIPs || []
}
async function load() {
  if (busy.value) return
  const sequence = ++requestSequence
  try {
    const response = await fetch(endpoint(), { credentials: 'include', cache: 'no-store' })
    if (!response.ok) throw new Error()
    const data = await response.json()
    if (sequence === requestSequence) { apply(data); error.value = '' }
  } catch { if (!disposed && sequence === requestSequence) error.value = 'Impossibile aggiornare lo stato dei domini.' }
  finally { loading.value = false }
}
async function change(method, suffix = '', payload) {
  if (busy.value) return false
  busy.value = true
  const sequence = ++requestSequence
  error.value = ''
  notice.value = ''
  try {
    const response = await fetch(endpoint() + suffix, {
      method, credentials: 'include', headers: { 'Content-Type': 'application/json' },
      body: payload === undefined ? undefined : JSON.stringify(payload),
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) throw new Error(errors[data.error] || 'Operazione non riuscita. Riprova.')
    if (sequence === requestSequence) apply(data)
    if (suffix === '/verify') notice.value = 'Verifica richiesta. Lo stato si aggiorna automaticamente.'
    return true
  } catch (cause) { error.value = cause.message; return false }
  finally { busy.value = false }
}
async function add() {
  if (!hostname.value.trim()) return
  if (await change('POST', '', { hostname: hostname.value.trim(), includeWWW: includeWWW.value })) hostname.value = ''
}
function remove(domain) {
  if (window.confirm(`Scollegare ${domain.hostname} da ${props.club.name}? Il portale non sarà più accessibile da questo indirizzo.`)) return change('DELETE', `/${domain.id}`)
}
function copy(value) { navigator.clipboard?.writeText(value) }
onMounted(async () => {
  await load()
  if (!disposed) timer = setInterval(() => { if (!busy.value && document.visibilityState !== 'hidden') load() }, 15000)
})
onBeforeUnmount(() => { disposed = true; clearInterval(timer) })
</script>

<template>
  <section class="club-domains" aria-label="Domini personalizzati">
    <header><div><h3>Domini personalizzati</h3><p>Collega il sito ufficiale a {{ club.name }}. Dopo aver inserito i record DNS, verifica e autorizzazione avvengono automaticamente.</p></div><button type="button" :disabled="busy || loading" @click="change('POST', '/verify')">{{ busy ? 'Operazione in corso…' : 'Verifica ora' }}</button></header>
    <form class="domain-form" @submit.prevent="add">
      <label>Nuovo dominio<input v-model="hostname" type="text" inputmode="url" autocomplete="off" spellcheck="false" placeholder="nomesocieta.it" :disabled="busy" required /></label>
      <label class="www-option"><input v-model="includeWWW" type="checkbox" :disabled="busy" /> Collega anche www</label>
      <button type="submit" class="primary" :disabled="busy || !hostname.trim()">Associa dominio</button>
    </form>
    <p v-if="error" class="domain-error" role="alert">{{ error }}</p>
    <p v-if="notice" role="status">{{ notice }}</p>
    <p v-if="loading" role="status">Caricamento domini…</p>
    <template v-else-if="domains.length">
      <div class="dns-guide"><h4>Record da inserire presso il gestore del dominio</h4><p>Configura ogni indirizzo con record A/AAAA verso <strong>{{ targetIPs.length ? targetIPs.join(', ') : `gli IP pubblici di ${target}` }}</strong>. Per gli indirizzi www puoi usare un CNAME verso <strong>{{ target }}</strong>. Aggiungi poi i record TXT qui sotto, mantenendoli attivi.</p>
        <div class="dns-record" v-for="record in verificationRecords" :key="record.verificationName"><b>TXT</b><div><span>Nome / Host</span><code>{{ record.verificationName }}</code><span>Valore</span><code>{{ record.verificationValue }}</code></div><button type="button" @click="copy(record.verificationValue)">Copia valore</button></div>
        <small>Il controllo viene ripetuto ogni minuto. Il certificato HTTPS viene generato al primo accesso a un dominio autorizzato.</small>
      </div>
      <article v-for="domain in domains" :key="domain.id" class="domain-row"><div><strong>{{ domain.hostname }}</strong><span v-if="domain.primary" class="primary-badge">Principale</span><span class="domain-status" :class="domain.status">{{ domain.status === 'active' ? 'Autorizzato' : 'In attesa dei DNS' }}</span><p>{{ domain.message }}</p><small v-if="domain.checkedAt">Ultimo controllo: {{ new Date(domain.checkedAt).toLocaleString('it-IT') }}</small></div><div class="domain-actions"><a v-if="domain.status === 'active'" :href="`https://${domain.hostname}`" target="_blank" rel="noopener noreferrer">Apri sito ↗</a><button v-if="!domain.primary" type="button" :disabled="busy" @click="change('PUT', `/${domain.id}/primary`)">Rendi principale</button><button type="button" class="remove" :disabled="busy" @click="remove(domain)">Scollega</button></div></article>
    </template>
    <p v-else class="domain-hint">Nessun dominio associato. Il sito resta disponibile all’indirizzo ClubHub della piattaforma.</p>
  </section>
</template>

<style scoped>
.club-domains{grid-column:1/-1;min-width:0;padding:20px;border:1px solid #334155;border-radius:14px;background:#07101f;color:#e2e8f0}.club-domains>header{display:flex;align-items:flex-start;justify-content:space-between;gap:20px}.club-domains h3{margin:0;font-size:18px}.club-domains p{margin:7px 0;color:#94a3b8;font-size:13px;line-height:1.5}.club-domains button{flex:none;min-height:38px;padding:8px 12px;border:1px solid #475569;border-radius:8px;background:#172033;color:#e2e8f0;font:inherit;font-size:12px;cursor:pointer}.club-domains button:disabled{opacity:.5;cursor:default}.club-domains :is(button,input,a):focus-visible{outline:2px solid #fbbf24;outline-offset:3px}.domain-form{display:flex;align-items:end;flex-wrap:wrap;gap:14px;margin:20px 0}.domain-form>label:first-child{flex:1;min-width:180px;display:grid;gap:6px;color:#cbd5e1;font-size:12px}.domain-form input[type=text]{min-width:0;width:100%;padding:11px 12px;border:1px solid #475569;border-radius:8px;background:#0f172a;color:#fff;font:inherit}.www-option{display:flex;align-items:center;gap:8px;min-height:40px;font-size:12px}.www-option input{accent-color:#f59e0b;width:16px;height:16px}.club-domains .primary{background:#f59e0b;border-color:#f59e0b;color:#111827;font-weight:800}.club-domains .domain-error{color:#fca5a5}.dns-guide{margin:18px 0;padding:16px;border:1px solid #334155;border-radius:10px;background:#0f172a}.dns-guide h4{margin:0 0 10px;font-size:14px}.dns-guide strong{color:#e2e8f0;overflow-wrap:anywhere}.dns-guide>small,.domain-row small{display:block;color:#94a3b8;font-size:11px;line-height:1.5}.dns-record{display:flex;align-items:start;gap:12px;margin:14px 0;padding:13px;border:1px solid #334155;border-radius:8px;background:#07101f}.dns-record>b{color:#fbbf24;font-size:12px}.dns-record>div{flex:1;min-width:0}.dns-record span{display:block;margin-bottom:4px;color:#94a3b8;font-size:10px}.dns-record code{display:block;margin-bottom:10px;color:#f8fafc;white-space:normal;overflow-wrap:anywhere;font-size:12px}.domain-row{display:flex;align-items:center;justify-content:space-between;gap:16px;padding:16px 0;border-top:1px solid #293548}.domain-row>div:first-child{min-width:0}.domain-row strong{overflow-wrap:anywhere}.domain-status,.primary-badge{display:inline-block;padding:4px 7px;margin:6px 0 0 8px;border-radius:6px;font-size:10px}.primary-badge{background:#334155;color:#e2e8f0}.domain-status{background:#3b2a12;color:#fcd34d}.domain-status.active{background:#064e3b;color:#a7f3d0}.domain-actions{display:flex;flex-wrap:wrap;align-items:center;justify-content:end;gap:8px}.domain-actions a{color:#fbbf24;font-size:12px;white-space:nowrap}.club-domains .remove{color:#fca5a5}.club-domains :is(input,button){box-sizing:border-box}@media(max-width:700px){.club-domains{padding:14px}.club-domains>header,.domain-row{flex-direction:column;align-items:stretch}.domain-actions{justify-content:start}.dns-record{flex-wrap:wrap}.dns-record>div{flex-basis:calc(100% - 45px)}.domain-form>label:first-child{flex-basis:100%}.domain-status,.primary-badge{margin-left:0;margin-right:8px}}
</style>
