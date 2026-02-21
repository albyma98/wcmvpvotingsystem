const EARN_COOLDOWN_PREFIX = 'earnCooldown:';

function getStorageKey(id) {
  return `${EARN_COOLDOWN_PREFIX}${id}`;
}

export function getEarnCooldownRemainingSeconds(id, now = Date.now()) {
  if (typeof window === 'undefined') {
    return 0;
  }

  const rawValue = window.localStorage.getItem(getStorageKey(id));
  const nextAvailableAt = Number.parseInt(rawValue || '', 10);
  if (!Number.isFinite(nextAvailableAt)) {
    return 0;
  }

  return Math.max(0, Math.ceil((nextAvailableAt - now) / 1000));
}

export function startEarnCooldown(id, cooldownSeconds, now = Date.now()) {
  if (typeof window === 'undefined') {
    return;
  }

  const safeCooldown = Math.max(0, Number(cooldownSeconds) || 0);
  const nextAvailableAt = now + safeCooldown * 1000;
  window.localStorage.setItem(getStorageKey(id), String(nextAvailableAt));
}
