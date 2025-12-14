type TrackEventFn = (
  category: string,
  action: string,
  label?: string,
  extraData?: Record<string, unknown>,
) => unknown;

declare global {
  interface Window {
    trackEvent?: TrackEventFn;
  }
}

const getTrackEvent = (): TrackEventFn | null => {
  if (typeof window === "undefined") {
    return null;
  }

  const fn = typeof window.trackEvent === "function" ? window.trackEvent : null;
  return fn;
};

export function safeTrackEvent(
  category: string,
  action: string,
  label?: string,
  extraData?: Record<string, unknown>,
) {
  try {
    const tracker = getTrackEvent();
    if (!tracker) {
      return;
    }
    const result = tracker(category, action, label, extraData);
    if (result && typeof (result as Promise<unknown>).catch === "function") {
      (result as Promise<unknown>).catch(() => {});
    }
  } catch (error) {
    console.warn("trackEvent failed", error);
  }
}
