<template>
  <section class="crossover" :class="{ 'is-won': celebration }" aria-label="Crossover Rush">
    <header class="cr-header"><div class="cr-brand"><span>{{ clubName || 'COURTSIDE' }}</span><b>CROSSOVER <em>RUSH</em></b><small>BATTI IL DIFENSORE</small></div><div class="cr-wallet" aria-label="Saldo monete"><i class="cr-coin" aria-hidden="true"></i>{{ walletCoins }}</div><button type="button" class="cr-sound" :aria-pressed="soundEnabled" :aria-label="soundEnabled ? 'Disattiva audio' : 'Attiva audio'" @click="toggleSound"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true"><path d="M11 5 6 9H3v6h3l5 4Z"/><path v-if="soundEnabled" d="M15 8c3 2 3 6 0 8m3-11c5 4 5 10 0 14"/><path v-else d="m16 9 6 6m0-6-6 6"/></svg></button></header>
    <div class="cr-round"><span>DUELLO <b>{{ Math.min(3, duelIndex + 1) }}</b><small>/ 3</small></span><div class="cr-round-dots" aria-hidden="true"><i v-for="i in 3" :key="i" :class="{ completed: i <= duelIndex, current: i === duelIndex + 1 }"></i></div><b class="cr-time" :class="{ urgent: remainingMs <= 5000 && session }">{{ secondsLeft }}<small>s</small></b></div>
    <div class="cr-arena">
      <div class="cr-stats" aria-live="off"><span><small>VITTORIE</small><b>{{ wins }}</b></span><span><small>DI FILA</small><b>{{ streak }}</b></span><span><small>PUNTI TECNICA</small><b>{{ qualityTotal }}</b></span></div>
      <svg ref="courtEl" class="cr-court" viewBox="0 0 360 560" preserveAspectRatio="xMidYMid meet" tabindex="0" role="application" aria-label="Nella parte bassa del campo trascina a sinistra o destra per fintare, mantieni un istante e cambia rapidamente lato. Con la tastiera usa le frecce sinistra e destra."
        @pointerdown="onPointerDown" @pointermove="onPointerMove" @pointerup="onPointerUp" @pointercancel="cancelControl" @lostpointercapture="onLostCapture" @keydown="onCourtKeydown">
        <defs>
          <radialGradient id="cr-night"><stop stop-color="#26323e"/><stop offset="1" stop-color="#09121e"/></radialGradient>
          <linearGradient id="cr-floor" x2="0" y2="1"><stop stop-color="#4d3d32"/><stop offset=".6" stop-color="#916543"/><stop offset="1" stop-color="#b27c4c"/></linearGradient>
          <linearGradient id="cr-kit" x2="1" y2="1"><stop stop-color="var(--red, #45b7a1)"/><stop offset="1" stop-color="var(--red-deep, #1a6d66)"/></linearGradient>
          <linearGradient id="cr-enemy" x2="1" y2="1"><stop stop-color="#ef8e78"/><stop offset="1" stop-color="#a8404c"/></linearGradient>
          <linearGradient id="cr-skin" x2="1" y2="1"><stop stop-color="#e3b790"/><stop offset="1" stop-color="#95634e"/></linearGradient>
          <radialGradient id="cr-ball" cx=".3" cy=".25" r=".8"><stop stop-color="#ffe2a0"/><stop offset=".35" stop-color="#f6a54d"/><stop offset=".8" stop-color="#b9511d"/><stop offset="1" stop-color="#632a17"/></radialGradient>
          <linearGradient id="cr-glass" x2="1" y2="1"><stop stop-color="#daf9ff" stop-opacity=".28"/><stop offset="1" stop-color="#9ddef7" stop-opacity=".04"/></linearGradient>
          <pattern id="cr-wood" width="30" height="90" patternUnits="userSpaceOnUse"><path d="M0 0v90M0 45h30M8 0q-4 23 0 45t0 45m14-90q4 23 0 45t0 45" fill="none" stroke="#edc69d" stroke-opacity=".12"/><path d="M30 0v90" stroke="#4a3024" stroke-opacity=".2"/></pattern>
          <g id="cr-attacker"><path d="M-20-7-33 26l10 5 15-26m28-12 13 33-10 5L8 5" fill="url(#cr-skin)"/><path d="M-17-18-22 28h44l-5-46-10-5H-7Z" fill="url(#cr-kit)" stroke="#ffffff40"/><path d="M-22 28-23 52h17l6-17 6 17h17l-1-24Z" fill="url(#cr-kit)"/><path d="m-21 52 2 17h11l2-17m12 0 2 17h11l2-17" fill="url(#cr-skin)"/><path d="M-19 67q-7 0-8 9H-7v-9m14 0v9h20q-1-9-8-9" fill="#e5f2f8"/><circle cy="-38" r="13" fill="url(#cr-skin)"/><path d="M-13-37q-2-18 13-17 15-1 13 17l-4 7H-9Z" fill="#282833"/><path d="m-14-14 6 4m22-4-6 4" stroke="#ffffff80" stroke-width="3"/><text y="11" text-anchor="middle" fill="var(--on-primary, #fff)" font-size="19" font-weight="900">1</text></g>
          <g id="cr-defender"><path d="M-16-8-40 12l-16-5-3 8 23 8 27-17m25-14 24 20 16-5 3 8-23 8L9 6" fill="url(#cr-skin)"/><path d="M-16-13-19 24h38l-3-37-10-5H-6Z" fill="url(#cr-enemy)" stroke="#ffe4df50"/><path d="M-19 24-26 48h15L0 34l11 14h15l-7-24Z" fill="url(#cr-enemy)"/><path d="m-24 48-4 13 9 3 8-16m22 0 8 16 9-3-4-13" fill="url(#cr-skin)"/><path d="m-28 59-9 8 15 4 5-9m34 0 5 9 15-4-9-8" fill="#d8e8ed"/><circle cy="-31" r="12" fill="url(#cr-skin)"/><path d="M-12-34q0-16 13-13 13 0 11 15l-7-7-13 2Z" fill="#24232d"/><text y="12" text-anchor="middle" fill="#ffe7e1" font-size="15" font-weight="900">{{ duelIndex + 4 }}</text></g>
        </defs>
        <rect width="360" height="560" fill="url(#cr-night)"/>
        <path d="M-40 0h100l130 270H-40Zm440 0H300L170 270h230Z" fill="#a9d7ee" opacity=".04"/>
        <g fill="#9dbbd1" opacity=".2"><circle v-for="(dot, i) in crowd" :key="i" :cx="dot.x" :cy="dot.y" :r="dot.r"/></g>
        <g :transform="`translate(${-pose.player.lean * .2}, 0)`">
          <path d="M40 149h280l106 411H-66Z" fill="url(#cr-floor)"/><path d="M40 149h280l106 411H-66Z" fill="url(#cr-wood)"/>
          <g fill="none" stroke="#ffe6bd" stroke-width="1.5" opacity=".45"><path d="M47 159h266l89 373H-42ZM126 159 101 295q79 33 158 0l-25-136M110 252q70 29 140 0M12 289q168 253 336 0"/><ellipse cx="180" cy="526" rx="60" ry="16"/></g>
          <path d="M126 159 101 295q79 33 158 0l-25-136Z" fill="var(--red, #3fa78f)" opacity=".1"/>
          <image v-if="clubLogoUrl" :href="clubLogoUrl" x="160" y="335" width="40" height="40" preserveAspectRatio="xMidYMid meet" opacity=".2"/>
          <text x="180" y="399" text-anchor="middle" fill="#fce2b7" opacity=".13" font-size="13" font-weight="900" letter-spacing="4">CROSSOVER</text>
          <g v-if="state === 'animating' && outcome?.won" class="cr-speed" fill="none" stroke="#ffe4b0" stroke-opacity=".25" stroke-width="2"><path d="M17 390-25 545m368-155 42 155M48 428 14 556m298-128 34 128"/></g>
        </g>
        <g transform="translate(180 112)"><path d="M-4-110h8v57H-4Z" fill="#304453"/><rect x="-52" y="-61" width="104" height="65" rx="5" fill="url(#cr-glass)" stroke="#d2eef5" stroke-opacity=".6" stroke-width="2"/><rect x="-19" y="-30" width="38" height="26" fill="none" stroke="#e9f8ff" stroke-width="2" stroke-opacity=".8"/><path d="M-55 4H55" stroke="var(--red, #4cbd9f)" stroke-width="4"/><path d="M-25 0q25-13 50 0" fill="none" stroke="#ffb35d" stroke-width="4"/></g>
        <g :transform="`translate(${pose.defender.x}, ${pose.defender.y}) scale(${pose.defender.scale}) rotate(${pose.defender.lean})`"><ellipse cy="68" rx="33" ry="10" fill="#000" opacity=".26"/><use href="#cr-defender"/></g>
        <g :transform="`translate(${pose.player.x}, ${pose.player.y}) scale(${pose.player.scale}) rotate(${pose.player.lean})`"><ellipse cy="77" rx="29" ry="9" fill="#000" opacity=".25"/><use href="#cr-attacker"/><path v-if="pose.phase === 'dunk'" d="m-17-11-6-34m40 34 6-34" stroke="url(#cr-skin)" stroke-width="9" stroke-linecap="round"/></g>
        <g v-if="pose.ball.visible" :transform="`translate(${pose.ball.x}, ${pose.ball.y}) scale(${pose.ball.scale}) rotate(${pose.ball.rotation})`" aria-hidden="true"><circle r="24" fill="url(#cr-ball)"/><g fill="none" stroke="#542a17" stroke-width="1.6"><path d="M-24 0q24 8 48 0M0-24q-8 24 0 48M-20-14q25 8 0 32M20-14q-25 8 0 32"/><circle r="23.5" stroke-opacity=".35"/></g><path d="M-13-14q6-6 12-6" fill="none" stroke="#fff0c8" stroke-width="2.5" stroke-linecap="round" opacity=".5"/></g>
        <g transform="translate(180 112)" aria-hidden="true"><g class="cr-net" :class="{ 'net-hit': pose.netHit }" fill="none" stroke="#effbff" stroke-width="1" stroke-opacity=".7"><path d="M-24 1-16 37q16 8 32 0L24 1M-16 5-10 39M-8 7-4 40M0 7v34M8 7 4 40M16 5 10 39M-22 12q22 11 44 0M-19 25q19 10 38 0"/></g><path d="M-25 0q25 14 50 0" fill="none" stroke="#f8873d" stroke-width="4" stroke-linecap="round"/></g>
        <g v-if="state === 'ready' || state === 'controlling'" class="cr-control-area" aria-hidden="true"><rect x="38" y="507" width="284" height="35" rx="17.5" fill="#0d2131" fill-opacity=".8" stroke="#e1f6ff" stroke-opacity=".16"/><path d="m68 521-5 5 5 5m224-10 5 5-5 5M119 526h122" fill="none" stroke="#b6d1de" stroke-width="1.5" stroke-opacity=".6"/><circle :cx="180 + controlX * 104" cy="525" r="13" fill="#ebd3a4"/><path :d="`M${174 + controlX * 104} 525h12`" stroke="#74603d" stroke-width="2" stroke-linecap="round"/></g>
        <g v-if="celebration" class="cr-coin-burst" aria-hidden="true"><text x="180" y="199" text-anchor="middle" fill="#ffe0a0" font-size="34" font-weight="900">+1</text><circle cx="211" cy="187" r="10" fill="#ecb849" stroke="#ffe6ae" stroke-width="2"/></g>
      </svg>
      <div v-if="state === 'controlling'" class="cr-gesture-hint" aria-hidden="true"><b>{{ feintReady ? 'CAMBIA LATO!' : 'FINTA DA UN LATO' }}</b><span>{{ Math.max(0, Math.ceil((6000 - duelElapsedMs) / 1000)) }}s per superarlo</span></div>
      <div v-if="celebration" class="cr-celebration" aria-hidden="true"><b>{{ lastReceipt?.clean ? 'CROSSOVER PERFETTO!' : 'DIFENSORE BATTUTO!' }}</b><span>+{{ lastReceipt?.quality }} punti tecnica · 1 moneta salvata</span></div>
      <div v-if="state === 'intro' || state === 'starting'" class="cr-cover"><span class="cr-eyebrow">CREA LO SPAZIO. ATTACCA IL FERRO.</span><div class="cr-cover-icon" aria-hidden="true">↔</div><h2>Una finta.<br>La strada è tua.</h2><p>Trascina da un lato, mantieni un istante,<br>poi cambia rapidamente direzione.</p><button type="button" class="cr-primary" :disabled="state === 'starting'" @click="startSession">{{ state === 'starting' ? 'Prepariamo il duello…' : 'Entra in campo · 3 duelli' }}<span aria-hidden="true">→</span></button><small>25 secondi · 1 moneta per duello vinto</small></div>
      <div v-if="state === 'finished'" class="cr-cover cr-summary"><span class="cr-eyebrow">LA TUA FIRMA SUL PARQUET</span><h2>{{ newRecord ? 'Nuovo record!' : wins === 3 ? 'Imprendibile.' : 'Ogni finta conta.' }}</h2><div class="cr-summary-stats"><span><b>{{ wins }}<small>/3</small></b><small>DUELLI VINTI</small></span><span><b>{{ qualityTotal }}</b><small>PUNTI TECNICA</small></span><span><b class="cr-gold">{{ wins }}<i class="cr-coin" aria-hidden="true"></i></b><small>MONETE SALVATE</small></span></div><p>{{ errorMessage || `Serie migliore: ${bestStreak} · Record personale: ${personalBest} punti` }}</p><button type="button" class="cr-primary" @click="startSession">Gioca ancora<span aria-hidden="true">↻</span></button><small>I punti tecnica misurano la finta. Ogni vittoria vale 1 moneta.</small></div>
    </div>
    <footer class="cr-footer"><div role="status" aria-live="polite"><b>{{ statusTitle }}</b><span>{{ statusDetail }}</span></div><button v-if="state === 'error'" type="button" class="cr-retry" @click="retry">{{ pendingDuel ? 'Salva il duello' : 'Riprova' }}</button><span v-else class="cr-match"><i class="cr-coin" aria-hidden="true"></i>{{ matchEarned }}<small>nella partita</small></span></footer>
  </section>
