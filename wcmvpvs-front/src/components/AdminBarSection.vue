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

    <div v-else-if="activeTab==='history'" class="stacked-section">
      <header class="simple-header">
        <h3>Storico ordini</h3>
        <p class="muted">Controlla gli ordini passati con filtri rapidi.</p>
      </header>

      <div class="filter-grid">
        <label>
          Data
          <input v-model="historyDate" class="input" type="date" />
        </label>
        <label>
          Stato ordine
          <select v-model="historyStatus" class="input">
            <option value="">Tutti</option>
            <option value="new">Nuovo</option>
            <option value="in_preparazione">In preparazione</option>
            <option value="pronto">Pronto</option>
            <option value="completato">Completato</option>
            <option value="annullato">Annullato</option>
          </select>
        </label>
        <label>
          Cerca ordine
          <input v-model="historySearch" class="input" type="search" placeholder="Es: 1042" />
        </label>
      </div>

      <div class="simple-list">
        <button v-for="order in historyOrders" :key="`history-${order.id}`" class="simple-list__row" type="button" @click="openOrderDetail(order)">
          <strong>#{{ order.id }}</strong>
          <span>{{ formatOrderTime(order.created_at, true) }}</span>
          <span>€ {{ euro(order.total_cents) }}</span>
          <span>{{ statusLabel(order.order_status) }}</span>
          <span>{{ paymentLabel(order.payment_status) }}</span>
          <span>{{ customerLabel(order) }}</span>
        </button>
        <p v-if="!historyOrders.length" class="muted">Nessun ordine trovato con i filtri selezionati.</p>
      </div>
    </div>

    <div v-else-if="activeTab==='menu'" class="stacked-section">
      <header class="simple-header">
        <h3>Menu bar</h3>
        <p class="muted">Crea/elimina prodotti e crea pacchetti menù scontati.</p>
      </header>

      <article class="info-card">
        <h4>Nuovo prodotto</h4>
        <div class="filter-grid">
          <label>Nome<input v-model="newProduct.name" class="input" type="text" /></label>
          <label>Prezzo (€)<input v-model.number="newProduct.priceEuro" class="input" type="number" min="0" step="0.01" /></label>
          <label>Descrizione<input v-model="newProduct.description" class="input" type="text" /></label>
          <label>Categoria
            <select v-model.number="newProduct.categoryId" class="input">
              <option :value="0">Seleziona categoria</option>
              <option v-for="category in barCategories" :key="`product-category-${category.id}`" :value="category.id">{{ category.name }}</option>
            </select>
          </label>
        </div>
        <button class="btn primary" type="button" @click="createProduct">Crea prodotto</button>
      </article>

      <div class="menu-grid">
        <article v-for="product in productCards" :key="product.id" class="menu-card">
          <h4>{{ product.name }}</h4>
          <p class="menu-card__price">€ {{ euro(product.price_cents) }}</p>
          <p class="muted">{{ product.description || 'Prodotto bar' }}</p>
          <p class="muted"><strong>Categoria:</strong> {{ product.category || 'Non assegnata' }}</p>
          <div class="menu-card__actions">
            <button class="btn danger" type="button" @click="deleteProduct(product.id)">Elimina definitivamente</button>
          </div>
        </article>
      </div>

      <article class="info-card">
        <h4>Nuovo menù (pacchetto)</h4>
        <div class="filter-grid">
          <label>Nome<input v-model="newMenu.name" class="input" type="text" /></label>
          <label>Prezzo menù (€)<input v-model.number="newMenu.priceEuro" class="input" type="number" min="0" step="0.01" /></label>
          <label>Descrizione<input v-model="newMenu.description" class="input" type="text" /></label>
        </div>
        <div class="simple-list">
          <label v-for="product in productCards" :key="`menu-item-${product.id}`" class="setting-row">
            <input type="checkbox" :checked="isMenuProductSelected(product.id)" @change="toggleMenuProduct(product.id, $event.target.checked)" />
            {{ product.name }}
            <input class="input" type="number" min="1" style="max-width:90px" :disabled="!isMenuProductSelected(product.id)" :value="menuQty(product.id)" @input="setMenuQty(product.id, $event.target.value)" />
          </label>
        </div>
        <button class="btn primary" type="button" @click="createMenu">Crea menù</button>
      </article>

      <article class="info-card">
        <h4>Menù esistenti</h4>
        <div class="simple-list">
          <div v-for="menu in barMenus" :key="`bar-menu-${menu.id}`" class="simple-list__row">
            <strong>{{ menu.name }}</strong>
            <span>€ {{ euro(menu.price_cents) }}</span>
            <span>{{ menu.description || '—' }}</span>
            <span>{{ (menu.items || []).length }} prodotti</span>
            <button class="btn danger" type="button" @click="deleteMenu(menu.id)">Elimina definitivamente</button>
          </div>
        </div>
      </article>
    </div>


    <div v-else-if="activeTab==='categories'" class="stacked-section">
      <header class="simple-header">
        <h3>Categorie</h3>
        <p class="muted">CRUD categorie prodotti BAR con immagine obbligatoria.</p>
      </header>

      <article class="info-card">
        <h4>{{ editingCategoryId ? 'Modifica categoria' : 'Nuova categoria' }}</h4>
        <div class="filter-grid">
          <label>Nome<input v-model="newCategory.name" class="input" type="text" /></label>
          <label>Immagine categoria (obbligatoria)
            <input class="input" type="file" accept="image/*" @change="onCategoryImageSelect" />
          </label>
        </div>
        <p v-if="newCategory.image_url" class="muted">Immagine selezionata ✅</p>
        <div class="menu-card__actions">
          <button class="btn primary" type="button" @click="saveCategory">{{ editingCategoryId ? 'Salva modifiche' : 'Crea categoria' }}</button>
          <button v-if="editingCategoryId" class="btn outline" type="button" @click="resetCategoryForm">Annulla modifica</button>
        </div>
      </article>

      <div class="simple-list">
        <div v-for="category in barCategories" :key="`bar-category-${category.id}`" class="simple-list__row">
          <strong>{{ category.name }}</strong>
          <img :src="category.image_url" :alt="category.name" class="h-12 w-16 rounded object-cover" />
          <button class="btn outline" type="button" @click="editCategory(category)">Modifica</button>
          <button class="btn danger" type="button" @click="deleteCategory(category.id)">Elimina</button>
        </div>
        <p v-if="!barCategories.length" class="muted">Nessuna categoria disponibile.</p>
      </div>
    </div>

    <div v-else-if="activeTab==='quick'" class="stacked-section">
      <header class="simple-header">
        <h3>Disponibilità rapida</h3>
        <p class="muted">Aggiorna la disponibilità in pochi secondi durante la partita.</p>
      </header>

      <div class="quick-list">
        <article v-for="product in productCards" :key="`quick-${product.id}`" class="quick-card">
          <div>
            <h4>{{ product.name }}</h4>
            <p class="muted">Stato attuale: <strong>{{ product.available ? 'Disponibile' : 'Esaurito' }}</strong></p>
          </div>
          <div class="quick-card__actions">
            <button class="btn quick-btn" :class="product.available ? 'primary' : 'outline'" type="button" @click="setProductAvailability(product.id, true)">Disponibile</button>
            <button class="btn quick-btn" :class="!product.available ? 'danger' : 'outline'" type="button" @click="setProductAvailability(product.id, false)">Esaurito</button>
          </div>
        </article>
      </div>
    </div>

    <div v-else-if="activeTab==='stats'" class="stacked-section">
      <header class="simple-header">
        <h3>Statistiche</h3>
        <p class="muted">Panoramica rapida vendite bar per evento.</p>
      </header>

      <div class="kpi-grid">
        <article class="kpi" v-for="card in statsCards" :key="card.label">
          <h3>{{ card.label }}</h3>
          <p>{{ card.value }}</p>
        </article>
      </div>

      <article class="info-card">
        <h4>Prodotti più venduti</h4>
        <ul class="item-list compact">
          <li v-for="item in topProducts" :key="item.id">{{ item.name }} • {{ item.qty }} pz</li>
          <li v-if="!topProducts.length" class="muted">Ancora nessun dato disponibile.</li>
        </ul>
      </article>
    </div>

    <div v-else-if="activeTab==='clients'" class="stacked-section clients-layout">
      <header class="simple-header">
        <h3>Clienti</h3>
        <p class="muted">Riepilogo clienti e storico ordini.</p>
      </header>

      <div class="clients-grid">
        <article class="info-card">
          <h4>Lista clienti</h4>
          <button
            v-for="client in clientRows"
            :key="client.id"
            class="simple-list__row"
            type="button"
            @click="selectedClientId = client.id"
          >
            <strong>{{ client.name }}</strong>
            <span>{{ client.orders }} ordini</span>
            <span>€ {{ euro(client.totalCents) }}</span>
            <span>Ultimo: {{ formatOrderTime(client.lastOrderAt, true) }}</span>
          </button>
          <p v-if="!clientRows.length" class="muted">Nessun cliente disponibile.</p>
        </article>

        <article class="info-card">
          <h4>Storico cliente</h4>
          <p v-if="!selectedClient" class="muted">Seleziona un cliente per vedere gli ordini.</p>
          <template v-else>
            <p><strong>{{ selectedClient.name }}</strong> • {{ selectedClient.orders }} ordini • € {{ euro(selectedClient.totalCents) }}</p>
            <ul class="item-list compact">
              <li v-for="order in selectedClientOrders" :key="`client-order-${order.id}`">
                <button type="button" class="link-btn" @click="openOrderDetail(order)">
                  #{{ order.id }} • {{ formatOrderTime(order.created_at, true) }} • {{ statusLabel(order.order_status) }} • € {{ euro(order.total_cents) }}
                </button>
              </li>
            </ul>
          </template>
        </article>
      </div>
    </div>

    <div v-else-if="activeTab==='cash'" class="stacked-section">
      <header class="simple-header">
        <h3>Cassa</h3>
        <p class="muted">Riepilogo chiusura giornata o evento.</p>
      </header>

      <div class="kpi-grid">
        <article class="kpi" v-for="card in cashCards" :key="card.label">
          <h3>{{ card.label }}</h3>
          <p>{{ card.value }}</p>
        </article>
      </div>
    </div>

    <div v-else-if="activeTab==='settings'" class="stacked-section">
      <header class="simple-header">
        <h3>Impostazioni BAR</h3>
        <p class="muted">Configura preferenze operative del bar.</p>
      </header>

      <div class="settings-grid">
        <article class="info-card">
          <h4>Dati bar</h4>
          <p><strong>Partner BAR:</strong> {{ settings.partnerName }}</p>
          <p><strong>Società associata:</strong> {{ settings.companyName }}</p>
          <p class="muted">Accesso dati: ruolo BAR vede solo il proprio bar, super admin vede tutti i bar.</p>
        </article>

        <article class="info-card">
          <h4>Preferenze operative</h4>
          <label class="setting-row">
            <input v-model="settings.notifyNewOrders" type="checkbox" />
            Notifiche nuovi ordini
          </label>
          <label class="setting-row">
            <input v-model="settings.soundAlerts" type="checkbox" />
            Avvisi sonori priorità ordini
          </label>
          <label class="setting-row">
            <input v-model="settings.autoRefreshLive" type="checkbox" />
            Aggiornamento automatico ordini live
          </label>
        </article>
      </div>
    </div>

    <div v-else class="muted">Sezione non disponibile.</div>

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
              <span>{{ item.name }} • € {{ euro(item.priceCents) }}</span>
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
const productsCatalog = ref([]);
const productState = ref({});
const barMenus = ref([]);
const barCategories = ref([]);
const editingCategoryId = ref(0);
const newCategory = ref({ name: '', image_url: '' });
const newProduct = ref({ name: '', description: '', priceEuro: 0, categoryId: 0 });
const newMenu = ref({ name: '', description: '', priceEuro: 0, items: {} });
const historyDate = ref('');
const historyStatus = ref('');
const historySearch = ref('');
const selectedClientId = ref('');
const settings = ref({
  partnerName: 'BAR Partner',
  companyName: 'Società evento',
  notifyNewOrders: true,
  soundAlerts: true,
  autoRefreshLive: true,
});

