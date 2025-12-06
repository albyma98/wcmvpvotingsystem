<template>
  <div class="libero-reflex">
    <p class="libero-reflex__eyebrow">Mini gioco</p>
    <h4 class="libero-reflex__title">Libero Reflex</h4>
    <p class="libero-reflex__subtitle">
      Tre livelli di difficoltà: tocca lo schermo solo quando la palla entra nella zona verde!
    </p>

    <div class="libero-reflex__level-header">
      <span class="libero-reflex__level-label">Livello {{ currentLevel }} / 3</span>
      <div class="libero-reflex__level-progress" role="presentation">
        <span
          v-for="level in levelConfigs"
          :key="level.id"
          class="libero-reflex__level-dot"
          :class="{
            'libero-reflex__level-dot--done': completedLevels.includes(level.id),
            'libero-reflex__level-dot--active': level.id === currentLevel,
          }"
        />
      </div>
    </div>

    <div
      class="libero-reflex__field"
      :class="{
        'libero-reflex__field--success': feedbackState === 'success',
        'libero-reflex__field--fail': feedbackState === 'fail',
        'libero-reflex__field--disabled': !props.enabled,
      }"
      role="button"
      tabindex="0"
      @pointerdown.prevent="handleTap"
      @keydown.space.prevent="handleTap"
      @keydown.enter.prevent="handleTap"
    >
      <div class="libero-reflex__zone" :style="zoneStyle">
        <span class="libero-reflex__zone-label">Zona verde</span>
      </div>
      <div class="libero-reflex__ball" :class="ballClasses" :style="ballStyle">
        <span v-if="feedbackState === 'success'" class="libero-reflex__ball-text">BLOCK!</span>
        <span v-else-if="feedbackState === 'fail'" class="libero-reflex__ball-text">OUT!</span>
      </div>
      <Transition name="libero-reflex-feedback">
        <div v-if="feedbackMessage" class="libero-reflex__feedback" :class="feedbackClasses">
          {{ feedbackMessage }}
        </div>
      </Transition>
    </div>

    <div class="libero-reflex__info">
      <p class="libero-reflex__attempts">Tentativi: {{ attempts }} / {{ maxAttempts }}</p>
      <p class="libero-reflex__status" :class="{ 'libero-reflex__status--alert': feedbackState === 'fail' }">
        {{ statusLabel }}
      </p>
    </div>

    <div v-if="gameOver" class="libero-reflex__summary" aria-live="polite">
      <p class="libero-reflex__summary-title">Riepilogo</p>
      <p class="libero-reflex__summary-stats">Tentativi riusciti: {{ successCount }} / {{ maxAttempts }}</p>
      <p class="libero-reflex__summary-reward">{{ rewardMessage }}</p>
    </div>

    <div v-if="levelComplete" class="libero-reflex__level-end" aria-live="polite">
      <p class="libero-reflex__level-end-title">{{ levelEndTitle }}</p>
      <p class="libero-reflex__level-end-subtitle">{{ levelEndSubtitle }}</p>
      <div class="libero-reflex__level-end-actions">
        <button
          v-if="showNextLevelCta"
          type="button"
          class="libero-reflex__primary"
          @click="goToNextLevel"
        >
          {{ nextLevelLabel }}
        </button>
        <button type="button" class="libero-reflex__secondary" @click="emit('close')">
          Chiudi
        </button>
      </div>
    </div>

    <div class="libero-reflex__actions">
      <button
        type="button"
        class="libero-reflex__primary"
        :disabled="!props.enabled || isPlaying"
        @click="startGame"
      >
        {{ gameCtaLabel }}
      </button>
      <button type="button" class="libero-reflex__secondary" @click="emit('close')">Chiudi</button>
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
const levelConfigs = [
  {
    id: 1,
    baseSpeed: 0.07,
    randomBoost: 0.02,
    zoneStart: 33,
    zoneEnd: 67,
    requiredSuccess: 2,
    completionTitle: 'Livello 1 completato',
    completionSubtitle: '+1 chance extra sbloccata. Vuoi salire di livello?',
  },
  {
    id: 2,
    baseSpeed: 0.09,
    randomBoost: 0.025,
    zoneStart: 38,
    zoneEnd: 62,
    requiredSuccess: 2,
    completionTitle: 'Livello 2 completato',
    completionSubtitle: '+1 chance extra pronta! Passa al livello finale.',
  },
  {
    id: 3,
    baseSpeed: 0.105,
    randomBoost: 0.03,
    zoneStart: 42,
    zoneEnd: 58,
    requiredSuccess: 3,
    completionTitle: 'Sei un Libero d\'Acciaio!',
    completionSubtitle: 'Percorso perfetto! Badge speciale sbloccato.',
  },
];
const attempts = ref(0);
const successCount = ref(0);
const isPlaying = ref(false);
const gameOver = ref(false);
const feedbackMessage = ref('');
const feedbackState = ref('');
const statusLabel = ref('3 livelli: blocca la palla solo nel verde!');
const ballPosition = ref(10);
const ballDirection = ref(1);
const baseSpeed = ref(0.08);
const currentLevel = ref(1);
const levelComplete = ref(false);
const levelPassed = ref(false);
const completedLevels = ref([]);
const pauseTimer = ref(null);
let animationFrame = null;
let lastTs = 0;

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
  const config = currentLevelConfig.value;
  const passedLevel = successCount.value >= config.requiredSuccess;
  if (!passedLevel) {
    return 'none';
  }

  if (currentLevel.value === 3 && successCount.value === maxAttempts) {
    return 'badge';
  }

  return 'chance';
});

