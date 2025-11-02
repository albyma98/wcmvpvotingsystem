<template>
  <div class="shop">
    <header class="shop-header">
      <div class="brand" @click="goToShopHome">
        <span class="brand-mark">WC</span>
        <span class="brand-name">Wearing Cash</span>
      </div>
      <nav class="shop-actions">
        <button
          type="button"
          class="nav-btn"
          :class="{ active: routeInfo.name === 'list' }"
          @click="goToShopHome"
        >
          Shop
        </button>
        <button
          type="button"
          class="nav-btn"
          :class="{ active: routeInfo.name === 'checkout' }"
          @click="goToCheckout"
        >
          Carrello
          <span v-if="cartCount">({{ cartCount }})</span>
          <span v-if="cartTotalCents">· {{ cartTotalFormatted }}</span>
        </button>
      </nav>
    </header>

    <main class="shop-main">
      <section v-if="routeInfo.name === 'list'" class="view view-list">
        <div class="intro">
          <h1>Collezione Wearing Cash</h1>
          <p>
            Una selezione di capi iconici del brand Wearing Cash. Scorri i prodotti, aggiungi al carrello e completa
            un ordine di prova in pochi passaggi.
          </p>
          <div class="intro-actions">
            <button type="button" class="btn btn-primary" :disabled="cartItems.length === 0" @click="goToCheckout">
              Vai al checkout
            </button>
          </div>
        </div>
        <div v-if="productsError" class="message message-error">{{ productsError }}</div>
        <div v-else-if="isLoadingProducts" class="message">Caricamento dei prodotti…</div>
        <div v-else-if="products.length === 0" class="message">
          Non ci sono prodotti disponibili al momento. Torna più tardi!
        </div>
        <div v-else class="product-grid">
          <article v-for="product in products" :key="product.id" class="product-card" @click="viewProduct(product.id)">
            <div class="product-image">
              <img :src="product.imageUrl" :alt="product.name" loading="lazy" />
            </div>
            <div class="product-content">
              <h3>{{ product.name }}</h3>
              <p>{{ product.description }}</p>
            </div>
            <div class="product-footer">
              <span class="product-price">{{ formatPrice(product.priceCents) }}</span>
              <div class="product-actions">
                <button type="button" class="btn btn-secondary" @click.stop="addToCart(product)">Aggiungi</button>
                <button type="button" class="btn btn-outline" @click.stop="viewProduct(product.id)">Dettagli</button>
              </div>
            </div>
          </article>
        </div>
      </section>

      <section v-else-if="routeInfo.name === 'detail'" class="view view-detail">
        <button type="button" class="link" @click="goToShopHome">← Torna alla collezione</button>
        <div v-if="isLoadingProduct" class="message">Caricamento prodotto…</div>
        <div v-else-if="productError" class="message message-error">{{ productError }}</div>
        <div v-else-if="selectedProduct" class="detail-card">
          <div class="detail-image">
            <img :src="selectedProduct.imageUrl" :alt="selectedProduct.name" />
          </div>
          <div class="detail-info">
            <h1>{{ selectedProduct.name }}</h1>
            <p class="detail-price">{{ formatPrice(selectedProduct.priceCents) }}</p>
            <p class="detail-description">{{ selectedProduct.description }}</p>
            <div class="detail-actions">
              <button type="button" class="btn btn-primary" @click="addToCart(selectedProduct)">Aggiungi al carrello</button>
              <button type="button" class="btn btn-secondary" :disabled="cartItems.length === 0" @click="goToCheckout">
                Vai al checkout
              </button>
            </div>
          </div>
        </div>
      </section>

      <section v-else-if="routeInfo.name === 'checkout'" class="view view-checkout">
        <div class="checkout-columns">
          <div class="card">
            <h1>Checkout</h1>
            <p class="checkout-subtitle">
              Inserisci nome ed email per simulare la conferma dell'ordine Wearing Cash.
            </p>
            <form class="checkout-form" @submit.prevent="submitOrder">
              <label class="field">
                <span>Nome e cognome</span>
                <input v-model="checkoutForm.name" type="text" placeholder="Il tuo nome" autocomplete="name" required />
              </label>
              <label class="field">
                <span>Email</span>
                <input
                  v-model="checkoutForm.email"
                  type="email"
                  placeholder="nome@email.com"
                  autocomplete="email"
                  required
                />
              </label>
              <label class="field">
                <span>Note (facoltative)</span>
                <textarea v-model="checkoutForm.notes" rows="3" placeholder="Richieste particolari o preferenze"></textarea>
              </label>
              <p v-if="checkoutError" class="form-error">{{ checkoutError }}</p>
              <button type="submit" class="btn btn-primary" :disabled="isCheckoutDisabled">
                {{ checkoutButtonLabel }}
              </button>
            </form>
          </div>
          <div class="card">
            <div class="summary-header">
              <h2>Carrello</h2>
              <span v-if="cartItems.length">{{ cartTotalFormatted }}</span>
            </div>
            <div v-if="cartItems.length === 0" class="message">
              Il carrello è vuoto. Aggiungi qualche prodotto dalla collezione.
            </div>
            <ul v-else class="summary-list">
              <li v-for="item in cartItems" :key="item.product.id" class="summary-item">
                <div class="summary-info">
                  <img :src="item.product.imageUrl" :alt="item.product.name" />
                  <div>
                    <p class="summary-name">{{ item.product.name }}</p>
                    <p class="summary-price">{{ formatPrice(item.product.priceCents) }}</p>
                  </div>
                </div>
                <div class="summary-controls">
                  <div class="quantity">
                    <button type="button" @click="decrementCart(item.product.id)" :disabled="item.quantity <= 1">−</button>
                    <input
                      type="number"
                      min="1"
                      :value="item.quantity"
                      @input="handleQuantityInput(item.product.id, $event)"
                    />
                    <button type="button" @click="incrementCart(item.product.id)">+</button>
                  </div>
                  <button type="button" class="link" @click="removeCartItem(item.product.id)">Rimuovi</button>
                </div>
              </li>
            </ul>
            <div class="summary-footer">
              <div>
                <span>Totale ordine</span>
                <strong>{{ cartTotalFormatted }}</strong>
              </div>
              <button type="button" class="btn btn-outline" @click="goToShopHome">Continua lo shopping</button>
            </div>
          </div>
        </div>
      </section>

      <section v-else class="view view-success">
        <div class="card success-card">
          <span class="success-icon">✓</span>
          <h1>Ordine completato</h1>
          <p v-if="successOrderNumber" class="success-code">
            Numero ordine <strong>#{{ successOrderNumber }}</strong>
          </p>
          <p v-if="successOrder?.customerEmail" class="success-text">
            Conferma inviata a <strong>{{ successOrder.customerEmail }}</strong>.
          </p>
          <p v-else class="success-text">Riceverai a breve una conferma via email.</p>

          <div v-if="successOrder && successOrder.items.length" class="success-summary">
            <div v-for="item in successOrder.items" :key="item.id || item.productId" class="success-item">
              <div class="success-item-info">
                <img :src="item.productImageUrl || selectedProductImage(item.productId)" :alt="item.productName" />
                <div>
                  <p class="summary-name">{{ item.productName }}</p>
                  <p class="summary-qty">Quantità: {{ item.quantity }}</p>
                </div>
              </div>
              <div class="summary-price">{{ formatPrice(item.unitPriceCents * item.quantity) }}</div>
            </div>
            <div class="success-total">
              <span>Totale</span>
              <strong>{{ formatPrice(successOrder.totalCents) }}</strong>
            </div>
          </div>

          <div class="success-actions">
            <button type="button" class="btn btn-primary" @click="goToShopHome">Torna allo shop</button>
            <button type="button" class="btn btn-outline" @click="goToCheckout">Rivedi il carrello</button>
          </div>
        </div>
      </section>
    </main>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue';
