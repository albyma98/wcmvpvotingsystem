<template>
  <div class="scanner-shell">
    <header class="scanner-header">
      <div>
        <h1>Verifica ticket lotteria</h1>
        <p class="subtitle">Scansiona i QR code dei ticket per validarli</p>
      </div>
      <div class="header-actions" v-if="isAuthenticated">
        <span class="muted">Connesso come <strong>{{ activeUsername }}</strong></span>
        <button class="btn ghost" type="button" @click="goToLottery">Lotteria</button>
        <button class="btn secondary" type="button" @click="logout">Esci</button>
      </div>
    </header>

    <section v-if="!isAuthenticated" class="card login-card">
      <h2>Accedi</h2>
      <form @submit.prevent="login" class="form-grid">
        <label>
          Username
          <input v-model.trim="loginForm.username" type="text" autocomplete="username" required />
        </label>
        <label>
          Password
          <input v-model="loginForm.password" type="password" autocomplete="current-password" required />
        </label>
        <button class="btn primary" type="submit" :disabled="isLoggingIn">
          {{ isLoggingIn ? 'Accesso in corso…' : 'Entra' }}
        </button>
      </form>
      <p v-if="loginError" class="error">{{ loginError }}</p>
    </section>

    <section v-else class="card scanner-card">
      <div class="scanner-grid">
        <div class="video-wrapper">
          <video ref="video" autoplay playsinline muted></video>
          <p v-if="cameraError" class="error">{{ cameraError }}</p>
        </div>
        <div class="status-panel">
          <h2>Esito scansione</h2>
          <p v-if="statusMessage" :class="['status-message', statusClass]">{{ statusMessage }}</p>
          <p v-else class="muted">Inquadra un QR code per iniziare.</p>
          <p v-if="lastValidatedCode" class="last-code">Ultimo ticket: <strong>{{ lastValidatedCode }}</strong></p>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';
import { apiClient } from '../api';

const basePath = import.meta.env.BASE_URL ?? '/';

const video = ref(null);
const detectorSupported = typeof window !== 'undefined' && 'BarcodeDetector' in window;
const cameraError = ref(detectorSupported ? '' : 'BarcodeDetector API non supportata dal browser.');
const statusMessage = ref('');
const statusType = ref('info');
const lastValidatedCode = ref('');
const lastDetectedPayload = ref('');
const lastDetectionAt = ref(0);
const isValidating = ref(false);
let stream;
let detector;
let animationFrameId;

const token = ref(localStorage.getItem('adminToken') || '');
const activeUsername = ref(localStorage.getItem('adminUsername') || '');
const isAuthenticated = computed(() => Boolean(token.value));

const loginForm = reactive({
  username: '',
  password: '',
});

const isLoggingIn = ref(false);
const loginError = ref('');

const statusClass = computed(() => {
  switch (statusType.value) {
    case 'success':
      return 'status-success';
    case 'warning':
      return 'status-warning';
    case 'error':
      return 'status-error';
    default:
      return 'status-info';
  }
});

const authHeaders = computed(() => ({
  headers: {
    Authorization: token.value ? `Bearer ${token.value}` : '',
  },
}));

function goToLottery() {
  window.location.assign(`${basePath}admin/lottery`);
}

async function login() {
  if (isLoggingIn.value) {
    return;
  }
  loginError.value = '';
  statusMessage.value = '';
  isLoggingIn.value = true;
  try {
    const { data } = await apiClient.post('/admin/login', {
      username: loginForm.username,
      password: loginForm.password,
    });
    token.value = data?.token || '';
    activeUsername.value = data?.username || loginForm.username;
    localStorage.setItem('adminToken', token.value);
    localStorage.setItem('adminUsername', activeUsername.value);
    loginForm.username = '';
    loginForm.password = '';
  } catch (error) {
    console.error('login error', error);
    loginError.value = 'Credenziali non valide.';
  } finally {
    isLoggingIn.value = false;
  }
}

function logout() {
  localStorage.removeItem('adminToken');
  localStorage.removeItem('adminUsername');
  token.value = '';
  activeUsername.value = '';
  stopScanner();
}

