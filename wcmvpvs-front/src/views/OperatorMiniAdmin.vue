<script setup>
import { computed, reactive, ref, watch } from 'vue'

const props = defineProps({
  token: { type: String, required: true },
  role: { type: String, required: true },
  data: { type: Object, default: () => ({}) }
})
const emit = defineEmits(['reload'])
const API = path => `/api/v1/op/${props.token}${path}`
const request = (method, path, body) => fetch(API(path), {
  method,
  credentials: 'include',
  headers: body ? { 'Content-Type': 'application/json' } : undefined,
  body: body ? JSON.stringify(body) : undefined
})

const busy = ref(false)
const notice = ref('')
const teams = computed(() => props.data?.teams ?? [])
const matches = computed(() => props.data?.matches ?? [])
const courts = computed(() => props.data?.courts?.length ? props.data.courts : ['CAMPO 1'])
const mvp = computed(() => props.data?.mvp ?? { teams: [], totalVotes: 0 })
const blankPlayers = () => Array.from({ length: 8 }, () => ({ firstName: '', lastName: '', gender: 'male' }))
const newTeam = reactive({ name: '', shortName: '', city: '', groupName: '', logoUrl: '', players: blankPlayers() })
const rosterTeamId = ref(0)
const rosterDraft = ref(blankPlayers())

function flash (message) {
  notice.value = message
  window.setTimeout(() => { if (notice.value === message) notice.value = '' }, 2600)
}
async function mutate (method, path, body, success) {
  busy.value = true
  const response = await request(method, path, body)
  busy.value = false
  if (!response.ok) {
    const error = (await response.json().catch(() => ({}))).error
    flash(error === 'team_in_use' ? 'La squadra è già usata in una partita.' : 'Operazione non riuscita.')
    return false
  }
  flash(success)
  emit('reload')
  return true
}
async function addTeam () {
  if (!newTeam.name.trim()) return
  if (await mutate('POST', '/teams', newTeam, 'Squadra e rosa aggiunte.')) {
    Object.assign(newTeam, { name: '', shortName: '', city: '', groupName: '', logoUrl: '', players: blankPlayers() })
  }
}
function selectTeamLogo (event) {
  const file = event.target.files?.[0]
  if (!file) return
  if (!file.type.startsWith('image/')) {
    flash('Seleziona un file immagine.')
    return
  }
  const reader = new FileReader()
  reader.onload = () => { newTeam.logoUrl = String(reader.result || '') }
  reader.readAsDataURL(file)
}
async function deleteTeam (team) {
  if (!window.confirm(`Eliminare ${team.name}?`)) return
  await mutate('DELETE', `/teams/${team.id}`, null, 'Squadra eliminata.')
}
function editRoster (team) {
  rosterTeamId.value = team.id
  const rows = (team.players || []).map(player => ({
    firstName: player.firstName || '', lastName: player.lastName || '',
    gender: player.gender === 'female' ? 'female' : 'male'
  }))
  while (rows.length < 8) rows.push({ firstName: '', lastName: '', gender: 'male' })
  rosterDraft.value = rows.slice(0, 8)
}
async function saveRoster () {
  if (await mutate('PUT', `/teams/${rosterTeamId.value}/players`, { players: rosterDraft.value }, 'Rosa aggiornata.')) {
    rosterTeamId.value = 0
  }
}

const blankMatch = () => ({ court: courts.value[0] || 'CAMPO 1', time: '', stage: '', teamAId: 0, teamBId: 0 })
const matchForm = reactive(blankMatch())
const editingMatchId = ref('')
const suggestedMatchTime = ref('')
watch(courts, values => {
  if (!values.includes(matchForm.court)) matchForm.court = values[0]
}, { immediate: true })

