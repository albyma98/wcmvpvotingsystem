import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { compileScript, compileTemplate, parse } from '@vue/compiler-sfc';
import * as Vue from 'vue';
import * as VueSSR from 'vue/server-renderer';
import { DEFAULT_ORGANIZATION_COLORS, isHexColor, normalizeOrganizationColors } from '../organizationTheme.js';

const { descriptor } = parse(readFileSync(new URL('./MasterPortal.vue', import.meta.url), 'utf8'));
const compiled = compileScript(descriptor, { id: 'master-sport-test' });
const source = compiled.content.replace(/^import .* from .*\r?\n/gm, '').replace('export default', 'return');
const createComponent = new Function('Vue', 'apiClient', 'localStorage', 'DEFAULT_ORGANIZATION_COLORS', 'isHexColor', 'normalizeOrganizationColors', 'VoteTrendChart',
  `const { computed, defineAsyncComponent, reactive, ref, watch } = Vue; const onMounted = () => {};\n${source}`);
const template = compileTemplate({ source: descriptor.template.content, id: 'master-sport-test', ssr: true, ssrCssVars: [], compilerOptions: { bindingMetadata: compiled.bindings } });
assert.deepEqual(template.errors, []);
const helpers = {};
const renderCode = template.code.replace(/^import \{([^}]+)\} from "([^"]+)"\r?$/gm, (_, imports, module) => {
  for (const binding of imports.split(',')) {
    const [name, alias = name] = binding.trim().split(/\s+as\s+/);
    helpers[alias] = (module === 'vue' ? Vue : VueSSR)[name];
  }
  return '';
}).replace('export function ssrRender', 'return function ssrRender');
const render = new Function(...Object.keys(helpers), renderCode)(...Object.values(helpers));

function fixture(t) {
  const writes = [];
  const client = {
    get: async () => ({ data: [] }),
    post: async (path, body) => { writes.push({ method: 'POST', path, body }); },
    put: async (path, body) => { writes.push({ method: 'PUT', path, body }); },
  };
  const component = createComponent(Vue, client, { getItem: key => key === 'adminRole' ? 'superadmin' : 'Master' }, DEFAULT_ORGANIZATION_COLORS, isHexColor, normalizeOrganizationColors, {});
  const scope = Vue.effectScope();
  t.after(() => scope.stop());
  const state = scope.run(() => component.setup({}, { expose() {} }));
  state.activeSection.value = 'organizations';
  return { state, writes, html() { let html = ''; render({}, chunk => { html += chunk; }, null, {}, {}, Vue.proxyRefs(state), {}, {}); return html; } };
}

test('master requires a sport, creates each supported sport and restores it when editing', async t => {
  const { state, writes, html } = fixture(t);
  for (const sport of ['Calcio', 'Pallacanestro', 'Pallavolo']) {
    state.openCreateOrganization();
    assert.equal(state.organizationForm.sport, '');
    Object.assign(state.organizationForm, { name: 'Aurora', slug: 'aurora' });
    const before = writes.length;
    await state.submitOrganizationForm();
    assert.equal(writes.length, before);
    assert.match(state.organizationFormError.value, /Seleziona lo sport/);
    state.organizationForm.sport = sport;
    const radios = html().match(/<input\b[^>]*type="radio"[^>]*>/g) || [];
    assert.equal(radios.length, 3);
    assert.ok(radios.every(radio => radio.includes('name="organization-sport"') && radio.includes(' required')));
    const checked = radios.filter(radio => radio.includes(' checked'));
    assert.equal(checked.length, 1);
    assert.match(checked[0], new RegExp(`value="${sport}"`));
    await state.submitOrganizationForm();
    assert.equal(writes.at(-1).method, 'POST');
    assert.equal(writes.at(-1).body.sport, sport);
    state.openEditOrganization({ id: 7, ...writes.at(-1).body });
    assert.equal(state.organizationForm.sport, sport);
  }
  state.organizationForm.sport = 'Calcio';
  await state.submitOrganizationForm();
  assert.equal(writes.at(-1).method, 'PUT');
  assert.equal(writes.at(-1).path, '/admin/master/organizations/7');
  assert.equal(writes.at(-1).body.sport, 'Calcio');
});

test('master shows stored sports and marks legacy organizations as awaiting selection', t => {
  const { state, html } = fixture(t);
  state.organizations.value = [{ id: 1, name: 'Legacy' }, { id: 2, name: 'Aurora', sport: 'Pallavolo' }];
  assert.match(html(), /<td>Pallavolo<\/td>/);
  assert.match(html(), /<td>Da impostare<\/td>/);
  state.openEditOrganization(state.organizations.value[0]);
  assert.equal(state.organizationForm.sport, '');
  const radios = html().match(/<input\b[^>]*type="radio"[^>]*>/g) || [];
  assert.equal(radios.length, 3);
  assert.ok(radios.every(radio => !radio.includes(' checked')));
  state.activeSection.value = 'organization-detail';
  state.organizationDetail.value = { organization: state.organizations.value[1], stats: {} };
  assert.match(html(), /<dt>Sport praticato<\/dt>\s*<dd>Pallavolo<\/dd>/);
});
