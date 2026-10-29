import test from "node:test";
import assert from "node:assert/strict";
import {
  CONTACTS,
  PLAYER_SPOTS,
  RALLY_SECONDS,
  sampleRally,
  playerPose,
} from "./volleyballRally.js";

test("six players per side stay inside their half throughout a complete rally", () => {
  assert.equal(PLAYER_SPOTS.filter((player) => player.side === -1).length, 6);
  assert.equal(PLAYER_SPOTS.filter((player) => player.side === 1).length, 6);
  for (let time = 0; time < RALLY_SECONDS * 2; time += 1 / 60) {
    const rally = sampleRally(time);
    assert.ok(rally.ball.every(Number.isFinite));
    assert.ok(rally.ball[1] > 0.18);
    PLAYER_SPOTS.forEach((spot, index) => {
      const pose = playerPose(index, rally, time);
      for (const key of ["x", "z", "jump", "reach", "dig", "stride"])
        assert.ok(Number.isFinite(pose[key]));
      assert.ok(pose.x * spot.side > 0.3 && Math.abs(pose.x) < 9);
      assert.ok(Math.abs(pose.z) < 4.5);
      assert.ok(pose.jump >= 0);
    });
  }
});

test("every flight reaches its next contact continuously and clears the net", () => {
  let elapsed = 0;
  CONTACTS.forEach((hit, index) => {
    const next = CONTACTS[(index + 1) % CONTACTS.length];
    const atContact = sampleRally(elapsed);
    hit.point.forEach((value, axis) =>
      assert.ok(Math.abs(atContact.ball[axis] - value) < 1e-8),
    );
    const before = sampleRally(elapsed + hit.duration - 1e-6);
    next.point.forEach((value, axis) =>
      assert.ok(Math.abs(before.ball[axis] - value) < 0.00002),
    );
    if (hit.point[0] * next.point[0] < 0) {
      const crossing = -hit.point[0] / (next.point[0] - hit.point[0]);
      const ball = sampleRally(elapsed + crossing * hit.duration).ball;
      assert.ok(Math.abs(ball[0]) < 1e-8);
      assert.ok(ball[1] - 0.18 > 2.43, "the whole ball passes above the tape");
      assert.ok(Math.abs(ball[2]) < 4.5);
    }
    elapsed += hit.duration;
  });
  assert.deepEqual(sampleRally(RALLY_SECONDS).ball, sampleRally(0).ball);
});

test("reduced motion leaves all twelve athletes in a grounded ready pose", () => {
  const rally = sampleRally(3);
  PLAYER_SPOTS.forEach((spot, index) => {
    const pose = playerPose(index, rally, 3, true);
    assert.equal(pose.x, spot.x);
    assert.equal(pose.z, spot.z);
    assert.equal(pose.jump, 0);
    assert.equal(pose.stride, 0);
    assert.equal(pose.reach, 0);
  });
});
