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
          <div class="relative w-full max-w-[88%]" style="aspect-ratio: 1 / 1">
            <div class="absolute inset-0 overflow-hidden rounded-[1.55rem]">
              <img :src="avatarUrl" :alt="player.name" class="h-full w-full object-cover" />
              <div class="absolute inset-0 bg-gradient-to-b from-black/10 via-black/20 to-black/65"></div>
            </div>
            <div
              class="pointer-events-none absolute left-1/2 top-[90%] w-full -translate-x-1/2 -translate-y-full overflow-visible px-3 text-center font-bold uppercase text-white z-10"
            >
              <span class="block text-[clamp(1.5rem,3.6vw,3rem)] leading-none tracking-[0.1em] drop-shadow-[0_0_8px_rgba(0,0,0,0.75)]">
                {{ overlayLastName }}
              </span>
              <span
                v-if="overlayNumber"
                class="mt-1 block text-[clamp(1.1rem,2.8vw,2.25rem)] leading-none tracking-[0.2em] drop-shadow-[0_0_6px_rgba(0,0,0,0.7)]"
              >
                {{ overlayNumber }}
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
