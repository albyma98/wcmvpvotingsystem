import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import * as contract from '../fanAnalytics.js';

function fixture({ pathname = '/club/new', stored = new Map(), navigator = {}, apiKey = 'test-project-token' } = {}) {
  const requests = [], initializations = [], changes = [];
  let applicationDeviceReads = 0;
  const window = {
    location: { pathname }, navigator,
    localStorage: { getItem: key => stored.get(key) ?? null, setItem: (key, value) => stored.set(key, value) },
    CustomEvent: class { constructor(type, options) { this.type = type; this.detail = options.detail; } },
    dispatchEvent: event => changes.push(event),
  };
  const instances = [];
  let legacyStops = 0;
  const posthog = { stopSessionRecording() { legacyStops++; }, init(key, config, name) {
    const instance = {
      optedOut: false, stops: 0,
      capture(event, properties, options) {
        if (this.optedOut) return;
        const payload = { event, uuid: 'sdk-event-id', timestamp: options?.timestamp, properties: { ...properties, $current_url: 'https://site.test/?otp=SECRET', $referrer: 'https://private.test', $set: { email: 'private@example.test' }, $device_id: 'sdk-device' } };
        const result = config.before_send ? config.before_send(payload) : payload;
        if (result) requests.push(result);
      },
      opt_in_capturing(options) { this.optedOut = false; this.optInOptions = options; },
      opt_out_capturing() { this.optedOut = true; },
      stopSessionRecording() { this.stops++; },
    };
    initializations.push({ key, config, name });
    instances.push(instance);
    return instance;
  } };
  const source = readFileSync(new URL('./posthog.js', import.meta.url), 'utf8')
    .replace(/^import .*;\r?\n/gm, '').replace(/^export /gm, '')
    .replaceAll('import.meta.env', JSON.stringify({ VITE_POSTHOG_KEY: apiKey, MODE: 'test' }));
  const api = new Function('posthog', 'getOrCreateDeviceId', 'window', 'FAN_ANALYTICS_EVENTS', 'FAN_ANALYTICS_SCHEMA_VERSION', 'sanitizeFanAnalyticsMetadata', `${source}\nreturn { initPostHog, getPostHog, isPostHogReady, getFanAnalyticsConsent, setFanAnalyticsConsent, captureFanAnalytics, prepareFanAnalytics, isFanAnalyticsAllowed };`)(posthog, () => { applicationDeviceReads++; return 'application-device'; }, window, contract.FAN_ANALYTICS_EVENTS, contract.FAN_ANALYTICS_SCHEMA_VERSION, contract.sanitizeFanAnalyticsMetadata);
  return { api, window, stored, requests, initializations, instances, changes, applicationDeviceReads: () => applicationDeviceReads, legacyStops: () => legacyStops };
}
const context = { organization_slug: 'club', event_id: 42, environment: 'test' };
const signal = (id = 'event-123', metadata = {}) => ({ id, name: 'vote_completed', session_id: 'session-123', occurred_at: '2026-10-25T12:00:00.000Z', metadata });

test('new fan analytics requires opt-in, never starts replay, and does not backfill rejected events', () => {
  const f = fixture();
  assert.equal(f.api.getFanAnalyticsConsent(), null);
  assert.equal(f.api.initPostHog(), null);
  assert.equal(f.api.captureFanAnalytics(signal('before-choice'), context), false);
  assert.equal(f.initializations.length, 0);
  assert.equal([...f.stored.keys()].some(key => key.includes('actor')), false);
  assert.equal(f.api.setFanAnalyticsConsent(true), true);
  assert.equal(f.initializations.length, 1);
  const { config, name } = f.initializations[0];
  assert.equal(name, 'fan_new');
  for (const property of ['autocapture', 'capture_pageview', 'capture_pageleave', 'capture_performance', 'capture_exceptions', 'capture_heatmaps', 'capture_dead_clicks', 'enable_recording_console_log']) assert.equal(config[property], false);
  assert.equal(config.disable_session_recording, true);
  assert.equal(config.disable_external_dependency_loading, true);
  assert.equal(config.person_profiles, 'never');
  assert.equal(f.applicationDeviceReads(), 0);
  assert.equal(f.api.getPostHog(), null, 'legacy tracker is excluded from the fan surface');
  assert.equal(f.api.captureFanAnalytics(signal('before-choice'), context), false);
  assert.equal(f.api.captureFanAnalytics(signal('after-choice'), context), true);
  assert.equal(f.api.captureFanAnalytics(signal('after-choice'), context), false);
  assert.equal(f.requests.length, 1);
  f.api.setFanAnalyticsConsent(false);
  assert.equal(f.api.captureFanAnalytics(signal('revoked'), context), false);
  assert.equal(f.instances[0].optedOut, true);
  assert.equal(f.instances[0].stops, 1);
  assert.deepEqual(f.instances[0].optInOptions, { captureEventName: false });
});

