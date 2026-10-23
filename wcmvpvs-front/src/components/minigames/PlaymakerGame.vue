<template>
  <section class="playmaker" :class="{ 'is-assist': celebration }" aria-label="Playmaker, l’assist perfetto">
    <header class="pm-header">
      <div class="pm-brand"><span>{{ clubName || 'COURTSIDE' }}</span><b>PLAY<span>MAKER</span></b><small>L’ASSIST PERFETTO</small></div>
      <div class="pm-wallet" aria-label="Saldo monete"><i class="pm-coin" aria-hidden="true"></i>{{ walletCoins }}</div>
      <button class="pm-sound" type="button" :aria-pressed="soundEnabled" :aria-label="soundEnabled ? 'Disattiva audio' : 'Attiva audio'" @click="toggleSound">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true"><path d="M11 5 6 9H3v6h3l5 4Z"/><path v-if="soundEnabled" d="M15 8c3 2 3 6 0 8m3-11c5 4 5 10 0 14"/><path v-else d="m16 9 6 6m0-6-6 6"/></svg>
      </button>
    </header>
    <div class="pm-clock" :class="{ 'clock-low': remainingMs <= 5000 && session }">
      <span>{{ session && state !== 'finished' ? 'TEMPO' : 'LA TUA REGIA' }}</span>
      <div><i :style="{ width: `${remainingMs / 300}%` }"></i></div><b>{{ secondsLeft }}<small>s</small></b>
    </div>
    <div class="pm-arena">
      <div class="pm-stats" aria-live="off"><span><small>ASSIST</small><b>{{ sessionAssists }}</b></span><span><small>DI FILA</small><b>{{ streak }}<i v-if="streak > 1">🔥</i></b></span><span><small>RECORD PERSONALE</small><b>{{ personalBest }}</b></span></div>
      <svg ref="courtEl" class="pm-court" viewBox="0 0 360 560" preserveAspectRatio="xMidYMid meet" tabindex="0" role="application" aria-label="Trascina la palla verso il compagno libero e rilascia. Con la tastiera scegli il compagno con le frecce e passa con Invio."
        @pointerdown="onPointerDown" @pointermove="onPointerMove" @pointerup="onPointerUp" @pointercancel="cancelAim" @lostpointercapture="onLostCapture" @keydown="onCourtKeydown">
        <defs>
          <radialGradient id="pm-arena"><stop stop-color="#203b46"/><stop offset="1" stop-color="#08141e"/></radialGradient>
          <linearGradient id="pm-floor" x2="0" y2="1"><stop stop-color="#4e4336"/><stop offset=".5" stop-color="#916842"/><stop offset="1" stop-color="#b08455"/></linearGradient>
          <linearGradient id="pm-jersey" x2="1" y2="1"><stop stop-color="var(--red, #37bcb5)"/><stop offset="1" stop-color="var(--red-deep, #167775)"/></linearGradient>
          <linearGradient id="pm-defense" x2="1" y2="1"><stop stop-color="#eb7477"/><stop offset="1" stop-color="#903c52"/></linearGradient>
          <linearGradient id="pm-glass" x2="1" y2="1"><stop stop-color="#daffff" stop-opacity=".3"/><stop offset="1" stop-color="#b2e8f3" stop-opacity=".05"/></linearGradient>
          <radialGradient id="pm-ball" cx=".3" cy=".25" r=".8"><stop stop-color="#ffda92"/><stop offset=".4" stop-color="#f6a041"/><stop offset=".8" stop-color="#c55a1d"/><stop offset="1" stop-color="#632c15"/></radialGradient>
          <linearGradient id="pm-skin" x2="1" y2="1"><stop stop-color="#eac39c"/><stop offset="1" stop-color="#a97155"/></linearGradient>
          <radialGradient id="pm-target"><stop stop-color="#b9ffe9" stop-opacity=".24"/><stop offset="1" stop-color="#b9ffe9" stop-opacity="0"/></radialGradient>
          <pattern id="pm-wood" width="28" height="96" patternUnits="userSpaceOnUse"><path d="M0 0v96M0 48h28M6 0q-3 24 0 48t0 48m15-96q3 24 0 48t0 48" fill="none" stroke="#eac193" stroke-opacity=".1"/><path d="M28 0v96" stroke="#36271e" stroke-opacity=".2"/></pattern>
          <g id="pm-athlete"><path d="m-15-4-14 21 6 6 15-17m23-10 14 21-6 6L8 6" fill="url(#pm-skin)"/><path d="m-13-10-4 29 34 0-4-29-8-4H-5Z" fill="var(--pm-kit, #2f8b86)" stroke="#ffffff40" stroke-width="1"/><path d="M-13 19-15 35h12l3-10 3 10h12l-2-16Z" fill="var(--pm-kit, #2f8b86)"/><path d="m-13 35 1 10h7l2-10m8 0 2 10h7l-1-10" fill="url(#pm-skin)"/><path d="M-12 43q-7 0-7 7h14v-7m10 0v7h14q-1-7-9-7" fill="#e8f5f7"/><circle cy="-23" r="10" fill="url(#pm-skin)"/><path d="M-10-24q0-14 11-11 10 0 9 13l-7-5-10 1Z" fill="#26252b"/><path d="M-5-13q5 5 10 0M-12-7l5 3M12-7 7-4" fill="none" stroke="#ffffff90" stroke-width="2"/></g>
        </defs>
        <rect width="360" height="560" fill="url(#pm-arena)"/>
        <path d="M-20 0h90l80 280H-20Zm400 0h-90l-80 280h170Z" fill="#b5fff7" opacity=".035"/>
        <g fill="#adced7" opacity=".2"><circle v-for="(dot, i) in crowd" :key="i" :cx="dot.x" :cy="dot.y" :r="dot.r"/></g>
        <path d="M33 158h294l98 402H-65Z" fill="url(#pm-floor)"/><path d="M33 158h294l98 402H-65Z" fill="url(#pm-wood)"/>
        <g fill="none" stroke="#f9e5bf" stroke-width="1.5" opacity=".5"><path d="M40 169h280l76 368H-36ZM126 169 101 314q79 33 158 0l-25-145M111 267q69 28 138 0M14 292q166 245 332 0"/><ellipse cx="180" cy="505" rx="61" ry="18"/></g>
        <path d="M126 169 101 314q79 33 158 0l-25-145Z" fill="var(--red, #3fa7a1)" opacity=".09"/>
        <text x="180" y="405" text-anchor="middle" font-size="15" font-weight="900" letter-spacing="4" fill="#ffe2b0" opacity=".12">PLAYMAKER</text>
        <image v-if="clubLogoUrl" :href="clubLogoUrl" x="161" y="342" width="38" height="38" opacity=".2" preserveAspectRatio="xMidYMid meet"/>
        <g transform="translate(180 118)"><path d="M-4-115h8v61H-4Z" fill="#36515d"/><rect x="-53" y="-61" width="106" height="64" rx="5" fill="url(#pm-glass)" stroke="#d5f4f9" stroke-opacity=".6" stroke-width="2"/><rect x="-20" y="-30" width="40" height="25" fill="none" stroke="#edfaff" stroke-opacity=".8" stroke-width="2"/><path d="M-56 3H56" stroke="var(--red, #60d6c6)" stroke-width="4"/><ellipse cy="4" rx="25" ry="7" fill="#13232b"/><path d="M-25 0q25-13 50 0" fill="none" stroke="#ffb460" stroke-width="4"/></g>

        <g v-for="mate in scene.teammates" :key="mate.id" :transform="`translate(${mate.x}, ${mate.y})`" :opacity="animation.runner && activePass?.receiver?.id === mate.id ? 0 : 1">
          <ellipse cy="49" rx="21" ry="7" fill="#000" opacity=".25"/><ellipse cy="11" rx="39" ry="48" fill="url(#pm-target)"/>
          <ellipse v-if="state === 'ready' || state === 'aiming'" cy="45" rx="27" ry="9" fill="none" stroke="#b7ffe3" stroke-opacity=".6" stroke-dasharray="4 4"/>
          <use href="#pm-athlete" :style="{ '--pm-kit': 'url(#pm-jersey)' }"/><text y="9" text-anchor="middle" fill="var(--on-primary, #fff)" font-size="12" font-weight="900">{{ mate.id === 'left' ? '7' : '11' }}</text>
          <g v-if="state === 'ready' || state === 'aiming'" transform="translate(0 -50)"><rect x="-24" y="-9" width="48" height="17" rx="8.5" fill="#132e33" stroke="#a8f6d6" stroke-opacity=".25"/><text y="2.5" text-anchor="middle" fill="#b6fbe0" font-size="7" font-weight="900" letter-spacing="1">PASSA</text></g>
        </g>
        <g v-for="defender in scene.defenders" :key="defender.id" :transform="`translate(${defender.x}, ${defender.y}) scale(.98)`">
          <ellipse cy="49" rx="23" ry="7" fill="#000" opacity=".3"/><ellipse cy="12" :rx="defender.radius + 5" ry="34" fill="#fc7881" opacity=".065"/>
          <use href="#pm-athlete" :style="{ '--pm-kit': 'url(#pm-defense)' }"/><path d="M-29 16-37 5m66 11 8-11" stroke="url(#pm-skin)" stroke-width="7" stroke-linecap="round"/><text y="9" text-anchor="middle" fill="#ffe2e5" font-size="11" font-weight="900">{{ defender.id === 'guard' ? '4' : '9' }}</text>
        </g>
        <g v-if="state === 'aiming' && preview" aria-hidden="true"><path :d="`M180 466Q${(180 + preview.aim.x) / 2} ${(466 + preview.aim.y) / 2 - 18} ${preview.aim.x} ${preview.aim.y}`" fill="none" stroke="#e1fff2" stroke-width="2" stroke-dasharray="4 7" opacity=".65"/><circle :cx="preview.aim.x" :cy="preview.aim.y" r="16" fill="#b7ffe0" fill-opacity=".1" stroke="#d8fff0" stroke-dasharray="3 4"/><path :d="`M${preview.aim.x - 5} ${preview.aim.y}h10m-5-5v10`" stroke="#d8fff0"/></g>
        <g v-if="animation.runner" :transform="`translate(${animation.runner.x}, ${animation.runner.y}) scale(${animation.runner.scale})`" aria-hidden="true"><ellipse :cy="49 + animation.runner.jump" rx="21" ry="7" fill="#000" opacity=".2"/><use href="#pm-athlete" :style="{ '--pm-kit': 'url(#pm-jersey)' }"/><text y="9" text-anchor="middle" fill="var(--on-primary, #fff)" font-size="12" font-weight="900">{{ activePass?.receiver?.id === 'left' ? '7' : '11' }}</text><path v-if="animation.phase === 'dunk'" d="m-13-3-3-27m29 27 3-27" stroke="url(#pm-skin)" stroke-width="7" stroke-linecap="round"/></g>
        <g v-if="state === 'ready' || state === 'aiming'" :transform="`translate(${drag.x * .06}, ${drag.y * .06})`" aria-hidden="true"><path d="M112 560q2-42 20-70l20-26q6-5 9-1t-3 12l-12 21q12-12 16-6t-6 19l-17 28-1 23m110 0q-2-42-20-70l-20-26q-6-5-9-1t3 12l12 21q-12-12-16-6t6 19l17 28 1 23" fill="url(#pm-skin)"/><path d="m110 548 31-3 1 15h-33m142-12-31-3-1 15h33" fill="var(--red, #35a99b)"/></g>
        <g v-if="animation.ball.visible" :transform="`translate(${animation.ball.x}, ${animation.ball.y}) scale(${animation.ball.scale}) rotate(${animation.ball.rotation})`" aria-hidden="true"><circle r="26" fill="url(#pm-ball)"/><g fill="none" stroke="#562a15" stroke-width="1.6"><path d="M-26 0q26 8 52 0M0-26q-8 26 0 52M-20-16q26 9 0 36M20-16q-26 9 0 36"/><circle r="25" stroke-opacity=".35"/></g><path d="M-14-15q7-6 14-6" fill="none" stroke="#ffedbd" stroke-width="2.5" stroke-linecap="round" opacity=".5"/></g>
        <g transform="translate(180 118)" aria-hidden="true"><g class="pm-net" :class="{ 'net-hit': animation.netHit }" fill="none" stroke="#f0fbff" stroke-opacity=".65" stroke-width="1"><path d="M-24 1-16 36q16 9 32 0L24 1M-16 5-10 39M-8 7-4 40M0 7v34M8 7 4 40M16 5 10 39M-22 12q22 11 44 0M-19 24q19 10 38 0M-16 35q16 8 32 0"/></g><path d="M-25 0q25 14 50 0" fill="none" stroke="#f5873f" stroke-width="4" stroke-linecap="round"/></g>
        <g v-if="celebration" class="pm-reward" aria-hidden="true"><text x="180" y="190" text-anchor="middle" fill="#ffe1a0" font-size="34" font-weight="900">+1</text><circle cx="211" cy="178" r="10" fill="#eeb744" stroke="#ffe4ab" stroke-width="2"/></g>
        <g v-if="state === 'ready' && !sessionAttempts" class="pm-drag-hint" fill="none" stroke="#e2fff3" stroke-opacity=".7" aria-hidden="true"><path d="M166 426 127 345m-3 10 3-10 10 3" stroke-width="2" stroke-linecap="round"/><circle cx="180" cy="466" r="40" stroke-dasharray="3 7"/></g>
      </svg>
      <div v-if="celebration" class="pm-celebration" aria-hidden="true"><b>{{ lastReceipt?.clean ? 'ASSIST PERFETTO!' : 'CHE ASSIST!' }}</b><span>{{ streak > 1 ? `${streak} di fila · ` : '' }}1 moneta salvata</span></div>
      <div v-if="state === 'intro' || state === 'starting'" class="pm-cover pm-intro"><span class="pm-cover-eyebrow">LEGGI. SCEGLI. SERVI.</span><div class="pm-cover-icon" aria-hidden="true">↗</div><h2>Apri la difesa.</h2><p>Trascina la palla verso il compagno libero.<br>Ogni assist a canestro vale una moneta.</p><button type="button" class="pm-primary" :disabled="state === 'starting'" @click="startSession">{{ state === 'starting' ? 'Entriamo in campo…' : 'Scendi in campo · 30s' }}<span aria-hidden="true">→</span></button><small>Si gioca con un dito. Audio facoltativo.</small></div>
      <div v-if="state === 'finished'" class="pm-cover pm-summary"><span class="pm-cover-eyebrow">LA TUA REGIA, IN NUMERI</span><h2>{{ newRecord ? 'Nuovo record!' : sessionAssists ? 'Bella visione di gioco.' : 'Il prossimo assist è tuo.' }}</h2><div class="pm-summary-stats"><span><b>{{ sessionAssists }}</b><small>ASSIST</small></span><span><b>{{ bestStreak }}</b><small>SERIE MIGLIORE</small></span><span><b class="pm-gold">{{ sessionAssists }}<i class="pm-coin" aria-hidden="true"></i></b><small>MONETE SALVATE</small></span></div><p>{{ errorMessage || 'Cerca lo spazio, aspetta il momento e trova il compagno.' }}</p><button type="button" class="pm-primary" @click="startSession">Gioca ancora · 30s<span aria-hidden="true">↻</span></button><small>Record personale: {{ personalBest }} assist</small></div>
    </div>
    <footer class="pm-footer"><div role="status" aria-live="polite"><b>{{ statusTitle }}</b><span>{{ statusDetail }}</span></div><button v-if="state === 'error'" type="button" class="pm-retry" @click="retry">{{ pendingPass ? 'Salva l’assist' : 'Riprova' }}</button><span v-else class="pm-match"><i class="pm-coin" aria-hidden="true"></i>{{ matchEarned }}<small>nella partita</small></span></footer>
  </section>
