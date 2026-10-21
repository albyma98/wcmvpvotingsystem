import test from 'node:test';
import assert from 'node:assert/strict';
import { setupSfc, testVue, testWindow } from '../../test/sfcSetup.js';
import * as journey from '../fanJourney.js';

function fixture(overrides = {}, completionReward = 15) {
  const window = testWindow();
  const setup = setupSfc(new URL('./QuickQuizGame.vue', import.meta.url), {
    vue: testVue,
    '../fanJourney': journey,
    '../api': {
      fetchQuickQuiz: async () => ({ ok: true, data: { questions: [{ id: 1, text: 'Domanda', answers: ['A', 'B'] }], config: { completion_bonus: 5 } } }),
      submitQuickQuizAnswer: async () => ({ ok: true, data: { isCorrect: false, coinsEarned: 0 } }),
      ...overrides,
    },
    '../deviceId': { getOrCreateDeviceId: () => 'device' },
  }, { eventId: 42, completionReward, claiming: false }, { window });
  return { ...setup, window };
}

function finishFeedback(window) {
  const callbacks = [...window.timers.values()];
  window.timers.clear();
  for (const callback of callbacks) callback();
}

test('completing the quiz gives the advertised fixed bonus even with an incorrect answer', async (t) => {
  const { state, scope, events, window } = fixture();
  t.after(() => scope.stop());
  await state.loadQuiz();
  state.claimCoins();
  assert.equal(events.length, 0);
  await state.submitAnswer(1);
  finishFeedback(window);
  assert.equal(state.state.value, 'summary');
  assert.equal(state.totalCoins.value, 15);
  state.claimCoins();
  state.claimCoins();
  assert.equal(events.length, 1);
  assert.equal(events[0][1].coins, 15);
  const firstRound = events[0][1].roundId;
  await state.loadQuiz();
  await state.submitAnswer(1);
  finishFeedback(window);
  state.claimCoins();
  assert.equal(events.length, 2);
  assert.notEqual(events[1][1].roundId, firstRound);
});

test('network errors do not count as answers or produce a completion reward', async (t) => {
  const { state, scope, events, window } = fixture({ submitQuickQuizAnswer: async () => ({ ok: false }) });
  t.after(() => scope.stop());
  await state.loadQuiz();
  await state.submitAnswer(0);
  finishFeedback(window);
  assert.equal(state.state.value, 'question');
  assert.equal(state.totalCoins.value, 0);
  assert.ok(state.errorMessage.value);
  state.claimCoins();
  assert.equal(events.length, 0);
});

test('an unavailable quiz cannot be claimed and busy wallet saves disable claims', async (t) => {
  const failed = fixture({ fetchQuickQuiz: async () => ({ ok: false }) });
  t.after(() => failed.scope.stop());
  await failed.state.loadQuiz();
  failed.state.claimCoins();
  assert.equal(failed.state.state.value, 'unavailable');
  assert.equal(failed.events.length, 0);
  const completed = fixture();
  t.after(() => completed.scope.stop());
  await completed.state.loadQuiz();
  await completed.state.submitAnswer(0);
  finishFeedback(completed.window);
  completed.props.claiming = true;
  completed.state.claimCoins();
  assert.equal(completed.events.length, 0);
});

test('other pages retain the existing variable quiz rewards', async (t) => {
  const { state, scope, window } = fixture({ submitQuickQuizAnswer: async () => ({ ok: true, data: { isCorrect: true, coinsEarned: 3 } }) }, 0);
  t.after(() => scope.stop());
  await state.loadQuiz();
  await state.submitAnswer(0);
  finishFeedback(window);
  assert.equal(state.totalCoins.value, 8);
});
