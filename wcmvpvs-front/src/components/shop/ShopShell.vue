<template>
  <div class="shop">
    <header class="shop-header">
      <div class="brand" @click="goToShopHome">
        <span class="brand-mark">WC</span>
        <span class="brand-name">Wearing Cash</span>
      </div>
      <div class="header-right">
        <transition name="fade-slide">
          <div v-if="cartFeedbackVisible" class="cart-feedback">{{ cartFeedbackMessage }}</div>
        </transition>
        <nav class="shop-actions">
          <button
            type="button"
            class="nav-link"
            :class="{ active: routeInfo.name === 'list' }"
            @click="goToShopHome"
          >
            Shop
          </button>
          <button
            type="button"
            class="nav-link"
            :class="{ active: routeInfo.name === 'checkout', pulse: cartPulse }"
            @click="goToCheckout"
          >
            Carrello
            <span v-if="cartCount" class="cart-count">{{ cartCount }}</span>
            <span v-if="cartTotalCents" class="cart-total">{{ cartTotalFormatted }}</span>
          </button>
        </nav>
      </div>
    </header>

    <main class="shop-main">
      <section v-if="routeInfo.name === 'list'" class="view view-list">
        <section class="hero">
          <div class="hero-inner">
            <p class="hero-label">Collezione 2024</p>
            <h1 class="hero-title">Wear Your Grind.</h1>
            <p class="hero-subtitle">Capi essenziali, materiali premium, identità vera.</p>
            <button type="button" class="btn hero-button" @click="scrollToCollection">
              Scopri la collezione
            </button>
          </div>
        </section>
        <div class="collection-header" ref="collectionAnchor">
          <h2>Selezione Wearing Cash</h2>
          <p>
            Una linea pensata per la strada e per la scena: fit sicuro, tessuti pesanti e dettagli firmati Wearing
            Cash.
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
            <div class="product-body">
              <h3 class="product-name">{{ product.name }}</h3>
              <p class="product-description">{{ product.description }}</p>
            </div>
            <div class="product-bottom" @click.stop>
              <span class="product-price">{{ formatPrice(product.priceCents) }}</span>
              <button type="button" class="btn btn-add" @click="addToCart(product)">Aggiungi</button>
            </div>
            <transition name="fade">
              <span v-if="lastAddedProductId === product.id" class="product-feedback">Aggiunto al carrello</span>
            </transition>
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
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';
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
const cartFeedbackMessage = ref('Aggiunto al carrello');
const cartFeedbackVisible = ref(false);
const collectionAnchor = ref(null);
let highlightTimer = null;
let pulseTimer = null;
let cartFeedbackTimer = null;

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

  cartFeedbackMessage.value = `${normalized.name} aggiunto al carrello`;
  cartFeedbackVisible.value = false;
  if (cartFeedbackTimer) {
    clearTimeout(cartFeedbackTimer);
  }
  nextTick(() => {
    cartFeedbackVisible.value = true;
  });
  cartFeedbackTimer = setTimeout(() => {
    cartFeedbackVisible.value = false;
    cartFeedbackTimer = null;
  }, 1800);
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
  if (cartFeedbackTimer) {
    clearTimeout(cartFeedbackTimer);
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
  background: radial-gradient(circle at top left, rgba(249, 115, 22, 0.05), transparent 45%),
    radial-gradient(circle at 80% 10%, rgba(251, 191, 36, 0.08), transparent 55%),
    linear-gradient(180deg, #050505 0%, #0b0d11 45%, #050505 100%);
  color: #f5f7fa;
  font-family: 'Oswald', 'Inter', 'Segoe UI', sans-serif;
}

.shop::before {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(135deg, rgba(249, 115, 22, 0.08), transparent 55%);
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
  padding: 22px 48px;
  background: rgba(5, 5, 7, 0.94);
  border-bottom: 1px solid rgba(148, 163, 184, 0.18);
  backdrop-filter: blur(18px);
  position: sticky;
  top: 0;
  z-index: 20;
}

.brand {
  display: flex;
  align-items: center;
  gap: 16px;
  cursor: pointer;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  font-weight: 700;
  color: #f8fafc;
}

.brand-mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border-radius: 12px;
  background: linear-gradient(140deg, #f97316 0%, #fbbf24 90%);
  color: #050505;
  font-size: 1.1rem;
  font-weight: 800;
  box-shadow: 0 16px 32px rgba(249, 115, 22, 0.35);
}

.brand-name {
  font-size: 1.25rem;
  letter-spacing: 0.16em;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 32px;
  position: relative;
}

.cart-feedback {
  background: rgba(249, 115, 22, 0.12);
  border: 1px solid rgba(249, 115, 22, 0.4);
  color: #fbbf24;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  font-size: 0.72rem;
  padding: 6px 16px;
  border-radius: 999px;
  backdrop-filter: blur(6px);
}

.shop-actions {
  display: flex;
  align-items: center;
  gap: 28px;
}

.nav-link {
  border: none;
  background: transparent;
  color: rgba(226, 232, 240, 0.88);
  text-transform: uppercase;
  letter-spacing: 0.14em;
  font-weight: 600;
  font-size: 0.95rem;
  position: relative;
  padding: 0;
  cursor: pointer;
  transition: color 0.25s ease;
}

.nav-link::after {
  content: '';
  position: absolute;
  left: 0;
  right: 0;
  bottom: -10px;
  height: 2px;
  background: linear-gradient(90deg, rgba(249, 115, 22, 0), rgba(249, 115, 22, 0.9), rgba(249, 115, 22, 0));
  transform: scaleX(0);
  transform-origin: center;
  transition: transform 0.3s ease;
}

.nav-link:hover {
  color: #fbbf24;
}

.nav-link:hover::after,
.nav-link.active::after {
  transform: scaleX(1);
}

.nav-link.active {
  color: #fbbf24;
}

.nav-link.pulse {
  animation: navPulse 1.2s ease-out;
}

.cart-count {
  margin-left: 12px;
  padding: 2px 10px;
  border-radius: 999px;
  background: rgba(249, 115, 22, 0.18);
  font-size: 0.75rem;
  letter-spacing: 0.08em;
}

.cart-total {
  margin-left: 12px;
  font-size: 0.85rem;
  letter-spacing: 0.08em;
  color: rgba(226, 232, 240, 0.64);
}

.shop-main {
  flex: 1;
  padding: 64px 48px 96px;
}

.view {
  max-width: 1240px;
  margin: 0 auto;
}

.hero {
  position: relative;
  overflow: hidden;
  border-radius: 36px;
  padding: 128px 64px;
  margin-bottom: 72px;
  background: linear-gradient(135deg, rgba(9, 10, 14, 0.96) 0%, rgba(17, 18, 22, 0.88) 45%, rgba(0, 0, 0, 0.92) 100%);
  border: 1px solid rgba(148, 163, 184, 0.18);
  box-shadow: 0 60px 120px rgba(0, 0, 0, 0.6);
}

.hero::before {
  content: '';
  position: absolute;
  inset: -25%;
  background: radial-gradient(circle at 20% 20%, rgba(249, 115, 22, 0.22), transparent 65%);
  opacity: 0.8;
  pointer-events: none;
}

.hero-inner {
  position: relative;
  z-index: 1;
  max-width: 520px;
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.hero-label {
  text-transform: uppercase;
  letter-spacing: 0.26em;
  font-size: 0.8rem;
  color: rgba(248, 250, 252, 0.6);
}

.hero-title {
  font-size: clamp(3rem, 6vw, 4.6rem);
  text-transform: uppercase;
  letter-spacing: 0.12em;
  color: #f8fafc;
  line-height: 1.05;
}

.hero-subtitle {
  font-size: 1.15rem;
  color: rgba(226, 232, 240, 0.78);
  max-width: 360px;
}

.hero-button {
  align-self: flex-start;
  margin-top: 18px;
}

.collection-header {
  max-width: 620px;
  display: flex;
  flex-direction: column;
  gap: 14px;
  color: rgba(226, 232, 240, 0.78);
}

.collection-header h2 {
  font-size: 2.2rem;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  color: #f8fafc;
}

.collection-header p {
  line-height: 1.7;
}

.product-grid {
  display: grid;
  gap: 32px;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
}

.product-card {
  position: relative;
  display: flex;
  flex-direction: column;
  background: rgba(7, 8, 10, 0.94);
  border-radius: 24px;
  overflow: hidden;
  border: 1px solid rgba(148, 163, 184, 0.14);
  box-shadow: 0 30px 60px rgba(0, 0, 0, 0.58);
  transition: transform 0.4s ease, box-shadow 0.4s ease, border-color 0.4s ease;
  cursor: pointer;
}

.product-card::after {
  content: '';
  position: absolute;
  inset: -20% -40% auto;
  height: 60%;
  background: radial-gradient(circle at top right, rgba(249, 115, 22, 0.22), transparent 65%);
  opacity: 0;
  transition: opacity 0.4s ease;
  pointer-events: none;
}

.product-card:hover {
  transform: translateY(-8px);
  box-shadow: 0 44px 84px rgba(0, 0, 0, 0.7);
  border-color: rgba(249, 115, 22, 0.38);
}

.product-card:hover::after {
  opacity: 0.7;
}

.product-card--highlight {
  border-color: rgba(249, 115, 22, 0.55);
  box-shadow: 0 44px 90px rgba(249, 115, 22, 0.2);
}

.product-image {
  position: relative;
  padding-top: 122%;
  overflow: hidden;
  background: linear-gradient(135deg, rgba(24, 24, 27, 0.9), rgba(9, 9, 11, 0.94));
}

.product-image img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 0.5s ease;
}

.product-card:hover .product-image img {
  transform: scale(1.06);
}

.product-body {
  padding: 28px 28px 18px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.product-name {
  font-size: 1.6rem;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  color: #f8fafc;
  line-height: 1.2;
}

.product-description {
  color: rgba(226, 232, 240, 0.74);
  line-height: 1.6;
  font-size: 0.98rem;
}

.product-bottom {
  margin-top: auto;
  padding: 0 28px 28px;
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.product-price {
  font-size: 1.4rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: #fefce8;
}

.btn {
  border: none;
  cursor: pointer;
  text-transform: uppercase;
  letter-spacing: 0.14em;
  font-weight: 600;
  padding: 16px 26px;
  border-radius: 14px;
  transition: transform 0.25s ease, box-shadow 0.25s ease, background 0.25s ease, color 0.25s ease;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-add,
.hero-button,
.btn-primary {
  background: linear-gradient(135deg, #f97316 0%, #fbbf24 100%);
  color: #050505;
  box-shadow: 0 22px 44px rgba(249, 115, 22, 0.35);
}

.btn-add:hover:not(:disabled),
.hero-button:hover,
.btn-primary:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 30px 64px rgba(249, 115, 22, 0.45);
}

.btn-secondary {
  background: rgba(15, 16, 20, 0.92);
  color: #f8fafc;
  box-shadow: inset 0 0 0 1px rgba(148, 163, 184, 0.32);
}

.btn-secondary:hover:not(:disabled) {
  background: rgba(31, 32, 38, 0.92);
}

.btn-outline {
  background: transparent;
  color: rgba(248, 250, 252, 0.86);
  box-shadow: inset 0 0 0 1px rgba(248, 250, 252, 0.4);
}

.btn-outline:hover:not(:disabled) {
  background: rgba(248, 250, 252, 0.92);
  color: #050505;
}

.product-feedback {
  position: absolute;
  top: 18px;
  right: 18px;
  background: rgba(249, 115, 22, 0.18);
  border: 1px solid rgba(249, 115, 22, 0.45);
  color: #fbbf24;
  padding: 6px 14px;
  border-radius: 999px;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  font-size: 0.72rem;
  backdrop-filter: blur(6px);
}

.message {
  text-align: center;
  padding: 36px;
  border-radius: 22px;
  background: rgba(11, 13, 17, 0.82);
  color: rgba(226, 232, 240, 0.82);
  font-size: 1.05rem;
  box-shadow: 0 26px 60px rgba(0, 0, 0, 0.6);
}

.message-error {
  color: #fecaca;
  border: 1px solid rgba(239, 68, 68, 0.45);
  background: rgba(69, 10, 10, 0.6);
}

.view-detail .link {
  margin-bottom: 32px;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: rgba(248, 250, 252, 0.7);
}

.detail-card {
  display: grid;
  grid-template-columns: minmax(0, 1.05fr) minmax(0, 1fr);
  gap: 48px;
  padding: 52px;
  border-radius: 32px;
  background: rgba(11, 13, 17, 0.86);
  border: 1px solid rgba(148, 163, 184, 0.2);
  box-shadow: 0 52px 100px rgba(0, 0, 0, 0.62);
}

.detail-image {
  border-radius: 26px;
  overflow: hidden;
  background: rgba(17, 24, 39, 0.66);
}

.detail-image img {
  width: 100%;
  display: block;
  object-fit: cover;
}

.detail-info {
  display: flex;
  flex-direction: column;
  gap: 20px;
  color: rgba(226, 232, 240, 0.82);
}

.detail-info h1 {
  font-size: clamp(2.4rem, 4vw, 3rem);
  text-transform: uppercase;
  letter-spacing: 0.12em;
  color: #f8fafc;
}

.detail-price {
  font-size: 1.9rem;
  font-weight: 700;
  color: #fbbf24;
}

.detail-description {
  line-height: 1.7;
}

.detail-actions {
  display: flex;
  gap: 18px;
  flex-wrap: wrap;
}

.checkout-columns {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 40px;
}

.card {
  background: rgba(11, 13, 17, 0.85);
  border-radius: 30px;
  padding: 40px;
  border: 1px solid rgba(148, 163, 184, 0.22);
  box-shadow: 0 40px 90px rgba(0, 0, 0, 0.58);
}

.card h1,
.card h2 {
  text-transform: uppercase;
  letter-spacing: 0.12em;
  color: #f8fafc;
}

.checkout-subtitle {
  margin: 14px 0 28px;
  color: rgba(226, 232, 240, 0.72);
}

.checkout-form {
  display: flex;
  flex-direction: column;
  gap: 22px;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 10px;
  color: rgba(226, 232, 240, 0.78);
}

.field span {
  font-size: 0.9rem;
  text-transform: uppercase;
  letter-spacing: 0.12em;
}

.field input,
.field textarea {
  background: rgba(17, 19, 24, 0.92);
  border: 1px solid rgba(148, 163, 184, 0.26);
  border-radius: 16px;
  padding: 16px 18px;
  color: #f8fafc;
  font-size: 1rem;
  transition: border-color 0.25s ease, box-shadow 0.25s ease;
}

.field input:focus,
.field textarea:focus {
  outline: none;
  border-color: rgba(249, 115, 22, 0.5);
  box-shadow: 0 0 0 1px rgba(249, 115, 22, 0.45);
}

.form-error {
  color: #fecaca;
  font-size: 0.95rem;
}

.summary-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
  color: rgba(226, 232, 240, 0.82);
}

.summary-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.summary-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  padding: 18px;
  border-radius: 18px;
  background: rgba(15, 16, 20, 0.78);
  border: 1px solid rgba(148, 163, 184, 0.24);
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
  border-radius: 12px;
}

.summary-name {
  font-weight: 600;
  color: #f8fafc;
}

.summary-price {
  color: rgba(226, 232, 240, 0.72);
}

.summary-controls {
  display: flex;
  align-items: center;
  gap: 12px;
}

.quantity {
  display: inline-flex;
  align-items: center;
  border-radius: 999px;
  background: rgba(5, 7, 12, 0.9);
  border: 1px solid rgba(148, 163, 184, 0.26);
}

.quantity button {
  background: none;
  border: none;
  color: rgba(248, 250, 252, 0.78);
  font-size: 1.1rem;
  width: 36px;
  height: 36px;
  cursor: pointer;
}

.quantity input {
  width: 50px;
  background: transparent;
  border: none;
  text-align: center;
  color: #f8fafc;
  font-weight: 600;
}

.summary-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 28px;
  padding-top: 24px;
  border-top: 1px solid rgba(148, 163, 184, 0.2);
}

.summary-footer strong {
  font-size: 1.25rem;
  color: #fbbf24;
}

.success-card {
  align-items: center;
  text-align: center;
  gap: 20px;
}

.success-icon {
  width: 72px;
  height: 72px;
  border-radius: 50%;
  background: rgba(34, 197, 94, 0.2);
  color: rgba(134, 239, 172, 0.95);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 2.2rem;
}

.success-text {
  color: rgba(226, 232, 240, 0.78);
}

.success-summary {
  margin-top: 32px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  width: 100%;
}

.success-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 18px;
  border-radius: 18px;
  background: rgba(15, 16, 20, 0.78);
  border: 1px solid rgba(148, 163, 184, 0.24);
}

