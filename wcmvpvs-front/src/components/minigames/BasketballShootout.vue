<template>
  <section class="basket-rush" :class="{ 'is-scored': showCelebration }" aria-label="Basket Rush">
    <header class="basket-hud">
      <div><span class="basket-eyebrow">{{ clubName || 'COURTSIDE' }}</span><b>BASKET <em>RUSH</em></b></div>
      <div class="basket-wallet" aria-label="Saldo monete"><i class="basket-coin" aria-hidden="true"></i>{{ walletCoins }}</div>
      <button class="sound-toggle" type="button" :aria-pressed="soundEnabled" :aria-label="soundEnabled ? 'Disattiva audio' : 'Attiva audio'" @click="toggleSound">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true"><path d="M11 5 6 9H3v6h3l5 4Z" /><path v-if="soundEnabled" d="M15 8c3 2 3 6 0 8m3-11c5 4 5 10 0 14" /><path v-else d="m16 9 6 6m0-6-6 6" /></svg>
      </button>
    </header>

    <div class="basket-court">
      <div class="basket-stats" aria-live="off"><span><small>CANESTRI</small><b>{{ sessionMakes }}</b></span><span><small>DI FILA</small><b>{{ streak }}<i v-if="streak > 1" aria-hidden="true">🔥</i></b></span><span><small>MONETE PARTITA</small><b class="gold-value">{{ matchEarned }}<i class="basket-coin" aria-hidden="true"></i></b></span></div>
      <svg ref="courtEl" class="court-svg" viewBox="0 0 360 560" preserveAspectRatio="xMidYMid meet" tabindex="0" role="application" aria-label="Tiro a canestro: trascina la palla verso il ferro e rilascia. Con la tastiera, usa le frecce per mirare e Invio per tirare."
        @pointerdown="onPointerDown" @pointermove="onPointerMove" @pointerup="onPointerUp" @pointercancel="cancelAim" @lostpointercapture="onLostCapture" @keydown="onCourtKeydown">
        <defs>
          <radialGradient id="br-arena"><stop stop-color="#193c4a" /><stop offset="1" stop-color="#07131e" /></radialGradient>
          <linearGradient id="br-floor" x2="0" y2="1"><stop stop-color="#1d272e" /><stop offset="1" stop-color="#67472f" /></linearGradient>
          <linearGradient id="br-glass" x2="1" y2="1"><stop stop-color="#dbf8ff" stop-opacity=".22" /><stop offset=".5" stop-color="#8bd9ed" stop-opacity=".04" /><stop offset="1" stop-color="#dffaff" stop-opacity=".16" /></linearGradient>
          <radialGradient id="br-ball" cx=".31" cy=".25" r=".78"><stop stop-color="#ffd280" /><stop offset=".35" stop-color="#f79a36" /><stop offset=".75" stop-color="#c65114" /><stop offset="1" stop-color="#63260c" /></radialGradient>
          <linearGradient id="br-light"><stop stop-color="#8decff" stop-opacity="0" /><stop offset=".5" stop-color="#a1f3ff" stop-opacity=".17" /><stop offset="1" stop-color="#8decff" stop-opacity="0" /></linearGradient>
          <linearGradient id="br-arm" x2="1" y2="1"><stop stop-color="#edbc91" /><stop offset=".65" stop-color="#c48a65" /><stop offset="1" stop-color="#85563f" /></linearGradient>
          <radialGradient id="br-halo"><stop stop-color="#75f5df" stop-opacity=".3" /><stop offset="1" stop-color="#75f5df" stop-opacity="0" /></radialGradient>
          <pattern id="br-grain" width="6" height="6" patternUnits="userSpaceOnUse"><circle cx="1" cy="1" r=".6" fill="#54220f" opacity=".3" /><circle cx="4" cy="4" r=".45" fill="#ffe3a0" opacity=".4" /></pattern>
          <clipPath id="br-ball-clip"><circle r="27" /></clipPath>
        </defs>
        <rect width="360" height="560" fill="url(#br-arena)" />
        <path d="M-30 0h70l170 360H-30Zm420 0h-70L150 360h240Z" fill="url(#br-light)" />
        <g fill="#88d9e8" opacity=".18"><circle v-for="(seat, index) in crowdDots" :key="index" :cx="seat.x" :cy="seat.y" :r="seat.r" /></g>
        <path d="M0 325Q180 295 360 325v235H0Z" fill="url(#br-floor)" />
        <g fill="none" stroke="#e9cf9c" stroke-width="1" opacity=".13"><path v-for="x in floorLines" :key="x" :d="`M180 307 ${x} 560`" /><path d="M0 359h360M0 410h360M0 481h360" /></g>
        <g fill="none" stroke="#e9e0c3" stroke-width="1.5" opacity=".3"><path d="M120 322 75 449q105 62 210 0l-45-127M100 381q80 32 160 0M0 480q180 122 360 0" /><ellipse cx="180" cy="487" rx="126" ry="35" /></g>
        <text x="180" y="364" text-anchor="middle" fill="#effdff" opacity=".12" font-size="17" font-weight="900" letter-spacing="8">BASKET RUSH</text>

        <g :transform="`translate(${hoop.x}, ${hoop.y})`" class="hoop-assembly">
          <ellipse cy="30" rx="70" ry="55" fill="url(#br-halo)" />
          <path d="M-5-130h10v79H-5Z" fill="#223a47" /><path d="M0-130v78" stroke="#82acb8" stroke-opacity=".3" />
          <rect x="-57" y="-70" width="114" height="74" rx="7" fill="#000" opacity=".25" transform="translate(0 4)" />
          <rect x="-57" y="-70" width="114" height="74" rx="7" fill="url(#br-glass)" stroke="#c7eff6" stroke-opacity=".65" stroke-width="2" />
          <path d="M-50-62H43L-50-7Z" fill="#fff" opacity=".04" /><rect x="-22" y="-36" width="44" height="30" rx="1" fill="none" stroke="#fff" stroke-opacity=".8" stroke-width="2" />
          <image v-if="clubLogoUrl" :href="clubLogoUrl" x="-48" y="-56" width="18" height="18" preserveAspectRatio="xMidYMid meet" />
          <rect x="-60" y="1" width="120" height="5" rx="2.5" fill="var(--red, #ef613b)" />
          <ellipse cy="11" rx="29" ry="8" fill="#091722" opacity=".7" />
          <path d="M-28 0q28-15 56 0" fill="none" stroke="#ffb24e" stroke-width="5" />
        </g>

        <g v-if="state === 'aiming' && previewShot" class="aim-guide" aria-hidden="true">
          <circle v-for="(point, index) in guidePoints" :key="index" :cx="point.x" :cy="point.y" :r="2.2 - index * .06" fill="#b9fff1" :opacity=".7 - index * .035" />
          <g :transform="`translate(${previewShot.aim.x}, ${previewShot.aim.y})`"><circle r="15" fill="#a9fff1" fill-opacity=".07" stroke="#b9fff1" stroke-opacity=".65" stroke-dasharray="3 4" /><path d="M-5 0H5M0-5V5" stroke="#b9fff1" stroke-width="1.5" /></g>
        </g>
        <ellipse :cx="ballPose.x" cy="505" :rx="29 * (ballPose.shadowScale || 1)" :ry="7 * (ballPose.shadowScale || 1)" fill="#000" :opacity="state === 'flying' ? .15 : .35" />

        <g v-if="state === 'ready' || state === 'aiming'" :transform="`translate(${heldOffset.x}, ${heldOffset.y})`" aria-hidden="true">
          <path d="M101 560q15-42 30-66l20-28q5-6 9-2t-2 12l-11 19q9-10 14-7t0 12l-18 31-2 29Z" fill="url(#br-arm)" />
          <path d="M259 560q-15-42-30-66l-20-28q-5-6-9-2t2 12l11 19q-9-10-14-7t0 12l18 31 2 29Z" fill="url(#br-arm)" />
          <path d="m98 552 47-6 1 14H97Zm164 0-47-6-1 14h49Z" fill="var(--red, #f26739)" />
        </g>
        <g v-if="ballPose.visible" :transform="`translate(${ballPose.x}, ${ballPose.y}) scale(${ballPose.scale}) rotate(${ballPose.rotation})`" class="basket-ball" aria-hidden="true">
          <circle r="29" fill="#ffb55f" opacity=".08" /><circle r="27" fill="url(#br-ball)" />
          <circle r="26.7" fill="url(#br-grain)" />
          <g clip-path="url(#br-ball-clip)" fill="none" stroke="#4b230f" stroke-width="1.8"><path d="M-27-2Q0 5 27-2M-3-27Q-9 0 3 27M-24-18Q8-9-22 20M22-20Q-8-8 24 18" /></g>
          <path d="M-16-15q7-7 15-6" fill="none" stroke="#fff2c9" stroke-width="2.5" stroke-linecap="round" opacity=".45" />
        </g>

        <g :transform="`translate(${hoop.x}, ${hoop.y})`" aria-hidden="true">
          <g class="basket-net" :class="{ 'net-hit': netHit }" fill="none" stroke="#e9faff" stroke-width="1.2" stroke-opacity=".7">
            <path d="M-27 2-17 42Q0 49 17 42L27 2M-18 6-11 44M-9 8-5 46M0 8v39M9 8 5 46M18 6l-7 38M-24 14q24 13 48 0M-21 27q21 11 42 0M-18 39q18 9 36 0" />
            <path d="m-27 2 16 42M-18 6 13 43M0 8 22 26M27 2 11 44M18 6-13 43M0 8-22 26" stroke-opacity=".35" />
          </g>
          <path d="M-28 0q28 17 56 0" fill="none" stroke="#f97832" stroke-width="5" stroke-linecap="round" /><path d="M-24 3q24 12 48 0" fill="none" stroke="#ffc87b" stroke-width="1.2" opacity=".7" />
          <g v-if="showCelebration" class="basket-burst"><path v-for="angle in burstAngles" :key="angle" :transform="`rotate(${angle})`" d="M0-38v-10" stroke="#ffe59a" stroke-width="2.5" stroke-linecap="round" /></g>
        </g>

        <g v-if="showCelebration" class="court-reward" :transform="`translate(${hoop.x}, ${hoop.y - 50})`" aria-hidden="true"><text text-anchor="middle" fill="#ffe18b" font-size="32" font-weight="900">+1</text><circle cx="28" cy="-12" r="10" fill="#edac30" stroke="#ffe2a1" stroke-width="2" /><path d="M28-18v12" stroke="#a96a0a" stroke-width="2" /></g>
        <g v-if="state === 'ready' && !sessionAttempts" class="drag-hint" aria-hidden="true"><path d="M180 410v-50m-8 9 8-9 8 9" fill="none" stroke="#d1fff5" stroke-opacity=".7" stroke-width="2" stroke-linecap="round" /><circle cx="180" cy="466" r="43" fill="none" stroke="#d1fff5" stroke-opacity=".25" stroke-dasharray="3 8" /></g>
      </svg>

      <div v-if="state === 'loading'" class="court-cover"><span class="basket-loader"></span><b>Prepariamo il pallone…</b></div>
      <div v-if="state === 'aiming'" class="aim-meter" aria-hidden="true"><span>MIRA</span><i><b :style="{ width: `${aimPower}%` }"></b></i><span>RILASCIA</span></div>
      <div v-if="showCelebration" class="basket-feedback"><b>{{ lastReceipt?.clean ? 'SOLO RETE!' : 'CANESTRO!' }}</b><span>{{ streak > 1 ? `${streak} di fila · ` : '' }}1 moneta salvata</span></div>
    </div>

    <footer class="basket-controls">
      <div class="basket-status" role="status" aria-live="polite"><b>{{ statusTitle }}</b><span>{{ statusDetail }}</span></div>
      <button v-if="errorMessage" class="basket-retry" type="button" @click="retry">{{ pendingThrow ? 'Salva il tiro' : 'Ricarica il pallone' }}</button>
      <span v-else class="basket-rule"><i class="basket-coin" aria-hidden="true"></i>1 canestro = 1 moneta</span>
    </footer>
  </section>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { prepareBasketballShot, submitBasketballShot } from '../../api';
