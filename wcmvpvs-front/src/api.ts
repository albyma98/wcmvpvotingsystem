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

export async function vote({ eventId, playerId }) {
  try {
    const { data: voteData } = await apiClient.post('/vote', {
      player_id: playerId,
      event_id: eventId,
      device_id: getOrCreateDeviceId(),
    });

    return { ok: true, vote: voteData };
  } catch (error) {
    console.error('vote api error', error);
    return { ok: false, error };
  }
}
