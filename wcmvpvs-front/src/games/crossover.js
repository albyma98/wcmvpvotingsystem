/** Crossover Rush: deterministic gesture rules shared with the reward API. */
export const CROSSOVER_WORLD = Object.freeze({ width: 360, height: 560 });
export const CROSSOVER_MAX_DUEL_MS = 6000;
export const CROSSOVER_ANIMATION_MS = 1600;
export const CROSSOVER_EPSILON = 1e-7;

const WINDOWS = [1000, 850, 700];
const DELAYS = [300, 250, 200];
const EPS = CROSSOVER_EPSILON;
const clamp = (value, min, max) => Math.min(max, Math.max(min, value));
const lerp = (from, to, amount) => from + (to - from) * amount;

function validSamples(samples, minimum = 2) {
  if (!Array.isArray(samples) || samples.length < minimum || samples.length > 128) return false;
  if (samples[0]?.t_ms !== 0 || samples[0]?.x !== 0) return false;
  let previous = -1;
  for (const sample of samples) {
    if (!Number.isInteger(sample?.t_ms) || sample.t_ms < 0 || sample.t_ms > CROSSOVER_MAX_DUEL_MS || sample.t_ms <= previous || !Number.isFinite(sample?.x) || sample.x < -1 || sample.x > 1) return false;
    previous = sample.t_ms;
  }
  return true;
}

function traceX(samples, time) {
  if (time <= 0) return 0;
  for (let i = 1; i < samples.length; i += 1) {
    const current = samples[i];
    if (time <= current.t_ms) {
      const previous = samples[i - 1];
      return lerp(previous.x, current.x, (time - previous.t_ms) / (current.t_ms - previous.t_ms));
    }
  }
  return samples.at(-1).x;
}

/** Interpolated intervals continuously beyond +.45 or -.45, in time order. */
function baitIntervals(samples) {
  const intervals = [];
  for (const side of [-1, 1]) {
    let latest = null;
    for (let i = 1; i < samples.length; i += 1) {
      const a = samples[i - 1];
      const b = samples[i];
      const ax = a.x * side;
      const bx = b.x * side;
      if (ax < 0.45 && bx < 0.45) continue;
      const crossing = ax === bx ? a.t_ms : a.t_ms + (0.45 - ax) / (bx - ax) * (b.t_ms - a.t_ms);
      const start = ax >= 0.45 ? a.t_ms : crossing;
      const end = bx >= 0.45 ? b.t_ms : crossing;
      if (latest && start <= latest.end + EPS) latest.end = end;
      else { latest = { side, start, end }; intervals.push(latest); }
    }
  }
  return intervals.filter((interval) => interval.end - interval.start + EPS >= 180).sort((a, b) => a.end - b.end);
}

/** First exact crossing of the opposite .65 threshold after leaving a bait. */
function oppositeCrossing(samples, interval) {
  for (let i = 1; i < samples.length; i += 1) {
    const a = samples[i - 1];
    const b = samples[i];
    if (b.t_ms < interval.end || b.x * interval.side > -0.65) continue;
    const crossing = a.x * interval.side <= -0.65
      ? a.t_ms
      : a.t_ms + (-0.65 - a.x * interval.side) / ((b.x - a.x) * interval.side) * (b.t_ms - a.t_ms);
    if (crossing + EPS >= interval.end) return Math.max(crossing, interval.end);
  }
  return null;
}

/**
 * Returns null for an invalid trace/duel. Valid input returns:
 * {won,clean,quality,reason,baitSide,exitSide,cutAtMs,playerX,defenderX}.
 * Side/position values are normalized to [-1,1]. Successful reason is beat;
 * failure priority is blocked > too_slow > no_crossover > no_feint.
 * The first successful crossing wins, even after a previously failed feint.
 * Rewards are exclusively confirmed by the server, one coin per won duel.
 */
export function evaluateCrossoverDuel({ duelIndex = 0, samples } = {}) {
  if (!Number.isInteger(duelIndex) || duelIndex < 0 || duelIndex > 2 || !validSamples(samples)) return null;
  const delay = DELAYS[duelIndex];
  const window = WINDOWS[duelIndex];
  const end = samples.at(-1).t_ms;
  const failure = (reason, interval = null, cutAtMs = end) => ({
    won: false, clean: false, quality: 0, reason,
    baitSide: interval?.side || 0,
    exitSide: interval ? -interval.side : Math.sign(samples.at(-1).x),
    cutAtMs,
    playerX: traceX(samples, cutAtMs),
    defenderX: 0.9 * traceX(samples, Math.max(0, cutAtMs - delay)),
  });
  let bestFailure = failure('no_feint');
  const failures = { no_feint: 0, no_crossover: 1, too_slow: 2, blocked: 3 };
  const winners = [];
  for (const interval of baitIntervals(samples)) {
    const cutAtMs = oppositeCrossing(samples, interval);
    let candidate;
    if (cutAtMs === null) candidate = failure('no_crossover', interval);
    else {
      const reversalMs = cutAtMs - interval.end;
      if (reversalMs > window + EPS) candidate = failure('too_slow', interval, cutAtMs);
      else {
        const playerX = -interval.side * 0.65;
        const defenderX = 0.9 * traceX(samples, Math.max(0, cutAtMs - delay));
        const gap = Math.abs(playerX - defenderX);
        if (gap + EPS < 0.75) candidate = failure('blocked', interval, cutAtMs);
        else {
          const quality = Math.round(clamp(60 + (gap - 0.75) * 25 + (1 - reversalMs / window) * 20, 0, 100));
          winners.push({ won: true, clean: quality >= 90, quality, reason: 'beat', baitSide: interval.side, exitSide: -interval.side, cutAtMs, playerX, defenderX });
        }
      }
    }
    if (candidate && failures[candidate.reason] > failures[bestFailure.reason]) bestFailure = candidate;
  }
  return winners.sort((a, b) => a.cutAtMs - b.cutAtMs)[0] || bestFailure;
}

