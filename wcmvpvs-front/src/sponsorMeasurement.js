const sampleInterval = 125;
const maxSampleGap = 350;
const observerThresholds = Array.from({ length: 21 }, (_, i) => i / 20);

function sponsorEventID(crypto) {
  if (crypto?.randomUUID) return crypto.randomUUID();
  if (!crypto?.getRandomValues) return null;
  const bytes = crypto.getRandomValues(new Uint8Array(16));
  bytes[6] = (bytes[6] & 15) | 64;
  bytes[8] = (bytes[8] & 63) | 128;
  const hex = Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('');
  return [hex.slice(0, 8), hex.slice(8, 12), hex.slice(12, 16), hex.slice(16, 20), hex.slice(20)].join('-');
}

function rendered(element, win) {
  if (!element.isConnected || element.closest('[inert]')) return false;
  if (!element.ownerSVGElement && element.checkVisibility) {
    return element.checkVisibility({ checkOpacity: true, checkVisibilityCSS: true });
  }
  for (let parent = element; parent; parent = parent.parentElement) {
    const style = win.getComputedStyle(parent);
    if (style.display === 'none' || style.visibility !== 'visible' || Number(style.opacity) === 0) return false;
  }
  return true;
}

function ready(element) {
  const image = element.tagName === 'IMG' ? element : element.querySelector('img');
  return !image || (image.complete && image.naturalWidth > 0);
}

function pointVisible(target, x, y, doc) {
  const hit = doc.elementFromPoint(x, y);
  if (!hit) return false;
  if (!target.ownerSVGElement) return target.contains(hit);
  if (target.contains(hit)) return true;
  // The boards are intentionally pointer-events:none. A later SVG paint hit
  // masks the sample. Hits outside the court are HTML overlays or clipping.
  const svg = target.ownerSVGElement;
  if (!svg.contains(hit) && hit !== svg) return false;
  return hit === svg || !(target.compareDocumentPosition(hit) & 4);
}

export function sponsorVisiblePercentage(element, entry, doc, win) {
  if (!entry?.isIntersecting || !rendered(element, win) || !ready(element)) return 0;
  const rect = element.getBoundingClientRect();
  const svg = element.ownerSVGElement;
  const box = svg ? element.getBBox() : rect;
  const matrix = svg ? element.getScreenCTM() : null;
  if (rect.width <= 0 || rect.height <= 0 || !box.width || !box.height || (svg && !matrix)) return 0;

  let visible = 0;
  const samples = 6;
  for (let row = 0; row < samples; row++) {
    for (let col = 0; col < samples; col++) {
      const x = box.x + box.width * (col + .5) / samples;
      const y = box.y + box.height * (row + .5) / samples;
      const screenX = matrix ? matrix.a * x + matrix.c * y + matrix.e : x;
      const screenY = matrix ? matrix.b * x + matrix.d * y + matrix.f : y;
      if (pointVisible(element, screenX, screenY, doc)) visible++;
    }
  }
  const estimate = visible / (samples * samples);
  // IO accounts for rectangular clipping exactly for our axis-aligned HTML
  // placements. SVG uses transformed local samples and browser hit testing.
  return Math.floor(100 * (svg ? estimate : Math.min(estimate, entry.intersectionRatio)));
}

