import test from 'node:test';
import assert from 'node:assert/strict';
import { setupSfc, testVue, testWindow } from '../../../test/sfcSetup.js';

function game(t, props = {}) {
  const setup = setupSfc(new URL('./SponsorRushGame.vue', import.meta.url), {
    vue: testVue, '../../api': { apiClient: { get: async () => ({ data: [] }) } },
  }, { completionReward: null, alreadyRewarded: false, ...props }, { window: testWindow() });
  t.after(() => setup.scope.stop());
  return setup;
}

test('finishing Sponsor Rush with zero logos requests the fixed reward once', async (t) => {
  const setup = game(t, { completionReward: 12 });
  setup.state.finishRound();
  assert.equal(setup.state.rewardCoins.value, 12);
  assert.equal(setup.state.won.value, false);
  await setup.state.claimReward();
  await setup.state.claimReward();
  assert.equal(setup.events.length, 1);
  assert.equal(setup.events[0][1].coins, 12);
});

test('already rewarded players exit without another claim and legacy scoring keeps its variable amount', (t) => {
  const completed = game(t, { completionReward: 12, alreadyRewarded: true });
  completed.state.phase.value = 'result';
  completed.state.onPrimaryAction();
  assert.deepEqual(completed.events, [['exit']]);
  const legacy = game(t);
  legacy.state.score.value = 2;
  assert.equal(legacy.state.rewardCoins.value, 2);
  legacy.state.score.value = 5;
  assert.equal(legacy.state.rewardCoins.value, 12);
  assert.equal(legacy.state.hasFixedReward.value, false);
});