import { apiClient } from '../../api';

const props = defineProps({
  currentPath: { type: String, required: true },
  currentSearch: { type: String, default: '' },
  onNavigate: { type: Function, required: true },
});

const products = ref([]);
const isLoadingProducts = ref(false);
const productsError = ref('');

const selectedProduct = ref(null);
const isLoadingProduct = ref(false);
const productError = ref('');

const cartItems = ref([]);
const checkoutForm = reactive({ name: '', email: '', notes: '' });
const checkoutError = ref('');
const isSubmittingOrder = ref(false);

const lastOrder = ref(null);
const successDetails = ref(null);

const routeInfo = computed(() => {
  const path = props.currentPath || '/shop';
  const sanitized = path === '/' ? '/shop' : path;
  const trimmed = sanitized.replace(/\/+$/, '');
  if (trimmed.startsWith('/shop/checkout/success')) {
    return { name: 'success' };
  }
  if (trimmed.startsWith('/shop/checkout')) {
    return { name: 'checkout' };
  }
  const match = trimmed.match(/^\/shop\/product\/(\d+)/);
  if (match) {
    const productId = Number.parseInt(match[1], 10);
    return { name: 'detail', productId: Number.isFinite(productId) ? productId : null };
  }
  return { name: 'list' };
});

const currentProductId = computed(() => (routeInfo.value.name === 'detail' ? routeInfo.value.productId : null));

