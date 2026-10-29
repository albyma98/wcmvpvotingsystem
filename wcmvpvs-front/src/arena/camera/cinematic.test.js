import test from "node:test";
import assert from "node:assert/strict";
import { createTourSampler, TOUR_SECONDS } from "./cinematic.js";

test("queued frame before the start of the tour has a valid, clamped camera pose", () => {
  const sample = createTourSampler()(-16);
  assert.equal(sample.progress, 0);
  assert.equal(sample.ease, 0);
  assert.deepEqual(sample.position.toArray(), [25, 17, 27]);
});

test("the complete 18-second path stays above the court and has finite camera poses", () => {
  const sample = createTourSampler();
  for (let ms = 0; ms <= TOUR_SECONDS * 1000; ms += 16) {
    const pose = sample(ms);
    assert.ok(
      [...pose.position.toArray(), ...pose.target.toArray()].every(
        Number.isFinite,
      ),
    );
    assert.ok(pose.position.y >= 1.3);
    assert.ok(pose.position.distanceTo(pose.target) > 5);
  }
});

test("a delayed frame ends at the final view instead of overshooting the spline", () => {
  const sample = createTourSampler()(TOUR_SECONDS * 1000 + 5000);
  assert.equal(sample.progress, 1);
  assert.equal(sample.ease, 1);
  sample.position
    .toArray()
    .forEach((value, i) => assert.ok(Math.abs(value - [22, 12, 23][i]) < 1e-9));
});
