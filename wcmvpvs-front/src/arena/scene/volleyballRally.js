// A choreographed, continuous six-a-side rally. Coordinates follow the court:
// X is the direction of play, Z runs along the net, Y is height in metres.
export const PLAYER_SPOTS = [-1, 1].flatMap((side) =>
  [-2.8, 0, 2.8, -2.6, 0, 2.6].map((z, slot) => ({
    side,
    slot,
    x: side * (slot < 3 ? 1.7 : 6),
    z,
  })),
);

export const CONTACTS = [
  { player: 5, point: [-6, 0.95, 2.6], kind: "dig", duration: 1.5, arc: 1.3 },
  { player: 1, point: [-1.7, 2.12, 0], kind: "set", duration: 1.35, arc: 1.65 },
  {
    player: 2,
    point: [-0.85, 3.12, 2.5],
    kind: "spike",
    duration: 1.25,
    arc: 0.65,
  },
  { player: 11, point: [6, 0.95, 2.6], kind: "dig", duration: 1.5, arc: 1.3 },
  { player: 7, point: [1.7, 2.12, 0], kind: "set", duration: 1.35, arc: 1.65 },
  {
    player: 6,
    point: [0.85, 3.12, -2.5],
    kind: "spike",
    duration: 1.25,
    arc: 0.65,
  },
];
export const RALLY_SECONDS = CONTACTS.reduce(
  (sum, hit) => sum + hit.duration,
  0,
);

export function sampleRally(seconds) {
  const time = ((seconds % RALLY_SECONDS) + RALLY_SECONDS) % RALLY_SECONDS;
  let elapsed = 0,
    index = 0;
  for (; index < CONTACTS.length - 1; index++) {
    if (time < elapsed + CONTACTS[index].duration) break;
    elapsed += CONTACTS[index].duration;
  }
  const from = CONTACTS[index],
    to = CONTACTS[(index + 1) % CONTACTS.length];
  const t = (time - elapsed) / from.duration;
  return {
    index,
    t,
    from,
    to,
    ball: from.point.map(
      (value, axis) =>
        value +
        (to.point[axis] - value) * t +
        (axis === 1 ? 4 * from.arc * t * (1 - t) : 0),
    ),
  };
}

export function playerPose(index, rally, seconds, reducedMotion = false) {
  const spot = PLAYER_SPOTS[index];
  const pose = {
    x: spot.x,
    z: spot.z,
    jump: 0,
    reach: 0,
    dig: 0,
    stride: 0,
    block: false,
  };
  if (reducedMotion) return pose;
  // Ease into the next contact, then recover from the previous one.
  const anticipate = Math.max(0, (rally.t - 0.38) / 0.62);
  const recover = Math.max(0, 1 - rally.t / 0.43);
  let hit,
    strength = 0;
  if (rally.to.player === index) {
    hit = rally.to;
    strength = anticipate;
  }
  if (rally.from.player === index) {
    hit = rally.from;
    strength = recover;
  }
  if (hit) {
    const ease = strength * strength * (3 - 2 * strength);
    pose.x += (hit.point[0] - spot.x) * ease;
    pose.z += (hit.point[2] - spot.z) * ease;
    pose.reach = hit.kind === "dig" ? 0 : ease;
    pose.dig = hit.kind === "dig" ? ease : 0;
    pose.jump =
      hit.kind === "spike" ? 1.04 * ease : hit.kind === "set" ? 0.08 * ease : 0;
    pose.stride = Math.sin(seconds * 13 + index) * (1 - strength) * 0.32;
  } else {
    // Front row on the defending side closes for a two-player block.
    const attack =
      rally.to.kind === "spike"
        ? rally.to
        : rally.from.kind === "spike"
          ? rally.from
          : null;
    if (
      attack &&
      spot.slot < 3 &&
      spot.side !== Math.sign(attack.point[0]) &&
      spot.slot !== 1
    ) {
      const strength =
        rally.to === attack ? anticipate : Math.max(0, 1 - rally.t / 0.32);
      pose.x += (spot.side * 0.58 - spot.x) * strength;
      pose.z +=
        (attack.point[2] + (spot.slot === 0 ? -0.4 : 0.4) - spot.z) * strength;
      pose.jump = 0.62 * strength;
      pose.reach = strength;
      pose.block = true;
    } else {
      pose.x += Math.sin(seconds * 1.7 + index * 2) * 0.18;
      pose.z += Math.cos(seconds * 1.3 + index) * 0.2;
      pose.stride = Math.sin(seconds * 5 + index) * 0.08;
    }
  }
  return pose;
}
