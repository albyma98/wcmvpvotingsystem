const STORAGE_KEY = 'wcmvpvs:device_id';

const fallbackRandomId = () => {
  const random = () => Math.random().toString(36).slice(2, 10);
  return `${Date.now().toString(36)}-${random()}-${random()}`;
};

const hashString = (value: string) => {
  let hash = 0;
  for (let index = 0; index < value.length; index += 1) {
    hash = (hash << 5) - hash + value.charCodeAt(index);
    hash |= 0; // Convert to 32bit integer
  }
  return `fp-${(hash >>> 0).toString(16)}`;
};

const getDeterministicDeviceId = (): string | null => {
  if (typeof window === 'undefined') {
    return null;
  }

  try {
    const navigatorInfo = window.navigator || ({} as Navigator);
    const screenInfo = window.screen || ({} as Screen);

    const dataPoints = [
      navigatorInfo.platform ?? '',
      navigatorInfo.language ?? '',
      Array.isArray(navigatorInfo.languages) ? navigatorInfo.languages.join(',') : '',
      navigatorInfo.vendor ?? '',
      String(navigatorInfo.hardwareConcurrency ?? ''),
      String(navigatorInfo.maxTouchPoints ?? ''),
      `${screenInfo.width ?? ''}x${screenInfo.height ?? ''}x${screenInfo.colorDepth ?? ''}`,
      String(Math.round((window.devicePixelRatio ?? 0) * 100) / 100),
      String(new Date().getTimezoneOffset()),
    ];

    const fingerprintSource = dataPoints.join('::');
    if (fingerprintSource.trim() === '') {
      return null;
    }

    return hashString(fingerprintSource);
  } catch (error) {
    console.error('Unable to compute deterministic device id', error);
    return null;
  }
};

export const getOrCreateDeviceId = () => {
  if (typeof window === 'undefined') {
    return 'server';
  }

  let storage: Storage | null = null;
  try {
    storage = window.localStorage;
    const existing = storage.getItem(STORAGE_KEY);
    if (existing && existing.trim() !== '') {
      return existing;
    }
  } catch (error) {
    console.error('Unable to access localStorage for device id', error);
  }

  const deterministicId = getDeterministicDeviceId();
  const newId = deterministicId ?? (typeof crypto !== 'undefined' && 'randomUUID' in crypto
    ? crypto.randomUUID()
    : fallbackRandomId());

  if (storage) {
    try {
      storage.setItem(STORAGE_KEY, newId);
    } catch (error) {
      console.error('Unable to persist device id', error);
    }
  }

  return newId;
};
