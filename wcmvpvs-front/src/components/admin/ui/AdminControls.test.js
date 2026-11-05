import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { compileScript, compileTemplate, parse } from '@vue/compiler-sfc';
import * as Vue from 'vue';
import * as VueSSR from 'vue/server-renderer';

function component(filename) {
  const { descriptor } = parse(readFileSync(new URL(filename, import.meta.url), 'utf8'));
  const compiled = descriptor.script || descriptor.scriptSetup ? compileScript(descriptor, { id: filename }) : { content: 'export default {}', bindings: {} };
  const script = compiled.content.replace(/^import .* from .*\r?\n/gm, '').replace('export default', 'return');
  const result = new Function(script)();
  const template = compileTemplate({ source: descriptor.template.content, id: filename, ssr: true, ssrCssVars: [], compilerOptions: { bindingMetadata: compiled.bindings } });
  assert.deepEqual(template.errors, []);
  const helpers = {};
  const render = template.code.replace(/^import \{([^}]+)\} from "([^"]+)"\r?$/gm, (_, names, module) => {
    for (const item of names.split(',')) {
      const [name, alias = name] = item.trim().split(/\s+as\s+/);
      helpers[alias] = (module === 'vue' ? Vue : VueSSR)[name];
    }
    return '';
  }).replace('export function ssrRender', 'return function ssrRender');
  result.ssrRender = new Function(...Object.keys(helpers), render)(...Object.values(helpers));
  return result;
}

const Input = component('./BaseInput.vue');
const Select = component('./BaseSelect.vue');
const Table = component('./BaseTable.vue');
const html = (control, props, slots) => VueSSR.renderToString(Vue.createSSRApp({ render: () => Vue.h(control, props, slots) }));

test('standalone inputs retain native attributes and their supplied value', async () => {
  const rendered = await html(Input, { type: 'date', name: 'match-date', value: '2026-11-05', required: true, 'aria-label': 'Data partita' });
  assert.equal((rendered.match(/<input\b/g) || []).length, 1);
  assert.match(rendered, /type="date"/);
  assert.match(rendered, /value="2026-11-05"/);
  assert.match(rendered, /aria-label="Data partita"/);
  assert.match(rendered, /\brequired\b/);
});

test('marketing form inputs supplied through slots remain visible and editable', async () => {
  const rendered = await html(Input, {}, { default: () => Vue.h('input', { name: 'campaign', value: 'Matchday', placeholder: 'Nome campagna' }) });
  assert.equal((rendered.match(/<input\b/g) || []).length, 1);
  assert.match(rendered, /name="campaign"/);
  assert.match(rendered, /value="Matchday"/);
  assert.match(rendered, /placeholder="Nome campagna"/);
});

test('native select options and disabled state survive the shared wrapper', async () => {
  const rendered = await html(Select, {}, { default: () => Vue.h('select', { disabled: true, 'aria-label': 'Categoria' }, [Vue.h('option', { value: 'event', selected: true }, 'Eventi')]) });
  assert.match(rendered, /<select[^>]*disabled[^>]*aria-label="Categoria"/);
  assert.match(rendered, /<option[^>]*value="event"[^>]*selected[^>]*>Eventi/);
});

test('wide tables keep their contents in a named keyboard-accessible scroll region', async () => {
  const rendered = await html(Table, {}, { default: () => Vue.h('table', null, [Vue.h('tbody', null, [Vue.h('tr', null, [Vue.h('td', null, 'Dati società')])])]) });
  assert.match(rendered, /role="region"/);
  assert.match(rendered, /aria-label="Tabella dati"/);
  assert.match(rendered, /tabindex="0"/);
  assert.match(rendered, /<table>/);
  assert.match(rendered, /Dati società/);
});
