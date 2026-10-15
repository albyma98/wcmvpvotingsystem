<template>
  <div class="quick-quiz flex h-full min-h-0 flex-col overflow-hidden" :class="{ 'quick-quiz--club': completionReward > 0 }">
    <div class="mb-3 flex items-center justify-between text-sm text-slate-200">
      <span>Domanda {{ progressLabel }}</span>
      <span class="rounded-full px-3 py-1 font-bold" :class="timerClass">{{ remainingSeconds }}s</span>
    </div>

    <div class="mb-4 h-2 overflow-hidden rounded-full bg-white/10">
      <div class="h-full bg-emerald-400 transition-all duration-300" :style="{ width: `${progressPercent}%` }" />
    </div>

    <transition name="quiz-fade-slide" mode="out-in">
      <div v-if="state === 'loading'" key="loading" class="flex flex-1 items-center justify-center text-slate-300">Caricamento quiz…</div>

      <div v-else-if="state === 'unavailable'" key="unavailable" class="flex flex-1 flex-col items-center justify-center gap-4 text-center text-white">
        <p>{{ errorMessage || 'Il quiz non è disponibile in questo momento.' }}</p>
        <button type="button" class="answer-btn" @click="emit('exit')">Torna alle attività</button>
      </div>

      <div v-else-if="state === 'question'" key="question" class="flex flex-1 flex-col">
        <div class="question-card mb-4 rounded-2xl border border-white/20 bg-white/10 p-5 text-lg font-bold text-white backdrop-blur">{{ currentQuestion?.text }}</div>
        <div class="grid gap-3">
          <button
            v-for="(answer, index) in currentQuestion?.answers || []"
            :key="`${currentQuestion?.id}-${index}`"
            type="button"
            class="answer-btn"
            :disabled="isSubmitting"
            @click="submitAnswer(index)"
          >
            {{ answer }}
          </button>
        </div>
        <div v-if="errorMessage" class="mt-3 text-sm text-white" role="alert">
          <p>{{ errorMessage }}</p>
          <button type="button" class="answer-btn mt-2" :disabled="isSubmitting" @click="submitAnswer(pendingAnswer)">Riprova l'invio</button>
        </div>
      </div>

      <div v-else-if="state === 'feedback'" key="feedback" class="flex flex-1 items-center justify-center">
        <div :class="['feedback-card', lastFeedback?.isCorrect ? 'correct' : 'wrong']">
          <p class="text-2xl font-black">{{ lastFeedback?.isCorrect ? 'Corretto! ✅' : 'Sbagliato ❌' }}</p>
          <p class="mt-2 text-slate-100">{{ completionReward > 0 ? 'Risposta registrata' : `+${lastFeedback?.coinsEarned || 0} monete` }}</p>
          <p v-if="streak > 1" class="streak mt-3">Streak x{{ streak }}</p>
        </div>
      </div>

      <div v-else key="summary" class="flex flex-1 flex-col items-center justify-center text-center text-white">
        <p class="text-sm uppercase tracking-widest text-slate-300">Quiz terminato</p>
        <h3 class="mt-2 text-4xl font-black">{{ alreadyRewarded ? 'Ben giocato!' : `+${totalCoins} monete` }}</h3>
        <p v-if="completionReward > 0" class="mt-2 text-sm text-slate-200">{{ alreadyRewarded ? 'Hai già ricevuto il bonus per questa partita.' : 'Bonus per aver completato il quiz' }}</p>
        <button type="button" :disabled="claiming" class="mt-6 rounded-full bg-amber-400 px-7 py-3 text-base font-black text-slate-900 disabled:opacity-50" @click="claimCoins">{{ claiming ? 'Salvataggio…' : alreadyRewarded ? 'Torna alle attività' : 'Ritira le monete' }}</button>
      </div>
    </transition>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { fetchQuickQuiz, submitQuickQuizAnswer } from '../api';
import { getOrCreateDeviceId } from '../deviceId';

const props = defineProps({
  eventId: { type: Number, required: true },
  completionReward: { type: Number, default: 0 },
  claiming: { type: Boolean, default: false },
  alreadyRewarded: { type: Boolean, default: false },
});
const emit = defineEmits(['claim', 'exit']);

const state = ref('loading');
const questions = ref([]);
const config = ref({ questions_per_session: 5, seconds_per_question: 8, completion_bonus: 0, streak_bonus: 0 });
const index = ref(0);
const remainingSeconds = ref(0);
const totalCoins = ref(0);
const streak = ref(0);
const isSubmitting = ref(false);
const startedAt = ref(0);
const lastFeedback = ref(null);
const errorMessage = ref('');
const pendingAnswer = ref(-1);
let timerId;
let feedbackTimerId;
let disposed = false;

const currentQuestion = computed(() => questions.value[index.value] || null);
const progressLabel = computed(() => `${Math.min(index.value + 1, questions.value.length)}/${questions.value.length || 0}`);
const progressPercent = computed(() => (questions.value.length ? (index.value / questions.value.length) * 100 : 0));
const timerClass = computed(() => remainingSeconds.value <= 3 ? 'bg-red-500/80 animate-pulse' : 'bg-white/15');

