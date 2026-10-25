<template>
  <span class="fan-match-card" :class="{ 'is-compact': compact }" role="img" :aria-label="accessibleLabel">
    <img class="match-card-art" :src="matchCardArtwork" alt="" aria-hidden="true" loading="lazy" decoding="async" />
    <span class="match-card-shade" aria-hidden="true"></span>
    <span class="match-card-club-light" aria-hidden="true"></span>
    <span class="match-card-heading">
      <span class="match-card-edition">LA TUA PARTITA</span>
      <span class="match-card-logo-plaque" aria-hidden="true"><img v-if="safeClubLogoUrl" :src="safeClubLogoUrl" alt="" loading="lazy" decoding="async" /><span v-else>{{ clubInitials }}</span></span>
    </span>
    <span class="match-card-copy">
      <span class="match-card-caption">RICORDO DAL PALAZZETTO</span>
      <strong class="match-card-club-name">{{ displayClubName }}</strong>
      <span v-if="opponent" class="match-card-opponent"><small>VS</small><strong>{{ opponent }}</strong></span>
      <span class="match-card-footer"><time v-if="eventDateLabel" :datetime="eventDateISO">{{ eventDateLabel }}</time><span v-else>LA TUA COLLEZIONE</span><span class="match-card-sport" aria-hidden="true"><svg viewBox="0 0 22 22" fill="none"><circle cx="11" cy="11" r="9"/><path v-if="isVolleyball" d="M11 11Q6 8 7 3M11 11Q16 8 19 11M11 11Q11 17 7 19M4 6Q3 12 8 15M16 3Q11 4 9 9M19 15Q14 18 10 15"/><path v-else d="M2 11h18M11 2v18M5 4q12 7 0 14M17 4q-12 7 0 14"/></svg></span></span>
    </span>
    <span class="match-card-sheen" aria-hidden="true"></span>
  </span>
</template>

<script setup>
import { computed } from 'vue';
import basketballCardArtwork from '../assets/fan-collectibles/match-card.webp';
import volleyballCardArtwork from '../assets/fan-collectibles/match-card-volleyball.webp';

const props = defineProps({
  memory: { type: Object, default: () => ({}) },
  clubName: { type: String, default: '' },
  clubLogoUrl: { type: String, default: '' },
  sport: { type: String, default: '' },
  compact: { type: Boolean, default: false },
});

function cleanText(value) { return typeof value === 'string' ? value.trim() : ''; }
const isVolleyball = computed(() => cleanText(props.sport).toLocaleLowerCase('it-IT') === 'pallavolo');
const matchCardArtwork = computed(() => isVolleyball.value ? volleyballCardArtwork : basketballCardArtwork);
const safeMemory = computed(() => props.memory && typeof props.memory === 'object' && !Array.isArray(props.memory) ? props.memory : {});
const displayClubName = computed(() => cleanText(safeMemory.value.club_name) || cleanText(props.clubName) || 'La tua squadra');
const opponent = computed(() => cleanText(safeMemory.value.opponent));
const safeClubLogoUrl = computed(() => cleanText(props.clubLogoUrl));
const clubInitials = computed(() => displayClubName.value.split(/\s+/).slice(0, 2).map((part) => Array.from(part)[0]).join('').toLocaleUpperCase('it-IT'));
const eventDate = computed(() => {
  const value = cleanText(safeMemory.value.event_date);
  if (!value) return null;
  const date = new Date(value);
  return Number.isFinite(date.getTime()) ? date : null;
});
const eventDateLabel = computed(() => eventDate.value ? eventDate.value.toLocaleDateString('it-IT', { day: '2-digit', month: 'short', year: 'numeric', timeZone: 'UTC' }) : '');
const eventDateISO = computed(() => eventDate.value ? eventDate.value.toISOString().slice(0, 10) : '');
const accessibleLabel = computed(() => ['Card ricordo della partita', displayClubName.value, opponent.value ? `contro ${opponent.value}` : '', eventDateLabel.value].filter(Boolean).join(', '));
</script>

