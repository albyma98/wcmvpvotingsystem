<template>
  <section class="card">
    <SectionHeader
      title="Coupon"
      description="Crea e gestisci i coupon promozionali collegati agli sponsor e alle partite."
    />

    <form @submit.prevent="createCoupon" class="form-grid coupon-form">
      <label>
        Titolo
        <input v-model.trim="newCoupon.title" type="text" placeholder="Sconto speciale" required />
      </label>
      <label>
        Descrizione breve
        <textarea v-model.trim="newCoupon.shortDesc" rows="2" placeholder="Testo sintetico per il coupon"></textarea>
      </label>
      <label class="choice-group">
        Partner
        <div class="status-options" role="radiogroup" aria-label="Partner associato al coupon">
          <label v-for="partner in props.partners" :key="partner.id" class="status-option">
            <input
              type="radio"
              name="new-coupon-partner"
              :value="partner.id"
              v-model.number="newCoupon.sponsorId"
              :disabled="!props.partners.length"
            />
            <span>{{ partner.displayName || partner.username || `Partner ${partner.id}` }}</span>
          </label>
        </div>
        <small class="field-hint" v-if="!props.partners.length">Aggiungi almeno un partner per creare un coupon.</small>
      </label>
      <label class="choice-group">
        Stato
        <div class="status-options" role="radiogroup" aria-label="Stato coupon">
          <label v-for="status in couponStatusOptions" :key="status" class="status-option">
            <input type="radio" name="new-coupon-status" :value="status" v-model="newCoupon.status" />
            <span>{{ getCouponStatusLabel(status) }}</span>
          </label>
        </div>
      </label>
      <label>
        Limite utilizzi
        <input v-model.number="newCoupon.maxUses" type="number" min="0" placeholder="0 per illimitato" />
      </label>
      <label>
        Data inizio
        <input v-model="newCoupon.startDateInput" type="datetime-local" />
      </label>
      <label>
        Data fine
        <input v-model="newCoupon.endDateInput" type="datetime-local" />
      </label>
      <label>
        Immagine (URL)
        <input
          v-model.trim="newCoupon.imageUrl"
          type="text"
          inputmode="url"
          placeholder="https://example.com/immagine.jpg"
          @input="syncCouponImageSource(newCoupon)"
        />
        <small class="field-hint">In alternativa puoi caricare un file dal dispositivo.</small>
      </label>
      <label class="file-input coupon-file-input">
        Carica immagine
        <input type="file" accept="image/*" @change="(event) => handleCouponImageFileChange(event, newCoupon)" />
      </label>
      <div v-if="newCoupon.imagePreview" class="coupon-image-preview" aria-label="Anteprima immagine coupon">
        <img :src="newCoupon.imagePreview" alt="Anteprima coupon" />
        <button class="btn link" type="button" @click="clearCouponImage(newCoupon)">Rimuovi immagine</button>
      </div>
      <div class="match-selector">
        <span>Associa alle partite</span>
        <BaseSearchInput v-model="couponEventSearch" class="match-search" placeholder="Cerca partita per squadra o data" />
        <div class="match-selector__grid match-selector__grid--scroll">
          <label v-for="event in filteredCouponEvents" :key="event.id" class="checkbox">
            <input type="checkbox" :value="event.id" v-model.number="newCoupon.matchIds" />
            <span>
              {{ couponMatchLabel(event) }}
              <small class="muted block">{{ event.start_datetime }}</small>
            </span>
          </label>
          <p v-if="!props.events.length" class="muted small">
            Nessuna partita disponibile: crea un evento per associare dei coupon.
          </p>
        </div>
      </div>

      <button class="btn primary" type="submit" :disabled="isCreatingCoupon">
        {{ isCreatingCoupon ? 'Creazione…' : 'Crea coupon' }}
      </button>
    </form>

    <p v-if="couponError" class="error">{{ couponError }}</p>
    <p v-if="couponSuccess" class="success-message">{{ couponSuccess }}</p>

    <ul v-if="coupons.length" class="item-list coupons-list">
      <li v-for="coupon in coupons" :key="coupon.id" class="item coupon-item">
        <div class="coupon-fields">
          <div class="form-grid compact">
            <label>
              Titolo
              <input v-model.trim="coupon.title" type="text" />
            </label>
            <label class="choice-group">
              Partner
              <div class="status-options" role="radiogroup" :aria-label="`Partner per il coupon ${coupon.title || coupon.id}`">
                <label v-for="partner in props.partners" :key="partner.id" class="status-option">
                  <input
                    type="radio"
                    :name="`coupon-partner-${coupon.id}`"
                    :value="partner.id"
                    v-model.number="coupon.sponsorId"
                    :disabled="!props.partners.length"
                  />
                  <span>{{ partner.displayName || partner.username || `Partner ${partner.id}` }}</span>
                </label>
              </div>
            </label>
            <label class="choice-group">
              Stato
              <div class="status-options" role="radiogroup" :aria-label="`Stato coupon ${coupon.title || coupon.id}`">
                <label v-for="status in couponStatusOptions" :key="status" class="status-option">
                  <input type="radio" :name="`coupon-status-${coupon.id}`" :value="status" v-model="coupon.status" />
                  <span>{{ getCouponStatusLabel(status) }}</span>
                </label>
              </div>
            </label>
            <label>
              Limite utilizzi
              <input v-model.number="coupon.maxUses" type="number" min="0" />
            </label>
            <label>
              Data inizio
              <input v-model="coupon.startDateInput" type="datetime-local" />
            </label>
            <label>
              Data fine
              <input v-model="coupon.endDateInput" type="datetime-local" />
            </label>
            <label>
              Immagine (URL)
              <input
                v-model.trim="coupon.imageUrl"
                type="text"
                inputmode="url"
                @input="syncCouponImageSource(coupon)"
              />
              <small class="field-hint">Puoi inserire un URL oppure caricare un file.</small>
            </label>
            <label class="file-input coupon-file-input">
              Carica immagine
              <input type="file" accept="image/*" @change="(event) => handleCouponImageFileChange(event, coupon)" />
            </label>
            <div
              v-if="coupon.imagePreview"
              class="coupon-image-preview"
              :aria-label="`Anteprima immagine per ${coupon.title || coupon.id}`"
            >
              <img :src="coupon.imagePreview" alt="Anteprima coupon" />
              <button class="btn link" type="button" @click="clearCouponImage(coupon)">Rimuovi immagine</button>
            </div>
          </div>
          <label>
            Descrizione breve
            <textarea v-model.trim="coupon.shortDesc" rows="2"></textarea>
          </label>
          <div class="match-selector inline">
            <span class="muted small">Partite associate</span>
            <div class="match-selector__grid">
              <label v-for="event in filteredCouponEvents" :key="event.id" class="checkbox">
                <input type="checkbox" :value="event.id" v-model.number="coupon.matchIds" />
                <span>{{ couponMatchLabel(event) }}</span>
              </label>
            </div>
          </div>
          <p class="muted coupon-meta">
            Viste: {{ coupon.totalViews }} • Richieste: {{ coupon.totalClaims }} •
            Utilizzi: {{ coupon.totalRedemptions }}
          </p>
        </div>
        <div class="item-actions vertical">
          <button
            class="btn secondary"
            type="button"
            @click="updateCouponEntry(coupon)"
            :disabled="couponBeingSaved === coupon.id"
          >
            <span v-if="couponBeingSaved === coupon.id">Salvataggio…</span>
            <span v-else>Salva</span>
          </button>
          <button
            class="btn danger"
            type="button"
            @click="deleteCouponEntry(coupon.id)"
            :disabled="couponBeingDeleted === coupon.id"
          >
            <span v-if="couponBeingDeleted === coupon.id">Eliminazione…</span>
            <span v-else>Elimina</span>
          </button>
        </div>
      </li>
    </ul>
    <p v-else class="muted text-center">Nessun coupon configurato.</p>
  </section>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue';
