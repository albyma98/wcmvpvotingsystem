import axios from 'axios';
import { getOrCreateDeviceId } from './deviceId';

const ensureApiPath = (baseUrl: string) => {
  const sanitized = baseUrl.replace(/\/+$/, '');
  if (sanitized === '' || sanitized === '.') {
    return '/api';
  }

  if (/\/api$/i.test(sanitized)) {
    return sanitized;
  }

  return `${sanitized}/api`;
};

const resolveApiBaseUrl = () => {
  const envUrl = import.meta.env.VITE_API_BASE_URL?.trim();
  const resolveFromWindow = () => {
    if (typeof window === 'undefined') {
      return { hostname: 'localhost', protocol: 'http:', port: '' };
    }
    return {
      hostname: window.location.hostname || 'localhost',
      protocol: window.location.protocol || 'http:',
      port: window.location.port || '',
    };
  };

  if (envUrl) {
    if (envUrl.toLowerCase() === 'auto') {
      // fall through to auto-detected host/port
    } else if (envUrl.includes('{host}')) {
      const { hostname } = resolveFromWindow();
      return envUrl.replace('{host}', hostname).replace(/\/+$/, '');
    } else {
      return envUrl.replace(/\/+$/, '');
    }
  }

  const envPortRaw = import.meta.env.VITE_API_PORT;
  const envPort = typeof envPortRaw === 'number' ? envPortRaw.toString() : envPortRaw?.toString().trim();
  const { hostname, protocol, port: windowPort } = resolveFromWindow();

  if (envPort) {
    const sanitizedPort = envPort.replace(/^:/, '');
    const targetHost = hostname || 'localhost';
    return ensureApiPath(`${protocol}//${targetHost}:${sanitizedPort}`);
  }

  if (import.meta.env.DEV) {
    const devHost = hostname || 'localhost';
    return ensureApiPath(`${protocol}//${devHost}:3000`);
  }

  const originPort = windowPort ? `:${windowPort}` : '';
  return ensureApiPath(`${protocol}//${hostname}${originPort}`);
};

function detectOrganizationSlug(pathname: string | undefined, search?: string) {
  const reservedPrefixes = new Set(['admin', 'shop', 'lottery', 'welcome', 'partner']);

  if (pathname) {
    const segments = pathname
      .split('/')
      .map((part) => part.trim())
      .filter(Boolean);

    if (segments.length) {
      if (segments[0].toLowerCase() === 'newui') {
        return segments[1] || '';
      }

      if (segments[segments.length - 1].toLowerCase() === 'newui') {
        return segments[0] || '';
      }

      const first = segments[0].toLowerCase();
      if (!reservedPrefixes.has(first)) {
        return segments[0];
      }
    }
  }

  const params = new URLSearchParams(search || '');
  const slugFromQuery =
    params.get('organization_slug') || params.get('org') || params.get('organization') || '';
  return slugFromQuery.trim();
}

function getOrganizationSlugFromLocation() {
  if (typeof window === 'undefined') {
    return '';
  }
  return detectOrganizationSlug(window.location?.pathname || '', window.location?.search || '');
}

export function getOrganizationSlug(): string {
  return getOrganizationSlugFromLocation();
}

export const apiClient = axios.create({
  baseURL: resolveApiBaseUrl(),
});

apiClient.interceptors.request.use((config) => {
  const orgSlug = getOrganizationSlugFromLocation();
  if (orgSlug) {
    config.headers = config.headers || {};
    config.headers['X-Organization-Slug'] = orgSlug;
  }
  return config;
});

export function resolveApiUrl(path: string) {
  if (!path) {
    return '';
  }

  if (/^https?:\/\//i.test(path)) {
    return path;
  }

  const sanitizedPath = path.startsWith('/') ? path : `/${path}`;
  const baseURL = apiClient.defaults?.baseURL;

  const joinUrl = (base: string) => {
    const normalizedBase = base.replace(/\/+$/, '');
    return `${normalizedBase}${sanitizedPath}`;
  };

  if (typeof baseURL === 'string' && baseURL) {
    if (/^https?:\/\//i.test(baseURL)) {
      return joinUrl(baseURL);
    }

    if (baseURL.startsWith('/')) {
      if (typeof window !== 'undefined' && window.location?.origin) {
        return joinUrl(`${window.location.origin}${baseURL}`);
      }
      return joinUrl(baseURL);
    }

    try {
      return new URL(sanitizedPath, baseURL).toString();
    } catch (error) {
      // ignore and fall back
    }
  }

  if (typeof window !== 'undefined' && window.location?.origin) {
    return `${window.location.origin}${sanitizedPath}`;
  }

  return sanitizedPath;
}

