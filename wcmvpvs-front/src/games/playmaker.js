/**
 * Playmaker rules shared with the reward API. Coordinates use a 360 x 560
 * world. `elapsedMs` runs from the start of the 30-second session. Movement
 * is deterministic and frozen at release, so
 * animation timing and frame rate never decide whether an assist is earned.
 */
export const PLAYMAKER_WORLD = Object.freeze({ width: 360, height: 560 });
export const PLAYMAKER_START = Object.freeze({ x: 180, y: 466 });
export const PLAYMAKER_HOOP = Object.freeze({ x: 180, y: 118 });
export const PLAYMAKER_ANIMATION_MS = 1400;
export const PLAYMAKER_MAX_ELAPSED_MS = 30000;

const clamp = (value, min, max) => Math.min(max, Math.max(min, value));
const lerp = (from, to, amount) => from + (to - from) * amount;
const isPoint = (point) => point != null && Number.isFinite(point.x) && Number.isFinite(point.y);

/**
 * Both defenders share the same movement sign. At any instant they occupy
 * the same court half, leaving at least the opposite teammate's lane clear.
 * Increasing attempts raise speed; from attempt 8 a second defender joins.
 */
export function getPlaymakerScene({ attempt = 0, elapsedMs = 0 } = {}) {
  const index = Number.isFinite(attempt) ? Math.max(0, Math.floor(attempt)) : 0;
  const elapsed = clamp(Number.isFinite(elapsedMs) ? elapsedMs : 0, 0, PLAYMAKER_MAX_ELAPSED_MS);
  const level = Math.min(5, Math.floor(index / 4));
  const speed = 1.15 + level * 0.16;
  const phase = (index % 8) * 0.65;
  const time = elapsed / 1000;
  const teammateWave = Math.sin(time * speed * 0.72 + phase);
  const defenderWave = Math.sin(time * speed + phase + 0.5);
  const defenders = [{ id: 'guard', x: 180 + 60 * defenderWave, y: 350, radius: 22 }];
  if (index >= 8) defenders.push({ id: 'help', x: 180 + 92 * defenderWave, y: 318, radius: 20 });
  return {
    attempt: index,
    elapsedMs: elapsed,
    level,
    speed,
    teammates: [
      { id: 'left', x: 82 + 18 * teammateWave, y: 248 },
      { id: 'right', x: 278 - 18 * teammateWave, y: 248 },
    ],
    defenders,
  };
}

/** Closest point on a segment, also useful for the intercepted-ball animation. */
function pointOnSegment(start, end, point) {
  const dx = end.x - start.x;
  const dy = end.y - start.y;
  const lengthSquared = dx * dx + dy * dy;
  const t = lengthSquared ? clamp(((point.x - start.x) * dx + (point.y - start.y) * dy) / lengthSquared, 0, 1) : 0;
  const x = start.x + dx * t;
  const y = start.y + dy * t;
  return { x, y, t, distanceSquared: (point.x - x) ** 2 + (point.y - y) ** 2 };
}

/**
 * Returns null for cancelled/tap/invalid input, otherwise a frozen-in-time
 * pass description: { scene, start, hoop, aim, target, receiver, interceptor,
 * made, clean, precision, outcome, flightMs }. `made` is a prediction only;
 * the API confirms the sole one-coin reward after validating this same pass.
 */
export function createPlaymakerPass({ attempt = 0, elapsedMs = 0, drag, cancelled = false } = {}) {
  if (cancelled || !isPoint(drag) || !Number.isFinite(elapsedMs) || elapsedMs < 0 || elapsedMs > PLAYMAKER_MAX_ELAPSED_MS || !Number.isInteger(attempt) || attempt < 0) return null;
  const length = Math.hypot(drag.x, drag.y);
  if (!Number.isFinite(length) || length < 42 || drag.y > -30) return null;

  const scene = getPlaymakerScene({ attempt, elapsedMs });
  const start = { ...PLAYMAKER_START };
  const aim = { x: start.x + drag.x, y: start.y + drag.y };
  let receiver = null;
  let bestError = Infinity;
  for (const teammate of scene.teammates) {
    const error = ((aim.x - teammate.x) / 35) ** 2 + ((aim.y - teammate.y) / 42) ** 2;
    if (error <= 1 && error < bestError) { receiver = { ...teammate }; bestError = error; }
  }
  let interceptor = null;
  if (receiver) {
    for (const defender of scene.defenders) {
      const closest = pointOnSegment(start, receiver, defender);
      if (closest.distanceSquared <= defender.radius ** 2 && (!interceptor || closest.t < interceptor.t)) {
        interceptor = { ...defender, ...closest };
      }
    }
  }
  const made = receiver !== null && interceptor === null;
  return {
    scene,
    start,
    hoop: { ...PLAYMAKER_HOOP },
    aim,
    target: receiver ? { x: receiver.x, y: receiver.y } : { x: clamp(aim.x, -40, 400), y: clamp(aim.y, 80, 500) },
    receiver,
    defenders: scene.defenders.map((defender) => ({ ...defender })),
    interceptPoint: interceptor ? { x: interceptor.x, y: interceptor.y } : null,
    interceptor,
    assisted: made,
    made,
    clean: made && bestError <= 0.16,
    precision: Number.isFinite(bestError) ? Math.max(0, 1 - bestError) : 0,
    outcome: made ? 'assist' : interceptor ? 'intercepted' : 'off_target',
    reason: made ? 'assist' : interceptor ? 'intercepted' : 'off_target',
    flightMs: PLAYMAKER_ANIMATION_MS,
  };
}

