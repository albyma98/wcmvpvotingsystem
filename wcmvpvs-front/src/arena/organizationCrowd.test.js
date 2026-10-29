import test from "node:test";
import assert from "node:assert/strict";
import { organizationCrowd } from "./organizationCrowd.js";
test("real profiles map to unique deterministic seats across browsers and appended registrations", () => {
  const fans = [
    { id: 10, nickname: "Uno" },
    { id: 20, nickname: "Due", is_me: true },
  ];
  const a = organizationCrowd(fans, 7),
    b = organizationCrowd([...fans, { id: 30, nickname: "Tre" }], 7);
  assert.equal(a.selectedId, "20");
  assert.equal(a.actors.length, 2);
  assert.deepEqual(
    a.actors.map((x) => x.seat.id),
    b.actors.slice(0, 2).map((x) => x.seat.id),
  );
  assert.equal(new Set(b.actors.map((x) => x.seat.id)).size, 3);
  assert.ok(a.actors.every((x) => x.kind === "profile" && !x.profile.active));
  assert.equal(
    organizationCrowd(
      fans.map(({ is_me, ...fan }) => fan),
      7,
    ).selectedId,
    null,
  );
  assert.equal(organizationCrowd([], 7).actors.length, 0);
  assert.notDeepEqual(
    a.actors.map((x) => x.seat.id),
    organizationCrowd(fans, 8).actors.map((x) => x.seat.id),
  );
});