const cartCount = computed(() => cartItems.value.reduce((total, item) => total + item.quantity, 0));
const cartTotalCents = computed(() =>
  cartItems.value.reduce((total, item) => total + item.quantity * (item.product?.priceCents ?? 0), 0)
);
const cartTotalFormatted = computed(() => formatPrice(cartTotalCents.value));

const isCheckoutDisabled = computed(() => {
  return (
    cartItems.value.length === 0 ||
    isSubmittingOrder.value ||
    checkoutForm.name.trim() === '' ||
    checkoutForm.email.trim() === ''
  );
});

const checkoutButtonLabel = computed(() => (isSubmittingOrder.value ? 'Elaborazione…' : "Completa l'ordine"));

const successOrder = computed(() =>
  routeInfo.value.name === 'success' ? successDetails.value || lastOrder.value : null
);

const successOrderNumber = computed(() => {
  if (routeInfo.value.name !== 'success') {
    return '';
  }
  if (successOrder.value?.id) {
    return successOrder.value.id;
  }
  const search = props.currentSearch || (typeof window !== 'undefined' ? window.location.search : '');
  const params = new URLSearchParams(search || '');
  return params.get('order') || params.get('orderId') || '';
});

const currencyFormatter = new Intl.NumberFormat('it-IT', {
  style: 'currency',
  currency: 'EUR',
});

function formatPrice(cents) {
  const value = Number(cents ?? 0);
  const normalized = Number.isFinite(value) ? value : 0;
  return currencyFormatter.format(normalized / 100);
}

function normalizeProduct(raw) {
  if (!raw || typeof raw !== 'object') {
    return null;
  }
  const priceValue = Number(raw.price_cents ?? raw.priceCents ?? 0);
  return {
    id: raw.id ?? 0,
    name: raw.name ?? '',
    description: raw.description ?? '',
    priceCents: Number.isFinite(priceValue) ? Math.round(priceValue) : 0,
    imageUrl: raw.image_url ?? raw.imageUrl ?? '',
    createdAt: raw.created_at ?? raw.createdAt ?? '',
  };
}

function normalizeOrderItem(raw) {
  if (!raw || typeof raw !== 'object') {
    return null;
  }
  const unitPrice = Number(raw.unit_price_cents ?? raw.unitPriceCents ?? 0);
  return {
    id: raw.id ?? 0,
    orderId: raw.order_id ?? raw.orderId ?? 0,
    productId: raw.product_id ?? raw.productId ?? 0,
    productName: raw.product_name ?? raw.productName ?? '',
    productImageUrl: raw.product_image_url ?? raw.productImageUrl ?? '',
    quantity: Number(raw.quantity ?? 0) || 0,
    unitPriceCents: Number.isFinite(unitPrice) ? Math.round(unitPrice) : 0,
  };
}