import { apiClient, resolveApiUrl } from '../../api';
import SectionHeader from './ui/SectionHeader.vue';
import BaseSearchInput from './ui/BaseSearchInput.vue';

const props = defineProps({
  authHeaders: { type: Object, required: true },
  isSuperAdmin: { type: Boolean, default: false },
  events: { type: Array, default: () => [] },
  partners: { type: Array, default: () => [] },
});

const couponStatusOptions = ['draft', 'active', 'paused', 'archived'];
const couponStatusLabels = { draft: 'Bozza', active: 'Attivo', paused: 'In pausa', archived: 'Archiviato' };
const getCouponStatusLabel = (status) => couponStatusLabels[status] || status;

const coupons = ref([]);
const couponError = ref('');
const couponSuccess = ref('');
const isCreatingCoupon = ref(false);
const couponBeingSaved = ref(0);
const couponBeingDeleted = ref(0);
const couponEventSearch = ref('');

function createEmptyCouponDraft() {
  return {
    title: '',
    shortDesc: '',
    sponsorId: props.partners?.[0]?.id ?? 0,
    merchantId: props.partners?.[0]?.id ?? 0,
    matchIds: [],
    startDateInput: '',
    endDateInput: '',
    maxUses: 0,
    status: 'draft',
    imageUrl: '',
    imagePreview: '',
  };
}

