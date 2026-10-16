import test from 'node:test';
import assert from 'node:assert/strict';
import { CROSSOVER_ANIMATION_MS, CROSSOVER_MAX_DUEL_MS, CROSSOVER_WORLD, evaluateCrossoverDuel, getCrossoverScene, sampleCrossoverFinish } from './crossover.js';

const trace = (points) => points.map(([t_ms, x]) => ({ t_ms, x }));
const fast = trace([[0, 0], [120, 0.6], [320, 0.6], [460, -0.8]]);
const approx = (actual, expected, epsilon = 0.000001) => assert.ok(Math.abs(actual - expected) < epsilon, `${actual} != ${expected}`);

test('one continuous feint followed by a sharp opposite cut beats all three defenders', () => {
  for (const duelIndex of [0, 1, 2]) {
    const result = evaluateCrossoverDuel({ duelIndex, samples: fast });
    assert.equal(result.won, true);
    assert.equal(result.reason, 'beat');
    assert.equal(result.baitSide, 1);
    assert.equal(result.exitSide, -1);
    approx(result.cutAtMs, 445);
    approx(result.playerX, -0.65);
    approx(result.defenderX, 0.54);
    assert.ok(result.quality >= 80 && result.quality <= 100);
    const mirrored = evaluateCrossoverDuel({ duelIndex, samples: fast.map((sample) => ({ ...sample, x: -sample.x })) });
    assert.equal(mirrored.won, true);
    assert.equal(mirrored.quality, result.quality);
    approx(mirrored.playerX, -result.playerX);
    approx(mirrored.defenderX, -result.defenderX);
  }
  assert.equal(evaluateCrossoverDuel({ samples: fast }).quality, 89);
});

test('hold duration and threshold crossings are interpolated independently of sample rate', () => {
  const dense = [];
  for (let index = 1; index < fast.length; index += 1) {
    const a = fast[index - 1];
    const b = fast[index];
    if (!dense.length) dense.push({ ...a });
    for (let time = a.t_ms + 10; time < b.t_ms; time += 10) dense.push({ t_ms: time, x: a.x + (b.x - a.x) * (time - a.t_ms) / (b.t_ms - a.t_ms) });
    dense.push({ ...b });
  }
  const sparseResult = evaluateCrossoverDuel({ samples: fast });
  const denseResult = evaluateCrossoverDuel({ samples: dense });
  assert.equal(denseResult.won, sparseResult.won);
  assert.equal(denseResult.quality, sparseResult.quality);
  approx(denseResult.cutAtMs, sparseResult.cutAtMs);
  approx(denseResult.defenderX, sparseResult.defenderX);
});

test('a decisive clean crossover has the same quality golden value in all three duels', () => {
  const samples = trace([[0, 0], [100, 0.8], [400, 0.8], [500, -0.8]]);
  for (const duelIndex of [0, 1, 2]) {
    const result = evaluateCrossoverDuel({ duelIndex, samples });
    assert.equal(result.won, true);
    assert.equal(result.clean, true);
    assert.equal(result.quality, 94);
    approx(result.cutAtMs, 490.625);
    approx(result.defenderX, 0.72);
    assert.equal(Object.hasOwn(result, 'coins'), false, 'quality never decides a coin bonus');
  }
});

test('holding the bait for longer does not spend the reversal window', () => {
  const longHold = trace([[0, 0], [120, 0.6], [4000, 0.6], [4140, -0.8]]);
  for (const duelIndex of [0, 1, 2]) assert.equal(evaluateCrossoverDuel({ duelIndex, samples: longHold }).won, true);
});

test('incomplete feints, missing changes, slow changes and recovered defenders give no win', () => {
  assert.equal(evaluateCrossoverDuel({ samples: trace([[0, 0], [100, 0.7], [180, -0.8]]) }).reason, 'no_feint');
  assert.equal(evaluateCrossoverDuel({ samples: trace([[0, 0], [100, 0.6], [500, 0.6]]) }).reason, 'no_crossover');
  const tooSlow = evaluateCrossoverDuel({ samples: trace([[0, 0], [100, 0.6], [400, 0.6], [600, 0], [2000, -0.8]]) });
  assert.equal(tooSlow.reason, 'too_slow');
  const blocked = evaluateCrossoverDuel({ samples: trace([[0, 0], [100, 0.6], [400, 0.6], [1100, -0.8]]) });
  assert.equal(blocked.reason, 'blocked');
  for (const result of [tooSlow, blocked]) {
    assert.equal(result.won, false);
    assert.equal(result.quality, 0);
    assert.equal(result.clean, false);
  }
});

test('a subsequent corrected feint can win within the same six-second duel', () => {
  const correction = trace([[0, 0], [100, 0.6], [400, 0.6], [1100, -0.8], [1450, -0.8], [1590, 0.8]]);
  const result = evaluateCrossoverDuel({ samples: correction });
  assert.equal(result.won, true);
  assert.equal(result.baitSide, -1);
  assert.equal(result.exitSide, 1);
  assert.ok(result.cutAtMs > 1450 && result.cutAtMs < 1590);
});

