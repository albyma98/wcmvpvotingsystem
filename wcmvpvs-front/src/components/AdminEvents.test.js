import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { compileScript, compileTemplate, parse } from '@vue/compiler-sfc';
import * as Vue from 'vue';

const { descriptor } = parse(readFileSync(new URL('./AdminPortal.vue', import.meta.url), 'utf8'));
const compiled = compileScript(descriptor, { id: 'admin-events-test' });
const imports = [];
const source = compiled.content
  .replace(/import\s+([\s\S]*?)\s+from\s+['"]([^'"]+)['"];?/g, (_, bindings, module) => {
    if (bindings.trim().startsWith('{')) {
      for (const item of bindings.replace(/[{}]/g, '').split(',')) {
        if (!item.trim()) continue;
        const [name, alias = name] = item.trim().split(/\s+as\s+/);
        imports.push({ name, alias, module });
      }
    } else imports.push({ name: 'default', alias: bindings.trim(), module });
    return '';
  })
  .replaceAll('import.meta.env.BASE_URL', JSON.stringify('/'))
  .replace('export default', 'return');
const factory = new Function(...imports.map(item => item.alias), 'localStorage', 'window', source);
const template = compileTemplate({ source: descriptor.template.content, id: 'admin-events-test', compilerOptions: { bindingMetadata: compiled.bindings } });
assert.deepEqual(template.errors, []);
const helpers = {};
const renderCode = template.code.replace(/^import \{([^}]+)\} from "vue"\r?$/gm, (_, names) => {
  for (const item of names.split(',')) {
    const [name, alias = name] = item.trim().split(/\s+as\s+/);
    // VNode inspection needs no DOM directive lifecycle.
    helpers[alias] = name === 'withDirectives' ? vnode => vnode : Vue[name];
  }
  return '';
}).replace('export function render', 'return function render');
const render = new Function(...Object.keys(helpers), renderCode)(...Object.values(helpers));

function renderedButtons(state) {
  const setup = Vue.proxyRefs(state);
  const root = render(setup, [], { organizationSlug: 'club' }, setup);
  const buttons = [];
  const visit = (node, inEvent = false) => {
    if (!node || typeof node !== 'object') return;
    if (Array.isArray(node)) return node.forEach(child => visit(child, inEvent));
    inEvent ||= /(^|\s)ev-card(\s|$)/.test(node.props?.class ?? '');
    if (inEvent && node.type === 'button') buttons.push(node);
    if (Array.isArray(node.children)) node.children.forEach(child => visit(child, inEvent));
    else if (node.children?.default) visit(node.children.default(), inEvent);
  };
  visit(root);
  const text = node => typeof node === 'string' ? node
    : Array.isArray(node) ? node.map(text).join('') : text(node?.children ?? '');
  return label => {
    const button = buttons.find(button => text(button.children).trim() === label);
    assert.ok(button, `missing button ${label}`);
    return button.props.onClick();
  };
}
const clone = value => JSON.parse(JSON.stringify(value));
const settle = async () => { await new Promise(setImmediate); await Vue.nextTick(); };

function fixture(t) {
  let nextPrizeId = 100;
  const server = [9, 10].map(id => ({
    id, team1_id: 1, team2_id: 2, start_datetime: '2026-11-01T19:00', location: 'Arena',
    show_feedback_survey: true, show_selfie: true, show_vote_counter: true,
    prizes: [{ id: id * 10, name: `Premio ${id}`, position: 1, win_sms_text: 'SMS originale' }],
  }));
  const requests = [];
  const control = { failNext: false, beforePut: null };
  const apiClient = {
    get: async path => {
      if (path === '/events') return { data: clone(server) };
      if (path.endsWith('/questions') || path.endsWith('/stories')) return { data: [] };
      return { data: {} };
    },
    put: async (path, payload) => {
      requests.push({ method: 'PUT', path, payload: clone(payload) });
      if (control.beforePut) await control.beforePut(path);
      if (control.failNext) {
        control.failNext = false;
        throw { response: { status: 500 } };
      }
      const event = server.find(event => path === `/events/${event.id}`);
      assert.ok(event, `unknown event ${path}`);
      Object.assign(event, clone(payload));
      if (payload.prizes) event.prizes.forEach(prize => { if (!prize.id) prize.id = nextPrizeId++; });
      return { data: {} };
    },
    post: async (path, payload) => {
      assert.equal(path, '/events');
      requests.push({ method: 'POST', path, payload: clone(payload) });
      const event = { id: server.length + 20, ...clone(payload) };
      event.prizes.forEach(prize => { prize.id = nextPrizeId++; });
      server.push(event);
      return { data: { id: event.id } };
    },
  };
  const runtime = { ...Vue, onMounted() {}, onBeforeUnmount() {} };
  const api = { apiClient, resolveApiUrl: path => path, getOrganizationSlug: () => 'club' };
  const values = imports.map(({ name, module }) => module === 'vue' ? runtime[name]
    : module.endsWith('/api') ? api[name]
      : module.endsWith('/roster') ? ({ DEFAULT_ROSTER_SCHEMA: 13, MAX_PLAYER_SLOTS: 14 })[name]
        : { render: () => null });
  const storage = new Map();
  const localStorage = { getItem: key => storage.get(key) || null, setItem: (key, value) => storage.set(key, value), removeItem: key => storage.delete(key) };
  const window = { location: { origin: 'https://example.test', pathname: '/club/admin' }, addEventListener() {}, removeEventListener() {} };
  const component = factory(...values, localStorage, window);
  const scope = Vue.effectScope();
  const state = scope.run(() => component.setup(Vue.reactive({ organizationSlug: 'club' }), { expose() {} }));
  t.after(() => scope.stop());
  return { state, server, requests, control };
}

