<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { gsap } from 'gsap';

const props = defineProps({
  players: {
    type: Array,
    default: () => [],
  },
  active: {
    type: Boolean,
    default: false,
  },
  calm: {
    type: Boolean,
    default: false,
  },
  prematch: {
    type: Boolean,
    default: false,
  },
  spotlightedPlayerId: {
    type: Number,
    default: null,
  },
});

const beamLeft = ref(null);
const beamRight = ref(null);
const washRef = ref(null);
const spotlightRef = ref(null);

let beamsTimeline = null;
let sweepTimer = null;

const availableTargets = computed(() => {
  if (!Array.isArray(props.players) || !props.players.length) {
    return [
      { position: { x: 28, y: 36 } },
      { position: { x: 52, y: 52 } },
      { position: { x: 72, y: 64 } },
    ];
  }
  return props.players;
});

const isSpotlightFrozen = computed(() =>
  Number.isFinite(props.spotlightedPlayerId) && props.spotlightedPlayerId > 0,
);

const resolveTargetPosition = (player) => {
  if (!player?.position) {
    return { left: '50%', top: '50%' };
  }
  const x = typeof player.position.x === 'number' ? player.position.x : 50;
  const y = typeof player.position.y === 'number' ? player.position.y : 50;
  return {
    left: `${x}%`,
    top: `${y}%`,
  };
};

const stopBeams = () => {
  if (beamsTimeline) {
    beamsTimeline.kill();
    beamsTimeline = null;
  }
};

const animateBeams = () => {
  stopBeams();
  beamsTimeline = gsap.timeline({ repeat: -1, defaults: { ease: 'sine.inOut' } });
  if (beamLeft.value && beamRight.value) {
    beamsTimeline
      .to(beamLeft.value, { rotation: -9, duration: 6, transformOrigin: '50% 0%' }, 0)
      .to(beamLeft.value, { rotation: -2, duration: 5 }, 6)
      .to(beamLeft.value, { rotation: -12, duration: 4 }, 11)
      .to(beamRight.value, { rotation: 6, duration: 6, transformOrigin: '50% 0%' }, 0)
      .to(beamRight.value, { rotation: 14, duration: 5 }, 6)
      .to(beamRight.value, { rotation: 4, duration: 4 }, 11);
  }
  if (washRef.value) {
    beamsTimeline.to(washRef.value, { opacity: 0.45, duration: 5 }, 0);
    beamsTimeline.to(washRef.value, { opacity: 0.7, duration: 6 }, 5);
  }
};

const stopSpotlightSweep = () => {
  if (sweepTimer) {
    window.clearTimeout(sweepTimer);
    sweepTimer = null;
  }
};

const moveSpotlightTo = (target, options = {}) => {
  const element = spotlightRef.value;
  if (!element) {
    return;
  }
  const { left, top } = resolveTargetPosition(target);
  gsap.to(element, {
    left,
    top,
    duration: options.duration ?? 2.2,
    ease: options.ease ?? 'power2.inOut',
    scale: options.scale ?? (props.prematch ? 1.2 : 1),
    opacity: options.opacity ?? 0.92,
  });
};

const scheduleSweep = () => {
  stopSpotlightSweep();
  if (!props.active || isSpotlightFrozen.value) {
    return;
  }
  const targets = availableTargets.value;
  if (!targets.length) {
    return;
  }
  const next = targets[Math.floor(Math.random() * targets.length)];
  moveSpotlightTo(next);
  sweepTimer = window.setTimeout(scheduleSweep, props.prematch ? 2600 : 3200);
};

const applySpotlightLock = () => {
  if (!isSpotlightFrozen.value) {
    return;
  }
  const target = availableTargets.value.find(
    (player) => player.id === props.spotlightedPlayerId,
  );
  if (target) {
    moveSpotlightTo(target, { duration: 1.2, ease: 'power3.out', scale: 1.1 });
  }
  stopSpotlightSweep();
};

watch(
  () => props.active,
  (active) => {
    if (active) {
      animateBeams();
      scheduleSweep();
    } else {
      stopBeams();
      stopSpotlightSweep();
    }
  },
  { immediate: true },
);

watch(
  () => props.spotlightedPlayerId,
  () => {
    if (props.active) {
      applySpotlightLock();
    }
  },
);

watch(
  () => props.prematch,
  (prematch) => {
    if (!props.active) {
      return;
    }
    if (prematch) {
      scheduleSweep();
    } else if (!isSpotlightFrozen.value) {
      moveSpotlightTo({ position: { x: 52, y: 50 } }, { duration: 1.8 });
    }
  },
);

onMounted(() => {
  if (props.active) {
    animateBeams();
    scheduleSweep();
  }
});

onBeforeUnmount(() => {
  stopBeams();
  stopSpotlightSweep();
});
</script>

<template>
  <div
    class="arena-lights"
    :class="{ 'arena-lights--calm': calm, 'arena-lights--prematch': prematch }"
  >
    <div ref="beamLeft" class="arena-lights__beam arena-lights__beam--left" aria-hidden="true"></div>
    <div ref="beamRight" class="arena-lights__beam arena-lights__beam--right" aria-hidden="true"></div>
    <div ref="washRef" class="arena-lights__wash" aria-hidden="true"></div>
    <div ref="spotlightRef" class="arena-lights__spot" aria-hidden="true"></div>
  </div>
</template>

<style scoped>
.arena-lights {
  position: absolute;
  inset: 0;
  pointer-events: none;
  overflow: hidden;
  mix-blend-mode: screen;
  opacity: 0.9;
  transition: opacity 0.45s ease, filter 0.45s ease;
  z-index: 1;
}

.arena-lights--calm {
  opacity: 0.55;
  filter: saturate(0.9);
}

.arena-lights--prematch {
  filter: saturate(1.1) brightness(0.95);
}

.arena-lights__beam {
  position: absolute;
  width: 140%;
  height: 90%;
  top: -12%;
  background: linear-gradient(120deg, rgba(255, 220, 160, 0.45), rgba(255, 255, 255, 0));
  filter: blur(22px);
  opacity: 0.5;
  transform-origin: 50% 0%;
}

.arena-lights__beam--left {
  left: -38%;
}

.arena-lights__beam--right {
  right: -38%;
}

.arena-lights__wash {
  position: absolute;
  inset: -6% -8% auto;
  height: 72%;
  background: radial-gradient(circle at 50% 0%, rgba(255, 237, 213, 0.32), transparent 65%);
  filter: blur(38px);
  opacity: 0.55;
}

.arena-lights__spot {
  position: absolute;
  width: 38%;
  height: 38%;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  background: radial-gradient(circle at 50% 40%, rgba(255, 223, 155, 0.7), rgba(255, 223, 155, 0.25), transparent 70%);
  filter: blur(14px) drop-shadow(0 0 20px rgba(255, 214, 102, 0.25));
  opacity: 0.85;
  border-radius: 999px;
  box-shadow: 0 0 120px rgba(255, 214, 102, 0.28);
}
</style>