function resetStatus() {
  if (statusType.value !== 'info') {
    statusType.value = 'info';
    statusMessage.value = '';
  }
}

function clearLastPayload() {
  lastDetectedPayload.value = '';
  lastDetectionAt.value = 0;
}

async function ensureScanner() {
  if (!detectorSupported) {
    return;
  }
  if (!isAuthenticated.value) {
    stopScanner();
    return;
  }
  if (typeof navigator === 'undefined' || !navigator.mediaDevices?.getUserMedia) {
    cameraError.value = 'Questo dispositivo non supporta la scansione.';
    return;
  }
  if (stream) {
    return;
  }
  try {
    detector = detector || new window.BarcodeDetector({ formats: ['qr_code'] });
    stream = await navigator.mediaDevices.getUserMedia({ video: { facingMode: 'environment' } });
    if (video.value) {
      video.value.srcObject = stream;
    }
    cameraError.value = '';
    scheduleScan();
  } catch (error) {
    console.error('camera access error', error);
    cameraError.value = 'Impossibile accedere alla fotocamera.';
  }
}

function stopScanner() {
  if (animationFrameId) {
    cancelAnimationFrame(animationFrameId);
    animationFrameId = undefined;
  }
  if (stream) {
    stream.getTracks().forEach((track) => track.stop());
    stream = undefined;
  }
  detector = undefined;
}

function scheduleScan() {
  if (!detectorSupported || !video.value) {
    return;
  }
  animationFrameId = requestAnimationFrame(scanFrame);
}

async function scanFrame() {
  if (!detector || !video.value || !isAuthenticated.value) {
    animationFrameId = requestAnimationFrame(scanFrame);
    return;
  }
  if (video.value.readyState >= HTMLMediaElement.HAVE_ENOUGH_DATA) {
    try {
      const barcodes = await detector.detect(video.value);
      if (barcodes.length > 0) {
        const rawValue = barcodes[0].rawValue?.trim();
        if (rawValue) {
          handlePayload(rawValue);
        }
      }
    } catch (error) {
      console.error('barcode detection error', error);
    }
  }
  animationFrameId = requestAnimationFrame(scanFrame);
}

function handlePayload(rawValue) {
  const now = Date.now();
  if (rawValue === lastDetectedPayload.value && now - lastDetectionAt.value < 2500) {
    return;
  }
  lastDetectedPayload.value = rawValue;
  lastDetectionAt.value = now;

  let parsed;
  try {
    parsed = JSON.parse(rawValue);
  } catch (error) {
    statusType.value = 'error';
    statusMessage.value = 'QR code non riconosciuto.';
    console.error('invalid qr payload', error, rawValue);
    return;
  }
  const code = String(parsed?.code || '').trim();
  const signature = String(parsed?.signature || '').trim();
  if (!code || !signature) {
    statusType.value = 'error';
    statusMessage.value = 'QR code incompleto.';
    return;
  }
  validateTicket(code, signature);
}

async function validateTicket(code, signature) {
  if (isValidating.value) {
    return;
  }
  isValidating.value = true;
  statusType.value = 'info';
  statusMessage.value = 'Verifica in corso…';
  try {
    const { data } = await apiClient.post(
      '/lottery/scan',
      { code, signature },
      authHeaders.value,
    );
    const status = data?.status || 'invalid';
    lastValidatedCode.value = code;
    switch (status) {
      case 'valid':
        statusType.value = 'success';
        statusMessage.value = 'Ticket valido';
        break;
      case 'already_scanned':
        statusType.value = 'warning';
        statusMessage.value = data?.message || 'Già scannerizzato';
        break;
      default:
        statusType.value = 'error';
        statusMessage.value = data?.message || 'Ticket non valido';
        break;
    }
  } catch (error) {
    console.error('ticket validation error', error);
    statusType.value = 'error';
    statusMessage.value = 'Errore durante la verifica del ticket.';
  } finally {
    isValidating.value = false;
    setTimeout(() => {
      if (statusType.value === 'success') {
        resetStatus();
      }
      clearLastPayload();
    }, 2000);
  }
}

