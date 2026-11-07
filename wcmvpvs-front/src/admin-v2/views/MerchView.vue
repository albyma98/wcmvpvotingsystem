<template>
  <section class="v2-stack" aria-label="Gestione merchandising">
    <div class="merch-heading">
      <nav class="merch-tabs" aria-label="Sezioni merchandising">
        <a v-for="[id, label] in tabs" :key="id" :href="adminV2Path(c.organizationSlug, 'merch', id)"
          :aria-current="tab === id ? 'page' : undefined" @click.prevent="c.navigate('merch', id)">{{ label }}</a>
      </nav>
      <a class="v2-button" :href="'/' + encodeURIComponent(c.organizationSlug) + '/new/shop'" target="_blank" rel="noopener">
        Apri Shop <V2Icon name="arrow-up-right" :size="16" />
      </a>
    </div>

    <p v-if="loading && tab !== 'orders'" class="v2-muted" role="status">Caricamento merchandising…</p>
    <div v-else-if="error && tab !== 'orders'" class="v2-error" role="alert">
      <p>{{ error }}</p><button type="button" class="v2-link" @click="load">Riprova</button>
    </div>
    <template v-else>
      <template v-if="(tab === '' || tab === 'analytics') && metrics">
        <div class="v2-grid-4">
          <div v-for="metric in summary" :key="metric.label" class="v2-metric">
            <span>{{ metric.label }}</span><strong>{{ metric.value }}</strong><small v-if="metric.note">{{ metric.note }}</small>
          </div>
        </div>
        <div class="v2-grid-2">
          <V2Panel title="Prodotti più visualizzati">
            <div v-if="!metrics.top_products?.length" class="v2-empty">Le visualizzazioni appariranno qui quando i tifosi visiteranno lo Shop.</div>
            <ul v-else class="merch-list">
              <li v-for="product in metrics.top_products" :key="product.id"><span>{{ product.name }}</span><strong>{{ product.views }} visite</strong></li>
            </ul>
          </V2Panel>
          <V2Panel title="Disponibilità limitata">
            <div v-if="!metrics.low_stock?.length" class="v2-empty">Nessun prodotto pubblicato con disponibilità limitata.</div>
            <ul v-else class="merch-list">
              <li v-for="product in metrics.low_stock" :key="product.id"><span>{{ product.name }}</span><span class="v2-badge warning">{{ product.stock }} disponibili</span></li>
            </ul>
          </V2Panel>
        </div>
        <V2Panel v-if="tab === 'analytics'" title="Eventi nello Shop">
          <dl class="merch-list">
            <div v-for="[event, label] in analyticsEvents" :key="event">
              <dt>{{ label }}</dt><dd>{{ metrics.events?.[event] || 0 }}</dd>
            </div>
          </dl>
        </V2Panel>
      </template>

      <V2Panel v-else-if="tab === 'products'" title="Catalogo prodotti" description="Organizza le schede pubbliche, le varianti e le disponibilità.">
        <template #actions>
          <button class="v2-button v2-primary" type="button" @click="editing = {}"><V2Icon name="plus" :size="18" /> Nuovo prodotto</button>
        </template>
        <div class="v2-stack">
          <div class="v2-toolbar">
            <label class="v2-field">Cerca prodotti<input v-model="productQuery" type="search" placeholder="Nome, categoria o SKU" /></label>
            <label class="v2-field">Pubblicazione<select v-model="productStatus">
              <option value="">Tutti</option><option value="published">Pubblicati</option><option value="draft">Bozze</option>
            </select></label>
          </div>
          <div v-if="!filteredProducts.length" class="v2-empty">{{ products.length ? 'Nessun prodotto corrisponde ai filtri.' : 'Il catalogo è vuoto. Crea il primo prodotto.' }}</div>
          <div v-else class="v2-table-wrap" tabindex="0" role="region" aria-label="Catalogo prodotti">
            <table class="v2-table">
              <thead><tr><th scope="col">Prodotto</th><th scope="col">Prezzo</th><th scope="col">Disponibilità</th><th scope="col">Stato</th><th scope="col">Azioni</th></tr></thead>
              <tbody>
                <tr v-for="product in filteredProducts" :key="product.id">
                  <td>
                    <div class="merch-product">
                      <img v-if="product.media?.length" :src="resolveApiUrl(merchMediaPath(product.media[0].url))" alt="" loading="lazy" />
                      <span v-else class="merch-thumbnail"><V2Icon name="merch" :size="20" /></span>
                      <div><strong>{{ product.name }}</strong><br /><small>{{ [product.category, product.sku].filter(Boolean).join(' · ') }}</small></div>
                    </div>
                  </td>
                  <td>{{ money(product.promo_price_cents ?? product.price_cents) }}</td>
                  <td>{{ productAvailability(product) }}</td>
                  <td>
                    <span class="v2-badge" :class="{ success: product.published }">{{ product.published ? 'Pubblicato' : 'Bozza' }}</span>
                    <small v-if="product.featured" class="merch-sub">In evidenza</small>
                  </td>
                  <td><div class="v2-toolbar">
                    <button class="v2-button" type="button" @click="editing = product">Modifica</button>
                    <button class="v2-button v2-danger" type="button" @click="deleteProduct(product)">Elimina</button>
                  </div></td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </V2Panel>

      <V2Panel v-else-if="tab === 'orders'" title="Ordini e prenotazioni" description="Controlla separatamente incassi, prenotazioni ed evasione.">
        <div class="v2-stack">
          <div class="v2-toolbar">
            <label class="v2-field">Cerca ordini<input v-model="orderQuery" type="search" maxlength="160" placeholder="Codice, cliente o recapito" /></label>
            <label class="v2-field">Stato<select v-model="orderStatus"><option v-for="[id, label] in orderFilters" :key="id" :value="id">{{ label }}</option></select></label>
            <button class="v2-button merch-refresh" type="button" @click="load"><V2Icon name="refresh" :size="17" /> Aggiorna</button>
          </div>
          <p v-if="orderError || error" class="v2-error" role="alert">{{ orderError || error }}</p>
          <p v-if="loading" class="v2-muted" role="status">Caricamento ordini…</p>
          <div v-else-if="!orders.length" class="v2-empty">{{ orderQuery || orderStatus ? 'Nessun ordine corrisponde ai filtri.' : 'Qui appariranno ordini e prenotazioni dello Shop.' }}</div>
          <div v-else class="v2-table-wrap" tabindex="0" role="region" aria-label="Ordini e prenotazioni">
            <table class="v2-table">
              <thead><tr><th scope="col">Codice e data</th><th scope="col">Cliente</th><th scope="col">Totale</th><th scope="col">Pagamento</th><th scope="col">Evasione</th><th scope="col">Dettagli</th></tr></thead>
              <tbody>
                <tr v-for="order in orders" :key="order.id">
                  <td><strong>{{ order.code }}</strong><small class="merch-sub">{{ timestamp(order.created_at) }}</small></td>
                  <td>{{ order.name }}<small class="merch-sub">{{ order.email || order.phone }}</small></td>
                  <td>{{ money(order.total_cents, order.currency) }}</td>
                  <td><span class="v2-badge" :class="{ success: paid(order) }">{{ paymentLabel(order.payment_status) }}</span><small class="merch-sub">{{ order.mode === 'online' ? 'Stripe Connect' : 'Prenotazione' }}</small></td>
                  <td><span class="v2-badge">{{ fulfillmentLabel(order.fulfillment_status) }}</span></td>
                  <td><button class="v2-button" type="button" :disabled="Boolean(opening)" @click="openOrder(order.id)">{{ opening === order.id ? 'Apertura…' : 'Apri' }}</button></td>
                </tr>
              </tbody>
            </table>
          </div>
          <div v-if="orderOffset || hasMoreOrders" class="v2-toolbar" aria-label="Paginazione ordini">
            <button class="v2-button" type="button" :disabled="loading || !orderOffset" @click="changeOrderPage(-100)">Precedente</button>
            <span class="v2-muted">Pagina {{ 1 + orderOffset / 100 }}</span>
            <button class="v2-button" type="button" :disabled="loading || !hasMoreOrders" @click="changeOrderPage(100)">Successiva</button>
          </div>
        </div>
      </V2Panel>

      <MerchSettings v-else-if="tab === 'settings' && settings" :settings="settings" :persist="saveSettings" @saved="settings = $event" />
    </template>

    <MerchProductEditor v-if="editing" :product="editing" :persist="saveProduct" :upload="uploadMedia" @close="editing = null" @saved="productSaved" />
    <MerchOrderDetail v-if="selected" :order="selected" :persist="updateOrder" @close="selected = null" @updated="orderUpdated" />
  </section>
