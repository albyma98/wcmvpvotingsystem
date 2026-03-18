<template>
  <div class="tap-challenge flex h-full min-h-0 w-full flex-col">
    <header class="rounded-[24px] border border-white/12 bg-slate-950/65 p-3 text-slate-100 shadow-[0_16px_40px_rgba(2,6,23,0.28)] backdrop-blur-xl">
      <div class="flex items-start justify-between gap-2.5">
        <div>
          <p class="text-[10px] font-semibold uppercase tracking-[0.28em] text-cyan-200/80">Tap Challenge 1vs1</p>
          <h2 class="mt-1.5 text-base font-black leading-tight text-white sm:text-lg">{{ headline }}</h2>
          <p class="mt-1 text-xs leading-5 text-slate-300 sm:text-sm">{{ subline }}</p>
        </div>
        <div class="rounded-[18px] border border-white/10 bg-white/5 px-2.5 py-2 text-right shadow-inner shadow-white/5">
          <p class="text-[9px] font-semibold uppercase tracking-[0.24em] text-slate-400">Stato</p>
          <p class="mt-1 text-[11px] font-bold leading-4 text-white sm:text-xs">{{ statusLabel }}</p>
        </div>
      </div>

      <div class="mt-3 grid grid-cols-2 gap-2.5">
        <div class="stat-card">
          <span class="stat-card__label">Tempo</span>
          <strong class="stat-card__value">{{ timeLabel }}</strong>
        </div>
        <div class="stat-card">
          <span class="stat-card__label">Tap</span>
          <strong class="stat-card__value">{{ tapCount }}</strong>
        </div>
      </div>
    </header>

    <main
      ref="gameAreaRef"
      class="game-area relative mt-3 flex-1 overflow-hidden rounded-[26px] border border-white/14 bg-[radial-gradient(circle_at_top,_rgba(34,211,238,0.18),_rgba(15,23,42,0.9)_42%,_rgba(2,6,23,0.98)_100%)]"
      @touchmove.prevent
    >
      <div class="pointer-events-none absolute inset-0 opacity-80">
        <div class="absolute inset-x-[-20%] top-[-18%] h-40 rounded-full bg-cyan-400/18 blur-3xl"></div>
        <div class="absolute bottom-[-18%] right-[-8%] h-44 w-44 rounded-full bg-fuchsia-500/18 blur-3xl"></div>
      </div>

      <button
        v-if="status === 'playing'"
        ref="ballRef"
        type="button"
        class="ball"
        :style="ballStyle"
        aria-label="Tappa la palla"
        @click="onTap"
      >
        <span>🏐</span>
      </button>

      <section v-else-if="liveStep === 'searching'" class="overlay px-4 text-center">
        <div class="search-pulse mb-5"></div>
        <p class="eyebrow">MATCHMAKING</p>
        <h3 class="mt-2 text-2xl font-black text-white sm:text-3xl">Cerchiamo un avversario live</h3>
        <p class="mt-2.5 max-w-[280px] text-xs leading-5 text-slate-200 sm:text-sm sm:leading-6">Stiamo sincronizzando una sfida reale 1vs1. Resta pronto: appena troviamo l’avversario comparirà la schermata versus.</p>
        <div class="mt-4 rounded-[24px] border border-white/10 bg-white/5 px-3.5 py-3 text-left shadow-lg shadow-black/20">
          <div class="flex items-center justify-between gap-3">
            <div>
              <p class="text-[10px] font-semibold uppercase tracking-[0.28em] text-cyan-200/80">Tempo stimato</p>
              <p class="mt-1 text-lg font-black text-white">{{ matchmakingLabel }}</p>
            </div>
            <div class="search-dots" aria-hidden="true">
              <span></span><span></span><span></span>
            </div>
          </div>
        </div>
      </section>

      <section v-else-if="liveStep === 'countdown'" class="overlay px-3.5 py-4">
        <div class="versus-card w-full max-w-[320px]">
          <p class="eyebrow text-center">MATCH TROVATO</p>
          <div class="mt-4 grid grid-cols-[1fr_auto_1fr] items-center gap-2.5">
            <div class="versus-player versus-player--me">
              <div class="versus-player__avatar">🧑</div>
              <p class="versus-player__label">Tu</p>
              <h4>{{ myNicknameLabel }}</h4>
            </div>
            <div class="versus-center">
              <span>VS</span>
            </div>
            <div class="versus-player versus-player--opponent">
              <div class="versus-player__avatar">⚡</div>
              <p class="versus-player__label">Avversario</p>
              <h4>{{ opponentNicknameLabel }}</h4>
            </div>
          </div>
          <div class="mt-4 rounded-[20px] border border-cyan-300/20 bg-slate-950/55 px-3.5 py-3.5 text-center">
            <p class="text-[10px] font-semibold uppercase tracking-[0.28em] text-cyan-200/80">Countdown sincronizzato</p>
            <div class="mt-2 flex items-end justify-center gap-3">
              <span class="countdown-ring">{{ countdownDisplay }}</span>
              <span class="pb-2 text-sm font-semibold uppercase tracking-[0.25em] text-emerald-300">{{ countdownWord }}</span>
            </div>
            <p class="mt-3 text-xs text-slate-300">Il match si attiverà per entrambi nello stesso istante reale.</p>
          </div>
        </div>
      </section>

      <section v-else-if="liveStep === 'result'" class="overlay px-3.5 py-4">
        <div class="result-card w-full max-w-[360px]">
          <div class="flex items-start justify-between gap-4">
            <div>
              <p class="eyebrow">RISULTATO MATCH</p>
              <h3 class="mt-2 text-2xl font-black text-white sm:text-3xl">{{ resultTitle }}</h3>
              <p class="mt-2 text-xs leading-5 text-slate-300 sm:text-sm">{{ resultSubtitle }}</p>
            </div>
            <div class="result-badge" :class="`result-badge--${resultTone}`">{{ resultBadge }}</div>
          </div>

          <div class="mt-4 grid grid-cols-[1fr_auto_1fr] items-center gap-2.5 rounded-[22px] border border-white/10 bg-white/5 p-3.5">
            <div class="score-card text-left">
              <p class="score-card__label">{{ myNicknameLabel }}</p>
              <strong class="score-card__value">{{ liveSummary.myScore }}</strong>
            </div>
            <div class="score-divider">VS</div>
            <div class="score-card text-right">
              <p class="score-card__label">{{ opponentNicknameLabel }}</p>
              <strong class="score-card__value">{{ liveSummary.opponentScore }}</strong>
            </div>
          </div>

          <div class="mt-3 grid grid-cols-2 gap-2.5">
            <div class="result-stat">
              <span class="result-stat__label">Differenza</span>
              <strong class="result-stat__value">{{ tapDeltaLabel }}</strong>
            </div>
            <div class="result-stat">
              <span class="result-stat__label">Premio</span>
              <strong class="result-stat__value">+{{ liveCoins }}</strong>
            </div>
            <div class="result-stat col-span-2">
              <span class="result-stat__label">Stato avversario</span>
              <strong class="result-stat__value text-base">{{ rematchStatusLabel }}</strong>
            </div>
          </div>

          <p class="mt-3 rounded-[18px] border border-white/10 bg-slate-950/50 px-3.5 py-3 text-xs leading-5 text-slate-200 sm:text-sm">{{ rematchHint }}</p>

          <div class="mt-4 grid grid-cols-1 gap-2">
            <button type="button" class="cta-primary" :disabled="rematchButtonDisabled" @click="requestRematchAction">{{ rematchButtonLabel }}</button>
            <div class="grid grid-cols-2 gap-2">
              <button type="button" class="cta-secondary" :disabled="rematchBusy" @click="searchNewOpponent">Cerca nuovo avversario</button>
              <button type="button" class="cta-secondary" :disabled="rematchBusy" @click="leavePostMatchAndExit">Torna al menu</button>
            </div>
          </div>
        </div>
      </section>

      <div v-else class="overlay px-4 text-center">
        <p class="eyebrow">TAP CHALLENGE</p>
        <h3 class="mt-2 text-2xl font-black text-white sm:text-3xl">Sfida da 10 secondi</h3>
        <p class="mt-2.5 max-w-[280px] text-xs leading-5 text-slate-200 sm:text-sm sm:leading-6">Tappa la palla più velocemente possibile: in live 1vs1 partirai con countdown condiviso e schermata versus dedicata.</p>
        <p v-if="status === 'finished'" class="mt-3 text-sm font-semibold text-amber-200">Hai guadagnato {{ earnedCoins }} monete nel singleplayer.</p>
      </div>
    </main>

    <p v-if="errorMessage" class="mt-2.5 rounded-[18px] border border-red-300/40 bg-red-500/10 px-3.5 py-2.5 text-xs text-red-100 sm:text-sm">{{ errorMessage }}</p>

    <footer class="mt-3 grid grid-cols-1 gap-2 sm:grid-cols-2">
      <button type="button" class="cta-primary" :disabled="isPrimaryDisabled" @click="onPrimaryAction">{{ primaryLabel }}</button>
      <button type="button" class="cta-secondary" @click="handleSecondaryAction">{{ secondaryLabel }}</button>
    </footer>

    <div v-if="showLiveButton" class="mt-1.5">
      <button class="cta-secondary w-full" type="button" :disabled="liveBusy" @click="startLive">Sfida live</button>
    </div>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, ref } from 'vue';
