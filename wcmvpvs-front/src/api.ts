import axios from 'axios';
import { API_BASE_URL } from './apiBaseUrl';
import { collectDeviceFingerprint, type FingerprintPayload } from './fingerprint';
import { getOrCreateDeviceToken } from './deviceToken';

export const apiClient = axios.create({
  baseURL: API_BASE_URL,
});

const fallbackFingerprint: FingerprintPayload = {
  browser: 'unknown',
  platform: 'unknown',
  screen: 'unknown',
  color_depth: 0,
  timezone: 'unknown',
  timezone_offset: 0,
  device_memory: 'unknown',
  hardware_concurrency: 0,
  languages: 'unknown',
  graphics: 'unknown',
  touch_support: 'unknown',
};

export async function vote({ eventId, playerId }) {
  try {
    const [deviceToken, fingerprint] = await Promise.all([
      getOrCreateDeviceToken(),
      collectDeviceFingerprint().catch((error) => {
        console.warn('collectDeviceFingerprint failed', error);
        return fallbackFingerprint;
      }),
    ]);

    const { data: voteData } = await apiClient.post('/vote', {
      player_id: playerId,
      event_id: eventId,
      device_token: deviceToken,
      fingerprint,
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