</template>

<script setup>
import { computed, inject, onBeforeUnmount, ref, watch } from 'vue';
import { apiClient, resolveApiUrl } from '../../api';
import { adminV2Path } from '../router/routes';
import V2Panel from '../components/shared/V2Panel.vue';
import V2Icon from '../components/shared/V2Icon.vue';
import MerchProductEditor from '../components/MerchProductEditor.vue';
import MerchOrderDetail from '../components/MerchOrderDetail.vue';
import MerchSettings from '../components/MerchSettings.vue';
import { fulfillmentLabel, merchMediaPath, merchRequestError, money, orderFilters, paid, paymentLabel, timestamp } from '../merchModel';

const props = defineProps({ detail: { type: String, default: '' } });
const c = inject('admin-v2');
const base = '/v1/club-admin/' + encodeURIComponent(c.organizationSlug) + '/merch';
const tabs = [
  ['', 'Panoramica'], ['products', 'Catalogo'], ['orders', 'Ordini e prenotazioni'],
  ['settings', 'Impostazioni vendita'], ['analytics', 'Analytics'],
];
const endpoints = { '': 'analytics', products: 'products', orders: 'orders', settings: 'settings', analytics: 'analytics' };
const tab = computed(() => Object.hasOwn(endpoints, props.detail) ? props.detail : '');
const loading = ref(true);
const error = ref('');
const products = ref([]);
const orders = ref([]);
const settings = ref(null);
const metrics = ref(null);
const editing = ref(null);
const selected = ref(null);
const productQuery = ref('');
const productStatus = ref('');
const orderStatus = ref('');
const orderQuery = ref('');
const orderOffset = ref(0);
const hasMoreOrders = ref(false);
const orderError = ref('');
const opening = ref('');
let controller;

