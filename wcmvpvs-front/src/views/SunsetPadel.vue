<template>
  <div class="sp-stage">
    <div class="sp-phone">
      <!-- GLITTER -->
      <div class="sp-glitter">
        <span v-for="(s, i) in sparks" :key="i" class="sp-spark"
              :style="{ top: s.top, left: s.left, fontSize: s.size, color: s.color, animationDelay: s.delay }">✦</span>
      </div>

      <!-- HERO -->
      <div class="sp-hero">
        <div class="sp-sun"></div>
        <div class="sp-nightsky"></div>
        <span v-for="(st, i) in stars" :key="'st'+i" class="sp-star"
              :style="{ top: st.top, left: st.left, fontSize: st.size, animationDelay: st.delay }">✦</span>
        <div class="sp-horizon"></div>
        <div class="sp-wave"></div>
        <button class="sp-ball" type="button" aria-label="Premi" @click="showPrizes = true">🏆</button>

        <div class="sp-top">
          <span class="sp-status"><span class="sp-dot"></span> {{ status }}</span>
        </div>

        <div class="sp-titlewrap" :class="{ 'has-logo': logo }">
          <img v-if="logo" :src="logo" class="sp-logo" :alt="`${brandTop} ${brandBottom}`" />
          <h1 v-else class="sp-brand">{{ brandTop }}<span class="sp-pad">{{ brandBottom }}</span></h1>
        </div>

        <div class="sp-meta">
          <div class="sp-chip">📅 {{ date }}</div>
          <div class="sp-chip">📍 {{ place }}</div>
        </div>

      </div>

      <!-- BODY -->
      <div class="sp-body">
        <!-- LIVE: se ci sono partite in corso prendono il posto di "Prossima
             partita"; scroll orizzontale se più campi sono live insieme. -->
        <div v-if="liveMatches.length" class="sp-live-row">
          <div v-for="m in liveMatches" :key="m.id" class="sp-match sp-match-live">
            <div class="sp-match-head">
              <span class="sp-match-label live"><span class="sp-live-dot"></span> LIVE · {{ m.court }}</span>
              <span class="sp-match-time">{{ m.setLabel }}</span>
            </div>
            <div class="sp-teams">
              <div class="sp-team">{{ m.teamA.name }}</div>
              <div class="sp-live-score">{{ m.cur ? m.cur.a : m.score.a }}<span>:</span>{{ m.cur ? m.cur.b : m.score.b }}</div>
              <div class="sp-team away">{{ m.teamB.name }}</div>
            </div>
            <div class="sp-live-sets">Set {{ m.score.a }}–{{ m.score.b }}</div>
          </div>
        </div>

        <!-- PROSSIMA PARTITA: solo quando NON c'è nulla live -->
        <div v-else class="sp-match">
          <div class="sp-match-head">
            <span class="sp-match-label">⚡ PROSSIMA PARTITA</span>
            <span class="sp-match-time">ORE {{ nextMatch.time }}</span>
          </div>
          <div class="sp-teams">
            <div class="sp-team home" v-html="nextMatch.home"></div>
            <div class="sp-vs">VS</div>
            <div class="sp-team away" v-html="nextMatch.away"></div>
          </div>
        </div>

        <!-- GRID -->
        <div class="sp-grid">
          <button v-for="(t, i) in visibleTiles" :key="i" class="sp-tile" :class="[('t' + (i % 8 + 1)), { 'is-sponsor': t.route === SPONSORS_ROUTE }]"
                  @click="onTile(t)">
            <div class="sp-tile-sheen"></div>
            <template v-if="t.route === SPONSORS_ROUTE && t.sponsor">
              <img v-if="t.sponsor.logo" class="sp-tile-sponsor-logo" :src="t.sponsor.logo" :alt="t.sponsor.name" />
              <span v-else class="sp-tile-label sp-tile-sponsor-name">{{ t.sponsor.name }}</span>
              <span class="sp-tile-sub">{{ t.sub }}</span>
            </template>
            <template v-else>
              <span class="sp-tile-icon">{{ tileIcon(t) }}</span>
              <span class="sp-tile-label">{{ t.label }}</span>
              <span class="sp-tile-sub">{{ t.sub }}</span>
            </template>
          </button>
        </div>

        <div class="sp-home"><span></span></div>
      </div>

      <!-- MODALE PREMI (aperta dalla 🏆 in alto a destra) -->
      <transition name="sp-modal">
        <div v-if="showPrizes" class="sp-prizes-scrim" @click.self="showPrizes = false">
          <div class="sp-prizes-modal">
            <button class="sp-prizes-x" @click="showPrizes = false" aria-label="Chiudi">✕</button>
            <h3 class="sp-prizes-title">🏆 PREMI</h3>
            <img v-if="prizesImage" :src="prizesImage" class="sp-prizes-img" alt="Premi del torneo" />
            <p v-else class="sp-prizes-empty">Premi in arrivo.</p>
          </div>
        </div>
      </transition>

    </div>
  </div>
