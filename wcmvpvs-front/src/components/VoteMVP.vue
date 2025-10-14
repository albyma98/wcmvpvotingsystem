<template>
  <div class="container">
    <h4 class="center-align" v-if="eventInfo">
      {{ eventInfo.team1 }} vs {{ eventInfo.team2 }}
    </h4>
    <p class="center-align" v-if="eventInfo">{{ eventInfo.location }}</p>
    <h5 class="center-align">Vota il tuo MVP</h5>
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
import { apiClient, vote as submitVote } from '../api'

const props = defineProps({
  eventId: Number,
})

const players = ref([])
const eventInfo = ref(null)
const selectedPlayer = ref(null)
const showConfirm = ref(false)
const showCode = ref(false)
const voteCode = ref('')
const qrUrl = ref('')

async function loadPlayers() {
  if (!props.eventId) {
    players.value = []
    eventInfo.value = null
    return
  }
  const events = (await apiClient.get('/events')).data
  const ev = events.find((e) => e.id === props.eventId)
  if (!ev) {
    players.value = []
    eventInfo.value = null
    return
  }
  const teams = (await apiClient.get('/teams')).data
  const t1 = teams.find((t) => t.id === ev.team1_id)
  const t2 = teams.find((t) => t.id === ev.team2_id)
  eventInfo.value = {
    team1: t1 ? t1.name : '',
    team2: t2 ? t2.name : '',
    location: ev.location,
  }
  const allPlayers = (await apiClient.get('/players')).data
  players.value = allPlayers
    .filter((p) => p.team_id === ev.team1_id)
    .map((p) => ({
      id: p.id,
      name: p.first_name + ' ' + p.last_name,
      role: p.role,
      number: p.jersey_number,
      image: p.image_url || `https://via.placeholder.com/100?text=${p.jersey_number}`,
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
    if (!selectedPlayer.value) {
      return
    }
    console.log('confirmVote() sending vote for', selectedPlayer.value.id)
    const response = await submitVote({
      eventId: props.eventId,
      playerId: selectedPlayer.value.id,
    })
    if (response?.ok) {
      const voteResult = response.vote
      const ticket = response.ticket
      const codeSource = voteResult?.code || ticket?.code || ''
      const qrSource = voteResult?.qr_data || ticket?.qr_data || ''

      voteCode.value = codeSource
      qrUrl.value = qrSource
        ? `https://api.qrserver.com/v1/create-qr-code/?size=150x150&data=${encodeURIComponent(qrSource)}`
        : ''
      showConfirm.value = false
      showCode.value = true
    } else {
      console.error('confirmVote() ticket generation failed', response)
    }
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
.card {
  margin: 1rem 0;
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
.container {
  margin-top: 2rem;
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
