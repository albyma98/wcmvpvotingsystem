<template>
  <ul
    class="mvp-player-grid"
    :class="{ 'is-active': active, 'is-busy': busy }"
    aria-label="Scegli il tuo MVP"
    :aria-busy="busy"
  >
    <li v-for="(player, index) in players" :key="player.id" class="mvp-roster-slot" :style="{ '--card-delay': `${Math.min(index * 38, 228)}ms` }">
      <button
        class="mvp-player"
        :class="{ 'is-selected': samePlayer(player.id, selectedPlayerId), 'is-pending': samePlayer(player.id, pendingPlayerId) }"
        type="button"
        :disabled="!active || busy || !canVote"
        :aria-label="playerLabel(player)"
        :aria-pressed="samePlayer(player.id, selectedPlayerId)"
        @click="selectPlayer(player)"
      >
        <span class="mvp-card-entrance">
          <span class="mvp-card-surface">
            <FanPlayerCard
              :player="player"
              :club-logo-url="clubLogoUrl"
              :selected="samePlayer(player.id, selectedPlayerId)"
              selected-label="Il tuo MVP"
              aria-hidden="true"
            />
            <span class="mvp-card-glint" aria-hidden="true"></span>
            <span v-if="samePlayer(player.id, pendingPlayerId)" class="mvp-card-pending" aria-hidden="true">
              <span class="mvp-card-spinner"></span> Invio del voto…
            </span>
          </span>
        </span>
      </button>
    </li>
  </ul>
</template>

<script setup>
import FanPlayerCard from './FanPlayerCard.vue';

const props = defineProps({
  players: { type: Array, default: () => [] },
  active: { type: Boolean, default: false },
  selectedPlayerId: { type: Number, default: null },
  pendingPlayerId: { type: Number, default: null },
  busy: { type: Boolean, default: false },
  canVote: { type: Boolean, default: false },
  clubLogoUrl: { type: String, default: '' },
});
const emit = defineEmits(['select']);

function samePlayer(left, right) {
  return left != null && right != null && String(left) === String(right);
}

function playerLabel(player) {
  const name = player.fullName || player.name || 'Giocatore';
  const number = player.number !== '' && player.number != null ? `, maglia ${player.number}` : '';
  const status = samePlayer(player.id, props.selectedPlayerId)
    ? 'Il tuo MVP'
    : samePlayer(player.id, props.pendingPlayerId) ? 'Invio del voto in corso' : 'Vota come MVP';
  return `${name}${number}, ${status}`;
}

function selectPlayer(player) {
  if (!props.active || props.busy || !props.canVote) return;
  const rosterPlayer = props.players.find((candidate) => samePlayer(candidate.id, player?.id));
  if (!rosterPlayer) return;
  emit('select', rosterPlayer);
}
</script>

