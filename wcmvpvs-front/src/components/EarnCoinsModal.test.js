import test from 'node:test';
import assert from 'node:assert/strict';
import { setupSfc, testVue } from '../../test/sfcSetup.js';

function fixture(t, get) {
  const setup = setupSfc(new URL('./EarnCoinsModal.vue', import.meta.url), {
    vue: testVue, '../api': { apiClient: { get } }, '../eventTracking': { trackAppEvent() {} },
    './CoinCollectAnimation.vue': {}, './ReactionTestGame.vue': {}, './QuickQuizGame.vue': {},
    './minigames/TapChallenge.vue': {}, './minigames/MemoryFlashGame.vue': {}, './minigames/SponsorRushGame.vue': {},
  }, { modelValue: false, eventId: 42, walletCoins: 0 });
  t.after(() => setup.scope.stop());
  return setup;
}

test('the legacy game catalog shows only the server-approved games and blocks direct launches', async t => {
  const { state } = fixture(t, async path => {
    assert.equal(path, '/events/42/minigames');
    return { data: ['tap', 'memory-flash'] };
  });
  assert.deepEqual(state.earnOptions.value, []);
  await state.loadMinigames();
  assert.deepEqual(state.earnOptions.value.map(game => game.id), ['tap', 'memory-flash']);
  state.openGame({ id: 'reaction' });
  assert.equal(state.activeGame.value, null);
  state.openGame(state.earnOptions.value[0]);
  assert.equal(state.activeGame.value.id, 'tap');
});

test('failed and stale catalog requests cannot expose games for another event', async t => {
  let resolveOld;
  let calls = 0;
  const { state, props } = fixture(t, async () => {
    calls += 1;
    if (calls === 1) return new Promise(resolve => { resolveOld = resolve; });
    throw new Error('offline');
  });
  const old = state.loadMinigames();
  props.eventId = 43;
  await state.loadMinigames();
  resolveOld({ data: ['tap'] });
  await old;
  assert.deepEqual(state.earnOptions.value, []);
  assert.match(state.catalogError.value, /Impossibile caricare/);
});