</template>

<script setup>
import { computed, onBeforeUnmount, ref } from 'vue';
import { startPlaymakerSession, submitPlaymakerPass } from '../../api';
import { PLAYMAKER_WORLD, PLAYMAKER_START, getPlaymakerScene, createPlaymakerPass, samplePlaymakerPass } from '../../games/playmaker';

const props = defineProps({ eventId: { type: Number, required: true }, walletCoins: { type: Number, default: 0 }, earnedCoins: { type: Number, default: 0 }, clubName: { type: String, default: '' }, clubLogoUrl: { type: String, default: '' }, organizationSlug: { type: String, default: '' } });
const emit = defineEmits(['credited', 'saving-change', 'round-started', 'round-completed']);
const gameEventId = props.eventId;
const state = ref('intro');
const courtEl = ref(null);
const session = ref(null);
const elapsedMs = ref(0);
const remainingMs = ref(30000);
const attempt = ref(0);
const drag = ref({ x: 0, y: 0 });
const activePass = ref(null);
const pendingPass = ref(null);
const flightProgress = ref(0);
const sessionAssists = ref(0);
const sessionAttempts = ref(0);
const streak = ref(0);
const bestStreak = ref(0);
const confirmedEarned = ref(0);
const celebration = ref(false);
const lastReceipt = ref(null);
const errorMessage = ref('');
const hint = ref('');
const personalBest = ref(readBest());
const newRecord = ref(false);
const soundEnabled = ref(false);
const crowd = Array.from({ length: 58 }, (_, i) => ({ x: i % 15 * 27 - 8, y: 75 + Math.floor(i / 15) * 23, r: 1 + i % 3 * .4 }));
let pointerId = null;
let pointerStart = null;
let frameId;
let resultTimer;
let clockStartedAt = 0;
let clockElapsedBase = 0;
let flightStartedAt = 0;
let disposed = false;
let receiptBusy = false;
let audioContext;

