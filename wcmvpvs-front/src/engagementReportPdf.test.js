import test from 'node:test';
import assert from 'node:assert/strict';
import { createEngagementReportPdf, pdfText } from './engagementReportPdf.js';
import { engagementReportFixture } from '../test/engagementReportFixture.js';
import { REPORT_KPIS } from './engagementReport.js';

function pageText(doc) { return doc.internal.pages.flat().join('\n'); }

test('one pager contains all twenty KPIs and report identity without a visible issuer name', () => {
  const report = engagementReportFixture('one_page');
  const doc = createEngagementReportPdf(report);
  assert.equal(doc.getNumberOfPages(), 1);
  assert.ok(Math.abs(doc.internal.pageSize.getWidth() - 210) < .01);
  assert.ok(Math.abs(doc.internal.pageSize.getHeight() - 297) < .01);
  const rendered = pageText(doc);
  const nativeText = [...rendered.matchAll(/\(([^)]*)\) Tj/g)].map(match => match[1]).join(' ');
  for (const kpi of REPORT_KPIS) assert.ok(nativeText.includes(pdfText(kpi.label)), kpi.label);
  assert.ok(rendered.includes('Report #42'));
  assert.ok(!rendered.includes('ArenaBoostX'));
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
  assert.ok(!rendered.includes('REVISIONI:'));
  assert.ok(rendered.includes('3/3'));
});

test('combined-source PDFs preserve coverage-specific labels and distinguish operational sources', () => {
  for (const format of ['one_page', 'extended']) {
    const report = engagementReportFixture(format);
    report.overrides = {};
    Object.assign(report.kpis.find(kpi => kpi.id === 'visitors'), { label: 'Partecipanti osservabili', source: 'mixed', corrected: false });
    Object.assign(report.kpis.find(kpi => kpi.id === 'completed_rounds'), { source: 'operational' });
    report.details.coverage_notes = ['Partecipazione minima verificabile. Visite passive non ricostruibili.'];
    report.details.operational = { votes: 14, shop_redemptions: 3, coins_earned: 1200, coins_spent: 150, feedback_responses: 8, sponsor_clicks: 5, sponsor_clicks_without_identity: 2 };
    report.details.sources = [{ table: 'votes', rows: 14 }, { table: 'engagement_report_signals', rows: 3 }];
    report.notes = '';
    const doc = createEngagementReportPdf(report);
    assert.equal(doc.getNumberOfPages(), format === 'one_page' ? 1 : 3);
    const rendered = pageText(doc);
    assert.ok(rendered.includes('Partecipanti osservabili'));
    assert.ok(rendered.includes('Partecipazione minima verificabile.'));
    assert.ok(rendered.includes('14 voti'));
    assert.ok(rendered.includes('3 riscatti'));
    assert.ok(rendered.includes('coin +1200/-150'));
    assert.ok(rendered.includes('8 feedback'));
    assert.ok(rendered.includes('5 clic sponsor'));
    if (format === 'extended') {
      assert.ok(rendered.includes('Telemetria + operativi'));
      assert.ok(rendered.includes('Dati operativi'));
      assert.ok(rendered.includes('Voti registrati: 14'));
      assert.ok(rendered.includes('2 tabelle consultate.'));
    }
  }
});

test('sponsor detail leaves exposure metrics unavailable for a partner observed only through clicks', () => {
  const report = engagementReportFixture('extended');
  report.details.sponsors = [{ id: 91, name: 'Partner con soli clic', exposure_measured: false, reach: 0, impressions: 0, clickers: 0, ctr: 0 }];
  const doc = createEngagementReportPdf(report);
  const page = doc.internal.pages[2].join('\n');
  const nativeText = [...page.matchAll(/\(([^)]*)\) Tj/g)].map(match => match[1]);
  assert.ok(nativeText.includes('Partner con soli clic'));
  assert.equal(nativeText.filter(text => text === 'N/D').length, 4);
  assert.ok(!nativeText.includes('0%'));
  report.details.sponsors[0].exposure_measured = true;
  const measured = createEngagementReportPdf(report).internal.pages[2].join('\n');
  assert.ok(measured.includes('(0%)'));
  delete report.details.sponsors[0].exposure_measured;
  const previous = createEngagementReportPdf(report).internal.pages[2].join('\n');
  assert.ok(previous.includes('(0%)'));
});

