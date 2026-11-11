import test from 'node:test';
import assert from 'node:assert/strict';
import * as vue from 'vue';
import { setupSfc } from '../../../test/sfcSetup.js';

test('commercial sponsor filters use the authenticated club and render unavailable metrics as N/D', async (t) => {
  const requests = [];
  const total = {
    impressions: 2, geometric_impressions: 2, viewable_impressions: null, viewability_rate: null,
    reach: 1, frequency: null, total_visible_ms: 1800, average_visible_ms: 900, duration_samples: 2,
    clicks: null, unique_clickers: null, ctr: null, full_view_rate: null, conversions: null,
  };
  const { state, scope } = setupSfc(new URL('./AdminSponsorAnalytics.vue', import.meta.url), {
    vue: { ...vue, onMounted() {}, onUnmounted() {} },
    '../../api': { apiClient: { get: async (path, config) => {
      requests.push({ path, config });
      return { data: { total, sponsors: [{ sponsor_id: 12, sponsor_name: 'Partner storico' }], trend: [] } };
    } } },
  }, { authHeaders: { headers: { 'X-Organization-Slug': 'club-a' } }, events: [], sponsors: [{ id: 5, name: 'Partner attivo' }] });
  t.after(() => scope.stop());

  await state.load();
  assert.deepEqual(state.sponsorOptions.value.map(({ id }) => id), [5, 12]);
  state.selection.sponsor = '12';
  state.selection.match = '42';
  state.season.value = 2025;
  state.setSeason();
  await state.load();
  assert.equal(requests[1].path, '/admin/sponsors/analytics');
  assert.equal(requests[1].config.headers['X-Organization-Slug'], 'club-a');
  assert.deepEqual(requests[1].config.params, { from: '2025-07-01', to: '2026-06-30', sponsor_id: '12', match_id: '42' });
  const metrics = Object.fromEntries(state.overview.value.map(({ label, value }) => [label, value]));
  assert.equal(metrics['Impression misurabili'], '2');
  assert.equal(metrics['Impression viewable'], 'N/D');
  assert.equal(metrics['Tempo totale'], '1,8 s');
  assert.equal(metrics.CTR, 'N/D');
  assert.equal(metrics['Conversioni attribuite'], 'N/D');
});
