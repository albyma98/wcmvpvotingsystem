import test from 'node:test';
import assert from 'node:assert/strict';
import { FAN_ANALYTICS_EVENTS, FAN_ANALYTICS_SCHEMA_VERSION, sanitizeFanAnalyticsMetadata } from './fanAnalytics.js';

test('the fan analytics contract strips form values, arbitrary strings, URLs and nested data', () => {
  const safe = sanitizeFanAnalyticsMetadata({
    section: 'profile', mode: 'register', trigger: 'after_vote', outcome: 'otp_verified', count: 2,
    player_id: 7, previous_player_id: 8, is_change: true, registered: true,
    game_id: 'tap-volley', sponsor_id: 4, placement: 'home_partner', source: 'arena',
    duration_ms: 1234, schema_version: FAN_ANALYTICS_SCHEMA_VERSION, device_type: 'mobile', browser_family: 'Safari',
    phone: '+393331234567', nickname: 'CurvaFan', otp: '123456', name: 'Personal Name',
    error: 'Private API response', ticket_code: '0017', qr_data: 'https://example.test/?token=secret',
    url: 'https://example.test/?phone=123', token: 'secret', suggestion: 'My private comment',
    details: { phone: '+39333' }, arbitrary: ['sensitive'],
  });
  assert.deepEqual(safe, { section: 'profile', mode: 'register', trigger: 'after_vote', outcome: 'otp_verified', count: 2, player_id: 7, previous_player_id: 8, is_change: true, registered: true, game_id: 'tap-volley', sponsor_id: 4, placement: 'home_partner', source: 'arena', duration_ms: 1234, schema_version: 1, device_type: 'mobile', browser_family: 'Safari' });
});

test('only known enums, public IDs, bounded numbers and validated round/date values survive', () => {
  assert.deepEqual(sanitizeFanAnalyticsMetadata({
    section: '+393331234567', reason: 'raw form input', error_code: 'server response with email',
    mode: 'phone', outcome: 'private text', source: 'https://private.test',
    game_id: 'unregistered-game', item_key: 'secret', player_id: -1, sponsor_id: '42',
    count: Infinity, duration_ms: -1, seconds: 60, score: 1e12,
    round_id: 'https://example.test/qr', started_at: 'My address',
    registered: 'yes', app_version: '+393331234567',
  }), {});
  assert.deepEqual(sanitizeFanAnalyticsMetadata({
    round_id: 'round_12345678', started_at: '2026-10-25T12:00:00.000Z', seconds: 15,
    reason: 'network', status_code: 503, visible_seconds: 130, active_seconds: 75,
  }), { round_id: 'round_12345678', started_at: '2026-10-25T12:00:00.000Z', seconds: 15, reason: 'network', status_code: 503, visible_seconds: 130, active_seconds: 75 });
  assert.equal(FAN_ANALYTICS_EVENTS.has('vote_changed'), true);
  assert.equal(FAN_ANALYTICS_EVENTS.has('input_changed'), false);
});
