import test from 'node:test';
import assert from 'node:assert/strict';
import { setupSfc, testVue, testWindow } from '../../../test/sfcSetup.js';
import * as basketball from '../../games/basketballShot.js';

const shotId = '00000000-0000-0000-0000-000000000042';
const receipt = { shot_id: shotId, made: true, clean: true, coins_earned: 1, wallet: 18, basketball_earned: 4 };
const flush = async () => { for (let index = 0; index < 8; index += 1) await Promise.resolve(); };
function deferred() {
  let resolve;
  const promise = new Promise((done) => { resolve = done; });
  return { promise, resolve };
}

function fixture(overrides = {}, rect = { left: 10, top: 20, width: 360, height: 560 }, props = {}) {
  const window = testWindow();
  const frames = new Map();
  let nextFrame = 1;
  window.requestAnimationFrame = (callback) => { const id = nextFrame++; frames.set(id, callback); return id; };
  window.cancelAnimationFrame = (id) => frames.delete(id);
  const calls = { prepare: [], submit: [], captured: [], released: [], focus: 0 };
  const unmount = [];
  const setup = setupSfc(new URL('./BasketballShootout.vue', import.meta.url), {
    vue: { ...testVue, onBeforeUnmount: (callback) => unmount.push(callback) },
    '../../api': {
      prepareBasketballShot: async (...args) => {
        calls.prepare.push(args);
        return { ok: true, data: { shot_id: shotId, hoop: { ...basketball.BASKETBALL_HOOP } } };
      },
      submitBasketballShot: async (...args) => {
        calls.submit.push(args);
        return { ok: true, data: { ...receipt } };
      },
      ...overrides,
    },
    '../../games/basketballShot': basketball,
  }, { eventId: 42, walletCoins: 17, earnedCoins: 0, clubName: 'Club', ...props }, { window });
  setup.state.courtEl.value = {
    getBoundingClientRect: () => rect,
    focus: () => { calls.focus += 1; },
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
  function dragTo(x, y) {
    setup.state.onPointerDown(point(180, 466));
    setup.state.onPointerMove(point(x, y));
    setup.state.onPointerUp(point(x, y));
  }
  function frame(timestamp) {
    const queued = [...frames.values()];
    frames.clear();
    for (const callback of queued) callback(timestamp);
  }
  function timers() {
    const queued = [...window.timers.values()];
    window.timers.clear();
    for (const callback of queued) callback();
  }
  function dispose() { for (const callback of unmount) callback(); setup.scope.stop(); }
  return { ...setup, window, calls, frames, point, dragTo, frame, timers, dispose };
}

test('cancelled pointers, taps and foreign pointers never submit or consume a shot', async (t) => {
  const game = fixture();
  t.after(game.dispose);
  const { state, point, calls } = game;
  await state.prepare();
  state.onPointerDown(point(180, 466));
  state.onPointerMove(point(180, 200, 9));
  assert.deepEqual(state.drag.value, { x: 0, y: 0 });
  state.cancelAim(point(180, 200, 9));
  assert.equal(state.state.value, 'aiming');
  state.cancelAim(point(180, 200));
  assert.equal(state.state.value, 'ready');
  assert.deepEqual(calls.released, [7]);
  state.onPointerDown(point(180, 466));
  state.onPointerUp(point(180, 466));
  state.onPointerDown(point(180, 466));
  state.onLostCapture(point(180, 200));
  await flush();
  assert.equal(state.state.value, 'ready');
  assert.equal(state.challenge.value.shot_id, shotId);
  assert.equal(calls.prepare.length, 1);
  assert.equal(calls.submit.length, 0);
  assert.equal(game.frames.size, 0);
  assert.equal(game.events.some(([name]) => name === 'credited'), false);
});

test('an expired keyboard challenge refreshes the ball before starting a flight or submitting', async (t) => {
  const game = fixture();
  t.after(game.dispose);
  await game.state.prepare();
  game.state.challenge.value.expires_at = new Date(Date.now() - 1000).toISOString();
  game.state.onCourtKeydown({ key: 'ArrowUp', preventDefault() {} });
  game.state.onCourtKeydown({ key: 'Enter', preventDefault() {} });
  await flush();
  assert.equal(game.calls.prepare.length, 2);
  assert.equal(game.calls.submit.length, 0);
  assert.equal(game.frames.size, 0);
  assert.equal(game.state.state.value, 'ready');
});

test('a valid drag animates first and credits exactly the server acknowledgement', async (t) => {
  const server = deferred();
  const calls = [];
  const game = fixture({ submitBasketballShot: (...args) => { calls.push(args); return server.promise; } });
  t.after(game.dispose);
  await game.state.prepare();
  game.dragTo(180, 200);
  assert.equal(game.state.state.value, 'flying');
  assert.equal(calls.length, 0);
  assert.equal(game.state.sessionMakes.value, 0);
  game.frame(10);
  game.frame(1110);
  await flush();
  assert.equal(game.state.state.value, 'saving');
  assert.deepEqual(calls, [[42, shotId, { dx: 0, dy: -266 }]]);
  assert.equal(game.state.showCelebration.value, false);
  assert.equal(game.events.some(([name]) => name === 'credited'), false);
  server.resolve({ ok: true, data: { ...receipt } });
  await flush();
  assert.equal(game.state.state.value, 'result');
  assert.equal(game.state.sessionMakes.value, 1);
  assert.equal(game.state.streak.value, 1);
  assert.equal(game.state.matchEarned.value, 4);
  assert.equal(game.state.showCelebration.value, true);
  assert.deepEqual(game.events, [
    ['saving-change', true],
    ['credited', { ...receipt, event_id: 42 }],
    ['saving-change', false],
  ]);
});

test('mobile and letterboxed SVG coordinates preserve the actual drag to the hoop', async (t) => {
  for (const rect of [
    { left: 0, top: 80, width: 320, height: 500 },
    { left: 50, top: 100, width: 390, height: 420 },
    { left: 0, top: 0, width: 300, height: 650 },
  ]) {
    const game = fixture({}, rect);
    t.after(game.dispose);
    await game.state.prepare();
    game.dragTo(180, 200);
    assert.equal(game.state.activeShot.value.made, true);
    assert.ok(Math.abs(game.state.pendingThrow.value.dx) < .000001);
    assert.ok(Math.abs(game.state.pendingThrow.value.dy + 266) < .000001);
    assert.equal(game.calls.submit.length, 0);
  }
});

test('network failure keeps the same shot and gesture for an idempotent retry', async (t) => {
  const calls = [];
  const second = deferred();
  const game = fixture({ submitBasketballShot: async (...args) => {
    calls.push(args);
    return calls.length === 1 ? { ok: false, status: 503 } : second.promise;
  } });
  t.after(game.dispose);
  await game.state.prepare();
  game.dragTo(180, 200);
  game.frame(0);
  game.frame(1100);
  await flush();
  assert.equal(game.state.state.value, 'error');
  assert.ok(game.state.errorMessage.value);
  assert.equal(game.state.sessionMakes.value, 0);
  assert.equal(game.state.pendingThrow.value.shotId, shotId);
  game.state.retry();
  game.state.retry();
  await flush();
  assert.equal(calls.length, 2, 'parallel retry clicks do not duplicate the request');
  assert.deepEqual(calls[1], calls[0]);
  assert.equal(game.calls.prepare.length, 1);
  second.resolve({ ok: true, data: { ...receipt, coins_earned: 0 } });
  await flush();
  assert.equal(game.state.sessionMakes.value, 1);
  assert.equal(game.state.pendingThrow.value, null);
  assert.equal(game.events.filter(([name]) => name === 'credited').length, 1);
  assert.equal(game.events.find(([name]) => name === 'credited')[1].coins_earned, 0);
});

test('a missed basket gives no positive feedback and prepares a new target after the result', async (t) => {
  const game = fixture({ submitBasketballShot: async () => ({ ok: true, data: { ...receipt, made: false, clean: false, coins_earned: 0, basketball_earned: 3 } }) });
  t.after(game.dispose);
  await game.state.prepare();
  game.state.streak.value = 2;
  game.dragTo(270, 200);
  game.frame(0);
  game.frame(1100);
  await flush();
  assert.equal(game.state.sessionAttempts.value, 1);
  assert.equal(game.state.sessionMakes.value, 0);
  assert.equal(game.state.streak.value, 0);
  assert.equal(game.state.showCelebration.value, false);
  assert.ok(game.state.hint.value);
  game.timers();
  await flush();
  assert.equal(game.state.state.value, 'ready');
  assert.equal(game.calls.prepare.length, 2);
});

test('expired shots cannot be retried for credit and request a fresh ball instead', async (t) => {
  const game = fixture({ submitBasketballShot: async () => ({ ok: false, status: 410 }) });
  t.after(game.dispose);
  await game.state.prepare();
  game.dragTo(180, 200);
  game.frame(0);
  game.frame(1100);
  await flush();
  assert.equal(game.state.pendingThrow.value, null);
  assert.equal(game.state.challenge.value, null);
  assert.equal(game.events.some(([name]) => name === 'credited'), false);
  game.state.retry();
  await flush();
  assert.equal(game.state.state.value, 'ready');
  assert.equal(game.calls.prepare.length, 2);
});

test('unmount cancels an active flight and prevents a request after animation disposal', async () => {
  const game = fixture();
  await game.state.prepare();
  game.dragTo(180, 200);
  assert.equal(game.frames.size, 1);
  game.dispose();
  await flush();
  assert.equal(game.frames.size, 0);
  assert.equal(game.calls.submit.length, 0);
  assert.equal(game.window.timers.size, 0);
  assert.equal(game.events.some(([name]) => name === 'credited'), false);
});

test('late server acknowledgements after unmount do not update the old game or schedule timers', async () => {
  const server = deferred();
  const game = fixture({ submitBasketballShot: () => server.promise });
  await game.state.prepare();
  game.dragTo(180, 200);
  game.frame(0);
  game.frame(1100);
  await flush();
  assert.equal(game.state.state.value, 'saving');
  game.dispose();
  server.resolve({ ok: true, data: { ...receipt } });
  await flush();
  assert.equal(game.state.sessionMakes.value, 0);
  assert.equal(game.state.matchEarned.value, 0);
  assert.equal(game.window.timers.size, 0);
  assert.equal(game.events.some(([name]) => name === 'credited'), false);
  assert.deepEqual(game.events.at(-1), ['saving-change', false]);
});

test('unmount clears the next-ball timer after a confirmed basket', async () => {
  const game = fixture();
  await game.state.prepare();
  game.dragTo(180, 200);
  game.frame(0);
  game.frame(1100);
  await flush();
  assert.equal(game.window.timers.size, 1);
  game.dispose();
  assert.equal(game.window.timers.size, 0);
  game.timers();
  await flush();
  assert.equal(game.calls.prepare.length, 1);
});

test('reopening the game restores previously saved match earnings before a new shot', async (t) => {
  const game = fixture({}, undefined, { earnedCoins: 7 });
  t.after(game.dispose);
  assert.equal(game.state.matchEarned.value, 7);
  await game.state.prepare();
  assert.equal(game.state.matchEarned.value, 7);
  assert.equal(game.state.sessionMakes.value, 0, 'the session counter remains distinct from match earnings');
  assert.equal(game.calls.submit.length, 0);
  assert.equal(game.events.some(([name]) => name === 'credited'), false);
});
