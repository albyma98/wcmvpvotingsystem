import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { parse as parseJavaScript } from '@babel/parser';
import { parse as parseSfc } from '@vue/compiler-sfc';
import { baseParse } from '@vue/compiler-dom';
import * as Vue from 'vue';
import * as Report from '../engagementReport.js';
import * as Recipient from '../recipientReport.js';
import { compileAdminV2Controller } from '../../build/adminV2Controllers.js';
import { adminV2Path, readAdminV2Route } from './router/routes.js';

const sourceFile = new URL('../components/AdminPortal.vue', import.meta.url);
const legacySource = readFileSync(sourceFile, 'utf8');
const compiledPortal = compileAdminV2Controller(legacySource, fileURLToPath(sourceFile));

test('V2 routes preserve slug, nested detail and legacy route boundaries', () => {
  assert.deepEqual(readAdminV2Route('/joy-volley/admin/new'), { slug: 'joy-volley', section: 'dashboard', detail: '' });
  assert.deepEqual(readAdminV2Route('/joy-volley/admin/new/events/42/settings/'), { slug: 'joy-volley', section: 'events', detail: '42/settings' });
  assert.equal(adminV2Path('joy-volley'), '/joy-volley/admin/new');
  assert.equal(adminV2Path('joy-volley', 'events', '42/settings'), '/joy-volley/admin/new/events/42/settings');
  for (const route of ['/joy-volley/admin', '/joy-volley/admin/lottery', '/admin/new', '/joy-volley/new', '/club/admin/new', '/joy%2fother/admin/new', '/%zz/admin/new', '/%61dmin/admin/new', '/%63lub/admin/new']) assert.equal(readAdminV2Route(route), null, route);
  assert.throws(() => adminV2Path('joy/other'));
});

