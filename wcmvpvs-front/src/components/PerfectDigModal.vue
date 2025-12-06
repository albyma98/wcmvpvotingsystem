<template>
  <div class="perfect-dig">
    <p class="perfect-dig__eyebrow">Mini gioco</p>
    <h4 class="perfect-dig__title">Perfect DIG – Difesa perfetta</h4>
    <p class="perfect-dig__subtitle">
      Blocca la palla con la piattaforma! Tre livelli, tre tentativi: sincronizza il tocco
      quando la palla arriva nella zona target e rimbalza verso il palleggiatore.
    </p>

    <div class="perfect-dig__level-header">
      <span class="perfect-dig__level-label">Livello {{ currentLevel }} / 3</span>
      <div class="perfect-dig__level-progress" role="presentation" aria-hidden="true">
        <span
          v-for="level in levelConfigs"
          :key="level.id"
          class="perfect-dig__level-dot"
          :class="{
            'perfect-dig__level-dot--done': completedLevels.includes(level.id),
            'perfect-dig__level-dot--active': level.id === currentLevel,
          }"
        >
          {{ completedLevels.includes(level.id) ? '●' : '○' }}
        </span>
      </div>
    </div>

    <div
      class="perfect-dig__field"
      :class="{
        'perfect-dig__field--success': feedbackState === 'success',
        'perfect-dig__field--fail': feedbackState === 'fail',
        'perfect-dig__field--disabled': !props.enabled,
      }"
      role="button"
      tabindex="0"
      @pointerdown.prevent="handleTap"
      @keydown.space.prevent="handleTap"
      @keydown.enter.prevent="handleTap"
    >
      <div class="perfect-dig__court">
        <div class="perfect-dig__line perfect-dig__line--base"></div>
        <div class="perfect-dig__line perfect-dig__line--attack"></div>
        <div class="perfect-dig__line perfect-dig__line--center"></div>
      </div>

      <div class="perfect-dig__target" :class="{ 'perfect-dig__target--hidden': !showTargetHint }" :style="targetStyle">
        <span v-if="showTargetHint" class="perfect-dig__target-label">Zona target</span>
      </div>

      <Transition name="perfect-dig-arms">
        <div v-if="showArms" class="perfect-dig__arms" aria-hidden="true">
          <div class="perfect-dig__arm perfect-dig__arm--left" />
          <div class="perfect-dig__arm perfect-dig__arm--right" />
        </div>
      </Transition>

      <div class="perfect-dig__ball" :class="ballClasses" :style="ballStyle">
        <span class="perfect-dig__ball-band perfect-dig__ball-band--one"></span>
        <span class="perfect-dig__ball-band perfect-dig__ball-band--two"></span>
      </div>

      <Transition name="perfect-dig-feedback">
        <div v-if="feedbackMessage" class="perfect-dig__feedback" :class="feedbackClasses">
          {{ feedbackMessage }}
        </div>
      </Transition>

      <div v-if="confettiPieces.length" class="perfect-dig__confetti" aria-hidden="true">
        <span
          v-for="piece in confettiPieces"
          :key="piece.id"
          class="perfect-dig__confetti-piece"
          :style="{
            left: piece.left,
            animationDelay: piece.delay,
            backgroundColor: piece.color,
          }"
        />
      </div>
    </div>

    <div class="perfect-dig__info">
      <p class="perfect-dig__attempts">Tentativi: {{ attempts }} / {{ maxAttempts }}</p>
      <p class="perfect-dig__status" :class="{ 'perfect-dig__status--alert': feedbackState === 'fail' }">
        {{ statusLabel }}
      </p>
    </div>

    <div v-if="gameOver" class="perfect-dig__summary" aria-live="polite">
      <p class="perfect-dig__summary-title">Riepilogo</p>
      <p class="perfect-dig__summary-stats">Ricezioni riuscite: {{ successCount }} / {{ maxAttempts }}</p>
      <p class="perfect-dig__summary-reward">{{ rewardMessage }}</p>
    </div>

    <div v-if="levelComplete" class="perfect-dig__level-end" aria-live="polite">
      <p class="perfect-dig__level-end-title">{{ levelEndTitle }}</p>
      <p class="perfect-dig__level-end-subtitle">{{ levelEndSubtitle }}</p>
      <div class="perfect-dig__level-end-actions">
        <button
          v-if="showNextLevelCta"
          type="button"
          class="perfect-dig__primary"
          @click="goToNextLevel"
        >
          {{ nextLevelLabel }}
        </button>
        <button
          v-else
          type="button"
          class="perfect-dig__primary"
          @click="startGame"
        >
          Riprova livello
        </button>
        <button type="button" class="perfect-dig__secondary" @click="emit('close')">
          Chiudi
        </button>
      </div>
    </div>

    <div class="perfect-dig__actions">
      <button
        type="button"
        class="perfect-dig__primary"
        :disabled="!props.enabled || isPlaying"
        @click="startGame"
      >
        {{ gameCtaLabel }}
      </button>
      <button type="button" class="perfect-dig__secondary" @click="emit('close')">Chiudi</button>
    </div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, ref } from 'vue';

