<template>
  <main class="merch-shell" :style="theme">
    <header class="shop-nav">
      <a class="club-brand" :href="homePath" :aria-label="`Torna al portale di ${brandName}`">
        <img v-if="brand.logo_url" :src="brand.logo_url" alt="" />
        <span v-else class="brand-mark" aria-hidden="true">{{ brandName.slice(0, 2).toUpperCase() }}</span>
        <span>{{ brandName }}</span>
      </a>
      <nav aria-label="Navigazione shop">
        <a v-if="(shopEnabled && settings.visible) || route.page === 'shop'" :href="shopPath" :aria-current="route.page === 'shop' ? 'page' : undefined" @click="follow($event, shopPath)">Shop</a>
        <a :href="ordersPath" :aria-current="route.page === 'orders' ? 'page' : undefined" @click="follow($event, ordersPath)">I miei ordini</a>
        <button v-if="route.page === 'shop' && shopEnabled && cart.length" class="cart-toggle" type="button" @click="openCart">Carrello <span>{{ cartCount }}</span></button>
      </nav>
    </header>

    <div class="shop-container">
      <MerchOrders v-if="route.page === 'orders'" :slug="slug" :search="search" :latest-order="latestOrder" />
      <template v-else>
        <section class="shop-hero">
          <p class="eyebrow">MERCHANDISING UFFICIALE</p>
          <h1>Shop del club</h1>
          <p v-if="shopEnabled">Scopri i prodotti di {{ brandName }} e scegli come riceverli.</p>
        </section>
        <p v-if="loading" class="empty" role="status">Caricamento del catalogo…</p>
        <div v-else-if="catalogError" class="empty" role="alert">
          <p>{{ catalogError }}</p><button class="secondary" type="button" @click="loadCatalog">Riprova</button>
        </div>
        <p v-else-if="!shopEnabled" class="empty">Lo Shop di {{ brandName }} al momento è chiuso.</p>
        <template v-else>
          <div v-if="products.length" class="filters" aria-label="Filtri catalogo">
            <label class="field">Cerca un prodotto
              <input v-model.trim="filters.search" type="search" placeholder="Nome o categoria" />
            </label>
            <label v-if="categories.length" class="field">Categoria
              <select v-model="filters.category"><option value="">Tutte</option><option v-for="category in categories" :key="category">{{ category }}</option></select>
            </label>
            <label class="check"><input v-model="filters.inStock" type="checkbox" /> Solo disponibili</label>
          </div>
          <p v-if="!products.length" class="empty">Il club sta preparando i prossimi prodotti.</p>
          <p v-else-if="!filtered.length" class="empty">Nessun prodotto corrisponde alla ricerca.</p>
          <div v-else class="product-grid">
            <a v-for="item in filtered" :key="item.id" :href="productPath(item.id)" class="product-card" :data-product-id="item.id" :ref="observeProduct" @click="follow($event, productPath(item.id))">
              <div class="product-photo">
                <img v-if="item.media?.[0]?.url" :src="mediaUrl(item.media[0].url)" alt="" loading="lazy" decoding="async" />
                <span v-else class="photo-fallback" aria-hidden="true">ABX</span>
                <span v-if="item.featured" class="featured">In evidenza</span>
              </div>
              <div class="card-copy">
                <small v-if="item.category">{{ item.category }}</small>
                <h2>{{ item.name }}</h2>
                <p v-if="item.short_description">{{ item.short_description }}</p>
                <div class="card-price"><span>{{ cardPrice(item) }}</span><span v-if="!isAvailable(item)" class="muted">Esaurito</span></div>
              </div>
            </a>
          </div>
          <p v-if="route.productId && !product" class="empty" role="status">Questo prodotto non è disponibile. <a :href="shopPath" @click="follow($event, shopPath)">Torna al catalogo</a></p>
        </template>
      </template>
      <footer v-if="settings.seller_info || settings.contact_email" class="seller">
        <p v-if="settings.seller_info">{{ settings.seller_info }}</p>
        <a v-if="settings.contact_email" :href="`mailto:${settings.contact_email}`">Contatta il club</a>
      </footer>
    </div>
    <div v-if="notice" class="toast" role="status">{{ notice }}</div>

    <dialog ref="dialog" class="merch-dialog" aria-labelledby="merch-dialog-title" @cancel.prevent="dismiss" @click.self="dismiss">
      <div v-if="hasDialog" class="dialog-body">
        <button class="dialog-close" type="button" aria-label="Chiudi" :disabled="checkoutBusy" @click="dismiss">×</button>
        <section v-if="stripeFallback" class="checkout">
          <div class="dialog-heading">
            <h2 id="merch-dialog-title">Vai al pagamento sicuro</h2>
            <p>Il browser impedisce di salvare l’ordine. Conserva il collegamento privato prima di continuare.</p>
          </div>
          <label class="field">Collegamento privato<input :value="stripeFallback.link" readonly @focus="$event.target.select()" /></label>
          <a class="primary" :href="stripeFallback.url">Continua su Stripe</a>
        </section>
        <MerchCheckout v-else-if="mode" :key="mode" :slug="slug" :mode="mode" :lines="checkoutLines" :settings="settings" :delivery-options="deliveryOptions" @back="mode = ''" @created="handleOrder" @recoverable-order="rememberOrder" @busy-change="checkoutBusy = $event" />
        <section v-else-if="showCart" class="cart-view">
          <div class="dialog-heading"><h2 id="merch-dialog-title">Il tuo carrello</h2><p>Completa più articoli in una sola operazione.</p></div>
          <p v-if="!cart.length" class="empty">Il carrello è vuoto.</p>
          <template v-else>
            <ul class="cart-list">
              <li v-for="line in cartDetails" :key="merchItemKey(line)">
                <div><strong>{{ line.product?.name || 'Articolo non disponibile' }}</strong><p v-if="line.variant">{{ merchVariantLabel(line.variant) }}</p></div>
                <label v-if="line.max" class="field small-field">Quantità<input type="number" min="1" :max="line.max" :value="line.quantity" @change="updateCart(line, $event)" /></label>
                <button type="button" class="text-button" @click="removeFromCart(line)">Rimuovi</button>
              </li>
            </ul>
            <p v-if="!cartReady" class="inset">Controlla le quantità e rimuovi gli articoli esauriti per continuare.</p>
            <p v-else-if="!cartOptions.online && !cartOptions.offline" class="inset">Questi articoli hanno modalità di vendita diverse. Puoi ordinare separatamente gli articoli con la stessa modalità.</p>
            <p v-else-if="!cartOptions.pickup && !cartOptions.shipping" class="inset">Questi articoli non hanno una modalità di consegna in comune.</p>
            <div v-else class="actions">
              <button v-for="(action, index) in saleActions(cartOptions)" :key="action" :class="index ? 'secondary' : 'primary'" type="button" @click="startCheckout(action, cart)">{{ saleLabels[action] }}</button>
            </div>
          </template>
        </section>
        <section v-else-if="product" class="product-view">
          <div class="product-gallery">
            <div class="product-photo">
              <img v-if="product.media?.[photoIndex]?.url" :src="mediaUrl(product.media[photoIndex].url)" :alt="product.name" decoding="async" />
              <span v-else class="photo-fallback" aria-hidden="true">ABX</span>
            </div>
            <div v-if="product.media?.length > 1" class="thumbnails">
              <button v-for="(photo, index) in product.media" :key="photo.id" type="button" :aria-label="`Mostra foto ${index + 1}`" :aria-pressed="photoIndex === index" @click="photoIndex = index">
                <img :src="mediaUrl(photo.url)" alt="" loading="lazy" />
              </button>
            </div>
          </div>
          <div class="product-detail">
            <div class="dialog-heading">
              <p v-if="product.category" class="eyebrow">{{ product.category }}</p>
              <h2 id="merch-dialog-title">{{ product.name }}</h2>
              <p v-if="product.short_description">{{ product.short_description }}</p>
            </div>
            <div class="detail-price">
              <strong>{{ variant || !product.variants?.length ? merchMoney(merchUnitPrice(product, variant)) : cardPrice(product) }}</strong>
              <del v-if="product.promo_price_cents != null && variant?.price_cents == null && !product.variants?.length">{{ merchMoney(product.price_cents) }}</del>
              <span class="muted">{{ maxQuantity ? (variant?.on_demand || (!variant && product.on_demand) ? 'Su ordinazione' : 'Disponibile') : isAvailable(product) ? 'Scegli una variante' : 'Esaurito' }}</span>
            </div>
            <fieldset v-if="product.variants?.length" class="variant-field">
              <legend>{{ product.variants.some(option => option.size || option.color) ? 'Taglia e colore' : 'Variante' }}</legend>
              <div class="variant-list">
                <button v-for="option in product.variants" :key="option.id" type="button" :aria-pressed="variant?.id === option.id" :disabled="!merchMaxQuantity(product, option)" @click="chooseVariant(option)">
                  {{ merchVariantLabel(option) || option.sku || 'Standard' }}<small v-if="!merchMaxQuantity(product, option)">Esaurito</small>
                </button>
              </div>
            </fieldset>
            <label v-if="maxQuantity" class="field small-field">Quantità
              <select v-model.number="quantity">
                <option v-for="count in maxQuantity" :key="count" :value="count">{{ count }}</option>
              </select>
            </label>
            <div v-if="productOptions.online || productOptions.offline" class="actions">
              <button v-for="(action, index) in saleActions(productOptions)" :key="action" :class="index ? 'secondary' : 'primary'" type="button" :disabled="!productReady" @click="startCheckout(action, [selectedLine])">{{ saleLabels[action] }}</button>
            </div>
            <p v-else class="muted">La società renderà disponibile l’ordine a breve.</p>
            <button v-if="productReady && (productOptions.online || productOptions.offline)" class="text-button" type="button" @click="addToCart">Aggiungi al carrello</button>
            <details v-if="product.description">
              <summary>Dettagli prodotto</summary><p>{{ product.description }}</p>
            </details>
          </div>
        </section>
      </div>
    </dialog>
  </main>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';