</template>

<script setup>
// Layout tifoso "Sunset" — grafica alternativa, stessi dati/navigazione della
// home classica: il parent (TournamentHomeView) passa i dati reali e gestisce
// gli eventi (navigate = route della tile, live = scorciatoia al calendario).
import { computed, ref } from 'vue'
import { track as posthogTrack, EVENTS as PH_EVENTS } from '@/lib/track'

const emit = defineEmits(['live', 'signup', 'navigate'])

const props = defineProps({
  tournamentSlug: { type: String, default: '' },
  prizesImage: { type: String, default: '' },   // immagine verticale premi (modale 🏆)
  status:      { type: String, default: '' },
  brandTop:    { type: String, default: '' },
  brandBottom: { type: String, default: '' },
  logo:        { type: String, default: '' },   // immagine intestazione (al posto del brand testuale)
  subtitle:    { type: String, default: '' },
  date:        { type: String, default: '' },
  place:       { type: String, default: '' },
  liveLabel:   { type: String, default: 'LIVE ORA' },
  signupLabel: { type: String, default: 'INFO EVENTO' },
  nextMatch:   { type: Object, default: () => ({ time: '—', home: '', away: '' }) },
  liveMatches: { type: Array, default: () => [] },   // partite in corso: { id, court, teamA, teamB, score, cur, setLabel }
  // Dati reali dal torneo:
  tiles:       { type: Array, default: () => [] },   // { icon, label, sub, route }
  sponsors:    { type: Array, default: () => [] },    // { name, logo }
})

// Sponsor "della piattaforma": un solo Main sponsor, mostrato DENTRO la tile
// (niente più modale). La tile "Premi" (/prizes) diventa la tile Sponsor.
const showPrizes = ref(false)   // modale Premi (🏆 in alto a destra)
const mainSponsor = computed(() => props.sponsors.find(s => s.tier === 'main') || null)
const SPONSORS_ROUTE = '__sponsors__' // route sintetica: identifica la tile sponsor

// Tile: nel layout Sunset nascondiamo "Regolamento" (/rules) e "Info evento"
// (/event); la tile "Premi" (/prizes) mostra il Main sponsor.
const hiddenTileRoutes = ['/rules', '/event']
const visibleTiles = computed(() =>
  props.tiles
    .filter(t => !hiddenTileRoutes.includes(t.route))
    .map(t => t.route === '/prizes'
      ? {
          ...t,
          label: mainSponsor.value?.name || 'Sponsor',
          sub: 'sponsor della piattaforma',
          icon: 'sponsor',
          route: SPONSORS_ROUTE,
          sponsor: mainSponsor.value,
        }
      : t))

// Tap su una tile: la tile Sponsor apre l'eventuale link dello sponsor (niente
// modale); le altre navigano.
function onTile (t) {
  if (t.route === SPONSORS_ROUTE) {
    const s = mainSponsor.value
    if (s) {
      onSponsorClick(s)
      if (s.url) window.open(s.url, '_blank', 'noopener')
    }
    return
  }
  emit('navigate', t.route)
}

function onSponsorClick (s) {
  posthogTrack(PH_EVENTS.TOURNAMENT_SPONSOR_CLICKED, {
    tournament_slug: props.tournamentSlug,
    surface: 'tournament',
    sponsor_id: s.id,
    sponsor_name: s.name,
    tier: s.tier || 'partner',
    has_url: !!s.url,
  })
}

