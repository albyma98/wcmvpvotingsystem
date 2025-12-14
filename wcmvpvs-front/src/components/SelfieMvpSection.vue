<template>
  <section
    class="selfie-section"
    :class="{ 'selfie-section--compact': props.compact }"
  >
    <input
      ref="fileInputRef"
      type="file"
      accept="image/*"
      capture="user"
      class="sr-only"
      @change="handleFileChange"
    />
    <div class="selfie-card">
      <div class="selfie-card__decor"></div>
      <div class="selfie-card__content">
        <header class="selfie-card__header">
          <p class="selfie-card__eyebrow">Selfie MVP</p>
          <h2 class="selfie-card__title">Scatta un selfie mentre tifi o con i tuoi amici!</h2>
          <p class="selfie-card__subtitle">
            Condividi il tuo entusiasmo: i selfie approvati potranno essere scelti e pubblicate sulle storie Instagram di JOY VOLLEY.
          </p>
        </header>

        <div v-if="showLoader" class="selfie-loader" role="status" aria-live="polite">
          <span class="selfie-spinner" aria-hidden="true"></span>
          <p>Verifica in corso…</p>
        </div>

        <div v-else class="selfie-body">
          <div
            class="selfie-preview"
            :class="{ 'selfie-preview--empty': !previewSource }"
            :style="previewStyle"
          >
            <img
              v-if="previewSource"
              :src="previewSource"
              alt="Anteprima selfie"
              @load="handleImageLoad"
              @error="handleImageError"
            />
            <div v-else class="selfie-preview__placeholder">
              <span class="selfie-preview__icon" aria-hidden="true">📸</span>
              <p>Seleziona uno scatto per vedere l'anteprima.</p>
            </div>
          </div>

          <div class="selfie-consent-block">
            <label class="selfie-consent">
              <input
                v-model="acceptedImageTerms"
                type="checkbox"
                required
                :disabled="interactionDisabled"
              />
              <span style="font-size:6px;">
Accetto la privacy policy.<br>
              </span>
            </label>
            <button
              type="button"
              class="selfie-link"
              :disabled="interactionDisabled"
              @click="showInformativaModal = true"
            >
             Privacy policy
            </button>
          </div>

          <div class="selfie-actions">
            <button
              type="button"
              class="selfie-button"
              :class="{ disabled: interactionDisabled }"
              :disabled="interactionDisabled"
              @click="triggerCapture"
            >
              Scatta ora
            </button>
            <button
              v-if="selectedFile"
              type="button"
              class="selfie-button outline"
              :disabled="isSubmitting"
              @click="clearSelection"
            >
              Annulla
            </button>
            <button
              type="button"
              class="selfie-button primary"
              :disabled="!canSubmit"
              @click="submitSelfie"
            >
              {{ isSubmitting ? 'Invio…' : 'Invia selfie' }}
            </button>
          </div>

          <p v-if="errorMessage" class="selfie-message error">{{ errorMessage }}</p>
          <p v-if="successMessage" class="selfie-message success">{{ successMessage }}</p>
          <p v-if="!selfie" class="selfie-hint">Il selfie verrà inviato allo staff per l'approvazione.</p>
        </div>
      </div>
    </div>
  </section>

  <div
    v-if="showInformativaModal"
    class="selfie-modal"
    role="dialog"
    aria-modal="true"
    aria-labelledby="selfie-privacy-title"
    @click.self="showInformativaModal = false"
  >
    <div class="selfie-modal__content">
      <header class="selfie-modal__header">
        <h3 id="selfie-privacy-title">Informativa Selfie MVP – Uso dell’immagine e Privacy</h3>
        <button
          type="button"
          class="selfie-modal__close"
          aria-label="Chiudi"
          @click="showInformativaModal = false"
        >
          ✕
        </button>
      </header>
