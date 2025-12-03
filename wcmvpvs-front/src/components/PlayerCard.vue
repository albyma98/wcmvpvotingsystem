<script setup>
import { computed } from 'vue';

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
      <div
        class="pointer-events-none absolute left-1/2 top-[100%] z-20  -translate-x-1/2 -translate-y-full px-6 text-center font-bold uppercase text-white"
      >
        <span class="block text-[clamp(1rem,3vw,2.5rem)] leading-none tracking-[0.1em] drop-shadow-[0_0_12px_rgba(0,0,0,0.85)]">
          {{ overlayLastName }}
        </span>
        <span
          v-if="overlayNumber"
          class="mt-1 block text-[clamp(1.1rem,2.8vw,2.25rem)] leading-none tracking-[0.2em] drop-shadow-[0_0_8px_rgba(0,0,0,0.8)]"
        >
          {{ overlayNumber }}
        </span>
      </div>
      <div class="flex h-full w-full flex-col items-center">
        <div class="flex w-full items-center justify-center">
          <div class="relative w-full max-w-100%]" style="aspect-ratio: 1 / 1">
            <div class="absolute overflow-hidden rounded-[1.55rem]">
              <img :src="avatarUrl" :alt="player.name" class="h-full w-full object-fill" />
              <div
                class="absolute inset-0 bg-gradient-to-b from-[rgba(0,0,0,0.05)] via-[rgba(0,0,0,0.1)] to-[rgba(0,0,0,0.325)]"
              ></div>
            </div>
          </div>
        </div>
      </div>
    </div>
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
  filter: brightness(0.5) saturate(0.8);
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

@media (hover: none) {
  .player-card--live:active {
    animation: card-bounce 260ms ease;
  }
}
</style>
