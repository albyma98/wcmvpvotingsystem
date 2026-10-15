import test from 'node:test';
import assert from 'node:assert/strict';
import { FIRST_FAN_GOAL, journeyProgress, normalizeJourney, recordJourneyReward, journeyStorageKey } from './fanJourney.js';

test('vote and quiz unlock the first badge at 25 earned coins', () => {
  let state = recordJourneyReward(normalizeJourney(), 'vote', 10);
  assert.equal(journeyProgress(state).remaining, 15);
  state = recordJourneyReward(state, 'quiz', 15);
  assert.deepEqual(journeyProgress(state), { earned: 25, current: 25, remaining: 0, percent: 100, unlocked: true });
  assert.deepEqual(recordJourneyReward(state, 'vote', 10), state);
  assert.deepEqual(recordJourneyReward(state, 'quiz', 15), state);
});

test('progress is capped and independent of spending wallet coins', () => {
  const state = normalizeJourney({ rewards: { vote: 10, quiz: 15, reaction: 10 }, wallet: 0 });
  assert.equal(journeyProgress(state).unlocked, true);
  assert.equal(journeyProgress(state).current, FIRST_FAN_GOAL);
  assert.equal(journeyProgress(state).earned, 35);
  assert.equal(journeyProgress(state).percent, 100);
});

test('invalid stored progress is ignored and data is scoped to club and match', () => {
  const state = normalizeJourney({ rewards: { vote: -10, quiz: '15', unknown: 1000 }, pendingVoteBonus: 'yes' });
  assert.equal(journeyProgress(state).earned, 0);
  assert.equal(state.pendingVoteBonus, false);
  assert.notEqual(journeyStorageKey('club-a', 1), journeyStorageKey('club-a', 2));
  assert.notEqual(journeyStorageKey('club-a', 1), journeyStorageKey('club-b', 1));
});
