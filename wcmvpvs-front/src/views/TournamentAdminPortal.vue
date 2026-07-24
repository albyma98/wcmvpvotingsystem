<script setup>
// Pannello admin del singolo torneo (/ta/:slug).
// Mondo parallelo alle società: credenziali dedicate, cookie separato,
// scoping hard sull'evento lato backend.
// Due anime: SETUP (pre-torneo, con calma) e LIVE (console scoring, mobile,
// pulsanti grandi, undo sempre visibile).
import { computed, onMounted, reactive, ref } from 'vue'
import { useTournamentStream } from '@/composables/useTournamentStream'

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
  { id: 'shop', label: 'Shop' },
  { id: 'sponsors', label: 'Sponsor' },
  { id: 'gallery', label: 'Gallery' },
  { id: 'mvp', label: 'MVP' },
  { id: 'prizes', label: 'Premi' },
  { id: 'settings', label: 'Impostazioni' }
]

const overview = ref(null)
const teams = ref([])
const matches = ref([])
const sponsors = ref([])
const shopProducts = ref([])
const shopReservations = ref([])
const gallery = ref([])
const mvp = ref(null)
const emptyPrizes = () => ({ first: '', second: '', third: '', orgMvpMale: '', orgMvpFemale: '', publicMvpMale: '', publicMvpFemale: '' })
const defaultStandingsLegend = 'Primi 2 di ogni girone alla fase finale · Ordinamento: punti, quoziente set, quoziente punti'
const settings = reactive({ name: '', format: '', dateLabel: '', location: '', statusLabel: '', phaseLabel: '', logoUrl: '', prizes: emptyPrizes(), pointsPerWin: 3, pointsPerDraw: 1, pointsPerLoss: 0, setsBestOf: 3, pointsPerTieWin: 2, pointsPerTieLoss: 1, allowDraws: true, mvpByGender: true, tournamentStarted: false, bracketQualifiers: 2, bracketThirdPlace: false, standingsLegendText: defaultStandingsLegend, fanLayout: 'classic' })
const generatingBracket = ref(false)
const busy = ref('')
const notice = ref('')

function flash (msg) { notice.value = msg; setTimeout(() => { if (notice.value === msg) notice.value = '' }, 2500) }

async function bootstrap (ov) {
  overview.value = ov
  Object.assign(settings, ov.settings)
  ensurePrizes()
  await Promise.all([loadTeams(), loadMatches(), loadSponsors(), loadShopProducts(), loadShopReservations(), loadOperators(), loadGallery(), loadMvp()])
}
async function loadTeams () { const r = await j('GET', '/teams'); if (r.ok) teams.value = (await r.json()).teams }
async function loadMvp () { const r = await j('GET', '/mvp'); if (r.ok) mvp.value = await r.json() }
async function loadMatches () { const r = await j('GET', '/matches'); if (r.ok) matches.value = (await r.json()).matches }
async function loadSponsors () { const r = await j('GET', '/sponsors'); if (r.ok) sponsors.value = (await r.json()).sponsors }
async function loadShopProducts () { const r = await j('GET', '/shop'); if (r.ok) shopProducts.value = (await r.json()).products ?? [] }
async function loadShopReservations () { const r = await j('GET', '/shop/reservations'); if (r.ok) shopReservations.value = (await r.json()).reservations ?? [] }
// La lista gallery è l'endpoint pubblico (foto auto-pubblicate); il delete è admin.
async function loadGallery () {
  const r = await fetch(`/api/v1/tournaments/${props.slug}/gallery`)
  if (r.ok) gallery.value = (await r.json()).photos ?? []
}
const galleryImg = id => `/api/v1/tournaments/${props.slug}/gallery/${id}/thumb`
async function deleteGalleryPhoto (id) {
  if (!window.confirm('Rimuovere questa foto dalla gallery?')) return
  const r = await j('DELETE', `/gallery/${id}`)
  if (r.ok) { await loadGallery(); flash('Foto rimossa.') }
}

// ---------- mvp (monitoraggio votazioni) ----------
const mvpTotal = computed(() => mvp.value?.totalVotes ?? 0)
// Squadre ordinate per voti totali della squadra (più votate in cima), con i
// giocatori ordinati per voti decrescenti. Ogni giocatore porta la % sul totale.
const mvpTeams = computed(() => {
  const teams = (mvp.value?.teams ?? []).map(t => {
    const candidates = [...t.candidates].sort((a, b) => b.votes - a.votes)
    const teamVotes = candidates.reduce((s, c) => s + c.votes, 0)
    return { ...t, candidates, teamVotes }
  })
  return teams.sort((a, b) => b.teamVotes - a.teamVotes)
})
// Giocatore/i più votato/i. Con la votazione MVP separata uomo/donna mostriamo
// due leader distinti; `mvpLeaderBy` scorre le rose filtrando per genere.
const mvpLeaderBy = gender => {
  let best = null
  for (const t of mvp.value?.teams ?? []) {
    for (const c of t.candidates) {
      if ((c.gender || 'male') !== gender) continue
      if (c.votes > 0 && (!best || c.votes > best.votes)) best = { ...c, team: t.name }
    }
  }
  return best
}
const mvpLeaderMale = computed(() => mvpLeaderBy('male'))
const mvpLeaderFemale = computed(() => mvpLeaderBy('female'))
// Modalità della votazione (dal board): true = uomo/donna, false = MVP unico.
const mvpModeByGender = computed(() => mvp.value?.byGender !== false)
// MVP unico: il giocatore più votato in assoluto (indipendente dal sesso).
const mvpLeaderOverall = computed(() => {
  let best = null
  for (const t of mvp.value?.teams ?? []) {
    for (const c of t.candidates) {
      if (c.votes > 0 && (!best || c.votes > best.votes)) best = { ...c, team: t.name }
    }
  }
  return best
})
// Un candidato è "leader" (evidenziato in lista): l'MVP del suo genere nella
// modalità separata, oppure l'MVP assoluto nella modalità unica.
const isMvpLeader = c => mvpModeByGender.value
  ? ((mvpLeaderMale.value && c.id === mvpLeaderMale.value.id) ||
     (mvpLeaderFemale.value && c.id === mvpLeaderFemale.value.id))
  : (mvpLeaderOverall.value && c.id === mvpLeaderOverall.value.id)
const genderLabel = g => ((g || 'male') === 'female' ? 'Femmina' : 'Maschio')
const mvpPct = votes => (mvpTotal.value ? Math.round((votes / mvpTotal.value) * 100) : 0)

// ---------- squadre ----------
const MAX_PLAYERS = 8
// Rosa vuota: 8 coppie Nome/Cognome facoltative. Le coppie vuote vengono
// scartate lato server, così si può inviare la griglia piena senza problemi.
const blankRoster = () => Array.from({ length: MAX_PLAYERS }, () => ({ firstName: '', lastName: '', gender: 'male' }))

