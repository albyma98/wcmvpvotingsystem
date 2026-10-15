/**
 * Deterministic shot rules shared with the basketball reward API.
 * Coordinates are in a 360 x 560 SVG world, independent of screen size.
 * A drag aims at start + drag; its endpoint must land inside the hoop's
 * 30 x 40 aiming ellipse. The animation never decides whether coins are due.
 */
export const BASKETBALL_WORLD = Object.freeze({ width: 360, height: 560 });
export const BASKETBALL_START = Object.freeze({ x: 180, y: 466 });
export const BASKETBALL_HOOP = Object.freeze({ x: 180, y: 200 });
export const BASKETBALL_FLIGHT_MS = 1100;
export const BASKETBALL_MIN_DRAG = 42;

const HOOP_SEQUENCE = Object.freeze([
  BASKETBALL_HOOP,
  Object.freeze({ x: 120, y: 205 }),
  Object.freeze({ x: 240, y: 195 }),
  Object.freeze({ x: 150, y: 215 }),
  Object.freeze({ x: 225, y: 205 }),
  Object.freeze({ x: 105, y: 195 }),
  Object.freeze({ x: 255, y: 215 }),
  Object.freeze({ x: 180, y: 185 }),
]);

const clamp = (value, min, max) => Math.min(max, Math.max(min, value));
const lerp = (from, to, amount) => from + (to - from) * amount;
const isPoint = (point) => point != null && Number.isFinite(point.x) && Number.isFinite(point.y);

/** Returns a fresh target, so an animation cannot mutate the target sequence. */
export function basketballHoopForAttempt(attempt = 0) {
  const index = Number.isFinite(attempt) ? Math.max(0, Math.floor(attempt)) : 0;
  return { ...HOOP_SEQUENCE[index % HOOP_SEQUENCE.length] };
}

/**
 * Converts pointer positions in a rendered play area's CSS coordinates into
 * the world-space delta expected by createBasketballShot and the server.
 * Use the actual SVG drawing area (excluding any letterbox margins).
 */
export function basketballDragFromPoints({ start, end, width, height } = {}) {
  if (!isPoint(start) || !isPoint(end) || !Number.isFinite(width) || !Number.isFinite(height) || width <= 0 || height <= 0) return null;
  return {
    x: (end.x - start.x) * BASKETBALL_WORLD.width / width,
    y: (end.y - start.y) * BASKETBALL_WORLD.height / height,
  };
}

/**
 * Returns null for a cancelled/non-shot gesture. A valid shot contains:
 * { start, hoop, drag, aim, target, made, clean, quality, missKind,
 *   arcHeight, flightMs, direction }.
 * `aim` preserves the actual endpoint; `target` is the visual entry point.
 * Every made shot is worth exactly one coin, as confirmed by the server.
 */
export function createBasketballShot({ start = BASKETBALL_START, hoop = BASKETBALL_HOOP, drag, cancelled = false } = {}) {
  if (cancelled || !isPoint(start) || !isPoint(hoop) || !isPoint(drag)) return null;
  const length = Math.hypot(drag.x, drag.y);
  if (!Number.isFinite(length) || length < BASKETBALL_MIN_DRAG || drag.y > -30) return null;

  const aim = { x: start.x + drag.x, y: start.y + drag.y };
  const errorX = (aim.x - hoop.x) / 30;
  const errorY = (aim.y - hoop.y) / 40;
  const error = errorX * errorX + errorY * errorY;
  const made = error <= 1;
  const clean = made && error <= 0.16;
  const target = made
    ? { ...hoop }
    : { x: clamp(aim.x, -80, BASKETBALL_WORLD.width + 80), y: clamp(aim.y, 55, 500) };

  return {
    start: { ...start },
    hoop: { ...hoop },
    drag: { ...drag },
    aim,
    target,
    made,
    clean,
    quality: made ? (clean ? 'swish' : 'rim') : 'miss',
    missKind: made ? null : Math.abs(errorX) > Math.abs(errorY) ? 'wide' : errorY > 0 ? 'short' : 'long',
    arcHeight: 118 + Math.min(54, Math.hypot(target.x - start.x, target.y - start.y) * 0.08),
    flightMs: BASKETBALL_FLIGHT_MS,
    direction: drag.x < 0 ? -1 : 1,
  };
}

/**
 * Samples a continuous pseudo-3D flight, then a net drop or missed-shot bounce.
 * Returns { x, y, scale, rotation, phase, visible, shadowScale }.
 * The ball reaches the hoop at 74% of the flight, descending and smaller.
 */
export function sampleBasketballShot(shot, progress = 0) {
  if (!shot || !isPoint(shot.start) || !isPoint(shot.target)) return null;
  const p = clamp(Number.isFinite(progress) ? progress : 0, 0, 1);
  const approachEnd = 0.74;
  const direction = shot.direction || 1;

  if (p <= approachEnd) {
    const t = p / approachEnd;
    return {
      x: lerp(shot.start.x, shot.target.x, t),
      y: lerp(shot.start.y, shot.target.y, t) - Math.sin(Math.PI * t) * shot.arcHeight,
      scale: lerp(1, 0.38, t),
      rotation: direction * 420 * p,
      phase: 'flight',
      visible: true,
      shadowScale: lerp(1, 0.28, t),
    };
  }

  const t = (p - approachEnd) / (1 - approachEnd);
  const made = Boolean(shot.made);
  const rimDirection = shot.aim.x < shot.hoop.x ? -1 : 1;
  return {
    x: shot.target.x + (made ? Math.sin(t * Math.PI) * (shot.clean ? 2 : 5) : rimDirection * (38 * t + 38 * t * t)),
    y: shot.target.y + (made ? 110 * t * t : -38 * Math.sin(t * Math.PI) + 150 * t * t),
    scale: lerp(0.38, made ? 0.42 : 0.52, t),
    rotation: direction * 420 * p,
    phase: made ? 'net' : 'miss',
    visible: p < 1,
    shadowScale: lerp(0.28, 0.4, t),
  };
}
