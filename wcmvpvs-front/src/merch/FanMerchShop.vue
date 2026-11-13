<template>
  <Teleport to="body" :disabled="!sheet">
    <component :is="sheet ? 'dialog' : 'main'" ref="sheetDialog" class="merch-shell" :class="{ 'merch-sheet': sheet }" :style="theme"
      :aria-labelledby="sheet ? 'merch-sheet-title' : undefined" @cancel.prevent="closeSheet">
      <header v-if="sheet" class="sheet-header">
        <div class="sheet-grip" aria-hidden="true" @pointerdown="startDrag" @pointerup="endDrag" @pointercancel="dragStart = null"><span></span></div>
        <div class="sheet-heading">
          <button v-if="hasDetail" class="sheet-icon" type="button" aria-label="Indietro" :disabled="checkoutBusy" @click="sheetNavigation.back()">
            <svg viewBox="0 0 24 24" aria-hidden="true"><path d="m14 6-6 6 6 6m-6-6h12"/></svg>
          </button>
          <h1 id="merch-sheet-title" ref="sheetTitle" tabindex="-1">{{ sheetHeading }}</h1>
          <button v-if="!hasDetail && cart.length && shopEnabled" class="sheet-icon sheet-cart" type="button" :aria-label="`Carrello, ${cartCount} articoli`" @click="openCart">
            <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M5 8h14l2 13H3L5 8Zm3 2V7a4 4 0 0 1 8 0v3"/></svg>
            <span>{{ cartCount }}</span>
          </button>
          <button class="sheet-icon sheet-close" type="button" aria-label="Chiudi Shop" :disabled="checkoutBusy" @click="closeSheet">
            <svg viewBox="0 0 24 24" aria-hidden="true"><path d="m6 6 12 12M18 6 6 18"/></svg>
          </button>
        </div>
        <div v-if="!hasDetail" class="sheet-tabs" role="tablist" aria-label="Shop" @keydown="onTabKey">
          <button v-for="tab in tabs" :id="`merch-tab-${tab.page}`" :key="tab.page" type="button" role="tab"
            :aria-selected="page === tab.page" :tabindex="page === tab.page ? 0 : -1" :aria-controls="`merch-panel-${tab.page}`"
            @click="chooseTab(tab.page)">{{ tab.label }}</button>
        </div>
      </header>
      <header v-else class="shop-nav">
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

      <div ref="sheetContent" :class="{ 'sheet-content': sheet }">
        <div v-show="!sheet || !hasDialog" class="shop-container">
          <div v-if="sheet || page === 'orders'" v-show="page === 'orders'" :id="sheet ? 'merch-panel-orders' : undefined"
            :role="sheet && !hasDetail ? 'tabpanel' : undefined" :aria-labelledby="sheet && !hasDetail ? 'merch-tab-orders' : undefined">
            <MerchOrders v-if="!sheet || ordersVisited" :slug="slug" :search="sheet ? orderSearch : search" :latest-order="latestOrder"
              :compact="sheet" :selected-id="sheetView?.orderId" :products="products"
              @select="openOrder" @browse="chooseTab('shop')" @list="chooseTab('orders')" @busy-change="checkoutBusy = $event" />
          </div>
          <div v-if="sheet || page === 'shop'" v-show="page === 'shop'" :id="sheet ? 'merch-panel-shop' : undefined"
            :role="sheet ? 'tabpanel' : undefined" :aria-labelledby="sheet ? 'merch-tab-shop' : undefined" :tabindex="sheet ? 0 : undefined">
            <section v-if="!sheet" class="shop-hero">
              <p class="eyebrow">MERCHANDISING UFFICIALE</p>
              <h1>Shop del club</h1>
              <p v-if="shopEnabled">Scopri i prodotti di {{ brandName }} e scegli come riceverli.</p>
            </section>
            <p v-if="loading" class="empty" role="status">Caricamento del catalogo…</p>
            <div v-else-if="catalogError" class="empty" role="alert">
              <p>{{ catalogError }}</p><button class="secondary" type="button" @click="loadCatalog">Riprova</button>
            </div>
            <p v-else-if="!shopEnabled" class="empty">{{ sheet ? 'Shop al momento chiuso.' : `Lo Shop di ${brandName} al momento è chiuso.` }}</p>
            <template v-else>
              <div v-if="!sheet && products.length" class="filters" aria-label="Filtri catalogo">
                <label class="field">Cerca un prodotto
                  <input v-model.trim="filters.search" type="search" placeholder="Nome o categoria" />
                </label>
                <label v-if="categories.length" class="field">Categoria
                  <select v-model="filters.category"><option value="">Tutte</option><option v-for="category in categories" :key="category">{{ category }}</option></select>
                </label>
                <label class="check"><input v-model="filters.inStock" type="checkbox" /> Solo disponibili</label>
              </div>
              <p v-if="!products.length" class="empty">Il club sta preparando i prossimi prodotti.</p>
              <p v-else-if="!catalogItems.length" class="empty">Nessun prodotto corrisponde alla ricerca.</p>
              <div v-else class="product-grid">
                <component :is="sheet ? 'button' : 'a'" v-for="(item, index) in catalogItems" :key="item.id" :href="sheet ? undefined : productPath(item.id)"
                  :type="sheet ? 'button' : undefined" class="product-card" :data-product-id="item.id" :ref="observeProduct" @click="openProduct($event, item.id)">
                  <span class="product-photo">
                    <img v-if="item.media?.[0]?.url" :src="mediaUrl(item.media[0].url)" alt="" :loading="!sheet || index > 1 ? 'lazy' : 'eager'" decoding="async" />
                    <svg v-else-if="sheet" class="sheet-photo-icon" viewBox="0 0 24 24" aria-hidden="true"><path d="M5 8h14l2 13H3L5 8Zm3 2V7a4 4 0 0 1 8 0v3"/></svg>
                    <span v-else class="photo-fallback" aria-hidden="true">ABX</span>
                    <span v-if="item.featured" class="featured">In evidenza</span>
                  </span>
                  <span class="card-copy">
                    <small v-if="!sheet && item.category">{{ item.category }}</small>
                    <component :is="sheet ? 'span' : 'h2'" class="card-name">{{ item.name }}</component>
                    <span v-if="!sheet && item.short_description" class="card-description">{{ item.short_description }}</span>
                    <span class="card-price"><span>{{ cardPrice(item) }}</span><span v-if="!sheet && !isAvailable(item)" class="muted">Esaurito</span></span>
                  </span>
                </component>
              </div>
              <p v-if="!sheet && route.productId && !product" class="empty" role="status">Questo prodotto non è disponibile. <a :href="shopPath" @click="follow($event, shopPath)">Torna al catalogo</a></p>
            </template>
          </div>
          <footer v-if="!sheet && (settings.seller_info || settings.contact_email)" class="seller">
            <p v-if="settings.seller_info">{{ settings.seller_info }}</p>
            <a v-if="settings.contact_email" :href="`mailto:${settings.contact_email}`">Contatta il club</a>
          </footer>
        </div>
        <div v-if="!sheet && notice" class="toast" role="status">{{ notice }}</div>
        <component :is="sheet ? 'div' : 'dialog'" ref="dialog" :class="sheet ? 'sheet-detail' : 'merch-dialog'"
          :aria-labelledby="sheet ? undefined : 'merch-dialog-title'" @cancel.prevent.stop="dismiss" @click.self="!sheet && dismiss()">
          <div v-if="hasDialog" class="dialog-body">
            <button v-if="!sheet" class="dialog-close" type="button" aria-label="Chiudi" :disabled="checkoutBusy" @click="dismiss">×</button>
            <component :is="sheet ? KeepAlive : 'div'" :max="sheet ? 3 : undefined">
              <MerchCheckout v-if="checkoutMode && !currentFallback" :key="sheetView?.checkoutId || checkoutMode"
                :compact="sheet" :slug="slug" :event-id="eventId" :mode="checkoutMode" :lines="activeLines" :settings="settings"
                :delivery-options="deliveryOptions" @back="mode = ''" @created="handleOrder" @recoverable-order="rememberOrder" @busy-change="checkoutBusy = $event" />
            </component>
            <section v-if="currentFallback" class="checkout">
              <div class="dialog-heading">
                <h2 v-if="!sheet" id="merch-dialog-title">Vai al pagamento sicuro</h2>
                <p>Il browser impedisce di salvare l’ordine. Conserva il collegamento privato prima di continuare.</p>
              </div>
              <label class="field">Collegamento privato<input :value="currentFallback.link" readonly @focus="$event.target.select()" /></label>
              <a class="primary" :href="currentFallback.url">Continua su Stripe</a>
            </section>
            <section v-else-if="!checkoutMode && showCart" class="cart-view">
              <div v-if="!sheet" class="dialog-heading"><h2 id="merch-dialog-title">Il tuo carrello</h2><p>Completa più articoli in una sola operazione.</p></div>
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
            <section v-else-if="!checkoutMode && product" class="product-view">
              <div class="product-gallery">
                <div class="product-photo">
                  <img v-if="product.media?.[photoIndex]?.url" :src="mediaUrl(product.media[photoIndex].url)" :alt="product.name" decoding="async" />
                  <svg v-else-if="sheet" class="sheet-photo-icon" viewBox="0 0 24 24" aria-hidden="true"><path d="M5 8h14l2 13H3L5 8Zm3 2V7a4 4 0 0 1 8 0v3"/></svg>
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
                  <p v-if="!sheet && product.category" class="eyebrow">{{ product.category }}</p>
                  <h2 id="merch-dialog-title">{{ product.name }}</h2>
                  <p v-if="!sheet && product.short_description">{{ product.short_description }}</p>
                </div>
                <div class="detail-price">
                  <strong>{{ variant || !product.variants?.length ? merchMoney(merchUnitPrice(product, variant)) : cardPrice(product) }}</strong>
                  <del v-if="product.promo_price_cents != null && variant?.price_cents == null && !product.variants?.length">{{ merchMoney(product.price_cents) }}</del>
                  <span class="muted">{{ maxQuantity ? (variant?.on_demand || (!variant && product.on_demand) ? 'Su ordinazione' : 'Disponibile') : isAvailable(product) ? 'Scegli una variante' : 'Esaurito' }}</span>
                </div>
                <fieldset v-if="product.variants?.length" class="variant-field">
                  <legend>{{ variantLegend }}</legend>
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
                <p v-if="sheet && product.short_description" class="product-description">{{ product.short_description }}</p>
                <div v-if="productOptions.online || productOptions.offline" class="actions">
                  <button v-for="(action, index) in saleActions(productOptions)" :key="action" :class="index ? 'secondary' : 'primary'" type="button" :disabled="!productReady" @click="startCheckout(action, [selectedLine])">{{ saleLabels[action] }}</button>
                </div>
                <p v-else class="muted">La società renderà disponibile l’ordine a breve.</p>
                <button v-if="productReady && (productOptions.online || productOptions.offline)" class="text-button" type="button" @click="addToCart">Aggiungi al carrello</button>
                <details v-if="product.description && (!sheet || product.description !== product.short_description)">
                  <summary>Dettagli prodotto</summary><p>{{ product.description }}</p>
                </details>
              </div>
            </section>
            <p v-else-if="sheet && !checkoutMode" class="empty" role="status">Questo prodotto non è disponibile.</p>
          </div>
        </component>
      </div>
      <div v-if="sheet && notice" class="toast" role="status">{{ notice }}</div>
    </component>
  </Teleport>