.success-item-info {
  display: flex;
  align-items: center;
  gap: 16px;
}

.success-item-info img {
  width: 58px;
  height: 58px;
  object-fit: cover;
  border-radius: 12px;
}

.success-total {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid rgba(148, 163, 184, 0.24);
}

.success-total strong {
  color: #fbbf24;
}

.success-actions {
  margin-top: 32px;
  display: flex;
  gap: 16px;
  justify-content: center;
  flex-wrap: wrap;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

.fade-slide-enter-active,
.fade-slide-leave-active {
  transition: all 0.35s ease;
}

.fade-slide-enter-from,
.fade-slide-leave-to {
  opacity: 0;
  transform: translateY(-6px);
}

.grid-fade-enter-active,
.grid-fade-leave-active {
  transition: all 0.4s ease;
}

.grid-fade-enter-from,
.grid-fade-leave-to {
  opacity: 0;
  transform: translateY(24px);
}

.list-fade-enter-active,
.list-fade-leave-active {
  transition: all 0.3s ease;
}

.list-fade-enter-from,
.list-fade-leave-to {
  opacity: 0;
  transform: translateY(12px);
}

@keyframes navPulse {
  0% {
    transform: translateY(0);
  }
  40% {
    transform: translateY(-2px);
  }
  100% {
    transform: translateY(0);
  }
}

@media (min-width: 1024px) {
  .product-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (min-width: 1320px) {
  .product-grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}

@media (max-width: 1024px) {
  .shop-header {
    padding: 20px 32px;
  }

  .hero {
    padding: 96px 40px;
  }

  .detail-card {
    grid-template-columns: 1fr;
    padding: 40px;
  }

  .checkout-columns {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .shop-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 18px;
  }

  .header-right {
    width: 100%;
    justify-content: space-between;
  }

  .shop-main {
    padding: 40px 24px 72px;
  }

  .hero {
    border-radius: 28px;
    padding: 72px 28px;
    margin-bottom: 56px;
  }

  .hero-inner {
    max-width: none;
  }

  .collection-header {
    max-width: none;
  }

  .product-grid {
    gap: 24px;
  }

  .product-body {
    padding: 24px 22px 16px;
  }

  .product-bottom {
    padding: 0 22px 22px;
  }

  .card {
    padding: 32px;
  }
}

@media (max-width: 640px) {
  .header-right {
    flex-direction: column;
    align-items: flex-start;
    gap: 16px;
  }

  .cart-feedback {
    align-self: flex-end;
  }

  .shop-actions {
    width: 100%;
    justify-content: space-between;
  }

  .collection-header {
    text-align: left;
  }

  .collection-header h2 {
    font-size: 1.8rem;
  }

  .product-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .product-name {
    font-size: 1.3rem;
  }

  .summary-footer,
  .success-actions {
    flex-direction: column;
    gap: 18px;
    align-items: stretch;
  }
}

@media (max-width: 480px) {
  .product-grid {
    grid-template-columns: 1fr;
  }
}
</style>

