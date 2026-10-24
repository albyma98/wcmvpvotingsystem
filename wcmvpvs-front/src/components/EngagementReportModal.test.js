import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { compileScript, parse } from '@vue/compiler-sfc';
import { testVue } from '../../test/sfcSetup.js';
import * as Report from '../engagementReport.js';

const { descriptor } = parse(readFileSync(new URL('./EngagementReportModal.vue', import.meta.url), 'utf8'));
const source = compileScript(descriptor, { id: 'report-modal-test' }).content
  .replace(/^import\s+([\s\S]*?)\s+from\s+(['"])([^'"]+)\2;?\r?\n/gm, (_, binding, quote, name) => `const ${binding.trim()} = imports[${JSON.stringify(name)}];\n`)
  .replace("await import('../engagementReportPdf.js')", 'await Promise.resolve({ createEngagementReportPdf: createPdf })')
  .replace('export default', 'return');
const componentFactory = new Function('imports', 'document', 'createPdf', source);
const measured = { calculation_id: 'calc-42', event: { id: 42, title: 'Finale' }, branding: { company_name: 'ArenaBoostX' }, calculated_at: '2026-10-23T10:00:00Z', facts: { visitors: 100, engagement_eligible_visitors: 100, active_visitors: 40 }, kpis: [{ id: 'visitors', value: 100, display_description: 'Pubblico digitale dell’evento' }, { id: 'active_participation', value: 40, display_description: 'Tifosi che interagiscono' }] };
const flush = () => new Promise(setImmediate);

async function fixture({ get, post, downloadError = false } = {}) {
  const requests = [], events = [], saved = [], rendered = [], handlers = new Map();
  let loaded = measured;
  const http = {
    get: async (...args) => { requests.push(['get', ...args]); const response = get ? await get(...args) : { data: structuredClone(measured) }; loaded = response.data; return response; },
    post: async (...args) => { requests.push(['post', ...args]); return post ? post(...args) : { data: { ...structuredClone(loaded), id: 73, approved_at: '2026-10-23T11:00:00Z', format: args[1].format, facts: args[1].facts, overrides: args[1].overrides, selected_kpi_ids: args[1].selected_kpi_ids, kpis: Report.calculateReportKpis(args[1].facts, loaded.kpis, args[1].overrides, loaded.facts, args[1].descriptions).filter(kpi => loaded.kpis.some(available => available.id === kpi.id)) } }; },
  };
  const focus = { calls: 0, focus() { this.calls++; } };
  const document = { body: { style: { overflow: 'auto' } }, activeElement: focus, addEventListener: (key, handler) => handlers.set(key, handler), removeEventListener: key => handlers.delete(key) };
  const createPdf = snapshot => { rendered.push(snapshot); if (downloadError) throw new Error('PDF temporarily unavailable'); return { save: name => saved.push(name) }; };
  const component = componentFactory({ vue: testVue, '../api': { apiClient: http }, '../engagementReport.js': Report }, document, createPdf);
  const props = testVue.reactive({ modelValue: true, eventId: 42, eventTitle: 'Finale', authHeaders: { headers: { 'X-Organization-Slug': 'club' } } });
  const scope = testVue.effectScope();
  const state = scope.run(() => component.setup(props, { expose() {}, emit: (...args) => events.push(args) }));
  await flush();
  return { state, props, scope, requests, events, saved, rendered, document, handlers, focus };
}

test('opening fetches measurements and base edits approve without requiring a correction reason', async t => {
  const f = await fixture(); t.after(() => { f.state.releaseDialog(); f.scope.stop(); });
  assert.equal(f.requests[0][1], '/admin/events/42/engagement-report');
  assert.equal(f.state.kpis.value.length, 2);
  assert.equal(f.document.body.style.overflow, 'hidden');
  f.state.facts.visitors = 200;
  assert.equal(f.state.kpis.value.find(kpi => kpi.id === 'active_participation').value, 40);
  f.state.facts.engagement_eligible_visitors = 200;
  assert.equal(f.state.kpis.value.find(kpi => kpi.id === 'active_participation').value, 20);
  f.state.format.value = 'extended';
  await f.state.approveAndDownload();
  assert.equal(f.requests[1][2].calculation_id, 'calc-42');
  assert.equal(f.requests[1][2].facts.visitors, 200);
  assert.equal(f.requests[1][2].facts.engagement_eligible_visitors, 200);
  assert.equal(f.requests[1][2].format, 'extended');
  assert.equal(f.requests[1][2].notes, '');
  assert.deepEqual(f.state.validationErrors.value, []);
  assert.equal(f.rendered[0].id, 73);
  assert.equal(f.rendered[0].kpis.find(kpi => kpi.id === 'active_participation').value, 20);
  assert.equal(f.saved.length, 1);
  assert.ok(f.events.some(([name]) => name === 'generated'));
  f.props.modelValue = false; await testVue.nextTick();
  assert.equal(f.document.body.style.overflow, 'auto');
  assert.equal(f.focus.calls, 1);
});

test('failed PDF keeps the approved snapshot for download retry', async t => {
  const f = await fixture({ downloadError: true }); t.after(() => { f.state.releaseDialog(); f.scope.stop(); });
  f.state.descriptions.visitors = 'Descrizione approvata';
  await f.state.approveAndDownload();
  assert.equal(f.state.approved.value.id, 73);
  assert.equal(f.state.approved.value.kpis[0].display_description, 'Descrizione approvata');
  assert.match(f.state.error.value, /approvato e salvato/);
  await f.state.approveAndDownload();
  assert.equal(f.requests.filter(([verb]) => verb === 'post').length, 1);
  assert.equal(f.rendered[1].kpis[0].display_description, 'Descrizione approvata');
  assert.equal(f.state.saving.value, false);
  f.state.descriptions.visitors = 'Nuova descrizione';
  assert.equal(f.state.approved.value, null);
});

test('all KPI descriptions are editable, clearable, resettable and included in approval', async t => {
  const f = await fixture(); t.after(() => { f.state.releaseDialog(); f.scope.stop(); });
  assert.equal(Object.keys(f.state.descriptions).length, 2);
  assert.equal(f.state.descriptions.visitors, measured.kpis[0].display_description);
  f.state.descriptions.visitors = 'Descrizione temporanea';
  f.state.descriptions.active_participation = '';
  f.state.resetReview();
  assert.equal(f.state.descriptions.visitors, measured.kpis[0].display_description);
  assert.equal(f.state.descriptions.active_participation, measured.kpis[1].display_description);
  f.state.descriptions.visitors = '  Pubblico personalizzato  ';
  f.state.descriptions.active_participation = '';
  f.state.toggleOverride(f.state.kpis.value[0]);
  f.state.overrides.visitors.value = 110;
  await f.state.approveAndDownload();
  const body = f.requests.find(([verb]) => verb === 'post')[2];
  assert.equal(Object.keys(body.descriptions).length, 2);
  assert.equal(body.descriptions.visitors, 'Pubblico personalizzato');
  assert.equal(body.descriptions.active_participation, '');
  assert.deepEqual(body.overrides.visitors, { value: 110 });
  assert.equal(body.notes, '');
  assert.equal(f.rendered[0].kpis[0].display_description, 'Pubblico personalizzato');
  assert.equal(f.rendered[0].kpis[1].display_description, '');
});

test('mixed-source review exposes coverage and operational provenance without renaming observed participants as visits', async t => {
  const data = structuredClone(measured);
  data.kpis = Report.REPORT_KPIS.slice(0, 20).map(definition => ({ id: definition.id, group: definition.id === 'visitors' ? 'Pubblico' : 'Engagement', value: measured.kpis.find(kpi => kpi.id === definition.id)?.value ?? null }));
  data.kpis[0] = { ...data.kpis[0], group: 'Pubblico', label: 'Partecipanti digitali osservabili', source: 'mixed', note: 'Partecipazione minima verificabile.' };
  data.details = {
    coverage_notes: ['Le visite prive di azioni richiedono telemetria.'],
    operational: { votes: 19, identified_participants: 7, sponsor_clicks_without_identity: 3 },
    sources: [{ table: 'votes', rows: 19 }, { table: 'engagement_report_signals', rows: 4 }],
  };
  const f = await fixture({ get: async () => ({ data }) });
  t.after(() => { f.state.releaseDialog(); f.scope.stop(); });
  assert.equal(f.state.kpis.value[0].label, data.kpis[0].label);
  assert.equal(f.state.kpis.value[0].source, 'mixed');
  assert.equal(f.state.groups.value[0].name, 'Pubblico');
  assert.equal(f.state.groups.value[1].name, 'Engagement');
  assert.equal(f.state.groups.value.flatMap(group => group.kpis).length, 20);
  assert.equal(f.state.groups.value[0].kpis[0].id, 'visitors');
  assert.equal(f.state.factLabel('visitors'), data.kpis[0].label);
  assert.deepEqual(f.state.coverageNotes.value, data.details.coverage_notes);
  assert.equal(f.state.sourceTables.value[0].table, 'votes');
  assert.equal(f.state.operationalItems.value.find(item => item.key === 'identified_participants').label, 'Profili identificati attivi');
  f.state.facts.visitors = 101;
  assert.equal(f.state.kpis.value[0].source, 'manual');
  assert.equal(f.state.kpis.value[0].note, data.kpis[0].note);
});

test('failed approval preserves corrections and does not generate an unapproved PDF', async t => {
  const f = await fixture({ post: async () => { throw { response: { status: 422, data: { error: 'Dati non coerenti' } } }; } });
  t.after(() => { f.state.releaseDialog(); f.scope.stop(); });
  const visitors = f.state.kpis.value.find(kpi => kpi.id === 'visitors');
  f.state.toggleOverride(visitors);
  f.state.overrides.visitors.value = 110;
  await f.state.approveAndDownload();
  assert.equal(f.state.error.value, 'Dati non coerenti');
  assert.equal(f.state.overrides.visitors.value, 110);
  assert.equal(f.rendered.length, 0);
  assert.equal(f.state.approved.value, null);
});

test('authorization failure and Escape close are surfaced without losing scroll cleanup', async t => {
  const f = await fixture({ get: async () => { throw { response: { status: 401 } }; } });
  t.after(() => { f.state.releaseDialog(); f.scope.stop(); });
  assert.ok(f.events.some(([name]) => name === 'unauthorized'));
  assert.ok(f.state.error.value);
  let prevented = false;
  f.handlers.get('keydown')({ key: 'Escape', preventDefault() { prevented = true; } });
  assert.equal(prevented, true);
  assert.ok(f.events.some(([name, value]) => name === 'update:modelValue' && value === false));
});

function thirtyKpiSnapshot() {
  const data = structuredClone(measured);
  data.facts = { ...data.facts, estimated_seconds: 600, estimated_timed_visitors: 3, identified_visitors: 60, guest_visitors: 40 };
  data.kpis = Report.calculateReportKpis(data.facts).map(kpi => ({ ...kpi, display_description: `Descrizione ${kpi.id}`, source: kpi.id.startsWith('estimated_') ? 'estimated' : 'mixed' }));
  data.selected_kpi_ids = data.kpis.slice(0, 20).map(kpi => kpi.id);
  data.details = { time_estimate: { method: 'Intervalli tra azioni vicine, riconciliati con il tempo misurato.', covered_visitors: 3, total_visitors: 100, excluded_visitors: 97, gap_limit_seconds: 300, interval_cap_seconds: 60, measured_seconds: 100, inferred_seconds: 500 } };
  return data;
}

test('thirty KPI form defaults to the frozen selection and caps one page without discarding choices', async t => {
  const data = thirtyKpiSnapshot();
  const f = await fixture({ get: async () => ({ data: structuredClone(data) }) });
  t.after(() => { f.state.releaseDialog(); f.scope.stop(); });
  assert.equal(f.state.kpis.value.length, 30);
  assert.deepEqual(f.state.selectedKpiIds.value, data.selected_kpi_ids);
  assert.equal(f.state.timeEstimate.value.covered_visitors, 3);
  assert.equal(f.state.sourceLabel('estimated'), 'Stima da azioni');
  f.state.selectKpis('all');
  assert.equal(f.state.selectedKpiIds.value.length, 30);
  await f.state.approveAndDownload();
  assert.equal(f.requests.filter(([verb]) => verb === 'post').length, 0);
  assert.ok(f.state.validationErrors.value.some(message => message.includes('20 KPI')));
  assert.equal(f.state.selectedKpiIds.value.length, 30);
  f.state.format.value = 'extended';
  await f.state.approveAndDownload();
  const body = f.requests.find(([verb]) => verb === 'post')[2];
  assert.equal(body.selected_kpi_ids.length, 30);
  assert.deepEqual(body.selected_kpi_ids, data.kpis.map(kpi => kpi.id));
  assert.equal(f.rendered[0].selected_kpi_ids.length, 30);
  f.state.selectedKpiIds.value = ['estimated_avg_time', 'visitors'];
  assert.equal(f.state.approved.value, null);
  f.state.resetReview();
  assert.deepEqual(f.state.selectedKpiIds.value, data.selected_kpi_ids);
});

test('selection and unselected KPI edits are archived while PDF retry uses its approved selection', async t => {
  const data = thirtyKpiSnapshot();
  data.selected_kpi_ids = ['estimated_total_time', 'guest_visitors'];
  const f = await fixture({ get: async () => ({ data: structuredClone(data) }), downloadError: true });
  t.after(() => { f.state.releaseDialog(); f.scope.stop(); });
  f.state.descriptions.visitors = 'Conserva anche la descrizione non selezionata';
  f.state.toggleOverride(f.state.kpis.value.find(kpi => kpi.id === 'identified_visitors'));
  f.state.overrides.identified_visitors.value = 61;
  await f.state.approveAndDownload();
  const body = f.requests.find(([verb]) => verb === 'post')[2];
  assert.deepEqual(body.selected_kpi_ids, ['estimated_total_time', 'guest_visitors']);
  assert.equal(Object.keys(body.descriptions).length, 30);
  assert.equal(body.descriptions.visitors, 'Conserva anche la descrizione non selezionata');
  assert.deepEqual(body.overrides.identified_visitors, { value: 61 });
  await f.state.approveAndDownload();
  assert.equal(f.requests.filter(([verb]) => verb === 'post').length, 1);
  assert.deepEqual(f.rendered[1].selected_kpi_ids, body.selected_kpi_ids);
  f.state.selectedKpiIds.value = ['visitors'];
  assert.equal(f.state.approved.value, null);
});

test('empty or unknown selections are rejected and only data selection retains zero values', async t => {
  const data = thirtyKpiSnapshot();
  data.facts.shop_redemptions = 0;
  data.kpis = Report.calculateReportKpis(data.facts);
  const f = await fixture({ get: async () => ({ data: structuredClone(data) }) });
  t.after(() => { f.state.releaseDialog(); f.scope.stop(); });
  f.state.selectKpis('none');
  await f.state.approveAndDownload();
  assert.ok(f.state.validationErrors.value.some(message => message.includes('almeno un')));
  f.state.selectedKpiIds.value = ['unknown'];
  await f.state.approveAndDownload();
  assert.ok(f.state.validationErrors.value.some(message => message.includes('non disponibili')));
  f.state.selectKpis('available');
  assert.ok(f.state.selectedKpiIds.value.includes('shop_redemptions'));
  assert.ok(!f.state.selectedKpiIds.value.includes('satisfaction'));
  assert.deepEqual(f.state.selectedKpiIds.value, f.state.kpis.value.filter(kpi => kpi.value !== null).map(kpi => kpi.id));
  assert.equal(f.requests.filter(([verb]) => verb === 'post').length, 0);
});

test('twenty KPI snapshots without selection preserve their complete original form and payload', async t => {
  const data = thirtyKpiSnapshot();
  data.kpis = data.kpis.slice(0, 20);
  delete data.selected_kpi_ids;
  const f = await fixture({ get: async () => ({ data: structuredClone(data) }) });
  t.after(() => { f.state.releaseDialog(); f.scope.stop(); });
  assert.equal(f.state.kpis.value.length, 20);
  assert.equal(f.state.selectedKpiIds.value.length, 20);
  assert.equal(f.state.factKeys.value.includes('estimated_seconds'), false);
  await f.state.approveAndDownload();
  const body = f.requests.find(([verb]) => verb === 'post')[2];
  assert.equal(body.selected_kpi_ids.length, 20);
  assert.equal(Object.keys(body.descriptions).length, 20);
  assert.equal(body.facts.estimated_seconds, undefined);
  assert.equal(body.descriptions.estimated_avg_time, undefined);
});
