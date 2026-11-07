import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { parse, compileScript } from '@vue/compiler-sfc';
import { parse as parseJavaScript } from '@babel/parser';
import * as Vue from 'vue';
import { renderToString } from '@vue/server-renderer';
import { readAdminV2Route } from './router/routes.js';
import { sectionMeta } from './config/navigation.js';
import { useFeedback } from './composables/useFeedback.js';

// Compile the actual root template and setup. Leaf views are lightweight stand-ins
// so the tests isolate routing, session boundaries and component lifecycle keys.
const filename = new URL('./AdminV2App.vue', import.meta.url);
const descriptor = parse(readFileSync(filename, 'utf8'), { filename: filename.pathname }).descriptor;
const compiled = compileScript(descriptor, { id: 'workspace-render-test', inlineTemplate: true }).content;

function fixture(overrides = {}) {
  const selected = [], navigated = [], mounted = [];
  const events = new Map(), windowEvents = new Map(); let responseHandler; let confirmationAllowed = true; let prompts = 0; let feedback;
  const document = { activeElement: null, title: 'Original title', addEventListener: (name, handler) => events.set(name, handler), removeEventListener: name => events.delete(name) };
  const window = { addEventListener: (name, handler) => windowEvents.set(name, handler), removeEventListener: name => windowEvents.delete(name), confirm: () => { prompts++; return confirmationAllowed; } };
  const state = Vue.reactive({
    isAuthenticated: true, sessionReady: true, initialLoading: false, accessError: '', globalError: '',
    isSuperAdmin: true, isStaff: false, isBarAdmin: false, activeUsername: 'admin',
    availableTabs: Object.keys(sectionMeta).filter(id => id !== 'settings').map(id => ({ id })),
    loginForm: { username: '', password: '' }, loginError: '', isLoggingIn: false,
    selectSection(section) { selected.push(section); }, stopResultsPolling() {}, retrySession() {}, logout() {},
    ...overrides,
  });
  const view = name => ({
    name, props: { section: String, detail: String, settings: Boolean },
    setup(props) {
      const key = Vue.getCurrentInstance().vnode.key;
      mounted.push({ name, key, settings: props.settings === true, section: props.section });
      return () => Vue.h('div', { 'data-view': name, 'data-key': key, 'data-settings': String(props.settings === true) });
    },
  });
  const children = Object.fromEntries(['EventsView', 'EntitiesView', 'InsightsView', 'BarView', 'MarketingView', 'VotingView', 'LotteryView', 'MerchView'].map(name => [name, view(name)]));
  const Workspace = { setup: (_, { slots }) => () => Vue.h('main', { 'data-workspace': 'verified' }, slots.default?.()) };
  const Panel = { props: ['title'], setup: (props, { slots }) => () => Vue.h('section', [Vue.h('h2', props.title), slots.default?.()]) };
  const Dialog = { props: ['open'], setup: (props, { slots }) => () => props.open ? Vue.h('aside', slots.default?.()) : null };
  const providers = {
    vue: Vue,
    '../api': { apiClient: { interceptors: { request: { use: () => 1, eject() {} }, response: { use: success => { responseHandler = success; return 1; }, eject() {} } } } },
    './composables/useAdminV2Controller': { useAdminV2Controller: () => state },
    './composables/useFeedback': { useFeedback: () => (feedback = useFeedback()) },
    './router/routes': { readAdminV2Route, navigateAdminV2: (...args) => navigated.push(args) },
    './config/navigation': { sectionMeta },
    './layouts/V2Workspace.vue': { default: Workspace },
    './components/shared/V2Panel.vue': { default: Panel },
    './components/shared/V2Dialog.vue': { default: Dialog },
    './components/shared/V2Icon.vue': { default: { render: () => Vue.h('svg', { 'aria-hidden': 'true' }) } },
    './views/DashboardView.vue': { default: view('DashboardView') },
  };
  const ast = parseJavaScript(compiled, { sourceType: 'module' });
  const params = [], values = [], edits = [];
  for (const node of ast.program.body) {
    if (node.type !== 'ImportDeclaration') continue;
    const provider = providers[node.source.value];
    for (const specifier of node.specifiers) {
      const imported = specifier.type === 'ImportDefaultSpecifier' ? 'default' : specifier.imported?.name;
      assert.ok(provider && imported in provider, `Unconfigured dependency: ${node.source.value} ${imported}`);
      params.push(specifier.local.name); values.push(provider[imported]);
    }
    edits.push(node);
  }
  let code = compiled;
  for (const node of edits.sort((a, b) => b.start - a.start)) code = code.slice(0, node.start) + code.slice(node.end);
  code = code.replaceAll(/import\('\.\/views\/([A-Za-z]+)\.vue'\)/g, 'Promise.resolve(__views.$1)').replace('export default', 'return');
  const component = new Function(...params, '__views', 'window', 'document', code)(...values, children, window, document);
  return { component, state, selected, navigated, mounted, events, windowEvents, document, get feedback() { return feedback; }, mutate: () => responseHandler({ config: { method: 'post' } }), setConfirmation: value => { confirmationAllowed = value; }, get prompts() { return prompts; } };
}

