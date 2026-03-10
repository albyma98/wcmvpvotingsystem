<template>
  <section class="card bar-admin">
    <header class="bar-top">
      <h2>Sezione BAR</h2>
      <div class="bar-tabs">
        <button v-for="t in tabs" :key="t.id" class="btn" :class="activeTab===t.id?'primary':'outline'" @click="activeTab=t.id">{{ t.label }}</button>
      </div>
    </header>

    <div v-if="activeTab==='dashboard'">
      <div class="kpi-grid">
        <article class="kpi" v-for="k in kpis" :key="k.label"><h3>{{ k.label }}</h3><p>{{ k.value }}</p></article>
      </div>
      <h3>Ultimi ordini</h3>
      <ul class="item-list compact"><li v-for="o in overview.latest_orders||[]" :key="o.id">#{{o.id}} • {{ statusLabel(o.order_status) }} • € {{(o.total_cents/100).toFixed(2)}} • {{ seatingLabel(o) }}</li></ul>
    </div>

    <div v-else-if="activeTab==='live'" class="live-board-wrap">
      <header class="live-toolbar">
        <div>
          <h3>Ordini Live</h3>
          <p class="muted">Vista operativa: solo ordini aperti, con priorità visuale e azioni rapide.</p>
        </div>
        <div class="live-toolbar__actions">
          <span class="muted">Aggiornato {{ lastRefreshLabel }}</span>
          <button class="btn outline" type="button" @click="load" :disabled="isLoading">{{ isLoading ? 'Aggiornamento…' : 'Aggiorna' }}</button>
        </div>
      </header>

      <div class="live-columns" role="list" aria-label="Ordini live per stato">
        <section v-for="column in liveColumns" :key="column.id" class="live-column" role="listitem">
          <header class="live-column__header" :class="`live-column__header--${column.id}`">
            <h4>{{ column.title }}</h4>
            <span class="live-column__count">{{ column.orders.length }}</span>
          </header>

          <div v-if="!column.orders.length" class="live-column__empty">
            <p>{{ column.emptyText }}</p>
          </div>

          <article
            v-for="order in column.orders"
            :key="order.id"
            class="order-card"
            :class="[`order-card--${order.order_status}`, urgencyClass(order.elapsedMinutes)]"
            @click="openOrderDetail(order)"
          >
            <header class="order-card__header">
              <div>
                <p class="order-card__id">Ordine #{{ order.id }}</p>
                <p class="order-card__time">{{ formatOrderTime(order.created_at) }}</p>
              </div>
              <div class="order-card__badges">
                <span class="badge" :class="`badge--${paymentClass(order.payment_status)}`">{{ paymentLabel(order.payment_status) }}</span>
                <span class="badge" :class="urgencyBadgeClass(order.elapsedMinutes)">{{ order.elapsedMinutes }} min</span>
              </div>
            </header>

            <div class="order-card__meta">
              <p><strong>Totale:</strong> € {{ euro(order.total_cents) }}</p>
              <p><strong>Cliente:</strong> {{ customerLabel(order) }}</p>
              <p><strong>Posizione:</strong> {{ seatingLabel(order) }}</p>
            </div>

            <ul class="order-card__items">
              <li v-for="item in order.parsedItems" :key="`${order.id}-${item.id}`">
                <strong>x{{ item.quantity }}</strong> {{ item.name }}
              </li>
              <li v-if="!order.parsedItems.length" class="muted">Prodotti non disponibili</li>
            </ul>

            <p v-if="order.notes" class="order-card__notes">📝 {{ order.notes }}</p>

            <footer class="order-card__actions" @click.stop>
              <button
                v-if="nextStatusAction(order.order_status)"
                class="btn primary"
                type="button"
                @click="setStatus(order.id, nextStatusAction(order.order_status).status)"
              >
                {{ nextStatusAction(order.order_status).label }}
              </button>
              <button
                v-if="canCancel(order.order_status)"
                class="btn danger"
                type="button"
                @click="setStatus(order.id, 'annullato')"
              >
                Annulla
              </button>
              <button class="btn outline" type="button" @click="openOrderDetail(order)">Dettaglio</button>
            </footer>
          </article>
        </section>
      </div>
    </div>

    <div v-else class="muted">{{ tabText }}</div>

    <div v-if="selectedOrder" class="order-modal" role="dialog" aria-modal="true" aria-label="Dettaglio ordine">
      <div class="order-modal__backdrop" @click="closeOrderDetail"></div>
      <div class="order-modal__panel">
        <header class="order-modal__header">
          <div>
            <p class="muted">Dettaglio ordine</p>
            <h3>#{{ selectedOrder.id }} • {{ statusLabel(selectedOrder.order_status) }}</h3>
          </div>
          <button class="btn outline" type="button" @click="closeOrderDetail">Chiudi</button>
        </header>

        <div class="order-modal__grid">
          <p><strong>Creato alle:</strong> {{ formatOrderTime(selectedOrder.created_at, true) }}</p>
          <p><strong>Aggiornato alle:</strong> {{ formatOrderTime(selectedOrder.updated_at, true) }}</p>
          <p><strong>Pagamento:</strong> {{ paymentLabel(selectedOrder.payment_status) }}</p>
          <p><strong>Totale:</strong> € {{ euro(selectedOrder.total_cents) }}</p>
          <p><strong>Cliente:</strong> {{ customerLabel(selectedOrder) }}</p>
          <p><strong>Posizione:</strong> {{ seatingLabel(selectedOrder) }}</p>
        </div>

        <section>
          <h4>Prodotti</h4>
          <ul class="order-modal__items">
            <li v-for="item in selectedOrder.parsedItems" :key="`detail-${selectedOrder.id}-${item.id}`">
              <span>{{ item.name }}</span>
              <strong>x{{ item.quantity }}</strong>
            </li>
          </ul>
        </section>

        <section>
          <h4>Note</h4>
          <p class="order-modal__notes">{{ selectedOrder.notes || 'Nessuna nota' }}</p>
        </section>

        <section>
          <h4>Cronologia</h4>
          <ul class="order-modal__timeline">
            <li>
              <span>Creato</span>
              <strong>{{ formatOrderTime(selectedOrder.created_at, true) }}</strong>
            </li>
            <li>
              <span>Ultimo aggiornamento</span>
              <strong>{{ formatOrderTime(selectedOrder.updated_at, true) }}</strong>
            </li>
            <li>
              <span>Stato corrente</span>
              <strong>{{ statusLabel(selectedOrder.order_status) }}</strong>
            </li>
          </ul>
        </section>
      </div>
    </div>
  </section>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { apiClient } from '../api';

