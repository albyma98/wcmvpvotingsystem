<template>
  <div class="tap-challenge flex h-full min-h-0 w-full flex-col">
    <header class="rounded-2xl border border-white/15 bg-slate-900/60 p-3 text-slate-100 backdrop-blur">
      <div class="flex items-center justify-between gap-2 text-sm font-semibold">
        <p>Tempo: {{ timeLabel }}</p>
        <p>Tap: {{ tapCount }}</p>
      </div>
      <p class="mt-1 text-xs uppercase tracking-[0.2em] text-slate-300">{{ statusLabel }}</p>
    </header>

    <main ref="gameAreaRef" class="game-area mt-4 flex-1 rounded-2xl border border-white/20 bg-slate-900/70" @touchmove.prevent>
      <button v-if="status === 'playing'" ref="ballRef" type="button" class="ball" :style="ballStyle" aria-label="Tappa la palla" @click="onTap"><span>🏐</span></button>

      <div v-else class="overlay">
        <p class="text-xs uppercase tracking-[0.2em] text-slate-300">{{ statusLabel }}</p>
        <h3 class="mt-2 text-2xl font-black text-white">{{ headline }}</h3>
        <p class="mt-2 text-sm text-slate-200">{{ subline }}</p>
        <p v-if="status === 'finished' || liveStep === 'result'" class="mt-2 text-sm text-slate-200">Totale tap validi: {{ tapCount }}</p>
      </div>
    </main>

    <p v-if="errorMessage" class="mt-3 rounded-xl border border-red-300/40 bg-red-500/10 px-3 py-2 text-sm text-red-200">{{ errorMessage }}</p>

    <footer class="mt-4 grid grid-cols-1 gap-2 sm:grid-cols-2">
      <button type="button" class="cta-primary" :disabled="isPrimaryDisabled" @click="onPrimaryAction">{{ primaryLabel }}</button>
      <button type="button" class="cta-secondary" @click="emit('exit')">Esci</button>
    </footer>

    <div v-if="showLiveButton" class="mt-2">
      <button class="cta-secondary w-full" type="button" :disabled="liveBusy" @click="startLive">Sfida live</button>
    </div>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, ref } from 'vue';
import { awardTapChallengeCoins } from '../../services/coins';
import { abortTapLiveMatch, buildTapLiveSseUrl, cancelTapLiveQueue, fetchTapLiveResult, fetchTapLiveState, getFanSessionToken, joinTapLiveQueue, submitTapLiveScore } from '../../api';