const RECENT_CANCELLED_MINUTES = 20;
const refreshEveryMs = 15000;
let refreshTimer = null;

const tabs = [
  { id: 'dashboard', label: 'Dashboard' },
  { id: 'live', label: 'Ordini live' },
  { id: 'history', label: 'Storico ordini' },
  { id: 'menu', label: 'Menu' },
  { id: 'categories', label: 'Categorie' },
  { id: 'quick', label: 'Disponibilità rapida' },
  { id: 'stats', label: 'Statistiche' },
  { id: 'clients', label: 'Clienti' },
  { id: 'cash', label: 'Cassa' },
  { id: 'settings', label: 'Impostazioni' },
];

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
  }).sort((a, b) => parseDate(b.created_at) - parseDate(a.created_at));
});

const historyOrders = computed(() => parsedOrders.value.filter((order) => {
  if (historyStatus.value && order.order_status !== historyStatus.value) {
    return false;
  }
  if (historyDate.value) {
    const day = parseDate(order.created_at).toISOString().slice(0, 10);
    if (day !== historyDate.value) {
      return false;
    }
  }
  if (historySearch.value) {
    return String(order.id).includes(historySearch.value.trim());
  }
  return true;
}));

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

const productCards = computed(() => productsCatalog.value.map((product) => {
  const state = productState.value[product.id] || { active: true, available: true };
  return { ...product, ...state };
}));

