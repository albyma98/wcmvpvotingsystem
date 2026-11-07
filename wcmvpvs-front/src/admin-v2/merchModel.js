export const saleModes = {
  inherit: 'Predefinite della società',
  online: 'Solo pagamento online',
  offline: 'Solo prenotazione',
  both: 'Entrambe le modalità',
};

const paymentLabels = {
  PENDING_ONLINE: 'In attesa di Stripe',
  UNPAID_OFFLINE: 'Da pagare alla società',
  PAID: 'Pagato online',
  PAID_OFFLINE: 'Incasso registrato dalla società',
  FAILED: 'Pagamento fallito',
  EXPIRED: 'Pagamento scaduto',
  REFUNDED: 'Rimborsato',
  REFUND_PENDING: 'Rimborso in corso',
  REFUND_FAILED: 'Rimborso fallito',
};
const fulfillmentLabels = {
  PENDING: 'Da confermare',
  CONFIRMED: 'Confermato',
  PREPARING: 'In preparazione',
  READY_FOR_PICKUP: 'Pronto al ritiro',
  COMPLETED: 'Completato',
  CANCELLED: 'Annullato',
  EXPIRED: 'Scaduto',
  REVIEW_REQUIRED: 'Da verificare',
};

export const paymentLabel = status => paymentLabels[status] || status;
export const fulfillmentLabel = status => fulfillmentLabels[status] || status;
export const paid = order => ['PAID', 'PAID_OFFLINE'].includes(order.payment_status);
export const money = (cents, currency = 'EUR') => new Intl.NumberFormat('it-IT', {
  style: 'currency', currency,
}).format((cents ?? 0) / 100);
export const timestamp = value => value ? new Intl.DateTimeFormat('it-IT', {
  dateStyle: 'medium', timeStyle: 'short',
}).format(new Date(value * 1000)) : '—';
export const euroInput = cents => cents == null ? '' : (cents / 100).toFixed(2);

export function toCents(value) {
  const input = String(value).trim().replace(',', '.');
  if (!/^\d{1,7}(\.\d{1,2})?$/.test(input)) {
    throw new Error('Indica un importo in euro con al massimo due cifre decimali.');
  }
  const [whole, fraction = ''] = input.split('.');
  return Number(whole) * 100 + Number(fraction.padEnd(2, '0'));
}

export const orderFilters = [
  ['', 'Tutti'],
  ['PAID', 'Pagati online'],
  ['UNPAID_OFFLINE', 'Da pagare alla società'],
  ['TO_PREPARE', 'Da preparare'],
  ['READY_FOR_PICKUP', 'Pronti al ritiro'],
  ['COMPLETED', 'Completati'],
  ['CANCELLED', 'Annullati'],
  ['EXPIRED', 'Scaduti'],
  ['REVIEW_REQUIRED', 'Da verificare'],
];

export const merchMediaPath = url => url.replace(/^\/api(?=\/)/, '');

export function orderActions(order) {
  const fulfillment = order.fulfillment_status;
  const active = ['PENDING', 'CONFIRMED', 'PREPARING', 'READY_FOR_PICKUP'].includes(fulfillment);
  const offline = order.mode === 'offline';
  const offlineUnpaid = offline && order.payment_status === 'UNPAID_OFFLINE';
  const canFulfill = paid(order) || offlineUnpaid;
  const actions = [];
  if (offlineUnpaid && fulfillment === 'PENDING') actions.push(['confirm', 'Conferma prenotazione']);
  if (canFulfill && fulfillment === 'CONFIRMED') actions.push(['prepare', 'Metti in preparazione']);
  if (canFulfill && fulfillment === 'PREPARING' && order.fulfillment === 'pickup') {
    actions.push(['ready', 'Segna come pronto']);
  }
  if (active && offlineUnpaid) actions.push(['record_payment', 'Registra pagamento']);
  const finalStage = order.fulfillment === 'shipping' ? 'PREPARING' : 'READY_FOR_PICKUP';
  if (paid(order) && fulfillment === finalStage) {
    actions.push(['complete', 'Segna come consegnato']);
  }
  if (active && offlineUnpaid) actions.push(['cancel', 'Annulla prenotazione']);
  if (!offline && order.payment_status === 'PAID') actions.push(['refund', 'Rimborsa con Stripe']);
  return actions;
}

export function merchRequestError(error) {
  return error?.response?.data?.message || error?.message || 'Operazione non riuscita. Riprova.';
}
