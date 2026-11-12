import test from 'node:test';
import assert from 'node:assert/strict';
import { setupSfc, testVue } from '../../test/sfcSetup.js';
import * as Report from '../engagementReport.js';
import * as Recipient from '../recipientReport.js';
import { recipientReportFixture } from '../../test/recipientReportFixture.js';

const flush = () => new Promise(setImmediate);
async function fixture({ failure, deferredAI, reviewFailure } = {}) {
  let current = recipientReportFixture();
  current.kpis.find(k => k.id === 'venue_adoption').value = null;
  current.kpis.find(k => k.id === 'venue_adoption').availability = { reason: 'manual_source_missing', message: 'Presenze mancanti', manual_allowed: true };
  current.kpis.find(k => k.id === 'sponsored_participation').value = null;
  current.kpis.find(k => k.id === 'sponsored_participation').availability = { reason: 'tracking_missing', message: 'Esposizione insufficiente', manual_allowed: false };
  current.manual_sources = [];
  const calls = [];
  const reviews = new Map();
  const reviewKey = (recipient, compare) => `${current.data_hash}|${recipient}|${compare}`;
  const getReview = (recipient, compare) => {
    const key = reviewKey(recipient, compare);
    if (!reviews.has(key)) reviews.set(key, { analysis: structuredClone(current.analysis), status: 'fallback', review: { state: 'draft', revision: 0, generation_version: 0 }, warnings: [] });
    return reviews.get(key);
  };
  const http = {
    get: async (path, options) => { calls.push(['get', path, options]); if (path === '/admin/sponsors') return { data: [] }; if (path.includes('/analysis/review?')) { const query = new URL(path, 'https://test.invalid').searchParams; return { data: structuredClone(getReview(query.get('recipient'), query.get('compare_previous') === 'true')) }; } return { data: structuredClone(current) }; },
    put: async (path, body) => {
      calls.push(['put', path, body]);
      if (path.endsWith('/analysis/review')) {
        if (reviewFailure) throw reviewFailure;
        const entry = getReview(body.recipient, body.compare_previous);
        if (body.analysis) entry.analysis = JSON.parse(JSON.stringify(body.analysis));
        entry.review = { ...entry.review, revision: entry.review.revision + 1, state: { draft: 'draft', approve: 'approved', discard: 'discarded' }[body.action], reviewed_by: 'reviewer', reviewed_at: '2026-11-05T10:00:00Z' };
        return { data: structuredClone(entry) };
      }
      if (failure) throw failure; const m = body.sources[0]; current.manual_sources = [{ ...m, revision: 1 }]; const k = current.kpis.find(k => k.id === 'venue_adoption'); k.value = 50; k.availability = { reason: 'available', manual_allowed: true }; k.provenance = current.manual_sources; k.source = 'manual_verified'; current.data_hash = 'updated'; return { data: { saved: true } };
    },
    post: async (path, body, options) => {
      calls.push(['post', path, body, options]);
      const entry = getReview(body.recipient, body.compare_previous);
      if (path.endsWith('/analysis')) {
        if (deferredAI) return deferredAI;
        if (body.regenerate || !entry.review.generation_version) { entry.analysis = structuredClone(current.analysis); entry.review = { state: 'draft', revision: entry.review.revision, generation_version: entry.review.generation_version + 1 }; }
        entry.status = 'ready'; return { data: structuredClone(entry) };
      }
      return { data: { ...structuredClone(current), analysis: body.include_ai ? structuredClone(entry.analysis) : undefined, analysis_review: body.include_ai ? structuredClone(entry.review) : undefined, recipient: current.profiles.find(p => p.id === body.recipient), selected_kpi_ids: current.profiles.find(p => p.id === body.recipient).kpi_ids, include_appendix: body.include_appendix } };
    },
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
  assert.equal(f.calls.filter(c => c[0] === 'get' && c[1].includes('engagement-report?')).length, 1);
});
test('official attendance is saved as a base fact and updates adoption', async t => {
  const f = await fixture(); t.after(() => f.scope.stop());
  assert.equal(f.state.missing.value.length, 1);
  f.state.recipient.value = 'sponsor';
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
  assert.equal(f.state.missing.value.length, 0);
});

