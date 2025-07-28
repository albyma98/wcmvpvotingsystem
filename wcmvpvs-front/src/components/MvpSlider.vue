<template>
  <div class="mvp-slider">
    <div class="header">
      <div class="title">VOTE YOUR <strong>MVP</strong></div>
      <div class="teams">{{ team1 }} - {{ team2 }}</div>
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
import { ref, computed } from 'vue'

const props = defineProps({
  players: { type: Array, default: () => [] },
  team1: { type: String, default: '' },
  team2: { type: String, default: '' },
  url: { type: String, default: 'example.com' },
})

const emit = defineEmits(['vote'])
const index = ref(0)
const placeholder = 'https://via.placeholder.com/150?text=Player'

const current = computed(() => {
  return props.players.length ? props.players[index.value] : { name: '', number: '', image: placeholder, id: null }
})

function next() {
  if (!props.players.length) return
  index.value = (index.value + 1) % props.players.length
}

function prev() {
  if (!props.players.length) return
  index.value = (index.value + props.players.length - 1) % props.players.length
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