</template>

<script setup>
import { computed, onBeforeUnmount, ref } from 'vue';
import { startCrossoverSession, submitCrossoverDuel } from '../../api';
import { CROSSOVER_WORLD, CROSSOVER_ANIMATION_MS, evaluateCrossoverDuel, getCrossoverScene, sampleCrossoverFinish } from '../../games/crossover';

const props = defineProps({ eventId: { type: Number, required: true }, walletCoins: { type: Number, default: 0 }, earnedCoins: { type: Number, default: 0 }, clubName: { type: String, default: '' }, clubLogoUrl: { type: String, default: '' }, organizationSlug: { type: String, default: '' } });
const emit = defineEmits(['credited', 'saving-change']);
const gameEventId = props.eventId;
const state = ref('intro');
const courtEl = ref(null);
const session = ref(null);
const elapsedMs = ref(0);
const remainingMs = ref(25000);
const duelIndex = ref(0);
const samples = ref([]);
const controlX = ref(0);
const duelElapsedMs = ref(0);
const outcome = ref(null);
const pendingDuel = ref(null);
const animationProgress = ref(0);
const releasePose = ref(null);
const wins = ref(0);
const streak = ref(0);
const bestStreak = ref(0);
const qualityTotal = ref(0);
const confirmedEarned = ref(0);
const personalBest = ref(readBest());
const newRecord = ref(false);
const celebration = ref(false);
const lastReceipt = ref(null);
const errorMessage = ref('');
const hint = ref('');
const soundEnabled = ref(false);
const crowd = Array.from({ length: 65 }, (_, i) => ({ x: i % 15 * 27 - 8, y: 64 + Math.floor(i / 15) * 21, r: 1 + i % 3 * .4 }));
let clockStartedAt = 0;
let clockElapsedBase = 0;
let duelStartedAt = 0;
let duelStartMs = 0;
let animationStartedAt = 0;
let pointerId = null;
let pointerStart = null;
let frameId;
let resultTimer;
let disposed = false;
let receiptBusy = false;
let audioContext;

