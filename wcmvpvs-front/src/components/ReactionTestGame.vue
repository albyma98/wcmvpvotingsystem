<template>
  <div class="reaction-game">
    <section class="reaction-shell">
      <header class="reaction-header">
        <div class="reaction-brand"><span class="reaction-mark" aria-hidden="true">↯</span><div><span class="eyebrow">IL TUO ALLENAMENTO</span><h2>Reaction <em>Test</em></h2></div></div>
        <div class="reward-pill"><span class="coin-icon" aria-hidden="true"></span><b>{{ hasFixedReward ? `+${fixedReward}` : `+${previewCoins}` }}</b><small>{{ hasFixedReward ? 'a ogni partita completata' : 'monete in palio' }}</small></div>
      </header>

      <div class="reaction-dashboard">
        <div class="round-track" aria-label="Avanzamento dei tre round">
          <span v-for="n in TOTAL_ROUNDS" :key="n" class="round-chip" :class="{ current: currentRound === n && phase !== 'summary', done: roundResults[n - 1], missed: roundResults[n - 1]?.falstart }">
            <small>0{{ n }}</small><b>{{ roundResults[n - 1] ? roundResults[n - 1].falstart ? '+300' : roundResults[n - 1].totalMs : '—' }}<i v-if="roundResults[n - 1]">ms</i></b>
          </span>
        </div>
        <div class="best-time"><span>MIGLIORE</span><b>{{ bestDisplay }}</b></div>
      </div>

      <button type="button" class="play-area" :class="[playAreaClass, `phase-${phase}`]" :aria-label="playAreaAriaLabel" @click="onPlayTap">
        <span class="arena-grid" aria-hidden="true"></span>
        <span class="arena-corner arena-corner--tl" aria-hidden="true"></span><span class="arena-corner arena-corner--br" aria-hidden="true"></span>
        <span class="arena-topline"><span class="status-light" aria-hidden="true"></span>{{ phase === 'summary' ? 'SESSIONE COMPLETATA' : `ROUND 0${currentRound} / 0${TOTAL_ROUNDS}` }}</span>

        <div class="reactor-content">
          <div v-if="['intro', 'countdown', 'wait', 'go'].includes(phase)" class="reactor" aria-hidden="true">
            <span class="reactor-orbit"></span><span class="reactor-orbit reactor-orbit--inner"></span>
            <span class="reactor-core">
              <span v-if="phase === 'countdown'" :key="countdownValue" class="countdown-number">{{ countdownValue }}</span>
              <svg v-else-if="phase === 'wait'" viewBox="0 0 48 48" fill="none"><rect x="14" y="11" width="7" height="26" rx="2" fill="currentColor"/><rect x="27" y="11" width="7" height="26" rx="2" fill="currentColor"/></svg>
              <svg v-else viewBox="0 0 48 48" fill="none"><path d="M28 5 11 27h12l-3 16 17-23H25Z" fill="currentColor"/></svg>
            </span>
          </div>

          <template v-if="phase === 'result'">
            <span class="result-kicker">{{ latestResult?.falstart ? 'TROPPO PRESTO' : 'IL TUO TEMPO' }}</span>
            <strong class="result-time">{{ latestResult?.falstart ? '+300' : latestResult?.reactionMs }}<small>ms</small></strong>
            <span class="result-verdict">{{ performanceLabel }}</span>
            <p class="arena-copy">{{ latestResult?.falstart ? 'Aspetta il verde. Il prossimo round è tuo.' : 'Un istante. Tutta la tua velocità.' }}</p>
          </template>
          <template v-else-if="phase === 'summary'">
            <span class="result-kicker">LA TUA MEDIA · 3 ROUND</span>
            <strong class="result-time">{{ avgMs }}<small>ms</small></strong>
            <span class="result-verdict">{{ performanceLabel }}</span>
            <div class="summary-stats"><span>Migliore <b>{{ bestDisplay }}</b></span><span>False partenze <b>{{ falseStarts }}</b></span></div>
          </template>
          <template v-else>
            <h3>{{ phase === 'intro' ? 'Questione di riflessi.' : phase === 'countdown' ? 'Concentrati.' : phase === 'wait' ? 'Aspetta il verde.' : 'TOCCA ORA!' }}</h3>
            <p class="arena-copy">{{ phase === 'intro' ? 'Tre round. Un solo obiettivo: toccare appena si accende il verde.' : phase === 'countdown' ? 'Il segnale sta per arrivare. Non toccare ancora.' : phase === 'wait' ? 'Non anticipare: una falsa partenza vale +300 ms.' : 'Tocca un punto qualsiasi di questa area.' }}</p>
          </template>
        </div>

        <span class="arena-bottomline"><span aria-hidden="true">{{ phase === 'go' ? '↯' : '◎' }}</span>{{ phase === 'go' ? 'È IL TUO MOMENTO' : phase === 'result' || phase === 'summary' ? 'OGNI MILLISECONDO CONTA' : 'OCCHI SUL SEGNALE' }}</span>
        <span v-if="showCoinBurst" class="coin-burst">+{{ roundCoins }} <span class="coin-icon" aria-hidden="true"></span></span>
      </button>

      <footer class="reaction-controls">
        <button type="button" class="cta-primary" :aria-label="primaryLabel" :disabled="isPrimaryDisabled" @click="onPrimaryAction">{{ primaryLabel }}<span v-if="!isPrimaryDisabled" aria-hidden="true">↗</span></button>
        <button type="button" class="cta-secondary" aria-label="Esci dal gioco" @click="emit('exit')">Esci</button>
      </footer>
      <p class="reaction-announcement" role="status">{{ mainMessage }}. {{ phase === 'result' ? resultMessage : subMessage }}</p>
    </section>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, ref } from 'vue';
