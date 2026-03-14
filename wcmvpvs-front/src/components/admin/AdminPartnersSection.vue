<template>
  <section class="card">
    <header class="section-header">
      <h2>Partners</h2>
      <p>
        Crea le credenziali per gli esercenti che convalidano i coupon. Ogni
        partner accede con username e password dedicati.
      </p>
    </header>
    <form @submit.prevent="createPartner" class="form-grid">
      <label>
        Nome partner
        <input v-model.trim="newPartner.name" type="text" placeholder="Es. Bar dello Stadio" />
      </label>
      <label>
        Username
        <input v-model.trim="newPartner.username" type="text" placeholder="Credenziale di accesso" required />
      </label>
      <label>
        Password
        <input v-model="newPartner.password" type="password" autocomplete="new-password" required />
      </label>
      <button class="btn primary" type="submit">Aggiungi partner</button>
    </form>
    <ul class="item-list compact" v-if="partners.length">
      <li v-for="partner in partners" :key="partner.id" class="item">
        <div class="item-body">
          <div>
            <strong>{{ partner.displayName }}</strong>
            <span class="muted"> • {{ partner.username }}</span>
            <p class="muted small" v-if="partner.createdAtLabel">Creato il {{ partner.createdAtLabel }}</p>
          </div>
          <div class="partner-actions">
            <label class="inline-input">
              <span>Nuova password</span>
              <input v-model="partner.newPassword" type="password" placeholder="Aggiorna password" />
            </label>
            <button
              class="btn secondary"
              type="button"
              @click="updatePartnerPassword(partner)"
              :disabled="partner.isUpdating || !partner.newPassword"
            >
              <span v-if="partner.isUpdating">Salvataggio…</span>
              <span v-else>Aggiorna</span>
            </button>
            <button
              class="btn danger"
              type="button"
              @click="deletePartner(partner.id)"
              :disabled="partner.isDeleting"
            >
              <span v-if="partner.isDeleting">Eliminazione…</span>
              <span v-else>Elimina</span>
            </button>
          </div>
        </div>
      </li>
    </ul>
    <p v-else class="muted text-center">Nessun partner configurato.</p>
  </section>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue';
import { apiClient } from '../../api';

const props = defineProps({
  authHeaders: { type: Object, required: true },
  isSuperAdmin: { type: Boolean, default: false },
});

const emit = defineEmits(['updated']);

const partners = ref([]);
const newPartner = reactive({ name: '', username: '', password: '' });

function normalizePartnerResponse(item) {
  if (!item || typeof item !== 'object') return null;
  const createdAt = item.created_at ? new Date(item.created_at) : null;
  return {
    id: Number(item.id) || 0,
    username: item.username || '',
    displayName: item.username || '',
    createdAtLabel: createdAt ? createdAt.toLocaleString('it-IT') : '',
    newPassword: '',
    isUpdating: false,
    isDeleting: false,
  };
}

async function loadPartners() {
  try {
    const { data } = await apiClient.get('/admin/partners', props.authHeaders);
    partners.value = Array.isArray(data)
      ? data.map(normalizePartnerResponse).filter((p) => p && p.id)
      : [];
    emit('updated', partners.value);
  } catch (e) {
    console.error('Errore caricamento partner', e);
  }
}

async function createPartner() {
  const username = (newPartner.username || newPartner.name).trim();
  const password = newPartner.password;
  if (!username || !password) return;
  try {
    await apiClient.post('/admin/partners', { username, password }, props.authHeaders);
    Object.assign(newPartner, { name: '', username: '', password: '' });
    await loadPartners();
  } catch (e) {
    console.error('Errore creazione partner', e);
  }
}

async function updatePartnerPassword(partner) {
  if (!partner?.id || partner.isUpdating) return;
  const trimmed = (partner.newPassword || '').trim();
  if (!trimmed) return;
  partner.isUpdating = true;
  try {
    await apiClient.put(`/admin/partners/${partner.id}`, { password: trimmed }, props.authHeaders);
    partner.newPassword = '';
    await loadPartners();
  } finally {
    partner.isUpdating = false;
  }
}

async function deletePartner(id) {
  if (!id) return;
  const partner = partners.value.find((p) => p.id === id);
  if (!partner || partner.isDeleting) return;
  partner.isDeleting = true;
  try {
    await apiClient.delete(`/admin/partners/${id}`, props.authHeaders);
    await loadPartners();
  } finally {
    partner.isDeleting = false;
  }
}

onMounted(loadPartners);
</script>