export function sendJsonBeacon(path: string, payload: Record<string, unknown> = {}) {
  if (!path) {
    return Promise.resolve();
  }

  const url = resolveApiUrl(path);
  const body = JSON.stringify(payload);

  if (typeof navigator !== 'undefined' && typeof navigator.sendBeacon === 'function') {
    try {
      const blob = new Blob([body], { type: 'application/json' });
      if (navigator.sendBeacon(url, blob)) {
        return Promise.resolve();
      }
    } catch (error) {
      // fall back to fetch
    }
  }

  if (typeof fetch === 'function') {
    return fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body,
      keepalive: true,
    }).then(() => {});
  }

  return apiClient.post(path, payload).then(() => {});
}

export function resolveStaticAssetUrl(path: string) {
  if (!path) {
    return '';
  }

  if (/^https?:\/\//i.test(path)) {
    return path;
  }

  const sanitizedPath = path.startsWith('/') ? path : `/${path}`;

  if (typeof window !== 'undefined' && window.location?.origin) {
    return `${window.location.origin}${sanitizedPath}`;
  }

  return sanitizedPath;
}

export function getDeviceHeaders() {
  const deviceId = getOrCreateDeviceId();
  return deviceId ? { 'X-Device-ID': deviceId } : {};
}

export async function submitContactChance({
  eventId,
  contactValue,
  contactType,
  marketingConsent,
}: {
  eventId: number;
  contactValue: string;
  contactType: string;
  marketingConsent: boolean;
}) {
  const payload = {
    event_id: eventId,
    device_id: getOrCreateDeviceId(),
    contact_value: contactValue,
    contact_type: contactType,
    marketing_consent: marketingConsent,
    timestamp: new Date().toISOString(),
  };

  try {
    const { data } = await apiClient.post(`/events/${eventId}/contacts`, payload);
    return { ok: true, data };
  } catch (error) {
    const status = axios.isAxiosError(error) ? error.response?.status : undefined;
    const message = axios.isAxiosError(error) ? error.response?.data?.message : undefined;
    const data = axios.isAxiosError(error) ? error.response?.data : undefined;
    return { ok: false, status, message, data, error };
  }
}

export async function fetchEventCoupons(eventId: number) {
  if (!eventId) {
    return { ok: false, status: 400, message: "Evento non valido" };
  }

  try {
    const { data } = await apiClient.get(`/events/${eventId}/coupons`, {
      headers: getDeviceHeaders(),
    });

    return { ok: true, coupons: Array.isArray(data) ? data : [] };
  } catch (error) {
    console.error("fetchEventCoupons error", error);
    const status = axios.isAxiosError(error) ? error.response?.status : undefined;
    const message = axios.isAxiosError(error) ? error.response?.data?.message : undefined;
    return { ok: false, status, message, error };
  }
}

export async function claimCoupon(
  couponId: number,
  payload: { userId?: number; matchId?: number } = {},
) {
  if (!couponId) {
    return { ok: false, status: 400, message: "Coupon non valido" };
  }

  try {
    const { data } = await apiClient.post(
      `/coupons/${couponId}/claim`,
      {
        user_id: payload.userId,
        match_id: payload.matchId,
      },
      { headers: getDeviceHeaders() },
    );

    return { ok: true, claim: data };
  } catch (error) {
    console.error("claimCoupon error", error);
    const status = axios.isAxiosError(error) ? error.response?.status : undefined;
    const message = axios.isAxiosError(error) ? error.response?.data?.message : undefined;
    return { ok: false, status, message, error };
  }
}

