<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { clubContactPhoneLink, clubInitials, formatClubDate } from '../clubHub'
import ClubMatchCard from './ClubMatchCard.vue'

const props = defineProps({
  panel: { type: String, required: true },
  club: { type: Object, default: () => ({}) },
  data: { type: Object, default: () => ({}) },
  teamData: { type: Object, default: null },
  teamTitle: { type: String, default: '' },
  teamLoading: Boolean,
  teamError: { type: String, default: '' },
  section: { type: String, default: 'overview' },
})
const emit = defineEmits(['close', 'update:section'])
const dialog = ref(null)
const isTeam = computed(() => props.panel === 'team')
const payload = computed(() => isTeam.value ? (props.teamData || {}) : props.data)
const currentView = computed(() => isTeam.value ? props.section : props.panel)
const panelTitle = computed(() => isTeam.value
  ? (props.teamData?.team?.name || props.teamTitle || 'Squadra')
  : ({ matches: 'Partite', announcements: 'Annunci', contacts: 'Contatti', sponsors: 'I nostri Sponsor', news: 'News', gallery: 'Photo gallery' }[props.panel] || 'Club'))
const sections = [
  { id: 'overview', label: 'Panoramica' }, { id: 'matches', label: 'Partite' },
  { id: 'roster', label: 'Roster' }, { id: 'news', label: 'News' }, { id: 'gallery', label: 'Gallery' },
]
const matches = computed(() => payload.value.matches || [])
const players = computed(() => payload.value.players || [])
const posts = computed(() => payload.value.posts || [])
const announcements = computed(() => posts.value.filter(post => post.kind === 'announcement'))
const news = computed(() => isTeam.value ? posts.value : posts.value.filter(post => post.kind !== 'announcement'))
const gallery = computed(() => payload.value.gallery || [])
const contacts = computed(() => props.club.contacts || [])
const sponsors = computed(() => props.data.sponsors || [])
const featuredMatch = computed(() => matches.value.find(match => match.status === 'live')
  || [...matches.value].filter(match => match.status === 'scheduled').sort((a, b) => String(a.scheduledAt || '').localeCompare(String(b.scheduledAt || '')))[0]
  || matches.value.find(match => match.status === 'completed') || null)
const stats = computed(() => [
  { section: 'matches', label: 'Partite', count: matches.value.length },
  { section: 'roster', label: 'Atleti', count: players.value.length },
  { section: 'news', label: 'Notizie', count: posts.value.length },
  { section: 'gallery', label: 'Foto', count: gallery.value.length },
])
function matchTeamName(match) {
  return payload.value.team?.name || (props.data.teams || []).find(team => team.id === match.teamId)?.name || 'Squadra'
}
function closeOutside(event) {
  if (event.target !== dialog.value) return
  const rect = dialog.value.getBoundingClientRect()
  if (event.clientX < rect.left || event.clientX > rect.right || event.clientY < rect.top || event.clientY > rect.bottom) emit('close')
}
onMounted(() => dialog.value?.showModal())
onBeforeUnmount(() => dialog.value?.close())
</script>