import { BASKETBALL_WORLD, BASKETBALL_START, BASKETBALL_HOOP, createBasketballShot, sampleBasketballShot } from '../../games/basketballShot';

const props = defineProps({ eventId: { type: Number, required: true }, walletCoins: { type: Number, default: 0 }, earnedCoins: { type: Number, default: 0 }, clubName: { type: String, default: '' }, clubLogoUrl: { type: String, default: '' } });
const emit = defineEmits(['credited', 'saving-change']);
const gameEventId = props.eventId;
const state = ref('loading');
const courtEl = ref(null);
const challenge = ref(null);
const hoop = ref({ ...BASKETBALL_HOOP });
const drag = ref({ x: 0, y: 0 });
const activeShot = ref(null);
const pendingThrow = ref(null);
const flightProgress = ref(0);
const sessionMakes = ref(0);
const sessionAttempts = ref(0);
const streak = ref(0);
const matchEarned = ref(props.earnedCoins || 0);
const lastReceipt = ref(null);
const errorMessage = ref('');
const hint = ref('');
const showCelebration = ref(false);
const netHit = ref(false);
const soundEnabled = ref(false);
const crowdDots = Array.from({ length: 84 }, (_, index) => ({ x: (index % 14) * 28 - 3, y: 95 + Math.floor(index / 14) * 32, r: 1 + index % 3 * .4 }));
const floorLines = [-240, -120, 0, 90, 180, 270, 360, 480, 600];
const burstAngles = [0, 45, 90, 135, 180, 225, 270, 315];
let pointerId = null;
let pointerStart = null;
let frameId;
let resultTimer;
let resolveFlight;
let disposed = false;
let audioContext;
let receiptBusy = false;

