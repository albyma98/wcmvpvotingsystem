const STORAGE_KEY = 'wcmvpvs:device_id';

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
      return existing;
    }

    const newId = typeof crypto !== 'undefined' && 'randomUUID' in crypto
      ? crypto.randomUUID()
      : fallbackRandomId();

    storage.setItem(STORAGE_KEY, newId);
    return newId;
  } catch (error) {
    console.error('Unable to access localStorage for device id', error);
    return fallbackRandomId();
  }
};