onMounted(() => {
  if (isAuthenticated.value) {
    ensureScanner();
  }
});

onBeforeUnmount(() => {
  stopScanner();
});

watch(isAuthenticated, (auth) => {
  if (auth) {
    ensureScanner();
  } else {
    resetStatus();
    clearLastPayload();
  }
});
</script>

<style scoped>
.scanner-shell {
  min-height: 100vh;
  padding: 2rem clamp(1rem, 4vw, 3rem);
  background: linear-gradient(180deg, #0f172a 0%, #1e293b 45%, #0f172a 100%);
  color: #e2e8f0;
  display: flex;
  flex-direction: column;
  gap: 2rem;
}

.scanner-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1.5rem;
}

.scanner-header h1 {
  font-size: clamp(1.75rem, 2.5vw, 2.5rem);
  margin: 0;
}

.subtitle {
  margin: 0.25rem 0 0;
  color: #94a3b8;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.muted {
  color: #cbd5f5;
  font-size: 0.9rem;
}

.card {
  background: rgba(15, 23, 42, 0.65);
  border: 1px solid rgba(148, 163, 184, 0.15);
  border-radius: 1.5rem;
  padding: clamp(1.5rem, 2.5vw, 2.5rem);
  box-shadow: 0 24px 48px rgba(8, 15, 28, 0.45);
}

.login-card {
  max-width: 420px;
}

.form-grid {
  display: grid;
  gap: 1rem;
}

.form-grid label {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  font-weight: 600;
}

.form-grid input {
  padding: 0.75rem 1rem;
  border-radius: 0.75rem;
  border: 1px solid rgba(148, 163, 184, 0.25);
  background: rgba(15, 23, 42, 0.5);
  color: inherit;
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.35rem;
  border-radius: 999px;
  padding: 0.65rem 1.5rem;
  font-weight: 600;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
  cursor: pointer;
  border: none;
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn.primary {
  background: linear-gradient(135deg, #06b6d4, #3b82f6);
  color: #0f172a;
  box-shadow: 0 12px 24px rgba(14, 165, 233, 0.35);
}

.btn.secondary {
  background: rgba(51, 65, 85, 0.75);
  color: #e2e8f0;
}

.btn.ghost {
  background: transparent;
  border: 1px solid rgba(148, 163, 184, 0.35);
  color: inherit;
}

.error {
  margin-top: 0.75rem;
  color: #f87171;
}

.scanner-card {
  width: 100%;
}

.scanner-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 2rem;
  align-items: flex-start;
}

.video-wrapper {
  position: relative;
  border-radius: 1.25rem;
  overflow: hidden;
  border: 1px solid rgba(148, 163, 184, 0.2);
  background: rgba(15, 23, 42, 0.5);
}

video {
  display: block;
  width: 100%;
  height: 100%;
  max-height: 360px;
  object-fit: cover;
}

.status-panel {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  padding: 0.5rem 0;
}

.status-panel h2 {
  margin: 0;
  font-size: 1.25rem;
}

.status-message {
  font-size: 1.1rem;
  font-weight: 600;
  padding: 0.75rem 1rem;
  border-radius: 1rem;
  border: 1px solid transparent;
}

.status-success {
  background: rgba(34, 197, 94, 0.15);
  border-color: rgba(34, 197, 94, 0.35);
  color: #86efac;
}

.status-warning {
  background: rgba(234, 179, 8, 0.15);
  border-color: rgba(234, 179, 8, 0.35);
  color: #facc15;
}

.status-error {
  background: rgba(248, 113, 113, 0.12);
  border-color: rgba(248, 113, 113, 0.35);
  color: #fca5a5;
}

.status-info {
  background: rgba(59, 130, 246, 0.12);
  border-color: rgba(59, 130, 246, 0.35);
  color: #bfdbfe;
}

.last-code {
  font-size: 0.95rem;
  color: #cbd5f5;
}

@media (max-width: 640px) {
  .scanner-header {
    flex-direction: column;
    align-items: flex-start;
  }
  .header-actions {
    width: 100%;
    justify-content: space-between;
  }
}
</style>