const topProducts = computed(() => {
  const map = new Map();
  parsedOrders.value.forEach((order) => {
    order.parsedItems.forEach((item) => {
      map.set(item.id, { id: item.id, name: item.name, qty: (map.get(item.id)?.qty || 0) + item.quantity });
    });
  });
  return Array.from(map.values()).sort((a, b) => b.qty - a.qty).slice(0, 5);
});

const busiestHour = computed(() => {
  const byHour = new Map();
  parsedOrders.value.forEach((order) => {
    const hour = parseDate(order.created_at).getHours();
    byHour.set(hour, (byHour.get(hour) || 0) + 1);
  });
  const sorted = Array.from(byHour.entries()).sort((a, b) => b[1] - a[1]);
  if (!sorted.length) return 'N/D';
  const [hour, count] = sorted[0];
  return `${String(hour).padStart(2, '0')}:00 • ${count} ordini`;
});

const statsCards = computed(() => {
  const total = parsedOrders.value.reduce((acc, order) => acc + Number(order.total_cents || 0), 0);
  const count = parsedOrders.value.length;
  return [
    { label: 'Incasso totale evento', value: `€ ${euro(total)}` },
    { label: 'Numero totale ordini', value: count },
    { label: 'Ticket medio', value: `€ ${count ? euro(total / count) : '0.00'}` },
    { label: 'Fascia oraria con più ordini', value: busiestHour.value },
  ];
});