const previewShot = computed(() => state.value === 'aiming' ? createBasketballShot({ hoop: hoop.value, drag: drag.value }) : null);
const guidePoints = computed(() => previewShot.value ? Array.from({ length: 13 }, (_, index) => sampleBasketballShot(previewShot.value, index / 12 * .72)) : []);
const aimPower = computed(() => Math.min(100, Math.hypot(drag.value.x, drag.value.y) / 320 * 100));
const heldOffset = computed(() => state.value === 'aiming' ? { x: drag.value.x * .09, y: drag.value.y * .09 } : { x: 0, y: 0 });
const ballPose = computed(() => activeShot.value && ['flying', 'saving', 'result', 'error'].includes(state.value)
  ? sampleBasketballShot(activeShot.value, flightProgress.value)
  : { x: BASKETBALL_START.x + heldOffset.value.x, y: BASKETBALL_START.y + heldOffset.value.y, scale: 1, rotation: 0, visible: true, shadowScale: 1 });
const statusTitle = computed(() => errorMessage.value ? pendingThrow.value ? 'Il tiro resta da verificare' : 'Rientra in campo' : state.value === 'loading' ? 'Entra in campo' : state.value === 'aiming' ? 'Rilascia per tirare' : state.value === 'flying' ? 'Palla in volo…' : state.value === 'saving' ? 'Verifichiamo il tiro…' : showCelebration.value ? 'Bel canestro!' : 'Prendi la palla e mira al ferro');
const statusDetail = computed(() => errorMessage.value || hint.value || (state.value === 'ready' ? 'Trascina verso il canestro, poi rilascia.' : state.value === 'saving' ? 'Attendi la conferma delle monete.' : 'Ogni canestro riuscito vale una moneta.'));