import { newMinigameRoundId } from '../fanJourney';

const emit = defineEmits(['claim', 'exit', 'round-started', 'round-completed']);
const props = defineProps({
  completionReward: { type: Number, default: null },
});
const hasFixedReward = computed(() => Number.isFinite(props.completionReward) && props.completionReward >= 0);
const fixedReward = computed(() => hasFixedReward.value ? Math.floor(props.completionReward) : 0);

const TOTAL_ROUNDS = 3;
const FALSTART_PENALTY_MS = 300;

const phase = ref('intro');
const currentRound = ref(1);
const countdownValue = ref(3);
const falseStarts = ref(0);
const roundResults = ref([]);
const goStartTime = ref(0);
const roundCoins = ref(0);
const showCoinBurst = ref(false);
const tapLocked = ref(false);
const claimRequested = ref(false);
let roundId = newMinigameRoundId();

let countdownInterval;
let waitTimeout;
let coinBurstTimeout;

const bestMs = computed(() => {
  if (!roundResults.value.length) {
    return null;
  }

  return Math.min(...roundResults.value.map((result) => result.totalMs));
});

const avgMs = computed(() => {
  if (!roundResults.value.length) {
    return null;
  }

  const total = roundResults.value.reduce((accumulator, result) => accumulator + result.totalMs, 0);
  return Math.round(total / roundResults.value.length);
});

const coins = computed(() => {
  if (roundResults.value.length < TOTAL_ROUNDS || avgMs.value == null) {
    return 0;
  }
  if (hasFixedReward.value) return fixedReward.value;

  let amount = 5;

  if (avgMs.value <= 230) {
    amount = 15;
  } else if (avgMs.value <= 300) {
    amount = 10;
  } else if (avgMs.value <= 380) {
    amount = 7;
  }

  if (falseStarts.value >= 2) {
    return Math.min(amount, 5);
  }

  return amount;
});

const previewCoins = computed(() => {
  if (hasFixedReward.value) return fixedReward.value;
  if (phase.value === 'summary') {
    return coins.value;
  }

  return Math.max(coins.value, 5);
});

const bestDisplay = computed(() => (bestMs.value == null ? '—' : `${bestMs.value} ms`));
const latestResult = computed(() => roundResults.value[roundResults.value.length - 1]);
const performanceLabel = computed(() => {
  if (phase.value === 'result' && latestResult.value?.falstart) return 'Falsa partenza · penalità';
  if (phase.value === 'summary' && roundResults.value.length && roundResults.value.every((result) => result.falstart)) return 'Aspetta il verde. Puoi farcela.';
  const ms = phase.value === 'summary' ? avgMs.value : latestResult.value?.reactionMs;
  if (ms == null) return 'Pronto al prossimo round';
  return ms <= 230 ? 'Riflessi fulminei.' : ms <= 300 ? 'Ottimo tempismo.' : ms <= 380 ? 'Bel colpo.' : 'Il prossimo sarà ancora più veloce.';
});

