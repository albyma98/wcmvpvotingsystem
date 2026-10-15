export const FIRST_FAN_GOAL = 25;
export const VOTE_REWARD = 10;
export const QUIZ_REWARD = 15;
export const FAN_BADGE_NAME = 'Tifoso Attivo';

const rewardKeys = ['vote', 'quiz', 'reaction', 'tap', 'sponsor-rush', 'memory-flash'];

export function journeyStorageKey(slug, eventId) {
  return `cento:journey:${encodeURIComponent(slug || '')}:${eventId}`;
}

export function normalizeJourney(raw) {
  const rewards = {};
  for (const key of rewardKeys) {
    const amount = raw?.rewards?.[key];
    if (Number.isFinite(amount) && amount > 0) rewards[key] = Math.floor(amount);
  }
  return { rewards, pendingVoteBonus: raw?.pendingVoteBonus === true };
}

export function journeyProgress(journey) {
  const { rewards } = normalizeJourney(journey);
  const earned = Object.values(rewards).reduce((sum, amount) => sum + amount, 0);
  return {
    earned,
    current: Math.min(FIRST_FAN_GOAL, earned),
    remaining: Math.max(0, FIRST_FAN_GOAL - earned),
    percent: Math.min(100, earned / FIRST_FAN_GOAL * 100),
    unlocked: earned >= FIRST_FAN_GOAL,
  };
}

// The caller records a reward only after the wallet save has succeeded.
export function recordJourneyReward(journey, key, amount) {
  const next = normalizeJourney(journey);
  if (!rewardKeys.includes(key) || next.rewards[key] || !Number.isFinite(amount) || amount <= 0) return next;
  next.rewards[key] = Math.floor(amount);
  if (key === 'vote') next.pendingVoteBonus = false;
  return next;
}