<div class="selfie-modal__body">
  <section class="selfie-modal__section">
    <h4>Liberatoria sull’uso dell’immagine</h4>
    <p>
      Inviando questa foto confermo di avere almeno 18 anni e/o il
      consenso delle persone riconoscibili presenti nell’immagine,
      inclusi i genitori/tutori legali in caso di minorenni. Autorizzo
      gratuitamente JOY VOLLEY a utilizzare la fotografia inviata,
      senza limiti territoriali e temporali, per finalità informative e
      promozionali sui propri canali ufficiali (social, sito web,
      materiali di comunicazione). L’autorizzazione è concedibile senza
      alcun compenso e potrà essere revocata in qualsiasi momento nei
      limiti di legge.
    </p>
  </section>

  <section class="selfie-modal__section">
    <h4>Informativa Privacy</h4>
    <p>
      I dati personali e le immagini trasmesse tramite questa funzione
      sono trattati da JOY VOLLEY in qualità di Titolare del
      trattamento e da Alberto Marra in qualità di
      responsabile esterno. Il trattamento avviene per permettere la
      partecipazione all’attività “Selfie MVP” e la pubblicazione dei
      contenuti sui canali ufficiali della squadra, sulla base del
      consenso dell’utente. I dati saranno conservati per il tempo
      strettamente necessario alle attività indicate o fino alla
      revoca del consenso. Il conferimento è facoltativo, ma in sua
      assenza non sarà possibile utilizzare la foto inviata.
    </p>
  </section>

  <section class="selfie-modal__section">
    <h4>Diritti dell’interessato</h4>
    <p>
      Gli utenti possono esercitare i diritti previsti dagli artt.
      15-22 del GDPR, tra cui l’accesso, la rettifica, la cancellazione
      dell’immagine e la revoca del consenso in qualsiasi momento,
      scrivendo ai recapiti indicati di seguito.
    </p>
  </section>

  <section class="selfie-modal__section">
    <h3>Contatti Privacy</h3>
    <p>
      Per richieste o informazioni: <br>
      <strong>Responsabile trattamento tecnico:</strong> marralby@yahoo.it
    </p>
  </section>
</div>

<footer class="selfie-modal__footer">
  <button type="button" class="selfie-button primary"
          @click="showInformativaModal = false">
    Chiudi
  </button>
</footer>

    </div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, ref, watch } from 'vue';
import { fetchMySelfie, resolveApiUrl, trackPostVoteAction, uploadSelfie } from '../api';
import { safeTrackEvent } from '../tracking';

const MAX_FILE_SIZE = 8 * 1024 * 1024;
const DEFAULT_ASPECT_RATIO = '16 / 10';

const recordSelfieAction = (action) => {
  if (!props.eventId || !action) {
    return;
  }
  trackPostVoteAction(props.eventId, action);
};