const levelEndTitle = computed(() => {
  if (!levelPassed.value) {
    return 'Ritenta il livello';
  }
  return currentLevelConfig.value.completionTitle;
});

const levelEndSubtitle = computed(() => {
  if (!levelPassed.value) {
    return 'Ti servono almeno 2 blocchi su 3 (o 3/3 nel livello finale). Riprova!';
  }
  if (currentLevel.value === 3 && rewardType.value === 'badge') {
    return 'Sei un Libero d\'Acciaio! Badge e chance extra sbloccati.';
  }
  return currentLevelConfig.value.completionSubtitle;
});

const showNextLevelCta = computed(() => levelPassed.value && currentLevel.value < levelConfigs.length);

const rewardMessage = computed(() => {
  if (rewardType.value === 'badge') {
    return 'Perfetto! Hai i riflessi di un vero libero! 🛡️ Libero d\'Acciaio sbloccato';
  }
  if (rewardType.value === 'chance') {
    return 'Ottimo! +1 chance extra per questo livello.';
  }
  return 'Quasi! Riprova per ottenere la chance.';
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
  left: `${ballPosition.value}%`,
}));

const ballClasses = computed(() => ({
  'libero-reflex__ball--active': isPlaying.value,
  'libero-reflex__ball--success': feedbackState.value === 'success',
  'libero-reflex__ball--fail': feedbackState.value === 'fail',
}));

const feedbackClasses = computed(() => ({
  'libero-reflex__feedback--success': feedbackState.value === 'success',
  'libero-reflex__feedback--fail': feedbackState.value === 'fail',
}));

const zoneStyle = computed(() => ({
  left: `${currentLevelConfig.value.zoneStart}%`,
  width: `${currentLevelConfig.value.zoneEnd - currentLevelConfig.value.zoneStart}%`,
}));

function resetState() {
  attempts.value = 0;
  successCount.value = 0;
  gameOver.value = false;
  levelComplete.value = false;
  levelPassed.value = false;
  feedbackMessage.value = '';
  feedbackState.value = '';
  statusLabel.value = '3 livelli: blocca la palla solo nel verde!';
  ballPosition.value = 10 + Math.random() * 80;
  ballDirection.value = Math.random() > 0.5 ? 1 : -1;
  baseSpeed.value =
    currentLevelConfig.value.baseSpeed + Math.random() * currentLevelConfig.value.randomBoost;
}

function stopAnimation() {
  if (animationFrame) {
    window.cancelAnimationFrame(animationFrame);
    animationFrame = null;
  }
  lastTs = 0;
}

function animateBall(timestamp) {
  if (!isPlaying.value || gameOver.value) {
    return;
  }
  if (!lastTs) {
    lastTs = timestamp;
  }
  const delta = Math.min(48, timestamp - lastTs);
  lastTs = timestamp;
  const speed = baseSpeed.value * delta * (0.9 + Math.random() * 0.2);
  let next = ballPosition.value + ballDirection.value * speed;

  if (next <= 4) {
    next = 4;
    ballDirection.value = 1;
  } else if (next >= 96) {
    next = 96;
    ballDirection.value = -1;
  }

  ballPosition.value = next;
  animationFrame = window.requestAnimationFrame(animateBall);
}

function startAnimation() {
  stopAnimation();
  animationFrame = window.requestAnimationFrame(animateBall);
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
  levelComplete.value = levelPassed.value;
  statusLabel.value = levelPassed.value
    ? 'Livello superato!'
    : 'Non hai raggiunto i blocchi necessari.';
  emit('game-finished', {
    level: currentLevel.value,
    attempts: attempts.value,
    successCount: successCount.value,
    reward: rewardType.value,
  });
}

function scheduleResume() {
  if (pauseTimer.value) {
    window.clearTimeout(pauseTimer.value);
  }
  pauseTimer.value = window.setTimeout(() => {
    feedbackState.value = '';
    feedbackMessage.value = '';
    if (attempts.value >= maxAttempts) {
      finishGame();
      return;
    }
    if (!gameOver.value) {
      startAnimation();
    }
  }, 650);
}

