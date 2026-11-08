<template>
  <section class="orders-page">
    <div v-if="!selectedId" class="section-heading">
      <h2 v-if="compact" class="sheet-orders-title">I miei ordini</h2>
      <div v-else><p class="eyebrow">SHOP DEL CLUB</p><h1>I miei ordini</h1></div>
      <button :class="compact ? 'text-button' : 'secondary'" type="button" :disabled="loading" @click="refresh">Aggiorna</button>
    </div>
    <div v-if="received" class="state-banner" role="status">
      <strong>Prenotazione ricevuta</strong>
      <span>{{ received.confirmation_text || 'Trovi qui il codice e le istruzioni per il pagamento alla società.' }}</span>
    </div>
    <div v-if="showTargetStatus && paymentReturn" class="state-banner" role="status">
      <strong>{{ paymentReturn === 'cancel' ? 'Checkout interrotto' : 'Verifica del pagamento' }}</strong>
      <span v-if="paymentReturn === 'cancel'">Puoi riprendere il pagamento se il checkout è ancora disponibile.</span>
      <span v-else-if="targetOrder && !onlinePaymentPending(targetOrder)">{{ orderPaymentLabel(targetOrder) }}.</span>
      <span v-else>La conferma apparirà appena il pagamento sarà verificato.</span>
    </div>
    <p v-if="loading && !orders.length" class="empty" role="status">Caricamento ordini…</p>
    <p v-if="error" class="error" role="alert">{{ error }}</p>
    <template v-else-if="!loading && !orders.length">
      <div v-if="compact" class="sheet-empty">
        <svg class="sheet-photo-icon" viewBox="0 0 24 24" aria-hidden="true"><path d="M5 8h14l2 13H3L5 8Zm3 2V7a4 4 0 0 1 8 0v3"/></svg>
        <strong>Nessun ordine ancora</strong>
        <p>Quando acquisti o prenoti, li trovi qui.</p>
        <button class="text-button" type="button" @click="$emit('browse')">Scopri i prodotti</button>
      </div>
      <p v-else class="empty">Qui compaiono gli ordini del tuo profilo e quelli salvati su questo dispositivo. Se hai un collegamento di recupero, aprilo per ritrovare il tuo ordine.</p>
    </template>
    <template v-if="compact && !selectedId">
      <ul v-if="orders.length" class="sheet-order-list">
        <li v-for="order in orders" :key="order.id">
          <button type="button" class="sheet-order-row" :data-order-id="order.id" @click="$emit('select', order.id)">
            <span class="sheet-order-thumb">
              <img v-if="thumbnail(order)" :src="thumbnail(order)" alt="" loading="lazy" decoding="async" />
              <svg v-else class="sheet-photo-icon" viewBox="0 0 24 24" aria-hidden="true"><path d="M5 8h14l2 13H3L5 8Zm3 2V7a4 4 0 0 1 8 0v3"/></svg>
            </span>
            <span class="sheet-order-copy">
              <strong>{{ order.items?.[0]?.name || order.code }}</strong>
              <span class="sheet-order-meta">
                <time>{{ orderDate(order.created_at, true) }}</time>
                <span v-if="order.items?.length > 1">+{{ order.items.length - 1 }} articoli</span>
              </span>
              <span class="badges"><span>{{ orderFulfillmentLabel(order) }}</span></span>
            </span>
            <span class="sheet-order-total">{{ merchMoney(order.total_cents) }}</span>
            <svg class="sheet-order-arrow" viewBox="0 0 24 24" aria-hidden="true"><path d="m9 6 6 6-6 6"/></svg>
          </button>
        </li>
      </ul>
      <details class="recovery sheet-recovery">
        <summary>Recupera un ordine</summary>
        <form class="sheet-recovery-form" @submit.prevent="recoverLink">
          <label class="field">Link privato dell’ordine
            <input v-model.trim="privateLink" required autocomplete="off" autocapitalize="none" spellcheck="false" inputmode="url" />
          </label>
          <button class="secondary" type="submit" :disabled="Boolean(recoveryRequest)">Apri ordine</button>
          <p v-if="recoveryError" class="error" role="alert">{{ recoveryError }}</p>
        </form>
      </details>
    </template>
    <button v-if="compact && selectedId && (received || paymentReturn)" class="text-button" type="button" :disabled="Boolean(resumingId)" @click="$emit('list')">Tutti gli ordini</button>
    <article v-for="order in detailOrders" :key="order.id" class="order-card">
      <div class="order-heading">
        <div><p class="eyebrow">{{ order.mode === 'offline' ? 'PRENOTAZIONE' : 'ORDINE' }}</p><h2>{{ order.code }}</h2></div>
        <time>{{ orderDate(order.created_at) }}</time>
      </div>
      <div class="badges">
        <span>{{ orderPaymentLabel(order) }}</span><span>{{ orderFulfillmentLabel(order) }}</span>
      </div>
      <ul>
        <li v-for="item in order.items" :key="merchItemKey(item)">
          <span>{{ item.quantity }} × {{ item.name }}<small v-if="item.variant_name"> · {{ item.variant_name }}</small></span>
          <span>{{ merchMoney(item.unit_cents * item.quantity) }}</span>
        </li>
      </ul>
      <div v-if="order.fulfillment === 'shipping'" class="total"><span>Spedizione</span><span>{{ merchMoney(order.shipping_cents) }}</span></div>
      <div class="total grand-total"><strong>Totale</strong><strong>{{ merchMoney(order.total_cents) }}</strong></div>
      <div class="inset order-instructions">
        <strong>{{ order.fulfillment === 'shipping' ? 'Spedizione' : 'Ritiro al club' }}</strong>
        <p v-if="order.fulfillment === 'pickup' && order.pickup_point">{{ order.pickup_point }}</p>
        <p v-if="order.delivery_instructions">{{ order.delivery_instructions }}</p>
        <p v-if="order.availability_text">{{ order.availability_text }}</p>
        <template v-if="order.mode === 'offline' && order.payment_status === 'UNPAID_OFFLINE' && !['CANCELLED', 'EXPIRED'].includes(order.fulfillment_status)">
          <p v-if="order.payment_methods">{{ order.payment_methods }}</p>
          <p v-if="order.payment_instructions">{{ order.payment_instructions }}</p>
          <p v-if="order.expires_at">Scadenza prenotazione {{ orderDate(order.expires_at) }}</p>
        </template>
      </div>
      <p v-if="order.contact_email" class="seller"><a :href="`mailto:${order.contact_email}`">Contatta il club</a></p>
      <button v-if="order.checkout_url || order.checkout_available" class="primary resume" type="button" :disabled="Boolean(resumingId)" @click="resume(order)">{{ resumingId === order.id ? 'Attendi…' : 'Riprendi pagamento' }}</button>
      <p v-if="resumeError.id === order.id" class="error" role="alert">{{ resumeError.message }}</p>
      <button v-if="access[order.id]" class="text-button" type="button" @click="copyOrder(order.id)">Copia link privato</button>
      <span v-if="copiedId === order.id" class="muted" role="status">Link copiato.</span>
      <label v-if="manualCopyId === order.id" class="field">Link privato
        <input :value="orderLink(order.id)" readonly @focus="$event.target.select()" />
      </label>
      <MerchRecovery v-if="access[order.id] && order.sms_recovery_available" :slug="slug" :order-id="order.id" :token="access[order.id]" />
    </article>
  </section>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import MerchRecovery from './MerchRecovery.vue';
