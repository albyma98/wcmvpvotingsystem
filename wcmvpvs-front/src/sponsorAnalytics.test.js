import test from 'node:test';
import assert from 'node:assert/strict';
import { createSponsorMeasurement, sponsorVisiblePercentage } from './sponsorMeasurement.js';
import { createSponsorAnalyticsService, existingSponsorDevice } from './sponsorAnalyticsService.js';

function fixture(measureGeometryViewability = false) {
  const events = [];
  const service = {
    record: (event, scope) => events.push({ ...event, device: scope.device }),
    device: () => 'existing-device',
    flush() {},
    close() {},
  };
  let now = 0;
  let sequence = 0;
  let callback;
  let tick;
  const listeners = new Map();
  const scope = { slug: 'club', match: 42 };
  const measurement = createSponsorMeasurement({
    service, scope: () => scope, measure: (element, entry) => entry ? element.ratio : 0, measureGeometryViewability,
    clock: { timeOrigin: 1760000000000, now: () => now }, uuid: () => String(++sequence),
    win: {
      IntersectionObserver: class {
        constructor(listener) { callback = listener; }
        observe() {}
        unobserve() {}
        disconnect() {}
      },
      setInterval(listener) { tick = listener; return 1; }, clearInterval() {},
      addEventListener(name, listener) { listeners.set(name, listener); }, removeEventListener() {},
    },
    doc: {
      get visibilityState() { return scope.hidden ? 'hidden' : 'visible'; },
      addEventListener(name, listener) { listeners.set(name, listener); }, removeEventListener() {},
    },
  });
  const advance = (ms) => {
    const end = now + ms;
    while (now < end) {
      now = Math.min(end, now + 125);
      tick?.();
    }
  };
  function target(id, placement, options = {}) {
    const element = { ratio: 100, addEventListener(_, listener) { this.click = listener; }, removeEventListener() {} };
    const info = { sponsor: { sponsorId: id, organizationId: 1, creativeId: 'logo', analyticsGrant: 'signed',
      analyticsMatchId: 42 }, placement, ...options };
    measurement.track(element, info);
    callback([{ target: element }]);
    return { element, info };
  }
  function hide(value) {
    scope.hidden = value;
    listeners.get('visibilitychange')();
  }
  return { events, measurement, advance, target, hide, scope, listeners, callback };
}

const eventsOf = (events, type) => events.filter((event) => event.type === 'sponsor_' + type);

test('three home sponsors keep separate impressions through rerender, modal and background', () => {
  const { events, measurement, advance, target, hide } = fixture();
  const slots = [1, 2, 3].map((id) => target(id, 'home_footer', { clickable: id === 1 }));
  advance(750);
  slots.forEach(({ element }) => { element.ratio = 0; });
  advance(1000);
  slots.forEach(({ element, info }) => {
    element.ratio = 100;
    measurement.track(element, { ...info, sponsor: { ...info.sponsor } });
  });
  advance(750);
  hide(true);
  advance(2500);
  hide(false);
  advance(875);
  assert.equal(eventsOf(events, 'viewable').length, 0);
  advance(250);
  assert.equal(eventsOf(events, 'viewable').length, 3);
  slots[0].element.click({ isTrusted: false });
  slots[0].element.click({ isTrusted: true });
  assert.deepEqual(eventsOf(events, 'click').map((event) => event.sponsor_id), [1]);
  measurement.close();
  assert.deepEqual(eventsOf(events, 'impression').map((event) => event.sponsor_id), [1, 2, 3]);
  assert.equal(new Set(eventsOf(events, 'exposure_end').map((event) => event.exposure_id)).size, 3);
  assert.ok(eventsOf(events, 'exposure_end').every((event) => event.visible_duration_ms < 3000));
});

test('post-vote sides qualify separately and only four continuous seconds reach full view', () => {
  const { events, measurement, advance, target } = fixture();
  const left = target(5, 'post_vote_left', { fullDuration: 4000 });
  const right = target(5, 'post_vote_right', { fullDuration: 4000 });
  advance(2750);
  measurement.track(right.element, null);
  advance(1250);
  measurement.track(left.element, null);
  assert.equal(eventsOf(events, 'viewable').length, 2);
  assert.deepEqual(eventsOf(events, 'full_view').map(({ placement_id }) => placement_id), ['post_vote_left']);
  assert.deepEqual(eventsOf(events, 'exposure_end').map(({ visible_duration_ms }) => visible_duration_ms), [2750, 4000]);
});

