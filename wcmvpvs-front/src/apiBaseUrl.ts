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

export const resolveApiBaseUrl = () => {
  const envUrl = import.meta.env.VITE_API_BASE_URL?.trim();

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

export const API_BASE_URL = resolveApiBaseUrl();