import { awardTapChallengeCoins } from '../../services/coins';
import {
  abortTapLiveMatch,
  buildTapLiveSseUrl,
  cancelTapLiveQueue,
  fetchTapLiveResult,
  fetchTapLiveState,
  getFanSessionToken,
  joinTapLiveQueue,
  leaveTapLivePostmatch,
  requestTapLiveRematch,
  submitTapLiveScore,
} from '../../api';

const props = defineProps({ eventId: { type: Number, default: 0 }, cooldownSeconds: { type: Number, default: 60 } });
const emit = defineEmits(['claim', 'exit']);

const ROUND_DURATION_MS = 10_000;
const MATCHMAKING_TIMEOUT_MS = 20_000;
const TICK_MS = 100;
const MIN_DISTANCE_PX = 30;
const BASE_MOVE_INTERVAL_MS = 850;
const MIN_MOVE_INTERVAL_MS = 220;
const SPEED_GAIN_PER_TAP = 0.92;

const status = ref('ready');
const timeLeftMs = ref(ROUND_DURATION_MS);
const tapCount = ref(0);
const cooldownUntil = ref(0);
const errorMessage = ref('');
const isSubmitting = ref(false);
const claimRequested = ref(false);
const nowTs = ref(Date.now());
const serverOffsetMs = ref(0);

