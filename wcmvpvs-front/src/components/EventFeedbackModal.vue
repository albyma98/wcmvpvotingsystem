<template>
  <Teleport to="body">
    <Transition name="feedback-modal-fade">
      <div
        v-if="modelValue"
        class="feedback-modal"
        role="dialog"
        aria-modal="true"
        aria-labelledby="feedback-modal-title"
        @click.self="close"
      >
        <div class="feedback-modal__panel">
          <header class="feedback-modal__header">
            <p class="feedback-modal__step">{{ stepLabel }}</p>
            <button type="button" class="feedback-modal__close" aria-label="Chiudi feedback" @click="close">×</button>
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
                @click="onSelectOption(option.value)"
              >
                <span class="feedback-modal__option-icon">{{ option.icon || '•' }}</span>
                <span>{{ option.label }}</span>
              </button>
            </div>

            <p v-if="errorMessage" class="feedback-modal__error">{{ errorMessage }}</p>
            <p v-if="successMessage" class="feedback-modal__success">{{ successMessage }}</p>
          </div>

          <footer class="feedback-modal__footer">
            <button v-if="step > 0" type="button" class="feedback-modal__back" :disabled="isSubmitting" @click="step -= 1">Indietro</button>
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
});

const emit = defineEmits(['update:modelValue', 'submitted']);

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

  return {
    questions,
    suggestionPrompt: String(props.feedbackSurvey?.suggestionPrompt || props.feedbackSurvey?.suggestion_prompt || defaultSurvey.suggestionPrompt || '').trim(),
  };
});

const mandatoryKeys = ['experience', 'team_spirit', 'perks_interest', 'mini_games_interest'];
const answers = reactive({ experience: '', team_spirit: '', perks_interest: '', mini_games_interest: '', suggestion: '' });
const step = ref(0);
const isSubmitting = ref(false);
const errorMessage = ref('');
const successMessage = ref('');

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
  answers.suggestion = '';
}

function onSelectOption(value) {
  const question = activeQuestion.value;
  if (!question?.answerKey) return;
  answers[question.answerKey] = value;
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
      return;
    }

    if (hasOptionalStep.value && step.value >= normalizedSurvey.value.questions.length - 1) {
      step.value = normalizedSurvey.value.questions.length;
      return;
    }
  }

  submit();
}

function skipOptional() {
  answers.suggestion = '';
  submit();
}

async function submit() {
  if (!props.eventId) {
    errorMessage.value = 'Evento non disponibile.';
    return;
  }
  for (const key of mandatoryKeys) {
    if (!answers[key]) {
      errorMessage.value = 'Rispondi a tutte le domande prima di inviare.';
      return;
    }
  }

  isSubmitting.value = true;
  errorMessage.value = '';

  try {
    const result = await submitEventFeedback(props.eventId, {
      experience: answers.experience,
      team_spirit: answers.team_spirit,
      perks_interest: answers.perks_interest,
      mini_games_interest: answers.mini_games_interest,
      suggestion: answers.suggestion.trim(),
    });

    if (!result?.ok) {
      errorMessage.value = 'Non siamo riusciti a salvare il feedback. Riprova tra poco.';
      return;
    }

    successMessage.value = 'Grazie! Feedback inviato con successo 💙';
    emit('submitted');
    setTimeout(() => {
      emit('update:modelValue', false);
      resetFlow();
    }, 550);
  } catch (error) {
    errorMessage.value = 'Non siamo riusciti a salvare il feedback. Riprova tra poco.';
  } finally {
    isSubmitting.value = false;
  }
}

watch(
  () => props.modelValue,
  (isOpen) => {
    if (!isOpen) {
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
  display: grid;
  place-items: center;
  background: rgba(2, 6, 23, 0.75);
  backdrop-filter: blur(5px);
  padding: clamp(0.8rem, 2.4vh, 1.1rem);
}
.feedback-modal__panel {
  width: min(100%, 34rem);
  border-radius: 1rem;
  border: 1px solid rgba(255, 255, 255, 0.2);
  background: linear-gradient(170deg, rgba(15, 23, 42, 0.96), rgba(30, 41, 59, 0.96));
  box-shadow: 0 24px 50px rgba(15, 23, 42, 0.5);
}
.feedback-modal__header,
.feedback-modal__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.9rem;
}
.feedback-modal__step { font-size: 0.76rem; font-weight: 800; color: #fde68a; text-transform: uppercase; letter-spacing: 0.08em; }
.feedback-modal__close {
  height: 2rem;
  width: 2rem;
  border-radius: 9999px;
  border: 1px solid rgba(255,255,255,0.25);
  background: rgba(255,255,255,0.08);
  color: #fff;
}
.feedback-modal__body { padding: 0 0.9rem 0.9rem; }
.feedback-modal__title { font-size: clamp(1rem, 4.2vw, 1.28rem); font-weight: 900; color: #fff; }
.feedback-modal__options { margin-top: 0.9rem; display: grid; gap: 0.55rem; }
.feedback-modal__option {
  display: flex; align-items: center; gap: 0.55rem;
  border-radius: 0.8rem;
  border: 1px solid rgba(255,255,255,0.2);
  background: rgba(15, 23, 42, 0.65);
  padding: 0.7rem;
  color: #e2e8f0;
  font-weight: 700;
}
.feedback-modal__option.active { border-color: rgba(251,191,36,0.7); background: rgba(251,191,36,0.15); color: #fef3c7; }
.feedback-modal__optional { margin-top: 0.8rem; }
.feedback-modal__input {
  width: 100%;
  border-radius: 0.8rem;
  border: 1px solid rgba(255,255,255,0.2);
  background: rgba(15, 23, 42, 0.78);
  color: #fff;
  padding: 0.72rem;
}
.feedback-modal__counter { margin-top: 0.4rem; text-align: right; font-size: 0.74rem; color: rgba(226,232,240,0.8); }
.feedback-modal__error { margin-top: 0.7rem; color: #fca5a5; font-weight: 700; }
.feedback-modal__success { margin-top: 0.7rem; color: #86efac; font-weight: 700; }
.feedback-modal__actions { margin-left: auto; display: flex; gap: 0.5rem; }
.feedback-modal__back,
.feedback-modal__skip,
.feedback-modal__submit {
  border-radius: 0.6rem;
  border: 1px solid rgba(255,255,255,0.22);
  padding: 0.55rem 0.8rem;
  font-weight: 800;
}
.feedback-modal__submit {
  background: linear-gradient(135deg, #f59e0b, #ef4444);
  color: #fff;
  border-color: rgba(251,191,36,0.65);
}
.feedback-modal__skip,
.feedback-modal__back { background: rgba(148,163,184,0.12); color: #e2e8f0; }

.feedback-modal-fade-enter-active,
.feedback-modal-fade-leave-active { transition: opacity 0.2s ease; }
.feedback-modal-fade-enter-from,
.feedback-modal-fade-leave-to { opacity: 0; }
</style>