const mainMessage = computed(() => {
  switch (phase.value) {
    case 'intro':
      return 'Reaction Test';
    case 'wait':
      return 'Aspetta…';
    case 'go':
      return 'TAP!';
    case 'result':
      return 'Round completato';
    case 'summary':
      return 'Risultato finale';
    default:
      return 'Ready';
  }
});

const subMessage = computed(() => {
  switch (phase.value) {
    case 'intro':
      return '3 round rapidi. Tappa appena diventa verde.';
    case 'wait':
      return 'Non toccare ora o prendi penalità';
    case 'go':
      return 'Tocca l’area di gioco!';
    case 'result':
      return 'Feedback istantaneo';
    case 'summary':
      return `Media ${avgMs.value ?? '—'} ms. Migliore ${bestMs.value ?? '—'} ms`;
    default:
      return '';
  }
});

const resultMessage = computed(() => {
  const latest = roundResults.value[roundResults.value.length - 1];
  if (!latest || phase.value !== 'result') {
    return '';
  }

  if (latest.falstart) {
    return `Falsa partenza! Penalità +${FALSTART_PENALTY_MS} ms`;
  }

  return `${latest.reactionMs} ms`;
});

const playAreaClass = computed(() => ({
  wait: phase.value === 'wait',
  go: phase.value === 'go',
  result: phase.value === 'result',
  summary: phase.value === 'summary',
  shake: phase.value === 'result' && roundResults.value[roundResults.value.length - 1]?.falstart,
  flash: phase.value === 'go',
}));

const playAreaAriaLabel = computed(() => {
  if (phase.value === 'go') {
    return 'Tocca ora per registrare il tempo di reazione';
  }

  return 'Area di gioco reaction test';
});

const primaryLabel = computed(() => {
  if (phase.value === 'intro') {
    return 'Inizia';
  }

  if (phase.value === 'result' && currentRound.value < TOTAL_ROUNDS) {
    return 'Prossimo round';
  }

  if (phase.value === 'result' && currentRound.value >= TOTAL_ROUNDS) {
    return 'Vai al riepilogo';
  }

  if (phase.value === 'summary') {
    return claimRequested.value ? 'Bonus in salvataggio…' : hasFixedReward.value ? 'Registra risultato e bonus' : 'Riscatta monete';
  }

  return 'Attendi…';
});

const isPrimaryDisabled = computed(() => ['countdown', 'wait', 'go'].includes(phase.value) || (phase.value === 'summary' && claimRequested.value));

function clearTimers() {
  if (typeof window === 'undefined') {
    return;
  }

  if (countdownInterval) {
    window.clearInterval(countdownInterval);
    countdownInterval = undefined;
  }

  if (waitTimeout) {
    window.clearTimeout(waitTimeout);
    waitTimeout = undefined;
  }

  if (coinBurstTimeout) {
    window.clearTimeout(coinBurstTimeout);
    coinBurstTimeout = undefined;
  }
}

function resetGame() {
  roundId = newMinigameRoundId();
  clearTimers();
  phase.value = 'intro';
  currentRound.value = 1;
  countdownValue.value = 3;
  falseStarts.value = 0;
  roundResults.value = [];
  tapLocked.value = false;
  showCoinBurst.value = false;
  claimRequested.value = false;
}

function startCountdown() {
  clearTimers();
  phase.value = 'countdown';
  countdownValue.value = 3;
  tapLocked.value = false;

  countdownInterval = window.setInterval(() => {
    countdownValue.value -= 1;

    if (countdownValue.value <= 0) {
      window.clearInterval(countdownInterval);
      countdownInterval = undefined;
      startWait();
    }
  }, 450);
}

function startWait() {
  phase.value = 'wait';
  const delay = Math.floor(Math.random() * 1601) + 600;

  waitTimeout = window.setTimeout(() => {
    phase.value = 'go';
    goStartTime.value = performance.now();
    tapLocked.value = false;
  }, delay);
}

function finalizeRound(result) {
  roundResults.value = [...roundResults.value, result];
  phase.value = 'result';
  tapLocked.value = true;
  if (currentRound.value === TOTAL_ROUNDS) emit('round-completed', { roundId });
}

