<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'

const players = ref([])
const voted = ref(false)
const matchId = 1 // replace with dynamic ID if using a router

onMounted(async () => {
  const res = await axios.get(`/match/${matchId}`)
  players.value = res.data.players
  getUUID()
})

function getUUID() {
  let u = localStorage.getItem('uuid')
  if (!u) {
    u = crypto.randomUUID()
    localStorage.setItem('uuid', u)
    document.cookie = `uuid=${u}; path=/;`
  }
  return u
}

async function vote(playerId) {
  if (voted.value) return
  try {
    await axios.post(`/vote/${matchId}`, { player_id: playerId })
    voted.value = true
  } catch (err) {
    console.error(err)
  }
}
</script>

<template>
  <div class="vote">
    <div v-if="!voted">
      <div v-for="player in players" :key="player.id" class="player">
        <img :src="player.image_url" :alt="player.name" />
        <button @click="vote(player.id)">{{ player.name }}</button>
      </div>
    </div>
    <div v-else>
      <p>Grazie per il voto!</p>
    </div>
  </div>
</template>

<style scoped>
.vote {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
}
.player {
  margin: 10px;
  text-align: center;
}
img {
  width: 120px;
  height: 120px;
  object-fit: cover;
  border-radius: 50%;
}
button {
  display: block;
  margin-top: 5px;
  padding: 6px 10px;
}
</style>
