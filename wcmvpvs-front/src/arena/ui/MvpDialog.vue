<script setup>
import { ref } from "vue";
import Icon from "../ArenaIcon.vue";
import { DEMO_PLAYERS } from "../mvp/players.js";
const props = defineProps({ fan: Object, votedPlayer: Object });
const emit = defineEmits(["vote", "open"]);
const dialog = ref(null),
  candidate = ref(null);
function open() {
  candidate.value = props.votedPlayer?.id || null;
  emit("open");
  dialog.value?.showModal();
}
function close() {
  dialog.value?.close();
}
function submit() {
  if (candidate.value && !props.votedPlayer) {
    emit("vote", candidate.value);
    close();
  }
}
defineExpose({ open, close });
</script>
<template>
  <dialog
    ref="dialog"
    class="mvp-dialog"
    aria-labelledby="mvp-dialog-title"
    @click="
      (event) => {
        if (event.target === dialog) close();
      }
    "
  >
    <div class="mvp-dialog-inner">
      <button class="dialog-close" aria-label="Chiudi voto MVP" @click="close">
        <Icon name="close" />
      </button>
      <span class="mvp-dialog-symbol"><Icon name="trophy" /></span>
      <p class="eyebrow">IL PROTAGONISTA LO SCEGLI TU</p>
      <h2 id="mvp-dialog-title">Chi è il tuo MVP?</h2>
      <p class="mvp-dialog-intro">
        Voti come <strong>{{ fan.nickname }}</strong
        >.<br />Tre giocatori demo. Una scelta che fa rumore.
      </p>
      <div class="mvp-player-list">
        <button
          v-for="player in DEMO_PLAYERS"
          :key="player.id"
          class="mvp-player"
          :class="{ chosen: candidate === player.id }"
          :aria-pressed="candidate === player.id"
          :disabled="Boolean(votedPlayer)"
          @click="candidate = player.id"
        >
          <span class="player-shirt" :style="{ '--jersey': player.color }">{{
            player.number
          }}</span
          ><span
            ><strong>{{ player.name }}</strong
            ><small>{{ player.role }} · #{{ player.number }}</small></span
          ><span class="player-selection"
            ><Icon v-if="candidate === player.id" name="check"
          /></span>
        </button>
      </div>
      <p v-if="votedPlayer" class="mvp-already-voted" role="status">
        <Icon name="check" />Hai già votato per {{ votedPlayer.name }}.
      </p>
      <button v-else class="confirm-mvp" :disabled="!candidate" @click="submit">
        <Icon name="trophy" />Conferma voto MVP<Icon name="arrow" />
      </button>
      <p class="mvp-local-note">
        Voto dimostrativo salvato su questo dispositivo.<br />Non partecipa alle
        votazioni delle partite reali.
      </p>
    </div>
  </dialog>
</template>
