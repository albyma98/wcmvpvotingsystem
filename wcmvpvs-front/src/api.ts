import axios from 'axios';

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

  const envPort = import.meta.env.VITE_API_PORT?.toString().trim();
  const { hostname, protocol, port: windowPort } = resolveFromWindow();

  if (envPort) {
    const sanitizedPort = envPort.replace(/^:/, '');
    return `${protocol}//${hostname}:${sanitizedPort}`;
  }

  const originPort = windowPort ? `:${windowPort}` : '';
  return `${protocol}//${hostname}${originPort}`;
};

export const apiClient = axios.create({
  baseURL: resolveApiBaseUrl(),
});

export async function vote({ eventId, playerId }) {
  try {
    const { data: voteData } = await apiClient.post('/vote', {
      player_id: playerId,
      event_id: eventId,
      device_id: 'web',
    });

    let ticket = null;
    try {
      const { data } = await apiClient.post('/ticket');
      ticket = data;
    } catch (ticketError) {
      console.error('ticket api error', ticketError);
    }

    return { ok: true, vote: voteData, ticket };
  } catch (error) {
    console.error('vote api error', error);
    return { ok: false, error };
  }
}
