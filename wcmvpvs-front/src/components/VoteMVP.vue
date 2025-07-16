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

<script>
import { reactive, ref } from 'vue'
import axios from 'axios'

export default {
  setup() {

const players = reactive([
  { id: 1, name: 'Giocatore 1', role: 'Schiacciatore', number: 1, image: 'https://via.placeholder.com/100?text=1' },
  { id: 2, name: 'Giocatore 2', role: 'Opposto', number: 2, image: 'https://via.placeholder.com/100?text=2' },
  { id: 3, name: 'Giocatore 3', role: 'Centrale', number: 3, image: 'https://via.placeholder.com/100?text=3' },
  { id: 4, name: 'Giocatore 4', role: 'Palleggiatore', number: 4, image: 'https://via.placeholder.com/100?text=4' },
  { id: 5, name: 'Giocatore 5', role: 'Libero', number: 5, image: 'https://via.placeholder.com/100?text=5' },
  { id: 6, name: 'Giocatore 6', role: 'Schiacciatore', number: 6, image: 'https://via.placeholder.com/100?text=6' },
  { id: 7, name: 'Giocatore 7', role: 'Opposto', number: 7, image: 'https://via.placeholder.com/100?text=7' },
  { id: 8, name: 'Giocatore 8', role: 'Centrale', number: 8, image: 'https://via.placeholder.com/100?text=8' },
  { id: 9, name: 'Giocatore 9', role: 'Palleggiatore', number: 9, image: 'https://via.placeholder.com/100?text=9' },
  { id: 10, name: 'Giocatore 10', role: 'Libero', number: 10, image: 'https://via.placeholder.com/100?text=10' },
  { id: 11, name: 'Giocatore 11', role: 'Schiacciatore', number: 11, image: 'https://via.placeholder.com/100?text=11' },
  { id: 12, name: 'Giocatore 12', role: 'Opposto', number: 12, image: 'https://via.placeholder.com/100?text=12' },
])

const selectedPlayer = ref(null)
const showConfirm = ref(false)
const showCode = ref(false)
const voteCode = ref('')
const signature = ref('')
const qrUrl = ref('')

const api = axios.create({
  baseURL: 'http://0.0.0.0:3000',
})
api.interceptors.request.use((config) => {
  console.log('API Request', config.method, config.url, config.data)
  return config
})

api.interceptors.response.use(
  (response) => {
    console.log('API Response', response.status, response.config.url)
    return response
  },
  (error) => {
    console.error('API Error', error)
    return Promise.reject(error)
  }
)

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
    await api.post('/vote', { player_id: selectedPlayer.value.id })
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

    return {
      players,
      selectedPlayer,
      showConfirm,
      showCode,
      voteCode,
      signature,
      qrUrl,
      vote,
      cancelVote,
      confirmVote,
      closeCode,
    }
  },
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
