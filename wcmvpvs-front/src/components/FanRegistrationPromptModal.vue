<template>
  <Teleport to="body">
    <Transition name="fan-auth">
      <div v-if="modelValue" class="fan-auth" :style="themeStyle" @click.self="closeAsLater" @keydown.esc.prevent="closeAsLater">
        <section ref="panelRef" class="auth-panel" role="dialog" aria-modal="true" aria-labelledby="fan-auth-title" tabindex="-1" @keydown.tab="trapFocus">
          <div class="auth-grabber" aria-hidden="true"></div>
          <header class="auth-header">
            <span class="auth-eyebrow">{{ clubName || 'Il tuo profilo tifoso' }}</span>
            <button type="button" class="auth-close" aria-label="Chiudi registrazione o accesso" :disabled="loading" @click="closeAsLater">×</button>
          </header>
          <div v-if="isFormStage" class="auth-progress" :aria-label="`Passaggio ${stepNumber} di ${authMode === 'login' ? 2 : 3}`">
            <span v-for="n in (authMode === 'login' ? 2 : 3)" :key="n" :class="{ filled: n <= stepNumber }"></span>
          </div>

          <Transition name="auth-step" mode="out-in" @after-enter="focusStage">
            <div :key="stage" class="auth-body">
              <template v-if="stage === 'prompt'">
                <div class="auth-ticket" aria-hidden="true"><span>IL TUO POSTO IN CURVA</span><b>Fan Club</b><i>★ ★ ★</i><small>Una squadra. La tua passione.</small></div>
                <h2 id="fan-auth-title">{{ content.title }}</h2>
                <p class="auth-copy">{{ content.subtitle }}</p>
                <div class="auth-benefits"><span>🪙 Monete salvate</span><span>🏆 Classifica tifosi</span><span>🎁 Premi del club</span></div>
                <p v-if="trigger === 'spend_redeem'" class="auth-note">Il tuo premio: <strong>{{ rewardLabel }}</strong></p>
                <button type="button" class="auth-primary" @click="startForm">{{ content.primaryCta }} <span aria-hidden="true">→</span></button>
                <button type="button" class="auth-secondary" @click="startLogin">Ho già un profilo · Accedi</button>
                <button type="button" class="auth-link" @click="closeAsLater">Continua come ospite</button>
              </template>

              <template v-else-if="stage === 'step1'">
                <span class="auth-step-label">01 · IL TUO NOME IN TRIBUNA</span>
                <h2 id="fan-auth-title">Come ti chiami in curva?</h2>
                <p class="auth-copy">Scegli il nickname con cui comparirai nella classifica di {{ clubName || 'questa società' }}.</p>
                <form @submit.prevent="goStep2">
                  <label for="fan-auth-nickname">Il tuo nickname</label>
                  <input id="fan-auth-nickname" ref="nicknameInputRef" v-model="form.nickname" required minlength="3" maxlength="24" autocomplete="nickname" placeholder="Es. CuoreDiTifoso" :aria-invalid="nicknameStatus === 'taken' || nicknameStatus === 'invalid'" aria-describedby="fan-nickname-status" :disabled="loading" />
                  <p id="fan-nickname-status" class="auth-field-status" :class="{ ok: nicknameStatus === 'available', error: ['taken', 'invalid', 'error'].includes(nicknameStatus) }" aria-live="polite">{{ nicknameMessage }}</p>
                  <div class="auth-suggestion-head"><span>Lasciati ispirare</span><button type="button" :disabled="suggestionsLoading || loading" @click="loadSuggestions">{{ suggestionsLoading ? 'Caricamento…' : 'Altri nomi ↻' }}</button></div>
                  <div class="auth-suggestions"><button v-for="suggestion in nicknameSuggestions" :key="suggestion" type="button" :disabled="loading" @click="form.nickname = suggestion">{{ suggestion }}</button></div>
                  <p v-if="suggestionsError" class="auth-note" role="status">{{ suggestionsError }}</p>
                  <p v-if="errorMessage" class="auth-error" role="alert">{{ errorMessage }}</p>
                  <button class="auth-primary" :disabled="loading || (!canContinueNickname && nicknameStatus !== 'error')">{{ loading ? 'Verifica…' : nicknameStatus === 'error' ? 'Riprova verifica' : 'Continua' }} <span aria-hidden="true">→</span></button>
                  <button type="button" class="auth-link" :disabled="loading" @click="goPrompt">Indietro</button>
                </form>
              </template>

              <template v-else-if="stage === 'step2' || stage === 'login'">
                <span class="auth-step-label">{{ authMode === 'login' ? 'BENTORNATO IN CURVA' : '02 · RESTIAMO IN CONTATTO' }}</span>
                <h2 id="fan-auth-title">{{ authMode === 'login' ? 'Riprendi il tuo posto.' : `Ci siamo, ${form.nickname.trim()}!` }}</h2>
                <p class="auth-copy">{{ authMode === 'login' ? 'Usa il numero con cui hai creato il profilo. Ti inviamo un codice per accedere.' : 'Verifica il tuo numero per attivare il profilo. Lo usiamo anche per contattarti se vinci un premio.' }}</p>
                <form @submit.prevent="submit">
                  <label for="fan-auth-phone">Numero di telefono</label>
                  <div class="auth-phone"><span>+39</span><input id="fan-auth-phone" ref="phoneInputRef" v-model="form.phone" required type="tel" inputmode="tel" autocomplete="tel-national" placeholder="333 123 4567" :disabled="loading" /></div>
                  <label v-if="authMode === 'register'" class="auth-consent"><input v-model="form.acceptedTerms" type="checkbox" required :disabled="loading" /><span>Acconsento alla gestione del profilo tifoso e alla consegna dei premi.</span></label>
                  <p v-if="errorMessage" class="auth-error" role="alert">{{ errorMessage }}</p>
                  <button v-if="phoneAlreadyExists" type="button" class="auth-secondary" :disabled="loading" @click="startLogin">Accedi con questo numero</button>
                  <button class="auth-primary" :disabled="loading || !isPhoneValid || (authMode === 'register' && !form.acceptedTerms)">{{ loading ? 'Un momento…' : phoneVerified && authMode === 'register' ? 'Attiva profilo' : 'Ricevi il codice' }} <span aria-hidden="true">→</span></button>
                  <p class="auth-note">{{ authMode === 'login' ? 'Le tue monete e i tuoi progressi ti aspettano.' : 'Bastano un numero e un codice. La tua passione è già qui.' }}</p>
                  <button type="button" class="auth-link" :disabled="loading" @click="authMode === 'login' ? goPrompt() : stage = 'step1'">Indietro</button>
                </form>
              </template>

              <template v-else-if="stage === 'otp'">
                <span class="auth-step-label">{{ authMode === 'login' ? '02' : '03' }} · VERIFICA IL NUMERO</span>
                <h2 id="fan-auth-title">Il tuo ingresso in curva.</h2>
                <p class="auth-copy">Inserisci il codice di 6 cifre inviato a <strong>{{ prettyPhone }}</strong>.</p>
                <form @submit.prevent="verifyOtpAndSubmit">
                  <label for="fan-auth-code">Codice di verifica</label>
                  <input id="fan-auth-code" ref="otpInputRef" v-model="otpCode" class="auth-otp" required maxlength="6" inputmode="numeric" pattern="[0-9]{6}" autocomplete="one-time-code" placeholder="000000" :disabled="loading" />
                  <p v-if="errorMessage" class="auth-error" role="alert">{{ errorMessage }}</p>
                  <button class="auth-primary" :disabled="loading || !isOtpValid">{{ loading ? 'Verifica…' : authMode === 'login' ? 'Accedi al profilo' : 'Attiva il mio profilo' }} <span aria-hidden="true">→</span></button>
                  <button type="button" class="auth-secondary" :disabled="loading || resendSeconds > 0" @click="sendOtp(true)">{{ resendSeconds > 0 ? `Reinvia tra ${resendSeconds}s` : 'Reinvia codice' }}</button>
                  <p v-if="otpNotice" class="auth-note" role="status">{{ otpNotice }}</p>
                  <button type="button" class="auth-link" :disabled="loading" @click="editPhone">Modifica numero</button>
                </form>
              </template>

              <template v-else>
                <div class="auth-success-mark" aria-hidden="true">✓</div>
                <span class="auth-step-label">SEI DEI NOSTRI</span>
                <h2 id="fan-auth-title">{{ authMode === 'login' ? 'Bentornato in curva!' : 'Il tuo profilo è attivo!' }}</h2>
                <p class="auth-copy">{{ authMode === 'login' ? 'Il tuo club, le tue sfide e i tuoi progressi: tutto pronto.' : 'Ora hai il tuo posto in classifica. Ci vediamo in cima!' }}</p>
                <div class="auth-saved"><span>Le tue monete al sicuro</span><strong>🪙 {{ savedCoins }}</strong></div>
                <button type="button" class="auth-primary" @click="closeSuccess">Andiamo! <span aria-hidden="true">→</span></button>
              </template>
            </div>
          </Transition>
        </section>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, reactive, ref, watch } from 'vue';
