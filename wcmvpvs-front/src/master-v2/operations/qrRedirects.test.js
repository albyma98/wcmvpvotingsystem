import assert from 'node:assert/strict';
import test from 'node:test';
import { setupSfc, testVue } from '../../../test/sfcSetup.js';
import { dateLabel, number } from './analyticsFormat.js';

const ENDPOINT = '/admin/master/qr-redirects';
const original = { id: 1, source_path: '/qr', target_path: '/club' };

function fixture({ failFirstDelete = false, invalidResponse = false } = {}) {
  const calls = [];
  const rows = [original];
  const masterApi = {
    async get() { return { data: [...rows] }; },
    async post(path, payload) {
      calls.push(['post', path, payload]);
      if (invalidResponse) return { data: null };
      const index = rows.findIndex(row => row.source_path === payload.source_path);
      const data = { ...payload, id: index === -1 ? 2 : rows[index].id };
      if (index === -1) rows.push(data);
      else rows[index] = data;
      return { data };
    },
    async delete(path) {
      calls.push(['delete', path]);
      if (failFirstDelete) {
        failFirstDelete = false;
        throw new Error('offline');
      }
      rows.splice(rows.findIndex(row => path.endsWith(`/${row.id}`)), 1);
    },
  };
  const { state } = setupSfc(new URL('./QrRedirectsView.vue', import.meta.url), {
    vue: testVue,
    '../api': { masterApi },
    '../workspace': { useMasterWorkspace: () => ({ useDirty() {} }) },
    './analyticsFormat': { dateLabel, number },
  }, {});
  return { state, calls };
}

test('changing the spelling of a QR source keeps the normalized row', async () => {
  const { state, calls } = fixture();
  await state.load();
  state.openEditor(original);
  state.form.source_path = ' qr ';
  state.form.target_path = '/next';
  await state.save();
  assert.deepEqual(calls, [['post', ENDPOINT, { source_path: '/qr', target_path: '/next' }]]);
  assert.deepEqual(state.redirects.value, [{ id: 1, source_path: '/qr', target_path: '/next' }]);
});

test('retries removal of the old QR without overwriting the new destination again', async () => {
  const { state, calls } = fixture({ failFirstDelete: true });
  await state.load();
  state.openEditor(original);
  state.form.source_path = '/renamed';
  await state.save();
  assert.equal(state.redirects.value.length, 2);
  assert.match(state.formError.value, /nuovo percorso è stato salvato/);
  await state.save();
  assert.deepEqual(calls, [
    ['post', ENDPOINT, { source_path: '/renamed', target_path: '/club' }],
    ['delete', `${ENDPOINT}/1`],
    ['delete', `${ENDPOINT}/1`],
  ]);
  assert.deepEqual(state.redirects.value, [{ id: 2, source_path: '/renamed', target_path: '/club' }]);
});

test('retains the old source until the replacement is confirmed by the API', async () => {
  const { state, calls } = fixture({ invalidResponse: true });
  await state.load();
  state.openEditor(original);
  state.form.source_path = '/renamed';
  await state.save();
  assert.deepEqual(calls, [['post', ENDPOINT, { source_path: '/renamed', target_path: '/club' }]]);
  assert.deepEqual(state.redirects.value, [original]);
  assert.match(state.formError.value, /Impossibile salvare/);
});
