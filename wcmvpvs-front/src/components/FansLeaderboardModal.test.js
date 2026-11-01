import test from 'node:test';
import assert from 'node:assert/strict';
import { nextTick } from 'vue';
import { setupSfc, testVue } from '../../test/sfcSetup.js';

const flush = async () => { await nextTick(); await new Promise(resolve => setImmediate(resolve)); };

function fixture(get) {
  return setupSfc(new URL('./FansLeaderboardModal.vue', import.meta.url), {
    vue: testVue,
    '../api': {
      apiClient: { get },
      getDeviceHeaders: () => ({ 'X-Device-ID': 'device' }),
      getFanSessionHeaders: () => ({ 'X-Fan-Session': 'session' }),
      getOrganizationSlug: () => 'club',
    },
  }, { modelValue: false, eventId: 42, topList: [] });
}

test('the modal opens on event points, switches to general and keeps every fan beyond the podium', async t => {
  const calls = [];
  const { state, props, scope } = fixture(async (path, config) => {
    calls.push({ path, ...config });
    return { data: { ranking: Array.from({ length: 5 }, (_, i) => ({
      rank: i + 1, name: `Fan ${i}`, points: config.params.period === 'event' ? 10 : 100, is_me: i === 4,
    })) } };
  });
  t.after(() => scope.stop());
  assert.equal(calls.length, 0);
  props.modelValue = true;
  await flush();
  assert.equal(state.period.value, 'event');
  assert.equal(calls[0].path, '/events/42/coins-leaderboard');
  assert.deepEqual(calls[0].params, { view: 'list', period: 'event' });
  assert.equal(calls[0].headers['X-Fan-Session'], 'session');
  assert.equal(state.rows.value.length, 5);
  assert.equal(state.rows.value[4].points, 10);
  await state.selectPeriod('general');
  assert.equal(state.rows.value[4].points, 100);
  props.modelValue = false;
  await flush();
  props.modelValue = true;
  await flush();
  assert.equal(state.period.value, 'event');
  assert.equal(state.rows.value[4].points, 10);
});

test('the modal discards a late event response after changing period and refreshes the selected period live', async t => {
  let resolveEvent;
  const pending = new Promise(resolve => { resolveEvent = resolve; });
  const calls = [];
  const { state, props, scope } = fixture(async (path, config) => {
    calls.push(config.params);
    if (config.params.period === 'event') return pending;
    return { data: { ranking: [{ rank: 1, name: 'General', points: 100 }] } };
  });
  t.after(() => scope.stop());
  props.modelValue = true;
  await flush();
  await state.selectPeriod('general');
  resolveEvent({ data: { ranking: [{ rank: 1, name: 'Event', points: 10 }] } });
  await flush();
  assert.equal(state.rows.value[0].name, 'General');
  props.topList = [{ name: 'New preview' }];
  await flush();
  assert.equal(calls.at(-1).period, 'general');
});

test('the modal clears old scores when the event changes and shows request errors', async t => {
  let resolveEvent;
  const pending = new Promise(resolve => { resolveEvent = resolve; });
  const { state, props, scope } = fixture(async path => {
    if (path.includes('/42/')) return pending;
    throw new Error('offline');
  });
  t.after(() => scope.stop());
  props.modelValue = true;
  await flush();
  props.eventId = 43;
  await flush();
  resolveEvent({ data: { ranking: [{ rank: 1, name: 'Old match', points: 10 }] } });
  await flush();
  assert.deepEqual(state.rows.value, []);
  assert.match(state.error.value, /Impossibile caricare/);
  props.eventId = 0;
  await flush();
  assert.equal(state.error.value, '');
  assert.equal(state.loading.value, false);
});
