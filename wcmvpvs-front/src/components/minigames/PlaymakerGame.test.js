import test from 'node:test';
import assert from 'node:assert/strict';
import { setupSfc, testVue, testWindow } from '../../../test/sfcSetup.js';
import * as playmaker from '../../games/playmaker.js';

const sessionId = '00000000-0000-0000-0000-000000000042';
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
    const pass = playmaker.createPlaymakerPass({ attempt: payload.attempt, elapsedMs: payload.elapsed_ms, drag: { x: payload.dx, y: payload.dy } });
    return {
      session_id: sessionId, attempt: payload.attempt, assisted: pass?.assisted || false, clean: pass?.clean || false,
      reason: pass?.reason || 'off_target', wallet: pass?.assisted ? 18 : 17, coins_earned: pass?.assisted ? 1 : 0,
      playmaker_earned: pass?.assisted ? 1 : 0, next_attempt: payload.attempt + 1, remaining_ms: Math.max(0, 30000 - now), ...extra,
    };
  }
  const setup = setupSfc(new URL('./PlaymakerGame.vue', import.meta.url), {
    vue: { ...testVue, onBeforeUnmount: (callback) => unmount.push(callback) },
    '../../api': {
      startPlaymakerSession: async (...args) => {
        calls.start.push(args);
        return overrides.startPlaymakerSession ? overrides.startPlaymakerSession(...args) : { ok: true, data: { session_id: sessionId, remaining_ms: 30000, attempt: 0 } };
      },
      submitPlaymakerPass: async (...args) => {
        calls.submit.push(args);
        return overrides.submitPlaymakerPass ? overrides.submitPlaymakerPass(...args) : { ok: true, data: receipt(args[2]) };
      },
    },
    '../../games/playmaker': playmaker,
  }, { eventId: 42, walletCoins: 17, earnedCoins: 0, clubName: 'Club', organizationSlug: 'club', ...props }, { window });
  setup.state.courtEl.value = {
    getBoundingClientRect: () => rect, focus() {},
    setPointerCapture: (id) => calls.captured.push(id),
    releasePointerCapture: (id) => calls.released.push(id),
  };
  function point(x, y, pointerId = 7) {
    const scale = Math.min(rect.width / 360, rect.height / 560);
    return {
      pointerId, isPrimary: true, button: 0, preventDefault() {},
      clientX: rect.left + (rect.width - 360 * scale) / 2 + x * scale,
      clientY: rect.top + (rect.height - 560 * scale) / 2 + y * scale,
    };
  }
  function freeMate() {
    const current = { attempt: setup.state.attempt.value, elapsedMs: setup.state.elapsedMs.value };
    return playmaker.getPlaymakerScene(current).teammates.find((mate) => playmaker.createPlaymakerPass({ ...current, drag: { x: mate.x - 180, y: mate.y - 466 } }).assisted);
  }
  function passTo(mate = freeMate()) {
    setup.state.onPointerDown(point(180, 466));
    setup.state.onPointerMove(point(mate.x, mate.y));
    setup.state.onPointerUp(point(mate.x, mate.y));
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
  let unmounted = false;
  function dispose() {
    if (unmounted) return;
    unmounted = true;
    for (const callback of unmount) callback();
    setup.scope.stop();
  }
  return { ...setup, window, frames, calls, receipt, point, freeMate, passTo, frame, timers, setNow: (at) => { now = at; }, dispose };
}

test('intro waits for an explicit start and restores match earnings without granting a reward', async (t) => {
  const game = fixture({}, { props: { earnedCoins: 7 } });
  t.after(game.dispose);
  assert.equal(game.state.state.value, 'intro');
  assert.equal(game.state.matchEarned.value, 7);
  assert.equal(game.frames.size, 0);
  assert.equal(game.calls.start.length, 0);
  assert.equal(game.events.length, 0);
  await game.state.startSession();
  assert.equal(game.state.state.value, 'ready');
  assert.equal(game.state.remainingMs.value, 30000);
  assert.deepEqual(game.calls.start, [[42]]);
  assert.equal(game.events.some(([name]) => name === 'credited'), false);
});

test('duplicate starts are ignored and server remaining time resumes the issued session', async (t) => {
  const server = deferred();
  const game = fixture({ startPlaymakerSession: () => server.promise });
  t.after(game.dispose);
  const start = game.state.startSession();
  void game.state.startSession();
  assert.equal(game.calls.start.length, 1);
  server.resolve({ ok: true, data: { session_id: sessionId, remaining_ms: 12000, attempt: 4 } });
  await start;
  assert.equal(game.state.elapsedMs.value, 18000);
  assert.equal(game.state.attempt.value, 4);
  game.frame(2500);
  assert.equal(game.state.remainingMs.value, 9500);
});