const props = defineProps({
  eventId: {
    type: Number,
    default: null,
  },
  enabled: {
    type: Boolean,
    default: true,
  },
  completed: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits(['game-finished', 'close']);

const maxAttempts = 3;
const targetLine = 74;
const targetHeight = 16;
const groundLine = 96;

const levelConfigs = [
  {
    id: 1,
    fallSpeed: 0.16,
    fallJitter: 0.018,
    lateralDrift: 0.025,
    floatVariance: 0,
    zoneWidth: 28,
    requiredSuccess: 2,
    completionTitle: 'Missione completata! +1 chance',
    completionSubtitle: 'Difesa solida, hai fermato il primo side-out.',
  },
  {
    id: 2,
    fallSpeed: 0.19,
    fallJitter: 0.024,
    lateralDrift: 0.032,
    floatVariance: 0.12,
    zoneWidth: 20,
    requiredSuccess: 2,
    completionTitle: '+1 chance extra!',
    completionSubtitle: 'Timing più stretto, ma piattaforma impeccabile.',
  },
  {
    id: 3,
    fallSpeed: 0.22,
    fallJitter: 0.03,
    lateralDrift: 0.042,
    floatVariance: 0.36,
    zoneWidth: 16,
    requiredSuccess: 2,
    completionTitle: 'LIBERO D\'ACCIAIO',
    completionSubtitle: 'Bagher d\'élite! Badge e confetti verdi sbloccati.',
  },
];

const attempts = ref(0);
const successCount = ref(0);
const isPlaying = ref(false);
const gameOver = ref(false);
const feedbackMessage = ref('');
const feedbackState = ref('');
const statusLabel = ref('3 livelli: bagher sulla traiettoria, punta il palleggiatore.');
const ballX = ref(50);
const ballY = ref(-20);
const ballRotation = ref(0);
const currentLevel = ref(1);
const levelComplete = ref(false);
const levelPassed = ref(false);
const completedLevels = ref([]);
const showTargetHint = ref(true);
const showArms = ref(false);
const confettiPieces = ref([]);
const pauseTimer = ref(null);
const targetHintTimer = ref(null);
let animationFrame = null;
let lastTs = 0;
let vx = 0;
let vy = 0;
let attemptConsumed = false;

const currentLevelConfig = computed(
  () => levelConfigs.find((level) => level.id === currentLevel.value) || levelConfigs[0],
);

const nextLevelLabel = computed(() => {
  if (currentLevel.value === levelConfigs.length) {
    return '';
  }
  return `Vai al livello ${currentLevel.value + 1}`;
});

const rewardType = computed(() => {
  const passed = successCount.value >= currentLevelConfig.value.requiredSuccess;
  if (!passed) {
    return 'none';
  }
  return currentLevel.value === 3 ? 'badge' : 'chance';
});

const levelEndTitle = computed(() => {
  if (!levelPassed.value) {
    return 'Riprova il livello';
  }
  return currentLevelConfig.value.completionTitle;
});

const levelEndSubtitle = computed(() => {
  if (!levelPassed.value) {
    return 'Ti servono almeno 2 ricezioni su 3. Riprova!';
  }
  if (currentLevel.value === 3 && rewardType.value === 'badge') {
    return 'Badge LIBERO D\'ACCIAIO sbloccato!';
  }
  return currentLevelConfig.value.completionSubtitle;
});

const showNextLevelCta = computed(() => levelPassed.value && currentLevel.value < levelConfigs.length);

const rewardMessage = computed(() => {
  if (rewardType.value === 'badge') {
    return 'Perfetto! Hai i riflessi di un libero pro: badge e chance extra assegnati.';
  }
  if (rewardType.value === 'chance') {
    return 'Ottimo! +1 chance extra per questo livello.';
  }
  return 'Ritenta per ottenere la chance bonus.';
});

const gameCtaLabel = computed(() => {
  if (isPlaying.value) {
    return 'In corso…';
  }
  if (attempts.value > 0 && !gameOver.value) {
    return 'In corso…';
  }
  if (gameOver.value && levelPassed.value) {
    return currentLevel.value === levelConfigs.length
      ? 'Rigioca il livello finale'
      : 'Riprova o passa al prossimo livello';
  }
  if (gameOver.value) {
    return 'Riprova il livello';
  }
  return `Avvia livello ${currentLevel.value}`;
});

const ballStyle = computed(() => ({
  left: `${ballX.value}%`,
  top: `${ballY.value}%`,
  transform: `translate(-50%, -50%) rotate(${ballRotation.value}deg)`,
}));

const ballClasses = computed(() => ({
  'perfect-dig__ball--active': isPlaying.value,
  'perfect-dig__ball--success': feedbackState.value === 'success',
  'perfect-dig__ball--fail': feedbackState.value === 'fail',
}));

const feedbackClasses = computed(() => ({
  'perfect-dig__feedback--success': feedbackState.value === 'success',
  'perfect-dig__feedback--fail': feedbackState.value === 'fail',
}));

const targetCenter = ref(50);

const targetStyle = computed(() => {
  const width = currentLevelConfig.value.zoneWidth;
  const left = targetCenter.value - width / 2;
  return {
    left: `${Math.max(4, Math.min(96 - width, left))}%`,
    width: `${width}%`,
  };
});

function resetState() {
  attempts.value = 0;
  successCount.value = 0;
  gameOver.value = false;
  levelComplete.value = false;
  levelPassed.value = false;
  feedbackMessage.value = '';
  feedbackState.value = '';
  statusLabel.value = '3 livelli: bagher sulla traiettoria, punta il palleggiatore.';
  ballX.value = 50;
  ballY.value = -20;
  ballRotation.value = 0;
  showArms.value = false;
  confettiPieces.value = [];
  showTargetHint.value = true;
  targetCenter.value = 50;
  attemptConsumed = false;
}

function stopAnimation() {
  if (animationFrame) {
    window.cancelAnimationFrame(animationFrame);
    animationFrame = null;
  }
  lastTs = 0;
}

function playTone({ frequency = 280, duration = 0.12, type = 'sine', volume = 0.2, sweep = 0 }) {
  if (typeof window === 'undefined' || typeof window.AudioContext === 'undefined') {
    return;
  }
  const ctx = playTone.ctx || new window.AudioContext();
  playTone.ctx = ctx;
  const oscillator = ctx.createOscillator();
  const gain = ctx.createGain();
  oscillator.type = type;
  oscillator.frequency.value = frequency;
  if (sweep !== 0) {
    oscillator.frequency.linearRampToValueAtTime(frequency + sweep, ctx.currentTime + duration);
  }
  gain.gain.value = volume;
  oscillator.connect(gain).connect(ctx.destination);
  oscillator.start();
  gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + duration);
  oscillator.stop(ctx.currentTime + duration + 0.05);
}

function playSound(kind) {
  if (kind === 'dig') {
    playTone({ frequency: 240, duration: 0.14, type: 'triangle', volume: 0.35, sweep: -40 });
  } else if (kind === 'fail') {
    playTone({ frequency: 170, duration: 0.18, type: 'sawtooth', volume: 0.22, sweep: -90 });
  } else if (kind === 'cheer') {
    playTone({ frequency: 520, duration: 0.32, type: 'square', volume: 0.2, sweep: -60 });
    window.setTimeout(() => playTone({ frequency: 360, duration: 0.22, type: 'triangle', volume: 0.18 }), 80);
  }
}

function applyVibration() {
  if (typeof navigator !== 'undefined' && typeof navigator.vibrate === 'function') {
    navigator.vibrate([80, 60]);
  }
}

function launchBall() {
  stopAnimation();
  if (attempts.value >= maxAttempts) {
    finishGame();
    return;
  }
  isPlaying.value = true;
  gameOver.value = false;
  levelComplete.value = false;
  feedbackState.value = '';
  feedbackMessage.value = '';
  showArms.value = false;
  confettiPieces.value = [];
  attemptConsumed = false;

  const config = currentLevelConfig.value;
  const fromLeft = Math.random() > 0.5;
  const startX = fromLeft ? 16 + Math.random() * 6 : 84 - Math.random() * 6;
  const randomCenter = 50 + (Math.random() * 6 - 3);
  targetCenter.value = randomCenter;
  ballX.value = startX;
  ballY.value = -12;
  ballRotation.value = Math.random() * 180;
  vx = (randomCenter - startX) * config.lateralDrift;
  vy = config.fallSpeed + Math.random() * config.fallJitter;

  showTargetHint.value = true;
  if (targetHintTimer.value) {
    window.clearTimeout(targetHintTimer.value);
  }
  targetHintTimer.value = window.setTimeout(() => {
    showTargetHint.value = false;
  }, 1000);

  startAnimation();
}

function animateBall(timestamp) {
  if (!isPlaying.value || levelComplete.value || gameOver.value) {
    return;
  }
  if (!lastTs) {
    lastTs = timestamp;
  }
  const delta = Math.min(48, timestamp - lastTs);
  lastTs = timestamp;
  const config = currentLevelConfig.value;
  const floatWobble = config.floatVariance ? Math.sin(timestamp / 120) * config.floatVariance : 0;

  ballX.value = Math.max(6, Math.min(94, ballX.value + (vx + floatWobble) * delta));
  ballY.value += vy * delta;
  ballRotation.value = (ballRotation.value + delta * 0.35) % 360;

  if (!attemptConsumed && ballY.value >= groundLine) {
    registerFail('FUORI TEMPO!');
    return;
  }

  animationFrame = window.requestAnimationFrame(animateBall);
}

function startAnimation() {
  stopAnimation();
  animationFrame = window.requestAnimationFrame(animateBall);
}

function consumeAttempt() {
  if (attemptConsumed) {
    return false;
  }
  attemptConsumed = true;
  attempts.value += 1;
  return true;
}

function registerSuccess() {
  if (!consumeAttempt()) {
    return;
  }
  successCount.value += 1;
  feedbackState.value = 'success';
  feedbackMessage.value = 'RICEZIONE PERFETTA! 🔥';
  statusLabel.value = 'Braccia compatte, traiettoria alzata!';
  showArms.value = true;
  playSound('dig');
  startDigAnimation();
}

function registerFail(reason = 'FUORI TEMPO!') {
  if (!consumeAttempt()) {
    return;
  }
  stopAnimation();
  feedbackState.value = 'fail';
  feedbackMessage.value = reason;
  statusLabel.value = reason === 'PIATTAFORMA APERTA!'
    ? 'Chiudi le braccia e aspetta il contatto.'
    : 'Resta basso e colpisci al tempo giusto.';
  applyVibration();
  playSound('fail');
  startDropAnimation();
}

function startDropAnimation() {
  const startY = ballY.value;
  const endY = groundLine;
  const startTs = performance.now();
  const duration = 380;

  function tick(now) {
    const progress = Math.min(1, (now - startTs) / duration);
    const eased = 1 - Math.pow(1 - progress, 2);
    ballY.value = startY + (endY - startY) * eased;
    ballRotation.value = (ballRotation.value + 3) % 360;

    if (progress < 1) {
      animationFrame = window.requestAnimationFrame(tick);
    } else {
      const bounceTs = performance.now();
      const bounceDuration = 260;
      function bounceStep(currentTs) {
        const bounceProgress = Math.min(1, (currentTs - bounceTs) / bounceDuration);
        const bounceOffset = Math.sin(bounceProgress * Math.PI) * 3;
        ballY.value = endY - bounceOffset;
        if (bounceProgress < 1) {
          animationFrame = window.requestAnimationFrame(bounceStep);
          return;
        }
        scheduleResume();
      }
      animationFrame = window.requestAnimationFrame(bounceStep);
    }
  }

  animationFrame = window.requestAnimationFrame(tick);
}

function startDigAnimation() {
  stopAnimation();
  const start = { x: ballX.value, y: ballY.value };
  const apex = { x: 50, y: 20 };
  const end = { x: 54, y: 6 };
  const startTs = performance.now();
  const duration = 780;

  function digStep(now) {
    const t = Math.min(1, (now - startTs) / duration);
    const ease = 1 - Math.pow(1 - t, 3);
    ballX.value = (1 - ease) * (1 - ease) * start.x + 2 * (1 - ease) * ease * apex.x + ease * ease * end.x;
    ballY.value = (1 - ease) * (1 - ease) * start.y + 2 * (1 - ease) * ease * apex.y + ease * ease * end.y;
    ballRotation.value = (ballRotation.value + 2) % 360;
    if (t < 1) {
      animationFrame = window.requestAnimationFrame(digStep);
      return;
    }
    scheduleResume();
  }

  animationFrame = window.requestAnimationFrame(digStep);
}

function scheduleResume() {
  if (pauseTimer.value) {
    window.clearTimeout(pauseTimer.value);
  }
  pauseTimer.value = window.setTimeout(() => {
    feedbackState.value = '';
    feedbackMessage.value = '';
    showArms.value = false;
    if (attempts.value >= maxAttempts) {
      finishGame();
      return;
    }
    if (!gameOver.value) {
      launchBall();
    }
  }, 680);
}

function handleTap() {
  if (!isPlaying.value || gameOver.value || !props.enabled || levelComplete.value) {
    return;
  }

  const inHorizontalRange =
    ballX.value >= targetCenter.value - currentLevelConfig.value.zoneWidth / 2 &&
    ballX.value <= targetCenter.value + currentLevelConfig.value.zoneWidth / 2;
  const inVerticalRange = ballY.value >= targetLine && ballY.value <= targetLine + targetHeight;

  if (inHorizontalRange && inVerticalRange) {
    registerSuccess();
  } else {
    const reason = Math.random() > 0.5 ? 'FUORI TEMPO!' : 'PIATTAFORMA APERTA!';
    registerFail(reason);
  }
}

function finishGame() {
  stopAnimation();
  isPlaying.value = false;
  gameOver.value = true;
  feedbackState.value = '';
  feedbackMessage.value = '';
  const config = currentLevelConfig.value;
  levelPassed.value = successCount.value >= config.requiredSuccess;
  if (levelPassed.value && !completedLevels.value.includes(currentLevel.value)) {
    completedLevels.value = [...completedLevels.value, currentLevel.value];
  }
  levelComplete.value = true;
  statusLabel.value = levelPassed.value
    ? 'Livello superato!'
    : 'Non hai raggiunto le ricezioni necessarie.';

  if (levelPassed.value && rewardType.value === 'badge') {
    playSound('cheer');
    triggerConfetti();
  }

  emit('game-finished', {
    level: currentLevel.value,
    attempts: attempts.value,
    successCount: successCount.value,
    reward: rewardType.value,
  });
}

function triggerConfetti() {
  const palette = ['#22c55e', '#16a34a', '#10b981', '#bbf7d0'];
  confettiPieces.value = Array.from({ length: 26 }).map((_, index) => ({
    id: `${Date.now()}-${index}`,
    left: `${8 + Math.random() * 84}%`,
    delay: `${Math.random() * 0.25}s`,
    color: palette[index % palette.length],
  }));
  window.setTimeout(() => {
    confettiPieces.value = [];
  }, 2000);
}

function goToNextLevel() {
  if (currentLevel.value >= levelConfigs.length) {
    return;
  }
  currentLevel.value += 1;
  resetState();
  startGame();
}

function startGame() {
  if (!props.enabled) {
    return;
  }
  resetState();
  isPlaying.value = true;
  launchBall();
}

onBeforeUnmount(() => {
  stopAnimation();
  if (pauseTimer.value) {
    window.clearTimeout(pauseTimer.value);
    pauseTimer.value = null;
  }
  if (targetHintTimer.value) {
    window.clearTimeout(targetHintTimer.value);
    targetHintTimer.value = null;
  }
});
</script>

<style scoped>
.perfect-dig {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 0.5rem;
}

.perfect-dig__eyebrow {
  margin: 0;
  font-size: 0.75rem;
  letter-spacing: 0.32em;
  text-transform: uppercase;
  color: #22d3ee;
}

.perfect-dig__title {
  margin: 0;
  font-size: 1.4rem;
  letter-spacing: 0.05em;
  color: #f8fafc;
}

.perfect-dig__subtitle {
  margin: 0;
  color: #cbd5e1;
  line-height: 1.5;
}

.perfect-dig__level-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.perfect-dig__level-label {
  color: #38bdf8;
  font-weight: 800;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.perfect-dig__level-progress {
  display: inline-flex;
  gap: 0.35rem;
  color: #94a3b8;
  font-weight: 800;
}

.perfect-dig__level-dot {
  min-width: 18px;
  text-align: center;
  padding: 3px 6px;
  border-radius: 999px;
  border: 1px solid rgba(148, 163, 184, 0.4);
  background: rgba(15, 23, 42, 0.5);
  color: #94a3b8;
  box-shadow: inset 0 1px 1px rgba(255, 255, 255, 0.08);
}

.perfect-dig__level-dot--active {
  border-color: rgba(56, 189, 248, 0.8);
  color: #e0f2fe;
  box-shadow: 0 0 0 4px rgba(56, 189, 248, 0.16);
}

.perfect-dig__level-dot--done {
  background: linear-gradient(135deg, #22c55e, #16a34a);
  border-color: rgba(34, 197, 94, 0.7);
  color: #022c14;
  box-shadow: 0 0 0 4px rgba(34, 197, 94, 0.2);
}

.perfect-dig__field {
  position: relative;
  height: 320px;
  border-radius: 18px;
  background: linear-gradient(180deg, #1e293b, #0f172a);
  border: 1px solid rgba(148, 163, 184, 0.35);
  overflow: hidden;
  cursor: pointer;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.06), 0 20px 40px rgba(0, 0, 0, 0.3);
}

.perfect-dig__field::before {
  content: '';
  position: absolute;
  inset: 0;
  background: repeating-linear-gradient(
      90deg,
      rgba(255, 255, 255, 0.06) 0,
      rgba(255, 255, 255, 0.06) 8px,
      transparent 8px,
      transparent 34px
    ),
    linear-gradient(180deg, rgba(255, 255, 255, 0.04), rgba(255, 255, 255, 0.08));
  mix-blend-mode: overlay;
  pointer-events: none;
}

.perfect-dig__field::after {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(
    135deg,
    rgba(255, 255, 255, 0.02),
    rgba(255, 255, 255, 0.04) 30%,
    rgba(255, 255, 255, 0.02)
  );
  pointer-events: none;
}

.perfect-dig__field--success {
  box-shadow: inset 0 0 0 2px rgba(34, 197, 94, 0.45), 0 20px 48px rgba(34, 197, 94, 0.2);
}

.perfect-dig__field--fail {
  animation: perfectDigFieldFlash 320ms ease-out;
}

.perfect-dig__field--disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.perfect-dig__court {
  position: absolute;
  inset: 0;
  background: linear-gradient(180deg, rgba(229, 174, 85, 0.6), rgba(173, 119, 60, 0.7)),
    repeating-linear-gradient(
      90deg,
      rgba(255, 255, 255, 0.05) 0,
      rgba(255, 255, 255, 0.05) 2px,
      transparent 2px,
      transparent 14px
    );
  filter: saturate(0.9);
}

.perfect-dig__line {
  position: absolute;
  left: 0;
  right: 0;
  height: 3px;
  background: rgba(255, 255, 255, 0.75);
}

.perfect-dig__line--base {
  bottom: 12%;
}

.perfect-dig__line--attack {
  bottom: 32%;
  height: 2px;
  opacity: 0.7;
}

.perfect-dig__line--center {
  bottom: 52%;
  height: 2px;
  opacity: 0.55;
}

.perfect-dig__target {
  position: absolute;
  top: 70%;
  height: 18%;
  border-radius: 18px;
  background: linear-gradient(180deg, rgba(34, 197, 94, 0.18), rgba(22, 163, 74, 0.28));
  border: 1px dashed rgba(34, 197, 94, 0.55);
  box-shadow: 0 0 0 12px rgba(34, 197, 94, 0.08);
  pointer-events: none;
  transition: opacity 220ms ease, box-shadow 220ms ease;
}

.perfect-dig__target--hidden {
  opacity: 0.05;
  box-shadow: none;
  border-style: dotted;
}

.perfect-dig__target-label {
  position: absolute;
  top: -0.75rem;
  left: 50%;
  transform: translateX(-50%);
  background: #0f172a;
  color: #22c55e;
  padding: 0.2rem 0.65rem;
  border-radius: 999px;
  font-size: 0.8rem;
  border: 1px solid rgba(34, 197, 94, 0.6);
  box-shadow: 0 8px 16px rgba(22, 163, 74, 0.2);
}

.perfect-dig__arms {
  position: absolute;
  left: 50%;
  bottom: 6%;
  width: 160px;
  height: 120px;
  transform: translateX(-50%) translateY(12px);
  display: flex;
  justify-content: space-between;
  filter: drop-shadow(0 8px 16px rgba(0, 0, 0, 0.3));
}

.perfect-dig__arm {
  width: 68px;
  height: 120px;
  background: linear-gradient(180deg, #fefefe, #f2c8a2);
  border-radius: 32px 32px 18px 18px;
  position: relative;
  transform-origin: top;
  animation: perfectDigArmRise 240ms ease-out;
}

.perfect-dig__arm::after {
  content: '';
  position: absolute;
  bottom: -2px;
  left: 50%;
  transform: translateX(-50%);
  width: 82px;
  height: 20px;
  background: linear-gradient(90deg, #fbbf77, #f59e0b, #fbbf77);
  border-radius: 999px;
  box-shadow: inset 0 2px 4px rgba(0, 0, 0, 0.18);
}

.perfect-dig__arm--left {
  transform: rotate(-6deg) translateY(4px);
  background: linear-gradient(180deg, #fefefe, #f1c7a0);
}

.perfect-dig__arm--right {
  transform: rotate(6deg) translateY(4px);
}

.perfect-dig__ball {
  position: absolute;
  width: 66px;
  height: 66px;
  border-radius: 50%;
  background: radial-gradient(circle at 30% 30%, #fef3c7, #fbbf24 55%, #f59e0b 100%);
  box-shadow: 0 10px 22px rgba(0, 0, 0, 0.25), 0 0 0 4px rgba(34, 197, 235, 0.2);
  display: grid;
  place-items: center;
  transition: transform 140ms ease, box-shadow 140ms ease;
}

.perfect-dig__ball-band {
  position: absolute;
  inset: 16%;
  border-radius: 50%;
  mix-blend-mode: multiply;
}

.perfect-dig__ball-band--one {
  background: radial-gradient(circle at 80% 20%, rgba(14, 165, 233, 0.85), rgba(14, 116, 144, 0.65));
}

.perfect-dig__ball-band--two {
  background: radial-gradient(circle at 20% 80%, rgba(37, 99, 235, 0.9), rgba(30, 64, 175, 0.75));
  transform: rotate(40deg);
}

.perfect-dig__ball--active {
  animation: perfectDigBallFloat 2s ease-in-out infinite;
}

.perfect-dig__ball--success {
  box-shadow: 0 0 0 10px rgba(34, 197, 94, 0.35), 0 16px 32px rgba(34, 197, 94, 0.35);
  transform: translate(-50%, -50%) scale(1.05);
}

.perfect-dig__ball--fail {
  box-shadow: 0 0 0 10px rgba(248, 113, 113, 0.25), 0 16px 32px rgba(248, 113, 113, 0.35);
  transform: translate(-50%, -50%) scale(0.96);
}

.perfect-dig__feedback {
  position: absolute;
  bottom: 0.85rem;
  left: 50%;
  transform: translateX(-50%);
  padding: 0.55rem 0.95rem;
  border-radius: 12px;
  background: rgba(15, 23, 42, 0.7);
  border: 1px solid rgba(255, 255, 255, 0.1);
  color: #e2e8f0;
  font-weight: 800;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  backdrop-filter: blur(4px);
}

.perfect-dig__feedback--success {
  color: #22c55e;
  border-color: rgba(34, 197, 94, 0.6);
}

.perfect-dig__feedback--fail {
  color: #f87171;
  border-color: rgba(248, 113, 113, 0.6);
}

.perfect-dig-feedback-enter-active,
.perfect-dig-feedback-leave-active {
  transition: opacity 160ms ease, transform 160ms ease;
}

.perfect-dig-feedback-enter-from,
.perfect-dig-feedback-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(6px);
}

.perfect-dig-arms-enter-active,
.perfect-dig-arms-leave-active {
  transition: opacity 220ms ease, transform 220ms ease;
}

.perfect-dig-arms-enter-from,
.perfect-dig-arms-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(20px);
}

.perfect-dig__info {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.perfect-dig__attempts {
  margin: 0;
  color: #e2e8f0;
  font-weight: 700;
  letter-spacing: 0.04em;
}

.perfect-dig__status {
  margin: 0;
  color: #94a3b8;
  font-size: 0.9rem;
}

.perfect-dig__status--alert {
  color: #fca5a5;
}

.perfect-dig__summary {
  padding: 0.9rem 1rem;
  border-radius: 16px;
  background: linear-gradient(135deg, rgba(15, 23, 42, 0.9), rgba(34, 197, 94, 0.16));
  border: 1px solid rgba(34, 197, 94, 0.35);
  box-shadow: 0 12px 24px rgba(0, 0, 0, 0.25);
}

.perfect-dig__summary-title {
  margin: 0 0 0.25rem;
  color: #e0f2fe;
  font-weight: 800;
}

.perfect-dig__summary-stats {
  margin: 0;
  color: #f8fafc;
  font-weight: 800;
}

.perfect-dig__summary-reward {
  margin: 0.35rem 0 0;
  color: #e2e8f0;
}

.perfect-dig__level-end {
  padding: 1rem 1.1rem;
  border-radius: 14px;
  background: linear-gradient(135deg, rgba(34, 197, 94, 0.12), rgba(59, 130, 246, 0.12));
  border: 1px solid rgba(255, 255, 255, 0.08);
  box-shadow: 0 12px 26px rgba(0, 0, 0, 0.28);
}

.perfect-dig__level-end-title {
  margin: 0 0 0.25rem;
  color: #e0f2fe;
  font-weight: 800;
}

.perfect-dig__level-end-subtitle {
  margin: 0 0 0.75rem;
  color: #cbd5e1;
}

.perfect-dig__level-end-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.6rem;
}

.perfect-dig__actions {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
  margin-top: 0.25rem;
}

.perfect-dig__primary {
  padding: 0.95rem 1rem;
  border-radius: 12px;
  border: 1px solid rgba(59, 130, 246, 0.5);
  background: linear-gradient(135deg, #38bdf8, #2563eb);
  color: #0b1224;
  font-weight: 800;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  box-shadow: 0 16px 28px rgba(37, 99, 235, 0.35);
  transition: transform 150ms ease, box-shadow 150ms ease, opacity 150ms ease;
}

.perfect-dig__primary:disabled {
  opacity: 0.65;
  cursor: not-allowed;
  box-shadow: none;
}

.perfect-dig__primary:not(:disabled):active {
  transform: translateY(1px) scale(0.99);
}

.perfect-dig__secondary {
  padding: 0.95rem 1rem;
  border-radius: 12px;
  border: 1px solid rgba(148, 163, 184, 0.4);
  background: rgba(15, 23, 42, 0.7);
  color: #e2e8f0;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  transition: border-color 150ms ease, color 150ms ease, opacity 150ms ease;
}

.perfect-dig__secondary:hover {
  border-color: rgba(226, 232, 240, 0.65);
  color: #f8fafc;
}

.perfect-dig__confetti {
  position: absolute;
  inset: 0;
  overflow: hidden;
  pointer-events: none;
}

.perfect-dig__confetti-piece {
  position: absolute;
  top: -8%;
  width: 10px;
  height: 14px;
  border-radius: 4px;
  animation: perfectDigConfetti 0.9s ease-in forwards;
}

@keyframes perfectDigBallFloat {
  0% {
    transform: translate(-50%, -50%) translateY(0);
  }
  50% {
    transform: translate(-50%, -50%) translateY(-6px);
  }
  100% {
    transform: translate(-50%, -50%) translateY(0);
  }
}

@keyframes perfectDigFieldFlash {
  0% {
    box-shadow: inset 0 0 0 0 rgba(248, 113, 113, 0.6);
  }
  80% {
    box-shadow: inset 0 0 0 6px rgba(248, 113, 113, 0.4);
  }
  100% {
    box-shadow: inset 0 0 0 0 rgba(248, 113, 113, 0.15);
  }
}

@keyframes perfectDigArmRise {
  0% {
    transform: translateY(26px) scale(0.96);
    opacity: 0;
  }
  100% {
    transform: translateY(0) scale(1);
    opacity: 1;
  }
}

@keyframes perfectDigConfetti {
  0% {
    transform: translateY(0) rotate(0deg) scale(0.9);
    opacity: 1;
  }
  100% {
    transform: translateY(320px) rotate(340deg) scale(1.1);
    opacity: 0;
  }
}

@media (max-width: 768px) {
  .perfect-dig__field {
    height: 340px;
  }

  .perfect-dig__info {
    flex-direction: column;
    align-items: flex-start;
  }

  .perfect-dig__actions {
    grid-template-columns: 1fr;
  }
}
</style>
