import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { stripTypeScriptTypes } from 'node:module';
import * as sessions from './fanSession.js';

function fixture(handler) {
  const data = new Map();
  const window = { location: { pathname: '/club-a/new', search: '', hostname: 'localhost', protocol: 'http:', port: '' }, localStorage: { getItem: (key) => data.get(key), setItem: (key, value) => data.set(key, value) } };
  let intercept;
  const requests = [];
  const send = (method, path, body, config = {}) => {
    const request = { method, path, body, config: intercept({ ...config, headers: { ...config.headers } }) };
    requests.push(request);
    return handler(request);
  };
  const client = { defaults: { baseURL: '/api' }, interceptors: { request: { use: (callback) => { intercept = callback; } } }, get: (path, config) => send('get', path, undefined, config), post: (path, body, config) => send('post', path, body, config) };
  const axios = { create: () => client, isAxiosError: (error) => error?.isAxiosError === true };
  const source = stripTypeScriptTypes(readFileSync(new URL('./api.ts', import.meta.url), 'utf8').replaceAll('import.meta.env', '({})'))
    .replace(/^import .*;\r?\n/gm, '')
    .replace(/^export /gm, '');
  const api = new Function('axios', 'getOrCreateDeviceId', 'readFanSessionToken', 'saveFanSessionToken', 'readLegacyFanSessionToken', 'window', `${source}\nreturn { getFanSessionToken, getFanSessionHeaders, startPhoneAuth, verifyPhoneAuth, registerFanProfile, fetchFanProfile, claimFanJourneyReward, submitQuickQuizAnswer, prepareVolleyballRound, submitVolleyballRound };`)(axios, () => 'device', sessions.readFanSessionToken, sessions.saveFanSessionToken, sessions.readLegacyFanSessionToken, window);
  return { api, window, requests };
}

test('volleyball rounds keep device and club scope and submit only the action to the round endpoint', async () => {
  const { api, requests } = fixture(async () => ({ data: {} }));
  assert.equal((await api.prepareVolleyballRound(42, 'unknown')).status, 400);
  assert.equal(requests.length, 0);
  for (const gameId of ['volley-rush', 'setter-pro', 'monster-block']) {
    await api.prepareVolleyballRound(42, gameId);
    const action = { aim_x: 220, aim_y: 180, timing: .7 };
    await api.submitVolleyballRound(42, gameId, 'round-a', action);
    assert.equal(requests.at(-1).path, `/events/42/volleyball/${gameId}/rounds/round-a`);
    assert.deepEqual(requests.at(-1).body, action);
    assert.equal(requests.at(-1).config.headers['X-Organization-Slug'], 'club-a');
    assert.equal(requests.at(-1).config.headers['X-Device-ID'], 'device');
  }
});

test('OTP tokens stay with the requesting club even if navigation changes during verification', async () => {
  let finish;
  const { api, window, requests } = fixture(() => new Promise((resolve) => { finish = resolve; }));
  const verification = api.verifyPhoneAuth('+393331234567', '123456');
  window.location.pathname = '/club-b/new';
  finish({ data: { token: 'session-a' } });
  assert.equal((await verification).ok, true);
  assert.equal(requests[0].config.headers['X-Organization-Slug'], 'club-a');
  assert.deepEqual(api.getFanSessionHeaders(), {});
  window.location.pathname = '/club-a/new';
  assert.deepEqual(api.getFanSessionHeaders(), { 'X-Fan-Session': 'session-a' });
});

test('minigame claims and quiz answers carry the same round receipt without changing vote claims', async () => {
  const { api, requests } = fixture(async () => ({ data: {} }));
  await api.submitQuickQuizAnswer(42, { questionId: 1, selectedIndex: 0, responseMs: 500, deviceId: 'device', journey: true, request_id: 'quiz-round-1' });
  await api.claimFanJourneyReward(42, 'quiz', 'quiz-round-1');
  await api.claimFanJourneyReward(42, 'vote');
  assert.equal(requests[0].body.request_id, 'quiz-round-1');
  assert.deepEqual(requests[1].body, { key: 'quiz', request_id: 'quiz-round-1' });
  assert.deepEqual(requests[2].body, { key: 'vote' });
});

test('only registration sends declarations with the OTP request; login remains unchanged', async () => {
  const { api, requests } = fixture(async () => ({ data: {} }));
  const acceptance = { accepted_terms: true, age_eligibility_declared: true, minimum_age_declared: 14, legal_version: '2026-10-21.1' };
  await api.startPhoneAuth('+393331234567', 'register', acceptance);
  assert.deepEqual(requests[0].body, { ...acceptance, phone: '+393331234567', mode: 'register' });
  await api.startPhoneAuth('+393331234567', 'login', acceptance);
  assert.deepEqual(requests[1].body, { phone: '+393331234567', mode: 'login' });
});

test('registration uses and updates only the current organization session', async () => {
  const { api, window, requests } = fixture(async () => ({ data: { session_token: 'updated-b' } }));
  sessions.saveFanSessionToken(window.localStorage, 'club-a', 'session-a');
  sessions.saveFanSessionToken(window.localStorage, 'club-b', 'session-b');
  window.location.pathname = '/club-b/new';
  await api.registerFanProfile({ phone: '+393331234567', nickname: 'FanB' });
  assert.equal(requests[0].config.headers['X-Fan-Session'], 'session-b');
  assert.equal(requests[0].config.headers['X-Organization-Slug'], 'club-b');
  assert.equal(api.getFanSessionToken(), 'updated-b');
  window.location.pathname = '/club-a/new';
  assert.equal(api.getFanSessionToken(), 'session-a');
});

test('a legacy token for another organization falls back to guest without being copied', async () => {
  const { api, window, requests } = fixture(async (request) => {
    if (request.config.headers['X-Fan-Session']) throw { isAxiosError: true, response: { status: 403 } };
    return { data: { registered: false, guest_coins: 12 } };
  });
  window.localStorage.setItem('fan:session_token', 'legacy-from-another-club');
  const response = await api.fetchFanProfile(42);
  assert.equal(response.ok, true);
  assert.equal(response.data.registered, false);
  assert.equal(requests.length, 2);
  assert.equal(requests[1].config.headers['X-Fan-Session'], undefined);
  assert.equal(api.getFanSessionToken(), '');
});

test('a legacy token is retained only after the server confirms its organization', async () => {
  const { api, window } = fixture(async () => ({ data: { registered: true, user: { id: 4, organization_id: 7 } } }));
  window.localStorage.setItem('fan:session_token', 'valid-legacy');
  await api.fetchFanProfile(42);
  assert.equal(api.getFanSessionToken(), 'valid-legacy');
  window.location.pathname = '/club-b/new';
  assert.equal(api.getFanSessionToken(), '');
});
