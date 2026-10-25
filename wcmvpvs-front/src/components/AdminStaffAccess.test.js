import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { compileScript, compileTemplate, parse } from '@vue/compiler-sfc';
import * as Vue from 'vue';
import * as VueSSR from 'vue/server-renderer';

function componentFactory(filename) {
  const { descriptor } = parse(readFileSync(new URL(filename, import.meta.url), 'utf8'));
  const compiled = compileScript(descriptor, { id: filename });
  const imports = [];
  const source = compiled.content.replace(/import\s+([\s\S]*?)\s+from\s+['"]([^'"]+)['"];?/g, (_, bindings, module) => {
    if (bindings.trim().startsWith('{')) {
      for (const item of bindings.replace(/[{}]/g, '').split(',')) {
        if (!item.trim()) continue;
        const [name, alias = name] = item.trim().split(/\s+as\s+/);
        imports.push({ name, alias, module });
      }
    } else imports.push({ name: 'default', alias: bindings.trim(), module });
    return '';
  }).replaceAll('import.meta.env.BASE_URL', JSON.stringify('/')).replace('export default', 'return');
  const factory = new Function(...imports.map(item => item.alias), 'localStorage', 'window', source);
  const template = compileTemplate({ source: descriptor.template.content, id: filename, ssr: true, ssrCssVars: [], compilerOptions: { bindingMetadata: compiled.bindings } });
  assert.deepEqual(template.errors, []);
  const helpers = {};
  const renderCode = template.code.replace(/^import \{([^}]+)\} from "([^"]+)"\r?$/gm, (_, names, module) => {
    for (const item of names.split(',')) {
      const [name, alias = name] = item.trim().split(/\s+as\s+/);
      helpers[alias] = (module === 'vue' ? Vue : VueSSR)[name];
    }
    return '';
  }).replace('export function ssrRender', 'return function ssrRender');
  return { imports, factory, render: new Function(...Object.keys(helpers), renderCode)(...Object.values(helpers)) };
}

const portal = componentFactory('./AdminPortal.vue');
const lottery = componentFactory('./AdminLottery.vue');
const sidebar = componentFactory('./admin/AdminSidebar.vue');
const settle = async () => { await new Promise(resolve => setImmediate(resolve)); await Vue.nextTick(); };

function fixture(t, compiled, { cachedRole = '', username = '', serverRole = 'staff', props = {} } = {}) {
  const calls = [];
  const mounted = [];
  const cleanup = [];
  const storage = new Map(Object.entries({ adminRole: cachedRole, adminUsername: username }));
  const localStorage = { getItem: key => storage.get(key) || null, setItem: (key, value) => storage.set(key, value), removeItem: key => storage.delete(key) };
  const redirects = [];
  const location = { origin: 'https://example.test', pathname: '/club/admin/lottery', href: 'https://example.test/club/admin/lottery', replace: url => redirects.push(url) };
  const window = { location, addEventListener() {}, removeEventListener() {} };
  let votesClosed = false;
  const apiClient = {
    get: async path => {
      calls.push(['GET', path]);
      if (path === '/admin/me') return { data: { username: 'Operator', role: serverRole, organization_name: 'Club' } };
      if (path === '/events') return { data: [{ id: 9, is_active: true, votes_closed: votesClosed, team1_id: 1, team2_id: 2, team1_name: 'Club', team2_name: 'Avversari', start_datetime: '2026-10-25T18:00:00Z', prizes: [] }] };
      throw new Error(`Unexpected staff request: ${path}`);
    },
    post: async path => {
      calls.push(['POST', path]);
      if (path === '/admin/login') return { data: { username: 'Operator', role: serverRole } };
      if (path === '/events/9/close-votes') { votesClosed = true; return { data: {} }; }
      throw new Error(`Unexpected staff request: ${path}`);
    },
  };
  const runtime = { ...Vue, onMounted: fn => mounted.push(fn), onBeforeUnmount: fn => cleanup.push(fn) };
  const api = { apiClient, resolveApiUrl: path => path, getOrganizationSlug: () => 'club' };
  const values = compiled.imports.map(({ name, module }) => module === 'vue' ? runtime[name] : module.endsWith('/api') ? api[name] : module.endsWith('/roster') ? ({ DEFAULT_ROSTER_SCHEMA: 13, MAX_PLAYER_SLOTS: 14 })[name] : { render: () => null });
  const component = compiled.factory(...values, localStorage, window);
  const reactiveProps = Vue.reactive({ organizationSlug: 'club', ...props });
  const scope = Vue.effectScope();
  const state = scope.run(() => component.setup(reactiveProps, { expose() {} }));
  t.after(() => { cleanup.forEach(fn => fn()); scope.stop(); });
  async function html() {
    return VueSSR.renderToString(Vue.createSSRApp({ props: component.props, setup: () => ({ ...Vue.proxyRefs(state), ...reactiveProps }), ssrRender: compiled.render }, reactiveProps));
  }
  return { state, calls, mounted, redirects, html };
}

