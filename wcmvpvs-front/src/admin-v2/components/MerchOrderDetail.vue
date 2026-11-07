<template>
  <V2Dialog :open="true" drawer :title="'Ordine ' + order.code" :busy="busy" @close="close">
    <div class="v2-stack">
      <div class="v2-toolbar">
        <span class="v2-badge">{{ paymentLabel(order.payment_status) }}</span>
        <span class="v2-badge">{{ fulfillmentLabel(order.fulfillment_status) }}</span>
      </div>
      <div class="v2-grid-2">
        <section>
          <h3>Cliente</h3>
          <p>{{ order.name }}</p>
          <p v-if="order.email">{{ order.email }}</p>
          <p v-if="order.phone">{{ order.phone }}</p>
        </section>
        <section>
          <h3>{{ order.fulfillment === 'shipping' ? 'Spedizione' : 'Ritiro' }}</h3>
          <p v-if="order.address">{{ address }}</p>
          <p>{{ timestamp(order.created_at) }}</p>
          <small v-if="order.expires_at && ['PENDING_ONLINE', 'UNPAID_OFFLINE'].includes(order.payment_status)">Scadenza {{ timestamp(order.expires_at) }}</small>
        </section>
      </div>
      <p v-if="order.notes" class="v2-notice">{{ order.notes }}</p>
      <div class="v2-table-wrap">
        <table class="v2-table">
          <thead><tr><th scope="col">Articolo</th><th scope="col">Qtà</th><th scope="col">Unitario</th></tr></thead>
          <tbody>
            <tr v-for="item in order.items" :key="item.product_id + '/' + item.variant_id">
              <td><strong>{{ item.name }}</strong><br v-if="item.variant_name || item.sku" /><small>{{ [item.variant_name, item.sku].filter(Boolean).join(' · ') }}</small></td>
              <td>{{ item.quantity }}</td><td>{{ money(item.unit_cents, order.currency) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div>
        <p v-if="order.shipping_cents">Spedizione {{ money(order.shipping_cents, order.currency) }}</p>
        <strong>Totale {{ money(order.total_cents, order.currency) }}</strong>
      </div>
      <fieldset v-if="actions.length" class="v2-stack">
        <legend>Operazioni</legend>
        <div v-if="!pending" class="v2-toolbar">
          <button v-for="[action, label] in actions" :key="action" type="button" class="v2-button"
            :class="{ 'v2-danger': ['cancel','refund'].includes(action) }" @click="select(action, label)">{{ label }}</button>
        </div>
        <form v-else ref="form" class="v2-stack" @submit.prevent="apply">
          <h3>{{ pending.label }}</h3>
          <p v-if="pending.action === 'refund'">Richiedi un rimborso integrale al cliente tramite Stripe.</p>
          <p v-if="pending.action === 'cancel'">La prenotazione verrà annullata e le unità riservate torneranno disponibili.</p>
          <template v-if="pending.action === 'record_payment'">
            <p>Registra l’importo effettivamente incassato dalla società.</p>
            <div class="v2-grid-2">
              <label class="v2-field">Importo ricevuto (€)<input v-model="pending.amount" inputmode="decimal" required /></label>
              <label class="v2-field">Metodo di pagamento<input v-model.trim="pending.method" maxlength="100" placeholder="Es. contanti, POS" required /></label>
            </div>
            <label class="v2-field">Riferimento<input v-model.trim="pending.reference" maxlength="160" /></label>
          </template>
          <label class="v2-field">Note operatore<textarea v-model.trim="pending.notes" rows="2" maxlength="500" /></label>
          <div class="v2-toolbar">
            <button class="v2-button v2-primary" :disabled="busy">{{ busy ? 'Operazione in corso…' : 'Conferma operazione' }}</button>
            <button class="v2-button" type="button" :disabled="busy" @click="cancelAction">Indietro</button>
          </div>
        </form>
        <p v-if="error" class="v2-error" role="alert">{{ error }}</p>
      </fieldset>
      <details v-if="order.payments?.length">
        <summary>Registro pagamenti</summary>
        <ul class="v2-stack">
          <li v-for="(payment, index) in order.payments" :key="index">
            <strong>{{ money(payment.amount_cents, order.currency) }} · {{ payment.method || payment.source }}</strong>
            <p>{{ paymentLabel(payment.status) }} · {{ timestamp(payment.created_at) }}</p>
            <small>{{ [payment.actor, payment.reference].filter(Boolean).join(' · ') }}</small>
          </li>
        </ul>
      </details>
      <details>
        <summary>Storico operazioni</summary>
        <ul v-if="order.events?.length" class="v2-stack">
          <li v-for="(event, index) in order.events" :key="index">
            <strong>{{ eventLabel(event.type) }}</strong>
            <p>{{ timestamp(event.created_at) }}<template v-if="event.actor"> · {{ event.actor }}</template></p>
            <small v-if="event.notes">{{ event.notes }}</small>
          </li>
        </ul>
        <p v-else class="v2-muted">Nessuna operazione registrata.</p>
      </details>
    </div>
  </V2Dialog>
</template>

<script setup>
import { computed, inject, ref } from 'vue';
import V2Dialog from './shared/V2Dialog.vue';
import { euroInput, fulfillmentLabel, merchRequestError, money, orderActions, paymentLabel, timestamp, toCents } from '../merchModel';

const props = defineProps({
  order: { type: Object, required: true },
  persist: { type: Function, required: true },
});
const emit = defineEmits(['close', 'updated']);
const c = inject('admin-v2');
const form = ref(null);
const actions = computed(() => orderActions(props.order));
const address = computed(() => typeof props.order.address === 'string'
  ? props.order.address : Object.values(props.order.address || {}).filter(Boolean).join(', '));
const pending = ref(null);
const error = ref('');
const busy = ref(false);

const eventLabels = {
  RESERVATION_CREATED: 'Prenotazione ricevuta',
  RESERVATION_CONFIRMED: 'Prenotazione confermata',
  PREPARING: 'Ordine in preparazione',
  READY_FOR_PICKUP: 'Pronto al ritiro',
  PAYMENT_COMPLETED: 'Pagamento confermato da Stripe',
  PAYMENT_OFFLINE_RECORDED: 'Incasso registrato dalla società',
  REFUND_REQUESTED: 'Rimborso richiesto',
  COMPLETED: 'Ordine consegnato',
  CANCELLED: 'Ordine annullato',
  EXPIRED: 'Riserva scaduta',
  REFUNDED: 'Rimborso confermato',
};
const eventLabel = type => eventLabels[type] || fulfillmentLabel(paymentLabel(type));

function select(action, label) {
  error.value = '';
  pending.value = { action, label, notes: '', method: '', reference: '', amount: euroInput(props.order.total_cents) };
}

function cancelAction() {
  c.discardDraft(form.value);
  pending.value = null;
  error.value = '';
}

function close() {
  cancelAction();
  emit('close');
}

async function apply() {
  if (busy.value) return;
  busy.value = true;
  error.value = '';
  try {
    const { action, notes, method, reference, amount } = pending.value;
    const payload = { action, notes, confirmed: true };
    if (action === 'record_payment') {
      const amountCents = toCents(amount);
      if (amountCents !== props.order.total_cents) throw new Error('L’importo ricevuto deve corrispondere al totale della prenotazione.');
      Object.assign(payload, { method, reference, amount_cents: amountCents });
    }
    const updated = await props.persist(props.order.id, payload);
    cancelAction();
    emit('updated', updated);
  } catch (failure) {
    error.value = merchRequestError(failure);
  } finally {
    busy.value = false;
  }
}
</script>