const secondsLeft = computed(() => Math.max(0, Math.ceil(remainingMs.value / 1000)));
const matchEarned = computed(() => Math.max(Number(props.earnedCoins) || 0, confirmedEarned.value));
const controlScene = computed(() => getCrossoverScene({ duelIndex: duelIndex.value, samples: samples.value, elapsedMs: duelElapsedMs.value }));
const feintReady = computed(() => controlScene.value.feintReady === true);
const pose = computed(() => {
  if (outcome.value && ['animating', 'saving', 'result', 'error'].includes(state.value)) {
    const sampled = sampleCrossoverFinish(outcome.value, animationProgress.value);
    if (releasePose.value && animationProgress.value < .2) {
      const first = sampleCrossoverFinish(outcome.value, 0);
      const weight = 1 - animationProgress.value / .2;
      for (const part of ['player', 'defender', 'ball']) {
        for (const key of ['x', 'y', 'scale', 'lean', 'rotation']) {
          if (Number.isFinite(sampled[part][key]) && Number.isFinite(releasePose.value[part][key])) {
            let delta = releasePose.value[part][key] - first[part][key];
            if (key === 'rotation') delta = ((delta % 360) + 540) % 360 - 180;
            sampled[part][key] += delta * weight;
          }
        }
      }
    }
    return sampled;
  }
  const x = controlX.value;
  const defenderX = controlScene.value.defenderX || 0;
  return { player: { x: 180 + x * 80, y: 414, scale: 1, lean: x * 8 }, defender: { x: 180 + defenderX * 80, y: 286, scale: .9, lean: -defenderX * 6 }, ball: { x: 180 + x * 80 + 28 * Math.sign(x || 1), y: 453 - Math.abs(Math.sin(elapsedMs.value / 145)) * 27, scale: .8, rotation: elapsedMs.value / 5, visible: true }, netHit: false, phase: 'dribble' };
});
const statusTitle = computed(() => state.value === 'error' ? pendingDuel.value ? 'Il duello resta da verificare' : 'Rientra in campo' : state.value === 'saving' ? 'Verifichiamo il duello…' : state.value === 'animating' ? outcome.value?.won ? 'Hai aperto la difesa!' : 'Il difensore ti ha letto' : celebration.value ? 'Una moneta salvata!' : state.value === 'controlling' ? 'Finta, poi cambia mano' : state.value === 'finished' ? 'Le monete confermate sono salvate' : 'Usa il pollice nella parte bassa');
const statusDetail = computed(() => errorMessage.value || hint.value || (state.value === 'saving' ? 'Attendi la conferma del risultato.' : 'Sinistra, un istante di finta, poi rapidamente a destra. O viceversa.'));

