<template>
  <div class="shop-admin">
    <header class="shop-admin__header">
      <div>
        <p class="eyebrow">Wearing Cash · Admin ecommerce</p>
        <h1>Dashboard shop</h1>
        <p class="subtitle">Gestisci ordini e catalogo prodotti del negozio.</p>
      </div>
      <button v-if="isAuthenticated" type="button" class="btn btn-outline" @click="goToStore">
        ← Torna allo shop
      </button>
    </header>

    <section v-if="!isAuthenticated" class="card login-card">
      <h2>Accedi</h2>
      <p class="login-card__subtitle">Utilizza le stesse credenziali dell'area amministratore principale.</p>
      <form class="form-grid" @submit.prevent="login">
        <label>
          Username
          <input v-model.trim="loginForm.username" type="text" autocomplete="username" required />
        </label>
        <label>
          Password
          <input v-model="loginForm.password" type="password" autocomplete="current-password" required />
        </label>
        <button type="submit" class="btn btn-primary" :disabled="isLoggingIn">
          {{ isLoggingIn ? 'Accesso in corso…' : 'Entra' }}
        </button>
      </form>
      <p v-if="loginError" class="feedback feedback--error">{{ loginError }}</p>
    </section>

    <section v-else class="portal" ref="portalRef">
      <div class="portal__toolbar">
        <div class="user-info">
          <span class="user-info__label">
            Connesso come <strong>{{ activeUsername }}</strong>
          </span>
          <nav class="tabs">
            <button
              type="button"
              class="tab"
              :class="{ active: activeSection === 'orders' }"
              @click="goToSection('orders')"
            >
              Ordini
            </button>
            <button
              type="button"
              class="tab"
              :class="{ active: activeSection === 'products' }"
              @click="goToSection('products')"
            >
              Prodotti
            </button>
          </nav>
        </div>
        <button type="button" class="btn btn-secondary" @click="logout">Esci</button>
      </div>

      <div class="portal__content">
        <p v-if="globalError" class="feedback feedback--error">{{ globalError }}</p>

        <section v-if="activeSection === 'orders'" class="panel">
          <header class="panel__header">
            <div>
              <h2>Ordini</h2>
              <p class="panel__subtitle">Consulta gli ordini completati dai clienti.</p>
            </div>
            <button type="button" class="btn btn-outline" :disabled="isLoadingOrders" @click="refreshOrders">
              Aggiorna
            </button>
          </header>
          <div v-if="ordersError" class="feedback feedback--error">{{ ordersError }}</div>
          <div v-else-if="isLoadingOrders" class="feedback">Caricamento ordini…</div>
          <div v-else-if="orders.length === 0" class="feedback">Nessun ordine disponibile al momento.</div>
          <ul v-else class="orders">
            <li v-for="order in orders" :key="order.id" class="orders__item card">
              <header class="orders__header">
                <div>
                  <h3>Ordine #{{ order.id }}</h3>
                  <p class="orders__meta">
                    {{ formatDateTime(order.created_at) }} · {{ order.customer_name }} · {{ order.customer_email }}
                  </p>
                </div>
                <strong class="orders__total">{{ formatPrice(order.total_cents) }}</strong>
              </header>
              <p v-if="order.customer_notes" class="orders__notes">
                <strong>Note cliente:</strong> {{ order.customer_notes }}
              </p>
              <ul class="orders__products">
                <li v-for="item in order.items" :key="item.id" class="orders__product">
                  <div class="orders__product-info">
                    <span class="orders__product-name">{{ item.product_name }}</span>
                    <span class="orders__product-qty">Quantità: {{ item.quantity }}</span>
                  </div>
                  <span class="orders__product-price">{{ formatPrice(item.unit_price_cents) }}</span>
                </li>
              </ul>
            </li>
          </ul>
        </section>

        <section v-else class="panel">
          <header class="panel__header">
            <div>
              <h2>Prodotti</h2>
              <p class="panel__subtitle">Gestisci il catalogo disponibile nello shop pubblico.</p>
            </div>
            <button type="button" class="btn btn-outline" :disabled="isLoadingProducts" @click="refreshProducts">
              Aggiorna
            </button>
          </header>

          <div class="product-form card">
            <h3>Nuovo prodotto</h3>
            <form class="form-grid" @submit.prevent="createProduct">
              <label>
                Nome
                <input v-model.trim="newProductForm.name" type="text" required />
              </label>
              <label>
                Prezzo (€)
                <input v-model="newProductForm.price" type="number" min="0" step="0.01" required />
              </label>
              <label class="form-grid__full">
                Descrizione
                <textarea v-model.trim="newProductForm.description" rows="3" placeholder="Dettagli del prodotto"></textarea>
              </label>
              <label class="form-grid__full">
                Immagine (URL)
                <input v-model.trim="newProductForm.imageUrl" type="url" placeholder="https://…" />
              </label>
              <div class="form-actions">
                <button type="submit" class="btn btn-primary" :disabled="isSavingProduct">
                  {{ isSavingProduct ? 'Salvataggio…' : 'Crea prodotto' }}
                </button>
              </div>
            </form>
            <p v-if="productError" class="feedback feedback--error">{{ productError }}</p>
            <p v-else-if="productSuccess" class="feedback feedback--success">{{ productSuccess }}</p>
          </div>

          <div v-if="productsError" class="feedback feedback--error">{{ productsError }}</div>
          <div v-else-if="isLoadingProducts" class="feedback">Caricamento prodotti…</div>
          <div v-else-if="products.length === 0" class="feedback">Non ci sono prodotti in catalogo.</div>
          <ul v-else class="products">
            <li v-for="product in products" :key="product.id" class="products__item card">
              <div class="products__main">
                <div class="products__info">
                  <h3>{{ product.name }}</h3>
                  <p class="products__description">{{ product.description || '—' }}</p>
                  <p class="products__meta">{{ formatDateTime(product.created_at) }}</p>
                </div>
                <strong class="products__price">{{ formatPrice(product.price_cents) }}</strong>
              </div>
              <p v-if="product.image_url" class="products__image-url">{{ product.image_url }}</p>
            </li>
          </ul>
        </section>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue';
