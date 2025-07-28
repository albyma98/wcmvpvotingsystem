<template>
  <div class="mvp-slider">
    <div class="header">
      <div class="title">VOTE YOUR <strong>MVP</strong></div>
      <div class="teams">{{ team1Name }} - {{ team2Name }}</div>
    </div>

    <div class="slider">
      <button class="nav prev" @click="prev">&lt;</button>
      <div class="card" @click="emitVote">
        <img class="player-img" :src="current.image || placeholder" alt="player" />
        <div class="player-name">{{ current.name }}</div>
        <div class="player-number">#{{ current.number }}</div>
      </div>
      <button class="nav next" @click="next">&gt;</button>
    </div>

    <div class="browser-bar">
      <div class="url">{{ url }}</div>
      <div class="icons">
        <span class="material-icons">home</span>
        <span class="material-icons">thumb_up</span>
        <span class="material-icons">menu_book</span>
        <span class="material-icons">devices</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import axios from 'axios'

const props = defineProps({
  eventId: { type: Number, default: null },
  players: { type: Array, default: () => [] },
  team1: { type: String, default: '' },
  team2: { type: String, default: '' },
  url: { type: String, default: 'example.com' },
})

const emit = defineEmits(['vote'])
const index = ref(0)
const placeholder = 'https://via.placeholder.com/150?text=Player'
const loadedPlayers = ref([])
const team1Name = ref('')
const team2Name = ref('')

const api = axios.create({ baseURL: 'http://localhost:3000' })

async function loadPlayers(id) {
  if (id) {
    const events = (await api.get('/events')).data
    const ev = events.find(e => e.id === id)
    if (!ev) {
      loadedPlayers.value = []
      team1Name.value = ''
      team2Name.value = ''
      return
    }
    const teams = (await api.get('/teams')).data
    const t1 = teams.find(t => t.id === ev.team1_id)
    const t2 = teams.find(t => t.id === ev.team2_id)
    team1Name.value = t1 ? t1.name : ''
    team2Name.value = t2 ? t2.name : ''
    const allPlayers = (await api.get('/players')).data
    loadedPlayers.value = allPlayers
      .filter(p => p.team_id === ev.team1_id)
      .map(p => ({
        id: p.id,
        name: p.first_name + ' ' + p.last_name,
        number: p.jersey_number,
        image: p.image_url || `https://via.placeholder.com/150?text=${p.jersey_number}`,
      }))
  } else {
    loadedPlayers.value = props.players
    team1Name.value = props.team1
    team2Name.value = props.team2
  }
}

watch(() => props.eventId, loadPlayers, { immediate: true })
onMounted(() => loadPlayers(props.eventId))

const current = computed(() => {
  return loadedPlayers.value.length ? loadedPlayers.value[index.value] : { name: '', number: '', image: placeholder, id: null }
})

function next() {
  if (!loadedPlayers.value.length) return
  index.value = (index.value + 1) % loadedPlayers.value.length
}

function prev() {
  if (!loadedPlayers.value.length) return
  index.value = (index.value + loadedPlayers.value.length - 1) % loadedPlayers.value.length
}

function emitVote() {
  if (current.value && current.value.id != null) {
    emit('vote', current.value.id)
  }
}
</script>

<style scoped>
.mvp-slider {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  align-items: center;
  padding: 1rem;
  box-sizing: border-box;
}
.header {
  text-align: center;
}
.title {
  font-size: 1.5rem;
}
.teams {
  font-size: 0.9rem;
  color: #666;
}
.slider {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  margin: 1rem 0;
}
.nav {
  background: none;
  border: none;
  color: inherit;
  font-size: 2rem;
  padding: 0 1rem;
  cursor: pointer;
}
.card {
  background: #fff;
  color: #000;
  border-radius: 10px;
  padding: 1rem;
  text-align: center;
  width: 70%;
  max-width: 320px;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  align-items: center;
}
.player-img {
  width: 100%;
  max-width: 200px;
  border-radius: 50%;
  object-fit: cover;
}
.player-name {
  margin-top: 0.5rem;
  font-weight: bold;
}
.player-number {
  color: #555;
}
.browser-bar {
  width: 100%;
  background: #eee;
  color: #000;
  border-radius: 20px;
  padding: 0.3rem 1rem;
  font-size: 0.8rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.browser-bar .icons {
  display: flex;
  gap: 0.5rem;
}
.browser-bar .material-icons {
  font-size: 1.2rem;
}
@media (min-width: 600px) {
  .title {
    font-size: 2rem;
  }
  .card {
    width: 300px;
  }
}
</style>