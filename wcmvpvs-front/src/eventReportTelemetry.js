import { FAN_ANALYTICS_EVENTS, FAN_ANALYTICS_SCHEMA_VERSION, sanitizeFanAnalyticsMetadata } from './fanAnalytics.js';

// Dedicated measurement for /:slug/new. Optional PostHog mirroring happens once at observation time.
const SESSION_KEY = 'arenaboostx:report-session:v1';
const MAX_QUEUE = 120;
const BATCH_SIZE = 20;
const HEARTBEAT_MS = 15000;
const IDLE_MS = 60000;

function randomId() {
  return globalThis.crypto?.randomUUID?.() || `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}-${Math.random().toString(36).slice(2)}`;
}

export function reportSource(search = '') {
  const source = new URLSearchParams(search).get('source');
  return source === 'arena' || source === 'remote' ? source : 'unknown';
}

export function createReportSignalSender({ apiClient, resolveApiUrl, fetch: fetchRequest = globalThis.fetch?.bind(globalThis) }) {
  return (eventId, events, headers, { keepalive = false } = {}) => {
    const path = `/events/${eventId}/report-signals`;
    if (keepalive && typeof fetchRequest === 'function') {
      // sendBeacon cannot carry the captured device, fan session and tenant headers.
      return fetchRequest(resolveApiUrl(path), {
        method: 'POST', credentials: 'include', keepalive: true,
        headers: { ...headers, 'Content-Type': 'application/json' },
        body: JSON.stringify({ events }),
      }).then((response) => {
        if (!response.ok) throw new Error(`Report delivery failed (${response.status})`);
      });
    }
    return apiClient.post(path, { events }, { headers: { ...headers }, timeout: 10000 });
  };
}

