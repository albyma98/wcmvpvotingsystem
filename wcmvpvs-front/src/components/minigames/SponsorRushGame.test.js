import test from 'node:test';
import assert from 'node:assert/strict';
import { setupSfc, testVue, testWindow } from '../../../test/sfcSetup.js';
import * as journey from '../../fanJourney.js';

function game(t, props = {}) {
  const setup = setupSfc(new URL('./SponsorRushGame.vue', import.meta.url), {
    vue: testVue, '../../api': { apiClient: { get: async () => ({ data: [] }) } },
    '../../fanJourney': journey,
  }, { completionReward: null, ...props }, { window: testWindow() });
  t.after(() => setup.scope.stop());
  return setup;
}

test('finishing Sponsor Rush with zero logos requests the fixed reward once', async (t) => {
  const setup = game(t, { completionReward: 12 });
  setup.state.phase.value = 'playing';
  setup.state.finishRound();
  setup.state.finishRound();
  assert.equal(setup.state.rewardCoins.value, 12);
  assert.equal(setup.state.won.value, false);
  await setup.state.claimReward();
  await setup.state.claimReward();
  assert.deepEqual(setup.events.map(([name]) => name), ['round-completed', 'claim']);
  assert.equal(setup.events[1][1].coins, 12);
});

test('new rounds can claim again and legacy scoring keeps its variable amount', (t) => {
  const completed = game(t, { completionReward: 12 });
  completed.state.phase.value = 'result';
  completed.state.onPrimaryAction();
  const firstRound = completed.events[0][1].roundId;
  completed.state.resetRound();
  completed.state.phase.value = 'result';
  completed.state.onPrimaryAction();
  assert.equal(completed.events.length, 2);
  assert.notEqual(completed.events[1][1].roundId, firstRound);
  const legacy = game(t);
  legacy.state.score.value = 2;
  assert.equal(legacy.state.rewardCoins.value, 2);
  legacy.state.score.value = 5;
  assert.equal(legacy.state.rewardCoins.value, 12);
  assert.equal(legacy.state.hasFixedReward.value, false);
});
