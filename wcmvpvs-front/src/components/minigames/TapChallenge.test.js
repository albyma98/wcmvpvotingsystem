import test from 'node:test';
import assert from 'node:assert/strict';
import { setupSfc, testVue, testWindow } from '../../../test/sfcSetup.js';

function game(t, props = {}) {
  const awards = [];
  const setup = setupSfc(new URL('./TapChallenge.vue', import.meta.url), {
    vue: testVue,
    '../../services/coins': { awardTapChallengeCoins: async (payload) => { awards.push(payload); return { ok: true }; } },
    '../../api': {},
  }, { eventId: 42, cooldownSeconds: 60, completionReward: null, alreadyRewarded: false, claiming: false, ...props }, { window: testWindow(), crypto: { randomUUID: () => 'test-request' } });
  t.after(() => setup.scope.stop());
  return { ...setup, awards };
}

test('fixed singleplayer emits only the journey reward once, even with zero taps, and bypasses the legacy wallet service', async (t) => {
  const setup = game(t, { completionReward: 8 });
  setup.state.status.value = 'finished';
  assert.equal(setup.state.earnedCoins.value, 8);
  assert.equal(setup.state.headline.value, 'Hai totalizzato 0 tap');
  await setup.state.claimReward();
  await setup.state.claimReward();
  assert.deepEqual(setup.events, [['claim', { coins: 8 }]]);
  assert.deepEqual(setup.awards, []);
});

test('server saving blocks a fixed claim and already rewarded singleplayers exit without an award', async (t) => {
  const setup = game(t, { completionReward: 8, claiming: true });
  setup.state.status.value = 'finished';
  await setup.state.claimReward();
  assert.deepEqual(setup.events, []);
  assert.equal(setup.state.isPrimaryDisabled.value, true);
  setup.props.claiming = false;
  setup.props.alreadyRewarded = true;
  assert.equal(setup.state.primaryLabel.value, 'Termina');
  await setup.state.claimReward();
  assert.deepEqual(setup.events, [['exit']]);
  assert.deepEqual(setup.awards, []);
});

test('legacy singleplayer and server-awarded live matches retain their original claim paths', async (t) => {
  const legacy = game(t);
  legacy.state.status.value = 'finished';
  legacy.state.tapCount.value = 3;
  await legacy.state.claimReward();
  assert.equal(legacy.awards.length, 1);
  assert.equal(legacy.awards[0].amount, 3);
  assert.equal(legacy.events[0][1].source, 'tap_challenge');
  const live = game(t, { completionReward: 8, alreadyRewarded: true, claiming: true });
  live.state.applyLiveResult({ outcome: 'win', coins_earned: 12, my_score: 5 });
  assert.equal(live.state.isFixedSingleplayer.value, false);
  assert.equal(live.events[0][1].source, 'tap_challenge_live');
  assert.equal(live.events[0][1].coins, 12);
  assert.deepEqual(live.awards, []);
});