test('the transport strips SDK-added URL/person data as well as sensitive caller properties', () => {
  const f = fixture();
  f.api.setFanAnalyticsConsent(true);
  f.api.captureFanAnalytics(signal('safe', { section: 'mvp', player_id: 7, duration_ms: 350, phone: '+393331234567', nickname: 'Personal', ticket_code: '0042', qr_data: 'secret', suggestion: 'My text' }), context);
  const [event] = f.requests;
  assert.equal(event.event, 'fan_new.vote_completed');
  assert.equal(event.properties.surface, 'new');
  assert.equal(event.properties.event_id, 42);
  assert.equal(event.properties.organization_slug, 'club');
  assert.equal(event.properties.analytics_session_id, 'session-123');
  assert.match(event.properties.distinct_id, /^fan_/);
  assert.equal(event.properties.$process_person_profile, false);
  assert.equal(event.properties.$geoip_disable, true);
  const encoded = JSON.stringify(event);
  assert.doesNotMatch(encoded, /SECRET|private|nickname|phone|0042|My text|current_url|referrer|sdk-device|Personal/);
  assert.equal(f.initializations[0].config.before_send({ event: '$autocapture', properties: {} }), null);
  assert.equal(f.initializations[0].config.before_send({ event: 'fan_new.unapproved', properties: {} }), null);
});

test('opaque browser IDs persist for the same society and stay separate between societies', () => {
  const stored = new Map();
  const first = fixture({ stored }); first.api.setFanAnalyticsConsent(true);
  first.api.captureFanAnalytics(signal('first'), context);
  const firstActor = first.requests[0].properties.distinct_id;
  const restored = fixture({ stored });
  restored.api.captureFanAnalytics(signal('restored'), context);
  assert.equal(restored.requests[0].properties.distinct_id, firstActor);
  restored.window.location.pathname = '/other/new';
  restored.api.captureFanAnalytics(signal('other'), { ...context, organization_slug: 'other' });
  assert.notEqual(restored.requests[1].properties.distinct_id, firstActor);
});

test('browser DNT/GPC disables the fan integration even with an accepted preference', () => {
  for (const navigator of [{ doNotTrack: '1' }, { globalPrivacyControl: true }]) {
    const f = fixture({ navigator });
    f.api.setFanAnalyticsConsent(true);
    assert.equal(f.api.isFanAnalyticsAllowed(), false);
    assert.equal(f.api.initPostHog(), null);
    assert.equal(f.api.captureFanAnalytics(signal(), context), false);
    assert.equal(f.initializations.length, 0);
  }
});

test('missing project token never initializes or sends fan analytics', () => {
  const f = fixture({ apiKey: '' });
  f.api.setFanAnalyticsConsent(true);
  assert.equal(f.api.captureFanAnalytics(signal(), context), false);
  assert.equal(f.initializations.length, 0);
  assert.equal(f.requests.length, 0);
});

test('entering the new experience stops legacy replay and rejects legacy captures', () => {
  const f = fixture({ pathname: '/club' });
  f.api.initPostHog();
  const config = f.initializations[0].config;
  const event = { event: '$snapshot', properties: { private: 'input' } };
  assert.equal(config.before_send(event), event);
  f.window.location.pathname = '/club/new';
  f.api.prepareFanAnalytics();
  assert.equal(f.legacyStops(), 1);
  assert.equal(config.before_send(event), null);
});

test('legacy pages retain their original SDK configuration and application identity', () => {
  const f = fixture({ pathname: '/club' });
  f.api.initPostHog();
  assert.equal(f.initializations[0].config.disable_session_recording, false);
  assert.equal(f.initializations[0].config.bootstrap.distinctID, 'application-device');
  assert.equal(f.applicationDeviceReads(), 1);
  assert.equal(f.api.isPostHogReady(), true);
});

test('a custom club domain using /new waits for the explicit society context and never initializes legacy analytics', () => {
  const f = fixture({ pathname: '/new' });
  assert.equal(f.api.initPostHog(), null);
  f.api.setFanAnalyticsConsent(true);
  assert.equal(f.initializations.length, 0);
  assert.equal(f.api.captureFanAnalytics(signal(), context), true);
  assert.equal(f.initializations[0].name, 'fan_new');
  assert.equal(f.applicationDeviceReads(), 0);
  assert.equal(f.requests[0].properties.organization_slug, 'club');
});