function applyVibration() {
  if (typeof navigator !== 'undefined' && typeof navigator.vibrate === 'function') {
    navigator.vibrate(120);
  }
}

function handleTap() {
  if (!isPlaying.value || gameOver.value || !props.enabled) {
    return;
  }

  attempts.value += 1;
  const isInside =
    ballPosition.value >= currentLevelConfig.value.zoneStart &&
    ballPosition.value <= currentLevelConfig.value.zoneEnd;
  stopAnimation();

  if (isInside) {
    successCount.value += 1;
    feedbackState.value = 'success';
    feedbackMessage.value = 'Ottimo blocco!';
    statusLabel.value = 'Colpo giusto! Preparati al prossimo.';
  } else {
    feedbackState.value = 'fail';
    feedbackMessage.value = 'Fuori tempo!';
    statusLabel.value = 'Resta concentrato, aspetta il verde.';
    applyVibration();
  }

  if (attempts.value >= maxAttempts) {
    scheduleResume();
    return;
  }

  scheduleResume();
}

function startGame() {
  if (!props.enabled) {
    return;
  }
  resetState();
  isPlaying.value = true;
  statusLabel.value = 'Muovi le dita: tocca solo nel verde!';
  startAnimation();
}

function goToNextLevel() {
  if (currentLevel.value >= levelConfigs.length) {
    return;
  }
  currentLevel.value += 1;
  resetState();
  startGame();
}

onBeforeUnmount(() => {
  stopAnimation();
  if (pauseTimer.value) {
    window.clearTimeout(pauseTimer.value);
    pauseTimer.value = null;
  }
});
</script>

<style scoped>
.libero-reflex {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 0.5rem;
}

.libero-reflex__eyebrow {
  margin: 0;
  font-size: 0.75rem;
  letter-spacing: 0.32em;
  text-transform: uppercase;
  color: #22d3ee;
}

.libero-reflex__title {
  margin: 0;
  font-size: 1.4rem;
  letter-spacing: 0.05em;
  color: #f8fafc;
}

.libero-reflex__subtitle {
  margin: 0;
  color: #cbd5e1;
  line-height: 1.5;
}