import { apiClient, resolveApiUrl } from '../api';
import { normalizeOrganizationColors, organizationThemeStyle } from '../organizationTheme';
import MerchCheckout from './MerchCheckout.vue';
import MerchOrders from './MerchOrders.vue';
import { fetchMerchCatalog, merchRequestError, trackMerchEvent } from './merchApi';
import {
  fanPortalPath, guestOrderFragmentFor, merchItemKey, merchMaxQuantity, merchMoney, merchOptions,
  merchUnitPrice, merchVariantLabel, saveGuestOrder, selectedMerchProducts, stripeCheckoutUrl,
} from './shopModel';

const props = defineProps({
  slug: { type: String, required: true },
  route: { type: Object, required: true },
  search: { type: String, default: '' },
});
const emit = defineEmits(['navigate']);
const homePath = computed(() => fanPortalPath(props.slug));
const shopPath = computed(() => `${homePath.value}/shop`);
const ordersPath = computed(() => `${homePath.value}/orders`);
const productPath = id => `${shopPath.value}/${encodeURIComponent(id)}`;
const mediaUrl = url => url.startsWith('/api/') ? resolveApiUrl(url.slice(4)) : url;
const brand = ref({});
const brandName = computed(() => brand.value.name || props.slug);
const theme = computed(() => organizationThemeStyle(normalizeOrganizationColors(brand.value)));
const products = ref([]);
const settings = ref({});
const loading = ref(true);
const catalogError = ref('');
const shopEnabled = computed(() => Boolean(settings.value.enabled));
const filters = reactive({ search: '', category: '', inStock: false });
const categories = computed(() => [...new Set(products.value.map(item => item.category).filter(Boolean))].sort((a, b) => a.localeCompare(b, 'it')));
const filtered = computed(() => selectedMerchProducts(products.value, filters));
const product = computed(() => shopEnabled.value && !loading.value ? products.value.find(item => item.id === props.route.productId) : null);
const variant = ref(null);
const quantity = ref(1);
const photoIndex = ref(0);
const maxQuantity = computed(() => merchMaxQuantity(product.value, variant.value));
const selectedLine = computed(() => ({
  product_id: product.value?.id,
  ...(variant.value ? { variant_id: variant.value.id } : {}),
  quantity: quantity.value,
}));
const productOptions = computed(() => merchOptions(products.value, settings.value, [selectedLine.value]));
const productReady = computed(() => maxQuantity.value > 0 && (productOptions.value.pickup || productOptions.value.shipping));

