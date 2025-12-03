<script setup>
import { computed, onMounted, ref, watch } from 'vue';
import { Motion } from '@vueuse/motion';
import { gsap } from 'gsap';

const props = defineProps({
  show: {
    type: Boolean,
    default: false,
  },
});

const emits = defineEmits(['close']);

const flashBackdrop = ref(null);
const streaksRef = ref(null);

const isVisible = computed(() => props.show);

const animateBackdrop = () => {
  if (!flashBackdrop.value) {
    return;
  }
  gsap.fromTo(
    flashBackdrop.value,
    { opacity: 0, scale: 0.98 },
    { opacity: 0.7, scale: 1, duration: 0.8, ease: 'power2.out' },
  );
};

const animateStreaks = () => {
  if (!streaksRef.value) {
    return;
  }
  gsap.fromTo(
    streaksRef.value.children,
    { opacity: 0, yPercent: 12 },
    {
      opacity: 0.4,
      yPercent: 0,
      duration: 0.9,
      ease: 'power3.out',
      stagger: 0.06,
    },
  );
};

const runEntrance = () => {
  animateBackdrop();
  animateStreaks();
};

onMounted(() => {
  if (isVisible.value) {
    runEntrance();
  }
});

watch(isVisible, (visible) => {
  if (visible) {
    runEntrance();
  }
});
</script>

<template>
  <transition name="fade">
    <div
      v-if="isVisible"
      class="fixed inset-0 z-[70] flex items-center justify-center bg-slate-950/85 px-6 py-10"
      role="dialog"
      aria-modal="true"
      aria-labelledby="opening-title"
    >
      <span ref="flashBackdrop" class="opening-flash" aria-hidden="true"></span>
      <div ref="streaksRef" class="opening-streaks" aria-hidden="true">
        <span class="opening-streak opening-streak--one"></span>
        <span class="opening-streak opening-streak--two"></span>
        <span class="opening-streak opening-streak--three"></span>
      </div>
      <Motion
        class="relative z-10 w-full max-w-2xl text-center"
        :initial="{ opacity: 0, scale: 0.92, y: 24 }"
        :enter="{ opacity: 1, scale: 1, y: 0, transition: { duration: 0.4, ease: 'easeOut' } }"
        :hover="{ scale: 1.01 }"
      >
        <div class="opening-panel">
          <p class="opening-panel__eyebrow">Live broadcast mode</p>
          <h3 id="opening-title" class="opening-panel__title">LE VOTAZIONI SONO APERTE</h3>
          <p class="opening-panel__subtitle">Scegli il campione della partita 🏆</p>
          <button type="button" class="opening-panel__cta" @click="emits('close')">
            VOTA ORA
          </button>
        </div>
      </Motion>
    </div>
  </transition>
</template>

<style scoped>
.opening-flash {
  position: absolute;
  inset: 0;
  background: radial-gradient(circle at 50% 40%, rgba(255, 255, 255, 0.32), rgba(56, 189, 248, 0.12), transparent 70%);
  filter: blur(18px);
  mix-blend-mode: screen;
  pointer-events: none;
}

.opening-streaks {
  position: absolute;
  inset: 0;
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 14%;
  opacity: 0.35;
  pointer-events: none;
}

.opening-streak {
  display: block;
  width: 100%;
  height: 100%;
  background: linear-gradient(180deg, rgba(59, 130, 246, 0), rgba(255, 255, 255, 0.6), rgba(14, 165, 233, 0));
  filter: blur(18px);
  transform: skewX(-12deg);
  mix-blend-mode: screen;
}

.opening-streak--one {
  opacity: 0.5;
}

.opening-streak--two {
  opacity: 0.8;
}

.opening-streak--three {
  opacity: 0.4;
}

.opening-panel {
  position: relative;
  overflow: hidden;
  border-radius: 2.5rem;
  padding: 2.5rem 1.75rem;
  background: linear-gradient(145deg, rgba(15, 23, 42, 0.9), rgba(30, 41, 59, 0.82));
  box-shadow: 0 28px 64px rgba(0, 0, 0, 0.55), inset 0 1px 0 rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(148, 163, 184, 0.35);
}

.opening-panel::before {
  content: '';
  position: absolute;
  inset: 0;
  background: radial-gradient(circle at 50% 0%, rgba(56, 189, 248, 0.18), transparent 60%);
  mix-blend-mode: screen;
  opacity: 0.8;
}

.opening-panel::after {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(120deg, rgba(255, 255, 255, 0.08), rgba(255, 255, 255, 0));
  opacity: 0.6;
  mix-blend-mode: screen;
}

.opening-panel__eyebrow {
  position: relative;
  z-index: 1;
  margin: 0;
  text-transform: uppercase;
  letter-spacing: 0.35em;
  color: #38bdf8;
  font-weight: 700;
  font-size: 0.8rem;
}

.opening-panel__title {
  position: relative;
  z-index: 1;
  margin: 1rem 0 0.35rem;
  font-size: clamp(1.75rem, 3vw, 2.6rem);
  letter-spacing: 0.12em;
  color: #f8fafc;
  text-transform: uppercase;
}

.opening-panel__subtitle {
  position: relative;
  z-index: 1;
  margin: 0;
  color: rgba(226, 232, 240, 0.9);
  font-size: 1.05rem;
}

.opening-panel__cta {
  position: relative;
  z-index: 1;
  margin-top: 1.75rem;
  padding: 0.95rem 2.75rem;
  border-radius: 999px;
  border: 1px solid rgba(255, 255, 255, 0.2);
  background: linear-gradient(90deg, rgba(56, 189, 248, 0.9), rgba(59, 130, 246, 0.85));
  color: #0b152d;
  font-weight: 800;
  letter-spacing: 0.3em;
  text-transform: uppercase;
  cursor: pointer;
  box-shadow: 0 0 24px rgba(56, 189, 248, 0.45), 0 20px 48px rgba(15, 23, 42, 0.6);
  transition: transform 180ms ease, box-shadow 220ms ease;
}

.opening-panel__cta:hover {
  transform: translateY(-1px) scale(1.01);
  box-shadow: 0 0 32px rgba(56, 189, 248, 0.6), 0 24px 52px rgba(15, 23, 42, 0.75);
}

.opening-panel__cta::after {
  content: '';
  position: absolute;
  inset: -6px;
  border-radius: 999px;
  background: radial-gradient(circle at 50% 50%, rgba(59, 130, 246, 0.35), transparent 55%);
  filter: blur(14px);
  opacity: 0.75;
  animation: pulse-glow 2.4s ease-in-out infinite;
  z-index: -1;
}

@keyframes pulse-glow {
  0% {
    opacity: 0.4;
    transform: scale(0.98);
  }
  50% {
    opacity: 0.9;
    transform: scale(1.03);
  }
  100% {
    opacity: 0.4;
    transform: scale(0.98);
  }
}
</style>
