<script>
import { ref } from "vue";
import VoteMVP from "./components/VoteMVP.vue";
import QRScanner from "./components/QRScanner.vue";
import AdminPortal from "./components/AdminPortal.vue";

export default {
  components: {
    VoteMVP,
    QRScanner,
    AdminPortal,
  },
  setup() {
    const view = ref("vote");
    return { view };
  },
};
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
    <button
      class="btn waves-effect"
      :class="{ 'blue lighten-1': view === 'admin' }"
      @click="view = 'admin'"
    >
      Admin
    </button>
  </div>
  <VoteMVP v-if="view === 'vote'" />
  <QRScanner v-else-if="view === 'scan'" />
  <AdminPortal v-else />

</template>

<style scoped>
.nav {
  margin-bottom: 1rem;
  display: flex;
  gap: 0.5rem;
}
</style>