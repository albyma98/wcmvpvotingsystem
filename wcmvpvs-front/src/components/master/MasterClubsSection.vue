<script setup>
import { onMounted, reactive, ref } from 'vue'

const clubs = ref([])
const loading = ref(false)
const submitting = ref(false)
const error = ref('')
const created = ref(null)
const deleting = ref(null)
const passwordOpen = ref(null)
const passwordInput = ref('')
const passwordResult = ref(null)

const form = reactive({
  name: '', slug: '', city: '', tagline: '', logoUrl: '', heroImageUrl: '',
  primaryColor: '#172554', secondaryColor: '#f59e0b', seasonLabel: 'Stagione 2026/27',
  categoriesText: 'Prima Divisione Maschile\nPrima Divisione Femminile\nGiovanile 1\nGiovanile 2',
})

const abs = path => `${window.location.origin}${path}`
const copy = text => navigator.clipboard?.writeText(text)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const response = await fetch('/api/admin/master/clubs', { credentials: 'include' })
    if (!response.ok) throw new Error(response.status)
    clubs.value = (await response.json()).clubs ?? []
  } catch {
    error.value = 'Impossibile caricare i Club Hub.'
  } finally { loading.value = false }
}

async function create() {
  if (!form.name.trim()) { error.value = 'Il nome della società è obbligatorio.'; return }
  submitting.value = true
  error.value = ''
  try {
    const payload = {
      ...form,
      categories: form.categoriesText.split('\n').map(value => value.trim()).filter(Boolean),
    }
    delete payload.categoriesText
    const response = await fetch('/api/admin/master/clubs', {
      method: 'POST', credentials: 'include',
      headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload),
    })
    const data = await response.json().catch(() => ({}))
    if (response.status === 409) { error.value = 'Slug già utilizzato o società già presente.'; return }
    if (!response.ok) throw new Error(data.error || response.status)
    created.value = data
    form.name = ''; form.slug = ''; form.city = ''; form.tagline = ''; form.logoUrl = ''; form.heroImageUrl = ''
    await load()
  } catch { error.value = 'Creazione del Club Hub fallita.' }
  finally { submitting.value = false }
}

async function resetPassword(club) {
  error.value = ''
  const response = await fetch(`/api/admin/master/clubs/${club.id}/password`, {
    method: 'PUT', credentials: 'include', headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ password: passwordInput.value.trim() }),
  })
  const data = await response.json().catch(() => ({}))
  if (!response.ok) { error.value = data.error === 'password_too_short' ? 'Password: minimo 6 caratteri.' : 'Reset fallito.'; return }
  passwordResult.value = { id: club.id, ...data }
  passwordOpen.value = null
  passwordInput.value = ''
}

async function remove(club) {
  if (!window.confirm(`Eliminare il Club Hub “${club.name}” e tutti i suoi contenuti?\nLa società Live collegata non verrà eliminata.`)) return
  deleting.value = club.id
  try {
    const response = await fetch(`/api/admin/master/clubs/${club.id}`, { method: 'DELETE', credentials: 'include' })
    if (!response.ok) throw new Error(response.status)
    await load()
  } catch { error.value = 'Eliminazione fallita.' }
  finally { deleting.value = null }
}

onMounted(load)
</script>

