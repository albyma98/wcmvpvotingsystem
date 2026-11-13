import { apiClient, getDeviceHeaders, getFanSessionHeaders, getMatchdayObservationHeaders, resolveApiUrl } from '../api';
import { stripeCheckoutUrl } from './shopModel';

const merchPath = slug => `/v1/clubs/${encodeURIComponent(slug)}/merch`;
export const merchMediaUrl = url => url.startsWith('/api/') ? resolveApiUrl(url.slice(4)) : url;
const options = (slug, config = {}) => ({
  timeout: 15000,
  ...config,
  headers: {
    'X-Organization-Slug': slug,
    ...getDeviceHeaders(),
    ...getFanSessionHeaders(slug),
    ...config.headers,
  },
});

export const fetchMerchCatalog = async (slug, signal) =>
  (await apiClient.get(merchPath(slug), options(slug, { signal }))).data;

export async function fetchMerchVisibility(slug, signal, eventId) {
  const { data } = await apiClient.get(merchPath(slug), {
    params: { summary: 1, ...(eventId > 0 ? { match_id: eventId } : {}) },
    signal,
    timeout: 7000,
  });
  return Boolean(data?.settings?.enabled && data.settings.visible);
}

export const fetchMerchQuote = async (slug, payload, signal) =>
  (await apiClient.post(`${merchPath(slug)}/quote`, payload, options(slug, { signal }))).data;

export const createMerchOrder = async (slug, payload) =>
  (await apiClient.post(`${merchPath(slug)}/orders`, payload, options(slug, { timeout: 25000 }))).data;

export const fetchFanMerchOrders = async (slug, signal) =>
  (await apiClient.get(`${merchPath(slug)}/orders`, options(slug, { signal }))).data.orders;

export async function fetchMerchOrder(slug, id, token, signal) {
  const { data } = await apiClient.get(`${merchPath(slug)}/orders/${encodeURIComponent(id)}`, options(slug, {
    signal,
    headers: token ? { 'X-Merch-Access': token } : {},
  }));
  const checkoutUrl = stripeCheckoutUrl(data.checkout_url);
  return { ...data.order, checkout_url: checkoutUrl };
}

export function requestMerchRecovery(slug, id, token, step, payload, signal) {
  return apiClient.post(`${merchPath(slug)}/orders/${encodeURIComponent(id)}/recovery/${step}`, payload, options(slug, {
    signal,
    headers: { 'X-Merch-Access': token },
  }));
}

export function trackMerchEvent(slug, type, productId, variantId, matchId) {
  if (!globalThis.crypto?.randomUUID) return;
  const payload = {
    event_id: globalThis.crypto.randomUUID(),
    type,
    product_id: productId,
    ...(variantId ? { variant_id: variantId } : {}),
    ...(matchId > 0 ? { match_id: matchId } : {}),
  };
  const config = options(slug, {
    timeout: 8000,
    headers: getMatchdayObservationHeaders(slug, matchId),
    ...(typeof globalThis.fetch === 'function' && { adapter: ['fetch', 'xhr'], fetchOptions: { keepalive: true } }),
  });
  return sendMerchEvent(`${merchPath(slug)}/events`, payload, config);
}

async function sendMerchEvent(path, payload, config) {
  for (const delay of [0, 500, 1500]) {
    if (delay > 0) await new Promise(resolve => setTimeout(resolve, delay));
    try {
      await apiClient.post(path, payload, config);
      return;
    } catch (error) {
      const status = error?.response?.status;
      if (status && status !== 408 && status < 500) return;
    }
  }
}

export function merchRequestError(error, fallback) {
  const message = error?.response?.data?.message;
  return typeof message === 'string' && message ? message : fallback;
}
