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
        <div class="sp-ball"></div>

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

        <div class="sp-cta">
          <button class="sp-btn sp-btn-live" @click="$emit('live')">🔥 {{ liveLabel }}</button>
          <button class="sp-btn sp-btn-signup" @click="$emit('signup')">{{ signupLabel }}</button>
        </div>
      </div>

      <!-- BODY -->
      <div class="sp-body">
        <!-- MATCH CARD -->
        <div class="sp-match">
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
          <button v-for="(t, i) in tiles" :key="i" class="sp-tile" :class="'t' + (i % 8 + 1)"
                  @click="$emit('navigate', t.route)">
            <div class="sp-tile-sheen"></div>
            <span class="sp-tile-icon">{{ tileIcon(t) }}</span>
            <span class="sp-tile-label">{{ t.label }}</span>
            <span class="sp-tile-sub">{{ t.sub }}</span>
          </button>
        </div>

        <!-- SPONSORS -->
        <div class="sp-sponsors-wrap" v-if="sponsorLogos().length">
          <div class="sp-sponsor-title">CON IL SUPPORTO DI</div>
          <div class="sp-sponsors">
            <div v-for="(s, i) in sponsorLogos()" :key="i" class="sp-sponsor-box">
              <img :src="s.logo" :alt="s.name" />
            </div>
          </div>
        </div>

        <div class="sp-home"><span></span></div>
      </div>
    </div>
  </div>
</template>

<script setup>
// Layout tifoso "Sunset" — grafica alternativa, stessi dati/navigazione della
// home classica: il parent (TournamentHomeView) passa i dati reali e gestisce
// gli eventi (navigate = route della tile, live/signup = scorciatoie).
defineEmits(['live', 'signup', 'navigate'])

const props = defineProps({
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
  // Dati reali dal torneo:
  tiles:       { type: Array, default: () => [] },   // { icon, label, sub, route }
  sponsors:    { type: Array, default: () => [] },    // { name, logo }
})

// La griglia mostra i loghi degli sponsor (max 4 per non strabordare); se nessuno
// ha un logo, la riga non compare.
const sponsorLogos = () => props.sponsors.filter(s => s.logo).slice(0, 4)

