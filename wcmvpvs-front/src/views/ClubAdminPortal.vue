<script setup>
import { computed, onMounted, reactive, ref } from 'vue'

const props = defineProps({ slug: { type: String, required: true } })
const authenticated = ref(false)
const checking = ref(true)
const busy = ref(false)
const error = ref('')
const notice = ref('')
const activeTab = ref('dashboard')
const overview = ref({ club: {}, teams: [], posts: [], gallery: [], matches: [], counts: {} })
const playersByTeam = ref({})
const login = reactive({ username: `club-${props.slug}`, password: '' })

const settings = reactive({})
const teamForm = reactive(emptyTeam())
const postForm = reactive({ kind: 'announcement', title: '', excerpt: '', body: '', imageUrl: '', teamId: 0, pinned: false, active: true, publishedAt: '' })
const galleryForm = reactive({ title: '', imageUrl: '', caption: '', eventDate: '', teamId: 0, active: true })
const matchForm = reactive({ teamId: 0, opponent: '', competition: '', roundLabel: '', scheduledAt: '', location: '', home: true, status: 'scheduled', scoreFor: 0, scoreAgainst: 0, setsJson: '[]' })
const rosterTeamId = ref(0)
const rosterText = ref('')

const tabs = [
  ['dashboard','Panoramica'], ['settings','Homepage'], ['teams','Squadre e roster'],
  ['posts','Annunci e news'], ['gallery','Photo gallery'], ['matches','Calendario'],
]
const teams = computed(() => overview.value.teams ?? [])

function emptyTeam() { return { id: 0, name: '', slug: '', category: '', gender: 'mixed', championship: '', description: '', logoUrl: '', heroImageUrl: '', position: 0, active: true } }
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

async function loadOverview() {
  checking.value = true
  try {
    overview.value = await request('/overview')
    authenticated.value = true
    resetObject(settings, overview.value.club)
    await loadTeams()
  } catch (e) { if (e.message !== 'unauthorized') error.value = 'Impossibile caricare il CMS.' }
  finally { checking.value = false }
}
async function loadTeams() { const data = await request('/teams'); overview.value.teams = data.teams ?? []; playersByTeam.value = data.players ?? {} }
async function loginSubmit() { busy.value = true; error.value = ''; try { await request('/login', { method:'POST', body: JSON.stringify(login) }); authenticated.value = true; await loadOverview() } catch { error.value = 'Credenziali non valide.' } finally { busy.value = false } }
async function logout() { await request('/logout', { method:'POST' }).catch(()=>{}); authenticated.value = false; login.password = '' }

async function saveSettings() { busy.value=true;try{await request('/settings',{method:'PUT',body:JSON.stringify(settings)});message('Homepage aggiornata.');await loadOverview()}catch{error.value='Salvataggio fallito.'}finally{busy.value=false} }
function editTeam(team) { resetObject(teamForm, { ...emptyTeam(), ...team }); activeTab.value='teams' }
async function saveTeam() { if(!teamForm.name.trim())return;busy.value=true;try{const path=teamForm.id?`/teams/${teamForm.id}`:'/teams';await request(path,{method:teamForm.id?'PUT':'POST',body:JSON.stringify(teamForm)});resetObject(teamForm,emptyTeam());await loadTeams();message('Squadra salvata.')}catch{error.value='Impossibile salvare la squadra.'}finally{busy.value=false} }
async function deleteTeam(team) { if(!confirm(`Eliminare ${team.name} con roster, partite e contenuti collegati?`))return;await request(`/teams/${team.id}`,{method:'DELETE'});await loadOverview() }
function openRoster(team) { rosterTeamId.value=team.id; rosterText.value=(playersByTeam.value[String(team.id)]??[]).map(p=>`${p.jerseyNumber||''};${p.firstName} ${p.lastName};${p.role};${p.imageUrl||''}`).join('\n') }
async function saveRoster() { const players=rosterText.value.split('\n').map((line,index)=>{const [jersey,name='',role='',imageUrl='']=line.split(';');const parts=name.trim().split(/\s+/);return{jerseyNumber:Number(jersey)||0,firstName:parts.shift()||'',lastName:parts.join(' '),role:role.trim(),imageUrl:imageUrl.trim(),position:index}}).filter(p=>p.firstName||p.lastName);await request(`/teams/${rosterTeamId.value}/players`,{method:'PUT',body:JSON.stringify({players})});await loadTeams();rosterTeamId.value=0;message('Roster aggiornato.') }