const props = defineProps({ authHeaders: { type: Object, required: true }, isSuperAdmin: { type: Boolean, default: false } });
const activeTab = ref('dashboard');
const overview = ref({});
const orders = ref([]);
const isLoading = ref(false);
const selectedOrder = ref(null);
const lastRefreshAt = ref(null);
const RECENT_CANCELLED_MINUTES = 20;
const refreshEveryMs = 15000;
let refreshTimer = null;

const tabs = [
  { id: 'dashboard', label: 'Dashboard' },
  { id: 'live', label: 'Ordini live' },
  { id: 'history', label: 'Storico ordini' },
  { id: 'menu', label: 'Menu' },
  { id: 'quick', label: 'Disponibilità rapida' },
  { id: 'stats', label: 'Statistiche' },
  { id: 'clients', label: 'Clienti' },
  { id: 'cash', label: 'Cassa' },
  { id: 'settings', label: 'Impostazioni' },
];

const tabText = computed(() => 'Sezione operativa pronta. Integra qui i flussi specifici del bar.');
const kpis = computed(() => {
  const o = overview.value || {};
  return [
    { label: 'Ordini ricevuti', value: o.orders_received || 0 },
    { label: 'In attesa', value: o.orders_pending || 0 },
    { label: 'In preparazione', value: o.orders_in_preparation || 0 },
    { label: 'Pronti', value: o.orders_ready || 0 },
    { label: 'Completati', value: o.orders_completed || 0 },
    { label: 'Incasso', value: `€ ${((o.revenue_cents||0)/100).toFixed(2)}` },
    { label: 'Ticket medio', value: `€ ${((o.avg_ticket_cents||0)/100).toFixed(2)}` },
  ];
});

