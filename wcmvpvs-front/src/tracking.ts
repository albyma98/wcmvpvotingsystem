declare global {
  interface Window {
    trackEvent?: (category: string, action: string, label: string, extraData?: unknown) => void;
  }

  // eslint-disable-next-line no-var
  var trackEvent:
    | ((category: string, action: string, label: string, extraData?: unknown) => void)
    | undefined;
}

function resolveTracker() {
  if (typeof trackEvent === 'function') {
    return trackEvent;
  }

  if (typeof window !== 'undefined' && typeof window.trackEvent === 'function') {
    return window.trackEvent;
  }

  return null;
}

export function safeTrackEvent(
  category: string,
  action: string,
  label: string,
  extraData?: unknown,
) {
  try {
    const tracker = resolveTracker();
    if (tracker) {
      tracker(category, action, label, extraData);
    }
  } catch (error) {
    // ignore tracking errors
  }
}
