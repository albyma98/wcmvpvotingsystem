import test from 'node:test';
import assert from 'node:assert/strict';
import { setupSfc, testVue, testWindow } from '../../../test/sfcSetup.js';
import * as crossover from '../../games/crossover.js';

const sessionId = '00000000-0000-0000-0000-000000000042';
const golden = [[0, 0], [120, .6], [320, .6], [460, -.8]];
const flush = async () => { for (let index = 0; index < 8; index += 1) await Promise.resolve(); };
function deferred() {
  let resolve;
  const promise = new Promise((done) => { resolve = done; });
  return { promise, resolve };
}

function fixture(overrides = {}, { rect = { left: 10, top: 20, width: 360, height: 560 }, props = {}, storage = new Map() } = {}) {
  const window = testWindow(storage);
  const frames = new Map();
  let now = 0;
  let nextFrame = 1;
  window.performance = { now: () => now };
  window.requestAnimationFrame = (callback) => { const id = nextFrame++; frames.set(id, callback); return id; };
  window.cancelAnimationFrame = (id) => frames.delete(id);
  const calls = { start: [], submit: [], captured: [], released: [] };
  const unmount = [];
  function receipt(payload, extra = {}) {
    const result = crossover.evaluateCrossoverDuel({ duelIndex: payload.duel, samples: payload.samples });
    const won = result?.won || false;
    return {
      session_id: sessionId, duel: payload.duel, won, clean: result?.clean || false, quality: result?.quality || 0,
      reason: result?.reason || 'no_feint', wallet: won ? 18 : 17, coins_earned: won ? 1 : 0,
      crossover_earned: won ? payload.duel + 1 : 0, next_duel: payload.duel + 1, remaining_ms: Math.max(0, 25000 - now),
      wins: won ? payload.duel + 1 : 0, streak: won ? payload.duel + 1 : 0, best_streak: won ? payload.duel + 1 : 0,
      quality_total: won ? (payload.duel + 1) * result.quality : 0, ...extra,
    };
  }
  const setup = setupSfc(new URL('./CrossoverRushGame.vue', import.meta.url), {
    vue: { ...testVue, onBeforeUnmount: (callback) => unmount.push(callback) },
    '../../api': {
      startCrossoverSession: async (...args) => {
        calls.start.push(args);
        return overrides.startCrossoverSession ? overrides.startCrossoverSession(...args) : { ok: true, data: { session_id: sessionId, remaining_ms: 25000, next_duel: 0, wins: 0, streak: 0, best_streak: 0, quality_total: 0 } };
      },
      submitCrossoverDuel: async (...args) => {
        calls.submit.push(args);
        return overrides.submitCrossoverDuel ? overrides.submitCrossoverDuel(...args) : { ok: true, data: receipt(args[2]) };
      },
    },
    '../../games/crossover': crossover,
  }, { eventId: 42, walletCoins: 17, earnedCoins: 0, clubName: 'Club', organizationSlug: 'club', ...props }, { window });
  setup.state.courtEl.value = {
    getBoundingClientRect: () => rect, focus() {},
    setPointerCapture: (id) => calls.captured.push(id), releasePointerCapture: (id) => calls.released.push(id),
  };
  function point(x, y = 525, pointerId = 7) {
    const scale = Math.min(rect.width / 360, rect.height / 560);
    return {
      pointerId, isPrimary: true, button: 0, preventDefault() {},
      clientX: rect.left + (rect.width - 360 * scale) / 2 + x * scale,
      clientY: rect.top + (rect.height - 560 * scale) / 2 + y * scale,
    };
  }
  function frame(at) {
    now = at;
    const queued = [...frames.values()]; frames.clear();
    for (const callback of queued) callback(at);
  }
  function timers(at = now) {
    now = at;
    const queued = [...window.timers.values()]; window.timers.clear();
    for (const callback of queued) callback();
  }
  function trace(points = golden, frameStep = 0) {
    const start = now;
    setup.state.onPointerDown(point(180));
    let nextTick = frameStep;
    for (const [time, x] of points.slice(1)) {
      while (frameStep > 0 && nextTick < time) { frame(start + nextTick); nextTick += frameStep; }
      now = start + time;
      setup.state.onPointerMove(point(180 + 104 * x));
    }
  }
  const key = (key) => setup.state.onCourtKeydown({ key, preventDefault() {}, stopPropagation() {} });
  let unmounted = false;
  function dispose() {
    if (unmounted) return;
    unmounted = true;
    for (const callback of unmount) callback();
    setup.scope.stop();
  }
  return { ...setup, window, frames, calls, receipt, point, trace, key, frame, timers, setNow: (at) => { now = at; }, now: () => now, dispose };
}