// Le icone della home classica sono chiavi (calendar, chart, …): qui le rendiamo
// come emoji per restare nello stile "pop" del layout Sunset.
const ICON_EMOJI = {
  calendar: '📅', chart: '📊', bracket: '🏆', star: '⭐',
  trophy: '🎁', gallery: '📸', doc: '📋', info: 'ℹ️'
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
  position: relative; flex: 0 0 auto; padding: 18px 20px 16px;
  background: linear-gradient(165deg,#1a0f3d 0%,#8A2BFF 18%,#FF2E9A 42%,#FF7A1A 65%,#C6FF3A 85%,#00E5FF 100%);
  background-size: 220% 220%; animation: sp-holo 9s ease infinite; overflow: hidden;
}
@keyframes sp-holo { 0%{background-position:0% 30%} 50%{background-position:100% 70%} 100%{background-position:0% 30%} }
.sp-sun {
  position: absolute; left: 50%; top: 36%; width: 180px; height: 180px;
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
  position: absolute; left: 0; right: 0; bottom: 0; height: 24px; z-index: 3; opacity: .5;
  background: repeating-linear-gradient(90deg,rgba(255,255,255,.35) 0 16px,rgba(255,255,255,0) 16px 32px);
}
.sp-ball {
  position: absolute; top: 16px; right: 18px; width: 32px; height: 32px; border-radius: 50%; z-index: 6;
  background: radial-gradient(circle at 32% 28%,#fff,#ffe9c7 22%,#C6FF3A 55%,#00E5FF 100%);
  box-shadow: 0 0 14px rgba(255,255,255,.7), 0 3px 0 rgba(0,0,0,.25);
}
.sp-top { position: relative; z-index: 4; }
.sp-status {
  display: inline-flex; align-items: center; gap: 6px; background: #0B0620; color: #C6FF3A;
  font-weight: 700; font-size: 11px; letter-spacing: .05em; padding: 6px 10px; border-radius: 20px;
  box-shadow: 0 0 12px rgba(198,255,58,.6);
}
.sp-dot { width: 7px; height: 7px; border-radius: 50%; background: #FF2E9A; box-shadow: 0 0 8px #FF2E9A; animation: sp-pulse 1.2s infinite; }
@keyframes sp-pulse { 0%,100%{opacity:1} 50%{opacity:.25} }

.sp-titlewrap { position: relative; z-index: 4; text-align: center; margin-top: 10px; }
/* Con logo: l'immagine è l'intestazione — copre il 90% della larghezza e si
   estende in verticale fin sopra i box data/luogo. */
.sp-titlewrap.has-logo { margin-top: 6px; }
.sp-logo {
  display: block; margin: 0 auto;
  width: 90%; max-width: 90%;
  max-height: clamp(135px, 27dvh, 252px); object-fit: contain;
  filter: drop-shadow(0 3px 10px rgba(11,6,32,.55)) drop-shadow(0 0 18px rgba(255,255,255,.35));
}
.sp-brand {
  font-family: 'Fredoka', sans-serif; font-weight: 700; font-size: 40px; line-height: .92; margin: 0;
  background: linear-gradient(180deg,#fff 0%,#dfffff 35%,#b9f7ff 55%,#fff 100%);
  -webkit-background-clip: text; background-clip: text; color: transparent;
  -webkit-text-stroke: 1.5px rgba(11,6,32,.9); letter-spacing: .5px;
  filter: drop-shadow(0 3px 0 rgba(11,6,32,.9)) drop-shadow(0 0 18px rgba(255,255,255,.55));
}
.sp-pad {
  display: block; margin-top: 2px;
  background: linear-gradient(180deg,#fff59d 0%,#ffd400 45%,#ff9d00 100%);
  -webkit-background-clip: text; background-clip: text; color: transparent;
  -webkit-text-stroke: 1.5px rgba(11,6,32,.9);
  filter: drop-shadow(0 3px 0 rgba(11,6,32,.9)) drop-shadow(0 0 18px rgba(255,157,0,.55));
}
.sp-meta { position: relative; z-index: 4; display: flex; justify-content: center; gap: 10px; margin-top: 12px; flex-wrap: wrap; }
.sp-chip {
  background: rgba(255,255,255,.92); border-radius: 14px; padding: 7px 13px; font-size: 12.5px;
  font-weight: 700; color: #0B0620; display: flex; align-items: center; gap: 6px; box-shadow: 0 3px 0 rgba(0,0,0,.15);
}
.sp-cta { position: relative; z-index: 4; display: flex; gap: 10px; margin-top: 12px; }
.sp-btn {
  flex: 1; text-align: center; padding: 12px 8px; border: none; border-radius: 20px; font-weight: 700;
  font-size: 13px; letter-spacing: .03em; color: #fff; cursor: pointer; font-family: inherit; transition: transform .12s;
}
.sp-btn:active { transform: scale(.96); }
.sp-btn-live { background: linear-gradient(180deg,#ff5fb0,#FF2E9A); box-shadow: 0 5px 0 #9c1160, 0 8px 20px rgba(255,46,154,.5); }
.sp-btn-signup { background: linear-gradient(180deg,#5b3d8a,#2c1a4d); box-shadow: 0 5px 0 #1a0f30; color: rgba(255,255,255,.6); }

/* BODY */
.sp-body { flex: 1 1 auto; display: flex; flex-direction: column; min-height: 0; padding: 0 16px; }

.sp-match {
  margin-top: 10px; background: linear-gradient(145deg,#1c0f3d,#100826); border-radius: 18px; padding: 10px 16px;
  position: relative; border: 1.5px solid rgba(255,255,255,.15); box-shadow: 0 10px 30px rgba(138,43,255,.35);
}
.sp-match-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
.sp-match-label {
  color: #0B0620; font-size: 10px; font-weight: 700; letter-spacing: .14em;
  background: linear-gradient(90deg,#C6FF3A,#00E5FF); padding: 5px 11px; border-radius: 20px; white-space: nowrap;
}
.sp-match-time { color: #fff; font-size: 11px; font-weight: 700; opacity: .55; }
.sp-teams { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
.sp-team {
  flex: 1; text-align: center; font-family: 'Fredoka', sans-serif; font-weight: 700; font-size: 15px; line-height: 1.2;
  background: linear-gradient(180deg,#fff,#d9c8ff); -webkit-background-clip: text; background-clip: text; color: transparent;
}
.sp-team.away { background: linear-gradient(180deg,#fff,#9dfff0); -webkit-background-clip: text; background-clip: text; color: transparent; }
.sp-vs {
  background: radial-gradient(circle at 30% 30%,#fff,#FF7A1A 60%,#FF2E9A); color: #0B0620; font-family: 'Fredoka', sans-serif;
  font-weight: 700; font-size: 12px; width: 34px; height: 34px; display: flex; align-items: center; justify-content: center;
  border-radius: 50%; box-shadow: 0 0 14px rgba(255,122,26,.7); flex-shrink: 0;
}

.sp-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 6px; margin-top: 8px; }
.sp-tile {
  border: none; border-radius: 16px; padding: 7px 12px; color: #0B0620; position: relative; overflow: hidden;
  cursor: pointer; text-align: left; font-family: inherit; transition: transform .12s;
  box-shadow: 0 5px 0 rgba(0,0,0,.18), 0 10px 24px rgba(0,0,0,.28);
}
.sp-tile:active { transform: translateY(3px) scale(.98); }
.sp-tile-sheen { position: absolute; top: 0; left: 0; right: 0; height: 45%; background: linear-gradient(180deg,rgba(255,255,255,.55),rgba(255,255,255,0)); }
.sp-tile-icon { font-size: 19px; display: block; position: relative; z-index: 2; }
.sp-tile-label { font-weight: 700; font-size: 13.5px; display: block; position: relative; z-index: 2; margin-top: 4px; }
.sp-tile-sub { font-size: 10.5px; opacity: .7; font-weight: 600; display: block; position: relative; z-index: 2; }
.t1 { background: linear-gradient(160deg,#6dfff0,#00E5FF); }
.t2 { background: linear-gradient(160deg,#ff8fce,#FF2E9A); color: #fff; }
.t3 { background: linear-gradient(160deg,#fff58a,#ffd400); }
.t4 { background: linear-gradient(160deg,#ffb46b,#FF7A1A); color: #fff; }
.t5 { background: linear-gradient(160deg,#f1ff8a,#C6FF3A); }
.t6 { background: linear-gradient(160deg,#c99bff,#8A2BFF); color: #fff; }
.t7 { background: linear-gradient(160deg,#6dfff0,#00E5FF); }
.t8 { background: linear-gradient(160deg,#ffb46b,#FF7A1A); color: #fff; }

.sp-sponsors-wrap { margin-top: auto; }
.sp-sponsor-title { text-align: center; font-size: 10px; letter-spacing: .18em; font-weight: 700; color: #fff; opacity: .4; margin: 10px 0 6px; }
.sp-sponsors { display: flex; gap: 10px; padding-bottom: 8px; }
.sp-sponsor-box {
  flex: 1; background: #fff; border-radius: 12px; display: flex; align-items: center; justify-content: center;
  height: 42px; padding: 6px; box-shadow: 0 4px 14px rgba(0,0,0,.25); overflow: hidden;
}
.sp-sponsor-box img { max-width: 100%; max-height: 100%; object-fit: contain; }
.sp-home { display: flex; justify-content: center; padding: 4px 0 6px; }
.sp-home span { width: 120px; height: 4px; border-radius: 4px; background: #fff; opacity: .25; }
</style>
