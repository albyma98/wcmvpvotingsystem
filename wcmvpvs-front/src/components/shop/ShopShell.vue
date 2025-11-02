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
          :class="{ active: routeInfo.name === 'checkout', pulse: cartPulse }"
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
        <div class="hero">
          <div class="hero-frame">
            <div class="hero-logo">WC</div>
            <p class="hero-tagline">Wear Your Grind.</p>
            <p class="hero-subtitle">
              Streetwear motivazionale con anima sportiva. Capi premium pensati per chi non smette mai di inseguire i
              propri obiettivi.
            </p>
            <div class="hero-actions">
              <button
                type="button"
                class="btn btn-primary"
                :disabled="cartItems.length === 0"
                @click="goToCheckout"
              >
                Vai al checkout
              </button>
              <button type="button" class="btn btn-outline" @click="scrollToCollection">
                Esplora la collezione
              </button>
            </div>
          </div>
        </div>
        <div class="collection-header" ref="collectionAnchor">
          <h2>Collezione Wearing Cash</h2>
          <p>
            Una selezione di capi iconici firmati Wearing Cash. Silhouette pulite, materiali premium e dettagli pensati
            per esprimere energia e determinazione.
          </p>
        </div>
        <div v-if="productsError" class="message message-error">{{ productsError }}</div>
        <div v-else-if="isLoadingProducts" class="message">Caricamento dei prodotti…</div>
        <div v-else-if="products.length === 0" class="message">
          Non ci sono prodotti disponibili al momento. Torna più tardi!
        </div>
        <TransitionGroup v-else name="grid-fade" tag="div" class="product-grid">
          <article
            v-for="product in products"
            :key="product.id"
            class="product-card"
            :class="{ 'product-card--highlight': lastAddedProductId === product.id }"
            @click="viewProduct(product.id)"
          >
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
        </TransitionGroup>
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
            <TransitionGroup v-else tag="ul" name="list-fade" class="summary-list">
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
            </TransitionGroup>
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
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';
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

const lastAddedProductId = ref(null);
const cartPulse = ref(false);
const collectionAnchor = ref(null);
let highlightTimer = null;
let pulseTimer = null;

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

function scrollToCollection() {
  if (collectionAnchor.value) {
    collectionAnchor.value.scrollIntoView({ behavior: 'smooth' });
  }
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

  lastAddedProductId.value = normalized.id;
  if (highlightTimer) {
    clearTimeout(highlightTimer);
  }
  highlightTimer = setTimeout(() => {
    lastAddedProductId.value = null;
  }, 1400);

  cartPulse.value = true;
  if (pulseTimer) {
    clearTimeout(pulseTimer);
  }
  pulseTimer = setTimeout(() => {
    cartPulse.value = false;
  }, 1200);
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

onBeforeUnmount(() => {
  if (highlightTimer) {
    clearTimeout(highlightTimer);
  }
  if (pulseTimer) {
    clearTimeout(pulseTimer);
  }
});

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
  position: relative;
  background: linear-gradient(160deg, #050505 0%, #111827 50%, #000000 100%);
  color: #f8fafc;
  font-family: 'Montserrat', 'Nexa', 'Inter', sans-serif;
}

.shop::before {
  content: '';
  position: absolute;
  inset: 0;
  background:
    radial-gradient(circle at 20% -10%, rgba(239, 68, 68, 0.22) 0%, transparent 55%),
    radial-gradient(circle at 85% 0%, rgba(212, 175, 55, 0.3) 0%, transparent 52%);
  pointer-events: none;
  mix-blend-mode: screen;
}

.shop-header,
.shop-main {
  position: relative;
  z-index: 1;
}

.shop-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 22px 32px;
  background: rgba(5, 5, 5, 0.82);
  border-bottom: 1px solid rgba(148, 163, 184, 0.22);
  backdrop-filter: blur(14px);
  position: sticky;
  top: 0;
  z-index: 20;
}

