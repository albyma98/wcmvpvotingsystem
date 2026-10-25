import test from 'node:test';
import assert from 'node:assert/strict';
import { VOLLEYBALL_MODES, volleyballTiming, previewVolleyballAttempt, sampleVolleyballFlight } from './volleyball.js';

test('timing moves continuously in both directions without jumping at the endpoints', () => {
  assert.equal(volleyballTiming(0, 1000), 0);
  assert.equal(volleyballTiming(700, 1000), .7);
  assert.equal(volleyballTiming(1000, 1000), 1);
  assert.ok(Math.abs(volleyballTiming(1300, 1000) - .7) < 1e-10);
  assert.equal(volleyballTiming(2000, 1000), 0);
});

test('visual forecasts use the sport-specific target ellipse and timing window', () => {
  for (const [id, mode] of Object.entries(VOLLEYBALL_MODES)) {
    const challenge = { target: { x: 90, y: 150 }, timing_target: .7, blocker: { x: 290 } };
    const attempt = { aim_x: 90 + mode.rx, aim_y: 150, timing: .7 + mode.timingWindow };
    assert.deepEqual(previewVolleyballAttempt(id, challenge, attempt).onTarget, true);
    assert.equal(previewVolleyballAttempt(id, challenge, attempt).inTime, true);
    assert.equal(previewVolleyballAttempt(id, challenge, { ...attempt, aim_x: attempt.aim_x + .1 }).onTarget, false);
    assert.equal(previewVolleyballAttempt(id, challenge, { ...attempt, timing: attempt.timing + .001 }).inTime, false);
  }
});

test('the spike defender checks the actual crossing of the net', () => {
  const challenge = { target: { x: 90, y: 200 }, timing_target: .7, blocker: { x: 120 } };
  const attempt = { aim_x: 90, aim_y: 200, timing: .7 };
  assert.equal(previewVolleyballAttempt('volley-rush', challenge, attempt).blocked, true);
  challenge.blocker.x = 265;
  assert.equal(previewVolleyballAttempt('volley-rush', challenge, attempt).blocked, false);
});

test('all three flight paths remain finite and preserve the aimed landing position', () => {
  for (const id of Object.keys(VOLLEYBALL_MODES)) {
    const challenge = { target: { x: 85, y: 150 }, timing_target: .7, blocker: { x: 290 } };
    const attempt = { aim_x: 85, aim_y: 150, timing: .7 };
    for (let step = 0; step <= 20; step += 1) {
      const pose = sampleVolleyballFlight(id, attempt, step / 20, challenge);
      for (const key of ['x', 'y', 'scale', 'rotation']) assert.ok(Number.isFinite(pose[key]));
    }
    if (id !== 'monster-block') {
      const final = sampleVolleyballFlight(id, attempt, 1, challenge);
      assert.ok(Math.abs(final.x - attempt.aim_x) < 1e-9);
      assert.ok(Math.abs(final.y - attempt.aim_y) < 1e-9);
    }
  }
});
