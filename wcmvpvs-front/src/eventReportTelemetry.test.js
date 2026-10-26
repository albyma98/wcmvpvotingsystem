import test from 'node:test';
import assert from 'node:assert/strict';
import { createEventReportTelemetry, createReportSignalSender, reportSource } from './eventReportTelemetry.js';

function fixture(send = async () => {}, options = {}) {
  let time = 1700000000000;
  let nextId = 0;
  const timers = new Map();
  const listeners = new Map();
  const storage = options.sessionStorage || new Map();
  const localStorage = options.localStorage || new Map();
  const observers = [];
  const document = { visibilityState: 'visible', addEventListener: (name, callback) => listeners.set(name, callback), removeEventListener: (name) => listeners.delete(name) };
  const window = {
    sessionStorage: { getItem: (key) => storage.get(key), setItem: (key, value) => storage.set(key, value) },
    localStorage: { getItem: (key) => localStorage.get(key), setItem: (key, value) => localStorage.set(key, value) },
    setTimeout: (callback, ms) => { const key = ++nextId; timers.set(key, { callback, at: time + ms }); return key; },
    clearTimeout: (key) => timers.delete(key),
    setInterval: (callback, ms) => { const key = ++nextId; timers.set(key, { callback, at: time + ms, repeat: ms }); return key; },
    clearInterval: (key) => timers.delete(key),
    addEventListener: (name, callback) => listeners.set(name, callback), removeEventListener: (name) => listeners.delete(name),
    IntersectionObserver: class {
      constructor(callback) { this.callback = callback; observers.push(this); }
      observe(element) { this.element = element; }
      disconnect() { this.disconnected = true; }
      expose(ratio) { this.callback([{ target: this.element, intersectionRatio: ratio, isIntersecting: ratio > 0 }]); }
    },
  };
  let identity = { 'X-Device-ID': 'guest' };
  const sent = [];
  const mirrors = [];
  const telemetry = createEventReportTelemetry({ eventId: options.eventId ?? 42, context: options.context || (() => ({})), canPersistVisits: options.canPersistVisits || (() => false), mirror: (event, context) => mirrors.push(structuredClone({ event, context })), window, document, now: () => time, id: () => `signal-${++nextId}`, identity: () => identity, send: async (...args) => { sent.push(args); await send(...args); } });
  async function advance(ms) {
    const until = time + ms;
    while (true) {
      const next = [...timers].filter(([, value]) => value.at <= until).sort((a, b) => a[1].at - b[1].at)[0];
      if (!next) break;
      time = next[1].at;
      timers.delete(next[0]);
      if (next[1].repeat) timers.set(next[0], { ...next[1], at: time + next[1].repeat });
      next[1].callback();
      for (let i = 0; i < 8; i++) await Promise.resolve();
    }
    time = until;
  }
  return { telemetry, sent, mirrors, storage, localStorage, window, document, observers, timers, listeners, advance, login() { identity = { 'X-Device-ID': 'guest', 'X-Fan-Session': 'verified' }; }, events() { return sent.flatMap(([, events]) => events); } };
}

test('PostHog mirroring observes sanitized actions once, without credentials or delivery retry duplicates', async () => {
  let fail = true;
  const f = fixture(async () => { if (fail) throw new Error('offline'); }, { context: () => ({ organization_slug: 'club', registered: false, source: 'arena' }) });
  const event = f.telemetry.record('vote_changed', { player_id: 7, previous_player_id: 8, is_change: true, phone: '+39333', name: 'Private', token: 'secret', nickname: 'Fan', qr_data: 'secret-url' });
  await f.telemetry.flush();
  fail = false;
  await f.telemetry.flush();
  assert.equal(f.mirrors.length, 1);
  assert.equal(f.mirrors[0].event.id, event.id);
  assert.deepEqual(event.metadata, { registered: false, source: 'arena', player_id: 7, previous_player_id: 8, is_change: true, schema_version: 1 });
  assert.doesNotMatch(JSON.stringify(f.mirrors), /X-Device|X-Fan|secret|Private|nickname|phone|qr_data/);
  assert.equal(f.telemetry.record('input_changed', { phone: '+39333' }), undefined);
  f.telemetry.stop();
});