const props = defineProps({ eventId: { type: Number, default: 0 }, cooldownSeconds: { type: Number, default: 60 } });
const emit = defineEmits(['claim', 'exit']);
const ROUND_DURATION_MS = 10_000; const MATCHMAKING_TIMEOUT_MS = 20_000; const TICK_MS = 100; const MIN_DISTANCE_PX = 30; const BASE_MOVE_INTERVAL_MS = 850; const MIN_MOVE_INTERVAL_MS = 220; const SPEED_GAIN_PER_TAP = 0.92;
const status = ref('ready'); const timeLeftMs = ref(ROUND_DURATION_MS); const tapCount = ref(0); const cooldownUntil = ref(0); const errorMessage = ref(''); const isSubmitting = ref(false); const claimRequested = ref(false); const nowTs = ref(Date.now());
const gameAreaRef = ref(null); const ballRef = ref(null); const ballX = ref(0); const ballY = ref(0); const lastBallX = ref(0); const lastBallY = ref(0); const moveIntervalMs = ref(BASE_MOVE_INTERVAL_MS);
const liveStep = ref('idle'); const liveBusy = ref(false); const liveMatchId = ref(''); const liveOutcome = ref(''); const liveCoins = ref(0); const liveMessage = ref(''); const matchmakingDeadline = ref(0); let sse;
let timerId; let moveTimerId; let gameEndsAt = 0; let cooldownTickId; let matchmakingTimerId;
const earnedCoins = computed(() => tapCount.value); const isCooldownActive = computed(() => cooldownUntil.value > nowTs.value); const cooldownRemainingMs = computed(() => Math.max(0, cooldownUntil.value - nowTs.value));
const statusLabel = computed(() => liveStep.value !== 'idle' ? `LIVE ${liveStep.value.toUpperCase()}` : status.value === 'playing' ? 'PLAYING' : status.value === 'finished' ? 'FINISHED' : 'READY');
const timeLabel = computed(() => `${(Math.max(0, timeLeftMs.value) / 1000).toFixed(1)}s`);
const showLiveButton = computed(() => status.value === 'ready' && liveStep.value === 'idle');
const headline = computed(() => {
  if (liveStep.value === 'searching') return 'Cerchiamo un avversario…';
  if (liveStep.value === 'countdown') return 'Avversario trovato';
  if (liveStep.value === 'result') return liveMessage.value || 'Tempo scaduto';
  if (status.value === 'ready') return 'Tap Challenge';
  if (status.value === 'finished') return `Hai guadagnato ${earnedCoins.value} monete`;
  return 'Tap Challenge';
});
const subline = computed(() => {
  if (liveStep.value === 'searching') return matchmakingDeadline.value > nowTs.value ? `Attendi ${formatSeconds(Math.max(0, matchmakingDeadline.value - nowTs.value))} o annulla.` : 'Attendi qualche secondo o annulla.';
  if (liveStep.value === 'countdown') return 'La sfida inizia tra…';
  if (liveStep.value === 'result') return `Hai guadagnato ${liveCoins.value} monete`;
  return status.value === 'ready' ? 'Tappa la palla più volte possibile in 10 secondi.' : 'Tappa la palla più volte possibile';
});
const primaryLabel = computed(() => {
  if (liveStep.value === 'searching') return 'Annulla ricerca';
  if (liveStep.value === 'countdown' || status.value === 'playing') return 'In corso…';
  if (liveStep.value === 'result') return 'Riprova';
  if (isSubmitting.value || claimRequested.value) return 'Accredito…';
  if (status.value === 'finished' && errorMessage.value) return 'Riprova accredito';
  if (status.value === 'finished') return 'Riscatta monete';
  if (isCooldownActive.value) return `In cooldown ${formatCooldown(cooldownRemainingMs.value)}`;
  return 'Inizia';
});
const isPrimaryDisabled = computed(() => liveStep.value === 'countdown' || status.value === 'playing' || isSubmitting.value || claimRequested.value || (status.value === 'ready' && isCooldownActive.value));
const ballStyle = computed(() => ({ transform: `translate3d(${ballX.value}px, ${ballY.value}px, 0)` }));

onBeforeUnmount(async () => { stopTimer(); stopBallMovement(); stopMatchmakingTimer(); sse?.close(); if (liveMatchId.value && liveStep.value !== 'result') await abortTapLiveMatch(props.eventId, liveMatchId.value); if (cooldownTickId && typeof window !== 'undefined') window.clearInterval(cooldownTickId); });
if (typeof window !== 'undefined') cooldownTickId = window.setInterval(() => { nowTs.value = Date.now(); }, 250);