const parsedOrders = computed(() => {
  return orders.value.map((order) => {
    const status = normalizeStatus(order.order_status);
    const createdAt = parseDate(order.created_at);
    const elapsedMinutes = Math.max(0, Math.round((Date.now() - createdAt.getTime()) / 60000));
    return {
      ...order,
      order_status: status,
      elapsedMinutes,
      parsedItems: parseItems(order),
    };
  });
});

const liveColumns = computed(() => {
  const active = parsedOrders.value.filter((order) => {
    if (['new', 'in_preparazione', 'pronto'].includes(order.order_status)) {
      return true;
    }
    return order.order_status === 'annullato' && order.elapsedMinutes <= RECENT_CANCELLED_MINUTES;
  });

  const byStatus = {
    new: active.filter((o) => o.order_status === 'new'),
    in_preparazione: active.filter((o) => o.order_status === 'in_preparazione'),
    pronto: active.filter((o) => o.order_status === 'pronto'),
    annullato: active.filter((o) => o.order_status === 'annullato'),
  };

  return [
    { id: 'new', title: 'Nuovi', orders: byStatus.new, emptyText: 'Nessun nuovo ordine' },
    { id: 'in_preparazione', title: 'In preparazione', orders: byStatus.in_preparazione, emptyText: 'Nessun ordine in preparazione' },
    { id: 'pronto', title: 'Pronti', orders: byStatus.pronto, emptyText: 'Nessun ordine pronto al ritiro' },
    { id: 'annullato', title: 'Annullati recenti', orders: byStatus.annullato, emptyText: 'Nessun annullamento recente' },
  ];
});

const lastRefreshLabel = computed(() => {
  if (!lastRefreshAt.value) {
    return 'mai';
  }
  return new Intl.DateTimeFormat('it-IT', { hour: '2-digit', minute: '2-digit', second: '2-digit' }).format(lastRefreshAt.value);
});

async function load() {
  isLoading.value = true;
  try {
    const [{ data: ov }, { data: os }] = await Promise.all([
      apiClient.get('/admin/bar/overview', props.authHeaders),
      apiClient.get('/admin/bar/orders', props.authHeaders),
    ]);
    overview.value = ov || {};
    orders.value = Array.isArray(os) ? os : [];
    lastRefreshAt.value = new Date();
    if (selectedOrder.value) {
      selectedOrder.value = parsedOrders.value.find((order) => order.id === selectedOrder.value.id) || null;
    }
  } finally {
    isLoading.value = false;
  }
}

async function setStatus(orderId, status) {
  await apiClient.put(`/admin/bar/orders/${orderId}/status`, { status }, props.authHeaders);
  await load();
}

function parseItems(order) {
  const products = safeJson(order.products || order.products_json || '[]');
  const quantities = safeJson(order.quantities || order.quantities_json || '[]');

  const quantityMap = new Map();
  quantities.forEach((row) => {
    if (row && row.id) {
      quantityMap.set(String(row.id), Number(row.quantity || 0));
    }
  });

  return products.map((product) => {
    const id = String(product.id || product.name || Math.random());
    return {
      id,
      name: product.name || product.id || 'Prodotto',
      quantity: Math.max(1, quantityMap.get(id) || 1),
    };
  });
}

function safeJson(raw) {
  try {
    const parsed = JSON.parse(String(raw || '[]'));
    return Array.isArray(parsed) ? parsed : [];
  } catch (error) {
    return [];
  }
}

function parseDate(raw) {
  const parsed = new Date(raw || Date.now());
  if (Number.isNaN(parsed.getTime())) {
    return new Date();
  }
  return parsed;
}

