<script setup>
import { ref } from 'vue'
import VoteMVP from './components/VoteMVP.vue'
import QRScanner from './components/QRScanner.vue'

const view = ref('vote')
</script>

<template>
  <div class="nav">
    <button @click="view = 'vote'" :class="{ active: view === 'vote' }">Vota</button>
    <button @click="view = 'scan'" :class="{ active: view === 'scan' }">Scannerizza</button>
  </div>
  <VoteMVP v-if="view === 'vote'" />
  <QRScanner v-else />
</template>

<style scoped>
.nav {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 1rem;
}
.nav button {
  padding: 0.5rem 1rem;
}
.nav button.active {
  font-weight: bold;
}
</style>