function stopTimer() { if (!timerId || typeof window === 'undefined') return; window.clearInterval(timerId); timerId = undefined; }
function stopBallMovement() { if (!moveTimerId || typeof window === 'undefined') return; window.clearTimeout(moveTimerId); moveTimerId = undefined; }
function stopMatchmakingTimer() { if (!matchmakingTimerId || typeof window === 'undefined') return; window.clearTimeout(matchmakingTimerId); matchmakingTimerId = undefined; }
function scheduleBallMovement() { stopBallMovement(); if (typeof window === 'undefined' || status.value !== 'playing') return; moveTimerId = window.setTimeout(() => { if (status.value !== 'playing') return; repositionBall(); scheduleBallMovement(); }, moveIntervalMs.value); }
function onPrimaryAction() { if (liveStep.value === 'searching') { cancelLiveSearch(); return; } if (liveStep.value === 'result') { resetLive(); return; } if (status.value === 'playing' || isSubmitting.value || claimRequested.value) return; if (status.value === 'finished') { claimReward(); return; } if (isCooldownActive.value) return; startGame(); }
async function startGame() { stopTimer(); stopBallMovement(); errorMessage.value=''; claimRequested.value=false; status.value='playing'; tapCount.value=0; timeLeftMs.value=ROUND_DURATION_MS; moveIntervalMs.value=BASE_MOVE_INTERVAL_MS; gameEndsAt=Date.now()+ROUND_DURATION_MS; await nextTick(); repositionBall(true); scheduleBallMovement(); if (typeof window === 'undefined') return; timerId=window.setInterval(() => { const remaining=gameEndsAt-Date.now(); timeLeftMs.value=Math.max(0,remaining); if (remaining<=0) finishGame(); }, TICK_MS); }
function repositionBall(force=false){ const area=gameAreaRef.value; const ball=ballRef.value; if(!area||!ball) return; const areaRect=area.getBoundingClientRect(); const ballRect=ball.getBoundingClientRect(); const maxX=Math.max(0,areaRect.width-ballRect.width); const maxY=Math.max(0,areaRect.height-ballRect.height); let nextX=0,nextY=0,attempts=0; do { nextX=clamp(Math.random()*maxX,0,maxX); nextY=clamp(Math.random()*maxY,0,maxY); attempts+=1; } while(!force && attempts<8 && distance(nextX,nextY,lastBallX.value,lastBallY.value)<MIN_DISTANCE_PX); ballX.value=nextX; ballY.value=nextY; lastBallX.value=nextX; lastBallY.value=nextY; }
function onTap(){ if(status.value!=='playing') return; tapCount.value+=1; moveIntervalMs.value=Math.max(MIN_MOVE_INTERVAL_MS,moveIntervalMs.value*SPEED_GAIN_PER_TAP); navigator?.vibrate?.(10); repositionBall(); scheduleBallMovement(); }
async function finishGame(){ if(status.value!=='playing') return; stopTimer(); stopBallMovement(); status.value='finished'; timeLeftMs.value=0; if (liveStep.value === 'playing' && liveMatchId.value) { await submitTapLiveScore(props.eventId, liveMatchId.value, tapCount.value); const result = await fetchTapLiveResult(props.eventId, liveMatchId.value); if (result.ok) { liveOutcome.value = result.data.outcome; liveCoins.value = Number(result.data.coins_earned || 0); liveMessage.value = result.data.message || 'Tempo scaduto'; liveStep.value = 'result'; emit('claim', { coins: liveCoins.value, source: 'tap_challenge_live', meta: { matchId: liveMatchId.value, taps: tapCount.value, outcome: liveOutcome.value } }); return; } }
  const nextCooldown=Date.now()+Math.max(0,props.cooldownSeconds)*1000; cooldownUntil.value=nextCooldown; window?.localStorage?.setItem('tap_challenge_cooldown_until',String(nextCooldown));
}
async function claimReward(){ if(isSubmitting.value||claimRequested.value||status.value!=='finished') return; claimRequested.value=true; if(earnedCoins.value<=0){ claimRequested.value=false; emit('claim',{coins:0}); return; } isSubmitting.value=true; errorMessage.value=''; const requestId=crypto?.randomUUID?crypto.randomUUID():`tap_${Date.now()}`; const result=await awardTapChallengeCoins({ amount: earnedCoins.value, requestId, eventContextId: props.eventId, meta:{taps:tapCount.value,durationMs:ROUND_DURATION_MS} }); isSubmitting.value=false; if(!result.ok){ errorMessage.value='Errore accredito, riprova.'; claimRequested.value=false; return; } emit('claim',{coins:earnedCoins.value,source:'tap_challenge',meta:{taps:tapCount.value}}); }

