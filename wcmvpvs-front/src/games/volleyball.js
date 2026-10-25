export const VOLLEYBALL_WORLD = Object.freeze({ width: 360, height: 560 });
export const VOLLEYBALL_START = Object.freeze({ x: 180, y: 466 });
export const VOLLEYBALL_MODES = Object.freeze({
  'volley-rush': { title: 'VOLLEY', emphasis: 'RUSH', eyebrow: 'UN GESTO. PALLA A TERRA.', action: 'SCHIACCIA', noun: 'SCHIACCIATE', success: 'PUNTO A TERRA!', rx: 48, ry: 40, timingWindow: .24, cycleMs: 1750, flightMs: 360, target: { x: 180, y: 205 } },
  'setter-pro': { title: 'SETTER', emphasis: 'PRO', eyebrow: 'UN TOCCO. L’ASSIST PERFETTO.', action: 'ALZA', noun: 'ALZATE', success: 'ALZATA PERFETTA!', rx: 34, ry: 30, timingWindow: .20, cycleMs: 1850, flightMs: 420, target: { x: 180, y: 135 } },
  'monster-block': { title: 'MONSTER', emphasis: 'BLOCK', eyebrow: 'LEGGI LA PALLA. TOCCA E MURA.', action: 'MURA', noun: 'MURI', success: 'MURO VINCENTE!', rx: 44, ry: 34, timingWindow: .16, cycleMs: 900, flightMs: 320, target: { x: 180, y: 275 } },
});

export const clamp = (value, min, max) => Math.max(min, Math.min(max, Number.isFinite(Number(value)) ? Number(value) : min));

export function volleyballTiming(elapsedMs, cycleMs) {
  const phase = Math.max(0, elapsedMs) / cycleMs % 2;
  return phase <= 1 ? phase : 2 - phase;
}

// A visual forecast only: receipts from the server are the sole source of rewards.
export function previewVolleyballAttempt(gameId, challenge, attempt) {
  const mode = VOLLEYBALL_MODES[gameId];
  if (!mode || !challenge?.target || !attempt) return { onTarget: false, inTime: false, blocked: false };
  const target = challenge.target;
  const onTarget = ((attempt.aim_x - target.x) / mode.rx) ** 2 + ((attempt.aim_y - target.y) / mode.ry) ** 2 <= 1 + 1e-10;
  const inTime = Math.abs(attempt.timing - (challenge.timing_target ?? .7)) <= mode.timingWindow + 1e-10;
  const netX = VOLLEYBALL_START.x + (attempt.aim_x - VOLLEYBALL_START.x) * (VOLLEYBALL_START.y - 285) / (VOLLEYBALL_START.y - attempt.aim_y);
  const blocked = gameId === 'volley-rush' && Number.isFinite(challenge.blocker?.x) && Math.abs(netX - challenge.blocker.x) <= 30;
  return { onTarget, inTime, blocked, netX };
}

export function sampleVolleyballFlight(gameId, attempt, progress, challenge) {
  const t = clamp(progress, 0, 1);
  const preview = previewVolleyballAttempt(gameId, challenge, attempt);
  if (gameId === 'monster-block') {
    const contact = { x: challenge?.target?.x ?? 180, y: challenge?.target?.y ?? 275 };
    const good = preview.onTarget && preview.inTime;
    return { x: contact.x + (good ? 40 : -22) * t, y: contact.y + (good ? -85 : 210) * t - Math.sin(t * Math.PI) * 34, scale: .8 - t * (good ? .3 : -.35), rotation: t * 310, visible: t < 1 };
  }
  if (gameId === 'setter-pro') {
    const miss = preview.inTime ? 0 : 48;
    return { x: 180 + (attempt.aim_x - 180) * t, y: 466 + (attempt.aim_y + miss - 466) * t - Math.sin(Math.PI * t) * 140, scale: 1 - t * .43, rotation: t * 190, visible: true };
  }
  const hitBlock = preview.blocked && t > .65;
  if (hitBlock) {
    const fall = (t - .65) / .35;
    return { x: preview.netX + (180 - preview.netX) * fall * .5, y: 265 + fall * 235, scale: .65 + fall * .3, rotation: t * 500, visible: true };
  }
  // The ball rises for the attack, clears the net, then drives into the far court.
  const split = .45;
  const shotT = Math.max(0, (t - split) / (1 - split));
  const peak = { x: 180 + (attempt.aim_x - 180) * .34, y: 98 };
  return t < split
    ? { x: 180 + (peak.x - 180) * t / split, y: 466 + (peak.y - 466) * Math.sin(t / split * Math.PI / 2), scale: 1 - t / split * .23, rotation: t * 390, visible: true }
    : { x: peak.x + (attempt.aim_x - peak.x) * shotT, y: peak.y + (attempt.aim_y + (preview.inTime ? 0 : 55) - peak.y) * shotT ** .75, scale: .77 - shotT * .23, rotation: t * 570, visible: true };
}
