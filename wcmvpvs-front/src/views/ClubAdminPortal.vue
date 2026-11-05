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
const sidebarCollapsed = ref(false)
const mobileMenuOpen = ref(false)
const mobileMenuTrigger = ref(null)
const navAside = ref(null)
const navGroups = [
  { label: 'Workspace', ids: ['dashboard'] },
  { label: 'Contenuti', ids: ['settings', 'posts', 'gallery', 'matches'] },
  { label: 'Società', ids: ['teams', 'sponsors', 'contacts'] },
]
const navIcons = {
  dashboard: 'M3 3h7v7H3z M14 3h7v7h-7z M3 14h7v7H3z M14 14h7v7h-7z',
  settings: 'M3 10l9-7 9 7 M5 9v12h14V9 M9 21v-8h6v8',
  teams: 'M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2 M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8 M20 21v-2a4 4 0 0 0-3-3.87 M16 3.13a4 4 0 0 1 0 7.75',
  posts: 'M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z M14 2v6h6 M8 13h8 M8 17h6',
  gallery: 'M3 3h18v18H3z M3 16l6-6 5 5 3-3 4 4 M16 7h.01',
  matches: 'M8 2v4 M16 2v4 M3 10h18 M5 4h14a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2 M8 14h.01 M12 14h.01 M16 14h.01 M8 18h.01 M12 18h.01',
  sponsors: 'M3 8h18v13H3z M8 8V5a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v3 M3 13a22 22 0 0 0 18 0 M12 12v4',
  contacts: 'M21 16.5v3a2 2 0 0 1-2.2 2 19.8 19.8 0 0 1-8.6-3.1 19.4 19.4 0 0 1-6-6A19.8 19.8 0 0 1 1.1 3.8 2 2 0 0 1 3.1 1.6h3a2 2 0 0 1 2 1.7c.1.9.4 1.7.7 2.5a2 2 0 0 1-.5 2.1L7 9.2a16 16 0 0 0 6 6l1.3-1.3a2 2 0 0 1 2.1-.5c.8.3 1.6.6 2.5.7a2 2 0 0 1 2.1 2.4z',
}
async function openMobileMenu() {
  mobileMenuOpen.value = true
  await nextTick()
  navAside.value?.querySelector('button:not(:disabled), a')?.focus()
}
function closeMobileMenu() {
  mobileMenuOpen.value = false
  nextTick(() => { if (!mobileMenuOpen.value) mobileMenuTrigger.value?.focus() })
}
function selectTab(id) { activeTab.value = id; if (mobileMenuOpen.value) closeMobileMenu() }
function navigationKeydown(event) {
  if (!mobileMenuOpen.value) return
  if (event.key === 'Escape') { event.preventDefault(); closeMobileMenu(); return }
  if (event.key !== 'Tab') return
  const focusable = [...navAside.value.querySelectorAll('button:not(:disabled), a[href]')].filter(item => item.getClientRects().length)
  const first = focusable[0], last = focusable[focusable.length - 1]
  if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last?.focus() }
  else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first?.focus() }
}
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
const matchClubName = computed(() => overview.value.club.name || 'Società')
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
const filteredMatches = computed(() => filterItems('matches', overview.value.matches ?? [], match => [matchClubName.value, match.opponent, match.competition, match.location, teamName(match.teamId)]))
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
function navigationResize() { if (window.innerWidth > 900) mobileMenuOpen.value = false }
onMounted(() => { loadOverview(); window.addEventListener('beforeunload', beforeUnload); window.addEventListener('resize', navigationResize) })
onUnmounted(() => { window.removeEventListener('beforeunload', beforeUnload); window.removeEventListener('resize', navigationResize) })
</script>