const analyticsEvents = [
  ['merch_impression', 'Impressioni'],
  ['merch_product_view', 'Visualizzazioni prodotto'],
  ['merch_variant_selected', 'Selezioni variante'],
  ['merch_checkout_started', 'Aperture checkout'],
  ['merch_payment_completed', 'Pagamenti online verificati'],
  ['merch_reservation_created', 'Prenotazioni ricevute'],
  ['merch_outbound_click', 'Click verso il checkout'],
  ['merch_avatar_try_on', 'Interazioni avatar'],
];

const summary = computed(() => {
  const m = metrics.value;
  const entries = [
    { label: 'Prodotti pubblicati', value: m.published_products || 0 },
    { label: 'Ordini ricevuti', value: m.orders_received || 0 },
    { label: 'Prenotazioni da gestire', value: m.pending_reservations || 0 },
    { label: 'Ricavi online confermati', value: money(m.online_revenue_cents), note: 'Pagamenti Stripe al netto dei rimborsi' },
    { label: 'Prenotazioni da pagare', value: m.unpaid_reservations || 0 },
    { label: 'Valore delle prenotazioni offline', value: money(m.offline_reservation_value_cents), note: 'Totale delle prenotazioni attive' },
    { label: 'Aperture checkout', value: m.checkout_started || 0 },
    { label: 'Click verso il checkout', value: m.events?.merch_outbound_click || 0 },
  ];
  if (tab.value === 'analytics') entries.push(
    { label: 'Pagamenti online confermati', value: m.payment_completed || 0 },
    { label: 'Prenotazioni confermate', value: m.confirmed_reservations || 0 },
    { label: 'Conversione checkout', value: new Intl.NumberFormat('it-IT', { style: 'percent', maximumFractionDigits: 1 }).format(m.conversion_rate || 0) },
    { label: 'Checkout abbandonati', value: m.checkout_abandoned || 0 },
  );
  return entries;
});

const filteredProducts = computed(() => {
  const term = productQuery.value.trim().toLocaleLowerCase('it');
  return products.value.filter(product => {
    if (productStatus.value && product.published !== (productStatus.value === 'published')) return false;
    return [product.name, product.sku, product.category].some(value => String(value ?? '').toLocaleLowerCase('it').includes(term));
  });
});
function productAvailability(product) {
  const inventory = product.variants?.length ? product.variants : [product];
  if (inventory.every(item => item.on_demand)) return 'Su ordinazione';
  const total = inventory.filter(item => !item.on_demand).reduce((sum, item) => sum + (item.available ?? item.stock ?? 0), 0);
  return total + ' disponibili' + (inventory.some(item => item.on_demand) ? ' + su ordinazione' : '');
}

