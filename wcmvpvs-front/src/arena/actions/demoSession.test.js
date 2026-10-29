import test from "node:test";
import assert from "node:assert/strict";
import { createDemoSession, STORAGE_KEY } from "./demoSession.js";
import { SEATS, SEAT_BY_ID, assignSeats } from "../seating/seatMap.js";
import { createDemoProfiles } from "../crowd/demoProfiles.js";
import { createCrowdDirector } from "./crowdDirector.js";

function memoryStorage(initial = {}) {
  const data = new Map(Object.entries(initial));
  return {
    getItem: (key) => data.get(key) || null,
    setItem: (key, value) => data.set(key, value),
    data,
  };
}

test("60 demo profiles and 280 fillers occupy unique valid seats outside the stairs", () => {
  const session = createDemoSession(memoryStorage());
  assert.equal(session.profiles.length, 60);
  assert.equal(
    session.actors.filter((actor) => actor.kind === "filler").length,
    280,
  );
  assert.equal(new Set(session.actors.map((actor) => actor.seat.id)).size, 340);
  for (const actor of session.actors) {
    assert.ok(SEAT_BY_ID.has(actor.seat.id));
    assert.ok(actor.seat.row >= 1 && actor.seat.row <= 19);
    assert.ok(Math.abs(actor.seat.z) > 8.5);
  }
  assert.equal(SEATS.length, 1330);
  assert.ok(
    session.actors.filter((actor) => actor.seat.stand === "B").length > 170,
  );
});

test("seats, selected profile and vote survive reloading the local demo", () => {
  const storage = memoryStorage(),
    first = createDemoSession(storage);
  first.select("fan-010");
  assert.equal(first.vote("rinaldi").ok, true);
  const second = createDemoSession(storage);
  assert.equal(second.snapshot().selectedId, "fan-010");
  assert.equal(second.snapshot().votes["fan-010"], "rinaldi");
  assert.equal(second.snapshot().profiles[9].recentAction, "Ha votato l’MVP");
  assert.deepEqual(
    second.profiles.map((p) => p.seatId),
    first.profiles.map((p) => p.seatId),
  );
  assert.equal(storage.data.size, 1);
  assert.ok(storage.data.has(STORAGE_KEY));
});

test("duplicate and unknown votes cannot change the tally or interaction count", () => {
  const session = createDemoSession(memoryStorage());
  const before = session.snapshot().profiles[0].interactions;
  assert.equal(session.vote("unknown").ok, false);
  assert.equal(session.vote("leone").ok, true);
  assert.equal(session.vote("greco").ok, false);
  assert.equal(session.snapshot().votes["fan-001"], "leone");
  assert.equal(session.snapshot().profiles[0].interactions, before + 1);
  session.select("fan-002");
  assert.equal(session.vote("greco").ok, true);
  assert.equal(Object.keys(session.snapshot().votes).length, 2);
});

test("cooldowns are enforced per action and profile and expire", () => {
  const session = createDemoSession(memoryStorage()),
    now = Date.now();
  assert.equal(session.action("constructor", now).ok, false);
  assert.equal(session.action("clap", now).ok, true);
  assert.equal(session.action("clap", now + 100).ok, false);
  assert.equal(session.action("scarf", now + 100).ok, true);
  session.select("fan-002");
  assert.equal(session.action("clap", now + 100).ok, true);
  session.select("fan-001");
  assert.equal(session.action("clap", now + 4501).ok, true);
});

test("invalid persisted assignments and malformed storage recover without duplicate seats", () => {
  const profiles = createDemoProfiles();
  const repaired = assignSeats(profiles, {
    "fan-001": "B-08-24",
    "fan-002": "B-08-24",
    "fan-003": "no-seat",
  });
  assert.equal(new Set(Object.values(repaired)).size, 60);
  const session = createDemoSession(
    memoryStorage({ [STORAGE_KEY]: "{invalid json" }),
  );
  assert.equal(session.profiles.length, 60);
  assert.equal(session.snapshot().selectedId, "fan-001");
});

test("blocked browser storage keeps the demo usable and reports session-only persistence", () => {
  const storage = {
    getItem() {
      throw new Error("denied");
    },
    setItem() {
      throw new Error("quota");
    },
  };
  const session = createDemoSession(storage);
  assert.equal(session.snapshot().persistent, false);
  assert.equal(session.vote("leone").ok, true);
  assert.equal(session.snapshot().votes["fan-001"], "leone");
});

test("reset clears only demo interactions and votes while preserving seats", () => {
  const session = createDemoSession(memoryStorage()),
    seats = session.profiles.map((p) => p.seatId);
  session.vote("leone");
  session.action("scarf");
  session.reset();
  assert.deepEqual(session.snapshot().votes, {});
  assert.deepEqual(session.snapshot().cooldowns, {});
  assert.deepEqual(
    session.profiles.map((p) => p.seatId),
    seats,
  );
});

const actor = (id, x, stand = "B", kind = "profile") => ({
  id,
  kind,
  seat: { x, row: 8, stand },
});
test("applause reaches neighbours in the same stand and settles after the action", () => {
  const own = actor("own", 0),
    near = actor("near", 0.55, "B", "filler"),
    far = actor("far", 0, "A");
  const director = createCrowdDirector([own, near, far]);
  assert.equal(director.trigger("clap", "own", 0), true);
  assert.ok(director.sample(own, 1000).clap > 0);
  assert.ok(director.sample(near, 1000).clap > 0);
  assert.equal(director.sample(far, 1000), null);
  assert.equal(director.sample(near, 6000), null);
  assert.equal(director.trigger("clap", "near", 0), false);
});

test("the ola propagates across the stand instead of raising everybody at once", () => {
  const own = actor("own", 0),
    left = actor("left", -10),
    right = actor("right", 10);
  const director = createCrowdDirector([own, left, right]);
  director.trigger("wave", "own", 0);
  assert.ok(director.sample(left, 900).standing > 0);
  assert.equal(director.sample(right, 900), null);
  assert.ok(director.sample(right, 5000).standing > 0);
  assert.equal(director.sample(right, 9000), null);
});
