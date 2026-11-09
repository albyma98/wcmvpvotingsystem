
import assert from 'node:assert/strict';
import { test } from 'node:test';
import { activeCheckIn, addDays, healthAssessment, metricText, precedingMatch, romeDay } from './model.js';

test('CRM uses Rome calendar days and keeps ND distinct from measured zero', () => {
  assert.equal(romeDay('2026-07-03T22:30:00Z'), '2026-07-04');
  assert.equal(addDays('2026-10-24', 2), '2026-10-26');
  assert.equal(metricText(0, 'percent'), '0%');
  assert.equal(metricText(null, 'percent'), 'ND');
});

test('trend compares closed matches with compatible KPI sources', () => {
  const base = { id: 'visitors', unit: 'count', description: 'Identità osservabili', source: 'new_tracking' };
  const match = (id, value, source = base.source) => ({ id, concluded: true, kpis: [{ ...base, value, source }] });
  const matches = [match(4, 0), match(3, null), match(2, 20, 'operational'), match(1, 8)];
  assert.equal(precedingMatch(matches)?.id, 1);
  assert.equal(precedingMatch(matches.slice(0, 3)), null);
  assert.equal(precedingMatch([{ ...matches[0], concluded: false }, matches[3]]), null);
});

test('customer health uses actual check-ins, critical issues and fixture cadence', () => {
  const company = { status: 'active', checkIn: '2026-10-15', cadenceDays: 10 };
  const checkin = { kind: 'activity', type: 'checkin', date: '2026-10-15T13:30:00Z' };
  const latest = { day: '2026-10-01' };
  const trend = { lastActivity: latest, events: [], matches: [] };
  const today = '2026-11-03';
  assert.equal(activeCheckIn(company, [checkin]), undefined);
  assert.equal(healthAssessment(company, [checkin], trend, today).status, 'insufficient');
  assert.equal(healthAssessment(company, [], trend, today).status, 'watch');
  assert.equal(healthAssessment(company, [checkin], { ...trend, events: [{ day: '2026-10-20', activated: false }] }, today).status, 'watch');
  assert.equal(healthAssessment({ ...company, status: 'lead' }, [], trend, today).status, 'insufficient');
  const issue = { kind: 'issue', status: 'open', priority: 'high', title: 'Materiali mancanti' };
  assert.equal(healthAssessment(company, [issue], trend, today).status, 'risk');
});
