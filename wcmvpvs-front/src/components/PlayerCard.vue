<script setup>
import { computed } from 'vue';
import { Motion } from '@vueuse/motion';

const props = defineProps({
  player: {
    type: Object,
    required: true,
  },
  cardSize: {
    type: Number,
    default: 90,
  },
  isSelected: {
    type: Boolean,
    default: false,
  },
  disabled: {
    type: Boolean,
    default: false,
  },
  isVoting: {
    type: Boolean,
    default: false,
  },
  isPrematch: {
    type: Boolean,
    default: false,
  },
  activationCue: {
    type: Boolean,
    default: false,
  },
  hasSpotlight: {
    type: Boolean,
    default: false,
  },
});

const emits = defineEmits(['select']);

const tierRingClass = computed(
  () => 'ring-yellow-300/80 shadow-[0_0_32px_rgba(250,204,21,0.4)]',
);

const entryDelay = computed(() => {
  const y = Number(props.player?.position?.y ?? 50);
  return Math.min(Math.max(y / 140, 0), 1.1);
});

const motionConfig = computed(() => ({
  initial: {
    opacity: 0,
    y: 18,
    scale: 0.96,
    filter: 'blur(4px)',
  },
  enter: {
    opacity: 1,
    y: 0,
    scale: 1,
    filter: 'blur(0px)',
    transition: {
      delay: entryDelay.value,
      duration: 0.45,
      easing: 'ease-out',
    },
  },
  hover: {
    scale: props.disabled ? 1 : 1.04,
  },
  press: {
    scale: props.disabled ? 1 : 0.98,
  },
}));

const fallbackAvatar = computed(
  () => `https://api.dicebear.com/7.x/adventurer/svg?seed=${encodeURIComponent(props.player.name ?? props.player.number)}`,
);

const avatarUrl = computed(() => props.player.avatar || fallbackAvatar.value);

const cardStyle = computed(() => ({
  width: `${props.cardSize}px`,
  height: `${props.cardSize * 1.5}px`,
}));

const wrapperStyle = computed(() => ({
  width: `${props.cardSize}px`,
}));

const playerNameParts = computed(() => {
  const rawName = props.player.name?.trim();
  if (!rawName) {
    return { firstName: '', lastName: '' };
  }
  const [firstName, ...rest] = rawName.split(/\s+/);
  return {
    firstName,
    lastName: rest.join(' '),
  };
});

const overlayLastName = computed(() => {
  const { lastName, firstName } = playerNameParts.value;
  if (lastName) {
    return lastName;
  }
  if (firstName) {
    return firstName;
  }
  return props.player.name ?? '';
});

const overlayNumber = computed(() =>
  props.player.number !== undefined && props.player.number !== null
    ? `#${props.player.number}`
    : '',
);

const handleSelect = () => {
  if ((props.disabled && !props.isSelected) || props.isVoting) {
    return;
  }
  emits('select');
};
</script>

<template>
  <div class="flex flex-col items-center" :style="wrapperStyle">
    <Motion v-bind="motionConfig" class="w-full">
      <div
        :style="cardStyle"
        class="player-card relative rounded-[1.75rem] border border-white/10 bg-slate-950/60 transition-transform duration-200 ease-out"
        :class="[
          tierRingClass,
          isSelected ? 'scale-[1.05]' : 'hover:scale-[1.03]',
          disabled && !isSelected ? 'cursor-not-allowed opacity-60' : 'cursor-pointer',
          isSelected ? 'ring-4' : 'ring-2',
          isPrematch ? 'player-card--prematch' : 'player-card--live',
          activationCue && !isPrematch ? 'player-card--activation' : '',
          hasSpotlight ? 'player-card--spotlight-focus' : '',
        ]"
        @click="handleSelect"
      >
        <span class="player-card__spotlight" aria-hidden="true"></span>
        <span class="player-card__sweep" aria-hidden="true"></span>
        <span v-if="isSelected" class="player-card__halo" aria-hidden="true"></span>

        <div class="player-card__annotation">
          <span class="player-card__annotation-name">{{ overlayLastName }}</span>
          <span v-if="overlayNumber" class="player-card__annotation-number">
            {{ overlayNumber }}
          </span>
        </div>

        <div class="player-card__frame" aria-hidden="true"></div>
        <div class="player-card__frame player-card__frame--glow" aria-hidden="true"></div>
        <div class="player-card__frame player-card__frame--inner" aria-hidden="true"></div>

        <div class="player-card__content">
          <div class="player-card__header">
            <div class="player-card__meta">
              <p class="player-card__role">{{ player.role }}</p>
              <p class="player-card__number">{{ player.number ?? '#' }}</p>
            </div>
            <div class="player-card__status" :data-live="!isPrematch">
              <span class="player-card__pill"></span>
              <span>{{ isPrematch ? 'In attesa' : 'Live' }}</span>
            </div>
          </div>

          <div class="player-card__portrait" :class="isPrematch ? 'muted' : ''">
            <img :src="avatarUrl" :alt="player.name" loading="lazy" />
            <span class="player-card__shine" aria-hidden="true"></span>
          </div>

          <div class="player-card__footer">
            <p class="player-card__name">{{ player.name }}</p>
            <p class="player-card__team">{{ player.team }}</p>
          </div>
        </div>
      </div>
    </Motion>
  </div>
