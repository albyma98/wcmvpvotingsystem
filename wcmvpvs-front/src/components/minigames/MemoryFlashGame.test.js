import test from 'node:test';
import assert from 'node:assert/strict';
import { setupSfc, testVue, testWindow } from '../../../test/sfcSetup.js';
import * as journey from '../../fanJourney.js';

function game(t, props = {}) {
  const window = testWindow();
  const setup = setupSfc(new URL('./MemoryFlashGame.vue', import.meta.url), { vue: testVue, '../../fanJourney': journey }, {
    walletCoins: 0, claiming: false, ...props,
  }, { window, navigator: {} });
  t.after(() => setup.scope.stop());
  function revealBoard() {
    setup.state.startRound();
    const preview = [...window.timers.values()][0];
    window.timers.clear();
    preview();
  }
  function completeRound() {
    revealBoard();
    const pairs = new Map();
    for (const card of setup.state.cards.value) {
      if (!pairs.has(card.emoji)) pairs.set(card.emoji, []);
      pairs.get(card.emoji).push(card.id);
    }
    for (const ids of pairs.values()) for (const id of ids) setup.state.onCardTap(id);
  }
  return { ...setup, completeRound };
}

test('a first completion requests eight coins once and wallet changes only through the server prop', (t) => {
  const setup = game(t, { walletCoins: 3 });
  setup.completeRound();
  assert.equal(setup.state.state.value, 'win');
  assert.equal(setup.events.length, 1);
  assert.equal(setup.events[0][1].coins, 8);
  const firstRound = setup.events[0][1].roundId;
  setup.state.onWin();
  assert.equal(setup.events.length, 1);
  assert.equal(setup.state.displayedCoins.value, 3);
  setup.props.walletCoins = 11;
  assert.equal(setup.state.displayedCoins.value, 11);
  setup.completeRound();
  assert.equal(setup.events.length, 2, 'a second victory requests a second reward');
  assert.notEqual(setup.events[1][1].roundId, firstRound);
  assert.equal(setup.state.displayedCoins.value, 11);
});

test('retry is free even with a zero balance and another victory can earn coins', (t) => {
  const setup = game(t);
  setup.state.onLose();
  setup.state.retryNow();
  assert.equal(setup.state.state.value, 'preview');
  assert.equal(setup.state.displayedCoins.value, 0);
  setup.completeRound();
  assert.equal(setup.state.state.value, 'win');
  assert.equal(setup.events.length, 1);
  assert.equal(setup.events[0][1].coins, 8);
  assert.equal(setup.state.displayedCoins.value, 0);
});
