import test from "node:test";
import assert from "node:assert/strict";
import { Group, PerspectiveCamera, Matrix4 } from "three";
import { createCrowd } from "./createCrowd.js";
import { createDemoSession } from "../actions/demoSession.js";

test("empty club and guest crowd do not select a fan implicitly", () => {
  globalThis.window = { matchMedia: () => ({ matches: false }) };
  for (const actors of [[], createDemoSession().actors.slice(0, 2)]) {
    const crowd = createCrowd(new Group(), actors, { selectedId: null });
    crowd.update(100, new PerspectiveCamera());
    assert.equal(crowd.selectedPosition(), null);
  }
});

test("crowd poses stay finite through actions, filtering and an elapsed background interval", () => {
  globalThis.window = { matchMedia: () => ({ matches: false }) };
  const session = createDemoSession(),
    root = new Group(),
    camera = new PerspectiveCamera();
  camera.position.set(17, 10.5, 18);
  const crowd = createCrowd(root, session.actors, { selectedId: "fan-001" });
  for (const [kind, start] of [
    ["clap", 1000],
    ["scarf", 20000],
    ["wave", 40000],
    ["vote", 60000],
  ]) {
    crowd.trigger(kind, "fan-001", start);
    for (const time of [start, start + 500, start + 2800])
      crowd.update(time, camera);
    if (kind === "scarf")
      assert.ok(
        root.getObjectByName("fan-scarves").instanceMatrix.array[0] > 0.1,
      );
    crowd.update(start + 15000, camera);
  }
  crowd.setFilters({ fillers: false, onlyActive: true });
  crowd.update(90000, camera);
  const torsos = root.getObjectByName("fan-shirts");
  assert.equal(torsos.count, 340);
  assert.equal(
    new Matrix4().fromArray(torsos.instanceMatrix.array, 60 * 16).determinant(),
    0,
  );
  crowd.setFilters({ fillers: true, onlyActive: false });
  crowd.update(91000, camera);
  for (const object of crowd.group.children.filter(
    (object) => object.isInstancedMesh,
  ))
    assert.ok(object.instanceMatrix.array.every(Number.isFinite));
  assert.equal(crowd.selectedPosition().x, session.actors[0].seat.x);
});