const cart = ref([]);
const cartCount = computed(() => cart.value.reduce((sum, line) => sum + line.quantity, 0));
const cartByProduct = computed(() => {
  const quantities = new Map();
  for (const line of cart.value) quantities.set(line.product_id, (quantities.get(line.product_id) || 0) + line.quantity);
  return quantities;
});
const cartDetails = computed(() => cart.value.map(line => {
  const item = products.value.find(candidate => candidate.id === line.product_id);
  const option = item?.variants?.find(candidate => candidate.id === line.variant_id);
  const otherVariants = cartByProduct.value.get(line.product_id) - line.quantity;
  const max = Math.max(0, Math.min(merchMaxQuantity(item, option), (item?.max_quantity || 0) - otherVariants));
  return { ...line, product: item, variant: option, max };
}));
const cartOptions = computed(() => merchOptions(products.value, settings.value, cart.value));
const cartReady = computed(() => cartDetails.value.every(line => line.max >= line.quantity));
const showCart = ref(false);
const mode = ref('');
const checkoutLines = ref([]);
const deliveryOptions = computed(() => merchOptions(products.value, settings.value, checkoutLines.value));
const checkoutBusy = ref(false);
const latestOrder = ref(null);
const stripeFallback = ref(null);
const hasDialog = computed(() => props.route.page === 'shop' && (showCart.value || Boolean(product.value)));
const dialog = ref(null);
const notice = ref('');
let noticeTimer;
let catalogRequest;
let brandRequest;
let observer;
let dialogTrigger;
let catalogLoaded = false;
const impressed = new Set();
const observedCards = new WeakSet();
const saleLabels = { online: 'Acquista ora', offline: 'Prenota e paga alla società' };