function normalizeOrder(raw) {
  if (!raw || typeof raw !== 'object') {
    return null;
  }
  const totalValue = Number(raw.total_cents ?? raw.totalCents ?? 0);
  const items = Array.isArray(raw.items)
    ? raw.items.map(normalizeOrderItem).filter((item) => item !== null)
    : [];
  return {
    id: raw.id ?? 0,
    customerName: raw.customer_name ?? raw.customerName ?? '',
    customerEmail: raw.customer_email ?? raw.customerEmail ?? '',
    customerNotes: raw.customer_notes ?? raw.customerNotes ?? '',
    totalCents: Number.isFinite(totalValue) ? Math.round(totalValue) : 0,
    createdAt: raw.created_at ?? raw.createdAt ?? '',
    items,
  };
}

function synchronizeCartProduct(updatedProduct) {
  const normalized = normalizeProduct(updatedProduct);
  if (!normalized) {
    return;
  }
  const match = cartItems.value.find((item) => item.product.id === normalized.id);
  if (match) {
    match.product = normalized;
  }
}

async function fetchProducts(force = false) {
  if (isLoadingProducts.value) {
    return;
  }
  if (!force && products.value.length > 0) {
    return;
  }

  isLoadingProducts.value = true;
  productsError.value = '';
  try {
    const { data } = await apiClient.get('/shop/products');
    const list = Array.isArray(data) ? data : Array.isArray(data?.products) ? data.products : [];
    const normalized = list
      .map((item) => normalizeProduct(item))
      .filter((item) => item !== null);
    products.value = normalized;
    normalized.forEach((product) => synchronizeCartProduct(product));
    if (currentProductId.value) {
      const match = normalized.find((product) => product.id === currentProductId.value);
      if (match) {
        selectedProduct.value = match;
      }
    }
  } catch (error) {
    console.error('Errore caricamento prodotti Wearing Cash', error);
    productsError.value = 'Impossibile caricare i prodotti Wearing Cash. Riprova più tardi.';
  } finally {
    isLoadingProducts.value = false;
  }
}

async function loadProduct(productId) {
  if (!productId) {
    selectedProduct.value = null;
    productError.value = '';
    return;
  }

  const cached = products.value.find((product) => product.id === productId);
  if (cached) {
    selectedProduct.value = cached;
    return;
  }

  isLoadingProduct.value = true;
  productError.value = '';
  try {
    const { data } = await apiClient.get(`/shop/products/${productId}`);
    const normalized = normalizeProduct(data);
    if (!normalized) {
      productError.value = 'Dettaglio prodotto non disponibile.';
      selectedProduct.value = null;
      return;
    }
    selectedProduct.value = normalized;
    synchronizeCartProduct(normalized);
    if (!products.value.some((product) => product.id === normalized.id)) {
      products.value = [...products.value, normalized];
    }
  } catch (error) {
    if (error?.response?.status === 404) {
      productError.value = 'Il prodotto selezionato non è più disponibile.';
    } else {
      productError.value = 'Impossibile caricare il prodotto selezionato.';
    }
    selectedProduct.value = null;
  } finally {
    isLoadingProduct.value = false;
  }
}

function emitNavigate(path, replace = false) {
  if (typeof props.onNavigate === 'function') {
    props.onNavigate(path, replace);
  }
  if (typeof window !== 'undefined') {
    window.scrollTo({ top: 0, behavior: 'smooth' });
  }
}

function goToShopHome() {
  emitNavigate('/shop');
}

function goToCheckout() {
  emitNavigate('/shop/checkout');
}

function viewProduct(productId) {
  if (productId) {
    emitNavigate(`/shop/product/${productId}`);
  }
}

function selectedProductImage(productId) {
  const product = products.value.find((item) => item.id === productId);
  return product?.imageUrl ?? '';
}

function addToCart(product, quantity = 1) {
  const normalized = normalizeProduct(product);
  if (!normalized) {
    return;
  }
  const amount = Number.isFinite(quantity) ? Math.max(1, Math.trunc(quantity)) : 1;
  const existingIndex = cartItems.value.findIndex((item) => item.product.id === normalized.id);
  if (existingIndex >= 0) {
    cartItems.value[existingIndex].quantity += amount;
    cartItems.value[existingIndex].product = normalized;
  } else {
    cartItems.value.push({ product: normalized, quantity: amount });
  }
}

