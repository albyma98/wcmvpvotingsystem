import test from 'node:test';
import assert from 'node:assert/strict';
import { PLAYMAKER_START, PLAYMAKER_HOOP, PLAYMAKER_ANIMATION_MS, getPlaymakerScene, createPlaymakerPass, samplePlaymakerPass } from './playmaker.js';

const to = (point) => ({ x: point.x - PLAYMAKER_START.x, y: point.y - PLAYMAKER_START.y });
const approx = (actual, expected, tolerance = 0.000001) => assert.ok(Math.abs(actual - expected) < tolerance, `${actual} != ${expected}`);

test('opening scene and lanes are deterministic golden values shared with the backend', () => {
  const scene = getPlaymakerScene();
  assert.deepEqual(scene.teammates, [{ id: 'left', x: 82, y: 248 }, { id: 'right', x: 278, y: 248 }]);
  approx(scene.defenders[0].x, 208.76553231625218);
  assert.equal(scene.speed, 1.15);
  const left = createPlaymakerPass({ drag: to(scene.teammates[0]) });
  const right = createPlaymakerPass({ drag: to(scene.teammates[1]) });
  assert.equal(left.outcome, 'assist');
  assert.equal(left.assisted, true);
  assert.equal(left.reason, 'assist');
  assert.equal(left.clean, true);
  assert.equal(right.outcome, 'intercepted');
  assert.equal(right.made, false);
  assert.equal(right.interceptor.id, 'guard');
  assert.equal(right.reason, 'intercepted');
  assert.deepEqual(right.interceptPoint, { x: right.interceptor.x, y: right.interceptor.y });
  assert.deepEqual(createPlaymakerPass({ drag: to(scene.teammates[0]) }), left);
});

test('successful receptions require an aimed endpoint and a free passing lane', () => {
  const teammate = getPlaymakerScene().teammates[0];
  const aimed = to(teammate);
  const near = createPlaymakerPass({ drag: { x: aimed.x + 28, y: aimed.y } });
  assert.equal(near.made, true);
  assert.equal(near.clean, false);
  for (const offset of [{ x: 36, y: 0 }, { x: 0, y: 43 }, { x: 28, y: 34 }]) {
    const miss = createPlaymakerPass({ drag: { x: aimed.x + offset.x, y: aimed.y + offset.y } });
    assert.equal(miss.outcome, 'off_target');
    assert.equal(miss.receiver, null);
    assert.equal(miss.made, false);
  }
});

test('at least one teammate always has a clear lane as difficulty rises', () => {
  for (let attempt = 0; attempt <= 40; attempt += 1) {
    for (let elapsedMs = 0; elapsedMs <= 30000; elapsedMs += 250) {
      const scene = getPlaymakerScene({ attempt, elapsedMs });
      const options = scene.teammates.map((teammate) => createPlaymakerPass({ attempt, elapsedMs, drag: to(teammate) }));
      assert.ok(options.some((pass) => pass.made), `no clear lane at attempt ${attempt}, time ${elapsedMs}`);
      assert.equal(scene.defenders.length, attempt >= 8 ? 2 : 1);
      assert.ok(scene.speed >= 1.15 && scene.speed <= 1.95);
    }
  }
});

test('release snapshots change with session time, scene previews clamp to 30 seconds and do not share mutable state', () => {
  const atZero = getPlaymakerScene({ attempt: 12, elapsedMs: 0 });
  const later = getPlaymakerScene({ attempt: 12, elapsedMs: 1000 });
  assert.notDeepEqual(later.teammates, atZero.teammates);
  assert.deepEqual(getPlaymakerScene({ attempt: 12, elapsedMs: -1000 }), atZero);
  assert.deepEqual(getPlaymakerScene({ attempt: 12, elapsedMs: 50000 }), getPlaymakerScene({ attempt: 12, elapsedMs: 30000 }));
  const pass = createPlaymakerPass({ attempt: 12, elapsedMs: 1000, drag: to(later.teammates[0]) });
  later.teammates[0].x = -100;
  assert.notEqual(pass.scene.teammates[0].x, -100);
  assert.equal(pass.scene.attempt, 12);
  assert.equal(pass.scene.elapsedMs, 1000);
});

