<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { clubHomeTiles, clubInitials as initials, clubScoreLabel as scoreLabel, formatClubDate as formatDate } from '../clubHub'
import { organizationThemeStyle } from '../organizationTheme'
import { clubTypography } from '../clubTypography'
import ClubContentModal from '../components/ClubContentModal.vue'
import ClubGalleryLightbox from '../components/ClubGalleryLightbox.vue'
import { clubHomePath } from '../clubDomain'

const props = defineProps({ slug: { type: String, required: true }, teamSlug: { type: String, default: '' }, customDomain: Boolean })
const homePath = computed(() => clubHomePath(props.slug, props.customDomain))
const data = ref(null)
const loading = ref(true)
const error = ref('')
const selectedPhoto = ref(null)

const club = computed(() => data.value?.club ?? {})
const typography = computed(() => clubTypography(club.value))
const isTeam = computed(() => Boolean(props.teamSlug))
const matchClubName = computed(() => club.value.name || 'Società')
const announcements = computed(() => (data.value?.posts ?? []).filter(post => post.kind === 'announcement'))
const upcoming = computed(() => (data.value?.matches ?? []).filter(match => match.status === 'scheduled').sort((a,b) => a.scheduledAt.localeCompare(b.scheduledAt)).slice(0, 6))
const results = computed(() => (data.value?.matches ?? []).filter(match => match.status === 'completed').slice(0, 8))
const activePanel = ref('')
const activeTeamData = ref(null)
const activeTeamTitle = ref('')
const activeTeamSection = ref('overview')
const teamModalLoading = ref(false)
const teamModalError = ref('')
const teamModalCache = new Map()
let teamRequestSequence = 0
const featuredMatch = computed(() => {
  const matches = data.value?.matches ?? []
  return matches.find(match => match.status === 'live') || upcoming.value[0] || results.value[0] || null
})
const homeTiles = computed(() => clubHomeTiles(data.value ?? {}))

const theme = computed(() => {
  const primary = club.value.primaryColor || '#172554'
  const secondary = club.value.secondaryColor || '#f59e0b'
  const colors = organizationThemeStyle({ primary_color: primary, secondary_color: secondary })
  return {
    ...typography.value.style,
    '--club-primary': primary,
    '--club-accent': secondary,
    '--club-on-primary': colors['--on-primary'],
    '--club-on-accent': colors['--on-secondary'],
  }
})

async function load() {
  loading.value = true
  error.value = ''
  try {
    const endpoint = props.teamSlug
      ? `/api/v1/clubs/${encodeURIComponent(props.slug)}/teams/${encodeURIComponent(props.teamSlug)}`
      : `/api/v1/clubs/${encodeURIComponent(props.slug)}/home`
    const response = await fetch(endpoint)
    if (!response.ok) throw new Error(response.status)
    data.value = await response.json()
  } catch { error.value = 'Questa pagina non è ancora disponibile.' }
  finally { loading.value = false }
}

function openPanel(panel) {
  teamRequestSequence += 1
  activeTeamData.value = null
  activeTeamTitle.value = ''
  teamModalLoading.value = false
  teamModalError.value = ''
  activePanel.value = panel
}
async function openTeam(team) {
  const requestId = ++teamRequestSequence
  activePanel.value = 'team'
  activeTeamTitle.value = team.name
  activeTeamSection.value = 'overview'
  activeTeamData.value = null
  teamModalError.value = ''
  teamModalLoading.value = true
  if (teamModalCache.has(team.slug)) {
    activeTeamData.value = teamModalCache.get(team.slug)
    teamModalLoading.value = false
    return
  }
  try {
    const response = await fetch(`/api/v1/clubs/${encodeURIComponent(props.slug)}/teams/${encodeURIComponent(team.slug)}`)
    if (!response.ok) throw new Error(response.status)
    const payload = await response.json()
    if (requestId !== teamRequestSequence) return
    teamModalCache.set(team.slug, payload)
    activeTeamData.value = payload
  } catch {
    if (requestId === teamRequestSequence) teamModalError.value = 'I dettagli della squadra non sono disponibili.'
  } finally {
    if (requestId === teamRequestSequence) teamModalLoading.value = false
  }
}
function closePanel() {
  teamRequestSequence += 1
  activePanel.value = ''
  activeTeamData.value = null
  activeTeamTitle.value = ''
  teamModalLoading.value = false
  teamModalError.value = ''
}
watch(() => [props.slug, props.teamSlug], () => { selectedPhoto.value = null; teamModalCache.clear(); closePanel(); load() })
onMounted(load)
</script>