onMounted(loadQuiz);
onBeforeUnmount(() => {
  disposed = true;
  stopTimer();
  window.clearTimeout(feedbackTimerId);
});

async function loadQuiz() {
  state.value = 'loading';
  const { ok, data } = await fetchQuickQuiz(props.eventId);
  if (disposed) return;
  if (!ok || !Array.isArray(data?.questions) || !data.questions.length) {
    state.value = 'unavailable';
    errorMessage.value = 'Il quiz non è disponibile in questo momento.';
    return;
  }
  config.value = { ...config.value, ...(data.config || {}) };
  questions.value = data.questions;
  index.value = 0;
  totalCoins.value = 0;
  streak.value = 0;
  startQuestion();
}

function startQuestion() {
  state.value = 'question';
  remainingSeconds.value = Number(config.value.seconds_per_question || 8);
  startedAt.value = Date.now();
  stopTimer();
  timerId = window.setInterval(() => {
    remainingSeconds.value -= 1;
    if (remainingSeconds.value <= 0) {
      stopTimer();
      submitAnswer(-1);
    }
  }, 1000);
}

function stopTimer() {
  if (timerId) {
    window.clearInterval(timerId);
    timerId = undefined;
  }
}

async function submitAnswer(selectedIndex) {
  if (!currentQuestion.value || isSubmitting.value) return;
  isSubmitting.value = true;
  errorMessage.value = '';
  pendingAnswer.value = selectedIndex;
  stopTimer();
  const responseMs = Math.max(0, Date.now() - startedAt.value);
  const { ok, data } = await submitQuickQuizAnswer(props.eventId, {
    questionId: currentQuestion.value.id,
    selectedIndex,
    responseMs,
    deviceId: getOrCreateDeviceId(),
    journey: props.completionReward > 0,
  });
  if (disposed) return;
  if (!ok) {
    errorMessage.value = 'La risposta non è stata inviata. Controlla la connessione e riprova.';
    isSubmitting.value = false;
    return;
  }
  const isCorrect = Boolean(data?.is_correct ?? data?.isCorrect);
  const coinsEarned = Math.max(0, Number(data?.coins_earned ?? data?.coinsEarned ?? 0));
  if (!props.completionReward) totalCoins.value += coinsEarned;
  streak.value = isCorrect ? streak.value + 1 : 0;

  lastFeedback.value = { isCorrect, coinsEarned };
  state.value = 'feedback';

  feedbackTimerId = window.setTimeout(() => {
    if (index.value >= questions.value.length - 1) {
      totalCoins.value = props.completionReward > 0
        ? props.completionReward
        : totalCoins.value + Number(config.value.completion_bonus || 0);
      state.value = 'summary';
    } else {
      index.value += 1;
      startQuestion();
    }
    isSubmitting.value = false;
  }, 700);
}

function claimCoins() {
  if (props.claiming || state.value !== 'summary') return;
  if (props.alreadyRewarded) { emit('exit'); return; }
  emit('claim', { coins: totalCoins.value });
}
</script>

<style scoped>
.quick-quiz--club { color: var(--ink); overflow-y: auto; }
.quick-quiz--club :deep(.text-white),
.quick-quiz--club :deep(.text-slate-100),
.quick-quiz--club :deep(.text-slate-200),
.quick-quiz--club :deep(.text-slate-300) { color: var(--ink); }
.quick-quiz--club .question-card { color: var(--ink); background: var(--card); border: 2px solid var(--line); font-family: 'Fraunces', serif; }
.quick-quiz--club .answer-btn { color: var(--ink); background: var(--card); border: 2px solid var(--line); box-shadow: 0 3px 0 rgba(35,22,26,0.1); }
.quick-quiz--club .feedback-card { background: var(--card); color: var(--ink); border: 2px solid var(--red-line); }
.quick-quiz--club .feedback-card.correct { border-color: var(--gold); }
.quiz-fade-slide-enter-active,.quiz-fade-slide-leave-active{transition:all .25s ease}
.quiz-fade-slide-enter-from{opacity:0;transform:translateX(16px)}
.quiz-fade-slide-leave-to{opacity:0;transform:translateX(-16px)}
.answer-btn{padding:.9rem 1rem;border-radius:.9rem;border:1px solid rgba(255,255,255,.2);background:rgba(15,23,42,.55);font-weight:700;text-align:left}
.answer-btn:active{transform:scale(.98)}
.feedback-card{padding:1.4rem;border-radius:1rem;border:1px solid rgba(255,255,255,.2);text-align:center}
.feedback-card.correct{background:rgba(16,185,129,.25);animation:pop .28s ease}
.feedback-card.wrong{background:rgba(239,68,68,.25);animation:shake .28s ease}
.streak{display:inline-block;background:#f59e0b;color:#111827;padding:.2rem .6rem;border-radius:9999px;font-weight:800;animation:pop .3s ease}
@keyframes shake{0%,100%{transform:translateX(0)}25%{transform:translateX(-6px)}75%{transform:translateX(6px)}}
@keyframes pop{0%{transform:scale(.94)}100%{transform:scale(1)}}
</style>
