import test from 'node:test';
import assert from 'node:assert/strict';
import {
  BASKETBALL_START,
  BASKETBALL_HOOP,
  BASKETBALL_FLIGHT_MS,
  basketballDragFromPoints,
  basketballHoopForAttempt,
  createBasketballShot,
  sampleBasketballShot,
} from './basketballShot.js';

const aimedDrag = (hoop = BASKETBALL_HOOP) => ({ x: hoop.x - BASKETBALL_START.x, y: hoop.y - BASKETBALL_START.y });
const approx = (actual, expected) => assert.ok(Math.abs(actual - expected) < 0.000001, `${actual} should equal ${expected}`);

test('a drag aimed at the hoop produces one deterministic clean basket', () => {
  const shot = createBasketballShot({ drag: aimedDrag() });
  assert.equal(shot.made, true);
  assert.equal(shot.clean, true);
  assert.equal(shot.quality, 'swish');
  assert.equal(shot.flightMs, BASKETBALL_FLIGHT_MS);
  assert.deepEqual(shot, createBasketballShot({ drag: aimedDrag() }));
  assert.equal(Object.hasOwn(shot, 'coins'), false, 'animation does not award coins');
});

test('near-rim aiming can score while short, wide and long shots miss', () => {
  const center = aimedDrag();
  const rim = createBasketballShot({ drag: { x: center.x + 24, y: center.y } });
  assert.equal(rim.made, true);
  assert.equal(rim.clean, false);
  assert.equal(rim.quality, 'rim');
  for (const [drag, missKind] of [
    [{ x: center.x + 31, y: center.y }, 'wide'],
    [{ x: center.x, y: center.y + 41 }, 'short'],
    [{ x: center.x, y: center.y - 41 }, 'long'],
    [{ x: center.x + 24, y: center.y + 32 }, null],
  ]) {
    const shot = createBasketballShot({ drag });
    if (missKind) {
      assert.equal(shot.made, false);
      assert.equal(shot.missKind, missKind);
    } else {
      assert.equal(shot.made, false, 'two individually close offsets outside the ellipse still miss');
    }
  }
});

test('taps, short drags, sideways drags, cancellation and invalid input do not launch a shot', () => {
  for (const drag of [{ x: 0, y: 0 }, { x: 0, y: -41 }, { x: 200, y: -29 }, { x: 0, y: 200 }, { x: NaN, y: -200 }, { x: 0, y: -Infinity }, { x: '0', y: -266 }]) {
    assert.equal(createBasketballShot({ drag }), null);
  }
  assert.equal(createBasketballShot(), null);
  assert.equal(createBasketballShot({ drag: aimedDrag(), cancelled: true }), null);
});

test('pointer coordinates score identically at phone, tablet and full-size viewport scales', () => {
  for (const scale of [0.65, 1, 1.5, 2]) {
    const drag = basketballDragFromPoints({
      start: { x: 30 + BASKETBALL_START.x * scale, y: 70 + BASKETBALL_START.y * scale },
      end: { x: 30 + BASKETBALL_HOOP.x * scale, y: 70 + BASKETBALL_HOOP.y * scale },
      width: 360 * scale,
      height: 560 * scale,
    });
    approx(drag.x, 0);
    approx(drag.y, -266);
    assert.equal(createBasketballShot({ drag }).made, true);
  }
  assert.equal(basketballDragFromPoints({ start: { x: 0, y: 0 }, end: { x: 0, y: 0 }, width: 0, height: 560 }), null);
});

test('hoop targets change predictably and every issued target is aimable', () => {
  const targets = Array.from({ length: 8 }, (_, attempt) => basketballHoopForAttempt(attempt));
  assert.deepEqual(targets, [
    { x: 180, y: 200 }, { x: 120, y: 205 }, { x: 240, y: 195 }, { x: 150, y: 215 },
    { x: 225, y: 205 }, { x: 105, y: 195 }, { x: 255, y: 215 }, { x: 180, y: 185 },
  ]);
  for (const hoop of targets) assert.equal(createBasketballShot({ hoop, drag: aimedDrag(hoop) }).made, true);
  assert.deepEqual(basketballHoopForAttempt(8), targets[0]);
  assert.deepEqual(basketballHoopForAttempt(-1), targets[0]);
  const copy = basketballHoopForAttempt(0);
  copy.x = -100;
  assert.deepEqual(basketballHoopForAttempt(0), targets[0]);
});

test('the made-ball flight shrinks, arcs above the rim and drops continuously through the net', () => {
  const shot = createBasketballShot({ drag: aimedDrag() });
  const launch = sampleBasketballShot(shot, 0);
  const apex = sampleBasketballShot(shot, 0.37);
  const entry = sampleBasketballShot(shot, 0.74);
  const afterEntry = sampleBasketballShot(shot, 0.740001);
  const drop = sampleBasketballShot(shot, 0.92);
  assert.deepEqual({ x: launch.x, y: launch.y, scale: launch.scale }, { ...BASKETBALL_START, scale: 1 });
  assert.ok(apex.y < BASKETBALL_HOOP.y);
  approx(entry.x, BASKETBALL_HOOP.x);
  approx(entry.y, BASKETBALL_HOOP.y);
  approx(entry.scale, 0.38);
  assert.ok(Math.abs(afterEntry.y - entry.y) < 0.001);
  assert.ok(Math.abs(afterEntry.x - entry.x) < 0.001);
  assert.ok(drop.y > entry.y);
  assert.equal(drop.phase, 'net');
  assert.equal(sampleBasketballShot(shot, 1).visible, false);
});

test('missed-shot animation remains finite and cannot change its scoring result', () => {
  const shot = createBasketballShot({ drag: { x: 500, y: -200 } });
  for (const progress of [-1, 0, 0.5, 0.74, 0.9, 1, 2, NaN]) {
    const sample = sampleBasketballShot(shot, progress);
    for (const key of ['x', 'y', 'scale', 'rotation', 'shadowScale']) assert.ok(Number.isFinite(sample[key]));
  }
  assert.equal(sampleBasketballShot(shot, 0.9).phase, 'miss');
  assert.equal(shot.made, false);
  assert.equal(sampleBasketballShot(null, 0.5), null);
});