import { fetchFanMerchOrders, fetchMerchOrder, merchMediaUrl } from './merchApi';
import {
  fanPortalPath, guestOrderFragment, guestOrderFragmentFor, merchItemKey, merchMoney, onlinePaymentPending,
  orderFulfillmentLabel, orderPaymentLabel, readGuestOrders, saveGuestOrder,
} from './shopModel';

const props = defineProps({
  slug: { type: String, required: true },
  search: { type: String, default: '' },
  latestOrder: { type: Object, default: null },
  compact: Boolean,
  selectedId: { type: String, default: '' },
  products: { type: Array, default: () => [] },
});
const emit = defineEmits(['select', 'browse', 'list', 'busy-change']);
const orders = ref(props.latestOrder?.order ? [props.latestOrder.order] : []);
const access = ref({});
const loading = ref(true);
const error = ref('');
const copiedId = ref('');
const manualCopyId = ref('');
const resumingId = ref('');
const resumeError = ref({});
const privateLink = ref('');
const recoveryError = ref('');
const recoveryRequest = ref(null);
const query = computed(() => new URLSearchParams(props.search));
const linkedOrderId = ref('');
const targetId = computed(() => query.value.get('order') || linkedOrderId.value);
const targetOrder = computed(() => orders.value.find(order => order.id === targetId.value));
const showTargetStatus = computed(() => !props.compact || props.selectedId === targetId.value);
const received = computed(() => showTargetStatus.value && query.value.get('reservation') === 'received' && targetOrder.value?.mode === 'offline' ? targetOrder.value : null);
const paymentReturn = computed(() => ['return', 'cancel'].includes(query.value.get('payment')) ? query.value.get('payment') : '');
const detailOrders = computed(() => props.compact ? orders.value.filter(order => order.id === props.selectedId) : orders.value);
const productPhotos = computed(() => new Map(props.products.map(product => [product.id, product.media?.[0]?.url])));
let request;
let timer;
let checks = 0;

