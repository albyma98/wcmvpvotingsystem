<template>
  <Teleport to="body">
    <Transition name="feedback-modal-fade">
      <div
        v-if="modelValue"
        class="feedback-modal"
        :style="themeStyle"
        role="dialog"
        aria-modal="true"
        aria-labelledby="feedback-modal-title"
        @click.self="close"
      >
        <div class="feedback-modal__panel">
          <div class="feedback-modal__grabber" aria-hidden="true"></div>
          <header class="feedback-modal__header">
            <p class="feedback-modal__step">{{ stepLabel }}</p>
            <button type="button" class="feedback-modal__close" aria-label="Chiudi feedback" :disabled="isSubmitting" @click="close">✕</button>
          </header>

          <div class="feedback-modal__body">
            <h2 id="feedback-modal-title" class="feedback-modal__title">{{ activeQuestion?.title }}</h2>

            <div v-if="isOptionalStep" class="feedback-modal__optional">
              <textarea
                v-model="answers.suggestion"
                class="feedback-modal__input"
                maxlength="80"
                rows="3"
                :placeholder="activeQuestion?.title"
                aria-labelledby="feedback-modal-title"
                :disabled="isSubmitting"
              />
              <p class="feedback-modal__counter">{{ answers.suggestion.length }}/80</p>
            </div>

            <div v-else class="feedback-modal__options">
              <button
                v-for="option in activeQuestion?.options || []"
                :key="option.value"
                type="button"
                class="feedback-modal__option"
                :class="{ active: answers[activeQuestion.answerKey] === option.value }"
                :aria-pressed="answers[activeQuestion.answerKey] === option.value"
                :disabled="isSubmitting"
                @click="onSelectOption(option.value)"
              >
                <span class="feedback-modal__option-icon" aria-hidden="true">{{ option.icon || '•' }}</span>
                <span>{{ option.label }}</span>
              </button>
            </div>

            <p v-if="errorMessage" class="feedback-modal__error" role="alert">{{ errorMessage }}</p>
            <p v-if="successMessage" class="feedback-modal__success" role="status">{{ successMessage }}</p>
          </div>

          <footer class="feedback-modal__footer">
            <button v-if="step > 0" type="button" class="feedback-modal__back" :disabled="isSubmitting" @click="goBack">Indietro</button>
            <div class="feedback-modal__actions">
              <button v-if="isOptionalStep" type="button" class="feedback-modal__skip" :disabled="isSubmitting" @click="skipOptional">Salta</button>
              <button type="button" class="feedback-modal__submit" :disabled="isSubmitting" @click="onContinue">
                {{ isFinalStep ? (isSubmitting ? 'Invio...' : 'Invia') : 'Continua' }}
              </button>
            </div>
          </footer>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue';
import { submitEventFeedback } from '../api';

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false,
  },
  eventId: {
    type: Number,
    default: 0,
  },
  feedbackSurvey: {
    type: Object,
    default: () => ({}),
  },
  themeStyle: {
    type: Object,
    default: () => ({}),
  },
  reportExperienceEnabled: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits(['update:modelValue', 'submitted', 'telemetry']);