test('visit duration distinguishes active foreground time from idle and background time, and leaves only once', async () => {
  const f = fixture();
  f.telemetry.start();
  await f.advance(30000);
  f.document.visibilityState = 'hidden'; f.listeners.get('visibilitychange')();
  await f.advance(45000);
  f.document.visibilityState = 'visible'; f.listeners.get('visibilitychange')();
  f.listeners.get('pointerdown')();
  await f.advance(15000);
  f.listeners.get('pagehide')();
  f.telemetry.stop();
  const leaves = f.mirrors.filter(({ event }) => event.name === 'page_leave');
  assert.equal(leaves.length, 1);
  assert.equal(leaves[0].event.metadata.duration_ms, 90000);
  assert.equal(leaves[0].event.metadata.active_seconds, 45);
  assert.equal(leaves[0].event.metadata.visible_seconds, 45);
  assert.equal(f.mirrors.filter(({ event }) => event.name === 'page_hidden').length, 1);
  assert.equal(f.mirrors.filter(({ event }) => event.name === 'page_visible').length, 1);
});

test('mid-visit opt-in mirrors only its seven consented seconds while internal timing keeps all twenty-two', async () => {
  const f = fixture();
  f.telemetry.resetMirrorClock(false);
  f.telemetry.start();
  await f.advance(15000);
  assert.equal(f.mirrors.length, 0);
  f.telemetry.resetMirrorClock(true);
  f.telemetry.record('page_view', { duration_ms: 15000, active_seconds: 15, visible_seconds: 15 });
  assert.equal(f.mirrors[0].event.metadata.duration_ms, 0);
  assert.equal(f.mirrors[0].event.metadata.active_seconds, 0);
  assert.equal(f.mirrors[0].event.metadata.visible_seconds, 0);
  await f.advance(7000);
  f.listeners.get('pagehide')();
  await f.telemetry.flush();
  const mirroredLeave = f.mirrors.find(({ event }) => event.name === 'page_leave').event;
  const internalLeave = f.events().find((event) => event.name === 'page_leave');
  assert.equal(mirroredLeave.metadata.duration_ms, 7000);
  assert.equal(mirroredLeave.metadata.active_seconds, 7);
  assert.equal(mirroredLeave.metadata.visible_seconds, 7);
  assert.equal(internalLeave.metadata.duration_ms, 22000);
  assert.equal(internalLeave.metadata.active_seconds, 22);
  assert.equal(internalLeave.metadata.visible_seconds, 22);
  assert.equal(f.mirrors.filter(({ event }) => event.name === 'active_time').reduce((sum, { event }) => sum + event.metadata.seconds, 0), 7);
  f.telemetry.stop();
});

test('revocation discards mirror observations and re-enabling clips the next heartbeat and accumulated lifecycle totals', async () => {
  const f = fixture();
  f.telemetry.start();
  await f.advance(5000);
  f.telemetry.resetMirrorClock(false);
  const observed = f.mirrors.length;
  await f.advance(15000);
  f.telemetry.record('vote_completed');
  assert.equal(f.mirrors.length, observed);
  f.telemetry.resetMirrorClock(true);
  await f.advance(7000);
  f.telemetry.record('game_completed', { duration_ms: 27000, elapsed_ms: 27000 });
  f.listeners.get('pagehide')();
  await f.telemetry.flush();
  const game = f.mirrors.find(({ event }) => event.name === 'game_completed').event;
  assert.equal(game.metadata.duration_ms, 7000);
  assert.equal(game.metadata.elapsed_ms, 7000);
  const mirroredLeave = f.mirrors.find(({ event }) => event.name === 'page_leave').event;
  assert.equal(mirroredLeave.metadata.duration_ms, 7000);
  assert.equal(mirroredLeave.metadata.active_seconds, 7);
  assert.equal(mirroredLeave.metadata.visible_seconds, 7);
  const active = f.mirrors.find(({ event }) => event.name === 'active_time').event.metadata;
  assert.equal(active.seconds, 7);
  assert.equal(Date.parse(active.ended_at) - Date.parse(active.started_at), 7000);
  assert.equal(f.events().find((event) => event.name === 'page_leave').metadata.active_seconds, 27);
  assert.equal(f.events().some((event) => event.name === 'vote_completed'), true);
  f.telemetry.stop();
});

