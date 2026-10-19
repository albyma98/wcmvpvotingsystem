import test from 'node:test';
import assert from 'node:assert/strict';
import { readFanSessionToken, saveFanSessionToken, readLegacyFanSessionToken } from './fanSession.js';

function storage() {
  const data = new Map();
  return { getItem: (key) => data.get(key), setItem: (key, value) => data.set(key, value) };
}

test('each organization keeps its own session when switching between clubs', () => {
  const store = storage();
  saveFanSessionToken(store, 'club-a', 'session-a');
  saveFanSessionToken(store, 'club-b', 'session-b');
  assert.equal(readFanSessionToken(store, 'club-a'), 'session-a');
  assert.equal(readFanSessionToken(store, 'club-b'), 'session-b');
  assert.equal(readFanSessionToken(store, 'club-c'), '');
  assert.equal(readFanSessionToken(store, ''), '');
  saveFanSessionToken(store, 'CLUB-A', 'new-a');
  assert.equal(readFanSessionToken(store, 'club-a'), 'new-a');
  assert.equal(readFanSessionToken(store, 'club-b'), 'session-b');
});

test('legacy global sessions are not used as sessions for any organization', () => {
  const store = storage();
  store.setItem('fan:session_token', 'legacy-session');
  assert.equal(readLegacyFanSessionToken(store), 'legacy-session');
  assert.equal(readFanSessionToken(store, 'club-a'), '');
  assert.equal(readFanSessionToken(store, 'club-b'), '');
  saveFanSessionToken(store, 'club-a', readLegacyFanSessionToken(store));
  assert.equal(readFanSessionToken(store, 'club-a'), 'legacy-session');
  assert.equal(readFanSessionToken(store, 'club-b'), '');
});

test('blocked browser storage does not crash session handling', () => {
  const blocked = { getItem() { throw new Error('blocked'); }, setItem() { throw new Error('blocked'); } };
  assert.equal(readFanSessionToken(blocked, 'club'), '');
  assert.equal(readLegacyFanSessionToken(blocked), '');
  assert.doesNotThrow(() => saveFanSessionToken(blocked, 'club', 'session'));
});