async function startLive() {
  if (!props.eventId || !getFanSessionToken()) { errorMessage.value = 'Solo utenti registrati possono usare la sfida live.'; return; }
  errorMessage.value = ''; liveBusy.value = true; liveStep.value = 'searching';
  const q = await joinTapLiveQueue(props.eventId); liveBusy.value = false;
  if (!q.ok) { liveStep.value = 'idle'; errorMessage.value = q.message || 'Nessun avversario trovato'; return; }
  liveMatchId.value = String(q.data?.match_id || ''); matchmakingDeadline.value = Number(q.data?.waiting_deadline || 0) * 1000 || (Date.now() + MATCHMAKING_TIMEOUT_MS); scheduleMatchmakingTimeout();
  openLiveSSE();
  await syncLiveState();
}
async function cancelLiveSearch() { await cancelTapLiveQueue(props.eventId); resetLive(); }
function resetLive() { liveStep.value = 'idle'; liveMatchId.value = ''; liveOutcome.value=''; liveCoins.value=0; liveMessage.value=''; matchmakingDeadline.value = 0; status.value='ready'; timeLeftMs.value = ROUND_DURATION_MS; tapCount.value=0; stopMatchmakingTimer(); sse?.close(); sse = undefined; }
function scheduleMatchmakingTimeout() { stopMatchmakingTimer(); if (typeof window === 'undefined' || !matchmakingDeadline.value || liveStep.value !== 'searching') return; const delay = Math.max(0, matchmakingDeadline.value - Date.now()); matchmakingTimerId = window.setTimeout(async () => { if (liveStep.value !== 'searching') return; await syncLiveState(); if (liveStep.value === 'searching') { errorMessage.value = 'Tempo di attesa scaduto, riprova.'; await cancelTapLiveQueue(props.eventId); resetLive(); } }, delay + 50); }
function openLiveSSE() { if (typeof EventSource === 'undefined') return; sse?.close(); sse = new EventSource(buildTapLiveSseUrl(props.eventId)); sse.onmessage = () => { syncLiveState(); }; sse.addEventListener('update', () => { syncLiveState(); }); sse.onerror = () => {}; }
async function syncLiveState() { const state = await fetchTapLiveState(props.eventId); if (!state.ok) return; const s = state.data?.status || 'idle'; if (s === 'idle') { if (liveStep.value === 'searching') errorMessage.value = 'Nessun avversario trovato'; resetLive(); return; }
  liveMatchId.value = String(state.data?.match_id || liveMatchId.value || '');
  if (s === 'searching') { liveStep.value = 'searching'; scheduleMatchmakingTimeout(); return; }
  if (s === 'matched' || s === 'countdown') { stopMatchmakingTimer(); matchmakingDeadline.value = 0; liveStep.value = 'countdown'; const startAt = Number(state.data?.start_at || 0) * 1000; const ms = Math.max(0, startAt - Date.now()); timeLeftMs.value = ms; stopTimer(); timerId = window.setInterval(() => { const rem = Math.max(0, startAt - Date.now()); timeLeftMs.value = rem; if (rem <= 0) { stopTimer(); liveStep.value = 'playing'; startGame(); } }, 100); return; }
  if (s === 'playing') { liveStep.value = 'playing'; if (status.value !== 'playing') startGame(); return; }
  if (s === 'finished') { const result = await fetchTapLiveResult(props.eventId, liveMatchId.value); if (result.ok) { liveOutcome.value=result.data.outcome; liveCoins.value=Number(result.data.coins_earned||0); liveMessage.value=result.data.message||'Tempo scaduto'; liveStep.value='result'; status.value='finished'; } }
}
function clamp(value,min,max){return Math.min(max,Math.max(min,value));} function distance(ax,ay,bx,by){return Math.hypot(ax-bx,ay-by);} function formatCooldown(ms){const ts=Math.max(0,Math.ceil(ms/1000)); const m=Math.floor(ts/60); const s=ts%60; return `${String(m).padStart(2,'0')}:${String(s).padStart(2,'0')}`;} function formatSeconds(ms){return `${Math.max(1, Math.ceil(ms/1000))}s`;}
if (typeof window !== 'undefined') { const stored=Number.parseInt(window.localStorage.getItem('tap_challenge_cooldown_until')||'',10); if(Number.isFinite(stored)&&stored>Date.now()) cooldownUntil.value=stored; }
</script>

<style scoped>
.tap-challenge { user-select: none; }
.game-area { min-height: 280px; position: relative; overflow: hidden; touch-action: manipulation; }
.ball { position: absolute; left:0; top:0; width:68px; height:68px; border-radius:9999px; border:2px solid rgba(255,255,255,.6); background: radial-gradient(circle at 30% 25%, #fef3c7, #f59e0b 62%, #b45309); color:#111827; font-size:33px; display:inline-flex; align-items:center; justify-content:center; box-shadow:0 12px 26px rgba(2,6,23,.4); }
.overlay { height:100%; display:flex; flex-direction:column; align-items:center; justify-content:center; text-align:center; padding:1rem; }
.cta-primary,.cta-secondary { border-radius:9999px; padding:.75rem 1rem; font-weight:800; }
.cta-primary { background:#fbbf24; color:#0f172a; }
.cta-primary:disabled { opacity:.65; }
.cta-secondary { border:1px solid rgba(255,255,255,.2); color:#fff; background:rgba(255,255,255,.08); }
</style>
