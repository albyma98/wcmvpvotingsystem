<script setup>
// Pannello admin del singolo torneo (/ta/:slug).
// Mondo parallelo alle società: credenziali dedicate, cookie separato,
// scoping hard sull'evento lato backend.
// Due anime: SETUP (pre-torneo, con calma) e LIVE (console scoring, mobile,
// pulsanti grandi, undo sempre visibile).
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'

const props = defineProps({ slug: { type: String, required: true } })

const API = path => `/api/v1/ta/${props.slug}${path}`
const j = (method, path, body) =>
  fetch(API(path), {
    method,
    credentials: 'include',
    headers: body ? { 'Content-Type': 'application/json' } : undefined,
    body: body ? JSON.stringify(body) : undefined
  })

// ---------- auth ----------
const authed = ref(false)
const authChecked = ref(false)
const login = reactive({ username: '', password: '' })
const loginError = ref('')

async function checkAuth () {
  const res = await j('GET', '/overview')
  authed.value = res.ok
  authChecked.value = true
  if (res.ok) await bootstrap(await res.json())
}

async function doLogin () {
  loginError.value = ''
  const res = await j('POST', '/login', { username: login.username.trim(), password: login.password })
  if (!res.ok) { loginError.value = 'Credenziali non valide.'; return }
  login.password = ''
  await checkAuth()
}

async function doLogout () {
  await j('POST', '/logout')
  authed.value = false
}

// ---------- stato ----------
const tab = ref('live')
const tabs = [
  { id: 'live', label: 'Live' },
  { id: 'matches', label: 'Calendario' },
  { id: 'teams', label: 'Squadre' },
  { id: 'operators', label: 'Operatori' },
  { id: 'sponsors', label: 'Sponsor' },
  { id: 'settings', label: 'Impostazioni' }
]

const overview = ref(null)
const teams = ref([])
const matches = ref([])
const sponsors = ref([])
const settings = reactive({ name: '', format: '', dateLabel: '', location: '', statusLabel: '', phaseLabel: '' })
const busy = ref('')
const notice = ref('')

function flash (msg) { notice.value = msg; setTimeout(() => { if (notice.value === msg) notice.value = '' }, 2500) }

async function bootstrap (ov) {
  overview.value = ov
  Object.assign(settings, ov.settings)
  await Promise.all([loadTeams(), loadMatches(), loadSponsors(), loadOperators()])
}
async function loadTeams () { const r = await j('GET', '/teams'); if (r.ok) teams.value = (await r.json()).teams }
async function loadMatches () { const r = await j('GET', '/matches'); if (r.ok) matches.value = (await r.json()).matches }
async function loadSponsors () { const r = await j('GET', '/sponsors'); if (r.ok) sponsors.value = (await r.json()).sponsors }

// ---------- squadre ----------
const newTeam = reactive({ name: '', shortName: '', city: '', groupName: '' })
async function addTeam () {
  const name = newTeam.name.trim()
  if (!name) { flash('Inserisci il nome della squadra.'); return }
  busy.value = 'teams'
  const r = await j('POST', '/teams', { teams: [{
    name,
    shortName: newTeam.shortName.trim(),
    city: newTeam.city.trim(),
    groupName: newTeam.groupName.trim()
  }] })
  busy.value = ''
  if (r.ok) {
    newTeam.name = ''; newTeam.shortName = ''; newTeam.city = ''; newTeam.groupName = ''
    await loadTeams()
    flash('Squadra aggiunta.')
  } else {
    flash('Errore nel salvataggio della squadra.')
  }
}
async function deleteTeam (id) {
  const r = await j('DELETE', `/teams/${id}`)
  if (r.status === 409) { flash('Squadra usata in una partita: elimina prima la partita.'); return }
  if (r.ok) { await loadTeams(); flash('Squadra eliminata.') }
}

