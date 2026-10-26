import test from 'node:test';
import assert from 'node:assert/strict';
import { setupSfc, testVue, testWindow } from '../../test/sfcSetup.js';
import * as theme from '../organizationTheme.js';
import * as registration from '../fanRegistration.js';
import { readFileSync } from 'node:fs';

const flush = async () => { for (let i = 0; i < 8; i++) await Promise.resolve(); };
function fixture(t, overrides = {}) {
  const calls = { checks: [], starts: [], resends: [], verifies: [], saves: [] };
  const clock = testWindow();
  const api = {
    checkFanNickname: async (nickname) => { calls.checks.push(nickname); return { ok: true, data: { available: nickname !== 'Taken', suggestions: ['FreeFan', 'CurvaVIP'] } }; },
    startPhoneAuth: async (...args) => { calls.starts.push(args); return { ok: true }; },
    resendPhoneAuth: async (...args) => { calls.resends.push(args); return { ok: true }; },
    verifyPhoneAuth: async (...args) => { calls.verifies.push(args); return { ok: true }; },
    ...overrides,
  };
  const setup = setupSfc(new URL('./FanRegistrationPromptModal.vue', import.meta.url), {
    vue: testVue, '../api': api, '../organizationTheme': theme,
    '../fanRegistration': registration,
    '../legal/fan-privacy.md?raw': readFileSync(new URL('../legal/fan-privacy.md', import.meta.url), 'utf8'),
    '../legal/fan-terms.md?raw': readFileSync(new URL('../legal/fan-terms.md', import.meta.url), 'utf8'),
  }, {
    modelValue: true, organizationSlug: 'club', trigger: 'profile_overlay', walletCoins: 12,
    onSubmit: async (form) => { calls.saves.push(form); return { ok: true, wallet: 12 }; },
    onLogin: async () => ({ ok: true, wallet: 42 }),
  }, { setTimeout: clock.setTimeout, clearTimeout: clock.clearTimeout, setInterval: clock.setInterval, clearInterval: clock.clearInterval });
  t.after(() => { setup.state.reset(); setup.scope.stop(); });
  setup.state.form.ageEligibilityDeclared = true;
  return { ...setup, calls };
}

test('registration checks availability and rejects occupied or invalid nicknames before the phone', async (t) => {
  const { state, calls } = fixture(t);
  state.startForm();
  await flush();
  assert.deepEqual(calls.checks, [], 'opening registration does not request nickname suggestions');
  state.form.nickname = 'Taken';
  await state.goStep2();
  assert.equal(state.stage.value, 'step1');
  assert.equal(state.nicknameStatus.value, 'taken');
  assert.equal(state.canContinueNickname.value, false);
  state.form.nickname = 'ab';
  const checks = calls.checks.length;
  await state.goStep2();
  assert.equal(calls.checks.length, checks);
  state.form.nickname = ' FreeFan ';
  await state.goStep2();
  assert.equal(state.stage.value, 'step2');
  assert.equal(state.checkedNickname.value, 'FreeFan');
});

test('late nickname responses cannot validate a changed name or another organization', async (t) => {
  const pending = [];
  const { state, props } = fixture(t, { checkFanNickname: () => new Promise((resolve) => pending.push(resolve)) });
  state.form.nickname = 'FirstFan';
  const first = state.checkNickname();
  state.form.nickname = 'SecondFan';
  pending[0]({ ok: true, data: { available: true } });
  await first;
  assert.equal(state.canContinueNickname.value, false);
  const second = state.checkNickname();
  props.organizationSlug = 'other';
  await flush();
  pending[1]({ ok: true, data: { available: true, suggestions: ['OldClub'] } });
  await second;
  assert.equal(state.form.nickname, '');
  assert.equal(state.canContinueNickname.value, false);
});

