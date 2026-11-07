<template>
  <V2Dialog :open="true" drawer :title="product.id ? 'Modifica prodotto' : 'Nuovo prodotto'" :busy="busy || uploading" @close="close">
    <form :id="formId" ref="form" class="v2-stack" @submit.prevent="save">
      <fieldset class="v2-stack">
        <legend>Informazioni prodotto</legend>
        <div class="v2-grid-2">
          <label class="v2-field">Nome<input v-model.trim="draft.name" required maxlength="120" /></label>
          <label class="v2-field">Categoria<input v-model.trim="draft.category" maxlength="80" placeholder="Abbigliamento, accessori…" /></label>
        </div>
        <label class="v2-field">Descrizione breve<input v-model.trim="draft.short_description" maxlength="240" /></label>
        <label class="v2-field">Descrizione completa<textarea v-model.trim="draft.description" maxlength="5000" /></label>
      </fieldset>

      <fieldset class="v2-stack">
        <legend>Fotografie</legend>
        <label class="v2-field merch-drop" :class="{ dragging }" @dragover.prevent="dragging = true"
          @dragleave="dragging = false" @drop.prevent="dropFiles">
          Trascina qui le immagini o selezionale
          <small>JPG, PNG o WebP. Fino a 12 immagini, massimo 5 MB per file.</small>
          <input type="file" multiple accept="image/jpeg,image/png,image/webp" :disabled="uploading || busy"
            @change="selectFiles" />
        </label>
        <p v-if="uploading" class="v2-muted" role="status">Caricamento immagini…</p>
        <p v-if="uploadError" class="v2-error" role="alert">{{ uploadError }}</p>
        <div v-if="draft.media.length" class="merch-gallery">
          <div v-for="(image, index) in draft.media" :key="image.id" class="merch-image">
            <img :src="resolveApiUrl(merchMediaPath(image.url))" :alt="index ? 'Foto prodotto ' + (index + 1) : 'Copertina prodotto'" />
            <span v-if="index === 0" class="v2-badge success">Copertina</span>
            <button v-else type="button" class="v2-link" @click="setCover(index)">Usa come copertina</button>
            <button type="button" class="v2-button" :aria-label="'Rimuovi foto ' + (index + 1)" @click="draft.media.splice(index, 1)">Rimuovi</button>
          </div>
        </div>
      </fieldset>

      <fieldset class="v2-stack">
        <legend>Prezzi e disponibilità</legend>
        <div class="v2-grid-2">
          <label class="v2-field">Prezzo (€)<input v-model="draft.price" inputmode="decimal" placeholder="29,90" required /></label>
          <label class="v2-field">Prezzo promozionale (€)<input v-model="draft.promoPrice" inputmode="decimal" placeholder="Facoltativo" /></label>
          <label class="v2-field">SKU<input v-model.trim="draft.sku" maxlength="100" /></label>
          <label class="v2-field">Quantità massima per ordine<input v-model.number="draft.max_quantity" type="number" min="1" max="99" required /></label>
          <label v-if="!draft.variants.length" class="v2-field">
            Stock fisico<input v-model.number="draft.stock" type="number" min="0" max="1000000" required />
            <small v-if="product.id">{{ product.available ?? product.stock }} unità disponibili al momento del caricamento.</small>
          </label>
          <label v-if="!draft.variants.length" class="v2-check"><input v-model="draft.on_demand" type="checkbox" /> Prodotto su ordinazione</label>
        </div>
        <div class="v2-stack">
          <div class="v2-toolbar">
            <h3>Varianti</h3>
            <button class="v2-button" type="button" :disabled="draft.variants.length >= 100" @click="addVariant">
              <V2Icon name="plus" :size="16" /> Aggiungi variante
            </button>
          </div>
          <p v-if="!draft.variants.length" class="v2-muted">Aggiungi taglie o colori solo se il prodotto li prevede.</p>
          <fieldset v-for="(variant, index) in draft.variants" :key="variant.key" class="v2-stack">
            <legend>Variante {{ index + 1 }}</legend>
            <div class="v2-grid-2">
              <label class="v2-field">Taglia<input v-model.trim="variant.size" maxlength="40" placeholder="Es. M" /></label>
              <label class="v2-field">Colore<input v-model.trim="variant.color" maxlength="60" placeholder="Es. Blu" /></label>
              <label class="v2-field">SKU variante<input v-model.trim="variant.sku" maxlength="100" /></label>
              <label class="v2-field">Prezzo variante (€)<input v-model="variant.price" inputmode="decimal" placeholder="Usa il prezzo prodotto" /></label>
              <label class="v2-field">
                Stock fisico<input v-model.number="variant.stock" type="number" min="0" max="1000000" required />
                <small v-if="variant.id">{{ variant.available ?? variant.stock }} unità disponibili al momento del caricamento.</small>
              </label>
              <label class="v2-check"><input v-model="variant.on_demand" type="checkbox" /> Su ordinazione</label>
            </div>
            <button class="v2-link" type="button" @click="draft.variants.splice(index, 1)">Rimuovi variante {{ index + 1 }}</button>
          </fieldset>
        </div>
      </fieldset>

      <fieldset class="v2-stack">
        <legend>Vetrina e vendita</legend>
        <div class="v2-grid-2">
          <label class="v2-field">Modalità di vendita<select v-model="draft.sale_mode"><option v-for="(label, mode) in saleModes" :key="mode" :value="mode">{{ label }}</option></select></label>
          <label class="v2-field">Posizione in vetrina<input v-model.number="draft.position" type="number" min="0" max="100000" required /></label>
          <label class="v2-field">Identificativo asset Arena 3D<input v-model.trim="draft.digital_asset_id" maxlength="160" placeholder="Facoltativo" /><small>Chiave di un articolo esistente nel guardaroba Arena.</small></label>
          <label class="v2-check"><input v-model="draft.shipping_allowed" type="checkbox" /> Consenti spedizione</label>
          <label class="v2-check"><input v-model="draft.published" type="checkbox" /> Pubblicato</label>
          <label class="v2-check"><input v-model="draft.featured" type="checkbox" /> Metti in evidenza</label>
        </div>
      </fieldset>
      <p v-if="error" class="v2-error" role="alert">{{ error }}</p>
    </form>
    <template #footer>
      <button class="v2-button" type="button" :disabled="busy || uploading" @click="close">Chiudi</button>
      <button class="v2-button v2-primary" type="submit" :form="formId" :disabled="busy || uploading">
        {{ busy ? 'Salvataggio…' : 'Salva prodotto' }}
      </button>
    </template>
  </V2Dialog>
