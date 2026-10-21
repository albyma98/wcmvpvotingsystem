import axios from 'axios';
import { getOrCreateDeviceId } from './deviceId';
import { readFanSessionToken, saveFanSessionToken, readLegacyFanSessionToken } from './fanSession';

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
    // In dev, use relative path so the Vite proxy handles forwarding to the backend.
    // The proxy target is configured in vite.config.js via VITE_API_BASE_URL.
    return '/api';
  }

  const originPort = windowPort ? `:${windowPort}` : '';
  return ensureApiPath(`${protocol}//${hostname}${originPort}`);
};

function detectOrganizationSlug(pathname: string | undefined, search?: string) {
  const reservedPrefixes = new Set(['admin', 'lottery', 'welcome', 'partner']);

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
  withCredentials: true,
});

apiClient.interceptors.request.use((config) => {
  const orgSlug = getOrganizationSlugFromLocation();
  if (orgSlug && !config.headers?.['X-Organization-Slug']) {
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

function resolveBeaconUrl(path: string) {
  const baseUrl = resolveApiUrl(path);
  const organizationSlug = getOrganizationSlugFromLocation();

  if (!organizationSlug) {
    return baseUrl;
  }

  try {
    const url = new URL(baseUrl);
    if (!url.searchParams.get('organization_slug')) {
      url.searchParams.set('organization_slug', organizationSlug);
    }
    return url.toString();
  } catch (error) {
    const separator = baseUrl.includes('?') ? '&' : '?';
    return `${baseUrl}${separator}organization_slug=${encodeURIComponent(organizationSlug)}`;
  }
}

export function sendJsonBeacon(path: string, payload: Record<string, unknown> = {}) {
  if (!path) {
    return Promise.resolve();
  }

  const url = resolveBeaconUrl(path);
  const body = JSON.stringify(payload);
  const organizationSlug = getOrganizationSlugFromLocation();
  const deviceId = getOrCreateDeviceId();
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };

  if (organizationSlug) {
    headers['X-Organization-Slug'] = organizationSlug;
  }
  if (deviceId) {
    headers['X-Device-ID'] = deviceId;
  }

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
      headers,
      body,
      keepalive: true,
    }).then(() => {});
  }

  return apiClient.post(path, payload, { headers }).then(() => {});
}