import { checkFanNickname, resendPhoneAuth, startPhoneAuth, verifyPhoneAuth } from '../api';
import { organizationThemeStyle } from '../organizationTheme';

const props = defineProps({
  modelValue: Boolean,
  trigger: { type: String, default: 'after_vote' },
  earnedCoins: { type: Number, default: 0 },
  walletCoins: { type: Number, default: 0 },
  rewardLabel: { type: String, default: 'Coupon Match Day' },
  clubName: { type: String, default: '' },
  organizationSlug: { type: String, default: '' },
  themeStyle: { type: Object, default: () => organizationThemeStyle() },
  onSubmit: { type: Function, default: null },
  onLogin: { type: Function, default: null },
});
const emit = defineEmits(['update:modelValue', 'dismissed', 'completed']);
const stage = ref('prompt');
const authMode = ref('register');
const loading = ref(false);
const errorMessage = ref('');
const phoneAlreadyExists = ref(false);
const phoneVerified = ref(false);
const form = reactive({ nickname: '', phone: '', acceptedTerms: false });
const savedCoins = ref(0);
const panelRef = ref(null);
const nicknameInputRef = ref(null);
const phoneInputRef = ref(null);
const otpInputRef = ref(null);
const otpCode = ref('');
const otpNotice = ref('');
const nicknameSuggestions = ref([]);
const nicknameStatus = ref('idle');
const nicknameMessage = ref('Da 3 a 24 caratteri. Scegli il tuo nome da tifoso.');
const checkedNickname = ref('');
const suggestionsLoading = ref(false);
const suggestionsError = ref('');
const resendSeconds = ref(0);
let nicknameTimer;
let resendTimer;
let nicknameRevision = 0;
let suggestionRevision = 0;
let sessionRevision = 0;
let returnFocus;
const nationalPhone = computed(() => String(form.phone || '').replace(/[\s()-]/g, ''));
const isPhoneValid = computed(() => /^\d{8,15}$/.test(nationalPhone.value));
const e164Phone = computed(() => `+39${nationalPhone.value}`);
const prettyPhone = computed(() => `+39 ${nationalPhone.value}`);
const isOtpValid = computed(() => /^\d{6}$/.test(otpCode.value));
const canContinueNickname = computed(() => nicknameStatus.value === 'available' && checkedNickname.value === form.nickname.trim());
const isFormStage = computed(() => ['step1', 'step2', 'login', 'otp'].includes(stage.value));
const stepNumber = computed(() => stage.value === 'otp' ? (authMode.value === 'login' ? 2 : 3) : stage.value === 'step2' ? 2 : 1);
const content = computed(() => ({
  after_vote: { title: 'Il tuo voto conta. E tu?', subtitle: 'Attiva il profilo per salvare le monete, entrare in classifica e partecipare ai premi.', primaryCta: 'Crea il tuo profilo' },
  leaderboard: { title: 'La curva ti aspetta.', subtitle: 'Scegli il tuo nome, scala la classifica e sfida gli altri tifosi.', primaryCta: 'Entra in classifica' },
  spend_redeem: { title: 'Quel premio ti aspetta.', subtitle: 'Attiva il profilo e usa le monete che hai guadagnato con il tuo tifo.', primaryCta: 'Attiva il profilo' },
  after_earn: { title: 'Tifo che vale.', subtitle: `Hai guadagnato ${props.earnedCoins} monete. Crea il profilo per tenerle sempre con te.`, primaryCta: 'Salva le tue monete' },
  profile_overlay: { title: 'Il tuo posto in curva.', subtitle: 'Una passione, un nickname. Salva i tuoi progressi e vivi ogni partita con il tuo club.', primaryCta: 'Crea il tuo profilo' },
}[props.trigger] || { title: 'Il tuo posto in curva.', subtitle: 'Salva le monete, scala la classifica e scopri i premi del tuo club.', primaryCta: 'Crea il tuo profilo' }));

