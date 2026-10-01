<script setup>
import { computed, onMounted, ref, watch } from 'vue'

const props = defineProps({ slug: { type: String, required: true }, teamSlug: { type: String, default: '' } })
const data = ref(null)
const loading = ref(true)
const error = ref('')

const club = computed(() => data.value?.club ?? {})
const isTeam = computed(() => Boolean(props.teamSlug))
const teamsById = computed(() => Object.fromEntries((data.value?.teams ?? []).map(team => [team.id, team])))
const announcements = computed(() => (data.value?.posts ?? []).filter(post => post.kind === 'announcement'))
const news = computed(() => (data.value?.posts ?? []).filter(post => post.kind !== 'announcement'))
const upcoming = computed(() => (data.value?.matches ?? []).filter(match => match.status === 'scheduled').sort((a,b) => a.scheduledAt.localeCompare(b.scheduledAt)).slice(0, 6))
const results = computed(() => (data.value?.matches ?? []).filter(match => match.status === 'completed').slice(0, 8))

const theme = computed(() => ({
  '--club-primary': club.value.primaryColor || '#172554',
  '--club-accent': club.value.secondaryColor || '#f59e0b',
}))

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

function formatDate(value) {
  if (!value) return 'Data da definire'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat('it-IT', { day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' }).format(date)
}
function initials(name) { return String(name || '').split(/\s+/).map(part => part[0]).join('').slice(0,2).toUpperCase() }
function teamName(match) { return data.value?.team?.name || teamsById.value[match.teamId]?.name || 'Squadra' }
function scoreLabel(match) { return match.home ? `${match.scoreFor} – ${match.scoreAgainst}` : `${match.scoreAgainst} – ${match.scoreFor}` }

watch(() => [props.slug, props.teamSlug], load)
onMounted(load)
</script>

<template>
  <main class="club-page" :style="theme">
    <div v-if="loading" class="state">Caricamento Club Hub…</div>
    <div v-else-if="error" class="state error"><strong>Pagina non disponibile</strong><span>{{ error }}</span></div>
    <template v-else>
      <header class="topbar">
        <a class="brand" :href="`/club/${slug}`">
          <img v-if="club.logoUrl" :src="club.logoUrl" :alt="club.name" />
          <span v-else class="logo-fallback">{{ initials(club.name) }}</span>
          <span><strong>{{ club.name }}</strong><small>{{ club.seasonLabel }}</small></span>
        </a>
        <nav><a :href="`/club/${slug}`">Home</a><a :href="`/club/${slug}#squadre`">Squadre</a><a :href="`/club/${slug}#notizie`">Notizie</a><a :href="`/club/${slug}#gallery`">Gallery</a></nav>
      </header>

      <template v-if="!isTeam">
        <section class="hero" :style="club.heroImageUrl ? { backgroundImage: `linear-gradient(110deg, rgba(4,10,24,.94), rgba(4,10,24,.32)), url('${club.heroImageUrl}')` } : {}">
          <div class="hero-copy"><p class="kicker">IL PORTALE UFFICIALE</p><h1>{{ club.name }}</h1><p>{{ club.tagline || 'Tutte le squadre, tutta la stagione, in un unico posto.' }}</p><div class="hero-meta"><span>{{ club.city }}</span><span>{{ club.seasonLabel }}</span></div><a class="cta" href="#squadre">Scopri le squadre</a></div>
          <div class="hero-mark"><img v-if="club.logoUrl" :src="club.logoUrl" alt="" /><span v-else>{{ initials(club.name) }}</span></div>
        </section>

        <section v-if="announcements.length" class="announcements shell">
          <article v-for="post in announcements.slice(0,3)" :key="post.id" :class="['announcement', { pinned: post.pinned }]">
            <span>{{ post.pinned ? 'IN EVIDENZA' : 'ANNUNCIO' }}</span><div><h3>{{ post.title }}</h3><p>{{ post.excerpt || post.body }}</p></div><time>{{ formatDate(post.publishedAt) }}</time>
          </article>
        </section>

        <section v-if="upcoming.length" class="section shell">
          <div class="section-heading"><div><p class="kicker">AGENDA</p><h2>Prossime partite</h2></div></div>
          <div class="match-strip">
            <article v-for="match in upcoming" :key="match.id" class="match-card">
              <span class="team-label">{{ teamName(match) }}</span><time>{{ formatDate(match.scheduledAt) }}</time><div class="versus"><strong>{{ match.home ? teamName(match) : match.opponent }}</strong><b>VS</b><strong>{{ match.home ? match.opponent : teamName(match) }}</strong></div><small>{{ match.location || 'Luogo da definire' }} · {{ match.roundLabel }}</small>
            </article>
          </div>
        </section>

        <section id="squadre" class="section shell">
          <div class="section-heading"><div><p class="kicker">UNA SOLA SOCIETÀ</p><h2>Le nostre squadre</h2></div><p>Segui calendario, risultati e roster di ogni categoria.</p></div>
          <div class="team-grid">
            <a v-for="team in data.teams" :key="team.id" class="team-card" :href="`/club/${slug}/team/${team.slug}`" :style="team.heroImageUrl ? { backgroundImage: `linear-gradient(180deg, rgba(6,12,24,.08), rgba(6,12,24,.94)), url('${team.heroImageUrl}')` } : {}">
              <div class="team-badge"><img v-if="team.logoUrl" :src="team.logoUrl" alt="" /><span v-else>{{ initials(team.name) }}</span></div><div class="team-copy"><span>{{ team.category || 'VOLLEY' }}</span><h3>{{ team.name }}</h3><p>{{ team.championship || team.description || 'Scopri la squadra' }}</p><b>Entra nella squadra →</b></div>
            </a>
          </div>
        </section>

        <section id="notizie" v-if="news.length" class="section shell">
          <div class="section-heading"><div><p class="kicker">DAL CLUB</p><h2>Ultime notizie</h2></div></div>
          <div class="news-grid"><article v-for="post in news.slice(0,6)" :key="post.id" class="news-card"><div v-if="post.imageUrl" class="news-image" :style="{ backgroundImage: `url('${post.imageUrl}')` }"></div><div class="news-body"><span>{{ post.teamName || 'Società' }}</span><h3>{{ post.title }}</h3><p>{{ post.excerpt || post.body }}</p><time>{{ formatDate(post.publishedAt) }}</time></div></article></div>
        </section>

        <section id="gallery" v-if="data.gallery?.length" class="section gallery-section">
          <div class="shell"><div class="section-heading light"><div><p class="kicker">DENTRO IL CLUB</p><h2>Photo gallery</h2></div></div><div class="gallery-grid"><figure v-for="photo in data.gallery.slice(0,12)" :key="photo.id"><img :src="photo.imageUrl" :alt="photo.title || photo.caption" loading="lazy" /><figcaption><strong>{{ photo.title }}</strong><span>{{ photo.teamName || photo.caption }}</span></figcaption></figure></div></div>
        </section>
      </template>

      <template v-else>
        <section class="team-hero" :style="data.team.heroImageUrl ? { backgroundImage: `linear-gradient(110deg, rgba(4,10,24,.95), rgba(4,10,24,.3)), url('${data.team.heroImageUrl}')` } : {}">
          <div class="shell"><a class="back" :href="`/club/${slug}`">← Tutte le squadre</a><div class="team-title"><div class="team-badge large"><img v-if="data.team.logoUrl" :src="data.team.logoUrl" alt="" /><span v-else>{{ initials(data.team.name) }}</span></div><div><p class="kicker">{{ data.team.category }}</p><h1>{{ data.team.name }}</h1><p>{{ data.team.championship }}</p></div></div></div>
        </section>
        <div class="team-content shell">
          <section class="section compact"><div class="section-heading"><div><p class="kicker">STAGIONE</p><h2>Calendario e risultati</h2></div></div><div class="fixture-list"><article v-for="match in data.matches" :key="match.id" class="fixture"><div><time>{{ formatDate(match.scheduledAt) }}</time><small>{{ match.competition }} · {{ match.roundLabel }}</small></div><div class="fixture-teams"><strong>{{ match.home ? data.team.name : match.opponent }}</strong><b :class="{ completed: match.status === 'completed' }">{{ match.status === 'completed' ? scoreLabel(match) : 'VS' }}</b><strong>{{ match.home ? match.opponent : data.team.name }}</strong></div><span :class="['status', match.status]">{{ match.status === 'completed' ? 'Conclusa' : match.status === 'live' ? 'Live' : 'Programmata' }}</span></article><p v-if="!data.matches?.length" class="empty">Il calendario sarà pubblicato a breve.</p></div></section>
          <section class="section compact"><div class="section-heading"><div><p class="kicker">LA SQUADRA</p><h2>Roster</h2></div></div><div class="roster-grid"><article v-for="player in data.players" :key="player.id" class="player"><div class="player-photo"><img v-if="player.imageUrl" :src="player.imageUrl" :alt="`${player.firstName} ${player.lastName}`" /><span v-else>{{ player.jerseyNumber || initials(`${player.firstName} ${player.lastName}`) }}</span></div><div><small>{{ player.role }}</small><h3>{{ player.firstName }} {{ player.lastName }}</h3><b v-if="player.jerseyNumber">#{{ player.jerseyNumber }}</b></div></article><p v-if="!data.players?.length" class="empty">Roster in aggiornamento.</p></div></section>
          <section v-if="data.posts?.length" class="section compact"><div class="section-heading"><div><p class="kicker">NOTIZIE</p><h2>Dalla squadra</h2></div></div><div class="news-grid"><article v-for="post in data.posts.slice(0,6)" :key="post.id" class="news-card"><div v-if="post.imageUrl" class="news-image" :style="{ backgroundImage: `url('${post.imageUrl}')` }"></div><div class="news-body"><h3>{{ post.title }}</h3><p>{{ post.excerpt || post.body }}</p></div></article></div></section>
          <section v-if="data.gallery?.length" class="section compact"><div class="section-heading"><div><p class="kicker">GALLERY</p><h2>Momenti della squadra</h2></div></div><div class="gallery-grid team-gallery"><figure v-for="photo in data.gallery" :key="photo.id"><img :src="photo.imageUrl" :alt="photo.title" loading="lazy" /></figure></div></section>
        </div>
      </template>

      <footer><div class="brand"><img v-if="club.logoUrl" :src="club.logoUrl" alt="" /><span><strong>{{ club.name }}</strong><small>{{ club.city }}</small></span></div><p>Club Hub powered by ArenaBoostX</p></footer>
    </template>
  </main>
</template>

<style scoped>
*{box-sizing:border-box}.club-page{min-height:100vh;background:#f5f3ee;color:#0f172a;font-family:Inter,ui-sans-serif,system-ui,-apple-system,sans-serif}.shell{width:min(1180px,calc(100% - 36px));margin-inline:auto}.state{min-height:100vh;display:grid;place-content:center;gap:8px;text-align:center;background:#07101f;color:#fff}.state.error span{color:#94a3b8}.topbar{height:80px;padding:0 max(24px,calc((100vw - 1180px)/2));display:flex;align-items:center;justify-content:space-between;background:#07101f;color:#fff;position:relative;z-index:5}.brand{display:flex;align-items:center;gap:11px;color:inherit;text-decoration:none}.brand img,.logo-fallback{width:46px;height:46px;object-fit:contain;border-radius:12px;background:#fff;padding:4px}.logo-fallback{display:grid;place-items:center;color:var(--club-primary);font-weight:900}.brand span{display:flex;flex-direction:column}.brand small{color:#94a3b8}.topbar nav{display:flex;gap:26px}.topbar nav a{color:#cbd5e1;text-decoration:none;font-size:13px;font-weight:700}.hero{min-height:610px;background:linear-gradient(135deg,#07101f,var(--club-primary));background-size:cover;background-position:center;display:flex;align-items:center;justify-content:space-between;padding:80px max(24px,calc((100vw - 1180px)/2));color:#fff}.hero-copy{max-width:700px}.kicker{margin:0 0 10px;color:var(--club-accent);font-size:11px;font-weight:950;letter-spacing:.18em}.hero h1,.team-hero h1{font-size:clamp(48px,8vw,96px);line-height:.94;letter-spacing:-.06em;margin:0;max-width:850px}.hero-copy>p:not(.kicker){font-size:clamp(18px,2.2vw,28px);max-width:650px;color:#dbe4ef}.hero-meta{display:flex;gap:10px;margin:25px 0}.hero-meta span{border:1px solid rgba(255,255,255,.25);border-radius:999px;padding:8px 12px;font-size:12px}.cta{display:inline-block;padding:14px 22px;border-radius:999px;background:var(--club-accent);color:#111827;font-weight:900;text-decoration:none}.hero-mark{width:260px;height:260px;border-radius:50%;display:grid;place-items:center;background:rgba(255,255,255,.1);backdrop-filter:blur(10px);border:1px solid rgba(255,255,255,.2);font-size:58px;font-weight:950}.hero-mark img{width:75%;height:75%;object-fit:contain}.announcements{margin-top:-36px;position:relative;z-index:2;display:grid;gap:8px}.announcement{display:grid;grid-template-columns:100px 1fr auto;gap:18px;align-items:center;background:#fff;border-radius:14px;padding:18px 22px;box-shadow:0 16px 45px rgba(15,23,42,.12)}.announcement>span{font-size:10px;font-weight:900;color:var(--club-primary)}.announcement.pinned>span{color:#b45309}.announcement h3,.announcement p{margin:0}.announcement p{color:#64748b;font-size:13px;margin-top:4px}.announcement time{color:#94a3b8;font-size:12px}.section{padding:86px 0}.section.compact{padding:62px 0;border-bottom:1px solid #d9d6cf}.section-heading{display:flex;align-items:end;justify-content:space-between;gap:30px;margin-bottom:28px}.section-heading h2{font-size:clamp(32px,4vw,54px);letter-spacing:-.045em;margin:0}.section-heading>p{max-width:420px;color:#64748b}.match-strip{display:grid;grid-template-columns:repeat(3,1fr);gap:14px}.match-card{background:#fff;border-radius:16px;padding:20px;border:1px solid #e4e1da}.match-card .team-label{color:var(--club-primary);font-size:11px;font-weight:900;text-transform:uppercase}.match-card time{display:block;color:#64748b;font-size:12px;margin:7px 0 18px}.versus{display:grid;grid-template-columns:1fr auto 1fr;align-items:center;gap:10px}.versus strong:last-child{text-align:right}.versus b{color:var(--club-accent)}.match-card small{display:block;margin-top:18px;color:#94a3b8}.team-grid{display:grid;grid-template-columns:repeat(2,1fr);gap:16px}.team-card{min-height:390px;border-radius:22px;padding:28px;display:flex;flex-direction:column;justify-content:space-between;background:linear-gradient(145deg,var(--club-primary),#07101f);background-size:cover;background-position:center;color:#fff;text-decoration:none;overflow:hidden}.team-badge{width:64px;height:64px;border-radius:17px;background:#fff;display:grid;place-items:center;color:var(--club-primary);font-weight:950}.team-badge img{width:82%;height:82%;object-fit:contain}.team-copy span,.news-body>span{color:var(--club-accent);font-size:10px;font-weight:900;letter-spacing:.12em}.team-copy h3{font-size:36px;line-height:1;margin:8px 0}.team-copy p{color:#cbd5e1}.team-copy b{font-size:13px}.news-grid{display:grid;grid-template-columns:repeat(3,1fr);gap:16px}.news-card{background:#fff;border:1px solid #e4e1da;border-radius:16px;overflow:hidden}.news-image{height:190px;background-size:cover;background-position:center}.news-body{padding:20px}.news-body h3{font-size:20px;margin:7px 0}.news-body p{color:#64748b;display:-webkit-box;-webkit-line-clamp:3;-webkit-box-orient:vertical;overflow:hidden}.news-body time{font-size:11px;color:#94a3b8}.gallery-section{background:#07101f}.section-heading.light h2{color:#fff}.gallery-grid{display:grid;grid-template-columns:repeat(4,1fr);gap:10px}.gallery-grid figure{margin:0;border-radius:13px;overflow:hidden;position:relative;aspect-ratio:1}.gallery-grid img{width:100%;height:100%;object-fit:cover}.gallery-grid figcaption{position:absolute;inset:auto 0 0;padding:34px 14px 12px;background:linear-gradient(transparent,rgba(0,0,0,.8));display:flex;flex-direction:column;color:#fff}.gallery-grid figcaption span{font-size:11px;color:#cbd5e1}.team-hero{min-height:430px;background:linear-gradient(135deg,#07101f,var(--club-primary));background-size:cover;background-position:center;color:#fff;padding:60px 0}.back{color:#cbd5e1;text-decoration:none}.team-title{display:flex;align-items:center;gap:24px;margin-top:80px}.team-badge.large{width:100px;height:100px;border-radius:24px}.team-hero h1{font-size:clamp(42px,7vw,82px)}.team-hero p{color:#cbd5e1}.fixture-list{display:grid;gap:8px}.fixture{display:grid;grid-template-columns:1fr 2fr auto;align-items:center;gap:20px;background:#fff;border:1px solid #e4e1da;border-radius:13px;padding:17px}.fixture time,.fixture small{display:block}.fixture small{color:#94a3b8;margin-top:4px}.fixture-teams{display:grid;grid-template-columns:1fr 70px 1fr;align-items:center;text-align:center}.fixture-teams strong:first-child{text-align:right}.fixture-teams strong:last-child{text-align:left}.fixture-teams b{font-size:18px}.fixture-teams b.completed{font-size:24px;color:var(--club-primary)}.status{font-size:10px;font-weight:900;text-transform:uppercase;border-radius:999px;padding:7px 10px;background:#e2e8f0}.status.live{background:#fee2e2;color:#b91c1c}.status.completed{background:#dcfce7;color:#166534}.roster-grid{display:grid;grid-template-columns:repeat(3,1fr);gap:12px}.player{display:flex;align-items:center;gap:15px;background:#fff;border-radius:14px;padding:14px;border:1px solid #e4e1da}.player-photo{width:70px;height:82px;border-radius:10px;background:linear-gradient(145deg,var(--club-primary),#07101f);overflow:hidden;display:grid;place-items:center;color:#fff;font-size:25px;font-weight:950}.player-photo img{width:100%;height:100%;object-fit:cover}.player h3{margin:3px 0}.player small{color:#64748b;text-transform:uppercase}.player b{color:var(--club-accent)}.team-gallery{grid-template-columns:repeat(5,1fr)}.empty{color:#94a3b8}footer{display:flex;align-items:center;justify-content:space-between;padding:32px max(24px,calc((100vw - 1180px)/2));background:#030712;color:#fff}footer p{color:#64748b;font-size:12px}@media(max-width:800px){.topbar{height:68px}.topbar nav{display:none}.hero{min-height:560px;padding-block:60px}.hero-mark{display:none}.announcement{grid-template-columns:1fr}.announcement time{display:none}.section{padding:58px 0}.section-heading{display:block}.match-strip,.news-grid,.roster-grid{grid-template-columns:1fr}.team-grid{grid-template-columns:1fr}.team-card{min-height:330px}.gallery-grid,.team-gallery{grid-template-columns:repeat(2,1fr)}.fixture{grid-template-columns:1fr}.fixture-teams{grid-template-columns:1fr 50px 1fr}.team-title{margin-top:60px}.team-badge.large{width:76px;height:76px;flex:none}footer{align-items:flex-start;gap:20px;flex-direction:column}}
</style>
