import test from "node:test";
import assert from "node:assert/strict";
import { Group, PerspectiveCamera, Matrix4 } from "three";
import { avatarLook, HAIR_NAMES, BODY_NAMES } from "./avatarLook.js";
import { avatarPose } from "./avatarPose.js";
import { createCrowd, MAX_DETAILED_AVATARS } from "./createCrowd.js";
import { createDemoSession } from "../actions/demoSession.js";
test("cosmetic variants are stable by profile and wardrobe overrides do not reset the base look", () => {
  const actor = createDemoSession().actors[0];
  const before = avatarLook(actor);
  assert.deepEqual(before, avatarLook({ ...actor }));
  assert.equal(HAIR_NAMES.length, 4);
  assert.equal(BODY_NAMES.length, 3);
  const changed = avatarLook({
    ...actor,
    equipment: {
      "look-hair": "3",
      "look-skin": "1",
      "look-body": "1",
      "look-shirt": "4",
      shirt: "club-shirt",
    },
  });
  assert.equal(changed.hair, 3);
  assert.equal(changed.body, 1);
  assert.equal(changed.shirt, "#3558a0");
  assert.notEqual(changed.skin, before.skin);
  const base = avatarLook({ ...actor, equipment: { "look-shirt": "4" } });
  assert.notEqual(base.shirt, changed.shirt);
  assert.deepEqual(actor.seat, createDemoSession().actors[0].seat);
});
test("seated and standing shoes rest on the walkway in front of the assigned seat", () => {
  const actor = createDemoSession().actors.find(
    (a) => a.kind === "profile" && a.seed % 13 !== 0,
  );
  for (const standing of [0, 1]) {
    const p = avatarPose(
      { ...actor, poseOverride: { standing } },
      0,
      null,
      true,
    );
    assert.ok(
      p.arms
        .flatMap((a) => [...a.shoulder, ...a.elbow, ...a.hand])
        .every(Number.isFinite),
    );
    for (const leg of p.legs) {
      assert.ok(leg.foot[1] - 0.05 >= -0.32);
      assert.ok(Math.abs(leg.foot[1] + 0.25) < 1e-8);
    }
  }
});
test("340 animated avatars keep a bounded detailed pool and consistent LOD transitions", () => {
  globalThis.window = { matchMedia: () => ({ matches: false }) };
  const actors = createDemoSession().actors;
  const original = actors.map((a) => ({ ...a.seat }));
  const crowd = createCrowd(new Group(), actors, { selectedId: actors[0].id });
  const c = new PerspectiveCamera();
  c.position.set(actors[0].seat.x, actors[0].seat.y + 2, actors[0].seat.z - 2);
  crowd.trigger("wave", actors[0].id, 1000);
  for (const time of [1000, 2200, 3500, 6000, 13000]) crowd.update(time, c);
  assert.equal(actors.length, 340);
  assert.ok(crowd.stats().detailed <= MAX_DETAILED_AVATARS);
  assert.ok(crowd.stats().detailed > 0);
  assert.ok(crowd.stats().triangles < 550000);
  let meshes = 0;
  crowd.group.traverse((o) => {
    if (o.isInstancedMesh) {
      meshes++;
      assert.ok(o.instanceMatrix.array.every(Number.isFinite));
    }
  });
  assert.ok(meshes <= 32);
  crowd.setLod("far");
  crowd.update(14000, c);
  assert.equal(crowd.stats().detailed, 0);
  crowd.setLod("auto");
  c.position.set(100, 40, 100);
  crowd.update(15000, c);
  assert.equal(crowd.stats().detailed, 1);
  window.innerWidth = 390;
  crowd.setLod("near");
  crowd.update(16000, c);
  assert.equal(crowd.stats().detailed, 4);
  assert.deepEqual(
    actors.map((a) => a.seat),
    original,
  );
});
