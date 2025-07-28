<script>
import { ref, onMounted } from "vue";
import VoteMVP from "./components/VoteMVP.vue";
import QRScanner from "./components/QRScanner.vue";
import AdminPortal from "./components/AdminPortal.vue";
import MvpSlider from "./components/MvpSlider.vue";
import EventList from "./components/EventList.vue";

export default {
  components: {
    VoteMVP,
    QRScanner,
    AdminPortal,
    MvpSlider,
    EventList,
  },
  setup() {
    const view = ref("events");
    const selectedEventId = ref(null);

    onMounted(() => {
      const params = new URLSearchParams(window.location.search);
      const ev = parseInt(params.get("eventID") || params.get("event"));
      if (ev) {
        selectedEventId.value = ev;
        view.value = "slider";
      } else {
        view.value = "events";
      }
    });

    function handleVoteEvent(id) {
      selectedEventId.value = id;
      view.value = "vote";
    }

    function handleEventSelect(id) {
      selectedEventId.value = id;
      view.value = "slider";
      const url = `?eventID=${id}`;
      window.history.replaceState({}, "", url);
    }

    return { view, selectedEventId, handleVoteEvent, handleEventSelect };
  },
};
</script>

<template>
  <div class="nav center-align">
        <button
      class="btn waves-effect"
      :class="{ 'blue lighten-1': view === 'events' }"
      @click="view = 'events'"
    >
      Eventi
    </button>
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
  <VoteMVP v-if="view === 'vote'" :event-id="selectedEventId" />
  <QRScanner v-else-if="view === 'scan'" />
  <AdminPortal v-else-if="view === 'admin'" @vote-event="handleVoteEvent" />
  <MvpSlider v-else-if="view === 'slider'" :event-id="selectedEventId" />
  <EventList v-else-if="view === 'events'" @select="handleEventSelect" />

</template>

<style scoped>
.nav {
  margin-bottom: 1rem;
  display: flex;
  gap: 0.5rem;
}
</style>