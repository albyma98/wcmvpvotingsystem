import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { organizationCrowd } from "./organizationCrowd.js";
import { readArenaScene, sceneSeatMap } from "./scene/sceneConfig.js";
import { SEATS } from "./seating/seatMap.js";
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

test("Baltur preserves all persisted fan slots across the configured spectator sectors", () => {
  const source = new URL(
    "../../../wcmvpvs-back/service/database/arena_scenes/baltur-arena.json",
    import.meta.url,
  );
  const scene = readArenaScene(JSON.parse(readFileSync(source, "utf8")));
  assert.ok(scene);
  assert.equal(scene.sectors.find(({ slug }) => slug === "tribuna-centrale").rows_count, 2);
  assert.equal(scene.sectors.find(({ slug }) => slug === "tribuna-laterale").rows_count, 8);
  assert.equal(scene.sectors.filter(({ glass }) => glass).length, 2);
  assert.equal(scene.activityAnchors.treasure.length, 10);

  const seats = sceneSeatMap(scene);
  assert.equal(seats.length, SEATS.length);
  assert.equal(new Set(seats.map(({ id }) => id)).size, SEATS.length);
  assert.ok(seats.some(({ sector, x }) => sector === "Settore Zimmer" && x < -scene.court.length / 2));
  assert.ok(seats.every(({ sector }) => !["Tribuna Press", "Dirigenza"].includes(sector)));
  const fans = seats.map((_, slot) => ({ id: seats.length - slot, slot }));
  const { actors } = organizationCrowd(fans, 7, seats);
  assert.equal(actors.length, seats.length);
  assert.equal(actors[0].seat, seats.at(-1));
  assert.equal(actors.at(-1).seat, seats[0]);
  assert.equal(sceneSeatMap(readArenaScene({ renderer: "joy-volley" })), SEATS);
});