function matchTimeMinutes (value) {
  const match = /^(\d{1,2}):([0-5]\d)$/.exec(String(value || '').trim())
  if (!match || Number(match[1]) > 23) return null
  return Number(match[1]) * 60 + Number(match[2])
}
function prefillNextGroupMatchTime () {
  // Se l'operatore modifica manualmente l'orario, gli aggiornamenti live
  // non devono sovrascriverlo.
  if (matchForm.time && matchForm.time !== suggestedMatchTime.value) return

  const groupMatches = matches.value.filter(match =>
    !String(match.stage || '').trim() && matchTimeMinutes(match.time) !== null
  )
  if (!groupMatches.length) {
    if (matchForm.time === suggestedMatchTime.value) matchForm.time = ''
    suggestedMatchTime.value = ''
    return
  }

  // Gestisce correttamente anche tornei che proseguono dopo mezzanotte:
  // l'eventuale partita àncora indica da dove inizia la giornata sportiva.
  const anchor = groupMatches.find(match => match.isAnchor)
  const anchorMinutes = anchor ? matchTimeMinutes(anchor.time) : null
  let latest = -1
  for (const match of groupMatches) {
    const minutes = matchTimeMinutes(match.time)
    const orderedMinutes = anchorMinutes !== null && minutes < anchorMinutes
      ? minutes + 24 * 60
      : minutes
    if (orderedMinutes > latest) latest = orderedMinutes
  }

  const next = (latest + 30) % 1440
  const suggestion = `${String(Math.floor(next / 60)).padStart(2, '0')}:${String(next % 60).padStart(2, '0')}`
  suggestedMatchTime.value = suggestion
  matchForm.time = suggestion
}
watch(matches, () => {
  if (!editingMatchId.value) prefillNextGroupMatchTime()
}, { immediate: true })
function editMatch (match) {
  editingMatchId.value = match.id
  Object.assign(matchForm, {
    court: match.court, time: match.time, stage: match.stage || '',
    teamAId: match.teamAId, teamBId: match.teamBId
  })
}
function resetMatch () {
  editingMatchId.value = ''
  Object.assign(matchForm, blankMatch())
}
async function saveMatch () {
  if (!matchForm.teamAId || !matchForm.teamBId || matchForm.teamAId === matchForm.teamBId) {
    flash('Seleziona due squadre diverse.')
    return
  }
  const scheduledAt = new Date().toISOString().slice(0, 10) + 'T' + (matchForm.time || '00:00')
  const path = editingMatchId.value ? `/calendar/${editingMatchId.value}` : '/calendar'
  const method = editingMatchId.value ? 'PUT' : 'POST'
  if (await mutate(method, path, { ...matchForm, scheduledAt }, editingMatchId.value ? 'Partita aggiornata.' : 'Partita aggiunta.')) {
    // Rimane vuoto per pochi istanti: il reload aggiorna matches e il watcher
    // inserisce il nuovo suggerimento calcolato includendo la partita salvata.
    resetMatch()
  }
}
async function deleteMatch (match) {
  if (!window.confirm(`Eliminare ${match.teamAName} vs ${match.teamBName}?`)) return
  await mutate('DELETE', `/calendar/${match.id}`, null, 'Partita eliminata.')
}

const mvpTeams = computed(() => (mvp.value.teams || []).map(team => ({
  ...team,
  candidates: [...(team.candidates || [])].sort((a, b) => b.votes - a.votes),
  votes: (team.candidates || []).reduce((sum, item) => sum + item.votes, 0)
})).sort((a, b) => b.votes - a.votes))
</script>