export async function fetchContactBonuses(eventId: number) {
  if (!eventId) {
    return { ok: false, status: 400 };
  }

  try {
    const { data } = await apiClient.get(`/events/${eventId}/contacts`, {
      headers: getDeviceHeaders(),
    });
    return { ok: true, bonuses: data?.bonuses ?? [] };
  } catch (error) {
    const status = axios.isAxiosError(error) ? error.response?.status : undefined;
    const message = axios.isAxiosError(error) ? error.response?.data?.message : undefined;
    return { ok: false, status, message, error };
  }
}

export async function vote({ eventId, playerId }) {
  try {
    const { data: voteData } = await apiClient.post('/vote', {
      player_id: playerId,
      event_id: eventId,
      device_id: getOrCreateDeviceId(),
    });

    return { ok: true, vote: voteData, message: voteData?.message };
  } catch (error) {
    console.error('vote api error', error);
    const status = axios.isAxiosError(error) ? error.response?.status : undefined;
    const message = axios.isAxiosError(error) ? error.response?.data?.message : undefined;
    return { ok: false, error, status, message };
  }
}

export async function validateTicketStatus({ eventId, code, signature }) {
  try {
    const params = new URLSearchParams();
    if (eventId) {
      params.set('e', String(eventId));
    }
    if (code) {
      params.set('c', code);
    }
    if (signature) {
      params.set('s', signature);
    }

    const { data } = await apiClient.get(`/tickets/validate?${params.toString()}`);
    return { ok: true, data };
  } catch (error) {
    const responseError = error?.response?.data?.error;
    return { ok: false, error: responseError || 'unknown_error', details: error };
  }
}

export async function fetchVoteStatus(eventId: number) {
  if (!eventId) {
    return { ok: true, hasVoted: false };
  }

  const headers = getDeviceHeaders();
  if (!headers['X-Device-ID']) {
    return { ok: false, error: new Error('missing_device_id') };
  }

  try {
    const { data } = await apiClient.get(`/events/${eventId}/vote-status`, {
      headers,
    });
    return {
      ok: true,
      hasVoted: Boolean(data?.has_voted),
      playerId: typeof data?.player_id === 'number' ? data.player_id : undefined,
    };
  } catch (error) {
    return { ok: false, error };
  }
}

export async function fetchLiveVoteSummary(eventId: number) {
  if (!eventId) {
    return { ok: false, error: new Error('invalid_event_id') };
  }

  try {
    const { data } = await apiClient.get(`/events/${eventId}/votes/live`);
    return { ok: true, data };
  } catch (error) {
    return { ok: false, error };
  }
}

export async function fetchMySelfie(eventId: number) {
  if (!eventId) {
    return { ok: true, selfie: null };
  }
  const headers = getDeviceHeaders();
  if (!headers['X-Device-ID']) {
    return { ok: false, error: new Error('missing_device_id') };
  }

  try {
    const { data } = await apiClient.get(`/events/${eventId}/selfies/me`, {
      headers,
    });
    return { ok: true, selfie: data };
  } catch (error) {
    if (axios.isAxiosError(error) && error.response?.status === 404) {
      return { ok: true, selfie: null };
    }
    return { ok: false, error };
  }
}

export async function fetchReactionTestStatus(eventId: number) {
  if (!eventId) {
    return { ok: false, error: new Error('missing_event_id') };
  }

  const headers = getDeviceHeaders();
  if (!headers['X-Device-ID']) {
    return { ok: false, error: new Error('missing_device_id') };
  }

  try {
    const { data } = await apiClient.get(`/events/${eventId}/reaction-test`, {
      headers,
    });
    return { ok: true, data };
  } catch (error) {
    return { ok: false, error };
  }
}

export async function submitReactionTestResult(eventId: number, reactionTimeMs: number) {
  if (!eventId) {
    return { ok: false, error: new Error('missing_event_id') };
  }

  const headers = getDeviceHeaders();
  if (!headers['X-Device-ID']) {
    return { ok: false, error: new Error('missing_device_id') };
  }

  try {
    const { data } = await apiClient.post(
      `/events/${eventId}/reaction-test`,
      { reaction_time_ms: reactionTimeMs },
      { headers },
    );
    return { ok: true, data };
  } catch (error) {
    if (axios.isAxiosError(error)) {
      return {
        ok: false,
        status: error.response?.status,
        data: error.response?.data,
        error,
      };
    }
    return { ok: false, error };
  }
}


