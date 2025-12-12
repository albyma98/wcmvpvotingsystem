<template>
  <div class="partner-shell">
    <header class="partner-header">
      <div>
        <p class="eyebrow">Validazione coupon</p>
        <h1>Area esercenti</h1>
        <p class="subtitle">
          Accedi per validare i coupon provenienti dai QR generati nell'app pubblica.
        </p>
      </div>
      <div class="session-actions" v-if="isAuthenticated">
        <p class="session-user">{{ activeUsername }}</p>
        <button type="button" class="btn" @click="logout">Esci</button>
      </div>
    </header>

    <section v-if="!isAuthenticated" class="card login-card">
      <h2>Accedi come esercente</h2>
      <p class="login-hint">
        Utilizza le credenziali fornite dall'amministrazione. Dopo il login potrai aprire i link QR dei
        coupon e convalidarli automaticamente.
      </p>
      <form class="form-grid" @submit.prevent="login">
        <label>
          Username
          <input v-model.trim="loginForm.username" type="text" autocomplete="username" required />
        </label>
        <label>
          Password
          <input v-model="loginForm.password" type="password" autocomplete="current-password" required />
        </label>
        <button type="submit" class="btn btn-primary" :disabled="isLoggingIn">
          {{ isLoggingIn ? 'Accesso in corso…' : 'Entra' }}
        </button>
      </form>
      <p v-if="loginError" class="feedback feedback--error">{{ loginError }}</p>
      <p v-else-if="pendingCode" class="feedback">
        Effettua l'accesso per procedere con la validazione del coupon.
      </p>
    </section>

    <section v-else class="card validation-card">
      <div class="validation-top">
        <div>
          <p class="eyebrow">QR coupon</p>
          <h2>Validazione rapida</h2>
          <p class="subtitle">
            Se hai già scansionato il QR con la fotocamera, questa pagina leggerà i parametri presenti
            nel link e convaliderà il coupon dopo l'accesso.
          </p>
        </div>
        <div class="pill" v-if="pendingCode">
          Pronto a validare il codice <strong>{{ pendingCode }}</strong>
        </div>
      </div>

      <div class="validation-body">
        <div class="input-grid">
          <label>
            Codice coupon
            <input v-model.trim="manualCode" type="text" placeholder="Es. 2F4A9C" />
          </label>
          <label>
            Firma QR
            <input v-model.trim="manualSignature" type="text" placeholder="Firma digitale" />
          </label>
          <label>
            ID esercente (opzionale)
            <input v-model.number="manualSponsorId" type="number" min="0" placeholder="0" />
          </label>
        </div>
        <div class="actions">
          <button type="button" class="btn btn-secondary" @click="prefillFromQuery">
            Carica dati dal link
          </button>
          <button type="button" class="btn btn-primary" :disabled="isValidating" @click="validateCoupon">
            {{ isValidating ? 'Validazione…' : 'Valida coupon' }}
          </button>
        </div>
        <p v-if="validationError" class="feedback feedback--error">{{ validationError }}</p>
        <p v-if="validationInfo" class="feedback">{{ validationInfo }}</p>
      </div>

      <div v-if="validationResult" class="result">
        <div class="result-header" :class="validationResult.status">
          <span class="icon">{{ validationResult.status === 'success' ? '✅' : '⚠️' }}</span>
          <div>
            <p class="eyebrow">Esito</p>
            <h3>{{ validationResult.title }}</h3>
            <p class="subtitle">{{ validationResult.message }}</p>
          </div>
        </div>

        <dl v-if="validationResult.coupon" class="coupon-details">
          <div>
            <dt>Codice</dt>
            <dd>{{ validationResult.coupon.code }}</dd>
          </div>
          <div v-if="validationResult.coupon.claimed_at">
            <dt>Riscattato il</dt>
            <dd>{{ formatDateTime(validationResult.coupon.claimed_at) }}</dd>
          </div>
          <div v-if="validationResult.coupon.coupon?.title">
            <dt>Offerta</dt>
            <dd>{{ validationResult.coupon.coupon.title }}</dd>
          </div>
          <div v-if="validationResult.coupon.coupon?.sponsor_id">
            <dt>Sponsor</dt>
            <dd>#{{ validationResult.coupon.coupon.sponsor_id }}</dd>
          </div>
        </dl>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue';
import { apiClient } from '../api';

const props = defineProps({
  currentPath: { type: String, default: '/' },
  currentSearch: { type: String, default: '' },
});