// ---------- calendario ----------
const newMatch = reactive({ court: 'CAMPO 1', time: '', stage: '', teamAId: 0, teamBId: 0 })
async function createMatch () {
  if (!newMatch.teamAId || !newMatch.teamBId || newMatch.teamAId === newMatch.teamBId) { flash('Scegli due squadre diverse.'); return }
  const scheduledAt = new Date().toISOString().slice(0, 10) + 'T' + (newMatch.time || '00:00')
  const r = await j('POST', '/matches', { ...newMatch, scheduledAt })
  if (r.ok) { newMatch.time = ''; newMatch.stage = ''; newMatch.teamAId = 0; newMatch.teamBId = 0; await loadMatches(); flash('Partita creata.') }
}
async function deleteMatch (id) {
  if (!confirm('Eliminare la partita?')) return
  const r = await j('DELETE', `/matches/${id}`)
  if (r.ok) { await loadMatches(); flash('Partita eliminata.') }
}

// ---------- console scoring ----------
const liveMatches = computed(() => matches.value.filter(m => m.status === 'live'))
const scheduledMatches = computed(() => matches.value.filter(m => m.status === 'scheduled'))
const finishedMatches = computed(() => matches.value.filter(m => m.status === 'finished'))

async function score (matchId, action) {
  const r = await j('POST', `/matches/${matchId}/score`, { action })
  if (r.ok) {
    matches.value = (await r.json()).matches // stato fresco dal server, sempre
  } else {
    const err = (await r.json().catch(() => ({}))).error
    if (err === 'set_tied') flash('Il set non può chiudersi in parità.')
  }
}

// Refresh live leggero: la console può restare aperta ore a bordo campo.
let timer = null
onMounted(() => {
  checkAuth()
  timer = setInterval(() => { if (authed.value && tab.value === 'live' && !document.hidden) loadMatches() }, 15000)
})
onUnmounted(() => clearInterval(timer))

// ---------- sponsor ----------
const newSponsor = reactive({ name: '', tier: 'partner', url: '', brandColor: '', logoUrl: '' })
const logoBusy = ref(false)

// Ridimensiona l'immagine lato client (lato lungo max `maxSize`) e la converte
// in data-URL: il logo viaggia inline nel JSON e finisce così in logo_url, senza
// dipendere da uno storage su disco. WebP quando supportato, PNG di fallback:
// entrambi preservano la trasparenza tipica dei loghi.
function fileToLogoDataURL (file, maxSize = 400) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    const img = new Image()
    reader.onload = () => { img.src = reader.result }
    reader.onerror = () => reject(new Error('read'))
    img.onerror = () => reject(new Error('decode'))
    img.onload = () => {
      const scale = Math.min(1, maxSize / Math.max(img.width, img.height))
      const w = Math.max(1, Math.round(img.width * scale))
      const h = Math.max(1, Math.round(img.height * scale))
      const canvas = document.createElement('canvas')
      canvas.width = w; canvas.height = h
      canvas.getContext('2d').drawImage(img, 0, 0, w, h)
      let out = canvas.toDataURL('image/webp', 0.85)
      if (!out.startsWith('data:image/webp')) out = canvas.toDataURL('image/png')
      resolve(out)
    }
    reader.readAsDataURL(file)
  })
}

async function onLogoPick (e) {
  const file = e.target.files?.[0]
  if (!file) return
  if (!file.type.startsWith('image/')) { flash('Seleziona un file immagine.'); e.target.value = ''; return }
  logoBusy.value = true
  try {
    newSponsor.logoUrl = await fileToLogoDataURL(file, 400)
  } catch { flash('Immagine non valida.') }
  finally { logoBusy.value = false; e.target.value = '' }
}

async function createSponsor () {
  if (!newSponsor.name.trim()) return
  const r = await j('POST', '/sponsors', { ...newSponsor })
  if (r.ok) {
    newSponsor.name = ''; newSponsor.url = ''; newSponsor.brandColor = ''; newSponsor.logoUrl = ''
    await loadSponsors(); flash('Sponsor aggiunto.')
  } else {
    const err = (await r.json().catch(() => ({}))).error
    flash(err === 'logo_too_large' ? 'Logo troppo pesante.' : err && err.startsWith('logo') ? 'Logo non valido.' : 'Aggiunta non riuscita.')
  }
}
async function deleteSponsor (id) {
  const r = await j('DELETE', `/sponsors/${id}`)
  if (r.ok) { await loadSponsors(); flash('Sponsor rimosso.') }
}