test('pointer cancellation, lost capture, taps and foreign pointers do not consume an attempt', async (t) => {
  const game = fixture();
  t.after(game.dispose);
  await game.state.startSession();
  game.state.onPointerDown(game.point(180, 466));
  game.state.onPointerMove(game.point(82, 248, 9));
  assert.deepEqual(game.state.drag.value, { x: 0, y: 0 });
  game.state.cancelAim(game.point(82, 248, 9));
  assert.equal(game.state.state.value, 'aiming');
  game.state.cancelAim(game.point(82, 248));
  assert.equal(game.state.state.value, 'ready');
  game.state.onPointerDown(game.point(180, 466));
  game.state.onPointerUp(game.point(180, 466));
  game.state.onPointerDown(game.point(180, 466));
  game.state.onLostCapture(game.point(82, 248));
  game.frame(2000);
  await flush();
  assert.equal(game.state.state.value, 'ready');
  assert.equal(game.state.attempt.value, 0);
  assert.equal(game.state.pendingPass.value, null);
  assert.equal(game.calls.submit.length, 0);
  assert.equal(game.events.some(([name]) => name === 'credited'), false);
  assert.deepEqual(game.calls.released, [7, 7, 7]);
});

test('phone and letterboxed SVG drags target the same teammate in world coordinates', async (t) => {
  for (const rect of [
    { left: 0, top: 80, width: 320, height: 500 },
    { left: 50, top: 100, width: 390, height: 420 },
    { left: 0, top: 0, width: 300, height: 650 },
  ]) {
    const game = fixture({}, { rect });
    t.after(game.dispose);
    await game.state.startSession();
    game.passTo();
    assert.equal(game.state.activePass.value.assisted, true);
    assert.ok(Math.abs(game.state.pendingPass.value.dx + 98) < .000001);
    assert.ok(Math.abs(game.state.pendingPass.value.dy + 218) < .000001);
    assert.equal(game.calls.submit.length, 0);
  }
});

test('a predicted assist animates first and only a valid server receipt grants progress', async (t) => {
  const server = deferred();
  const game = fixture({ submitPlaymakerPass: () => server.promise });
  t.after(game.dispose);
  await game.state.startSession();
  game.passTo();
  assert.equal(game.state.state.value, 'flying');
  assert.equal(game.state.activePass.value.assisted, true);
  game.frame(1399);
  assert.equal(game.calls.submit.length, 0);
  game.frame(1400);
  await flush();
  assert.equal(game.state.state.value, 'saving');
  assert.deepEqual(game.calls.submit, [[42, sessionId, { attempt: 0, elapsed_ms: 0, dx: -98, dy: -218 }]]);
  assert.equal(game.state.sessionAssists.value, 0);
  assert.equal(game.state.matchEarned.value, 0);
  assert.equal(game.state.celebration.value, false);
  assert.equal(game.events.some(([name]) => name === 'credited'), false);
  const receipt = game.receipt(game.calls.submit[0][2], { playmaker_earned: 5 });
  server.resolve({ ok: true, data: receipt });
  await flush();
  assert.equal(game.state.sessionAssists.value, 1);
  assert.equal(game.state.bestStreak.value, 1);
  assert.equal(game.state.matchEarned.value, 5);
  assert.equal(game.state.attempt.value, 1);
  assert.deepEqual(game.events, [['round-started', { roundId: sessionId }], ['saving-change', true], ['credited', { ...receipt, event_id: 42 }], ['saving-change', false]]);
  assert.equal(game.window.localStorage.getItem('cento:playmaker:best:club:42'), '1');
});

test('a server-confirmed miss overrides the visual prediction and awards no coin', async (t) => {
  let game;
  game = fixture({ submitPlaymakerPass: async (eventId, id, payload) => ({ ok: true, data: game.receipt(payload, { assisted: false, clean: false, reason: 'intercepted', wallet: 17, coins_earned: 0, playmaker_earned: 0 }) }) });
  t.after(game.dispose);
  await game.state.startSession();
  game.passTo();
  assert.equal(game.state.activePass.value.assisted, true);
  game.frame(1400);
  await flush();
  assert.equal(game.state.sessionAttempts.value, 1);
  assert.equal(game.state.sessionAssists.value, 0);
  assert.equal(game.state.matchEarned.value, 0);
  assert.equal(game.state.celebration.value, false);
  assert.match(game.state.hint.value, /Intercettato/);
  assert.equal(game.events.find(([name]) => name === 'credited')[1].coins_earned, 0);
});