const newCoupon = reactive(createEmptyCouponDraft());

function toDateTimeLocalInput(value) {
  if (!value) return '';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '';
  const pad = (n) => `${n}`.padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

function fromDateTimeLocalInput(value) {
  if (!value) return '';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '';
  return date.toISOString();
}

function resolveCouponImageSource(value) {
  const trimmed = typeof value === 'string' ? value.trim() : '';
  if (!trimmed) return '';
  if (/^data:/i.test(trimmed)) return trimmed;
  return resolveApiUrl(trimmed);
}

function normalizeCouponResponse(item) {
  if (!item || typeof item !== 'object') return null;
  const normalizedMatchIds = Array.isArray(item.match_ids ?? item.matchIds)
    ? (item.match_ids ?? item.matchIds).map(Number).filter((v) => Number.isFinite(v) && v > 0)
    : [];
  return {
    id: Number(item.id) || 0,
    title: typeof item.title === 'string' ? item.title : '',
    shortDesc: typeof item.short_desc === 'string' ? item.short_desc : (typeof item.shortDesc === 'string' ? item.shortDesc : ''),
    sponsorId: Number(item.sponsor_id ?? item.sponsorId) || 0,
    merchantId: Number(item.merchant_id ?? item.merchantId) || 0,
    matchIds: normalizedMatchIds,
    startDate: typeof item.start_date === 'string' ? item.start_date : (typeof item.startDate === 'string' ? item.startDate : ''),
    endDate: typeof item.end_date === 'string' ? item.end_date : (typeof item.endDate === 'string' ? item.endDate : ''),
    maxUses: Number(item.max_uses ?? item.maxUses) || 0,
    status: typeof item.status === 'string' ? item.status : '',
    imageUrl: typeof item.image_url === 'string' ? item.image_url : (typeof item.imageUrl === 'string' ? item.imageUrl : ''),
    totalViews: Number(item.total_views ?? item.totalViews) || 0,
    totalClaims: Number(item.total_claims ?? item.totalClaims) || 0,
    totalRedemptions: Number(item.total_redemptions ?? item.totalRedemptions) || 0,
  };
}

function toEditableCoupon(coupon) {
  const normalized = normalizeCouponResponse(coupon);
  if (!normalized || !normalized.id) return null;
  return {
    ...normalized,
    startDateInput: toDateTimeLocalInput(normalized.startDate),
    endDateInput: toDateTimeLocalInput(normalized.endDate),
    imagePreview: resolveCouponImageSource(normalized.imageUrl),
  };
}

function serializeCouponPayload(coupon) {
  const normalized = normalizeCouponResponse(coupon);
  return {
    title: normalized?.title?.trim() || '',
    short_desc: normalized?.shortDesc?.trim() || '',
    sponsor_id: normalized?.sponsorId || 0,
    merchant_id: normalized?.merchantId || normalized?.sponsorId || 0,
    match_ids: Array.isArray(coupon?.matchIds)
      ? coupon.matchIds.map(Number).filter((v) => Number.isFinite(v) && v > 0)
      : [],
    start_date: fromDateTimeLocalInput(coupon?.startDateInput) || normalized?.startDate,
    end_date: fromDateTimeLocalInput(coupon?.endDateInput) || normalized?.endDate,
    max_uses: Number.isFinite(normalized?.maxUses) ? normalized.maxUses : 0,
    status: normalized?.status?.trim() || 'draft',
    image_url: normalized?.imageUrl?.trim() || '',
    highlight: false,
  };
}

function couponMatchLabel(event) {
  const team1 = event?.team1_name || `Squadra ${event?.team1_id || '?'}`;
  const team2 = event?.team2_name || `Squadra ${event?.team2_id || '?'}`;
  const base = `${team1} vs ${team2}`;
  const rawDate = event?.start_datetime;
  if (!rawDate) return base;
  const parsed = new Date(rawDate);
  if (Number.isNaN(parsed.getTime())) return base;
  const dateLabel = parsed.toLocaleDateString('it-IT', { day: '2-digit', month: '2-digit' });
  return dateLabel ? `${base} ${dateLabel}` : base;
}

const filteredCouponEvents = computed(() => {
  const term = couponEventSearch.value.trim().toLowerCase();
  if (!term) return props.events;
  return props.events.filter((event) =>
    couponMatchLabel(event).toLowerCase().includes(term) ||
    String(event.start_datetime || '').toLowerCase().includes(term),
  );
});

function syncCouponImageSource(targetCoupon) {
  if (!targetCoupon) return;
  targetCoupon.imagePreview = resolveCouponImageSource(targetCoupon.imageUrl);
}

async function readFileAsDataUrl(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(typeof reader.result === 'string' ? reader.result : '');
    reader.onerror = () => reject(reader.error || new Error('Impossibile leggere il file'));
    reader.readAsDataURL(file);
  });
}