</template>

<script setup>
import { computed, KeepAlive, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';
import { apiClient } from '../api';
import { normalizeOrganizationColors, organizationThemeStyle } from '../organizationTheme';
import MerchCheckout from './MerchCheckout.vue';
import MerchOrders from './MerchOrders.vue';
import { fetchMerchCatalog, merchMediaUrl as mediaUrl, merchRequestError, trackMerchEvent } from './merchApi';
import { createSheetNavigation } from './sheetNavigation';
import {
  fanPortalPath, guestOrderFragmentFor, merchItemKey, merchMaxQuantity, merchMoney, merchOptions,
  merchUnitPrice, merchVariantLabel, saveGuestOrder, selectedMerchProducts, stripeCheckoutUrl,
} from './shopModel';

const props = defineProps({
  slug: { type: String, required: true },
  eventId: { type: Number, default: 0 },
  route: { type: Object, default: null },
  search: { type: String, default: '' },
  sheet: Boolean,
  openRequest: { type: Object, default: null },
  clubName: { type: String, default: '' },
  colors: { type: Object, default: null },
});
const emit = defineEmits(['navigate']);
const homePath = computed(() => fanPortalPath(props.slug));
const shopPath = computed(() => `${homePath.value}/shop`);
const ordersPath = computed(() => `${homePath.value}/orders`);
const productPath = id => `${shopPath.value}/${encodeURIComponent(id)}`;
const brand = ref({});
const brandName = computed(() => props.clubName || brand.value.name || props.slug);
const theme = computed(() => organizationThemeStyle(props.sheet
  ? { ...props.colors, tertiary_color: '#f7f8fb' }
  : normalizeOrganizationColors(brand.value)));
const sheetView = ref(null);
const sheetDialog = ref(null);
const sheetContent = ref(null);
const sheetTitle = ref(null);
const ordersVisited = ref(false);
const orderSearch = ref('');
const tabs = [{ page: 'shop', label: 'Prodotti' }, { page: 'orders', label: 'I miei ordini' }];
const page = computed(() => props.sheet ? sheetView.value?.page || 'shop' : props.route?.page);
const productId = computed(() => props.sheet ? sheetView.value?.productId : props.route?.productId);
let sheetNavigation;
let sheetTrigger;
let unlockScroll;
let dragStart;
let checkoutSequence = 0;
const sheetScroll = new Map();
const productSelections = new Map();
const products = ref([]);
const settings = ref({});
const loading = ref(true);
const catalogError = ref('');
const shopEnabled = computed(() => Boolean(settings.value.enabled));
const filters = reactive({ search: '', category: '', inStock: false });
const categories = computed(() => [...new Set(products.value.map(item => item.category).filter(Boolean))].sort((a, b) => a.localeCompare(b, 'it')));
const catalogItems = computed(() => props.sheet ? products.value : selectedMerchProducts(products.value, filters));
const product = computed(() => shopEnabled.value && !loading.value ? products.value.find(item => item.id === productId.value) : null);
const variant = ref(null);
const quantity = ref(1);
const photoIndex = ref(0);
const maxQuantity = computed(() => merchMaxQuantity(product.value, variant.value));
const variantLegend = computed(() => {
  const sizes = product.value.variants.some(option => option.size);
  const colors = product.value.variants.some(option => option.color);
  if (!props.sheet && (sizes || colors)) return 'Taglia e colore';
  return sizes && colors ? 'Taglia e colore' : sizes ? 'Taglia' : colors ? 'Colore' : 'Variante';
});
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
const legacyCart = ref(false);
const showCart = computed(() => props.sheet ? Boolean(sheetView.value?.cart) : legacyCart.value);
const mode = ref('');
const checkoutLines = ref([]);
const checkoutMode = computed(() => props.sheet ? sheetView.value?.mode : mode.value);
const activeLines = computed(() => props.sheet ? sheetView.value?.lines || [] : checkoutLines.value);
const deliveryOptions = computed(() => merchOptions(products.value, settings.value, activeLines.value));
const checkoutBusy = ref(false);
const latestOrder = ref(null);
const stripeFallback = ref(null);
const currentFallback = computed(() => props.sheet ? sheetView.value?.fallback : stripeFallback.value);
const hasDialog = computed(() => page.value === 'shop' && (showCart.value || Boolean(props.sheet ? productId.value : product.value)));
const hasDetail = computed(() => hasDialog.value || Boolean(sheetView.value?.orderId));
const sheetHeading = computed(() => {
  if (currentFallback.value) return 'Vai al pagamento sicuro';
  if (checkoutMode.value) return checkoutMode.value === 'offline' ? 'Conferma prenotazione' : 'Riepilogo acquisto';
  if (showCart.value) return 'Carrello';
  if (productId.value) return product.value?.name || 'Prodotto';
  if (sheetView.value?.orderId) return 'Dettaglio ordine';
  return `Shop ${brandName.value}`;
});
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
const saleLabels = props.sheet
  ? { online: 'Acquista', offline: 'Prenota' }
  : { online: 'Acquista ora', offline: 'Prenota e paga alla società' };

function saleActions(options) {
  const preferred = settings.value.default_mode === 'offline' ? ['offline', 'online'] : ['online', 'offline'];
  return preferred.filter(action => options[action]);
}

function follow(event, path) {
  if (event.button || event.ctrlKey || event.shiftKey || event.metaKey || event.altKey) return;
  event.preventDefault();
  emit('navigate', path);
}

function openProduct(event, id) {
  if (props.sheet) sheetNavigation.show({ page: 'shop', productId: id });
  else follow(event, productPath(id));
}

function openOrder(id) {
  if (sheetView.value?.page === 'orders') sheetNavigation.show({ page: 'orders', orderId: id });
}

function chooseTab(nextPage) {
  if (sheetView.value && !checkoutBusy.value && (page.value !== nextPage || hasDetail.value)) {
    sheetNavigation.show({ page: nextPage }, true);
  }
}

function onTabKey(event) {
  if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return;
  event.preventDefault();
  const next = event.key === 'Home' ? 'shop' : event.key === 'End' ? 'orders' : page.value === 'shop' ? 'orders' : 'shop';
  chooseTab(next);
  nextTick(() => document.getElementById(`merch-tab-${next}`)?.focus());
}

function openSheet() {
  if (!props.openRequest || sheetView.value) return;
  sheetTrigger = props.openRequest.trigger;
  sheetScroll.clear();
  sheetNavigation.show({ page: 'shop' });
  if (props.eventId > 0) recordShopEvent('merch_shop_opened');
  void loadCatalog();
}

function recordShopEvent(type, productId, variantId) {
  trackMerchEvent(props.slug, type, productId, variantId, props.eventId);
}

function closeSheet() {
  sheetNavigation?.back(true);
}

function startDrag(event) {
  if (checkoutBusy.value || !event.isPrimary || event.button !== 0) return;
  dragStart = event.clientY;
  event.currentTarget.setPointerCapture(event.pointerId);
}

function endDrag(event) {
  if (dragStart != null && event.clientY - dragStart >= 60) closeSheet();
  dragStart = null;
}

function lockPageScroll() {
  const { scrollX, scrollY } = window;
  const style = document.body.style;
  const properties = { position: 'fixed', top: `-${scrollY}px`, left: `-${scrollX}px`, width: '100%', overflow: 'hidden' };
  const previous = Object.fromEntries(Object.keys(properties).map(key => [key, style[key]]));
  Object.assign(style, properties);
  return () => {
    Object.assign(style, previous);
    if (window.scrollX !== scrollX || window.scrollY !== scrollY) window.scrollTo(scrollX, scrollY);
  };
}

function viewKey(view) {
  if (view.mode) return 'checkout';
  if (view.productId) return `product:${view.productId}`;
  if (view.orderId) return `order:${view.orderId}`;
  return view.cart ? 'cart' : view.page;
}

watch(sheetView, async (view, previous) => {
  if (previous) sheetScroll.set(viewKey(previous), sheetContent.value.scrollTop);
  if (view?.page === 'orders') ordersVisited.value = true;
  const focusedTab = document.activeElement?.getAttribute('role') === 'tab';
  await nextTick();
  if (sheetView.value !== view || !sheetDialog.value) return;
  if (!view) {
    sheetDialog.value.close();
    unlockScroll?.();
    unlockScroll = undefined;
    sheetTrigger?.focus({ preventScroll: true });
    return;
  }
  if (!sheetDialog.value.open) {
    unlockScroll = lockPageScroll();
    sheetDialog.value.showModal();
  }
  const key = viewKey(view);
  sheetContent.value.scrollTop = view.mode || view.fallback ? 0 : sheetScroll.get(key) || 0;
  let focusTarget;
  if (key === 'shop' && previous?.productId) {
    focusTarget = sheetContent.value.querySelector(`[data-product-id="${CSS.escape(previous.productId)}"]`);
  } else if (key === 'orders' && previous?.orderId) {
    focusTarget = sheetContent.value.querySelector(`[data-order-id="${CSS.escape(previous.orderId)}"]`);
  } else if (focusedTab && !hasDetail.value) {
    focusTarget = document.getElementById(`merch-tab-${view.page}`);
  }
  (focusTarget || sheetTitle.value).focus({ preventScroll: true });
});

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
          recordShopEvent('merch_impression', id);
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
  recordShopEvent('merch_variant_selected', product.value.id, option.id);
}