async function savePost(){if(!postForm.title.trim())return;await request('/posts',{method:'POST',body:JSON.stringify({...postForm,teamId:Number(postForm.teamId)||0})});Object.assign(postForm,{kind:'announcement',title:'',excerpt:'',body:'',imageUrl:'',teamId:0,pinned:false,active:true,publishedAt:''});await loadOverview();message('Contenuto pubblicato.')}
async function deletePost(item){await request(`/posts/${item.id}`,{method:'DELETE'});await loadOverview()}
async function saveGallery(){if(!galleryForm.imageUrl.trim())return;await request('/gallery',{method:'POST',body:JSON.stringify({...galleryForm,teamId:Number(galleryForm.teamId)||0})});Object.assign(galleryForm,{title:'',imageUrl:'',caption:'',eventDate:'',teamId:0,active:true});await loadOverview();message('Foto aggiunta.')}
async function deleteGallery(item){await request(`/gallery/${item.id}`,{method:'DELETE'});await loadOverview()}
async function saveMatch(){if(!matchForm.teamId||!matchForm.opponent.trim()||!matchForm.scheduledAt)return;await request('/matches',{method:'POST',body:JSON.stringify({...matchForm,teamId:Number(matchForm.teamId),scoreFor:Number(matchForm.scoreFor)||0,scoreAgainst:Number(matchForm.scoreAgainst)||0})});Object.assign(matchForm,{teamId:0,opponent:'',competition:'',roundLabel:'',scheduledAt:'',location:'',home:true,status:'scheduled',scoreFor:0,scoreAgainst:0,setsJson:'[]'});await loadOverview();message('Partita aggiunta.')}
async function deleteMatch(item){await request(`/matches/${item.id}`,{method:'DELETE'});await loadOverview()}
function teamName(id){return teams.value.find(team=>team.id===id)?.name||'Tutta la società'}
function formatDate(value){if(!value)return '—';const d=new Date(value);return Number.isNaN(d.getTime())?value:new Intl.DateTimeFormat('it-IT',{dateStyle:'medium',timeStyle:value.includes('T')?'short':undefined}).format(d)}

onMounted(loadOverview)
</script>