function onPlayTap() {
  if (phase.value === 'go') {
    if (tapLocked.value) {
      return;
    }

    tapLocked.value = true;
    const reactionMs = Math.max(1, Math.round(performance.now() - goStartTime.value));
    clearTimers();

    finalizeRound({
      round: currentRound.value,
      reactionMs,
      totalMs: reactionMs,
      falstart: false,
    });

    showRoundCoins();
    return;
  }

  if (phase.value === 'wait' || phase.value === 'countdown') {
    clearTimers();
    falseStarts.value += 1;

    finalizeRound({
      round: currentRound.value,
      reactionMs: null,
      totalMs: FALSTART_PENALTY_MS,
      falstart: true,
    });
  }
}

function showRoundCoins() {
  if (hasFixedReward.value) return;
  roundCoins.value = Math.max(1, Math.round(coins.value / TOTAL_ROUNDS) || 2);
  showCoinBurst.value = true;

  coinBurstTimeout = window.setTimeout(() => {
    showCoinBurst.value = false;
  }, 900);
}

function onPrimaryAction() {
  if (phase.value === 'intro') {
    resetGame();
    emit('round-started', { roundId });
    startCountdown();
    return;
  }

  if (phase.value === 'result') {
    if (currentRound.value < TOTAL_ROUNDS) {
      currentRound.value += 1;
      startCountdown();
      return;
    }

    phase.value = 'summary';
    return;
  }

  if (phase.value === 'summary') {
    if (claimRequested.value) {
      return;
    }

    claimRequested.value = true;
    emit('claim', {
      gameId: 'reaction',
      roundId,
      coins: coins.value,
      avgMs: avgMs.value,
      bestMs: bestMs.value,
    });
  }
}

onBeforeUnmount(() => {
  clearTimers();
});
</script>