<template>
  <dialog ref="dialog" class="club-modal" aria-labelledby="club-content-title" @cancel.prevent="emit('close')" @click="closeOutside">
    <header class="club-modal-header">
      <h2 id="club-content-title">{{ panelTitle }}</h2>
      <button class="club-close" type="button" aria-label="Chiudi" @click="emit('close')">
        <svg viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="m6 6 12 12M18 6 6 18" /></svg>
      </button>
    </header>

    <nav v-if="isTeam && teamData && !teamLoading && !teamError" class="club-modal-tabs" aria-label="Sezioni squadra">
      <button v-for="item in sections" :key="item.id" type="button" class="club-tab" :aria-current="section === item.id ? 'page' : undefined" @click="emit('update:section', item.id)">{{ item.label }}</button>
    </nav>

    <div class="club-modal-body" :aria-busy="isTeam && teamLoading ? 'true' : undefined">
      <div v-if="isTeam && teamLoading" class="club-state" role="status"><span class="club-spinner" aria-hidden="true"></span><h3>Caricamento squadra…</h3><p>Stiamo preparando tutti i contenuti.</p></div>
      <div v-else-if="isTeam && teamError" class="club-state" role="alert"><span class="club-state-icon" aria-hidden="true">!</span><h3>Squadra non disponibile</h3><p>{{ teamError }}</p><button class="club-state-button" type="button" @click="emit('close')">Torna alla home</button></div>

      <template v-else-if="!isTeam || teamData">
        <section v-if="currentView === 'overview'" class="club-overview" aria-label="Panoramica squadra">
          <div class="club-stats">
            <button v-for="stat in stats" :key="stat.section" class="club-stat" type="button" @click="emit('update:section', stat.section)"><b>{{ stat.count }}</b><span>{{ stat.label }}</span><svg viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="M5 12h14m-6-6 6 6-6 6" /></svg></button>
          </div>
          <ClubMatchCard v-if="featuredMatch" :match="featuredMatch" :team-name="matchTeamName(featuredMatch)" featured />
          <div v-else class="club-state"><span class="club-state-icon" aria-hidden="true">↗</span><h3>Ci vediamo in campo</h3><p>Il prossimo appuntamento sarà pubblicato a breve.</p></div>
        </section>

        <section v-else-if="currentView === 'matches'" class="club-match-list" aria-label="Calendario e risultati">
          <ClubMatchCard v-for="match in matches" :key="match.id" :match="match" :team-name="matchTeamName(match)" />
          <div v-if="!matches.length" class="club-state"><span class="club-state-icon" aria-hidden="true">↗</span><h3>Il calendario prende forma</h3><p>Le prossime partite saranno disponibili qui.</p></div>
        </section>

        <section v-else-if="currentView === 'roster'" class="club-roster" aria-label="Roster squadra">
          <article v-for="player in players" :key="player.id" class="club-player">
            <div class="club-player-photo"><img v-if="player.imageUrl" :src="player.imageUrl" :alt="`${player.firstName} ${player.lastName}`" loading="lazy" /><span v-else>{{ clubInitials(`${player.firstName} ${player.lastName}`) }}</span></div>
            <div class="club-player-copy"><small>{{ player.role || 'Atleta' }}</small><h3>{{ player.firstName }} {{ player.lastName }}</h3></div>
            <b v-if="player.jerseyNumber" class="club-player-number">#{{ player.jerseyNumber }}</b>
          </article>
          <div v-if="!players.length" class="club-state"><span class="club-state-icon" aria-hidden="true">◎</span><h3>La squadra sta arrivando</h3><p>Il roster sarà pubblicato a breve.</p></div>
        </section>

        <section v-else-if="currentView === 'announcements'" class="club-post-list" aria-label="Annunci del club">
          <article v-for="post in announcements" :key="post.id" class="club-post" :class="{ 'is-pinned': post.pinned }"><div class="club-post-meta"><span class="club-badge">{{ post.pinned ? 'In evidenza' : post.teamName || 'Società' }}</span><time v-if="post.publishedAt" :datetime="post.publishedAt">{{ formatClubDate(post.publishedAt) }}</time></div><h3>{{ post.title }}</h3><p>{{ post.body || post.excerpt }}</p></article>
          <div v-if="!announcements.length" class="club-state"><span class="club-state-icon" aria-hidden="true">✦</span><h3>Tutte le novità, qui</h3><p>Gli annunci del club saranno pubblicati in questa sezione.</p></div>
        </section>

        <section v-else-if="currentView === 'news'" class="club-news" aria-label="Notizie">
          <article v-for="post in news" :key="post.id" class="club-news-card"><img v-if="post.imageUrl" :src="post.imageUrl" :alt="post.title" loading="lazy" /><div class="club-news-copy"><div class="club-post-meta"><span class="club-badge">{{ post.kind === 'announcement' ? 'Annuncio' : post.teamName || 'News' }}</span><time v-if="post.publishedAt" :datetime="post.publishedAt">{{ formatClubDate(post.publishedAt) }}</time></div><h3>{{ post.title }}</h3><p>{{ post.body || post.excerpt }}</p></div></article>
          <div v-if="!news.length" class="club-state"><span class="club-state-icon" aria-hidden="true">✦</span><h3>Le storie della squadra</h3><p>Le prossime notizie saranno disponibili qui.</p></div>
        </section>

        <section v-else-if="currentView === 'contacts'" class="club-contacts" aria-label="Contatti del club">
          <article v-for="(contact, index) in contacts" :key="index" class="club-contact"><header><span class="club-contact-avatar" aria-hidden="true">{{ clubInitials(contact.name) }}</span><div><p v-if="contact.role">{{ contact.role }}</p><h3>{{ contact.name }}</h3></div></header><div class="club-contact-links"><a v-if="contact.phone && clubContactPhoneLink(contact.phone)" :href="clubContactPhoneLink(contact.phone)"><span>Telefono</span><strong>{{ contact.phone }}</strong><i aria-hidden="true">↗</i></a><p v-else-if="contact.phone"><span>Telefono</span><strong>{{ contact.phone }}</strong></p><a v-if="contact.email" :href="`mailto:${contact.email}`"><span>Email</span><strong>{{ contact.email }}</strong><i aria-hidden="true">↗</i></a><p v-if="contact.address"><span>Indirizzo</span><strong>{{ contact.address }}</strong></p><a v-if="contact.website" :href="contact.website" target="_blank" rel="noopener noreferrer"><span>Sito web</span><strong>{{ contact.website }}</strong><i aria-hidden="true">↗</i></a></div></article>
          <div v-if="!contacts.length" class="club-state"><span class="club-state-icon" aria-hidden="true">@</span><h3>Restiamo in contatto</h3><p>I recapiti della società saranno disponibili a breve.</p></div>
        </section>

        <section v-else-if="currentView === 'gallery'" class="club-gallery" aria-label="Foto del club">
          <figure v-for="photo in gallery" :key="photo.id"><div><img :src="photo.imageUrl" :alt="photo.title || photo.caption || 'Foto del club'" loading="lazy" /></div><figcaption v-if="photo.title || photo.caption || photo.teamName"><strong v-if="photo.title">{{ photo.title }}</strong><span v-if="photo.caption || photo.teamName">{{ photo.caption || photo.teamName }}</span></figcaption></figure>
          <div v-if="!gallery.length" class="club-state"><span class="club-state-icon" aria-hidden="true">▧</span><h3>I nostri momenti migliori</h3><p>Le foto della stagione saranno raccolte qui.</p></div>
        </section>

        <section v-else-if="currentView === 'sponsors'" class="club-sponsors" aria-label="Sponsor del club">
          <a v-for="sponsor in sponsors" :key="sponsor.id" class="club-sponsor" :href="sponsor.website" target="_blank" rel="noopener noreferrer" :aria-label="`${sponsor.name}: visita il sito (si apre in una nuova scheda)`">
            <div class="club-sponsor-logo"><img :src="sponsor.logoUrl" :alt="sponsor.name" loading="lazy" /></div>
            <div class="club-sponsor-copy"><h3>{{ sponsor.name }}</h3><p v-if="sponsor.description">{{ sponsor.description }}</p><span class="club-sponsor-link">Visita il sito <svg viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="M7 17 17 7M7 7h10v10" /></svg></span></div>
          </a>
          <div v-if="!sponsors.length" class="club-state"><span class="club-state-icon" aria-hidden="true">✦</span><h3>Insieme, per il club</h3><p>I nostri sponsor saranno disponibili a breve.</p></div>
        </section>
      </template>
    </div>
  </dialog>