function normalizeStatus(status) {
  const s = String(status || '').toLowerCase().trim();
  if (s === 'pending' || s === 'paid') {
    return 'new';
  }
  return s || 'new';
}

function statusLabel(status) {
  const labels = {
    new: 'Nuovo',
    pending: 'Nuovo',
    in_preparazione: 'In preparazione',
    pronto: 'Pronto',
    completato: 'Completato',
    annullato: 'Annullato',
  };
  return labels[status] || status;
}

function paymentLabel(status) {
  const labels = {
    paid: 'Pagato',
    pending: 'Pagamento in attesa',
    cancelled: 'Pagamento annullato',
    expired: 'Pagamento scaduto',
  };
  return labels[String(status || '').toLowerCase()] || 'Pagamento n/d';
}

function paymentClass(status) {
  const s = String(status || '').toLowerCase();
  if (s === 'paid') {
    return 'ok';
  }
  if (s === 'pending') {
    return 'wait';
  }
  return 'alert';
}

function euro(value) {
  return (Number(value || 0) / 100).toFixed(2);
}

function formatOrderTime(raw, withDate = false) {
  const date = parseDate(raw);
  return new Intl.DateTimeFormat('it-IT', {
    day: withDate ? '2-digit' : undefined,
    month: withDate ? '2-digit' : undefined,
    hour: '2-digit',
    minute: '2-digit',
  }).format(date);
}

function customerLabel(order) {
  if (order.customer_name) {
    return order.customer_name;
  }
  return `Ticket ${String(order.stripe_reference || order.id).slice(-8)}`;
}

function seatingLabel(order) {
  const parts = [order.sector, order.row, order.seat].filter(Boolean);
  return parts.length ? parts.join(' • ') : 'Non disponibile';
}

function urgencyClass(minutes) {
  if (minutes >= 18) {
    return 'order-card--urgent';
  }
  if (minutes >= 10) {
    return 'order-card--attention';
  }
  return 'order-card--fresh';
}

function urgencyBadgeClass(minutes) {
  if (minutes >= 18) {
    return 'badge--urgent';
  }
  if (minutes >= 10) {
    return 'badge--attention';
  }
  return 'badge--ok';
}

function nextStatusAction(status) {
  const actions = {
    new: { status: 'in_preparazione', label: 'In preparazione' },
    in_preparazione: { status: 'pronto', label: 'Segna pronto' },
    pronto: { status: 'completato', label: 'Completa' },
  };
  return actions[status] || null;
}

function canCancel(status) {
  return ['new', 'in_preparazione'].includes(status);
}

function openOrderDetail(order) {
  selectedOrder.value = order;
}

function closeOrderDetail() {
  selectedOrder.value = null;
}

onMounted(async () => {
  await load();
  refreshTimer = window.setInterval(load, refreshEveryMs);
});

onBeforeUnmount(() => {
  if (refreshTimer) {
    window.clearInterval(refreshTimer);
  }
});
</script>