// Le icone della home classica sono chiavi (calendar, chart, …): qui le rendiamo
// come emoji per restare nello stile "pop" del layout Sunset.
const ICON_EMOJI = {
  calendar: '📅', chart: '📊', bracket: '🏆', star: '⭐',
  trophy: '🎁', gallery: '📸', doc: '📋', info: 'ℹ️', sponsor: '🤝'
}
const tileIcon = t => ICON_EMOJI[t.icon] || '▪️'

const sparks = [
  { top:'6%', left:'12%', size:'14px', color:'#fff',     delay:'.1s' },
  { top:'9%', left:'78%', size:'10px', color:'#C6FF3A',  delay:'.9s' },
  { top:'15%', left:'45%', size:'18px', color:'#fff',    delay:'.4s' },
  { top:'22%', left:'88%', size:'12px', color:'#00E5FF', delay:'1.3s' },
  { top:'33%', left:'63%', size:'16px', color:'#FF2E9A', delay:'1.6s' },
  { top:'44%', left:'82%', size:'11px', color:'#C6FF3A', delay:'1.8s' },
  { top:'62%', left:'90%', size:'12px', color:'#fff',    delay:'.3s' },
  { top:'74%', left:'8%',  size:'13px', color:'#fff',    delay:'.85s' },
]

const stars = [
  { top:'4%',  left:'20%', size:'9px',  delay:'0s' },
  { top:'2%',  left:'70%', size:'11px', delay:'.6s' },
  { top:'13%', left:'8%',  size:'10px', delay:'.3s' },
  { top:'16%', left:'58%', size:'7px',  delay:'2.1s' },
  { top:'20%', left:'30%', size:'9px',  delay:'.9s' },
]
</script>

<style scoped>
@import url('https://fonts.googleapis.com/css2?family=Fredoka:wght@600;700&family=Space+Grotesk:wght@500;600;700&display=swap');

/* In-app: niente cornice-telefono da mockup. La pagina riempie il device. */
.sp-stage {
  background: #0B0620;
  font-family: 'Space Grotesk', sans-serif;
}
.sp-phone {
  /* Scala unica ancorata al mockup iPhone 15 Pro Max (430 × 932 CSS px):
     --s = "1px del mockup" adattato al device. Prende il minore fra il rapporto
     in larghezza (limitato a 430) e in altezza, così il layout non trabocca mai
     e mantiene ESATTAMENTE le proporzioni del mockup — cambia solo la scala. */
  --vw: min(100dvw, 430px);
  --s: min(var(--vw) / 430, 100dvh / 932);
  width: 100%;
  max-width: 430px;
  margin: 0 auto;
  height: 100vh;
  height: 100dvh;
  background: #0B0620;
  overflow: hidden;
  position: relative;
  display: flex;
  flex-direction: column;
}

/* GLITTER */
.sp-glitter { position: absolute; inset: 0; pointer-events: none; overflow: hidden; z-index: 5; }
.sp-spark {
  position: absolute; color: #fff;
  text-shadow: 0 0 6px currentColor, 0 0 14px currentColor;
  animation: sp-twinkle 1.6s infinite ease-in-out;
}
@keyframes sp-twinkle { 0%,100%{opacity:0;transform:scale(.2) rotate(0)} 50%{opacity:1;transform:scale(1) rotate(25deg)} }

