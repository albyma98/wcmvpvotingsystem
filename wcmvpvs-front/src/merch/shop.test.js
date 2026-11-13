import assert from 'node:assert/strict';
import test from 'node:test';
import { nextTick } from 'vue';
import { setupSfc, testVue } from '../../test/sfcSetup.js';
import * as shop from './shopModel.js';
import { createSheetNavigation } from './sheetNavigation.js';

const maglia = {
  id: 'maglia', name: 'Maglia ufficiale', category: 'Abbigliamento', short_description: 'Serie città',
  price_cents: 4500, promo_price_cents: 3900, max_quantity: 2, online_enabled: true, offline_enabled: true,
  shipping_allowed: true, variants: [{ id: 'm', size: 'M', color: 'Nero', price_cents: null, stock: 3, available: 1 }],
};
const sciarpa = {
  id: 'sciarpa', name: 'Sciarpa', category: 'Accessori', price_cents: 1800, stock: 0, max_quantity: 5,
  online_enabled: false, offline_enabled: true, shipping_allowed: false,
};

test('the fan portal exposes only valid shop, product and personal-order routes', () => {
  assert.deepEqual(shop.readFanMerchRoute('/club-1/new/shop/sku-a/'), { page: 'shop', productId: 'sku-a' });
  assert.deepEqual(shop.readFanMerchRoute('/club-1/new/orders'), { page: 'orders', productId: '' });
  for (const route of ['/club-1/new', '/club-1/admin/new/merch', '/club-1/new/orders/extra', '/club-1/new/shop/%FF']) {
    assert.equal(shop.readFanMerchRoute(route), null);
  }
});

test('merch visibility is measured when the page obtains its match grant', async () => {
  const calls = [];
  const { props, scope } = setupSfc(new URL('./FanMerchLink.vue', import.meta.url), {
    vue: testVue,
    './merchApi': { fetchMerchVisibility: async (slug, signal, matchId) => { calls.push({ slug, signal, matchId }); return true; } },
    './shopModel': shop,
  }, { slug: 'home', eventId: 42, matchdayReady: false, sheet: true }, { window: {} });
  try {
    props.matchdayReady = true;
    await nextTick();
    assert.deepEqual(calls.map(({ slug, matchId }) => [slug, matchId]), [['home', 0], ['home', 42]]);
    assert.equal(calls[0].signal.aborted, true);
    assert.equal(calls[1].signal.aborted, false);
  } finally {
    scope.stop();
  }
});

test('filters, held stock, variant overrides and combined cart modes follow public catalog data', () => {
  assert.deepEqual(shop.selectedMerchProducts([maglia, sciarpa], { search: 'citta', inStock: true }), [maglia]);
  assert.deepEqual(shop.selectedMerchProducts([maglia, sciarpa], { category: 'Accessori', inStock: true }), []);
  assert.equal(shop.merchMaxQuantity(maglia), 0);
  assert.equal(shop.merchMaxQuantity(maglia, maglia.variants[0]), 1);
  assert.equal(shop.merchMaxQuantity({ ...sciarpa, on_demand: true }), 5);
  assert.equal(shop.merchUnitPrice(maglia, maglia.variants[0]), 3900);
  assert.equal(shop.merchUnitPrice(maglia, { price_cents: 4200 }), 4200);
  assert.deepEqual(shop.merchOptions([maglia, sciarpa], { pickup_enabled: true, shipping_enabled: true }, [
    { product_id: maglia.id }, { product_id: sciarpa.id },
  ]), { online: false, offline: true, pickup: true, shipping: false });
});

test('guest access uses private fragments and stays scoped to the club, even on storage fallback', () => {
  const token = 'a'.repeat(64);
  const tab = new Map();
  const browser = {
    localStorage: { getItem() { throw Error('blocked'); }, setItem() { throw Error('blocked'); } },
    sessionStorage: { getItem: key => tab.get(key), setItem: (key, value) => tab.set(key, value) },
  };
  assert.equal(shop.saveGuestOrder(browser, 'home', 'order-1', token), true);
  assert.equal(shop.readGuestOrders(browser, 'home')['order-1'], token);
  assert.equal(shop.readGuestOrders(browser, 'other')['order-1'], undefined);
  assert.deepEqual(shop.guestOrderFragment(shop.guestOrderFragmentFor('order-1', token)), { id: 'order-1', token });
  assert.equal(shop.guestOrderFragment('#order=order-1&access=123'), null);
});

test('receipts distinguish unpaid, confirmed and late online payments', () => {
  assert.equal(shop.orderPaymentLabel({ mode: 'offline', payment_status: 'UNPAID_OFFLINE' }), 'Pagamento da effettuare al club');
  assert.equal(shop.orderPaymentLabel({ mode: 'offline', payment_status: 'PAID_OFFLINE' }), 'Pagamento registrato dal club');
  assert.equal(shop.orderFulfillmentLabel({ payment_status: 'PAID', fulfillment_status: 'REVIEW_REQUIRED' }), 'Verifica del club in corso');
  assert.equal(shop.onlinePaymentPending({ payment_status: 'PENDING_ONLINE' }), true);
  assert.equal(shop.onlinePaymentPending({ payment_status: 'PAID' }), false);
});

test('sheet history keeps the homepage URL and unwinds nested views on close', () => {
  const { browser, traverse } = sheetHistoryFixture();
  let view;
  const navigation = createSheetNavigation(browser, next => { view = next; }, () => false);
  navigation.show({ page: 'shop' });
  navigation.show({ page: 'orders' }, true);
  navigation.show({ page: 'orders', orderId: 'order-1' });
  assert.equal(view.orderId, 'order-1');
  assert.equal(browser.history.state.eventId, 42);
  assert.equal(browser.history.state.orderId, undefined);
  navigation.back();
  traverse();
  assert.deepEqual(view, { page: 'orders' });
  navigation.show({ page: 'orders', orderId: 'order-2' });
  navigation.back(true);
  traverse();
  assert.equal(view, null);
  assert.deepEqual(browser.history.state, { eventId: 42 });
  navigation.dispose();
});