<style scoped>
.mvp-player-grid {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  flex-shrink: 0;
  gap: 19px 10px;
  margin: 0;
  padding: 14px 4px 19px;
  list-style: none;
  perspective: 850px;
}
.mvp-roster-slot {
  flex: 0 0 calc((100% - 20px) / 3);
  min-width: 0;
}
.mvp-player {
  position: relative;
  display: block;
  width: 100%;
  min-width: 0;
  padding: 0;
  border: 0;
  border-radius: 13px;
  background: transparent;
  font: inherit;
  text-align: center;
  cursor: pointer;
  -webkit-tap-highlight-color: transparent;
  transition: transform 220ms cubic-bezier(.2, .75, .2, 1), opacity 220ms ease, filter 220ms ease;
}
.mvp-player:focus-visible {
  outline: 3px solid var(--red-text, #923022);
  outline-offset: 5px;
}
.mvp-card-entrance,
.mvp-card-surface {
  position: relative;
  display: block;
  width: 100%;
}
.mvp-card-surface {
  isolation: isolate;
  filter: drop-shadow(0 9px 7px rgb(36 25 15 / 12%));
  transition: filter 220ms ease;
}
.is-active .mvp-card-entrance {
  animation: mvp-card-deal 460ms cubic-bezier(.16, .8, .25, 1) var(--card-delay) both;
}
.mvp-card-glint {
  position: absolute;
  z-index: 3;
  inset: 0;
  overflow: hidden;
  pointer-events: none;
  clip-path: polygon(0 9%, 17% 4%, 50% 0, 83% 4%, 100% 9%, 100% 85%, 89% 93%, 50% 100%, 11% 93%, 0 85%);
}
.mvp-card-glint::after {
  content: '';
  position: absolute;
  inset: -30% -65%;
  background: linear-gradient(110deg, transparent 35%, rgb(255 255 255 / 0%) 39%, rgb(255 255 255 / 66%) 48%, rgb(255 255 255 / 8%) 56%, transparent 61%);
  transform: translateX(-65%);
}
.mvp-card-glint::before {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(115deg, transparent 25%, rgb(255 255 255 / 22%) 46%, transparent 65%);
  opacity: 0;
  transition: opacity 180ms ease;
}
.is-active .mvp-card-glint::after {
  animation: mvp-card-shine 680ms ease calc(var(--card-delay) + 140ms) both;
}
.mvp-player.is-selected { transform: translateY(-3px); }
.mvp-player.is-pending {
  z-index: 3;
  transform: translateY(-7px) scale(1.055);
}
.mvp-player.is-pending .mvp-card-surface,
.mvp-player.is-selected .mvp-card-surface {
  filter: drop-shadow(0 8px 12px var(--red-line, rgb(156 56 29 / 24%)));
}
.is-busy .mvp-player:not(.is-pending) {
  opacity: .48;
  filter: saturate(.65);
}
.mvp-player:disabled { cursor: default; }
.mvp-card-pending {
  position: absolute;
  z-index: 4;
  inset: auto 0 -8px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  min-height: 24px;
  padding: 5px 2px;
  border-radius: 9px;
  background: var(--ink, #29221e);
  color: var(--bg, #fffaf2);
  font-size: clamp(7px, 2.1vw, 9px);
  line-height: 1.15;
  font-weight: 750;
  box-shadow: 0 3px 8px rgb(0 0 0 / 18%);
}
.mvp-card-spinner {
  flex: 0 0 9px;
  height: 9px;
  border: 1.5px solid currentColor;
  border-right-color: transparent;
  border-radius: 50%;
  animation: mvp-card-spin 800ms linear infinite;
}
@media (hover: hover) {
  .mvp-player:hover:not(:disabled) { transform: translateY(-5px) rotate(-1deg) scale(1.025); }
  .mvp-player:hover:not(:disabled) .mvp-card-surface { filter: drop-shadow(0 12px 9px rgb(36 25 15 / 20%)); }
  .mvp-player:hover:not(:disabled) .mvp-card-glint::before { opacity: 1; }
}
.mvp-player.is-pending .mvp-card-glint::before { opacity: 1; }
.mvp-player:active:not(:disabled) { transform: translateY(-2px) scale(.97); }
@keyframes mvp-card-deal {
  from { opacity: 0; transform: translateY(24px) rotateX(13deg) scale(.91); }
  to { opacity: 1; transform: translateY(0) rotateX(0) scale(1); }
}
@keyframes mvp-card-shine {
  from { transform: translateX(-65%); }
  to { transform: translateX(65%); }
}
@keyframes mvp-card-spin { to { transform: rotate(360deg); } }
@media (prefers-reduced-motion: reduce) {
  .mvp-player,
  .mvp-card-surface,
  .mvp-card-glint::before { transition: none; }
  .is-active .mvp-card-entrance,
  .is-active .mvp-card-glint::after,
  .mvp-card-spinner { animation: none; }
  .mvp-player.is-pending,
  .mvp-player.is-selected,
  .mvp-player:hover:not(:disabled),
  .mvp-player:active:not(:disabled) { transform: none; }
}
</style>