function clockNow() { return window.performance?.now?.() ?? Date.now(); }
function recordKey() { return `cento:crossover:best:${encodeURIComponent(props.organizationSlug)}:${gameEventId}`; }
function readBest() { try { const value = Number(window.localStorage.getItem(recordKey())); return Number.isInteger(value) && value >= 0 && value <= 300 ? value : 0; } catch { return 0; } }
function updateBest() {
  if (qualityTotal.value <= personalBest.value) return;
  personalBest.value = qualityTotal.value; newRecord.value = true;
  try { window.localStorage.setItem(recordKey(), String(personalBest.value)); } catch { /* record locale facoltativo */ }
}
function updateClock() {
  if (!session.value) return;
  elapsedMs.value = Math.min(25000, Math.max(0, clockElapsedBase + clockNow() - clockStartedAt));
  remainingMs.value = 25000 - elapsedMs.value;
  if (state.value === 'controlling') duelElapsedMs.value = Math.min(6000, Math.max(0, Math.floor(clockNow() - duelStartedAt)));
}
function tick() {
  if (disposed) return;
  updateClock();
  if (!remainingMs.value && ['ready', 'controlling', 'result'].includes(state.value)) finish();
  if (state.value === 'controlling') {
    const prediction = evaluateCrossoverDuel({ duelIndex: duelIndex.value, samples: snapshotSamples() });
    if (prediction?.won || duelElapsedMs.value >= 6000) launchDuel(true);
  }
  if (state.value === 'animating') {
    animationProgress.value = Math.min(1, Math.max(0, (clockNow() - animationStartedAt) / CROSSOVER_ANIMATION_MS));
    if (animationProgress.value >= 1) void saveDuel();
  }
  if (session.value && state.value !== 'finished' && (remainingMs.value > 0 || state.value === 'animating')) frameId = window.requestAnimationFrame(tick);
}
function finish() {
  releasePointer(); window.clearTimeout(resultTimer); window.cancelAnimationFrame(frameId); celebration.value = false; state.value = 'finished'; updateBest();
}
function readStats(data) {
  wins.value = data.wins; streak.value = data.streak; bestStreak.value = data.best_streak; qualityTotal.value = data.quality_total;
  updateBest();
}
function validStats(data) {
  return ['wins', 'streak', 'best_streak'].every((key) => Number.isInteger(data?.[key]) && data[key] >= 0 && data[key] <= 3)
    && Number.isInteger(data?.quality_total) && data.quality_total >= 0 && data.quality_total <= 300;
}
async function startSession() {
  if (disposed || state.value === 'starting' || receiptBusy || pendingDuel.value) return;
  window.cancelAnimationFrame(frameId); window.clearTimeout(resultTimer); state.value = 'starting'; errorMessage.value = '';
  try {
    const response = await startCrossoverSession(gameEventId);
    if (disposed) return;
    const data = response.data;
    if (!response.ok || !data?.session_id || !Number.isInteger(data.next_duel) || data.next_duel < 0 || data.next_duel >= 3 || !Number.isFinite(data.remaining_ms) || data.remaining_ms <= 0 || data.remaining_ms > 25000 || !validStats(data)) {
      state.value = 'error'; errorMessage.value = response.message || 'Campo non disponibile. Controlla la connessione e riprova.'; return;
    }
    session.value = data; duelIndex.value = data.next_duel; clockElapsedBase = 25000 - data.remaining_ms; clockStartedAt = clockNow();
    elapsedMs.value = clockElapsedBase; remainingMs.value = data.remaining_ms; samples.value = []; controlX.value = 0; duelElapsedMs.value = 0;
    outcome.value = null; releasePose.value = null; animationProgress.value = 0; celebration.value = false; hint.value = ''; newRecord.value = false;
    readStats(data); state.value = 'ready'; frameId = window.requestAnimationFrame(tick);
  } catch { if (!disposed) { state.value = 'error'; errorMessage.value = 'Non siamo riusciti a preparare il campo. Riprova.'; } }
}
function courtPoint(event) {
  const rect = courtEl.value?.getBoundingClientRect();
  if (!rect?.width || !rect.height) return null;
  const scale = Math.min(rect.width / CROSSOVER_WORLD.width, rect.height / CROSSOVER_WORLD.height);
  return { x: (event.clientX - rect.left - (rect.width - 360 * scale) / 2) / scale, y: (event.clientY - rect.top - (rect.height - 560 * scale) / 2) / scale };
}
function beginControl() {
  updateClock();
  if (!remainingMs.value) { finish(); return false; }
  duelStartMs = Math.floor(elapsedMs.value); duelStartedAt = clockNow(); duelElapsedMs.value = 0;
  samples.value = [{ t_ms: 0, x: 0 }]; controlX.value = 0; outcome.value = null; hint.value = ''; state.value = 'controlling'; return true;
}
function pushSample() {
  const t = duelElapsedMs.value;
  const last = samples.value.at(-1);
  if (!last || t - last.t_ms < 50 || samples.value.length >= 127) return;
  samples.value.push({ t_ms: t, x: controlX.value });
}
function snapshotSamples() {
  const points = samples.value.map((point) => ({ ...point }));
  const last = points.at(-1);
  if (!last) return points;
  if (duelElapsedMs.value > last.t_ms && points.length < 128) points.push({ t_ms: duelElapsedMs.value, x: controlX.value });
  else if (last.t_ms > 0) last.x = controlX.value;
  return points;
}
function onPointerDown(event) {
  if (state.value !== 'ready' || !session.value || event.isPrimary === false || (event.button != null && event.button !== 0)) return;
  const point = courtPoint(event);
  if (!point || point.x < 0 || point.x > 360 || point.y < 405 || point.y > 560 || !beginControl()) return;
  event.preventDefault(); courtEl.value?.focus({ preventScroll: true }); pointerId = event.pointerId; pointerStart = point;
  try { courtEl.value?.setPointerCapture(pointerId); } catch { /* capture non disponibile */ }
}
function onPointerMove(event) {
  if (state.value !== 'controlling' || pointerId !== event.pointerId || !pointerStart) return;
  updateClock();
  if (!remainingMs.value) { finish(); return; }
  const point = courtPoint(event);
  if (!point) return;
  controlX.value = Math.max(-1, Math.min(1, (point.x - pointerStart.x) / 104));
  pushSample();
  const prediction = evaluateCrossoverDuel({ duelIndex: duelIndex.value, samples: snapshotSamples() });
  if (prediction?.won || duelElapsedMs.value >= 6000) launchDuel(true);
}
function releasePointer() {
  const previous = pointerId; pointerId = null; pointerStart = null;
  if (previous != null) { try { courtEl.value?.releasePointerCapture(previous); } catch { /* capture già rilasciata */ } }
}
function cancelControl(event) {
  if (pointerId != null && event?.pointerId != null && event.pointerId !== pointerId) return;
  releasePointer();
  if (state.value === 'controlling') { state.value = 'ready'; samples.value = []; controlX.value = 0; duelElapsedMs.value = 0; }
}
function onLostCapture(event) { if (pointerId === event.pointerId) cancelControl(event); }
function onPointerUp(event) {
  if (state.value !== 'controlling' || pointerId !== event.pointerId) return;
  onPointerMove(event);
  if (state.value === 'controlling') launchDuel();
}
function onCourtKeydown(event) {
  if (!['ArrowLeft', 'ArrowRight', 'Enter', ' ', 'Escape'].includes(event.key)) return;
  if (event.key === 'Escape') { if (state.value === 'controlling') { event.preventDefault(); event.stopPropagation(); cancelControl(); } return; }
  if (pointerId != null || !['ready', 'controlling'].includes(state.value)) return;
  event.preventDefault();
  if (event.key === 'Enter' || event.key === ' ') { if (state.value === 'controlling') launchDuel(); return; }
  if (state.value === 'ready' && !beginControl()) return;
  updateClock(); if (!remainingMs.value) { finish(); return; }
  const nextX = event.key === 'ArrowLeft' ? -.85 : .85;
  if (nextX !== controlX.value) {
    const t = Math.max(1, duelElapsedMs.value);
    const last = samples.value.at(-1);
    if (t - 1 > last.t_ms && samples.value.length < 126) samples.value.push({ t_ms: t - 1, x: controlX.value });
    if (t > samples.value.at(-1).t_ms && samples.value.length < 127) samples.value.push({ t_ms: t, x: nextX });
  } else pushSample();
  controlX.value = nextX;
  if (evaluateCrossoverDuel({ duelIndex: duelIndex.value, samples: snapshotSamples() })?.won) launchDuel(true);
}
function launchDuel(timedOut = false) {
  if (state.value !== 'controlling' || !session.value) return;
  updateClock();
  if (!remainingMs.value) { finish(); return; }
  const points = snapshotSamples();
  if (!timedOut && (duelElapsedMs.value < 80 || !points.some((point) => Math.abs(point.x) >= .12))) { cancelControl(); return; }
  const result = evaluateCrossoverDuel({ duelIndex: duelIndex.value, samples: points });
  releasePointer();
  if (!result) { cancelControl(); return; }
  const held = pose.value;
  releasePose.value = { player: { ...held.player }, defender: { ...held.defender }, ball: { ...held.ball } };
  samples.value = points; outcome.value = result;
  pendingDuel.value = { sessionId: session.value.session_id, duel: duelIndex.value, duel_start_ms: duelStartMs, samples: points.map((point) => ({ ...point })) };
  animationProgress.value = 0; animationStartedAt = clockNow(); celebration.value = false; hint.value = ''; state.value = 'animating';
}
async function saveDuel() {
  if (disposed || receiptBusy || !pendingDuel.value) return;
  receiptBusy = true; state.value = 'saving'; errorMessage.value = ''; emit('saving-change', true);
  const pending = pendingDuel.value;
  try {
    const response = await submitCrossoverDuel(gameEventId, pending.sessionId, { duel: pending.duel, duel_start_ms: pending.duel_start_ms, samples: pending.samples.map((point) => ({ ...point })) });
    if (disposed) return;
    if (!response.ok) {
      if ([404, 410].includes(response.status)) { pendingDuel.value = null; errorMessage.value = response.message || 'Il tempo per recuperare questo duello è terminato.'; finish(); }
      else { state.value = 'error'; errorMessage.value = response.message || 'Connessione interrotta. Riprova a salvare lo stesso duello: sarà conteggiato una sola volta.'; }
      return;
    }
    const receipt = response.data;
    if (receipt?.session_id !== pending.sessionId || receipt.duel !== pending.duel || typeof receipt.won !== 'boolean' || !Number.isInteger(receipt.wallet) || receipt.wallet < 0 || !Number.isInteger(receipt.crossover_earned) || receipt.crossover_earned < 0 || !Number.isInteger(receipt.next_duel) || receipt.next_duel <= pending.duel || receipt.next_duel > 3 || !validStats(receipt) || !Number.isInteger(receipt.quality) || receipt.quality < 0 || receipt.quality > 100) throw new Error('Invalid receipt');
    lastReceipt.value = receipt; confirmedEarned.value = Math.max(confirmedEarned.value, receipt.crossover_earned); readStats(receipt);
    celebration.value = receipt.won;
    if (receipt.won) playWinSound();
    else hint.value = receipt.reason === 'no_feint' ? 'Mantieni la finta un istante prima di cambiare mano.' : receipt.reason === 'no_crossover' ? 'Dopo la finta, porta la palla dall’altra parte.' : 'Il difensore ha recuperato: prova un cambio più rapido.';
    emit('credited', { ...receipt, event_id: gameEventId });
    pendingDuel.value = null; duelIndex.value = receipt.next_duel; updateClock();
    if (Number.isFinite(receipt.remaining_ms)) { clockElapsedBase = Math.max(elapsedMs.value, 25000 - Math.max(0, Math.min(25000, receipt.remaining_ms))); clockStartedAt = clockNow(); updateClock(); }
    state.value = 'result';
    if (!remainingMs.value || duelIndex.value >= 3) finish();
    else resultTimer = window.setTimeout(() => {
      if (disposed) return;
      celebration.value = false; outcome.value = null; releasePose.value = null; samples.value = []; controlX.value = 0; duelElapsedMs.value = 0; updateClock();
      if (!remainingMs.value) finish(); else state.value = 'ready';
    }, receipt.won ? 650 : 450);
  } catch { if (!disposed) { state.value = 'error'; errorMessage.value = 'Non siamo riusciti a verificare il duello. Riprova il salvataggio.'; } }
  finally { receiptBusy = false; if (!disposed) emit('saving-change', false); }
}
function retry() { if (pendingDuel.value) void saveDuel(); else void startSession(); }
function toggleSound() {
  soundEnabled.value = !soundEnabled.value;
  if (!soundEnabled.value) return;
  try { const Audio = window.AudioContext || window.webkitAudioContext; if (!Audio) { soundEnabled.value = false; return; } audioContext ||= new Audio(); void audioContext.resume().catch(() => { soundEnabled.value = false; }); } catch { soundEnabled.value = false; }
}
function playWinSound() {
  if (!soundEnabled.value || !audioContext) return;
  try {
    [440, 660, 880].forEach((frequency, i) => {
      const tone = audioContext.createOscillator(); const gain = audioContext.createGain(); const at = audioContext.currentTime + i * .065;
      tone.frequency.value = frequency; gain.gain.setValueAtTime(0, at); gain.gain.linearRampToValueAtTime(.035, at + .012); gain.gain.exponentialRampToValueAtTime(.001, at + .22);
      tone.connect(gain); gain.connect(audioContext.destination); tone.start(at); tone.stop(at + .23);
    });
  } catch { /* audio facoltativo */ }
}
onBeforeUnmount(() => {
  disposed = true; window.cancelAnimationFrame(frameId); window.clearTimeout(resultTimer); releasePointer(); emit('saving-change', false);
  if (audioContext) void audioContext.close().catch(() => {});
});
</script>

