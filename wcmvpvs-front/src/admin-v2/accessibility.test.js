import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { compileScript, compileTemplate, parse } from '@vue/compiler-sfc';
import * as Vue from 'vue';
import * as SSR from 'vue/server-renderer';
import { navigation } from './config/navigation.js';
import { adminV2Path } from './router/routes.js';

function compile(filename) {
 const { descriptor } = parse(readFileSync(new URL(filename, import.meta.url), 'utf8'));
 const script = compileScript(descriptor, { id: filename });
 const imports = [];
 const source = script.content.replace(/import\s+([\s\S]*?)\s+from\s+['"]([^'"]+)['"];?/g, (_, bindings, module) => {
  if (bindings.trim().startsWith('{')) for (const item of bindings.replace(/[{}]/g, '').split(',')) { if (!item.trim()) continue; const [name, alias = name] = item.trim().split(/\s+as\s+/); imports.push({ name, alias, module }); }
  else imports.push({ name: 'default', alias: bindings.trim(), module });
  return '';
 }).replace('export default', 'return');
 const template = compileTemplate({ source: descriptor.template.content, id: filename, ssr: true, ssrCssVars: [], compilerOptions: { bindingMetadata: script.bindings } });
 assert.deepEqual(template.errors, []);
 const helpers = {};
 const render = template.code.replace(/^import \{([^}]+)\} from "([^"]+)"\r?$/gm, (_, bindings, module) => {
  for (const item of bindings.split(',')) { const [name, alias = name] = item.trim().split(/\s+as\s+/); helpers[alias] = (module === 'vue' ? Vue : SSR)[name]; } return '';
 }).replace('export function ssrRender', 'return function ssrRender');
 return { imports, create: new Function(...imports.map(item => item.alias), 'document', source), render: new Function(...Object.keys(helpers), render)(...Object.values(helpers)) };
}
const dialog = compile('./components/shared/V2Dialog.vue');
const workspace = compile('./layouts/V2Workspace.vue');

function dialogFixture(t) {
 const watches = [], cleanup = [], events = [];
 const previous = { isConnected: true, focus() { document.activeElement = previous; } };
 const app = { inert: false, tagName: 'DIV' }, backdrop = { tagName: 'DIV' };
 const document = { activeElement: previous, body: { style: { overflow: 'auto' }, children: [app, backdrop] } };
 const runtime = { ...Vue, useId: () => 'dialog-label', nextTick: async () => {}, watch: (_, fn) => watches.push(fn), onBeforeUnmount: fn => cleanup.push(fn) };
 const values = dialog.imports.map(item => item.module === 'vue' ? runtime[item.name] : { render() {} });
 const component = dialog.create(...values, document);
 const props = Vue.reactive({ open: false, busy: false, title: 'Conferma operazione' });
 const state = component.setup(props, { expose() {}, emit: (...args) => events.push(args) });
 const panel = { parentElement: backdrop, focus() { document.activeElement = panel; }, querySelectorAll() { return []; } };
 state.panel.value = panel;
 t.after(() => cleanup.forEach(fn => fn()));
 return { state, props, watches, events, document, app, panel, previous };
}

test('dialog blocks background interaction and restores focus and scrolling on close', async t => {
 const f = dialogFixture(t);
 f.props.open = true; await f.watches[0](true);
 assert.equal(f.app.inert, true);
 assert.equal(f.document.body.style.overflow, 'hidden');
 assert.equal(f.document.activeElement, f.panel);
 f.props.open = false; await f.watches[0](false);
 assert.equal(f.app.inert, false);
 assert.equal(f.document.body.style.overflow, 'auto');
 assert.equal(f.document.activeElement, f.previous);
});

test('dialog supports Escape and prevents closing a pending operation', t => {
 const f = dialogFixture(t); let prevented = 0;
 const event = { key: 'Escape', preventDefault() { prevented++; } };
 f.props.busy = true; f.state.keydown(event); assert.deepEqual(f.events, []);
 f.props.busy = false; f.state.keydown(event); assert.deepEqual(f.events, [['close']]);
 assert.equal(prevented, 2);
});

test('dialog keyboard focus wraps in both directions', t => {
 const f = dialogFixture(t);
 const first = { getClientRects: () => [1], focus() { f.document.activeElement = first; } };
 const last = { getClientRects: () => [1], focus() { f.document.activeElement = last; } };
 f.panel.querySelectorAll = () => [first, last];
 f.document.activeElement = last;
 f.state.keydown({ key: 'Tab', shiftKey: false, preventDefault() {} });
 assert.equal(f.document.activeElement, first);
 f.state.keydown({ key: 'Tab', shiftKey: true, preventDefault() {} });
 assert.equal(f.document.activeElement, last);
});

async function navigationHTML(t, role) {
 const c = { organizationSlug: 'joy-volley', organizationName: 'Joy Volley', activeUsername: 'Operatore', isSuperAdmin: role === 'superadmin', isBarAdmin: role === 'bar', availableTabs: role === 'staff' ? [{ id: 'closing' }] : role === 'bar' ? [{ id: 'bar' }] : navigation.flatMap(group => group.items) };
 const cleanup = [];
 const runtime = { ...Vue, inject: () => c, onMounted() {}, onBeforeUnmount: fn => cleanup.push(fn) };
 const values = workspace.imports.map(item => item.module === 'vue' ? runtime[item.name] : item.module.endsWith('/navigation') ? navigation : item.module.endsWith('/routes') ? adminV2Path : { render() {} });
 const component = workspace.create(...values, {});
 const props = Vue.reactive({ section: role === 'bar' ? 'bar' : 'closing', detail: '', meta: { label: 'Regia votazioni', group: 'Centro operativo' } });
 const scope = Vue.effectScope();
 const state = scope.run(() => component.setup(props, { expose() {} }));
 t.after(() => { cleanup.forEach(fn => fn()); scope.stop(); });
 return SSR.renderToString(Vue.createSSRApp({ props: component.props, setup: () => ({ ...Vue.proxyRefs(state), ...props }), ssrRender: workspace.render }, props));
}

test('staff navigation contains only its existing voting permission and preserves society slug', async t => {
 const html = await navigationHTML(t, 'staff');
 assert.match(html, /href="\/joy-volley\/admin\/new\/closing"[^>]*aria-current="page"/);
 assert.doesNotMatch(html, /href="\/joy-volley\/admin\/new\/(?:marketing|admins|bar|settings)"/);
 assert.match(html, /Vai al contenuto/);
 assert.match(html, /aria-controls="v2-navigation"/);
});

test('BAR navigation exposes orders without management privileges', async t => {
 const html = await navigationHTML(t, 'bar');
 assert.match(html, /href="\/joy-volley\/admin\/new\/bar"/);
 assert.doesNotMatch(html, /href="\/joy-volley\/admin\/new\/(?:closing|marketing|admins|settings)"/);
});
