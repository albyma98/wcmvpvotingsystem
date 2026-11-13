import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';

const source = readFileSync(new URL('./merchApi.js', import.meta.url), 'utf8')
  .replace(/^import .*;\r?\n/gm, '')
  .replace(/^export /gm, '');

function fixture(statuses) {
  const requests = [];
  const delays = [];
  let grant = 'first-match-grant';
  const client = { post: async (path, body, config) => {
    requests.push({ path, body, config });
    grant = 'new-match-grant';
    const status = statuses.shift();
    if (status) throw { response: { status } };
  } };
  const track = new Function('apiClient', 'getDeviceHeaders', 'getFanSessionHeaders', 'getMatchdayObservationHeaders', 'setTimeout',
    `${source}\nreturn trackMerchEvent;`)(client, () => ({ 'X-Device-ID': 'fan-device' }), slug => ({ 'X-Fan-Session': `${slug}-fan` }),
    (slug, eventId) => slug === 'home' && eventId === 42 ? { 'X-Matchday-Grant': grant } : {},
    (callback, delay) => { delays.push(delay); callback(); });
  return { track, requests, delays };
}

test('shop event retry preserves its identifier, event and captured credentials', async () => {
  const { track, requests, delays } = fixture([503, 408, 0]);
  await track('home', 'merch_product_view', 'shirt', 'large', 42);
  assert.deepEqual(delays, [500, 1500]);
  assert.equal(requests.length, 3);
  assert.deepEqual(requests[0], requests[1]);
  assert.deepEqual(requests[1], requests[2]);
  assert.equal(requests[0].path, '/v1/clubs/home/merch/events');
  assert.equal(requests[0].body.match_id, 42);
  assert.match(requests[0].body.event_id, /^[a-f0-9-]{36}$/);
  assert.deepEqual(requests[0].config.headers, {
    'X-Organization-Slug': 'home', 'X-Device-ID': 'fan-device', 'X-Fan-Session': 'home-fan', 'X-Matchday-Grant': 'first-match-grant',
  });
  assert.equal(requests[0].config.fetchOptions.keepalive, true);
});

test('shop events stop retrying when rejected by validation or rate limits', async () => {
  for (const status of [400, 429]) {
    const { track, requests, delays } = fixture([status]);
    await track('home', 'merch_shop_opened', undefined, undefined, 42);
    assert.equal(requests.length, 1);
    assert.deepEqual(delays, []);
  }
});
