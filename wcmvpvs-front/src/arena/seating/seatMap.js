import { arenaLayout } from "../scene/layout.js";

export function seatColumns() {
  return Array.from({ length: 41 }, (_, column) => ({
    column,
    x: -11 + column * 0.55,
  })).filter(
    ({ x }) =>
      !arenaLayout.stand.aisles.some((aisle) => Math.abs(x - aisle) < 0.7),
  );
}

export function createSeatMap() {
  const seats = [];
  for (const side of [-1, 1])
    for (let row = 0; row < arenaLayout.stand.rows; row++) {
      for (const { column, x } of seatColumns()) {
        const stand = side < 0 ? "A" : "B";
        const zone =
          column < 13
            ? "Laterale ovest"
            : column > 27
              ? "Laterale est"
              : "Centrale";
        seats.push({
          id: `${stand}-${String(row + 1).padStart(2, "0")}-${String(column + 1).padStart(2, "0")}`,
          stand,
          side,
          row: row + 1,
          number: column + 1,
          column,
          sector: `${stand} · ${zone}`,
          x: side < 0 ? -x : x,
          y: (row + 1) * arenaLayout.stand.rowRise,
          z:
            side *
            (arenaLayout.stand.frontZ + row * arenaLayout.stand.rowDepth),
          yaw: side < 0 ? Math.PI : 0,
          hasSeat: side > 0,
        });
      }
    }
  return seats;
}

export const SEATS = createSeatMap();
export const SEAT_BY_ID = new Map(SEATS.map((seat) => [seat.id, seat]));

export function seededRandom(seed) {
  let value = seed >>> 0;
  return () => {
    value = (Math.imul(value, 1664525) + 1013904223) >>> 0;
    return (value + 1) / 4294967297;
  };
}

// Warm sectors have a denser core and softer shoulders, with irregular gaps.
export function seatWeight(seat) {
  const gaussian = (row, col, radius) =>
    Math.exp(
      -((seat.row - row) ** 2 + (seat.column - col) ** 2) / (2 * radius ** 2),
    );
  const clusters =
    1.7 * gaussian(8, 20, 4.5) +
    1.1 * gaussian(12, 30, 3.8) +
    0.8 * gaussian(5, 9, 3);
  return (0.2 + clusters) * (seat.stand === "B" ? 1.65 : 1);
}

export function weightedSeats(seats, seed, weight = seatWeight) {
  const random = seededRandom(seed);
  return seats
    .map((seat) => ({ seat, key: -Math.log(random()) / weight(seat) }))
    .sort((a, b) => a.key - b.key)
    .map(({ seat }) => seat);
}

export function assignSeats(profiles, stored = {}) {
  const assigned = {},
    used = new Set();
  for (const profile of profiles) {
    const id = stored?.[profile.id];
    if (SEAT_BY_ID.has(id) && !used.has(id)) {
      assigned[profile.id] = id;
      used.add(id);
    }
  }
  // The first local fan starts in a central, easy-to-find seat.
  if (!assigned[profiles[0].id] && !used.has("B-08-24")) {
    assigned[profiles[0].id] = "B-08-24";
    used.add("B-08-24");
  }
  const available = weightedSeats(
    SEATS.filter((seat) => !used.has(seat.id)),
    20261029,
  );
  for (const profile of profiles)
    if (!assigned[profile.id]) assigned[profile.id] = available.shift().id;
  return assigned;
}

export function createFillers(assignments, count = 280) {
  const occupied = new Set(Object.values(assignments));
  const profiles = Array.from(occupied, (id) => SEAT_BY_ID.get(id)).filter(
    Boolean,
  );
  const crowdWeight = (seat) => {
    let nearest = Infinity;
    for (const profile of profiles)
      if (profile.stand === seat.stand)
        nearest = Math.min(
          nearest,
          (profile.row - seat.row) ** 2 + (profile.column - seat.column) ** 2,
        );
    return seatWeight(seat) * (1 + 0.7 * Math.exp(-nearest / 12));
  };
  return weightedSeats(
    SEATS.filter((seat) => !occupied.has(seat.id)),
    83172,
    crowdWeight,
  )
    .slice(0, count)
    .map((seat, index) => ({
      id: `crowd-${index}`,
      kind: "filler",
      seat,
      seed: index * 19 + 317,
    }));
}