<template>
  <div class="club-master">
    <section class="panel">
      <div class="heading">
        <div><p class="eyebrow">ArenaBoostX Club</p><h2>Crea un portale societario</h2></div>
        <span class="pill">PRODOTTO ANNUALE</span>
      </div>
      <p class="hint">Crea la società, le categorie iniziali e le credenziali del CMS in un solo passaggio.</p>
      <div class="form-grid">
        <label>Nome società<input v-model="form.name" placeholder="Volley Club Aurora" /></label>
        <label>Slug facoltativo<input v-model="form.slug" placeholder="volley-club-aurora" /></label>
        <label>Città<input v-model="form.city" placeholder="Bari" /></label>
        <label>Stagione<input v-model="form.seasonLabel" placeholder="Stagione 2026/27" /></label>
        <label class="wide">Claim<input v-model="form.tagline" placeholder="Una società, quattro squadre, una sola passione." /></label>
        <label>Logo URL<input v-model="form.logoUrl" placeholder="https://…" /></label>
        <label>Hero URL<input v-model="form.heroImageUrl" placeholder="https://…" /></label>
        <label>Colore primario<input v-model="form.primaryColor" type="color" /></label>
        <label>Colore accento<input v-model="form.secondaryColor" type="color" /></label>
        <label class="wide">Categorie, una per riga<textarea v-model="form.categoriesText" rows="5" /></label>
      </div>
      <button class="primary" :disabled="submitting" @click="create">{{ submitting ? 'Creazione…' : 'Crea Club Hub' }}</button>

      <div v-if="created" class="credentials">
        <h3>Club Hub creato</h3>
        <p>Le credenziali sono visibili soltanto ora.</p>
        <div v-for="row in [
          ['Sito pubblico', abs(created.publicPath)], ['CMS Club', abs(created.adminPath)],
          ['Username', created.adminUsername], ['Password', created.adminPassword]
        ]" :key="row[0]" class="credential-row">
          <span>{{ row[0] }}</span><code>{{ row[1] }}</code><button @click="copy(row[1])">Copia</button>
        </div>
      </div>
      <p v-if="error" class="error">{{ error }}</p>
    </section>

    <section class="panel">
      <div class="heading"><div><p class="eyebrow">Portafoglio</p><h2>Club attivi</h2></div><button class="ghost" @click="load">Aggiorna</button></div>
      <p v-if="loading" class="hint">Caricamento…</p>
      <p v-else-if="!clubs.length" class="hint">Nessun Club Hub creato.</p>
      <div v-else class="club-list">
        <article v-for="club in clubs" :key="club.id" class="club-row">
          <div><strong>{{ club.name }}</strong><small>{{ club.city || 'Città non indicata' }} · {{ club.slug }}</small></div>
          <div class="metrics"><span>{{ club.teamsCount }} squadre</span><span>{{ club.postsCount }} contenuti</span></div>
          <div class="links"><a :href="`/club/${club.slug}`" target="_blank">Sito</a><a :href="`/club-admin/${club.slug}`" target="_blank">CMS</a></div>
          <div class="actions"><button @click="passwordOpen = passwordOpen === club.id ? null : club.id">Password</button><button class="danger" :disabled="deleting === club.id" @click="remove(club)">Elimina</button></div>
          <div v-if="passwordOpen === club.id" class="password-row"><input v-model="passwordInput" placeholder="Vuoto = genera password" /><button class="primary" @click="resetPassword(club)">Imposta</button></div>
          <div v-if="passwordResult?.id === club.id" class="password-row result"><code>{{ passwordResult.adminUsername }}</code><code>{{ passwordResult.adminPassword }}</code><button @click="copy(passwordResult.adminPassword)">Copia password</button></div>
        </article>
      </div>
    </section>
  </div>
</template>

<style scoped>
.club-master{display:flex;flex-direction:column;gap:18px}.panel{background:rgba(15,23,42,.78);border:1px solid rgba(148,163,184,.18);border-radius:16px;padding:20px;color:#e2e8f0}.heading{display:flex;align-items:center;justify-content:space-between;gap:16px}.heading h2{margin:2px 0 0;font-size:20px}.eyebrow{margin:0;color:#f59e0b;font-size:11px;font-weight:900;letter-spacing:.14em}.pill{border:1px solid rgba(245,158,11,.4);border-radius:999px;padding:6px 10px;color:#fbbf24;font-size:10px;font-weight:900}.hint{color:#94a3b8;font-size:13px}.form-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px;margin:18px 0}.form-grid label{display:flex;flex-direction:column;gap:6px;color:#94a3b8;font-size:12px}.form-grid .wide{grid-column:1/-1}.form-grid input,.form-grid textarea,.password-row input{background:#07101f;border:1px solid #293548;border-radius:9px;padding:10px;color:#f8fafc;font:inherit}.primary{border:0;border-radius:9px;padding:10px 16px;background:#f59e0b;color:#111827;font-weight:900;cursor:pointer}.ghost,.actions button,.credential-row button{border:1px solid #334155;border-radius:8px;padding:7px 11px;background:transparent;color:#cbd5e1;cursor:pointer}.credentials{margin-top:18px;border:1px solid rgba(245,158,11,.4);border-radius:12px;padding:15px}.credentials h3{color:#fbbf24;margin:0}.credential-row{display:flex;align-items:center;gap:10px;margin-top:8px}.credential-row span{width:100px;color:#94a3b8;font-size:12px}.credential-row code{flex:1;overflow:hidden;text-overflow:ellipsis}.error{color:#f87171}.club-list{display:grid;gap:10px;margin-top:14px}.club-row{display:grid;grid-template-columns:2fr 1fr 1fr auto;align-items:center;gap:14px;border:1px solid rgba(148,163,184,.14);border-radius:12px;padding:14px}.club-row small{display:block;color:#64748b;margin-top:4px}.metrics{display:flex;flex-direction:column;color:#94a3b8;font-size:12px}.links{display:flex;gap:10px}.links a{color:#fbbf24}.actions{white-space:nowrap}.actions .danger{color:#f87171;border-color:rgba(248,113,113,.4);margin-left:6px}.password-row{grid-column:1/-1;display:flex;gap:8px;padding-top:10px}.password-row input{flex:1}.password-row.result{color:#fbbf24}@media(max-width:800px){.form-grid{grid-template-columns:1fr}.form-grid .wide{grid-column:auto}.club-row{grid-template-columns:1fr}.password-row{grid-column:auto}.credential-row{align-items:flex-start;flex-wrap:wrap}.credential-row code{flex-basis:100%}}
</style>
