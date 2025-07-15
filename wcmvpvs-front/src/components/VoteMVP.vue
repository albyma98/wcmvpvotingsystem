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
  </div>
</template>

<script setup>
import { reactive } from 'vue'

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

function vote(player) {
  alert(`Hai votato ${player.name}`)
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
</style>