const newTeam = reactive({ name: '', shortName: '', city: '', groupName: '', players: blankRoster() })
async function addTeam () {
  const name = newTeam.name.trim()
  if (!name) { flash('Inserisci il nome della squadra.'); return }
  busy.value = 'teams'
  const r = await j('POST', '/teams', { teams: [{
    name,
    shortName: newTeam.shortName.trim(),
    city: newTeam.city.trim(),
    groupName: newTeam.groupName.trim(),
    players: newTeam.players
  }] })
  busy.value = ''
  if (r.ok) {
    newTeam.name = ''; newTeam.shortName = ''; newTeam.city = ''; newTeam.groupName = ''
    newTeam.players = blankRoster()
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

// ---------- modifica squadra (nome, sigla, città, girone) ----------
const editingTeamId = ref(null)
const teamDraft = reactive({ name: '', shortName: '', city: '', groupName: '' })
function openTeamEdit (t) {
  if (editingRosterId.value === t.id) closeRoster() // non tenere aperti entrambi gli editor
  editingTeamId.value = t.id
  teamDraft.name = t.name || ''
  teamDraft.shortName = t.shortName || ''
  teamDraft.city = t.city || ''
  teamDraft.groupName = t.groupName || ''
}
function cancelTeamEdit () { editingTeamId.value = null }
async function saveTeamEdit (id) {
  if (!teamDraft.name.trim()) { flash('Il nome della squadra è obbligatorio.'); return }
  busy.value = 'teamedit'
  const r = await j('PUT', `/teams/${id}`, {
    name: teamDraft.name.trim(),
    shortName: teamDraft.shortName.trim(),
    city: teamDraft.city.trim(),
    groupName: teamDraft.groupName.trim()
  })
  busy.value = ''
  if (r.ok) { editingTeamId.value = null; await loadTeams(); flash('Squadra aggiornata.') }
  else flash('Modifica non riuscita.')
}

// ---------- rosa giocatori (squadre già create) ----------
const editingRosterId = ref(null) // id squadra con l'editor rosa aperto
const rosterDraft = ref(blankRoster())
function openRoster (team) {
  if (editingTeamId.value === team.id) cancelTeamEdit() // non tenere aperti entrambi gli editor
  // Precarica i giocatori esistenti e completa fino a 8 righe.
  const rows = (team.players || []).map(p => ({ firstName: p.firstName || '', lastName: p.lastName || '', gender: p.gender === 'female' ? 'female' : 'male' }))
  while (rows.length < MAX_PLAYERS) rows.push({ firstName: '', lastName: '', gender: 'male' })
  rosterDraft.value = rows.slice(0, MAX_PLAYERS)
  editingRosterId.value = team.id
}
function closeRoster () { editingRosterId.value = null }
async function saveRoster (teamId) {
  busy.value = 'roster'
  const r = await j('PUT', `/teams/${teamId}/players`, { players: rosterDraft.value })
  busy.value = ''
  if (r.ok) { editingRosterId.value = null; await loadTeams(); flash('Rosa salvata.') }
  else flash('Errore nel salvataggio della rosa.')
}
const playerLabel = p => [p.firstName, p.lastName].filter(Boolean).join(' ')

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
// Segna/desegna la partita "inizio torneo": il suo orario diventa il taglio del
// ciclo cronologico (le partite dopo mezzanotte non scavalcano più quelle serali).
// Ne può esistere una sola: il backend azzera le altre.
async function toggleAnchor (m) {
  const on = !m.isAnchor
  const r = await j('PUT', `/matches/${m.id}/anchor`, { anchor: on })
  if (r.ok) { await loadMatches(); flash(on ? 'Impostata come inizio torneo.' : 'Inizio torneo rimosso.') }
  else flash('Operazione non riuscita.')
}

// ---------- modifica partita ----------
const editingMatchId = ref(null)
const matchDraft = reactive({ court: '', time: '', stage: '', teamAId: 0, teamBId: 0 })
function openMatchEdit (m) {
  editingMatchId.value = m.id
  matchDraft.court = m.court || ''
  matchDraft.time = m.time || ''
  matchDraft.stage = m.stage || ''
  matchDraft.teamAId = m.teamAId
  matchDraft.teamBId = m.teamBId
}
function cancelMatchEdit () { editingMatchId.value = null }
async function saveMatchEdit (id) {
  if (!matchDraft.teamAId || !matchDraft.teamBId || matchDraft.teamAId === matchDraft.teamBId) {
    flash('Scegli due squadre diverse.'); return
  }
  busy.value = 'matchedit'
  const scheduledAt = new Date().toISOString().slice(0, 10) + 'T' + (matchDraft.time || '00:00')
  const r = await j('PUT', `/matches/${id}`, { ...matchDraft, scheduledAt })
  busy.value = ''
  if (r.ok) { editingMatchId.value = null; await loadMatches(); flash('Partita aggiornata.') }
  else flash('Modifica non riuscita.')
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
    else if (err === 'teams_not_ready') flash('Squadre non ancora definite: attendi l\'esito del turno precedente.')
  }
}

// Refresh live via SSE (push): la console può restare aperta ore a bordo campo.
// Aggiorna il calendario partite quando l'admin è sui tab che lo mostrano
// (es. un operatore segna un punto da un altro campo).
onMounted(checkAuth)
useTournamentStream(props.slug, () => {
  if (!authed.value) return
  if (tab.value === 'live' || tab.value === 'matches') loadMatches()
  else if (tab.value === 'gallery') loadGallery()
  else if (tab.value === 'mvp') loadMvp()
  else if (tab.value === 'shop') loadShopReservations()
})

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

// ---------- shop ----------
const blankShopExtra = () => ({ title: '', price: '' })
const newShopProduct = reactive({
  imageUrl: '',
  title: '',
  description: '',
  price: '',
  extras: [blankShopExtra()]
})
const shopImageBusy = ref(false)

// Ritaglio centrale 16:9: ogni prodotto arriva al backend già con una vera
// immagine orizzontale uniforme, leggera e pronta per la modale dei tifosi.
function fileToHorizontalDataURL (file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    const img = new Image()
    reader.onload = () => { img.src = reader.result }
    reader.onerror = () => reject(new Error('read'))
    img.onerror = () => reject(new Error('decode'))
    img.onload = () => {
      const ratio = 16 / 9
      let sx = 0; let sy = 0; let sw = img.width; let sh = img.height
      if (img.width / img.height > ratio) {
        sw = img.height * ratio
        sx = (img.width - sw) / 2
      } else {
        sh = img.width / ratio
        sy = (img.height - sh) / 2
      }
      const canvas = document.createElement('canvas')
      canvas.width = 960
      canvas.height = 540
      canvas.getContext('2d').drawImage(img, sx, sy, sw, sh, 0, 0, canvas.width, canvas.height)
      let out = canvas.toDataURL('image/webp', 0.82)
      if (!out.startsWith('data:image/webp')) out = canvas.toDataURL('image/jpeg', 0.85)
      resolve(out)
    }
    reader.readAsDataURL(file)
  })
}

async function onShopImagePick (e) {
  const file = e.target.files?.[0]
  if (!file) return
  if (!file.type.startsWith('image/')) {
    flash('Seleziona un file immagine.')
    e.target.value = ''
    return
  }
  shopImageBusy.value = true
  try {
    newShopProduct.imageUrl = await fileToHorizontalDataURL(file)
  } catch {
    flash('Immagine prodotto non valida.')
  } finally {
    shopImageBusy.value = false
    e.target.value = ''
  }
}

function addShopExtra () {
  if (newShopProduct.extras.length < 12) newShopProduct.extras.push(blankShopExtra())
}
function removeShopExtra (index) {
  newShopProduct.extras.splice(index, 1)
  if (!newShopProduct.extras.length) newShopProduct.extras.push(blankShopExtra())
}
function euroToCents (value) {
  const amount = Number(String(value ?? '').replace(',', '.'))
  return Number.isFinite(amount) ? Math.round(amount * 100) : 0
}
function formatEuro (cents) {
  return new Intl.NumberFormat('it-IT', { style: 'currency', currency: 'EUR' }).format((cents || 0) / 100)
}
function formatReservationDate (value) {
  if (!value) return ''
  const normalized = value.includes('T') ? value : value.replace(' ', 'T') + 'Z'
  const date = new Date(normalized)
  return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat('it-IT', {
    dateStyle: 'short', timeStyle: 'short'
  }).format(date)
}
const phoneHref = phone => `tel:${String(phone || '').replace(/[^\d+]/g, '')}`
function resetShopForm () {
  newShopProduct.imageUrl = ''
  newShopProduct.title = ''
  newShopProduct.description = ''
  newShopProduct.price = ''
  newShopProduct.extras.splice(0, newShopProduct.extras.length, blankShopExtra())
}
async function createShopProduct () {
  if (!newShopProduct.imageUrl) { flash('Carica l’immagine orizzontale del prodotto.'); return }
  const priceCents = euroToCents(newShopProduct.price)
  if (priceCents <= 0) { flash('Inserisci un prezzo valido maggiore di zero.'); return }
  const extras = []
  for (const extra of newShopProduct.extras) {
    const title = extra.title.trim()
    const extraPrice = euroToCents(extra.price)
    if (!title && !extra.price) continue
    if (!title || extraPrice <= 0) {
      flash('Ogni extra deve avere sia il titolo sia un prezzo valido.')
      return
    }
    extras.push({ title, priceCents: extraPrice })
  }
  busy.value = 'shop'
  const r = await j('POST', '/shop', {
    imageUrl: newShopProduct.imageUrl,
    title: newShopProduct.title.trim(),
    description: newShopProduct.description.trim(),
    priceCents,
    extras
  })
  busy.value = ''
  if (r.ok) {
    resetShopForm()
    await loadShopProducts()
    flash('Prodotto aggiunto allo Shop.')
    return
  }
  const err = (await r.json().catch(() => ({}))).error
  flash(err === 'logo_too_large' ? 'Immagine troppo pesante.'
    : err === 'bad_extra' ? 'Controlla titoli e prezzi degli extra.'
    : 'Prodotto non salvato.')
}
async function deleteShopProduct (id) {
  if (!window.confirm('Rimuovere questo prodotto dallo Shop?')) return
  const r = await j('DELETE', `/shop/${id}`)
  if (r.ok) {
    await loadShopProducts()
    flash('Prodotto rimosso.')
  }
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
  else {
    const err = (await r.json().catch(() => ({}))).error
    flash(err === 'logo_too_large' ? 'Immagine intestazione troppo pesante.'
      : err && err.startsWith('logo') ? 'Immagine intestazione non valida.'
      : 'Salvataggio non riuscito.')
  }
}