test('survey, prizes and settings save independently and keep drafts in other tabs and events', async t => {
  const { state, server, requests } = fixture(t);
  await state.loadEvents();
  await settle();
  state.prizeDraftsFor(9)[0].name = 'Maglia in bozza';
  state.prizeDraftsFor(9)[0].winSmsText = 'SMS in bozza';
  state.feedbackDraftFor(9).questions[0].title = 'Sondaggio personalizzato';
  state.events.value[0].show_selfie = false;
  state.prizeDraftsFor(10)[0].name = 'Premio altro evento in bozza';
  state.feedbackDraftFor(10).suggestionPrompt = 'Suggerimenti altro evento';

  await state.saveEventFeedbackSurvey(state.events.value[0]);
  assert.deepEqual(Object.keys(requests[0].payload), ['feedback_survey']);
  assert.equal(server[0].feedback_survey.questions[0].title, 'Sondaggio personalizzato');
  assert.equal(server[0].prizes[0].name, 'Premio 9');
  assert.equal(server[0].show_selfie, true);
  assert.equal(state.prizeDraftsFor(9)[0].name, 'Maglia in bozza');
  assert.equal(state.prizeDraftsFor(9)[0].winSmsText, 'SMS in bozza');
  assert.equal(state.events.value[0].show_selfie, false);
  assert.equal(state.prizeDraftsFor(10)[0].name, 'Premio altro evento in bozza');
  assert.equal(state.feedbackDraftFor(10).suggestionPrompt, 'Suggerimenti altro evento');

  state.feedbackDraftFor(9).questions[0].title = 'Seconda bozza sondaggio';
  await state.saveEventPrizes(state.events.value[0]);
  assert.deepEqual(Object.keys(requests[1].payload), ['prizes']);
  assert.equal(server[0].prizes[0].name, 'Maglia in bozza');
  assert.equal(server[0].prizes[0].win_sms_text, 'SMS in bozza');
  assert.equal(server[0].feedback_survey.questions[0].title, 'Sondaggio personalizzato');
  assert.equal(state.feedbackDraftFor(9).questions[0].title, 'Seconda bozza sondaggio');
  assert.equal(state.events.value[0].show_selfie, false);

  await state.saveEventSettings(state.events.value[0]);
  assert.equal(Object.hasOwn(requests[2].payload, 'prizes'), false);
  assert.equal(Object.hasOwn(requests[2].payload, 'feedback_survey'), false);
  assert.equal(server[0].show_selfie, false);
  assert.equal(server[0].prizes[0].name, 'Maglia in bozza');
  assert.equal(server[0].feedback_survey.questions[0].title, 'Sondaggio personalizzato');
  await state.saveEventFeedbackSurvey(state.events.value[0]);
  await state.loadEvents();
  assert.equal(state.feedbackDraftFor(9).questions[0].title, 'Seconda bozza sondaggio');
  assert.equal(state.prizeDraftsFor(9)[0].name, 'Maglia in bozza');
});

test('new event saves prizes and survey together, resets its form and survives later survey edits', async t => {
  const { state, server, requests } = fixture(t);
  state.teams.value = [{ id: 1, name: 'Casa' }, { id: 2, name: 'Ospiti' }];
  await settle();
  Object.assign(state.newEvent, { team1_id: 1, team2_id: 2, start_datetime: '2026-11-02T20:00', show_feedback_survey: false });
  state.newEventPrizes.value[0] = { name: '  Pallone  ', winSmsText: '  Hai vinto!  ' };
  state.newEventSurvey.questions[0].title = 'Domanda nuovo evento';
  await state.createEvent();
  assert.equal(requests[0].method, 'POST');
  const created = server.at(-1);
  assert.equal(created.show_feedback_survey, false);
  assert.equal(created.prizes[0].name, 'Pallone');
  assert.equal(created.prizes[0].win_sms_text, 'Hai vinto!');
  assert.equal(created.feedback_survey.questions[0].title, 'Domanda nuovo evento');
  assert.equal(state.newEventPrizes.value[0].name, '');
  assert.equal(state.newEvent.show_feedback_survey, true);

  // Preparing the next event must also survive a save on the newly created one.
  state.newEventPrizes.value[0].name = 'Premio prossimo evento';
  state.newEventSurvey.questions[0].title = 'Domanda prossimo evento';
  state.feedbackDraftFor(created.id).questions[0].title = 'Domanda modificata';
  await state.saveEventFeedbackSurvey(state.events.value.find(event => event.id === created.id));
  assert.equal(created.prizes[0].name, 'Pallone');
  assert.equal(created.show_feedback_survey, false);
  assert.equal(state.newEventPrizes.value[0].name, 'Premio prossimo evento');
  assert.equal(state.newEventSurvey.questions[0].title, 'Domanda prossimo evento');
});

