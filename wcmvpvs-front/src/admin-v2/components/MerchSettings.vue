<template>
  <form class="v2-stack" @submit.prevent="save">
    <V2Panel title="Vetrina" description="Gestisci la disponibilità dello Shop nel portale della società.">
      <div class="v2-grid-2">
        <label class="v2-check"><input v-model="draft.enabled" type="checkbox" /> Attiva Shop</label>
        <label class="v2-check"><input v-model="draft.visible" type="checkbox" /> Mostra Shop nella navigazione del portale</label>
      </div>
    </V2Panel>

    <V2Panel title="Modalità di vendita" description="Le singole schede prodotto possono restringere le modalità attive.">
      <div class="v2-stack">
        <div class="v2-grid-2">
          <label class="v2-check"><input v-model="draft.online_enabled" type="checkbox" /> Pagamento online</label>
          <label class="v2-check"><input v-model="draft.offline_enabled" type="checkbox" /> Prenotazione con pagamento alla società</label>
        </div>
        <div class="v2-grid-2">
          <label class="v2-field">
            Modalità in evidenza al checkout
            <select v-model="draft.default_mode">
              <option value="offline">Prenotazione alla società</option>
              <option value="online">Pagamento online</option>
            </select>
            <small>Se entrambe le modalità sono attive, il tifoso può scegliere.</small>
          </label>
          <div class="v2-notice">
            <strong>Stripe Connect</strong>
            <p>{{ stripeText }}</p>
            <p v-if="settings.stripe_account_id"><code>{{ settings.stripe_account_id }}</code></p>
            <small>Serve un account con Dashboard Stripe completa, pagamenti e accrediti attivi.</small>
          </div>
        </div>
      </div>
    </V2Panel>

    <V2Panel title="Prenotazione e pagamento alla società">
      <div class="v2-stack">
        <div class="v2-grid-2">
          <label class="v2-field">Durata della prenotazione (ore)<input v-model.number="draft.reservation_hours" type="number" min="1" max="720" required /></label>
          <label class="v2-field">Recapito richiesto<select v-model="draft.contact_requirement">
            <option value="either">Email o telefono</option><option value="email">Email</option><option value="phone">Telefono</option>
          </select></label>
        </div>
        <label class="v2-check"><input v-model="draft.reserve_offline" type="checkbox" /> Riserva stock alla ricezione della prenotazione</label>
        <label class="v2-field">Metodi di pagamento accettati<input v-model.trim="draft.payment_methods" placeholder="Es. contanti o POS al ritiro" maxlength="1000" /></label>
        <label class="v2-field">Istruzioni di pagamento<textarea v-model.trim="draft.payment_instructions" maxlength="2000" /></label>
        <label class="v2-field">Tempi indicativi di disponibilità<input v-model.trim="draft.availability_text" maxlength="500" /></label>
      </div>
    </V2Panel>

    <V2Panel title="Ritiro e consegna">
      <div class="v2-stack">
        <div class="v2-grid-2">
          <label class="v2-check"><input v-model="draft.pickup_enabled" type="checkbox" /> Ritiro in sede</label>
          <label class="v2-check"><input v-model="draft.shipping_enabled" type="checkbox" /> Spedizione per i prodotti abilitati</label>
          <label class="v2-field">Punto di ritiro<input v-model.trim="draft.pickup_point" maxlength="500" /></label>
          <label class="v2-field">Spese di spedizione (€)<input v-model="shipping" inputmode="decimal" :disabled="!draft.shipping_enabled" /></label>
        </div>
        <label class="v2-field">Istruzioni di consegna<textarea v-model.trim="draft.delivery_instructions" maxlength="2000" /></label>
      </div>
    </V2Panel>

    <V2Panel title="Venditore e conferme">
      <div class="v2-stack">
        <label class="v2-field">Email di contatto<input v-model.trim="draft.contact_email" type="email" maxlength="254" /></label>
        <label class="v2-field">Identità e informazioni del venditore<textarea v-model.trim="draft.seller_info" maxlength="2000" placeholder="Ragione sociale, contatti e riferimenti all’informativa" /></label>
        <label class="v2-field">Politica di annullamento<textarea v-model.trim="draft.cancellation_policy" maxlength="2000" /></label>
        <label class="v2-field">Testo di conferma<textarea v-model.trim="draft.confirmation_text" maxlength="2000" /></label>
        <label class="v2-field">Note del club<textarea v-model.trim="draft.notes" maxlength="2000" /></label>
        <label v-if="'sms_notifications' in draft" class="v2-check"><input v-model="draft.sms_notifications" type="checkbox" /> Abilita conferme e recupero prenotazioni via SMS con recapito verificato</label>
      </div>
    </V2Panel>
    <p v-if="error" class="v2-error" role="alert">{{ error }}</p>
    <div class="v2-toolbar"><button class="v2-button v2-primary" :disabled="busy">{{ busy ? 'Salvataggio…' : 'Salva impostazioni' }}</button></div>
  </form>
</template>

<script setup>
import { computed, inject, reactive, ref } from 'vue';
import V2Panel from './shared/V2Panel.vue';
import { euroInput, merchRequestError, toCents } from '../merchModel';

const props = defineProps({
  settings: { type: Object, required: true },
  persist: { type: Function, required: true },
});
const emit = defineEmits(['saved']);
const c = inject('admin-v2');
const { stripe_account_id, stripe_status, ...editable } = props.settings;
const draft = reactive(editable);
const shipping = ref(euroInput(editable.shipping_cents ?? 0));
const error = ref('');
const busy = computed(() => c.busy('merch-settings'));
const stripeText = computed(() => ({
  READY: 'Account verificato. I pagamenti online possono essere attivati.',
  UNAVAILABLE: 'Account al momento indisponibile. Verifica la configurazione Connect con ArenaBoostX.',
  NOT_CONFIGURED: 'Nessun account associato. Contatta ArenaBoostX per configurare gli incassi.',
}[props.settings.stripe_status] || 'Lo stato dell’account non è disponibile.'));

function save() {
  if (busy.value) return;
  error.value = '';
  return c.run('merch-settings', async () => {
    try {
      const settings = await props.persist({
        ...draft,
        shipping_cents: draft.shipping_enabled ? toCents(shipping.value) : draft.shipping_cents,
      });
      const { stripe_account_id, stripe_status, ...editable } = settings;
      Object.assign(draft, editable);
      shipping.value = euroInput(settings.shipping_cents ?? 0);
      emit('saved', settings);
    } catch (failure) {
      error.value = merchRequestError(failure);
    }
  }, 'Impostazioni salvate.');
}
</script>