<style scoped>
.bar-tabs{display:flex;gap:.5rem;flex-wrap:wrap}.kpi-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(160px,1fr));gap:.75rem}.kpi{padding:1rem;border:1px solid #e5e7eb;border-radius:12px}.status-actions{display:flex;gap:.5rem;flex-wrap:wrap;margin-top:.5rem}
.live-board-wrap { display: grid; gap: 1rem; }
.live-toolbar { display: flex; justify-content: space-between; align-items: center; gap: 1rem; flex-wrap: wrap; }
.live-toolbar__actions { display: flex; align-items: center; gap: .65rem; }
.live-columns { display: grid; grid-template-columns: repeat(4, minmax(260px, 1fr)); gap: 1rem; overflow-x: auto; padding-bottom: .35rem; }
.live-column { background: #f8fafc; border: 1px solid #dbe2ea; border-radius: 14px; min-height: 320px; padding: .75rem; display: grid; gap: .75rem; align-content: start; }
.live-column__header { display: flex; justify-content: space-between; align-items: center; padding: .6rem .75rem; border-radius: 10px; color: #0f172a; }
.live-column__header--new { background: #dbeafe; }
.live-column__header--in_preparazione { background: #fef3c7; }
.live-column__header--pronto { background: #dcfce7; }
.live-column__header--annullato { background: #f1f5f9; }
.live-column__count { display: inline-flex; align-items: center; justify-content: center; min-width: 2rem; height: 2rem; background: #0f172a; color: white; border-radius: 999px; font-weight: 700; }
.live-column__empty { border: 1px dashed #c9d6e4; border-radius: 10px; padding: 1rem; color: #64748b; text-align: center; }
.order-card { border: 1px solid #d5deea; border-left-width: 6px; border-radius: 12px; background: white; padding: .8rem; display: grid; gap: .6rem; cursor: pointer; }
.order-card__header { display: flex; justify-content: space-between; gap: .5rem; }
.order-card__id { margin: 0; font-size: 1.15rem; font-weight: 800; color: #0f172a; }
.order-card__time { margin: 0; color: #64748b; font-size: .92rem; }
.order-card__badges { display: flex; gap: .35rem; flex-wrap: wrap; justify-content: flex-end; }
.order-card__meta p { margin: 0 0 .2rem; color: #0f172a; }
.order-card__items { margin: 0; padding-left: 1rem; display: grid; gap: .18rem; }
.order-card__notes { margin: 0; padding: .55rem; border-radius: 8px; background: #fffbeb; color: #854d0e; font-size: .92rem; }
.order-card__actions { display: flex; gap: .4rem; flex-wrap: wrap; }
.order-card--new { border-left-color: #3b82f6; }
.order-card--in_preparazione { border-left-color: #f59e0b; }
.order-card--pronto { border-left-color: #22c55e; }
.order-card--annullato { border-left-color: #94a3b8; }
.order-card--attention { box-shadow: 0 0 0 2px rgba(245, 158, 11, 0.2); }
.order-card--urgent { box-shadow: 0 0 0 2px rgba(239, 68, 68, 0.25); }
.badge { display: inline-flex; align-items: center; justify-content: center; border-radius: 999px; padding: .22rem .5rem; font-size: .78rem; font-weight: 700; }
.badge--ok { background: #dcfce7; color: #166534; }
.badge--wait { background: #fef3c7; color: #92400e; }
.badge--alert { background: #fee2e2; color: #b91c1c; }
.badge--attention { background: #ffedd5; color: #9a3412; }
.badge--urgent { background: #fee2e2; color: #b91c1c; }
.order-modal { position: fixed; inset: 0; z-index: 60; display: grid; place-items: center; }
.order-modal__backdrop { position: absolute; inset: 0; background: rgba(15, 23, 42, 0.55); }
.order-modal__panel { position: relative; width: min(860px, 96vw); max-height: 92vh; overflow: auto; background: white; border-radius: 16px; padding: 1rem; display: grid; gap: 1rem; }
.order-modal__header { display: flex; justify-content: space-between; gap: 1rem; align-items: flex-start; }
.order-modal__grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px,1fr)); gap: .65rem; }
.order-modal__items { margin: 0; padding: 0; list-style: none; display: grid; gap: .45rem; }
.order-modal__items li { display: flex; justify-content: space-between; padding: .5rem .65rem; border: 1px solid #dbe2ea; border-radius: 8px; }
.order-modal__notes { margin: 0; padding: .7rem; border-radius: 8px; background: #f8fafc; border: 1px solid #dbe2ea; }
.order-modal__timeline { margin: 0; padding: 0; list-style: none; display: grid; gap: .4rem; }
.order-modal__timeline li { display: flex; justify-content: space-between; border-bottom: 1px dashed #dbe2ea; padding-bottom: .35rem; }
@media (max-width: 1100px) {
  .live-columns { grid-template-columns: repeat(4, minmax(290px, 1fr)); }
}
</style>