/* HERO */
.sp-hero {
  position: relative; flex: 0 0 auto; padding: calc(18*var(--s)) calc(20*var(--s)) calc(16*var(--s));
  background: linear-gradient(165deg,#1a0f3d 0%,#8A2BFF 18%,#FF2E9A 42%,#FF7A1A 65%,#C6FF3A 85%,#00E5FF 100%);
  background-size: 220% 220%; animation: sp-holo 9s ease infinite; overflow: hidden;
}
@keyframes sp-holo { 0%{background-position:0% 30%} 50%{background-position:100% 70%} 100%{background-position:0% 30%} }
.sp-sun {
  position: absolute; left: 50%; top: 36%; width: calc(180*var(--s)); height: calc(180*var(--s));
  transform: translate(-50%,-50%); border-radius: 50%;
  background: radial-gradient(circle at 35% 30%,#fff 0%,#FFE45E 18%,#FF9D2E 45%,#FF2E9A 72%,transparent 100%);
  filter: blur(.5px) drop-shadow(0 0 30px rgba(255,157,46,.6)); opacity: .85;
}
.sp-nightsky {
  position: absolute; inset: 0; z-index: 1;
  background: linear-gradient(180deg,rgba(6,3,20,.92) 0%,rgba(11,6,32,.55) 30%,rgba(11,6,32,.05) 55%,rgba(11,6,32,0) 75%);
}
.sp-star {
  position: absolute; color: #fff; z-index: 2;
  text-shadow: 0 0 4px #fff; animation: sp-startw 2.8s infinite ease-in-out;
}
@keyframes sp-startw { 0%,100%{opacity:.25;transform:scale(.8)} 50%{opacity:1;transform:scale(1.15)} }
.sp-horizon {
  position: absolute; left: 0; right: 0; bottom: 0; height: 44%; z-index: 2;
  background: linear-gradient(180deg,rgba(11,6,32,0) 0%,rgba(11,6,32,.55) 65%,rgba(11,6,32,.88) 100%);
}
.sp-wave {
  position: absolute; left: 0; right: 0; bottom: 0; height: calc(24*var(--s)); z-index: 3; opacity: .5;
  background: repeating-linear-gradient(90deg,rgba(255,255,255,.35) 0 16px,rgba(255,255,255,0) 16px 32px);
}
.sp-ball {
  position: absolute; top: calc(16*var(--s)); right: calc(18*var(--s)); width: calc(38*var(--s)); height: calc(38*var(--s)); border-radius: 50%; z-index: 6;
  background: radial-gradient(circle at 32% 28%,#fff,#ffe9c7 22%,#C6FF3A 55%,#00E5FF 100%);
  box-shadow: 0 0 14px rgba(255,255,255,.7), 0 3px 0 rgba(0,0,0,.25);
  border: none; padding: 0; cursor: pointer; display: grid; place-items: center;
  font-size: calc(19*var(--s)); line-height: 1; -webkit-tap-highlight-color: transparent;
  transition: transform .12s;
}
.sp-ball:active { transform: scale(.9); }
.sp-top { position: relative; z-index: 4; }
.sp-status {
  display: inline-flex; align-items: center; gap: calc(6*var(--s)); background: #0B0620; color: #C6FF3A;
  font-weight: 700; font-size: calc(12.5*var(--s)); letter-spacing: .05em; padding: calc(6*var(--s)) calc(11*var(--s)); border-radius: calc(20*var(--s));
  box-shadow: 0 0 12px rgba(198,255,58,.6);
}
.sp-dot { width: calc(7*var(--s)); height: calc(7*var(--s)); border-radius: 50%; background: #FF2E9A; box-shadow: 0 0 8px #FF2E9A; animation: sp-pulse 1.2s infinite; }
@keyframes sp-pulse { 0%,100%{opacity:1} 50%{opacity:.25} }

.sp-titlewrap { position: relative; z-index: 4; text-align: center; margin-top: calc(10*var(--s)); }
/* Con logo: l'immagine è l'intestazione — copre il 90% della larghezza, ma
   contenuta in altezza così che l'header resti ~30% dello schermo. */
.sp-titlewrap.has-logo { margin-top: calc(4*var(--s)); }
.sp-logo {
  display: block; margin: 0 auto;
  width: 100%; max-width: 100%;
  max-height: calc(180*var(--s)); object-fit: contain;   /* ~+20% rispetto a prima */
  filter: drop-shadow(0 3px 10px rgba(11,6,32,.55)) drop-shadow(0 0 18px rgba(255,255,255,.35));
}
.sp-brand {
  font-family: 'Fredoka', sans-serif; font-weight: 700; font-size: calc(40*var(--s)); line-height: .92; margin: 0;
  background: linear-gradient(180deg,#fff 0%,#dfffff 35%,#b9f7ff 55%,#fff 100%);
  -webkit-background-clip: text; background-clip: text; color: transparent;
  -webkit-text-stroke: 1.5px rgba(11,6,32,.9); letter-spacing: .5px;
  filter: drop-shadow(0 3px 0 rgba(11,6,32,.9)) drop-shadow(0 0 18px rgba(255,255,255,.55));
}
.sp-pad {
  display: block; margin-top: calc(2*var(--s));
  background: linear-gradient(180deg,#fff59d 0%,#ffd400 45%,#ff9d00 100%);
  -webkit-background-clip: text; background-clip: text; color: transparent;
  -webkit-text-stroke: 1.5px rgba(11,6,32,.9);
  filter: drop-shadow(0 3px 0 rgba(11,6,32,.9)) drop-shadow(0 0 18px rgba(255,157,0,.55));
}
.sp-meta { position: relative; z-index: 4; display: flex; justify-content: center; gap: calc(10*var(--s)); margin-top: calc(12*var(--s)); flex-wrap: wrap; }
.sp-chip {
  background: rgba(255,255,255,.92); border-radius: calc(14*var(--s)); padding: calc(8*var(--s)) calc(14*var(--s)); font-size: calc(13.5*var(--s));
  font-weight: 700; color: #0B0620; display: flex; align-items: center; gap: calc(6*var(--s)); box-shadow: 0 3px 0 rgba(0,0,0,.15);
}
.sp-cta { position: relative; z-index: 4; display: flex; gap: calc(10*var(--s)); margin-top: calc(12*var(--s)); }
.sp-btn {
  flex: 1; display: flex; align-items: center; justify-content: center; gap: calc(6*var(--s));
  padding: calc(13*var(--s)) calc(8*var(--s)); border: none; border-radius: calc(20*var(--s)); font-weight: 800;
  font-size: calc(15*var(--s)); letter-spacing: .03em; color: #fff; cursor: pointer; font-family: inherit; transition: transform .12s;
}
.sp-btn:active { transform: scale(.96); }
.sp-btn-live { background: linear-gradient(180deg,#ff5fb0,#FF2E9A); box-shadow: 0 5px 0 #9c1160, 0 8px 20px rgba(255,46,154,.5); }
.sp-btn-signup { background: linear-gradient(180deg,#5b3d8a,#2c1a4d); box-shadow: 0 5px 0 #1a0f30; color: rgba(255,255,255,.6); }

/* BODY */
.sp-body { flex: 1 1 auto; display: flex; flex-direction: column; min-height: 0; padding: 0 calc(16*var(--s)); }

.sp-match {
  margin-top: calc(10*var(--s)); background: linear-gradient(145deg,#1c0f3d,#100826); border-radius: calc(18*var(--s)); padding: calc(10*var(--s)) calc(16*var(--s));
  position: relative; border: 1.5px solid rgba(255,255,255,.15); box-shadow: 0 10px 30px rgba(138,43,255,.35);
}
.sp-match-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: calc(8*var(--s)); }
.sp-match-label {
  color: #0B0620; font-size: calc(11*var(--s)); font-weight: 700; letter-spacing: .14em;
  background: linear-gradient(90deg,#C6FF3A,#00E5FF); padding: calc(5*var(--s)) calc(11*var(--s)); border-radius: calc(20*var(--s)); white-space: nowrap;
}
.sp-match-time { color: #fff; font-size: calc(12*var(--s)); font-weight: 700; opacity: .55; }
.sp-teams { display: flex; align-items: center; justify-content: space-between; gap: calc(8*var(--s)); }
.sp-team {
  flex: 1; text-align: center; font-family: 'Fredoka', sans-serif; font-weight: 700; font-size: calc(17*var(--s)); line-height: 1.2;
  background: linear-gradient(180deg,#fff,#d9c8ff); -webkit-background-clip: text; background-clip: text; color: transparent;
}
.sp-team.away { background: linear-gradient(180deg,#fff,#9dfff0); -webkit-background-clip: text; background-clip: text; color: transparent; }
.sp-vs {
  background: radial-gradient(circle at 30% 30%,#fff,#FF7A1A 60%,#FF2E9A); color: #0B0620; font-family: 'Fredoka', sans-serif;
  font-weight: 700; font-size: calc(12*var(--s)); width: calc(34*var(--s)); height: calc(34*var(--s)); display: flex; align-items: center; justify-content: center;
  border-radius: 50%; box-shadow: 0 0 14px rgba(255,122,26,.7); flex-shrink: 0;
}

/* LIVE nella home: card in corso (scroll orizzontale se più campi insieme) */
.sp-live-row { display: flex; gap: calc(10*var(--s)); margin-top: calc(10*var(--s)); overflow-x: auto; scroll-snap-type: x mandatory; scrollbar-width: none; }
.sp-live-row::-webkit-scrollbar { display: none; }
.sp-live-row .sp-match { margin-top: 0; flex: 0 0 100%; scroll-snap-align: center; }
.sp-match-live {
  border-color: rgba(255,46,154,.6);
  box-shadow: 0 0 0 1px rgba(255,46,154,.35) inset, 0 10px 30px rgba(255,46,154,.3);
}
.sp-match-label.live {
  background: linear-gradient(90deg,#FF2E9A,#FF7A1A); color: #fff;
  display: inline-flex; align-items: center; gap: 6px;
}
.sp-live-dot { width: calc(7*var(--s)); height: calc(7*var(--s)); border-radius: 50%; background: #fff; animation: sp-livepulse 1.2s infinite ease-in-out; }
@keyframes sp-livepulse { 0%,100%{opacity:1;transform:scale(1)} 50%{opacity:.35;transform:scale(.7)} }
.sp-live-score {
  flex-shrink: 0; font-family: 'Fredoka', sans-serif; font-weight: 700; font-size: calc(26*var(--s)); line-height: 1;
  color: #fff; font-variant-numeric: tabular-nums; display: flex; align-items: baseline; gap: calc(4*var(--s));
  text-shadow: 0 0 14px rgba(255,46,154,.7);
}
.sp-live-score span { color: #FF2E9A; font-size: calc(20*var(--s)); }
.sp-live-sets { text-align: center; margin-top: calc(6*var(--s)); font-size: calc(10.5*var(--s)); font-weight: 700; letter-spacing: .1em; color: rgba(255,255,255,.6); }

/* La griglia riempie lo spazio residuo del body: le tile si allungano (righe 1fr)
   per non lasciare vuoto sotto. Contenuto centrato verticalmente nella tile. */
.sp-grid { display: grid; grid-template-columns: 1fr 1fr; gap: calc(6*var(--s)); margin-top: calc(8*var(--s)); flex: 1 1 auto; grid-auto-rows: 1fr; }
.sp-tile {
  border: none; border-radius: calc(16*var(--s)); padding: calc(9*var(--s)) calc(13*var(--s)); color: #0B0620; position: relative; overflow: hidden;
  cursor: pointer; text-align: center; font-family: inherit; transition: transform .12s;
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  box-shadow: 0 5px 0 rgba(0,0,0,.18), 0 10px 24px rgba(0,0,0,.28);
}
.sp-tile:active { transform: translateY(3px) scale(.98); }
.sp-tile-sheen { position: absolute; top: 0; left: 0; right: 0; height: 45%; background: linear-gradient(180deg,rgba(255,255,255,.55),rgba(255,255,255,0)); }
.sp-tile-icon { font-size: calc(27*var(--s)); display: block; position: relative; z-index: 2; }
.sp-tile-label { font-weight: 700; font-size: calc(17.5*var(--s)); display: block; position: relative; z-index: 2; margin-top: calc(5*var(--s)); }
.sp-tile-sub { font-size: calc(13*var(--s)); opacity: .7; font-weight: 600; display: block; position: relative; z-index: 2; }
/* Tile Sponsor: il logo del Main sponsor riempie l'area icona/etichetta */
.sp-tile.is-sponsor { background: #fff; color: #0B0620; padding: calc(6*var(--s)) calc(4*var(--s)); }
.sp-tile-sponsor-logo { max-width: 100%; max-height: calc(121*var(--s)); object-fit: contain; position: relative; z-index: 2; margin: calc(2*var(--s)) 0; }
.sp-tile-sponsor-name { font-size: calc(18*var(--s)); }
.t1 { background: linear-gradient(160deg,#6dfff0,#00E5FF); }
.t2 { background: linear-gradient(160deg,#ff8fce,#FF2E9A); color: #fff; }
.t3 { background: linear-gradient(160deg,#fff58a,#ffd400); }
.t4 { background: linear-gradient(160deg,#ffb46b,#FF7A1A); color: #fff; }
.t5 { background: linear-gradient(160deg,#f1ff8a,#C6FF3A); }
.t6 { background: linear-gradient(160deg,#c99bff,#8A2BFF); color: #fff; }
.t7 { background: linear-gradient(160deg,#6dfff0,#00E5FF); }
.t8 { background: linear-gradient(160deg,#ffb46b,#FF7A1A); color: #fff; }

.sp-home { display: flex; justify-content: center; padding: calc(4*var(--s)) 0 calc(6*var(--s)); margin-top: auto; }
.sp-home span { width: calc(120*var(--s)); height: calc(4*var(--s)); border-radius: calc(4*var(--s)); background: #fff; opacity: .25; }

/* MODALE SPONSOR */
.sp-modal-scrim {
  position: absolute; inset: 0; z-index: 20; display: flex; align-items: center; justify-content: center;
  padding: 20px; background: rgba(6,3,18,.72); backdrop-filter: blur(4px);
}
.sp-modal {
  position: relative; width: 100%; max-width: 360px; max-height: 82%; overflow-y: auto;
  background: linear-gradient(180deg,#1c0f3d,#100826); border: 1.5px solid rgba(255,255,255,.16);
  border-radius: 22px; padding: 22px 18px 20px; box-shadow: 0 20px 60px rgba(138,43,255,.4);
}
.sp-modal-x {
  position: absolute; top: 12px; right: 12px; width: 30px; height: 30px; border-radius: 50%;
  border: none; background: rgba(255,255,255,.12); color: #fff; font-size: 15px; font-weight: 700; cursor: pointer;
}
.sp-modal-title {
  margin: 0 0 16px; text-align: center; font-size: 15px; font-weight: 800; letter-spacing: .08em; color: #00E5FF;
}
.sp-modal-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
.sp-modal-sponsor {
  display: flex; align-items: center; justify-content: center; text-decoration: none;
  background: #fff; border-radius: 14px; min-height: 74px; padding: 10px; overflow: hidden;
  box-shadow: 0 6px 18px rgba(0,0,0,.3);
}
.sp-modal-sponsor.main { grid-column: 1 / -1; min-height: 92px; }
.sp-modal-sponsor img { max-width: 100%; max-height: 68px; object-fit: contain; }
.sp-modal-name { color: #0B0620; font-weight: 800; font-size: 14px; text-align: center; }
.sp-modal-empty { text-align: center; color: rgba(255,255,255,.6); font-size: 13px; padding: 20px 0; }

/* transizione modale */
.sp-modal-enter-active, .sp-modal-leave-active { transition: opacity .2s ease; }
.sp-modal-enter-active .sp-modal, .sp-modal-leave-active .sp-modal { transition: transform .25s cubic-bezier(.22,1.3,.5,1); }
.sp-modal-enter-from, .sp-modal-leave-to { opacity: 0; }
.sp-modal-enter-from .sp-modal, .sp-modal-leave-to .sp-modal { transform: translateY(24px) scale(.94); }

/* Modale Premi (immagine verticale caricata da admin) */
.sp-prizes-scrim {
  position: fixed; inset: 0; z-index: 40; background: rgba(6,3,18,.86);
  display: flex; align-items: center; justify-content: center; padding: calc(18*var(--s));
  backdrop-filter: blur(3px);
}
.sp-prizes-modal {
  position: relative; width: 100%; max-width: calc(360*var(--s));
  display: flex; flex-direction: column; align-items: center; gap: calc(10*var(--s));
}
.sp-prizes-x {
  position: absolute; top: calc(-6*var(--s)); right: 0; width: calc(34*var(--s)); height: calc(34*var(--s));
  border-radius: 50%; border: none; cursor: pointer; background: rgba(255,255,255,.16); color: #fff; font-size: calc(16*var(--s));
}
.sp-prizes-title {
  margin: 0; color: #fff; font-weight: 900; font-size: calc(16*var(--s)); letter-spacing: .06em;
}
.sp-prizes-img {
  width: 100%; max-height: 78dvh; object-fit: contain; border-radius: calc(14*var(--s));
  box-shadow: 0 18px 44px rgba(0,0,0,.5);
}
.sp-prizes-empty { color: rgba(255,255,255,.65); font-size: calc(14*var(--s)); padding: calc(30*var(--s)) 0; }
</style>
