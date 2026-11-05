<template>
  <section class="card">
    <header class="section-header">
      <h2>Elenco squadre</h2>
      <p>Aggiungi le squadre da associare alle partite e ai giocatori.</p>
    </header>
    <form @submit.prevent="createTeam" class="form-inline">
      <label class="admin-team-name">Nome della squadra<input v-model.trim="newTeamName" type="text" placeholder="Es. Prima squadra" required /></label>
      <button class="btn primary" type="submit">Aggiungi</button>
    </form>
    <ul class="item-list compact">
      <li v-for="team in teams" :key="team.id" class="item">
        <span>{{ team.name }}</span>
        <button class="btn danger" type="button" @click="deleteTeam(team.id)">Elimina</button>
      </li>
    </ul>
    <p v-if="!teams.length" class="admin-empty-state">Non ci sono ancora squadre. Aggiungi la prima per iniziare a organizzare le tue partite.</p>
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

<style scoped>
.admin-team-name { display: grid; gap: 8px; flex: 1; min-width: min(100%, 220px); font-size: 13px; font-weight: 600; }
.form-inline { align-items: flex-end; }
.admin-empty-state { margin: 0; padding: 32px 20px; text-align: center; color: var(--text-muted); font-size: 13px; border: 1px dashed var(--border); border-radius: 10px; }
</style>