const gameAreaRef = ref(null);
const ballRef = ref(null);
const ballX = ref(0);
const ballY = ref(0);
const lastBallX = ref(0);
const lastBallY = ref(0);
const moveIntervalMs = ref(BASE_MOVE_INTERVAL_MS);

const liveStep = ref('idle');
const liveBusy = ref(false);
const liveMatchId = ref('');
const liveOutcome = ref('');
const liveCoins = ref(0);
const liveMessage = ref('');
const matchmakingDeadline = ref(0);
const countdownValue = ref(3);
const opponentNickname = ref('Avversario');
const myNickname = ref('Player');
const liveSummary = ref({ myScore: 0, opponentScore: 0, tapDelta: 0, status: '', rematch: {} });
const rematchState = ref({ status: 'idle', message: '', my_choice: 'waiting', opponent_choice: 'waiting', opponent_available: true, accepted: false, next_match_id: '' });
const rematchBusy = ref(false);

let sse;
let timerId;
let moveTimerId;
let gameEndsAt = 0;
let cooldownTickId;
let matchmakingTimerId;
let preMatchTimerId;
let awaitingResult = false;

const earnedCoins = computed(() => tapCount.value);
const isCooldownActive = computed(() => cooldownUntil.value > nowTs.value);
const cooldownRemainingMs = computed(() => Math.max(0, cooldownUntil.value - nowTs.value));
const isLiveFlow = computed(() => liveStep.value !== 'idle');
const statusLabel = computed(() => {
  if (liveStep.value === 'searching') return 'Ricerca avversario';
  if (liveStep.value === 'countdown') return 'Versus + countdown';
  if (liveStep.value === 'playing') return 'Match live';
  if (liveStep.value === 'result') return 'Post-partita';
  if (status.value === 'playing') return 'Singleplayer attivo';
  if (status.value === 'finished') return 'Singleplayer terminato';
  return 'Pronto';
});
const timeLabel = computed(() => `${(Math.max(0, timeLeftMs.value) / 1000).toFixed(status.value === 'playing' ? 1 : 0)}s`);
const showLiveButton = computed(() => status.value === 'ready' && liveStep.value === 'idle');
const headline = computed(() => {
  if (liveStep.value === 'searching') return 'Matchmaking in corso';
  if (liveStep.value === 'countdown') return 'Duello confermato';
  if (liveStep.value === 'playing') return 'Tapta più forte che puoi';
  if (liveStep.value === 'result') return 'Match concluso';
  if (status.value === 'finished') return `Hai guadagnato ${earnedCoins.value} monete`;
  return 'Tap Challenge';
});
const subline = computed(() => {
  if (liveStep.value === 'searching') return 'Ti colleghiamo a un avversario reale compatibile con il tuo evento.';
  if (liveStep.value === 'countdown') return 'Versus reveal con partenza sincronizzata per entrambi i player.';
  if (liveStep.value === 'playing') return 'Countdown terminato: il match è davvero live adesso.';
  if (liveStep.value === 'result') return 'Resta qui per vedere esito, premio e stato rivincita.';
  return status.value === 'ready' ? 'Tappa la palla più volte possibile in 10 secondi.' : 'Tappa la palla più volte possibile';
});
const primaryLabel = computed(() => {
  if (liveStep.value === 'searching') return 'Annulla ricerca';
  if (liveStep.value === 'countdown' || liveStep.value === 'playing') return 'Match in corso';
  if (liveStep.value === 'result') return 'Chiudi risultato';
  if (isSubmitting.value || claimRequested.value) return 'Accredito…';
  if (status.value === 'finished' && errorMessage.value) return 'Riprova accredito';
  if (status.value === 'finished') return 'Riscatta monete';
  if (isCooldownActive.value) return `In cooldown ${formatCooldown(cooldownRemainingMs.value)}`;
  return 'Inizia';
});
const secondaryLabel = computed(() => {
  if (liveStep.value === 'result') return 'Esci';
  if (liveStep.value === 'searching') return 'Esci';
  return 'Esci';
});
const isPrimaryDisabled = computed(() => liveStep.value === 'countdown' || liveStep.value === 'playing' || isSubmitting.value || claimRequested.value || (status.value === 'ready' && isCooldownActive.value));
const ballStyle = computed(() => ({ transform: `translate3d(${ballX.value}px, ${ballY.value}px, 0)` }));
const matchmakingLabel = computed(() => matchmakingDeadline.value > nowTs.value ? formatSeconds(matchmakingDeadline.value - nowTs.value) : 'quasi pronto');
const countdownDisplay = computed(() => countdownValue.value <= 0 ? 'GO' : countdownValue.value);
const countdownWord = computed(() => countdownValue.value <= 0 ? 'VIA' : 'READY');
const myNicknameLabel = computed(() => myNickname.value || 'Tu');
const opponentNicknameLabel = computed(() => opponentNickname.value || 'Avversario');
const resultTone = computed(() => liveOutcome.value === 'win' || liveOutcome.value === 'forfeit_win' ? 'win' : liveOutcome.value === 'draw' ? 'draw' : 'lose');
const resultBadge = computed(() => resultTone.value === 'win' ? 'WIN' : resultTone.value === 'draw' ? 'DRAW' : 'LOSE');
const resultTitle = computed(() => {
  if (liveOutcome.value === 'win' || liveOutcome.value === 'forfeit_win') return 'Hai vinto il duello';
  if (liveOutcome.value === 'draw') return 'Sfida in parità';
  return 'Match perso';
});
const resultSubtitle = computed(() => liveMessage.value || 'Match completato');
const tapDeltaLabel = computed(() => {
  const delta = Number(liveSummary.value.tapDelta || 0);
  if (delta === 0) return 'Parità perfetta';
  return `${delta > 0 ? '+' : ''}${delta} tap`;
});
const rematchStatusLabel = computed(() => rematchState.value.message || 'In attesa');
const rematchHint = computed(() => {
  if (!rematchState.value.opponent_available || rematchState.value.status === 'opponent_left') return 'L’avversario non è più disponibile. Puoi tornare al menu o cercarne subito un altro.';
  if (rematchState.value.status === 'accepted') return 'Entrambi avete confermato la rivincita: stiamo preparando la nuova schermata versus.';
  if (rematchState.value.status === 'opponent_requested') return 'L’avversario ha già chiesto la rivincita. Conferma per far partire automaticamente la nuova sfida.';
  if (rematchState.value.status === 'waiting_opponent') return 'Richiesta inviata. Resta in questa schermata finché l’avversario decide.';
  return 'La schermata resta aperta per gestire la rivincita e gli stati live dell’avversario.';
});
const rematchButtonLabel = computed(() => {
  if (rematchBusy.value) return 'Invio…';
  if (rematchState.value.status === 'accepted') return 'Rivincita accettata';
  if (rematchState.value.status === 'waiting_opponent') return 'Rivincita richiesta';
  if (rematchState.value.status === 'opponent_requested') return 'Accetta rivincita';
  if (!rematchState.value.opponent_available || rematchState.value.status === 'opponent_left') return 'Avversario uscito';
  return 'Rivincita';
});
const rematchButtonDisabled = computed(() => rematchBusy.value || rematchState.value.status === 'accepted' || (!rematchState.value.opponent_available && rematchState.value.status !== 'opponent_requested'));