export async function fetchQuickQuiz(eventId: number) {
  if (!eventId) {
    return { ok: false, error: new Error('missing_event_id') };
  }
  try {
    const { data } = await apiClient.get(`/public/events/${eventId}/quiz`);
    return { ok: true, data };
  } catch (error) {
    return { ok: false, error };
  }
}

export async function submitQuickQuizAnswer(eventId: number, payload: { questionId: number; selectedIndex: number; responseMs: number; deviceId: string; }) {
  if (!eventId) {
    return { ok: false, error: new Error('missing_event_id') };
  }
  try {
    const { data } = await apiClient.post(`/public/events/${eventId}/quiz/answer`, {
      questionId: payload.questionId,
      selectedIndex: payload.selectedIndex,
      responseMs: payload.responseMs,
      deviceId: payload.deviceId,
    });
    return { ok: true, data };
  } catch (error) {
    return { ok: false, error };
  }
}
export async function completeMission(
  missionId: string,
  payload: Record<string, unknown> = {},
) {
  if (!missionId) {
    return { ok: false, error: new Error('missing_mission_id') };
  }

  const headers = getDeviceHeaders();
  if (!headers['X-Device-ID']) {
    return { ok: false, error: new Error('missing_device_id') };
  }

  try {
    const { data } = await apiClient.post(`/missions/${missionId}/complete`, payload, {
      headers,
    });
    return { ok: true, data };
  } catch (error) {
    if (axios.isAxiosError(error)) {
      return {
        ok: false,
        status: error.response?.status,
        data: error.response?.data,
        error,
      };
    }
    return { ok: false, error };
  }
}

export async function submitPerfectDigResult(
  eventId: number,
  payload: { attempts: number; successCount: number; reward: string },
) {
  if (!eventId) {
    return { ok: false, error: new Error('missing_event_id') };
  }

  const headers = getDeviceHeaders();
  if (!headers['X-Device-ID']) {
    return { ok: false, error: new Error('missing_device_id') };
  }

  try {
    const { data } = await apiClient.post(`/events/${eventId}/libero-reflex`, payload, {
      headers,
    });
    return { ok: true, data };
  } catch (error) {
    if (axios.isAxiosError(error)) {
      return {
        ok: false,
        status: error.response?.status,
        data: error.response?.data,
        error,
      };
    }
    return { ok: false, error };
  }
}

export async function uploadSelfie(
  eventId: number,
  {
    file,
    caption,
    imageBase64,
    acceptedImageTerms,
  }: { file?: File; caption?: string; imageBase64?: string; acceptedImageTerms?: boolean },
) {
  if (!eventId) {
    return { ok: false, error: new Error('missing_event_id') };
  }

  const headers = getDeviceHeaders();
  if (!headers['X-Device-ID']) {
    return { ok: false, error: new Error('missing_device_id') };
  }

  if (!acceptedImageTerms) {
    return { ok: false, error: new Error('missing_image_terms') };
  }

  try {
    if (file instanceof File) {
      const formData = new FormData();
      formData.append('image', file);
      if (caption) {
        formData.append('caption', caption);
      }
      formData.append('accepted_image_terms', 'true');

      const { data } = await apiClient.post(`/events/${eventId}/selfies`, formData, {
        headers,
      });
      return { ok: true, selfie: data };
    }

    if (typeof imageBase64 === 'string' && imageBase64.trim()) {
      const payload = {
        caption: caption ?? '',
        image_base64: imageBase64,
        accepted_image_terms: Boolean(acceptedImageTerms),
      };
      const { data } = await apiClient.post(`/events/${eventId}/selfies`, payload, {
        headers: { ...headers, 'Content-Type': 'application/json' },
      });
      return { ok: true, selfie: data };
    }

    return { ok: false, error: new Error('missing_image_data') };
  } catch (error) {
    return { ok: false, error };
  }
}

