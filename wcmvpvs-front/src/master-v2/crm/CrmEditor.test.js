
import assert from 'node:assert/strict';
import { test } from 'node:test';
import { setupSfc, testVue } from '../../../test/sfcSetup.js';
import * as model from './model.js';

function editor(t, spec) {
  const result = setupSfc(new URL('./CrmEditor.vue', import.meta.url), {
    vue: testVue,
    '../ui/MasterDialog.vue': {},
    './model': model,
  }, { spec, records: [], operators: [{ id: 9, name: 'Master' }], organizations: [], events: [], today: '2026-11-09' });
  t.after(() => result.scope.stop());
  return result;
}

test('CRM change phase control submits a won opportunity with measured amount zero', (t) => {
  const { state, events } = editor(t, { kind: 'opportunity', mode: 'phase', initial: { id: 5, revision: 2, status: 'pilot', estimated: 95050 } });
  state.draft.status = 'won';
  state.draft.amount = 0;
  assert.equal(state.fields.value.find((item) => item.key === 'amount')?.required, true);
  state.submit();
  assert.equal(events[0][1].amount, 0);
  assert.equal(events[0][1].estimated, 95050);
  assert.equal(events[0][1].revision, 2);
});

test('CRM sends the contact and assigned follow-up together', (t) => {
  const { state, events } = editor(t, { kind: 'activity', initial: { kind: 'activity', type: 'call', title: 'Telefonata', contactId: 2, opportunityId: 5 } });
  state.draft.date = '2026-11-09T10:30';
  state.wantFollowUp.value = true;
  Object.assign(state.followUp, { title: 'Fissare demo', ownerId: 9 });
  state.submit();
  const [action, payload] = events[0];
  assert.equal(action, 'save');
  assert.equal(payload.date, '2026-11-09T10:30');
  assert.equal(payload.contactId, 2);
  assert.equal(payload.opportunityId, 5);
  assert.deepEqual(payload.followUp, { title: 'Fissare demo', due: '2026-11-09', ownerId: 9, priority: 'normal' });
});