test('registration requires terms and recovers a nickname conflict without repeating OTP', async (t) => {
  const { state, props, calls, events } = fixture(t);
  state.form.nickname = 'FreeFan';
  await state.goStep2();
  state.form.phone = '333 123 4567';
  await flush();
  await state.submit();
  assert.equal(calls.starts.length, 0);
  state.form.acceptedTerms = true;
  await state.submit();
  assert.deepEqual(calls.starts, [['+393331234567', 'register', registration.fanRegistrationAcceptance(state.form)]]);
  assert.equal(state.stage.value, 'otp');
  state.otpCode.value = '123456';
  props.onSubmit = async () => ({ ok: false, code: 'NICKNAME_TAKEN', message: 'Nickname occupato' });
  await state.verifyOtpAndSubmit();
  assert.equal(state.stage.value, 'step1');
  assert.equal(state.phoneVerified.value, true);
  state.form.nickname = 'AnotherFan';
  await state.goStep2();
  props.onSubmit = async (form) => { calls.saves.push(form); return { ok: true, wallet: 19 }; };
  await state.submit();
  assert.equal(calls.verifies.length, 1);
  assert.equal(calls.starts.length, 1);
  assert.equal(state.stage.value, 'success');
  assert.equal(state.savedCoins.value, 19);
  assert.equal(calls.saves[0].nickname, 'AnotherFan');
  assert.equal(calls.saves[0].ageEligibilityDeclared, true);
  assert.equal('gender' in calls.saves[0], false);
  state.closeSuccess();
  assert.deepEqual(events.slice(-2), [['update:modelValue', false], ['completed', 'register']]);
});

test('login edits return to login and retries do not verify an already consumed code', async (t) => {
  const { state, props, calls } = fixture(t);
  state.startLogin();
  state.form.phone = '3331234567';
  await flush();
  await state.submit();
  state.editPhone();
  assert.equal(state.stage.value, 'login');
  assert.equal(state.authMode.value, 'login');
  await state.submit();
  state.otpCode.value = '123456';
  props.onLogin = async () => ({ ok: false, message: 'Connessione interrotta' });
  await state.verifyOtpAndSubmit();
  assert.equal(state.stage.value, 'otp');
  props.onLogin = async () => ({ ok: true, wallet: 42 });
  await state.verifyOtpAndSubmit();
  assert.equal(calls.verifies.length, 1);
  assert.equal(state.stage.value, 'success');
  assert.equal(state.savedCoins.value, 42);
});

test('unavailable nickname service fails closed and can be retried', async (t) => {
  let available = false;
  const { state } = fixture(t, { checkFanNickname: async () => {
    if (!available) throw new Error('Rete assente');
    return { ok: true, data: { available: true } };
  } });
  state.form.nickname = 'CurvaFan';
  await state.goStep2();
  assert.equal(state.nicknameStatus.value, 'error');
  assert.equal(state.canContinueNickname.value, false);
  assert.equal(state.loading.value, false);
  available = true;
  await state.goStep2();
  assert.equal(state.stage.value, 'step2');
});

test('registration resends through the resend endpoint without creating the profile again', async (t) => {
  const { state, calls } = fixture(t);
  state.form.nickname = 'CurvaFan';
  await state.goStep2();
  state.form.phone = '3331234567';
  state.form.acceptedTerms = true;
  await flush();
  await state.submit();
  await state.sendOtp(true);
  assert.equal(calls.resends.length, 0);
  state.resendSeconds.value = 0;
  await state.sendOtp(true);
  assert.deepEqual(calls.resends, [['+393331234567']]);
  assert.equal(calls.starts.length, 1);
  assert.equal(state.stage.value, 'otp');
  assert.ok(state.otpNotice.value);
});

test('login to a pending profile completes registration using the verified phone', async (t) => {
  const { state, props, calls } = fixture(t);
  props.onLogin = async () => ({ ok: false, code: 'REGISTRATION_INCOMPLETE' });
  state.startLogin();
  state.form.phone = '3331234567';
  await flush();
  await state.submit();
  state.otpCode.value = '123456';
  await state.verifyOtpAndSubmit();
  assert.equal(state.stage.value, 'step1');
  assert.equal(state.authMode.value, 'register');
  state.form.nickname = 'CurvaFan';
  await state.goStep2();
  state.form.acceptedTerms = true;
  await state.submit();
  assert.equal(state.stage.value, 'success');
  assert.equal(calls.starts.length, 1);
  assert.equal(calls.verifies.length, 1);
  assert.equal(calls.saves[0].nickname, 'CurvaFan');
});

test('no nickname checks or OTP are sent before the age declaration; missing terms also blocks OTP', async (t) => {
  const { state, calls } = fixture(t);
  state.form.ageEligibilityDeclared = false;
  state.form.nickname = 'YoungFan';
  await state.checkNickname();
  await state.goStep2();
  assert.equal(calls.checks.length, 0);
  state.form.phone = '3331234567';
  state.form.acceptedTerms = true;
  await state.submit();
  await state.sendOtp();
  assert.equal(calls.starts.length, 0);
  state.form.ageEligibilityDeclared = true;
  state.form.acceptedTerms = false;
  await state.submit();
  assert.equal(calls.starts.length, 0);
});