const clientRows = computed(() => {
  const map = new Map();
  parsedOrders.value.forEach((order) => {
    const id = customerId(order);
    const current = map.get(id) || {
      id,
      name: customerLabel(order),
      orders: 0,
      totalCents: 0,
      lastOrderAt: order.created_at,
    };
    current.orders += 1;
    current.totalCents += Number(order.total_cents || 0);
    if (parseDate(order.created_at) > parseDate(current.lastOrderAt)) {
      current.lastOrderAt = order.created_at;
    }
    map.set(id, current);
  });
  return Array.from(map.values()).sort((a, b) => b.orders - a.orders);
});

const selectedClient = computed(() => clientRows.value.find((client) => client.id === selectedClientId.value) || null);
const selectedClientOrders = computed(() => {
  if (!selectedClient.value) return [];
  return parsedOrders.value.filter((order) => customerId(order) === selectedClient.value.id);
});

const cashCards = computed(() => {
  const completed = parsedOrders.value.filter((order) => order.order_status === 'completato').length;
  const cancelled = parsedOrders.value.filter((order) => order.order_status === 'annullato').length;
  const soldProducts = parsedOrders.value.reduce((acc, order) => acc + order.parsedItems.reduce((inner, item) => inner + item.quantity, 0), 0);
  const total = parsedOrders.value.reduce((acc, order) => acc + Number(order.total_cents || 0), 0);
  return [
    { label: 'Incasso totale', value: `€ ${euro(total)}` },
    { label: 'Ordini completati', value: completed },
    { label: 'Ordini annullati', value: cancelled },
    { label: 'Prodotti venduti', value: soldProducts },
  ];
});

