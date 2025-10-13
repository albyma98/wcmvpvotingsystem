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
  height: `${props.cardSize * 1.5}px`,
}));

const wrapperStyle = computed(() => ({
  width: `${props.cardSize}px`,
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
  <div class="flex flex-col items-center" :style="wrapperStyle">
    <span
      class="mb-3 inline-flex min-w-[72px] items-center justify-center rounded-full bg-black/70 px-5 py-1 text-base font-semibold tracking-[0.35em] text-white backdrop-blur-sm"
      :class="isSelected ? 'shadow-[0_0_20px_rgba(250,204,21,0.45)]' : ''"
    >
      {{ numberLabel }}
    </span>

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
      <div class="flex h-full w-full flex-col items-center px-4 py-6">
        <div class="flex w-full flex-1 items-center justify-center">
          <div class="relative w-full max-w-[88%] overflow-hidden rounded-[1.55rem]" style="aspect-ratio: 1 / 1">
            <img :src="avatarUrl" :alt="player.name" class="h-full w-full object-cover" />
            <div class="absolute inset-0 bg-gradient-to-b from-black/10 via-black/20 to-black/45"></div>
          </div>
        </div>
      </div>
    </div>

    <div class="mt-3 w-full text-center leading-tight">
      <p class="text-base font-semibold uppercase tracking-[0.28em] text-white">{{ playerNameParts.firstName }}</p>
      <p v-if="playerNameParts.lastName" class="mt-1 text-sm font-semibold uppercase tracking-[0.3em] text-white/75">
        {{ playerNameParts.lastName }}
      </p>
    </div>
  </div>
</template>
