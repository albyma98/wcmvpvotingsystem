import test from 'node:test';
import assert from 'node:assert/strict';
import { setupSfc, testVue, testWindow } from '../../test/sfcSetup.js';
import * as journey from '../fanJourney.js';

function game(t, props = {}) {
  const setup = setupSfc(new URL('./ReactionTestGame.vue', import.meta.url), { vue: testVue, '../fanJourney': journey }, { completionReward: null, ...props }, { window: testWindow() });
  t.after(() => setup.scope.stop());
  return setup;
}

test('fixed completion reward overrides slow reactions and false starts, requests once, and shows no round coin credit', (t) => {
  const setup = game(t, { completionReward: 10 });
  setup.state.falseStarts.value = 3;
  for (let round = 1; round <= 3; round += 1) setup.state.finalizeRound({ round, totalMs: 900, reactionMs: null, falstart: true });
  assert.equal(setup.state.coins.value, 10);
  assert.equal(setup.state.previewCoins.value, 10);
  setup.state.showRoundCoins();
  assert.equal(setup.state.showCoinBurst.value, false);
  setup.state.phase.value = 'summary';
  setup.state.onPrimaryAction();
  setup.state.onPrimaryAction();
  assert.equal(setup.events.length, 1);
  assert.equal(setup.events[0][1].coins, 10);
});

test('each new reaction session can claim again, while duplicate claims stay blocked', (t) => {
  const completed = game(t, { completionReward: 10 });
  completed.state.phase.value = 'summary';
  completed.state.onPrimaryAction();
  completed.state.onPrimaryAction();
  assert.equal(completed.events.length, 1);
  completed.state.resetGame();
  completed.state.phase.value = 'summary';
  completed.state.onPrimaryAction();
  assert.equal(completed.events.length, 2);
  assert.notEqual(completed.events[0][1].roundId, completed.events[1][1].roundId);
  const legacy = game(t);
  legacy.state.roundResults.value = [1, 2, 3].map((round) => ({ round, totalMs: 200, falstart: false }));
  assert.equal(legacy.state.coins.value, 15);
  assert.equal(legacy.state.hasFixedReward.value, false);
});