function updateCartQuantity(productId, quantity) {
  const index = cartItems.value.findIndex((item) => item.product.id === productId);
  if (index === -1) {
    return;
  }
  const sanitized = Number.isFinite(quantity) ? Math.trunc(quantity) : 0;
  if (sanitized <= 0) {
    cartItems.value.splice(index, 1);
    return;
  }
  cartItems.value[index].quantity = sanitized;
}

function incrementCart(productId) {
  const item = cartItems.value.find((entry) => entry.product.id === productId);
  if (!item) {
    return;
  }
  updateCartQuantity(productId, item.quantity + 1);
}

function decrementCart(productId) {
  const item = cartItems.value.find((entry) => entry.product.id === productId);
  if (!item) {
    return;
  }
  updateCartQuantity(productId, item.quantity - 1);
}

function removeCartItem(productId) {
  const index = cartItems.value.findIndex((entry) => entry.product.id === productId);
  if (index >= 0) {
    cartItems.value.splice(index, 1);
  }
}

function handleQuantityInput(productId, event) {
  const value = Number.parseInt(event.target.value, 10);
  if (Number.isNaN(value)) {
    return;
  }
  updateCartQuantity(productId, value);
}

async function submitOrder() {
  if (isCheckoutDisabled.value) {
    checkoutError.value = 'Completa i dati richiesti per procedere al checkout.';
    return;
  }

  isSubmittingOrder.value = true;
  checkoutError.value = '';
  try {
    const payload = {
      customer_name: checkoutForm.name.trim(),
      customer_email: checkoutForm.email.trim(),
      customer_notes: checkoutForm.notes.trim(),
      items: cartItems.value.map((item) => ({
        product_id: item.product.id,
        quantity: item.quantity,
      })),
    };
    const { data } = await apiClient.post('/shop/checkout', payload);
    const normalizedOrder = normalizeOrder(data?.order);
    if (normalizedOrder) {
      lastOrder.value = normalizedOrder;
      successDetails.value = normalizedOrder;
    } else {
      successDetails.value = null;
    }
    cartItems.value = [];
    checkoutForm.name = '';
    checkoutForm.email = '';
    checkoutForm.notes = '';
    const orderId = normalizedOrder?.id;
    const target = orderId ? `/shop/checkout/success?order=${encodeURIComponent(orderId)}` : '/shop/checkout/success';
    emitNavigate(target, true);
  } catch (error) {
    const message = error?.response?.data?.message || 'Impossibile completare il checkout, riprova più tardi.';
    checkoutError.value = message;
  } finally {
    isSubmittingOrder.value = false;
  }
}

watch(
  () => routeInfo.value.name,
  (name) => {
    if ((name === 'list' || name === 'detail') && products.value.length === 0 && !isLoadingProducts.value) {
      fetchProducts();
    }
    if (name !== 'checkout') {
      checkoutError.value = '';
      isSubmittingOrder.value = false;
    }
    if (name === 'success' && !successDetails.value && lastOrder.value) {
      successDetails.value = lastOrder.value;
    }
    if (name !== 'success' && !lastOrder.value) {
      successDetails.value = null;
    }
  },
  { immediate: true }
);

watch(
  currentProductId,
  (productId) => {
    if (!productId) {
      selectedProduct.value = null;
      productError.value = '';
      return;
    }
    loadProduct(productId);
  },
  { immediate: true }
);

watch(products, (list) => {
  if (!currentProductId.value) {
    return;
  }
  const match = list.find((product) => product.id === currentProductId.value);
  if (match) {
    selectedProduct.value = match;
  }
});

watch(
  () => props.currentSearch,
  () => {
    if (routeInfo.value.name === 'success' && !successDetails.value && lastOrder.value) {
      successDetails.value = lastOrder.value;
    }
  }
);

onMounted(() => {
  fetchProducts();
  if (routeInfo.value.name === 'detail' && currentProductId.value) {
    loadProduct(currentProductId.value);
  }
});
</script>