onBeforeUnmount(async () => {
  stopTimer();
  stopBallMovement();
  stopMatchmakingTimer();
  stopCountdownTimer();
  sse?.close();
  if (liveMatchId.value && (liveStep.value === 'searching' || liveStep.value === 'countdown' || liveStep.value === 'playing')) {
    await abortTapLiveMatch(props.eventId, liveMatchId.value);
  } else if (liveMatchId.value && liveStep.value === 'result') {
    await leaveTapLivePostmatch(props.eventId, liveMatchId.value);
  }
  if (cooldownTickId && typeof window !== 'undefined') window.clearInterval(cooldownTickId);
});

if (typeof window !== 'undefined') {
  cooldownTickId = window.setInterval(() => {
    nowTs.value = Date.now();
  }, 250);
}

function stopTimer() {
  if (!timerId || typeof window === 'undefined') return;
  window.clearInterval(timerId);
  timerId = undefined;
}
function stopBallMovement() {
  if (!moveTimerId || typeof window === 'undefined') return;
  window.clearTimeout(moveTimerId);
  moveTimerId = undefined;
}
function stopMatchmakingTimer() {
  if (!matchmakingTimerId || typeof window === 'undefined') return;
  window.clearTimeout(matchmakingTimerId);
  matchmakingTimerId = undefined;
}
function stopCountdownTimer() {
  if (!preMatchTimerId || typeof window === 'undefined') return;
  window.clearInterval(preMatchTimerId);
  preMatchTimerId = undefined;
}
function getServerNow() {
  return Date.now() + serverOffsetMs.value;
}
function syncServerOffset(serverNow) {
  const parsed = Number(serverNow || 0);
  if (parsed > 0) serverOffsetMs.value = parsed - Date.now();
}
function scheduleBallMovement() {
  stopBallMovement();
  if (typeof window === 'undefined' || status.value !== 'playing') return;
  moveTimerId = window.setTimeout(() => {
    if (status.value !== 'playing') return;
    repositionBall();
    scheduleBallMovement();
  }, moveIntervalMs.value);
}
function onPrimaryAction() {
  if (liveStep.value === 'searching') {
    cancelLiveSearch();
    return;
  }
  if (liveStep.value === 'result') {
    leavePostMatchAndExit();
    return;
  }
  if (status.value === 'playing' || isSubmitting.value || claimRequested.value) return;
  if (status.value === 'finished') {
    claimReward();
    return;
  }
  if (isCooldownActive.value) return;
  startGame();
}
function handleSecondaryAction() {
  if (liveStep.value === 'result') {
    leavePostMatchAndExit();
    return;
  }
  emit('exit');
}
async function startGame() {
  stopTimer();
  stopBallMovement();
  errorMessage.value = '';
  claimRequested.value = false;
  status.value = 'playing';
  tapCount.value = 0;
  timeLeftMs.value = ROUND_DURATION_MS;
  moveIntervalMs.value = BASE_MOVE_INTERVAL_MS;
  gameEndsAt = Date.now() + ROUND_DURATION_MS;
  await nextTick();
  repositionBall(true);
  scheduleBallMovement();
  if (typeof window === 'undefined') return;
  timerId = window.setInterval(() => {
    const remaining = gameEndsAt - Date.now();
    timeLeftMs.value = Math.max(0, remaining);
    if (remaining <= 0) finishGame();
  }, TICK_MS);
}
function repositionBall(force = false) {
  const area = gameAreaRef.value;
  const ball = ballRef.value;
  if (!area || !ball) return;
  const areaRect = area.getBoundingClientRect();
  const ballRect = ball.getBoundingClientRect();
  const maxX = Math.max(0, areaRect.width - ballRect.width);
  const maxY = Math.max(0, areaRect.height - ballRect.height);
  let nextX = 0;
  let nextY = 0;
  let attempts = 0;
  do {
    nextX = clamp(Math.random() * maxX, 0, maxX);
    nextY = clamp(Math.random() * maxY, 0, maxY);
    attempts += 1;
  } while (!force && attempts < 8 && distance(nextX, nextY, lastBallX.value, lastBallY.value) < MIN_DISTANCE_PX);
  ballX.value = nextX;
  ballY.value = nextY;
  lastBallX.value = nextX;
  lastBallY.value = nextY;
}
function onTap() {
  if (status.value !== 'playing') return;
  tapCount.value += 1;
  moveIntervalMs.value = Math.max(MIN_MOVE_INTERVAL_MS, moveIntervalMs.value * SPEED_GAIN_PER_TAP);
  navigator?.vibrate?.(10);
  repositionBall();
  scheduleBallMovement();
}
async function finishGame() {
  if (status.value !== 'playing') return;
  stopTimer();
  stopBallMovement();
  status.value = 'finished';
  timeLeftMs.value = 0;
  if (liveStep.value === 'playing' && liveMatchId.value) {
    awaitingResult = true;
    await submitTapLiveScore(props.eventId, liveMatchId.value, tapCount.value);
    await loadLiveResult();
    awaitingResult = false;
    return;
  }
  const nextCooldown = Date.now() + Math.max(0, props.cooldownSeconds) * 1000;
  cooldownUntil.value = nextCooldown;
  window?.localStorage?.setItem('tap_challenge_cooldown_until', String(nextCooldown));
}
async function loadLiveResult() {
  const result = await fetchTapLiveResult(props.eventId, liveMatchId.value);
  if (!result.ok) {
    errorMessage.value = 'Stiamo recuperando il risultato finale…';
    return;
  }
  applyLiveResult(result.data);
}
function applyLiveResult(data = {}) {
  syncServerOffset(data.server_now);
  liveOutcome.value = data.outcome || '';
  liveCoins.value = Number(data.coins_earned || 0);
  liveMessage.value = data.message || 'Tempo scaduto';
  opponentNickname.value = String(data.opponent_nickname || opponentNickname.value || 'Avversario');
  myNickname.value = String(data.my_nickname || myNickname.value || 'Tu');
  liveSummary.value = {
    myScore: Number(data.my_score || 0),
    opponentScore: Number(data.opponent_score || 0),
    tapDelta: Number(data.tap_delta || 0),
    status: String(data.status || ''),
    rematch: data.rematch || {},
  };
  rematchState.value = normalizeRematchState(data.rematch || {});
  liveStep.value = 'result';
  status.value = 'finished';
  emit('claim', { coins: liveCoins.value, source: 'tap_challenge_live', meta: { matchId: liveMatchId.value, taps: tapCount.value, outcome: liveOutcome.value } });
}
async function claimReward() {
  if (isSubmitting.value || claimRequested.value || status.value !== 'finished') return;
  claimRequested.value = true;
  if (earnedCoins.value <= 0) {
    claimRequested.value = false;
    emit('claim', { coins: 0 });
    return;
  }
  isSubmitting.value = true;
  errorMessage.value = '';
  const requestId = crypto?.randomUUID ? crypto.randomUUID() : `tap_${Date.now()}`;
  const result = await awardTapChallengeCoins({ amount: earnedCoins.value, requestId, eventContextId: props.eventId, meta: { taps: tapCount.value, durationMs: ROUND_DURATION_MS } });
  isSubmitting.value = false;
  if (!result.ok) {
    errorMessage.value = 'Errore accredito, riprova.';
    claimRequested.value = false;
    return;
  }
  emit('claim', { coins: earnedCoins.value, source: 'tap_challenge', meta: { taps: tapCount.value } });
}