function openCart() {
  if (props.sheet) sheetNavigation.show({ page: 'shop', cart: true });
  else {
    legacyCart.value = true;
    mode.value = '';
  }
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
  if (props.sheet) sheetNavigation.back();
  else dismiss();
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
  const snapshot = lines.map(line => ({ ...line }));
  recordShopEvent('merch_outbound_click', snapshot[0]?.product_id);
  if (props.sheet) sheetNavigation.show({ ...sheetView.value, mode: nextMode, lines: snapshot, checkoutId: ++checkoutSequence });
  else {
    checkoutLines.value = snapshot;
    mode.value = nextMode;
  }
}

function dismiss() {
  if (checkoutBusy.value) return;
  mode.value = '';
  stripeFallback.value = null;
  legacyCart.value = false;
  if (props.route.productId) emit('navigate', shopPath.value, true);
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
      if (props.sheet) sheetNavigation.show({ ...sheetView.value, fallback: { url, link } }, true);
      else stripeFallback.value = { url, link };
      return;
    }
    window.location.assign(url);
    return;
  }
  const query = new URLSearchParams({ order: order.id, ...(order.mode === 'offline' ? { reservation: 'received' } : { payment: 'return' }) });
  if (props.sheet) {
    orderSearch.value = query.toString();
    sheetNavigation.show({ page: 'orders', orderId: order.id }, true);
  } else emit('navigate', `${ordersPath.value}?${query}`, true);
}

