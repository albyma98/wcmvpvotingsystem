export interface FingerprintPayload {
  browser: string;
  platform: string;
  screen: string;
  color_depth: number;
  timezone: string;
  timezone_offset: number;
  device_memory: string;
  hardware_concurrency: number;
  languages: string;
  graphics: string;
  touch_support: string;
}

const normalizeString = (value: unknown, fallback = '') => {
  if (typeof value !== 'string') {
    return fallback;
  }
  const trimmed = value.trim();
  return trimmed || fallback;
};

const readNavigator = () => {
  if (typeof window === 'undefined' || typeof navigator === 'undefined') {
    return null;
  }
  return navigator;
};

const readScreen = () => {
  if (typeof window === 'undefined' || typeof screen === 'undefined') {
    return null;
  }
  return screen;
};

const detectGraphicsRenderer = (): string => {
  if (typeof document === 'undefined') {
    return 'unknown';
  }
  try {
    const canvas = document.createElement('canvas');
    const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
    if (!gl) {
      return 'unknown';
    }
    const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
    if (debugInfo) {
      const renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
      if (typeof renderer === 'string' && renderer.trim()) {
        return renderer.trim();
      }
    }
    const fallbackRenderer = gl.getParameter(gl.RENDERER);
    if (typeof fallbackRenderer === 'string' && fallbackRenderer.trim()) {
      return fallbackRenderer.trim();
    }
  } catch (error) {
    console.warn('Unable to detect graphics renderer', error);
  }
  return 'unknown';
};

const detectTouchSupport = (): string => {
  if (typeof window === 'undefined') {
    return 'unknown';
  }
  try {
    const hasTouch = 'ontouchstart' in window || (readNavigator()?.maxTouchPoints ?? 0) > 0;
    return hasTouch ? 'touch-enabled' : 'no-touch';
  } catch {
    return 'unknown';
  }
};

let cachedFingerprint: FingerprintPayload | null = null;

export const collectDeviceFingerprint = async (): Promise<FingerprintPayload> => {
  if (cachedFingerprint) {
    return cachedFingerprint;
  }

  const nav = readNavigator();
  const scr = readScreen();
  const timezone = (() => {
    try {
      return Intl.DateTimeFormat().resolvedOptions().timeZone || 'unknown';
    } catch {
      return 'unknown';
    }
  })();
  const timezoneOffset = typeof Date === 'function' ? new Date().getTimezoneOffset() : 0;

  const fingerprint: FingerprintPayload = {
    browser: normalizeString(nav?.userAgent, 'unknown'),
    platform: normalizeString(nav?.platform, 'unknown'),
    screen: scr ? `${scr.width}x${scr.height}` : 'unknown',
    color_depth: scr?.colorDepth ?? 0,
    timezone,
    timezone_offset: Number.isFinite(timezoneOffset) ? timezoneOffset : 0,
    device_memory: (() => {
      const memory = (nav as unknown as { deviceMemory?: number })?.deviceMemory;
      if (typeof memory === 'number' && Number.isFinite(memory)) {
        return `${memory}`;
      }
      return 'unknown';
    })(),
    hardware_concurrency: (() => {
      const concurrency = nav?.hardwareConcurrency;
      if (typeof concurrency === 'number' && Number.isFinite(concurrency)) {
        return concurrency;
      }
      return 0;
    })(),
    languages: (() => {
      if (Array.isArray(nav?.languages) && nav.languages.length > 0) {
        return nav.languages.join(',');
      }
      return normalizeString(nav?.language, 'unknown');
    })(),
    graphics: detectGraphicsRenderer(),
    touch_support: detectTouchSupport(),
  };

  cachedFingerprint = fingerprint;
  return fingerprint;
};
