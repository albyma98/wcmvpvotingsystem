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
  const api = new Function('axios', 'getOrCreateDeviceId', 'readFanSessionToken', 'saveFanSessionToken', 'readLegacyFanSessionToken', 'window', `${source}\nreturn { getFanSessionToken, getFanSessionHeaders, verifyPhoneAuth, registerFanProfile, fetchFanProfile };`)(axios, () => 'device', sessions.readFanSessionToken, sessions.saveFanSessionToken, sessions.readLegacyFanSessionToken, window);
  return { api, window, requests };
}

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