export function createEventReportTelemetry({ eventId, send, identity = () => ({}), context = () => ({}), mirror = () => {}, canPersistVisits = () => false, window: win = globalThis.window, document: doc = globalThis.document, now = Date.now, id = randomId } = {}) {
  if (!Number.isInteger(eventId) || eventId <= 0 || typeof send !== 'function') {
    return { record() {}, start() {}, observeSponsor() {}, observeVisibility() {}, resetMirrorClock() {}, setMirrorConsent() {}, stop() {}, flush: async () => {} };
  }
  function currentContext() { try { return context() || {}; } catch { return {}; } }
  const organization = String(currentContext().organization_slug || '');
  const sessionKey = `${SESSION_KEY}:${organization}:${eventId}`;
  const visitsKey = `arenaboostx:report-visits:v1:${organization}:${eventId}`;
  let sessionId = id();
  try {
    const previous = JSON.parse(win?.sessionStorage?.getItem(sessionKey) || 'null');
    if (previous?.id && now() - previous.last < 30 * 60000 && now() >= previous.last) sessionId = previous.id;
    win?.sessionStorage?.setItem(sessionKey, JSON.stringify({ id: sessionId, last: now() }));
  } catch { /* storage is optional */ }
  let queue = [];
  let timer;
  let heartbeat;
  let pending;
  let inFlight = [];
  const keepaliveInFlight = new Set();
  let started = false;
  let stopped = false;
  let failures = 0;
  let lastActivity = now();
  let lastTick = now();
  let wasVisible = doc?.visibilityState !== 'hidden';
  let visitStartedAt = now();
  let activeSeconds = 0;
  let visibleMs = 0;
  let visibleStartedAt = wasVisible ? now() : null;
  let leaveRecorded = false;
  let visitCount = 0;
  let persistentVisitRecorded = false;
  let externalVisitRecorded = false;
  let mirrorEnabled = true;
  let mirrorStartedAt = now();
  let mirrorActiveSeconds = 0;
  let mirrorVisibleMs = 0;
  let mirrorVisibleStartedAt = wasVisible ? now() : null;
  const sponsorObservers = new Map();
  const sponsorStorageKey = `arenaboostx:report-views:${organization}:${eventId}:${sessionId}`;
  let measuredSponsors = new Set();
  try { measuredSponsors = new Set(JSON.parse(win?.sessionStorage?.getItem(sponsorStorageKey) || '[]').slice(0, 100)); } catch { /* optional */ }

  function schedule() {
    if (!queue.length || timer || !win || (stopped && failures >= 5)) return;
    timer = win.setTimeout(() => { timer = undefined; void flush(); }, Math.min(30000, 3000 * 2 ** failures));
  }

  function record(name, metadata = {}) {
    if (stopped || !FAN_ANALYTICS_EVENTS.has(name)) return;
    const snapshot = currentContext();
    const event = { id: id(), name, session_id: sessionId, occurred_at: new Date(now()).toISOString(), metadata: sanitizeFanAnalyticsMetadata({ ...snapshot, ...metadata, schema_version: FAN_ANALYTICS_SCHEMA_VERSION }) };
    // Only the external copy uses the current consent window. Internal report timing stays intact.
    const mirroredEvent = mirrorEvent(event);
    if (mirroredEvent) {
      // The external mirror must not receive credentials and must never run during retries.
      try { mirror(mirroredEvent, snapshot); } catch { /* Analytics must never interrupt the fan flow. */ }
    }
    // Capture credentials at observation time: login cannot reattribute a queued guest batch.
    let credentials = {};
    try { credentials = { ...identity() }; } catch { return event; }
    queue.push({ event, credentials });
    if (queue.length > MAX_QUEUE) queue.splice(0, queue.length - MAX_QUEUE);
    try { win?.sessionStorage?.setItem(sessionKey, JSON.stringify({ id: sessionId, last: now() })); } catch { /* optional */ }
    if (queue.length >= BATCH_SIZE) void flush(); else schedule();
    return event;
  }

  function mirrorOnly(name, metadata = {}) {
    const snapshot = currentContext();
    const event = { id: id(), name, session_id: sessionId, occurred_at: new Date(now()).toISOString(), metadata: sanitizeFanAnalyticsMetadata({ ...snapshot, ...metadata, schema_version: FAN_ANALYTICS_SCHEMA_VERSION }) };
    const external = mirrorEvent(event);
    if (external) { try { mirror(external, snapshot); } catch { /* optional analytics */ } }
  }

  // A new consent window is an external baseline, not a second internal page view.
  function setMirrorConsent(enabled, metadata = {}) {
    resetMirrorClock(enabled);
    if (!mirrorEnabled || !started || stopped || externalVisitRecorded) return;
    let persist = false;
    try { persist = canPersistVisits(); } catch { /* optional storage */ }
    if (persist && !persistentVisitRecorded) {
      let storage;
      try { storage = win?.localStorage; } catch { /* optional persistence */ }
      visitCount = nextVisitCount(storage);
      persistentVisitRecorded = true;
    }
    mirrorOnly('visit_started', { visit_count: visitCount, phase: 'active' });
    if (visitCount > 1) mirrorOnly('visit_returned', { visit_count: visitCount });
    mirrorOnly('page_view', metadata);
    externalVisitRecorded = true;
  }

  function resetMirrorClock(enabled = true) {
    mirrorEnabled = enabled === true;
    mirrorStartedAt = now();
    mirrorActiveSeconds = 0;
    mirrorVisibleMs = 0;
    mirrorVisibleStartedAt = mirrorEnabled && doc?.visibilityState !== 'hidden' ? now() : null;
  }
  function mirrorEvent(event) {
    if (!mirrorEnabled) return null;
    const metadata = { ...event.metadata };
    const elapsed = Math.max(0, now() - mirrorStartedAt);
    if (event.name === 'active_time') {
      const end = Date.parse(metadata.ended_at);
      const start = Math.max(Date.parse(metadata.started_at), mirrorStartedAt);
      if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) return null;
      metadata.seconds = Math.min(15, (end - start) / 1000);
      metadata.started_at = new Date(end - metadata.seconds * 1000).toISOString();
      mirrorActiveSeconds += metadata.seconds;
    }
    for (const key of ['duration_ms', 'elapsed_ms', 'latency_ms', 'time_to_first_action_ms']) {
      if (typeof metadata[key] === 'number') metadata[key] = Math.min(metadata[key], elapsed);
    }
    if (typeof metadata.active_seconds === 'number') metadata.active_seconds = mirrorActiveSeconds;
    if (typeof metadata.visible_seconds === 'number') {
      metadata.visible_seconds = (mirrorVisibleMs + (mirrorVisibleStartedAt === null ? 0 : Math.max(0, now() - mirrorVisibleStartedAt))) / 1000;
    }
    return { ...event, metadata };
  }

  async function flush({ keepalive = false } = {}) {
    if (keepalive) {
      // Also resend a normal request already in flight: navigation may cancel it.
      // Both paths use the same IDs, so the server safely accepts either response.
      const snapshot = [...inFlight, ...queue].filter(({ event }) => !keepaliveInFlight.has(event.id));
      const requests = [];
      while (snapshot.length) {
        const key = JSON.stringify(snapshot[0].credentials);
        const batch = [];
        while (snapshot.length && batch.length < BATCH_SIZE && JSON.stringify(snapshot[0].credentials) === key) batch.push(snapshot.shift());
        const ids = new Set(batch.map(({ event }) => event.id));
        for (const eventId of ids) keepaliveInFlight.add(eventId);
        requests.push(Promise.resolve().then(() => send(eventId, batch.map(({ event }) => event), batch[0].credentials, { keepalive: true })).then(() => {
          queue = queue.filter(({ event }) => !ids.has(event.id));
        }).catch(() => {
          // Leave observations queued for the normal Axios retry path.
        }).finally(() => {
          for (const eventId of ids) keepaliveInFlight.delete(eventId);
          schedule();
        }));
      }
      return Promise.all(requests);
    }
    if (pending) return pending;
    if (timer) { win.clearTimeout(timer); timer = undefined; }
    if (!queue.length) return;
    const key = JSON.stringify(queue[0].credentials);
    const batch = [];
    while (queue.length && batch.length < BATCH_SIZE && JSON.stringify(queue[0].credentials) === key) batch.push(queue.shift());
    inFlight = batch;
    pending = Promise.resolve().then(() => send(eventId, batch.map((entry) => entry.event), batch[0].credentials)).then(() => {
      failures = 0;
    }).catch(() => {
      failures += 1;
      queue = [...batch, ...queue].slice(0, MAX_QUEUE);
    }).finally(() => { pending = undefined; inFlight = []; schedule(); });
    return pending;
  }

  function tick() {
    const time = now();
    const visible = doc?.visibilityState !== 'hidden';
    const end = Math.min(time, lastActivity + IDLE_MS);
    const seconds = Math.min(15, Math.max(0, end - lastTick) / 1000);
    if (wasVisible && seconds > 0) {
      activeSeconds += seconds;
      record('active_time', { seconds, started_at: new Date(end - seconds * 1000).toISOString(), ended_at: new Date(end).toISOString() });
    }
    lastTick = time;
    wasVisible = visible;
  }
  function activity() {
    if (now() - lastActivity >= IDLE_MS) tick();
    lastActivity = now();
  }
  function visibility() {
    const previouslyVisible = wasVisible;
    tick();
    updateVisibleClock();
    if (previouslyVisible !== wasVisible) record(wasVisible ? 'page_visible' : 'page_hidden', visitMetadata());
    for (const observer of sponsorObservers.values()) observer.visibility();
    if (doc?.visibilityState === 'hidden') void flush();
  }
  function start(metadata = {}) {
    if (started || stopped) return;
    started = true;
    lastTick = now(); lastActivity = now(); wasVisible = doc?.visibilityState !== 'hidden';
    visitStartedAt = now(); visibleStartedAt = wasVisible ? now() : null;
    resetMirrorClock(mirrorEnabled);
    recordVisit();
    record('page_view', metadata);
    for (const name of ['pointerdown', 'keydown', 'touchstart', 'scroll']) win?.addEventListener(name, activity, { passive: true, capture: name === 'scroll' });
    doc?.addEventListener('visibilitychange', visibility);
    win?.addEventListener('pagehide', onPageHide);
    win?.addEventListener('pageshow', onPageShow);
    heartbeat = win?.setInterval(tick, HEARTBEAT_MS);
  }
  function updateVisibleClock() {
    if (visibleStartedAt !== null) visibleMs += Math.max(0, now() - visibleStartedAt);
    visibleStartedAt = doc?.visibilityState !== 'hidden' ? now() : null;
    if (mirrorVisibleStartedAt !== null) mirrorVisibleMs += Math.max(0, now() - mirrorVisibleStartedAt);
    mirrorVisibleStartedAt = mirrorEnabled && doc?.visibilityState !== 'hidden' ? now() : null;
  }
  function visitMetadata() {
    return { duration_ms: Math.max(0, now() - visitStartedAt), active_seconds: activeSeconds, visible_seconds: visibleMs / 1000 };
  }
  function nextVisitCount(storage) {
    let count = 0;
    try {
      const previous = storage?.getItem(visitsKey);
      if (previous != null) count = Math.max(0, Math.min(1e9 - 1, Number(previous) || 0));
    } catch { /* optional */ }
    count += 1;
    try { storage?.setItem(visitsKey, String(count)); } catch { /* optional */ }
    return count;
  }
  function recordVisit() {
    let storage;
    try { storage = win?.sessionStorage; } catch { /* optional storage */ }
    persistentVisitRecorded = false;
    try { if (canPersistVisits()) { storage = win?.localStorage; persistentVisitRecorded = true; } } catch { /* same-tab fallback */ }
    visitCount = nextVisitCount(storage);
    externalVisitRecorded = persistentVisitRecorded;
    record('visit_started', { visit_count: visitCount });
    if (visitCount > 1) record('visit_returned', { visit_count: visitCount });
  }
  function recordLeave() {
    if (leaveRecorded || !started) return;
    leaveRecorded = true;
    record('page_leave', visitMetadata());
  }
  function onPageHide() {
    tick(); updateVisibleClock(); recordLeave();
    visibleStartedAt = null;
    mirrorVisibleStartedAt = null;
    wasVisible = false;
    void flush({ keepalive: true });
  }
  function onPageShow(event) {
    if (!event?.persisted || stopped) return;
    leaveRecorded = false;
    visitStartedAt = now(); visibleMs = 0; activeSeconds = 0;
    lastTick = now(); lastActivity = now(); wasVisible = doc?.visibilityState !== 'hidden';
    visibleStartedAt = wasVisible ? now() : null;
    resetMirrorClock(mirrorEnabled);
    recordVisit();
  }

  function observeSponsor(element, metadata, signalName = 'sponsor_view') {
    if (!metadata?.sponsor_id) return;
    return observeVisibility(element, signalName, metadata, { onceKey: `${signalName}:${metadata.sponsor_id}:${metadata.placement || 'home'}` });
  }
  function observeVisibility(element, signalName, metadata = {}, { onceKey } = {}) {
    if (!element || stopped || !FAN_ANALYTICS_EVENTS.has(signalName) || !win?.IntersectionObserver) return;
    const key = onceKey || `${signalName}:${metadata.cta || metadata.section || metadata.placement || 'home'}`;
    if (sponsorObservers.has(element)) return;
    let visible = false;
    let dwellTimer;
    function cancel() { if (dwellTimer) win.clearTimeout(dwellTimer); dwellTimer = undefined; }
    function check() {
      if (!visible || doc?.visibilityState === 'hidden' || element.closest?.('[inert], [aria-hidden="true"]') || measuredSponsors.has(key)) { cancel(); return; }
      if (dwellTimer) return;
      dwellTimer = win.setTimeout(() => {
        dwellTimer = undefined;
        if (!visible || doc?.visibilityState === 'hidden' || element.closest?.('[inert], [aria-hidden="true"]') || stopped || measuredSponsors.has(key)) return;
        measuredSponsors.add(key);
        try { win.sessionStorage?.setItem(sponsorStorageKey, JSON.stringify([...measuredSponsors].slice(-100))); } catch { /* optional */ }
        record(signalName, metadata);
        observer.disconnect();
      }, 1000);
    }
    const observer = new win.IntersectionObserver((entries) => {
      for (const entry of entries) if (entry.target === element) { visible = entry.isIntersecting && entry.intersectionRatio >= .5; check(); }
    }, { threshold: [0, .5, 1] });
    observer.observe(element);
    sponsorObservers.set(element, { visibility: check, stop() { cancel(); observer.disconnect(); } });
  }
  function stop() {
    if (stopped) return;
    if (started) { tick(); updateVisibleClock(); recordLeave(); }
    stopped = true;
    for (const name of ['pointerdown', 'keydown', 'touchstart', 'scroll']) win?.removeEventListener(name, activity, name === 'scroll');
    doc?.removeEventListener('visibilitychange', visibility);
    win?.removeEventListener('pagehide', onPageHide);
    win?.removeEventListener('pageshow', onPageShow);
    if (heartbeat) win.clearInterval(heartbeat);
    for (const observer of sponsorObservers.values()) observer.stop();
    sponsorObservers.clear();
    void flush({ keepalive: true });
  }
  return { record, start, observeSponsor, observeVisibility, resetMirrorClock, setMirrorConsent, stop, flush, session_id: sessionId };
}