watch(product, (item, previous) => {
  if (props.sheet && previous) {
    productSelections.set(previous.id, { variantId: variant.value?.id, quantity: quantity.value, photoIndex: photoIndex.value });
  }
  const selection = props.sheet && item ? productSelections.get(item.id) : null;
  const preferred = item?.variants?.find(option => option.id === selection?.variantId)
    || (item?.variants?.length === 1 ? item.variants[0] : null);
  variant.value = merchMaxQuantity(item, preferred) ? preferred : null;
  quantity.value = Math.max(1, Math.min(selection?.quantity || 1, maxQuantity.value));
  photoIndex.value = selection?.photoIndex < item?.media?.length ? selection.photoIndex : 0;
  if (item) recordShopEvent('merch_product_view', item.id);
});
watch(() => props.route?.page, () => {
  legacyCart.value = false;
  mode.value = '';
  stripeFallback.value = null;
  if (props.route.page === 'shop' && !catalogLoaded && !catalogError.value) void loadCatalog();
});
watch(() => props.route?.productId, () => {
  mode.value = '';
  stripeFallback.value = null;
});
watch(hasDialog, async (open) => {
  if (props.sheet) return;
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
  if (props.sheet) {
    sheetNavigation = createSheetNavigation(window, view => { sheetView.value = view; }, () => checkoutBusy.value);
    openSheet();
    return;
  }
  void loadCatalog();
  brandRequest = new AbortController();
  try {
    const { data } = await apiClient.get('/organization-theme', { params: { organization_slug: props.slug }, signal: brandRequest.signal, timeout: 10000 });
    if (!brandRequest.signal.aborted) brand.value = data || {};
  } catch { /* The club slug and default theme keep the shop usable. */ }
});
watch(() => props.openRequest, () => { if (sheetNavigation) openSheet(); });
onBeforeUnmount(() => {
  sheetNavigation?.dispose();
  unlockScroll?.();
  catalogRequest?.abort();
  brandRequest?.abort();
  observer?.disconnect();
  window.clearTimeout(noticeTimer);
});
</script>

<style src="./shop.css"></style>
<style src="./sheet.css"></style>