test('visit returns are scoped to organization/event and use persistent counters only with opt-in', () => {
  const local = new Map(), session = new Map();
  const context = () => ({ organization_slug: 'club' });
  const first = fixture(undefined, { context, localStorage: local, sessionStorage: session });
  first.telemetry.start(); first.telemetry.stop();
  assert.equal(local.size, 0);
  const sameTab = fixture(undefined, { context, localStorage: local, sessionStorage: session });
  sameTab.telemetry.start();
  assert.equal(sameTab.mirrors.find(({ event }) => event.name === 'visit_returned').event.metadata.visit_count, 2);
  sameTab.telemetry.stop();
  const otherEvent = fixture(undefined, { context, eventId: 43, localStorage: local, sessionStorage: session });
  otherEvent.telemetry.start(); otherEvent.telemetry.stop();
  assert.equal(otherEvent.mirrors.some(({ event }) => event.name === 'visit_returned'), false);
  const optedIn = fixture(undefined, { context, localStorage: local, canPersistVisits: () => true });
  optedIn.telemetry.start(); optedIn.telemetry.stop();
  assert.equal(local.size, 1);
  const newTab = fixture(undefined, { context, localStorage: local, canPersistVisits: () => true });
  newTab.telemetry.start(); newTab.telemetry.stop();
  assert.equal(newTab.mirrors.find(({ event }) => event.name === 'visit_returned').event.metadata.visit_count, 2);
});

test('first opt-in initializes cross-tab returns without inserting another internal page view', async () => {
  let consent = false;
  const local = new Map();
  const options = { context: () => ({ organization_slug: 'club' }), localStorage: local, canPersistVisits: () => consent };
  const first = fixture(undefined, options);
  first.telemetry.setMirrorConsent(false);
  first.telemetry.start({ games_available: true });
  await first.advance(5000);
  consent = true;
  first.telemetry.setMirrorConsent(true, { games_available: true });
  await first.telemetry.flush();
  assert.equal(first.events().filter(({ name }) => name === 'page_view').length, 1);
  assert.equal(first.events().filter(({ name }) => name === 'visit_started').length, 1);
  assert.equal(first.mirrors.filter(({ event }) => event.name === 'page_view').length, 1);
  assert.equal(first.mirrors.find(({ event }) => event.name === 'visit_started').event.metadata.visit_count, 1);
  first.telemetry.setMirrorConsent(false);
  first.telemetry.setMirrorConsent(true);
  assert.equal([...local.values()][0], '1', 'reacceptance during one visit does not count another return');
  assert.equal(first.mirrors.filter(({ event }) => event.name === 'page_view').length, 1, 'reacceptance cannot inflate page views');
  first.telemetry.stop();
  const second = fixture(undefined, options);
  second.telemetry.start();
  assert.equal(second.mirrors.find(({ event }) => event.name === 'visit_returned').event.metadata.visit_count, 2);
  second.telemetry.setMirrorConsent(false);
  second.telemetry.setMirrorConsent(true);
  assert.equal(second.mirrors.filter(({ event }) => event.name === 'visit_returned').length, 1, 'one actual return stays one return');
  second.telemetry.stop();
});

test('consent chosen before readiness defers its baseline until the single real visit starts', async () => {
  const f = fixture(undefined, { canPersistVisits: () => true });
  f.telemetry.setMirrorConsent(true, { games_available: true });
  assert.equal(f.mirrors.length, 0);
  f.telemetry.start({ games_available: true });
  await f.telemetry.flush();
  assert.equal(f.mirrors.filter(({ event }) => event.name === 'page_view').length, 1);
  assert.equal(f.events().filter(({ name }) => name === 'page_view').length, 1);
  f.telemetry.stop();
});

