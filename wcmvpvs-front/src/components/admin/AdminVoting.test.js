import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { compileScript, compileTemplate, parse } from '@vue/compiler-sfc';
import * as Vue from 'vue';
import * as VueSSR from 'vue/server-renderer';

function compileComponent(filename) {
  const { descriptor } = parse(readFileSync(new URL(filename, import.meta.url), 'utf8'));
  const compiled = compileScript(descriptor, { id: filename });
  const script = compiled.content.replace(/^import .* from .*\r?\n/gm, '').replace('export default', 'return');
  const create = new Function('Vue', 'apiClient', 'onUnmounted', 'setTimeout', 'clearTimeout',
    `const { computed, reactive, ref, watch } = Vue; ${script}`);
  const template = compileTemplate({ source: descriptor.template.content, id: filename, ssr: true, ssrCssVars: [], compilerOptions: { bindingMetadata: compiled.bindings } });
  assert.deepEqual(template.errors, []);
  const helpers = {};
  const renderCode = template.code.replace(/^import \{([^}]+)\} from "([^"]+)"\r?$/gm, (_, imports, module) => {
    for (const item of imports.split(',')) {
      const [name, alias = name] = item.trim().split(/\s+as\s+/);
      helpers[alias] = (module === 'vue' ? Vue : VueSSR)[name];
    }
    return '';
  }).replace('export function ssrRender', 'return function ssrRender');
  return { create, render: new Function(...Object.keys(helpers), renderCode)(...Object.values(helpers)) };
}

const settingsComponent = compileComponent('./AdminVotingSettings.vue');
const closureComponent = compileComponent('./AdminVotingClosure.vue');
const settle = async () => { await new Promise(resolve => setImmediate(resolve)); await Vue.nextTick(); };

function fixture(t, compiled, apiClient, initialProps) {
  const timers = new Map();
  const cleanup = [];
  let timerId = 0;
  const component = compiled.create(Vue, apiClient, fn => cleanup.push(fn), fn => { timers.set(++timerId, fn); return timerId; }, id => timers.delete(id));
  const props = Vue.reactive({ authHeaders: { headers: { Authorization: 'Bearer test' } }, ...initialProps });
  const events = [];
  const scope = Vue.effectScope();
  const state = scope.run(() => component.setup(props, { expose() {}, emit: (...args) => events.push(args) }));
  t.after(() => { cleanup.forEach(fn => fn()); scope.stop(); });
  async function html() {
    const renderComponent = { setup: () => ({ ...Vue.proxyRefs(state), ...props }), ssrRender: compiled.render };
    return VueSSR.renderToString(Vue.createSSRApp(renderComponent));
  }
  return { state, props, events, timers, html };
}

const initialSettings = () => ({ template_sid: 'HX' + 'a'.repeat(32), draw_mode: 'random', recipients: [{ name: 'Segreteria', phone: '+393331234567', enabled: true }] });
const closureData = status => ({
  summary: { event_id: 9, draw_mode: 'leaderboard', mvp_name: 'Mario Rossi', percentage: '62,50', total_votes: 80,
    prizes: [{ id: 1, name: 'Maglia', ticket_code: '0007', nickname: 'Tifoso' }], warnings: [] },
  notifications: [{ id: 1, name: 'Segreteria', phone: '+39******4567', status }],
});

test('organizer form loads the club configuration and saves template, mode and recipient changes', async t => {
  const writes = [];
  const api = {
    get: async path => { assert.equal(path, '/admin/voting-settings'); return { data: initialSettings() }; },
    put: async (path, body) => { writes.push({ path, body: structuredClone(body) }); return { data: structuredClone(body) }; },
  };
  const { state, html } = fixture(t, settingsComponent, api, { organizationSlug: 'club' });
  await settle();
  assert.equal(state.loaded.value, true);
  state.settings.draw_mode = 'leaderboard';
  state.settings.template_sid = 'HX' + 'b'.repeat(32);
  state.addRecipient();
  Object.assign(state.settings.recipients[1], { name: 'Presidente', phone: '+393331234568', enabled: false });
  await state.save();
  assert.equal(writes.length, 1);
  assert.equal(writes[0].path, '/admin/voting-settings');
  assert.equal(writes[0].body.draw_mode, 'leaderboard');
  assert.equal(writes[0].body.template_sid, 'HX' + 'b'.repeat(32));
  assert.equal(writes[0].body.recipients[1].enabled, false);
  assert.equal(state.saved.value, true);
  state.settings.recipients[0].name = 'Responsabile';
  assert.equal(state.saved.value, false);
  const rendered = await html();
  assert.match(rendered, /Estrazione casuale/);
  assert.match(rendered, /Estrai 3 migliori in classifica tifoso/);
  const selectedRadio = rendered.match(/<input[^>]*name="voting-draw-mode"[^>]*value="leaderboard"[^>]*>/)?.[0];
  assert.match(selectedRadio || '', /\bchecked\b/);
  assert.match(rendered, /Template SID Twilio della società/);
});