// ---------- operatori campo ----------
const operators = ref([])
const newOperator = reactive({ court: 'CAMPO 1', label: '' })
const createdOperator = ref(null)
async function loadOperators () { const r = await j('GET', '/operators'); if (r.ok) operators.value = (await r.json()).operators }
async function createOperator () {
  if (!newOperator.court.trim()) return
  const r = await j('POST', '/operators', { ...newOperator })
  if (r.ok) {
    createdOperator.value = await r.json()
    newOperator.label = ''
    await loadOperators()
  }
}
async function deleteOperator (id) {
  if (!confirm('Revocare il link? L\'operatore perde subito l\'accesso.')) return
  const r = await j('DELETE', `/operators/${id}`)
  if (r.ok) { await loadOperators(); flash('Operatore revocato.') }
}
const opLink = t => `${window.location.origin}/op/${t}`
function copy (text) { navigator.clipboard?.writeText(text) }

// ---------- impostazioni ----------
async function saveSettings () {
  busy.value = 'settings'
  const r = await j('PUT', '/settings', { ...settings })
  busy.value = ''
  if (r.ok) flash('Impostazioni salvate — visibili ai tifosi al prossimo refresh.')
}
</script>

<template>
  <div class="ta-page">
    <!-- LOGIN -->
    <div v-if="authChecked && !authed" class="ta-login">
      <h1>Admin Torneo</h1>
      <p class="slug">/{{ slug }}</p>
      <input v-model="login.username" placeholder="Username" autocomplete="username" />
      <input v-model="login.password" type="password" placeholder="Password" autocomplete="current-password" @keyup.enter="doLogin" />
      <button @click="doLogin">Entra</button>
      <p v-if="loginError" class="err">{{ loginError }}</p>
    </div>

    <!-- PANNELLO -->
    <template v-else-if="authed">
      <header class="ta-head">
        <div>
          <h1>{{ settings.name || slug }}</h1>
          <p>{{ settings.phaseLabel }} · {{ teams.length }} squadre · {{ matches.length }} partite</p>
        </div>
        <button class="ghost" @click="doLogout">Esci</button>
      </header>

      <nav class="ta-tabs">
        <button v-for="t in tabs" :key="t.id" :class="{ active: tab === t.id }" @click="tab = t.id">{{ t.label }}</button>
      </nav>

      <p v-if="notice" class="ta-notice">{{ notice }}</p>

      <!-- LIVE: console scoring -->
      <section v-if="tab === 'live'" class="ta-body">
        <article v-for="m in liveMatches" :key="m.id" class="score-card">
          <div class="sc-head"><span class="dot"></span>{{ m.court }} · {{ m.setLabel }}</div>
          <div class="sc-grid">
            <div class="sc-team">
              <div class="name">{{ m.teamAName }}</div>
              <div class="cur">{{ m.curA }}</div>
              <button class="big" @click="score(m.id, 'point_a')">+1</button>
              <button class="undo" @click="score(m.id, 'undo_a')">annulla</button>
            </div>
            <div class="sc-mid">
              <div class="sets">{{ m.scoreA }} : {{ m.scoreB }}</div>
              <div class="setlist">{{ m.sets.join(' | ') || '—' }}</div>
              <button class="close-set" @click="score(m.id, 'close_set')">Chiudi set</button>
              <button class="finish" @click="score(m.id, 'finish')">Fine partita</button>
            </div>
            <div class="sc-team">
              <div class="name">{{ m.teamBName }}</div>
              <div class="cur">{{ m.curB }}</div>
              <button class="big" @click="score(m.id, 'point_b')">+1</button>
              <button class="undo" @click="score(m.id, 'undo_b')">annulla</button>
            </div>
          </div>
        </article>
        <p v-if="!liveMatches.length" class="hint">Nessuna partita live. Avviane una qui sotto.</p>

        <h3 v-if="scheduledMatches.length">In programma</h3>
        <div v-for="m in scheduledMatches" :key="m.id" class="row-match">
          <span>{{ m.court }} · {{ m.time || '—' }} · <b>{{ m.teamAName }}</b> vs <b>{{ m.teamBName }}</b></span>
          <button class="start" @click="score(m.id, 'start')">▶ Avvia</button>
        </div>

        <h3 v-if="finishedMatches.length">Concluse</h3>
        <div v-for="m in finishedMatches" :key="m.id" class="row-match done">
          <span>{{ m.teamAName }} <b>{{ m.scoreA }}:{{ m.scoreB }}</b> {{ m.teamBName }} <small>({{ m.sets.join(', ') }})</small></span>
          <button class="ghost" @click="score(m.id, 'reopen')">Riapri</button>
        </div>
      </section>

      <!-- CALENDARIO -->
      <section v-else-if="tab === 'matches'" class="ta-body">
        <div class="form-row">
          <input v-model="newMatch.court" placeholder="CAMPO 1" style="max-width:110px" />
          <input v-model="newMatch.time" placeholder="18:30" style="max-width:90px" />
          <select v-model.number="newMatch.teamAId"><option :value="0">Squadra A</option><option v-for="t in teams" :key="t.id" :value="t.id">{{ t.name }}</option></select>
          <select v-model.number="newMatch.teamBId"><option :value="0">Squadra B</option><option v-for="t in teams" :key="t.id" :value="t.id">{{ t.name }}</option></select>
          <select v-model="newMatch.stage"><option value="">Girone</option><option>QUARTI</option><option>SEMIFINALE</option><option>FINALE 3° POSTO</option><option>FINALE</option></select>
          <button @click="createMatch">Aggiungi</button>
        </div>
        <div v-for="m in matches" :key="m.id" class="row-match">
          <span>{{ m.court }} · {{ m.time || '—' }} · {{ m.teamAName }} vs {{ m.teamBName }} <small>[{{ m.status }}]</small></span>
          <button class="danger" @click="deleteMatch(m.id)">Elimina</button>
        </div>
        <p v-if="!matches.length" class="hint">Nessuna partita. Prima importa le squadre, poi crea il calendario.</p>
      </section>

      <!-- SQUADRE -->
      <section v-else-if="tab === 'teams'" class="ta-body">
        <p class="hint">Aggiungi una squadra alla volta. Solo il nome è obbligatorio; sigla, città e girone sono facoltativi.</p>
        <div class="team-form">
          <input v-model="newTeam.name" placeholder="Nome squadra *" @keyup.enter="addTeam" />
          <div class="form-row">
            <input v-model="newTeam.shortName" placeholder="Sigla (es. MAMBO)" maxlength="6" style="max-width:150px" @keyup.enter="addTeam" />
            <input v-model="newTeam.city" placeholder="Città" @keyup.enter="addTeam" />
            <input v-model="newTeam.groupName" placeholder="Girone" maxlength="4" style="max-width:100px" @keyup.enter="addTeam" />
          </div>
          <button :disabled="busy === 'teams' || !newTeam.name.trim()" @click="addTeam">
            {{ busy === 'teams' ? 'Salvo…' : 'Aggiungi squadra' }}
          </button>
        </div>

        <div class="team-list">
          <p v-if="!teams.length" class="hint">Nessuna squadra ancora. Aggiungi la prima qui sopra.</p>
          <div v-for="t in teams" :key="t.id" class="row-match">
            <span>
              <b>{{ t.name }}</b>
              <small v-if="t.shortName">({{ t.shortName }})</small>
              <small v-if="t.city">· {{ t.city }}</small>
              <small v-if="t.groupName">· Girone {{ t.groupName }}</small>
            </span>
            <button class="danger" @click="deleteTeam(t.id)">Elimina</button>
          </div>
        </div>
      </section>

      <!-- OPERATORI CAMPO -->
      <section v-else-if="tab === 'operators'" class="ta-body">
        <p class="hint">Un link per campo: mandalo su WhatsApp al volontario insieme al PIN. Revocabile in ogni momento.</p>
        <div class="form-row">
          <input v-model="newOperator.court" placeholder="CAMPO 1" style="max-width:130px" />
          <input v-model="newOperator.label" placeholder="Nome operatore (opzionale)" />
          <button @click="createOperator">Genera link</button>
        </div>
        <div v-if="createdOperator" class="op-created">
          <p><b>{{ createdOperator.operator.court }}</b> — consegna questi due dati:</p>
          <div class="cred"><code>{{ opLink(createdOperator.operator.token) }}</code><button @click="copy(opLink(createdOperator.operator.token))">Copia link</button></div>
          <div class="cred"><code>PIN {{ createdOperator.operator.pin }}</code><button @click="copy(createdOperator.operator.pin)">Copia PIN</button></div>
        </div>
        <div v-for="o in operators" :key="o.id" class="row-match">
          <span><b>{{ o.court }}</b> <small v-if="o.label">· {{ o.label }}</small> <small>· PIN {{ o.pin }}</small></span>
          <span style="display:flex;gap:6px">
            <button class="ghost" @click="copy(opLink(o.token))">Copia link</button>
            <button class="danger" @click="deleteOperator(o.id)">Revoca</button>
          </span>
        </div>
      </section>

      <!-- SPONSOR -->
      <section v-else-if="tab === 'sponsors'" class="ta-body">
        <div class="form-row">
          <input v-model="newSponsor.name" placeholder="Nome sponsor" />
          <select v-model="newSponsor.tier"><option value="main">Main</option><option value="partner">Partner</option></select>
          <input v-model="newSponsor.brandColor" placeholder="#FF6B1A" style="max-width:110px" />
          <input v-model="newSponsor.url" placeholder="Sito: https://…" />
          <button @click="createSponsor">Aggiungi</button>
        </div>
        <div class="form-row logo-row">
          <label class="logo-pick">
            {{ logoBusy ? 'Carico…' : 'Logo (immagine)' }}
            <input type="file" accept="image/*" @change="onLogoPick" :disabled="logoBusy" hidden />
          </label>
          <div v-if="newSponsor.logoUrl" class="logo-preview">
            <img :src="newSponsor.logoUrl" alt="anteprima logo" />
            <button class="danger" @click="newSponsor.logoUrl = ''">Togli logo</button>
          </div>
          <span v-else class="hint">Facoltativo — PNG/JPG/WebP, ridimensionato in automatico.</span>
        </div>
        <div v-for="s in sponsors" :key="s.id" class="row-match">
          <span class="sponsor-line">
            <img v-if="s.logoUrl" :src="s.logoUrl" :alt="s.name" class="logo-thumb" />
            <b>{{ s.name }}</b> <small>[{{ s.tier }}]</small>
            <span v-if="s.brandColor" class="swatch" :style="{ background: s.brandColor }"></span>
          </span>
          <button class="danger" @click="deleteSponsor(s.id)">Rimuovi</button>
        </div>
        <p class="hint">I «main» vanno nella riga grande fissa, i «partner» nel marquee che scorre.</p>
      </section>

      <!-- IMPOSTAZIONI -->
      <section v-else class="ta-body">
        <div class="settings-grid">
          <label>Nome <input v-model="settings.name" /></label>
          <label>Formato <input v-model="settings.format" /></label>
          <label>Date <input v-model="settings.dateLabel" /></label>
          <label>Luogo <input v-model="settings.location" /></label>
          <label>Stato (pill) <input v-model="settings.statusLabel" placeholder="TORNEO IN CORSO" /></label>
          <label>Fase (pill) <input v-model="settings.phaseLabel" placeholder="FASE A GIRONI" /></label>
        </div>
        <button :disabled="busy === 'settings'" @click="saveSettings">{{ busy === 'settings' ? 'Salvo…' : 'Salva' }}</button>
        <p class="hint">Stato e fase sono le due pill nell'hero dei tifosi: cambiale quando avanzi il torneo (es. «FASE FINALE»).</p>
      </section>
    </template>

    <p v-else class="ta-loading">Caricamento…</p>
  </div>
