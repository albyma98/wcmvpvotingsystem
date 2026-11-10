<template>
  <div class="mvp-carousel" role="group" aria-label="Giocatori candidati all’MVP">
    <div
      ref="track"
      class="mvp-carousel__track"
      @scroll.passive="updateFocus"
      @scrollend="settleScroll"
      @pointerdown="startBrowsing"
      @wheel.passive="startBrowsing"
      @keydown.left.prevent="move(-1)"
      @keydown.right.prevent="move(1)"
    >
      <div
        v-for="({ player, index, cycle }, slot) in slots"
        :key="`${cycle}:${player.id}`"
        class="mvp-carousel__slot"
        :aria-hidden="cycle !== 0 || undefined"
        @mousedown="cycle !== 0 && $event.preventDefault()"
      >
        <div
          class="mvp-carousel__player"
          :class="{
            'is-focused': focusedIndex === index && selectedPlayerId === player.id,
            'is-selected': selectedPlayerId === player.id,
            'has-confirm': showsConfirm(player, slot),
          }"
        >
          <button
            type="button"
            class="mvp-carousel__pick"
            :tabindex="cycle !== 0 ? -1 : undefined"
            :aria-label="playerLabel(player)"
            :aria-pressed="selectedPlayerId === player.id"
            :disabled="disabled"
            @focus="centerOnKeyboardFocus($event, slot)"
            @click="select(player, slot)"
          ></button>
          <FanPlayerCard
            class="mvp-carousel__art"
            :player="player"
            :club-logo-url="clubLogoUrl"
            :selected="votedPlayerId === player.id"
            :aria-hidden="!showsConfirm(player, slot) || undefined"
          >
            <template v-if="showsConfirm(player, slot) || (slot === centeredSlot && player.id === votedPlayerId)" #action>
              <span v-if="player.id === votedPlayerId" class="mvp-carousel__hint">Scegli un nuovo MVP</span>
              <button
                v-else
                class="mvp-carousel__confirm"
                type="button"
                :tabindex="cycle !== 0 ? -1 : undefined"
                :aria-label="submitting ? 'Invio del voto…' : `Conferma voto per ${player.fullName || player.name}`"
                :disabled="disabled || !canConfirm"
                @click="confirm(player, slot)"
              >{{ submitting ? 'Invio…' : 'Conferma' }}</button>
            </template>
          </FanPlayerCard>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import FanPlayerCard from './FanPlayerCard.vue';

const props = defineProps({
  players: { type: Array, required: true },
  votedPlayerId: { type: Number, default: null },
  selectedPlayerId: { type: Number, default: null },
  clubLogoUrl: { type: String, default: '' },
  disabled: { type: Boolean, default: false },
  canConfirm: { type: Boolean, default: false },
  submitting: { type: Boolean, default: false },
});
const emit = defineEmits(['select', 'confirm']);
const track = ref(null);
const focusedIndex = ref(Math.max(0, props.players.findIndex((player) => player.id === props.votedPlayerId)));
// Extra cards on each side let native swipes cross the seam before we reset the position.
const loopPadding = computed(() => props.players.length > 1 ? 6 : 0);
const centeredSlot = ref(loopPadding.value + focusedIndex.value);
const slots = computed(() => {
  const count = props.players.length;
  return Array.from({ length: count + 2 * loopPadding.value }, (_, slot) => {
    const position = slot - loopPadding.value;
    const cycle = Math.floor(position / count);
    const index = position - cycle * count;
    return { player: props.players[index], index, cycle };
  });
});
let scrollFrame = 0;
let scrollTimer;
let targetSlot = null;
let movingFocus = false;
let hasBrowsed = false;

function playerLabel(player) {
  const number = player.number !== '' && player.number != null ? `maglia ${player.number}` : '';
  const voted = player.id === props.votedPlayerId ? 'Il tuo MVP. Scegli un nuovo MVP.' : '';
  return [player.fullName || player.name, number, player.role, voted].filter(Boolean).join(', ');
}

function showsConfirm(player, slot) {
  return centeredSlot.value === slot && props.selectedPlayerId === player.id && props.votedPlayerId !== player.id;
}

function confirm(player, slot) {
  if (props.disabled || !props.canConfirm || !showsConfirm(player, slot) || nearestSlot() !== slot) return;
  emit('confirm', player.id);
}

function slotLeft(element, slot) {
  return slot.offsetLeft + (slot.offsetWidth - element.clientWidth) / 2;
}

function centerSlot(index, smooth = true) {
  const element = track.value;
  const slot = element?.children[index];
  if (!slot) return;
  const left = slotLeft(element, slot);
  const animate = smooth && !window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  targetSlot = animate && Math.abs(element.scrollLeft - left) > 1 ? index : null;
  element.scrollTo({ left, behavior: animate ? 'smooth' : 'auto' });
}

function nearestSlot() {
  const element = track.value;
  const count = element?.children.length ?? 0;
  if (count < 2) return 0;
  const first = element.firstElementChild;
  const step = (element.lastElementChild.offsetLeft - first.offsetLeft) / (count - 1);
  const index = step > 0 ? Math.round((element.scrollLeft - slotLeft(element, first)) / step) : loopPadding.value;
  return Math.max(0, Math.min(index, count - 1));
}

function getPlayerCardRect(playerId) {
  const slot = slots.value[targetSlot]?.player.id === playerId ? targetSlot : nearestSlot();
  if (slots.value[slot]?.player.id !== playerId) return null;
  const element = track.value;
  const rect = element?.children[slot]?.querySelector('.fan-player-card')?.getBoundingClientRect();
  if (!rect?.width || !rect.height || rect.bottom <= 0 || rect.top >= window.innerHeight) return null;
  const viewport = element.getBoundingClientRect();
  return rect.right > viewport.left && rect.left < viewport.right ? rect : null;
}