const defaultSurvey = {
  questions: [
    {
      id: 'experience',
      title: 'Com\'è stata la tua esperienza oggi?',
      answers: [
        { value: 'awesome', label: 'Fantastica', icon: '😍' },
        { value: 'good', label: 'Bella', icon: '🙂' },
        { value: 'ok', label: 'Così così', icon: '😐' },
        { value: 'bad', label: 'Da migliorare', icon: '😕' },
      ],
    },
    {
      id: 'team_spirit',
      title: 'Quanto ti sei sentito coinvolto?',
      answers: [
        { value: 'high', label: 'Moltissimo', icon: '🔥' },
        { value: 'mid', label: 'Abbastanza', icon: '👏' },
        { value: 'low', label: 'Poco', icon: '🤏' },
        { value: 'none', label: 'Per niente', icon: '🫥' },
      ],
    },
    {
      id: 'perks_interest',
      title: 'Ti interessano premi e vantaggi esclusivi?',
      answers: [
        { value: 'yes', label: 'Sì molto', icon: '🎁' },
        { value: 'maybe', label: 'Forse', icon: '🤔' },
        { value: 'little', label: 'Poco', icon: '😶' },
        { value: 'no', label: 'No', icon: '🙅' },
      ],
    },
    {
      id: 'mini_games_interest',
      title: 'Vorresti più mini giochi live?',
      answers: [
        { value: 'yes', label: 'Assolutamente sì', icon: '⚡' },
        { value: 'some', label: 'Qualcuno in più', icon: '🎮' },
        { value: 'same', label: 'Va bene così', icon: '👌' },
        { value: 'no', label: 'Non necessari', icon: '🧘' },
      ],
    },
  ],
  suggestionPrompt: 'Hai un suggerimento veloce?',
};

const normalizedSurvey = computed(() => {
  const rawQuestions = Array.isArray(props.feedbackSurvey?.questions) && props.feedbackSurvey.questions.length
    ? props.feedbackSurvey.questions
    : defaultSurvey.questions;

  const questions = rawQuestions
    .map((question) => {
      const id = String(question?.id || '').trim();
      const title = String(question?.title || '').trim();
      const answers = Array.isArray(question?.answers)
        ? question.answers
            .map((answer) => ({
              value: String(answer?.value || '').trim(),
              label: String(answer?.label || '').trim(),
              icon: String(answer?.icon || '').trim(),
            }))
            .filter((answer) => answer.value && answer.label)
        : [];

      if (!id || !title || !answers.length) {
        return null;
      }

      return { id, title, options: answers, answerKey: id };
    })
    .filter(Boolean);

  if (props.reportExperienceEnabled) {
    questions.push({
      id: 'report_experience',
      answerKey: 'report_experience',
      title: 'Com’è stata la tua esperienza con ArenaBoostX oggi?',
      options: [
        { value: 'awesome', label: 'Fantastica', icon: '😍' },
        { value: 'good', label: 'Bella', icon: '🙂' },
        { value: 'ok', label: 'Così così', icon: '😐' },
        { value: 'bad', label: 'Da migliorare', icon: '😕' },
      ],
    });
  }

  return {
    questions,
    suggestionPrompt: String(props.feedbackSurvey?.suggestionPrompt || props.feedbackSurvey?.suggestion_prompt || defaultSurvey.suggestionPrompt || '').trim(),
  };
});

const mandatoryKeys = ['experience', 'team_spirit', 'perks_interest', 'mini_games_interest'];
const answers = reactive({ experience: '', team_spirit: '', perks_interest: '', mini_games_interest: '', report_experience: '', suggestion: '' });
const step = ref(0);
const isSubmitting = ref(false);
const errorMessage = ref('');
const successMessage = ref('');
let flowStartedAt = Date.now();
let dismissed = false;
function telemetry(name, metadata = {}) {
  emit('telemetry', { name, metadata: { count: step.value + 1, ...metadata } });
}
function feedbackFailure(reason) { telemetry('feedback_failed', { reason, outcome: 'failure' }); }
function goBack() {
  if (isSubmitting.value || step.value <= 0) return;
  telemetry('feedback_step', { outcome: 'back' });
  step.value -= 1;
}

