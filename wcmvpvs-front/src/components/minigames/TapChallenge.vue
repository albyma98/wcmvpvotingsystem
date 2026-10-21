<template>
  <div class="tap-challenge" :class="{ 'is-playing': status === 'playing', 'is-urgent': status === 'playing' && timeLeftMs <= 3000 }">
    <header class="tap-header">
      <div class="tap-brand"><span class="tap-mark" aria-hidden="true">TC</span><div><span class="eyebrow">INSEGUI IL TUO RITMO</span><h2>Tap <em>Challenge</em></h2></div></div>
      <span class="mode-pill"><i aria-hidden="true"></i>{{ isLiveFlow ? 'LIVE 1 VS 1' : '10 SECONDI' }}</span>
    </header>

    <div v-if="showGameHud" class="hud">
      <div class="hud-card"><span class="hud-card__label">{{ liveStep === 'countdown' ? 'Si parte tra' : 'Tempo rimasto' }}</span><strong class="hud-card__value">{{ liveStep === 'countdown' ? countdownDisplay : timeLabel }}</strong></div>
      <div class="hud-card hud-card--accent"><span class="hud-card__label">Colpi a segno</span><strong :key="tapCount" class="hud-card__value score-pop">{{ tapCount.toString().padStart(2, '0') }}</strong></div>
    </div>
    <div v-if="showGameHud" class="time-rail" role="progressbar" aria-label="Secondi rimanenti" :aria-valuenow="Math.ceil(timeLeftMs / 1000)" :aria-valuemin="0" :aria-valuemax="10"><span :style="{ transform: `scaleX(${Math.max(0, Math.min(1, timeLeftMs / ROUND_DURATION_MS))})` }"></span></div>

    <main
      ref="gameAreaRef"
      class="game-area"
      @touchmove="status === 'playing' && $event.preventDefault()"
    >
      <svg class="tap-court" viewBox="0 0 360 500" preserveAspectRatio="none" aria-hidden="true">
        <path d="M22 22h316v456H22Z" fill="#213c3d" fill-opacity=".5" stroke="#d4e4d1" stroke-opacity=".3" stroke-width="1.3"/>
        <path d="M22 174h316v152H22Z" fill="#e9bd68" fill-opacity=".04"/>
        <g stroke="#d4e4d1" stroke-opacity=".2" stroke-width="1.3" fill="none"><path d="M22 174h316M22 326h316M22 250h316"/><path d="M34 34h292v432H34Z" stroke-opacity=".06"/></g>
        <path d="M10 246h340M10 254h340" stroke="#081d22" stroke-width="2"/><path d="M10 250h340" stroke="#a4c3b5" stroke-opacity=".35" stroke-width="5" stroke-dasharray="1 5"/>
        <g fill="#d5dfca" fill-opacity=".4"><circle cx="10" cy="250" r="4"/><circle cx="350" cy="250" r="4"/></g>
        <text x="180" y="410" text-anchor="middle" fill="#c7d7c4" fill-opacity=".07" font-family="system-ui, sans-serif" font-size="26" font-weight="900" letter-spacing="5">TAP CLUB</text>
      </svg>

      <div v-if="status === 'playing' && hitEffect" :key="hitEffect.id" class="tap-impact" :style="{ left: `${hitEffect.x}px`, top: `${hitEffect.y}px` }" aria-hidden="true"><i></i><b>+1</b></div>

      <button
        v-if="status === 'playing'"
        ref="ballRef"
        type="button"
        class="ball"
        :style="ballStyle"
        aria-label="Tocca la palla"
        @click="onTap"
      >
        <ArcadeVolleyball class="ball-art" />
      </button>

      <span v-if="status === 'playing'" class="tap-play-hint" aria-hidden="true">{{ timeLeftMs <= 3000 ? 'ULTIMI SECONDI. DAI TUTTO!' : 'COLPISCI. INSEGUI. RIPETI.' }}</span>

      <section v-else-if="liveStep === 'searching'" class="overlay overlay--single px-2 text-center">
        <div class="search-pulse mb-5"></div>
        <p class="eyebrow">MATCHMAKING</p>
        <h3 class="mt-2 text-2xl font-black text-white sm:text-3xl">Cerchiamo un avversario live</h3>
        <p class="mt-2.5 max-w-[280px] text-xs leading-5 text-slate-200 sm:text-sm sm:leading-6">Stiamo sincronizzando una sfida reale 1vs1. Resta pronto: appena troviamo l’avversario comparirà la schermata versus.</p>
        <div class="mt-5 w-full max-w-[320px] rounded-[24px] border border-white/10 bg-white/5 px-3.5 py-3 text-left shadow-lg shadow-black/20">
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

      <section v-else-if="liveStep === 'countdown'" class="overlay overlay--single px-1 py-2">
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
          <div class="mt-4 rounded-[20px] border border-cyan-300/20 bg-slate-950/35 px-3.5 py-3.5 text-center">
            <p class="text-[10px] font-semibold uppercase tracking-[0.28em] text-cyan-200/80">Countdown sincronizzato</p>
            <div class="mt-2 flex items-end justify-center gap-3">
              <span class="countdown-ring">{{ countdownDisplay }}</span>
              <span class="pb-2 text-sm font-semibold uppercase tracking-[0.25em] text-emerald-300">{{ countdownWord }}</span>
            </div>
            <p class="mt-3 text-xs text-slate-300">Il match si attiverà per entrambi nello stesso istante reale.</p>
          </div>
      </section>

      <section v-else-if="liveStep === 'result'" class="overlay overlay--single px-1 py-2">
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
      </section>

      <div v-else-if="status !== 'playing'" class="overlay tap-cover" :class="{ 'tap-cover--finished': status === 'finished' }">
        <div class="tap-hero" aria-hidden="true"><span class="hero-orbit"></span><ArcadeVolleyball /><span class="hero-label">{{ status === 'finished' ? 'TIME OUT' : 'INSEGUI LA PALLA' }}</span></div>
        <p class="eyebrow">{{ status === 'finished' ? 'SESSIONE COMPLETATA' : 'VELOCITÀ. PRECISIONE. ISTINTO.' }}</p>
        <template v-if="status === 'finished'"><strong class="final-taps">{{ tapCount }}<small>tap</small></strong><h3>Ogni colpo conta.</h3></template>
        <h3 v-else>Dieci secondi.<br />Lascia il segno.</h3>
        <p class="cover-copy">{{ status === 'finished' ? 'Il tempo si ferma. Il tuo risultato resta.' : 'Tocca la palla e inseguila sul campo. A ogni colpo il ritmo aumenta.' }}</p>
        <div class="tap-rules" v-if="status !== 'finished'"><span><b>10</b> secondi</span><i></i><span><b>1</b> palla</span><i></i><span><b>∞</b> energia</span></div>
        <p v-if="isFixedSingleplayer" class="cover-reward"><span class="tap-coin" aria-hidden="true"></span>{{ props.alreadyRewarded ? 'Bonus già ottenuto in questa partita' : `+${fixedReward} monete al primo completamento` }}</p>
        <p v-else-if="status === 'finished'" class="cover-reward"><span class="tap-coin" aria-hidden="true"></span>{{ earnedCoins }} monete da riscattare</p>
      </div>
    </main>

    <p v-if="errorMessage" class="mt-2.5 rounded-[18px] border border-red-300/40 bg-red-500/10 px-3.5 py-2.5 text-xs text-red-100 sm:text-sm">{{ errorMessage }}</p>

    <footer class="tap-controls" v-if="liveStep !== 'result'">
      <button type="button" class="cta-primary w-full" :disabled="isPrimaryDisabled" @click="onPrimaryAction">{{ primaryLabel }}<span v-if="!isPrimaryDisabled" aria-hidden="true">↗</span></button>
    </footer>

    <div v-if="showLiveButton" class="mt-1.5">
      <button class="cta-secondary w-full" type="button" :disabled="liveBusy" @click="startLive">Sfida live</button>
    </div>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, ref } from 'vue';
