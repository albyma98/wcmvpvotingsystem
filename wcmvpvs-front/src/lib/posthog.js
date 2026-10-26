import posthog from 'posthog-js';
import { getOrCreateDeviceId } from '../deviceId';
import { FAN_ANALYTICS_EVENTS, FAN_ANALYTICS_SCHEMA_VERSION, sanitizeFanAnalyticsMetadata } from '../fanAnalytics';

const POSTHOG_HOST = 'https://eu.i.posthog.com';

let initialized = false;
let initFailed = false;
const FAN_CONSENT_KEY = 'arenaboostx:fan-analytics-consent:v1';
const FAN_ACTOR_PREFIX = 'arenaboostx:fan-analytics-actor:v1:';
let fanConsent;
let fanInstance = null;
const fanActors = new Map();
const fanCaptured = new Set();

function isNewFanExperience() {
  return typeof window !== 'undefined' && /^\/(?:[^/]+\/)?new(?:\/|$)/.test(window.location.pathname);
}

export function initPostHog() {
  if (isNewFanExperience()) return initFanPostHog();
  if (initialized || initFailed) {
    return initialized ? posthog : null;
  }

  const apiKey = import.meta.env.VITE_POSTHOG_KEY;
  if (!apiKey) {
    if (import.meta.env.DEV) {
      console.warn('[posthog] VITE_POSTHOG_KEY non impostata: PostHog disabilitato.');
    }
    initFailed = true;
    return null;
  }

  try {
    posthog.init(apiKey, {
      api_host: POSTHOG_HOST,
      autocapture: false,
      capture_pageview: false,
      capture_pageleave: false,
      disable_session_recording: false,
      // A legacy instance can outlive a Vue route change into the fan experience.
      before_send: (event) => isNewFanExperience() ? null : event,
      // Identità persistente (no cookie): il person_id coincide col device_id
      // dell'app (localStorage, stabile tra sessioni). Abilita utenti unici reali
      // e retention/ritornati. Segmentazione torneo via surface/tournament_slug.
      persistence: 'localStorage',
      bootstrap: { distinctID: getOrCreateDeviceId() },
      loaded: (instance) => {
        if (import.meta.env.DEV) {
          instance.debug?.(true);
        }
      },
    });
    initialized = true;
    return posthog;
  } catch (error) {
    initFailed = true;
    console.warn('[posthog] init failed', error);
    return null;
  }
}

export function getPostHog() {
  return initialized && !isNewFanExperience() ? posthog : null;
}

export function isPostHogReady() {
  return initialized && !isNewFanExperience();
}

export function getFanAnalyticsConsent() {
  try {
    const value = window.localStorage.getItem(FAN_CONSENT_KEY);
    if (value === 'accepted') return true;
    if (value === 'rejected') return false;
  } catch { /* The current choice remains usable when storage is unavailable. */ }
  return fanConsent ?? null;
}

function fanPrivacySignal() {
  const navigator = typeof window !== 'undefined' ? window.navigator : undefined;
  return navigator?.globalPrivacyControl === true || navigator?.doNotTrack === '1' || navigator?.doNotTrack === 'yes' || (typeof window !== 'undefined' && window.doNotTrack === '1');
}

export function isFanAnalyticsAllowed() {
  return getFanAnalyticsConsent() === true && !fanPrivacySignal();
}

export function prepareFanAnalytics() {
  if (initialized && isNewFanExperience()) posthog.stopSessionRecording?.();
}

export function setFanAnalyticsConsent(accepted) {
  fanConsent = accepted === true;
  try { window.localStorage.setItem(FAN_CONSENT_KEY, fanConsent ? 'accepted' : 'rejected'); } catch { /* optional persistence */ }
  if (fanConsent && isNewFanExperience() && !fanPrivacySignal()) {
    const instance = initFanPostHog();
    instance?.opt_in_capturing?.({ captureEventName: false });
  } else {
    fanInstance?.opt_out_capturing?.();
    fanInstance?.stopSessionRecording?.();
  }
  if (typeof window !== 'undefined' && typeof window.CustomEvent === 'function') {
    window.dispatchEvent(new window.CustomEvent('arenaboostx:fan-analytics-consent', { detail: { consent: fanConsent } }));
  }
  return fanConsent;
}

function fanOrganizationSlug() {
  if (typeof window === 'undefined') return '';
  const slug = window.location.pathname.split('/')[1] || '';
  if (slug === 'new' && /^\/new(?:\/|$)/.test(window.location.pathname)) return '';
  return /^[a-z0-9]+(?:-[a-z0-9]+)*$/i.test(slug) && slug.length <= 100 ? slug : '';
}

