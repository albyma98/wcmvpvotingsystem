<template>
  <details class="recovery">
    <summary>Ricevi il link privato via SMS</summary>
    <p class="muted">Verifica il numero associato alla prenotazione per ricevere il collegamento su quel telefono.</p>
    <p v-if="sent" role="status">Numero verificato. Riceverai il link privato via SMS.</p>
    <form v-else-if="awaitingCode" class="recovery-form" @submit.prevent="send('verify')">
      <label class="field">Codice SMS
        <input v-model.trim="code" autocomplete="one-time-code" inputmode="numeric" pattern="[0-9]{4,10}" maxlength="10" required />
      </label>
      <button class="secondary" type="submit" :disabled="busy">Verifica e invia il link</button>
      <button class="text-button" type="button" :disabled="busy" @click="send('start')">Reinvia il codice</button>
    </form>
    <button v-else class="secondary" type="button" :disabled="busy" @click="send('start')">{{ busy ? 'Attendi…' : 'Invia il codice di verifica' }}</button>
    <p v-if="error" class="error" role="alert">{{ error }}</p>
  </details>
</template>

<script setup>
import { onBeforeUnmount, ref } from 'vue';
import { merchRequestError, requestMerchRecovery } from './merchApi';

const props = defineProps({ slug: String, orderId: String, token: String });
const awaitingCode = ref(false);
const code = ref('');
const sent = ref(false);
const busy = ref(false);
const error = ref('');
const controller = new AbortController();

async function send(step) {
  if (busy.value) return;
  busy.value = true;
  error.value = '';
  try {
    await requestMerchRecovery(props.slug, props.orderId, props.token, step, step === 'verify' ? { code: code.value } : null, controller.signal);
    if (step === 'start') {
      awaitingCode.value = true;
      code.value = '';
    } else {
      sent.value = true;
    }
  } catch (cause) {
    if (!controller.signal.aborted) error.value = merchRequestError(cause, 'Impossibile completare la verifica. Riprova.');
  } finally {
    busy.value = false;
  }
}
onBeforeUnmount(() => controller.abort());
</script>
