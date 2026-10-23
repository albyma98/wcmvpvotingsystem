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

function fixture(result = { ok: true }, extraProps = {}) {
  const requests = [], events = [], timers = [];
  const scope = Vue.effectScope();
  const component = createComponent(Vue, async (eventId, payload) => {
    requests.push({ eventId, payload });
    return result;
  }, (callback) => timers.push(callback));
  const props = Vue.reactive({ modelValue: true, eventId: 42, feedbackSurvey: survey, ...extraProps });
  const state = scope.run(() => component.setup(
    props,
    { expose() {}, emit: (...args) => events.push(args) },
  ));
  return { state, requests, events, timers, scope, props };
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
  assert.deepEqual(events.find(([event]) => event === 'submitted'), ['submitted', { experience: choices.experience, perks_interest: choices.perks_interest }]);
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

test('a late feedback acknowledgement cannot submit or close the next event', async (t) => {
  let acknowledge;
  const pending = new Promise((resolve) => { acknowledge = resolve; });
  const { state, props, events, timers, scope } = fixture(pending);
  t.after(() => scope.stop());
  answerQuestions(state);
  state.skipOptional();
  props.eventId = 43;
  acknowledge({ ok: true });
  await new Promise(setImmediate);
  assert.equal(events.length, 0);
  assert.equal(timers.length, 0);
});

test('the report satisfaction question is required and remains separate from legacy voting ease', async (t) => {
  const { state, requests, events, timers, scope } = fixture({ ok: true }, { reportExperienceEnabled: true });
  t.after(() => scope.stop());
  assert.equal(state.normalizedSurvey.value.questions.length, 5);
  for (const value of Object.values(choices)) state.onSelectOption(value);
  assert.equal(state.activeQuestion.value.answerKey, 'report_experience');
  assert.equal(state.activeQuestion.value.title, 'Com’è stata la tua esperienza con ArenaBoostX oggi?');
  assert.deepEqual(state.activeQuestion.value.options.map(({ value }) => value), ['awesome', 'good', 'ok', 'bad']);
  state.onContinue();
  assert.equal(state.step.value, 4);
  assert.ok(state.errorMessage.value);
  state.skipOptional();
  await new Promise(setImmediate);
  assert.equal(requests.length, 0, 'even a direct submit cannot omit the report question');
  state.onSelectOption('good');
  state.onContinue();
  assert.equal(state.isOptionalStep.value, true);
  state.answers.suggestion = '  Più giochi  ';
  state.onContinue();
  await new Promise(setImmediate);
  assert.deepEqual(requests, [{ eventId: 42, payload: { ...choices, suggestion: 'Più giochi' } }]);
  assert.equal(Object.hasOwn(requests[0].payload, 'report_experience'), false);
  assert.deepEqual(events.find(([event]) => event === 'submitted'), ['submitted', {
    experience: 'very_easy', perks_interest: 'yes', report_experience: 'good',
  }]);
  timers[0]();
  assert.equal(state.answers.report_experience, '');
});

test('a failed feedback save retains report satisfaction without emitting a successful measurement', async (t) => {
  const { state, requests, events, timers, scope } = fixture({ ok: false }, { reportExperienceEnabled: true });
  t.after(() => scope.stop());
  for (const value of Object.values(choices)) state.onSelectOption(value);
  state.onSelectOption('bad');
  state.onContinue();
  state.skipOptional();
  await new Promise(setImmediate);
  assert.equal(requests.length, 1);
  assert.equal(state.answers.report_experience, 'bad');
  assert.equal(events.length, 0);
  assert.equal(timers.length, 0);
  assert.ok(state.errorMessage.value);
});

test('the shared feedback modal keeps its original flow when report satisfaction is disabled', (t) => {
  const { state, scope } = fixture();
  t.after(() => scope.stop());
  assert.equal(state.normalizedSurvey.value.questions.length, 4);
  assert.equal(state.normalizedSurvey.value.questions.some(({ id }) => id === 'report_experience'), false);
});