.libero-reflex__level-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.libero-reflex__level-label {
  color: #38bdf8;
  font-weight: 800;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.libero-reflex__level-progress {
  display: inline-flex;
  gap: 0.35rem;
}

.libero-reflex__level-dot {
  width: 12px;
  height: 12px;
  border-radius: 999px;
  background: rgba(148, 163, 184, 0.4);
  border: 1px solid rgba(148, 163, 184, 0.4);
  box-shadow: inset 0 1px 2px rgba(255, 255, 255, 0.12);
}

.libero-reflex__level-dot--active {
  background: #22d3ee;
  border-color: #38bdf8;
  box-shadow: 0 0 0 3px rgba(56, 189, 248, 0.2);
}

.libero-reflex__level-dot--done {
  background: linear-gradient(135deg, #22c55e, #16a34a);
  border-color: rgba(34, 197, 94, 0.8);
  box-shadow: 0 0 0 3px rgba(34, 197, 94, 0.2);
}

.libero-reflex__field {
  position: relative;
  height: 280px;
  border-radius: 18px;
  background: linear-gradient(180deg, rgba(15, 23, 42, 0.9), rgba(30, 41, 59, 0.95));
  border: 1px solid rgba(148, 163, 184, 0.35);
  overflow: hidden;
  cursor: pointer;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.06), 0 20px 40px rgba(0, 0, 0, 0.3);
}

.libero-reflex__field::before {
  content: '';
  position: absolute;
  inset: 0;
  background: radial-gradient(circle at 50% 20%, rgba(255, 255, 255, 0.06), transparent 60%);
  pointer-events: none;
}

.libero-reflex__field--success {
  box-shadow: inset 0 0 0 2px rgba(34, 197, 94, 0.45), 0 20px 48px rgba(34, 197, 94, 0.2);
}

.libero-reflex__field--fail {
  animation: liberoFieldFlash 320ms ease-out;
}

.libero-reflex__field--disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.libero-reflex__zone {
  position: absolute;
  inset: 18%;
  border-radius: 18px;
  background: linear-gradient(180deg, rgba(34, 197, 94, 0.18), rgba(22, 163, 74, 0.28));
  border: 1px dashed rgba(34, 197, 94, 0.55);
  pointer-events: none;
}

.libero-reflex__zone-label {
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

.libero-reflex__ball {
  position: absolute;
  top: 50%;
  width: 62px;
  height: 62px;
  margin-top: -31px;
  border-radius: 50%;
  background: radial-gradient(circle at 30% 30%, #fef3c7, #f59e0b 55%, #d97706 100%);
  box-shadow: 0 10px 22px rgba(0, 0, 0, 0.25), 0 0 0 4px rgba(251, 191, 36, 0.18);
  transform: translateX(-50%);
  display: grid;
  place-items: center;
  color: #0b1224;
  font-weight: 800;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  transition: transform 140ms ease, box-shadow 140ms ease;
}

.libero-reflex__ball--active {
  animation: liberoBallFloat 2.6s ease-in-out infinite;
}

.libero-reflex__ball--success {
  box-shadow: 0 0 0 8px rgba(34, 197, 94, 0.3), 0 16px 32px rgba(34, 197, 94, 0.35);
  transform: translateX(-50%) scale(1.06);
}

.libero-reflex__ball--fail {
  box-shadow: 0 0 0 8px rgba(248, 113, 113, 0.3), 0 16px 32px rgba(248, 113, 113, 0.35);
  transform: translateX(-50%) scale(0.97);
}

.libero-reflex__ball-text {
  font-size: 0.75rem;
}

.libero-reflex__feedback {
  position: absolute;
  bottom: 0.85rem;
  left: 50%;
  transform: translateX(-50%);
  padding: 0.5rem 0.85rem;
  border-radius: 12px;
  background: rgba(15, 23, 42, 0.7);
  border: 1px solid rgba(255, 255, 255, 0.1);
  color: #e2e8f0;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  backdrop-filter: blur(4px);
}

.libero-reflex__feedback--success {
  color: #22c55e;
  border-color: rgba(34, 197, 94, 0.6);
}

.libero-reflex__feedback--fail {
  color: #f87171;
  border-color: rgba(248, 113, 113, 0.6);
}

.libero-reflex-feedback-enter-active,
.libero-reflex-feedback-leave-active {
  transition: opacity 160ms ease, transform 160ms ease;
}

.libero-reflex-feedback-enter-from,
.libero-reflex-feedback-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(6px);
}

.libero-reflex__info {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.libero-reflex__attempts {
  margin: 0;
  color: #e2e8f0;
  font-weight: 700;
  letter-spacing: 0.04em;
}

.libero-reflex__status {
  margin: 0;
  color: #94a3b8;
  font-size: 0.9rem;
}

.libero-reflex__status--alert {
  color: #fca5a5;
}

.libero-reflex__summary {
  padding: 0.9rem 1rem;
  border-radius: 16px;
  background: linear-gradient(135deg, rgba(15, 23, 42, 0.9), rgba(34, 197, 94, 0.16));
  border: 1px solid rgba(34, 197, 94, 0.35);
  box-shadow: 0 16px 32px rgba(34, 197, 94, 0.12);
}

.libero-reflex__summary-title {
  margin: 0 0 0.35rem;
  color: #bbf7d0;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.libero-reflex__summary-stats {
  margin: 0;
  color: #f8fafc;
  font-weight: 800;
}

.libero-reflex__summary-reward {
  margin: 0.35rem 0 0;
  color: #e2e8f0;
}

.libero-reflex__level-end {
  padding: 1rem 1.1rem;
  border-radius: 14px;
  background: linear-gradient(135deg, rgba(34, 197, 94, 0.12), rgba(59, 130, 246, 0.12));
  border: 1px solid rgba(255, 255, 255, 0.08);
  box-shadow: 0 12px 26px rgba(0, 0, 0, 0.28);
}

.libero-reflex__level-end-title {
  margin: 0 0 0.25rem;
  color: #e0f2fe;
  font-weight: 800;
}

.libero-reflex__level-end-subtitle {
  margin: 0 0 0.75rem;
  color: #cbd5e1;
}

.libero-reflex__level-end-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.6rem;
}

.libero-reflex__actions {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
  margin-top: 0.25rem;
}

.libero-reflex__primary {
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

.libero-reflex__primary:disabled {
  opacity: 0.65;
  cursor: not-allowed;
  box-shadow: none;
}

.libero-reflex__primary:not(:disabled):active {
  transform: translateY(1px) scale(0.99);
}

.libero-reflex__secondary {
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

.libero-reflex__secondary:hover {
  border-color: rgba(226, 232, 240, 0.65);
  color: #f8fafc;
}

@keyframes liberoBallFloat {
  0% {
    transform: translateX(-50%) translateY(0);
  }
  50% {
    transform: translateX(-50%) translateY(-6px);
  }
  100% {
    transform: translateX(-50%) translateY(0);
  }
}

@keyframes liberoFieldFlash {
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

@media (max-width: 768px) {
  .libero-reflex__field {
    height: 320px;
  }

  .libero-reflex__info {
    flex-direction: column;
    align-items: flex-start;
  }

  .libero-reflex__actions {
    grid-template-columns: 1fr;
  }
}
</style>