<template>
  <p v-if="notice" class="mini-notice">{{ notice }}</p>

  <section v-if="role === 'teams'" class="mini-section">
    <h2>Squadre e rose</h2>
    <div class="mini-card form-card">
      <input v-model="newTeam.name" placeholder="Nome squadra *" />
      <div class="form-grid">
        <input v-model="newTeam.shortName" maxlength="6" placeholder="Sigla" />
        <input v-model="newTeam.city" placeholder="Città" />
        <input v-model="newTeam.groupName" maxlength="4" placeholder="Girone" />
      </div>
      <label class="logo-picker">
        <img v-if="newTeam.logoUrl" :src="newTeam.logoUrl" alt="" />
        <span>{{ newTeam.logoUrl ? 'Cambia icona squadra' : 'Aggiungi icona squadra' }}</span>
        <input type="file" accept="image/*" @change="selectTeamLogo" />
      </label>
      <details>
        <summary>Rosa giocatori</summary>
        <div v-for="(player, index) in newTeam.players" :key="index" class="player-row">
          <span>{{ index + 1 }}</span>
          <input v-model="player.firstName" placeholder="Nome" />
          <input v-model="player.lastName" placeholder="Cognome" />
          <select v-model="player.gender"><option value="male">M</option><option value="female">F</option></select>
        </div>
      </details>
      <button class="primary" :disabled="busy || !newTeam.name.trim()" @click="addTeam">Aggiungi squadra</button>
    </div>

    <article v-for="team in teams" :key="team.id" class="mini-card">
      <div class="row-head">
        <div><b>{{ team.name }}</b><small>{{ team.groupName ? `Girone ${team.groupName}` : 'Nessun girone' }} · {{ team.players?.length || 0 }} giocatori</small></div>
        <button class="danger" @click="deleteTeam(team)">Elimina</button>
      </div>
      <div class="chips"><span v-for="player in team.players" :key="player.id">{{ player.firstName }} {{ player.lastName }}</span></div>
      <button class="secondary" @click="editRoster(team)">Modifica rosa</button>
      <div v-if="rosterTeamId === team.id" class="roster-edit">
        <div v-for="(player, index) in rosterDraft" :key="index" class="player-row">
          <span>{{ index + 1 }}</span><input v-model="player.firstName" placeholder="Nome" />
          <input v-model="player.lastName" placeholder="Cognome" />
          <select v-model="player.gender"><option value="male">M</option><option value="female">F</option></select>
        </div>
        <div class="actions"><button class="primary" @click="saveRoster">Salva rosa</button><button class="secondary" @click="rosterTeamId = 0">Annulla</button></div>
      </div>
    </article>
  </section>

  <section v-else-if="role === 'calendar'" class="mini-section">
    <h2>Calendario</h2>
    <div class="mini-card form-card">
      <div class="form-grid">
        <select v-model="matchForm.court"><option v-for="court in courts" :key="court">{{ court }}</option></select>
        <input v-model="matchForm.time" placeholder="18:30" />
        <select v-model="matchForm.stage"><option value="">Girone</option><option>OTTAVI</option><option>QUARTI</option><option>SEMIFINALE</option><option>FINALE 3° POSTO</option><option>FINALE</option></select>
      </div>
      <small v-if="!editingMatchId && !matchForm.stage && suggestedMatchTime && matchForm.time === suggestedMatchTime" class="time-hint">
        Orario suggerito automaticamente: 30 minuti dopo l’ultima partita del girone.
      </small>
      <select v-model.number="matchForm.teamAId"><option :value="0">Squadra A</option><option v-for="team in teams" :key="team.id" :value="team.id">{{ team.name }}</option></select>
      <select v-model.number="matchForm.teamBId"><option :value="0">Squadra B</option><option v-for="team in teams" :key="team.id" :value="team.id">{{ team.name }}</option></select>
      <div class="actions">
        <button class="primary" :disabled="busy" @click="saveMatch">{{ editingMatchId ? 'Salva modifica' : 'Aggiungi partita' }}</button>
        <button v-if="editingMatchId" class="secondary" @click="resetMatch">Annulla</button>
      </div>
    </div>
    <article v-for="match in matches" :key="match.id" class="mini-card match-row">
      <div>
        <b class="match-team"><img v-if="match.teamALogo" :src="match.teamALogo" :alt="match.teamAName" />{{ match.teamAName }}</b>
        <small>vs</small>
        <b class="match-team"><img v-if="match.teamBLogo" :src="match.teamBLogo" :alt="match.teamBName" />{{ match.teamBName }}</b>
        <small>{{ match.court }} · {{ match.time || '—' }} · {{ match.stage || 'Girone' }} · {{ match.status }}</small>
      </div>
      <div class="actions"><button class="secondary" @click="editMatch(match)">Modifica</button><button class="danger" @click="deleteMatch(match)">Elimina</button></div>
    </article>
  </section>

  <section v-else-if="role === 'mvp'" class="mini-section">
    <div class="mvp-title"><div><h2>Andamento MVP</h2><p>Aggiornamento in tempo reale</p></div><strong>{{ mvp.totalVotes || 0 }}<small>voti</small></strong></div>
    <article v-for="team in mvpTeams" :key="team.id" class="mini-card">
      <div class="row-head"><b>{{ team.name }}</b><strong>{{ team.votes }}</strong></div>
      <div v-for="candidate in team.candidates" :key="candidate.id" class="candidate">
        <span>{{ candidate.name }} <small>{{ candidate.gender === 'female' ? 'F' : 'M' }}</small></span><b>{{ candidate.votes }}</b>
      </div>
    </article>
    <p v-if="!mvpTeams.length" class="empty">Nessun giocatore inserito nelle rose.</p>
  </section>