function courtPoint(event) {
  const rect = courtEl.value?.getBoundingClientRect();
  if (!rect || !rect.width || !rect.height) return null;
  const scale = Math.min(rect.width / BASKETBALL_WORLD.width, rect.height / BASKETBALL_WORLD.height);
  return { x: (event.clientX - rect.left - (rect.width - BASKETBALL_WORLD.width * scale) / 2) / scale, y: (event.clientY - rect.top - (rect.height - BASKETBALL_WORLD.height * scale) / 2) / scale };
}
function onPointerDown(event) {
  if (state.value !== 'ready' || !challenge.value || event.isPrimary === false || (event.button != null && event.button !== 0)) return;
  const expiresAt = Date.parse(challenge.value.expires_at || '');
  if (Number.isFinite(expiresAt) && expiresAt < Date.now() + 10000) { void prepare(); return; }
  const point = courtPoint(event);
  if (!point || Math.hypot(point.x - BASKETBALL_START.x, point.y - BASKETBALL_START.y) > 60) return;
  event.preventDefault();
  courtEl.value?.focus({ preventScroll: true });
  pointerId = event.pointerId;
  pointerStart = point;
  drag.value = { x: 0, y: 0 };
  hint.value = '';
  state.value = 'aiming';
  try { courtEl.value?.setPointerCapture(pointerId); } catch { /* il tiro resta utilizzabile senza capture */ }
}
function onPointerMove(event) {
  if (state.value !== 'aiming' || event.pointerId !== pointerId || !pointerStart) return;
  const point = courtPoint(event);
  if (!point) return;
  drag.value = { x: Math.max(-400, Math.min(400, point.x - pointerStart.x)), y: Math.max(-560, Math.min(100, point.y - pointerStart.y)) };
}
function releasePointer() {
  const previousId = pointerId;
  pointerId = null;
  pointerStart = null;
  if (previousId != null) { try { courtEl.value?.releasePointerCapture(previousId); } catch { /* capture già rilasciata */ } }
}
function onPointerUp(event) {
  if (state.value !== 'aiming' || event.pointerId !== pointerId) return;
  onPointerMove(event);
  releasePointer();
  void shoot();
}
function cancelAim(event) {
  if (pointerId != null && event?.pointerId != null && event.pointerId !== pointerId) return;
  releasePointer();
  if (state.value === 'aiming') { state.value = 'ready'; drag.value = { x: 0, y: 0 }; }
}
function onLostCapture(event) { if (pointerId === event.pointerId) cancelAim(event); }
function onCourtKeydown(event) {
  if (!['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Enter', ' ', 'Escape'].includes(event.key)) return;
  if (event.key === 'Escape') { if (state.value === 'aiming') { event.preventDefault(); event.stopPropagation(); cancelAim(); } return; }
  if (pointerId != null || !['ready', 'aiming'].includes(state.value)) return;
  event.preventDefault();
  if (event.key === 'Enter' || event.key === ' ') { if (state.value === 'aiming') void shoot(); return; }
  if (state.value === 'ready') { drag.value = { x: 0, y: -180 }; state.value = 'aiming'; }
  const offsets = { ArrowLeft: [-8, 0], ArrowRight: [8, 0], ArrowUp: [0, -8], ArrowDown: [0, 8] };
  const offset = offsets[event.key];
  drag.value = { x: Math.max(-400, Math.min(400, drag.value.x + offset[0])), y: Math.max(-560, Math.min(0, drag.value.y + offset[1])) };
}

