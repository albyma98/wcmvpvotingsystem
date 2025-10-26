<template>
  <div class="validation-shell">
    <div class="validation-card" :class="statusClass">
      <div class="validation-header">
        <span class="validation-icon">{{ leadingEmoji }}</span>
        <h1 class="validation-title">{{ titleMessage }}</h1>
      </div>
      <p v-if="detailMessage" class="validation-detail">{{ detailMessage }}</p>
      <p v-if="secondaryDetail" class="validation-secondary">{{ secondaryDetail }}</p>
      <button v-if="showRetry" class="retry-button" type="button" @click="validate()">
        Riprova
      </button>
    </div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { validateTicketStatus } from '../api';

const status = ref('loading');
const message = ref('Verifica del ticket in corso…');
const detailMessage = ref('');
const secondaryDetail = ref('');
const alreadyRedeemed = ref(false);
const currentSearch = ref(typeof window !== 'undefined' ? window.location.search : '');

const showRetry = computed(() => status.value === 'error');

const leadingEmoji = computed(() => {
  if (status.value === 'success' && alreadyRedeemed.value) {
    return 'ℹ️';
  }
  if (status.value === 'success') {
    return '🎉';
  }
  if (status.value === 'error') {
    return '⚠️';
  }
  return '⏳';
});

const titleMessage = computed(() => {
  if (status.value === 'loading') {
    return 'Verifica in corso…';
  }
  if (status.value === 'success') {
    return alreadyRedeemed.value ? 'QR valido (già riscattato)' : 'QR valido!';
  }
  return message.value;
});

const statusClass = computed(() => {
  if (status.value === 'success') {
    return alreadyRedeemed.value ? 'is-info' : 'is-success';
  }
  if (status.value === 'error') {
    return 'is-error';
  }
  return 'is-loading';
});

function parseTicketParams(search) {
  const params = new URLSearchParams(search || '');
  const eventRaw = params.get('e');
  const eventId = Number.parseInt(eventRaw ?? '', 10);
  const code = (params.get('c') || '').trim();
  const signature = (params.get('s') || '').trim();

  return {
    eventId: Number.isFinite(eventId) && eventId > 0 ? eventId : undefined,
    code,
    signature,
  };
}

function buildErrorMessage(code) {
  const messages = {
    missing_parameters: {
      title: 'QR incompleto',
      detail: 'I dati necessari per validare questo QR non sono completi.',
    },
    invalid_event_id: {
      title: 'Evento non valido',
      detail: 'Non riconosciamo l\'evento associato a questo QR.',
    },
    invalid_signature: {
      title: 'QR non valido',
      detail: 'La firma digitale del ticket non è corretta.',
    },
    ticket_not_found: {
      title: 'Ticket inesistente',
      detail: 'Non troviamo alcun ticket con questo codice.',
    },
    stored_signature_mismatch: {
      title: 'QR non valido',
      detail: 'Il ticket non è più valido perché la firma non coincide.',
    },
    redemption_signature_mismatch: {
      title: 'QR non valido',
      detail: 'Il ticket è stato modificato e non può essere validato.',
    },
    internal_error: {
      title: 'Errore temporaneo',
      detail: 'Si è verificato un problema interno. Riprova tra qualche istante.',
    },
    unknown_error: {
      title: 'Errore sconosciuto',
      detail: 'Si è verificato un errore inaspettato durante la verifica.',
    },
  };

  return (
    messages[code] || {
      title: 'Verifica non riuscita',
      detail: 'Non siamo riusciti a verificare il QR code. Riprova più tardi.',
    }
  );
}

async function validate() {
  status.value = 'loading';
  message.value = 'Verifica del ticket in corso…';
  detailMessage.value = '';
  secondaryDetail.value = '';
  alreadyRedeemed.value = false;

  const { eventId, code, signature } = parseTicketParams(currentSearch.value);
  if (!eventId || !code || !signature) {
    status.value = 'error';
    const info = buildErrorMessage('missing_parameters');
    message.value = info.title;
    detailMessage.value = info.detail;
    return;
  }

  const response = await validateTicketStatus({ eventId, code, signature });
  if (response.ok) {
    status.value = 'success';
    detailMessage.value = 'Il ticket è valido e può essere utilizzato per il ritiro del premio.';
    if (response.data?.already_redeemed) {
      alreadyRedeemed.value = true;
      secondaryDetail.value = 'Attenzione: questo ticket risulta già riscattato.';
    }
    return;
  }

  status.value = 'error';
  const info = buildErrorMessage(response.error);
  message.value = info.title;
  detailMessage.value = info.detail;
}