/**
 * Samples pass -> reception -> drive -> dunk -> net, or a loose ball.
 * Returns { ball:{x,y,scale,rotation,visible}, runner, phase, netHit }.
 * Flat ball fields and receiverPose are also provided for SVG convenience.
 * receiverPose is null before reception; afterward it contains x,y,scale
 * for moving the selected teammate while other scene players stay frozen.
 */
export function samplePlaymakerPass(pass, progress = 0) {
  if (!pass || !isPoint(pass.start) || !isPoint(pass.target)) return null;
  const p = clamp(Number.isFinite(progress) ? progress : 0, 0, 1);
  const base = { rotation: p * 500, visible: p < 1, receiverPose: null, netHit: false };
  const passEnd = 0.34;
  const target = pass.interceptor || pass.target;
  if (p <= passEnd) {
    const t = p / passEnd;
    return withBallAndRunner({
      ...base,
      x: lerp(pass.start.x, target.x, t),
      y: lerp(pass.start.y, target.y, t) - Math.sin(Math.PI * t) * 12,
      scale: lerp(1, 0.65, t),
      phase: 'pass',
    });
  }
  if (!pass.made) {
    const t = (p - passEnd) / (1 - passEnd);
    const direction = target.x < pass.start.x ? -1 : 1;
    return withBallAndRunner({
      ...base,
      x: target.x + direction * 55 * t,
      y: target.y - Math.sin(Math.PI * t) * 30 + 150 * t * t,
      scale: lerp(0.65, 0.8, t),
      phase: pass.interceptor ? 'intercepted' : 'miss',
    });
  }

  const driveEnd = 0.72;
  const dunkEnd = 0.88;
  const driveTarget = { x: pass.hoop.x, y: pass.hoop.y + 40 };
  if (p <= driveEnd) {
    const t = (p - passEnd) / (driveEnd - passEnd);
    const x = lerp(pass.receiver.x, driveTarget.x, t);
    const y = lerp(pass.receiver.y, driveTarget.y, t);
    return withBallAndRunner({
      ...base,
      x,
      y: y + Math.abs(Math.sin(t * Math.PI * 3)) * 13,
      scale: lerp(0.65, 0.46, t),
      phase: 'drive',
      receiverPose: { x, y, scale: lerp(1, 0.82, t) },
    });
  }
  if (p <= dunkEnd) {
    const t = (p - driveEnd) / (dunkEnd - driveEnd);
    return withBallAndRunner({
      ...base,
      x: pass.hoop.x,
      y: lerp(driveTarget.y, pass.hoop.y, t) - Math.sin(Math.PI * t) * 23,
      scale: lerp(0.46, 0.36, t),
      phase: 'dunk',
      receiverPose: { x: driveTarget.x, y: driveTarget.y - Math.sin(Math.PI * t) * 16, scale: 0.82, jump: Math.sin(Math.PI * t) * 16 },
    });
  }
  const t = (p - dunkEnd) / (1 - dunkEnd);
  return withBallAndRunner({
    ...base,
    x: pass.hoop.x,
    y: pass.hoop.y + 54 * t * t,
    scale: lerp(0.36, 0.4, t),
    phase: 'net',
    receiverPose: { ...driveTarget, scale: 0.82 },
    netHit: true,
  });
}

function withBallAndRunner(pose) {
  return {
    ...pose,
    ball: { x: pose.x, y: pose.y, scale: pose.scale, rotation: pose.rotation, visible: pose.visible },
    runner: pose.receiverPose ? { jump: 0, ...pose.receiverPose } : null,
  };
}