</template>

<style scoped>
.ta-page { min-height: 100dvh; background: #0a0a0e; color: #f1f5f9; padding: 16px; max-width: 760px; margin: 0 auto; font-size: 14.5px; }
.ta-login { max-width: 340px; margin: 12vh auto 0; display: flex; flex-direction: column; gap: 10px; text-align: center; }
.ta-login h1 { font-size: 22px; margin: 0; }
.ta-login .slug { color: #f2b928; margin: 0 0 8px; font-weight: 700; }
.ta-login input { background: #15151b; border: 1px solid rgba(255,255,255,.14); border-radius: 10px; padding: 12px; color: #fff; font-size: 15px; }
.ta-login button { background: #f2b928; color: #111; border: none; border-radius: 10px; padding: 12px; font-weight: 800; font-size: 15px; cursor: pointer; }
.err { color: #f87171; }
.ta-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-bottom: 10px; }
.ta-head h1 { font-size: 19px; margin: 0; }
.ta-head p { margin: 2px 0 0; color: #94a3b8; font-size: 12.5px; }
.ta-tabs { display: flex; gap: 6px; overflow-x: auto; padding-bottom: 4px; margin-bottom: 10px; scrollbar-width: none; }
.ta-tabs button { flex: none; background: #15151b; border: 1px solid rgba(255,255,255,.12); color: #cbd5e1; border-radius: 999px; padding: 8px 16px; font-weight: 700; cursor: pointer; }
.ta-tabs button.active { background: #f2b928; color: #111; border-color: #f2b928; }
.ta-notice { background: rgba(242,185,40,.12); border: 1px solid rgba(242,185,40,.35); color: #fbd34d; border-radius: 8px; padding: 8px 12px; font-size: 13px; }
.ta-body { display: flex; flex-direction: column; gap: 10px; }
.ta-body h3 { margin: 12px 0 2px; font-size: 13px; color: #94a3b8; text-transform: uppercase; letter-spacing: 1px; }
.hint { color: #94a3b8; font-size: 13px; }
.hint code { background: #15151b; padding: 2px 6px; border-radius: 5px; }
button { cursor: pointer; }
.form-row { display: flex; flex-wrap: wrap; gap: 8px; }
.form-row input, .form-row select, textarea { background: #15151b; border: 1px solid rgba(255,255,255,.14); border-radius: 8px; padding: 9px 11px; color: #fff; flex: 1; min-width: 120px; font-size: 14px; }
/* Materialize (CSS globale da index.html) nasconde i <select> nativi con
   `select{display:none}` (+ background chiaro e height:3rem) e lascia le <option>
   senza contrasto. Qui li ripristiniamo e li rendiamo leggibili (scoped → solo
   il pannello torneo, non tocca il resto dell'app club). */
select {
  display: block; height: auto;
  background: #15151b; color: #fff;
  border: 1px solid rgba(255,255,255,.14); border-radius: 8px;
  padding: 9px 11px; font-size: 14px;
}
option { background: #15151b; color: #fff; }
textarea { width: 100%; font-family: inherit; }
.form-row button, .ta-body > button { background: #f2b928; color: #111; border: none; border-radius: 8px; padding: 9px 16px; font-weight: 800; align-self: flex-start; }
.team-form { display: flex; flex-direction: column; gap: 8px; }
.team-form > input { background: #15151b; border: 1px solid rgba(255,255,255,.14); border-radius: 8px; padding: 9px 11px; color: #fff; font-size: 14px; width: 100%; }
.team-form button { background: #f2b928; color: #111; border: none; border-radius: 8px; padding: 10px 16px; font-weight: 800; align-self: flex-start; }
.team-form button:disabled { opacity: .5; cursor: default; }
.team-list { margin-top: 14px; display: flex; flex-direction: column; gap: 8px; }
.row-match { display: flex; align-items: center; justify-content: space-between; gap: 10px; background: #15151b; border: 1px solid rgba(255,255,255,.09); border-radius: 10px; padding: 10px 12px; }
.row-match.done { opacity: .75; }
.row-match small { color: #94a3b8; }
.danger { background: transparent; border: 1px solid rgba(248,113,113,.4); color: #f87171; border-radius: 7px; padding: 5px 12px; }
.ghost { background: transparent; border: 1px solid rgba(255,255,255,.2); color: #cbd5e1; border-radius: 7px; padding: 6px 12px; }
.start { background: #16a34a; color: #fff; border: none; border-radius: 8px; padding: 8px 16px; font-weight: 800; }
.swatch { display: inline-block; width: 13px; height: 13px; border-radius: 4px; vertical-align: -2px; margin-left: 4px; }
.logo-row { align-items: center; }
.logo-pick { background: #23232c; border: 1px dashed rgba(255,255,255,.25); border-radius: 8px; padding: 9px 14px; color: #e2e8f0; font-size: 13px; font-weight: 700; cursor: pointer; }
.logo-pick:hover { border-color: #f2b928; }
.logo-preview { display: flex; align-items: center; gap: 10px; }
.logo-preview img { height: 40px; width: auto; max-width: 120px; object-fit: contain; background: #fff; border-radius: 6px; padding: 3px; }
.logo-preview .danger { font-size: 12px; padding: 6px 10px; }
.sponsor-line { display: flex; align-items: center; gap: 8px; min-width: 0; }
.logo-thumb { height: 26px; width: auto; max-width: 70px; object-fit: contain; background: #fff; border-radius: 5px; padding: 2px; flex: none; }
/* --- console scoring: pulsanti da pollice, sole in faccia --- */
.score-card { background: linear-gradient(180deg, rgba(139,32,38,.4), #15151b 60%); border: 1px solid rgba(255,255,255,.1); border-radius: 14px; padding: 12px; }
.sc-head { display: flex; align-items: center; gap: 8px; font-size: 12px; font-weight: 800; letter-spacing: 1px; color: #fca5a5; margin-bottom: 8px; }
.sc-head .dot { width: 8px; height: 8px; border-radius: 50%; background: #e5484d; }
.sc-grid { display: grid; grid-template-columns: 1fr auto 1fr; gap: 10px; align-items: start; }
.sc-team { display: flex; flex-direction: column; align-items: center; gap: 8px; }
.sc-team .name { font-weight: 900; text-transform: uppercase; font-size: 13px; text-align: center; }
.sc-team .cur { font-size: 42px; font-weight: 900; line-height: 1; font-variant-numeric: tabular-nums; }
.sc-team .big { width: 100%; max-width: 130px; font-size: 26px; font-weight: 900; padding: 16px 0; border: none; border-radius: 14px; background: #f2b928; color: #111; }
.sc-team .undo { background: transparent; border: 1px solid rgba(255,255,255,.25); color: #cbd5e1; border-radius: 8px; padding: 7px 14px; font-size: 13px; }
.sc-mid { display: flex; flex-direction: column; align-items: center; gap: 6px; padding-top: 4px; }
.sc-mid .sets { font-size: 22px; font-weight: 900; color: #f2b928; }
.sc-mid .setlist { font-size: 12px; color: #94a3b8; }
.close-set { background: #1d4ed8; color: #fff; border: none; border-radius: 8px; padding: 9px 14px; font-weight: 800; font-size: 13px; }
.finish { background: transparent; border: 1px solid rgba(248,113,113,.5); color: #f87171; border-radius: 8px; padding: 7px 14px; font-size: 12px; }
.ta-loading { text-align: center; color: #94a3b8; padding-top: 20vh; }
</style>