</template>

<style scoped>
.player-card {
  overflow: hidden;
  isolation: isolate;
  transition: transform 220ms ease, filter 300ms ease;
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.06), 0 18px 32px rgba(0, 0, 0, 0.35);
}

.player-card--prematch {
  filter: brightness(0.52) saturate(0.85);
}

.player-card--live {
  filter: brightness(1);
}

.player-card--activation {
  animation: card-ignite 1s ease;
}

.player-card__spotlight,
.player-card__sweep {
  position: absolute;
  inset: -20%;
  pointer-events: none;
  transition: opacity 300ms ease;
}

.player-card__spotlight {
  background: radial-gradient(circle at 50% 40%, rgba(255, 221, 120, 0.32), transparent 45%);
  opacity: 0;
}

.player-card__halo {
  position: absolute;
  inset: -30%;
  border-radius: 2rem;
  background: radial-gradient(circle at 50% 50%, rgba(255, 214, 102, 0.35), transparent 65%);
  filter: blur(24px);
  animation: crown-pulse 1.2s ease forwards;
  mix-blend-mode: screen;
}

.player-card__sweep {
  background: linear-gradient(100deg, transparent 30%, rgba(255, 255, 255, 0.32) 50%, transparent 70%);
  transform: translateX(-120%);
  opacity: 0;
}

.player-card--prematch .player-card__sweep {
  opacity: 0.75;
  animation: stadium-sweep 6s linear infinite;
  animation-delay: calc(var(--card-index, 0) * 0.6s);
}

.player-card--live:hover .player-card__spotlight,
.player-card--live:focus-visible .player-card__spotlight {
  opacity: 0.9;
  mix-blend-mode: screen;
}

.player-card--spotlight-focus {
  box-shadow: inset 0 0 0 1px rgba(255, 214, 102, 0.4), 0 22px 48px rgba(251, 191, 36, 0.25),
    0 0 85px rgba(255, 214, 102, 0.25);
  filter: brightness(1.02) saturate(1.05);
}

.player-card--spotlight-focus .player-card__spotlight {
  opacity: 0.9;
  background: radial-gradient(circle at 50% 35%, rgba(255, 225, 130, 0.45), transparent 50%);
}

.player-card--live:hover,
.player-card--live:focus-visible {
  transform: scale(1.03);
}

.player-card--live:active {
  animation: card-bounce 260ms ease;
}

.player-card__frame {
  position: absolute;
  inset: 0;
  border-radius: 1.75rem;
  border: 1px solid rgba(255, 255, 255, 0.08);
  background: radial-gradient(circle at 20% 18%, rgba(148, 163, 184, 0.14), transparent 40%),
    radial-gradient(circle at 80% 18%, rgba(148, 163, 184, 0.12), transparent 42%);
  opacity: 0.7;
}

.player-card__frame--glow {
  border-color: rgba(56, 189, 248, 0.35);
  filter: blur(12px);
  opacity: 0.7;
}

.player-card__frame--inner {
  inset: 8px;
  border-color: rgba(255, 255, 255, 0.05);
  opacity: 0.5;
}

.player-card__content {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  height: 100%;
  gap: 0.35rem;
  padding: 0.6rem 0.65rem 0.55rem;
  color: #e2e8f0;
}

.player-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.player-card__meta {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
}

.player-card__role {
  margin: 0;
  font-size: 0.68rem;
  letter-spacing: 0.22em;
  text-transform: uppercase;
  color: rgba(148, 163, 184, 0.9);
}