import { apiClient } from '../../api';

const props = defineProps({
  currentPath: { type: String, default: '/' },
  currentSearch: { type: String, default: '' },
  onNavigate: { type: Function, default: undefined },
});

const loginForm = reactive({
  username: '',
  password: '',
});

const portalRef = ref(null);
const token = ref(localStorage.getItem('adminToken') || '');
const activeUsername = ref(localStorage.getItem('adminUsername') || '');
const isLoggingIn = ref(false);
const loginError = ref('');
const globalError = ref('');

const orders = ref([]);
const ordersError = ref('');
const isLoadingOrders = ref(false);
const ordersLoaded = ref(false);

const products = ref([]);
const productsError = ref('');
const isLoadingProducts = ref(false);
const productsLoaded = ref(false);

const newProductForm = reactive({
  name: '',
  price: '',
  description: '',
  imageUrl: '',
});
const isSavingProduct = ref(false);
const productError = ref('');
const productSuccess = ref('');

const currencyFormatter = new Intl.NumberFormat('it-IT', {
  style: 'currency',
  currency: 'EUR',
});

const authHeaders = computed(() => ({
  headers: {
    Authorization: token.value ? `Bearer ${token.value}` : '',
  },
}));

const routeInfo = computed(() => {
  const path = props.currentPath || '/shop/admin';
  const trimmed = path.replace(/\/+$/, '');
  if (trimmed.startsWith('/shop/admin/products')) {
    return { section: 'products' };
  }
  if (trimmed.startsWith('/shop/admin/orders')) {
    return { section: 'orders' };
  }
  return { section: 'orders' };
});

const activeSection = computed(() => routeInfo.value.section);
const isAuthenticated = computed(() => Boolean(token.value));

function goTo(path, replace = false) {
  if (typeof props.onNavigate === 'function') {
    props.onNavigate(path, replace);
    return;
  }
  if (typeof window !== 'undefined') {
    if (replace) {
      window.location.replace(path);
    } else {
      window.location.assign(path);
    }
  }
}

function goToSection(section) {
  if (section === 'products') {
    goTo('/shop/admin/products');
  } else {
    goTo('/shop/admin/orders');
  }
}

function goToStore() {
  goTo('/shop');
}

function resetCollections() {
  orders.value = [];
  ordersLoaded.value = false;
  ordersError.value = '';
  products.value = [];
  productsLoaded.value = false;
  productsError.value = '';
}

function logout() {
  token.value = '';
  activeUsername.value = '';
  localStorage.removeItem('adminToken');
  localStorage.removeItem('adminUsername');
  localStorage.removeItem('adminRole');
  resetCollections();
  goTo('/shop/admin', true);
}

function handleUnauthorized() {
  logout();
  loginError.value = 'Sessione scaduta. Effettua di nuovo il login.';
}

async function secureRequest(executor) {
  try {
    return await executor();
  } catch (error) {
    if (error?.response?.status === 401) {
      handleUnauthorized();
    } else {
      globalError.value = 'Si è verificato un errore imprevisto. Riprova più tardi.';
    }
    throw error;
  }
}

