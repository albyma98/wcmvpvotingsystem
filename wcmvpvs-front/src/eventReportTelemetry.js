// Dedicated measurement for the /:slug/new experience. Never mirrors legacy analytics.
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

export function createEventReportTelemetry({ eventId, send, identity = () => ({}), window: win = globalThis.window, document: doc = globalThis.document, now = Date.now, id = randomId } = {}) {
  if (!Number.isInteger(eventId) || eventId <= 0 || typeof send !== 'function') {
    return { record() {}, start() {}, observeSponsor() {}, stop() {}, flush: async () => {} };
  }
  let sessionId = id();
  try {
    const previous = JSON.parse(win?.sessionStorage?.getItem(SESSION_KEY) || 'null');
    if (previous?.id && now() - previous.last < 30 * 60000 && now() >= previous.last) sessionId = previous.id;
    win?.sessionStorage?.setItem(SESSION_KEY, JSON.stringify({ id: sessionId, last: now() }));
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
  const sponsorObservers = new Map();
  const sponsorStorageKey = `arenaboostx:report-views:${eventId}:${sessionId}`;
  let measuredSponsors = new Set();
  try { measuredSponsors = new Set(JSON.parse(win?.sessionStorage?.getItem(sponsorStorageKey) || '[]').slice(0, 100)); } catch { /* optional */ }

  function schedule() {
    if (!queue.length || timer || !win || (stopped && failures >= 5)) return;
    timer = win.setTimeout(() => { timer = undefined; void flush(); }, Math.min(30000, 3000 * 2 ** failures));
  }

  function record(name, metadata = {}) {
    if (stopped) return;
    // Capture credentials at observation time: login cannot reattribute a queued guest batch.
    let credentials = {};
    try { credentials = { ...identity() }; } catch { return; }
    queue.push({ event: { id: id(), name, session_id: sessionId, occurred_at: new Date(now()).toISOString(), metadata }, credentials });
    if (queue.length > MAX_QUEUE) queue.splice(0, queue.length - MAX_QUEUE);
    try { win?.sessionStorage?.setItem(SESSION_KEY, JSON.stringify({ id: sessionId, last: now() })); } catch { /* optional */ }
    if (queue.length >= BATCH_SIZE) void flush(); else schedule();
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
    if (wasVisible && seconds > 0) record('active_time', { seconds, started_at: new Date(end - seconds * 1000).toISOString(), ended_at: new Date(end).toISOString() });
    lastTick = time;
    wasVisible = visible;
  }
  function activity() {
    if (now() - lastActivity >= IDLE_MS) tick();
    lastActivity = now();
  }
  function visibility() {
    tick();
    for (const observer of sponsorObservers.values()) observer.visibility();
    if (doc?.visibilityState === 'hidden') void flush();
  }
  function start(metadata = {}) {
    if (started || stopped) return;
    started = true;
    lastTick = now(); lastActivity = now(); wasVisible = doc?.visibilityState !== 'hidden';
    record('page_view', metadata);
    for (const name of ['pointerdown', 'keydown', 'touchstart', 'scroll']) win?.addEventListener(name, activity, { passive: true });
    doc?.addEventListener('visibilitychange', visibility);
    win?.addEventListener('pagehide', onPageHide);
    heartbeat = win?.setInterval(tick, HEARTBEAT_MS);
  }
  function onPageHide() { tick(); void flush({ keepalive: true }); }

  function observeSponsor(element, metadata, signalName = 'sponsor_view') {
    if (!element || stopped || !metadata?.sponsor_id || !win?.IntersectionObserver) return;
    const key = `${signalName}:${metadata.sponsor_id}:${metadata.placement || 'home'}`;
    if (sponsorObservers.has(element)) return;
    let visible = false;
    let dwellTimer;
    function cancel() { if (dwellTimer) win.clearTimeout(dwellTimer); dwellTimer = undefined; }
    function check() {
      if (!visible || doc?.visibilityState === 'hidden' || measuredSponsors.has(key)) { cancel(); return; }
      if (dwellTimer) return;
      dwellTimer = win.setTimeout(() => {
        dwellTimer = undefined;
        if (!visible || doc?.visibilityState === 'hidden' || stopped || measuredSponsors.has(key)) return;
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
    if (started) tick();
    stopped = true;
    for (const name of ['pointerdown', 'keydown', 'touchstart', 'scroll']) win?.removeEventListener(name, activity);
    doc?.removeEventListener('visibilitychange', visibility);
    win?.removeEventListener('pagehide', onPageHide);
    if (heartbeat) win.clearInterval(heartbeat);
    for (const observer of sponsorObservers.values()) observer.stop();
    sponsorObservers.clear();
    void flush({ keepalive: true });
  }
  return { record, start, observeSponsor, stop, flush };
}