export async function listApprovedSelfies(eventId: number) {
  if (!eventId) {
    return { ok: true, selfies: [] };
  }
  try {
    const { data } = await apiClient.get(`/events/${eventId}/selfies/approved`);
    return { ok: true, selfies: Array.isArray(data) ? data : [] };
  } catch (error) {
    return { ok: false, error };
  }
}

type EventFeedbackPayload = {
  experience: string;
  team_spirit: string;
  perks_interest: string;
  mini_games_interest: string;
  suggestion?: string;
};

export async function submitEventFeedback(eventId: number, feedback: EventFeedbackPayload) {
  if (!eventId) {
    return { ok: false, error: new Error('missing_event_id') };
  }

  const payload: EventFeedbackPayload = {
    experience: feedback.experience,
    team_spirit: feedback.team_spirit,
    perks_interest: feedback.perks_interest,
    mini_games_interest: feedback.mini_games_interest,
    suggestion: (feedback.suggestion ?? '').slice(0, 80),
  };

  try {
    await apiClient.post(`/events/${eventId}/feedback`, payload);
    return { ok: true };
  } catch (error) {
    const status = axios.isAxiosError(error) ? error.response?.status : undefined;
    return { ok: false, error, status };
  }
}

export function trackPageEngagement(eventId: number, durationSeconds: number) {
  if (!eventId || durationSeconds <= 0) {
    return Promise.resolve();
  }

  const headers = getDeviceHeaders();
  const deviceId = headers['X-Device-ID'];

  return sendJsonBeacon(`/events/${eventId}/engagement`, {
    duration_seconds: durationSeconds,
    device_id: deviceId,
  });
}

export function trackPostVoteAction(eventId: number, action: string) {
  if (!eventId || !action?.trim()) {
    return Promise.resolve();
  }

  const headers = getDeviceHeaders();
  const deviceId = headers['X-Device-ID'];

  return sendJsonBeacon(`/events/${eventId}/post-vote-actions`, {
    action: action.trim(),
    device_id: deviceId,
  });
}

export async function fetchEventEngagement(eventId: number) {
  if (!eventId) {
    return { ok: false, error: new Error('missing_event_id') };
  }

  try {
    const { data } = await apiClient.get(`/events/${eventId}/engagement`);
    return { ok: true, data };
  } catch (error) {
    return { ok: false, error };
  }
}


export function getFanSessionToken() {
  if (typeof window === 'undefined') {
    return '';
  }
  return String(window.localStorage.getItem('fan:session_token') || '').trim();
}

export function getFanSessionHeaders() {
  const token = getFanSessionToken();
  return token ? { 'X-Fan-Session': token } : {};
}

export async function fetchFanProfile(eventId) {
  try {
    const { data } = await apiClient.get('/fan/me', {
      params: { event_id: eventId },
      headers: { ...getDeviceHeaders(), ...getFanSessionHeaders() },
    });
    return { ok: true, data };
  } catch (error) {
    return { ok: false, error };
  }
}

export async function registerFanProfile(payload) {
  try {
    const { data } = await apiClient.post('/fan/register', payload, {
      headers: { ...getDeviceHeaders(), ...getFanSessionHeaders() },
    });
    if (data?.session_token && typeof window !== 'undefined') {
      window.localStorage.setItem('fan:session_token', data.session_token);
    }
    return { ok: true, data };
  } catch (error) {
    const message = axios.isAxiosError(error) ? error.response?.data?.message : undefined;
    return { ok: false, error, message };
  }
}

export async function startPhoneAuth(phone: string, mode: 'register' | 'login' = 'register') {
  try {
    await apiClient.post('/auth/start', { phone, mode }, { headers: getDeviceHeaders() });
    return { ok: true };
  } catch (error) {
    const code = axios.isAxiosError(error) ? error.response?.data?.error : undefined;
    return { ok: false, error, code };
  }
}

