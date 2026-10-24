import test from 'node:test';
import assert from 'node:assert/strict';
import { createEngagementReportPdf, pdfText } from './engagementReportPdf.js';
import { engagementReportFixture as completeReportFixture } from '../test/engagementReportFixture.js';
import { REPORT_KPIS } from './engagementReport.js';

function pageText(doc) { return doc.internal.pages.flat().join('\n'); }

function engagementReportFixture(format = 'one_page', stress = false) {
  const report = completeReportFixture(format, stress);
  report.kpis = report.kpis.slice(0, 20);
  delete report.selected_kpi_ids;
  return report;
}

function thirtyKpiFixture() {
  const report = completeReportFixture('extended');
  report.kpis = REPORT_KPIS.map((definition, index) => ({ ...definition, value: definition.unit === 'percent' ? 30 + index : 1200 + index, source: ['estimated_total_time', 'estimated_avg_time'].includes(definition.id) ? 'estimated' : 'operational', display_description: `Descrizione selezionabile ${String(index + 1).padStart(2, '0')}.` }));
  report.overrides = {};
  report.selected_kpi_ids = report.kpis.map(kpi => kpi.id);
  report.details.time_estimate = { method: 'Intervalli tra azioni con limite per intervallo e inattività.', covered_visitors: 380, total_visitors: 520, excluded_visitors: 140 };
  return report;
}

