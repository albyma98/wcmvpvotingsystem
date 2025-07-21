<template>
  <div class="container">
    <h4 class="center-align">Vota il tuo MVP</h4>
    <div class="row">
      <div class="col s12 m6 l4" v-for="player in players" :key="player.id">
        <div class="card hoverable">
          <div class="card-image center-align">
            <img :src="player.image" :alt="player.name" class="player-image" />
          </div>
          <div class="card-content center-align">
            <span class="player-name">{{ player.name }}</span>
            <p class="player-info">{{ player.role }} - #{{ player.number }}</p>
          </div>
          <div class="card-action center-align">
            <button class="btn waves-effect" @click="vote(player)">Vota</button>
          </div>
        </div>
      </div>
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
import { ref, watchEffect } from 'vue'
import axios from 'axios'

const props = defineProps({
  eventId: Number,
})

const players = ref([])
const selectedPlayer = ref(null)
const showConfirm = ref(false)
const showCode = ref(false)
const voteCode = ref('')
const signature = ref('')
const qrUrl = ref('')

const api = axios.create({
  baseURL: 'http://localhost:3000',
})
async function loadPlayers() {
  if (!props.eventId) {
    players.value = []
    return
  }
  const events = (await api.get('/events')).data
  const ev = events.find((e) => e.id === props.eventId)
  if (!ev) {
    players.value = []
    return
  }
  const allPlayers = (await api.get('/players')).data
  players.value = allPlayers
    .filter((p) => p.team_id === ev.team1_id)
    .map((p) => ({
      id: p.id,
      name: p.first_name + ' ' + p.last_name,
      role: p.role,
      number: p.jersey_number,
      image: `https://via.placeholder.com/100?text=${p.jersey_number}`,
    }))
}

watchEffect(loadPlayers)

function vote(player) {
  console.log('vote() selected', player)
  selectedPlayer.value = player
  showConfirm.value = true
}

function cancelVote() {
  selectedPlayer.value = null
  showConfirm.value = false
}

async function confirmVote() {
  try {
    console.log('confirmVote() sending vote for', selectedPlayer.value.id)
    await api.post('/vote', { player_id: selectedPlayer.value.id, event_id: props.eventId, device_id: 'web' })
    console.log('confirmVote() requesting ticket')
    const ticketRes = await api.post('/ticket')
    const ticket = ticketRes.data
    console.log('confirmVote() ticket received', ticket)
    voteCode.value = ticket.code
    signature.value = ticket.signature
    qrUrl.value = `https://api.qrserver.com/v1/create-qr-code/?size=150x150&data=${encodeURIComponent(ticket.qr_data)}`

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
.player-image {
  width: 150px;
  height: 150px;
  object-fit: cover;
  border-radius: 50%;
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

.player-name {
  color: #000;
  font-weight: bold;
  display: block;
  margin-top: 0.5rem;
}

.player-info {
  color: #666;
  margin-top: 0.25rem;
}
</style>