const props = defineProps({
  eventId: {
    type: Number,
    required: true,
  },
  enabled: {
    type: Boolean,
    default: false,
  },
  loadingStatus: {
    type: Boolean,
    default: false,
  },
  compact: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits(['selfie-submitted']);

const selfie = ref(null);
const isLoadingSelfie = ref(false);
const selectedFile = ref(null);
const previewUrl = ref('');
const previewDimensions = ref({ width: 0, height: 0 });
const errorMessage = ref('');
const successMessage = ref('');
const isSubmitting = ref(false);
const fileInputRef = ref(null);
const acceptedImageTerms = ref(false);
const showInformativaModal = ref(false);

const showLoader = computed(() => props.loadingStatus || isLoadingSelfie.value);
const interactionDisabled = computed(() => !props.enabled || isSubmitting.value || props.loadingStatus);

const storedImageUrl = computed(() => {
  if (!selfie.value?.image_url) {
    return '';
  }
  try {
    return resolveApiUrl(selfie.value.image_url);
  } catch (error) {
    return selfie.value.image_url;
  }
});

const previewSource = computed(() => previewUrl.value || storedImageUrl.value);

const previewStyle = computed(() => {
  const { width, height } = previewDimensions.value || {};
  if (width > 0 && height > 0) {
    return { aspectRatio: `${width} / ${height}` };
  }
  return { aspectRatio: DEFAULT_ASPECT_RATIO };
});

const canSubmit = computed(
  () => Boolean(selectedFile.value) && acceptedImageTerms.value && !isSubmitting.value && props.enabled,
);

function revokePreview() {
  if (previewUrl.value) {
    URL.revokeObjectURL(previewUrl.value);
    previewUrl.value = '';
  }
}

function clearSelection() {
  revokePreview();
  selectedFile.value = null;
  if (fileInputRef.value) {
    fileInputRef.value.value = '';
  }
  errorMessage.value = '';
  resetPreviewDimensions();
}

function triggerCapture() {
  if (interactionDisabled.value) {
    return;
  }
  if (selfie.value) {
    recordSelfieAction("photo_edit_open");
  }
  safeTrackEvent('mission', 'start', 'self_mvp');
  if (fileInputRef.value) {
    fileInputRef.value.click();
  }
}

function handleFileChange(event) {
  const files = event?.target?.files;
  if (!files || !files.length) {
    return;
  }
  const [file] = files;
  if (!(file instanceof File)) {
    return;
  }
  if (!file.type?.startsWith('image/')) {
    clearSelection();
    errorMessage.value = 'Errore';
    return;
  }
  if (file.size > MAX_FILE_SIZE) {
    clearSelection();
    errorMessage.value = 'Errore';
    return;
  }
  revokePreview();
  selectedFile.value = file;
  previewUrl.value = URL.createObjectURL(file);
  successMessage.value = '';
  errorMessage.value = '';
}

function handleImageLoad(event) {
  const target = event?.target;
  const width = target?.naturalWidth || 0;
  const height = target?.naturalHeight || 0;
  if (width > 0 && height > 0) {
    previewDimensions.value = { width, height };
  }
}

function handleImageError() {
  resetPreviewDimensions();
}

function resetPreviewDimensions() {
  previewDimensions.value = { width: 0, height: 0 };
}

async function loadSelfie(eventId) {
  if (!eventId || !props.enabled) {
    selfie.value = null;
    return;
  }
  isLoadingSelfie.value = true;
  errorMessage.value = '';
  try {
    const { ok, selfie: data } = await fetchMySelfie(eventId);
    if (ok) {
      selfie.value = data || null;
      successMessage.value = '';
      clearSelection();
    }
  } catch (error) {
    console.error('Impossibile caricare il selfie', error);
    errorMessage.value = 'Errore';
    selfie.value = null;
  } finally {
    isLoadingSelfie.value = false;
  }
}

async function submitSelfie() {
  if (!props.eventId || !selectedFile.value || isSubmitting.value) {
    return;
  }
  if (!acceptedImageTerms.value) {
    errorMessage.value = 'Devi accettare il consenso per inviare il selfie.';
    return;
  }
  isSubmitting.value = true;
  errorMessage.value = '';
  successMessage.value = '';
  try {
    const { ok, selfie: data } = await uploadSelfie(props.eventId, {
      file: selectedFile.value,
      caption: '',
      acceptedImageTerms: acceptedImageTerms.value,
    });
    if (ok) {
      selfie.value = data || null;
      successMessage.value = 'Selfie inviato';
      safeTrackEvent('mission', 'complete', 'self_mvp');
      emit('selfie-submitted', data);
      clearSelection();
    } else {
      errorMessage.value = 'Errore';
    }
  } catch (error) {
    console.error('Impossibile inviare il selfie', error);
    errorMessage.value = 'Errore';
  } finally {
    isSubmitting.value = false;
  }
}

watch(
  () => [props.eventId, props.enabled],
  ([eventId, enabled]) => {
    if (!enabled) {
      selfie.value = null;
      successMessage.value = '';
      errorMessage.value = '';
      clearSelection();
      return;
    }
    if (eventId) {
      loadSelfie(eventId);
    }
  },
  { immediate: true },
);

watch(previewSource, (value) => {
  if (!value) {
    resetPreviewDimensions();
  }
});

onBeforeUnmount(() => {
  revokePreview();
});
</script>

<style scoped>
.selfie-section {
  position: relative;
}

.selfie-card {
  position: relative;
  overflow: hidden;
  border-radius: 2.25rem;
  border: 1px solid rgba(255, 255, 255, 0.12);
  background: linear-gradient(145deg, rgba(15, 23, 42, 0.92), rgba(30, 41, 59, 0.82));
  box-shadow: 0 30px 60px rgba(8, 15, 28, 0.55);
}

.selfie-card__decor {
  position: absolute;
  inset: 0;
  background: radial-gradient(circle at top right, rgba(148, 163, 184, 0.2), transparent 55%);
  opacity: 0.6;
  pointer-events: none;
}

.selfie-card__content {
  position: relative;
  padding: 1.75rem 1.5rem 1.75rem;
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  color: #e2e8f0;
}

.selfie-card__header {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  text-align: left;
}

.selfie-card__eyebrow {
  text-transform: uppercase;
  font-size: 0.7rem;
  letter-spacing: 0.4em;
  color: rgba(248, 250, 252, 0.85);
  margin: 0;
}

.selfie-card__title {
  margin: 0;
  font-size: 1.3rem;
  line-height: 1.4;
  color: #f8fafc;
}

.selfie-card__subtitle {
  margin: 0;
  font-size: 0.9rem;
  color: rgba(226, 232, 240, 0.85);
}

.selfie-loader {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 1.5rem 0.5rem;
  font-size: 0.95rem;
  color: #cbd5f5;
}

.selfie-spinner {
  width: 1.75rem;
  height: 1.75rem;
  border-radius: 999px;
  border: 2px solid rgba(148, 163, 184, 0.35);
  border-top-color: rgba(248, 250, 252, 0.9);
  animation: selfie-spin 0.8s linear infinite;
}

@keyframes selfie-spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.selfie-body {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.selfie-preview {
  position: relative;
  width: 75%;
  margin: 0 auto;
  border-radius: 1.75rem;
  overflow: hidden;
  border: 1px solid rgba(255, 255, 255, 0.12);
  background: rgba(15, 23, 42, 0.65);
}

.selfie-preview img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: contain;
  background: rgba(15, 23, 42, 0.85);
}

.selfie-preview--empty {
  border-style: dashed;
  border-color: rgba(148, 163, 184, 0.45);
}

.selfie-preview__placeholder {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  color: rgba(148, 163, 184, 0.8);
  font-size: 0.95rem;
  text-align: center;
  padding: 0 1rem;
}

.selfie-preview__icon {
  font-size: 1.75rem;
}

.selfie-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  justify-content: center;
}

.selfie-consent-block {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 0.75rem 1rem;
  border-radius: 1rem;
  background: rgba(15, 23, 42, 0.55);
  border: 1px solid rgba(255, 255, 255, 0.08);
}

.selfie-consent {
  display: flex;
  gap: 0.75rem;
  align-items: flex-start;
  color: #e2e8f0;
  font-size: 0.9rem;
  line-height: 1.5;
  margin: 0;
}

.selfie-consent input[type='checkbox'] {
  margin-top: 0.2rem;
  accent-color: #facc15;
}

.selfie-consent span {
  display: block;
}

.selfie-link {
  align-self: flex-start;
  background: none;
  border: none;
  color: #facc15;
  font-weight: 600;
  font-size: 0.9rem;
  text-decoration: underline;
  cursor: pointer;
  padding: 0;
}

.selfie-link:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.selfie-button {
  flex: 1 1 auto;
  min-width: 140px;
  border-radius: 999px;
  padding: 0.75rem 1.5rem;
  font-size: 0.85rem;
  font-weight: 600;
  letter-spacing: 0.2em;
  text-transform: uppercase;
  border: 1px solid rgba(255, 255, 255, 0.15);
  background: rgba(15, 23, 42, 0.65);
  color: #f8fafc;
  transition: background 0.2s ease, border-color 0.2s ease;
}

.selfie-button.primary {
  background: #facc15;
  color: #0f172a;
  border-color: #facc15;
}

.selfie-button.outline {
  background: transparent;
}

.selfie-button.disabled,
.selfie-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.selfie-button:not(.disabled):not(:disabled):hover {
  border-color: rgba(255, 255, 255, 0.4);
}

.selfie-button.primary:not(.disabled):not(:disabled):hover {
  background: #fde047;
  border-color: #fde047;
}

.selfie-message {
  margin: 0;
  font-size: 0.85rem;
}

.selfie-message.error {
  color: #fda4af;
}

.selfie-message.success {
  color: #bbf7d0;
}

.selfie-hint {
  margin: 0;
  font-size: 0.8rem;
  color: rgba(148, 163, 184, 0.8);
}

.selfie-section--compact .selfie-card {
  max-width: 420px;
  margin: 0 auto;
  border-radius: 1.75rem;
}

.selfie-section--compact .selfie-card__content {
  padding: 1.25rem 1.25rem 1.5rem;
  gap: 1rem;
}

.selfie-section--compact .selfie-card__header {
  text-align: center;
}

.selfie-section--compact .selfie-card__eyebrow {
  font-size: 0.65rem;
  letter-spacing: 0.35em;
}

.selfie-section--compact .selfie-card__title {
  font-size: 1.05rem;
}

.selfie-modal {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
  z-index: 30;
}

.selfie-modal__content {
  background: linear-gradient(160deg, rgba(15, 23, 42, 0.96), rgba(15, 23, 42, 0.9));
  color: #e2e8f0;
  border-radius: 1.5rem;
  border: 1px solid rgba(255, 255, 255, 0.12);
  box-shadow: 0 30px 80px rgba(0, 0, 0, 0.55);
  width: min(900px, 95vw);
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.selfie-modal__header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem;
  padding: 1.5rem 1.5rem 0.5rem;
}

.selfie-modal__header h3 {
  margin: 0;
  font-size: 1.15rem;
  letter-spacing: 0.02em;
}

.selfie-modal__close {
  border: 1px solid rgba(255, 255, 255, 0.12);
  background: rgba(255, 255, 255, 0.08);
  color: #f8fafc;
  border-radius: 999px;
  width: 2.5rem;
  height: 2.5rem;
  cursor: pointer;
}

.selfie-modal__body {
  padding: 0 1.5rem 1rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
  overflow-y: auto;
}

.selfie-modal__section {
  background: rgba(255, 255, 255, 0.02);
  border: 1px solid rgba(255, 255, 255, 0.05);
  border-radius: 1rem;
  padding: 1rem;
}

.selfie-modal__section h4 {
  margin: 0 0 0.35rem;
  font-size: 1rem;
}

.selfie-modal__section p {
  margin: 0;
  line-height: 1.6;
  color: rgba(226, 232, 240, 0.92);
  font-size: 0.95rem;
}

.selfie-modal__footer {
  padding: 0 1.5rem 1.5rem;
  display: flex;
  justify-content: flex-end;
}

.selfie-section--compact .selfie-card__subtitle {
  font-size: 0.8rem;
}

.selfie-section--compact .selfie-body {
  gap: 0.9rem;
}

.selfie-section--compact .selfie-preview {
  max-width: 165px;
  margin: 0 auto;
  border-radius: 1.25rem;
}

.selfie-section--compact .selfie-preview__placeholder {
  font-size: 0.85rem;
}

.selfie-section--compact .selfie-actions {
  gap: 0.5rem;
}

.selfie-section--compact .selfie-button {
  min-width: 120px;
  padding: 0.6rem 1rem;
  font-size: 0.75rem;
  letter-spacing: 0.18em;
}

.selfie-section--compact .selfie-message,
.selfie-section--compact .selfie-hint {
  font-size: 0.75rem;
}

@media (min-width: 768px) {
  .selfie-card__content {
    padding: 2.5rem 2.25rem 2.5rem;
  }
}
</style>
