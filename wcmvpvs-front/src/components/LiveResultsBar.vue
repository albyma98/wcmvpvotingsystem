<template>
  <section class="rounded-xl border border-slate-200/25 bg-slate-950/70 px-3 py-3 backdrop-blur-sm">
    <div class="mb-2 flex items-center gap-2 text-white/95">
      <span class="h-px flex-1 bg-gradient-to-r from-white/0 via-white/40 to-white/0" />
      <h2 class="text-sm font-black uppercase tracking-wide">RISULTATI LIVE</h2>
      <span class="h-px flex-1 bg-gradient-to-r from-white/0 via-white/40 to-white/0" />
    </div>

    <div class="overflow-hidden rounded-full border border-white/40 bg-black/25">
      <div class="flex h-8 w-full text-sm font-black uppercase leading-none">
        <div
          v-for="segment in normalizedResults"
          :key="segment.name"
          class="flex items-center justify-center whitespace-nowrap px-2 text-[clamp(0.72rem,2vw,1rem)]"
          :class="segment.className"
          :style="{ width: `${segment.width}%` }"
        >
          {{ segment.name }} {{ segment.value }}%
        </div>
      </div>
    </div>
  </section>
</template>

<script setup>
import { computed } from 'vue';

const props = defineProps({
  results: {
    type: Array,
    default: () => [],
  },
});

const classes = ['bg-gradient-to-r from-red-700 to-red-500 text-white', 'bg-gradient-to-r from-gray-100 to-slate-200 text-slate-900', 'bg-gradient-to-r from-green-700 to-green-500 text-white'];

const normalizedResults = computed(() => {
  const sum = props.results.reduce((acc, item) => acc + (Number(item.value) || 0), 0) || 1;
  return props.results.map((item, index) => {
    const numericValue = Number(item.value) || 0;
    const width = (numericValue / sum) * 100;
    return {
      name: item.name,
      value: numericValue,
      width,
      className: classes[index] ?? 'bg-slate-700 text-white',
    };
  });
});
</script>
