<template>
  <article
    class="group relative flex min-h-[220px] w-full cursor-pointer flex-col overflow-hidden rounded-xl border border-white/35 p-2.5 text-white shadow-lg transition duration-150 active:scale-[0.99]"
    :class="[cardBackgroundClass, cardGlowClass]"
    role="button"
    tabindex="0"
    :aria-label="`Apri ${title}`"
    @click="emitSelect"
    @keydown.enter.prevent="emitSelect"
    @keydown.space.prevent="emitSelect"
  >
    <div class="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_75%_15%,rgba(255,255,255,0.22),transparent_38%)]" />
    <div class="pointer-events-none absolute inset-0 bg-gradient-to-b from-black/20 via-black/10 to-black/45" />

    <div class="relative z-10 text-center">
      <h3 class="text-[clamp(1rem,2.8vw,1.9rem)] font-black uppercase leading-none tracking-tight drop-shadow-md">
        {{ title }}
      </h3>
      <p class="mt-0.5 text-[clamp(0.7rem,2vw,1rem)] font-bold leading-tight text-white/95">
        {{ subtitle }}
      </p>
    </div>

    <div
      v-if="previewImageUrl"
      class="relative z-10 mt-2 flex-1 overflow-hidden rounded-lg border border-white/30 bg-black/30"
    >
      <img
        :src="previewImageUrl"
        :alt="previewAlt || `MVP selezionato per ${title}`"
        class="h-full min-h-[108px] w-full"
        :class="previewImageFitClass"
      >
    </div>

    <div class="relative z-10 mt-auto rounded-md bg-black/35 px-2 py-1.5 text-[clamp(0.65rem,1.9vw,0.88rem)] font-semibold leading-tight text-white/95">
      <div class="flex items-center justify-center gap-1.5">
        <span aria-hidden="true" class="inline-flex h-4 w-4 items-center justify-center">{{ icon }}</span>
        <span class="text-center">{{ description }}</span>
      </div>
    </div>

    <button
      type="button"
      class="relative z-10 mt-2 w-full rounded-md border border-white/25 px-2 py-2 text-[clamp(0.8rem,2.2vw,1rem)] font-black uppercase tracking-wide text-white shadow-[inset_0_1px_0_rgba(255,255,255,0.4)] transition duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white/70 focus-visible:ring-offset-1 focus-visible:ring-offset-black/60 active:translate-y-[1px]"
      :class="buttonClass"
      :aria-label="buttonAriaLabel"
      @click.stop="emitSelect"
    >
      {{ actionLabel }}
    </button>
  </article>
</template>

<script setup>
import { computed } from 'vue';

const props = defineProps({
  id: {
    type: [String, Number],
    required: true,
  },
  title: {
    type: String,
    required: true,
  },
  subtitle: {
    type: String,
    required: true,
  },
  description: {
    type: String,
    required: true,
  },
  actionLabel: {
    type: String,
    required: true,
  },
  icon: {
    type: String,
    default: '•',
  },
  theme: {
    type: String,
    default: 'orange',
  },
  previewImageUrl: {
    type: String,
    default: '',
  },
  previewAlt: {
    type: String,
    default: '',
  },
  previewImageFit: {
    type: String,
    default: 'cover',
  },
});

const emit = defineEmits(['select']);

const themeMap = {
  orange: {
    cardBg: 'bg-gradient-to-b from-orange-300/25 via-orange-700/30 to-orange-950/85',
    cardGlow: 'shadow-[0_0_22px_rgba(249,115,22,0.44)]',
    button: 'bg-gradient-to-b from-orange-400 to-orange-700 hover:brightness-110',
  },
  blue: {
    cardBg: 'bg-gradient-to-b from-blue-300/25 via-blue-700/30 to-blue-950/85',
    cardGlow: 'shadow-[0_0_22px_rgba(59,130,246,0.44)]',
    button: 'bg-gradient-to-b from-blue-400 to-blue-700 hover:brightness-110',
  },
  green: {
    cardBg: 'bg-gradient-to-b from-emerald-300/25 via-emerald-700/30 to-emerald-950/85',
    cardGlow: 'shadow-[0_0_22px_rgba(34,197,94,0.44)]',
    button: 'bg-gradient-to-b from-lime-400 to-green-700 hover:brightness-110',
  },
};

const currentTheme = computed(() => themeMap[props.theme] ?? themeMap.orange);
const cardBackgroundClass = computed(() => currentTheme.value.cardBg);
const cardGlowClass = computed(() => currentTheme.value.cardGlow);
const buttonClass = computed(() => currentTheme.value.button);
const buttonAriaLabel = computed(() => `${props.actionLabel} - ${props.title}`);
const previewImageFitClass = computed(() => (props.previewImageFit === 'contain' ? 'object-contain' : 'object-cover'));

function emitSelect() {
  emit('select', props.id);
}
</script>
