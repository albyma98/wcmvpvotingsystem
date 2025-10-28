import { API_BASE_URL } from './apiBaseUrl';

const STORAGE_KEY = 'wcmvpvs:device_token';

const randomToken = () => {
  const random = () => Math.random().toString(36).slice(2, 10);
  return `${Date.now().toString(36)}-${random()}-${random()}`;
};

const readStoredToken = () => {
  if (typeof window === 'undefined') {
    return '';
  }
  try {
    const stored = window.localStorage.getItem(STORAGE_KEY);
    return stored?.trim() || '';
  } catch {
    return '';
  }
};

const persistToken = (token: string) => {
  if (typeof window === 'undefined') {
    return;
  }
  try {
    window.localStorage.setItem(STORAGE_KEY, token);
  } catch (error) {
    console.warn('Unable to persist device token', error);
  }
};

let inFlightRequest: Promise<string> | null = null;

export const getOrCreateDeviceToken = async (): Promise<string> => {
  if (typeof window === 'undefined') {
    return 'server';
  }

  const stored = readStoredToken();
  if (stored) {
    return stored;
  }

  if (inFlightRequest) {
    return inFlightRequest;
  }

  inFlightRequest = (async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/device-token`, {
        credentials: 'include',
        method: 'GET',
      });
      if (!response.ok) {
        throw new Error(`device token request failed with ${response.status}`);
      }
      const data = await response.json();
      const token = typeof data?.token === 'string' ? data.token.trim() : '';
      if (token) {
        persistToken(token);
        return token;
      }
      const fallback = randomToken();
      persistToken(fallback);
      return fallback;
    } catch (error) {
      console.warn('Falling back to random device token', error);
      const fallback = randomToken();
      persistToken(fallback);
      return fallback;
    } finally {
      inFlightRequest = null;
    }
  })();

  return inFlightRequest;
};