test('cancelled gestures, taps, sideways passes and nonfinite input never launch', () => {
  for (const drag of [{ x: 0, y: 0 }, { x: 0, y: -41 }, { x: 100, y: -29 }, { x: NaN, y: -200 }, { x: 0, y: -Infinity }, { x: '0', y: -218 }]) {
    assert.equal(createPlaymakerPass({ drag }), null);
  }
  const drag = to(getPlaymakerScene().teammates[0]);
  assert.equal(createPlaymakerPass({ drag, cancelled: true }), null);
  assert.equal(createPlaymakerPass({ drag, elapsedMs: NaN }), null);
  assert.equal(createPlaymakerPass({ drag, elapsedMs: -1 }), null);
  assert.equal(createPlaymakerPass({ drag, elapsedMs: 30001 }), null);
  assert.equal(createPlaymakerPass({ drag, attempt: -1 }), null);
  assert.equal(createPlaymakerPass({ drag, attempt: 1.5 }), null);
  assert.equal(createPlaymakerPass({ drag, attempt: Infinity }), null);
  assert.equal(createPlaymakerPass({ drag, attempt: '1' }), null);
  assert.equal(createPlaymakerPass(), null);
});

test('animation follows the receiver through the drive and dunk without deciding credit', () => {
  const pass = createPlaymakerPass({ drag: to(getPlaymakerScene().teammates[0]) });
  assert.equal(pass.flightMs, PLAYMAKER_ANIMATION_MS);
  assert.equal(Object.hasOwn(pass, 'coins'), false);
  const launch = samplePlaymakerPass(pass, 0);
  assert.deepEqual({ x: launch.x, y: launch.y, scale: launch.scale }, { ...PLAYMAKER_START, scale: 1 });
  assert.equal(launch.phase, 'pass');
  assert.deepEqual(launch.ball, { x: launch.x, y: launch.y, scale: launch.scale, rotation: launch.rotation, visible: launch.visible });
  assert.equal(launch.runner, null);
  const reception = samplePlaymakerPass(pass, 0.34);
  approx(reception.x, pass.receiver.x);
  approx(reception.y, pass.receiver.y);
  assert.equal(samplePlaymakerPass(pass, 0.5).phase, 'drive');
  assert.ok(samplePlaymakerPass(pass, 0.5).receiverPose);
  assert.ok(samplePlaymakerPass(pass, 0.5).runner);
  assert.equal(samplePlaymakerPass(pass, 0.5).runner.jump, 0);
  assert.equal(samplePlaymakerPass(pass, 0.8).phase, 'dunk');
  assert.ok(samplePlaymakerPass(pass, 0.8).runner.jump > 0);
  const entry = samplePlaymakerPass(pass, 0.88);
  approx(entry.x, PLAYMAKER_HOOP.x);
  approx(entry.y, PLAYMAKER_HOOP.y);
  const net = samplePlaymakerPass(pass, 0.94);
  assert.equal(net.phase, 'net');
  assert.equal(net.netHit, true);
  assert.ok(net.y > PLAYMAKER_HOOP.y);
  assert.equal(samplePlaymakerPass(pass, 1).visible, false);
  for (const boundary of [0.34, 0.72, 0.88]) {
    const before = samplePlaymakerPass(pass, boundary - 0.000001);
    const after = samplePlaymakerPass(pass, boundary + 0.000001);
    approx(before.x, after.x, 0.01);
    approx(before.y, after.y, 0.01);
    approx(before.scale, after.scale, 0.01);
  }
});

test('interceptions stop at the defender and miss animations remain finite', () => {
  const pass = createPlaymakerPass({ drag: to(getPlaymakerScene().teammates[1]) });
  const contact = samplePlaymakerPass(pass, 0.34);
  approx(contact.x, pass.interceptor.x);
  approx(contact.y, pass.interceptor.y);
  assert.equal(samplePlaymakerPass(pass, 0.5).phase, 'intercepted');
  assert.equal(samplePlaymakerPass(pass, 0.9).receiverPose, null);
  const miss = createPlaymakerPass({ drag: { x: 500, y: -400 } });
  for (const progress of [-1, 0, 0.34, 0.75, 1, 2, NaN]) {
    for (const shot of [pass, miss]) {
      const point = samplePlaymakerPass(shot, progress);
      for (const key of ['x', 'y', 'scale', 'rotation']) assert.ok(Number.isFinite(point[key]));
      assert.equal(point.netHit, false);
    }
  }
  assert.equal(samplePlaymakerPass(null), null);
});