const hasOptionalStep = computed(() => Boolean(normalizedSurvey.value.suggestionPrompt));
const activeQuestion = computed(() => {
  if (step.value < normalizedSurvey.value.questions.length) {
    return normalizedSurvey.value.questions[step.value];
  }
  return hasOptionalStep.value ? { title: normalizedSurvey.value.suggestionPrompt } : null;
});
const isOptionalStep = computed(() => hasOptionalStep.value && step.value >= normalizedSurvey.value.questions.length);
const isFinalStep = computed(() => isOptionalStep.value || step.value >= normalizedSurvey.value.questions.length - 1);
const stepLabel = computed(() => {
  if (isOptionalStep.value) {
    return 'Extra (opzionale)';
  }
  return `Step ${Math.min(step.value + 1, normalizedSurvey.value.questions.length)} di ${normalizedSurvey.value.questions.length}`;
});

function close() {
  if (isSubmitting.value) return;
  if (!dismissed && !successMessage.value) {
    dismissed = true;
    telemetry('feedback_dismissed', { reason: 'user_closed', duration_ms: Math.min(86400000, Math.max(0, Date.now() - flowStartedAt)) });
  }
  emit('update:modelValue', false);
}

function resetFlow() {
  step.value = 0;
  errorMessage.value = '';
  successMessage.value = '';
  isSubmitting.value = false;
  answers.experience = '';
  answers.team_spirit = '';
  answers.perks_interest = '';
  answers.mini_games_interest = '';
  answers.report_experience = '';
  answers.suggestion = '';
}

function onSelectOption(value) {
  const question = activeQuestion.value;
  if (!question?.answerKey) return;
  answers[question.answerKey] = value;
  telemetry('feedback_step', { outcome: 'selected' });
  errorMessage.value = '';
  if (step.value < normalizedSurvey.value.questions.length - 1) {
    step.value += 1;
  }
}

function onContinue() {
  if (isSubmitting.value) return;

  if (!isOptionalStep.value) {
    const question = activeQuestion.value;
    if (!question?.answerKey || !answers[question.answerKey]) {
      errorMessage.value = 'Rispondi per continuare.';
      feedbackFailure('validation');
      return;
    }

    if (step.value < normalizedSurvey.value.questions.length - 1) {
      telemetry('feedback_step', { outcome: 'advanced' });
      step.value += 1;
      return;
    }

    if (hasOptionalStep.value && step.value >= normalizedSurvey.value.questions.length - 1) {
      telemetry('feedback_step', { outcome: 'advanced' });
      step.value = normalizedSurvey.value.questions.length;
      return;
    }
  }

  submit();
}

function skipOptional() {
  if (isSubmitting.value) return;
  telemetry('feedback_step', { outcome: 'skipped' });
  answers.suggestion = '';
  submit();
}

async function submit() {
  if (isSubmitting.value) return;
  if (!props.eventId) {
    errorMessage.value = 'Evento non disponibile.';
    feedbackFailure('unavailable');
    return;
  }
  const requiredKeys = props.reportExperienceEnabled ? [...mandatoryKeys, 'report_experience'] : mandatoryKeys;
  for (const key of requiredKeys) {
    if (!answers[key]) {
      errorMessage.value = 'Rispondi a tutte le domande prima di inviare.';
      feedbackFailure('validation');
      return;
    }
  }

  isSubmitting.value = true;
  errorMessage.value = '';
  const eventId = props.eventId;
  const requestStartedAt = Date.now();
  telemetry('feedback_step', { outcome: 'details_submitted' });
  const reportAnswers = { experience: answers.experience, perks_interest: answers.perks_interest };
  if (props.reportExperienceEnabled) reportAnswers.report_experience = answers.report_experience;

  try {
    const result = await submitEventFeedback(eventId, {
      experience: answers.experience,
      team_spirit: answers.team_spirit,
      perks_interest: answers.perks_interest,
      mini_games_interest: answers.mini_games_interest,
      suggestion: answers.suggestion.trim(),
    });
    if (eventId !== props.eventId) return;

    if (!result?.ok) {
      errorMessage.value = 'Non siamo riusciti a salvare il feedback. Riprova tra poco.';
      feedbackFailure('save_failed');
      return;
    }

    successMessage.value = 'Grazie! Feedback inviato con successo.';
    emit('submitted', reportAnswers);
    setTimeout(() => {
      if (eventId !== props.eventId) return;
      emit('update:modelValue', false);
      resetFlow();
    }, 550);
  } catch (error) {
    if (eventId !== props.eventId) return;
    telemetry('feedback_failed', { reason: 'network', outcome: 'failure', latency_ms: Math.min(86400000, Math.max(0, Date.now() - requestStartedAt)) });
    errorMessage.value = 'Non siamo riusciti a salvare il feedback. Riprova tra poco.';
  } finally {
    isSubmitting.value = false;
  }
}

