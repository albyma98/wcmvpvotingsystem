<template>
  <section class="card">
    <header class="section-header">
      <h2>Squadre</h2>
    </header>
    <form @submit.prevent="createTeam" class="form-inline">
      <input v-model.trim="newTeamName" type="text" placeholder="Nome squadra" required />
      <button class="btn primary" type="submit">Aggiungi</button>
    </form>
    <ul class="item-list compact">
      <li v-for="team in teams" :key="team.id" class="item">
        <span>{{ team.name }}</span>
        <button class="btn danger" type="button" @click="deleteTeam(team.id)">Elimina</button>
      </li>
    </ul>
  </section>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import { apiClient } from '../../api';

const props = defineProps({
  authHeaders: { type: Object, required: true },
  isSuperAdmin: { type: Boolean, default: false },
});

const emit = defineEmits(['updated']);

const teams = ref([]);
const newTeamName = ref('');

async function loadTeams() {
  try {
    const { data } = await apiClient.get('/teams', props.authHeaders);
    teams.value = Array.isArray(data) ? data : [];
    emit('updated', teams.value);
  } catch (e) {
    console.error('Errore caricamento squadre', e);
  }
}

async function createTeam() {
  if (!newTeamName.value) return;
  try {
    await apiClient.post('/teams', { name: newTeamName.value }, props.authHeaders);
    newTeamName.value = '';
    await loadTeams();
  } catch (e) {
    console.error('Errore creazione squadra', e);
  }
}

async function deleteTeam(id) {
  try {
    await apiClient.delete(`/teams/${id}`, props.authHeaders);
    await loadTeams();
  } catch (e) {
    console.error('Errore eliminazione squadra', e);
  }
}

onMounted(loadTeams);
</script>