async function handleCouponImageFileChange(event, targetCoupon) {
  const [file] = event?.target?.files || [];
  if (!file || !targetCoupon) return;
  couponError.value = '';
  try {
    const dataUrl = await readFileAsDataUrl(file);
    if (dataUrl) {
      targetCoupon.imageUrl = dataUrl;
      targetCoupon.imagePreview = dataUrl;
    }
  } catch {
    couponError.value = 'Impossibile caricare l\'immagine del coupon.';
  } finally {
    if (event?.target) event.target.value = '';
  }
}

function clearCouponImage(targetCoupon) {
  if (!targetCoupon) return;
  targetCoupon.imageUrl = '';
  targetCoupon.imagePreview = '';
}

function resetNewCouponForm() {
  Object.assign(newCoupon, createEmptyCouponDraft());
  couponError.value = '';
  couponSuccess.value = '';
}

async function loadCoupons() {
  try {
    const { data } = await apiClient.get('/admin/coupons', props.authHeaders);
    coupons.value = Array.isArray(data)
      ? data.map(toEditableCoupon).filter((c) => c && c.id)
      : [];
  } catch (e) {
    console.error('Errore caricamento coupon', e);
  }
}

async function createCoupon() {
  if (isCreatingCoupon.value) return;
  couponError.value = '';
  couponSuccess.value = '';
  if (!newCoupon.title.trim()) {
    couponError.value = 'Inserisci un titolo per il coupon.';
    return;
  }
  if (!newCoupon.sponsorId) {
    couponError.value = 'Seleziona il partner associato.';
    return;
  }
  const payload = serializeCouponPayload(newCoupon);
  isCreatingCoupon.value = true;
  try {
    await apiClient.post('/admin/coupons', payload, props.authHeaders);
    couponSuccess.value = 'Coupon creato correttamente.';
    resetNewCouponForm();
    await loadCoupons();
  } catch (e) {
    if (e?.response?.status === 400) couponError.value = 'Controlla i dati inseriti per il coupon.';
  } finally {
    isCreatingCoupon.value = false;
  }
}

async function updateCouponEntry(coupon) {
  if (!coupon?.id || couponBeingSaved.value === coupon.id) return;
  couponError.value = '';
  couponSuccess.value = '';
  const payload = serializeCouponPayload(coupon);
  couponBeingSaved.value = coupon.id;
  try {
    await apiClient.put(`/admin/coupons/${coupon.id}`, payload, props.authHeaders);
    couponSuccess.value = 'Coupon aggiornato.';
    await loadCoupons();
  } catch (e) {
    if (e?.response?.status === 400) couponError.value = 'Impossibile salvare il coupon. Verifica i campi.';
  } finally {
    couponBeingSaved.value = 0;
  }
}

async function deleteCouponEntry(id) {
  if (!id || couponBeingDeleted.value === id) return;
  couponError.value = '';
  couponSuccess.value = '';
  couponBeingDeleted.value = id;
  try {
    await apiClient.delete(`/admin/coupons/${id}`, props.authHeaders);
    await loadCoupons();
  } catch (e) {
    if (e?.response?.status === 404) couponError.value = 'Coupon già rimosso.';
  } finally {
    couponBeingDeleted.value = 0;
  }
}

onMounted(loadCoupons);
</script>