// Intestazione home tifosi: carica un'immagine (ridimensionata lato client a
// data-URL, lato lungo 800px) da mostrare al posto del titolo. Vuota = nome.
const headerBusy = ref(false)
async function onHeaderPick (e) {
  const file = e.target.files?.[0]
  if (!file) return
  if (!file.type.startsWith('image/')) { flash('Seleziona un file immagine.'); e.target.value = ''; return }
  headerBusy.value = true
  try {
    settings.logoUrl = await fileToLogoDataURL(file, 800)
  } catch { flash('Immagine non valida.') }
  finally { headerBusy.value = false; e.target.value = '' }
}
function removeHeader () { settings.logoUrl = '' }

// Premi: assicura che l'oggetto abbia tutte le chiavi anche se il backend ne
// omette qualcuna (difesa contro settings caricate parziali).
function ensurePrizes () {
  if (!settings.prizes || typeof settings.prizes !== 'object') settings.prizes = emptyPrizes()
  else settings.prizes = { ...emptyPrizes(), ...settings.prizes }
}

// ---------- fase finale ----------
const bracketErrors = {
  groups_not_finished: 'Prima concludi tutte le partite dei gironi.',
  no_group_matches: 'Non ci sono partite di girone concluse.',
  no_groups: 'Nessun girone trovato.',
  not_enough_teams: 'Un girone ha meno squadre delle qualificate impostate.'
}
async function generateBracket () {
  const nGroups = new Set(teams.value.map(t => t.groupName || '')).size
  const q = settings.bracketQualifiers * nGroups
  const isPow2 = q >= 2 && (q & (q - 1)) === 0
  const msg = isPow2
    ? `Genero il tabellone con ${q} squadre qualificate. ATTENZIONE: sostituisce l'eventuale tabellone esistente (punteggi della fase finale persi). Procedere?`
    : `Con ${settings.bracketQualifiers} qualificate × ${nGroups} gironi = ${q} squadre: non è una potenza di 2 (servono 2, 4, 8, 16). Sistema gironi/qualificate. Provo comunque?`
  if (!window.confirm(msg)) return
  generatingBracket.value = true
  try {
    await j('PUT', '/settings', { ...settings }) // persisti qualificate/finalina prima di generare
    const r = await j('POST', '/bracket/generate')
    const data = await r.json().catch(() => ({}))
    if (!r.ok) {
      if (data.error?.startsWith('not_power_of_two')) {
        flash(`Le squadre qualificate (${data.error.split(':')[1]}) devono essere una potenza di 2 (2/4/8/16).`)
      } else {
        flash(bracketErrors[data.error] || 'Generazione tabellone non riuscita.')
      }
      return
    }
    await loadMatches()
    tab.value = 'matches'
    flash(`Tabellone generato: ${data.created} partite.`)
  } finally {
    generatingBracket.value = false
  }
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
        <p class="hint">🏁 Segna una partita come <b>inizio torneo</b>: il suo orario diventa il taglio del calendario. Le partite dopo mezzanotte (es. le 02:00) restano così in coda alla giornata invece di scavalcare quelle serali. Ne puoi impostare una sola.</p>
        <div class="form-row">
          <input v-model="newMatch.court" placeholder="CAMPO 1" style="max-width:110px" />
          <input v-model="newMatch.time" placeholder="18:30" style="max-width:90px" />
          <select v-model.number="newMatch.teamAId"><option :value="0">Squadra A</option><option v-for="t in teams" :key="t.id" :value="t.id">{{ t.name }}</option></select>
          <select v-model.number="newMatch.teamBId"><option :value="0">Squadra B</option><option v-for="t in teams" :key="t.id" :value="t.id">{{ t.name }}</option></select>
          <select v-model="newMatch.stage"><option value="">Girone</option><option>QUARTI</option><option>SEMIFINALE</option><option>FINALE 3° POSTO</option><option>FINALE</option></select>
          <button @click="createMatch">Aggiungi</button>
        </div>
        <div v-for="m in matches" :key="m.id" class="match-item">
          <div v-if="editingMatchId !== m.id" class="row-match">
            <span>
              <span v-if="m.isAnchor" class="anchor-flag" title="Inizio torneo: da qui parte il calendario">🏁</span>
              {{ m.court }} · {{ m.time || '—' }} · {{ m.teamAName }} vs {{ m.teamBName }}
              <small v-if="m.stage">· {{ m.stage }}</small>
              <small>[{{ m.status }}]</small>
            </span>
            <span class="team-actions">
              <button class="ghost" :class="{ 'anchor-on': m.isAnchor }" @click="toggleAnchor(m)"
                      :title="m.isAnchor ? 'Rimuovi «inizio torneo»' : 'Segna come inizio torneo (taglio del calendario)'">
                🏁 {{ m.isAnchor ? 'Inizio ✓' : 'Inizio' }}
              </button>
              <button class="ghost" @click="openMatchEdit(m)">Modifica</button>
              <button class="danger" @click="deleteMatch(m.id)">Elimina</button>
            </span>
          </div>

          <div v-else class="match-editor">
            <div class="form-row">
              <input v-model="matchDraft.court" placeholder="CAMPO 1" style="max-width:110px" />
              <input v-model="matchDraft.time" placeholder="18:30" style="max-width:90px" />
              <select v-model="matchDraft.stage"><option value="">Girone</option><option>QUARTI</option><option>SEMIFINALE</option><option>FINALE 3° POSTO</option><option>FINALE</option></select>
            </div>
            <div class="form-row">
              <select v-model.number="matchDraft.teamAId"><option :value="0">Squadra A</option><option v-for="t in teams" :key="t.id" :value="t.id">{{ t.name }}</option></select>
              <select v-model.number="matchDraft.teamBId"><option :value="0">Squadra B</option><option v-for="t in teams" :key="t.id" :value="t.id">{{ t.name }}</option></select>
            </div>
            <div class="form-row">
              <button :disabled="busy === 'matchedit'" @click="saveMatchEdit(m.id)">{{ busy === 'matchedit' ? 'Salvo…' : 'Salva' }}</button>
              <button class="ghost" @click="cancelMatchEdit">Annulla</button>
            </div>
            <p v-if="m.status !== 'scheduled'" class="hint">Attenzione: questa partita è «{{ m.status }}». Modificare le squadre può rendere incoerenti punteggi/classifica.</p>
          </div>
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

          <details class="roster-block">
            <summary>Giocatori (facoltativi) — serviranno per la votazione MVP del pubblico</summary>
            <p class="hint">Inserisci Nome e Cognome dei giocatori di questa squadra: fino a {{ MAX_PLAYERS }}, tutti facoltativi. Puoi anche aggiungerli in seguito.</p>
            <div class="roster-grid">
              <div v-for="(p, i) in newTeam.players" :key="i" class="roster-row">
                <span class="roster-num">{{ i + 1 }}</span>
                <input v-model="p.firstName" placeholder="Nome" />
                <input v-model="p.lastName" placeholder="Cognome" />
                <span class="roster-gender">
                  <label><input type="radio" :name="`ng-${i}`" value="male" v-model="p.gender" /> M</label>
                  <label><input type="radio" :name="`ng-${i}`" value="female" v-model="p.gender" /> F</label>
                </span>
              </div>
            </div>
          </details>

          <button :disabled="busy === 'teams' || !newTeam.name.trim()" @click="addTeam">
            {{ busy === 'teams' ? 'Salvo…' : 'Aggiungi squadra' }}
          </button>
        </div>

        <div class="team-list">
          <p v-if="!teams.length" class="hint">Nessuna squadra ancora. Aggiungi la prima qui sopra.</p>
          <div v-for="t in teams" :key="t.id" class="team-item">
            <div class="row-match">
              <span>
                <b>{{ t.name }}</b>
                <small v-if="t.shortName">({{ t.shortName }})</small>
                <small v-if="t.city">· {{ t.city }}</small>
                <small v-if="t.groupName">· Girone {{ t.groupName }}</small>
                <small v-if="t.players && t.players.length" class="roster-count">· {{ t.players.length }} giocatori</small>
              </span>
              <span class="team-actions">
                <button class="ghost" @click="editingTeamId === t.id ? cancelTeamEdit() : openTeamEdit(t)">
                  {{ editingTeamId === t.id ? 'Chiudi' : 'Modifica' }}
                </button>
                <button class="ghost" @click="editingRosterId === t.id ? closeRoster() : openRoster(t)">
                  {{ editingRosterId === t.id ? 'Chiudi' : 'Rosa' }}
                </button>
                <button class="danger" @click="deleteTeam(t.id)">Elimina</button>
              </span>
            </div>

            <!-- Editor squadra: nome, sigla, città, girone -->
            <div v-if="editingTeamId === t.id" class="team-editor">
              <div class="form-row">
                <input v-model="teamDraft.name" placeholder="Nome squadra *" @keyup.enter="saveTeamEdit(t.id)" />
                <input v-model="teamDraft.shortName" placeholder="Sigla" maxlength="6" style="max-width:120px" @keyup.enter="saveTeamEdit(t.id)" />
              </div>
              <div class="form-row">
                <input v-model="teamDraft.city" placeholder="Città" @keyup.enter="saveTeamEdit(t.id)" />
                <input v-model="teamDraft.groupName" placeholder="Girone" maxlength="4" style="max-width:100px" @keyup.enter="saveTeamEdit(t.id)" />
              </div>
              <div class="form-row">
                <button :disabled="busy === 'teamedit'" @click="saveTeamEdit(t.id)">
                  {{ busy === 'teamedit' ? 'Salvo…' : 'Salva' }}
                </button>
                <button class="ghost" @click="cancelTeamEdit">Annulla</button>
              </div>
            </div>

            <div v-if="t.players && t.players.length && editingRosterId !== t.id && editingTeamId !== t.id" class="roster-chips">
              <span v-for="p in t.players" :key="p.id" class="chip">{{ playerLabel(p) }}</span>
            </div>

            <div v-if="editingRosterId === t.id" class="roster-editor">
              <div class="roster-grid">
                <div v-for="(p, i) in rosterDraft" :key="i" class="roster-row">
                  <span class="roster-num">{{ i + 1 }}</span>
                  <input v-model="p.firstName" placeholder="Nome" />
                  <input v-model="p.lastName" placeholder="Cognome" />
                  <span class="roster-gender">
                    <label><input type="radio" :name="`rg-${i}`" value="male" v-model="p.gender" /> M</label>
                    <label><input type="radio" :name="`rg-${i}`" value="female" v-model="p.gender" /> F</label>
                  </span>
                </div>
              </div>
              <div class="form-row">
                <button :disabled="busy === 'roster'" @click="saveRoster(t.id)">
                  {{ busy === 'roster' ? 'Salvo…' : 'Salva rosa' }}
                </button>
                <button class="ghost" @click="closeRoster">Annulla</button>
              </div>
            </div>
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

      <!-- SHOP -->
      <section v-else-if="tab === 'shop'" class="ta-body">
        <p class="hint">Crea i prodotti mostrati nella modale Shop della home Sunset. Immagine e prezzo sono obbligatori; titolo, descrizione ed extra sono facoltativi.</p>

        <div class="shop-editor">
          <div class="shop-image-editor">
            <div class="shop-image-preview" :class="{ empty: !newShopProduct.imageUrl }">
              <img v-if="newShopProduct.imageUrl" :src="newShopProduct.imageUrl" alt="Anteprima prodotto" />
              <span v-else>Immagine orizzontale 16:9</span>
            </div>
            <div class="shop-image-actions">
              <label class="shop-upload">
                {{ shopImageBusy ? 'Elaborazione…' : (newShopProduct.imageUrl ? 'Cambia immagine' : 'Carica immagine *') }}
                <input type="file" accept="image/*" :disabled="shopImageBusy" @change="onShopImagePick" hidden />
              </label>
              <button v-if="newShopProduct.imageUrl" class="ghost" type="button" @click="newShopProduct.imageUrl = ''">Rimuovi</button>
            </div>
          </div>

          <div class="shop-main-fields">
            <label>Titolo <span>(opzionale)</span>
              <input v-model="newShopProduct.title" maxlength="120" placeholder="Es. Panino special" />
            </label>
            <label>Prezzo *
              <div class="price-field"><span>€</span><input v-model="newShopProduct.price" type="number" min="0.01" step="0.01" inputmode="decimal" placeholder="0,00" /></div>
            </label>
            <label class="shop-description">Descrizione <span>(opzionale)</span>
              <textarea v-model="newShopProduct.description" maxlength="600" rows="3" placeholder="Descrivi brevemente il prodotto"></textarea>
            </label>
          </div>

          <div class="shop-extras">
            <div class="shop-extras-head">
              <div><b>Extra</b> <span>(opzionali)</span></div>
              <button class="ghost" type="button" @click="addShopExtra">+ Aggiungi extra</button>
            </div>
            <div v-for="(extra, index) in newShopProduct.extras" :key="index" class="shop-extra-row">
              <input v-model="extra.title" maxlength="80" placeholder="Titolo extra (es. Bacon)" />
              <div class="price-field"><span>€</span><input v-model="extra.price" type="number" min="0.01" step="0.01" inputmode="decimal" placeholder="0,00" /></div>
              <button class="danger" type="button" aria-label="Rimuovi extra" @click="removeShopExtra(index)">×</button>
            </div>
          </div>

          <button class="shop-save" :disabled="busy === 'shop' || shopImageBusy" @click="createShopProduct">
            {{ busy === 'shop' ? 'Salvataggio…' : 'Aggiungi prodotto' }}
          </button>
        </div>

        <h3>Prodotti caricati</h3>
        <p v-if="!shopProducts.length" class="hint">Non hai ancora caricato prodotti.</p>
        <div v-else class="shop-admin-list">
          <article v-for="product in shopProducts" :key="product.id" class="shop-admin-product">
            <img :src="product.imageUrl" :alt="product.title || 'Prodotto Shop'" />
            <div class="shop-admin-copy">
              <b v-if="product.title">{{ product.title }}</b>
              <b v-else class="shop-untitled">Prodotto senza titolo</b>
              <p v-if="product.description">{{ product.description }}</p>
              <div v-if="product.extras?.length" class="shop-admin-extras">
                <span v-for="extra in product.extras" :key="`${product.id}-${extra.title}`">
                  + {{ extra.title }} {{ formatEuro(extra.priceCents) }}
                </span>
              </div>
            </div>
            <strong class="shop-admin-price">{{ formatEuro(product.priceCents) }}</strong>
            <button class="danger" type="button" @click="deleteShopProduct(product.id)">Rimuovi</button>
          </article>
        </div>

        <div class="shop-reservations-head">
          <div>
            <h3>Prenotazioni ricevute</h3>
            <p class="hint">Contatta il cliente al numero indicato per concordare pagamento e consegna.</p>
          </div>
          <button class="ghost" type="button" @click="loadShopReservations">Aggiorna</button>
        </div>
        <p v-if="!shopReservations.length" class="hint">Non sono ancora arrivate prenotazioni.</p>
        <div v-else class="shop-reservation-list">
          <article v-for="reservation in shopReservations" :key="reservation.id" class="shop-reservation">
            <img :src="reservation.productImageUrl" :alt="reservation.productTitle || 'Prodotto Shop'" />
            <div class="shop-reservation-product">
              <b>{{ reservation.productTitle || 'Prodotto Shop' }}</b>
              <span v-for="extra in reservation.selectedExtras" :key="`${reservation.id}-${extra.title}`">
                + {{ extra.title }} {{ formatEuro(extra.priceCents) }}
              </span>
              <small>{{ formatReservationDate(reservation.createdAt) }}</small>
            </div>
            <div class="shop-reservation-customer">
              <b>{{ reservation.firstName }} {{ reservation.lastName }}</b>
              <a :href="phoneHref(reservation.phone)">☎ {{ reservation.phone }}</a>
            </div>
            <strong class="shop-admin-price">{{ formatEuro(reservation.totalPriceCents) }}</strong>
          </article>
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

      <!-- GALLERY (moderazione) -->
      <section v-else-if="tab === 'gallery'" class="ta-body">
        <p class="hint">Le foto sono pubblicate dai tifosi e visibili a tutti. Rimuovi qui quelle inappropriate: la rimozione è immediata su ogni dispositivo.</p>
        <p v-if="!gallery.length" class="hint">Nessuna foto pubblicata dai tifosi al momento.</p>
        <div v-else class="mod-grid">
          <div v-for="p in gallery" :key="p.id" class="mod-cell">
            <img :src="galleryImg(p.id)" alt="Foto tifoso" loading="lazy" />
            <button class="mod-del" title="Rimuovi" @click="deleteGalleryPhoto(p.id)">🗑</button>
          </div>
        </div>
      </section>

      <!-- MVP: monitoraggio votazioni -->
      <section v-else-if="tab === 'mvp'" class="ta-body">
        <p class="hint">Andamento in tempo reale della votazione MVP del pubblico. I conteggi si aggiornano da soli man mano che i tifosi votano.</p>

        <div class="mvp-summary">
          <div class="mvp-stat">
            <span class="num">{{ mvpTotal }}</span>
            <span class="lbl">{{ mvpTotal === 1 ? 'voto totale' : 'voti totali' }}</span>
          </div>
          <template v-if="mvpModeByGender">
            <div class="mvp-leader" :class="{ empty: !mvpLeaderMale }">
              <span class="crown">👑</span>
              <div>
                <span class="mvp-leader__tag">MVP Uomo</span>
                <template v-if="mvpLeaderMale">
                  <b>{{ mvpLeaderMale.name }}</b>
                  <small>{{ mvpLeaderMale.team }} · {{ mvpLeaderMale.votes }} voti · {{ mvpPct(mvpLeaderMale.votes) }}%</small>
                </template>
                <small v-else>Ancora nessun voto</small>
              </div>
            </div>
            <div class="mvp-leader" :class="{ empty: !mvpLeaderFemale }">
              <span class="crown">👑</span>
              <div>
                <span class="mvp-leader__tag">MVP Donna</span>
                <template v-if="mvpLeaderFemale">
                  <b>{{ mvpLeaderFemale.name }}</b>
                  <small>{{ mvpLeaderFemale.team }} · {{ mvpLeaderFemale.votes }} voti · {{ mvpPct(mvpLeaderFemale.votes) }}%</small>
                </template>
                <small v-else>Ancora nessun voto</small>
              </div>
            </div>
          </template>
          <div v-else class="mvp-leader" :class="{ empty: !mvpLeaderOverall }">
            <span class="crown">👑</span>
            <div>
              <span class="mvp-leader__tag">MVP</span>
              <template v-if="mvpLeaderOverall">
                <b>{{ mvpLeaderOverall.name }}</b>
                <small>{{ mvpLeaderOverall.team }} · {{ mvpLeaderOverall.votes }} voti · {{ mvpPct(mvpLeaderOverall.votes) }}%</small>
              </template>
              <small v-else>Ancora nessun voto</small>
            </div>
          </div>
        </div>

        <p v-if="!mvpTeams.length" class="hint">Nessuna squadra con giocatori in rosa: aggiungi i giocatori dalla scheda «Squadre» per abilitare la votazione.</p>
        <p v-else-if="!mvpTotal" class="hint">Ancora nessun voto. Appena i tifosi votano, qui vedrai i risultati per squadra.</p>

        <div class="mvp-a-grid">
          <section v-for="t in mvpTeams" :key="t.id" class="mvp-a-box">
            <header class="mab-head">
              <span class="mab-name">{{ t.name }}</span>
              <span class="mab-total">{{ t.teamVotes }} <small>voti</small></span>
            </header>
            <div v-for="c in t.candidates" :key="c.id" class="mab-row" :class="{ lead: isMvpLeader(c) }">
              <span class="bar" :style="{ width: mvpPct(c.votes) + '%' }"></span>
              <span class="mab-player">{{ c.name }} <small class="mab-gender">{{ (c.gender || 'male') === 'female' ? '♀' : '♂' }}</small></span>
              <span class="mab-count">{{ c.votes }} · {{ mvpPct(c.votes) }}%</span>
            </div>
          </section>
        </div>
      </section>

      <!-- PREMI -->
      <section v-else-if="tab === 'prizes'" class="ta-body">
        <p class="hint">Compila i premi del torneo: verranno mostrati ai tifosi nella modale «Premi» (icona 🏆 in alto a destra della home Sunset). I campi lasciati vuoti non vengono mostrati; se sono tutti vuoti i tifosi vedranno «Premi in arrivo».</p>

        <h3 class="settings-sub">Classifica</h3>
        <div class="settings-grid">
          <label>🥇 1° classificato <input v-model="settings.prizes.first" maxlength="200" placeholder="Es. Buono 300€ + medaglie" /></label>
          <label>🥈 2° classificato <input v-model="settings.prizes.second" maxlength="200" placeholder="Es. Buono 150€" /></label>
          <label>🥉 3° classificato <input v-model="settings.prizes.third" maxlength="200" placeholder="Es. Kit sportivo" /></label>
        </div>

        <h3 class="settings-sub">MVP scelti dagli organizzatori</h3>
        <div class="settings-grid">
          <label>♂ MVP maschile <input v-model="settings.prizes.orgMvpMale" maxlength="200" placeholder="Es. Pallone ufficiale" /></label>
          <label>♀ MVP femminile <input v-model="settings.prizes.orgMvpFemale" maxlength="200" placeholder="Es. Pallone ufficiale" /></label>
        </div>

        <h3 class="settings-sub">MVP scelti dal pubblico</h3>
        <div class="settings-grid">
          <label>♂ MVP maschile <input v-model="settings.prizes.publicMvpMale" maxlength="200" placeholder="Es. Buono cena per due" /></label>
          <label>♀ MVP femminile <input v-model="settings.prizes.publicMvpFemale" maxlength="200" placeholder="Es. Buono cena per due" /></label>
        </div>

        <button :disabled="busy === 'settings'" @click="saveSettings">{{ busy === 'settings' ? 'Salvo…' : 'Salva premi' }}</button>
      </section>

      <!-- IMPOSTAZIONI -->
      <section v-else class="ta-body">
        <h3 class="settings-sub">Stato del torneo</h3>
        <div class="settings-grid">
          <label class="check">
            <input type="checkbox" v-model="settings.tournamentStarted" />
            <span>Torneo iniziato</span>
          </label>
        </div>
        <p class="hint">
          Se disattivato, i tifosi possono consultare la pagina ma non possono votare l'MVP
          né caricare immagini nella Gallery.
        </p>

        <div class="settings-grid">
          <label>Nome <input v-model="settings.name" /></label>
          <label>Formato <input v-model="settings.format" /></label>
          <label>Date <input v-model="settings.dateLabel" /></label>
          <label>Luogo <input v-model="settings.location" /></label>
          <label>Stato (pill) <input v-model="settings.statusLabel" placeholder="TORNEO IN CORSO" /></label>
          <label>Fase (pill) <input v-model="settings.phaseLabel" placeholder="FASE A GIRONI" /></label>
        </div>

        <h3 class="settings-sub">Intestazione home tifosi</h3>
        <div class="header-img">
          <div class="hi-preview" :class="{ empty: !settings.logoUrl }">
            <img v-if="settings.logoUrl" :src="settings.logoUrl" alt="Intestazione torneo" />
            <span v-else class="hi-fallback">{{ settings.name || 'Nome torneo' }}</span>
          </div>
          <div class="hi-actions">
            <label class="hi-upload">
              {{ headerBusy ? 'Carico…' : (settings.logoUrl ? 'Cambia immagine' : 'Carica immagine') }}
              <input type="file" accept="image/*" :disabled="headerBusy" @change="onHeaderPick" hidden />
            </label>
            <button v-if="settings.logoUrl" class="ghost" @click="removeHeader">Rimuovi</button>
          </div>
        </div>
        <p class="hint">Immagine mostrata in cima alla pagina tifosi al posto del titolo (es. il logo del torneo). Se non carichi nulla, viene usato il <b>nome del torneo</b>. L'immagine viene ridimensionata automaticamente.</p>

        <h3 class="settings-sub">Grafica home tifosi</h3>
        <div class="settings-grid">
          <label>Layout
            <select v-model="settings.fanLayout">
              <option value="classic">Classic (scuro + oro)</option>
              <option value="sunset">Sunset (pop estivo)</option>
            </select>
          </label>
        </div>
        <p class="hint">Cambia l'aspetto della pagina che vedono i tifosi (/t/{{ slug }}). Le funzioni (live, calendario, classifiche, gallery…) restano identiche. Salva e ricarica la pagina tifosi per vedere il nuovo layout.</p>

        <h3 class="settings-sub">Formula set</h3>
        <div class="settings-grid">
          <label>Partita al meglio di
            <select v-model.number="settings.setsBestOf">
              <option :value="3">2 su 3 (al meglio dei 3 set)</option>
              <option :value="5">3 su 5 (al meglio dei 5 set)</option>
            </select>
          </label>
        </div>
        <p class="hint">Vince la partita chi arriva prima a {{ settings.setsBestOf === 5 ? '3 set (su 5)' : '2 set (su 3)' }}. Il <b>tie-break</b> è il set decisivo: {{ settings.setsBestOf === 5 ? '3-2' : '2-1' }}.</p>

        <h3 class="settings-sub">Votazione MVP (pubblico)</h3>
        <div class="settings-grid">
          <label>Modalità
            <select v-model="settings.mvpByGender">
              <option :value="true">MVP Uomo + MVP Donna (2 voti)</option>
              <option :value="false">MVP unico (1 voto)</option>
            </select>
          </label>
        </div>
        <p class="hint">{{ settings.mvpByGender ? 'Il tifoso ha due voti: un MVP uomo e un MVP donna (in base al sesso indicato nella rosa).' : 'Il tifoso ha un solo voto per il miglior giocatore, indipendentemente dal sesso.' }} Conviene deciderla <b>prima</b> che inizi la votazione. Salva e ricarica la pagina tifosi.</p>

        <h3 class="settings-sub">Punti classifica (gironi)</h3>
        <div class="settings-grid">
          <label class="check">
            <input type="checkbox" v-model="settings.allowDraws" />
            <span>Consenti pareggi in classifica</span>
          </label>
        </div>
        <div class="settings-grid">
          <label>Punti per vittoria <input type="number" min="0" max="100" v-model.number="settings.pointsPerWin" /></label>
          <label>Punti vittoria al tie-break <input type="number" min="0" max="100" v-model.number="settings.pointsPerTieWin" /></label>
          <label>Punti sconfitta al tie-break <input type="number" min="0" max="100" v-model.number="settings.pointsPerTieLoss" /></label>
          <label v-if="settings.allowDraws">Punti per pareggio <input type="number" min="0" max="100" v-model.number="settings.pointsPerDraw" /></label>
          <label>Punti per sconfitta <input type="number" min="0" max="100" v-model.number="settings.pointsPerLoss" /></label>
        </div>
        <p class="hint">Assegnati a ogni squadra per ogni partita di girone conclusa. Al <b>tie-break</b> ({{ settings.setsBestOf === 5 ? '3-2' : '2-1' }}) chi vince prende i «Punti vittoria al tie-break» e chi perde i «Punti sconfitta al tie-break», invece dei punti pieni. {{ settings.allowDraws ? 'Il pareggio scatta quando la partita finisce con set pari (dove il formato lo prevede).' : 'I pareggi sono disattivati: le partite con set pari non contano come pareggio e la colonna «N» è nascosta ai tifosi.' }} La classifica ordina per punti totali, poi quoziente set e quoziente punti. La fase finale (tabellone) non è influenzata.</p>

        <h3 class="settings-sub">Fase finale (tabellone)</h3>
        <div class="settings-grid">
          <label>Qualificate per girone <input type="number" min="1" max="8" v-model.number="settings.bracketQualifiers" /></label>
          <label class="check">
            <input type="checkbox" v-model="settings.bracketThirdPlace" />
            <span>Finale 3°/4° posto</span>
          </label>
          <label class="settings-wide">Testo informativo sotto la classifica
            <textarea v-model="settings.standingsLegendText" maxlength="300" rows="2" :placeholder="defaultStandingsLegend"></textarea>
          </label>
        </div>
        <p class="hint">A gironi conclusi, «Genera tabellone» crea in automatico gli incroci di quarti/semifinali/finale: primo turno con le squadre reali, i turni dopo con segnaposto («Vincente Q1») che si riempiono da soli man mano che le partite finiscono. Le squadre totali qualificate devono essere una potenza di 2 (2/4/8/16).</p>
        <div class="bracket-actions">
          <button :disabled="busy === 'settings'" @click="saveSettings">{{ busy === 'settings' ? 'Salvo…' : 'Salva impostazioni' }}</button>
          <button class="gen" :disabled="generatingBracket" @click="generateBracket">
            {{ generatingBracket ? 'Genero…' : '⚡ Genera tabellone' }}
          </button>
        </div>
        <p class="hint">Stato e fase sono le due pill nell'hero dei tifosi: cambiale quando avanzi il torneo (es. «FASE FINALE»).</p>
      </section>
    </template>

    <p v-else class="ta-loading">Caricamento…</p>
  </div>
</template>

<style scoped>
.ta-page {
  width: 100%;
  min-height: 100vh;
  min-height: 100dvh;
  box-sizing: border-box;
  overflow-x: hidden;
  background: #0a0a0e;
  color: #f1f5f9;
  padding: clamp(16px, 2vw, 32px);
  font-size: 14.5px;
}
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
.team-item { display: flex; flex-direction: column; gap: 6px; }
.team-actions { display: flex; gap: 6px; flex: none; }
.roster-count { color: #f2b928 !important; }
/* Blocco rosa nel form di aggiunta squadra */
.roster-block { border: 1px solid rgba(255,255,255,.12); border-radius: 8px; padding: 4px 10px; }
.roster-block > summary { cursor: pointer; font-size: 13px; font-weight: 700; color: #f2b928; padding: 6px 0; }
.roster-grid { display: flex; flex-direction: column; gap: 6px; margin: 6px 0; }
.roster-row { display: flex; align-items: center; gap: 8px; }
.roster-num { flex: none; width: 20px; text-align: center; color: #64748b; font-size: 13px; font-weight: 700; }
.roster-row input { flex: 1; min-width: 0; background: #15151b; border: 1px solid rgba(255,255,255,.14); border-radius: 8px; padding: 8px 10px; color: #fff; font-size: 14px; }
.roster-gender { flex: none; display: flex; gap: 8px; }
.roster-gender label { display: inline-flex; align-items: center; gap: 3px; font-size: 12.5px; color: #cbd5e1; cursor: pointer; }
.roster-gender input[type="radio"] { flex: none; width: auto; margin: 0; accent-color: #f2b928; }
.roster-editor { background: #101017; border: 1px solid rgba(255,255,255,.09); border-radius: 10px; padding: 10px 12px; display: flex; flex-direction: column; gap: 8px; }
.match-item { display: flex; flex-direction: column; gap: 6px; }
.match-editor, .team-editor { background: #101017; border: 1px solid rgba(255,255,255,.09); border-radius: 10px; padding: 10px 12px; display: flex; flex-direction: column; gap: 8px; }
.roster-chips { display: flex; flex-wrap: wrap; gap: 6px; padding: 0 2px; }
.chip { background: rgba(242,185,40,.12); border: 1px solid rgba(242,185,40,.3); color: #fbd34d; border-radius: 999px; padding: 3px 10px; font-size: 12.5px; }
/* Intestazione home tifosi (upload immagine-titolo) */
.header-img { display: flex; gap: 14px; align-items: center; flex-wrap: wrap; }
.hi-preview {
  flex: none; width: 200px; height: 90px; border-radius: 10px; overflow: hidden;
  display: grid; place-items: center; border: 1px solid rgba(255,255,255,.14);
  background: radial-gradient(120% 120% at 50% 0%, #2E4B6E 0%, #7A4A3A 55%, #15151b 100%);
}
.hi-preview.empty { background: #15151b; border-style: dashed; }
.hi-preview img { max-width: 100%; max-height: 100%; object-fit: contain; }
.hi-fallback { font-weight: 900; font-style: italic; text-transform: uppercase; font-size: 15px; color: #fff; letter-spacing: .5px; text-align: center; padding: 0 8px; }
.hi-actions { display: flex; gap: 8px; align-items: center; }
.hi-upload { background: #f2b928; color: #111; border: none; border-radius: 8px; padding: 9px 16px; font-weight: 800; font-size: 14px; cursor: pointer; }
.hi-upload input:disabled { cursor: default; }

/* MVP admin: monitoraggio votazioni */
.mvp-summary { display: flex; flex-wrap: wrap; gap: 10px; margin: 4px 0 8px; }
.mvp-stat { flex: none; background: #15151b; border: 1px solid rgba(255,255,255,.1); border-radius: 12px; padding: 10px 16px; text-align: center; display: flex; flex-direction: column; }
.mvp-stat .num { font-size: 26px; font-weight: 900; color: #f2b928; line-height: 1; font-variant-numeric: tabular-nums; }
.mvp-stat .lbl { font-size: 10.5px; letter-spacing: .5px; color: #94a3b8; text-transform: uppercase; margin-top: 4px; }
.mvp-leader { flex: 1; min-width: 180px; display: flex; align-items: center; gap: 10px; background: linear-gradient(90deg, rgba(242,185,40,.16), rgba(242,185,40,.05)); border: 1px solid rgba(242,185,40,.35); border-radius: 12px; padding: 10px 14px; }
.mvp-leader.empty { background: #15151b; border-color: rgba(255,255,255,.1); opacity: .75; }
.mvp-leader .crown { font-size: 24px; }
.mvp-leader b { display: block; font-size: 15px; font-weight: 900; }
.mvp-leader small { color: #94a3b8; font-size: 12px; }
.mvp-leader__tag { display: block; font-size: 10px; letter-spacing: .5px; text-transform: uppercase; color: #f2b928; font-weight: 800; margin-bottom: 2px; }
.mab-gender { color: #94a3b8; font-weight: 700; font-size: 12px; }
.mvp-a-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(230px, 1fr)); gap: 12px; }
.mvp-a-box { background: #15151b; border: 1px solid rgba(255,255,255,.09); border-radius: 12px; overflow: hidden; }
.mab-head { display: flex; align-items: center; justify-content: space-between; padding: 9px 12px; border-bottom: 1px solid rgba(255,255,255,.08); background: rgba(242,185,40,.06); }
.mab-name { font-weight: 900; font-size: 13px; text-transform: uppercase; letter-spacing: .3px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.mab-total { flex: none; color: #f2b928; font-weight: 800; font-size: 13px; font-variant-numeric: tabular-nums; }
.mab-total small { color: #94a3b8; font-weight: 600; font-size: 10px; }
.mab-row { position: relative; display: flex; align-items: center; justify-content: space-between; gap: 8px; padding: 9px 12px; border-bottom: 1px solid rgba(255,255,255,.04); overflow: hidden; }
.mab-row:last-child { border-bottom: none; }
.mab-row .bar { position: absolute; left: 0; top: 0; bottom: 0; z-index: 0; background: rgba(242,185,40,.12); transition: width .5s cubic-bezier(.22,1,.36,1); }
.mab-row.lead .bar { background: rgba(242,185,40,.24); }
.mab-player { position: relative; z-index: 1; font-size: 13.5px; font-weight: 700; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.mab-row.lead .mab-player { color: #ffe08a; }
.mab-count { position: relative; z-index: 1; flex: none; font-size: 11.5px; font-weight: 800; color: #cbd5e1; font-variant-numeric: tabular-nums; }
.row-match { display: flex; align-items: center; justify-content: space-between; gap: 10px; background: #15151b; border: 1px solid rgba(255,255,255,.09); border-radius: 10px; padding: 10px 12px; }
.row-match.done { opacity: .75; }
.row-match small { color: #94a3b8; }
.danger { background: transparent; border: 1px solid rgba(248,113,113,.4); color: #f87171; border-radius: 7px; padding: 5px 12px; }
.ghost { background: transparent; border: 1px solid rgba(255,255,255,.2); color: #cbd5e1; border-radius: 7px; padding: 6px 12px; }
.ghost.anchor-on { border-color: #f2b928; color: #f2b928; background: rgba(242,185,40,.1); }
.anchor-flag { margin-right: 4px; }
.form-row button.ghost { background: transparent; color: #cbd5e1; } /* batte `.form-row button` (sfondo giallo) */
.start { background: #16a34a; color: #fff; border: none; border-radius: 8px; padding: 8px 16px; font-weight: 800; }
.swatch { display: inline-block; width: 13px; height: 13px; border-radius: 4px; vertical-align: -2px; margin-left: 4px; }
.settings-sub { margin: 18px 0 8px; font-size: 13px; font-weight: 800; letter-spacing: .3px; color: #f2b928; }
.settings-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 360px));
  align-items: start;
  gap: 12px;
}
.settings-grid label {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 6px;
}
.settings-grid input,
.settings-grid select { width: 100%; box-sizing: border-box; }
.settings-grid .settings-wide { grid-column: 1 / -1; width: min(100%, 732px); }
.settings-grid .settings-wide textarea { box-sizing: border-box; resize: vertical; }
.settings-grid label.check { flex-direction: row; align-items: center; gap: 8px; }
.settings-grid label.check input { width: 20px; height: 20px; flex: none; }
.bracket-actions { display: flex; flex-wrap: wrap; gap: 10px; align-items: center; }
.bracket-actions .gen { background: linear-gradient(135deg, #FFD23F, #FF7A18); color: #14110A; font-weight: 900; }
.bracket-actions .gen:disabled { opacity: .7; }
.mod-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 6px; }
.mod-cell { position: relative; aspect-ratio: 1; border-radius: 8px; overflow: hidden; background: #15151b; }
.mod-cell img { width: 100%; height: 100%; object-fit: cover; display: block; }
.mod-del {
  position: absolute; top: 5px; right: 5px; width: 30px; height: 30px; border: none;
  border-radius: 50%; cursor: pointer; font-size: 14px; line-height: 1;
  background: rgba(220,38,38,.9); color: #fff; box-shadow: 0 2px 6px rgba(0,0,0,.4);
}
.logo-row { align-items: center; }
.logo-pick { background: #23232c; border: 1px dashed rgba(255,255,255,.25); border-radius: 8px; padding: 9px 14px; color: #e2e8f0; font-size: 13px; font-weight: 700; cursor: pointer; }
.logo-pick:hover { border-color: #f2b928; }
.logo-preview { display: flex; align-items: center; gap: 10px; }
.logo-preview img { height: 40px; width: auto; max-width: 120px; object-fit: contain; background: #fff; border-radius: 6px; padding: 3px; }
.logo-preview .danger { font-size: 12px; padding: 6px 10px; }
.sponsor-line { display: flex; align-items: center; gap: 8px; min-width: 0; }
.logo-thumb { height: 26px; width: auto; max-width: 70px; object-fit: contain; background: #fff; border-radius: 5px; padding: 2px; flex: none; }

/* Shop torneo */
.shop-editor {
  display: grid;
  grid-template-columns: minmax(280px, 420px) minmax(320px, 1fr);
  gap: 18px;
  align-items: start;
  background: #101017;
  border: 1px solid rgba(255,255,255,.1);
  border-radius: 14px;
  padding: 16px;
}
.shop-image-editor { display: flex; flex-direction: column; gap: 10px; }
.shop-image-preview {
  width: 100%;
  aspect-ratio: 16 / 9;
  overflow: hidden;
  border-radius: 12px;
  border: 1px solid rgba(255,255,255,.14);
  background: #15151b;
  display: grid;
  place-items: center;
  color: #64748b;
  font-weight: 700;
}
.shop-image-preview.empty { border-style: dashed; }
.shop-image-preview img { width: 100%; height: 100%; object-fit: cover; display: block; }
.shop-image-actions { display: flex; align-items: center; gap: 8px; }
.shop-upload {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: #f2b928;
  color: #111;
  border-radius: 8px;
  padding: 9px 16px;
  font-weight: 800;
  cursor: pointer;
}
.shop-main-fields { display: grid; grid-template-columns: minmax(0, 1fr) minmax(150px, 220px); gap: 12px; }
.shop-main-fields label {
  display: flex;
  flex-direction: column;
  gap: 6px;
  color: #e2e8f0;
  font-weight: 700;
}
.shop-main-fields label > span,
.shop-extras span { color: #64748b; font-size: 12px; font-weight: 500; }
.shop-main-fields input,
.shop-main-fields textarea,
.shop-extra-row input {
  width: 100%;
  box-sizing: border-box;
  background: #15151b;
  border: 1px solid rgba(255,255,255,.14);
  border-radius: 8px;
  padding: 10px 11px;
  color: #fff;
  font: inherit;
}
.shop-description { grid-column: 1 / -1; }
.price-field {
  display: flex;
  align-items: center;
  overflow: hidden;
  background: #15151b;
  border: 1px solid rgba(255,255,255,.14);
  border-radius: 8px;
}
.price-field > span { padding-left: 11px; color: #f2b928; font-weight: 900; }
.price-field input { border: none; background: transparent; min-width: 0; }
.shop-extras {
  grid-column: 1 / -1;
  display: flex;
  flex-direction: column;
  gap: 8px;
  border-top: 1px solid rgba(255,255,255,.08);
  padding-top: 14px;
}
.shop-extras-head { display: flex; align-items: center; justify-content: space-between; gap: 10px; }
.shop-extra-row { display: grid; grid-template-columns: minmax(180px, 1fr) minmax(130px, 220px) auto; gap: 8px; align-items: center; }
.shop-extra-row .danger { width: 36px; height: 36px; padding: 0; font-size: 20px; }
.shop-save {
  grid-column: 1 / -1;
  justify-self: start;
  border: none;
  border-radius: 9px;
  padding: 11px 20px;
  background: #f2b928;
  color: #111;
  font-weight: 900;
}
.shop-save:disabled { opacity: .55; cursor: default; }
.shop-admin-list { display: flex; flex-direction: column; gap: 10px; }
.shop-admin-product {
  display: grid;
  grid-template-columns: 180px minmax(0, 1fr) auto auto;
  align-items: center;
  gap: 14px;
  padding: 10px;
  border: 1px solid rgba(255,255,255,.09);
  border-radius: 12px;
  background: #15151b;
}
.shop-admin-product > img { width: 180px; aspect-ratio: 16 / 9; object-fit: cover; border-radius: 9px; }
.shop-admin-copy { min-width: 0; }
.shop-admin-copy > b { display: block; font-size: 15px; }
.shop-admin-copy p { margin: 4px 0 0; color: #94a3b8; font-size: 13px; white-space: pre-wrap; }
.shop-untitled { color: #64748b; font-style: italic; }
.shop-admin-extras { display: flex; flex-wrap: wrap; gap: 6px; margin-top: 7px; }
.shop-admin-extras span { padding: 3px 7px; border-radius: 999px; background: rgba(242,185,40,.1); color: #fbd34d; }
.shop-admin-price { color: #f2b928; font-size: 17px; white-space: nowrap; }
.shop-reservations-head {
  display: flex; align-items: flex-end; justify-content: space-between; gap: 12px;
  margin-top: 16px; padding-top: 12px; border-top: 1px solid rgba(255,255,255,.1);
}
.shop-reservations-head h3 { margin: 0 0 3px; }
.shop-reservations-head p { margin: 0; }
.shop-reservation-list { display: flex; flex-direction: column; gap: 9px; }
.shop-reservation {
  display: grid; grid-template-columns: 120px minmax(180px, 1fr) minmax(180px, 1fr) auto;
  align-items: center; gap: 13px; padding: 10px;
  background: linear-gradient(90deg, rgba(242,185,40,.09), #15151b 28%);
  border: 1px solid rgba(242,185,40,.22); border-radius: 12px;
}
.shop-reservation > img { width: 120px; aspect-ratio: 16 / 9; object-fit: cover; border-radius: 8px; }
.shop-reservation-product,
.shop-reservation-customer { display: flex; min-width: 0; flex-direction: column; gap: 3px; }
.shop-reservation-product > b,
.shop-reservation-customer > b { overflow-wrap: anywhere; }
.shop-reservation-product > span { color: #fbd34d; font-size: 12px; }
.shop-reservation-product > small { color: #64748b; font-size: 11px; margin-top: 2px; }
.shop-reservation-customer a {
  width: fit-content; color: #67e8f9; font-weight: 800; text-decoration: none;
  padding: 4px 8px; border-radius: 7px; background: rgba(34,211,238,.09);
}
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

@media (max-width: 600px) {
  .settings-grid { grid-template-columns: 1fr; }
  .shop-editor { grid-template-columns: 1fr; padding: 12px; }
  .shop-main-fields { grid-template-columns: 1fr; }
  .shop-description { grid-column: auto; }
  .shop-extras { grid-column: auto; }
  .shop-extra-row { grid-template-columns: minmax(0, 1fr) 120px auto; }
  .shop-save { grid-column: auto; width: 100%; }
  .shop-admin-product { grid-template-columns: 110px minmax(0, 1fr) auto; gap: 9px; }
  .shop-admin-product > img { width: 110px; }
  .shop-admin-product > .danger { grid-column: 1 / -1; justify-self: end; }
  .shop-admin-price { font-size: 14px; }
  .shop-reservations-head { align-items: flex-start; }
  .shop-reservation { grid-template-columns: 90px minmax(0, 1fr) auto; gap: 9px; }
  .shop-reservation > img { width: 90px; }
  .shop-reservation-customer { grid-column: 1 / 3; }
}
</style>