test('missing sponsor data cannot be replaced by a manual KPI', async t => {
  const f = await fixture(); t.after(() => f.scope.stop());
  f.state.recipient.value = 'sponsor';
  assert.equal(f.state.selectedSource.value, null);
  assert.equal(f.state.kpis.value.find(k => k.id === 'sponsored_participation').value, null);
  await f.state.saveSources();
  assert.equal(f.calls.filter(c => c[0] === 'put').length, 0);
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

test('editable numeric drafts require approval, persist, and export only the approved revision', async t => {
  const f = await fixture(); t.after(() => f.scope.stop());
  assert.equal(f.state.analysis.value.review.state, 'draft');
  await f.state.renderReport(false);
  assert.match(f.state.error.value, /Approva/);
  assert.equal(f.calls.filter(c => c[0] === 'post').length, 0);
  await f.state.generateAnalysis();
  const edit = JSON.parse(JSON.stringify(f.state.analysis.value.analysis));
  edit.headline = 'Due opportunità da valutare';
  edit.summary = 'Sintesi verificata al 42%, modificabile dal responsabile.';
  edit.positive_insights[0].title = 'Titolo rivisto';
  edit.positive_insights[0].description = 'Testo rivisto';
  edit.recommendations[0].action = 'Indicazione rivista';
  edit.recommendations[0].reason = 'Motivazione rivista';
  edit.limitations[0] = 'Nota rivista';
  f.state.editAnalysis(edit);
  assert.equal(f.state.analysis.value.dirty, true);
  await f.state.reviewAnalysis('draft');
  assert.equal(f.state.analysisApproved.value, false);
  await f.state.loadAnalysisReview();
  assert.equal(f.state.analysis.value.analysis.summary, edit.summary);
  await f.state.reviewAnalysis('approve');
  assert.equal(f.state.analysisApproved.value, true);
  assert.equal(f.state.analysis.value.review.revision, 2);
  await f.state.renderReport(false);
  assert.ok(f.state.previewUrl.value);
  const saved = f.calls.filter(c => c[0] === 'post' && !c[1].endsWith('/analysis')).at(-1)[2];
  assert.equal(saved.analysis_review_revision, 2);
  assert.equal(saved.include_ai, true);
  assert.equal(f.state.approved.value.analysis.summary, edit.summary);
  assert.equal(f.calls.filter(c => c[0] === 'post' && c[1].endsWith('/analysis')).length, 1);
  edit.summary = 'Modifica dopo approvazione';
  f.state.editAnalysis(edit);
  assert.equal(f.state.analysisApproved.value, false);
  assert.equal(f.state.approved.value, null);
  assert.equal(f.state.previewUrl.value, '');
});

test('discard omits the analysis and regeneration requires approval again without extra automatic AI calls', async t => {
  const f = await fixture(); t.after(() => f.scope.stop());
  await f.state.reviewAnalysis('approve');
  await f.state.generateAnalysis(true);
  assert.equal(f.state.analysis.value.review.state, 'draft');
  assert.equal(f.state.analysisApproved.value, false);
  await f.state.reviewAnalysis('discard');
  assert.equal(f.state.analysis.value.review.state, 'discarded');
  assert.equal(f.state.includeAI.value, false);
  await f.state.renderReport(false);
  assert.ok(f.state.previewUrl.value);
  assert.equal(f.state.approved.value.analysis, undefined);
  assert.equal(f.calls.filter(c => c[0] === 'post' && c[1].endsWith('/analysis')).length, 1);
});

test('conflicting editorial saves retain the unsaved text and leave export unapproved', async t => {
  const f = await fixture({ reviewFailure: { response: { status: 409, data: { message: 'Bozza modificata in un’altra finestra' } } } }); t.after(() => f.scope.stop());
  const edit = JSON.parse(JSON.stringify(f.state.analysis.value.analysis)); edit.summary = 'Modifica da conservare';
  f.state.editAnalysis(edit);
  await f.state.reviewAnalysis('approve');
  assert.equal(f.state.analysis.value.analysis.summary, edit.summary);
  assert.equal(f.state.analysis.value.dirty, true);
  assert.equal(f.state.analysisApproved.value, false);
  assert.match(f.state.error.value, /altra finestra/);
  assert.equal(f.calls.filter(c => c[0] === 'post').length, 0);
});