</template>

<script setup>
import { inject, reactive, ref, useId } from 'vue';
import { resolveApiUrl } from '../../api';
import V2Dialog from './shared/V2Dialog.vue';
import V2Icon from './shared/V2Icon.vue';
import { euroInput, merchMediaPath, merchRequestError, saleModes, toCents } from '../merchModel';

const props = defineProps({
  product: { type: Object, required: true },
  persist: { type: Function, required: true },
  upload: { type: Function, required: true },
});
const emit = defineEmits(['close', 'saved']);
const c = inject('admin-v2');
const form = ref(null);
const formId = useId();
let variantKey = 0;
const draft = reactive({
  id: '', name: '', category: '', short_description: '', description: '', sku: '',
  stock: 0, on_demand: false, max_quantity: 5, published: false, featured: false,
  position: 0, sale_mode: 'inherit', digital_asset_id: '', shipping_allowed: false,
  version: 0, ...props.product,
  price: euroInput(props.product.price_cents), promoPrice: euroInput(props.product.promo_price_cents),
  media: (props.product.media || []).map(image => ({ ...image })),
  variants: (props.product.variants || []).map(variant => ({
    ...variant, key: ++variantKey, price: euroInput(variant.price_cents),
  })),
});
const initial = JSON.stringify(draft);
const busy = ref(false);
const dragging = ref(false);
const uploading = ref(false);
const error = ref('');
const uploadError = ref('');

function addVariant() {
  draft.variants.push({ key: ++variantKey, id: '', size: '', color: '', sku: '', price: '', stock: 0, on_demand: false });
}

function setCover(index) {
  draft.media.unshift(draft.media.splice(index, 1)[0]);
}

function close() {
  if (JSON.stringify(draft) !== initial && !window.confirm('Chiudere senza salvare le modifiche al prodotto?')) return;
  c.discardDraft(form.value);
  emit('close');
}

function dropFiles(event) {
  dragging.value = false;
  return uploadFiles([...event.dataTransfer.files]);
}

function selectFiles(event) {
  const files = [...event.target.files];
  event.target.value = '';
  return uploadFiles(files);
}

async function uploadFiles(files) {
  if (!files.length || uploading.value || busy.value) return;
  uploadError.value = '';
  if (draft.media.length + files.length > 12) {
    uploadError.value = 'La galleria può contenere fino a 12 immagini.';
    return;
  }
  uploading.value = true;
  const failures = [];
  try {
    for (const file of files) {
      if (!['image/jpeg', 'image/png', 'image/webp'].includes(file.type) || file.size > 5 * 1024 * 1024 || !file.size) {
        failures.push(file.name + ' · formato o dimensione non validi');
        continue;
      }
      try {
        draft.media.push(await props.upload(file));
      } catch (failure) {
        failures.push(file.name + ' · ' + merchRequestError(failure));
      }
    }
    uploadError.value = failures.join(' / ');
  } finally {
    uploading.value = false;
  }
}

async function save() {
  if (busy.value || uploading.value) return;
  busy.value = true;
  error.value = '';
  try {
    const { price, promoPrice, variants, available, ...product } = draft;
    const payload = {
      ...product,
      price_cents: toCents(price),
      promo_price_cents: promoPrice.trim() ? toCents(promoPrice) : null,
      variants: variants.map(({ key, price, available, ...variant }) => ({
        ...variant, price_cents: price.trim() ? toCents(price) : null,
      })),
    };
    if (payload.price_cents < 1) throw new Error('Il prezzo di listino deve essere positivo.');
    if (payload.promo_price_cents !== null && (payload.promo_price_cents < 1 || payload.promo_price_cents >= payload.price_cents)) {
      throw new Error('Il prezzo promozionale deve essere inferiore al prezzo di listino.');
    }
    const saved = await props.persist(payload);
    c.discardDraft(form.value);
    emit('saved', saved);
  } catch (failure) {
    error.value = merchRequestError(failure);
  } finally {
    busy.value = false;
  }
}
</script>

<style scoped>
.merch-drop { padding: 18px; border: 1px dashed #9db5a9; border-radius: 10px; background: #f6faf8; }
.merch-drop.dragging { border-color: var(--v2-primary); background: var(--v2-primary-soft); }
.merch-gallery { display: grid; grid-template-columns: repeat(auto-fill, minmax(115px, 1fr)); gap: 10px; }
.merch-image { display: flex; flex-direction: column; align-items: start; gap: 8px; min-width: 0; }
.merch-image img { width: 100%; aspect-ratio: 1; object-fit: cover; border-radius: 8px; background: var(--v2-soft); }
</style>
