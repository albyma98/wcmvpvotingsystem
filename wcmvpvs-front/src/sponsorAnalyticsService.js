export const sponsorAnalyticsKey = Symbol('sponsorAnalytics');

export function sponsorAnalyticsFields(item, matchId) {
  return {
    sponsorId: Number(item?.id) || null,
    organizationId: Number(item?.organization_id) || null,
    creativeId: item?.creative_id ?? null,
    textCreativeId: item?.analytics_text_creative_id ?? null,
    analyticsGrant: item?.analytics_grant,
    analyticsMatchId: matchId,
  };
}

export function existingSponsorDevice(win) {
  try {
    const value = win.localStorage.getItem('wcmvpvs:device_id') || '';
    return /^[a-zA-Z0-9_-]{16,80}$/.test(value) ? value : '';
  } catch {
    return '';
  }
}

export function createSponsorAnalyticsService({ url, win = globalThis.window, fetchRequest = globalThis.fetch,
  timers = globalThis, device = () => existingSponsorDevice(win) } = {}) {
  const batchSize = 24;
  const queue = [];
  let timer;
  let inFlight;
  let delay = 5000;
  let closed = false;

  function schedule() {
    if (timer || inFlight || !queue.length) return;
    timer = timers.setTimeout(() => {
      timer = undefined;
      void flush();
    }, delay);
  }

  function record(event, scope) {
    if (closed || !scope?.slug || queue.length >= 480) return;
    queue.push({ event, scope });
    schedule();
  }

  function nextBatch() {
    const first = queue[0];
    if (!first) return [];
    const batch = [];
    for (const item of queue) {
      if (item.scope.slug !== first.scope.slug || item.scope.match !== first.scope.match ||
        item.scope.device !== first.scope.device || batch.length === batchSize) break;
      batch.push(item);
    }
    return batch;
  }

  function send(batch) {
    const { scope } = batch[0];
    const path = '/sponsors/analytics/events?organization_slug=' + encodeURIComponent(scope.slug);
    return fetchRequest(url(path), {
      method: 'POST', keepalive: true, credentials: 'omit',
      headers: { 'Content-Type': 'text/plain;charset=UTF-8' },
      body: JSON.stringify({ match_id: scope.match, device_id: scope.device, events: batch.map(({ event }) => event) }),
    });
  }

  async function flush({ urgent = false } = {}) {
    if (timer) {
      timers.clearTimeout(timer);
      timer = undefined;
    }
    if (!queue.length || !fetchRequest) return;
    if (inFlight) {
      // Resend the ordered prefix on page exit when a newer event is still
      // queued behind a pending response. The server accepts each ID once.
      if (urgent && queue.length > inFlight.length) {
        try { await send(nextBatch()); } catch { /* Best effort on departure. */ }
      }
      return;
    }
    const batch = nextBatch();
    inFlight = batch;
    let retry = true;
    try {
      const response = await send(batch);
      retry = response.status >= 500 || [408, 429].includes(response.status);
      if (!retry) queue.splice(0, batch.length);
      if (response.status === 429) {
        delay = Math.max(delay, Number(response.headers?.get('Retry-After')) * 1000 || 0);
      }
    } catch {
      // Network failures keep their IDs and order for the next attempt.
    } finally {
      inFlight = undefined;
      delay = retry ? Math.min(60000, delay * 2) : 5000;
      if (!retry && (closed || queue.length >= batchSize)) void flush();
      else schedule();
    }
  }

  return {
    record, flush, device,
    close() { closed = true; void flush({ urgent: true }); },
  };
}