export async function verifyPhoneAuth(phone: string, code: string) {
  try {
    const { data } = await apiClient.post('/auth/verify', { phone, code }, { headers: getDeviceHeaders() });
    const sessionToken = String(data?.token || data?.session_token || '').trim();
    if (sessionToken && typeof window !== 'undefined') {
      window.localStorage.setItem('fan:session_token', sessionToken);
    }
    return { ok: true, data };
  } catch (error) {
    const errCode = axios.isAxiosError(error) ? error.response?.data?.error : undefined;
    return { ok: false, error, code: errCode };
  }
}

export async function resendPhoneAuth(phone: string) {
  try {
    await apiClient.post('/auth/resend', { phone }, { headers: getDeviceHeaders() });
    return { ok: true };
  } catch (error) {
    const code = axios.isAxiosError(error) ? error.response?.data?.error : undefined;
    return { ok: false, error, code };
  }
}

export async function syncGuestCoins(eventId, coins) {
  try {
    await apiClient.post(`/events/${eventId}/guest-coins`, { coins }, {
      headers: { ...getDeviceHeaders(), ...getFanSessionHeaders() },
    });
    return { ok: true };
  } catch (error) {
    return { ok: false, error };
  }
}

export async function redeemFanReward(eventId, rewardKey, costCoins) {
  try {
    const { data } = await apiClient.post(`/events/${eventId}/rewards/redeem`, { reward_key: rewardKey, cost_coins: costCoins }, {
      headers: { ...getDeviceHeaders(), ...getFanSessionHeaders() },
    });
    return { ok: true, data };
  } catch (error) {
    const message = axios.isAxiosError(error) ? error.response?.data?.message : undefined;
    return { ok: false, error, message };
  }
}

export async function joinTapLiveQueue(eventId: number) {
  try {
    const { data } = await apiClient.post(`/events/${eventId}/tap-live/queue`, {}, {
      headers: { ...getDeviceHeaders(), ...getFanSessionHeaders() },
    });
    return { ok: true, data };
  } catch (error) {
    const message = axios.isAxiosError(error) ? error.response?.data?.message : undefined;
    const status = axios.isAxiosError(error) ? error.response?.status : undefined;
    return { ok: false, status, message, error };
  }
}

export async function cancelTapLiveQueue(eventId: number) {
  try {
    await apiClient.delete(`/events/${eventId}/tap-live/queue`, {
      headers: { ...getDeviceHeaders(), ...getFanSessionHeaders() },
    });
    return { ok: true };
  } catch (error) {
    return { ok: false, error };
  }
}

export async function fetchTapLiveState(eventId: number) {
  try {
    const { data } = await apiClient.get(`/events/${eventId}/tap-live/state`, {
      headers: { ...getDeviceHeaders(), ...getFanSessionHeaders() },
    });
    return { ok: true, data };
  } catch (error) {
    const status = axios.isAxiosError(error) ? error.response?.status : undefined;
    const message = axios.isAxiosError(error) ? error.response?.data?.message : undefined;
    return { ok: false, status, message, error };
  }
}

export async function submitTapLiveScore(eventId: number, matchId: string, score: number) {
  try {
    await apiClient.post(`/events/${eventId}/tap-live/submit`, {
      match_id: matchId,
      score,
    }, {
      headers: { ...getDeviceHeaders(), ...getFanSessionHeaders() },
    });
    return { ok: true };
  } catch (error) {
    return { ok: false, error };
  }
}

export async function abortTapLiveMatch(eventId: number, matchId: string) {
  try {
    await apiClient.post(`/events/${eventId}/tap-live/abort`, {
      match_id: matchId,
    }, {
      headers: { ...getDeviceHeaders(), ...getFanSessionHeaders() },
    });
    return { ok: true };
  } catch (error) {
    return { ok: false, error };
  }
}

export async function fetchTapLiveResult(eventId: number, matchId: string) {
  try {
    const { data } = await apiClient.get(`/events/${eventId}/tap-live/result`, {
      params: { match_id: matchId },
      headers: { ...getDeviceHeaders(), ...getFanSessionHeaders() },
    });
    return { ok: true, data };
  } catch (error) {
    return { ok: false, error };
  }
}

export function buildTapLiveSseUrl(eventId: number) {
  const token = encodeURIComponent(getFanSessionToken());
  return resolveApiUrl(`/events/${eventId}/tap-live/stream?fan_session=${token}`);
}
