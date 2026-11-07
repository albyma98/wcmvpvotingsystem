<template>
  <div class="container">
    <h4 class="text-center text-xl font-semibold" v-if="eventInfo">
      {{ eventInfo.team1 }} vs {{ eventInfo.team2 }}
    </h4>
    <p class="text-center" v-if="eventInfo">{{ eventInfo.location }}</p>
    <h5 class="my-4 text-center text-lg font-semibold">Vota il tuo MVP</h5>
    <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <article v-for="player in players" :key="player.id" class="rounded-xl bg-white p-5 text-center text-slate-900 shadow">
        <img :src="player.image" :alt="player.name" class="player-image" />
        <span class="player-name">{{ player.name }}</span>
        <p class="player-info">{{ player.role }} - #{{ player.number }}</p>
        <button class="vote-button mt-4" @click="vote(player)">Vota</button>
      </article>
    </div>
    <!-- Conferma voto -->
    <div class="custom-modal-overlay" v-if="showConfirm">
      <div class="custom-modal">
        <div class="modal-content">
          <p>Confermi il voto per {{ selectedPlayer?.name }}?</p>
        </div>
        <div class="mt-4 flex flex-wrap gap-2">
          <button class="vote-button" @click="confirmVote">Conferma</button>
          <button class="vote-button secondary" @click="cancelVote">Annulla</button>
        </div>
      </div>
    </div>
    <!-- Codice e QR -->
    <div class="custom-modal-overlay" v-if="showCode">
      <div class="custom-modal">
        <div class="text-center">
          <h5>Voto registrato</h5>
          <p>Codice: {{ voteCode }}</p>
          <img :src="qrUrl" alt="QR code" class="mx-auto" />
        </div>
        <div class="mt-4 text-center">
          <button class="vote-button" @click="closeCode">Chiudi</button>
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
      const voteResult = response.vote || {}
      const codeSource = voteResult.code || ''
      const qrSource = voteResult.qr_data || ''

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
  margin-inline: auto;
}
.vote-button { border-radius: .5rem; padding: .5rem 1rem; background: #1d4ed8; color: white; font-weight: 600; }
.vote-button:hover { background: #1e40af; }
.vote-button.secondary { background: #e2e8f0; color: #0f172a; }

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
  margin: 2rem auto 0;
  padding-inline: 1rem;
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