test('Back during checkout restores its entry before showing an order confirmation', () => {
  const { browser, traverse } = sheetHistoryFixture();
  let view;
  let busy = false;
  const navigation = createSheetNavigation(browser, next => { view = next; }, () => busy);
  navigation.show({ page: 'shop' });
  navigation.show({ page: 'shop', productId: 'maglia' });
  navigation.show({ page: 'shop', productId: 'maglia', mode: 'offline' });
  busy = true;
  navigation.back(true);
  assert.equal(traverse(), false);
  browser.history.go(-1);
  traverse();
  assert.equal(view.mode, 'offline');
  navigation.show({ page: 'orders', orderId: 'order-1' }, true);
  busy = false;
  traverse();
  assert.equal(view.orderId, 'order-1');
  navigation.back();
  traverse();
  assert.deepEqual(view, { page: 'shop', productId: 'maglia' });
  navigation.dispose();
});

function sheetHistoryFixture() {
  const browser = new EventTarget();
  const states = [{ eventId: 42 }];
  const queue = [];
  let index = 0;
  browser.history = {
    get state() { return states[index]; },
    pushState(state, title, url) {
      assert.equal(url, undefined);
      states.splice(++index, states.length, state);
    },
    replaceState(state, title, url) {
      assert.equal(url, undefined);
      states[index] = state;
    },
    go(delta) { queue.push(delta); },
  };
  return {
    browser,
    traverse() {
      if (!queue.length) return false;
      index += queue.shift();
      browser.dispatchEvent(Object.assign(new Event('popstate'), { state: states[index] }));
      return true;
    },
  };
}

function checkoutFixture(t, { mode = 'offline', api, profile = { ok: false } }) {
  const mounted = [];
  const fixture = setupSfc(new URL('./MerchCheckout.vue', import.meta.url), {
    vue: { ...testVue, onMounted: callback => mounted.push(callback) },
    '../api': { fetchFanProfile: async () => profile },
    './merchApi': { merchRequestError: (error, fallback) => error?.response?.data?.message || fallback, ...api },
    './shopModel': shop,
  }, {
    slug: 'club', mode, lines: [{ product_id: 'maglia', variant_id: 'm', quantity: 1 }],
    settings: { contact_requirement: 'either' }, deliveryOptions: { pickup: true, shipping: true },
  });
  t.after(() => fixture.scope.stop());
  return { ...fixture, mount: async () => { for (const callback of mounted) await callback(); await nextTick(); } };
}

test('offline uses a verified profile phone and retries exactly the submitted snapshot after a lost response', async t => {
  const requests = [];
  const fixture = checkoutFixture(t, {
    api: {
      fetchMerchQuote: async () => ({ total_cents: 3900, items: [] }),
      createMerchOrder: async (slug, payload) => {
        requests.push(JSON.parse(JSON.stringify(payload)));
        if (requests.length === 1) throw Error('network disconnected');
        return { order: { id: 'abc', mode: 'offline', payment_status: 'UNPAID_OFFLINE' } };
      },
    },
    profile: { ok: true, data: { registered: true, user: { phone: '+39012345678', phone_verified_at: 'verified' } } },
  });
  const { state } = fixture;
  await fixture.mount();
  assert.equal(state.contactField.value, 'phone');
  assert.equal(state.contact.phone, '+39012345678');
  state.contact.name = 'Tifoso Test';
  await state.submit();
  assert.equal(state.uncertain.value, true);
  state.contact.name = 'Changed after submission';
  await state.submit();
  assert.deepEqual(requests[0], requests[1]);
  assert.equal(requests[0].expected_total_cents, 3900);
  assert.equal(requests[0].phone, '+39012345678');
  assert.equal(requests[0].address, undefined);
  assert.equal(fixture.events.find(event => event[0] === 'created')[1].order.payment_status, 'UNPAID_OFFLINE');
});

test('online pickup sends a minimal guest request, requires reconfirmation after repricing and returns the hosted URL', async t => {
  const requests = [];
  let quotes = 0;
  const fixture = checkoutFixture(t, {
    mode: 'online',
    api: {
      fetchMerchQuote: async () => ({ total_cents: ++quotes === 1 ? 3900 : 4000, items: [] }),
      createMerchOrder: async (slug, payload) => {
        requests.push(JSON.parse(JSON.stringify(payload)));
        if (requests.length === 1) throw { response: { status: 409, data: { message: 'Il totale è cambiato' } } };
        return { order: { id: 'abc', mode: 'online', payment_status: 'PENDING_ONLINE' }, checkout_url: 'https://checkout.stripe.com/test' };
      },
    },
  });
  await fixture.mount();
  await fixture.state.submit();
  assert.equal(requests.length, 1);
  assert.equal(fixture.state.quote.value.total_cents, 4000);
  assert.equal(fixture.state.error.value, 'Il totale è cambiato');
  await fixture.state.submit();
  assert.equal(requests[0].email, undefined);
  assert.equal(requests[0].name, undefined);
  assert.equal(requests[0].fulfillment, 'pickup');
  assert.notEqual(requests[0].request_id, requests[1].request_id);
  assert.equal(requests[1].expected_total_cents, 4000);
  assert.equal(fixture.events.find(event => event[0] === 'created')[1].checkout_url, 'https://checkout.stripe.com/test');
});