test('all three SVG courts report geometry, while a rotation closes and replaces the creative', () => {
  const { events, measurement, advance, target, callback, listeners } = fixture();
  for (const game of ['basketball', 'playmaker', 'crossover']) {
    target(9, 'game_wall_left', { game, method: 'svg_geometry' });
    target(9, 'game_wall_right', { game, method: 'svg_geometry' });
  }
  const { element, info } = target(5, 'home_footer');
  advance(600);
  measurement.track(element, { ...info, sponsor: { ...info.sponsor, creativeId: 'replacement' } });
  callback([{ target: element }]);
  advance(1250);
  listeners.get('pagehide')();
  assert.equal(eventsOf(events, 'impression').filter((e) => e.measurement_method === 'svg_geometry').length, 6);
  assert.deepEqual(eventsOf(events, 'viewable').map((e) => e.creative_id), ['replacement']);
  assert.equal(eventsOf(events, 'exposure_end').length, 8);
  assert.ok(eventsOf(events, 'exposure_end').every((e) => e.visibility_percentage === 100));
});

test('Matchday measures continuous geometry and checkpoints without relabeling HTML viewability', () => {
  const { events, advance, target, hide } = fixture(true);
  target(9, 'game_wall_left', { game: 'basketball', method: 'svg_geometry' });
  advance(900);
  hide(true);
  hide(false);
  // The first sample after visibility resumes establishes a new baseline.
  advance(5250);
  assert.equal(eventsOf(events, 'impression')[0].geometry_sampled, true);
  assert.equal(eventsOf(events, 'geometry_viewable').length, 1);
  assert.ok(eventsOf(events, 'geometry_viewable')[0].continuous_visible_ms >= 1000);
  assert.equal(eventsOf(events, 'viewable').length, 0);
  assert.ok(eventsOf(events, 'exposure_sample').length >= 2);
});

test('SVG measurement samples transformed screen coordinates and discounts paint above the board', () => {
  const behind = {};
  const above = {};
  const svg = { contains: (element) => [above, behind].includes(element) };
  const element = {
    ownerSVGElement: svg, isConnected: true, closest: () => null, querySelector: () => null,
    getBoundingClientRect: () => ({ x: 100, y: 200, width: 10, height: 5 }),
    getBBox: () => ({ x: 0, y: 0, width: 20, height: 10 }),
    getScreenCTM: () => ({ a: .5, b: 0, c: 0, d: .5, e: 100, f: 200 }),
    contains: () => false, compareDocumentPosition: (hit) => hit === above ? 4 : 2,
  };
  const win = { getComputedStyle: () => ({ display: 'inline', opacity: '1', visibility: 'visible' }) };
  const doc = { elementFromPoint(x, y) {
    assert.ok(x > 100 && x < 110 && y > 200 && y < 205);
    return x < 105 ? behind : above;
  } };
  assert.equal(sponsorVisiblePercentage(element, { isIntersecting: true }, doc, win), 50);
});

test('batches stay ordered and reuse the same event IDs after a network or rate-limit failure', async () => {
  const requests = [];
  const delays = [];
  const service = createSponsorAnalyticsService({
    url: (path) => path,
    timers: { setTimeout(_, delay) { delays.push(delay); return 1; }, clearTimeout() {} },
    fetchRequest: async (url, options) => {
      requests.push({ url, ...JSON.parse(options.body) });
      if (requests.length === 1) throw new Error('offline');
      if (requests.length === 2) return { status: 429, headers: { get: () => '20' } };
      return { status: 200 };
    },
  });
  const scope = { slug: 'club', match: 42, device: 'browser' };
  service.record({ event_id: 'start' }, scope);
  service.record({ event_id: 'end' }, scope);
  await service.flush();
  await service.flush();
  await service.flush();
  assert.deepEqual(requests[0], requests[1]);
  assert.deepEqual(requests[1], requests[2]);
  assert.deepEqual(delays, [5000, 10000, 40000]);
  service.record({ event_id: 'other' }, { ...scope, slug: 'other' });
  await service.flush();
  assert.equal(requests[3].url, '/sponsors/analytics/events?organization_slug=other');
});

test('reach reuses the existing device across refreshes and never writes an identifier', () => {
  const win = { localStorage: { getItem: (key) => key === 'wcmvpvs:device_id' ? 'persisted-browser-uuid' : '' } };
  assert.equal(existingSponsorDevice(win), 'persisted-browser-uuid');
  assert.equal(existingSponsorDevice(win), 'persisted-browser-uuid');
  assert.equal(existingSponsorDevice({}), '');
});