test('intro waits for a start and restores earned match coins without granting any', async (t) => {
  const game = fixture({}, { props: { earnedCoins: 7 } });
  t.after(game.dispose);
  assert.equal(game.state.state.value, 'intro');
  assert.equal(game.state.matchEarned.value, 7);
  assert.equal(game.calls.start.length, 0);
  assert.equal(game.frames.size, 0);
  assert.equal(game.events.length, 0);
  await game.state.startSession();
  assert.equal(game.state.state.value, 'ready');
  assert.equal(game.state.remainingMs.value, 25000);
  assert.deepEqual(game.calls.start, [[42]]);
  assert.equal(game.events.some(([name]) => name === 'credited'), false);
});

test('duplicate starts are ignored and resumed counters come from the server', async (t) => {
  const server = deferred();
  const game = fixture({ startCrossoverSession: () => server.promise });
  t.after(game.dispose);
  const start = game.state.startSession();
  void game.state.startSession();
  assert.equal(game.calls.start.length, 1);
  server.resolve({ ok: true, data: { session_id: sessionId, remaining_ms: 12000, next_duel: 2, wins: 1, streak: 0, best_streak: 1, quality_total: 89 } });
  await start;
  assert.equal(game.state.duelIndex.value, 2);
  assert.equal(game.state.elapsedMs.value, 13000);
  assert.equal(game.state.wins.value, 1);
  assert.equal(game.state.bestStreak.value, 1);
  assert.equal(game.state.streak.value, 0);
  assert.equal(game.state.qualityTotal.value, 89);
  game.frame(2500);
  assert.equal(game.state.remainingMs.value, 9500);
});

test('upper court, cancellation, taps and foreign pointers cannot consume a duel', async (t) => {
  const game = fixture();
  t.after(game.dispose);
  await game.state.startSession();
  game.state.onPointerDown(game.point(180, 200));
  assert.equal(game.state.state.value, 'ready');
  game.state.onPointerDown(game.point(180));
  game.setNow(120);
  game.state.onPointerMove(game.point(242, 525, 9));
  assert.equal(game.state.controlX.value, 0);
  game.state.cancelControl(game.point(180, 525, 9));
  assert.equal(game.state.state.value, 'controlling');
  game.state.cancelControl(game.point(180));
  assert.equal(game.state.state.value, 'ready');
  game.state.onPointerDown(game.point(180));
  game.state.onPointerUp(game.point(180));
  game.state.onPointerDown(game.point(180));
  game.state.onLostCapture(game.point(180));
  await flush();
  assert.equal(game.state.state.value, 'ready');
  assert.equal(game.state.duelIndex.value, 0);
  assert.equal(game.state.pendingDuel.value, null);
  assert.equal(game.calls.submit.length, 0);
  assert.equal(game.events.some(([name]) => name === 'credited'), false);
});

test('the golden thumb feint launches a predicted dunk and credits only after server acknowledgement', async (t) => {
  const server = deferred();
  const game = fixture({ submitCrossoverDuel: () => server.promise });
  t.after(game.dispose);
  await game.state.startSession();
  game.trace();
  assert.equal(game.state.state.value, 'animating');
  assert.equal(game.state.outcome.value.won, true);
  assert.equal(game.state.wins.value, 0);
  assert.equal(game.state.matchEarned.value, 0);
  assert.equal(game.state.celebration.value, false);
  assert.equal(game.calls.submit.length, 0);
  game.frame(2059);
  assert.equal(game.calls.submit.length, 0);
  game.frame(2060);
  await flush();
  assert.equal(game.state.state.value, 'saving');
  assert.equal(game.events.some(([name]) => name === 'credited'), false);
  assert.equal(game.calls.submit.length, 1);
  const [eventId, savedSession, payload] = game.calls.submit[0];
  assert.equal(eventId, 42);
  assert.equal(savedSession, sessionId);
  assert.equal(payload.duel, 0);
  assert.equal(payload.duel_start_ms, 0);
  assert.deepEqual(payload.samples.map((sample) => sample.t_ms), [0, 120, 320, 460]);
  for (let index = 0; index < payload.samples.length; index += 1) assert.ok(Math.abs(payload.samples[index].x - golden[index][1]) < .000001);
  const receipt = game.receipt(payload, { crossover_earned: 5 });
  server.resolve({ ok: true, data: receipt });
  await flush();
  assert.equal(game.state.wins.value, 1);
  assert.equal(game.state.matchEarned.value, 5);
  assert.equal(game.state.duelIndex.value, 1);
  assert.equal(game.state.celebration.value, true);
  assert.deepEqual(game.events, [['round-started', { roundId: sessionId }], ['saving-change', true], ['credited', { ...receipt, event_id: 42 }], ['saving-change', false]]);
});