watch(
  () => props.modelValue,
  (isOpen) => {
    if (isOpen) {
      flowStartedAt = Date.now();
      dismissed = false;
    } else {
      resetFlow();
    }
  },
);
</script>

<style scoped>
.feedback-modal {
  position: fixed;
  inset: 0;
  z-index: 130;
  display: flex;
  align-items: flex-end;
  justify-content: center;
  background: rgba(35, 22, 26, 0.42);
  backdrop-filter: blur(3px);
  color: var(--ink, #23161a);
  font-family: 'Inter', system-ui, sans-serif;
}
.feedback-modal * { box-sizing: border-box; }
.feedback-modal__panel {
  width: 100%;
  max-width: 430px;
  max-height: 82dvh;
  display: flex;
  flex-direction: column;
  padding: 10px 18px calc(18px + env(safe-area-inset-bottom));
  border-radius: 26px 26px 0 0;
  background: var(--bg, #fff7f2);
  box-shadow: 0 -8px 30px rgba(35, 22, 26, 0.18);
  overflow: hidden;
}
.feedback-modal__grabber {
  width: 38px;
  height: 5px;
  border-radius: 99px;
  background: var(--red-line, #ffd2ce);
  margin: 0 auto 12px;
  flex: none;
}
.feedback-modal__header,
.feedback-modal__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  flex: none;
}
.feedback-modal__header { margin-bottom: 12px; }
.feedback-modal__step {
  margin: 0;
  padding: 6px 10px;
  border-radius: 999px;
  background: var(--card, #ffffff);
  border: 2px solid var(--red-line, #ffd2ce);
  color: var(--red-text, #e4212e);
  font-size: 11.5px;
  font-weight: 800;
}
.feedback-modal__close {
  height: 32px;
  width: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  border-radius: 50%;
  border: 2px solid var(--line, #f1e4e0);
  background: var(--card, #ffffff);
  color: var(--muted, #8c7a80);
  font: inherit;
  font-weight: 700;
  cursor: pointer;
}
.feedback-modal__body { overflow-y: auto; min-height: 0; padding-bottom: 4px; }
.feedback-modal__title {
  margin: 0;
  font-family: 'Fraunces', Georgia, serif;
  font-size: 19px;
  font-weight: 750;
  line-height: 1.3;
  letter-spacing: -0.01em;
  color: var(--ink, #23161a);
  overflow-wrap: anywhere;
}
.feedback-modal__options { margin-top: 16px; display: grid; gap: 9px; padding-bottom: 4px; }
.feedback-modal__option {
  display: flex;
  align-items: center;
  gap: 11px;
  width: 100%;
  border-radius: 15px;
  border: 2px solid var(--line, #f1e4e0);
  background: var(--card, #ffffff);
  padding: 11px 13px;
  color: var(--ink, #23161a);
  font-family: inherit;
  font-size: 13.5px;
  font-weight: 800;
  text-align: left;
  cursor: pointer;
  box-shadow: 0 3px 0 rgba(35, 22, 26, 0.1);
  transition: border-color 0.15s ease, transform 0.1s ease;
}
.feedback-modal__option:hover { border-color: var(--red-line, #ffd2ce); }
.feedback-modal__option:active { transform: translateY(2px); box-shadow: none; }
.feedback-modal__option.active { border-color: var(--red, #e4212e); background: var(--red-soft, #ffe8e6); }
.feedback-modal__option-icon {
  width: 38px;
  min-height: 38px;
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  border: 2px solid var(--red-line, #ffd2ce);
  background: var(--red-soft, #ffe8e6);
  font-size: 20px;
  transform: rotate(-3deg);
}
.feedback-modal__optional { margin-top: 16px; }
.feedback-modal__input {
  width: 100%;
  min-height: 104px;
  border-radius: 13px;
  border: 2px solid var(--line, #f1e4e0);
  background: var(--card, #ffffff);
  color: var(--ink, #23161a);
  padding: 11px 13px;
  font-family: inherit;
  font-size: 13.5px;
  line-height: 1.5;
  resize: vertical;
}
.feedback-modal__input::placeholder { color: var(--muted, #8c7a80); }
.feedback-modal__input:focus { border-color: var(--red, #e4212e); }
.feedback-modal__counter { margin: 6px 0 0; text-align: right; font-size: 11px; color: var(--muted, #8c7a80); }
.feedback-modal__error,
.feedback-modal__success {
  margin: 12px 0 0;
  padding: 10px 12px;
  border-radius: 13px;
  border: 2px solid var(--red-line, #ffd2ce);
  background: var(--card, #ffffff);
  color: var(--red-text, #e4212e);
  font-size: 12.5px;
  font-weight: 700;
}
.feedback-modal__success { border-color: var(--gold, #f5b321); color: var(--gold-deep, #986700); }
.feedback-modal__footer { padding-top: 16px; flex-wrap: wrap; }
.feedback-modal__actions { margin-left: auto; display: flex; gap: 8px; }
.feedback-modal__back,
.feedback-modal__skip,
.feedback-modal__submit {
  border-radius: 13px;
  border: 2px solid var(--line, #f1e4e0);
  padding: 10px 14px;
  font-family: inherit;
  font-size: 13px;
  font-weight: 800;
  cursor: pointer;
  transition: transform 0.1s ease;
}
.feedback-modal__submit {
  background: var(--red, #e4212e);
  color: var(--on-primary, #ffffff);
  border-color: var(--red, #e4212e);
  box-shadow: 0 4px 0 var(--red-deep, #c0121f);
}
.feedback-modal__skip,
.feedback-modal__back { background: var(--card, #ffffff); color: var(--ink, #23161a); box-shadow: 0 3px 0 rgba(35, 22, 26, 0.1); }
.feedback-modal__footer button:active { transform: translateY(2px); box-shadow: none; }
.feedback-modal button:disabled,
.feedback-modal textarea:disabled { opacity: 0.5; cursor: not-allowed; box-shadow: none; }
.feedback-modal button:focus-visible,
.feedback-modal textarea:focus-visible { outline: 2px solid var(--red-text, #e4212e); outline-offset: 3px; }

.feedback-modal-fade-enter-active,
.feedback-modal-fade-leave-active { transition: opacity 0.22s ease; }
.feedback-modal-fade-enter-active .feedback-modal__panel,
.feedback-modal-fade-leave-active .feedback-modal__panel { transition: transform 0.28s cubic-bezier(0.32, 0.72, 0.28, 1); }
.feedback-modal-fade-enter-from,
.feedback-modal-fade-leave-to { opacity: 0; }
.feedback-modal-fade-enter-from .feedback-modal__panel,
.feedback-modal-fade-leave-to .feedback-modal__panel { transform: translateY(105%); }

@media (prefers-reduced-motion: reduce) {
  .feedback-modal-fade-enter-active,
  .feedback-modal-fade-leave-active,
  .feedback-modal-fade-enter-active .feedback-modal__panel,
  .feedback-modal-fade-leave-active .feedback-modal__panel,
  .feedback-modal__option,
  .feedback-modal__footer button { transition: none; }
}
</style>