const loginForm = reactive({
  username: '',
  password: '',
});
const token = ref(localStorage.getItem('partnerToken') || '');
const activeUsername = ref(localStorage.getItem('partnerUsername') || '');
const isLoggingIn = ref(false);
const loginError = ref('');

const manualCode = ref('');
const manualSignature = ref('');
const manualSponsorId = ref(0);
const isValidating = ref(false);
const validationError = ref('');
const validationInfo = ref('');
const validationResult = ref(null);

const pendingParams = ref({ code: '', signature: '', sponsorId: 0 });

const isAuthenticated = computed(() => Boolean(token.value));
const pendingCode = computed(() => pendingParams.value.code || '');

const authHeaders = computed(() => ({
  headers: {
    Authorization: token.value ? `Bearer ${token.value}` : '',
  },
}));

function parseQueryParams(rawSearch) {
  const params = new URLSearchParams(rawSearch || '');
  const code = (params.get('c') || '').trim();
  const signature = (params.get('s') || '').trim();
  const sponsorIdRaw = params.get('m') || params.get('sp') || '';
  const sponsorId = Number.parseInt(sponsorIdRaw, 10);

  return {
    code,
    signature,
    sponsorId: Number.isFinite(sponsorId) && sponsorId > 0 ? sponsorId : 0,
  };
}

function prefillFromQuery() {
  const parsed = parseQueryParams(props.currentSearch || window.location.search);
  if (!parsed.code || !parsed.signature) {
    validationInfo.value = 'Nessun parametro valido trovato nel link corrente.';
    return;
  }
  manualCode.value = parsed.code;
  manualSignature.value = parsed.signature;
  manualSponsorId.value = parsed.sponsorId;
  validationInfo.value = 'Dati caricati dal link del QR.';
}

async function login() {
  if (isLoggingIn.value) return;
  loginError.value = '';
  isLoggingIn.value = true;
  try {
    const { data } = await apiClient.post('/partner/login', {
      username: loginForm.username,
      password: loginForm.password,
    });
    token.value = data.token || '';
    activeUsername.value = data.username || '';
    localStorage.setItem('partnerToken', token.value);
    localStorage.setItem('partnerUsername', activeUsername.value);
    loginForm.username = '';
    loginForm.password = '';
    validationInfo.value = pendingCode.value
      ? 'Accesso completato, puoi procedere con la validazione.'
      : 'Accesso completato.';
    if (pendingCode.value) {
      validateCoupon();
    }
  } catch (error) {
    loginError.value = error?.response?.status === 401
      ? 'Credenziali non valide.'
      : "Accesso non riuscito. Riprova.";
  } finally {
    isLoggingIn.value = false;
  }
}

function logout() {
  token.value = '';
  activeUsername.value = '';
  localStorage.removeItem('partnerToken');
  localStorage.removeItem('partnerUsername');
}

function syncPendingFromSearch() {
  const parsed = parseQueryParams(props.currentSearch || window.location.search);
  pendingParams.value = parsed;
  if (parsed.code && parsed.signature && isAuthenticated.value) {
    validateCoupon(parsed);
  }
}

function formatDateTime(raw) {
  if (!raw) return '';
  const date = new Date(raw);
  if (Number.isNaN(date.getTime())) {
    return raw;
  }
  return new Intl.DateTimeFormat('it-IT', {
    dateStyle: 'short',
    timeStyle: 'short',
  }).format(date);
}

async function validateCoupon(overrides = {}) {
  const code = (overrides.code || manualCode.value || pendingParams.value.code || '').trim();
  const signature =
    (overrides.signature || manualSignature.value || pendingParams.value.signature || '').trim();
  const sponsorId = overrides.sponsorId || manualSponsorId.value || pendingParams.value.sponsorId || 0;

  validationError.value = '';
  validationResult.value = null;
  validationInfo.value = '';

  if (!isAuthenticated.value) {
    validationError.value = 'Accedi per procedere con la validazione.';
    return;
  }
  if (!code || !signature) {
    validationError.value = 'Inserisci codice e firma del coupon o apri il link dal QR.';
    return;
  }

  isValidating.value = true;
  try {
    const { data } = await apiClient.post(
      '/partner/coupons/validate',
      {
        code,
        signature,
        merchant_id: sponsorId,
      },
      authHeaders.value,
    );
    validationResult.value = {
      status: 'success',
      title: 'Coupon valido',
      message: 'Il coupon è stato convalidato con successo.',
      coupon: data?.coupon || null,
    };
    manualCode.value = '';
    manualSignature.value = '';
    manualSponsorId.value = 0;
  } catch (error) {
    const status = error?.response?.status;
    const message = error?.response?.data?.error || 'Validazione non riuscita.';
    validationResult.value = {
      status: 'error',
      title: status === 401 ? 'Sessione non valida' : 'Coupon non valido',
      message,
      coupon: null,
    };
    if (status === 401) {
      logout();
      validationError.value = 'Sessione scaduta, effettua di nuovo l\'accesso.';
    }
  } finally {
    isValidating.value = false;
  }
}