async function render(path, overrides) {
  const result = fixture(overrides);
  const html = await renderToString(Vue.createSSRApp(result.component, { organizationSlug: 'test-club', currentPath: path }));
  result.html = html;
  return result;
}

test('unverified restored sessions show verification without mounting protected content', async () => {
  const { html, mounted, selected } = await render('/test-club/admin/new/teams', { sessionReady: false, initialLoading: true });
  assert.match(html, /Verifica dell’accesso alla società/);
  assert.doesNotMatch(html, /data-workspace|data-view/);
  assert.deepEqual(mounted, []);
  assert.deepEqual(selected, []);
});

test('logged-out direct routes show the existing credential flow and keep the club slug', async () => {
  const { html, mounted } = await render('/test-club/admin/new/history', { isAuthenticated: false, sessionReady: false });
  assert.match(html, /autocomplete="username"/);
  assert.match(html, /autocomplete="current-password"/);
  assert.match(html, /href="\/test-club\/admin"/);
  assert.deepEqual(mounted, []);
});

test('verified entity and insight deep links select only their requested section', async () => {
  for (const [section, expected] of [['teams', 'EntitiesView'], ['players', 'EntitiesView'], ['results', 'InsightsView'], ['history', 'InsightsView']]) {
    const { html, selected, mounted } = await render(`/test-club/admin/new/${section}`);
    assert.match(html, new RegExp(`data-view="${expected}"`));
    assert.deepEqual(selected, [section]);
    assert.deepEqual(mounted.map(view => [view.name, view.section, view.key]), [[expected, section, section]]);
  }
});

test('the merchandising deep link requires the verified society administrator role', async () => {
  const path = '/test-club/admin/new/merch/orders';
  const admin = await render(path);
  assert.match(admin.html, /data-view="MerchView"/);
  assert.deepEqual(admin.selected, []);
  for (const state of [{ isSuperAdmin: false, isStaff: true }, { isSuperAdmin: false, isBarAdmin: true }, { sessionReady: false }]) {
    const denied = await render(path, state);
    assert.deepEqual(denied.mounted, []);
  }
});

test('settings and match control have independent keyed voting component instances', async () => {
  const settings = await render('/test-club/admin/new/settings');
  const closing = await render('/test-club/admin/new/closing');
  assert.deepEqual(settings.mounted, [{ name: 'VotingView', key: 'settings', settings: true, section: undefined }]);
  assert.deepEqual(closing.mounted, [{ name: 'VotingView', key: 'closing', settings: false, section: undefined }]);
  assert.deepEqual(settings.selected, ['dashboard']);
  assert.deepEqual(closing.selected, ['closing']);
});

test('staff root redirects to its permitted page and forbidden routes never mount their leaf view', async () => {
  const staff = { isSuperAdmin: false, isStaff: true, availableTabs: [{ id: 'closing' }] };
  const root = await render('/test-club/admin/new', staff);
  assert.deepEqual(root.navigated, [['test-club', 'closing', '', true]]);
  const forbidden = await render('/test-club/admin/new/admins', staff);
  assert.match(forbidden.html, /Accesso non disponibile/);
  assert.deepEqual(forbidden.mounted, []);
  assert.deepEqual(forbidden.selected, []);
});