async function prepare() {
  if (disposed) return;
  state.value = 'loading';
  errorMessage.value = '';
  const response = await prepareBasketballShot(gameEventId);
  if (disposed) return;
  if (!response.ok || !response.data?.shot_id) {
    state.value = 'error';
    errorMessage.value = response.message || 'Campo non disponibile. Controlla la connessione e riprova.';
    return;
  }
  challenge.value = response.data;
  hoop.value = response.data.hoop || { ...BASKETBALL_HOOP };
  activeShot.value = null;
  flightProgress.value = 0;
  state.value = 'ready';
}
function animateShot() {
  return new Promise((resolve) => {
    resolveFlight = resolve;
    let start;
    const frame = (timestamp) => {
      if (disposed) { resolve(); return; }
      start ??= timestamp;
      flightProgress.value = Math.min(1, (timestamp - start) / activeShot.value.flightMs);
      netHit.value = activeShot.value.made && flightProgress.value >= .74 && flightProgress.value < .99;
      if (flightProgress.value < 1) frameId = window.requestAnimationFrame(frame);
      else { netHit.value = false; resolveFlight = null; resolve(); }
    };
    frameId = window.requestAnimationFrame(frame);
  });
}
async function shoot() {
  if (state.value !== 'aiming' || !challenge.value) return;
  const expiresAt = Date.parse(challenge.value.expires_at || '');
  if (Number.isFinite(expiresAt) && expiresAt < Date.now() + 10000) {
    hint.value = 'Nuovo pallone pronto: mira di nuovo al canestro.';
    await prepare();
    return;
  }
  const shot = createBasketballShot({ hoop: hoop.value, drag: drag.value });
  if (!shot) { state.value = 'ready'; hint.value = 'Accompagna la palla più in alto, verso il canestro.'; return; }
  activeShot.value = shot;
  pendingThrow.value = { shotId: challenge.value.shot_id, dx: drag.value.x, dy: drag.value.y };
  state.value = 'flying';
  hint.value = '';
  showCelebration.value = false;
  await animateShot();
  if (!disposed) await saveThrow();
}
async function saveThrow() {
  if (disposed || receiptBusy || !pendingThrow.value) return;
  receiptBusy = true;
  state.value = 'saving';
  errorMessage.value = '';
  emit('saving-change', true);
  const attempt = { ...pendingThrow.value };
  try {
    const response = await submitBasketballShot(gameEventId, attempt.shotId, { dx: attempt.dx, dy: attempt.dy });
    if (disposed) return;
    if (!response.ok) {
      state.value = 'error';
      if ([404, 410].includes(response.status)) { pendingThrow.value = null; challenge.value = null; }
      errorMessage.value = response.message || 'Connessione interrotta. Riprova a salvare questo tiro: verrà conteggiato una sola volta.';
      return;
    }
    const receipt = response.data;
    lastReceipt.value = receipt;
    matchEarned.value = Math.max(matchEarned.value, Number(receipt.basketball_earned) || 0);
    sessionAttempts.value += 1;
    if (receipt.made) { sessionMakes.value += 1; streak.value += 1; showCelebration.value = true; playSwish(); }
    else { streak.value = 0; hint.value = activeShot.value?.missKind === 'short' ? 'Tiro corto: trascina un po’ più in alto.' : activeShot.value?.missKind === 'long' ? 'Tiro lungo: accorcia il trascinamento.' : 'Aggiusta la direzione e riprova.'; }
    emit('credited', { ...receipt, event_id: gameEventId });
    pendingThrow.value = null;
    challenge.value = null;
    state.value = 'result';
    resultTimer = window.setTimeout(() => { showCelebration.value = false; void prepare(); }, receipt.made ? 900 : 600);
  } catch {
    if (!disposed) { state.value = 'error'; errorMessage.value = 'Non siamo riusciti a verificare il tiro. Riprova il salvataggio.'; }
  } finally { receiptBusy = false; if (!disposed) emit('saving-change', false); }
}
function retry() { if (pendingThrow.value) void saveThrow(); else void prepare(); }
function toggleSound() {
  soundEnabled.value = !soundEnabled.value;
  if (soundEnabled.value) { try { const Audio = window.AudioContext || window.webkitAudioContext; if (Audio) { audioContext ||= new Audio(); void audioContext.resume(); } } catch { soundEnabled.value = false; } }
}
function playSwish() {
  if (!soundEnabled.value || !audioContext) return;
  try {
    [660, 880, 1320].forEach((frequency, index) => {
      const tone = audioContext.createOscillator();
      const gain = audioContext.createGain();
      const at = audioContext.currentTime + index * .055;
      tone.type = 'sine'; tone.frequency.value = frequency;
      gain.gain.setValueAtTime(0, at); gain.gain.linearRampToValueAtTime(.035, at + .012); gain.gain.exponentialRampToValueAtTime(.001, at + .2);
      tone.connect(gain); gain.connect(audioContext.destination); tone.start(at); tone.stop(at + .21);
    });
  } catch { /* l’audio è opzionale */ }
}
onMounted(prepare);
onBeforeUnmount(() => {
  disposed = true;
  window.cancelAnimationFrame(frameId);
  window.clearTimeout(resultTimer);
  resolveFlight?.();
  releasePointer();
  emit('saving-change', false);
  if (audioContext) void audioContext.close();
});
</script>