.player-card__number {
  margin: 0;
  font-size: 1.2rem;
  font-weight: 800;
  letter-spacing: 0.12em;
}

.player-card__status {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.35rem 0.65rem;
  border-radius: 999px;
  background: rgba(15, 23, 42, 0.5);
  border: 1px solid rgba(255, 255, 255, 0.08);
  font-size: 0.7rem;
  letter-spacing: 0.14em;
  text-transform: uppercase;
}

.player-card__status[data-live='true'] {
  border-color: rgba(56, 189, 248, 0.4);
  color: #bae6fd;
}

.player-card__pill {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background: #22c55e;
  box-shadow: 0 0 0 6px rgba(34, 197, 94, 0.18);
}

.player-card__status[data-live='false'] .player-card__pill {
  background: #f59e0b;
  box-shadow: 0 0 0 6px rgba(245, 158, 11, 0.16);
}

.player-card__portrait {
  position: relative;
  margin-top: 0.35rem;
  aspect-ratio: 2 / 2.65;
  border-radius: 1.4rem;
  overflow: hidden;
  background: linear-gradient(180deg, rgba(15, 23, 42, 0.8), rgba(30, 41, 59, 0.95));
  display: grid;
  place-items: center;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.05);
}

.player-card__portrait img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  filter: saturate(1.05);
}

.player-card__portrait.muted img {
  filter: saturate(0.8) brightness(0.8);
}

.player-card__shine {
  position: absolute;
  inset: -20% 10% auto;
  height: 55%;
  background: linear-gradient(120deg, rgba(255, 255, 255, 0.28), rgba(255, 255, 255, 0));
  transform: rotate(-6deg);
  opacity: 0.35;
}

.player-card__footer {
  margin-top: auto;
  display: flex;
  flex-direction: column;
  gap: 0.08rem;
}

.player-card__name {
  margin: 0;
  font-size: 0.95rem;
  font-weight: 800;
  letter-spacing: 0.08em;
  color: #f8fafc;
}

.player-card__team {
  margin: 0;
  font-size: 0.78rem;
  letter-spacing: 0.06em;
  color: rgba(148, 163, 184, 0.9);
}

.player-card__annotation {
  position: absolute;
  left: 50%;
  top: 100%;
  transform: translate(-50%, -65%);
  text-align: center;
  text-transform: uppercase;
  font-weight: 800;
  pointer-events: none;
  z-index: 2;
}

.player-card__annotation-name {
  display: block;
  font-size: clamp(1.1rem, 3vw, 2.4rem);
  letter-spacing: 0.1em;
  color: #f8fafc;
  text-shadow: 0 0 18px rgba(0, 0, 0, 0.75);
}

.player-card__annotation-number {
  display: block;
  margin-top: 0.15rem;
  font-size: clamp(0.95rem, 2.4vw, 1.6rem);
  letter-spacing: 0.18em;
  color: #38bdf8;
  text-shadow: 0 0 16px rgba(0, 0, 0, 0.65);
}

@keyframes stadium-sweep {
  0% {
    transform: translateX(-120%);
  }
  50% {
    transform: translateX(120%);
  }
  100% {
    transform: translateX(180%);
  }
}

@keyframes card-ignite {
  0% {
    filter: brightness(0.6) saturate(0.8);
    box-shadow: inset 0 0 0 1px rgba(255, 214, 102, 0.35), 0 0 0 rgba(255, 214, 102, 0.55);
    transform: scale(0.98);
  }
  50% {
    filter: brightness(1.08) saturate(1.05);
    box-shadow: inset 0 0 20px rgba(255, 214, 102, 0.5), 0 18px 40px rgba(255, 214, 102, 0.25);
    transform: scale(1.04);
  }
  100% {
    filter: brightness(1);
    box-shadow: inset 0 0 0 1px rgba(255, 214, 102, 0.3), 0 18px 32px rgba(0, 0, 0, 0.35);
    transform: scale(1);
  }
}

@keyframes card-bounce {
  0% {
    transform: scale(1);
  }
  40% {
    transform: scale(0.98);
  }
  100% {
    transform: scale(1.02);
  }
}

@keyframes crown-pulse {
  0% {
    opacity: 0.35;
    transform: scale(0.96);
  }
  50% {
    opacity: 0.9;
    transform: scale(1.04);
  }
  100% {
    opacity: 0.5;
    transform: scale(1);
  }
}

@media (hover: none) {
  .player-card--live:active {
    animation: card-bounce 260ms ease;
  }
}
</style>
