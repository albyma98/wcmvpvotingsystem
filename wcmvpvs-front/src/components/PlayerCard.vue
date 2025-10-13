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
  compact: {
    type: Boolean,
    default: false,
  },
});

const emits = defineEmits(['vote']);

const tierClass = computed(() => {
  switch (props.player.tier) {
    case 'silver':
      return 'card-tier-silver';
    case 'bronze':
      return 'card-tier-bronze';
    default:
      return 'card-tier-gold';
  }
});

const initials = computed(() => {
  const tokens = props.player.name.split(' ');
  return tokens
    .map((token) => token.charAt(0))
    .join('')
    .slice(0, 2)
    .toUpperCase();
});

const actionLabel = computed(() => {
  if (props.isSelected) {
    return 'Votato';
  }
  if (props.isVoting && !props.disabled) {
    return 'Invio...';
  }
  return 'Vota';
});

const cardStyle = computed(() => ({
  width: `${props.cardSize}px`,
  minHeight: `${props.cardSize * 1.45}px`,
}));

const badgeClass = computed(() =>
  props.compact
    ? 'w-12 h-12 text-base'
    : 'w-16 h-16 text-lg'
);

const roleClass = computed(() =>
  props.compact ? 'text-[0.6rem]' : 'text-[0.7rem]'
);

const nameClass = computed(() =>
  props.compact ? 'text-sm leading-tight' : 'text-base leading-tight'
);

const handleVote = () => {
  if ((props.disabled && !props.isSelected) || props.isVoting) {
    return;
  }
  emits('vote');
};
</script>

<template>
  <div
    :style="cardStyle"
    class="relative flex flex-col items-center rounded-[1.75rem] p-3 shadow-lg shadow-slate-900/45 transition-all duration-200 ease-out"
    :class="[
      tierClass,
      isSelected ? 'ring-4 ring-yellow-300 shadow-glow scale-[1.02]' : 'ring-1 ring-slate-900/10',
      disabled && !isSelected ? 'cursor-not-allowed opacity-70' : 'cursor-pointer hover:-translate-y-1',
    ]"
    @click="handleVote"
  >
    <div class="flex w-full items-center justify-between text-[0.65rem] font-semibold uppercase tracking-[0.18em] text-slate-700">
      <span>#{{ String(player.number).padStart(2, '0') }}</span>
      <span :class="roleClass">{{ player.role }}</span>
    </div>

    <div class="mt-3 flex w-full flex-col items-center gap-3">
      <div
        class="flex items-center justify-center rounded-full border-2 border-white/70 bg-white/80 text-slate-800 font-semibold shadow-inner shadow-slate-500/30"
        :class="badgeClass"
      >
        <span v-if="player.avatar" class="overflow-hidden rounded-full">
          <img :src="player.avatar" :alt="player.name" class="h-full w-full object-cover" />
        </span>
        <span v-else>{{ initials }}</span>
      </div>
      <p class="text-center font-semibold" :class="nameClass">
        {{ player.name }}
      </p>
      <span
        v-if="player.libero"
        class="rounded-full bg-amber-200/70 px-3 py-0.5 text-[0.6rem] font-semibold uppercase tracking-[0.3em] text-amber-700"
      >
        Libero
      </span>
    </div>

    <button
      class="mt-auto w-full rounded-full border border-slate-900/20 bg-slate-900/90 px-4 py-2 text-[0.65rem] font-semibold uppercase tracking-[0.3em] text-white transition-colors duration-200"
      :class="[
        isSelected ? 'bg-yellow-400 text-slate-900' : '',
        disabled && !isSelected ? 'opacity-60' : 'hover:bg-yellow-300 hover:text-slate-900',
      ]"
      :disabled="(disabled && !isSelected) || isVoting"
      @click.stop="handleVote"
    >
      {{ actionLabel }}
    </button>
  </div>
</template>