function orderDate(timestamp, dateOnly = false) {
  if (!timestamp) return '';
  const date = /^\d+$/.test(String(timestamp)) ? new Date(Number(timestamp) * 1000) : new Date(timestamp);
  return Number.isNaN(date.getTime()) ? '' : new Intl.DateTimeFormat('it-IT', { dateStyle: 'medium', ...(dateOnly ? {} : { timeStyle: 'short' }) }).format(date);
}

function thumbnail(order) {
  const url = productPhotos.value.get(order.items?.[0]?.product_id);
  return url ? merchMediaUrl(url) : '';
}

function captureAccess() {
  const initial = props.latestOrder;
  access.value = { ...access.value, ...readGuestOrders(window, props.slug), ...(initial?.access_token && initial.order ? { [initial.order.id]: initial.access_token } : {}) };
  if (props.compact) return;
  const link = guestOrderFragment(window.location.hash);
  if (link) {
    linkedOrderId.value = link.id;
    saveGuestOrder(window, props.slug, link.id, link.token);
    access.value[link.id] = link.token;
    window.history.replaceState(window.history.state, '', window.location.pathname + window.location.search);
  }
}

function refresh() {
  checks = 0;
  void load();
}

async function load() {
  request?.abort();
  window.clearTimeout(timer);
  request = new AbortController();
  const { signal } = request;
  loading.value = true;
  error.value = '';
  const guests = Object.entries(access.value);
  const id = targetId.value;
  if (id && /^[A-Za-z0-9_-]{1,128}$/.test(id) && !Object.hasOwn(access.value, id)) guests.unshift([id, '']);
  const [profileResult, ...guestResults] = await Promise.allSettled([
    fetchFanMerchOrders(props.slug, signal),
    ...guests.map(([orderId, token]) => fetchMerchOrder(props.slug, orderId, token, signal)),
  ]);
  if (signal.aborted) return;
  const merged = new Map(orders.value.map(order => [order.id, order]));
  if (profileResult.status === 'fulfilled') {
    for (const order of profileResult.value || []) merged.set(order.id, order);
  }
  for (const result of guestResults) {
    if (result.status === 'fulfilled' && result.value) merged.set(result.value.id, result.value);
  }
  orders.value = [...merged.values()].sort((a, b) =>
    Number(b.id === id) - Number(a.id === id) || (b.created_at > a.created_at ? 1 : -1),
  );
  if (id && !targetOrder.value) error.value = 'Impossibile recuperare questo ordine. Usa il link privato originale oppure il tuo profilo tifoso.';
  else if (profileResult.status === 'rejected' && profileResult.reason?.response?.status !== 401 && !orders.value.length) error.value = 'Impossibile caricare gli ordini. Riprova.';
  loading.value = false;
  if (paymentReturn.value === 'return' && targetOrder.value && onlinePaymentPending(targetOrder.value) && checks++ < 10) {
    timer = window.setTimeout(updatePayment, 4000);
  }
}

