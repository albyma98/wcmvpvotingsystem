import posthog from 'posthog-js';
import { getOrCreateDeviceId } from '../deviceId';

const POSTHOG_HOST = 'https://eu.i.posthog.com';

let initialized = false;
let initFailed = false;

export function initPostHog() {
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
  return initialized ? posthog : null;
}

export function isPostHogReady() {
  return initialized;
}