test('corrections hide measured fractions without exporting markers or reasons from older snapshots', () => {
  const report = engagementReportFixture('extended');
  report.overrides = { active_participation: { value: 70, note: 'Valore finale riconciliato' } };
  const participation = report.kpis.find(kpi => kpi.id === 'active_participation');
  Object.assign(participation, { value: 70, numerator: 1, denominator: 2 });
  const satisfaction = report.kpis.find(kpi => kpi.id === 'satisfaction');
  Object.assign(satisfaction, { corrected: true, source: 'manual', note: 'Rettifica documentata questionario' });
  report.notes = 'Contesto generale facoltativo dell’evento';
  const rendered = pageText(createEngagementReportPdf(report));
  assert.ok(!rendered.includes('(1/2)'));
  assert.ok(!rendered.includes('(1 / 2)'));
  assert.ok(!rendered.includes('Rettifica documentata questionario'));
  assert.ok(!rendered.includes('Valore finale riconciliato'));
  assert.ok(rendered.includes(pdfText('Contesto generale facoltativo dell’evento')));
  assert.ok(!rendered.includes('REVISIONI:'));
  assert.ok(!rendered.includes('*'));
});

test('optional general notes remain plain text in both formats', () => {
  for (const format of ['one_page', 'extended']) {
    const report = engagementReportFixture(format);
    report.notes = 'Contesto generale condiviso con la società.';
    const rendered = pageText(createEngagementReportPdf(report));
    assert.ok(rendered.includes(pdfText(`Note: ${report.notes}`)));
    assert.ok(!rendered.includes('REVISIONI:'));
    assert.ok(!rendered.includes('LETTURA DEI DATI'));
    report.notes = '';
    assert.ok(!pageText(createEngagementReportPdf(report)).includes(pdfText('Contesto generale condiviso con la società.')));
  }
});

test('both formats print all optional KPI descriptions and omit removed explanatory cards', () => {
  for (const format of ['one_page', 'extended']) {
    const report = engagementReportFixture(format);
    report.kpis.forEach((kpi, index) => { kpi.display_description = `Descrizione strategica KPI ${String(index + 1).padStart(2, '0')}.`; });
    const doc = createEngagementReportPdf(report);
    const rendered = pageText(doc);
    assert.equal(doc.getNumberOfPages(), format === 'one_page' ? 1 : 3);
    for (const kpi of report.kpis) assert.ok(rendered.includes(kpi.display_description), kpi.id);
    assert.ok(!rendered.includes('LETTURA DEI DATI'));
    assert.ok(!rendered.includes('COME UTILIZZARE QUESTI RISULTATI'));
    assert.ok(!rendered.includes('0.91 0.96 0.94 rg'), 'the removed pale-green card fill must not be emitted');
    report.kpis[0].display_description = '';
    report.kpis[0].description = 'Definizione tecnica che non deve apparire come mini descrizione';
    delete report.kpis[1].display_description;
    const without = pageText(createEngagementReportPdf(report));
    assert.ok(!without.includes('Descrizione strategica KPI 01.'));
    assert.ok(!without.includes('Descrizione strategica KPI 02.'));
    assert.ok(without.includes('Descrizione strategica KPI 03.'));
    assert.ok(!without.includes(report.kpis[0].description), 'older snapshots do not receive a description fallback');
  }
});

test('missing data, extreme names and long notes remain bounded with explicit abbreviations', () => {
  for (const format of ['one_page', 'extended']) {
    const report = engagementReportFixture(format, true);
    report.kpis.forEach(kpi => { kpi.display_description = 'Descrizione strategica modificabile con testo lungo da verificare nella pagina e nel contesto dell’evento. '.repeat(2).slice(0, 120); });
    const doc = createEngagementReportPdf(report);
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

test('both report formats sign every page with only the logo and no issuer-name fallback', () => {
  // A small opaque test image, generated locally rather than a third-party logo.
  const logo = 'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABgAAAAMCAIAAAD3UuoiAAAAGklEQVR4nGPkqUhloAZgooopowaNGjRsDQIAR8oBAcx3rNgAAAAASUVORK5CYII=';
  for (const format of ['one_page', 'extended']) {
    const report = engagementReportFixture(format);
    report.branding = { company_name: 'Issuer name must remain invisible', logo_data: logo };
    const branded = createEngagementReportPdf(report);
    assert.equal(Object.keys(branded.internal.collections.addImage_images).length, 1);
    assert.ok(!pageText(branded).includes(report.branding.company_name));
    for (const page of branded.internal.pages.slice(1)) assert.equal((page.join('\n').match(/\/I\d+ Do/g) || []).length, 2);
    for (const unavailable of ['', 'data:image/png;base64,invalid']) {
      report.branding.logo_data = unavailable;
      const unbranded = createEngagementReportPdf(report);
      assert.ok(!pageText(unbranded).includes(report.branding.company_name));
      assert.ok(!pageText(unbranded).includes('ArenaBoostX'));
    }
  }
});