test('documents use the selected club, keep declarations and return to the current form', async (t) => {
  const { state, props, calls } = fixture(t);
  props.clubName = 'Club Basket';
  state.stage.value = 'step2';
  state.form.nickname = 'CurvaFan';
  state.form.phone = '3331234567';
  state.form.acceptedTerms = true;
  await state.openLegalDocument('privacy');
  assert.ok(state.legalDocumentBlocks.value.some((block) => block.text.includes('Club Basket')));
  assert.equal(state.stage.value, 'step2');
  await state.closeLegalDocument();
  assert.equal(state.form.acceptedTerms, true);
  assert.equal(state.form.ageEligibilityDeclared, true);
  assert.equal(state.form.phone, '3331234567');
  assert.equal(calls.starts.length, 0);
  props.organizationSlug = 'another-club';
  await flush();
  assert.equal(state.form.acceptedTerms, false);
  assert.equal(state.form.ageEligibilityDeclared, false);
  assert.equal(state.activeLegalDocument.value, '');
});

test('an expired verification returns to the SMS step while retaining accepted terms', async (t) => {
  const { state, props, calls } = fixture(t);
  state.form.nickname = 'CurvaFan';
  await state.goStep2();
  state.form.phone = '3331234567';
  state.form.acceptedTerms = true;
  await flush();
  await state.submit();
  state.otpCode.value = '123456';
  props.onSubmit = async () => ({ ok: false, code: 'PHONE_VERIFICATION_REQUIRED' });
  await state.verifyOtpAndSubmit();
  assert.equal(state.stage.value, 'step2');
  assert.equal(state.phoneVerified.value, false);
  assert.equal(state.form.acceptedTerms, true);
  await state.submit();
  assert.equal(calls.starts.length, 2);
});

test('login telemetry measures confirmed OTP and authentication without private form data', async (t) => {
  let verified = false;
  const { state, events } = fixture(t, { verifyPhoneAuth: async () => verified ? { ok: true } : { ok: false, code: 'INVALID_OTP', message: 'private failure payload' } });
  state.startLogin();
  state.form.phone = '3331234567';
  await flush();
  await state.submit();
  state.resendSeconds.value = 0;
  await state.sendOtp(true);
  state.otpCode.value = '123456';
  await state.verifyOtpAndSubmit();
  let signals = events.filter(([name]) => name === 'telemetry').map(([, signal]) => signal);
  assert.equal(signals.some(({ name }) => name === 'login_completed'), false);
  assert.ok(signals.some(({ name, metadata }) => name === 'login_failed' && metadata.reason === 'otp_invalid'));
  verified = true;
  await state.verifyOtpAndSubmit();
  signals = events.filter(([name]) => name === 'telemetry').map(([, signal]) => signal);
  assert.equal(signals.filter(({ name }) => name === 'login_completed').length, 1);
  for (const outcome of ['otp_requested', 'otp_resent', 'otp_verified', 'details_submitted']) assert.ok(signals.some(({ metadata }) => metadata.outcome === outcome));
  assert.equal(signals.some(({ name }) => name === 'registration_completed'), false);
  for (const value of ['3331234567', '123456', 'private failure payload', 'phone', 'otpCode', 'nickname', 'acceptedTerms']) assert.equal(JSON.stringify(signals).includes(value), false);
});

test('registration telemetry excludes typed details and deduplicates dismissal', async (t) => {
  const { state, events } = fixture(t);
  state.startForm();
  state.form.nickname = 'PrivateNickname';
  await state.goStep2();
  state.closeAsLater(); state.closeAsLater();
  const signals = events.filter(([name]) => name === 'telemetry').map(([, signal]) => signal);
  assert.ok(signals.some(({ name, metadata }) => name === 'registration_step' && metadata.outcome === 'advanced'));
  assert.equal(signals.filter(({ name }) => name === 'registration_dismissed').length, 1);
  assert.equal(JSON.stringify(signals).includes('PrivateNickname'), false);
});

test('late OTP responses cannot emit analytics for a different society', async (t) => {
  let resolveOtp;
  const { state, props, events } = fixture(t, { startPhoneAuth: () => new Promise((resolve) => { resolveOtp = resolve; }) });
  state.startLogin();
  state.form.phone = '3331234567';
  await flush();
  const request = state.submit();
  props.organizationSlug = 'new-club';
  await flush();
  const count = events.length;
  resolveOtp({ ok: true });
  await request;
  assert.equal(events.length, count);
});