<template>
  <main class="club-admin">
    <section v-if="checking" class="center">Caricamento CMS Club…</section>
    <section v-else-if="!authenticated" class="login-shell">
      <div class="login-card"><p class="eyebrow">ARENABOOSTX CLUB</p><h1>Gestisci il tuo Club Hub</h1><p>Homepage, squadre, roster, annunci, fotografie e calendario.</p><form @submit.prevent="loginSubmit"><label>Username<input v-model="login.username" autocomplete="username" /></label><label>Password<input v-model="login.password" type="password" autocomplete="current-password" /></label><button :disabled="busy">{{busy?'Accesso…':'Entra nel CMS'}}</button></form><p v-if="error" class="error">{{error}}</p></div>
    </section>
    <template v-else>
      <aside>
        <a class="admin-brand" :href="`/club/${slug}`" target="_blank"><span>AX</span><div><strong>Club CMS</strong><small>{{overview.club.name}}</small></div></a>
        <nav><button v-for="tab in tabs" :key="tab[0]" :class="{active:activeTab===tab[0]}" @click="activeTab=tab[0]">{{tab[1]}}</button></nav>
        <div class="aside-bottom"><a :href="`/club/${slug}`" target="_blank">Apri sito pubblico ↗</a><button @click="logout">Esci</button></div>
      </aside>
      <section class="workspace">
        <header><div><p class="eyebrow">{{overview.club.seasonLabel}}</p><h1>{{tabs.find(tab=>tab[0]===activeTab)?.[1]}}</h1></div><span class="status">ONLINE</span></header>
        <p v-if="notice" class="notice">{{notice}}</p><p v-if="error" class="error">{{error}}</p>

        <div v-if="activeTab==='dashboard'">
          <div class="stats"><article v-for="entry in [['Squadre',overview.counts.teams],['Atleti',overview.counts.players],['Contenuti',overview.counts.posts],['Foto',overview.counts.photos],['Partite',overview.counts.matches]]" :key="entry[0]"><span>{{entry[0]}}</span><strong>{{entry[1]||0}}</strong></article></div>
          <section class="card welcome"><div><p class="eyebrow">IL TUO HUB ANNUALE</p><h2>{{overview.club.name}}</h2><p>{{overview.club.tagline||'Completa la homepage e inizia a pubblicare contenuti.'}}</p></div><a :href="`/club/${slug}`" target="_blank">Visualizza sito</a></section>
          <section class="card"><h2>Avvio rapido</h2><div class="quick"><button @click="activeTab='settings'">1. Personalizza homepage</button><button @click="activeTab='teams'">2. Completa squadre e roster</button><button @click="activeTab='matches'">3. Inserisci calendario</button><button @click="activeTab='posts'">4. Pubblica un annuncio</button></div></section>
        </div>

        <section v-else-if="activeTab==='settings'" class="card"><h2>Identità e homepage</h2><div class="form-grid"><label>Nome<input v-model="settings.name" /></label><label>Città<input v-model="settings.city" /></label><label class="wide">Claim<input v-model="settings.tagline" /></label><label>Stagione<input v-model="settings.seasonLabel" /></label><label>Logo URL<input v-model="settings.logoUrl" /></label><label class="wide">Immagine hero URL<input v-model="settings.heroImageUrl" /></label><label>Colore primario<input v-model="settings.primaryColor" type="color" /></label><label>Colore accento<input v-model="settings.secondaryColor" type="color" /></label><label class="check"><input v-model="settings.active" type="checkbox" /> Sito pubblico attivo</label></div><button class="primary" :disabled="busy" @click="saveSettings">Salva homepage</button></section>

        <div v-else-if="activeTab==='teams'" class="split">
          <section class="card"><h2>{{teamForm.id?'Modifica squadra':'Nuova squadra'}}</h2><div class="form-grid single"><label>Nome<input v-model="teamForm.name" placeholder="Prima Divisione Femminile" /></label><label>Categoria<input v-model="teamForm.category" placeholder="Prima Divisione" /></label><label>Genere<select v-model="teamForm.gender"><option value="female">Femminile</option><option value="male">Maschile</option><option value="mixed">Misto</option></select></label><label>Campionato<input v-model="teamForm.championship" /></label><label>Logo URL<input v-model="teamForm.logoUrl" /></label><label>Hero URL<input v-model="teamForm.heroImageUrl" /></label><label>Descrizione<textarea v-model="teamForm.description" rows="4" /></label><label class="check"><input v-model="teamForm.active" type="checkbox" /> Visibile</label></div><div class="buttons"><button class="primary" @click="saveTeam">Salva</button><button v-if="teamForm.id" @click="resetObject(teamForm,emptyTeam())">Annulla</button></div></section>
          <section class="card"><h2>Squadre</h2><div class="list"><article v-for="team in teams" :key="team.id"><div><strong>{{team.name}}</strong><small>{{team.championship||team.category}} · {{(playersByTeam[String(team.id)]||[]).length}} atleti</small></div><div><button @click="openRoster(team)">Roster</button><button @click="editTeam(team)">Modifica</button><button class="danger" @click="deleteTeam(team)">Elimina</button></div></article></div></section>
          <section v-if="rosterTeamId" class="card roster-editor"><h2>Roster · {{teamName(rosterTeamId)}}</h2><p>Una persona per riga: <code>numero; Nome Cognome; ruolo; URL foto</code></p><textarea v-model="rosterText" rows="13" placeholder="7; Anna Rossi; Schiacciatrice; https://…" /><div class="buttons"><button class="primary" @click="saveRoster">Salva roster</button><button @click="rosterTeamId=0">Chiudi</button></div></section>
        </div>

        <div v-else-if="activeTab==='posts'" class="split">
          <section class="card"><h2>Nuovo contenuto</h2><div class="form-grid single"><label>Tipo<select v-model="postForm.kind"><option value="announcement">Annuncio</option><option value="news">Notizia</option></select></label><label>Squadra<select v-model.number="postForm.teamId"><option :value="0">Tutta la società</option><option v-for="team in teams" :key="team.id" :value="team.id">{{team.name}}</option></select></label><label>Titolo<input v-model="postForm.title" /></label><label>Sommario<textarea v-model="postForm.excerpt" rows="3" /></label><label>Testo<textarea v-model="postForm.body" rows="5" /></label><label>Immagine URL<input v-model="postForm.imageUrl" /></label><label>Data<input v-model="postForm.publishedAt" type="date" /></label><label class="check"><input v-model="postForm.pinned" type="checkbox" /> In evidenza</label></div><button class="primary" @click="savePost">Pubblica</button></section>
          <section class="card"><h2>Pubblicati</h2><div class="list"><article v-for="post in overview.posts" :key="post.id"><div><strong>{{post.title}}</strong><small>{{post.kind}} · {{post.teamName||'Società'}} · {{formatDate(post.publishedAt)}}</small></div><button class="danger" @click="deletePost(post)">Elimina</button></article></div></section>
        </div>

        <div v-else-if="activeTab==='gallery'" class="split">
          <section class="card"><h2>Aggiungi fotografia</h2><div class="form-grid single"><label>Immagine URL<input v-model="galleryForm.imageUrl" /></label><label>Titolo<input v-model="galleryForm.title" /></label><label>Didascalia<textarea v-model="galleryForm.caption" /></label><label>Squadra<select v-model.number="galleryForm.teamId"><option :value="0">Tutta la società</option><option v-for="team in teams" :key="team.id" :value="team.id">{{team.name}}</option></select></label><label>Data evento<input v-model="galleryForm.eventDate" type="date" /></label></div><button class="primary" @click="saveGallery">Aggiungi</button></section>
          <section class="card"><h2>Gallery</h2><div class="photo-admin"><figure v-for="photo in overview.gallery" :key="photo.id"><img :src="photo.imageUrl" :alt="photo.title" /><figcaption>{{photo.title||photo.teamName}}<button @click="deleteGallery(photo)">×</button></figcaption></figure></div></section>
        </div>

        <div v-else-if="activeTab==='matches'" class="split">
          <section class="card"><h2>Nuova partita</h2><div class="form-grid single"><label>Squadra<select v-model.number="matchForm.teamId"><option :value="0">Seleziona</option><option v-for="team in teams" :key="team.id" :value="team.id">{{team.name}}</option></select></label><label>Avversario<input v-model="matchForm.opponent" /></label><label>Competizione<input v-model="matchForm.competition" /></label><label>Giornata<input v-model="matchForm.roundLabel" /></label><label>Data e ora<input v-model="matchForm.scheduledAt" type="datetime-local" /></label><label>Luogo<input v-model="matchForm.location" /></label><label class="check"><input v-model="matchForm.home" type="checkbox" /> Partita in casa</label><label>Stato<select v-model="matchForm.status"><option value="scheduled">Programmata</option><option value="live">Live</option><option value="completed">Conclusa</option></select></label><template v-if="matchForm.status==='completed'"><label>Set squadra<input v-model.number="matchForm.scoreFor" type="number" min="0" /></label><label>Set avversario<input v-model.number="matchForm.scoreAgainst" type="number" min="0" /></label><label>Parziali JSON<input v-model="matchForm.setsJson" placeholder='["25-18","25-21","25-20"]' /></label></template></div><button class="primary" @click="saveMatch">Aggiungi partita</button></section>
          <section class="card"><h2>Calendario</h2><div class="list"><article v-for="match in overview.matches" :key="match.id"><div><strong>{{teamName(match.teamId)}} – {{match.opponent}}</strong><small>{{formatDate(match.scheduledAt)}} · {{match.status}}<template v-if="match.status==='completed'"> · {{match.scoreFor}}–{{match.scoreAgainst}}</template></small></div><button class="danger" @click="deleteMatch(match)">Elimina</button></article></div></section>
        </div>
      </section>
    </template>
  </main>
