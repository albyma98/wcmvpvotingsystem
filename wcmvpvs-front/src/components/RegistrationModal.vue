<template>
  <transition name="modal-fade">
    <div v-if="visible" class="registration-overlay" @click.self="emitClose">
      <div class="registration-modal">
        <header class="registration-modal__header">
          <p class="registration-modal__eyebrow">Ricompense sbloccate</p>
          <h3 class="registration-modal__title">🎉 Hai sbloccato ricompense speciali!</h3>
          <p class="registration-modal__subtitle">
            Hai completato {{ completedMissions }} missioni e ottenuto più chance nel sorteggio.
            Crea il tuo profilo in 15 secondi per non perdere progressi, badge e premi futuri 💙
          </p>
        </header>

        <form class="registration-form" @submit.prevent="handleSubmit">
          <label class="registration-field">
            <span>Nome</span>
            <input
              v-model.trim="name"
              type="text"
              name="name"
              autocomplete="name"
              placeholder="Il tuo nome"
              required
            />
          </label>

          <label class="registration-field">
            <span>Email o numero di telefono</span>
            <input
              v-model.trim="contactValue"
              :type="inputType"
              name="contact"
              autocomplete="email"
              placeholder="email@esempio.it oppure +39..."
              required
            />
            <p v-if="contactType" class="registration-field__hint">Rilevato: {{ contactTypeLabel }}</p>
          </label>

          <label class="registration-checkbox">
            <input v-model="marketingConsent" type="checkbox" required />
            <span>Accetto di ricevere notizie e premi della squadra</span>
          </label>

          <div v-if="errorMessage" class="registration-error" role="alert">{{ errorMessage }}</div>

          <div class="registration-actions">
            <button
              type="submit"
              class="registration-button primary"
              :disabled="submitting"
            >
              {{ submitting ? 'Invio…' : 'BLOCCA LE TUE RICOMPENSE 🔒' }}
            </button>
            <button
              type="button"
              class="registration-button ghost"
              :disabled="submitting"
              @click="emitLater"
            >
              Magari più tardi
            </button>
          </div>
        </form>
      </div>
    </div>
  </transition>
</template>

<script setup>
import { computed, ref, watch } from 'vue';

const props = defineProps({
  visible: { type: Boolean, default: false },
  completedMissions: { type: Number, default: 0 },
  submitting: { type: Boolean, default: false },
  initialContact: { type: String, default: '' },
  error: { type: String, default: '' },
});

const emit = defineEmits(['submit', 'close', 'later']);

const name = ref('');
const contactValue = ref(props.initialContact || '');
const marketingConsent = ref(true);
const errorMessage = ref('');

watch(
  () => props.initialContact,
  (value) => {
    if (value && !contactValue.value) {
      contactValue.value = value;
    }
  },
);

watch(
  () => props.visible,
  (visible) => {
    if (visible) {
      errorMessage.value = '';
    }
  },
);

watch(
  () => props.error,
  (value) => {
    errorMessage.value = value || '';
  },
);

const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/i;
const phonePattern = /^\+?[0-9]{6,15}$/;

const contactType = computed(() => {
  const trimmed = (contactValue.value || '').trim();
  if (emailPattern.test(trimmed)) {
    return 'email';
  }
  const phoneCandidate = trimmed.replace(/[^0-9+]/g, '');
  if (phonePattern.test(phoneCandidate)) {
    return 'phone';
  }
  return '';
});

const contactTypeLabel = computed(() => (contactType.value === 'phone' ? 'Telefono' : 'Email'));

const inputType = computed(() => (contactType.value === 'phone' ? 'tel' : 'email'));

const emitClose = () => emit('close');
const emitLater = () => emit('later');

const sanitizeContact = (value) => {
  const trimmed = (value || '').trim();
  if (contactType.value === 'phone') {
    const cleaned = trimmed.replace(/[^0-9+]/g, '');
    if (cleaned.startsWith('++')) {
      return cleaned.replace(/^\++/, '+');
    }
    if (cleaned.split('+').length > 2) {
      return cleaned.replace(/\+/g, '');
    }
    return cleaned;
  }
  if (contactType.value === 'email') {
    return trimmed.toLowerCase();
  }
  return trimmed;
};