function validationMessage(nickname) {
  if (nickname.length < 3 || nickname.length > 24) return 'Scegli un nickname da 3 a 24 caratteri.';
  if (!/^[A-Za-z0-9._ -]+$/.test(nickname)) return 'Usa lettere, numeri, spazi, punto, trattino o underscore.';
  return '';
}
async function checkNickname() {
  clearTimeout(nicknameTimer);
  const nickname = form.nickname.trim();
  const revision = ++nicknameRevision;
  const invalid = validationMessage(nickname);
  checkedNickname.value = '';
  if (invalid) { nicknameStatus.value = 'invalid'; nicknameMessage.value = invalid; return false; }
  nicknameStatus.value = 'checking';
  nicknameMessage.value = 'Controllo disponibilità nella società…';
  try {
    const response = await checkFanNickname(nickname);
    if (revision !== nicknameRevision || !props.modelValue) return false;
    if (!response.ok) throw new Error(response.message || 'Controllo non riuscito. Riprova tra poco.');
    checkedNickname.value = nickname;
    nicknameStatus.value = response.data?.available ? 'available' : 'taken';
    nicknameMessage.value = response.data?.available ? 'È libero. Questo posto è tuo!' : 'Già usato in questa società. Prova un altro nome.';
    if (!response.data?.available) nicknameSuggestions.value = response.data?.suggestions || [];
    return Boolean(response.data?.available);
  } catch (error) {
    if (revision !== nicknameRevision) return false;
    nicknameStatus.value = 'error'; nicknameMessage.value = error.message || 'Impossibile verificare il nickname.';
    return false;
  }
}
async function loadSuggestions() {
  const revision = ++suggestionRevision;
  suggestionsLoading.value = true; suggestionsError.value = '';
  nicknameSuggestions.value = [];
  try {
    const response = await checkFanNickname();
    if (revision !== suggestionRevision || !props.modelValue) return;
    if (!response.ok) throw new Error('Suggerimenti non disponibili. Puoi scrivere il tuo nickname o riprovare.');
    nicknameSuggestions.value = response.data?.suggestions || [];
  } catch (error) {
    if (revision === suggestionRevision) suggestionsError.value = error.message;
  } finally { if (revision === suggestionRevision) suggestionsLoading.value = false; }
}
watch(() => form.nickname, () => {
  clearTimeout(nicknameTimer); ++nicknameRevision;
  checkedNickname.value = ''; nicknameStatus.value = 'idle'; errorMessage.value = '';
  nicknameSuggestions.value = nicknameSuggestions.value.filter((name) => name.toLowerCase() !== form.nickname.trim().toLowerCase());
  const invalid = validationMessage(form.nickname.trim());
  nicknameMessage.value = invalid || 'Controllo disponibilità nella società…';
  if (invalid) { nicknameStatus.value = 'invalid'; return; }
  nicknameStatus.value = 'checking';
  nicknameTimer = setTimeout(checkNickname, 350);
}, { flush: 'sync' });
watch(() => form.phone, () => { phoneVerified.value = false; phoneAlreadyExists.value = false; errorMessage.value = ''; });
watch(() => props.modelValue, async (open) => {
  if (open) { returnFocus = typeof document !== 'undefined' ? document.activeElement : null; await focusStage(); }
  else { reset(); returnFocus?.focus?.(); }
}, { immediate: true });
watch(() => props.organizationSlug, reset);
watch(stage, async () => { errorMessage.value = ''; await focusStage(); });
async function focusStage() {
  await nextTick();
  const target = stage.value === 'step1' ? nicknameInputRef.value : ['step2', 'login'].includes(stage.value) ? phoneInputRef.value : stage.value === 'otp' ? otpInputRef.value : panelRef.value;
  target?.focus();
}
function trapFocus(event) {
  const nodes = panelRef.value?.querySelectorAll('button:not(:disabled), input:not(:disabled), [tabindex="0"]');
  if (!nodes?.length) { event.preventDefault(); return; }
  const first = nodes[0], last = nodes[nodes.length - 1];
  if (event.shiftKey && (document.activeElement === first || document.activeElement === panelRef.value)) { event.preventDefault(); last.focus(); }
  else if (!event.shiftKey && (document.activeElement === last || document.activeElement === panelRef.value)) { event.preventDefault(); first.focus(); }
}
function reset() {
  ++sessionRevision; ++nicknameRevision; ++suggestionRevision;
  clearTimeout(nicknameTimer); clearInterval(resendTimer);
  stage.value = 'prompt'; authMode.value = 'register'; loading.value = false;
  Object.assign(form, { nickname: '', phone: '', acceptedTerms: false });
  errorMessage.value = ''; phoneVerified.value = false; phoneAlreadyExists.value = false;
  savedCoins.value = 0; otpCode.value = ''; otpNotice.value = ''; resendSeconds.value = 0;
  nicknameSuggestions.value = []; suggestionsLoading.value = false; suggestionsError.value = '';
}
onBeforeUnmount(() => { ++sessionRevision; ++nicknameRevision; ++suggestionRevision; clearTimeout(nicknameTimer); clearInterval(resendTimer); returnFocus?.focus?.(); });
function startForm() { authMode.value = 'register'; stage.value = 'step1'; loadSuggestions(); if (form.nickname) checkNickname(); }
function startLogin() { authMode.value = 'login'; phoneVerified.value = false; errorMessage.value = ''; stage.value = 'login'; }
function goPrompt() { stage.value = 'prompt'; }
async function goStep2() {
  if (loading.value) return;
  const revision = sessionRevision;
  loading.value = true;
  const available = await checkNickname();
  if (revision !== sessionRevision) return;
  loading.value = false;
  if (available) stage.value = 'step2';
}
function closeAsLater() { if (loading.value) return; if (stage.value === 'success') { closeSuccess(); return; } emit('dismissed', props.trigger); emit('update:modelValue', false); }
function closeSuccess() { emit('update:modelValue', false); emit('completed', authMode.value); }
function editPhone() { phoneVerified.value = false; otpCode.value = ''; stage.value = authMode.value === 'login' ? 'login' : 'step2'; }
function mapOtpError(code) {
  return ({ INVALID_PHONE: 'Numero non valido. Controlla e riprova.', RATE_LIMITED: 'Troppi tentativi. Riprova tra qualche minuto.', PHONE_ALREADY_EXISTS: 'Questo numero ha già un profilo. Accedi per ritrovarlo.', USER_NOT_FOUND: 'Profilo non trovato. Verifica il numero o crea un profilo.', OTP_SEND_FAILED: 'Invio del codice non riuscito. Riprova tra poco.', INVALID_OTP: 'Codice non valido o scaduto. Controlla e riprova.', OTP_CHECK_FAILED: 'Verifica non riuscita. Riprova.' })[code] || 'Verifica del telefono non riuscita. Riprova.';
}
function startResendCountdown() {
  clearInterval(resendTimer); resendSeconds.value = 30;
  resendTimer = setInterval(() => { resendSeconds.value = Math.max(0, resendSeconds.value - 1); if (!resendSeconds.value) clearInterval(resendTimer); }, 1000);
}
async function submit() {
  if (loading.value || !isPhoneValid.value || (authMode.value === 'register' && !form.acceptedTerms)) return;
  if (phoneVerified.value && authMode.value === 'register') {
    loading.value = true;
    try { await finishAuth(); } catch { errorMessage.value = 'Salvataggio non riuscito. Riprova.'; } finally { loading.value = false; }
  } else await sendOtp(false);
}
async function sendOtp(isResend = false) {
  if (loading.value || !isPhoneValid.value || (isResend && resendSeconds.value > 0)) return;
  const revision = sessionRevision;
  loading.value = true; errorMessage.value = ''; otpNotice.value = ''; phoneAlreadyExists.value = false;
  try {
    const response = isResend ? await resendPhoneAuth(e164Phone.value) : await startPhoneAuth(e164Phone.value, authMode.value);
    if (revision !== sessionRevision) return;
    if (!response.ok) { errorMessage.value = mapOtpError(response.code); phoneAlreadyExists.value = response.code === 'PHONE_ALREADY_EXISTS'; return; }
    if (!isResend) { otpCode.value = ''; stage.value = 'otp'; } else otpNotice.value = 'Un nuovo codice è stato inviato.';
    startResendCountdown();
  } catch { if (revision === sessionRevision) errorMessage.value = 'Invio non riuscito. Controlla la connessione e riprova.'; }
  finally { if (revision === sessionRevision) loading.value = false; }
}
async function finishAuth() {
  const revision = sessionRevision;
  const result = authMode.value === 'login'
    ? await props.onLogin?.({ phone: e164Phone.value, trigger: props.trigger })
    : await props.onSubmit?.({ nickname: form.nickname.trim(), phone: e164Phone.value, acceptedTerms: form.acceptedTerms, trigger: props.trigger });
  if (revision !== sessionRevision) return;
  if (!result || result.ok === false) {
    if (result?.code === 'REGISTRATION_INCOMPLETE') {
      authMode.value = 'register'; stage.value = 'step1'; loadSuggestions();
      if (form.nickname) await checkNickname();
      return;
    }
    errorMessage.value = result?.message || 'Impossibile caricare il profilo. Riprova.';
    if (result?.code === 'NICKNAME_TAKEN') { stage.value = 'step1'; await checkNickname(); }
    return;
  }
  savedCoins.value = Math.max(0, Number(result.wallet ?? props.walletCoins) || 0);
  stage.value = 'success';
}
async function verifyOtpAndSubmit() {
  if (loading.value || !isOtpValid.value) return;
  const revision = sessionRevision;
  loading.value = true; errorMessage.value = '';
  try {
    if (!phoneVerified.value) {
      const response = await verifyPhoneAuth(e164Phone.value, otpCode.value);
      if (revision !== sessionRevision) return;
      if (!response.ok) { errorMessage.value = mapOtpError(response.code); return; }
      phoneVerified.value = true;
    }
    await finishAuth();
  } catch { if (revision === sessionRevision) errorMessage.value = 'Connessione non riuscita. Riprova.'; }
  finally { if (revision === sessionRevision) loading.value = false; }
}
</script>