test('thirty elapsed seconds end the session even after one long RAF pause', async (t) => {
  const game = fixture();
  t.after(game.dispose);
  await game.state.startSession();
  game.frame(500);
  assert.equal(game.state.remainingMs.value, 29500);
  game.state.onPointerDown(game.point(180, 466));
  game.frame(60000);
  assert.equal(game.state.state.value, 'finished');
  assert.equal(game.state.remainingMs.value, 0);
  assert.equal(game.state.secondsLeft.value, 0);
  assert.equal(game.frames.size, 0);
  game.passTo();
  assert.equal(game.calls.submit.length, 0);
  assert.equal(game.state.pendingPass.value, null);
});

test('pointer release checks the real deadline even when no animation frame has run', async (t) => {
  const game = fixture();
  t.after(game.dispose);
  await game.state.startSession();
  game.state.onPointerDown(game.point(180, 466));
  game.state.onPointerMove(game.point(82, 248));
  game.setNow(30001);
  game.state.onPointerUp(game.point(82, 248));
  assert.equal(game.state.state.value, 'finished');
  assert.equal(game.state.pendingPass.value, null);
  assert.equal(game.calls.submit.length, 0);
});

test('lost receipt retries the same action after countdown without double counting UI rewards', async (t) => {
  const server = deferred();
  let request = 0;
  const game = fixture({ submitPlaymakerPass: async () => ++request === 1 ? { ok: false, status: 503 } : server.promise });
  t.after(game.dispose);
  await game.state.startSession();
  game.passTo();
  game.frame(1400);
  await flush();
  assert.equal(game.state.state.value, 'error');
  assert.equal(game.state.sessionAssists.value, 0);
  const pending = { ...game.state.pendingPass.value };
  game.frame(35000);
  assert.equal(game.state.remainingMs.value, 0);
  assert.equal(game.frames.size, 0, 'an offline pending pass does not run a permanent frame loop after time expires');
  assert.deepEqual(game.state.pendingPass.value, pending);
  game.state.retry();
  game.state.retry();
  await flush();
  assert.equal(game.calls.submit.length, 2);
  assert.deepEqual(game.calls.submit[1], game.calls.submit[0]);
  server.resolve({ ok: true, data: game.receipt(game.calls.submit[0][2], { coins_earned: 0, remaining_ms: 0 }) });
  await flush();
  assert.equal(game.state.state.value, 'finished');
  assert.equal(game.state.sessionAssists.value, 1);
  assert.equal(game.state.sessionAttempts.value, 1);
  assert.equal(game.state.pendingPass.value, null);
  assert.equal(game.events.filter(([name]) => name === 'credited').length, 1);
  assert.equal(game.events.find(([name]) => name === 'credited')[1].coins_earned, 0);
  assert.equal(game.frames.size, 0);
});

test('an action released before the deadline can finish and be credited after thirty seconds', async (t) => {
  const game = fixture();
  t.after(game.dispose);
  await game.state.startSession();
  game.frame(29999);
  game.passTo();
  assert.equal(game.state.pendingPass.value.elapsed_ms, 29999);
  game.frame(31399);
  await flush();
  assert.equal(game.calls.submit.length, 1);
  assert.equal(game.state.state.value, 'finished');
  assert.equal(game.state.sessionAssists.value, 1);
  assert.equal(game.state.matchEarned.value, 1);
  assert.equal(game.frames.size, 0);
  assert.equal(game.events.filter(([name]) => name === 'credited').length, 1);
});

test('a stale server time estimate cannot rewind the running countdown', async (t) => {
  let game;
  game = fixture({ submitPlaymakerPass: async (eventId, id, payload) => ({ ok: true, data: game.receipt(payload, { remaining_ms: 29900 }) }) });
  t.after(game.dispose);
  await game.state.startSession();
  game.frame(4000);
  game.passTo();
  game.frame(5400);
  await flush();
  assert.equal(game.state.remainingMs.value, 24600);
  game.timers(5900);
  assert.equal(game.state.remainingMs.value, 24100);
  assert.equal(game.state.state.value, 'ready');
});

