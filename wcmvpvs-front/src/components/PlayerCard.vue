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

const tierRingClass = computed(() => {
  switch (props.player.tier) {
    case 'silver':
      return 'ring-slate-200/70 shadow-[0_0_28px_rgba(148,163,184,0.35)]';
    case 'bronze':
      return 'ring-amber-300/70 shadow-[0_0_28px_rgba(234,179,8,0.35)]';
    default:
      return 'ring-yellow-300/80 shadow-[0_0_32px_rgba(250,204,21,0.4)]';
  }
});

const fallbackAvatar = computed(
  () => `https://api.dicebear.com/7.x/adventurer/svg?seed=${encodeURIComponent(props.player.name ?? props.player.number)}`,
);

const avatarUrl = computed(() => props.player.avatar || fallbackAvatar.value);

const cardStyle = computed(() => ({
  width: `${props.cardSize}px`,
  height: `${props.cardSize * 1.4}px`,
}));

const numberLabel = computed(() => `#${String(props.player.number).padStart(2, '0')}`);

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

const handleSelect = () => {
  if ((props.disabled && !props.isSelected) || props.isVoting) {
    return;
  }
  emits('select');
};
</script>

<template>
  <div
    :style="cardStyle"
    class="relative overflow-hidden rounded-[1.75rem] border border-white/10 bg-slate-950/60 transition-transform duration-200 ease-out"
    :class="[
      tierRingClass,
      isSelected ? 'scale-[1.05]' : 'hover:scale-[1.03]',
      disabled && !isSelected ? 'cursor-not-allowed opacity-60' : 'cursor-pointer',
      isSelected ? 'ring-4' : 'ring-2',
    ]"
    @click="handleSelect"
  >
    <div class="flex h-full w-full flex-col items-center gap-3 px-4 py-5">
      <span
        class="rounded-full bg-black/70 px-4 py-1 text-lg font-semibold tracking-[0.3em] text-white backdrop-blur-sm"
        :class="isSelected ? 'shadow-[0_0_20px_rgba(250,204,21,0.45)]' : ''"
      >
        {{ numberLabel }}
      </span>

      <div class="relative w-full overflow-hidden rounded-[1.5rem]" style="aspect-ratio: 1 / 1">
        <img :src="avatarUrl" :alt="player.name" class="h-full w-full object-cover" />
        <div class="absolute inset-0 bg-gradient-to-b from-black/10 via-black/20 to-black/50"></div>
      </div>

      <div class="mt-auto w-full text-center leading-tight">
        <p class="text-base font-semibold uppercase tracking-[0.2em] text-white">{{ playerNameParts.firstName }}</p>
        <p
          v-if="playerNameParts.lastName"
          class="mt-1 text-sm font-semibold uppercase tracking-[0.25em] text-white/80"
        >
          {{ playerNameParts.lastName }}
        </p>
      </div>
    </div>
  </div>
</template>
