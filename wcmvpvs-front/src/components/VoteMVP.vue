<template>
  <div>
    <h1>Vota il tuo MVP</h1>
    <div class="grid">
      <div class="player-card" v-for="player in players" :key="player.id">
        <img :src="player.image" :alt="player.name" class="player-image" />
        <h3>{{ player.name }}</h3>
        <p>{{ player.role }} - #{{ player.number }}</p>
        <button @click="vote(player)">Vota</button>
      </div>
    </div>
    <!-- Conferma voto -->
    <div class="modal" v-if="showConfirm">
      <div class="modal-content">
        <p>Confermi il voto per {{ selectedPlayer?.name }}?</p>
        <button @click="confirmVote">Conferma</button>
        <button @click="cancelVote">Annulla</button>
      </div>
    </div>
    <!-- Codice e QR -->
    <div class="modal" v-if="showCode">
      <div class="modal-content">
        <h2>Voto registrato</h2>
        <p>Codice: {{ voteCode }}</p>
        <p>Firma HMAC: {{ signature }}</p>
        <img :src="qrUrl" alt="QR code" />
        <button @click="closeCode">Chiudi</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'

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

function vote(player) {
  selectedPlayer.value = player
  showConfirm.value = true
}

function cancelVote() {
  selectedPlayer.value = null
  showConfirm.value = false
}

async function confirmVote() {
  voteCode.value = Math.random().toString(36).slice(-8)
  signature.value = await generateHmac(voteCode.value, 'secret-key')
  qrUrl.value = `https://api.qrserver.com/v1/create-qr-code/?size=150x150&data=${encodeURIComponent(voteCode.value)}`
  showConfirm.value = false
  showCode.value = true
}

function closeCode() {
  showCode.value = false
}

async function generateHmac(message, secret) {
  const enc = new TextEncoder()
  const key = await crypto.subtle.importKey(
    'raw',
    enc.encode(secret),
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign']
  )
  const buf = await crypto.subtle.sign('HMAC', key, enc.encode(message))
  return Array.from(new Uint8Array(buf))
    .map(b => b.toString(16).padStart(2, '0'))
    .join('')
}
</script>

<style scoped>
.grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1rem;
}
.player-card {
  border: 1px solid #ccc;
  border-radius: 8px;
  padding: 1rem;
  display: flex;
  flex-direction: column;
  align-items: center;
}
.player-image {
  width: 100px;
  height: 100px;
  object-fit: cover;
  border-radius: 50%;
  margin-bottom: 0.5rem;
}

.modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
}

.modal-content {
  background: #fff;
  padding: 1rem;
  border-radius: 8px;
  text-align: center;
}
</style>