test('phone and letterboxed SVG coordinates produce the same normalized thumb trace', async (t) => {
  for (const rect of [
    { left: 0, top: 80, width: 320, height: 500 },
    { left: 50, top: 100, width: 390, height: 420 },
    { left: 0, top: 0, width: 300, height: 650 },
  ]) {
    const game = fixture({}, { rect });
    t.after(game.dispose);
    await game.state.startSession();
    game.trace();
    assert.equal(game.state.outcome.value.won, true);
    const samples = game.state.pendingDuel.value.samples;
    assert.deepEqual(samples.map((sample) => sample.t_ms), [0, 120, 320, 460]);
    for (let index = 0; index < samples.length; index += 1) assert.ok(Math.abs(samples[index].x - golden[index][1]) < .000001);
    assert.equal(game.calls.submit.length, 0);
  }
});

test('the same input event trace has the same quality at sparse and dense animation frame rates', async (t) => {
  const sparse = fixture(); const dense = fixture();
  t.after(sparse.dispose); t.after(dense.dispose);
  await sparse.state.startSession(); await dense.state.startSession();
  sparse.trace(); dense.trace(golden, 16);
  assert.equal(sparse.state.outcome.value.won, dense.state.outcome.value.won);
  assert.equal(sparse.state.outcome.value.quality, dense.state.outcome.value.quality);
  assert.equal(sparse.state.outcome.value.clean, dense.state.outcome.value.clean);
  assert.ok(Math.abs(sparse.state.outcome.value.cutAtMs - dense.state.outcome.value.cutAtMs) < .000001);
});

test('holding a feint exposes the instruction to change sides', async (t) => {
  const game = fixture();
  t.after(game.dispose);
  await game.state.startSession();
  game.state.onPointerDown(game.point(180));
  game.setNow(120); game.state.onPointerMove(game.point(180 + 104 * .6));
  game.setNow(320); game.state.onPointerMove(game.point(180 + 104 * .6));
  assert.equal(game.state.feintReady.value, true);
});

test('the server result overrides the predicted win without a local coin increment', async (t) => {
  let game;
  game = fixture({ submitCrossoverDuel: async (eventId, id, payload) => ({ ok: true, data: game.receipt(payload, { won: false, clean: false, quality: 0, reason: 'blocked', wallet: 17, coins_earned: 0, crossover_earned: 0, wins: 0, streak: 0, best_streak: 0, quality_total: 0 }) }) });
  t.after(game.dispose);
  await game.state.startSession();
  game.trace(); game.frame(2060);
  await flush();
  assert.equal(game.state.wins.value, 0);
  assert.equal(game.state.matchEarned.value, 0);
  assert.equal(game.state.qualityTotal.value, 0);
  assert.equal(game.state.celebration.value, false);
  assert.equal(game.events.find(([name]) => name === 'credited')[1].coins_earned, 0);
});

test('three acknowledged duels end the run and further touch input is ignored', async (t) => {
  const game = fixture();
  t.after(game.dispose);
  await game.state.startSession();
  for (let duel = 0; duel < 3; duel += 1) {
    const start = game.now();
    game.trace(); game.frame(start + 2060);
    await flush();
    if (duel < 2) game.timers(start + 2710);
  }
  assert.equal(game.state.state.value, 'finished');
  assert.equal(game.state.duelIndex.value, 3);
  assert.equal(game.state.wins.value, 3);
  assert.equal(game.calls.submit.length, 3);
  assert.equal(game.frames.size, 0);
  game.state.onPointerDown(game.point(180));
  assert.equal(game.state.state.value, 'finished');
  assert.equal(game.state.pendingDuel.value, null);
});

test('twenty-five real seconds finish the run after a long RAF pause or before input', async (t) => {
  const game = fixture();
  t.after(game.dispose);
  await game.state.startSession();
  game.frame(60000);
  assert.equal(game.state.remainingMs.value, 0);
  assert.equal(game.state.state.value, 'finished');
  assert.equal(game.frames.size, 0);
  assert.equal(game.calls.submit.length, 0);
  const release = fixture();
  t.after(release.dispose);
  await release.state.startSession();
  release.state.onPointerDown(release.point(180));
  release.setNow(25001);
  release.state.onPointerUp(release.point(240));
  assert.equal(release.state.state.value, 'finished');
  assert.equal(release.state.pendingDuel.value, null);
  assert.equal(release.calls.submit.length, 0);
});

test('a six-second duel timeout consumes one failed attempt instead of awarding a coin', async (t) => {
  const game = fixture();
  t.after(game.dispose);
  await game.state.startSession();
  game.state.onPointerDown(game.point(180));
  game.frame(6000);
  assert.equal(game.state.state.value, 'animating');
  assert.equal(game.state.outcome.value.won, false);
  game.frame(7600);
  await flush();
  assert.equal(game.calls.submit.length, 1);
  assert.equal(game.state.duelIndex.value, 1);
  assert.equal(game.state.wins.value, 0);
  assert.equal(game.state.matchEarned.value, 0);
});

