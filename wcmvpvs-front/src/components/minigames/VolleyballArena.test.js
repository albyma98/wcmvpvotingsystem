import test from 'node:test';
import assert from 'node:assert/strict';
import { setupSfc, testVue, testWindow } from '../../../test/sfcSetup.js';
import * as volleyball from '../../games/volleyball.js';

const roundId = '00000000-0000-0000-0000-000000000042';
const flush = async () => { for (let index = 0; index < 12; index += 1) await Promise.resolve(); };
function deferred() { let resolve; const promise = new Promise((done) => { resolve = done; }); return { promise, resolve }; }

function fixture(gameId = 'volley-rush', overrides = {}, rect = { left: 10, top: 20, width: 360, height: 560 }) {
  const window = testWindow();
  const frames = new Map();
  const delays = new Map();
  const unmount = [];
  const document = { hidden: false, addEventListener() {}, removeEventListener() {} };
  let nextFrame = 1;
  let clock = 0;
  window.performance = { now: () => clock };
  window.requestAnimationFrame = (callback) => { const id = nextFrame++; frames.set(id, callback); return id; };
  window.cancelAnimationFrame = (id) => frames.delete(id);
  const schedule = window.setTimeout;
  window.setTimeout = (callback, milliseconds) => { const id = schedule(callback); delays.set(id, milliseconds); return id; };
  const calls = { prepare: [], submit: [], released: [], vibrate: [] };
  window.navigator = { vibrate: (value) => calls.vibrate.push(value) };
  const target = { x: 90, y: gameId === 'setter-pro' ? 126 : gameId === 'monster-block' ? 275 : 164 };
  const receipt = { round_id: roundId, game_id: gameId, success: true, perfect: true, wallet: 18, coins_earned: 1, game_earned: 4 };
  const setup = setupSfc(new URL('./VolleyballArena.vue', import.meta.url), {
    vue: { ...testVue, onBeforeUnmount: (callback) => unmount.push(callback) },
    '../../api': {
      prepareVolleyballRound: async (...args) => { calls.prepare.push(args); return { ok: true, data: { round_id: roundId, game_id: gameId, target, blocker: { x: 270, y: 285 }, timing_target: .7, min_action_ms: 180 } }; },
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
  function tapTarget() { setup.state.onPointerDown(point(target.x, target.y)); setup.state.onPointerUp(point(target.x, target.y)); }
  function frame(timestamp) {
    clock = timestamp;
    const queued = [...frames.values()]; frames.clear();
    for (const callback of queued) callback(timestamp);
  }
  function timeWell() { frame(clock + volleyball.VOLLEYBALL_MODES[gameId].cycleMs * .7); }
  function readyToPlay() { if (gameId === 'monster-block') timeWell(); else frame(clock + 200); }
  function finishFlight() { frame(clock); frame(clock + volleyball.VOLLEYBALL_MODES[gameId].flightMs); }
  function timers() { const queued = [...window.timers.values()]; window.timers.clear(); for (const callback of queued) callback(); }
  function key(value, repeat = false) { setup.state.onCourtKeydown({ key: value, repeat, preventDefault() {}, stopPropagation() {} }); }
  function dispose() { for (const callback of unmount) callback(); setup.scope.stop(); }
  return { ...setup, window, document, frames, delays, calls, target, receipt, point, frame, aimAtTarget, tapTarget, timeWell, readyToPlay, finishFlight, timers, key, dispose };
}

test('one swipe launches each mode immediately and only the server receipt rewards the play', async (t) => {
  for (const gameId of Object.keys(volleyball.VOLLEYBALL_MODES)) {
    const server = deferred();
    const sent = [];
    const game = fixture(gameId, { submitVolleyballRound: (...args) => { sent.push(args); return server.promise; } });
    t.after(game.dispose);
    await game.state.prepare();
    game.readyToPlay();
    game.aimAtTarget();
    assert.equal(game.state.state.value, 'flying');
    assert.equal(game.state.wins.value, 0);
    assert.equal(game.state.celebration.value, false);
    assert.deepEqual(sent, [[42, gameId, roundId, { aim_x: game.target.x, aim_y: game.target.y, timing: .7 }]]);
    assert.deepEqual(game.events, [['round-started', { roundId }], ['saving-change', true]]);
    game.finishFlight(); await flush();
    assert.equal(game.state.state.value, 'saving');
    assert.equal(game.state.celebration.value, false);
    server.resolve({ ok: true, data: game.receipt }); await flush();
    assert.equal(game.state.state.value, 'result');
    assert.equal(game.state.wins.value, 1);
    assert.equal(game.state.streak.value, 1);
    assert.equal(game.state.matchEarned.value, 4);
    assert.deepEqual(game.events, [
      ['round-started', { roundId }], ['saving-change', true], ['credited', { ...game.receipt, event_id: 42 }], ['round-completed', { roundId }], ['saving-change', false],
    ]);
  }
});

test('a direct target tap fires spike and set with automatic timing at any ready phase', async (t) => {
  for (const gameId of ['volley-rush', 'setter-pro']) {
    const game = fixture(gameId); t.after(game.dispose);
    await game.state.prepare(); game.frame(1234); game.tapTarget();
    assert.equal(game.state.state.value, 'flying');
    assert.deepEqual(game.state.pendingAttempt.value.input, { aim_x: game.target.x, aim_y: game.target.y, timing: .7 });
    game.finishFlight(); await flush();
    assert.equal(game.state.state.value, 'result');
  }
});

test('cancelled and foreign pointers, a held-ball tap, and Escape never consume a round', async (t) => {
  const game = fixture(); t.after(game.dispose);
  const { state, point } = game;
  await state.prepare();
  state.onPointerDown(point(180, 466));
  state.onPointerMove(point(90, 164, 9)); state.cancelAim(point(90, 164, 9));
  assert.equal(state.state.value, 'aiming');
  state.cancelAim(point(90, 164));
  assert.equal(state.state.value, 'ready');
  state.onPointerDown(point(180, 466)); state.onPointerUp(point(180, 466));
  assert.equal(state.state.value, 'ready');
  state.onPointerDown(point(180, 466)); state.onLostCapture(point(180, 466));
  assert.equal(state.state.value, 'ready');
  state.onPointerDown(point(180, 466)); game.key('Escape');
  assert.equal(state.state.value, 'ready');
  assert.equal(game.frames.size, 0);
  assert.equal(game.calls.submit.length, 0);
  assert.equal(game.events.some(([name]) => name === 'round-started'), false);
});

test('mobile and letterboxed courts preserve the target coordinates in a single gesture', async (t) => {
  for (const rect of [{ left: 0, top: 80, width: 320, height: 500 }, { left: 50, top: 100, width: 390, height: 420 }, { left: 0, top: 0, width: 300, height: 650 }]) {
    const game = fixture('setter-pro', {}, rect); t.after(game.dispose);
    await game.state.prepare(); game.frame(200); game.aimAtTarget();
    assert.ok(Math.abs(game.state.pendingAttempt.value.input.aim_x - game.target.x) < 1e-8);
    assert.ok(Math.abs(game.state.pendingAttempt.value.input.aim_y - game.target.y) < 1e-8);
    assert.equal(game.state.state.value, 'flying');
  }
});

test('a fast server cannot reveal rewards or prepare the next round before the flight finishes', async (t) => {
  const game = fixture(); t.after(game.dispose);
  await game.state.prepare(); game.readyToPlay(); game.tapTarget(); await flush();
  assert.equal(game.calls.submit.length, 1);
  assert.equal(game.state.state.value, 'flying');
  assert.equal(game.state.wins.value, 0);
  assert.equal(game.window.timers.size, 0);
  assert.deepEqual(game.events.at(-1), ['saving-change', true]);
  game.finishFlight(); await flush();
  assert.equal(game.state.state.value, 'result');
  assert.equal([...game.delays.values()].at(-1), 220);
  game.timers(); await flush();
  assert.equal(game.state.state.value, 'ready');
  assert.equal(game.state.lastReceipt.value, null);
  assert.equal(game.state.celebration.value, false);
  assert.equal(game.calls.prepare.length, 2);
  game.timers(); await flush();
  assert.equal(game.calls.submit.length, 1);
  assert.equal(game.calls.prepare.length, 2);
});

test('failed submissions retain one immutable attempt and retry without preparing another round', async (t) => {
  const sent = []; const server = deferred();
  const game = fixture('volley-rush', { submitVolleyballRound: (...args) => { sent.push(args); return sent.length === 1 ? { ok: false, status: 503 } : server.promise; } });
  t.after(game.dispose);
  await game.state.prepare(); game.readyToPlay(); game.aimAtTarget(); game.finishFlight(); await flush();
  assert.equal(game.state.state.value, 'error');
  assert.equal(game.state.pendingAttempt.value.roundId, roundId);
  game.state.moveAim({ x: 310, y: 250 });
  await game.state.prepare();
  assert.equal(game.calls.prepare.length, 1);
  game.state.retry(); game.state.retry(); await flush();
  assert.equal(sent.length, 2);
  assert.deepEqual(sent[1], sent[0]);
  server.resolve({ ok: true, data: { ...game.receipt, coins_earned: 0 } }); await flush();
  assert.equal(game.state.pendingAttempt.value, null);
  assert.equal(game.events.filter(([name]) => name === 'credited').length, 1);
  assert.equal(game.events.find(([name]) => name === 'credited')[1].coins_earned, 0);
  assert.equal(game.state.lastReceipt.value.coins_earned, 0);
});

test('server rejection overrides a perfect aim and refreshes quickly after miss feedback', async (t) => {
  const game = fixture('setter-pro', { submitVolleyballRound: async () => ({ ok: true, data: { round_id: roundId, game_id: 'setter-pro', success: false, perfect: false, coins_earned: 0, game_earned: 0, wallet: 17 } }) });
  t.after(game.dispose);
  await game.state.prepare(); game.state.streak.value = 3;
  game.readyToPlay(); game.aimAtTarget(); game.finishFlight(); await flush();
  assert.equal(game.state.celebration.value, false);
  assert.equal(game.state.wins.value, 0);
  assert.equal(game.state.streak.value, 0);
  assert.ok(game.state.hint.value);
  assert.equal([...game.delays.values()].at(-1), 140);
  game.timers(); await flush();
  assert.equal(game.state.state.value, 'ready');
  assert.equal(game.calls.prepare.length, 2);
});

test('expired challenges refresh before shooting and terminal receipts never complete or credit', async (t) => {
  const game = fixture('volley-rush', { submitVolleyballRound: async () => ({ ok: false, status: 410 }) }); t.after(game.dispose);
  await game.state.prepare();
  game.state.challenge.value.expires_at = new Date(Date.now() - 1000).toISOString();
  game.tapTarget(); await flush();
  assert.equal(game.calls.prepare.length, 2);
  assert.equal(game.events.some(([name]) => name === 'round-started'), false);
  game.readyToPlay(); game.aimAtTarget(); game.finishFlight(); await flush();
  assert.equal(game.state.pendingAttempt.value, null);
  assert.equal(game.state.challenge.value, null);
  assert.equal(game.events.some(([name]) => name === 'round-completed' || name === 'credited'), false);
});

test('incomplete or mismatched receipts retain the original attempt and never credit', async (t) => {
  for (const invalid of [{ game_id: 'setter-pro' }, { wallet: undefined }, { game_earned: -1 }, { success: 'true' }, { coins_earned: 2 }, { perfect: null }]) {
    const game = fixture('volley-rush', { submitVolleyballRound: async () => ({ ok: true, data: { round_id: roundId, game_id: 'volley-rush', success: true, perfect: true, wallet: 18, game_earned: 1, coins_earned: 1, ...invalid } }) });
    t.after(game.dispose);
    await game.state.prepare(); game.readyToPlay(); game.aimAtTarget(); game.finishFlight(); await flush();
    assert.equal(game.state.state.value, 'error');
    assert.equal(game.state.pendingAttempt.value.roundId, roundId);
    assert.equal(game.state.wins.value, 0);
    assert.equal(game.events.some(([name]) => name === 'credited' || name === 'round-completed'), false);
  }
});

test('keyboard arrows aim and one Enter fires, while held execution keys cannot start the next round', async (t) => {
  for (const gameId of Object.keys(volleyball.VOLLEYBALL_MODES)) {
    const game = fixture(gameId); t.after(game.dispose);
    await game.state.prepare(); game.readyToPlay();
    game.key('ArrowLeft'); assert.equal(game.state.aim.value.x, 172);
    if (gameId === 'monster-block') { game.key('ArrowUp'); assert.equal(game.state.aim.value.y, game.target.y); }
    game.key('Enter');
    assert.equal(game.state.state.value, 'flying');
    game.finishFlight(); await flush(); game.timers(); await flush();
    game.key('Enter', true); game.key(' ', true);
    assert.equal(game.state.state.value, 'ready');
    assert.equal(game.calls.submit.length, 1);
    let prevented = 0;
    game.state.onActionKeydown({ key: 'Enter', repeat: true, preventDefault: () => prevented++ });
    assert.equal(prevented, 1);
  }
});

test('Monster Block animates automatically and a lane tap or horizontal swipe closes the block', async (t) => {
  for (const gesture of ['tap', 'swipe']) {
    const game = fixture('monster-block'); t.after(game.dispose);
    await game.state.prepare();
    assert.equal(game.state.state.value, 'ready');
    assert.equal(game.frames.size, 1);
    assert.equal(game.state.ballPose.value.y, 110);
    game.timeWell();
    assert.equal(game.state.state.value, 'ready');
    assert.equal(game.state.ballPose.value.y, game.target.y);
    assert.equal(game.state.timingGood.value, true);
    game.state.onPointerDown(game.point(gesture === 'tap' ? game.target.x : 180, 440));
    if (gesture === 'swipe') game.state.onPointerMove(game.point(130, 440));
    game.state.onPointerUp(game.point(game.target.x, 440));
    assert.equal(game.state.state.value, 'flying');
    assert.deepEqual(game.state.pendingAttempt.value.input, { aim_x: game.target.x, aim_y: game.target.y, timing: .7 });
  }
});

test('backgrounding cancels a drag, restarts incoming timing on return, and safely settles flight', async (t) => {
  const game = fixture('monster-block'); t.after(game.dispose);
  await game.state.prepare(); game.frame(300);
  game.state.onPointerDown(game.point(180, 440));
  game.document.hidden = true; game.state.onVisibilityChange();
  assert.equal(game.state.state.value, 'ready'); assert.equal(game.frames.size, 0);
  game.frame(5000); game.document.hidden = false; game.state.onVisibilityChange();
  assert.equal(game.state.timing.value, 0);
  assert.equal(game.frames.size, 1);
  game.timeWell(); game.aimAtTarget();
  game.document.hidden = true; game.state.onVisibilityChange(); await flush();
  assert.equal(game.calls.submit.length, 1);
  assert.equal(game.state.state.value, 'result');
});

test('an immediate reduced-motion shot uses the advertised minimum, with a safe older-server fallback', async (t) => {
  for (const [advertised, minimum] of [[180, 180], [undefined, 600], [0, 600]]) {
    const game = fixture(); t.after(game.dispose);
    game.window.matchMedia = () => ({ matches: true });
    await game.state.prepare(); game.state.challenge.value.min_action_ms = advertised;
    game.state.primaryAction();
    assert.equal(game.state.state.value, 'flying');
    assert.equal([...game.delays.values()].at(-1), minimum);
    game.frame(0); game.frame(150); await flush();
    assert.equal(game.state.state.value, 'saving');
    assert.equal(game.calls.submit.length, 0);
    game.frame(minimum); game.timers(); await flush();
    assert.equal(game.calls.submit.length, 1);
    assert.equal(game.state.state.value, 'result');
    assert.equal(game.calls.vibrate.length, 0);
  }
});

test('streak feedback escalates cosmetically without multiplying authoritative coins', async (t) => {
  const game = fixture(); t.after(game.dispose);
  await game.state.prepare(); game.state.streak.value = 4;
  game.readyToPlay(); game.tapTarget(); game.finishFlight(); await flush();
  assert.equal(game.state.streakCallout.value, 'ON FIRE ×5');
  assert.deepEqual(game.calls.vibrate, [[12, 25, 12]]);
  assert.equal(game.state.lastReceipt.value.coins_earned, 1);
  assert.equal(game.events.find(([name]) => name === 'credited')[1].coins_earned, 1);
  game.state.streak.value = 10;
  assert.equal(game.state.streakCallout.value, 'INARRESTABILE ×10');
});

test('disposal during the minimum age wait cancels the deferred submission', async () => {
  const game = fixture();
  await game.state.prepare(); game.state.primaryAction();
  assert.equal(game.window.timers.size, 1);
  game.dispose(); await flush();
  assert.equal(game.window.timers.size, 0);
  assert.equal(game.frames.size, 0);
  assert.equal(game.calls.submit.length, 0);
});

test('unmount during flight after a parallel response never credits or prepares another round', async () => {
  const game = fixture();
  await game.state.prepare(); game.readyToPlay(); game.tapTarget(); await flush();
  assert.equal(game.calls.submit.length, 1);
  assert.equal(game.state.state.value, 'flying');
  game.dispose(); await flush();
  assert.equal(game.frames.size, 0);
  assert.equal(game.calls.prepare.length, 1);
  assert.equal(game.events.some(([name]) => name === 'credited'), false);
});