<style scoped>
.shop {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  background: radial-gradient(circle at top, #1f2937 0%, #0f172a 55%, #020617 100%);
  color: #e2e8f0;
}

.shop-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 24px 32px;
  background: rgba(2, 6, 23, 0.85);
  border-bottom: 1px solid rgba(148, 163, 184, 0.2);
  backdrop-filter: blur(10px);
  position: sticky;
  top: 0;
  z-index: 20;
}

.brand {
  display: flex;
  align-items: center;
  gap: 12px;
  cursor: pointer;
  font-weight: 600;
}

.brand-mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: 14px;
  background: linear-gradient(135deg, #38bdf8, #22d3ee);
  color: #0f172a;
  font-weight: 700;
}

.brand-name {
  font-size: 1.25rem;
  letter-spacing: 0.05em;
}

.shop-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.nav-btn {
  padding: 10px 18px;
  border-radius: 999px;
  border: 1px solid rgba(148, 163, 184, 0.3);
  background: rgba(15, 23, 42, 0.65);
  color: inherit;
  transition: all 0.2s ease;
}

.nav-btn:hover {
  border-color: rgba(255, 255, 255, 0.35);
  background: rgba(30, 41, 59, 0.75);
}

.nav-btn.active {
  background: linear-gradient(135deg, #38bdf8, #22d3ee);
  color: #0f172a;
  border-color: transparent;
}

.shop-main {
  flex: 1;
  padding: 40px 32px 64px;
}

.view {
  max-width: 1080px;
  margin: 0 auto;
}

.intro {
  text-align: center;
  margin-bottom: 40px;
}

.intro h1 {
  font-size: 2.3rem;
  margin-bottom: 12px;
}

.intro p {
  max-width: 620px;
  margin: 0 auto 24px;
  color: rgba(226, 232, 240, 0.75);
  line-height: 1.6;
}

.intro-actions {
  display: flex;
  justify-content: center;
}

.btn {
  border: none;
  border-radius: 14px;
  padding: 12px 20px;
  font-weight: 600;
  cursor: pointer;
  transition: transform 0.15s ease, box-shadow 0.15s ease, background 0.15s ease;
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-primary {
  background: linear-gradient(135deg, #38bdf8, #22d3ee);
  color: #0f172a;
  box-shadow: 0 14px 28px rgba(8, 47, 73, 0.35);
}

.btn-primary:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 18px 32px rgba(8, 47, 73, 0.45);
}

.btn-secondary {
  background: rgba(148, 163, 184, 0.25);
  color: inherit;
  border: 1px solid rgba(148, 163, 184, 0.35);
}

.btn-outline {
  background: transparent;
  border: 1px solid rgba(148, 163, 184, 0.35);
  color: inherit;
}

.message {
  padding: 20px;
  border-radius: 16px;
  background: rgba(15, 23, 42, 0.75);
  border: 1px solid rgba(148, 163, 184, 0.2);
  text-align: center;
  color: rgba(226, 232, 240, 0.85);
}

.message-error {
  color: #fca5a5;
  border-color: rgba(248, 113, 113, 0.35);
}

.product-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
  gap: 24px;
}

.product-card {
  background: rgba(15, 23, 42, 0.75);
  border-radius: 22px;
  overflow: hidden;
  border: 1px solid rgba(148, 163, 184, 0.18);
  display: flex;
  flex-direction: column;
  cursor: pointer;
  transition: transform 0.2s ease, border-color 0.2s ease, box-shadow 0.2s ease;
}

.product-card:hover {
  transform: translateY(-6px);
  border-color: rgba(56, 189, 248, 0.6);
  box-shadow: 0 18px 36px rgba(8, 47, 73, 0.35);
}

.product-image {
  width: 100%;
  aspect-ratio: 4 / 5;
  background: rgba(30, 41, 59, 0.6);
}

.product-image img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.product-content {
  padding: 20px;
  flex: 1;
}

.product-content h3 {
  font-size: 1.1rem;
  margin-bottom: 8px;
}

.product-content p {
  color: rgba(226, 232, 240, 0.75);
  font-size: 0.95rem;
  line-height: 1.5;
}

.product-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px 20px;
  gap: 12px;
}