export function createSponsorMeasurement({ service, scope, win = globalThis.window, doc = globalThis.document,
  clock = win?.performance, uuid = () => sponsorEventID(win?.crypto), measure = sponsorVisiblePercentage,
  measureGeometryViewability = false } = {}) {
  const targets = new Map();
  const session = uuid();
  const observer = session && win?.IntersectionObserver
    ? new win.IntersectionObserver((entries) => {
      for (const entry of entries) {
        const state = targets.get(entry.target);
        if (state) {
          state.entry = entry;
          sample(state, clock.now());
        }
      }
    }, { threshold: observerThresholds }) : null;
  let timer;

  function send(state, type, now, percentage) {
    const { info, exposure } = state;
    const id = uuid();
    if (!id) return;
    service.record({
      type, event_id: id, exposure_id: exposure.id, session_id: session,
      club_id: info.sponsor.organizationId, match_id: exposure.scope.match,
      sponsor_id: info.sponsor.sponsorId, creative_id: info.creativeId,
      placement_id: info.placement, game_id: info.game || '', measurement_method: info.method,
      timestamp: Math.floor(clock.timeOrigin + now),
      visible_duration_ms: Math.floor(exposure.total), continuous_visible_ms: Math.floor(exposure.longest),
      visibility_percentage: percentage, grant: info.sponsor.analyticsGrant,
      ...(measureGeometryViewability && info.method !== 'html_display' ? { geometry_sampled: true } : {}),
    }, exposure.scope);
  }

  function currentScope(state) {
    const current = scope();
    return current.match === state.info.sponsor.analyticsMatchId ? current : null;
  }

  function begin(state) {
    const current = currentScope(state);
    const id = uuid();
    if (!current || !id) return null;
    return { id, scope: { ...current, device: service.device() }, total: 0, longest: 0,
      continuous: 0, last: null, percentage: 0, peak: 0, viewable: false, full: false, sampledTotal: 0 };
  }

  function pause(state) {
    if (state.exposure) {
      state.exposure.last = null;
      state.exposure.continuous = 0;
      state.exposure.percentage = 0;
    }
  }

  function sample(state, now) {
    const allowed = doc.visibilityState === 'visible' && state.info.enabled !== false && currentScope(state);
    if (state.exposure && (!allowed || allowed.slug !== state.exposure.scope.slug ||
      allowed.match !== state.exposure.scope.match)) {
      pause(state);
      return;
    }
    const percentage = allowed ? measure(state.element, state.entry, doc, win) : 0;
    if (percentage > 0 && !state.exposure) {
      state.exposure = begin(state);
      if (state.exposure) send(state, 'sponsor_impression', now, percentage);
    }
    const exposure = state.exposure;
    if (!exposure) return;
    exposure.peak = Math.max(exposure.peak, percentage);
    if (percentage < 50) {
      pause(state);
      exposure.percentage = percentage;
      return;
    }
    exposure.percentage = percentage;
    const elapsed = exposure.last === null ? 0 : now - exposure.last;
    exposure.last = now;
    if (elapsed > maxSampleGap) exposure.continuous = 0;
    else {
      exposure.total = Math.min(7200000, exposure.total + elapsed);
      exposure.continuous = Math.min(exposure.total, exposure.continuous + elapsed);
      exposure.longest = Math.max(exposure.longest, exposure.continuous);
    }
    if ((state.info.method === 'html_display' || measureGeometryViewability) && !exposure.viewable && exposure.continuous >= 1000) {
      exposure.viewable = true;
      send(state, state.info.method === 'html_display' ? 'sponsor_viewable' : 'sponsor_geometry_viewable', now, percentage);
    }
    if (!exposure.full && state.info.fullDuration && exposure.continuous >= state.info.fullDuration) {
      exposure.full = true;
      send(state, 'sponsor_full_view', now, percentage);
    }
    if (measureGeometryViewability && exposure.total - exposure.sampledTotal >= 5000) {
      exposure.sampledTotal = exposure.total;
      send(state, 'sponsor_exposure_sample', now, percentage);
    }
  }

  function finish(state) {
    if (!state.exposure) return;
    send(state, 'sponsor_exposure_end', clock.now(), state.exposure.peak);
    state.exposure = null;
  }

  function track(element, value) {
    if (!observer) return;
    const creativeId = value?.creativeVariant === 'text' ? value.sponsor?.textCreativeId : value?.sponsor?.creativeId;
    const info = creativeId && value.sponsor.analyticsGrant && value.sponsor.sponsorId
      ? { method: 'html_display', ...value, creativeId } : null;
    const previous = targets.get(element);
    const key = info && JSON.stringify([info.sponsor.organizationId, info.sponsor.analyticsMatchId,
      info.sponsor.sponsorId, info.creativeId, info.placement, info.game, info.method]);
    if (previous && key === previous.key) {
      if (info.enabled === false) {
        if (info.endWhenDisabled) finish(previous);
        else pause(previous);
      }
      previous.info = info;
      return;
    }
    if (previous) {
      finish(previous);
      observer.unobserve(element);
      element.removeEventListener('click', previous.click, true);
      targets.delete(element);
    }
    if (info) {
      const state = { element, key, info, exposure: null, entry: null };
      state.click = (event) => {
        if (!event.isTrusted || !state.info.clickable || doc.visibilityState !== 'visible' || state.info.enabled === false) return;
        const now = clock.now();
        sample(state, now);
        if (state.exposure) send(state, 'sponsor_click', now, state.exposure.percentage);
        else {
          state.exposure = begin(state);
          if (state.exposure) send(state, 'sponsor_click', now, 0);
          state.exposure = null;
        }
      };
      targets.set(element, state);
      observer.observe(element);
      if (info.placement === 'home_footer') element.addEventListener('click', state.click, true);
      if (timer === undefined) {
        timer = win.setInterval(() => {
          const now = clock.now();
          for (const placement of targets.values()) sample(placement, now);
        }, sampleInterval);
      }
    } else if (!targets.size && timer !== undefined) {
      win.clearInterval(timer);
      timer = undefined;
    }
  }

  function visibility() {
    if (doc.visibilityState === 'visible') return;
    if (measureGeometryViewability) {
      for (const state of targets.values()) {
        const exposure = state.exposure;
        if (exposure && exposure.total > exposure.sampledTotal) {
          exposure.sampledTotal = exposure.total;
          send(state, 'sponsor_exposure_sample', clock.now(), exposure.percentage);
        }
      }
    }
    targets.forEach(pause);
    void service.flush({ urgent: true });
  }
  function pageHide() {
    targets.forEach(finish);
    void service.flush({ urgent: true });
  }
  if (observer) {
    doc.addEventListener('visibilitychange', visibility);
    win.addEventListener('pagehide', pageHide);
    win.addEventListener('online', service.flush);
  }
  return {
    track,
    close() {
      if (!observer) return;
      for (const element of targets.keys()) track(element, null);
      observer.disconnect();
      doc.removeEventListener('visibilitychange', visibility);
      win.removeEventListener('pagehide', pageHide);
      win.removeEventListener('online', service.flush);
      service.close();
    },
  };
}