function fanActor(slug) {
  if (fanActors.has(slug)) return fanActors.get(slug);
  let actor = '';
  try { actor = window.localStorage.getItem(FAN_ACTOR_PREFIX + slug) || ''; } catch { /* optional persistence */ }
  if (!/^fan_[A-Za-z0-9_-]{16,100}$/.test(actor)) {
    actor = `fan_${globalThis.crypto?.randomUUID?.() || `${Date.now().toString(36)}_${Math.random().toString(36).slice(2)}_${Math.random().toString(36).slice(2)}`}`;
    try { window.localStorage.setItem(FAN_ACTOR_PREFIX + slug, actor); } catch { /* per-page identity fallback */ }
  }
  fanActors.set(slug, actor);
  return actor;
}

function fanProperties(event, context = {}) {
  const slug = context.organization_slug;
  const eventId = Number(context.event_id);
  if (typeof slug !== 'string' || slug.length > 100 || !/^[a-z0-9]+(?:-[a-z0-9]+)*$/i.test(slug) || !Number.isSafeInteger(eventId) || eventId <= 0) return null;
  const properties = {
    ...sanitizeFanAnalyticsMetadata(event.metadata),
    app: 'arenaboostx', surface: 'new', schema_version: FAN_ANALYTICS_SCHEMA_VERSION,
    environment: ['production', 'development', 'test'].includes(context.environment) ? context.environment : import.meta.env.MODE,
    organization_slug: slug, event_id: eventId, distinct_id: fanActor(slug),
    $process_person_profile: false, $geoip_disable: true,
  };
  if (typeof event.id === 'string' && /^[A-Za-z0-9_-]{1,100}$/.test(event.id)) properties.signal_id = event.id;
  if (typeof event.session_id === 'string' && /^[A-Za-z0-9_-]{1,100}$/.test(event.session_id)) properties.analytics_session_id = event.session_id;
  return properties;
}

function sanitizeFanPostHogEvent(event) {
  if (getFanAnalyticsConsent() !== true || fanPrivacySignal() || !event?.event?.startsWith('fan_new.')) return null;
  const name = event.event.slice('fan_new.'.length);
  if (!FAN_ANALYTICS_EVENTS.has(name)) return null;
  const input = event.properties || {};
  const properties = fanProperties({ id: input.signal_id, session_id: input.analytics_session_id, metadata: input }, input);
  if (!properties) return null;
  // SDK-added URLs, referrers, person properties and form/error payloads never leave the browser.
  return { event: `fan_new.${name}`, properties, uuid: event.uuid, timestamp: event.timestamp };
}

function initFanPostHog(organizationSlug = fanOrganizationSlug()) {
  if (getFanAnalyticsConsent() !== true || fanPrivacySignal()) return null;
  if (fanInstance) return fanInstance;
  const apiKey = import.meta.env.VITE_POSTHOG_KEY;
  const slug = organizationSlug;
  if (!apiKey || !slug) return null;
  try {
    fanInstance = posthog.init(apiKey, {
      api_host: POSTHOG_HOST,
      autocapture: false, capture_pageview: false, capture_pageleave: false,
      disable_session_recording: true, enable_recording_console_log: false,
      capture_performance: false, capture_exceptions: false, capture_heatmaps: false, capture_dead_clicks: false,
      disable_surveys: true, disable_surveys_automatic_display: true, disable_external_dependency_loading: true,
      advanced_disable_flags: true, advanced_disable_decide: true,
      save_referrer: false, store_google: false, respect_dnt: true, ip: false,
      person_profiles: 'never', persistence: 'memory', request_batching: false,
      bootstrap: { distinctID: fanActor(slug) },
      get_device_id: () => fanActor(slug),
      before_send: sanitizeFanPostHogEvent,
    }, 'fan_new');
    return fanInstance;
  } catch { return null; }
}

export function captureFanAnalytics(event, context = {}) {
  if (!isNewFanExperience() || !FAN_ANALYTICS_EVENTS.has(event?.name)) return false;
  const deduplicationKey = `${context.organization_slug}:${context.event_id}:${event.id}`;
  if (!event.id || fanCaptured.has(deduplicationKey)) return false;
  // Remember only the ID even when collection is disabled: an old observation is never replayed after consent.
  fanCaptured.add(deduplicationKey);
  if (fanCaptured.size > 4096) fanCaptured.delete(fanCaptured.values().next().value);
  if (getFanAnalyticsConsent() !== true || fanPrivacySignal()) return false;
  const properties = fanProperties(event, context);
  const instance = properties && initFanPostHog(context.organization_slug);
  if (!instance) return false;
  try {
    const options = {};
    if (typeof event.occurred_at === 'string' && Number.isFinite(Date.parse(event.occurred_at))) options.timestamp = new Date(event.occurred_at);
    instance.capture(`fan_new.${event.name}`, properties, options);
    return true;
  } catch { return false; }
}
