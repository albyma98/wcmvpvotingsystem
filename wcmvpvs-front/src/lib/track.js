import { getPostHog, isPostHogReady } from './posthog';

export const EVENTS = Object.freeze({
  HOME_OPENED: 'home.opened',
  VOTE_OPENED: 'vote.opened',
  VOTE_PLAYER_SELECTED: 'vote.player_selected',
  VOTE_SUBMITTED: 'vote.submitted',
  FEEDBACK_OPENED: 'feedback.opened',
  FEEDBACK_SUBMITTED: 'feedback.submitted',
  COUPON_OPENED: 'coupon.opened',
  COUPON_REDEEMED: 'coupon.redeemed',
  SPONSOR_CLICKED: 'sponsor.clicked',
  GAME_STARTED: 'game.started',
  GAME_COMPLETED: 'game.completed',
  COINS_EARNED: 'coins.earned',
  COINS_SPENT: 'coins.spent',
  BAR_MENU_OPENED: 'bar.menu_opened',
  BAR_CHECKOUT_STARTED: 'bar.checkout_started',
  BAR_ORDER_COMPLETED: 'bar.order_completed',
  // Tornei (surface: 'tournament')
  TOURNAMENT_HOME_OPENED: 'tournament.home_opened',
  TOURNAMENT_TILE_SELECTED: 'tournament.tile_selected',
  TOURNAMENT_SECTION_VIEWED: 'tournament.section_viewed',
  TOURNAMENT_MVP_VOTED: 'tournament.mvp_voted',
  TOURNAMENT_SPONSOR_SHOWN: 'tournament.sponsor_strip_shown',
  TOURNAMENT_SPONSOR_CLICKED: 'tournament.sponsor_clicked',
  TOURNAMENT_GALLERY_PHOTO_OPENED: 'tournament.gallery_photo_opened',
  TOURNAMENT_GALLERY_UPLOAD: 'tournament.gallery_upload',
});

const SENSITIVE_KEY_PATTERNS = [
  /phone/i,
  /tel(?:efono)?/i,
  /mobile/i,
  /email/i,
  /mail$/i,
  /otp/i,
  /pin$/i,
  /password/i,
  /pwd/i,
  /token/i,
  /auth/i,
  /jwt/i,
  /secret/i,
  /api[_-]?key/i,
  /full[_-]?name/i,
  /first[_-]?name/i,
  /last[_-]?name/i,
  /surname/i,
  /given[_-]?name/i,
  /family[_-]?name/i,
  /tax[_-]?code/i,
  /codice[_-]?fiscale/i,
  /cf$/i,
  /card/i,
  /iban/i,
  /cvv/i,
  /pan$/i,
  /payment/i,
  /coupon[_-]?code/i,
  /redeem[_-]?code/i,
  /personal[_-]?code/i,
];

function isSensitiveKey(key) {
  if (typeof key !== 'string' || !key) {
    return false;
  }
  return SENSITIVE_KEY_PATTERNS.some((pattern) => pattern.test(key));
}

function sanitizeProps(props) {
  if (!props || typeof props !== 'object') {
    return {};
  }
  const out = {};
  for (const [key, value] of Object.entries(props)) {
    if (value === undefined) continue;
    if (isSensitiveKey(key)) {
      if (import.meta.env.DEV) {
        console.warn(`[track] dropped sensitive prop "${key}"`);
      }
      continue;
    }
    if (value && typeof value === 'object' && !Array.isArray(value)) {
      out[key] = sanitizeProps(value);
    } else {
      out[key] = value;
    }
  }
  return out;
}

function buildCommonProps() {
  const common = {
    app: 'arenaboostx',
    environment: import.meta.env.MODE,
    client_ts: new Date().toISOString(),
  };
  if (typeof window !== 'undefined' && window.location) {
    common.path = window.location.pathname;
    common.route = window.location.pathname + window.location.search;
  }
  return common;
}

export function track(eventName, props = {}) {
  const name = String(eventName || '').trim();
  if (!name) return;

  const ph = getPostHog();
  const safeProps = sanitizeProps(props);
  const payload = { ...buildCommonProps(), ...safeProps };

  if (import.meta.env.DEV) {
    console.log(`[track] ${name}`, payload);
  }

  if (!isPostHogReady() || !ph) {
    return;
  }

  try {
    ph.capture(name, payload);
  } catch (error) {
    if (import.meta.env.DEV) {
      console.warn('[track] capture failed', error);
    }
  }
}

export function identifyFan(distinctId, traits = {}) {
  const ph = getPostHog();
  if (!ph || !distinctId) return;
  try {
    ph.identify(String(distinctId), sanitizeProps(traits));
  } catch (error) {
    if (import.meta.env.DEV) {
      console.warn('[track] identify failed', error);
    }
  }
}

export function resetTracking() {
  const ph = getPostHog();
  if (!ph) return;
  try {
    ph.reset();
  } catch {
    /* noop */
  }
}