<style scoped>
.reaction-game { --react-accent: #c5f87b; box-sizing: border-box; width: 100%; max-width: 760px; height: 100%; min-height: 0; margin-inline: auto; padding: 14px; overflow-y: auto; scrollbar-width: thin; border-radius: 24px; background: #0a1714; color: #edf6ed; font-family: 'Inter', system-ui, sans-serif; }
.reaction-game *, .reaction-game *::before, .reaction-game *::after { box-sizing: border-box; }
.reaction-shell { display: flex; flex-direction: column; height: 100%; min-height: 0; gap: 12px; }
.reaction-header, .reaction-dashboard, .reaction-controls { flex-shrink: 0; }
.reaction-announcement { position: absolute; width: 1px; height: 1px; padding: 0; overflow: hidden; clip-path: inset(50%); white-space: nowrap; }
button { font: inherit; cursor: pointer; -webkit-tap-highlight-color: transparent; }
button:focus-visible { outline: 3px solid #7ad5ff; outline-offset: 3px; }
.reaction-header { display: flex; align-items: center; justify-content: space-between; gap: 10px; padding: 2px 2px 0; }
.reaction-brand { display: flex; align-items: center; gap: 9px; min-width: 0; }
.reaction-mark { display: grid; place-items: center; width: 37px; height: 43px; flex: none; border-radius: 12px; color: #17251d; background: var(--react-accent); font-size: 34px; line-height: 1; transform: rotate(-7deg); }
.eyebrow { display: block; color: #9da9a1; font-size: 8px; font-weight: 800; letter-spacing: .16em; }
h2 { margin: 4px 0 0; color: #edf6ed; font-size: clamp(16px, 4.7vw, 23px); font-weight: 850; letter-spacing: -.06em; line-height: 1.1; }
h2 em { color: var(--react-accent); font-style: normal; }
.reward-pill { display: grid; grid-template-columns: auto auto; align-items: center; justify-content: end; gap: 3px 6px; color: #f1d69d; }
.reward-pill b { font-size: 12px; white-space: nowrap; }
.reward-pill small { grid-column: 1 / -1; color: #a8ada5; text-align: right; font-size: 8px; }
.coin-icon { display: inline-block; flex: none; width: 14px; height: 14px; border: 2px solid #ffe7ae; border-radius: 50%; background: #dba94e; box-shadow: inset 0 0 0 2px #b88336; }
.reaction-dashboard { display: flex; align-items: stretch; gap: 10px; }
.round-track { display: flex; flex: 1; gap: 5px; }
.round-chip { display: flex; flex: 1; min-width: 0; flex-direction: column; gap: 5px; border: 1px solid #ffffff16; border-radius: 11px; background: #14251f; padding: 9px 10px; }
.round-chip small { color: #859c91; font-size: 8px; font-weight: 800; letter-spacing: .12em; }
.round-chip b { font-size: 14px; font-variant-numeric: tabular-nums; line-height: 1; color: #eaf2e9; }
.round-chip i { font-size: 8px; font-style: normal; color: #9baa9d; margin-left: 2px; }
.round-chip.current { border-color: #c5f87b77; }
.round-chip.current small, .round-chip.done b { color: var(--react-accent); }
.round-chip.missed b { color: #ffb49c; }
.best-time { display: flex; flex-direction: column; justify-content: center; gap: 6px; padding: 0 3px 0 1px; }
.best-time span { color: #7f9187; font-size: 8px; font-weight: 800; letter-spacing: .1em; }
.best-time b { color: #e7eee8; font-size: 14px; font-variant-numeric: tabular-nums; white-space: nowrap; }
.play-area { --signal: #c5f87b; position: relative; display: flex; flex: 1; min-height: 200px; width: 100%; overflow: hidden; flex-direction: column; align-items: center; justify-content: space-between; gap: 8px; padding: 18px 20px; border: 1px solid #c5f87b30; border-radius: 24px; background: radial-gradient(ellipse at 50% 42%, #234631 0, #10271e 40%, #0a1714 80%); color: #edf6ed; touch-action: manipulation; user-select: none; }
.arena-grid { position: absolute; inset: 0; pointer-events: none; background-image: radial-gradient(#b2d2b923 .8px, transparent .8px); background-size: 18px 18px; mask-image: linear-gradient(transparent, #000 30%, #000 70%, transparent); }
.arena-corner { position: absolute; width: 17px; height: 17px; opacity: .5; border-color: var(--signal); border-style: solid; pointer-events: none; }
.arena-corner--tl { top: 13px; left: 13px; border-width: 1px 0 0 1px; border-radius: 3px 0 0; }
.arena-corner--br { bottom: 13px; right: 13px; border-width: 0 1px 1px 0; border-radius: 0 0 3px; }
.arena-topline, .arena-bottomline { position: relative; display: flex; align-items: center; justify-content: center; gap: 7px; color: #99b4a1; font-size: 8px; font-weight: 800; letter-spacing: .2em; line-height: 1.5; }
.arena-bottomline { font-size: 8px; letter-spacing: .15em; }
.status-light { width: 5px; height: 5px; border-radius: 50%; background: var(--signal); box-shadow: 0 0 9px var(--signal); }
.reactor-content { position: relative; display: flex; flex-direction: column; align-items: center; justify-content: center; width: 100%; pointer-events: none; }
.reactor { position: relative; display: grid; place-items: center; width: clamp(110px, 24vh, 200px); aspect-ratio: 1; margin: 10px 0 24px; }
.reactor-orbit { position: absolute; inset: 0; border: 1px dashed #c5f87b50; border-radius: 50%; animation: orbit 28s linear infinite; }
.reactor-orbit::after { content: ''; position: absolute; width: 6px; height: 6px; left: calc(50% - 3px); top: -3px; border-radius: 50%; background: var(--signal); box-shadow: 0 0 12px var(--signal); }
.reactor-orbit--inner { inset: 10%; border: 1px solid #c5f87b1f; animation: none; }
.reactor-orbit--inner::after { display: none; }
.reactor-core { display: grid; place-items: center; width: 65%; height: 65%; border: 1px solid #c5f87b40; border-radius: 50%; color: var(--signal); background: radial-gradient(circle at 40% 30%, #35553b, #152c21 85%); box-shadow: 0 0 50px #c5f87b0a, inset 0 2px 10px #ffffff08; }
.reactor-core svg { width: 48%; height: 48%; filter: drop-shadow(0 0 12px #c5f87b35); }
.countdown-number { font-size: clamp(42px, 8vh, 76px); font-weight: 850; letter-spacing: -.06em; animation: number-in .25s ease-out; }
h3 { max-width: 100%; margin: 0; font-size: clamp(24px, 7vw, 36px); font-weight: 850; line-height: 1.04; letter-spacing: -.065em; color: inherit; }
.arena-copy { max-width: 255px; margin: 12px 0 0; color: #b4c6b9; font-size: 12px; line-height: 1.65; font-weight: 450; }
.play-area.wait { --signal: #f3c777; background: radial-gradient(ellipse at 50% 42%, #473923, #231e17 48%, #161a15); border-color: #f3c77740; }
.wait .reactor-orbit { border-color: #f3c77750; }
.wait .reactor-core { color: #f3c777; border-color: #f3c77755; background: radial-gradient(circle at 40% 30%, #5a4931, #2d261a); animation: breathe 2s ease-in-out infinite; }
.wait .arena-copy { color: #c6bba5; }
.play-area.go { --signal: #23461a; border-color: #c5f87b; background: #c5f87b; color: #15291a; }
.go .arena-grid { opacity: .25; }
.go .reactor-core { color: #c5f87b; border-color: #23461a; background: #23461a; box-shadow: 0 0 0 12px #23461a0d; transform: scale(1.06); }
.go .reactor-orbit { border-color: #23461a60; }
.go .arena-topline, .go .arena-bottomline, .go .arena-copy { color: #345629; }
.go h3 { font-size: clamp(31px, 9vw, 44px); }
.result-kicker { margin: 12px 0 8px; font-size: 9px; font-weight: 800; letter-spacing: .2em; color: #a5b8a8; }
.result-time { font-size: clamp(64px, 16vw, 102px); line-height: 1.1; letter-spacing: -.085em; font-weight: 850; font-variant-numeric: tabular-nums; color: var(--react-accent); }
.result-time small { font-size: 21px; font-weight: 500; letter-spacing: -.04em; margin-left: 6px; color: #b0c5b4; }
.result-verdict { display: inline-block; margin-top: 14px; padding: 7px 13px; border: 1px solid #c5f87b30; border-radius: 30px; color: #d3efa9; background: #c5f87b09; font-size: 11px; font-weight: 700; }
.shake .result-time, .shake .result-verdict { color: #ffb49c; }
.summary-stats { display: flex; flex-wrap: wrap; justify-content: center; gap: 10px 20px; margin-top: 24px; color: #9eb0a4; font-size: 10px; }
.summary-stats b { display: block; margin-top: 5px; color: #e9f3ea; font-size: 15px; }
.reaction-controls { display: grid; grid-template-columns: 1fr auto; gap: 8px; }
.cta-primary, .cta-secondary { min-height: 48px; border-radius: 14px; padding: 12px 16px; font-size: 12px; font-weight: 800; }
.cta-primary { display: flex; align-items: center; justify-content: space-between; gap: 10px; border: 1px solid #c5f87b; background: #c5f87b; color: #15271a; box-shadow: 0 3px 0 #81a54c; }
.cta-primary > span { font-size: 20px; line-height: 1; }
.cta-primary:disabled { cursor: default; opacity: .55; box-shadow: none; }
.cta-secondary { border: 1px solid #ffffff24; background: #15241e; color: #bacabe; }
.cta-primary:not(:disabled):active { transform: translateY(2px); box-shadow: none; }
.coin-burst { position: absolute; top: 18%; right: 10%; display: flex; align-items: center; gap: 5px; color: #f5d18d; font-size: 16px; font-weight: 800; animation: float-up .9s ease forwards; pointer-events: none; }
.shake .reactor-content { animation: shake .24s ease; }
@keyframes orbit { to { transform: rotate(360deg); } }
@keyframes breathe { 50% { box-shadow: 0 0 0 10px #f3c77706, 0 0 45px #f3c77710; } }
@keyframes number-in { from { transform: scale(.82); opacity: .3; } to { transform: scale(1); opacity: 1; } }
@keyframes shake { 25%, 75% { transform: translateX(-4px); } 50% { transform: translateX(4px); } }
@keyframes float-up { from { opacity: 1; transform: translateY(0); } to { opacity: 0; transform: translateY(-35px); } }
@media (max-height: 650px) { .reaction-shell { gap: 8px; } .reaction-mark { height: 32px; width: 30px; font-size: 27px; } .round-chip { padding: 7px 8px; gap: 4px; } .play-area { min-height: 180px; padding: 13px 15px; gap: 6px; } .reactor { width: clamp(80px, 20vh, 130px); margin: 4px 0 14px; } h3 { font-size: 25px; } .arena-copy { margin-top: 8px; font-size: 11px; } .result-time { font-size: 72px; } .cta-primary, .cta-secondary { min-height: 44px; padding: 10px 13px; } }
@media (max-width: 350px) { .reaction-mark { display: none; } .reward-pill small { max-width: 85px; } .round-chip { padding: 8px; } .best-time b { font-size: 12px; } }
@media (prefers-reduced-motion: reduce) { *, *::before, *::after { animation: none !important; transition: none !important; } }
</style>
