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
  isPreMatch: {
    type: Boolean,
    default: false,
  },
  votingOpen: {
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

const interactionClass = computed(() => {
  if (props.isPreMatch && !props.votingOpen) {
    return 'pointer-events-none prematch-breathing';
  }
  if (props.disabled && !props.isSelected) {
    return 'cursor-not-allowed opacity-60';
  }
  return 'cursor-pointer hover:scale-[1.03]';
});

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
      class="relative rounded-[1.75rem] border border-white/10 bg-slate-950/60 transition-transform duration-300 ease-out"
      :class="[
        tierRingClass,
        isSelected ? 'scale-[1.05]' : '',
        interactionClass,
        isSelected ? 'ring-4' : 'ring-2',
      ]"
      @click="handleSelect"
    >
      <div v-if="isSelected" class="selection-badge" aria-label="Il tuo MVP">
        <span class="selection-badge__dot" aria-hidden="true"></span>
        <span class="selection-badge__label">Il tuo MVP</span>
      </div>
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
.prematch-breathing {
  animation: prematch-breath 5.2s ease-in-out infinite;
  will-change: transform;
}

.selection-badge {
  position: absolute;
  top: 0.65rem;
  right: 0.65rem;
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.4rem 0.7rem;
  border-radius: 9999px;
  background: rgba(250, 204, 21, 0.14);
  color: #facc15;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.18em;
  font-size: 0.55rem;
  box-shadow: 0 10px 24px rgba(0, 0, 0, 0.25), 0 0 0 1px rgba(250, 204, 21, 0.35);
}

.selection-badge__dot {
  width: 0.65rem;
  height: 0.65rem;
  border-radius: 9999px;
  background: linear-gradient(135deg, #fde68a, #facc15);
  box-shadow: 0 0 0 4px rgba(250, 204, 21, 0.2);
}

.selection-badge__label {
  white-space: nowrap;
}

@keyframes prematch-breath {
  0% {
    transform: scale(0.95);
  }
  50% {
    transform: scale(1.04);
  }
  100% {
    transform: scale(0.95);
  }
}
</style>