test('headless compiler supports every reused controller and removes their presentation', () => {
  for (const filename of ['AdminPortal.vue', 'AdminLottery.vue', 'AdminBarSection.vue', 'admin/marketing/MarketingSection.vue', 'admin/AdminVotingSettings.vue', 'admin/AdminVotingClosure.vue', 'EngagementReportModal.vue', 'RecipientReportBuilder.vue', 'ReportKpiCard.vue', 'ReportAnalysisView.vue']) {
    const url = new URL(`../components/${filename}`, import.meta.url);
    const output = compileAdminV2Controller(readFileSync(url, 'utf8'), fileURLToPath(url));
    assert.doesNotThrow(() => parseJavaScript(output, { sourceType: 'module' }), filename);
    assert.doesNotMatch(output, /import[^\n]+\.vue['"]/);
    assert.doesNotMatch(output, /(?:defineProps|defineEmits)\(/);
    assert.doesNotMatch(output, /<template>|<style/);
    assert.match(output, /export function useController/);
  }
  assert.equal(readFileSync(sourceFile, 'utf8'), legacySource);
});

test('V2 report approval keeps selections, facts and descriptions and resolves the existing PDF module', async t => {
  const url = new URL('../components/EngagementReportModal.vue', import.meta.url);
  const generated = compileAdminV2Controller(readFileSync(url, 'utf8'), fileURLToPath(url));
  assert.doesNotMatch(generated, /import\(['"]\.\.\/engagementReportPdf/);
  assert.match(generated, /import\("[^"]+\/src\/engagementReportPdf\.js"\)/);
  const source = generated.replace(/await import\("[^"]+engagementReportPdf\.js"\)/g, 'await Promise.resolve({createEngagementReportPdf:()=>({save(){}})})');
  const measured = {calculation_id:'calc-v2',event:{id:42,title:'Finale'},facts:{visitors:100,engagement_eligible_visitors:100,active_visitors:40},kpis:[{id:'visitors',value:100,display_description:'Pubblico digitale'},{id:'active_participation',value:40,display_description:'Partecipazione'}]};
  const calls = [];
  const api = {post:async(path,body)=>{calls.push([path,body]);return {data:{...measured,id:73,approved_at:'2026-11-05T10:00:00Z',kpis:Report.calculateReportKpis(body.facts,measured.kpis,body.overrides,measured.facts,body.descriptions)}};}};
  const {state} = controllerFixture(t,api,'superadmin',source,{modelValue:false,eventId:42,authHeaders:{headers:{'X-Organization-Slug':'club'}}});
  state.snapshot.value=structuredClone(measured);state.resetReview();state.facts.visitors=200;state.descriptions.visitors='Descrizione approvata';state.format.value='extended';
  await state.approveAndDownload();
  assert.equal(calls[0][0],'/admin/events/42/engagement-report');
  assert.equal(calls[0][1].calculation_id,'calc-v2');assert.equal(calls[0][1].facts.visitors,200);assert.equal(calls[0][1].descriptions.visitors,'Descrizione approvata');
  assert.deepEqual(calls[0][1].selected_kpi_ids,['visitors','active_participation']);
  assert.equal(state.approved.value.id,73);assert.equal(state.error.value,'');
  await state.approveAndDownload();assert.equal(calls.length,1,'Download retry must not approve a second snapshot');
});

test('V2 report exposes the editable analysis and connects every review action to its shared controller', () => {
  const read = name => parseSfc(readFileSync(new URL(`./components/reports/${name}.vue`, import.meta.url), 'utf8')).descriptor;
  const nodes = node => [node, ...(node.children || []).flatMap(nodes)];
  const builder = nodes(baseParse(read('V2RecipientReportBuilder').template.content)).filter(n => n.type === 1);
  const editor = builder.find(n => n.tag === 'ReportAnalysisView');
  const directive = (node, kind, name) => node.props.find(p => p.type === 7 && p.name === kind && p.arg?.content === name);
  for (const prop of ['analysis', 'review', 'dirty', 'busy', 'warnings']) assert.ok(directive(editor, 'bind', prop), prop);
  for (const event of ['update:analysis', 'save', 'approve', 'discard']) assert.ok(directive(editor, 'on', event), event);
  for (const button of builder.filter(n => n.tag === 'button' && directive(n, 'on', 'click')?.exp.content.startsWith('renderReport('))) assert.match(directive(button, 'bind', 'disabled').exp.content, /analysisApproved/);
  const view = read('V2ReportAnalysisView');
  const definition = parseJavaScript(view.script.content, { sourceType: 'module' }).program.body.find(n => n.type === 'ExportDefaultDeclaration').declaration;
  const props = definition.properties.find(p => p.key.name === 'props').value.properties.map(p => p.key.name);
  const emits = definition.properties.find(p => p.key.name === 'emits').value.elements.map(n => n.value);
  for (const prop of ['review', 'dirty', 'busy', 'warnings']) assert.ok(props.includes(prop), prop);
  for (const event of ['update:analysis', 'save', 'approve', 'discard']) assert.ok(emits.includes(event), event);
  const fields = nodes(baseParse(view.template.content)).filter(n => n.type === 1 && ['input', 'textarea'].includes(n.tag));
  for (const value of ['analysis.headline', 'analysis.summary', 'item.title', 'item.description', 'item.action', 'item.reason', 'limit']) {
    const field = fields.find(n => directive(n, 'bind', 'value')?.exp.content === value);
    assert.ok(field && directive(field, 'on', 'input'), `${value} must be an editable field`);
  }
});

test('headless compiler fails closed when the session integration contract changes', () => {
  assert.throws(() => compileAdminV2Controller(legacySource.replace('async function loadOrganizationName()', 'async function changedSessionFunction()'), fileURLToPath(sourceFile)), /missing controller integration/);
});

test('lottery lifecycle and prize operations retain the existing API contracts', async t => {
  const url = new URL('../components/AdminLottery.vue', import.meta.url);
  const source = compileAdminV2Controller(readFileSync(url, 'utf8'), fileURLToPath(url));
  const calls = [];
  let prize = { id: 3, event_id: 9, name: 'Premio di prova', position: 1 };
  const api = {
    get: async path => {
      calls.push(['GET', path]);
      if (path === '/admin/me') return { data: { role: 'superadmin' } };
      if (path === '/teams') return { data: [{ id: 1, name: 'Casa' }, { id: 2, name: 'Ospiti' }] };
      if (path === '/events') return { data: [{ id: 9, team1_id: 1, team2_id: 2, start_datetime: '2026-11-01T18:00:00Z', prizes: [prize] }] };
      if (path === '/events/9/tickets') return { data: [{ vote_id: 77, ticket_code: 'TEST77', player_id: 1 }], headers: { 'x-lottery-total-codes': '12' } };
      throw new Error(`Unexpected lottery request ${path}`);
    },
    post: async (path, body) => { calls.push(['POST', path, body]); prize = { ...prize, winner: { vote_id: body.vote_id, ticket_code: 'TEST77', status: 'assigned' } }; return { data: prize }; },
    delete: async path => { calls.push(['DELETE', path]); prize = { ...prize, winner: null }; return { data: {} }; },
  };
  const { state, mounts, observations } = controllerFixture(t, api, 'superadmin', source);
  await mounts[0](); await Vue.nextTick(); await new Promise(resolve => setImmediate(resolve));
  assert.equal(observations.mounted, true);
  assert.equal(state.selectedEventId.value, 9);
  assert.equal(state.totalCodesCount.value, 12);
  assert.equal(state.allowedCodesCount.value, 1);
  assert.equal(state.canDrawPrize(state.selectedPrizes.value[0]), true);
  const assigned = await state.assignPrizeWinner(state.selectedPrizes.value[0], state.tickets.value[0]);
  state.updateEventPrize(assigned);
  assert.deepEqual(calls.find(call => call[0] === 'POST'), ['POST', '/events/9/prizes/3/assign', { vote_id: 77 }]);
  assert.equal(state.canDrawPrize(state.selectedPrizes.value[0]), false);
  await state.resetPrize(state.selectedPrizes.value[0]);
  assert.ok(calls.some(call => call[0] === 'DELETE' && call[1] === '/events/9/prizes/3/winner'));
  assert.equal(state.selectedPrizes.value[0].winner, null);
});

function controllerFixture(t, apiClient, cachedRole = 'superadmin', compiledSource = compiledPortal, extraProps = {}) {
  const lifecycle = [];
  const mounts = [];
  const values = [];
  const args = [];
  const runtime = { ...Vue, onMounted: fn => mounts.push(fn), onBeforeUnmount: fn => lifecycle.push(fn) };
  const api = { apiClient, resolveApiUrl: value => value, getOrganizationSlug: () => 'club' };
  const code = compiledSource.replace(/^import (.+) from (".+");$/gm, (_, bindings, moduleJSON) => {
    const module = JSON.parse(moduleJSON);
    const provider = module === 'vue' ? runtime : module.endsWith('/api') ? api : module.endsWith('/engagementReport.js') ? Report : module.endsWith('/recipientReport.js') ? Recipient : { DEFAULT_ROSTER_SCHEMA: 13, MAX_PLAYER_SLOTS: 14 };
    for (const binding of bindings.replace(/[{}]/g, '').split(',')) {
      const [name, alias = name] = binding.trim().split(/\s+as\s+/);
      args.push(alias); values.push(provider[name]);
    }
    return '';
  }).replaceAll('import.meta.env.BASE_URL', "'/'").replace('export function useController', 'return function useController');
  const storage = new Map([['adminUsername', 'Cached admin'], ['adminRole', cachedRole]]);
  const localStorage = { getItem: key => storage.get(key), setItem: (key, value) => storage.set(key, value), removeItem: key => storage.delete(key) };
  const timers = new Map(); let timerId = 0;
  const window = { location: { origin: 'https://example.test', pathname: '/club/admin/new' }, addEventListener() {}, removeEventListener() {}, setInterval(fn) { timers.set(++timerId, fn); return timerId; }, clearInterval(id) { timers.delete(id); } };
  const create = new Function(...args, 'localStorage', 'window', code)(...values, localStorage, window);
  const observations = { ready: false, loading: false, error: null };
  const hooks = {
    onLoading: value => { observations.loading = value; },
    onSessionVerified: () => { observations.ready = true; },
    onSessionError: error => { observations.error = error; },
    onSessionEnded: () => { observations.ready = false; },
    onLifecycleError: error => { observations.error = error; },
    onLifecycleStart: () => { observations.mounting = true; },
    onLifecycleComplete: () => { observations.mounting = false; observations.mounted = true; },
  };
  const scope = Vue.effectScope();
  const state = scope.run(() => create(Vue.reactive({ organizationSlug: 'club', ...extraProps }), () => {}, hooks));
  t.after(() => { lifecycle.forEach(fn => fn()); scope.stop(); });
  return { state, observations, lifecycle, mounts, timers };
}

test('restored session waits for server role before requesting resources and preserves staff permissions', async t => {
  const calls = [];
  let verify;
  const api = { get: async path => {
    calls.push(path);
    if (path === '/admin/me') return new Promise(resolve => { verify = resolve; });
    if (path === '/events') return { data: [] };
    throw new Error(`Unexpected request ${path}`);
  } };
  const { state, observations } = controllerFixture(t, api);
  assert.deepEqual(calls, []);
  const pending = state.loadAll();
  assert.equal(observations.loading, true);
  assert.equal(observations.ready, false);
  assert.deepEqual(calls, ['/admin/me']);
  verify({ data: { role: 'staff', organization_name: 'Club' } });
  await pending;
  assert.equal(observations.ready, true);
  assert.equal(observations.loading, false);
  assert.deepEqual(calls, ['/admin/me', '/events']);
  assert.deepEqual(state.availableTabs.value.map(tab => tab.id), ['closing']);
  state.selectSection('admins');
  assert.equal(state.section.value, 'closing');
});

for (const status of [401, 403, 500]) test(`failed server verification (${status}) prevents every resource load`, async t => {
  const calls = [];
  const failure = { response: { status } };
  const api = { get: async path => { calls.push(path); throw failure; }, post: async path => { calls.push(path); return { data: {} }; } };
  const { state, observations } = controllerFixture(t, api);
  await assert.rejects(state.loadAll(), error => error === failure);
  assert.equal(observations.ready, false);
  assert.equal(observations.loading, false);
  assert.equal(observations.error, failure);
  assert.deepEqual(calls, status === 401 ? ['/admin/me', '/admin/logout'] : ['/admin/me']);
});

test('controller keeps role and BAR feature restrictions and registers cleanup', t => {
  const { state, lifecycle } = controllerFixture(t, {});
  state.activeRole.value = 'bar';
  assert.deepEqual(state.availableTabs.value.map(tab => tab.id), ['bar']);
  state.isBarFeatureEnabled.value = false;
  assert.deepEqual(state.availableTabs.value, []);
  state.activeRole.value = 'superadmin';
  assert.equal(state.availableTabs.value.length, 13);
  assert.equal(lifecycle.length, 1);
  assert.match(compiledPortal, /onBeforeUnmount\(\(\) => \{[\s\S]*stopResultsPolling\(\)/);
});

const barFile = new URL('../components/AdminBarSection.vue', import.meta.url);
const compiledBar = compileAdminV2Controller(readFileSync(barFile, 'utf8'), fileURLToPath(barFile));

test('generated BAR controller preserves product, menu and category payloads', async t => {
  const writes = [];
  const api = {
    get: async () => ({ data: [] }),
    post: async (path, body) => { writes.push([path, structuredClone(body)]); return { data: {} }; },
  };
  const { state } = controllerFixture(t, api, 'bar', compiledBar, { authHeaders: { headers: { 'X-Organization-Slug': 'club' } }, isSuperAdmin: false });
  Object.assign(state.newProduct.value, { name: 'Acqua', description: 'Naturale', priceEuro: 1.5, categoryId: 2, image_url: 'data:image/png;base64,test' });
  await state.createProduct();
  Object.assign(state.newMenu.value, { name: 'Pausa', description: '', priceEuro: 5, items: { 3: 2, 4: 0 } });
  await state.createMenu();
  Object.assign(state.newCategory.value, { name: 'Bevande', image_url: 'data:image/png;base64,test' });
  await state.saveCategory();
  assert.deepEqual(writes, [
    ['/admin/bar/products', { name: 'Acqua', description: 'Naturale', price_cents: 150, image_url: 'data:image/png;base64,test', category_id: 2 }],
    ['/admin/bar/menus', { name: 'Pausa', description: '', price_cents: 500, items: [{ product_id: 3, quantity: 2 }] }],
    ['/admin/bar/categories', { name: 'Bevande', image_url: 'data:image/png;base64,test' }],
  ]);
  assert.equal(state.newProduct.value.name, '');
  assert.equal(state.newMenu.value.name, '');
});

test('BAR initial load errors reach presentation and do not start polling', async t => {
  const failure = new Error('Service unavailable');
  const { mounts, observations } = controllerFixture(t, { get: async () => { throw failure; } }, 'bar', compiledBar, { authHeaders: {} });
  assert.equal(mounts.length, 1);
  await mounts[0]();
  assert.equal(observations.error, failure);
  assert.equal(observations.mounted, undefined);
});

test('generic lifecycle completion is reported after asynchronous initialization', async t => {
  const file = new URL('../components/admin/marketing/MarketingSection.vue', import.meta.url);
  const compiled = compileAdminV2Controller(readFileSync(file, 'utf8'), fileURLToPath(file));
  let resolveAudience;
  const api = { get: async path => path.endsWith('/audience') ? new Promise(resolve => { resolveAudience = resolve; }) : ({ data: [] }) };
  const { mounts, observations } = controllerFixture(t, api, 'superadmin', compiled);
  const loading = mounts[0]();
  assert.equal(observations.mounting, true);
  assert.equal(observations.mounted, undefined);
  resolveAudience({ data: [] });
  await loading;
  assert.equal(observations.mounted, true);
  assert.equal(observations.mounting, false);
});

test('a malformed session response cannot reuse a cached administrator role', async t => {
  const calls = [];
  const { state, observations } = controllerFixture(t, { get: async path => { calls.push(path); return { data: { organization_name: 'Club' } }; } });
  await assert.rejects(state.loadAll(), /sessione non valida/);
  assert.equal(observations.ready, false);
  assert.deepEqual(calls, ['/admin/me']);
});

test('BAR retry restarts initialization and retains exactly one polling timer', async t => {
  let fail = true;
  const { state, mounts, timers } = controllerFixture(t, { get: async () => { if (fail) throw new Error('Offline'); return { data: [] }; } }, 'bar', compiledBar, { authHeaders: {} });
  await mounts[0](); assert.equal(timers.size, 0);
  fail = false; await state.retryLifecycle(); assert.equal(timers.size, 1);
  await state.retryLifecycle(); assert.equal(timers.size, 1);
});

test('BAR unmount during initialization never starts a late polling timer', async t => {
  let finish;
  const { mounts, lifecycle, timers } = controllerFixture(t, { get: async path => path === '/admin/bar/overview' ? new Promise(resolve => { finish = resolve; }) : ({ data: [] }) }, 'bar', compiledBar, { authHeaders: {} });
  const loading = mounts[0]();
  lifecycle.forEach(fn => fn());
  finish({ data: {} }); await loading;
  assert.equal(timers.size, 0);
});

test('concurrent unauthorized handlers issue one logout and a new login waits for it', async t => {
  const calls = [];
  let finishLogout;
  const api = {
    post: async path => {
      calls.push(path);
      if (path === '/admin/logout') return new Promise(resolve => { finishLogout = resolve; });
      if (path === '/admin/login') return { data: { username: 'Reconnected', role: 'staff' } };
      throw new Error(`Unexpected POST ${path}`);
    },
    get: async path => {
      calls.push(path);
      if (path === '/admin/me') return { data: { role: 'staff', organization_name: 'Club' } };
      if (path === '/events') return { data: [] };
      throw new Error(`Unexpected GET ${path}`);
    },
  };
  const { state } = controllerFixture(t, api);
  state.handleUnauthorized(); // Response interceptor.
  state.handleUnauthorized(); // Another request completed at the same time.
  await assert.rejects(state.secureRequest(async () => { throw { response: { status: 401 } }; })); // Existing lexical handler.
  assert.equal(state.isAuthenticated.value, false);
  assert.deepEqual(calls, ['/admin/logout']);
  Object.assign(state.loginForm, { username: 'Reconnected', password: 'test' });
  const login = state.login();
  assert.equal(state.isLoggingIn.value, true);
  assert.deepEqual(calls, ['/admin/logout']);
  finishLogout({ data: {} });
  await login;
  assert.deepEqual(calls, ['/admin/logout', '/admin/login', '/admin/me', '/events']);
  assert.equal(state.isAuthenticated.value, true);
  assert.equal(state.activeUsername.value, 'Reconnected');
  assert.equal(state.section.value, 'closing');
});