function saleActions(options) {
  const preferred = settings.value.default_mode === 'offline' ? ['offline', 'online'] : ['online', 'offline'];
  return preferred.filter(action => options[action]);
}

function follow(event, path) {
  if (event.button || event.ctrlKey || event.shiftKey || event.metaKey || event.altKey) return;
  event.preventDefault();
  emit('navigate', path);
}

async function loadCatalog() {
  catalogRequest?.abort();
  catalogRequest = new AbortController();
  const { signal } = catalogRequest;
  loading.value = true;
  catalogError.value = '';
  try {
    const data = await fetchMerchCatalog(props.slug, signal);
    if (signal.aborted) return;
    settings.value = data.settings || {};
    products.value = data.products || [];
    catalogLoaded = true;
  } catch (error) {
    if (signal.aborted) return;
    if (error?.response?.status === 404) {
      settings.value = {};
      products.value = [];
      catalogLoaded = true;
    } else {
      catalogError.value = merchRequestError(error, 'Impossibile caricare il catalogo.');
    }
  } finally {
    if (!signal.aborted) loading.value = false;
  }
}

function isAvailable(item) {
  return item.variants?.length ? item.variants.some(option => merchMaxQuantity(item, option) > 0) : merchMaxQuantity(item) > 0;
}

function cardPrice(item) {
  const prices = item.variants?.length ? item.variants.map(option => merchUnitPrice(item, option)) : [merchUnitPrice(item)];
  const least = Math.min(...prices);
  return `${prices.some(price => price !== least) ? 'Da ' : ''}${merchMoney(least)}`;
}

function observeProduct(element) {
  if (!element || observedCards.has(element)) return;
  if (!observer && typeof IntersectionObserver === 'function') {
    observer = new IntersectionObserver(entries => {
      for (const entry of entries) {
        if (!entry.isIntersecting) continue;
        const id = entry.target.dataset.productId;
        observer.unobserve(entry.target);
        if (!impressed.has(id)) {
          impressed.add(id);
          trackMerchEvent(props.slug, 'merch_impression', id);
        }
      }
    }, { threshold: 0.4 });
  }
  if (observer) {
    observedCards.add(element);
    observer.observe(element);
  }
}

function chooseVariant(option) {
  if (variant.value?.id === option.id) return;
  variant.value = option;
  quantity.value = Math.min(quantity.value, merchMaxQuantity(product.value, option));
  trackMerchEvent(props.slug, 'merch_variant_selected', product.value.id, option.id);
}

function openCart() {
  showCart.value = true;
  mode.value = '';
}

function notify(message) {
  notice.value = message;
  window.clearTimeout(noticeTimer);
  noticeTimer = window.setTimeout(() => { notice.value = ''; }, 3600);
}