const lastRefreshLabel = computed(() => {
  if (!lastRefreshAt.value) {
    return 'mai';
  }
  return new Intl.DateTimeFormat('it-IT', { hour: '2-digit', minute: '2-digit', second: '2-digit' }).format(lastRefreshAt.value);
});

async function loadProducts() {
  const { data } = await apiClient.get('/admin/bar/products', props.authHeaders);
  productsCatalog.value = Array.isArray(data) ? data : [];
  const next = {};
  productsCatalog.value.forEach((product) => {
    next[product.id] = productState.value[product.id] || { active: true, available: true };
  });
  productState.value = next;
}

async function loadCategories() {
  const { data } = await apiClient.get('/admin/bar/categories', props.authHeaders);
  barCategories.value = Array.isArray(data) ? data : [];
}

async function loadMenus() {
  const { data } = await apiClient.get('/admin/bar/menus', props.authHeaders);
  barMenus.value = Array.isArray(data) ? data : [];
}

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
    if (!selectedClientId.value && clientRows.value.length) {
      selectedClientId.value = clientRows.value[0].id;
    }
  } finally {
    isLoading.value = false;
  }
}

async function setStatus(orderId, status) {
  await apiClient.put(`/admin/bar/orders/${orderId}/status`, { status }, props.authHeaders);
  await load();
}

async function createProduct() {
  const payload = {
    name: String(newProduct.value.name || '').trim(),
    description: String(newProduct.value.description || '').trim(),
    price_cents: Math.round(Number(newProduct.value.priceEuro || 0) * 100),
    image_url: '',
    category_id: Number(newProduct.value.categoryId || 0),
  };
  if (!payload.name || payload.price_cents <= 0 || payload.category_id <= 0) return;
  await apiClient.post('/admin/bar/products', payload, props.authHeaders);
  newProduct.value = { name: '', description: '', priceEuro: 0, categoryId: 0 };
  await loadProducts();
}

async function deleteProduct(productId) {
  await apiClient.delete(`/admin/bar/products/${productId}`, props.authHeaders);
  await Promise.all([loadProducts(), loadMenus()]);
}

function isMenuProductSelected(productId) {
  return Number(newMenu.value.items[productId] || 0) > 0;
}

function toggleMenuProduct(productId, checked) {
  newMenu.value.items = {
    ...newMenu.value.items,
    [productId]: checked ? Math.max(1, Number(newMenu.value.items[productId] || 1)) : 0,
  };
}

function setMenuQty(productId, value) {
  const qty = Math.max(1, Number(value || 1));
  newMenu.value.items = { ...newMenu.value.items, [productId]: qty };
}

function menuQty(productId) {
  return Math.max(1, Number(newMenu.value.items[productId] || 1));
}

async function createMenu() {
  const items = Object.entries(newMenu.value.items)
    .filter(([, qty]) => Number(qty) > 0)
    .map(([productId, qty]) => ({ product_id: Number(productId), quantity: Number(qty) }));
  const payload = {
    name: String(newMenu.value.name || '').trim(),
    description: String(newMenu.value.description || '').trim(),
    price_cents: Math.round(Number(newMenu.value.priceEuro || 0) * 100),
    items,
  };
  if (!payload.name || payload.price_cents <= 0 || !payload.items.length) return;
  await apiClient.post('/admin/bar/menus', payload, props.authHeaders);
  newMenu.value = { name: '', description: '', priceEuro: 0, items: {} };
  await loadMenus();
}


async function onCategoryImageSelect(event) {
  const file = event?.target?.files?.[0];
  if (!file) return;
  newCategory.value.image_url = await fileToDataUrl(file);
}

function fileToDataUrl(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result || ''));
    reader.onerror = reject;
    reader.readAsDataURL(file);
  });
}

function editCategory(category) {
  editingCategoryId.value = Number(category.id || 0);
  newCategory.value = { name: String(category.name || ''), image_url: String(category.image_url || '') };
}

function resetCategoryForm() {
  editingCategoryId.value = 0;
  newCategory.value = { name: '', image_url: '' };
}

