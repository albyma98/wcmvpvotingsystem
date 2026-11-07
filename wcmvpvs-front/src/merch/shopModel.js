const isKey = value => typeof value === 'string' && /^[A-Za-z0-9_-]{1,128}$/.test(value);

export function readFanMerchRoute(path) {
  const parts = path.split('/').filter(Boolean);
  if (parts.length < 3 || parts[1] !== 'new' || !isKey(parts[0])) return null;
  if (parts[2] === 'orders' && parts.length === 3) return { page: 'orders', productId: '' };
  if (parts[2] === 'shop' && (parts.length === 3 || (parts.length === 4 && isKey(parts[3])))) {
    return { page: 'shop', productId: parts[3] || '' };
  }
  return null;
}

export const fanPortalPath = slug => `/${encodeURIComponent(slug)}/new`;
export const merchItemKey = item => `${item.product_id}:${item.variant_id || ''}`;
export const merchVariantLabel = variant => [variant?.size, variant?.color].filter(Boolean).join(' · ');

const euros = new Intl.NumberFormat('it-IT', { style: 'currency', currency: 'EUR' });
export const merchMoney = cents => euros.format(Number(cents) / 100);

export function merchUnitPrice(product, variant) {
  return variant?.price_cents ?? product.promo_price_cents ?? product.price_cents;
}

export function merchMaxQuantity(product, variant) {
  if (!product || (product.variants?.length && !variant)) return 0;
  const stock = variant ?? product;
  return stock.on_demand ? product.max_quantity : Math.max(0, Math.min(product.max_quantity, stock.available ?? stock.stock));
}

export function merchOptions(products, settings, lines) {
  const selections = lines.map(line => products.find(product => product.id === line.product_id));
  return {
    online: Boolean(selections.length && selections.every(product => product?.online_enabled)),
    offline: Boolean(selections.length && selections.every(product => product?.offline_enabled)),
    pickup: Boolean(settings?.pickup_enabled),
    shipping: Boolean(settings?.shipping_enabled && selections.length && selections.every(product => product?.shipping_allowed)),
  };
}

export function selectedMerchProducts(products, filters) {
  const normalize = text => String(text ?? '').normalize('NFD').replace(/\p{Diacritic}/gu, '').toLowerCase();
  const term = normalize(filters.search).trim();
  return products.filter(product =>
    (!filters.category || product.category === filters.category) &&
    (!filters.inStock || (product.variants?.length
      ? product.variants.some(variant => merchMaxQuantity(product, variant) > 0)
      : merchMaxQuantity(product) > 0)) &&
    (!term || normalize([product.name, product.category, product.short_description].join(' ')).includes(term)),
  );
}

export function guestOrderFragment(hash) {
  const params = new URLSearchParams(hash.replace(/^#/, ''));
  const id = params.get('order');
  const token = params.get('access');
  return isKey(id) && isKey(token) && token.length >= 24 ? { id, token } : null;
}

export const guestOrderFragmentFor = (id, token) => `#${new URLSearchParams({ order: id, access: token })}`;

const guestOrdersKey = slug => `arena:merch-orders:${slug}`;

export function readGuestOrders(browser, slug) {
  const orders = Object.create(null);
  for (const store of ['sessionStorage', 'localStorage']) {
    try {
      const saved = JSON.parse(browser[store].getItem(guestOrdersKey(slug)) || '{}');
      for (const [id, token] of Object.entries(saved)) {
        if (isKey(id) && isKey(token)) orders[id] = token;
      }
    } catch { /* Browser storage can be unavailable. */ }
  }
  return orders;
}

export function saveGuestOrder(browser, slug, id, token) {
  if (!isKey(id) || !isKey(token)) return false;
  const previous = Object.entries(readGuestOrders(browser, slug)).filter(([otherId]) => otherId !== id);
  const entries = [[id, token], ...previous.slice(0, 29)];
  for (const store of ['localStorage', 'sessionStorage']) {
    try {
      browser[store].setItem(guestOrdersKey(slug), JSON.stringify(Object.fromEntries(entries)));
      return true;
    } catch { /* Use session storage if persistent storage is unavailable. */ }
  }
  return false;
}

export function orderPaymentLabel(order) {
  if (order.mode === 'offline' && order.payment_status === 'UNPAID_OFFLINE' && order.fulfillment_status === 'CANCELLED') {
    return 'Nessun pagamento dovuto';
  }
  return ({
    PAID: order.mode === 'offline' ? 'Pagamento registrato dal club' : 'Pagato online',
    PAID_ONLINE: 'Pagato online',
    PAID_OFFLINE: 'Pagamento registrato dal club',
    UNPAID_OFFLINE: 'Pagamento da effettuare al club',
    PENDING: 'Pagamento online in verifica',
    PENDING_ONLINE: 'Pagamento online in verifica',
    PROCESSING: 'Pagamento in elaborazione',
    FAILED: 'Pagamento non riuscito',
    EXPIRED: order.mode === 'offline' ? 'Nessun pagamento dovuto' : 'Pagamento scaduto',
    REFUNDED: 'Rimborsato',
    PAYMENT_REVIEW: 'Pagamento da verificare con il club',
    PAID_REVIEW: 'Pagamento ricevuto, verifica del club in corso',
  })[order.payment_status] || 'Pagamento in verifica';
}

export function orderFulfillmentLabel(order) {
  return ({
    PENDING: order.mode === 'offline' ? 'Prenotazione ricevuta' : 'In attesa',
    CONFIRMED: 'Confermato',
    PREPARING: 'In preparazione',
    READY_FOR_PICKUP: 'Pronto al ritiro',
    SHIPPED: 'Spedito',
    COMPLETED: 'Completato',
    CANCELLED: 'Annullato',
    EXPIRED: 'Scaduto',
    REVIEW: 'Verifica del club in corso',
    REVIEW_REQUIRED: 'Verifica del club in corso',
  })[order.fulfillment_status] || 'In attesa';
}

export const onlinePaymentPending = order => !order || ['PENDING', 'PENDING_ONLINE', 'PROCESSING'].includes(order.payment_status);

export function stripeCheckoutUrl(value) {
  if (!value) return '';
  try {
    const url = new URL(value);
    return url.protocol === 'https:' && !url.username && !url.password ? url.href : '';
  } catch {
    return '';
  }
}
