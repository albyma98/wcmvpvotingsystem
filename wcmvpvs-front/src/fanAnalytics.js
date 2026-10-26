// Explicit event/property contract for /:slug/new. Never accept arbitrary DOM/form data.
export const FAN_ANALYTICS_SCHEMA_VERSION = 1;
export const FAN_ANALYTICS_EVENTS = new Set([
  'page_view', 'active_time', 'vote_completed', 'game_started', 'game_completed',
  'registration_started', 'registration_completed', 'redemption_completed',
  'sponsor_view', 'sponsor_click', 'sponsored_game_view', 'shop_view', 'feedback_completed',
  'home_loading_started', 'home_ready', 'home_load_failed', 'visit_started', 'visit_returned',
  'page_leave', 'page_hidden', 'page_visible', 'section_opened', 'section_closed', 'cta_clicked', 'cta_viewed',
  'vote_opened', 'vote_player_selected', 'vote_submitted', 'vote_changed', 'vote_failed',
  'game_opened', 'game_closed', 'game_abandoned', 'game_reward_claimed', 'game_reward_failed',
  'registration_dismissed', 'registration_step', 'registration_failed',
  'login_started', 'login_completed', 'login_failed', 'shop_item_viewed',
  'redemption_started', 'redemption_failed', 'frame_equipped', 'frame_equip_failed',
  'feedback_opened', 'feedback_step', 'feedback_dismissed', 'feedback_failed',
  'lottery_shortcut_view', 'lottery_shortcut_clicked', 'lottery_qr_view',
  'leaderboard_loaded', 'leaderboard_failed', 'profile_loaded', 'profile_failed',
  'profile_edit_started', 'profile_updated', 'profile_update_failed', 'wallet_loaded', 'wallet_failed',
  'vote_reward_claimed', 'vote_reward_failed',
]);

const sections = ['home', 'mvp', 'vote-result', 'earn', 'wallet', 'spend', 'collectible', 'leaderboard', 'match-summary', 'profile', 'info'];
const reasons = ['unknown', 'network', 'timeout', 'server_error', 'unavailable', 'validation', 'unauthorized', 'forbidden', 'not_found', 'conflict', 'insufficient_coins', 'closed', 'already_completed', 'cancelled', 'user_exit', 'load_failed', 'save_failed', 'invalid_response', 'registration_required', 'otp_requested', 'otp_invalid', 'otp_expired', 'rate_limited', 'storage_unavailable', 'not_ready', 'request_failed', 'not_allowed', 'already_owned', 'session_expired', 'user_closed', 'navigation'];
const enums = {
  section: sections, from_section: sections,
  placement: ['home_featured', 'game_catalog', 'home_grid', 'topbar', 'profile', 'lottery_shortcut', 'feedback_home', 'profile_ticket', 'info', 'home_partner', 'sponsor_rush_entry', 'home', 'game'],
  trigger: ['profile_overlay', 'wallet', 'spend_redeem', 'after_vote', 'leaderboard', 'home', 'shortcut', 'cta', 'manual', 'profile', 'shop', 'mvp', 'feedback', 'lottery'],
  mode: ['singleplayer', 'live', 'register', 'login'],
  source: ['arena', 'remote', 'unknown'],
  outcome: ['success', 'failure', 'win', 'lose', 'draw', 'forfeit_win', 'forfeit_lose', 'completed', 'abandoned', 'selected', 'advanced', 'back', 'skipped', 'started', 'otp_requested', 'otp_resent', 'otp_verified', 'details_submitted'],
  reason: reasons, error_code: reasons,
  phase: ['loading', 'ready', 'open', 'closed', 'active', 'concluded', 'request', 'confirmed'],
  cta: ['mvp', 'earn', 'shop', 'leaderboard', 'profile', 'wallet', 'feedback', 'lottery', 'info', 'collectible', 'vote_bonus_retry'],
  game_id: ['reaction', 'quiz', 'tap', 'tap-volley', 'sponsor-rush', 'memory-flash', 'basketball', 'playmaker', 'crossover', 'volley-rush', 'setter-pro', 'monster-block'],
  item_key: ['club_frame', 'match_card'],
  experience: ['awesome', 'good', 'ok', 'bad', 'very_easy', 'easy', 'neutral', 'difficult'],
  perks_interest: ['yes', 'maybe', 'little', 'no'],
  sport: ['', 'Calcio', 'Pallacanestro', 'Pallavolo'],
  device_type: ['mobile', 'tablet', 'desktop', 'unknown'],
  browser_family: ['Chrome', 'Edge', 'Firefox', 'Safari', 'Other', 'unknown'],
};
const booleans = new Set(['vote_available', 'games_available', 'feedback_available', 'registered', 'has_voted', 'is_concluded', 'votes_closed', 'is_change', 'has_ticket', 'success']);
const identifiers = new Set(['sponsor_id', 'player_id', 'previous_player_id']);
const numbers = new Set(['score', 'taps', 'count', 'amount', 'coins', 'coins_earned', 'visit_count']);
const milliseconds = new Set(['duration_ms', 'elapsed_ms', 'latency_ms', 'time_to_first_action_ms']);

export function sanitizeFanAnalyticsMetadata(input) {
  if (!input || typeof input !== 'object' || Array.isArray(input)) return {};
  const output = {};
  for (const [key, value] of Object.entries(input)) {
    if (enums[key]?.includes(value)) output[key] = value;
    else if (booleans.has(key) && typeof value === 'boolean') output[key] = value;
    else if (identifiers.has(key) && Number.isSafeInteger(value) && value > 0 && value <= 1e12) output[key] = value;
    else if (numbers.has(key) && Number.isFinite(value) && value >= 0 && value <= 1e9) output[key] = value;
    else if (milliseconds.has(key) && Number.isFinite(value) && value >= 0 && value <= 86400000) output[key] = value;
    else if (key === 'seconds' && Number.isFinite(value) && value >= 0 && value <= 15) output[key] = value;
    else if ((key === 'active_seconds' || key === 'visible_seconds') && Number.isFinite(value) && value >= 0 && value <= 86400) output[key] = value;
    else if (key === 'status_code' && Number.isInteger(value) && value >= 100 && value <= 599) output[key] = value;
    else if (key === 'round_id' && typeof value === 'string' && /^[A-Za-z0-9_:-]{8,128}$/.test(value)) output[key] = value;
    else if ((key === 'started_at' || key === 'ended_at') && typeof value === 'string' && value.length <= 35 && /^\d{4}-\d{2}-\d{2}T/.test(value) && Number.isFinite(Date.parse(value))) output[key] = value;
    else if (key === 'schema_version' && value === FAN_ANALYTICS_SCHEMA_VERSION) output[key] = value;
    else if (key === 'app_version' && typeof value === 'string' && /^\d+\.\d+\.\d+(?:[-+][A-Za-z0-9.-]{1,30})?$/.test(value)) output[key] = value;
  }
  return output;
}