const secondsLeft = computed(() => Math.max(0, Math.ceil(remainingMs.value / 1000)));
const matchEarned = computed(() => Math.max(Number(props.earnedCoins) || 0, confirmedEarned.value));
const scene = computed(() => activePass.value && ['flying', 'saving', 'result', 'error'].includes(state.value) ? activePass.value.scene : getPlaymakerScene({ attempt: attempt.value, elapsedMs: elapsedMs.value }));
const preview = computed(() => state.value === 'aiming' ? createPlaymakerPass({ attempt: attempt.value, elapsedMs: elapsedMs.value, drag: drag.value }) : null);
const animation = computed(() => activePass.value && ['flying', 'saving', 'result', 'error'].includes(state.value)
  ? samplePlaymakerPass(activePass.value, flightProgress.value)
  : { ball: { x: 180 + drag.value.x * .06, y: 466 + drag.value.y * .06, scale: 1, rotation: 0, visible: true }, runner: null, netHit: false, phase: 'held' });
const statusTitle = computed(() => state.value === 'error' ? pendingPass.value ? 'L’azione resta da verificare' : 'Rientra in campo' : state.value === 'flying' ? 'L’azione prende vita…' : state.value === 'saving' ? 'Verifichiamo l’assist…' : celebration.value ? 'Hai costruito il canestro!' : state.value === 'aiming' ? 'Rilascia per passare' : state.value === 'finished' ? 'Le monete confermate sono salvate' : 'Trova il compagno libero');
const statusDetail = computed(() => errorMessage.value || hint.value || (state.value === 'saving' ? 'Attendi la conferma delle monete.' : 'Evita il difensore · 1 moneta per assist a canestro.'));