</template>

<style scoped>
.mini-section { display:flex; flex-direction:column; gap:10px; }
h2 { margin:6px 0 2px; font-size:20px; color:#f2b928; }
.mini-notice { position:sticky; top:6px; z-index:5; margin:0; padding:9px 12px; border-radius:9px; background:#3b3219; color:#fde68a; }
.mini-card { background:#15151b; border:1px solid rgba(255,255,255,.1); border-radius:12px; padding:12px; display:flex; flex-direction:column; gap:9px; }
.form-card input,.form-card select,.player-row input,.player-row select { box-sizing:border-box; width:100%; min-width:0; background:#0d0d12; border:1px solid rgba(255,255,255,.14); border-radius:8px; padding:10px; color:#fff; }
/* Materialize globale nasconde i select nativi: nella console operatore
   devono restare visibili e utilizzabili anche da smartphone. */
.mini-section select { display:block; height:auto; appearance:auto; -webkit-appearance:menulist; }
.mini-section option { background:#15151b; color:#fff; }
.form-grid { display:grid; grid-template-columns:repeat(3,1fr); gap:7px; }
.logo-picker { display:flex; align-items:center; gap:9px; width:max-content; max-width:100%; cursor:pointer; color:#fbd34d; font-size:12px; font-weight:750; }
.logo-picker img { width:34px; height:34px; object-fit:contain; border-radius:7px; background:#fff; }
.logo-picker input { position:absolute; width:1px; height:1px; opacity:0; pointer-events:none; }
.time-hint { color:#fbd34d; line-height:1.35; }
details summary { cursor:pointer; color:#f2b928; font-weight:750; padding:5px 0; }
.player-row { display:grid; grid-template-columns:22px 1fr 1fr 54px; gap:6px; align-items:center; margin-top:6px; }
.player-row > span { color:#64748b; text-align:center; font-size:12px; }
button { border:0; border-radius:8px; padding:9px 12px; font-weight:800; cursor:pointer; }
button:disabled { opacity:.5; }
.primary { background:#f2b928; color:#111; }
.secondary { background:#292933; color:#e2e8f0; }
.danger { background:transparent; color:#f87171; border:1px solid rgba(248,113,113,.35); }
.row-head,.match-row,.actions,.mvp-title { display:flex; align-items:center; justify-content:space-between; gap:8px; }
.row-head > div,.match-row > div { min-width:0; display:flex; flex-direction:column; }
.match-team { display:flex; align-items:center; gap:7px; }
.match-team img { width:24px; height:24px; flex:none; object-fit:contain; border-radius:50%; background:#fff; }
small { color:#94a3b8; font-size:11px; }
.chips { display:flex; flex-wrap:wrap; gap:5px; }
.chips span { padding:3px 8px; border-radius:999px; background:rgba(242,185,40,.1); color:#fbd34d; font-size:11px; }
.roster-edit { border-top:1px solid rgba(255,255,255,.08); padding-top:6px; }
.mvp-title h2,.mvp-title p { margin:0; }
.mvp-title p { color:#94a3b8; font-size:12px; }
.mvp-title > strong { font-size:28px; color:#f2b928; display:flex; flex-direction:column; text-align:center; }
.mvp-title > strong small { text-transform:uppercase; letter-spacing:.1em; }
.candidate { display:flex; justify-content:space-between; padding:7px 0; border-top:1px solid rgba(255,255,255,.06); }
.candidate b,.row-head > strong { color:#f2b928; }
.empty { color:#94a3b8; text-align:center; }
@media(max-width:430px){ .form-grid{grid-template-columns:1fr 1fr}.player-row{grid-template-columns:20px 1fr 1fr 48px}.match-row{align-items:flex-start;flex-direction:column}.match-row .actions{width:100%;justify-content:flex-end} }
</style>