test('an invalid receipt keeps the original pending pass and cannot mutate rewards', async (t) => {
  let game;
  game = fixture({ submitPlaymakerPass: async (eventId, id, payload) => ({ ok: true, data: game.receipt(payload, { session_id: 'different-session' }) }) });
  t.after(game.dispose);
  await game.state.startSession();
  game.passTo();
  game.frame(1400);
  await flush();
  assert.equal(game.state.state.value, 'error');
  assert.ok(game.state.pendingPass.value);
  assert.equal(game.state.sessionAssists.value, 0);
  assert.equal(game.state.matchEarned.value, 0);
  assert.equal(game.events.some(([name]) => name === 'credited'), false);
});

test('server expiry finishes the session without a claimable pending pass', async (t) => {
  const game = fixture({ submitPlaymakerPass: async () => ({ ok: false, status: 410 }) });
  t.after(game.dispose);
  await game.state.startSession();
  game.passTo();
  game.frame(1400);
  await flush();
  assert.equal(game.state.state.value, 'finished');
  assert.equal(game.state.pendingPass.value, null);
  assert.equal(game.frames.size, 0);
  assert.equal(game.events.some(([name]) => name === 'credited'), false);
});

test('keyboard selection can be cancelled or launch a pass to the selected teammate', async (t) => {
  const game = fixture();
  t.after(game.dispose);
  const key = (key) => ({ key, preventDefault() {}, stopPropagation() {} });
  await game.state.startSession();
  game.state.onCourtKeydown(key('ArrowLeft'));
  assert.equal(game.state.state.value, 'aiming');
  game.state.onCourtKeydown(key('Escape'));
  assert.equal(game.state.state.value, 'ready');
  assert.equal(game.state.pendingPass.value, null);
  game.state.onCourtKeydown(key('ArrowLeft'));
  game.state.onCourtKeydown(key('Enter'));
  assert.equal(game.state.state.value, 'flying');
  assert.equal(game.state.activePass.value.receiver.id, 'left');
  assert.equal(game.calls.submit.length, 0);
});

test('unmount cancels a flying action and prevents any later API submission', async () => {
  const game = fixture();
  await game.state.startSession();
  game.passTo();
  game.dispose();
  game.frame(30000);
  await flush();
  assert.equal(game.frames.size, 0);
  assert.equal(game.window.timers.size, 0);
  assert.equal(game.calls.submit.length, 0);
  assert.equal(game.events.some(([name]) => name === 'credited'), false);
});

test('a late session response after unmount cannot start a new clock', async () => {
  const server = deferred();
  const game = fixture({ startPlaymakerSession: () => server.promise });
  const starting = game.state.startSession();
  game.dispose();
  server.resolve({ ok: true, data: { session_id: sessionId, remaining_ms: 30000, attempt: 0 } });
  await starting;
  assert.equal(game.state.session.value, null);
  assert.equal(game.frames.size, 0);
  assert.equal(game.events.some(([name]) => name === 'credited'), false);
});

test('a late pass acknowledgement after unmount cannot credit or schedule another action', async () => {
  const server = deferred();
  const game = fixture({ submitPlaymakerPass: () => server.promise });
  await game.state.startSession();
  game.passTo();
  game.frame(1400);
  await flush();
  assert.equal(game.state.state.value, 'saving');
  game.dispose();
  server.resolve({ ok: true, data: game.receipt(game.calls.submit[0][2]) });
  await flush();
  assert.equal(game.state.sessionAssists.value, 0);
  assert.equal(game.state.matchEarned.value, 0);
  assert.equal(game.frames.size, 0);
  assert.equal(game.window.timers.size, 0);
  assert.equal(game.events.some(([name]) => name === 'credited'), false);
  assert.deepEqual(game.events.at(-1), ['saving-change', false]);
});

test('unmount clears the result timer and personal best is scoped to organization and event', async () => {
  const storage = new Map([['cento:playmaker:best:other:42', '90'], ['cento:playmaker:best:club:99', '50']]);
  const game = fixture({}, { storage });
  assert.equal(game.state.personalBest.value, 0);
  await game.state.startSession();
  game.passTo();
  game.frame(1400);
  await flush();
  assert.equal(game.window.timers.size, 1);
  assert.equal(game.state.personalBest.value, 1);
  game.dispose();
  game.timers(1900);
  assert.equal(game.window.timers.size, 0);
  assert.equal(game.frames.size, 0);
  const reopened = fixture({}, { storage });
  assert.equal(reopened.state.personalBest.value, 1);
  assert.equal(reopened.state.sessionAssists.value, 0);
  reopened.dispose();
});