</template>

<style scoped>
.club-modal{--modal-primary:var(--club-primary,#172554);--modal-accent:var(--club-accent,#f59e0b);--modal-on-primary:var(--club-on-primary,#fff);--modal-on-accent:var(--club-on-accent,#111827);box-sizing:border-box;width:min(1100px,calc(100vw - 48px));max-width:none;height:min(860px,calc(100dvh - 48px));max-height:none;margin:auto;padding:0;border:1px solid #dce3ee;border-radius:28px;overflow:hidden;background:#f3f5f9;color:#0f172a;font-family:var(--club-font-body,Inter,system-ui,sans-serif);box-shadow:0 28px 80px rgba(3,7,18,.4)}
.club-modal[open]{display:flex;flex-direction:column}.club-modal::backdrop{background:rgba(3,7,18,.72)}
.club-modal *{box-sizing:border-box}.club-modal button{font-family:inherit;cursor:pointer;-webkit-tap-highlight-color:transparent;touch-action:manipulation}.club-modal :is(h2,h3){font-family:var(--club-font-title,inherit)}.club-modal :is(button,a):focus-visible{outline:3px solid #0f172a;outline-offset:3px}
.club-modal-header{flex:none;display:grid;grid-template-columns:minmax(0,1fr) 44px;align-items:center;gap:18px;padding:22px 24px 25px;background:var(--modal-primary);color:var(--modal-on-primary);border-top:1px solid color-mix(in srgb,var(--modal-primary) 60%,#fff);box-shadow:inset 0 2px 2px rgba(255,255,255,.35),inset 0 -5px 0 color-mix(in srgb,var(--modal-primary) 62%,#030712)}
.club-modal-header h2{min-width:0;margin:0;font-size:clamp(24px,3vw,34px);line-height:1.15;letter-spacing:-.035em;overflow-wrap:anywhere}
.club-close{display:grid;place-items:center;width:44px;height:44px;padding:0;border:1px solid color-mix(in srgb,var(--modal-accent) 65%,#fff);border-radius:14px;background:var(--modal-accent);color:var(--modal-on-accent);box-shadow:inset 0 2px 1px rgba(255,255,255,.45),0 3px 0 color-mix(in srgb,var(--modal-accent) 60%,#030712)}.club-close svg{display:block;width:22px;height:22px;stroke:currentColor;stroke-width:2.2;stroke-linecap:round}.club-modal-header .club-close:focus-visible{outline-color:var(--modal-on-primary)}
.club-modal-tabs{flex:none;display:grid;grid-template-columns:repeat(5,minmax(0,1fr));grid-template-rows:minmax(46px,auto);grid-auto-flow:column;align-items:stretch;min-width:0;max-width:100%;gap:10px;padding:18px 24px 20px;border-bottom:1px solid #dfe5ef;background:#eef1f6}
.club-tab{display:flex;align-items:center;justify-content:center;min-width:0;max-width:100%;min-height:46px;margin:0;padding:12px 10px;border:1px solid #d6deeb;border-radius:13px;background:#fff;color:#475569;text-align:center;font-size:13px;font-weight:850;line-height:1.2;white-space:normal;overflow-wrap:anywhere;box-shadow:inset 0 2px 1px #fff,0 3px 0 #cdd5e1;transition:transform .15s ease,box-shadow .15s ease}
.club-tab[aria-current="page"]{background:var(--modal-primary);color:var(--modal-on-primary);border-color:color-mix(in srgb,var(--modal-primary) 70%,#fff);box-shadow:inset 0 2px 1px rgba(255,255,255,.35),0 3px 0 color-mix(in srgb,var(--modal-primary) 60%,#030712)}
.club-modal-body{flex:1;min-height:0;overflow:auto;overscroll-behavior:contain;scrollbar-gutter:stable;padding:24px 24px 30px}
.club-overview,.club-match-list,.club-post-list{display:grid;align-content:start;gap:20px}.club-match-list{grid-template-columns:repeat(2,minmax(0,1fr))}.club-stats{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:14px;padding-bottom:4px}
.club-stat{--stat-bg:var(--modal-primary);--stat-ink:var(--modal-on-primary);position:relative;min-width:0;display:flex;flex-direction:column;align-items:flex-start;gap:7px;padding:22px;border:1px solid color-mix(in srgb,var(--stat-bg) 65%,#fff);border-radius:20px;background:var(--stat-bg);color:var(--stat-ink);text-align:left;box-shadow:inset 0 2px 2px rgba(255,255,255,.4),inset 0 -4px 8px rgba(3,7,18,.13),0 4px 0 color-mix(in srgb,var(--stat-bg) 62%,#030712),0 9px 16px rgba(15,23,42,.1);transition:transform .15s ease}
.club-stat:nth-child(even){--stat-bg:var(--modal-accent);--stat-ink:var(--modal-on-accent)}.club-stat>b{font-family:var(--club-font-title,inherit);font-size:36px;line-height:1.1}.club-stat>span{font-size:13px;font-weight:750;line-height:1.3}.club-stat svg{position:absolute;top:20px;right:17px;width:19px;height:19px;stroke:currentColor;stroke-width:1.8;stroke-linecap:round;stroke-linejoin:round}
.club-post,.club-news-card,.club-contact,.club-player,.club-gallery figure{min-width:0;border:1px solid #dbe2ec;border-radius:22px;background:#fff;box-shadow:inset 0 2px 0 #fff,0 4px 0 #dce2eb,0 10px 22px rgba(15,23,42,.05)}
.club-post{padding:24px}.club-post.is-pinned{border-top:4px solid var(--modal-accent)}.club-post-meta{display:flex;align-items:center;justify-content:space-between;flex-wrap:wrap;gap:10px}.club-post-meta time{color:#64748b;font-size:11px;line-height:1.5}.club-badge{display:inline-flex;align-items:center;max-width:100%;padding:6px 9px;border-radius:8px;background:var(--modal-primary);color:var(--modal-on-primary);font-size:10px;font-weight:850;line-height:1.2;overflow-wrap:anywhere}.is-pinned .club-badge{background:var(--modal-accent);color:var(--modal-on-accent)}
.club-post h3,.club-news-copy h3{margin:16px 0 10px;font-size:22px;line-height:1.25;letter-spacing:-.025em;overflow-wrap:anywhere}.club-post>p,.club-news-copy>p{margin:0;color:#475569;font-size:14px;line-height:1.75;white-space:pre-line;overflow-wrap:anywhere}
.club-roster,.club-news,.club-contacts{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));align-content:start;gap:18px}
.club-player{display:grid;grid-template-columns:72px minmax(0,1fr) auto;align-items:center;gap:14px;padding:14px;border-radius:18px}.club-player-photo{display:grid;place-items:center;width:72px;height:88px;overflow:hidden;border-radius:13px;background:var(--modal-primary);color:var(--modal-on-primary);font-family:var(--club-font-title,inherit);font-size:24px;font-weight:850;box-shadow:inset 0 2px 2px rgba(255,255,255,.3),0 3px 0 color-mix(in srgb,var(--modal-primary) 62%,#030712)}.club-player-photo img{width:100%;height:100%;object-fit:cover}.club-player-copy{min-width:0}.club-player-copy small{color:#64748b;font-size:10px;line-height:1.5;text-transform:uppercase;letter-spacing:.06em}.club-player-copy h3{margin:6px 0 0;font-size:17px;line-height:1.3;overflow-wrap:anywhere}.club-player-number{align-self:start;display:grid;place-items:center;min-width:34px;min-height:30px;padding:5px;border-radius:8px;background:var(--modal-accent);color:var(--modal-on-accent);font-size:12px;line-height:1.1}
.club-news-card{overflow:hidden}.club-news-card>img{display:block;width:100%;aspect-ratio:16/9;object-fit:cover}.club-news-copy{padding:22px}
.club-contact{padding:22px}.club-contact>header{display:flex;align-items:center;gap:14px;margin-bottom:20px}.club-contact>header>div{min-width:0}.club-contact-avatar{flex:none;display:grid;place-items:center;width:52px;height:52px;border-radius:15px;background:var(--modal-primary);color:var(--modal-on-primary);font-family:var(--club-font-title,inherit);font-size:18px;font-weight:900;box-shadow:inset 0 2px 2px rgba(255,255,255,.35),0 3px 0 color-mix(in srgb,var(--modal-primary) 62%,#030712)}.club-contact header p{margin:0 0 4px;color:#64748b;font-size:11px;line-height:1.5}.club-contact h3{margin:0;font-size:21px;line-height:1.25;overflow-wrap:anywhere}
.club-contact-links{display:grid;gap:9px}.club-contact-links>:is(a,p){position:relative;display:block;min-width:0;margin:0;padding:12px 36px 12px 13px;border:1px solid #e3e8f0;border-radius:12px;background:#f7f9fc;color:#334155;text-decoration:none;transition:border-color .15s ease}.club-contact-links span{display:block;margin-bottom:4px;color:#64748b;font-size:10px;font-weight:800;text-transform:uppercase;letter-spacing:.05em}.club-contact-links strong{font-size:13px;font-weight:600;line-height:1.5;white-space:pre-line;overflow-wrap:anywhere}.club-contact-links i{position:absolute;right:13px;top:50%;transform:translateY(-50%);font-size:19px;font-style:normal;color:#64748b}
.club-gallery{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));align-content:start;gap:18px}.club-gallery figure{margin:0;overflow:hidden;border-radius:18px}.club-gallery figure>div{aspect-ratio:4/3;background:#e2e8f0;overflow:hidden}.club-gallery img{display:block;width:100%;height:100%;object-fit:cover}.club-gallery figcaption{display:grid;gap:5px;padding:14px 16px;overflow-wrap:anywhere}.club-gallery figcaption strong{font-family:var(--club-font-title,inherit);font-size:14px;line-height:1.3}.club-gallery figcaption span{color:#64748b;font-size:11px;line-height:1.5}
.club-sponsors{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));align-content:start;gap:20px}.club-sponsor{min-width:0;display:flex;flex-direction:column;overflow:hidden;border:1px solid #dbe2ec;border-radius:22px;background:#fff;color:#0f172a;text-decoration:none;box-shadow:inset 0 2px 0 #fff,0 4px 0 #dce2eb,0 10px 22px rgba(15,23,42,.05);transition:border-color .15s ease,transform .15s ease}.club-sponsor-logo{display:grid;place-items:center;min-width:0;height:180px;padding:28px;border-bottom:1px solid #edf0f5;background:#fff}.club-sponsor-logo img{display:block;width:100%;height:100%;max-height:124px;object-fit:contain}.club-sponsor-copy{flex:1;min-width:0;display:flex;flex-direction:column;align-items:flex-start;padding:22px}.club-sponsor h3{margin:0;font-size:22px;line-height:1.25;letter-spacing:-.025em;overflow-wrap:anywhere}.club-sponsor-copy>p{margin:12px 0 0;color:#475569;font-size:14px;line-height:1.7;white-space:pre-line;overflow-wrap:anywhere}.club-sponsor-link{display:inline-flex;align-items:center;gap:8px;min-height:40px;margin-top:auto;padding-top:16px;color:var(--modal-primary);font-size:13px;font-weight:800;line-height:1.4}.club-sponsor-link svg{flex:none;width:18px;height:18px;stroke:currentColor;stroke-width:1.8;stroke-linecap:round;stroke-linejoin:round}
@media(hover:hover) and (pointer:fine){.club-sponsor:hover{border-color:var(--modal-primary);transform:translateY(-2px)}}
@media(max-width:700px){.club-sponsors{grid-template-columns:minmax(0,1fr);gap:18px}.club-sponsor-logo{height:160px;padding:24px}.club-sponsor-copy{padding:18px}.club-sponsor h3{font-size:20px}}
.club-state{grid-column:1/-1;min-height:260px;display:flex;flex-direction:column;align-items:center;justify-content:center;gap:12px;padding:30px 20px;text-align:center;border:1px dashed #ccd5e3;border-radius:22px;background:#fff}.club-state-icon{display:grid;place-items:center;width:58px;height:58px;border-radius:18px;background:var(--modal-accent);color:var(--modal-on-accent);font-size:28px;font-weight:800;box-shadow:inset 0 2px 2px rgba(255,255,255,.5),0 4px 0 color-mix(in srgb,var(--modal-accent) 65%,#030712)}.club-state h3{margin:6px 0 0;font-size:22px;line-height:1.25}.club-state p{max-width:380px;margin:0;color:#64748b;font-size:14px;line-height:1.6}.club-state-button{display:grid;place-items:center;min-height:44px;margin-top:8px;padding:10px 18px;border:1px solid color-mix(in srgb,var(--modal-primary) 65%,#fff);border-radius:12px;background:var(--modal-primary);color:var(--modal-on-primary);font-size:13px;font-weight:800;line-height:1.2;box-shadow:inset 0 2px 1px rgba(255,255,255,.3),0 3px 0 color-mix(in srgb,var(--modal-primary) 62%,#030712)}
.club-spinner{width:34px;height:34px;border:3px solid #e2e8f0;border-top-color:var(--modal-primary);border-radius:50%;animation:club-modal-spin .75s linear infinite}@keyframes club-modal-spin{to{transform:rotate(360deg)}}
.club-close:active,.club-tab:active,.club-state-button:active{transform:translateY(2px)}.club-stat:active{transform:translateY(3px)}
@media(hover:hover) and (pointer:fine){.club-tab:hover,.club-stat:hover{transform:translateY(-1px)}.club-contact-links>a:hover{border-color:#94a3b8;background:#fff}}
@media(max-width:700px){.club-modal{width:100vw;height:100dvh;border:0;border-radius:0}.club-modal-header{padding:max(16px,env(safe-area-inset-top)) max(16px,env(safe-area-inset-right)) 19px max(16px,env(safe-area-inset-left));gap:12px}.club-modal-header h2{font-size:25px}.club-modal-tabs{padding:14px 16px 17px;gap:8px}.club-tab{padding:11px 6px;font-size:12px}.club-modal-body{padding:18px max(16px,env(safe-area-inset-right)) max(28px,env(safe-area-inset-bottom)) max(16px,env(safe-area-inset-left));scrollbar-gutter:auto}.club-match-list,.club-roster,.club-news,.club-contacts{grid-template-columns:minmax(0,1fr)}.club-stats{grid-template-columns:repeat(2,minmax(0,1fr));gap:12px}.club-stat{padding:18px}.club-stat>b{font-size:32px}.club-gallery{grid-template-columns:repeat(2,minmax(0,1fr));gap:12px}.club-post,.club-contact,.club-news-copy{padding:18px}.club-post h3,.club-news-copy h3{font-size:20px}.club-player{grid-template-columns:58px minmax(0,1fr) auto;gap:11px;padding:12px}.club-player-photo{width:58px;height:74px}.club-player-copy h3{font-size:16px}.club-state{min-height:230px;padding:26px 16px}.club-state h3{font-size:21px}}
/* Cinque pulsanti sempre nella stessa riga, anche sui telefoni più stretti. */
@media(max-width:560px){
  .club-modal-tabs{grid-template-columns:minmax(0,1.6fr) repeat(4,minmax(0,1fr));grid-template-rows:minmax(44px,auto);gap:clamp(4px,1.4vw,8px);padding:12px max(10px,env(safe-area-inset-right)) 16px max(10px,env(safe-area-inset-left))}
  .club-tab{min-height:44px;padding:9px 3px;border-radius:10px;font-size:clamp(10px,2.7vw,12px)}
}
@media(prefers-reduced-motion:reduce){.club-modal *{transition:none;animation:none}}
</style>