test('one pager contains all twenty KPIs and report identity without a visible issuer name', () => {
  const report = engagementReportFixture('one_page');
  const doc = createEngagementReportPdf(report);
  assert.equal(doc.getNumberOfPages(), 1);
  assert.ok(Math.abs(doc.internal.pageSize.getWidth() - 210) < .01);
  assert.ok(Math.abs(doc.internal.pageSize.getHeight() - 297) < .01);
  const rendered = pageText(doc);
  const nativeText = [...rendered.matchAll(/\(([^)]*)\) Tj/g)].map(match => match[1]).join(' ');
  for (const kpi of report.kpis) assert.ok(nativeText.includes(pdfText(kpi.label)), kpi.label);
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
    for (const id of ['shop_redemptions', 'feedback_collected', 'sponsor_clicks']) report.kpis.push({ ...REPORT_KPIS.find(kpi => kpi.id === id), value: 3, source: 'operational' });
    report.selected_kpi_ids = [...report.kpis.slice(0, 17).map(kpi => kpi.id), 'shop_redemptions', 'feedback_collected', 'sponsor_clicks'];
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
    assert.ok(rendered.includes('coin virtuali +1200/-150'));
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

test('selection controls overview cards, KPI rows, source rows and PDF metadata', () => {
  for (const format of ['one_page', 'extended']) {
    const report = engagementReportFixture(format);
    report.selected_kpi_ids = ['satisfaction', 'perks_interest'];
    report.notes = '';
    report.kpis.forEach(kpi => { kpi.display_description = `Testo specifico ${kpi.id}`; });
    const doc = createEngagementReportPdf(report);
    const rendered = pageText(doc);
    assert.ok(rendered.includes(pdfText('Soddisfazione esperienza')));
    assert.ok(rendered.includes(pdfText('Interesse per premi e vantaggi')));
    assert.ok(rendered.includes('2 indicatori selezionati'));
    for (const kpi of report.kpis.filter(kpi => !report.selected_kpi_ids.includes(kpi.id))) {
      assert.ok(!rendered.includes(pdfText(kpi.label)), kpi.label);
      assert.ok(!rendered.includes(`Testo specifico ${kpi.id}`), kpi.id);
    }
    assert.ok(doc.output().includes('/Subject (2 KPI - Report approvato 42)'));
  }
});

test('legacy approved snapshots export their actual twenty KPIs without inventing new indicators', () => {
  const report = engagementReportFixture('extended');
  assert.equal(report.kpis.length, 20);
  const rendered = pageText(createEngagementReportPdf(report));
  assert.ok(rendered.includes('20 indicatori selezionati'));
  for (const definition of REPORT_KPIS.slice(20)) assert.ok(!rendered.includes(pdfText(definition.label)), definition.id);
  report.selected_kpi_ids = ['estimated_total_time'];
  assert.throws(() => createEngagementReportPdf(report), /non presente/);
});

test('PDF rejects empty, unknown, duplicate and over-limit selections before rendering', () => {
  const report = thirtyKpiFixture();
  assert.equal(report.kpis.length, 30);
  for (const selection of [[], null, 'visitors']) {
    report.selected_kpi_ids = selection;
    assert.throws(() => createEngagementReportPdf(report), /almeno una KPI/);
  }
  report.selected_kpi_ids = ['visitors', 'visitors'];
  assert.throws(() => createEngagementReportPdf(report), /duplicati/);
  report.selected_kpi_ids = ['unknown-kpi'];
  assert.throws(() => createEngagementReportPdf(report), /non presente/);
  report.selected_kpi_ids = report.kpis.map(kpi => kpi.id);
  report.format = 'one_page';
  assert.throws(() => createEngagementReportPdf(report), /massimo 20 KPI/);
  report.selected_kpi_ids = report.selected_kpi_ids.slice(0, 20);
  assert.equal(createEngagementReportPdf(report).getNumberOfPages(), 1);
});

test('all thirty selected indicators and their descriptions fit in exactly three pages', () => {
  const report = thirtyKpiFixture();
  const doc = createEngagementReportPdf(report);
  assert.equal(doc.getNumberOfPages(), 3);
  const rendered = pageText(doc);
  assert.ok(doc.internal.pages[1].join('\n').includes('1-15 di 30'));
  assert.ok(doc.internal.pages[2].join('\n').includes('16-30 di 30'));
  for (const kpi of report.kpis) {
    assert.ok(rendered.includes(pdfText(kpi.label)), kpi.id);
    assert.ok(rendered.includes(kpi.display_description), kpi.id);
  }
  assert.ok(rendered.includes('Basketball Shootout'));
  assert.ok(rendered.includes('Partner principale'));
  assert.ok(rendered.includes('Fonti delle 30 KPI selezionate:'));
  assert.ok(!rendered.includes('LETTURA DEI DATI'));
  assert.ok(!rendered.includes('COME UTILIZZARE QUESTI RISULTATI'));
  assert.ok(!rendered.includes('*'));
});

test('selection above twenty never drops selected rows and excludes unselected KPI descriptions', () => {
  const report = thirtyKpiFixture();
  report.selected_kpi_ids = report.kpis.slice(0, 21).map(kpi => kpi.id);
  const doc = createEngagementReportPdf(report);
  const rendered = pageText(doc);
  assert.equal(doc.getNumberOfPages(), 3);
  assert.ok(doc.internal.pages[2].join('\n').includes('16-21 di 21'));
  for (const kpi of report.kpis.slice(0, 21)) assert.ok(rendered.includes(kpi.display_description), kpi.id);
  for (const kpi of report.kpis.slice(21)) assert.ok(!rendered.includes(kpi.display_description), kpi.id);
});

test('estimated time is labeled beside values and includes only supplied method and coverage', () => {
  for (const format of ['one_page', 'extended']) {
    const report = thirtyKpiFixture();
    report.format = format;
    report.selected_kpi_ids = ['estimated_total_time', 'estimated_avg_time'];
    report.notes = '';
    const total = report.kpis.find(kpi => kpi.id === 'estimated_total_time');
    total.source = 'manual';
    total.corrected = true;
    const rendered = pageText(createEngagementReportPdf(report));
    assert.ok(rendered.includes('(Stima)'));
    assert.ok(rendered.includes('Stima del tempo:'));
    assert.ok(rendered.includes('Copertura: 380/520 partecipanti osservabili'));
    assert.ok(rendered.includes('140 senza intervalli stimabili'));
    assert.ok(rendered.includes(pdfText(report.details.time_estimate.method)));
    if (format === 'extended') assert.ok(rendered.includes('Stima verificata'));
    delete report.details.time_estimate;
    const noCoverage = pageText(createEngagementReportPdf(report));
    assert.ok(noCoverage.includes('(Stima)'));
    assert.ok(!noCoverage.includes('Copertura:'));
    report.selected_kpi_ids = ['visitors'];
    report.details.time_estimate = { method: 'Metodo escluso insieme alle KPI stimate', covered_visitors: 380, total_visitors: 520 };
    assert.ok(!pageText(createEngagementReportPdf(report)).includes('Metodo escluso'));
  }
});

test('time-only PDFs omit unselected operational totals and unrelated game and sponsor tables', () => {
  for (const format of ['one_page', 'extended']) {
    const report = thirtyKpiFixture();
    report.format = format;
    report.selected_kpi_ids = ['estimated_total_time', 'estimated_avg_time'];
    report.notes = '';
    report.details.operational = { votes: 99111, identified_participants: 99222, game_reward_receipts: 99333, shop_redemptions: 99444, coins_earned: 99555, coins_spent: 99666, feedback_responses: 99777, sponsor_clicks: 99888, sponsor_clicks_without_identity: 99999 };
    Object.assign(report.details.time_estimate, { interval_cap_seconds: 60, gap_limit_seconds: 300 });
    const rendered = pageText(createEngagementReportPdf(report));
    for (const label of ['Voti registrati', 'Profili identificati attivi', 'Premi di gioco confermati', 'Riscatti shop confermati', 'Risposte al feedback', 'Clic sponsor registrati', 'Coin virtuali', 'Basketball Shootout', 'Partner principale', 'Operativo:']) assert.ok(!rendered.includes(label), label);
    assert.ok(rendered.includes('Copertura: 380/520 partecipanti osservabili'));
    assert.ok(rendered.includes('Intervalli fino a 60 s; pause ammesse fino a 300 s'));
    const methodStart = rendered.indexOf('Intervalli tra azioni');
    assert.ok(methodStart === -1 || rendered.indexOf('Copertura: 380/520') < methodStart);
  }
});

test('operational count KPIs excluded from selection cannot reappear as raw totals', () => {
  const report = thirtyKpiFixture();
  report.selected_kpi_ids = report.kpis.filter(kpi => !['shop_redemptions', 'feedback_collected', 'sponsor_clicks'].includes(kpi.id)).map(kpi => kpi.id);
  report.details.operational = { votes: 14, shop_redemptions: 33441, coins_earned: 1200, coins_spent: 150, feedback_responses: 33442, sponsor_clicks: 33443, sponsor_clicks_without_identity: 33444 };
  const rendered = pageText(createEngagementReportPdf(report));
  for (const label of ['Riscatti shop confermati', 'Risposte al feedback', 'Clic sponsor registrati', 'Clic sponsor senza identità', '33.441', '33.442', '33.443', '33.444']) assert.ok(!rendered.includes(pdfText(label)), label);
  assert.ok(rendered.includes('Coin virtuali guadagnati'));
});

test('experience detail columns follow the selected related KPIs', () => {
  const report = thirtyKpiFixture();
  report.selected_kpi_ids = ['completed_rounds', 'sponsor_impressions'];
  report.notes = '';
  const page = createEngagementReportPdf(report).internal.pages[2].join('\n');
  assert.ok(page.includes('(COMPLETATI)'));
  assert.ok(page.includes('(ESPOSIZIONI)'));
  for (const column of ['GIOCATORI', 'AVVIATI', 'COPERTURA', 'UTENTI CLIC', 'TASSO']) assert.ok(!page.includes(`(${column})`), column);
});

test('generic backend time coverage notes do not leak unselected estimated KPI measurements', () => {
  for (const format of ['one_page', 'extended']) {
    const report = thirtyKpiFixture();
    report.format = format;
    report.selected_kpi_ids = report.kpis.slice(0, 20).map(kpi => kpi.id);
    report.notes = '';
    report.details.coverage_notes = ['Tempo stimato: durata ricostruibile per 777 partecipanti su 999.', 'Copertura verificabile delle fonti operative.'];
    const rendered = pageText(createEngagementReportPdf(report));
    assert.ok(!rendered.includes('Tempo stimato:'));
    assert.ok(!rendered.includes('777 partecipanti'));
    assert.ok(!rendered.includes('Stima del tempo:'));
    assert.ok(rendered.includes('Copertura verificabile delle fonti operative.'));
    report.selected_kpi_ids = ['estimated_total_time'];
    const selected = pageText(createEngagementReportPdf(report));
    assert.ok(!selected.includes('Tempo stimato:'));
    assert.ok(selected.includes('Copertura: 380/520 partecipanti osservabili'));
  }
});
