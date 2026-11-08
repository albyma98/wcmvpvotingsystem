<template>
  <div class="mvp-carousel" role="group" aria-label="Giocatori candidati all’MVP">
    <div
      ref="track"
      class="mvp-carousel__track"
      @scroll.passive="updateFocus"
      @keydown.left.prevent="move(-1)"
      @keydown.right.prevent="move(1)"
    >
      <div v-for="(player, index) in players" :key="player.id" class="mvp-carousel__slot">
        <button
          type="button"
          class="mvp-carousel__player"
          :class="{ 'is-focused': focusedIndex === index }"
          :aria-label="playerLabel(player)"
          :aria-pressed="selectedPlayerId === player.id"
          :disabled="disabled"
          @focus="centerOnKeyboardFocus($event, index)"
          @click="select(player, index)"
        >
          <FanPlayerCard
            :player="player"
            :club-logo-url="clubLogoUrl"
            :selected="selectedPlayerId === player.id"
            selected-label="Scelto"
            aria-hidden="true"
          />
        </button>
      </div>
    </div>
    <div v-if="players.length > 1" class="mvp-carousel__controls">
      <button type="button" aria-label="Giocatore precedente" :aria-disabled="focusedIndex === 0" @click="move(-1)">‹</button>
      <span aria-label="Posizione nel carosello">{{ focusedIndex + 1 }} / {{ players.length }}</span>
      <button type="button" aria-label="Giocatore successivo" :aria-disabled="focusedIndex === players.length - 1" @click="move(1)">›</button>
    </div>
  </div>
</template>

<script setup>
import { onBeforeUnmount, onMounted, ref } from 'vue';
import FanPlayerCard from './FanPlayerCard.vue';

const props = defineProps({
  players: { type: Array, required: true },
  initialPlayerId: { type: Number, default: null },
  selectedPlayerId: { type: Number, default: null },
  clubLogoUrl: { type: String, default: '' },
  disabled: { type: Boolean, default: false },
});
const emit = defineEmits(['select']);
const track = ref(null);
const focusedIndex = ref(Math.max(0, props.players.findIndex((player) => player.id === props.initialPlayerId)));
let scrollFrame = 0;

function playerLabel(player) {
  const number = player.number !== '' && player.number != null ? `maglia ${player.number}` : '';
  return [player.fullName || player.name, number, player.role].filter(Boolean).join(', ');
}

function centerPlayer(index, smooth = true) {
  const element = track.value;
  const slot = element?.children[index];
  if (!slot) return;
  const left = slot.offsetLeft + (slot.offsetWidth - element.clientWidth) / 2;
  element.scrollTo({ left, behavior: smooth && !window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 'smooth' : 'auto' });
}

function updateFocus() {
  if (scrollFrame) return;
  scrollFrame = requestAnimationFrame(() => {
    scrollFrame = 0;
    const element = track.value;
    const first = element?.children[0];
    const second = element?.children[1];
    if (first && second) {
      const step = second.offsetLeft - first.offsetLeft;
      if (step > 0) focusedIndex.value = Math.min(props.players.length - 1, Math.max(0, Math.round(element.scrollLeft / step)));
    }
  });
}

function move(direction) {
  const next = focusedIndex.value + direction;
  if (next < 0 || next >= props.players.length) return;
  centerPlayer(next);
  const focused = track.value?.querySelector('button:focus');
  if (focused && !props.disabled) track.value.children[next].firstElementChild.focus({ preventScroll: true });
}

function centerOnKeyboardFocus(event, index) {
  if (event.currentTarget.matches(':focus-visible')) centerPlayer(index);
}

function select(player, index) {
  if (props.disabled) return;
  emit('select', player);
  centerPlayer(index);
}

onMounted(() => centerPlayer(focusedIndex.value, false));
onBeforeUnmount(() => cancelAnimationFrame(scrollFrame));
</script>

<style scoped>
.mvp-carousel { min-width: 0; }
.mvp-carousel__track {
  --player-width: clamp(125px, 34vw, 142px);
  position: relative;
  display: flex;
  gap: 10px;
  padding: 6px max(0px, calc((100% - var(--player-width)) / 2));
  overflow-x: auto;
  scroll-snap-type: x mandatory;
  scrollbar-width: none;
}
.mvp-carousel__track::-webkit-scrollbar { display: none; }
.mvp-carousel__slot {
  flex: 0 0 var(--player-width);
  min-width: 0;
  scroll-snap-align: center;
}
.mvp-carousel__player {
  display: block;
  width: 100%;
  padding: 2px;
  border: 2px solid transparent;
  border-radius: 15px;
  background: transparent;
  cursor: pointer;
  font: inherit;
  transform: scale(.93);
  transition: transform .18s ease, border-color .18s ease;
}
.mvp-carousel__player:disabled { cursor: default; }
.mvp-carousel__player.is-focused { transform: none; border-color: #9e6c1d; }
.mvp-carousel__player[aria-pressed='true'] { border-color: var(--red-text); background: var(--red-soft); }
.mvp-carousel__player :deep(.fan-player-card),
.mvp-carousel__player :deep(.mvp-player-card) {
  aspect-ratio: auto;
  height: clamp(160px, 43vw, 174px);
}
.mvp-carousel__player :deep(.mvp-player-card) {
  background:
    repeating-linear-gradient(130deg, transparent 0 16px, #ffffff24 17px, transparent 18px 29px),
    radial-gradient(ellipse at 70% 15%, #ffffff75, transparent 60%),
    linear-gradient(145deg, var(--red-soft), var(--red) 65%, var(--red-deep));
}
.mvp-carousel__player :deep(.mvp-player-number) {
  padding: 4px 5px;
  border-radius: 7px;
  color: var(--ink);
  background: var(--card);
  text-shadow: none;
  font-size: 19px;
}
.mvp-carousel__player :deep(.mvp-player-initials) { color: var(--ink); background: var(--red-soft); }
.mvp-carousel__player :deep(.mvp-player-caption) {
  padding: 12% 7% 13%;
  background: linear-gradient(transparent, var(--red-soft) 23%, var(--card));
}
.mvp-carousel__player :deep(.mvp-player-caption strong) { color: var(--ink); font-size: .75rem; }
.mvp-carousel__player :deep(.mvp-player-role) { color: var(--ink); font-size: .6875rem; }
.mvp-carousel__player :deep(.mvp-player-selected) { font-size: .625rem; }
.mvp-carousel__controls {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 8px;
  height: 30px;
  margin: -2px 0 2px;
  color: var(--ink);
  font-size: .75rem;
  font-variant-numeric: tabular-nums;
}
.mvp-carousel__controls button {
  position: relative;
  z-index: 0;
  display: grid;
  width: 44px;
  height: 30px;
  place-items: center;
  border: 0;
  border-radius: 11px;
  background: transparent;
  color: var(--red-text);
  font: 700 1.4rem/1 system-ui;
  cursor: pointer;
}
.mvp-carousel__controls button[aria-disabled='true'] { opacity: .42; cursor: default; }
.mvp-carousel__player:focus-visible,
.mvp-carousel__controls button:focus-visible { outline: 2px solid var(--red-text); outline-offset: 2px; }
@media (prefers-reduced-motion: reduce) {
  .mvp-carousel__player { transition: none; }
}
</style>
