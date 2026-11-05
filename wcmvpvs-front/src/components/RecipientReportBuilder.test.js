import test from 'node:test';
import assert from 'node:assert/strict';
import { setupSfc, testVue } from '../../test/sfcSetup.js';
import * as Report from '../engagementReport.js';
import * as Recipient from '../recipientReport.js';
import { recipientReportFixture } from '../../test/recipientReportFixture.js';

const flush = () => new Promise(setImmediate);
async function fixture({ failure, deferredAI } = {}) {
  let current = recipientReportFixture();
  current.kpis.find(k => k.id === 'venue_adoption').value = null;
  current.kpis.find(k => k.id === 'venue_adoption').availability = { reason: 'manual_source_missing', message: 'Presenze mancanti', manual_allowed: true };
  current.kpis.find(k => k.id === 'sponsored_participation').value = null;
  current.kpis.find(k => k.id === 'sponsored_participation').availability = { reason: 'tracking_missing', message: 'Esposizione insufficiente', manual_allowed: true };
  current.manual_sources = [];
  const calls = [];
  const http = {
    get: async (path, options) => { calls.push(['get', path, options]); if (path === '/admin/sponsors') return { data: [] }; return { data: structuredClone(current) }; },
    put: async (path, body) => { calls.push(['put', path, body]); if (failure) throw failure; const m = body.sources[0]; current.manual_sources = [{ ...m, revision: 1 }]; const k = current.kpis.find(k => k.id === (m.field_key === 'attendees' ? 'venue_adoption' : m.field_key.slice(4))); k.value = m.field_key === 'attendees' ? 50 : m.value; k.availability = { reason: 'available', manual_allowed: true }; k.provenance = m.field_key === 'attendees' ? [] : current.manual_sources; k.source = 'manual_verified'; current.data_hash = 'updated'; return { data: { saved: true } }; },
    post: async (path, body, options) => { calls.push(['post', path, body, options]); if (path.endsWith('/analysis')) return deferredAI ? deferredAI : { data: { analysis: current.analysis, status: 'ready' } }; return { data: { ...structuredClone(current), recipient: current.profiles.find(p => p.id === body.recipient), selected_kpi_ids: current.profiles.find(p => p.id === body.recipient).kpi_ids, include_appendix: body.include_appendix } }; },
  };
  const f = setupSfc(new URL('./RecipientReportBuilder.vue', import.meta.url), { vue: testVue, '../api': { apiClient: http }, '../engagementReport.js': Report, '../recipientReport.js': Recipient, './ReportKpiCard.vue': {}, './ReportAnalysisView.vue': {} }, { eventId: 87, authHeaders: { headers: { 'X-Organization-Slug': 'club' } } });
  await flush(); return { ...f, calls };
}
test('recipient changes update KPIs immediately and never request AI automatically', async t => {
  const f = await fixture(); t.after(() => f.scope.stop());
  assert.equal(f.state.kpis.value.length, 5);
  f.state.recipient.value = 'marketing';
  assert.equal(f.state.kpis.value.length, 10);
  f.state.recipient.value = 'analytics';
  assert.equal(f.state.kpis.value.length, 20);
  f.state.recipient.value = 'sponsor';
  assert.equal(f.state.kpis.value.length, 5);
  await testVue.nextTick();
  assert.equal(f.calls.filter(c => c[0] === 'post').length, 0);
  assert.equal(f.calls.filter(c => c[0] === 'get' && c[1].includes('engagement-report')).length, 1);
});
test('all N/D indicators can be completed and persist between profiles', async t => {
  const f = await fixture(); t.after(() => f.scope.stop());
  assert.equal(f.state.recoverable.value.length, 1);
  f.state.recipient.value = 'sponsor';
  assert.equal(f.state.recoverable.value.length, 1);
  assert.equal(f.state.missing.value.length, 1);
  f.state.recipient.value = 'executive';
  f.state.openSources();
  await f.state.saveSources();
  assert.equal(f.calls.filter(c => c[0] === 'put').length, 0);
  Object.assign(f.state.sourceForm, { value: 1800, source_description: 'Biglietteria ufficiale', verified: true, confirmed: true });
  await f.state.saveSources();
  assert.equal(f.calls.filter(c => c[0] === 'put').length, 1);
  f.state.recipient.value = 'marketing'; f.state.recipient.value = 'executive';
  assert.equal(f.state.kpis.value.find(k => k.id === 'venue_adoption').value, 50);
  assert.equal(f.state.recoverable.value.length, 0);
});

test('contextual manual editing supports decimal rates and sponsor-specific provenance', async t => {
  const f = await fixture(); t.after(() => f.scope.stop());
  f.state.recipient.value = 'sponsor';
  f.state.snapshot.value.sponsor_id = 7;
  const kpi = f.state.kpis.value.find(k => k.id === 'sponsored_participation');
  f.state.openSources(kpi);
  assert.equal(f.state.selectedSource.value.key, 'kpi:sponsored_participation');
  assert.equal(f.state.selectedSource.value.unit, 'percent');
  Object.assign(f.state.sourceForm, { value: 101, source_description: 'Registro sponsor', confirmed: true });
  await f.state.saveSources();
  assert.equal(f.calls.filter(c => c[0] === 'put').length, 0);
  f.state.sourceForm.value = 42.5;
  await f.state.saveSources();
  const saved = f.calls.find(c => c[0] === 'put')[2].sources[0];
  assert.equal(saved.value, 42.5);
  assert.equal(saved.sponsor_id, 7);
  assert.equal(saved.source_type, 'manual_kpi');
  assert.equal(saved.verification_status, 'unverified');
});
test('explicit AI generation is cached per profile and comparison, without extra calls on switching', async t => {
  const f = await fixture(); t.after(() => f.scope.stop());
  await f.state.generateAnalysis();
  assert.equal(f.state.analysis.value.status, 'ready');
  f.state.recipient.value = 'marketing';
  assert.equal(f.state.analysis.value, undefined);
  f.state.recipient.value = 'executive';
  assert.equal(f.state.analysis.value.status, 'ready');
  f.state.compare.value = false;
  assert.equal(f.state.analysis.value, undefined);
  assert.equal(f.calls.filter(c => c[0] === 'post').length, 1);
});
test('source permission errors remain visible and retain unsaved input', async t => {
  const f = await fixture({ failure: { response: { status: 403, data: { message: 'Permessi insufficienti' } } } }); t.after(() => f.scope.stop());
  f.state.openSources(); Object.assign(f.state.sourceForm, { value: 100, source_description: 'Biglietteria', confirmed: true });
  await f.state.saveSources();
  assert.match(f.state.error.value, /Permessi/);
  assert.equal(f.state.sourceForm.value, 100);
  assert.equal(f.state.sourceOpen.value, true);
});