<template>
  <main class="club-page" :class="{ 'club-home-page': !isTeam }" :style="theme">
    <link v-for="stylesheet in typography.stylesheets" :key="stylesheet" rel="stylesheet" :href="stylesheet" />
    <div v-if="loading" class="state">Caricamento Club Hub…</div>
    <div v-else-if="error" class="state error"><strong>Pagina non disponibile</strong><span>{{ error }}</span></div>
    <template v-else>
      <header v-if="isTeam" class="topbar">
        <a class="brand" :href="homePath">
          <img v-if="club.logoUrl" :src="club.logoUrl" :alt="club.name" />
          <span v-else class="logo-fallback">{{ initials(club.name) }}</span>
          <span><strong>{{ club.name }}</strong><small>{{ club.seasonLabel }}</small></span>
        </a>
        <nav><a :href="homePath">Home</a><a :href="`${homePath}#squadre`">Squadre</a><a :href="`${homePath}#notizie`">Notizie</a><a :href="`${homePath}#gallery`">Gallery</a></nav>
      </header>

      <template v-if="!isTeam">
        <section class="cd-stage">
          <div class="cd-shell">
            <header class="cd-hero" :style="club.heroImageUrl ? { backgroundImage: `linear-gradient(150deg, color-mix(in srgb, var(--club-primary) 95%, transparent), color-mix(in srgb, var(--club-primary) 90%, transparent)), url('${club.heroImageUrl}')` } : {}">
              <div class="cd-hero-top"><span class="cd-status"><i></i> PORTALE UFFICIALE</span><span class="cd-season">{{ club.seasonLabel }}</span></div>
              <div class="cd-identity">
                <div class="cd-logo"><img v-if="club.logoUrl" :src="club.logoUrl" :alt="club.name" /><span v-else>{{ initials(club.name) }}</span></div>
                <div><p>{{ club.city || 'VOLLEY CLUB' }}</p><h1>{{ club.name }}</h1><small>{{ club.tagline || 'Una società, tutte le squadre, una sola passione.' }}</small></div>
              </div>
              <button v-if="announcements[0]" class="cd-breaking" type="button" @click="openPanel('announcements')"><b>{{ announcements[0].pinned ? 'IN EVIDENZA' : 'ANNUNCIO' }}</b><span>{{ announcements[0].title }}</span><i>›</i></button>
            </header>

            <div class="cd-body">
              <button v-if="featuredMatch" class="cd-match" type="button" @click="openPanel('matches')">
                <div class="cd-match-head"><span :class="{ live: featuredMatch.status === 'live' }">{{ featuredMatch.status === 'live' ? '● LIVE ORA' : featuredMatch.status === 'completed' ? 'ULTIMO RISULTATO' : '⚡ PROSSIMA PARTITA' }}</span><time>{{ formatDate(featuredMatch.scheduledAt) }}</time></div>
                <div class="cd-versus"><strong>{{ featuredMatch.home ? matchClubName : featuredMatch.opponent }}</strong><b>{{ featuredMatch.status === 'completed' ? scoreLabel(featuredMatch) : 'VS' }}</b><strong>{{ featuredMatch.home ? featuredMatch.opponent : matchClubName }}</strong></div>
                <small>{{ matchClubName }} · {{ featuredMatch.location || featuredMatch.competition || 'Dettagli partita' }}</small>
              </button>
              <button v-else class="cd-match cd-match-empty" type="button" @click="openPanel('matches')"><span>📅</span><div><strong>Calendario in arrivo</strong><small>Apri la sezione partite</small></div><i>›</i></button>

              <div class="cd-grid">
                <template v-for="tile in homeTiles" :key="tile.id">
                  <button v-if="tile.kind === 'team'" class="cd-tile" type="button" :style="tile.team.heroImageUrl ? { backgroundImage: `linear-gradient(150deg, color-mix(in srgb, var(--tile-bg) 95%, transparent), color-mix(in srgb, var(--tile-bg) 90%, transparent)), url('${tile.team.heroImageUrl}')` } : {}" @click="openTeam(tile.team)">
                    <span class="cd-icon"><img v-if="tile.iconImage" :src="tile.iconImage" alt="" /><i v-else>{{ tile.icon }}</i></span><span class="cd-tile-copy"><strong>{{ tile.label }}</strong><small>{{ tile.sub }}</small><b>APRI <i>›</i></b></span>
                  </button>
                  <button v-else class="cd-tile cd-action" type="button" @click="openPanel(tile.panel)"><span class="cd-icon"><img v-if="tile.iconImage" :src="tile.iconImage" alt="" /><i v-else>{{ tile.icon }}</i></span><span class="cd-tile-copy"><strong>{{ tile.label }}</strong><small>{{ tile.sub }}</small><b>VEDI <i>›</i></b></span></button>
                </template>
              </div>
            </div>
          </div>
        </section>

        <ClubContentModal v-if="activePanel" :panel="activePanel" :club="club" :data="data"
          :team-data="activeTeamData" :team-title="activeTeamTitle" :team-loading="teamModalLoading"
          :team-error="teamModalError" v-model:section="activeTeamSection" @close="closePanel" />
      </template>

      <template v-else>
        <section class="team-hero" :style="data.team.heroImageUrl ? { backgroundImage: `linear-gradient(110deg, rgba(4,10,24,.95), rgba(4,10,24,.3)), url('${data.team.heroImageUrl}')` } : {}">
          <div class="shell"><a class="back" :href="homePath">← Tutte le squadre</a><div class="team-title"><div class="team-badge large"><img v-if="data.team.logoUrl" :src="data.team.logoUrl" alt="" /><span v-else>{{ initials(data.team.name) }}</span></div><div><p class="kicker">{{ data.team.category }}</p><h1>{{ data.team.name }}</h1><p>{{ data.team.championship }}</p></div></div></div>
        </section>
        <div class="team-content shell">
          <section class="section compact"><div class="section-heading"><div><p class="kicker">STAGIONE</p><h2>Calendario e risultati</h2></div></div><div class="fixture-list"><article v-for="match in data.matches" :key="match.id" class="fixture"><div><time>{{ formatDate(match.scheduledAt) }}</time><small>{{ match.competition }} · {{ match.roundLabel }}</small></div><div class="fixture-teams"><strong>{{ match.home ? matchClubName : match.opponent }}</strong><b :class="{ completed: match.status === 'completed' }">{{ match.status === 'completed' ? scoreLabel(match) : 'VS' }}</b><strong>{{ match.home ? match.opponent : matchClubName }}</strong></div><span :class="['status', match.status]">{{ match.status === 'completed' ? 'Conclusa' : match.status === 'live' ? 'Live' : 'Programmata' }}</span></article><p v-if="!data.matches?.length" class="empty">Il calendario sarà pubblicato a breve.</p></div></section>
          <section class="section compact"><div class="section-heading"><div><p class="kicker">LA SQUADRA</p><h2>Roster</h2></div></div><div class="roster-grid"><article v-for="player in data.players" :key="player.id" class="player"><div class="player-photo"><img v-if="player.imageUrl" :src="player.imageUrl" :alt="`${player.firstName} ${player.lastName}`" /><span v-else>{{ player.jerseyNumber || initials(`${player.firstName} ${player.lastName}`) }}</span></div><div><small>{{ player.role }}</small><h3>{{ player.firstName }} {{ player.lastName }}</h3><b v-if="player.jerseyNumber">#{{ player.jerseyNumber }}</b></div></article><p v-if="!data.players?.length" class="empty">Roster in aggiornamento.</p></div></section>
          <section v-if="data.posts?.length" class="section compact"><div class="section-heading"><div><p class="kicker">NOTIZIE</p><h2>Dalla squadra</h2></div></div><div class="news-grid"><article v-for="post in data.posts.slice(0,6)" :key="post.id" class="news-card"><div v-if="post.imageUrl" class="news-image" :style="{ backgroundImage: `url('${post.imageUrl}')` }"></div><div class="news-body"><h3>{{ post.title }}</h3><p>{{ post.excerpt || post.body }}</p></div></article></div></section>
          <section v-if="data.gallery?.length" class="section compact"><div class="section-heading"><div><p class="kicker">GALLERY</p><h2>Momenti della squadra</h2></div></div><div class="gallery-grid team-gallery"><figure v-for="photo in data.gallery" :key="photo.id"><button type="button" class="gallery-open" :aria-label="`Ingrandisci immagine: ${photo.title || photo.caption || 'Foto del club'}`" aria-haspopup="dialog" @click="selectedPhoto = photo"><img :src="photo.imageUrl" :alt="photo.title || photo.caption || 'Foto del club'" loading="lazy" /></button></figure></div></section>
        </div>
      </template>

      <footer v-if="isTeam"><div class="brand"><img v-if="club.logoUrl" :src="club.logoUrl" alt="" /><span><strong>{{ club.name }}</strong><small>{{ club.city }}</small></span></div><p>Club Hub powered by ArenaBoostX</p></footer>
      <ClubGalleryLightbox v-if="selectedPhoto" :photo="selectedPhoto" @close="selectedPhoto = null" />
    </template>
  </main>