function addToCart() {
  const line = selectedLine.value;
  const key = merchItemKey(line);
  const previous = cart.value.find(item => merchItemKey(item) === key);
  if (!previous && cart.value.length >= 20) {
    notify('Puoi ordinare fino a 20 articoli diversi alla volta.');
    return;
  }
  const remaining = Math.min(maxQuantity.value - (previous?.quantity || 0), product.value.max_quantity - (cartByProduct.value.get(line.product_id) || 0));
  if (remaining <= 0) {
    notify('Hai raggiunto la quantità disponibile per questo articolo.');
    return;
  }
  const added = Math.min(remaining, line.quantity);
  if (previous) previous.quantity += added;
  else cart.value.push({ ...line, quantity: added });
  dismiss();
  notify(added < line.quantity ? `Aggiunte ${added} unità disponibili.` : 'Articolo aggiunto al carrello.');
}

function updateCart(line, event) {
  const entry = cart.value.find(item => merchItemKey(item) === merchItemKey(line));
  entry.quantity = Math.max(1, Math.min(line.max, Math.floor(Number(event.target.value)) || 1));
  event.target.value = entry.quantity;
}

function removeFromCart(line) {
  cart.value = cart.value.filter(item => merchItemKey(item) !== merchItemKey(line));
}

function startCheckout(nextMode, lines) {
  checkoutLines.value = lines.map(line => ({ ...line }));
  trackOutbound();
  mode.value = nextMode;
}

function dismiss() {
  if (checkoutBusy.value) return;
  mode.value = '';
  stripeFallback.value = null;
  if (showCart.value) showCart.value = false;
  if (props.route.productId) emit('navigate', shopPath.value, true);
}

function trackOutbound() {
  trackMerchEvent(props.slug, 'merch_outbound_click', checkoutLines.value[0]?.product_id);
}

function rememberOrder(result) {
  latestOrder.value = result;
  return !result.access_token || saveGuestOrder(window, props.slug, result.order.id, result.access_token);
}

function handleOrder(result) {
  if (!result?.order) return;
  const stored = rememberOrder(result);
  const order = result.order;
  const access = result.access_token;
  if (showCart.value) cart.value = [];
  const url = stripeCheckoutUrl(result.checkout_url);
  if (url) {
    if (!stored) {
      const link = `${window.location.origin}${ordersPath.value}${guestOrderFragmentFor(order.id, access)}`;
      stripeFallback.value = { url, link };
      return;
    }
    window.location.assign(url);
    return;
  }
  const query = new URLSearchParams({ order: order.id, ...(order.mode === 'offline' ? { reservation: 'received' } : { payment: 'return' }) });
  emit('navigate', `${ordersPath.value}?${query}`, true);
}

watch(product, (item) => {
  variant.value = item?.variants?.length === 1 && merchMaxQuantity(item, item.variants[0]) ? item.variants[0] : null;
  quantity.value = 1;
  photoIndex.value = 0;
  if (item) trackMerchEvent(props.slug, 'merch_product_view', item.id);
});
watch(() => props.route.page, () => {
  showCart.value = false;
  mode.value = '';
  stripeFallback.value = null;
  if (props.route.page === 'shop' && !catalogLoaded && !catalogError.value) void loadCatalog();
});
watch(() => props.route.productId, () => {
  mode.value = '';
  stripeFallback.value = null;
});
watch(hasDialog, async (open) => {
  await nextTick();
  if (!dialog.value || hasDialog.value !== open) return;
  if (open && !dialog.value.open) {
    dialogTrigger = document.activeElement;
    dialog.value.showModal();
  } else if (!open && dialog.value.open) {
    dialog.value.close();
    if (dialogTrigger instanceof HTMLElement && document.contains(dialogTrigger)) dialogTrigger.focus();
  }
});
watch([mode, stripeFallback], async () => {
  await nextTick();
  if (dialog.value?.open) dialog.value.scrollTop = 0;
});

onMounted(async () => {
  void loadCatalog();
  brandRequest = new AbortController();
  try {
    const { data } = await apiClient.get('/organization-theme', { params: { organization_slug: props.slug }, signal: brandRequest.signal, timeout: 10000 });
    if (!brandRequest.signal.aborted) brand.value = data || {};
  } catch { /* The club slug and default theme keep the shop usable. */ }
});
onBeforeUnmount(() => {
  catalogRequest?.abort();
  brandRequest?.abort();
  observer?.disconnect();
  window.clearTimeout(noticeTimer);
});
</script>

<style src="./shop.css"></style>
