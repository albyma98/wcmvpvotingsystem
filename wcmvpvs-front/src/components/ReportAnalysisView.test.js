import test from 'node:test';
import assert from 'node:assert/strict';
import { setupSfc } from '../../test/sfcSetup.js';
import { analysisMetricReferences } from '../recipientReport.js';
import { recipientReportFixture } from '../../test/recipientReportFixture.js';

test('all generated title and text fields emit editable copies while KPI references remain intact', t => {
  const analysis = recipientReportFixture().analysis;
  analysis.critical_insights = [{ title: 'Criticità', description: 'Da verificare', metric_ids: ['visitors'] }];
  const original = structuredClone(analysis);
  const f = setupSfc(new URL('./ReportAnalysisView.vue', import.meta.url), { '../recipientReport.js': { analysisMetricReferences } }, { analysis, review: { state: 'draft' }, busy: false, dirty: false, kpis: [], warnings: [] });
  t.after(() => f.scope.stop());
  const apply = action => { action(); const event = f.events.at(-1); assert.equal(event[0], 'update:analysis'); f.props.analysis = event[1]; };
  apply(() => f.state.setText('headline', 'Titolo modificato 42%'));
  apply(() => f.state.setText('summary', 'Sintesi modificata'));
  for (const list of ['positive_insights', 'critical_insights']) {
    apply(() => f.state.setItem(list, 0, 'title', 'Evidenza modificata'));
    apply(() => f.state.setItem(list, 0, 'description', 'Testo modificato'));
    assert.deepEqual([...f.props.analysis[list][0].metric_ids], ['visitors']);
  }
  apply(() => f.state.setItem('recommendations', 0, 'action', 'Azione modificata'));
  apply(() => f.state.setItem('recommendations', 0, 'reason', 'Motivo modificato'));
  apply(() => f.state.setItem('recommendations', 0, 'priority', 'high'));
  apply(() => f.state.setLimit(0, 'Nota modificata'));
  assert.equal(f.props.analysis.headline, 'Titolo modificato 42%');
  assert.equal(f.props.analysis.summary, 'Sintesi modificata');
  assert.equal(f.props.analysis.recommendations[0].action, 'Azione modificata');
  assert.equal(f.props.analysis.recommendations[0].reason, 'Motivo modificato');
  assert.equal(f.props.analysis.limitations[0], 'Nota modificata');
  assert.deepEqual(analysis, original);
  f.props.busy = true;
  const count = f.events.length;
  f.state.setText('summary', 'Non applicare durante il salvataggio');
  assert.equal(f.events.length, count);
});
