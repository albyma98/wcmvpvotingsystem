<template>
  <section class="checkout">
    <div v-if="!compact" class="dialog-heading">
      <button class="text-button" type="button" :disabled="busy" @click="$emit('back')">← Indietro</button>
      <h2 id="merch-dialog-title">{{ mode === 'offline' ? 'Conferma prenotazione' : 'Riepilogo acquisto' }}</h2>
      <p>{{ mode === 'offline' ? 'Pagherai direttamente alla società secondo le istruzioni qui sotto.' : 'Continuerai su Stripe. I metodi disponibili dipendono dal dispositivo e dal paese.' }}</p>
    </div>
    <form class="checkout-form" @submit.prevent="submit">
      <fieldset v-if="deliveryOptions.pickup && deliveryOptions.shipping" :disabled="busy || uncertain">
        <legend>Consegna</legend>
        <div class="choices">
          <label><input v-model="fulfillment" type="radio" value="pickup" /> Ritiro</label>
          <label><input v-model="fulfillment" type="radio" value="shipping" /> Spedizione</label>
        </div>
      </fieldset>
      <div v-if="fulfillment === 'pickup' || settings.delivery_instructions" class="inset">
        <strong>{{ fulfillment === 'pickup' ? 'Ritiro' : 'Spedizione' }}</strong>
        <p v-if="fulfillment === 'pickup' && settings.pickup_point">{{ settings.pickup_point }}</p>
        <p v-if="settings.delivery_instructions">{{ settings.delivery_instructions }}</p>
        <p v-if="settings.availability_text">{{ settings.availability_text }}</p>
      </div>
      <div v-if="mode === 'offline' && (settings.payment_methods || settings.payment_instructions)" class="inset">
        <strong>Pagamento alla società</strong>
        <p v-if="settings.payment_methods">{{ settings.payment_methods }}</p>
        <p v-if="settings.payment_instructions">{{ settings.payment_instructions }}</p>
      </div>
      <fieldset v-if="mode === 'offline' || fulfillment === 'shipping'" :disabled="busy || uncertain">
        <legend>Dati per {{ fulfillment === 'shipping' ? 'la consegna' : 'la prenotazione' }}</legend>
        <label class="field">Nome e cognome
          <input v-model.trim="contact.name" required autocomplete="name" maxlength="120" />
        </label>
        <label v-if="fulfillment === 'shipping'" class="field">Indirizzo di spedizione
          <textarea v-model.trim="contact.address" required autocomplete="street-address" maxlength="500" rows="2" placeholder="Via, numero civico, CAP, città e paese"></textarea>
        </label>
        <label v-if="contactField === 'email'" class="field">Email
          <input v-model.trim="contact.email" type="email" autocomplete="email" maxlength="254" required />
        </label>
        <label v-else class="field">Telefono
          <input v-model.trim="contact.phone" type="tel" autocomplete="tel" maxlength="32" placeholder="+39…" required />
        </label>
        <button v-if="mode === 'offline' && settings.contact_requirement === 'either'" class="text-button" type="button" @click="preferredContact = contactField === 'email' ? 'phone' : 'email'">
          {{ contactField === 'email' ? 'Usa il telefono' : "Usa l’email" }}
        </button>
      </fieldset>
      <div class="inset quote" aria-live="polite">
        <p v-if="loadingQuote" role="status">Aggiornamento del totale…</p>
        <p v-else-if="quoteError" role="alert">{{ quoteError }} <button class="text-button" type="button" @click="loadQuote">Riprova</button></p>
        <template v-else-if="quote">
          <ul>
            <li v-for="item in quote.items" :key="merchItemKey(item)">
              <span>{{ item.quantity }} × {{ item.name }}<small v-if="item.variant_name"> · {{ item.variant_name }}</small></span>
              <span>{{ merchMoney(item.quantity * item.unit_cents) }}</span>
            </li>
          </ul>
          <div v-if="fulfillment === 'shipping'" class="total"><span>Spedizione</span><span>{{ merchMoney(quote.shipping_cents) }}</span></div>
          <div class="total grand-total"><strong>{{ mode === 'offline' ? 'Totale da pagare al club' : 'Totale' }}</strong><strong>{{ merchMoney(quote.total_cents) }}</strong></div>
        </template>
      </div>
      <div v-if="settings.seller_info || settings.cancellation_policy" class="seller">
        <p v-if="settings.seller_info"><strong>Venditore</strong> {{ settings.seller_info }}</p>
        <p v-if="settings.cancellation_policy"><strong>Condizioni di annullamento</strong> {{ settings.cancellation_policy }}</p>
      </div>
      <p v-if="error" class="error" role="alert">{{ error }}</p>
      <button class="primary" type="submit" :disabled="!quote || busy || loadingQuote">
        {{ busy ? 'Attendi…' : uncertain ? 'Riprova la stessa richiesta' : mode === 'offline' ? 'Conferma prenotazione' : 'Vai al pagamento sicuro' }}
      </button>
    </form>
  </section>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';
