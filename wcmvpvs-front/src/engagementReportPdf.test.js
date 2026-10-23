import test from 'node:test';
import assert from 'node:assert/strict';
import { createEngagementReportPdf, pdfText } from './engagementReportPdf.js';
import { engagementReportFixture } from '../test/engagementReportFixture.js';
import { REPORT_KPIS } from './engagementReport.js';

function pageText(doc) { return doc.internal.pages.flat().join('\n'); }

test('one pager contains all twenty KPIs, report identity and issuer on one A4 page', () => {
  const report = engagementReportFixture('one_page');
  const doc = createEngagementReportPdf(report);
  assert.equal(doc.getNumberOfPages(), 1);
  assert.ok(Math.abs(doc.internal.pageSize.getWidth() - 210) < .01);
  assert.ok(Math.abs(doc.internal.pageSize.getHeight() - 297) < .01);
  const rendered = pageText(doc);
  const nativeText = [...rendered.matchAll(/\(([^)]*)\) Tj/g)].map(match => match[1]).join(' ');
  for (const kpi of REPORT_KPIS) assert.ok(nativeText.includes(pdfText(kpi.label)), kpi.label);
  assert.ok(rendered.includes('Report #42'));
  assert.ok(rendered.includes('ArenaBoostX'));
  assert.ok(doc.output('arraybuffer').byteLength > 3000);
  assert.ok(doc.output().includes('/Author (ArenaBoostX)'));
});

test('extended report stays at three pages with measured game and sponsor details', () => {
  const doc = createEngagementReportPdf(engagementReportFixture('extended'));
  assert.equal(doc.getNumberOfPages(), 3);
  const rendered = pageText(doc);
  assert.ok(rendered.includes('Basketball Shootout'));
  assert.ok(rendered.includes('Reaction Test'));
  assert.ok(rendered.includes('Partner principale'));
  assert.ok(rendered.includes('REVISIONI:'));
  assert.ok(rendered.includes('3/3'));
});

test('direct overrides hide measured fractions and base-fact corrections retain their reason', () => {
  const report = engagementReportFixture('extended');
  report.overrides = { active_participation: { value: 70, note: 'Valore finale riconciliato' } };
  const participation = report.kpis.find(kpi => kpi.id === 'active_participation');
  Object.assign(participation, { value: 70, numerator: 1, denominator: 2 });
  const satisfaction = report.kpis.find(kpi => kpi.id === 'satisfaction');
  Object.assign(satisfaction, { corrected: true, source: 'manual', note: 'Rettifica documentata questionario' });
  const rendered = pageText(createEngagementReportPdf(report));
  assert.ok(!rendered.includes('(1/2)'));
  assert.ok(!rendered.includes('(1 / 2)'));
  assert.ok(rendered.includes('Rettifica documentata questionario'));
});

test('missing data, extreme names and long notes remain bounded with explicit abbreviations', () => {
  for (const format of ['one_page', 'extended']) {
    const doc = createEngagementReportPdf(engagementReportFixture(format, true));
    assert.equal(doc.getNumberOfPages(), format === 'one_page' ? 1 : 3);
    const rendered = pageText(doc);
    assert.ok(rendered.includes('N/D'));
    assert.ok(rendered.includes('...'));
    assert.ok(!rendered.includes('NaN'));
    for (const [, x, y] of rendered.matchAll(/(-?\d+(?:\.\d+)?) (-?\d+(?:\.\d+)?) Td/g)) {
      assert.ok(Number(x) >= 0 && Number(x) <= 596, `x position ${x}`);
      assert.ok(Number(y) >= 8 && Number(y) <= 842, `y position ${y}`);
    }
  }
});

test('unapproved snapshots and unsupported formats cannot be exported', () => {
  assert.throws(() => createEngagementReportPdf({ kpis: [], format: 'one_page' }), /approvata/);
  assert.throws(() => createEngagementReportPdf({ ...engagementReportFixture(), format: 'infinite' }), /Formato/);
  assert.equal(pdfText('Società 🏀 — città'), 'Società - città');
});

test('validated issuer PNG is embedded while an unreadable logo retains the issuer name', () => {
  const report = engagementReportFixture();
  // A small opaque test image, generated locally rather than a third-party logo.
  report.branding.logo_data = 'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABgAAAAMCAIAAAD3UuoiAAAAGklEQVR4nGPkqUhloAZgooopowaNGjRsDQIAR8oBAcx3rNgAAAAASUVORK5CYII=';
  const branded = createEngagementReportPdf(report);
  assert.equal(Object.keys(branded.internal.collections.addImage_images).length, 1);
  report.branding.logo_data = 'data:image/png;base64,invalid';
  assert.ok(pageText(createEngagementReportPdf(report)).includes('ArenaBoostX'));
});
