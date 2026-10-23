import test from 'node:test';
import assert from 'node:assert/strict';
import { REPORT_KPIS, calculateReportKpis, formatReportValue, validateReportReview } from './engagementReport.js';

test('review changes recalculate dependent rates without mutating measured values', () => {
  const measured = [{ id: 'visitors', value: 100 }, { id: 'active_participation', value: 40 }];
  const reviewed = calculateReportKpis({ visitors: 200, active_visitors: 40, active_seconds: 1200 }, measured);
  assert.equal(reviewed.length, 20);
  assert.equal(reviewed.find(k => k.id === 'active_participation').value, 20);
  assert.equal(reviewed.find(k => k.id === 'active_participation').measured_value, 40);
  assert.equal(reviewed.find(k => k.id === 'active_time').value, 6);
  assert.equal(measured[1].value, 40);
});

test('missing measurements remain missing and do not become zero engagement', () => {
  const reviewed = calculateReportKpis({ visitors: 0, active_visitors: 0, rounds_completed: 0 });
  assert.equal(reviewed.find(k => k.id === 'active_participation').value, null);
  assert.equal(reviewed.find(k => k.id === 'satisfaction').value, null);
  assert.equal(reviewed.find(k => k.id === 'completed_rounds').value, 0);
  assert.equal(formatReportValue(null), 'N/D');
  assert.equal(formatReportValue(0, 'percent'), '0%');
  assert.equal(formatReportValue(66.666, 'percent'), '66,7%');
});

test('manual overrides preserve their justification and leave source facts intact', () => {
  const facts = { visitors: 100, active_visitors: 40 };
  const reviewed = calculateReportKpis(facts, [{ id: 'active_participation', value: 40 }], { active_participation: { value: 42, note: 'Riconciliazione ingressi' } });
  const kpi = reviewed.find(k => k.id === 'active_participation');
  assert.equal(kpi.value, 42);
  assert.equal(kpi.calculated_value, 40);
  assert.equal(kpi.correction_note, 'Riconciliazione ingressi');
  assert.equal(facts.active_visitors, 40);
  assert.equal(kpi.corrected, true);
});

test('proportional changes to a reference population remain visibly corrected', () => {
  const original = [{ id: 'active_participation', value: 50, numerator: 1, denominator: 2, source: 'new_tracking' }];
  const kpi = calculateReportKpis({ visitors: 4, active_visitors: 2 }, original).find(k => k.id === 'active_participation');
  assert.equal(kpi.value, 50);
  assert.equal(kpi.corrected, true);
  assert.equal(kpi.source, 'manual');
  assert.equal(kpi.numerator, 2);
  assert.equal(kpi.denominator, 4);
});

test('a direct rate override never displays a contradictory measured fraction', () => {
  const kpi = calculateReportKpis({ visitors: 2, active_visitors: 1 }, [{ id: 'active_participation', value: 50 }], { active_participation: { value: 70, note: 'Dato riconciliato' } }).find(k => k.id === 'active_participation');
  assert.equal(kpi.value, 70);
  assert.equal(kpi.numerator, null);
  assert.equal(kpi.denominator, null);
  assert.equal(kpi.calculated_numerator, 1);
  assert.equal(kpi.calculated_denominator, 2);
});

test('review rejects impossible rates, negative counts and unmotivated overrides', () => {
  assert.ok(validateReportReview({ visitors: 10, active_visitors: 11 }, {}).some(item => item.includes('numeratore')));
  assert.ok(validateReportReview({ visitors: -1 }, {}).length);
  assert.ok(validateReportReview({ visitors: 1.5 }, {}).length);
  assert.ok(validateReportReview({}, { visitors: { value: 5, note: '' } }).length);
  assert.ok(validateReportReview({}, { satisfaction: { value: 101, note: 'Corretto' } }).length);
  assert.deepEqual(validateReportReview({ visitors: 100, active_visitors: 40 }, { satisfaction: { value: null, note: 'Questionario non comparabile' } }), []);
  assert.equal(new Set(REPORT_KPIS.map(item => item.id)).size, 20);
});