<style scoped>
.crossover { display: flex; flex-direction: column; width: 100%; height: 100%; min-height: 0; overflow: hidden; border-radius: 24px; background: #0c1724; color: #f3faff; box-shadow: 0 12px 42px #0005; font-family: 'Inter', system-ui, sans-serif; }
.crossover * { box-sizing: border-box; }
.cr-header { display: flex; align-items: center; gap: 9px; padding: 12px 14px; flex: none; background: linear-gradient(125deg, #233548, #111f30); border-bottom: 1px solid #dfffff12; }
.cr-brand { flex: 1; min-width: 0; }
.cr-brand > span { display: block; max-width: 170px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: #a9bdce; font-size: 8px; font-weight: 700; letter-spacing: .16em; text-transform: uppercase; }
.cr-brand b { display: block; margin-top: 4px; font-size: 16px; font-weight: 950; line-height: 1; letter-spacing: -.05em; white-space: nowrap; }
.cr-brand em { color: #e5c48f; font-style: normal; }
.cr-brand > small { display: block; margin-top: 5px; color: #7c99b1; font-size: 6px; font-weight: 800; letter-spacing: .19em; }
.cr-wallet { display: flex; align-items: center; gap: 6px; padding: 7px 10px; border-radius: 99px; background: #f7c25b18; color: #ffe0a0; font-size: 13px; font-weight: 800; font-variant-numeric: tabular-nums; }
.cr-coin { display: inline-block; width: 14px; height: 14px; flex: none; border-radius: 50%; background: radial-gradient(circle at 30% 25%, #ffecb8, #f3bd49 55%, #ac6b12); box-shadow: inset 0 0 0 2px #ffd980, 0 2px 4px #0003; }
.cr-sound { display: grid; place-items: center; width: 30px; height: 30px; padding: 6px; border: 1px solid #d5faff20; border-radius: 50%; background: #fff1; color: #d0e3eb; cursor: pointer; }
.cr-sound svg { width: 100%; height: 100%; }
.cr-round { display: flex; align-items: center; justify-content: space-between; gap: 9px; padding: 9px 15px; flex: none; background: #142637; }
.cr-round > span { color: #93aec3; font-size: 8px; font-weight: 800; letter-spacing: .09em; }
.cr-round > span b { margin-left: 3px; color: #f1dfbd; font-size: 13px; }
.cr-round > span small { margin-left: 3px; color: #6c8ca3; font-size: 9px; }
.cr-round-dots { display: flex; gap: 5px; }
.cr-round-dots i { width: 22px; height: 4px; border-radius: 9px; background: #d5e8ff18; }
.cr-round-dots .current { background: #f4d69c; }
.cr-round-dots .completed { background: #81bbad; }
.cr-time { min-width: 35px; text-align: right; color: #f4dfb8; font-size: 14px; font-variant-numeric: tabular-nums; }
.cr-time small { margin-left: 1px; color: #91adbf; font-size: 8px; }
.cr-time.urgent { color: #ff9e8c; }
.cr-arena { position: relative; display: flex; flex: 1; min-height: 0; overflow: hidden; background: #0b1726; }
.cr-court { display: block; width: 100%; height: 100%; touch-action: none; user-select: none; -webkit-user-select: none; -webkit-tap-highlight-color: transparent; outline-offset: -3px; }
.cr-court:focus-visible { outline: 2px solid #e9d29f; }
.cr-stats { position: absolute; z-index: 2; top: 9px; left: 14px; right: 14px; display: flex; justify-content: space-between; pointer-events: none; }
.cr-stats > span { text-align: center; }
.cr-stats small { display: block; color: #8ea9c0; font-size: 6px; font-weight: 800; letter-spacing: .1em; }
.cr-stats b { display: block; margin-top: 4px; color: #f5e6c9; font-size: 21px; font-weight: 900; line-height: 1; font-variant-numeric: tabular-nums; }
.cr-gesture-hint { position: absolute; z-index: 2; top: 40%; left: 0; right: 0; display: flex; flex-direction: column; align-items: center; gap: 5px; pointer-events: none; text-shadow: 0 2px 12px #000; }
.cr-gesture-hint b { color: #ffe0a3; font-size: 13px; font-weight: 900; letter-spacing: .08em; }
.cr-gesture-hint span { color: #b9cfdc; font-size: 9px; }
.cr-net.net-hit { transform-origin: 0 0; transform: scaleY(1.2) skewX(-4deg); }
.cr-cover { position: absolute; z-index: 4; inset: 0; display: flex; flex-direction: column; align-items: center; justify-content: safe center; gap: 14px; padding: 22px; overflow-y: auto; background: linear-gradient(180deg, #0b1a2ba8, #0b1a25ef); backdrop-filter: blur(3px); text-align: center; }
.cr-cover > * { flex-shrink: 0; }
.cr-eyebrow { color: #d9be8d; font-size: 7px; font-weight: 800; letter-spacing: .16em; }
.cr-cover-icon { display: grid; place-items: center; width: 65px; height: 65px; border: 1px solid #d4bd8938; border-radius: 23px; background: linear-gradient(135deg, #d8be7f25, #bfe3d105); color: #efdbab; font-size: 35px; box-shadow: 0 5px 28px #0004; }
.cr-cover h2 { margin: 0; max-width: 270px; color: #fff7e6; font-size: clamp(22px, 7vw, 29px); font-weight: 900; line-height: 1.12; letter-spacing: -.04em; }
.cr-cover p { margin: 0; max-width: 270px; color: #b9ccd8; font-size: 10px; line-height: 1.65; }
.cr-cover > small { max-width: 260px; color: #829fb6; font-size: 8px; line-height: 1.5; }
.cr-primary { display: flex; align-items: center; justify-content: space-between; gap: 18px; width: 100%; max-width: 265px; padding: 14px 17px; border: 0; border-radius: 14px; background: linear-gradient(120deg, #f0d8a6, #cfac71); color: #332b21; font: inherit; font-size: 12px; font-weight: 850; box-shadow: 0 5px 18px #0003; cursor: pointer; }
.cr-primary span { font-size: 17px; }
.cr-primary:disabled { opacity: .6; cursor: wait; }
.cr-summary-stats { display: flex; gap: 20px; margin: 4px 0; }
.cr-summary-stats > span { display: flex; flex-direction: column; align-items: center; gap: 7px; }
.cr-summary-stats b { display: flex; align-items: center; gap: 4px; color: #f2e5ce; font-size: 29px; line-height: 1; font-weight: 900; }
.cr-summary-stats b small { margin-top: 10px; font-size: 12px; }
.cr-summary-stats > span > small { color: #91aabf; font-size: 6px; font-weight: 800; letter-spacing: .06em; }
.cr-summary-stats .cr-gold { color: #ffdc91; }
.cr-footer { display: flex; align-items: center; justify-content: space-between; gap: 10px; padding: 12px 14px; min-height: 65px; flex: none; background: linear-gradient(120deg, #21364a, #10232f); border-top: 1px solid #d5f9ff10; }
.cr-footer > div { flex: 1; min-width: 0; }
.cr-footer b { display: block; font-size: 11px; font-weight: 800; }
.cr-footer div > span { display: block; margin-top: 4px; color: #9db5c8; font-size: 9px; line-height: 1.45; }
.cr-match { display: flex; align-items: center; justify-content: center; flex-wrap: wrap; gap: 4px; width: 65px; flex: none; color: #ffdf9c; font-size: 15px; font-weight: 850; font-variant-numeric: tabular-nums; }
.cr-match small { flex: 0 0 100%; text-align: center; color: #8fa9be; font-size: 7px; font-weight: 600; }
.cr-retry { padding: 9px 11px; max-width: 112px; border: 0; border-radius: 11px; background: #d9c293; color: #272d2c; font: inherit; font-size: 10px; font-weight: 800; cursor: pointer; }
.cr-celebration { position: absolute; z-index: 3; top: 40%; left: 0; right: 0; display: flex; flex-direction: column; align-items: center; gap: 6px; pointer-events: none; text-shadow: 0 3px 18px #000; animation: cr-pop .2s ease-out; }
.cr-celebration b { color: #ffe1a0; font-size: clamp(19px, 6.5vw, 28px); font-weight: 950; letter-spacing: -.045em; }
.cr-celebration span { color: #e6f9ff; font-size: 9px; font-weight: 700; }
.cr-coin-burst { animation: cr-float .65s ease-out both; }
.is-won .cr-wallet { box-shadow: 0 0 18px #ffe09b30; }
@keyframes cr-pop { from { opacity: 0; transform: translateY(6px) scale(.95); } to { opacity: 1; transform: none; } }
@keyframes cr-float { from { opacity: 0; translate: 0 10px; } 30% { opacity: 1; } to { opacity: 0; translate: 0 -18px; } }
@media (max-height: 650px) { .cr-header { padding: 9px 12px; } .cr-cover { gap: 10px; padding: 16px; } .cr-cover-icon { width: 43px; height: 43px; border-radius: 15px; font-size: 26px; } .cr-footer { min-height: 57px; padding: 9px 12px; } }
@media (prefers-reduced-motion: reduce) { .cr-coin-burst, .cr-celebration { animation: none; } .cr-speed { display: none; } }
</style>
