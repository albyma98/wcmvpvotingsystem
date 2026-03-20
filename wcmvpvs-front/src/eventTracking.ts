import { getOrCreateDeviceId } from './deviceId';
import { getFanSessionToken, getOrganizationSlug, sendJsonBeacon } from './api';

type TrackingPrimitive = string | number | boolean | null;
type TrackingValue = TrackingPrimitive | TrackingValue[] | { [key: string]: TrackingValue };
export type TrackingPayload = Record<string, TrackingValue | undefined>;

type TrackingContext = {
  eventId?: number;
  organizationId?: number;
  organizationSlug?: string;
  fanId?: number;
  sessionId?: string;
  page?: string;
  section?: string;
  source?: string;
  loginState?: string;
  profileState?: string;
  fanSessionToken?: string;
  metadata?: TrackingPayload;
};

type QueuedTrackingEvent = {
  name: string;
  domain: string;
  occurred_at: string;
  page?: string;
  section?: string;
  source?: string;
  session_id?: string;
  device_id?: string;
  fan_id?: number;
  organization_id?: number;
  organization_slug?: string;
  login_state?: string;
  profile_state?: string;
  metadata?: TrackingPayload;
};

type LifecycleOptions = {
  eventId?: number;
  organizationId?: number;
  page: string;
  source?: string;
  idleThresholdMs?: number;
};

const SESSION_STORAGE_KEY = 'mvp:tracking:session_id';
const SESSION_LAST_ACTIVITY_KEY = 'mvp:tracking:last_activity';
const DEFAULT_IDLE_THRESHOLD_MS = 60_000;
const FLUSH_INTERVAL_MS = 5_000;
const MAX_BATCH_SIZE = 20;
const MAX_QUEUE_SIZE = 100;

let context: TrackingContext = {};
let queue: QueuedTrackingEvent[] = [];
let flushTimer: number | null = null;
let visibilityListenerAttached = false;
let beforeUnloadAttached = false;
let activityListenersAttached = false;
let idleTimer: number | null = null;
let isIdle = false;
let lifecycleStartedAt = 0;
let idleThresholdMs = DEFAULT_IDLE_THRESHOLD_MS;
let lastActivityAt = 0;
let currentPage = '';
let pendingExitReason = 'page_unload';

function safeNowIso() {
  return new Date().toISOString();
}

function sanitizePayload(payload?: TrackingPayload): TrackingPayload | undefined {
  if (!payload || typeof payload !== 'object') {
    return undefined;
  }

  const entries = Object.entries(payload).filter(([, value]) => value !== undefined);
  if (!entries.length) {
    return undefined;
  }

  return Object.fromEntries(entries);
}

function ensureSessionId() {
  if (context.sessionId) {
    return context.sessionId;
  }

  if (typeof window === 'undefined') {
    const generated = `srv-${Date.now()}`;
    context.sessionId = generated;
    return generated;
  }

  const now = Date.now();
  const lastActivity = Number.parseInt(window.sessionStorage.getItem(SESSION_LAST_ACTIVITY_KEY) || '0', 10);
  const existing = String(window.sessionStorage.getItem(SESSION_STORAGE_KEY) || '').trim();
  const shouldRotate = !existing || !Number.isFinite(lastActivity) || now - lastActivity > 30 * 60 * 1000;
  const next = shouldRotate
    ? `sess_${Math.random().toString(36).slice(2, 10)}_${now.toString(36)}`
    : existing;

  window.sessionStorage.setItem(SESSION_STORAGE_KEY, next);
  window.sessionStorage.setItem(SESSION_LAST_ACTIVITY_KEY, String(now));
  context.sessionId = next;
  return next;
}

export function updateTrackingContext(partial: TrackingContext = {}) {
  context = {
    ...context,
    ...partial,
    organizationSlug: partial.organizationSlug ?? context.organizationSlug ?? (getOrganizationSlug() || undefined),
    fanSessionToken: partial.fanSessionToken ?? context.fanSessionToken ?? (getFanSessionToken() || undefined),
  };
  if (!context.sessionId) {
    context.sessionId = ensureSessionId();
  }
  if (!context.page && currentPage) {
    context.page = currentPage;
  }
}

function scheduleFlush() {
  if (typeof window === 'undefined' || flushTimer !== null) {
    return;
  }

  flushTimer = window.setTimeout(() => {
    flushTimer = null;
    void flushTrackingQueue();
  }, FLUSH_INTERVAL_MS);
}

export function trackAppEvent(name: string, payload: TrackingPayload = {}, domain?: string) {
  const trimmedName = String(name || '').trim();
  if (!trimmedName) {
    return;
  }

  const finalDomain = String(domain || trimmedName.split('.')[0] || 'app').trim() || 'app';
  const metadata = sanitizePayload({
    ...context.metadata,
    ...payload,
  });

  const event: QueuedTrackingEvent = {
    name: trimmedName,
    domain: finalDomain,
    occurred_at: safeNowIso(),
    page: context.page,
    section: context.section,
    source: context.source,
    session_id: ensureSessionId(),
    device_id: getOrCreateDeviceId() || undefined,
    fan_id: context.fanId,
    organization_id: context.organizationId,
    organization_slug: context.organizationSlug,
    login_state: context.loginState,
    profile_state: context.profileState,
    metadata,
  };

  queue.push(event);
  if (queue.length > MAX_QUEUE_SIZE) {
    queue = queue.slice(queue.length - MAX_QUEUE_SIZE);
  }

  if (typeof window !== 'undefined') {
    window.sessionStorage.setItem(SESSION_LAST_ACTIVITY_KEY, String(Date.now()));
  }

  if (queue.length >= MAX_BATCH_SIZE) {
    void flushTrackingQueue();
    return;
  }

  scheduleFlush();
}

