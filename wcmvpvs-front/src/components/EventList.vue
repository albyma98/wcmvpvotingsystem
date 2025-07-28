<template>
  <div class="event-list">
    <h4>Seleziona evento</h4>
    <ul>
      <li v-for="e in events" :key="e.id">
        <a :href="`?eventID=${e.id}`">
          {{ teamName(e.team1_id) }} vs {{ teamName(e.team2_id) }} - {{ formatDate(e.start_datetime) }}
        </a>
      </li>
    </ul>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'

const api = axios.create({ baseURL: 'http://localhost:3000' })

const events = ref([])
const teams = ref([])

async function load() {
  events.value = (await api.get('/events')).data
  teams.value = (await api.get('/teams')).data
}

function teamName(id) {
  const t = teams.value.find(t => t.id === id)
  return t ? t.name : ''
}

function formatDate(dt) {
  return dt ? dt.slice(0, 10) : ''
}

onMounted(load)
</script>

<style scoped>
.event-list ul {
  list-style: none;
  padding: 0;
}
.event-list li {
  margin: 0.3rem 0;
}
</style>