.product-actions {
  display: flex;
  gap: 8px;
}

.link {
  background: none;
  border: none;
  color: rgba(148, 163, 184, 0.85);
  cursor: pointer;
  padding: 0;
}

.link:hover {
  color: #f8fafc;
}

.detail-card {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 32px;
  align-items: center;
}

.detail-image {
  border-radius: 24px;
  overflow: hidden;
  border: 1px solid rgba(148, 163, 184, 0.2);
  background: rgba(15, 23, 42, 0.6);
}

.detail-image img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.detail-price {
  font-size: 1.4rem;
  font-weight: 600;
  margin-bottom: 12px;
}

.detail-description {
  color: rgba(226, 232, 240, 0.8);
  line-height: 1.7;
  margin-bottom: 24px;
}

.detail-actions {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.checkout-columns {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 32px;
}

.card {
  background: rgba(15, 23, 42, 0.85);
  border-radius: 24px;
  padding: 28px;
  border: 1px solid rgba(148, 163, 184, 0.18);
  box-shadow: 0 18px 40px rgba(8, 47, 73, 0.25);
}

.checkout-form {
  display: flex;
  flex-direction: column;
  gap: 18px;
  margin-top: 24px;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.field span {
  font-size: 0.9rem;
  color: rgba(226, 232, 240, 0.7);
}

.field input,
.field textarea {
  width: 100%;
  border-radius: 14px;
  border: 1px solid rgba(148, 163, 184, 0.25);
  background: rgba(15, 23, 42, 0.6);
  color: inherit;
  padding: 12px 14px;
}

.field input:focus,
.field textarea:focus {
  outline: none;
  border-color: rgba(56, 189, 248, 0.8);
  box-shadow: 0 0 0 3px rgba(56, 189, 248, 0.18);
}

.form-error {
  color: #fca5a5;
  font-size: 0.95rem;
}

.summary-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.summary-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
  margin: 24px 0;
  max-height: 320px;
  overflow-y: auto;
  padding-right: 6px;
}

.summary-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  background: rgba(2, 6, 23, 0.6);
  border-radius: 18px;
  padding: 12px 16px;
}

.summary-info {
  display: flex;
  align-items: center;
  gap: 14px;
}

.summary-info img {
  width: 60px;
  height: 60px;
  object-fit: cover;
  border-radius: 14px;
  border: 1px solid rgba(148, 163, 184, 0.2);
}

.summary-controls {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 8px;
}

.quantity {
  display: inline-flex;
  align-items: center;
  border: 1px solid rgba(148, 163, 184, 0.3);
  border-radius: 12px;
  overflow: hidden;
}

.quantity button {
  width: 32px;
  height: 32px;
  background: rgba(15, 23, 42, 0.7);
  color: inherit;
  border: none;
  cursor: pointer;
}

.quantity input {
  width: 48px;
  border: none;
  background: transparent;
  text-align: center;
  color: inherit;
}

.summary-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.success-card {
  text-align: center;
  align-items: center;
  gap: 16px;
}

.success-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 56px;
  border-radius: 50%;
  background: linear-gradient(135deg, #22d3ee, #38bdf8);
  color: #0f172a;
  font-size: 1.6rem;
  margin-bottom: 12px;
}

.success-summary {
  margin: 24px 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.success-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  background: rgba(2, 6, 23, 0.55);
  border-radius: 16px;
  padding: 12px 16px;
}

.success-item-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.success-item-info img {
  width: 52px;
  height: 52px;
  object-fit: cover;
  border-radius: 14px;
  border: 1px solid rgba(148, 163, 184, 0.2);
}

.success-actions {
  display: flex;
  justify-content: center;
  gap: 12px;
  flex-wrap: wrap;
  margin-top: 12px;
}

@media (max-width: 768px) {
  .shop-main {
    padding: 32px 20px 48px;
  }

  .product-grid {
    grid-template-columns: repeat(auto-fill, minmax(210px, 1fr));
  }

  .checkout-columns {
    gap: 20px;
  }
}
</style>
