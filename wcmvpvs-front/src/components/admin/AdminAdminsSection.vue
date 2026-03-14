<template>
  <section class="card">
    <header class="section-header">
      <h2>Utenti amministratori</h2>
    </header>
    <form @submit.prevent="createAdmin" class="form-grid">
      <input v-model.trim="newAdmin.username" type="text" placeholder="Username" required />
      <input v-model="newAdmin.password" type="password" placeholder="Password" required />
      <input v-model.trim="newAdmin.role" type="text" placeholder="Ruolo (es. staff)" />
      <button class="btn primary" type="submit">Aggiungi</button>
    </form>
    <ul class="item-list compact">
      <li v-for="admin in admins" :key="admin.id" class="item">
        <div>
          <strong>{{ admin.username }}</strong>
          <span class="muted"> • {{ admin.role || 'staff' }}</span>
        </div>
        <button class="btn danger" type="button" @click="deleteAdmin(admin.id)">Elimina</button>
      </li>
    </ul>
  </section>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue';
import { apiClient } from '../../api';

const props = defineProps({
  authHeaders: { type: Object, required: true },
  isSuperAdmin: { type: Boolean, default: false },
});

const admins = ref([]);
const newAdmin = reactive({ username: '', password: '', role: '' });

async function loadAdmins() {
  try {
    const { data } = await apiClient.get('/admins', props.authHeaders);
    admins.value = Array.isArray(data) ? data : [];
  } catch (e) {
    console.error('Errore caricamento admin', e);
  }
}

async function createAdmin() {
  try {
    await apiClient.post('/admins', { ...newAdmin }, props.authHeaders);
    Object.assign(newAdmin, { username: '', password: '', role: '' });
    await loadAdmins();
  } catch (e) {
    console.error('Errore creazione admin', e);
  }
}

async function deleteAdmin(id) {
  try {
    await apiClient.delete(`/admins/${id}`, props.authHeaders);
    await loadAdmins();
  } catch (e) {
    console.error('Errore eliminazione admin', e);
  }
}

onMounted(loadAdmins);
</script>