async function login() {
  if (isLoggingIn.value) {
    return;
  }
  loginError.value = '';
  globalError.value = '';
  try {
    isLoggingIn.value = true;
    const { data } = await apiClient.post('/admin/login', {
      username: loginForm.username,
      password: loginForm.password,
    });
    token.value = data.token;
    activeUsername.value = data.username;
    localStorage.setItem('adminToken', token.value);
    localStorage.setItem('adminUsername', activeUsername.value);
    if (data.role) {
      localStorage.setItem('adminRole', data.role);
    }
    loginForm.username = '';
    loginForm.password = '';
    await refreshSection();
  } catch (error) {
    if (error?.response?.status === 401) {
      loginError.value = 'Credenziali non valide.';
    } else {
      loginError.value = 'Accesso non riuscito. Riprova.';
    }
  } finally {
    isLoggingIn.value = false;
  }
}

function formatPrice(cents) {
  const value = Number.isFinite(Number(cents)) ? Number(cents) : 0;
  return currencyFormatter.format(value / 100);
}

function formatDateTime(value) {
  if (!value) {
    return '—';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString('it-IT', { dateStyle: 'short', timeStyle: 'short' });
}

async function loadOrders(force = false) {
  if (!isAuthenticated.value || (ordersLoaded.value && !force)) {
    return;
  }
  ordersError.value = '';
  isLoadingOrders.value = true;
  try {
    const { data } = await secureRequest(() => apiClient.get('/admin/shop/orders', authHeaders.value));
    orders.value = Array.isArray(data) ? data : [];
    ordersLoaded.value = true;
  } catch (error) {
    if (error?.response?.status && error.response.status !== 401) {
      ordersError.value = 'Impossibile recuperare gli ordini.';
    }
  } finally {
    isLoadingOrders.value = false;
  }
}

async function loadProducts(force = false) {
  if (!isAuthenticated.value || (productsLoaded.value && !force)) {
    return;
  }
  productsError.value = '';
  isLoadingProducts.value = true;
  try {
    const { data } = await secureRequest(() => apiClient.get('/admin/shop/products', authHeaders.value));
    products.value = Array.isArray(data) ? data : [];
    productsLoaded.value = true;
  } catch (error) {
    if (error?.response?.status && error.response.status !== 401) {
      productsError.value = 'Impossibile recuperare i prodotti.';
    }
  } finally {
    isLoadingProducts.value = false;
  }
}

async function refreshOrders() {
  await loadOrders(true);
}

async function refreshProducts() {
  await loadProducts(true);
}

async function refreshSection() {
  if (activeSection.value === 'products') {
    await loadProducts(true);
  } else {
    await loadOrders(true);
  }
}

function resetProductFeedback() {
  productError.value = '';
  productSuccess.value = '';
}

async function createProduct() {
  if (!isAuthenticated.value || isSavingProduct.value) {
    return;
  }
  resetProductFeedback();

  const priceRaw = String(newProductForm.price ?? '').replace(',', '.');
  const priceValue = Number.parseFloat(priceRaw);
  if (!Number.isFinite(priceValue) || priceValue <= 0) {
    productError.value = 'Inserisci un prezzo valido.';
    return;
  }

  const payload = {
    name: newProductForm.name.trim(),
    description: newProductForm.description.trim(),
    price_cents: Math.round(priceValue * 100),
    image_url: newProductForm.imageUrl.trim(),
  };

  if (!payload.name) {
    productError.value = 'Inserisci un nome per il prodotto.';
    return;
  }

  try {
    isSavingProduct.value = true;
    await secureRequest(() => apiClient.post('/admin/shop/products', payload, authHeaders.value));
    productSuccess.value = 'Prodotto creato con successo.';
    Object.assign(newProductForm, { name: '', price: '', description: '', imageUrl: '' });
    productsLoaded.value = false;
    await loadProducts(true);
  } catch (error) {
    if (error?.response?.status === 400) {
      productError.value = error.response?.data?.message || 'Impossibile creare il prodotto.';
    } else if (error?.response?.status !== 401) {
      productError.value = 'Impossibile creare il prodotto.';
    }
  } finally {
    isSavingProduct.value = false;
  }
}

watch(
  [() => activeSection.value, () => token.value],
  async ([section, tokenValue]) => {
    if (!tokenValue) {
      return;
    }
    if (section === 'products') {
      await loadProducts();
    } else {
      await loadOrders();
    }
  },
  { immediate: true }
);

onMounted(() => {
  if (!props.currentPath.startsWith('/shop/admin')) {
    goTo('/shop/admin', true);
  }
});
</script>

<style scoped>
.shop-admin {
  min-height: 100vh;
  padding: 3rem 1.5rem 4rem;
  max-width: 1200px;
  margin: 0 auto;
  color: #0f172a;
}

.shop-admin__header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1.5rem;
  margin-bottom: 2rem;
}

.eyebrow {
  text-transform: uppercase;
  letter-spacing: 0.12em;
  font-size: 0.75rem;
  color: #475569;
  margin-bottom: 0.25rem;
}