<style scoped>
.basket-rush { display: flex; flex-direction: column; width: 100%; height: 100%; min-height: 0; overflow: hidden; border-radius: 24px; background: #08141f; color: #effaff; font-family: 'Inter', system-ui, sans-serif; box-shadow: 0 12px 45px #0005; }
.basket-rush * { box-sizing: border-box; }
.basket-hud { display: flex; align-items: center; gap: 9px; padding: 13px 14px 11px; flex: none; background: linear-gradient(120deg, #19303e, #0d202d); border-bottom: 1px solid #d3ffff12; }
.basket-hud > div:first-child { flex: 1; min-width: 0; }
.basket-eyebrow { display: block; max-width: 150px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: #a3bbc7; font-size: 8px; font-weight: 700; letter-spacing: .17em; text-transform: uppercase; }
.basket-hud b { display: block; margin-top: 3px; font-size: 17px; line-height: 1; font-weight: 900; letter-spacing: -.04em; }
.basket-hud em { color: #83f5db; font-style: normal; }
.basket-wallet { display: flex; align-items: center; gap: 6px; padding: 7px 10px; border-radius: 999px; background: #f4be4620; color: #ffdf98; font-size: 13px; font-weight: 800; font-variant-numeric: tabular-nums; }
.basket-coin { display: inline-block; width: 14px; height: 14px; flex: none; border-radius: 50%; background: radial-gradient(circle at 30% 25%, #ffeab0, #f6bd43 55%, #b76f08); box-shadow: inset 0 0 0 2px #ffd26d, 0 1px 4px #0004; }
.sound-toggle { display: grid; place-items: center; width: 30px; height: 30px; padding: 6px; border: 1px solid #d5faff20; border-radius: 50%; background: #ffffff08; color: #c0d7e0; cursor: pointer; }
.sound-toggle svg { width: 100%; height: 100%; }
.basket-court { position: relative; display: flex; flex: 1; min-height: 0; overflow: hidden; background: radial-gradient(ellipse at center, #183443, #07131e 75%); }
.court-svg { display: block; width: 100%; height: 100%; touch-action: none; user-select: none; -webkit-user-select: none; -webkit-tap-highlight-color: transparent; outline-offset: -3px; }
.court-svg:focus-visible { outline: 2px solid #91f6df; }
.basket-stats { position: absolute; z-index: 2; top: 11px; left: 12px; right: 12px; display: flex; justify-content: space-between; gap: 6px; pointer-events: none; }
.basket-stats > span { display: flex; flex-direction: column; gap: 3px; text-align: center; }
.basket-stats small { color: #91aebc; font-size: 7px; font-weight: 800; letter-spacing: .09em; }
.basket-stats b { display: flex; align-items: center; justify-content: center; gap: 5px; font-size: 21px; font-weight: 900; line-height: 1; font-variant-numeric: tabular-nums; text-shadow: 0 2px 12px #0008; }
.basket-stats b i:not(.basket-coin) { font-size: 13px; font-style: normal; }
.gold-value { color: #ffdc8f; }
.hoop-assembly { transition: transform .3s ease; }
.basket-net { transform-origin: 0 0; transition: transform .12s; }
.basket-net.net-hit { transform: scaleY(1.25) skewX(-5deg); }
.drag-hint { animation: hint-rise 1.5s ease-in-out infinite; pointer-events: none; }
.basket-burst { animation: burst-flash .65s ease-out both; }
.court-reward { animation: reward-float .9s ease-out both; }
.basket-feedback { position: absolute; z-index: 3; top: 42%; left: 0; right: 0; display: flex; flex-direction: column; align-items: center; gap: 6px; pointer-events: none; animation: feedback-pop .25s ease-out; text-shadow: 0 3px 20px #000; }
.basket-feedback b { color: #ffdf94; font-size: clamp(24px, 8vw, 35px); line-height: 1; font-weight: 900; letter-spacing: -.045em; }
.basket-feedback span { color: #e9fff5; font-size: 11px; font-weight: 700; }
.court-cover { position: absolute; inset: 0; display: flex; align-items: center; justify-content: center; flex-direction: column; gap: 13px; background: #091722b8; backdrop-filter: blur(3px); font-size: 12px; }
.basket-loader { width: 26px; height: 26px; border: 2px solid #b6fff023; border-top-color: #83f5db; border-radius: 50%; animation: loader-turn .75s linear infinite; }
.aim-meter { position: absolute; bottom: 9px; left: 18%; right: 18%; display: flex; align-items: center; gap: 7px; color: #c7ece8; font-size: 7px; font-weight: 800; letter-spacing: .09em; pointer-events: none; }
.aim-meter > i { flex: 1; height: 4px; overflow: hidden; border-radius: 99px; background: #fff2; }
.aim-meter b { display: block; height: 100%; background: linear-gradient(90deg, #74efd3, #f9d67e); border-radius: inherit; }
.basket-controls { display: flex; align-items: center; justify-content: space-between; gap: 10px; min-height: 68px; padding: 12px 14px; flex: none; background: linear-gradient(120deg, #172b37, #0c1c29); border-top: 1px solid #d3ffff12; }
.basket-status { flex: 1; min-width: 0; }
.basket-status b { display: block; color: #f0fbff; font-size: 12px; font-weight: 800; }
.basket-status span { display: block; margin-top: 4px; color: #9db5c0; font-size: 10px; line-height: 1.45; }
.basket-rule { display: flex; align-items: center; gap: 5px; max-width: 100px; color: #f9d585; font-size: 9px; line-height: 1.45; font-weight: 700; }
.basket-retry { flex: none; max-width: 120px; padding: 9px 10px; border: 0; border-radius: 12px; background: #a4f8e1; color: #102722; font: inherit; font-size: 10px; font-weight: 800; cursor: pointer; }
.is-scored .basket-wallet { box-shadow: 0 0 18px #ffcf6030; }
@keyframes hint-rise { 50% { opacity: .4; } }
@keyframes reward-float { from { opacity: 0; translate: 0 10px; } 25% { opacity: 1; } to { opacity: 0; translate: 0 -25px; } }
@keyframes burst-flash { from { opacity: 1; scale: .6; } to { opacity: 0; scale: 1.4; } }
@keyframes feedback-pop { from { opacity: 0; transform: translateY(6px) scale(.9); } to { opacity: 1; transform: none; } }
@keyframes loader-turn { to { transform: rotate(360deg); } }
@media (prefers-reduced-motion: reduce) { .drag-hint, .court-reward, .basket-burst, .basket-feedback { animation: none; } .hoop-assembly, .basket-net { transition: none; } }
</style>
