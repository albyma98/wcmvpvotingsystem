import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { compileScript, parse } from '@vue/compiler-sfc';
import * as Vue from 'vue';

// Exercise the actual SFC setup with only the HTTP request and delayed close stubbed.
const { descriptor } = parse(readFileSync(new URL('./EventFeedbackModal.vue', import.meta.url), 'utf8'));
const script = compileScript(descriptor, { id: 'feedback-flow-test' }).content
  .replace(/^import .* from 'vue';?\r?\n/gm, '')
  .replace(/^import .* from '\.\.\/api';?\r?\n/gm, '')
  .replace('export default', 'return');
const createComponent = new Function('Vue', 'submitEventFeedback', 'setTimeout',
  `const { computed, reactive, ref, watch } = Vue;\n${script}`);

const choices = { experience: 'very_easy', team_spirit: 'high', perks_interest: 'yes', mini_games_interest: 'super_excited' };
const survey = {
  questions: Object.entries(choices).map(([id, value]) => ({ id, title: `Domanda ${id}`, answers: [{ value, label: 'Risposta' }] })),
  suggestion_prompt: 'Cosa miglioreresti?',
};

function fixture(result = { ok: true }) {
  const requests = [], events = [], timers = [];
  const scope = Vue.effectScope();
  const component = createComponent(Vue, async (eventId, payload) => {
    requests.push({ eventId, payload });
    return result;
  }, (callback) => timers.push(callback));
  const state = scope.run(() => component.setup(
    Vue.reactive({ modelValue: true, eventId: 42, feedbackSurvey: survey }),
    { expose() {}, emit: (...args) => events.push(args) },
  ));
  return { state, requests, events, timers, scope };
}

function answerQuestions(state) {
  for (const value of Object.values(choices)) state.onSelectOption(value);
  state.onContinue();
}

test('questions, back navigation, final open answer and successful submission', async (t) => {
  const { state, requests, events, timers, scope } = fixture();
  t.after(() => scope.stop());
  state.onContinue();
  assert.equal(state.step.value, 0);
  assert.ok(state.errorMessage.value);
  state.onSelectOption('very_easy');
  assert.equal(state.step.value, 1);
  // Revisiting an answered question must continue to the next question without submitting.
  state.step.value = 0;
  state.onContinue();
  assert.equal(state.step.value, 1);
  assert.equal(requests.length, 0);
  for (const value of ['high', 'yes', 'super_excited']) state.onSelectOption(value);
  state.onContinue();
  assert.equal(state.isOptionalStep.value, true);
  assert.equal(state.activeQuestion.value.title, 'Cosa miglioreresti?');
  state.answers.suggestion = '  Più attività per i tifosi  ';
  state.onContinue();
  await new Promise(setImmediate);
  assert.deepEqual(requests, [{ eventId: 42, payload: { ...choices, suggestion: 'Più attività per i tifosi' } }]);
  assert.ok(events.some(([event]) => event === 'submitted'));
  timers[0]();
  assert.ok(events.some(([event, value]) => event === 'update:modelValue' && value === false));
  assert.equal(state.step.value, 0);
});

test('failed save keeps the modal open and retains answers', async (t) => {
  const { state, events, timers, scope } = fixture({ ok: false });
  t.after(() => scope.stop());
  answerQuestions(state);
  state.answers.suggestion = 'Un suggerimento';
  state.onContinue();
  await new Promise(setImmediate);
  assert.ok(state.errorMessage.value);
  assert.equal(state.answers.suggestion, 'Un suggerimento');
  assert.equal(state.isSubmitting.value, false);
  assert.equal(events.length, 0);
  assert.equal(timers.length, 0);
});

test('the final open question can be skipped without losing the other answers', async (t) => {
  const { state, requests, scope } = fixture();
  t.after(() => scope.stop());
  answerQuestions(state);
  state.answers.suggestion = 'Bozza';
  state.skipOptional();
  await new Promise(setImmediate);
  assert.deepEqual(requests[0], { eventId: 42, payload: { ...choices, suggestion: '' } });
});