.shop-admin__header h1 {
  font-size: 2.25rem;
  margin: 0;
  color: #111827;
}

.subtitle {
  margin-top: 0.5rem;
  color: #475569;
}

.card {
  background: #ffffff;
  border-radius: 1rem;
  box-shadow: 0 25px 50px -12px rgba(15, 23, 42, 0.15);
  padding: 1.75rem;
}

.login-card {
  max-width: 420px;
}

.login-card__subtitle {
  color: #475569;
  margin-bottom: 1rem;
}

.form-grid {
  display: grid;
  gap: 1rem;
}

.form-grid label {
  display: flex;
  flex-direction: column;
  font-weight: 600;
  color: #0f172a;
  gap: 0.35rem;
}

.form-grid input,
.form-grid textarea {
  border: 1px solid #cbd5f5;
  border-radius: 0.75rem;
  padding: 0.65rem 0.85rem;
  font-size: 1rem;
}

.form-grid textarea {
  resize: vertical;
}

.form-grid__full {
  grid-column: 1 / -1;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
}

.btn {
  border-radius: 999px;
  padding: 0.65rem 1.5rem;
  font-weight: 600;
  cursor: pointer;
  transition: transform 0.15s ease, box-shadow 0.15s ease;
  border: none;
}

.btn:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 10px 25px rgba(15, 23, 42, 0.1);
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  box-shadow: none;
}

.btn-primary {
  background: linear-gradient(135deg, #0f172a, #1d4ed8);
  color: #fff;
}

.btn-secondary {
  background: #334155;
  color: #fff;
}

.btn-outline {
  background: transparent;
  border: 1px solid rgba(15, 23, 42, 0.12);
  color: #0f172a;
}

.portal {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.portal__toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 1rem;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 1.25rem;
}

.user-info__label {
  color: #475569;
}

.tabs {
  display: flex;
  gap: 0.5rem;
}

.tab {
  background: rgba(15, 23, 42, 0.05);
  border: none;
  padding: 0.5rem 1.2rem;
  border-radius: 999px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease;
}

.tab.active {
  background: #0f172a;
  color: #ffffff;
}

.portal__content {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.panel {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.panel__header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem;
}

.panel__header h2 {
  margin: 0;
  font-size: 1.75rem;
  color: #0f172a;
}

.panel__subtitle {
  margin: 0.35rem 0 0;
  color: #475569;
}

.feedback {
  background: rgba(15, 23, 42, 0.04);
  border-radius: 0.75rem;
  padding: 1rem 1.25rem;
  color: #0f172a;
}

.feedback--error {
  background: rgba(220, 38, 38, 0.12);
  color: #7f1d1d;
}

.feedback--success {
  background: rgba(16, 185, 129, 0.12);
  color: #065f46;
}

.orders {
  display: grid;
  gap: 1.25rem;
}

.orders__item {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.orders__header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem;
}

.orders__header h3 {
  margin: 0;
  font-size: 1.25rem;
}

.orders__meta {
  margin: 0.25rem 0 0;
  color: #475569;
  font-size: 0.95rem;
}

.orders__total {
  font-size: 1.4rem;
  color: #0f172a;
}

.orders__notes {
  margin: 0;
  padding: 0.75rem 1rem;
  border-radius: 0.75rem;
  background: rgba(59, 130, 246, 0.08);
  color: #1e3a8a;
}

.orders__products {
  list-style: none;
  margin: 0;
  padding: 0;
  display: grid;
  gap: 0.75rem;
}

.orders__product {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 1rem;
}

.orders__product-info {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
}

.orders__product-name {
  font-weight: 600;
  color: #0f172a;
}

.orders__product-qty {
  color: #475569;
}

.orders__product-price {
  font-weight: 600;
  color: #0f172a;
}

.product-form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.product-form h3 {
  margin: 0;
}

.products {
  display: grid;
  gap: 1rem;
}

.products__item {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.products__main {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  align-items: flex-start;
}

.products__description {
  color: #475569;
  margin: 0.4rem 0 0;
}

.products__meta {
  margin: 0.35rem 0 0;
  color: #94a3b8;
  font-size: 0.9rem;
}

.products__price {
  font-size: 1.25rem;
  color: #0f172a;
}

.products__image-url {
  margin: 0;
  color: #2563eb;
  word-break: break-word;
}

@media (max-width: 768px) {
  .shop-admin {
    padding: 2rem 1rem 3rem;
  }

  .shop-admin__header {
    flex-direction: column;
  }

  .portal__toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .user-info {
    flex-direction: column;
    align-items: flex-start;
  }

  .orders__header,
  .products__main,
  .panel__header {
    flex-direction: column;
    align-items: flex-start;
  }

  .orders__total,
  .products__price {
    align-self: flex-end;
  }
}
</style>