function clockNow() { return window.performance?.now?.() ?? Date.now(); }
function recordKey() { return `cento:playmaker:best:${encodeURIComponent(props.organizationSlug)}:${gameEventId}`; }
function readBest() { try { const value = Number(window.localStorage.getItem(recordKey())); return Number.isInteger(value) && value >= 0 ? value : 0; } catch { return 0; } }
function updateBest() {
  if (sessionAssists.value <= personalBest.value) return;
  personalBest.value = sessionAssists.value;
  newRecord.value = true;
  try { window.localStorage.setItem(recordKey(), String(personalBest.value)); } catch { /* record locale facoltativo */ }
}
function updateClock() {
  if (!session.value) return;
  elapsedMs.value = Math.min(30000, Math.max(0, clockElapsedBase + clockNow() - clockStartedAt));
  remainingMs.value = 30000 - elapsedMs.value;
}
function tick() {
  if (disposed) return;
  updateClock();
  if (remainingMs.value <= 0 && ['ready', 'aiming', 'result'].includes(state.value)) finish();
  if (state.value === 'flying') {
    flightProgress.value = Math.min(1, Math.max(0, (clockNow() - flightStartedAt) / activePass.value.flightMs));
    if (flightProgress.value >= 1) void savePass();
  }
  if (session.value && state.value !== 'finished' && (remainingMs.value > 0 || state.value === 'flying')) frameId = window.requestAnimationFrame(tick);
}
function finish() {
  if (session.value && state.value !== 'finished') emit('round-completed', { roundId: session.value.session_id });
  releasePointer();
  window.clearTimeout(resultTimer);
  window.cancelAnimationFrame(frameId);
  celebration.value = false;
  state.value = 'finished';
  updateBest();
}
async function startSession() {
  if (disposed || state.value === 'starting' || pendingPass.value || receiptBusy) return;
  window.cancelAnimationFrame(frameId);
  window.clearTimeout(resultTimer);
  state.value = 'starting';
  errorMessage.value = '';
  try {
    const response = await startPlaymakerSession(gameEventId);
    if (disposed) return;
    const data = response.data;
    if (!response.ok || !data?.session_id || !Number.isInteger(data.attempt) || data.attempt < 0 || !Number.isFinite(data.remaining_ms) || data.remaining_ms <= 0 || data.remaining_ms > 30000) {
      state.value = 'error'; errorMessage.value = response.message || 'Campo non disponibile. Controlla la connessione e riprova.'; return;
    }
    session.value = data;
    emit('round-started', { roundId: data.session_id });
    attempt.value = data.attempt;
    clockElapsedBase = 30000 - data.remaining_ms;
    clockStartedAt = clockNow();
    elapsedMs.value = clockElapsedBase;
    remainingMs.value = data.remaining_ms;
    sessionAssists.value = Number.isInteger(data.assists) && data.assists >= 0 ? data.assists : 0;
    sessionAttempts.value = data.attempt;
    streak.value = Number.isInteger(data.streak) && data.streak >= 0 ? data.streak : 0;
    bestStreak.value = Number.isInteger(data.best_streak) && data.best_streak >= 0 ? data.best_streak : 0;
    activePass.value = null; flightProgress.value = 0; drag.value = { x: 0, y: 0 };
    newRecord.value = false; celebration.value = false; hint.value = '';
    updateBest();
    state.value = 'ready';
    frameId = window.requestAnimationFrame(tick);
  } catch {
    if (!disposed) { state.value = 'error'; errorMessage.value = 'Non siamo riusciti a preparare il campo. Riprova.'; }
  }
}
function courtPoint(event) {
  const rect = courtEl.value?.getBoundingClientRect();
  if (!rect?.width || !rect.height) return null;
  const scale = Math.min(rect.width / PLAYMAKER_WORLD.width, rect.height / PLAYMAKER_WORLD.height);
  return { x: (event.clientX - rect.left - (rect.width - 360 * scale) / 2) / scale, y: (event.clientY - rect.top - (rect.height - 560 * scale) / 2) / scale };
}
function onPointerDown(event) {
  if (state.value !== 'ready' || !session.value || event.isPrimary === false || (event.button != null && event.button !== 0)) return;
  updateClock();
  if (!remainingMs.value) { finish(); return; }
  const point = courtPoint(event);
  if (!point || Math.hypot(point.x - 180, point.y - 466) > 55) return;
  event.preventDefault(); courtEl.value?.focus({ preventScroll: true });
  pointerId = event.pointerId; pointerStart = point;
  drag.value = { x: 0, y: 0 }; hint.value = ''; state.value = 'aiming';
  try { courtEl.value?.setPointerCapture(pointerId); } catch { /* capture non disponibile */ }
}
function onPointerMove(event) {
  if (state.value !== 'aiming' || pointerId !== event.pointerId || !pointerStart) return;
  const point = courtPoint(event);
  if (point) drag.value = { x: Math.max(-400, Math.min(400, point.x - pointerStart.x)), y: Math.max(-560, Math.min(100, point.y - pointerStart.y)) };
}
function releasePointer() {
  const previous = pointerId; pointerId = null; pointerStart = null;
  if (previous != null) { try { courtEl.value?.releasePointerCapture(previous); } catch { /* capture già rilasciata */ } }
}
function cancelAim(event) {
  if (pointerId != null && event?.pointerId != null && event.pointerId !== pointerId) return;
  releasePointer();
  if (state.value === 'aiming') { drag.value = { x: 0, y: 0 }; state.value = 'ready'; }
}
function onLostCapture(event) { if (pointerId === event.pointerId) cancelAim(event); }
function onPointerUp(event) {
  if (state.value !== 'aiming' || pointerId !== event.pointerId) return;
  onPointerMove(event); releasePointer(); launchPass();
}
function onCourtKeydown(event) {
  if (!['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Enter', ' ', 'Escape'].includes(event.key)) return;
  if (event.key === 'Escape') { if (state.value === 'aiming') { event.preventDefault(); event.stopPropagation(); cancelAim(); } return; }
  if (pointerId != null || !['ready', 'aiming'].includes(state.value)) return;
  event.preventDefault(); updateClock();
  if (!remainingMs.value) { finish(); return; }
  if (event.key === 'Enter' || event.key === ' ') { if (state.value === 'aiming') launchPass(); return; }
  const mate = getPlaymakerScene({ attempt: attempt.value, elapsedMs: elapsedMs.value }).teammates[event.key === 'ArrowRight' || event.key === 'ArrowDown' ? 1 : 0];
  drag.value = { x: mate.x - PLAYMAKER_START.x, y: mate.y - PLAYMAKER_START.y };
  state.value = 'aiming'; hint.value = '';
}
function launchPass() {
  if (state.value !== 'aiming' || !session.value) return;
  updateClock();
  if (!remainingMs.value) { finish(); return; }
  const pass = createPlaymakerPass({ attempt: attempt.value, elapsedMs: Math.floor(elapsedMs.value), drag: drag.value });
  if (!pass) { state.value = 'ready'; drag.value = { x: 0, y: 0 }; hint.value = 'Accompagna la palla verso uno dei due compagni.'; return; }
  activePass.value = pass;
  pendingPass.value = { sessionId: session.value.session_id, attempt: attempt.value, elapsed_ms: Math.floor(elapsedMs.value), dx: drag.value.x, dy: drag.value.y };
  flightProgress.value = 0; flightStartedAt = clockNow(); hint.value = ''; celebration.value = false;
  state.value = 'flying';
}
async function savePass() {
  if (disposed || receiptBusy || !pendingPass.value) return;
  receiptBusy = true; state.value = 'saving'; errorMessage.value = ''; emit('saving-change', true);
  const pending = { ...pendingPass.value };
  try {
    const response = await submitPlaymakerPass(gameEventId, pending.sessionId, { attempt: pending.attempt, elapsed_ms: pending.elapsed_ms, dx: pending.dx, dy: pending.dy });
    if (disposed) return;
    if (!response.ok) {
      if ([404, 410].includes(response.status)) {
        pendingPass.value = null; errorMessage.value = response.message || 'Il tempo per recuperare questa azione è terminato.'; finish();
      } else { state.value = 'error'; errorMessage.value = response.message || 'Connessione interrotta. Riprova a salvare la stessa azione: sarà conteggiata una sola volta.'; }
      return;
    }
    const receipt = response.data;
    if (receipt?.session_id !== pending.sessionId || receipt.attempt !== pending.attempt || typeof receipt.assisted !== 'boolean' || !Number.isInteger(receipt.wallet) || receipt.wallet < 0 || !Number.isInteger(receipt.playmaker_earned) || receipt.playmaker_earned < 0 || !Number.isInteger(receipt.next_attempt) || receipt.next_attempt <= pending.attempt) throw new Error('Invalid receipt');
    lastReceipt.value = receipt; confirmedEarned.value = Math.max(confirmedEarned.value, receipt.playmaker_earned);
    sessionAttempts.value += 1;
    if (receipt.assisted) { sessionAssists.value += 1; streak.value += 1; bestStreak.value = Math.max(bestStreak.value, streak.value); celebration.value = true; updateBest(); playAssistSound(); }
    else { streak.value = 0; hint.value = receipt.reason === 'intercepted' ? 'Intercettato. Cerca la corsia dall’altra parte.' : 'Passaggio lungo: mira verso il corpo del compagno.'; }
    emit('credited', { ...receipt, event_id: gameEventId });
    pendingPass.value = null; attempt.value = receipt.next_attempt;
    updateClock();
    if (Number.isFinite(receipt.remaining_ms)) {
      clockElapsedBase = Math.max(elapsedMs.value, 30000 - Math.max(0, Math.min(30000, receipt.remaining_ms)));
      clockStartedAt = clockNow(); updateClock();
    }
    state.value = 'result';
    if (!remainingMs.value) finish();
    else resultTimer = window.setTimeout(() => {
      if (disposed) return;
      celebration.value = false; activePass.value = null; drag.value = { x: 0, y: 0 }; updateClock();
      if (!remainingMs.value) finish(); else state.value = 'ready';
    }, receipt.assisted ? 500 : 350);
  } catch {
    if (!disposed) { state.value = 'error'; errorMessage.value = 'Non siamo riusciti a verificare l’azione. Riprova il salvataggio.'; }
  } finally { receiptBusy = false; if (!disposed) emit('saving-change', false); }
}
function retry() { if (pendingPass.value) void savePass(); else void startSession(); }
function toggleSound() {
  soundEnabled.value = !soundEnabled.value;
  if (!soundEnabled.value) return;
  try { const Audio = window.AudioContext || window.webkitAudioContext; if (!Audio) { soundEnabled.value = false; return; } audioContext ||= new Audio(); void audioContext.resume().catch(() => { soundEnabled.value = false; }); } catch { soundEnabled.value = false; }
}
function playAssistSound() {
  if (!soundEnabled.value || !audioContext) return;
  try {
    [523, 784, 1046].forEach((frequency, i) => {
      const tone = audioContext.createOscillator(); const gain = audioContext.createGain(); const at = audioContext.currentTime + i * .06;
      tone.frequency.value = frequency; gain.gain.setValueAtTime(0, at); gain.gain.linearRampToValueAtTime(.03, at + .012); gain.gain.exponentialRampToValueAtTime(.001, at + .22);
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
.playmaker { display: flex; flex-direction: column; width: 100%; height: 100%; min-height: 0; overflow: hidden; border-radius: 24px; background: #0b1822; color: #f1fbff; box-shadow: 0 12px 42px #0005; font-family: 'Inter', system-ui, sans-serif; }
.playmaker * { box-sizing: border-box; }
.pm-header { display: flex; align-items: center; gap: 9px; padding: 12px 14px; flex: none; background: linear-gradient(125deg, #203440, #10232e); border-bottom: 1px solid #dfffff12; }
.pm-brand { flex: 1; min-width: 0; }
.pm-brand > span { display: block; max-width: 170px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: #afc4cd; font-size: 8px; font-weight: 700; letter-spacing: .16em; text-transform: uppercase; }
.pm-brand b { display: block; margin-top: 3px; font-size: 20px; font-weight: 950; line-height: 1; letter-spacing: -.055em; }
.pm-brand b span { color: #d9ba7f; }
.pm-brand small { display: block; margin-top: 4px; color: #789eaa; font-size: 6px; font-weight: 800; letter-spacing: .22em; }
.pm-wallet { display: flex; align-items: center; gap: 6px; padding: 7px 10px; border-radius: 99px; background: #f7c25b18; color: #ffe0a0; font-size: 13px; font-weight: 800; font-variant-numeric: tabular-nums; }
.pm-coin { display: inline-block; width: 14px; height: 14px; flex: none; border-radius: 50%; background: radial-gradient(circle at 30% 25%, #ffecb8, #f3bd49 55%, #ac6b12); box-shadow: inset 0 0 0 2px #ffd980, 0 2px 4px #0003; }
.pm-sound { display: grid; place-items: center; width: 30px; height: 30px; padding: 6px; border: 1px solid #d5faff20; border-radius: 50%; background: #fff1; color: #d0e3eb; cursor: pointer; }
.pm-sound svg { width: 100%; height: 100%; }
.pm-clock { display: flex; align-items: center; gap: 9px; padding: 9px 15px; flex: none; background: #10232e; }
.pm-clock > span { min-width: 70px; color: #91b3bc; font-size: 7px; font-weight: 800; letter-spacing: .12em; }
.pm-clock > div { flex: 1; height: 4px; overflow: hidden; border-radius: 99px; background: #ffffff10; }
.pm-clock i { display: block; height: 100%; border-radius: inherit; background: linear-gradient(90deg, #72c8ba, #e1cb9c); }
.pm-clock > b { min-width: 25px; color: #f0e0bc; font-size: 14px; text-align: right; font-variant-numeric: tabular-nums; }
.pm-clock small { margin-left: 1px; font-size: 8px; color: #91b3bc; }
.clock-low > b { color: #ff9c91; }
.clock-low i { background: #ee9184; }
.pm-arena { position: relative; display: flex; flex: 1; min-height: 0; overflow: hidden; background: #0b1822; }
.pm-court { display: block; width: 100%; height: 100%; touch-action: none; user-select: none; -webkit-user-select: none; -webkit-tap-highlight-color: transparent; outline-offset: -3px; }
.pm-court:focus-visible { outline: 2px solid #b5ead9; }
.pm-stats { position: absolute; z-index: 2; top: 9px; left: 14px; right: 14px; display: flex; justify-content: space-between; pointer-events: none; }
.pm-stats > span { text-align: center; }
.pm-stats small { display: block; color: #93aeba; font-size: 6px; font-weight: 800; letter-spacing: .1em; }
.pm-stats b { display: flex; align-items: center; justify-content: center; gap: 4px; margin-top: 3px; color: #f8efdc; font-size: 22px; font-weight: 900; line-height: 1; font-variant-numeric: tabular-nums; }
.pm-stats i { font-size: 12px; font-style: normal; }
.pm-net { transform-origin: 0 0; }
.pm-net.net-hit { transform: scaleY(1.2) skewX(-4deg); }
.pm-cover { position: absolute; z-index: 4; inset: 0; display: flex; flex-direction: column; align-items: center; justify-content: safe center; gap: 14px; padding: 22px; overflow-y: auto; background: linear-gradient(180deg, #0b1a25a8, #0b1a25ee); backdrop-filter: blur(3px); text-align: center; }
.pm-cover > * { flex-shrink: 0; }
.pm-cover-eyebrow { color: #d3bb8e; font-size: 8px; font-weight: 800; letter-spacing: .18em; }
.pm-cover-icon { display: grid; place-items: center; width: 66px; height: 66px; border: 1px solid #d4bd8938; border-radius: 23px; background: linear-gradient(135deg, #d8be7f25, #bfe3d105); color: #efdbab; font-size: 37px; box-shadow: 0 5px 28px #0004; }
.pm-cover h2 { margin: 0; max-width: 260px; color: #fff8e9; font-size: clamp(21px, 7vw, 29px); font-weight: 900; line-height: 1.12; letter-spacing: -.04em; }
.pm-cover p { margin: 0; max-width: 270px; color: #b7cbd0; font-size: 11px; line-height: 1.65; }
.pm-cover > small { color: #829da7; font-size: 9px; }
.pm-primary { display: flex; align-items: center; justify-content: space-between; gap: 18px; width: 100%; max-width: 265px; padding: 14px 17px; border: 0; border-radius: 14px; background: linear-gradient(120deg, #f0d8a6, #cfac71); color: #332b21; font: inherit; font-size: 12px; font-weight: 850; box-shadow: 0 5px 18px #0003; cursor: pointer; }
.pm-primary span { font-size: 17px; }
.pm-primary:disabled { opacity: .6; cursor: wait; }
.pm-summary-stats { display: flex; gap: 20px; margin: 4px 0; }
.pm-summary-stats > span { display: flex; flex-direction: column; align-items: center; gap: 7px; }
.pm-summary-stats b { display: flex; align-items: center; gap: 5px; color: #f2e5ce; font-size: 29px; line-height: 1; font-weight: 900; }
.pm-summary-stats small { color: #91aab4; font-size: 6px; font-weight: 800; letter-spacing: .06em; }
.pm-summary-stats .pm-gold { color: #ffdc91; }
.pm-footer { display: flex; align-items: center; justify-content: space-between; gap: 10px; padding: 12px 14px; min-height: 65px; flex: none; background: linear-gradient(120deg, #1c323e, #10232f); border-top: 1px solid #d5f9ff10; }
.pm-footer > div { flex: 1; min-width: 0; }
.pm-footer b { display: block; font-size: 11px; font-weight: 800; }
.pm-footer div > span { display: block; margin-top: 4px; color: #9db5bf; font-size: 9px; line-height: 1.45; }
.pm-match { display: flex; align-items: center; justify-content: center; flex-wrap: wrap; gap: 4px; width: 65px; flex: none; color: #ffdf9c; font-size: 15px; font-weight: 850; font-variant-numeric: tabular-nums; }
.pm-match small { flex: 0 0 100%; text-align: center; color: #8fa9b4; font-size: 7px; font-weight: 600; }
.pm-retry { padding: 9px 11px; max-width: 112px; border: 0; border-radius: 11px; background: #d9c293; color: #272d2c; font: inherit; font-size: 10px; font-weight: 800; cursor: pointer; }
.pm-celebration { position: absolute; z-index: 3; top: 42%; left: 0; right: 0; display: flex; flex-direction: column; align-items: center; gap: 6px; pointer-events: none; text-shadow: 0 3px 18px #000; animation: pm-pop .2s ease-out; }
.pm-celebration b { color: #ffe1a0; font-size: clamp(21px, 7vw, 30px); font-weight: 950; letter-spacing: -.045em; }
.pm-celebration span { color: #e7fff0; font-size: 10px; font-weight: 700; }
.pm-reward { animation: pm-float .6s ease-out both; }
.pm-drag-hint { animation: pm-hint 1.8s ease-in-out infinite; }
.is-assist .pm-wallet { box-shadow: 0 0 18px #ffe09b30; }
@keyframes pm-pop { from { opacity: 0; transform: translateY(6px) scale(.95); } to { opacity: 1; transform: none; } }
@keyframes pm-float { from { opacity: 0; translate: 0 10px; } 30% { opacity: 1; } to { opacity: 0; translate: 0 -18px; } }
@keyframes pm-hint { 50% { opacity: .35; } }
@media (max-height: 650px) { .pm-header { padding: 9px 12px; } .pm-cover { gap: 10px; padding: 16px; } .pm-cover-icon { width: 45px; height: 45px; border-radius: 16px; font-size: 27px; } .pm-footer { min-height: 57px; padding: 9px 12px; } }
@media (prefers-reduced-motion: reduce) { .pm-reward, .pm-drag-hint, .pm-celebration { animation: none; } }
</style>
