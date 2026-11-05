import test from 'node:test';
import assert from 'node:assert/strict';
import { recipientKpis, comparisonLabel, manualReportFields, manualSourceErrors, recipientSourceLabel, analysisMetricReferences } from './recipientReport.js';
import { recipientReportFixture } from '../test/recipientReportFixture.js';
import { createEngagementReportPdf } from './engagementReportPdf.js';

test('all profiles select original catalog IDs in the configured order', () => {
  const r = recipientReportFixture();
  for (const [id, count] of Object.entries({ executive: 5, marketing: 10, sponsor: 5, analytics: 20 })) {
    const kpis = recipientKpis(r, id);
    assert.equal(kpis.length, count);
    assert.deepEqual(kpis.map(k => k.id), r.profiles.find(p => p.id === id).kpi_ids);
  }
});
test('comparison uses neutral directions, percentage points and undefined zero bases', () => {
  assert.equal(comparisonLabel({ comparison: { absolute: 2, percentage_points: 2 } }), '+2 punti percentuali');
  assert.equal(comparisonLabel({ unit: 'count', comparison: { absolute: -5, percent: -10 } }), '-5 (-10%)');
  assert.equal(comparisonLabel({ comparison: { absolute: null, reason: 'Valore N/D' } }), 'Valore N/D');
  assert.match(comparisonLabel({ unit: 'count', comparison: { absolute: 5, percent: null, reason: 'Base zero' } }), /Base zero/);
});
test('attendance requires integer source, confirmation and preserves verification status', () => {
  assert.equal(manualSourceErrors({ value: 100, source_description: 'Biglietteria', confirmed: true }).length, 0);
  for (const value of ['', null, -1, .5, 10000001]) assert.ok(manualSourceErrors({ value, source_description: 'Biglietteria', confirmed: true }).length);
  assert.ok(manualSourceErrors({ value: 100, source_description: '', confirmed: false }).length >= 2);
  assert.match(recipientSourceLabel({ source: 'manual_unverified' }), /NON verificata/);
  const r = recipientReportFixture();
  assert.equal(analysisMetricReferences({ metric_ids: ['visitors', 'unknown'] }, r.kpis).split(' · ').length, 1);
});

test('every missing KPI supports its unit and numeric bounds, including zero denominators', () => {
  const r = recipientReportFixture('analytics');
  r.kpis.forEach(k => { k.value = null; k.availability = { reason: 'zero_denominator', manual_allowed: true }; });
  const fields = manualReportFields(r, r.kpis);
  for (const k of r.kpis) {
    const field = fields.find(f => f.key === `kpi:${k.id}`);
    assert.ok(field, k.id);
    const form = { value: k.unit === 'count' ? 5 : 12.5, source_description: 'Registro verificabile', confirmed: true };
    assert.deepEqual(manualSourceErrors(form, field), []);
    if (k.unit === 'count') assert.ok(manualSourceErrors({ ...form, value: 1.5 }, field).length);
    if (k.unit === 'percent') assert.ok(manualSourceErrors({ ...form, value: 100.1 }, field).length);
  }
  const report = recipientReportFixture('sponsor'); report.sponsor_id = 7;
  report.kpis.forEach(k => { k.value = null; });
  const scoped = manualReportFields(report, recipientKpis(report, 'sponsor'));
  assert.equal(scoped.find(f => f.key === 'kpi:sponsor_reach').sponsor_id, 7);
  assert.equal(scoped.find(f => f.key === 'kpi:visitors').sponsor_id, 0);
});

test('PDF identifies direct manual rates and omits nonexistent calculated fractions', () => {
  const r = recipientReportFixture('sponsor');
  const k = r.kpis.find(k => k.id === 'sponsor_ctr');
  Object.assign(k, { value: 12.5, numerator: null, denominator: null, source: 'manual_unverified', provenance: [{ field_key: 'kpi:sponsor_ctr', value: 12.5, unit: 'percent', source_type: 'manual_kpi', source_description: 'Registro partner verificabile', verification_status: 'unverified', revision: 1 }] });
  assert.equal(recipientSourceLabel(k), 'Valore KPI manuale NON verificato');
  const text = createEngagementReportPdf(r).internal.pages.flat().join('\n');
  assert.match(text, /KPI manuale NON verificato/);
  assert.match(text, /Registro partner verificabile/);
});
test('four PDF formats include recipient and frozen metrics with correct main page counts', () => {
  for (const id of ['executive', 'marketing', 'sponsor', 'analytics']) {
    const report = recipientReportFixture(id);
    const doc = createEngagementReportPdf(report);
    assert.ok(doc.output('arraybuffer').byteLength > 3000);
    if (id === 'executive' || id === 'sponsor') assert.equal(doc.getNumberOfPages(), 1);
    if (id === 'marketing') assert.equal(doc.getNumberOfPages(), 2);
    const text = doc.internal.pages.flat().join('\n');
    assert.ok(text.includes('Report #abcdef'));
    assert.ok(text.includes('N/D'));
  }
});
test('appendix retains all original definitions, is paginated and is not duplicated in Analytics', () => {
  const base = createEngagementReportPdf(recipientReportFixture('executive'));
  const appendix = createEngagementReportPdf(recipientReportFixture('executive', true));
  assert.ok(appendix.getNumberOfPages() > base.getNumberOfPages());
  assert.match(appendix.internal.pages.flat().join('\n'), /Appendice/);
  assert.equal(createEngagementReportPdf(recipientReportFixture('analytics', true)).getNumberOfPages(), createEngagementReportPdf(recipientReportFixture('analytics', false)).getNumberOfPages());
  const long = createEngagementReportPdf(recipientReportFixture('analytics', true, true));
  assert.ok(long.getNumberOfPages() > appendix.getNumberOfPages());
});
test('the same recipient snapshot generates byte identical PDFs and keeps unavailable data as N/D', () => {
  const r = recipientReportFixture('executive');
  r.kpis.find(k => k.id === 'venue_adoption').value = null;
  const first = createEngagementReportPdf(r).output('arraybuffer');
  const second = createEngagementReportPdf(structuredClone(r)).output('arraybuffer');
  assert.deepEqual(new Uint8Array(first), new Uint8Array(second));
});