const handleSubmit = () => {
  errorMessage.value = '';
  if (!name.value.trim()) {
    errorMessage.value = 'Inserisci il tuo nome.';
    return;
  }
  if (!contactType.value) {
    errorMessage.value = 'Inserisci una email o un numero di telefono valido.';
    return;
  }
  if (!marketingConsent.value) {
    errorMessage.value = 'Accetta di ricevere notizie per salvare le ricompense.';
    return;
  }

  emit('submit', {
    name: name.value.trim(),
    contactValue: sanitizeContact(contactValue.value),
    contactType: contactType.value,
    marketingConsent: true,
  });
};
</script>

<style scoped>
.registration-overlay {
  position: fixed;
  inset: 0;
  z-index: 90;
  background: rgba(2, 6, 23, 0.78);
  backdrop-filter: blur(10px);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1.25rem;
}

.registration-modal {
  width: min(520px, 100%);
  border-radius: 1.5rem;
  border: 1px solid rgba(148, 163, 184, 0.3);
  background: linear-gradient(145deg, rgba(15, 23, 42, 0.95), rgba(30, 41, 59, 0.9));
  padding: 1.5rem;
  color: #e2e8f0;
  box-shadow: 0 28px 64px rgba(8, 15, 28, 0.65);
}

.registration-modal__header {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  margin-bottom: 1.25rem;
}

.registration-modal__eyebrow {
  text-transform: uppercase;
  letter-spacing: 0.2em;
  font-size: 0.8rem;
  color: #a5b4fc;
  font-weight: 700;
}

.registration-modal__title {
  font-size: 1.35rem;
  font-weight: 800;
  color: #f8fafc;
}

.registration-modal__subtitle {
  font-size: 0.98rem;
  color: #cbd5e1;
  line-height: 1.5;
}

.registration-form {
  display: flex;
  flex-direction: column;
  gap: 0.9rem;
}

.registration-field {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  font-weight: 600;
  color: #cbd5e1;
}

.registration-field input {
  width: 100%;
  border-radius: 0.9rem;
  border: 1px solid rgba(148, 163, 184, 0.35);
  background: rgba(15, 23, 42, 0.7);
  color: #e2e8f0;
  padding: 0.85rem 1rem;
  font-size: 1rem;
  transition: border-color 160ms ease, box-shadow 160ms ease;
}

.registration-field input:focus {
  outline: none;
  border-color: rgba(34, 211, 238, 0.7);
  box-shadow: 0 0 0 1px rgba(34, 211, 238, 0.35);
}

.registration-field__hint {
  font-size: 0.85rem;
  color: #38bdf8;
}

.registration-checkbox {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.95rem;
  color: #e2e8f0;
}

.registration-checkbox input {
  width: 18px;
  height: 18px;
  accent-color: #22d3ee;
}

.registration-error {
  background: rgba(248, 113, 113, 0.12);
  color: #fecdd3;
  border: 1px solid rgba(248, 113, 113, 0.35);
  border-radius: 0.9rem;
  padding: 0.7rem 0.85rem;
  font-weight: 600;
}

.registration-actions {
  display: flex;
  flex-direction: column;
  gap: 0.65rem;
  margin-top: 0.35rem;
}

.registration-button {
  width: 100%;
  border-radius: 999px;
  padding: 0.95rem 1.25rem;
  font-weight: 800;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  transition: transform 160ms ease, box-shadow 200ms ease, background 200ms ease, color 200ms ease;
}

.registration-button.primary {
  background: linear-gradient(135deg, #22d3ee, #8b5cf6);
  color: #0f172a;
  border: none;
  box-shadow: 0 18px 44px rgba(59, 130, 246, 0.35);
}

.registration-button.ghost {
  background: transparent;
  color: #e2e8f0;
  border: 1px solid rgba(148, 163, 184, 0.35);
}

.registration-button:hover:not(:disabled) {
  transform: translateY(-1px);
}

.registration-button:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}

.modal-fade-enter-active,
.modal-fade-leave-active {
  transition: opacity 150ms ease;
}

.modal-fade-enter-from,
.modal-fade-leave-to {
  opacity: 0;
}
</style>