/** Live normalized positions; holding still keeps the last recorded position. */
export function getCrossoverScene({ duelIndex = 0, samples = [{ t_ms: 0, x: 0 }], elapsedMs = 0 } = {}) {
  const duel = Number.isInteger(duelIndex) ? clamp(duelIndex, 0, 2) : 0;
  const trace = validSamples(samples, 1) ? samples : [{ t_ms: 0, x: 0 }];
  const time = clamp(Number.isFinite(elapsedMs) ? elapsedMs : 0, 0, CROSSOVER_MAX_DUEL_MS);
  const playerX = traceX(trace, time);
  const defenderX = 0.9 * traceX(trace, Math.max(0, time - DELAYS[duel]));
  const liveTrace = trace.filter((sample) => sample.t_ms <= time);
  if (liveTrace.at(-1).t_ms < time) liveTrace.push({ t_ms: time, x: playerX });
  const feintReady = Math.abs(playerX) >= 0.45 && baitIntervals(liveTrace).some((interval) => interval.side === Math.sign(playerX) && interval.end + EPS >= time);
  return { playerX, defenderX, gap: Math.abs(playerX - defenderX), delayMs: DELAYS[duel], reversalWindowMs: WINDOWS[duel], feintReady };
}

/**
 * Result animation poses in world coordinates. Player/defender y are the
 * athlete torso origins; ball ends at the hoop (180,112). Progress is 0..1.
 */
export function sampleCrossoverFinish(result, progress = 0) {
  if (!result || !Number.isFinite(result.playerX) || !Number.isFinite(result.defenderX)) return null;
  const p = clamp(Number.isFinite(progress) ? progress : 0, 0, 1);
  const side = result.exitSide || Math.sign(result.playerX) || 1;
  const startX = 180 + result.playerX * 80;
  const ballSide = Math.sign(result.playerX) || 1;
  const defender = { x: 180 + result.defenderX * 80, y: 286 + (result.won ? 15 * Math.sin(p * Math.PI) : 0), scale: 0.9, lean: result.won ? -side * 23 * Math.sin(p * Math.PI) : 0 };
  if (!result.won) {
    return {
      player: { x: startX, y: 414, scale: 1, lean: side * Math.sin(p * Math.PI) * 9 },
      defender,
      ball: { x: startX + ballSide * 26 + side * 52 * p, y: 455 - Math.sin(p * Math.PI) * 40 + 70 * p * p, scale: 0.8, rotation: p * 420, visible: p < 1 },
      phase: 'blocked', netHit: false,
    };
  }
  let player;
  let ball;
  let phase;
  if (p <= 0.2) {
    const t = p / 0.2;
    player = { x: lerp(startX, 180 + side * 105, t), y: lerp(414, 330, t), scale: lerp(1, 0.94, t), lean: side * 15 * Math.sin(t * Math.PI) };
    ball = { x: player.x + lerp(ballSide * 26, 0, t) + side * 15 * Math.sin(t * Math.PI), y: player.y + 41 + Math.abs(Math.sin(t * Math.PI * 2)) * 25, scale: lerp(0.8, 0.7, t), rotation: p * 700, visible: true };
    phase = 'crossover';
  } else if (p <= 0.68) {
    const t = (p - 0.2) / 0.48;
    player = { x: lerp(180 + side * 105, 180, t), y: lerp(330, 150, t), scale: lerp(0.94, 0.78, t), lean: -side * 8 * Math.sin(t * Math.PI) };
    ball = { x: player.x, y: player.y + 41 + Math.abs(Math.sin(t * Math.PI * 3)) * 20, scale: lerp(0.7, 0.46, t), rotation: p * 700, visible: true };
    phase = 'drive';
  } else if (p <= 0.88) {
    const t = (p - 0.68) / 0.2;
    player = { x: 180, y: 150 - Math.sin(t * Math.PI) * 28, scale: 0.78, lean: side * 4 * Math.sin(t * Math.PI) };
    ball = { x: 180, y: lerp(191, 112, t) - Math.sin(t * Math.PI) * 25, scale: lerp(0.46, 0.36, t), rotation: p * 700, visible: true };
    phase = 'dunk';
  } else {
    const t = (p - 0.88) / 0.12;
    player = { x: 180, y: lerp(150, 160, t), scale: 0.78, lean: 0 };
    ball = { x: 180, y: 112 + 58 * t * t, scale: lerp(0.36, 0.4, t), rotation: p * 700, visible: p < 1 };
    phase = 'net';
  }
  return { player, defender, ball, phase, netHit: phase === 'net' };
}
