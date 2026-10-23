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
  const http = {
    get: async (...args) => { requests.push(['get', ...args]); return get ? get(...args) : { data: structuredClone(measured) }; },
    post: async (...args) => { requests.push(['post', ...args]); return post ? post(...args) : { data: { ...structuredClone(measured), id: 73, approved_at: '2026-10-23T11:00:00Z', format: args[1].format, facts: args[1].facts, overrides: args[1].overrides, kpis: Report.calculateReportKpis(args[1].facts, measured.kpis, args[1].overrides, measured.facts, args[1].descriptions) } }; },
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
  assert.equal(f.state.kpis.value.length, 20);
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
  assert.equal(Object.keys(f.state.descriptions).length, 20);
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
  assert.equal(Object.keys(body.descriptions).length, 20);
  assert.equal(body.descriptions.visitors, 'Pubblico personalizzato');
  assert.equal(body.descriptions.active_participation, '');
  assert.deepEqual(body.overrides.visitors, { value: 110 });
  assert.equal(body.notes, '');
  assert.equal(f.rendered[0].kpis[0].display_description, 'Pubblico personalizzato');
  assert.equal(f.rendered[0].kpis[1].display_description, '');
});

test('mixed-source review exposes coverage and operational provenance without renaming observed participants as visits', async t => {
  const data = structuredClone(measured);
  data.kpis = Report.REPORT_KPIS.map(definition => ({ id: definition.id, group: definition.id === 'visitors' ? 'Pubblico' : 'Engagement', value: measured.kpis.find(kpi => kpi.id === definition.id)?.value ?? null }));
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
