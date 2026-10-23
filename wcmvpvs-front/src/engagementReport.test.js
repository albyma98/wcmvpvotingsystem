import test from 'node:test';
import assert from 'node:assert/strict';
import { REPORT_KPIS, calculateReportKpis, formatReportValue, reportOperationalItems, reportSourceLabel, validateReportReview } from './engagementReport.js';

test('review changes recalculate dependent rates without mutating measured values', () => {
  const measured = [{ id: 'visitors', value: 100 }, { id: 'active_participation', value: 40 }];
  const reviewed = calculateReportKpis({ visitors: 200, engagement_eligible_visitors: 200, timed_visitors: 20, active_visitors: 40, active_seconds: 1200 }, measured);
  assert.equal(reviewed.length, 20);
  assert.equal(reviewed.find(k => k.id === 'active_participation').value, 20);
  assert.equal(reviewed.find(k => k.id === 'active_participation').measured_value, 40);
  assert.equal(reviewed.find(k => k.id === 'active_time').value, 60);
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

test('operational participation does not invent visit or timing populations', () => {
  const facts = { visitors: 30, active_visitors: 30, multi_experience_visitors: 6, active_seconds: 120, engagement_eligible_visitors: null, timed_visitors: null };
  const operational = calculateReportKpis(facts);
  assert.equal(operational.find(kpi => kpi.id === 'active_participation').value, null);
  assert.equal(operational.find(kpi => kpi.id === 'active_time').value, null);
  assert.equal(operational.find(kpi => kpi.id === 'multi_experience').value, 20);
  const measured = calculateReportKpis({ ...facts, engagement_eligible_visitors: 60, timed_visitors: 3 });
  assert.equal(measured.find(kpi => kpi.id === 'active_participation').value, 50);
  assert.equal(measured.find(kpi => kpi.id === 'active_time').value, 40);
});

test('previous calculations retain their server-frozen denominator definitions', () => {
  const facts = { visitors: 20, active_visitors: 5, active_seconds: 200 };
  const original = [
    { id: 'active_participation', value: 25, numerator_key: 'active_visitors', denominator_key: 'visitors', source: 'new_tracking' },
    { id: 'active_time', value: 10, numerator_key: 'active_seconds', denominator_key: 'visitors', source: 'new_tracking' },
  ];
  const reviewed = calculateReportKpis(facts, original, {}, facts);
  assert.equal(reviewed.find(kpi => kpi.id === 'active_participation').value, 25);
  assert.equal(reviewed.find(kpi => kpi.id === 'active_participation').denominatorKey, 'visitors');
  assert.equal(reviewed.find(kpi => kpi.id === 'active_participation').corrected, false);
  assert.equal(reviewed.find(kpi => kpi.id === 'active_time').value, 10);
  assert.ok(validateReportReview({ ...facts, active_visitors: 21 }, {}, '', original).some(error => error.includes('numeratore')));
  assert.deepEqual(validateReportReview({ ...facts, engagement_eligible_visitors: 1 }, {}, '', original), []);
});

test('manual overrides preserve their justification and leave source facts intact', () => {
  const facts = { visitors: 100, engagement_eligible_visitors: 100, active_visitors: 40 };
  const reviewed = calculateReportKpis(facts, [{ id: 'active_participation', value: 40 }], { active_participation: { value: 42, note: 'Riconciliazione ingressi' } });
  const kpi = reviewed.find(k => k.id === 'active_participation');
  assert.equal(kpi.value, 42);
  assert.equal(kpi.calculated_value, 40);
  assert.equal(kpi.correction_note, 'Riconciliazione ingressi');
  assert.equal(facts.active_visitors, 40);
  assert.equal(kpi.corrected, true);
});

test('review retains backend coverage labels and provenance for reconciled operational data', () => {
  const facts = { visitors: 12, rounds_completed: 5 };
  const original = [
    { id: 'visitors', value: 12, label: 'Partecipanti digitali osservabili', source: 'mixed', note: 'Minimo verificabile con azioni e telemetria.', description: 'Identità riconciliate per evento.' },
    { id: 'completed_rounds', value: 5, label: 'Round confermati', source: 'operational', note: 'Conferme presenti nei dati operativi.' },
  ];
  const reviewed = calculateReportKpis(facts, original, {}, facts);
  const participants = reviewed.find(kpi => kpi.id === 'visitors');
  assert.equal(participants.label, original[0].label);
  assert.equal(participants.note, original[0].note);
  assert.equal(participants.description, original[0].description);
  assert.equal(participants.source, 'mixed');
  assert.equal(participants.corrected, false);
  assert.equal(reviewed.find(kpi => kpi.id === 'completed_rounds').source, 'operational');
  const changed = calculateReportKpis({ ...facts, visitors: 14 }, original, {}, facts).find(kpi => kpi.id === 'visitors');
  assert.equal(changed.label, original[0].label);
  assert.equal(changed.note, original[0].note);
  assert.equal(changed.source, 'manual');
});

test('operational totals retain measured zeroes and distinguish action counts from people', () => {
  const items = reportOperationalItems({ operational: { votes: 0, coins_earned: 120, feedback_responses: 2, sponsor_clicks: null, arbitrary: 1 } });
  assert.deepEqual(items.map(item => item.key), ['votes', 'coins_earned', 'feedback_responses']);
  assert.equal(items[0].value, 0);
  assert.deepEqual(reportOperationalItems(), []);
  assert.equal(reportSourceLabel('operational'), 'Dati operativi dell’evento');
  assert.equal(reportSourceLabel('mixed', true), 'Telemetria + operativi');
  assert.equal(reportSourceLabel('unknown'), 'Misura non disponibile');
});

test('proportional changes to a reference population remain visibly corrected', () => {
  const original = [{ id: 'active_participation', value: 50, numerator: 1, denominator: 2, source: 'new_tracking' }];
  const kpi = calculateReportKpis({ visitors: 4, engagement_eligible_visitors: 4, active_visitors: 2 }, original).find(k => k.id === 'active_participation');
  assert.equal(kpi.value, 50);
  assert.equal(kpi.corrected, true);
  assert.equal(kpi.source, 'manual');
  assert.equal(kpi.numerator, 2);
  assert.equal(kpi.denominator, 4);
});

test('a direct rate override never displays a contradictory measured fraction', () => {
  const kpi = calculateReportKpis({ visitors: 2, engagement_eligible_visitors: 2, active_visitors: 1 }, [{ id: 'active_participation', value: 50 }], { active_participation: { value: 70, note: 'Dato riconciliato' } }).find(k => k.id === 'active_participation');
  assert.equal(kpi.value, 70);
  assert.equal(kpi.numerator, null);
  assert.equal(kpi.denominator, null);
  assert.equal(kpi.calculated_numerator, 1);
  assert.equal(kpi.calculated_denominator, 2);
});

test('review rejects impossible rates, negative counts and unmotivated overrides', () => {
  assert.ok(validateReportReview({ visitors: 1, active_visitors: 2, engagement_eligible_visitors: null }, {}).includes('I partecipanti attivi non possono superare i partecipanti digitali osservabili.'));
  assert.ok(validateReportReview({ visitors: 10, engagement_eligible_visitors: 10, active_visitors: 11 }, {}).some(item => item.includes('numeratore')));
  assert.ok(validateReportReview({ visitors: -1 }, {}).length);
  assert.ok(validateReportReview({ visitors: 1.5 }, {}).length);
  assert.ok(validateReportReview({}, { visitors: { value: 5, note: '' } }).length);
  assert.ok(validateReportReview({}, { satisfaction: { value: 101, note: 'Corretto' } }).length);
  assert.deepEqual(validateReportReview({ visitors: 100, active_visitors: 40 }, { satisfaction: { value: null, note: 'Questionario non comparabile' } }), []);
  assert.equal(new Set(REPORT_KPIS.map(item => item.id)).size, 20);
});
