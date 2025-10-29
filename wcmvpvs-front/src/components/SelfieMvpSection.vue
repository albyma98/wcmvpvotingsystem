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
            Condividi il tuo entusiasmo: i selfie approvati potranno essere mostrati sul maxischermo durante la partita.
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

          <div v-if="selfie" class="selfie-status">
            <p class="selfie-status__label">
              Stato: <strong>{{ statusLabel }}</strong>
            </p>
            <p v-if="submittedAtLabel" class="selfie-status__time">Inviato il {{ submittedAtLabel }}</p>
          </div>

          <label class="selfie-caption">
            <span>Didascalia (max {{ CAPTION_LIMIT }} caratteri)</span>
            <textarea
              v-model="captionInput"
              :maxlength="CAPTION_LIMIT"
              :disabled="interactionDisabled"
              placeholder="Es. Forza squadra!"
              rows="2"
            ></textarea>
            <span class="selfie-caption__counter">{{ captionRemaining }} caratteri rimanenti</span>
          </label>

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
</template>

<script setup>
import { computed, onBeforeUnmount, ref, watch } from 'vue';
import { fetchMySelfie, resolveApiUrl, uploadSelfie } from '../api';

const CAPTION_LIMIT = 80;
const MAX_FILE_SIZE = 8 * 1024 * 1024;
const DEFAULT_ASPECT_RATIO = '16 / 10';

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
const captionInput = ref('');
const errorMessage = ref('');
const successMessage = ref('');
const isSubmitting = ref(false);
const fileInputRef = ref(null);

const showLoader = computed(() => props.loadingStatus || isLoadingSelfie.value);
const interactionDisabled = computed(() => !props.enabled || isSubmitting.value || props.loadingStatus);

const captionRemaining = computed(() => {
  const length = Array.from(captionInput.value || '').length;
  return Math.max(0, CAPTION_LIMIT - length);
});

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

const statusLabel = computed(() => {
  if (!selfie.value) {
    return 'In attesa di invio';
  }
  if (selfie.value.approved) {
    return selfie.value.show_on_screen ? 'Approvato per il maxischermo' : 'Approvato';
  }
  return 'In attesa di approvazione';
});

const submittedAtLabel = computed(() => {
  if (!selfie.value?.submitted_at && !selfie.value?.created_at) {
    return '';
  }
  const raw = selfie.value.submitted_at || selfie.value.created_at;
  try {
    return new Intl.DateTimeFormat('it-IT', {
      dateStyle: 'medium',
      timeStyle: 'short',
    }).format(new Date(raw));
  } catch (error) {
    return raw;
  }
});

const canSubmit = computed(() => Boolean(selectedFile.value) && !isSubmitting.value && props.enabled);

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
    errorMessage.value = 'Seleziona un file immagine valido.';
    clearSelection();
    return;
  }
  if (file.size > MAX_FILE_SIZE) {
    errorMessage.value = 'L\'immagine è troppo pesante. Dimensione massima 8 MB.';
    clearSelection();
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
    captionInput.value = '';
    return;
  }
  isLoadingSelfie.value = true;
  errorMessage.value = '';
  try {
    const { ok, selfie: data } = await fetchMySelfie(eventId);
    if (ok) {
      selfie.value = data || null;
      captionInput.value = data?.caption || '';
      successMessage.value = '';
      clearSelection();
    }
  } catch (error) {
    console.error('Impossibile caricare il selfie', error);
    errorMessage.value = 'Impossibile caricare il tuo selfie in questo momento.';
    selfie.value = null;
  } finally {
    isLoadingSelfie.value = false;
  }
}

async function submitSelfie() {
  if (!props.eventId || !selectedFile.value || isSubmitting.value) {
    return;
  }
  isSubmitting.value = true;
  errorMessage.value = '';
  successMessage.value = '';
  try {
    const { ok, selfie: data, error } = await uploadSelfie(props.eventId, {
      file: selectedFile.value,
      caption: captionInput.value.trim(),
    });
    if (ok) {
      selfie.value = data || null;
      captionInput.value = data?.caption || '';
      successMessage.value = 'Selfie inviato! Attendi la conferma dello staff.';
      emit('selfie-submitted', data);
      clearSelection();
    } else {
      const message =
        error?.response?.data?.message || error?.message || 'Impossibile inviare il selfie. Riprova.';
      errorMessage.value = message;
    }
  } catch (error) {
    const message =
      error?.response?.data?.message || error?.message || 'Impossibile inviare il selfie. Riprova.';
    errorMessage.value = message;
  } finally {
    isSubmitting.value = false;
  }
}

watch(
  () => [props.eventId, props.enabled],
  ([eventId, enabled]) => {
    if (!enabled) {
      selfie.value = null;
      captionInput.value = '';
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

watch(captionInput, (value) => {
  const limited = Array.from(value || '').slice(0, CAPTION_LIMIT).join('');
  if (limited !== value) {
    captionInput.value = limited;
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
  width: 100%;
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

.selfie-status {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  font-size: 0.9rem;
  color: rgba(226, 232, 240, 0.9);
}

.selfie-status__label {
  margin: 0;
}

.selfie-status__time {
  margin: 0;
  color: rgba(148, 163, 184, 0.85);
}

.selfie-caption {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  font-size: 0.9rem;
  color: rgba(226, 232, 240, 0.92);
}

.selfie-caption textarea {
  width: 100%;
  border-radius: 1rem;
  border: 1px solid rgba(148, 163, 184, 0.35);
  background: rgba(15, 23, 42, 0.6);
  color: #f8fafc;
  padding: 0.75rem 1rem;
  font-size: 0.95rem;
  resize: none;
}

.selfie-caption textarea:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.selfie-caption__counter {
  font-size: 0.75rem;
  color: rgba(148, 163, 184, 0.85);
  align-self: flex-end;
}

.selfie-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
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

.selfie-section--compact .selfie-card__subtitle {
  font-size: 0.8rem;
}

.selfie-section--compact .selfie-body {
  gap: 0.9rem;
}

.selfie-section--compact .selfie-preview {
  max-width: 220px;
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

.selfie-section--compact .selfie-caption textarea {
  font-size: 0.85rem;
  padding: 0.65rem 0.9rem;
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