test('adding/removing prizes persists their IDs and clean drafts follow server updates', async t => {
  const { state, server } = fixture(t);
  await state.loadEvents();
  state.addPrizeDraft(9);
  state.prizeDraftsFor(9)[1].name = '  Secondo premio  ';
  await state.saveEventPrizes(state.events.value[0]);
  const second = state.prizeDraftsFor(9)[1];
  assert.ok(second.id > 0);
  assert.equal(second.name, 'Secondo premio');
  state.removePrizeDraft(9, 0);
  await state.saveEventPrizes(state.events.value[0]);
  assert.equal(server[0].prizes.length, 1);
  assert.equal(server[0].prizes[0].id, second.id);
  server[0].prizes[0].name = 'Premio aggiornato altrove';
  server[0].feedback_survey = { questions: [{ id: 'experience', title: 'Domanda aggiornata altrove' }] };
  await state.loadEvents();
  assert.equal(state.prizeDraftsFor(9)[0].name, 'Premio aggiornato altrove');
  assert.equal(state.feedbackDraftFor(9).questions[0].title, 'Domanda aggiornata altrove');
});

test('the event tab buttons call their respective save and prize editing handlers', async t => {
  const { state, server, requests } = fixture(t);
  await state.loadEvents();
  state.activeUsername.value = 'Admin';
  state.activeRole.value = 'superadmin';
  state.section.value = 'events';
  await settle();
  state.events.value[0].show_feedback_survey = false;
  await renderedButtons(state)('Salva impostazioni');
  assert.equal(server[0].show_feedback_survey, false);

  state.setEventTab(9, 'prizes');
  renderedButtons(state)('＋ Aggiungi premio');
  assert.equal(state.prizeDraftsFor(9).length, 2);
  renderedButtons(state)('✕');
  assert.equal(state.prizeDraftsFor(9).length, 1);
  state.prizeDraftsFor(9)[0].name = 'Premio dal pulsante';
  await renderedButtons(state)('Salva premi');
  assert.deepEqual(Object.keys(requests.at(-1).payload), ['prizes']);
  assert.equal(server[0].prizes[0].name, 'Premio dal pulsante');

  state.setEventTab(9, 'feedback');
  state.feedbackDraftFor(9).questions[0].title = 'Domanda dal pulsante';
  await renderedButtons(state)('Salva sondaggio');
  assert.deepEqual(Object.keys(requests.at(-1).payload), ['feedback_survey']);
  assert.equal(server[0].feedback_survey.questions[0].title, 'Domanda dal pulsante');
});

test('failed saves keep drafts and concurrent saves are locked per event', async t => {
  const { state, server, requests, control } = fixture(t);
  await state.loadEvents();
  state.prizeDraftsFor(9)[0].name = 'Premio da conservare';
  state.feedbackDraftFor(9).questions[0].title = 'Domanda da conservare';
  control.failNext = true;
  await state.saveEventFeedbackSurvey(state.events.value[0]);
  assert.ok(state.eventFeedbackErrors[9]);
  assert.equal(state.feedbackDraftFor(9).questions[0].title, 'Domanda da conservare');
  assert.equal(state.prizeDraftsFor(9)[0].name, 'Premio da conservare');
  assert.equal(state.isSavingPrizesFor(9), false);

  let finish;
  control.beforePut = path => path === '/events/9' ? new Promise(resolve => { finish = resolve; }) : undefined;
  const saving = state.saveEventFeedbackSurvey(state.events.value[0]);
  assert.equal(state.isSavingPrizesFor(9), true);
  await state.saveEventPrizes(state.events.value[0]);
  await state.saveEventPrizes(state.events.value[1]);
  assert.equal(state.isSavingPrizesFor(9), true);
  assert.equal(requests.length, 3, 'another save on the same event is ignored');
  finish();
  await saving;
  assert.equal(state.isSavingPrizesFor(9), false);
  assert.equal(server[0].prizes[0].name, 'Premio 9');
  assert.equal(state.prizeDraftsFor(9)[0].name, 'Premio da conservare');
  assert.equal(server[0].feedback_survey.questions[0].title, 'Domanda da conservare');
  state.clearCollections();
  assert.deepEqual(Object.keys(state.eventFeedbackDrafts), []);
  assert.deepEqual(Object.keys(state.eventPrizeDrafts), []);
});
