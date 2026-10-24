<script setup>
import { computed, nextTick, onMounted, onUnmounted, reactive, ref } from 'vue'
import ClubTileIconUpload from '../components/ClubTileIconUpload.vue'
import ClubMediaField from '../components/ClubMediaField.vue'
import { clubHomeTiles } from '../clubHub'
import { CLUB_FONTS, clubTypography } from '../clubTypography'

const props = defineProps({ slug: { type: String, required: true }, customDomain: Boolean })
const publicPath = computed(() => props.customDomain ? '/' : `/club/${encodeURIComponent(props.slug)}`)
const authenticated = ref(false)
const checking = ref(true)
const busy = ref(false)
const error = ref('')
const notice = ref('')
const activeTab = ref('dashboard')
const overview = ref({ club: {}, teams: [], posts: [], gallery: [], matches: [], sponsors: [], counts: {} })
const playersByTeam = ref({})
const login = reactive({ username: `club-${props.slug}`, password: '' })

const settings = reactive({ contacts: [], tileIcons: {}, titleFont: 'Inter', bodyFont: 'Inter' })
const imageUploads = ref(0)
const processing = computed(() => busy.value || imageUploads.value > 0)
const teamForm = reactive(emptyTeam())
const postForm = reactive(emptyPost())
const galleryForm = reactive(emptyGallery())
const matchForm = reactive(emptyMatch())
const sponsorForm = reactive(emptySponsor())
const rosterTeamId = ref(0)
const rosterPlayers = ref([])
const editorPanel = ref(null)
const deleteDialog = ref(null)
const pendingDelete = ref(null)
const baselines = reactive({ settings: '', team: '', post: '', gallery: '', match: '', sponsor: '', roster: '' })
const collectionTabs = ['teams', 'posts', 'gallery', 'matches', 'sponsors']
const filters = reactive(Object.fromEntries(collectionTabs.map(tab => [tab, { query: '', teamId: 0, state: '' }])))
let playerKey = 0
let setKey = 0

const tabs = [
  ['dashboard','Panoramica'], ['settings','Homepage'], ['teams','Squadre e roster'],
  ['posts','Annunci e news'], ['gallery','Photo gallery'], ['matches','Calendario'],
  ['sponsors','Sponsor'], ['contacts','Contatti'],
]
const teams = computed(() => overview.value.teams ?? [])
const editableTiles = computed(() => clubHomeTiles({ ...overview.value, club: settings, teams: teams.value.filter(team => team.active) }))
const typography = computed(() => clubTypography(settings))
const descriptions = {
  dashboard: 'Tutti i contenuti e le attività del tuo Club Hub, in un unico spazio.',
  settings: 'Personalizza identità, immagini e stile del sito del club.',
  teams: 'Gestisci le schede delle squadre e i dati di ogni atleta.',
  posts: 'Crea, aggiorna e organizza le comunicazioni della società.',
  gallery: 'Organizza le fotografie con didascalie, date e squadre.',
  matches: 'Pianifica gli incontri e aggiorna risultati e parziali.',
  sponsors: 'Presenta i partner che sostengono il club e collega i loro siti web.',
  contacts: 'Mantieni aggiornati i riferimenti di uffici e persone.',
}
const statusLabels = { scheduled: 'Programmata', live: 'In diretta', completed: 'Conclusa' }
const filteredTeams = computed(() => filterItems('teams', teams.value, team => [team.name, team.category, team.championship]))
const filteredPosts = computed(() => filterItems('posts', overview.value.posts ?? [], post => [post.title, post.excerpt, teamName(post.teamId)]))
const filteredGallery = computed(() => filterItems('gallery', overview.value.gallery ?? [], photo => [photo.title, photo.caption, teamName(photo.teamId)]))
const filteredMatches = computed(() => filterItems('matches', overview.value.matches ?? [], match => [match.opponent, match.competition, match.location, teamName(match.teamId)]))
const filteredSponsors = computed(() => filterItems('sponsors', overview.value.sponsors ?? [], sponsor => [sponsor.name, sponsor.website, sponsor.description]))
const resultCount = computed(() => ({ teams: filteredTeams.value.length, posts: filteredPosts.value.length, gallery: filteredGallery.value.length, matches: filteredMatches.value.length, sponsors: filteredSponsors.value.length })[activeTab.value])
const hasUnsavedChanges = computed(() => Boolean(
  imageUploads.value ||
  (baselines.settings && JSON.stringify(settings) !== baselines.settings) ||
  ['team', 'post', 'gallery', 'match', 'sponsor'].some(kind => isDirty(kind)) ||
  (rosterTeamId.value && JSON.stringify(rosterPlayers.value) !== baselines.roster)))

