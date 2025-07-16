<script setup>
import { ref } from "vue";
import VoteMVP from "./components/VoteMVP.vue";
import QRScanner from "./components/QRScanner.vue";

const view = ref("vote");
</script>

<template>
  <div class="nav center-align">
    <button
      class="btn waves-effect"
      :class="{ 'blue lighten-1': view === 'vote' }"
      @click="view = 'vote'"
    >
      Vota
    </button>
    <button
      class="btn waves-effect"
      :class="{ 'blue lighten-1': view === 'scan' }"
      @click="view = 'scan'"
    >
      Scannerizza
    </button>

  </div>
  <VoteMVP v-if="view === 'vote'" />
  <QRScanner v-else />

</template>

<style scoped>
.nav {
  margin-bottom: 1rem;
  display: flex;
  gap: 0.5rem;
}
</style>