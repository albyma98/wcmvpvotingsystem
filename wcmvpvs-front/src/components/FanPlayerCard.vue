<template>
  <span class="fan-player-card" :class="{ 'is-selected': selected }" role="img" :aria-label="accessibleLabel">
    <span class="mvp-player-card">
      <span v-if="hasNumber" class="mvp-player-number" aria-hidden="true"><small>MAGLIA</small>{{ player.number }}</span>
      <img v-if="clubLogoUrl" class="mvp-player-club" :src="clubLogoUrl" alt="" :loading="loading" />
      <span class="mvp-player-portrait">
        <img v-if="player.image" :src="player.image" alt="" :loading="loading" decoding="async" />
        <span v-else class="mvp-player-initials" aria-hidden="true">{{ player.badge || fallbackInitials }}</span>
      </span>
      <span class="mvp-player-caption">
        <strong>{{ player.name || 'Giocatore' }}</strong>
        <span v-if="player.subtitle" class="mvp-player-role">{{ player.subtitle }}</span>
        <span v-if="selected" class="mvp-player-selected"><span aria-hidden="true">✓</span> {{ selectedLabel }}</span>
      </span>
    </span>
  </span>
</template>

<script setup>
import { computed } from 'vue';

const props = defineProps({
  player: { type: Object, required: true },
  clubLogoUrl: { type: String, default: '' },
  selected: { type: Boolean, default: false },
  selectedLabel: { type: String, default: 'Il tuo MVP' },
  loading: { type: String, default: 'lazy' },
});

const hasNumber = computed(() => props.player.number !== '' && props.player.number != null);
const fallbackInitials = computed(() => String(props.player.fullName || props.player.name || 'Giocatore')
  .trim().split(/\s+/).filter(Boolean).slice(0, 3).map((part) => part[0]).join('').toUpperCase());
const accessibleLabel = computed(() => [
  props.player.fullName || props.player.name || 'Giocatore',
  hasNumber.value ? `maglia ${props.player.number}` : '',
  props.player.subtitle,
  props.selected ? props.selectedLabel : '',
].filter(Boolean).join(', '));
</script>

<style scoped>
.fan-player-card {
  display: block;
  width: 100%;
  aspect-ratio: 0.66;
  min-width: 0;
  color: #39270f;
  font-family: inherit;
  text-align: center;
  filter: drop-shadow(0 5px 4px rgba(78, 51, 12, 0.13));
}
.fan-player-card.is-selected { filter: drop-shadow(0 4px 8px var(--red-line)); }
.fan-player-card * { box-sizing: border-box; }
.mvp-player-card {
  position: relative;
  display: block;
  aspect-ratio: 0.66;
  overflow: hidden;
  isolation: isolate;
  clip-path: polygon(0 9%, 17% 4%, 50% 0, 83% 4%, 100% 9%, 100% 85%, 89% 93%, 50% 100%, 11% 93%, 0 85%);
  background:
    linear-gradient(135deg, transparent 18%, rgba(255, 255, 255, 0.5) 18.5%, transparent 19.5%, transparent 72%, rgba(255, 255, 255, 0.35) 72.5%, transparent 74%),
    radial-gradient(ellipse at 72% 18%, #fff9dc 0, transparent 55%),
    linear-gradient(145deg, #f9e7ad, #d4ad5c 48%, #f3da91 75%, #be9242);
}
.mvp-player-card::before {
  content: '';
  position: absolute;
  inset: 10% 0 32%;
  z-index: -1;
  background: linear-gradient(125deg, transparent 30%, rgba(255, 255, 255, 0.25) 30%, rgba(255, 255, 255, 0.25) 53%, transparent 53%);
}
.mvp-player-number {
  position: absolute;
  top: 12%;
  left: 9%;
  z-index: 2;
  display: flex;
  flex-direction: column;
  align-items: center;
  font-size: clamp(19px, 6vw, 28px);
  line-height: 1;
  font-weight: 900;
  letter-spacing: -0.06em;
  text-shadow: 0 1px 5px #f9e7ad;
}
.mvp-player-number small {
  margin-bottom: 3px;
  font-size: 6px;
  letter-spacing: 0.08em;
  font-weight: 800;
}
.mvp-player-club {
  position: absolute;
  z-index: 2;
  top: 12%;
  right: 8%;
  width: 21%;
  height: 15%;
  object-fit: contain;
}
.mvp-player-portrait {
  position: absolute;
  inset: 15% 0 17%;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  mask-image: linear-gradient(to bottom, #000 68%, transparent 100%);
  -webkit-mask-image: linear-gradient(to bottom, #000 68%, transparent 100%);
}
.mvp-player-portrait img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  object-position: center top;
}
.mvp-player-initials {
  display: grid;
  place-items: center;
  width: 60%;
  aspect-ratio: 1;
  border-radius: 50%;
  background: rgba(255, 249, 220, 0.55);
  color: #785923;
  font-family: 'Fraunces', serif;
  font-size: clamp(24px, 7vw, 34px);
  font-weight: 750;
}
.mvp-player-caption {
  position: absolute;
  inset: auto 0 0;
  min-height: 37%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 14% 7% 12%;
  background: linear-gradient(to bottom, transparent, #f3da91 27%, #e2bf70 100%);
}
.mvp-player-caption strong {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  overflow: hidden;
  width: 100%;
  color: #39270f;
  font-size: clamp(10px, 2.9vw, 12px);
  line-height: 1.15;
  font-weight: 900;
  text-transform: uppercase;
  overflow-wrap: anywhere;
}
.mvp-player-role {
  max-width: 100%;
  margin-top: 3px;
  overflow: hidden;
  color: #66491f;
  font-size: 8px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.mvp-player-selected {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 3px;
  margin-top: 4px;
  color: var(--on-primary, #fff);
  background: var(--red);
  padding: 3px 6px;
  border-radius: 20px;
  font-size: 8px;
  line-height: 1.1;
  font-weight: 800;
}
.mvp-player-card { height: 100%; }
</style>