test('unknown section displays an explicit missing-page state', async () => {
  const { html, mounted } = await render('/test-club/admin/new/nonexistent');
  assert.match(html, /Pagina non trovata/);
  assert.deepEqual(mounted, []);
});

test('lottery stays inside V2 and retains the existing staff exclusion', async () => {
  const admin = await render('/test-club/admin/new/lottery');
  assert.ok(admin.mounted.some(item => item.name === 'LotteryView'));
  const staff = await render('/test-club/admin/new/lottery', { isSuperAdmin: false, isStaff: true, availableTabs: [{ id: 'closing' }] });
  assert.match(staff.html, /Accesso non disponibile/);
  assert.ok(!staff.mounted.some(item => item.name === 'LotteryView'));
});

function editForm(context, { connected = true } = {}) {
  const input = { id: 'draft-message', type: 'text', value: 'Saved value' };
  const form = { isConnected: connected, querySelectorAll: () => [input] };
  const target = { closest: selector => selector === '.admin-v2' ? {} : selector === 'form' ? form : null };
  context.events.get('focusin')({ target });
  input.value = 'Unsaved value';
  context.events.get('input')({ target });
  return { form, target, input };
}

test('document capture guards teleported drafts and retains dirty state after internal tabs unmount a form', async () => {
  const context = await render('/test-club/admin/new/marketing');
  editForm(context, { connected: false });
  context.setConfirmation(false);
  context.state.navigate('teams');
  assert.deepEqual(context.navigated, []);
  assert.equal(context.prompts, 1);
  const unload = { prevented: false, preventDefault() { this.prevented = true; } };
  context.windowEvents.get('beforeunload')(unload);
  assert.equal(unload.prevented, true);
  assert.equal(unload.returnValue, '');
  context.setConfirmation(true);
  context.state.navigate('teams');
  assert.deepEqual(context.navigated, [['test-club', 'teams', '']]);
});

test('successful form submission clears its draft guard while auxiliary mutations keep unsaved input protected', async () => {
  const saved = await render('/test-club/admin/new/marketing');
  const { target } = editForm(saved);
  saved.events.get('submit')({ target });
  await saved.state.run('template-save', async () => saved.mutate());
  saved.setConfirmation(false);
  saved.state.navigate('teams');
  assert.equal(saved.prompts, 0);
  assert.deepEqual(saved.navigated, [['test-club', 'teams', '']]);

  const auxiliary = await render('/test-club/admin/new/marketing');
  editForm(auxiliary);
  await auxiliary.state.run('campaign-test', async () => auxiliary.mutate());
  auxiliary.setConfirmation(false);
  auxiliary.state.navigate('teams');
  assert.equal(auxiliary.prompts, 1);
  assert.deepEqual(auxiliary.navigated, []);
});

test('confirmation preserves the originating submitted form across the dialog interaction', async () => {
  const context = await render('/test-club/admin/new/marketing');
  const { target } = editForm(context);
  context.events.get('submit')({ target });
  context.state.confirmAction('Programmare?', 'Conferma campagna', () => context.state.run('campaign-create', async () => context.mutate()));
  await Vue.nextTick();
  await context.feedback.acceptConfirmation();
  context.setConfirmation(false);
  context.state.navigate('teams');
  assert.equal(context.prompts, 0);
  assert.deepEqual(context.navigated, [['test-club', 'teams', '']]);
});

test('feedback adapter prevents duplicate submissions while the first request is pending', async () => {
  const context = await render('/test-club/admin/new/teams');
  let complete, invocations = 0;
  const pending = context.state.run('create-team', async () => {
    invocations++;
    await new Promise(resolve => { complete = resolve; });
    context.mutate();
  });
  const duplicate = context.state.run('create-team', async () => { invocations++; });
  assert.equal(context.state.busy('create-team'), true);
  assert.equal(invocations, 1);
  complete();
  await Promise.all([pending, duplicate]);
  assert.equal(context.state.busy('create-team'), false);
  assert.equal(invocations, 1);
});
