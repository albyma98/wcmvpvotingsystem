import axios from 'axios';
import { apiClient, getDeviceHeaders } from '../api';

type AwardTapChallengePayload = {
  amount: number;
  eventId: string;
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
    const { data } = await apiClient.post(
      '/coins/earn',
      {
        source: 'tap_challenge',
        amount,
        eventId: payload.eventId,
        eventContextId: payload.eventContextId || 0,
        meta: payload.meta || {},
      },
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