async function updatePayment() {
  const { signal } = request;
  const id = targetId.value;
  try {
    const order = await fetchMerchOrder(props.slug, id, accessToken(id), signal);
    if (!signal.aborted && order) orders.value = orders.value.map(entry => entry.id === id ? order : entry);
  } catch { /* The manual refresh can retry a failed status check. */ }
  if (!signal.aborted && onlinePaymentPending(targetOrder.value) && checks++ < 10) timer = window.setTimeout(updatePayment, 4000);
}

function accessToken(id) {
  return Object.hasOwn(access.value, id) ? access.value[id] : '';
}

async function resume(order) {
  if (resumingId.value) return;
  resumingId.value = order.id;
  emit('busy-change', true);
  resumeError.value = {};
  try {
    const current = await fetchMerchOrder(props.slug, order.id, accessToken(order.id));
    orders.value = orders.value.map(entry => entry.id === order.id ? current : entry);
    if (current.checkout_url) {
      window.location.assign(current.checkout_url);
      return;
    }
    resumeError.value = { id: order.id, message: 'Il pagamento non può essere ripreso. Controlla lo stato qui sopra.' };
  } catch {
    resumeError.value = { id: order.id, message: 'Impossibile riprendere il pagamento. Riprova.' };
  } finally {
    resumingId.value = '';
    emit('busy-change', false);
  }
}

async function recoverLink() {
  if (recoveryRequest.value) return;
  recoveryError.value = '';
  let link;
  try {
    const url = new URL(privateLink.value);
    if (['https:', 'http:'].includes(url.protocol) && url.pathname.replace(/\/$/, '') === `${fanPortalPath(props.slug)}/orders`) {
      link = guestOrderFragment(url.hash);
    }
  } catch { /* Only complete private links can be used. */ }
  if (!link) {
    recoveryError.value = 'Inserisci il link privato di un ordine di questo club.';
    return;
  }
  const controller = new AbortController();
  recoveryRequest.value = controller;
  try {
    const order = await fetchMerchOrder(props.slug, link.id, link.token, controller.signal);
    if (controller.signal.aborted) return;
    access.value = { ...access.value, [order.id]: link.token };
    saveGuestOrder(window, props.slug, order.id, link.token);
    orders.value = [order, ...orders.value.filter(entry => entry.id !== order.id)];
    privateLink.value = '';
    emit('select', order.id);
  } catch {
    if (!controller.signal.aborted) recoveryError.value = 'Impossibile recuperare l’ordine. Controlla il link e riprova.';
  } finally {
    recoveryRequest.value = null;
  }
}

function orderLink(id) {
  return `${window.location.origin}${fanPortalPath(props.slug)}/orders${guestOrderFragmentFor(id, access.value[id])}`;
}

async function copyOrder(id) {
  copiedId.value = '';
  manualCopyId.value = '';
  try {
    await navigator.clipboard.writeText(orderLink(id));
    copiedId.value = id;
  } catch {
    manualCopyId.value = id;
  }
}

onMounted(() => {
  captureAccess();
  void load();
});
watch([() => props.search, () => props.latestOrder], ([, latest], [, previous]) => {
  if (latest?.order && latest !== previous) {
    orders.value = [latest.order, ...orders.value.filter(order => order.id !== latest.order.id)];
  }
  captureAccess();
  refresh();
});
onBeforeUnmount(() => {
  request?.abort();
  recoveryRequest.value?.abort();
  window.clearTimeout(timer);
});
</script>