test('the first winning crossing remains the result even if later movements continue', () => {
  const extended = [...fast, ...trace([[1000, -0.8], [1200, 0.9], [1600, 0.9], [1800, -0.9]])];
  assert.deepEqual(evaluateCrossoverDuel({ samples: extended }), evaluateCrossoverDuel({ samples: fast }));
});

test('invalid traces and invalid duel indices are rejected without an animation result', () => {
  const invalid = [[], [{ t_ms: 0, x: 0 }], trace([[1, 0], [100, 0.5]]), trace([[0, 0.1], [100, 0.5]]), trace([[0, 0], [100, Infinity]]), trace([[0, 0], [100, 1.01]]), trace([[0, 0], [100, -1.01]]), trace([[0, 0], [100.5, 0.5]]), trace([[0, 0], [100, 0.5], [100, 0.6]]), trace([[0, 0], [6001, 0.5]]), Array.from({ length: 129 }, (_, t_ms) => ({ t_ms, x: 0 }))];
  for (const samples of invalid) assert.equal(evaluateCrossoverDuel({ samples }), null);
  for (const duelIndex of [-1, 3, 0.5, '0', NaN]) assert.equal(evaluateCrossoverDuel({ duelIndex, samples: fast }), null);
  assert.equal(evaluateCrossoverDuel(), null);
});

test('the live defender follows the trace with the difficulty-specific delay', () => {
  assert.deepEqual(getCrossoverScene(), { playerX: 0, defenderX: 0, gap: 0, delayMs: 300, reversalWindowMs: 1000, feintReady: false });
  const scene = getCrossoverScene({ samples: fast, elapsedMs: 445 });
  approx(scene.playerX, -0.65);
  approx(scene.defenderX, 0.54);
  approx(scene.gap, 1.19);
  const recovered = getCrossoverScene({ samples: fast, elapsedMs: 1000 });
  approx(recovered.defenderX, -0.72);
  assert.equal(getCrossoverScene({ duelIndex: 2 }).delayMs, 200);
});

test('live feint guidance qualifies a continued hold between pointer events and clears after leaving that side', () => {
  const samples = trace([[0, 0], [100, 0.6]]);
  assert.equal(getCrossoverScene({ samples, elapsedMs: 254 }).feintReady, false);
  assert.equal(getCrossoverScene({ samples, elapsedMs: 255 }).feintReady, true);
  const changing = trace([[0, 0], [100, 0.6], [400, 0.6], [500, 0]]);
  assert.equal(getCrossoverScene({ samples: changing, elapsedMs: 400 }).feintReady, true);
  assert.equal(getCrossoverScene({ samples: changing, elapsedMs: 490 }).feintReady, false);
  assert.equal(getCrossoverScene({ samples: samples.map((point) => ({ ...point, x: -point.x })), elapsedMs: 255 }).feintReady, true);
});

test('the finish animation beats the defender, drives and dunks through the hoop continuously', () => {
  assert.deepEqual(CROSSOVER_WORLD, { width: 360, height: 560 });
  assert.equal(CROSSOVER_MAX_DUEL_MS, 6000);
  assert.equal(CROSSOVER_ANIMATION_MS, 1600);
  const result = evaluateCrossoverDuel({ samples: fast });
  assert.equal(sampleCrossoverFinish(result, 0).phase, 'crossover');
  assert.equal(sampleCrossoverFinish(result, 0.4).phase, 'drive');
  assert.equal(sampleCrossoverFinish(result, 0.8).phase, 'dunk');
  const net = sampleCrossoverFinish(result, 0.94);
  assert.equal(net.phase, 'net');
  assert.equal(net.netHit, true);
  assert.ok(net.ball.y > 112);
  const rim = sampleCrossoverFinish(result, 0.88);
  approx(rim.ball.x, 180);
  approx(rim.ball.y, 112);
  assert.equal(sampleCrossoverFinish(result, 1).ball.visible, false);
  for (const boundary of [0.2, 0.68, 0.88]) {
    const before = sampleCrossoverFinish(result, boundary - 0.000001);
    const after = sampleCrossoverFinish(result, boundary + 0.000001);
    for (const actor of ['player', 'ball']) {
      approx(before[actor].x, after[actor].x, 0.01);
      approx(before[actor].y, after[actor].y, 0.01);
      approx(before[actor].scale, after[actor].scale, 0.01);
    }
  }
});

test('failed duel animation stays finite and never reaches a scoring phase', () => {
  const result = evaluateCrossoverDuel({ samples: trace([[0, 0], [100, 0.7]]) });
  for (const progress of [-1, 0, 0.5, 1, 2, NaN]) {
    const pose = sampleCrossoverFinish(result, progress);
    assert.equal(pose.phase, 'blocked');
    assert.equal(pose.netHit, false);
    for (const actor of ['player', 'defender', 'ball']) for (const key of ['x', 'y', 'scale']) assert.ok(Number.isFinite(pose[actor][key]));
  }
  assert.equal(sampleCrossoverFinish(null), null);
});