function handlePopState() {
  currentSearch.value = typeof window !== 'undefined' ? window.location.search : '';
}

onMounted(() => {
  validate();
  if (typeof window !== 'undefined') {
    window.addEventListener('popstate', handlePopState, { passive: true });
  }
});

onBeforeUnmount(() => {
  if (typeof window !== 'undefined') {
    window.removeEventListener('popstate', handlePopState);
  }
});

watch(currentSearch, () => {
  validate();
});
</script>

<style scoped>
.validation-shell {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: radial-gradient(circle at 20% 20%, rgba(56, 189, 248, 0.18), transparent 55%),
    radial-gradient(circle at 80% 0%, rgba(139, 92, 246, 0.24), transparent 52%),
    linear-gradient(180deg, #020617 0%, #0f172a 50%, #020617 100%);
  padding: 2.5rem 1.5rem;
}

.validation-card {
  max-width: 420px;
  width: 100%;
  padding: 2.75rem 2.5rem;
  border-radius: 1.75rem;
  text-align: center;
  color: #f8fafc;
  background: rgba(15, 23, 42, 0.9);
  box-shadow: 0 30px 60px rgba(2, 6, 23, 0.45), inset 0 0 0 1px rgba(148, 163, 184, 0.2);
  backdrop-filter: blur(18px);
  transition: transform 0.3s ease, box-shadow 0.3s ease;
}

.validation-card.is-loading {
  box-shadow: 0 18px 40px rgba(30, 64, 175, 0.25), inset 0 0 0 1px rgba(129, 140, 248, 0.4);
}

.validation-card.is-success {
  background: rgba(22, 101, 52, 0.9);
  box-shadow: 0 24px 48px rgba(16, 185, 129, 0.3), inset 0 0 0 1px rgba(134, 239, 172, 0.4);
}

.validation-card.is-info {
  background: rgba(30, 64, 175, 0.9);
  box-shadow: 0 24px 48px rgba(96, 165, 250, 0.32), inset 0 0 0 1px rgba(191, 219, 254, 0.4);
}

.validation-card.is-error {
  background: rgba(159, 18, 57, 0.92);
  box-shadow: 0 24px 48px rgba(248, 113, 113, 0.35), inset 0 0 0 1px rgba(254, 202, 202, 0.45);
}

.validation-header {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 1rem;
}

.validation-icon {
  font-size: 3.25rem;
}

.validation-title {
  font-size: 1.85rem;
  margin: 0;
  font-weight: 700;
  letter-spacing: 0.03em;
}

.validation-detail {
  margin: 0.75rem 0 0;
  font-size: 1.05rem;
  line-height: 1.6;
  color: rgba(241, 245, 249, 0.9);
}

.validation-secondary {
  margin-top: 0.75rem;
  font-size: 0.95rem;
  color: rgba(226, 232, 240, 0.8);
}

.retry-button {
  margin-top: 1.75rem;
  padding: 0.9rem 1.75rem;
  border-radius: 999px;
  border: none;
  background: rgba(15, 23, 42, 0.95);
  color: #f8fafc;
  font-weight: 600;
  letter-spacing: 0.04em;
  cursor: pointer;
  box-shadow: 0 12px 24px rgba(2, 6, 23, 0.45);
  transition: transform 0.2s ease, box-shadow 0.2s ease, background 0.2s ease;
}

.retry-button:hover {
  transform: translateY(-2px);
  background: rgba(15, 23, 42, 1);
  box-shadow: 0 16px 32px rgba(2, 6, 23, 0.55);
}

.retry-button:active {
  transform: translateY(0);
}

@media (max-width: 480px) {
  .validation-card {
    padding: 2.25rem 1.75rem;
  }

  .validation-title {
    font-size: 1.6rem;
  }
}
</style>