export function trackSectionView(section: string, payload: TrackingPayload = {}) {
  updateTrackingContext({ section });
  trackAppEvent('content.section_viewed', { section, ...payload }, 'content');
}

export function flushTrackingQueue() {
  if (!queue.length) {
    return Promise.resolve();
  }

  const eventId = Number(context.eventId) || 0;
  if (!eventId) {
    queue = [];
    return Promise.resolve();
  }

  const batch = queue.slice(0, MAX_BATCH_SIZE);
  queue = queue.slice(batch.length);

  const payload = {
    session_id: ensureSessionId(),
    page: context.page,
    source: context.source,
    fan_session_token: context.fanSessionToken || getFanSessionToken() || undefined,
    events: batch,
  };

  return sendJsonBeacon(`/events/${eventId}/tracking/events`, payload)
    .catch(() => {})
    .finally(() => {
      if (queue.length) {
        scheduleFlush();
      }
    });
}

function handleVisibilityChange() {
  if (typeof document === 'undefined') {
    return;
  }

  if (document.visibilityState === 'hidden') {
    pendingExitReason = 'app_backgrounded';
    trackAppEvent('session.hidden', {
      hidden_duration_ms: Math.max(0, Date.now() - lifecycleStartedAt),
    }, 'session');
    void flushTrackingQueue();
    return;
  }

  pendingExitReason = 'page_unload';
  trackAppEvent('session.resumed', {
    inactive_ms: Math.max(0, Date.now() - lastActivityAt),
  }, 'session');
}

function clearIdleTimer() {
  if (typeof window !== 'undefined' && idleTimer !== null) {
    window.clearTimeout(idleTimer);
    idleTimer = null;
  }
}

function scheduleIdleTimer() {
  clearIdleTimer();
  if (typeof window === 'undefined') {
    return;
  }

  idleTimer = window.setTimeout(() => {
    if (isIdle) {
      return;
    }
    isIdle = true;
    trackAppEvent('session.idle_started', {
      idle_threshold_ms: idleThresholdMs,
      since_last_activity_ms: Math.max(0, Date.now() - lastActivityAt),
    }, 'session');
  }, idleThresholdMs);
}

function registerActivity() {
  const now = Date.now();
  const previousActivityAt = lastActivityAt;
  const wasIdle = isIdle;
  lastActivityAt = now;
  if (wasIdle) {
    isIdle = false;
    trackAppEvent('session.idle_resumed', {
      idle_duration_ms: Math.max(0, now - previousActivityAt),
    }, 'session');
  }
  scheduleIdleTimer();
}

function attachLifecycleListeners() {
  if (typeof window === 'undefined' || typeof document === 'undefined') {
    return;
  }

  if (!visibilityListenerAttached) {
    document.addEventListener('visibilitychange', handleVisibilityChange);
    visibilityListenerAttached = true;
  }

  if (!beforeUnloadAttached) {
    window.addEventListener('beforeunload', () => {
      endTrackingLifecycle(pendingExitReason);
    });
    beforeUnloadAttached = true;
  }

  if (!activityListenersAttached) {
    ['click', 'keydown', 'touchstart', 'scroll'].forEach((eventName) => {
      window.addEventListener(eventName, registerActivity, { passive: true });
    });
    activityListenersAttached = true;
  }
}

export function startTrackingLifecycle(options: LifecycleOptions) {
  idleThresholdMs = options.idleThresholdMs || DEFAULT_IDLE_THRESHOLD_MS;
  currentPage = options.page;
  lifecycleStartedAt = Date.now();
  lastActivityAt = lifecycleStartedAt;
  pendingExitReason = 'page_unload';
  updateTrackingContext({
    eventId: options.eventId,
    organizationId: options.organizationId,
    organizationSlug: getOrganizationSlug() || undefined,
    page: options.page,
    source: options.source,
    sessionId: ensureSessionId(),
  });
  attachLifecycleListeners();
  scheduleIdleTimer();
  trackAppEvent('session.started', {
    referrer: typeof document !== 'undefined' ? document.referrer || undefined : undefined,
    url: typeof window !== 'undefined' ? window.location.href : undefined,
  }, 'session');
  trackAppEvent('content.page_viewed', {
    page: options.page,
    path: typeof window !== 'undefined' ? window.location.pathname : undefined,
  }, 'content');
}

export function endTrackingLifecycle(reason = 'component_unmounted') {
  if (!lifecycleStartedAt) {
    return;
  }

  trackAppEvent('session.ended', {
    reason,
    duration_ms: Math.max(0, Date.now() - lifecycleStartedAt),
    queue_size: queue.length,
  }, 'session');
  clearIdleTimer();
  lifecycleStartedAt = 0;
  void flushTrackingQueue();
}