test('staff login opens only voting closure and loads only its required data', async t => {
  const { state, calls } = fixture(t, portal);
  state.loginForm.username = 'Operator';
  state.loginForm.password = 'test';
  await state.login();
  await settle();
  assert.equal(state.section.value, 'closing');
  assert.deepEqual(state.availableTabs.value.map(tab => tab.id), ['closing']);
  assert.deepEqual(state.navigationGroups.value.flatMap(group => group.items.map(item => item.id)), ['closing']);
  assert.deepEqual(calls, [['POST', '/admin/login'], ['GET', '/admin/me'], ['GET', '/events']]);
  assert.equal(state.activeEventLabel.value, 'Club vs Avversari');
  for (const section of ['dashboard', 'results', 'events', 'admins', 'marketing']) {
    state.selectSection(section);
    assert.equal(state.section.value, 'closing');
  }
  state.goToLottery();
  await state.closeActiveEventVoting();
  assert.equal(state.activeEventVotesClosed.value, true);
  assert.deepEqual(calls.slice(-2), [['POST', '/events/9/close-votes'], ['GET', '/events']]);
});

test('restored session verifies the server role before loading administrator resources', async t => {
  const { state, calls } = fixture(t, portal, { cachedRole: 'superadmin', username: 'Operator', serverRole: 'staff' });
  await settle();
  assert.equal(state.section.value, 'closing');
  assert.equal(state.isStaff.value, true);
  assert.deepEqual(calls, [['GET', '/admin/me'], ['GET', '/events']]);
  assert.deepEqual(state.availableTabs.value.map(tab => tab.id), ['closing']);
});

test('superadmin navigation still includes dashboard and results', async t => {
  const { state } = fixture(t, portal);
  state.activeRole.value = 'superadmin';
  state.selectDefaultSection();
  assert.equal(state.section.value, 'dashboard');
  assert.ok(state.availableTabs.value.some(tab => tab.id === 'results'));
  state.selectSection('events');
  assert.equal(state.section.value, 'events');
});

test('staff sidebar shows voting closure and logout without the lottery shortcut', async t => {
  const { html } = fixture(t, sidebar, { props: { groups: [{ label: 'Risultati', items: [{ id: 'closing', label: 'Chiusura votazioni' }] }], activeSection: 'closing', showLottery: false } });
  const rendered = await html();
  assert.match(rendered, /Chiusura votazioni/);
  assert.match(rendered, /Esci/);
  assert.doesNotMatch(rendered, /sidebar__action--lottery/);
});

test('opening the lottery URL with a staff session returns to the society admin', async t => {
  const { calls, redirects, mounted, state } = fixture(t, lottery, { username: 'Operator', cachedRole: 'superadmin', serverRole: 'staff' });
  mounted.forEach(fn => fn());
  await settle();
  assert.equal(state.isStaff.value, true);
  assert.deepEqual(calls, [['GET', '/admin/me']]);
  assert.deepEqual(redirects, ['https://example.test/club/admin']);
});