test('network retry preserves every sample and deduplicates UI progress even after the deadline', async (t) => {
  const server = deferred();
  let count = 0;
  const game = fixture({ submitCrossoverDuel: async () => ++count === 1 ? { ok: false, status: 503 } : server.promise });
  t.after(game.dispose);
  await game.state.startSession();
  game.trace(); game.frame(2060);
  await flush();
  assert.equal(game.state.state.value, 'error');
  const pending = { ...game.state.pendingDuel.value, samples: game.state.pendingDuel.value.samples.map((point) => ({ ...point })) };
  game.frame(30000);
  assert.equal(game.frames.size, 0);
  assert.equal(game.state.remainingMs.value, 0);
  assert.deepEqual(game.state.pendingDuel.value, pending);
  game.state.retry(); game.state.retry();
  await flush();
  assert.equal(game.calls.submit.length, 2);
  assert.deepEqual(game.calls.submit[0], game.calls.submit[1]);
  server.resolve({ ok: true, data: game.receipt(game.calls.submit[0][2], { coins_earned: 0, remaining_ms: 0 }) });
  await flush();
  assert.equal(game.state.state.value, 'finished');
  assert.equal(game.state.wins.value, 1);
  assert.equal(game.state.pendingDuel.value, null);
  assert.equal(game.events.filter(([name]) => name === 'credited').length, 1);
  assert.equal(game.events.find(([name]) => name === 'credited')[1].coins_earned, 0);
});

test('invalid receipts leave the original duel pending and expiry ends it without reward', async (t) => {
  let invalid;
  invalid = fixture({ submitCrossoverDuel: async (eventId, id, payload) => ({ ok: true, data: invalid.receipt(payload, { session_id: 'different-session' }) }) });
  t.after(invalid.dispose);
  await invalid.state.startSession();
  invalid.trace(); invalid.frame(2060);
  await flush();
  assert.equal(invalid.state.state.value, 'error');
  assert.ok(invalid.state.pendingDuel.value);
  assert.equal(invalid.events.some(([name]) => name === 'credited'), false);
  const expired = fixture({ submitCrossoverDuel: async () => ({ ok: false, status: 410 }) });
  t.after(expired.dispose);
  await expired.state.startSession();
  expired.trace(); expired.frame(2060);
  await flush();
  assert.equal(expired.state.state.value, 'finished');
  assert.equal(expired.state.pendingDuel.value, null);
  assert.equal(expired.events.some(([name]) => name === 'credited'), false);
});

test('keyboard feints support cancellation and an opposite cut without pointer input', async (t) => {
  const game = fixture();
  t.after(game.dispose);
  await game.state.startSession();
  game.key('ArrowRight'); game.key('Escape');
  assert.equal(game.state.state.value, 'ready');
  assert.equal(game.state.pendingDuel.value, null);
  game.key('ArrowRight');
  game.frame(100); game.frame(300);
  game.setNow(440); game.key('ArrowLeft');
  assert.equal(game.state.state.value, 'animating');
  assert.equal(game.state.outcome.value.won, true);
  assert.equal(game.calls.submit.length, 0);
});

test('unmount stops an animation before submission and ignores a late session response', async () => {
  const game = fixture();
  await game.state.startSession();
  game.trace(); game.dispose(); game.frame(30000);
  await flush();
  assert.equal(game.frames.size, 0);
  assert.equal(game.calls.submit.length, 0);
  const server = deferred();
  const starting = fixture({ startCrossoverSession: () => server.promise });
  const request = starting.state.startSession();
  starting.dispose();
  server.resolve({ ok: true, data: { session_id: sessionId, remaining_ms: 25000, next_duel: 0, wins: 0, streak: 0, best_streak: 0, quality_total: 0 } });
  await request;
  assert.equal(starting.state.session.value, null);
  assert.equal(starting.frames.size, 0);
});

test('unmount during saving ignores a late acknowledgement and clears result timers', async () => {
  const server = deferred();
  const game = fixture({ submitCrossoverDuel: () => server.promise });
  await game.state.startSession();
  game.trace(); game.frame(2060);
  await flush();
  game.dispose();
  server.resolve({ ok: true, data: game.receipt(game.calls.submit[0][2]) });
  await flush();
  assert.equal(game.state.wins.value, 0);
  assert.equal(game.state.matchEarned.value, 0);
  assert.equal(game.frames.size, 0);
  assert.equal(game.window.timers.size, 0);
  assert.equal(game.events.some(([name]) => name === 'credited'), false);
  const result = fixture();
  await result.state.startSession();
  result.trace(); result.frame(2060);
  await flush();
  assert.equal(result.window.timers.size, 1);
  result.dispose(); result.timers(2710);
  assert.equal(result.window.timers.size, 0);
  assert.equal(result.frames.size, 0);
});
