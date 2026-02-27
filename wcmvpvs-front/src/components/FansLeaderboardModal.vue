<template>
  <Teleport to="body">
    <Transition name="leaderboard-fade">
      <div
        v-if="modelValue"
        class="leaderboard-modal fixed inset-0 z-[90] flex items-end bg-slate-950/80 p-2 sm:items-center sm:justify-center"
        role="dialog"
        aria-modal="true"
      >
        <div class="leaderboard-modal__card w-full max-w-md overflow-hidden rounded-2xl border border-white/25 bg-slate-950 text-white shadow-2xl">
          <header class="flex items-center justify-between border-b border-white/15 px-4 py-3">
            <h2 class="text-base font-black uppercase tracking-wide">Classifica tifosi</h2>
            <button
              type="button"
              class="rounded-md border border-white/25 px-2 py-1 text-xs font-bold"
              @click="emit('update:modelValue', false)"
            >
              CHIUDI
            </button>
          </header>

          <div class="p-4">
            <ol class="space-y-2">
              <li
                v-for="(entry, index) in safeTopList"
                :key="`${entry.name}-${index}`"
                class="flex items-center justify-between rounded-lg border border-white/15 bg-white/5 px-3 py-2"
              >
                <p class="font-bold">{{ medals[index] }} {{ entry.name }}</p>
                <p class="font-black">{{ entry.coins }} 🪙</p>
              </li>
            </ol>

            <p v-if="userRank" class="mt-3 rounded-md border border-amber-300/35 bg-amber-400/10 px-3 py-2 text-sm font-bold">
              Tu: #{{ userRank.rank }} • {{ userRank.coins }} 🪙
            </p>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { computed } from 'vue';

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false,
  },
  topList: {
    type: Array,
    default: () => [],
  },
  userRank: {
    type: Object,
    default: null,
  },
});

const emit = defineEmits(['update:modelValue']);

const medals = ['🥇', '🥈', '🥉'];
const safeTopList = computed(() => {
  if (!Array.isArray(props.topList) || !props.topList.length) {
    return [
      { name: 'TIFO1', coins: 0 },
      { name: 'TIFO2', coins: 0 },
      { name: 'TIFO3', coins: 0 },
    ];
  }
  return props.topList.slice(0, 3);
});
</script>

<style scoped>
.leaderboard-fade-enter-active,
.leaderboard-fade-leave-active {
  transition: opacity 0.2s ease;
}

.leaderboard-fade-enter-from,
.leaderboard-fade-leave-to {
  opacity: 0;
}

.leaderboard-modal__card {
  background:
    radial-gradient(circle at 20% 0%, rgba(251, 191, 36, 0.2), transparent 55%),
    radial-gradient(circle at 90% 10%, rgba(59, 130, 246, 0.2), transparent 45%),
    #020617;
}
</style>