<template>
  <main class="club-admin" :class="{ 'club-admin--collapsed': sidebarCollapsed, 'club-admin--menu-open': authenticated && !checking && mobileMenuOpen }">
    <link v-for="stylesheet in typography.stylesheets" :key="stylesheet" rel="stylesheet" :href="stylesheet" />
    <section v-if="checking" class="center" role="status"><span class="loading-ring" aria-hidden="true" />Caricamento del workspace…</section>
    <section v-else-if="!authenticated" class="login-shell">
      <div class="login-card"><p class="eyebrow">ARENABOOSTX CLUB</p><h1>Gestisci il tuo Club Hub</h1><p>Homepage, squadre, roster, annunci, fotografie, calendario e sponsor.</p><form @submit.prevent="loginSubmit"><label>Username<input v-model="login.username" autocomplete="username" /></label><label>Password<input v-model="login.password" type="password" autocomplete="current-password" /></label><button :disabled="processing">{{busy?'Accesso…':'Entra nel CMS'}}</button></form><p v-if="error" class="error">{{error}}</p></div>
    </section>
    <template v-else>
      <aside id="club-cms-navigation" ref="navAside" :role="mobileMenuOpen ? 'dialog' : undefined" :aria-modal="mobileMenuOpen || undefined" aria-label="Navigazione Club CMS" @keydown="navigationKeydown">
        <div class="sidebar-top"><a class="admin-brand" :href="publicPath" target="_blank" rel="noopener"><span>AX</span><div><strong>ArenaBoost<span>X</span></strong><small>Club workspace</small></div></a><button class="mobile-close icon-button" type="button" aria-label="Chiudi navigazione" @click="closeMobileMenu">×</button></div>
        <div class="club-identity"><span class="club-avatar">{{ (overview.club.name || 'C').slice(0, 2).toUpperCase() }}</span><div><strong>{{overview.club.name || 'La tua società'}}</strong><small>{{overview.club.seasonLabel || 'Club Hub'}}</small></div><span class="identity-dot" aria-hidden="true" /></div>
        <nav aria-label="Sezioni CMS"><div v-for="group in navGroups" :key="group.label" class="nav-group"><p class="nav-group-label">{{ group.label }}</p><button v-for="tab in tabs.filter(item => group.ids.includes(item[0]))" :key="tab[0]" type="button" :disabled="processing" :class="{active:activeTab===tab[0]}" :title="tab[1]" :aria-current="activeTab===tab[0] ? 'page' : undefined" @click="selectTab(tab[0])"><svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.65" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path :d="navIcons[tab[0]]" /></svg><span class="nav-label">{{tab[1]}}</span><span v-if="activeTab===tab[0]" class="nav-active-dot" aria-hidden="true" /></button></div></nav>
        <div class="aside-bottom"><p v-if="hasUnsavedChanges" class="draft-indicator" role="status"><span aria-hidden="true">●</span> <span class="footer-label">Modifiche da salvare</span></p><a :href="publicPath" target="_blank" rel="noopener" title="Apri sito pubblico"><svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" aria-hidden="true"><path d="M15 3h6v6 M21 3l-9 9 M10 3H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-5" /></svg><span class="footer-label">Apri sito pubblico</span></a><button type="button" :disabled="processing" title="Esci dal CMS" @click="logout"><svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" aria-hidden="true"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4 M16 17l5-5-5-5 M21 12H9" /></svg><span class="footer-label">Esci dal CMS</span></button><button class="collapse-button" type="button" :aria-expanded="!sidebarCollapsed" :title="sidebarCollapsed ? 'Espandi menu' : 'Comprimi menu'" @click="sidebarCollapsed = !sidebarCollapsed"><svg class="nav-icon" :class="{ flipped: sidebarCollapsed }" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" aria-hidden="true"><path d="m14 7-5 5 5 5 M20 3v18" /></svg><span class="footer-label">Comprimi menu</span></button></div>
      </aside>
      <button v-if="mobileMenuOpen" class="mobile-backdrop" type="button" aria-label="Chiudi navigazione" tabindex="-1" @click="closeMobileMenu" />
      <section class="workspace" :inert="mobileMenuOpen || undefined">
        <div class="workspace-topbar"><div><button ref="mobileMenuTrigger" class="mobile-menu icon-button" type="button" aria-controls="club-cms-navigation" :aria-expanded="mobileMenuOpen" aria-label="Apri navigazione" @click="openMobileMenu"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" aria-hidden="true"><path d="M4 6h16 M4 12h16 M4 18h16" /></svg></button><span>Workspace</span><span aria-hidden="true">/</span><strong>{{tabs.find(tab=>tab[0]===activeTab)?.[1]}}</strong></div><span class="workspace-context">Club CMS</span></div>
        <header><div><p class="eyebrow">{{overview.club.seasonLabel || 'Gestione della società'}}</p><h1>{{tabs.find(tab=>tab[0]===activeTab)?.[1]}}</h1><p class="page-description">{{ descriptions[activeTab] }}</p></div><a class="public-link" :href="publicPath" target="_blank" rel="noopener">Apri Club Hub <span aria-hidden="true">↗</span></a></header>
        <p v-if="notice" class="notice" role="status">{{notice}}</p><p v-if="error" class="error error-banner" role="alert">{{error}}<button type="button" aria-label="Chiudi messaggio di errore" @click="error=''">×</button></p>
        <p v-if="imageUploads" class="processing-banner" role="status">Ottimizzazione dell’immagine in corso. Attendi prima di salvare.</p>

        <section v-if="collectionTabs.includes(activeTab) && (activeTab !== 'teams' || !rosterTeamId)" class="collection-toolbar" aria-label="Ricerca e filtri">
          <label class="search-field">Cerca {{ activeTab === 'teams' ? 'squadre' : activeTab === 'posts' ? 'contenuti' : activeTab === 'gallery' ? 'fotografie' : activeTab === 'sponsors' ? 'sponsor' : 'partite' }}<input v-model="filters[activeTab].query" type="search" placeholder="Digita per cercare…" /></label>
          <label v-if="activeTab !== 'teams' && activeTab !== 'sponsors'">Squadra<select v-model.number="filters[activeTab].teamId"><option :value="0">Tutte le squadre</option><option v-for="team in teams" :key="team.id" :value="team.id">{{team.name}}</option></select></label>
          <label>Stato<select v-model="filters[activeTab].state"><option value="">Tutti gli stati</option><template v-if="activeTab === 'matches'"><option v-for="(label, value) in statusLabels" :key="value" :value="value">{{ label }}</option></template><template v-else><option value="true">{{ activeTab === 'sponsors' ? 'Pubblicato' : 'Visibile' }}</option><option value="false">{{ activeTab === 'sponsors' ? 'Bozza' : 'Nascosto' }}</option></template></select></label>
          <div class="filter-summary"><span aria-live="polite">{{ resultCount }} risultati</span><button v-if="filters[activeTab].query || filters[activeTab].teamId || filters[activeTab].state" type="button" @click="resetFilters">Azzera filtri</button></div>
        </section>

        <div v-if="activeTab==='dashboard'" class="club-dashboard">
          <section class="card welcome"><div><span class="welcome-kicker"><span aria-hidden="true">●</span> Il tuo Club Hub</span><h2>{{overview.club.name || 'Il tuo club, al centro.'}}</h2><p>{{overview.club.tagline||'Dai spazio alla tua società. Squadre, storie e appuntamenti, in un solo workspace.'}}</p><button class="welcome-action" type="button" @click="activeTab='posts'">Crea un contenuto <span aria-hidden="true">↗</span></button></div><div class="welcome-side"><span class="welcome-mark" aria-hidden="true">AX</span><a :href="publicPath" target="_blank" rel="noopener">Visualizza sito <span aria-hidden="true">↗</span></a></div></section>
          <div class="dashboard-section-heading"><div><h2>La società in numeri</h2><p>I contenuti e le risorse presenti nel tuo workspace.</p></div><span class="badge" :class="{ muted: !overview.club.active }">{{ overview.club.active ? 'Sito pubblico attivo' : 'Sito pubblico disattivato' }}</span></div>
          <div class="stats"><article v-for="entry in [['Squadre',overview.counts.teams,'teams'],['Atleti',overview.counts.players,'teams'],['Contenuti',overview.counts.posts,'posts'],['Foto',overview.counts.photos,'gallery'],['Partite',overview.counts.matches,'matches'],['Sponsor',overview.counts.sponsors,'sponsors']]" :key="entry[0]"><div><span>{{entry[0]}}</span><svg class="stat-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.65" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path :d="navIcons[entry[2]]" /></svg></div><strong>{{entry[1]||0}}</strong><small>Nel Club Hub</small></article></div>
          <section class="card quick-panel"><div class="panel-heading"><div><h2>Azioni rapide</h2><p class="form-hint">Dalla gestione alla pubblicazione, un passo alla volta.</p></div><span class="panel-kicker">Il tuo workspace</span></div><div class="quick"><button v-for="action in [['settings','Personalizza homepage','Identità, stile e immagini del club'],['teams','Squadre e roster','Tutti gli atleti, sempre aggiornati'],['matches','Gestisci il calendario','Incontri, risultati e parziali'],['posts','Pubblica un annuncio','Le novità da condividere con i tifosi']]" :key="action[0]" type="button" @click="activeTab=action[0]"><span class="quick-icon"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.65" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path :d="navIcons[action[0]]" /></svg></span><strong>{{action[1]}}</strong><small>{{action[2]}}</small><span class="quick-arrow" aria-hidden="true">↗</span></button></div></section>
          <div class="dashboard-detail-grid"><section class="card dashboard-list"><div class="panel-heading"><h2>Dal calendario</h2><button type="button" class="text-button" @click="activeTab='matches'">Vedi tutto <span aria-hidden="true">→</span></button></div><p v-if="!overview.matches?.length" class="empty-state">Il calendario è pronto per la tua stagione. Inserisci il primo incontro.</p><article v-for="match in (overview.matches || []).slice(0, 3)" :key="match.id"><span class="dashboard-list-icon"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" aria-hidden="true"><path :d="navIcons.matches" /></svg></span><div><strong>{{matchClubName}} · {{match.opponent}}</strong><small>{{formatDate(match.scheduledAt)}}</small></div><span class="badge" :class="match.status">{{statusLabels[match.status] || match.status}}</span></article></section><section class="card dashboard-list"><div class="panel-heading"><h2>Le comunicazioni</h2><button type="button" class="text-button" @click="activeTab='posts'">Vedi tutto <span aria-hidden="true">→</span></button></div><p v-if="!overview.posts?.length" class="empty-state">Le tue storie iniziano qui. Crea la prima comunicazione per il club.</p><article v-for="post in (overview.posts || []).slice(0, 3)" :key="post.id"><span class="dashboard-list-icon"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" aria-hidden="true"><path :d="navIcons.posts" /></svg></span><div><strong>{{post.title}}</strong><small>{{formatDate(post.publishedAt)}}</small></div><span class="badge" :class="{muted:!post.active}">{{post.active ? 'Visibile' : 'Bozza'}}</span></article></section></div>
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
          <div class="form-grid"><label>Nome del club *<input v-model="settings.name" required maxlength="200" /></label><label>Città<input v-model="settings.city" placeholder="Es. Bologna" /></label><label class="wide">Claim<input v-model="settings.tagline" placeholder="Il messaggio che racconta il tuo club" /></label><label>Stagione<input v-model="settings.seasonLabel" placeholder="Es. 2026 / 2027" /></label><label>Colore primario<input v-model="settings.primaryColor" type="color" /></label><label>Colore accento<input v-model="settings.secondaryColor" type="color" /></label><label class="check"><input v-model="settings.active" type="checkbox" /><span class="checkbox-box" aria-hidden="true"><svg viewBox="0 0 16 16" fill="none"><path d="m3 8 3 3 7-7" /></svg></span><span class="checkbox-text">Sito pubblico attivo</span></label>
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
                  <label class="check"><input v-model="teamForm.active" type="checkbox" /><span class="checkbox-box" aria-hidden="true"><svg viewBox="0 0 16 16" fill="none"><path d="m3 8 3 3 7-7" /></svg></span><span class="checkbox-text">Visibile sul sito pubblico</span></label>
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
                <div class="toggle-group"><label class="check"><input v-model="postForm.pinned" type="checkbox" /><span class="checkbox-box" aria-hidden="true"><svg viewBox="0 0 16 16" fill="none"><path d="m3 8 3 3 7-7" /></svg></span><span class="checkbox-text">In evidenza</span></label><label class="check"><input v-model="postForm.active" type="checkbox" /><span class="checkbox-box" aria-hidden="true"><svg viewBox="0 0 16 16" fill="none"><path d="m3 8 3 3 7-7" /></svg></span><span class="checkbox-text">Visibile sul sito pubblico</span></label></div>
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
                <label class="check"><input v-model="galleryForm.active" type="checkbox" /><span class="checkbox-box" aria-hidden="true"><svg viewBox="0 0 16 16" fill="none"><path d="m3 8 3 3 7-7" /></svg></span><span class="checkbox-text">Visibile sul sito pubblico</span></label>
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
                <label class="check"><input v-model="sponsorForm.active" type="checkbox" /><span class="checkbox-box" aria-hidden="true"><svg viewBox="0 0 16 16" fill="none"><path d="m3 8 3 3 7-7" /></svg></span><span class="checkbox-text">Pubblicato sul sito pubblico</span></label>
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
                <label>Società<input :value="matchClubName" readonly /></label>
                <label>Avversario *<input v-model="matchForm.opponent" required placeholder="Nome della squadra avversaria" /></label>
                <label class="wide">Categoria / squadra *<select v-model.number="matchForm.teamId" required><option :value="0" disabled>Seleziona una categoria / squadra</option><option v-for="team in teams" :key="team.id" :value="team.id">{{team.name}}</option></select><small>Associa l’incontro al calendario della categoria.</small></label>
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
                  <label>Risultato · {{ matchClubName }}<input v-model.number="matchForm.scoreFor" type="number" min="0" step="1" required /></label>
                  <label>Risultato · {{ matchForm.opponent || 'Avversario' }}<input v-model.number="matchForm.scoreAgainst" type="number" min="0" step="1" required /></label>
                  <p class="form-hint wide">Inserisci il risultato complessivo (set vinti per la pallavolo). I parziali sono facoltativi e seguono sempre l’ordine squadra del club / avversario, anche in trasferta.</p>
                  <div class="set-editor wide">
                    <div class="panel-heading"><h3>Parziali dell’incontro</h3><span class="badge">{{matchForm.sets.length}}</span></div>
                    <p v-if="!matchForm.sets.length" class="form-hint">Nessun parziale. Aggiungi una riga per ogni set o periodo di gioco.</p>
                    <div v-for="(set, index) in matchForm.sets" :key="set.key" class="set-row">
                      <span class="set-number">{{index + 1}}°</span>
                      <label>{{matchClubName}}<input v-model.number="set.scoreFor" type="number" min="0" step="1" required :aria-label="`Parziale ${index + 1}, squadra del club`" placeholder="0" /></label>
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
              <h3>{{matchClubName}} <span class="versus">vs</span> {{match.opponent}}</h3>
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
* { box-sizing: border-box; }
.club-admin { --club-admin-canvas: #f6f7fb; --club-admin-surface: #fff; --club-admin-ink: #182238; --club-admin-secondary: #526078; --club-admin-border: #e1e5ee; --club-admin-accent: #4f46e5; min-height: 100vh; background: var(--club-admin-canvas); color: var(--club-admin-ink); font-family: Inter, 'IBM Plex Sans', ui-sans-serif, system-ui, sans-serif; font-size: 14px; line-height: 1.5; color-scheme: light; }
.club-admin h1, .club-admin h2, .club-admin h3 { color: var(--club-admin-ink); font-family: inherit; text-transform: none; letter-spacing: -.025em; }
.club-admin button, .club-admin input, .club-admin select, .club-admin textarea { font: inherit; }
.club-admin button { min-height: 40px; border: 1px solid #d6dce8; border-radius: 8px; padding: 9px 14px; background: #fff; color: #34415a; font-size: 13px; font-weight: 600; line-height: 1.45; cursor: pointer; transition: background .15s, border-color .15s, box-shadow .15s; }
.club-admin button:hover:not(:disabled) { background: #f3f4fc; border-color: #a5a8e7; }
.club-admin button:disabled { opacity: .55; cursor: not-allowed; }
.club-admin button.primary, .login-card button { background: #4f46e5; border-color: #4f46e5; color: #fff; padding: 11px 18px; box-shadow: 0 2px 3px #312e8112; }
.club-admin button.primary:hover:not(:disabled), .login-card button:hover:not(:disabled) { background: #4338ca; border-color: #4338ca; }
.club-admin button.danger { color: #b42338; }
.club-admin button.danger:hover:not(:disabled) { background: #fff1f2; border-color: #f2a9b2; }
.club-admin button.danger-filled { background: #bd2541; color: #fff; border-color: #bd2541; }
.club-admin button.danger-filled:hover:not(:disabled) { background: #a61d36; }
.club-admin button:focus-visible, .club-admin a:focus-visible, .club-admin summary:focus-visible, .club-admin input:focus-visible, .club-admin select:focus-visible, .club-admin textarea:focus-visible { outline: 3px solid #818cf8; outline-offset: 3px; }
.club-admin label, .club-admin legend { min-width: 0; max-width: 100%; overflow-wrap: anywhere; }
.club-admin input, .club-admin select, .club-admin textarea { min-width: 0; max-width: 100%; color-scheme: light; }
.club-admin input::placeholder, .club-admin textarea::placeholder { color: #69758b; opacity: 1; }
.club-admin input:disabled, .club-admin textarea:disabled, .club-admin select:disabled { background: #f1f3f7; color: #64748b; cursor: not-allowed; }
.club-admin select option { background: #fff; color: #182238; }
.club-admin select option:disabled { color: #69758b; }
.center, .login-shell { min-height: 100vh; display: grid; place-content: center; justify-items: center; gap: 18px; }
.center { color: #526078; }
.loading-ring { width: 32px; height: 32px; border: 3px solid #e0e3f4; border-top-color: #4f46e5; border-radius: 50%; animation: spin .8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
.login-shell { padding: 24px; background: radial-gradient(ellipse at 20% 0%, #e8e7ff, transparent 55%), #f6f7fb; }
.login-card { width: min(460px, calc(100vw - 48px)); padding: 40px; border: 1px solid #dfe3ef; border-radius: 20px; background: #fff; box-shadow: 0 20px 70px #27244d0d; }
.login-card h1 { margin: 12px 0; font-size: 32px; line-height: 1.18; }
.login-card > p { color: #526078; }
.login-card form { display: grid; gap: 18px; margin-top: 28px; }
.login-card label { display: grid; gap: 7px; color: #34415a; font-size: 13px; font-weight: 600; }
.login-card input { width: 100%; min-height: 46px; border: 1px solid #cbd2e0; border-radius: 8px; padding: 10px 12px; color: #182238; background: #fff; }
.eyebrow { color: #6359bd; margin: 0; font-size: 10px; font-weight: 700; letter-spacing: .12em; text-transform: uppercase; }
.club-admin > aside { position: fixed; z-index: 60; inset: 0 auto 0 0; width: 252px; height: 100vh; height: 100dvh; padding: 22px 16px 14px; display: flex; flex-direction: column; background: #111827; color: #e5e7ef; border-right: 1px solid #202b40; }
.sidebar-top { display: flex; align-items: center; justify-content: space-between; gap: 8px; margin-bottom: 26px; }
.admin-brand { display: flex; align-items: center; gap: 10px; color: #fff; text-decoration: none; min-width: 0; }
.admin-brand > span { display: grid; place-items: center; flex: none; width: 37px; height: 37px; border-radius: 10px; background: #6155e9; color: #fff; font-weight: 800; font-size: 15px; letter-spacing: -.08em; }
.admin-brand > div { display: grid; gap: 1px; min-width: 0; }
.admin-brand strong { font-size: 16px; font-weight: 650; letter-spacing: -.04em; }
.admin-brand strong > span { color: #a69dfa; }
.admin-brand small { color: #a8b3c6; font-size: 10px; letter-spacing: .04em; }
.club-identity { display: flex; align-items: center; gap: 10px; padding: 12px 10px; border: 1px solid #2c374c; border-radius: 10px; margin-bottom: 26px; background: #192234; }
.club-avatar { display: grid; place-items: center; flex: none; width: 32px; height: 32px; border-radius: 8px; background: #353255; color: #d6ceff; font-size: 11px; font-weight: 750; }
.club-identity > div { min-width: 0; display: grid; gap: 2px; flex: 1; }
.club-identity strong { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 12px; font-weight: 600; }
.club-identity small { color: #b0bbcd; font-size: 10px; }
.identity-dot { width: 6px; height: 6px; border-radius: 50%; background: #7b82ac; }
.club-admin nav { min-height: 0; flex: 1; overflow-y: auto; scrollbar-width: thin; scrollbar-color: #3b465e transparent; }
.nav-group + .nav-group { margin-top: 24px; }
.nav-group-label { margin: 0 12px 10px; color: #9aa8c0; text-transform: uppercase; font-size: 9px; font-weight: 650; letter-spacing: .14em; }
.club-admin nav button { display: flex; align-items: center; gap: 12px; width: 100%; padding: 11px 12px; margin: 3px 0; min-height: 43px; border: 1px solid transparent; background: transparent; color: #b8c2d4; text-align: left; font-size: 12px; font-weight: 500; }
.club-admin nav button:hover:not(:disabled) { color: #fff; background: #202b40; border-color: transparent; }
.club-admin nav button.active { color: #fff; background: #4f46e5; border-color: #635be7; box-shadow: 0 4px 12px #00000015; }
.nav-icon { flex: none; width: 18px; height: 18px; }
.nav-label { flex: 1; }
.nav-active-dot { flex: none; width: 5px; height: 5px; border-radius: 50%; background: #d3ceff; }
.aside-bottom { display: grid; gap: 4px; padding-top: 14px; margin-top: 18px; border-top: 1px solid #2b3549; }
.aside-bottom > a, .club-admin .aside-bottom > button { display: flex; align-items: center; gap: 12px; min-height: 40px; background: transparent; border: 1px solid transparent; color: #b8c2d4; border-radius: 8px; padding: 10px 12px; text-decoration: none; font-size: 12px; font-weight: 500; text-align: left; }
.aside-bottom > a:hover, .club-admin .aside-bottom > button:hover:not(:disabled) { background: #202b40; color: #fff; border-color: transparent; }
.club-admin .aside-bottom .collapse-button { min-height: 34px; padding-top: 7px; padding-bottom: 7px; margin-top: 7px; color: #9aa8c0; }
.flipped { transform: rotate(180deg); }
.draft-indicator { display: flex; align-items: center; gap: 9px; color: #f7d788; padding: 0 12px; font-size: 11px; }
.workspace { margin-left: 252px; min-width: 0; min-height: 100vh; padding: 0 clamp(22px, 3vw, 48px) 48px; }
.workspace-topbar { display: flex; align-items: center; justify-content: space-between; gap: 16px; min-height: 68px; margin: 0 calc(-1 * clamp(22px, 3vw, 48px)) 32px; padding: 12px clamp(22px, 3vw, 48px); border-bottom: 1px solid #e3e7ef; background: #fff; color: #647087; font-size: 11px; }
.workspace-topbar > div { display: flex; gap: 14px; align-items: center; min-width: 0; }
.workspace-topbar strong { font-size: 11px; font-weight: 550; color: #34415a; }
.workspace-context { border: 1px solid #e0e5ed; border-radius: 6px; padding: 5px 8px; font-size: 10px; white-space: nowrap; }
.workspace > header { display: flex; align-items: center; justify-content: space-between; gap: 24px; margin-bottom: 28px; }
.workspace > header > div { min-width: 0; }
.workspace h1 { font-size: clamp(26px, 2.6vw, 34px); line-height: 1.22; font-weight: 650; margin: 5px 0 8px; }
.page-description { margin: 0; color: #606d82; font-size: 13px; max-width: 660px; line-height: 1.65; }
.public-link { display: inline-flex; align-items: center; gap: 12px; flex-shrink: 0; border: 1px solid #d5dce8; border-radius: 8px; background: #fff; padding: 10px 14px; color: #34415a; text-decoration: none; font-size: 12px; font-weight: 600; }
.public-link:hover { color: #4338ca; border-color: #a5a8e7; background: #f9f9ff; }
.card { min-width: 0; margin: 0 0 22px; padding: 25px; background: #fff; border: 1px solid #e0e5ee; border-radius: 13px; box-shadow: 0 2px 3px #10182803; }
.card h2 { margin: 0; font-size: 18px; font-weight: 650; line-height: 1.4; }
.card h3 { font-size: 15px; font-weight: 600; }
.welcome { position: relative; display: flex; align-items: center; justify-content: space-between; gap: 28px; padding: 30px 34px; overflow: hidden; background: linear-gradient(112deg, #29295c, #4542a6); border: 0; color: #fff; }
.welcome::after { position: absolute; content: ''; pointer-events: none; right: -90px; top: -190px; width: 380px; height: 380px; border: 1px solid #ffffff14; border-radius: 50%; box-shadow: 0 0 0 42px #ffffff04, 0 0 0 84px #ffffff04; }
.welcome > div { position: relative; z-index: 1; }
.welcome-kicker { display: inline-flex; align-items: center; gap: 8px; color: #e0dfff; border: 1px solid #ffffff30; background: #ffffff0b; border-radius: 20px; padding: 4px 10px; font-size: 10px; }
.welcome-kicker > span { color: #b9afff; font-size: 8px; }
.welcome h2 { color: #fff; font-size: clamp(24px, 2.7vw, 32px); font-weight: 600; margin: 13px 0 9px; }
.welcome p { max-width: 520px; color: #e0dff5; font-size: 12px; line-height: 1.8; margin: 0 0 18px; }
.club-admin .welcome-action { background: #fff; color: #373191; border: 1px solid #fff; min-height: 39px; padding: 9px 13px; font-size: 11px; }
.club-admin .welcome-action:hover { background: #eeedff; border-color: #eeedff; }
.welcome-action > span { margin-left: 16px; }
.welcome-side { display: grid; justify-items: center; gap: 12px; min-width: 135px; }
.welcome-mark { display: grid; place-items: center; width: 94px; height: 94px; border: 1px solid #ffffff25; border-radius: 26px; background: #ffffff0a; transform: rotate(-7deg); color: #e4e0ff; font-size: 36px; letter-spacing: -.1em; font-weight: 750; }
.welcome-side a { color: #ecebff; font-size: 11px; text-decoration: none; }
.welcome-side a > span { margin-left: 10px; }
.dashboard-section-heading, .panel-heading, .editor-heading { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 14px; }
.dashboard-section-heading { margin: 30px 0 16px; }
.dashboard-section-heading h2 { margin: 0 0 4px; font-size: 17px; font-weight: 650; }
.dashboard-section-heading p { margin: 0; font-size: 12px; color: #647087; }
.stats { display: grid; grid-template-columns: repeat(6, minmax(0, 1fr)); gap: 13px; margin-bottom: 24px; }
.stats article { min-width: 0; padding: 18px; background: #fff; border: 1px solid #e0e5ee; border-radius: 11px; }
.stats article > div { display: flex; justify-content: space-between; align-items: center; gap: 8px; }
.stats span { color: #526078; font-size: 11px; font-weight: 550; }
.stat-icon { width: 16px; height: 16px; color: #7b74bb; }
.stats strong { display: block; font-size: 29px; letter-spacing: -.045em; font-weight: 650; font-variant-numeric: tabular-nums; line-height: 1.3; margin: 13px 0 4px; }
.stats small { color: #69758b; font-size: 9px; }
.quick-panel .panel-heading { margin-bottom: 20px; }
.quick-panel .form-hint { margin: 5px 0 0; font-size: 12px; }
.panel-kicker { color: #69758b; font-size: 10px; }
.quick { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 14px; }
.club-admin .quick > button { position: relative; display: flex; flex-direction: column; align-items: flex-start; gap: 8px; padding: 18px; background: #fff; border: 1px solid #e0e5ee; border-radius: 9px; text-align: left; }
.club-admin .quick > button:hover { border-color: #a8a2f1; background: #fafaff; box-shadow: 0 4px 16px #4f46e508; }
.quick-icon { display: grid; place-items: center; width: 36px; height: 36px; border: 1px solid #e3e0fc; background: #f1efff; border-radius: 9px; color: #6256c3; margin-bottom: 4px; }
.quick-icon svg { width: 18px; height: 18px; }
.quick strong { font-size: 11px; font-weight: 600; color: #26334b; }
.quick small { font-size: 10px; font-weight: 400; color: #647087; line-height: 1.6; }
.quick-arrow { position: absolute; top: 17px; right: 17px; color: #8a94a8; font-size: 15px; }
.dashboard-detail-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 22px; }
.dashboard-list { margin-bottom: 0; }
.dashboard-list .panel-heading { margin-bottom: 18px; gap: 10px; }
.dashboard-list h2 { font-size: 15px; }
.club-admin .text-button { border: 0; background: transparent; color: #5144bb; padding: 5px 0; font-size: 10px; min-height: 30px; }
.club-admin .text-button:hover { background: transparent; color: #3730a3; }
.dashboard-list > article { display: flex; align-items: center; gap: 12px; padding: 15px 0; border-top: 1px solid #eef0f5; }
.dashboard-list > article > div { flex: 1; min-width: 0; }
.dashboard-list > article strong { display: block; overflow-wrap: anywhere; color: #34415a; font-size: 11px; font-weight: 600; }
.dashboard-list > article small { display: block; margin-top: 4px; color: #69758b; font-size: 10px; }
.dashboard-list-icon { width: 34px; height: 34px; flex: none; display: grid; place-items: center; background: #f4f5fa; color: #737f98; border-radius: 8px; }
.dashboard-list-icon svg { width: 16px; height: 16px; }
.dashboard-list .empty-state { font-size: 12px; padding: 25px 16px; }
.collection-toolbar { display: flex; align-items: flex-end; flex-wrap: wrap; gap: 16px; padding: 18px 20px; margin-bottom: 24px; border: 1px solid #e0e5ee; border-radius: 11px; background: #fff; }
.collection-toolbar label { flex: 1 1 145px; }
.collection-toolbar .search-field { flex: 2 1 230px; }
.collection-toolbar label, .form-grid label { min-width: 0; display: flex; flex-direction: column; gap: 8px; color: #34415a; font-size: 12px; font-weight: 600; }
.collection-toolbar input, .collection-toolbar select, .form-grid input:not([type=checkbox]), .form-grid textarea, .form-grid select { display: block; width: 100%; min-width: 0; min-height: 44px; border: 1px solid #cbd2df; border-radius: 8px; padding: 10px 12px; color: #182238; background: #fff; font: inherit; font-size: 13px; font-weight: 400; line-height: 1.5; transition: border-color .15s, box-shadow .15s; }
.collection-toolbar input:hover, .collection-toolbar select:hover, .form-grid input:hover:not(:disabled), .form-grid textarea:hover:not(:disabled), .form-grid select:hover:not(:disabled) { border-color: #9aa6bd; }
.collection-toolbar input:focus, .collection-toolbar select:focus, .form-grid input:focus, .form-grid textarea:focus, .form-grid select:focus { border-color: #7062df; box-shadow: 0 0 0 3px #6155e912; }
.form-grid input[readonly] { background: #f4f5f9; color: #526078; }
.form-grid textarea { resize: vertical; }
.form-grid input[type=color] { padding: 5px; cursor: pointer; }
.form-grid label small { font-size: 11px; font-weight: 400; line-height: 1.6; color: #647087; }
.filter-summary { display: flex; align-items: center; justify-content: flex-end; flex: 1 1 140px; gap: 10px; min-height: 44px; color: #647087; font-size: 11px; }
.club-admin .filter-summary button { border: 0; background: transparent; color: #5144bb; font-size: 11px; padding: 6px; white-space: nowrap; }
.editor-layout { display: grid; grid-template-columns: minmax(0, 1.2fr) minmax(300px, .85fr); align-items: start; gap: 24px; }
.editor-layout.roster-open { grid-template-columns: minmax(0, 1fr); }
.editor-heading { align-items: flex-start; }
.editor-heading > div { min-width: 0; }
.editor-heading h2 { overflow-wrap: anywhere; }
.editor-heading .eyebrow { margin-bottom: 8px; }
.required-hint { font-size: 11px; margin: 13px 0 26px; }
.form-hint { color: #647087; font-size: 12px; line-height: 1.7; }
.form-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(100%, 230px), 1fr)); gap: 18px 16px; }
.form-grid > * { min-width: 0; }
.form-grid.single { grid-template-columns: minmax(0, 1fr); }
.form-grid .wide { grid-column: 1 / -1; }
.form-body { min-width: 0; border: 0; padding: 0; margin: 0; }
.form-section { min-width: 0; border: 0; border-top: 1px solid #e2e6ef; padding: 22px 0 0; margin: 0 0 28px; }
.form-section legend { color: #34415a; padding: 0 12px 0 0; font-size: 13px; font-weight: 600; }
.form-section legend > span { color: #7566c9; font-size: 10px; font-weight: 600; padding-right: 8px; }
.settings-editor .form-section { margin-top: 28px; }
.typography-editor { min-width: 0; border: 1px solid #e0e5ee; border-radius: 10px; padding: 20px; margin: 24px 0 28px; background: #fbfbfe; }
.typography-editor legend { padding: 0 8px; font-size: 15px; color: #34415a; font-weight: 600; }
.typography-preview { margin-top: 20px; padding: 24px; border: 1px solid #e0e5ee; border-radius: 9px; color: #182238; background: #fff; font-family: var(--club-font-body); }
.typography-preview small { color: #647087; font-size: 10px; }
.typography-preview h3 { margin: 12px 0; font-family: var(--club-font-title); font-size: clamp(22px, 3vw, 32px); line-height: 1.25; }
.typography-preview p { margin: 0; font-size: 14px; line-height: 1.7; }
.club-admin .font-select { appearance: auto; font-size: 14px; }
.tile-icons-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 14px; margin: 20px 0 24px; }
.contact-editor, .player-editor { min-width: 0; border: 1px solid #dce2ec; border-radius: 11px; padding: 20px; margin: 24px 0; background: #fbfcfe; }
.contact-editor legend, .player-editor legend { padding: 0 8px; font-size: 13px; font-weight: 600; color: #34415a; }
.contact-actions, .row-actions { display: flex; flex-wrap: wrap; justify-content: flex-end; gap: 8px; margin-bottom: 18px; }
.editor-footer { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 14px; border-top: 1px solid #e2e6ef; padding-top: 20px; margin-top: 5px; }
.editor-footer > span { max-width: 65%; color: #647087; font-size: 11px; }
.editor-footer button { max-width: 100%; white-space: normal; overflow-wrap: anywhere; }
.badge { display: inline-flex; align-items: center; justify-content: center; gap: 5px; padding: 4px 8px; border: 1px solid #dedbf8; border-radius: 6px; background: #f0edff; color: #5d46a6; font-size: 10px; font-weight: 600; white-space: nowrap; line-height: 1.5; }
.badge.muted { background: #f2f4f8; border-color: #e0e5ee; color: #606d82; }
.badge.completed { background: #ecf8f1; border-color: #c5e9d6; color: #21734b; }
.badge.live { background: #fff0f2; border-color: #f5c4ce; color: #b42348; }
.badge.scheduled { background: #eef3ff; border-color: #d4e0ff; color: #3d5aa8; }
.records-panel > .panel-heading { margin-bottom: 20px; }
.records-panel h2 { font-size: 16px; }
.records-list { display: grid; gap: 14px; }
.record-card { min-width: 0; border: 1px solid #e1e6ef; border-radius: 10px; padding: 18px; background: #fff; transition: border-color .15s; }
.record-card:hover { border-color: #bcc4d6; }
.record-card.selected { border-color: #8c7ee9; box-shadow: inset 3px 0 #6b5cdd; background: #fdfcff; }
.record-meta { display: flex; align-items: center; justify-content: space-between; gap: 8px; flex-wrap: wrap; color: #647087; font-size: 10px; }
.record-card h3 { margin: 14px 0 6px; color: #26334b; font-size: 15px; line-height: 1.5; overflow-wrap: anywhere; }
.record-card > p { margin: 5px 0; color: #606d82; font-size: 12px; line-height: 1.7; overflow-wrap: anywhere; }
.record-excerpt { display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
.record-actions { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 16px; padding-top: 14px; border-top: 1px solid #edf0f5; }
.record-actions button { flex: 1 0 auto; font-size: 11px; padding: 8px 10px; }
.club-admin .record-actions .danger { flex: 0 0 auto; background: transparent; border-color: transparent; }
.empty-state { margin: 16px 0; padding: 28px 20px; border: 1px dashed #cdd5e4; border-radius: 9px; background: #fafbfe; color: #606d82; font-size: 12px; line-height: 1.8; text-align: center; }
.error { color: #b42338; }
.error-banner { display: flex; justify-content: space-between; align-items: center; gap: 12px; margin-bottom: 20px; border: 1px solid #f1bcc7; border-radius: 9px; padding: 13px 16px; background: #fff0f3; color: #a51d38; font-size: 13px; }
.club-admin .error-banner button { min-height: 32px; padding: 0 7px; border: 0; background: transparent; color: inherit; font-size: 24px; }
.notice { position: fixed; z-index: 80; right: 24px; top: 20px; max-width: calc(100vw - 48px); padding: 12px 18px; border: 1px solid #b9dfce; border-radius: 9px; color: #1c6b47; background: #eefaf3; box-shadow: 0 8px 30px #18223814; font-size: 13px; }
.processing-banner { padding: 13px 16px; margin-bottom: 20px; border: 1px solid #ccd9f9; border-radius: 9px; color: #355395; background: #eff4ff; font-size: 12px; }
.roster-editor { grid-column: 1 / -1; }
.roster-summary { display: flex; gap: 14px; align-items: center; flex-wrap: wrap; margin: 24px 0; color: #647087; font-size: 11px; }
.player-grid { grid-template-columns: repeat(auto-fit, minmax(min(100%, 180px), 1fr)); }
.add-row { padding: 12px 0 24px; }
.club-admin .add-row button, .club-admin .add-set { border-style: dashed; border-color: #c3bbed; background: #f8f6ff; color: #5943b4; }
.set-editor { padding: 18px; border: 1px solid #dce2ec; border-radius: 9px; background: #f8f9fc; }
.set-editor h3 { margin: 0; font-size: 13px; }
.set-row { display: grid; grid-template-columns: 28px minmax(0, 1fr) 8px minmax(0, 1fr) 30px; align-items: end; gap: 8px; margin: 18px 0; }
.set-row > label { font-size: 11px; }
.set-number { line-height: 44px; color: #647087; font-weight: 600; }
.set-row > span:not(.set-number) { line-height: 44px; text-align: center; }
.club-admin .set-row button { height: 44px; padding: 8px 0; font-size: 20px; border-color: transparent; background: transparent; }
.set-row input { text-align: center; font-variant-numeric: tabular-nums; }
.add-set { margin-top: 8px; width: 100%; }
.match-result { margin-top: 16px; color: #26334b; font-size: 26px; font-weight: 650; font-variant-numeric: tabular-nums; }
.match-result > span { color: #8b96aa; margin: 0 6px; }
.match-result small { display: block; color: #647087; font-size: 10px; font-weight: 400; }
.versus { color: #778298; font-size: 11px; font-weight: 400; }
.photo-record > img { display: block; width: 100%; aspect-ratio: 16 / 9; object-fit: cover; border-radius: 7px; background: #f1f3f8; margin-bottom: 14px; }
.sponsor-record > img { display: block; width: 100%; height: 130px; object-fit: contain; border-radius: 7px; background: #f7f8fb; padding: 14px; margin-bottom: 14px; }
.form-grid .check { position: relative; display: grid; grid-column: 1 / -1; grid-template-columns: 20px minmax(0, 1fr); gap: 11px; align-items: center; align-content: center; width: 100%; min-height: 48px; padding: 12px; border: 1px solid #d7ddea; border-radius: 8px; color: #34415a; background: #fafbfe; cursor: pointer; font-weight: 500; }
.check:has(input:checked) { border-color: #b7adee; background: #f7f5ff; }
.check .checkbox-text { min-width: 0; overflow-wrap: anywhere; }
.toggle-group { grid-column: 1 / -1; min-width: 0; display: grid; gap: 12px; }
.club-admin .check input[type=checkbox] { position: absolute; left: 12px; top: 50%; transform: translateY(-50%); display: block; width: 20px; height: 20px; min-width: 20px; min-height: 20px; padding: 0; margin: 0; opacity: 0; cursor: pointer; }
.check .checkbox-box { display: grid; place-items: center; width: 20px; height: 20px; border: 1.5px solid #99a5bb; border-radius: 5px; background: #fff; color: #fff; pointer-events: none; }
.checkbox-box svg { display: block; width: 15px; height: 15px; visibility: hidden; stroke: currentColor; stroke-width: 2.5; stroke-linecap: round; stroke-linejoin: round; }
.check input:checked + .checkbox-box { border-color: #5f50cb; background: #5f50cb; }
.check input:checked + .checkbox-box svg { visibility: visible; }
.check input:focus-visible + .checkbox-box { outline: 3px solid #818cf8; outline-offset: 4px; }
.check:has(input:disabled) { opacity: .6; cursor: not-allowed; }
.delete-dialog { width: min(490px, calc(100vw - 32px)); max-height: calc(100dvh - 32px); overflow-y: auto; padding: 30px; border: 1px solid #d6dce8; border-radius: 16px; background: #fff; color: #182238; box-shadow: 0 24px 80px #0f172a33; }
.delete-dialog::backdrop { background: #1118278c; backdrop-filter: blur(3px); }
.delete-dialog h2 { margin: 12px 0; font-size: 23px; line-height: 1.3; }
.delete-dialog p { color: #606d82; font-size: 13px; line-height: 1.8; }
.delete-dialog .eyebrow { color: #b42338; font-size: 10px; }
.delete-dialog .delete-name { color: #26334b; font-weight: 600; overflow-wrap: anywhere; }
.dialog-actions { display: flex; flex-wrap: wrap; justify-content: flex-end; gap: 10px; margin-top: 24px; }
.club-admin .icon-button { display: grid; place-items: center; width: 38px; height: 38px; min-height: 38px; padding: 8px; }
.icon-button svg { width: 19px; height: 19px; }
.club-admin .mobile-menu, .club-admin .mobile-close, .club-admin .mobile-backdrop { display: none; }
/* Media controls retain upload behavior while inheriting the workspace palette. */
.club-admin :deep(.media-field) { color: #34415a; font-size: 12px; }
.club-admin :deep(.media-dropzone) { border-color: #cbd4e5; background: #fafbfe; }
.club-admin :deep(.media-thumbnail) { background: #edf0f8; }
.club-admin :deep(.media-controls > span), .club-admin :deep(.media-link summary) { color: #647087; }
.club-admin :deep(.media-field button) { background: #fff; border-color: #cbd2e0; color: #34415a; }
.club-admin :deep(.media-field .remove-media) { color: #b42338; border: 0; background: none; }
.club-admin :deep(.media-link input) { background: #fff; border-color: #cbd2e0; color: #182238; }
.club-admin :deep(.media-error), .club-admin :deep(.media-thumbnail > span) { color: #b42338; }
.club-admin :deep(.media-ready) { color: #21734b; }
.club-admin :deep(.tile-icon-editor) { background: #fafbfe; border-color: #dce2ed; }
.club-admin :deep(.icon-controls strong) { color: #34415a; font-size: 13px; }
.club-admin :deep(.upload-label) { background: #eeebff; color: #5943b4; }
.club-admin :deep(.icon-controls button) { color: #647087; }
.club-admin :deep(.icon-controls p) { color: #b42338; }
.club-admin :deep(.media-field button:focus-visible), .club-admin :deep(.media-link summary:focus-visible), .club-admin :deep(.media-link input:focus-visible), .club-admin :deep(.upload-label:focus-within) { outline: 3px solid #818cf8; outline-offset: 3px; }
@media (min-width: 901px) {
  .club-admin--collapsed > aside { width: 80px; padding-inline: 12px; }
  .club-admin--collapsed .workspace { margin-left: 80px; }
  .club-admin--collapsed .admin-brand > div, .club-admin--collapsed .club-identity > div, .club-admin--collapsed .identity-dot, .club-admin--collapsed .nav-active-dot { display: none; }
  .club-admin--collapsed .sidebar-top { justify-content: center; }
  .club-admin--collapsed .club-identity { justify-content: center; padding: 10px 0; border-color: transparent; background: transparent; }
  .club-admin--collapsed .nav-label, .club-admin--collapsed .footer-label { position: absolute; width: 1px; height: 1px; overflow: hidden; clip-path: inset(50%); white-space: nowrap; }
  .club-admin--collapsed .nav-group-label { height: 10px; margin: 0 0 10px; font-size: 0; border-bottom: 1px solid #293349; }
  .club-admin--collapsed nav button, .club-admin--collapsed .aside-bottom > a, .club-admin--collapsed .aside-bottom > button { justify-content: center; padding-inline: 10px; }
  .club-admin--collapsed .draft-indicator { justify-content: center; padding: 0; }
}
@media (min-width: 1600px) { .editor-layout { grid-template-columns: minmax(0, 1.3fr) minmax(340px, .85fr); } }
@media (max-width: 1320px) { .stats { grid-template-columns: repeat(3, minmax(0, 1fr)); } .quick { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@media (max-width: 1200px) { .editor-layout { grid-template-columns: minmax(0, 1fr); } .records-list { grid-template-columns: repeat(2, minmax(0, 1fr)); } .tile-icons-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } .dashboard-detail-grid { grid-template-columns: minmax(0, 1fr); } }
@media (max-width: 900px) {
  .club-admin > aside { width: min(300px, calc(100vw - 52px)); padding-top: 20px; visibility: hidden; transform: translateX(-101%); transition: transform .2s ease, visibility .2s; }
  .club-admin--menu-open > aside { visibility: visible; transform: translateX(0); }
  .club-admin .mobile-close { display: grid; border-color: #35415b; background: #1c263a; color: #dfe5f2; font-size: 23px; }
  .club-admin .mobile-menu { display: grid; }
  .club-admin .collapse-button { display: none; }
  .club-admin .mobile-backdrop { display: block; position: fixed; z-index: 55; inset: 0; width: 100%; height: 100dvh; min-height: 100vh; border: 0; border-radius: 0; padding: 0; background: #11182788; backdrop-filter: blur(2px); }
  .club-admin .mobile-backdrop:hover { background: #11182788; }
  :global(body:has(.club-admin--menu-open)) { overflow: hidden; }
  .workspace { margin-left: 0; padding: 0 22px 36px; }
  .workspace-topbar { margin-inline: -22px; padding-inline: 22px; margin-bottom: 26px; }
  .club-identity { margin-bottom: 22px; }
}
@media (max-width: 600px) {
  .club-admin button { min-height: 44px; }
  .workspace { padding: 0 16px 30px; }
  .workspace-topbar { min-height: 64px; padding-inline: 16px; margin-inline: -16px; gap: 8px; }
  .workspace-topbar > div { gap: 9px; font-size: 10px; }
  .workspace-context { display: none; }
  .workspace > header { align-items: flex-start; flex-direction: column; gap: 15px; margin-bottom: 22px; }
  .workspace h1 { font-size: 28px; }
  .public-link { min-height: 44px; }
  .card { padding: 19px; border-radius: 11px; }
  .welcome { padding: 25px; align-items: flex-start; }
  .welcome-side { display: none; }
  .welcome h2 { font-size: 26px; }
  .welcome p { font-size: 12px; }
  .stats { grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 11px; }
  .stats article { padding: 16px; }
  .stats strong { font-size: 28px; margin-top: 12px; }
  .stats small { font-size: 10px; }
  .stats span { font-size: 12px; }
  .quick { grid-template-columns: minmax(0, 1fr); gap: 11px; }
  .club-admin .quick > button { display: grid; grid-template-columns: 38px minmax(0, 1fr); align-items: center; gap: 4px 13px; padding: 16px; }
  .quick-icon { grid-row: span 2; margin: 0; }
  .quick strong { font-size: 12px; padding-right: 14px; }
  .quick small { font-size: 11px; grid-column: 2; }
  .quick-arrow { top: 15px; right: 14px; }
  .panel-kicker { display: none; }
  .dashboard-section-heading { margin-top: 26px; }
  .dashboard-detail-grid { gap: 18px; }
  .dashboard-list .badge { font-size: 9px; padding: 3px 5px; }
  .dashboard-list > article { gap: 9px; }
  .dashboard-list > article strong { font-size: 11px; }
  .form-grid, .player-grid, .records-list, .tile-icons-grid { grid-template-columns: minmax(0, 1fr); }
  .collection-toolbar { padding: 16px; gap: 13px; }
  .collection-toolbar label, .collection-toolbar .search-field { flex-basis: 100%; }
  .filter-summary { justify-content: space-between; }
  .editor-heading { gap: 12px; }
  .editor-footer > span { max-width: 100%; flex-basis: 100%; }
  .editor-footer button.primary { flex: 1; }
  .player-editor, .contact-editor { padding: 15px; }
  .typography-editor { padding: 15px; }
  .typography-preview { padding: 18px; }
  .set-editor { padding: 12px; }
  .set-row { grid-template-columns: 23px minmax(0, 1fr) 5px minmax(0, 1fr) 26px; gap: 5px; }
  .set-row label { font-size: 10px; }
  .set-row input { padding: 10px 6px; }
  .dialog-actions { flex-direction: column-reverse; }
  .dialog-actions button { width: 100%; }
  .delete-dialog { padding: 24px; }
  .login-card { padding: 28px; }
}
@media (forced-colors: active) { .check .checkbox-box { forced-color-adjust: none; border-color: CanvasText; background: Canvas; color: CanvasText; } .check input:checked + .checkbox-box { background: Highlight; border-color: Highlight; color: HighlightText; } .check input:focus-visible + .checkbox-box { outline-color: Highlight; } }
@media (prefers-reduced-motion: reduce) { *, *::before, *::after { animation: none !important; transition: none !important; scroll-behavior: auto !important; } }
</style>
