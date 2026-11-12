import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { compileScript, parse } from '@vue/compiler-sfc';
import { testVue } from '../../test/sfcSetup.js';
import * as Kpis from './components/reports/matchdayKpis.js';

const { descriptor } = parse(readFileSync(new URL('./components/reports/MatchdayKpiDialog.vue', import.meta.url), 'utf8'));
const source = compileScript(descriptor, { id: 'matchday-dialog-test' }).content
  .replace(/^import '[^']+';\r?\n/gm, '')
  .replace(/^import (.+) from (['"])([^'"]+)\2;?\r?\n/gm, (_, binding, quote, name) => `const ${binding} = imports[${JSON.stringify(name)}];\n`)
  .replace('export default', 'return');
const createComponent = new Function('imports', 'window', source);

const event = { id: 42, title: 'Finale', division: 'A2' };
const fields = [{ code: 'attendees', label: 'Spettatori', unit: 'count', revision: 0 }, { code: 'qr_scans', unit: 'count', revision: 2, value: 0, note: 'Contatore' }];
const measured = {
  event, scope: 'division', limit: 8, executive: ['penetration', 'far'], availability: { available: 1, total: 2 },
  sections: [{ id: 'performance', metrics: [{ code: 'penetration', value: null }, { code: 'far', value: 63.33 }] }],
  manual_fields: fields,
};
const afterSave = {
  ...measured, availability: { available: 2, total: 2 },
  sections: [{ id: 'performance', metrics: [{ code: 'penetration', value: 12.5 }, { code: 'far', value: 63.33 }] }],
  manual_fields: [{ ...fields[0], revision: 1, value: 1200, note: 'Report biglietteria' }, fields[1]],
};

async function fixture(t, { get, put } = {}) {
  const requests = [];
  const http = {
    get: async (url, config) => { requests.push({ method: 'GET', url, config }); return { data: get ? await get(url, config) : measured }; },
    put: async (url, body, config) => { requests.push({ method: 'PUT', url, body, config }); return { data: put ? await put(url, body, config) : afterSave }; },
  };
  const runtime = { ...testVue, useId: () => 'matchday' };
  const component = createComponent({ vue: runtime, '../../../api': { apiClient: http }, './matchdayKpis.js': Kpis }, {});
  const props = runtime.reactive({ event: null, authHeaders: { headers: { 'X-Organization-Slug': 'club' } } });
  const scope = runtime.effectScope();
  const events = [];
  const state = scope.run(() => component.setup(props, { expose() {}, emit: (...args) => events.push(args) }));
  t.after(() => scope.stop());
  assert.equal(requests.length, 0);
  props.event = event;
  await new Promise(setImmediate);
  return { state, props, requests, events };
}

test('Matchday saves only changed values and renders the server recalculation', async t => {
  const { state, requests } = await fixture(t);
  assert.equal(requests[0].url, '/admin/events/42/matchday-kpis');
  assert.deepEqual(requests[0].config.headers, { 'X-Organization-Slug': 'club' });
  assert.deepEqual(state.currentMetrics.value.map(metric => metric.value), [null, 63.33]);
  assert.deepEqual(state.changes.value, []);
  state.drafts.value.attendees = { value: 1200, note: ' Report biglietteria ' };
  await state.save();
  assert.deepEqual(requests[1].body, { values: [{ code: 'attendees', value: 1200, note: 'Report biglietteria', revision: 0 }] });
  assert.deepEqual(requests[1].config.params, { scope: 'division', limit: 8 });
  assert.deepEqual(state.currentMetrics.value.map(metric => metric.value), [12.5, 63.33]);
  assert.deepEqual(state.changes.value, []);
});

test('Matchday preserves edits across a revision conflict and refreshes the revision before retry', async t => {
  let loads = 0;
  let writes = 0;
  const refreshed = { ...afterSave, manual_fields: [{ ...afterSave.manual_fields[0], revision: 4, value: 1190 }, fields[1]] };
  const { state, requests } = await fixture(t, {
    get: async () => ++loads === 1 ? measured : refreshed,
    put: async () => {
      if (++writes === 1) throw { response: { status: 409 } };
      return afterSave;
    },
  });
  state.drafts.value.attendees = { value: 1200, note: 'Report biglietteria' };
  await state.save();
  assert.equal(state.errorAction.value, 'conflict');
  await state.save();
  assert.equal(writes, 1);
  await state.loadReport();
  assert.equal(state.drafts.value.attendees.value, 1200);
  assert.equal(state.manualFields.value[0].value, 1190);
  await state.save();
  assert.equal(requests.at(-1).body.values[0].revision, 4);
});

test('Matchday formats zero, missing results and percentage-point changes distinctly', () => {
  assert.equal(Kpis.formatKpi(null, 'percent'), '—');
  assert.equal(Kpis.formatKpi(0, 'percent'), '0%');
  assert.equal(Kpis.formatDelta(12.5, 'percent'), '+12,5 p.p.');
  assert.equal(Kpis.manualChangeError({ value: 0, note: 'Fonte' }, fields[0]), null);
  assert.equal(Kpis.manualChangeError({ value: null, note: 'Fonte' }, fields[0]).input, 'value');
});
