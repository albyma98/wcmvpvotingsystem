const STORAGE_KEY = 'wcmvpvs:device_id';
let sessionDeviceId: string | undefined;

const fallbackRandomId = () => {
  const random = () => Math.random().toString(36).slice(2, 10);
  return `${Date.now().toString(36)}-${random()}-${random()}`;
};

export const getOrCreateDeviceId = () => {
  if (typeof window === 'undefined') {
    return 'server';
  }

  try {
    const storage = window.localStorage;
    const existing = storage.getItem(STORAGE_KEY);
    if (existing && existing.trim() !== '') {
      return sessionDeviceId = existing;
    }

    const newId = sessionDeviceId ||= createDeviceId();
    storage.setItem(STORAGE_KEY, newId);
    return newId;
  } catch (error) {
    return sessionDeviceId ||= createDeviceId();
  }
};

const createDeviceId = () => typeof crypto !== 'undefined' && 'randomUUID' in crypto
  ? crypto.randomUUID()
  : fallbackRandomId();