import { fetchFanProfile } from '../api';
import { createMerchOrder, fetchMerchQuote, merchRequestError } from './merchApi';
import { merchItemKey, merchMoney } from './shopModel';

const props = defineProps({
  slug: { type: String, required: true },
  mode: { type: String, required: true },
  lines: { type: Array, required: true },
  settings: { type: Object, required: true },
  deliveryOptions: { type: Object, required: true },
  compact: Boolean,
});
const emit = defineEmits(['back', 'created', 'recoverable-order', 'busy-change']);
const fulfillment = ref(props.deliveryOptions.pickup ? 'pickup' : 'shipping');
const contact = reactive({ name: '', email: '', phone: '', address: '' });
const preferredContact = ref('email');
const contactField = computed(() =>
  props.mode === 'online' ? 'email' :
    props.settings.contact_requirement === 'either' ? preferredContact.value : props.settings.contact_requirement,
);
const quote = ref(null);
const quoteError = ref('');
const loadingQuote = ref(false);
const busy = ref(false);
const uncertain = ref(false);
const error = ref('');
let quoteRequest;
let submittedInput;
let disposed = false;

const quoteInput = () => ({ mode: props.mode, fulfillment: fulfillment.value, items: props.lines });

async function loadQuote() {
  quoteRequest?.abort();
  const request = new AbortController();
  quoteRequest = request;
  loadingQuote.value = true;
  quote.value = null;
  quoteError.value = '';
  try {
    const data = await fetchMerchQuote(props.slug, quoteInput(), request.signal);
    if (!request.signal.aborted) quote.value = data;
  } catch (cause) {
    if (!request.signal.aborted) quoteError.value = merchRequestError(cause, 'Impossibile calcolare il totale. Riprova.');
  } finally {
    if (!request.signal.aborted) loadingQuote.value = false;
  }
}

async function submit() {
  if (busy.value || !quote.value) return;
  if (!globalThis.crypto?.randomUUID) {
    error.value = 'Per proseguire apri lo Shop tramite una connessione sicura.';
    return;
  }
  busy.value = true;
  emit('busy-change', true);
  error.value = '';
  const collectContact = props.mode === 'offline' || fulfillment.value === 'shipping';
  submittedInput ||= {
    ...quoteInput(),
    expected_total_cents: quote.value.total_cents,
    request_id: globalThis.crypto.randomUUID(),
    ...(collectContact ? {
      name: contact.name,
      [contactField.value]: contactField.value === 'phone' ? contact.phone.replace(/[\s().-]/g, '') : contact.email,
    } : {}),
    ...(fulfillment.value === 'shipping' ? { address: contact.address } : {}),
  };
  try {
    const response = await createMerchOrder(props.slug, submittedInput);
    if (!disposed) emit('created', response);
  } catch (cause) {
    if (disposed) return;
    const status = cause?.response?.status;
    uncertain.value = !status || status >= 500 || (uncertain.value && status === 429);
    if (uncertain.value && cause?.response?.data?.order) emit('recoverable-order', cause.response.data);
    if (!uncertain.value) submittedInput = undefined;
    if (status === 409) {
      error.value = merchRequestError(cause, 'Prezzo o disponibilità aggiornati. Controlla il riepilogo prima di confermare.');
      await loadQuote();
    } else {
      error.value = merchRequestError(cause, 'La conferma è ancora in sospeso. Riprova la stessa richiesta da qui.');
    }
  } finally {
    if (!disposed) {
      busy.value = false;
      emit('busy-change', false);
    }
  }
}

watch(fulfillment, () => {
  submittedInput = undefined;
  error.value = '';
  void loadQuote();
});
onMounted(async () => {
  void loadQuote();
  if (props.mode !== 'offline' || props.settings.contact_requirement === 'email') return;
  const profile = await fetchFanProfile();
  if (disposed || uncertain.value || busy.value || !profile.ok || !profile.data?.registered || !profile.data.user?.phone_verified_at) return;
  if (!contact.phone) contact.phone = profile.data.user.phone || '';
  if (!contact.email && contact.phone) preferredContact.value = 'phone';
});
onBeforeUnmount(() => {
  disposed = true;
  quoteRequest?.abort();
  emit('busy-change', false);
});
</script>
