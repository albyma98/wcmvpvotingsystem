<template>
  <section
    v-if="visibleSponsors.length"
    class="sponsor-box rounded-2xl border border-white/12 bg-slate-950/70 px-2 py-1.5"
    :style="containerStyle"
    aria-label="Sponsor"
  >
    <p class="mb-1 pl-1 text-[0.62rem] font-black uppercase tracking-[0.22em] text-slate-300/85">
      Sponsor
    </p>

    <div ref="viewportRef" class="sponsor-viewport">
      <div
        class="sponsor-track"
        :class="{ 'is-animating': shouldAnimate }"
        :style="trackStyle"
      >
        <component
          :is="item.linkUrl ? 'a' : 'div'"
          v-for="item in renderedSponsors"
          :key="item.trackKey"
          class="sponsor-item"
          :href="item.linkUrl || undefined"
          :target="item.linkUrl ? '_blank' : undefined"
          :rel="item.linkUrl ? 'noopener noreferrer' : undefined"
          @click="onSponsorClick(item)"
        >
          <img
            class="sponsor-image"
            :src="item.imageUrl"
            :alt="item.name || 'Sponsor'"
            loading="lazy"
            draggable="false"
            @load="emit('image-loaded')"
            @error="emit('image-loaded')"
          />
        </component>
      </div>
    </div>
  </section>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';

const props = defineProps({
  sponsors: {
    type: Array,
    default: () => [],
  },
  heightPx: {
    type: Number,
    default: 0,
  },
  animationDelayMs: {
    type: Number,
    default: 450,
  },
  speedPxPerSecond: {
    type: Number,
    default: 62,
  },
  eventId: {
    type: Number,
    default: 0,
  },
});

const emit = defineEmits(['image-loaded', 'sponsor-click']);

const prefersReducedMotion = ref(false);
const shouldAnimate = ref(false);
const cycleSeconds = ref(16);
let animationTimer = 0;

const visibleSponsors = computed(() =>
  (Array.isArray(props.sponsors) ? props.sponsors : [])
    .filter((item) => item && item.imageUrl)
    .slice(0, 40),
);

const canLoopSeamlessly = computed(() => visibleSponsors.value.length > 1);

const renderedSponsors = computed(() => {
  if (!visibleSponsors.value.length) {
    return [];
  }

  if (!canLoopSeamlessly.value) {
    return visibleSponsors.value.map((item) => ({
      ...item,
      trackKey: `single-${item.id}`,
    }));
  }

  return [...visibleSponsors.value, ...visibleSponsors.value].map((item, index) => ({
    ...item,
    trackKey: `${item.id}-${index}`,
  }));
});

const viewportRef = ref(null);

const containerStyle = computed(() => {
  const resolved = Math.max(0, Math.floor(props.heightPx || 0));
  return {
    height: `${resolved}px`,
    maxHeight: `${resolved}px`,
  };
});

const trackStyle = computed(() => ({
  '--sponsor-cycle-duration': `${cycleSeconds.value}s`,
}));

function computeAnimationDuration() {
  if (!viewportRef.value || !canLoopSeamlessly.value) {
    cycleSeconds.value = 16;
    return;
  }

  const totalWidth = Math.max(1, viewportRef.value.scrollWidth / 2);
  const speed = Math.max(35, Number(props.speedPxPerSecond) || 62);
  cycleSeconds.value = Math.max(8, totalWidth / speed);
}

function scheduleAnimation() {
  if (animationTimer) {
    window.clearTimeout(animationTimer);
    animationTimer = 0;
  }

  shouldAnimate.value = false;
  if (prefersReducedMotion.value || !canLoopSeamlessly.value) {
    return;
  }

  animationTimer = window.setTimeout(() => {
    shouldAnimate.value = true;
  }, Math.max(0, Number(props.animationDelayMs) || 0));
}

function onSponsorClick(item) {
  if (!item?.linkUrl) {
    return;
  }
  emit('sponsor-click', item);
}

function setupReducedMotionListener() {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
    return;
  }

  const media = window.matchMedia('(prefers-reduced-motion: reduce)');
  const sync = () => {
    prefersReducedMotion.value = media.matches;
    scheduleAnimation();
  };

  sync();

  if (typeof media.addEventListener === 'function') {
    media.addEventListener('change', sync);
    onBeforeUnmount(() => media.removeEventListener('change', sync));
  } else if (typeof media.addListener === 'function') {
    media.addListener(sync);
    onBeforeUnmount(() => media.removeListener(sync));
  }
}

watch(
  () => [visibleSponsors.value.length, props.heightPx],
  () => {
    computeAnimationDuration();
    scheduleAnimation();
  },
  { immediate: true },
);

onMounted(() => {
  setupReducedMotionListener();
  computeAnimationDuration();
  scheduleAnimation();
});

onBeforeUnmount(() => {
  if (animationTimer) {
    window.clearTimeout(animationTimer);
  }
});
</script>

<style scoped>
.sponsor-box {
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.03);
}

.sponsor-viewport {
  height: calc(100% - 1rem);
  overflow: hidden;
  display: flex;
  align-items: center;
}

.sponsor-track {
  display: flex;
  align-items: center;
  width: max-content;
  gap: 0.9rem;
  padding-inline: 0.2rem;
  will-change: transform;
}

.sponsor-track.is-animating {
  animation: sponsor-marquee var(--sponsor-cycle-duration, 16s) linear infinite;
}

.sponsor-item {
  flex: 0 0 auto;
  height: 100%;
  min-width: 78px;
  max-width: 220px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0.2rem 0.45rem;
  border-radius: 0.55rem;
  background: rgba(15, 23, 42, 0.38);
}

.sponsor-image {
  height: 78%;
  width: auto;
  max-width: min(28vw, 164px);
  object-fit: contain;
}

@keyframes sponsor-marquee {
  from {
    transform: translateX(0);
  }
  to {
    transform: translateX(-50%);
  }
}

@media (prefers-reduced-motion: reduce) {
  .sponsor-track.is-animating {
    animation: none;
  }
}
</style>