test('generic CTA and QR visibility ignores inert panels and requires a visible second', async () => {
  const f = fixture();
  f.telemetry.start();
  let inert = true;
  const target = { closest: () => inert ? {} : null };
  f.telemetry.observeVisibility(target, 'lottery_qr_view', { placement: 'profile_ticket' }, { onceKey: 'own-ticket' });
  f.observers[0].expose(1);
  await f.advance(1200);
  assert.equal(f.mirrors.some(({ event }) => event.name === 'lottery_qr_view'), false);
  inert = false;
  f.observers[0].expose(.75);
  await f.advance(999);
  assert.equal(f.mirrors.some(({ event }) => event.name === 'lottery_qr_view'), false);
  await f.advance(1);
  assert.equal(f.mirrors.filter(({ event }) => event.name === 'lottery_qr_view').length, 1);
  f.telemetry.stop();
});

test('failed requests retry the same IDs with original guest identity and never mix an authenticated batch', async () => {
  let fail = true;
  const f = fixture(async () => { if (fail) throw new Error('offline'); });
  f.telemetry.record('page_view', { source: 'unknown' });
  await f.telemetry.flush();
  f.login();
  f.telemetry.record('registration_completed');
  fail = false;
  await f.telemetry.flush();
  assert.deepEqual(f.sent[0], f.sent[1]);
  assert.equal('X-Fan-Session' in f.sent[1][2], false);
  await f.telemetry.flush();
  assert.equal(f.sent[2][1][0].name, 'registration_completed');
  assert.equal(f.sent[2][2]['X-Fan-Session'], 'verified');
  f.telemetry.stop();
});

test('active time counts foreground activity for up to one idle minute and resumes without background time', async () => {
  const f = fixture();
  f.telemetry.start({ games_available: true });
  await f.advance(90000);
  await f.telemetry.flush();
  assert.equal(f.events().filter((event) => event.name === 'active_time').reduce((sum, event) => sum + event.metadata.seconds, 0), 60);
  f.document.visibilityState = 'hidden'; f.listeners.get('visibilitychange')();
  await f.advance(45000);
  f.document.visibilityState = 'visible'; f.listeners.get('visibilitychange')();
  f.listeners.get('pointerdown')();
  await f.advance(15000);
  await f.telemetry.flush();
  const intervals = f.events().filter((event) => event.name === 'active_time');
  assert.equal(intervals.reduce((sum, event) => sum + event.metadata.seconds, 0), 75);
  assert.ok(intervals.every((event) => event.metadata.seconds <= 15 && Date.parse(event.metadata.ended_at) - Date.parse(event.metadata.started_at) === event.metadata.seconds * 1000));
  f.telemetry.stop();
  assert.equal(f.listeners.size, 0);
});

test('sponsor views require 50% for an uninterrupted visible second and deduplicate per placement', async () => {
  const f = fixture();
  f.telemetry.start();
  const element = {};
  f.telemetry.observeSponsor(element, { sponsor_id: 7, name: 'Partner', placement: 'home' });
  f.observers[0].expose(.6);
  await f.advance(600);
  f.observers[0].expose(.1);
  await f.advance(500);
  await f.telemetry.flush();
  assert.equal(f.events().filter((event) => event.name === 'sponsor_view').length, 0);
  f.observers[0].expose(.6);
  await f.advance(600);
  f.document.visibilityState = 'hidden'; f.listeners.get('visibilitychange')();
  await f.advance(1000);
  f.document.visibilityState = 'visible'; f.listeners.get('visibilitychange')();
  await f.advance(1000);
  await f.telemetry.flush();
  assert.equal(f.events().filter((event) => event.name === 'sponsor_view').length, 1);
  f.telemetry.observeSponsor({}, { sponsor_id: 7, placement: 'home' });
  f.observers[1].expose(1);
  await f.advance(1000);
  await f.telemetry.flush();
  assert.equal(f.events().filter((event) => event.name === 'sponsor_view').length, 1);
  f.telemetry.stop();
  assert.ok(f.observers.every((observer) => observer.disconnected));
});