</template>

<style scoped>
*{box-sizing:border-box}.club-page{min-height:100vh;background:#f5f3ee;color:#0f172a;font-family:var(--club-font-body, Inter,ui-sans-serif,system-ui,-apple-system,sans-serif)}.shell{width:min(1180px,calc(100% - 36px));margin-inline:auto}.state{min-height:100vh;display:grid;place-content:center;gap:8px;text-align:center;background:#07101f;color:#fff}.state.error span{color:#94a3b8}.topbar{height:80px;padding:0 max(24px,calc((100vw - 1180px)/2));display:flex;align-items:center;justify-content:space-between;background:#07101f;color:#fff;position:relative;z-index:5}.brand{display:flex;align-items:center;gap:11px;color:inherit;text-decoration:none}.brand img,.logo-fallback{width:46px;height:46px;object-fit:contain;border-radius:12px;background:#fff;padding:4px}.logo-fallback{display:grid;place-items:center;color:var(--club-primary);font-weight:900}.brand span{display:flex;flex-direction:column}.brand small{color:#94a3b8}.topbar nav{display:flex;gap:26px}.topbar nav a{color:#cbd5e1;text-decoration:none;font-size:13px;font-weight:700}.hero{min-height:610px;background:linear-gradient(135deg,#07101f,var(--club-primary));background-size:cover;background-position:center;display:flex;align-items:center;justify-content:space-between;padding:80px max(24px,calc((100vw - 1180px)/2));color:#fff}.hero-copy{max-width:700px}.kicker{margin:0 0 10px;color:var(--club-accent);font-size:11px;font-weight:950;letter-spacing:.18em}.hero h1,.team-hero h1{font-size:clamp(48px,8vw,96px);line-height:.94;letter-spacing:-.06em;margin:0;max-width:850px}.hero-copy>p:not(.kicker){font-size:clamp(18px,2.2vw,28px);max-width:650px;color:#dbe4ef}.hero-meta{display:flex;gap:10px;margin:25px 0}.hero-meta span{border:1px solid rgba(255,255,255,.25);border-radius:999px;padding:8px 12px;font-size:12px}.cta{display:inline-block;padding:14px 22px;border-radius:999px;background:var(--club-accent);color:#111827;font-weight:900;text-decoration:none}.hero-mark{width:260px;height:260px;border-radius:50%;display:grid;place-items:center;background:rgba(255,255,255,.1);backdrop-filter:blur(10px);border:1px solid rgba(255,255,255,.2);font-size:58px;font-weight:950}.hero-mark img{width:75%;height:75%;object-fit:contain}.announcements{margin-top:-36px;position:relative;z-index:2;display:grid;gap:8px}.announcement{display:grid;grid-template-columns:100px 1fr auto;gap:18px;align-items:center;background:#fff;border-radius:14px;padding:18px 22px;box-shadow:0 16px 45px rgba(15,23,42,.12)}.announcement>span{font-size:10px;font-weight:900;color:var(--club-primary)}.announcement.pinned>span{color:#b45309}.announcement h3,.announcement p{margin:0}.announcement p{color:#64748b;font-size:13px;margin-top:4px}.announcement time{color:#94a3b8;font-size:12px}.section{padding:86px 0}.section.compact{padding:62px 0;border-bottom:1px solid #d9d6cf}.section-heading{display:flex;align-items:end;justify-content:space-between;gap:30px;margin-bottom:28px}.section-heading h2{font-size:clamp(32px,4vw,54px);letter-spacing:-.045em;margin:0}.section-heading>p{max-width:420px;color:#64748b}.match-strip{display:grid;grid-template-columns:repeat(3,1fr);gap:14px}.match-card{background:#fff;border-radius:16px;padding:20px;border:1px solid #e4e1da}.match-card .team-label{color:var(--club-primary);font-size:11px;font-weight:900;text-transform:uppercase}.match-card time{display:block;color:#64748b;font-size:12px;margin:7px 0 18px}.versus{display:grid;grid-template-columns:1fr auto 1fr;align-items:center;gap:10px}.versus strong:last-child{text-align:right}.versus b{color:var(--club-accent)}.match-card small{display:block;margin-top:18px;color:#94a3b8}.team-grid{display:grid;grid-template-columns:repeat(2,1fr);gap:16px}.team-card{min-height:390px;border-radius:22px;padding:28px;display:flex;flex-direction:column;justify-content:space-between;background:linear-gradient(145deg,var(--club-primary),#07101f);background-size:cover;background-position:center;color:#fff;text-decoration:none;overflow:hidden}.team-badge{width:64px;height:64px;border-radius:17px;background:#fff;display:grid;place-items:center;color:var(--club-primary);font-weight:950}.team-badge img{width:82%;height:82%;object-fit:contain}.team-copy span,.news-body>span{color:var(--club-accent);font-size:10px;font-weight:900;letter-spacing:.12em}.team-copy h3{font-size:36px;line-height:1;margin:8px 0}.team-copy p{color:#cbd5e1}.team-copy b{font-size:13px}.news-grid{display:grid;grid-template-columns:repeat(3,1fr);gap:16px}.news-card{background:#fff;border:1px solid #e4e1da;border-radius:16px;overflow:hidden}.news-image{height:190px;background-size:cover;background-position:center}.news-body{padding:20px}.news-body h3{font-size:20px;margin:7px 0}.news-body p{color:#64748b;display:-webkit-box;-webkit-line-clamp:3;-webkit-box-orient:vertical;overflow:hidden}.news-body time{font-size:11px;color:#94a3b8}.gallery-section{background:#07101f}.section-heading.light h2{color:#fff}.gallery-grid{display:grid;grid-template-columns:repeat(4,1fr);gap:10px}.gallery-grid figure{margin:0;border-radius:13px;overflow:hidden;position:relative;aspect-ratio:1}.gallery-grid img{width:100%;height:100%;object-fit:cover}.gallery-grid figcaption{position:absolute;inset:auto 0 0;padding:34px 14px 12px;background:linear-gradient(transparent,rgba(0,0,0,.8));display:flex;flex-direction:column;color:#fff}.gallery-grid figcaption span{font-size:11px;color:#cbd5e1}.team-hero{min-height:430px;background:linear-gradient(135deg,#07101f,var(--club-primary));background-size:cover;background-position:center;color:#fff;padding:60px 0}.back{color:#cbd5e1;text-decoration:none}.team-title{display:flex;align-items:center;gap:24px;margin-top:80px}.team-badge.large{width:100px;height:100px;border-radius:24px}.team-hero h1{font-size:clamp(42px,7vw,82px)}.team-hero p{color:#cbd5e1}.fixture-list{display:grid;gap:8px}.fixture{display:grid;grid-template-columns:1fr 2fr auto;align-items:center;gap:20px;background:#fff;border:1px solid #e4e1da;border-radius:13px;padding:17px}.fixture time,.fixture small{display:block}.fixture small{color:#94a3b8;margin-top:4px}.fixture-teams{display:grid;grid-template-columns:1fr 70px 1fr;align-items:center;text-align:center}.fixture-teams strong:first-child{text-align:right}.fixture-teams strong:last-child{text-align:left}.fixture-teams b{font-size:18px}.fixture-teams b.completed{font-size:24px;color:var(--club-primary)}.status{font-size:10px;font-weight:900;text-transform:uppercase;border-radius:999px;padding:7px 10px;background:#e2e8f0}.status.live{background:#fee2e2;color:#b91c1c}.status.completed{background:#dcfce7;color:#166534}.roster-grid{display:grid;grid-template-columns:repeat(3,1fr);gap:12px}.player{display:flex;align-items:center;gap:15px;background:#fff;border-radius:14px;padding:14px;border:1px solid #e4e1da}.player-photo{width:70px;height:82px;border-radius:10px;background:linear-gradient(145deg,var(--club-primary),#07101f);overflow:hidden;display:grid;place-items:center;color:#fff;font-size:25px;font-weight:950}.player-photo img{width:100%;height:100%;object-fit:cover}.player h3{margin:3px 0}.player small{color:#64748b;text-transform:uppercase}.player b{color:var(--club-accent)}.team-gallery{grid-template-columns:repeat(5,1fr)}.empty{color:#94a3b8}footer{display:flex;align-items:center;justify-content:space-between;padding:32px max(24px,calc((100vw - 1180px)/2));background:#030712;color:#fff}footer p{color:#64748b;font-size:12px}@media(max-width:800px){.topbar{height:68px}.topbar nav{display:none}.hero{min-height:560px;padding-block:60px}.hero-mark{display:none}.announcement{grid-template-columns:1fr}.announcement time{display:none}.section{padding:58px 0}.section-heading{display:block}.match-strip,.news-grid,.roster-grid{grid-template-columns:1fr}.team-grid{grid-template-columns:1fr}.team-card{min-height:330px}.gallery-grid,.team-gallery{grid-template-columns:repeat(2,1fr)}.fixture{grid-template-columns:1fr}.fixture-teams{grid-template-columns:1fr 50px 1fr}.team-title{margin-top:60px}.team-badge.large{width:76px;height:76px;flex:none}footer{align-items:flex-start;gap:20px;flex-direction:column}}
/* Home Club: dashboard Sunset-like, interamente contenuta nel viewport. */
.cd-stage{height:100vh;height:100dvh;overflow:hidden;background:color-mix(in srgb,var(--club-primary) 18%,#030712);padding:clamp(10px,1.5vw,22px);font-family:inherit}.cd-shell{height:100%;max-width:1440px;margin:auto;display:grid;grid-template-columns:minmax(300px,.72fr) minmax(560px,1.28fr);gap:clamp(10px,1.3vw,18px)}
.cd-hero{
  --tile-bg:var(--club-primary);--tile-ink:var(--club-on-primary);--tile-depth:6px;
  position:relative;isolation:isolate;overflow:hidden;min-width:0;
  border-radius:clamp(20px,2vw,32px);padding:clamp(22px,3vw,42px);
  display:flex;flex-direction:column;color:var(--tile-ink);
  background:var(--tile-bg);background-size:cover;background-position:center;
}

.cd-hero-top{display:flex;align-items:center;justify-content:space-between;gap:12px;position:relative;z-index:2}
.cd-status,.cd-season{border-radius:999px;padding:8px 11px;font-size:10px;font-weight:900;letter-spacing:.08em;backdrop-filter:none}
.cd-status{background:var(--club-accent);color:var(--club-on-accent);border:1px solid color-mix(in srgb,var(--club-accent) 65%,#fff);box-shadow:inset 0 1px 0 rgba(255,255,255,.5),0 2px 0 color-mix(in srgb,var(--club-accent) 70%,#030712)}
.cd-status i{display:inline-block;width:7px;height:7px;border-radius:50%;background:currentColor;margin-right:5px}
.cd-season{background:var(--club-primary);color:var(--club-on-primary);border:1px solid color-mix(in srgb,var(--club-primary) 65%,var(--club-on-primary))}

.cd-identity{margin:auto 0;position:relative;z-index:2}.cd-logo{width:clamp(84px,10vw,150px);height:clamp(84px,10vw,150px);border-radius:28%;background:#fff;border:1px solid #fff;display:grid;place-items:center;padding:10px;color:var(--club-primary);font-size:38px;font-weight:950;box-shadow:inset 0 2px 2px #fff,0 4px 0 rgba(3,7,18,.18),0 8px 14px rgba(0,0,0,.2)}.cd-logo img{width:100%;height:100%;object-fit:contain}.cd-identity p{display:inline-block;padding:4px 7px;border-radius:6px;background:var(--club-accent);margin:24px 0 7px;color:var(--club-on-accent);font-size:11px;font-weight:950;letter-spacing:.18em}.cd-identity h1{margin:0;font-size:clamp(42px,5vw,76px);line-height:1;letter-spacing:-.045em;max-width:620px;overflow-wrap:anywhere}.cd-identity small{display:block;margin-top:15px;max-width:520px;color:inherit;font-size:clamp(13px,1.2vw,18px);line-height:1.35}
.cd-breaking{
  position:relative;z-index:2;width:100%;display:grid;grid-template-columns:auto minmax(0,1fr) auto;
  align-items:center;gap:12px;padding:13px 15px;border:1px solid color-mix(in srgb,var(--club-accent) 65%,#fff);
  border-radius:14px;background:var(--club-accent);color:var(--club-on-accent);text-align:left;cursor:pointer;
  box-shadow:inset 0 2px 1px rgba(255,255,255,.45),inset 0 -2px 3px rgba(3,7,18,.12),0 3px 0 color-mix(in srgb,var(--club-accent) 65%,#030712);
}
.cd-breaking b{color:inherit;font-size:9px;letter-spacing:.1em}
.cd-breaking span{min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;font-weight:750}
.cd-breaking i{font-size:22px;font-style:normal}
.cd-breaking:active{transform:translateY(2px)}
.cd-breaking:focus-visible{outline:3px solid var(--club-on-primary);outline-offset:3px}

.cd-body{min-width:0;min-height:0;display:grid;grid-template-rows:auto 1fr;gap:clamp(9px,1.1vw,15px)}.cd-match{min-height:130px;border:0;border-radius:clamp(18px,1.7vw,26px);padding:clamp(15px,1.8vw,24px);background:#fff;color:#0f172a;text-align:left;cursor:pointer;box-shadow:0 20px 55px rgba(0,0,0,.2)}.cd-match-head{display:flex;justify-content:space-between;gap:15px;font-size:10px;font-weight:900;letter-spacing:.08em}.cd-match-head span{color:var(--club-primary)}.cd-match-head span.live{color:#dc2626}.cd-match-head time{color:#64748b}.cd-versus{display:grid;grid-template-columns:1fr auto 1fr;align-items:center;gap:16px;margin:18px 0 9px;font-size:clamp(15px,1.5vw,22px)}.cd-versus strong:last-child{text-align:right}.cd-versus b{color:var(--club-primary);font-size:clamp(18px,2vw,30px)}.cd-match>small{display:block;text-align:center;color:#64748b}.cd-match-empty{display:flex;align-items:center;gap:15px}.cd-match-empty>span{font-size:34px}.cd-match-empty div{display:flex;flex-direction:column}.cd-match-empty small{margin-top:4px;text-align:left}.cd-match-empty>i{font-style:normal;margin-left:auto;font-size:28px}
.cd-grid{min-height:0;display:grid;grid-template-columns:repeat(4,minmax(0,1fr));grid-template-rows:repeat(2,minmax(0,1fr));gap:clamp(8px,1vw,14px)}
/* Superficie bombata, bordo inferiore e ombra di appoggio dei pulsanti. */
.cd-tile{
  --tile-bg:var(--club-primary);--tile-ink:var(--club-on-primary);--tile-depth:6px;
  position:relative;isolation:isolate;overflow:hidden;min-width:0;
  border-radius:clamp(17px,1.6vw,25px);padding:clamp(14px,1.5vw,22px);
  display:flex;flex-direction:column;align-items:flex-start;
  color:var(--tile-ink);text-decoration:none;text-align:left;cursor:pointer;
  background:var(--tile-bg);background-size:cover;background-position:center;
  transition:transform .16s ease,box-shadow .16s ease;
  -webkit-tap-highlight-color:transparent;touch-action:manipulation;
}
.cd-hero,.cd-tile{
  border:1px solid color-mix(in srgb,var(--tile-bg) 70%,#fff);
  border-bottom-color:color-mix(in srgb,var(--tile-bg) 75%,#030712);
  box-shadow:inset 0 2px 2px rgba(255,255,255,.5),inset 0 -5px 9px rgba(3,7,18,.22),
    inset 2px 0 4px rgba(255,255,255,.12),inset -2px 0 4px rgba(3,7,18,.12),
    0 var(--tile-depth) 0 color-mix(in srgb,var(--tile-bg) 62%,#030712),
    0 calc(var(--tile-depth) + 8px) 22px rgba(0,0,0,.32);
}
.cd-hero::before,.cd-hero::after,.cd-tile::before,.cd-tile::after{content:"";position:absolute;pointer-events:none;border-radius:inherit;z-index:0}
.cd-hero::before,.cd-tile::before{
  inset:0;
  background:linear-gradient(180deg,rgba(255,255,255,.18),rgba(255,255,255,.04) 12%,transparent 32%,transparent 78%,rgba(3,7,18,.06));
}
.cd-hero::after,.cd-tile::after{inset:4px;border:1px solid rgba(255,255,255,.14);border-bottom-color:rgba(255,255,255,.04)}
.cd-hero>*,.cd-tile>*{position:relative;z-index:1}
.cd-tile:nth-child(8n+2),.cd-tile:nth-child(8n+4),.cd-tile:nth-child(8n+5),.cd-tile:nth-child(8n+7){--tile-bg:var(--club-accent);--tile-ink:var(--club-on-accent)}
@media(hover:hover) and (pointer:fine){
  .cd-tile:hover{
    transform:translateY(-2px);
    box-shadow:inset 0 2px 3px rgba(255,255,255,.6),inset 0 -5px 9px rgba(3,7,18,.2),
      inset 2px 0 4px rgba(255,255,255,.15),inset -2px 0 4px rgba(3,7,18,.12),
      0 calc(var(--tile-depth) + 2px) 0 color-mix(in srgb,var(--tile-bg) 62%,#030712),
      0 calc(var(--tile-depth) + 12px) 26px rgba(0,0,0,.38);
  }
}
.cd-tile:active{
  transform:translateY(calc(var(--tile-depth) - 1px));
  box-shadow:inset 0 1px 4px rgba(3,7,18,.2),inset 0 -2px 3px rgba(255,255,255,.1),
    0 1px 0 color-mix(in srgb,var(--tile-bg) 62%,#030712),0 4px 10px rgba(0,0,0,.25);
}
.cd-tile:focus-visible{outline:3px solid #fff;outline-offset:3px}
@media(prefers-reduced-motion:reduce){.cd-tile{transition:none}}
.cd-tile-copy{display:flex;flex-direction:column;align-items:flex-start;min-width:0;width:100%;margin-top:auto}
.cd-icon{width:clamp(38px,3.5vw,55px);height:clamp(38px,3.5vw,55px);display:grid;place-items:center;border-radius:15px;background:linear-gradient(145deg,#fff,#e2e8f0);border:1px solid rgba(255,255,255,.8);color:var(--club-primary);font-size:clamp(18px,2vw,28px);box-shadow:inset 0 2px 2px #fff,0 3px 0 rgba(3,7,18,.15),0 8px 16px rgba(0,0,0,.2)}.cd-icon img{width:85%;height:85%;object-fit:contain}.cd-icon i{font-style:normal}.cd-tile-copy>strong{margin-top:auto;padding-top:10px;font-size:clamp(14px,1.4vw,21px);line-height:1.05;display:-webkit-box;-webkit-line-clamp:2;-webkit-box-orient:vertical;overflow:hidden}.cd-tile-copy>small{width:100%;margin-top:5px;color:inherit;font-size:clamp(9px,.8vw,12px);overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.cd-tile-copy>b{margin-top:8px;font-size:9px;letter-spacing:.12em;color:inherit}.cd-tile-copy>b i{font-size:16px;font-style:normal}
@media(max-width:900px){.cd-stage{padding:0;background:#030712}.cd-shell{display:flex;flex-direction:column;gap:0;max-width:500px}.cd-hero{flex:0 0 auto;min-height:176px;max-height:none;border-radius:22px;margin:12px 12px 0;gap:12px;padding:clamp(13px,2.2dvh,20px) 16px 14px}.cd-status,.cd-season{padding:5px 8px;font-size:8px}.cd-identity{display:grid;grid-template-columns:auto 1fr;align-items:center;gap:13px;margin:auto 0}.cd-logo{width:clamp(62px,10dvh,92px);height:clamp(62px,10dvh,92px);border-radius:20px;padding:7px}.cd-identity p{margin:0 0 4px;font-size:8px}.cd-identity h1{font-size:clamp(27px,5.3dvh,45px);line-height:.9}.cd-identity small{margin-top:7px;font-size:clamp(9px,1.45dvh,12px);white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.cd-breaking{padding:8px 10px;border-radius:10px;gap:7px}.cd-breaking b{font-size:7px}.cd-breaking span{font-size:10px}.cd-breaking i{font-size:16px}.cd-body{flex:1;min-height:0;padding:9px 9px max(9px,env(safe-area-inset-bottom));gap:8px;grid-template-rows:minmax(78px,12.5dvh) 1fr}.cd-match{min-height:0;height:100%;border-radius:15px;padding:10px 13px}.cd-match-head{font-size:8px}.cd-versus{margin:9px 0 4px;gap:7px;font-size:clamp(10px,2.8vw,14px)}.cd-versus b{font-size:clamp(14px,4vw,20px)}.cd-match>small{font-size:8px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.cd-match-empty>span{font-size:25px}.cd-match-empty strong{font-size:12px}.cd-match-empty small{font-size:8px}.cd-grid{grid-template-columns:repeat(2,minmax(0,1fr));grid-template-rows:repeat(4,minmax(0,1fr));gap:7px}.cd-tile{--tile-depth:4px;border-radius:14px;padding:clamp(8px,1.35dvh,12px)}.cd-icon{width:clamp(29px,4.5dvh,39px);height:clamp(29px,4.5dvh,39px);border-radius:10px;font-size:clamp(14px,2.5dvh,20px)}.cd-tile-copy>strong{padding-top:4px;font-size:clamp(10px,1.75dvh,14px);-webkit-line-clamp:1}.cd-tile-copy>small{margin-top:2px;font-size:clamp(7px,1.1dvh,9px)}.cd-tile-copy>b{margin-top:3px;font-size:6px}.cd-tile-copy>b i{font-size:10px}}
@media(max-width:900px) and (max-height:620px) and (orientation:portrait){.cd-hero{flex-basis:auto;min-height:140px;max-height:none;padding:9px 11px}.cd-logo{width:48px;height:48px;border-radius:14px;padding:5px}.cd-identity{gap:9px}.cd-identity h1{font-size:clamp(22px,4.7dvh,28px)}.cd-identity small{margin-top:3px;font-size:8px}.cd-breaking{padding:5px 8px}.cd-body{padding:6px;gap:6px;grid-template-rows:minmax(64px,10.5dvh) 1fr}.cd-match{padding:7px 10px}.cd-versus{margin:5px 0 2px}.cd-grid{gap:5px}.cd-tile{padding:6px}.cd-icon{width:24px;height:24px;border-radius:7px;font-size:13px}.cd-tile-copy>strong{font-size:9px}.cd-tile-copy>small,.cd-tile-copy>b{display:none}}
@media(max-width:960px) and (max-height:520px) and (orientation:landscape){.cd-stage{padding:6px}.cd-shell{max-width:none;display:grid;grid-template-columns:minmax(220px,34%) 1fr;gap:6px}.cd-hero{min-height:0;max-height:none;height:auto;margin:0;border-radius:18px;padding:10px}.cd-status,.cd-season{font-size:7px}.cd-identity{gap:8px}.cd-logo{width:48px;height:48px;border-radius:13px;padding:5px}.cd-identity h1{font-size:clamp(22px,4.8vw,34px)}.cd-identity small{margin-top:4px;font-size:8px}.cd-breaking{padding:6px 8px}.cd-body{padding:0;gap:6px;grid-template-rows:72px 1fr}.cd-match{padding:7px 10px}.cd-versus{margin:5px 0 2px}.cd-grid{grid-template-columns:repeat(4,minmax(0,1fr));grid-template-rows:repeat(2,minmax(0,1fr));gap:6px}.cd-tile{padding:7px}.cd-icon{width:26px;height:26px;border-radius:8px;font-size:14px}.cd-tile-copy>strong{font-size:10px}.cd-tile-copy>small,.cd-tile-copy>b{display:none}}
@media(min-width:901px) and (max-height:720px){.cd-hero{padding:22px}.cd-logo{width:82px;height:82px}.cd-identity p{margin-top:15px}.cd-identity h1{font-size:42px}.cd-identity small{font-size:13px}.cd-match{min-height:105px;padding:14px 18px}.cd-versus{margin:10px 0 5px}.cd-tile{padding:13px}.cd-icon{width:38px;height:38px}.cd-tile-copy>strong{font-size:14px}}
/* Tipografia Club Hub: dimensioni e contrasto leggibili senza perdere l'above the fold. */
.club-page{-webkit-font-smoothing:antialiased;text-rendering:optimizeLegibility}.cd-status,.cd-season{font-size:11px;letter-spacing:.055em}.cd-identity p{font-size:12px;letter-spacing:.12em}.cd-identity small{color:inherit;line-height:1.45}.cd-breaking b{font-size:10px}.cd-breaking span{font-size:14px;line-height:1.25}.cd-match-head{font-size:11px;letter-spacing:.055em}.cd-match>small{color:#475569;font-size:12px}.cd-tile-copy>strong{font-size:clamp(15px,1.4vw,21px);line-height:1.15}.cd-tile-copy>small{color:inherit;font-size:11px}.cd-tile-copy>b{font-size:10px;letter-spacing:.08em}
@media(max-width:900px){.cd-status,.cd-season{font-size:9px}.cd-identity p{font-size:10px}.cd-identity small{font-size:11px}.cd-breaking b{font-size:9px}.cd-breaking span{font-size:12px}.cd-match-head{font-size:9px}.cd-match>small{font-size:10px}.cd-tile-copy>strong{font-size:clamp(12px,1.8dvh,15px)}.cd-tile-copy>small{font-size:9px}.cd-tile-copy>b{font-size:8px}}
@media(max-width:900px) and (max-height:620px) and (orientation:portrait){.cd-identity small{font-size:9px}.cd-tile-copy>strong{font-size:10px}.cd-match>small{display:none}}
@media(max-width:960px) and (max-height:520px) and (orientation:landscape){.cd-tile-copy>strong{font-size:11px}.cd-match>small{display:none}}
/* Scacchiera: due colonne su mobile, quattro su desktop e in landscape compatto. */
@media(max-width:900px){
  .cd-tile:nth-child(n){--tile-bg:var(--club-primary);--tile-ink:var(--club-on-primary)}
  .cd-tile:nth-child(4n+2),.cd-tile:nth-child(4n+3){--tile-bg:var(--club-accent);--tile-ink:var(--club-on-accent)}
}
@media(max-width:960px) and (max-height:520px) and (orientation:landscape){
  .cd-tile:nth-child(n){--tile-bg:var(--club-primary);--tile-ink:var(--club-on-primary)}
  .cd-tile:nth-child(8n+2),.cd-tile:nth-child(8n+4),.cd-tile:nth-child(8n+5),.cd-tile:nth-child(8n+7){--tile-bg:var(--club-accent);--tile-ink:var(--club-on-accent)}
}
/* La tipografia del club comprende anche pulsanti e contenuti delle modali. */
.club-page :is(button,input,select,textarea){font-family:inherit}
.club-page :is(h1,h2,h3,h4,h5,h6),.cd-tile-copy>strong,.cd-versus>strong,.cd-match-empty strong,.cd-breaking>span,.fixture-teams>strong,.brand strong{font-family:var(--club-font-title)}
/* Home mobile: Hero, partita e otto pulsanti dentro il viewport, comprese le safe area. */
@media(max-width:900px),(max-width:960px) and (max-height:520px) and (orientation:landscape){
  :global(body:has(.club-home-page)),
  :global(#app:has(.club-home-page)),
  :global(.app-shell:has(.club-home-page)){height:100dvh;min-height:0;overflow:hidden}
  .club-home-page{height:100dvh;min-height:0;overflow:hidden}
  .cd-stage{
    height:100%;min-height:0;overflow:hidden;
    padding:max(8px,env(safe-area-inset-top)) max(8px,env(safe-area-inset-right)) max(8px,env(safe-area-inset-bottom)) max(8px,env(safe-area-inset-left));
  }
  .cd-shell{
    display:grid;height:100%;min-height:0;max-width:500px;
    grid-template-columns:minmax(0,1fr);
    grid-template-rows:minmax(0,clamp(116px,24dvh,205px)) minmax(0,1fr);gap:10px;
  }
  .cd-hero{
    --tile-depth:3px;display:grid;grid-template-rows:auto minmax(0,1fr) auto;
    height:100%;min-height:0;max-height:none;margin:0;gap:5px;
    padding:clamp(8px,1.5dvh,14px) 12px;border-radius:18px;
  }
  .cd-hero-top{min-width:0;gap:6px}
  .cd-status,.cd-season{padding:3px 7px;font-size:8px;line-height:1.2;letter-spacing:.04em}
  .cd-status i{width:5px;height:5px;margin-right:3px}
  .cd-identity{display:grid;grid-template-columns:auto minmax(0,1fr);align-items:center;min-height:0;margin:0;gap:10px}
  .cd-identity>div:last-child{min-width:0}
  .cd-logo{width:clamp(36px,8dvh,68px);height:clamp(36px,8dvh,68px);padding:5px;border-radius:14px;font-size:24px}
  .cd-identity p{max-width:100%;margin:0 0 3px;padding:2px 5px;font-size:8px;line-height:1.1;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
  .cd-identity h1{font-size:clamp(22px,3.8dvh,34px);line-height:1.05;letter-spacing:-.035em}
  .cd-identity small{margin-top:4px;font-size:10px;line-height:1.2;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
  .cd-breaking{min-height:0;padding:5px 8px;border-radius:8px;gap:6px}
  .cd-breaking b{font-size:7px;line-height:1.2;white-space:nowrap}
  .cd-breaking span{font-size:10px;line-height:1.2}
  .cd-breaking i{font-size:14px;line-height:1}
  .cd-body{display:grid;min-height:0;padding:0;grid-template-rows:clamp(60px,10dvh,88px) minmax(0,1fr);gap:10px}
  .cd-match{display:flex;flex-direction:column;justify-content:center;min-height:0;height:100%;padding:8px 10px;border-radius:14px}
  .cd-match-head{width:100%;flex-shrink:0;gap:8px;font-size:8px;line-height:1.2}
  .cd-match-head time{white-space:nowrap}
  .cd-versus{width:100%;min-height:0;flex-shrink:0;grid-template-columns:minmax(0,1fr) auto minmax(0,1fr);gap:6px;margin:5px 0 2px;font-size:clamp(10px,2.8vw,12px);line-height:1.15}
  .cd-versus>strong{min-width:0;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
  .cd-versus b{font-size:16px;line-height:1}
  .cd-match>small{max-width:100%;font-size:9px;line-height:1.2;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
  .cd-match-empty{flex-direction:row;justify-content:flex-start;gap:10px}
  .cd-match-empty>span{font-size:24px}
  .cd-match-empty strong{font-size:12px;line-height:1.2}
  .cd-match-empty small{font-size:9px;line-height:1.2}
  .cd-grid{min-height:0;grid-template-columns:repeat(2,minmax(0,1fr));grid-template-rows:repeat(4,minmax(0,1fr));grid-auto-rows:auto;gap:9px}
  .cd-tile{--tile-depth:3px;flex-direction:row;align-items:center;min-height:0;padding:clamp(7px,1.15dvh,11px);gap:8px;border-radius:14px}
  .cd-icon{flex:none;width:clamp(24px,4.3dvh,38px);height:clamp(24px,4.3dvh,38px);border-radius:9px;font-size:clamp(14px,2.5dvh,20px)}
  .cd-tile-copy{flex:1;min-width:0;max-width:100%;width:auto;margin:0;gap:3px}
  .cd-tile-copy>strong{
    display:block;flex:none;width:100%;margin:0;padding:0;
    font-size:clamp(11px,1.7dvh,14px);line-height:1.22;
    -webkit-line-clamp:unset;overflow:visible;white-space:normal;overflow-wrap:anywhere;
  }
  .cd-tile-copy>small{display:block;flex:none;width:100%;margin:0;font-size:9px;line-height:1.2;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
  .cd-tile-copy>b{display:none}
}
@media(max-width:900px) and (max-height:700px){
  .cd-identity small,.cd-tile-copy>small{display:none}
}
@media(max-width:900px) and (max-height:640px){
  .cd-shell{grid-template-rows:minmax(0,clamp(104px,23dvh,136px)) minmax(0,1fr);gap:8px}
  .cd-hero{padding:7px 9px;gap:4px}
  .cd-identity p{display:none}
  .cd-identity{gap:8px}
  .cd-body{grid-template-rows:clamp(54px,10dvh,64px) minmax(0,1fr);gap:8px}
  .cd-match>small{display:none}
  .cd-tile{padding:7px;gap:6px}
}
@media(max-width:900px) and (max-height:560px){
  .cd-shell{grid-template-rows:104px minmax(0,1fr);gap:6px}
  .cd-hero{padding:6px 9px;gap:3px}
  .cd-status,.cd-season{padding:2px 6px;font-size:7px}
  .cd-logo{width:32px;height:32px;border-radius:10px}
  .cd-identity h1{font-size:20px}
  .cd-breaking{padding:4px 7px}
  .cd-breaking span{font-size:9px}
  .cd-breaking i{font-size:12px}
  .cd-body{grid-template-rows:52px minmax(0,1fr);gap:6px}
  .cd-tile{padding:6px}
  .cd-tile-copy>strong{font-size:10.5px}
}
@media(max-width:960px) and (max-height:520px) and (orientation:landscape){
  .cd-shell{max-width:none;grid-template-columns:minmax(190px,32%) minmax(0,1fr);grid-template-rows:minmax(0,1fr);gap:10px}
  .cd-hero{display:flex;flex-direction:column;gap:10px;padding:10px}
  .cd-hero-top{flex-wrap:wrap;gap:4px}
  .cd-identity{display:flex;flex-direction:column;align-items:flex-start;gap:8px;margin:auto 0}
  .cd-logo{width:clamp(32px,12dvh,48px);height:clamp(32px,12dvh,48px)}
  .cd-identity h1{font-size:clamp(20px,3.2vw,30px)}
  .cd-identity p,.cd-identity small{display:none}
  .cd-body{grid-template-rows:clamp(54px,18dvh,72px) minmax(0,1fr);gap:10px}
  .cd-grid{grid-template-columns:repeat(4,minmax(0,1fr));grid-template-rows:repeat(2,minmax(0,1fr))}
  .cd-tile{flex-direction:column;align-items:flex-start;gap:5px}
  .cd-tile-copy{flex:none;width:100%;margin-top:auto}
  .cd-tile-copy>small,.cd-match>small{display:none}
}
.gallery-open{display:block;width:100%;height:100%;padding:0;border:0;background:#e2e8f0;cursor:zoom-in}.gallery-open:focus-visible{outline:3px solid var(--club-accent);outline-offset:-4px}.gallery-open img{display:block}
</style>
