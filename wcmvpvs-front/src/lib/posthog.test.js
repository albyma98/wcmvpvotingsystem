import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';

function fixture({ pathname = '/club/new', stored = new Map(), apiKey = 'test-project-token' } = {}) {
  const requests = [], initializations = [];
  let applicationDeviceReads = 0;
  let storageAccesses = 0;
  let recordingStops = 0;
  const window = {
    location: { pathname },
    localStorage: {
      getItem(key) { storageAccesses++; return stored.get(key) ?? null; },
      setItem(key, value) { storageAccesses++; stored.set(key, value); },
    },
  };
  const posthog = {
    stopSessionRecording() { recordingStops++; },
    init(key, config, name) {
      this.config = config;
      initializations.push({ key, config, name });
      return this;
    },
    capture(event, properties = {}) {
      const payload = { event, properties };
      const result = this.config.before_send(payload);
      if (result) requests.push(result);
    },
  };
  const source = readFileSync(new URL('./posthog.js', import.meta.url), 'utf8')
    .replace(/^import .*;\r?\n/gm, '').replace(/^export /gm, '')
    .replaceAll('import.meta.env', JSON.stringify({ VITE_POSTHOG_KEY: apiKey, MODE: 'test' }));
  const api = new Function('posthog', 'getOrCreateDeviceId', 'window', `${source}\nreturn { initPostHog, getPostHog, isPostHogReady, stopPostHogForNewExperience, stopPostHogOnUntrackedPage };`)(
    posthog, () => { applicationDeviceReads++; return 'application-device'; }, window,
  );
  return { api, window, stored, requests, initializations, posthog, applicationDeviceReads: () => applicationDeviceReads, storageAccesses: () => storageAccesses, recordingStops: () => recordingStops };
}

test('new fan pages never initialize PostHog or access analytics identifiers, including on custom domains', () => {
  for (const pathname of ['/club/new', '/club/new/', '/other-club/new', '/new', '/new/']) {
    const f = fixture({ pathname });
    assert.equal(f.api.initPostHog(), null, pathname);
    f.api.stopPostHogForNewExperience();
    assert.equal(f.api.getPostHog(), null);
    assert.equal(f.api.isPostHogReady(), false);
    assert.equal(f.initializations.length, 0);
    assert.equal(f.applicationDeviceReads(), 0);
    assert.equal(f.storageAccesses(), 0);
    assert.equal(f.recordingStops(), 0);
    assert.equal(f.requests.length, 0);
  }
});

test('previously accepted analytics consent cannot enable PostHog on the new experience', () => {
  for (const pathname of ['/club/new', '/new']) {
    const stored = new Map([
      ['arenaboostx:fan-analytics-consent:v1', 'accepted'],
      ['arenaboostx:fan-analytics-actor:v1:club', 'fan_previously_saved_actor'],
    ]);
    const f = fixture({ pathname, stored });
    assert.equal(f.api.initPostHog(), null);
    assert.equal(f.initializations.length, 0);
    assert.equal(f.applicationDeviceReads(), 0);
    assert.equal(f.storageAccesses(), 0);
    assert.equal(f.requests.length, 0);
  }
});

test('entering the new experience stops legacy replay and rejects all legacy captures', () => {
  for (const pathname of ['/club/new', '/new']) {
    const f = fixture({ pathname: '/club' });
    f.api.initPostHog();
    f.api.stopPostHogForNewExperience();
    assert.equal(f.recordingStops(), 0);
    f.posthog.capture('home.opened');
    assert.equal(f.requests.length, 1);
    f.window.location.pathname = pathname;
    f.api.stopPostHogForNewExperience();
    assert.equal(f.recordingStops(), 1);
    assert.equal(f.api.getPostHog(), null);
    assert.equal(f.api.isPostHogReady(), false);
    f.posthog.capture('$snapshot', { private: 'input' });
    f.posthog.capture('vote.submitted');
    assert.equal(f.requests.length, 1);
    assert.equal(f.api.initPostHog(), null);
    assert.equal(f.initializations.length, 1);
    f.window.location.pathname = '/club';
    assert.equal(f.api.getPostHog(), f.posthog);
    assert.equal(f.api.isPostHogReady(), true);
    f.posthog.capture('home.opened');
    assert.equal(f.requests.length, 2);
  }
});

test('legacy pages retain their original SDK configuration and application identity', () => {
  const f = fixture({ pathname: '/club' });
  assert.equal(f.api.initPostHog(), f.posthog);
  assert.equal(f.api.initPostHog(), f.posthog);
  assert.equal(f.initializations.length, 1);
  assert.equal(f.initializations[0].config.disable_session_recording, false);
  assert.equal(f.initializations[0].config.bootstrap.distinctID, 'application-device');
  assert.equal(f.applicationDeviceReads(), 1);
  assert.equal(f.api.isPostHogReady(), true);
});

test('CRM skips tracking on direct entry and stops replay when entered from the master workspace', () => {
  for (const pathname of ['/admin/master/crm', '/admin/master/crm/14/opportunita/28']) {
    const direct = fixture({ pathname });
    assert.equal(direct.api.initPostHog(), null);
    assert.equal(direct.initializations.length, 0);

    const navigation = fixture({ pathname: '/admin/master/new' });
    navigation.api.initPostHog();
    navigation.window.location.pathname = pathname;
    navigation.api.stopPostHogOnUntrackedPage();
    navigation.posthog.capture('$snapshot');
    assert.equal(navigation.recordingStops(), 1);
    assert.equal(navigation.requests.length, 0);
    assert.equal(navigation.api.getPostHog(), null);
  }
});

test('a missing project token keeps PostHog disabled', () => {
  for (const pathname of ['/club', '/club/new', '/new']) {
    const f = fixture({ pathname, apiKey: '' });
    assert.equal(f.api.initPostHog(), null);
    assert.equal(f.api.getPostHog(), null);
    assert.equal(f.initializations.length, 0);
    assert.equal(f.applicationDeviceReads(), 0);
  }
});
