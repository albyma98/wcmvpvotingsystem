import axios from 'axios';
import { apiClient, getDeviceHeaders } from '../api';

type AwardTapChallengePayload = {
  amount: number;
  requestId: string;
  eventContextId?: number;
  meta?: Record<string, unknown>;
};

export async function awardTapChallengeCoins(payload: AwardTapChallengePayload) {
  const amount = Math.max(0, Number(payload.amount) || 0);
  if (!amount) {
    return { ok: true, data: { amount: 0 } };
  }

  const headers = getDeviceHeaders();

  try {
    const eventContextId = Math.max(0, Number(payload.eventContextId) || 0);
    const endpoint = eventContextId ? `/events/${eventContextId}/guest-coins` : '/coins/earn';
    const requestBody = eventContextId
      ? { coins: amount }
      : {
          source: 'tap_challenge',
          amount,
          // Keep both naming styles while backend contracts are being aligned.
          request_id: payload.requestId,
          requestId: payload.requestId,
          event_id: eventContextId || undefined,
          eventId: eventContextId || undefined,
          event_context_id: eventContextId,
          eventContextId: eventContextId,
          meta: payload.meta || {},
        };

    const { data } = await apiClient.post(endpoint, requestBody, { headers });
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