test('failed organizer save retains unsaved edits and shows the server validation message', async t => {
  const { state } = fixture(t, settingsComponent, {
    get: async () => ({ data: initialSettings() }),
    put: async () => { throw { response: { status: 400, data: { message: 'Numero duplicato' } } }; },
  }, { organizationSlug: 'club' });
  await settle();
  state.settings.recipients[0].name = 'Modifica in corso';
  await state.save();
  assert.equal(state.saved.value, false);
  assert.equal(state.saving.value, false);
  assert.equal(state.error.value, 'Numero duplicato');
  assert.equal(state.settings.recipients[0].name, 'Modifica in corso');
});

test('configuration from an older club response cannot overwrite the newly selected society', async t => {
  let firstResolve;
  let requests = 0;
  const { state, props } = fixture(t, settingsComponent, {
    get: () => ++requests === 1 ? new Promise(resolve => { firstResolve = resolve; }) : Promise.resolve({ data: { ...initialSettings(), template_sid: 'HX' + 'c'.repeat(32) } }),
  }, { organizationSlug: 'first' });
  props.organizationSlug = 'second';
  await settle();
  firstResolve({ data: initialSettings() });
  await settle();
  assert.equal(state.settings.template_sid, 'HX' + 'c'.repeat(32));
});

test('closing status polls queued organizer messages, displays real codes and stops after provider acceptance', async t => {
  let currentStatus = 'pending';
  const { state, timers, html } = fixture(t, closureComponent, { get: async path => {
    assert.equal(path, '/events/9/voting-closure');
    return { data: closureData(currentStatus) };
  } }, { eventId: 9 });
  await settle();
  assert.equal(timers.size, 1);
  assert.equal(state.canRetry.value, false);
  currentStatus = 'submitted';
  await state.load();
  assert.equal(timers.size, 0);
  const rendered = await html();
  assert.match(rendered, /Mario Rossi/);
  assert.match(rendered, /62,50%/);
  assert.match(rendered, /0007/);
  assert.match(rendered, /Accettato da Twilio/);
});

test('retrying a failed message uses only the notification endpoint and never closes or redraws the event', async t => {
  const requests = [];
  let currentStatus = 'failed';
  const { state, html } = fixture(t, closureComponent, {
    get: async () => ({ data: closureData(currentStatus) }),
    post: async path => { requests.push(path); currentStatus = 'pending'; return {}; },
  }, { eventId: 9 });
  await settle();
  assert.equal(state.canRetry.value, true);
  assert.match(await html(), /Riprova notifiche non inviate/);
  await state.retry();
  assert.deepEqual(requests, ['/events/9/voting-closure/retry']);
  assert.equal(state.notifications.value[0].status, 'pending');
  assert.equal(state.summary.value.prizes[0].ticket_code, '0007');
  assert.equal(state.retrying.value, false);
});

test('ambiguous sends have no blind retry button and expired sessions notify the portal', async t => {
  let unauthorized = false;
  const { state, events, timers, html } = fixture(t, closureComponent, { get: async () => {
    if (unauthorized) throw { response: { status: 401 } };
    return { data: closureData('unknown') };
  } }, { eventId: 9 });
  await settle();
  assert.equal(state.canRetry.value, false);
  assert.equal(timers.size, 0);
  assert.match(await html(), /Esito da verificare/);
  unauthorized = true;
  await state.load();
  assert.deepEqual(events, [['unauthorized']]);
});