watch(
  () => props.currentSearch,
  () => {
    syncPendingFromSearch();
  },
  { immediate: true },
);

onMounted(() => {
  syncPendingFromSearch();
});
</script>

<style scoped>
.partner-shell {
  min-height: 100vh;
  padding: 32px 24px 64px;
  max-width: 960px;
  margin: 0 auto;
  color: #e2e8f0;
}

.partner-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 16px;
}

.eyebrow {
  text-transform: uppercase;
  letter-spacing: 0.08em;
  font-size: 12px;
  color: #94a3b8;
  margin: 0 0 4px;
}

.subtitle {
  color: #cbd5e1;
  margin: 4px 0 0;
}

.partner-header h1 {
  font-size: 32px;
  margin: 0;
}

.session-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.session-user {
  margin: 0;
  color: #cbd5e1;
}

.card {
  background: rgba(15, 23, 42, 0.8);
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 16px;
  padding: 20px;
  box-shadow: 0 12px 48px rgba(0, 0, 0, 0.25);
}

.login-card,
.validation-card {
  margin-top: 12px;
}

.login-hint {
  color: #cbd5e1;
  margin: 0 0 12px;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 12px;
  margin-bottom: 12px;
}

.form-grid label {
  display: flex;
  flex-direction: column;
  gap: 6px;
  color: #cbd5e1;
}

input[type='text'],
input[type='password'],
input[type='number'] {
  padding: 10px 12px;
  border-radius: 10px;
  border: 1px solid rgba(148, 163, 184, 0.3);
  background: rgba(255, 255, 255, 0.04);
  color: #e2e8f0;
}

.btn {
  background: linear-gradient(135deg, #22c55e, #16a34a);
  color: #0f172a;
  padding: 10px 16px;
  border-radius: 12px;
  border: none;
  font-weight: 700;
  cursor: pointer;
}

.btn:hover {
  filter: brightness(1.05);
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-primary {
  background: linear-gradient(135deg, #60a5fa, #2563eb);
  color: #0f172a;
  border: none;
}

.btn-secondary {
  background: rgba(148, 163, 184, 0.2);
  color: #e2e8f0;
  border: 1px solid rgba(148, 163, 184, 0.3);
}

.validation-top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}

.pill {
  background: rgba(34, 197, 94, 0.16);
  color: #bbf7d0;
  border: 1px solid rgba(34, 197, 94, 0.5);
  padding: 10px 14px;
  border-radius: 999px;
}

.validation-body {
  border-top: 1px solid rgba(148, 163, 184, 0.2);
  padding-top: 12px;
}

.input-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 12px;
}

.actions {
  display: flex;
  gap: 12px;
  margin-top: 12px;
  flex-wrap: wrap;
}

.feedback {
  margin-top: 10px;
  color: #cbd5e1;
}

.feedback--error {
  color: #fca5a5;
}

.result {
  margin-top: 20px;
  border-top: 1px solid rgba(148, 163, 184, 0.2);
  padding-top: 16px;
}

.result-header {
  display: flex;
  gap: 12px;
  align-items: center;
  padding: 12px;
  border-radius: 12px;
  background: rgba(15, 23, 42, 0.6);
}

.result-header.success {
  border: 1px solid rgba(74, 222, 128, 0.5);
}

.result-header.error {
  border: 1px solid rgba(248, 113, 113, 0.5);
}

.icon {
  font-size: 28px;
}

.coupon-details {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 10px 16px;
  margin: 14px 0 0;
}

.coupon-details dt {
  color: #94a3b8;
  margin-bottom: 4px;
}

.coupon-details dd {
  margin: 0;
  font-weight: 600;
}

@media (max-width: 720px) {
  .partner-shell {
    padding: 24px 16px 48px;
  }
  .partner-header,
  .validation-top {
    flex-direction: column;
  }
  .actions {
    flex-direction: column;
  }
}
</style>
