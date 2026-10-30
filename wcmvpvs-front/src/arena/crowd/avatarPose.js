// Both LODs use the same seated/standing joints, in the existing seat space.
export function avatarPose(
  actor,
  now,
  reaction,
  reducedMotion = false,
  animated = true,
) {
  const t = now / 1000 + (actor.seed || 0) * 0.117,
    cycle = t % 24;
  const idleClap =
    !reducedMotion && animated && cycle > 9 && cycle < 11
      ? Math.sin(((cycle - 9) / 2) * Math.PI)
      : 0;
  const idleCheer =
    !reducedMotion && animated && cycle > 3 && cycle < 4.8
      ? Math.sin(((cycle - 3) / 1.8) * Math.PI) * 0.55
      : 0;
  const idleWave =
    !reducedMotion && animated && cycle > 17 && cycle < 19
      ? Math.sin(((cycle - 17) / 2) * Math.PI)
      : 0;
  const standing = Math.max(
    actor.poseOverride?.standing ?? (actor.seed % 13 === 0 ? 1 : 0),
    reaction?.standing || 0,
    idleCheer * 0.4,
  );
  const hip =
    (actor.seat.hasSeat ? 0.25 : 0.1) +
    (0.55 - (actor.seat.hasSeat ? 0.25 : 0.1)) * standing;
  const bob = !reducedMotion && animated ? Math.sin(t * 1.15) * 0.007 : 0;
  const clap = reaction?.clap || idleClap,
    raise = reaction?.raise || idleCheer,
    scarf = reaction?.scarf || 0;
  const p = {
    hip: hip + bob,
    standing,
    forward: -0.27 * standing,
    clap,
    scarf,
    turn: !reducedMotion && animated ? Math.sin(t * 0.6) * 0.045 : 0,
    headTurn: !reducedMotion && animated ? Math.sin(t * 0.42) * 0.09 : 0,
    arms: [],
    legs: [],
  };
  for (const sign of [-1, 1]) {
    let elbow = [sign * 0.24, hip + 0.19, -0.025],
      hand = [sign * 0.24, hip + 0.04, -0.1];
    if (clap) {
      elbow = [sign * 0.235, hip + 0.235, -0.11];
      hand = [
        sign * (0.037 + 0.045 * (0.5 + 0.5 * Math.sin(t * 12))),
        hip + 0.3,
        -0.27,
      ];
    }
    const lift = Math.max(raise, scarf, sign > 0 ? idleWave : 0);
    if (lift) {
      elbow = [sign * 0.24, hip + 0.18 + 0.36 * lift, -0.035];
      hand = [sign * (0.22 + 0.1 * lift), hip + 0.07 + 0.765 * lift, -0.045];
    }
    p.arms.push({ shoulder: [sign * 0.19, hip + 0.33, 0], elbow, hand });
    const knee = [
      sign * 0.09,
      hip - 0.065 + (0.22 - (hip - 0.065)) * standing,
      -0.24 * (1 - standing) - 0.012,
    ];
    const foot = [sign * 0.09, -0.25, -0.255 * (1 - standing) - 0.025];
    p.legs.push({ hip: [sign * 0.085, hip - 0.06, 0], knee, foot });
  }
  return p;
}
