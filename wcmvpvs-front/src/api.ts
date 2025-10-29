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

export const apiClient = axios.create({
  baseURL: resolveApiBaseUrl(),
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
    return { ok: true, hasVoted: Boolean(data?.has_voted) };
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

export async function uploadSelfie(
  eventId: number,
  { file, caption, imageBase64 }: { file?: File; caption?: string; imageBase64?: string },
) {
  if (!eventId) {
    return { ok: false, error: new Error('missing_event_id') };
  }

  const headers = getDeviceHeaders();
  if (!headers['X-Device-ID']) {
    return { ok: false, error: new Error('missing_device_id') };
  }

  try {
    if (file instanceof File) {
      const formData = new FormData();
      formData.append('image', file);
      if (caption) {
        formData.append('caption', caption);
      }

      const { data } = await apiClient.post(`/events/${eventId}/selfies`, formData, {
        headers,
      });
      return { ok: true, selfie: data };
    }

    if (typeof imageBase64 === 'string' && imageBase64.trim()) {
      const payload = {
        caption: caption ?? '',
        image_base64: imageBase64,
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
