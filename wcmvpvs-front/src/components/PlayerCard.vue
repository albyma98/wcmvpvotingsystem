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
  height: `${props.cardSize * 1.6}px`,
}));

const wrapperStyle = computed(() => ({
  width: `${props.cardSize}px`,
  minHeight: `${props.cardSize * 1.8}px`,
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
  <div class="player-card" :style="wrapperStyle">
    <div
      :style="cardStyle"
      class="player-card__body"
      :class="[
        tierRingClass,
        isSelected ? 'scale-[1.05]' : 'hover:scale-[1.03]',
        disabled && !isSelected ? 'cursor-not-allowed opacity-60' : 'cursor-pointer',
        isSelected ? 'ring-4' : 'ring-2',
      ]"
      @click="handleSelect"
    >
      <div class="player-card__image">
        <img :src="avatarUrl" :alt="player.name" class="player-card__photo" />
        <div class="player-card__image-mask"></div>
      </div>
      <div class="player-card__label" aria-hidden="true">
        <span class="player-card__name">{{ overlayLastName }}</span>
        <span v-if="overlayNumber" class="player-card__number">{{ overlayNumber }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.player-card {
  display: flex;
  justify-content: flex-start;
}

.player-card__body {
  position: relative;
  border-radius: 1.75rem;
  border: 1px solid rgb(255 255 255 / 0.1);
  background-color: rgb(15 23 42 / 0.6);
  transition: transform 200ms ease-out;
  overflow: hidden;
  box-shadow: 0 20px 45px rgba(8, 15, 28, 0.45);
  display: grid;
  place-items: center;
}

.player-card__image {
  position: relative;
  width: 100%;
  height: 100%;
  overflow: hidden;
  border-radius: 1.55rem;
}

.player-card__photo {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.player-card__image-mask {
  position: absolute;
  inset: 0;
  background: linear-gradient(
    180deg,
    rgba(0, 0, 0, 0.05) 0%,
    rgba(0, 0, 0, 0.15) 35%,
    rgba(0, 0, 0, 0.38) 100%
  );
}

.player-card__label {
  position: absolute;
  inset-inline: 0;
  bottom: 0.75rem;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.35rem;
  padding-inline: 1.25rem;
  pointer-events: none;
  text-transform: uppercase;
  color: white;
  text-shadow: 0 0 12px rgba(0, 0, 0, 0.85);
}

.player-card__name {
  font-size: clamp(1rem, 3vw, 1.75rem);
  letter-spacing: 0.12em;
  line-height: 1.05;
  font-weight: 700;
}

.player-card__number {
  font-size: clamp(1.1rem, 3vw, 1.5rem);
  letter-spacing: 0.2em;
  line-height: 1.05;
  font-weight: 700;
}
</style>