.brand {
  display: flex;
  align-items: center;
  gap: 14px;
  cursor: pointer;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.brand-mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border-radius: 14px;
  background: linear-gradient(135deg, #d4af37 0%, #ef4444 100%);
  color: #0b0b0b;
  font-weight: 800;
  font-size: 1.05rem;
  box-shadow: 0 14px 28px rgba(212, 175, 55, 0.35);
}

.brand-name {
  font-size: 1.2rem;
  color: #f8fafc;
}

.shop-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.nav-btn {
  padding: 10px 20px;
  border-radius: 999px;
  border: 1px solid rgba(248, 250, 252, 0.18);
  background: rgba(17, 24, 39, 0.72);
  color: inherit;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  font-weight: 600;
  transition: all 0.25s ease;
}

.nav-btn:hover {
  border-color: rgba(212, 175, 55, 0.55);
  color: #fefce8;
  box-shadow: 0 0 18px rgba(212, 175, 55, 0.25);
}

.nav-btn.active {
  background: linear-gradient(135deg, #d4af37 0%, #f59e0b 45%, #ef4444 100%);
  color: #0b0b0b;
  border-color: transparent;
  box-shadow: 0 18px 32px rgba(239, 68, 68, 0.28);
}

.nav-btn.pulse {
  animation: pulseGlow 1s ease-out;
}

.shop-main {
  flex: 1;
  padding: 48px 32px 72px;
}

.view {
  max-width: 1160px;
  margin: 0 auto;
}

.hero {
  position: relative;
  overflow: hidden;
  border-radius: 32px;
  padding: 96px 24px;
  margin-bottom: 64px;
  background: linear-gradient(145deg, rgba(13, 13, 16, 0.92) 0%, rgba(17, 24, 39, 0.88) 60%, rgba(0, 0, 0, 0.92) 100%);
  border: 1px solid rgba(212, 175, 55, 0.24);
  box-shadow: 0 48px 120px rgba(0, 0, 0, 0.55);
}

.hero::before,
.hero::after {
  content: '';
  position: absolute;
  inset: 0;
  pointer-events: none;
}

.hero::before {
  background: radial-gradient(circle at 30% 20%, rgba(212, 175, 55, 0.32) 0%, transparent 58%);
  opacity: 0.85;
}

.hero::after {
  background: radial-gradient(circle at 72% 35%, rgba(239, 68, 68, 0.26) 0%, transparent 62%);
}

.hero-frame {
  position: relative;
  z-index: 1;
  max-width: 560px;
  margin: 0 auto;
  text-align: center;
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.hero-logo {
  margin: 0 auto;
  width: 90px;
  height: 90px;
  border-radius: 24px;
  display: grid;
  place-items: center;
  font-size: 1.8rem;
  font-weight: 800;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  background: linear-gradient(135deg, rgba(212, 175, 55, 0.95), rgba(239, 68, 68, 0.9));
  color: #09090b;
  box-shadow: 0 30px 60px rgba(212, 175, 55, 0.38);
}

.hero-tagline {
  font-size: clamp(2.8rem, 4vw, 3.6rem);
  font-weight: 800;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: #fefce8;
}

.hero-subtitle {
  color: rgba(248, 250, 252, 0.78);
  line-height: 1.7;
  font-size: 1.05rem;
}

.hero-actions {
  margin-top: 8px;
  display: flex;
  justify-content: center;
  gap: 16px;
  flex-wrap: wrap;
}

.collection-header {
  margin-bottom: 32px;
  max-width: 720px;
}

.collection-header h2 {
  font-size: 1.9rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  margin-bottom: 12px;
}

.collection-header p {
  color: rgba(226, 232, 240, 0.82);
  line-height: 1.7;
}

.message {
  padding: 20px;
  border-radius: 20px;
  background: rgba(15, 15, 18, 0.78);
  border: 1px solid rgba(148, 163, 184, 0.25);
  text-align: center;
  color: rgba(226, 232, 240, 0.9);
}

.message-error {
  color: #fca5a5;
  border-color: rgba(239, 68, 68, 0.45);
  box-shadow: 0 0 25px rgba(239, 68, 68, 0.15);
}

.product-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 28px;
}

.product-card {
  position: relative;
  background: rgba(13, 13, 16, 0.88);
  border-radius: 26px;
  overflow: hidden;
  border: 1px solid rgba(212, 175, 55, 0.1);
  display: flex;
  flex-direction: column;
  cursor: pointer;
  transition: transform 0.25s ease, border-color 0.25s ease, box-shadow 0.25s ease;
}

.product-card::after {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(180deg, transparent 45%, rgba(212, 175, 55, 0.08));
  opacity: 0;
  transition: opacity 0.3s ease;
  pointer-events: none;
}

.product-card:hover {
  transform: translateY(-8px);
  border-color: rgba(212, 175, 55, 0.45);
  box-shadow: 0 32px 56px rgba(0, 0, 0, 0.45);
}

.product-card:hover::after {
  opacity: 1;
}

.product-card--highlight {
  border-color: rgba(212, 175, 55, 0.7);
  box-shadow: 0 0 0 2px rgba(212, 175, 55, 0.6), 0 24px 48px rgba(212, 175, 55, 0.25);
  animation: highlightFlash 1s ease-out;
}

.product-image {
  width: 100%;
  aspect-ratio: 4 / 5;
  background: rgba(24, 24, 27, 0.75);
}

.product-image img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.product-content {
  padding: 22px;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.product-content h3 {
  font-size: 1.15rem;
  font-weight: 700;
}

.product-content p {
  color: rgba(226, 232, 240, 0.7);
  font-size: 0.97rem;
  line-height: 1.55;
}

.product-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 22px 24px;
  gap: 14px;
}

.product-actions {
  display: flex;
  gap: 10px;
}

.btn {
  position: relative;
  border: none;
  border-radius: 16px;
  padding: 12px 22px;
  font-weight: 700;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  cursor: pointer;
  transition: transform 0.18s ease, box-shadow 0.18s ease, background 0.18s ease;
}

.btn:disabled {
  opacity: 0.55;
  cursor: not-allowed;
  transform: none;
  box-shadow: none;
}

.btn-primary {
  background: linear-gradient(135deg, #d4af37 0%, #f59e0b 45%, #ef4444 100%);
  color: #0f0f0f;
  box-shadow: 0 20px 35px rgba(239, 68, 68, 0.35);
}

.btn-primary:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 26px 44px rgba(239, 68, 68, 0.4);
}

.btn-secondary {
  background: rgba(255, 255, 255, 0.08);
  color: #f8fafc;
  border: 1px solid rgba(255, 255, 255, 0.12);
}

.btn-secondary:hover:not(:disabled) {
  transform: translateY(-2px);
  border-color: rgba(212, 175, 55, 0.45);
}

.btn-outline {
  background: transparent;
  border: 1px solid rgba(255, 255, 255, 0.22);
  color: #f8fafc;
}

.btn-outline:hover:not(:disabled) {
  transform: translateY(-2px);
  border-color: rgba(212, 175, 55, 0.6);
  box-shadow: 0 18px 36px rgba(212, 175, 55, 0.22);
}

.product-price,
.summary-price {
  font-weight: 700;
  color: #fefce8;
}

.link {
  background: none;
  border: none;
  color: rgba(226, 232, 240, 0.7);
  cursor: pointer;
  padding: 0;
  transition: color 0.2s ease;
}

.link:hover {
  color: #fde68a;
}

.detail-card {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 36px;
  align-items: center;
}

.detail-image {
  border-radius: 28px;
  overflow: hidden;
  border: 1px solid rgba(212, 175, 55, 0.2);
  background: rgba(15, 15, 18, 0.65);
  box-shadow: 0 32px 64px rgba(0, 0, 0, 0.45);
}

.detail-price {
  font-size: 1.6rem;
  font-weight: 700;
  margin-bottom: 16px;
  color: #fefce8;
}

.detail-description {
  color: rgba(226, 232, 240, 0.78);
  line-height: 1.8;
  margin-bottom: 28px;
}

.detail-actions {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.checkout-columns {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(330px, 1fr));
  gap: 36px;
}

.card {
  background: rgba(10, 10, 12, 0.88);
  border-radius: 28px;
  padding: 32px;
  border: 1px solid rgba(148, 163, 184, 0.2);
  box-shadow: 0 40px 80px rgba(0, 0, 0, 0.45);
}

.checkout-subtitle {
  color: rgba(226, 232, 240, 0.75);
  line-height: 1.6;
}

.checkout-form {
  display: flex;
  flex-direction: column;
  gap: 20px;
  margin-top: 28px;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.field span {
  font-size: 0.9rem;
  color: rgba(226, 232, 240, 0.7);
  letter-spacing: 0.03em;
}

.field input,
.field textarea {
  width: 100%;
  border-radius: 16px;
  border: 1px solid rgba(148, 163, 184, 0.28);
  background: rgba(248, 250, 252, 0.05);
  color: #f8fafc;
  padding: 13px 16px;
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}

.field input:focus,
.field textarea:focus {
  outline: none;
  border-color: rgba(212, 175, 55, 0.55);
  box-shadow: 0 0 0 4px rgba(212, 175, 55, 0.18);
}

.form-error {
  color: #f87171;
  font-size: 0.95rem;
}

.summary-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 18px;
}

.summary-list {
  display: flex;
  flex-direction: column;
  gap: 18px;
  margin: 28px 0;
  max-height: 320px;
  overflow-y: auto;
  padding-right: 6px;
}

.summary-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  background: rgba(14, 14, 18, 0.78);
  border-radius: 20px;
  padding: 14px 18px;
  border: 1px solid rgba(148, 163, 184, 0.18);
}

.summary-info {
  display: flex;
  align-items: center;
  gap: 16px;
}

.summary-info img {
  width: 64px;
  height: 64px;
  object-fit: cover;
  border-radius: 16px;
  border: 1px solid rgba(212, 175, 55, 0.2);
}

.summary-controls {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 10px;
}

.quantity {
  display: inline-flex;
  align-items: center;
  border: 1px solid rgba(148, 163, 184, 0.35);
  border-radius: 14px;
  overflow: hidden;
  background: rgba(15, 23, 42, 0.6);
}

.quantity button {
  width: 34px;
  height: 34px;
  background: transparent;
  color: #f8fafc;
  border: none;
  cursor: pointer;
  transition: background 0.2s ease;
}

.quantity button:hover:not(:disabled) {
  background: rgba(212, 175, 55, 0.25);
  color: #0b0b0b;
}

.quantity input {
  width: 54px;
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
  gap: 18px;
}

.success-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 64px;
  height: 64px;
  border-radius: 18px;
  background: linear-gradient(135deg, #d4af37 0%, #ef4444 100%);
  color: #0b0b0b;
  font-size: 1.8rem;
  margin-bottom: 14px;
  box-shadow: 0 20px 40px rgba(239, 68, 68, 0.35);
}

.success-summary {
  margin: 26px 0;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.success-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  background: rgba(14, 14, 18, 0.75);
  border-radius: 18px;
  padding: 14px 18px;
  border: 1px solid rgba(212, 175, 55, 0.16);
}

.success-item-info {
  display: flex;
  align-items: center;
  gap: 14px;
}

.success-item-info img {
  width: 56px;
  height: 56px;
  object-fit: cover;
  border-radius: 16px;
  border: 1px solid rgba(212, 175, 55, 0.25);
}

.success-code,
.success-text {
  color: rgba(226, 232, 240, 0.78);
}

.success-actions {
  display: flex;
  justify-content: center;
  gap: 14px;
  flex-wrap: wrap;
  margin-top: 14px;
}

.grid-fade-enter-active,
.grid-fade-leave-active,
.list-fade-enter-active,
.list-fade-leave-active {
  transition: all 0.3s ease;
}

.grid-fade-enter-from,
.list-fade-enter-from {
  opacity: 0;
  transform: translateY(12px);
}

.grid-fade-leave-to,
.list-fade-leave-to {
  opacity: 0;
  transform: translateY(-12px);
}

@keyframes pulseGlow {
  0% {
    box-shadow: 0 0 0 0 rgba(212, 175, 55, 0.5);
  }
  70% {
    box-shadow: 0 0 0 12px rgba(212, 175, 55, 0);
  }
  100% {
    box-shadow: 0 0 0 0 rgba(212, 175, 55, 0);
  }
}

@keyframes highlightFlash {
  0% {
    box-shadow: 0 0 0 0 rgba(212, 175, 55, 0.65);
  }
  60% {
    box-shadow: 0 0 0 12px rgba(212, 175, 55, 0);
  }
  100% {
    box-shadow: 0 24px 48px rgba(212, 175, 55, 0.2);
  }
}

@media (max-width: 992px) {
  .shop-main {
    padding: 40px 24px 64px;
  }

  .hero {
    padding: 80px 24px;
  }
}

@media (max-width: 768px) {
  .shop-header {
    flex-direction: column;
    gap: 18px;
    text-align: center;
  }

  .hero {
    margin-bottom: 48px;
  }

  .product-grid {
    grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  }

  .checkout-columns {
    gap: 24px;
  }
}

@media (max-width: 520px) {
  .shop-main {
    padding: 32px 18px 56px;
  }

  .hero {
    padding: 68px 18px;
  }

  .hero-logo {
    width: 74px;
    height: 74px;
    font-size: 1.4rem;
  }

  .collection-header h2 {
    font-size: 1.6rem;
  }
}
</style>