<style scoped>
.fan-auth { position: fixed; inset: 0; z-index: 240; display: flex; align-items: flex-end; justify-content: center; background: #23161a70; backdrop-filter: blur(4px); color: var(--ink); font-family: 'Inter', system-ui, sans-serif; }
.fan-auth * { box-sizing: border-box; }
.auth-panel { width: 100%; max-width: 430px; max-height: 94dvh; overflow-y: auto; overscroll-behavior: contain; padding: 10px 24px calc(22px + env(safe-area-inset-bottom)); border-radius: 28px 28px 0 0; background: radial-gradient(90% 35% at 50% 0, var(--hero-glow), transparent 80%), var(--bg); box-shadow: 0 -8px 40px #23161a26; outline: none; }
.auth-grabber { width: 38px; height: 5px; margin: 0 auto 12px; border-radius: 99px; background: var(--red-line); }
.auth-header { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-bottom: 18px; }
.auth-eyebrow, .auth-step-label { font-size: 10px; font-weight: 800; letter-spacing: .12em; text-transform: uppercase; color: var(--red-text); }
.auth-eyebrow { overflow-wrap: anywhere; }
.auth-close { flex: none; width: 32px; height: 32px; border: 2px solid var(--line); border-radius: 50%; background: var(--card); color: var(--muted); font-size: 23px; line-height: 1; }
.auth-progress { display: flex; gap: 6px; margin-bottom: 24px; }
.auth-progress span { flex: 1; height: 4px; border-radius: 99px; background: var(--line); transition: background .25s; }
.auth-progress .filled { background: var(--red); }
.auth-body { text-align: left; }
h2 { margin: 10px 0 12px; color: var(--ink); font-family: 'Fraunces', Georgia, serif; font-size: clamp(27px, 7.4vw, 34px); font-weight: 750; line-height: 1.08; letter-spacing: -.035em; overflow-wrap: anywhere; }
.auth-copy { margin: 0 0 22px; color: var(--muted); font-size: 13px; line-height: 1.65; }
.auth-copy strong { color: var(--ink); white-space: nowrap; }
.auth-ticket { position: relative; display: flex; flex-direction: column; align-items: center; gap: 5px; width: 230px; max-width: 90%; margin: 4px auto 28px; padding: 20px 10px; background: var(--card); border: 2px solid var(--red-line); border-radius: 17px; box-shadow: 0 5px 0 var(--red-soft); transform: rotate(-4deg); }
.auth-ticket::after { content: ''; position: absolute; left: 12px; right: 12px; bottom: 40px; border-top: 2px dashed var(--line); }
.auth-ticket span { color: var(--red-text); font-size: 8px; letter-spacing: .15em; font-weight: 800; }
.auth-ticket b { font: 750 35px 'Fraunces', Georgia, serif; color: var(--ink); }
.auth-ticket i { color: var(--gold-deep); font-style: normal; letter-spacing: .25em; font-size: 15px; margin-bottom: 10px; }
.auth-ticket small { color: var(--muted); font-size: 9px; }
.auth-benefits { display: flex; flex-wrap: wrap; gap: 7px; margin-bottom: 24px; }
.auth-benefits span { padding: 8px 9px; border: 1px solid var(--line); border-radius: 11px; background: var(--card); color: var(--ink); font-size: 10px; font-weight: 700; }
button, input { font-family: inherit; }
button { cursor: pointer; -webkit-tap-highlight-color: transparent; }
button:disabled { opacity: .5; cursor: default; }
button:focus-visible, input:focus-visible { outline: 3px solid var(--red-line); outline-offset: 3px; }
.auth-primary, .auth-secondary { display: flex; width: 100%; align-items: center; justify-content: center; gap: 14px; min-height: 50px; padding: 13px 16px; border-radius: 15px; font-size: 13px; font-weight: 800; transition: transform .18s, box-shadow .18s; }
.auth-primary { margin-top: 16px; border: 2px solid var(--red); color: var(--on-primary); background: var(--red); box-shadow: 0 4px 0 var(--red-deep); }
.auth-primary span { font-size: 20px; line-height: 1; }
.auth-primary:not(:disabled):active { transform: translateY(3px); box-shadow: 0 1px 0 var(--red-deep); }
.auth-primary:disabled { box-shadow: none; }
.auth-secondary { margin-top: 12px; border: 2px solid var(--line); background: var(--card); color: var(--ink); box-shadow: 0 3px 0 #23161a0d; }
.auth-link { display: block; margin: 18px auto 0; padding: 6px; border: 0; background: transparent; color: var(--muted); font-size: 11px; font-weight: 700; }
form > label { display: block; margin-bottom: 8px; color: var(--ink); font-size: 12px; font-weight: 750; }
input:not([type=checkbox]) { width: 100%; min-width: 0; min-height: 53px; padding: 13px 14px; border: 2px solid var(--line); border-radius: 14px; background: var(--card); color: var(--ink); font-size: 16px; font-weight: 650; transition: border-color .18s; }
input:not([type=checkbox]):focus { border-color: var(--red); outline: none; }
input::placeholder { color: var(--muted); opacity: .7; }
input[aria-invalid=true] { border-color: var(--red-text); }
.auth-field-status { min-height: 34px; margin: 8px 0 12px; font-size: 11px; line-height: 1.5; color: var(--muted); }
.auth-field-status.ok { color: var(--ink); font-weight: 750; }
.auth-field-status.ok::before { content: '✓ '; color: var(--gold-deep); }
.auth-field-status.error, .auth-error { color: var(--red-text); }
.auth-suggestion-head { display: flex; align-items: center; justify-content: space-between; gap: 8px; margin-bottom: 10px; color: var(--muted); font-size: 10px; font-weight: 700; }
.auth-suggestion-head button { padding: 7px 0 7px 7px; background: transparent; border: 0; color: var(--red-text); font-size: 11px; font-weight: 750; }
.auth-suggestions { display: flex; flex-wrap: wrap; gap: 8px; min-height: 38px; }
.auth-suggestions button { border: 1.5px solid var(--red-line); border-radius: 99px; background: var(--red-soft); color: var(--red-text); padding: 9px 12px; font-size: 11px; font-weight: 700; transition: transform .16s; }
.auth-suggestions button:active { transform: scale(.95); }
.auth-phone { display: flex; align-items: center; gap: 10px; }
.auth-phone > span { padding: 16px 12px; border: 2px solid var(--line); border-radius: 14px; background: var(--card); font-size: 14px; font-weight: 750; }
form > .auth-consent { display: flex; align-items: flex-start; gap: 10px; padding: 13px; margin: 17px 0 0; border: 2px solid var(--line); border-radius: 14px; background: var(--card); font-size: 11px; font-weight: 500; line-height: 1.6; cursor: pointer; }
.auth-consent input { flex: none; width: 18px; height: 18px; margin: 0; accent-color: var(--red); }
.auth-note { margin: 14px 0 0; color: var(--muted); font-size: 10px; line-height: 1.6; text-align: center; }
.auth-error { margin: 12px 0; font-size: 12px; line-height: 1.5; }
input.auth-otp { text-align: center; font-size: 27px; letter-spacing: .25em; font-variant-numeric: tabular-nums; }
.auth-success-mark { display: grid; place-items: center; width: 74px; height: 74px; margin: 5px 0 24px; border: 2px solid var(--gold); border-radius: 24px; background: var(--gold-soft); color: var(--gold-deep); font-size: 40px; box-shadow: 0 5px 0 var(--line); animation: auth-stamp .45s cubic-bezier(.2,1.6,.5,1); }
.auth-saved { display: flex; justify-content: space-between; align-items: center; padding: 16px; margin: 20px 0; border: 2px solid var(--gold); border-radius: 15px; background: var(--gold-soft); color: var(--gold-deep); }
.auth-saved span { font-size: 11px; font-weight: 700; }
.auth-saved strong { font: 750 25px 'Fraunces', Georgia, serif; }
.fan-auth-enter-active, .fan-auth-leave-active { transition: opacity .25s; }
.fan-auth-enter-active .auth-panel, .fan-auth-leave-active .auth-panel { transition: transform .32s cubic-bezier(.32,.72,.28,1); }
.fan-auth-enter-from, .fan-auth-leave-to { opacity: 0; }
.fan-auth-enter-from .auth-panel, .fan-auth-leave-to .auth-panel { transform: translateY(100%); }
.auth-step-enter-active, .auth-step-leave-active { transition: opacity .15s, transform .18s; }
.auth-step-enter-from { opacity: 0; transform: translateY(10px); }
.auth-step-leave-to { opacity: 0; transform: translateY(-6px); }
@keyframes auth-stamp { from { opacity: 0; transform: scale(.65) rotate(-12deg); } to { opacity: 1; transform: scale(1) rotate(0); } }
@media (max-height: 650px) { .auth-panel { max-height: 97dvh; } .auth-ticket { margin-bottom: 18px; padding-block: 12px; } .auth-copy { margin-bottom: 14px; } }
@media (prefers-reduced-motion: reduce) { .fan-auth *, .fan-auth-enter-active, .fan-auth-leave-active { animation: none !important; transition: none !important; } }
</style>