import ArcadeVolleyball from './ArcadeVolleyball.vue';
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

const props = defineProps({
  eventId: { type: Number, default: 0 }, cooldownSeconds: { type: Number, default: 60 },
  completionReward: { type: Number, default: null }, alreadyRewarded: { type: Boolean, default: false }, claiming: { type: Boolean, default: false },
});
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
const hitEffect = ref(null);

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

const hasFixedReward = computed(() => Number.isFinite(props.completionReward) && props.completionReward >= 0);
const fixedReward = computed(() => hasFixedReward.value ? Math.floor(props.completionReward) : 0);
const isFixedSingleplayer = computed(() => hasFixedReward.value && liveStep.value === 'idle');
const earnedCoins = computed(() => isFixedSingleplayer.value ? fixedReward.value : tapCount.value);
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
const showGameHud = computed(() => liveStep.value === 'countdown' || status.value === 'playing');
const headline = computed(() => {
  if (liveStep.value === 'searching') return 'Matchmaking in corso';
  if (liveStep.value === 'countdown') return 'Duello confermato';
  if (liveStep.value === 'playing') return 'Tapta più forte che puoi';
  if (liveStep.value === 'result') return 'Match concluso';
  if (status.value === 'finished') return isFixedSingleplayer.value ? `Hai totalizzato ${tapCount.value} tap` : `Hai guadagnato ${earnedCoins.value} monete`;
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
  if (liveStep.value === 'result') return 'Torna al menu';
  if (status.value === 'playing') return 'In corso';
  if (isFixedSingleplayer.value && status.value === 'finished' && props.alreadyRewarded) return 'Termina';
  if (isFixedSingleplayer.value && (props.claiming || claimRequested.value)) return 'Bonus in salvataggio…';
  if (isSubmitting.value || claimRequested.value) return 'Accredito…';
  if (status.value === 'finished' && errorMessage.value) return 'Riprova accredito';
  if (status.value === 'finished') return isFixedSingleplayer.value ? 'Registra risultato e bonus' : 'Riscatta monete';
  if (isCooldownActive.value) return `In cooldown ${formatCooldown(cooldownRemainingMs.value)}`;
  return 'Inizia';
});
const isPrimaryDisabled = computed(() => status.value === 'playing' || liveStep.value === 'countdown' || liveStep.value === 'playing' || isSubmitting.value || (claimRequested.value && !(isFixedSingleplayer.value && props.alreadyRewarded)) || (isFixedSingleplayer.value && props.claiming) || (status.value === 'ready' && isCooldownActive.value));
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
  if (isFixedSingleplayer.value && props.claiming) return;
  if (status.value === 'playing' || isSubmitting.value || (claimRequested.value && !(isFixedSingleplayer.value && props.alreadyRewarded))) return;
  if (status.value === 'finished') {
    claimReward();
    return;
  }
  if (isCooldownActive.value) return;
  startGame();
}
async function startGame() {
  stopTimer();
  stopBallMovement();
  errorMessage.value = '';
  claimRequested.value = false;
  status.value = 'playing';
  tapCount.value = 0;
  hitEffect.value = null;
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
  hitEffect.value = { id: tapCount.value + 1, x: ballX.value + 26, y: ballY.value + 26 };
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
  emit('claim', { coins: liveCoins.value, source: 'tap_challenge_live', keepOpen: true, meta: { matchId: liveMatchId.value, taps: tapCount.value, outcome: liveOutcome.value } });
}
async function claimReward() {
  if (isFixedSingleplayer.value) {
    if (props.claiming || isSubmitting.value || status.value !== 'finished') return;
    if (props.alreadyRewarded) { emit('exit'); return; }
    if (claimRequested.value) return;
    claimRequested.value = true;
    emit('claim', { coins: fixedReward.value });
    return;
  }
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
.tap-challenge { --tap-accent: #f5cf81; box-sizing: border-box; display: flex; flex-direction: column; gap: 10px; width: 100%; max-width: 760px; height: 100%; min-height: 0; margin-inline: auto; padding: 14px; overflow-y: auto; scrollbar-width: thin; border-radius: 24px; background: #102226; color: #f5f1df; user-select: none; font-family: 'Inter', system-ui, sans-serif; }
.tap-challenge *, .tap-challenge *::before, .tap-challenge *::after { box-sizing: border-box; }
button { font: inherit; cursor: pointer; -webkit-tap-highlight-color: transparent; }
button:focus-visible { outline: 3px solid #8ce0e2; outline-offset: 3px; }
.tap-header { display: flex; flex: none; align-items: center; justify-content: space-between; gap: 8px; padding: 2px 0 4px; }
.tap-brand { display: flex; align-items: center; gap: 9px; min-width: 0; }
.tap-mark { display: grid; place-items: center; width: 37px; height: 43px; border: 1px solid #f5cf8150; border-radius: 12px; background: #f5cf8112; color: #f5cf81; font-size: 13px; font-weight: 900; letter-spacing: -.08em; transform: rotate(-7deg); }
.eyebrow { color: #b7c6bc; font-size: 8px; font-weight: 800; letter-spacing: .16em; text-transform: uppercase; }
h2 { margin: 4px 0 0; color: #f5f1df; font-size: clamp(16px, 4.6vw, 23px); font-weight: 850; letter-spacing: -.06em; line-height: 1.1; }
h2 em { color: var(--tap-accent); font-style: normal; }
.mode-pill { display: flex; flex: none; align-items: center; gap: 5px; border: 1px solid #ffffff1a; border-radius: 30px; padding: 7px 8px; color: #d4ded5; font-size: 7px; font-weight: 800; letter-spacing: .09em; }
.mode-pill i { width: 4px; height: 4px; border-radius: 50%; background: #bde4a3; box-shadow: 0 0 8px #bde4a340; }
.hud { display: grid; flex: none; grid-template-columns: 1fr 1fr; gap: 8px; }
.hud-card { display: flex; align-items: center; justify-content: space-between; gap: 8px; min-width: 0; padding: 10px 13px; border: 1px solid #ffffff14; border-radius: 13px; background: #172e32; }
.hud-card--accent { border-color: #f5cf8130; }
.hud-card__label { color: #a5bcb9; font-size: 8px; font-weight: 750; letter-spacing: .08em; text-transform: uppercase; }
.hud-card__value { color: #f6f2de; font-size: 24px; font-weight: 850; font-variant-numeric: tabular-nums; letter-spacing: -.06em; line-height: 1; }
.hud-card--accent .hud-card__value { color: var(--tap-accent); }
.score-pop { animation: score-pop .16s ease-out; }
.time-rail { flex: none; height: 3px; margin: -4px 2px 0; border-radius: 3px; background: #ffffff12; overflow: hidden; }
.time-rail > span { display: block; height: 100%; width: 100%; transform-origin: left; background: #b9dda2; transition: transform .1s linear; }
.is-urgent .time-rail > span { background: #ffad83; }
.is-urgent .hud-card:first-child .hud-card__value { color: #ffad83; }
.game-area { position: relative; display: flex; flex-direction: column; flex: 1; min-height: 170px; width: 100%; isolation: isolate; border: 1px solid #ffffff20; border-radius: 20px; overflow: hidden; background: radial-gradient(ellipse at 50% 30%, #2e5150, #17383c 55%, #122c32); touch-action: manipulation; }
.is-urgent .game-area { border-color: #ffad8370; }
.tap-court { position: absolute; inset: 0; width: 100%; height: 100%; pointer-events: none; }
.game-area::before { position: absolute; content: ''; inset: 0; background: repeating-linear-gradient(0deg, transparent 0 34px, #e7e0bd04 35px 36px); pointer-events: none; }
.overlay { position: relative; z-index: 3; display: flex; flex: 1; min-height: 0; flex-direction: column; align-items: center; justify-content: center; width: 100%; padding: 22px 18px; }
.overlay--single { align-items: stretch; justify-content: flex-start; overflow-y: auto; background: #0e252bea; backdrop-filter: blur(8px); }
.overlay--single.text-center { align-items: center; justify-content: center; }
.overlay--single > * { flex-shrink: 0; }
.overlay--single h3 { letter-spacing: -.04em; line-height: 1.1; }
.tap-cover { text-align: center; justify-content: safe center; background: linear-gradient(#10292c10, #10292c90); overflow-y: auto; }
.tap-cover > * { flex-shrink: 0; }
.tap-hero { position: relative; width: clamp(100px, 23vh, 180px); aspect-ratio: 1; display: grid; place-items: center; margin: 0 0 28px; }
.tap-hero > svg { width: 82%; height: 82%; filter: drop-shadow(0 18px 16px #061c2580); transform: rotate(-18deg); animation: ball-float 4s ease-in-out infinite; }
.hero-orbit { position: absolute; inset: -1%; border: 1px dashed #f5cf8140; border-radius: 50%; transform: rotate(-20deg) scaleY(.8); }
.hero-orbit::after { content: ''; position: absolute; top: 13%; left: 14%; width: 5px; height: 5px; border-radius: 50%; background: #f5cf81; box-shadow: 0 0 12px #f5cf8170; }
.hero-label { position: absolute; right: -8px; bottom: 6px; padding: 6px 10px; border: 1px solid #f5cf8160; border-radius: 5px; background: #f5cf81; color: #203632; font-size: 7px; font-weight: 900; letter-spacing: .14em; transform: rotate(-8deg); }
.tap-cover > h3 { margin: 10px 0 0; color: #fbf4df; font-size: clamp(30px, 8.3vw, 43px); font-weight: 850; line-height: 1.02; letter-spacing: -.065em; }
.cover-copy { max-width: 265px; margin: 14px 0 0; color: #c1d1c5; font-size: 12px; line-height: 1.65; }
.tap-rules { display: flex; align-items: center; justify-content: center; gap: 13px; margin-top: 22px; color: #a9c0b5; font-size: 9px; }
.tap-rules b { display: block; margin-bottom: 4px; color: #f7e6bc; font-size: 21px; line-height: 1; letter-spacing: -.05em; }
.tap-rules i { width: 1px; height: 23px; background: #dbe2c523; }
.cover-reward { display: flex; align-items: center; justify-content: center; gap: 7px; margin: 22px 0 0; color: #f3d69a; font-size: 9px; line-height: 1.5; }
.tap-coin { flex: none; width: 13px; height: 13px; border: 2px solid #ffe7ae; border-radius: 50%; background: #dba94e; box-shadow: inset 0 0 0 2px #b88336; }
.tap-cover--finished .tap-hero { width: 75px; margin-bottom: 18px; }
.tap-cover--finished .hero-label { right: -20px; bottom: 0; }
.final-taps { margin-top: 8px; color: #f5cf81; font-size: clamp(66px, 16vw, 100px); line-height: 1; font-weight: 850; letter-spacing: -.09em; font-variant-numeric: tabular-nums; }
.final-taps small { margin-left: 8px; color: #bacdbb; font-size: 22px; letter-spacing: -.04em; font-weight: 500; }
.tap-cover--finished > h3 { font-size: 25px; }
.ball { position: absolute; left: 0; top: 0; z-index: 2; display: grid; place-items: center; width: 52px; height: 52px; padding: 0; border: 0; border-radius: 50%; background: transparent; touch-action: manipulation; cursor: crosshair; }
.ball::before { content: ''; position: absolute; inset: -5px; border: 1px solid #f5cf8160; border-radius: 50%; background: #f5cf8106; box-shadow: 0 0 20px #f5cf8110; pointer-events: none; }
.ball-art { width: 100%; height: 100%; filter: drop-shadow(0 8px 5px #001c2080); }
.ball:active .ball-art { transform: scale(.91); }
.tap-impact { position: absolute; z-index: 1; width: 52px; height: 52px; transform: translate(-50%, -50%); pointer-events: none; }
.tap-impact i { position: absolute; inset: 0; border: 2px solid #f5cf81; border-radius: 50%; animation: tap-ripple .4s ease-out both; }
.tap-impact b { position: absolute; top: -8px; left: 50%; color: #fff0c7; font-size: 19px; font-weight: 850; text-shadow: 0 2px 8px #072128; animation: tap-points .5s ease-out both; }
.tap-play-hint { position: absolute; z-index: 1; bottom: 10px; left: 10px; right: 10px; color: #d3dbc18a; font-size: 7px; font-weight: 800; letter-spacing: .17em; text-align: center; pointer-events: none; }
.is-urgent .tap-play-hint { color: #ffcc99; }
.search-pulse { width: 76px; height: 76px; margin-inline: auto; border-radius: 50%; border: 1px solid #f5cf8170; background: radial-gradient(circle, #f5cf8130, #f5cf8100 70%); animation: pulse-ring 1.8s ease-out infinite; }
.search-dots { display: inline-flex; gap: 5px; }
.search-dots span { width: 6px; height: 6px; border-radius: 50%; background: #f5cf81; animation: search-bounce 1.1s infinite ease-in-out; }
.search-dots span:nth-child(2) { animation-delay: .12s; }
.search-dots span:nth-child(3) { animation-delay: .24s; }
.versus-player { position: relative; min-width: 0; padding: 14px 8px; border: 1px solid #ffffff18; border-radius: 16px; background: #f5cf8108; text-align: center; }
.versus-player--opponent { background: #a7d4d108; }
.versus-player__avatar { width: 42px; height: 42px; margin: 0 auto; display: grid; place-items: center; border: 1px solid #f5cf8130; border-radius: 50%; background: #f5cf8115; font-size: 20px; }
.versus-player__label { margin-top: 8px; color: #b9cfc3; font-size: 8px; letter-spacing: .12em; text-transform: uppercase; }
.versus-player h4 { margin: 5px 0 0; font-size: 12px; font-weight: 800; color: #fff1d2; overflow-wrap: anywhere; }
.versus-center span { display: grid; place-items: center; width: 35px; height: 35px; color: #f5cf81; font-style: italic; font-size: 18px; font-weight: 900; letter-spacing: -.08em; }
.countdown-ring { display: grid; place-items: center; width: 66px; height: 66px; border: 1px solid #f5cf8180; border-radius: 50%; background: #f5cf8110; color: #f5cf81; font-size: 35px; font-weight: 850; }
.result-badge { flex: none; border-radius: 7px; padding: 8px 10px; font-size: 10px; font-weight: 900; letter-spacing: .12em; transform: rotate(6deg); }
.result-badge--win { background: #c5f87b; color: #1e3b24; }
.result-badge--draw { background: #f5cf81; color: #3b331e; }
.result-badge--lose { background: #f4b4a0; color: #422c22; }
.result-stat { padding: 11px 13px; border: 1px solid #ffffff15; border-radius: 13px; background: #ffffff05; }
.result-stat__label, .score-card__label { display: block; color: #a8c0b7; font-size: 8px; font-weight: 750; letter-spacing: .08em; text-transform: uppercase; overflow-wrap: anywhere; }
.result-stat__value, .score-card__value { display: block; margin-top: 4px; color: #f6e4bd; font-size: 22px; font-weight: 850; font-variant-numeric: tabular-nums; }
.score-card__value { font-size: 42px; letter-spacing: -.07em; }
.score-divider { color: #a5bdb7; font-size: 10px; font-weight: 800; font-style: italic; }
.tap-controls { flex: none; }
.cta-primary, .cta-secondary { min-height: 46px; border-radius: 13px; padding: 12px 15px; font-size: 12px; font-weight: 800; line-height: 1.3; transition: transform .15s; }
.cta-primary { display: flex; align-items: center; justify-content: center; gap: 12px; border: 1px solid #f5cf81; background: #f5cf81; color: #24322a; box-shadow: 0 3px 0 #aa8d50; }
.cta-primary > span { margin-left: auto; font-size: 20px; line-height: 1; }
.cta-primary:not(:disabled):active { transform: translateY(2px); box-shadow: none; }
.cta-primary:disabled { opacity: .5; box-shadow: none; cursor: default; }
.cta-secondary { border: 1px solid #ffffff20; background: #ffffff06; color: #d8e2d5; }
.cta-secondary:disabled { opacity: .5; cursor: default; }
@keyframes ball-float { 50% { transform: translateY(-7px) rotate(-12deg); } }
@keyframes tap-ripple { from { transform: scale(.7); opacity: .9; } to { transform: scale(2); opacity: 0; } }
@keyframes tap-points { from { transform: translate(-50%, 0); opacity: 1; } to { transform: translate(-50%, -35px); opacity: 0; } }
@keyframes score-pop { from { transform: scale(1.15); } to { transform: scale(1); } }
@keyframes pulse-ring { 70% { box-shadow: 0 0 0 20px #f5cf8100; } from { box-shadow: 0 0 0 0 #f5cf8120; } }
@keyframes search-bounce { 0%, 80%, 100% { transform: translateY(0); opacity: .45; } 40% { transform: translateY(-5px); opacity: 1; } }
@media (max-height: 650px) { .tap-challenge { gap: 8px; padding: 10px; } .tap-mark { height: 31px; width: 30px; } .tap-hero { width: clamp(70px, 18vh, 115px); margin-bottom: 18px; } .tap-cover { padding: 16px 12px; } .tap-cover > h3 { font-size: 30px; } .cover-copy { margin-top: 10px; font-size: 11px; } .tap-rules { margin-top: 14px; } .cover-reward { margin-top: 14px; font-size: 8px; } .tap-rules b { font-size: 18px; } .hud-card { padding: 9px 10px; } .hud-card__value { font-size: 22px; } .tap-cover--finished .tap-hero { width: 52px; margin-bottom: 12px; } .final-taps { font-size: 64px; } .tap-cover--finished > h3 { font-size: 23px; } .cta-primary, .cta-secondary { min-height: 42px; padding: 10px 12px; } }
@media (max-width: 350px) { .tap-mark { display: none; } .mode-pill { font-size: 6px; padding: 6px; } .hud-card__label { font-size: 7px; } }
@media (prefers-reduced-motion: reduce) { *, *::before, *::after { animation: none !important; transition: none !important; } .tap-impact { display: none; } }
</style>
