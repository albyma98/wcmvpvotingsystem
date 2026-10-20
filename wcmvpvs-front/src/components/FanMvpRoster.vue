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
            <span class="mvp-card-energy" aria-hidden="true"></span>
            <span class="mvp-card-sparks" aria-hidden="true">
              <span v-for="spark in 6" :key="spark" :style="{ '--spark-angle': `${spark * 60 - 30}deg`, '--spark-delay': `${(spark % 3) * 35}ms` }"></span>
            </span>
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
  --mvp-energy: var(--gold, #d4ad5c);
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
  position: relative;
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
  touch-action: manipulation;
  transition: transform 240ms cubic-bezier(.2, .85, .25, 1.3), opacity 220ms ease, filter 220ms ease;
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
  animation: mvp-card-deal 580ms cubic-bezier(.16, .8, .25, 1) var(--card-delay) both;
}
.mvp-card-energy {
  position: absolute;
  z-index: -1;
  inset: 2% -9%;
  pointer-events: none;
  opacity: 0;
  transition: opacity 220ms ease;
}
.mvp-card-energy::before,
.mvp-card-energy::after {
  content: '';
  position: absolute;
  inset: 7% 0;
  border-radius: 50%;
}
.mvp-card-energy::before {
  background: radial-gradient(ellipse, var(--mvp-energy), transparent 68%);
  filter: blur(7px);
}
.mvp-card-energy::after {
  inset: 20% 6%;
  border: 2px solid var(--mvp-energy);
  opacity: 0;
}
.mvp-card-sparks {
  position: absolute;
  z-index: 3;
  inset: 0;
  pointer-events: none;
}
.mvp-card-sparks > span {
  position: absolute;
  left: calc(50% - 2px);
  top: 45%;
  width: 4px;
  height: 12px;
  border-radius: 3px;
  background: linear-gradient(#fff8d1, var(--mvp-energy));
  box-shadow: 0 0 7px var(--mvp-energy);
  opacity: 0;
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
.mvp-player.is-selected {
  z-index: 2;
  transform: translateY(-5px) scale(1.025);
}
.mvp-player.is-pending {
  z-index: 3;
  transform: translateY(-8px) scale(1.055);
}
.mvp-player.is-pending .mvp-card-surface,
.mvp-player.is-selected .mvp-card-surface {
  filter: drop-shadow(0 8px 12px var(--red-line, rgb(156 56 29 / 24%)));
}
.mvp-player.is-pending .mvp-card-energy,
.mvp-player.is-selected .mvp-card-energy { opacity: .8; }
.is-active .mvp-player.is-pending .mvp-card-energy::before {
  animation: mvp-card-charge 1100ms ease-in-out infinite;
}
.is-active .mvp-player.is-selected .mvp-card-surface {
  animation: mvp-card-crown 580ms cubic-bezier(.2, .8, .3, 1) both;
}
.is-active .mvp-player.is-selected .mvp-card-energy::after {
  animation: mvp-card-ring 620ms ease-out both;
}
.is-active .mvp-player.is-selected .mvp-card-sparks > span {
  animation: mvp-card-spark 650ms cubic-bezier(.12, .65, .3, 1) var(--spark-delay) both;
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
}
.is-active .mvp-card-spinner {
  animation: mvp-card-spin 800ms linear infinite;
}
@media (hover: hover) {
  .mvp-player:hover:not(:disabled) { z-index: 2; transform: translateY(-7px) rotate(-1.5deg) scale(1.045); }
  .mvp-player:hover:not(:disabled) .mvp-card-surface { filter: drop-shadow(0 12px 9px rgb(36 25 15 / 20%)); }
  .mvp-player:hover:not(:disabled) .mvp-card-glint::before { opacity: 1; }
}
.mvp-player.is-pending .mvp-card-glint::before { opacity: 1; }
.mvp-player:active:not(:disabled) { transform: translateY(1px) rotate(0) scale(.94); transition-duration: 80ms; }
@keyframes mvp-card-deal {
  0% { opacity: 0; transform: translateY(34px) rotateX(18deg) rotateZ(-3deg) scale(.84); }
  70% { opacity: 1; transform: translateY(-3px) rotateX(0) rotateZ(.7deg) scale(1.015); }
  100% { opacity: 1; transform: translateY(0) rotateX(0) rotateZ(0) scale(1); }
}
@keyframes mvp-card-shine {
  from { transform: translateX(-65%); }
  to { transform: translateX(65%); }
}
@keyframes mvp-card-spin { to { transform: rotate(360deg); } }
@keyframes mvp-card-charge {
  0%, 100% { opacity: .45; transform: scale(.9); }
  50% { opacity: .85; transform: scale(1.12); }
}
@keyframes mvp-card-crown {
  0% { transform: scale(1); }
  35% { transform: translateY(-3px) scale(1.085) rotate(-2deg); }
  65% { transform: scale(.985) rotate(.8deg); }
  100% { transform: scale(1) rotate(0); }
}
@keyframes mvp-card-ring {
  0% { opacity: .85; transform: scale(.55); }
  100% { opacity: 0; transform: scale(1.7); }
}
@keyframes mvp-card-spark {
  0% { opacity: 0; transform: rotate(var(--spark-angle)) translateY(-22px) scale(.4); }
  20% { opacity: 1; }
  100% { opacity: 0; transform: rotate(var(--spark-angle)) translateY(-100px) scale(.1); }
}
@media (prefers-reduced-motion: reduce) {
  .mvp-player,
  .mvp-card-surface,
  .mvp-card-glint::before,
  .mvp-card-energy { transition: none; }
  .is-active .mvp-card-entrance,
  .is-active .mvp-card-glint::after,
  .is-active .mvp-player.is-pending .mvp-card-energy::before,
  .is-active .mvp-player.is-selected .mvp-card-surface,
  .is-active .mvp-player.is-selected .mvp-card-energy::after,
  .is-active .mvp-player.is-selected .mvp-card-sparks > span,
  .is-active .mvp-card-spinner { animation: none; }
  .mvp-player.is-pending,
  .mvp-player.is-selected,
  .mvp-player:hover:not(:disabled),
  .mvp-player:active:not(:disabled) { transform: none; }
}
</style>
