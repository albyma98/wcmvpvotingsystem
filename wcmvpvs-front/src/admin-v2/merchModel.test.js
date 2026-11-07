import assert from 'node:assert/strict';
import test from 'node:test';
import { fulfillmentLabel, merchMediaPath, orderActions, orderFilters, paymentLabel, toCents } from './merchModel.js';

test('merch prices are parsed in integer cents and reject malformed amounts', () => {
  assert.equal(toCents('29,90'), 2990);
  assert.equal(toCents('0.09'), 9);
  assert.equal(toCents(' 11.2 '), 1120);
  for (const value of ['', '-1', '0.001', '1e3', '1.000,00']) assert.throws(() => toCents(value));
});

test('order filters use the server contract and separate payment from fulfillment', () => {
  const filters = new Map(orderFilters);
  assert.equal(filters.get('PAID'), 'Pagati online');
  assert.equal(filters.get('UNPAID_OFFLINE'), 'Da pagare alla società');
  assert.equal(filters.get('TO_PREPARE'), 'Da preparare');
  assert.equal(filters.get('READY_FOR_PICKUP'), 'Pronti al ritiro');
  assert.notEqual(paymentLabel('PAID_OFFLINE'), fulfillmentLabel('COMPLETED'));
});

test('uploaded media paths resolve against the configured API prefix only once', () => {
  assert.equal(merchMediaPath('/api/v1/clubs/test/merch/media/id'), '/v1/clubs/test/merch/media/id');
  assert.equal(merchMediaPath('https://test.example/api/images/id'), 'https://test.example/api/images/id');
});

test('offline registration and Stripe refunds appear only for the matching payment mode', () => {
  const actions = order => orderActions(order).map(([action]) => action);
  const reservation = { mode: 'offline', payment_status: 'UNPAID_OFFLINE', fulfillment_status: 'PENDING', fulfillment: 'pickup' };
  const settled = { mode: 'online', payment_status: 'PAID', fulfillment_status: 'READY_FOR_PICKUP', fulfillment: 'pickup' };
  assert.deepEqual(actions(reservation), ['confirm', 'record_payment', 'cancel']);
  assert.deepEqual(actions(settled), ['complete', 'refund']);
  assert.deepEqual(actions({ ...settled, payment_status: 'PENDING_ONLINE', fulfillment_status: 'PENDING' }), []);
});
