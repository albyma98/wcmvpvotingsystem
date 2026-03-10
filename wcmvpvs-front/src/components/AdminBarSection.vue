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
      <ul class="item-list compact"><li v-for="o in overview.latest_orders||[]" :key="o.id">#{{o.id}} • {{ o.order_status }} • € {{(o.total_cents/100).toFixed(2)}} • {{ o.sector }}/{{o.row}}/{{o.seat}}</li></ul>
    </div>

    <div v-else-if="activeTab==='live'">
      <ul class="item-list">
        <li v-for="o in orders" :key="o.id" class="item">
          <strong>#{{ o.id }}</strong> - {{ o.order_status }} - € {{ (o.total_cents/100).toFixed(2) }}
          <div class="status-actions">
            <button v-for="s in statuses" :key="s" class="btn outline" @click="setStatus(o.id,s)">{{ s }}</button>
          </div>
        </li>
      </ul>
    </div>

    <div v-else class="muted">{{ tabText }}</div>
  </section>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue';
import { apiClient } from '../api';

const props = defineProps({ authHeaders: { type: Object, required: true }, isSuperAdmin: { type: Boolean, default: false } });
const activeTab = ref('dashboard');
const overview = ref({});
const orders = ref([]);
const statuses = ['new', 'in_preparazione', 'pronto', 'completato', 'annullato'];
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

async function load() {
  const [{ data: ov }, { data: os }] = await Promise.all([
    apiClient.get('/admin/bar/overview', props.authHeaders),
    apiClient.get('/admin/bar/orders', props.authHeaders),
  ]);
  overview.value = ov || {};
  orders.value = Array.isArray(os) ? os : [];
}

async function setStatus(orderId, status) {
  await apiClient.put(`/admin/bar/orders/${orderId}/status`, { status }, props.authHeaders);
  await load();
}

onMounted(load);
</script>

<style scoped>
.bar-tabs{display:flex;gap:.5rem;flex-wrap:wrap}.kpi-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(160px,1fr));gap:.75rem}.kpi{padding:1rem;border:1px solid #e5e7eb;border-radius:12px}.status-actions{display:flex;gap:.5rem;flex-wrap:wrap;margin-top:.5rem}
</style>
