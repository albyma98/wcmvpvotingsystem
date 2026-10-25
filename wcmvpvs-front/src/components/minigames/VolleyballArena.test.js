import test from 'node:test';
import assert from 'node:assert/strict';
import { setupSfc, testVue, testWindow } from '../../../test/sfcSetup.js';
import * as volleyball from '../../games/volleyball.js';

const roundId = '00000000-0000-0000-0000-000000000042';
const flush = async () => { for (let index = 0; index < 10; index += 1) await Promise.resolve(); };
function deferred() { let resolve; const promise = new Promise((done) => { resolve = done; }); return { promise, resolve }; }

function fixture(gameId = 'volley-rush', overrides = {}, rect = { left: 10, top: 20, width: 360, height: 560 }) {
  const window = testWindow();
  const frames = new Map();
  const unmount = [];
  const document = { hidden: false, addEventListener() {}, removeEventListener() {} };
  let nextFrame = 1;
  let clock = 0;
  window.performance = { now: () => clock };
  window.requestAnimationFrame = (callback) => { const id = nextFrame++; frames.set(id, callback); return id; };
  window.cancelAnimationFrame = (id) => frames.delete(id);
  const calls = { prepare: [], submit: [], released: [] };
  const target = { x: 90, y: gameId === 'setter-pro' ? 126 : gameId === 'monster-block' ? 275 : 164 };
  const receipt = { round_id: roundId, game_id: gameId, success: true, perfect: true, wallet: 18, coins_earned: 1, game_earned: 4 };
  const setup = setupSfc(new URL('./VolleyballArena.vue', import.meta.url), {
    vue: { ...testVue, onBeforeUnmount: (callback) => unmount.push(callback) },
    '../../api': {
      prepareVolleyballRound: async (...args) => { calls.prepare.push(args); return { ok: true, data: { round_id: roundId, game_id: gameId, target, blocker: { x: 270, y: 285 }, timing_target: .7 } }; },
      submitVolleyballRound: async (...args) => { calls.submit.push(args); return { ok: true, data: { ...receipt } }; },
      ...overrides,
    },
    '../../games/volleyball': volleyball,
  }, { eventId: 42, gameId, walletCoins: 17, earnedCoins: 0, clubName: 'Club' }, { window, document });
  setup.state.courtEl.value = {
    getBoundingClientRect: () => rect, focus() {}, setPointerCapture() {}, releasePointerCapture: (id) => calls.released.push(id),
  };
  function point(x, y, pointerId = 7) {
    const scale = Math.min(rect.width / 360, rect.height / 560);
    return { pointerId, isPrimary: true, button: 0, preventDefault() {}, clientX: rect.left + (rect.width - 360 * scale) / 2 + x * scale, clientY: rect.top + (rect.height - 560 * scale) / 2 + y * scale };
  }
  function aimAtTarget() {
    setup.state.onPointerDown(point(180, 466));
    setup.state.onPointerMove(point(target.x, target.y));
    setup.state.onPointerUp(point(target.x, target.y));
  }
  function frame(timestamp) {
    clock = timestamp;
    const queued = [...frames.values()]; frames.clear();
    for (const callback of queued) callback(timestamp);
  }
  function timeWell() { frame(clock + volleyball.VOLLEYBALL_MODES[gameId].cycleMs * .7); }
  function finishFlight() { frame(clock); frame(clock + volleyball.VOLLEYBALL_MODES[gameId].flightMs); }
  function timers() { const queued = [...window.timers.values()]; window.timers.clear(); for (const callback of queued) callback(); }
  function dispose() { for (const callback of unmount) callback(); setup.scope.stop(); }
  return { ...setup, window, document, frames, calls, target, receipt, point, frame, aimAtTarget, timeWell, finishFlight, timers, dispose };
}

test('all three modes send the actual aim and timing, and only reward a server receipt', async (t) => {
  for (const gameId of Object.keys(volleyball.VOLLEYBALL_MODES)) {
    const server = deferred();
    const sent = [];
    const game = fixture(gameId, { submitVolleyballRound: (...args) => { sent.push(args); return server.promise; } });
    t.after(game.dispose);
    await game.state.prepare();
    game.aimAtTarget();
    assert.equal(game.state.state.value, 'timing');
    game.timeWell();
    game.state.primaryAction();
    assert.equal(game.state.state.value, 'flying');
    assert.equal(game.state.wins.value, 0);
    assert.equal(sent.length, 0);
    game.finishFlight();
    await flush();
    assert.equal(game.state.state.value, 'saving');
    assert.equal(game.state.celebration.value, false);
    assert.deepEqual(sent, [[42, gameId, roundId, { aim_x: game.target.x, aim_y: game.target.y, timing: .7 }]]);
    server.resolve({ ok: true, data: game.receipt });
    await flush();
    assert.equal(game.state.state.value, 'result');
    assert.equal(game.state.wins.value, 1);
    assert.equal(game.state.streak.value, 1);
    assert.equal(game.state.matchEarned.value, 4);
    assert.deepEqual(game.events, [
      ['round-started', { roundId }], ['saving-change', true], ['credited', { ...game.receipt, event_id: 42 }], ['round-completed', { roundId }], ['saving-change', false],
    ]);
  }
});