export function trackEventBatch(eventId: number, payload: Record<string, unknown>) {
  if (!eventId) {
    return Promise.resolve();
  }

  return sendJsonBeacon(`/events/${eventId}/tracking/events`, payload);
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

export async function submitQuickQuizAnswer(eventId: number, payload: { questionId: number; selectedIndex: number; responseMs: number; deviceId: string; journey?: boolean; }) {
  if (!eventId) {
    return { ok: false, error: new Error('missing_event_id') };
  }
  try {
    const { data } = await apiClient.post(`/public/events/${eventId}/quiz/answer`, {
      questionId: payload.questionId,
      selectedIndex: payload.selectedIndex,
      responseMs: payload.responseMs,
      deviceId: payload.deviceId,
      journey: payload.journey === true,
    }, { headers: getDeviceHeaders() });
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


export function getFanSessionToken(organizationSlug = getOrganizationSlug()) {
  if (typeof window === 'undefined') {
    return '';
  }
  return readFanSessionToken(window.localStorage, organizationSlug);
}

export function getFanSessionHeaders(organizationSlug = getOrganizationSlug()) {
  const token = getFanSessionToken(organizationSlug);
  return token ? { 'X-Fan-Session': token } : {};
}

export async function fetchFanProfile(eventId) {
  const organizationSlug = getOrganizationSlug();
  const storage = typeof window !== 'undefined' ? window.localStorage : null;
  const token = getFanSessionToken(organizationSlug);
  const legacyToken = token ? '' : readLegacyFanSessionToken(storage);
  const headers = { ...getDeviceHeaders(), 'X-Organization-Slug': organizationSlug, ...(token || legacyToken ? { 'X-Fan-Session': token || legacyToken } : {}) };
  try {
    const { data } = await apiClient.get('/fan/me', {
      params: { event_id: eventId },
      headers,
    });
    if (legacyToken && (data?.registered || data?.registration_pending)) saveFanSessionToken(storage, organizationSlug, legacyToken);
    return { ok: true, data };
  } catch (error) {
    if (legacyToken && axios.isAxiosError(error) && error.response?.status === 403) {
      try {
        const { data } = await apiClient.get('/fan/me', { params: { event_id: eventId }, headers: { ...getDeviceHeaders(), 'X-Organization-Slug': organizationSlug } });
        return { ok: true, data };
      } catch (guestError) { return { ok: false, error: guestError }; }
    }
    return { ok: false, error };
  }
}

export async function registerFanProfile(payload) {
  const organizationSlug = getOrganizationSlug();
  try {
    const { data } = await apiClient.post('/fan/register', payload, {
      headers: { ...getDeviceHeaders(), ...getFanSessionHeaders(organizationSlug), 'X-Organization-Slug': organizationSlug },
    });
    if (data?.session_token && typeof window !== 'undefined') {
      saveFanSessionToken(window.localStorage, organizationSlug, data.session_token);
    }
    return { ok: true, data };
  } catch (error) {
    const message = axios.isAxiosError(error) ? error.response?.data?.message : undefined;
    const code = axios.isAxiosError(error) ? error.response?.data?.code : undefined;
    return { ok: false, error, message, code };
  }
}

export async function checkFanNickname(nickname = '') {
  try {
    const { data } = await apiClient.get('/fan/nickname', { params: { nickname }, timeout: 10000 });
    return { ok: true, data };
  } catch (error) {
    const message = axios.isAxiosError(error) ? error.response?.data?.message : undefined;
    return { ok: false, error, message };
  }
}

export async function updateFanNickname(nickname: string) {
  try {
    const { data } = await apiClient.put('/fan/nickname', { nickname }, {
      headers: { ...getDeviceHeaders(), ...getFanSessionHeaders() },
    });
    return { ok: true, data };
  } catch (error) {
    const status = axios.isAxiosError(error) ? error.response?.status : undefined;
    const message = axios.isAxiosError(error) ? error.response?.data?.message : undefined;
    return { ok: false, status, message, error };
  }
}

export async function startPhoneAuth(phone: string, mode: 'register' | 'login' = 'register', acceptance: Record<string, unknown> = {}) {
  const organizationSlug = getOrganizationSlug();
  try {
    await apiClient.post('/auth/start', { ...(mode === 'register' ? acceptance : {}), phone, mode }, { headers: { ...getDeviceHeaders(), 'X-Organization-Slug': organizationSlug } });
    return { ok: true };
  } catch (error) {
    const code = axios.isAxiosError(error) ? error.response?.data?.error : undefined;
    return { ok: false, error, code };
  }
}

export async function verifyPhoneAuth(phone: string, code: string) {
  const organizationSlug = getOrganizationSlug();
  try {
    const { data } = await apiClient.post('/auth/verify', { phone, code }, { headers: { ...getDeviceHeaders(), 'X-Organization-Slug': organizationSlug } });
    const sessionToken = String(data?.token || data?.session_token || '').trim();
    if (sessionToken && typeof window !== 'undefined') {
      saveFanSessionToken(window.localStorage, organizationSlug, sessionToken);
    }
    return { ok: true, data };
  } catch (error) {
    const errCode = axios.isAxiosError(error) ? error.response?.data?.error : undefined;
    return { ok: false, error, code: errCode };
  }
}

export async function resendPhoneAuth(phone: string) {
  const organizationSlug = getOrganizationSlug();
  try {
    await apiClient.post('/auth/resend', { phone }, { headers: { ...getDeviceHeaders(), 'X-Organization-Slug': organizationSlug } });
    return { ok: true };
  } catch (error) {
    const code = axios.isAxiosError(error) ? error.response?.data?.error : undefined;
    return { ok: false, error, code };
  }
}

export async function syncGuestCoins(eventId, coins) {
  try {
    const { data } = await apiClient.post(`/events/${eventId}/guest-coins`, { coins }, {
      headers: { ...getDeviceHeaders(), ...getFanSessionHeaders() },
    });
    return { ok: true, data };
  } catch (error) {
    return { ok: false, error };
  }
}

export async function claimFanJourneyReward(eventId: number, key: 'vote' | 'quiz' | 'reaction' | 'tap' | 'sponsor-rush' | 'memory-flash') {
  try {
    const { data } = await apiClient.post(`/events/${eventId}/journey-rewards`, { key }, { headers: { ...getDeviceHeaders(), ...getFanSessionHeaders() } });
    return { ok: true, data };
  } catch (error) {
    const message = axios.isAxiosError(error) ? error.response?.data?.message : undefined;
    return { ok: false, error, message };
  }
}

async function requestFanWallet(eventId: number, action?: 'purchase' | 'equip', itemKey?: string) {
  if (!Number.isInteger(eventId) || eventId <= 0) return { ok: false, message: 'Disponibile con una partita del club.' };
  try {
    const config = { headers: { ...getDeviceHeaders(), ...getFanSessionHeaders() }, timeout: 12000 };
    const { data } = action
      ? await apiClient.post(`/events/${eventId}/fan-shop/${action}`, { item_key: itemKey || '' }, config)
      : await apiClient.get(`/events/${eventId}/fan-wallet`, config);
    return { ok: true, data };
  } catch (error) {
    const response = axios.isAxiosError(error) ? error.response : undefined;
    return { ok: false, status: response?.status, code: response?.data?.code, message: response?.data?.message || 'Connessione non disponibile. Riprova.', error };
  }
}

export const fetchFanWallet = (eventId: number) => requestFanWallet(eventId);
export const purchaseFanItem = (eventId: number, itemKey: string) => requestFanWallet(eventId, 'purchase', itemKey);
export const equipFanItem = (eventId: number, itemKey: string) => requestFanWallet(eventId, 'equip', itemKey);

async function requestBasketballShot(eventId: number, shotId?: string, gesture?: { dx: number; dy: number }) {
  if (!Number.isInteger(eventId) || eventId <= 0) return { ok: false, message: 'Partita non disponibile.', status: 400 };
  try {
    const path = `/events/${eventId}/basketball/shots${shotId ? `/${encodeURIComponent(shotId)}` : ''}`;
    const { data } = await apiClient.post(path, gesture || {}, {
      headers: { ...getDeviceHeaders(), ...getFanSessionHeaders() }, timeout: 12000,
    });
    return { ok: true, data };
  } catch (error) {
    const response = axios.isAxiosError(error) ? error.response : undefined;
    return { ok: false, error, status: response?.status, message: response?.data?.message };
  }
}

export function prepareBasketballShot(eventId: number) {
  return requestBasketballShot(eventId);
}

export function submitBasketballShot(eventId: number, shotId: string, gesture: { dx: number; dy: number }) {
  return requestBasketballShot(eventId, shotId, gesture);
}

async function requestPlaymaker(eventId: number, sessionId?: string, pass?: { attempt: number; elapsed_ms: number; dx: number; dy: number }) {
  if (!Number.isInteger(eventId) || eventId <= 0) return { ok: false, message: 'Partita non disponibile.', status: 400 };
  try {
    const path = `/events/${eventId}/playmaker/sessions${sessionId ? `/${encodeURIComponent(sessionId)}/passes` : ''}`;
    const { data } = await apiClient.post(path, pass || {}, {
      headers: { ...getDeviceHeaders(), ...getFanSessionHeaders() }, timeout: 12000,
    });
    return { ok: true, data };
  } catch (error) {
    const response = axios.isAxiosError(error) ? error.response : undefined;
    return { ok: false, error, status: response?.status, message: response?.data?.message };
  }
}

export function startPlaymakerSession(eventId: number) {
  return requestPlaymaker(eventId);
}

export function submitPlaymakerPass(eventId: number, sessionId: string, pass: { attempt: number; elapsed_ms: number; dx: number; dy: number }) {
  return requestPlaymaker(eventId, sessionId, pass);
}

async function requestCrossover(eventId: number, sessionId?: string, duel?: { duel: number; duel_start_ms: number; samples: Array<{ t_ms: number; x: number }> }) {
  if (!Number.isInteger(eventId) || eventId <= 0) return { ok: false, message: 'Partita non disponibile.', status: 400 };
  try {
    const path = `/events/${eventId}/crossover/sessions${sessionId ? `/${encodeURIComponent(sessionId)}/duels` : ''}`;
    const { data } = await apiClient.post(path, duel || {}, {
      headers: { ...getDeviceHeaders(), ...getFanSessionHeaders() }, timeout: 12000,
    });
    return { ok: true, data };
  } catch (error) {
    const response = axios.isAxiosError(error) ? error.response : undefined;
    return { ok: false, error, status: response?.status, message: response?.data?.message };
  }
}

export function startCrossoverSession(eventId: number) {
  return requestCrossover(eventId);
}

export function submitCrossoverDuel(eventId: number, sessionId: string, duel: { duel: number; duel_start_ms: number; samples: Array<{ t_ms: number; x: number }> }) {
  return requestCrossover(eventId, sessionId, duel);
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

export async function donateFanSupport(eventId, coins) {
  try {
    const { data } = await apiClient.post(`/events/${eventId}/support/donate`, { coins }, {
      headers: { ...getDeviceHeaders(), ...getFanSessionHeaders() },
    });
    return { ok: true, data };
  } catch (error) {
    const message = axios.isAxiosError(error) ? error.response?.data?.message : undefined;
    const status = axios.isAxiosError(error) ? error.response?.status : undefined;
    return { ok: false, message, status, error };
  }
}

export async function fetchFanSupportLeaderboard(eventId, limit = 10) {
  try {
    const { data } = await apiClient.get(`/events/${eventId}/support/leaderboard`, {
      params: { limit },
      headers: { ...getDeviceHeaders(), ...getFanSessionHeaders() },
    });
    return { ok: true, data };
  } catch (error) {
    return { ok: false, error };
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

export async function requestTapLiveRematch(eventId: number, matchId: string, action: 'request' | 'decline' | 'leave' = 'request') {
  try {
    const { data } = await apiClient.post(`/events/${eventId}/tap-live/rematch`, {
      match_id: matchId,
      action,
    }, {
      headers: { ...getDeviceHeaders(), ...getFanSessionHeaders() },
    });
    return { ok: true, data };
  } catch (error) {
    const message = axios.isAxiosError(error) ? error.response?.data?.message : undefined;
    return { ok: false, error, message };
  }
}

export async function leaveTapLivePostmatch(eventId: number, matchId: string) {
  try {
    await apiClient.post(`/events/${eventId}/tap-live/postmatch-leave`, {
      match_id: matchId,
    }, {
      headers: { ...getDeviceHeaders(), ...getFanSessionHeaders() },
    });
    return { ok: true };
  } catch (error) {
    return { ok: false, error };
  }
}

export function buildTapLiveSseUrl(eventId: number) {
  const token = encodeURIComponent(getFanSessionToken());
  return resolveApiUrl(`/events/${eventId}/tap-live/stream?fan_session=${token}`);
}

export async function generateAIBarUpsell(payload: Record<string, unknown>) {
  const { data } = await apiClient.post('/ai/bar/upsell', payload, {
    headers: getDeviceHeaders(),
  });
  return data;
}

export async function generateAIPopup(payload: Record<string, unknown>) {
  const { data } = await apiClient.post('/ai/popups/generate', payload, {
    headers: getDeviceHeaders(),
  });
  return data;
}

export async function trackAIInteraction(interactionId: number, outcome: string, extra: Record<string, unknown> = {}) {
  if (!interactionId) {
    return;
  }
  await apiClient.post(`/ai/interactions/${interactionId}/track`, {
    outcome,
    occurred_at: new Date().toISOString(),
    ...extra,
  }, {
    headers: getDeviceHeaders(),
  });
}