<style scoped>
.fan-match-card { position: relative; isolation: isolate; container-type: inline-size; display: block; width: 100%; min-width: 0; aspect-ratio: 3 / 4; overflow: hidden; clip-path: polygon(7% 0,93% 0,100% 5%,100% 95%,93% 100%,7% 100%,0 95%,0 5%); background: #101820; color: #fff; font-family: inherit; text-align: left; box-shadow: 0 2px 0 #ffffff70 inset, 0 -2px 0 #0b101c inset; transform: translateZ(0); }
.fan-match-card, .fan-match-card * { box-sizing: border-box; }
.fan-match-card::before { content: ''; position: absolute; z-index: 6; inset: 2.2%; border: 1px solid #d3e0f04f; border-radius: 4% / 3%; pointer-events: none; }
.fan-match-card::after { content: ''; position: absolute; z-index: 5; top: 0; right: 5%; width: 25%; height: 1px; background: #fff; box-shadow: 0 0 12px #e8f3ff; opacity: .75; pointer-events: none; }
.match-card-art { position: absolute; z-index: -3; inset: 0; width: 100%; height: 100%; object-fit: cover; pointer-events: none; }
.match-card-shade { position: absolute; z-index: -2; inset: 0; background: linear-gradient(180deg, #070d16b8 0%, #08111b12 30%, #08111b00 37%, #07101bd6 62%, #060d18f2 100%); pointer-events: none; }
.match-card-club-light { position: absolute; z-index: -1; inset: 0; background: radial-gradient(ellipse at 0% 62%, color-mix(in srgb, var(--red, #bc243e) 42%, transparent), transparent 57%), radial-gradient(ellipse at 100% 38%, color-mix(in srgb, var(--gold, #4f78ac) 32%, transparent), transparent 51%); mix-blend-mode: screen; opacity: .8; pointer-events: none; }
.match-card-heading { position: absolute; inset: 7% 8% auto; display: flex; justify-content: space-between; align-items: flex-start; gap: 5%; }
.match-card-edition { max-width: 60%; padding-top: 3%; color: #f3f6fd; font-size: clamp(6px, 3.35cqi, 12px); font-weight: 900; letter-spacing: .16em; line-height: 1.45; text-shadow: 0 1px 7px #000; }
.match-card-logo-plaque { display: grid; place-items: center; width: 24%; aspect-ratio: 1; flex: none; border: 1px solid #dbe8f2a8; border-radius: 19%; padding: clamp(3px,1.4cqi,6px); background: linear-gradient(140deg, #f7fbff, #bdcad8 60%, #f8fcff); box-shadow: 0 3px 9px #0005, 0 0 0 1px #141f2d; color: #182333; font-size: clamp(8px, 6cqi, 22px); font-weight: 900; }
.match-card-logo-plaque img { width: 100%; height: 100%; object-fit: contain; filter: drop-shadow(0 1px 1px #0002); }
.match-card-copy { position: absolute; inset: auto 8% 7%; display: flex; flex-direction: column; min-width: 0; text-shadow: 0 2px 7px #0009; }
.match-card-caption { margin-bottom: 4%; color: #b9c8d9; font-size: clamp(5px, 2.75cqi, 10px); font-weight: 800; letter-spacing: .12em; line-height: 1.5; }
.match-card-club-name { display: -webkit-box; -webkit-box-orient: vertical; -webkit-line-clamp: 2; overflow: hidden; color: #fff; font-family: 'Barlow Condensed', Impact, 'Arial Narrow', sans-serif; font-size: clamp(16px, 10cqi, 36px); font-weight: 800; letter-spacing: .015em; line-height: 1.04; text-transform: uppercase; overflow-wrap: anywhere; }
.match-card-opponent { display: flex; align-items: baseline; gap: 5%; min-width: 0; margin-top: 5%; padding-top: 5%; border-top: 1px solid #d3e0f04f; }
.match-card-opponent small { flex: none; color: #99a9bc; font-size: clamp(6px, 3.2cqi, 12px); font-weight: 900; letter-spacing: .1em; }
.match-card-opponent strong { display: -webkit-box; -webkit-box-orient: vertical; -webkit-line-clamp: 2; overflow: hidden; min-width: 0; color: #f1f5fc; font-size: clamp(9px, 5.25cqi, 19px); font-weight: 850; line-height: 1.2; overflow-wrap: anywhere; }
.match-card-footer { display: flex; justify-content: space-between; align-items: center; gap: 6%; margin-top: 10%; color: #cbd7e5; font-size: clamp(6px, 3.2cqi, 12px); font-weight: 700; letter-spacing: .07em; line-height: 1.4; text-transform: uppercase; }
.match-card-sport { width: 10%; aspect-ratio: 1; flex: none; color: #b8cadd; }
.match-card-sport svg { width: 100%; height: 100%; stroke: currentColor; stroke-width: .9; }
.match-card-sheen { position: absolute; z-index: 7; inset: 0; overflow: hidden; pointer-events: none; }
.match-card-sheen::after { content: ''; position: absolute; inset: -30% -60%; background: linear-gradient(115deg, transparent 40%, #e7f5ff00 43%, #e7f5ff55 49%, #e7f5ff00 55%, transparent 59%); transform: translateX(-50%); opacity: 0; }
@media (hover: hover) { .fan-match-card:hover .match-card-sheen::after { animation: card-reflection .7s ease-out both; } }
:focus-visible .fan-match-card .match-card-sheen::after { animation: card-reflection .7s ease-out both; }
.is-compact { box-shadow: 0 2px 0 #ffffff70 inset, 0 4px 10px #0c172526; }
@keyframes card-reflection { 0% { transform: translateX(-50%); opacity: 0; } 15% { opacity: .8; } 100% { transform: translateX(50%); opacity: 0; } }
@media (prefers-reduced-motion: reduce) { .match-card-sheen { display: none; } }
</style>