test('arena source requires an explicit exact query value', () => {
  assert.equal(reportSource('?source=arena'), 'arena');
  assert.equal(reportSource('?source=remote'), 'remote');
  assert.equal(reportSource('?utm_source=arena'), 'unknown');
  assert.equal(reportSource('?source=ARENA'), 'unknown');
  assert.equal(reportSource(''), 'unknown');
});

test('normal delivery uses Axios while keepalive carries the exact captured identity to the resolved API URL', async () => {
  const requests = [];
  const axiosRequests = [];
  const sender = createReportSignalSender({
    apiClient: { post: async (...args) => axiosRequests.push(args) },
    resolveApiUrl: (path) => `https://api.example.test/api${path}`,
    fetch: async (...args) => { requests.push(args); return { ok: true }; },
  });
  const headers = { 'X-Device-ID': 'device-1', 'X-Fan-Session': 'fan-old-club', 'X-Organization-Slug': 'old-club' };
  const events = [{ id: 'stable-id', name: 'vote_completed' }];
  await sender(42, events, headers);
  assert.deepEqual(axiosRequests, [['/events/42/report-signals', { events }, { headers, timeout: 10000 }]]);
  await sender(42, events, headers, { keepalive: true });
  assert.equal(requests[0][0], 'https://api.example.test/api/events/42/report-signals');
  assert.deepEqual(requests[0][1], {
    method: 'POST', credentials: 'include', keepalive: true,
    headers: { ...headers, 'Content-Type': 'application/json' }, body: JSON.stringify({ events }),
  });
  assert.equal(axiosRequests.length, 1);
});

test('pagehide resends in-flight and queued observations with stable IDs and separate captured credentials', async () => {
  let finishNormal;
  const pending = new Promise((resolve) => { finishNormal = resolve; });
  const f = fixture(async (eventId, events, headers, options) => { if (!options?.keepalive) await pending; });
  f.telemetry.start({ source: 'arena' });
  const sending = f.telemetry.flush();
  await Promise.resolve();
  f.login();
  f.telemetry.record('registration_completed');
  f.listeners.get('pagehide')();
  for (let index = 0; index < 10; index++) await Promise.resolve();
  const keepalive = f.sent.filter(([, , , options]) => options?.keepalive);
  assert.equal(keepalive.length, 2);
  assert.equal(keepalive[0][1][0].id, f.sent[0][1][0].id);
  assert.equal('X-Fan-Session' in keepalive[0][2], false);
  assert.equal(keepalive[1][2]['X-Fan-Session'], 'verified');
  assert.equal(keepalive[1][1][0].name, 'registration_completed');
  finishNormal();
  await sending;
  f.telemetry.stop();
});

test('failed teardown keepalive keeps the same observations available for normal Axios retry', async () => {
  const f = fixture(async (eventId, events, headers, options) => { if (options?.keepalive) throw new Error('keepalive rejected'); });
  f.telemetry.record('feedback_completed', { experience: 'very_easy', perks_interest: 'yes' });
  f.telemetry.stop();
  for (let index = 0; index < 10; index++) await Promise.resolve();
  assert.equal(f.sent[0][3].keepalive, true);
  await f.advance(3000);
  assert.equal(f.sent[1][3], undefined);
  assert.deepEqual(f.sent[1].slice(0, 3), f.sent[0].slice(0, 3));
});

test('HTTP failures in fetch are rejected rather than removing undelivered observations', async () => {
  const sender = createReportSignalSender({
    apiClient: { post: async () => {} }, resolveApiUrl: (path) => path,
    fetch: async () => ({ ok: false, status: 503 }),
  });
  await assert.rejects(sender(42, [], { 'X-Device-ID': 'guest' }, { keepalive: true }), /503/);
});

test('two simultaneously visible copies of a sponsor placement count once', async () => {
  const f = fixture();
  f.telemetry.start();
  for (let index = 0; index < 2; index++) f.telemetry.observeSponsor({}, { sponsor_id: 7, placement: 'home' });
  for (const observer of f.observers) observer.expose(1);
  await f.advance(1000);
  await f.telemetry.flush();
  assert.equal(f.events().filter(({ name }) => name === 'sponsor_view').length, 1);
  f.telemetry.stop();
});