async function load() {
  controller?.abort();
  const request = new AbortController();
  controller = request;
  const key = endpoints[tab.value];
  loading.value = true;
  error.value = '';
  const params = key === 'orders' ? { q: orderQuery.value.trim(), status: orderStatus.value, offset: orderOffset.value } : undefined;
  try {
    const { data } = await apiClient.get(base + '/' + key, { ...c.authHeaders, signal: request.signal, params });
    if (request.signal.aborted) return;
    if (key === 'analytics') metrics.value = data;
    if (key === 'products') products.value = data.products || [];
    if (key === 'orders') {
      orders.value = data.orders || [];
      hasMoreOrders.value = Boolean(data.has_more);
    }
    if (key === 'settings') settings.value = data.settings;
  } catch (failure) {
    if (!request.signal.aborted) error.value = merchRequestError(failure);
  } finally {
    if (!request.signal.aborted) loading.value = false;
  }
}

watch(tab, () => {
  editing.value = null;
  selected.value = null;
  orderError.value = '';
  load();
}, { immediate: true });
watch([orderStatus, orderQuery], (_, __, onCleanup) => {
  if (tab.value !== 'orders') return;
  orderOffset.value = 0;
  const timer = setTimeout(load, 250);
  onCleanup(() => clearTimeout(timer));
});
onBeforeUnmount(() => controller?.abort());

function changeOrderPage(delta) {
  orderOffset.value += delta;
  load();
}

async function uploadMedia(file) {
  const form = new FormData();
  form.set('file', file);
  return (await apiClient.post(base + '/media', form, c.authHeaders)).data;
}

async function saveProduct(product) {
  const url = base + '/products';
  const { data } = product.id
    ? await apiClient.put(url + '/' + encodeURIComponent(product.id), product, c.authHeaders)
    : await apiClient.post(url, product, c.authHeaders);
  return data.product;
}

function productSaved(product) {
  editing.value = null;
  c.notify(product.published ? 'Prodotto salvato e pubblicato.' : 'Bozza salvata.');
  load();
}

function deleteProduct(product) {
  c.confirmAction('Eliminare il prodotto?', 'La scheda di “' + product.name + '” sarà rimossa dal catalogo. Gli ordini già ricevuti resteranno nello storico.', async () => {
    try {
      await apiClient.delete(base + '/products/' + encodeURIComponent(product.id), c.authHeaders);
      c.notify('Prodotto eliminato.');
      await load();
    } catch (failure) {
      throw new Error(merchRequestError(failure));
    }
  });
}

async function openOrder(id) {
  const request = controller;
  opening.value = id;
  orderError.value = '';
  try {
    const { data } = await apiClient.get(base + '/orders/' + encodeURIComponent(id), { ...c.authHeaders, signal: request.signal });
    if (!request.signal.aborted) selected.value = data.order;
  } catch (failure) {
    if (!request.signal.aborted) orderError.value = merchRequestError(failure);
  } finally {
    opening.value = '';
  }
}

async function updateOrder(id, payload) {
  const { data } = await apiClient.post(base + '/orders/' + encodeURIComponent(id) + '/actions', payload, c.authHeaders);
  return data.order;
}

function orderUpdated(order) {
  selected.value = order;
  orders.value = orders.value.map(item => item.id === order.id ? order : item);
  c.notify('Ordine aggiornato.');
}

async function saveSettings(payload) {
  const { data } = await apiClient.put(base + '/settings', payload, c.authHeaders);
  return data.settings;
}
</script>

<style scoped>
.merch-heading { display: flex; align-items: center; justify-content: space-between; gap: 12px; flex-wrap: wrap; }
.merch-tabs { display: flex; gap: 4px; flex-wrap: wrap; }
.merch-tabs a { display: inline-flex; align-items: center; min-height: 44px; padding: 9px 13px; border-radius: 8px; color: var(--v2-muted); font-weight: 600; }
.merch-tabs a[aria-current] { background: var(--v2-primary-soft); color: var(--v2-primary); }
.merch-product { display: flex; align-items: center; gap: 10px; min-width: 165px; }
.merch-product img, .merch-thumbnail { width: 45px; height: 45px; flex: none; object-fit: cover; border-radius: 7px; background: var(--v2-soft); }
.merch-thumbnail { display: grid; place-items: center; color: var(--v2-muted); }
.merch-sub { display: block; margin-top: 3px; }
.merch-list { margin: 0; padding: 0; list-style: none; }
.merch-list > * { display: flex; justify-content: space-between; align-items: center; gap: 8px; padding: 12px 0; border-bottom: 1px solid var(--v2-line); }
.merch-list > :last-child { border-bottom: 0; }
.merch-list dd { margin: 0; font-weight: 650; }
.merch-refresh { align-self: end; }
</style>
