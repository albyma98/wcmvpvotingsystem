<template>
  <div class="mvp-slider">
    <div class="header">
      <div class="title">VOTE YOUR <strong>MVP</strong></div>
      <div class="teams">{{ team1Name }} - {{ team2Name }}</div>
    </div>

    <div class="slider">
      <button class="nav prev" @click="prev">&lt;</button>
      <div class="card" @click="vote">
        <img class="player-img" :src="current.image || placeholder" alt="player" />
        <div class="player-overlay">
          <div class="player-name">{{ current.name }}</div>
          <div class="player-number">#{{ current.number }}</div>
        </div>
      </div>
      <button class="nav next" @click="next">&gt;</button>
    </div>
    
    <!-- Conferma voto -->
    <div class="custom-modal-overlay" v-if="showConfirm">
      <div class="custom-modal">
        <div class="modal-content">
          <p>Confermi il voto per {{ selectedPlayer?.name }}?</p>
        </div>
        <div class="modal-footer">
          <button class="btn waves-effect" @click="confirmVote">Conferma</button>
          <button class="btn-flat" @click="cancelVote">Annulla</button>
        </div>
      </div>
    </div>

    <!-- Codice e QR -->
    <div class="custom-modal-overlay" v-if="showCode">
      <div class="custom-modal">
        <div class="modal-content center-align">
          <h5>Voto registrato</h5>
          <p>Codice: {{ voteCode }}</p>
          <p>Firma HMAC: {{ signature }}</p>
          <img :src="qrUrl" alt="QR code" />
        </div>
        <div class="modal-footer center-align">
          <button class="btn waves-effect" @click="closeCode">Chiudi</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import axios from 'axios'
import { getOrCreateDeviceId } from '../deviceId'

const props = defineProps({
  eventId: { type: Number, default: null },
  players: { type: Array, default: () => [] },
  team1: { type: String, default: '' },
  team2: { type: String, default: '' },
  url: { type: String, default: 'example.com' },
})


const index = ref(0)
const placeholder = 'https://via.placeholder.com/150?text=Player'
const loadedPlayers = ref([])
const team1Name = ref('')
const team2Name = ref('')
const selectedPlayer = ref(null)
const showConfirm = ref(false)
const showCode = ref(false)
const voteCode = ref('')
const signature = ref('')
const qrUrl = ref('')

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

function vote() {
  if (current.value && current.value.id != null) {
    selectedPlayer.value = current.value
    showConfirm.value = true
  }
}

function cancelVote() {
  selectedPlayer.value = null
  showConfirm.value = false
}

async function confirmVote() {
  try {
    const { data: voteResult } = await api.post('/vote', {
      player_id: selectedPlayer.value.id,
      event_id: props.eventId,
      device_id: getOrCreateDeviceId(),
    })
    voteCode.value = voteResult.code
    signature.value = voteResult.signature
    qrUrl.value = `https://api.qrserver.com/v1/create-qr-code/?size=150x150&data=${encodeURIComponent(voteResult.qr_data)}`

    showConfirm.value = false
    showCode.value = true
  } catch (err) {
    console.error('confirmVote() error', err)
  }
}

function closeCode() {
  showCode.value = false
}
</script>

<style scoped>
.mvp-slider {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  padding: 1rem;
  box-sizing: border-box;
  gap: 2rem;
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
  width: 80%;
  max-width: 360px;
  height: 420px;
  cursor: pointer;
  position: relative;
  overflow: hidden;
}
.player-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.player-overlay {
  position: absolute;
  top: 70%;
  left: 0;
  width: 100%;
  transform: translateY(-50%);
  background: rgba(0, 0, 0, 0.5);
  color: #fff;
  text-align: center;
  padding: 0.5rem 0;
}
.player-name {
  text-shadow: 0 0 5px rgba(0, 0, 0, 0.7);
  font-weight: bold;
}
.player-number {
  font-size: 1rem;
  text-shadow: 0 0 5px rgba(0, 0, 0, 0.7);
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
    width: 320px;
  }
}

.custom-modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.custom-modal {
  background: white;
  color: #000;
  padding: 1rem;
  border-radius: 8px;
  max-width: 400px;
  width: 90%;
}
</style>