test('cancelled and foreign pointers, a tap on the held ball, and Escape never consume a round', async (t) => {
  const game = fixture(); t.after(game.dispose);
  const { state, point } = game;
  await state.prepare();
  state.onPointerDown(point(180, 466));
  state.onPointerMove(point(90, 164, 9));
  state.cancelAim(point(90, 164, 9));
  assert.equal(state.state.value, 'aiming');
  state.cancelAim(point(90, 164));
  assert.equal(state.state.value, 'ready');
  state.onPointerDown(point(180, 466)); state.onPointerUp(point(180, 466));
  assert.equal(state.state.value, 'ready');
  state.onPointerDown(point(180, 466)); state.onLostCapture(point(180, 466));
  assert.equal(state.state.value, 'ready');
  game.aimAtTarget();
  state.onCourtKeydown({ key: 'Escape', preventDefault() {}, stopPropagation() {} });
  assert.equal(state.state.value, 'ready');
  assert.equal(game.frames.size, 0);
  assert.equal(game.calls.submit.length, 0);
  assert.equal(game.events.some(([name]) => name === 'round-started'), false);
});

test('mobile and letterboxed courts preserve exact target coordinates', async (t) => {
  for (const rect of [{ left: 0, top: 80, width: 320, height: 500 }, { left: 50, top: 100, width: 390, height: 420 }, { left: 0, top: 0, width: 300, height: 650 }]) {
    const game = fixture('setter-pro', {}, rect); t.after(game.dispose);
    await game.state.prepare(); game.aimAtTarget();
    assert.ok(Math.abs(game.state.aim.value.x - game.target.x) < 1e-8);
    assert.ok(Math.abs(game.state.aim.value.y - game.target.y) < 1e-8);
    assert.equal(game.state.state.value, 'timing');
  }
});

test('failed submissions retain one immutable attempt and retry it without preparing another round', async (t) => {
  const sent = []; const server = deferred();
  const game = fixture('volley-rush', { submitVolleyballRound: (...args) => { sent.push(args); return sent.length === 1 ? { ok: false, status: 503 } : server.promise; } });
  t.after(game.dispose);
  await game.state.prepare(); game.aimAtTarget(); game.timeWell(); game.state.primaryAction(); game.finishFlight(); await flush();
  assert.equal(game.state.state.value, 'error');
  assert.equal(game.state.pendingAttempt.value.roundId, roundId);
  await game.state.prepare();
  assert.equal(game.calls.prepare.length, 1);
  game.state.retry(); game.state.retry(); await flush();
  assert.equal(sent.length, 2);
  assert.deepEqual(sent[1], sent[0]);
  server.resolve({ ok: true, data: { ...game.receipt, coins_earned: 0 } }); await flush();
  assert.equal(game.state.pendingAttempt.value, null);
  assert.equal(game.events.filter(([name]) => name === 'credited').length, 1);
  assert.equal(game.events.find(([name]) => name === 'credited')[1].coins_earned, 0);
});

test('a server rejection overrides a visually perfect play and refreshes the court after feedback', async (t) => {
  const game = fixture('setter-pro', { submitVolleyballRound: async () => ({ ok: true, data: { round_id: roundId, game_id: 'setter-pro', success: false, perfect: false, coins_earned: 0, game_earned: 0, wallet: 17 } }) });
  t.after(game.dispose);
  await game.state.prepare(); game.state.streak.value = 3;
  game.aimAtTarget(); game.timeWell(); game.state.primaryAction(); game.finishFlight(); await flush();
  assert.equal(game.state.celebration.value, false);
  assert.equal(game.state.wins.value, 0);
  assert.equal(game.state.streak.value, 0);
  assert.ok(game.state.hint.value);
  game.timers(); await flush();
  assert.equal(game.state.state.value, 'ready');
  assert.equal(game.calls.prepare.length, 2);
});

test('expired challenges refresh before a shot and terminal expired receipts never report a completed round', async (t) => {
  const game = fixture('volley-rush', { submitVolleyballRound: async () => ({ ok: false, status: 410 }) }); t.after(game.dispose);
  await game.state.prepare();
  game.aimAtTarget(); game.state.challenge.value.expires_at = new Date(Date.now() - 1000).toISOString();
  game.state.primaryAction(); await flush();
  assert.equal(game.calls.prepare.length, 2);
  assert.equal(game.events.some(([name]) => name === 'round-started'), false);
  game.aimAtTarget(); game.timeWell(); game.state.primaryAction(); game.finishFlight(); await flush();
  assert.equal(game.state.pendingAttempt.value, null);
  assert.equal(game.state.challenge.value, null);
  assert.equal(game.events.some(([name]) => name === 'round-completed'), false);
  assert.equal(game.events.some(([name]) => name === 'credited'), false);
});