</template>

<style scoped>
*{box-sizing:border-box}.club-admin{min-height:100vh;background:#07101f;color:#e5e7eb;font-family:Inter,ui-sans-serif,system-ui,sans-serif}.center,.login-shell{min-height:100vh;display:grid;place-items:center}.login-shell{background:radial-gradient(circle at 20% 20%,#172554,#07101f 55%)}.login-card{width:min(460px,calc(100% - 32px));padding:38px;background:rgba(15,23,42,.88);border:1px solid #263449;border-radius:24px;box-shadow:0 30px 80px rgba(0,0,0,.35)}.login-card h1{font-size:38px;letter-spacing:-.05em;margin:8px 0}.login-card>p{color:#94a3b8}.login-card form{display:grid;gap:14px;margin-top:24px}.login-card label,.form-grid label{display:flex;flex-direction:column;gap:6px;font-size:12px;color:#94a3b8}.login-card input,.form-grid input,.form-grid textarea,.form-grid select,.roster-editor textarea{width:100%;background:#060d19;border:1px solid #293548;border-radius:9px;padding:10px 11px;color:#f8fafc;font:inherit}.login-card button,.primary{border:0;border-radius:9px;padding:11px 16px;background:#f59e0b;color:#111827;font-weight:900;cursor:pointer}.eyebrow{margin:0;color:#f59e0b;font-size:10px;font-weight:950;letter-spacing:.16em}.error{color:#f87171}.notice{position:fixed;right:24px;top:22px;background:#dcfce7;color:#166534;padding:11px 16px;border-radius:10px;z-index:10}aside{position:fixed;inset:0 auto 0 0;width:245px;background:#030712;border-right:1px solid #172033;padding:24px 18px;display:flex;flex-direction:column}.admin-brand{display:flex;align-items:center;gap:11px;color:#fff;text-decoration:none;margin-bottom:34px}.admin-brand>span{width:43px;height:43px;display:grid;place-items:center;border-radius:11px;background:#f59e0b;color:#111827;font-weight:950}.admin-brand div{display:flex;flex-direction:column}.admin-brand small{color:#64748b}.club-admin nav{display:grid;gap:6px}.club-admin nav button{border:0;border-radius:9px;padding:11px 13px;text-align:left;background:transparent;color:#94a3b8;font-weight:700;cursor:pointer}.club-admin nav button.active{background:#172554;color:#fff}.aside-bottom{margin-top:auto;display:grid;gap:10px}.aside-bottom a,.aside-bottom button{color:#94a3b8;background:transparent;border:0;text-align:left;text-decoration:none;cursor:pointer}.workspace{margin-left:245px;min-height:100vh;padding:34px max(28px,calc((100vw - 1220px)/2))}.workspace>header{display:flex;align-items:center;justify-content:space-between;margin-bottom:30px}.workspace h1{font-size:36px;letter-spacing:-.04em;margin:4px 0}.status{font-size:10px;font-weight:900;padding:7px 10px;border-radius:999px;background:#052e16;color:#4ade80}.stats{display:grid;grid-template-columns:repeat(5,1fr);gap:12px;margin-bottom:18px}.stats article,.card{background:#0e1727;border:1px solid #233047;border-radius:16px;padding:20px}.stats span{display:block;color:#64748b;font-size:12px}.stats strong{font-size:34px}.card{margin-bottom:16px}.card h2{margin:0 0 18px}.welcome{display:flex;align-items:center;justify-content:space-between;background:linear-gradient(120deg,#172554,#111827)}.welcome h2{font-size:30px;margin:6px 0}.welcome p{color:#94a3b8}.welcome a{background:#f59e0b;color:#111827;padding:11px 16px;border-radius:9px;text-decoration:none;font-weight:900}.quick{display:grid;grid-template-columns:repeat(4,1fr);gap:10px}.quick button,.buttons button,.list button{background:#111d31;border:1px solid #2b3a52;border-radius:9px;color:#cbd5e1;padding:10px;cursor:pointer}.form-grid{display:grid;grid-template-columns:repeat(2,1fr);gap:13px}.form-grid.single{grid-template-columns:1fr}.form-grid .wide{grid-column:1/-1}.form-grid .check{flex-direction:row;align-items:center}.form-grid .check input{width:auto}.card>.primary{margin-top:18px}.split{display:grid;grid-template-columns:minmax(290px,.75fr) minmax(420px,1.25fr);gap:16px;align-items:start}.roster-editor{grid-column:1/-1}.roster-editor>p{color:#94a3b8}.roster-editor textarea{resize:vertical}.buttons{display:flex;gap:8px;margin-top:14px}.list{display:grid;gap:8px}.list article{display:flex;align-items:center;justify-content:space-between;gap:12px;padding:13px;background:#091221;border:1px solid #1c2940;border-radius:10px}.list small{display:block;color:#64748b;margin-top:3px}.list .danger,.actions .danger{color:#f87171}.photo-admin{display:grid;grid-template-columns:repeat(3,1fr);gap:9px}.photo-admin figure{margin:0;background:#091221;border-radius:10px;overflow:hidden}.photo-admin img{width:100%;aspect-ratio:1;object-fit:cover}.photo-admin figcaption{display:flex;justify-content:space-between;padding:8px;font-size:11px}.photo-admin button{border:0;background:transparent;color:#f87171;cursor:pointer}@media(max-width:900px){aside{position:static;width:100%;height:auto}.club-admin nav{display:flex;overflow:auto}.aside-bottom{display:none}.workspace{margin-left:0;padding:22px 16px}.stats{grid-template-columns:repeat(2,1fr)}.split,.form-grid{grid-template-columns:1fr}.quick{grid-template-columns:1fr 1fr}.welcome{align-items:flex-start;gap:20px;flex-direction:column}.photo-admin{grid-template-columns:repeat(2,1fr)}}
</style>