function emptyTeam() { return { id: 0, name: '', slug: '', category: '', gender: 'mixed', championship: '', description: '', logoUrl: '', heroImageUrl: '', position: 0, active: true } }
function emptyPost() { return { id: 0, kind: 'announcement', title: '', excerpt: '', body: '', imageUrl: '', teamId: 0, pinned: false, active: true, publishedAt: '' } }
function emptyGallery() { return { id: 0, title: '', imageUrl: '', caption: '', eventDate: '', teamId: 0, active: true, position: 0 } }
function emptyMatch() { return { id: 0, teamId: 0, opponent: '', competition: '', roundLabel: '', scheduledAt: '', location: '', home: true, status: 'scheduled', scoreFor: 0, scoreAgainst: 0, sets: [] } }
function emptySponsor() { return { id: 0, name: '', logoUrl: '', website: '', description: '', position: 0, active: true } }
function editor(kind) { return { team: [teamForm, emptyTeam], post: [postForm, emptyPost], gallery: [galleryForm, emptyGallery], match: [matchForm, emptyMatch], sponsor: [sponsorForm, emptySponsor] }[kind] }
function isDirty(kind) { const [form, empty] = editor(kind); return JSON.stringify(form) !== (baselines[kind] || JSON.stringify(empty())) }
function resetEditor(kind) { const [form, empty] = editor(kind); resetObject(form, empty()); baselines[kind] = JSON.stringify(form) }
function clearEditor(kind) { if (busy.value || imageUploads.value || (isDirty(kind) && !confirm('Scartare le modifiche non salvate di questa scheda?'))) return; resetEditor(kind) }
async function editRecord(kind, item) {
  if (busy.value || imageUploads.value || (isDirty(kind) && !confirm('Scartare le modifiche non salvate e aprire questa scheda?'))) return
  const [form, empty] = editor(kind)
  const data = { ...empty(), ...item }
  if (kind === 'post') data.publishedAt = (data.publishedAt || '').slice(0, 10)
  if (kind === 'gallery') data.eventDate = (data.eventDate || '').slice(0, 10)
  if (kind === 'match') {
    data.scheduledAt = localDateTime(data.scheduledAt)
    // Keep the API representation out of the editor; existing partials remain editable.
    const partials = JSON.parse(data.setsJson || '[]')
    if (!Array.isArray(partials) || partials.some(value => !/^\d+\s*[-–:]\s*\d+$/.test(String(value)))) throw new Error('invalid_sets')
    data.sets = partials.map(value => { const [scoreFor, scoreAgainst] = String(value).split(/[-–:]/).map(Number); return { key: ++setKey, scoreFor, scoreAgainst } })
    delete data.setsJson
  }
  resetObject(form, data)
  baselines[kind] = JSON.stringify(form)
  await nextTick()
  editorPanel.value?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  editorPanel.value?.querySelector('input, select')?.focus({ preventScroll: true })
}
function editTeam(item) { return editRecord('team', item) }
function editPost(item) { return editRecord('post', item) }
function editGallery(item) { return editRecord('gallery', item) }
function editSponsor(item) { return editRecord('sponsor', item) }
async function editMatch(item) { try { await editRecord('match', item) } catch { error.value = 'I parziali di questa partita non sono leggibili. Verifica i dati esistenti prima di modificarli.' } }
function localDateTime(value) {
  if (!value) return ''
  if (!/(Z|[+-]\d{2}:\d{2})$/.test(value)) return value.slice(0, 16)
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '' : `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}T${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
}
function filterItems(tab, items, values) {
  const filter = filters[tab]
  const query = filter.query.trim().toLocaleLowerCase('it')
  return items.filter(item => (!query || values(item).some(value => String(value || '').toLocaleLowerCase('it').includes(query))) &&
    (!filter.teamId || Number(item.teamId) === Number(filter.teamId)) &&
    (!filter.state || (tab === 'matches' ? item.status === filter.state : String(item.active !== false) === filter.state)))
}
function resetFilters() { Object.assign(filters[activeTab.value], { query: '', teamId: 0, state: '' }) }
function resetObject(target, source) { Object.keys(target).forEach(key => delete target[key]); Object.assign(target, source) }
function message(value) { notice.value = value; setTimeout(() => { if (notice.value === value) notice.value = '' }, 2500) }

async function request(path, options = {}) {
  const response = await fetch(`/api/v1/club-admin/${encodeURIComponent(props.slug)}${path}`, {
    credentials: 'include', ...options,
    headers: options.body ? { 'Content-Type': 'application/json', ...(options.headers || {}) } : options.headers,
  })
  if (response.status === 401) { authenticated.value = false; throw new Error('unauthorized') }
  const data = await response.json().catch(() => ({}))
  if (!response.ok) throw new Error(data.error || `HTTP ${response.status}`)
  return data
}

async function loadOverview({ resetSettings = true } = {}) {
  checking.value = !authenticated.value
  try {
    overview.value = await request('/overview')
    authenticated.value = true
    if (resetSettings) resetObject(settings, {
      ...overview.value.club,
      titleFont: overview.value.club.titleFont || 'Inter',
      bodyFont: overview.value.club.bodyFont || 'Inter',
      contacts: (overview.value.club.contacts ?? []).map(contact => ({ ...contact })),
      tileIcons: { ...(overview.value.club.tileIcons ?? {}) },
    })
    if (resetSettings) baselines.settings = JSON.stringify(settings)
    await loadTeams()
  } catch (e) { if (e.message !== 'unauthorized') error.value = 'Impossibile caricare il CMS.' }
  finally { checking.value = false }
}
async function loadTeams() { const data = await request('/teams'); overview.value.teams = data.teams ?? []; playersByTeam.value = data.players ?? {} }
async function loginSubmit() {
  if (busy.value) return
  busy.value = true
  error.value = ''
  try {
    await request('/login', { method: 'POST', body: JSON.stringify(login) })
    authenticated.value = true
    await loadOverview({ resetSettings: !baselines.settings })
  } catch { error.value = 'Credenziali non valide.' }
  finally { busy.value = false }
}
async function logout() {
  if (processing.value || (hasUnsavedChanges.value && !confirm('Hai modifiche non salvate. Uscire dal CMS e scartarle?'))) return
  busy.value = true
  try {
    await request('/logout', { method: 'POST' })
    authenticated.value = false
    login.password = ''
    for (const kind of ['team', 'post', 'gallery', 'match', 'sponsor']) resetEditor(kind)
    rosterTeamId.value = 0
    baselines.settings = ''
  } catch { error.value = 'Impossibile uscire dal CMS. Riprova.' }
  finally { busy.value = false }
}

async function saveSettings() {
  if (busy.value || imageUploads.value) return
  const body = JSON.stringify(settings)
  if (new TextEncoder().encode(body).length > 2 * 1024 * 1024) return invalid('Le immagini della homepage superano lo spazio disponibile. Riduci alcune immagini o usa collegamenti a immagini già ospitate.')
  busy.value = true
  error.value = ''
  try {
    await request('/settings', { method: 'PUT', body })
    baselines.settings = JSON.stringify(settings)
    message(activeTab.value === 'contacts' ? 'Contatti aggiornati.' : 'Homepage aggiornata.')
    await loadOverview({ resetSettings: false })
  } catch (e) {
    const messages = {
      invalid_contact: 'Inserisci un nome per ogni contatto e verifica la lunghezza dei campi.',
      invalid_contact_email: 'Verifica gli indirizzi email dei contatti.',
      invalid_contact_website: 'I siti web dei contatti devono iniziare con https:// o http://.',
      too_many_contacts: 'Puoi inserire al massimo 50 contatti.',
      tile_icon_too_large: 'Un’icona è troppo pesante. Carica un’immagine più piccola.',
      invalid_tile_icon: 'Una delle icone non è valida. Prova a caricarla nuovamente.',
      invalid_font: 'Scegli i font dall’elenco disponibile.',
    }
    error.value = messages[e.message] || 'Salvataggio fallito. Riprova.'
  } finally { busy.value = false }
}
function addContact() {
  settings.contacts.push({ name: '', role: '', phone: '', email: '', address: '', website: '' })
}
function moveContact(index, direction) {
  moveItem(settings.contacts, index, direction)
}
function moveItem(items, index, direction) { const target = index + direction; if (busy.value || target < 0 || target >= items.length) return; const [item] = items.splice(index, 1); items.splice(target, 0, item) }
function setTileIcon(id, value) {
  if (value) settings.tileIcons[id] = value
  else delete settings.tileIcons[id]
}
function trackImageUpload(loading) { imageUploads.value = Math.max(0, imageUploads.value + (loading ? 1 : -1)) }
async function saveRecord(kind, path, payload, success) {
  if (busy.value || imageUploads.value) return
  busy.value = true
  error.value = ''
  try {
    await request(payload.id ? `${path}/${payload.id}` : path, { method: payload.id ? 'PUT' : 'POST', body: JSON.stringify(payload) })
    resetEditor(kind)
    message(success)
    await loadOverview({ resetSettings: false })
  } catch (e) {
    const sponsorErrors = {
      invalid_sponsor_name: 'Inserisci il nome dello sponsor (massimo 200 caratteri).',
      invalid_sponsor_logo: 'Inserisci un logo valido, fino a 300 KB. Prova a caricarlo nuovamente.',
      invalid_sponsor_website: 'Inserisci un sito web valido che inizi con https:// o http://.',
      invalid_sponsor_description: 'La descrizione dello sponsor può contenere al massimo 3000 caratteri.',
      invalid_sponsor_position: 'L’ordine dello sponsor deve essere un numero intero tra 0 e 1000000.',
    }
    error.value = e.message === 'unauthorized' ? 'La sessione è scaduta. Accedi nuovamente per continuare.' : (kind === 'sponsor' && sponsorErrors[e.message]) || 'Salvataggio non riuscito. I dati inseriti sono conservati: verifica i campi e riprova.'
  }
  finally { busy.value = false }
}
function invalid(message) { error.value = message; return false }
function saveTeam() {
  if (!teamForm.name.trim()) return invalid('Inserisci il nome della squadra.')
  return saveRecord('team', '/teams', { ...teamForm, name: teamForm.name.trim() }, 'Squadra salvata.')
}
function savePost() {
  if (!postForm.title.trim()) return invalid('Inserisci il titolo del contenuto.')
  return saveRecord('post', '/posts', { ...postForm, title: postForm.title.trim(), teamId: Number(postForm.teamId) || 0 }, 'Contenuto salvato.')
}
function saveGallery() {
  if (!galleryForm.imageUrl.trim()) return invalid('Inserisci il collegamento alla fotografia.')
  return saveRecord('gallery', '/gallery', { ...galleryForm, imageUrl: galleryForm.imageUrl.trim(), teamId: Number(galleryForm.teamId) || 0 }, 'Fotografia salvata.')
}
function saveSponsor() {
  if (processing.value) return
  const name = sponsorForm.name.trim()
  const logoUrl = sponsorForm.logoUrl.trim()
  const website = sponsorForm.website.trim()
  const description = sponsorForm.description.trim()
  if (!name || name.length > 200) return invalid('Inserisci il nome dello sponsor (massimo 200 caratteri).')
  if (!logoUrl) return invalid('Carica il logo dello sponsor o inserisci il collegamento a un’immagine.')
  try {
    const url = new URL(website)
    if (!/^https?:\/\//i.test(website) || !['https:', 'http:'].includes(url.protocol) || !url.hostname || url.username || url.password || /[\s\\\u0000-\u001f\u007f]/.test(website) || website.length > 2000) throw new Error('invalid_url')
  } catch { return invalid('Inserisci un sito web valido che inizi con https:// o http://.') }
  if (description.length > 3000) return invalid('La descrizione dello sponsor può contenere al massimo 3000 caratteri.')
  if (!validScore(sponsorForm.position) || Number(sponsorForm.position) > 1000000) return invalid('L’ordine dello sponsor deve essere un numero intero tra 0 e 1000000.')
  return saveRecord('sponsor', '/sponsors', { ...sponsorForm, name, logoUrl, website, description, position: Number(sponsorForm.position) }, 'Sponsor salvato.')
}
function addSet() { matchForm.sets.push({ key: ++setKey, scoreFor: '', scoreAgainst: '' }) }
function validScore(value) { return value !== '' && value !== null && Number.isSafeInteger(Number(value)) && Number(value) >= 0 }
function saveMatch() {
  if (!matchForm.teamId || !matchForm.opponent.trim() || !matchForm.scheduledAt) return invalid('Completa squadra, avversario e data della partita.')
  if (!validScore(matchForm.scoreFor) || !validScore(matchForm.scoreAgainst)) return invalid('I risultati devono essere numeri interi maggiori o uguali a zero.')
  if (matchForm.sets.some(set => !validScore(set.scoreFor) || !validScore(set.scoreAgainst))) return invalid('Completa entrambi i punteggi di ogni parziale con numeri interi maggiori o uguali a zero.')
  const { sets, ...payload } = matchForm
  return saveRecord('match', '/matches', { ...payload, opponent: payload.opponent.trim(), teamId: Number(payload.teamId), scoreFor: Number(payload.scoreFor), scoreAgainst: Number(payload.scoreAgainst), setsJson: JSON.stringify(sets.map(set => `${Number(set.scoreFor)}-${Number(set.scoreAgainst)}`)) }, 'Partita salvata.')
}
function openRoster(team) {
  if (busy.value || imageUploads.value || (rosterTeamId.value && JSON.stringify(rosterPlayers.value) !== baselines.roster && !confirm('Scartare le modifiche non salvate al roster?'))) return
  rosterTeamId.value = team.id
  rosterPlayers.value = (playersByTeam.value[String(team.id)] ?? []).map(player => ({ ...player, key: ++playerKey }))
  baselines.roster = JSON.stringify(rosterPlayers.value)
}
function closeRoster() {
  if (busy.value || imageUploads.value || (JSON.stringify(rosterPlayers.value) !== baselines.roster && !confirm('Scartare le modifiche non salvate al roster?'))) return
  rosterTeamId.value = 0
}
function addPlayer() { rosterPlayers.value.push({ key: ++playerKey, firstName: '', lastName: '', jerseyNumber: '', role: '', imageUrl: '' }) }
async function saveRoster() {
  if (busy.value || imageUploads.value || !rosterTeamId.value) return
  if (rosterPlayers.value.some(player => !player.firstName.trim() && !player.lastName.trim())) return invalid('Inserisci almeno il nome o il cognome di ogni atleta, oppure rimuovi la scheda vuota.')
  if (rosterPlayers.value.some(player => player.jerseyNumber !== '' && !validScore(player.jerseyNumber))) return invalid('Il numero di maglia deve essere un numero intero maggiore o uguale a zero.')
  busy.value = true
  error.value = ''
  try {
    const players = rosterPlayers.value.map((player, position) => ({ firstName: player.firstName.trim(), lastName: player.lastName.trim(), jerseyNumber: Number(player.jerseyNumber) || 0, role: player.role.trim(), imageUrl: player.imageUrl.trim(), position }))
    await request(`/teams/${rosterTeamId.value}/players`, { method: 'PUT', body: JSON.stringify({ players }) })
    baselines.roster = JSON.stringify(rosterPlayers.value)
    message('Roster aggiornato.')
    await loadOverview({ resetSettings: false })
  } catch { error.value = 'Impossibile salvare il roster. Le modifiche sono conservate: riprova.' }
  finally { busy.value = false }
}
async function askDelete(kind, item) {
  if (processing.value) return
  pendingDelete.value = { kind, item }
  await nextTick()
  deleteDialog.value?.showModal()
}
function cancelDelete() { if (busy.value) return; deleteDialog.value?.close(); pendingDelete.value = null }
async function confirmDelete() {
  if (busy.value || !pendingDelete.value) return
  const { kind, item } = pendingDelete.value
  busy.value = true
  error.value = ''
  try {
    await request(`/${kind}/${item.id}`, { method: 'DELETE' })
    const editorKind = { teams: 'team', posts: 'post', gallery: 'gallery', matches: 'match', sponsors: 'sponsor' }[kind]
    if (editor(editorKind)[0].id === item.id) resetEditor(editorKind)
    if (kind === 'teams' && rosterTeamId.value === item.id) rosterTeamId.value = 0
    deleteDialog.value?.close()
    pendingDelete.value = null
    message('Elemento eliminato.')
    await loadOverview({ resetSettings: false })
  } catch { error.value = 'Eliminazione non riuscita. Riprova.'; deleteDialog.value?.close(); pendingDelete.value = null }
  finally { busy.value = false }
}
function teamName(id){return teams.value.find(team=>team.id===id)?.name||'Tutta la società'}
function formatDate(value){if(!value)return '—';const d=new Date(value);return Number.isNaN(d.getTime())?value:new Intl.DateTimeFormat('it-IT',{dateStyle:'medium',timeStyle:value.includes('T')?'short':undefined}).format(d)}

function beforeUnload(event) { if (hasUnsavedChanges.value) { event.preventDefault(); event.returnValue = '' } }
onMounted(() => { loadOverview(); window.addEventListener('beforeunload', beforeUnload) })
onUnmounted(() => { window.removeEventListener('beforeunload', beforeUnload) })
</script>

<template>
  <main class="club-admin">
    <link v-for="stylesheet in typography.stylesheets" :key="stylesheet" rel="stylesheet" :href="stylesheet" />
    <section v-if="checking" class="center">Caricamento CMS Club…</section>
    <section v-else-if="!authenticated" class="login-shell">
      <div class="login-card"><p class="eyebrow">ARENABOOSTX CLUB</p><h1>Gestisci il tuo Club Hub</h1><p>Homepage, squadre, roster, annunci, fotografie, calendario e sponsor.</p><form @submit.prevent="loginSubmit"><label>Username<input v-model="login.username" autocomplete="username" /></label><label>Password<input v-model="login.password" type="password" autocomplete="current-password" /></label><button :disabled="processing">{{busy?'Accesso…':'Entra nel CMS'}}</button></form><p v-if="error" class="error">{{error}}</p></div>
    </section>
    <template v-else>
      <aside>
        <a class="admin-brand" :href="publicPath" target="_blank"><span>AX</span><div><strong>Club CMS</strong><small>{{overview.club.name}}</small></div></a>
        <nav aria-label="Sezioni CMS"><button v-for="tab in tabs" :key="tab[0]" :disabled="processing" :class="{active:activeTab===tab[0]}" :aria-current="activeTab===tab[0] ? 'page' : undefined" @click="activeTab=tab[0]">{{tab[1]}}</button></nav>
        <div class="aside-bottom"><p v-if="hasUnsavedChanges" class="draft-indicator">● Modifiche da salvare</p><a :href="publicPath" target="_blank" rel="noopener">Apri sito pubblico ↗</a><button :disabled="processing" @click="logout">Esci</button></div>
      </aside>
      <section class="workspace">
        <header><div><p class="eyebrow">CLUB WORKSPACE · {{overview.club.seasonLabel || 'Gestione contenuti'}}</p><h1>{{tabs.find(tab=>tab[0]===activeTab)?.[1]}}</h1><p class="page-description">{{ descriptions[activeTab] }}</p></div><a class="public-link" :href="publicPath" target="_blank" rel="noopener">Apri Club Hub ↗</a></header>
        <p v-if="notice" class="notice" role="status">{{notice}}</p><p v-if="error" class="error error-banner" role="alert">{{error}}<button type="button" aria-label="Chiudi messaggio di errore" @click="error=''">×</button></p>
        <p v-if="imageUploads" class="processing-banner" role="status">Ottimizzazione dell’immagine in corso. Attendi prima di salvare.</p>

        <section v-if="collectionTabs.includes(activeTab) && (activeTab !== 'teams' || !rosterTeamId)" class="collection-toolbar" aria-label="Ricerca e filtri">
          <label class="search-field">Cerca {{ activeTab === 'teams' ? 'squadre' : activeTab === 'posts' ? 'contenuti' : activeTab === 'gallery' ? 'fotografie' : activeTab === 'sponsors' ? 'sponsor' : 'partite' }}<input v-model="filters[activeTab].query" type="search" placeholder="Digita per cercare…" /></label>
          <label v-if="activeTab !== 'teams' && activeTab !== 'sponsors'">Squadra<select v-model.number="filters[activeTab].teamId"><option :value="0">Tutte le squadre</option><option v-for="team in teams" :key="team.id" :value="team.id">{{team.name}}</option></select></label>
          <label>Stato<select v-model="filters[activeTab].state"><option value="">Tutti gli stati</option><template v-if="activeTab === 'matches'"><option v-for="(label, value) in statusLabels" :key="value" :value="value">{{ label }}</option></template><template v-else><option value="true">{{ activeTab === 'sponsors' ? 'Pubblicato' : 'Visibile' }}</option><option value="false">{{ activeTab === 'sponsors' ? 'Bozza' : 'Nascosto' }}</option></template></select></label>
          <div class="filter-summary"><span aria-live="polite">{{ resultCount }} risultati</span><button v-if="filters[activeTab].query || filters[activeTab].teamId || filters[activeTab].state" type="button" @click="resetFilters">Azzera filtri</button></div>
        </section>

        <div v-if="activeTab==='dashboard'">
          <div class="stats"><article v-for="entry in [['Squadre',overview.counts.teams],['Atleti',overview.counts.players],['Contenuti',overview.counts.posts],['Foto',overview.counts.photos],['Partite',overview.counts.matches],['Sponsor',overview.counts.sponsors]]" :key="entry[0]"><span>{{entry[0]}}</span><strong>{{entry[1]||0}}</strong></article></div>
          <section class="card welcome"><div><p class="eyebrow">IL TUO HUB ANNUALE</p><h2>{{overview.club.name}}</h2><p>{{overview.club.tagline||'Completa la homepage e inizia a pubblicare contenuti.'}}</p></div><a :href="publicPath" target="_blank">Visualizza sito</a></section>
          <section class="card"><h2>Avvio rapido</h2><div class="quick"><button @click="activeTab='settings'">1. Personalizza homepage</button><button @click="activeTab='teams'">2. Completa squadre e roster</button><button @click="activeTab='matches'">3. Inserisci calendario</button><button @click="activeTab='posts'">4. Pubblica un annuncio</button></div></section>
        </div>

        <form v-else-if="activeTab==='settings'" class="card settings-editor" :aria-busy="busy" @submit.prevent="saveSettings">
          <div class="editor-heading"><div><p class="eyebrow">CONFIGURAZIONE</p><h2>La tua homepage</h2><p class="form-hint">I campi contrassegnati con * sono obbligatori. Le modifiche vengono pubblicate al salvataggio.</p></div></div>
          <fieldset class="form-body" :disabled="processing">
          <fieldset class="typography-editor">
            <legend>Font della home e delle modali</legend>
            <p class="form-hint">Scegli separatamente il font dei titoli e quello dei testi, poi premi Salva homepage.</p>
            <div class="form-grid">
              <label for="club-title-font">Font titoli<select id="club-title-font" v-model="settings.titleFont" class="font-select"><option v-for="font in CLUB_FONTS" :key="font.name" :value="font.name" :disabled="font.disabled">{{ font.name }}{{ font.disabled ? ' (da attivare)' : '' }}</option></select></label>
              <label for="club-body-font">Font testi<select id="club-body-font" v-model="settings.bodyFont" class="font-select"><option v-for="font in CLUB_FONTS" :key="font.name" :value="font.name" :disabled="font.disabled">{{ font.name }}{{ font.disabled ? ' (da attivare)' : '' }}</option></select></label>
            </div>
            <p v-if="CLUB_FONTS.find(font => font.name === 'ENDURA')?.disabled" class="form-hint">ENDURA sarà disponibile dopo l’attivazione del webfont.</p>
            <div class="typography-preview" :style="typography.style">
              <small>Anteprima</small>
              <h3>{{ settings.name || 'Il tuo club' }}: una sola passione</h3>
              <p>Notizie, squadre e appuntamenti della società. Segui il club e scopri tutte le novità della stagione.</p>
            </div>
          </fieldset>
          <fieldset class="form-section"><legend>Identità e homepage</legend>
          <div class="form-grid"><label>Nome del club *<input v-model="settings.name" required maxlength="200" /></label><label>Città<input v-model="settings.city" placeholder="Es. Bologna" /></label><label class="wide">Claim<input v-model="settings.tagline" placeholder="Il messaggio che racconta il tuo club" /></label><label>Stagione<input v-model="settings.seasonLabel" placeholder="Es. 2026 / 2027" /></label><label>Colore primario<input v-model="settings.primaryColor" type="color" /></label><label>Colore accento<input v-model="settings.secondaryColor" type="color" /></label><label class="check"><input v-model="settings.active" type="checkbox" /> Sito pubblico attivo</label>
            <ClubMediaField v-model="settings.logoUrl" class="wide" label="Logo del club" :max-side="512" @uploading="trackImageUpload" />
            <ClubMediaField v-model="settings.heroImageUrl" class="wide" label="Copertina della homepage" @uploading="trackImageUpload" />
          </div>
          </fieldset>
          <fieldset class="form-section"><legend>Icone della homepage</legend>
          <p class="form-hint">Carica un’immagine per ogni tile della homepage. Le immagini vengono adattate automaticamente; premi Salva homepage per pubblicarle.</p>
          <div class="tile-icons-grid">
            <ClubTileIconUpload v-for="tile in editableTiles" :key="tile.id" :model-value="settings.tileIcons[tile.id] || ''" :label="tile.label" :fallback-image="tile.team?.logoUrl || ''" :fallback-icon="tile.icon" @update:model-value="setTileIcon(tile.id, $event)" @uploading="trackImageUpload" />
          </div>
          </fieldset></fieldset>
          <footer class="editor-footer"><span>{{ imageUploads ? 'Elaborazione immagini in corso…' : 'Le modifiche saranno visibili sul sito pubblico.' }}</span><button class="primary" :disabled="processing">{{ busy ? 'Salvataggio…' : 'Salva homepage' }}</button></footer>
        </form>

        <form v-else-if="activeTab==='contacts'" class="card contacts-editor" :aria-busy="busy" @submit.prevent="saveSettings">
          <h2>Contatti della società</h2>
          <p class="form-hint">Questi contatti compaiono nella modale Contatti del sito pubblico, nell’ordine scelto qui.</p>
          <fieldset class="form-body" :disabled="processing">
          <p v-if="!settings.contacts.length" class="empty-state">Nessun contatto inserito. Aggiungi il primo contatto della società.</p>
          <fieldset v-for="(contact, index) in settings.contacts" :key="index" class="contact-editor">
            <legend>Contatto {{ index + 1 }}</legend>
            <div class="contact-actions">
              <button type="button" :disabled="index === 0 || busy" :aria-label="`Sposta in alto il contatto ${index + 1}`" @click="moveContact(index, -1)">↑</button>
              <button type="button" :disabled="index === settings.contacts.length - 1 || busy" :aria-label="`Sposta in basso il contatto ${index + 1}`" @click="moveContact(index, 1)">↓</button>
              <button class="danger" type="button" :disabled="processing" :aria-label="`Rimuovi il contatto ${index + 1}`" @click="settings.contacts.splice(index, 1)">Rimuovi</button>
            </div>
            <div class="form-grid">
              <label>Nome o ufficio *<input v-model="contact.name" required maxlength="200" placeholder="Segreteria del club" /></label>
              <label>Ruolo o funzione<input v-model="contact.role" maxlength="200" placeholder="Informazioni e iscrizioni" /></label>
              <label>Telefono<input v-model="contact.phone" type="tel" maxlength="100" placeholder="+39 012 3456789" /></label>
              <label>Email<input v-model="contact.email" type="email" maxlength="254" placeholder="info@club.it" /></label>
              <label class="wide">Indirizzo<textarea v-model="contact.address" rows="2" maxlength="500" placeholder="Via, numero civico, città" /></label>
              <label class="wide">Sito web<input v-model="contact.website" type="url" maxlength="2000" placeholder="https://…" /></label>
            </div>
          </fieldset>
          </fieldset>
          <footer class="editor-footer">
            <button type="button" :disabled="processing || settings.contacts.length >= 50" @click="addContact">+ Aggiungi contatto</button>
            <button class="primary" :disabled="processing">{{ busy ? 'Salvataggio…' : 'Salva contatti' }}</button>
          </footer>
        </form>

        <div v-else-if="activeTab==='teams'" class="editor-layout" :class="{ 'roster-open': rosterTeamId }">
          <template v-if="!rosterTeamId">
            <form ref="editorPanel" class="card content-editor" :aria-busy="busy" @submit.prevent="saveTeam">
              <div class="editor-heading"><div><p class="eyebrow">SCHEDA SQUADRA</p><h2>{{ teamForm.id ? 'Modifica squadra' : 'Nuova squadra' }}</h2></div><span v-if="teamForm.id" class="badge">In modifica</span></div>
              <p class="form-hint required-hint">* Campi obbligatori</p>
              <fieldset class="form-body" :disabled="processing">
                <fieldset class="form-section"><legend><span>01</span> Informazioni della squadra</legend><div class="form-grid">
                  <label class="wide">Nome squadra *<input v-model="teamForm.name" required maxlength="200" placeholder="Es. Under 18 Femminile" /></label>
                  <label>Categoria<input v-model="teamForm.category" placeholder="Es. Under 18" /></label>
                  <label>Genere<select v-model="teamForm.gender"><option value="female">Femminile</option><option value="male">Maschile</option><option value="mixed">Misto</option></select></label>
                  <label class="wide">Campionato<input v-model="teamForm.championship" placeholder="Es. Campionato regionale" /></label>
                  <label class="wide">Descrizione<textarea v-model="teamForm.description" rows="4" placeholder="Presenta la squadra e i suoi obiettivi…" /></label>
                </div></fieldset>
                <fieldset class="form-section"><legend><span>02</span> Immagini e visibilità</legend><div class="form-grid">
                  <ClubMediaField v-model="teamForm.logoUrl" class="wide" label="Logo della squadra" :max-side="512" @uploading="trackImageUpload" />
                  <ClubMediaField v-model="teamForm.heroImageUrl" class="wide" label="Copertina della squadra" @uploading="trackImageUpload" />
                  <label>Ordine di visualizzazione<input v-model.number="teamForm.position" type="number" min="0" step="1" required /></label>
                  <label class="check"><input v-model="teamForm.active" type="checkbox" /> Visibile sul sito pubblico</label>
                </div></fieldset>
              </fieldset>
              <footer class="editor-footer"><button type="button" :disabled="processing" @click="clearEditor('team')">{{ teamForm.id ? 'Annulla modifica' : 'Svuota campi' }}</button><button class="primary" :disabled="processing">{{ busy ? 'Salvataggio…' : 'Salva squadra' }}</button></footer>
            </form>
            <section class="card records-panel"><div class="panel-heading"><h2>Le tue squadre</h2><span class="badge">{{ filteredTeams.length }}</span></div>
              <p v-if="!filteredTeams.length" class="empty-state">{{ teams.length ? 'Nessuna squadra corrisponde ai filtri.' : 'Crea la prima squadra per iniziare a gestire roster e calendario.' }}</p>
              <div class="records-list"><article v-for="team in filteredTeams" :key="team.id" class="record-card" :class="{ selected: teamForm.id === team.id }">
                <div class="record-meta"><span class="badge" :class="{ muted: !team.active }">{{ team.active ? 'Visibile' : 'Nascosta' }}</span><span>{{ (playersByTeam[String(team.id)] || []).length }} atleti</span></div>
                <h3>{{ team.name }}</h3><p>{{ team.championship || team.category || 'Campionato da definire' }}</p>
                <div class="record-actions"><button :disabled="processing" :aria-label="`Gestisci roster ${team.name}`" @click="openRoster(team)">Gestisci roster</button><button :disabled="processing" :aria-label="`Modifica squadra ${team.name}`" @click="editTeam(team)">Modifica</button><button class="danger" :disabled="processing" :aria-label="`Elimina squadra ${team.name}`" @click="askDelete('teams', team)">Elimina</button></div>
              </article></div>
            </section>
          </template>
          <form v-else class="card roster-editor" :aria-busy="busy" @submit.prevent="saveRoster">
            <div class="editor-heading"><div><p class="eyebrow">ROSTER SQUADRA</p><h2>{{ teamName(rosterTeamId) }}</h2><p class="form-hint">Una scheda per ogni atleta. Usa le frecce per scegliere l’ordine sul sito.</p></div><button type="button" :disabled="processing" @click="closeRoster">← Torna alle squadre</button></div>
            <div class="roster-summary"><span class="badge">{{ rosterPlayers.length }} atleti</span><span>Nome e cognome hanno campi separati, anche per i nomi composti.</span></div>
            <fieldset class="form-body" :disabled="processing">
              <p v-if="!rosterPlayers.length" class="empty-state">Il roster è vuoto. Aggiungi il primo atleta per compilare la sua scheda.</p>
              <fieldset v-for="(player, index) in rosterPlayers" :key="player.key" class="player-editor">
                <legend>Atleta {{ index + 1 }}<template v-if="player.firstName || player.lastName"> · {{ player.firstName }} {{ player.lastName }}</template></legend>
                <div class="row-actions"><button type="button" :disabled="index === 0" :aria-label="`Sposta in alto atleta ${index + 1}`" @click="moveItem(rosterPlayers, index, -1)">↑</button><button type="button" :disabled="index === rosterPlayers.length - 1" :aria-label="`Sposta in basso atleta ${index + 1}`" @click="moveItem(rosterPlayers, index, 1)">↓</button><button type="button" class="danger" :aria-label="`Rimuovi atleta ${index + 1}`" @click="rosterPlayers.splice(index, 1)">Rimuovi</button></div>
                <div class="form-grid player-grid">
                  <label>Nome<input v-model="player.firstName" :required="!player.lastName.trim()" autocomplete="off" placeholder="Es. Maria Luisa" /></label>
                  <label>Cognome<input v-model="player.lastName" autocomplete="off" placeholder="Es. De Luca" /></label>
                  <label>Numero di maglia<input v-model.number="player.jerseyNumber" type="number" min="0" step="1" placeholder="Es. 7" /></label>
                  <label>Ruolo<input v-model="player.role" placeholder="Es. Schiacciatrice" /></label>
                  <ClubMediaField v-model="player.imageUrl" class="wide" :label="`Foto atleta ${index + 1}`" :max-side="800" @uploading="trackImageUpload" />
                </div>
              </fieldset>
            </fieldset>
            <div class="add-row"><button type="button" :disabled="processing" @click="addPlayer">+ Aggiungi atleta</button></div>
            <footer class="editor-footer"><span>Le aggiunte, le rimozioni e l’ordine si applicano con Salva roster.</span><button class="primary" :disabled="processing">{{ busy ? 'Salvataggio…' : 'Salva roster' }}</button></footer>
          </form>
        </div>

        <div v-else-if="activeTab==='posts'" class="editor-layout">
          <form ref="editorPanel" class="card content-editor" :aria-busy="busy" @submit.prevent="savePost">
            <div class="editor-heading"><div><p class="eyebrow">COMUNICAZIONI</p><h2>{{ postForm.id ? 'Modifica contenuto' : 'Nuovo contenuto' }}</h2></div><span v-if="postForm.id" class="badge">In modifica</span></div>
            <p class="form-hint required-hint">* Campi obbligatori</p>
            <fieldset class="form-body" :disabled="processing">
              <fieldset class="form-section"><legend><span>01</span> Contenuto editoriale</legend><div class="form-grid">
                <label>Tipo di contenuto<select v-model="postForm.kind"><option value="announcement">Annuncio</option><option value="news">Notizia</option></select></label>
                <label>Destinazione<select v-model.number="postForm.teamId"><option :value="0">Tutta la società</option><option v-for="team in teams" :key="team.id" :value="team.id">{{team.name}}</option></select></label>
                <label class="wide">Titolo *<input v-model="postForm.title" required maxlength="300" placeholder="Un titolo chiaro per la tua comunicazione" /></label>
                <label class="wide">Sommario<textarea v-model="postForm.excerpt" rows="3" placeholder="Una breve introduzione per le anteprime…" /></label>
                <label class="wide">Testo completo<textarea v-model="postForm.body" rows="9" placeholder="Scrivi il contenuto della comunicazione…" /></label>
              </div></fieldset>
              <fieldset class="form-section"><legend><span>02</span> Presentazione e pubblicazione</legend><div class="form-grid">
                <ClubMediaField v-model="postForm.imageUrl" class="wide" label="Immagine del contenuto" @uploading="trackImageUpload" />
                <label>Data di pubblicazione<input v-model="postForm.publishedAt" type="date" /><small>Se vuota, viene usata la data corrente. Non programma la pubblicazione.</small></label>
                <div class="toggle-group"><label class="check"><input v-model="postForm.pinned" type="checkbox" /> In evidenza</label><label class="check"><input v-model="postForm.active" type="checkbox" /> Visibile sul sito pubblico</label></div>
              </div></fieldset>
            </fieldset>
            <footer class="editor-footer"><button type="button" :disabled="processing" @click="clearEditor('post')">{{ postForm.id ? 'Annulla modifica' : 'Svuota campi' }}</button><button class="primary" :disabled="processing">{{ busy ? 'Salvataggio…' : 'Salva contenuto' }}</button></footer>
          </form>
          <section class="card records-panel"><div class="panel-heading"><h2>Archivio contenuti</h2><span class="badge">{{ filteredPosts.length }}</span></div>
            <p v-if="!filteredPosts.length" class="empty-state">{{ overview.posts?.length ? 'Nessun contenuto corrisponde ai filtri.' : 'Non ci sono ancora comunicazioni. Scrivi il primo annuncio del club.' }}</p>
            <div class="records-list"><article v-for="post in filteredPosts" :key="post.id" class="record-card" :class="{ selected: postForm.id === post.id }">
              <div class="record-meta"><span class="badge" :class="{ muted: !post.active }">{{ post.active ? 'Visibile' : 'Nascosto' }}</span><span>{{ post.kind === 'news' ? 'Notizia' : 'Annuncio' }}{{ post.pinned ? ' · In evidenza' : '' }}</span></div>
              <h3>{{ post.title }}</h3><p>{{ teamName(post.teamId) }} · {{ formatDate(post.publishedAt) }}</p><p v-if="post.excerpt" class="record-excerpt">{{post.excerpt}}</p>
              <div class="record-actions"><button :disabled="processing" :aria-label="`Modifica contenuto ${post.title}`" @click="editPost(post)">Modifica</button><button class="danger" :disabled="processing" :aria-label="`Elimina contenuto ${post.title}`" @click="askDelete('posts', post)">Elimina</button></div>
            </article></div>
          </section>
        </div>

        <div v-else-if="activeTab==='gallery'" class="editor-layout">
          <form ref="editorPanel" class="card content-editor" :aria-busy="busy" @submit.prevent="saveGallery">
            <div class="editor-heading"><div><p class="eyebrow">MEDIA DEL CLUB</p><h2>{{ galleryForm.id ? 'Modifica fotografia' : 'Nuova fotografia' }}</h2></div><span v-if="galleryForm.id" class="badge">In modifica</span></div>
            <p class="form-hint required-hint">* Campi obbligatori</p>
            <fieldset class="form-body" :disabled="processing">
              <fieldset class="form-section"><legend><span>01</span> Fotografia e descrizione</legend><div class="form-grid">
                <ClubMediaField v-model="galleryForm.imageUrl" class="wide" label="Fotografia" required @uploading="trackImageUpload" />
                <label class="wide">Titolo<input v-model="galleryForm.title" placeholder="Es. La squadra alla prima di campionato" /></label>
                <label class="wide">Didascalia<textarea v-model="galleryForm.caption" rows="4" placeholder="Racconta il momento rappresentato…" /></label>
              </div></fieldset>
              <fieldset class="form-section"><legend><span>02</span> Organizzazione</legend><div class="form-grid">
                <label>Squadra<select v-model.number="galleryForm.teamId"><option :value="0">Tutta la società</option><option v-for="team in teams" :key="team.id" :value="team.id">{{team.name}}</option></select></label>
                <label>Data dell’evento<input v-model="galleryForm.eventDate" type="date" /></label>
                <label>Ordine di visualizzazione<input v-model.number="galleryForm.position" type="number" min="0" step="1" required /></label>
                <label class="check"><input v-model="galleryForm.active" type="checkbox" /> Visibile sul sito pubblico</label>
              </div></fieldset>
            </fieldset>
            <footer class="editor-footer"><button type="button" :disabled="processing" @click="clearEditor('gallery')">{{ galleryForm.id ? 'Annulla modifica' : 'Svuota campi' }}</button><button class="primary" :disabled="processing">{{ busy ? 'Salvataggio…' : 'Salva fotografia' }}</button></footer>
          </form>
          <section class="card records-panel"><div class="panel-heading"><h2>Archivio fotografico</h2><span class="badge">{{ filteredGallery.length }}</span></div>
            <p v-if="!filteredGallery.length" class="empty-state">{{ overview.gallery?.length ? 'Nessuna fotografia corrisponde ai filtri.' : 'La gallery è vuota. Aggiungi la prima fotografia della stagione.' }}</p>
            <div class="records-list"><article v-for="photo in filteredGallery" :key="photo.id" class="record-card photo-record" :class="{ selected: galleryForm.id === photo.id }">
              <img :src="photo.imageUrl" :alt="photo.title || 'Fotografia del club'" loading="lazy" />
              <div class="record-meta"><span class="badge" :class="{ muted: !photo.active }">{{photo.active ? 'Visibile' : 'Nascosta'}}</span><span>{{formatDate(photo.eventDate)}}</span></div>
              <h3>{{ photo.title || 'Fotografia senza titolo' }}</h3><p>{{teamName(photo.teamId)}}</p>
              <div class="record-actions"><button :disabled="processing" :aria-label="`Modifica fotografia ${photo.title || photo.id}`" @click="editGallery(photo)">Modifica</button><button class="danger" :disabled="processing" :aria-label="`Elimina fotografia ${photo.title || photo.id}`" @click="askDelete('gallery', photo)">Elimina</button></div>
            </article></div>
          </section>
        </div>

        <div v-else-if="activeTab==='sponsors'" class="editor-layout">
          <form ref="editorPanel" class="card content-editor" :aria-busy="busy" @submit.prevent="saveSponsor">
            <div class="editor-heading"><div><p class="eyebrow">PARTNER DEL CLUB</p><h2>{{ sponsorForm.id ? 'Modifica sponsor' : 'Nuovo sponsor' }}</h2></div><span v-if="sponsorForm.id" class="badge">In modifica</span></div>
            <p class="form-hint required-hint">* Campi obbligatori. Gli sponsor pubblicati compaiono nella modale “I nostri Sponsor”.</p>
            <fieldset class="form-body" :disabled="processing">
              <fieldset class="form-section"><legend><span>01</span> Presentazione dello sponsor</legend><div class="form-grid">
                <label class="wide">Nome sponsor *<input v-model="sponsorForm.name" required maxlength="200" placeholder="Nome dell’azienda o del partner" /></label>
                <ClubMediaField v-model="sponsorForm.logoUrl" class="wide" label="Logo dello sponsor" :max-side="512" required @uploading="trackImageUpload" />
                <label class="wide">Sito web *<input v-model="sponsorForm.website" type="url" required maxlength="2000" placeholder="https://www.sponsor.it" /><small>Il logo e il nome apriranno questo sito in una nuova scheda.</small></label>
                <label class="wide">Descrizione (opzionale)<textarea v-model="sponsorForm.description" rows="4" maxlength="3000" placeholder="Presenta lo sponsor e il suo sostegno al club…" /></label>
              </div></fieldset>
              <fieldset class="form-section"><legend><span>02</span> Ordine e pubblicazione</legend><div class="form-grid">
                <label>Ordine di visualizzazione<input v-model.number="sponsorForm.position" type="number" min="0" max="1000000" step="1" required /><small>I valori più bassi compaiono per primi.</small></label>
                <label class="check"><input v-model="sponsorForm.active" type="checkbox" /> Pubblicato sul sito pubblico</label>
              </div></fieldset>
            </fieldset>
            <footer class="editor-footer"><button type="button" :disabled="processing" @click="clearEditor('sponsor')">{{ sponsorForm.id ? 'Annulla modifica' : 'Svuota campi' }}</button><button class="primary" :disabled="processing">{{ busy ? 'Salvataggio…' : 'Salva sponsor' }}</button></footer>
          </form>
          <section class="card records-panel"><div class="panel-heading"><h2>Gli sponsor del club</h2><span class="badge">{{ filteredSponsors.length }}</span></div>
            <p v-if="!filteredSponsors.length" class="empty-state">{{ overview.sponsors?.length ? 'Nessuno sponsor corrisponde ai filtri.' : 'Aggiungi il primo sponsor con logo e collegamento al suo sito.' }}</p>
            <div class="records-list"><article v-for="sponsor in filteredSponsors" :key="sponsor.id" class="record-card sponsor-record" :class="{ selected: sponsorForm.id === sponsor.id }">
              <img :src="sponsor.logoUrl" :alt="`Logo ${sponsor.name}`" loading="lazy" />
              <div class="record-meta"><span class="badge" :class="{ muted: !sponsor.active }">{{ sponsor.active ? 'Pubblicato' : 'Bozza' }}</span><span>Ordine {{ sponsor.position }}</span></div>
              <h3>{{ sponsor.name }}</h3><p>{{ sponsor.website }}</p><p v-if="sponsor.description" class="record-excerpt">{{ sponsor.description }}</p>
              <div class="record-actions"><button :disabled="processing" :aria-label="`Modifica sponsor ${sponsor.name}`" @click="editSponsor(sponsor)">Modifica</button><button class="danger" :disabled="processing" :aria-label="`Elimina sponsor ${sponsor.name}`" @click="askDelete('sponsors', sponsor)">Elimina</button></div>
            </article></div>
          </section>
        </div>

        <div v-else-if="activeTab==='matches'" class="editor-layout">
          <form ref="editorPanel" class="card content-editor" :aria-busy="busy" @submit.prevent="saveMatch">
            <div class="editor-heading"><div><p class="eyebrow">CALENDARIO E RISULTATI</p><h2>{{ matchForm.id ? 'Modifica partita' : 'Nuova partita' }}</h2></div><span v-if="matchForm.id" class="badge">In modifica</span></div>
            <p class="form-hint required-hint">* Campi obbligatori</p>
            <fieldset class="form-body" :disabled="processing">
              <fieldset class="form-section"><legend><span>01</span> Squadre e competizione</legend><div class="form-grid">
                <label>Squadra del club *<select v-model.number="matchForm.teamId" required><option :value="0" disabled>Seleziona una squadra</option><option v-for="team in teams" :key="team.id" :value="team.id">{{team.name}}</option></select></label>
                <label>Avversario *<input v-model="matchForm.opponent" required placeholder="Nome della squadra avversaria" /></label>
                <label>Competizione<input v-model="matchForm.competition" placeholder="Es. Campionato regionale" /></label>
                <label>Giornata o turno<input v-model="matchForm.roundLabel" placeholder="Es. 5ª giornata · Andata" /></label>
              </div></fieldset>
              <fieldset class="form-section"><legend><span>02</span> Data e impianto</legend><div class="form-grid">
                <label>Data e ora *<input v-model="matchForm.scheduledAt" type="datetime-local" required /></label>
                <label>Campo di gioco<select v-model="matchForm.home"><option :value="true">In casa</option><option :value="false">In trasferta</option></select></label>
                <label class="wide">Impianto o indirizzo<input v-model="matchForm.location" placeholder="Es. Palazzetto dello Sport, Via Roma 10" /></label>
              </div></fieldset>
              <fieldset class="form-section"><legend><span>03</span> Stato e risultato</legend><div class="form-grid">
                <label class="wide">Stato della partita<select v-model="matchForm.status"><option v-for="(label, value) in statusLabels" :key="value" :value="value">{{label}}</option></select></label>
                <template v-if="matchForm.status !== 'scheduled' || matchForm.sets.length">
                  <label>Risultato · {{ teams.find(team => team.id === matchForm.teamId)?.name || 'Squadra del club' }}<input v-model.number="matchForm.scoreFor" type="number" min="0" step="1" required /></label>
                  <label>Risultato · {{ matchForm.opponent || 'Avversario' }}<input v-model.number="matchForm.scoreAgainst" type="number" min="0" step="1" required /></label>
                  <p class="form-hint wide">Inserisci il risultato complessivo (set vinti per la pallavolo). I parziali sono facoltativi e seguono sempre l’ordine squadra del club / avversario, anche in trasferta.</p>
                  <div class="set-editor wide">
                    <div class="panel-heading"><h3>Parziali dell’incontro</h3><span class="badge">{{matchForm.sets.length}}</span></div>
                    <p v-if="!matchForm.sets.length" class="form-hint">Nessun parziale. Aggiungi una riga per ogni set o periodo di gioco.</p>
                    <div v-for="(set, index) in matchForm.sets" :key="set.key" class="set-row">
                      <span class="set-number">{{index + 1}}°</span>
                      <label>{{teams.find(team => team.id === matchForm.teamId)?.name || 'Squadra'}}<input v-model.number="set.scoreFor" type="number" min="0" step="1" required :aria-label="`Parziale ${index + 1}, squadra del club`" placeholder="0" /></label>
                      <span aria-hidden="true">:</span>
                      <label>{{matchForm.opponent || 'Avversario'}}<input v-model.number="set.scoreAgainst" type="number" min="0" step="1" required :aria-label="`Parziale ${index + 1}, avversario`" placeholder="0" /></label>
                      <button type="button" class="danger" :aria-label="`Rimuovi parziale ${index + 1}`" @click="matchForm.sets.splice(index, 1)">×</button>
                    </div>
                    <button type="button" class="add-set" @click="addSet">+ Aggiungi parziale</button>
                  </div>
                </template>
              </div></fieldset>
            </fieldset>
            <footer class="editor-footer"><button type="button" :disabled="processing" @click="clearEditor('match')">{{ matchForm.id ? 'Annulla modifica' : 'Svuota campi' }}</button><button class="primary" :disabled="processing || !teams.length">{{ busy ? 'Salvataggio…' : 'Salva partita' }}</button></footer>
            <p v-if="!teams.length" class="form-hint">Crea prima una squadra nella sezione Squadre e roster.</p>
          </form>
          <section class="card records-panel"><div class="panel-heading"><h2>Incontri del club</h2><span class="badge">{{filteredMatches.length}}</span></div>
            <p v-if="!filteredMatches.length" class="empty-state">{{ overview.matches?.length ? 'Nessuna partita corrisponde ai filtri.' : 'Il calendario è vuoto. Inserisci il primo incontro della stagione.' }}</p>
            <div class="records-list"><article v-for="match in filteredMatches" :key="match.id" class="record-card" :class="{ selected: matchForm.id === match.id }">
              <div class="record-meta"><span class="badge" :class="match.status">{{ statusLabels[match.status] || match.status }}</span><span>{{match.home ? 'In casa' : 'Trasferta'}}</span></div>
              <h3>{{teamName(match.teamId)}} <span class="versus">vs</span> {{match.opponent}}</h3>
              <p>{{formatDate(match.scheduledAt)}}</p><p v-if="match.competition || match.roundLabel">{{[match.competition, match.roundLabel].filter(Boolean).join(' · ')}}</p><p v-if="match.location">{{match.location}}</p>
              <div v-if="match.status !== 'scheduled'" class="match-result">{{match.scoreFor}} <span>:</span> {{match.scoreAgainst}}<small>Squadra del club / avversario</small></div>
              <div class="record-actions"><button :disabled="processing" :aria-label="`Modifica partita contro ${match.opponent}`" @click="editMatch(match)">Modifica</button><button class="danger" :disabled="processing" :aria-label="`Elimina partita contro ${match.opponent}`" @click="askDelete('matches', match)">Elimina</button></div>
            </article></div>
          </section>
        </div>
      </section>
      <dialog ref="deleteDialog" class="delete-dialog" aria-labelledby="delete-title" aria-describedby="delete-description" @cancel.prevent="cancelDelete">
        <template v-if="pendingDelete">
          <p class="eyebrow">CONFERMA ELIMINAZIONE</p><h2 id="delete-title">Eliminare questo elemento?</h2>
          <p class="delete-name">{{ pendingDelete.item.name || pendingDelete.item.title || pendingDelete.item.opponent || 'Fotografia selezionata' }}</p>
          <p id="delete-description">{{pendingDelete.kind === 'teams' ? 'Verranno eliminati anche tutti gli atleti, gli incontri, le fotografie e i contenuti collegati alla squadra.' : 'L’elemento verrà rimosso dal CMS e dal sito pubblico.'}} Questa operazione non può essere annullata.</p>
          <div class="dialog-actions"><button type="button" autofocus :disabled="processing" @click="cancelDelete">Annulla</button><button type="button" class="danger-filled" :disabled="processing" @click="confirmDelete">{{busy ? 'Eliminazione…' : 'Elimina definitivamente'}}</button></div>
        </template>
      </dialog>
    </template>
  </main>
</template>

<style scoped>
*{box-sizing:border-box}.club-admin{min-height:100vh;background:#07101f;color:#e5e7eb;font-family:Inter,ui-sans-serif,system-ui,sans-serif}.center,.login-shell{min-height:100vh;display:grid;place-items:center}.login-shell{background:radial-gradient(circle at 20% 20%,#172554,#07101f 55%)}.login-card{width:min(460px,calc(100% - 32px));padding:38px;background:rgba(15,23,42,.88);border:1px solid #263449;border-radius:24px;box-shadow:0 30px 80px rgba(0,0,0,.35)}.login-card h1{font-size:38px;letter-spacing:-.05em;margin:8px 0}.login-card>p{color:#94a3b8}.login-card form{display:grid;gap:14px;margin-top:24px}.login-card label,.form-grid label{display:flex;flex-direction:column;gap:6px;font-size:12px;color:#94a3b8}.login-card input,.form-grid input,.form-grid textarea,.form-grid select,.roster-editor textarea{width:100%;background:#060d19;border:1px solid #293548;border-radius:9px;padding:10px 11px;color:#f8fafc;font:inherit}.login-card button,.primary{border:0;border-radius:9px;padding:11px 16px;background:#f59e0b;color:#111827;font-weight:900;cursor:pointer}.eyebrow{margin:0;color:#f59e0b;font-size:10px;font-weight:950;letter-spacing:.16em}.error{color:#f87171}.notice{position:fixed;right:24px;top:22px;background:#dcfce7;color:#166534;padding:11px 16px;border-radius:10px;z-index:10}aside{position:fixed;inset:0 auto 0 0;width:245px;background:#030712;border-right:1px solid #172033;padding:24px 18px;display:flex;flex-direction:column}.admin-brand{display:flex;align-items:center;gap:11px;color:#fff;text-decoration:none;margin-bottom:34px}.admin-brand>span{width:43px;height:43px;display:grid;place-items:center;border-radius:11px;background:#f59e0b;color:#111827;font-weight:950}.admin-brand div{display:flex;flex-direction:column}.admin-brand small{color:#64748b}.club-admin nav{display:grid;gap:6px}.club-admin nav button{border:0;border-radius:9px;padding:11px 13px;text-align:left;background:transparent;color:#94a3b8;font-weight:700;cursor:pointer}.club-admin nav button.active{background:#172554;color:#fff}.aside-bottom{margin-top:auto;display:grid;gap:10px}.aside-bottom a,.aside-bottom button{color:#94a3b8;background:transparent;border:0;text-align:left;text-decoration:none;cursor:pointer}.workspace{margin-left:245px;min-height:100vh;padding:34px max(28px,calc((100vw - 1220px)/2))}.workspace>header{display:flex;align-items:center;justify-content:space-between;margin-bottom:30px}.workspace h1{font-size:36px;letter-spacing:-.04em;margin:4px 0}.status{font-size:10px;font-weight:900;padding:7px 10px;border-radius:999px;background:#052e16;color:#4ade80}.stats{display:grid;grid-template-columns:repeat(5,1fr);gap:12px;margin-bottom:18px}.stats article,.card{background:#0e1727;border:1px solid #233047;border-radius:16px;padding:20px}.stats span{display:block;color:#64748b;font-size:12px}.stats strong{font-size:34px}.card{margin-bottom:16px}.card h2{margin:0 0 18px}.welcome{display:flex;align-items:center;justify-content:space-between;background:linear-gradient(120deg,#172554,#111827)}.welcome h2{font-size:30px;margin:6px 0}.welcome p{color:#94a3b8}.welcome a{background:#f59e0b;color:#111827;padding:11px 16px;border-radius:9px;text-decoration:none;font-weight:900}.quick{display:grid;grid-template-columns:repeat(4,1fr);gap:10px}.quick button,.buttons button,.list button{background:#111d31;border:1px solid #2b3a52;border-radius:9px;color:#cbd5e1;padding:10px;cursor:pointer}.form-grid{display:grid;grid-template-columns:repeat(2,1fr);gap:13px}.form-grid.single{grid-template-columns:1fr}.form-grid .wide{grid-column:1/-1}.form-grid .check{flex-direction:row;align-items:center}.form-grid .check input{width:auto}.card>.primary{margin-top:18px}.split{display:grid;grid-template-columns:minmax(290px,.75fr) minmax(420px,1.25fr);gap:16px;align-items:start}.roster-editor{grid-column:1/-1}.roster-editor>p{color:#94a3b8}.roster-editor textarea{resize:vertical}.buttons{display:flex;gap:8px;margin-top:14px}.list{display:grid;gap:8px}.list article{display:flex;align-items:center;justify-content:space-between;gap:12px;padding:13px;background:#091221;border:1px solid #1c2940;border-radius:10px}.list small{display:block;color:#64748b;margin-top:3px}.list .danger,.actions .danger{color:#f87171}.photo-admin{display:grid;grid-template-columns:repeat(3,1fr);gap:9px}.photo-admin figure{margin:0;background:#091221;border-radius:10px;overflow:hidden}.photo-admin img{width:100%;aspect-ratio:1;object-fit:cover}.photo-admin figcaption{display:flex;justify-content:space-between;padding:8px;font-size:11px}.photo-admin button{border:0;background:transparent;color:#f87171;cursor:pointer}@media(max-width:900px){aside{position:static;width:100%;height:auto}.club-admin nav{display:flex;overflow:auto}.aside-bottom{display:none}.workspace{margin-left:0;padding:22px 16px}.stats{grid-template-columns:repeat(2,1fr)}.split,.form-grid{grid-template-columns:1fr}.quick{grid-template-columns:1fr 1fr}.welcome{align-items:flex-start;gap:20px;flex-direction:column}.photo-admin{grid-template-columns:repeat(2,1fr)}}
.form-hint{color:#94a3b8;font-size:13px;line-height:1.5}.tile-icons-grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:12px;margin:20px 0 24px}.contact-editor{position:relative;min-width:0;border:1px solid #263449;border-radius:16px;padding:18px;margin:20px 0}.contact-editor legend{padding:0 8px;color:#f8fafc;font-size:14px;font-weight:800}.contact-actions{display:flex;justify-content:flex-end;gap:8px;margin-bottom:16px}.contact-actions button,.contacts-editor .buttons>button:not(.primary){border:1px solid #334155;background:#172238;color:#cbd5e1;border-radius:8px;padding:8px 12px;cursor:pointer}.contact-actions button.danger{color:#fca5a5}.club-admin button:disabled{opacity:.5;cursor:not-allowed}
.form-grid+h2{margin-top:32px}.contacts-editor .buttons>.primary{background:#f59e0b;color:#111827;border-color:#f59e0b}
@media(max-width:1100px){.tile-icons-grid{grid-template-columns:repeat(2,minmax(0,1fr))}}@media(max-width:600px){.tile-icons-grid{grid-template-columns:1fr}.contact-editor{padding:12px}.contacts-editor .buttons{flex-wrap:wrap}}
.typography-preview{margin:18px 0 28px;padding:22px;border:1px solid #334155;border-radius:12px;background:#f8fafc;color:#0f172a;font-family:var(--club-font-body)}
.typography-preview small{color:#475569;font-size:12px}.typography-preview h3{margin:12px 0;font-family:var(--club-font-title);font-size:clamp(22px,3vw,32px);line-height:1.2}.typography-preview p{margin:0;font-size:15px;line-height:1.6}
.typography-editor{min-width:0;margin:0 0 28px;padding:18px;border:1px solid #475569;border-radius:14px;background:#111d31}
.typography-editor legend{padding:0 8px;color:#f8fafc;font-size:20px;font-weight:800}.typography-editor .typography-preview{margin-bottom:0}
.club-admin select{display:block;min-height:46px;appearance:auto;color-scheme:dark}
.club-admin select option{background:#060d19;color:#f8fafc}
.club-admin .font-select{min-width:0;min-height:48px;padding:10px 12px;border:2px solid #94a3b8;border-radius:10px;background:#fff;color:#0f172a;color-scheme:light;font-size:16px;line-height:1.5;cursor:pointer}
.club-admin .font-select option{background:#fff;color:#0f172a}.club-admin .font-select option:disabled{color:#64748b}
.club-admin select:focus-visible{outline:3px solid #f59e0b;outline-offset:3px}
.club-admin{color:#e2e8f0;background:#080f1b;font-size:14px;line-height:1.5}
.workspace{padding:32px clamp(20px,3vw,48px);max-width:1800px}
.workspace>header{gap:24px;align-items:flex-start;margin-bottom:26px}.workspace h1{font-size:clamp(28px,3vw,36px);line-height:1.2;margin:9px 0}.page-description{margin:0;color:#9aabc1;font-size:14px;max-width:680px}
.public-link{flex-shrink:0;border:1px solid #3c4b62;border-radius:9px;padding:10px 14px;text-decoration:none;color:#e2e8f0;font-weight:650;font-size:13px}.public-link:hover{background:#18263b}
.draft-indicator{font-size:12px;color:#fbbf24}.aside-bottom{padding-top:24px}.admin-brand small{max-width:140px;overflow-wrap:anywhere;color:#94a3b8}
.processing-banner{padding:12px 16px;border:1px solid #345174;border-radius:9px;background:#132842;color:#bfdbfe;font-size:13px;margin-bottom:20px}
.club-admin button{font:inherit;font-size:13px;font-weight:650;line-height:1.4}.club-admin button:not(.primary):not(.danger-filled){border:1px solid #35445b;border-radius:8px;background:#142238;color:#dbe5f3;padding:9px 12px;cursor:pointer}
.club-admin nav button:not(.primary){border-color:transparent;background:transparent;color:#a5b4c9;padding:12px}.club-admin nav button.active{border-color:#344d77;background:#172c4b;color:#fff}.club-admin .aside-bottom button{background:transparent;text-align:left;border:0;padding:0;color:#a5b4c9}
.club-admin button.primary{background:#fbbf24;color:#172033;border:1px solid #fbbf24;border-radius:9px;padding:11px 18px;font-weight:800}.club-admin button.primary:hover:not(:disabled){background:#fcd34d}.club-admin button.danger{color:#fca5a5}.club-admin button.danger:hover:not(:disabled){background:#431e2b;border-color:#9f4b59}
.club-admin button:focus-visible,.club-admin a:focus-visible,.club-admin input:focus-visible,.club-admin textarea:focus-visible{outline:3px solid #fbbf24;outline-offset:3px}.club-admin input,.club-admin select,.club-admin textarea{color-scheme:dark}
.card{padding:24px;border-color:#29384e;border-radius:14px;min-width:0;margin:0 0 20px;background:#101b2c;box-shadow:0 4px 16px #00000012}.card h2{font-size:20px;line-height:1.35;margin:0}.card h3{font-size:15px}
.collection-toolbar{display:flex;align-items:flex-end;flex-wrap:wrap;gap:14px;border:1px solid #29384e;border-radius:12px;background:#101b2c;padding:16px 18px;margin-bottom:24px}.collection-toolbar label{flex:1 1 160px}.collection-toolbar .search-field{flex:2 1 240px}
.collection-toolbar label,.form-grid label{font-size:13px;font-weight:600;color:#bdcbdd;display:flex;flex-direction:column;gap:8px}.collection-toolbar input,.collection-toolbar select,.form-grid input,.form-grid textarea,.form-grid select{width:100%;min-width:0;min-height:44px;border:1px solid #40516b;border-radius:8px;padding:11px 12px;background:#091322;color:#f1f5f9;font:inherit;font-size:14px;font-weight:400;line-height:1.5;transition:border-color .15s}
.form-grid textarea{resize:vertical}.form-grid input::placeholder,.form-grid textarea::placeholder,.collection-toolbar input::placeholder{color:#7c90ac;opacity:1}.form-grid input[type=color]{padding:5px}.form-grid input[type=checkbox]{min-height:auto;accent-color:#fbbf24;width:17px;height:17px;flex-shrink:0}.form-grid label small{font-size:12px;font-weight:400;color:#9aabc1;line-height:1.5}
.filter-summary{display:flex;flex:1 1 150px;align-items:center;justify-content:flex-end;gap:12px;min-height:44px;color:#a5b4c9;font-size:12px}.club-admin .filter-summary button{border:0;background:transparent;padding:6px;color:#fcd34d;white-space:nowrap}
.editor-layout{display:grid;grid-template-columns:minmax(0,1.2fr) minmax(300px,.85fr);gap:24px;align-items:start}.editor-layout.roster-open{grid-template-columns:1fr}
.editor-heading,.panel-heading{display:flex;justify-content:space-between;align-items:flex-start;gap:16px}.editor-heading .eyebrow{font-size:10px;margin-bottom:8px}.editor-heading>div{min-width:0}.editor-heading h2{overflow-wrap:anywhere}.required-hint{font-size:12px;margin:14px 0 26px}
.form-body{border:0;padding:0;margin:0;min-width:0}.form-section{min-width:0;border:0;border-top:1px solid #2d3b51;padding:22px 0 0;margin:0 0 28px}.form-section legend{padding:0 12px 0 0;color:#f1f5f9;font-size:14px;font-weight:750}.form-section legend>span{color:#fbbf24;font-size:11px;padding-right:8px;font-variant-numeric:tabular-nums}.form-section .form-grid{gap:18px 16px}.form-grid>*{min-width:0}
.form-grid .check{gap:9px;font-weight:500;line-height:1.6}.toggle-group{display:flex;flex-direction:column;gap:14px;justify-content:center}.editor-footer{display:flex;align-items:center;justify-content:space-between;gap:14px;border-top:1px solid #2d3b51;padding-top:20px;margin-top:4px}.editor-footer>span{color:#9aabc1;font-size:12px;max-width:65%}.editor-footer button{flex-shrink:0}
.records-panel>.panel-heading{align-items:center;margin-bottom:20px}.records-panel h2{font-size:18px}.badge{display:inline-flex;align-items:center;justify-content:center;gap:5px;background:#1f3654;border:1px solid #345174;color:#bfdbfe;border-radius:6px;padding:4px 8px;font-size:11px;font-weight:650;white-space:nowrap}.badge.muted{background:#222e41;color:#a9b6c8;border-color:#39475e}.badge.completed{background:#103b31;color:#86efac;border-color:#21634d}.badge.live{background:#491f2c;color:#fda4af;border-color:#89364a}.badge.scheduled{background:#1f3654;color:#bfdbfe}
.records-list{display:grid;gap:14px}.record-card{min-width:0;padding:18px;border:1px solid #2b3c55;border-radius:10px;background:#0c1626}.record-card.selected{border-color:#d5a238;box-shadow:inset 3px 0 #fbbf24}.record-meta{display:flex;align-items:center;justify-content:space-between;flex-wrap:wrap;gap:8px;font-size:11px;color:#9aabc1}.record-card h3{font-size:16px;margin:14px 0 6px;line-height:1.45;overflow-wrap:anywhere;color:#f1f5f9}.record-card>p{font-size:12px;line-height:1.6;color:#a5b4c9;margin:5px 0;overflow-wrap:anywhere}.record-excerpt{display:-webkit-box;-webkit-line-clamp:2;-webkit-box-orient:vertical;overflow:hidden}.record-actions{display:flex;gap:8px;flex-wrap:wrap;border-top:1px solid #22324a;padding-top:14px;margin-top:16px}.record-actions button{flex:1 0 auto}.club-admin .record-actions .danger{flex:0 0 auto;border-color:transparent;background:transparent}
.empty-state{border:1px dashed #3e506b;border-radius:10px;background:#0b1525;padding:28px 20px;text-align:center;font-size:13px;line-height:1.8;color:#a9bad0;margin:16px 0}.error-banner{display:flex;align-items:center;justify-content:space-between;gap:12px;border:1px solid #9d4355;background:#401e2c;color:#fecdd3;border-radius:10px;padding:14px 18px;font-size:14px;position:sticky;top:12px;z-index:5}.club-admin .error-banner button{padding:0 6px;background:transparent;border:0;color:inherit;font-size:22px}.notice{box-shadow:0 8px 30px #0004;max-width:calc(100vw - 32px)}
.roster-editor{grid-column:1/-1}.roster-summary{display:flex;gap:14px;align-items:center;flex-wrap:wrap;margin:24px 0;color:#a5b4c9;font-size:12px}.player-editor{min-width:0;position:relative;margin:20px 0;padding:20px;border:1px solid #344660;border-radius:12px;background:#0c1626}.player-editor legend{color:#e2e8f0;font-size:14px;font-weight:750;padding:0 8px;max-width:100%;overflow-wrap:anywhere}.row-actions{display:flex;justify-content:flex-end;gap:8px;margin:0 0 18px}.player-grid{grid-template-columns:repeat(4,minmax(0,1fr))}.add-row{padding:16px 0 24px}.club-admin .add-row button,.club-admin .add-set{border-style:dashed;color:#fcd34d;background:#19263a}
.set-editor{padding:18px;background:#0a1424;border:1px solid #30415b;border-radius:10px}.set-editor h3{margin:0;font-size:14px}.set-row{display:grid;grid-template-columns:28px minmax(0,1fr) 8px minmax(0,1fr) 30px;align-items:end;gap:8px;margin:18px 0}.set-row>label{font-size:11px}.set-number{align-self:end;line-height:44px;color:#94a3b8;font-weight:700}.set-row>span:not(.set-number){line-height:44px;text-align:center}.club-admin .set-row button{padding:8px 0;height:44px;font-size:20px;border-color:transparent;background:transparent}.set-row input{text-align:center;font-variant-numeric:tabular-nums}.add-set{margin-top:8px;width:100%}.match-result{margin-top:16px;font-size:26px;font-weight:800;color:#f8fafc;font-variant-numeric:tabular-nums}.match-result>span{color:#64748b;margin:0 6px}.match-result small{display:block;font-size:11px;font-weight:500;color:#9aabc1}.versus{font-size:11px;color:#94a3b8;font-weight:500}
.media-preview{display:grid;justify-items:center;gap:10px;padding:14px;border:1px dashed #3e506b;border-radius:10px;background:#091322}.media-preview img{width:100%;max-height:220px;object-fit:contain;border-radius:6px}.media-preview span{font-size:11px;color:#9aabc1;text-align:center}.photo-record>img{display:block;width:100%;aspect-ratio:16/9;object-fit:cover;border-radius:6px;background:#17243a;margin-bottom:14px}.sponsor-record>img{display:block;width:100%;height:130px;object-fit:contain;border-radius:6px;background:#f8fafc;padding:14px;margin-bottom:14px}.stats{grid-template-columns:repeat(6,minmax(0,1fr))}.typography-editor{background:#0c1626;border-color:#344660}.typography-editor legend{font-size:16px}.settings-editor .form-section{margin-top:28px}
.delete-dialog{width:min(480px,calc(100vw - 32px));border:1px solid #485a74;border-radius:16px;padding:28px;background:#101b2c;color:#e2e8f0;box-shadow:0 24px 80px #0008}.delete-dialog::backdrop{background:#020817b8;backdrop-filter:blur(3px)}.delete-dialog h2{font-size:22px;line-height:1.3;margin:12px 0}.delete-dialog p{font-size:14px;line-height:1.7;color:#afbed2}.delete-dialog .eyebrow{font-size:10px;color:#fca5a5}.delete-dialog .delete-name{color:#fff;font-weight:700;overflow-wrap:anywhere}.dialog-actions{display:flex;gap:10px;justify-content:flex-end;margin-top:24px}.club-admin .danger-filled{border:1px solid #dc4c67;border-radius:8px;padding:10px 14px;background:#b42342;color:#fff;cursor:pointer}
@media(min-width:1600px){.editor-layout{grid-template-columns:minmax(0,1.3fr) minmax(340px,.85fr)}}
@media(max-width:1200px){.editor-layout{grid-template-columns:minmax(0,1fr)}.records-list{grid-template-columns:repeat(2,minmax(0,1fr))}.player-grid{grid-template-columns:repeat(2,minmax(0,1fr))}}
@media(max-width:900px){.workspace{margin-left:0;padding:24px 20px}.stats{grid-template-columns:repeat(3,minmax(0,1fr))}.admin-brand{margin-bottom:20px}.club-admin nav{padding-bottom:4px}.club-admin nav button:not(.primary){white-space:nowrap}.workspace>header{margin-bottom:24px}.collection-toolbar{gap:12px}.aside-bottom{display:flex;align-items:center;justify-content:space-between;flex-wrap:wrap;gap:12px;padding-top:18px}.draft-indicator{margin:0}}
@media(max-width:600px){.workspace{padding:22px 14px}.workspace>header{flex-direction:column;gap:16px}.workspace h1{font-size:28px}.public-link{align-self:flex-start}.card{padding:18px}.form-grid,.player-grid{grid-template-columns:minmax(0,1fr)}.records-list{grid-template-columns:minmax(0,1fr)}.collection-toolbar{padding:14px}.collection-toolbar label,.collection-toolbar .search-field{flex-basis:100%}.filter-summary{justify-content:space-between}.editor-heading{flex-wrap:wrap}.editor-footer{flex-wrap:wrap}.editor-footer>span{max-width:100%;flex-basis:100%}.editor-footer button.primary{flex:1}.player-editor{padding:14px}.set-editor{padding:12px}.set-row{grid-template-columns:24px minmax(0,1fr) 5px minmax(0,1fr) 26px;gap:5px}.set-row label{font-size:10px}.set-row input{padding:10px 6px}.dialog-actions{flex-direction:column-reverse}.dialog-actions button{width:100%}}
@media(prefers-reduced-motion:reduce){*{scroll-behavior:auto!important;transition:none!important}}
</style>