async function startLive() {
  if (!props.eventId || !getFanSessionToken()) {
    errorMessage.value = 'Solo utenti registrati possono usare la sfida live.';
    return;
  }
  errorMessage.value = '';
  liveBusy.value = true;
  liveStep.value = 'searching';
  const q = await joinTapLiveQueue(props.eventId);
  liveBusy.value = false;
  if (!q.ok) {
    liveStep.value = 'idle';
    errorMessage.value = q.message || 'Nessun avversario trovato';
    return;
  }
  liveMatchId.value = String(q.data?.match_id || '');
  matchmakingDeadline.value = Number(q.data?.waiting_deadline || 0) * 1000 || (Date.now() + MATCHMAKING_TIMEOUT_MS);
  scheduleMatchmakingTimeout();
  openLiveSSE();
  await syncLiveState();
}
async function cancelLiveSearch() {
  await cancelTapLiveQueue(props.eventId);
  resetLive();
}
function resetLive() {
  liveStep.value = 'idle';
  liveMatchId.value = '';
  liveOutcome.value = '';
  liveCoins.value = 0;
  liveMessage.value = '';
  matchmakingDeadline.value = 0;
  countdownValue.value = 3;
  rematchBusy.value = false;
  rematchState.value = normalizeRematchState({});
  liveSummary.value = { myScore: 0, opponentScore: 0, tapDelta: 0, status: '', rematch: {} };
  status.value = 'ready';
  timeLeftMs.value = ROUND_DURATION_MS;
  tapCount.value = 0;
  stopTimer();
  stopCountdownTimer();
  stopBallMovement();
  stopMatchmakingTimer();
  sse?.close();
  sse = undefined;
}
function scheduleMatchmakingTimeout() {
  stopMatchmakingTimer();
  if (typeof window === 'undefined' || !matchmakingDeadline.value || liveStep.value !== 'searching') return;
  const delay = Math.max(0, matchmakingDeadline.value - Date.now());
  matchmakingTimerId = window.setTimeout(async () => {
    if (liveStep.value !== 'searching') return;
    await syncLiveState();
    if (liveStep.value === 'searching') {
      errorMessage.value = 'Tempo di attesa scaduto, riprova.';
      await cancelTapLiveQueue(props.eventId);
      resetLive();
    }
  }, delay + 80);
}
function openLiveSSE() {
  if (typeof EventSource === 'undefined') return;
  sse?.close();
  sse = new EventSource(buildTapLiveSseUrl(props.eventId));
  sse.onmessage = () => {
    syncLiveState();
  };
  sse.addEventListener('update', () => {
    syncLiveState();
  });
  sse.onerror = () => {};
}
async function syncLiveState() {
  const state = await fetchTapLiveState(props.eventId);
  if (!state.ok) return;
  const data = state.data || {};
  syncServerOffset(data.server_now);
  const s = data.status || 'idle';
  myNickname.value = String(data.my_nickname || myNickname.value || 'Tu');
  opponentNickname.value = String(data.opponent_nickname || opponentNickname.value || 'Avversario');
  if (s === 'idle') {
    if (liveStep.value === 'searching') errorMessage.value = 'Nessun avversario trovato';
    resetLive();
    return;
  }
  liveMatchId.value = String(data.match_id || liveMatchId.value || '');
  if (s === 'searching') {
    liveStep.value = 'searching';
    matchmakingDeadline.value = Number(data.waiting_deadline || 0) * 1000 || matchmakingDeadline.value;
    scheduleMatchmakingTimeout();
    return;
  }
  if (s === 'matched' || s === 'countdown') {
    stopMatchmakingTimer();
    matchmakingDeadline.value = 0;
    liveStep.value = 'countdown';
    startPrematchCountdown(Number(data.start_at || 0) * 1000);
    return;
  }
  if (s === 'playing') {
    liveStep.value = 'playing';
    if (status.value !== 'playing') startSynchronizedGame(Number(data.start_at || 0) * 1000);
    return;
  }
  if (s === 'finished') {
    rematchState.value = normalizeRematchState(data.rematch || rematchState.value);
    if (liveStep.value !== 'result' || awaitingResult) await loadLiveResult();
  }
}
function startPrematchCountdown(startAtMs) {
  stopCountdownTimer();
  const safeStartAt = Number(startAtMs || 0);
  const updateCountdown = () => {
    const remainingMs = Math.max(0, safeStartAt - getServerNow());
    const remainingSeconds = Math.ceil(remainingMs / 1000);
    countdownValue.value = remainingMs <= 120 ? 0 : remainingSeconds;
    timeLeftMs.value = remainingMs;
    if (remainingMs <= 0) {
      stopCountdownTimer();
      liveStep.value = 'playing';
      startSynchronizedGame(safeStartAt);
    }
  };
  updateCountdown();
  if (typeof window === 'undefined') return;
  preMatchTimerId = window.setInterval(updateCountdown, 80);
}
async function startSynchronizedGame(startAtMs) {
  if (status.value === 'playing') return;
  const delay = Math.max(0, startAtMs - getServerNow());
  if (delay > 30) {
    if (typeof window !== 'undefined') {
      window.setTimeout(() => startSynchronizedGame(startAtMs), delay);
    }
    return;
  }
  liveStep.value = 'playing';
  await startGame();
}
async function requestRematchAction() {
  if (!liveMatchId.value || rematchButtonDisabled.value) return;
  rematchBusy.value = true;
  const action = rematchState.value.status === 'opponent_requested' ? 'request' : 'request';
  const response = await requestTapLiveRematch(props.eventId, liveMatchId.value, action);
  rematchBusy.value = false;
  if (!response.ok) {
    errorMessage.value = response.message || 'Impossibile inviare la richiesta di rivincita.';
    return;
  }
  rematchState.value = normalizeRematchState(response.data || {});
  if (rematchState.value.accepted) {
    errorMessage.value = '';
    await syncLiveState();
  }
}
async function leavePostMatchAndExit() {
  if (liveMatchId.value) await leaveTapLivePostmatch(props.eventId, liveMatchId.value);
  resetLive();
  emit('exit');
}
async function searchNewOpponent() {
  if (liveMatchId.value) await leaveTapLivePostmatch(props.eventId, liveMatchId.value);
  resetLive();
  await startLive();
}
function normalizeRematchState(data = {}) {
  return {
    status: String(data.status || 'idle'),
    message: String(data.message || 'Scegli se chiedere la rivincita.'),
    my_choice: String(data.my_choice || 'waiting'),
    opponent_choice: String(data.opponent_choice || 'waiting'),
    opponent_available: data.opponent_available !== false,
    accepted: Boolean(data.accepted),
    next_match_id: String(data.next_match_id || ''),
  };
}
function clamp(value, min, max) { return Math.min(max, Math.max(min, value)); }
function distance(ax, ay, bx, by) { return Math.hypot(ax - bx, ay - by); }
function formatCooldown(ms) {
  const ts = Math.max(0, Math.ceil(ms / 1000));
  const m = Math.floor(ts / 60);
  const s = ts % 60;
  return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`;
}
function formatSeconds(ms) { return `${Math.max(1, Math.ceil(ms / 1000))}s`; }

if (typeof window !== 'undefined') {
  const stored = Number.parseInt(window.localStorage.getItem('tap_challenge_cooldown_until') || '', 10);
  if (Number.isFinite(stored) && stored > Date.now()) cooldownUntil.value = stored;
}
</script>

<style scoped>
.tap-challenge { user-select: none; }
.game-area { min-height: min(43vh, 300px); max-height: 300px; touch-action: manipulation; }
.overlay { position: relative; z-index: 1; height: 100%; display: flex; flex-direction: column; align-items: center; justify-content: center; }
.tap-challenge :deep(.cta-primary), .tap-challenge :deep(.cta-secondary) { min-height: 44px; }
.eyebrow { font-size: 10px; font-weight: 800; letter-spacing: 0.32em; text-transform: uppercase; color: rgba(165, 243, 252, 0.8); }
.stat-card, .result-stat {
  border-radius: 22px;
  border: 1px solid rgba(255,255,255,0.1);
  background: rgba(255,255,255,0.06);
  padding: 0.75rem 0.85rem;
  box-shadow: inset 0 1px 0 rgba(255,255,255,0.04);
}
.stat-card__label, .result-stat__label, .score-card__label {
  display: block;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.22em;
  text-transform: uppercase;
  color: rgba(203, 213, 225, 0.75);
}
.stat-card__value, .result-stat__value, .score-card__value {
  display: block;
  margin-top: 0.25rem;
  font-size: 1.05rem;
  font-weight: 900;
  color: #fff;
}
.ball {
  position: absolute;
  left: 0;
  top: 0;
  z-index: 2;
  width: 52px;
  height: 52px;
  border-radius: 9999px;
  border: 2px solid rgba(255,255,255,.72);
  background: radial-gradient(circle at 30% 25%, #fef3c7, #f59e0b 60%, #b45309);
  color: #111827;
  font-size: 30px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 14px 24px rgba(2,6,23,.42), 0 0 0 6px rgba(251,191,36,0.12);
}
.search-pulse {
  width: 92px;
  height: 92px;
  border-radius: 9999px;
  border: 1px solid rgba(34, 211, 238, 0.38);
  background: radial-gradient(circle, rgba(34,211,238,0.38), rgba(34,211,238,0.04));
  box-shadow: 0 0 0 0 rgba(34,211,238,0.4);
  animation: pulse-ring 1.8s ease-out infinite;
}
.search-dots { display: inline-flex; gap: 6px; }
.search-dots span {
  width: 8px; height: 8px; border-radius: 9999px; background: #67e8f9; animation: search-bounce 1.1s infinite ease-in-out;
}
.search-dots span:nth-child(2) { animation-delay: 0.12s; }
.search-dots span:nth-child(3) { animation-delay: 0.24s; }
.versus-card, .result-card {
  border-radius: 32px;
  border: 1px solid rgba(255,255,255,0.12);
  background: linear-gradient(180deg, rgba(15,23,42,0.82), rgba(2,6,23,0.96));
  padding: 1rem;
  box-shadow: 0 24px 70px rgba(2,6,23,0.55);
  animation: card-rise 380ms cubic-bezier(.21,1,.21,1);
}
.versus-player {
  position: relative;
  border-radius: 26px;
  border: 1px solid rgba(255,255,255,0.1);
  padding: 0.85rem 0.7rem;
  text-align: center;
  background: rgba(255,255,255,0.04);
  overflow: hidden;
}
.versus-player::after {
  content: '';
  position: absolute;
  inset: -30% auto auto -20%;
  width: 100px;
  height: 100px;
  background: radial-gradient(circle, rgba(34,211,238,0.32), transparent 68%);
  filter: blur(10px);
}
.versus-player--opponent::after { background: radial-gradient(circle, rgba(244,114,182,0.28), transparent 68%); }
.versus-player__avatar {
  margin: 0 auto;
  width: 46px;
  height: 46px;
  border-radius: 18px;
  border: 1px solid rgba(255,255,255,0.14);
  background: rgba(255,255,255,0.09);
  display: grid;
  place-items: center;
  font-size: 1.2rem;
}
.versus-player__label { margin-top: 0.55rem; font-size: 9px; letter-spacing: 0.24em; text-transform: uppercase; color: rgba(186, 230, 253, 0.72); }
.versus-player h4 { margin-top: 0.3rem; font-size: 0.92rem; font-weight: 900; color: #fff; word-break: break-word; }
.versus-center span {
  display: grid;
  place-items: center;
  width: 52px;
  height: 52px;
  border-radius: 9999px;
  font-size: 1rem;
  font-weight: 900;
  color: #fff;
  background: linear-gradient(135deg, rgba(34,211,238,0.4), rgba(168,85,247,0.35));
  box-shadow: 0 0 28px rgba(34,211,238,0.28);
}
.countdown-ring {
  width: 72px;
  height: 72px;
  border-radius: 9999px;
  border: 1px solid rgba(52,211,153,0.5);
  display: grid;
  place-items: center;
  font-size: 1.7rem;
  font-weight: 900;
  color: #fff;
  background: radial-gradient(circle, rgba(16,185,129,0.26), rgba(15,23,42,0.35));
  box-shadow: 0 0 35px rgba(16,185,129,0.18);
}
.result-badge {
  border-radius: 9999px;
  padding: 0.45rem 0.7rem;
  font-size: 0.75rem;
  font-weight: 900;
  letter-spacing: 0.18em;
}
.result-badge--win { background: rgba(16,185,129,0.18); color: #6ee7b7; }
.result-badge--draw { background: rgba(250,204,21,0.16); color: #fde68a; }
.result-badge--lose { background: rgba(248,113,113,0.16); color: #fca5a5; }
.score-divider {
  font-size: 0.75rem;
  font-weight: 900;
  letter-spacing: 0.28em;
  color: rgba(186, 230, 253, 0.75);
}
.cta-primary, .cta-secondary {
  border-radius: 9999px;
  padding: 0.75rem 0.85rem;
  font-weight: 900;
  transition: transform 160ms ease, opacity 160ms ease, border-color 160ms ease;
}
.cta-primary {
  background: linear-gradient(135deg, #fbbf24, #fb7185);
  color: #0f172a;
  box-shadow: 0 12px 24px rgba(251, 191, 36, 0.24);
}
.cta-primary:disabled { opacity: .6; }
.cta-secondary {
  border: 1px solid rgba(255,255,255,.18);
  color: #fff;
  background: rgba(255,255,255,.07);
}
@keyframes pulse-ring {
  0% { transform: scale(.92); box-shadow: 0 0 0 0 rgba(34,211,238,0.35); }
  70% { transform: scale(1); box-shadow: 0 0 0 22px rgba(34,211,238,0); }
  100% { transform: scale(.92); }
}
@keyframes search-bounce {
  0%, 80%, 100% { transform: translateY(0); opacity: .45; }
  40% { transform: translateY(-5px); opacity: 1; }
}
@keyframes card-rise {
  from { opacity: 0; transform: translateY(18px) scale(.97); }
  to { opacity: 1; transform: translateY(0) scale(1); }
}
</style>