async function saveCategory() {
  const payload = { name: String(newCategory.value.name || '').trim(), image_url: String(newCategory.value.image_url || '').trim() };
  if (!payload.name || !payload.image_url) return;
  if (editingCategoryId.value > 0) {
    await apiClient.put(`/admin/bar/categories/${editingCategoryId.value}`, payload, props.authHeaders);
  } else {
    await apiClient.post('/admin/bar/categories', payload, props.authHeaders);
  }
  resetCategoryForm();
  await loadCategories();
}

async function deleteCategory(categoryId) {
  await apiClient.delete(`/admin/bar/categories/${categoryId}`, props.authHeaders);
  await Promise.all([loadCategories(), loadProducts()]);
}

async function deleteMenu(menuId) {
  await apiClient.delete(`/admin/bar/menus/${menuId}`, props.authHeaders);
  await loadMenus();
}

function setProductAvailability(productId, available) {
  const prev = productState.value[productId] || { active: true, available: true };
  productState.value = {
    ...productState.value,
    [productId]: { ...prev, available },
  };
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
    const priceCents = Number(product.price_cents || product.priceCents || 0);
    const quantity = Math.max(1, quantityMap.get(id) || 1);
    return {
      id,
      name: product.name || product.id || 'Prodotto',
      quantity,
      priceCents,
      lineTotalCents: priceCents * quantity,
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

function customerId(order) {
  return String(order.customer_name || order.stripe_reference || `ordine-${order.id}`);
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
  await Promise.all([load(), loadCategories(), loadProducts(), loadMenus()]);
  refreshTimer = window.setInterval(() => {
    if (settings.value.autoRefreshLive) {
      load();
    }
  }, refreshEveryMs);
});

onBeforeUnmount(() => {
  if (refreshTimer) {
    window.clearInterval(refreshTimer);
  }
});
</script>

<style scoped>
.bar-tabs{display:flex;gap:.5rem;flex-wrap:wrap}.kpi-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(180px,1fr));gap:.75rem}.kpi{padding:1rem;border:1px solid #e5e7eb;border-radius:12px;background:#fff}.kpi h3{margin:0 0 .3rem}.kpi p{margin:0;font-size:1.25rem;font-weight:700}
.stacked-section{display:grid;gap:1rem}
.simple-header h3{margin-bottom:.2rem}
.filter-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(180px,1fr));gap:.75rem}
.input{width:100%;border:1px solid #d0d9e5;border-radius:10px;padding:.6rem .7rem;background:#fff;margin-top:.3rem}
.simple-list{display:grid;gap:.5rem}
.simple-list__row{display:grid;grid-template-columns:1fr repeat(5,minmax(80px,auto));gap:.5rem;align-items:center;border:1px solid #dbe2ea;background:#fff;border-radius:12px;padding:.8rem;text-align:left}
.menu-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:.75rem}
.menu-card{border:1px solid #dbe2ea;border-radius:14px;padding:1rem;background:#fff;display:grid;gap:.6rem}
.menu-card__price{font-size:1.2rem;font-weight:700;margin:0}
.menu-card__badges,.menu-card__actions{display:flex;gap:.45rem;flex-wrap:wrap}
.quick-list{display:grid;gap:.6rem}
.quick-card{border:1px solid #dbe2ea;border-radius:14px;padding:1rem;background:#fff;display:flex;justify-content:space-between;gap:1rem;align-items:center;flex-wrap:wrap}
.quick-card__actions{display:flex;gap:.5rem}
.quick-btn{font-size:1.05rem;min-width:140px;min-height:44px}
.info-card{border:1px solid #dbe2ea;border-radius:14px;background:#fff;padding:1rem}
.clients-grid{display:grid;grid-template-columns:1.1fr 1fr;gap:.8rem}
.settings-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(250px,1fr));gap:.8rem}
.setting-row{display:flex;gap:.5rem;align-items:center;padding:.35rem 0}
.link-btn{border:none;background:none;color:#1d4ed8;padding:0;text-decoration:underline;cursor:pointer}
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
@media (max-width: 900px) {
  .simple-list__row { grid-template-columns: 1fr; }
  .clients-grid { grid-template-columns: 1fr; }
}
</style>