defineExpose({ getPlayerCardRect });

function startBrowsing() {
  targetSlot = null;
  hasBrowsed = true;
}

function updateCenteredPlayer(slotIndex) {
  const slot = slots.value[slotIndex];
  if (!slot) return;
  centeredSlot.value = slotIndex;
  if (focusedIndex.value === slot.index) return;
  focusedIndex.value = slot.index;
  if (hasBrowsed && !props.disabled && props.selectedPlayerId !== slot.player.id) emit('select', slot.player);
}

function updateFocus() {
  if (!track.value) return;
  if (!('onscrollend' in track.value)) {
    clearTimeout(scrollTimer);
    scrollTimer = setTimeout(settleScroll, 150);
  }
  if (scrollFrame) return;
  scrollFrame = requestAnimationFrame(() => {
    scrollFrame = 0;
    updateCenteredPlayer(nearestSlot());
  });
}

function settleScroll() {
  targetSlot = null;
  const slotIndex = nearestSlot();
  const slot = slots.value[slotIndex];
  if (!slot) return;
  updateCenteredPlayer(slotIndex);
  if (slot.cycle !== 0) {
    centeredSlot.value = loopPadding.value + slot.index;
    centerSlot(centeredSlot.value, false);
  }
}

function move(direction) {
  if (!loopPadding.value) return;
  hasBrowsed = true;
  const element = track.value;
  const current = targetSlot ?? nearestSlot();
  let next = current + direction;
  if (next <= 0 || next >= slots.value.length - 1) {
    const middle = loopPadding.value + slots.value[current].index;
    centerSlot(middle, false);
    next = middle + direction;
  }
  if (!props.disabled && element.contains(document.activeElement)) {
    const middle = loopPadding.value + slots.value[next].index;
    movingFocus = true;
    element.children[middle].querySelector('.mvp-carousel__pick').focus({ preventScroll: true });
    movingFocus = false;
  }
  centerSlot(next);
}

function centerOnKeyboardFocus(event, slot) {
  if (!movingFocus && event.currentTarget.matches(':focus-visible')) {
    const { player, index } = slots.value[slot];
    select(player, slot, false);
    centeredSlot.value = slot;
    focusedIndex.value = index;
  }
}

function select(player, slot, smooth = true) {
  if (props.disabled) return;
  hasBrowsed = true;
  emit('select', player);
  centerSlot(smooth && slots.value[targetSlot]?.player === player ? targetSlot : slot, smooth);
}

watch(() => props.players.map((player) => player.id), (ids, previousIds) => {
  if (ids.length === previousIds.length && ids.every((id, index) => id === previousIds[index])) return;
  const previous = ids.indexOf(previousIds[focusedIndex.value]);
  focusedIndex.value = previous >= 0 ? previous : Math.max(0, ids.indexOf(props.votedPlayerId));
  centeredSlot.value = loopPadding.value + focusedIndex.value;
  centerSlot(centeredSlot.value, false);
}, { flush: 'post' });

onMounted(() => centerSlot(centeredSlot.value, false));
onBeforeUnmount(() => {
  cancelAnimationFrame(scrollFrame);
  clearTimeout(scrollTimer);
});
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
  overscroll-behavior-x: none;
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
  position: relative;
  display: block;
  width: 100%;
  padding: 2px;
  border: 2px solid transparent;
  border-radius: 15px;
  background: transparent;
  transform: scale(.93);
  transition: transform .18s ease, border-color .18s ease;
}
.mvp-carousel__player.is-focused { transform: none; border-color: #9e6c1d; }
.mvp-carousel__player.is-selected { border-color: var(--red-text); background: var(--red-soft); }
.mvp-carousel__pick {
  position: absolute;
  inset: 0;
  padding: 0;
  border: 0;
  border-radius: inherit;
  background: transparent;
  cursor: pointer;
}
.mvp-carousel__pick:disabled { cursor: default; }
.mvp-carousel__art { position: relative; pointer-events: none; }
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
.mvp-carousel__player.has-confirm :deep(.mvp-player-caption) { padding: 8%; }
.mvp-carousel__player :deep(.mvp-player-caption strong) { color: var(--ink); font-size: .75rem; }
.mvp-carousel__player :deep(.mvp-player-role) { color: var(--ink); font-size: .6875rem; }
.mvp-carousel__player :deep(.mvp-player-selected) { font-size: .625rem; }
.mvp-carousel__hint {
  color: var(--ink);
  margin-top: 3px;
  font-size: .625rem;
  font-weight: 700;
  line-height: 1.15;
}
.mvp-carousel__confirm {
  position: relative;
  isolation: isolate;
  display: grid;
  align-self: stretch;
  min-height: 44px;
  place-items: center;
  padding: 0 4px;
  border: 0;
  border-radius: 9px;
  background: transparent;
  color: var(--on-primary);
  font: 800 .75rem/1.2 system-ui;
  cursor: pointer;
  pointer-events: auto;
}
.mvp-carousel__confirm::before {
  position: absolute;
  inset: 8px 9px;
  z-index: -1;
  border-radius: inherit;
  background: var(--red);
  content: '';
}
.mvp-carousel__confirm:disabled { opacity: .7; cursor: default; }
.mvp-carousel__pick:focus-visible,
.mvp-carousel__confirm:focus-visible {
  outline: 2px solid var(--red-text);
  outline-offset: 2px;
}
.mvp-carousel__confirm:focus-visible { outline-offset: -2px; }
@media (prefers-reduced-motion: reduce) {
  .mvp-carousel__player { transition: none; }
}
</style>