test('incomplete or mismatched receipts retain the original attempt for recovery and never credit', async (t) => {
  for (const invalid of [{ game_id: 'setter-pro' }, { wallet: undefined }, { game_earned: -1 }, { success: 'true' }, { coins_earned: 2 }, { perfect: null }]) {
    const game = fixture('volley-rush', { submitVolleyballRound: async () => ({ ok: true, data: { round_id: roundId, game_id: 'volley-rush', success: true, perfect: true, wallet: 18, game_earned: 1, coins_earned: 1, ...invalid } }) });
    t.after(game.dispose);
    await game.state.prepare(); game.aimAtTarget(); game.timeWell(); game.state.primaryAction(); game.finishFlight(); await flush();
    assert.equal(game.state.state.value, 'error');
    assert.equal(game.state.pendingAttempt.value.roundId, roundId);
    assert.equal(game.state.wins.value, 0);
    assert.equal(game.events.some(([name]) => name === 'credited' || name === 'round-completed'), false);
  }
});

test('keyboard controls can aim, prepare and execute a block without touch', async (t) => {
  const game = fixture('monster-block'); t.after(game.dispose);
  await game.state.prepare();
  const key = (key) => game.state.onCourtKeydown({ key, preventDefault() {} });
  key('ArrowLeft');
  assert.equal(game.state.aim.value.x, 172);
  key('ArrowUp'); assert.equal(game.state.aim.value.y, game.target.y);
  key('Enter'); assert.equal(game.state.state.value, 'timing');
  game.timeWell(); key(' '); assert.equal(game.state.state.value, 'flying');
});

test('a timing tap anywhere preserves aligned blocking hands and a deliberate drag adjusts them relatively', async (t) => {
  const tap = fixture('monster-block'); t.after(tap.dispose);
  await tap.state.prepare(); tap.aimAtTarget(); tap.timeWell();
  tap.state.onPointerDown(tap.point(280, 440));
  assert.equal(tap.state.aim.value.x, tap.target.x);
  tap.state.onPointerUp(tap.point(281, 440));
  assert.equal(tap.state.pendingAttempt.value.input.aim_x, tap.target.x);
  assert.equal(tap.state.state.value, 'flying');

  const drag = fixture('monster-block'); t.after(drag.dispose);
  await drag.state.prepare(); drag.aimAtTarget(); drag.timeWell();
  drag.state.onPointerDown(drag.point(180, 440));
  drag.state.onPointerMove(drag.point(200, 440));
  assert.equal(drag.state.aim.value.x, drag.target.x + 20);
  drag.state.onPointerUp(drag.point(210, 440));
  assert.equal(drag.state.pendingAttempt.value.input.aim_x, drag.target.x + 30);
});

test('holding an execution key cannot prepare and automatically fire a second key repeat', async (t) => {
  const game = fixture(); t.after(game.dispose);
  await game.state.prepare();
  game.state.onCourtKeydown({ key: 'Enter', preventDefault() {} });
  assert.equal(game.state.state.value, 'timing');
  game.state.onCourtKeydown({ key: 'Enter', repeat: true, preventDefault() {} });
  assert.equal(game.state.state.value, 'timing');
  assert.equal(game.events.some(([name]) => name === 'round-started'), false);
});

test('backgrounding cancels preparation timing and completes an in-flight receipt safely', async (t) => {
  const game = fixture('monster-block'); t.after(game.dispose);
  await game.state.prepare(); game.aimAtTarget();
  game.document.hidden = true; game.state.onVisibilityChange();
  assert.equal(game.state.state.value, 'ready'); assert.equal(game.frames.size, 0);
  game.document.hidden = false; game.aimAtTarget(); game.timeWell(); game.state.primaryAction();
  game.document.hidden = true; game.state.onVisibilityChange(); await flush();
  assert.equal(game.calls.submit.length, 1);
  assert.equal(game.state.state.value, 'result');
});

test('reduced motion waits for the server minimum age before submitting even an immediate early miss', async (t) => {
  const game = fixture(); t.after(game.dispose);
  game.window.matchMedia = () => ({ matches: true });
  await game.state.prepare();
  game.state.primaryAction(); game.state.primaryAction();
  game.frame(0); game.frame(180); await flush();
  assert.equal(game.state.state.value, 'saving');
  assert.equal(game.calls.submit.length, 0);
  game.frame(600); game.timers(); await flush();
  assert.equal(game.calls.submit.length, 1);
});

test('disposal during the minimum age wait cancels the deferred submission', async () => {
  const game = fixture();
  game.window.matchMedia = () => ({ matches: true });
  await game.state.prepare();
  game.state.primaryAction(); game.state.primaryAction();
  game.frame(0); game.frame(180); await flush();
  assert.equal(game.window.timers.size, 1);
  game.dispose(); await flush();
  assert.equal(game.window.timers.size, 0);
  assert.equal(game.calls.submit.length, 0);
});

test('unmount cancels animation and never submits or credits after disposal', async () => {
  const game = fixture();
  await game.state.prepare(); game.aimAtTarget(); game.timeWell(); game.state.primaryAction();
  game.dispose(); await flush();
  assert.equal(game.frames.size, 0);
  assert.equal(game.calls.submit.length, 0);
  assert.equal(game.events.some(([name]) => name === 'credited'), false);
});
