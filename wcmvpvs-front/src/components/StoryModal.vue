<template>
  <div v-if="open && currentStory" class="story-modal" @click="handleSurfaceClick">
    <button class="story-close" type="button" aria-label="Chiudi" @click.stop="$emit('close')">✕</button>

    <header class="story-head" @click.stop>
      <strong>{{ currentStory.player_name }}</strong>
      <small v-if="currentStory.title">{{ currentStory.title }}</small>
    </header>

    <div class="story-player-wrap" @click.stop>
      <video
        ref="videoRef"
        class="story-player"
        :src="currentStory.video_url"
        playsinline
        preload="metadata"
        @ended="emit('next')"
        @click="playVideo"
      />
      <button v-if="showTapHint" class="story-tap-hint" type="button" @click="playVideo">
        Tocca per avviare
      </button>
    </div>

    <button
      v-if="showPrev"
      class="story-nav story-nav--prev"
      type="button"
      aria-label="Story precedente"
      @click.stop="$emit('prev')"
    >
      ‹
    </button>
    <button class="story-nav story-nav--next" type="button" aria-label="Story successiva" @click.stop="$emit('next')">
      ›
    </button>
  </div>
</template>

<script setup>
import { computed, nextTick, ref, watch } from 'vue';

const props = defineProps({
  open: Boolean,
  currentStory: {
    type: Object,
    default: null,
  },
  showPrev: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits(['close', 'next', 'prev']);
const videoRef = ref(null);
const showTapHint = ref(false);

async function playVideo() {
  const el = videoRef.value;
  if (!el) {
    return;
  }

  try {
    await el.play();
    showTapHint.value = false;
  } catch (error) {
    showTapHint.value = true;
  }
}

function handleSurfaceClick(event) {
  const x = event?.clientX || 0;
  const width = window?.innerWidth || 1;
  if (x < width * 0.25) {
    emit('prev');
    return;
  }
  emit('next');
}

watch(
  () => [props.open, props.currentStory?.id],
  async ([isOpen]) => {
    if (!isOpen) {
      showTapHint.value = false;
      return;
    }
    await nextTick();
    playVideo();
  },
);

const showPrev = computed(() => props.showPrev);
</script>

<style scoped>
.story-modal {
  position: fixed;
  inset: 0;
  z-index: 60;
  background: rgba(2, 6, 23, 0.95);
  display: flex;
  align-items: center;
  justify-content: center;
}
.story-close {
  position: absolute;
  top: 1rem;
  right: 1rem;
  color: #e2e8f0;
  background: rgba(15, 23, 42, 0.8);
  border: 1px solid rgba(148, 163, 184, 0.4);
  border-radius: 999px;
  width: 2.2rem;
  height: 2.2rem;
}
.story-head {
  position: absolute;
  top: 1rem;
  left: 1rem;
  display: flex;
  flex-direction: column;
  color: #f8fafc;
}
.story-head small {
  color: #cbd5e1;
}
.story-player-wrap {
  width: min(92vw, 420px);
  aspect-ratio: 9 / 16;
  border-radius: 1rem;
  overflow: hidden;
  background: #000;
  position: relative;
}
.story-player {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.story-tap-hint {
  position: absolute;
  left: 50%;
  bottom: 1rem;
  transform: translateX(-50%);
  border-radius: 999px;
  border: 1px solid rgba(255, 255, 255, 0.3);
  background: rgba(2, 6, 23, 0.75);
  color: #f8fafc;
  padding: 0.45rem 0.85rem;
}
.story-nav {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  color: #fff;
  border: 0;
  width: 2.4rem;
  height: 2.4rem;
  border-radius: 999px;
  background: rgba(15, 23, 42, 0.5);
}
.story-nav--prev {
  left: 0.6rem;
}
.story-nav--next {
  right: 0.6rem;
}
</style>
